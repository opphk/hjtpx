-- Rollback: Enhanced Migration Tracking Table
-- Created: 2026-05-15
-- Description: Removes migration_tracking table and related objects

-- Drop views (in reverse order of creation)
DROP VIEW IF EXISTS v_migration_statistics;
DROP VIEW IF EXISTS v_migration_history;
DROP VIEW IF EXISTS v_pending_migrations;

-- Drop function
DROP FUNCTION IF EXISTS validate_migration_hash();

-- Drop indexes
DROP INDEX IF EXISTS idx_migration_tracking_hash;
DROP INDEX IF EXISTS idx_migration_tracking_executed_at;
DROP INDEX IF EXISTS idx_migration_tracking_status;
DROP INDEX IF EXISTS idx_migration_tracking_version;
DROP INDEX IF EXISTS idx_migration_tracking_unique;

-- Drop table
DROP TABLE IF EXISTS migration_tracking;
