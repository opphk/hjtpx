#!/bin/bash

# 前端界面优化测试脚本

echo "==========================================="
echo "前端界面优化测试"
echo "==========================================="
echo ""

# 1. 检查CSS文件是否存在
echo "1. 检查CSS优化文件..."
if [ -f "/workspace/frontend/static/css/captcha-ui-optimized.css" ]; then
    echo "   ✓ captcha-ui-optimized.css 存在"
    echo "   文件大小: $(wc -c < /workspace/frontend/static/css/captcha-ui-optimized.css) bytes"
else
    echo "   ✗ captcha-ui-optimized.css 不存在"
fi
echo ""

# 2. 检查JavaScript增强文件是否存在
echo "2. 检查JavaScript增强文件..."
if [ -f "/workspace/frontend/static/js/captcha-ui-enhancer.js" ]; then
    echo "   ✓ captcha-ui-enhancer.js 存在"
    echo "   文件大小: $(wc -c < /workspace/frontend/static/js/captcha-ui-enhancer.js) bytes"
else
    echo "   ✗ captcha-ui-enhancer.js 不存在"
fi
echo ""

# 3. 检查所有模板文件是否引用了优化后的CSS
echo "3. 检查模板文件CSS引用..."
templates=(
    "/workspace/frontend/templates/captcha.html"
    "/workspace/frontend/templates/home.html"
    "/workspace/frontend/templates/lianliankan.html"
    "/workspace/frontend/templates/voice-captcha.html"
    "/workspace/frontend/templates/seamless.html"
    "/workspace/frontend/templates/3dcaptcha.html"
)

for template in "${templates[@]}"; do
    if [ -f "$template" ]; then
        if grep -q "captcha-ui-optimized.css" "$template"; then
            echo "   ✓ $(basename $template) 已引用优化CSS"
        else
            echo "   ✗ $(basename $template) 未引用优化CSS"
        fi
    fi
done
echo ""

# 4. 检查BootCDN资源引用
echo "4. 检查BootCDN资源引用..."
if grep -q "cdn.bootcdn.net" /workspace/frontend/templates/captcha.html; then
    echo "   ✓ captcha.html 正确引用BootCDN资源"
else
    echo "   ✗ captcha.html 未引用BootCDN资源"
fi
echo ""

# 5. 检查CSS内容
echo "5. 检查CSS优化内容..."
if grep -q "\.captcha-slider-container" /workspace/frontend/static/css/captcha-ui-optimized.css; then
    echo "   ✓ 滑块验证UI样式已优化"
else
    echo "   ✗ 滑块验证UI样式未优化"
fi

if grep -q "\.captcha-click-grid" /workspace/frontend/static/css/captcha-ui-optimized.css; then
    echo "   ✓ 点选验证UI样式已优化"
else
    echo "   ✗ 点选验证UI样式未优化"
fi

if grep -q "@keyframes" /workspace/frontend/static/css/captcha-ui-optimized.css; then
    echo "   ✓ 动画效果已添加"
else
    echo "   ✗ 动画效果未添加"
fi

if grep -q "@media.*max-width" /workspace/frontend/static/css/captcha-ui-optimized.css; then
    echo "   ✓ 移动端适配已优化"
else
    echo "   ✗ 移动端适配未优化"
fi

if grep -q "prefers-reduced-motion" /workspace/frontend/static/css/captcha-ui-optimized.css; then
    echo "   ✓ 无障碍功能已优化"
else
    echo "   ✗ 无障碍功能未优化"
fi
echo ""

# 6. 检查JavaScript功能
echo "6. 检查JavaScript增强功能..."
if grep -q "EnhancedSliderCaptcha" /workspace/frontend/static/js/captcha-ui-enhancer.js; then
    echo "   ✓ 增强滑块验证组件已创建"
else
    echo "   ✗ 增强滑块验证组件未创建"
fi

if grep -q "EnhancedClickCaptcha" /workspace/frontend/static/js/captcha-ui-enhancer.js; then
    echo "   ✓ 增强点选验证组件已创建"
else
    echo "   ✗ 增强点选验证组件未创建"
fi

if grep -q "showSuccessCelebration" /workspace/frontend/static/js/captcha-ui-enhancer.js; then
    echo "   ✓ 成功庆祝动画已添加"
else
    echo "   ✗ 成功庆祝动画未添加"
fi

if grep -q "CDNResourceMonitor" /workspace/frontend/static/js/captcha-ui-enhancer.js; then
    echo "   ✓ BootCDN资源监控已添加"
else
    echo "   ✗ BootCDN资源监控未添加"
fi
echo ""

echo "==========================================="
echo "测试完成"
echo "==========================================="
