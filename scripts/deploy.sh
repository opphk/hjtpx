#!/bin/bash
set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOG_FILE="${DIR}/logs/deploy_${TIMESTAMP}.log"
BACKUP_DIR="${DIR}/backups"

log_info() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] INFO: $1"
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] INFO: $1" >> "$LOG_FILE"
}

log_error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $1" | tee -a "$LOG_FILE" >&2
}

log_success() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] SUCCESS: $1"
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] SUCCESS: $1" >> "$LOG_FILE"
}

mkdir -p "${DIR}/logs"
mkdir -p "$BACKUP_DIR"

echo "===== HJTPX 部署脚本 ====="
echo "开始部署..."
echo "日志文件: $LOG_FILE"
echo ""

log_info "===== HJTPX 部署开始 ====="
log_info "部署时间: $(date)"
log_info "工作目录: $DIR"

if [ ! -f ".env" ]; then
    log_error ".env 文件不存在，复制 .env.example 作为模板"
    cp .env.example .env
    echo "请编辑 .env 文件配置您的环境变量"
    echo "完成后重新运行此脚本"
    exit 1
fi

log_info "1. 运行部署前检查..."
if [ -f "./scripts/pre-check.sh" ]; then
    if ./scripts/pre-check.sh >> "$LOG_FILE" 2>&1; then
        log_success "部署前检查通过"
    else
        log_error "部署前检查未通过"
        echo ""
        echo "查看详细日志: cat $LOG_FILE"
        echo "或运行: ./scripts/pre-check.sh"
        exit 1
    fi
else
    log_info "跳过部署前检查 (脚本不存在)"
fi

echo ""
log_info "2. 检查Docker环境..."
if ! command -v docker &> /dev/null; then
    log_error "Docker 未安装"
    exit 1
fi

if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    log_error "Docker Compose 未安装"
    exit 1
fi

DOCKER_COMPOSE_CMD="docker-compose"
if ! command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE_CMD="docker compose"
fi

log_success "Docker 环境检查通过"
log_info "使用命令: $DOCKER_COMPOSE_CMD"

echo ""
log_info "3. 创建必要目录..."
mkdir -p backend/config
mkdir -p nginx/ssl
mkdir -p monitoring/prometheus/rules
mkdir -p monitoring/grafana/provisioning/dashboards
log_success "目录创建完成"

echo ""
log_info "4. 生成SSL证书（用于测试）..."
if [ ! -f "nginx/ssl/cert.pem" ]; then
    openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
        -keyout nginx/ssl/key.pem \
        -out nginx/ssl/cert.pem \
        -subj "/C=CN/ST=Beijing/L=Beijing/O=HJTPX/CN=localhost" 2>/dev/null
    log_success "SSL证书已生成"
else
    log_info "SSL证书已存在，跳过"
fi

echo ""
log_info "5. 创建数据库备份..."
BACKUP_FILE="${BACKUP_DIR}/db_backup_${TIMESTAMP}.sql.gz"
if $DOCKER_COMPOSE_CMD exec -T postgres pg_dump -U postgres hjtpx_db 2>/dev/null | gzip > "$BACKUP_FILE"; then
    log_success "数据库备份已创建: $BACKUP_FILE"
else
    log_info "数据库备份跳过 (可能数据库未运行)"
fi

echo ""
log_info "6. 构建Docker镜像..."
BUILD_START=$(date +%s)

$DOCKER_COMPOSE_CMD build --no-cache app 2>&1 | tee -a "$LOG_FILE"
BUILD_RESULT=$?

BUILD_END=$(date +%s)
BUILD_DURATION=$((BUILD_END - BUILD_START))

if [ $BUILD_RESULT -eq 0 ]; then
    log_success "Docker镜像构建成功 (耗时: ${BUILD_DURATION}秒)"
else
    log_error "Docker镜像构建失败"
    echo ""
    echo "查看构建日志: cat $LOG_FILE"
    exit 1
fi

echo ""
log_info "7. 停止旧容器..."
$DOCKER_COMPOSE_CMD down 2>&1 | tee -a "$LOG_FILE" || true
log_success "旧容器已停止"

echo ""
log_info "8. 启动服务..."
DEPLOY_START=$(date +%s)
$DOCKER_COMPOSE_CMD up -d 2>&1 | tee -a "$LOG_FILE"

if [ $? -eq 0 ]; then
    log_success "服务启动命令已执行"
else
    log_error "服务启动失败"
    echo ""
    echo "查看日志: $DOCKER_COMPOSE_CMD logs -f"
    echo "查看部署日志: cat $LOG_FILE"
    exit 1
fi

echo ""
log_info "9. 等待服务启动..."
log_info "等待时间: 15秒"
sleep 15

echo ""
log_info "10. 检查服务状态..."
$DOCKER_COMPOSE_CMD ps 2>&1 | tee -a "$LOG_FILE"

echo ""
log_info "11. 健康检查..."
HEALTH_CHECK_PASSED=false
for i in {1..10}; do
    if curl -sf "http://localhost:${APP_PORT:-8080}/health" > /dev/null 2>&1; then
        log_success "健康检查通过"
        HEALTH_CHECK_PASSED=true
        break
    fi
    log_info "等待健康检查... ($i/10)"
    sleep 3
done

if [ "$HEALTH_CHECK_PASSED" = false ]; then
    log_error "健康检查失败"
    echo ""
    echo "查看应用日志:"
    $DOCKER_COMPOSE_CMD logs app --tail 50
    echo ""
    echo "查看完整日志: cat $LOG_FILE"
    exit 1
fi

DEPLOY_END=$(date +%s)
DEPLOY_DURATION=$((DEPLOY_END - DEPLOY_START))

log_success "===== 部署完成 ====="
log_info "总耗时: ${DEPLOY_DURATION}秒"
log_info "详细日志: $LOG_FILE"

echo ""
echo "========================================"
echo "        部署完成"
echo "========================================"
echo ""
echo "服务地址:"
echo "  - 应用API: http://localhost:${APP_PORT:-8080}"
echo "  - 前端页面: http://localhost:${NGINX_PORT:-80}"
echo "  - Prometheus: http://localhost:${PROMETHEUS_PORT:-9090}"
echo "  - Grafana: http://localhost:${GRAFANA_PORT:-3000}"
echo "  - Loki: http://localhost:${LOKI_PORT:-3100}"
echo ""
echo "默认管理员账号:"
echo "  - 用户名: admin"
echo "  - 密码: admin123"
echo ""
echo "常用命令:"
echo "  查看日志: $DOCKER_COMPOSE_CMD logs -f"
echo "  停止服务: $DOCKER_COMPOSE_CMD down"
echo "  健康检查: ./scripts/health-check.sh"
echo "  回滚部署: ./scripts/rollback.sh"
echo "  备份数据: ./scripts/backup.sh"
echo ""
echo "日志文件: $LOG_FILE"
echo ""
echo "========================================"

