-- 初始化数据库脚本

-- 创建应用表
CREATE TABLE IF NOT EXISTS applications (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    app_key VARCHAR(64) NOT NULL UNIQUE,
    secret_key VARCHAR(128) NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_request_at TIMESTAMP,
    request_count BIGINT DEFAULT 0,
    allow_ips TEXT,
    rate_limit INT DEFAULT 100,
    config JSONB DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_applications_app_key ON applications (app_key);
CREATE INDEX IF NOT EXISTS idx_applications_status ON applications (status);
CREATE INDEX IF NOT EXISTS idx_applications_created_at ON applications (created_at);

-- 创建用户表
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(100) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    email_verified BOOLEAN DEFAULT FALSE,
    verification_token VARCHAR(255),
    password_reset_token VARCHAR(255),
    password_reset_expires TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP,
    login_count INT DEFAULT 0,
    status VARCHAR(20) DEFAULT 'active'
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users (username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users (created_at);

-- 创建管理员表
CREATE TABLE IF NOT EXISTS admins (
    id SERIAL PRIMARY KEY,
    username VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    is_super_admin BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP,
    login_count INT DEFAULT 0,
    status VARCHAR(20) DEFAULT 'active'
);

CREATE INDEX IF NOT EXISTS idx_admins_username ON admins (username);

-- 创建验证码验证日志表
CREATE TABLE IF NOT EXISTS verification_logs (
    id SERIAL PRIMARY KEY,
    session_id VARCHAR(64) NOT NULL,
    captcha_type VARCHAR(50) NOT NULL,
    client_ip VARCHAR(45),
    user_agent TEXT,
    app_key VARCHAR(64),
    result VARCHAR(20) NOT NULL,
    response_time_ms INT,
    risk_score DECIMAL(5,2),
    risk_factors JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_verification_logs_session_id ON verification_logs (session_id);
CREATE INDEX IF NOT EXISTS idx_verification_logs_captcha_type ON verification_logs (captcha_type);
CREATE INDEX IF NOT EXISTS idx_verification_logs_result ON verification_logs (result);
CREATE INDEX IF NOT EXISTS idx_verification_logs_created_at ON verification_logs (created_at);
CREATE INDEX IF NOT EXISTS idx_verification_logs_app_key ON verification_logs (app_key);
CREATE INDEX IF NOT EXISTS idx_verification_logs_client_ip ON verification_logs (client_ip);
CREATE INDEX IF NOT EXISTS idx_verification_logs_time_range ON verification_logs (created_at DESC);

-- 创建安全日志表
CREATE TABLE IF NOT EXISTS security_logs (
    id SERIAL PRIMARY KEY,
    event_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) DEFAULT 'info',
    client_ip VARCHAR(45),
    user_agent TEXT,
    request_path VARCHAR(500),
    request_method VARCHAR(10),
    request_body TEXT,
    error_message TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_security_logs_event_type ON security_logs (event_type);
CREATE INDEX IF NOT EXISTS idx_security_logs_severity ON security_logs (severity);
CREATE INDEX IF NOT EXISTS idx_security_logs_client_ip ON security_logs (client_ip);
CREATE INDEX IF NOT EXISTS idx_security_logs_created_at ON security_logs (created_at);
CREATE INDEX IF NOT EXISTS idx_security_logs_time_range ON security_logs (created_at DESC);

-- 创建黑名单表
CREATE TABLE IF NOT EXISTS blacklist (
    id SERIAL PRIMARY KEY,
    ip_address VARCHAR(45) NOT NULL UNIQUE,
    reason TEXT,
    expires_at TIMESTAMP,
    created_by INTEGER REFERENCES admins(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_blacklist_ip_address ON blacklist (ip_address);
CREATE INDEX IF NOT EXISTS idx_blacklist_expires_at ON blacklist (expires_at);

-- 创建配置表（用于动态配置）
CREATE TABLE IF NOT EXISTS system_config (
    id SERIAL PRIMARY KEY,
    config_key VARCHAR(100) NOT NULL UNIQUE,
    config_value JSONB NOT NULL,
    description TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_by INTEGER REFERENCES admins(id)
);

CREATE INDEX IF NOT EXISTS idx_system_config_config_key ON system_config (config_key);

-- 创建速率限制记录表
CREATE TABLE IF NOT EXISTS rate_limit_records (
    id SERIAL PRIMARY KEY,
    identifier VARCHAR(255) NOT NULL,
    identifier_type VARCHAR(20) NOT NULL,
    request_count INT DEFAULT 1,
    window_start TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(identifier, identifier_type, window_start)
);

CREATE INDEX IF NOT EXISTS idx_rate_limit_identifier ON rate_limit_records (identifier);
CREATE INDEX IF NOT EXISTS idx_rate_limit_window_start ON rate_limit_records (window_start);
CREATE INDEX IF NOT EXISTS idx_rate_limit_window ON rate_limit_records (window_start DESC);

-- 授予用户权限
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO verification_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO verification_user;
