-- CaptchaX Database Schema Migration
-- Version: 011_user_profiles
-- Description: User profile system for risk management
-- Created: 2026-05-15

CREATE TABLE IF NOT EXISTS user_profiles (
    id BIGSERIAL PRIMARY KEY,
    identifier VARCHAR(255) NOT NULL,
    identifier_type VARCHAR(20) NOT NULL CHECK (identifier_type IN ('ip', 'device', 'cookie', 'session')),
    ip VARCHAR(45),
    device_fingerprint VARCHAR(255),
    cookie_id VARCHAR(255),
    session_id VARCHAR(255),
    
    total_attempts BIGINT DEFAULT 0,
    success_count BIGINT DEFAULT 0,
    fail_count BIGINT DEFAULT 0,
    success_rate DECIMAL(5,2) DEFAULT 0,
    
    avg_response_time DECIMAL(10,2) DEFAULT 0,
    min_response_time DECIMAL(10,2) DEFAULT 0,
    max_response_time DECIMAL(10,2) DEFAULT 0,
    
    preferred_captcha_type VARCHAR(50),
    captcha_type_distribution JSONB DEFAULT '{}',
    
    active_hours JSONB DEFAULT '{}',
    active_days JSONB DEFAULT '{}',
    
    location_distribution JSONB DEFAULT '{}',
    device_distribution JSONB DEFAULT '{}',
    
    total_risk_events BIGINT DEFAULT 0,
    high_risk_events BIGINT DEFAULT 0,
    last_risk_event_at TIMESTAMP,
    
    first_seen_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    UNIQUE(identifier, identifier_type)
);

CREATE INDEX idx_user_profiles_identifier ON user_profiles(identifier);
CREATE INDEX idx_user_profiles_identifier_type ON user_profiles(identifier_type);
CREATE INDEX idx_user_profiles_ip ON user_profiles(ip);
CREATE INDEX idx_user_profiles_device_fingerprint ON user_profiles(device_fingerprint);
CREATE INDEX idx_user_profiles_success_rate ON user_profiles(success_rate);
CREATE INDEX idx_user_profiles_total_risk_events ON user_profiles(total_risk_events);
CREATE INDEX idx_user_profiles_high_risk_events ON user_profiles(high_risk_events);
CREATE INDEX idx_user_profiles_first_seen_at ON user_profiles(first_seen_at);
CREATE INDEX idx_user_profiles_last_seen_at ON user_profiles(last_seen_at);
CREATE INDEX idx_user_profiles_created_at ON user_profiles(created_at);

CREATE OR REPLACE FUNCTION update_user_profile_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_user_profile_updated_at
    BEFORE UPDATE ON user_profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_user_profile_timestamp();

CREATE OR REPLACE FUNCTION calculate_profile_success_rate()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.total_attempts > 0 THEN
        NEW.success_rate = (NEW.success_count::DECIMAL / NEW.total_attempts::DECIMAL) * 100;
    ELSE
        NEW.success_rate = 0;
    END IF;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER calculate_user_profile_success_rate
    BEFORE INSERT OR UPDATE ON user_profiles
    FOR EACH ROW
    EXECUTE FUNCTION calculate_profile_success_rate();

CREATE OR REPLACE FUNCTION increment_profile_attempts(
    p_identifier VARCHAR,
    p_identifier_type VARCHAR,
    p_success BOOLEAN,
    p_response_time BIGINT
)
RETURNS VOID AS $$
BEGIN
    UPDATE user_profiles SET
        total_attempts = total_attempts + 1,
        success_count = CASE WHEN p_success THEN success_count + 1 ELSE success_count END,
        fail_count = CASE WHEN NOT p_success THEN fail_count + 1 ELSE fail_count END,
        avg_response_time = CASE 
            WHEN total_attempts = 0 THEN p_response_time 
            ELSE (avg_response_time * total_attempts + p_response_time) / (total_attempts + 1)
        END,
        min_response_time = CASE 
            WHEN total_attempts = 0 THEN p_response_time 
            ELSE LEAST(min_response_time, p_response_time)
        END,
        max_response_time = CASE 
            WHEN total_attempts = 0 THEN p_response_time 
            ELSE GREATEST(max_response_time, p_response_time)
        END,
        last_seen_at = NOW(),
        updated_at = NOW()
    WHERE identifier = p_identifier AND identifier_type = p_identifier_type;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION add_profile_risk_event(
    p_identifier VARCHAR,
    p_identifier_type VARCHAR,
    p_high_risk BOOLEAN DEFAULT FALSE
)
RETURNS VOID AS $$
BEGIN
    UPDATE user_profiles SET
        total_risk_events = total_risk_events + 1,
        high_risk_events = CASE WHEN p_high_risk THEN high_risk_events + 1 ELSE high_risk_events END,
        last_risk_event_at = NOW(),
        updated_at = NOW()
    WHERE identifier = p_identifier AND identifier_type = p_identifier_type;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE VIEW v_user_profile_stats AS
SELECT 
    COUNT(*) as total_profiles,
    COUNT(*) FILTER (WHERE success_rate >= 90) as trusted_profiles,
    COUNT(*) FILTER (WHERE success_rate < 50) as high_risk_profiles,
    COUNT(*) FILTER (WHERE success_rate >= 50 AND success_rate < 90) as suspicious_profiles,
    COALESCE(SUM(total_attempts), 0) as total_verifications,
    COALESCE(AVG(success_rate), 0) as avg_success_rate,
    COALESCE(AVG(avg_response_time), 0) as avg_response_time
FROM user_profiles;

CREATE OR REPLACE VIEW v_high_risk_profiles AS
SELECT 
    id,
    identifier,
    identifier_type,
    ip,
    success_rate,
    total_attempts,
    high_risk_events,
    total_risk_events,
    last_risk_event_at,
    last_seen_at
FROM user_profiles
WHERE (high_risk_events * 1.0 / NULLIF(total_risk_events, 0) > 0.3)
   OR (success_rate < 50 AND total_attempts >= 10)
ORDER BY high_risk_events DESC, success_rate ASC;

CREATE OR REPLACE VIEW v_trusted_profiles AS
SELECT 
    id,
    identifier,
    identifier_type,
    ip,
    success_rate,
    total_attempts,
    last_seen_at
FROM user_profiles
WHERE success_rate >= 90 AND (total_risk_events = 0 OR high_risk_events * 1.0 / NULLIF(total_risk_events, 0) < 0.1)
ORDER BY success_rate DESC, total_attempts DESC;

CREATE OR REPLACE VIEW v_active_hours_distribution AS
SELECT 
    hour,
    COUNT(*) as profile_count,
    SUM(total_attempts) as total_attempts
FROM user_profiles,
     jsonb_each_text(active_hours)
GROUP BY hour
ORDER BY hour;

CREATE OR REPLACE VIEW v_captcha_type_popularity AS
SELECT 
    captcha_type,
    COUNT(*) as profile_count,
    SUM((value)::BIGINT) as total_attempts
FROM user_profiles,
     jsonb_each_text(captcha_type_distribution)
GROUP BY captcha_type
ORDER BY total_attempts DESC;

CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(50) PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT NOW()
);

INSERT INTO schema_migrations (version) VALUES ('011_user_profiles')
ON CONFLICT (version) DO NOTHING;
