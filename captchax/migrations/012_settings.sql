-- CaptchaX Database Schema Migration
-- Version: 012_settings
-- Description: System settings table for admin management
-- Created: 2026-05-15

CREATE TABLE IF NOT EXISTS system_settings (
    id BIGSERIAL PRIMARY KEY,
    site_name VARCHAR(255) DEFAULT 'CaptchaX',
    site_description TEXT DEFAULT 'CaptchaX 验证码管理系统',
    jwt_expiry_hours INTEGER DEFAULT 24,
    min_password_length INTEGER DEFAULT 8,
    captcha_difficulty VARCHAR(20) DEFAULT 'medium' CHECK (captcha_difficulty IN ('easy', 'medium', 'hard')),
    captcha_types TEXT DEFAULT 'slider,click,rotate',
    email_notification BOOLEAN DEFAULT false,
    webhook_url VARCHAR(500) DEFAULT '',
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

INSERT INTO system_settings (site_name, site_description, jwt_expiry_hours, min_password_length, captcha_difficulty, captcha_types, email_notification, webhook_url)
VALUES ('CaptchaX', 'CaptchaX 验证码管理系统', 24, 8, 'medium', 'slider,click,rotate', false, '')
ON CONFLICT DO NOTHING;

CREATE INDEX idx_system_settings_updated_at ON system_settings(updated_at);

INSERT INTO schema_migrations (version) VALUES ('012_settings')
ON CONFLICT (version) DO NOTHING;