#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

APP_URL="${APP_URL:-http://localhost:8080}"
APP_HOST="${APP_HOST:-localhost}"
POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_USER="${POSTGRES_USER:-postgres}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-postgres}"
POSTGRES_DB="${POSTGRES_DB:-hjtpx_db}"
REDIS_HOST="${REDIS_HOST:-localhost}"
REDIS_PORT="${REDIS_PORT:-6379}"
REDIS_PASSWORD="${REDIS_PASSWORD:-}"
NGINX_HOST="${NGINX_HOST:-localhost}"
NGINX_PORT="${NGINX_PORT:-80}"

VERBOSE="${VERBOSE:-false}"
TIMEOUT=5

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
}

check_backend() {
    log_info "检查后端服务..."
    
    local start_time=$(date +%s%3N)
    local response=$(curl -sf --connect-timeout $TIMEOUT --max-time $((TIMEOUT * 2)) "$APP_URL/health" 2>&1)
    local curl_exit=$?
    local end_time=$(date +%s%3N)
    local response_time=$((end_time - start_time))
    
    if [ $curl_exit -eq 0 ]; then
        if [ "$VERBOSE" = "true" ]; then
            log_success "后端服务正常 (响应时间: ${response_time}ms)"
            [ -n "$response" ] && log_info "响应内容: $response"
        else
            log_success "后端服务正常 (响应时间: ${response_time}ms)"
        fi
        
        if echo "$response" | grep -q '"status"'; then
            local status=$(echo "$response" | grep -o '"status"[[:space:]]*:[[:space:]]*"[^"]*"' | cut -d'"' -f4)
            if [ "$status" = "healthy" ] || [ "$status" = "ok" ]; then
                return 0
            fi
        fi
        
        return 0
    else
        log_error "后端服务异常 (HTTP: $curl_exit)"
        [ "$VERBOSE" = "true" ] && log_error "错误详情: $response"
        return 1
    fi
}

check_database() {
    log_info "检查数据库连接..."
    
    if ! command -v psql &> /dev/null; then
        log_warn "psql 命令不可用，跳过数据库检查"
        return 0
    fi
    
    local start_time=$(date +%s%3N)
    local result=$(PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT 1 AS check;" -t 2>&1)
    local psql_exit=$?
    local end_time=$(date +%s%3N)
    local response_time=$((end_time - start_time))
    
    if [ $psql_exit -eq 0 ]; then
        log_success "数据库连接正常 (响应时间: ${response_time}ms)"
        return 0
    else
        log_error "数据库连接失败"
        [ "$VERBOSE" = "true" ] && log_error "错误详情: $result"
        return 1
    fi
}

check_redis() {
    log_info "检查Redis连接..."
    
    if ! command -v redis-cli &> /dev/null; then
        log_warn "redis-cli 命令不可用，跳过Redis检查"
        return 0
    fi
    
    local start_time=$(date +%s%3N)
    local result
    if [ -n "$REDIS_PASSWORD" ]; then
        result=$(redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" -a "$REDIS_PASSWORD" --no-auth-warning ping 2>&1)
    else
        result=$(redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" ping 2>&1)
    fi
    local redis_exit=$?
    local end_time=$(date +%s%3N)
    local response_time=$((end_time - start_time))
    
    if [ "$result" = "PONG" ]; then
        log_success "Redis连接正常 (响应时间: ${response_time}ms)"
        return 0
    else
        log_error "Redis连接失败"
        [ "$VERBOSE" = "true" ] && log_error "错误详情: $result"
        return 1
    fi
}

check_nginx() {
    log_info "检查Nginx服务..."
    
    local start_time=$(date +%s%3N)
    local response=$(curl -sf --connect-timeout $TIMEOUT --max-time $((TIMEOUT * 2)) "http://${NGINX_HOST}:${NGINX_PORT}/" 2>&1)
    local curl_exit=$?
    local end_time=$(date +%s%3N)
    local response_time=$((end_time - start_time))
    
    if [ $curl_exit -eq 0 ]; then
        log_success "Nginx服务正常 (响应时间: ${response_time}ms)"
        return 0
    else
        log_error "Nginx服务异常 (HTTP: $curl_exit)"
        [ "$VERBOSE" = "true" ] && log_error "错误详情: $response"
        return 1
    fi
}

check_prometheus() {
    log_info "检查Prometheus服务..."
    
    local prometheus_url="${PROMETHEUS_URL:-http://localhost:9090}"
    local start_time=$(date +%s%3N)
    local response=$(curl -sf --connect-timeout $TIMEOUT --max-time $((TIMEOUT * 2)) "${prometheus_url}/-/ready" 2>&1)
    local curl_exit=$?
    local end_time=$(date +%s%3N)
    local response_time=$((end_time - start_time))
    
    if [ $curl_exit -eq 0 ]; then
        log_success "Prometheus服务正常 (响应时间: ${response_time}ms)"
        return 0
    else
        log_warn "Prometheus服务不可用 (HTTP: $curl_exit)"
        return 1
    fi
}

check_grafana() {
    log_info "检查Grafana服务..."
    
    local grafana_url="${GRAFANA_URL:-http://localhost:3000}"
    local start_time=$(date +%s%3N)
    local response=$(curl -sf --connect-timeout $TIMEOUT --max-time $((TIMEOUT * 2)) "${grafana_url}/api/health" 2>&1)
    local curl_exit=$?
    local end_time=$(date +%s%3N)
    local response_time=$((end_time - start_time))
    
    if [ $curl_exit -eq 0 ]; then
        log_success "Grafana服务正常 (响应时间: ${response_time}ms)"
        return 0
    else
        log_warn "Grafana服务不可用 (HTTP: $curl_exit)"
        return 1
    fi
}

check_docker_containers() {
    log_info "检查Docker容器状态..."
    
    if ! command -v docker &> /dev/null; then
        log_warn "Docker不可用，跳过容器检查"
        return 0
    fi
    
    local running_containers=$(docker ps --format "{{.Names}}" 2>/dev/null | wc -l)
    local total_containers=$(docker ps -a --format "{{.Names}}" 2>/dev/null | wc -l)
    
    if [ $running_containers -gt 0 ]; then
        log_success "Docker容器运行中: ${running_containers}/${total_containers}"
        
        if [ "$VERBOSE" = "true" ]; then
            echo ""
            docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null
        fi
        return 0
    else
        log_error "没有运行的容器"
        return 1
    fi
}

check_disk_space() {
    log_info "检查磁盘空间..."
    
    local threshold=90
    local usage=$(df -h / | awk 'NR==2 {print $5}' | tr -d '%')
    
    if [ "$usage" -lt "$threshold" ]; then
        log_success "磁盘空间充足 (使用率: ${usage}%)"
        return 0
    else
        log_error "磁盘空间不足 (使用率: ${usage}%)"
        df -h /
        return 1
    fi
}

check_memory() {
    log_info "检查内存使用..."
    
    local threshold=90
    local usage=$(free | awk '/Mem:/ {printf "%.0f", $3/$2 * 100}')
    
    if [ "$usage" -lt "$threshold" ]; then
        log_success "内存使用正常 (使用率: ${usage}%)"
        return 0
    else
        log_warn "内存使用较高 (使用率: ${usage}%)"
        free -h
        return 1
    fi
}

main() {
    echo ""
    echo "========================================"
    echo "       HJTPX 健康检查脚本"
    echo "========================================"
    echo ""
    
    local total=0
    local passed=0
    local failed=0
    local warnings=0
    
    local checks=(
        "check_backend:Backend"
        "check_database:Database"
        "check_redis:Redis"
        "check_nginx:Nginx"
        "check_prometheus:Prometheus"
        "check_grafana:Grafana"
        "check_docker_containers:Docker"
        "check_disk_space:Disk"
        "check_memory:Memory"
    )
    
    for check_info in "${checks[@]}"; do
        IFS=':' read -r check_func check_name <<< "$check_info"
        total=$((total + 1))
        
        if $check_func; then
            passed=$((passed + 1))
        else
            if [ "$?" -eq 0 ]; then
                warnings=$((warnings + 1))
            else
                failed=$((failed + 1))
            fi
        fi
        echo ""
    done
    
    echo "========================================"
    echo "       检查结果汇总"
    echo "========================================"
    printf "%-20s %s\n" "总计检查:" "$total"
    printf "%-20s %s\n" "通过:" "$passed"
    printf "%-20s %s\n" "失败:" "$failed"
    printf "%-20s %s\n" "警告:" "$warnings"
    echo "========================================"
    echo ""
    
    if [ $failed -gt 0 ]; then
        log_error "部分健康检查失败"
        exit 1
    elif [ $warnings -gt 0 ]; then
        log_warn "健康检查完成，但有警告"
        exit 0
    else
        log_success "所有健康检查通过"
        exit 0
    fi
}

main "$@"
