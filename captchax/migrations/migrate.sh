#!/bin/bash

# CaptchaX Database Migration Script
# A comprehensive migration management tool for CaptchaX

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MIGRATIONS_DIR="$SCRIPT_DIR"
DEFAULT_DB_HOST=${DB_HOST:-localhost}
DEFAULT_DB_PORT=${DB_PORT:-5432}
DEFAULT_DB_NAME=${DB_NAME:-captcha_db}
DEFAULT_DB_USER=${DB_USER:-postgres}
DEFAULT_DB_PASSWORD=${DB_PASSWORD:-postgres}
DEFAULT_DB_SSLMODE=${DB_SSLMODE:-disable}

# Print banner
print_banner() {
    echo -e "${BLUE}"
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║                     CaptchaX Migrations                      ║"
    echo "║                  Database Migration Manager                  ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

# Print usage
print_usage() {
    echo "Usage: $0 [COMMAND] [OPTIONS]"
    echo ""
    echo "Commands:"
    echo "  up [N]          Apply all or N up migrations"
    echo "  down [N]        Rollback all or N down migrations"
    echo "  goto V          Migrate to version V"
    echo "  version         Print current migration version"
    echo "  force V         Set version V without running migrations"
    echo "  drop            Drop everything in the database"
    echo "  create NAME     Create a new migration with the given NAME"
    echo "  status          Show migration status"
    echo "  help            Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  DB_HOST         Database host (default: localhost)"
    echo "  DB_PORT         Database port (default: 5432)"
    echo "  DB_NAME         Database name (default: captcha_db)"
    echo "  DB_USER         Database user (default: postgres)"
    echo "  DB_PASSWORD     Database password (default: postgres)"
    echo "  DB_SSLMODE      SSL mode (default: disable)"
    echo ""
    echo "Examples:"
    echo "  $0 up                    # Apply all migrations"
    echo "  $0 up 1                  # Apply 1 migration"
    echo "  $0 down                  # Rollback all migrations"
    echo "  $0 down 1                # Rollback 1 migration"
    echo "  $0 goto 003              # Migrate to version 003"
    echo "  $0 create add_new_table  # Create new migration files"
}

# Build database URL
build_db_url() {
    echo "postgres://${DB_USER:-$DEFAULT_DB_USER}:${DB_PASSWORD:-$DEFAULT_DB_PASSWORD}@${DB_HOST:-$DEFAULT_DB_HOST}:${DB_PORT:-$DEFAULT_DB_PORT}/${DB_NAME:-$DEFAULT_DB_NAME}?sslmode=${DB_SSLMODE:-$DEFAULT_DB_SSLMODE}"
}

# Check if golang-migrate is installed
check_migrate_installed() {
    if ! command -v migrate &> /dev/null; then
        echo -e "${YELLOW}golang-migrate not found, installing...${NC}"
        install_migrate
    fi
}

# Install golang-migrate
install_migrate() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    if [ "$ARCH" = "x86_64" ]; then
        ARCH="amd64"
    fi
    
    VERSION="4.17.0"
    URL="https://github.com/golang-migrate/migrate/releases/download/v${VERSION}/migrate.${OS}-${ARCH}.tar.gz"
    
    echo -e "${BLUE}Downloading migrate v${VERSION}...${NC}"
    
    TMP_DIR=$(mktemp -d)
    curl -L "$URL" -o "$TMP_DIR/migrate.tar.gz"
    tar -xzf "$TMP_DIR/migrate.tar.gz" -C "$TMP_DIR"
    sudo mv "$TMP_DIR/migrate.${OS}-${ARCH}" /usr/local/bin/migrate
    sudo chmod +x /usr/local/bin/migrate
    
    rm -rf "$TMP_DIR"
    
    if command -v migrate &> /dev/null; then
        echo -e "${GREEN}migrate installed successfully!${NC}"
    else
        echo -e "${RED}Failed to install migrate${NC}"
        exit 1
    fi
}

# Run migrate command
run_migrate() {
    local cmd=$1
    shift
    local args="$@"
    
    DB_URL=$(build_db_url)
    echo -e "${BLUE}Running migration: migrate -path \"$MIGRATIONS_DIR\" -database \"$DB_URL\" $cmd $args${NC}"
    echo ""
    
    migrate -path "$MIGRATIONS_DIR" -database "$DB_URL" $cmd $args
}

# Create new migration
create_migration() {
    local name=$1
    if [ -z "$name" ]; then
        echo -e "${RED}Error: Migration name required${NC}"
        print_usage
        exit 1
    fi
    
    local timestamp=$(date +%Y%m%d%H%M%S)
    local up_file="${MIGRATIONS_DIR}/${timestamp}_${name}.up.sql"
    local down_file="${MIGRATIONS_DIR}/${timestamp}_${name}.down.sql"
    
    cat > "$up_file" <<EOF
-- ${name} Up Migration
-- Created: $(date +%Y-%m-%d)

BEGIN;

-- Add your migration here

COMMIT;
EOF
    
    cat > "$down_file" <<EOF
-- ${name} Down Migration
-- Created: $(date +%Y-%m-%d)

BEGIN;

-- Add your rollback here

COMMIT;
EOF
    
    echo -e "${GREEN}Created migration files:${NC}"
    echo "  $up_file"
    echo "  $down_file"
}

# Main function
main() {
    print_banner
    
    local command=$1
    shift
    
    check_migrate_installed
    
    case $command in
        up)
            run_migrate up "$@"
            ;;
        down)
            run_migrate down "$@"
            ;;
        goto)
            run_migrate goto "$@"
            ;;
        version)
            run_migrate version
            ;;
        force)
            run_migrate force "$@"
            ;;
        drop)
            echo -e "${YELLOW}WARNING: This will drop ALL tables and data!${NC}"
            read -p "Are you sure? (y/N) " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                run_migrate drop -f
            fi
            ;;
        create)
            create_migration "$@"
            ;;
        status)
            DB_URL=$(build_db_url)
            echo -e "${BLUE}Migration Directory: $MIGRATIONS_DIR${NC}"
            echo -e "${BLUE}Database: $DB_URL${NC}"
            echo ""
            ls -la "$MIGRATIONS_DIR"
            ;;
        help|--help|-h)
            print_usage
            ;;
        *)
            echo -e "${RED}Error: Unknown command '$command'${NC}"
            echo ""
            print_usage
            exit 1
            ;;
    esac
}

# Execute main
main "$@"
