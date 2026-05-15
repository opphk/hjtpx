# 细粒度限流系统

## 概述

本系统实现了多维度的细粒度限流控制，支持按用户、IP、端点等不同维度进行限流。

## 特性

### 1. 多维度限流
- **IP限流**: 限制每个IP地址的请求频率
- **用户限流**: 限制每个用户的请求频率
- **端点限流**: 限制每个API端点的请求频率
- **组合限流**: 同时应用多个维度的限流

### 2. 动态限流规则
- **负载感知**: 根据服务器CPU/内存负载动态调整限流阈值
- **时间感知**: 根据时段（峰值/非峰值）调整限流
- **行为感知**: 根据用户异常行为动态调整限流

### 3. 限流豁免
- IP白名单
- 用户豁免
- 全局豁免

## 使用方法

### 预定义限流器

```javascript
const { rateLimiters } = require('./middleware/rateLimiterAdvanced');

// 登录限流（更严格）
router.post('/auth/login', rateLimiters.auth, handler);

// 搜索限流
router.get('/search', rateLimiters.search, handler);

// 文件上传限流
router.post('/upload', rateLimiters.upload, handler);

// 管理员限流
router.post('/admin/action', rateLimiters.admin, handler);
```

### 自定义限流器

```javascript
const { advancedLimiter } = require('./middleware/rateLimiterAdvanced');

const customLimiter = advancedLimiter.createMultiDimensionalLimiter({
  dimensions: ['ip', 'user', 'endpoint'],
  limits: {
    ip: { max: 50, window: 60000 },
    user: { max: 100, window: 60000 },
    endpoint: { max: 200, window: 60000 }
  },
  skip: (req) => req.user?.role === 'admin'
});
```

## 限流响应

### 429 Too Many Requests

```json
{
  "success": false,
  "error": "Too Many Requests",
  "message": "Rate limit exceeded for ip",
  "limit": 100,
  "remaining": 0,
  "reset": 1621234567890,
  "retryAfter": 60,
  "dimension": "ip",
  "details": [
    {
      "dimension": "ip",
      "allowed": false,
      "remaining": 0,
      "limit": 100,
      "reset": 1621234567890
    }
  ]
}
```

## 限流头

每个响应都包含以下限流头：

- `X-RateLimit-Limit`: 当前限制的最大请求数
- `X-RateLimit-Remaining`: 剩余可用请求数
- `X-RateLimit-Reset`: 限流重置时间戳（Unix时间戳）

## 配置

### 默认配置

```javascript
// src/backend/config/rateLimit.js
module.exports = {
  default: {
    windowMs: 60000,
    max: 100
  }
};
```

### 端点配置

```javascript
endpoints: {
  '/api/auth/login': {
    windowMs: 300000,
    max: 5
  }
}
```

## 最佳实践

1. **选择合适的限流维度**
   - 登录：使用IP+用户双重限流
   - 公开API：使用IP限流
   - 私有API：使用用户限流

2. **设置合理的阈值**
   - 根据业务需求调整
   - 考虑正常用户行为
   - 预留突发流量空间

3. **使用Redis集群**
   - 确保高可用性
   - 支持分布式限流
   - 提高性能

4. **监控和分析**
   - 跟踪限流触发情况
   - 分析异常流量
   - 持续优化配置
