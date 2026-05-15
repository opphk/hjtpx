# CaptchaX 数据库迁移

完整的 CaptchaX 数据库迁移系统，支持 PostgreSQL 数据库的版本控制、回滚和管理。

## 目录结构

```
migrations/
├── 001_initial_schema.up.sql          # 初始架构
├── 001_initial_schema.down.sql        # 回滚初始架构
├── 002_optimize_indexes.up.sql        # 优化索引
├── 002_optimize_indexes.down.sql      # 回滚索引优化
├── 003_partition_tables.up.sql        # 表分区
├── 003_partition_tables.down.sql      # 回滚表分区
├── 004_cold_hot_separation.up.sql     # 冷热数据分离
├── 004_cold_hot_separation.down.sql   # 回滚冷热分离
├── 005_archiving_strategy.up.sql      # 归档策略
├── 005_archiving_strategy.down.sql    # 回滚归档策略
├── 006_optimize_queries.up.sql        # 查询优化
├── 006_optimize_queries.down.sql      # 回滚查询优化
├── 007_read_write_split.up.sql        # 读写分离
├── 007_read_write_split.down.sql      # 回滚读写分离
├── 008_monitoring_metrics.up.sql      # 监控指标
├── 008_monitoring_metrics.down.sql    # 回滚监控指标
├── 009_add_captcha_and_apps_tables.up.sql   # 添加captcha和apps表
├── 009_add_captcha_and_apps_tables.down.sql # 回滚captcha和apps表
├── migrate.sh                         # 迁移管理脚本
└── README.md                          # 本文档
```

## 快速开始

### 前置要求

- PostgreSQL 12+
- golang-migrate (自动安装)

### 使用迁移脚本

```bash
# 进入迁移目录
cd /workspace/captchax/migrations

# 设置数据库连接信息
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=captcha_db
export DB_USER=postgres
export DB_PASSWORD=your_password
export DB_SSLMODE=disable

# 运行所有迁移
./migrate.sh up

# 查看帮助
./migrate.sh help
```

## 迁移命令

### 基本命令

| 命令 | 描述 |
|------|------|
| `./migrate.sh up` | 应用所有迁移 |
| `./migrate.sh up N` | 应用 N 个迁移 |
| `./migrate.sh down` | 回滚所有迁移 |
| `./migrate.sh down N` | 回滚 N 个迁移 |
| `./migrate.sh goto V` | 迁移到指定版本 V |
| `./migrate.sh version` | 查看当前版本 |
| `./migrate.sh status` | 查看迁移状态 |
| `./migrate.sh create NAME` | 创建新迁移 |
| `./migrate.sh drop` | 删除所有表 |

### 使用示例

```bash
# 应用所有迁移
./migrate.sh up

# 只应用 1 个迁移
./migrate.sh up 1

# 回滚 1 个迁移
./migrate.sh down 1

# 迁移到版本 003
./migrate.sh goto 003

# 查看当前版本
./migrate.sh version

# 创建新的迁移
./migrate.sh create add_user_table
```

## 数据库表结构

### 核心表

#### 1. captcha - 验证码实例表
| 字段 | 类型 | 说明 |
|------|------|------|
| id | VARCHAR(36) | 主键，UUID |
| app_id | VARCHAR(64) | 应用ID |
| type | VARCHAR(20) | 验证码类型 |
| answer | VARCHAR(128) | 答案（加密） |
| image_data | TEXT | 图片数据 |
| status | INTEGER | 状态 |
| attempts | INTEGER | 尝试次数 |
| client_info | VARCHAR(512) | 客户端信息 |
| user_agent | VARCHAR(512) | User Agent |
| ip_address | VARCHAR(45) | IP地址 |
| expired_at | TIMESTAMP | 过期时间 |
| verified_at | TIMESTAMP | 验证时间 |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |
| deleted_at | TIMESTAMP | 软删除时间 |

#### 2. apps - 应用管理表
| 字段 | 类型 | 说明 |
|------|------|------|
| id | SERIAL | 主键 |
| app_id | VARCHAR(64) | 应用ID（唯一） |
| app_secret | VARCHAR(128) | 应用密钥 |
| name | VARCHAR(100) | 应用名称 |
| description | TEXT | 描述 |
| owner_id | INTEGER | 所有者ID（外键） |
| status | INTEGER | 状态 |
| domain | VARCHAR(255) | 域名 |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |
| deleted_at | TIMESTAMP | 软删除时间 |

#### 3. admins - 管理员表
| 字段 | 类型 | 说明 |
|------|------|------|
| id | SERIAL | 主键 |
| username | VARCHAR(50) | 用户名（唯一） |
| password_hash | VARCHAR(128) | 密码哈希 |
| email | VARCHAR(100) | 邮箱 |
| nickname | VARCHAR(100) | 昵称 |
| role | VARCHAR(20) | 角色 |
| status | INTEGER | 状态 |
| last_login_at | TIMESTAMP | 最后登录时间 |
| last_login_ip | VARCHAR(45) | 最后登录IP |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |
| deleted_at | TIMESTAMP | 软删除时间 |

#### 4. captcha_logs - 验证码日志表
| 字段 | 类型 | 说明 |
|------|------|------|
| id | SERIAL | 主键 |
| captcha_type | VARCHAR(20) | 验证码类型 |
| client_id | VARCHAR(64) | 客户端ID |
| ip | VARCHAR(45) | IP地址 |
| user_agent | TEXT | User Agent |
| result | BOOLEAN | 验证结果 |
| duration | INTEGER | 耗时（毫秒） |
| risk_score | INTEGER | 风险分数 |
| created_at | TIMESTAMP | 创建时间 |

#### 5. blacklist - 黑名单表
| 字段 | 类型 | 说明 |
|------|------|------|
| id | SERIAL | 主键 |
| ip | VARCHAR(45) | IP地址 |
| reason | TEXT | 原因 |
| expire_at | TIMESTAMP | 过期时间 |
| created_at | TIMESTAMP | 创建时间 |

#### 6. whitelist - 白名单表
| 字段 | 类型 | 说明 |
|------|------|------|
| id | SERIAL | 主键 |
| ip | VARCHAR(45) | IP地址 |
| domain | VARCHAR(255) | 域名 |
| reason | TEXT | 原因 |
| created_at | TIMESTAMP | 创建时间 |

#### 7. captcha_config - 配置表
| 字段 | 类型 | 说明 |
|------|------|------|
| id | SERIAL | 主键 |
| key | VARCHAR(100) | 配置键（唯一） |
| value | TEXT | 配置值 |
| description | TEXT | 描述 |
| updated_at | TIMESTAMP | 更新时间 |

## 默认数据

### 默认管理员账号

| 用户名 | 密码 | 角色 |
|--------|------|------|
| admin | admin123 | super |

### 默认配置

| 配置键 | 默认值 | 说明 |
|--------|--------|------|
| max_attempts_per_ip | 10 | 每IP最大尝试次数 |
| block_duration_minutes | 30 | 阻止时长（分钟） |
| risk_threshold | 70 | 风险阈值 |
| session_timeout_seconds | 300 | 会话超时（秒） |
| enable_whitelist | true | 启用白名单 |
| enable_blacklist | true | 启用黑名单 |

### 默认测试应用

| app_id | app_secret | name | domain |
|--------|------------|------|--------|
| test_app_001 | test_secret_123 | Test Application | localhost |

## 高级功能

### 表分区

`captcha_logs` 表按时间进行分区，支持按月/季度分区，提高查询性能。

### 冷热数据分离

- 热数据：最近90天数据在主表
- 冷数据：归档到 `captcha_logs_archive` 表

### 索引优化

- 复合索引支持常用查询模式
- 部分索引优化特定场景
- 覆盖索引减少回表

### 监控指标

内置多种监控视图和函数：
- `v_table_sizes` - 表大小统计
- `v_index_usage` - 索引使用情况
- `v_query_performance` - 查询性能
- `v_maintenance_alerts` - 维护告警

## 使用 Golang 代码迁移

除了使用命令行工具，还可以在 Go 代码中集成迁移功能：

```go
package main

import (
    "database/sql"
    "fmt"
    "log"

    "github.com/golang-migrate/migrate/v4"
    "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
    _ "github.com/lib/pq"
)

func main() {
    db, err := sql.Open("postgres", "postgres://user:pass@localhost:5432/db?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }

    driver, err := postgres.WithInstance(db, &postgres.Config{})
    if err != nil {
        log.Fatal(err)
    }

    m, err := migrate.NewWithDatabaseInstance(
        "file:///path/to/migrations",
        "postgres", driver)
    if err != nil {
        log.Fatal(err)
    }

    // 执行迁移
    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        log.Fatal(err)
    }

    fmt.Println("Migrations applied successfully!")
}
```

## Docker 部署

### 使用 Docker Compose

```yaml
version: '3.8'
services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: captcha_db
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  migrate:
    image: migrate/migrate
    depends_on:
      - postgres
    volumes:
      - ./migrations:/migrations
    command: [
      "-path", "/migrations",
      "-database", "postgres://postgres:postgres@postgres:5432/captcha_db?sslmode=disable",
      "up"
    ]

volumes:
  postgres_data:
```

## 故障排查

### 常见问题

**Q: 迁移卡住了怎么办？**
A: 使用 `force` 命令修复版本：
```bash
./migrate.sh force <version>
```

**Q: 如何查看当前版本？**
A: 运行：
```bash
./migrate.sh version
```

**Q: 迁移失败怎么办？**
A: 检查数据库连接和权限，确保 PostgreSQL 正常运行。

### 数据库连接测试

```bash
psql -h localhost -U postgres -d captcha_db
```

## 开发指南

### 创建新迁移

1. 使用脚本创建：
```bash
./migrate.sh create feature_name
```

2. 编辑生成的 `.up.sql` 和 `.down.sql` 文件

3. 测试迁移：
```bash
./migrate.sh up 1
./migrate.sh down 1
```

### 迁移文件格式

```sql
-- migration_name.up.sql
BEGIN;

-- 你的 SQL 语句

-- 记录迁移
INSERT INTO schema_migrations (version) VALUES ('010_migration_name')
ON CONFLICT (version) DO NOTHING;

COMMIT;
```

```sql
-- migration_name.down.sql
BEGIN;

-- 删除迁移记录
DELETE FROM schema_migrations WHERE version = '010_migration_name';

-- 你的回滚 SQL 语句

COMMIT;
```

## 许可证

MIT License
