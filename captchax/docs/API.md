# CaptchaX API 接口文档

## 基础信息

- **基础URL**: `http://localhost:8080`
- **API版本**: v1, v2
- **数据格式**: JSON
- **字符编码**: UTF-8

## 版本说明

### v1 vs v2 差异

| 功能 | v1 | v2 |
|------|----|----|
| 基础验证码生成/验证 | ✅ | ✅ |
| 批量验证 | ❌ | ✅ |
| 自定义验证场景 | ❌ | ✅ |
| Webhook 回调 | ❌ | ✅ |
| 响应缓存策略 | ❌ | ✅ |

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

| 错误码 | HTTP状态 | 说明 |
|--------|----------|------|
| 0 | 200 | 成功 |
| 400 | 400 | 请求参数错误 |
| 401 | 401 | 未授权/认证失败 |
| 404 | 404 | 资源不存在 |
| 409 | 409 | 重复请求 |
| 429 | 429 | 请求过于频繁（限流） |
| 500 | 500 | 服务器内部错误 |

---

## 通用请求头

| 请求头 | 必填 | 说明 |
|--------|------|------|
| Content-Type | 是 | application/json |
| X-Request-ID | 否 | 请求唯一标识，用于请求去重和追踪 |
| X-Deduplication-ID | 否 | 去重标识，支持幂等操作 |
| X-App-ID | 否 | 应用标识 |
| Accept-Encoding | 否 | 支持 gzip, br 压缩 |

## 通用响应头

| 响应头 | 说明 |
|--------|------|
| X-Request-ID | 请求追踪标识 |
| X-Deduplication-ID | 去重标识 |
| X-Duplicate-Request | 是否为重复请求 (true/false) |
| X-RateLimit-Limit | 速率限制上限 |
| X-RateLimit-Remaining | 剩余请求次数 |
| X-RateLimit-Reset | 限流重置时间戳 |
| ETag | 资源标识，用于缓存验证 |
| Cache-Control | 缓存控制策略 |
| Content-Encoding | 响应编码 (gzip, br) |

---

## API v1

### 1. 生成滑块验证码

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

**响应参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 验证码ID |
| background_b64 | string | 背景图（Base64） |
| slider_b64 | string | 滑块图（Base64） |
| target_x | int | 目标位置X坐标 |
| target_y | int | 目标位置Y坐标 |

---

### 2. 验证滑块验证码

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

**响应参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| success | bool | 验证是否成功 |
| message | string | 验证结果描述 |

---

### 3. 生成点选验证码

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
| client_info | string | 否 | 客户端信息 |

**响应参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 验证码ID |
| image | string | 验证码图片（Base64） |
| target_chars | string[] | 需要点击的字符列表 |
| char_positions | object[] | 字符位置信息 |

---

### 4. 验证点选验证码

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

---

### 5. 生成拼图验证码

**请求**

```http
POST /api/v1/captcha/puzzle
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| app_id | string | 是 | 应用标识 |
| width | int | 否 | 图片宽度 |
| height | int | 否 | 图片高度 |
| client_info | string | 否 | 客户端信息 |

**响应参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 验证码ID |
| background_b64 | string | 背景图（Base64） |
| puzzle_b64 | string | 拼图块（Base64） |
| target_x | int | 目标位置X坐标 |
| target_y | int | 目标位置Y坐标 |

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

---

## API v2

### 验证码接口 (v2)

v2 版本接口与 v1 类似，但增加了以下特性：
- 支持 scenario_id 参数指定验证场景
- 响应包含 difficulty 和 expires_in 字段
- 验证结果自动触发 webhook 回调

---

### 1. 批量验证

一次请求验证多个验证码。

**请求**

```http
POST /api/v2/captcha/batch/verify
Content-Type: application/json
X-Deduplication-ID: <unique-id>
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | array | 是 | 验证项列表（最多100个） |

**items 参数结构**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| captcha_id | string | 是 | 验证码ID |
| type | string | 是 | 验证码类型 (slider/click/puzzle) |
| target_x | int | 是 | X坐标 |
| target_y | int | 否 | Y坐标 |
| clicks | array | 否 | 点选验证码的点击位置 |

**请求示例**

```json
{
  "items": [
    {
      "captcha_id": "cap_123",
      "type": "slider",
      "target_x": 150,
      "target_y": 25
    },
    {
      "captcha_id": "cap_456",
      "type": "click",
      "clicks": [
        {"char": "中", "x": 45, "y": 30},
        {"char": "国", "x": 120, "y": 25}
      ]
    }
  ]
}
```

**响应参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| results | array | 每个验证码的验证结果 |
| summary | object | 汇总统计 |

**响应示例**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "results": [
      {
        "captcha_id": "cap_123",
        "success": true,
        "message": "verification successful"
      },
      {
        "captcha_id": "cap_456",
        "success": true,
        "score": 0.95,
        "message": "verification successful"
      }
    ],
    "summary": {
      "total": 2,
      "success": 2,
      "failed": 0,
      "skipped": 0
    }
  }
}
```

---

### 2. 验证场景管理

#### 2.1 创建验证场景

**请求**

```http
POST /api/v2/captcha/scenarios
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 场景名称 |
| description | string | 否 | 场景描述 |
| difficulty | string | 否 | 难度等级 (easy/medium/hard) |
| config | object | 否 | 自定义配置 |

**请求示例**

```json
{
  "name": "登录验证",
  "description": "用于用户登录时的验证码验证",
  "difficulty": "medium",
  "config": {
    "tolerance": 5,
    "timeout": 300,
    "max_attempts": 3
  }
}
```

**响应示例**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "sce_abc123",
    "name": "登录验证",
    "description": "用于用户登录时的验证码验证",
    "difficulty": "medium",
    "config": {
      "tolerance": 5,
      "timeout": 300,
      "max_attempts": 3
    },
    "created_at": "2026-05-14T10:00:00Z",
    "updated_at": "2026-05-14T10:00:00Z"
  }
}
```

#### 2.2 获取场景列表

**请求**

```http
GET /api/v2/captcha/scenarios
```

**响应参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| scenarios | array | 场景列表 |
| total | int | 场景总数 |

#### 2.3 获取单个场景

**请求**

```http
GET /api/v2/captcha/scenarios/:id
```

#### 2.4 更新场景

**请求**

```http
PUT /api/v2/captcha/scenarios/:id
Content-Type: application/json
```

**可更新字段**

| 参数 | 类型 | 说明 |
|------|------|------|
| name | string | 场景名称 |
| description | string | 场景描述 |
| difficulty | string | 难度等级 |
| config | object | 自定义配置 |

#### 2.5 删除场景

**请求**

```http
DELETE /api/v2/captcha/scenarios/:id
```

---

### 3. Webhook 回调

#### 3.1 注册 Webhook

**请求**

```http
POST /api/v2/captcha/webhook/register
Content-Type: application/json
```

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| app_id | string | 是 | 应用标识 |
| url | string | 是 | 回调URL |
| secret | string | 否 | 签名密钥 |
| events | array | 是 | 订阅的事件 |
| headers | object | 否 | 自定义请求头 |

**events 支持的事件**

| 事件 | 说明 |
|------|------|
| verification.completed | 验证完成 |
| batch.verification.completed | 批量验证完成 |
| * | 订阅所有事件 |

**请求示例**

```json
{
  "app_id": "my-app-001",
  "url": "https://example.com/webhook/captcha",
  "secret": "my-secret-key",
  "events": ["verification.completed", "batch.verification.completed"],
  "headers": {
    "Authorization": "Bearer token123"
  }
}
```

**响应示例**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "wh_xyz789",
    "app_id": "my-app-001",
    "url": "https://example.com/webhook/captcha",
    "secret": "my-secret-key",
    "events": ["verification.completed", "batch.verification.completed"],
    "enabled": true,
    "headers": {
      "Authorization": "Bearer token123"
    },
    "created_at": "2026-05-14T10:00:00Z",
    "updated_at": "2026-05-14T10:00:00Z"
  }
}
```

#### 3.2 获取 Webhook 列表

**请求**

```http
GET /api/v2/captcha/webhook?app_id=my-app-001
```

**查询参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| app_id | string | 按应用筛选 |

#### 3.3 更新 Webhook

**请求**

```http
PUT /api/v2/captcha/webhook/:id
Content-Type: application/json
```

#### 3.4 删除 Webhook

**请求**

```http
DELETE /api/v2/captcha/webhook/:id
```

---

### 4. Webhook 回调格式

当验证事件触发时，系统会向已注册的 URL 发送 POST 请求。

**回调请求头**

| 请求头 | 说明 |
|--------|------|
| Content-Type | application/json |
| X-Webhook-Event | 事件类型 |
| X-Webhook-ID | Webhook ID |
| X-Webhook-Signature | 签名（如果配置了secret） |

**回调请求体 (verification.completed)**

```json
{
  "captcha_id": "cap_abc123",
  "type": "slider",
  "success": true,
  "message": "verification successful",
  "timestamp": "2026-05-14T10:00:00Z"
}
```

**回调请求体 (batch.verification.completed)**

```json
{
  "results": [
    {
      "captcha_id": "cap_abc123",
      "success": true,
      "message": "verification successful"
    }
  ],
  "summary": {
    "total": 1,
    "success": 1,
    "failed": 0,
    "skipped": 0
  },
  "timestamp": "2026-05-14T10:00:00Z"
}
```

---

## 健康检查

**请求**

```http
GET /health
```

**响应示例 (v2)**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "status": "healthy",
    "service": "captchax-api",
    "timestamp": "2026-05-14T10:00:00Z",
    "version": "2.0.0"
  }
}
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
| 4091 | 409 | 重复请求 | 使用相同的 X-Deduplication-ID |
| 5001 | 500 | 服务器内部错误 | 联系技术支持 |

---

## 缓存策略

### ETag 支持

v2 API 支持 HTTP 缓存机制：

1. **ETag**: 服务端返回 ETag 响应头
2. **If-None-Match**: 客户端可在后续请求中携带此头
3. **Last-Modified**: 资源最后修改时间
4. **If-Modified-Since**: 条件请求头

### 缓存响应示例

```http
GET /api/v2/captcha/scenarios HTTP/1.1
Accept: application/json

HTTP/1.1 200 OK
Content-Type: application/json
ETag: "abc123"
Cache-Control: private, max-age=300
Last-Modified: Wed, 14 May 2026 10:00:00 GMT
```

---

## 请求去重

### 工作原理

1. 客户端在请求头中设置 `X-Deduplication-ID`
2. 服务端在配置的窗口时间内记录该 ID
3. 如果收到相同 ID 的请求，返回 409 Conflict

### 去重响应

```http
HTTP/1.1 409 Conflict
X-Duplicate-Request: true
Retry-After: 60
```

---

## 压缩支持

### 支持的编码

- **gzip**: 最广泛支持
- **br**: Brotli，更高压缩率

### 请求示例

```http
GET /api/v2/captcha/scenarios HTTP/1.1
Accept-Encoding: gzip, br
```

### 响应示例

```http
HTTP/1.1 200 OK
Content-Encoding: gzip
Vary: Accept-Encoding
```
