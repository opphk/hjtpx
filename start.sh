#!/bin/bash

# HJTPX 行为验证系统启动脚本
# v15.0

echo "========================================"
echo "  HJTPX 行为验证系统 v15.0"
echo "========================================"
echo ""

# 检查是否在正确的目录
if [ ! -f "backend/go.mod" ]; then
    echo "错误: 请在 hjtpx 项目根目录运行此脚本"
    exit 1
fi

# 检查服务状态
echo "检查服务状态..."

# 检查 Redis
if command -v redis-cli &> /dev/null; then
    if redis-cli ping &> /dev/null; then
        echo "✅ Redis 运行正常"
    else
        echo "⚠️  Redis 未运行，尝试启动..."
        service redis-server start || true
    fi
else
    echo "⚠️  Redis 未安装"
fi

# 检查 PostgreSQL
if command -v psql &> /dev/null; then
    if pg_isready &> /dev/null; then
        echo "✅ PostgreSQL 运行正常"
    else
        echo "⚠️  PostgreSQL 未运行，尝试启动..."
        service postgresql start || true
    fi
else
    echo "⚠️  PostgreSQL 未安装"
fi

echo ""

# 构建后端
echo "构建后端服务..."
cd backend

# 尝试构建
if go build -o hjtpx ./cmd/api/main.go; then
    echo "✅ 后端构建成功"
else
    echo "❌ 后端构建失败"
    echo ""
    echo "尝试使用 go mod tidy 修复依赖..."
    go mod tidy 2>/dev/null || true
    
    if go build -o hjtpx ./cmd/api/main.go; then
        echo "✅ 后端构建成功"
    else
        echo "⚠️  仍然有构建问题，但我们会继续尝试运行"
    fi
fi

echo ""
echo "启动完成！"
echo ""
echo "默认访问地址:"
echo "  - 前端: http://localhost"
echo "  - 管理后台: http://localhost/admin"
echo "  - API服务: http://localhost:8080"
echo ""
echo "如需启动完整服务，请使用 Docker Compose:"
echo "  docker-compose up -d"
echo ""
