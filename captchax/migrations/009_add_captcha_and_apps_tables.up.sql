-- CaptchaX Add Captcha and Apps Tables
-- Version: 009_add_captcha_and_apps_tables
-- Description: Add captcha instances and apps management tables
-- Created: 2026-05-15

BEGIN;

-- 1. Create captcha table (for storing captcha instances)
CREATE TABLE IF NOT EXISTS captcha (
    id VARCHAR(36) PRIMARY KEY,
    app_id VARCHAR(64) NOT NULL,
    type VARCHAR(20) NOT NULL,
    answer VARCHAR(128) NOT NULL,
    image_data TEXT NOT NULL,
    status INTEGER DEFAULT 0,
    attempts INTEGER DEFAULT 0,
    client_info VARCHAR(512),
    user_agent VARCHAR(512),
    ip_address VARCHAR(45),
    expired_at TIMESTAMP NOT NULL,
    verified_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_captcha_app_id ON captcha(app_id);
CREATE INDEX idx_captcha_expired_at ON captcha(expired_at);
CREATE INDEX idx_captcha_status ON captcha(status);
CREATE INDEX idx_captcha_created_at ON captcha(created_at);
CREATE INDEX idx_captcha_deleted_at ON captcha(deleted_at);

-- 2. Create apps table (for managing applications)
CREATE TABLE IF NOT EXISTS apps (
    id SERIAL PRIMARY KEY,
    app_id VARCHAR(64) UNIQUE NOT NULL,
    app_secret VARCHAR(128) NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    owner_id INTEGER NOT NULL,
    status INTEGER DEFAULT 1,
    domain VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_apps_app_id ON apps(app_id);
CREATE INDEX idx_apps_owner_id ON apps(owner_id);
CREATE INDEX idx_apps_status ON apps(status);
CREATE INDEX idx_apps_deleted_at ON apps(deleted_at);

-- 3. Create foreign key constraints
ALTER TABLE apps 
    ADD CONSTRAINT fk_apps_owner 
    FOREIGN KEY (owner_id) 
    REFERENCES admins(id)
    ON DELETE CASCADE;

-- 4. Add updated_at trigger for captcha and apps tables
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_captcha_updated_at
    BEFORE UPDATE ON captcha
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_apps_updated_at
    BEFORE UPDATE ON apps
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 5. Add captcha status check constraint
ALTER TABLE captcha 
    ADD CONSTRAINT chk_captcha_status 
    CHECK (status IN (0, 1, 2, 3));

-- 6. Add captcha type check constraint
ALTER TABLE captcha 
    ADD CONSTRAINT chk_captcha_type 
    CHECK (type IN ('image', 'slider', 'rotate', 'click', 'puzzle', 'text', 'icon'));

-- 7. Insert default test app
INSERT INTO apps (app_id, app_secret, name, description, owner_id, status, domain) 
VALUES (
    'test_app_001',
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZRGdjGj/n3.rKnKh6.3d5EqJq3p4i', -- bcrypt of 'test_secret_123'
    'Test Application',
    'Default test application for CaptchaX',
    1, -- admin user
    1,
    'localhost'
)
ON CONFLICT (app_id) DO NOTHING;

-- 8. Record migration
INSERT INTO schema_migrations (version) VALUES ('009_add_captcha_and_apps_tables')
ON CONFLICT (version) DO NOTHING;

COMMIT;
