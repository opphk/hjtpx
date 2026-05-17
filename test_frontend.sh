#!/bin/bash

echo "========================================"
echo "前端滑块验证码组件验证测试"
echo "========================================"
echo ""

echo "1. 检查文件是否存在..."
files=(
    "/workspace/hjtpx/static/js/captcha.js"
    "/workspace/hjtpx/static/css/style.css"
    "/workspace/hjtpx/templates/index.html"
    "/workspace/hjtpx/templates/demo.html"
)

all_exist=true
for file in "${files[@]}"; do
    if [ -f "$file" ]; then
        echo "   ✓ $file"
    else
        echo "   ✗ $file (不存在)"
        all_exist=false
    fi
done

echo ""
echo "2. 检查文件内容..."

if [ -f "/workspace/hjtpx/static/js/captcha.js" ]; then
    if grep -q "class SliderCaptcha" /workspace/hjtpx/static/js/captcha.js; then
        echo "   ✓ captcha.js 包含 SliderCaptcha 类"
    else
        echo "   ✗ captcha.js 缺少 SliderCaptcha 类"
    fi
    
    if grep -q "generateFingerprint" /workspace/hjtpx/static/js/captcha.js; then
        echo "   ✓ captcha.js 包含指纹生成功能"
    else
        echo "   ✗ captcha.js 缺少指纹生成功能"
    fi
    
    if grep -q "trackData" /workspace/hjtpx/static/js/captcha.js; then
        echo "   ✓ captcha.js 包含轨迹采集功能"
    else
        echo "   ✗ captcha.js 缺少轨迹采集功能"
    fi
fi

if [ -f "/workspace/hjtpx/static/css/style.css" ]; then
    if grep -q ".captcha-wrapper" /workspace/hjtpx/static/css/style.css; then
        echo "   ✓ style.css 包含滑块样式"
    else
        echo "   ✗ style.css 缺少滑块样式"
    fi
    
    if grep -q "@media" /workspace/hjtpx/static/css/style.css; then
        echo "   ✓ style.css 包含响应式设计"
    else
        echo "   ✗ style.css 缺少响应式设计"
    fi
fi

if [ -f "/workspace/hjtpx/templates/index.html" ]; then
    if grep -q "captcha-container" /workspace/hjtpx/templates/index.html; then
        echo "   ✓ index.html 包含验证码容器"
    else
        echo "   ✗ index.html 缺少验证码容器"
    fi
    
    if grep -q "bootstrap" /workspace/hjtpx/templates/index.html; then
        echo "   ✓ index.html 引用 Bootstrap CSS"
    else
        echo "   ✗ index.html 缺少 Bootstrap 引用"
    fi
fi

echo ""
echo "3. 检查Go后端路由配置..."

if [ -f "/workspace/hjtpx/internal/api/router.go" ]; then
    if grep -q 'router.Static("/static"' /workspace/hjtpx/internal/api/router.go; then
        echo "   ✓ router.go 配置了静态文件路由"
    else
        echo "   ✗ router.go 缺少静态文件路由配置"
    fi
    
    if grep -q 'LoadHTMLGlob' /workspace/hjtpx/internal/api/router.go; then
        echo "   ✓ router.go 配置了HTML模板加载"
    else
        echo "   ✗ router.go 缺少HTML模板加载配置"
    fi
fi

echo ""
echo "4. 统计信息..."

js_lines=$(wc -l < /workspace/hjtpx/static/js/captcha.js 2>/dev/null || echo "0")
css_lines=$(wc -l < /workspace/hjtpx/static/css/style.css 2>/dev/null || echo "0")
html1_lines=$(wc -l < /workspace/hjtpx/templates/index.html 2>/dev/null || echo "0")
html2_lines=$(wc -l < /workspace/hjtpx/templates/demo.html 2>/dev/null || echo "0")

echo "   代码行数统计:"
echo "   - captcha.js: $js_lines 行"
echo "   - style.css: $css_lines 行"
echo "   - index.html: $html1_lines 行"
echo "   - demo.html: $html2_lines 行"
echo "   - 总计: $((js_lines + css_lines + html1_lines + html2_lines)) 行"

echo ""
echo "========================================"
echo "验证完成！"
echo "========================================"
echo ""
echo "使用方法:"
echo "1. 启动后端服务: go run cmd/server/main.go"
echo "2. 访问演示页面: http://localhost:8080/"
echo "3. 访问集成示例: http://localhost:8080/demo"
echo ""
echo "注意: 需要先配置数据库和Redis连接"
echo "========================================"
