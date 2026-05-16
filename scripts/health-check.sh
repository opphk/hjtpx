#!/bin/sh
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

APP_URL="${APP_URL:-http://localhost:8080}"
APP_HOST="${APP_HOST:-app}"
POSTGRES_HOST="${POSTGRES_HOST:-postgres}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_USER="${POSTGRES_USER:-postgres}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-postgres}"
POSTGRES_DB="${POSTGRES_DB:-hjtpx_db}"
REDIS_HOST="${REDIS_HOST:-redis}"
REDIS_PORT="${REDIS_PORT:-6379}"
REDIS_PASSWORD="${REDIS_PASSWORD:-}"

check_backend() {
    echo "检查后端服务..."
    if curl -sf "$APP_URL/health" > /dev/null 2>&1; then
        echo "✓ 后端服务正常"
        return 0
    else
        echo "✗ 后端服务异常"
        return 1
    fi
}

check_database() {
    echo "检查数据库连接..."
    if command -v psql > /dev/null 2>&1; then
        PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT 1" > /dev/null 2>&1
        if [ $? -eq 0 ]; then
            echo "✓ 数据库连接正常"
            return 0
        fi
    fi
    echo "✗ 数据库连接失败"
    return 1
}

check_redis() {
    echo "检查Redis连接..."
    if command -v redis-cli > /dev/null 2>&1; then
        if [ -n "$REDIS_PASSWORD" ]; then
            redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" -a "$REDIS_PASSWORD" ping > /dev/null 2>&1
        else
            redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" ping > /dev/null 2>&1
        fi
        if [ $? -eq 0 ]; then
            echo "✓ Redis连接正常"
            return 0
        fi
    fi
    echo "✗ Redis连接失败"
    return 1
}

main() {
    echo "===== HJTPX 健康检查 ====="
    echo ""

    total=0
    passed=0

    total=$((total + 1))
    if check_backend; then
        passed=$((passed + 1))
    fi

    total=$((total + 1))
    if check_database; then
        passed=$((passed + 1))
    fi

    total=$((total + 1))
    if check_redis; then
        passed=$((passed + 1))
    fi

    echo ""
    echo "===== 检查结果: $passed/$total 通过 ====="

    if [ $passed -eq $total ]; then
        echo "所有健康检查通过 ✓"
        exit 0
    else
        echo "部分健康检查失败 ✗"
        exit 1
    fi
}

main "$@"
