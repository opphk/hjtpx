# 日志聚合功能集成指南

## 概述

后端日志聚合系统提供了完整的结构化日志记录、请求追踪和敏感信息保护功能。

## 核心功能

### 1. 结构化JSON日志

所有日志以JSON格式输出，包含以下字段：
- `timestamp`: ISO 8601格式时间戳
- `level`: 日志级别 (error, warn, info, http, debug)
- `message`: 日志消息
- `requestId`: 请求追踪ID
- `userId`: 用户ID（如果有）
- `ip`: 客户端IP地址
- `method`: HTTP方法
- `path`: 请求路径
- `statusCode`: 响应状态码
- `duration`: 请求持续时间(ms)

### 2. 日志分级管理

根据环境自动调整日志级别：
- `development`: 显示 debug 级别
- `production`: 显示 info 级别
- `test`: 禁用日志输出

### 3. 文件输出分离

日志文件按类型分离存储在 `logs/` 目录：
- `error-%DATE%.log`: 仅错误日志
- `warn-%DATE%.log`: 警告和错误日志
- `combined-%DATE%.log`: 所有日志
- `http-%DATE%.log`: HTTP请求日志（保留14天）
- `performance-%DATE%.log`: 性能日志（保留7天）
- `security-%DATE%.log`: 安全事件日志（保留30天）

## 使用方法

### 1. 在Express应用中集成中间件

```javascript
const express = require('express');
const requestIdMiddleware = require('./src/backend/middleware/requestId');
const sanitizeLogsMiddleware = require('./src/backend/middleware/sanitizeLogs');
const { logRequest, logError } = require('./src/backend/utils/logger');

const app = express();

app.use(requestIdMiddleware);
app.use(sanitizeLogsMiddleware);
app.use((req, res, next) => {
  const start = Date.now();
  res.on('finish', () => {
    const duration = Date.now() - start;
    logRequest(req, res, duration);
  });
  next();
});
```

### 2. 手动记录日志

```javascript
const { logInfo, logWarn, logError, logDebug } = require('./src/backend/utils/logger');

logInfo('用户登录成功', {
  userId: user.id,
  method: 'POST',
  path: '/api/auth/login'
});

logWarn('请求频率过高', {
  ip: req.ip,
  endpoint: '/api/search'
});

logError(new Error('数据库连接失败'), req);
```

### 3. 创建子日志记录器

```javascript
const { createChildLogger } = require('./src/backend/utils/logger');

const paymentLogger = createChildLogger({ module: 'payment' });

paymentLogger.info('支付处理完成', {
  transactionId: 'txn_123',
  amount: 99.99
});
```

### 4. 记录安全事件

```javascript
const { logSecurity } = require('./src/backend/utils/logger');

logSecurity('登录失败次数超限', {
  ip: '192.168.1.100',
  userId: 'user123',
  attempts: 5
});
```

### 5. 记录性能指标

```javascript
const { logPerformance } = require('./src/backend/utils/logger');

logPerformance('数据库查询', {
  query: 'SELECT * FROM users',
  duration: 234,
  rows: 150
});
```

## 环境配置

在 `.env` 文件中配置日志行为：

```env
LOG_LEVEL=info
LOG_FORMAT=json
LOG_CONSOLE=true
LOG_FILE=true
LOG_DIR=logs
LOG_MAX_SIZE=20m
LOG_MAX_FILES=30d
LOG_ZIP=true
LOG_SENSITIVE_FIELDS=password,token,authorization,cookie,secret,apiKey,creditCard
NODE_ENV=production
SERVICE_NAME=hjtpx-api
APP_VERSION=1.0.0
```

## 敏感信息保护

系统自动过滤以下敏感字段：
- `password`
- `token`
- `authorization`
- `cookie`
- `secret`
- `apiKey`
- `creditCard`
- 以及任何包含这些关键词的字段

所有敏感信息在日志中显示为 `***MASKED***`。

## 请求追踪

每个请求自动获得唯一ID：
- 格式: `req_{timestamp}_{uuid8}`
- 存储在 `req.requestId`
- 通过响应头 `X-Request-ID` 返回客户端
- 支持传入已有的 `X-Request-ID` 头

## 测试

运行日志系统测试：

```bash
npm test -- tests/logger/structured-log.test.js
```

运行所有测试：

```bash
npm test
```

## 日志分析示例

```javascript
const { Logger } = require('./src/backend/utils/logger');

Logger.query({
  from: new Date(Date.now() - 24 * 60 * 60 * 1000),
  until: new Date(),
  level: 'error',
  limit: 100
}).then(results => {
  console.log(`Found ${results.length} errors in the last 24 hours`);
});
```

## 最佳实践

1. **始终使用结构化数据**: 使用对象而非字符串拼接记录日志
2. **包含请求上下文**: 在日志中包含 requestId、userId 等追踪信息
3. **合理选择日志级别**: 
   - `error`: 实际错误
   - `warn`: 可恢复的问题
   - `info`: 重要业务事件
   - `debug`: 开发调试信息
4. **保护敏感信息**: 不要手动记录密码、token等敏感信息
5. **性能监控**: 使用 `logPerformance` 记录关键操作的耗时
6. **安全审计**: 使用 `logSecurity` 记录安全相关事件
