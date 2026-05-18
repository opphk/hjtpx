#!/bin/bash
set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$DIR/.." && pwd)"
cd "$PROJECT_ROOT"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

BACKUP_DIR="${BACKUP_DIR:-$PROJECT_ROOT/backups}"
ROLLBACK_DIR="${ROLLBACK_DIR:-$PROJECT_ROOT/.rollback}"
KEEP_VERSIONS="${KEEP_VERSIONS:-5}"

get_docker_compose_cmd() {
    if command -v docker-compose &> /dev/null; then
        echo "docker-compose"
    elif docker compose version &> /dev/null; then
        echo "docker compose"
    else
        echo "docker-compose"
    fi
}

DOCKER_COMPOSE=$(get_docker_compose_cmd)

list_backups() {
    log_info "可用备份版本列表:"

    if [ ! -d "$BACKUP_DIR" ]; then
        log_warning "备份目录不存在: $BACKUP_DIR"
        return 1
    fi

    local count=0
    echo ""
    printf "%-5s %-25s %-15s %-20s\n" "序号" "版本" "大小" "创建时间"
    echo "------------------------------------------------------------"

    for backup in $(ls -t "$BACKUP_DIR"/rollback_*.tar.gz 2>/dev/null | head -20); do
        count=$((count + 1))
        size=$(du -h "$backup" | cut -f1)
        time=$(stat -c %y "$backup" 2>/dev/null | cut -d' ' -f1,2 | cut -d'.' -f1 || ls -l --time-style=full-iso "$backup" | awk '{print $6, $7}' | cut -d'.' -f1)
        basename_backup=$(basename "$backup" .tar.gz)
        printf "%-5s %-25s %-15s %-20s\n" "$count" "$basename_backup" "$size" "$time"
    done

    if [ $count -eq 0 ]; then
        log_warning "没有找到备份版本"
        return 1
    fi

    echo ""
    return 0
}

create_backup() {
    local backup_name="rollback_$(date +%Y%m%d_%H%M%S)"
    local backup_path="$BACKUP_DIR/${backup_name}.tar.gz"

    log_info "创建当前版本备份: $backup_name"

    mkdir -p "$BACKUP_DIR"
    mkdir -p "$ROLLBACK_DIR"

    local temp_dir="$ROLLBACK_DIR/$backup_name"
    rm -rf "$temp_dir"
    mkdir -p "$temp_dir"

    log_info "备份Docker镜像..."
    if $DOCKER_COMPOSE images -q 2>/dev/null | head -1 | xargs -I {} docker save -o "$temp_dir/images.tar" {} 2>/dev/null; then
        log_success "Docker镜像备份完成"
    else
        log_warning "Docker镜像备份失败或无镜像，跳过"
    fi

    log_info "备份docker-compose配置..."
    cp docker-compose.yml "$temp_dir/" 2>/dev/null || true
    cp .env "$temp_dir/.env.backup" 2>/dev/null || true

    log_info "备份数据库..."
    if $DOCKER_COMPOSE exec -T postgres pg_dump -U postgres hjtpx_db > "$temp_dir/database.sql" 2>/dev/null; then
        log_success "数据库备份完成"
    else
        log_warning "数据库备份失败，跳过"
    fi

    log_info "备份Redis数据..."
    if $DOCKER_COMPOSE exec -T redis redis-cli SAVE 2>/dev/null; then
        $DOCKER_COMPOSE cp redis:/data/dump.rdb "$temp_dir/redis.rdb" 2>/dev/null && log_success "Redis备份完成" || log_warning "Redis备份失败"
    else
        log_warning "Redis备份失败，跳过"
    fi

    log_info "备份配置文件..."
    tar -czf "$backup_path" -C "$temp_dir" . 2>/dev/null

    if [ -f "$backup_path" ]; then
        local size=$(du -h "$backup_path" | cut -f1)
        log_success "备份创建成功: $backup_path (大小: $size)"

        cleanup_old_backups
        echo "$backup_name" > "$ROLLBACK_DIR/current_backup.txt"
        echo "$backup_path" > "$ROLLBACK_DIR/current_backup_path.txt"

        return 0
    else
        log_error "备份创建失败"
        return 1
    fi
}

restore_backup() {
    local backup_path="$1"

    if [ -z "$backup_path" ]; then
        log_error "未指定备份路径"
        list_backups
        return 1
    fi

    if [ ! -f "$backup_path" ]; then
        log_error "备份文件不存在: $backup_path"
        return 1
    fi

    log_warning "即将恢复到备份: $(basename "$backup_path")"
    log_warning "当前服务将会停止，是否继续? (y/N)"

    read -r response
    if [ "$response" != "y" ] && [ "$response" != "Y" ]; then
        log_info "回滚操作已取消"
        return 0
    fi

    log_info "停止当前服务..."
    $DOCKER_COMPOSE down --remove-orphans 2>/dev/null || true

    local temp_dir="$ROLLBACK_DIR/restore_temp_$$"
    rm -rf "$temp_dir"
    mkdir -p "$temp_dir"

    log_info "解压备份文件..."
    if tar -xzf "$backup_path" -C "$temp_dir"; then
        log_success "备份文件解压完成"
    else
        log_error "备份文件解压失败"
        rm -rf "$temp_dir"
        return 1
    fi

    log_info "恢复Docker镜像..."
    if [ -f "$temp_dir/images.tar" ]; then
        docker load -i "$temp_dir/images.tar" 2>/dev/null && log_success "Docker镜像恢复完成" || log_warning "Docker镜像恢复失败"
    fi

    log_info "恢复docker-compose配置..."
    cp "$temp_dir/docker-compose.yml" . 2>/dev/null || true
    [ -f "$temp_dir/.env.backup" ] && cp "$temp_dir/.env.backup" .env 2>/dev/null || true

    log_info "恢复数据库..."
    if [ -f "$temp_dir/database.sql" ]; then
        $DOCKER_COMPOSE up -d postgres 2>/dev/null
        sleep 5
        if $DOCKER_COMPOSE exec -T postgres psql -U postgres -d hjtpx_db < "$temp_dir/database.sql" 2>/dev/null; then
            log_success "数据库恢复完成"
        else
            log_warning "数据库恢复失败"
        fi
    fi

    log_info "恢复Redis数据..."
    if [ -f "$temp_dir/redis.rdb" ]; then
        $DOCKER_COMPOSE up -d redis 2>/dev/null
        sleep 3
        $DOCKER_COMPOSE cp "$temp_dir/redis.rdb" redis:/data/dump.rdb 2>/dev/null && log_success "Redis数据恢复完成" || log_warning "Redis数据恢复失败"
    fi

    rm -rf "$temp_dir"

    log_info "启动服务..."
    $DOCKER_COMPOSE up -d

    log_success "回滚完成!"
    return 0
}

cleanup_old_backups() {
    log_info "清理旧备份，保留最近 $KEEP_VERSIONS 个版本..."

    local count=$(ls -1 "$BACKUP_DIR"/rollback_*.tar.gz 2>/dev/null | wc -l)

    if [ $count -gt $KEEP_VERSIONS ]; then
        local to_delete=$(ls -1t "$BACKUP_DIR"/rollback_*.tar.gz 2>/dev/null | tail -$((count - KEEP_VERSIONS)))
        echo "$to_delete" | xargs rm -f 2>/dev/null && log_success "旧备份清理完成" || log_warning "清理旧备份时出现错误"
    else
        log_info "备份数量在保留范围内，无需清理"
    fi
}

quick_rollback() {
    local backup_path=$(cat "$ROLLBACK_DIR/current_backup_path.txt" 2>/dev/null)

    if [ -z "$backup_path" ] || [ ! -f "$backup_path" ]; then
        log_error "没有找到可用的备份"
        return 1
    fi

    log_info "执行快速回滚到: $(basename "$backup_path")"
    restore_backup "$backup_path"
}

show_status() {
    log_info "回滚系统状态:"

    if [ -f "$ROLLBACK_DIR/current_backup_path.txt" ]; then
        local current_backup=$(cat "$ROLLBACK_DIR/current_backup_path.txt")
        if [ -f "$current_backup" ]; then
            log_success "当前备份: $(basename "$current_backup")"
            log_info "备份大小: $(du -h "$current_backup" | cut -f1)"
            log_info "备份时间: $(stat -c %y "$current_backup" 2>/dev/null | cut -d' ' -f1,2 | cut -d'.' -f1 || ls -l --time-style=full-iso "$current_backup" | awk '{print $6, $7}' | cut -d'.' -f1)"
        else
            log_warning "备份文件已不存在: $current_backup"
        fi
    else
        log_warning "没有当前备份记录"
    fi

    local total_backups=$(ls -1 "$BACKUP_DIR"/rollback_*.tar.gz 2>/dev/null | wc -l)
    log_info "总备份数量: $total_backups"
}

usage() {
    cat << EOF
HJTPX 回滚脚本 v2.0

用法: $0 [命令] [参数]

命令:
    create              创建当前版本的备份
    restore <备份文件>   恢复到指定备份
    quick               快速回滚到上一个备份
    list                列出所有可用备份
    status              显示回滚系统状态
    cleanup             清理旧备份
    help                显示帮助信息

示例:
    $0 create                    # 创建备份
    $0 restore backups/rollback_20260518_120000.tar.gz
    $0 quick                     # 快速回滚
    $0 list                      # 查看备份列表
    $0 status                    # 查看状态

环境变量:
    BACKUP_DIR      备份存储目录 (默认: ./backups)
    ROLLBACK_DIR    回滚元数据目录 (默认: ./.rollback)
    KEEP_VERSIONS   保留备份版本数 (默认: 5)

EOF
}

main() {
    if [ $# -eq 0 ]; then
        usage
        exit 0
    fi

    case "$1" in
        create)
            create_backup
            ;;
        restore)
            if [ -z "$2" ]; then
                log_error "请指定备份文件路径"
                list_backups
                exit 1
            fi
            restore_backup "$2"
            ;;
        quick)
            quick_rollback
            ;;
        list)
            list_backups
            ;;
        status)
            show_status
            ;;
        cleanup)
            cleanup_old_backups
            ;;
        help|--help|-h)
            usage
            ;;
        *)
            log_error "未知命令: $1"
            usage
            exit 1
            ;;
    esac
}

main "$@"
