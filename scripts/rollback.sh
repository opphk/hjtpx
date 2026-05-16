#!/bin/bash

set -e

PROJECT_NAME="hjtpx"
DEPLOY_DIR="/opt/${PROJECT_NAME}"
BACKUP_DIR="/opt/${PROJECT_NAME}/backups"
LOG_FILE="/var/log/${PROJECT_NAME}/rollback.log"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "${LOG_FILE}"
}

error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $1" | tee -a "${LOG_FILE}" >&2
    exit 1
}

show_usage() {
    echo "用法: $0 <版本号|备份文件名>"
    echo ""
    echo "示例:"
    echo "  $0 20240115_120000    # 回滚到指定时间戳版本"
    echo "  $0 hjtpx_20240115_120000.tar.gz  # 回滚到指定备份文件"
    echo "  $0 previous           # 回滚到上一个版本"
    echo ""
    echo "可用备份:"
    ls -lh "${BACKUP_DIR}" 2>/dev/null || echo "没有找到备份文件"
}

list_backups() {
    log "可用的备份版本:"
    echo ""
    ls -lht "${BACKUP_DIR}"/*.tar.gz 2>/dev/null | head -10 || echo "没有找到备份文件"
    echo ""
}

rollback_from_backup() {
    local BACKUP_FILE="$1"

    if [ ! -f "${BACKUP_DIR}/${BACKUP_FILE}" ]; then
        error "备份文件不存在: ${BACKUP_FILE}"
    fi

    log "从备份回滚: ${BACKUP_FILE}"

    log "停止当前服务..."
    cd "${DEPLOY_DIR}"
    docker-compose down

    log "恢复备份..."
    rm -rf "${DEPLOY_DIR:?}"/*
    tar -xzf "${BACKUP_DIR}/${BACKUP_FILE}" -C "${DEPLOY_DIR}"

    log "重新构建并启动..."
    docker-compose build app
    docker-compose up -d

    log "等待服务启动..."
    sleep 10

    if docker-compose ps | grep -q "Up"; then
        log "回滚完成"
    else
        error "服务启动失败"
    fi
}

rollback_to_previous() {
    log "回滚到上一个版本..."

    LATEST_BACKUP=$(ls -t "${BACKUP_DIR}"/${PROJECT_NAME}_*.tar.gz 2>/dev/null | head -1)

    if [ -z "${LATEST_BACKUP}" ]; then
        error "没有找到可用的备份"
    fi

    BASENAME=$(basename "${LATEST_BACKUP}")
    rollback_from_backup "${BASENAME}"
}

stop_services() {
    log "停止当前服务..."
    cd "${DEPLOY_DIR}"
    docker-compose down
}

restore_and_restart() {
    local VERSION="$1"

    log "回滚到版本: ${VERSION}"

    if [ -f "${BACKUP_DIR}/${PROJECT_NAME}_${VERSION}.tar.gz" ]; then
        rollback_from_backup "${PROJECT_NAME}_${VERSION}.tar.gz"
    elif [ -f "${BACKUP_DIR}/${VERSION}" ]; then
        rollback_from_backup "${VERSION}"
    else
        error "未找到指定版本: ${VERSION}"
    fi
}

verify_rollback() {
    log "验证回滚..."

    MAX_RETRIES=30
    RETRY_COUNT=0

    while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
        if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
            log "健康检查通过"
            return 0
        fi

        RETRY_COUNT=$((RETRY_COUNT + 1))
        sleep 2
    done

    error "健康检查失败"
}

main() {
    if [ -z "$1" ]; then
        show_usage
        exit 1
    fi

    if [ "$1" = "--list" ] || [ "$1" = "-l" ]; then
        list_backups
        exit 0
    fi

    if [ ! -d "${DEPLOY_DIR}" ]; then
        error "部署目录不存在: ${DEPLOY_DIR}"
    fi

    if [ ! -d "${BACKUP_DIR}" ]; then
        error "备份目录不存在: ${BACKUP_DIR}"
    fi

    log "========================================="
    log "开始回滚 ${PROJECT_NAME}"
    log "========================================="

    if [ "$1" = "previous" ]; then
        rollback_to_previous
    else
        restore_and_restart "$1"
    fi

    verify_rollback

    log "========================================="
    log "回滚完成！"
    log "========================================="
}

if [ "$EUID" -ne 0 ]; then
    echo "请使用 sudo 运行此脚本"
    exit 1
fi

main "$@"
