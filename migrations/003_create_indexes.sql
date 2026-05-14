-- Migration: Create Additional Indexes
-- Created: 2024-01-03
-- Description: Creates additional indexes for query optimization

-- Composite index for user lookup by email and role
CREATE INDEX IF NOT EXISTS idx_users_email_role ON users(email, role);

-- Composite index for active user queries
CREATE INDEX IF NOT EXISTS idx_users_is_active_role ON users(is_active, role);

-- Index for finding users created after a certain date
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at DESC);

-- Partial index for active sessions
CREATE INDEX IF NOT EXISTS idx_sessions_active ON sessions(user_id, expires_at)
WHERE expires_at > CURRENT_TIMESTAMP;

-- Composite index for session token lookups
CREATE INDEX IF NOT EXISTS idx_sessions_token_expires ON sessions(token, expires_at);

-- Index for cleaning up expired sessions
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at)
WHERE expires_at > CURRENT_TIMESTAMP;

-- Analyze tables to update statistics
ANALYZE users;
ANALYZE sessions;
