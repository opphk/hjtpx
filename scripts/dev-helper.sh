#!/bin/bash

# HJTPX v19.0 开发助手脚本
# 用于提高开发效率和本地开发体验

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_DIR="$PROJECT_ROOT/backend"

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

show_help() {
    cat << EOF
HJTPX v19.0 开发助手

用法:
    $(basename "$0") [命令] [选项]

命令:
    setup          设置开发环境
    run            启动开发服务器
    test           运行测试
    test-fuzz      运行模糊测试
    test-chaos     运行混沌工程测试
    test-security  运行安全测试
    test-all       运行所有测试
    build          构建项目
    clean          清理构建产物
    lint           运行代码检查
    fmt            格式化代码
    help           显示帮助信息

选项:
    -h, --help     显示帮助信息
    -v, --verbose  详细输出

示例:
    $(basename "$0") setup
    $(basename "$0") run
    $(basename "$0") test-all
EOF
}

setup_environment() {
    log_info "正在设置开发环境..."
    
    cd "$BACKEND_DIR"
    
    if ! command -v go &> /dev/null; then
        log_error "未找到 Go，请先安装 Go 1.21+"
        exit 1
    fi
    
    log_info "Go 版本: $(go version)"
    
    log_info "下载依赖..."
    go mod download
    go mod tidy
    
    log_info "检查 Docker..."
    if command -v docker &> /dev/null; then
        log_info "Docker 版本: $(docker --version)"
    else
        log_warning "未找到 Docker，某些功能可能不可用"
    fi
    
    log_success "开发环境设置完成！"
}

run_development() {
    log_info "启动开发服务器..."
    cd "$BACKEND_DIR"
    
    if [ ! -f "$PROJECT_ROOT/.env" ]; then
        log_warning "未找到 .env 文件，正在从 .env.example 创建..."
        cp "$PROJECT_ROOT/.env.example" "$PROJECT_ROOT/.env"
    fi
    
    go run cmd/api/main.go
}

run_tests() {
    log_info "运行测试..."
    cd "$BACKEND_DIR"
    make test
}

run_fuzz_tests() {
    log_info "运行模糊测试..."
    cd "$BACKEND_DIR"
    make test-fuzz
}

run_chaos_tests() {
    log_info "运行混沌工程测试..."
    cd "$BACKEND_DIR"
    make test-chaos
}

run_security_tests() {
    log_info "运行安全测试..."
    cd "$BACKEND_DIR"
    make test-security
}

run_all_tests() {
    log_info "运行所有测试 (v19.0)..."
    cd "$BACKEND_DIR"
    make test-all-v3
}

build_project() {
    log_info "构建项目..."
    cd "$BACKEND_DIR"
    make build
}

clean_project() {
    log_info "清理构建产物..."
    cd "$BACKEND_DIR"
    make clean
}

run_lint() {
    log_info "运行代码检查..."
    cd "$BACKEND_DIR"
    if command -v golangci-lint &> /dev/null; then
        make lint
    else
        log_warning "未找到 golangci-lint，正在使用 go vet"
        make vet
    fi
}

format_code() {
    log_info "格式化代码..."
    cd "$BACKEND_DIR"
    make fmt
}

main() {
    case "${1:-help}" in
        setup)
            setup_environment
            ;;
        run)
            run_development
            ;;
        test)
            run_tests
            ;;
        test-fuzz)
            run_fuzz_tests
            ;;
        test-chaos)
            run_chaos_tests
            ;;
        test-security)
            run_security_tests
            ;;
        test-all)
            run_all_tests
            ;;
        build)
            build_project
            ;;
        clean)
            clean_project
            ;;
        lint)
            run_lint
            ;;
        fmt)
            format_code
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            log_error "未知命令: $1"
            echo
            show_help
            exit 1
            ;;
    esac
}

main "$@"
