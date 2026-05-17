#!/bin/bash

echo "=========================================="
echo "高级前端测试 - $(date)"
echo "=========================================="
echo ""

BASE_URL="http://localhost:8080"
RESULTS_FILE="/tmp/test_results/test_advanced_$(date +%Y%m%d_%H%M%S).txt"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "8. 验证码功能组件检查" | tee -a "$RESULTS_FILE"
echo "-------------------------------------------" | tee -a "$RESULTS_FILE"

# 获取验证码页面HTML
CAPTCHA_HTML=$(curl -s "$BASE_URL/frontend/templates/captcha.html")

# 检查滑块验证组件
if echo "$CAPTCHA_HTML" | grep -qE "(slider|drag|滑块)"; then
    echo -e "${GREEN}[✓] 包含滑块验证组件${NC}"
    echo "[✓] 包含滑块验证组件" >> "$RESULTS_FILE"
else
    echo -e "${YELLOW}[⚠] 未检测到滑块验证组件${NC}"
    echo "[⚠] 未检测到滑块验证组件" >> "$RESULTS_FILE"
fi

# 检查点选验证组件
if echo "$CAPTCHA_HTML" | grep -qE "(click|select|点选)"; then
    echo -e "${GREEN}[✓] 包含点选验证组件${NC}"
    echo "[✓] 包含点选验证组件" >> "$RESULTS_FILE"
else
    echo -e "${YELLOW}[⚠] 未检测到点选验证组件${NC}"
    echo "[⚠] 未检测到点选验证组件" >> "$RESULTS_FILE"
fi

# 检查无感验证组件
if echo "$CAPTCHA_HTML" | grep -qE "(seamless|invisible|无感)"; then
    echo -e "${GREEN}[✓] 包含无感验证组件${NC}"
    echo "[✓] 包含无感验证组件" >> "$RESULTS_FILE"
else
    echo -e "${YELLOW}[⚠] 未检测到无感验证组件${NC}"
    echo "[⚠] 未检测到无感验证组件" >> "$RESULTS_FILE"
fi

# 检查验证结果展示
if echo "$CAPTCHA_HTML" | grep -qE "(success|fail|result|验证)"; then
    echo -e "${GREEN}[✓] 包含验证结果展示${NC}"
    echo "[✓] 包含验证结果展示" >> "$RESULTS_FILE"
else
    echo -e "${RED}[✗] 缺少验证结果展示${NC}"
    echo "[✗] 缺少验证结果展示" >> "$RESULTS_FILE"
fi

# 检查刷新按钮
if echo "$CAPTCHA_HTML" | grep -qE "(refresh|reload|刷新)"; then
    echo -e "${GREEN}[✓] 包含刷新按钮${NC}"
    echo "[✓] 包含刷新按钮" >> "$RESULTS_FILE"
else
    echo -e "${YELLOW}[⚠] 未检测到刷新按钮${NC}"
    echo "[⚠] 未检测到刷新按钮" >> "$RESULTS_FILE"
fi

echo ""
echo "9. CSS样式和动画检查" | tee -a "$RESULTS_FILE"
echo "-------------------------------------------" | tee -a "$RESULTS_FILE"

# 检查CSS变量（主题）
if echo "$CAPTCHA_HTML" | grep -qE ":root\s*\{"; then
    echo -e "${GREEN}[✓] 使用CSS变量定义主题${NC}"
    echo "[✓] 使用CSS变量定义主题" >> "$RESULTS_FILE"
else
    echo -e "${YELLOW}[⚠] 未使用CSS变量${NC}"
    echo "[⚠] 未使用CSS变量" >> "$RESULTS_FILE"
fi

# 检查过渡动画
if echo "$CAPTCHA_HTML" | grep -qE "(transition|animation|@keyframes)"; then
    echo -e "${GREEN}[✓] 包含过渡和动画效果${NC}"
    echo "[✓] 包含过渡和动画效果" >> "$RESULTS_FILE"
else
    echo -e "${YELLOW}[⚠] 未使用动画效果${NC}"
    echo "[⚠] 未使用动画效果" >> "$RESULTS_FILE"
fi

# 检查Flexbox/Grid布局
if echo "$CAPTCHA_HTML" | grep -qE "(display:\s*(flex|grid)|display:\s*-webkit-flex)"; then
    echo -e "${GREEN}[✓] 使用现代CSS布局（Flexbox/Grid）${NC}"
    echo "[✓] 使用现代CSS布局" >> "$RESULTS_FILE"
else
    echo -e "${YELLOW}[⚠] 未使用Flexbox/Grid布局${NC}"
    echo "[⚠] 未使用Flexbox/Grid布局" >> "$RESULTS_FILE"
fi

# 检查阴影和圆角
if echo "$CAPTCHA_HTML" | grep -qE "(box-shadow|border-radius)"; then
    echo -e "${GREEN}[✓] 使用阴影和圆角美化${NC}"
    echo "[✓] 使用阴影和圆角美化" >> "$RESULTS_FILE"
else
    echo -e "${YELLOW}[⚠] 未使用阴影和圆角${NC}"
    echo "[⚠] 未使用阴影和圆角" >> "$RESULTS_FILE"
fi

echo ""
echo "10. 管理后台功能检查" | tee -a "$RESULTS_FILE"
echo "-------------------------------------------" | tee -a "$RESULTS_FILE"

# 获取管理后台HTML
ADMIN_HTML=$(curl -s "$BASE_URL/admin/templates/dashboard.html")

# 检查登录表单
if echo "$ADMIN_HTML" | grep -qE "<form"; then
    echo -e "${GREEN}[✓] 包含表单元素${NC}"
    echo "[✓] 包含表单元素" >> "$RESULTS_FILE"
else
    echo -e "${RED}[✗] 缺少表单元素${NC}"
    echo "[✗] 缺少表单元素" >> "$RESULTS_FILE"
fi

# 检查输入框
if echo "$ADMIN_HTML" | grep -qE "<input"; then
    echo -e "${GREEN}[✓] 包含输入框${NC}"
    echo "[✓] 包含输入框" >> "$RESULTS_FILE"
else
    echo -e "${RED}[✗] 缺少输入框${NC}"
    echo "[✗] 缺少输入框" >> "$RESULTS_FILE"
fi

# 检查按钮
if echo "$ADMIN_HTML" | grep -qE "<button"; then
    echo -e "${GREEN}[✓] 包含按钮${NC}"
    echo "[✓] 包含按钮" >> "$RESULTS_FILE"
else
    echo -e "${RED}[✗] 缺少按钮${NC}"
    echo "[✗] 缺少按钮" >> "$RESULTS_FILE"
fi

# 检查数据表格
if echo "$ADMIN_HTML" | grep -qE "(table|tbody|thead)"; then
    echo -e "${GREEN}[✓] 包含数据表格${NC}"
    echo "[✓] 包含数据表格" >> "$RESULTS_FILE"
else
    echo -e "${YELLOW}[⚠] 未检测到数据表格${NC}"
    echo "[⚠] 未检测到数据表格" >> "$RESULTS_FILE"
fi

# 检查图表引用
if echo "$ADMIN_HTML" | grep -qE "(echarts|chart|graph)"; then
    echo -e "${GREEN}[✓] 包含图表库引用${NC}"
    echo "[✓] 包含图表库引用" >> "$RESULTS_FILE"
else
    echo -e "${YELLOW}[⚠] 未检测到图表库${NC}"
    echo "[⚠] 未检测到图表库" >> "$RESULTS_FILE"
fi

echo ""
echo "11. 响应式断点检查" | tee -a "$RESULTS_FILE"
echo "-------------------------------------------" | tee -a "$RESULTS_FILE"

# 检查媒体查询
if echo "$CAPTCHA_HTML" | grep -qE "@media"; then
    media_queries=$(echo "$CAPTCHA_HTML" | grep -oE "@media[^{]*" | wc -l)
    echo -e "${GREEN}[✓] 包含 $media_queries 个媒体查询${NC}"
    echo "[✓] 包含 $media_queries 个媒体查询" >> "$RESULTS_FILE"
    
    # 检查常见断点
    if echo "$CAPTCHA_HTML" | grep -qE "(max-width:\s*768px|min-width:\s*768px)"; then
        echo -e "${GREEN}[✓] 包含平板设备断点 (768px)${NC}"
        echo "[✓] 包含平板设备断点" >> "$RESULTS_FILE"
    fi
    
    if echo "$CAPTCHA_HTML" | grep -qE "(max-width:\s*576px|min-width:\s*576px)"; then
        echo -e "${GREEN}[✓] 包含手机设备断点 (576px)${NC}"
        echo "[✓] 包含手机设备断点" >> "$RESULTS_FILE"
    fi
    
    if echo "$CAPTCHA_HTML" | grep -qE "(max-width:\s*992px|min-width:\s*992px)"; then
        echo -e "${GREEN}[✓] 包含桌面设备断点 (992px)${NC}"
        echo "[✓] 包含桌面设备断点" >> "$RESULTS_FILE"
    fi
else
    echo -e "${YELLOW}[⚠] 未使用媒体查询${NC}"
    echo "[⚠] 未使用媒体查询" >> "$RESULTS_FILE"
fi

echo ""
echo "12. 无障碍访问检查" | tee -a "$RESULTS_FILE"
echo "-------------------------------------------" | tee -a "$RESULTS_FILE"

# 检查ARIA属性
if echo "$CAPTCHA_HTML" | grep -qE "aria-"; then
    echo -e "${GREEN}[✓] 使用ARIA无障碍属性${NC}"
    echo "[✓] 使用ARIA无障碍属性" >> "$RESULTS_FILE"
else
    echo -e "${YELLOW}[⚠] 未使用ARIA属性${NC}"
    echo "[⚠] 未使用ARIA属性" >> "$RESULTS_FILE"
fi

# 检查role属性
if echo "$CAPTCHA_HTML" | grep -qE "role="; then
    echo -e "${GREEN}[✓] 使用role属性${NC}"
    echo "[✓] 使用role属性" >> "$RESULTS_FILE"
else
    echo -e "${YELLOW}[⚠] 未使用role属性${NC}"
    echo "[⚠] 未使用role属性" >> "$RESULTS_FILE"
fi

# 检查alt文本
if echo "$CAPTCHA_HTML" | grep -qE "<img[^>]*alt="; then
    echo -e "${GREEN}[✓] 图片包含alt文本${NC}"
    echo "[✓] 图片包含alt文本" >> "$RESULTS_FILE"
else
    echo -e "${YELLOW}[⚠] 未检测到图片alt文本${NC}"
    echo "[⚠] 未检测到图片alt文本" >> "$RESULTS_FILE"
fi

# 检查label标签
if echo "$ADMIN_HTML" | grep -qE "<label"; then
    echo -e "${GREEN}[✓] 表单包含label标签${NC}"
    echo "[✓] 表单包含label标签" >> "$RESULTS_FILE"
else
    echo -e "${YELLOW}[⚠] 表单未使用label标签${NC}"
    echo "[⚠] 表单未使用label标签" >> "$RESULTS_FILE"
fi

echo ""
echo "13. 性能优化检查" | tee -a "$RESULTS_FILE"
echo "-------------------------------------------" | tee -a "$RESULTS_FILE"

# 检查preconnect
if echo "$CAPTCHA_HTML" | grep -qE "preconnect"; then
    echo -e "${GREEN}[✓] 使用preconnect优化连接${NC}"
    echo "[✓] 使用preconnect优化连接" >> "$RESULTS_FILE"
else
    echo -e "${YELLOW}[⚠] 未使用preconnect${NC}"
    echo "[⚠] 未使用preconnect" >> "$RESULTS_FILE"
fi

# 检查preload
if echo "$CAPTCHA_HTML" | grep -qE "preload"; then
    echo -e "${GREEN}[✓] 使用preload预加载资源${NC}"
    echo "[✓] 使用preload预加载资源" >> "$RESULTS_FILE"
else
    echo -e "${YELLOW}[⚠] 未使用preload${NC}"
    echo "[⚠] 未使用preload" >> "$RESULTS_FILE"
fi

# 检查defer/async
if echo "$CAPTCHA_HTML" | grep -qE "defer"; then
    echo -e "${GREEN}[✓] 使用defer延迟脚本加载${NC}"
    echo "[✓] 使用defer延迟脚本加载" >> "$RESULTS_FILE"
else
    echo -e "${YELLOW}[⚠] 未使用defer${NC}"
    echo "[⚠] 未使用defer" >> "$RESULTS_FILE"
fi

# 检查内联CSS
if echo "$CAPTCHA_HTML" | grep -qE "<style>"; then
    echo -e "${GREEN}[✓] 使用关键CSS内联${NC}"
    echo "[✓] 使用关键CSS内联" >> "$RESULTS_FILE"
else
    echo -e "${YELLOW}[⚠] 未使用内联CSS${NC}"
    echo "[⚠] 未使用内联CSS" >> "$RESULTS_FILE"
fi

echo ""
echo "14. JavaScript代码质量检查" | tee -a "$RESULTS_FILE"
echo "-------------------------------------------" | tee -a "$RESULTS_FILE"

# 检查环境检测代码
if echo "$CAPTCHA_HTML" | grep -qE "environment-detector|envDetector"; then
    echo -e "${GREEN}[✓] 包含环境检测功能${NC}"
    echo "[✓] 包含环境检测功能" >> "$RESULTS_FILE"
else
    echo -e "${YELLOW}[⚠] 未检测到环境检测${NC}"
    echo "[⚠] 未检测到环境检测" >> "$RESULTS_FILE"
fi

# 检查事件监听
if echo "$CAPTCHA_HTML" | grep -qE "(addEventListener|onclick|onload)"; then
    echo -e "${GREEN}[✓] 使用事件监听器${NC}"
    echo "[✓] 使用事件监听器" >> "$RESULTS_FILE"
else
    echo -e "${YELLOW}[⚠] 未使用标准事件监听${NC}"
    echo "[⚠] 未使用标准事件监听" >> "$RESULTS_FILE"
fi

# 检查Promise/async
if echo "$CAPTCHA_HTML" | grep -qE "(Promise|async|await|fetch)"; then
    echo -e "${GREEN}[✓] 使用现代异步编程${NC}"
    echo "[✓] 使用现代异步编程" >> "$RESULTS_FILE"
else
    echo -e "${YELLOW}[⚠] 未使用现代异步API${NC}"
    echo "[⚠] 未使用现代异步API" >> "$RESULTS_FILE"
fi

# 检查模块化
if echo "$CAPTCHA_HTML" | grep -qE "(import|export|module)"; then
    echo -e "${GREEN}[✓] 使用ES模块化${NC}"
    echo "[✓] 使用ES模块化" >> "$RESULTS_FILE"
else
    echo -e "${YELLOW}[⚠] 未使用ES模块化${NC}"
    echo "[⚠] 未使用ES模块化" >> "$RESULTS_FILE"
fi

echo ""
echo "===========================================" | tee -a "$RESULTS_FILE"
echo "高级测试完成 - 结果已保存到 $RESULTS_FILE" | tee -a "$RESULTS_FILE"
echo "===========================================" | tee -a "$RESULTS_FILE"

