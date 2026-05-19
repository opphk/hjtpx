#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

LOG_DIR="${LOG_DIR:-./logs}"
APP_URL="${APP_URL:-http://localhost:8080}"
POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_USER="${POSTGRES_USER:-postgres}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-postgres}"
POSTGRES_DB="${POSTGRES_DB:-hjtpx_db}"
REDIS_HOST="${REDIS_HOST:-localhost}"
REDIS_PORT="${REDIS_PORT:-6379}"
REDIS_PASSWORD="${REDIS_PASSWORD:-}"
REPORT_DIR="${REPORT_DIR:-./log-reports}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

mkdir -p "$REPORT_DIR"

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

show_help() {
    cat << EOF
HJTPX 日志分析工具

用法: $0 [选项] [命令]

选项:
    -l, --log-dir DIR        日志目录 (默认: ./logs)
    -o, --output-dir DIR     报告输出目录 (默认: ./log-reports)
    -s, --service SERVICE    服务名称 (app, postgres, redis, all)
    -t, --time TIME_RANGE    时间范围 (1h, 6h, 24h, 7d)
    -f, --format FORMAT     输出格式 (text, json, html)
    -h, --help              显示帮助信息

命令:
    stats                   显示日志统计信息
    errors                  分析错误日志
    performance             性能分析
    security                安全事件分析
    report                  生成完整报告
    watch                   实时监控日志
    export                  导出日志数据

示例:
    $0 stats
    $0 errors --time 24h
    $0 report --format html
    $0 watch --service app

EOF
}

parse_log_line() {
    local line="$1"
    echo "$line" | python3 -c "
import sys, json
try:
    log = json.loads(sys.stdin.read().strip())
    print(json.dumps(log, indent=2))
except:
    print(sys.stdin.read().strip())
" 2>/dev/null || echo "$line"
}

analyze_stats() {
    log_info "开始统计日志信息..."

    local stats_file="$REPORT_DIR/stats_${TIMESTAMP}.txt"
    {
        echo "==================================="
        echo "HJTPX 日志统计报告"
        echo "生成时间: $(date '+%Y-%m-%d %H:%M:%S')"
        echo "==================================="
        echo ""

        echo "1. 日志文件统计"
        echo "-------------------------------------------"
        if [ -d "$LOG_DIR" ]; then
            echo "日志目录: $LOG_DIR"
            echo "日志文件总数: $(find "$LOG_DIR" -type f 2>/dev/null | wc -l)"
            echo "日志总大小: $(du -sh "$LOG_DIR" 2>/dev/null | cut -f1)"
            echo ""

            echo "各服务日志统计:"
            for service in app postgres redis; do
                service_logs=$(find "$LOG_DIR" -name "*${service}*" -type f 2>/dev/null)
                if [ -n "$service_logs" ]; then
                    echo "  $service:"
                    echo "    文件数: $(echo "$service_logs" | wc -w)"
                    echo "    总大小: $(du -ch $service_logs 2>/dev/null | tail -1 | cut -f1)"
                    echo "    行数: $(wc -l $service_logs 2>/dev/null | tail -1 | awk '{print $1}')"
                fi
            done
        fi
        echo ""

        echo "2. 应用日志统计"
        echo "-------------------------------------------"
        if [ -f "$LOG_DIR/app/app.log" ]; then
            app_log="$LOG_DIR/app/app.log"

            echo "总日志行数: $(wc -l < "$app_log")"
            echo "INFO 日志: $(grep -c '"level":"info"' "$app_log" 2>/dev/null || echo '0')"
            echo "WARNING 日志: $(grep -c '"level":"warn"' "$app_log" 2>/dev/null || echo '0')"
            echo "ERROR 日志: $(grep -c '"level":"error"' "$app_log" 2>/dev/null || echo '0')"
            echo "DEBUG 日志: $(grep -c '"level":"debug"' "$app_log" 2>/dev/null || echo '0')"
            echo ""

            echo "请求统计:"
            echo "  总请求数: $(grep -c '"type":"request"' "$app_log" 2>/dev/null || echo '0')"
            echo "  验证码请求: $(grep -c '"type":"captcha"' "$app_log" 2>/dev/null || echo '0')"
            echo "  验证请求: $(grep -c '"type":"verify"' "$app_log" 2>/dev/null || echo '0')"
            echo ""

            echo "HTTP 方法统计:"
            for method in GET POST PUT DELETE PATCH; do
                count=$(grep "\"method\":\"$method\"" "$app_log" 2>/dev/null | wc -l)
                echo "  $method: $count"
            done

            echo ""
            echo "状态码统计:"
            for status in 200 201 400 401 403 404 500 502 503; do
                count=$(grep "\"status\":$status" "$app_log" 2>/dev/null | wc -l)
                if [ "$count" -gt 0 ]; then
                    echo "  $status: $count"
                fi
            done
        fi
        echo ""

        echo "3. 验证码统计"
        echo "-------------------------------------------"
        if [ -f "$LOG_DIR/app/app.log" ]; then
            app_log="$LOG_DIR/app/app.log"
            echo "验证码生成: $(grep -c '"event":"captcha_generated"' "$app_log" 2>/dev/null || echo '0')"
            echo "验证码验证成功: $(grep -c '"event":"captcha_verified".*"result":"success"' "$app_log" 2>/dev/null || echo '0')"
            echo "验证码验证失败: $(grep -c '"event":"captcha_verified".*"result":"failure"' "$app_log" 2>/dev/null || echo '0')"
            echo "验证码阻止: $(grep -c '"event":"captcha_blocked"' "$app_log" 2>/dev/null || echo '0')"
        fi
        echo ""

        echo "4. 性能统计"
        echo "-------------------------------------------"
        if [ -f "$LOG_DIR/app/app.log" ]; then
            app_log="$LOG_DIR/app/app.log"

            echo "响应时间分布:"
            if command -v python3 &> /dev/null; then
                python3 << 'PYEOF'
import re, sys
try:
    with open("$app_log", 'r') as f:
        times = []
        for line in f:
            match = re.search(r'"duration":([0-9.]+)', line)
            if match:
                times.append(float(match.group(1)))

        if times:
            times.sort()
            print(f"  最小值: {min(times):.3f}s")
            print(f"  最大值: {max(times):.3f}s")
            print(f"  平均值: {sum(times)/len(times):.3f}s")
            print(f"  P50: {times[int(len(times)*0.5)]:.3f}s")
            print(f"  P95: {times[int(len(times)*0.95)]:.3f}s")
            print(f"  P99: {times[int(len(times)*0.99)]:.3f}s")
except Exception as e:
    pass
PYEOF
            fi
        fi

    } > "$stats_file"

    log_success "统计报告已保存到: $stats_file"
    cat "$stats_file"
}

analyze_errors() {
    log_info "开始分析错误日志..."

    local error_file="$REPORT_DIR/errors_${TIMESTAMP}.txt"
    local time_range="${1:-1h}"

    {
        echo "==================================="
        echo "HJTPX 错误分析报告"
        echo "生成时间: $(date '+%Y-%m-%d %H:%M:%S')"
        echo "时间范围: $time_range"
        echo "==================================="
        echo ""

        echo "1. 错误类型统计"
        echo "-------------------------------------------"
        if [ -f "$LOG_DIR/app/app.log" ]; then
            app_log="$LOG_DIR/app/app.log"

            echo "错误总数: $(grep -c '"level":"error"' "$app_log" 2>/dev/null || echo '0')"

            echo ""
            echo "按错误类型:"
            for error_type in "database" "redis" "timeout" "validation" "authentication" "authorization" "internal"; do
                count=$(grep "\"error_type\":\"$error_type\"" "$app_log" 2>/dev/null | wc -l)
                if [ "$count" -gt 0 ]; then
                    echo "  $error_type: $count"
                fi
            done

            echo ""
            echo "错误信息 TOP 10:"
            grep '"level":"error"' "$app_log" 2>/dev/null | \
                grep -oP '"message":"[^"]*"' | \
                sort | uniq -c | sort -rn | head -10 | \
                awk '{printf "  %s: %s\n", $2, $1}'
        fi
        echo ""

        echo "2. 最近错误详情"
        echo "-------------------------------------------"
        if [ -f "$LOG_DIR/app/app.log" ]; then
            app_log="$LOG_DIR/app/app.log"
            grep '"level":"error"' "$app_log" 2>/dev/null | tail -20 | while read -r line; do
                echo "$line"
                echo ""
            done
        fi
        echo ""

        echo "3. 错误趋势分析"
        echo "-------------------------------------------"
        if [ -f "$LOG_DIR/app/app.log" ]; then
            app_log="$LOG_DIR/app/app.log"

            echo "按小时统计错误数量:"
            for hour in $(seq 0 23); do
                hour_padded=$(printf "%02d" $hour)
                count=$(grep '"level":"error"' "$app_log" 2>/dev/null | \
                    grep "$(date '+%Y-%m-%d')T$hour_padded" | wc -l)
                if [ "$count" -gt 0 ]; then
                    echo "  $hour:00 - $count errors"
                fi
            done
        fi
        echo ""

        echo "4. 错误建议"
        echo "-------------------------------------------"
        echo "基于错误分析的建议:"
        echo ""

        error_count=$(grep -c '"level":"error"' "$LOG_DIR/app/app.log" 2>/dev/null || echo '0')
        if [ "$error_count" -gt 100 ]; then
            echo "[HIGH] 错误数量较高，建议检查系统状态"
        elif [ "$error_count" -gt 50 ]; then
            echo "[MEDIUM] 错误数量中等，建议关注"
        else
            echo "[LOW] 错误数量在正常范围内"
        fi

        db_errors=$(grep '"error_type":"database"' "$LOG_DIR/app/app.log" 2>/dev/null | wc -l)
        if [ "$db_errors" -gt 10 ]; then
            echo "[WARNING] 数据库错误较多，建议检查数据库连接和查询性能"
        fi

        redis_errors=$(grep '"error_type":"redis"' "$LOG_DIR/app/app.log" 2>/dev/null | wc -l)
        if [ "$redis_errors" -gt 10 ]; then
            echo "[WARNING] Redis错误较多，建议检查Redis连接"
        fi

    } > "$error_file"

    log_success "错误分析报告已保存到: $error_file"
    cat "$error_file"
}

analyze_performance() {
    log_info "开始性能分析..."

    local perf_file="$REPORT_DIR/performance_${TIMESTAMP}.txt"

    {
        echo "==================================="
        echo "HJTPX 性能分析报告"
        echo "生成时间: $(date '+%Y-%m-%d %H:%M:%S')"
        echo "==================================="
        echo ""

        echo "1. 响应时间分析"
        echo "-------------------------------------------"
        if [ -f "$LOG_DIR/app/app.log" ]; then
            app_log="$LOG_DIR/app/app.log"

            if command -v python3 &> /dev/null; then
                python3 << 'PYEOF'
import re, json
from collections import defaultdict

try:
    durations = []
    slow_requests = []

    with open("$app_log", 'r') as f:
        for line in f:
            try:
                log = json.loads(line)
                if 'duration' in log:
                    duration = float(log['duration'])
                    durations.append(duration)

                    if duration > 1.0:
                        slow_requests.append({
                            'timestamp': log.get('timestamp', 'N/A'),
                            'path': log.get('path', 'N/A'),
                            'duration': duration,
                            'status': log.get('status', 'N/A')
                        })
            except:
                pass

    if durations:
        durations.sort()
        print(f"样本数: {len(durations)}")
        print(f"最小值: {min(durations):.3f}s")
        print(f"最大值: {max(durations):.3f}s")
        print(f"平均值: {sum(durations)/len(durations):.3f}s")
        print(f"P50: {durations[int(len(durations)*0.5)]:.3f}s")
        print(f"P90: {durations[int(len(durations)*0.9)]:.3f}s")
        print(f"P95: {durations[int(len(durations)*0.95)]:.3f}s")
        print(f"P99: {durations[int(len(durations)*0.99)]:.3f}s")

        print(f"\n慢请求数量 (>1s): {len(slow_requests)}")
        if slow_requests:
            print("\n最慢的10个请求:")
            slow_requests.sort(key=lambda x: x['duration'], reverse=True)
            for i, req in enumerate(slow_requests[:10], 1):
                print(f"  {i}. {req['timestamp']} - {req['path']} - {req['duration']:.3f}s - status {req['status']}")
except Exception as e:
    print(f"Error: {e}")
PYEOF
            fi
        fi
        echo ""

        echo "2. 吞吐量分析"
        echo "-------------------------------------------"
        if [ -f "$LOG_DIR/app/app.log" ]; then
            app_log="$LOG_DIR/app/app.log"

            echo "按小时统计请求数量:"
            for hour in $(seq 0 23); do
                hour_padded=$(printf "%02d" $hour)
                count=$(grep "$(date '+%Y-%m-%d')T$hour_padded" "$app_log" 2>/dev/null | wc -l)
                if [ "$count" -gt 0 ]; then
                    printf "  %s:00 - %d requests\n" "$hour_padded" "$count"
                fi
            done
        fi
        echo ""

        echo "3. 端点性能分析"
        echo "-------------------------------------------"
        if [ -f "$LOG_DIR/app/app.log" ] && command -v python3 &> /dev/null; then
            python3 << 'PYEOF'
import re, json
from collections import defaultdict

try:
    endpoint_times = defaultdict(list)

    with open("$app_log", 'r') as f:
        for line in f:
            try:
                log = json.loads(line)
                if 'duration' in log and 'path' in log:
                    path = log['path']
                    duration = float(log['duration'])
                    endpoint_times[path].append(duration)
            except:
                pass

    if endpoint_times:
        print("端点平均响应时间 (TOP 10):")
        avg_times = [(path, sum(times)/len(times), len(times))
                     for path, times in endpoint_times.items()]
        avg_times.sort(key=lambda x: x[1], reverse=True)

        for path, avg, count in avg_times[:10]:
            print(f"  {path}: {avg:.3f}s (请求数: {count})")
except Exception as e:
    print(f"Error: {e}")
PYEOF
        fi
        echo ""

        echo "4. 性能建议"
        echo "-------------------------------------------"
        echo "基于性能分析的建议:"
        echo ""

        if [ -f "$LOG_DIR/app/app.log" ] && command -v python3 &> /dev/null; then
            python3 << 'PYEOF'
import re, json
try:
    durations = []
    with open("$app_log", 'r') as f:
        for line in f:
            try:
                log = json.loads(line)
                if 'duration' in log:
                    durations.append(float(log['duration']))
            except:
                pass

    if durations:
        durations.sort()
        p95 = durations[int(len(durations)*0.95)]

        if p95 > 3.0:
            print("[HIGH] P95响应时间较高 (>3s)，建议优化慢查询或增加资源")
        elif p95 > 1.0:
            print("[MEDIUM] P95响应时间中等 (>1s)，建议关注性能趋势")
        else:
            print("[LOW] 响应时间表现良好")
except:
    pass
PYEOF
        fi

    } > "$perf_file"

    log_success "性能分析报告已保存到: $perf_file"
    cat "$perf_file"
}

analyze_security() {
    log_info "开始安全事件分析..."

    local security_file="$REPORT_DIR/security_${TIMESTAMP}.txt"

    {
        echo "==================================="
        echo "HJTPX 安全事件分析报告"
        echo "生成时间: $(date '+%Y-%m-%d %H:%M:%S')"
        echo "==================================="
        echo ""

        echo "1. 安全事件统计"
        echo "-------------------------------------------"
        if [ -f "$LOG_DIR/app/app.log" ]; then
            app_log="$LOG_DIR/app/app.log"

            echo "安全事件总数: $(grep -c '"category":"security"' "$app_log" 2>/dev/null || echo '0')"
            echo ""

            echo "按事件类型:"
            for event_type in "bot_detection" "proxy_detection" "rate_limit" "blacklist" "brute_force" "xss" "sql_injection" "csrf"; do
                count=$(grep "\"event\":\"$event_type\"" "$app_log" 2>/dev/null | wc -l)
                if [ "$count" -gt 0 ]; then
                    echo "  $event_type: $count"
                fi
            done
        fi
        echo ""

        echo "2. 阻止请求统计"
        echo "-------------------------------------------"
        if [ -f "$LOG_DIR/app/app.log" ]; then
            app_log="$LOG_DIR/app/app.log"

            echo "总阻止数: $(grep -c '"action":"blocked"' "$app_log" 2>/dev/null || echo '0')"
            echo "总允许数: $(grep -c '"action":"allowed"' "$app_log" 2>/dev/null || echo '0')"

            echo ""
            echo "阻止原因分布:"
            for reason in "bot" "proxy" "rate_limit" "blacklist" "suspicious" "invalid"; do
                count=$(grep "\"blocked_reason\":\"$reason\"" "$app_log" 2>/dev/null | wc -l)
                if [ "$count" -gt 0 ]; then
                    echo "  $reason: $count"
                fi
            done
        fi
        echo ""

        echo "3. IP 地址分析"
        echo "-------------------------------------------"
        if [ -f "$LOG_DIR/app/app.log" ] && command -v python3 &> /dev/null; then
            python3 << 'PYEOF'
import re, json
from collections import Counter

try:
    ip_counter = Counter()

    with open("$app_log", 'r') as f:
        for line in f:
            try:
                log = json.loads(line)
                if 'ip' in log:
                    ip_counter[log['ip']] += 1
            except:
                pass

    if ip_counter:
        print("请求最多的 IP 地址 (TOP 10):")
        for ip, count in ip_counter.most_common(10):
            print(f"  {ip}: {count} requests")
except Exception as e:
    print(f"Error: {e}")
PYEOF
        fi
        echo ""

        echo "4. 安全建议"
        echo "-------------------------------------------"
        echo "基于安全分析的建议:"
        echo ""

        bot_count=$(grep '"event":"bot_detection"' "$LOG_DIR/app/app.log" 2>/dev/null | wc -l)
        if [ "$bot_count" -gt 100 ]; then
            echo "[HIGH] 检测到大量机器人流量，建议检查IP黑名单"
        fi

        brute_count=$(grep '"event":"brute_force"' "$LOG_DIR/app/app.log" 2>/dev/null | wc -l)
        if [ "$brute_count" -gt 10 ]; then
            echo "[WARNING] 检测到暴力破解尝试，建议增强登录限制"
        fi

        rate_limit_count=$(grep '"event":"rate_limit"' "$LOG_DIR/app/app.log" 2>/dev/null | wc -l)
        if [ "$rate_limit_count" -gt 100 ]; then
            echo "[INFO] 限流触发较多，可能是正常的高流量或攻击"
        fi

    } > "$security_file"

    log_success "安全分析报告已保存到: $security_file"
    cat "$security_file"
}

generate_full_report() {
    log_info "生成完整日志分析报告..."

    local report_file="$REPORT_DIR/full_report_${TIMESTAMP}.html"

    {
        echo "<!DOCTYPE html>"
        echo "<html lang='zh-CN'>"
        echo "<head>"
        echo "  <meta charset='UTF-8'>"
        echo "  <meta name='viewport' content='width=device-width, initial-scale=1.0'>"
        echo "  <title>HJTPX 日志分析报告</title>"
        echo "  <style>"
        echo "    body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }"
        echo "    .container { max-width: 1200px; margin: 0 auto; background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }"
        echo "    h1 { color: #1976d2; border-bottom: 2px solid #1976d2; padding-bottom: 10px; }"
        echo "    h2 { color: #424242; margin-top: 30px; }"
        echo "    .metric { display: inline-block; padding: 15px; margin: 10px; background: #e3f2fd; border-radius: 5px; min-width: 150px; }"
        echo "    .metric-label { font-size: 12px; color: #666; }"
        echo "    .metric-value { font-size: 24px; font-weight: bold; color: #1976d2; }"
        echo "    table { width: 100%; border-collapse: collapse; margin: 20px 0; }"
        echo "    th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }"
        echo "    th { background-color: #1976d2; color: white; }"
        echo "    .warning { color: #ff9800; }"
        echo "    .error { color: #f44336; }"
        echo "    .success { color: #4caf50; }"
        echo "    .footer { margin-top: 40px; padding-top: 20px; border-top: 1px solid #ddd; color: #666; }"
        echo "  </style>"
        echo "</head>"
        echo "<body>"
        echo "  <div class='container'>"
        echo "    <h1>HJTPX 日志分析报告</h1>"
        echo "    <p>生成时间: $(date '+%Y-%m-%d %H:%M:%S')</p>"
        echo "    <p>日志目录: $LOG_DIR</p>"

        echo "    <h2>1. 系统概览</h2>"
        echo "    <div class='metric'>"
        echo "      <div class='metric-label'>日志文件</div>"
        echo "      <div class='metric-value'>$(find "$LOG_DIR" -type f 2>/dev/null | wc -l)</div>"
        echo "    </div>"
        echo "    <div class='metric'>"
        echo "      <div class='metric-label'>总大小</div>"
        echo "      <div class='metric-value'>$(du -sh "$LOG_DIR" 2>/dev/null | cut -f1)</div>"
        echo "    </div>"

        if [ -f "$LOG_DIR/app/app.log" ]; then
            echo "    <h2>2. 应用统计</h2>"
            echo "    <table>"
            echo "      <tr><th>指标</th><th>数量</th></tr>"
            echo "      <tr><td>INFO 日志</td><td>$(grep -c '"level":"info"' "$LOG_DIR/app/app.log" 2>/dev/null || echo '0')</td></tr>"
            echo "      <tr><td>WARNING 日志</td><td class='warning'>$(grep -c '"level":"warn"' "$LOG_DIR/app/app.log" 2>/dev/null || echo '0')</td></tr>"
            echo "      <tr><td>ERROR 日志</td><td class='error'>$(grep -c '"level":"error"' "$LOG_DIR/app/app.log" 2>/dev/null || echo '0')</td></tr>"
            echo "    </table>"
        fi

        echo "    <div class='footer'>"
        echo "      <p>此报告由 HJTPX 日志分析工具自动生成</p>"
        echo "    </div>"
        echo "  </div>"
        echo "</body>"
        echo "</html>"

    } > "$report_file"

    log_success "完整报告已保存到: $report_file"
    echo "报告位置: $report_file"
}

watch_logs() {
    local service="${1:-app}"
    log_info "开始实时监控 $service 日志 (Ctrl+C 退出)..."

    if [ -f "$LOG_DIR/$service/$service.log" ]; then
        tail -f "$LOG_DIR/$service/$service.log" | while read -r line; do
            if echo "$line" | grep -q '"level":"error"'; then
                echo -e "${RED}$line${NC}"
            elif echo "$line" | grep -q '"level":"warn"'; then
                echo -e "${YELLOW}$line${NC}"
            else
                echo "$line"
            fi
        done
    else
        log_error "日志文件不存在: $LOG_DIR/$service/$service.log"
        exit 1
    fi
}

export_logs() {
    local export_format="${1:-json}"
    local export_file="$REPORT_DIR/logs_export_${TIMESTAMP}.$export_format"

    log_info "导出日志数据到: $export_file"

    case "$export_format" in
        json)
            find "$LOG_DIR" -name "*.log" -type f -exec cat {} \; > "$export_file"
            ;;
        csv)
            if command -v python3 &> /dev/null; then
                python3 << 'PYEOF'
import json, csv, sys

try:
    with open("$export_file", 'w', newline='') as f:
        writer = csv.writer(f)
        writer.writerow(['timestamp', 'level', 'message', 'service'])

        for log_file in []:  # Add log files
            with open(log_file, 'r') as lf:
                for line in lf:
                    try:
                        log = json.loads(line)
                        writer.writerow([
                            log.get('timestamp', ''),
                            log.get('level', ''),
                            log.get('message', ''),
                            log.get('service', '')
                        ])
                    except:
                        pass
except Exception as e:
    print(f"Error: {e}", file=sys.stderr)
PYEOF
            fi
            ;;
        *)
            log_error "不支持的格式: $export_format"
            exit 1
            ;;
    esac

    log_success "日志导出完成: $export_file"
}

main() {
    local command="${1:-help}"
    shift || true

    case "$command" in
        stats)
            analyze_stats
            ;;
        errors)
            local time_range="${1:-1h}"
            analyze_errors "$time_range"
            ;;
        performance)
            analyze_performance
            ;;
        security)
            analyze_security
            ;;
        report)
            local format="${1:-html}"
            generate_full_report
            ;;
        watch)
            local service="${1:-app}"
            watch_logs "$service"
            ;;
        export)
            local format="${1:-json}"
            export_logs "$format"
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            log_error "未知命令: $command"
            show_help
            exit 1
            ;;
    esac
}

main "$@"
