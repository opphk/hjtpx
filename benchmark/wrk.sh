#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BENCHMARK_DIR="$PROJECT_ROOT/benchmark"
REPORTS_DIR="$BENCHMARK_DIR/reports/wrk"
SCRIPTS_DIR="$BENCHMARK_DIR/scripts/wrk"
LUA_DIR="$BENCHMARK_DIR/scripts/wrk/lua"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"
DURATION="${DURATION:-60s}"
CONNECTIONS="${CONNECTIONS:-100}"
THREADS="${THREADS:-4}"

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

check_wrk() {
    if ! command -v wrk &> /dev/null; then
        log_warning "wrk not found, installing..."
        install_wrk
    fi
    log_success "wrk is ready"
}

install_wrk() {
    log_info "Installing wrk..."
    
    if command -v apt-get &> /dev/null; then
        sudo apt-get update -qq
        sudo apt-get install -y wrk
    elif command -v yum &> /dev/null; then
        sudo yum install -y wrk
    elif command -v brew &> /dev/null; then
        brew install wrk
    else
        log_info "Building wrk from source..."
        build_wrk_from_source
    fi
    
    log_success "wrk installed successfully"
}

build_wrk_from_source() {
    log_info "Building wrk from source..."
    
    local temp_dir="/tmp/wrk_build"
    mkdir -p "$temp_dir"
    cd "$temp_dir"
    
    if command -v git &> /dev/null; then
        if [ ! -d wrk ]; then
            git clone https://github.com/wg/wrk.git wrk
        fi
        cd wrk
        make
        sudo cp wrk /usr/local/bin/wrk
    else
        log_error "git not found, cannot build wrk from source"
        exit 1
    fi
    
    cd /tmp
    rm -rf "$temp_dir"
}

setup_directories() {
    mkdir -p "$REPORTS_DIR"
    mkdir -p "$SCRIPTS_DIR"
    mkdir -p "$LUA_DIR"
    log_success "Directories created"
}

generate_lua_scripts() {
    log_info "Generating Lua scripts for wrk..."
    
    cat > "$LUA_DIR/captcha_request.lua" << 'EOF'
-- Captcha Request Script
-- Generates dynamic request data for captcha endpoints

counter = 0

function setup(thread)
   thread:set("session_id", 0)
end

function init(args)
   math.randomseed(os.time() + math.random(1, 10000))
   counter = 0
end

function request()
   counter = counter + 1
   local session_id = string.format("wrk_session_%d_%d", counter, math.random(1000, 9999))
   
   wrk.headers["Content-Type"] = "application/json"
   
   local body = string.format([[
   {
       "app_id": 1,
       "session_id": "%s",
       "length": 4,
       "width": 120,
       "height": 40
   }]], session_id)
   
   return wrk.format("POST", "/api/v1/captcha/image/generate", wrk.headers, body)
end

function response(status, headers, body)
   if status ~= 200 and status ~= 201 then
      io.write("Error: " .. status .. "\n")
   end
end

function done(summary, latency, requests)
   io.write("\n")
   io.write("------------------------------\n")
   io.write("Latency Distribution:\n")
   io.write(string.format("  50%%: %d ms\n", latency:percentile(50) / 1000))
   io.write(string.format("  75%%: %d ms\n", latency:percentile(75) / 1000))
   io.write(string.format("  90%%: %d ms\n", latency:percentile(90) / 1000))
   io.write(string.format("  99%%: %d ms\n", latency:percentile(99) / 1000))
   io.write("------------------------------\n")
end
EOF

    cat > "$LUA_DIR/slider_request.lua" << 'EOF'
-- Slider Captcha Request Script
-- Generates dynamic slider captcha requests

counter = 0

function init(args)
   math.randomseed(os.time() + math.random(1, 10000))
   counter = 0
end

function request()
   counter = counter + 1
   local session_id = string.format("wrk_slider_%d_%d", counter, math.random(1000, 9999))
   
   wrk.headers["Content-Type"] = "application/json"
   
   local body = string.format([[
   {
       "app_id": 1,
       "width": 320,
       "height": 160,
       "slider_size": 40,
       "session_id": "%s"
   }]], session_id)
   
   return wrk.format("POST", "/api/v1/captcha/slider/generate", wrk.headers, body)
end
EOF

    cat > "$LUA_DIR/click_request.lua" << 'EOF'
-- Click Captcha Request Script
-- Generates dynamic click captcha requests

counter = 0

function init(args)
   math.randomseed(os.time() + math.random(1, 10000))
   counter = 0
end

function request()
   counter = counter + 1
   local session_id = string.format("wrk_click_%d_%d", counter, math.random(1000, 9999))
   
   wrk.headers["Content-Type"] = "application/json"
   
   local body = string.format([[
   {
       "app_id": 1,
       "width": 320,
       "height": 160,
       "target_count": 4,
       "session_id": "%s"
   }]], session_id)
   
   return wrk.format("POST", "/api/v1/captcha/click/generate", wrk.headers, body)
end
EOF

    cat > "$LUA_DIR/mixed_request.lua" << 'EOF'
-- Mixed Captcha Request Script
-- Rotates between different captcha types

local methods = {"image", "slider", "click"}
local counters = {0, 0, 0}

function init(args)
   math.randomseed(os.time() + math.random(1, 10000))
end

function request()
   local index = math.random(1, 3)
   local method = methods[index]
   counters[index] = counters[index] + 1
   local session_id = string.format("wrk_%s_%d_%d", method, counters[index], math.random(1000, 9999))
   
   wrk.headers["Content-Type"] = "application/json"
   
   local path, body
   
   if method == "image" then
      path = "/api/v1/captcha/image/generate"
      body = string.format([[{"app_id": 1, "length": 4, "width": 120, "height": 40, "session_id": "%s"}]], session_id)
   elseif method == "slider" then
      path = "/api/v1/captcha/slider/generate"
      body = string.format([[{"app_id": 1, "width": 320, "height": 160, "slider_size": 40, "session_id": "%s"}]], session_id)
   else
      path = "/api/v1/captcha/click/generate"
      body = string.format([[{"app_id": 1, "width": 320, "height": 160, "target_count": 4, "session_id": "%s"}]], session_id)
   end
   
   return wrk.format("POST", path, wrk.headers, body)
end
EOF

    cat > "$LUA_DIR/health_check.lua" << 'EOF'
-- Health Check Request Script
-- Simple health check endpoint

function request()
   return wrk.format("GET", "/health")
end
EOF

    log_success "Lua scripts generated"
}

generate_wrk_configs() {
    log_info "Generating wrk configuration files..."
    
    cat > "$SCRIPTS_DIR/image_generate.wrk" << EOF
#!/usr/bin/env bash

# Image Captcha Generation Benchmark
# Target: ${API_BASE_URL}/api/v1/captcha/image/generate

wrk \
    -t${THREADS} \
    -c${CONNECTIONS} \
    -d${DURATION} \
    -s "$LUA_DIR/captcha_request.lua" \
    "${API_BASE_URL}/api/v1/captcha/image/generate"
EOF

    cat > "$SCRIPTS_DIR/slider_generate.wrk" << EOF
#!/usr/bin/env bash

# Slider Captcha Generation Benchmark
# Target: ${API_BASE_URL}/api/v1/captcha/slider/generate

wrk \
    -t${THREADS} \
    -c${CONNECTIONS} \
    -d${DURATION} \
    -s "$LUA_DIR/slider_request.lua" \
    "${API_BASE_URL}/api/v1/captcha/slider/generate"
EOF

    cat > "$SCRIPTS_DIR/click_generate.wrk" << EOF
#!/usr/bin/env bash

# Click Captcha Generation Benchmark
# Target: ${API_BASE_URL}/api/v1/captcha/click/generate

wrk \
    -t${THREADS} \
    -c${CONNECTIONS} \
    -d${DURATION} \
    -s "$LUA_DIR/click_request.lua" \
    "${API_BASE_URL}/api/v1/captcha/click/generate"
EOF

    cat > "$SCRIPTS_DIR/mixed.wrk" << EOF
#!/usr/bin/env bash

# Mixed Captcha Workload Benchmark
# Rotates between image, slider, and click captcha generation

wrk \
    -t${THREADS} \
    -c${CONNECTIONS} \
    -d${DURATION} \
    -s "$LUA_DIR/mixed_request.lua" \
    "${API_BASE_URL}/api/v1/captcha/image/generate"
EOF

    chmod +x "$SCRIPTS_DIR"/*.wrk
    
    log_success "wrk configuration files generated"
}

run_wrk_benchmark() {
    local name=$1
    local script=$2
    local duration=${3:-$DURATION}
    local connections=${4:-$CONNECTIONS}
    local threads=${5:-$THREADS}
    
    log_info "Running wrk benchmark: $name"
    
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local output_file="$REPORTS_DIR/${name}_${timestamp}.txt"
    local json_file="$REPORTS_DIR/${name}_${timestamp}.json"
    
    echo "Running wrk benchmark: $name" | tee "$output_file"
    echo "Timestamp: $(date)" | tee -a "$output_file"
    echo "Duration: $duration" | tee -a "$output_file"
    echo "Connections: $connections" | tee -a "$output_file"
    echo "Threads: $threads" | tee -a "$output_file"
    echo "----------------------------------------" | tee -a "$output_file"
    
    if [ -n "$script" ] && [ -f "$LUA_DIR/$script.lua" ]; then
        wrk \
            -t"$threads" \
            -c"$connections" \
            -d"$duration" \
            -s "$LUA_DIR/$script.lua" \
            "${API_BASE_URL}/api/v1/captcha/image/generate" 2>&1 | tee -a "$output_file"
    else
        wrk \
            -t"$threads" \
            -c"$connections" \
            -d"$duration" \
            "${API_BASE_URL}/api/v1/captcha/image/generate" 2>&1 | tee -a "$output_file"
    fi
    
    parse_wrk_output "$output_file" "$json_file"
    
    log_success "Benchmark completed: $name"
    log_info "Output: $output_file"
}

parse_wrk_output() {
    local input_file=$1
    local json_file=$2
    
    if ! command -v python3 &> /dev/null && ! command -v python &> /dev/null; then
        log_warning "Python not found, skipping JSON parsing"
        return
    fi
    
    local python_cmd="python3"
    if ! command -v python3 &> /dev/null; then
        python_cmd="python"
    fi
    
    $python_cmd << PYEOF
import json
import re
import sys

input_file = "$input_file"
json_file = "$json_file"

with open(input_file, 'r') as f:
    content = f.read()

result = {
    "name": "$(basename "$input_file")",
    "timestamp": "$(date -Iseconds)",
    "requests": 0,
    "duration": 0,
    "qps": 0,
    "latency_avg": 0,
    "latency_stdev": 0,
    "latency_p50": 0,
    "latency_p75": 0,
    "latency_p90": 0,
    "latency_p99": 0,
    "latency_p999": 0,
    "errors": 0,
    "non_2xx": 0
}

req_match = re.search(r'(\d+) requests in (\d+\.?\d*)([a-z]+)', content)
if req_match:
    result['requests'] = int(req_match.group(1))
    dur_str = req_match.group(2)
    dur_unit = req_match.group(3)
    duration = float(dur_str)
    if dur_unit == 's':
        duration *= 1
    elif dur_unit == 'm':
        duration *= 60
    elif dur_unit == 'h':
        duration *= 3600
    result['duration'] = duration
    result['qps'] = result['requests'] / duration

latency_match = re.search(r'Latency\s+(\d+\.?\d*)([a-z]+)\s+(\d+\.?\d*)([a-z]+)\s+(\d+\.?\d*)([a-z]+)', content)
if latency_match:
    result['latency_avg'] = latency_match.group(1)
    
stdev_match = re.search(r'(\d+\.?\d*)([a-z]+)\s+\+/\s+(\d+\.?\d*)([a-z]+)', content)
if stdev_match:
    result['latency_stdev'] = stdev_match.group(3)

percentile_match = re.findall(r'(\d+)%\s+(\d+\.?\d*)([a-z]+)', content)
for p, val, unit in percentile_match:
    if p == '50':
        result['latency_p50'] = val
    elif p == '75':
        result['latency_p75'] = val
    elif p == '90':
        result['latency_p90'] = val
    elif p == '99':
        result['latency_p99'] = val
    elif p == '99.9':
        result['latency_p999'] = val

error_match = re.search(r'(\d+) errors', content)
if error_match:
    result['errors'] = int(error_match.group(1))

non2xx_match = re.search(r'(\d+) non-2xx responses', content)
if non2xx_match:
    result['non_2xx'] = int(non2xx_match.group(1))

with open(json_file, 'w') as f:
    json.dump(result, f, indent=2)

print(f"JSON report saved to: {json_file}")
PYEOF
}

generate_html_report() {
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local html_file="$REPORTS_DIR/wrk_report_${timestamp}.html"
    
    log_info "Generating HTML report..."
    
    cat > "$html_file" << EOF
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>wrk Benchmark Report</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; padding: 20px; }
        .container { max-width: 1400px; margin: 0 auto; }
        h1 { color: #333; margin-bottom: 20px; padding: 20px; background: white; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .metrics { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 15px; margin-bottom: 30px; }
        .metric-card { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .metric-card h3 { font-size: 14px; color: #888; margin-bottom: 5px; }
        .metric-card .value { font-size: 24px; font-weight: bold; color: #333; }
        .metric-card .value.success { color: #28a745; }
        .metric-card .value.warning { color: #ffc107; }
        .metric-card .value.danger { color: #dc3545; }
        table { width: 100%; border-collapse: collapse; background: white; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 4px rgba(0,0,0,0.1); margin-bottom: 30px; }
        th { background: #4a90d9; color: white; padding: 12px; text-align: left; }
        td { padding: 12px; border-bottom: 1px solid #eee; }
        .timestamp { color: #888; font-size: 12px; }
        .status { padding: 4px 8px; border-radius: 4px; font-size: 12px; font-weight: bold; }
        .status.pass { background: #d4edda; color: #155724; }
        .status.fail { background: #f8d7da; color: #721c24; }
    </style>
</head>
<body>
    <div class="container">
        <h1>wrk Benchmark Report</h1>
        <p class="timestamp">Generated at: $(date -Iseconds)</p>
        
        <h2>Summary</h2>
        <div class="metrics">
EOF

    local total_requests=0
    local total_qps=0
    local count=0
    
    for json_file in "$REPORTS_DIR"/*.json; do
        if [ -f "$json_file" ]; then
            local requests=$(grep -o '"requests": [0-9]*' "$json_file" | grep -o '[0-9]*' || echo "0")
            local qps=$(grep -o '"qps": [0-9.]*' "$json_file" | grep -o '[0-9.]*' || echo "0")
            total_requests=$((total_requests + requests))
            total_qps=$(echo "$total_qps + $qps" | bc -l 2>/dev/null || echo "$total_qps")
            count=$((count + 1))
        fi
    done
    
    echo "            <div class=\"metric-card\">
                <h3>Total Requests</h3>
                <div class=\"value\">$total_requests</div>
            </div>
            <div class=\"metric-card\">
                <h3>Combined QPS</h3>
                <div class=\"value\">$(printf '%.2f' "$total_qps")</div>
            </div>
            <div class=\"metric-card\">
                <h3>Benchmarks Run</h3>
                <div class=\"value\">$count</div>
            </div>
        </div>
        
        <h2>Detailed Results</h2>
        <table>
            <thead>
                <tr>
                    <th>Benchmark</th>
                    <th>Requests</th>
                    <th>QPS</th>
                    <th>P50</th>
                    <th>P90</th>
                    <th>P99</th>
                    <th>Status</th>
                </tr>
            </thead>
            <tbody>" >> "$html_file"
    
    for json_file in "$REPORTS_DIR"/*.json; do
        if [ -f "$json_file" ]; then
            local name=$(basename "$json_file" .json | sed 's/_[0-9]*_[0-9]*$//')
            local requests=$(grep -o '"requests": [0-9]*' "$json_file" | head -1 | grep -o '[0-9]*' || echo "0")
            local qps=$(grep -o '"qps": [0-9.]*' "$json_file" | head -1 | grep -o '[0-9.]*' || echo "0")
            local p50=$(grep -o '"latency_p50": [0-9.]*' "$json_file" | head -1 | grep -o '[0-9.]*' || echo "0")
            local p90=$(grep -o '"latency_p90": [0-9.]*' "$json_file" | head -1 | grep -o '[0-9.]*' || echo "0")
            local p99=$(grep -o '"latency_p99": [0-9.]*' "$json_file" | head -1 | grep -o '[0-9.]*' || echo "0")
            
            local status="pass"
            if (( $(echo "$qps < 1000" | bc -l 2>/dev/null || echo 1) )); then
                status="fail"
            fi
            
            echo "                <tr>
                    <td>$name</td>
                    <td>$requests</td>
                    <td>$(printf '%.2f' "$qps")</td>
                    <td>${p50}ms</td>
                    <td>${p90}ms</td>
                    <td>${p99}ms</td>
                    <td><span class=\"status $status\">${status^^}</span></td>
                </tr>" >> "$html_file"
        fi
    done
    
    cat >> "$html_file" << 'EOF'
            </tbody>
        </table>
    </div>
</body>
</html>
EOF

    log_success "HTML report generated: $html_file"
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

run_all_benchmarks() {
    log_info "Running all wrk benchmarks..."
    
    run_wrk_benchmark "image_generate" "captcha_request.lua"
    run_wrk_benchmark "slider_generate" "slider_request.lua"
    run_wrk_benchmark "click_generate" "click_request.lua"
    run_wrk_benchmark "mixed" "mixed_request.lua"
    
    generate_html_report
    
    log_success "All benchmarks completed"
}

show_help() {
    cat << EOF
Usage: $0 [OPTIONS] COMMAND

wrk-based HTTP benchmarking tool for hjtpx captcha system.

Commands:
    install          Install wrk
    setup            Generate Lua scripts and configs
    benchmark        Run all wrk benchmarks
    image            Run image captcha benchmark
    slider           Run slider captcha benchmark
    click            Run click captcha benchmark
    mixed            Run mixed workload benchmark
    report           Generate HTML report
    all              Run all: install setup benchmark

Options:
    --url URL        API base URL (default: http://localhost:8080)
    --duration DUR   Test duration (default: 60s)
    --connections N  Number of connections (default: 100)
    --threads N      Number of threads (default: 4)
    --help           Show this help message

Examples:
    $0 install
    $0 setup
    $0 benchmark
    $0 --url http://api:8080 --duration 30s benchmark
    $0 image
    $0 mixed

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
            --duration)
                DURATION="$2"
                shift 2
                ;;
            --connections)
                CONNECTIONS="$2"
                shift 2
                ;;
            --threads)
                THREADS="$2"
                shift 2
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            install|setup|benchmark|image|slider|click|mixed|report|all)
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
    
    export API_BASE_URL DURATION CONNECTIONS THREADS
    
    case $command in
        install)
            install_wrk
            ;;
        setup)
            check_wrk
            setup_directories
            generate_lua_scripts
            generate_wrk_configs
            ;;
        benchmark)
            check_wrk
            setup_directories
            generate_lua_scripts
            generate_wrk_configs
            
            if ! check_service; then
                log_error "Service not available"
                exit 1
            fi
            
            run_all_benchmarks
            ;;
        image)
            check_wrk
            setup_directories
            generate_lua_scripts
            
            if ! check_service; then
                log_error "Service not available"
                exit 1
            fi
            
            run_wrk_benchmark "image_generate" "captcha_request.lua"
            ;;
        slider)
            check_wrk
            setup_directories
            generate_lua_scripts
            
            if ! check_service; then
                log_error "Service not available"
                exit 1
            fi
            
            run_wrk_benchmark "slider_generate" "slider_request.lua"
            ;;
        click)
            check_wrk
            setup_directories
            generate_lua_scripts
            
            if ! check_service; then
                log_error "Service not available"
                exit 1
            fi
            
            run_wrk_benchmark "click_generate" "click_request.lua"
            ;;
        mixed)
            check_wrk
            setup_directories
            generate_lua_scripts
            
            if ! check_service; then
                log_error "Service not available"
                exit 1
            fi
            
            run_wrk_benchmark "mixed" "mixed_request.lua"
            ;;
        report)
            generate_html_report
            ;;
        all)
            install_wrk
            check_wrk
            setup_directories
            generate_lua_scripts
            generate_wrk_configs
            
            if check_service; then
                run_all_benchmarks
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
