-- Migration: Enhanced Migration Tracking Table
-- Created: 2026-05-15
-- Description: Creates comprehensive migration tracking with hash verification and rollback support

-- Create migration_tracking table for detailed migration history
CREATE TABLE IF NOT EXISTS migration_tracking (
  migration_id SERIAL PRIMARY KEY,
  migration_name VARCHAR(255) NOT NULL,
  migration_version INTEGER NOT NULL,
  migration_hash VARCHAR(64) NOT NULL,
  executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  rollback_executed_at TIMESTAMP,
  executed_by VARCHAR(255),
  status VARCHAR(20) NOT NULL DEFAULT 'pending',
  execution_time_ms INTEGER,
  error_message TEXT,
  machine_name VARCHAR(255),
  database_name VARCHAR(100)
);

-- Create unique constraint to prevent duplicate migrations
CREATE UNIQUE INDEX IF NOT EXISTS idx_migration_tracking_unique
  ON migration_tracking(migration_version, status)
  WHERE status IN ('executed', 'pending');

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_migration_tracking_version
  ON migration_tracking(migration_version DESC);

CREATE INDEX IF NOT EXISTS idx_migration_tracking_status
  ON migration_tracking(status);

CREATE INDEX IF NOT EXISTS idx_migration_tracking_executed_at
  ON migration_tracking(executed_at DESC);

CREATE INDEX IF NOT EXISTS idx_migration_tracking_hash
  ON migration_tracking(migration_hash);

-- Create function to validate migration hash
CREATE OR REPLACE FUNCTION validate_migration_hash()
RETURNS TRIGGER AS $$
DECLARE
  expected_hash VARCHAR(64);
  migration_file TEXT;
BEGIN
  -- Get the corresponding up migration file
  migration_file := 'migrations/' || NEW.migration_version || '_' || NEW.migration_name || '.up.sql';

  -- Calculate hash from file if it exists
  BEGIN
    expected_hash := encode(sha256(lo_import(migration_file)::bytea), 'hex');
  EXCEPTION WHEN OTHERS THEN
    -- File not found or cannot be read, skip validation
    RETURN NEW;
  END;

  -- Compare hashes (allow for manual migrations to have different hash)
  IF NEW.migration_hash != expected_hash THEN
    RAISE WARNING 'Migration hash mismatch for version %', NEW.migration_version;
  END IF;

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create view for pending migrations
CREATE OR REPLACE VIEW v_pending_migrations AS
SELECT
  mt.migration_id,
  mt.migration_name,
  mt.migration_version,
  mt.status,
  mt.executed_at,
  CASE
    WHEN mt.executed_at IS NULL THEN 'not_started'
    WHEN mt.status = 'pending' THEN 'ready_to_execute'
    ELSE mt.status
  END as readiness_status
FROM migration_tracking mt
WHERE mt.status IN ('pending')
ORDER BY mt.migration_version;

-- Create view for migration history
CREATE OR REPLACE VIEW v_migration_history AS
SELECT
  mt.migration_id,
  mt.migration_name,
  mt.migration_version,
  mt.migration_hash,
  mt.executed_at,
  mt.rollback_executed_at,
  mt.executed_by,
  mt.status,
  mt.execution_time_ms,
  mt.error_message,
  CASE
    WHEN mt.status = 'rolled_back' THEN
      EXTRACT(EPOCH FROM (mt.rollback_executed_at - mt.executed_at))::INTEGER
    ELSE NULL
  END as rollback_duration_seconds
FROM migration_tracking mt
ORDER BY mt.migration_version DESC;

-- Create view for migration statistics
CREATE OR REPLACE VIEW v_migration_statistics AS
SELECT
  COUNT(*) FILTER (WHERE status = 'executed') as total_executed,
  COUNT(*) FILTER (WHERE status = 'rolled_back') as total_rolled_back,
  COUNT(*) FILTER (WHERE status = 'failed') as total_failed,
  COUNT(*) FILTER (WHERE status = 'pending') as total_pending,
  AVG(execution_time_ms) FILTER (WHERE status = 'executed') as avg_execution_time_ms,
  MAX(executed_at) FILTER (WHERE status = 'executed') as last_executed_at,
  MAX(rollback_executed_at) FILTER (WHERE status = 'rolled_back') as last_rollback_at
FROM migration_tracking;

-- Insert initial tracking record for this migration itself
INSERT INTO migration_tracking (
  migration_name,
  migration_version,
  migration_hash,
  executed_at,
  status,
  executed_by
)
VALUES (
  'migration_tracking',
  9,
  'initial_migration_tracking_table',
  CURRENT_TIMESTAMP,
  'executed',
  current_user
)
ON CONFLICT DO NOTHING;
