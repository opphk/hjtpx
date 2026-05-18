#!/bin/sh
# =============================================================================
# Docker容器启动脚本
# =============================================================================

set -e

echo "Starting hjtpx application..."

# =============================================================================
# 环境变量验证
# =============================================================================

validate_env() {
    echo "Validating environment variables..."

    # 必需的环境变量
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
            echo "Error: $var is not set"
            missing_vars="$missing_vars $var"
        fi
    done

    if [ -n "$missing_vars" ]; then
        echo "Error: Missing required environment variables:$missing_vars"
        exit 1
    fi

    # JWT密钥长度检查
    if [ ${#JWT_SECRET} -lt 32 ]; then
        echo "Warning: JWT_SECRET should be at least 32 characters long for security"
    fi

    echo "Environment variables validated successfully"
}

# =============================================================================
# 依赖服务健康检查
# =============================================================================

wait_for_postgres() {
    echo "Waiting for PostgreSQL at ${POSTGRES_HOST}:${POSTGRES_PORT}..."
    max_attempts=30
    attempt=1

    while [ $attempt -le $max_attempts ]; do
        if pg_isready -h "${POSTGRES_HOST}" -p "${POSTGRES_PORT}" -U "${POSTGRES_USER}" > /dev/null 2>&1; then
            echo "PostgreSQL is ready!"
            return 0
        fi
        echo "PostgreSQL is unavailable (attempt $attempt/$max_attempts) - sleeping"
        sleep 2
        attempt=$((attempt + 1))
    done

    echo "Error: PostgreSQL failed to start after $max_attempts attempts"
    return 1
}

wait_for_redis() {
    echo "Waiting for Redis at ${REDIS_HOST}:${REDIS_PORT}..."

    redis_password="${REDIS_PASSWORD:-}"
    max_attempts=30
    attempt=1

    while [ $attempt -le $max_attempts ]; do
        if [ -n "$redis_password" ]; then
            if redis-cli -h "${REDIS_HOST}" -p "${REDIS_PORT}" -a "$redis_password" ping > /dev/null 2>&1; then
                echo "Redis is ready!"
                return 0
            fi
        else
            if redis-cli -h "${REDIS_HOST}" -p "${REDIS_PORT}" ping > /dev/null 2>&1; then
                echo "Redis is ready!"
                return 0
            fi
        fi
        echo "Redis is unavailable (attempt $attempt/$max_attempts) - sleeping"
        sleep 2
        attempt=$((attempt + 1))
    done

    echo "Error: Redis failed to start after $max_attempts attempts"
    return 1
}

# =============================================================================
# 系统设置
# =============================================================================

setup_system() {
    echo "Setting up system environment..."

    # 创建必要的目录
    mkdir -p /var/log/hjtpx
    mkdir -p /tmp/hjtpx

    # 设置日志目录权限
    chmod 755 /var/log/hjtpx
    chmod 1777 /tmp/hjtpx

    echo "System environment setup completed"
}

# =============================================================================
# 应用启动
# =============================================================================

start_application() {
    echo "Starting application..."
    echo "Server will listen on port ${SERVER_PORT:-8080}"
    echo "Gin mode: ${GIN_MODE:-release}"

    # 执行应用程序
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
