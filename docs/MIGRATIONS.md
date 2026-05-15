# 数据库迁移指南

本文档描述了 HJTPX 项目的数据库迁移系统的使用方法和最佳实践。

## 目录

- [概述](#概述)
- [迁移文件结构](#迁移文件结构)
- [命令使用](#命令使用)
- [创建新迁移](#创建新迁移)
- [迁移历史](#迁移历史)
- [最佳实践](#最佳实践)

## 概述

本项目使用自定义的数据库迁移系统，支持：

- 版本化的数据库变更
- 向上迁移（up）和回滚（down）
- 迁移状态追踪
- 执行时间记录
- 文件完整性校验（checksum）

## 迁移文件结构

迁移文件位于 `migrations/` 目录，命名规范为：

```
{version}_{name}.{type}.sql
```

例如：

```
001_initial_schema.up.sql
001_initial_schema.down.sql
002_roles_permissions.up.sql
002_roles_permissions.down.sql
```

### 文件说明

- **version**: 数字版本号，用于排序和追踪
- **name**: 迁移的描述性名称
- **type**: `up` 表示迁移脚本，`down` 表示回滚脚本

### 当前迁移列表

| 版本 | 名称 | 描述 |
|------|------|------|
| 001 | initial_schema | 初始数据库架构（用户、会话、迁移表） |
| 002 | roles_permissions | 角色和权限管理系统 |
| 003 | notifications | 通知系统 |
| 004 | login_history | 登录历史和尝试追踪 |
| 005 | presence | 用户在线状态和在场管理 |
| 006 | analytics | 分析系统 |
| 007 | security | 安全功能（CSRF、审计日志、安全事件） |

## 命令使用

### 查看迁移状态

```bash
node scripts/migrate.js status
```

显示当前迁移版本和所有迁移的状态。

### 应用迁移

```bash
# 应用所有待处理的迁移
node scripts/migrate.js up

# 应用到特定版本
node scripts/migrate.js up 5
```

### 回滚迁移

```bash
# 回滚最后一个迁移
node scripts/migrate.js down

# 回滚指定数量的迁移
node scripts/migrate.js down 3

# 回滚到特定版本
node scripts/migrate.js down --to 2
```

### 创建新迁移

```bash
node scripts/migrate.js create <migration_name>
```

这会自动创建一对 `.up.sql` 和 `.down.sql` 文件。

## 创建新迁移

### 步骤

1. 使用 `create` 命令创建迁移文件
2. 编辑 `.up.sql` 文件，添加数据库变更
3. 编辑 `.down.sql` 文件，添加回滚逻辑
4. 测试迁移和回滚
5. 提交代码

### 示例

```bash
node scripts/migrate.js create add_email_column
```

这会创建：
- `migrations/008_add_email_column.up.sql`
- `migrations/008_add_email_column.down.sql`

**up.sql**:
```sql
-- Migration: add_email_column
-- Created: 2024-05-14
-- Description: Add email column to users table

ALTER TABLE users ADD COLUMN email VARCHAR(255);
CREATE INDEX idx_users_email ON users(email);
```

**down.sql**:
```sql
-- Rollback: add_email_column
-- Description: Remove email column from users table

DROP INDEX IF EXISTS idx_users_email;
ALTER TABLE users DROP COLUMN IF EXISTS email;
```

## 迁移历史

迁移信息存储在 `migrations` 表中：

| 列名 | 类型 | 描述 |
|------|------|------|
| id | SERIAL | 主键 |
| version | INTEGER | 迁移版本 |
| name | VARCHAR(255) | 迁移名称 |
| type | VARCHAR(20) | 类型：up/down |
| applied_at | TIMESTAMP | 应用时间 |
| execution_time_ms | INTEGER | 执行时间（毫秒） |
| status | VARCHAR(20) | 状态：success/failed |
| checksum | VARCHAR(64) | 文件SHA256校验和 |
| error_message | TEXT | 错误信息 |

## 最佳实践

### 1. 编写可逆的迁移

始终为每个迁移编写对应的回滚脚本，确保可以安全地回滚。

### 2. 保持迁移小型化

每个迁移应该只做一件事，便于测试和回滚。

### 3. 使用事务

迁移脚本中的 DDL 和 DML 操作已经被脚本包裹在事务中，但要注意某些数据库（如 PostgreSQL）对 DDL 事务的支持。

### 4. 测试迁移

在应用到生产环境前：
- 先在开发环境测试
- 测试迁移和回滚
- 检查数据完整性

### 5. 备份数据库

在生产环境执行迁移前，务必备份数据库。

### 6. 文档化变更

在迁移文件中添加详细的注释说明变更内容和原因。

### 7. 避免破坏性变更

除非绝对必要，避免删除表或列。可以先弃用，在后续版本中删除。

## 故障排除

### 迁移失败

如果迁移失败：
1. 检查错误信息
2. 查看 `migrations` 表中的状态
3. 修复问题
4. 重试迁移

### 数据库版本不一致

如果数据库版本和代码不匹配：
1. 使用 `status` 命令查看当前状态
2. 使用 `up` 或 `down` 命令调整到正确版本

### 手动修复

在极端情况下，可以：
1. 备份数据库
2. 手动应用/回滚 SQL
3. 更新 `migrations` 表

## NPM 脚本

为方便使用，可以在 `package.json` 中添加以下脚本：

```json
{
  "scripts": {
    "migrate:up": "node scripts/migrate.js up",
    "migrate:down": "node scripts/migrate.js down",
    "migrate:status": "node scripts/migrate.js status",
    "migrate:create": "node scripts/migrate.js create"
  }
}
```

然后可以使用：

```bash
npm run migrate:up
npm run migrate:down
npm run migrate:status
npm run migrate:create -- add_feature
```

## 相关文件

- `scripts/migrate.js`: 迁移脚本主文件
- `migrations/`: 迁移文件目录
- `.env`: 数据库配置（需要创建）
