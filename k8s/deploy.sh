#!/bin/bash

set -e

NAMESPACE="hjtpx"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "===== 开始部署 HJTPX 到 Kubernetes ====="
echo ""

echo "1. 创建命名空间..."
kubectl apply -f namespace.yaml

echo ""
echo "2. 部署配置映射..."
kubectl apply -f configmap.yaml

echo ""
echo "3. 部署密钥..."
read -p "是否要创建新的密钥? (y/n): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    kubectl create secret generic hjtpx-secrets \
        --from-literal=DATABASE_HOST=hjtpx-postgres \
        --from-literal=DATABASE_PORT=5432 \
        --from-literal=DATABASE_USER=postgres \
        --from-literal=DATABASE_PASSWORD="$(openssl rand -base64 32)" \
        --from-literal=DATABASE_NAME=hjtpx_db \
        --from-literal=REDIS_HOST=hjtpx-redis \
        --from-literal=REDIS_PORT=6379 \
        --from-literal=REDIS_PASSWORD="$(openssl rand -base64 32)" \
        --from-literal=JWT_SECRET="$(openssl rand -base64 32)" \
        --namespace=$NAMESPACE \
        --dry-run=client -o yaml | kubectl apply -f -
fi

echo ""
echo "4. 部署 PostgreSQL..."
kubectl apply -f postgres-deployment.yaml

echo ""
echo "5. 等待 PostgreSQL 就绪..."
kubectl wait --for=condition=ready pod -l app=hjtpx-postgres --timeout=300s -n $NAMESPACE

echo ""
echo "6. 部署 Redis..."
kubectl apply -f redis-deployment.yaml

echo ""
echo "7. 等待 Redis 就绪..."
kubectl wait --for=condition=ready pod -l app=hjtpx-redis --timeout=300s -n $NAMESPACE

echo ""
echo "8. 部署应用..."
kubectl apply -f deployment.yaml

echo ""
echo "9. 部署服务..."
kubectl apply -f service.yaml

echo ""
echo "10. 部署入口..."
kubectl apply -f ingress.yaml

echo ""
echo "11. 部署自动扩展..."
kubectl apply -f hpa.yaml

echo ""
echo "12. 部署 Pod 中断预算..."
kubectl apply -f pdb.yaml

echo ""
echo "13. 部署定时任务..."
kubectl apply -f cronjob.yaml

echo ""
echo "===== 部署完成 ====="
echo ""
echo "检查 Pod 状态..."
kubectl get pods -n $NAMESPACE
echo ""
echo "检查服务状态..."
kubectl get svc -n $NAMESPACE
echo ""
echo "访问应用..."
echo "  NodePort: http://<NODE_IP>:30080"
echo "  Ingress: https://hjtpx.example.com (需要配置 DNS)"
