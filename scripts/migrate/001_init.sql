-- 初始化数据库表结构
-- 创建时间: 2026-05-17

-- 1. 管理员表 (admin_users) - 需首先创建，因为其他表可能引用
CREATE TABLE IF NOT EXISTS admin_users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL CHECK (role IN ('super_admin', 'admin', 'operator', 'viewer')),
    permissions JSONB DEFAULT '[]',
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'locked')),
    last_login_at TIMESTAMP,
    last_login_ip VARCHAR(45),
    failed_attempts INTEGER DEFAULT 0,
    locked_until TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by INTEGER REFERENCES admin_users(id) ON DELETE SET NULL,
    password_changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_admin_username ON admin_users(username);
CREATE INDEX idx_admin_email ON admin_users(email);
CREATE INDEX idx_admin_role ON admin_users(role);
CREATE INDEX idx_admin_status ON admin_users(status);

-- 2. 用户表 (users)
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP,
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'banned')),
    login_count INTEGER DEFAULT 0,
    failed_attempts INTEGER DEFAULT 0,
    locked_until TIMESTAMP,
    metadata JSONB DEFAULT '{}'
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_created_at ON users(created_at);

-- 3. 验证码会话表 (captcha_sessions)
CREATE TABLE IF NOT EXISTS captcha_sessions (
    id SERIAL PRIMARY KEY,
    session_id VARCHAR(64) UNIQUE NOT NULL,
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    captcha_type VARCHAR(20) NOT NULL CHECK (captcha_type IN ('slider', 'point', 'rotate')),
    difficulty VARCHAR(20) DEFAULT 'medium' CHECK (difficulty IN ('easy', 'medium', 'hard')),
    target_position JSONB NOT NULL,
    background_image_url TEXT,
    slider_image_url TEXT,
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'success', 'failed', 'expired')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    verified_at TIMESTAMP,
    verification_data JSONB DEFAULT '{}',
    client_ip VARCHAR(45),
    user_agent TEXT,
    risk_score DECIMAL(5,2) DEFAULT 0
);

CREATE INDEX idx_captcha_session_id ON captcha_sessions(session_id);
CREATE INDEX idx_captcha_user_id ON captcha_sessions(user_id);
CREATE INDEX idx_captcha_status ON captcha_sessions(status);
CREATE INDEX idx_captcha_created_at ON captcha_sessions(created_at);
CREATE INDEX idx_captcha_expires_at ON captcha_sessions(expires_at);

-- 4. 验证轨迹表 (verification_traces)
CREATE TABLE IF NOT EXISTS verification_traces (
    id SERIAL PRIMARY KEY,
    session_id VARCHAR(64) NOT NULL,
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    event_type VARCHAR(50) NOT NULL,
    event_data JSONB DEFAULT '{}',
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    duration_ms INTEGER,
    mouse_trajectory JSONB,
    mouse_velocity JSONB,
    mouse_acceleration JSONB,
    click_positions JSONB,
    scroll_behavior JSONB,
    touch_events JSONB,
    device_fingerprint VARCHAR(128),
    browser_info JSONB,
    screen_resolution VARCHAR(20),
    timezone VARCHAR(50),
    language VARCHAR(10),
    client_ip VARCHAR(45)
);

CREATE INDEX idx_trace_session_id ON verification_traces(session_id);
CREATE INDEX idx_trace_user_id ON verification_traces(user_id);
CREATE INDEX idx_trace_event_type ON verification_traces(event_type);
CREATE INDEX idx_trace_timestamp ON verification_traces(timestamp);
CREATE INDEX idx_trace_client_ip ON verification_traces(client_ip);

-- 5. 风控日志表 (risk_logs)
CREATE TABLE IF NOT EXISTS risk_logs (
    id SERIAL PRIMARY KEY,
    session_id VARCHAR(64),
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    risk_type VARCHAR(50) NOT NULL,
    risk_level VARCHAR(20) NOT NULL CHECK (risk_level IN ('low', 'medium', 'high', 'critical')),
    risk_score DECIMAL(5,2) NOT NULL,
    risk_factors JSONB DEFAULT '[]',
    decision VARCHAR(20) NOT NULL CHECK (decision IN ('allow', 'block', 'challenge', 'review')),
    action_taken TEXT,
    description TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    client_ip VARCHAR(45),
    user_agent TEXT,
    resolved BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMP,
    resolved_by INTEGER REFERENCES admin_users(id) ON DELETE SET NULL,
    resolution_notes TEXT
);

CREATE INDEX idx_risk_session_id ON risk_logs(session_id);
CREATE INDEX idx_risk_user_id ON risk_logs(user_id);
CREATE INDEX idx_risk_type ON risk_logs(risk_type);
CREATE INDEX idx_risk_level ON risk_logs(risk_level);
CREATE INDEX idx_risk_decision ON risk_logs(decision);
CREATE INDEX idx_risk_created_at ON risk_logs(created_at);
CREATE INDEX idx_risk_client_ip ON risk_logs(client_ip);
CREATE INDEX idx_risk_resolved ON risk_logs(resolved);

-- 6. 系统配置表 (system_configs)
CREATE TABLE IF NOT EXISTS system_configs (
    id SERIAL PRIMARY KEY,
    config_key VARCHAR(100) UNIQUE NOT NULL,
    config_value TEXT NOT NULL,
    config_type VARCHAR(20) NOT NULL CHECK (config_type IN ('string', 'number', 'boolean', 'json', 'array')),
    description TEXT,
    category VARCHAR(50) DEFAULT 'general',
    is_sensitive BOOLEAN DEFAULT FALSE,
    is_public BOOLEAN DEFAULT FALSE,
    validation_rules JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_by INTEGER REFERENCES admin_users(id) ON DELETE SET NULL
);

CREATE INDEX idx_config_key ON system_configs(config_key);
CREATE INDEX idx_config_category ON system_configs(category);
CREATE INDEX idx_config_type ON system_configs(config_type);

-- 插入默认配置
INSERT INTO system_configs (config_key, config_value, config_type, description, category) VALUES
('captcha.default_difficulty', 'medium', 'string', '默认验证码难度', 'captcha'),
('captcha.max_attempts', '3', 'number', '最大验证尝试次数', 'captcha'),
('captcha.expiration_seconds', '300', 'number', '验证码过期时间(秒)', 'captcha'),
('captcha.slider_width', '50', 'number', '滑块宽度(像素)', 'captcha'),
('captcha.slider_height', '50', 'number', '滑块高度(像素)', 'captcha'),
('captcha.image_width', '300', 'number', '验证码图片宽度', 'captcha'),
('captcha.image_height', '150', 'number', '验证码图片高度', 'captcha'),
('risk.score_threshold_low', '30', 'number', '低风险分数阈值', 'risk'),
('risk.score_threshold_medium', '60', 'number', '中风险分数阈值', 'risk'),
('risk.score_threshold_high', '80', 'number', '高风险分数阈值', 'risk'),
('risk.max_requests_per_minute', '60', 'number', '每分钟最大请求数', 'risk'),
('session.timeout_minutes', '30', 'number', '会话超时时间(分钟)', 'session'),
('security.password_min_length', '8', 'number', '密码最小长度', 'security'),
('security.enable_captcha', 'true', 'boolean', '是否启用验证码', 'security'),
('security.allow_multiple_sessions', 'false', 'boolean', '是否允许多会话', 'security')
ON CONFLICT (config_key) DO NOTHING;

-- 7. 黑名单表 (blacklist)
CREATE TABLE IF NOT EXISTS blacklist (
    id SERIAL PRIMARY KEY,
    blacklist_type VARCHAR(30) NOT NULL CHECK (blacklist_type IN ('ip', 'user', 'device', 'session')),
    blacklisted_value VARCHAR(255) NOT NULL,
    reason TEXT,
    severity VARCHAR(20) DEFAULT 'medium' CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by INTEGER REFERENCES admin_users(id) ON DELETE SET NULL,
    is_active BOOLEAN DEFAULT TRUE,
    hit_count INTEGER DEFAULT 0,
    last_hit_at TIMESTAMP,
    metadata JSONB DEFAULT '{}',
    UNIQUE(blacklist_type, blacklisted_value)
);

CREATE INDEX idx_blacklist_type ON blacklist(blacklist_type);
CREATE INDEX idx_blacklist_value ON blacklist(blacklisted_value);
CREATE INDEX idx_blacklist_active ON blacklist(is_active);
CREATE INDEX idx_blacklist_expires_at ON blacklist(expires_at);
CREATE INDEX idx_blacklist_created_at ON blacklist(created_at);

-- 创建更新时间戳的函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 为需要自动更新updated_at的表创建触发器
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_admin_users_updated_at BEFORE UPDATE ON admin_users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_system_configs_updated_at BEFORE UPDATE ON system_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 授予权限
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO captcha;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO captcha;
