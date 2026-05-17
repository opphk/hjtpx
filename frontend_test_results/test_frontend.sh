#!/bin/bash

echo "=========================================="
echo "前端测试报告 - $(date)"
echo "=========================================="
echo ""

BASE_URL="http://localhost:8080"
RESULTS_FILE="/tmp/test_results/test_results_$(date +%Y%m%d_%H%M%S).txt"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试函数
test_page() {
    local name=$1
    local url=$2
    
    echo -n "测试 [$name]... "
    response=$(curl -s -o /dev/null -w "%{http_code}" "$url")
    if [ "$response" = "200" ]; then
        echo -e "${GREEN}通过 (HTTP $response)${NC}"
        echo "[✓] $name - 通过 (HTTP $response)" >> "$RESULTS_FILE"
        return 0
    else
        echo -e "${RED}失败 (HTTP $response)${NC}"
        echo "[✗] $name - 失败 (HTTP $response)" >> "$RESULTS_FILE"
        return 1
    fi
}

# 测试JS和CSS资源
test_resource() {
    local name=$1
    local url=$2
    
    echo -n "  - 资源 [$name]... "
    response=$(curl -s -o /dev/null -w "%{http_code}" "$url")
    if [ "$response" = "200" ]; then
        echo -e "${GREEN}✓${NC}"
        return 0
    else
        echo -e "${RED}✗${NC}"
        return 1
    fi
}

# 开始测试
echo "1. 页面访问测试" | tee -a "$RESULTS_FILE"
echo "-------------------------------------------" | tee -a "$RESULTS_FILE"

# 前端页面
test_page "验证码演示页" "$BASE_URL/frontend/templates/captcha.html"
test_page "首页" "$BASE_URL/frontend/templates/home.html"
test_page "无缝验证页" "$BASE_URL/frontend/templates/seamless.html"

# 管理后台页面
test_page "管理后台登录页" "$BASE_URL/admin/templates/login.html"
test_page "管理后台仪表盘" "$BASE_URL/admin/templates/dashboard.html"
test_page "管理后台配置页" "$BASE_URL/admin/templates/config.html"
test_page "管理后台日志页" "$BASE_URL/admin/templates/logs.html"

# 开发工具页面
test_page "验证码测试工具" "$BASE_URL/devtools/templates/captcha-test.html"
test_page "开发工具文档" "$BASE_URL/devtools/templates/docs.html"
test_page "API控制台" "$BASE_URL/devtools/templates/api-console.html"

echo ""
echo "2. 静态资源测试" | tee -a "$RESULTS_FILE"
echo "-------------------------------------------" | tee -a "$RESULTS_FILE"

# JS资源
test_resource "jQuery" "https://cdn.bootcdn.net/ajax/libs/jquery/3.7.1/jquery.min.js"
test_resource "Bootstrap CSS" "https://cdn.bootcdn.net/ajax/libs/twitter-bootstrap/5.3.8/css/bootstrap.min.css"
test_resource "Font Awesome" "https://cdn.bootcdn.net/ajax/libs/font-awesome/6.5.1/css/all.min.css"

echo ""
echo "3. 响应式布局测试 (通过代码分析)" | tee -a "$RESULTS_FILE"
echo "-------------------------------------------" | tee -a "$RESULTS_FILE"

# 检查viewport meta标签
if grep -q "viewport" "$BASE_URL/frontend/templates/captcha.html" 2>/dev/null || curl -s "$BASE_URL/frontend/templates/captcha.html" | grep -q "viewport"; then
    echo -e "${GREEN}[✓] 验证码页面包含viewport meta标签${NC}"
    echo "[✓] 验证码页面包含viewport meta标签" >> "$RESULTS_FILE"
else
    echo -e "${RED}[✗] 验证码页面缺少viewport meta标签${NC}"
    echo "[✗] 验证码页面缺少viewport meta标签" >> "$RESULTS_FILE"
fi

# 检查响应式CSS类
if curl -s "$BASE_URL/frontend/templates/captcha.html" | grep -qE "(container|row|col-|d-none|d-md-block)"; then
    echo -e "${GREEN}[✓] 验证码页面使用响应式Bootstrap类${NC}"
    echo "[✓] 验证码页面使用响应式Bootstrap类" >> "$RESULTS_FILE"
else
    echo -e "${RED}[✗] 验证码页面未使用响应式Bootstrap类${NC}"
    echo "[✗] 验证码页面未使用响应式Bootstrap类" >> "$RESULTS_FILE"
fi

# 检查移动端meta标签
if curl -s "$BASE_URL/frontend/templates/captcha.html" | grep -qE "(apple-mobile-web-app-capable|mobile-web-app-capable)"; then
    echo -e "${GREEN}[✓] 验证码页面包含移动端PWA标签${NC}"
    echo "[✓] 验证码页面包含移动端PWA标签" >> "$RESULTS_FILE"
else
    echo -e "${RED}[✗] 验证码页面缺少移动端PWA标签${NC}"
    echo "[✗] 验证码页面缺少移动端PWA标签" >> "$RESULTS_FILE"
fi

echo ""
echo "4. 浏览器兼容性检查" | tee -a "$RESULTS_FILE"
echo "-------------------------------------------" | tee -a "$RESULTS_FILE"

# 检查HTML5语义标签
if curl -s "$BASE_URL/frontend/templates/captcha.html" | grep -qE "(<header|<nav|<main|<section|<article|<footer|<aside)"; then
    echo -e "${GREEN}[✓] 使用HTML5语义化标签${NC}"
    echo "[✓] 使用HTML5语义化标签" >> "$RESULTS_FILE"
else
    echo -e "${YELLOW}[⚠] 未使用HTML5语义化标签${NC}"
    echo "[⚠] 未使用HTML5语义化标签" >> "$RESULTS_FILE"
fi

# 检查CSS兼容性
if curl -s "$BASE_URL/frontend/templates/captcha.html" | grep -qE "(-webkit-|-moz-|-ms-|-o-)"; then
    echo -e "${GREEN}[✓] 包含浏览器前缀（webkit, moz, ms, o）${NC}"
    echo "[✓] 包含浏览器前缀" >> "$RESULTS_FILE"
else
    echo -e "${YELLOW}[⚠] 未使用CSS浏览器前缀${NC}"
    echo "[⚠] 未使用CSS浏览器前缀" >> "$RESULTS_FILE"
fi

echo ""
echo "5. JavaScript功能检查" | tee -a "$RESULTS_FILE"
echo "-------------------------------------------" | tee -a "$RESULTS_FILE"

# 检查JS引用
if curl -s "$BASE_URL/frontend/templates/captcha.html" | grep -qE "<script"; then
    script_count=$(curl -s "$BASE_URL/frontend/templates/captcha.html" | grep -c "<script")
    echo -e "${GREEN}[✓] 验证码页面包含 $script_count 个script标签${NC}"
    echo "[✓] 验证码页面包含 $script_count 个script标签" >> "$RESULTS_FILE"
else
    echo -e "${RED}[✗] 验证码页面缺少script标签${NC}"
    echo "[✗] 验证码页面缺少script标签" >> "$RESULTS_FILE"
fi

# 检查defer/async属性
if curl -s "$BASE_URL/frontend/templates/captcha.html" | grep -qE "(defer|async)"; then
    echo -e "${GREEN}[✓] 使用脚本延迟加载（defer/async）${NC}"
    echo "[✓] 使用脚本延迟加载" >> "$RESULTS_FILE"
else
    echo -e "${YELLOW}[⚠] 未使用脚本延迟加载${NC}"
    echo "[⚠] 未使用脚本延迟加载" >> "$RESULTS_FILE"
fi

echo ""
echo "6. 安全性检查" | tee -a "$RESULTS_FILE"
echo "-------------------------------------------" | tee -a "$RESULTS_FILE"

# 检查HTTPS CDN
if curl -s "$BASE_URL/frontend/templates/captcha.html" | grep -q "https://"; then
    echo -e "${GREEN}[✓] 使用HTTPS CDN资源${NC}"
    echo "[✓] 使用HTTPS CDN资源" >> "$RESULTS_FILE"
else
    echo -e "${YELLOW}[⚠] 未使用HTTPS CDN${NC}"
    echo "[⚠] 未使用HTTPS CDN" >> "$RESULTS_FILE"
fi

# 检查CSP meta标签
if curl -s "$BASE_URL/frontend/templates/captcha.html" | grep -qE "Content-Security-Policy"; then
    echo -e "${GREEN}[✓] 包含CSP安全策略${NC}"
    echo "[✓] 包含CSP安全策略" >> "$RESULTS_FILE"
else
    echo -e "${YELLOW}[⚠] 未包含CSP安全策略${NC}"
    echo "[⚠] 未包含CSP安全策略" >> "$RESULTS_FILE"
fi

# 检查X-UA-Compatible
if curl -s "$BASE_URL/frontend/templates/captcha.html" | grep -qE "X-UA-Compatible"; then
    echo -e "${GREEN}[✓] 包含IE兼容模式设置${NC}"
    echo "[✓] 包含IE兼容模式设置" >> "$RESULTS_FILE"
else
    echo -e "${YELLOW}[⚠] 未包含IE兼容模式设置${NC}"
    echo "[⚠] 未包含IE兼容模式设置" >> "$RESULTS_FILE"
fi

echo ""
echo "7. SEO和可访问性检查" | tee -a "$RESULTS_FILE"
echo "-------------------------------------------" | tee -a "$RESULTS_FILE"

# 检查meta描述
if curl -s "$BASE_URL/frontend/templates/captcha.html" | grep -qE '<meta name="description"'; then
    echo -e "${GREEN}[✓] 包含meta描述${NC}"
    echo "[✓] 包含meta描述" >> "$RESULTS_FILE"
else
    echo -e "${RED}[✗] 缺少meta描述${NC}"
    echo "[✗] 缺少meta描述" >> "$RESULTS_FILE"
fi

# 检查lang属性
if curl -s "$BASE_URL/frontend/templates/captcha.html" | grep -qE '<html.*lang='; then
    echo -e "${GREEN}[✓] HTML包含lang属性${NC}"
    echo "[✓] HTML包含lang属性" >> "$RESULTS_FILE"
else
    echo -e "${RED}[✗] HTML缺少lang属性${NC}"
    echo "[✗] HTML缺少lang属性" >> "$RESULTS_FILE"
fi

# 检查charset
if curl -s "$BASE_URL/frontend/templates/captcha.html" | grep -qE '<meta charset='; then
    echo -e "${GREEN}[✓] 包含字符集声明${NC}"
    echo "[✓] 包含字符集声明" >> "$RESULTS_FILE"
else
    echo -e "${RED}[✗] 缺少字符集声明${NC}"
    echo "[✗] 缺少字符集声明" >> "$RESULTS_FILE"
fi

echo ""
echo "===========================================" | tee -a "$RESULTS_FILE"
echo "测试完成 - 结果已保存到 $RESULTS_FILE" | tee -a "$RESULTS_FILE"
echo "===========================================" | tee -a "$RESULTS_FILE"

