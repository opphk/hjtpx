#!/bin/sh
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

SSL_DIR="${SSL_DIR:-./nginx/ssl}"
CERT_EMAIL="${CERT_EMAIL:-admin@example.com}"
DOMAINS="${DOMAINS:-}"
CERTBOT_IMAGE="certbot/certbot:latest"
NGINX_CONTAINER="${NGINX_CONTAINER:-hjtpx-nginx}"

echo "===== SSL 证书自动续期脚本 ====="
echo "SSL目录: $SSL_DIR"
echo ""

check_certbot() {
    if command -v certbot > /dev/null 2>&1; then
        return 0
    fi
    return 1
}

check_docker() {
    if command -v docker > /dev/null 2>&1; then
        return 0
    fi
    return 1
}

generate_self_signed() {
    echo "生成自签名SSL证书..."
    mkdir -p "$SSL_DIR"

    openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
        -keyout "$SSL_DIR/key.pem" \
        -out "$SSL_DIR/cert.pem" \
        -subj "/C=CN/ST=Beijing/L=Beijing/O=HJTPX/CN=localhost" \
        2>/dev/null

    if [ -f "$SSL_DIR/cert.pem" ] && [ -f "$SSL_DIR/key.pem" ]; then
        echo "✓ 自签名证书生成成功"
        return 0
    else
        echo "✗ 自签名证书生成失败"
        return 1
    fi
}

renew_with_certbot_docker() {
    echo "使用Docker运行Certbot续期证书..."

    docker run --rm \
        -v "$SSL_DIR:/etc/letsencrypt/live" \
        -v "$SSL_DIR:/var/lib/letsencrypt" \
        -v "$SSL_DIR:/var/log/letsencrypt" \
        "$CERTBOT_IMAGE" renew \
        --webroot -w /var/www/html \
        --quiet \
        --deploy-hook "docker exec $NGINX_CONTAINER nginx -s reload" \
        2>/dev/null || true

    echo "✓ 证书续期完成"
}

renew_with_certbot_host() {
    echo "使用主机Certbot续期证书..."

    certbot renew \
        --webroot -w /var/www/html \
        --quiet \
        --deploy-hook "docker exec $NGINX_CONTAINER nginx -s reload" \
        2>/dev/null || true

    echo "✓ 证书续期完成"
}

setup_cron() {
    echo "设置定时续期任务..."
    CRON_JOB="0 3 * * * $SCRIPT_DIR/ssl-renew.sh >> /var/log/ssl-renew.log 2>&1"

    if command -v crontab > /dev/null 2>&1; then
        crontab -l 2>/dev/null | grep -v "ssl-renew.sh" | { cat; echo "$CRON_JOB"; } | crontab -
        echo "✓ 定时任务已添加"
    else
        echo "⚠ crontab未安装,跳过定时任务设置"
    fi
}

main() {
    echo "开始SSL证书续期检查..."
    echo ""

    if [ ! -d "$SSL_DIR" ]; then
        mkdir -p "$SSL_DIR"
    fi

    if [ ! -f "$SSL_DIR/cert.pem" ] || [ ! -f "$SSL_DIR/key.pem" ]; then
        echo "检测到SSL证书不存在,正在生成..."
        generate_self_signed
        exit 0
    fi

    CERT_EXPIRY=$(openssl x509 -in "$SSL_DIR/cert.pem" -noout -dates 2>/dev/null | grep notAfter | cut -d= -f2)
    CERT_DAYS=$(openssl x509 -in "$SSL_DIR/cert.pem" -noout -days 2>/dev/null | cut -d= -f2)

    echo "证书到期时间: $CERT_EXPIRY"
    echo "剩余天数: $CERT_DAYS 天"

    if [ -n "$CERT_DAYS" ] && [ "$CERT_DAYS" -gt 30 ]; then
        echo "证书有效期充足,无需续期"
        exit 0
    fi

    if [ -n "$DOMAINS" ]; then
        echo "检测到自定义域名: $DOMAINS"
        if check_certbot; then
            renew_with_certbot_host
        elif check_docker; then
            renew_with_certbot_docker
        else
            echo "⚠ 未找到Certbot或Docker,跳过续期"
        fi
    else
        echo "使用自签名证书,跳过Let's Encrypt续期"
    fi

    setup_cron

    echo ""
    echo "===== SSL 证书续期完成 ====="
}

main "$@"
