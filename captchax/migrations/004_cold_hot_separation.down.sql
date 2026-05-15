-- Rollback 004_cold_hot_separation
-- Drop archive tables and related objects

BEGIN;

-- Remove migration record
DELETE FROM schema_migrations WHERE version = '004_cold_hot_separation';

-- Drop functions
DROP FUNCTION IF EXISTS archive_captcha_logs(TIMESTAMP, INTEGER, BOOLEAN);
DROP FUNCTION IF EXISTS restore_archived_logs(TIMESTAMP, TIMESTAMP, INTEGER);
DROP FUNCTION IF EXISTS get_archive_stats();
DROP FUNCTION IF EXISTS purge_archived_data(INTEGER);
DROP FUNCTION IF EXISTS check_archive_threshold();

-- Drop tables
DROP TABLE IF EXISTS archive_policy CASCADE;
DROP TABLE IF EXISTS archive_metadata CASCADE;
DROP TABLE IF EXISTS captcha_logs_archive CASCADE;
DROP TABLE IF EXISTS captcha_logs_archive_2025 CASCADE;
DROP TABLE IF EXISTS captcha_logs_archive_older CASCADE;

COMMIT;
