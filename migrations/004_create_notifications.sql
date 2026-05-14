-- Create notifications table
CREATE TABLE IF NOT EXISTS notifications (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL DEFAULT 'info',
    title VARCHAR(200) NOT NULL,
    message TEXT NOT NULL,
    data JSONB DEFAULT '{}',
    priority VARCHAR(20) DEFAULT 'normal',
    status VARCHAR(20) DEFAULT 'unread',
    read_at TIMESTAMP,
    expires_at TIMESTAMP,
    action_url VARCHAR(500),
    action_label VARCHAR(50),
    channels VARCHAR(50)[] DEFAULT ARRAY['in_app'],
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for notifications
CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_user_status ON notifications(user_id, status);
CREATE INDEX idx_notifications_user_created ON notifications(user_id, created_at DESC);
CREATE INDEX idx_notifications_type ON notifications(type);
CREATE INDEX idx_notifications_expires_at ON notifications(expires_at);
CREATE INDEX idx_notifications_status ON notifications(status);

-- Create index for TTL (auto-delete expired notifications)
CREATE INDEX idx_notifications_ttl ON notifications(expires_at) WHERE expires_at IS NOT NULL;

-- Add comments
COMMENT ON TABLE notifications IS 'User notifications and alerts';
COMMENT ON COLUMN notifications.type IS 'Notification type: info, success, warning, error, system, message, reminder, alert';
COMMENT ON COLUMN notifications.priority IS 'Priority level: low, normal, high, urgent';
COMMENT ON COLUMN notifications.status IS 'Notification status: unread, read, archived';
COMMENT ON COLUMN notifications.channels IS 'Delivery channels: in_app, email, sms, push';
