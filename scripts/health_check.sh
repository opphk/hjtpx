#!/bin/bash

set -e

PROJECT_NAME="hjtpx"
HEALTH_ENDPOINT="http://localhost:8080/health"
METRICS_ENDPOINT="http://localhost:8080/metrics"
TIMEOUT=5

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_app_health() {
    log_info "检查应用健康状态..."

    HTTP_CODE=$(curl -sf -o /dev/null -w "%{http_code}" --max-time ${TIMEOUT} ${HEALTH_ENDPOINT})

    if [ "$HTTP_CODE" = "200" ]; then
        log_info "应用健康检查通过 (HTTP ${HTTP_CODE})"
        return 0
    else
        log_error "应用健康检查失败 (HTTP ${HTTP_CODE})"
        return 1
    fi
}

check_database_connection() {
    log_info "检查数据库连接..."

    if docker exec hjtpx-postgres pg_isready -U postgres -d verification > /dev/null 2>&1; then
        log_info "数据库连接正常"
        return 0
    else
        log_error "数据库连接失败"
        return 1
    fi
}

check_redis_connection() {
    log_info "检查 Redis 连接..."

    if docker exec hjtpx-redis redis-cli ping > /dev/null 2>&1; then
        log_info "Redis 连接正常"
        return 0
    else
        log_error "Redis 连接失败"
        return 1
    fi
}

check_docker_containers() {
    log_info "检查 Docker 容器状态..."

    CONTAINERS=$(docker-compose ps --format json 2>/dev/null | jq -r 'select(.Service != null) | .Service + ": " + .State' 2>/dev/null || docker-compose ps 2>/dev/null)

    if echo "${CONTAINERS}" | grep -q "Up"; then
        log_info "所有容器运行正常"
        return 0
    else
        log_error "部分容器未运行"
        echo "${CONTAINERS}"
        return 1
    fi
}

check_disk_space() {
    log_info "检查磁盘空间..."

    AVAILABLE=$(df -h / | awk 'NR==2 {print $5}' | sed 's/%//')

    if [ "$AVAILABLE" -lt 90 ]; then
        log_info "磁盘空间充足 (${AVAILABLE}% 已使用)"
        return 0
    else
        log_warn "磁盘空间不足 (${AVAILABLE}% 已使用)"
        return 1
    fi
}

check_memory_usage() {
    log_info "检查内存使用..."

    if command -v free &> /dev/null; then
        MEMORY=$(free | awk 'NR==2{printf "%.0f", $3/$2 * 100}')
        log_info "内存使用率: ${MEMORY}%"

        if [ "$MEMORY" -lt 90 ]; then
            return 0
        else
            log_warn "内存使用率较高"
            return 1
        fi
    fi
    return 0
}

check_api_response_time() {
    log_info "检查 API 响应时间..."

    RESPONSE_TIME=$(curl -sf -o /dev/null -w "%{time_total}" --max-time ${TIMEOUT} ${HEALTH_ENDPOINT})

    if [ -n "$RESPONSE_TIME" ]; then
        RESPONSE_MS=$(echo "$RESPONSE_TIME * 1000" | bc | cut -d'.' -f1)
        log_info "API 响应时间: ${RESPONSE_MS}ms"

        if [ "$RESPONSE_MS" -lt 1000 ]; then
            return 0
        else
            log_warn "API 响应时间过长"
            return 1
        fi
    else
        log_error "无法获取 API 响应时间"
        return 1
    fi
}

check_logs_for_errors() {
    log_info "检查错误日志..."

    ERROR_COUNT=$(docker-compose logs --tail=100 app 2>/dev/null | grep -c "ERROR" || echo "0")

    if [ "$ERROR_COUNT" -gt 0 ]; then
        log_warn "发现 ${ERROR_COUNT} 条错误日志"
        docker-compose logs --tail=20 app 2>/dev/null | grep "ERROR" | tail -5
        return 1
    else
        log_info "没有发现错误日志"
        return 0
    fi
}

check_certificates() {
    if [ -d "/etc/nginx/ssl" ]; then
        log_info "检查 SSL 证书..."

        CERT_FILE="/etc/nginx/ssl/server.crt"
        if [ -f "$CERT_FILE" ]; then
            EXPIRY=$(openssl x509 -in "$CERT_FILE" -noout -enddate 2>/dev/null | cut -d= -f2)

            if [ -n "$EXPIRY" ]; then
                log_info "证书到期时间: ${EXPIRY}"
                return 0
            fi
        fi
    fi
    return 0
}

generate_report() {
    log_info "生成健康检查报告..."

    REPORT_FILE="/tmp/${PROJECT_NAME}_health_report_$(date +%Y%m%d_%H%M%S).txt"

    {
        echo "========================================="
        echo "${PROJECT_NAME} 健康检查报告"
        echo "检查时间: $(date '+%Y-%m-%d %H:%M:%S')"
        echo "========================================="
        echo ""
        echo "1. 应用健康状态:"
        curl -sf ${HEALTH_ENDPOINT} 2>/dev/null || echo "无法获取"
        echo ""
        echo "2. Docker 容器状态:"
        docker-compose ps 2>/dev/null
        echo ""
        echo "3. 数据库连接:"
        docker exec hjtpx-postgres pg_isready -U postgres -d verification 2>/dev/null || echo "失败"
        echo ""
        echo "4. Redis 连接:"
        docker exec hjtpx-redis redis-cli ping 2>/dev/null || echo "失败"
        echo ""
        echo "5. 系统资源:"
        df -h /
        echo ""
        free -h
        echo ""
    } > "${REPORT_FILE}"

    log_info "报告已保存: ${REPORT_FILE}"
}

run_all_checks() {
    log_info "========================================="
    log_info "开始 ${PROJECT_NAME} 健康检查"
    log_info "========================================="
    echo ""

    FAILED_CHECKS=0

    check_app_health || ((FAILED_CHECKS++))
    echo ""

    check_docker_containers || ((FAILED_CHECKS++))
    echo ""

    check_database_connection || ((FAILED_CHECKS++))
    echo ""

    check_redis_connection || ((FAILED_CHECKS++))
    echo ""

    check_disk_space || ((FAILED_CHECKS++))
    echo ""

    check_memory_usage || ((FAILED_CHECKS++))
    echo ""

    check_api_response_time || ((FAILED_CHECKS++))
    echo ""

    check_logs_for_errors || ((FAILED_CHECKS++))
    echo ""

    check_certificates || ((FAILED_CHECKS++))
    echo ""

    log_info "========================================="
    log_info "健康检查完成"
    log_info "========================================="

    if [ $FAILED_CHECKS -eq 0 ]; then
        log_info "所有检查通过！"
        return 0
    else
        log_warn "${FAILED_CHECKS} 项检查失败"
        return 1
    fi
}

main() {
    case "$1" in
        --app|-a)
            check_app_health
            ;;
        --db|-d)
            check_database_connection
            ;;
        --redis|-r)
            check_redis_connection
            ;;
        --containers|-c)
            check_docker_containers
            ;;
        --disk|-k)
            check_disk_space
            ;;
        --memory|-m)
            check_memory_usage
            ;;
        --response|-t)
            check_api_response_time
            ;;
        --logs|-l)
            check_logs_for_errors
            ;;
        --report|-R)
            generate_report
            ;;
        --all|-A|"")
            run_all_checks
            ;;
        --help|-h)
            echo "用法: $0 [选项]"
            echo ""
            echo "选项:"
            echo "  -a, --app        检查应用健康状态"
            echo "  -d, --db         检查数据库连接"
            echo "  -r, --redis      检查 Redis 连接"
            echo "  -c, --containers 检查 Docker 容器"
            echo "  -k, --disk       检查磁盘空间"
            echo "  -m, --memory     检查内存使用"
            echo "  -t, --response   检查 API 响应时间"
            echo "  -l, --logs       检查错误日志"
            echo "  -R, --report     生成健康报告"
            echo "  -A, --all        运行所有检查（默认）"
            echo "  -h, --help       显示帮助"
            ;;
        *)
            log_error "未知选项: $1"
            echo "使用 $0 --help 查看帮助"
            exit 1
            ;;
    esac
}

main "$@"
