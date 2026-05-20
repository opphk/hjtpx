#!/bin/bash

# =============================================================================
# HJTPX 行为验证系统启动脚本
# v20.0
# =============================================================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

# 显示横幅
echo ""
echo "========================================"
echo "  HJTPX 行为验证系统 v20.0"
echo "  AGI智能验证 | 多模态验证 | 轨迹加密"
echo "========================================"
echo ""

# 检查是否在正确的目录
if [ ! -f "backend/go.mod" ]; then
    log_error "请在 hjtpx 项目根目录运行此脚本"
    exit 1
fi

# 检查Docker环境
check_docker() {
    log_info "检查Docker环境..."
    
    if command -v docker &> /dev/null; then
        if docker ps &> /dev/null; then
            log_success "Docker 运行正常"
            return 0
        else
            log_warning "Docker 未运行或没有权限"
            return 1
        fi
    else
        log_warning "Docker 未安装"
        return 1
    fi
}

# 检查Docker Compose
check_docker_compose() {
    log_info "检查Docker Compose..."
    
    if command -v docker-compose &> /dev/null; then
        log_success "Docker Compose 可用"
        return 0
    elif docker compose version &> /dev/null; then
        log_success "Docker Compose (v2) 可用"
        return 0
    else
        log_warning "Docker Compose 未安装"
        return 1
    fi
}

# 检查服务状态
check_services() {
    log_info "检查服务状态..."
    
    # 检查 Redis
    if command -v redis-cli &> /dev/null; then
        if redis-cli ping &> /dev/null; then
            log_success "Redis 运行正常"
        else
            log_warning "Redis 未运行"
        fi
    else
        log_warning "Redis 客户端未安装"
    fi
    
    # 检查 PostgreSQL
    if command -v pg_isready &> /dev/null; then
        if pg_isready &> /dev/null; then
            log_success "PostgreSQL 运行正常"
        else
            log_warning "PostgreSQL 未运行"
        fi
    else
        log_warning "PostgreSQL 客户端未安装"
    fi
}

# 检查端口占用
check_port() {
    local port=$1
    log_info "检查端口 ${port}..."
    
    if lsof -i:${port} &> /dev/null; then
        log_warning "端口 ${port} 已被占用"
        return 1
    else
        log_success "端口 ${port} 可用"
        return 0
    fi
}

# Docker Compose 启动
start_docker_compose() {
    log_info "启动Docker Compose服务..."
    
    # 创建必要的目录
    mkdir -p logs/postgres logs/redis data/postgres data/redis
    
    # 启动服务
    docker-compose up -d
    
    # 等待服务启动
    log_info "等待服务启动..."
    sleep 10
    
    # 检查服务状态
    if docker-compose ps | grep -q "Up"; then
        log_success "Docker Compose 服务启动成功"
        
        # 检查健康状态
        local retries=5
        while [ $retries -gt 0 ]; do
            if curl -sf http://localhost:8080/health &> /dev/null; then
                log_success "应用健康检查通过"
                return 0
            fi
            log_info "等待应用启动... (剩余 $retries 次)"
            sleep 5
            retries=$((retries - 1))
        done
        
        log_warning "应用可能尚未完全启动，请稍后检查"
        return 0
    else
        log_error "Docker Compose 服务启动失败"
        docker-compose logs
        return 1
    fi
}

# 构建后端
build_backend() {
    log_info "构建后端服务..."
    cd backend
    
    # 尝试构建
    if go build -o hjtpx ./cmd/api/main.go; then
        log_success "后端构建成功"
        cd ..
        return 0
    else
        log_error "后端构建失败"
        echo ""
        log_info "尝试使用 go mod tidy 修复依赖..."
        cd ..
        go mod tidy 2>/dev/null || true
        
        cd backend
        if go build -o hjtpx ./cmd/api/main.go; then
            log_success "后端构建成功"
            cd ..
            return 0
        else
            log_error "仍然有构建问题"
            cd ..
            return 1
        fi
    fi
}

# 显示访问信息
show_access_info() {
    echo ""
    echo "========================================"
    echo "  启动完成！"
    echo "========================================"
    echo ""
    echo "默认访问地址:"
    echo "  - 前端: http://localhost"
    echo "  - 管理后台: http://localhost/admin"
    echo "  - API服务: http://localhost:8080"
    echo "  - 健康检查: http://localhost:8080/health"
    echo "  - AGI健康: http://localhost:8080/health/agi"
    echo "  - Prometheus: http://localhost:9090 (如已配置)"
    echo "  - Grafana: http://localhost:3000 (如已配置)"
    echo ""
    echo "Docker Compose 命令:"
    echo "  启动服务: docker-compose up -d"
    echo "  查看日志: docker-compose logs -f"
    echo "  停止服务: docker-compose down"
    echo ""
}

# 主函数
main() {
    local mode=${1:-"docker"}
    
    case $mode in
        docker)
            log_info "使用Docker Compose模式启动..."
            check_docker
            check_docker_compose
            
            if check_port 8080; then
                start_docker_compose
                show_access_info
            else
                log_warning "端口检查未通过，跳过Docker启动"
                show_access_info
            fi
            ;;
        build)
            log_info "使用本地构建模式..."
            check_services
            build_backend
            show_access_info
            ;;
        check)
            log_info "仅检查环境..."
            check_docker
            check_docker_compose
            check_services
            ;;
        *)
            echo "用法: $0 [docker|build|check]"
            echo ""
            echo "  docker - 使用Docker Compose启动 (默认)"
            echo "  build - 本地构建并启动"
            echo "  check - 仅检查环境"
            exit 1
            ;;
    esac
}

# 运行主函数
main "$@"
