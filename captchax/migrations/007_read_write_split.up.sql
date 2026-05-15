-- CaptchaX Read/Write Splitting - Primary/Replica Configuration
-- Version: 007_read_write_split
-- Description: Configure database for read/write splitting with primary and replica
-- Created: 2026-05-14

BEGIN;

-- 1. Create application role for read-only replica access
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'captcha_replica_reader') THEN
        CREATE ROLE captcha_replica_reader WITH LOGIN PASSWORD 'replica_reader_pass_2026';
    END IF;
END
$$;

-- Grant read-only permissions to replica reader
GRANT CONNECT ON DATABASE captcha_db TO captcha_replica_reader;
GRANT USAGE ON SCHEMA public TO captcha_replica_reader;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO captcha_replica_reader;
GRANT SELECT ON ALL SEQUENCES IN SCHEMA public TO captcha_replica_reader;

-- 2. Create application role for read/write primary access
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'captcha_primary_writer') THEN
        CREATE ROLE captcha_primary_writer WITH LOGIN PASSWORD 'primary_writer_pass_2026';
    END IF;
END
$$;

GRANT CONNECT ON DATABASE captcha_db TO captcha_primary_writer;
GRANT USAGE ON SCHEMA public TO captcha_primary_writer;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO captcha_primary_writer;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO captcha_primary_writer;

-- 3. Create role for maintenance operations
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'captcha_maintenance') THEN
        CREATE ROLE captcha_maintenance WITH LOGIN PASSWORD 'maintenance_pass_2026';
    END IF;
END
$$;

GRANT ALL PRIVILEGES ON DATABASE captcha_db TO captcha_maintenance;
GRANT ALL PRIVILEGES ON SCHEMA public TO captcha_maintenance;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO captcha_maintenance;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO captcha_maintenance;

-- 4. Create replication slot for streaming replication (if not using pgpool)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_replication_slots WHERE slot_name = 'captcha_replica_slot') THEN
        PERFORM pg_create_physical_replication_slot('captcha_replica_slot', false);
    END IF;
END
$$;

-- 5. Create publication for logical replication (optional)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_publication WHERE pubname = 'captcha_changes') THEN
        CREATE PUBLICATION captcha_changes FOR TABLE captcha_logs, blacklist, whitelist, captcha_config, admins;
    END IF;
END
$$;

-- 6. Create function to check replication lag
CREATE OR REPLACE FUNCTION get_replication_lag()
RETURNS TABLE(
    slot_name TEXT,
    lag_bytes BIGINT,
    lag_seconds DOUBLE PRECISION
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        slot_name,
        pg_wal_lag_diff(slot_name, 0, 0) AS lag_bytes,
        GREATEST(0, EXTRACT(EPOCH FROM (NOW() - replay_timestamp))) AS lag_seconds
    FROM pg_replication_slots
    WHERE active = true;
END;
$$ LANGUAGE plpgsql;

-- 7. Create function to get replica status
CREATE OR REPLACE FUNCTION get_replica_status()
RETECTS TABLE(
    client_addr INET,
    state TEXT,
    sent_lsn TEXT,
    write_lsn TEXT,
    flush_lsn TEXT,
    replay_lsn TEXT,
    write_lag TEXT,
    flush_lag TEXT,
    replay_lag TEXT,
    sync_state TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        client_addr,
        state,
        sent_lsn::TEXT,
        write_lsn::TEXT,
        flush_lsn::TEXT,
        replay_lsn::TEXT,
        write_lag::TEXT,
        flush_lag::TEXT,
        replay_lag::TEXT,
        sync_state::TEXT
    FROM pg_stat_replication;
END;
$$ LANGUAGE plpgsql;

-- 8. Create view for connection pool routing hints
CREATE OR REPLACE VIEW v_connection_info AS
SELECT 
    pid,
    usename,
    application_name,
    client_addr,
    state,
    now() - state_change AS connection_duration,
    wait_event_type,
    wait_event,
    query
FROM pg_stat_activity
WHERE datname = current_database()
AND state IS NOT NULL
ORDER BY state, connection_duration DESC;

-- 9. Create routing rules table for application-level routing
CREATE TABLE IF NOT EXISTS routing_rules (
    id SERIAL PRIMARY KEY,
    route_name VARCHAR(50) NOT NULL UNIQUE,
    target_type VARCHAR(20) NOT NULL CHECK (target_type IN ('primary', 'replica', 'any')),
    priority INTEGER DEFAULT 1,
    match_pattern TEXT,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW()
);

INSERT INTO routing_rules (route_name, target_type, priority, match_pattern, description) VALUES
    ('captcha_logs_read', 'replica', 1, 'SELECT.*FROM captcha_logs', 'Read from replica for captcha logs'),
    ('captcha_logs_write', 'primary', 1, 'INSERT.*INTO captcha_logs', 'Write to primary for captcha logs'),
    ('captcha_logs_update', 'primary', 1, 'UPDATE.*captcha_logs', 'Update primary for captcha logs'),
    ('captcha_logs_delete', 'primary', 1, 'DELETE.*FROM captcha_logs', 'Delete from primary for captcha logs'),
    ('blacklist_read', 'any', 1, 'SELECT.*FROM blacklist', 'Blacklist can be read from any'),
    ('blacklist_write', 'primary', 1, 'INSERT.*INTO blacklist', 'Blacklist writes to primary'),
    ('stats_aggregation', 'replica', 1, 'SELECT.*COUNT.*|SELECT.*AVG.*|SELECT.*SUM.*', 'Stats aggregation from replica'),
    ('admin_operations', 'primary', 1, 'SELECT.*FROM admins.*|UPDATE.*admins', 'Admin operations on primary')
ON CONFLICT (route_name) DO NOTHING;

-- 10. Create function to determine routing target
CREATE OR REPLACE FUNCTION determine_route(p_query TEXT)
RETURNS VARCHAR(20) AS $$
DECLARE
    v_route VARCHAR(20) := 'primary';
    v_rule RECORD;
BEGIN
    FOR v_rule IN 
        SELECT target_type, match_pattern 
        FROM routing_rules 
        WHERE is_active = true 
        ORDER BY priority DESC
    LOOP
        IF p_query ~* v_rule.match_pattern THEN
            RETURN v_rule.target_type;
        END IF;
    END LOOP;
    RETURN v_route;
END;
$$ LANGUAGE plpgsql;

-- 11. Create load balancing stats view
CREATE OR REPLACE VIEW v_load_balancing_stats AS
SELECT 
    application_name,
    COUNT(*) AS connection_count,
    COUNT(*) FILTER (WHERE state = 'active') AS active_count,
    COUNT(*) FILTER (WHERE state = 'idle') AS idle_count,
    COUNT(*) FILTER (WHERE state = 'idle in transaction') AS idle_in_tx_count,
    COUNT(*) FILTER (WHERE wait_event IS NOT NULL) AS waiting_count
FROM pg_stat_activity
WHERE datname = current_database()
GROUP BY application_name;

-- 12. Create health check function for connection validation
CREATE OR REPLACE FUNCTION health_check_primary() 
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM pg_stat_database 
        WHERE datname = current_database() 
        AND NOT is_in_recovery
    );
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION health_check_replica() 
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM pg_stat_database 
        WHERE datname = current_database() 
        AND is_in_recovery
    );
END;
$$ LANGUAGE plpgsql;

-- 13. Create read replica lag monitoring
CREATE OR REPLACE FUNCTION check_read_replica_health()
RETURNS TABLE(
    replica_name TEXT,
    lag_bytes BIGINT,
    lag_seconds DOUBLE PRECISION,
    is_healthy BOOLEAN
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COALESCE(application_name, 'unknown')::TEXT,
        pg_wal_lag_diff(COALESCE(slot_name, ''), 0, 0)::BIGINT,
        GREATEST(0, EXTRACT(EPOCH FROM (NOW() - COALESCE(replay_timestamp, NOW()))))::DOUBLE PRECISION,
        CASE 
            WHEN pg_wal_lag_diff(COALESCE(slot_name, ''), 0, 0) < 10485760  -- < 10MB
            AND GREATEST(0, EXTRACT(EPOCH FROM (NOW() - COALESCE(replay_timestamp, NOW())))) < 30  -- < 30 seconds
            THEN true
            ELSE false
        END AS is_healthy
    FROM pg_stat_replication;
END;
$$ LANGUAGE plpgsql;

-- Record migration
INSERT INTO schema_migrations (version) VALUES ('007_read_write_split')
ON CONFLICT (version) DO NOTHING;

COMMIT;
