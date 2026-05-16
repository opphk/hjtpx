#!/bin/bash
set -e

echo "===== HJTPX 更新脚本 ====="

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

echo "1. 拉取最新代码..."
git pull origin main

echo "2. 重新构建Docker镜像..."
if command -v docker-compose &> /dev/null; then
    docker-compose build --no-cache app
else
    docker compose build --no-cache app
fi

echo "3. 重启应用容器..."
if command -v docker-compose &> /dev/null; then
    docker-compose up -d app
else
    docker compose up -d app
fi

echo "4. 检查应用状态..."
sleep 5
if command -v docker-compose &> /dev/null; then
    docker-compose ps app
else
    docker compose ps app
fi

echo "===== 更新完成 ====="
