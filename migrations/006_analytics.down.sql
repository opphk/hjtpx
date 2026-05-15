-- Rollback: Analytics System
-- Description: Removes analytics tables and related objects

-- Drop views
DROP VIEW IF EXISTS error_rate_by_endpoint;
DROP VIEW IF EXISTS feature_popularity;
DROP VIEW IF EXISTS real_time_active_users;

-- Drop triggers
DROP TRIGGER IF EXISTS update_analytics_summary_updated_at ON analytics_daily_summary;
DROP TRIGGER IF EXISTS update_feature_usage_updated_at ON feature_usage;

-- Drop indexes
DROP INDEX IF EXISTS idx_user_events_user_id;
DROP INDEX IF EXISTS idx_user_events_event_type;
DROP INDEX IF EXISTS idx_user_events_created_at;
DROP INDEX IF EXISTS idx_user_events_user_type;
DROP INDEX IF EXISTS idx_user_events_date;
DROP INDEX IF EXISTS idx_api_perf_endpoint;
DROP INDEX IF EXISTS idx_api_perf_method;
DROP INDEX IF EXISTS idx_api_perf_created_at;
DROP INDEX IF EXISTS idx_api_perf_status;
DROP INDEX IF EXISTS idx_api_perf_duration;
DROP INDEX IF EXISTS idx_api_perf_endpoint_method;
DROP INDEX IF EXISTS idx_api_perf_user;
DROP INDEX IF EXISTS idx_analytics_daily_date;
DROP INDEX IF EXISTS idx_analytics_daily_users;
DROP INDEX IF EXISTS idx_feature_usage_name;
DROP INDEX IF EXISTS idx_feature_usage_action;
DROP INDEX IF EXISTS idx_feature_usage_user;
DROP INDEX IF EXISTS idx_feature_usage_created;
DROP INDEX IF EXISTS idx_feature_usage_daily;
DROP INDEX IF EXISTS idx_page_views_user;
DROP INDEX IF EXISTS idx_page_views_page;
DROP INDEX IF EXISTS idx_page_views_created;
DROP INDEX IF EXISTS idx_page_views_session;
DROP INDEX IF EXISTS idx_page_views_device;
DROP INDEX IF EXISTS idx_anomalies_type;
DROP INDEX IF EXISTS idx_anomalies_severity;
DROP INDEX IF EXISTS idx_anomalies_detected;
DROP INDEX IF EXISTS idx_anomalies_resolved;
DROP INDEX IF EXISTS idx_user_engagement_user;
DROP INDEX IF EXISTS idx_user_engagement_session;
DROP INDEX IF EXISTS idx_user_engagement_created;
DROP INDEX IF EXISTS idx_user_engagement_score;

-- Drop tables (order matters due to foreign keys)
DROP TABLE IF EXISTS user_engagement;
DROP TABLE IF EXISTS anomalies;
DROP TABLE IF EXISTS page_views;
DROP TABLE IF EXISTS feature_usage;
DROP TABLE IF EXISTS analytics_daily_summary;
DROP TABLE IF EXISTS api_performance;
DROP TABLE IF EXISTS user_events;
