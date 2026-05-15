-- Rollback 008_monitoring_metrics
-- Drop monitoring related objects

BEGIN;

-- Remove migration record
DELETE FROM schema_migrations WHERE version = '008_monitoring_metrics';

-- Drop functions
DROP FUNCTION IF EXISTS collect_db_metrics();
DROP FUNCTION IF EXISTS get_detailed_table_sizes();
DROP FUNCTION IF EXISTS get_db_health_status();
DROP FUNCTION IF EXISTS aggregate_metrics(VARCHAR, TIMESTAMP, TIMESTAMP, VARCHAR);

-- Drop views
DROP VIEW IF EXISTS v_monitoring_alerts;
DROP VIEW IF EXISTS v_long_running_queries;
DROP VIEW IF EXISTS v_blocking_queries;
DROP VIEW IF EXISTS v_maintenance_history;
DROP VIEW IF EXISTS v_replication_metrics;
DROP VIEW IF EXISTS v_query_performance;
DROP VIEW IF EXISTS v_index_effectiveness;
DROP VIEW IF EXISTS v_connection_pool_metrics;

-- Drop tables
DROP TABLE IF EXISTS db_metrics_history CASCADE;

COMMIT;
