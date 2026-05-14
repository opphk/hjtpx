#!/bin/bash

set -e

echo "=== 行为验证系统部署脚本 ==="

PROJECT_DIR="/opt/captcha-system"
SERVICE_NAME="captcha-system"
PORT=8080

if [ "$EUID" -ne 0 ]; then
    echo "请使用 root 权限运行此脚本"
    exit 1
fi

echo "[1/5] 创建项目目录..."
mkdir -p $PROJECT_DIR

echo "[2/5] 编译项目..."
cd $PROJECT_DIR
if [ -f "go.mod" ]; then
    go build -o captcha-server ./cmd/server
    echo "编译完成"
else
    echo "错误: 未找到 go.mod 文件"
    exit 1
fi

echo "[3/5] 创建 systemd 服务..."
cat > /etc/systemd/system/${SERVICE_NAME}.service <<EOF
[Unit]
Description=Behavioral Captcha System
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=root
WorkingDirectory=$PROJECT_DIR
ExecStart=$PROJECT_DIR/captcha-server
Restart=always
RestartSec=5
Environment=PORT=$PORT

[Install]
WantedBy=multi-user.target
EOF

echo "[4/5] 启动服务..."
systemctl daemon-reload
systemctl enable $SERVICE_NAME
systemctl restart $SERVICE_NAME

echo "[5/5] 检查服务状态..."
sleep 2
if systemctl is-active --quiet $SERVICE_NAME; then
    echo "✅ 服务启动成功!"
    echo "访问地址: http://localhost:$PORT"
    echo "API文档: http://localhost:$PORT/api/v1/health"
else
    echo "❌ 服务启动失败，请检查日志:"
    systemctl status $SERVICE_NAME
fi

echo ""
echo "=== 常用命令 ==="
echo "查看日志: journalctl -u $SERVICE_NAME -f"
echo "重启服务: systemctl restart $SERVICE_NAME"
echo "停止服务: systemctl stop $SERVICE_NAME"
