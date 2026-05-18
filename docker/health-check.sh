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
TIMEOUT="${HEALTH_CHECK_TIMEOUT:-10}"

LOG_FILE="${LOG_FILE:-/var/log/hjtpx/health-check.log}"

log_message() {
    echo "[$(date -u +'%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE" 2>/dev/null || echo "[$(date -u +'%Y-%m-%d %H:%M:%S')] $1"
}

# 检查应用健康状态
check_health() {
    log_message "Checking health at ${APP_URL}/health"

    if command -v wget > /dev/null 2>&1; then
        log_message "Using wget for health check"
        if wget --timeout="${TIMEOUT}" --no-verbose --tries=1 --spider "${APP_URL}/health" 2>&1 | tee -a "$LOG_FILE"; then
            log_message "Health check passed (wget)"
            return 0
        else
            log_message "Health check failed (wget)"
            return 1
        fi
    elif command -v curl > /dev/null 2>&1; then
        log_message "Using curl for health check"
        response=$(curl -sf --max-time "${TIMEOUT}" "${APP_URL}/health" 2>&1)
        if [ $? -eq 0 ]; then
            log_message "Health check passed (curl): ${response}"
            return 0
        else
            log_message "Health check failed (curl): ${response}"
            return 1
        fi
    else
        log_message "Error: Neither wget nor curl is available"
        return 1
    fi
}

# 带有重试的健康检查
retry_check() {
    retries=0
    while [ $retries -lt $MAX_RETRIES ]; do
        log_message "Health check attempt $((retries + 1))/$MAX_RETRIES"
        if check_health; then
            log_message "Health check passed"
            exit 0
        fi
        retries=$((retries + 1))
        if [ $retries -lt $MAX_RETRIES ]; then
            log_message "Health check failed (attempt $retries/$MAX_RETRIES), retrying in ${RETRY_INTERVAL}s..."
            sleep $RETRY_INTERVAL
        fi
    done

    log_message "Health check failed after $MAX_RETRIES attempts"
    log_message "Last health check response time: $(date -u +'%Y-%m-%d %H:%M:%S')"
    log_message "Checking system status..."
    log_message "Memory usage: $(free -h 2>/dev/null || echo 'N/A')"
    log_message "Disk usage: $(df -h / 2>/dev/null || echo 'N/A')"
    exit 1
}

# 如果设置了单次检查模式，直接检查
if [ "$1" = "--quick" ]; then
    log_message "Running quick health check"
    check_health
else
    retry_check
fi
