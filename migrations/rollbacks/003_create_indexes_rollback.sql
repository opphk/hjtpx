-- Rollback for: 003_create_indexes.sql
-- Generated: 2026-05-14
-- This script reverts the additional indexes

BEGIN;

-- Drop composite indexes
DROP INDEX IF EXISTS idx_sessions_token_expires;
DROP INDEX IF EXISTS idx_sessions_active;
DROP INDEX IF EXISTS idx_sessions_expires_at;

-- Drop user indexes
DROP INDEX IF EXISTS idx_users_created_at;
DROP INDEX IF EXISTS idx_users_is_active_role;
DROP INDEX IF EXISTS idx_users_email_role;

COMMIT;
