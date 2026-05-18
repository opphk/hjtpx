#!/bin/sh
# =============================================================================
# PostgreSQL 初始化脚本
# =============================================================================

set -e

echo "Initializing PostgreSQL database..."

# 等待PostgreSQL完全启动
until psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" -c 'SELECT 1' > /dev/null 2>&1; do
    echo "Waiting for PostgreSQL to be ready..."
    sleep 1
done

# 创建扩展
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    -- 启用UUID生成
    CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

    -- 启用性能监控
    CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";

    -- 设置优化参数
    ALTER SYSTEM SET shared_preload_libraries = 'pg_stat_statements';
    ALTER SYSTEM SET track_activities = on;
    ALTER SYSTEM SET track_counts = on;
    ALTER SYSTEM SET track_io_timing = on;
    ALTER SYSTEM SET track_wal_io_timing = on;

    -- 优化配置
    ALTER SYSTEM SET max_connections = 200;
    ALTER SYSTEM SET shared_buffers = '256MB';
    ALTER SYSTEM SET effective_cache_size = '1GB';
    ALTER SYSTEM SET maintenance_work_mem = '64MB';
    ALTER SYSTEM SET checkpoint_completion_target = 0.9;
    ALTER SYSTEM SET wal_buffers = '16MB';
    ALTER SYSTEM SET default_statistics_target = 100;
    ALTER SYSTEM SET random_page_cost = 1.1;
    ALTER SYSTEM SET effective_io_concurrency = 200;
    ALTER SYSTEM SET work_mem = '4MB';
    ALTER SYSTEM SET min_wal_size = '1GB';
    ALTER SYSTEM SET max_wal_size = '4GB';
EOSQL

echo "PostgreSQL initialization completed successfully!"
