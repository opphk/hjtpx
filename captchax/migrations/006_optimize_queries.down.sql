-- Rollback 006_optimize_queries
-- Drop query optimization objects

BEGIN;

-- Remove migration record
DELETE FROM schema_migrations WHERE version = '006_optimize_queries';

-- Drop functions
DROP FUNCTION IF EXISTS analyze_query_plan(TEXT);
DROP FUNCTION IF EXISTS check_missing_indexes(TEXT);
DROP FUNCTION IF EXISTS refresh_captcha_stats();
DROP FUNCTION IF EXISTS get_cached_result(VARCHAR);
DROP FUNCTION IF EXISTS get_hourly_trend(DATE, DATE);

-- Drop materialized views
DROP MATERIALIZED VIEW IF EXISTS mv_client_stats;
DROP MATERIALIZED VIEW IF EXISTS mv_ip_stats;
DROP MATERIALIZED VIEW IF EXISTS mv_captcha_daily_stats;

-- Drop views
DROP VIEW IF EXISTS v_table_health;
DROP VIEW IF EXISTS v_slow_queries;

-- Drop tables
DROP TABLE IF EXISTS query_cache CASCADE;

-- Deallocate prepared statements
DEALLOCATE PREPARE IF EXISTS get_recent_logs;
DEALLOCATE PREPARE IF EXISTS count_by_ip_window;
DEALLOCATE PREPARE IF EXISTS stats_by_type;

COMMIT;
