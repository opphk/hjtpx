# 错误处理系统文档

## 概述

本文档描述了前后端统一的错误处理系统实现。

## 错误码定义

### 后端错误码 (ErrorCode)

错误码采用前缀分类系统：

- `AUTH_*`: 认证相关错误
- `VAL_*`: 验证相关错误
- `DB_*`: 数据库相关错误
- `SRV_*`: 服务器相关错误
- `SEC_*`: 安全相关错误
- `FILE_*`: 文件相关错误

### 前端错误码

前端使用相同的错误码体系，确保API响应与前端错误处理一致。

## 错误类

### 后端错误类

```javascript
// AppError - 基础错误类
// AuthenticationError - 认证错误
// ValidationError - 验证错误
// NotFoundError - 未找到错误
// DatabaseError - 数据库错误
// SecurityError - 安全错误
// RateLimitError - 速率限制错误
```

### 前端错误类

```javascript
// AppError - 基础错误类
// NetworkError - 网络错误
// TimeoutError - 超时错误
// AuthenticationError - 认证错误
// ValidationError - 验证错误
// NotFoundError - 未找到错误
```

## 统一响应格式

```json
{
  "success": false,
  "error": {
    "code": "AUTH_001",
    "message": "Invalid credentials",
    "category": "Authentication",
    "statusCode": 401,
    "details": null,
    "timestamp": "2024-01-01T00:00:00.000Z"
  }
}
```

## 中间件使用

### 后端

```javascript
const { errorHandler, notFoundHandler, asyncHandler } = require('./middleware/errorHandler');

// 在Express应用中使用
app.use(errorHandler);
app.use(notFoundHandler);

// 使用asyncHandler包装异步路由
app.get('/api/users', asyncHandler(async (req, res) => {
  // 异步代码
}));
```

### 前端

```javascript
import { errorHandler, handleApiError, AppError } from './utils/errorHandler';

// 监听错误
const unsubscribe = errorHandler.addListener((error) => {
  console.error('Error occurred:', error);
});

// 处理API错误
try {
  await apiCall();
} catch (error) {
  const handledError = handleApiError(error);
}
```

## 错误日志

错误日志记录到 `logs/errors/` 目录，包含：
- 错误消息
- 错误码
- 堆栈跟踪
- 请求上下文
- 时间戳

## 测试覆盖

- 后端错误处理中间件测试
- 前端错误类测试
- 错误码功能测试
- 错误日志服务测试

## 最佳实践

1. 始终使用统一错误码
2. 在生产环境隐藏详细错误信息
3. 记录所有错误以便调试
4. 使用asyncHandler处理异步错误
5. 为不同类型的错误使用专门的错误类
