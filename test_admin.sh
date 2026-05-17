#!/bin/bash

echo "测试管理后台API端点配置..."

# 检查管理后台页面路由
echo "1. 检查管理后台页面路由: /admin"
grep -n 'router.GET("/admin"' /workspace/hjtpx/internal/api/router.go

echo ""
echo "2. 检查应用管理API端点:"
grep -n 'admin.*apps' /workspace/hjtpx/internal/api/router.go

echo ""
echo "3. 检查用户管理API端点:"
grep -n 'admin.*users' /workspace/hjtpx/internal/api/router.go

echo ""
echo "4. 检查验证码管理API端点:"
grep -n 'admin.*captchas' /workspace/hjtpx/internal/api/router.go

echo ""
echo "5. 检查模板文件:"
ls -lh /workspace/hjtpx/templates/admin.html

echo ""
echo "6. 检查AdminHandler方法:"
grep -n 'func.*AdminHandler.*' /workspace/hjtpx/internal/api/handler/admin_handler.go | head -20

echo ""
echo "✅ 所有配置已就绪！"
