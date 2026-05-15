-- Rollback: Notifications System
-- Description: Removes notifications table and related objects

-- Drop trigger
DROP TRIGGER IF EXISTS update_notifications_updated_at ON notifications;

-- Drop indexes
DROP INDEX IF EXISTS idx_notifications_user_id;
DROP INDEX IF EXISTS idx_notifications_user_status;
DROP INDEX IF EXISTS idx_notifications_user_created;
DROP INDEX IF EXISTS idx_notifications_type;
DROP INDEX IF EXISTS idx_notifications_expires_at;
DROP INDEX IF EXISTS idx_notifications_status;
DROP INDEX IF EXISTS idx_notifications_is_read;
DROP INDEX IF EXISTS idx_notifications_ttl;

-- Drop table
DROP TABLE IF EXISTS notifications;
