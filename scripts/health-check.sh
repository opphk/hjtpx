#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

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
SLACK_WEBHOOK_URL="${SLACK_WEBHOOK_URL:-}"
EMAIL_RECIPIENT="${EMAIL_RECIPIENT:-admin@example.com}"
LOG_FILE="${LOG_FILE:-./logs/health-check.log}"
REPORT_DIR="${REPORT_DIR:-./health-reports}"

mkdir -p "$(dirname "$LOG_FILE")"
mkdir -p "$REPORT_DIR"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
LOG_PREFIX="[${TIMESTAMP}]"

log_message() {
    local level="$1"
    shift
    local message="$*"
    echo "$LOG_PREFIX [$level] $message" >> "$LOG_FILE"
    echo -e "${BLUE}$LOG_PREFIX${NC} [$level] $message"
}

log_info() {
    log_message "INFO" "$@"
}

log_success() {
    log_message "SUCCESS" "$@"
}

log_warning() {
    log_message "WARNING" "$@"
}

log_error() {
    log_message "ERROR" "$@"
}

send_alert() {
    local severity="$1"
    local title="$2"
    local message="$3"
    local details="${4:-}"

    log_warning "[ALERT] $severity: $title - $message"

    if [ -n "$SLACK_WEBHOOK_URL" ]; then
        curl -s -X POST "$SLACK_WEBHOOK_URL" \
            -H 'Content-Type: application/json' \
            -d "{
                \"text\": \"[$severity] $title\",
                \"attachments\": [{
                    \"color\": \"$(if [ \"$severity\" = \"CRITICAL\" ]; then echo \"#f44336\"; elif [ \"$severity\" = \"WARNING\" ]; then echo \"#ff9800\"; else echo \"#2196f3\"; fi)\",
                    \"fields\": [
                        {\"title\": \"Severity\", \"value\": \"$severity\", \"short\": true},
                        {\"title\": \"Message\", \"value\": \"$message\", \"short\": false}
                    ],
                    \"footer\": \"HJTPX Health Check\",
                    \"ts\": $(date +%s)
                }]
            }" > /dev/null 2>&1 || true
    fi

    if [ -n "$EMAIL_RECIPIENT" ]; then
        echo "Subject: [$severity] $title

Severity: $severity
Message: $message
Details: $details
Time: $TIMESTAMP
Host: $(hostname)

--
HJTPX Health Check System
" | sendmail -f "healthcheck@hjtpx.local" "$EMAIL_RECIPIENT" 2>/dev/null || true
    fi
}

check_http_endpoint() {
    local url="$1"
    local name="$2"
    local timeout="${3:-5}"

    log_info "检查 $name: $url"

    if response=$(curl -sf -m "$timeout" -w "\n%{http_code}" "$url" 2>/dev/null); then
        http_code=$(echo "$response" | tail -n1)
        body=$(echo "$response" | sed '$d')

        if [ "$http_code" = "200" ] || [ "$http_code" = "204" ]; then
            log_success "$name 正常 (HTTP $http_code)"
            return 0
        else
            log_warning "$name 返回异常状态码: $http_code"
            return 1
        fi
    else
        log_error "$name 连接失败"
        return 1
    fi
}

check_backend_liveness() {
    log_info "检查后端存活状态..."
    check_http_endpoint "$APP_URL/health" "后端存活检查"
}

check_backend_readiness() {
    log_info "检查后端就绪状态..."
    check_http_endpoint "$APP_URL/health/ready" "后端就绪检查"
}

check_backend_detailed() {
    log_info "检查后端详细状态..."
    local response=$(curl -sf -m 5 "$APP_URL/health/detailed" 2>/dev/null)

    if [ -z "$response" ]; then
        log_error "后端详细状态检查失败"
        return 1
    fi

    log_success "后端详细状态: $response"
    return 0
}

check_database() {
    log_info "检查数据库连接..."

    if ! command -v psql &> /dev/null; then
        log_warning "psql 命令不存在，跳过数据库检查"
        return 0
    fi

    local start_time=$(date +%s%3N)
    local query_result

    if query_result=$(PGPASSWORD="$POSTGRES_PASSWORD" psql \
        -h "$POSTGRES_HOST" \
        -p "$POSTGRES_PORT" \
        -U "$POSTGRES_USER" \
        -d "$POSTGRES_DB" \
        -c "SELECT 1 AS result;" \
        -t \
        2>/dev/null); then

        local end_time=$(date +%s%3N)
        local duration=$((end_time - start_time))

        if echo "$query_result" | grep -q "1"; then
            log_success "数据库连接正常 (延迟: ${duration}ms)"
            return 0
        else
            log_error "数据库查询失败"
            return 1
        fi
    else
        log_error "数据库连接失败"
        return 1
    fi
}

check_database_detailed() {
    log_info "检查数据库详细状态..."

    if ! command -v psql &> /dev/null; then
        log_warning "psql 命令不存在，跳过详细数据库检查"
        return 0
    fi

    PGPASSWORD="$POSTGRES_PASSWORD" psql \
        -h "$POSTGRES_HOST" \
        -p "$POSTGRES_PORT" \
        -U "$POSTGRES_USER" \
        -d "$POSTGRES_DB" \
        -c "SELECT pg_database_size('$POSTGRES_DB') AS size;" \
        -t \
        2>/dev/null || true

    PGPASSWORD="$POSTGRES_PASSWORD" psql \
        -h "$POSTGRES_HOST" \
        -p "$POSTGRES_PORT" \
        -U "$POSTGRES_USER" \
        -d "$POSTGRES_DB" \
        -c "SELECT count(*) FROM pg_stat_activity WHERE state = 'active';" \
        -t \
        2>/dev/null || true

    return 0
}

check_redis() {
    log_info "检查Redis连接..."

    if ! command -v redis-cli &> /dev/null; then
        log_warning "redis-cli 命令不存在，跳过Redis检查"
        return 0
    fi

    local redis_cmd="redis-cli -h $REDIS_HOST -p $REDIS_PORT"
    [ -n "$REDIS_PASSWORD" ] && redis_cmd="$redis_cmd -a $REDIS_PASSWORD"

    if $redis_cmd ping > /dev/null 2>&1; then
        log_success "Redis连接正常"
        return 0
    else
        log_error "Redis连接失败"
        return 1
    fi
}

check_redis_detailed() {
    log_info "检查Redis详细状态..."

    if ! command -v redis-cli &> /dev/null; then
        log_warning "redis-cli 命令不存在，跳过详细Redis检查"
        return 0
    fi

    local redis_cmd="redis-cli -h $REDIS_HOST -p $REDIS_PORT"
    [ -n "$REDIS_PASSWORD" ] && redis_cmd="$redis_cmd -a $REDIS_PASSWORD"

    echo "=== Redis Info ==="
    $redis_cmd info 2>/dev/null | grep -E "redis_version|used_memory_human|connected_clients|maxmemory" || true

    echo ""
    echo "=== Redis Stats ==="
    $redis_cmd info stats 2>/dev/null | grep -E "instantaneous_ops_per_sec|total_connections_received|keyspace_hits|keyspace_misses" || true

    return 0
}

check_disk_space() {
    log_info "检查磁盘空间..."

    local threshold=85
    local usage=$(df -h . | awk 'NR==2 {print $5}' | sed 's/%//')

    if [ "$usage" -gt "$threshold" ]; then
        log_warning "磁盘使用率较高: ${usage}%"
        return 1
    else
        log_success "磁盘空间充足: ${usage}%"
        return 0
    fi
}

check_memory() {
    log_info "检查内存使用..."

    if command -v free &> /dev/null; then
        local usage=$(free | awk '/Mem:/ {printf "%.0f", $3/$2 * 100}')
        log_info "内存使用率: ${usage}%"

        if [ "$usage" -gt 90 ]; then
            log_warning "内存使用率过高: ${usage}%"
            return 1
        else
            log_success "内存使用正常"
            return 0
        fi
    else
        log_warning "free 命令不存在，跳过内存检查"
        return 0
    fi
}

check_cpu() {
    log_info "检查CPU使用..."

    if command -v uptime &> /dev/null; then
        local load=$(uptime | awk -F'load average:' '{print $2}' | awk '{print $1}' | sed 's/,//')
        log_info "系统负载: $load"

        local load_value=$(echo "$load" | awk -F. '{print $1}')
        local cpu_count=$(nproc 2>/dev/null || echo 4)

        if [ "$load_value" -gt "$((cpu_count * 2))" ]; then
            log_warning "系统负载较高: $load"
            return 1
        else
            log_success "系统负载正常"
            return 0
        fi
    else
        log_warning "uptime 命令不存在，跳过CPU检查"
        return 0
    fi
}

check_process() {
    local process_name="$1"
    log_info "检查进程: $process_name"

    if pgrep -x "$process_name" > /dev/null; then
        log_success "进程 $process_name 运行中"
        return 0
    else
        log_error "进程 $process_name 未运行"
        return 1
    fi
}

perform_health_check() {
    local check_level="${1:-basic}"
    local report_file="$REPORT_DIR/health-check-$(date +%Y%m%d_%H%M%S).json"

    log_info "======================================"
    log_info "开始健康检查 (级别: $check_level)"
    log_info "======================================"

    local total_checks=0
    local passed_checks=0
    local failed_checks=0
    local check_results=()

    log_info ""
    log_info "=== 第1层: 基础检查 ==="
    total_checks=$((total_checks + 1))
    if check_backend_liveness; then
        passed_checks=$((passed_checks + 1))
        check_results+=("{\"check\": \"backend_liveness\", \"status\": \"passed\"}")
    else
        failed_checks=$((failed_checks + 1))
        check_results+=("{\"check\": \"backend_liveness\", \"status\": \"failed\"}")
        send_alert "CRITICAL" "后端存活检查失败" "后端服务未响应健康检查"
    fi

    log_info ""
    log_info "=== 第2层: 就绪检查 ==="
    total_checks=$((total_checks + 1))
    if check_backend_readiness; then
        passed_checks=$((passed_checks + 1))
        check_results+=("{\"check\": \"backend_readiness\", \"status\": \"passed\"}")
    else
        failed_checks=$((failed_checks + 1))
        check_results+=("{\"check\": \"backend_readiness\", \"status\": \"failed\"}")
        send_alert "WARNING" "后端就绪检查失败" "后端服务未就绪"
    fi

    log_info ""
    log_info "=== 第3层: 依赖服务检查 ==="

    total_checks=$((total_checks + 1))
    if check_database; then
        passed_checks=$((passed_checks + 1))
        check_results+=("{\"check\": \"database\", \"status\": \"passed\"}")
    else
        failed_checks=$((failed_checks + 1))
        check_results+=("{\"check\": \"database\", \"status\": \"failed\"}")
        send_alert "CRITICAL" "数据库连接失败" "无法连接到PostgreSQL"
    fi

    total_checks=$((total_checks + 1))
    if check_redis; then
        passed_checks=$((passed_checks + 1))
        check_results+=("{\"check\": \"redis\", \"status\": \"passed\"}")
    else
        failed_checks=$((failed_checks + 1))
        check_results+=("{\"check\": \"redis\", \"status\": \"failed\"}")
        send_alert "CRITICAL" "Redis连接失败" "无法连接到Redis"
    fi

    if [ "$check_level" = "detailed" ] || [ "$check_level" = "full" ]; then
        log_info ""
        log_info "=== 第4层: 详细检查 ==="

        total_checks=$((total_checks + 1))
        if check_backend_detailed; then
            passed_checks=$((passed_checks + 1))
            check_results+=("{\"check\": \"backend_detailed\", \"status\": \"passed\"}")
        else
            failed_checks=$((failed_checks + 1))
            check_results+=("{\"check\": \"backend_detailed\", \"status\": \"failed\"}")
        fi

        total_checks=$((total_checks + 1))
        if check_database_detailed; then
            passed_checks=$((passed_checks + 1))
            check_results+=("{\"check\": \"database_detailed\", \"status\": \"passed\"}")
        else
            failed_checks=$((failed_checks + 1))
            check_results+=("{\"check\": \"database_detailed\", \"status\": \"failed\"}")
        fi

        total_checks=$((total_checks + 1))
        if check_redis_detailed; then
            passed_checks=$((passed_checks + 1))
            check_results+=("{\"check\": \"redis_detailed\", \"status\": \"passed\"}")
        else
            failed_checks=$((failed_checks + 1))
            check_results+=("{\"check\": \"redis_detailed\", \"status\": \"failed\"}")
        fi
    fi

    if [ "$check_level" = "full" ]; then
        log_info ""
        log_info "=== 第5层: 系统资源检查 ==="

        total_checks=$((total_checks + 1))
        if check_disk_space; then
            passed_checks=$((passed_checks + 1))
            check_results+=("{\"check\": \"disk_space\", \"status\": \"passed\"}")
        else
            failed_checks=$((failed_checks + 1))
            check_results+=("{\"check\": \"disk_space\", \"status\": \"failed\"}")
            send_alert "WARNING" "磁盘空间不足" "磁盘使用率超过85%"
        fi

        total_checks=$((total_checks + 1))
        if check_memory; then
            passed_checks=$((passed_checks + 1))
            check_results+=("{\"check\": \"memory\", \"status\": \"passed\"}")
        else
            failed_checks=$((failed_checks + 1))
            check_results+=("{\"check\": \"memory\", \"status\": \"failed\"}")
            send_alert "WARNING" "内存使用率过高" "内存使用率超过90%"
        fi

        total_checks=$((total_checks + 1))
        if check_cpu; then
            passed_checks=$((passed_checks + 1))
            check_results+=("{\"check\": \"cpu\", \"status\": \"passed\"}")
        else
            failed_checks=$((failed_checks + 1))
            check_results+=("{\"check\": \"cpu\", \"status\": \"failed\"}")
            send_alert "WARNING" "CPU负载过高" "系统负载异常"
        fi
    fi

    log_info ""
    log_info "======================================"
    log_info "健康检查完成"
    log_info "总计: $total_checks | 通过: $passed_checks | 失败: $failed_checks"
    log_info "======================================"

    {
        echo "{"
        echo "  \"timestamp\": \"$TIMESTAMP\","
        echo "  \"check_level\": \"$check_level\","
        echo "  \"total_checks\": $total_checks,"
        echo "  \"passed_checks\": $passed_checks,"
        echo "  \"failed_checks\": $failed_checks,"
        echo "  \"overall_status\": \"$(if [ $failed_checks -eq 0 ]; then echo 'healthy'; else echo 'unhealthy'; fi)\","
        echo "  \"results\": ["
        local first=true
        for result in "${check_results[@]}"; do
            if [ "$first" = true ]; then
                first=false
            else
                echo ","
            fi
            echo -n "    $result"
        done
        echo ""
        echo "  ]"
        echo "}"
    } > "$report_file"

    log_info "报告已保存到: $report_file"

    if [ $failed_checks -gt 0 ]; then
        return 1
    else
        return 0
    fi
}

auto_recovery() {
    log_info "尝试自动恢复..."

    if ! check_backend_liveness; then
        log_warning "尝试重启后端服务..."

        if command -v docker-compose &> /dev/null; then
            docker-compose restart app 2>/dev/null || docker compose restart app 2>/dev/null
            sleep 10
            if check_backend_liveness; then
                log_success "后端服务重启成功"
                send_alert "INFO" "服务自动恢复" "后端服务已自动重启并恢复"
                return 0
            fi
        elif command -v systemctl &> /dev/null; then
            sudo systemctl restart hjtpx 2>/dev/null || true
            sleep 10
            if check_backend_liveness; then
                log_success "后端服务重启成功"
                send_alert "INFO" "服务自动恢复" "后端服务已自动重启并恢复"
                return 0
            fi
        fi

        log_error "自动恢复失败"
        send_alert "CRITICAL" "自动恢复失败" "无法自动恢复服务，需要人工干预"
        return 1
    fi

    if ! check_database; then
        log_warning "数据库连接异常，尝试重连..."

        if command -v docker-compose &> /dev/null; then
            docker-compose restart postgres 2>/dev/null || docker compose restart postgres 2>/dev/null
            sleep 15
            if check_database; then
                log_success "数据库重连成功"
                return 0
            fi
        fi

        log_error "数据库恢复失败"
        send_alert "CRITICAL" "数据库恢复失败" "无法恢复数据库连接"
        return 1
    fi

    if ! check_redis; then
        log_warning "Redis连接异常，尝试重连..."

        if command -v docker-compose &> /dev/null; then
            docker-compose restart redis 2>/dev/null || docker compose restart redis 2>/dev/null
            sleep 10
            if check_redis; then
                log_success "Redis重连成功"
                return 0
            fi
        fi

        log_error "Redis恢复失败"
        send_alert "CRITICAL" "Redis恢复失败" "无法恢复Redis连接"
        return 1
    fi

    return 0
}

continuous_monitoring() {
    local interval="${1:-60}"
    log_info "开始持续监控 (间隔: ${interval}s, Ctrl+C 退出)"

    while true; do
        if ! perform_health_check "basic"; then
            log_warning "健康检查失败，尝试自动恢复..."
            auto_recovery
        fi

        sleep "$interval"
    done
}

show_help() {
    cat << EOF
HJTPX 健康检查工具

用法: $0 [选项] [命令]

选项:
    -u, --url URL           应用URL (默认: http://localhost:8080)
    -h, --help              显示帮助信息

命令:
    basic                   执行基础健康检查
    detailed                执行详细健康检查
    full                    执行完整健康检查（含系统资源）
    monitor                 持续监控模式
    recover                 尝试自动恢复
    help                    显示帮助信息

示例:
    $0 basic
    $0 detailed
    $0 full
    $0 monitor --interval 60

EOF
}

main() {
    local command="${1:-help}"
    shift || true

    case "$command" in
        basic)
            perform_health_check "basic"
            ;;
        detailed)
            perform_health_check "detailed"
            ;;
        full)
            perform_health_check "full"
            ;;
        monitor)
            local interval="${1:-60}"
            continuous_monitoring "$interval"
            ;;
        recover)
            auto_recovery
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            log_error "未知命令: $command"
            show_help
            exit 1
            ;;
    esac
}

main "$@"
