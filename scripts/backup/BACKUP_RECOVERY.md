# Database Backup and Recovery Guide

## Overview

This document describes the automated backup and recovery system for the HJTPX project.

## Backup Strategy

### Components Backed Up

- **MongoDB**: Primary database
- **PostgreSQL**: Analytics and reporting database
- **Redis**: Cache and session storage
- **Configuration**: Application configuration files

### Backup Types

1. **Full Backup**: Complete database snapshot
2. **Incremental Backup**: Changes since last full backup

### Retention Policy

- Default retention: 30 days
- Configurable via `BACKUP_RETENTION_DAYS` environment variable

## Quick Start

### Manual Backup

```bash
cd /workspace/hjtpx
./scripts/backup/backup.sh
```

### Manual Restore

```bash
./scripts/backup/restore.sh /var/backups/hjtpx/backup_20240115_120000.full.tar.gz
```

### Verify Backup

```bash
./scripts/backup/verify-backup.sh /var/backups/hjtpx/backup_20240115_120000.full.tar.gz
```

## Automated Backup Setup

### Cron Job Configuration

Add to crontab (`crontab -e`):

```cron
# Daily full backup at 2 AM
0 2 * * * /workspace/hjtpx/scripts/backup/backup.sh >> /var/log/backup.log 2>&1

# Verify backup daily at 3 AM
0 3 * * * /workspace/hjtpx/scripts/backup/verify-backup.sh $(ls -t /var/backups/hjtpx/*.full.tar.gz | head -1) >> /var/log/backup_verify.log 2>&1

# Weekly restore test (Sunday at 4 AM)
0 4 * * 0 /workspace/hjtpx/scripts/backup/restore.sh --dry-run $(ls -t /var/backups/hjtpx/*.full.tar.gz | head -1) >> /var/log/restore_test.log 2>&1
```

### Systemd Timer (Alternative)

Create `/etc/systemd/system/hjtpx-backup.timer`:

```ini
[Unit]
Description=HJTPX Database Backup Timer

[Timer]
OnCalendar=daily
Persistent=true

[Install]
WantedBy=timers.target
```

Create `/etc/systemd/system/hjtpx-backup.service`:

```ini
[Unit]
Description=HJTPX Database Backup

[Service]
Type=oneshot
ExecStart=/workspace/hjtpx/scripts/backup/backup.sh
User=www-data
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable hjtpx-backup.timer
sudo systemctl start hjtpx-backup.timer
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `BACKUP_DIR` | Backup storage directory | `/var/backups/hjtpx` |
| `BACKUP_RETENTION_DAYS` | Days to keep backups | `30` |
| `INCREMENTAL_BACKUP` | Enable incremental backups | `true` |
| `MONGODB_URI` | MongoDB connection string | - |
| `POSTGRES_HOST` | PostgreSQL host | - |
| `POSTGRES_PORT` | PostgreSQL port | `5432` |
| `POSTGRES_USER` | PostgreSQL user | - |
| `POSTGRES_PASSWORD` | PostgreSQL password | - |
| `REDIS_HOST` | Redis host | - |
| `REDIS_PORT` | Redis port | `6379` |
| `REDIS_PASSWORD` | Redis password | - |
| `SLACK_WEBHOOK_URL` | Slack webhook for notifications | - |

## Backup Verification

The verification script checks:

1. **Checksum Validation**: SHA256 verification
2. **Archive Integrity**: File extraction test
3. **Component Validation**:
   - MongoDB: BSON file presence
   - PostgreSQL: Dump file presence
   - Redis: RDB file presence
   - Config: Configuration file count

## Disaster Recovery Procedure

### 1. Identify the Issue

- Check application logs for errors
- Verify database connectivity
- Review recent changes

### 2. Select Backup

```bash
# List available backups
ls -la /var/backups/hjtpx/

# Check backup metadata
cat /var/backups/hjtpx/backup_20240115_120000.meta.json
```

### 3. Verify Backup

```bash
./scripts/backup/verify-backup.sh /var/backups/hjtpx/backup_20240115_120000.full.tar.gz
```

### 4. Dry Run Restore

```bash
./scripts/backup/restore.sh /var/backups/hjtpx/backup_20240115_120000.full.tar.gz --dry-run
```

### 5. Perform Restore

```bash
# Stop the application
sudo systemctl stop hjtpx

# Restore database
./scripts/backup/restore.sh /var/backups/hjtpx/backup_20240115_120000.full.tar.gz

# Start the application
sudo systemctl start hjtpx
```

### 6. Verify Restoration

```bash
# Check application health
curl http://localhost:3000/api/v1/health

# Review application logs
tail -f /var/log/hjtpx/app.log
```

## Monitoring

### Backup Success Notifications

Configure Slack webhook for notifications:

```bash
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
```

### Backup Metrics

Monitor backup metrics:

- Backup duration
- Backup size
- Backup success/failure rate
- Verification score

## Troubleshooting

### Backup Fails

1. Check disk space: `df -h`
2. Verify database connectivity
3. Check permissions on backup directory
4. Review backup logs: `/var/log/backup.log`

### Restore Fails

1. Verify backup archive integrity
2. Check available disk space
3. Ensure database services are stopped
4. Review restore logs

### Incremental Backup Issues

If incremental backups are failing:

1. Ensure a full backup exists
2. Check that previous backup is accessible
3. Verify backup directory permissions

## Security Considerations

- Store backups on encrypted storage
- Limit access to backup directory
- Use secure transport for remote backups
- Regularly test backup restoration
- Rotate encryption keys periodically
