#!/bin/bash
set -e

echo "===== HJTPX 数据库备份脚本 ====="

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

BACKUP_DIR="${BACKUP_DIR:-./backups}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/backup_$TIMESTAMP.sql.gz"

mkdir -p "$BACKUP_DIR"

echo "开始备份数据库..."
echo "备份文件: $BACKUP_FILE"

if command -v docker-compose &> /dev/null; then
    docker-compose exec -T postgres pg_dump -U postgres hjtpx_db | gzip > "$BACKUP_FILE"
else
    docker compose exec -T postgres pg_dump -U postgres hjtpx_db | gzip > "$BACKUP_FILE"
fi

BACKUP_SIZE=$(du -h "$BACKUP_FILE" | cut -f1)
echo "备份完成! 文件大小: $BACKUP_SIZE"

echo "删除30天前的旧备份..."
find "$BACKUP_DIR" -name "backup_*.sql.gz" -mtime +30 -delete

echo "===== 备份完成 ====="
echo "备份文件位置: $BACKUP_FILE"
