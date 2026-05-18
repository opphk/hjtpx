#!/bin/bash
set -e

PROD_USER="hjtpx"
PROD_DIR="/opt/hjtpx"
APP_PORT=8080
NGINX_PORT=80
NGINX_SSL_PORT=443

echo "===== HJTPX 生产环境部署脚本 ====="

if [ "$EUID" -ne 0 ]; then
    echo "错误: 此脚本需要 root 权限运行"
    echo "请使用: sudo $0"
    exit 1
fi

echo "1. 创建应用用户..."
if ! id "$PROD_USER" &>/dev/null; then
    useradd -r -s /bin/false -d "$PROD_DIR" -m "$PROD_USER"
    echo "用户 $PROD_USER 创建成功"
else
    echo "用户 $PROD_USER 已存在"
fi

echo "2. 创建部署目录..."
mkdir -p "$PROD_DIR"/{data,logs,config,scripts}
mkdir -p /var/log/hjtpx

echo "3. 编译 Go 二进制文件..."
cd "$(dirname "$0")/.."
CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o hjtpx-server ./backend/cmd/server
echo "编译完成"

echo "4. 复制文件到部署目录..."
cp -f hjtpx-server "$PROD_DIR/"
cp -f config.yaml "$PROD_DIR/config/"
cp -f scripts/hjtpx.service /etc/systemd/system/
chmod +x "$PROD_DIR/hjtpx-server"

echo "5. 设置权限..."
chown -R "$PROD_USER:$PROD_USER" "$PROD_DIR"
chown -R "$PROD_USER:$PROD_USER" /var/log/hjtpx

echo "6. 重新加载 systemd..."
systemctl daemon-reload

echo "7. 启用服务..."
systemctl enable hjtpx
systemctl enable nginx

echo "8. 检查配置语法..."
nginx -t

echo "9. 启动服务..."
systemctl restart hjtpx
systemctl restart nginx

sleep 3

echo "10. 检查服务状态..."
systemctl status hjtpx --no-pager || true
echo ""

echo "11. 验证健康检查..."
for i in {1..10}; do
    if curl -sf "http://localhost:$APP_PORT/health" > /dev/null; then
        echo "✓ 健康检查通过"
        break
    fi
    if [ $i -eq 10 ]; then
        echo "✗ 健康检查失败"
        journalctl -u hjtpx -n 20 --no-pager
        exit 1
    fi
    sleep 1
done

echo ""
echo "===== 部署完成 ====="
echo ""
echo "服务地址:"
echo "  - 应用API: http://localhost:$APP_PORT"
echo "  - Nginx前端: http://localhost:$NGINX_PORT"
echo "  - 健康检查: http://localhost:$APP_PORT/health"
echo ""
echo "常用命令:"
echo "  systemctl status hjtpx    - 查看服务状态"
echo "  systemctl restart hjtpx    - 重启服务"
echo "  journalctl -u hjtpx -f     - 查看日志"
echo "  systemctl stop hjtpx       - 停止服务"
