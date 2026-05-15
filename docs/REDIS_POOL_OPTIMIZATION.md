# Redis连接池优化文档

## 概述

本文档描述了Redis连接池的优化实现，包括连接池管理、健康检查和连接泄漏检测功能。

## 文件结构

```
src/backend/
├── config/database/
│   └── redisPool.js          # Redis连接池配置
├── services/
│   └── redis_service.js      # Redis服务实现
└── tests/database/
    └── redisPool.test.js     # 性能测试
```

## 功能特性

### 1. 连接池配置优化

**文件**: [redisPool.js](file:///workspace/hjtpx/src/backend/config/database/redisPool.js)

- **连接参数配置**:
  - `connectTimeout`: 10000ms - 连接超时时间
  - `commandTimeout`: 5000ms - 命令执行超时
  - `keepAlive`: 30000ms - Keep-Alive间隔
  - `maxRetriesPerRequest`: 3 - 最大重试次数

- **重连策略**:
  - 指数退避: 100ms → 3000ms
  - 最大重试次数: 20次

### 2. 健康检查机制

**功能**:
- 定时健康检查（默认30秒间隔）
- 响应时间监控
- 连接状态实时追踪
- 慢查询警告

**指标收集**:
- `totalConnections`: 总连接数
- `failedConnections`: 失败连接数
- `totalCommands`: 总命令数
- `averageLatency`: 平均延迟

### 3. 连接泄漏检测

**实现**:
- 命令执行时间跟踪
- 可配置的泄漏阈值
- 泄漏事件通知
- 自动泄漏记录

**配置**:
```javascript
connectionLeakThreshold: 30000  // 30秒
```

### 4. 性能测试

**文件**: [redisPool.test.js](file:///workspace/hjtpx/src/backend/tests/database/redisPool.test.js)

测试覆盖:
- ✅ 基础操作测试 (SET/GET/DEL/EXISTS/TTL)
- ✅ Hash操作测试 (HSET/HGET/HGETALL)
- ✅ 批量操作测试 (MSET/MGET)
- ✅ 并发操作测试 (100并发请求)
- ✅ 健康检查测试
- ✅ 性能基准测试 (1000次操作)
- ✅ 连接泄漏检测测试

## 使用示例

```javascript
const RedisPoolManager = require('./services/redis_service');

// 获取连接池管理器
const pool = require('./services/redis_service');

// 基本操作
await pool.set('key', 'value', { EX: 3600 });
const value = await pool.get('key');

// 获取统计信息
const stats = pool.getStats();
console.log('平均延迟:', stats.averageLatency);
console.log('总命令数:', stats.totalCommands);

// 健康检查
const health = await pool.healthCheck();
console.log('健康状态:', health.status);
```

## 性能指标

| 指标 | 目标值 | 说明 |
|------|--------|------|
| 平均延迟 | < 10ms | 正常操作 |
| 连接成功率 | > 99.9% |  |
| 健康检查频率 | 30秒 | 可配置 |
| 最大重试次数 | 20 | 指数退避 |

## 配置参数

### 环境变量

```bash
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_HEALTH_CHECK_INTERVAL=30000
REDIS_COMMAND_TIMEOUT=5000
```

### 自定义配置

```javascript
const pool = new RedisConnectionPool({
  host: 'redis.example.com',
  port: 6380,
  password: 'secret',
  db: 1,
  connectTimeout: 15000,
  commandTimeout: 5000,
  maxRetriesPerRequest: 5
});
```

## 监控指标

### Prometheus指标

```javascript
// 建议导出到Prometheus
redis_pool_connections_total
redis_pool_commands_total
redis_pool_latency_seconds
redis_pool_errors_total
```

## 优化建议

1. **生产环境**:
   - 使用Redis集群提高可用性
   - 配置合适的连接池大小
   - 启用连接监控

2. **性能优化**:
   - 使用管道(Pipeline)批量操作
   - 合理设置TTL避免过期数据
   - 使用压缩减少网络传输

3. **监控告警**:
   - 设置延迟阈值告警
   - 监控连接失败率
   - 定期审查慢查询日志

## 测试验证

```bash
# 运行性能测试
node src/backend/tests/database/redisPool.test.js

# 预期输出
✓ 基础操作测试通过
✓ Hash操作测试通过
✓ 批量操作测试通过
✓ 并发操作测试通过
✓ 健康检查测试通过
✓ 性能基准测试通过
✓ 连接泄漏检测测试通过
```

## 故障排除

### 连接失败
1. 检查Redis服务器状态
2. 验证网络连接
3. 检查认证信息

### 高延迟
1. 检查Redis服务器负载
2. 分析慢查询
3. 考虑连接池扩容

### 连接泄漏
1. 检查代码中的未关闭连接
2. 调整泄漏阈值
3. 启用自动清理

## 维护计划

- [ ] 每周审查连接池统计
- [ ] 每月更新依赖版本
- [ ] 每季度进行性能基准测试
- [ ] 持续监控关键指标

---

**版本**: 1.0.0  
**创建日期**: 2026-05-15  
**最后更新**: 2026-05-15
