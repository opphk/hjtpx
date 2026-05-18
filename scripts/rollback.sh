#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ENVIRONMENT="${1:-staging}"
VERSION="${2:-previous}"
APP_NAME="hjtpx-${ENVIRONMENT}"
ENV_FILE="/opt/hjtpx/.env.${ENVIRONMENT}"
BACKUP_DIR="/opt/hjtpx/backups"
DEPLOYMENT_METADATA="/opt/hjtpx/deployments"

log_info() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] INFO: $1"
}

log_error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $1" >&2
}

log_success() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] SUCCESS: $1"
}

check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "此脚本需要 root 权限运行"
        echo "请使用: sudo $0 [environment] [version]"
        exit 1
    fi
}

check_docker() {
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装"
        exit 1
    fi
}

check_environment() {
    if [[ ! "$ENVIRONMENT" =~ ^(staging|production)$ ]]; then
        log_error "无效的环境: $ENVIRONMENT"
        echo "有效环境: staging, production"
        exit 1
    fi
}

get_current_container() {
    log_info "获取当前容器信息..."
    CURRENT_IMAGE=$(docker inspect --format='{{.Config.Image}}' ${APP_NAME} 2>/dev/null || echo "none")
    log_info "当前镜像: $CURRENT_IMAGE"
}

get_available_versions() {
    log_info "获取可用版本..."
    echo ""
    echo "=== 可用的回滚版本 ==="
    docker images | grep ${APP_NAME} | grep -v current || echo "无历史版本"
    echo ""
}

list_deployments() {
    log_info "部署历史..."
    if [ -d "$DEPLOYMENT_METADATA" ]; then
        echo ""
        ls -lt "$DEPLOYMENT_METADATA" | head -20
        echo ""
    else
        echo "无部署记录"
    fi
}

create_emergency_backup() {
    log_info "创建紧急备份..."
    
    TIMESTAMP=$(date +%Y%m%d_%H%M%S)
    BACKUP_PATH="${BACKUP_DIR}/emergency_${TIMESTAMP}"
    mkdir -p "$BACKUP_PATH"
    
    if docker inspect ${APP_NAME} &>/dev/null; then
        docker commit ${APP_NAME} ${APP_NAME}:emergency_${TIMESTAMP}
        log_success "紧急备份创建成功: ${APP_NAME}:emergency_${TIMESTAMP}"
        
        cat > "${BACKUP_PATH}/metadata.json" <<EOF
{
    "type": "emergency_backup",
    "container": "${APP_NAME}",
    "image": "${CURRENT_IMAGE}",
    "timestamp": "${TIMESTAMP}",
    "rollback_to": "${VERSION}"
}
EOF
    else
        log_error "无法备份当前容器"
    fi
}

pre_rollback_check() {
    log_info "回滚前检查..."
    
    if [ ! -f "$ENV_FILE" ]; then
        log_error "环境配置文件不存在: $ENV_FILE"
        exit 1
    fi
    
    if docker images | grep -q "${APP_NAME}:${VERSION}"; then
        log_success "目标版本存在: ${APP_NAME}:${VERSION}"
    else
        log_error "目标版本不存在: ${APP_NAME}:${VERSION}"
        get_available_versions
        exit 1
    fi
}

perform_rollback() {
    log_info "开始回滚到版本: $VERSION"
    
    log_info "停止当前容器..."
    docker stop ${APP_NAME} || true
    docker rm ${APP_NAME} || true
    
    log_info "启动回滚容器..."
    docker run -d \
        --name ${APP_NAME} \
        -p ${ENVIRONMENT == 'staging' && echo '8080:8080' || echo '8080:8080'} \
        --env-file "$ENV_FILE" \
        --restart unless-stopped \
        --memory-reservation=256m \
        --memory=512m \
        --cpus=1.0 \
        --health-cmd="wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1" \
        --health-interval=30s \
        --health-timeout=10s \
        --health-retries=3 \
        --health-start-period=15s \
        --log-driver=json-file \
        --log-opt=max-size=10m \
        --log-opt=max-file=5 \
        ${APP_NAME}:${VERSION}
    
    log_info "等待容器启动..."
    sleep 10
}

verify_rollback() {
    log_info "验证回滚结果..."
    
    MAX_RETRIES=12
    RETRY_INTERVAL=10
    
    for i in $(seq 1 $MAX_RETRIES); do
        if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
            log_success "健康检查通过!"
            return 0
        fi
        log_info "等待健康检查... ($i/$MAX_RETRIES)"
        sleep $RETRY_INTERVAL
    done
    
    log_error "健康检查失败"
    log_info "查看容器日志:"
    docker logs ${APP_NAME} --tail 50
    return 1
}

show_rollback_info() {
    echo ""
    echo "===== 回滚完成 ====="
    echo ""
    echo "环境: $ENVIRONMENT"
    echo "回滚版本: $VERSION"
    echo "当前镜像: $(docker inspect --format='{{.Config.Image}}' ${APP_NAME} 2>/dev/null || echo 'unknown')"
    echo ""
    echo "容器状态:"
    docker ps --filter "name=${APP_NAME}"
    echo ""
    echo "资源使用:"
    docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}" ${APP_NAME} || true
    echo ""
}

main() {
    echo "===== HJTPX 回滚脚本 ====="
    echo ""
    
    if [ "$1" == "--help" ] || [ "$1" == "-h" ]; then
        echo "用法: $0 [environment] [version]"
        echo ""
        echo "参数:"
        echo "  environment  - 环境名称 (staging, production) [默认: staging]"
        echo "  version       - 回滚版本 [默认: previous]"
        echo ""
        echo "示例:"
        echo "  $0 staging previous"
        echo "  $0 production v1.2.3"
        echo "  $0 production emergency_20260518_143022"
        echo ""
        echo "命令:"
        echo "  --list        - 列出可用版本"
        echo "  --history     - 显示部署历史"
        echo "  --help        - 显示帮助"
        exit 0
    fi
    
    if [ "$1" == "--list" ]; then
        get_available_versions
        exit 0
    fi
    
    if [ "$1" == "--history" ]; then
        list_deployments
        exit 0
    fi
    
    check_root
    check_docker
    check_environment
    
    log_info "开始回滚流程..."
    log_info "环境: $ENVIRONMENT"
    log_info "目标版本: $VERSION"
    
    get_current_container
    get_available_versions
    create_emergency_backup
    pre_rollback_check
    perform_rollback
    
    if verify_rollback; then
        show_rollback_info
        log_success "回滚完成!"
        
        echo ""
        echo "下一步操作:"
        echo "  1. 验证应用功能正常"
        echo "  2. 通知相关人员"
        echo "  3. 检查监控和日志"
        echo ""
        
        exit 0
    else
        log_error "回滚失败，请检查日志"
        
        echo ""
        echo "紧急恢复选项:"
        echo "  1. 检查 docker logs ${APP_NAME}"
        echo "  2. 运行: docker run -it --rm ${APP_NAME}:emergency_* /bin/sh"
        echo "  3. 联系运维团队"
        echo ""
        
        exit 1
    fi
}

main "$@"
