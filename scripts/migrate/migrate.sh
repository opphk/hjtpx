#!/bin/bash
# 数据库迁移脚本
# 版本: 1.0.0
# 创建时间: 2026-05-20
# 描述: 管理数据库迁移操作

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MIGRATE_DIR="${SCRIPT_DIR}/migrate"

# 默认配置
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-postgres123}"
DB_NAME="${DB_NAME:-hjtpx_db}"
DB_SSLMODE="${DB_SSLMODE:-disable}"

# 颜色定义
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

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

export PGPASSWORD="${DB_PASSWORD}"

check_psql() {
    if ! command -v psql &> /dev/null; then
        log_error "psql 命令未找到，请安装 PostgreSQL 客户端"
        exit 1
    fi
}

check_connection() {
    log_info "检查数据库连接..."

    if PGPASSWORD="${DB_PASSWORD}" psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d postgres -c "SELECT 1" &> /dev/null; then
        log_success "数据库连接成功"
        return 0
    else
        log_error "无法连接到数据库 ${DB_HOST}:${DB_PORT}"
        return 1
    fi
}

database_exists() {
    local dbname=$1
    PGPASSWORD="${DB_PASSWORD}" psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='${dbname}'" | grep -q 1
}

create_database() {
    log_info "创建数据库 ${DB_NAME}..."

    if database_exists "${DB_NAME}"; then
        log_warn "数据库 ${DB_NAME} 已存在"
        return 0
    fi

    PGPASSWORD="${DB_PASSWORD}" psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d postgres -c "CREATE DATABASE ${DB_NAME}" &> /dev/null

    if database_exists "${DB_NAME}"; then
        log_success "数据库 ${DB_NAME} 创建成功"
        return 0
    else
        log_error "数据库 ${DB_NAME} 创建失败"
        return 1
    fi
}

execute_sql() {
    local sql_file=$1
    local description=$2

    log_info "执行: ${description}"
    log_info "文件: ${sql_file}"

    PGPASSWORD="${DB_PASSWORD}" psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -f "${sql_file}" 2>&1

    if [ $? -eq 0 ]; then
        log_success "${description} 完成"
        return 0
    else
        log_error "${description} 失败"
        return 1
    fi
}

get_migration_version() {
    PGPASSWORD="${DB_PASSWORD}" psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -tAc "SELECT version FROM schema_migrations ORDER BY applied_at DESC LIMIT 1" 2>/dev/null | tr -d ' '
}

get_table_count() {
    PGPASSWORD="${DB_PASSWORD}" psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -tAc "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE'" 2>/dev/null | tr -d ' '
}

show_help() {
    cat << EOF
数据库迁移管理脚本

用法: $0 [命令] [选项]

命令:
    status          显示当前迁移状态
    init            初始化数据库 (创建数据库和表结构)
    migrate         执行所有待迁移
    seed            初始化种子数据
    rollback        回滚到最后一次迁移
    reset           重置数据库 (删除所有表)
    help            显示帮助信息

选项:
    --host          数据库主机 (默认: localhost)
    --port          数据库端口 (默认: 5432)
    --user          数据库用户 (默认: postgres)
    --password      数据库密码 (默认: postgres123)
    --dbname        数据库名称 (默认: hjtpx_db)
    --sslmode       SSL模式 (默认: disable)

环境变量:
    DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, DB_SSLMODE

示例:
    $0 status
    $0 init --dbname myapp
    $0 migrate --host 192.168.1.100 --user admin
EOF
}

cmd_status() {
    log_info "============================================"
    log_info "数据库迁移状态"
    log_info "============================================"

    if ! check_connection; then
        exit 1
    fi

    log_info "数据库: ${DB_NAME}"
    log_info "主机: ${DB_HOST}:${DB_PORT}"

    if ! database_exists "${DB_NAME}"; then
        log_warn "数据库 ${DB_NAME} 不存在"
        exit 1
    fi

    local current_version=$(get_migration_version)
    if [ -z "${current_version}" ]; then
        log_info "当前版本: 未迁移"
    else
        log_info "当前版本: ${current_version}"
    fi

    local table_count=$(get_table_count)
    log_info "表数量: ${table_count}"

    log_info ""
    log_info "可用的迁移文件:"
    if [ -d "${MIGRATE_DIR}" ]; then
        ls -1 "${MIGRATE_DIR}"/*.sql 2>/dev/null | while read -r file; do
            local basename=$(basename "${file}")
            local applied=$(PGPASSWORD="${DB_PASSWORD}" psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -tAc "SELECT 1 FROM schema_migrations WHERE version='${basename%.sql}'" 2>/dev/null | grep -q 1 && echo "✓" || echo "○")
            echo "  ${applied} ${basename}"
        done
    fi

    log_info "============================================"
}

cmd_init() {
    log_info "============================================"
    log_info "初始化数据库"
    log_info "============================================"

    check_psql

    if ! check_connection; then
        exit 1
    fi

    create_database || exit 1

    local schema_file="${MIGRATE_DIR}/001_init.sql"
    if [ -f "${schema_file}" ]; then
        execute_sql "${schema_file}" "创建表结构" || exit 1
    else
        log_error "找不到表结构文件: ${schema_file}"
        exit 1
    fi

    log_success "数据库初始化完成"
    cmd_status
}

cmd_migrate() {
    log_info "============================================"
    log_info "执行数据库迁移"
    log_info "============================================"

    check_psql

    if ! check_connection; then
        exit 1
    fi

    if ! database_exists "${DB_NAME}"; then
        log_error "数据库 ${DB_NAME} 不存在，请先运行 init 命令"
        exit 1
    fi

    if [ ! -d "${MIGRATE_DIR}" ]; then
        log_error "迁移目录不存在: ${MIGRATE_DIR}"
        exit 1
    fi

    local migration_count=0
    for sql_file in "${MIGRATE_DIR}"/*.sql; do
        if [ -f "${sql_file}" ]; then
            local basename=$(basename "${sql_file}")
            local version="${basename%.sql}"

            local already_applied=$(PGPASSWORD="${DB_PASSWORD}" psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -tAc "SELECT 1 FROM schema_migrations WHERE version='${version}'" 2>/dev/null | grep -q 1 && echo "yes" || echo "no")

            if [ "${already_applied}" = "no" ]; then
                execute_sql "${sql_file}" "迁移 ${basename}" || exit 1
                ((migration_count++))
            else
                log_info "跳过已应用的迁移: ${basename}"
            fi
        fi
    done

    if [ ${migration_count} -eq 0 ]; then
        log_info "没有待执行的迁移"
    else
        log_success "成功执行 ${migration_count} 个迁移"
    fi

    cmd_status
}

cmd_seed() {
    log_info "============================================"
    log_info "初始化种子数据"
    log_info "============================================"

    check_psql

    if ! check_connection; then
        exit 1
    fi

    if ! database_exists "${DB_NAME}"; then
        log_error "数据库 ${DB_NAME} 不存在，请先运行 init 命令"
        exit 1
    fi

    local seed_file="${MIGRATE_DIR}/init-data.sql"
    if [ -f "${seed_file}" ]; then
        execute_sql "${seed_file}" "初始化种子数据" || exit 1
        log_success "种子数据初始化完成"
    else
        log_error "找不到种子数据文件: ${seed_file}"
        exit 1
    fi
}

cmd_rollback() {
    log_info "============================================"
    log_info "回滚数据库"
    log_info "============================================"

    check_psql

    if ! check_connection; then
        exit 1
    fi

    if ! database_exists "${DB_NAME}"; then
        log_error "数据库 ${DB_NAME} 不存在"
        exit 1
    fi

    read -p "确定要回滚吗？这将删除所有表和数据 (yes/no): " confirm
    if [ "${confirm}" != "yes" ]; then
        log_info "取消回滚"
        exit 0
    fi

    log_info "开始回滚..."

    PGPASSWORD="${DB_PASSWORD}" psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" <<-EOSQL
		DROP SCHEMA public CASCADE;
		CREATE SCHEMA public;
		GRANT ALL ON SCHEMA public TO PUBLIC;
EOSQL

    log_success "回滚完成"
}

cmd_reset() {
    log_info "============================================"
    log_info "重置数据库"
    log_info "============================================"

    check_psql

    if ! check_connection; then
        exit 1
    fi

    read -p "确定要重置数据库吗？这将删除数据库 ${DB_NAME} (yes/no): " confirm
    if [ "${confirm}" != "yes" ]; then
        log_info "取消重置"
        exit 0
    fi

    log_info "删除数据库 ${DB_NAME}..."

    PGPASSWORD="${DB_PASSWORD}" psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d postgres -c "DROP DATABASE IF EXISTS ${DB_NAME}"

    log_success "数据库 ${DB_NAME} 已删除"
}

parse_args() {
    while [ $# -gt 0 ]; do
        case $1 in
            --host)
                DB_HOST="$2"
                shift 2
                ;;
            --port)
                DB_PORT="$2"
                shift 2
                ;;
            --user)
                DB_USER="$2"
                shift 2
                ;;
            --password)
                DB_PASSWORD="$2"
                shift 2
                ;;
            --dbname)
                DB_NAME="$2"
                shift 2
                ;;
            --sslmode)
                DB_SSLMODE="$2"
                shift 2
                ;;
            status|init|migrate|seed|rollback|reset|help)
                CMD="$1"
                shift
                ;;
            *)
                log_error "未知参数: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

main() {
    local cmd="${1:-help}"
    shift || true

    parse_args "$@"

    case "${CMD}" in
        status)
            cmd_status
            ;;
        init)
            cmd_init
            ;;
        migrate)
            cmd_migrate
            ;;
        seed)
            cmd_seed
            ;;
        rollback)
            cmd_rollback
            ;;
        reset)
            cmd_reset
            ;;
        help)
            show_help
            ;;
        *)
            log_error "未知命令: ${CMD}"
            show_help
            exit 1
            ;;
    esac
}

main "$@"
