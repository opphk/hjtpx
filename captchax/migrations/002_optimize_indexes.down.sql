-- Rollback 002_optimize_indexes
-- Drop all indexes and objects created by this migration

BEGIN;

-- Remove migration record
DELETE FROM schema_migrations WHERE version = '002_optimize_indexes';

-- Drop indexes created by this migration
DROP INDEX IF EXISTS idx_captcha_logs_type_created;
DROP INDEX IF EXISTS idx_captcha_logs_client_created;
DROP INDEX IF EXISTS idx_captcha_logs_ip_created_result;
DROP INDEX IF EXISTS idx_captcha_logs_risk_created;
DROP INDEX IF EXISTS idx_captcha_logs_duration_created;
DROP INDEX IF EXISTS idx_blacklist_active;
DROP INDEX IF EXISTS idx_whitelist_active;
DROP INDEX IF EXISTS idx_captcha_logs_ip_text_pattern;
DROP INDEX IF EXISTS idx_admins_username_lower;
DROP INDEX IF EXISTS idx_captcha_logs_covering_list;
DROP INDEX IF EXISTS idx_captcha_logs_covering_stats;

-- Analyze tables after dropping indexes
ANALYZE captcha_logs;
ANALYZE blacklist;
ANALYZE whitelist;
ANALYZE admins;

COMMIT;
