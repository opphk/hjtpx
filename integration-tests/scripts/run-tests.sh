#!/bin/bash

# HJTPX 集成测试运行脚本 v15.0

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
TEST_DIR="$PROJECT_ROOT/tests"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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

# 检查依赖
check_dependencies() {
    log_info "检查测试依赖..."
    
    if ! command -v go &> /dev/null; then
        log_error "Go 未安装，请先安装 Go 1.21+"
        exit 1
    fi
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装，请先安装 Docker"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose 未安装，请先安装 Docker Compose"
        exit 1
    fi
    
    go_version=$(go version | grep -oP '\d+\.\d+' | head -1)
    if [ "$(echo "$go_version < 1.21" | bc)" -eq 1 ]; then
        log_error "Go 版本需要 1.21+，当前版本: $go_version"
        exit 1
    fi
    
    log_success "所有依赖检查通过"
}

# 启动测试环境
start_test_environment() {
    log_info "启动测试环境..."
    
    cd "$PROJECT_ROOT"
    
    if [ -f docker-compose-test.yml ]; then
        docker-compose -f docker-compose-test.yml up -d
        
        log_info "等待服务启动..."
        sleep 10
        
        for i in {1..30}; do
            if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
                log_success "服务启动成功"
                return 0
            fi
            log_warning "等待服务启动... ($i/30)"
            sleep 2
        done
        
        log_error "服务启动超时"
        docker-compose -f docker-compose-test.yml logs
        exit 1
    else
        log_warning "未找到 docker-compose-test.yml，跳过环境启动"
        return 0
    fi
}

# 停止测试环境
stop_test_environment() {
    log_info "停止测试环境..."
    
    cd "$PROJECT_ROOT"
    
    if [ -f docker-compose-test.yml ]; then
        docker-compose -f docker-compose-test.yml down -v
        log_success "测试环境已停止"
    fi
}

# 运行测试
run_tests() {
    log_info "运行集成测试..."
    
    cd "$PROJECT_ROOT"
    
    # 设置测试环境变量
    export TEST_MODE=true
    export LOG_LEVEL=debug
    
    # 创建测试报告目录
    mkdir -p "$PROJECT_ROOT/reports"
    
    # 确定要运行的测试
    TEST_MODULE="${1:-all}"
    
    case "$TEST_MODULE" in
        "all")
            log_info "运行所有测试模块"
            go test ./tests/... \
                -v \
                -timeout 60s \
                -coverprofile=coverage.out \
                -json > test-results.json
            ;;
        "captcha")
            log_info "运行验证码模块测试"
            go test ./tests/captcha/... -v -timeout 60s
            ;;
        "auth")
            log_info "运行认证模块测试"
            go test ./tests/auth/... -v -timeout 60s
            ;;
        "admin")
            log_info "运行管理模块测试"
            go test ./tests/admin/... -v -timeout 60s
            ;;
        "environment")
            log_info "运行环境检测模块测试"
            go test ./tests/environment/... -v -timeout 60s
            ;;
        "performance")
            log_info "运行性能测试"
            go test ./tests/performance/... -v -timeout 120s
            ;;
        *)
            log_error "未知的测试模块: $TEST_MODULE"
            echo "可用模块: all, captcha, auth, admin, environment, performance"
            exit 1
            ;;
    esac
    
    # 生成测试报告
    if [ -f test-results.json ]; then
        log_info "生成测试报告..."
        go tool test2doc -i test-results.json -o reports/test-report.html 2>/dev/null || true
        log_success "测试报告已生成: reports/test-report.html"
    fi
    
    # 显示覆盖率
    if [ -f coverage.out ]; then
        log_info "测试覆盖率:"
        go tool cover -func=coverage.out
    fi
}

# 显示帮助
show_help() {
    cat << EOF
HJTPX 集成测试运行脚本 v15.0

用法: $0 [选项] [测试模块]

选项:
    -h, --help              显示帮助信息
    -s, --start             启动测试环境
    -k, --stop              停止测试环境
    -r, --restart           重启测试环境
    -t, --test <模块>       运行特定测试模块
    -a, --all               运行所有测试（默认）
    -c, --check             检查依赖
    -v, --verbose           显示详细输出

测试模块:
    all                     所有测试（默认）
    captcha                 验证码模块测试
    auth                    认证模块测试
    admin                   管理模块测试
    environment             环境检测模块测试
    performance             性能测试

示例:
    $0 --check              检查依赖
    $0 --start              启动测试环境
    $0 captcha              运行验证码测试
    $0 --all                运行所有测试
    $0 --stop               停止测试环境

EOF
}

# 主函数
main() {
    case "${1:-}" in
        -h|--help)
            show_help
            exit 0
            ;;
        -c|--check)
            check_dependencies
            ;;
        -s|--start)
            check_dependencies
            start_test_environment
            ;;
        -k|--stop)
            stop_test_environment
            ;;
        -r|--restart)
            check_dependencies
            stop_test_environment
            start_test_environment
            ;;
        -t|--test)
            check_dependencies
            run_tests "$2"
            ;;
        -a|--all)
            check_dependencies
            start_test_environment
            run_tests all
            ;;
        -v|--verbose)
            check_dependencies
            start_test_environment
            run_tests all
            ;;
        *)
            check_dependencies
            start_test_environment
            run_tests "${1:-all}"
            ;;
    esac
}

# 执行主函数
main "$@"
