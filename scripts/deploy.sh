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

log_step() {
    echo -e "${BLUE}[STEP]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

check_prerequisites() {
    log_step "1. 检查前置条件..."

    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装"
        exit 1
    fi
    log_success "Docker 已安装: $(docker --version | cut -d' ' -f3 | cut -d',' -f1)"

    if command -v docker-compose &> /dev/null; then
        DOCKER_COMPOSE="docker-compose"
        log_success "Docker Compose 已安装: $(docker-compose --version | cut -d' ' -f5 | cut -d',' -f1)"
    elif docker compose version &> /dev/null; then
        DOCKER_COMPOSE="docker compose"
        log_success "Docker Compose (plugin) 已安装"
    else
        log_error "Docker Compose 未安装"
        exit 1
    fi

    if [ ! -f ".env" ]; then
        log_warning ".env 文件不存在，复制 .env.example 作为模板"
        cp .env.example .env
        log_warning "请编辑 .env 文件配置您的环境变量"
        log_warning "完成后重新运行此脚本"
        exit 1
    fi

    if ! grep -q "POSTGRES_PASSWORD" .env || grep -q "your-secure-password" .env 2>/dev/null; then
        log_warning "检测到默认密码配置，请修改 .env 中的密码"
    fi

    return 0
}

create_directories() {
    log_step "2. 创建必要目录..."

    mkdir -p backend/config
    mkdir -p nginx/ssl
    mkdir -p monitoring/prometheus/rules
    mkdir -p monitoring/grafana/provisioning/dashboards
    mkdir -p monitoring/grafana/provisioning/datasources
    mkdir -p data/postgres
    mkdir -p data/redis
    mkdir -p logs
    mkdir -p backups

    chmod -R 755 nginx/ssl 2>/dev/null || true

    log_success "目录创建完成"
}

generate_ssl_cert() {
    log_step "3. 生成SSL证书..."

    if [ ! -f "nginx/ssl/cert.pem" ] || [ ! -f "nginx/ssl/key.pem" ]; then
        DOMAIN=$(grep -E "^DOMAIN=" .env 2>/dev/null | cut -d'=' -f2 || echo "localhost")

        openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
            -keyout nginx/ssl/key.pem \
            -out nginx/ssl/cert.pem \
            -subj "/C=CN/ST=Beijing/L=Beijing/O=HJTPX/CN=${DOMAIN:-localhost}" \
            2>/dev/null

        log_success "SSL证书已生成"
    else
        log_success "SSL证书已存在，跳过生成"
    fi
}

pull_images() {
    log_step "4. 拉取最新镜像..."

    if $DOCKER_COMPOSE pull 2>/dev/null; then
        log_success "镜像拉取完成"
    else
        log_warning "镜像拉取失败或无远程镜像，将使用本地构建"
    fi
}

build_images() {
    log_step "5. 构建Docker镜像..."

    START_TIME=$(date +%s)

    if $DOCKER_COMPOSE build --no-cache 2>&1 | tee /tmp/hjtpx_build.log; then
        BUILD_TIME=$(($(date +%s) - START_TIME))
        log_success "镜像构建完成，耗时: ${BUILD_TIME}秒"
    else
        log_error "镜像构建失败"
        log_info "查看构建日志: tail -100 /tmp/hjtpx_build.log"
        exit 1
    fi
}

stop_old_containers() {
    log_step "6. 停止旧容器..."

    if $DOCKER_COMPOSE down --remove-orphans 2>/dev/null; then
        log_success "旧容器已停止"
    else
        log_warning "停止旧容器时出现警告，继续部署"
    fi
}

start_services() {
    log_step "7. 启动服务..."

    if $DOCKER_COMPOSE up -d; then
        log_success "服务启动命令已执行"
    else
        log_error "服务启动失败"
        exit 1
    fi
}

wait_for_services() {
    log_step "8. 等待服务就绪..."

    APP_PORT=${APP_PORT:-8080}
    MAX_WAIT=60
    COUNTER=0

    log_info "等待应用服务启动 (端口 $APP_PORT)..."

    while [ $COUNTER -lt $MAX_WAIT ]; do
        if curl -sf "http://localhost:$APP_PORT/health" > /dev/null 2>&1; then
            log_success "应用服务已就绪"
            return 0
        fi

        COUNTER=$((COUNTER + 2))
        sleep 2

        if [ $((COUNTER % 10)) -eq 0 ]; then
            log_info "仍在等待... ($COUNTER/$MAX_WAIT 秒)"
        fi
    done

    log_error "服务启动超时"
    log_info "查看应用日志: $DOCKER_COMPOSE logs app"
    return 1
}

check_services() {
    log_step "9. 检查服务状态..."

    local all_healthy=true

    for service in app postgres redis nginx; do
        if $DOCKER_COMPOSE ps $service 2>/dev/null | grep -q "Up"; then
            log_success "$service: 运行中"
        else
            log_error "$service: 未运行"
            all_healthy=false
        fi
    done

    if [ "$all_healthy" = true ]; then
        return 0
    else
        return 1
    fi
}

run_health_check() {
    log_step "10. 执行健康检查..."

    if [ -f "./scripts/health-check.sh" ]; then
        if bash ./scripts/health-check.sh; then
            log_success "健康检查通过"
            return 0
        else
            log_warning "健康检查未完全通过，请检查配置"
            return 1
        fi
    else
        log_warning "健康检查脚本不存在，跳过"
        return 0
    fi
}

print_summary() {
    echo ""
    echo "========================================"
    echo "       HJTPX 部署完成"
    echo "========================================"
    echo ""
    echo -e "${GREEN}服务地址:${NC}"
    echo "  - 应用API: http://localhost:${APP_PORT:-8080}"
    echo "  - 前端页面: http://localhost:${NGINX_PORT:-80}"
    echo "  - 健康检查: http://localhost:${APP_PORT:-8080}/health"
    echo "  - Prometheus: http://localhost:${PROMETHEUS_PORT:-9090}"
    echo "  - Grafana: http://localhost:${GRAFANA_PORT:-3000}"
    echo "  - Loki: http://localhost:${LOKI_PORT:-3100}"
    echo ""
    echo -e "${GREEN}默认管理员账号:${NC}"
    echo "  - 用户名: admin"
    echo "  - 密码: admin123"
    echo ""
    echo -e "${GREEN}常用命令:${NC}"
    echo "  查看日志: $DOCKER_COMPOSE logs -f"
    echo "  查看状态: $DOCKER_COMPOSE ps"
    echo "  停止服务: $DOCKER_COMPOSE down"
    echo "  重启服务: $DOCKER_COMPOSE restart"
    echo "  重新部署: ./scripts/deploy.sh"
    echo ""
    echo -e "${YELLOW}首次登录后请立即修改默认密码！${NC}"
    echo ""
}

cleanup() {
    log_info "清理临时文件..."
    rm -f /tmp/hjtpx_build.log 2>/dev/null || true
}

main() {
    echo ""
    echo "========================================"
    echo "       HJTPX 部署脚本 v2.0"
    echo "========================================"
    echo ""

    START_TIME=$(date +%s)

    trap cleanup EXIT

    check_prerequisites || exit 1
    create_directories
    generate_ssl_cert
    pull_images
    build_images
    stop_old_containers
    start_services
    wait_for_services || log_warning "服务可能未完全就绪"
    check_services || log_warning "部分服务可能未正常运行"
    run_health_check || log_warning "健康检查发现问题"

    TOTAL_TIME=$(($(date +%s) - START_TIME))

    print_summary

    log_info "总耗时: ${TOTAL_TIME}秒"

    exit 0
}

main "$@"
