# HJTPX 缓存最佳实践指南

## 目录

1. [概述](#概述)
2. [多级缓存架构](#多级缓存架构)
3. [缓存使用场景](#缓存使用场景)
4. [缓存配置指南](#缓存配置指南)
5. [性能优化建议](#性能优化建议)
6. [缓存失效策略](#缓存失效策略)
7. [分布式缓存一致性](#分布式缓存一致性)
8. [监控与告警](#监控与告警)
9. [故障排除](#故障排除)

## 概述

本文档提供了 HJTPX 项目中缓存系统的最佳实践指南。缓存是提高应用性能的关键组件，正确的缓存策略可以显著提升系统响应速度和吞吐量。

### 缓存的优势

- **提高性能**：减少数据库查询次数，降低响应延迟
- **减轻负载**：减少对后端服务的请求压力
- **提升可用性**：在部分服务故障时提供降级支持
- **成本优化**：减少计算资源和数据库连接的使用

### 缓存的风险

- **数据一致性**：缓存数据可能与源数据不同步
- **复杂度增加**：系统架构变得更加复杂
- **内存消耗**：缓存占用系统内存资源
- **运维难度**：需要监控和管理缓存系统

## 多级缓存架构

HJTPX 采用三级缓存架构，每一级都有其特定用途和优势。

### L1 缓存（进程内内存）

**位置**：应用进程内存中
**实现**：Map 数据结构 + LRU 策略
**特点**：

- 访问速度最快（纳秒级）
- 进程内共享，无需网络开销
- 容量受限，受应用内存限制
- 重启后数据丢失

**适用场景**：

- 热点数据访问
- 高频访问的配置信息
- 会话数据（短期）

**配置建议**：

```javascript
{
  enableL1: true,
  l1MaxSize: 1000,      // 最大条目数
  l1Ttl: 60000          // 过期时间（毫秒）
}
```

### L2 缓存（Redis 分布式缓存）

**位置**：独立的 Redis 服务器
**特点**：

- 访问速度快（毫秒级）
- 支持分布式部署
- 数据持久化（可选）
- 支持丰富的数据结构

**适用场景**：

- 跨进程共享数据
- 会话管理
- 分布式锁
- 热点数据缓存

**配置建议**：

```javascript
{
  enableL2: true,
  l2Ttl: 300,           // 默认过期时间（秒）
  compressionThreshold: 1024  // 压缩阈值（字节）
}
```

### L3 缓存（数据库）

**位置**：PostgreSQL 数据库
**特点**：

- 数据持久化
- 支持复杂查询
- 事务支持
- 数据可靠

**适用场景**：

- 冷数据存储
- 需要持久化的配置
- 审计日志
- 备份数据

## 缓存使用场景

### 1. 会话缓存

**场景**：用户登录状态管理

**实现**：

```javascript
const sessionKey = cacheService.generateSessionKey(sessionToken);
await cacheService.setSession(sessionToken, sessionData, 604800);
const session = await cacheService.getSession(sessionToken);
```

**TTL 配置**：

- 活跃会话：7 天（604800 秒）
- 不活跃会话：1 天

**注意事项**：

- 会话数据不包含敏感密码信息
- 定期检查会话有效性
- 实现会话续期机制

### 2. 用户数据缓存

**场景**：用户信息快速访问

**实现**：

```javascript
const userKey = cacheService.generateUserKey(userId);
const user = await cacheService.getCachedUser(userId);

if (!user) {
  user = await database.users.findById(userId);
  await cacheService.setCachedUser(userId, user);
}
```

**TTL 配置**：

- 用户信息：30 分钟
- 用户权限：1 小时
- 用户配置：24 小时

**注意事项**：

- 用户更新时及时失效缓存
- 权限变更需要清除相关缓存
- 考虑缓存击穿问题

### 3. API 响应缓存

**场景**：减少重复计算和数据库查询

**实现**：

```javascript
const apiKey = cacheService.generateApiKey(endpoint, userId, params);
const cached = await cacheService.getCachedApiResponse(apiKey);

if (cached) {
  return cached;
}

const response = await fetchDataFromDatabase();
await cacheService.setCachedApiResponse(apiKey, response, isPublic);
```

**TTL 配置**：

- 公开数据：5 分钟
- 私有数据：1 分钟
- 统计数据：5 分钟

**注意事项**：

- 根据数据变化频率设置 TTL
- 考虑缓存雪崩问题
- 使用标签管理相关缓存

### 4. 热点数据缓存

**场景**：高访问量的数据

**实现**：

```javascript
const hotDataKey = `hot:${dataType}:${id}`;
await advancedCacheService.set(hotDataKey, data, {
  ttl: 3600,
  priority: 'high'
});
```

**策略**：

- 启动时预热热点数据
- 运行时监控访问频率
- 动态调整热点数据缓存

### 5. 分布式锁

**场景**：并发控制、资源竞争

**实现**：

```javascript
const lock = await cacheConsistency.acquireLock('resource:lock');

if (lock.acquired) {
  try {
    await processResource();
  } finally {
    await cacheConsistency.releaseLock('resource:lock', lock.value);
  }
}
```

**注意事项**：

- 设置合理的锁超时时间
- 实现锁续期机制
- 处理锁获取失败的情况

## 缓存配置指南

### 基础配置

```javascript
const cacheConfig = {
  // L1 缓存配置
  enableL1: true,
  l1MaxSize: 1000,
  l1Ttl: 60000,

  // L2 缓存配置
  enableL2: true,
  l2Ttl: 300,
  
  // L3 缓存配置
  enableL3: true,

  // 压缩配置
  enableCompression: true,
  compressionThreshold: 1024,

  // 分片配置
  enableSharding: false,
  shardCount: 4
};
```

### 环境特定配置

#### 开发环境

```javascript
{
  enableL1: true,
  enableL2: false,      // 开发环境可能没有 Redis
  enableL3: true,
  l1MaxSize: 100,
  l1Ttl: 30000
}
```

#### 生产环境

```javascript
{
  enableL1: true,
  enableL2: true,
  enableL3: true,
  l1MaxSize: 1000,
  l1Ttl: 60000,
  l2Ttl: 300,
  enableCompression: true,
  compressionThreshold: 1024,
  enableSharding: true,
  shardCount: 4
}
```

### TTL 配置参考

| 数据类型 | TTL | 说明 |
|---------|-----|------|
| 会话数据 | 604800 秒（7 天） | 用户登录会话 |
| 用户信息 | 1800 秒（30 分钟） | 基本用户信息 |
| 用户权限 | 3600 秒（1 小时） | 权限列表 |
| API 公开响应 | 300 秒（5 分钟） | 公开接口数据 |
| API 私有响应 | 60 秒（1 分钟） | 私有接口数据 |
| 热点数据 | 3600 秒（1 小时） | 高频访问数据 |
| 统计数据 | 300 秒（5 分钟） | 聚合统计数据 |
| 配置信息 | 86400 秒（1 天） | 系统配置 |

## 性能优化建议

### 1. 缓存命中率优化

**关键指标**：

```
命中率 = 缓存命中数 / 总访问数 × 100%
```

**优化策略**：

- 合理设置 TTL，避免数据过早过期
- 实现缓存预热机制
- 使用缓存标签管理相关数据
- 监控热点数据并优先缓存

**监控命令**：

```javascript
const stats = advancedCacheService.getStats();
console.log(`L1 命中率: ${stats.l1.hitRate}`);
console.log(`L2 命中率: ${stats.l2.hitRate}`);
console.log(`总体命中率: ${stats.total.hitRate}`);
```

### 2. 延迟优化

**目标**：

- L1 延迟：< 1ms
- L2 延迟：< 10ms
- L3 延迟：< 100ms

**优化方法**：

- 使用连接池
- 启用压缩减少传输数据量
- 使用 Pipeline 批量操作
- 优化网络拓扑

### 3. 内存使用优化

**策略**：

- 限制 L1 缓存大小
- 实现 LRU 淘汰策略
- 使用缓存分片
- 定期清理过期数据

**监控内存使用**：

```javascript
const memoryStats = advancedCacheService.getStats().memory;
console.log(`内存使用: ${memoryStats.usedFormatted}`);
console.log(`峰值使用: ${memoryStats.peakFormatted}`);
```

### 4. 批量操作优化

**批量获取**：

```javascript
const keys = ['user:1', 'user:2', 'user:3'];
const users = await advancedCacheService.getMulti(keys);
```

**批量设置**：

```javascript
const items = {
  'user:1': userData1,
  'user:2': userData2,
  'user:3': userData3
};
await advancedCacheService.setMulti(items, 1800);
```

### 5. 异步操作优化

**异步缓存更新**：

```javascript
// 不阻塞主流程
setImmediate(async () => {
  await advancedCacheService.set(key, data);
});

// 或使用队列
await queue.add('cache-update', { key, data });
```

## 缓存失效策略

### 1. TTL 过期策略

**实现**：

```javascript
await cacheService.set(key, value, {
  ttl: 300,  // 5 分钟后自动过期
  bypassL1: false,
  bypassL2: false
});
```

**建议**：

- 热点数据设置较短 TTL
- 冷数据设置较长 TTL
- 使用随机 TTL 避免雪崩

### 2. LRU 淘汰策略

**自动淘汰**：

```javascript
// 当缓存满时自动淘汰最少使用的条目
if (this.l1Cache.size >= this.config.l1MaxSize) {
  this.evictFromL1();
}
```

**手动触发**：

```javascript
advancedCacheService.evictFromL1();
```

### 3. 手动失效机制

**单条失效**：

```javascript
await advancedCacheService.delete('user:123');
```

**模式失效**：

```javascript
await advancedCacheService.invalidatePattern('user:*');
```

**标签失效**：

```javascript
await advancedCacheService.invalidateTag('user:123');
```

### 4. 批量失效机制

**批量删除**：

```javascript
const keys = ['user:1', 'user:2', 'user:3'];
for (const key of keys) {
  await advancedCacheService.delete(key);
}
```

**条件失效**：

```javascript
// 失效所有用户相关的缓存
await advancedCacheService.invalidateAllUserCache(userId);
```

### 5. 主动刷新策略

**定时刷新**：

```javascript
// 每小时刷新一次
setInterval(async () => {
  await cacheWarmer.performScheduledWarming();
}, 3600000);
```

**被动刷新**：

```javascript
// 数据更新时刷新缓存
await advancedCacheService.set(key, newValue, { ttl: 300 });
```

## 分布式缓存一致性

### 1. 缓存锁机制

**获取锁**：

```javascript
const lock = await cacheConsistency.acquireLock('resource:update', {
  timeout: 10000,
  retries: 50,
  delay: 100
});

if (lock.acquired) {
  try {
    await updateResource();
  } finally {
    await cacheConsistency.releaseLock('resource:update', lock.value);
  }
}
```

**锁续期**：

```javascript
await cacheConsistency.extendLock('resource:update', lockValue, 20000);
```

### 2. 版本控制

**版本递增**：

```javascript
const newVersion = await cacheConsistency.incrementVersion(key);
```

**版本检查**：

```javascript
const version = await cacheConsistency.getVersion(key);
```

**监听变更**：

```javascript
cacheConsistency.onVersionChange(key, (data) => {
  console.log('Cache updated:', data);
});
```

### 3. 事务支持

**缓存事务**：

```javascript
const tx = await cacheConsistency.startTransaction();
await tx.set('user:1', user1);
await tx.set('user:2', user2);
await tx.set('user:3', user3);
await tx.commit(advancedCacheService);
```

### 4. 一致性模式

**Cache-Aside（旁路缓存）**：

```javascript
const result = await cacheConsistency.cacheAside(
  'user:123',
  () => database.users.findById(123),
  { ttl: 300 }
);
```

**Write-Through（写穿透）**：

```javascript
await cacheConsistency.writeThrough(
  'user:123',
  userData,
  { ttl: 300, writeFn: (data) => database.users.update(data) }
);
```

**Write-Behind（写回）**：

```javascript
await cacheConsistency.writeBehind(
  'user:123',
  userData,
  { ttl: 300, writeFn: (data) => database.users.update(data) }
);
```

### 5. 乐观锁

```javascript
const result = await cacheConsistency.compareAndSet(
  'counter:views',
  currentValue,
  newValue,
  { ttl: 300 }
);
```

## 缓存预热机制

### 1. 启动预热

```javascript
cacheWarmer.warmOnStartup().then(() => {
  console.log('Cache warmed successfully');
});
```

### 2. 定时预热

```javascript
cacheWarmer.updateConfig({
  scheduled: {
    enabled: true,
    interval: 3600000  // 每小时执行一次
  }
});
```

### 3. 热点数据预热

```javascript
cacheWarmer.performHotDataWarming();
```

### 4. 自定义预热

```javascript
await cacheWarmer.warmCustomCache([
  { key: 'config:app', value: appConfig, ttl: 86400 },
  { key: 'config:feature', value: featureFlags, ttl: 3600 }
]);
```

### 5. 按标签预热

```javascript
await cacheWarmer.warmByTag('user');
await cacheWarmer.warmByPattern('api:popular:*');
```

## 监控与告警

### 1. 性能监控

**获取统计数据**：

```javascript
const stats = advancedCacheService.getStats();
console.log(JSON.stringify(stats, null, 2));
```

**输出示例**：

```json
{
  "l1": {
    "hits": 1000,
    "misses": 100,
    "hitRate": "90.91%",
    "size": 500,
    "maxSize": 1000
  },
  "l2": {
    "hits": 95,
    "misses": 5,
    "hitRate": "95.00%",
    "connected": true
  },
  "total": {
    "hits": 1095,
    "misses": 105,
    "hitRate": "91.25%"
  }
}
```

### 2. 监控报告

**生成报告**：

```javascript
const report = cacheMonitor.generateReport();
console.log(JSON.stringify(report, null, 2));
```

**导出指标**：

```javascript
const jsonReport = cacheMonitor.exportMetrics('json');
const csvReport = cacheMonitor.exportMetrics('csv');
```

### 3. 告警配置

**配置告警阈值**：

```javascript
cacheMonitor.setAlertThresholds({
  hitRate: { warning: 60, critical: 40 },
  latency: { warning: 100, critical: 500 },
  memory: { warning: 80, critical: 95 },
  errorRate: { warning: 5, critical: 10 }
});
```

**查看活跃告警**：

```javascript
const alerts = cacheMonitor.getAlerts();
console.log('Active alerts:', alerts);
```

### 4. 健康检查

```javascript
const health = cacheMonitor.getHealthStatus();
console.log('Cache health:', health);
```

### 5. 关键监控指标

| 指标 | 正常范围 | 警告阈值 | 严重阈值 |
|------|----------|----------|----------|
| 缓存命中率 | > 80% | 60-80% | < 60% |
| L1 延迟 | < 1ms | 1-5ms | > 5ms |
| L2 延迟 | < 10ms | 10-50ms | > 50ms |
| 内存使用率 | < 70% | 70-90% | > 90% |
| 错误率 | < 1% | 1-5% | > 5% |

## 故障排除

### 1. 缓存未命中率高

**可能原因**：

- TTL 设置过短
- 缓存被意外清除
- 数据未正确写入缓存
- 热点数据未被预热

**排查步骤**：

1. 检查 TTL 配置
2. 查看缓存写入日志
3. 检查是否有批量清除操作
4. 分析访问模式

**解决方案**：

```javascript
// 增加 TTL
await cacheService.set(key, value, { ttl: 3600 });

// 实现缓存预热
await cacheWarmer.warmCustomCache(hotData);
```

### 2. 缓存延迟过高

**可能原因**：

- Redis 服务器负载高
- 网络延迟
- 大数据未压缩
- 连接池配置不当

**排查步骤**：

1. 检查 Redis 服务器性能
2. 测量网络延迟
3. 审查数据大小
4. 检查连接池配置

**解决方案**：

```javascript
// 启用压缩
await cacheService.set(key, value, { compress: true });

// 优化连接池
redisClient.configure({
  maxRetries: 3,
  retryDelay: 100
});
```

### 3. 内存使用过高

**可能原因**：

- L1 缓存过大
- 未清理过期数据
- 数据结构设计不当
- 内存泄漏

**排查步骤**：

1. 检查 L1 缓存大小
2. 查看内存使用统计
3. 分析数据大小
4. 检查是否有内存泄漏

**解决方案**：

```javascript
// 限制 L1 缓存大小
advancedCacheService.config.l1MaxSize = 500;

// 手动触发清理
advancedCacheService.cleanup();

// 使用缓存分片
const sharding = new CacheSharding({ shardCount: 8 });
```

### 4. 数据一致性问题

**可能原因**：

- 多级缓存数据不同步
- 并发更新导致冲突
- 缓存失效策略不当
- 分布式环境下的竞态条件

**排查步骤**：

1. 检查版本控制是否正常工作
2. 分析并发更新场景
3. 审查失效策略
4. 检查锁机制

**解决方案**：

```javascript
// 使用分布式锁
await cacheConsistency.withLock(key, async () => {
  const data = await fetchData();
  await cacheService.set(key, data);
});

// 使用乐观锁
const result = await cacheConsistency.compareAndSet(
  key, expectedValue, newValue
);

// 使用版本控制
await cacheConsistency.incrementVersion(key);
```

### 5. Redis 连接问题

**可能原因**：

- Redis 服务器宕机
- 网络连接失败
- 认证失败
- 连接池耗尽

**排查步骤**：

1. 检查 Redis 服务器状态
2. 测试网络连通性
3. 验证认证信息
4. 检查连接池配置

**解决方案**：

```javascript
// 实现降级策略
const cacheService = new AdvancedCacheService({
  enableL2: true,
  fallbackToL1: true
});

// 健康检查
const isHealthy = await advancedCacheService.isHealthy();

// 重连机制
redisClient.on('error', (err) => {
  console.error('Redis error:', err);
  redisClient.reconnect();
});
```

## 总结

缓存是提升系统性能的重要手段，但需要谨慎使用。遵循以下原则：

1. **适度缓存**：不要过度依赖缓存，保持数据一致性
2. **合理 TTL**：根据数据特性设置合适的过期时间
3. **多层架构**：利用多级缓存提高性能和可用性
4. **监控告警**：持续监控缓存性能，及时发现问题
5. **容错设计**：实现降级策略，确保系统稳定性
6. **文档记录**：记录缓存策略和使用方式，便于维护

通过遵循这些最佳实践，可以充分发挥缓存的优势，提高系统性能和用户体验。
