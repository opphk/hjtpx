#!/bin/bash
set -e

echo "开始优化Docker构建..."

# 清理旧的构建缓存和产物
echo "1. 清理旧构建..."
docker system prune -f --volumes || true

# 记录构建时间
START_TIME=$(date +%s)

# 构建后端镜像
echo "2. 构建后端镜像..."
docker build \
  --target builder \
  -f Dockerfile.optimized \
  -t hjtpx-app:builder \
  --cache-from hjtpx-app:latest \
  --pull \
  .

# 构建前端镜像
echo "3. 构建前端镜像..."
docker build \
  --target builder \
  -f Dockerfile.frontend.optimized \
  -t hjtpx-frontend:builder \
  --cache-from hjtpx-frontend:latest \
  --pull \
  .

# 记录构建时间
END_TIME=$(date +%s)
BUILD_TIME=$((END_TIME - START_TIME))

echo "================================"
echo "构建完成！"
echo "总耗时: ${BUILD_TIME}秒"
echo "================================"

# 显示镜像大小
echo ""
echo "镜像大小:"
docker images | grep hjtpx

# 启动容器
echo ""
echo "4. 启动容器..."
docker-compose -f docker-compose.optimized.yml up -d

echo ""
echo "5. 验证容器状态..."
sleep 5
docker-compose -f docker-compose.optimized.yml ps
