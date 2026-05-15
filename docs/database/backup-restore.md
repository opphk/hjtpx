# HJTPX 数据库备份恢复完全指南

本文档提供 HJTPX 项目数据库备份恢复系统的完整技术文档，包括完整备份、增量备份、备份验证、恢复演练等功能的详细说明。

## 目录

- [1. 备份系统架构](#1-备份系统架构)
- [2. 备份脚本详解](#2-备份脚本详解)
- [3. 备份策略配置](#3-备份策略配置)
- [4. 恢复操作指南](#4-恢复操作指南)
- [5. 备份验证](#5-备份验证)
- [6. 恢复演练](#6-恢复演练)
- [7. 定时任务配置](#7-定时任务配置)
- [8. 故障排除](#8-故障排除)
- [9. 最佳实践](#9-最佳实践)

---

## 1. 备份系统架构

### 1.1 备份类型

```
┌─────────────────────────────────────────────────────────────┐
│                    HJTPX 备份系统                            │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐   │
│  │  完整备份    │    │  增量备份    │    │  WAL归档     │   │
│  │ (pg_dump)   │    │(pg_basebackup)│   │ (连续归档)   │   │
│  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘   │
│         │                   │                   │           │
│         └───────────────────┼───────────────────┘           │
│                             │                               │
│                    ┌────────▼────────┐                      │
│                    │    备份存储      │                      │
│                    │  /var/backups/  │                      │
│                    └─────────────────┘                      │
│                             │                               │
│         ┌────────────────────┼────────────────────┐         │
│         │                    │                    │         │
│    ┌────▼─────┐        ┌────▼─────┐        ┌────▼────┐    │
│    │   full   │        │incr/     │        │  wal    │    │
│    │          │        │base      │        │archive  │    │
│    └──────────┘        └──────────┘        └─────────┘    │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 备份目录结构

```
/var/backups/hjtpx/
├── full/                    # 完整数据库备份
│   ├── db_20240101_020000.sql.gz
│   ├── db_20240108_020000.sql.gz
│   └── latest.txt           # 最新备份链接
├── incremental/             # 增量备份
│   ├── db_incr_20240102_020000.sql.gz
│   └── db_incr_20240103_020000.sql.gz
├── base/                    # 基础备份(pg_basebackup)
│   ├── base_backup_20240101_020000/
│   │   ├── 000000010000000000000001.gz
│   │   └── backup_label
│   └── latest -> base_backup_20240101_020000
├── wal/                     # WAL文件
├── archive/                 # WAL归档
│   └── archive_20240101_020000/
├── redis/                   # Redis备份
│   └── redis_20240101_020000.rdb.gz
├── config/                  # 配置备份
│   └── config_20240101_020000.tar.gz
└── backup_metadata.json     # 备份元数据
```

### 1.3 支持的备份组件

| 组件 | 备份方式 | 默认保留 | 说明 |
|------|----------|----------|------|
| PostgreSQL | pg_dump (自定义格式) | 30天 | 完整数据库结构和数据 |
| PostgreSQL | pg_basebackup | 3个基础链 | 全量物理备份 |
| Redis | BGSAVE + RDB | 7天 | 内存数据持久化 |
| Config | tar.gz | 30天 | 配置文件和迁移脚本 |

---

## 2. 备份脚本详解

### 2.1 backup.sh - 主备份脚本

**位置**: `scripts/backup.sh`

**功能**:
- 完整数据库备份
- 增量备份
- Redis数据备份
- 配置文件备份
- 自动清理过期备份

**基本用法**:

```bash
./scripts/backup.sh [命令]
```

**可用命令**:

| 命令 | 说明 | 示例 |
|------|------|------|
| `full` | 完整备份（默认） | `./backup.sh full` |
| `incremental` | 增量备份 | `./backup.sh incremental` |
| `db-only` | 仅数据库备份 | `./backup.sh db-only` |
| `redis` | 仅Redis备份 | `./backup.sh redis` |
| `config` | 仅配置备份 | `./backup.sh config` |
| `cleanup` | 清理过期备份 | `./backup.sh cleanup` |
| `list` | 列出所有备份 | `./backup.sh list` |

**环境变量**:

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `BACKUP_DIR` | 备份存储目录 | `/var/backups/hjtpx` |
| `DB_HOST` | 数据库主机 | `localhost` |
| `DB_PORT` | 数据库端口 | `5432` |
| `DB_NAME` | 数据库名 | `hjtpx` |
| `DB_USER` | 数据库用户 | `postgres` |
| `DB_PASSWORD` | 数据库密码 | `postgres` |
| `REDIS_HOST` | Redis主机 | `localhost` |
| `REDIS_PORT` | Redis端口 | `6379` |
| `FULL_RETENTION_DAYS` | 完整备份保留天数 | `30` |
| `INCR_RETENTION_DAYS` | 增量备份保留天数 | `7` |
| `REDIS_RETENTION_DAYS` | Redis备份保留天数 | `7` |
| `CONFIG_RETENTION_DAYS` | 配置备份保留天数 | `30` |
| `AUTO_CLEANUP` | 是否自动清理 | `true` |

**示例**:

```bash
# 完整备份
./backup.sh full

# 仅备份数据库
./backup.sh db-only

# 自定义备份目录
BACKUP_DIR=/data/backups ./backup.sh full

# 自定义保留策略
FULL_RETENTION_DAYS=7 INCR_RETENTION_DAYS=3 ./backup.sh full

# 禁用自动清理
AUTO_CLEANUP=false ./backup.sh full

# 列出所有备份
./backup.sh list

# 清理旧备份
./backup.sh cleanup
```

### 2.2 backup-incremental.sh - 增量备份脚本

**位置**: `scripts/backup-incremental.sh`

**功能**:
- 使用 pg_basebackup 进行基础备份
- WAL (Write-Ahead Logging) 连续归档
- 备份链管理
- 增量备份元数据追踪

**基本用法**:

```bash
./scripts/backup-incremental.sh [命令]
```

**可用命令**:

| 命令 | 说明 | 示例 |
|------|------|------|
| `base` | 创建基础备份 | `./backup-incremental.sh base` |
| `incremental` | 创建增量备份 | `./backup-incremental.sh incr` |
| `full` | 完整周期（基础+清理） | `./backup-incremental.sh full` |
| `verify` | 验证备份链 | `./backup-incremental.sh verify` |
| `list` | 列出备份链 | `./backup-incremental.sh list` |
| `size` | 显示备份大小统计 | `./backup-incremental.sh size` |
| `cleanup` | 清理旧备份 | `./backup-incremental.sh cleanup` |

**环境变量**:

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `RETAIN_CHAINS` | 保留的基础备份链数 | `3` |
| `WAL_DIR` | WAL文件目录 | `/var/lib/postgresql/data/pg_wal` |

**示例**:

```bash
# 创建基础备份
./backup-incremental.sh base

# 创建增量备份
./backup-incremental.sh incremental

# 完整备份周期
./backup-incremental.sh full

# 验证备份链完整性
./backup-incremental.sh verify

# 列出备份链
./backup-incremental.sh list

# 查看备份大小
./backup-incremental.sh size

# 清理旧备份（保留最近3个基础链）
RETAIN_CHAINS=3 ./backup-incremental.sh cleanup
```

### 2.3 verify-backup.sh - 备份验证脚本

**位置**: `scripts/verify-backup.sh`

**功能**:
- 验证备份文件完整性
- 检查备份可恢复性
- 生成验证报告

**基本用法**:

```bash
./scripts/verify-backup.sh [命令] [备份文件]
```

**可用命令**:

| 命令 | 说明 | 示例 |
|------|------|------|
| `latest` | 验证最新备份 | `./verify-backup.sh latest` |
| `all` | 验证所有备份 | `./verify-backup.sh all` |
| `specific <文件>` | 验证指定备份 | `./verify-backup.sh specific /path/to/backup` |

**示例**:

```bash
# 验证最新备份
./verify-backup.sh latest

# 验证所有备份
./verify-backup.sh all

# 验证指定备份文件
./verify-backup.sh specific /var/backups/hjtpx/full/db_20240101_020000.sql.gz
```

**验证报告格式**:

```json
{
  "timestamp": "2024-01-01T12:00:00Z",
  "total": 10,
  "passed": 9,
  "failed": 1,
  "results": [
    {
      "type": "full",
      "file": "/var/backups/hjtpx/full/db_20240101_020000.sql.gz",
      "status": "ok"
    }
  ]
}
```

### 2.4 restore-drill.sh - 恢复演练脚本

**位置**: `scripts/restore-drill.sh`

**功能**:
- 自动化恢复测试
- 临时数据库恢复验证
- 数据完整性检查
- 生成演练报告

**基本用法**:

```bash
./scripts/restore-drill.sh [命令]
```

**可用命令**:

| 命令 | 说明 | 示例 |
|------|------|------|
| `full` | 运行完整恢复演练 | `./restore-drill.sh full` |

**示例**:

```bash
# 运行恢复演练
./restore-drill.sh full
```

**演练流程**:

1. 查找最新的完整备份
2. 创建临时数据库
3. 恢复备份到临时数据库
4. 验证数据完整性
5. 检查关键表
6. 清理临时数据库
7. 生成演练报告

**演练报告格式**:

```json
{
  "drill_id": "20240101_120000",
  "status": "success",
  "timestamp": "2024-01-01T12:00:00Z",
  "total_duration_seconds": 120,
  "backup_used": "/var/backups/hjtpx/full/db_20240101_020000.sql.gz",
  "steps": [
    {
      "step": "check_backup",
      "status": "ok",
      "backup": "/var/backups/hjtpx/full/db_20240101_020000.sql.gz"
    },
    {
      "step": "check_connection",
      "status": "ok"
    },
    {
      "step": "create_database",
      "status": "ok"
    },
    {
      "step": "restore_backup",
      "status": "ok",
      "duration_seconds": 95
    },
    {
      "step": "verify_integrity",
      "status": "ok",
      "duration_seconds": 20
    },
    {
      "step": "cleanup",
      "status": "ok"
    }
  ]
}
```

---

## 3. 备份策略配置

### 3.1 推荐备份策略

```
每周计划:
┌────────────────────────────────────────────────────────────┐
│  周日 02:00 - 完整备份 (full)                               │
│  周一 02:00 - 增量备份 (incremental)                        │
│  周二 02:00 - 增量备份 (incremental)                        │
│  周三 02:00 - 增量备份 (incremental)                        │
│  周四 02:00 - 增量备份 (incremental)                        │
│  周五 02:00 - 增量备份 (incremental)                        │
│  周六 02:00 - 增量备份 (incremental)                        │
│  周日 04:00 - 备份验证 (verify-backup.sh)                  │
│  周日 05:00 - 恢复演练 (restore-drill.sh)                  │
└────────────────────────────────────────────────────────────┘
```

### 3.2 保留策略

| 备份类型 | 保留时间 | 说明 |
|----------|----------|------|
| 完整备份 | 30天 | 每天创建一个完整备份 |
| 增量备份 | 7天 | 基于最新完整备份的增量 |
| Redis备份 | 7天 | 缓存数据备份 |
| 配置备份 | 30天 | 配置文件快照 |

### 3.3 环境变量配置

创建 `backup-env.conf`:

```bash
# 备份目录配置
BACKUP_DIR=/var/backups/hjtpx

# 数据库配置
DB_HOST=localhost
DB_PORT=5432
DB_NAME=hjtpx
DB_USER=postgres
DB_PASSWORD=your_secure_password

# Redis配置
REDIS_HOST=localhost
REDIS_PORT=6379

# 保留策略
FULL_RETENTION_DAYS=30
INCR_RETENTION_DAYS=7
REDIS_RETENTION_DAYS=7
CONFIG_RETENTION_DAYS=30

# 自动清理
AUTO_CLEANUP=true
```

使用配置文件运行备份:

```bash
source /path/to/backup-env.conf
./scripts/backup.sh full
```

---

## 4. 恢复操作指南

### 4.1 恢复前的准备

**重要提醒**: 
- 恢复操作会覆盖现有数据
- 建议在恢复前创建当前数据库的快照
- 通知相关人员暂停写入操作

### 4.2 完整恢复

#### 4.2.1 恢复最新备份

```bash
# 1. 停止应用服务
systemctl stop hjtpx

# 2. 创建当前数据库快照（可选）
pg_dump -h localhost -U postgres -d hjtpx -F c -f /tmp/pre_restore_snapshot.dump

# 3. 恢复最新备份
./scripts/restore.sh latest

# 4. 验证恢复
psql -h localhost -U postgres -d hjtpx -c "SELECT COUNT(*) FROM users;"

# 5. 重启应用服务
systemctl start hjtpx
```

#### 4.2.2 恢复指定备份

```bash
# 恢复指定日期的备份
./scripts/restore.sh full /var/backups/hjtpx/full/db_20240101_020000.sql.gz
```

#### 4.2.3 恢复到指定时间点

```bash
# 使用 pg_restore 的 --data-only 和时间点恢复
export PGPASSWORD='your_password'
pg_restore -h localhost -U postgres -d hjtpx \
    --data-only \
    --clean \
    /var/backups/hjtpx/full/db_20240101_020000.sql.gz

# 注意: 需要配合 WAL 归档实现精确时间点恢复
```

### 4.3 增量备份恢复

#### 4.3.1 恢复备份链

```bash
# 1. 识别完整备份链
./scripts/backup-incremental.sh list

# 2. 确定要恢复的基础备份
BASE_BACKUP="/var/backups/hjtpx/base/base_backup_20240101_020000"

# 3. 恢复基础备份
pg_restore -h localhost -U postgres -d hjtpx -c "${BASE_BACKUP}/database.tar.gz"

# 4. 按顺序恢复增量备份
for INCR in /var/backups/hjtpx/incremental/incr_backup_*; do
    pg_restore -a -h localhost -U postgres -d hjtpx "$INCR"
done
```

### 4.4 Redis恢复

```bash
# 1. 停止Redis
systemctl stop redis

# 2. 恢复RDB文件
cp /var/backups/hjtpx/redis/redis_20240101_020000.rdb.gz /tmp/
gunzip /tmp/redis_20240101_020000.rdb.gz

# 3. 替换Redis RDB文件
mv /tmp/redis_20240101_020000.rdb $(redis-cli CONFIG GET dir | tail -n 1)/dump.rdb

# 4. 重启Redis
systemctl start redis
```

### 4.5 配置恢复

```bash
# 1. 提取配置备份
tar -xzf /var/backups/hjtpx/config/config_20240101_020000.tar.gz -C /tmp/

# 2. 查看备份内容
tar -tzf /var/backups/hjtpx/config/config_20240101_020000.tar.gz

# 3. 恢复配置文件
cp /tmp/.env /workspace/hjtpx/.env
cp -r /tmp/config/* /workspace/hjtpx/config/

# 4. 重启应用使配置生效
systemctl restart hjtpx
```

### 4.6 测试恢复（不影响生产）

```bash
# 创建测试数据库并恢复
./scripts/restore.sh test

# 验证恢复的数据
psql -h localhost -U postgres -d hjtpx_drill_20240101_120000 -c "SELECT COUNT(*) FROM users;"

# 清理测试数据库
./scripts/restore.sh clean-temp
```

---

## 5. 备份验证

### 5.1 验证方法

#### 5.1.1 文件完整性检查

```bash
# 检查备份文件是否存在
ls -lh /var/backups/hjtpx/full/

# 检查文件大小
stat /var/backups/hjtpx/full/db_20240101_020000.sql.gz

# 验证压缩文件完整性
gunzip -t /var/backups/hjtpx/full/db_20240101_020000.sql.gz
```

#### 5.1.2 PostgreSQL备份验证

```bash
# 使用 pg_restore 验证
export PGPASSWORD='your_password'
pg_restore --list /var/backups/hjtpx/full/db_20240101_020000.sql.gz

# 统计备份中的对象数量
pg_restore --list /var/backups/hjtpx/full/db_20240101_020000.sql.gz | wc -l
```

#### 5.1.3 Redis备份验证

```bash
# 验证RDB文件
redis-cli -e 'DEBUG STRLEN dumpkey' < /var/backups/hjtpx/redis/redis_20240101_020000.rdb.gz
gunzip -t /var/backups/hjtpx/redis/redis_20240101_020000.rdb.gz
```

### 5.2 自动化验证

```bash
# 验证最新备份
./scripts/verify-backup.sh latest

# 验证所有备份
./scripts/verify-backup.sh all

# 验证结果
cat /workspace/hjtpx/logs/backup_verification_report_*.json | jq .
```

### 5.3 验证检查清单

- [ ] 备份文件存在
- [ ] 备份文件大小大于0
- [ ] 备份文件可解压
- [ ] PostgreSQL备份可被 pg_restore 识别
- [ ] Redis备份文件格式正确
- [ ] 配置备份包含所有必要文件
- [ ] 备份创建时间在预期范围内
- [ ] 备份元数据记录正确

---

## 6. 恢复演练

### 6.1 为什么要进行恢复演练

- 验证备份的可恢复性
- 测试恢复流程的完整性
- 评估恢复时间窗口
- 发现潜在问题
- 培训运维人员

### 6.2 演练频率

| 环境 | 演练频率 | 说明 |
|------|----------|------|
| 生产环境 | 每周 | 完整恢复演练 |
| 测试环境 | 每日 | 增量验证 |
| 开发环境 | 按需 | 功能验证 |

### 6.3 执行恢复演练

```bash
# 运行完整恢复演练
./scripts/restore-drill.sh full

# 查看演练报告
cat /workspace/hjtpx/logs/restore_drill_report_*.json | jq .

# 查看演练日志
tail -f /workspace/hjtpx/logs/restore_drill_*.log
```

### 6.4 演练检查清单

- [ ] 成功创建临时数据库
- [ ] 备份成功恢复到临时数据库
- [ ] 数据库表结构完整
- [ ] 数据行数符合预期
- [ ] 关键业务表存在
- [ ] 索引完整性验证
- [ ] 外键约束有效
- [ ] 临时数据库清理成功
- [ ] 演练日志完整

### 6.5 演练报告分析

检查演练报告中的以下指标:

- **总时长**: 是否在可接受的恢复时间窗口内
- **各步骤耗时**: 识别可能的瓶颈
- **数据完整性**: 行数、表数是否正确
- **错误信息**: 任何失败步骤都需要调查

---

## 7. 定时任务配置

### 7.1 Cron 配置

编辑 crontab:

```bash
crontab -e
```

添加以下配置:

```cron
# HJTPX 备份计划

# 完整备份 - 周日 02:00
0 2 * * 0 cd /workspace/hjtpx && ./scripts/backup.sh full >> /workspace/hjtpx/logs/backup_cron.log 2>&1

# 增量备份 - 周一至周六 02:00
0 2 * * 1-6 cd /workspace/hjtpx && ./scripts/backup.sh incremental >> /workspace/hjtpx/logs/backup_cron.log 2>&1

# 备份验证 - 每天 04:00
0 4 * * * cd /workspace/hjtpx && ./scripts/verify-backup.sh latest >> /workspace/hjtpx/logs/verify_cron.log 2>&1

# 恢复演练 - 周日 05:00
0 5 * * 0 cd /workspace/hjtpx && ./scripts/restore-drill.sh full >> /workspace/hjtpx/logs/drill_cron.log 2>&1
```

### 7.2 Systemd Timer 配置

**完整备份 Timer**:

```ini
# /etc/systemd/system/hjtpx-backup-full.timer
[Unit]
Description=HJTPX Full Backup Timer
Requires=hjtpx-backup-full.service

[Timer]
OnCalendar=Sun *-*-* 02:00:00
Persistent=true

[Install]
WantedBy=timers.target
```

**完整备份 Service**:

```ini
# /etc/systemd/system/hjtpx-backup-full.service
[Unit]
Description=HJTPX Full Backup
After=network.target postgresql.service redis.service

[Service]
Type=oneshot
WorkingDirectory=/workspace/hjtpx
ExecStart=/workspace/hjtpx/scripts/backup.sh full
User=postgres
Group=postgres
Environment=BACKUP_DIR=/var/backups/hjtpx
EnvironmentFile=/workspace/hjtpx/.env

[Install]
WantedBy=multi-user.target
```

启用定时器:

```bash
# 复制服务文件
sudo cp /workspace/hjtpx/scripts/backup-config/*.service /etc/systemd/system/
sudo cp /workspace/hjtpx/scripts/backup-config/*.timer /etc/systemd/system/

# 重新加载配置
sudo systemctl daemon-reload

# 启用定时器
sudo systemctl enable --now hjtpx-backup-full.timer
sudo systemctl enable --now hjtpx-backup-incremental.timer
sudo systemctl enable --now hjtpx-verify-backups.timer
sudo systemctl enable --now hjtpx-restore-drill.timer

# 查看定时器状态
sudo systemctl list-timers 'hjtpx-*'
```

### 7.3 监控备份任务

```bash
# 查看备份状态
systemctl status hjtpx-backup-full.service

# 查看备份日志
journalctl -u hjtpx-backup-full.service -n 50

# 查看下次执行时间
systemctl list-timers --all | grep hjtpx
```

---

## 8. 故障排除

### 8.1 常见问题

#### 8.1.1 备份失败

**症状**: 备份脚本执行失败

**排查步骤**:

1. 检查日志文件:
   ```bash
   tail -f /workspace/hjtpx/logs/backup_full.log
   ```

2. 验证数据库连接:
   ```bash
   export PGPASSWORD='your_password'
   psql -h localhost -U postgres -d hjtpx -c "SELECT 1;"
   ```

3. 检查备份目录权限:
   ```bash
   ls -la /var/backups/hjtpx/
   ```

4. 验证磁盘空间:
   ```bash
   df -h /var/backups
   ```

**常见原因及解决方案**:

| 问题 | 原因 | 解决方案 |
|------|------|----------|
| 连接失败 | 认证配置错误 | 检查 `pg_hba.conf` |
| 权限不足 | 文件权限问题 | `chmod 755 /var/backups/hjtpx` |
| 磁盘空间不足 | 存储已满 | 清理旧备份或扩展存储 |
| 备份文件为空 | pg_dump 失败 | 检查数据库状态 |

#### 8.1.2 恢复失败

**症状**: 恢复操作失败

**排查步骤**:

1. 验证备份文件完整性:
   ```bash
   ./scripts/verify-backup.sh specific /path/to/backup
   ```

2. 检查目标数据库状态:
   ```bash
   export PGPASSWORD='your_password'
   psql -h localhost -U postgres -d postgres -c "SELECT datname FROM pg_database;"
   ```

3. 确认磁盘空间:
   ```bash
   df -h /var/lib/postgresql
   ```

**常见原因及解决方案**:

| 问题 | 原因 | 解决方案 |
|------|------|----------|
| 备份文件损坏 | 压缩或传输错误 | 重新备份 |
| 目标数据库不存在 | 拼写错误 | 创建数据库 |
| 空间不足 | 恢复需要空间 | 扩展存储 |
| 连接被拒绝 | 权限问题 | 检查用户权限 |

#### 8.1.3 增量备份问题

**症状**: 增量备份无法正常工作

**排查步骤**:

1. 检查基础备份:
   ```bash
   ./scripts/backup-incremental.sh verify
   ```

2. 验证WAL配置:
   ```bash
   psql -h localhost -U postgres -d postgres -c "SHOW wal_level;"
   ```

3. 检查WAL归档:
   ```bash
   ls -la /var/lib/postgresql/data/pg_wal/
   ```

**解决方案**:

```bash
# 重新创建基础备份
./scripts/backup-incremental.sh base

# 检查WAL配置
psql -h localhost -U postgres -d postgres -c "ALTER SYSTEM SET wal_level = 'archive';"
psql -h localhost -U postgres -d postgres -c "ALTER SYSTEM SET archive_mode = 'on';"
psql -h localhost -U postgres -d postgres -c "ALTER SYSTEM SET archive_command = 'test ! -f /var/backups/hjtpx/wal/%f && cp %p /var/backups/hjtpx/wal/%f';"

# 重启PostgreSQL
systemctl restart postgresql
```

### 8.2 日志分析

**备份日志位置**: `/workspace/hjtpx/logs/`

| 日志文件 | 说明 |
|----------|------|
| `backup_*.log` | 主备份脚本日志 |
| `backup_incremental_*.log` | 增量备份日志 |
| `verify_backup_*.log` | 备份验证日志 |
| `restore_drill_*.log` | 恢复演练日志 |
| `backup_*.log` | 备份还原日志 |

**查看日志示例**:

```bash
# 查看最近的备份日志
tail -n 100 /workspace/hjtpx/logs/backup_*.log | tail -n 50

# 搜索错误
grep -i error /workspace/hjtpx/logs/backup_*.log

# 查看验证报告
cat /workspace/hjtpx/logs/backup_verification_report_*.json | jq .
```

### 8.3 性能优化

**备份性能调优**:

```bash
# 使用并行备份
pg_dump -h localhost -U postgres -d hjtpx -j 4 -F d -f /tmp/parallel_backup

# 调整压缩级别（平衡速度和大小）
pg_dump -h localhost -U postgres -d hjtpx -Z 6 -f backup.sql.gz
```

**恢复性能调优**:

```bash
# 使用并行恢复
pg_restore -h localhost -U postgres -d hjtpx -j 4 backup.dump

# 禁用触发器（快速导入）
pg_restore -h localhost -U postgres -d hjtpx --disable-triggers -1 backup.dump
```

---

## 9. 最佳实践

### 9.1 备份策略

1. **遵循 3-2-1 原则**:
   - 3 份数据副本
   - 2 种不同存储介质
   - 1 份异地备份

2. **定期验证备份**:
   - 每日验证最新备份
   - 每周完整验证
   - 定期恢复演练

3. **监控备份状态**:
   - 配置备份成功/失败告警
   - 监控备份文件大小变化
   - 监控磁盘空间使用

### 9.2 安全最佳实践

1. **加密敏感备份**:
   ```bash
   # 加密备份
   gpg --symmetric --cipher-algo AES256 backup.sql.gz
   
   # 解密备份
   gpg --decrypt backup.sql.gz.gpg > backup.sql.gz
   ```

2. **安全传输**:
   ```bash
   # SCP 安全传输
   scp -P 22 /var/backups/hjtpx/full/* user@remote-server:/backup/
   
   # S3 加密上传
   aws s3 cp /var/backups/hjtpx/full/* s3://my-bucket/hjtpx/ --sse AES256
   ```

3. **权限控制**:
   ```bash
   # 设置备份目录权限
   chmod 700 /var/backups/hjtpx
   chmod 600 /var/backups/hjtpx/*.sql.gz
   
   # 限制访问用户
   chown -R postgres:postgres /var/backups/hjtpx
   ```

### 9.3 自动化建议

1. **GitHub Actions 集成**:
   ```yaml
   # .github/workflows/backup-verify.yml
   name: Backup Verification
   
   on:
     schedule:
       - cron: '0 4 * * *'
   
   jobs:
     verify:
       runs-on: ubuntu-latest
       steps:
         - uses: actions/checkout@v2
         - name: Verify Backups
           run: |
             ./scripts/verify-backup.sh latest
   ```

2. **告警集成**:
   ```bash
   # 备份失败告警
   if [ $? -ne 0 ]; then
       curl -X POST "https://hooks.slack.com/services/xxx" \
         -d "{\"text\": \"HJTPX 备份失败!\"}"
   fi
   ```

### 9.4 灾难恢复计划

**恢复时间目标 (RTO)**: 4小时
**恢复点目标 (RPO)**: 24小时

**灾难恢复步骤**:

1. **宣布灾难**: 通知相关团队
2. **评估损失**: 确定数据丢失范围
3. **选择恢复点**: 选择最近的可用备份
4. **执行恢复**: 按优先级恢复组件
5. **验证恢复**: 确认数据完整性
6. **恢复服务**: 逐步恢复对外服务

---

## 附录

### A. 相关文件路径

| 文件 | 路径 |
|------|------|
| 主备份脚本 | `/workspace/hjtpx/scripts/backup.sh` |
| 增量备份脚本 | `/workspace/hjtpx/scripts/backup-incremental.sh` |
| 验证脚本 | `/workspace/hjtpx/scripts/verify-backup.sh` |
| 演练脚本 | `/workspace/hjtpx/scripts/restore-drill.sh` |
| 恢复脚本 | `/workspace/hjtpx/scripts/restore.sh` |
| 备份目录 | `/var/backups/hjtpx` |
| 日志目录 | `/workspace/hjtpx/logs` |

### B. 相关文档

- [PostgreSQL 备份文档](https://www.postgresql.org/docs/current/backup.html)
- [Redis RDB 备份文档](https://redis.io/topics/persistence)
- [pg_basebackup 文档](https://www.postgresql.org/docs/current/app-pgbasebackup.html)
- [Systemd Timer 文档](https://www.freedesktop.org/software/systemd/man/systemd.timer.html)

### C. 联系支持

如遇到问题，请检查:
1. 日志文件: `/workspace/hjtpx/logs/`
2. 配置文件: `/workspace/hjtpx/.env.production`
3. Systemd 日志: `journalctl -u hjtpx-*`
4. PostgreSQL 日志: `/var/log/postgresql/`

---

**文档版本**: 1.0.0  
**最后更新**: 2024-01-01  
**维护者**: HJTPX 运维团队
