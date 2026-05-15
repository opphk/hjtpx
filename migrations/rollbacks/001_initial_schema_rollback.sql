-- Rollback for: 001_initial_schema.sql
-- Generated: 2026-05-14
-- This script reverts the initial schema changes

BEGIN;

-- Drop trigger for users table
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop function to update updated_at timestamp
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop migrations tracking table
DROP TABLE IF EXISTS migrations;

-- Drop sessions table
DROP TABLE IF EXISTS sessions CASCADE;

-- Drop indexes on users table
DROP INDEX IF EXISTS idx_sessions_expires_at;
DROP INDEX IF EXISTS idx_sessions_token;
DROP INDEX IF EXISTS idx_sessions_user_id;
DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_users_created_at;
DROP INDEX IF EXISTS idx_users_email;

-- Drop users table
DROP TABLE IF EXISTS users CASCADE;

COMMIT;
