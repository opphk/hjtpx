#!/bin/bash

# =============================================================================
# HJTPX Backend启动脚本
# 功能：环境检查、服务依赖检测、优雅启动
# 版本：v21.0
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(dirname "$SCRIPT_DIR")"
cd "$BACKEND_DIR"

# =============================================================================
# 颜色定义
# =============================================================================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# =============================================================================
# 日志函数
# =============================================================================
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

# =============================================================================
# 显示启动横幅
# =============================================================================
show_banner() {
    echo "========================================"
    echo "  HJTPX Captcha System v21.0"
    echo "  Backend Startup Script"
    echo "========================================"
    echo ""
}

# =============================================================================
# 环境检查
# =============================================================================
check_environment() {
    log_info "检查运行环境..."

    # 检查操作系统
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        log_info "操作系统: ${PRETTY_NAME:-Unknown}"
    fi

    # 检查系统架构
    log_info "系统架构: $(uname -m)"

    # 检查内存
    if command -v free > /dev/null 2>&1; then
        TOTAL_MEM=$(free -h | awk '/^Mem:/ {print $2}')
        AVAILABLE_MEM=$(free -h | awk '/^Mem:/ {print $7}')
        log_info "内存信息: 总计=${TOTAL_MEM}, 可用=${AVAILABLE_MEM}"
    fi

    # 检查磁盘空间
    if command -v df > /dev/null 2>&1; then
        AVAILABLE_DISK=$(df -h . | awk 'NR==2 {print $4}')
        log_info "磁盘空间: 可用=${AVAILABLE_DISK}"
    fi

    # 检查用户权限
    if [ "$(id -u)" -eq 0 ]; then
        log_warning "以root用户运行，某些功能可能受限"
    else
        log_info "运行用户: $(whoami) (UID: $(id -u))"
    fi

    echo ""
}

# =============================================================================
# 配置检查
# =============================================================================
check_configuration() {
    log_info "检查配置文件..."
    echo ""

    # 检查主配置文件
    if [ ! -f "config/config.yaml" ]; then
        log_warning "config/config.yaml 未找到，将使用默认配置"
    else
        log_success "配置文件: config/config.yaml"
    fi

    # 检查环境变量配置
    if [ -f ".env" ]; then
        log_success ".env 配置文件已存在"
    elif [ -f ".env.example" ]; then
        log_warning ".env 文件不存在，请复制 .env.example 创建"
    else
        log_warning "未找到环境配置文件"
    fi

    echo ""
}

# =============================================================================
# 服务依赖检查
# =============================================================================
check_service_dependencies() {
    log_info "检查服务依赖..."
    echo ""

    # PostgreSQL检查
    check_postgres() {
        log_info "  [1/4] PostgreSQL 连接..."
        if timeout 5 bash -c 'cat < /dev/null > /dev/tcp/localhost/5432' 2>/dev/null; then
            log_success "    ✓ PostgreSQL 端口 (5432) 可达"

            # 尝试数据库特定检查
            if command -v pg_isready > /dev/null 2>&1; then
                if pg_isready -h localhost -p 5432 > /dev/null 2>&1; then
                    log_success "    ✓ PostgreSQL 数据库响应正常"
                else
                    log_warning "    ⚠ PostgreSQL 端口可达但数据库未就绪"
                fi
            fi
        else
            log_warning "    ✗ PostgreSQL 端口 (5432) 不可达"
            log_warning "    应用将继续启动，但数据库功能可能不可用"
        fi
    }

    # Redis检查
    check_redis() {
        log_info "  [2/4] Redis 连接..."
        if timeout 5 bash -c 'cat < /dev/null > /dev/tcp/localhost/6379' 2>/dev/null; then
            log_success "    ✓ Redis 端口 (6379) 可达"

            # 尝试Redis特定检查
            if command -v redis-cli > /dev/null 2>&1; then
                if redis-cli -h localhost -p 6379 ping > /dev/null 2>&1; then
                    log_success "    ✓ Redis 服务响应正常"
                else
                    log_warning "    ⚠ Redis 端口可达但服务未就绪"
                fi
            fi
        else
            log_warning "    ✗ Redis 端口 (6379) 不可达"
            log_warning "    应用将继续启动，但缓存功能可能不可用"
        fi
    }

    # 健康检查端点检查
    check_health_endpoint() {
        log_info "  [3/4] 健康检查端点..."
        if command -v curl > /dev/null 2>&1; then
            if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
                log_success "    ✓ 健康检查端点可访问"
            else
                log_info "    ℹ 健康检查端点尚未启动（应用可能还在启动中）"
            fi
        elif command -v wget > /dev/null 2>&1; then
            if wget -q -O- http://localhost:8080/health > /dev/null 2>&1; then
                log_success "    ✓ 健康检查端点可访问"
            else
                log_info "    ℹ 健康检查端点尚未启动（应用可能还在启动中）"
            fi
        else
            log_warning "    ⚠ 未找到 curl 或 wget，无法检查健康端点"
        fi
    }

    # 网络连接检查
    check_network() {
        log_info "  [4/4] 网络连接..."
        if command -v netstat > /dev/null 2>&1; then
            LISTENING_PORTS=$(netstat -tuln 2>/dev/null | grep LISTEN | wc -l)
            log_info "    监听端口数: ${LISTENING_PORTS}"
        fi

        # 检查DNS解析
        if command -v nslookup > /dev/null 2>&1; then
            if nslookup google.com > /dev/null 2>&1; then
                log_success "    ✓ DNS 解析正常"
            else
                log_warning "    ⚠ DNS 解析可能有问题"
            fi
        fi
    }

    check_postgres
    check_redis
    check_health_endpoint
    check_network

    echo ""
}

# =============================================================================
# 可执行文件检查
# =============================================================================
check_executable() {
    log_info "检查可执行文件..."
    echo ""

    if [ ! -f "./hjtpx" ]; then
        log_error "错误: hjtpx 可执行文件未找到！"
        log_info "请先运行编译命令："
        echo ""
        echo "  make build"
        echo "  或者"
        echo "  go build -o hjtpx ./backend/cmd/api/main.go"
        echo ""
        exit 1
    fi

    # 检查文件权限
    if [ ! -x "./hjtpx" ]; then
        log_warning "hjtpx 文件没有执行权限，正在添加..."
        chmod +x ./hjtpx
    fi

    # 显示文件信息
    FILE_SIZE=$(du -h ./hjtpx | cut -f1)
    FILE_HASH=$(md5sum ./hjtpx 2>/dev/null | cut -d' ' -f1 || sha256sum ./hjtpx | cut -d' ' -f1)

    log_success "可执行文件: ./hjtpx"
    log_info "  大小: ${FILE_SIZE}"
    log_info "  哈希: ${FILE_HASH}"
    echo ""
}

# =============================================================================
# 启动应用
# =============================================================================
start_application() {
    log_info "启动后端服务..."
    echo ""

    # 设置环境变量
    export GIN_MODE=${GIN_MODE:-release}

    # 显示启动信息
    log_info "Gin 模式: ${GIN_MODE}"
    log_info "服务器端口: ${SERVER_PORT:-8080}"
    log_info "日志级别: ${LOG_LEVEL:-info}"
    log_info "日志格式: ${LOG_FORMAT:-json}"
    echo ""

    # 启动前钩子（如果存在）
    if [ -f "./scripts/pre-start.sh" ]; then
        log_info "执行预启动脚本..."
        bash ./scripts/pre-start.sh
        echo ""
    fi

    # 启动应用
    log_success "正在启动 HJTPX 服务..."
    echo ""

    # 使用exec确保进程信号正确传递
    exec ./hjtpx
}

# =============================================================================
# 信号处理
# =============================================================================
trap_handler() {
    echo ""
    log_warning "收到终止信号，正在优雅关闭..."
    exit 0
}

trap trap_handler SIGINT SIGTERM

# =============================================================================
# 主流程
# =============================================================================
main() {
    show_banner
    check_environment
    check_configuration
    check_service_dependencies
    check_executable
    start_application
}

main "$@"
