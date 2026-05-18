#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BENCHMARK_DIR="$PROJECT_ROOT/benchmark"
REPORTS_DIR="$BENCHMARK_DIR/reports/vegeta"
TARGETS_DIR="$BENCHMARK_DIR/targets"
VEGETA_VERSION="0.10.1"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"
DURATION="${DURATION:-60s}"
RATE="${RATE:-1000}"
WORKERS="${WORKERS:-100}"
TIMEOUT="${TIMEOUT:-30s}"

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

check_vegeta() {
    if ! command -v vegeta &> /dev/null; then
        log_warning "Vegeta not found, installing..."
        install_vegeta
    fi
    log_success "Vegeta is ready"
}

install_vegeta() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)
    
    case $arch in
        x86_64)
            arch="amd64"
            ;;
        aarch64|arm64)
            arch="arm64"
            ;;
        *)
            log_error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac
    
    local vegeta_file="vegeta_${VEGETA_VERSION}_${os}_${arch}.tar.gz"
    local vegeta_url="https://github.com/tsenart/vegeta/releases/download/v${VEGETA_VERSION}/${vegeta_file}"
    
    log_info "Downloading Vegeta from $vegeta_url..."
    
    mkdir -p /tmp/vegeta
    cd /tmp/vegeta
    
    if command -v curl &> /dev/null; then
        curl -sL "$vegeta_url" -o "$vegeta_file"
    elif command -v wget &> /dev/null; then
        wget -q "$vegeta_url" -O "$vegeta_file"
    else
        log_error "Neither curl nor wget found"
        exit 1
    fi
    
    tar -xzf "$vegeta_file"
    
    sudo mv vegeta /usr/local/bin/vegeta
    sudo chmod +x /usr/local/bin/vegeta
    
    cd /tmp
    rm -rf /tmp/vegeta
    
    log_success "Vegeta installed successfully"
}

setup_directories() {
    mkdir -p "$REPORTS_DIR"
    mkdir -p "$TARGETS_DIR"
    log_success "Directories created"
}

generate_targets() {
    log_info "Generating Vegeta target files..."
    
    cat > "$TARGETS_DIR/slider_generate.target" << EOF
POST ${API_BASE_URL}/api/v1/captcha/slider/generate
Content-Type: application/json
@slider_generate.json
EOF

    cat > "$TARGETS_DIR/slider_generate.json" << EOF
{
    "app_id": 1,
    "width": 320,
    "height": 160,
    "slider_size": 40
}
EOF

    cat > "$TARGETS_DIR/slider_verify.target" << EOF
POST ${API_BASE_URL}/api/v1/captcha/slider/verify
Content-Type: application/json
@slider_verify.json
EOF

    cat > "$TARGETS_DIR/slider_verify.json" << EOF
{
    "app_id": 1,
    "session_id": "vegeta_test_session",
    "x": 150,
    "track_data": [
        {"x": 10, "y": 5, "t": 50},
        {"x": 30, "y": 8, "t": 100},
        {"x": 60, "y": 10, "t": 150},
        {"x": 100, "y": 12, "t": 200},
        {"x": 150, "y": 15, "t": 300}
    ]
}
EOF

    cat > "$TARGETS_DIR/click_generate.target" << EOF
POST ${API_BASE_URL}/api/v1/captcha/click/generate
Content-Type: application/json
@click_generate.json
EOF

    cat > "$TARGETS_DIR/click_generate.json" << EOF
{
    "app_id": 1,
    "width": 320,
    "height": 160,
    "target_count": 4
}
EOF

    cat > "$TARGETS_DIR/image_generate.target" << EOF
POST ${API_BASE_URL}/api/v1/captcha/image/generate
Content-Type: application/json
@image_generate.json
EOF

    cat > "$TARGETS_DIR/image_generate.json" << EOF
{
    "app_id": 1,
    "length": 4,
    "width": 120,
    "height": 40
}
EOF

    cat > "$TARGETS_DIR/image_verify.target" << EOF
POST ${API_BASE_URL}/api/v1/captcha/image/verify
Content-Type: application/json
@image_verify.json
EOF

    cat > "$TARGETS_DIR/image_verify.json" << EOF
{
    "app_id": 1,
    "session_id": "vegeta_test_session",
    "captcha": "ABCD"
}
EOF

    cat > "$TARGETS_DIR/rotate_generate.target" << EOF
POST ${API_BASE_URL}/api/v1/captcha/rotate/generate
Content-Type: application/json
@rotate_generate.json
EOF

    cat > "$TARGETS_DIR/rotate_generate.json" << EOF
{
    "app_id": 1,
    "difficulty": "medium"
}
EOF

    cat > "$TARGETS_DIR/mixed.target" << EOF
POST ${API_BASE_URL}/api/v1/captcha/image/generate
Content-Type: application/json
@image_generate.json

POST ${API_BASE_URL}/api/v1/captcha/slider/generate
Content-Type: application/json
@slider_generate.json

POST ${API_BASE_URL}/api/v1/captcha/click/generate
Content-Type: application/json
@click_generate.json
EOF

    log_success "Target files generated"
}

run_benchmark() {
    local target_file=$1
    local name=$2
    local rate=${3:-$RATE}
    local duration=${4:-$DURATION}
    
    log_info "Running benchmark: $name"
    log_info "Target: $target_file"
    log_info "Rate: $rate req/s, Duration: $duration"
    
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local binary_file="$REPORTS_DIR/${name}_${timestamp}.bin"
    local report_file="$REPORTS_DIR/${name}_${timestamp}.json"
    local html_report="$REPORTS_DIR/${name}_${timestamp}.html"
    
    vegeta attack \
        -target="$TARGETS_DIR/$target_file" \
        -rate="$rate" \
        -duration="$duration" \
        -timeout="$TIMEOUT" \
        -workers="$WORKERS" \
        -output="$binary_file" 2>&1 | tee "$REPORTS_DIR/${name}_${timestamp}_attack.log"
    
    vegeta report \
        -type=json \
        "$binary_file" > "$report_file"
    
    vegeta report \
        -type=text \
        "$binary_file" > "$REPORTS_DIR/${name}_${timestamp}_report.txt"
    
    vegeta report \
        -type=histogram \
        -buckets='[10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000]' \
        "$binary_file" > "$REPORTS_DIR/${name}_${timestamp}_histogram.txt"
    
    generate_vegeta_html_report "$report_file" "$html_report" "$name"
    
    log_success "Benchmark completed: $name"
    log_info "Binary data: $binary_file"
    log_info "JSON report: $report_file"
    log_info "HTML report: $html_report"
}

generate_vegeta_html_report() {
    local json_file=$1
    local html_file=$2
    local name=$3
    
    local qps=$(jq -r '.throughput' "$json_file" 2>/dev/null || echo "0")
    local p50=$(jq -r '.latencies.p50' "$json_file" 2>/dev/null || echo "0")
    local p95=$(jq -r '.latencies.p95' "$json_file" 2>/dev/null || echo "0")
    local p99=$(jq -r '.latencies.p99' "$json_file" 2>/dev/null || echo "0")
    local errors=$(jq -r '.errors' "$json_file" 2>/dev/null || echo "0")
    local requests=$(jq -r '.requests' "$json_file" 2>/dev/null || echo "0")
    
    cat > "$html_file" << EOF
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Vegeta Benchmark Report - $name</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; padding: 20px; }
        .container { max-width: 1200px; margin: 0 auto; }
        h1 { color: #333; margin-bottom: 20px; padding: 20px; background: white; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .metrics { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 15px; margin-bottom: 30px; }
        .metric-card { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .metric-card h3 { font-size: 14px; color: #888; margin-bottom: 5px; }
        .metric-card .value { font-size: 24px; font-weight: bold; color: #333; }
        .metric-card .value.success { color: #28a745; }
        .metric-card .value.warning { color: #ffc107; }
        .metric-card .value.danger { color: #dc3545; }
        pre { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); overflow-x: auto; }
        .timestamp { color: #888; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Vegeta Benchmark Report: $name</h1>
        <p class="timestamp">Generated at: $(date -Iseconds)</p>
        
        <div class="metrics">
            <div class="metric-card">
                <h3>QPS (Throughput)</h3>
                <div class="value">$qps</div>
            </div>
            <div class="metric-card">
                <h3>P50 Latency</h3>
                <div class="value">$((p50 / 1000000))ms</div>
            </div>
            <div class="metric-card">
                <h3>P95 Latency</h3>
                <div class="value">$((p95 / 1000000))ms</div>
            </div>
            <div class="metric-card">
                <h3>P99 Latency</h3>
                <div class="value">$((p99 / 1000000))ms</div>
            </div>
            <div class="metric-card">
                <h3>Total Requests</h3>
                <div class="value">$requests</div>
            </div>
            <div class="metric-card">
                <h3>Errors</h3>
                <div class="value">$errors</div>
            </div>
        </div>
        
        <h2>Full JSON Report</h2>
        <pre>$(cat "$json_file")</pre>
    </div>
</body>
</html>
EOF
}

run_all_benchmarks() {
    log_info "Running all Vegeta benchmarks..."
    
    run_benchmark "slider_generate.target" "slider_generate" "$RATE" "$DURATION"
    run_benchmark "slider_verify.target" "slider_verify" "$RATE" "$DURATION"
    run_benchmark "click_generate.target" "click_generate" "$RATE" "$DURATION"
    run_benchmark "image_generate.target" "image_generate" "$RATE" "$DURATION"
    run_benchmark "image_verify.target" "image_verify" "$RATE" "$DURATION"
    
    log_success "All benchmarks completed"
}

run_mixed_benchmark() {
    log_info "Running mixed workload benchmark..."
    
    run_benchmark "mixed.target" "mixed_workload" "$RATE" "$DURATION"
}

run_peak_load() {
    log_info "Running peak load test..."
    
    local peak_rate=$((RATE * 2))
    local peak_duration="30s"
    
    run_benchmark "slider_generate.target" "peak_load" "$peak_rate" "$peak_duration"
}

generate_summary_report() {
    log_info "Generating summary report..."
    
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local summary_file="$REPORTS_DIR/summary_${timestamp}.html"
    
    cat > "$summary_file" << 'EOF'
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Vegeta Benchmark Summary</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; padding: 20px; }
        .container { max-width: 1400px; margin: 0 auto; }
        h1 { color: #333; margin-bottom: 20px; padding: 20px; background: white; border-radius: 8px; }
        table { width: 100%; border-collapse: collapse; background: white; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        th { background: #4a90d9; color: white; padding: 12px; text-align: left; }
        td { padding: 12px; border-bottom: 1px solid #eee; }
        .status { padding: 4px 8px; border-radius: 4px; font-size: 12px; font-weight: bold; }
        .status.pass { background: #d4edda; color: #155724; }
        .status.fail { background: #f8d7da; color: #721c24; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Vegeta Benchmark Summary</h1>
        <p>Generated at: 
EOF

    echo "          $(date -Iseconds)</p>" >> "$summary_file"
    
    cat >> "$summary_file" << 'EOF'
        
        <h2>All Benchmark Results</h2>
        <table>
            <thead>
                <tr>
                    <th>Benchmark</th>
                    <th>QPS</th>
                    <th>P50</th>
                    <th>P95</th>
                    <th>P99</th>
                    <th>Errors</th>
                    <th>Status</th>
                </tr>
            </thead>
            <tbody>
EOF

    for json_file in "$REPORTS_DIR"/*.json; do
        if [ -f "$json_file" ] && [[ "$json_file" != *"summary"* ]]; then
            local name=$(basename "$json_file" .json | sed 's/_[0-9]*_[0-9]*$//')
            local qps=$(jq -r '.throughput' "$json_file" 2>/dev/null || echo "0")
            local p50=$(( $(jq -r '.latencies.p50' "$json_file" 2>/dev/null || echo "0") / 1000000 ))
            local p95=$(( $(jq -r '.latencies.p95' "$json_file" 2>/dev/null || echo "0") / 1000000 ))
            local p99=$(( $(jq -r '.latencies.p99' "$json_file" 2>/dev/null || echo "0") / 1000000 ))
            local errors=$(jq -r '.errors' "$json_file" 2>/dev/null || echo "0")
            local status="pass"
            if (( $(echo "$qps < 1000" | bc -l 2>/dev/null || echo 1) )) || [ "$errors" -gt 100 ]; then
                status="fail"
            fi
            
            echo "                <tr>
                    <td>$name</td>
                    <td>$qps</td>
                    <td>${p50}ms</td>
                    <td>${p95}ms</td>
                    <td>${p99}ms</td>
                    <td>$errors</td>
                    <td><span class=\"status $status\">${status^^}</span></td>
                </tr>" >> "$summary_file"
        fi
    done
    
    cat >> "$summary_file" << 'EOF'
            </tbody>
        </table>
    </div>
</body>
</html>
EOF

    log_success "Summary report generated: $summary_file"
}

check_service() {
    log_info "Checking service at $API_BASE_URL..."
    
    if curl -sf "${API_BASE_URL}/health" > /dev/null 2>&1; then
        log_success "Service is running"
        return 0
    else
        log_error "Service is not running at $API_BASE_URL"
        return 1
    fi
}

show_help() {
    cat << EOF
Usage: $0 [OPTIONS] COMMAND

Vegeta-based HTTP load testing tool for hjtpx captcha system.

Commands:
    install          Install Vegeta
    targets          Generate target files
    benchmark        Run all benchmarks
    slider           Run slider captcha benchmarks
    click            Run click captcha benchmarks
    image            Run image captcha benchmarks
    mixed            Run mixed workload benchmark
    peak             Run peak load test
    summary          Generate summary report
    all              Run all: install targets benchmark summary

Options:
    --url URL        API base URL (default: http://localhost:8080)
    --rate RATE      Requests per second (default: 1000)
    --duration DUR   Test duration (default: 60s)
    --workers N      Number of workers (default: 100)
    --timeout TIMEOUT Request timeout (default: 30s)
    --help           Show this help message

Examples:
    $0 install
    $0 targets
    $0 benchmark
    $0 --url http://api:8080 --rate 2000 benchmark
    $0 mixed
    $0 summary

EOF
}

main() {
    local command=""
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --url)
                API_BASE_URL="$2"
                shift 2
                ;;
            --rate)
                RATE="$2"
                shift 2
                ;;
            --duration)
                DURATION="$2"
                shift 2
                ;;
            --workers)
                WORKERS="$2"
                shift 2
                ;;
            --timeout)
                TIMEOUT="$2"
                shift 2
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            install|targets|benchmark|slider|click|image|mixed|peak|summary|all)
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
    
    export API_BASE_URL DURATION RATE WORKERS TIMEOUT
    
    case $command in
        install)
            install_vegeta
            ;;
        targets)
            check_vegeta
            setup_directories
            generate_targets
            ;;
        benchmark)
            check_vegeta
            setup_directories
            if ! check_service; then
                log_error "Service not available"
                exit 1
            fi
            generate_targets
            run_all_benchmarks
            ;;
        slider)
            check_vegeta
            setup_directories
            if ! check_service; then
                log_error "Service not available"
                exit 1
            fi
            generate_targets
            run_benchmark "slider_generate.target" "slider_generate"
            run_benchmark "slider_verify.target" "slider_verify"
            ;;
        click)
            check_vegeta
            setup_directories
            if ! check_service; then
                log_error "Service not available"
                exit 1
            fi
            generate_targets
            run_benchmark "click_generate.target" "click_generate"
            ;;
        image)
            check_vegeta
            setup_directories
            if ! check_service; then
                log_error "Service not available"
                exit 1
            fi
            generate_targets
            run_benchmark "image_generate.target" "image_generate"
            run_benchmark "image_verify.target" "image_verify"
            ;;
        mixed)
            check_vegeta
            setup_directories
            if ! check_service; then
                log_error "Service not available"
                exit 1
            fi
            generate_targets
            run_mixed_benchmark
            ;;
        peak)
            check_vegeta
            setup_directories
            if ! check_service; then
                log_error "Service not available"
                exit 1
            fi
            generate_targets
            run_peak_load
            ;;
        summary)
            generate_summary_report
            ;;
        all)
            install_vegeta
            setup_directories
            generate_targets
            if check_service; then
                run_all_benchmarks
                run_mixed_benchmark
                run_peak_load
                generate_summary_report
            fi
            ;;
        *)
            log_error "Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

if [ "${BASH_SOURCE[0]}" = "$0" ]; then
    main "$@"
fi
