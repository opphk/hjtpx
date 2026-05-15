-- Rollback: Enhanced Migration Tracking
-- Description: Removes enhanced migration tracking tables and functions

-- Drop views
DROP VIEW IF EXISTS migration_status_summary;
DROP VIEW IF EXISTS pending_migrations;

-- Drop function
DROP FUNCTION IF EXISTS cleanup_expired_migration_locks;

-- Drop indexes (in reverse order of creation)
DROP INDEX IF EXISTS idx_migration_health_status;
DROP INDEX IF EXISTS idx_migration_health_time;
DROP INDEX IF EXISTS idx_changelog_time;
DROP INDEX IF EXISTS idx_changelog_status;
DROP INDEX IF EXISTS idx_changelog_action;
DROP INDEX IF EXISTS idx_changelog_version;
DROP INDEX IF EXISTS idx_migration_deps_depends;
DROP INDEX IF EXISTS idx_migration_deps_version;
DROP INDEX IF EXISTS idx_migration_locks_active;

-- Drop tables (in reverse order of dependencies)
DROP TABLE IF EXISTS migration_health;
DROP TABLE IF EXISTS migration_changelog;
DROP TABLE IF EXISTS migration_dependencies;
DROP TABLE IF EXISTS migration_locks;
