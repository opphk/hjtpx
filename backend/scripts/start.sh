#!/bin/sh
# =============================================================================
# 启动脚本 - 优化版本，添加环境变量验证和依赖检查
# =============================================================================

set -e

echo "Starting hjtpx application..."

validate_env() {
    echo "Validating environment variables..."

    REQUIRED_VARS="
        DATABASE_HOST
        DATABASE_PORT
        DATABASE_USER
        DATABASE_PASSWORD
        DATABASE_NAME
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

    if [ ${#JWT_SECRET} -lt 32 ]; then
        echo "Warning: JWT_SECRET should be at least 32 characters long for security"
    fi

    echo "Environment variables validated successfully"
}

wait_for_postgres() {
    echo "Waiting for PostgreSQL at ${DATABASE_HOST}:${DATABASE_PORT}..."
    max_attempts=30
    attempt=1

    while [ $attempt -le $max_attempts ]; do
        if PGPASSWORD="$DATABASE_PASSWORD" pg_isready -h "${DATABASE_HOST}" -p "${DATABASE_PORT}" -U "${DATABASE_USER}" -d "${DATABASE_NAME}" > /dev/null 2>&1; then
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

    max_attempts=30
    attempt=1

    while [ $attempt -le $max_attempts ]; do
        if [ -n "$REDIS_PASSWORD" ]; then
            if redis-cli -h "${REDIS_HOST}" -p "${REDIS_PORT}" -a "$REDIS_PASSWORD" ping > /dev/null 2>&1; then
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

setup_system() {
    echo "Setting up system environment..."

    mkdir -p /var/log/hjtpx
    mkdir -p /tmp/hjtpx

    chmod 755 /var/log/hjtpx
    chmod 1777 /tmp/hjtpx

    echo "System environment setup completed"
}

start_application() {
    echo "Starting application..."
    echo "Server will listen on port ${SERVER_PORT:-8080}"
    echo "Gin mode: ${GIN_MODE:-release}"

    exec /server
}

main() {
    validate_env
    wait_for_postgres
    wait_for_redis
    setup_system
    start_application
}

main "$@"
