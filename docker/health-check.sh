#!/bin/sh
# =============================================================================
# 健康检查脚本 - 用于Docker健康检查
# 功能：端到端检查、告警机制、性能监控
# 版本：v21.0
# =============================================================================

set -e

APP_HOST="${APP_HOST:-localhost}"
APP_PORT="${SERVER_PORT:-8080}"
APP_URL="http://${APP_HOST}:${APP_PORT}"

MAX_RETRIES="${HEALTH_CHECK_RETRIES:-3}"
RETRY_INTERVAL="${HEALTH_CHECK_INTERVAL:-5}"
TIMEOUT="${HEALTH_CHECK_TIMEOUT:-10}"

LOG_FILE="${LOG_FILE:-/var/log/hjtpx/health-check.log}"
METRICS_FILE="${METRICS_FILE:-/var/log/hjtpx/health-metrics.log}"

log_message() {
    echo "[$(date -u +'%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE" 2>/dev/null || echo "[$(date -u +'%Y-%m-%d %H:%M:%S')] $1"
}

log_error() {
    echo "[$(date -u +'%Y-%m-%d %H:%M:%S')] ERROR: $1" | tee -a "$LOG_FILE" 2>/dev/null || echo "[$(date -u +'%Y-%m-%d %H:%M:%S')] ERROR: $1"
}

# =============================================================================
# 依赖服务检查
# =============================================================================

check_postgres() {
    if command -v pg_isready > /dev/null 2>&1; then
        if pg_isready -h "${DATABASE_HOST:-postgres}" -p "${DATABASE_PORT:-5432}" -U "${DATABASE_USER:-postgres}" > /dev/null 2>&1; then
            return 0
        fi
    fi
    return 1
}

check_redis() {
    if command -v redis-cli > /dev/null 2>&1; then
        redis_password="${REDIS_PASSWORD:-}"
        if [ -n "$redis_password" ]; then
            if redis-cli -h "${REDIS_HOST:-redis}" -p "${REDIS_PORT:-6379}" -a "$redis_password" ping > /dev/null 2>&1; then
                return 0
            fi
        else
            if redis-cli -h "${REDIS_HOST:-redis}" -p "${REDIS_PORT:-6379}" ping > /dev/null 2>&1; then
                return 0
            fi
        fi
    fi
    return 1
}

# =============================================================================
# 应用健康状态检查
# =============================================================================

check_health() {
    start_time=$(date +%s)

    if command -v wget > /dev/null 2>&1; then
        wget_output=$(wget --timeout="${TIMEOUT}" --no-verbose --tries=1 --spider "${APP_URL}/health" 2>&1)
        wget_exit=$?
        end_time=$(date +%s)
        response_time=$((end_time - start_time))

        if [ $wget_exit -eq 0 ]; then
            log_message "Health check passed (wget) - Response time: ${response_time}s"
            return 0
        else
            log_error "Health check failed (wget): ${wget_output}"
            return 1
        fi
    elif command -v curl > /dev/null 2>&1; then
        response=$(curl -sf --max-time "${TIMEOUT}" "${APP_URL}/health" 2>&1)
        curl_exit=$?
        end_time=$(date +%s)
        response_time=$((end_time - start_time))

        if [ $curl_exit -eq 0 ]; then
            log_message "Health check passed (curl) - Response: ${response} - Time: ${response_time}s"
            return 0
        else
            log_error "Health check failed (curl): ${response}"
            return 1
        fi
    else
        log_error "Error: Neither wget nor curl is available"
        return 1
    fi
}

# =============================================================================
# 端到端检查
# =============================================================================

check_end_to_end() {
    log_message "Starting end-to-end health check..."

    # 检查依赖服务
    if ! check_postgres; then
        log_error "PostgreSQL health check failed"
        return 1
    fi
    log_message "PostgreSQL: OK"

    if ! check_redis; then
        log_error "Redis health check failed"
        return 1
    fi
    log_message "Redis: OK"

    # 检查应用健康
    if ! check_health; then
        return 1
    fi

    log_message "End-to-end health check passed"
    return 0
}

# =============================================================================
# 系统状态检查
# =============================================================================

check_system_status() {
    log_message "System status:"
    log_message "  Memory usage: $(free -h 2>/dev/null | awk '/^Mem:/ {print $3 "/" $2}')"
    log_message "  Disk usage: $(df -h / 2>/dev/null | awk 'NR==2 {print $5 " (" $4 " available)"}')"
    log_message "  Load average: $(uptime 2>/dev/null | awk -F'load average:' '{print $2}')"
}

# =============================================================================
# 性能指标记录
# =============================================================================

record_metrics() {
    timestamp=$(date -u +'%Y-%m-%d %H:%M:%S')

    if command -v curl > /dev/null 2>&1; then
        response_time=$(curl -o /dev/null -s -w '%{time_total}' --max-time "${TIMEOUT}" "${APP_URL}/health" 2>/dev/null)
        echo "${timestamp},${response_time},healthy" >> "$METRICS_FILE" 2>/dev/null || true
    fi
}

# =============================================================================
# 告警机制
# =============================================================================

send_alert() {
    alert_message="$1"
    log_error "ALERT: ${alert_message}"

    # 如果配置了WEBHOOK_URL，发送告警
    if [ -n "${WEBHOOK_URL}" ]; then
        if command -v curl > /dev/null 2>&1; then
            curl -sf -X POST "${WEBHOOK_URL}" \
                -H "Content-Type: application/json" \
                -d "{\"text\":\"[HJTPX Alert] ${alert_message}\"}" > /dev/null 2>&1 || true
        fi
    fi
}

# =============================================================================
# 带重试的健康检查
# =============================================================================

retry_check() {
    retries=0
    consecutive_failures=0

    while [ $retries -lt $MAX_RETRIES ]; do
        log_message "Health check attempt $((retries + 1))/$MAX_RETRIES"

        if check_end_to_end; then
            consecutive_failures=0
            record_metrics
            exit 0
        else
            consecutive_failures=$((consecutive_failures + 1))

            if [ $consecutive_failures -ge 2 ]; then
                check_system_status
                send_alert "Consecutive health check failures: ${consecutive_failures}"
            fi

            retries=$((retries + 1))
            if [ $retries -lt $MAX_RETRIES ]; then
                log_message "Health check failed (attempt $retries/$MAX_RETRIES), retrying in ${RETRY_INTERVAL}s..."
                sleep $RETRY_INTERVAL
            fi
        fi
    done

    log_error "Health check failed after $MAX_RETRIES attempts"
    check_system_status
    send_alert "Health check failed after ${MAX_RETRIES} attempts - Manual intervention required"
    exit 1
}

# =============================================================================
# 主流程
# =============================================================================

main() {
    case "$1" in
        --quick)
            log_message "Running quick health check"
            check_health
            ;;
        --e2e)
            log_message "Running end-to-end health check"
            check_end_to_end
            ;;
        --postgres)
            log_message "Checking PostgreSQL"
            check_postgres
            ;;
        --redis)
            log_message "Checking Redis"
            check_redis
            ;;
        --system)
            log_message "Checking system status"
            check_system_status
            ;;
        *)
            retry_check
            ;;
    esac
}

main "$@"
