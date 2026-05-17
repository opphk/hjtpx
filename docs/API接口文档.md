# API 接口文档 v6.0

## 目录

1. [概述](#概述)
2. [认证](#认证)
3. [用户端 API](#用户端-api)
4. [管理端 API](#管理端-api)
5. [错误码](#错误码)
6. [示例](#示例)

## 概述

### 基础 URL

- 生产环境: `https://api.example.com/api/v1`
- 开发环境: `http://localhost:8080/api/v1`

### 数据格式

所有请求和响应均使用 JSON 格式。

### 认证方式

除公开接口外，所有接口都需要携带 JWT Token 进行认证：

```
Authorization: Bearer <token>
```

### 统一响应格式

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

---

## 认证

### 管理员登录

```
POST /auth/login
```

**请求参数：**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| username | string | 是 | 用户名 |
| password | string | 是 | 密码 |

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_at": "2026-05-17T12:00:00Z",
    "user": {
      "id": 1,
      "username": "admin",
      "role": "admin"
    }
  }
}
```

### 用户登录

```
POST /auth/login
```

**请求参数：**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| username | string | 是 | 用户名（邮箱） |
| password | string | 是 | 密码 |

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_at": "2026-05-17T12:00:00Z",
    "user": {
      "id": 1,
      "username": "user@example.com"
    }
  }
}
```

### 用户注册

```
POST /auth/register
```

**请求参数：**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| username | string | 是 | 用户名（邮箱） |
| password | string | 是 | 密码（6-32位） |
| email | string | 是 | 邮箱地址 |

**响应示例：**

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

---

## 用户端 API

### 滑块验证码

#### 生成滑块验证码

```
POST /captcha/slider/generate
```

**请求参数：**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| app_id | string | 是 | 应用ID |
| app_key | string | 是 | 应用密钥 |

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "captcha_id": "uuid",
    "background_image": "base64...",
    "slider_image": "base64...",
    "track_data": {
      "y": 100,
      "width": 200
    }
  }
}
```

#### 验证滑块

```
POST /captcha/verify
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

**响应示例：**

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

---

### 点选验证码

#### 生成点选验证码

```
POST /captcha/click/generate
```

**请求参数：**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| mode | string | 否 | 字符模式：number, letter, chinese, mixed |
| shuffle | boolean | 否 | 是否打乱点击顺序 |
| points | int | 否 | 目标点数量（2-6），默认3 |

**响应示例：**

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

#### 验证点选

```
POST /captcha/verify
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

**响应示例：**

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

### 旋转验证码

#### 生成旋转验证码

```
POST /captcha/rotate/generate
```

**请求参数：**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| difficulty | string | 否 | 难度级别：easy, medium, hard, expert |

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "session_id": "sess_rotate_xxx",
    "background_image": "data:image/png;base64...",
    "rotated_image": "data:image/png;base64...",
    "target_angle": 127,
    "difficulty": "medium"
  }
}
```

#### 验证旋转

```
POST /captcha/rotate/verify
Content-Type: application/json

{
  "session_id": "sess_rotate_xxx",
  "angle": 125,
  "behavior_data": [...]
}
```

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "success": true,
    "risk_score": 10.5,
    "captcha_pass": true
  }
}
```

---

### 手势验证码

#### 生成手势验证码

```
POST /captcha/gesture/generate
```

**请求参数：**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| difficulty | string | 否 | 难度级别：easy, medium, hard, expert |

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "session_id": "sess_gesture_xxx",
    "pattern_image": "data:image/png;base64...",
    "pattern_type": "L",
    "difficulty": "medium"
  }
}
```

#### 验证手势

```
POST /captcha/gesture/verify
Content-Type: application/json

{
  "session_id": "sess_gesture_xxx",
  "gesture_path": [
    [50, 50], [150, 50], [150, 150]
  ],
  "behavior_data": [...]
}
```

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "success": true,
    "similarity": 0.92,
    "risk_score": 8.5,
    "captcha_pass": true
  }
}
```

---

### 拼图验证码

#### 生成拼图验证码

```
POST /captcha/puzzle/generate
```

**请求参数：**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| difficulty | string | 否 | 难度级别：easy, medium, hard, expert |
| pieces | int | 否 | 碎片数量（4-9），默认4 |

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "session_id": "sess_puzzle_xxx",
    "background_image": "data:image/png;base64...",
    "puzzle_image": "data:image/png;base64...",
    "piece_positions": [[120, 80]],
    "difficulty": "medium",
    "pieces": 4
  }
}
```

#### 验证拼图

```
POST /captcha/puzzle/verify
Content-Type: application/json

{
  "session_id": "sess_puzzle_xxx",
  "piece_positions": [[122, 82]],
  "behavior_data": [...]
}
```

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "success": true,
    "accuracy": 0.95,
    "risk_score": 11.2,
    "captcha_pass": true
  }
}
```

---

### 图形验证码

#### 生成图形验证码

```
GET /captcha/image
```

**Query参数说明：**

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| type | string | mixed | 字符类型：number, letter, mixed |
| count | int | 4 | 字符数量（4-6） |

**响应示例：**

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

```
POST /captcha/image/verify
Content-Type: application/json

{
  "challenge_id": "img_1715000000000",
  "answer": "A3B7"
}
```

**响应示例：**

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

### 无感验证

#### 获取设备信任状态

```
POST /seamless/check
Content-Type: application/json

{
  "device_fingerprint": "fp_xxx",
  "behavior_sequence": [
    {"event": "mousemove", "timestamp": 1715000001000},
    {"event": "click", "timestamp": 1715000001500}
  ]
}
```

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "trust_level": "high",
    "risk_score": 5.2,
    "requires_captcha": false,
    "trust_duration": 3600
  }
}
```

---

### 环境检测

#### 环境检测

```
POST /detect/check
Content-Type: application/json

{
  "fingerprint": {
    "canvas": "canvas_fp_hash",
    "webgl": "webgl_fp_hash",
    "fonts": ["font1", "font2"],
    "plugins": ["plugin1"],
    "timezone": "Asia/Shanghai",
    "language": "zh-CN",
    "user_agent": "Mozilla/5.0..."
  }
}
```

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "is_proxy": false,
    "is_vpn": false,
    "is_tor": false,
    "is_emulator": false,
    "is_real_browser": true,
    "risk_score": 10.5,
    "fingerprint_id": "fp_unique_id"
  }
}
```

---

## 管理端 API

> 所有管理端 API 都需要管理员权限

### 应用管理

#### 创建应用

```
POST /admin/applications
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "name": "新应用",
  "description": "应用描述"
}
```

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 2,
    "name": "新应用",
    "app_key": "app_yyyyyyyyyyyy",
    "app_secret": "secret_zzzzzzzzzzzz",
    "created_at": "2026-05-17T12:00:00Z"
  }
}
```

#### 更新应用

```
PUT /admin/applications/:id
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "name": "更新后的应用名",
  "status": "inactive"
}
```

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 2,
    "name": "更新后的应用名",
    "status": "inactive",
    "updated_at": "2026-05-17T12:00:00Z"
  }
}
```

#### 获取应用列表

```
GET /admin/applications
Authorization: Bearer <admin_token>
```

**响应示例：**

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
        "created_at": "2026-05-01T10:00:00Z"
      }
    ],
    "total": 10,
    "page": 1,
    "page_size": 20
  }
}
```

#### 删除应用

```
DELETE /admin/applications/:id
Authorization: Bearer <admin_token>
```

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

---

### 统计接口

#### 获取验证统计

```
GET /admin/stats/verification
Authorization: Bearer <admin_token>
```

**响应示例：**

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

#### 获取实时监控数据

```
GET /admin/stats/realtime
Authorization: Bearer <admin_token>
```

**响应示例：**

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

#### 获取趋势数据

```
GET /admin/stats/trend
Authorization: Bearer <admin_token>
```

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "labels": ["2026-05-10", "2026-05-11", "2026-05-12"],
    "verification_counts": [1000, 1200, 1100],
    "pass_rates": [85.2, 86.1, 84.8],
    "avg_response_times": [45, 42, 48]
  }
}
```

#### 获取风险分布

```
GET /admin/stats/risk-distribution
Authorization: Bearer <admin_token>
```

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "low": 7500,
    "medium": 1500,
    "high": 500,
    "critical": 100
  }
}
```

---

### 实时监控 WebSocket

#### 连接实时监控

```
WS /admin/realtime/monitor
Authorization: Bearer <admin_token>
```

**推送数据示例：**

```json
{
  "type": "metrics",
  "data": {
    "qps": 125,
    "active_sessions": 23,
    "cpu_usage": 45.2,
    "memory_usage": 62.8,
    "redis_hits": 9500,
    "redis_misses": 500
  }
}
```

---

### 黑名单管理

#### 获取黑名单

```
GET /admin/blacklist
Authorization: Bearer <admin_token>
```

**响应示例：**

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
        "created_at": "2026-05-15T10:00:00Z"
      }
    ],
    "total": 10
  }
}
```

#### 添加黑名单

```
POST /admin/blacklist
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "type": "ip",
  "value": "192.168.1.100",
  "reason": "恶意攻击"
}
```

#### 删除黑名单

```
DELETE /admin/blacklist/:id
Authorization: Bearer <admin_token>
```

---

### 风控规则管理

#### 获取风控规则

```
GET /admin/risk-rules
Authorization: Bearer <admin_token>
```

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "rules": [
      {
        "id": 1,
        "name": "快速连续点击",
        "type": "click_pattern",
        "threshold": 5,
        "action": "block",
        "enabled": true
      }
    ]
  }
}
```

#### 更新风控规则

```
PUT /admin/risk-rules
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "rules": [
    {
      "id": 1,
      "threshold": 10,
      "enabled": true
    }
  ]
}
```

---

### 日志管理

#### 获取验证日志

```
GET /admin/logs
Authorization: Bearer <admin_token>
```

**响应示例：**

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
        "created_at": "2026-05-17T10:00:00Z"
      }
    ],
    "total": 1000,
    "page": 1,
    "page_size": 20
  }
}
```

#### 导出日志

```
GET /admin/logs/export
Authorization: Bearer <admin_token>
```

返回 CSV 格式的日志文件。

---

### 告警管理

#### 获取告警列表

```
GET /admin/alerts
Authorization: Bearer <admin_token>
```

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": 1,
        "type": "high_risk",
        "message": "检测到高风险行为",
        "status": "resolved",
        "created_at": "2026-05-17T10:00:00Z"
      }
    ],
    "total": 5
  }
}
```

#### 创建告警规则

```
POST /admin/alerts
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "type": "high_risk",
  "threshold": 80,
  "channels": ["email", "webhook"],
  "enabled": true
}
```

---

### 审计日志

#### 获取审计日志

```
GET /admin/audit-logs
Authorization: Bearer <admin_token>
```

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": 1,
        "user_id": 1,
        "action": "update_config",
        "details": "修改了验证码难度设置",
        "ip_address": "192.168.1.1",
        "created_at": "2026-05-17T10:00:00Z"
      }
    ],
    "total": 100
  }
}
```

---

### 健康检查

#### 健康检查接口

```
GET /health
```

**响应示例：**

```json
{
  "status": "healthy",
  "timestamp": "2026-05-17T12:00:00Z",
  "services": {
    "database": "up",
    "redis": "up"
  }
}
```

---

## 错误码

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 1001 | 参数错误 |
| 1002 | 认证失败 |
| 1003 | 权限不足 |
| 2001 | 验证码生成失败 |
| 2002 | 验证码验证失败 |
| 2003 | 验证码已过期 |
| 2004 | 验证码类型不支持 |
| 3001 | 服务器内部错误 |

---

## 示例

### Go SDK 示例

```go
package main

import (
    "fmt"
    "log"
    "time"

    "github.com/opphk/hjtpx/sdk/go/captcha"
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
        Action:      "slider",
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

### JavaScript SDK 示例

```javascript
const { CaptchaClient } = require('hjtpx-sdk');

const client = new CaptchaClient({
    endpoint: 'http://localhost:8080',
    timeout: 30000,
});

async function demo() {
    // 生成滑块验证码
    const slider = await client.generateSlider();
    console.log('Captcha ID:', slider.captcha_id);

    // 验证
    const result = await client.verify(slider.captcha_id, {
        type: 'slider',
        x: 185,
        y: 120,
    });
    console.log('验证结果:', result.success);
}

demo().catch(console.error);
```

### Python SDK 示例

```python
from hjtpx import CaptchaClient

client = CaptchaClient(endpoint="http://localhost:8080")

# 生成滑块验证码
slider = client.generate_slider()
print(f"Session ID: {slider['captcha_id']}")

# 验证
result = client.verify(slider['captcha_id'], {
    "type": "slider",
    "x": 185,
    "y": 120
})
print(f"验证结果: {result['success']}")
```

---

## 速率限制

系统对 API 接口实施了速率限制：

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
