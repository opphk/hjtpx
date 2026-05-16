#!/bin/bash

set -e

PROJECT_NAME="hjtpx"
DEPLOY_DIR="/opt/${PROJECT_NAME}"
BACKUP_DIR="/opt/${PROJECT_NAME}/backups"
LOG_FILE="/var/log/${PROJECT_NAME}/deploy.log"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "${LOG_FILE}"
}

error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $1" | tee -a "${LOG_FILE}" >&2
    exit 1
}

check_requirements() {
    log "检查部署环境..."

    if ! command -v git &> /dev/null; then
        error "Git 未安装"
    fi

    if ! command -v docker &> /dev/null; then
        error "Docker 未安装"
    fi

    if ! command -v docker-compose &> /dev/null; then
        error "Docker Compose 未安装"
    fi

    log "环境检查完成"
}

backup_current() {
    if [ -d "${DEPLOY_DIR}" ]; then
        log "创建当前版本备份..."
        mkdir -p "${BACKUP_DIR}"
        BACKUP_NAME="${PROJECT_NAME}_${TIMESTAMP}"
        tar -czf "${BACKUP_DIR}/${BACKUP_NAME}.tar.gz" -C "${DEPLOY_DIR}" . 2>/dev/null || true
        log "备份已创建: ${BACKUP_NAME}.tar.gz"

        find "${BACKUP_DIR}" -name "${PROJECT_NAME}_*.tar.gz" -mtime +7 -delete
    fi
}

pull_latest_code() {
    log "拉取最新代码..."

    if [ ! -d "${DEPLOY_DIR}" ]; then
        git clone "$(git remote get-url origin)" "${DEPLOY_DIR}"
        cd "${DEPLOY_DIR}"
    else
        cd "${DEPLOY_DIR}"
        git pull origin main
    fi

    log "代码更新完成"
}

build_docker_images() {
    log "构建 Docker 镜像..."

    cd "${DEPLOY_DIR}"

    docker-compose build --no-cache app

    log "Docker 镜像构建完成"
}

run_database_migrations() {
    log "执行数据库迁移..."

    docker-compose run --rm app /app/main migrate

    log "数据库迁移完成"
}

stop_services() {
    log "停止现有服务..."

    cd "${DEPLOY_DIR}"
    docker-compose down || true

    log "服务已停止"
}

start_services() {
    log "启动新服务..."

    cd "${DEPLOY_DIR}"
    docker-compose up -d

    log "等待服务启动..."
    sleep 10

    if docker-compose ps | grep -q "Up"; then
        log "服务启动成功"
    else
        error "服务启动失败，请检查日志"
    fi
}

verify_deployment() {
    log "验证部署..."

    MAX_RETRIES=30
    RETRY_COUNT=0

    while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
        if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
            log "健康检查通过"
            return 0
        fi

        RETRY_COUNT=$((RETRY_COUNT + 1))
        log "等待服务就绪... (${RETRY_COUNT}/${MAX_RETRIES})"
        sleep 2
    done

    error "健康检查失败"
}

cleanup_old_images() {
    log "清理旧镜像..."

    docker image prune -f

    log "镜像清理完成"
}

main() {
    log "========================================="
    log "开始部署 ${PROJECT_NAME}"
    log "========================================="

    check_requirements
    backup_current
    stop_services
    pull_latest_code
    build_docker_images
    run_database_migrations
    start_services
    verify_deployment
    cleanup_old_images

    log "========================================="
    log "部署完成！"
    log "========================================="
}

if [ "$EUID" -ne 0 ]; then
    echo "请使用 sudo 运行此脚本"
    exit 1
fi

main "$@"
