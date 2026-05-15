#!/bin/bash
# 任务16：Docker镜像优化
# 使用多阶段构建
# 优化层缓存
# 减少镜像体积
# 添加.dockerignore
# 测试构建速度

echo "=========================================="
echo "任务16：Docker镜像优化"
echo "=========================================="

cd /workspace/hjtpx

# 1. 优化主Dockerfile
echo "[16.1] 优化主Dockerfile - 多阶段构建..."

cat > Dockerfile.optimized << 'EOF'
# 第一阶段：构建
FROM node:18-alpine AS builder

WORKDIR /app

# 先复制package文件以利用缓存
COPY package*.json ./

# 安装依赖
RUN npm ci --only=production

# 复制源代码
COPY src ./src
COPY scripts ./scripts
COPY configs ./configs

# 设置环境变量
ENV NODE_ENV=production
ENV PORT=3000

# 运行健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://localhost:3000/health || exit 1

# 第二阶段：运行
FROM node:18-alpine AS runner

# 创建非root用户
RUN addgroup -g 1001 -S nodejs && \
    adduser -S nodejs -u 1001

WORKDIR /app

# 复制node_modules（只包含生产依赖）
COPY --from=builder /app/node_modules ./node_modules

# 复制应用代码
COPY --from=builder /app/src ./src
COPY --from=builder /app/scripts ./scripts
COPY --from=builder /app/configs ./configs

# 复制package文件
COPY --from=builder /app/package*.json ./

# 设置环境变量
ENV NODE_ENV=production
ENV PORT=3000

# 改变文件所有者
RUN chown -R nodejs:nodejs /app

# 切换到非root用户
USER nodejs

# 暴露端口
EXPOSE 3000

# 启动命令
CMD ["node", "src/index.js"]
EOF

# 2. 优化前端Dockerfile
echo "[16.2] 优化前端Dockerfile - 多阶段构建..."

cat > Dockerfile.frontend.optimized << 'EOF'
# 第一阶段：构建
FROM node:18-alpine AS builder

WORKDIR /app

# 复制package文件
COPY package*.json ./

# 安装依赖（包括devDependencies用于构建）
RUN npm ci

# 复制源代码
COPY src/frontend ./src/frontend
COPY public ./public

# 设置环境变量
ENV NODE_ENV=production
ENV REACT_APP_API_URL=/api

# 构建应用
RUN npm run build

# 第二阶段：运行
FROM nginx:alpine AS runner

# 复制nginx配置
COPY nginx-production.conf /etc/nginx/conf.d/default.conf

# 从builder阶段复制构建产物
COPY --from=builder /app/src/frontend/build /usr/share/nginx/html

# 创建非root用户
RUN chown -R nginx:nginx /usr/share/nginx/html && \
    chown -R nginx:nginx /var/cache/nginx && \
    chown -R nginx:nginx /var/log/nginx && \
    touch /var/run/nginx.pid && \
    chown -R nginx:nginx /var/run/nginx.pid

# 暴露端口
EXPOSE 80

# 健康检查
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://localhost/health || exit 1

# 启动nginx
CMD ["nginx", "-g", "daemon off;"]
EOF

# 3. 优化docker-compose配置
echo "[16.3] 优化docker-compose配置..."

cat > docker-compose.optimized.yml << 'EOF'
version: '3.8'

services:
  app:
    build:
      context: .
      dockerfile: Dockerfile.optimized
    image: hjtpx-app:latest
    container_name: hjtpx-app
    restart: unless-stopped
    ports:
      - "3000:3000"
    environment:
      - NODE_ENV=production
      - PORT=3000
      - DATABASE_URL=${DATABASE_URL}
      - REDIS_URL=${REDIS_URL}
    volumes:
      - ./logs:/app/logs
      - ./uploads:/app/uploads
    networks:
      - hjtpx-network
    depends_on:
      - postgres
      - redis
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 1G
        reservations:
          cpus: '0.5'
          memory: 512M
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

  frontend:
    build:
      context: .
      dockerfile: Dockerfile.frontend.optimized
    image: hjtpx-frontend:latest
    container_name: hjtpx-frontend
    restart: unless-stopped
    ports:
      - "80:80"
    depends_on:
      - app
    networks:
      - hjtpx-network

  postgres:
    image: postgres:15-alpine
    container_name: hjtpx-postgres
    restart: unless-stopped
    environment:
      - POSTGRES_USER=${POSTGRES_USER:-hjtpx}
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
      - POSTGRES_DB=${POSTGRES_DB:-hjtpx}
    volumes:
      - postgres-data:/var/lib/postgresql/data
    networks:
      - hjtpx-network
    command: >
      postgres
      -c max_connections=200
      -c shared_buffers=256MB
      -c effective_cache_size=512MB
      -c maintenance_work_mem=64MB
      -c checkpoint_completion_target=0.9
      -c wal_buffers=16MB
      -c default_statistics_target=100
      -c random_page_cost=1.1
      -c effective_io_concurrency=200
      -c work_mem=4MB
      -c min_wal_size=1GB
      -c max_wal_size=4GB

  redis:
    image: redis:7-alpine
    container_name: hjtpx-redis
    restart: unless-stopped
    command: redis-server --maxmemory 512mb --maxmemory-policy allkeys-lru --save "" --appendonly no
    volumes:
      - redis-data:/data
    networks:
      - hjtpx-network

networks:
  hjtpx-network:
    driver: bridge

volumes:
  postgres-data:
    driver: local
  redis-data:
    driver: local
EOF

# 4. 创建.dockerignore
echo "[16.4] 创建.dockerignore..."

cat > .dockerignore << 'EOF'
# 版本控制
.git
.gitignore
.gitattributes

# IDE
.vscode
.idea
*.swp
*.swo
*~

# 环境文件
.env
.env.*
!.env.example

# 文档
*.md
docs/
LICENSE

# 测试
tests/
coverage/
*.test.js
*.spec.js
jest.config.js
playwright.config.js

# 开发工具
node_modules/
npm-debug.log*
yarn-debug.log*
yarn-error.log*
.npm
.yarn

# 构建产物
dist/
build/

# 其他
.DS_Store
Thumbs.db
*.log
logs/
*.pid
*.seed
*.pid.lock

# 前端开发文件
src/frontend/src/**/*.test.js
src/frontend/src/**/*.test.jsx
src/frontend/src/**/*.stories.jsx
src/frontend/src/**/*.scss
src/frontend/public/mock/
src/frontend/src/mocks/

# Storybook
.storybook/
storybook-static/

# CI/CD
.github/
.gitlab-ci.yml
.travis.yml
.jenkins/

# 备份和临时文件
backup/
tmp/
temp/
*.bak
EOF

# 5. 创建构建优化脚本
echo "[16.5] 创建构建优化脚本..."

cat > scripts/docker-build-optimized.sh << 'EOF'
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
EOF

chmod +x scripts/docker-build-optimized.sh

# 6. 创建镜像分析脚本
echo "[16.6] 创建镜像分析脚本..."

cat > scripts/analyze-docker-size.sh << 'EOF'
#!/bin/bash

echo "Docker镜像大小分析"
echo "================================"

# 分析每个镜像的层
echo ""
echo "镜像层大小分析:"
for image in $(docker images --format "{{.Repository}}:{{.Tag}}" | grep hjtpx); do
  echo ""
  echo "镜像: $image"
  docker history "$image" --no-trunc --format "{{.Size}}\t{{.CreatedBy}}" | \
    awk '{size=$1; cmd=$2; if (size ~ /MB/) {mb+=substr(size,1,length(size)-2)} else if (size ~ /KB/) {kb+=substr(size,1,length(size)-2)/1024}} END {printf "  总大小: %.2f MB\n", mb+kb}'
done

echo ""
echo "最大的层:"
docker images | grep hjtpx | head -1 | awk '{print $3}' | xargs -I {} docker history {} --no-trunc --format "{{.Size}}\t{{.CreatedBy}}" | sort -hr | head -10

echo ""
echo "优化建议:"
echo "- 使用多阶段构建分离构建和运行环境"
echo "- 合并RUN指令减少层数"
echo "- 使用.dockerignore排除不必要的文件"
echo "- 利用构建缓存优化构建速度"
echo "- 考虑使用更小的基础镜像(alpine)"
EOF

chmod +x scripts/analyze-docker-size.sh

echo "=========================================="
echo "任务16完成：Docker镜像优化"
echo "=========================================="
