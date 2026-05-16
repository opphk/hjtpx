#!/bin/bash
set -e

echo "===== HJTPX 停止脚本 ====="

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

echo "停止所有服务..."
if command -v docker-compose &> /dev/null; then
    docker-compose down
else
    docker compose down
fi

echo "清理未使用的镜像..."
docker image prune -f

echo "===== 停止完成 ====="
