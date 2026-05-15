# Database Connection Pool Optimization Guide

## 概述

本文档说明了数据库连接池优化的实现，包括连接池管理器、健康检查、泄漏检测和监控功能。

## 目录

- [连接池管理器](#连接池管理器)
- [连接健康检查](#连接健康检查)
- [连接泄漏检测](#连接泄漏检测)
- [连接池监控](#连接池监控)
- [连接池优化器](#连接池优化器)
- [使用示例](#使用示例)

## 连接池管理器

### 位置
`src/backend/config/database/dbPoolManager.js`

### 主要功能
- 优化的连接池配置
- 自动健康检查
- 连接泄漏检测
- 查询统计和慢查询追踪
- 事务支持
- 批处理查询

### 配置参数

```javascript
{
  min: 最小连接数,
  max: 最大连接数,
  idleTimeoutMillis: 空闲超时时间,
  connectionTimeoutMillis: 连接超时时间,
  statementTimeout: 语句超时时间,
  keepAlive: 保持连接活跃
}
```

### 使用示例

```javascript
const dbPoolManager = require('./config/database/dbPoolManager');

// 初始化连接池
dbPoolManager.initialize();

// 执行查询
const result = await dbPoolManager.query('SELECT * FROM users WHERE id = $1', [userId]);

// 执行事务
const transactionResult = await dbPoolManager.transaction(async (client) => {
  const user = await client.query('SELECT * FROM users WHERE id = $1', [userId]);
  await client.query('UPDATE users SET last_login = NOW() WHERE id = $1', [userId]);
  return user;
});

// 获取连接池统计
const poolStats = dbPoolManager.getPoolStats();

// 获取查询统计
const queryStats = dbPoolManager.getQueryStats();

// 健康检查
const health = await dbPoolManager.healthCheck();

// 关闭连接池
await dbPoolManager.close();
```

## 连接健康检查

### 位置
`src/backend/services/connectionHealthCheck.js`

### 主要功能
- 定期健康检查
- 数据库连接性检查
- 连接池状态检查
- 查询性能检查
- 数据库大小检查
- 活跃连接检查
- 深度健康检查

### 使用示例

```javascript
const ConnectionHealthCheck = require('./services/connectionHealthCheck');
const dbPoolManager = require('./config/database/dbPoolManager');

const healthCheck = new ConnectionHealthCheck(dbPoolManager);

// 启动健康检查（每30秒一次）
healthCheck.start(30000);

// 执行单次健康检查
const health = await healthCheck.performHealthCheck();

// 执行深度健康检查
const deepHealth = await healthCheck.performDeepHealthCheck();

// 获取健康报告
const report = healthCheck.getHealthReport();

// 获取详细统计
const stats = healthCheck.getDetailedStats();

// 监听健康检查事件
healthCheck.on('healthCheck', (result) => {
  console.log('Health check result:', result);
});

healthCheck.on('unhealthyState', (alert) => {
  console.error('Database is unhealthy:', alert);
});

// 停止健康检查
healthCheck.stop();
```

## 连接泄漏检测

### 位置
`src/backend/services/connectionLeakDetector.js`

### 主要功能
- 跟踪连接使用情况
- 检测长时间未释放的连接
- 自动清理泄漏的连接
- 泄漏事件记录和告警
- 统计分析和趋势预测

### 使用示例

```javascript
const ConnectionLeakDetector = require('./services/connectionLeakDetector');
const dbPoolManager = require('./config/database/dbPoolManager');

const leakDetector = new ConnectionLeakDetector(dbPoolManager);

// 启动泄漏检测
leakDetector.start();

// 跟踪连接
const trackingInfo = leakDetector.trackConnection('client-123', {
  query: 'SELECT * FROM users',
  user: 'test_user'
});

// 取消跟踪连接
leakDetector.untrackConnection('client-123');

// 获取统计信息
const stats = leakDetector.getStatistics();

// 获取泄漏报告
const report = leakDetector.getLeakReport();

// 获取活跃连接列表
const activeConnections = leakDetector.getActiveConnections();

// 获取特定连接的详细信息
const connectionDetails = leakDetector.getConnectionDetails('client-123');

// 设置新的泄漏阈值
leakDetector.setThreshold(45000);

// 设置检查间隔
leakDetector.setCheckInterval(5000);

// 标记误报
leakDetector.markAsFalsePositive('client-123');

// 导出泄漏数据
const exportedData = leakDetector.exportLeakData('json');
const csvData = leakDetector.exportLeakData('csv');

// 监听泄漏事件
leakDetector.on('leaksDetected', (alert) => {
  console.error('Leaks detected:', alert);
});

leakDetector.on('criticalLeakAlert', (alert) => {
  console.error('Critical leak alert:', alert);
});

// 停止泄漏检测
leakDetector.stop();
```

## 连接池监控

### 位置
`src/backend/services/connectionPoolMonitor.js`

### 主要功能
- 实时指标收集
- 连接池状态监控
- 查询性能监控
- 系统资源监控
- 告警管理
- 趋势分析
- 报告生成

### 使用示例

```javascript
const ConnectionPoolMonitor = require('./services/connectionPoolMonitor');
const dbPoolManager = require('./config/database/dbPoolManager');

const monitor = new ConnectionPoolMonitor(dbPoolManager);

// 启动监控（每30秒生成报告）
monitor.start(30000);

// 获取健康报告
const healthReport = monitor.generateHealthReport();

// 获取聚合指标
const aggregated = monitor.getAggregatedMetrics('1h');

// 获取告警列表
const alerts = monitor.getAlerts('critical');

// 获取特定时间范围的告警
const recentAlerts = monitor.getAlerts(null, 20);

// 获取历史指标
const history = monitor.getMetricsHistory('pool_stats', 100);

// 强制收集指标
const currentMetrics = monitor.forceCollection();

// 导出指标数据
const exportedMetrics = monitor.exportMetrics('json');

// 设置告警阈值
monitor.setAlertThreshold('highConnectionUsage', 0.80);
monitor.setAlertThreshold('highQueryTime', 2000);

// 监听告警事件
monitor.on('alert', (alert) => {
  console.warn('Alert:', alert);
});

monitor.on('criticalAlert', (alert) => {
  console.error('Critical alert:', alert);
});

// 清除所有告警
monitor.clearAlerts();

// 停止监控
monitor.stop();
```

## 连接池优化器

### 位置
`src/backend/config/database/poolOptimizer.js`

### 主要功能
- 基于CPU核心数的连接池大小优化
- 基于内存的连接池大小计算
- 环境特定的配置
- 超时参数优化
- 健康检查配置
- 泄漏检测配置
- 配置验证

### 使用示例

```javascript
const ConnectionPoolOptimizer = require('./config/database/poolOptimizer');

const optimizer = new ConnectionPoolOptimizer();

// 获取最优连接池大小
const poolSize = optimizer.getOptimalPoolSize();
console.log('Optimal pool size:', poolSize);
// {
//   min: 5,
//   max: 20,
//   cpuCores: 8,
//   cpuBasedSize: 20,
//   memoryBasedSize: 30,
//   recommendedSize: 20,
//   reasoning: { ... }
// }

// 获取最优超时配置
const timeouts = optimizer.getOptimalTimeouts();
console.log('Optimal timeouts:', timeouts);

// 获取最优健康检查配置
const healthCheck = optimizer.getOptimalHealthCheckConfig();
console.log('Health check config:', healthCheck);

// 获取最优泄漏检测配置
const leakDetection = optimizer.getOptimalLeakDetectionConfig();
console.log('Leak detection config:', leakDetection);

// 获取完整配置
const completeConfig = optimizer.getCompleteConfiguration();
console.log('Complete config:', completeConfig);

// 验证配置
const validation = optimizer.validateConfiguration(completeConfig);
console.log('Validation:', validation);
// {
//   valid: true,
//   errors: [],
//   warnings: []
// }
```

## 环境变量配置

### 连接池配置
```bash
DB_HOST=localhost
DB_PORT=5432
DB_NAME=hjtpx
DB_USER=postgres
DB_PASSWORD=your_password
DB_POOL_MIN=5
DB_POOL_MAX=20
DB_IDLE_TIMEOUT=60000
DB_CONNECTION_TIMEOUT=10000
DB_STATEMENT_TIMEOUT=60000
```

### 健康检查配置
```bash
DB_HEALTH_CHECK_INTERVAL=30000
DB_HEALTH_MAX_RESPONSE_TIME=5000
DB_HEALTH_MIN_IDLE=2
DB_HEALTH_MAX_USAGE=0.85
```

### 泄漏检测配置
```bash
DB_LEAK_THRESHOLD=30000
DB_LEAK_CHECK_INTERVAL=10000
DB_LEAK_MAX_RECORDS=100
DB_LEAK_AUTO_CLEANUP=true
DB_LEAK_AUTO_CLEANUP_TIMEOUT=60000
```

### 监控配置
```bash
MONITOR_HIGH_CONNECTION_USAGE=0.85
MONITOR_CRITICAL_CONNECTION_USAGE=0.95
MONITOR_HIGH_QUERY_TIME=1000
MONITOR_CRITICAL_QUERY_TIME=5000
MONITOR_HIGH_ERROR_RATE=0.05
MONITOR_CRITICAL_ERROR_RATE=0.10
MONITOR_HIGH_POOL_WAITERS=5
MONITOR_CRITICAL_POOL_WAITERS=10
```

## 最佳实践

### 1. 生产环境配置
```javascript
// 生产环境推荐配置
const optimizer = new ConnectionPoolOptimizer();
// 自动根据CPU核心数和内存优化
const config = optimizer.getCompleteConfiguration();
```

### 2. 监控和告警
```javascript
// 设置关键告警
monitor.setAlertThreshold('criticalConnectionUsage', 0.95);
monitor.setAlertThreshold('criticalQueryTime', 5000);

monitor.on('criticalAlert', (alert) => {
  // 发送告警通知
  sendAlert(alert);
});
```

### 3. 定期健康检查
```javascript
// 启动定期健康检查
healthCheck.start(30000); // 每30秒

// 在应用关闭时清理
process.on('SIGTERM', () => {
  healthCheck.stop();
  leakDetector.stop();
  monitor.stop();
  dbPoolManager.close();
});
```

### 4. 性能优化
```javascript
// 使用批处理查询
const queries = [
  { query: 'INSERT INTO logs (msg) VALUES ($1)', params: ['log1'] },
  { query: 'INSERT INTO logs (msg) VALUES ($1)', params: ['log2'] },
  { query: 'INSERT INTO logs (msg) VALUES ($1)', params: ['log3'] }
];
await dbPoolManager.batchQuery(queries);
```

## 故障排除

### 连接池耗尽
如果遇到连接池耗尽的问题：
1. 检查是否有未释放的连接
2. 增加 `DB_POOL_MAX` 配置
3. 使用泄漏检测器分析问题

### 查询超时
如果查询经常超时：
1. 增加 `DB_STATEMENT_TIMEOUT` 配置
2. 优化慢查询
3. 增加数据库索引

### 健康检查失败
如果健康检查失败：
1. 检查数据库连接
2. 查看错误日志
3. 调整超时阈值

## 性能指标

### 关键指标
- **连接利用率**: busy connections / total connections
- **查询响应时间**: p50, p95, p99
- **错误率**: errors / total queries
- **泄漏检测**: detected leaks over time

### 告警阈值
- **警告**: 连接利用率 > 85%
- **严重**: 连接利用率 > 95%
- **警告**: p95查询时间 > 1000ms
- **严重**: p95查询时间 > 5000ms
- **警告**: 错误率 > 5%
- **严重**: 错误率 > 10%

## 相关文件

- `src/backend/config/database/dbPoolManager.js` - 连接池管理器
- `src/backend/config/database/poolOptimizer.js` - 连接池优化器
- `src/backend/services/connectionHealthCheck.js` - 健康检查服务
- `src/backend/services/connectionLeakDetector.js` - 泄漏检测服务
- `src/backend/services/connectionPoolMonitor.js` - 监控服务

## 测试

运行连接池相关测试：
```bash
npm test -- --testPathPattern="poolManager|connectionHealthCheck|connectionLeakDetector|connectionPoolMonitor|poolOptimizer"
```

## 总结

通过使用这些优化工具，你可以：
1. 自动优化连接池大小
2. 实时监控系统性能
3. 快速检测和解决连接泄漏问题
4. 提高数据库连接的整体可靠性
5. 减少数据库相关的生产问题
