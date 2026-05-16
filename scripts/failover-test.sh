#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "===== 极验行为验证系统 - 故障转移测试 ====="

check_cluster_health() {
    echo "检查集群健康状态..."

    local NODES=(8081 8082 8083)
    for PORT in "${NODES[@]}"; do
        if curl -sf "http://localhost:$PORT/health/ha" > /dev/null 2>&1; then
            echo "✓ 节点端口 $PORT 可用"
        else
            echo "✗ 节点端口 $PORT 不可用"
        fi
    done
}

simulate_node_failure() {
    local NODE_NUM=$1
    local CONTAINER_NAME="hjtpx-app-node${NODE_NUM}"

    echo ""
    echo "模拟节点 $NODE_NUM 故障..."

    docker pause "$CONTAINER_NAME" 2>/dev/null || {
        echo "警告: 无法暂停容器，尝试停止..."
        docker stop "$CONTAINER_NAME"
    }

    echo "✓ 节点 $NODE_NUM 已停止/暂停"
}

test_failover() {
    local FROM_NODE=$1
    local TO_NODE=$2

    echo ""
    echo "测试故障转移: 节点 $FROM_NODE -> 节点 $TO_NODE"

    echo "触发故障转移..."
    curl -X POST "http://localhost:8081/failover/manual" \
        -H "Content-Type: application/json" \
        -d "{\"from_node\":\"node-${FROM_NODE}\",\"to_node\":\"node-${TO_NODE}\"}" \
        2>/dev/null || echo "故障转移请求失败或端点不可用"

    echo "等待故障转移完成..."
    sleep 10

    echo "检查故障转移状态..."
    curl -s "http://localhost:8081/failover/status" | jq '.' 2>/dev/null || \
        echo "无法获取故障转移状态"
}

test_health_check() {
    echo ""
    echo "测试健康检查机制..."

    local NODES=(8081 8082 8083)
    for PORT in "${NODES[@]}"; do
        RESPONSE=$(curl -s "http://localhost:$PORT/health" 2>/dev/null)
        if [ -n "$RESPONSE" ]; then
            echo "节点 $PORT 健康状态:"
            echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
        fi
    done
}

test_load_balancing() {
    echo ""
    echo "测试负载均衡..."

    for i in {1..10}; do
        RESPONSE=$(curl -s "http://localhost/health" 2>/dev/null)
        BACKEND=$(echo "$RESPONSE" | jq -r '.backend // empty' 2>/dev/null)
        echo "请求 $i: $BACKEND"
    done
}

test_data_consistency() {
    echo ""
    echo "测试数据一致性..."

    local TEST_KEY="test_failover_$(date +%s)"
    local TEST_VALUE="test_value_$(date +%s)"

    echo "写入测试数据到节点 8081..."
    curl -X POST "http://localhost:8081/api/v1/test/set" \
        -H "Content-Type: application/json" \
        -d "{\"key\":\"$TEST_KEY\",\"value\":\"$TEST_VALUE\"}" \
        2>/dev/null || echo "写入请求失败"

    sleep 2

    echo "从节点 8082 读取测试数据..."
    curl -s "http://localhost:8082/api/v1/test/get?key=$TEST_KEY" 2>/dev/null || \
        echo "读取请求失败"

    echo "从节点 8083 读取测试数据..."
    curl -s "http://localhost:8083/api/v1/test/get?key=$TEST_KEY" 2>/dev/null || \
        echo "读取请求失败"
}

recover_node() {
    local NODE_NUM=$1
    local CONTAINER_NAME="hjtpx-app-node${NODE_NUM}"

    echo ""
    echo "恢复节点 $NODE_NUM..."

    docker unpause "$CONTAINER_NAME" 2>/dev/null || docker start "$CONTAINER_NAME"

    echo "等待节点恢复..."
    sleep 10

    if curl -sf "http://localhost:$((8080 + NODE_NUM))/health" > /dev/null 2>&1; then
        echo "✓ 节点 $NODE_NUM 恢复成功"
    else
        echo "✗ 节点 $NODE_NUM 恢复失败"
    fi
}

test_auto_recovery() {
    echo ""
    echo "测试自动恢复机制..."

    local NODE_NUM=3
    local CONTAINER_NAME="hjtpx-app-node${NODE_NUM}"

    echo "停止节点 $NODE_NUM..."
    docker stop "$CONTAINER_NAME"

    echo "等待自动故障转移..."
    sleep 15

    echo "检查集群状态..."
    curl -s "http://localhost:8081/health/cluster" | jq '.' 2>/dev/null || \
        echo "无法获取集群状态"

    echo "恢复节点 $NODE_NUM..."
    docker start "$CONTAINER_NAME"

    echo "等待节点重新加入集群..."
    sleep 15

    echo "检查节点状态..."
    curl -s "http://localhost:$((8080 + NODE_NUM))/health" | jq '.' 2>/dev/null || \
        echo "节点状态检查失败"
}

show_failover_events() {
    echo ""
    echo "===== 故障转移事件日志 ====="

    curl -s "http://localhost:8081/failover/events" 2>/dev/null | jq '.' || \
        echo "无法获取事件日志"
}

show_metrics() {
    echo ""
    echo "===== 故障转移指标 ====="

    curl -s "http://localhost:8081/failover/metrics" 2>/dev/null | jq '.' || \
        echo "无法获取指标数据"
}

main() {
    cd "$PROJECT_ROOT"

    case "${1:-all}" in
        health)
            check_cluster_health
            test_health_check
            ;;
        failover)
            simulate_node_failure 2
            test_failover 2 1
            test_health_check
            ;;
        recover)
            recover_node 2
            sleep 10
            test_health_check
            ;;
        auto)
            test_auto_recovery
            ;;
        consistency)
            test_data_consistency
            ;;
        events)
            show_failover_events
            ;;
        metrics)
            show_metrics
            ;;
        all)
            check_cluster_health
            test_health_check
            test_load_balancing
            echo ""
            echo "===== 开始故障转移测试 ====="
            simulate_node_failure 2
            test_failover 2 1
            sleep 10
            test_health_check
            show_failover_events
            show_metrics
            echo ""
            echo "===== 恢复测试 ====="
            recover_node 2
            sleep 10
            test_health_check
            echo ""
            echo "===== 故障转移测试完成 ====="
            ;;
        *)
            echo "用法: $0 {health|failover|recover|auto|consistency|events|metrics|all}"
            echo ""
            echo "选项:"
            echo "  health       - 检查集群健康状态"
            echo "  failover     - 模拟故障并测试故障转移"
            echo "  recover      - 恢复故障节点"
            echo "  auto         - 测试自动恢复机制"
            echo "  consistency  - 测试数据一致性"
            echo "  events       - 显示故障转移事件日志"
            echo "  metrics      - 显示故障转移指标"
            echo "  all          - 运行所有测试 (默认)"
            ;;
    esac
}

main "$@"
