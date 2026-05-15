#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BACKUP_DIR="${BACKUP_DIR:-/var/backups/hjtpx}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOG_DIR="${PROJECT_ROOT}/logs"
LOG_FILE="${LOG_DIR}/backup_${TIMESTAMP}.log"
METADATA_FILE="${BACKUP_DIR}/backup_metadata.json"

mkdir -p "$LOG_DIR"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

error_exit() {
    log "ERROR: $1"
    exit 1
}

setup_backup_dir() {
    mkdir -p "${BACKUP_DIR}/full"
    mkdir -p "${BACKUP_DIR}/incremental"
    mkdir -p "${BACKUP_DIR}/redis"
    mkdir -p "${BACKUP_DIR}/config"
    
    log "Backup directory structure created at ${BACKUP_DIR}"
}

load_env() {
    if [ -f "${PROJECT_ROOT}/.env.production" ]; then
        export $(grep -v '^#' "${PROJECT_ROOT}/.env.production" | xargs)
        log "Loaded environment from .env.production"
    elif [ -f "${PROJECT_ROOT}/.env" ]; then
        export $(grep -v '^#' "${PROJECT_ROOT}/.env" | xargs)
        log "Loaded environment from .env"
    fi
    
    DB_HOST="${DB_HOST:-localhost}"
    DB_PORT="${DB_PORT:-5432}"
    DB_NAME="${DB_NAME:-hjtpx}"
    DB_USER="${DB_USER:-postgres}"
    DB_PASSWORD="${DB_PASSWORD:-postgres}"
    REDIS_HOST="${REDIS_HOST:-localhost}"
    REDIS_PORT="${REDIS_PORT:-6379}"
}

check_postgres_connection() {
    log "Checking PostgreSQL connection..."
    
    export PGPASSWORD="${DB_PASSWORD}"
    if ! pg_isready -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" > /dev/null 2>&1; then
        error_exit "PostgreSQL connection failed"
    fi
    log "PostgreSQL connection successful"
}

check_redis_connection() {
    log "Checking Redis connection..."
    
    if ! redis-cli -h "${REDIS_HOST}" -p "${REDIS_PORT}" ping > /dev/null 2>&1; then
        log "Redis connection failed, skipping Redis backup"
        return 1
    fi
    log "Redis connection successful"
    return 0
}

get_latest_full_backup() {
    ls -1t "${BACKUP_DIR}/full"/db_*.sql.gz 2>/dev/null | head -n 1
}

create_full_backup() {
    log "Starting full database backup..."
    
    export PGPASSWORD="${DB_PASSWORD}"
    
    BACKUP_FILE="${BACKUP_DIR}/full/db_${TIMESTAMP}.sql.gz"
    pg_dump -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" \
        --format=c --blobs --no-owner --no-acl -Z 9 -f "${BACKUP_FILE}"
    
    if [ -f "${BACKUP_FILE}" ] && [ -s "${BACKUP_FILE}" ]; then
        BACKUP_SIZE=$(du -h "${BACKUP_FILE}" | cut -f1)
        log "Full database backup created: ${BACKUP_FILE} (${BACKUP_SIZE})"
        echo "${BACKUP_FILE}" > "${BACKUP_DIR}/full/latest.txt"
        update_backup_metadata "full" "${BACKUP_FILE}" "${BACKUP_SIZE}"
        return 0
    else
        error_exit "Full database backup failed"
    fi
}

create_incremental_backup() {
    log "Starting incremental backup..."
    
    LATEST_FULL=$(get_latest_full_backup)
    if [ -z "${LATEST_FULL}" ]; then
        log "No full backup found, creating full backup instead"
        create_full_backup
        return 0
    fi
    
    export PGPASSWORD="${DB_PASSWORD}"
    
    BACKUP_FILE="${BACKUP_DIR}/incremental/db_incr_${TIMESTAMP}.sql.gz"
    
    pg_dump -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" \
        --format=c --blobs --no-owner --no-acl \
        --data-only --disable-triggers --inserts -Z 9 -f "${BACKUP_FILE}"
    
    if [ -f "${BACKUP_FILE}" ] && [ -s "${BACKUP_FILE}" ]; then
        BACKUP_SIZE=$(du -h "${BACKUP_FILE}" | cut -f1)
        log "Incremental backup created: ${BACKUP_FILE} (${BACKUP_SIZE})"
        echo "${LATEST_FULL}" > "${BACKUP_FILE}.base"
        update_backup_metadata "incremental" "${BACKUP_FILE}" "${BACKUP_SIZE}"
        return 0
    else
        error_exit "Incremental backup failed"
    fi
}

backup_redis() {
    if ! check_redis_connection; then
        return 1
    fi
    
    log "Starting Redis backup..."
    
    REDIS_BACKUP_FILE="${BACKUP_DIR}/redis/redis_${TIMESTAMP}.rdb"
    
    redis-cli -h "${REDIS_HOST}" -p "${REDIS_PORT}" BGSAVE > /dev/null 2>&1
    
    sleep 5
    
    RDB_PATH=$(redis-cli -h "${REDIS_HOST}" -p "${REDIS_PORT}" CONFIG GET dir | tail -n 1)
    if [ -f "${RDB_PATH}/dump.rdb" ]; then
        cp "${RDB_PATH}/dump.rdb" "${REDIS_BACKUP_FILE}"
        gzip -f "${REDIS_BACKUP_FILE}"
        BACKUP_SIZE=$(du -h "${REDIS_BACKUP_FILE}.gz" | cut -f1)
        log "Redis backup created: ${REDIS_BACKUP_FILE}.gz (${BACKUP_SIZE})"
        update_backup_metadata "redis" "${REDIS_BACKUP_FILE}.gz" "${BACKUP_SIZE}"
    else
        log "Redis RDB file not found"
    fi
}

backup_config() {
    log "Starting configuration backup..."
    
    CONFIG_BACKUP_FILE="${BACKUP_DIR}/config/config_${TIMESTAMP}.tar.gz"
    
    CONFIG_FILES=()
    [ -f "${PROJECT_ROOT}/.env" ] && CONFIG_FILES+=(".env")
    [ -f "${PROJECT_ROOT}/.env.production" ] && CONFIG_FILES+=(".env.production")
    [ -f "${PROJECT_ROOT}/.env.staging" ] && CONFIG_FILES+=(".env.staging")
    [ -d "${PROJECT_ROOT}/config" ] && CONFIG_FILES+=("config/")
    [ -d "${PROJECT_ROOT}/migrations" ] && CONFIG_FILES+=("migrations/")
    
    if [ ${#CONFIG_FILES[@]} -gt 0 ]; then
        tar -czf "${CONFIG_BACKUP_FILE}" -C "${PROJECT_ROOT}" "${CONFIG_FILES[@]}"
        CONFIG_SIZE=$(du -h "${CONFIG_BACKUP_FILE}" | cut -f1)
        log "Configuration backup created: ${CONFIG_BACKUP_FILE} (${CONFIG_SIZE})"
        update_backup_metadata "config" "${CONFIG_BACKUP_FILE}" "${CONFIG_SIZE}"
    else
        log "No configuration files found to backup"
    fi
}

update_backup_metadata() {
    local BACKUP_TYPE="$1"
    local BACKUP_PATH="$2"
    local BACKUP_SIZE="$3"
    
    local METADATA_TEMP="${LOG_DIR}/.backup_metadata_temp.json"
    
    if [ -f "${METADATA_FILE}" ]; then
        local EXISTING=$(cat "${METADATA_FILE}")
    else
        local EXISTING="{\"backups\": []}"
    fi
    
    local ENTRY="{\"type\": \"${BACKUP_TYPE}\", \"path\": \"${BACKUP_PATH}\", \"size\": \"${BACKUP_SIZE}\", \"timestamp\": \"${TIMESTAMP}\", \"date\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}"
    
    if command -v jq &> /dev/null; then
        echo "${EXISTING}" | jq --argjson entry "${ENTRY}" '.backups += [$entry]' > "${METADATA_TEMP}" 2>/dev/null || echo "${EXISTING}" > "${METADATA_TEMP}"
        mv "${METADATA_TEMP}" "${METADATA_FILE}"
    else
        log "jq not found, skipping metadata update"
    fi
}

cleanup_old_backups() {
    log "Cleaning up old backups..."
    
    FULL_RETENTION_DAYS="${FULL_RETENTION_DAYS:-30}"
    INCR_RETENTION_DAYS="${INCR_RETENTION_DAYS:-7}"
    REDIS_RETENTION_DAYS="${REDIS_RETENTION_DAYS:-7}"
    CONFIG_RETENTION_DAYS="${CONFIG_RETENTION_DAYS:-30}"
    
    find "${BACKUP_DIR}/full" -name "db_*.sql.gz" -mtime +${FULL_RETENTION_DAYS} -delete
    find "${BACKUP_DIR}/incremental" -name "db_incr_*.sql.gz" -mtime +${INCR_RETENTION_DAYS} -delete
    find "${BACKUP_DIR}/incremental" -name "db_incr_*.sql.gz.base" -mtime +${INCR_RETENTION_DAYS} -delete
    find "${BACKUP_DIR}/redis" -name "redis_*.rdb.gz" -mtime +${REDIS_RETENTION_DAYS} -delete
    find "${BACKUP_DIR}/config" -name "config_*.tar.gz" -mtime +${CONFIG_RETENTION_DAYS} -delete
    
    log "Old backups cleaned (full: ${FULL_RETENTION_DAYS}d, incr: ${INCR_RETENTION_DAYS}d, redis: ${REDIS_RETENTION_DAYS}d, config: ${CONFIG_RETENTION_DAYS}d)"
}

verify_backup_integrity() {
    log "Verifying backup integrity..."
    
    local BACKUP_FILE="$1"
    
    if [ -f "${BACKUP_FILE}" ]; then
        if pg_restore --list "${BACKUP_FILE}" > /dev/null 2>&1; then
            log "Backup verified successfully: ${BACKUP_FILE}"
            return 0
        else
            error_exit "Backup verification failed: ${BACKUP_FILE}"
        fi
    fi
}

list_backups() {
    log "Current backups:"
    
    echo -e "\n=== Full Backups ==="
    ls -lh "${BACKUP_DIR}/full" 2>/dev/null || echo "No full backups"
    
    echo -e "\n=== Incremental Backups ==="
    ls -lh "${BACKUP_DIR}/incremental" 2>/dev/null || echo "No incremental backups"
    
    echo -e "\n=== Redis Backups ==="
    ls -lh "${BACKUP_DIR}/redis" 2>/dev/null || echo "No Redis backups"
    
    echo -e "\n=== Config Backups ==="
    ls -lh "${BACKUP_DIR}/config" 2>/dev/null || echo "No config backups"
}

main() {
    local BACKUP_TYPE="${1:-full}"
    
    log "========================================="
    log "HJTPX Backup System"
    log "Type: ${BACKUP_TYPE}"
    log "Timestamp: ${TIMESTAMP}"
    log "========================================="
    
    setup_backup_dir
    load_env
    
    case "${BACKUP_TYPE}" in
        full|incremental|incr|db-only|redis|config)
            check_postgres_connection
            ;;
    esac
    
    case "${BACKUP_TYPE}" in
        full)
            create_full_backup
            verify_backup_integrity "$(cat "${BACKUP_DIR}/full/latest.txt")"
            backup_redis
            backup_config
            ;;
        incremental|incr)
            create_incremental_backup
            backup_redis
            ;;
        db-only)
            create_full_backup
            verify_backup_integrity "$(cat "${BACKUP_DIR}/full/latest.txt")"
            ;;
        redis)
            backup_redis
            ;;
        config)
            backup_config
            ;;
        cleanup)
            cleanup_old_backups
            ;;
        list)
            list_backups
            ;;
        *)
            echo "Usage: $0 {full|incremental|db-only|redis|config|cleanup|list}"
            echo "  full          - Full backup (default)"
            echo "  incremental   - Incremental backup"
            echo "  db-only       - Database only"
            echo "  redis         - Redis only"
            echo "  config        - Configuration only"
            echo "  cleanup       - Remove old backups"
            echo "  list          - List existing backups"
            exit 1
            ;;
    esac
    
    if [ "${AUTO_CLEANUP:-true}" = "true" ] && [ "${BACKUP_TYPE}" != "cleanup" ] && [ "${BACKUP_TYPE}" != "list" ]; then
        cleanup_old_backups
    fi
    
    log "========================================="
    log "Backup completed successfully!"
    log "========================================="
}

main "$@"
