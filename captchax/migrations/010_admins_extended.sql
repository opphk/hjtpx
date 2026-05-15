-- CaptchaX Database Schema Migration
-- Version: 010_admins_extended
-- Description: Add extended fields to admins table for enhanced admin management
-- Created: 2026-05-15

BEGIN;

-- Add extended fields to admins table
ALTER TABLE admins ADD COLUMN IF NOT EXISTS email VARCHAR(100);
ALTER TABLE admins ADD COLUMN IF NOT EXISTS nickname VARCHAR(100);
ALTER TABLE admins ADD COLUMN IF NOT EXISTS phone VARCHAR(20);
ALTER TABLE admins ADD COLUMN IF NOT EXISTS avatar VARCHAR(255);
ALTER TABLE admins ADD COLUMN IF NOT EXISTS status INTEGER DEFAULT 1 CHECK (status IN (0, 1, 2));
ALTER TABLE admins ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMP;
ALTER TABLE admins ADD COLUMN IF NOT EXISTS last_login_ip VARCHAR(45);
ALTER TABLE admins ADD COLUMN IF NOT EXISTS login_count INTEGER DEFAULT 0;
ALTER TABLE admins ADD COLUMN IF NOT EXISTS department VARCHAR(100);
ALTER TABLE admins ADD COLUMN IF NOT EXISTS notes TEXT;

-- Update existing admin record with email
UPDATE admins SET email = 'admin@example.com' WHERE username = 'admin' AND email IS NULL;

-- Create indexes for new fields
CREATE INDEX IF NOT EXISTS idx_admins_email ON admins(email);
CREATE INDEX IF NOT EXISTS idx_admins_status ON admins(status);
CREATE INDEX IF NOT EXISTS idx_admins_created_at ON admins(created_at);

-- admin_operation_logs: Admin operation audit logs
CREATE TABLE IF NOT EXISTS admin_operation_logs (
    id SERIAL PRIMARY KEY,
    admin_id INTEGER REFERENCES admins(id) ON DELETE SET NULL,
    username VARCHAR(50) NOT NULL,
    action VARCHAR(50) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id VARCHAR(100),
    details JSONB,
    ip VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_admin_operation_logs_admin ON admin_operation_logs(admin_id);
CREATE INDEX idx_admin_operation_logs_action ON admin_operation_logs(action);
CREATE INDEX idx_admin_operation_logs_resource ON admin_operation_logs(resource_type, resource_id);
CREATE INDEX idx_admin_operation_logs_created ON admin_operation_logs(created_at);

-- admin_sessions: Admin session management for tracking active sessions
CREATE TABLE IF NOT EXISTS admin_sessions (
    id SERIAL PRIMARY KEY,
    admin_id INTEGER NOT NULL REFERENCES admins(id) ON DELETE CASCADE,
    session_token VARCHAR(255) NOT NULL UNIQUE,
    ip_address VARCHAR(45),
    user_agent TEXT,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_admin_sessions_token ON admin_sessions(session_token);
CREATE INDEX idx_admin_sessions_admin ON admin_sessions(admin_id);
CREATE INDEX idx_admin_sessions_expires ON admin_sessions(expires_at);

-- Create function to clean up expired sessions
CREATE OR REPLACE FUNCTION cleanup_expired_admin_sessions()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM admin_sessions WHERE expires_at < NOW();
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Record this migration
INSERT INTO schema_migrations (version) VALUES ('010_admins_extended')
ON CONFLICT (version) DO NOTHING;

COMMIT;
