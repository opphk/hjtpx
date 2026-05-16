#!/bin/sh
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

BACKUP_DIR="${BACKUP_DIR:-./backups}"
RETENTION_DAYS="${RETENTION_DAYS:-30}"
COMPRESSION_LEVEL="${COMPRESSION_LEVEL:-6}"
DB_NAME="${POSTGRES_DB:-hjtpx_db}"
DB_USER="${POSTGRES_USER:-postgres}"
DB_PASS="${POSTGRES_PASSWORD:-postgres}"
S3_BUCKET="${S3_BUCKET:-}"
S3_PREFIX="${S3_PREFIX:-hjtpx-backups}"
REMOTE_HOST="${REMOTE_HOST:-}"
REMOTE_PATH="${REMOTE_PATH:-}"

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="backup_${TIMESTAMP}"
BACKUP_GZ="$BACKUP_DIR/${BACKUP_FILE}.sql.gz"

log_info() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] INFO: $1"
}

log_error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $1" >&2
}

check_dependencies() {
    log_info "检查依赖项..."

    if ! command -v gzip > /dev/null 2>&1; then
        log_error "gzip未安装"
        return 1
    fi

    if command -v docker-compose > /dev/null 2>&1; then
        DOCKER_COMPOSE="docker-compose"
    elif command -v docker > /dev/null 2>&1; then
        DOCKER_COMPOSE="docker compose"
    else
        log_error "Docker未安装"
        return 1
    fi

    log_info "使用Docker Compose: $DOCKER_COMPOSE"
    return 0
}

create_backup() {
    log_info "开始备份数据库: $DB_NAME"
    log_info "备份文件: $BACKUP_GZ"

    mkdir -p "$BACKUP_DIR"

    if $DOCKER_COMPOSE exec -T postgres pg_dump -U "$DB_USER" "$DB_NAME" 2>/dev/null | gzip -"$COMPRESSION_LEVEL" > "$BACKUP_GZ"; then
        log_info "✓ 数据库备份成功"
    else
        log_error "数据库备份失败"
        return 1
    fi

    BACKUP_SIZE=$(du -h "$BACKUP_GZ" | cut -f1)
    log_info "备份文件大小: $BACKUP_SIZE"
}

backup_redis() {
    log_info "开始备份Redis..."

    REDIS_BACKUP="$BACKUP_DIR/redis_${TIMESTAMP}.rdb"

    if $DOCKER_COMPOSE exec -T redis redis-cli -a "$DB_PASS" SAVE 2>/dev/null; then
        $DOCKER_COMPOSE cp "redis:/data/dump.rdb" "$REDIS_BACKUP" 2>/dev/null
        if [ -f "$REDIS_BACKUP" ]; then
            gzip -"$COMPRESSION_LEVEL" "$REDIS_BACKUP"
            log_info "✓ Redis备份成功"
        fi
    else
        log_info "Redis备份跳过(可能无密码)"
    fi
}

cleanup_old_backups() {
    log_info "清理超过 $RETENTION_DAYS 天的旧备份..."

    find "$BACKUP_DIR" -name "backup_*.sql.gz" -mtime +"$RETENTION_DAYS" -delete
    find "$BACKUP_DIR" -name "redis_*.rdb.gz" -mtime +"$RETENTION_DAYS" -delete

    log_info "✓ 旧备份清理完成"
}

upload_to_s3() {
    if [ -z "$S3_BUCKET" ]; then
        log_info "S3_BUCKET未配置,跳过云存储上传"
        return 0
    fi

    log_info "上传备份到S3: $S3_BUCKET/$S3_PREFIX/"

    if command -v aws > /dev/null 2>&1; then
        aws s3 cp "$BACKUP_GZ" "s3://$S3_BUCKET/$S3_PREFIX/" 2>/dev/null
        log_info "✓ 上传到S3成功"
    elif command -v rclone > /dev/null 2>&1; then
        rclone copy "$BACKUP_GZ" "$S3_BUCKET/$S3_PREFIX/" 2>/dev/null
        log_info "✓ 上传到远程存储成功"
    else
        log_info "⚠ 未找到aws或rclone,跳过云存储上传"
    fi
}

upload_to_remote() {
    if [ -z "$REMOTE_HOST" ] || [ -z "$REMOTE_PATH" ]; then
        log_info "远程备份未配置,跳过"
        return 0
    fi

    log_info "上传备份到远程服务器: $REMOTE_HOST:$REMOTE_PATH"

    if command -v scp > /dev/null 2>&1; then
        scp -o StrictHostKeyChecking=no "$BACKUP_GZ" "$REMOTE_HOST:$REMOTE_PATH/" 2>/dev/null
        log_info "✓ 上传到远程服务器成功"
    else
        log_info "⚠ scp未安装,跳过远程上传"
    fi
}

verify_backup() {
    log_info "验证备份文件..."

    if [ -f "$BACKUP_GZ" ] && [ -s "$BACKUP_GZ" ]; then
        if gzip -t "$BACKUP_GZ" 2>/dev/null; then
            log_info "✓ 备份文件验证通过"
            return 0
        fi
    fi

    log_error "备份文件验证失败"
    return 1
}

main() {
    echo "===== HJTPX 数据库备份脚本 ====="
    echo ""

    log_info "备份配置:"
    log_info "  数据库: $DB_NAME"
    log_info "  备份目录: $BACKUP_DIR"
    log_info "  保留天数: $RETENTION_DAYS"
    log_info "  压缩级别: $COMPRESSION_LEVEL"
    echo ""

    if ! check_dependencies; then
        exit 1
    fi

    create_backup || exit 1

    backup_redis

    verify_backup || exit 1

    cleanup_old_backups

    upload_to_s3

    upload_to_remote

    echo ""
    log_info "===== 备份完成 ====="
    log_info "备份文件: $BACKUP_GZ"
    log_info "备份大小: $(du -h "$BACKUP_GZ" | cut -f1)"
}

main "$@"
