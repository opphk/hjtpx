-- CaptchaX Database Schema Migration
-- Version: 009_roles_permissions
-- Description: Create roles and permissions tables for multi-admin RBAC system
-- Created: 2026-05-15

BEGIN;

-- permissions: Permission definitions
CREATE TABLE IF NOT EXISTS permissions (
    id SERIAL PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    category VARCHAR(50) NOT NULL DEFAULT 'general',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_permissions_code ON permissions(code);
CREATE INDEX idx_permissions_category ON permissions(category);

-- Insert default permissions
INSERT INTO permissions (code, name, description, category) VALUES
    ('read', '读取', '查看数据', 'general'),
    ('write', '写入', '创建和编辑数据', 'general'),
    ('delete', '删除', '删除数据', 'general'),
    ('manage_users', '用户管理', '管理用户账号', 'admin'),
    ('manage_admins', '管理员管理', '管理系统管理员', 'admin'),
    ('manage_roles', '角色管理', '管理系统角色和权限', 'admin'),
    ('manage_config', '配置管理', '修改系统配置', 'config'),
    ('view_logs', '查看日志', '访问系统日志', 'logs'),
    ('export_data', '导出数据', '导出系统数据', 'data'),
    ('manage_whitelist', '白名单管理', '管理IP白名单', 'security'),
    ('manage_blacklist', '黑名单管理', '管理IP黑名单', 'security')
ON CONFLICT (code) DO NOTHING;

-- roles: Role definitions
CREATE TABLE IF NOT EXISTS roles (
    id SERIAL PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_system BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_roles_code ON roles(code);

-- Insert default roles
INSERT INTO roles (code, name, description, is_system) VALUES
    ('super_admin', '超级管理员', '拥有系统所有权限，可管理系统所有功能', TRUE),
    ('admin', '管理员', '拥有大部分管理权限，可管理日常运营', TRUE),
    ('operator', '操作员', '负责日常运营操作，如查看统计、管理黑白名单', TRUE),
    ('viewer', '访客', '仅可查看数据，无修改权限', TRUE)
ON CONFLICT (code) DO NOTHING;

-- role_permissions: Many-to-many relationship between roles and permissions
CREATE TABLE IF NOT EXISTS role_permissions (
    id SERIAL PRIMARY KEY,
    role_id INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id INTEGER NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(role_id, permission_id)
);

CREATE INDEX idx_role_permissions_role ON role_permissions(role_id);
CREATE INDEX idx_role_permissions_permission ON role_permissions(permission_id);

-- Assign permissions to default roles
-- Super Admin: All permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.code = 'super_admin'
ON CONFLICT DO NOTHING;

-- Admin: Most permissions except role management
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.code = 'admin' AND p.code NOT IN ('manage_admins', 'manage_roles')
ON CONFLICT DO NOTHING;

-- Operator: Read, write, manage whitelist/blacklist, view logs, export data
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.code = 'operator' AND p.code IN ('read', 'write', 'manage_whitelist', 'manage_blacklist', 'view_logs', 'export_data')
ON CONFLICT DO NOTHING;

-- Viewer: Read only
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.code = 'viewer' AND p.code IN ('read', 'view_logs')
ON CONFLICT DO NOTHING;

-- admin_roles: Many-to-many relationship between admins and roles
CREATE TABLE IF NOT EXISTS admin_roles (
    id SERIAL PRIMARY KEY,
    admin_id INTEGER NOT NULL REFERENCES admins(id) ON DELETE CASCADE,
    role_id INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(admin_id, role_id)
);

CREATE INDEX idx_admin_roles_admin ON admin_roles(admin_id);
CREATE INDEX idx_admin_roles_role ON admin_roles(role_id);

-- Migrate existing admins to have a default role
INSERT INTO admin_roles (admin_id, role_id)
SELECT a.id, r.id FROM admins a, roles r
WHERE r.code = 'admin' AND a.role = 'admin'
ON CONFLICT DO NOTHING;

INSERT INTO admin_roles (admin_id, role_id)
SELECT a.id, r.id FROM admins a, roles r
WHERE r.code = 'operator' AND a.role = 'operator'
ON CONFLICT DO NOTHING;

INSERT INTO admin_roles (admin_id, role_id)
SELECT a.id, r.id FROM admins a, roles r
WHERE r.code = 'viewer' AND a.role = 'viewer'
ON CONFLICT DO NOTHING;

-- Record this migration
INSERT INTO schema_migrations (version) VALUES ('009_roles_permissions')
ON CONFLICT (version) DO NOTHING;

COMMIT;
