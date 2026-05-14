-- Rollback for: 005_create_sessions.sql
-- Generated: 2026-05-14
-- This script reverts the sessions and login history enhancements

BEGIN;

-- Drop login_attempts table
DROP TABLE IF EXISTS login_attempts CASCADE;

-- Drop function to cleanup expired sessions
DROP FUNCTION IF EXISTS cleanup_expired_sessions();

-- Drop trigger for sessions table
DROP TRIGGER IF EXISTS update_sessions_last_activity ON sessions;

-- Drop function to update last_activity
DROP FUNCTION IF EXISTS update_last_activity();

-- Drop indexes for sessions
DROP INDEX IF EXISTS idx_sessions_is_current;
DROP INDEX IF EXISTS idx_sessions_last_activity;
DROP INDEX IF EXISTS idx_sessions_is_revoked;

-- Remove columns from sessions table
ALTER TABLE sessions DROP COLUMN IF EXISTS is_current;
ALTER TABLE sessions DROP COLUMN IF EXISTS last_activity;
ALTER TABLE sessions DROP COLUMN IF EXISTS is_revoked;
ALTER TABLE sessions DROP COLUMN IF EXISTS user_agent;
ALTER TABLE sessions DROP COLUMN IF EXISTS ip_address;
ALTER TABLE sessions DROP COLUMN IF EXISTS device_info;

-- Drop login_history indexes
DROP INDEX IF EXISTS idx_login_history_success;
DROP INDEX IF EXISTS idx_login_history_action;
DROP INDEX IF EXISTS idx_login_history_created_at;
DROP INDEX IF EXISTS idx_login_history_user_id;

-- Drop login_history table
DROP TABLE IF EXISTS login_history CASCADE;

-- Remove migration tracking entry
DELETE FROM migrations WHERE name = '005_create_sessions';

COMMIT;
