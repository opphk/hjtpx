# API 错误码说明文档

## 概述

本文档定义了系统中所有API错误码及其含义，便于开发者理解和处理API调用过程中可能遇到的各种错误情况。

## 错误码分类

### 1. 客户端错误 (4xx)

| 错误码 | HTTP状态码 | 错误信息 | 说明 |
| :--- | :--- | :--- | :--- |
| 1001 | 400 | Invalid request | 请求参数格式错误或缺少必要参数 |
| 1002 | 400 | Invalid parameter | 参数值不符合要求 |
| 1003 | 400 | Validation failed | 数据校验失败 |
| 1004 | 400 | Missing required field | 缺少必填字段 |
| 1005 | 400 | Invalid JSON format | JSON格式错误 |
| 1006 | 400 | Invalid email format | 邮箱格式不正确 |
| 1007 | 400 | Invalid phone format | 手机号格式不正确 |
| 1008 | 400 | Password too weak | 密码强度不足 |
| 1009 | 400 | Captcha expired | 验证码已过期 |
| 1010 | 400 | Captcha invalid | 验证码不正确 |
| 1011 | 400 | Rate limit exceeded | 请求频率超限 |
| 1012 | 400 | Invalid file type | 文件类型不支持 |
| 1013 | 400 | File too large | 文件大小超过限制 |

### 2. 认证与授权错误 (4xx)

| 错误码 | HTTP状态码 | 错误信息 | 说明 |
| :--- | :--- | :--- | :--- |
| 2001 | 401 | Unauthorized | 未授权访问 |
| 2002 | 401 | Invalid token | Token无效 |
| 2003 | 401 | Token expired | Token已过期 |
| 2004 | 401 | Token not found | Token不存在 |
| 2005 | 403 | Forbidden | 禁止访问 |
| 2006 | 403 | Insufficient permissions | 权限不足 |
| 2007 | 401 | Invalid credentials | 用户名或密码错误 |
| 2008 | 401 | Account locked | 账户已被锁定 |
| 2009 | 401 | Account disabled | 账户已被禁用 |
| 2010 | 401 | Email not verified | 邮箱未验证 |

### 3. 资源错误 (4xx)

| 错误码 | HTTP状态码 | 错误信息 | 说明 |
| :--- | :--- | :--- | :--- |
| 3001 | 404 | Resource not found | 资源不存在 |
| 3002 | 404 | User not found | 用户不存在 |
| 3003 | 404 | Application not found | 应用不存在 |
| 3004 | 404 | Channel not found | 通道不存在 |
| 3005 | 404 | Rule not found | 规则不存在 |
| 3006 | 404 | Alert not found | 告警不存在 |
| 3007 | 409 | Resource already exists | 资源已存在 |
| 3008 | 409 | Email already registered | 邮箱已被注册 |
| 3009 | 409 | Username already taken | 用户名已被占用 |
| 3010 | 409 | Conflict | 资源冲突 |

### 4. 服务器错误 (5xx)

| 错误码 | HTTP状态码 | 错误信息 | 说明 |
| :--- | :--- | :--- | :--- |
| 5001 | 500 | Internal server error | 服务器内部错误 |
| 5002 | 500 | Database error | 数据库操作失败 |
| 5003 | 500 | Cache error | 缓存操作失败 |
| 5004 | 500 | External API error | 外部API调用失败 |
| 5005 | 500 | Service unavailable | 服务暂时不可用 |
| 5006 | 500 | Timeout | 请求超时 |
| 5007 | 500 | Encryption error | 加密操作失败 |
| 5008 | 500 | Decryption error | 解密操作失败 |
| 5009 | 500 | Backup failed | 备份操作失败 |
| 5010 | 500 | Restore failed | 恢复操作失败 |

### 5. 业务逻辑错误 (4xx)

| 错误码 | HTTP状态码 | 错误信息 | 说明 |
| :--- | :--- | :--- | :--- |
| 4001 | 400 | Risk score too high | 风险评分过高 |
| 4002 | 400 | Verification failed | 验证失败 |
| 4003 | 400 | Biometric mismatch | 生物特征不匹配 |
| 4004 | 400 | Not enough samples | 样本数量不足 |
| 4005 | 400 | Profile not found | 用户档案不存在 |
| 4006 | 400 | Profile not registered | 用户档案未注册 |
| 4007 | 400 | Operation not allowed | 操作不允许 |
| 4008 | 400 | Maintenance mode | 系统维护中 |
| 4009 | 400 | Feature disabled | 功能已禁用 |
| 4010 | 400 | Quota exceeded | 配额已用尽 |

## 错误响应格式

所有API错误响应遵循统一格式：

```json
{
  "code": 1001,
  "message": "Invalid request",
  "details": "The request body contains invalid JSON",
  "timestamp": "2024-01-15T10:30:00Z",
  "request_id": "abc123def456"
}
```

### 字段说明

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| code | integer | 错误码 |
| message | string | 错误简要描述 |
| details | string | 错误详细信息（可选） |
| timestamp | string | 错误发生时间（ISO 8601格式） |
| request_id | string | 请求唯一标识（用于日志追踪） |

## 错误处理建议

1. **客户端错误(10xx)**: 检查请求参数是否正确，确保所有必填字段都已提供且格式正确。

2. **认证错误(20xx)**: 检查Token是否有效，必要时重新获取Token或重新登录。

3. **资源错误(30xx)**: 确认资源ID是否正确，检查资源是否存在或是否已被删除。

4. **服务器错误(50xx)**: 记录错误信息并稍后重试，如问题持续存在请联系系统管理员。

5. **业务错误(40xx)**: 根据具体错误信息调整业务逻辑或用户操作。

## 示例

### 示例1：参数验证失败

```json
{
  "code": 1003,
  "message": "Validation failed",
  "details": "Email format is invalid",
  "timestamp": "2024-01-15T10:30:00Z",
  "request_id": "req-12345"
}
```

### 示例2：Token过期

```json
{
  "code": 2003,
  "message": "Token expired",
  "details": "Please refresh your token",
  "timestamp": "2024-01-15T10:30:00Z",
  "request_id": "req-67890"
}
```

### 示例3：资源不存在

```json
{
  "code": 3001,
  "message": "Resource not found",
  "details": "Application with id 999 does not exist",
  "timestamp": "2024-01-15T10:30:00Z",
  "request_id": "req-abcde"
}
```

---

*文档版本: 1.0*  
*最后更新: 2024年1月*
