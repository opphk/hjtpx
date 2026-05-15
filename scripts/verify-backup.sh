#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BACKUP_DIR="${BACKUP_DIR:-/var/backups/hjtpx}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOG_DIR="${PROJECT_ROOT}/logs"
LOG_FILE="${LOG_DIR}/verify_backup_${TIMESTAMP}.log"
REPORT_FILE="${LOG_DIR}/backup_verification_report_${TIMESTAMP}.json"

mkdir -p "$LOG_DIR"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

error_exit() {
    log "ERROR: $1"
    echo "{\"status\": \"failed\", \"error\": \"$1\"}" > "$REPORT_FILE"
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

check_file_exists() {
    local FILE="$1"
    if [ ! -f "$FILE" ]; then
        log "File not found: $FILE"
        return 1
    fi
    return 0
}

check_file_size() {
    local FILE="$1"
    local MIN_SIZE="${2:-1024}"
    
    local SIZE=$(stat -f%z "$FILE" 2>/dev/null || stat -c%s "$FILE" 2>/dev/null)
    if [ "$SIZE" -lt "$MIN_SIZE" ]; then
        log "File too small: $FILE ($SIZE bytes, min: $MIN_SIZE)"
        return 1
    fi
    log "File size OK: $FILE ($SIZE bytes)"
    return 0
}

verify_postgres_backup() {
    local BACKUP_FILE="$1"
    
    log "Verifying PostgreSQL backup: $BACKUP_FILE"
    
    if ! check_file_exists "$BACKUP_FILE"; then
        return 1
    fi
    
    if ! check_file_size "$BACKUP_FILE" 1024; then
        return 1
    fi
    
    export PGPASSWORD="${DB_PASSWORD}"
    if ! pg_restore --list "$BACKUP_FILE" > /dev/null 2>&1; then
        log "Backup file is corrupted or not a valid PostgreSQL dump: $BACKUP_FILE"
        return 1
    fi
    
    local OBJECT_COUNT=$(pg_restore --list "$BACKUP_FILE" 2>/dev/null | wc -l)
    log "PostgreSQL backup contains $OBJECT_COUNT objects"
    
    if [ "$OBJECT_COUNT" -lt 5 ]; then
        log "Warning: Backup contains very few objects ($OBJECT_COUNT)"
    fi
    
    log "PostgreSQL backup verified successfully: $BACKUP_FILE"
    return 0
}

verify_redis_backup() {
    local BACKUP_FILE="$1"
    
    log "Verifying Redis backup: $BACKUP_FILE"
    
    if ! check_file_exists "$BACKUP_FILE"; then
        return 1
    fi
    
    if ! check_file_size "$BACKUP_FILE" 10; then
        return 1
    fi
    
    if ! gunzip -t "$BACKUP_FILE" > /dev/null 2>&1; then
        log "Redis backup file is corrupted: $BACKUP_FILE"
        return 1
    fi
    
    log "Redis backup verified successfully: $BACKUP_FILE"
    return 0
}

verify_config_backup() {
    local BACKUP_FILE="$1"
    
    log "Verifying config backup: $BACKUP_FILE"
    
    if ! check_file_exists "$BACKUP_FILE"; then
        return 1
    fi
    
    if ! check_file_size "$BACKUP_FILE" 100; then
        return 1
    fi
    
    if ! tar -tzf "$BACKUP_FILE" > /dev/null 2>&1; then
        log "Config backup file is corrupted: $BACKUP_FILE"
        return 1
    fi
    
    local FILE_COUNT=$(tar -tzf "$BACKUP_FILE" 2>/dev/null | wc -l)
    log "Config backup contains $FILE_COUNT files"
    
    log "Config backup verified successfully: $BACKUP_FILE"
    return 0
}

verify_latest_backups() {
    log "Verifying latest backups"
    
    local RESULTS=()
    
    local LATEST_FULL=$(ls -1t "${BACKUP_DIR}/full"/db_*.sql.gz 2>/dev/null | head -n 1)
    if [ -n "$LATEST_FULL" ]; then
        if verify_postgres_backup "$LATEST_FULL"; then
            RESULTS+=("{\"type\": \"full\", \"file\": \"$LATEST_FULL\", \"status\": \"ok\"}")
        else
            RESULTS+=("{\"type\": \"full\", \"file\": \"$LATEST_FULL\", \"status\": \"failed\"}")
        fi
    else
        log "No full backups found"
        RESULTS+=("{\"type\": \"full\", \"status\": \"missing\"}")
    fi
    
    local LATEST_INCR=$(ls -1t "${BACKUP_DIR}/incremental"/db_incr_*.sql.gz 2>/dev/null | head -n 1)
    if [ -n "$LATEST_INCR" ]; then
        if verify_postgres_backup "$LATEST_INCR"; then
            RESULTS+=("{\"type\": \"incremental\", \"file\": \"$LATEST_INCR\", \"status\": \"ok\"}")
        else
            RESULTS+=("{\"type\": \"incremental\", \"file\": \"$LATEST_INCR\", \"status\": \"failed\"}")
        fi
    fi
    
    local LATEST_REDIS=$(ls -1t "${BACKUP_DIR}/redis"/redis_*.rdb.gz 2>/dev/null | head -n 1)
    if [ -n "$LATEST_REDIS" ]; then
        if verify_redis_backup "$LATEST_REDIS"; then
            RESULTS+=("{\"type\": \"redis\", \"file\": \"$LATEST_REDIS\", \"status\": \"ok\"}")
        else
            RESULTS+=("{\"type\": \"redis\", \"file\": \"$LATEST_REDIS\", \"status\": \"failed\"}")
        fi
    fi
    
    local LATEST_CONFIG=$(ls -1t "${BACKUP_DIR}/config"/config_*.tar.gz 2>/dev/null | head -n 1)
    if [ -n "$LATEST_CONFIG" ]; then
        if verify_config_backup "$LATEST_CONFIG"; then
            RESULTS+=("{\"type\": \"config\", \"file\": \"$LATEST_CONFIG\", \"status\": \"ok\"}")
        else
            RESULTS+=("{\"type\": \"config\", \"file\": \"$LATEST_CONFIG\", \"status\": \"failed\"}")
        fi
    fi
    
    echo "{\"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\", \"results\": [${RESULTS[*]}]}" > "$REPORT_FILE"
    
    log "Verification report saved to: $REPORT_FILE"
}

verify_all_backups() {
    log "Verifying all backups"
    
    local RESULTS=()
    local TOTAL=0
    local PASSED=0
    local FAILED=0
    
    shopt -s nullglob
    for BACKUP in "${BACKUP_DIR}/full"/db_*.sql.gz; do
        if [ -f "$BACKUP" ]; then
            ((TOTAL++))
            if verify_postgres_backup "$BACKUP"; then
                ((PASSED++))
                RESULTS+=("{\"type\": \"full\", \"file\": \"$BACKUP\", \"status\": \"ok\"}")
            else
                ((FAILED++))
                RESULTS+=("{\"type\": \"full\", \"file\": \"$BACKUP\", \"status\": \"failed\"}")
            fi
        fi
    done
    shopt -u nullglob
    
    shopt -s nullglob
    for BACKUP in "${BACKUP_DIR}/incremental"/db_incr_*.sql.gz; do
        if [ -f "$BACKUP" ]; then
            ((TOTAL++))
            if verify_postgres_backup "$BACKUP"; then
                ((PASSED++))
                RESULTS+=("{\"type\": \"incremental\", \"file\": \"$BACKUP\", \"status\": \"ok\"}")
            else
                ((FAILED++))
                RESULTS+=("{\"type\": \"incremental\", \"file\": \"$BACKUP\", \"status\": \"failed\"}")
            fi
        fi
    done
    shopt -u nullglob
    
    echo "{\"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\", \"total\": $TOTAL, \"passed\": $PASSED, \"failed\": $FAILED, \"results\": [${RESULTS[*]}]}" > "$REPORT_FILE"
    
    log "Verification complete: Total=$TOTAL, Passed=$PASSED, Failed=$FAILED"
    log "Report saved to: $REPORT_FILE"
}

verify_specific_backup() {
    local BACKUP_FILE="$1"
    
    if [[ "$BACKUP_FILE" == *"/full/"* ]]; then
        verify_postgres_backup "$BACKUP_FILE"
    elif [[ "$BACKUP_FILE" == *"/incremental/"* ]]; then
        verify_postgres_backup "$BACKUP_FILE"
    elif [[ "$BACKUP_FILE" == *"/redis/"* ]]; then
        verify_redis_backup "$BACKUP_FILE"
    elif [[ "$BACKUP_FILE" == *"/config/"* ]]; then
        verify_config_backup "$BACKUP_FILE"
    else
        error_exit "Unknown backup type: $BACKUP_FILE"
    fi
}

main() {
    local ACTION="${1:-latest}"
    local BACKUP_FILE="${2:-}"
    
    log "========================================="
    log "HJTPX Backup Verification System"
    log "Timestamp: ${TIMESTAMP}"
    log "========================================="
    
    load_env
    
    case "${ACTION}" in
        latest)
            verify_latest_backups
            ;;
        all)
            verify_all_backups
            ;;
        specific)
            if [ -z "$BACKUP_FILE" ]; then
                error_exit "Please specify the backup file"
            fi
            verify_specific_backup "$BACKUP_FILE"
            ;;
        *)
            echo "Usage: $0 {latest|all|specific} [backup_file]"
            echo "  latest      - Verify latest backups (default)"
            echo "  all         - Verify all backups"
            echo "  specific <file> - Verify a specific backup file"
            exit 1
            ;;
    esac
    
    log "========================================="
    log "Verification completed!"
    log "========================================="
}

main "$@"
