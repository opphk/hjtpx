-- 数据库表结构初始化脚本
-- 版本: 1.0.0
-- 创建时间: 2026-05-20
-- 描述: 创建核心业务表结构
-- 使用方式: psql -h localhost -U postgres -d hjtpx_db -f 001_init.sql

-- 启用扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 设置默认字符集
SET client_encoding TO 'UTF8';

-- ============================================================
-- 1. 用户表 (users)
-- ============================================================
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    nickname VARCHAR(100) DEFAULT '',
    avatar VARCHAR(500) DEFAULT '',
    phone VARCHAR(20) DEFAULT '',
    bio VARCHAR(500) DEFAULT '',
    is_verified BOOLEAN DEFAULT FALSE,
    verified_at TIMESTAMP WITH TIME ZONE,
    verification_token VARCHAR(100) DEFAULT '',
    password_reset_token VARCHAR(100) DEFAULT '',
    password_reset_at TIMESTAMP WITH TIME ZONE,
    login_count INTEGER DEFAULT 0,
    last_login_at TIMESTAMP WITH TIME ZONE,
    last_login_ip VARCHAR(50) DEFAULT '',
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'banned', 'suspended')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users(username) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
CREATE INDEX IF NOT EXISTS idx_users_verified ON users(is_verified);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at) WHERE deleted_at IS NOT NULL;

-- ============================================================
-- 2. 管理员表 (admins)
-- ============================================================
CREATE TABLE IF NOT EXISTS admins (
    id SERIAL PRIMARY KEY,
    username VARCHAR(100) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'locked', 'suspended')),
    last_login_at TIMESTAMP WITH TIME ZONE,
    last_login_ip VARCHAR(50) DEFAULT '',
    login_count INTEGER DEFAULT 0,
    is_super_admin BOOLEAN DEFAULT FALSE,
    permissions JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_admins_username ON admins(username) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_admins_email ON admins(email) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_admins_status ON admins(status);
CREATE INDEX IF NOT EXISTS idx_admins_super_admin ON admins(is_super_admin);

-- ============================================================
-- 3. 管理员登录日志表 (admin_login_logs)
-- ============================================================
CREATE TABLE IF NOT EXISTS admin_login_logs (
    id SERIAL PRIMARY KEY,
    admin_id INTEGER NOT NULL,
    ip_address VARCHAR(50) DEFAULT '',
    user_agent VARCHAR(500) DEFAULT '',
    status VARCHAR(20) DEFAULT 'success' CHECK (status IN ('success', 'failed', 'mfa_required', 'locked')),
    fail_reason VARCHAR(255) DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_admin_login_logs_admin_id ON admin_login_logs(admin_id);
CREATE INDEX IF NOT EXISTS idx_admin_login_logs_ip ON admin_login_logs(ip_address);
CREATE INDEX IF NOT EXISTS idx_admin_login_logs_status ON admin_login_logs(status);
CREATE INDEX IF NOT EXISTS idx_admin_login_logs_created_at ON admin_login_logs(created_at DESC);

-- ============================================================
-- 4. 应用表 (applications)
-- ============================================================
CREATE TABLE IF NOT EXISTS applications (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    user_id INTEGER NOT NULL,
    description TEXT DEFAULT '',
    api_key VARCHAR(255) NOT NULL,
    domain VARCHAR(255) DEFAULT '',
    website VARCHAR(255) DEFAULT '',
    is_active BOOLEAN DEFAULT TRUE,
    config TEXT DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_applications_api_key ON applications(api_key) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_applications_name ON applications(name, user_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_applications_user_id ON applications(user_id);
CREATE INDEX IF NOT EXISTS idx_applications_domain ON applications(domain);
CREATE INDEX IF NOT EXISTS idx_applications_active ON applications(is_active);

-- ============================================================
-- 5. API密钥历史表 (api_key_histories)
-- ============================================================
CREATE TABLE IF NOT EXISTS api_key_histories (
    id SERIAL PRIMARY KEY,
    application_id INTEGER NOT NULL,
    old_api_key VARCHAR(255) DEFAULT '',
    new_api_key VARCHAR(255) NOT NULL,
    changed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_api_key_histories_app_id ON api_key_histories(application_id);
CREATE INDEX IF NOT EXISTS idx_api_key_histories_changed_at ON api_key_histories(changed_at DESC);

-- ============================================================
-- 6. 验证码记录表 (verifications)
-- ============================================================
CREATE TABLE IF NOT EXISTS verifications (
    id SERIAL PRIMARY KEY,
    application_id INTEGER,
    user_id INTEGER,
    session_id VARCHAR(100) NOT NULL,
    captcha_type VARCHAR(50) NOT NULL CHECK (captcha_type IN ('slider', 'point', 'rotate', 'voice', 'emoji', '3d', 'click')),
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'success', 'failed', 'expired', 'blocked')),
    ip_address VARCHAR(50) DEFAULT '',
    user_agent VARCHAR(500) DEFAULT '',
    risk_score DECIMAL(5,2) DEFAULT 0,
    duration INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_verifications_session_id ON verifications(session_id);
CREATE INDEX IF NOT EXISTS idx_verifications_app_id ON verifications(application_id);
CREATE INDEX IF NOT EXISTS idx_verifications_user_id ON verifications(user_id);
CREATE INDEX IF NOT EXISTS idx_verifications_type ON verifications(captcha_type);
CREATE INDEX IF NOT EXISTS idx_verifications_status ON verifications(status);
CREATE INDEX IF NOT EXISTS idx_verifications_ip ON verifications(ip_address);
CREATE INDEX IF NOT EXISTS idx_verifications_risk ON verifications(risk_score);
CREATE INDEX IF NOT EXISTS idx_verifications_created ON verifications(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_verifications_app_status ON verifications(application_id, status);
CREATE INDEX IF NOT EXISTS idx_verifications_app_created ON verifications(application_id, created_at DESC);

-- ============================================================
-- 7. 行为数据表 (behavior_data)
-- ============================================================
CREATE TABLE IF NOT EXISTS behavior_data (
    id SERIAL PRIMARY KEY,
    verification_id INTEGER NOT NULL,
    data TEXT DEFAULT '',
    data_type VARCHAR(100) DEFAULT 'trajectory',
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_behavior_data_verification_id ON behavior_data(verification_id);
CREATE INDEX IF NOT EXISTS idx_behavior_data_type ON behavior_data(data_type);
CREATE INDEX IF NOT EXISTS idx_behavior_data_timestamp ON behavior_data(timestamp DESC);

-- ============================================================
-- 8. 验证码会话表 (captcha_sessions)
-- ============================================================
CREATE TABLE IF NOT EXISTS captcha_sessions (
    id BIGSERIAL PRIMARY KEY,
    session_id VARCHAR(100) UNIQUE NOT NULL,
    background_url VARCHAR(500) DEFAULT '',
    slider_url VARCHAR(500) DEFAULT '',
    gap_x INTEGER DEFAULT 0,
    gap_y INTEGER DEFAULT 0,
    target_emojis TEXT DEFAULT '[]',
    shuffled_emojis TEXT DEFAULT '[]',
    status VARCHAR(50) DEFAULT 'pending' CHECK (status IN ('pending', 'success', 'failed', 'expired')),
    verify_count INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    risk_score DECIMAL(5,2) DEFAULT 0,
    trace_score DECIMAL(5,2) DEFAULT 0,
    env_score DECIMAL(5,2) DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expired_at TIMESTAMP WITH TIME ZONE NOT NULL,
    verified_at TIMESTAMP WITH TIME ZONE,
    client_ip VARCHAR(50) DEFAULT '',
    user_agent VARCHAR(500) DEFAULT '',
    fingerprint VARCHAR(255) DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_captcha_sessions_session_id ON captcha_sessions(session_id);
CREATE INDEX IF NOT EXISTS idx_captcha_sessions_status ON captcha_sessions(status);
CREATE INDEX IF NOT EXISTS idx_captcha_sessions_expired_at ON captcha_sessions(expired_at);
CREATE INDEX IF NOT EXISTS idx_captcha_sessions_created_at ON captcha_sessions(created_at DESC);

-- ============================================================
-- 9. 语音验证码会话表 (voice_captcha_sessions)
-- ============================================================
CREATE TABLE IF NOT EXISTS voice_captcha_sessions (
    id BIGSERIAL PRIMARY KEY,
    session_id VARCHAR(100) UNIQUE NOT NULL,
    code VARCHAR(20) NOT NULL,
    language VARCHAR(20) DEFAULT 'zh-CN',
    status VARCHAR(50) DEFAULT 'pending' CHECK (status IN ('pending', 'success', 'failed', 'expired')),
    verify_count INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expired_at TIMESTAMP WITH TIME ZONE NOT NULL,
    verified_at TIMESTAMP WITH TIME ZONE,
    client_ip VARCHAR(50) DEFAULT '',
    user_agent VARCHAR(500) DEFAULT '',
    fingerprint VARCHAR(255) DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_voice_sessions_session_id ON voice_captcha_sessions(session_id);
CREATE INDEX IF NOT EXISTS idx_voice_sessions_status ON voice_captcha_sessions(status);

-- ============================================================
-- 10. 设备指纹表 (device_fingerprints)
-- ============================================================
CREATE TABLE IF NOT EXISTS device_fingerprints (
    id SERIAL PRIMARY KEY,
    fingerprint VARCHAR(64) UNIQUE NOT NULL,
    canvas_hash VARCHAR(64) DEFAULT '',
    webgl_vendor VARCHAR(100) DEFAULT '',
    webgl_renderer VARCHAR(100) DEFAULT '',
    user_agent VARCHAR(500) DEFAULT '',
    ip_address VARCHAR(45) DEFAULT '',
    screen_info VARCHAR(100) DEFAULT '',
    timezone VARCHAR(100) DEFAULT '',
    language VARCHAR(50) DEFAULT '',
    fonts TEXT DEFAULT '',
    plugins TEXT DEFAULT '',
    first_seen TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_seen TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    visit_count INTEGER DEFAULT 1,
    is_bot BOOLEAN DEFAULT FALSE,
    risk_level VARCHAR(20) DEFAULT 'low' CHECK (risk_level IN ('low', 'medium', 'high', 'critical')),
    risk_score DECIMAL(5,2) DEFAULT 0,
    proxy_detected BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_device_fingerprints_fp ON device_fingerprints(fingerprint);
CREATE INDEX IF NOT EXISTS idx_device_fingerprints_ip ON device_fingerprints(ip_address);
CREATE INDEX IF NOT EXISTS idx_device_fingerprints_last_seen ON device_fingerprints(last_seen DESC);
CREATE INDEX IF NOT EXISTS idx_device_fingerprints_bot ON device_fingerprints(is_bot);
CREATE INDEX IF NOT EXISTS idx_device_fingerprints_risk_level ON device_fingerprints(risk_level);

-- ============================================================
-- 11. 风控规则表 (risk_rules)
-- ============================================================
CREATE TABLE IF NOT EXISTS risk_rules (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '',
    template_id INTEGER,
    rule_type VARCHAR(50) NOT NULL CHECK (rule_type IN ('rate_limit', 'behavior', 'device', 'ip', 'custom')),
    condition TEXT NOT NULL,
    action VARCHAR(50) NOT NULL CHECK (action IN ('allow', 'block', 'challenge', 'review', 'log')),
    params TEXT DEFAULT '{}',
    severity VARCHAR(20) DEFAULT 'medium' CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    priority INTEGER DEFAULT 100,
    is_enabled BOOLEAN DEFAULT TRUE,
    application_ids TEXT DEFAULT '[]',
    created_by INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_risk_rules_name ON risk_rules(name);
CREATE INDEX IF NOT EXISTS idx_risk_rules_type ON risk_rules(rule_type);
CREATE INDEX IF NOT EXISTS idx_risk_rules_severity ON risk_rules(severity);
CREATE INDEX IF NOT EXISTS idx_risk_rules_priority ON risk_rules(priority DESC);
CREATE INDEX IF NOT EXISTS idx_risk_rules_enabled ON risk_rules(is_enabled) WHERE is_enabled = TRUE;
CREATE INDEX IF NOT EXISTS idx_risk_rules_template ON risk_rules(template_id);

-- ============================================================
-- 12. 风控规则模板表 (risk_rule_templates)
-- ============================================================
CREATE TABLE IF NOT EXISTS risk_rule_templates (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '',
    category VARCHAR(100) DEFAULT '' CHECK (category IN ('rate_limit', 'behavior', 'device', 'ip', 'geo', 'time', 'custom')),
    rule_type VARCHAR(50) NOT NULL,
    condition TEXT NOT NULL,
    action VARCHAR(50) NOT NULL,
    params TEXT DEFAULT '{}',
    severity VARCHAR(20) DEFAULT 'medium',
    is_active BOOLEAN DEFAULT TRUE,
    is_system BOOLEAN DEFAULT FALSE,
    created_by INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_risk_rule_templates_name ON risk_rule_templates(name);
CREATE INDEX IF NOT EXISTS idx_risk_rule_templates_category ON risk_rule_templates(category);
CREATE INDEX IF NOT EXISTS idx_risk_rule_templates_active ON risk_rule_templates(is_active) WHERE is_active = TRUE;

-- ============================================================
-- 13. 规则触发历史表 (risk_rule_trigger_histories)
-- ============================================================
CREATE TABLE IF NOT EXISTS risk_rule_trigger_histories (
    id SERIAL PRIMARY KEY,
    rule_id INTEGER NOT NULL,
    rule_name VARCHAR(255) DEFAULT '',
    session_id VARCHAR(100) DEFAULT '',
    application_id INTEGER,
    user_id INTEGER,
    ip_address VARCHAR(50) DEFAULT '',
    input_data TEXT DEFAULT '{}',
    trigger_result BOOLEAN DEFAULT FALSE,
    action_taken VARCHAR(50) DEFAULT '',
    execution_time INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_rule_trigger_rule_id ON risk_rule_trigger_histories(rule_id);
CREATE INDEX IF NOT EXISTS idx_rule_trigger_session ON risk_rule_trigger_histories(session_id);
CREATE INDEX IF NOT EXISTS idx_rule_trigger_app ON risk_rule_trigger_histories(application_id);
CREATE INDEX IF NOT EXISTS idx_rule_trigger_user ON risk_rule_trigger_histories(user_id);
CREATE INDEX IF NOT EXISTS idx_rule_trigger_time ON risk_rule_trigger_histories(created_at DESC);

-- ============================================================
-- 14. 风控日志表 (risk_logs)
-- ============================================================
CREATE TABLE IF NOT EXISTS risk_logs (
    id SERIAL PRIMARY KEY,
    session_id VARCHAR(100) DEFAULT '',
    user_id INTEGER,
    risk_type VARCHAR(50) NOT NULL,
    risk_level VARCHAR(20) NOT NULL CHECK (risk_level IN ('low', 'medium', 'high', 'critical')),
    risk_score DECIMAL(5,2) NOT NULL DEFAULT 0,
    risk_factors TEXT DEFAULT '[]',
    decision VARCHAR(20) NOT NULL CHECK (decision IN ('allow', 'block', 'challenge', 'review')),
    action_taken TEXT DEFAULT '',
    description TEXT DEFAULT '',
    metadata TEXT DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    client_ip VARCHAR(50) DEFAULT '',
    user_agent TEXT DEFAULT '',
    resolved BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolved_by INTEGER,
    resolution_notes TEXT DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_risk_logs_session_id ON risk_logs(session_id);
CREATE INDEX IF NOT EXISTS idx_risk_logs_user_id ON risk_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_risk_logs_type ON risk_logs(risk_type);
CREATE INDEX IF NOT EXISTS idx_risk_logs_level ON risk_logs(risk_level);
CREATE INDEX IF NOT EXISTS idx_risk_logs_decision ON risk_logs(decision);
CREATE INDEX IF NOT EXISTS idx_risk_logs_created_at ON risk_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_risk_logs_ip ON risk_logs(client_ip);
CREATE INDEX IF NOT EXISTS idx_risk_logs_resolved ON risk_logs(resolved) WHERE resolved = FALSE;

-- ============================================================
-- 15. 验证轨迹记录表 (trace_records)
-- ============================================================
CREATE TABLE IF NOT EXISTS trace_records (
    id SERIAL PRIMARY KEY,
    session_id VARCHAR(100) DEFAULT '',
    verification_id INTEGER,
    application_id INTEGER,
    raw_data TEXT DEFAULT '',
    features_data TEXT DEFAULT '{}',
    score_data TEXT DEFAULT '{}',
    total_time BIGINT DEFAULT 0,
    total_score DECIMAL(5,2) DEFAULT 0,
    move_count INTEGER DEFAULT 0,
    avg_speed DECIMAL(10,2) DEFAULT 0,
    max_speed DECIMAL(10,2) DEFAULT 0,
    risk_factors TEXT DEFAULT '[]',
    ip_address VARCHAR(50) DEFAULT '',
    user_agent VARCHAR(500) DEFAULT '',
    device_info VARCHAR(255) DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_trace_records_session_id ON trace_records(session_id);
CREATE INDEX IF NOT EXISTS idx_trace_records_verification_id ON trace_records(verification_id);
CREATE INDEX IF NOT EXISTS idx_trace_records_app_id ON trace_records(application_id);
CREATE INDEX IF NOT EXISTS idx_trace_records_created_at ON trace_records(created_at DESC);

-- ============================================================
-- 16. 审计日志表 (audit_logs)
-- ============================================================
CREATE TABLE IF NOT EXISTS audit_logs (
    id SERIAL PRIMARY KEY,
    log_type VARCHAR(50) DEFAULT 'system' CHECK (log_type IN ('system', 'security', 'user', 'admin', 'api', 'config')),
    level VARCHAR(20) DEFAULT 'info' CHECK (level IN ('debug', 'info', 'warning', 'error', 'critical')),
    user_id INTEGER,
    username VARCHAR(100) DEFAULT '',
    ip_address VARCHAR(50) DEFAULT '',
    user_agent VARCHAR(500) DEFAULT '',
    action VARCHAR(255) NOT NULL,
    resource_type VARCHAR(50) DEFAULT '',
    resource_id VARCHAR(100) DEFAULT '',
    status VARCHAR(20) DEFAULT 'success',
    error_message TEXT DEFAULT '',
    changes TEXT DEFAULT '{}',
    metadata TEXT DEFAULT '{}',
    duration BIGINT DEFAULT 0,
    session_id VARCHAR(100) DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_type ON audit_logs(log_type);
CREATE INDEX IF NOT EXISTS idx_audit_logs_level ON audit_logs(level);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_ip ON audit_logs(ip_address);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource_type, resource_id);

-- ============================================================
-- 17. 黑名单表 (blacklist)
-- ============================================================
CREATE TABLE IF NOT EXISTS blacklist (
    id SERIAL PRIMARY KEY,
    target VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('ip', 'user', 'device', 'session', 'email', 'phone')),
    source VARCHAR(50) DEFAULT 'manual' CHECK (source IN ('manual', 'auto', 'system', 'rule')),
    reason TEXT DEFAULT '',
    action VARCHAR(50) DEFAULT 'block' CHECK (action IN ('block', 'challenge', 'review', 'allow')),
    status VARCHAR(50) DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'expired')),
    note TEXT DEFAULT '',
    created_by INTEGER DEFAULT 0,
    hit_count INTEGER DEFAULT 0,
    application_ids TEXT DEFAULT '[]',
    expiration VARCHAR(50) DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_blacklist_target ON blacklist(target);
CREATE INDEX IF NOT EXISTS idx_blacklist_type ON blacklist(type);
CREATE INDEX IF NOT EXISTS idx_blacklist_status ON blacklist(status);
CREATE INDEX IF NOT EXISTS idx_blacklist_created_at ON blacklist(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_blacklist_target_type ON blacklist(target, type);
CREATE INDEX IF NOT EXISTS idx_blacklist_expires_at ON blacklist(expires_at) WHERE expires_at IS NOT NULL;

-- ============================================================
-- 18. 系统配置表 (system_configs)
-- ============================================================
CREATE TABLE IF NOT EXISTS system_configs (
    id SERIAL PRIMARY KEY,
    config_key VARCHAR(100) UNIQUE NOT NULL,
    config_value TEXT NOT NULL,
    config_type VARCHAR(20) NOT NULL CHECK (config_type IN ('string', 'number', 'boolean', 'json', 'array')),
    description TEXT DEFAULT '',
    category VARCHAR(50) DEFAULT 'general' CHECK (category IN ('general', 'captcha', 'risk', 'security', 'rate_limit', 'session', 'notification', 'backup', 'integration')),
    is_sensitive BOOLEAN DEFAULT FALSE,
    is_public BOOLEAN DEFAULT FALSE,
    validation_rules TEXT DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_by INTEGER
);

CREATE INDEX IF NOT EXISTS idx_system_configs_key ON system_configs(config_key);
CREATE INDEX IF NOT EXISTS idx_system_configs_category ON system_configs(category);
CREATE INDEX IF NOT EXISTS idx_system_configs_type ON system_configs(config_type);

-- ============================================================
-- 19. 验证日志表 (verification_logs)
-- ============================================================
CREATE TABLE IF NOT EXISTS verification_logs (
    id SERIAL PRIMARY KEY,
    verification_id INTEGER NOT NULL,
    session_id VARCHAR(100) DEFAULT '',
    application_id INTEGER NOT NULL,
    captcha_type VARCHAR(50) DEFAULT '',
    status VARCHAR(50) NOT NULL,
    ip_address VARCHAR(50) DEFAULT '',
    user_agent VARCHAR(500) DEFAULT '',
    risk_score DECIMAL(5,2) DEFAULT 0,
    analysis_result TEXT DEFAULT '',
    duration INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_verification_logs_verification_id ON verification_logs(verification_id);
CREATE INDEX IF NOT EXISTS idx_verification_logs_session_id ON verification_logs(session_id);
CREATE INDEX IF NOT EXISTS idx_verification_logs_app_id ON verification_logs(application_id);
CREATE INDEX IF NOT EXISTS idx_verification_logs_status ON verification_logs(status);
CREATE INDEX IF NOT EXISTS idx_verification_logs_created_at ON verification_logs(created_at DESC);

-- ============================================================
-- 20. MFA配置表 (user_mfa)
-- ============================================================
CREATE TABLE IF NOT EXISTS user_mfa (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL UNIQUE,
    mfa_type VARCHAR(20) NOT NULL CHECK (mfa_type IN ('totp', 'sms', 'email', 'backup')),
    secret VARCHAR(255) DEFAULT '',
    phone VARCHAR(20) DEFAULT '',
    email VARCHAR(255) DEFAULT '',
    is_enabled BOOLEAN DEFAULT FALSE,
    backup_codes TEXT DEFAULT '[]',
    last_verified_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_user_mfa_user_id ON user_mfa(user_id);
CREATE INDEX IF NOT EXISTS idx_user_mfa_type ON user_mfa(mfa_type);
CREATE INDEX IF NOT EXISTS idx_user_mfa_enabled ON user_mfa(is_enabled) WHERE is_enabled = TRUE;

-- ============================================================
-- 21. MFA配置表 (admin_mfa)
-- ============================================================
CREATE TABLE IF NOT EXISTS admin_mfa (
    id SERIAL PRIMARY KEY,
    admin_id INTEGER NOT NULL UNIQUE,
    mfa_type VARCHAR(20) NOT NULL CHECK (mfa_type IN ('totp', 'sms', 'email', 'backup')),
    secret VARCHAR(255) DEFAULT '',
    phone VARCHAR(20) DEFAULT '',
    email VARCHAR(255) DEFAULT '',
    is_enabled BOOLEAN DEFAULT FALSE,
    backup_codes TEXT DEFAULT '[]',
    last_verified_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_admin_mfa_admin_id ON admin_mfa(admin_id);
CREATE INDEX IF NOT EXISTS idx_admin_mfa_type ON admin_mfa(mfa_type);
CREATE INDEX IF NOT EXISTS idx_admin_mfa_enabled ON admin_mfa(is_enabled) WHERE is_enabled = TRUE;

-- ============================================================
-- 22. MFA验证码表 (mfa_codes)
-- ============================================================
CREATE TABLE IF NOT EXISTS mfa_codes (
    id SERIAL PRIMARY KEY,
    target_type VARCHAR(20) NOT NULL CHECK (target_type IN ('user', 'admin')),
    target_id INTEGER NOT NULL,
    mfa_type VARCHAR(20) NOT NULL CHECK (mfa_type IN ('sms', 'email')),
    code VARCHAR(20) NOT NULL,
    destination VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    is_used BOOLEAN DEFAULT FALSE,
    used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_mfa_codes_target ON mfa_codes(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_mfa_codes_expires ON mfa_codes(expires_at);
CREATE INDEX IF NOT EXISTS idx_mfa_codes_code ON mfa_codes(code, destination);
CREATE INDEX IF NOT EXISTS idx_mfa_codes_used ON mfa_codes(is_used) WHERE is_used = FALSE;

-- ============================================================
-- 创建更新时间戳的函数
-- ============================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- ============================================================
-- 为需要自动更新updated_at的表创建触发器
-- ============================================================
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_admins_updated_at
    BEFORE UPDATE ON admins
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_applications_updated_at
    BEFORE UPDATE ON applications
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_system_configs_updated_at
    BEFORE UPDATE ON system_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_device_fingerprints_updated_at
    BEFORE UPDATE ON device_fingerprints
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_risk_rules_updated_at
    BEFORE UPDATE ON risk_rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 创建部分索引用于性能优化
-- ============================================================
CREATE INDEX IF NOT EXISTS idx_users_email_lowercase ON users(LOWER(email)) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_username_lowercase ON users(LOWER(username)) WHERE deleted_at IS NULL;

-- ============================================================
-- 创建复合索引用于常见查询
-- ============================================================
CREATE INDEX IF NOT EXISTS idx_verifications_app_status_created ON verifications(application_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_verifications_user_status_created ON verifications(user_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_action_created ON audit_logs(user_id, action, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_risk_logs_level_created ON risk_logs(risk_level, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_applications_user_active ON applications(user_id, is_active) WHERE deleted_at IS NULL;

-- ============================================================
-- 注释说明
-- ============================================================
COMMENT ON TABLE users IS '用户表 - 存储用户基本信息';
COMMENT ON TABLE admins IS '管理员表 - 存储管理员账户信息';
COMMENT ON TABLE applications IS '应用表 - 存储接入的应用程序信息';
COMMENT ON TABLE verifications IS '验证码记录表 - 存储验证码验证记录';
COMMENT ON TABLE risk_rules IS '风控规则表 - 存储风控规则配置';
COMMENT ON TABLE audit_logs IS '审计日志表 - 存储系统审计日志';
COMMENT ON TABLE risk_logs IS '风控日志表 - 存储风控检测日志';

-- ============================================================
-- 授予权限 (根据实际情况调整)
-- ============================================================
-- GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO app_user;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO app_user;

-- ============================================================
-- 迁移版本记录表
-- ============================================================
CREATE TABLE IF NOT EXISTS schema_migrations (
    id SERIAL PRIMARY KEY,
    version VARCHAR(50) NOT NULL UNIQUE,
    applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    description TEXT DEFAULT ''
);

INSERT INTO schema_migrations (version, description) VALUES ('001_init', 'Initial schema with core tables')
ON CONFLICT (version) DO NOTHING;
