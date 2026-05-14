-- Rollback for: 005_performance_indexes.sql
-- Generated: 2026-05-14
-- This script reverts the performance optimization indexes

BEGIN;

-- Drop user indexes
DROP INDEX IF EXISTS idx_users_created_range;
DROP INDEX IF EXISTS idx_users_search;
DROP INDEX IF EXISTS idx_notifications_unread;
DROP INDEX IF EXISTS idx_users_active;
DROP INDEX IF EXISTS idx_query_stats_execution_time;
DROP INDEX IF EXISTS idx_sessions_expires_at;
DROP INDEX IF EXISTS idx_sessions_token;
DROP INDEX IF EXISTS idx_sessions_user_id;
DROP INDEX IF EXISTS idx_notifications_created_at;
DROP INDEX IF EXISTS idx_notifications_user_read;
DROP INDEX IF EXISTS idx_notifications_user_id;
DROP INDEX IF EXISTS idx_users_list;
DROP INDEX IF EXISTS idx_users_email_role;
DROP INDEX IF EXISTS idx_users_created_at;
DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_users_email;

COMMIT;
