-- Rollback migration for query optimization
BEGIN;

DROP INDEX IF EXISTS idx_sessions_user_id_created;
DROP INDEX IF EXISTS idx_sessions_expires_is_revoked;
DROP INDEX IF EXISTS idx_users_email_role;
DROP INDEX IF EXISTS idx_users_created_role;
DROP INDEX IF EXISTS idx_users_is_active_created;
DROP INDEX IF EXISTS idx_captcha_logs_user_id_created;
DROP INDEX IF EXISTS idx_captcha_logs_type_created;
DROP INDEX IF EXISTS idx_captcha_logs_verification_status;
DROP INDEX IF EXISTS idx_notifications_user_read;
DROP INDEX IF EXISTS idx_analytics_date_event;
DROP INDEX IF EXISTS idx_blacklist_ip_type;
DROP INDEX IF EXISTS idx_whitelist_ip_type;

DROP FUNCTION IF EXISTS get_slow_queries(INTEGER);
DROP FUNCTION IF EXISTS get_table_stats();
DROP FUNCTION IF EXISTS get_unused_indexes();
DROP FUNCTION IF EXISTS find_missing_indexes();
DROP FUNCTION IF EXISTS optimize_table(TEXT);
DROP FUNCTION IF EXISTS analyze_all_tables();

COMMIT;
