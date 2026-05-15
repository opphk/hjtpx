-- CaptchaX Database Schema Migration
-- Version: 013_audit_logs
-- Description: Audit log system for tracking admin operations
-- Created: 2026-05-15

CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT DEFAULT 0,
    username VARCHAR(255) NOT NULL DEFAULT '',
    action VARCHAR(100) NOT NULL,
    detail TEXT DEFAULT '',
    ip_address VARCHAR(45) DEFAULT '',
    user_agent TEXT DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_username ON audit_logs(username);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
CREATE INDEX idx_audit_logs_user_action ON audit_logs(user_id, action);
CREATE INDEX idx_audit_logs_ip_address ON audit_logs(ip_address);

INSERT INTO schema_migrations (version) VALUES ('013_audit_logs')
ON CONFLICT (version) DO NOTHING;