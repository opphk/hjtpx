-- PostgreSQL 数据库初始化脚本
-- 注意：GORM AutoMigrate 会自动创建 models 包中定义的表结构
-- 此脚本仅用于：创建数据库、扩展、以及 GORM 不管理的辅助表

-- 创建数据库（需在 template1 或 postgres 数据库中执行）
-- CREATE DATABASE verification;

-- 创建 uuid 扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================
-- GORM 模型对应的表（由 AutoMigrate 自动管理，此处仅作参考）
-- ============================================================
-- users, admins, applications, api_key_histories,
-- verifications, behavior_data, verification_logs
--
-- 以上表由 GORM AutoMigrate 自动创建/更新，无需手动管理

-- ============================================================
-- 辅助表：GORM 不管理但业务需要的表
-- ============================================================

-- 安全审计日志表
CREATE TABLE IF NOT EXISTS security_logs (
    id BIGSERIAL PRIMARY KEY,
    event_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL DEFAULT 'info',
    client_ip VARCHAR(45),
    user_agent TEXT,
    request_path VARCHAR(500),
    request_method VARCHAR(10),
    request_body TEXT,
    error_message TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_security_logs_event_type ON security_logs (event_type);
CREATE INDEX IF NOT EXISTS idx_security_logs_severity ON security_logs (severity);
CREATE INDEX IF NOT EXISTS idx_security_logs_created_at ON security_logs (created_at DESC);

-- 速率限制记录表
CREATE TABLE IF NOT EXISTS rate_limit_records (
    id BIGSERIAL PRIMARY KEY,
    identifier VARCHAR(255) NOT NULL,
    identifier_type VARCHAR(20) NOT NULL,
    request_count INT NOT NULL DEFAULT 1,
    window_start TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE (identifier, identifier_type, window_start)
);

CREATE INDEX IF NOT EXISTS idx_rate_limit_identifier ON rate_limit_records (identifier);
CREATE INDEX IF NOT EXISTS idx_rate_limit_window_start ON rate_limit_records (window_start DESC);