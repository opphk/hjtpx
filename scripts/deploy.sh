#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

DEPLOY_LOG_DIR="${PROJECT_ROOT}/logs"
DEPLOY_LOG_FILE="${DEPLOY_LOG_DIR}/deploy-$(date +%Y%m%d-%H%M%S).log"
BACKUP_DIR="${PROJECT_ROOT}/backups"
MAX_DEPLOY_RETRIES=3
HEALTH_CHECK_TIMEOUT=60
HEALTH_CHECK_INTERVAL=5
ROLLBACK_ON_FAILURE="${ROLLBACK_ON_FAILURE:-true}"

log() {
    local level="$1"
    local message="$2"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[${timestamp}] [${level}] ${message}" | tee -a "$DEPLOY_LOG_FILE"
}

log_separator() {
    echo "========================================" | tee -a "$DEPLOY_LOG_FILE"
}

init_deploy_env() {
    log_separator
    log "INFO" "===== HJTPX 部署脚本初始化 ====="
    
    mkdir -p "$DEPLOY_LOG_DIR"
    mkdir -p "$BACKUP_DIR"
    
    log "INFO" "部署日志文件: $DEPLOY_LOG_FILE"
}

check_prerequisites() {
    log_separator
    log "INFO" "1. 检查前置条件..."
    
    if ! command -v docker &> /dev/null; then
        log "ERROR" "Docker 未安装"
        return 1
    fi
    log "INFO" "  ✓ Docker 已安装: $(docker --version | cut -d' ' -f3 | tr -d ',')"
    
    local compose_cmd=""
    if command -v docker-compose &> /dev/null; then
        compose_cmd="docker-compose"
    elif docker compose version &> /dev/null; then
        compose_cmd="docker compose"
    else
        log "ERROR" "Docker Compose 未安装"
        return 1
    fi
    export COMPOSE_CMD="$compose_cmd"
    log "INFO" "  ✓ Docker Compose 已安装: $(${compose_cmd} --version | head -n1)"
    
    if [ ! -f ".env" ]; then
        log "WARN" ".env 文件不存在，复制 .env.example 作为模板"
        if [ -f ".env.example" ]; then
            cp .env.example .env
            log "INFO" "请编辑 .env 文件配置您的环境变量"
            log "INFO" "完成后重新运行此脚本"
            return 1
        else
            log "ERROR" ".env.example 也不存在，无法创建环境配置"
            return 1
        fi
    fi
    log "INFO" "  ✓ 环境配置文件已存在"
    
    return 0
}

prepare_directories() {
    log_separator
    log "INFO" "2. 创建必要目录..."
    
    mkdir -p backend/config
    mkdir -p nginx/ssl
    mkdir -p monitoring/prometheus/rules
    mkdir -p monitoring/grafana/provisioning/dashboards
    mkdir -p monitoring/promtail
    mkdir -p monitoring/loki
    
    log "INFO" "  ✓ 目录结构已准备"
}

generate_ssl_cert() {
    log_separator
    log "INFO" "3. 生成SSL证书..."
    
    if [ ! -f "nginx/ssl/cert.pem" ]; then
        openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
            -keyout nginx/ssl/key.pem \
            -out nginx/ssl/cert.pem \
            -subj "/C=CN/ST=Beijing/L=Beijing/O=HJTPX/CN=localhost" 2>/dev/null
        log "INFO" "  ✓ SSL证书已生成"
    else
        log "INFO" "  ✓ SSL证书已存在，跳过生成"
    fi
}

backup_current_version() {
    log_separator
    log "INFO" "4. 备份当前版本..."
    
    local backup_name="backup-$(date +%Y%m%d-%H%M%S)"
    local backup_path="${BACKUP_DIR}/${backup_name}"
    
    mkdir -p "$backup_path"
    
    if [ -f "docker-compose.yml" ]; then
        cp docker-compose.yml "$backup_path/"
    fi
    if [ -d "backend" ]; then
        tar -czf "$backup_path/backend.tar.gz" backend/ 2>/dev/null || true
    fi
    if [ -d "nginx" ]; then
        cp -r nginx/conf.d "$backup_path/" 2>/dev/null || true
    fi
    
    echo "$backup_path" > "${BACKUP_DIR}/latest_backup"
    log "INFO" "  ✓ 备份已保存: ${backup_name}"
    log "INFO" "  备份路径: $backup_path"
}

build_docker_images() {
    log_separator
    log "INFO" "5. 构建Docker镜像..."
    
    $COMPOSE_CMD build --no-cache
    log "INFO" "  ✓ Docker镜像构建完成"
}

pull_latest_images() {
    log_separator
    log "INFO" "6. 拉取最新镜像..."
    
    $COMPOSE_CMD pull
    log "INFO" "  ✓ 镜像拉取完成"
}

stop_services() {
    log_separator
    log "INFO" "7. 停止现有服务..."
    
    $COMPOSE_CMD down --remove-orphans 2>/dev/null || true
    log "INFO" "  ✓ 现有服务已停止"
}

start_services() {
    log_separator
    log "INFO" "8. 启动服务..."
    
    $COMPOSE_CMD up -d
    log "INFO" "  ✓ 服务启动命令已执行"
}

wait_for_services() {
    log_separator
    log "INFO" "9. 等待服务启动..."
    
    local elapsed=0
    while [ $elapsed -lt $HEALTH_CHECK_TIMEOUT ]; do
        if $COMPOSE_CMD ps | grep -q "Up"; then
            log "INFO" "  ✓ Docker容器已启动 (耗时: ${elapsed}s)"
            return 0
        fi
        sleep 2
        elapsed=$((elapsed + 2))
    done
    
    log "WARN" "  ⚠ 等待超时，但继续进行健康检查"
    return 0
}

perform_health_check() {
    log_separator
    log "INFO" "10. 执行健康检查..."
    
    local backend_url="${APP_URL:-http://localhost:8080}"
    local check_curl="${backend_url}/health"
    local elapsed=0
    local health_passed=false
    
    while [ $elapsed -lt $HEALTH_CHECK_TIMEOUT ]; do
        if curl -sf --connect-timeout 5 "$check_curl" > /dev/null 2>&1; then
            log "INFO" "  ✓ 后端服务健康检查通过 (响应时间: ${elapsed}s)"
            health_passed=true
            break
        fi
        sleep $HEALTH_CHECK_INTERVAL
        elapsed=$((elapsed + HEALTH_CHECK_INTERVAL))
        log "INFO" "  等待服务就绪... (${elapsed}/${HEALTH_CHECK_TIMEOUT}s)"
    done
    
    if [ "$health_passed" = false ]; then
        log "ERROR" "  ✗ 后端服务健康检查失败"
        return 1
    fi
    
    return 0
}

rollback_to_backup() {
    log_separator
    log "WARN" "部署失败，开始回滚..."
    
    local latest_backup=$(cat "${BACKUP_DIR}/latest_backup" 2>/dev/null)
    
    if [ -z "$latest_backup" ] || [ ! -d "$latest_backup" ]; then
        log "ERROR" "  ✗ 未找到可用的备份，无法回滚"
        return 1
    fi
    
    log "INFO" "  正在从备份恢复: $(basename "$latest_backup")"
    
    $COMPOSE_CMD down --remove-orphans 2>/dev/null || true
    
    if [ -f "${latest_backup}/docker-compose.yml" ]; then
        cp "${latest_backup}/docker-compose.yml" ./
    fi
    if [ -f "${latest_backup}/backend.tar.gz" ]; then
        tar -xzf "${latest_backup}/backend.tar.gz" -C . 2>/dev/null || true
    fi
    if [ -d "${latest_backup}/conf.d" ]; then
        cp -r "${latest_backup}/conf.d" nginx/ 2>/dev/null || true
    fi
    
    $COMPOSE_CMD up -d
    log "INFO" "  ✓ 回滚完成"
    
    return 0
}

deploy() {
    local attempt=1
    local success=false
    
    while [ $attempt -le $MAX_DEPLOY_RETRIES ]; do
        log_separator
        log "INFO" "===== 部署尝试 ${attempt}/${MAX_DEPLOY_RETRIES} ====="
        
        if [ $attempt -gt 1 ]; then
            log "INFO" "等待 ${attempt}0 秒后重试..."
            sleep $((attempt * 10))
        fi
        
        init_deploy_env || true
        check_prerequisites || { log "ERROR" "前置条件检查失败"; attempt=$((attempt + 1)); continue; }
        prepare_directories || true
        generate_ssl_cert || true
        backup_current_version || true
        build_docker_images || { log "ERROR" "Docker镜像构建失败"; attempt=$((attempt + 1)); continue; }
        stop_services || true
        start_services || { log "ERROR" "服务启动失败"; attempt=$((attempt + 1)); continue; }
        wait_for_services || true
        
        if perform_health_check; then
            success=true
            break
        else
            log "WARN" "健康检查失败"
            if [ "$ROLLBACK_ON_FAILURE" = "true" ]; then
                rollback_to_backup || true
            fi
        fi
        
        attempt=$((attempt + 1))
    done
    
    if [ "$success" = true ]; then
        log_separator
        log "INFO" "===== 部署成功 ====="
        return 0
    else
        log_separator
        log "ERROR" "===== 部署失败 (已达最大重试次数) ====="
        return 1
    fi
}

show_service_status() {
    log_separator
    log "INFO" "===== 服务状态 ====="
    
    $COMPOSE_CMD ps
    
    echo ""
    log "INFO" "容器资源使用:"
    docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}" 2>/dev/null || true
}

main() {
    init_deploy_env
    
    if deploy; then
        show_service_status
        show_deployment_summary
        exit 0
    else
        show_service_status
        log_separator
        log "ERROR" "===== 部署失败 ====="
        log "ERROR" "请检查日志: $DEPLOY_LOG_FILE"
        exit 1
    fi
}

show_deployment_summary() {
    log_separator
    log "INFO" "===== 部署摘要 ====="
    log "INFO" "部署时间: $(date '+%Y-%m-%d %H:%M:%S')"
    log "INFO" "部署日志: $DEPLOY_LOG_FILE"
    log "INFO" "备份目录: $BACKUP_DIR"
    log ""
    log "INFO" "服务地址:"
    log "INFO" "  - 应用API: http://localhost:${APP_PORT:-8080}"
    log "INFO" "  - 前端页面: http://localhost:${NGINX_PORT:-80}"
    log "INFO" "  - Prometheus: http://localhost:${PROMETHEUS_PORT:-9090}"
    log "INFO" "  - Grafana: http://localhost:${GRAFANA_PORT:-3000}"
    log "INFO" "  - Loki: http://localhost:${LOKI_PORT:-3100}"
    log ""
    log "INFO" "运维命令:"
    log "INFO" "  查看日志: $COMPOSE_CMD logs -f"
    log "INFO" "  停止服务: $COMPOSE_CMD down"
    log "INFO" "  健康检查: $SCRIPT_DIR/health-check.sh"
    log "INFO" "  回滚部署: $SCRIPT_DIR/rollback.sh"
}

main "$@"
