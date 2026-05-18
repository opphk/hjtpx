#!/bin/sh
# =============================================================================
# 健康检查脚本 - 用于Docker健康检查
# =============================================================================

set -e

APP_HOST="${APP_HOST:-localhost}"
APP_PORT="${SERVER_PORT:-8080}"
APP_URL="http://${APP_HOST}:${APP_PORT}"

MAX_RETRIES="${HEALTH_CHECK_RETRIES:-3}"
RETRY_INTERVAL="${HEALTH_CHECK_INTERVAL:-5}"

# 检查应用健康状态
check_health() {
    if command -v wget > /dev/null 2>&1; then
        wget --no-verbose --tries=1 --spider "${APP_URL}/health" 2>/dev/null
    elif command -v curl > /dev/null 2>&1; then
        curl -sf "${APP_URL}/health" > /dev/null 2>&1
    else
        echo "Error: Neither wget nor curl is available"
        return 1
    fi
}

# 带有重试的健康检查
retry_check() {
    retries=0
    while [ $retries -lt $MAX_RETRIES ]; do
        if check_health; then
            echo "Health check passed"
            exit 0
        fi
        retries=$((retries + 1))
        echo "Health check failed (attempt $retries/$MAX_RETRIES), retrying in ${RETRY_INTERVAL}s..."
        sleep $RETRY_INTERVAL
    done

    echo "Health check failed after $MAX_RETRIES attempts"
    exit 1
}

# 如果设置了单次检查模式，直接检查
if [ "$1" = "--quick" ]; then
    check_health
else
    retry_check
fi
