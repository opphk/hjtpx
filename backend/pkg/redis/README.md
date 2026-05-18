# 缓存策略优化说明

## 概述

本优化实现了多级缓存架构、Redis连接池优化、缓存预热策略、缓存一致性机制和监控功能。

## 主要优化内容

### 1. Redis缓存配置优化

**文件:** `backend/pkg/config/cache_config.go`

**优化点:**
- 连接池大小从100提升到200
- 最小空闲连接从10提升到20
- 新增自适应调优功能
- 新增健康检查机制
- 空闲超时时间优化为5分钟
- 连接最大生命周期优化为30分钟

**配置示例:**
```yaml
redis:
  pool_size: 200
  min_idle_conns: 20
  max_idle_conns: 100
  dial_timeout: 5s
  read_timeout: 3s
  write_timeout: 3s
  pool_timeout: 4s
  idle_timeout: 300s
  enable_auto_tuning: true
  health_check_enabled: true
  health_check_period: 30s
```

### 2. 多级缓存架构

**文件:** `backend/pkg/redis/multi_level_cache.go`

**架构设计:**
- **L1缓存**: 本地内存缓存，使用LRU策略
  - 最大10000条目
  - 最大内存100MB
  - TTL: 5分钟
  - 自动驱逐策略
  
- **L2缓存**: Redis分布式缓存
  - 默认TTL: 30分钟
  - 最大TTL: 2小时
  - 支持压缩
  - 支持版本控制

**工作流程:**
1. 先查询L1缓存
2. L1未命中则查询L2缓存
3. L2命中则回填L1
4. 完全未命中返回错误

**使用示例:**
```go
mlc := GetMultiLevelCache()

// 设置缓存
mlc.Set(ctx, "key", []byte("value"), &CacheOptions{
    TTL:   30 * time.Minute,
    Level: CacheLevelBoth,
})

// 获取缓存
val, err := mlc.Get(ctx, "key", nil)

// 删除缓存
mlc.Delete(ctx, "key", nil)
```

### 3. 缓存预热策略

**文件:** `backend/pkg/redis/enhanced_warmup.go`

**特性:**
- 支持优先级预热（Critical > High > Normal > Low）
- 自适应预热策略
- 智能预测加载
- 峰值时段预热
- 并发控制
- 批量处理

**优先级策略:**
- Critical: 关键业务数据，如认证配置
- High: 高频访问数据，如验证码类型配置
- Normal: 普通业务数据
- Low: 低优先级数据

**使用示例:**
```go
manager := GetEnhancedWarmupManager()

task := &EnhancedWarmupTask{
    Name:     "captcha_config",
    Key:      "config:captcha:types",
    Priority: WarmupPriorityHigh,
    Policy:   WarmupPolicyEager,
    TTL:      1 * time.Hour,
    Loader: func(ctx context.Context) ([]byte, error) {
        return fetchCaptchaConfig()
    },
}

manager.AddTask(task)
manager.Start()
```

### 4. 缓存一致性机制

**文件:** `backend/pkg/redis/enhanced_consistency.go`

**一致性级别:**
- Eventual（最终一致）
- Strong（强一致）
- Linearizable（线性化）

**更新模式:**
- Write-Through（写穿透）
- Write-Behind（写回）
- Refresh-Ahead（预刷新）

**失效策略:**
- 基于版本号的失效
- 基于Pub/Sub的失效通知
- 事件总线机制

**使用示例:**
```go
consistency := GetEnhancedCacheConsistency()

// 一致性读取
val, err := consistency.GetWithConsistency(ctx, "key")

// 一致性写入
err := consistency.SetWithConsistency(ctx, "key", value, ttl)

// 删除并通知
err := consistency.DeleteWithConsistency(ctx, "key")

// 按标签失效
err := consistency.InvalidateByTag(ctx, "tag_name")
```

### 5. 缓存监控和命中率

**文件:** `backend/pkg/redis/cache_monitoring_v2.go`

**监控指标:**
- 命中率（L1、L2、总命中率）
- 延迟分布（P50、P95、P99、P999）
- 热键追踪
- 趋势分析
- 健康检查

**告警规则:**
- 低命中率告警（<80%）
- 高错误率告警（>5%）
- 高延迟告警（P99 >100ms）

**使用示例:**
```go
monitoring := GetCacheMonitoringService()

// 获取当前指标
metrics := monitoring.GetMetrics()
fmt.Printf("Hit Rate: %.2f%%\n", metrics["hit_rate"])

// 获取热键
hotKeys := monitoring.GetHotKeys(10)

// 获取最近告警
alerts := monitoring.GetAlerts(10)

// 获取趋势
trend := monitoring.GetTrend()
predictedRate := monitoring.PredictHitRate()
```

## 配置说明

### 环境变量配置

```bash
# Redis连接池
REDIS_POOL_SIZE=200
REDIS_MIN_IDLE_CONNS=20
REDIS_MAX_IDLE_CONNS=100
REDIS_ENABLE_AUTO_TUNING=true

# L1缓存
CACHE_L1_ENABLED=true
CACHE_L1_MAX_SIZE=10000
CACHE_L1_MAX_MEMORY=104857600
CACHE_L1_TTL=300

# L2缓存
CACHE_L2_ENABLED=true
CACHE_L2_DEFAULT_TTL=1800
CACHE_L2_MAX_TTL=7200

# 预热
CACHE_WARMUP_ENABLED=true
CACHE_WARMUP_PRIORITY_ENABLED=true
CACHE_WARMUP_ADAPTIVE_ENABLED=true
CACHE_WARMUP_SMART_TRIGGER_ENABLED=true

# 一致性
CACHE_CONSISTENCY_ENABLED=true
CACHE_CONSISTENCY_LEVEL=eventual
CACHE_INVALIDATION_STRATEGY=pubsub

# 监控
CACHE_MONITORING_ENABLED=true
CACHE_METRICS_ENABLED=true
CACHE_HIT_RATE_ALERT_THRESHOLD=80
```

## 性能优化建议

### 1. 连接池调优

- 根据实际QPS调整pool_size
- 监控超时和连接数调整
- 开启自动调优功能

### 2. 缓存容量规划

- L1内存建议控制在100MB以内
- 根据数据量调整L1最大条目数
- 合理设置TTL避免内存浪费

### 3. 预热策略

- 关键数据启动时预热
- 峰值时段前智能预热
- 根据访问频率动态调整

### 4. 一致性选择

- 读多写少：使用最终一致
- 写多读少：使用写穿透
- 关键数据：使用强一致

## 测试验证

运行测试验证功能:
```bash
cd backend
go test -v ./pkg/redis/... -run TestCache
```

## 监控指标

建议使用Prometheus监控以下指标:
- `cache_hit_rate`: 缓存命中率
- `cache_l1_hit_rate`: L1缓存命中率
- `cache_l2_hit_rate`: L2缓存命中率
- `cache_latency_seconds`: 缓存延迟分布
- `cache_errors_total`: 缓存错误总数
- `cache_evictions_total`: 缓存驱逐总数
