# Database Backup and Restore Guide

## Overview

This document describes the automated backup and restore procedures for the HJTPX project database system.

## Features

- **Full Backup**: Complete database backup with compression
- **Incremental Backup**: Delta backups based on changes since last full backup
- **Backup Verification**: Integrity checking with checksums
- **Disaster Recovery Drills**: Automated testing of backup restore procedures
- **Scheduled Automation**: Cron-based scheduling for regular backups

## Quick Start

### Manual Backup

```bash
# Full backup
npm run backup:full

# Incremental backup
npm run backup:incremental

# Verify latest backup
npm run backup:verify

# Restore from backup
npm run backup:restore

# Run disaster recovery drill
npm run backup:drill
```

### Scheduled Backup

```bash
# Start the backup scheduler
npm run backup:schedule
```

## Configuration

Configure backup settings in `.env`:

```bash
# Backup directory
BACKUP_DIR=./backups

# Retention period (days)
BACKUP_RETENTION_DAYS=30

# Cron schedules
FULL_BACKUP_CRON=0 2 * * 0        # Every Sunday at 2 AM
INCREMENTAL_BACKUP_CRON=0 2 * * 1-6  # Monday-Saturday at 2 AM
VERIFY_BACKUP_CRON=0 3 * * 0       # Every Sunday at 3 AM
CLEANUP_CRON=0 4 * * 0            # Every Sunday at 4 AM
TIMEZONE=UTC
```

## Backup Types

### Full Backup

Complete database backup including all tables, indexes, and data.

```bash
node scripts/backup.js --type=full
```

### Incremental Backup

Captures only changes since the last backup.

```bash
node scripts/backup.js --type=incremental
```

### Verification

Validates backup integrity using SHA-256 checksums.

```bash
node scripts/backup.js --verify [backup_path]
```

### Restore

Restores database from a backup file.

```bash
node scripts/backup.js --restore [backup_path]
```

## Disaster Recovery Drill

The DRILL feature tests backup and restore procedures in an isolated environment:

```bash
node scripts/backup-drill.js
```

The drill performs:
1. Creates isolated drill database
2. Restores latest backup
3. Verifies data integrity
4. Tests query performance
5. Cleans up drill environment
6. Generates detailed report

### Drill Reports

Drill reports are saved to `backups/drills/drill_[timestamp].json` with:
- Step-by-step results
- Success/failure counts
- Performance metrics
- Recommendations

## Supported Databases

- PostgreSQL (primary)
- MongoDB

## Backup Storage

Backups are stored in the following structure:

```
backups/
├── full/                    # Full backups
├── incremental/             # Incremental backups
├── verify/                  # Verification reports
└── drills/                  # DRILL reports
    └── drill_[id].json
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `BACKUP_DIR` | Backup storage directory | `./backups` |
| `BACKUP_RETENTION_DAYS` | Days to keep backups | `30` |
| `DB_TYPE` | Database type (postgresql/mongodb) | `postgresql` |
| `DB_HOST` | Database host | `localhost` |
| `DB_PORT` | Database port | `5432` |
| `DB_NAME` | Database name | `hjtpx` |
| `DB_USER` | Database user | `postgres` |

## Retention Policy

- Backups older than `BACKUP_RETENTION_DAYS` are automatically cleaned up
- Minimum recommended retention: 7 days
- For critical data: 30+ days recommended

## Monitoring

Monitor backup jobs through:

1. **Application logs**: Check console output for backup status
2. **Verification reports**: Located in `backups/verify/`
3. **DRILL reports**: Located in `backups/drills/`

## Troubleshooting

### Backup Fails

1. Check database connection settings
2. Verify sufficient disk space
3. Ensure proper permissions on backup directory

### Restore Fails

1. Confirm backup file exists and is not corrupted
2. Verify checksum matches
3. Check target database is accessible

### DRILL Fails

1. Ensure drill database user has sufficient privileges
2. Check for disk space in drill environment
3. Review drill report for specific failure details

## Security Considerations

- Store backups in secure, offsite location
- Use encryption for sensitive data backups
- Restrict access to backup directory
- Regularly test backup restore procedures

## Best Practices

1. **Regular Testing**: Run DRILL at least monthly
2. **Multiple Backup Types**: Use both full and incremental backups
3. **Offsite Storage**: Copy backups to remote location
4. **Monitoring**: Set up alerts for backup failures
5. **Documentation**: Keep this guide updated with any changes
