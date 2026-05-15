#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BACKUP_DIR="${BACKUP_DIR:-/var/backups/hjtpx}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOG_DIR="${PROJECT_ROOT}/logs"
LOG_FILE="${LOG_DIR}/restore_drill_${TIMESTAMP}.log"
REPORT_FILE="${LOG_DIR}/restore_drill_report_${TIMESTAMP}.json"

mkdir -p "$LOG_DIR"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

error_exit() {
    log "ERROR: $1"
    echo "{\"status\": \"failed\", \"error\": \"$1\", \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}" > "$REPORT_FILE"
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
    DB_NAME_DRILL="${DB_NAME}_drill_${TIMESTAMP}"
}

check_postgres_connection() {
    log "Checking PostgreSQL connection..."
    
    export PGPASSWORD="${DB_PASSWORD}"
    if ! pg_isready -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" > /dev/null 2>&1; then
        error_exit "PostgreSQL connection failed"
    fi
    log "PostgreSQL connection successful"
}

create_drill_database() {
    log "Creating drill database: ${DB_NAME_DRILL}"
    
    export PGPASSWORD="${DB_PASSWORD}"
    psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d postgres -c \
        "CREATE DATABASE \"${DB_NAME_DRILL}\" TEMPLATE template0;" > /dev/null 2>&1
}

drop_drill_database() {
    log "Dropping drill database: ${DB_NAME_DRILL}"
    
    export PGPASSWORD="${DB_PASSWORD}"
    psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d postgres -c \
        "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '${DB_NAME_DRILL}';" > /dev/null 2>&1 || true
    psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d postgres -c \
        "DROP DATABASE IF EXISTS \"${DB_NAME_DRILL}\";" > /dev/null 2>&1
}

restore_backup() {
    local BACKUP_FILE="$1"
    local TARGET_DB="$2"
    
    log "Restoring backup: ${BACKUP_FILE} to ${TARGET_DB}"
    
    if [ ! -f "${BACKUP_FILE}" ]; then
        error_exit "Backup file not found: ${BACKUP_FILE}"
    fi
    
    export PGPASSWORD="${DB_PASSWORD}"
    pg_restore -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${TARGET_DB}" \
        --no-owner --no-acl --clean --if-exists "${BACKUP_FILE}"
    
    log "Backup restored successfully"
}

verify_data_integrity() {
    local DB_NAME="$1"
    
    log "Verifying data integrity in ${DB_NAME}"
    
    export PGPASSWORD="${DB_PASSWORD}"
    
    local TABLE_COUNT=$(psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -t -c \
        "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';" | tr -d '[:space:]')
    
    log "Database has ${TABLE_COUNT} tables"
    
    if [ "${TABLE_COUNT}" -eq 0 ]; then
        error_exit "Data verification failed: No tables found"
    fi
    
    local TOTAL_ROWS=0
    local TABLES=$(psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -t -c \
        "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' ORDER BY table_name;" | tr -d '[:space:]')
    
    for TABLE in $TABLES; do
        local ROW_COUNT=$(psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -t -c \
            "SELECT COUNT(*) FROM \"${TABLE}\";" 2>/dev/null | tr -d '[:space:]' || echo 0)
        TOTAL_ROWS=$((TOTAL_ROWS + ROW_COUNT))
        log "  - ${TABLE}: ${ROW_COUNT} rows"
    done
    
    log "Total rows across all tables: ${TOTAL_ROWS}"
    
    if [ "${TOTAL_ROWS}" -eq 0 ]; then
        log "Warning: No data rows found in any table"
    fi
    
    return 0
}

check_critical_tables() {
    local DB_NAME="$1"
    local -a CRITICAL_TABLES=("$@")
    shift
    
    log "Checking critical tables in ${DB_NAME}"
    
    export PGPASSWORD="${DB_PASSWORD}"
    
    for TABLE in "${CRITICAL_TABLES[@]}"; do
        local EXISTS=$(psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -t -c \
            "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = '${TABLE}');" | tr -d '[:space:]')
        
        if [ "$EXISTS" = "t" ]; then
            log "  ✓ Critical table found: ${TABLE}"
        else
            log "  ✗ Critical table missing: ${TABLE}"
            return 1
        fi
    done
    
    return 0
}

get_latest_full_backup() {
    ls -1t "${BACKUP_DIR}/full"/db_*.sql.gz 2>/dev/null | head -n 1
}

run_full_drill() {
    log "Starting full restore drill"
    
    local -a RESULTS=()
    local START_TIME=$(date +%s)
    local DRILL_STATUS="success"
    
    local LATEST_FULL=$(get_latest_full_backup)
    if [ -z "$LATEST_FULL" ]; then
        error_exit "No full backup found for drill"
    fi
    
    RESULTS+=("{\"step\": \"check_backup\", \"status\": \"ok\", \"backup\": \"$LATEST_FULL\"}")
    
    check_postgres_connection
    RESULTS+=("{\"step\": \"check_connection\", \"status\": \"ok\"}")
    
    create_drill_database
    RESULTS+=("{\"step\": \"create_database\", \"status\": \"ok\"}")
    
    local RESTORE_START=$(date +%s)
    restore_backup "$LATEST_FULL" "$DB_NAME_DRILL"
    local RESTORE_TIME=$(( $(date +%s) - RESTORE_START ))
    RESULTS+=("{\"step\": \"restore_backup\", \"status\": \"ok\", \"duration_seconds\": $RESTORE_TIME}")
    
    local VERIFY_START=$(date +%s)
    verify_data_integrity "$DB_NAME_DRILL"
    local VERIFY_TIME=$(( $(date +%s) - VERIFY_START ))
    RESULTS+=("{\"step\": \"verify_integrity\", \"status\": \"ok\", \"duration_seconds\": $VERIFY_TIME}")
    
    drop_drill_database
    RESULTS+=("{\"step\": \"cleanup\", \"status\": \"ok\"}")
    
    local TOTAL_TIME=$(( $(date +%s) - START_TIME ))
    
    echo "{\"drill_id\": \"$TIMESTAMP\", \"status\": \"$DRILL_STATUS\", \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\", \"total_duration_seconds\": $TOTAL_TIME, \"backup_used\": \"$LATEST_FULL\", \"steps\": [${RESULTS[*]}]}" > "$REPORT_FILE"
    
    log "========================================="
    log "Full restore drill completed successfully!"
    log "Total duration: ${TOTAL_TIME} seconds"
    log "Report saved to: $REPORT_FILE"
    log "========================================="
}

main() {
    local ACTION="${1:-full}"
    
    log "========================================="
    log "HJTPX Restore Drill System"
    log "Timestamp: ${TIMESTAMP}"
    log "========================================="
    
    load_env
    
    trap 'drop_drill_database 2>/dev/null || true' EXIT
    
    case "${ACTION}" in
        full)
            run_full_drill
            ;;
        *)
            echo "Usage: $0 {full}"
            echo "  full  - Run full restore drill (default)"
            exit 1
            ;;
    esac
}

main "$@"
