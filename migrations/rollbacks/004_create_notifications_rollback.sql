-- Rollback for: 004_create_notifications.sql
-- Generated: 2026-05-14
-- This script reverts the notifications table creation

BEGIN;

-- Drop indexes for notifications
DROP INDEX IF EXISTS idx_notifications_ttl;
DROP INDEX IF EXISTS idx_notifications_status;
DROP INDEX IF EXISTS idx_notifications_expires_at;
DROP INDEX IF EXISTS idx_notifications_type;
DROP INDEX IF EXISTS idx_notifications_user_created;
DROP INDEX IF EXISTS idx_notifications_user_status;
DROP INDEX IF EXISTS idx_notifications_user_id;

-- Drop notifications table
DROP TABLE IF EXISTS notifications CASCADE;

COMMIT;
