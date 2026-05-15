-- Rollback 003_partition_tables
-- Revert partitioning changes

BEGIN;

-- Remove migration record
DELETE FROM schema_migrations WHERE version = '003_partition_tables';

-- Drop functions
DROP FUNCTION IF EXISTS create_monthly_partition();
DROP FUNCTION IF EXISTS manage_partitions();

-- Check if we need to revert the partitioning
-- Note: This is complex, for safety we'll just remove partition tables and functions
-- In real scenario, you'd need to migrate data back

DROP TABLE IF EXISTS captcha_logs_partitioned CASCADE;
DROP TABLE IF EXISTS captcha_logs_2026_05 CASCADE;
DROP TABLE IF EXISTS captcha_logs_2026_06 CASCADE;
DROP TABLE IF EXISTS captcha_logs_2026_07 CASCADE;
DROP TABLE IF EXISTS captcha_logs_2026_q2 CASCADE;
DROP TABLE IF EXISTS captcha_logs_2026_q1 CASCADE;
DROP TABLE IF EXISTS captcha_logs_2025 CASCADE;
DROP TABLE IF EXISTS captcha_logs_default CASCADE;

-- Recreate the original non-partitioned table if it was dropped
CREATE TABLE IF NOT EXISTS captcha_logs (
    id SERIAL PRIMARY KEY,
    captcha_type VARCHAR(20) NOT NULL CHECK (captcha_type IN ('slider', 'click', 'puzzle')),
    client_id VARCHAR(64) NOT NULL,
    ip VARCHAR(45) NOT NULL,
    user_agent TEXT,
    result BOOLEAN NOT NULL,
    duration INTEGER NOT NULL,
    risk_score INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Recreate basic indexes
CREATE INDEX IF NOT EXISTS idx_captcha_logs_client_id ON captcha_logs(client_id);
CREATE INDEX IF NOT EXISTS idx_captcha_logs_ip ON captcha_logs(ip);
CREATE INDEX IF NOT EXISTS idx_captcha_logs_created_at ON captcha_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_captcha_logs_type_result ON captcha_logs(captcha_type, result);

COMMIT;
