#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

APP_URL="${APP_URL:-http://localhost:8080}"
APP_PORT="${APP_PORT:-8080}"
CHECK_TIMEOUT=5
MAX_RETRIES=3

log_info() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] INFO: $1"
}

log_success() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] SUCCESS: $1"
}

log_error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $1" >&2
}

log_warning() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] WARNING: $1"
}

check_prerequisites() {
    echo ""
    echo "### 1. 前置条件检查"
    echo ""
    
    local passed=0
    local total=5
    
    if command -v docker &> /dev/null; then
        log_success "✓ Docker 已安装: $(docker --version)"
        passed=$((passed + 1))
    else
        log_error "✗ Docker 未安装"
    fi
    
    if command -v docker-compose &> /dev/null || docker compose version &> /dev/null; then
        log_success "✓ Docker Compose 已安装"
        passed=$((passed + 1))
    else
        log_error "✗ Docker Compose 未安装"
    fi
    
    if [ -f ".env" ]; then
        log_success "✓ .env 配置文件存在"
        passed=$((passed + 1))
    else
        log_error "✗ .env 配置文件不存在"
    fi
    
    if [ -f "docker-compose.yml" ]; then
        log_success "✓ docker-compose.yml 存在"
        passed=$((passed + 1))
    else
        log_error "✗ docker-compose.yml 不存在"
    fi
    
    if [ -f "Dockerfile" ]; then
        log_success "✓ Dockerfile 存在"
        passed=$((passed + 1))
    else
        log_error "✗ Dockerfile 不存在"
    fi
    
    echo ""
    echo "前置条件检查: $passed/$total 通过"
    echo ""
    
    if [ $passed -ne $total ]; then
        log_error "前置条件检查未通过"
        return 1
    fi
    
    return 0
}

check_system_resources() {
    echo ""
    echo "### 2. 系统资源检查"
    echo ""
    
    local passed=0
    local total=4
    
    MEM_TOTAL=$(free -m | awk '/^Mem:/{print $2}')
    if [ "$MEM_TOTAL" -ge 2048 ]; then
        log_success "✓ 内存充足: ${MEM_TOTAL}MB"
        passed=$((passed + 1))
    else
        log_warning "⚠ 内存可能不足: ${MEM_TOTAL}MB (建议 ≥2048MB)"
        passed=$((passed + 1))
    fi
    
    DISK_AVAILABLE=$(df -h / | awk 'NR==2 {print $4}')
    DISK_PERCENT=$(df / | awk 'NR==2 {print $5}' | sed 's/%//')
    if [ "$DISK_PERCENT" -lt 80 ]; then
        log_success "✓ 磁盘空间充足: $DISK_AVAILABLE 可用"
        passed=$((passed + 1))
    else
        log_warning "⚠ 磁盘空间可能不足: $DISK_AVAILABLE 可用"
        passed=$((passed + 1))
    fi
    
    CPU_CORES=$(nproc)
    if [ "$CPU_CORES" -ge 2 ]; then
        log_success "✓ CPU核心数充足: $CPU_CORES 核心"
        passed=$((passed + 1))
    else
        log_warning "⚠ CPU核心数较少: $CPU_CORES 核心 (建议 ≥2)"
        passed=$((passed + 1))
    fi
    
    if docker info &> /dev/null; then
        log_success "✓ Docker 守护进程运行正常"
        passed=$((passed + 1))
    else
        log_error "✗ Docker 守护进程未运行"
    fi
    
    echo ""
    echo "系统资源检查: $passed/$total 通过"
    echo ""
    
    return 0
}

check_network_connectivity() {
    echo ""
    echo "### 3. 网络连接检查"
    echo ""
    
    local passed=0
    local total=3
    
    if ping -c 1 -W 2 8.8.8.8 &> /dev/null; then
        log_success "✓ 外网连接正常"
        passed=$((passed + 1))
    else
        log_warning "⚠ 外网连接可能受限"
        passed=$((passed + 1))
    fi
    
    if curl -sf --connect-timeout 3 https://registry-1.docker.io/v2/ &> /dev/null; then
        log_success "✓ Docker Hub 可访问"
        passed=$((passed + 1))
    else
        log_warning "⚠ Docker Hub 可能无法访问"
        passed=$((passed + 1))
    fi
    
    if nslookup postgres &> /dev/null || [ "$POSTGRES_HOST" == "localhost" ] || [ "$POSTGRES_HOST" == "127.0.0.1" ]; then
        log_success "✓ PostgreSQL 主机可解析"
        passed=$((passed + 1))
    else
        log_warning "⚠ PostgreSQL 主机解析可能有问题"
        passed=$((passed + 1))
    fi
    
    echo ""
    echo "网络连接检查: $passed/$total 通过"
    echo ""
    
    return 0
}

check_environment_variables() {
    echo ""
    echo "### 4. 环境变量检查"
    echo ""
    
    source .env 2>/dev/null || true
    
    local passed=0
    local total=8
    
    local required_vars=(
        "POSTGRES_HOST"
        "POSTGRES_PORT"
        "POSTGRES_USER"
        "POSTGRES_PASSWORD"
        "POSTGRES_DB"
        "REDIS_HOST"
        "REDIS_PORT"
        "JWT_SECRET"
    )
    
    for var in "${required_vars[@]}"; do
        value="${!var}"
        if [ -n "$value" ]; then
            log_success "✓ $var 已设置"
            passed=$((passed + 1))
        else
            log_error "✗ $var 未设置"
        fi
    done
    
    if [ ${#JWT_SECRET} -ge 32 ]; then
        log_success "✓ JWT_SECRET 长度符合要求 (≥32字符)"
    else
        log_warning "⚠ JWT_SECRET 长度不足 (建议 ≥32字符)"
    fi
    
    echo ""
    echo "环境变量检查: $passed/$total 通过"
    echo ""
    
    if [ $passed -ne $total ]; then
        log_error "环境变量检查未通过"
        return 1
    fi
    
    return 0
}

check_docker_images() {
    echo ""
    echo "### 5. Docker 镜像检查"
    echo ""
    
    local passed=0
    local total=3
    
    REQUIRED_IMAGES=(
        "postgres:14-alpine"
        "redis:7-alpine"
    )
    
    for image in "${REQUIRED_IMAGES[@]}"; do
        if docker images | grep -q "^${image} "; then
            log_success "✓ 镜像已存在: $image"
            passed=$((passed + 1))
        else
            log_info "镜像需要下载: $image"
            passed=$((passed + 1))
        fi
    done
    
    if docker images | grep -q "^hjtpx/hjtpx"; then
        log_success "✓ 应用镜像已构建"
    else
        log_info "应用镜像需要构建"
    fi
    passed=$((passed + 1))
    
    echo ""
    echo "Docker 镜像检查: $passed/$total 通过"
    echo ""
    
    return 0
}

check_file_permissions() {
    echo ""
    echo "### 6. 文件权限检查"
    echo ""
    
    local passed=0
    local total=4
    
    if [ -r "." ]; then
        log_success "✓ 当前目录可读"
        passed=$((passed + 1))
    else
        log_error "✗ 当前目录不可读"
    fi
    
    if [ -w "." ]; then
        log_success "✓ 当前目录可写"
        passed=$((passed + 1))
    else
        log_error "✗ 当前目录不可写"
    fi
    
    if [ -x "./scripts" ]; then
        log_success "✓ scripts 目录可执行"
        passed=$((passed + 1))
    else
        log_warning "⚠ scripts 目录缺少执行权限"
        passed=$((passed + 1))
    fi
    
    if [ -d "./logs" ] || mkdir -p ./logs 2>/dev/null; then
        log_success "✓ logs 目录可访问"
        passed=$((passed + 1))
    else
        log_error "✗ logs 目录不可访问"
    fi
    
    echo ""
    echo "文件权限检查: $passed/$total 通过"
    echo ""
    
    return 0
}

check_directory_structure() {
    echo ""
    echo "### 7. 目录结构检查"
    echo ""
    
    local passed=0
    local total=5
    
    local required_dirs=(
        "backend"
        "scripts"
        "docker"
        "monitoring"
        "nginx"
    )
    
    for dir in "${required_dirs[@]}"; do
        if [ -d "$dir" ]; then
            log_success "✓ 目录存在: $dir"
            passed=$((passed + 1))
        else
            log_error "✗ 目录缺失: $dir"
        fi
    done
    
    echo ""
    echo "目录结构检查: $passed/$total 通过"
    echo ""
    
    return 0
}

check_configuration_files() {
    echo ""
    echo "### 8. 配置文件检查"
    echo ""
    
    local passed=0
    local total=5
    
    local config_files=(
        "config.yaml"
        "backend/config/config.yaml"
        "nginx/nginx.conf"
        "docker/health-check.sh"
        "docker/entrypoint.sh"
    )
    
    for file in "${config_files[@]}"; do
        if [ -f "$file" ]; then
            log_success "✓ 配置文件存在: $file"
            passed=$((passed + 1))
        else
            log_warning "⚠ 配置文件缺失: $file"
        fi
    done
    
    echo ""
    echo "配置文件检查: $passed/$total 通过"
    echo ""
    
    return 0
}

generate_summary() {
    echo ""
    echo "========================================"
    echo "       部署前检查清单 - 汇总报告"
    echo "========================================"
    echo ""
    echo "检查时间: $(date '+%Y-%m-%d %H:%M:%S')"
    echo ""
    echo "所有检查项目已完成。"
    echo ""
    echo "下一步操作:"
    echo "  1. 确保所有检查通过 (标记为 ✓)"
    echo "  2. 如有警告 (标记为 ⚠)，请评估风险"
    echo "  3. 如有错误 (标记为 ✗)，请修复后再继续"
    echo ""
    echo "开始部署:"
    echo "  ./scripts/deploy.sh"
    echo ""
    echo "或使用管理脚本:"
    echo "  ./scripts/manage.sh start"
    echo ""
}

main() {
    echo "========================================"
    echo "       HJTPX 部署前检查清单"
    echo "========================================"
    echo ""
    
    local exit_code=0
    
    check_prerequisites || exit_code=1
    check_system_resources || exit_code=1
    check_network_connectivity || exit_code=1
    check_environment_variables || exit_code=1
    check_docker_images || exit_code=1
    check_file_permissions || exit_code=1
    check_directory_structure || exit_code=1
    check_configuration_files || exit_code=1
    
    generate_summary
    
    if [ $exit_code -eq 0 ]; then
        log_success "所有关键检查通过，可以开始部署!"
    else
        log_warning "部分检查未通过，请在修复后继续"
    fi
    
    exit $exit_code
}

main "$@"
