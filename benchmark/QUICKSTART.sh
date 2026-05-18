#!/usr/bin/env bash

# 性能压测自动化 - 快速参考指南
# ============================

echo "性能压测自动化工具 - 快速参考"
echo "============================="
echo ""

# 检查环境
echo "1. 环境检查"
echo "------------"
go version 2>/dev/null && echo "✓ Go 已安装" || echo "✗ Go 未安装"
curl -sf http://localhost:8080/health >/dev/null 2>&1 && echo "✓ 服务运行中" || echo "○ 服务未运行 (需要先启动服务)"
echo ""

# 基本用法
echo "2. 基本用法"
echo "------------"
echo "  # 启动服务并运行压测"
echo "  make benchmark-serve"
echo "  # 或"
echo "  ./scripts/run_benchmark.sh serve"
echo ""
echo "  # 快速压测 (30秒)"
echo "  make benchmark-quick"
echo ""
echo "  # 标准压测 (60秒)"
echo "  make benchmark"
echo ""
echo "  # 完整压测 (120秒)"
echo "  make benchmark-full"
echo ""

# 基准管理
echo "3. 基准管理"
echo "------------"
echo "  # 生成新的性能基准"
echo "  make benchmark-baseline"
echo ""
echo "  # 与基准对比检测回归"
echo "  make benchmark-compare"
echo ""

# CI/CD
echo "4. CI/CD 集成"
echo "--------------"
echo "  # GitHub Actions 自动运行"
echo "  # - 定时任务: 每天凌晨2点"
echo "  # - 推送触发: main/develop分支"
echo "  # - 手动触发: workflow_dispatch"
echo ""
echo "  # 本地模拟CI模式"
echo "  make benchmark-ci"
echo ""

# 报告查看
echo "5. 报告位置"
echo "------------"
echo "  JSON报告: benchmark/reports/"
echo "  HTML报告: benchmark/reports/"
echo "  基准数据: benchmark/baselines/"
echo ""

# CLI选项
echo "6. CLI 选项"
echo "------------"
echo "  --url string        API地址 (默认: http://localhost:8080)"
echo "  --duration int      时长秒数 (默认: 60)"
echo "  --concurrency int   并发数 (默认: 100)"
echo "  --scenario string   场景过滤 (如: slider)"
echo "  --format string     报告格式: json|html|both (默认: both)"
echo "  --output string     输出目录 (默认: ./reports)"
echo "  --baseline          保存为基准"
echo "  --compare           与基准对比"
echo "  --quiet             静默模式"
echo "  --no-color          禁用颜色"
echo ""

# 示例命令
echo "7. 示例命令"
echo "------------"
echo "  # 只测试slider场景"
echo "  cd benchmark && go run cmd/benchmark/main.go --scenario=slider"
echo ""
echo "  # 100并发60秒压测"
echo "  cd benchmark && go run cmd/benchmark/main.go --concurrency=100 --duration=60"
echo ""
echo "  # 对比基准并生成报告"
echo "  cd benchmark && go run cmd/benchmark/main.go --compare --format=both"
echo ""
echo "  # 自定义API地址"
echo "  cd benchmark && go run cmd/benchmark/main.go --url=http://api.example.com:8080"
echo ""

# 性能目标
echo "8. 性能目标"
echo "------------"
echo "  ✓ PASS:  QPS > 5000, P99 < 100ms, 错误率 < 1%"
echo "  ⚠ WARN:  QPS > 1000, P99 < 200ms, 错误率 < 5%"
echo "  ✗ FAIL:  QPS < 1000 或 P99 >= 200ms 或 错误率 >= 5%"
echo ""

# 回归检测
echo "9. 回归检测阈值"
echo "---------------"
echo "  QPS下降:      > 20%"
echo "  P99延迟增加:  > 50% 且超过500ms"
echo "  错误率增加:  > 2倍且超过5%"
echo "  内存增长:    > 20% 且超过50MB"
echo ""

echo "详细文档: PERFORMANCE_BENCHMARK_IMPLEMENTATION.md"
echo ""
