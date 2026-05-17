#!/bin/bash

PROD_DIR="/opt/hjtpx"
CONFIG_FILE="$PROD_DIR/config/config.yaml"
LOG_FILE="/var/log/hjtpx/startup.log"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

log "开始预检查..."

if [ ! -f "$CONFIG_FILE" ]; then
    log "错误: 配置文件不存在: $CONFIG_FILE"
    exit 1
fi

REDIS_HOST=$(grep -A5 "redis:" "$CONFIG_FILE" | grep "host:" | awk '{print $2}')
REDIS_PORT=$(grep -A5 "redis:" "$CONFIG_FILE" | grep "port:" | awk '{print $2}')

if [ -n "$REDIS_HOST" ] && [ -n "$REDIS_PORT" ]; then
    if ! timeout 5 bash -c "echo > /dev/tcp/$REDIS_HOST/$REDIS_PORT" 2>/dev/null; then
        log "警告: Redis 连接失败: $REDIS_HOST:$REDIS_PORT"
    else
        log "✓ Redis 连接正常: $REDIS_HOST:$REDIS_PORT"
    fi
fi

POSTGRES_HOST=$(grep -A5 "database:" "$CONFIG_FILE" | grep "host:" | awk '{print $2}')
POSTGRES_PORT=$(grep -A5 "database:" "$CONFIG_FILE" | grep "port:" | awk '{print $2}')

if [ -n "$POSTGRES_HOST" ] && [ -n "$POSTGRES_PORT" ]; then
    if ! timeout 5 bash -c "echo > /dev/tcp/$POSTGRES_HOST/$POSTGRES_PORT" 2>/dev/null; then
        log "警告: PostgreSQL 连接失败: $POSTGRES_HOST:$POSTGRES_PORT"
    else
        log "✓ PostgreSQL 连接正常: $POSTGRES_HOST:$POSTGRES_PORT"
    fi
fi

if [ ! -x "$PROD_DIR/hjtpx-server" ]; then
    log "错误: 服务器二进制文件不可执行或不存在"
    exit 1
fi

log "✓ 预检查完成"
exit 0
