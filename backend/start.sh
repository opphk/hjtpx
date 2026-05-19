#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(dirname "$SCRIPT_DIR")"
cd "$BACKEND_DIR"

echo "========================================"
echo "HJTPX Captcha System Startup Script"
echo "========================================"
echo ""

echo "[1/4] Checking configuration..."
if [ ! -f "config/config.yaml" ]; then
    echo "Warning: config/config.yaml not found, using default configuration"
fi

echo "[2/4] Checking database connection..."
timeout 5 bash -c 'cat < /dev/null > /dev/tcp/localhost/5432' 2>/dev/null
if [ $? -eq 0 ]; then
    echo "  ✓ PostgreSQL is reachable"
else
    echo "  ⚠ PostgreSQL is not reachable on localhost:5432"
    echo "    Application will continue but database features may not work"
fi

echo "[3/4] Checking Redis connection..."
timeout 5 bash -c 'cat < /dev/null > /dev/tcp/localhost/6379' 2>/dev/null
if [ $? -eq 0 ]; then
    echo "  ✓ Redis is reachable"
else
    echo "  ⚠ Redis is not reachable on localhost:6379"
    echo "    Application will continue but caching features may not work"
fi

echo "[4/4] Starting backend server..."
echo ""

if [ ! -f "./hjtpx" ]; then
    echo "Error: hjtpx executable not found!"
    echo "Please run 'go build -o hjtpx ./cmd/api/main.go' first"
    exit 1
fi

export GIN_MODE=release
./hjtpx
