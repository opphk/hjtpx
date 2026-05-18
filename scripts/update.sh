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

VERSION="${VERSION:-latest}"
AUTO_ROLLBACK="${AUTO_ROLLBACK:-true}"
SKIP_BACKUP="${SKIP_BACKUP:-false}"
SKIP_TESTS="${SKIP_TESTS:-false}"

update_code() {
    log_info "1. 获取最新代码..."

    if [ -d ".git" ]; then
        git fetch origin main
        CURRENT_BRANCH=$(git branch --show-current)
        log_info "当前分支: $CURRENT_BRANCH"

        if [ "$CURRENT_BRANCH" != "main" ]; then
            log_warning "当前不在 main 分支，切换到 main 分支"
            git checkout main
        fi

        LOCAL_COMMIT=$(git rev-parse HEAD)
        REMOTE_COMMIT=$(git rev-parse origin/main)

        if [ "$LOCAL_COMMIT" = "$REMOTE_COMMIT" ]; then
            log_info "代码已是最新版本，无需更新"
            return 0
        fi

        log_info "本地版本: $LOCAL_COMMIT"
        log_info "远程版本: $REMOTE_COMMIT"
    else
        log_warning "非 git 仓库，跳过代码更新"
    fi
}

create_backup() {
    if [ "$SKIP_BACKUP" = "true" ]; then
        log_info "跳过备份创建"
        return 0
    fi

    log_info "2. 创建当前版本备份..."

    if [ -f "./scripts/rollback.sh" ]; then
        if ./scripts/rollback.sh create; then
            log_success "备份创建成功"
        else
            log_warning "备份创建失败，但继续更新"
        fi
    else
        log_warning "回滚脚本不存在，跳过备份"
    fi
}

run_tests() {
    if [ "$SKIP_TESTS" = "true" ]; then
        log_info "跳过测试"
        return 0
    fi

    log_info "3. 运行测试..."

    if [ -f "Makefile" ]; then
        if make test 2>/dev/null; then
            log_success "测试通过"
        else
            log_warning "测试失败，请检查测试结果"
        fi
    else
        log_info "未找到 Makefile，跳过测试"
    fi
}

pull_latest_images() {
    log_info "4. 拉取最新镜像..."

    if [ "$VERSION" = "latest" ]; then
        log_info "拉取最新版本镜像..."
        if $DOCKER_COMPOSE pull; then
            log_success "镜像拉取完成"
        else
            log_warning "镜像拉取失败，继续使用本地构建"
        fi
    else
        log_info "拉取指定版本镜像: $VERSION"
        if $DOCKER_COMPOSE pull --platform linux/amd64 2>/dev/null; then
            log_success "镜像拉取完成"
        else
            log_warning "镜像拉取失败，继续使用本地构建"
        fi
    fi
}

build_images() {
    log_info "5. 构建Docker镜像..."

    START_TIME=$(date +%s)

    if $DOCKER_COMPOSE build --no-cache 2>&1 | tee /tmp/hjtpx_build.log; then
        BUILD_TIME=$(($(date +%s) - START_TIME))
        log_success "镜像构建完成，耗时: ${BUILD_TIME}秒"
    else
        log_error "镜像构建失败"
        if [ "$AUTO_ROLLBACK" = "true" ] && [ "$SKIP_BACKUP" != "true" ]; then
            log_warning "执行自动回滚..."
            ./scripts/rollback.sh quick 2>/dev/null || true
        fi
        exit 1
    fi
}

stop_services() {
    log_info "6. 停止旧服务..."

    $DOCKER_COMPOSE down --remove-orphans 2>/dev/null || true
    log_success "旧服务已停止"
}

start_services() {
    log_info "7. 启动新服务..."

    if $DOCKER_COMPOSE up -d; then
        log_success "服务启动命令已执行"
    else
        log_error "服务启动失败"
        if [ "$AUTO_ROLLBACK" = "true" ] && [ "$SKIP_BACKUP" != "true" ]; then
            log_warning "执行自动回滚..."
            ./scripts/rollback.sh quick 2>/dev/null || true
        fi
        exit 1
    fi
}

wait_for_services() {
    log_info "8. 等待服务就绪..."

    APP_PORT=${APP_PORT:-8080}
    MAX_WAIT=60
    COUNTER=0

    while [ $COUNTER -lt $MAX_WAIT ]; do
        if curl -sf "http://localhost:$APP_PORT/health" > /dev/null 2>&1; then
            log_success "服务已就绪"
            return 0
        fi

        COUNTER=$((COUNTER + 2))
        sleep 2

        if [ $((COUNTER % 10)) -eq 0 ]; then
            log_info "等待服务启动... ($COUNTER/$MAX_WAIT 秒)"
        fi
    done

    log_error "服务启动超时"
    return 1
}

verify_update() {
    log_info "9. 验证更新..."

    APP_PORT=${APP_PORT:-8080}

    if curl -sf "http://localhost:$APP_PORT/health" > /dev/null 2>&1; then
        log_success "健康检查通过"

        local version_info=$(curl -s "http://localhost:$APP_PORT/api/v1/info" 2>/dev/null || echo '{"version":"unknown"}')
        log_info "当前版本信息: $version_info"

        return 0
    else
        log_error "健康检查失败"
        return 1
    fi
}

print_summary() {
    echo ""
    echo "========================================"
    echo "       HJTPX 更新完成"
    echo "========================================"
    echo ""
    echo -e "${GREEN}更新信息:${NC}"
    echo "  - 更新版本: $VERSION"
    echo "  - 自动回滚: $AUTO_ROLLBACK"
    echo ""
    echo -e "${GREEN}服务地址:${NC}"
    echo "  - 应用API: http://localhost:${APP_PORT:-8080}"
    echo "  - 前端页面: http://localhost:${NGINX_PORT:-80}"
    echo ""
    echo -e "${GREEN}更新命令:${NC}"
    echo "  ./scripts/update.sh                    # 完整更新"
    echo "  ./scripts/update.sh --no-backup        # 跳过备份"
    echo "  ./scripts/update.sh --skip-tests       # 跳过测试"
    echo "  ./scripts/update.sh --version v11.0    # 指定版本"
    echo ""
}

main() {
    echo ""
    echo "========================================"
    echo "       HJTPX 更新脚本 v2.0"
    echo "========================================"
    echo ""

    while [[ $# -gt 0 ]]; do
        case $1 in
            --no-backup)
                SKIP_BACKUP="true"
                shift
                ;;
            --skip-tests)
                SKIP_TESTS="true"
                shift
                ;;
            --version)
                VERSION="$2"
                shift 2
                ;;
            --no-auto-rollback)
                AUTO_ROLLBACK="false"
                shift
                ;;
            --help|-h)
                cat << EOF
HJTPX 更新脚本 v2.0

用法: $0 [选项]

选项:
    --no-backup         跳过备份创建
    --skip-tests        跳过测试
    --version <版本>    指定版本号 (默认: latest)
    --no-auto-rollback  失败时不自动回滚
    --help              显示帮助信息

示例:
    $0                           # 完整更新
    $0 --no-backup              # 跳过备份更新
    $0 --version v11.0          # 更新到指定版本
    $0 --skip-tests --no-backup # 跳过测试和备份

EOF
                exit 0
                ;;
            *)
                log_error "未知参数: $1"
                exit 1
                ;;
        esac
    done

    START_TIME=$(date +%s)

    update_code
    create_backup
    run_tests
    pull_latest_images
    build_images
    stop_services
    start_services

    if ! wait_for_services; then
        log_warning "服务启动超时，尝试继续..."
    fi

    if ! verify_update; then
        log_warning "验证失败，请检查服务状态"

        if [ "$AUTO_ROLLBACK" = "true" ] && [ "$SKIP_BACKUP" != "true" ]; then
            log_warning "执行自动回滚..."
            ./scripts/rollback.sh quick 2>/dev/null || true
        fi
    fi

    TOTAL_TIME=$(($(date +%s) - START_TIME))

    print_summary

    log_info "总耗时: ${TOTAL_TIME}秒"

    exit 0
}

main "$@"
