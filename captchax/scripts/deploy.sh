#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOG_FILE="${PROJECT_ROOT}/logs/deploy_${TIMESTAMP}.log"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

error_exit() {
    log "ERROR: $1"
    exit 1
}

check_env() {
    log "Checking environment..."

    if ! command -v go &> /dev/null; then
        error_exit "Go is not installed"
    fi

    if [ ! -f "${PROJECT_ROOT}/go.mod" ]; then
        error_exit "go.mod not found"
    fi

    log "Environment check passed"
}

build_binary() {
    log "Building binary..."

    cd "${PROJECT_ROOT}"

    CGO_ENABLED=0 GOOS=linux go build \
        -ldflags="-w -s -X captchax/internal/config.version=${VERSION:-dev}" \
        -o captchax-server cmd/server/main.go

    CGO_ENABLED=0 GOOS=linux go build \
        -ldflags="-w -s" \
        -o captchax-admin cmd/admin/main.go

    log "Binary build completed"
}

run_migrations() {
    log "Running database migrations..."

    export PGPASSWORD="${DB_PASSWORD}"
    export PGHOST="${DB_HOST:-localhost}"
    export PGPORT="${DB_PORT:-5432}"
    export PGUSER="${DB_USER}"
    export PGDATABASE="${DB_NAME}"

    if [ ! -f "${PROJECT_ROOT}/migrations/001_initial_schema.sql" ]; then
        log "No migrations found"
        return 0
    fi

    psql -v ON_ERROR_STOP=1 -f "${PROJECT_ROOT}/migrations/001_initial_schema.sql" \
        || error_exit "Migration failed"

    log "Migrations completed"
}

install_service() {
    log "Installing systemd service..."

    if [ "$(id -u)" -ne 0 ]; then
        log "Skipping service installation (not root)"
        return 0
    fi

    cp "${PROJECT_ROOT}/systemd/captchax.service" /etc/systemd/system/
    systemctl daemon-reload
    systemctl enable captchax.service

    log "Service installed"
}

restart_service() {
    log "Restarting service..."

    if systemctl is-active --quiet captchax.service; then
        systemctl restart captchax.service
    else
        systemctl start captchax.service
    fi

    sleep 2

    if systemctl is-active --quiet captchax.service; then
        log "Service started successfully"
    else
        error_exit "Service failed to start"
    fi
}

health_check() {
    log "Running health check..."

    MAX_RETRIES=10
    RETRY_COUNT=0

    while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
        if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
            log "Health check passed"
            return 0
        fi

        RETRY_COUNT=$((RETRY_COUNT + 1))
        log "Health check attempt ${RETRY_COUNT}/${MAX_RETRIES} failed"
        sleep 3
    done

    error_exit "Health check failed"
}

rollback() {
    log "Rolling back..."

    if [ -f "${PROJECT_ROOT}/captchax-server.bak" ]; then
        cp "${PROJECT_ROOT}/captchax-server.bak" "${PROJECT_ROOT}/captchax-server"
    fi

    if systemctl is-active --quiet captchax.service; then
        systemctl restart captchax.service
    fi

    log "Rollback completed"
}

main() {
    log "========================================="
    log "CaptchaX Deployment Script"
    log "Version: ${VERSION:-dev}"
    log "Timestamp: ${TIMESTAMP}"
    log "========================================="

    mkdir -p "${PROJECT_ROOT}/logs"

    check_env

    if [ -f "${PROJECT_ROOT}/captchax-server" ]; then
        cp "${PROJECT_ROOT}/captchax-server" "${PROJECT_ROOT}/captchax-server.bak"
    fi

    build_binary

    if [ "${SKIP_MIGRATION:-false}" != "true" ]; then
        run_migrations || log "Migration skipped or failed"
    fi

    install_service

    restart_service

    health_check

    if [ -f "${PROJECT_ROOT}/captchax-server.bak" ]; then
        rm "${PROJECT_ROOT}/captchax-server.bak"
    fi

    log "========================================="
    log "Deployment completed successfully!"
    log "========================================="
}

trap 'rollback' ERR INT TERM

main "$@"
