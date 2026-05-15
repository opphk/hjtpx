# HJTPX 消息队列使用指南

## 概述

本文档详细介绍 HJTPX 项目中消息队列的架构设计、配置方法和使用实践。项目采用 Redis Streams 作为消息队列解决方案，充分利用现有 Redis 基础设施实现高性能、低延迟的消息传递。

### 技术选型理由

选择 Redis Streams 而非其他消息队列方案（如 RabbitMQ）主要基于以下考量：项目已广泛使用 Redis 作为缓存和数据存储，无需额外部署消息中间件；Redis Streams 提供持久化、有序性和消费者组等企业级特性；支持消息重试、死信队列和监控等高级功能；与现有 ioredis 客户端完全兼容。

### 架构概览

消息队列系统由以下核心组件构成：连接管理器负责 Redis 连接的生命周期管理；生产者服务负责向各队列发送消息；消费者服务负责从队列读取和处理消息；重试机制提供指数退避和重试策略；死信队列处理无法正常处理的消息；事件系统支持发布订阅模式；监控系统实时监控队列健康状态。

## 快速开始

### 环境配置

确保 Redis 服务正常运行，并在环境变量中配置连接信息：

```bash
# Redis 连接配置
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# 消息队列启用开关
STREAMS_ENABLED=true
```

### 初始化消息队列

在应用程序启动时初始化消息队列管理器：

```javascript
const messageQueueManager = require('./src/backend/services/message_queue');

async function startApp() {
  await messageQueueManager.initialize();
  
  await messageQueueManager.startConsumers({
    email: true,
    notification: true,
    export: true,
    logging: true,
    events: true
  });
}

startApp().catch(console.error);
```

## 核心组件详解

### 连接管理器

连接管理器（`connectionManager`）是整个消息队列系统的基础，负责与 Redis 建立和维护连接。该组件提供自动重连、连接池管理和健康检查等核心功能。

```javascript
const connectionManager = require('./src/backend/services/message_queue/connectionManager');

// 建立连接
await connectionManager.connect();

// 获取 Redis 客户端
const client = connectionManager.getClient();

// 健康检查
const health = await connectionManager.healthCheck();
console.log('Redis 连接状态:', health);

// 创建流和消费者组
await connectionManager.ensureStream('hjtpx:streams:email', 10000);
await connectionManager.ensureConsumerGroup('hjtpx:streams:email', 'hjtpx:consumers:email', '0');
```

### 生产者服务

生产者服务负责将消息发送到指定的队列。系统支持多种发送模式，包括普通发送、批量发送和延迟发送。

```javascript
const { producerManager } = require('./src/backend/services/message_queue/producers');

// 发送单条消息
const result = await producerManager.send('email', {
  to: 'user@example.com',
  templateName: 'welcome',
  variables: { username: '张三' }
}, {
  type: 'send_email',
  priority: 5,
  correlationId: 'unique-id-123'
});

// 批量发送
const results = await producerManager.sendBatch('notification', [
  { userId: 'user1', message: '消息1' },
  { userId: 'user2', message: '消息2' }
]);

// 延迟发送（5秒后执行）
await producerManager.sendWithDelay('email', {
  to: 'scheduled@example.com',
  templateName: 'notification'
}, 5000);

// 获取队列长度
const length = await producerManager.getProducer('email').getQueueLength();
console.log('邮件队列当前消息数:', length);
```

### 消费者服务

消费者服务从队列中读取消息并进行处理。每个消费者属于一个消费者组，支持消息确认、重试和负载均衡。

```javascript
const { consumerManager } = require('./src/backend/services/message_queue/consumers');

// 创建消费者
const consumer = await consumerManager.createConsumer('email', {
  consumerName: 'worker-1',
  blockTimeout: 5000,
  batchSize: 10
});

// 注册消息处理器
consumer.registerHandler('send_email', async (message) => {
  const { to, templateName, variables } = message.payload;
  console.log(`处理邮件: 收件人=${to}, 模板=${templateName}`);
  
  const emailService = require('./src/backend/services/emailService');
  await emailService.sendEmail(to, templateName, variables);
});

// 设置错误处理器
consumer.setErrorHandler(async (error) => {
  console.error('消费错误:', error);
  // 可发送到告警系统
});

// 启动消费（阻塞调用）
await consumer.consume();

// 处理待处理消息（消费者崩溃后恢复）
await consumer.consumePending();

// 停止消费者
consumer.stop();

// 健康检查
const health = await consumer.healthCheck();
```

## 异步任务处理

### 邮件发送队列

邮件队列专门用于处理电子邮件发送任务，支持模板渲染、批量发送和定时发送。

```javascript
const emailQueueService = require('./src/backend/services/message_queue/emailQueueService');

// 发送单封邮件
await emailQueueService.sendEmail('user@example.com', 'welcome', {
  username: '张三',
  appUrl: 'https://hjtpx.com'
}, { priority: 5 });

// 发送欢迎邮件
await emailQueueService.sendWelcomeEmail({
  _id: 'user-id-123',
  username: '新用户',
  email: 'newuser@example.com'
});

// 发送密码重置邮件
await emailQueueService.sendPasswordResetEmail({
  _id: 'user-id-456',
  username: '用户甲',
  email: 'user@example.com'
}, 'reset-token-xyz');

// 批量发送
await emailQueueService.sendBulkEmails([
  { email: 'user1@example.com', variables: { name: '用户1' } },
  { email: 'user2@example.com', variables: { name: '用户2' } }
], 'notification', { message: '系统通知' });

// 定时发送
await emailQueueService.scheduleEmail(
  'user@example.com',
  'notification',
  { title: '定时提醒', message: '这是一个定时消息' },
  new Date(Date.now() + 3600000) // 一小时后发送
);
```

### 通知推送队列

通知队列处理应用内通知、推送通知等多种通知类型。

```javascript
const notificationQueueService = require('./src/backend/services/message_queue/notificationQueueService');

// 发送推送通知
await notificationQueueService.sendPushNotification(
  'user-id-123',
  '新消息',
  '您收到一条新私信',
  { messageId: 'msg-456', type: 'direct_message' }
);

// 发送应用内通知
await notificationQueueService.sendInAppNotification('user-id-123', {
  title: '订单已发货',
  message: '您的订单 #12345 已发货',
  data: { orderId: '12345', trackingNumber: 'SF123456789' }
);

// 批量发送通知
await notificationQueueService.sendBulkNotifications(
  ['user-1', 'user-2', 'user-3'],
  { type: 'push', title: '系统公告', body: '系统将于今晚维护' }
);

// 定时通知
await notificationQueueService.scheduleNotification(
  'user-id-123',
  { type: 'push', title: '会议提醒', body: '会议将在30分钟后开始' },
  new Date(Date.now() + 1800000)
);
```

### 数据导出队列

导出队列处理大规模数据导出任务，支持 CSV、Excel 和 PDF 等多种格式。

```javascript
const exportQueueService = require('./src/backend/services/message_queue/exportQueueService');

// 导出用户数据
await exportQueueService.exportUsers('admin-user-id', {
  filters: { status: 'active', createdAt: { $gte: '2024-01-01' } },
  fields: ['id', 'username', 'email', 'createdAt']
});

// 导出分析数据
await exportQueueService.exportAnalytics('admin-user-id', {
  startDate: '2024-01-01',
  endDate: '2024-12-31'
});

// 导出为 CSV
await exportQueueService.exportToCSV('user-id', 'users', {
  filters: { status: 'active' }
});

// 导出为 Excel
await exportQueueService.exportToExcel('user-id', 'analytics', {
  dateRange: { start: '2024-01-01', end: '2024-06-30' }
});

// 导出为 PDF
await exportQueueService.exportToPDF('user-id', 'report', {
  reportType: 'monthly_summary'
});
```

### 日志处理队列

日志队列异步处理应用程序日志、安全事件和性能指标。

```javascript
const loggingQueueService = require('./src/backend/services/message_queue/loggingQueueService');

// 记录不同级别的日志
await loggingQueueService.logError('数据库连接失败', error, {
  host: 'db.example.com',
  query: 'SELECT * FROM users'
});

await loggingQueueService.logWarn('请求处理时间过长', {
  endpoint: '/api/users',
  duration: 2500
});

await loggingQueueService.logInfo('用户操作', {
  userId: 'user-123',
  action: 'update_profile'
});

await loggingQueueService.logDebug('缓存命中', {
  key: 'user:123:profile',
  ttl: 3600
});

// 记录安全事件
await loggingQueueService.logSecurityEvent('login_failed', {
  userId: 'user-123',
  ip: '192.168.1.100',
  reason: 'invalid_password',
  attempts: 3
});

// 记录用户操作
await loggingQueueService.logUserAction('user-123', 'create_post', {
  postId: 'post-456',
  category: 'technology'
});

// 记录性能指标
await loggingQueueService.logPerformanceMetric('api_response_time', 250, {
  unit: 'ms',
  endpoint: '/api/users',
  tags: ['production', 'high_traffic']
});
```

## 事件驱动架构

### 事件类型定义

系统预定义了丰富的事件类型，涵盖用户、通知、导出、安全和分析等领域。

```javascript
const { EventTypes, EventCategories, createEvent, createUserEvent } = require('./src/backend/services/message_queue/events');

// 使用预定义事件类型
console.log(EventTypes.USER_CREATED);   // 'user.created'
console.log(EventTypes.NOTIFICATION_CREATED);  // 'notification.created'
console.log(EventTypes.SECURITY_EVENT);  // 'security.event'

// 创建自定义事件
const event = createEvent('custom.action', {
  entityId: 'entity-123',
  action: 'process',
  result: 'success'
}, {
  source: 'my-service',
  correlationId: 'correlation-123'
});
```

### 发布事件

事件发布者将事件发送到 Redis 频道，供订阅者消费。

```javascript
const eventPublisher = require('./src/backend/services/message_queue/events/eventPublisher');

// 发布用户创建事件
await eventPublisher.publishUserCreated({
  _id: 'user-123',
  username: '张三',
  email: 'zhangsan@example.com',
  createdAt: new Date()
});

// 发布用户登录事件
await eventPublisher.publishUserLoggedIn({
  _id: 'user-123',
  username: '张三'
}, {
  ip: '192.168.1.100',
  userAgent: 'Mozilla/5.0...'
});

// 发布通知创建事件
await eventPublisher.publishNotificationCreated({
  _id: 'notification-456',
  userId: 'user-123',
  type: 'in_app',
  title: '新消息',
  message: '您收到一条新私信'
});

// 发布导出完成事件
await eventPublisher.publishExportCompleted(
  'export-789',
  'user-123',
  'https://cdn.example.com/exports/export-789.csv'
);

// 发布安全事件
await eventPublisher.publishSecurityEvent('suspicious_activity', {
  type: 'multiple_login_failed',
  userId: 'user-123',
  ip: '10.0.0.1',
  attempts: 10
}, { severity: 'high' });
```

### 订阅事件

事件订阅者监听并处理发布的事件。

```javascript
const { eventSubscriber, EventTypes } = require('./src/backend/services/message_queue/events');

// 订阅特定事件
await eventSubscriber.subscribe(EventTypes.USER_CREATED, async (event) => {
  console.log('新用户注册:', event.data);
  
  // 发送欢迎邮件
  const emailQueueService = require('./src/backend/services/message_queue/emailQueueService');
  await emailQueueService.sendWelcomeEmail(event.data.user);
});

await eventSubscriber.subscribe(EventTypes.USER_LOGGED_IN, async (event) => {
  console.log('用户登录:', event.data.userId);
  
  // 更新用户在线状态
  const sessionService = require('./src/backend/services/sessionService');
  await sessionService.updateLastActive(event.data.userId);
});

// 使用模式订阅多个事件
await eventSubscriber.subscribeToPattern(
  'hjtpx:events:security:*',
  async (event) => {
    console.log('安全事件:', event.type, event.data);
    
    // 发送告警通知
    const notificationQueueService = require('./src/backend/services/message_queue/notificationQueueService');
    await notificationQueueService.sendPushNotification(
      'admin-user-id',
      '安全告警',
      `${event.data.type}: ${event.data.details}`
    );
  }
);

// 订阅所有用户相关事件
await eventSubscriber.subscribeToPattern('hjtpx:events:user.*', async (event) => {
  console.log('用户事件:', event.type, event.data);
});

// 开始监听
await eventSubscriber.startListening();

// 取消订阅
await eventSubscriber.unsubscribe(EventTypes.USER_CREATED);

// 取消所有订阅
await eventSubscriber.unsubscribeAll();
```

### 事件处理器

事件处理器提供更高级的事件处理能力，包括处理器注册和批量处理。

```javascript
const { eventProcessor } = require('./src/backend/services/message_queue/events');

// 注册自定义处理器
eventProcessor.registerHandler('user.created', async (event) => {
  console.log('处理用户创建事件');
  // 初始化用户配置
  // 发送欢迎礼包
});

eventProcessor.registerHandler('notification.created', async (event) => {
  console.log('处理通知创建事件');
  // 推送实时通知
  // 记录分析数据
});

// 发布事件
await eventProcessor.publishEvent({
  type: 'custom.event',
  data: { key: 'value' }
});

// 健康检查
const health = await eventProcessor.healthCheck();
```

## 重试机制与死信队列

### 重试策略

系统采用指数退避策略进行消息重试，避免瞬时故障导致的消息丢失。

```javascript
const { retryManager, RetryStrategy } = require('./src/backend/services/message_queue/retry');

// 使用预定义策略
await retryManager.executeWithRetry(
  async () => {
    // 执行业务逻辑
    return await processMessage();
  },
  {
    strategy: 'conservative',  // 保守策略：5次重试，最大延迟60秒
    operationName: 'processMessage'
  }
);

// 自定义策略
const customStrategy = new RetryStrategy({
  maxAttempts: 3,
  initialDelay: 1000,
  maxDelay: 10000,
  backoffMultiplier: 2,
  jitter: true
});

retryManager.registerStrategy('custom', customStrategy);

// 获取重试统计
const stats = retryManager.getStats('processMessage');
console.log('重试统计:', stats);

// 重置统计
retryManager.resetStats('processMessage');
```

### 死信队列管理

无法正常处理的消息会被转移到死信队列，便于后续分析和处理。

```javascript
const deadLetterQueue = require('./src/backend/services/message_queue/retry/deadLetterQueue');

// 获取 DLQ 统计
const dlqStats = await deadLetterQueue.getAllDLQStats();
console.log('所有 DLQ 状态:', dlqStats);

// 获取特定 DLQ 的消息
const messages = await deadLetterQueue.getDLQMessages('hjtpx:streams:email:dlq', 10);
console.log('邮件 DLQ 消息:', messages);

// 重新处理 DLQ 消息
const reprocessed = await deadLetterQueue.reprocessDLQ(
  'hjtpx:streams:email:dlq',
  'hjtpx:streams:email'
);
console.log('重新处理的消息数:', reprocessed);

// 注册自定义处理器
deadLetterQueue.registerProcessor('email', async (message) => {
  console.log('处理死信消息:', message.originalMessageId);
  // 尝试修复并重新处理
});

// 使用处理器处理 DLQ
const results = await deadLetterQueue.processDLQ('email', {
  stream: 'hjtpx:streams:email',
  count: 5
});
console.log('处理结果:', results);

// 清空 DLQ（谨慎使用）
await deadLetterQueue.purgeDLQ('hjtpx:streams:email:dlq');
```

## 监控与告警

### 队列监控

监控系统实时跟踪队列健康状态，包括消息数量、消费者状态和处理延迟。

```javascript
const queueMonitor = require('./src/backend/services/message_queue/monitoring/queueMonitor');

// 获取所有队列的指标
const allMetrics = queueMonitor.getAllMetrics();
console.log('队列指标:', allMetrics);

// 获取特定队列的指标
const emailMetrics = queueMonitor.getMetrics('email');
console.log('邮件队列指标:', emailMetrics);

// 获取详细统计
const detailedStats = await queueMonitor.getDetailedStats();
console.log('详细统计:', detailedStats);

// 注册告警处理器
queueMonitor.registerAlertHandler('queue_length', async (alert) => {
  console.error('队列长度告警:', alert.message);
  // 发送告警通知
});

queueMonitor.registerAlertHandler('dead_letter', async (alert) => {
  console.error('死信队列告警:', alert.message);
  // 触发告警
});

// 健康检查
const health = await queueMonitor.healthCheck();
console.log('监控健康状态:', health);
```

### 健康检查汇总

消息队列管理器提供统一的健康检查接口。

```javascript
const messageQueueManager = require('./src/backend/services/message_queue');

// 完整的健康检查
const health = await messageQueueManager.healthCheck();
console.log('消息队列健康状态:', health);

// 检查结果示例
// {
//   healthy: true,
//   isRunning: true,
//   isInitialized: true,
//   checks: {
//     connection: { healthy: true, latency: 2 },
//     producers: { ... },
//     consumers: { ... },
//     monitor: { ... },
//     eventPublisher: { ... },
//     eventSubscriber: { ... },
//     eventProcessor: { ... },
//     dlqStats: { ... }
//   }
// }
```

## 最佳实践

### 消息设计

消息应包含足够的上下文信息，以便消费者能够独立处理。推荐的消息格式包括：唯一标识符（messageId）、消息类型（type）、负载数据（payload）、时间戳（timestamp）和可选的元数据（metadata）。

```javascript
// 推荐的消息格式
const message = {
  id: uuidv4(),
  type: 'process_order',
  payload: {
    orderId: 'order-123',
    userId: 'user-456',
    items: [...],
    totalAmount: 299.99
  },
  timestamp: new Date().toISOString(),
  metadata: {
    source: 'web',
    correlationId: 'session-789'
  }
};
```

### 错误处理

消费者应实现健壮的错误处理逻辑，区分可重试错误和不可重试错误。

```javascript
consumer.registerHandler('process_order', async (message) => {
  try {
    // 业务逻辑
    await processPayment(message.payload);
  } catch (error) {
    if (error.code === 'INSUFFICIENT_FUNDS') {
      // 不可重试的错误，直接标记完成
      await handlePaymentFailure(message.payload.orderId, error.message);
    } else {
      // 可重试的错误，抛出异常触发重试机制
      throw error;
    }
  }
});
```

### 并发控制

合理设置消费者数量和批处理大小，避免资源耗尽。

```javascript
// 根据服务器资源调整配置
const consumer = await consumerManager.createConsumer('orders', {
  consumerName: `worker-${process.env.INSTANCE_ID}`,
  batchSize: 10,        // 每批处理10条消息
  blockTimeout: 5000,    // 阻塞等待5秒
  concurrency: 5         // 并发处理数
});
```

### 监控告警阈值

根据业务需求调整监控告警阈值。

```bash
# 环境变量配置
MQ_MONITORING_ENABLED=true
MQ_METRICS_INTERVAL=30000
MQ_ALERT_QUEUE_LENGTH=1000
MQ_ALERT_PROCESSING_TIME=30000
MQ_ALERT_FAILURE_RATE=0.1
```

## 故障排查

### 常见问题

消费者无法连接：请检查 Redis 连接配置，确认网络连通性，验证消费者组是否存在。

消息堆积：检查消费者是否正常运行，分析处理耗时，增加消费者实例或优化处理逻辑。

消息丢失：确保消息确认机制正常工作，避免手动删除未处理的消息。

死信队列增长：分析失败原因，修复业务逻辑后重新处理死信消息。

### 日志分析

启用详细日志便于问题排查：

```javascript
// 设置日志级别
process.env.LOG_LEVEL = 'debug';

// 查看消费者日志
// [StreamConsumer] Message received: 1234567890-0
// [StreamConsumer] Processing message...
// [StreamConsumer] Message processed in 150ms
```

### 调试工具

Redis 提供了丰富的命令用于调试消息队列：

```bash
# 查看流信息
XINFO STREAM hjtpx:streams:email

# 查看消费者组
XINFO GROUPS hjtpx:streams:email

# 查看待处理消息
XPENDING hjtpx:streams:email hjtpx:consumers:email

# 读取消息（消费者组外）
XREAD COUNT 10 STREAMS hjtpx:streams:email $

# 查看死信队列
XRANGE hjtpx:streams:email:dlq - + COUNT 10
```

## API 参考

### messageQueueManager

消息队列管理器主接口，提供所有队列服务的统一访问。

| 方法 | 说明 | 返回值 |
|------|------|--------|
| initialize() | 初始化消息队列系统 | Promise<void> |
| startConsumers(options) | 启动消费者 | Promise<void> |
| stop() | 停止所有消费者 | Promise<void> |
| healthCheck() | 健康检查 | Promise<HealthStatus> |
| getEmailService() | 获取邮件队列服务 | EmailQueueService |
| getNotificationService() | 获取通知队列服务 | NotificationQueueService |
| getExportService() | 获取导出队列服务 | ExportQueueService |
| getLoggingService() | 获取日志队列服务 | LoggingQueueService |

### producerManager

生产者管理器，负责消息发送。

| 方法 | 说明 | 参数 |
|------|------|------|
| send(queueName, message, options) | 发送单条消息 | 队列名、消息内容、选项 |
| sendBatch(queueName, messages, options) | 批量发送 | 队列名、消息数组、选项 |
| sendWithDelay(queueName, message, delay, options) | 延迟发送 | 队列名、消息、延迟毫秒、选项 |
| getProducer(queueName) | 获取生产者实例 | 队列名 |

### consumerManager

消费者管理器，负责消息消费。

| 方法 | 说明 | 参数 |
|------|------|------|
| createConsumer(queueName, options) | 创建消费者 | 队列名、选项 |
| getConsumer(queueName) | 获取消费者实例 | 队列名 |
| startAll() | 启动所有消费者 | - |
| stopAll() | 停止所有消费者 | - |

## 更新日志

### v1.0.0（2026-05-15）

初始版本发布，包含以下功能：基于 Redis Streams 的消息队列实现；邮件、通知、导出、日志四大异步任务队列；完整的事件驱动架构；消息重试与死信队列；队列监控与告警系统。
