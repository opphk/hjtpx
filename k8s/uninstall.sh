#!/bin/bash

set -e

NAMESPACE="hjtpx"

echo "===== 卸载 HJTPX ====="

read -p "确定要卸载 HJTPX 吗? (y/n): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "取消卸载"
    exit 0
fi

echo ""
echo "1. 删除定时任务..."
kubectl delete -f cronjob.yaml --ignore-not-found -n $NAMESPACE

echo ""
echo "2. 删除 Pod 中断预算..."
kubectl delete -f pdb.yaml --ignore-not-found -n $NAMESPACE

echo ""
echo "3. 删除自动扩展..."
kubectl delete -f hpa.yaml --ignore-not-found -n $NAMESPACE

echo ""
echo "4. 删除入口..."
kubectl delete -f ingress.yaml --ignore-not-found -n $NAMESPACE

echo ""
echo "5. 删除服务..."
kubectl delete -f service.yaml --ignore-not-found -n $NAMESPACE

echo ""
echo "6. 删除应用..."
kubectl delete -f deployment.yaml --ignore-not-found -n $NAMESPACE

echo ""
echo "7. 删除 Redis..."
kubectl delete -f redis-deployment.yaml --ignore-not-found -n $NAMESPACE

echo ""
echo "8. 删除 PostgreSQL..."
kubectl delete -f postgres-deployment.yaml --ignore-not-found -n $NAMESPACE

echo ""
echo "9. 删除配置..."
kubectl delete -f configmap.yaml --ignore-not-found -n $NAMESPACE

echo ""
echo "10. 删除密钥..."
kubectl delete secret hjtpx-secrets --ignore-not-found -n $NAMESPACE

echo ""
echo "===== 卸载完成 ====="
echo ""
echo "注意: PVC (持久卷) 未删除，如需删除请手动执行:"
echo "  kubectl delete pvc -n $NAMESPACE --all"
