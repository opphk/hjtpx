#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

VERSION="${VERSION:-latest}"
CLUSTER_MODE="${CLUSTER_MODE:-false}"
NODE_COUNT="${NODE_COUNT:-3}"

echo "===== 极验行为验证系统 - 集群部署脚本 ====="
echo "版本: $VERSION"
echo "集群模式: $CLUSTER_MODE"
echo "节点数量: $NODE_COUNT"
echo ""

check_docker() {
    echo "检查 Docker 环境..."
    if ! command -v docker &> /dev/null; then
        echo "错误: Docker 未安装"
        exit 1
    fi

    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        echo "错误: Docker Compose 未安装"
        exit 1
    fi

    echo "✓ Docker 环境检查通过"
}

check_ports() {
    echo "检查端口占用..."
    PORTS=(80 443 5432 6379 8081 8082 8083 9090 3000 3100)

    for PORT in "${PORTS[@]}"; do
        if lsof -Pi :$PORT -sTCP:LISTEN -t >/dev/null 2>&1; then
            echo "警告: 端口 $PORT 已被占用"
        fi
    done
    echo "✓ 端口检查完成"
}

pull_images() {
    echo "拉取 Docker 镜像..."
    docker pull postgres:16-alpine
    docker pull redis:7-alpine
    docker pull nginx:1.25-alpine
    echo "✓ 镜像拉取完成"
}

create_networks() {
    echo "创建 Docker 网络..."
    docker network create hjtpx-cluster-network 2>/dev/null || echo "网络已存在"
    echo "✓ 网络创建完成"
}

start_postgres() {
    echo "启动 PostgreSQL..."
    docker run -d \
        --name hjtpx-postgres \
        --network hjtpx-cluster-network \
        -e POSTGRES_USER=postgres \
        -e POSTGRES_PASSWORD=postgres \
        -e POSTGRES_DB=hjtpx_db \
        -p 5432:5432 \
        -v hjtpx-postgres-data:/var/lib/postgresql/data \
        postgres:16-alpine
    echo "等待 PostgreSQL 启动..."
    sleep 10
    echo "✓ PostgreSQL 启动完成"
}

start_redis() {
    echo "启动 Redis..."
    docker run -d \
        --name hjtpx-redis \
        --network hjtpx-cluster-network \
        -e REDIS_PASSWORD=redis123 \
        -p 6379:6379 \
        -v hjtpx-redis-data:/data \
        redis:7-alpine redis-server --requirepass redis123 --appendonly yes
    echo "✓ Redis 启动完成"
}

start_app_node() {
    local NODE_NUM=$1
    local NODE_ROLE=$2

    echo "启动应用节点 $NODE_NUM (角色: $NODE_ROLE)..."

    local PORT=$((8080 + NODE_NUM))
    local CONTAINER_NAME="hjtpx-app-node${NODE_NUM}"

    docker run -d \
        --name "$CONTAINER_NAME" \
        --network hjtpx-cluster-network \
        -e NODE_ID="node-${NODE_NUM}" \
        -e NODE_ROLE="$NODE_ROLE" \
        -e CLUSTER_ENABLED="$CLUSTER_MODE" \
        -e POSTGRES_HOST=hjtpx-postgres \
        -e POSTGRES_PORT=5432 \
        -e POSTGRES_USER=postgres \
        -e POSTGRES_PASSWORD=postgres \
        -e POSTGRES_DB=hjtpx_db \
        -e REDIS_HOST=hjtpx-redis \
        -e REDIS_PORT=6379 \
        -e REDIS_PASSWORD=redis123 \
        -e JWT_SECRET=cluster-secret-key-change-in-production \
        -e HA_ENABLED=true \
        -e FAILOVER_ENABLED=true \
        -p "$PORT:8080" \
        -v hjtpx-app-logs-node${NODE_NUM}:/var/log/hjtpx \
        --health-cmd="wget --no-verbose --tries=1 --spider http://localhost:8080/health" \
        --health-interval=15s \
        --health-timeout=10s \
        --health-retries=5 \
        hjtpx/app:${VERSION}

    echo "✓ 节点 $NODE_NUM 启动完成 (端口: $PORT)"
}

start_nginx() {
    echo "启动 Nginx 负载均衡器..."
    docker run -d \
        --name hjtpx-nginx-cluster \
        --network hjtpx-cluster-network \
        -p 80:80 \
        -p 443:443 \
        -v "$PROJECT_ROOT/nginx/nginx-cluster.conf:/etc/nginx/nginx.conf:ro" \
        -v hjtpx-nginx-logs:/var/log/nginx \
        --depends-on hjtpx-app-node1:hjtpx-app-node1 \
        --depends-on hjtpx-app-node2:hjtpx-app-node2 \
        --depends-on hjtpx-app-node3:hjtpx-app-node3 \
        nginx:1.25-alpine
    echo "✓ Nginx 启动完成"
}

start_monitoring() {
    echo "启动监控组件..."

    docker run -d \
        --name hjtpx-prometheus \
        --network hjtpx-cluster-network \
        -p 9090:9090 \
        -v "$PROJECT_ROOT/monitoring/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro" \
        -v hjtpx-prometheus-data:/prometheus \
        prom/prometheus:v2.48.0

    docker run -d \
        --name hjtpx-grafana \
        --network hjtpx-cluster-network \
        -p 3000:3000 \
        -e GF_SECURITY_ADMIN_USER=admin \
        -e GF_SECURITY_ADMIN_PASSWORD=admin123 \
        -v hjtpx-grafana-data:/var/lib/grafana \
        grafana/grafana:10.2.2

    docker run -d \
        --name hjtpx-loki \
        --network hjtpx-cluster-network \
        -p 3100:3100 \
        -v "$PROJECT_ROOT/monitoring/loki/loki.yml:/etc/loki/loki.yml:ro" \
        -v hjtpx-loki-data:/loki \
        grafana/loki:2.9.3

    echo "✓ 监控组件启动完成"
}

wait_for_health() {
    echo "等待所有节点健康检查..."

    local MAX_WAIT=120
    local ELAPSED=0

    while [ $ELAPSED -lt $MAX_WAIT ]; do
        local ALL_HEALTHY=true

        for i in $(seq 1 $NODE_COUNT); do
            local PORT=$((8080 + i))
            if ! curl -sf "http://localhost:$PORT/health" > /dev/null 2>&1; then
                ALL_HEALTHY=false
                break
            fi
        done

        if [ "$ALL_HEALTHY" = true ]; then
            echo "✓ 所有节点健康检查通过"
            return 0
        fi

        sleep 5
        ELAPSED=$((ELAPSED + 5))
        echo "等待中... ($ELAPSED/$MAX_WAIT 秒)"
    done

    echo "警告: 健康检查超时，部分节点可能未就绪"
}

check_cluster_status() {
    echo ""
    echo "===== 集群状态 ====="

    for i in $(seq 1 $NODE_COUNT); do
        local PORT=$((8080 + i))
        echo ""
        echo "节点 $i (端口: $PORT):"
        curl -s "http://localhost:$PORT/health/ha" | jq '.' 2>/dev/null || echo "无法获取状态"
    done
}

cleanup() {
    echo ""
    echo "清理旧容器..."
    docker ps -a --filter "name=hjtpx-" --format "{{.Names}}" | xargs -r docker stop 2>/dev/null || true
    docker ps -a --filter "name=hjtpx-" --format "{{.Names}}" | xargs -r docker rm 2>/dev/null || true
}

main() {
    cd "$PROJECT_ROOT"

    if [ "$1" = "cleanup" ]; then
        cleanup
        echo "✓ 清理完成"
        exit 0
    fi

    check_docker
    check_ports

    if [ "$CLUSTER_MODE" = "true" ]; then
        echo ""
        echo "===== 启动集群模式 ====="

        pull_images
        create_networks
        start_postgres
        sleep 5
        start_redis

        start_app_node 1 "primary"

        for i in $(seq 2 $NODE_COUNT); do
            start_app_node $i "secondary"
        done

        start_nginx
        start_monitoring
        wait_for_health
        check_cluster_status
    else
        echo ""
        echo "===== 启动单节点模式 ====="

        cd "$PROJECT_ROOT"
        docker-compose up -d
    fi

    echo ""
    echo "===== 部署完成 ====="
    echo ""
    echo "访问地址:"
    echo "  - 应用: http://localhost"
    echo "  - Prometheus: http://localhost:9090"
    echo "  - Grafana: http://localhost:3000 (admin/admin123)"
    echo ""
}

main "$@"
