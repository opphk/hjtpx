-- Rollback 005_archiving_strategy
-- Drop archiving related objects

BEGIN;

-- Remove migration record
DELETE FROM schema_migrations WHERE version = '005_archiving_strategy';

-- Drop procedures
DROP PROCEDURE IF EXISTS cleanup_expired_blacklist_job();
DROP PROCEDURE IF EXISTS archive_old_captcha_logs_job(INTEGER);
DROP PROCEDURE IF EXISTS vacuum_cleanup_job();
DROP PROCEDURE IF EXISTS refresh_stats_materialized();
DROP PROCEDURE IF EXISTS run_database_maintenance();

-- Drop functions
DROP FUNCTION IF EXISTS enforce_retention_policy();

-- Drop views
DROP VIEW IF EXISTS v_maintenance_alerts;
DROP VIEW IF EXISTS v_partition_info;
DROP VIEW IF EXISTS v_index_usage;
DROP VIEW IF EXISTS v_table_sizes;

-- Drop tables
DROP TABLE IF EXISTS slow_query_log CASCADE;
DROP TABLE IF EXISTS cleanup_job_log CASCADE;

COMMIT;
