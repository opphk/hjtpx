#!/bin/sh
# =============================================================================
# Docker配置验证脚本
# =============================================================================

set -e

cd "$(dirname "$0")"

echo "========================================"
echo "  Docker配置验证"
echo "========================================"
echo ""

errors=0

# 检查Dockerfile
echo "检查 Dockerfile..."
if [ -f "Dockerfile" ]; then
    echo "✓ Dockerfile 存在"
    if grep -q "FROM golang:1.21-alpine AS builder" Dockerfile; then
        echo "✓ 使用多阶段构建"
    fi
    if grep -q "HEALTHCHECK" Dockerfile; then
        echo "✓ 包含健康检查"
    fi
else
    echo "✗ Dockerfile 不存在"
    errors=$((errors + 1))
fi
echo ""

# 检查docker-compose.yml
echo "检查 docker-compose.yml..."
if [ -f "docker-compose.yml" ]; then
    echo "✓ docker-compose.yml 存在"
    if grep -q "postgres:" docker-compose.yml; then
        echo "✓ PostgreSQL 服务已配置"
    fi
    if grep -q "redis:" docker-compose.yml; then
        echo "✓ Redis 服务已配置"
    fi
    if grep -q "healthcheck:" docker-compose.yml; then
        echo "✓ 健康检查已配置"
    fi
    if grep -q "depends_on:" docker-compose.yml; then
        echo "✓ 依赖关系已配置"
    fi
else
    echo "✗ docker-compose.yml 不存在"
    errors=$((errors + 1))
fi
echo ""

# 检查健康检查脚本
echo "检查健康检查脚本..."
if [ -f "docker/health-check.sh" ]; then
    echo "✓ docker/health-check.sh 存在"
else
    echo "✗ docker/health-check.sh 不存在"
    errors=$((errors + 1))
fi

if [ -f "docker/entrypoint.sh" ]; then
    echo "✓ docker/entrypoint.sh 存在"
else
    echo "✗ docker/entrypoint.sh 不存在"
    errors=$((errors + 1))
fi
echo ""

# 检查PostgreSQL和Redis配置
echo "检查数据库配置文件..."
if [ -f "scripts/docker/postgres-init.sh" ]; then
    echo "✓ PostgreSQL初始化脚本存在"
else
    echo "✗ PostgreSQL初始化脚本不存在"
    errors=$((errors + 1))
fi

if [ -f "scripts/docker/redis.conf" ]; then
    echo "✓ Redis配置文件存在"
else
    echo "✗ Redis配置文件不存在"
    errors=$((errors + 1))
fi
echo ""

# 检查.env.example
echo "检查环境变量模板..."
if [ -f ".env.example" ]; then
    echo "✓ .env.example 存在"
else
    echo "✗ .env.example 不存在"
    errors=$((errors + 1))
fi
echo ""

# 检查.dockerignore
echo "检查.dockerignore..."
if [ -f ".dockerignore" ]; then
    echo "✓ .dockerignore 存在"
else
    echo "✗ .dockerignore 不存在"
    errors=$((errors + 1))
fi
echo ""

echo "========================================"
if [ $errors -eq 0 ]; then
    echo "✓ 所有Docker配置验证通过!"
    echo "========================================"
    exit 0
else
    echo "✗ 发现 $errors 个错误"
    echo "========================================"
    exit 1
fi
