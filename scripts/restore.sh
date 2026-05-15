#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BACKUP_DIR="${BACKUP_DIR:-/var/backups/hjtpx}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOG_DIR="${PROJECT_ROOT}/logs"
LOG_FILE="${LOG_DIR}/restore_${TIMESTAMP}.log"

mkdir -p "$LOG_DIR"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

error_exit() {
    log "ERROR: $1"
    exit 1
}

load_env() {
    if [ -f "${PROJECT_ROOT}/.env.production" ]; then
        export $(grep -v '^#' "${PROJECT_ROOT}/.env.production" | xargs)
    elif [ -f "${PROJECT_ROOT}/.env" ]; then
        export $(grep -v '^#' "${PROJECT_ROOT}/.env" | xargs)
    fi
    
    DB_HOST="${DB_HOST:-localhost}"
    DB_PORT="${DB_PORT:-5432}"
    DB_NAME="${DB_NAME:-hjtpx}"
    DB_USER="${DB_USER:-postgres}"
    DB_PASSWORD="${DB_PASSWORD:-postgres}"
    DB_NAME_TEMP="${DB_NAME}_temp_${TIMESTAMP}"
}

confirm_action() {
    local MESSAGE="$1"
    read -p "$MESSAGE (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log "Action cancelled by user"
        exit 1
    fi
}

list_available_backups() {
    echo "Available full backups:"
    ls -1t "${BACKUP_DIR}/full"/db_*.sql.gz 2>/dev/null || echo "No full backups found"
    
    echo -e "\nAvailable incremental backups:"
    ls -1t "${BACKUP_DIR}/incremental"/db_incr_*.sql.gz 2>/dev/null || echo "No incremental backups found"
}

get_latest_full_backup() {
    ls -1t "${BACKUP_DIR}/full"/db_*.sql.gz 2>/dev/null | head -n 1
}

get_incrementals_for_full() {
    local FULL_BACKUP="$1"
    find "${BACKUP_DIR}/incremental" -name "db_incr_*.sql.gz" -exec bash -c '
        for f; do
            [ -f "$f.base" ] && grep -qF "'"$FULL_BACKUP"'" "$f.base" && echo "$f"
        done
    ' bash {} + | sort
}

check_postgres_connection() {
    log "Checking PostgreSQL connection..."
    
    export PGPASSWORD="${DB_PASSWORD}"
    if ! pg_isready -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" > /dev/null 2>&1; then
        error_exit "PostgreSQL connection failed"
    fi
    log "PostgreSQL connection successful"
}

create_temp_database() {
    log "Creating temporary database: ${DB_NAME_TEMP}"
    
    export PGPASSWORD="${DB_PASSWORD}"
    psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d postgres -c \
        "CREATE DATABASE \"${DB_NAME_TEMP}\" TEMPLATE template0;" > /dev/null 2>&1
}

drop_temp_database() {
    log "Dropping temporary database: ${DB_NAME_TEMP}"
    
    export PGPASSWORD="${DB_PASSWORD}"
    psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d postgres -c \
        "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '${DB_NAME_TEMP}';" > /dev/null 2>&1 || true
    psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d postgres -c \
        "DROP DATABASE IF EXISTS \"${DB_NAME_TEMP}\";" > /dev/null 2>&1
}

restore_full_backup() {
    local BACKUP_FILE="$1"
    local TARGET_DB="$2"
    
    log "Restoring full backup: ${BACKUP_FILE} to ${TARGET_DB}"
    
    if [ ! -f "${BACKUP_FILE}" ]; then
        error_exit "Backup file not found: ${BACKUP_FILE}"
    fi
    
    export PGPASSWORD="${DB_PASSWORD}"
    pg_restore -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${TARGET_DB}" \
        --no-owner --no-acl --clean --if-exists "${BACKUP_FILE}"
    
    log "Full backup restored successfully"
}

restore_incremental_backup() {
    local BACKUP_FILE="$1"
    local TARGET_DB="$2"
    
    log "Restoring incremental backup: ${BACKUP_FILE} to ${TARGET_DB}"
    
    if [ ! -f "${BACKUP_FILE}" ]; then
        error_exit "Incremental backup file not found: ${BACKUP_FILE}"
    fi
    
    export PGPASSWORD="${DB_PASSWORD}"
    pg_restore -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${TARGET_DB}" \
        --no-owner --no-acl --data-only --disable-triggers "${BACKUP_FILE}"
    
    log "Incremental backup restored successfully"
}

restore_to_live() {
    local SOURCE_DB="$1"
    local TARGET_DB="$2"
    
    log "Restoring to live database: ${TARGET_DB}"
    
    export PGPASSWORD="${DB_PASSWORD}"
    
    psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d postgres -c \
        "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '${TARGET_DB}';" > /dev/null 2>&1 || true
    
    psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d postgres -c \
        "DROP DATABASE IF EXISTS \"${TARGET_DB}\";" > /dev/null 2>&1
    
    psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d postgres -c \
        "ALTER DATABASE \"${SOURCE_DB}\" RENAME TO \"${TARGET_DB}\";" > /dev/null 2>&1
    
    log "Live database restored successfully"
}

restore_redis() {
    local REDIS_BACKUP="$1"
    
    log "Restoring Redis backup: ${REDIS_BACKUP}"
    
    if [ ! -f "${REDIS_BACKUP}" ]; then
        log "Redis backup file not found, skipping Redis restore"
        return 1
    fi
    
    local REDIS_DATA_DIR=$(redis-cli -h "${REDIS_HOST:-localhost}" -p "${REDIS_PORT:-6379}" CONFIG GET dir | tail -n 1)
    
    redis-cli -h "${REDIS_HOST:-localhost}" -p "${REDIS_PORT:-6379}" SHUTDOWN NOSAVE > /dev/null 2>&1 || true
    
    sleep 2
    
    gunzip -c "${REDIS_BACKUP}" > "${REDIS_DATA_DIR}/dump.rdb"
    
    log "Redis backup restored. Please restart Redis server manually."
}

verify_restored_data() {
    local DB_NAME="$1"
    
    log "Verifying restored data in ${DB_NAME}"
    
    export PGPASSWORD="${DB_PASSWORD}"
    
    local TABLE_COUNT=$(psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -t -c \
        "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';" | tr -d '[:space:]')
    
    log "Restored database has ${TABLE_COUNT} tables"
    
    if [ "${TABLE_COUNT}" -gt 0 ]; then
        log "Data verification successful"
        return 0
    else
        error_exit "Data verification failed: No tables found"
    fi
}

main() {
    local ACTION="${1:-}"
    local BACKUP_FILE="${2:-}"
    
    log "========================================="
    log "HJTPX Restore System"
    log "Timestamp: ${TIMESTAMP}"
    log "========================================="
    
    load_env
    
    case "${ACTION}" in
        list)
            list_available_backups
            exit 0
            ;;
        latest)
            local LATEST_FULL=$(get_latest_full_backup)
            if [ -z "${LATEST_FULL}" ]; then
                error_exit "No full backup found"
            fi
            
            confirm_action "This will restore the latest full backup to the live database. Continue?"
            
            check_postgres_connection
            create_temp_database
            restore_full_backup "${LATEST_FULL}" "${DB_NAME_TEMP}"
            verify_restored_data "${DB_NAME_TEMP}"
            restore_to_live "${DB_NAME_TEMP}" "${DB_NAME}"
            log "Latest backup restored successfully!"
            ;;
        full)
            if [ -z "${BACKUP_FILE}" ]; then
                error_exit "Please specify the full backup file"
            fi
            
            confirm_action "This will restore the full backup to the live database. Continue?"
            
            check_postgres_connection
            create_temp_database
            restore_full_backup "${BACKUP_FILE}" "${DB_NAME_TEMP}"
            verify_restored_data "${DB_NAME_TEMP}"
            restore_to_live "${DB_NAME_TEMP}" "${DB_NAME}"
            log "Full backup restored successfully!"
            ;;
        incremental)
            if [ -z "${BACKUP_FILE}" ]; then
                error_exit "Please specify the incremental backup file"
            fi
            
            confirm_action "This will restore the incremental backup chain. Continue?"
            
            local BASE_FILE=$(cat "${BACKUP_FILE}.base" 2>/dev/null || true)
            if [ -z "${BASE_FILE}" ] || [ ! -f "${BASE_FILE}" ]; then
                error_exit "Base full backup not found for this incremental backup"
            fi
            
            check_postgres_connection
            create_temp_database
            restore_full_backup "${BASE_FILE}" "${DB_NAME_TEMP}"
            
            local INCREMENTALS=($(get_incrementals_for_full "${BASE_FILE}"))
            for INCR in "${INCREMENTALS[@]}"; do
                if [ "$INCR" = "$BACKUP_FILE" ]; then
                    restore_incremental_backup "${INCR}" "${DB_NAME_TEMP}"
                    break
                fi
                restore_incremental_backup "${INCR}" "${DB_NAME_TEMP}"
            done
            
            verify_restored_data "${DB_NAME_TEMP}"
            restore_to_live "${DB_NAME_TEMP}" "${DB_NAME}"
            log "Incremental backup restored successfully!"
            ;;
        test)
            local LATEST_FULL=$(get_latest_full_backup)
            if [ -z "${LATEST_FULL}" ]; then
                error_exit "No full backup found for testing"
            fi
            
            log "Performing test restore (will not affect live database)"
            
            check_postgres_connection
            create_temp_database
            restore_full_backup "${LATEST_FULL}" "${DB_NAME_TEMP}"
            verify_restored_data "${DB_NAME_TEMP}"
            
            log "Test restore successful! Temporary database: ${DB_NAME_TEMP}"
            log "You can connect to it to verify the data, then it will be kept for 1 hour."
            
            (sleep 3600 && drop_temp_database) &
            ;;
        clean-temp)
            drop_temp_database
            ;;
        redis)
            if [ -z "${BACKUP_FILE}" ]; then
                local LATEST_REDIS=$(ls -1t "${BACKUP_DIR}/redis"/redis_*.rdb.gz 2>/dev/null | head -n 1)
                if [ -z "${LATEST_REDIS}" ]; then
                    error_exit "No Redis backup found"
                fi
                BACKUP_FILE="${LATEST_REDIS}"
            fi
            restore_redis "${BACKUP_FILE}"
            ;;
        *)
            echo "Usage: $0 {list|latest|full|incremental|test|clean-temp|redis} [backup_file]"
            echo "  list          - List available backups"
            echo "  latest        - Restore latest full backup"
            echo "  full <file>   - Restore specific full backup"
            echo "  incremental <file> - Restore up to specific incremental backup"
            echo "  test          - Test restore to temporary database"
            echo "  clean-temp    - Clean up temporary databases"
            echo "  redis [file]  - Restore Redis backup"
            exit 1
            ;;
    esac
    
    log "========================================="
    log "Restore completed successfully!"
    log "========================================="
}

main "$@"
