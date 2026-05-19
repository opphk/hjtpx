# Serverless 部署指南

## 概述

本文档介绍如何在 hjtpx 项目中部署和管理 Serverless 函数。Serverless 架构允许您专注于编写业务逻辑，而无需管理底层基础设施。

## 架构组件

### 核心组件

1. **Serverless Manager** (`serverless_manager.go`)
   - 函数注册和生命周期管理
   - 函数状态跟踪
   - 调用指标收集

2. **Function Deployer** (`function_deployer.go`)
   - 函数构建和打包
   - 部署流程自动化
   - 版本管理

3. **Trigger Manager** (`trigger_config.go`)
   - HTTP 触发器
   - 定时触发器
   - 队列触发器
   - 事件触发器

4. **Cold Start Optimizer** (`cold_start_optimizer.go`)
   - 预热策略
   - 延迟加载
   - 连接池管理
   - 运行时优化

5. **Auto Scaler** (`auto_scaler.go`)
   - 目标跟踪扩展
   - 步进扩展
   - 计划扩展
   - 预测扩展

6. **Cost Optimizer** (`cost_optimizer.go`)
   - 成本分析
   - 预算告警
   - 优化建议
   - 预留容量管理

7. **Serverless Runtime** (`serverless_runtime.go`)
   - 函数执行环境
   - 请求处理
   - 指标收集

## 快速开始

### 1. 环境准备

```bash
# 确保已安装 Go 1.20+
go version

# 确保已安装 Docker (可选)
docker --version
```

### 2. 部署函数

```bash
# 使用部署脚本
./scripts/deploy-serverless.sh -n my-function -r go1.20 -m 256

# 或使用配置脚本
./scripts/deploy-serverless.sh \
  --name my-function \
  --runtime go1.20 \
  --memory 256 \
  --timeout 30
```

### 3. 配置自动扩展

```bash
./scripts/configure-scaling.sh \
  --name my-function \
  --min 1 \
  --max 10 \
  --target 70 \
  --metric cpu_utilization
```

### 4. 优化成本

```bash
./scripts/optimize-costs.sh \
  --name my-function \
  --budget 100.0 \
  --threshold 80
```

## Kubernetes 部署

### 1. 部署命名空间和 RBAC

```bash
kubectl apply -f k8s/serverless-namespace.yaml
```

### 2. 部署 Serverless Runtime

```bash
kubectl apply -f k8s/serverless-deployment.yaml
```

### 3. 部署 Knative 服务 (可选)

```bash
kubectl apply -f k8s/knative-service.yaml
```

### 4. 部署定时任务

```bash
kubectl apply -f k8s/serverless-cronjobs.yaml
```

## 配置选项

### 函数配置

```yaml
function:
  name: my-function
  runtime: go1.20
  memory: 256
  timeout: 30
  handler: main.Handle
  max_instances: 10
  min_instances: 1
  concurrency: 1
```

### 触发器配置

#### HTTP 触发器

```json
{
  "trigger_type": "http",
  "path": "/api/my-function",
  "method": ["GET", "POST"],
  "auth_type": "none",
  "rate_limit": {
    "requests_per_second": 100,
    "burst_size": 200
  }
}
```

#### 定时触发器

```json
{
  "trigger_type": "timer",
  "expression": "0 */5 * * * *",
  "cron_enabled": true,
  "timezone": "UTC"
}
```

#### 队列触发器

```json
{
  "trigger_type": "queue",
  "queue_name": "my-queue",
  "batch_size": 10,
  "max_retries": 3
}
```

### 扩展配置

#### 目标跟踪扩展

```json
{
  "policy_type": "target_tracking",
  "metric": "cpu_utilization",
  "target_value": 70,
  "min_adjustment": 1,
  "max_adjustment": 10,
  "cooldown": 60
}
```

#### 步进扩展

```json
{
  "policy_type": "step_scaling",
  "metric": "request_count",
  "step_adjustments": [
    {"lower_bound": 0, "upper_bound": 100, "adjustment": 1},
    {"lower_bound": 100, "upper_bound": 200, "adjustment": 2},
    {"lower_bound": 200, "adjustment": 3}
  ],
  "cooldown": 60
}
```

## 冷启动优化

### 1. 启用预热

```go
optimizer := service.NewColdStartOptimizer(manager)
optimizer.Configure("my-function", &service.OptimizationConfig{
    Strategy:           service.StrategyPreWarming,
    PreWarmingEnabled:   true,
    PreWarmingInterval:  5 * time.Minute,
    PreWarmingCount:     2,
})
```

### 2. 启用延迟加载

```go
optimizer.Configure("my-function", &service.OptimizationConfig{
    Strategy:          service.StrategyLazyLoading,
    CacheEnabled:       true,
})
```

### 3. 连接池

```go
optimizer.Configure("my-function", &service.OptimizationConfig{
    Strategy:    service.StrategyConnectionPooling,
    PoolSize:    10,
})
```

## 成本优化

### 1. 内存优化

```go
optimizer := service.NewCostOptimizer(manager)
optimalMemory, err := optimizer.OptimizeMemory("my-function")
```

### 2. 超时优化

```go
optimalTimeout, err := optimizer.OptimizeTimeout("my-function")
```

### 3. 预算告警

```go
optimizer.SetBudgetAlert("my-function", 100.0)
alerts := optimizer.CheckBudgetAlerts()
```

### 4. 预留容量

```go
capacity, err := optimizer.PurchaseReservedCapacity(
    "my-function",
    "t3.medium",
    2,
    8760,
    0.05,
)
```

## 监控和日志

### 查看扩展指标

```bash
kubectl get hpa -n serverless
```

### 查看 Pod 日志

```bash
kubectl logs -n serverless -l app=serverless-runtime
```

### 查看函数日志

```bash
kubectl logs -n serverless -l function=my-function
```

## 故障排查

### 函数无法部署

1. 检查构建日志
2. 验证函数配置
3. 检查资源限制

### 冷启动时间过长

1. 启用预热
2. 减少依赖项
3. 使用 ARM64 架构

### 扩展不工作

1. 检查 HPA 配置
2. 验证指标可用性
3. 检查资源配额

## 最佳实践

1. **最小化函数大小**
   - 只包含必要的依赖
   - 使用树摇优化

2. **优化冷启动**
   - 启用预热
   - 使用延迟加载
   - 最小化初始化代码

3. **成本优化**
   - 选择合适的内存大小
   - 使用预留容量
   - 配置预算告警

4. **安全**
   - 使用最小权限原则
   - 加密敏感数据
   - 定期更新依赖

## 参考资料

- [Go Serverless 文档](https://golang.org/doc/)
- [Kubernetes 文档](https://kubernetes.io/docs/)
- [Knative 文档](https://knative.dev/docs/)
