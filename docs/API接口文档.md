# API接口文档

本文档详细描述行为验证系统的所有API接口，包括请求参数、响应格式和错误码。

## 基础信息

### 基础URL

```
生产环境: https://api.example.com
开发环境: http://localhost:8080
```

### 认证方式

除公开接口外，所有接口都需要携带JWT Token进行认证：

```
Authorization: Bearer <token>
```

### 统一响应格式

所有API响应均采用统一格式：

```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| code | int | 状态码，0表示成功 |
| message | string | 状态信息 |
| data | object | 响应数据 |

### 错误码

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 1001 | 参数错误 |
| 1002 | 缺少必填参数 |
| 1003 | 参数格式错误 |
| 2001 | 认证失败 |
| 2002 | Token过期 |
| 2003 | 权限不足 |
| 3001 | 验证码生成失败 |
| 3002 | 验证码验证失败 |
| 3003 | 验证码已过期 |
| 3004 | 验证码类型不匹配 |
| 4001 | 资源不存在 |
| 4002 | 资源已存在 |
| 5001 | 服务器内部错误 |
| 5002 | 服务暂不可用 |

---

## 验证码API

### 1. 滑块验证码

#### 生成滑块验证码

获取一个新的滑块验证码。

**请求**

```http
GET /api/v1/captcha/slider
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "session_id": "sess_1715000000000_1234",
    "image_url": "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcv...",
    "puzzle_y": 120
  }
}
```

**响应字段说明**

| 字段 | 类型 | 说明 |
|------|------|------|
| session_id | string | 验证码会话ID，用于后续验证 |
| image_url | string | 验证码图片URL（Base64编码） |
| puzzle_y | int | 拼图块Y轴位置（像素） |

#### 验证滑块

提交滑块验证请求。

**请求**

```http
POST /api/v1/captcha/verify
Content-Type: application/json

{
  "session_id": "sess_1715000000000_1234",
  "type": "slider",
  "x": 185,
  "y": 120,
  "behavior_data": [
    {
      "x": 100,
      "y": 120,
      "timestamp": 1715000001000,
      "event": "move"
    }
  ]
}
```

**请求参数说明**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| session_id | string | 是 | 验证码会话ID |
| type | string | 是 | 验证码类型：`slider` |
| x | int | 是 | 滑块最终X坐标 |
| y | int | 是 | 滑块最终Y坐标 |
| behavior_data | array | 否 | 行为数据数组 |
| application_id | int | 否 | 应用ID |

**behavior_data 字段说明**

| 字段 | 类型 | 说明 |
|------|------|------|
| x | int | 鼠标X坐标 |
| y | int | 鼠标Y坐标 |
| timestamp | int64 | 时间戳（毫秒） |
| event | string | 事件类型：`move`, `mousedown`, `mouseup` |

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "success": true,
    "message": "验证成功",
    "risk_score": 15.5,
    "captcha_pass": true
  }
}
```

**响应字段说明**

| 字段 | 类型 | 说明 |
|------|------|------|
| success | bool | 最终验证结果（结合行为分析） |
| message | string | 结果描述 |
| risk_score | float | 风险评分（0-100） |
| captcha_pass | bool | 验证码是否正确 |

---

### 2. 点选验证码

#### 生成点选验证码

获取一个新的点选验证码。

**请求**

```http
GET /api/v1/captcha/click?mode=number&shuffle=true&points=3
```

**Query参数说明**

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| mode | string | number | 字符模式：`number`, `letter`, `chinese`, `mixed` |
| shuffle | string | true | 是否打乱点击顺序 |
| points | int | 3 | 目标点数量（2-6） |

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "session_id": "sess_1715000000000_5678",
    "image_url": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
    "hint": "点击: 3 → 7 → 5",
    "hint_order": [2, 0, 1],
    "max_points": 3,
    "mode": "number",
    "allow_shuffle": true
  }
}
```

**响应字段说明**

| 字段 | 类型 | 说明 |
|------|------|------|
| session_id | string | 验证码会话ID |
| image_url | string | 验证码图片URL |
| hint | string | 点击提示文字 |
| hint_order | array | 期望的点击顺序 |
| max_points | int | 目标点数量 |
| mode | string | 当前使用的字符模式 |
| allow_shuffle | bool | 是否允许打乱顺序 |

#### 验证点选

提交点选验证请求。

**请求**

```http
POST /api/v1/captcha/verify
Content-Type: application/json

{
  "session_id": "sess_1715000000000_5678",
  "type": "click",
  "points": [
    [80, 100],
    [160, 100],
    [240, 100]
  ],
  "click_sequence": [0, 1, 2],
  "behavior_data": [...]
}
```

**请求参数说明**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| session_id | string | 是 | 验证码会话ID |
| type | string | 是 | 验证码类型：`click` |
| points | array | 是 | 点击坐标数组 [[x1,y1], [x2,y2], ...] |
| click_sequence | array | 否 | 点击顺序索引 |
| behavior_data | array | 否 | 行为数据数组 |

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "success": true,
    "message": "验证成功",
    "risk_score": 12.3,
    "captcha_pass": true
  }
}
```

---

### 3. 图形验证码

#### 生成图形验证码

获取一个新的图形验证码。

**请求**

```http
GET /api/v1/captcha/image?type=mixed&count=4
```

**Query参数说明**

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| type | string | mixed | 字符类型：`number`, `letter`, `mixed` |
| count | int | 4 | 字符数量（4-6） |
| noise_mode | int | 0 | 噪点模式（0-3） |
| line_mode | int | 0 | 干扰线模式（0-3） |

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "challenge_id": "img_1715000000000",
    "image": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
  }
}
```

#### 验证图形验证码

**请求**

```http
POST /api/v1/captcha/image/verify
Content-Type: application/json

{
  "challenge_id": "img_1715000000000",
  "answer": "A3B7"
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "success": true
  }
}
```

---

## 用户认证API

### 注册

创建新用户账户。

**请求**

```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "username": "user@example.com",
  "password": "password123",
  "email": "user@example.com"
}
```

**请求参数说明**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 用户名（邮箱） |
| password | string | 是 | 密码（6-32位） |
| email | string | 是 | 邮箱地址 |

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "user_id": 1,
    "username": "user@example.com"
  }
}
```

### 登录

用户登录获取Token。

**请求**

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "user@example.com",
  "password": "password123"
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_at": "2024-05-17T12:00:00Z",
    "user": {
      "id": 1,
      "username": "user@example.com"
    }
  }
}
```

### 登出

用户登出，使Token失效。

**请求**

```http
POST /api/v1/auth/logout
Authorization: Bearer <token>
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

### 刷新Token

刷新Access Token。

**请求**

```http
POST /api/v1/auth/refresh
Authorization: Bearer <token>
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_at": "2024-05-18T12:00:00Z"
  }
}
```

### 请求密码重置

发送密码重置邮件。

**请求**

```http
POST /api/v1/auth/request-password-reset
Content-Type: application/json

{
  "email": "user@example.com"
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

### 重置密码

使用Token重置密码。

**请求**

```http
POST /api/v1/auth/reset-password
Content-Type: application/json

{
  "token": "reset_token_here",
  "new_password": "newpassword123"
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

---

## 用户资料API

### 获取用户资料

**请求**

```http
GET /api/v1/user/profile
Authorization: Bearer <token>
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "username": "user@example.com",
    "email": "user@example.com",
    "created_at": "2024-05-01T10:00:00Z",
    "updated_at": "2024-05-10T15:30:00Z"
  }
}
```

### 更新用户资料

**请求**

```http
PUT /api/v1/user/profile
Authorization: Bearer <token>
Content-Type: application/json

{
  "email": "new_email@example.com"
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "username": "user@example.com",
    "email": "new_email@example.com",
    "updated_at": "2024-05-16T12:00:00Z"
  }
}
```

### 修改密码

**请求**

```http
POST /api/v1/user/change-password
Authorization: Bearer <token>
Content-Type: application/json

{
  "old_password": "oldpassword",
  "new_password": "newpassword123"
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

---

## 管理端API

> ⚠️ 所有管理端API都需要管理员权限

### 管理员登录

**请求**

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "admin123"
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_at": "2024-05-17T12:00:00Z",
    "user": {
      "id": 1,
      "username": "admin",
      "role": "admin"
    }
  }
}
```

### 验证统计

获取验证统计数据。

**请求**

```http
GET /api/v1/admin/stats/verification
Authorization: Bearer <admin_token>
```

**Query参数说明**

| 参数 | 类型 | 说明 |
|------|------|------|
| start_date | string | 开始日期（YYYY-MM-DD） |
| end_date | string | 结束日期（YYYY-MM-DD） |
| app_id | int | 应用ID筛选 |

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 10000,
    "success": 8500,
    "failed": 1500,
    "pass_rate": 85.0,
    "risk_stats": {
      "low": 7500,
      "medium": 800,
      "high": 200,
      "critical": 0
    }
  }
}
```

### 图表数据

获取用于图表展示的数据。

**请求**

```http
GET /api/v1/admin/stats/chart
Authorization: Bearer <admin_token>
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "labels": ["2024-05-10", "2024-05-11", "2024-05-12"],
    "datasets": [
      {
        "label": "验证总量",
        "data": [1000, 1200, 1100]
      },
      {
        "label": "成功数量",
        "data": [850, 1020, 935]
      }
    ]
  }
}
```

### 趋势数据

获取验证趋势数据。

**请求**

```http
GET /api/v1/admin/stats/trend
Authorization: Bearer <admin_token>
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "hourly": [...],
    "daily": [...],
    "weekly": [...],
    "monthly": [...]
  }
}
```

### 实时监控

获取实时验证监控数据。

**请求**

```http
GET /api/v1/admin/stats/realtime
Authorization: Bearer <admin_token>
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "current_qps": 125,
    "total_today": 5000,
    "success_rate": 85.5,
    "avg_response_time": 45,
    "active_sessions": 23
  }
}
```

### 风险分布

获取风险评分分布数据。

**请求**

```http
GET /api/v1/admin/stats/risk-distribution
Authorization: Bearer <admin_token>
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "distribution": {
      "0-20": 6000,
      "20-40": 2000,
      "40-60": 1000,
      "60-80": 500,
      "80-100": 500
    },
    "average_score": 25.5
  }
}
```

### 应用列表

获取应用列表。

**请求**

```http
GET /api/v1/admin/applications
Authorization: Bearer <admin_token>
```

**Query参数说明**

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码 |
| page_size | int | 每页数量 |
| keyword | string | 关键词搜索 |

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": 1,
        "name": "测试应用",
        "app_key": "app_xxxxxxxxxxxx",
        "status": "active",
        "created_at": "2024-05-01T10:00:00Z"
      }
    ],
    "total": 10,
    "page": 1,
    "page_size": 20
  }
}
```

### 创建应用

**请求**

```http
POST /api/v1/admin/applications
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "name": "新应用",
  "description": "应用描述"
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 2,
    "name": "新应用",
    "app_key": "app_yyyyyyyyyyyy",
    "app_secret": "secret_zzzzzzzzzzzz",
    "created_at": "2024-05-16T12:00:00Z"
  }
}
```

### 更新应用

**请求**

```http
PUT /api/v1/admin/applications/2
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "name": "更新后的应用名",
  "status": "inactive"
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 2,
    "name": "更新后的应用名",
    "status": "inactive",
    "updated_at": "2024-05-16T12:00:00Z"
  }
}
```

### 删除应用

**请求**

```http
DELETE /api/v1/admin/applications/2
Authorization: Bearer <admin_token>
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

### 验证日志

获取验证日志列表。

**请求**

```http
GET /api/v1/admin/logs
Authorization: Bearer <admin_token>
```

**Query参数说明**

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码 |
| page_size | int | 每页数量 |
| session_id | string | 会话ID筛选 |
| status | string | 状态筛选：`success`, `failed` |
| risk_level | string | 风险等级 |
| start_date | string | 开始日期 |
| end_date | string | 结束日期 |

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": 1,
        "session_id": "sess_xxx",
        "captcha_type": "slider",
        "status": "success",
        "risk_score": 15.5,
        "ip_address": "192.168.1.1",
        "created_at": "2024-05-16T10:00:00Z"
      }
    ],
    "total": 1000,
    "page": 1,
    "page_size": 20
  }
}
```

### 日志详情

获取单条日志详情。

**请求**

```http
GET /api/v1/admin/logs/1
Authorization: Bearer <admin_token>
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "session_id": "sess_xxx",
    "captcha_type": "slider",
    "status": "success",
    "risk_score": 15.5,
    "analysis_result": "轨迹正常，速度正常",
    "ip_address": "192.168.1.1",
    "user_agent": "Mozilla/5.0...",
    "duration": 1500,
    "created_at": "2024-05-16T10:00:00Z"
  }
}
```

### 导出日志

导出CSV格式的日志数据。

**请求**

```http
GET /api/v1/admin/logs/export?start_date=2024-05-01&end_date=2024-05-16
Authorization: Bearer <admin_token>
```

**响应**

CSV文件下载

### 黑名单管理

#### 获取黑名单

**请求**

```http
GET /api/v1/admin/blacklist
Authorization: Bearer <admin_token>
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": 1,
        "type": "ip",
        "value": "192.168.1.100",
        "reason": "恶意攻击",
        "created_at": "2024-05-15T10:00:00Z"
      }
    ],
    "total": 10
  }
}
```

#### 添加黑名单

**请求**

```http
POST /api/v1/admin/blacklist
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "type": "ip",
  "value": "192.168.1.100",
  "reason": "恶意攻击"
}
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "type": "ip",
    "value": "192.168.1.100",
    "created_at": "2024-05-16T12:00:00Z"
  }
}
```

#### 删除黑名单

**请求**

```http
DELETE /api/v1/admin/blacklist/1
Authorization: Bearer <admin_token>
```

**响应**

```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

---

## 健康检查

### 健康检查

获取服务健康状态。

**请求**

```http
GET /health
```

**响应**

```json
{
  "status": "healthy",
  "timestamp": "2024-05-16T12:00:00Z",
  "services": {
    "database": "up",
    "redis": "up"
  }
}
```

---

## SDK使用示例

### Go SDK

```go
package main

import (
    "fmt"
    "log"
    "time"

    "github.com/opphk/behavior-verification-system/sdk/go/captcha"
)

func main() {
    // 创建客户端
    client := captcha.NewClient(
        captcha.WithEndpoint("http://localhost:8080"),
        captcha.WithTimeout(30*time.Second),
    )

    // 生成滑块验证码
    sliderResp, err := client.GetSliderCaptcha(&captcha.SliderCaptchaRequest{
        Width:  360,
        Height: 220,
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("SessionID: %s\n", sliderResp.ChallengeID)

    // 验证
    verifyResp, err := client.VerifyCaptcha(&captcha.VerifyCaptchaRequest{
        ChallengeID: sliderResp.ChallengeID,
        Action:     "slider",
        Data: map[string]interface{}{
            "x": 185,
            "y": 120,
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("验证结果: %v\n", verifyResp.Success)
}
```

---

## 速率限制

系统对API接口实施了速率限制：

| 接口类型 | 限制 | 窗口 |
|----------|------|------|
| 验证码生成 | 100次/分钟 | 滑动窗口 |
| 验证码验证 | 200次/分钟 | 滑动窗口 |
| 用户认证 | 10次/分钟 | 固定窗口 |
| 管理接口 | 60次/分钟 | 滑动窗口 |

超出限制将返回 `429 Too Many Requests` 错误。

---

## 附录：错误响应示例

### 参数错误

```json
{
  "code": 1001,
  "message": "参数错误",
  "data": {
    "field": "session_id",
    "reason": "会话ID格式不正确"
  }
}
```

### 认证失败

```json
{
  "code": 2001,
  "message": "认证失败",
  "data": null
}
```

### 权限不足

```json
{
  "code": 2003,
  "message": "权限不足",
  "data": null
}
```

### 速率限制

```json
{
  "code": 429,
  "message": "请求过于频繁，请稍后再试",
  "data": {
    "retry_after": 60
  }
}
```
