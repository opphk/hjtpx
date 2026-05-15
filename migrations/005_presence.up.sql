-- Migration: Presence and Online Status
-- Created: 2026-05-14
-- Description: Creates presence table for tracking real-time user online status

-- Create presence table
CREATE TABLE IF NOT EXISTS presence (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  socket_id VARCHAR(255),
  status VARCHAR(20) DEFAULT 'online' CHECK (status IN ('online', 'away', 'busy', 'offline')),
  last_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  ip_address INET,
  user_agent TEXT,
  device_info JSONB,
  current_page VARCHAR(500),
  metadata JSONB DEFAULT '{}',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(user_id, socket_id)
);

-- Create presence indexes
CREATE INDEX IF NOT EXISTS idx_presence_user_id ON presence(user_id);
CREATE INDEX IF NOT EXISTS idx_presence_socket_id ON presence(socket_id);
CREATE INDEX IF NOT EXISTS idx_presence_status ON presence(status);
CREATE INDEX IF NOT EXISTS idx_presence_last_seen ON presence(last_seen_at DESC);
CREATE INDEX IF NOT EXISTS idx_presence_online ON presence(status) WHERE status = 'online';

-- Create presence_history table for tracking user activity
CREATE TABLE IF NOT EXISTS presence_history (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  action VARCHAR(50) NOT NULL,
  status VARCHAR(20) NOT NULL,
  duration_seconds INTEGER,
  ip_address INET,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create presence_history indexes
CREATE INDEX IF NOT EXISTS idx_presence_history_user_id ON presence_history(user_id);
CREATE INDEX IF NOT EXISTS idx_presence_history_action ON presence_history(action);
CREATE INDEX IF NOT EXISTS idx_presence_history_created_at ON presence_history(created_at DESC);

-- Create trigger for presence timestamp updates
DROP TRIGGER IF EXISTS update_presence_updated_at ON presence;
CREATE TRIGGER update_presence_updated_at
    BEFORE UPDATE ON presence
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create function to log presence changes
CREATE OR REPLACE FUNCTION log_presence_change()
RETURNS TRIGGER AS $$
DECLARE
  action_type VARCHAR(50);
  duration INTEGER := NULL;
BEGIN
  IF TG_OP = 'INSERT' THEN
    action_type := 'login';
  ELSIF TG_OP = 'DELETE' THEN
    action_type := 'logout';
    IF OLD.last_seen_at IS NOT NULL THEN
      duration := EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - OLD.last_seen_at))::INTEGER;
    END IF;
  ELSIF TG_OP = 'UPDATE' THEN
    IF OLD.status IS DISTINCT FROM NEW.status THEN
      action_type := 'status_change';
    ELSE
      RETURN NEW;
    END IF;
  ELSE
    RETURN NEW;
  END IF;

  INSERT INTO presence_history (user_id, action, status, duration_seconds, ip_address)
  VALUES (
    COALESCE(NEW.user_id, OLD.user_id),
    action_type,
    COALESCE(NEW.status, OLD.status),
    duration,
    COALESCE(NEW.ip_address, OLD.ip_address)
  );

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for presence changes
DROP TRIGGER IF EXISTS presence_change_trigger ON presence;
CREATE TRIGGER presence_change_trigger
    AFTER INSERT OR UPDATE OR DELETE ON presence
    FOR EACH ROW
    EXECUTE FUNCTION log_presence_change();

-- Create view for active online users
CREATE OR REPLACE VIEW active_users AS
SELECT 
  u.id AS user_id,
  u.username,
  u.email,
  p.status,
  p.last_seen_at,
  p.current_page,
  p.device_info,
  COUNT(p.socket_id) AS connection_count
FROM users u
LEFT JOIN presence p ON u.id = p.user_id AND p.status = 'online'
WHERE u.is_active = true
GROUP BY u.id, u.username, u.email, p.status, p.last_seen_at, p.current_page, p.device_info;
