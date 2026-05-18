# 错误码说明文档

## 通用错误码

| 错误码 | HTTP状态码 | 错误信息 | 说明 |
|--------|-----------|----------|------|
| 0 | 200 | success | 操作成功 |
| 400 | 400 | Bad Request | 请求参数错误 |
| 401 | 401 | Unauthorized | 未授权或Token无效 |
| 403 | 403 | Forbidden | 权限不足或账户被禁用 |
| 404 | 404 | Not Found | 资源不存在 |
| 409 | 409 | Conflict | 资源冲突（如用户名已存在） |
| 500 | 500 | Internal Server Error | 服务器内部错误 |

## 认证模块错误码

| 错误码 | HTTP状态码 | 错误信息 | 说明 |
|--------|-----------|----------|------|
| 1001 | 401 | invalid username or password | 用户名或密码错误 |
| 1002 | 403 | account is disabled | 账户已被禁用 |
| 1003 | 401 | token expired | Token已过期 |
| 1004 | 401 | invalid token | Token无效 |
| 1005 | 400 | invalid old password | 原密码错误 |
| 1006 | 400 | password must be at least 6 characters | 密码长度不足 |
| 1007 | 400 | invalid reset token | 重置Token无效 |
| 1008 | 400 | reset token has expired | 重置Token已过期 |
| 1009 | 400 | email already verified | 邮箱已验证 |
| 1010 | 400 | invalid verification token | 验证Token无效 |

## 验证码模块错误码

| 错误码 | HTTP状态码 | 错误信息 | 说明 |
|--------|-----------|----------|------|
| 2001 | 400 | invalid request parameters | 无效的请求参数 |
| 2002 | 404 | session not found or expired | 会话不存在或已过期 |
| 2003 | 400 | verification type mismatch | 验证类型不匹配 |
| 2004 | 400 | click position mismatch | 点击位置不匹配 |
| 2005 | 400 | click count mismatch | 点击数量不匹配 |
| 2006 | 400 | click sequence error | 点击顺序错误 |
| 2007 | 400 | slider position error | 滑块位置偏差过大 |
| 2008 | 400 | high risk score | 风险评分过高 |
| 2009 | 400 | verification failed | 验证失败 |

## 应用模块错误码

| 错误码 | HTTP状态码 | 错误信息 | 说明 |
|--------|-----------|----------|------|
| 3001 | 404 | user not found | 用户不存在 |
| 3002 | 404 | application not found | 应用不存在 |
| 3003 | 400 | invalid application name | 无效的应用名称 |
| 3004 | 400 | application name already exists | 应用名称已存在 |
| 3005 | 400 | invalid API key | 无效的API密钥 |
| 3006 | 400 | rate limit exceeded | 请求频率超限 |

## 风控模块错误码

| 错误码 | HTTP状态码 | 错误信息 | 说明 |
|--------|-----------|----------|------|
| 4001 | 403 | IP blocked | IP已被封禁 |
| 4002 | 429 | rate limit exceeded | 请求频率超限 |
| 4003 | 403 | device blocked | 设备已被封禁 |
| 4004 | 403 | suspicious activity detected | 检测到可疑活动 |
| 4005 | 400 | behavior risk score too high | 行为风险评分过高 |

## 响应格式示例

### 成功响应
```json
{
  "code": 0,
  "message": "success",
  "data": {
    // 业务数据
  }
}
```

### 错误响应
```json
{
  "code": 400,
  "message": "请求参数错误",
  "error": {
    "field": "username",
    "reason": "username cannot be empty"
  }
}
```

## 常见错误场景

### 1. 登录失败
- 错误码: 1001
- 原因: 用户名或密码错误
- 建议: 检查输入的用户名和密码是否正确

### 2. Token过期
- 错误码: 1003
- 原因: 访问Token已过期
- 建议: 使用RefreshToken获取新的访问Token

### 3. 验证码验证失败
- 错误码: 2009
- 原因: 验证码验证失败，可能原因：
  - 滑动位置偏差过大
  - 点击顺序错误
  - 行为分析风险评分过高
- 建议: 重新获取验证码进行验证

### 4. 应用不存在
- 错误码: 3002
- 原因: 请求中引用的应用ID不存在
- 建议: 检查应用ID是否正确，或创建新应用

### 5. IP被封禁
- 错误码: 4001
- 原因: 当前IP被风控系统封禁
- 建议: 联系管理员解封或等待解封时间到期
