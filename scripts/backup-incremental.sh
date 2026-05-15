#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BACKUP_DIR="${BACKUP_DIR:-/var/backups/hjtpx}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOG_DIR="${PROJECT_ROOT}/logs"
LOG_FILE="${LOG_DIR}/backup_incremental_${TIMESTAMP}.log"
RETAIN_CHAINS="${RETAIN_CHAINS:-3}"
WAL_DIR="${WAL_DIR:-/var/lib/postgresql/data/pg_wal}"

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
}

check_postgres_connection() {
    log "Checking PostgreSQL connection..."
    
    export PGPASSWORD="${DB_PASSWORD}"
    if ! pg_isready -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" > /dev/null 2>&1; then
        error_exit "PostgreSQL connection failed"
    fi
    
    local REPLICATION_ENABLED=$(psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d postgres -t -c \
        "SELECT COUNT(*) FROM pg_roles WHERE rolname = '${DB_USER}' AND rolreplication = true;" 2>/dev/null | tr -d '[:space:]')
    
    if [ "${REPLICATION_ENABLED}" -eq 0 ]; then
        log "Warning: User ${DB_USER} may not have replication privileges"
        log "Consider running: ALTER USER ${DB_USER} WITH REPLICATION;"
    fi
    
    log "PostgreSQL connection successful"
}

setup_backup_dir() {
    mkdir -p "${BACKUP_DIR}/base"
    mkdir -p "${BACKUP_DIR}/wal"
    mkdir -p "${BACKUP_DIR}/archive"
    log "Backup directory structure created at ${BACKUP_DIR}"
}

get_latest_base_backup() {
    ls -1td "${BACKUP_DIR}/base"/base_backup_* 2>/dev/null | head -n 1
}

get_base_backup_timestamp() {
    local BASE_BACKUP="$1"
    basename "$BASE_BACKUP" | sed 's/base_backup_//'
}

is_base_backup_valid() {
    local BACKUP_PATH="$1"
    
    if [ ! -d "$BACKUP_PATH" ]; then
        return 1
    fi
    
    if [ ! -f "${BACKUP_PATH}/backup_label" ]; then
        return 1
    fi
    
    if [ ! -f "${BACKUP_PATH}/backup_metadata" ]; then
        return 1
    fi
    
    return 0
}

create_base_backup() {
    log "Starting base backup using pg_basebackup..."
    
    local BASE_BACKUP_DIR="${BACKUP_DIR}/base/base_backup_${TIMESTAMP}"
    mkdir -p "$BASE_BACKUP_DIR"
    
    export PGPASSWORD="${DB_PASSWORD}"
    
    pg_basebackup -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -D "$BASE_BACKUP_DIR" \
        -Ft -z -P -X stream
    
    if [ -f "${BASE_BACKUP_DIR}/backup_label" ]; then
        log "Base backup created successfully: $BASE_BACKUP_DIR"
        
        cat > "${BASE_BACKUP_DIR}/backup_metadata" << EOF
{
    "type": "base",
    "timestamp": "${TIMESTAMP}",
    "created_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "wal_archive": "${BACKUP_DIR}/archive/archive_${TIMESTAMP}",
    "compression": "gzip",
    "format": "tar"
}
EOF
        
        mkdir -p "${BACKUP_DIR}/archive/archive_${TIMESTAMP}"
        
        local BASE_SIZE=$(du -sh "$BASE_BACKUP_DIR" | cut -f1)
        log "Base backup size: ${BASE_SIZE}"
        
        ln -sf "$BASE_BACKUP_DIR" "${BACKUP_DIR}/base/latest" 2>/dev/null || true
        
        return 0
    else
        rm -rf "$BASE_BACKUP_DIR"
        error_exit "Base backup failed - backup_label not found"
    fi
}

archive_wal_files() {
    local START_WAL="$1"
    local END_WAL="$2"
    
    log "Archiving WAL files from ${START_WAL} to ${END_WAL}..."
    
    export PGPASSWORD="${DB_PASSWORD}"
    
    local WAL_ARCHIVE_DIR="${BACKUP_DIR}/archive/archive_${TIMESTAMP}"
    mkdir -p "$WAL_ARCHIVE_DIR"
    
    local CURRENT_WAL="$START_WAL"
    while [ "$CURRENT_WAL" != "$END_WAL" ]; do
        local WAL_FILE=$(printf "%s%024s" "${WAL_DIR}/" "$CURRENT_WAL")
        
        if [ -f "${WAL_FILE}" ]; then
            gzip -c "$WAL_FILE" > "${WAL_ARCHIVE_DIR}/$(basename $WAL_FILE).gz"
        fi
        
        CURRENT_WAL=$(psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d postgres -t -c \
            "SELECT pg_walfile_name_offset('${CURRENT_WAL}'::pg_lsn);" 2>/dev/null | \
            psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d postgres -t -c \
            "SELECT '${CURRENT_WAL}'::pg_lsn + '8MB'::pg_lsn;" 2>/dev/null | tr -d '[:space:]' || \
            echo "$CURRENT_WAL")
    done
    
    local ARCHIVE_COUNT=$(ls -1 "${WAL_ARCHIVE_DIR}" 2>/dev/null | wc -l)
    log "Archived ${ARCHIVE_COUNT} WAL files to ${WAL_ARCHIVE_DIR}"
}

get_current_wal() {
    export PGPASSWORD="${DB_PASSWORD}"
    psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d postgres -t -c \
        "SELECT pg_current_wal_lsn();" 2>/dev/null | tr -d '[:space:]' || echo "0/0"
}

create_incremental_backup() {
    log "Starting incremental backup (continuous archiving)..."
    
    local LATEST_BASE=$(get_latest_base_backup)
    
    if [ -z "$LATEST_BASE" ] || ! is_base_backup_valid "$LATEST_BASE"; then
        log "No valid base backup found, creating base backup first..."
        create_base_backup
        return 0
    fi
    
    local BASE_TIMESTAMP=$(get_base_backup_timestamp "$LATEST_BASE")
    log "Latest valid base backup: ${LATEST_BASE} (timestamp: ${BASE_TIMESTAMP})"
    
    local START_LSN=$(grep "START LSN:" "${LATEST_BASE}/backup_label" 2>/dev/null | awk '{print $3}' || echo "0/0")
    local END_LSN=$(get_current_wal)
    
    local INCR_BACKUP_DIR="${BACKUP_DIR}/incremental/incr_backup_${TIMESTAMP}"
    mkdir -p "$INCR_BACKUP_DIR"
    
    cat > "${INCR_BACKUP_DIR}/incr_metadata" << EOF
{
    "type": "incremental",
    "timestamp": "${TIMESTAMP}",
    "created_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "base_backup": "${LATEST_BASE}",
    "base_timestamp": "${BASE_TIMESTAMP}",
    "start_lsn": "${START_LSN}",
    "end_lsn": "${END_LSN}"
}
EOF
    
    ln -sf "${LATEST_BASE}" "${INCR_BACKUP_DIR}/base_link" 2>/dev/null || true
    
    local INCR_SIZE=$(du -sh "$INCR_BACKUP_DIR" | cut -f1)
    log "Incremental backup metadata created: ${INCR_BACKUP_DIR}"
    log "Start LSN: ${START_LSN}, End LSN: ${END_LSN}"
}

cleanup_old_base_backups() {
    log "Cleaning up old base backups (keeping ${RETAIN_CHAINS} chains)..."
    
    local BACKUP_COUNT=$(ls -1td "${BACKUP_DIR}/base"/base_backup_* 2>/dev/null | wc -l)
    
    if [ "$BACKUP_COUNT" -gt "$RETAIN_CHAINS" ]; then
        local TO_DELETE=$(ls -1td "${BACKUP_DIR}/base"/base_backup_* 2>/dev/null | tail -n +$((RETAIN_CHAINS + 1)))
        
        for BACKUP in $TO_DELETE; do
            log "Removing old base backup: $BACKUP"
            rm -rf "$BACKUP"
            
            local ARCHIVE_DIR=$(grep "wal_archive:" "$BACKUP/backup_metadata" 2>/dev/null | cut -d'"' -f4)
            if [ -n "$ARCHIVE_DIR" ] && [ -d "$ARCHIVE_DIR" ]; then
                rm -rf "$ARCHIVE_DIR"
                log "Removed associated WAL archive: $ARCHIVE_DIR"
            fi
        done
    fi
    
    log "Base backup cleanup completed"
}

cleanup_old_incremental_backups() {
    local INCR_RETENTION_DAYS="${INCR_RETENTION_DAYS:-7}"
    
    log "Cleaning up incremental backups older than ${INCR_RETENTION_DAYS} days..."
    
    find "${BACKUP_DIR}/incremental" -type d -name "incr_backup_*" -mtime +${INCR_RETENTION_DAYS} -exec rm -rf {} + 2>/dev/null || true
    
    log "Incremental backup cleanup completed"
}

verify_backup_chain() {
    log "Verifying backup chain integrity..."
    
    local BASE_BACKUP=$(get_latest_base_backup)
    
    if [ -z "$BASE_BACKUP" ]; then
        log "No base backup found for chain verification"
        return 1
    fi
    
    if ! is_base_backup_valid "$BASE_BACKUP"; then
        log "Base backup is invalid or corrupted"
        return 1
    fi
    
    log "Backup chain verification passed for: $BASE_BACKUP"
    return 0
}

list_backup_chains() {
    log "=== Backup Chain Information ==="
    
    echo -e "\n--- Base Backups ---"
    if [ -d "${BACKUP_DIR}/base" ]; then
        for BACKUP in "${BACKUP_DIR}/base"/base_backup_*; do
            if [ -d "$BACKUP" ]; then
                local TIMESTAMP=$(basename "$BACKUP" | sed 's/base_backup_//')
                local SIZE=$(du -sh "$BACKUP" 2>/dev/null | cut -f1)
                local VALID=$(is_base_backup_valid "$BACKUP" && echo "VALID" || echo "INVALID")
                echo "  ${TIMESTAMP} - ${SIZE} - ${VALID}"
            fi
        done
        
        if [ -L "${BACKUP_DIR}/base/latest" ]; then
            echo "  Latest: $(readlink -f "${BACKUP_DIR}/base/latest")"
        fi
    else
        echo "  No base backups found"
    fi
    
    echo -e "\n--- Incremental Backups ---"
    local INCR_COUNT=$(ls -1td "${BACKUP_DIR}/incremental"/incr_backup_* 2>/dev/null | head -5 | wc -l)
    if [ "$INCR_COUNT" -gt 0 ]; then
        ls -1td "${BACKUP_DIR}/incremental"/incr_backup_* 2>/dev/null | head -5 | while read BACKUP; do
            local TIMESTAMP=$(basename "$BACKUP" | sed 's/incr_backup_//')
            echo "  ${TIMESTAMP}"
        done
    else
        echo "  No incremental backups found"
    fi
    
    echo -e "\n--- WAL Archives ---"
    local ARCHIVE_COUNT=$(ls -1td "${BACKUP_DIR}/archive"/archive_* 2>/dev/null | head -3 | wc -l)
    if [ "$ARCHIVE_COUNT" -gt 0 ]; then
        ls -1td "${BACKUP_DIR}/archive"/archive_* 2>/dev/null | head -3 | while read ARCHIVE; do
            local TIMESTAMP=$(basename "$ARCHIVE" | sed 's/archive_//')
            local SIZE=$(du -sh "$ARCHIVE" 2>/dev/null | cut -f1)
            local FILE_COUNT=$(ls -1 "$ARCHIVE" 2>/dev/null | wc -l)
            echo "  ${TIMESTAMP} - ${SIZE} - ${FILE_COUNT} files"
        done
    else
        echo "  No WAL archives found"
    fi
}

calculate_total_backup_size() {
    local TOTAL_SIZE=$(du -sh "${BACKUP_DIR}" 2>/dev/null | cut -f1)
    local BASE_SIZE=$(du -sh "${BACKUP_DIR}/base" 2>/dev/null | cut -f1)
    local INCR_SIZE=$(du -sh "${BACKUP_DIR}/incremental" 2>/dev/null | cut -f1)
    local ARCHIVE_SIZE=$(du -sh "${BACKUP_DIR}/archive" 2>/dev/null | cut -f1)
    
    log "=== Backup Size Summary ==="
    log "Total: ${TOTAL_SIZE}"
    log "  Base backups: ${BASE_SIZE}"
    log "  Incremental backups: ${INCR_SIZE}"
    log "  WAL archives: ${ARCHIVE_SIZE}"
}

main() {
    local ACTION="${1:-incremental}"
    
    log "========================================="
    log "HJTPX Incremental Backup System"
    log "Action: ${ACTION}"
    log "Timestamp: ${TIMESTAMP}"
    log "========================================="
    
    setup_backup_dir
    load_env
    
    case "${ACTION}" in
        base|incremental|incr|full|verify)
            check_postgres_connection
            ;;
    esac
    
    case "${ACTION}" in
        base)
            create_base_backup
            cleanup_old_base_backups
            ;;
        incremental|incr)
            create_incremental_backup
            cleanup_old_incremental_backups
            ;;
        full)
            create_base_backup
            cleanup_old_base_backups
            cleanup_old_incremental_backups
            ;;
        verify)
            verify_backup_chain
            ;;
        list)
            list_backup_chains
            ;;
        size)
            calculate_total_backup_size
            ;;
        cleanup)
            cleanup_old_base_backups
            cleanup_old_incremental_backups
            ;;
        *)
            echo "Usage: $0 {base|incremental|full|verify|list|size|cleanup}"
            echo "  base         - Create base backup (full PostgreSQL data)"
            echo "  incremental  - Create incremental backup (WAL archive)"
            echo "  full         - Full cycle: base + cleanup"
            echo "  verify       - Verify backup chain integrity"
            echo "  list         - List all backup chains"
            echo "  size         - Show backup size statistics"
            echo "  cleanup      - Clean up old backups"
            exit 1
            ;;
    esac
    
    log "========================================="
    log "Backup operation completed successfully!"
    log "========================================="
}

main "$@"
