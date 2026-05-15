-- Rollback 007_read_write_split
-- Drop read/write splitting related objects

BEGIN;

-- Remove migration record
DELETE FROM schema_migrations WHERE version = '007_read_write_split';

-- Drop functions
DROP FUNCTION IF EXISTS get_replication_lag();
DROP FUNCTION IF EXISTS get_replica_status();
DROP FUNCTION IF EXISTS determine_route(TEXT);
DROP FUNCTION IF EXISTS health_check_primary();
DROP FUNCTION IF EXISTS health_check_replica();
DROP FUNCTION IF EXISTS check_read_replica_health();

-- Drop views
DROP VIEW IF EXISTS v_load_balancing_stats;
DROP VIEW IF EXISTS v_connection_info;

-- Drop tables
DROP TABLE IF EXISTS routing_rules CASCADE;

-- Drop publication
DROP PUBLICATION IF EXISTS captcha_changes;

-- Drop replication slot (careful!)
-- SELECT pg_drop_replication_slot('captcha_replica_slot');

-- Revoke permissions
REVOKE ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public FROM captcha_maintenance;
REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM captcha_maintenance;
REVOKE ALL PRIVILEGES ON SCHEMA public FROM captcha_maintenance;
REVOKE ALL PRIVILEGES ON DATABASE captcha_db FROM captcha_maintenance;

REVOKE USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public FROM captcha_primary_writer;
REVOKE SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public FROM captcha_primary_writer;
REVOKE USAGE ON SCHEMA public FROM captcha_primary_writer;
REVOKE CONNECT ON DATABASE captcha_db FROM captcha_primary_writer;

REVOKE SELECT ON ALL SEQUENCES IN SCHEMA public FROM captcha_replica_reader;
REVOKE SELECT ON ALL TABLES IN SCHEMA public FROM captcha_replica_reader;
REVOKE USAGE ON SCHEMA public FROM captcha_replica_reader;
REVOKE CONNECT ON DATABASE captcha_db FROM captcha_replica_reader;

-- Drop roles (optional)
-- DROP ROLE IF EXISTS captcha_maintenance;
-- DROP ROLE IF EXISTS captcha_primary_writer;
-- DROP ROLE IF EXISTS captcha_replica_reader;

COMMIT;
