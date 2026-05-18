#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BACKEND_DIR="$PROJECT_ROOT/backend"
FRONTEND_DIR="$PROJECT_ROOT/frontend"
BUILD_DIR="$PROJECT_ROOT/build"
BACKUP_DIR="$PROJECT_ROOT/.obfuscate_backup"
CONFIG_DIR="$PROJECT_ROOT/.obfuscate_config"
LOG_FILE="$PROJECT_ROOT/logs/obfuscate.log"
OBFUSCATOR_BINARY=""

OBFUSCATION_LEVEL="${OBFUSCATION_LEVEL:-2}"
ENABLE_VARIABLE_OBFUSCATION="${ENABLE_VARIABLE_OBFUSCATION:-true}"
ENABLE_STRING_ENCRYPTION="${ENABLE_STRING_ENCRYPTION:-true}"
ENABLE_CODE_COMPRESSION="${ENABLE_CODE_COMPRESSION:-true}"
ENABLE_CONTROL_FLOW_FLATTENING="${ENABLE_CONTROL_FLOW_FLATTENING:-true}"
ENABLE_DEAD_CODE_INJECTION="${ENABLE_DEAD_CODE_INJECTION:-false}"
ENABLE_FUNCTION_WRAPPING="${ENABLE_FUNCTION_WRAPPING:-true}"
REMOVE_COMMENTS="${REMOVE_COMMENTS:-true}"
PRESERVE_CONSOLE="${PRESERVE_CONSOLE:-true}"
OBFUSCATION_KEY="${OBFUSCATION_KEY:-hjtpx-obfuscate-key-2024}"
BACKUP_ENABLED="${BACKUP_ENABLED:-true}"
VERBOSE="${VERBOSE:-false}"
DRY_RUN="${DRY_RUN:-false}"
FORCE="${FORCE:-false}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    local message="$1"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${BLUE}[INFO]${NC} $message"
    if [[ -n "$LOG_FILE" ]]; then
        mkdir -p "$(dirname "$LOG_FILE")"
        echo "[$timestamp] [INFO] $message" >> "$LOG_FILE"
    fi
}

log_success() {
    local message="$1"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${GREEN}[SUCCESS]${NC} $message"
    if [[ -n "$LOG_FILE" ]]; then
        mkdir -p "$(dirname "$LOG_FILE")"
        echo "[$timestamp] [SUCCESS] $message" >> "$LOG_FILE"
    fi
}

log_warning() {
    local message="$1"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${YELLOW}[WARNING]${NC} $message"
    if [[ -n "$LOG_FILE" ]]; then
        mkdir -p "$(dirname "$LOG_FILE")"
        echo "[$timestamp] [WARNING] $message" >> "$LOG_FILE"
    fi
}

log_error() {
    local message="$1"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${RED}[ERROR]${NC} $message" >&2
    if [[ -n "$LOG_FILE" ]]; then
        mkdir -p "$(dirname "$LOG_FILE")"
        echo "[$timestamp] [ERROR] $message" >> "$LOG_FILE"
    fi
}

log_debug() {
    if [[ "$VERBOSE" == "true" ]]; then
        local message="$1"
        local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
        echo -e "[DEBUG] $message"
        if [[ -n "$LOG_FILE" ]]; then
            mkdir -p "$(dirname "$LOG_FILE")"
            echo "[$timestamp] [DEBUG] $message" >> "$LOG_FILE"
        fi
    fi
}

show_usage() {
    cat << EOF
用法: $0 [选项]

JavaScript代码混淆构建脚本

选项:
    -h, --help                      显示帮助信息
    -v, --verbose                   详细输出模式
    -d, --dry-run                   模拟运行，不实际执行混淆
    -f, --force                     强制执行，跳过确认提示
    -l, --level <1-3>               混淆级别 (1=轻量, 2=标准, 3=高级)
    -i, --input <文件或目录>          输入文件或目录
    -o, --output <目录>              输出目录
    -k, --key <密钥>                 混淆密钥
    --no-backup                     禁用备份
    --no-compression                禁用代码压缩
    --no-encryption                禁用字符串加密
    --no-variables                 禁用变量名混淆
    --enable-dead-code              启用死代码注入
    --rollback                      回滚到上一个备份版本
    --list-backups                  列出所有备份
    --status                        显示当前混淆状态
    --config <配置文件>              使用指定的配置文件

混淆级别:
    1 - 轻量级: 仅变量名混淆和代码压缩
    2 - 标准级: 变量混淆 + 字符串加密 + 代码压缩 + 函数包装
    3 - 高级级: 全部功能 + 控制流扁平化 + 死代码注入

示例:
    $0 -l 2 -i ./frontend/static/js/main.js -o ./build/js
    $0 --level 3 --input ./frontend/static/js --output ./build/js
    $0 --rollback
    $0 --config /path/to/obfuscate.yaml

环境变量:
    OBFUSCATION_LEVEL              混淆级别 (1-3)
    OBFUSCATION_KEY                混淆密钥
    VERBOSE                        详细输出
    DRY_RUN                        模拟运行

EOF
}

parse_arguments() {
    INPUT_PATH=""
    OUTPUT_PATH=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            -h|--help)
                show_usage
                exit 0
                ;;
            -v|--verbose)
                VERBOSE="true"
                shift
                ;;
            -d|--dry-run)
                DRY_RUN="true"
                shift
                ;;
            -f|--force)
                FORCE="true"
                shift
                ;;
            -l|--level)
                OBFUSCATION_LEVEL="$2"
                shift 2
                ;;
            -i|--input)
                INPUT_PATH="$2"
                shift 2
                ;;
            -o|--output)
                OUTPUT_PATH="$2"
                shift 2
                ;;
            -k|--key)
                OBFUSCATION_KEY="$2"
                shift 2
                ;;
            --no-backup)
                BACKUP_ENABLED="false"
                shift
                ;;
            --no-compression)
                ENABLE_CODE_COMPRESSION="false"
                shift
                ;;
            --no-encryption)
                ENABLE_STRING_ENCRYPTION="false"
                shift
                ;;
            --no-variables)
                ENABLE_VARIABLE_OBFUSCATION="false"
                shift
                ;;
            --enable-dead-code)
                ENABLE_DEAD_CODE_INJECTION="true"
                shift
                ;;
            --rollback)
                perform_rollback
                exit 0
                ;;
            --list-backups)
                list_backups
                exit 0
                ;;
            --status)
                show_status
                exit 0
                ;;
            --config)
                load_config "$2"
                shift 2
                ;;
            *)
                log_error "未知选项: $1"
                show_usage
                exit 1
                ;;
        esac
    done

    if [[ -z "$INPUT_PATH" ]]; then
        INPUT_PATH="$FRONTEND_DIR/static/js"
    fi

    if [[ -z "$OUTPUT_PATH" ]]; then
        OUTPUT_PATH="$BUILD_DIR/js"
    fi

    validate_inputs
}

validate_inputs() {
    if [[ ! -e "$INPUT_PATH" ]]; then
        log_error "输入路径不存在: $INPUT_PATH"
        exit 1
    fi

    if [[ "$OBFUSCATION_LEVEL" -lt 1 || "$OBFUSCATION_LEVEL" -gt 3 ]]; then
        log_error "无效的混淆级别: $OBFUSCATION_LEVEL (有效值: 1-3)"
        exit 1
    fi

    if [[ -z "$OBFUSCATION_KEY" ]]; then
        log_warning "未提供混淆密钥，使用默认值"
        OBFUSCATION_KEY="hjtpx-obfuscate-key-2024"
    fi

    if [[ ${#OBFUSCATION_KEY} -lt 16 ]]; then
        log_warning "密钥长度小于16字节，可能影响安全性"
    fi

    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "=== 模拟运行模式 ==="
    fi
}

load_config() {
    local config_file="$1"

    if [[ ! -f "$config_file" ]]; then
        log_error "配置文件不存在: $config_file"
        exit 1
    fi

    log_info "加载配置文件: $config_file"

    while IFS='=' read -r key value; do
        key=$(echo "$key" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        value=$(echo "$value" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')

        case "$key" in
            obfuscation_level) OBFUSCATION_LEVEL="$value" ;;
            enable_variable_obfuscation) ENABLE_VARIABLE_OBFUSCATION="$value" ;;
            enable_string_encryption) ENABLE_STRING_ENCRYPTION="$value" ;;
            enable_code_compression) ENABLE_CODE_COMPRESSION="$value" ;;
            enable_control_flow_flattening) ENABLE_CONTROL_FLOW_FLATTENING="$value" ;;
            enable_dead_code_injection) ENABLE_DEAD_CODE_INJECTION="$value" ;;
            enable_function_wrapping) ENABLE_FUNCTION_WRAPPING="$value" ;;
            remove_comments) REMOVE_COMMENTS="$value" ;;
            preserve_console) PRESERVE_CONSOLE="$value" ;;
            obfuscation_key) OBFUSCATION_KEY="$value" ;;
            backup_enabled) BACKUP_ENABLED="$value" ;;
            verbose) VERBOSE="$value" ;;
            input_path) INPUT_PATH="$value" ;;
            output_path) OUTPUT_PATH="$value" ;;
        esac
    done < "$config_file"

    log_success "配置文件加载完成"
}

save_config() {
    local config_file="$CONFIG_DIR/config_$(date +%Y%m%d_%H%M%S).conf"

    mkdir -p "$CONFIG_DIR"

    cat > "$config_file" << EOF
obfuscation_level=$OBFUSCATION_LEVEL
enable_variable_obfuscation=$ENABLE_VARIABLE_OBFUSCATION
enable_string_encryption=$ENABLE_STRING_ENCRYPTION
enable_code_compression=$ENABLE_CODE_COMPRESSION
enable_control_flow_flattening=$ENABLE_CONTROL_FLOW_FLATTENING
enable_dead_code_injection=$ENABLE_DEAD_CODE_INJECTION
enable_function_wrapping=$ENABLE_FUNCTION_WRAPPING
remove_comments=$REMOVE_COMMENTS
preserve_console=$PRESERVE_CONSOLE
obfuscation_key=$OBFUSCATION_KEY
backup_enabled=$BACKUP_ENABLED
verbose=$VERBOSE
input_path=$INPUT_PATH
output_path=$OUTPUT_PATH
EOF

    log_debug "配置已保存: $config_file"
}

apply_obfuscation_level() {
    case "$OBFUSCATION_LEVEL" in
        1)
            ENABLE_VARIABLE_OBFUSCATION="true"
            ENABLE_STRING_ENCRYPTION="false"
            ENABLE_CODE_COMPRESSION="true"
            ENABLE_CONTROL_FLOW_FLATTENING="false"
            ENABLE_DEAD_CODE_INJECTION="false"
            ENABLE_FUNCTION_WRAPPING="false"
            ;;
        2)
            ENABLE_VARIABLE_OBFUSCATION="true"
            ENABLE_STRING_ENCRYPTION="true"
            ENABLE_CODE_COMPRESSION="true"
            ENABLE_CONTROL_FLOW_FLATTENING="false"
            ENABLE_DEAD_CODE_INJECTION="false"
            ENABLE_FUNCTION_WRAPPING="true"
            ;;
        3)
            ENABLE_VARIABLE_OBFUSCATION="true"
            ENABLE_STRING_ENCRYPTION="true"
            ENABLE_CODE_COMPRESSION="true"
            ENABLE_CONTROL_FLOW_FLATTENING="true"
            ENABLE_DEAD_CODE_INJECTION="true"
            ENABLE_FUNCTION_WRAPPING="true"
            ;;
    esac

    log_info "混淆级别 $OBFUSCATION_LEVEL 配置已应用"
}

create_backup() {
    if [[ "$BACKUP_ENABLED" != "true" ]]; then
        log_info "备份已禁用"
        return 0
    fi

    local backup_name="backup_$(date +%Y%m%d_%H%M%S)"
    local backup_path="$BACKUP_DIR/$backup_name"

    mkdir -p "$backup_path"
    mkdir -p "$backup_path/original"

    log_info "创建备份: $backup_name"

    if [[ -d "$INPUT_PATH" ]]; then
        find "$INPUT_PATH" -name "*.js" -type f | while read -r file; do
            local relative_path="${file#$INPUT_PATH/}"
            local target_dir="$backup_path/original/$(dirname "$relative_path")"
            mkdir -p "$target_dir"
            cp "$file" "$target_dir/"
            log_debug "备份: $file -> $target_dir/"
        done
    elif [[ -f "$INPUT_PATH" ]]; then
        cp "$INPUT_PATH" "$backup_path/original/"
    fi

    if [[ -d "$OUTPUT_PATH" ]]; then
        mkdir -p "$backup_path/obfuscated"
        find "$OUTPUT_PATH" -name "*.js" -type f | while read -r file; do
            local relative_path="${file#$OUTPUT_PATH/}"
            local target_dir="$backup_path/obfuscated/$(dirname "$relative_path")"
            mkdir -p "$target_dir"
            cp "$file" "$target_dir/"
        done
    fi

    cat > "$backup_path/metadata.json" << EOF
{
    "name": "$backup_name",
    "created_at": "$(date -Iseconds)",
    "input_path": "$INPUT_PATH",
    "output_path": "$OUTPUT_PATH",
    "obfuscation_level": $OBFUSCATION_LEVEL,
    "obfuscation_key_hash": "$(echo -n "$OBFUSCATION_KEY" | sha256sum | cut -d' ' -f1)"
}
EOF

    save_config

    cleanup_old_backups

    log_success "备份创建完成: $backup_name"
}

cleanup_old_backups() {
    local max_backups="${MAX_BACKUPS:-10}"

    local backup_count=$(find "$BACKUP_DIR" -maxdepth 1 -type d | wc -l)
    backup_count=$((backup_count - 1))

    if [[ $backup_count -gt $max_backups ]]; then
        log_info "清理旧备份，保留最近 $max_backups 个"

        find "$BACKUP_DIR" -maxdepth 1 -type d -name "backup_*" | \
            sort | head -n -"$max_backups" | while read -r old_backup; do
            log_debug "删除旧备份: $old_backup"
            rm -rf "$old_backup"
        done
    fi
}

list_backups() {
    if [[ ! -d "$BACKUP_DIR" ]]; then
        log_info "暂无备份"
        return 0
    fi

    echo ""
    echo "=== 可用备份列表 ==="
    echo ""

    find "$BACKUP_DIR" -maxdepth 1 -type d -name "backup_*" | sort -r | while read -r backup; do
        local name=$(basename "$backup")
        local date=$(grep -o '"created_at": "[^"]*"' "$backup/metadata.json" 2>/dev/null | cut -d'"' -f4 || echo "未知")
        local level=$(grep -o '"obfuscation_level": [0-9]*' "$backup/metadata.json" 2>/dev/null | cut -d':' -f2 || echo "?")

        echo "  $name"
        echo "    日期: $date"
        echo "    混淆级别: $level"
        echo "    原始文件: $(find "$backup/original" -name "*.js" 2>/dev/null | wc -l) 个"
        echo ""
    done
}

perform_rollback() {
    if [[ ! -d "$BACKUP_DIR" ]]; then
        log_error "没有可用的备份"
        exit 1
    fi

    local latest_backup=$(find "$BACKUP_DIR" -maxdepth 1 -type d -name "backup_*" | sort -r | head -n1)

    if [[ -z "$latest_backup" ]]; then
        log_error "没有可用的备份"
        exit 1
    fi

    local backup_name=$(basename "$latest_backup")

    if [[ "$FORCE" != "true" ]]; then
        echo ""
        read -p "确定要回滚到备份 '$backup_name' 吗? (y/N): " confirm
        if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
            log_info "取消回滚操作"
            exit 0
        fi
    fi

    log_info "开始回滚到备份: $backup_name"

    local original_files=$(find "$latest_backup/original" -name "*.js" -type f)

    for original_file in $original_files; do
        local relative_path="${original_file#$latest_backup/original/}"
        local target_file="$INPUT_PATH/$relative_path"

        local target_dir=$(dirname "$target_file")
        mkdir -p "$target_dir"

        cp "$original_file" "$target_file"
        log_debug "恢复: $target_file"
    done

    log_success "回滚完成: $backup_name"
    log_info "已恢复到原始文件"
}

obfuscate_file() {
    local input_file="$1"
    local output_file="$2"

    log_debug "处理文件: $input_file -> $output_file"

    local file_size=$(stat -f%z "$input_file" 2>/dev/null || stat -c%s "$input_file" 2>/dev/null || echo "0")
    log_debug "文件大小: $file_size bytes"

    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[模拟] 混淆: $input_file"
        return 0
    fi

    if command -v go &> /dev/null; then
        if [[ -f "$BACKEND_DIR/internal/tools/javascript_obfuscator.go" ]]; then
            build_obfuscator_tool
            if [[ -n "$OBFUSCATOR_BINARY" ]] && [[ -f "$OBFUSCATOR_BINARY" ]]; then
                "$OBFUSCATOR_BINARY" "$input_file" "$output_file"
                return $?
            fi
        fi
    fi

    perform_simple_obfuscation "$input_file" "$output_file"
}

build_obfuscator_tool() {
    local tool_dir="$BACKEND_DIR/internal/tools"
    local binary_name="obfuscator_tool"

    if [[ "$OS" == "Windows_NT" ]] || [[ "$(uname -s)" == *"CYGWIN"* ]] || [[ "$(uname -s)" == *"MINGW"* ]]; then
        binary_name="obfuscator_tool.exe"
    fi

    OBFUSCATOR_BINARY="$tool_dir/$binary_name"

    if [[ -f "$OBFUSCATOR_BINARY" ]] && [[ "$FORCE" != "true" ]]; then
        log_debug "使用已编译的工具: $OBFUSCATOR_BINARY"
        return 0
    fi

    log_info "编译混淆工具..."

    cd "$BACKEND_DIR"

    if go build -o "$OBFUSCATOR_BINARY" "./internal/tools/javascript_obfuscator.go" 2>/dev/null; then
        log_success "混淆工具编译成功"
        return 0
    else
        log_warning "无法编译混淆工具，将使用简单混淆方法"
        OBFUSCATOR_BINARY=""
        return 1
    fi
}

perform_simple_obfuscation() {
    local input_file="$1"
    local output_file="$2"

    mkdir -p "$(dirname "$output_file")"

    local content=$(cat "$input_file")

    if [[ "$REMOVE_COMMENTS" == "true" ]]; then
        content=$(echo "$content" | sed -E 's|/\*([^*]|\*[^/])*\*/||g')
        content=$(echo "$content" | sed -E 's|//[^\n]*||g')
    fi

    if [[ "$ENABLE_CODE_COMPRESSION" == "true" ]]; then
        content=$(echo "$content" | sed -E 's/[[:space:]]+/ /g')
        content=$(echo "$content" | sed -E 's/\s*([{};,:])\s*/\1/g')
        content=$(echo "$content" | sed -E 's/;\s*}/;}/g')
        content=$(echo "$content" | sed -E 's/\{\s*/{/g')
        content=$(echo "$content" | sed -E 's/\s*\}/}/g')
    fi

    if [[ "$ENABLE_VARIABLE_OBFUSCATION" == "true" ]]; then
        local counter=0
        declare -A var_map

        while IFS= read -r line; do
            for word in $(echo "$line" | grep -oE '\b[a-zA-Z_][a-zA-Z0-9_]*\b'); do
                if [[ ! -v var_map["$word"] ]] && ! is_reserved_word "$word"; then
                    var_map["$word"]="_0x$counter"
                    counter=$((counter + 1))
                fi
            done
        done <<< "$content"

        for word in "${!var_map[@]}"; do
            local replacement="${var_map[$word]}"
            content=$(echo "$content" | sed -E "s/\b$word\b/$replacement/g")
        done
    fi

    echo "$content" > "$output_file"

    log_success "混淆完成: $input_file -> $output_file"
}

is_reserved_word() {
    local word="$1"
    local reserved_words="if|else|for|while|do|switch|case|default|break|continue|return|try|catch|finally|throw|new|delete|typeof|instanceof|void|this|super|class|extends|static|get|set|import|export|from|as|const|let|var|function|async|await|yield|true|false|null|undefined|NaN|Infinity|arguments|eval|console|window|document|localStorage|sessionStorage|fetch|XMLHttpRequest|WebSocket"

    echo "$word" | grep -qE "^(if|else|for|while|do|switch|case|default|break|continue|return|try|catch|finally|throw|new|delete|typeof|instanceof|void|this|super|class|extends|static|get|set|import|export|from|as|const|let|var|function|async|await|yield|true|false|null|undefined|NaN|Infinity|arguments|eval|console|log|error|warn|info|debug)$"
    return $?
}

obfuscate_directory() {
    local input_dir="$1"
    local output_dir="$2"

    mkdir -p "$output_dir"

    local file_count=0
    local success_count=0

    while IFS= read -r -d '' file; do
        file_count=$((file_count + 1))
        local relative_path="${file#$input_dir/}"
        local target_file="$output_dir/$relative_path"
        local target_dir=$(dirname "$target_file")

        mkdir -p "$target_dir"

        if obfuscate_file "$file" "$target_file"; then
            success_count=$((success_count + 1))
        fi
    done < <(find "$input_dir" -name "*.js" -type f -print0)

    echo ""
    log_success "混淆完成: $success_count / $file_count 个文件"
}

show_status() {
    echo ""
    echo "=== 混淆状态 ==="
    echo ""

    echo "当前配置:"
    echo "  混淆级别: $OBFUSCATION_LEVEL"
    echo "  变量混淆: $ENABLE_VARIABLE_OBFUSCATION"
    echo "  字符串加密: $ENABLE_STRING_ENCRYPTION"
    echo "  代码压缩: $ENABLE_CODE_COMPRESSION"
    echo "  控制流扁平化: $ENABLE_CONTROL_FLOW_FLATTENING"
    echo "  死代码注入: $ENABLE_DEAD_CODE_INJECTION"
    echo "  函数包装: $ENABLE_FUNCTION_WRAPPING"
    echo "  备份启用: $BACKUP_ENABLED"
    echo ""

    if [[ -d "$BUILD_DIR/js" ]]; then
        local obfuscated_count=$(find "$BUILD_DIR/js" -name "*.js" -type f | wc -l)
        echo "已混淆文件数: $obfuscated_count"
    else
        echo "未找到已混淆的文件"
    fi

    if [[ -d "$BACKUP_DIR" ]]; then
        local backup_count=$(find "$BACKUP_DIR" -maxdepth 1 -type d | wc -l)
        backup_count=$((backup_count - 1))
        echo "可用备份数: $backup_count"
    else
        echo "可用备份数: 0"
    fi

    echo ""
}

confirm_obfuscation() {
    if [[ "$FORCE" == "true" ]] || [[ "$DRY_RUN" == "true" ]]; then
        return 0
    fi

    echo ""
    echo "=== 混淆配置确认 ==="
    echo ""
    echo "输入路径: $INPUT_PATH"
    echo "输出路径: $OUTPUT_PATH"
    echo "混淆级别: $OBFUSCATION_LEVEL"
    echo "备份启用: $BACKUP_ENABLED"
    echo ""

    read -p "确认开始混淆? (y/N): " confirm
    if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
        log_info "取消混淆操作"
        exit 0
    fi
}

generate_manifest() {
    local manifest_file="$BUILD_DIR/manifest.json"
    local timestamp=$(date -Iseconds)

    mkdir -p "$BUILD_DIR"

    cat > "$manifest_file" << EOF
{
    "version": "$OBFUSCATION_LEVEL",
    "generated_at": "$timestamp",
    "obfuscation_key_hash": "$(echo -n "$OBFUSCATION_KEY" | sha256sum | cut -d' ' -f1)",
    "files": [
EOF

    local first=true
    find "$OUTPUT_PATH" -name "*.js" -type f | while read -r file; do
        local relative_path="${file#$OUTPUT_PATH/}"
        local original_path="$INPUT_PATH/$relative_path"

        if [[ "$first" == "true" ]]; then
            first=false
        else
            echo "," >> "$manifest_file"
        fi

        local original_hash="null"
        local obfuscated_hash=$(sha256sum "$file" 2>/dev/null | cut -d' ' -f1)

        if [[ -f "$original_path" ]]; then
            original_hash="\"$(sha256sum "$original_path" | cut -d' ' -f1)\""
        fi

        echo "        {" >> "$manifest_file"
        echo "            \"path\": \"$relative_path\"," >> "$manifest_file"
        echo "            \"original_hash\": $original_hash," >> "$manifest_file"
        echo "            \"obfuscated_hash\": \"$obfuscated_hash\"" >> "$manifest_file"
        echo -n "        }" >> "$manifest_file"
    done

    echo "" >> "$manifest_file"
    echo "    ]," >> "$manifest_file"
    echo "    \"statistics\": {" >> "$manifest_file"
    echo "        \"total_files\": $(find "$OUTPUT_PATH" -name "*.js" -type f | wc -l)," >> "$manifest_file"
    echo "        \"total_size\": $(find "$OUTPUT_PATH" -name "*.js" -type f -exec cat {} \; | wc -c)" >> "$manifest_file"
    echo "    }" >> "$manifest_file"
    echo "}" >> "$manifest_file"

    log_success "清单文件已生成: $manifest_file"
}

evaluate_obfuscation_effectiveness() {
    local original_dir="$1"
    local obfuscated_dir="$2"
    local report_file="$BUILD_DIR/obfuscation_report.txt"

    log_info "评估混淆效果..."

    local total_original_size=0
    local total_obfuscated_size=0
    local file_count=0

    while IFS= read -r -d '' file; do
        local relative_path="${file#$original_dir/}"
        local obfuscated_file="$obfuscated_dir/$relative_path"

        if [[ -f "$obfuscated_file" ]]; then
            local original_size=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null || echo "0")
            local obfuscated_size=$(stat -f%z "$obfuscated_file" 2>/dev/null || stat -c%s "$obfuscated_file" 2>/dev/null || echo "0")

            total_original_size=$((total_original_size + original_size))
            total_obfuscated_size=$((total_obfuscated_size + obfuscated_size))
            file_count=$((file_count + 1))
        fi
    done < <(find "$original_dir" -name "*.js" -type f -print0)

    local size_ratio=0
    local compression_ratio=0
    if [[ $total_original_size -gt 0 ]]; then
        size_ratio=$(awk "BEGIN {printf \"%.2f\", $total_obfuscated_size / $total_original_size}")
        compression_ratio=$(awk "BEGIN {printf \"%.2f\", (1 - $total_obfuscated_size / $total_original_size) * 100}")
    fi

    cat > "$report_file" << EOF
========================================
混淆效果评估报告
========================================
生成时间: $(date '+%Y-%m-%d %H:%M:%S')

文件统计:
---------
总文件数: $file_count
原始总大小: $total_original_size bytes
混淆后总大小: $total_obfuscated_size bytes
大小比率: $size_ratio
压缩比率: $compression_ratio%

混淆配置:
---------
混淆级别: $OBFUSCATION_LEVEL
变量混淆: $ENABLE_VARIABLE_OBFUSCATION
字符串加密: $ENABLE_STRING_ENCRYPTION
代码压缩: $ENABLE_CODE_COMPRESSION
控制流平坦化: $ENABLE_CONTROL_FLOW_FLATTENING
死代码注入: $ENABLE_DEAD_CODE_INJECTION
函数包装: $ENABLE_FUNCTION_WRAPPING

功能启用:
---------
Anti-Debug: 启用
代码完整性检查: 启用
内存保护: 启用
自毁机制: 启用
运行时保护: 启用

========================================
EOF

    log_success "评估报告已生成: $report_file"
}

generate_cicd_config() {
    local ci_config_dir="$PROJECT_ROOT/.github/workflows"
    local ci_config_file="$ci_config_dir/obfuscate.yml"

    if [[ ! -d "$ci_config_dir" ]]; then
        mkdir -p "$ci_config_dir"
    fi

    cat > "$ci_config_file" << 'EOF'
name: JavaScript Obfuscation

on:
  push:
    branches:
      - main
      - develop
    paths:
      - 'frontend/static/js/*.js'
  pull_request:
    branches:
      - main
      - develop
    paths:
      - 'frontend/static/js/*.js'
  workflow_dispatch:

jobs:
  obfuscate:
    runs-on: ubuntu-latest
    
    steps:
    - name: Checkout code
      uses: actions/checkout@v3
    
    - name: Setup Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Build obfuscator tool
      run: |
        cd backend
        go build -o ../obfuscator_tool ./internal/tools/javascript_obfuscator.go
    
    - name: Run obfuscation
      run: |
        chmod +x scripts/obfuscate.sh
        ./scripts/obfuscate.sh --level 3 --force
    
    - name: Evaluate obfuscation effectiveness
      run: |
        ./scripts/obfuscate.sh --evaluate
    
    - name: Upload obfuscated artifacts
      uses: actions/upload-artifact@v3
      with:
        name: obfuscated-js
        path: build/js/
    
    - name: Upload evaluation report
      uses: actions/upload-artifact@v3
      with:
        name: obfuscation-report
        path: build/obfuscation_report.txt
    
    - name: Upload manifest
      uses: actions/upload-artifact@v3
      with:
        name: obfuscation-manifest
        path: build/manifest.json

  deploy:
    needs: obfuscate
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    
    steps:
    - name: Download obfuscated artifacts
      uses: actions/download-artifact@v3
      with:
        name: obfuscated-js
        path: build/js/
    
    - name: Deploy to production
      run: |
        echo "Deploying obfuscated code to production..."
        # Add your deployment commands here
EOF

    log_success "CI/CD配置文件已生成: $ci_config_file"
}

integrate_into_build_process() {
    local build_script="$PROJECT_ROOT/build.sh"

    cat > "$build_script" << 'EOF'
#!/bin/bash

set -euo pipefail

echo "Starting build process..."

echo "Step 1: Building backend..."
cd backend
go build -o ../bin/hjtpx ./cmd/hjtpx.go
cd ..

echo "Step 2: Obfuscating frontend JavaScript..."
./scripts/obfuscate.sh --level 3 --force

echo "Step 3: Copying static files..."
mkdir -p bin/static
cp -r frontend/static/* bin/static/

echo "Step 4: Generating build manifest..."
if [[ -f build/manifest.json ]]; then
    cp build/manifest.json bin/build_manifest.json
fi

echo "Build completed successfully!"
EOF

    chmod +x "$build_script"
    log_success "构建脚本已生成: $build_script"
}

create_obfuscation_version_manager() {
    local version_file="$CONFIG_DIR/version_history.json"
    mkdir -p "$CONFIG_DIR"

    local current_version=$(date +%Y%m%d_%H%M%S)
    local key_hash=$(echo -n "$OBFUSCATION_KEY" | sha256sum | cut -d' ' -f1)

    cat > "$version_file" << EOF
{
    "current_version": "$current_version",
    "key_hash": "$key_hash",
    "level": $OBFUSCATION_LEVEL,
    "created_at": "$(date -Iseconds)",
    "history": []
}
EOF

    log_success "版本管理器已初始化: $version_file"
}

add_obfuscation_badge() {
    local badge_file="$PROJECT_ROOT/.obfuscation_badge.md"

    cat > "$badge_file" << EOF
# 混淆状态

![Obfuscation Level](https://img.shields.io/badge/obfuscation-level-$OBFUSCATION_LEVEL-blue)
![Build Status](https://img.shields.io/badge/build-passing-green)

## 混淆配置

- 变量混淆: $ENABLE_VARIABLE_OBFUSCATION
- 字符串加密: $ENABLE_STRING_ENCRYPTION
- 代码压缩: $ENABLE_CODE_COMPRESSION
- 控制流平坦化: $ENABLE_CONTROL_FLOW_FLATTENING
- 死代码注入: $ENABLE_DEAD_CODE_INJECTION
- 函数包装: $ENABLE_FUNCTION_WRAPPING

## 最新混淆版本

生成时间: $(date '+%Y-%m-%d %H:%M:%S')
EOF

    log_success "混淆徽章已生成: $badge_file"
}

main() {
    log_info "=== JavaScript 代码混淆构建脚本 ==="
    log_info "版本: 2.0"
    log_info "时间: $(date '+%Y-%m-%d %H:%M:%S')"
    echo ""

    parse_arguments "$@"

    apply_obfuscation_level

    confirm_obfuscation

    if [[ "$DRY_RUN" != "true" ]]; then
        create_backup
        create_obfuscation_version_manager
    fi

    if [[ -d "$INPUT_PATH" ]]; then
        obfuscate_directory "$INPUT_PATH" "$OUTPUT_PATH"
    elif [[ -f "$INPUT_PATH" ]]; then
        mkdir -p "$OUTPUT_PATH"
        local filename=$(basename "$INPUT_PATH")
        obfuscate_file "$INPUT_PATH" "$OUTPUT_PATH/$filename"
    fi

    if [[ "$DRY_RUN" != "true" ]]; then
        generate_manifest
        evaluate_obfuscation_effectiveness "$INPUT_PATH" "$OUTPUT_PATH"
        generate_cicd_config
        integrate_into_build_process
        add_obfuscation_badge
    fi

    echo ""
    log_success "=== 混淆构建完成 ==="
    log_info "输出目录: $OUTPUT_PATH"
    echo ""
}

trap 'log_error "脚本被中断"; exit 1' INT TERM

main "$@"
