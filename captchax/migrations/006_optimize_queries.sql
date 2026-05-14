-- CaptchaX Query Optimization - Avoid Full Table Scans
-- Version: 006_optimize_queries
-- Description: Add hints, constraints, and query rewriting for optimal performance
-- Created: 2026-05-14

BEGIN;

-- 1. Create statistics and monitoring functions

-- Function to analyze query execution plan
CREATE OR REPLACE FUNCTION analyze_query_plan(p_query TEXT)
RETURNS TABLE(plan_output TEXT) AS $$
BEGIN
    RETURN QUERY SELECT EXPLAIN ANALYZE p_query;
END;
$$ LANGUAGE plpgsql;

-- Function to check for missing indexes on a table
CREATE OR REPLACE FUNCTION check_missing_indexes(p_table_name TEXT)
RETURNS TABLE(
    seq_scan BIGINT,
    idx_scan BIGINT,
    suggested_index TEXT
) AS $$
DECLARE
    v_seq_scan BIGINT;
    v_idx_scan BIGINT;
BEGIN
    SELECT seq_scan, idx_scan INTO v_seq_scan, v_idx_scan
    FROM pg_stat_user_tables
    WHERE relname = p_table_name;
    
    IF v_seq_scan > v_idx_scan * 10 AND v_seq_scan > 100 THEN
        RETURN QUERY SELECT 
            v_seq_scan,
            v_idx_scan,
            'CREATE INDEX CONCURRENTLY ON ' || p_table_name || ' (created_at);'::TEXT;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- 2. Create query templates with hints (for reference)

-- Template 1: Captcha log listing with time filter
-- This query is optimized for the common pattern:
-- SELECT * FROM captcha_logs WHERE created_at >= ? AND created_at <= ? ORDER BY created_at DESC

-- Template 2: IP-based frequency analysis
-- Optimized query for counting attempts per IP:
-- Uses the composite index idx_captcha_logs_ip_created_result

-- Template 3: Risk score threshold queries
-- Uses partial index on high risk scores

-- 3. Create prepared statements for frequently used queries

-- Get recent captcha logs (hot data query)
PREPARE get_recent_logs(TIMESTAMP, INTEGER) AS
SELECT 
    id, captcha_type, client_id, ip, result, risk_score, created_at
FROM captcha_logs
WHERE created_at >= $1
ORDER BY created_at DESC
LIMIT $2;

-- Count attempts by IP in time window
PREPARE count_by_ip_window(VARCHAR, TIMESTAMP, TIMESTAMP) AS
SELECT COUNT(*) 
FROM captcha_logs 
WHERE ip = $1 AND created_at >= $2 AND created_at <= $3;

-- Get stats by type in time range
PREPARE stats_by_type(TIMESTAMP, TIMESTAMP) AS
SELECT 
    captcha_type,
    COUNT(*) as total,
    COUNT(*) FILTER (WHERE result = true) as success,
    AVG(duration) as avg_duration
FROM captcha_logs
WHERE created_at >= $1 AND created_at <= $2
GROUP BY captcha_type;

-- 4. Create materialized view for frequently accessed aggregated data
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_captcha_daily_stats AS
SELECT 
    DATE(created_at) AS stat_date,
    captcha_type,
    COUNT(*) AS total_count,
    COUNT(*) FILTER (WHERE result = true) AS success_count,
    COUNT(*) FILTER (WHERE result = false) AS fail_count,
    AVG(duration) AS avg_duration,
    AVG(risk_score) AS avg_risk_score,
    PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY duration) AS p50_duration,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration) AS p95_duration,
    PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY duration) AS p99_duration
FROM captcha_logs
GROUP BY DATE(created_at), captcha_type
WITH DATA;

CREATE UNIQUE INDEX ON mv_captcha_daily_stats (stat_date, captcha_type);

-- 5. Create materialized view for IP analysis
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_ip_stats AS
SELECT 
    ip,
    COUNT(*) AS total_attempts,
    COUNT(*) FILTER (WHERE result = true) AS success_count,
    COUNT(*) FILTER (WHERE result = false) AS fail_count,
    AVG(risk_score) AS avg_risk_score,
    MAX(risk_score) AS max_risk_score,
    MIN(created_at) AS first_seen,
    MAX(created_at) AS last_seen,
    COUNT(DISTINCT client_id) AS unique_clients
FROM captcha_logs
WHERE created_at >= NOW() - INTERVAL '30 days'
GROUP BY ip
HAVING COUNT(*) > 10
WITH DATA;

CREATE INDEX ON mv_ip_stats (total_attempts DESC);
CREATE INDEX ON mv_ip_stats (avg_risk_score DESC);

-- 6. Create materialized view for client analysis
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_client_stats AS
SELECT 
    client_id,
    COUNT(*) AS total_attempts,
    COUNT(DISTINCT ip) AS unique_ips,
    COUNT(*) FILTER (WHERE result = true) AS success_count,
    AVG(risk_score) AS avg_risk_score
FROM captcha_logs
WHERE created_at >= NOW() - INTERVAL '7 days'
GROUP BY client_id
HAVING COUNT(*) > 5
WITH DATA;

CREATE INDEX ON mv_client_stats (total_attempts DESC);

-- 7. Create function to refresh materialized views
CREATE OR REPLACE FUNCTION refresh_captcha_stats()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_captcha_daily_stats;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_ip_stats;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_client_stats;
END;
$$ LANGUAGE plpgsql;

-- 8. Create optimization helper views

-- View to identify slow queries from pg_stat_statements
CREATE OR REPLACE VIEW v_slow_queries AS
SELECT 
    LEFT(query, 200) AS query_prefix,
    calls,
    total_exec_time / 1000 AS total_seconds,
    mean_exec_time AS avg_ms,
    max_exec_time AS max_ms,
    rows / calls AS avg_rows
FROM pg_stat_statements
WHERE calls > 10
ORDER BY total_exec_time DESC
LIMIT 20;

-- View to identify tables with potential issues
CREATE OR REPLACE VIEW v_table_health AS
SELECT
    schemaname,
    relname AS table_name,
    n_live_tup,
    n_dead_tup,
    n_mod_since_analyze,
    last_vacuum,
    last_autovacuum,
    last_analyze,
    vacuum_count,
    autovacuum_count,
    CASE 
        WHEN n_live_tup > 0 THEN ROUND(n_dead_tup::NUMERIC / n_live_tup * 100, 2)
        ELSE 0
    END AS dead_tuple_ratio_pct
FROM pg_stat_user_tables
WHERE schemaname = 'public'
ORDER BY n_dead_tup DESC;

-- 9. Create query result cache table (for expensive analytics)
CREATE TABLE IF NOT EXISTS query_cache (
    cache_key VARCHAR(255) PRIMARY KEY,
    query_hash VARCHAR(64) NOT NULL,
    result_json JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP,
    hit_count INTEGER DEFAULT 0,
    last_hit_at TIMESTAMP
);

CREATE INDEX ON query_cache (expires_at);
CREATE INDEX ON query_cache (created_at);

-- Function to get cached query results
CREATE OR REPLACE FUNCTION get_cached_result(p_cache_key VARCHAR)
RETURNS JSONB AS $$
DECLARE
    v_result JSONB;
BEGIN
    UPDATE query_cache
    SET hit_count = hit_count + 1, last_hit_at = NOW()
    WHERE cache_key = p_cache_key
    AND (expires_at IS NULL OR expires_at > NOW())
    RETURNING result_json INTO v_result;
    
    RETURN v_result;
END;
$$ LANGUAGE plpgsql;

-- 10. Create common table expressions (CTE) optimized query examples

-- Example 1: Hourly statistics with trend
CREATE OR REPLACE FUNCTION get_hourly_trend(
    p_start_date DATE,
    p_end_date DATE
)
RETURNS TABLE(
    hour TIMESTAMP,
    total BIGINT,
    success BIGINT,
    fail BIGINT,
    trend NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    WITH hourly_stats AS (
        SELECT 
            DATE_TRUNC('hour', created_at) AS hour,
            COUNT(*) AS total,
            COUNT(*) FILTER (WHERE result = true) AS success,
            COUNT(*) FILTER (WHERE result = false) AS fail
        FROM captcha_logs
        WHERE created_at >= p_start_date AND created_at < p_end_date + INTERVAL '1 day'
        GROUP BY DATE_TRUNC('hour', created_at)
    )
    SELECT 
        h.hour,
        h.total,
        h.success,
        h.fail,
        COALESCE(
            (h.total::NUMERIC - LAG(h.total) OVER (ORDER BY h.hour)) / NULLIF(LAG(h.total) OVER (ORDER BY h.hour), 0) * 100,
            0
        ) AS trend
    FROM hourly_stats h
    ORDER BY h.hour;
END;
$$ LANGUAGE plpgsql;

-- Record migration
INSERT INTO schema_migrations (version) VALUES ('006_optimize_queries')
ON CONFLICT (version) DO NOTHING;

COMMIT;
