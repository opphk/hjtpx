-- CaptchaX Database Schema Migration
-- Version: 001_initial_schema
-- Description: Initial database schema for CaptchaX verification system
-- Created: 2026-05-14

-- Create extension for UUID generation if not exists
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- captcha_logs: Store captcha verification logs
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

CREATE INDEX idx_captcha_logs_client_id ON captcha_logs(client_id);
CREATE INDEX idx_captcha_logs_ip ON captcha_logs(ip);
CREATE INDEX idx_captcha_logs_created_at ON captcha_logs(created_at);
CREATE INDEX idx_captcha_logs_type_result ON captcha_logs(captcha_type, result);

-- captcha_config: Store system configuration
CREATE TABLE IF NOT EXISTS captcha_config (
    id SERIAL PRIMARY KEY,
    key VARCHAR(100) NOT NULL UNIQUE,
    value TEXT NOT NULL,
    description TEXT,
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_captcha_config_key ON captcha_config(key);

-- Insert default configurations
INSERT INTO captcha_config (key, value, description) VALUES
    ('max_attempts_per_ip', '10', 'Maximum verification attempts per IP per hour'),
    ('block_duration_minutes', '30', 'Duration to block IP after failed attempts'),
    ('risk_threshold', '70', 'Risk score threshold for blocking'),
    ('session_timeout_seconds', '300', 'Session timeout in seconds'),
    ('enable_whitelist', 'true', 'Enable IP whitelist check'),
    ('enable_blacklist', 'true', 'Enable IP blacklist check')
ON CONFLICT (key) DO NOTHING;

-- whitelist: IP whitelist for trusted domains
CREATE TABLE IF NOT EXISTS whitelist (
    id SERIAL PRIMARY KEY,
    ip VARCHAR(45) NOT NULL,
    domain VARCHAR(255),
    reason TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(ip, domain)
);

CREATE INDEX idx_whitelist_ip ON whitelist(ip);
CREATE INDEX idx_whitelist_domain ON whitelist(domain);

-- blacklist: IP blacklist for blocked users
CREATE TABLE IF NOT EXISTS blacklist (
    id SERIAL PRIMARY KEY,
    ip VARCHAR(45) NOT NULL,
    reason TEXT,
    expire_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_blacklist_ip ON blacklist(ip);
CREATE INDEX idx_blacklist_expire ON blacklist(expire_at);

-- admins: System administrators
CREATE TABLE IF NOT EXISTS admins (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'viewer' CHECK (role IN ('admin', 'operator', 'viewer')),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_admins_username ON admins(username);

-- Insert default admin (password: admin123)
-- bcrypt hash of 'admin123'
INSERT INTO admins (username, password_hash, role) VALUES
    ('admin', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZRGdjGj/n3.rKnKh6.3d5EqJq3p4i', 'admin')
ON CONFLICT (username) DO NOTHING;

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger for captcha_config updated_at
CREATE TRIGGER update_captcha_config_updated_at
    BEFORE UPDATE ON captcha_config
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Function to clean up expired blacklist entries
CREATE OR REPLACE FUNCTION cleanup_expired_blacklist()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM blacklist WHERE expire_at IS NOT NULL AND expire_at < NOW();
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Create migrations tracking table
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(50) PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT NOW()
);

-- Record this migration
INSERT INTO schema_migrations (version) VALUES ('001_initial_schema')
ON CONFLICT (version) DO NOTHING;
