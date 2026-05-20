# HJTPX API 文档 v4.0

## 概述

本文档详细描述了 HJTPX 智能验证码系统的完整 REST API 接口。系统采用 OpenAPI 3.1 规范，支持多租户、分布式部署，提供企业级的验证码服务。

**基础URL**: `http://localhost:8080/api/v1`

**认证方式**: Bearer Token (JWT)

---

## 认证接口

### 1. 用户登录

**端点**: `POST /auth/login`

**请求体**:
```json
{
  "username": "string",
  "password": "string"
}
```

**响应** (200 OK):
```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expiresAt": "2026-05-21T12:00:00Z",
    "user": {
      "id": 1,
      "username": "admin",
      "role": "admin"
    }
  }
}
```

**错误响应** (401 Unauthorized):
```json
{
  "success": false,
  "error": {
    "code": "AUTH_FAILED",
    "message": "Invalid username or password"
  }
}
```

### 2. 用户登出

**端点**: `POST /auth/logout`

**请求头**:
```
Authorization: Bearer <token>
```

**响应** (200 OK):
```json
{
  "success": true,
  "message": "Logged out successfully"
}
```

### 3. 刷新令牌

**端点**: `POST /auth/refresh`

**请求体**:
```json
{
  "refreshToken": "string"
}
```

**响应** (200 OK):
```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expiresAt": "2026-05-21T13:00:00Z"
  }
}
```

### 4. MFA 验证

**端点**: `POST /auth/mfa/verify`

**请求体**:
```json
{
  "userId": 1,
  "code": "123456",
  "method": "totp"
}
```

**响应** (200 OK):
```json
{
  "success": true,
  "data": {
    "verified": true,
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
}
```

---

## 验证码接口

### 1. 获取滑块验证码

**端点**: `GET /captcha/slider`

**查询参数**:
- `appId` (string, optional): 应用ID
- `width` (int, optional): 验证码宽度，默认 300
- `height` (int, optional): 验证码高度，默认 150
- `difficulty` (string, optional): 难度等级 (easy, medium, hard)

**响应** (200 OK):
```json
{
  "success": true,
  "data": {
    "captchaId": "cap_abc123def456",
    "sessionId": "sess_xyz789",
    "backgroundImage": "data:image/png;base64,...",
    "sliderImage": "data:image/png;base64,...",
    "targetX": 150,
    "targetY": 75,
    "expiresAt": "2026-05-20T13:05:00Z",
    "expiresIn": 300
  }
}
```

### 2. 验证滑块验证码

**端点**: `POST /captcha/slider/verify`

**请求体**:
```json
{
  "captchaId": "cap_abc123def456",
  "sessionId": "sess_xyz789",
  "answer": {
    "x": 150,
    "track": [0, 10, 20, 30, ...]
  }
}
```

**响应** (200 OK):
```json
{
  "success": true,
  "data": {
    "verified": true,
    "score": 95.5,
    "confidence": 0.98
  }
}
```

### 3. 获取点击验证码

**端点**: `GET /captcha/click`

**查询参数**:
- `appId` (string, optional): 应用ID
- `count` (int, optional): 点击数量，默认 4

**响应** (200 OK):
```json
{
  "success": true,
  "data": {
    "captchaId": "cap_click_xyz123",
    "sessionId": "sess_click_abc",
    "backgroundImage": "data:image/png;base64,...",
    "targetWords": ["红绿灯", "汽车", "行人", "自行车"],
    "imageWidth": 400,
    "imageHeight": 300
  }
}
```

### 4. 验证点击验证码

**端点**: `POST /captcha/click/verify`

**请求体**:
```json
{
  "captchaId": "cap_click_xyz123",
  "sessionId": "sess_click_abc",
  "answer": {
    "points": [
      { "word": "红绿灯", "x": 150, "y": 200 },
      { "word": "汽车", "x": 300, "y": 150 }
    ]
  }
}
```

**响应** (200 OK):
```json
{
  "success": true,
  "data": {
    "verified": true,
    "correctCount": 4,
    "totalCount": 4,
    "score": 100
  }
}
```

### 5. 获取旋转验证码

**端点**: `GET /captcha/rotate`

**查询参数**:
- `appId` (string, optional): 应用ID
- `angle` (int, optional): 目标角度，默认随机

**响应** (200 OK):
```json
{
  "success": true,
  "data": {
    "captchaId": "cap_rotate_123abc",
    "sessionId": "sess_rotate_456",
    "backgroundImage": "data:image/png;base64,...",
    "targetAngle": 127,
    "tolerance": 10
  }
}
```

### 6. 验证旋转验证码

**端点**: `POST /captcha/rotate/verify`

**请求体**:
```json
{
  "captchaId": "cap_rotate_123abc",
  "sessionId": "sess_rotate_456",
  "answer": {
    "angle": 125
  }
}
```

**响应** (200 OK):
```json
{
  "success": true,
  "data": {
    "verified": true,
    "submittedAngle": 125,
    "targetAngle": 127,
    "difference": 2
  }
}
```

### 7. 获取图片识别验证码

**端点**: `GET /captcha/image`

**查询参数**:
- `appId` (string, optional): 应用ID
- `type` (string, optional): 图片类型 (vehicle, animal, fruit)

**响应** (200 OK):
```json
{
  "success": true,
  "data": {
    "captchaId": "cap_image_abc123",
    "sessionId": "sess_image_xyz",
    "image": "data:image/png;base64,...",
    "options": ["猫", "狗", "兔子", "鸭子"],
    "correctAnswer": "猫"
  }
}
```

### 8. 获取语音验证码

**端点**: `GET /captcha/voice`

**响应** (200 OK):
```json
{
  "success": true,
  "data": {
    "captchaId": "cap_voice_abc",
    "sessionId": "sess_voice_xyz",
    "audioUrl": "http://localhost:8080/audio/cap_voice_abc.mp3",
    "expiresIn": 120
  }
}
```

### 9. 智能组合验证码

**端点**: `GET /captcha/combo`

**响应** (200 OK):
```json
{
  "success": true,
  "data": {
    "captchaId": "cap_combo_abc123",
    "types": ["slider", "click"],
    "currentType": "slider",
    "progress": {
      "slider": { "verified": false },
      "click": { "verified": false }
    }
  }
}
```

---

## 统计分析接口

### 1. 获取验证统计

**端点**: `GET /stats/verification`

**请求头**:
```
Authorization: Bearer <token>
```

**查询参数**:
- `startDate` (string): 开始日期 (YYYY-MM-DD)
- `endDate` (string): 结束日期 (YYYY-MM-DD)
- `appId` (string, optional): 应用ID
- `type` (string, optional): 验证码类型

**响应** (200 OK):
```json
{
  "success": true,
  "data": {
    "totalRequests": 1000000,
    "successfulVerifications": 950000,
    "failedVerifications": 50000,
    "passRate": 95.0,
    "avgResponseTime": 125.5,
    "byType": {
      "slider": {
        "total": 600000,
        "success": 570000,
        "passRate": 95.0
      },
      "click": {
        "total": 400000,
        "success": 380000,
        "passRate": 95.0
      }
    },
    "byDay": [
      {
        "date": "2026-05-20",
        "total": 50000,
        "success": 47500
      }
    ]
  }
}
```

### 2. 获取实时监控数据

**端点**: `GET /stats/realtime`

**响应** (200 OK):
```json
{
  "success": true,
  "data": {
    "currentQPS": 1250,
    "activeSessions": 5000,
    "queueLength": 100,
    "successRate": 94.5,
    "avgLatency": 85.2,
    "cpuUsage": 45.5,
    "memoryUsage": 62.3
  }
}
```

### 3. 获取行为分析数据

**端点**: `GET /stats/behavior`

**响应** (200 OK):
```json
{
  "success": true,
  "data": {
    "totalAnalyses": 500000,
    "botDetections": 15000,
    "humanInteractions": 485000,
    "avgBehaviorScore": 85.5,
    "topRiskFactors": [
      { "factor": "headless_browser", "count": 5000 },
      { "factor": "automation_framework", "count": 4000 },
      { "factor": "vpn_usage", "count": 3000 }
    ]
  }
}
```

---

## 应用管理接口

### 1. 获取应用列表

**端点**: `GET /admin/applications`

**请求头**:
```
Authorization: Bearer <token>
```

**查询参数**:
- `page` (int): 页码
- `pageSize` (int): 每页数量
- `status` (string, optional): 状态 (active, inactive, suspended)

**响应** (200 OK):
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": 1,
        "name": "My Application",
        "appId": "app_abc123",
        "appKey": "****",
        "status": "active",
        "createdAt": "2026-05-01T00:00:00Z",
        "quota": {
          "dailyLimit": 100000,
          "dailyUsed": 25000,
          "monthlyLimit": 3000000,
          "monthlyUsed": 750000
        }
      }
    ],
    "total": 100,
    "page": 1,
    "pageSize": 20
  }
}
```

### 2. 创建应用

**端点**: `POST /admin/applications`

**请求体**:
```json
{
  "name": "New Application",
  "description": "Description for the application",
  "type": "web",
  "quota": {
    "dailyLimit": 100000,
    "monthlyLimit": 3000000
  }
}
```

**响应** (201 Created):
```json
{
  "success": true,
  "data": {
    "id": 2,
    "name": "New Application",
    "appId": "app_new123",
    "appKey": "sk_live_abc123xyz789",
    "status": "active"
  }
}
```

### 3. 更新应用

**端点**: `PUT /admin/applications/{id}`

**请求体**:
```json
{
  "name": "Updated Application",
  "quota": {
    "dailyLimit": 200000
  }
}
```

### 4. 删除应用

**端点**: `DELETE /admin/applications/{id}`

**响应** (200 OK):
```json
{
  "success": true,
  "message": "Application deleted successfully"
}
```

---

## 租户管理接口

### 1. 获取租户列表

**端点**: `GET /admin/tenants`

**响应** (200 OK):
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": 1,
        "name": "Enterprise Tenant",
        "plan": "enterprise",
        "status": "active",
        "applications": 10,
        "users": 100,
        "quota": {
          "maxApps": 50,
          "maxUsers": 500,
          "maxQPS": 5000
        }
      }
    ]
  }
}
```

### 2. 创建租户

**端点**: `POST /admin/tenants`

**请求体**:
```json
{
  "name": "New Tenant",
  "plan": "professional",
  "adminEmail": "admin@tenant.com",
  "settings": {
    "maxApps": 20,
    "maxUsers": 100
  }
}
```

---

## 日志与审计接口

### 1. 获取验证日志

**端点**: `GET /logs/verification`

**查询参数**:
- `startDate` (string): 开始日期
- `endDate` (string): 结束日期
- `appId` (string, optional): 应用ID
- `status` (string, optional): 验证状态
- `page` (int): 页码
- `pageSize` (int): 每页数量

**响应** (200 OK):
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": 12345,
        "captchaId": "cap_abc123",
        "appId": "app_xyz",
        "type": "slider",
        "status": "success",
        "ip": "192.168.1.100",
        "userAgent": "Mozilla/5.0...",
        "responseTime": 125,
        "createdAt": "2026-05-20T12:30:00Z"
      }
    ],
    "total": 10000,
    "page": 1,
    "pageSize": 50
  }
}
```

### 2. 获取访问审计日志

**端点**: `GET /logs/audit`

**响应** (200 OK):
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": 1,
        "eventType": "login",
        "userId": 1,
        "username": "admin",
        "ip": "192.168.1.100",
        "status": "success",
        "riskScore": 15.5,
        "timestamp": "2026-05-20T12:00:00Z"
      }
    ]
  }
}
```

---

## 系统配置接口

### 1. 获取系统配置

**端点**: `GET /config/system`

**响应** (200 OK):
```json
{
  "success": true,
  "data": {
    "version": "19.0.0",
    "environment": "production",
    "features": {
      "aiDetection": true,
      "behaviorAnalysis": true,
      "biometrics": true,
      "quantumSafe": true
    },
    "security": {
      "requireMFA": true,
      "sessionTimeout": 3600,
      "maxLoginAttempts": 5
    }
  }
}
```

### 2. 更新系统配置

**端点**: `PUT /config/system`

**请求体**:
```json
{
  "security": {
    "requireMFA": false,
    "sessionTimeout": 7200
  }
}
```

---

## WebSocket 接口

### 实时验证

**端点**: `WS /ws/captcha`

**连接参数**:
```
ws://localhost:8080/ws/captcha?sessionId=xxx&appId=xxx
```

**消息格式** (客户端发送):
```json
{
  "type": "verify",
  "data": {
    "captchaId": "cap_abc123",
    "answer": { "x": 150 }
  }
}
```

**消息格式** (服务端返回):
```json
{
  "type": "verification_result",
  "data": {
    "verified": true,
    "score": 95.5
  }
}
```

---

## 错误码

| 错误码 | 描述 | HTTP 状态码 |
|--------|------|-------------|
| AUTH_FAILED | 认证失败 | 401 |
| AUTH_TOKEN_EXPIRED | Token 过期 | 401 |
| AUTH_PERMISSION_DENIED | 权限不足 | 403 |
| CAPTCHA_NOT_FOUND | 验证码不存在 | 404 |
| CAPTCHA_EXPIRED | 验证码已过期 | 400 |
| CAPTCHA_ALREADY_USED | 验证码已使用 | 400 |
| CAPTCHA_VERIFICATION_FAILED | 验证失败 | 400 |
| RATE_LIMIT_EXCEEDED | 请求频率超限 | 429 |
| INVALID_PARAMETER | 参数错误 | 400 |
| INTERNAL_ERROR | 服务器内部错误 | 500 |

---

## 认证与授权

### JWT Token

所有需要认证的接口都需要在请求头中包含 JWT Token:

```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### 角色权限

- **admin**: 管理员，拥有所有权限
- **operator**: 操作员，可以管理应用和查看统计
- **viewer**: 查看者，仅可查看数据
- **developer**: 开发者，可以测试验证码接口

### API Key

对于服务端集成，可以使用 API Key:

```
X-API-Key: sk_live_abc123xyz789
```

---

## 速率限制

| 端点类型 | 限制 |
|---------|------|
| 验证码生成 | 1000次/分钟 |
| 验证码验证 | 5000次/分钟 |
| 统计数据 | 100次/分钟 |
| 管理接口 | 50次/分钟 |

超过限制将返回 HTTP 429 状态码。

---

## SDK 示例

### JavaScript/TypeScript

```javascript
import HJTPX from '@hjtpx/sdk';

const client = new HJTPX({
  appId: 'your_app_id',
  apiKey: 'your_api_key',
  apiUrl: 'https://your-domain.com/api/v1'
});

// 生成验证码
const captcha = await client.captcha.generate('slider');
console.log(captcha.data.captchaId);

// 验证验证码
const result = await client.captcha.verify(captcha.data.captchaId, { x: 150 });
console.log(result.data.verified);
```

### Python

```python
from hjtpx import HJTPXClient

client = HJTPXClient(
    app_id='your_app_id',
    api_key='your_api_key',
    api_url='https://your-domain.com/api/v1'
)

# 生成验证码
captcha = client.captcha.generate('slider')
print(captcha['data']['captcha_id'])

# 验证验证码
result = client.captcha.verify(captcha['data']['captcha_id'], {'x': 150})
print(result['data']['verified'])
```

### Go

```go
package main

import (
    "github.com/hjtpx/hjtpx-go"
)

func main() {
    client := hjtpx.NewClient(
        hjtpx.WithAppID("your_app_id"),
        hjtpx.WithAPIKey("your_api_key"),
        hjtpx.WithAPIURL("https://your-domain.com/api/v1"),
    )

    captcha, err := client.Captcha.Generate("slider")
    if err != nil {
        panic(err)
    }

    result, err := client.Captcha.Verify(captcha.Data.CaptchaID, map[string]interface{}{
        "x": 150,
    })
    if err != nil {
        panic(err)
    }

    println(result.Data.Verified)
}
```

---

## 更新日志

### v4.0 (2026-05-20)

- 新增智能组合验证码
- 新增量子安全加密接口
- 新增联邦学习隐私保护
- 优化行为分析算法
- 新增 WebSocket 实时验证
- 完善 API 文档

### v3.0 (2026-04-15)

- 新增多因素认证
- 新增生物特征识别
- 新增零信任安全架构
- 优化 AI 检测引擎

### v2.0 (2026-03-01)

- 新增多种验证码类型
- 新增多租户支持
- 新增统计分析功能
- 优化性能

---

## 技术支持

如有问题，请联系:

- 邮箱: support@hjtpx.com
- 电话: 400-123-4567
- 文档: https://docs.hjtpx.com
