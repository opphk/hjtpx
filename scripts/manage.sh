#!/bin/bash
set -e

echo "===== HJTPX 扩展脚本 ====="

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

case "${1:-}" in
    start)
        echo "启动所有服务..."
        if command -v docker-compose &> /dev/null; then
            docker-compose up -d
        else
            docker compose up -d
        fi
        ;;
    stop)
        echo "停止所有服务..."
        if command -v docker-compose &> /dev/null; then
            docker-compose down
        else
            docker compose down
        fi
        ;;
    restart)
        echo "重启所有服务..."
        if command -v docker-compose &> /dev/null; then
            docker-compose restart
        else
            docker compose restart
        fi
        ;;
    logs)
        ./scripts/logs.sh "${2:-app}"
        ;;
    status)
        if command -v docker-compose &> /dev/null; then
            docker-compose ps
        else
            docker compose ps
        fi
        ;;
    backup)
        ./scripts/backup.sh
        ;;
    health)
        ./scripts/health-check.sh
        ;;
    update)
        ./scripts/update.sh
        ;;
    *)
        echo "用法: $0 {start|stop|restart|logs|status|backup|health|update}"
        echo ""
        echo "命令说明:"
        echo "  start   - 启动所有服务"
        echo "  stop    - 停止所有服务"
        echo "  restart - 重启所有服务"
        echo "  logs    - 查看日志 (可选: 服务名)"
        echo "  status  - 查看服务状态"
        echo "  backup  - 备份数据库"
        echo "  health  - 健康检查"
        echo "  update  - 更新应用"
        exit 1
        ;;
esac
