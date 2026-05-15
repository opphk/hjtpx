#!/bin/bash
set -e

echo "优化构建流程"
echo "================================"

# 并行安装依赖
echo "1. 并行安装依赖..."
npm ci --prefer-offline &

# 等待所有后台任务完成
wait

# 并行运行lint和测试
echo "2. 并行运行lint和类型检查..."
npm run lint &
LINT_PID=$!

npm run typecheck &
TYPECHECK_PID=$!

wait $LINT_PID
wait $TYPECHECK_PID

# 并行运行测试
echo "3. 并行运行测试..."
npm run test:unit -- --coverage &
UNIT_TEST_PID=$!

npm run test:integration &
INTEGRATION_TEST_PID=$!

wait $UNIT_TEST_PID
wait $INTEGRATION_TEST_PID

# 构建
echo "4. 构建应用..."
npm run build

echo "================================"
echo "构建完成！"
