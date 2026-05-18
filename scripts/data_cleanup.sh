#!/bin/bash
# 数据清理和归档脚本
# 用于清理过期数据、归档冷数据、优化数据库性能
# 创建时间: 2026-05-18

set -e

# 配置
DB_HOST="${POSTGRES_HOST:-localhost}"
DB_PORT="${POSTGRES_PORT:-5432}"
DB_USER="${POSTGRES_USER:-postgres}"
DB_NAME="${POSTGRES_DB:-verification}"
DB_PASSWORD="${POSTGRES_PASSWORD:-postgres}"

export PGPASSWORD="$DB_PASSWORD"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_debug() {
    echo -e "${BLUE}[DEBUG]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# 默认配置
ARCHIVE_THRESHOLD_DAYS="${ARCHIVE_THRESHOLD_DAYS:-30}"
CLEANUP_THRESHOLD_DAYS="${CLEANUP_THRESHOLD_DAYS:-365}"
BATCH_SIZE="${BATCH_SIZE:-1000}"
DRY_RUN="${DRY_RUN:-false}"
BACKUP_ENABLED="${BACKUP_ENABLED:-true}"
BACKUP_DIR="${BACKUP_DIR:-./backups}"

# 帮助信息
show_help() {
    cat << EOF
用法: $0 [选项] <命令>

命令:
    archive         归档冷数据
    cleanup         清理过期数据
    vacuum           VACUUM 和 ANALYZE
    stats           显示表统计信息
    analyze         分析慢查询
    full            执行完整优化（归档+清理+VACUUM）
    help            显示帮助信息

选项:
    --dry-run           仅模拟操作，不实际执行
    --batch-size N      批处理大小 (默认: $BATCH_SIZE)
    --archive-days N    归档阈值天数 (默认: $ARCHIVE_THRESHOLD_DAYS)
    --cleanup-days N    清理阈值天数 (默认: $CLEANUP_THRESHOLD_DAYS)
    --table TABLE       指定操作的数据表
    --backup-dir DIR    备份目录 (默认: $BACKUP_DIR)

环境变量:
    POSTGRES_HOST       数据库主机 (默认: localhost)
    POSTGRES_PORT       数据库端口 (默认: 5432)
    POSTGRES_USER       数据库用户 (默认: postgres)
    POSTGRES_PASSWORD   数据库密码
    POSTGRES_DB         数据库名称 (默认: verification)

示例:
    $0 archive                                    # 归档30天前的数据
    $0 archive --archive-days 60                   # 归档60天前的数据
    $0 cleanup --dry-run                          # 模拟清理操作
    $0 full --batch-size 5000                     # 使用较大批次执行完整优化
EOF
}

# 执行 SQL 查询
execute_sql() {
    local sql="$1"
    local description="${2:-SQL execution}"

    if [ "$DRY_RUN" = "true" ]; then
        log_debug "[DRY-RUN] $description"
        log_debug "SQL: $sql"
        return 0
    fi

    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "$sql" 2>/dev/null
}

# 执行 SQL 并返回结果
query_sql() {
    local sql="$1"
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -A -c "$sql" 2>/dev/null
}

# 备份表
backup_table() {
    local table="$1"
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_file="${BACKUP_DIR}/${table}_${timestamp}.sql"

    if [ "$BACKUP_ENABLED" != "true" ]; then
        log_warn "备份已禁用，跳过 $table"
        return 0
    fi

    mkdir -p "$BACKUP_DIR"

    if [ "$DRY_RUN" = "true" ]; then
        log_debug "[DRY-RUN] 备份表 $table 到 $backup_file"
        return 0
    fi

    log_info "备份表 $table..."
    pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
        -t "$table" -f "$backup_file" 2>/dev/null

    if [ $? -eq 0 ]; then
        log_info "备份成功: $backup_file"
        gzip "$backup_file"
        log_info "压缩完成: ${backup_file}.gz"
    else
        log_error "备份表 $table 失败"
        return 1
    fi
}

# 归档数据
archive_data() {
    local table="$1"
    local date_field="$2"
    local archive_table="archive_${table}"

    log_info "开始归档表 $table (归档 ${ARCHIVE_THRESHOLD_DAYS} 天前的数据)..."

    local count=$(query_sql "SELECT COUNT(*) FROM $table WHERE $date_field < NOW() - INTERVAL '$ARCHIVE_THRESHOLD_DAYS days'")

    if [ -z "$count" ] || [ "$count" = "0" ]; then
        log_info "表 $table 没有需要归档的数据"
        return 0
    fi

    log_info "将归档 $count 条记录"

    # 备份将被归档的数据
    backup_table "${table}_to_archive"

    # 创建归档表（如果不存在）
    execute_sql "CREATE TABLE IF NOT EXISTS $archive_table (LIKE $table INCLUDING ALL)" "创建归档表"

    # 分批归档
    local offset=0
    local archived=0

    while [ $offset -lt $count ]; do
        local batch_count=$(query_sql "
            SELECT COUNT(*) FROM $table
            WHERE $date_field < NOW() - INTERVAL '$ARCHIVE_THRESHOLD_DAYS days'
            LIMIT $BATCH_SIZE OFFSET $offset
        ")

        execute_sql "
            INSERT INTO $archive_table
            SELECT * FROM $table
            WHERE $date_field < NOW() - INTERVAL '$ARCHIVE_THRESHOLD_DAYS days'
            LIMIT $BATCH_SIZE OFFSET $offset
        " "归档批次数据"

        execute_sql "
            DELETE FROM $table
            WHERE ctid IN (
                SELECT ctid FROM $table
                WHERE $date_field < NOW() - INTERVAL '$ARCHIVE_THRESHOLD_DAYS days'
                LIMIT $BATCH_SIZE OFFSET $offset
            )
        " "删除已归档数据"

        offset=$((offset + BATCH_SIZE))
        archived=$((archived + batch_count))
        log_info "已归档 $archived / $count 条记录"
    done

    # 创建归档索引
    execute_sql "CREATE INDEX IF NOT EXISTS idx_${archive_table}_archived_at ON $archive_table($date_field)" "创建归档索引"

    log_info "表 $table 归档完成，共归档 $archived 条记录"
}

# 清理数据
cleanup_data() {
    local table="$1"
    local date_field="$2"

    log_info "开始清理表 $table (清理 ${CLEANUP_THRESHOLD_DAYS} 天前的数据)..."

    # 检查表是否存在
    local exists=$(query_sql "SELECT EXISTS (SELECT FROM pg_tables WHERE tablename = '$table')")

    if [ "$exists" != "t" ]; then
        log_warn "表 $table 不存在，跳过"
        return 0
    fi

    local count=$(query_sql "SELECT COUNT(*) FROM $table WHERE $date_field < NOW() - INTERVAL '$CLEANUP_THRESHOLD_DAYS days'")

    if [ -z "$count" ] || [ "$count" = "0" ]; then
        log_info "表 $table 没有需要清理的数据"
        return 0
    fi

    log_info "将清理 $count 条记录"

    if [ "$DRY_RUN" = "true" ]; then
        log_debug "[DRY-RUN] 模拟清理 $count 条记录"
        return 0
    fi

    # 分批清理
    local offset=0
    local cleaned=0

    while [ $offset -lt $count ]; do
        execute_sql "
            DELETE FROM $table
            WHERE ctid IN (
                SELECT ctid FROM $table
                WHERE $date_field < NOW() - INTERVAL '$CLEANUP_THRESHOLD_DAYS days'
                LIMIT $BATCH_SIZE
            )
        " "清理批次数据"

        cleaned=$((cleaned + BATCH_SIZE))
        if [ $cleaned -gt $count ]; then
            cleaned=$count
        fi
        log_info "已清理 $cleaned / $count 条记录"
        offset=$((offset + BATCH_SIZE))
    done

    log_info "表 $table 清理完成，共清理 $cleaned 条记录"
}

# VACUUM 和 ANALYZE
vacuum_analyze() {
    log_info "开始 VACUUM 和 ANALYZE..."

    if [ "$DRY_RUN" = "true" ]; then
        log_debug "[DRY-RUN] 将执行 VACUUM ANALYZE"
        return 0
    fi

    execute_sql "VACUUM (VERBOSE, ANALYZE)" "VACUUM ANALYZE"

    log_info "VACUUM 和 ANALYZE 完成"
}

# 显示统计信息
show_stats() {
    log_info "数据库表统计信息:"
    echo ""

    query_sql "
        SELECT
            schemaname,
            relname AS table_name,
            pg_size_pretty(pg_total_relation_size(relid)) AS total_size,
            pg_size_pretty(pg_relation_size(relid)) AS table_size,
            pg_size_pretty(pg_indexes_size(relid)) AS index_size,
            n_live_tup AS live_rows,
            n_dead_tup AS dead_rows,
            last_vacuum,
            last_autovacuum,
            last_analyze,
            last_autoanalyze
        FROM pg_stat_user_tables
        WHERE schemaname = 'public'
        ORDER BY pg_total_relation_size(relid) DESC
        LIMIT 20
    " | column -t -s '|'

    echo ""
    log_info "索引使用统计:"
    echo ""

    query_sql "
        SELECT
            schemaname,
            relname AS table_name,
            indexrelname AS index_name,
            idx_scan,
            idx_tup_read,
            idx_tup_fetch,
            pg_size_pretty(pg_relation_size(indexrelid)) AS index_size,
            CASE
                WHEN idx_scan = 0 THEN '未使用'
                WHEN idx_tup_fetch::numeric / NULLIF(idx_scan, 0) < 0.5 THEN '低效'
                ELSE '正常'
            END AS status
        FROM pg_stat_user_indexes
        WHERE schemaname = 'public'
          AND indexrelname NOT LIKE '%pkey%'
        ORDER BY idx_scan ASC
        LIMIT 30
    " | column -t -s '|'
}

# 分析慢查询
analyze_slow_queries() {
    log_info "慢查询分析:"

    local has_pg_statements=$(query_sql "SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_stat_statements')")

    if [ "$has_pg_statements" != "t" ]; then
        log_warn "pg_stat_statements 扩展未启用，无法分析慢查询"
        return 0
    fi

    echo ""

    query_sql "
        SELECT
            ROUND(mean_time::numeric / 1000, 2) AS mean_time_ms,
            ROUND(total_time::numeric / 1000, 2) AS total_time_ms,
            calls,
            rows,
            LEFT(query, 100) AS query_preview
        FROM pg_stat_statements
        WHERE mean_time > 10000
        ORDER BY mean_time DESC
        LIMIT 10
    " | column -t -s '|'

    echo ""
    log_info "未使用的索引:"
    echo ""

    query_sql "
        SELECT
            schemaname,
            relname AS table_name,
            indexrelname AS index_name,
            pg_size_pretty(pg_relation_size(indexrelid)) AS index_size,
            idx_scan
        FROM pg_stat_user_indexes
        WHERE schemaname = 'public'
          AND indexrelname NOT LIKE '%pkey%'
          AND idx_scan = 0
        ORDER BY pg_relation_size(indexrelid) DESC
    " | column -t -s '|'
}

# 完整优化
full_optimization() {
    log_info "开始执行完整数据库优化..."

    # 归档数据
    log_info "=== 阶段 1: 数据归档 ==="
    archive_data "verification_logs" "created_at"
    archive_data "risk_logs" "created_at"
    archive_data "security_logs" "created_at"
    archive_data "admin_login_logs" "created_at"

    # 清理数据
    log_info "=== 阶段 2: 数据清理 ==="
    cleanup_data "verification_logs" "created_at"
    cleanup_data "risk_logs" "created_at"
    cleanup_data "security_logs" "created_at"
    cleanup_data "admin_login_logs" "created_at"

    # VACUUM 和 ANALYZE
    log_info "=== 阶段 3: VACUUM 和 ANALYZE ==="
    vacuum_analyze

    log_info "=== 完整优化完成 ==="
}

# 主函数
main() {
    local command="${1:-help}"
    shift || true

    case "$command" in
        archive)
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --dry-run)
                        DRY_RUN="true"
                        ;;
                    --batch-size)
                        BATCH_SIZE="$2"
                        shift
                        ;;
                    --archive-days)
                        ARCHIVE_THRESHOLD_DAYS="$2"
                        shift
                        ;;
                    --table)
                        TABLE="$2"
                        shift
                        ;;
                    *)
                        log_error "未知参数: $1"
                        show_help
                        exit 1
                        ;;
                esac
                shift
            done

            if [ -n "$TABLE" ]; then
                archive_data "$TABLE" "created_at"
            else
                archive_data "verification_logs" "created_at"
                archive_data "risk_logs" "created_at"
                archive_data "security_logs" "created_at"
            fi
            ;;

        cleanup)
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --dry-run)
                        DRY_RUN="true"
                        ;;
                    --batch-size)
                        BATCH_SIZE="$2"
                        shift
                        ;;
                    --cleanup-days)
                        CLEANUP_THRESHOLD_DAYS="$2"
                        shift
                        ;;
                    --table)
                        TABLE="$2"
                        shift
                        ;;
                    *)
                        log_error "未知参数: $1"
                        show_help
                        exit 1
                        ;;
                esac
                shift
            done

            if [ -n "$TABLE" ]; then
                cleanup_data "$TABLE" "created_at"
            else
                cleanup_data "verification_logs" "created_at"
                cleanup_data "risk_logs" "created_at"
                cleanup_data "security_logs" "created_at"
            fi
            ;;

        vacuum)
            vacuum_analyze
            ;;

        stats)
            show_stats
            ;;

        analyze)
            analyze_slow_queries
            ;;

        full)
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --dry-run)
                        DRY_RUN="true"
                        ;;
                    --batch-size)
                        BATCH_SIZE="$2"
                        shift
                        ;;
                    *)
                        ;;
                esac
                shift
            done

            full_optimization
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
