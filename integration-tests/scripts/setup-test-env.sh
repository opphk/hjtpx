#!/bin/bash

# HJTPX 集成测试环境设置脚本 v15.0

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查服务健康
check_service_health() {
    local service=$1
    local url=$2
    local max_attempts=${3:-30}
    
    log_info "检查 $service 服务健康状态..."
    
    for i in $(seq 1 $max_attempts); do
        if curl -sf "$url" > /dev/null 2>&1; then
            log_success "$service 服务健康检查通过"
            return 0
        fi
        log_warning "等待 $service 启动... ($i/$max_attempts)"
        sleep 2
    done
    
    log_error "$service 服务启动超时"
    return 1
}

# 等待数据库就绪
wait_for_postgres() {
    log_info "等待 PostgreSQL 就绪..."
    
    for i in $(seq 1 30); do
        if docker-compose -f "$PROJECT_ROOT/docker-compose-test.yml" exec -T postgres \
            pg_isready -U hjtpx_test -d hjtpx_test > /dev/null 2>&1; then
            log_success "PostgreSQL 已就绪"
            return 0
        fi
        log_warning "等待 PostgreSQL... ($i/30)"
        sleep 2
    done
    
    log_error "PostgreSQL 启动超时"
    return 1
}

# 等待Redis就绪
wait_for_redis() {
    log_info "等待 Redis 就绪..."
    
    for i in $(seq 1 30); do
        if docker-compose -f "$PROJECT_ROOT/docker-compose-test.yml" exec -T redis \
            redis-cli -a hjtpx_test_password ping > /dev/null 2>&1; then
            log_success "Redis 已就绪"
            return 0
        fi
        log_warning "等待 Redis... ($i/30)"
        sleep 2
    done
    
    log_error "Redis 启动超时"
    return 1
}

# 初始化测试数据库
init_test_database() {
    log_info "初始化测试数据库..."
    
    cd "$PROJECT_ROOT"
    
    # 运行数据库迁移
    if [ -d "$PROJECT_ROOT/scripts/migrate" ]; then
        docker-compose -f docker-compose-test.yml exec -T app \
            /app/migrate.sh up 2>/dev/null || true
    fi
    
    # 创建测试数据
    docker-compose -f docker-compose-test.yml exec -T postgres \
        psql -U hjtpx_test -d hjtpx_test -c "
        -- 创建测试用户表（如果不存在）
        CREATE TABLE IF NOT EXISTS test_users (
            id SERIAL PRIMARY KEY,
            username VARCHAR(50) UNIQUE NOT NULL,
            email VARCHAR(100) UNIQUE NOT NULL,
            password_hash VARCHAR(255) NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );
        
        -- 创建测试应用表（如果不存在）
        CREATE TABLE IF NOT EXISTS test_applications (
            id SERIAL PRIMARY KEY,
            name VARCHAR(100) NOT NULL,
            app_key VARCHAR(100) UNIQUE NOT NULL,
            app_secret VARCHAR(255) NOT NULL,
            status VARCHAR(20) DEFAULT 'active',
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );
        " || true
    
    log_success "测试数据库初始化完成"
}

# 清理测试数据
cleanup_test_data() {
    log_info "清理测试数据..."
    
    cd "$PROJECT_ROOT"
    
    docker-compose -f docker-compose-test.yml exec -T postgres \
        psql -U hjtpx_test -d hjtpx_test -c "
        DELETE FROM test_users WHERE username LIKE 'test%';
        DELETE FROM test_applications WHERE name LIKE 'Test%';
        " || true
    
    docker-compose -f docker-compose-test.yml exec -T redis \
        redis-cli -a hjtpx_test_password FLUSHDB || true
    
    log_success "测试数据清理完成"
}

# 验证测试环境
validate_test_environment() {
    log_info "验证测试环境..."
    
    local errors=0
    
    # 检查主应用
    if ! curl -sf http://localhost:8080/health > /dev/null 2>&1; then
        log_error "主应用健康检查失败"
        ((errors++))
    else
        log_success "主应用健康检查通过"
    fi
    
    # 检查数据库连接
    if ! docker-compose -f "$PROJECT_ROOT/docker-compose-test.yml" exec -T postgres \
        pg_isready -U hjtpx_test -d hjtpx_test > /dev/null 2>&1; then
        log_error "数据库连接检查失败"
        ((errors++))
    else
        log_success "数据库连接检查通过"
    fi
    
    # 检查Redis连接
    if ! docker-compose -f "$PROJECT_ROOT/docker-compose-test.yml" exec -T redis \
        redis-cli -a hjtpx_test_password ping > /dev/null 2>&1; then
        log_error "Redis连接检查失败"
        ((errors++))
    else
        log_success "Redis连接检查通过"
    fi
    
    if [ $errors -eq 0 ]; then
        log_success "测试环境验证完成，所有服务正常"
        return 0
    else
        log_error "测试环境验证失败，$errors 个服务异常"
        return 1
    fi
}

# 显示测试环境状态
show_environment_status() {
    log_info "测试环境状态:"
    
    cd "$PROJECT_ROOT"
    
    echo ""
    docker-compose -f docker-compose-test.yml ps
    
    echo ""
    log_info "服务健康状态:"
    
    if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
        log_success "✓ 应用服务 (http://localhost:8080)"
    else
        log_error "✗ 应用服务 (http://localhost:8080)"
    fi
    
    if docker-compose -f "$PROJECT_ROOT/docker-compose-test.yml" exec -T postgres \
        pg_isready -U hjtpx_test -d hjtpx_test > /dev/null 2>&1; then
        log_success "✓ PostgreSQL (localhost:5432)"
    else
        log_error "✗ PostgreSQL (localhost:5432)"
    fi
    
    if docker-compose -f "$PROJECT_ROOT/docker-compose-test.yml" exec -T redis \
        redis-cli -a hjtpx_test_password ping > /dev/null 2>&1; then
        log_success "✓ Redis (localhost:6379)"
    else
        log_error "✗ Redis (localhost:6379)"
    fi
    
    echo ""
}

# 显示帮助
show_help() {
    cat << EOF
HJTPX 集成测试环境设置脚本 v15.0

用法: $0 [选项]

选项:
    -h, --help              显示帮助信息
    -w, --wait              等待服务启动
    -i, --init              初始化测试数据库
    -c, --cleanup           清理测试数据
    -v, --validate          验证测试环境
    -s, --status            显示环境状态
    -a, --all               执行完整设置（等待+初始化+验证）

示例:
    $0 --wait               等待服务启动
    $0 --init               初始化测试数据库
    $0 --validate           验证测试环境
    $0 --status             显示环境状态
    $0 --all                执行完整设置

EOF
}

# 主函数
main() {
    case "${1:-}" in
        -h|--help)
            show_help
            ;;
        -w|--wait)
            wait_for_postgres
            wait_for_redis
            check_service_health "应用" "http://localhost:8080/health"
            ;;
        -i|--init)
            wait_for_postgres
            wait_for_redis
            init_test_database
            ;;
        -c|--cleanup)
            cleanup_test_data
            ;;
        -v|--validate)
            validate_test_environment
            ;;
        -s|--status)
            show_environment_status
            ;;
        -a|--all)
            wait_for_postgres
            wait_for_redis
            check_service_health "应用" "http://localhost:8080/health"
            init_test_database
            validate_test_environment
            ;;
        *)
            show_help
            ;;
    esac
}

# 执行主函数
main "$@"
