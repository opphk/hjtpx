# 数据库迁移脚本使用说明

## 概述

本目录包含用于管理数据库迁移的完整脚本系统，支持创建表结构、初始化数据、验证数据库等操作。

## 目录结构

```
migrate/
├── 001_init.sql       # 表结构初始化脚本
├── init-data.sql       # 数据初始化脚本
└── migrate.sh         # 迁移管理脚本

verify-db.sh           # 数据库验证脚本（位于 scripts/ 目录）
```

## 快速开始

### 1. 环境配置

设置环境变量或使用命令行参数：

```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=your_password
export DB_NAME=hjtpx_db
```

### 2. 初始化数据库

```bash
cd /workspace/scripts/migrate
./migrate.sh init
```

这将：
- 创建数据库（如果不存在）
- 创建所有表结构
- 创建索引和约束
- 创建触发器

### 3. 初始化种子数据

```bash
./migrate.sh seed
```

这将：
- 创建默认管理员账户
- 初始化系统配置
- 初始化风控规则模板
- 创建默认风控规则

### 4. 验证数据库

```bash
cd /workspace/scripts
./verify-db.sh
```

### 5. 查看迁移状态

```bash
./migrate.sh status
```

## 迁移管理命令

### init - 初始化数据库

```bash
./migrate.sh init [选项]
```

选项：
- `--host` 数据库主机
- `--port` 数据库端口
- `--user` 数据库用户
- `--password` 数据库密码
- `--dbname` 数据库名称

示例：
```bash
./migrate.sh init --dbname myapp --host 192.168.1.100
```

### migrate - 执行迁移

```bash
./migrate.sh migrate
```

### seed - 初始化数据

```bash
./migrate.sh seed
```

### rollback - 回滚

```bash
./migrate.sh rollback
```

### reset - 重置数据库

```bash
./migrate.sh reset
```

## 数据库验证

### 快速验证

```bash
./verify-db.sh --quick
```

### 完整验证

```bash
./verify-db.sh --full
```

验证内容包括：
1. 数据库连接
2. 表结构
3. 索引
4. 约束
5. 数据完整性
6. 性能指标

## 表结构说明

### 核心业务表

| 表名 | 描述 |
|------|------|
| users | 用户表 |
| admins | 管理员表 |
| applications | 应用表 |
| verifications | 验证码记录表 |
| captcha_sessions | 验证码会话表 |
| risk_rules | 风控规则表 |
| risk_logs | 风控日志表 |
| audit_logs | 审计日志表 |

### 支持表

| 表名 | 描述 |
|------|------|
| blacklist | 黑名单表 |
| system_configs | 系统配置表 |
| device_fingerprints | 设备指纹表 |
| behavior_data | 行为数据表 |
| verification_logs | 验证日志表 |
| user_mfa | 用户MFA配置表 |
| admin_mfa | 管理员MFA配置表 |
| mfa_codes | MFA验证码表 |
| api_key_histories | API密钥历史表 |
| voice_captcha_sessions | 语音验证码会话表 |
| admin_login_logs | 管理员登录日志表 |
| trace_records | 验证轨迹记录表 |
| risk_rule_templates | 风控规则模板表 |
| risk_rule_trigger_histories | 规则触发历史表 |
| schema_migrations | 迁移记录表 |

## 默认账户

初始化后创建的默认账户：

- **用户名**: admin
- **密码**: admin123（请在生产环境修改）
- **角色**: 超级管理员

- **用户名**: operator
- **密码**: admin123
- **角色**: 操作员

## 常见问题

### Q: 如何修改数据库配置？

A: 可以通过环境变量或命令行参数修改：

```bash
# 环境变量
export DB_HOST=localhost
export DB_PASSWORD=secret

# 命令行参数
./migrate.sh init --host localhost --password secret
```

### Q: 迁移失败怎么办？

A: 检查以下内容：
1. 数据库服务是否运行
2. 账户权限是否足够
3. 数据库是否已存在冲突数据
4. 查看具体错误信息

### Q: 如何查看详细的迁移状态？

```bash
./migrate.sh status
```

### Q: 如何备份数据库？

A: 使用 PostgreSQL 的 pg_dump 工具：

```bash
pg_dump -h localhost -U postgres -d hjtpx_db > backup.sql
```

## 最佳实践

1. **定期备份**: 在执行迁移前备份数据库
2. **测试环境**: 先在测试环境验证迁移脚本
3. **权限控制**: 使用最小权限原则创建数据库用户
4. **监控**: 使用 verify-db.sh 定期检查数据库健康状态
5. **日志**: 保留迁移历史记录

## 扩展迁移

要添加新的迁移：

1. 创建新的 SQL 文件（如 `002_xxx.sql`）
2. 在文件开头添加迁移记录：
   ```sql
   INSERT INTO schema_migrations (version, description)
   VALUES ('002_xxx', '描述')
   ON CONFLICT (version) DO NOTHING;
   ```
3. 执行 `./migrate.sh migrate`
