#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

source "$PROJECT_ROOT/.env" 2>/dev/null || true

BACKUP_DIR="${BACKUP_DIR:-/var/backups/hjtpx}"
RESTORE_DIR="${RESTORE_DIR:-/tmp/hjtpx_restore_$(date +%Y%m%d_%H%M%S)}"
DRY_RUN="${DRRY_RUN:-false}"

log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1"
}

error() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1" >&2
}

restore_mongodb() {
    local backup_path="$1"
    local mongodb_dir="$backup_path/mongodb"

    if [ ! -d "$mongodb_dir" ]; then
        log "No MongoDB backup found in archive"
        return 0
    fi

    log "Restoring MongoDB..."

    if [ "$DRY_RUN" = "true" ]; then
        log "[DRY RUN] Would restore MongoDB from $mongodb_dir"
        return 0
    fi

    mongorestore \
        --uri="$MONGODB_URI" \
        --drop \
        --dir="$mongodb_dir" \
        --gzip \
        --oplogReplay \
        2>&1 | while read line; do log "MongoDB: $line"; done

    log "MongoDB restore completed"
}

restore_postgresql() {
    local backup_path="$1"
    local postgresql_dir="$backup_path/postgresql"

    if [ ! -d "$postgresql_dir" ]; then
        log "No PostgreSQL backup found in archive"
        return 0
    fi

    if [ "$DRY_RUN" = "true" ]; then
        log "[DRY RUN] Would restore PostgreSQL databases"
        return 0
    fi

    log "Restoring PostgreSQL..."

    export PGPASSWORD="${POSTGRES_PASSWORD}"

    for dump_file in "$postgresql_dir"/*.dump; do
        if [ -f "$dump_file" ]; then
            dbname=$(basename "$dump_file" .dump)
            log "Restoring database: $dbname"

            psql -h "$POSTGRES_HOST" \
                 -p "${POSTGRES_PORT:-5432}" \
                 -U "${POSTGRES_USER}" \
                 -d "$dbname" \
                 -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '$dbname' AND pid <> pg_backend_pid();" \
                 2>/dev/null || true

            pg_restore -h "$POSTGRES_HOST" \
                       -p "${POSTGRES_PORT:-5432}" \
                       -U "${POSTGRES_USER}" \
                       -d "$dbname" \
                       --clean \
                       --if-exists \
                       "$dump_file" 2>&1 | while read line; do log "PostgreSQL: $line"; done
        fi
    done

    unset PGPASSWORD
    log "PostgreSQL restore completed"
}

restore_redis() {
    local backup_path="$1"
    local redis_rdb="$backup_path/redis/dump.rdb"

    if [ ! -f "$redis_rdb" ]; then
        log "No Redis backup found in archive"
        return 0
    fi

    log "Restoring Redis..."

    if [ "$DRY_RUN" = "true" ]; then
        log "[DRY RUN] Would restore Redis from $redis_rdb"
        return 0
    fi

    redis-cli -h "$REDIS_HOST" \
              -p "${REDIS_PORT:-6379}" \
              ${REDIS_PASSWORD:+-a "$REDIS_PASSWORD"} \
              SHUTDOWN NOSAVE 2>/dev/null || true

    sleep 2

    cp "$redis_rdb" "${REDIS_DATA_DIR:-/var/lib/redis}/dump.rdb"

    redis-server --daemonize yes

    sleep 2

    if redis-cli -h "$REDIS_HOST" \
                 -p "${REDIS_PORT:-6379}" \
                 ${REDIS_PASSWORD:+-a "$REDIS_PASSWORD"} \
                 PING > /dev/null 2>&1; then
        log "Redis restore completed"
    else
        error "Redis restore failed"
        return 1
    fi
}

log "==========================================="
log "Database Restore Script"
log "==========================================="
log "Restore directory: $RESTORE_DIR"
log "Dry run: $DRY_RUN"

if [ -z "$1" ]; then
    echo "Usage: $0 <backup_archive.tar.gz> [--dry-run]"
    echo ""
    echo "Available backups:"
    find "$BACKUP_DIR" -maxdepth 1 -name "*.tar.gz" -type f | sort -r | head -10
    exit 1
fi

BACKUP_ARCHIVE="$1"
shift

if [ "$1" = "--dry-run" ]; then
    DRY_RUN="true"
fi

if [ ! -f "$BACKUP_ARCHIVE" ]; then
    if [ -f "$BACKUP_DIR/$BACKUP_ARCHIVE" ]; then
        BACKUP_ARCHIVE="$BACKUP_DIR/$BACKUP_ARCHIVE"
    else
        error "Backup file not found: $BACKUP_ARCHIVE"
        exit 1
    fi
fi

log "Using backup: $BACKUP_ARCHIVE"

if [ "$DRY_RUN" != "true" ]; then
    read -p "This will overwrite current database data. Continue? (yes/no): " confirm
    if [ "$confirm" != "yes" ]; then
        log "Restore cancelled"
        exit 0
    fi
fi

log "Extracting backup..."
mkdir -p "$RESTORE_DIR"
tar -xzf "$BACKUP_ARCHIVE" -C "$RESTORE_DIR"

EXTRACTED_DIR=$(find "$RESTORE_DIR" -mindepth 1 -maxdepth 1 -type d | head -1)
if [ -z "$EXTRACTED_DIR" ]; then
    error "Failed to extract backup"
    exit 1
fi

log "Starting restore process..."

restore_mongodb "$EXTRACTED_DIR"
restore_postgresql "$EXTRACTED_DIR"
restore_redis "$EXTRACTED_DIR"

if [ "$DRY_RUN" != "true" ]; then
    log "Cleaning up temporary files..."
    rm -rf "$RESTORE_DIR"
fi

log "==========================================="
log "Restore completed successfully!"
log "==========================================="

if [ "$DRY_RUN" = "true" ]; then
    log "This was a dry run. No actual changes were made."
fi

exit 0
