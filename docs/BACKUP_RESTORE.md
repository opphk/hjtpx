# HJTPX 数据库备份与恢复指南

本文档介绍 HJTPX 项目的完整数据库备份与恢复解决方案，包括自动备份、增量备份、备份验证和恢复演练等功能。

## 目录

- [概述](#概述)
- [备份系统架构](#备份系统架构)
- [快速开始](#快速开始)
- [备份脚本使用](#备份脚本使用)
- [恢复脚本使用](#恢复脚本使用)
- [备份验证](#备份验证)
- [恢复演练](#恢复演练)
- [定时任务配置](#定时任务配置)
- [故障排除](#故障排除)

## 概述

HJTPX 备份系统提供以下功能:

1. **完整备份** - 每周日凌晨 2 点进行完整数据库备份
2. **增量备份** - 周一至周六凌晨 2 点进行增量数据备份
3. **Redis 备份** - 同时备份 Redis 数据
4. **配置备份** - 备份关键配置文件
5. **自动验证** - 每日凌晨 4 点验证备份文件完整性
6. **恢复演练** - 每周日凌晨 5 点进行恢复演练测试
7. **自动清理** - 自动清理过期备份文件

## 备份系统架构

```
/workspace/hjtpx/scripts/
├── backup.sh           # 备份脚本
├── restore.sh          # 恢复脚本
├── verify-backup.sh    # 备份验证脚本
├── restore-drill.sh    # 恢复演练脚本
└── backup-config/      # 定时任务配置
    ├── crontab.example
    ├── *.service
    └── *.timer
```

## 快速开始

### 1. 手动运行完整备份

```bash
cd /workspace/hjtpx
./scripts/backup.sh full
```

### 2. 列出可用的备份

```bash
./scripts/backup.sh list
```

### 3. 测试恢复到临时数据库

```bash
./scripts/restore.sh test
```

## 备份脚本使用

### 基本用法

```bash
./scripts/backup.sh [命令]
```

### 可用命令

| 命令 | 说明 |
|------|------|
| `full` | 完整备份（默认） |
| `incremental` \| `incr` | 增量备份 |
| `db-only` | 仅数据库备份 |
| `redis` | 仅 Redis 备份 |
| `config` | 仅配置文件备份 |
| `cleanup` | 清理过期备份 |
| `list` | 列出所有备份 |

### 示例

```bash
# 完整备份
./scripts/backup.sh full

# 增量备份
./scripts/backup.sh incremental

# 仅数据库备份
./scripts/backup.sh db-only

# 列出备份
./scripts/backup.sh list

# 清理过期备份
./scripts/backup.sh cleanup
```

### 环境变量

可以通过环境变量自定义备份行为:

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `BACKUP_DIR` | 备份存储目录 | `/var/backups/hjtpx` |
| `DB_HOST` | 数据库主机 | `localhost` |
| `DB_PORT` | 数据库端口 | `5432` |
| `DB_NAME` | 数据库名 | `hjtpx` |
| `DB_USER` | 数据库用户 | `postgres` |
| `DB_PASSWORD` | 数据库密码 | `postgres` |
| `FULL_RETENTION_DAYS` | 完整备份保留天数 | `30` |
| `INCR_RETENTION_DAYS` | 增量备份保留天数 | `7` |
| `AUTO_CLEANUP` | 是否自动清理 | `true` |

示例:

```bash
BACKUP_DIR=/data/backups ./scripts/backup.sh full
```

## 恢复脚本使用

### 基本用法

```bash
./scripts/restore.sh [命令] [备份文件]
```

### 可用命令

| 命令 | 说明 |
|------|------|
| `list` | 列出可用的备份 |
| `latest` | 恢复最新的完整备份 |
| `full <文件>` | 恢复指定的完整备份 |
| `incremental <文件>` | 恢复到指定增量备份 |
| `test` | 测试恢复到临时数据库 |
| `clean-temp` | 清理临时数据库 |
| `redis [文件]` | 恢复 Redis 备份 |

### 示例

```bash
# 列出可用备份
./scripts/restore.sh list

# 恢复最新备份
./scripts/restore.sh latest

# 恢复指定备份
./scripts/restore.sh full /var/backups/hjtpx/full/db_20250101_020000.sql.gz

# 测试恢复（不影响生产环境）
./scripts/restore.sh test
```

## 备份验证

### 基本用法

```bash
./scripts/verify-backup.sh [命令] [备份文件]
```

### 可用命令

| 命令 | 说明 |
|------|------|
| `latest` | 验证最新的备份（默认） |
| `all` | 验证所有备份 |
| `specific <文件>` | 验证指定的备份文件 |

### 示例

```bash
# 验证最新备份
./scripts/verify-backup.sh latest

# 验证所有备份
./scripts/verify-backup.sh all

# 验证指定备份
./scripts/verify-backup.sh specific /var/backups/hjtpx/full/db_20250101_020000.sql.gz
```

### 验证报告

验证完成后会生成 JSON 格式的报告，保存在 `/workspace/hjtpx/logs/backup_verification_report_*.json`。

## 恢复演练

### 基本用法

```bash
./scripts/restore-drill.sh [命令]
```

### 可用命令

| 命令 | 说明 |
|------|------|
| `full` | 运行完整恢复演练（默认） |

### 示例

```bash
# 运行恢复演练
./scripts/restore-drill.sh full
```

恢复演练会:
1. 查找最新的完整备份
2. 创建临时数据库
3. 恢复备份到临时数据库
4. 验证数据完整性
5. 清理临时数据库
6. 生成演练报告

### 演练报告

演练完成后会生成 JSON 格式的报告，保存在 `/workspace/hjtpx/logs/restore_drill_report_*.json`。

## 定时任务配置

### Cron 配置

详细使用说明请参考 [backup-config/README.md](../scripts/backup-config/README.md)。

快速设置:

```bash
# 复制示例配置
sudo cp /workspace/hjtpx/scripts/backup-config/crontab.example /etc/cron.d/hjtpx-backup
```

### Systemd 配置

详细使用说明请参考 [backup-config/README.md](../scripts/backup-config/README.md)。

快速设置:

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

## 故障排除

### 备份失败

1. 检查日志文件:
   ```bash
   tail -f /workspace/hjtpx/logs/backup_full.log
   ```

2. 验证数据库连接:
   ```bash
   export PGPASSWORD=your_password
   psql -h localhost -U postgres -d hjtpx
   ```

3. 检查备份目录权限:
   ```bash
   ls -la /var/backups/hjtpx/
   ```

### 恢复失败

1. 验证备份文件完整性:
   ```bash
   ./scripts/verify-backup.sh specific /path/to/backup.sql.gz
   ```

2. 检查是否有足够的磁盘空间:
   ```bash
   df -h
   ```

### Systemd 定时器不运行

1. 检查定时器状态:
   ```bash
   sudo systemctl list-timers 'hjtpx-*'
   ```

2. 查看服务日志:
   ```bash
   sudo journalctl -u hjtpx-backup-full.service -n 50
   ```

3. 手动测试服务:
   ```bash
   sudo systemctl start hjtpx-backup-full.service
   sudo systemctl status hjtpx-backup-full.service
   ```

## 最佳实践

1. **定期测试恢复** - 不要只备份，要定期测试恢复过程
2. **异地备份** - 将备份复制到远程服务器或云存储
3. **监控备份** - 配置告警，当备份失败时及时通知
4. **文档更新** - 保持本文档与实际配置同步
5. **权限控制** - 限制备份文件的访问权限

## 联系支持

如遇到问题，请检查:
1. 日志文件: `/workspace/hjtpx/logs/`
2. 配置文件: `/workspace/hjtpx/.env.production` 或 `/workspace/hjtpx/.env`
3. Systemd 日志: `journalctl -u hjtpx-*`

