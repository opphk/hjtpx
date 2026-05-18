#!/bin/bash
# HJTPX 快速测试脚本 v15.0
# 用于快速验证系统基本功能

set -e

echo "========================================="
echo "  HJTPX 行为验证系统 - 快速测试脚本"
echo "  版本: v15.0"
echo "========================================="
echo ""

# 默认配置
API_BASE="${API_BASE:-http://localhost:8080}"
APP_ID="${APP_ID:-test_app}"
APP_KEY="${APP_KEY:-test_key}"
OUTPUT_DIR="./test_results"

# 创建输出目录
mkdir -p "$OUTPUT_DIR"

echo "配置信息:"
echo "  API地址: $API_BASE"
echo "  应用ID: $APP_ID"
echo "  输出目录: $OUTPUT_DIR"
echo ""

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[!]${NC} $1"
}

print_error() {
    echo -e "${RED}[✗]${NC} $1"
}

print_info() {
    echo -e "[*] $1"
}

# 检查curl是否可用
if ! command -v curl &> /dev/null; then
    print_error "curl未安装，请先安装curl"
    exit 1
fi

# 检查jq是否可用（可选，用于美化JSON输出）
HAS_JQ=true
if ! command -v jq &> /dev/null; then
    print_warning "jq未安装，JSON输出将不美化"
    HAS_JQ=false
fi

# 1. 健康检查
echo ""
print_info "步骤1: 健康检查"
HEALTH_CHECK_URL="$API_BASE/health"

if $HAS_JQ; then
    HEALTH_RESPONSE=$(curl -s -w "\n%{http_code}" "$HEALTH_CHECK_URL")
else
    HEALTH_RESPONSE=$(curl -s -w "\n%{http_code}" "$HEALTH_CHECK_URL")
fi

HTTP_CODE=$(echo "$HEALTH_RESPONSE" | tail -n 1)
RESPONSE_BODY=$(echo "$HEALTH_RESPONSE" | head -n -1)

if [ "$HTTP_CODE" -eq 200 ]; then
    print_success "健康检查通过"
    if $HAS_JQ; then
        echo "$RESPONSE_BODY" | jq . > "$OUTPUT_DIR/health_check.json"
    else
        echo "$RESPONSE_BODY" > "$OUTPUT_DIR/health_check.json"
    fi
    print_info "结果已保存到: $OUTPUT_DIR/health_check.json"
else
    print_error "健康检查失败 (HTTP $HTTP_CODE)"
    echo "$RESPONSE_BODY" > "$OUTPUT_DIR/health_check_error.json"
    exit 1
fi

# 2. 测试滑块验证码生成
echo ""
print_info "步骤2: 测试滑块验证码生成"
SLIDER_GEN_URL="$API_BASE/api/v1/captcha/slider/generate"

SLIDER_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SLIDER_GEN_URL" \
    -H "Content-Type: application/json" \
    -d "{\"app_id\":\"$APP_ID\",\"app_key\":\"$APP_KEY\"}")

HTTP_CODE=$(echo "$SLIDER_RESPONSE" | tail -n 1)
RESPONSE_BODY=$(echo "$SLIDER_RESPONSE" | head -n -1)

if [ "$HTTP_CODE" -eq 200 ]; then
    print_success "滑块验证码生成成功"
    if $HAS_JQ; then
        echo "$RESPONSE_BODY" | jq . > "$OUTPUT_DIR/slider_generate.json"
    else
        echo "$RESPONSE_BODY" > "$OUTPUT_DIR/slider_generate.json"
    fi
    
    # 提取session_id用于后续验证
    if $HAS_JQ; then
        SESSION_ID=$(echo "$RESPONSE_BODY" | jq -r '.data.session_id // .captcha_id // ""')
    else
        # 简单的JSON解析尝试提取session_id
        SESSION_ID=$(echo "$RESPONSE_BODY" | grep -o '"session_id":"[^"]*"' | cut -d'"' -f4)
        if [ -z "$SESSION_ID" ]; then
            SESSION_ID=$(echo "$RESPONSE_BODY" | grep -o '"captcha_id":"[^"]*"' | cut -d'"' -f4)
        fi
    fi
    
    print_info "结果已保存到: $OUTPUT_DIR/slider_generate.json"
    if [ -n "$SESSION_ID" ]; then
        print_info "获取到的Session ID: $SESSION_ID"
    else
        print_warning "未能提取Session ID，跳过验证测试"
    fi
else
    print_warning "滑块验证码生成可能失败 (HTTP $HTTP_CODE)"
    echo "$RESPONSE_BODY" > "$OUTPUT_DIR/slider_generate_error.json"
fi

# 3. 测试点选验证码生成
echo ""
print_info "步骤3: 测试点选验证码生成"
CLICK_GEN_URL="$API_BASE/api/v1/captcha/click/generate"

CLICK_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$CLICK_GEN_URL" \
    -H "Content-Type: application/json" \
    -d "{\"app_id\":\"$APP_ID\",\"app_key\":\"$APP_KEY\",\"mode\":\"number\",\"points\":3}")

HTTP_CODE=$(echo "$CLICK_RESPONSE" | tail -n 1)
RESPONSE_BODY=$(echo "$CLICK_RESPONSE" | head -n -1)

if [ "$HTTP_CODE" -eq 200 ]; then
    print_success "点选验证码生成成功"
    if $HAS_JQ; then
        echo "$RESPONSE_BODY" | jq . > "$OUTPUT_DIR/click_generate.json"
    else
        echo "$RESPONSE_BODY" > "$OUTPUT_DIR/click_generate.json"
    fi
    print_info "结果已保存到: $OUTPUT_DIR/click_generate.json"
else
    print_warning "点选验证码生成可能失败 (HTTP $HTTP_CODE)"
    echo "$RESPONSE_BODY" > "$OUTPUT_DIR/click_generate_error.json"
fi

# 4. 测试图形验证码生成
echo ""
print_info "步骤4: 测试图形验证码生成"
IMAGE_GEN_URL="$API_BASE/api/v1/captcha/image?type=mixed&count=4"

IMAGE_RESPONSE=$(curl -s -w "\n%{http_code}" "$IMAGE_GEN_URL")

HTTP_CODE=$(echo "$IMAGE_RESPONSE" | tail -n 1)
RESPONSE_BODY=$(echo "$IMAGE_RESPONSE" | head -n -1)

if [ "$HTTP_CODE" -eq 200 ]; then
    print_success "图形验证码生成成功"
    if $HAS_JQ; then
        echo "$RESPONSE_BODY" | jq . > "$OUTPUT_DIR/image_generate.json"
    else
        echo "$RESPONSE_BODY" > "$OUTPUT_DIR/image_generate.json"
    fi
    print_info "结果已保存到: $OUTPUT_DIR/image_generate.json"
else
    print_warning "图形验证码生成可能失败 (HTTP $HTTP_CODE)"
    echo "$RESPONSE_BODY" > "$OUTPUT_DIR/image_generate_error.json"
fi

# 5. 测试环境检测接口（v15.0新增）
echo ""
print_info "步骤5: 测试环境检测接口 (v15.0新增)"
DETECT_URL="$API_BASE/api/v1/detect/enhanced"

DETECT_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$DETECT_URL" \
    -H "Content-Type: application/json" \
    -d '{"fingerprint":{"canvas":"test_canvas","webgl":"test_webgl","fonts":["Arial","Times New Roman"],"plugins":["Plugin 1","Plugin 2"]},"behavior_signals":{"mouse_movements":[],"keyboard_events":[]}}')

HTTP_CODE=$(echo "$DETECT_RESPONSE" | tail -n 1)
RESPONSE_BODY=$(echo "$DETECT_RESPONSE" | head -n -1)

if [ "$HTTP_CODE" -eq 200 ]; then
    print_success "环境检测接口正常"
    if $HAS_JQ; then
        echo "$RESPONSE_BODY" | jq . > "$OUTPUT_DIR/detect_enhanced.json"
    else
        echo "$RESPONSE_BODY" > "$OUTPUT_DIR/detect_enhanced.json"
    fi
    print_info "结果已保存到: $OUTPUT_DIR/detect_enhanced.json"
else
    print_warning "环境检测接口可能失败 (HTTP $HTTP_CODE)"
    echo "$RESPONSE_BODY" > "$OUTPUT_DIR/detect_enhanced_error.json"
fi

# 6. 测试缓存状态接口（v15.0新增）
echo ""
print_info "步骤6: 测试缓存状态接口 (v15.0新增 - 需要认证)"
CACHE_STATUS_URL="$API_BASE/api/v1/cache/status"

# 这个接口需要认证，可能会失败，这是预期的
CACHE_RESPONSE=$(curl -s -w "\n%{http_code}" "$CACHE_STATUS_URL")

HTTP_CODE=$(echo "$CACHE_RESPONSE" | tail -n 1)
RESPONSE_BODY=$(echo "$CACHE_RESPONSE" | head -n -1)

if [ "$HTTP_CODE" -eq 401 ] || [ "$HTTP_CODE" -eq 403 ]; then
    print_info "缓存状态接口返回预期的认证失败 (HTTP $HTTP_CODE) - 正常"
    if $HAS_JQ; then
        echo "$RESPONSE_BODY" | jq . > "$OUTPUT_DIR/cache_status.json"
    else
        echo "$RESPONSE_BODY" > "$OUTPUT_DIR/cache_status.json"
    fi
elif [ "$HTTP_CODE" -eq 200 ]; then
    print_success "缓存状态接口正常"
    if $HAS_JQ; then
        echo "$RESPONSE_BODY" | jq . > "$OUTPUT_DIR/cache_status.json"
    else
        echo "$RESPONSE_BODY" > "$OUTPUT_DIR/cache_status.json"
    fi
else
    print_warning "缓存状态接口返回意外状态 (HTTP $HTTP_CODE)"
    echo "$RESPONSE_BODY" > "$OUTPUT_DIR/cache_status_error.json"
fi

# 7. 测试系统指标接口（v15.0新增）
echo ""
print_info "步骤7: 测试系统指标接口 (v15.0新增 - 需要认证)"
METRICS_URL="$API_BASE/api/v1/system/metrics"

METRICS_RESPONSE=$(curl -s -w "\n%{http_code}" "$METRICS_URL")

HTTP_CODE=$(echo "$METRICS_RESPONSE" | tail -n 1)
RESPONSE_BODY=$(echo "$METRICS_RESPONSE" | head -n -1)

if [ "$HTTP_CODE" -eq 401 ] || [ "$HTTP_CODE" -eq 403 ]; then
    print_info "系统指标接口返回预期的认证失败 (HTTP $HTTP_CODE) - 正常"
    if $HAS_JQ; then
        echo "$RESPONSE_BODY" | jq . > "$OUTPUT_DIR/system_metrics.json"
    else
        echo "$RESPONSE_BODY" > "$OUTPUT_DIR/system_metrics.json"
    fi
elif [ "$HTTP_CODE" -eq 200 ]; then
    print_success "系统指标接口正常"
    if $HAS_JQ; then
        echo "$RESPONSE_BODY" | jq . > "$OUTPUT_DIR/system_metrics.json"
    else
        echo "$RESPONSE_BODY" > "$OUTPUT_DIR/system_metrics.json"
    fi
else
    print_warning "系统指标接口返回意外状态 (HTTP $HTTP_CODE)"
    echo "$RESPONSE_BODY" > "$OUTPUT_DIR/system_metrics_error.json"
fi

# 测试总结
echo ""
echo "========================================="
echo "  测试总结"
echo "========================================="
echo ""
print_info "测试已完成！所有结果已保存到: $OUTPUT_DIR"
echo ""
echo "文件列表:"
ls -la "$OUTPUT_DIR"
echo ""
print_success "快速测试脚本执行完毕"
echo ""
echo "提示:"
echo "  - 需要测试更多功能，请查看 docs/API接口文档.md"
echo "  - 需要运行完整测试，请查看 Makefile 中的 test 目标"
echo "  - 如需配置环境变量: API_BASE, APP_ID, APP_KEY"
echo ""

exit 0
