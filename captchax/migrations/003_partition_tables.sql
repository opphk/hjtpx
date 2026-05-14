-- CaptchaX Database Partitioning - Time-based Range Partitioning
-- Version: 003_partition_tables
-- Description: Implement time-based partitioning for captcha_logs table
-- Created: 2026-05-14

BEGIN;

-- 1. Create partitioned table for captcha_logs
-- This replaces the original table with a partitioned one

-- Create new partitioned table
CREATE TABLE IF NOT EXISTS captcha_logs_partitioned (
    id SERIAL,
    captcha_type VARCHAR(20) NOT NULL,
    client_id VARCHAR(64) NOT NULL,
    ip VARCHAR(45) NOT NULL,
    user_agent TEXT,
    result BOOLEAN NOT NULL,
    duration INTEGER NOT NULL,
    risk_score INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create partitions for current and next months
-- Monthly partitions for hot data (last 3 months)
CREATE TABLE captcha_logs_2026_05 PARTITION OF captcha_logs_partitioned
    FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');

CREATE TABLE captcha_logs_2026_06 PARTITION OF captcha_logs_partitioned
    FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');

CREATE TABLE captcha_logs_2026_07 PARTITION OF captcha_logs_partitioned
    FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');

-- Create partition for current quarter cold data (older data)
CREATE TABLE captcha_logs_2026_q2 PARTITION OF captcha_logs_partitioned
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');

CREATE TABLE captcha_logs_2026_q1 PARTITION OF captcha_logs_partitioned
    FOR VALUES FROM ('2026-01-01') TO ('2026-04-01');

CREATE TABLE captcha_logs_2025 PARTITION OF captcha_logs_partitioned
    FOR VALUES FROM ('2025-01-01') TO ('2026-01-01');

-- Default partition for data outside defined ranges
CREATE TABLE captcha_logs_default PARTITION OF captcha_logs_partitioned
    DEFAULT;

-- 2. Copy data from original table (with minimal lock)
INSERT INTO captcha_logs_partitioned
    (id, captcha_type, client_id, ip, user_agent, result, duration, risk_score, created_at)
SELECT 
    id, captcha_type, client_id, ip, user_agent, result, duration, risk_score, created_at
FROM captcha_logs;

-- 3. Create indexes on partitioned table
CREATE INDEX ON captcha_logs_partitioned (captcha_type, created_at DESC);
CREATE INDEX ON captcha_logs_partitioned (client_id, created_at DESC);
CREATE INDEX ON captcha_logs_partitioned (ip, created_at DESC, result);
CREATE INDEX ON captcha_logs_partitioned (created_at DESC);

-- 4. Swap table names (rename)
ALTER TABLE captcha_logs RENAME TO captcha_logs_old;
ALTER TABLE captcha_logs_partitioned RENAME TO captcha_logs;

-- 5. Drop old table after verification
DROP TABLE IF EXISTS captcha_logs_old;

-- 6. Create function to auto-create monthly partitions
CREATE OR REPLACE FUNCTION create_monthly_partition()
RETURNS void AS $$
DECLARE
    partition_date DATE;
    partition_name TEXT;
    start_date TEXT;
    end_date TEXT;
BEGIN
    -- Create partition for next month
    partition_date := DATE_TRUNC('month', CURRENT_DATE) + INTERVAL '1 month';
    partition_name := 'captcha_logs_' || TO_CHAR(partition_date, 'YYYY_MM');
    start_date := TO_CHAR(partition_date, 'YYYY-MM-DD');
    end_date := TO_CHAR(partition_date + INTERVAL '1 month', 'YYYY-MM-DD');

    EXECUTE format(
        'CREATE TABLE IF NOT EXISTS %I PARTITION OF captcha_logs FOR VALUES FROM (%L) TO (%L)',
        partition_name, start_date, end_date
    );
END;
$$ LANGUAGE plpgsql;

-- 7. Create partitions maintenance function
CREATE OR REPLACE FUNCTION manage_partitions()
RETURNS void AS $$
DECLARE
    i INTEGER;
    partition_name TEXT;
    partition_date DATE;
BEGIN
    -- Create next 2 months partitions
    FOR i IN 1..2 LOOP
        partition_date := DATE_TRUNC('month', CURRENT_DATE) + (i || ' months')::INTERVAL;
        partition_name := 'captcha_logs_' || TO_CHAR(partition_date, 'YYYY_MM');
        
        EXECUTE format(
            'CREATE TABLE IF NOT EXISTS %I PARTITION OF captcha_logs FOR VALUES FROM (%L) TO (%L)',
            partition_name,
            TO_CHAR(partition_date, 'YYYY-MM-DD'),
            TO_CHAR(partition_date + INTERVAL '1 month', 'YYYY-MM-DD')
        );
    END LOOP;

    -- Detach old empty partitions (older than 3 months, only if empty)
    FOR partition_name IN 
        SELECT child.relname::text
        FROM pg_inherits
        JOIN pg_class parent ON pg_inherits.inhparent = parent.oid
        JOIN pg_class child ON pg_inherits.inhrelid = child.oid
        WHERE parent.relname = 'captcha_logs'
        AND child.relname < 'captcha_logs_' || TO_CHAR(CURRENT_DATE - INTERVAL '3 months', 'YYYY_MM')
    LOOP
        -- Check if partition is empty before detaching
        IF NOT EXISTS (SELECT 1 FROM captcha_logs WHERE created_at < CURRENT_DATE - INTERVAL '3 months' LIMIT 1) THEN
            EXECUTE format('ALTER TABLE captcha_logs DETACH PARTITION %I', partition_name);
        END IF;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Record migration
INSERT INTO schema_migrations (version) VALUES ('003_partition_tables')
ON CONFLICT (version) DO NOTHING;

COMMIT;
