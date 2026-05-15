-- Rollback: Presence and Online Status
-- Description: Removes presence table and related objects

-- Drop view
DROP VIEW IF EXISTS active_users;

-- Drop trigger
DROP TRIGGER IF EXISTS presence_change_trigger ON presence;
DROP TRIGGER IF EXISTS update_presence_updated_at ON presence;

-- Drop function
DROP FUNCTION IF EXISTS log_presence_change;

-- Drop indexes
DROP INDEX IF EXISTS idx_presence_user_id;
DROP INDEX IF EXISTS idx_presence_socket_id;
DROP INDEX IF EXISTS idx_presence_status;
DROP INDEX IF EXISTS idx_presence_last_seen;
DROP INDEX IF EXISTS idx_presence_online;
DROP INDEX IF EXISTS idx_presence_history_user_id;
DROP INDEX IF EXISTS idx_presence_history_action;
DROP INDEX IF EXISTS idx_presence_history_created_at;

-- Drop tables
DROP TABLE IF EXISTS presence_history;
DROP TABLE IF EXISTS presence;
