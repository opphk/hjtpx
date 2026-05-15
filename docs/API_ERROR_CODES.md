# HJTPX API 错误码对照表

本文档详细列出了 HJTPX API 中所有可能的错误码、错误原因以及相应的解决方案。

## 目录

- [错误响应格式](#错误响应格式)
- [HTTP 状态码](#http-状态码)
- [业务错误码](#业务错误码)
  - [认证错误码](#认证错误码)
  - [用户管理错误码](#用户管理错误码)
  - [通知系统错误码](#通知系统错误码)
  - [系统级错误码](#系统级错误码)
- [常见错误场景与解决方案](#常见错误场景与解决方案)

---

## 错误响应格式

所有 API 错误响应都遵循统一的 JSON 格式：

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "错误描述信息",
    "details": []
  },
  "timestamp": "2026-05-15T10:30:00.000Z"
}
```

### 错误响应字段说明

| 字段 | 类型 | 描述 |
|------|------|------|
| success | boolean | 请求是否成功，错误时始终为 `false` |
| error | object | 错误详情对象 |
| error.code | string | 错误码，用于程序化处理 |
| error.message | string | 人类可读的错误描述 |
| error.details | array | 详细错误信息数组（可选） |
| timestamp | string | 服务器返回错误的时间戳 |

### 详细错误示例

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid input data",
    "details": [
      {
        "field": "email",
        "message": "请提供有效的邮箱地址",
        "type": "string.email"
      },
      {
        "field": "password",
        "message": "密码必须包含至少一个大写字母",
        "type": "string.pattern.base"
      }
    ]
  },
  "timestamp": "2026-05-15T10:30:00.000Z"
}
```

---

## HTTP 状态码

### 2xx - 成功状态码

| 状态码 | 名称 | 描述 |
|--------|------|------|
| 200 | OK | 请求成功，已返回预期结果 |
| 201 | Created | 资源创建成功 |
| 204 | No Content | 请求成功，但无返回内容（通常用于删除操作） |

### 4xx - 客户端错误状态码

| 状态码 | 名称 | 描述 |
|--------|------|------|
| 400 | Bad Request | 请求参数错误或无法解析 |
| 401 | Unauthorized | 未认证或认证失败 |
| 403 | Forbidden | 已认证但无访问权限 |
| 404 | Not Found | 请求的资源不存在 |
| 409 | Conflict | 请求与服务器状态冲突 |
| 422 | Unprocessable Entity | 请求格式正确但语义错误 |
| 429 | Too Many Requests | 请求频率超限，触发限流 |

### 5xx - 服务器错误状态码

| 状态码 | 名称 | 描述 |
|--------|------|------|
| 500 | Internal Server Error | 服务器内部错误 |
| 502 | Bad Gateway | 网关错误 |
| 503 | Service Unavailable | 服务暂时不可用 |
| 504 | Gateway Timeout | 网关超时 |

---

## 业务错误码

### 认证错误码

#### AUTH_ERROR

**描述**: 通用认证错误

**HTTP 状态码**: 401

**常见原因**:
- Token 生成失败
- 认证服务异常
- Session 创建失败

**解决方案**:
```javascript
// 重新登录获取新的 token
async function handleAuthError() {
  localStorage.removeItem('token');
  window.location.href = '/login';
}
```

---

#### UNAUTHORIZED

**描述**: 未授权访问

**HTTP 状态码**: 401

**常见原因**:
- 请求未携带 Token
- Token 已过期
- Token 格式不正确
- Token 被篡改或无效

**解决方案**:
```javascript
// 检查并刷新 token
async function ensureAuth() {
  const token = localStorage.getItem('token');
  
  if (!token) {
    throw new Error('No token available');
  }

  // 验证 token 是否有效
  const result = await fetch('/api/v1/auth/verify', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ token })
  });

  if (!result.data.valid) {
    // 尝试刷新 token
    const refreshResult = await fetch('/api/v1/auth/refresh', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token })
    });

    if (refreshResult.success) {
      localStorage.setItem('token', refreshResult.data.token);
      return refreshResult.data.token;
    }

    // 刷新失败，需要重新登录
    localStorage.removeItem('token');
    window.location.href = '/login';
  }

  return token;
}
```

---

#### INVALID_CREDENTIALS

**描述**: 无效的凭据

**HTTP 状态码**: 401

**常见原因**:
- 邮箱或密码错误
- 账户已被锁定
- 密码已过期

**错误响应示例**:
```json
{
  "success": false,
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Invalid email or password"
  }
}
```

**解决方案**:
- 检查输入的邮箱和密码是否正确
- 如果忘记密码，使用 `/api/v1/password/forgot` 请求重置
- 如果账户被锁定，等待 15 分钟后重试或联系管理员

---

#### TOKEN_EXPIRED

**描述**: Token 已过期

**HTTP 状态码**: 401

**常见原因**:
- Token 超过 7 天有效期
- 用户长时间未活动

**解决方案**:
```javascript
// 使用 refresh token 获取新 token
async function refreshToken() {
  const oldToken = localStorage.getItem('token');
  
  const response = await fetch('/api/v1/auth/refresh', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ token: oldToken })
  });

  if (response.success) {
    localStorage.setItem('token', response.data.token);
    return response.data.token;
  }
  
  throw new Error('Token refresh failed');
}
```

---

#### TOKEN_INVALID

**描述**: Token 无效

**HTTP 状态码**: 401

**常见原因**:
- Token 格式不正确
- Token 签名验证失败
- Token 已被撤销

**解决方案**:
- 清除本地存储的 token 并重新登录

---

#### ACCOUNT_LOCKED

**描述**: 账户已被锁定

**HTTP 状态码**: 401

**常见原因**:
- 连续 5 次登录失败
- 管理员手动锁定
- 安全策略触发锁定

**错误响应示例**:
```json
{
  "success": false,
  "error": {
    "code": "ACCOUNT_LOCKED",
    "message": "Account is temporarily locked due to multiple failed login attempts",
    "retryAfter": 900
  }
}
```

**解决方案**:
- 等待锁定时间结束（通常 15 分钟）
- 联系系统管理员解锁账户

---

#### REGISTRATION_ERROR

**描述**: 用户注册失败

**HTTP 状态码**: 400

**常见原因**:
- 邮箱格式不正确
- 密码不符合安全要求
- 邮箱已被注册
- 用户名不符合要求

**错误响应示例**:
```json
{
  "success": false,
  "error": {
    "code": "REGISTRATION_ERROR",
    "message": "Email already exists"
  }
}
```

**解决方案**:
```javascript
// 处理注册错误
async function handleRegistrationError(error) {
  if (error.code === 'REGISTRATION_ERROR') {
    if (error.message.includes('Email already exists')) {
      alert('该邮箱已被注册，请尝试登录或使用其他邮箱');
    } else if (error.message.includes('Password')) {
      alert('密码必须至少8个字符，包含大小写字母和数字');
    } else {
      alert('注册失败，请稍后重试');
    }
  }
}
```

---

### 用户管理错误码

#### FETCH_USERS_ERROR

**描述**: 获取用户列表失败

**HTTP 状态码**: 500

**常见原因**:
- 数据库连接失败
- 数据库查询超时
- 权限验证失败
- 服务器内部错误

**解决方案**:
```javascript
// 重试机制
async function fetchUsersWithRetry(retries = 3) {
  for (let i = 0; i < retries; i++) {
    try {
      const response = await fetch('/api/v1/users', {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      
      if (response.ok) {
        return response.json();
      }
      
      if (response.status === 500) {
        throw new Error('Server error');
      }
    } catch (error) {
      if (i === retries - 1) throw error;
      await new Promise(r => setTimeout(r, 1000 * (i + 1)));
    }
  }
}
```

---

#### FETCH_USER_ERROR

**描述**: 获取用户信息失败

**HTTP 状态码**: 500

**常见原因**:
- 用户不存在
- 数据库错误
- 权限不足

**解决方案**:
```javascript
async function getUser(userId) {
  const response = await fetch(`/api/v1/users/${userId}`, {
    headers: { 'Authorization': `Bearer ${token}` }
  });

  if (response.status === 404) {
    throw new Error('用户不存在');
  }

  return response.json();
}
```

---

#### CREATE_USER_ERROR

**描述**: 创建用户失败

**HTTP 状态码**: 400 或 500

**常见原因**:
- 邮箱格式不正确
- 密码不符合要求
- 邮箱已被占用
- 权限不足（仅管理员可创建）

**错误响应示例**:
```json
{
  "success": false,
  "error": {
    "code": "CREATE_USER_ERROR",
    "message": "Email already exists"
  }
}
```

**解决方案**:
```javascript
async function createUser(userData) {
  const response = await fetch('/api/v1/users', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`
    },
    body: JSON.stringify(userData)
  });

  if (!response.ok) {
    const error = await response.json();
    
    if (error.error.code === 'CREATE_USER_ERROR') {
      if (error.error.message.includes('Email')) {
        throw new Error('该邮箱已被注册');
      }
      throw new Error(error.error.message);
    }
  }

  return response.json();
}
```

---

#### UPDATE_USER_ERROR

**描述**: 更新用户失败

**HTTP 状态码**: 400 或 500

**常见原因**:
- 用户不存在
- 邮箱已被其他用户使用
- 权限不足
- 尝试修改受保护的字段

**解决方案**:
```javascript
async function updateUser(userId, updateData) {
  const response = await fetch(`/api/v1/users/${userId}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`
    },
    body: JSON.stringify(updateData)
  });

  if (response.status === 404) {
    throw new Error('用户不存在');
  }

  if (response.status === 403) {
    throw new Error('无权修改该用户');
  }

  return response.json();
}
```

---

#### DELETE_USER_ERROR

**描述**: 删除用户失败

**HTTP 状态码**: 500

**常见原因**:
- 用户不存在
- 数据库删除失败
- 权限不足
- 存在依赖关系无法删除

**解决方案**:
```javascript
async function deleteUser(userId) {
  const response = await fetch(`/api/v1/users/${userId}`, {
    method: 'DELETE',
    headers: { 'Authorization': `Bearer ${token}` }
  });

  if (response.status === 204) {
    return { success: true };
  }

  const error = await response.json();
  throw new Error(error.error?.message || '删除失败');
}
```

---

#### FORBIDDEN

**描述**: 禁止访问

**HTTP 状态码**: 403

**常见原因**:
- 用户角色权限不足
- 尝试访问受保护的资源
- 跨角色访问（如普通用户访问管理员功能）

**错误响应示例**:
```json
{
  "success": false,
  "error": {
    "code": "FORBIDDEN",
    "message": "Access denied"
  }
}
```

**解决方案**:
```javascript
async function handleForbidden() {
  alert('您没有权限执行此操作');
  // 可以重定向到无权限页面
  window.location.href = '/unauthorized';
}
```

---

### 通知系统错误码

#### NOTIFICATION_ERROR

**描述**: 通知操作失败

**HTTP 状态码**: 500

**常见原因**:
- 数据库错误
- Redis 连接失败
- 通知服务异常
- 参数验证失败

**解决方案**:
```javascript
async function handleNotificationError(error) {
  console.error('Notification error:', error);
  
  switch (error.code) {
    case 'NOTIFICATION_ERROR':
      alert('通知操作失败，请稍后重试');
      break;
    default:
      alert('发生未知错误');
  }
}
```

---

### 系统级错误码

#### VALIDATION_ERROR

**描述**: 输入验证错误

**HTTP 状态码**: 400

**常见原因**:
- 请求参数格式不正确
- 缺少必填字段
- 字段值超出范围
- 自定义验证规则失败

**错误响应示例**:
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid input data",
    "details": [
      {
        "field": "email",
        "message": "请提供有效的邮箱地址",
        "type": "string.email"
      }
    ]
  }
}
```

**解决方案**:
```javascript
function handleValidationError(error) {
  if (error.code !== 'VALIDATION_ERROR') return;

  const details = error.details || [];
  
  // 方式 1: 显示所有错误
  const messages = details.map(d => `${d.field}: ${d.message}`);
  alert(messages.join('\n'));

  // 方式 2: 将错误映射到表单字段
  details.forEach(detail => {
    const fieldElement = document.querySelector(`[name="${detail.field}"]`);
    if (fieldElement) {
      fieldElement.classList.add('error');
      fieldElement.nextElementSibling.textContent = detail.message;
    }
  });
}
```

---

#### TOO_MANY_REQUESTS

**描述**: 请求频率超限

**HTTP 状态码**: 429

**常见原因**:
- 短时间内请求过于频繁
- 触发限流策略
- IP 被临时封禁

**错误响应示例**:
```json
{
  "success": false,
  "error": {
    "code": "TOO_MANY_REQUESTS",
    "message": "Too many requests, please try again later.",
    "retryAfter": 60
  }
}
```

**解决方案**:
```javascript
async function handleRateLimit(error) {
  const retryAfter = error.retryAfter || 60;
  
  console.log(`Rate limited. Waiting ${retryAfter} seconds...`);
  
  // 等待后重试
  await new Promise(resolve => setTimeout(resolve, retryAfter * 1000));
  
  // 重试原请求
  return retryOriginalRequest();
}

// 全局请求拦截器
function createRateLimitedClient() {
  let isWaiting = false;
  let waitUntil = 0;

  return async function request(url, options) {
    if (isWaiting) {
      const now = Date.now();
      if (now < waitUntil) {
        await new Promise(r => setTimeout(r, waitUntil - now));
      }
      isWaiting = false;
    }

    try {
      return await fetch(url, options);
    } catch (error) {
      if (error.code === 'TOO_MANY_REQUESTS') {
        isWaiting = true;
        waitUntil = Date.now() + (error.retryAfter * 1000);
        throw error;
      }
      throw error;
    }
  };
}
```

---

#### NOT_FOUND

**描述**: 资源未找到

**HTTP 状态码**: 404

**常见原因**:
- 请求的资源不存在
- 资源已被删除
- URL 路径错误

**错误响应示例**:
```json
{
  "success": false,
  "error": {
    "code": "NOT_FOUND",
    "message": "User not found"
  }
}
```

**解决方案**:
```javascript
async function handleNotFound(resourceType) {
  switch (resourceType) {
    case 'User':
      alert('用户不存在');
      break;
    case 'Notification':
      alert('通知不存在或已被删除');
      break;
    default:
      alert('请求的资源不存在');
  }
}
```

---

#### BAD_REQUEST

**描述**: 错误的请求

**HTTP 状态码**: 400

**常见原因**:
- 请求体格式错误
- 不支持的 HTTP 方法
- 缺少必要的请求头

**解决方案**:
```javascript
async function validateAndSendRequest(url, data, method = 'POST') {
  if (!data || Object.keys(data).length === 0) {
    throw new Error('请求数据不能为空');
  }

  const response = await fetch(url, {
    method,
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(data)
  });

  if (response.status === 400) {
    const error = await response.json();
    throw new Error(error.error?.message || '无效的请求');
  }

  return response;
}
```

---

#### INTERNAL_ERROR

**描述**: 服务器内部错误

**HTTP 状态码**: 500

**常见原因**:
- 服务器代码异常
- 数据库连接失败
- 第三方服务不可用
- 内存不足

**解决方案**:
```javascript
async function handleInternalError() {
  // 记录错误详情
  console.error('Internal server error occurred');
  
  // 通知用户
  alert('服务器发生错误，请稍后重试。如果问题持续存在，请联系技术支持。');
  
  // 可选：自动重试
  setTimeout(() => {
    window.location.reload();
  }, 3000);
}
```

---

#### FORGOT_PASSWORD_ERROR

**描述**: 密码重置请求失败

**HTTP 状态码**: 500

**常见原因**:
- 邮箱服务配置错误
- 邮箱地址无效
- 邮件发送失败
- 数据库查询失败

**解决方案**:
```javascript
async function requestPasswordReset(email) {
  try {
    const response = await fetch('/api/v1/password/forgot', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email })
    });

    const result = await response.json();

    if (result.success) {
      alert('如果该邮箱已注册，重置链接已发送至您的邮箱');
    } else if (result.error.code === 'FORGOT_PASSWORD_ERROR') {
      alert('密码重置请求失败，请稍后重试');
    }

    return result;
  } catch (error) {
    alert('网络错误，请检查您的连接后重试');
  }
}
```

---

#### RESET_PASSWORD_ERROR

**描述**: 密码重置失败

**HTTP 状态码**: 400

**常见原因**:
- 重置令牌无效
- 重置令牌已过期
- 新密码不符合要求
- 令牌已被使用

**错误响应示例**:
```json
{
  "success": false,
  "error": {
    "code": "RESET_PASSWORD_ERROR",
    "message": "Invalid or expired reset token"
  }
}
```

**解决方案**:
```javascript
async function resetPassword(token, newPassword) {
  const response = await fetch('/api/v1/password/reset', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ token, newPassword })
  });

  if (response.status === 400) {
    const error = await response.json();
    
    if (error.error.message.includes('token')) {
      alert('重置链接无效或已过期，请重新请求密码重置');
    } else if (error.error.message.includes('Password')) {
      alert(error.error.message);
    }
    
    throw error;
  }

  return response.json();
}
```

---

## 常见错误场景与解决方案

### 场景 1: Token 过期处理

```javascript
class AuthInterceptor {
  constructor() {
    this.token = localStorage.getItem('token');
  }

  async request(url, options = {}) {
    if (this.token) {
      options.headers = {
        ...options.headers,
        'Authorization': `Bearer ${this.token}`
      };
    }

    const response = await fetch(url, options);

    if (response.status === 401) {
      // Token 过期，尝试刷新
      const refreshed = await this.tryRefreshToken();
      
      if (refreshed) {
        // 重试原请求
        options.headers['Authorization'] = `Bearer ${this.token}`;
        return fetch(url, options);
      }
      
      // 刷新失败，跳转登录
      window.location.href = '/login';
    }

    return response;
  }

  async tryRefreshToken() {
    try {
      const response = await fetch('/api/v1/auth/refresh', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: this.token })
      });

      if (response.ok) {
        const data = await response.json();
        this.token = data.data.token;
        localStorage.setItem('token', this.token);
        return true;
      }
    } catch (error) {
      console.error('Token refresh failed:', error);
    }
    
    return false;
  }
}
```

### 场景 2: 网络错误重试

```javascript
async function fetchWithRetry(url, options, maxRetries = 3) {
  let lastError;

  for (let i = 0; i < maxRetries; i++) {
    try {
      const response = await fetch(url, options);
      
      // 4xx 错误不需要重试
      if (response.status >= 400 && response.status < 500) {
        return response;
      }

      // 5xx 或网络错误可以重试
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }

      return response;
    } catch (error) {
      lastError = error;
      
      if (i < maxRetries - 1) {
        // 指数退避
        const delay = Math.pow(2, i) * 1000;
        console.log(`Retrying in ${delay}ms...`);
        await new Promise(r => setTimeout(r, delay));
      }
    }
  }

  throw lastError;
}
```

### 场景 3: 表单验证错误展示

```javascript
function displayValidationErrors(error) {
  if (error.code !== 'VALIDATION_ERROR') return;

  const details = error.details || [];
  const errorContainer = document.getElementById('form-errors');
  
  if (!errorContainer) return;

  errorContainer.innerHTML = details.map(detail => `
    <div class="error-item">
      <span class="field-name">${detail.field}:</span>
      <span class="error-message">${detail.message}</span>
    </div>
  `).join('');

  // 高亮错误字段
  details.forEach(detail => {
    const input = document.querySelector(`[name="${detail.field}"]`);
    if (input) {
      input.classList.add('error');
      input.addEventListener('input', () => {
        input.classList.remove('error');
      }, { once: true });
    }
  });
}
```

### 场景 4: 权限不足处理

```javascript
async function handlePermissionDenied() {
  // 检查用户角色
  const user = await getCurrentUser();
  
  if (!user) {
    alert('请先登录');
    window.location.href = '/login';
    return;
  }

  const requiredRoles = ['admin', 'moderator'];
  
  if (!requiredRoles.includes(user.role)) {
    alert('您的账户没有权限执行此操作');
    window.location.href = '/unauthorized';
    return;
  }
}
```

### 场景 5: 服务不可用处理

```javascript
async function handleServiceUnavailable() {
  // 检查服务健康状态
  const health = await checkHealth();
  
  if (health.status === 'degraded') {
    alert('服务目前降级运行，部分功能可能受影响');
  } else if (health.status === 'unhealthy') {
    alert('服务暂时不可用，我们正在紧急处理中。请稍后再试。');
  }
}

async function checkHealth() {
  try {
    const response = await fetch('/api/v1/health/detailed');
    if (response.ok) {
      const data = await response.json();
      return data.data;
    }
  } catch (error) {
    console.error('Health check failed:', error);
  }
  
  return { status: 'unknown' };
}
```

---

## 错误码速查表

| 错误码 | HTTP 状态码 | 描述 | 严重程度 |
|--------|-------------|------|----------|
| AUTH_ERROR | 401 | 认证错误 | 高 |
| UNAUTHORIZED | 401 | 未授权 | 高 |
| INVALID_CREDENTIALS | 401 | 无效凭据 | 高 |
| TOKEN_EXPIRED | 401 | Token 过期 | 中 |
| TOKEN_INVALID | 401 | Token 无效 | 高 |
| ACCOUNT_LOCKED | 401 | 账户锁定 | 高 |
| REGISTRATION_ERROR | 400 | 注册错误 | 中 |
| FETCH_USERS_ERROR | 500 | 获取用户列表错误 | 高 |
| FETCH_USER_ERROR | 500 | 获取用户错误 | 中 |
| CREATE_USER_ERROR | 400/500 | 创建用户错误 | 中 |
| UPDATE_USER_ERROR | 400/500 | 更新用户错误 | 中 |
| DELETE_USER_ERROR | 500 | 删除用户错误 | 中 |
| FORBIDDEN | 403 | 禁止访问 | 高 |
| NOTIFICATION_ERROR | 500 | 通知错误 | 低 |
| VALIDATION_ERROR | 400 | 验证错误 | 低 |
| TOO_MANY_REQUESTS | 429 | 请求超限 | 中 |
| NOT_FOUND | 404 | 资源未找到 | 低 |
| BAD_REQUEST | 400 | 错误请求 | 低 |
| INTERNAL_ERROR | 500 | 内部错误 | 高 |
| FORGOT_PASSWORD_ERROR | 500 | 密码重置请求错误 | 中 |
| RESET_PASSWORD_ERROR | 400 | 密码重置错误 | 中 |

---

## 获取帮助

如果遇到本文档未涵盖的错误，请联系技术支持：

- 邮箱: support@hjtpx.com
- 文档版本: 1.0.0
- 最后更新: 2026-05-15
