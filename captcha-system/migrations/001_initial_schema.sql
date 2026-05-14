-- 验证码挑战表
CREATE TABLE IF NOT EXISTS captcha_challenge (
    id SERIAL PRIMARY KEY,
    challenge_id VARCHAR(64) UNIQUE NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('slider', 'click', 'rotate')),
    difficulty VARCHAR(20) DEFAULT 'medium' CHECK (difficulty IN ('easy', 'medium', 'hard')),
    data JSONB NOT NULL,
    solution JSONB NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 验证码尝试记录表
CREATE TABLE IF NOT EXISTS captcha_attempt (
    id SERIAL PRIMARY KEY,
    challenge_id VARCHAR(64) NOT NULL REFERENCES captcha_challenge(challenge_id),
    session_id VARCHAR(128) NOT NULL,
    user_answer JSONB NOT NULL,
    is_valid BOOLEAN NOT NULL,
    response_time_ms INTEGER NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    fingerprint VARCHAR(256),
    risk_score DECIMAL(5,2) DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 验证码统计表
CREATE TABLE IF NOT EXISTS captcha_stats (
    id SERIAL PRIMARY KEY,
    stat_date DATE NOT NULL UNIQUE,
    total_attempts INTEGER DEFAULT 0,
    successful_attempts INTEGER DEFAULT 0,
    failed_attempts INTEGER DEFAULT 0,
    blocked_attempts INTEGER DEFAULT 0,
    slider_attempts INTEGER DEFAULT 0,
    click_attempts INTEGER DEFAULT 0,
    rotate_attempts INTEGER DEFAULT 0,
    avg_response_time_ms INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 管理员用户表
CREATE TABLE IF NOT EXISTS admin_user (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) DEFAULT 'admin' CHECK (role IN ('admin', 'superadmin')),
    is_active BOOLEAN DEFAULT TRUE,
    last_login_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 验证码配置表
CREATE TABLE IF NOT EXISTS captcha_config (
    id SERIAL PRIMARY KEY,
    config_key VARCHAR(50) UNIQUE NOT NULL,
    config_value JSONB NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 会话表
CREATE TABLE IF NOT EXISTS captcha_session (
    id SERIAL PRIMARY KEY,
    session_id VARCHAR(128) UNIQUE NOT NULL,
    fingerprint VARCHAR(256),
    ip_address VARCHAR(45),
    risk_score DECIMAL(5,2) DEFAULT 0,
    attempt_count INTEGER DEFAULT 0,
    blocked_until TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL
);

-- 操作日志表
CREATE TABLE IF NOT EXISTS captcha_log (
    id SERIAL PRIMARY KEY,
    level VARCHAR(20) NOT NULL,
    message TEXT NOT NULL,
    request_id VARCHAR(64),
    session_id VARCHAR(128),
    ip_address VARCHAR(45),
    user_agent TEXT,
    extra JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX idx_challenge_expires ON captcha_challenge(expires_at);
CREATE INDEX idx_challenge_type ON captcha_challenge(type);
CREATE INDEX idx_attempt_challenge ON captcha_attempt(challenge_id);
CREATE INDEX idx_attempt_session ON captcha_attempt(session_id);
CREATE INDEX idx_attempt_created ON captcha_attempt(created_at);
CREATE INDEX idx_session_id ON captcha_session(session_id);
CREATE INDEX idx_session_fingerprint ON captcha_session(fingerprint);
CREATE INDEX idx_log_level ON captcha_log(level);
CREATE INDEX idx_log_created ON captcha_log(created_at);
