-- Migration: Analytics System
-- Created: 2026-05-14
-- Description: Creates tables for user behavior tracking, event analytics, and performance metrics

-- Create user_events table for tracking user activities
CREATE TABLE IF NOT EXISTS user_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  event_type VARCHAR(100) NOT NULL,
  event_data JSONB DEFAULT '{}',
  ip_address INET,
  user_agent TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for user_events
CREATE INDEX IF NOT EXISTS idx_user_events_user_id ON user_events(user_id);
CREATE INDEX IF NOT EXISTS idx_user_events_event_type ON user_events(event_type);
CREATE INDEX IF NOT EXISTS idx_user_events_created_at ON user_events(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_events_user_type ON user_events(user_id, event_type);
CREATE INDEX IF NOT EXISTS idx_user_events_date ON user_events(DATE(created_at));

-- Create api_performance table for tracking API response times
CREATE TABLE IF NOT EXISTS api_performance (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  endpoint VARCHAR(500) NOT NULL,
  method VARCHAR(10) NOT NULL,
  duration_ms INTEGER NOT NULL,
  status_code INTEGER,
  user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  ip_address INET,
  request_size INTEGER,
  response_size INTEGER,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for api_performance
CREATE INDEX IF NOT EXISTS idx_api_perf_endpoint ON api_performance(endpoint);
CREATE INDEX IF NOT EXISTS idx_api_perf_method ON api_performance(method);
CREATE INDEX IF NOT EXISTS idx_api_perf_created_at ON api_performance(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_api_perf_status ON api_performance(status_code);
CREATE INDEX IF NOT EXISTS idx_api_perf_duration ON api_performance(duration_ms);
CREATE INDEX IF NOT EXISTS idx_api_perf_endpoint_method ON api_performance(endpoint, method);
CREATE INDEX IF NOT EXISTS idx_api_perf_user ON api_performance(user_id);

-- Create analytics_daily_summary table for aggregated daily statistics
CREATE TABLE IF NOT EXISTS analytics_daily_summary (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  date DATE NOT NULL UNIQUE,
  total_users INTEGER DEFAULT 0,
  active_users INTEGER DEFAULT 0,
  new_users INTEGER DEFAULT 0,
  total_events INTEGER DEFAULT 0,
  total_api_calls INTEGER DEFAULT 0,
  avg_response_time_ms INTEGER DEFAULT 0,
  error_count INTEGER DEFAULT 0,
  cache_hit_rate DECIMAL(5,2) DEFAULT 0,
  peak_concurrent_users INTEGER DEFAULT 0,
  metadata JSONB DEFAULT '{}',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for analytics_daily_summary
CREATE INDEX IF NOT EXISTS idx_analytics_daily_date ON analytics_daily_summary(date DESC);
CREATE INDEX IF NOT EXISTS idx_analytics_daily_users ON analytics_daily_summary(active_users DESC);

-- Create feature_usage table for tracking feature usage
CREATE TABLE IF NOT EXISTS feature_usage (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  feature_name VARCHAR(100) NOT NULL,
  action VARCHAR(50) NOT NULL,
  user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  usage_count INTEGER DEFAULT 1,
  metadata JSONB DEFAULT '{}',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(feature_name, action, user_id, created_at::DATE)
);

-- Create indexes for feature_usage
CREATE INDEX IF NOT EXISTS idx_feature_usage_name ON feature_usage(feature_name);
CREATE INDEX IF NOT EXISTS idx_feature_usage_action ON feature_usage(action);
CREATE INDEX IF NOT EXISTS idx_feature_usage_user ON feature_usage(user_id);
CREATE INDEX IF NOT EXISTS idx_feature_usage_created ON feature_usage(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_feature_usage_daily ON feature_usage(feature_name, action, created_at::DATE);

-- Create page_views table for tracking page views
CREATE TABLE IF NOT EXISTS page_views (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  page_path VARCHAR(500) NOT NULL,
  referrer TEXT,
  session_id VARCHAR(100),
  time_on_page INTEGER,
  scroll_depth INTEGER,
  device_type VARCHAR(20),
  browser VARCHAR(50),
  os VARCHAR(50),
  metadata JSONB DEFAULT '{}',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for page_views
CREATE INDEX IF NOT EXISTS idx_page_views_user ON page_views(user_id);
CREATE INDEX IF NOT EXISTS idx_page_views_page ON page_views(page_path);
CREATE INDEX IF NOT EXISTS idx_page_views_created ON page_views(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_page_views_session ON page_views(session_id);
CREATE INDEX IF NOT EXISTS idx_page_views_device ON page_views(device_type);

-- Create anomalies table for storing detected anomalies
CREATE TABLE IF NOT EXISTS anomalies (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  anomaly_type VARCHAR(50) NOT NULL,
  severity VARCHAR(20) NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
  description TEXT,
  affected_endpoint VARCHAR(500),
  affected_users UUID[],
  metric_value DECIMAL,
  threshold_value DECIMAL,
  metadata JSONB DEFAULT '{}',
  detected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  resolved_at TIMESTAMP,
  resolved_by UUID REFERENCES users(id) ON DELETE SET NULL
);

-- Create indexes for anomalies
CREATE INDEX IF NOT EXISTS idx_anomalies_type ON anomalies(anomaly_type);
CREATE INDEX IF NOT EXISTS idx_anomalies_severity ON anomalies(severity);
CREATE INDEX IF NOT EXISTS idx_anomalies_detected ON anomalies(detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_anomalies_resolved ON anomalies(resolved_at) WHERE resolved_at IS NULL;

-- Create user_engagement table for tracking user engagement metrics
CREATE TABLE IF NOT EXISTS user_engagement (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  session_id VARCHAR(100) NOT NULL,
  session_start TIMESTAMP NOT NULL,
  session_end TIMESTAMP,
  page_views INTEGER DEFAULT 0,
  clicks INTEGER DEFAULT 0,
  features_used INTEGER DEFAULT 0,
  duration_seconds INTEGER,
  device_type VARCHAR(20),
  engagement_score DECIMAL(5,2) DEFAULT 0,
  metadata JSONB DEFAULT '{}',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for user_engagement
CREATE INDEX IF NOT EXISTS idx_user_engagement_user ON user_engagement(user_id);
CREATE INDEX IF NOT EXISTS idx_user_engagement_session ON user_engagement(session_id);
CREATE INDEX IF NOT EXISTS idx_user_engagement_created ON user_engagement(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_engagement_score ON user_engagement(engagement_score DESC);

-- Create triggers for timestamp updates
DROP TRIGGER IF EXISTS update_analytics_summary_updated_at ON analytics_daily_summary;
CREATE TRIGGER update_analytics_summary_updated_at
    BEFORE UPDATE ON analytics_daily_summary
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_feature_usage_updated_at ON feature_usage;
CREATE TRIGGER update_feature_usage_updated_at
    BEFORE UPDATE ON feature_usage
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create view for real-time active users
CREATE OR REPLACE VIEW real_time_active_users AS
SELECT 
  ue.user_id,
  u.username,
  u.email,
  COUNT(DISTINCT ue.event_type) as unique_actions,
  COUNT(*) as total_events,
  MAX(ue.created_at) as last_activity
FROM user_events ue
LEFT JOIN users u ON ue.user_id = u.id
WHERE ue.created_at >= NOW() - INTERVAL '15 minutes'
GROUP BY ue.user_id, u.username, u.email;

-- Create view for feature popularity
CREATE OR REPLACE VIEW feature_popularity AS
SELECT 
  feature_name,
  action,
  COUNT(*) as usage_count,
  COUNT(DISTINCT user_id) as unique_users,
  AVG(usage_count) as avg_uses_per_session
FROM feature_usage
WHERE created_at >= NOW() - INTERVAL '30 days'
GROUP BY feature_name, action
ORDER BY usage_count DESC;

-- Create view for error rate by endpoint
CREATE OR REPLACE VIEW error_rate_by_endpoint AS
SELECT 
  endpoint,
  method,
  COUNT(*) as total_requests,
  COUNT(*) FILTER (WHERE status_code >= 400) as errors,
  ROUND((COUNT(*) FILTER (WHERE status_code >= 400)::NUMERIC / NULLIF(COUNT(*), 0)) * 100, 2) as error_rate
FROM api_performance
WHERE created_at >= NOW() - INTERVAL '24 hours'
GROUP BY endpoint, method
HAVING COUNT(*) FILTER (WHERE status_code >= 400) > 0
ORDER BY error_rate DESC;
