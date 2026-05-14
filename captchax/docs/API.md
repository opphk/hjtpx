# CaptchaX API 接口文档

## 基础信息

- **基础URL**: `http://localhost:8080`
- **API版本**: v1
- **数据格式**: JSON
- **字符编码**: UTF-8

## 通用响应格式

### 成功响应

```json
{
  "code": 200,
  "message": "success",
  "data": { ... }
}
```

### 错误响应

```json
{
  "code": 400,
  "message": "error description",
  "data": null
}
```

### 错误码说明

| 错误码 | 说明 |
|--------|------|
| 200 | 成功 |
| 400 | 请求参数错误 |
| 401 | 未授权/认证失败 |
| 404 | 资源不存在 |
| 429 | 请求过于频繁（限流） |
| 500 | 服务器内部错误 |

---

## 验证码 API

### 1. 生成滑块验证码

生成滑块缺口验证码，用户需要拖动滑块到正确位置完成验证。

**请求**

```http
POST /api/v1/captcha/slider
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| app_id | string | 是 | 应用标识 |
| width | int | 否 | 图片宽度（默认200） |
| height | int | 否 | 图片高度（默认80） |
| client_info | string | 否 | 客户端信息（用于风控） |

**请求示例**

```json
{
  "app_id": "my-app-001",
  "width": 200,
  "height": 80,
  "client_info": "Mozilla/5.0..."
}
```

**响应参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 验证码ID，用于后续验证 |
| background_b64 | string | 背景图（Base64编码） |
| slider_b64 | string | 滑块图（Base64编码） |
| target_x | int | 目标位置X坐标 |
| target_y | int | 目标位置Y坐标 |

**响应示例**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "cap_abc123def456",
    "background_b64": "data:image/png;base64,iVBORw0KGgo...",
    "slider_b64": "data:image/png;base64,iVBORw0KGgo...",
    "target_x": 156,
    "target_y": 25
  }
}
```

---

### 2. 验证滑块验证码

验证用户拖动的滑块位置是否正确。

**请求**

```http
POST /api/v1/captcha/slider/verify
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| captcha_id | string | 是 | 验证码ID |
| target_x | int | 是 | 用户拖动的X坐标 |
| target_y | int | 否 | 用户拖动的Y坐标 |

**请求示例**

```json
{
  "captcha_id": "cap_abc123def456",
  "target_x": 154,
  "target_y": 25
}
```

**响应参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| success | bool | 验证是否成功 |
| message | string | 验证结果描述 |

**响应示例**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "success": true,
    "message": "验证通过"
  }
}
```

---

### 3. 生成点选验证码

生成图片点选验证码，用户需要按正确顺序点击指定字符。

**请求**

```http
POST /api/v1/captcha/click
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| app_id | string | 是 | 应用标识 |
| char_count | int | 否 | 需要点击的字符数量（默认4） |
| client_info | string | 否 | 客户端信息（用于风控） |

**请求示例**

```json
{
  "app_id": "my-app-001",
  "char_count": 4,
  "client_info": "Mozilla/5.0..."
}
```

**响应参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 验证码ID |
| image | string | 验证码图片（Base64） |
| target_chars | string[] | 需要点击的字符列表 |
| char_positions | object[] | 字符位置信息 |

**响应示例**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "cap_xyz789ghi012",
    "image": "data:image/png;base64,iVBORw0KGgo...",
    "target_chars": ["中", "国", "天", "安"],
    "char_positions": [
      {"char": "中", "x": 45, "y": 30, "width": 20, "height": 25},
      {"char": "国", "x": 120, "y": 25, "width": 20, "height": 25},
      {"char": "天", "x": 80, "y": 55, "width": 20, "height": 25},
      {"char": "安", "x": 160, "y": 50, "width": 20, "height": 25}
    ]
  }
}
```

---

### 4. 验证点选验证码

验证用户点击的位置是否正确。

**请求**

```http
POST /api/v1/captcha/click/verify
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| captcha_id | string | 是 | 验证码ID |
| clicks | object[] | 是 | 用户点击位置列表 |

**clicks 参数结构**

| 参数 | 类型 | 说明 |
|------|------|------|
| char | string | 点击的字符 |
| x | int | 点击的X坐标 |
| y | int | 点击的Y坐标 |

**请求示例**

```json
{
  "captcha_id": "cap_xyz789ghi012",
  "clicks": [
    {"char": "中", "x": 45, "y": 32},
    {"char": "国", "x": 122, "y": 28},
    {"char": "天", "x": 82, "y": 57},
    {"char": "安", "x": 162, "y": 52}
  ]
}
```

**响应参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| success | bool | 验证是否成功 |
| score | float | 匹配分数（0-1） |
| message | string | 验证结果描述 |

**响应示例**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "success": true,
    "score": 0.95,
    "message": "验证通过"
  }
}
```

---

### 5. 生成拼图验证码

生成拼图缺口验证码，类似滑块但使用拼图块。

**请求**

```http
POST /api/v1/captcha/puzzle
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| app_id | string | 是 | 应用标识 |
| width | int | 否 | 图片宽度（默认200） |
| height | int | 否 | 图片高度（默认80） |
| client_info | string | 否 | 客户端信息（用于风控） |

**响应参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 验证码ID |
| background_b64 | string | 背景图（Base64） |
| puzzle_b64 | string | 拼图块（Base64） |
| target_x | int | 目标位置X坐标 |
| target_y | int | 目标位置Y坐标 |

**响应示例**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "cap_puz456jkl789",
    "background_b64": "data:image/png;base64,iVBORw0KGgo...",
    "puzzle_b64": "data:image/png;base64,iVBORw0KGgo...",
    "target_x": 150,
    "target_y": 20
  }
}
```

---

### 6. 验证拼图验证码

**请求**

```http
POST /api/v1/captcha/puzzle/verify
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| captcha_id | string | 是 | 验证码ID |
| target_x | int | 是 | 用户放置的X坐标 |
| target_y | int | 否 | 用户放置的Y坐标 |

**响应参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| success | bool | 验证是否成功 |
| message | string | 验证结果描述 |

---

### 7. 健康检查

**请求**

```http
GET /health
```

**响应示例**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "status": "healthy",
    "service": "captchax-api"
  }
}
```

---

## 管理 API

管理 API 需要在请求头中携带 JWT Token：

```http
Authorization: Bearer <jwt_token>
```

### 1. 管理员登录

**请求**

```http
POST /admin/api/login
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 用户名 |
| password | string | 是 | 密码 |

**请求示例**

```json
{
  "username": "admin",
  "password": "admin123"
}
```

**响应参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| token | string | JWT Token |
| expires_at | string | 过期时间 |
| username | string | 用户名 |
| role | string | 角色 |

**响应示例**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_at": "2026-05-15T12:00:00Z",
    "username": "admin",
    "role": "admin"
  }
}
```

---

### 2. 管理员登出

**请求**

```http
POST /admin/api/logout
Authorization: Bearer <jwt_token>
```

**响应示例**

```json
{
  "code": 200,
  "message": "logged out successfully",
  "data": null
}
```

---

### 3. 获取仪表盘数据

**请求**

```http
GET /admin/api/dashboard
Authorization: Bearer <jwt_token>
```

**响应参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| captcha_stats | object | 验证码统计 |
| whitelist_count | int | 白名单数量 |
| blacklist_count | int | 黑名单数量 |
| admin_count | int | 管理员数量 |
| system_config | object | 系统配置 |
| recent_logs | array | 最近验证日志 |
| admin_id | int | 当前管理员ID |
| username | string | 当前用户名 |
| role | string | 当前角色 |

**响应示例**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "captcha_stats": {
      "total": 1000,
      "success": 950,
      "failed": 50,
      "success_rate": 95.0
    },
    "whitelist_count": 10,
    "blacklist_count": 25,
    "admin_count": 3,
    "system_config": { ... },
    "recent_logs": [ ... ],
    "admin_id": 1,
    "username": "admin",
    "role": "admin"
  }
}
```

---

### 4. 获取统计数据

**请求**

```http
GET /admin/api/stats?period=7d
Authorization: Bearer <jwt_token>
```

**查询参数**

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| period | string | 7d | 时间范围：24h, 7d, 30d, 90d |

**响应示例**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "period": "7d",
    "start_date": "2026-05-07T00:00:00Z",
    "end_date": "2026-05-14T00:00:00Z",
    "captcha_stats": {
      "total": 5000,
      "success": 4800,
      "failed": 200,
      "success_rate": 96.0
    },
    "whitelist_count": 10,
    "blacklist_count": 25
  }
}
```

---

### 5. 获取配置列表

**请求**

```http
GET /admin/api/config
Authorization: Bearer <jwt_token>
```

**响应示例**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "configs": [
      {
        "id": 1,
        "key": "max_attempts_per_ip",
        "value": "10",
        "description": "Maximum verification attempts per IP per hour",
        "updated_at": "2026-05-14T10:00:00Z"
      }
    ]
  }
}
```

---

### 6. 更新配置

**请求**

```http
POST /admin/api/config
Authorization: Bearer <jwt_token>
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| key | string | 是 | 配置键名 |
| value | string | 是 | 配置值 |

**请求示例**

```json
{
  "key": "max_attempts_per_ip",
  "value": "20"
}
```

---

### 7. 获取白名单

**请求**

```http
GET /admin/api/whitelist?page=1&page_size=20
Authorization: Bearer <jwt_token>
```

**查询参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码（默认1） |
| page_size | int | 每页数量（默认20） |
| ip | string | 按IP筛选 |
| domain | string | 按域名筛选 |

**响应示例**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 1,
        "ip": "192.168.1.100",
        "domain": "example.com",
        "reason": "可信内部系统",
        "created_at": "2026-05-14T10:00:00Z"
      }
    ],
    "total": 10,
    "page": 1,
    "page_size": 20,
    "total_pages": 1
  }
}
```

---

### 8. 添加白名单

**请求**

```http
POST /admin/api/whitelist
Authorization: Bearer <jwt_token>
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| ip | string | 是 | IP地址 |
| domain | string | 否 | 关联域名 |
| reason | string | 否 | 添加原因 |

---

### 9. 删除白名单

**请求**

```http
DELETE /admin/api/whitelist/{id}
Authorization: Bearer <jwt_token>
```

---

### 10. 获取黑名单

**请求**

```http
GET /admin/api/blacklist?page=1&page_size=20
Authorization: Bearer <jwt_token>
```

**查询参数**

| 参数 | 类型 | 说明 |
|------|------|------|------|
| page | int | 页码 |
| page_size | int | 每页数量 |
| ip | string | 按IP筛选 |
| active_only | bool | 仅显示生效中 |

---

### 11. 添加黑名单

**请求**

```http
POST /admin/api/blacklist
Authorization: Bearer <jwt_token>
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| ip | string | 是 | IP地址 |
| reason | string | 否 | 封禁原因 |
| expire_at | string | 否 | 过期时间（ISO8601格式） |

---

### 12. 删除黑名单

**请求**

```http
DELETE /admin/api/blacklist/{id}
Authorization: Bearer <jwt_token>
```

---

## 错误码详细说明

| 错误码 | HTTP状态 | 说明 | 解决方案 |
|--------|----------|------|----------|
| 1001 | 400 | 参数缺失 | 检查必填参数 |
| 1002 | 400 | 参数格式错误 | 检查参数类型和格式 |
| 2001 | 404 | 验证码不存在 | 重新生成验证码 |
| 2002 | 400 | 验证码已过期 | 重新生成验证码 |
| 2003 | 400 | 验证次数超限 | 等待后重试 |
| 3001 | 401 | 认证失败 | 检查用户名密码 |
| 3002 | 401 | Token无效 | 重新登录 |
| 3003 | 403 | 无权限 | 联系管理员 |
| 4001 | 429 | 请求过于频繁 | 降低请求频率 |
| 5001 | 500 | 服务器内部错误 | 联系技术支持 |
