-- 数据初始化脚本
-- 版本: 1.0.0
-- 创建时间: 2026-05-20
-- 描述: 初始化默认数据
-- 使用方式: psql -h localhost -U hjtpx -d hjtpx -f init-data.sql

BEGIN;

-- ============================================================
-- 0. 确保 system_configs 表存在 (如果 001_init.sql 未执行)
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
-- 0.1 确保 admins 表存在
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

-- ============================================================
-- 0.2 确保 risk_rule_templates 表存在
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

-- ============================================================
-- 0.3 确保 risk_rules 表存在
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

-- ============================================================
-- 0.4 确保 schema_migrations 表存在
-- ============================================================
CREATE TABLE IF NOT EXISTS schema_migrations (
    id SERIAL PRIMARY KEY,
    version VARCHAR(50) NOT NULL UNIQUE,
    applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    description TEXT DEFAULT ''
);

-- ============================================================
-- 1. 创建默认管理员账户
-- 注意: 密码为 'admin123' 的 bcrypt hash，实际部署请修改
-- ============================================================
INSERT INTO admins (username, password_hash, email, is_super_admin, status, permissions)
VALUES (
    'admin',
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', -- admin123
    'admin@example.com',
    TRUE,
    'active',
    '["all"]'::jsonb
)
ON CONFLICT (username) WHERE deleted_at IS NULL DO NOTHING;

INSERT INTO admins (username, password_hash, email, is_super_admin, status, permissions)
VALUES (
    'operator',
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', -- admin123
    'operator@example.com',
    FALSE,
    'active',
    '["read", "write", "manage_applications", "view_logs"]'::jsonb
)
ON CONFLICT (username) WHERE deleted_at IS NULL DO NOTHING;

-- ============================================================
-- 2. 系统配置初始化
-- ============================================================
INSERT INTO system_configs (config_key, config_value, config_type, description, category)
VALUES
-- 验证码配置
('captcha.default_difficulty', 'medium', 'string', '默认验证码难度', 'captcha'),
('captcha.max_attempts', '3', 'number', '最大验证尝试次数', 'captcha'),
('captcha.expiration_seconds', '300', 'number', '验证码过期时间(秒)', 'captcha'),
('captcha.slider_width', '50', 'number', '滑块宽度(像素)', 'captcha'),
('captcha.slider_height', '50', 'number', '滑块高度(像素)', 'captcha'),
('captcha.image_width', '300', 'number', '验证码图片宽度', 'captcha'),
('captcha.image_height', '150', 'number', '验证码图片高度', 'captcha'),
('captcha.enable_voice', 'true', 'boolean', '是否启用语音验证码', 'captcha'),
('captcha.enable_emoji', 'true', 'boolean', '是否启用表情验证码', 'captcha'),
('captcha.enable_3d', 'true', 'boolean', '是否启用3D验证码', 'captcha'),

-- 风控配置
('risk.score_threshold_low', '30', 'number', '低风险分数阈值', 'risk'),
('risk.score_threshold_medium', '60', 'number', '中风险分数阈值', 'risk'),
('risk.score_threshold_high', '80', 'number', '高风险分数阈值', 'risk'),
('risk.max_requests_per_minute', '60', 'number', '每分钟最大请求数', 'risk'),
('risk.max_failed_attempts', '5', 'number', '最大失败尝试次数', 'risk'),
('risk.ip_whitelist', '[]', 'array', 'IP白名单', 'risk'),
('risk.enable_proxy_detection', 'true', 'boolean', '是否启用代理检测', 'risk'),
('risk.enable_vpn_detection', 'true', 'boolean', '是否启用VPN检测', 'risk'),
('risk.enable_tor_detection', 'true', 'boolean', '是否启用Tor检测', 'risk'),

-- 安全配置
('security.password_min_length', '8', 'number', '密码最小长度', 'security'),
('security.password_require_uppercase', 'false', 'boolean', '密码是否需要大写字母', 'security'),
('security.password_require_numbers', 'false', 'boolean', '密码是否需要数字', 'security'),
('security.password_require_special', 'false', 'boolean', '密码是否需要特殊字符', 'security'),
('security.enable_captcha', 'true', 'boolean', '是否启用验证码', 'security'),
('security.enable_mfa', 'false', 'boolean', '是否启用双因素认证', 'security'),
('security.max_login_attempts', '5', 'number', '最大登录尝试次数', 'security'),
('security.lockout_duration_minutes', '30', 'number', '账户锁定时长(分钟)', 'security'),
('security.session_timeout_minutes', '30', 'number', '会话超时时间(分钟)', 'security'),
('security.allow_multiple_sessions', 'false', 'boolean', '是否允许多会话', 'security'),
('security.enable_csrf', 'true', 'boolean', '是否启用CSRF保护', 'security'),
('security.enable_xss_protection', 'true', 'boolean', '是否启用XSS保护', 'security'),
('security.enable_signature', 'true', 'boolean', '是否启用签名验证', 'security'),

-- 会话配置
('session.timeout_minutes', '30', 'number', '会话超时时间(分钟)', 'session'),
('session.remember_me_days', '7', 'number', '记住我有效期(天)', 'session'),
('session.cookie_secure', 'false', 'boolean', 'Cookie是否仅HTTPS', 'session'),
('session.cookie_httponly', 'true', 'boolean', 'Cookie是否HttpOnly', 'session'),

-- 速率限制配置
('rate_limit.enabled', 'true', 'boolean', '是否启用速率限制', 'rate_limit'),
('rate_limit.default_limit', '100', 'number', '默认速率限制', 'rate_limit'),
('rate_limit.window_seconds', '60', 'number', '速率限制时间窗口(秒)', 'rate_limit'),
('rate_limit.burst_limit', '200', 'number', '突发限制', 'rate_limit'),

-- 通知配置
('notification.email.enabled', 'false', 'boolean', '是否启用邮件通知', 'notification'),
('notification.sms.enabled', 'false', 'boolean', '是否启用短信通知', 'notification'),
('notification.webhook.enabled', 'false', 'boolean', '是否启用Webhook通知', 'notification'),

-- 备份配置
('backup.enabled', 'false', 'boolean', '是否启用自动备份', 'backup'),
('backup.schedule', '0 2 * * *', 'string', '备份cron表达式', 'backup'),
('backup.retention_days', '7', 'number', '备份保留天数', 'backup'),

-- 通用配置
('general.site_name', '验证码服务', 'string', '站点名称', 'general'),
('general.site_url', 'http://localhost:8080', 'string', '站点URL', 'general'),
('general.api_version', 'v1', 'string', 'API版本', 'general'),
('general.maintenance_mode', 'false', 'boolean', '维护模式', 'general')
ON CONFLICT (config_key) DO NOTHING;

-- ============================================================
-- 3. 风控规则模板初始化
-- ============================================================
INSERT INTO risk_rule_templates (name, description, category, rule_type, condition, action, params, severity, is_system)
VALUES
-- IP相关规则
('IP频率限制', '限制单个IP的请求频率', 'rate_limit', 'rate_limit',
 '{"type": "ip", "threshold": 60, "window": 60}',
 'challenge',
 '{"max_challenges": 3, "block_after": 10}',
 'medium', TRUE),

('IP黑名单', '自动封禁恶意IP', 'ip', 'ip',
 '{"check_blacklist": true, "auto_block": true}',
 'block',
 '{"block_duration": 3600}',
 'high', TRUE),

('IP段限制', '限制IP段的请求频率', 'rate_limit', 'rate_limit',
 '{"type": "ip_range", "threshold": 100, "window": 60, "range": "/24"}',
 'challenge',
 '{}',
 'low', TRUE),

-- 行为相关规则
('异常速度检测', '检测异常快速的操作行为', 'behavior', 'behavior',
 '{"type": "speed", "max_speed": 2000, "min_time": 500}',
 'challenge',
 '{"evidence_required": true}',
 'medium', TRUE),

('机械行为检测', '检测机械化的重复行为', 'behavior', 'behavior',
 '{"type": "mechanical", "patterns": ["constant_speed", "linear_path"]}',
 'challenge',
 '{"sensitivity": 0.8}',
 'high', TRUE),

('异常轨迹检测', '检测异常的鼠标轨迹', 'behavior', 'behavior',
 '{"type": "trajectory", "threshold": 0.7}',
 'log',
 '{"store_trajectory": true}',
 'low', TRUE),

-- 设备相关规则
('新设备验证', '新设备登录需要额外验证', 'device', 'device',
 '{"type": "new_device", "trust_threshold": 3}',
 'challenge',
 '{"trust_after_verifications": 3}',
 'medium', TRUE),

('设备指纹异常', '设备指纹异常时需要验证', 'device', 'device',
 '{"type": "fingerprint_mismatch"}',
 'challenge',
 '{}',
 'high', TRUE),

-- 地理相关规则
('异常地理位置', '检测异常地理位置访问', 'geo', 'custom',
 '{"type": "geo_anomaly", "max_distance_km": 500, "max_time_hours": 2}',
 'review',
 '{"auto_review": true}',
 'medium', TRUE),

-- 时间相关规则
('异常时间段访问', '检测异常时间段的访问', 'time', 'custom',
 '{"type": "unusual_time", "allowed_hours": "0-6"}',
 'log',
 '{}',
 'low', TRUE),

-- 自定义规则
('批量注册检测', '检测批量注册行为', 'custom', 'custom',
 '{"type": "batch_action", "threshold": 10, "window": 300, "action_type": "register"}',
 'block',
 '{"auto_block": true, "duration": 3600}',
 'critical', TRUE),

('暴力破解检测', '检测暴力破解尝试', 'custom', 'custom',
 '{"type": "brute_force", "threshold": 5, "window": 300, "action_type": "login"}',
 'block',
 '{"auto_block": true, "duration": 1800}',
 'critical', TRUE)
ON CONFLICT DO NOTHING;

-- ============================================================
-- 4. 初始化默认风控规则 (基于模板)
-- ============================================================
INSERT INTO risk_rules (name, description, template_id, rule_type, condition, action, params, severity, priority, is_enabled, created_by)
SELECT
    name,
    description,
    id,
    rule_type,
    condition,
    action,
    params,
    severity,
    CASE
        WHEN severity = 'critical' THEN 10
        WHEN severity = 'high' THEN 20
        WHEN severity = 'medium' THEN 50
        ELSE 100
    END,
    TRUE,
    1
FROM risk_rule_templates
WHERE is_system = TRUE
ON CONFLICT DO NOTHING;

-- ============================================================
-- 5. 更新迁移记录
-- ============================================================
INSERT INTO schema_migrations (version, description)
VALUES ('002_init_data', 'Initialize default data')
ON CONFLICT (version) DO NOTHING;

COMMIT;

-- ============================================================
-- 验证数据插入
-- ============================================================
DO $$
BEGIN
    RAISE NOTICE '============================================';
    RAISE NOTICE '数据初始化完成!';
    RAISE NOTICE '============================================';
    RAISE NOTICE '管理员账户:';
    RAISE NOTICE '  - 用户名: admin, 密码: admin123 (超级管理员)';
    RAISE NOTICE '  - 用户名: operator, 密码: admin123 (操作员)';
    RAISE NOTICE '';
    RAISE NOTICE '系统配置: % 个配置项已初始化', (SELECT COUNT(*) FROM system_configs);
    RAISE NOTICE '风控规则模板: % 个模板已初始化', (SELECT COUNT(*) FROM risk_rule_templates);
    RAISE NOTICE '风控规则: % 个规则已初始化', (SELECT COUNT(*) FROM risk_rules);
    RAISE NOTICE '============================================';
    RAISE NOTICE '重要: 请在生产环境中修改默认密码!';
    RAISE NOTICE '============================================';
END $$;
