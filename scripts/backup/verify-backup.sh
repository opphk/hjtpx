#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

source "$PROJECT_ROOT/.env" 2>/dev/null || true

BACKUP_DIR="${BACKUP_DIR:-/var/backups/hjtpx}"
TEST_RESTORE_DIR="${TEST_RESTORE_DIR:-/tmp/hjtpx_restore_test}"
ALERT_EMAIL="${ALERT_EMAIL:-admin@example.com}"

log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1"
}

error() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1" >&2
}

if [ -z "$1" ]; then
    echo "Usage: $0 <backup_archive.tar.gz>"
    echo ""
    echo "Available backups:"
    find "$BACKUP_DIR" -maxdepth 1 -name "*.tar.gz" -type f | sort -r | head -10
    exit 1
fi

BACKUP_ARCHIVE="$1"

if [ ! -f "$BACKUP_ARCHIVE" ]; then
    if [ -f "$BACKUP_DIR/$BACKUP_ARCHIVE" ]; then
        BACKUP_ARCHIVE="$BACKUP_DIR/$BACKUP_ARCHIVE"
    else
        error "Backup file not found: $BACKUP_ARCHIVE"
        exit 1
    fi
fi

log "Starting backup verification for: $BACKUP_ARCHIVE"

if [ ! -f "${BACKUP_ARCHIVE}.sha256" ]; then
    error "SHA256 checksum file not found"
    exit 1
fi

log "Verifying checksum..."
CALCULATED_SHA=$(sha256sum "$BACKUP_ARCHIVE" | awk '{print $1}')
EXPECTED_SHA=$(cat "${BACKUP_ARCHIVE}.sha256")

if [ "$CALCULATED_SHA" != "$EXPECTED_SHA" ]; then
    error "Checksum verification failed!"
    error "Expected: $EXPECTED_SHA"
    error "Calculated: $CALCULATED_SHA"
    exit 1
fi
log "Checksum verified successfully"

log "Extracting backup archive..."
mkdir -p "$TEST_RESTORE_DIR"
tar -xzf "$BACKUP_ARCHIVE" -C "$TEST_RESTORE_DIR"

EXTRACTED_DIR=$(find "$TEST_RESTORE_DIR" -mindepth 1 -maxdepth 1 -type d | head -1)
if [ -z "$EXTRACTED_DIR" ]; then
    error "Failed to extract backup"
    exit 1
fi

log "Checking backup metadata..."
METADATA_FILE=$(find "$EXTRACTED_DIR" -name "*.meta.json" -type f 2>/dev/null | head -1)
if [ -n "$METADATA_FILE" ]; then
    log "Backup metadata:"
    cat "$METADATA_FILE" | jq .
    BACKUP_TYPE=$(jq -r '.type' "$METADATA_FILE")
    BACKUP_COMPONENTS=$(jq -r '.components | keys[]' "$METADATA_FILE")
    log "Backup type: $BACKUP_TYPE"
    log "Components: $BACKUP_COMPONENTS"
else
    error "No metadata file found"
fi

VERIFICATION_STATUS="SUCCESS"
VERIFICATION_DETAILS=""

verify_mongodb() {
    local mongodb_dir="$1"
    if [ -d "$mongodb_dir" ]; then
        log "Verifying MongoDB backup..."
        local bson_files=$(find "$mongodb_dir" -name "*.bson" | wc -l)
        if [ "$bson_files" -gt 0 ]; then
            log "MongoDB: Found $bson_files BSON files"
            return 0
        else
            log "MongoDB: No BSON files found"
            return 1
        fi
    fi
    return 1
}

verify_postgresql() {
    local postgresql_dir="$1"
    if [ -d "$postgresql_dir" ]; then
        log "Verifying PostgreSQL backup..."
        local dump_files=$(find "$postgresql_dir" -name "*.dump" | wc -l)
        if [ "$dump_files" -gt 0 ]; then
            log "PostgreSQL: Found $dump_files dump files"
            for dump_file in $(find "$postgresql_dir" -name "*.dump"); do
                local file_size=$(stat -f%z "$dump_file" 2>/dev/null || stat -c%s "$dump_file")
                if [ "$file_size" -lt 100 ]; then
                    log "PostgreSQL: Warning - small dump file: $(basename "$dump_file")"
                fi
            done
            return 0
        else
            log "PostgreSQL: No dump files found"
            return 1
        fi
    fi
    return 1
}

verify_redis() {
    local redis_dir="$1"
    if [ -d "$redis_dir" ]; then
        log "Verifying Redis backup..."
        if [ -f "$redis_dir/dump.rdb" ]; then
            local rdb_size=$(stat -f%z "$redis_dir/dump.rdb" 2>/dev/null || stat -c%s "$redis_dir/dump.rdb")
            log "Redis: Found dump.rdb (${rdb_size} bytes)"
            return 0
        fi
    fi
    return 1
}

verify_config() {
    local config_dir="$EXTRACTED_DIR/config"
    if [ -d "$config_dir" ]; then
        log "Verifying configuration files..."
        local config_count=$(find "$config_dir" -type f | wc -l)
        log "Configuration: Found $config_count files"
        return 0
    fi
    return 1
}

log "Running component verification..."
MONGODB_VERIFIED=false
POSTGRES_VERIFIED=false
REDIS_VERIFIED=false

if verify_mongodb "$EXTRACTED_DIR/mongodb"; then
    MONGODB_VERIFIED=true
fi

if verify_postgresql "$EXTRACTED_DIR/postgresql"; then
    POSTGRES_VERIFIED=true
fi

if verify_redis "$EXTRACTED_DIR/redis"; then
    REDIS_VERIFIED=true
fi

if verify_config; then
    CONFIG_VERIFIED=true
fi

log "Cleaning up test restore directory..."
rm -rf "$TEST_RESTORE_DIR"

TOTAL_COMPONENTS=0
VERIFIED_COMPONENTS=0

if [ "$MONGODB_VERIFIED" = true ]; then
    ((VERIFIED_COMPONENTS++))
fi
((TOTAL_COMPONENTS++))

if [ "$POSTGRES_VERIFIED" = true ]; then
    ((VERIFIED_COMPONENTS++))
fi
((TOTAL_COMPONENTS++))

if [ "$REDIS_VERIFIED" = true ]; then
    ((VERIFIED_COMPONENTS++))
fi
((TOTAL_COMPONENTS++))

VERIFICATION_SCORE=$((VERIFIED_COMPONENTS * 100 / TOTAL_COMPONENTS))

log "==========================================="
log "Backup Verification Summary"
log "==========================================="
log "Archive: $(basename "$BACKUP_ARCHIVE")"
log "Checksum: Verified ✓"
log "MongoDB: $([ "$MONGODB_VERIFIED" = true ] && echo "Verified ✓" || echo "Not included ✗")"
log "PostgreSQL: $([ "$POSTGRES_VERIFIED" = true ] && echo "Verified ✓" || echo "Not included ✗")"
log "Redis: $([ "$REDIS_VERIFIED" = true ] && echo "Verified ✓" || echo "Not included ✗")"
log "Configuration: $([ "$CONFIG_VERIFIED" = true ] && echo "Verified ✓" || echo "Not included ✗")"
log "Verification Score: ${VERIFICATION_SCORE}%"
log "==========================================="

if [ "$VERIFICATION_SCORE" -lt 50 ]; then
    error "Verification score below threshold"
    exit 1
fi

log "Backup verification completed successfully!"

if [ -n "$SLACK_WEBHOOK_URL" ]; then
    curl -s -X POST "$SLACK_WEBHOOK_URL" \
        -H 'Content-Type: application/json' \
        -d "{\"text\": \"✅ Backup verification completed: $(basename "$BACKUP_ARCHIVE") - Score: ${VERIFICATION_SCORE}%\"}" \
        > /dev/null 2>&1 || true
fi

exit 0
