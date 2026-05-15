# 数据库迁移指南

## 概述

本文档提供 HJTPX 项目的数据库迁移系统的完整指南，包括迁移策略、最佳实践、性能优化和故障排除。

## 目录

- [迁移系统架构](#迁移系统架构)
- [迁移命令使用](#迁移命令使用)
- [迁移追踪系统](#迁移追踪系统)
- [迁移文件结构](#迁移文件结构)
- [性能优化策略](#性能优化策略)
- [事务管理](#事务管理)
- [回滚策略](#回滚策略)
- [监控与追踪](#监控与追踪)
- [最佳实践](#最佳实践)
- [故障排除](#故障排除)

---

## 迁移系统架构

### 核心组件

| 组件 | 文件路径 | 功能 |
|------|----------|------|
| 迁移引擎 | `scripts/migrate.js` | 核心迁移执行引擎 |
| 迁移回滚 | `scripts/migrate-rollback.js` | 专业回滚脚本 |
| 迁移运行器 | `scripts/run-migrations.js` | 应用启动时自动运行 |
| 迁移脚本 | `migrations/*.sql` | SQL 迁移文件 |
| 迁移追踪器 | `migrations` 表 | 记录迁移历史 |
| 增强追踪 | `migration_tracking` 表 | 详细追踪信息 |

### 迁移流程

```
1. 应用启动 / 手动执行命令
   ↓
2. migrate.js 读取 migrations/ 目录
   ↓
3. 与数据库 migrations 表对比
   ↓
4. 确定待执行迁移
   ↓
5. 按顺序执行迁移（带事务）
   ↓
6. 记录执行结果到 migrations 表
   ↓
7. 记录详细追踪到 migration_tracking 表
   ↓
8. 完成/报告错误
```

---

## 迁移命令使用

### 基本命令

```bash
# 查看迁移状态
node scripts/migrate.js status

# 应用所有待处理迁移
node scripts/migrate.js up

# 应用到指定版本
node scripts/migrate.js up 5

# 回滚最后一个迁移
node scripts/migrate.js down

# 回滚多个迁移
node scripts/migrate.js down 3

# 回滚到指定版本
node scripts/migrate.js down --to 2

# 创建新迁移
node scripts/migrate.js create <migration_name>

# 查看追踪历史
node scripts/migrate.js tracking

# 健康检查
node scripts/migrate.js health

# 统计信息
node scripts/migrate.js stats
```

### 回滚专用命令

```bash
# 查看回滚计划
node scripts/migrate-rollback.js plan

# 回滚单个迁移
node scripts/migrate-rollback.js rollback

# 回滚多个迁移
node scripts/migrate-rollback.js rollback 3

# 回滚到指定版本
node scripts/migrate-rollback.js rollback --to 5

# 预览回滚（不执行）
node scripts/migrate-rollback.js rollback --dry-run

# 强制回滚（跳过确认）
node scripts/migrate-rollback.js rollback --force

# 查看当前状态
node scripts/migrate-rollback.js status
```

### NPM 脚本

在 `package.json` 中配置：

```json
{
  "scripts": {
    "migrate": "node scripts/migrate.js",
    "migrate:up": "node scripts/migrate.js up",
    "migrate:down": "node scripts/migrate.js down",
    "migrate:status": "node scripts/migrate.js status",
    "migrate:tracking": "node scripts/migrate.js tracking",
    "migrate:rollback": "node scripts/migrate-rollback.js rollback",
    "migrate:plan": "node scripts/migrate-rollback.js plan"
  }
}
```

---

## 迁移追踪系统

### 迁移追踪表结构

`migration_tracking` 表提供详细的迁移追踪信息：

```sql
CREATE TABLE migration_tracking (
  migration_id SERIAL PRIMARY KEY,
  migration_name VARCHAR(255) NOT NULL,
  migration_version INTEGER NOT NULL,
  migration_hash VARCHAR(64) NOT NULL,
  executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  rollback_executed_at TIMESTAMP,
  executed_by VARCHAR(255),
  status VARCHAR(20) NOT NULL DEFAULT 'pending',
  execution_time_ms INTEGER,
  error_message TEXT,
  machine_name VARCHAR(255),
  database_name VARCHAR(100)
);
```

### 追踪状态

| 状态 | 描述 |
|------|------|
| `pending` | 待执行 |
| `executed` | 已执行 |
| `rolled_back` | 已回滚 |
| `failed` | 执行失败 |

### 查询追踪信息

```sql
-- 查看所有追踪记录
SELECT * FROM migration_tracking ORDER BY migration_version;

-- 查看特定版本追踪
SELECT * FROM migration_tracking WHERE migration_version = 5;

-- 查看最近执行的迁移
SELECT * FROM migration_tracking 
WHERE status = 'executed' 
ORDER BY executed_at DESC 
LIMIT 10;

-- 查看所有回滚记录
SELECT * FROM migration_tracking 
WHERE status = 'rolled_back' 
ORDER BY rollback_executed_at DESC;
```

---

## 迁移文件结构

### 命名规范

```
{版本号}_{迁移名称}.{方向}.sql
```

示例：
```
001_initial_schema.up.sql
001_initial_schema.down.sql
002_roles_permissions.up.sql
002_roles_permissions.down.sql
```

### 版本号规则

- 使用三位数字（001, 002, ...）
- 按顺序递增
- 不重复使用版本号

### 迁移文件模板

**up.sql**
```sql
-- Migration: {迁移名称}
-- Created: {日期}
-- Description: {迁移描述}
-- Author: {作者}
-- Performance Notes: {性能相关说明}

-- 迁移 SQL 语句
```

**down.sql**
```sql
-- Rollback: {迁移名称}
-- Description: {回滚描述}
-- Author: {作者}

-- 回滚 SQL 语句
```

---

## 性能优化策略

### 1. 索引优化

#### 使用 IF NOT EXISTS

```sql
-- 好：幂等性索引创建
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- 差：重复执行会报错
CREATE INDEX idx_users_email ON users(email);
```

#### 部分索引（Partial Indexes）

```sql
-- 只索引活跃用户，减少索引大小
CREATE INDEX idx_users_active ON users(email)
  WHERE is_active = true;

-- 只索引有效会话
CREATE INDEX idx_sessions_valid ON sessions(user_id, expires_at)
  WHERE is_revoked = false AND expires_at > CURRENT_TIMESTAMP;
```

#### 复合索引（Composite Indexes）

```sql
-- 优化常见查询模式
CREATE INDEX idx_users_email_active ON users(email, is_active);

-- 支持多列排序
CREATE INDEX idx_users_created_role ON users(created_at DESC, role);
```

#### CONCURRENTLY 选项

> ⚠️ 重要：对于生产环境的大表，使用 CONCURRENTLY 选项避免锁表

```sql
-- 生产环境使用
CREATE INDEX CONCURRENTLY idx_large_table_column ON large_table(column);

-- 注意：不能在事务中使用 CONCURRENTLY
-- 这会导致错误：
-- ERROR: CREATE INDEX CONCURRENTLY cannot run inside a transaction block
```

### 2. 大表优化

#### 创建表时的优化

```sql
-- 使用合适的表空间
CREATE TABLE users (...) TABLESPACE postgres_tablespace;

-- 合适的填充因子
CREATE TABLE events (
  ...
) WITH (fillfactor = 70);
```

#### 分区表

对于超大型表，考虑使用分区：

```sql
-- 创建分区表
CREATE TABLE events (
  id SERIAL,
  event_time TIMESTAMP,
  data JSONB
) PARTITION BY RANGE (event_time);

-- 创建月度分区
CREATE TABLE events_2026_01 PARTITION OF events
  FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
```

### 3. 批量操作优化

```sql
-- 使用 COPY 代替 INSERT（批量插入）
COPY users (email, name) FROM stdin;
user1@example.com User One
user2@example.com User Two
\.

-- 批量更新使用 IN 子句
UPDATE users
SET status = 'inactive'
WHERE id IN (1, 2, 3, 4, 5);

-- 避免在循环中执行单个 INSERT
-- 好：批量插入
INSERT INTO audit_logs (user_id, action)
SELECT user_id, 'batch_action'
FROM users WHERE status = 'pending';

-- 差：逐行插入
DO $$
DECLARE
  user_record RECORD;
BEGIN
  FOR user_record IN SELECT user_id FROM users WHERE status = 'pending' LOOP
    INSERT INTO audit_logs (user_id, action) VALUES (user_record.user_id, 'batch_action');
  END LOOP;
END $$;
```

### 4. 锁管理

```sql
-- 监控锁
SELECT blocked_locks.pid     AS blocked_pid,
       blocked_activity.usename  AS blocked_user,
       blocking_locks.pid     AS blocking_pid,
       blocking_activity.usename AS blocking_user,
       blocked_activity.query    AS blocked_statement,
       blocking_activity.query   AS blocking_statement
FROM  pg_catalog.pg_locks         blocked_locks
JOIN  pg_catalog.pg_stat_activity blocked_activity ON blocked_activity.pid = blocked_locks.pid
JOIN  pg_catalog.pg_locks         blocking_locks
    ON blocking_locks.locktype = blocked_locks.locktype
    AND blocking_locks.database IS NOT DISTINCT FROM blocked_locks.database
    AND blocking_locks.relation IS NOT DISTINCT FROM blocked_locks.relation
    AND blocking_locks.page IS NOT DISTINCT FROM blocked_locks.page
    AND blocking_locks.tuple IS NOT DISTINCT FROM blocked_locks.tuple
    AND blocking_locks.virtualxid IS NOT DISTINCT FROM blocked_locks.virtualxid
    AND blocking_locks.transactionid IS NOT DISTINCT FROM blocked_locks.transactionid
    AND blocking_locks.classid IS NOT DISTINCT FROM blocked_locks.classid
    AND blocking_locks.objid IS NOT DISTINCT FROM blocked_locks.objid
    AND blocking_locks.objsubid IS NOT DISTINCT FROM blocked_locks.objsubid
    AND blocking_locks.pid != blocked_locks.pid
JOIN  pg_catalog.pg_stat_activity blocking_activity ON blocking_activity.pid = blocking_locks.pid
WHERE NOT blocked_locks.granted;
```

---

## 事务管理

### 自动事务包装

`migrate.js` 自动为每个迁移包装事务：

```javascript
async function executeSqlFile(pool, filePath) {
  const sql = fs.readFileSync(filePath, 'utf8');
  
  const client = await pool.connect();
  try {
    await client.query('BEGIN');
    const startTime = Date.now();
    await client.query(sql);
    const executionTime = Date.now() - startTime;
    await client.query('COMMIT');
    return { success: true, executionTime };
  } catch (error) {
    await client.query('ROLLBACK');
    throw error;
  } finally {
    client.release();
  }
}
```

### 事务限制

> ⚠️ 重要：某些 PostgreSQL DDL 操作不能回滚

不能回滚的操作：
- CREATE INDEX（除非使用 DROP INDEX）
- CREATE TABLE
- ALTER TABLE ADD COLUMN

可以回滚的操作：
- DROP TABLE
- DROP INDEX
- ALTER TABLE DROP COLUMN
- INSERT/UPDATE/DELETE

### 迁移中的事务策略

```sql
-- 策略 1：整个迁移在事务中
BEGIN;
  ALTER TABLE users ADD COLUMN new_field VARCHAR(100);
  UPDATE users SET new_field = 'default';
  CREATE INDEX idx_new_field ON users(new_field);
COMMIT;

-- 策略 2：分步执行（如果需要独立回滚）
BEGIN;
  ALTER TABLE users ADD COLUMN new_field VARCHAR(100);
COMMIT;

BEGIN;
  UPDATE users SET new_field = 'default';
COMMIT;

BEGIN;
  CREATE INDEX CONCURRENTLY idx_new_field ON users(new_field);
COMMIT;
```

---

## 回滚策略

### 回滚原则

1. **可逆性优先**：每个迁移必须有对应的 down.sql
2. **幂等性**：回滚脚本可以安全地多次执行
3. **数据保护**：回滚不应删除有价值的业务数据
4. **索引优先**：先删除索引，再删除表或列

### 回滚执行顺序

```bash
# 回滚顺序示例
001_initial_schema.up.sql     → 应用
002_roles_permissions.up.sql  → 应用
002_roles_permissions.down.sql → 回滚
001_initial_schema.down.sql    → 回滚
```

### 使用专业回滚脚本

`migrate-rollback.js` 提供了增强的回滚功能：

```bash
# 1. 查看回滚计划（预览模式）
node scripts/migrate-rollback.js plan

# 2. 预览回滚操作（不执行）
node scripts/migrate-rollback.js rollback --dry-run

# 3. 执行回滚（带确认提示）
node scripts/migrate-rollback.js rollback

# 4. 强制回滚（跳过确认）
node scripts/migrate-rollback.js rollback --force

# 5. 回滚到特定版本
node scripts/migrate-rollback.js rollback --to 3

# 6. 查看当前迁移状态
node scripts/migrate-rollback.js status
```

### 回滚验证

回滚脚本会自动执行以下验证：

- 检查外键约束依赖
- 检查活跃数据库连接
- 验证回滚脚本存在
- 记录回滚执行时间和状态

### 安全回滚检查

```sql
-- 回滚前检查：是否有依赖的外键
SELECT
  tc.table_name,
  tc.constraint_name,
  kcu.column_name,
  ccu.table_name AS foreign_table_name,
  ccu.column_name AS foreign_column_name
FROM information_schema.table_constraints AS tc
JOIN information_schema.key_column_usage AS kcu
  ON tc.constraint_name = kcu.constraint_name
JOIN information_schema.constraint_column_usage AS ccu
  ON ccu.constraint_name = tc.constraint_name
WHERE tc.constraint_type = 'FOREIGN KEY'
  AND tc.table_name = 'users';

-- 回滚前检查：是否有活跃连接
SELECT pid, usename, application_name, state, query_start
FROM pg_stat_activity
WHERE datname = current_database()
  AND pid <> pg_backend_pid();
```

---

## 监控与追踪

### 迁移追踪表结构

```sql
-- 基本追踪表
CREATE TABLE migrations (
  id SERIAL PRIMARY KEY,
  version INTEGER NOT NULL UNIQUE,
  name VARCHAR(255) NOT NULL,
  type VARCHAR(20) NOT NULL DEFAULT 'up',
  applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  execution_time_ms INTEGER,
  status VARCHAR(20) DEFAULT 'success',
  checksum VARCHAR(64),
  error_message TEXT
);
```

### 增强追踪功能（008_migration_tracking）

#### 迁移锁表

```sql
CREATE TABLE migration_locks (
  lock_key VARCHAR(100) PRIMARY KEY,
  locked_by VARCHAR(255) NOT NULL,
  locked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  expires_at TIMESTAMP NOT NULL,
  is_active BOOLEAN DEFAULT true
);
```

#### 变更日志表

```sql
CREATE TABLE migration_changelog (
  id SERIAL PRIMARY KEY,
  migration_version INTEGER NOT NULL,
  migration_name VARCHAR(255) NOT NULL,
  action VARCHAR(20) NOT NULL,
  executed_by VARCHAR(255),
  executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  execution_time_ms INTEGER,
  rows_affected INTEGER,
  sql_statement TEXT,
  status VARCHAR(20) NOT NULL,
  error_details TEXT
);
```

### 查询迁移状态

```sql
-- 查看当前迁移状态
SELECT * FROM migrations
WHERE type = 'up'
ORDER BY version DESC
LIMIT 1;

-- 查看所有失败的迁移
SELECT * FROM migrations
WHERE status = 'failed'
ORDER BY applied_at DESC;

-- 查看迁移执行时间统计
SELECT
  name,
  AVG(execution_time_ms) as avg_time,
  MIN(execution_time_ms) as min_time,
  MAX(execution_time_ms) as max_time,
  COUNT(*) as execution_count
FROM migrations
GROUP BY name
ORDER BY avg_time DESC;
```

### 健康检查

```sql
-- 查看迁移健康状态
SELECT * FROM migration_health
ORDER BY check_time DESC
LIMIT 1;

-- 查看待执行迁移
SELECT * FROM pending_migrations;

-- 查看迁移追踪统计
SELECT
  COUNT(*) FILTER (WHERE status = 'executed') as executed,
  COUNT(*) FILTER (WHERE status = 'rolled_back') as rolled_back,
  COUNT(*) FILTER (WHERE status = 'failed') as failed,
  COUNT(*) FILTER (WHERE status = 'pending') as pending
FROM migration_tracking;
```

### 使用追踪命令

```bash
# 查看详细追踪历史
node scripts/migrate.js tracking

# 查看追踪统计
node scripts/migrate.js status
```

---

## 相关文档

### 1. 迁移开发

✅ **推荐**
```sql
-- 使用描述性名称
002_add_user_roles_and_permissions.up.sql

-- 添加详细注释
-- Migration: Add User Roles
-- Description: Creates role-based access control system
-- Author: development-team
-- Reviewers: senior-dev-1, senior-dev-2
```

❌ **避免**
```sql
-- 模糊的名称
002_update.up.sql

-- 无注释
ALTER TABLE users ADD COLUMN role VARCHAR(50);
```

### 2. 性能考虑

✅ **推荐**
```sql
-- 先创建表和基础结构
-- 后创建索引（允许并发）
CREATE INDEX CONCURRENTLY idx_user_email ON users(email);

-- 使用合适的索引类型
CREATE INDEX idx_range ON logs(created_at) WHERE event_type = 'error';
```

❌ **避免**
```sql
-- 在同一语句中创建多个索引
CREATE INDEX idx1 ON users(email);
CREATE INDEX idx2 ON users(name);
CREATE INDEX idx3 ON users(created_at);
-- 考虑批量创建或使用 CONCURRENTLY
```

### 3. 测试策略

```bash
# 1. 在开发环境测试
npm run migrate:up
npm run migrate:down

# 2. 验证数据完整性
SELECT COUNT(*) FROM users;

# 3. 测试回滚
npm run migrate:down
npm run migrate:up

# 4. 验证功能
npm test
```

### 4. 生产环境部署

```bash
# 1. 备份数据库
pg_dump -Fc hjtpx > backup_$(date +%Y%m%d_%H%M%S).dump

# 2. 在测试环境验证迁移
npm run migrate:up

# 3. 检查执行时间
npm run migrate:status

# 4. 在生产环境执行（低峰期）
npm run migrate:up

# 5. 监控错误日志
tail -f /var/log/postgresql/migration.log
```

### 5. 错误处理

```sql
-- 使用 DO BLOCK 进行复杂逻辑
DO $$
BEGIN
  -- 检查条件
  IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                 WHERE table_name = 'users' AND column_name = 'new_field') THEN
    -- 执行变更
    ALTER TABLE users ADD COLUMN new_field VARCHAR(100);
    RAISE NOTICE 'Column added successfully';
  ELSE
    RAISE NOTICE 'Column already exists, skipping';
  END IF;
END $$;
```

---

## 故障排除

### 问题 1：迁移锁等待

**症状**：
```
ERROR: could not obtain lock on row in relation "migrations"
```

**解决方案**：
```sql
-- 查看锁
SELECT * FROM migration_locks WHERE is_active = true;

-- 手动释放锁（谨慎使用）
DELETE FROM migration_locks WHERE lock_key = 'your_lock_key';
```

### 问题 2：迁移执行失败

**症状**：
```
ERROR: duplicate key value violates unique constraint
```

**解决方案**：
```sql
-- 检查迁移状态
SELECT * FROM migrations WHERE version = 3;

-- 如果迁移已部分执行，手动修复
UPDATE migrations SET status = 'failed' WHERE version = 3;
```

### 问题 3：回滚失败

**症状**：
```
ERROR: cannot drop table because other objects depend on it
```

**解决方案**：
```sql
-- 查看依赖
SELECT
  tc.table_schema,
  tc.table_name,
  ccu.table_schema AS foreign_table_schema,
  ccu.table_name AS foreign_table_name
FROM information_schema.table_constraints AS tc
JOIN information_schema.key_column_usage AS kcu
  ON tc.constraint_name = kcu.constraint_name
JOIN information_schema.constraint_column_usage AS ccu
  ON ccu.constraint_name = tc.constraint_name
WHERE tc.constraint_type = 'FOREIGN KEY'
  AND ccu.table_name = 'users';
```

### 问题 4：索引重建影响性能

**症状**：
迁移执行期间数据库响应慢

**解决方案**：
```sql
-- 使用 CONCURRENTLY（推荐）
CREATE INDEX CONCURRENTLY idx_new ON users(email);

-- 在低峰期执行
-- 使用 pg_center_activity 等工具监控
```

### 问题 5：数据丢失风险

**症状**：
担心回滚会删除重要数据

**预防措施**：
```sql
-- 回滚前备份关键数据
CREATE TABLE users_backup_$(date +%Y%m%d) AS SELECT * FROM users;

-- 使用软删除代替硬删除
-- 在回滚脚本中先转移数据
```

---

## 相关文档

- [Schema 文档](schema.md)
- [备份恢复指南](../BACKUP_RESTORE.md)
- [MIGRATIONS.md](../MIGRATIONS.md)

---

## 版本历史

| 版本 | 日期 | 作者 | 描述 |
|------|------|------|------|
| 1.0 | 2026-05-15 | Development Team | 初始文档 |
| 1.1 | 2026-05-15 | Development Team | 添加性能优化章节 |
