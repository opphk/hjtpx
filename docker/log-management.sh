#!/bin/sh
# =============================================================================
# 日志管理脚本 - Docker容器日志轮转和管理
# =============================================================================

set -e

CONTAINER_NAME="${1:-hjtpx_app}"
MAX_SIZE="${2:-10m}"
MAX_FILES="${3:-5}"

log_message() {
    echo "[$(date -u +'%Y-%m-%d %H:%M:%S')] $1"
}

log_error() {
    echo "[$(date -u +'%Y-%m-%d %H:%M:%S')] ERROR: $1" >&2
}

# 检查Docker是否可用
check_docker() {
    if ! command -v docker > /dev/null 2>&1; then
        log_error "Docker command not found"
        return 1
    fi

    if ! docker info > /dev/null 2>&1; then
        log_error "Docker daemon is not running"
        return 1
    fi

    return 0
}

# 查看容器日志
view_logs() {
    local lines="${1:-100}"
    log_message "Viewing last $lines lines of logs for container: $CONTAINER_NAME"
    docker logs --tail "$lines" --timestamps "$CONTAINER_NAME" 2>&1 | less -r
}

# 实时跟踪日志
follow_logs() {
    log_message "Following logs for container: $CONTAINER_NAME (Press Ctrl+C to stop)"
    docker logs -f --tail 100 --timestamps "$CONTAINER_NAME" 2>&1
}

# 清理旧日志
clean_logs() {
    log_message "Cleaning old logs for container: $CONTAINER_NAME"

    # 获取容器日志文件路径
    local log_path=$(docker inspect --format='{{.LogPath}}' "$CONTAINER_NAME" 2>/dev/null)

    if [ -z "$log_path" ]; then
        log_error "Could not find log path for container: $CONTAINER_NAME"
        return 1
    fi

    log_message "Container log path: $log_path"

    # 检查日志文件大小
    local size=$(du -h "$log_path" 2>/dev/null | cut -f1 || echo "unknown")
    log_message "Current log size: $size"

    # 如果日志文件超过限制，执行轮转
    local current_size=$(stat -c%s "$log_path" 2>/dev/null || echo 0)
    local max_size_bytes=$(echo "$MAX_SIZE" | numfmt --from=iec 2>/dev/null || echo 10485760)

    if [ "$current_size" -gt "$max_size_bytes" ]; then
        log_message "Log file exceeds maximum size, rotating..."

        # 复制当前日志到轮转文件
        local timestamp=$(date -u +'%Y%m%d_%H%M%S')
        local rotated_log="${log_path}.${timestamp}"
        cp "$log_path" "$rotated_log"

        # 清空当前日志文件
        : > "$log_path"

        log_message "Log rotated to: $rotated_log"

        # 删除旧轮转日志
        find "$(dirname "$log_path")" -name "$(basename "$log_path").*" -mtime +7 -delete
        log_message "Old rotated logs cleaned up"
    else
        log_message "Log file size is within limits"
    fi
}

# 显示日志统计信息
show_stats() {
    log_message "Log statistics for container: $CONTAINER_NAME"

    local log_path=$(docker inspect --format='{{.LogPath}}' "$CONTAINER_NAME" 2>/dev/null)

    if [ -z "$log_path" ]; then
        log_error "Could not find log path for container: $CONTAINER_NAME"
        return 1
    fi

    log_message "Log path: $log_path"
    log_message "Log size: $(du -h "$log_path" 2>/dev/null | cut -f1 || echo 'N/A')"
    log_message "Log lines: $(wc -l < "$log_path" 2>/dev/null || echo 'N/A')"
    log_message "Last modified: $(stat -c%y "$log_path" 2>/dev/null || echo 'N/A')"

    # 显示日志文件列表
    log_message "Rotated logs:"
    ls -lh "$(dirname "$log_path")"/"$(basename "$log_path")".* 2>/dev/null || log_message "No rotated logs found"
}

# 配置日志轮转
configure_logging() {
    log_message "Configuring log rotation for container: $CONTAINER_NAME"

    # 更新容器的日志配置
    docker update \
        --log-driver json-file \
        --log-opt max-size="$MAX_SIZE" \
        --log-opt max-file="$MAX_FILES" \
        --log-opt compress=true \
        "$CONTAINER_NAME"

    log_message "Log configuration updated"
}

# 导出日志
export_logs() {
    local output_file="${1:-logs_export_$(date -u +'%Y%m%d_%H%M%S').log}"
    log_message "Exporting logs to: $output_file"

    docker logs "$CONTAINER_NAME" > "$output_file" 2>&1
    log_message "Logs exported successfully to: $output_file"
}

# 主流程
main() {
    local action="${1:-help}"

    case "$action" in
        view)
            view_logs "${2:-100}"
            ;;
        follow)
            follow_logs
            ;;
        clean)
            clean_logs
            ;;
        stats)
            show_stats
            ;;
        configure)
            configure_logging
            ;;
        export)
            export_logs "${2}"
            ;;
        help|--help|-h)
            echo "Usage: $0 <command> [options]"
            echo ""
            echo "Commands:"
            echo "  view [lines]     View last N lines of logs (default: 100)"
            echo "  follow           Follow logs in real-time"
            echo "  clean            Clean/rotate old logs"
            echo "  stats            Show log statistics"
            echo "  configure        Configure log rotation"
            echo "  export [file]    Export logs to file"
            echo "  help             Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0 view 200"
            echo "  $0 follow"
            echo "  $0 clean"
            echo "  $0 stats"
            echo "  $0 export my_logs.log"
            ;;
        *)
            log_error "Unknown command: $action"
            echo "Run '$0 help' for usage information"
            exit 1
            ;;
    esac
}

main "$@"