#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BACKUP_DIR="${BACKUP_DIR:-/var/backups/captchax}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOG_FILE="${PROJECT_ROOT}/logs/backup_${TIMESTAMP}.log"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

error_exit() {
    log "ERROR: $1"
    exit 1
}

setup_backup_dir() {
    mkdir -p "${BACKUP_DIR}"
    mkdir -p "${PROJECT_ROOT}/logs"

    log "Backup directory: ${BACKUP_DIR}"
}

backup_database() {
    log "Starting database backup..."

    export PGPASSWORD="${DB_PASSWORD:-captcha_pass_2026}"
    export PGHOST="${DB_HOST:-localhost}"
    export PGPORT="${DB_PORT:-5432}"
    export PGUSER="${DB_USER:-captcha_admin}"
    export PGDATABASE="${DB_NAME:-captcha_db}"

    DB_BACKUP_FILE="${BACKUP_DIR}/captcha_db_${TIMESTAMP}.sql.gz"

    pg_dump -h "${PGHOST}" -p "${PGPORT}" -U "${PGUSER}" -d "${PGDATABASE}" \
        --no-owner --no-acl | gzip > "${DB_BACKUP_FILE}"

    if [ -f "${DB_BACKUP_FILE}" ] && [ -s "${DB_BACKUP_FILE}" ]; then
        DB_BACKUP_SIZE=$(du -h "${DB_BACKUP_FILE}" | cut -f1)
        log "Database backup created: ${DB_BACKUP_FILE} (${DB_BACKUP_SIZE})"
    else
        error_exit "Database backup failed"
    fi
}

backup_config() {
    log "Starting configuration backup..."

    CONFIG_BACKUP_FILE="${BACKUP_DIR}/config_${TIMESTAMP}.tar.gz"

    if [ -d "${PROJECT_ROOT}/config" ]; then
        tar -czf "${CONFIG_BACKUP_FILE}" -C "${PROJECT_ROOT}" config/
        log "Configuration backup created: ${CONFIG_BACKUP_FILE}"
    else
        log "No configuration directory found"
    fi
}

backup_logs() {
    log "Starting logs backup..."

    LOG_BACKUP_FILE="${BACKUP_DIR}/logs_${TIMESTAMP}.tar.gz"

    if [ -d "${PROJECT_ROOT}/logs" ] && [ "$(ls -A "${PROJECT_ROOT}/logs" 2>/dev/null)" ]; then
        tar -czf "${LOG_BACKUP_FILE}" -C "${PROJECT_ROOT}" logs/
        log "Logs backup created: ${LOG_BACKUP_FILE}"
    else
        log "No logs to backup or logs directory empty"
    fi
}

cleanup_old_backups() {
    log "Cleaning up old backups..."

    RETENTION_DAYS="${RETENTION_DAYS:-7}"

    find "${BACKUP_DIR}" -name "*.sql.gz" -mtime +${RETENTION_DAYS} -delete
    find "${BACKUP_DIR}" -name "*.tar.gz" -mtime +${RETENTION_DAYS} -delete

    log "Old backups cleaned (retention: ${RETENTION_DAYS} days)"
}

verify_backup() {
    log "Verifying backup integrity..."

    if [ -f "${DB_BACKUP_FILE}" ]; then
        if zcat "${DB_BACKUP_FILE}" > /dev/null 2>&1; then
            log "Database backup verified successfully"
        else
            error_exit "Database backup verification failed"
        fi
    fi
}

list_backups() {
    log "Current backups in ${BACKUP_DIR}:"

    if [ -d "${BACKUP_DIR}" ]; then
        ls -lh "${BACKUP_DIR}" | tail -n +2 || log "No backups found"
    fi
}

main() {
    log "========================================="
    log "CaptchaX Backup Script"
    log "Timestamp: ${TIMESTAMP}"
    log "========================================="

    setup_backup_dir

    backup_database
    backup_config
    backup_logs

    if [ "${VERIFY_BACKUP:-true}" = "true" ]; then
        verify_backup
    fi

    if [ "${AUTO_CLEANUP:-true}" = "true" ]; then
        cleanup_old_backups
    fi

    list_backups

    log "========================================="
    log "Backup completed successfully!"
    log "========================================="
}

case "${1:-full}" in
    full)
        main
        ;;
    db)
        setup_backup_dir
        backup_database
        verify_backup
        ;;
    config)
        setup_backup_dir
        backup_config
        ;;
    logs)
        setup_backup_dir
        backup_logs
        ;;
    cleanup)
        cleanup_old_backups
        list_backups
        ;;
    list)
        list_backups
        ;;
    *)
        echo "Usage: $0 {full|db|config|logs|cleanup|list}"
        echo "  full     - Full backup (default)"
        echo "  db       - Database only"
        echo "  config   - Configuration only"
        echo "  logs     - Logs only"
        echo "  cleanup  - Remove old backups"
        echo "  list     - List existing backups"
        exit 1
        ;;
esac
