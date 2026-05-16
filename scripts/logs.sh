#!/bin/bash
set -e

echo "===== HJTPX 日志查看脚本 ====="

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

SERVICE="${1:-app}"

echo "查看 $SERVICE 服务日志 (Ctrl+C 退出)..."
if command -v docker-compose &> /dev/null; then
    docker-compose logs -f "$SERVICE"
else
    docker compose logs -f "$SERVICE"
fi
