#!/bin/bash
set -e

echo "===== HJTPX 部署脚本 ====="
echo "开始部署..."

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

if [ ! -f ".env" ]; then
    echo "警告: .env 文件不存在，复制 .env.example 作为模板"
    cp .env.example .env
    echo "请编辑 .env 文件配置您的环境变量"
    echo "完成后重新运行此脚本"
    exit 1
fi

echo "1. 检查Docker环境..."
if ! command -v docker &> /dev/null; then
    echo "错误: Docker 未安装"
    exit 1
fi

if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo "错误: Docker Compose 未安装"
    exit 1
fi

echo "2. 创建必要目录..."
mkdir -p backend/config
mkdir -p nginx/ssl
mkdir -p monitoring/prometheus/rules
mkdir -p monitoring/grafana/provisioning/dashboards

echo "3. 生成SSL证书（用于测试）..."
if [ ! -f "nginx/ssl/cert.pem" ]; then
    openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
        -keyout nginx/ssl/key.pem \
        -out nginx/ssl/cert.pem \
        -subj "/C=CN/ST=Beijing/L=Beijing/O=HJTPX/CN=localhost"
    echo "SSL证书已生成"
fi

echo "4. 构建Docker镜像..."
if command -v docker-compose &> /dev/null; then
    docker-compose build --no-cache
else
    docker compose build --no-cache
fi

echo "5. 启动服务..."
if command -v docker-compose &> /dev/null; then
    docker-compose up -d
else
    docker compose up -d
fi

echo "6. 等待服务启动..."
sleep 10

echo "7. 检查服务状态..."
if command -v docker-compose &> /dev/null; then
    docker-compose ps
else
    docker compose ps
fi

echo ""
echo "===== 部署完成 ====="
echo ""
echo "服务地址:"
echo "  - 应用API: http://localhost:${APP_PORT:-8080}"
echo "  - 前端页面: http://localhost:${NGINX_PORT:-80}"
echo "  - Prometheus: http://localhost:${PROMETHEUS_PORT:-9090}"
echo "  - Grafana: http://localhost:${GRAFANA_PORT:-3000}"
echo "  - Loki: http://localhost:${LOKI_PORT:-3100}"
echo ""
echo "默认管理员账号:"
echo "  - 用户名: admin"
echo "  - 密码: admin123"
echo ""
echo "查看日志: docker-compose logs -f [service]"
echo "停止服务: docker-compose down"
echo "重新部署: ./scripts/deploy.sh"
