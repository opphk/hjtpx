-- Rollback for: 007_create_analytics.sql
-- Generated: 2026-05-14
-- This script reverts the analytics tables creation

BEGIN;

-- Drop user_engagement indexes
DROP INDEX IF EXISTS idx_user_engagement_score;
DROP INDEX IF EXISTS idx_user_engagement_created;
DROP INDEX IF EXISTS idx_user_engagement_session;
DROP INDEX IF EXISTS idx_user_engagement_user;

-- Drop user_engagement table
DROP TABLE IF EXISTS user_engagement CASCADE;

-- Drop anomalies indexes
DROP INDEX IF EXISTS idx_anomalies_resolved;
DROP INDEX IF EXISTS idx_anomalies_detected;
DROP INDEX IF EXISTS idx_anomalies_severity;
DROP INDEX IF EXISTS idx_anomalies_type;

-- Drop anomalies table
DROP TABLE IF EXISTS anomalies CASCADE;

-- Drop page_views indexes
DROP INDEX IF EXISTS idx_page_views_device;
DROP INDEX IF EXISTS idx_page_views_session;
DROP INDEX IF EXISTS idx_page_views_created;
DROP INDEX IF EXISTS idx_page_views_page;
DROP INDEX IF EXISTS idx_page_views_user;

-- Drop page_views table
DROP TABLE IF EXISTS page_views CASCADE;

-- Drop feature_usage indexes
DROP INDEX IF EXISTS idx_feature_usage_daily;
DROP INDEX IF EXISTS idx_feature_usage_created;
DROP INDEX IF EXISTS idx_feature_usage_user;
DROP INDEX IF EXISTS idx_feature_usage_action;
DROP INDEX IF EXISTS idx_feature_usage_name;

-- Drop feature_usage table
DROP TABLE IF EXISTS feature_usage CASCADE;

-- Drop trigger for feature_usage
DROP TRIGGER IF EXISTS update_feature_usage_updated_at ON feature_usage;

-- Drop analytics_daily_summary indexes
DROP INDEX IF EXISTS idx_analytics_daily_users;
DROP INDEX IF EXISTS idx_analytics_daily_date;

-- Drop analytics_daily_summary table
DROP TABLE IF EXISTS analytics_daily_summary CASCADE;

-- Drop trigger for analytics_daily_summary
DROP TRIGGER IF EXISTS update_analytics_summary_updated_at ON analytics_daily_summary;

-- Drop function to update analytics timestamp
DROP FUNCTION IF EXISTS update_analytics_timestamp();

-- Drop api_performance indexes
DROP INDEX IF EXISTS idx_api_perf_user;
DROP INDEX IF EXISTS idx_api_perf_endpoint_method;
DROP INDEX IF EXISTS idx_api_perf_duration;
DROP INDEX IF EXISTS idx_api_perf_status;
DROP INDEX IF EXISTS idx_api_perf_created_at;
DROP INDEX IF EXISTS idx_api_perf_method;
DROP INDEX IF EXISTS idx_api_perf_endpoint;

-- Drop api_performance table
DROP TABLE IF EXISTS api_performance CASCADE;

-- Drop user_events indexes
DROP INDEX IF EXISTS idx_user_events_date;
DROP INDEX IF EXISTS idx_user_events_user_type;
DROP INDEX IF EXISTS idx_user_events_created_at;
DROP INDEX IF EXISTS idx_user_events_event_type;
DROP INDEX IF EXISTS idx_user_events_user_id;

-- Drop user_events table
DROP TABLE IF EXISTS user_events CASCADE;

-- Drop views
DROP VIEW IF EXISTS error_rate_by_endpoint;
DROP VIEW IF EXISTS feature_popularity;
DROP VIEW IF EXISTS real_time_active_users;

-- Remove migration tracking entry
DELETE FROM migrations WHERE name = '007_create_analytics';

COMMIT;
