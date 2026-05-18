#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BENCHMARK_DIR="$PROJECT_ROOT/benchmark"
REPORT_DIR="$PROJECT_ROOT/benchmark/reports"
BASELINE_DIR="$PROJECT_ROOT/benchmark/baselines"

API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"
BENCHMARK_DURATION="${BENCHMARK_DURATION:-60}"
CONCURRENCY="${CONCURRENCY:-100}"
OUTPUT_FORMAT="${OUTPUT_FORMAT:-html}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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

check_dependencies() {
    log_info "Checking dependencies..."

    if ! command -v go &> /dev/null; then
        log_error "Go is not installed"
        exit 1
    fi

    if ! command -v jq &> /dev/null; then
        log_warning "jq is not installed, JSON report formatting may be limited"
    fi

    if ! command -v curl &> /dev/null; then
        log_error "curl is not installed"
        exit 1
    fi

    log_success "Dependencies check passed"
}

check_service() {
    log_info "Checking if service is running at $API_BASE_URL..."

    if curl -sf "$API_BASE_URL/health" > /dev/null 2>&1; then
        log_success "Service is running"
        return 0
    else
        log_error "Service is not running at $API_BASE_URL"
        return 1
    fi
}

wait_for_service() {
    local max_attempts=30
    local attempt=1

    log_info "Waiting for service to be ready..."

    while [ $attempt -le $max_attempts ]; do
        if curl -sf "$API_BASE_URL/health" > /dev/null 2>&1; then
            log_success "Service is ready"
            return 0
        fi

        echo -n "."
        sleep 2
        ((attempt++))
    done

    echo ""
    log_error "Service did not become ready in time"
    return 1
}

setup_directories() {
    log_info "Setting up directories..."

    mkdir -p "$REPORT_DIR"
    mkdir -p "$BASELINE_DIR"
    mkdir -p "$BENCHMARK_DIR/cmd/benchmark"

    log_success "Directories created"
}

run_unit_benchmarks() {
    log_info "Running unit benchmarks..."

    cd "$PROJECT_ROOT/benchmark"

    go test -bench=. -benchmem -memprofile=mem.prof -cpuprofile=cpu.prof \
        -run=^$ -timeout=10m

    log_success "Unit benchmarks completed"
}

run_scenario_benchmarks() {
    log_info "Running scenario benchmarks..."
    log_info "Base URL: $API_BASE_URL"
    log_info "Duration: ${BENCHMARK_DURATION}s"
    log_info "Concurrency: $CONCURRENCY"

    cd "$BENCHMARK_DIR"

    go run cmd/benchmark/main.go

    log_success "Scenario benchmarks completed"
}

generate_baseline() {
    local mode="${1:-full}"

    log_info "Generating baseline (mode: $mode)..."

    case $mode in
        quick)
            export BENCHMARK_DURATION=30
            export CONCURRENCY=50
            ;;
        full)
            export BENCHMARK_DURATION=120
            export CONCURRENCY=200
            ;;
        *)
            log_warning "Unknown mode: $mode, using default"
            ;;
    esac

    run_scenario_benchmarks

    log_success "Baseline generation completed"
    log_info "Baseline stored in $BASELINE_DIR"
}

compare_baseline() {
    log_info "Comparing with baseline..."

    if [ ! -f "$BASELINE_DIR/baseline.json" ]; then
        log_error "No baseline found. Run with --baseline first."
        exit 1
    fi

    run_scenario_benchmarks

    log_success "Comparison completed"
    log_info "Results available in $REPORT_DIR"
}

run_regression_test() {
    log_info "Running regression tests..."

    export BENCHMARK_DURATION=60
    export CONCURRENCY=100

    if ! check_service; then
        if [ "${CI:-false}" = "true" ]; then
            log_error "Cannot run regression tests without service"
            exit 1
        fi
        log_warning "Starting service for regression tests..."
        start_service
        wait_for_service
    fi

    run_scenario_benchmarks

    local report_file=$(ls -t "$REPORT_DIR"/benchmark_report_*.html 2>/dev/null | head -1)

    if [ -n "$report_file" ]; then
        log_success "Regression test report: $report_file"
    fi
}

start_service() {
    log_info "Starting backend service..."

    cd "$PROJECT_ROOT/backend"

    nohup go run cmd/api/main.go > "$PROJECT_ROOT/backend.log" 2>&1 &
    echo $! > "$PROJECT_ROOT/backend.pid"

    log_success "Service started (PID: $(cat "$PROJECT_ROOT/backend.pid"))"
}

stop_service() {
    if [ -f "$PROJECT_ROOT/backend.pid" ]; then
        local pid=$(cat "$PROJECT_ROOT/backend.pid")
        if kill -0 "$pid" 2>/dev/null; then
            log_info "Stopping service (PID: $pid)..."
            kill "$pid" 2>/dev/null || true
            sleep 2
        fi
        rm -f "$PROJECT_ROOT/backend.pid"
    fi
}

run_ci_benchmark() {
    log_info "Running CI benchmark..."

    export CI=true
    export BENCHMARK_DURATION=60
    export CONCURRENCY=100
    export OUTPUT_FORMAT=json

    setup_directories

    if ! check_service; then
        log_warning "Service not available, skipping benchmark"
        exit 0
    fi

    run_scenario_benchmarks

    local report_file=$(ls -t "$REPORT_DIR"/benchmark_report_*.json 2>/dev/null | head -1)

    if [ -n "$report_file" ]; then
        log_success "CI benchmark report: $report_file"

        if command -v jq &> /dev/null; then
            local regressions=$(jq -r '.regressions | length' "$report_file" 2>/dev/null || echo "0")
            if [ "$regressions" -gt 0 ]; then
                log_error "Found $regressions performance regressions"
                exit 1
            fi
        fi
    fi

    log_success "CI benchmark completed"
}

show_help() {
    cat << EOF
Usage: $0 [OPTIONS] COMMAND

Performance Benchmark Automation Script

Commands:
    run                 Run all benchmarks
    quick               Run quick benchmarks (30s duration)
    full                Run full benchmarks (120s duration)
    baseline            Generate baseline metrics
    compare             Compare with existing baseline
    regression          Run regression tests
    ci                  Run CI benchmark
    serve               Start service and run benchmarks

Options:
    --url URL           API base URL (default: http://localhost:8080)
    --duration SECONDS  Benchmark duration (default: 60)
    --concurrency N     Concurrent requests (default: 100)
    --format FORMAT     Output format: html, json (default: html)
    --help              Show this help message

Examples:
    $0 run                                    # Run all benchmarks
    $0 quick                                  # Quick benchmark
    $0 full                                   # Full benchmark
    $0 baseline                               # Generate baseline
    $0 compare                                # Compare with baseline
    $0 --url http://api:8080 run               # Custom URL
    $0 --duration 30 --concurrency 50 quick    # Custom parameters

EOF
}

main() {
    local command=""
    local mode="normal"

    while [[ $# -gt 0 ]]; do
        case $1 in
            --url)
                API_BASE_URL="$2"
                shift 2
                ;;
            --duration)
                BENCHMARK_DURATION="$2"
                shift 2
                ;;
            --concurrency)
                CONCURRENCY="$2"
                shift 2
                ;;
            --format)
                OUTPUT_FORMAT="$2"
                shift 2
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            run|quick|full|baseline|compare|regression|ci|serve)
                command="$1"
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done

    if [ -z "$command" ]; then
        log_error "No command specified"
        show_help
        exit 1
    fi

    check_dependencies
    setup_directories

    case $command in
        run)
            if ! check_service; then
                log_error "Service is not running. Start it first or use 'serve' command."
                exit 1
            fi
            run_scenario_benchmarks
            ;;
        quick)
            if ! check_service; then
                log_error "Service is not running."
                exit 1
            fi
            export BENCHMARK_DURATION=30
            export CONCURRENCY=50
            run_scenario_benchmarks
            ;;
        full)
            if ! check_service; then
                log_error "Service is not running."
                exit 1
            fi
            export BENCHMARK_DURATION=120
            export CONCURRENCY=200
            run_scenario_benchmarks
            ;;
        baseline)
            if ! check_service; then
                log_error "Service is not running."
                exit 1
            fi
            generate_baseline full
            ;;
        compare)
            compare_baseline
            ;;
        regression)
            run_regression_test
            ;;
        ci)
            run_ci_benchmark
            ;;
        serve)
            trap stop_service EXIT
            start_service
            wait_for_service
            run_scenario_benchmarks
            ;;
        *)
            log_error "Unknown command: $command"
            exit 1
            ;;
    esac
}

if [ "${BASH_SOURCE[0]}" = "$0" ]; then
    main "$@"
fi
