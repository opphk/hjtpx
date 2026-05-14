-- CaptchaX Database Monitoring and Metrics
-- Version: 008_monitoring_metrics
-- Description: Add monitoring views, functions and metrics tables
-- Created: 2026-05-14

BEGIN;

-- 1. Create database metrics history table
CREATE TABLE IF NOT EXISTS db_metrics_history (
    id SERIAL PRIMARY KEY,
    metric_name VARCHAR(100) NOT NULL,
    metric_value NUMERIC NOT NULL,
    metric_unit VARCHAR(20),
    tags JSONB DEFAULT '{}',
    recorded_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX ON db_metrics_history (metric_name, recorded_at);
CREATE INDEX ON db_metrics_history (recorded_at DESC);

-- 2. Create function to collect database metrics
CREATE OR REPLACE FUNCTION collect_db_metrics()
RETURNS void AS $$
DECLARE
    v_db_size BIGINT;
    v_table_count BIGINT;
    v_index_count BIGINT;
    v_connections INTEGER;
    v_tx_count BIGINT;
    v_cache_hit REAL;
BEGIN
    SELECT pg_database_size(current_database()) INTO v_db_size;
    INSERT INTO db_metrics_history (metric_name, metric_value, metric_unit, tags)
    VALUES ('database_size_bytes', v_db_size, 'bytes', jsonb_build_object('database', current_database()));

    SELECT COUNT(*) INTO v_table_count FROM pg_tables WHERE schemaname = 'public';
    INSERT INTO db_metrics_history (metric_name, metric_value, metric_unit, tags)
    VALUES ('table_count', v_table_count, 'count', '{}');

    SELECT COUNT(*) INTO v_index_count FROM pg_indexes WHERE schemaname = 'public';
    INSERT INTO db_metrics_history (metric_name, metric_value, metric_unit, tags)
    VALUES ('index_count', v_index_count, 'count', '{}');

    SELECT COUNT(*) INTO v_connections FROM pg_stat_activity WHERE datname = current_database();
    INSERT INTO db_metrics_history (metric_name, metric_value, metric_unit, tags)
    VALUES ('active_connections', v_connections, 'count', '{}');

    SELECT sum(xact_commit + xact_rollback) INTO v_tx_count FROM pg_stat_database WHERE datname = current_database();
    INSERT INTO db_metrics_history (metric_name, metric_value, metric_unit, tags)
    VALUES ('transaction_count', COALESCE(v_tx_count, 0), 'count', '{}');

    SELECT COALESCE(sum(blks_hit) * 100.0 / NULLIF(sum(blks_hit) + sum(blks_read), 0), 0) 
    INTO v_cache_hit FROM pg_stat_database WHERE datname = current_database();
    INSERT INTO db_metrics_history (metric_name, metric_value, metric_unit, tags)
    VALUES ('cache_hit_ratio', v_cache_hit, 'percent', '{}');
END;
$$ LANGUAGE plpgsql;

-- 3. Create table size tracking function
CREATE OR REPLACE FUNCTION get_detailed_table_sizes()
RETURNS TABLE(
    table_name TEXT,
    row_count BIGINT,
    total_bytes BIGINT,
    table_bytes BIGINT,
    index_bytes BIGINT,
    toast_bytes BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        c.relname::TEXT AS table_name,
        COALESCE(c.reltuples, 0)::BIGINT AS row_count,
        COALESCE(pg_total_relation_size(c.oid), 0)::BIGINT AS total_bytes,
        COALESCE(pg_relation_size(c.oid), 0)::BIGINT AS table_bytes,
        COALESCE(pg_indexes_size(c.oid), 0)::BIGINT AS index_bytes,
        COALESCE(pg_total_relation_size(c.oid) - pg_relation_size(c.oid) - COALESCE(pg_indexes_size(c.oid), 0), 0)::BIGINT AS toast_bytes
    FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE c.relkind = 'r'
    AND n.nspname = 'public'
    ORDER BY pg_total_relation_size(c.oid) DESC;
END;
$$ LANGUAGE plpgsql;

-- 4. Create connection pool monitoring view
CREATE OR REPLACE VIEW v_connection_pool_metrics AS
SELECT
    application_name,
    usename,
    state,
    COUNT(*) AS connection_count,
    AVG(EXTRACT(EPOCH FROM (now() - state_change))) AS avg_connection_age_seconds,
    COUNT(*) FILTER (WHERE wait_event IS NOT NULL) AS waiting_count,
    MAX(EXTRACT(EPOCH FROM (now() - query_start))) AS max_query_duration_seconds
FROM pg_stat_activity
WHERE datname = current_database()
GROUP BY application_name, usename, state;

-- 5. Create index effectiveness view
CREATE OR REPLACE VIEW v_index_effectiveness AS
SELECT
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch,
    pg_size_pretty(pg_relation_size(indexrelid)) AS index_size,
    CASE 
        WHEN idx_scan = 0 THEN 'unused'
        WHEN idx_scan < 100 THEN 'low_usage'
        WHEN idx_scan < 1000 THEN 'medium_usage'
        ELSE 'high_usage'
    END AS usage_category
FROM pg_stat_user_indexes
JOIN pg_index USING (indexrelid)
WHERE schemaname = 'public'
ORDER BY idx_scan ASC;

-- 6. Create query performance tracking view
CREATE OR REPLACE VIEW v_query_performance AS
SELECT
    LEFT(query, 100) AS query_prefix,
    calls,
    total_exec_time / 1000 AS total_seconds,
    mean_exec_time AS avg_ms,
    max_exec_time AS max_ms,
    min_exec_time AS min_ms,
    rows / calls AS avg_rows,
    stddev_exec_time AS stddev_ms
FROM pg_stat_statements
WHERE calls > 0
ORDER BY total_exec_time DESC
LIMIT 50;

-- 7. Create replication lag monitoring view
CREATE OR REPLACE VIEW v_replication_metrics AS
SELECT
    application_name,
    client_addr,
    state,
    sent_lsn::TEXT,
    write_lsn::TEXT,
    flush_lsn::TEXT,
    replay_lsn::TEXT,
    write_lag,
    flush_lag,
    replay_lag,
    sync_state
FROM pg_stat_replication;

-- 8. Create vacuum and analyze tracking view
CREATE OR REPLACE VIEW v_maintenance_history AS
SELECT
    schemaname,
    relname AS table_name,
    last_vacuum,
    last_autovacuum,
    last_analyze,
    last_autoanalyze,
    vacuum_count,
    autovacuum_count,
    analyze_count,
    autoanalyze_count,
    n_live_tup,
    n_dead_tup,
    n_mod_since_analyze
FROM pg_stat_user_tables
WHERE schemaname = 'public'
ORDER BY n_dead_tup DESC;

-- 9. Create blocking query detection view
CREATE OR REPLACE VIEW v_blocking_queries AS
SELECT
    blocked_locks.pid AS blocked_pid,
    blocked_activity.usename AS blocked_user,
    blocking_locks.pid AS blocking_pid,
    blocking_activity.usename AS blocking_user,
    blocked_activity.query AS blocked_statement,
    blocking_activity.query AS blocking_statement,
    blocked_activity.application_name AS blocked_app,
    blocking_activity.application_name AS blocking_app
FROM pg_catalog.pg_locks blocked_locks
JOIN pg_catalog.pg_stat_activity blocked_activity ON blocked_activity.pid = blocked_locks.pid
JOIN pg_catalog.pg_locks blocking_locks 
    ON blocking_locks.locktype = blocked_locks.locktype
    AND blocking_locks.database IS NOT DISTINCT FROM blocked_locks.database
    AND blocking_locks.relation IS NOT DISTINCT FROM blocked_locks.relation
    AND blocking_locks.page IS NOT DISTINCT FROM blocked_locks.page
    AND blocking_locks.tuple IS NOT DISTINCT FROM blocked_locks.tuple
    AND blocking_locks.virtualxid IS NOT DISTINCT FROM blocked_locks.virtualxid
    AND blocking_locks.transactionid IS NOT DISTINCT FROM blocked_locks.transactionid
    AND blocking_locks.classid IS NOT DISTINCT FROM blocked_locks.classid
    AND blocking_locks.objid IS NOT DISTINCT FROM blocked_locks.objid
    AND blocking_locks.objsubid IS NOT DISTINCT FROM blocked_locks.objsubid
    AND blocking_locks.pid != blocked_locks.pid
JOIN pg_catalog.pg_stat_activity blocking_activity ON blocking_activity.pid = blocking_locks.pid
WHERE NOT blocked_locks.granted;

-- 10. Create long-running query detection view
CREATE OR REPLACE VIEW v_long_running_queries AS
SELECT
    pid,
    usename,
    application_name,
    state,
    query,
    EXTRACT(EPOCH FROM (now() - query_start)) AS duration_seconds,
    wait_event_type,
    wait_event
FROM pg_stat_activity
WHERE state != 'idle'
AND query_start < NOW() - INTERVAL '1 minute'
ORDER BY duration_seconds DESC;

-- 11. Create alerts view for monitoring
CREATE OR REPLACE VIEW v_monitoring_alerts AS
SELECT 'high_connection_count' AS alert_type, 
       COUNT(*)::TEXT AS value,
       'Connections exceeding threshold' AS message
FROM pg_stat_activity
WHERE datname = current_database()
HAVING COUNT(*) > 100

UNION ALL

SELECT 'high_replication_lag' AS alert_type,
       replay_lag::TEXT,
       'Replica lag exceeds threshold' AS message
FROM pg_stat_replication
WHERE replay_lag IS NOT NULL
AND replay_lag > INTERVAL '30 seconds'

UNION ALL

SELECT 'unused_indexes' AS alert_type,
       COUNT(*)::TEXT,
       'Indexes not being used' AS message
FROM pg_stat_user_indexes
WHERE idx_scan = 0
AND schemaname = 'public'

UNION ALL

SELECT 'high_dead_tuples' AS alert_type,
       SUM(n_dead_tup)::TEXT,
       'Tables need vacuum' AS message
FROM pg_stat_user_tables
WHERE schemaname = 'public'
GROUP BY schemaname
HAVING SUM(n_dead_tup) > 10000;

-- 12. Create function to get comprehensive health status
CREATE OR REPLACE FUNCTION get_db_health_status()
RETURNS TABLE(
    check_name TEXT,
    status VARCHAR(20),
    details JSONB
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        'connections'::TEXT AS check_name,
        CASE 
            WHEN COUNT(*) < 100 THEN 'healthy'
            WHEN COUNT(*) < 150 THEN 'warning'
            ELSE 'critical'
        END AS status,
        jsonb_build_object('current'::TEXT, COUNT(*), 'max'::TEXT, 100) AS details
    FROM pg_stat_activity WHERE datname = current_database();

    RETURN QUERY
    SELECT 
        'cache_hit_ratio'::TEXT AS check_name,
        CASE 
            WHEN COALESCE(sum(blks_hit) * 100.0 / NULLIF(sum(blks_hit) + sum(blks_read), 0), 0) > 95 THEN 'healthy'
            WHEN COALESCE(sum(blks_hit) * 100.0 / NULLIF(sum(blks_hit) + sum(blks_read), 0), 0) > 90 THEN 'warning'
            ELSE 'critical'
        END AS status,
        jsonb_build_object('ratio'::TEXT, ROUND(COALESCE(sum(blks_hit) * 100.0 / NULLIF(sum(blks_hit) + sum(blks_read), 0), 0)::NUMERIC, 2)) AS details
    FROM pg_stat_database WHERE datname = current_database();

    RETURN QUERY
    SELECT 
        'replication'::TEXT AS check_name,
        CASE 
            WHEN NOT EXISTS (SELECT 1 FROM pg_stat_replication) THEN 'not_configured'
            WHEN EXISTS (SELECT 1 FROM pg_stat_replication WHERE replay_lag > INTERVAL '30 seconds') THEN 'critical'
            ELSE 'healthy'
        END AS status,
        '{}'::JSONB AS details;

    RETURN QUERY
    SELECT 
        'disk_space'::TEXT AS check_name,
        CASE 
            WHEN pg_database_size(current_database()) < 1024*1024*1024 THEN 'healthy'
            WHEN pg_database_size(current_database()) < 5*1024*1024*1024 THEN 'warning'
            ELSE 'critical'
        END AS status,
        jsonb_build_object('size'::TEXT, pg_size_pretty(pg_database_size(current_database()))) AS details;
END;
$$ LANGUAGE plpgsql;

-- 13. Create metrics aggregation function
CREATE OR REPLACE FUNCTION aggregate_metrics(
    p_metric_name VARCHAR,
    p_start_time TIMESTAMP,
    p_end_time TIMESTAMP,
    p_interval VARCHAR DEFAULT 'hour'
)
RETURNS TABLE(
    bucket TIMESTAMP,
    sample_count BIGINT,
    min_value NUMERIC,
    max_value NUMERIC,
    avg_value NUMERIC,
    percentile_95 NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        DATE_TRUNC(p_interval, recorded_at) AS bucket,
        COUNT(*) AS sample_count,
        MIN(metric_value) AS min_value,
        MAX(metric_value) AS max_value,
        AVG(metric_value) AS avg_value,
        PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY metric_value) AS percentile_95
    FROM db_metrics_history
    WHERE metric_name = p_metric_name
    AND recorded_at >= p_start_time
    AND recorded_at <= p_end_time
    GROUP BY DATE_TRUNC(p_interval, recorded_at)
    ORDER BY bucket;
END;
$$ LANGUAGE plpgsql;

-- Record migration
INSERT INTO schema_migrations (version) VALUES ('008_monitoring_metrics')
ON CONFLICT (version) DO NOTHING;

COMMIT;
