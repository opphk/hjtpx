#!/bin/sh
# =============================================================================
# Docker容器启动脚本
# 功能：环境验证、依赖服务检查、优雅启动、告警机制
# 版本：v21.0
# =============================================================================

set -e

LOG_FILE="/var/log/hjtpx/docker-entrypoint.log"
METRICS_FILE="/var/log/hjtpx/startup-metrics.log"

log_message() {
    echo "[$(date -u +'%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE" 2>/dev/null || echo "[$(date -u +'%Y-%m-%d %H:%M:%S')] $1"
}

log_error() {
    echo "[$(date -u +'%Y-%m-%d %H:%M:%S')] ERROR: $1" | tee -a "$LOG_FILE" 2>/dev/null || echo "[$(date -u +'%Y-%m-%d %H:%M:%S')] ERROR: $1"
}

log_warning() {
    echo "[$(date -u +'%Y-%m-%d %H:%M:%S')] WARNING: $1" | tee -a "$LOG_FILE" 2>/dev/null || echo "[$(date -u +'%Y-%m-%d %H:%M:%S')] WARNING: $1"
}

log_success() {
    echo "[$(date -u +'%Y-%m-%d %H:%M:%S')] SUCCESS: $1" | tee -a "$LOG_FILE" 2>/dev/null || echo "[$(date -u +'%Y-%m-%d %H:%M:%S')] SUCCESS: $1"
}

echo "========================================"
echo "  HJTPX v21.0 Docker Container Startup"
echo "========================================"
log_message "Starting hjtpx application..."

# =============================================================================
# 告警机制
# =============================================================================

send_alert() {
    alert_message="$1"
    log_error "ALERT: ${alert_message}"

    if [ -n "${WEBHOOK_URL}" ]; then
        if command -v curl > /dev/null 2>&1; then
            curl -sf -X POST "${WEBHOOK_URL}" \
                -H "Content-Type: application/json" \
                -d "{\"text\":\"[HJTPX Container Alert] ${alert_message}\"}" > /dev/null 2>&1 || true
        fi
    fi
}

# =============================================================================
# 环境变量验证
# =============================================================================

validate_env() {
    log_message "Validating environment variables..."

    REQUIRED_VARS="
        POSTGRES_HOST
        POSTGRES_PORT
        POSTGRES_USER
        POSTGRES_PASSWORD
        POSTGRES_DB
        REDIS_HOST
        REDIS_PORT
        JWT_SECRET
    "

    missing_vars=""
    for var in $REQUIRED_VARS; do
        value=$(eval echo \$$var)
        if [ -z "$value" ]; then
            log_error "$var is not set"
            missing_vars="$missing_vars $var"
        fi
    done

    if [ -n "$missing_vars" ]; then
        log_error "Missing required environment variables:$missing_vars"
        send_alert "Missing required environment variables:$missing_vars"
        exit 1
    fi

    if [ ${#JWT_SECRET} -lt 32 ]; then
        log_warning "JWT_SECRET should be at least 32 characters long for security"
    fi

    log_message "Environment variables validated successfully"
}

# =============================================================================
# 依赖服务健康检查
# =============================================================================

wait_for_postgres() {
    log_message "Waiting for PostgreSQL at ${POSTGRES_HOST}:${POSTGRES_PORT}..."
    max_attempts="${POSTGRES_MAX_ATTEMPTS:-30}"
    attempt=1

    while [ $attempt -le $max_attempts ]; do
        if pg_isready -h "${POSTGRES_HOST}" -p "${POSTGRES_PORT}" -U "${POSTGRES_USER}" > /dev/null 2>&1; then
            log_success "PostgreSQL is ready!"

            if [ -n "${POSTGRES_DB}" ]; then
                log_message "Verifying database connection..."
                if PGPASSWORD="${POSTGRES_PASSWORD}" psql -h "${POSTGRES_HOST}" -p "${POSTGRES_PORT}" -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" -c "SELECT 1" > /dev/null 2>&1; then
                    log_success "Database connection verified"
                else
                    log_warning "Database connection verification failed"
                fi
            fi

            return 0
        fi
        log_message "PostgreSQL is unavailable (attempt $attempt/$max_attempts) - sleeping"
        sleep 2
        attempt=$((attempt + 1))
    done

    log_error "PostgreSQL failed to start after $max_attempts attempts"
    log_error "PostgreSQL connection details: host=${POSTGRES_HOST}, port=${POSTGRES_PORT}, user=${POSTGRES_USER}"
    send_alert "PostgreSQL failed to start after ${max_attempts} attempts"
    return 1
}

wait_for_redis() {
    log_message "Waiting for Redis at ${REDIS_HOST}:${REDIS_PORT}..."

    redis_password="${REDIS_PASSWORD:-}"
    max_attempts="${REDIS_MAX_ATTEMPTS:-30}"
    attempt=1

    while [ $attempt -le $max_attempts ]; do
        if [ -n "$redis_password" ]; then
            if redis-cli -h "${REDIS_HOST}" -p "${REDIS_PORT}" -a "$redis_password" ping > /dev/null 2>&1; then
                log_success "Redis is ready!"

                log_message "Verifying Redis connection..."
                if [ -n "$redis_password" ]; then
                    redis_test=$(redis-cli -h "${REDIS_HOST}" -p "${REDIS_PORT}" -a "$redis_password" set healthcheck test > /dev/null 2>&1)
                else
                    redis_test=$(redis-cli -h "${REDIS_HOST}" -p "${REDIS_PORT}" set healthcheck test > /dev/null 2>&1)
                fi

                if [ $? -eq 0 ]; then
                    log_success "Redis connection verified"
                else
                    log_warning "Redis connection verification failed"
                fi

                return 0
            fi
        else
            if redis-cli -h "${REDIS_HOST}" -p "${REDIS_PORT}" ping > /dev/null 2>&1; then
                log_success "Redis is ready!"
                log_message "Redis connection verified"
                return 0
            fi
        fi
        log_message "Redis is unavailable (attempt $attempt/$max_attempts) - sleeping"
        sleep 2
        attempt=$((attempt + 1))
    done

    log_error "Redis failed to start after $max_attempts attempts"
    log_error "Redis connection details: host=${REDIS_HOST}, port=${REDIS_PORT}"
    send_alert "Redis failed to start after ${max_attempts} attempts"
    return 1
}

# =============================================================================
# 系统设置
# =============================================================================

setup_system() {
    log_message "Setting up system environment..."

    mkdir -p /var/log/hjtpx
    mkdir -p /tmp/hjtpx

    chmod 755 /var/log/hjtpx
    chmod 1777 /tmp/hjtpx

    log_message "System environment setup completed"
}

# =============================================================================
# 启动指标记录
# =============================================================================

record_startup_metrics() {
    start_time=$(date +%s)
    echo "[$(date -u +'%Y-%m-%d %H:%M:%S')] startup_time: ${start_time}" >> "$METRICS_FILE" 2>/dev/null || true
}

# =============================================================================
# 应用启动
# =============================================================================

start_application() {
    log_message "Starting application..."
    log_message "Server will listen on port ${SERVER_PORT:-8080}"
    log_message "Gin mode: ${GIN_MODE:-release}"
    log_message "Build version: ${BUILD_VERSION:-v21.0}"

    record_startup_metrics

    exec /server
}

# =============================================================================
# 主流程
# =============================================================================

main() {
    validate_env
    wait_for_postgres
    wait_for_redis
    setup_system
    start_application
}

main "$@"
