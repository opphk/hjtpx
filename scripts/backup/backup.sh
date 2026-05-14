#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

source "$PROJECT_ROOT/.env" 2>/dev/null || true

BACKUP_DIR="${BACKUP_DIR:-/var/backups/hjtpx}"
RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-30}"
INCREMENTAL_BACKUP="${INCREMENTAL_BACKUP:-true}"

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_TYPE="full"
PREVIOUS_BACKUP=""

if [ "$INCREMENTAL_BACKUP" = "true" ]; then
    PREVIOUS_FULL=$(find "$BACKUP_DIR" -maxdepth 1 -name "*.full.tar.gz" -type f | sort -r | head -1)
    if [ -n "$PREVIOUS_FULL" ]; then
        BACKUP_TYPE="incremental"
        PREVIOUS_BACKUP=$(basename "$PREVIOUS_FULL" .full.tar.gz)
    fi
fi

BACKUP_NAME="backup_${TIMESTAMP}"
BACKUP_PATH="$BACKUP_DIR/${BACKUP_NAME}"
FINAL_ARCHIVE="$BACKUP_DIR/${BACKUP_NAME}.${BACKUP_TYPE}.tar.gz"

mkdir -p "$BACKUP_PATH"
mkdir -p "$BACKUP_DIR"

log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1"
}

error() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1" >&2
}

cleanup() {
    if [ -d "$BACKUP_PATH" ]; then
        rm -rf "$BACKUP_PATH"
    fi
}

trap cleanup ERR

log "Starting ${BACKUP_TYPE} backup..."
log "Backup directory: $BACKUP_DIR"
log "Retention: $RETENTION_DAYS days"

if [ -n "$MONGODB_URI" ]; then
    log "Backing up MongoDB..."
    MONGODB_DUMP_DIR="$BACKUP_PATH/mongodb"
    mkdir -p "$MONGODB_DUMP_DIR"

    mongodump \
        --uri="$MONGODB_URI" \
        --out="$MONGODB_DUMP_DIR" \
        --gzip \
        --oplog \
        2>&1 | while read line; do log "MongoDB: $line"; done

    log "MongoDB backup completed"
fi

if [ -n "$POSTGRES_HOST" ]; then
    log "Backing up PostgreSQL..."
    POSTGRES_DUMP_DIR="$BACKUP_PATH/postgresql"
    mkdir -p "$POSTGRES_DUMP_DIR"

    export PGPASSWORD="${POSTGRES_PASSWORD}"

    psql -h "$POSTGRES_HOST" \
         -p "${POSTGRES_PORT:-5432}" \
         -U "${POSTGRES_USER}" \
         -d postgres \
         -c "SELECT datname FROM pg_database WHERE datname NOT IN ('template0', 'template1', 'postgres')" \
         -t 2>/dev/null | while read dbname; do
        dbname=$(echo "$dbname" | xargs)
        if [ -n "$dbname" ]; then
            log "Dumping PostgreSQL database: $dbname"
            pg_dump -h "$POSTGRES_HOST" \
                    -p "${POSTGRES_PORT:-5432}" \
                    -U "${POSTGRES_USER}" \
                    -d "$dbname" \
                    -Fc \
                    -f "$POSTGRES_DUMP_DIR/${dbname}.dump"
        fi
    done

    unset PGPASSWORD
    log "PostgreSQL backup completed"
fi

if [ -n "$REDIS_HOST" ]; then
    log "Backing up Redis..."
    REDIS_DUMP_DIR="$BACKUP_PATH/redis"
    mkdir -p "$REDIS_DUMP_DIR"

    redis-cli -h "$REDIS_HOST" \
              -p "${REDIS_PORT:-6379}" \
              ${REDIS_PASSWORD:+-a "$REDIS_PASSWORD"} \
              --rdb "$REDIS_DUMP_DIR/dump.rdb" 2>&1 | while read line; do log "Redis: $line"; done

    log "Redis backup completed"
fi

if [ -d "$PROJECT_ROOT/config" ]; then
    log "Backing up configuration files..."
    cp -r "$PROJECT_ROOT/config" "$BACKUP_PATH/"
fi

if [ -d "$PROJECT_ROOT/.env" ]; then
    log "Backing up environment files..."
    cp "$PROJECT_ROOT/.env" "$BACKUP_PATH/.env"
fi

log "Creating archive..."
tar -czf "$FINAL_ARCHIVE" -C "$BACKUP_DIR" "$(basename "$BACKUP_PATH")"
rm -rf "$BACKUP_PATH"

BACKUP_SIZE=$(du -h "$FINAL_ARCHIVE" | cut -f1)
log "Backup created: $FINAL_ARCHIVE (Size: $BACKUP_SIZE)"

METADATA_FILE="$BACKUP_DIR/${BACKUP_NAME}.meta.json"
cat > "$METADATA_FILE" <<EOF
{
    "name": "${BACKUP_NAME}",
    "type": "${BACKUP_TYPE}",
    "timestamp": "$(date -Iseconds)",
    "size": "$BACKUP_SIZE",
    "archive": "$(basename "$FINAL_ARCHIVE")",
    "retention_days": $RETENTION_DAYS,
    "previous_backup": ${PREVIOUS_BACKUP:+"$PREVIOUS_BACKUP"},
    "components": {
        "mongodb": ${MONGODB_URI:+true},
        "postgresql": ${POSTGRES_HOST:+true},
        "redis": ${REDIS_HOST:+true},
        "config": true
    }
}
EOF

if [ "$BACKUP_TYPE" = "full" ]; then
    rm -f "$FINAL_ARCHIVE.${BACKUP_TYPE}.sha256"
fi

SHA256=$(sha256sum "$FINAL_ARCHIVE" | awk '{print $1}')
echo "$SHA256" > "$FINAL_ARCHIVE.sha256"

log "Cleaning up old backups (older than $RETENTION_DAYS days)..."
find "$BACKUP_DIR" -name "*.full.tar.gz" -mtime +$RETENTION_DAYS -type f -delete
find "$BACKUP_DIR" -name "*.incremental.tar.gz" -mtime +$RETENTION_DAYS -type f -delete
find "$BACKUP_DIR" -name "*.meta.json" -mtime +$RETENTION_DAYS -type f -delete
find "$BACKUP_DIR" -name "*.sha256" -mtime +$RETENTION_DAYS -type f -delete

BACKUP_COUNT=$(find "$BACKUP_DIR" -name "*.tar.gz" | wc -l)
log "Total backups: $BACKUP_COUNT"

log "Backup process completed successfully!"

if [ -n "$SLACK_WEBHOOK_URL" ]; then
    curl -s -X POST "$SLACK_WEBHOOK_URL" \
        -H 'Content-Type: application/json' \
        -d "{\"text\": \"✅ Database backup completed: ${BACKUP_NAME} (${BACKUP_TYPE}, ${BACKUP_SIZE})\"}" \
        > /dev/null 2>&1 || true
fi

exit 0
