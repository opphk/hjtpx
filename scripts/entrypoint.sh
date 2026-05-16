#!/bin/sh
set -e

echo "Starting hjtpx application..."

# 等待数据库就绪
echo "Waiting for PostgreSQL..."
until pg_isready -h ${POSTGRES_HOST:-postgres} -p ${POSTGRES_PORT:-5432} -U ${POSTGRES_USER:-postgres} > /dev/null 2>&1; do
    echo "PostgreSQL is unavailable - sleeping"
    sleep 2
done
echo "PostgreSQL is up"

# 等待Redis就绪
echo "Waiting for Redis..."
until redis-cli -h ${REDIS_HOST:-redis} -p ${REDIS_PORT:-6379} -a "${REDIS_PASSWORD:-}" ping > /dev/null 2>&1; do
    echo "Redis is unavailable - sleeping"
    sleep 2
done
echo "Redis is up"

# 创建日志目录
mkdir -p /var/log/hjtpx

# 启动应用
echo "Starting application..."
exec /app/hjtpx
