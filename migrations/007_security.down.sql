-- Rollback: Security Features
-- Description: Removes security-related tables and indexes

-- Drop indexes
DROP INDEX IF EXISTS idx_csrf_tokens_user_id;
DROP INDEX IF EXISTS idx_csrf_tokens_expires;
DROP INDEX IF EXISTS idx_csrf_tokens_session;
DROP INDEX IF EXISTS idx_audit_logs_user_id;
DROP INDEX IF EXISTS idx_audit_logs_action;
DROP INDEX IF EXISTS idx_audit_logs_created_at;
DROP INDEX IF EXISTS idx_audit_logs_resource;
DROP INDEX IF EXISTS idx_security_events_type;
DROP INDEX IF EXISTS idx_security_events_user;
DROP INDEX IF EXISTS idx_security_events_severity;
DROP INDEX IF EXISTS idx_security_events_created;
DROP INDEX IF EXISTS idx_security_events_resolved;
DROP INDEX IF EXISTS idx_account_locks_user_id;
DROP INDEX IF EXISTS idx_account_locks_locked_until;
DROP INDEX IF EXISTS idx_account_locks_active;

-- Drop tables
DROP TABLE IF EXISTS account_locks;
DROP TABLE IF EXISTS security_events;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS csrf_tokens;
