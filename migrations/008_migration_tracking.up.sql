-- Migration: Enhanced Migration Tracking
-- Created: 2026-05-15
-- Description: Adds enhanced migration tracking with lock management and detailed audit trail

-- Create enhanced migration_locks table for distributed lock management
CREATE TABLE IF NOT EXISTS migration_locks (
  lock_key VARCHAR(100) PRIMARY KEY,
  locked_by VARCHAR(255) NOT NULL,
  locked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  expires_at TIMESTAMP NOT NULL,
  is_active BOOLEAN DEFAULT true
);

-- Create migration_dependencies table to track migration dependencies
CREATE TABLE IF NOT EXISTS migration_dependencies (
  id SERIAL PRIMARY KEY,
  migration_version INTEGER NOT NULL,
  depends_on_version INTEGER NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(migration_version, depends_on_version)
);

-- Create migration_changelog table for detailed audit trail
CREATE TABLE IF NOT EXISTS migration_changelog (
  id SERIAL PRIMARY KEY,
  migration_version INTEGER NOT NULL,
  migration_name VARCHAR(255) NOT NULL,
  action VARCHAR(20) NOT NULL,
  executed_by VARCHAR(255),
  executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  execution_time_ms INTEGER,
  rows_affected INTEGER,
  sql_statement TEXT,
  status VARCHAR(20) NOT NULL,
  error_details TEXT,
  machine_name VARCHAR(255),
  environment VARCHAR(50)
);

-- Create migration_health table for monitoring
CREATE TABLE IF NOT EXISTS migration_health (
  id SERIAL PRIMARY KEY,
  check_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  total_migrations INTEGER,
  successful_migrations INTEGER,
  failed_migrations INTEGER,
  last_migration_at TIMESTAMP,
  last_successful_migration_at TIMESTAMP,
  database_version VARCHAR(50),
  connection_status VARCHAR(20),
  health_status VARCHAR(20)
);

-- Create indexes for migration_locks
CREATE INDEX IF NOT EXISTS idx_migration_locks_active ON migration_locks(is_active, expires_at)
  WHERE is_active = true;

-- Create indexes for migration_dependencies
CREATE INDEX IF NOT EXISTS idx_migration_deps_version ON migration_dependencies(migration_version);
CREATE INDEX IF NOT EXISTS idx_migration_deps_depends ON migration_dependencies(depends_on_version);

-- Create indexes for migration_changelog
CREATE INDEX IF NOT EXISTS idx_changelog_version ON migration_changelog(migration_version);
CREATE INDEX IF NOT EXISTS idx_changelog_action ON migration_changelog(action);
CREATE INDEX IF EXISTS idx_changelog_status ON migration_changelog(status);
CREATE INDEX IF NOT EXISTS idx_changelog_time ON migration_changelog(executed_at DESC);

-- Create indexes for migration_health
CREATE INDEX IF NOT EXISTS idx_migration_health_time ON migration_health(check_time DESC);
CREATE INDEX IF NOT EXISTS idx_migration_health_status ON migration_health(health_status);

-- Create function to automatically clean up expired locks
CREATE OR REPLACE FUNCTION cleanup_expired_migration_locks()
RETURNS INTEGER AS $$
DECLARE
  deleted_count INTEGER;
BEGIN
  DELETE FROM migration_locks
  WHERE is_active = true AND expires_at < CURRENT_TIMESTAMP;

  GET DIAGNOSTICS deleted_count = ROW_COUNT;
  RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Create view for current migration status
CREATE OR REPLACE VIEW migration_status_summary AS
SELECT
  m.version,
  m.name,
  m.type,
  m.status,
  m.applied_at,
  m.execution_time_ms,
  CASE
    WHEN m.status = 'success' THEN
      EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - m.applied_at))::INTEGER
    ELSE NULL
  END as seconds_since_applied,
  md.dependency_count
FROM migrations m
LEFT JOIN (
  SELECT migration_version, COUNT(*) as dependency_count
  FROM migration_dependencies
  GROUP BY migration_version
) md ON m.version = md.migration_version
ORDER BY m.version;

-- Create view for pending migrations
CREATE OR REPLACE VIEW pending_migrations AS
SELECT
  m.version,
  m.name,
  CASE
    WHEN md.depends_on_version IS NOT NULL THEN 'has_dependencies'
    ELSE 'ready'
  END as readiness_status,
  array_agg(md.depends_on_version) as dependencies
FROM migrations m
LEFT JOIN migration_dependencies md ON m.version = md.migration_version
WHERE m.type = 'up'
GROUP BY m.version, m.name;

-- Insert current health check
INSERT INTO migration_health (
  total_migrations,
  successful_migrations,
  failed_migrations,
  last_migration_at,
  last_successful_migration_at,
  health_status
)
SELECT
  COUNT(*) as total_migrations,
  COUNT(*) FILTER (WHERE status = 'success') as successful_migrations,
  COUNT(*) FILTER (WHERE status = 'failed') as failed_migrations,
  MAX(applied_at) as last_migration_at,
  MAX(applied_at) FILTER (WHERE status = 'success') as last_successful_migration_at,
  CASE
    WHEN COUNT(*) FILTER (WHERE status = 'failed') > 0 THEN 'degraded'
    WHEN COUNT(*) = 0 THEN 'unknown'
    ELSE 'healthy'
  END as health_status
FROM migrations
WHERE type = 'up';

-- Grant permissions (adjust as needed for your security model)
-- GRANT SELECT ON migration_locks TO app_user;
-- GRANT SELECT ON migration_dependencies TO app_user;
-- GRANT SELECT ON migration_changelog TO app_user;
-- GRANT SELECT ON migration_health TO app_user;
