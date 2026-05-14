-- Rollback for: 002_add_roles.sql
-- Generated: 2026-05-14
-- This script reverts the roles and permissions changes

BEGIN;

-- Remove role_permissions junction table data
DELETE FROM role_permissions;

-- Drop role_permissions junction table
DROP TABLE IF EXISTS role_permissions CASCADE;

-- Drop indexes
DROP INDEX IF EXISTS idx_users_reset_token;
DROP INDEX IF EXISTS idx_permissions_name;
DROP INDEX IF EXISTS idx_roles_name;

-- Drop permissions table
DROP TABLE IF EXISTS permissions CASCADE;

-- Drop roles table
DROP TABLE IF EXISTS roles CASCADE;

-- Remove columns from users table
ALTER TABLE users DROP COLUMN IF EXISTS reset_token_expires;
ALTER TABLE users DROP COLUMN IF EXISTS reset_token;

COMMIT;
