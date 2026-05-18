#!/bin/sh
# =============================================================================
# Docker容器化部署测试脚本
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 测试配置
TEST_IMAGE_NAME="hjtpx-test"
TEST_CONTAINER_NAME="hjtpx-test-container"
TEST_PORT=8888

# =============================================================================
# 辅助函数
# =============================================================================

log_info() {
    echo "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo "${RED}[ERROR]${NC} $1"
}

cleanup() {
    log_info "Cleaning up test resources..."
    docker rm -f "$TEST_CONTAINER_NAME" > /dev/null 2>&1 || true
    docker rmi -f "$TEST_IMAGE_NAME" > /dev/null 2>&1 || true
}

# =============================================================================
# 测试函数
# =============================================================================

test_docker_installation() {
    log_info "Testing Docker installation..."
    if command -v docker > /dev/null 2>&1; then
        docker_version=$(docker --version 2>/dev/null || echo "unknown")
        log_info "Docker is installed: $docker_version"
        return 0
    else
        log_error "Docker is not installed"
        return 1
    fi
}

test_docker_compose_installation() {
    log_info "Testing Docker Compose installation..."
    if command -v docker-compose > /dev/null 2>&1 || docker compose version > /dev/null 2>&1; then
        compose_version=$(docker-compose --version 2>/dev/null || docker compose version 2>/dev/null || echo "unknown")
        log_info "Docker Compose is installed: $compose_version"
        return 0
    else
        log_error "Docker Compose is not installed"
        return 1
    fi
}

test_dockerfile_syntax() {
    log_info "Testing Dockerfile syntax..."
    if [ ! -f "Dockerfile" ]; then
        log_error "Dockerfile not found"
        return 1
    fi

    # 使用docker build --check来验证Dockerfile语法
    if docker build --no-cache -f Dockerfile --target validate . > /dev/null 2>&1; then
        log_info "Dockerfile syntax is valid"
        return 0
    else
        # 如果不支持--target validate，尝试普通构建测试
        if docker build --no-cache . > /dev/null 2>&1; then
            log_info "Dockerfile builds successfully"
            return 0
        else
            log_error "Dockerfile has syntax errors or build failed"
            return 1
        fi
    fi
}

test_docker_compose_config() {
    log_info "Testing docker-compose configuration..."
    if [ ! -f "docker-compose.yml" ]; then
        log_error "docker-compose.yml not found"
        return 1
    fi

    if docker-compose config > /dev/null 2>&1 || docker compose config > /dev/null 2>&1; then
        log_info "docker-compose.yml configuration is valid"
        return 0
    else
        log_error "docker-compose.yml has configuration errors"
        return 1
    fi
}

test_environment_file() {
    log_info "Testing environment file..."
    if [ -f ".env" ]; then
        log_info ".env file exists"
    elif [ -f ".env.example" ]; then
        log_info ".env.example file exists (recommended to copy to .env)"
    else
        log_warn "No environment file found"
    fi
    return 0
}

test_required_files() {
    log_info "Checking required files..."
    required_files="
        Dockerfile
        docker-compose.yml
        .dockerignore
        config.yaml
    "

    all_exist=true
    for file in $required_files; do
        if [ ! -f "$file" ]; then
            log_error "Required file missing: $file"
            all_exist=false
        else
            log_info "Found: $file"
        fi
    done

    if [ "$all_exist" = true ]; then
        return 0
    else
        return 1
    fi
}

test_dockerignore() {
    log_info "Testing .dockerignore file..."
    if [ -f ".dockerignore" ]; then
        # 检查是否忽略了不必要的大目录
        if grep -q "node_modules" .dockerignore && grep -q ".git" .dockerignore; then
            log_info ".dockerignore is properly configured"
            return 0
        else
            log_warn ".dockerignore might be missing some entries"
            return 1
        fi
    else
        log_warn ".dockerignore not found (recommended for reducing build context)"
        return 1
    fi
}

test_docker_directories() {
    log_info "Testing Docker-related directories..."
    required_dirs="
        docker
        scripts/docker
    "

    for dir in $required_dirs; do
        if [ -d "$dir" ]; then
            log_info "Found directory: $dir"
            # 检查目录中的脚本文件
            if [ -f "$dir/health-check.sh" ]; then
                log_info "  - health-check.sh exists"
            fi
            if [ -f "$dir/entrypoint.sh" ]; then
                log_info "  - entrypoint.sh exists"
            fi
        fi
    done
    return 0
}

# =============================================================================
# 主测试流程
# =============================================================================

main() {
    echo ""
    echo "========================================"
    echo "  Docker Containerization Test"
    echo "========================================"
    echo ""

    total=0
    passed=0
    failed=0

    tests="
        test_docker_installation
        test_docker_compose_installation
        test_required_files
        test_dockerfile_syntax
        test_docker_compose_config
        test_environment_file
        test_dockerignore
        test_docker_directories
    "

    for test in $tests; do
        total=$((total + 1))
        echo ""
        echo "----------------------------------------"
        echo "Test $total: $test"
        echo "----------------------------------------"

        if $test; then
            passed=$((passed + 1))
        else
            failed=$((failed + 1))
        fi
    done

    echo ""
    echo "========================================"
    echo "  Test Summary"
    echo "========================================"
    echo "Total:  $total"
    echo "Passed: ${GREEN}$passed${NC}"
    echo "Failed: ${RED}$failed${NC}"
    echo ""

    if [ $failed -eq 0 ]; then
        log_info "All tests passed!"
        echo ""
        echo "Next steps:"
        echo "1. Copy .env.example to .env and configure"
        echo "2. Run: docker-compose up -d"
        echo "3. Check logs: docker-compose logs -f"
        echo ""
        exit 0
    else
        log_error "Some tests failed"
        exit 1
    fi
}

# 清理
trap cleanup EXIT

main "$@"
