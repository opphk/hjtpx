-- CaptchaX Data Archiving Strategy - Scheduled Cleanup
-- Version: 005_archiving_strategy
-- Description: Implement scheduled data retention and cleanup policies
-- Created: 2026-05-14

BEGIN;

-- 1. Create cleanup job tracking table
CREATE TABLE IF NOT EXISTS cleanup_job_log (
    id SERIAL PRIMARY KEY,
    job_name VARCHAR(100) NOT NULL,
    job_type VARCHAR(50) NOT NULL,
    started_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP,
    status VARCHAR(20) DEFAULT 'running',
    records_processed BIGINT DEFAULT 0,
    records_deleted BIGINT DEFAULT 0,
    execution_time_ms BIGINT,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX ON cleanup_job_log (job_name);
CREATE INDEX ON cleanup_job_log (started_at);
CREATE INDEX ON cleanup_job_log (status);

-- 2. Create table size monitoring view
CREATE OR REPLACE VIEW v_table_sizes AS
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS total_size,
    pg_total_relation_size(schemaname||'.'||tablename) AS total_bytes,
    pg_size_pretty(pg_relation_size(schemaname||'.'||tablename)) AS table_size,
    pg_relation_size(schemaname||'.'||tablename) AS table_bytes,
    pg_size_pretty(pg_indexes_size(schemaname||'.'||tablename)) AS indexes_size,
    pg_indexes_size(schemaname||'.'||tablename) AS indexes_bytes,
    n_live_tup,
    n_dead_tup,
    last_vacuum,
    last_autovacuum,
    last_analyze
FROM pg_stat_user_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- 3. Create index usage statistics view
CREATE OR REPLACE VIEW v_index_usage AS
SELECT
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch,
    pg_size_pretty(pg_relation_size(indexrelid)) AS index_size
FROM pg_stat_user_indexes
JOIN pg_index USING (indexrelid)
WHERE schemaname = 'public'
ORDER BY idx_scan ASC;

-- 4. Create partition information view
CREATE OR REPLACE VIEW v_partition_info AS
SELECT
    parent.relname AS parent_table,
    child.relname AS partition_name,
    pg_get_expr(child.relpartbound, child.oid) AS partition_range,
    pg_size_pretty(pg_relation_size(child.oid)) AS partition_size,
    pg_relation_size(child.oid) AS size_bytes,
    (SELECT COUNT(*) FROM captcha_logs cl WHERE EXISTS (
        SELECT 1 FROM pg_partition_tree(child.relname) pt
        WHERE pt.relid = cl.tableoid
    )) AS estimated_rows
FROM pg_inherits
JOIN pg_class parent ON pg_inherits.inhparent = parent.oid
JOIN pg_class child ON pg_inherits.inhrelid = child.oid
WHERE parent.relname = 'captcha_logs';

-- 5. Create slow query log table
CREATE TABLE IF NOT EXISTS slow_query_log (
    id SERIAL PRIMARY KEY,
    query TEXT NOT NULL,
    calls INTEGER DEFAULT 1,
    total_time_ms NUMERIC(20,2),
    min_time_ms NUMERIC(20,2),
    max_time_ms NUMERIC(20,2),
    mean_time_ms NUMERIC(20,2),
    stddev_time_ms NUMERIC(20,2),
    last_executed_at TIMESTAMP DEFAULT NOW(),
    recorded_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX ON slow_query_log (total_time_ms DESC);
CREATE INDEX ON slow_query_log (last_executed_at);

-- 6. Create cleanup stored procedures

-- Blacklist cleanup job
CREATE OR REPLACE PROCEDURE cleanup_expired_blacklist_job()
LANGUAGE plpgsql
AS $$
DECLARE
    v_deleted INTEGER;
    v_start TIMESTAMP := NOW();
BEGIN
    DELETE FROM blacklist 
    WHERE expire_at IS NOT NULL 
    AND expire_at < NOW();
    
    GET DIAGNOSTICS v_deleted = ROW_COUNT;
    
    INSERT INTO cleanup_job_log (job_name, job_type, completed_at, status, records_deleted, execution_time_ms)
    VALUES ('cleanup_expired_blacklist', 'blacklist_cleanup', NOW(), 'completed', v_deleted, 
            EXTRACT(MILLISECONDS FROM NOW() - v_start)::BIGINT);
END;
$$;

-- Captcha logs archival job
CREATE OR REPLACE PROCEDURE archive_old_captcha_logs_job(p_archive_after_days INTEGER DEFAULT 90)
LANGUAGE plpgsql
AS $$
DECLARE
    v_archived INTEGER;
    v_start TIMESTAMP := NOW();
BEGIN
    -- Archive data older than retention period
    WITH archived AS (
        DELETE FROM captcha_logs
        WHERE created_at < NOW() - (p_archive_after_days || ' days')::INTERVAL
        AND created_at >= NOW() - ((p_archive_after_days + 30) || ' days')::INTERVAL
        LIMIT 50000
        RETURNING *
    )
    INSERT INTO captcha_logs_archive 
        (captcha_type, client_id, ip, user_agent, result, duration, risk_score, created_at, archive_reason)
    SELECT captcha_type, client_id, ip, user_agent, result, duration, risk_score, created_at, 'scheduled_cleanup'
    FROM archived;
    
    GET DIAGNOSTICS v_archived = ROW_COUNT;
    
    INSERT INTO cleanup_job_log (job_name, job_type, completed_at, status, records_processed, records_deleted, execution_time_ms)
    VALUES ('archive_old_captcha_logs', 'data_archival', NOW(), 'completed', v_archived, v_archived,
            EXTRACT(MILLISECONDS FROM NOW() - v_start)::BIGINT);
    
    -- Update vacuum settings if needed
    ANALYZE captcha_logs;
END;
$$;

-- Dead tuple cleanup job
CREATE OR REPLACE PROCEDURE vacuum_cleanup_job()
LANGUAGE plpgsql
AS $$
DECLARE
    v_start TIMESTAMP := NOW();
    v_tables TEXT[] := ARRAY['captcha_logs', 'blacklist', 'whitelist', 'captcha_config'];
    v_table TEXT;
BEGIN
    FOREACH v_table IN ARRAY v_tables
    LOOP
        EXECUTE format('VACUUM ANALYZE %I', v_table);
    END LOOP;
    
    INSERT INTO cleanup_job_log (job_name, job_type, completed_at, status, execution_time_ms)
    VALUES ('vacuum_cleanup', 'maintenance', NOW(), 'completed', 
            EXTRACT(MILLISECONDS FROM NOW() - v_start)::BIGINT);
END;
$$;

-- Stats aggregation job (materialized views refresh)
CREATE OR REPLACE PROCEDURE refresh_stats_materialized()
LANGUAGE plpgsql
AS $$
DECLARE
    v_start TIMESTAMP := NOW();
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY v_table_sizes;
    REFRESH MATERIALIZED VIEW CONCURRENTLY v_index_usage;
    
    INSERT INTO cleanup_job_log (job_name, job_type, completed_at, status, execution_time_ms)
    VALUES ('refresh_stats', 'stats_refresh', NOW(), 'completed',
            EXTRACT(MILLISECONDS FROM NOW() - v_start)::BIGINT);
END;
$$;

-- 7. Create pg_cron schedule entries (commented for reference)
-- Note: Requires pg_cron extension to be installed
-- SELECT cron.schedule('cleanup-expired-blacklist', '*/15 * * * *', 'CALL cleanup_expired_blacklist_job()');
-- SELECT cron.schedule('archive-old-logs', '0 2 * * *', 'CALL archive_old_captcha_logs_job(90)');
-- SELECT cron.schedule('vacuum-cleanup', '0 3 * * *', 'CALL vacuum_cleanup_job()');
-- SELECT cron.schedule('refresh-stats', '*/30 * * * *', 'CALL refresh_stats_materialized()');

-- 8. Create retention policy enforcement function
CREATE OR REPLACE FUNCTION enforce_retention_policy()
RETURNS void AS $$
DECLARE
    v_policy RECORD;
    v_deleted BIGINT := 0;
BEGIN
    FOR v_policy IN 
        SELECT table_name, retention_days 
        FROM archive_policy 
        WHERE is_active = true
    LOOP
        IF v_policy.table_name = 'captcha_logs' AND v_policy.retention_days IS NOT NULL THEN
            -- Delete from archive table based on retention
            EXECUTE format(
                'DELETE FROM captcha_logs_archive WHERE archived_at < NOW() - $1',
                v_policy.retention_days || ' days'
            );
            GET DIAGNOSTICS v_deleted = ROW_COUNT;
            
            RAISE NOTICE 'Deleted % old records from captcha_logs_archive', v_deleted;
        END IF;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- 9. Create comprehensive maintenance procedure
CREATE OR REPLACE PROCEDURE run_database_maintenance()
LANGUAGE plpgsql
AS $$
DECLARE
    v_start TIMESTAMP := NOW();
    v_status VARCHAR(20) := 'completed';
    v_error TEXT;
BEGIN
    BEGIN
        -- Step 1: Cleanup expired blacklist
        CALL cleanup_expired_blacklist_job();
        
        -- Step 2: Archive old captcha logs
        CALL archive_old_captcha_logs_job(90);
        
        -- Step 3: Vacuum tables
        CALL vacuum_cleanup_job();
        
        -- Step 4: Enforce retention policies
        PERFORM enforce_retention_policy();
        
    EXCEPTION WHEN OTHERS THEN
        v_status := 'failed';
        v_error := SQLERRM;
    END;
    
    INSERT INTO cleanup_job_log (job_name, job_type, completed_at, status, execution_time_ms, error_message)
    VALUES ('full_maintenance', 'comprehensive', NOW(), v_status, 
            EXTRACT(MILLISECONDS FROM NOW() - v_start)::BIGINT, v_error);
END;
$$;

-- 10. Create monitoring alerts view
CREATE OR REPLACE VIEW v_maintenance_alerts AS
SELECT
    'large_table' AS alert_type,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_stat_user_tables
WHERE schemaname = 'public'
AND pg_total_relation_size(schemaname||'.'||tablename) > 1024*1024*1024  -- > 1GB
UNION ALL
SELECT
    'high_dead_tuples',
    tablename,
    n_dead_tup::TEXT
FROM pg_stat_user_tables
WHERE schemaname = 'public'
AND n_dead_tup > 10000
UNION ALL
SELECT
    'unused_indexes',
    indexname,
    idx_scan::TEXT
FROM pg_stat_user_indexes
WHERE schemaname = 'public'
AND idx_scan = 0
AND indexname NOT LIKE '%_pkey'
AND indexname NOT LIKE '%_seq%';

-- Record migration
INSERT INTO schema_migrations (version) VALUES ('005_archiving_strategy')
ON CONFLICT (version) DO NOTHING;

COMMIT;
