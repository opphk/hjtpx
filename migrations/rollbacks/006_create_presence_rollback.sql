-- Rollback for: 006_create_presence.sql
-- Generated: 2026-05-14
-- This script reverts the presence table creation

BEGIN;

-- Drop presence change trigger
DROP TRIGGER IF EXISTS presence_change_trigger ON presence;

-- Drop function to log presence changes
DROP FUNCTION IF EXISTS log_presence_change();

-- Drop trigger for presence timestamp
DROP TRIGGER IF EXISTS update_presence_updated_at ON presence;

-- Drop function to update presence timestamp
DROP FUNCTION IF EXISTS update_presence_timestamp();

-- Drop presence_history indexes
DROP INDEX IF EXISTS idx_presence_history_created_at;
DROP INDEX IF EXISTS idx_presence_history_action;
DROP INDEX IF EXISTS idx_presence_history_user_id;

-- Drop presence_history table
DROP TABLE IF EXISTS presence_history CASCADE;

-- Drop presence indexes
DROP INDEX IF EXISTS idx_presence_online;
DROP INDEX IF EXISTS idx_presence_last_seen;
DROP INDEX IF EXISTS idx_presence_status;
DROP INDEX IF EXISTS idx_presence_socket_id;
DROP INDEX IF EXISTS idx_presence_user_id;

-- Drop presence table
DROP TABLE IF EXISTS presence CASCADE;

-- Drop active_users view
DROP VIEW IF EXISTS active_users;

-- Remove migration tracking entry
DELETE FROM migrations WHERE name = '006_create_presence';

COMMIT;
