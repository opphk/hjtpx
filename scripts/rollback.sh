#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

BACKUP_DIR="${PROJECT_ROOT}/backups"
LOG_DIR="${PROJECT_ROOT}/logs"
ROLLBACK_LOG_FILE="${LOG_DIR}/rollback-$(date +%Y%m%d-%H%M%S).log"

log() {
    local level="$1"
    local message="$2"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[${timestamp}] [${level}] ${message}" | tee -a "$ROLLBACK_LOG_FILE"
}

log_separator() {
    echo "========================================" | tee -a "$ROLLBACK_LOG_FILE"
}

init_env() {
    mkdir -p "$LOG_DIR"
    
    local compose_cmd=""
    if command -v docker-compose &> /dev/null; then
        compose_cmd="docker-compose"
    elif docker compose version &> /dev/null; then
        compose_cmd="docker compose"
    else
        log "ERROR" "Docker Compose 未安装"
        exit 1
    fi
    export COMPOSE_CMD="$compose_cmd"
}

list_backups() {
    log_separator
    log "INFO" "===== 可用的备份列表 ====="
    
    if [ ! -d "$BACKUP_DIR" ] || [ -z "$(ls -A "$BACKUP_DIR" 2>/dev/null)" ]; then
        log "WARN" "没有找到任何备份"
        return 1
    fi
    
    echo ""
    printf "%-30s %s\n" "备份名称" "创建时间"
    printf "%-30s %s\n" "--------" "--------"
    
    for backup in "$BACKUP_DIR"/backup-*; do
        if [ -d "$backup" ]; then
            local backup_name=$(basename "$backup")
            local backup_time=$(stat -c %y "$backup" 2>/dev/null | cut -d' ' -f1,2 | cut -d'.' -f1)
            printf "%-30s %s\n" "$backup_name" "$backup_time"
        fi
    done
    
    echo ""
    local latest=$(cat "${BACKUP_DIR}/latest_backup" 2>/dev/null)
    if [ -n "$latest" ] && [ -d "$latest" ]; then
        log "INFO" "最新备份: $(basename "$latest")"
    fi
    
    return 0
}

rollback_to_backup() {
    local backup_path="$1"
    
    if [ ! -d "$backup_path" ]; then
        log "ERROR" "备份不存在: $backup_path"
        return 1
    fi
    
    log_separator
    log "INFO" "===== 开始回滚 ====="
    log "INFO" "备份: $(basename "$backup_path")"
    
    log "INFO" "1. 停止当前服务..."
    $COMPOSE_CMD down --remove-orphans 2>/dev/null || true
    log "INFO" "  ✓ 服务已停止"
    
    log "INFO" "2. 恢复配置文件..."
    if [ -f "${backup_path}/docker-compose.yml" ]; then
        cp "${backup_path}/docker-compose.yml" ./
        log "INFO" "  ✓ docker-compose.yml 已恢复"
    fi
    
    if [ -f "${backup_path}/backend.tar.gz" ]; then
        tar -xzf "${backup_path}/backend.tar.gz" -C . 2>/dev/null || true
        log "INFO" "  ✓ backend 已恢复"
    fi
    
    if [ -d "${backup_path}/conf.d" ]; then
        cp -r "${backup_path}/conf.d" nginx/ 2>/dev/null || true
        log "INFO" "  ✓ nginx配置已恢复"
    fi
    
    log "INFO" "3. 启动服务..."
    $COMPOSE_CMD up -d
    log "INFO" "  ✓ 服务已启动"
    
    log "INFO" "4. 等待服务就绪..."
    sleep 10
    
    log "INFO" "5. 验证服务状态..."
    if curl -sf --connect-timeout 5 "http://localhost:8080/health" > /dev/null 2>&1; then
        log "INFO" "  ✓ 后端服务健康"
    else
        log "WARN" "  ⚠ 后端服务可能未正常运行，请检查"
    fi
    
    log_separator
    log "INFO" "===== 回滚完成 ====="
    log "INFO" "备份路径: $backup_path"
    log ""
    log "INFO" "查看服务状态: $COMPOSE_CMD ps"
    log "INFO" "查看日志: $COMPOSE_CMD logs -f"
    
    return 0
}

rollback_to_latest() {
    local latest_backup=$(cat "${BACKUP_DIR}/latest_backup" 2>/dev/null)
    
    if [ -z "$latest_backup" ] || [ ! -d "$latest_backup" ]; then
        log "ERROR" "未找到最新备份"
        return 1
    fi
    
    log "INFO" "将回滚到最新备份: $(basename "$latest_backup")"
    rollback_to_backup "$latest_backup"
}

main() {
    init_env
    
    log_separator
    log "INFO" "===== HJTPX 回滚脚本 ====="
    
    if [ $# -eq 0 ]; then
        list_backups
        
        echo ""
        read -p "请选择回滚方式:
1) 回滚到最新备份
2) 指定备份路径
3) 退出

请输入选项 [1-3]: " choice
        
        case "$choice" in
            1)
                rollback_to_latest
                ;;
            2)
                read -p "请输入备份路径: " backup_path
                rollback_to_backup "$backup_path"
                ;;
            3)
                log "INFO" "退出回滚"
                exit 0
                ;;
            *)
                log "ERROR" "无效的选项"
                exit 1
                ;;
        esac
    else
        case "$1" in
            -l|--latest)
                rollback_to_latest
                ;;
            -l*)
                local backup_path="${1#-l}"
                [ -z "$backup_path" ] && backup_path="$2"
                rollback_to_backup "$backup_path"
                ;;
            -h|--help)
                echo "用法: $0 [选项] [备份路径]"
                echo ""
                echo "选项:"
                echo "  -l, --latest    回滚到最新备份"
                echo "  -h, --help      显示帮助信息"
                echo ""
                echo "示例:"
                echo "  $0                                      # 交互式选择"
                echo "  $0 --latest                             # 回滚到最新"
                echo "  $0 /path/to/backup                     # 回滚到指定备份"
                exit 0
                ;;
            *)
                rollback_to_backup "$1"
                ;;
        esac
    fi
}

main "$@"
