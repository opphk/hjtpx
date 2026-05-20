#!/bin/bash
# 数据库验证脚本
# 版本: 1.0.0
# 创建时间: 2026-05-20
# 描述: 验证数据库连接和表结构

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

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
CYAN='\033[0;36m'
NC='\033[0m'

# 统计变量
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0

export PGPASSWORD="${DB_PASSWORD}"

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓ PASS]${NC} $1"
    ((TESTS_PASSED++))
    ((TESTS_TOTAL++))
}

log_fail() {
    echo -e "${RED}[✗ FAIL]${NC} $1"
    ((TESTS_FAILED++))
    ((TESTS_TOTAL++))
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

run_test() {
    local test_name=$1
    local test_command=$2

    echo -n "  测试: ${test_name} ... "
    if eval "${test_command}" &> /dev/null; then
        log_success "${test_name}"
        return 0
    else
        log_fail "${test_name}"
        return 1
    fi
}

check_psql() {
    if ! command -v psql &> /dev/null; then
        log_error "psql 命令未找到，请安装 PostgreSQL 客户端"
        exit 1
    fi
}

log_section() {
    echo ""
    echo -e "${CYAN}========================================${NC}"
    echo -e "${CYAN}$1${NC}"
    echo -e "${CYAN}========================================${NC}"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

show_help() {
    cat << EOF
数据库验证脚本

用法: $0 [选项]

选项:
    --host          数据库主机 (默认: localhost)
    --port          数据库端口 (默认: 5432)
    --user          数据库用户 (默认: postgres)
    --password      数据库密码 (默认: postgres123)
    --dbname        数据库名称 (默认: hjtpx_db)
    --quick         快速模式 (仅检查连接和表)
    --full          完整模式 (包括性能检查)
    --help          显示帮助信息

环境变量:
    DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME

示例:
    $0 --dbname myapp
    $0 --quick
    $0 --full --host 192.168.1.100
EOF
}

verify_connection() {
    log_section "1. 数据库连接验证"

    run_test "psql命令可用" "command -v psql"

    local conn_str="-h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d postgres"
    run_test "数据库服务器可连接" "psql ${conn_str} -c 'SELECT 1'"

    run_test "数据库 ${DB_NAME} 存在" "PGPASSWORD='${DB_PASSWORD}' psql -h '${DB_HOST}' -p '${DB_PORT}' -U '${DB_USER}' -d postgres -tAc \"SELECT 1 FROM pg_database WHERE datname='${DB_NAME}'\" | grep -q 1"

    run_test "数据库用户有连接权限" "PGPASSWORD='${DB_PASSWORD}' psql -h '${DB_HOST}' -p '${DB_PORT}' -U '${DB_USER}' -d '${DB_NAME}' -c 'SELECT 1'"

    run_test "数据库版本检查" "PGPASSWORD='${DB_PASSWORD}' psql -h '${DB_HOST}' -p '${DB_PORT}' -U '${DB_USER}' -d '${DB_NAME}' -tAc 'SELECT version()' | grep -q PostgreSQL"
}

verify_schema() {
    log_section "2. 表结构验证"

    local required_tables=(
        "users"
        "admins"
        "applications"
        "verifications"
        "captcha_sessions"
        "risk_rules"
        "risk_rule_templates"
        "risk_logs"
        "audit_logs"
        "blacklist"
        "system_configs"
        "behavior_data"
        "device_fingerprints"
        "verification_logs"
        "user_mfa"
        "admin_mfa"
        "mfa_codes"
        "api_key_histories"
        "voice_captcha_sessions"
        "admin_login_logs"
        "trace_records"
        "risk_rule_trigger_histories"
        "schema_migrations"
    )

    for table in "${required_tables[@]}"; do
        run_test "表 ${table} 存在" "PGPASSWORD='${DB_PASSWORD}' psql -h '${DB_HOST}' -p '${DB_PORT}' -U '${DB_USER}' -d '${DB_NAME}' -tAc \"SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='${table}'\" | grep -q 1"
    done
}

verify_indexes() {
    log_section "3. 索引验证"

    local critical_indexes=(
        "idx_users_username"
        "idx_users_email"
        "idx_applications_api_key"
        "idx_verifications_session_id"
        "idx_verifications_app_id"
        "idx_risk_rules_enabled"
        "idx_audit_logs_created_at"
        "idx_risk_logs_created_at"
    )

    for idx in "${critical_indexes[@]}"; do
        run_test "索引 ${idx} 存在" "PGPASSWORD='${DB_PASSWORD}' psql -h '${DB_HOST}' -p '${DB_PORT}' -U '${DB_USER}' -d '${DB_NAME}' -tAc \"SELECT 1 FROM pg_indexes WHERE indexname='${idx}'\" | grep -q 1"
    done
}

verify_constraints() {
    log_section "4. 约束验证"

    run_test "users 表主键" "PGPASSWORD='${DB_PASSWORD}' psql -h '${DB_HOST}' -p '${DB_PORT}' -U '${DB_USER}' -d '${DB_NAME}' -tAc \"SELECT 1 FROM information_schema.table_constraints WHERE table_name='users' AND constraint_type='PRIMARY KEY'\" | grep -q 1"

    run_test "users.username 唯一约束" "PGPASSWORD='${DB_PASSWORD}' psql -h '${DB_HOST}' -p '${DB_PORT}' -U '${DB_USER}' -d '${DB_NAME}' -tAc \"SELECT 1 FROM information_schema.table_constraints WHERE table_name='users' AND constraint_name='idx_users_username'\" | grep -q 1"

    run_test "applications.api_key 唯一约束" "PGPASSWORD='${DB_PASSWORD}' psql -h '${DB_HOST}' -p '${DB_PORT}' -U '${DB_USER}' -d '${DB_NAME}' -tAc \"SELECT 1 FROM information_schema.table_constraints WHERE table_name='applications' AND constraint_name='idx_applications_api_key'\" | grep -q 1"

    run_test "update_updated_at_column 函数存在" "PGPASSWORD='${DB_PASSWORD}' psql -h '${DB_HOST}' -p '${DB_PORT}' -U '${DB_USER}' -d '${DB_NAME}' -tAc \"SELECT 1 FROM pg_proc WHERE proname='update_updated_at_column'\" | grep -q 1"

    run_test "自动更新触发器" "PGPASSWORD='${DB_PASSWORD}' psql -h '${DB_HOST}' -p '${DB_PORT}' -U '${DB_USER}' -d '${DB_NAME}' -tAc \"SELECT 1 FROM pg_trigger WHERE tgname LIKE '%updated_at%'\" | grep -q 1"
}

verify_data() {
    log_section "5. 数据验证"

    run_test "管理员账户存在" "PGPASSWORD='${DB_PASSWORD}' psql -h '${DB_HOST}' -p '${DB_PORT}' -U '${DB_USER}' -d '${DB_NAME}' -tAc \"SELECT 1 FROM admins WHERE username='admin'\" | grep -q 1"

    run_test "系统配置已初始化" "PGPASSWORD='${DB_PASSWORD}' psql -h '${DB_HOST}' -p '${DB_PORT}' -U '${DB_USER}' -d '${DB_NAME}' -tAc \"SELECT COUNT(*) FROM system_configs\" | awk '{exit !(\$1>0)}'"

    run_test "风控规则模板已初始化" "PGPASSWORD='${DB_PASSWORD}' psql -h '${DB_HOST}' -p '${DB_PORT}' -U '${DB_USER}' -d '${DB_NAME}' -tAc \"SELECT COUNT(*) FROM risk_rule_templates\" | awk '{exit !(\$1>0)}'"

    run_test "风控规则已初始化" "PGPASSWORD='${DB_PASSWORD}' psql -h '${DB_HOST}' -p '${DB_PORT}' -U '${DB_USER}' -d '${DB_NAME}' -tAc \"SELECT COUNT(*) FROM risk_rules\" | awk '{exit !(\$1>0)}'"

    run_test "迁移记录存在" "PGPASSWORD='${DB_PASSWORD}' psql -h '${DB_HOST}' -p '${DB_PORT}' -U '${DB_USER}' -d '${DB_NAME}' -tAc \"SELECT 1 FROM schema_migrations WHERE version='001_init'\" | grep -q 1"
}

verify_performance() {
    log_section "6. 性能验证"

    log_info "检查表统计信息..."

    local tables_with_stats=(
        "users"
        "applications"
        "verifications"
        "audit_logs"
        "risk_logs"
    )

    for table in "${tables_with_stats[@]}"; do
        local has_stats=$(PGPASSWORD="${DB_PASSWORD}" psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -tAc "SELECT 1 FROM pg_stat_user_tables WHERE relname='${table}'" 2>/dev/null | grep -q 1 && echo "yes" || echo "no")
        if [ "${has_stats}" = "yes" ]; then
            log_success "表 ${table} 统计信息可用"
        else
            log_warn "表 ${table} 统计信息不可用 (可能需要 ANALYZE)"
        fi
    done

    log_info "检查索引使用情况..."

    local unused_indexes=$(PGPASSWORD="${DB_PASSWORD}" psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -tAc "SELECT indexname FROM pg_stat_user_indexes WHERE idx_scan = 0 AND schemaname = 'public'" 2>/dev/null)

    if [ -n "${unused_indexes}" ]; then
        log_warn "发现未使用的索引:"
        echo "${unused_indexes}" | while read -r idx; do
            echo "    - ${idx}"
        done
    else
        log_success "所有索引都有使用记录"
    fi

    log_info "检查数据库大小..."

    local db_size=$(PGPASSWORD="${DB_PASSWORD}" psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -tAc "SELECT pg_size_pretty(pg_database_size('${DB_NAME}'))" 2>/dev/null | tr -d ' ')
    log_info "数据库大小: ${db_size}"

    local table_count=$(PGPASSWORD="${DB_PASSWORD}" psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -tAc "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE'" 2>/dev/null | tr -d ' ')
    log_info "表数量: ${table_count}"

    local index_count=$(PGPASSWORD="${DB_PASSWORD}" psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -tAc "SELECT COUNT(*) FROM pg_indexes WHERE schemaname='public'" 2>/dev/null | tr -d ' ')
    log_info "索引数量: ${index_count}"
}

verify_quick() {
    verify_connection
    verify_schema
    verify_data
}

verify_full() {
    verify_connection
    verify_schema
    verify_indexes
    verify_constraints
    verify_data
    verify_performance
}

print_summary() {
    log_section "验证总结"

    echo -e "总计测试: ${TESTS_TOTAL}"
    echo -e "${GREEN}通过: ${TESTS_PASSED}${NC}"
    echo -e "${RED}失败: ${TESTS_FAILED}${NC}"

    if [ ${TESTS_FAILED} -eq 0 ]; then
        echo ""
        log_success "所有验证测试通过!"
        return 0
    else
        echo ""
        log_error "有 ${TESTS_FAILED} 个测试失败"
        return 1
    fi
}

main() {
    local mode="full"

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
            --quick)
                mode="quick"
                shift
                ;;
            --full)
                mode="full"
                shift
                ;;
            --help)
                show_help
                exit 0
                ;;
            *)
                log_error "未知参数: $1"
                show_help
                exit 1
                ;;
        esac
    done

    log_info "数据库验证工具"
    log_info "主机: ${DB_HOST}:${DB_PORT}"
    log_info "数据库: ${DB_NAME}"
    log_info "模式: ${mode}"

    case ${mode} in
        quick)
            verify_quick
            ;;
        full)
            verify_full
            ;;
    esac

    print_summary
}

main "$@"
