# HJTPX 备份系统配置

本目录包含 HJTPX 数据库备份系统的配置文件，支持 cron 和 systemd 两种定时任务方式。

## 目录结构

- `crontab.example` - Cron 定时任务配置示例
- `*.service` - Systemd 服务文件
- `*.timer` - Systemd 定时器文件

## 使用 Cron

### 安装 Cron 任务

1. 将 `crontab.example` 复制到 `/etc/cron.d/hjtpx-backup`:

```bash
sudo cp /workspace/hjtpx/scripts/backup-config/crontab.example /etc/cron.d/hjtpx-backup
```

2. 或者将内容添加到当前用户的 crontab:

```bash
crontab -e
# 然后将 crontab.example 的内容粘贴进去
```

3. 验证 cron 任务:

```bash
crontab -l
# 或
sudo cat /etc/cron.d/hjtpx-backup
```

## 使用 Systemd

### 安装 Systemd 服务和定时器

1. 复制服务文件到 systemd 目录:

```bash
sudo cp /workspace/hjtpx/scripts/backup-config/*.service /etc/systemd/system/
sudo cp /workspace/hjtpx/scripts/backup-config/*.timer /etc/systemd/system/
```

2. 重新加载 systemd 配置:

```bash
sudo systemctl daemon-reload
```

3. 启用并启动定时器:

```bash
# 完整备份
sudo systemctl enable --now hjtpx-backup-full.timer

# 增量备份
sudo systemctl enable --now hjtpx-backup-incremental.timer

# 备份验证
sudo systemctl enable --now hjtpx-verify-backups.timer

# 恢复演练
sudo systemctl enable --now hjtpx-restore-drill.timer
```

4. 查看定时器状态:

```bash
sudo systemctl list-timers 'hjtpx-*'
```

5. 手动运行服务进行测试:

```bash
sudo systemctl start hjtpx-backup-full.service
sudo systemctl status hjtpx-backup-full.service
```

## 查看日志

所有备份相关的日志都会输出到 `/workspace/hjtpx/logs/` 目录:

```bash
# 查看完整备份日志
tail -f /workspace/hjtpx/logs/backup_full.log

# 查看增量备份日志
tail -f /workspace/hjtpx/logs/backup_incr.log

# 查看验证日志
tail -f /workspace/hjtpx/logs/backup_verify.log

# 查看恢复演练日志
tail -f /workspace/hjtpx/logs/restore_drill.log
```

使用 journalctl 查看 systemd 服务日志:

```bash
sudo journalctl -u hjtpx-backup-full.service -f
```

## 备份目录

默认情况下，备份会存储在 `/var/backups/hjtpx/` 目录中，结构如下:

```
/var/backups/hjtpx/
├── full/          # 完整备份
├── incremental/   # 增量备份
├── redis/         # Redis 备份
└── config/        # 配置文件备份
```

