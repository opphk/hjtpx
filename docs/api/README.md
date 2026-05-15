# HJTPX API 参考文档

本文档是 HJTPX 应用程序的完整中文 API 参考文档，提供了所有 API 端点的详细说明、请求参数、响应格式以及使用示例。

## 目录

- [概述](#概述)
- [认证 API](#认证-api)
- [用户管理 API](#用户管理-api)
- [通知 API](#通知-api)
- [健康检查 API](#健康检查-api)
- [响应格式](#响应格式)
- [错误处理](#错误处理)

---

## 概述

### 基本信息

| 项目 | 说明 |
|------|------|
| API 版本 | v1 |
| 基础 URL | `http://localhost:3000/api/v1` |
| 数据格式 | JSON |
| 字符编码 | UTF-8 |
| 认证方式 | JWT Bearer Token |

### 通用请求头

所有需要认证的请求都应包含以下请求头：

```http
Content-Type: application/json
Accept: application/json
Authorization: Bearer <your_jwt_token>
```

### 通用响应头

```http
Content-Type: application/json
X-Request-ID: <unique_request_id>
X-Response-Time: <response_time_ms>
```

---

## 认证 API

### 1. 用户登录

登录系统并获取访问令牌。

**端点**: `POST /api/v1/auth/login`

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| email | string | 是 | 用户邮箱地址，必须是有效的邮箱格式 |
| password | string | 是 | 用户密码（至少 8 个字符） |

**请求示例**:

```json
{
  "email": "user@example.com",
  "password": "SecurePass123"
}
```

**成功响应** (HTTP 200):

```json
{
  "success": true,
  "data": {
    "user": {
      "id": 1,
      "email": "user@example.com",
      "name": "张三",
      "role": "user"
    },
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expiresIn": "7d"
  },
  "message": "Login successful",
  "timestamp": "2026-05-15T10:30:00.000Z"
}
```

**错误响应** (HTTP 401):

```json
{
  "success": false,
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Invalid email or password"
  },
  "timestamp": "2026-05-15T10:30:00.000Z"
}
```

**curl 示例**:

```bash
curl -X POST http://localhost:3000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "SecurePass123"}'
```

**JavaScript 示例**:

```javascript
const response = await fetch('/api/v1/auth/login', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    email: 'user@example.com',
    password: 'SecurePass123'
  })
});

const data = await response.json();
if (data.success) {
  localStorage.setItem('token', data.data.token);
}
```

---

### 2. 用户注册

创建新用户账号。

**端点**: `POST /api/v1/auth/register`

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| email | string | 是 | 用户邮箱（必须是有效邮箱格式） |
| name | string | 是 | 用户名称（2-100 个字符） |
| password | string | 是 | 用户密码（至少 8 个字符，包含大小写字母和数字） |

**请求示例**:

```json
{
  "email": "newuser@example.com",
  "name": "新用户",
  "password": "SecurePass123"
}
```

**成功响应** (HTTP 201):

```json
{
  "success": true,
  "data": {
    "user": {
      "id": 2,
      "email": "newuser@example.com",
      "name": "新用户",
      "role": "user"
    },
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expiresIn": "7d"
  },
  "message": "Registration successful",
  "timestamp": "2026-05-15T10:30:00.000Z"
}
```

**curl 示例**:

```bash
curl -X POST http://localhost:3000/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "newuser@example.com", "name": "新用户", "password": "SecurePass123"}'
```

---

### 3. 验证 Token

验证 JWT Token 的有效性。

**端点**: `POST /api/v1/auth/verify`

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| token | string | 是 | 需要验证的 JWT Token |

**请求示例**:

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**成功响应** (HTTP 200):

```json
{
  "success": true,
  "data": {
    "valid": true,
    "user": {
      "id": 1,
      "email": "user@example.com",
      "role": "user"
    }
  },
  "message": "Token is valid",
  "timestamp": "2026-05-15T10:30:00.000Z"
}
```

**curl 示例**:

```bash
curl -X POST http://localhost:3000/api/v1/auth/verify \
  -H "Content-Type: application/json" \
  -d '{"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."}'
```

---

### 4. 刷新 Token

获取新的 JWT Token。

**端点**: `POST /api/v1/auth/refresh`

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| token | string | 是 | 当前有效的 JWT Token |

**成功响应** (HTTP 200):

```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expiresIn": "7d"
  },
  "message": "Token refreshed successfully",
  "timestamp": "2026-05-15T10:30:00.000Z"
}
```

**curl 示例**:

```bash
curl -X POST http://localhost:3000/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."}'
```

---

### 5. 用户登出

清除当前用户会话。

**端点**: `POST /api/v1/auth/logout`

**认证**: 需要

**成功响应** (HTTP 200):

```json
{
  "success": true,
  "data": {
    "message": "Logout successful"
  },
  "message": "Logout successful",
  "timestamp": "2026-05-15T10:30:00.000Z"
}
```

**curl 示例**:

```bash
curl -X POST http://localhost:3000/api/v1/auth/logout \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

---

### 6. 请求密码重置

发送密码重置邮件到用户邮箱。

**端点**: `POST /api/v1/password/forgot`

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| email | string | 是 | 用户邮箱地址 |

**请求示例**:

```json
{
  "email": "user@example.com"
}
```

**成功响应** (HTTP 200):

```json
{
  "success": true,
  "data": {
    "message": "If email exists, reset link will be sent"
  },
  "message": "Password reset email sent if account exists",
  "timestamp": "2026-05-15T10:30:00.000Z"
}
```

**curl 示例**:

```bash
curl -X POST http://localhost:3000/api/v1/password/forgot \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com"}'
```

---

### 7. 重置密码

使用重置令牌设置新密码。

**端点**: `POST /api/v1/password/reset`

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| token | string | 是 | 重置令牌（从邮件链接获取） |
| newPassword | string | 是 | 新密码（至少 8 个字符，包含大小写字母、数字和特殊字符） |

**请求示例**:

```json
{
  "token": "abc123def456...",
  "newPassword": "NewSecurePass123!"
}
```

**成功响应** (HTTP 200):

```json
{
  "success": true,
  "data": {
    "message": "Password successfully reset"
  },
  "message": "Password successfully reset",
  "timestamp": "2026-05-15T10:30:00.000Z"
}
```

**错误响应** (HTTP 400):

```json
{
  "success": false,
  "error": {
    "code": "BAD_REQUEST",
    "message": "Invalid or expired reset token"
  },
  "timestamp": "2026-05-15T10:30:00.000Z"
}
```

**curl 示例**:

```bash
curl -X POST http://localhost:3000/api/v1/password/reset \
  -H "Content-Type: application/json" \
  -d '{"token": "abc123def456...", "newPassword": "NewSecurePass123!"}'
```

---

## 用户管理 API

### 1. 获取当前用户

获取已认证用户的详细信息。

**端点**: `GET /api/v1/users/me`

**认证**: 需要

**成功响应** (HTTP 200):

```json
{
  "success": true,
  "data": {
    "id": 1,
    "email": "user@example.com",
    "name": "张三",
    "role": "user",
    "created_at": "2026-05-10T08:00:00.000Z"
  },
  "message": "User retrieved successfully",
  "timestamp": "2026-05-15T10:30:00.000Z"
}
```

**curl 示例**:

```bash
curl -X GET http://localhost:3000/api/v1/users/me \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**JavaScript 示例**:

```javascript
const response = await fetch('/api/v1/users/me', {
  headers: {
    'Authorization': `Bearer ${localStorage.getItem('token')}`
  }
});

const data = await response.json();
console.log(data.data);
```

---

### 2. 获取用户列表

获取所有用户列表（仅管理员可访问）。

**端点**: `GET /api/v1/users`

**认证**: 需要（管理员角色）

**查询参数**:

| 参数名 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| page | integer | 1 | 页码 |
| limit | integer | 20 | 每页数量 |

**成功响应** (HTTP 200):

```json
{
  "success": true,
  "data": [
    {
      "id": 1,
      "email": "user@example.com",
      "name": "张三",
      "role": "user",
      "created_at": "2026-05-10T08:00:00.000Z"
    },
    {
      "id": 2,
      "email": "admin@example.com",
      "name": "管理员",
      "role": "admin",
      "created_at": "2026-05-01T08:00:00.000Z"
    }
  ],
  "message": "Users retrieved successfully",
  "timestamp": "2026-05-15T10:30:00.000Z"
}
```

**curl 示例**:

```bash
curl -X GET "http://localhost:3000/api/v1/users?page=1&limit=20" \
  -H "Authorization: Bearer ADMIN_TOKEN_HERE"
```

---

### 3. 获取指定用户

根据 ID 获取用户信息。

**端点**: `GET /api/v1/users/:id`

**认证**: 需要

**权限说明**:
- 管理员可访问任意用户
- 普通用户仅可访问本人

**路径参数**:

| 参数名 | 类型 | 说明 |
|--------|------|------|
| id | integer | 用户 ID |

**成功响应** (HTTP 200):

```json
{
  "success": true,
  "data": {
    "id": 1,
    "email": "user@example.com",
    "name": "张三",
    "role": "user",
    "created_at": "2026-05-10T08:00:00.000Z"
  },
  "message": "User retrieved successfully",
  "timestamp": "2026-05-15T10:30:00.000Z"
}
```

**curl 示例**:

```bash
curl -X GET http://localhost:3000/api/v1/users/1 \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

---

### 4. 创建用户

创建新用户账号（仅管理员可访问）。

**端点**: `POST /api/v1/users`

**认证**: 需要（管理员角色）

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| email | string | 是 | 用户邮箱 |
| name | string | 是 | 用户名称（2-100 个字符） |
| password | string | 是 | 用户密码（至少 8 个字符） |

**请求示例**:

```json
{
  "email": "newuser@example.com",
  "name": "新用户",
  "password": "SecurePass123"
}
```

**成功响应** (HTTP 201):

```json
{
  "success": true,
  "data": {
    "id": 3,
    "email": "newuser@example.com",
    "name": "新用户",
    "role": "user",
    "created_at": "2026-05-15T10:30:00.000Z"
  },
  "message": "User created successfully",
  "timestamp": "2026-05-15T10:30:00.000Z"
}
```

**curl 示例**:

```bash
curl -X POST http://localhost:3000/api/v1/users \
  -H "Authorization: Bearer ADMIN_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{"email": "newuser@example.com", "name": "新用户", "password": "SecurePass123"}'
```

---

### 5. 更新当前用户

更新已认证用户的个人信息。

**端点**: `PUT /api/v1/users/me`

**认证**: 需要

**请求参数**（至少提供一项）:

| 参数名 | 类型 | 说明 |
|--------|------|------|
| email | string | 用户邮箱（必须是有效邮箱格式） |
| name | string | 用户名称（2-100 个字符） |
| password | string | 用户密码（至少 8 个字符） |

**请求示例**:

```json
{
  "name": "更新后的名称",
  "email": "updated@example.com"
}
```

**成功响应** (HTTP 200):

```json
{
  "success": true,
  "data": {
    "id": 1,
    "email": "updated@example.com",
    "name": "更新后的名称",
    "role": "user",
    "created_at": "2026-05-10T08:00:00.000Z"
  },
  "message": "User updated successfully",
  "timestamp": "2026-05-15T10:30:00.000Z"
}
```

**curl 示例**:

```bash
curl -X PUT http://localhost:3000/api/v1/users/me \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{"name": "更新后的名称"}'
```

---

### 6. 更新指定用户

更新指定用户信息。

**端点**: `PUT /api/v1/users/:id`

**认证**: 需要

**权限说明**:
- 管理员可更新任意用户的所有字段
- 普通用户仅可更新本人的非管理员字段

**路径参数**:

| 参数名 | 类型 | 说明 |
|--------|------|------|
| id | integer | 用户 ID |

**请求参数**:

| 参数名 | 类型 | 说明 |
|--------|------|------|
| email | string | 用户邮箱 |
| name | string | 用户名称 |
| password | string | 用户密码 |
| role | string | 用户角色（仅管理员可修改） |

**请求示例**:

```json
{
  "name": "管理员更新后的名称",
  "role": "moderator"
}
```

**成功响应** (HTTP 200):

```json
{
  "success": true,
  "data": {
    "id": 2,
    "email": "admin@example.com",
    "name": "管理员更新后的名称",
    "role": "moderator",
    "created_at": "2026-05-01T08:00:00.000Z"
  },
  "message": "User updated successfully",
  "timestamp": "2026-05-15T10:30:00.000Z"
}
```

**curl 示例**:

```bash
curl -X PUT http://localhost:3000/api/v1/users/2 \
  -H "Authorization: Bearer ADMIN_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{"role": "moderator"}'
```

---

### 7. 删除用户

删除指定用户（仅管理员可访问）。

**端点**: `DELETE /api/v1/users/:id`

**认证**: 需要（管理员角色）

**路径参数**:

| 参数名 | 类型 | 说明 |
|--------|------|------|
| id | integer | 用户 ID |

**成功响应** (HTTP 204):

无内容

**错误响应** (HTTP 404):

```json
{
  "success": false,
  "error": {
    "code": "NOT_FOUND",
    "message": "User not found"
  },
  "timestamp": "2026-05-15T10:30:00.000Z"
}
```

**curl 示例**:

```bash
curl -X DELETE http://localhost:3000/api/v1/users/3 \
  -H "Authorization: Bearer ADMIN_TOKEN_HERE"
```

---

## 通知 API

### 1. 获取通知列表

获取当前用户的通知列表。

**端点**: `GET /api/v1/notifications`

**认证**: 需要

**查询参数**:

| 参数名 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| page | integer | 1 | 页码 |
| limit | integer | 20 | 每页数量 |
| status | string | - | 通知状态筛选（unread/read/archived） |

**成功响应** (HTTP 200):

```json
{
  "success": true,
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "userId": "123e4567-e89b-12d3-a456-426614174000",
      "title": "系统通知",
      "message": "您的账户信息已更新",
      "type": "system",
      "status": "unread",
      "channels": ["in_app", "email"],
      "createdAt": "2026-05-15T09:00:00.000Z"
    }
  ],
  "timestamp": "2026-05-15T10:30:00.000Z"
}
```

**curl 示例**:

```bash
# 获取所有通知
curl -X GET http://localhost:3000/api/v1/notifications \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"

# 获取未读通知
curl -X GET "http://localhost:3000/api/v1/notifications?status=unread" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"

# 分页获取
curl -X GET "http://localhost:3000/api/v1/notifications?page=2&limit=10" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

---

### 2. 获取未读通知数量

获取当前用户的未读通知数量。

**端点**: `GET /api/v1/notifications/unread/count`

**认证**: 需要

**成功响应** (HTTP 200):

```json
{
  "success": true,
  "data": {
    "count": 5
  },
  "timestamp": "2026-05-15T10:30:00.000Z"
}
```

**curl 示例**:

```bash
curl -X GET http://localhost:3000/api/v1/notifications/unread/count \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

---

### 3. 获取通知详情

根据 ID 获取指定通知的详细信息。

**端点**: `GET /api/v1/notifications/:id`

**认证**: 需要

**路径参数**:

| 参数名 | 类型 | 说明 |
|--------|------|------|
| id | string | 通知 ID（UUID 格式） |

**成功响应** (HTTP 200):

```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "userId": "123e4567-e89b-12d3-a456-426614174000",
    "title": "系统通知",
    "message": "您的账户信息已更新",
    "type": "system",
    "status": "unread",
    "channels": ["in_app", "email"],
    "createdAt": "2026-05-15T09:00:00.000Z"
  },
  "timestamp": "2026-05-15T10:30:00.000Z"
}
```

**curl 示例**:

```bash
curl -X GET http://localhost:3000/api/v1/notifications/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

---

### 4. 标记通知为已读

将指定通知标记为已读状态。

**端点**: `PUT /api/v1/notifications/:id/read`

**认证**: 需要

**路径参数**:

| 参数名 | 类型 | 说明 |
|--------|------|------|
| id | string | 通知 ID |

**成功响应** (HTTP 200):

```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "read"
  },
  "timestamp": "2026-05-15T10:30:00.000Z"
}
```

**curl 示例**:

```bash
curl -X PUT http://localhost:3000/api/v1/notifications/550e8400-e29b-41d4-a716-446655440000/read \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

---

### 5. 标记所有通知为已读

将当前用户的所有通知标记为已读状态。

**端点**: `PUT /api/v1/notifications/read-all`

**认证**: 需要

**成功响应** (HTTP 200):

```json
{
  "success": true,
  "data": {
    "message": "All notifications marked as read"
  },
  "timestamp": "2026-05-15T10:30:00.000Z"
}
```

**curl 示例**:

```bash
curl -X PUT http://localhost:3000/api/v1/notifications/read-all \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

---

### 6. 删除通知

删除指定通知。

**端点**: `DELETE /api/v1/notifications/:id`

**认证**: 需要

**路径参数**:

| 参数名 | 类型 | 说明 |
|--------|------|------|
| id | string | 通知 ID |

**成功响应** (HTTP 200):

```json
{
  "success": true,
  "data": {
    "message": "Notification deleted"
  },
  "timestamp": "2026-05-15T10:30:00.000Z"
}
```

**curl 示例**:

```bash
curl -X DELETE http://localhost:3000/api/v1/notifications/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

---

### 7. 发送通知

向指定用户发送通知。

**端点**: `POST /api/v1/notifications/send`

**认证**: 需要

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| userId | string | 是 | 目标用户 ID |
| title | string | 是 | 通知标题 |
| message | string | 是 | 通知内容 |
| type | string | 否 | 通知类型（system/user/security/promotion），默认 system |
| channels | array | 否 | 通知渠道列表（email/sms/push/in_app），默认 ["in_app"] |

**请求示例**:

```json
{
  "userId": "123e4567-e89b-12d3-a456-426614174000",
  "title": "账户安全提醒",
  "message": "检测到您的账户在新设备登录",
  "type": "security",
  "channels": ["in_app", "email", "sms"]
}
```

**成功响应** (HTTP 201):

```json
{
  "success": true,
  "data": {
    "id": "660e8400-e29b-41d4-a716-446655440001",
    "userId": "123e4567-e89b-12d3-a456-426614174000",
    "title": "账户安全提醒",
    "message": "检测到您的账户在新设备登录",
    "type": "security",
    "status": "unread",
    "channels": ["in_app", "email", "sms"],
    "createdAt": "2026-05-15T10:30:00.000Z"
  },
  "timestamp": "2026-05-15T10:30:00.000Z"
}
```

**curl 示例**:

```bash
curl -X POST http://localhost:3000/api/v1/notifications/send \
  -H "Authorization: Bearer ADMIN_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "userId": "123e4567-e89b-12d3-a456-426614174000",
    "title": "账户安全提醒",
    "message": "检测到您的账户在新设备登录",
    "type": "security",
    "channels": ["in_app", "email"]
  }'
```

---

## 健康检查 API

### 1. 基础健康检查

验证服务是否正常运行。

**端点**: `GET /api/v1/health`

**认证**: 不需要

**成功响应** (HTTP 200):

```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "service": "HJTPX API",
    "version": "1.0.0",
    "timestamp": "2026-05-15T10:30:00.000Z",
    "uptime": 3600.5,
    "environment": "development"
  }
}
```

**健康状态说明**:

| 状态 | HTTP 状态码 | 说明 |
|------|-------------|------|
| healthy | 200 | 所有服务正常运行 |
| degraded | 200 | 部分服务降级，但核心功能可用 |
| unhealthy | 503 | 关键服务不可用 |

**curl 示例**:

```bash
curl -X GET http://localhost:3000/api/v1/health
```

---

### 2. 详细健康检查

返回所有依赖服务的详细状态。

**端点**: `GET /api/v1/health/detailed`

**认证**: 不需要

**成功响应** (HTTP 200):

```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "timestamp": "2026-05-15T10:30:00.000Z",
    "service": "HJTPX API",
    "version": "1.0.0",
    "uptime": 3600.5,
    "environment": "development",
    "checks": {
      "database": {
        "status": "healthy",
        "message": "Database connection is healthy",
        "responseTime": "15ms"
      },
      "redis": {
        "status": "healthy",
        "message": "Redis connection is healthy",
        "responseTime": "5ms"
      },
      "cache": {
        "status": "healthy",
        "message": "Cache service is healthy",
        "stats": {
          "hits": 1000,
          "misses": 50
        }
      },
      "memory": {
        "status": "healthy",
        "message": "Memory usage is normal",
        "usage": {
          "used": 45,
          "total": 128,
          "unit": "MB"
        }
      },
      "cpu": {
        "status": "healthy",
        "message": "CPU usage is normal",
        "loadAverage": [1.2, 0.8, 0.5]
      }
    },
    "responseTime": "25ms"
  }
}
```

**curl 示例**:

```bash
curl -X GET http://localhost:3000/api/v1/health/detailed
```

---

## 响应格式

### 成功响应格式

```json
{
  "success": true,
  "data": {},
  "message": "操作成功",
  "timestamp": "2026-05-15T10:30:00.000Z"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| success | boolean | 请求是否成功，始终为 true |
| data | object/array | 响应数据 |
| message | string | 操作描述信息 |
| timestamp | string | 服务器响应时间 |

### 错误响应格式

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "错误描述",
    "details": []
  },
  "timestamp": "2026-05-15T10:30:00.000Z"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| success | boolean | 请求是否成功，始终为 false |
| error | object | 错误详情 |
| error.code | string | 错误码 |
| error.message | string | 错误描述 |
| error.details | array | 详细错误信息（验证错误时） |
| timestamp | string | 服务器响应时间 |

---

## 错误处理

### HTTP 状态码

| 状态码 | 说明 |
|--------|------|
| 200 | 请求成功 |
| 201 | 资源创建成功 |
| 204 | 请求成功，无返回内容 |
| 400 | 请求参数错误 |
| 401 | 未认证或认证失败 |
| 403 | 无访问权限 |
| 404 | 资源不存在 |
| 429 | 请求过于频繁 |
| 500 | 服务器内部错误 |
| 503 | 服务不可用 |

### 常见错误码

| 错误码 | 说明 | 解决方案 |
|--------|------|----------|
| UNAUTHORIZED | 认证失败 | 重新登录获取 Token |
| FORBIDDEN | 权限不足 | 检查用户角色权限 |
| VALIDATION_ERROR | 输入验证失败 | 检查请求参数格式 |
| TOO_MANY_REQUESTS | 请求频率超限 | 等待后重试 |
| NOT_FOUND | 资源不存在 | 检查请求的资源 ID |
| INTERNAL_ERROR | 服务器错误 | 联系技术支持 |

### 错误处理示例

**JavaScript**:

```javascript
async function handleApiRequest(apiFunction) {
  try {
    const response = await apiFunction();
    
    if (!response.ok) {
      const error = await response.json();
      
      switch (error.error?.code) {
        case 'UNAUTHORIZED':
          // Token 过期，重新登录
          localStorage.removeItem('token');
          window.location.href = '/login';
          break;
          
        case 'FORBIDDEN':
          alert('您没有权限执行此操作');
          break;
          
        case 'VALIDATION_ERROR':
          const details = error.error?.details || [];
          alert(details.map(d => d.message).join('\n'));
          break;
          
        case 'TOO_MANY_REQUESTS':
          alert('请求过于频繁，请稍后再试');
          break;
          
        default:
          alert(error.error?.message || '发生未知错误');
      }
      
      return null;
    }
    
    return await response.json();
  } catch (error) {
    console.error('网络错误:', error);
    alert('网络连接失败，请检查网络后重试');
    return null;
  }
}
```

---

## 其他资源

- [API 使用示例](../API_EXAMPLES.md)
- [错误码对照表](../API_ERROR_CODES.md)
- [最佳实践指南](../API_BEST_PRACTICES.md)

---

**文档版本**: 1.0.0  
**最后更新**: 2026-05-15  
**维护团队**: HJTPX Development Team
