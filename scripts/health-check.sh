#!/bin/bash
set -e

echo "===== HJTPX 健康检查脚本 ====="

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

APP_URL="${APP_URL:-http://localhost:8080}"

check_service() {
    local name=$1
    local url=$2

    echo -n "检查 $name... "
    response=$(curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null)

    if [ "$response" = "200" ]; then
        echo "✓ 正常"
        return 0
    else
        echo "✗ 失败 (HTTP $response)"
        return 1
    fi
}

echo "===== 服务健康检查 ====="
echo ""

total=0
passed=0

total=$((total + 1))
if check_service "应用主服务" "$APP_URL/health"; then
    passed=$((passed + 1))
fi

total=$((total + 1))
if check_service "健康检查端点" "$APP_URL/healthz"; then
    passed=$((passed + 1))
fi

total=$((total + 1))
if check_service "就绪检查端点" "$APP_URL/readyz"; then
    passed=$((passed + 1))
fi

total=$((total + 1))
if check_service "API Ping" "$APP_URL/api/v1/ping"; then
    passed=$((passed + 1))
fi

echo ""
echo "===== 检查结果: $passed/$total 通过 ====="

if [ $passed -eq $total ]; then
    echo "所有检查通过 ✓"
    exit 0
else
    echo "部分检查失败 ✗"
    exit 1
fi
