#!/bin/bash
set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$DIR/.." && pwd)"
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

DEPLOY_MODE="${DEPLOY_MODE:-standard}"
PARALLEL_BUILDS="${PARALLEL_BUILDS:-4}"
ENABLE_TELEMETRY="${ENABLE_TELEMETRY:-false}"
DEPLOYMENT_TIMEOUT="${DEPLOYMENT_TIMEOUT:-300}"

check_prerequisites() {
    log_info "检查前置条件..."

    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装"
        exit 1
    fi
    log_success "Docker: $(docker --version | cut -d' ' -f3 | cut -d',' -f1)"

    if ! $DOCKER_COMPOSE version &> /dev/null; then
        log_error "Docker Compose 未安装"
        exit 1
    fi
    log_success "Docker Compose: 可用"

    local cpu_cores=$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)
    local mem_total=$(free -m 2>/dev/null | awk '/Mem:/ {print $2}' || echo 4096)

    log_info "系统资源: ${cpu_cores} CPU 核心, ${mem_total}MB 内存"

    if [ "$mem_total" -lt 2048 ]; then
        log_warning "内存低于推荐值 (4GB)，可能会影响性能"
    fi

    return 0
}

parse_deploy_mode() {
    case "$DEPLOY_MODE" in
        fast)
            log_info "快速部署模式"
            ;;
        standard)
            log_info "标准部署模式"
            ;;
        full)
            log_info "完整部署模式 (含测试)"
            ;;
        *)
            log_error "未知部署模式: $DEPLOY_MODE"
            exit 1
            ;;
    esac
}

validate_configuration() {
    log_info "验证配置..."

    if [ ! -f ".env" ]; then
        log_warning ".env 文件不存在，复制模板"
        cp .env.example .env
        log_error "请先配置 .env 文件后重新运行"
        exit 1
    fi

    local required_vars=("POSTGRES_PASSWORD" "JWT_SECRET")
    for var in "${required_vars[@]}"; do
        if ! grep -q "^${var}=" .env || grep -q "${var}=.*your-" .env 2>/dev/null; then
            log_error "必需变量未配置或使用默认值: $var"
            exit 1
        fi
    done

    log_success "配置验证通过"
}

pull_images() {
    if [ "$DEPLOY_MODE" = "fast" ]; then
        log_info "快速模式: 跳过镜像拉取"
        return 0
    fi

    log_info "拉取镜像..."

    if $DOCKER_COMPOSE pull 2>/dev/null; then
        log_success "镜像拉取完成"
    else
        log_warning "镜像拉取失败，将使用本地构建"
    fi
}

build_images() {
    log_info "构建镜像 (并行数: $PARALLEL_BUILDS)..."

    local start_time=$(date +%s)

    export DOCKER_BUILDKIT=1
    export COMPOSE_DOCKER_CLI_BUILD=1

    if $DOCKER_COMPOSE build --parallel --progress=plain 2>&1 | tee /tmp/hjtpx_build.log; then
        local build_time=$(($(date +%s) - start_time))
        log_success "镜像构建完成 (耗时: ${build_time}秒)"
    else
        log_error "镜像构建失败"
        log_info "查看构建日志: tail -100 /tmp/hjtpx_build.log"
        exit 1
    fi
}

deploy_services() {
    log_info "部署服务..."

    $DOCKER_COMPOSE down --remove-orphans 2>/dev/null || true

    if $DOCKER_COMPOSE up -d; then
        log_success "服务部署命令已执行"
    else
        log_error "服务部署失败"
        exit 1
    fi
}

wait_for_health() {
    log_info "等待服务就绪..."

    local app_port=${APP_PORT:-8080}
    local max_wait=${DEPLOYMENT_TIMEOUT}
    local counter=0

    while [ $counter -lt $max_wait ]; do
        if curl -sf "http://localhost:$app_port/health" > /dev/null 2>&1; then
            log_success "服务已就绪 (耗时: ${counter}秒)"
            return 0
        fi

        counter=$((counter + 3))
        sleep 3

        if [ $((counter % 30)) -eq 0 ]; then
            log_info "等待服务启动... ($counter/$max_wait 秒)"
        fi
    done

    log_error "服务启动超时"
    return 1
}

run_health_check() {
    log_info "执行健康检查..."

    if [ -f "./scripts/health-check.sh" ]; then
        if bash ./scripts/health-check.sh; then
            log_success "健康检查通过"
            return 0
        else
            log_warning "健康检查未完全通过"
            return 1
        fi
    fi

    return 0
}

run_integration_tests() {
    if [ "$DEPLOY_MODE" != "full" ]; then
        return 0
    fi

    log_info "运行集成测试..."

    if command -v curl &> /dev/null; then
        local tests_passed=0
        local tests_total=3

        if curl -sf "http://localhost:${APP_PORT:-8080}/health" > /dev/null; then
            log_success "Health API 测试通过"
            tests_passed=$((tests_passed + 1))
        else
            log_error "Health API 测试失败"
        fi

        if curl -sf "http://localhost:${APP_PORT:-8080}/api/v1/info" > /dev/null; then
            log_success "Info API 测试通过"
            tests_passed=$((tests_passed + 1))
        else
            log_warning "Info API 测试失败"
        fi

        if [ -f "docker-compose.yml" ]; then
            if $DOCKER_COMPOSE ps | grep -q "Up"; then
                log_success "容器状态测试通过"
                tests_passed=$((tests_passed + 1))
            fi
        fi

        log_info "测试结果: $tests_passed/$tests_total"
    fi
}

collect_telemetry() {
    if [ "$ENABLE_TELEMETRY" != "true" ]; then
        return 0
    fi

    log_info "收集部署遥测数据..."

    local deploy_info=$(cat << EOF
{
    "timestamp": "$(date -Iseconds)",
    "mode": "$DEPLOY_MODE",
    "docker_version": "$(docker --version | cut -d' ' -f3 | cut -d',' -f1)",
    "compose_version": "$($DOCKER_COMPOSE version --short 2>/dev/null || echo 'unknown')",
    "build_time": $(date +%s)
}
EOF
)

    log_info "部署信息: $deploy_info"
}

print_deployment_info() {
    local deploy_time=$(($(date +%s) - DEPLOY_START_TIME))

    echo ""
    echo "========================================"
    echo "       HJTPX 部署完成"
    echo "========================================"
    echo ""
    echo -e "${GREEN}部署信息:${NC}"
    echo "  - 部署模式: $DEPLOY_MODE"
    echo "  - 部署耗时: ${deploy_time}秒"
    echo "  - Docker: $(docker --version | cut -d' ' -f3 | cut -d',' -f1)"
    echo ""
    echo -e "${GREEN}服务地址:${NC}"
    echo "  - 应用API: http://localhost:${APP_PORT:-8080}"
    echo "  - 前端页面: http://localhost:${NGINX_PORT:-80}"
    echo "  - 健康检查: http://localhost:${APP_PORT:-8080}/health"
    echo "  - Prometheus: http://localhost:${PROMETHEUS_PORT:-9090}"
    echo "  - Grafana: http://localhost:${GRAFANA_PORT:-3000}"
    echo ""
    echo -e "${GREEN}管理命令:${NC}"
    echo "  查看状态: $DOCKER_COMPOSE ps"
    echo "  查看日志: $DOCKER_COMPOSE logs -f"
    echo "  健康检查: ./scripts/health-check.sh"
    echo "  更新服务: ./scripts/update.sh"
    echo "  回滚服务: ./scripts/rollback.sh"
    echo ""
    echo -e "${YELLOW}首次部署请立即修改默认密码！${NC}"
    echo ""
}

usage() {
    cat << EOF
HJTPX 自动化部署脚本 v3.0

用法: $0 [选项]

选项:
    --mode <模式>       部署模式: fast/standard/full (默认: standard)
    --parallel <数量>   并行构建数 (默认: 4)
    --timeout <秒>      部署超时时间 (默认: 300)
    --no-telemetry      禁用遥测数据收集
    --help              显示帮助

示例:
    $0                           # 标准部署
    $0 --mode fast               # 快速部署
    $0 --mode full               # 完整部署 (含测试)
    $0 --parallel 8 --timeout 600

EOF
}

main() {
    echo ""
    echo "========================================"
    echo "       HJTPX 自动化部署"
    echo "========================================"
    echo ""

    DEPLOY_START_TIME=$(date +%s)

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --mode)
                DEPLOY_MODE="$2"
                shift 2
                ;;
            --parallel)
                PARALLEL_BUILDS="$2"
                shift 2
                ;;
            --timeout)
                DEPLOYMENT_TIMEOUT="$2"
                shift 2
                ;;
            --no-telemetry)
                ENABLE_TELEMETRY="false"
                shift
                ;;
            --help|-h)
                usage
                exit 0
                ;;
            *)
                log_error "未知参数: $1"
                usage
                exit 1
                ;;
        esac
    done

    trap 'log_error "部署被中断"; exit 1' INT TERM

    parse_deploy_mode
    check_prerequisites
    validate_configuration
    pull_images
    build_images
    deploy_services

    if ! wait_for_health; then
        log_warning "服务启动超时，尝试继续..."
    fi

    run_health_check
    run_integration_tests
    collect_telemetry

    print_deployment_info

    log_info "部署完成!"

    exit 0
}

main "$@"
