# API 完整文档 v2.0

---

## 目录

1. [概述](#概述)
2. [基础信息](#基础信息)
3. [认证方式](#认证方式)
4. [统一响应格式](#统一响应格式)
5. [错误码说明](#错误码说明)
6. [用户端API](#用户端api)
   - [验证码API](#验证码api)
   - [用户认证API](#用户认证api)
   - [用户资料API](#用户资料api)
   - [环境检测API](#环境检测api)
7. [管理端API](#管理端api)
   - [管理端认证API](#管理端认证api)
   - [管理端统计API](#管理端统计api)
   - [管理端应用管理API](#管理端应用管理api)
   - [管理端日志API](#管理端日志api)
   - [管理端黑名单API](#管理端黑名单api)
   - [管理端风控规则API](#管理端风控规则api)
   - [高级分析API](#高级分析api)
8. [SDK示例](#sdk示例)
   - [Go SDK](#go-sdk)
   - [JavaScript SDK](#javascript-sdk)
   - [Python SDK](#python-sdk)
9. [使用场景示例](#使用场景示例)
10. [安全建议](#安全建议)

---

## 概述

本文档描述了行为验证系统的所有API接口，包括用户端和管理端功能。该系统提供了多种验证码类型、用户认证、环境检测、风险分析等功能。

---

## 基础信息

### 基础URL

| 环境 | URL |
|------|-----|
| 生产环境 | https://api.example.com/api/v1 |
| 开发环境 | http://localhost:8080/api/v1 |

### 数据格式

所有请求和响应均使用JSON格式。

### 请求头

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| Content-Type | string | 是 | application/json |
| Authorization | string | 否 | Bearer {token}，需要认证的接口必填 |

---

## 认证方式

### 用户认证

用户认证使用JWT Token机制，通过登录接口获取。

### 管理员认证

管理员认证同样使用JWT Token机制，通过管理员登录接口获取。

### Token使用

在需要认证的接口请求头中添加：

```
Authorization: Bearer {your_token_here}
```

---

## 统一响应格式

### 成功响应

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

### 错误响应

```json
{
  "code": 400,
  "message": "参数错误",
  "data": null
}
```

### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| code | int | 状态码，0表示成功，非0表示错误 |
| message | string | 状态信息 |
| data | object/array | 响应数据 |

---

## 错误码说明

| 错误码 | 说明 | 处理建议 |
|--------|------|----------|
| 0 | 成功 | - |
| 400 | 请求参数错误 | 检查请求参数是否正确 |
| 401 | 未授权 | 请先登录获取Token |
| 403 | 权限不足 | 当前用户没有访问该接口的权限 |
| 404 | 资源不存在 | 检查请求的资源是否存在 |
| 429 | 请求过于频繁 | 请稍后重试 |
| 500 | 服务器内部错误 | 请联系技术支持 |

---

## 用户端API

### 验证码API

#### 获取滑块验证码

**接口地址**：`GET /api/v1/captcha/slider`

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| width | int | 否 | 验证码图片宽度，默认320 |
| height | int | 否 | 验证码图片高度，默认160 |
| tolerance | int | 否 | 容差值，默认8 |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "session_id": "slider_abc123def456",
    "image_url": "data:image/png;base64,...",
    "puzzle_url": "data:image/png;base64,...",
    "hint_url": "data:image/png;base64,...",
    "shape": 1,
    "secret_y": 100,
    "image_width": 320,
    "image_height": 160
  }
}
```

**响应字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| session_id | string | 验证码会话ID，验证时需要传入 |
| image_url | string | 背景图片Base64编码 |
| puzzle_url | string | 拼图图片Base64编码 |
| hint_url | string | 提示图片Base64编码 |
| shape | int | 拼图形状：0-正方形，1-圆形，2-三角形，3-菱形，4-六边形 |
| secret_y | int | 拼图的Y坐标 |
| image_width | int | 图片宽度 |
| image_height | int | 图片高度 |

---

#### 获取点击验证码

**接口地址**：`GET /api/v1/captcha/click`

**请求参数**：无

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "session_id": "click_xyz789",
    "image_url": "data:image/png;base64,...",
    "hint": "点击：1 → 3 → 2",
    "max_points": 3
  }
}
```

---

#### 验证验证码

**接口地址**：`POST /api/v1/captcha/verify`

**请求参数**：

```json
{
  "session_id": "slider_abc123def456",
  "x": 185,
  "y": 100,
  "trajectory": [
    {"x": 0, "y": 100, "t": 1620000000000},
    {"x": 50, "y": 105, "t": 1620000000050},
    {"x": 100, "y": 98, "t": 1620000000100},
    {"x": 150, "y": 102, "t": 1620000000150},
    {"x": 185, "y": 100, "t": 1620000000200}
  ]
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| session_id | string | 是 | 验证码会话ID |
| x | int | 是 | 拼图的X坐标 |
| y | int | 否 | 拼图的Y坐标，不填则使用secret_y |
| trajectory | array | 否 | 用户滑动轨迹，用于行为分析 |

**轨迹点说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| x | int | X坐标 |
| y | int | Y坐标 |
| t | int | 时间戳（毫秒） |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "success": true,
    "message": "验证成功",
    "remaining_attempts": 4,
    "trajectory_result": {
      "score": 85,
      "passed": true,
      "reasons": []
    }
  }
}
```

**失败响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "success": false,
    "message": "位置偏差较大，准确度约70%",
    "remaining_attempts": 3,
    "trajectory_result": {
      "score": 40,
      "passed": false,
      "reasons": ["轨迹点较少", "Y轴无变化，疑似机器操作"]
    }
  }
}
```

---

#### 获取手势验证码

**接口地址**：`GET /api/v1/captcha/gesture`

**请求参数**：无

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "session_id": "gesture_123abc",
    "pattern": "1→3→5→7→9",
    "grid_size": 3
  }
}
```

---

#### 验证手势验证码

**接口地址**：`POST /api/v1/captcha/gesture/verify`

**请求参数**：

```json
{
  "session_id": "gesture_123abc",
  "pattern": [1, 3, 5, 7, 9]
}
```

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "success": true,
    "message": "验证成功"
  }
}
```

---

### 用户认证API

#### 用户注册

**接口地址**：`POST /api/v1/auth/register`

**请求参数**：

```json
{
  "username": "testuser",
  "email": "test@example.com",
  "password": "password123",
  "behavior_data": "..."
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 用户名，3-50个字符 |
| email | string | 是 | 邮箱地址 |
| password | string | 是 | 密码，至少6个字符 |
| behavior_data | string | 否 | 行为数据，用于风险分析 |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "user_id": 1,
    "username": "testuser",
    "email": "test@example.com",
    "verification_link": "/api/v1/auth/verify-email?token=xxx",
    "message": "registration successful, please verify your email"
  }
}
```

**错误响应**：

```json
{
  "code": 409,
  "message": "username or email already exists",
  "data": null
}
```

---

#### 用户登录

**接口地址**：`POST /api/v1/auth/login`

**请求参数**：

```json
{
  "username": "testuser",
  "password": "password123",
  "captcha_token": "captcha_123",
  "behavior_data": "..."
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 用户名 |
| password | string | 是 | 密码 |
| captcha_token | string | 否 | 验证码Token |
| behavior_data | string | 否 | 行为数据 |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 900,
    "user": {
      "id": 1,
      "username": "testuser",
      "email": "test@example.com"
    }
  }
}
```

**说明**：
- `access_token`有效期为15分钟
- `refresh_token`有效期为7天

---

#### 刷新Token

**接口地址**：`POST /api/v1/auth/refresh`

**请求参数**：

```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 900
  }
}
```

---

#### 用户登出

**接口地址**：`POST /api/v1/auth/logout`

**认证要求**：需要用户Token

**请求参数**：无

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

---

#### 验证邮箱

**接口地址**：`GET /api/v1/auth/verify-email`

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| token | string | 是 | 验证Token |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "message": "email verified successfully"
  }
}
```

---

#### 重新发送验证邮件

**接口地址**：`POST /api/v1/auth/resend-verification`

**请求参数**：

```json
{
  "email": "test@example.com"
}
```

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "verification_link": "/api/v1/auth/verify-email?token=xxx",
    "message": "new verification link generated"
  }
}
```

---

#### 请求重置密码

**接口地址**：`POST /api/v1/auth/request-password-reset`

**请求参数**：

```json
{
  "email": "test@example.com"
}
```

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "reset_link": "/api/v1/auth/reset-password?token=xxx",
    "message": "password reset link generated"
  }
}
```

---

#### 重置密码

**接口地址**：`POST /api/v1/auth/reset-password`

**请求参数**：

```json
{
  "token": "reset_token_123",
  "new_password": "newpassword123"
}
```

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "message": "password reset successfully"
  }
}
```

---

### 用户资料API

#### 获取用户资料

**接口地址**：`GET /api/v1/user/profile`

**认证要求**：需要用户Token

**请求参数**：无

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "username": "testuser",
    "email": "test@example.com",
    "nickname": "测试用户",
    "avatar": "https://example.com/avatar.jpg",
    "phone": "13800138000",
    "bio": "这是一段个人简介",
    "is_verified": true,
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

---

#### 更新用户资料

**接口地址**：`PUT /api/v1/user/profile`

**认证要求**：需要用户Token

**请求参数**：

```json
{
  "nickname": "新昵称",
  "avatar": "https://example.com/new-avatar.jpg",
  "phone": "13900139000",
  "bio": "更新后的个人简介"
}
```

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "username": "testuser",
    "email": "test@example.com",
    "nickname": "新昵称",
    "avatar": "https://example.com/new-avatar.jpg",
    "phone": "13900139000",
    "bio": "更新后的个人简介",
    "is_verified": true,
    "updated_at": "2024-01-15T12:00:00Z"
  }
}
```

---

#### 修改密码

**接口地址**：`POST /api/v1/user/change-password`

**认证要求**：需要用户Token

**请求参数**：

```json
{
  "old_password": "password123",
  "new_password": "newpassword456"
}
```

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "message": "password changed successfully"
  }
}
```

---

### 环境检测API

#### 获取检测脚本

**接口地址**：`GET /api/v1/detect/script`

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| callback | string | 否 | 回调函数名，用于JSONP |

**响应**：JavaScript代码

**说明**：该接口返回一段JavaScript代码，用于在客户端进行环境检测，包括浏览器指纹、自动化检测、代理检测等。

---

#### 提交检测数据

**接口地址**：`POST /api/v1/detect/submit`

**请求参数**：

```json
{
  "detection_id": "abc123def456",
  "risk_score": 15.5,
  "chain": ["webgl", "canvas", "audio", "fonts", "webdriver"],
  "fingerprint": "base64_encoded_fingerprint",
  "session_id": "sess_1715000000000_12345",
  "timestamp": 1715000000000,
  "details": {
    "webgl": "WebGL Renderer",
    "canvas": "canvas_hash",
    "webdriver": "no_wd"
  }
}
```

**响应示例**：

```json
{
  "success": true,
  "risk_score": 25.5,
  "anomalies": 10
}
```

---

#### 环境检测

**接口地址**：`POST /api/v1/detect/check`

**请求参数**：

```json
{
  "fingerprint": "fingerprint_hash",
  "canvas_hash": "canvas_hash_value",
  "webgl_vendor": "NVIDIA Corporation",
  "webgl_renderer": "GeForce GTX 1080",
  "fonts": ["Arial", "Helvetica", "Times New Roman"],
  "plugins": ["Chrome PDF Plugin", "Flash"],
  "proxy_detected": false,
  "screen_info": {
    "width": 1920,
    "height": 1080,
    "color_depth": 24
  },
  "timezone": "Asia/Shanghai",
  "language": "zh-CN",
  "user_agent": "Mozilla/5.0 ...",
  "ip_address": "192.168.1.1"
}
```

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "is_bot": false,
    "risk_level": "low",
    "risk_score": 10.5,
    "detected_flags": [],
    "fingerprint": "fingerprint_hash",
    "is_unique": true
  }
}
```

**风险等级说明**：

| 等级 | 分数范围 | 说明 |
|------|----------|------|
| low | 0-30 | 低风险，正常用户 |
| medium | 31-60 | 中等风险，需要关注 |
| high | 61-100 | 高风险，可能是机器人 |

---

#### 获取指纹信息

**接口地址**：`GET /api/v1/detect/fingerprint`

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| fingerprint | string | 是 | 指纹数据 |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "success": true,
    "fingerprint": "md5_hash_of_fingerprint",
    "query": "original_fingerprint_data"
  }
}
```

---

#### 获取指纹统计

**接口地址**：`GET /api/v1/detect/stats`

**请求参数**：无

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "success": true,
    "data": {
      "total_count": 10000,
      "bot_count": 500,
      "proxy_count": 200,
      "average_risk_score": 15.5,
      "risk_distribution": {
        "low": 8500,
        "medium": 1200,
        "high": 300
      },
      "top_fingerprints": []
    }
  }
}
```

---

## 管理端API

### 管理端认证API

#### 管理员登录

**接口地址**：`POST /api/v1/admin/login`

**请求参数**：

```json
{
  "username": "admin",
  "password": "admin123"
}
```

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": 1,
      "username": "admin",
      "is_super_admin": true
    }
  }
}
```

---

#### 管理员登出

**接口地址**：`POST /api/v1/admin/logout`

**认证要求**：需要管理员Token

**请求参数**：无

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

---

### 管理端统计API

#### 获取仪表盘统计

**接口地址**：`GET /api/v1/admin/dashboard/stats`

**认证要求**：需要管理员Token

**请求参数**：无

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total_verifications": 100000,
    "success_rate": 85.5,
    "avg_response_time": 45,
    "active_users": 1000,
    "risk_distribution": {
      "low": 75000,
      "medium": 15000,
      "high": 8000,
      "critical": 2000
    }
  }
}
```

---

#### 获取最近活动

**接口地址**：`GET /api/v1/admin/dashboard/activity`

**认证要求**：需要管理员Token

**请求参数**：无

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "activities": [
      {
        "id": 1,
        "type": "verification",
        "description": "用户完成滑块验证",
        "timestamp": "2024-01-15T12:00:00Z",
        "ip": "192.168.1.1"
      }
    ],
    "total": 100
  }
}
```

---

#### 获取系统状态

**接口地址**：`GET /api/v1/admin/dashboard/system-status`

**认证要求**：需要管理员Token

**请求参数**：无

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "status": "healthy",
    "services": {
      "database": "up",
      "redis": "up",
      "api": "up"
    },
    "resources": {
      "cpu_usage": 45.5,
      "memory_usage": 60.2,
      "disk_usage": 30.1
    }
  }
}
```

---

#### 获取请求趋势

**接口地址**：`GET /api/v1/admin/dashboard/request-trend`

**认证要求**：需要管理员Token

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start_date | string | 否 | 开始日期，格式：YYYY-MM-DD |
| end_date | string | 否 | 结束日期，格式：YYYY-MM-DD |
| interval | string | 否 | 时间间隔：hour, day, week |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "trend": [
      {
        "timestamp": "2024-01-01T00:00:00Z",
        "requests": 1000,
        "success_rate": 85.0
      }
    ]
  }
}
```

---

#### 获取验证统计

**接口地址**：`GET /api/v1/admin/stats/verification`

**认证要求**：需要管理员Token

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start_date | string | 否 | 开始日期 |
| end_date | string | 否 | 结束日期 |
| app_id | int | 否 | 应用ID |

**响应示例**：

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
      "medium": 1200,
      "high": 1000,
      "critical": 300
    }
  }
}
```

---

#### 获取图表数据

**接口地址**：`GET /api/v1/admin/stats/chart`

**认证要求**：需要管理员Token

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| type | string | 是 | 图表类型：verification, risk, performance |
| period | string | 是 | 时间周期：today, week, month |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "labels": ["00:00", "01:00", "02:00"],
    "datasets": [
      {
        "label": "验证次数",
        "data": [100, 150, 120]
      }
    ]
  }
}
```

---

#### 获取趋势数据

**接口地址**：`GET /api/v1/admin/stats/trend`

**认证要求**：需要管理员Token

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| days | int | 否 | 天数，默认7天 |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "trend": [
      {
        "date": "2024-01-01",
        "total": 1000,
        "success": 850,
        "failed": 150
      }
    ]
  }
}
```

---

#### 获取小时统计

**接口地址**：`GET /api/v1/admin/stats/hourly`

**认证要求**：需要管理员Token

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| date | string | 否 | 日期，默认今天 |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "hourly": [
      {
        "hour": 0,
        "total": 100,
        "success": 85,
        "failed": 15
      }
    ]
  }
}
```

---

#### 获取实时统计

**接口地址**：`GET /api/v1/admin/stats/realtime`

**认证要求**：需要管理员Token

**请求参数**：无

**响应示例**：

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

---

#### 获取风险分布

**接口地址**：`GET /api/v1/admin/stats/risk-distribution`

**认证要求**：需要管理员Token

**请求参数**：无

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "distribution": {
      "low": 7500,
      "medium": 1200,
      "high": 800,
      "critical": 500
    },
    "total": 10000
  }
}
```

---

#### 获取TOP IP

**接口地址**：`GET /api/v1/admin/stats/top-ips`

**认证要求**：需要管理员Token

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| limit | int | 否 | 返回数量，默认10 |
| type | string | 否 | 类型：all, blocked, high_risk |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "top_ips": [
      {
        "ip": "192.168.1.1",
        "count": 1000,
        "risk_score": 85.5,
        "is_blocked": false
      }
    ],
    "total": 100
  }
}
```

---

#### 获取应用统计

**接口地址**：`GET /api/v1/admin/stats/application`

**认证要求**：需要管理员Token

**请求参数**：无

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "applications": [
      {
        "id": 1,
        "name": "测试应用",
        "total_verifications": 5000,
        "success_rate": 88.5,
        "avg_risk_score": 12.3
      }
    ],
    "total": 10
  }
}
```

---

#### 获取验证码类型统计

**接口地址**：`GET /api/v1/admin/stats/captcha-type`

**认证要求**：需要管理员Token

**请求参数**：无

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "types": {
      "slider": {
        "total": 6000,
        "success_rate": 85.0
      },
      "click": {
        "total": 3000,
        "success_rate": 82.0
      },
      "image": {
        "total": 1000,
        "success_rate": 90.0
      }
    }
  }
}
```

---

#### 生成报告

**接口地址**：`GET /api/v1/admin/stats/report`

**认证要求**：需要管理员Token

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start_date | string | 是 | 开始日期 |
| end_date | string | 是 | 结束日期 |
| format | string | 否 | 格式：json, csv, pdf |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "report_url": "https://example.com/reports/report_202401.pdf",
    "generated_at": "2024-01-15T12:00:00Z"
  }
}
```

---

### 管理端应用管理API

#### 获取应用列表

**接口地址**：`GET /api/v1/admin/applications`

**认证要求**：需要管理员Token

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认1 |
| page_size | int | 否 | 每页数量，默认20 |
| status | string | 否 | 状态：active, inactive |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": 1,
        "name": "测试应用",
        "app_key": "app_abc123",
        "status": "active",
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-15T12:00:00Z"
      }
    ],
    "total": 10,
    "page": 1,
    "page_size": 20
  }
}
```

---

#### 创建应用

**接口地址**：`POST /api/v1/admin/applications`

**认证要求**：需要管理员Token

**请求参数**：

```json
{
  "name": "新应用",
  "description": "这是一个新应用的描述"
}
```

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 2,
    "name": "新应用",
    "description": "这是一个新应用的描述",
    "app_key": "app_xyz789",
    "app_secret": "secret_123456",
    "status": "active",
    "created_at": "2024-01-15T12:00:00Z"
  }
}
```

**重要提示**：`app_secret`只在创建时显示一次，请妥善保存。

---

#### 获取应用详情

**接口地址**：`GET /api/v1/admin/applications/:id`

**认证要求**：需要管理员Token

**路径参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 应用ID |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "name": "测试应用",
    "description": "应用描述",
    "app_key": "app_abc123",
    "status": "active",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-15T12:00:00Z",
    "stats": {
      "total_verifications": 5000,
      "success_rate": 88.5
    }
  }
}
```

---

#### 更新应用

**接口地址**：`PUT /api/v1/admin/applications/:id`

**认证要求**：需要管理员Token

**路径参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 应用ID |

**请求参数**：

```json
{
  "name": "更新后的应用名",
  "description": "更新后的描述",
  "status": "inactive"
}
```

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "name": "更新后的应用名",
    "description": "更新后的描述",
    "status": "inactive",
    "updated_at": "2024-01-15T12:00:00Z"
  }
}
```

---

#### 删除应用

**接口地址**：`DELETE /api/v1/admin/applications/:id`

**认证要求**：需要管理员Token

**路径参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 应用ID |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

---

#### 重新生成应用密钥

**接口地址**：`POST /api/v1/admin/applications/:id/regenerate-key`

**认证要求**：需要管理员Token

**路径参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 应用ID |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "app_key": "app_newkey123",
    "app_secret": "secret_newsecret456"
  }
}
```

---

#### 获取应用配置

**接口地址**：`GET /api/v1/admin/applications/:id/config`

**认证要求**：需要管理员Token

**路径参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 应用ID |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "captcha_types": ["slider", "click"],
    "risk_threshold": 50,
    "max_attempts": 5,
    "session_timeout": 300,
    "enable_behavior_analysis": true,
    "enable_ip_blacklist": true
  }
}
```

---

#### 更新应用配置

**接口地址**：`PUT /api/v1/admin/applications/:id/config`

**认证要求**：需要管理员Token

**路径参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 应用ID |

**请求参数**：

```json
{
  "captcha_types": ["slider", "click", "image"],
  "risk_threshold": 60,
  "max_attempts": 3,
  "session_timeout": 600,
  "enable_behavior_analysis": true,
  "enable_ip_blacklist": true
}
```

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "captcha_types": ["slider", "click", "image"],
    "risk_threshold": 60,
    "max_attempts": 3,
    "session_timeout": 600,
    "enable_behavior_analysis": true,
    "enable_ip_blacklist": true,
    "updated_at": "2024-01-15T12:00:00Z"
  }
}
```

---

#### 获取应用统计

**接口地址**：`GET /api/v1/admin/applications/:id/statistics`

**认证要求**：需要管理员Token

**路径参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 应用ID |

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start_date | string | 否 | 开始日期 |
| end_date | string | 否 | 结束日期 |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total_verifications": 5000,
    "success": 4250,
    "failed": 750,
    "success_rate": 85.0,
    "avg_risk_score": 15.5,
    "daily_trend": []
  }
}
```

---

#### 获取应用摘要

**接口地址**：`GET /api/v1/admin/applications/summary`

**认证要求**：需要管理员Token

**请求参数**：无

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total_applications": 10,
    "active_applications": 8,
    "total_verifications": 100000,
    "avg_success_rate": 85.5
  }
}
```

---

### 管理端日志API

#### 获取验证日志

**接口地址**：`GET /api/v1/admin/logs`

**认证要求**：需要管理员Token

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认1 |
| page_size | int | 否 | 每页数量，默认20 |
| app_id | int | 否 | 应用ID |
| type | string | 否 | 验证码类型：slider, click, image |
| status | string | 否 | 状态：success, failed |
| start_date | string | 否 | 开始日期 |
| end_date | string | 否 | 结束日期 |
| ip | string | 否 | IP地址 |
| risk_level | string | 否 | 风险等级：low, medium, high, critical |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": 1,
        "session_id": "sess_abc123",
        "captcha_type": "slider",
        "status": "success",
        "risk_score": 15.5,
        "ip_address": "192.168.1.1",
        "user_agent": "Mozilla/5.0 ...",
        "created_at": "2024-01-15T12:00:00Z"
      }
    ],
    "total": 1000,
    "page": 1,
    "page_size": 20
  }
}
```

---

#### 获取日志详情

**接口地址**：`GET /api/v1/admin/logs/:id`

**认证要求**：需要管理员Token

**路径参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 日志ID |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "session_id": "sess_abc123",
    "captcha_type": "slider",
    "status": "success",
    "risk_score": 15.5,
    "risk_level": "low",
    "ip_address": "192.168.1.1",
    "user_agent": "Mozilla/5.0 ...",
    "fingerprint": "fingerprint_hash",
    "trajectory_data": {},
    "behavior_data": {},
    "created_at": "2024-01-15T12:00:00Z"
  }
}
```

---

#### 根据会话ID获取日志

**接口地址**：`GET /api/v1/admin/logs/session/:session_id`

**认证要求**：需要管理员Token

**路径参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| session_id | string | 是 | 会话ID |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "logs": []
  }
}
```

---

#### 获取日志统计

**接口地址**：`GET /api/v1/admin/logs/statistics`

**认证要求**：需要管理员Token

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start_date | string | 否 | 开始日期 |
| end_date | string | 否 | 结束日期 |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 10000,
    "success": 8500,
    "failed": 1500,
    "risk_distribution": {
      "low": 7500,
      "medium": 1500,
      "high": 800,
      "critical": 200
    },
    "top_ips": [],
    "avg_response_time": 45
  }
}
```

---

#### 获取日志摘要

**接口地址**：`GET /api/v1/admin/logs/summary`

**认证要求**：需要管理员Token

**请求参数**：无

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "today_total": 5000,
    "today_success": 4250,
    "today_failed": 750,
    "active_sessions": 100
  }
}
```

---

#### 导出日志

**接口地址**：`GET /api/v1/admin/logs/export`

**认证要求**：需要管理员Token

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start_date | string | 是 | 开始日期 |
| end_date | string | 是 | 结束日期 |
| format | string | 否 | 格式：csv, json |

**响应**：文件下载

---

#### 清理旧日志

**接口地址**：`DELETE /api/v1/admin/logs/cleanup`

**认证要求**：需要管理员Token

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| days | int | 否 | 保留天数，默认30天 |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "deleted_count": 10000
  }
}
```

---

#### 清空日志

**接口地址**：`POST /api/v1/admin/logs/clear`

**认证要求**：需要管理员Token

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| confirm | boolean | 是 | 确认清空 |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "deleted_count": 100000
  }
}
```

---

### 管理端黑名单API

#### 获取黑名单列表

**接口地址**：`GET /api/v1/admin/blacklist`

**认证要求**：需要管理员Token

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认1 |
| page_size | int | 否 | 每页数量，默认20 |
| type | string | 否 | 类型：ip, fingerprint, user_id |
| status | string | 否 | 状态：active, expired |

**响应示例**：

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
        "created_by": "admin",
        "created_at": "2024-01-01T00:00:00Z",
        "expires_at": "2024-02-01T00:00:00Z",
        "status": "active"
      }
    ],
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

---

#### 添加黑名单

**接口地址**：`POST /api/v1/admin/blacklist`

**认证要求**：需要管理员Token

**请求参数**：

```json
{
  "type": "ip",
  "value": "192.168.1.100",
  "reason": "恶意攻击",
  "duration": 86400
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| type | string | 是 | 类型：ip, fingerprint, user_id |
| value | string | 是 | 值 |
| reason | string | 否 | 原因 |
| duration | int | 否 | 持续时间（秒），0表示永久 |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 2,
    "type": "ip",
    "value": "192.168.1.100",
    "reason": "恶意攻击",
    "created_at": "2024-01-15T12:00:00Z",
    "expires_at": "2024-01-16T12:00:00Z",
    "status": "active"
  }
}
```

---

#### 获取黑名单详情

**接口地址**：`GET /api/v1/admin/blacklist/:id`

**认证要求**：需要管理员Token

**路径参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 黑名单ID |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "type": "ip",
    "value": "192.168.1.100",
    "reason": "恶意攻击",
    "created_by": "admin",
    "created_at": "2024-01-01T00:00:00Z",
    "expires_at": "2024-02-01T00:00:00Z",
    "status": "active",
    "stats": {
      "blocked_count": 100,
      "last_blocked_at": "2024-01-15T12:00:00Z"
    }
  }
}
```

---

#### 更新黑名单

**接口地址**：`PUT /api/v1/admin/blacklist/:id`

**认证要求**：需要管理员Token

**路径参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 黑名单ID |

**请求参数**：

```json
{
  "reason": "更新后的原因",
  "expires_at": "2024-03-01T00:00:00Z"
}
```

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "reason": "更新后的原因",
    "expires_at": "2024-03-01T00:00:00Z",
    "updated_at": "2024-01-15T12:00:00Z"
  }
}
```

---

#### 删除黑名单

**接口地址**：`DELETE /api/v1/admin/blacklist/:id`

**认证要求**：需要管理员Token

**路径参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 黑名单ID |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

---

#### 解除黑名单

**接口地址**：`POST /api/v1/admin/blacklist/:id/unblock`

**认证要求**：需要管理员Token

**路径参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 黑名单ID |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "status": "expired",
    "unblocked_at": "2024-01-15T12:00:00Z"
  }
}
```

---

#### 导入黑名单

**接口地址**：`POST /api/v1/admin/blacklist/import`

**认证要求**：需要管理员Token

**请求参数**：

```json
{
  "items": [
    {
      "type": "ip",
      "value": "192.168.1.101",
      "reason": "批量导入",
      "duration": 86400
    }
  ]
}
```

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "imported": 1,
    "failed": 0,
    "errors": []
  }
}
```

---

#### 获取黑名单摘要

**接口地址**：`GET /api/v1/admin/blacklist/summary`

**认证要求**：需要管理员Token

**请求参数**：无

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 100,
    "active": 80,
    "expired": 20,
    "by_type": {
      "ip": 60,
      "fingerprint": 30,
      "user_id": 10
    },
    "today_blocked": 50
  }
}
```

---

### 管理端风控规则API

#### 获取风控规则列表

**接口地址**：`GET /api/v1/admin/risk-rules`

**认证要求**：需要管理员Token

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认1 |
| page_size | int | 否 | 每页数量，默认20 |
| enabled | boolean | 否 | 是否启用 |
| category | string | 否 | 分类：behavior, device, network |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": 1,
        "name": "轨迹异常检测",
        "description": "检测用户滑动轨迹是否异常",
        "category": "behavior",
        "enabled": true,
        "risk_score": 30,
        "conditions": {},
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-15T12:00:00Z"
      }
    ],
    "total": 50,
    "page": 1,
    "page_size": 20
  }
}
```

---

#### 创建风控规则

**接口地址**：`POST /api/v1/admin/risk-rules`

**认证要求**：需要管理员Token

**请求参数**：

```json
{
  "name": "新规则",
  "description": "规则描述",
  "category": "behavior",
  "risk_score": 20,
  "conditions": {
    "type": "and",
    "rules": []
  },
  "enabled": true
}
```

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 2,
    "name": "新规则",
    "description": "规则描述",
    "category": "behavior",
    "risk_score": 20,
    "conditions": {},
    "enabled": true,
    "created_at": "2024-01-15T12:00:00Z"
  }
}
```

---

#### 获取风控规则详情

**接口地址**：`GET /api/v1/admin/risk-rules/:id`

**认证要求**：需要管理员Token

**路径参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 规则ID |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "name": "轨迹异常检测",
    "description": "检测用户滑动轨迹是否异常",
    "category": "behavior",
    "risk_score": 30,
    "conditions": {},
    "enabled": true,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-15T12:00:00Z",
    "stats": {
      "triggered_count": 1000,
      "last_triggered_at": "2024-01-15T12:00:00Z"
    }
  }
}
```

---

#### 更新风控规则

**接口地址**：`PUT /api/v1/admin/risk-rules/:id`

**认证要求**：需要管理员Token

**路径参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 规则ID |

**请求参数**：

```json
{
  "name": "更新后的规则名",
  "description": "更新后的描述",
  "risk_score": 25,
  "conditions": {},
  "enabled": true
}
```

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "name": "更新后的规则名",
    "description": "更新后的描述",
    "risk_score": 25,
    "conditions": {},
    "enabled": true,
    "updated_at": "2024-01-15T12:00:00Z"
  }
}
```

---

#### 删除风控规则

**接口地址**：`DELETE /api/v1/admin/risk-rules/:id`

**认证要求**：需要管理员Token

**路径参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 规则ID |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

---

#### 切换风控规则状态

**接口地址**：`POST /api/v1/admin/risk-rules/:id/toggle`

**认证要求**：需要管理员Token

**路径参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 规则ID |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "enabled": false,
    "updated_at": "2024-01-15T12:00:00Z"
  }
}
```

---

#### 获取风控规则摘要

**接口地址**：`GET /api/v1/admin/risk-rules/summary`

**认证要求**：需要管理员Token

**请求参数**：无

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 50,
    "enabled": 40,
    "disabled": 10,
    "by_category": {
      "behavior": 20,
      "device": 15,
      "network": 15
    },
    "total_triggered_today": 500
  }
}
```

---

### 高级分析API

#### 获取用户行为分析

**接口地址**：`GET /api/v1/admin/analytics/user-behavior`

**认证要求**：需要管理员Token

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start_date | string | 否 | 开始日期 |
| end_date | string | 否 | 结束日期 |
| user_id | int | 否 | 用户ID |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "user_behavior": {
      "total_verifications": 100,
      "success_rate": 85.0,
      "avg_risk_score": 12.5,
      "behavior_patterns": [],
      "anomaly_count": 5
    }
  }
}
```

---

#### 获取攻击趋势分析

**接口地址**：`GET /api/v1/admin/analytics/attack-trend`

**认证要求**：需要管理员Token

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start_date | string | 否 | 开始日期 |
| end_date | string | 否 | 结束日期 |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "trend": [
      {
        "date": "2024-01-01",
        "attack_count": 10,
        "attack_types": {
          "automation": 5,
          "proxy": 3,
          "brute_force": 2
        }
      }
    ],
    "summary": {
      "total_attacks": 100,
      "blocked_attacks": 90,
      "success_rate": 90.0
    }
  }
}
```

---

#### 生成风险报告

**接口地址**：`POST /api/v1/admin/analytics/generate-report`

**认证要求**：需要管理员Token

**请求参数**：

```json
{
  "start_date": "2024-01-01",
  "end_date": "2024-01-15",
  "report_type": "risk",
  "format": "pdf"
}
```

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "report_id": "report_123",
    "status": "generating",
    "created_at": "2024-01-15T12:00:00Z"
  }
}
```

---

#### 获取可视化数据

**接口地址**：`GET /api/v1/admin/analytics/visualization`

**认证要求**：需要管理员Token

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| type | string | 否 | 可视化类型：heatmap, scatter, geographic |
| start_date | string | 否 | 开始日期 |
| end_date | string | 否 | 结束日期 |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "heatmap": [...],
    "scatter": [...],
    "geographic": [...]
  }
}
```

---

#### 获取热力图数据

**接口地址**：`GET /api/v1/admin/analytics/heatmap`

**认证要求**：需要管理员Token

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start_date | string | 否 | 开始日期 |
| end_date | string | 否 | 结束日期 |
| granularity | string | 否 | 粒度：hour, day, week |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "data": [
      {"x": "2024-01-01 00:00", "y": "2024-01-01 00:00", "value": 100},
      {"x": "2024-01-01 00:00", "y": "2024-01-01 01:00", "value": 150}
    ]
  }
}
```

---

#### 获取地理分布数据

**接口地址**：`GET /api/v1/admin/analytics/geographic`

**认证要求**：需要管理员Token

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start_date | string | 否 | 开始日期 |
| end_date | string | 否 | 结束日期 |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "countries": [
      {"code": "CN", "name": "中国", "count": 10000},
      {"code": "US", "name": "美国", "count": 5000}
    ],
    "regions": [
      {"country": "CN", "region": "北京", "count": 3000},
      {"country": "CN", "region": "上海", "count": 2500}
    ]
  }
}
```

---

#### 获取漏斗分析数据

**接口地址**：`GET /api/v1/admin/analytics/funnel`

**认证要求**：需要管理员Token

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| funnel_type | string | 是 | 漏斗类型：verification, conversion |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "steps": [
      {"name": "页面访问", "count": 10000, "rate": 100},
      {"name": "验证码展示", "count": 9000, "rate": 90},
      {"name": "验证完成", "count": 8500, "rate": 85},
      {"name": "验证成功", "count": 8200, "rate": 82}
    ],
    "total_conversion": 82
  }
}
```

---

### 管理端通知API

#### 获取通知列表

**接口地址**：`GET /api/v1/admin/notifications`

**认证要求**：需要管理员Token

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认1 |
| page_size | int | 否 | 每页数量，默认20 |
| type | string | 否 | 通知类型：alert, system, info |
| is_read | boolean | 否 | 是否已读 |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": 1,
        "type": "alert",
        "title": "高风险攻击告警",
        "content": "检测到来自 192.168.1.100 的异常请求",
        "is_read": false,
        "created_at": "2024-01-15T12:00:00Z"
      }
    ],
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

---

#### 标记通知已读

**接口地址**：`PUT /api/v1/admin/notifications/:id/read`

**认证要求**：需要管理员Token

**路径参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 通知ID |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "is_read": true,
    "read_at": "2024-01-15T12:30:00Z"
  }
}
```

---

#### 标记全部已读

**接口地址**：`PUT /api/v1/admin/notifications/read-all`

**认证要求**：需要管理员Token

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "updated_count": 50
  }
}
```

---

#### 删除通知

**接口地址**：`DELETE /api/v1/admin/notifications/:id`

**认证要求**：需要管理员Token

**路径参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 通知ID |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

---

### 管理端CSS切换API

#### 获取CSS配置

**接口地址**：`GET /api/v1/admin/css-config`

**认证要求**：需要管理员Token

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "current_css": "default",
    "available_css": ["default", "dark", "light", "custom"],
    "custom_css_url": ""
  }
}
```

---

#### 更新CSS配置

**接口地址**：`PUT /api/v1/admin/css-config`

**认证要求**：需要管理员Token

**请求参数**：

```json
{
  "current_css": "dark",
  "custom_css_url": ""
}
```

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "current_css": "dark",
    "updated_at": "2024-01-15T12:00:00Z"
  }
}
```

---

### 管理端实时监控WebSocket API

#### 连接实时监控

**接口地址**：`WS /api/v1/admin/monitoring/ws`

**认证要求**：需要管理员Token

**连接示例**：

```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/admin/monitoring/ws?token=xxx');

ws.onmessage = function(event) {
  const data = JSON.parse(event.data);
  if (data.type === 'metrics') {
    console.log('Current metrics:', data.payload);
  }
};

ws.onclose = function() {
  console.log('Connection closed');
};
```

**消息类型**：

| 类型 | 说明 | 频率 |
|------|------|------|
| metrics | 实时指标 | 每秒 |
| alert | 告警通知 | 实时 |
| stats | 统计更新 | 每5秒 |

---

### 管理端高级分析API

#### 获取行为分析数据

**接口地址**：`GET /api/v1/admin/behavior-analysis`

**认证要求**：需要管理员Token

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start_date | string | 否 | 开始日期 |
| end_date | string | 否 | 结束日期 |
| group_by | string | 否 | 分组：hour, day, week, month |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "trajectory_analysis": {
      "average_speed": 150.5,
      "average_acceleration": 2.3,
      "suspicious_rate": 5.2
    },
    "click_pattern": {
      "average_click_interval": 500,
      "average_click_count": 5
    },
    "mouse_movement": {
      "smoothness": 85.5,
      "straightness": 0.7
    }
  }
}
```

---

#### 获取批量操作结果

**接口地址**：`GET /api/v1/admin/batch-operations`

**认证要求**：需要管理员Token

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认1 |
| page_size | int | 否 | 每页数量，默认20 |
| type | string | 否 | 操作类型 |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": 1,
        "type": "bulk_blacklist",
        "status": "completed",
        "total": 100,
        "success": 95,
        "failed": 5,
        "created_at": "2024-01-15T12:00:00Z",
        "completed_at": "2024-01-15T12:05:00Z"
      }
    ],
    "total": 10,
    "page": 1,
    "page_size": 20
  }
}
```

---

#### 导出数据

**接口地址**：`POST /api/v1/admin/export`

**认证要求**：需要管理员Token

**请求参数**：

```json
{
  "type": "logs",
  "format": "csv",
  "filters": {
    "start_date": "2024-01-01",
    "end_date": "2024-01-15",
    "captcha_type": "slider"
  }
}
```

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "task_id": "export_123",
    "status": "processing",
    "estimated_time": 60
  }
}
```

---

#### 获取导出任务状态

**接口地址**：`GET /api/v1/admin/export/:task_id`

**认证要求**：需要管理员Token

**路径参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| task_id | string | 是 | 任务ID |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "task_id": "export_123",
    "status": "completed",
    "download_url": "/api/v1/admin/export/export_123/download",
    "expires_at": "2024-01-15T13:00:00Z"
  }
}
```

---

#### 下载导出文件

**接口地址**：`GET /api/v1/admin/export/:task_id/download`

**认证要求**：需要管理员Token

**响应**：文件下载

---

## SDK示例

### Go SDK