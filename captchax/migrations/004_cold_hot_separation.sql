-- CaptchaX Cold/Hot Data Separation - Archive Tables
-- Version: 004_cold_hot_separation
-- Description: Implement cold data archiving with separate tables and policies
-- Created: 2026-05-14

BEGIN;

-- 1. Create archive tables for historical data

-- captcha_logs_archive: Stores archived historical captcha logs
CREATE TABLE IF NOT EXISTS captcha_logs_archive (
    id SERIAL PRIMARY KEY,
    captcha_type VARCHAR(20) NOT NULL,
    client_id VARCHAR(64) NOT NULL,
    ip VARCHAR(45) NOT NULL,
    user_agent TEXT,
    result BOOLEAN NOT NULL,
    duration INTEGER NOT NULL,
    risk_score INTEGER DEFAULT 0,
    created_at TIMESTAMP NOT NULL,
    archived_at TIMESTAMP DEFAULT NOW(),
    archive_reason VARCHAR(50) DEFAULT 'age_based'
) PARTITION BY RANGE (created_at);

-- Create partitions for archived data by year
CREATE TABLE captcha_logs_archive_2025 PARTITION OF captcha_logs_archive
    FOR VALUES FROM ('2025-01-01') TO ('2026-01-01');

CREATE TABLE captcha_logs_archive_older PARTITION OF captcha_logs_archive
    FOR VALUES FROM ('1900-01-01') TO ('2025-01-01');

-- 2. Create indexes on archive table
CREATE INDEX ON captcha_logs_archive (captcha_type, created_at DESC);
CREATE INDEX ON captcha_logs_archive (client_id, created_at DESC);
CREATE INDEX ON captcha_logs_archive (ip, created_at DESC);
CREATE INDEX ON captcha_logs_archive (created_at DESC);
CREATE INDEX ON captcha_logs_archive (archived_at);

-- 3. Create archive metadata table
CREATE TABLE IF NOT EXISTS archive_metadata (
    id SERIAL PRIMARY KEY,
    source_table VARCHAR(100) NOT NULL,
    archive_table VARCHAR(100) NOT NULL,
    archived_count INTEGER DEFAULT 0,
    archived_before TIMESTAMP NOT NULL,
    archived_at TIMESTAMP DEFAULT NOW(),
    compressed BOOLEAN DEFAULT false,
    retention_days INTEGER,
    notes TEXT
);

CREATE INDEX ON archive_metadata (source_table);
CREATE INDEX ON archive_metadata (archived_at);

-- 4. Create archive policy configuration table
CREATE TABLE IF NOT EXISTS archive_policy (
    id SERIAL PRIMARY KEY,
    table_name VARCHAR(100) NOT NULL UNIQUE,
    archive_after_days INTEGER NOT NULL DEFAULT 90,
    archive_partition_type VARCHAR(20) DEFAULT 'yearly',
    compression_enabled BOOLEAN DEFAULT true,
    retention_days INTEGER NOT NULL DEFAULT 365,
    is_active BOOLEAN DEFAULT true,
    last_archived_at TIMESTAMP,
    last_archived_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Insert default archive policy for captcha_logs
INSERT INTO archive_policy (table_name, archive_after_days, archive_partition_type, compression_enabled, retention_days)
VALUES ('captcha_logs', 90, 'yearly', true, 365)
ON CONFLICT (table_name) DO NOTHING;

-- 5. Create trigger to auto-archive on insert (optional)
-- Note: This is disabled by default for performance reasons
CREATE OR REPLACE FUNCTION check_archive_threshold()
RETURNS TRIGGER AS $$
BEGIN
    -- Log warning if table size exceeds threshold (100k rows)
    -- In production, this could trigger async archive job
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 6. Create function to archive old data
CREATE OR REPLACE FUNCTION archive_captcha_logs(
    p_before_date TIMESTAMP,
    p_batch_size INTEGER DEFAULT 10000,
    p_compress BOOLEAN DEFAULT true
)
RETURNS TABLE(
    archived_count BIGINT,
    remaining_count BIGINT,
    execution_time_ms BIGINT
) AS $$
DECLARE
    v_start_time TIMESTAMP := NOW();
    v_archived BIGINT := 0;
    v_remaining BIGINT := 0;
    v_batch INTEGER := 0;
BEGIN
    LOOP
        -- Copy batch to archive
        WITH moved AS (
            DELETE FROM captcha_logs
            WHERE ctid IN (
                SELECT ctid FROM captcha_logs
                WHERE created_at < p_before_date
                LIMIT p_batch_size
            )
            RETURNING *
        )
        INSERT INTO captcha_logs_archive
            (captcha_type, client_id, ip, user_agent, result, duration, risk_score, created_at, archive_reason)
        SELECT 
            captcha_type, client_id, ip, user_agent, result, duration, risk_score, created_at, 'age_based'
        FROM moved;

        GET DIAGNOSTICS v_batch = ROW_COUNT;
        v_archived := v_archived + v_batch;
        
        EXIT WHEN v_batch < p_batch_size;
    END LOOP;

    -- Count remaining records
    SELECT COUNT(*) INTO v_remaining FROM captcha_logs WHERE created_at < p_before_date;

    -- Record archive metadata
    INSERT INTO archive_metadata 
        (source_table, archive_table, archived_count, archived_before, compressed)
    VALUES 
        ('captcha_logs', 'captcha_logs_archive', v_archived, p_before_date, p_compress);

    -- Update policy last run time
    UPDATE archive_policy 
    SET last_archived_at = NOW(), 
        last_archived_count = v_archived,
        updated_at = NOW()
    WHERE table_name = 'captcha_logs';

    RETURN QUERY SELECT v_archived, v_remaining, EXTRACT(MILLISECONDS FROM NOW() - v_start_time)::BIGINT;
END;
$$ LANGUAGE plpgsql;

-- 7. Create function to restore archived data
CREATE OR REPLACE FUNCTION restore_archived_logs(
    p_start_date TIMESTAMP,
    p_end_date TIMESTAMP,
    p_limit INTEGER DEFAULT 1000
)
RETURNS TABLE(restored_count BIGINT) AS $$
DECLARE
    v_restored BIGINT := 0;
BEGIN
    -- Restore in batches
    LOOP
        WITH restored AS (
            DELETE FROM captcha_logs_archive
            WHERE created_at >= p_start_date 
            AND created_at <= p_end_date
            LIMIT p_limit
            RETURNING *
        )
        INSERT INTO captcha_logs (captcha_type, client_id, ip, user_agent, result, duration, risk_score, created_at)
        SELECT captcha_type, client_id, ip, user_agent, result, duration, risk_score, created_at
        FROM restored;

        GET DIAGNOSTICS v_restored = ROW_COUNT;
        
        EXIT WHEN v_restored < p_limit;
    END LOOP;

    RETURN QUERY SELECT v_restored;
END;
$$ LANGUAGE plpgsql;

-- 8. Create function to get archive statistics
CREATE OR REPLACE FUNCTION get_archive_stats()
RETURNS TABLE(
    table_name TEXT,
    active_count BIGINT,
    archived_count BIGINT,
    archive_ratio NUMERIC,
    oldest_active TIMESTAMP,
    newest_archived TIMESTAMP,
    last_archive_at TIMESTAMP
) AS $$
BEGIN
    RETURN QUERY SELECT
        'captcha_logs'::TEXT,
        (SELECT COUNT(*) FROM captcha_logs)::BIGINT AS active_count,
        (SELECT COUNT(*) FROM captcha_logs_archive)::BIGINT AS archived_count,
        CASE 
            WHEN (SELECT COUNT(*) FROM captcha_logs_archive) > 0 
            THEN ROUND(
                (SELECT COUNT(*) FROM captcha_logs_archive)::NUMERIC / 
                ((SELECT COUNT(*) FROM captcha_logs) + (SELECT COUNT(*) FROM captcha_logs_archive))::NUMERIC * 100, 
                2
            )
            ELSE 0 
        END AS archive_ratio,
        (SELECT MIN(created_at) FROM captcha_logs) AS oldest_active,
        (SELECT MAX(created_at) FROM captcha_logs_archive) AS newest_archived,
        (SELECT MAX(archived_at) FROM archive_metadata WHERE source_table = 'captcha_logs') AS last_archive_at;
END;
$$ LANGUAGE plpgsql;

-- 9. Create function to purge old archived data (retention policy)
CREATE OR REPLACE FUNCTION purge_archived_data(p_retention_days INTEGER DEFAULT 365)
RETURNS TABLE(purged_count BIGINT) AS $$
DECLARE
    v_purged BIGINT := 0;
BEGIN
    -- Delete archived data older than retention period
    DELETE FROM captcha_logs_archive
    WHERE archived_at < NOW() - (p_retention_days || ' days')::INTERVAL;

    GET DIAGNOSTICS v_purged = ROW_COUNT;

    RETURN QUERY SELECT v_purged;
END;
$$ LANGUAGE plpgsql;

-- Record migration
INSERT INTO schema_migrations (version) VALUES ('004_cold_hot_separation')
ON CONFLICT (version) DO NOTHING;

COMMIT;
