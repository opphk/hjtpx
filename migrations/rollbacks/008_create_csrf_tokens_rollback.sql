-- Rollback for: 008_create_csrf_tokens.sql
-- Generated: 2026-05-14
-- This script reverts the CSRF tokens, audit logs, and security tables creation

BEGIN;

-- Drop account_locks indexes
DROP INDEX IF EXISTS idx_account_locks_locked_until;
DROP INDEX IF EXISTS idx_account_locks_user_id;

-- Drop account_locks table
DROP TABLE IF EXISTS account_locks CASCADE;

-- Drop security_events indexes
DROP INDEX IF EXISTS idx_security_events_created;
DROP INDEX IF EXISTS idx_security_events_severity;
DROP INDEX IF EXISTS idx_security_events_user;
DROP INDEX IF EXISTS idx_security_events_type;

-- Drop security_events table
DROP TABLE IF EXISTS security_events CASCADE;

-- Drop audit_logs indexes
DROP INDEX IF EXISTS idx_audit_logs_resource;
DROP INDEX IF EXISTS idx_audit_logs_created_at;
DROP INDEX IF EXISTS idx_audit_logs_action;
DROP INDEX IF EXISTS idx_audit_logs_user_id;

-- Drop audit_logs table
DROP TABLE IF EXISTS audit_logs CASCADE;

-- Drop csrf_tokens indexes
DROP INDEX IF EXISTS idx_csrf_tokens_expires;
DROP INDEX IF EXISTS idx_csrf_tokens_user_id;

-- Drop csrf_tokens table
DROP TABLE IF EXISTS csrf_tokens CASCADE;

COMMIT;
