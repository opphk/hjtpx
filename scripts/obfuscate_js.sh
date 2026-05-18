#!/bin/bash

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_DIR="$PROJECT_ROOT/backend"
FRONTEND_DIR="$PROJECT_ROOT/frontend"

echo "========================================="
echo "JavaScript代码混淆构建脚本"
echo "========================================="

if [ ! -f "$BACKEND_DIR/cmd/hjtpx" ]; then
    echo "构建后端程序..."
    cd "$BACKEND_DIR"
    go build -o cmd/hjtpx ./cmd/hjtpx
    cd "$PROJECT_ROOT"
fi

OBFUSCATOR="$BACKEND_DIR/cmd/hjtpx"

echo ""
echo "混淆前端JavaScript文件..."
echo ""

FRONTEND_JS_DIR="$FRONTEND_DIR/static/js"
BACKEND_JS_DIR="$BACKEND_DIR/static/js"

for js_file in "$FRONTEND_JS_DIR"/*.js "$BACKEND_JS_DIR"/*.js; do
    if [ -f "$js_file" ]; then
        filename=$(basename "$js_file")
        echo "混淆: $filename"

        TEMP_FILE=$(mktemp)
        "$OBFUSCATOR" --obfuscate "$js_file" --output "$TEMP_FILE" --level 3 --enable-all-protections || true

        if [ -f "$TEMP_FILE" ] && [ -s "$TEMP_FILE" ]; then
            cp "$TEMP_FILE" "$js_file"
            echo "  ✓ 完成混淆"
        else
            echo "  ✗ 混淆失败，保留原文件"
        fi

        rm -f "$TEMP_FILE"
    fi
done

echo ""
echo "========================================="
echo "混淆完成！"
echo "========================================="
