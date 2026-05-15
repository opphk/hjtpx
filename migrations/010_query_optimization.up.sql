-- Migration: Database Query Optimization and Index Review
-- Created: 2026-05-15
-- Purpose: Optimize slow queries and add missing indexes

BEGIN;

CREATE INDEX IF NOT EXISTS idx_sessions_user_id_created 
ON sessions(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_sessions_expires_is_revoked 
ON sessions(expires_at, is_revoked) 
WHERE is_revoked = false;

CREATE INDEX IF NOT EXISTS idx_users_email_role 
ON users(email, role) 
WHERE is_active = true;

CREATE INDEX IF NOT EXISTS idx_users_created_role 
ON users(created_at DESC, role);

CREATE INDEX IF NOT EXISTS idx_users_is_active_created 
ON users(is_active, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_captcha_logs_user_id_created 
ON captcha_logs(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_captcha_logs_type_created 
ON captcha_logs(captcha_type, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_captcha_logs_verification_status 
ON captcha_logs(verification_status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_notifications_user_read 
ON notifications(user_id, is_read, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_analytics_date_event 
ON analytics(date, event_type);

CREATE INDEX IF NOT EXISTS idx_blacklist_ip_type 
ON blacklist(ip_address, type);

CREATE INDEX IF NOT EXISTS idx_whitelist_ip_type 
ON whitelist(ip_address, type);

CREATE OR REPLACE FUNCTION get_slow_queries(threshold_ms INTEGER DEFAULT 100)
RETURNS TABLE(
    query TEXT,
    calls BIGINT,
    mean_time FLOAT,
    total_time FLOAT,
    min_time FLOAT,
    max_time FLOAT,
    rows BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        s.query,
        s.calls,
        s.mean_exec_time,
        s.total_exec_time,
        s.min_exec_time,
        s.max_exec_time,
        s.rows
    FROM pg_stat_statements s
    WHERE s.mean_exec_time > threshold_ms
    ORDER BY s.mean_exec_time DESC
    LIMIT 50;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION get_table_stats()
RETURNS TABLE(
    table_name TEXT,
    row_count BIGINT,
    total_size TEXT,
    index_size TEXT,
    seq_scans BIGINT,
    idx_scans BIGINT,
    last_vacuum TIMESTAMPTZ,
    last_autovacuum TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        t.relname::TEXT,
        t.n_live_tup::BIGINT,
        pg_size_pretty(pg_total_relation_size(t.relid))::TEXT,
        pg_size_pretty(pg_indexes_size(t.relid))::TEXT,
        t.seq_scan::BIGINT,
        t.idx_scan::BIGINT,
        t.last_vacuum,
        t.last_autovacuum
    FROM pg_stat_user_tables t
    WHERE t.n_live_tup > 0
    ORDER BY t.n_live_tup DESC;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION get_unused_indexes()
RETURNS TABLE(
    schema_name TEXT,
    table_name TEXT,
    index_name TEXT,
    index_size TEXT,
    scans BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        s.nspname::TEXT,
        t.relname::TEXT,
        i.relname::TEXT,
        pg_size_pretty(pg_relation_size(i.relid))::TEXT,
        s.idx_scan::BIGINT
    FROM pg_stat_user_indexes s
    JOIN pg_index idx ON s.indexrelid = idx.indexrelid
    JOIN pg_class i ON i.oid = s.indexrelid
    JOIN pg_class t ON t.oid = s.relid
    WHERE NOT idx.indisprimary
    AND s.idx_scan = 0
    ORDER BY pg_relation_size(i.relid) DESC;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION find_missing_indexes()
RETURNS TABLE(
    schema_name TEXT,
    table_name TEXT,
    seq_scans BIGINT,
    idx_scans BIGINT,
    seq_scan_ratio FLOAT,
    table_size TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        s.schemaname::TEXT,
        s.relname::TEXT,
        s.seq_scan::BIGINT,
        s.idx_scan::BIGINT,
        CASE 
            WHEN s.idx_scan = 0 THEN s.seq_scan::FLOAT
            ELSE s.seq_scan::FLOAT / s.idx_scan::FLOAT
        END AS seq_scan_ratio,
        pg_size_pretty(pg_relation_size(s.relid))::TEXT
    FROM pg_stat_user_tables s
    WHERE s.seq_scan > s.idx_scan * 5
    AND pg_relation_size(s.relid) > 1024 * 1024
    ORDER BY s.seq_scan DESC;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION optimize_table(table_name TEXT)
RETURNS void AS $$
BEGIN
    EXECUTE format('VACUUM ANALYZE %I', table_name);
    RAISE NOTICE 'Optimized table: %', table_name;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION analyze_all_tables()
RETURNS TABLE(
    table_name TEXT,
    last_analyzed TIMESTAMPTZ,
    row_count BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        s.relname::TEXT,
        s.last_analyze,
        s.n_live_tup::BIGINT
    FROM pg_stat_user_tables s
    WHERE s.last_analyze IS NULL
    OR s.last_analyze < NOW() - INTERVAL '7 days'
    ORDER BY s.n_live_tup DESC;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION get_slow_queries IS 'Get queries with execution time above threshold';
COMMENT ON FUNCTION get_table_stats IS 'Get statistics for all user tables';
COMMENT ON FUNCTION get_unused_indexes IS 'Find indexes that have never been scanned';
COMMENT ON FUNCTION find_missing_indexes IS 'Find tables with high sequential scan ratios';
COMMENT ON FUNCTION optimize_table IS 'VACUUM and ANALYZE a specific table';
COMMENT ON FUNCTION analyze_all_tables IS 'Find tables that need analysis';

COMMIT;
