#!/bin/sh
# =============================================================================
# 健康检查脚本 - 检查数据库连接、Redis连接和API响应
# =============================================================================

set -e

POSTGRES_HOST="${DATABASE_HOST:-postgres}"
POSTGRES_PORT="${DATABASE_PORT:-5432}"
POSTGRES_USER="${DATABASE_USER:-postgres}"
POSTGRES_PASSWORD="${DATABASE_PASSWORD:-}"
POSTGRES_DB="${DATABASE_NAME:-hjtpx_db}"
REDIS_HOST="${REDIS_HOST:-redis}"
REDIS_PORT="${REDIS_PORT:-6379}"
REDIS_PASSWORD="${REDIS_PASSWORD:-}"
APP_HOST="${APP_HOST:-localhost}"
APP_PORT="${SERVER_PORT:-8080}"
APP_URL="http://${APP_HOST}:${APP_PORT}"

check_postgres() {
    if command -v pg_isready > /dev/null 2>&1; then
        if [ -n "$POSTGRES_PASSWORD" ]; then
            PGPASSWORD="$POSTGRES_PASSWORD" pg_isready -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" > /dev/null 2>&1
        else
            pg_isready -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" > /dev/null 2>&1
        fi
    else
        return 0
    fi
}

check_redis() {
    if command -v redis-cli > /dev/null 2>&1; then
        if [ -n "$REDIS_PASSWORD" ]; then
            redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" -a "$REDIS_PASSWORD" ping > /dev/null 2>&1
        else
            redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" ping > /dev/null 2>&1
        fi
    else
        return 0
    fi
}

check_api() {
    if command -v wget > /dev/null 2>&1; then
        wget --no-verbose --tries=1 --spider "${APP_URL}/health" 2>/dev/null
    elif command -v curl > /dev/null 2>&1; then
        curl -sf "${APP_URL}/health" > /dev/null 2>&1
    else
        return 0
    fi
}

if [ "$1" = "--quick" ]; then
    check_api
else
    echo "=== HJTPX 健康检查 ==="
    echo "检查数据库连接..."
    check_postgres && echo "✓ 数据库连接正常" || echo "✗ 数据库连接失败"

    echo "检查Redis连接..."
    check_redis && echo "✓ Redis连接正常" || echo "✗ Redis连接失败"

    echo "检查API响应..."
    check_api && echo "✓ API响应正常" || echo "✗ API响应失败"

    check_postgres && check_redis && check_api
fi
