#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

get_docker_compose_cmd() {
    if command -v docker-compose &> /dev/null; then
        echo "docker-compose"
    elif docker compose version &> /dev/null; then
        echo "docker compose"
    else
        echo "docker-compose"
    fi
}

DOCKER_COMPOSE=$(get_docker_compose_cmd)

APP_URL="${APP_URL:-http://localhost:8080}"
APP_PORT="${APP_PORT:-8080}"
POSTGRES_HOST="${POSTGRES_HOST:-postgres}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_USER="${POSTGRES_USER:-postgres}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-postgres}"
POSTGRES_DB="${POSTGRES_DB:-hjtpx_db}"
REDIS_HOST="${REDIS_HOST:-redis}"
REDIS_PORT="${REDIS_PORT:-6379}"
REDIS_PASSWORD="${REDIS_PASSWORD:-}"

check_backend() {
    local attempt=1
    local max_attempts=5

    while [ $attempt -le $max_attempts ]; do
        if curl -sf "$APP_URL/health" > /dev/null 2>&1; then
            log_success "后端服务正常 (尝试 $attempt/$max_attempts)"

            local health_response=$(curl -s "$APP_URL/health" 2>/dev/null)
            if [ -n "$health_response" ]; then
                log_info "健康检查响应: $health_response"
            fi

            return 0
        fi

        log_warning "后端服务响应异常 (尝试 $attempt/$max_attempts)"
        attempt=$((attempt + 1))

        if [ $attempt -le $max_attempts ]; then
            sleep 2
        fi
    done

    log_error "后端服务异常 ($max_attempts 次尝试后)"
    return 1
}

check_database() {
    if command -v psql > /dev/null 2>&1; then
        if PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT 1 as connected;" > /dev/null 2>&1; then
            log_success "数据库连接正常"
            return 0
        else
            log_warning "数据库连接失败，尝试 Docker 检查"

            if $DOCKER_COMPOSE exec -T postgres pg_isready -U postgres > /dev/null 2>&1; then
                log_success "PostgreSQL 容器运行正常"
                return 0
            else
                log_error "PostgreSQL 连接失败"
                return 1
            fi
        fi
    else
        log_info "psql 未安装，使用 Docker 检查"

        if $DOCKER_COMPOSE exec -T postgres pg_isready -U postgres > /dev/null 2>&1; then
            log_success "PostgreSQL 容器运行正常"
            return 0
        else
            log_error "PostgreSQL 不可用"
            return 1
        fi
    fi
}

check_redis() {
    if command -v redis-cli > /dev/null 2>&1; then
        if [ -n "$REDIS_PASSWORD" ]; then
            if redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" -a "$REDIS_PASSWORD" ping 2>/dev/null | grep -q PONG; then
                log_success "Redis 连接正常"
                return 0
            fi
        else
            if redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" ping 2>/dev/null | grep -q PONG; then
                log_success "Redis 连接正常"
                return 0
            fi
        fi

        log_warning "Redis 连接失败，尝试 Docker 检查"

        if $DOCKER_COMPOSE exec -T redis redis-cli ping 2>/dev/null | grep -q PONG; then
            log_success "Redis 容器运行正常"
            return 0
        else
            log_error "Redis 不可用"
            return 1
        fi
    else
        log_info "redis-cli 未安装，使用 Docker 检查"

        if $DOCKER_COMPOSE exec -T redis redis-cli ping 2>/dev/null | grep -q PONG; then
            log_success "Redis 容器运行正常"
            return 0
        else
            log_error "Redis 不可用"
            return 1
        fi
    fi
}

check_container_status() {
    log_info "检查容器状态..."

    local all_running=true

    for service in app postgres redis nginx; do
        if $DOCKER_COMPOSE ps "$service" 2>/dev/null | grep -q "Up"; then
            log_success "容器 $service: 运行中"
        elif $DOCKER_COMPOSE ps "$service" 2>/dev/null | grep -q "Up (unhealthy)"; then
            log_warning "容器 $service: 运行中但健康检查未通过"
        else
            log_error "容器 $service: 未运行"
            all_running=false
        fi
    done

    if [ "$all_running" = true ]; then
        return 0
    else
        return 1
    fi
}

check_api_endpoints() {
    log_info "检查 API 端点..."

    local endpoints=(
        "/api/v1/info"
        "/api/v1/captcha/types"
    )

    local all_ok=true

    for endpoint in "${endpoints[@]}"; do
        if curl -sf "${APP_URL}${endpoint}" > /dev/null 2>&1; then
            log_success "API 端点 ${endpoint}: 正常"
        else
            log_warning "API 端点 ${endpoint}: 不可用"
            all_ok=false
        fi
    done

    if [ "$all_ok" = true ]; then
        return 0
    else
        return 1
    fi
}

check_system_resources() {
    log_info "检查系统资源..."

    local memory_usage=$(free -m 2>/dev/null | awk '/Mem:/ {print int($3/$2*100)}' || echo "unknown")
    local disk_usage=$(df -h / 2>/dev/null | awk 'NR==2 {print $5}' | sed 's/%//' || echo "0")

    if [ "$memory_usage" != "unknown" ] && [ "$memory_usage" -lt 90 ]; then
        log_success "内存使用率: ${memory_usage}%"
    elif [ "$memory_usage" != "unknown" ]; then
        log_warning "内存使用率较高: ${memory_usage}%"
    fi

    if [ "$disk_usage" -lt 90 ]; then
        log_success "磁盘使用率: ${disk_usage}%"
    else
        log_warning "磁盘使用率较高: ${disk_usage}%"
    fi
}

main() {
    echo ""
    echo "========================================"
    echo "       HJTPX 健康检查"
    echo "========================================"
    echo ""

    total=0
    passed=0
    failed_checks=""

    total=$((total + 1))
    if check_backend; then
        passed=$((passed + 1))
    else
        failed_checks="${failed_checks}backend "
    fi

    total=$((total + 1))
    if check_container_status; then
        passed=$((passed + 1))
    else
        failed_checks="${failed_checks}containers "
    fi

    total=$((total + 1))
    if check_database; then
        passed=$((passed + 1))
    else
        failed_checks="${failed_checks}database "
    fi

    total=$((total + 1))
    if check_redis; then
        passed=$((passed + 1))
    else
        failed_checks="${failed_checks}redis "
    fi

    total=$((total + 1))
    if check_api_endpoints; then
        passed=$((passed + 1))
    else
        failed_checks="${failed_checks}endpoints "
    fi

    check_system_resources

    echo ""
    echo "========================================"
    echo "检查结果: $passed/$total 通过"
    echo "========================================"
    echo ""

    if [ $passed -eq $total ]; then
        log_success "所有健康检查通过 ✓"
        exit 0
    else
        log_warning "部分健康检查失败: $failed_checks"
        log_info "建议操作:"
        log_info "  1. 查看日志: $DOCKER_COMPOSE logs -f"
        log_info "  2. 重启服务: $DOCKER_COMPOSE restart"
        log_info "  3. 查看容器状态: $DOCKER_COMPOSE ps"
        exit 1
    fi
}

main "$@"
