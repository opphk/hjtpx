#!/bin/sh
# =============================================================================
# 日志收集脚本 - 收集所有Docker容器日志
# =============================================================================

set -e

COLLECTION_DIR="${1:-/var/log/hjtpx/collected}"
TIMESTAMP=$(date -u +'%Y%m%d_%H%M%S')
OUTPUT_DIR="${COLLECTION_DIR}/${TIMESTAMP}"
LOG_FILE="${OUTPUT_DIR}/collection.log"

log_message() {
    echo "[$(date -u +'%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

log_error() {
    echo "[$(date -u +'%Y-%m-%d %H:%M:%S')] ERROR: $1" | tee -a "$LOG_FILE" >&2
}

# 创建输出目录
setup() {
    log_message "Setting up log collection directory: $OUTPUT_DIR"
    mkdir -p "$OUTPUT_DIR"

    # 检查Docker是否可用
    if ! docker info > /dev/null 2>&1; then
        log_error "Docker daemon is not running or not accessible"
        return 1
    fi

    return 0
}

# 收集所有容器日志
collect_container_logs() {
    log_message "Collecting logs from all containers..."

    local containers=$(docker ps --format '{{.Names}}')

    for container in $containers; do
        log_message "Collecting logs from container: $container"
        local container_log="${OUTPUT_DIR}/${container}.log"

        # 获取容器日志
        if docker logs "$container" > "$container_log" 2>&1; then
            local size=$(du -h "$container_log" | cut -f1)
            log_message "Collected logs from $container (size: $size)"
        else
            log_error "Failed to collect logs from container: $container"
        fi

        # 获取容器信息
        docker inspect "$container" > "${OUTPUT_DIR}/${container}_info.json" 2>&1
    done
}

# 收集Docker系统信息
collect_system_info() {
    log_message "Collecting Docker system information..."

    # Docker版本信息
    docker version > "${OUTPUT_DIR}/docker_version.log" 2>&1

    # Docker系统信息
    docker info > "${OUTPUT_DIR}/docker_info.log" 2>&1

    # 容器列表
    docker ps -a > "${OUTPUT_DIR}/containers_list.log" 2>&1

    # 镜像列表
    docker images > "${OUTPUT_DIR}/images_list.log" 2>&1

    # 网络列表
    docker network ls > "${OUTPUT_DIR}/networks_list.log" 2>&1

    # 卷列表
    docker volume ls > "${OUTPUT_DIR}/volumes_list.log" 2>&1

    log_message "Docker system information collected"
}

# 收集容器统计信息
collect_stats() {
    log_message "Collecting container statistics..."

    local containers=$(docker ps --format '{{.Names}}')

    for container in $containers; do
        local stats_log="${OUTPUT_DIR}/${container}_stats.log"

        # 获取容器统计信息
        docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}\t{{.PIDs}}" "$container" > "$stats_log" 2>&1

        log_message "Collected stats for container: $container"
    done
}

# 创建收集摘要
create_summary() {
    log_message "Creating collection summary..."

    cat > "${OUTPUT_DIR}/summary.txt" <<EOF
Docker Logs Collection Summary
==============================
Collection Time: $(date -u +'%Y-%m-%d %H:%M:%S')
Collection Directory: $OUTPUT_DIR

Containers Collected:
$(docker ps --format '{{.Names}} ({{.Image}}) - {{.Status}}' | nl)

Files Created:
$(find "$OUTPUT_DIR" -type f -exec ls -lh {} \; | awk '{print $9 " - " $5}')

Total Size:
$(du -sh "$OUTPUT_DIR" | cut -f1)

Log Collection completed successfully at $(date -u +'%Y-%m-%d %H:%M:%S')
EOF

    log_message "Summary created"
}

# 清理旧收集
cleanup_old_collections() {
    log_message "Cleaning up old collections (keeping last 10)..."

    # 获取所有收集目录，按时间排序
    local collections=$(ls -t "$COLLECTION_DIR" 2>/dev/null | tail -n +11)

    for collection in $collections; do
        log_message "Removing old collection: $collection"
        rm -rf "${COLLECTION_DIR}/${collection}"
    done

    log_message "Cleanup completed"
}

# 主流程
main() {
    log_message "Starting Docker log collection..."

    if ! setup; then
        log_error "Setup failed, exiting"
        exit 1
    fi

    collect_container_logs
    collect_system_info
    collect_stats
    create_summary
    cleanup_old_collections

    log_message "Log collection completed successfully"
    log_message "Logs are available at: $OUTPUT_DIR"
    log_message "Summary available at: ${OUTPUT_DIR}/summary.txt"

    # 显示摘要
    echo ""
    cat "${OUTPUT_DIR}/summary.txt"
}

main "$@"