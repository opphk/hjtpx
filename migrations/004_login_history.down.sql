-- Rollback: Login History and Session Management
-- Description: Removes login history and login attempt tracking

-- Drop indexes
DROP INDEX IF EXISTS idx_login_history_user_id;
DROP INDEX IF EXISTS idx_login_history_created_at;
DROP INDEX IF EXISTS idx_login_history_action;
DROP INDEX IF EXISTS idx_login_history_success;
DROP INDEX IF EXISTS idx_login_history_ip;
DROP INDEX IF EXISTS idx_login_attempts_email;
DROP INDEX IF EXISTS idx_login_attempts_ip;
DROP INDEX IF EXISTS idx_login_attempts_time;
DROP INDEX IF EXISTS idx_login_attempts_success;

-- Drop tables
DROP TABLE IF EXISTS login_attempts;
DROP TABLE IF EXISTS login_history;
