-- Migration: Create Sessions and Login History
-- Created: 2026-05-14
-- Description: Creates enhanced sessions table and login history for improved session management

-- Create login_history table
CREATE TABLE IF NOT EXISTS login_history (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  action VARCHAR(50) NOT NULL,
  ip_address INET,
  user_agent TEXT,
  device_info JSONB,
  success BOOLEAN DEFAULT true,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create login_history index for user_id lookups
CREATE INDEX IF NOT EXISTS idx_login_history_user_id ON login_history(user_id);

-- Create login_history index for time-based queries
CREATE INDEX IF NOT EXISTS idx_login_history_created_at ON login_history(created_at DESC);

-- Create login_history index for action type queries
CREATE INDEX IF NOT EXISTS idx_login_history_action ON login_history(action);

-- Create login_history index for success/failure queries
CREATE INDEX IF NOT EXISTS idx_login_history_success ON login_history(success);

-- Add new columns to sessions table if they don't exist
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sessions' AND column_name = 'device_info') THEN
    ALTER TABLE sessions ADD COLUMN device_info JSONB;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sessions' AND column_name = 'ip_address') THEN
    ALTER TABLE sessions ADD COLUMN ip_address INET;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sessions' AND column_name = 'user_agent') THEN
    ALTER TABLE sessions ADD COLUMN user_agent TEXT;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sessions' AND column_name = 'is_revoked') THEN
    ALTER TABLE sessions ADD COLUMN is_revoked BOOLEAN DEFAULT false;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sessions' AND column_name = 'last_activity') THEN
    ALTER TABLE sessions ADD COLUMN last_activity TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sessions' AND column_name = 'is_current') THEN
    ALTER TABLE sessions ADD COLUMN is_current BOOLEAN DEFAULT false;
  END IF;
END $$;

-- Create sessions index for revoked status
CREATE INDEX IF NOT EXISTS idx_sessions_is_revoked ON sessions(is_revoked);

-- Create sessions index for last_activity queries
CREATE INDEX IF NOT EXISTS idx_sessions_last_activity ON sessions(last_activity);

-- Create sessions index for is_current queries
CREATE INDEX IF NOT EXISTS idx_sessions_is_current ON sessions(is_current);

-- Create function to update last_activity timestamp
CREATE OR REPLACE FUNCTION update_last_activity()
RETURNS TRIGGER AS $$
BEGIN
    NEW.last_activity = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger for sessions table (if not exists)
DROP TRIGGER IF EXISTS update_sessions_last_activity ON sessions;
CREATE TRIGGER update_sessions_last_activity
    BEFORE UPDATE ON sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_last_activity();

-- Create function to automatically cleanup expired sessions
CREATE OR REPLACE FUNCTION cleanup_expired_sessions()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM sessions
    WHERE expires_at < CURRENT_TIMESTAMP OR is_revoked = true;

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Add login_attempts table for account lockout
CREATE TABLE IF NOT EXISTS login_attempts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email VARCHAR(255) NOT NULL,
  ip_address INET,
  attempted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  success BOOLEAN DEFAULT false
);

-- Create login_attempts indexes
CREATE INDEX IF NOT EXISTS idx_login_attempts_email ON login_attempts(email);
CREATE INDEX IF NOT EXISTS idx_login_attempts_ip ON login_attempts(ip_address);
CREATE INDEX IF NOT EXISTS idx_login_attempts_time ON login_attempts(attempted_at DESC);

-- Create migrations tracking entry
INSERT INTO migrations (name, applied_at)
VALUES ('005_create_sessions', CURRENT_TIMESTAMP)
ON CONFLICT (name) DO NOTHING;
