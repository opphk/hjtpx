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

**请求参数
---

## AI风控引擎API (v15.0)

### 概述

AI风控引擎API提供了基于深度强化学习的智能风控策略，包括多维度风险画像、实时风险评估、策略热更新和效果监控等功能。

---

### 实时风险评估API

#### 提交风险评估请求

**接口地址**：`POST /api/v1/risk/assess`

**认证要求**：无需认证

**请求参数**：

```json
{
  "fingerprint": "string",           // 设备指纹 [必需]
  "ip_address": "string",            // IP地址 [必需]
  "session_id": "string",            // 会话ID
  "device_info": {                   // 设备信息
    "user_agent": "string",
    "screen_resolution": "string",
    "timezone": "string",
    "language": "string",
    "is_mobile": boolean,
    "is_bot": boolean,
    "is_headless": boolean
  },
  "behavior_data": {                // 行为数据
    "score": number,
    "mouse_speed": number,
    "click_frequency": number,
    "path_efficiency": number
  },
  "geo_data": {                     // 地理位置数据
    "current_country": "string",
    "current_region": "string",
    "current_city": "string"
  },
  "metadata": {}                    // 其他元数据
}
```

**响应示例**：

```json
{
  "code": 0,
  "message": "风险评估成功",
  "data": {
    "request_id": "risk_1716054400000000000",
    "risk_score": 75.5,
    "risk_level": "medium",
    "action": "captcha",
    "factors": ["建议进行验证码挑战"],
    "device_score": 85.0,
    "ip_score": 78.0,
    "behavior_score": 72.0,
    "geo_score": 90.0,
    "historical_score": 80.0,
    "time_score": 100.0,
    "session_score": 88.0,
    "confidence": 0.95,
    "processing_time_ms": 8,
    "recommendations": ["增加设备验证", "监控IP行为"]
  }
}
```

**响应字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| request_id | string | 请求唯一标识 |
| risk_score | float | 综合风险评分 (0-100) |
| risk_level | string | 风险等级: low/medium/high/critical |
| action | string | 建议动作: allow/captcha/review/block/challenge |
| factors | array | 触发风险的因素列表 |
| device_score | float | 设备风险评分 |
| ip_score | float | IP风险评分 |
| behavior_score | float | 行为风险评分 |
| geo_score | float | 地理位置风险评分 |
| historical_score | float | 历史行为评分 |
| time_score | float | 时间段风险评分 |
| session_score | float | 会话风险评分 |
| confidence | float | 评估置信度 (0-1) |
| processing_time_ms | int | 处理时间(毫秒) |

---

### 风险画像API

#### 获取风险画像

**接口地址**：`GET /api/v1/risk/profile`

**认证要求**：无需认证

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| fingerprint | string | 是 | 设备指纹 |
| ip_address | string | 是 | IP地址 |
| type | string | 否 | 画像类型: unified/device/ip/behavior/geo (默认unified) |
| session_id | string | 否 | 会话ID (查询行为画像时必需) |

**响应示例**：

```json
{
  "code": 0,
  "message": "查询成功",
  "data": {
    "profile": {
      "id": 1,
      "fingerprint": "fp_xxx",
      "overall_risk_score": 78.5,
      "risk_level": "medium",
      "trust_level": 75,
      "request_count": 150,
      "block_count": 3,
      "first_seen_at": "2024-01-01T00:00:00Z",
      "last_seen_at": "2024-05-15T12:00:00Z"
    },
    "device_history": [],
    "ip_history": []
  }
}
```

#### 获取风险画像分析

**接口地址**：`GET /api/v1/risk/profile/analysis`

**认证要求**：无需认证

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| fingerprint | string | 是 | 设备指纹 |
| ip_address | string | 是 | IP地址 |

**响应示例**：

```json
{
  "code": 0,
  "message": "分析成功",
  "data": {
    "device_score": 85.0,
    "ip_score": 78.0,
    "device_risk_factors": ["检测到自动化框架特征"],
    "ip_risk_factors": ["24小时内被拦截2次"],
    "device_history": [],
    "ip_history": []
  }
}
```

#### 更新设备画像

**接口地址**：`PUT /api/v1/risk/profile/device`

**认证要求**：无需认证

**请求参数**：

```json
{
  "fingerprint": "string",
  "device_info": {
    "user_agent": "string",
    "screen_resolution": "string",
    "touch_points": 0,
    "has_web_socket": true
  }
}
```

**响应示例**：

```json
{
  "code": 0,
  "message": "设备画像更新成功",
  "data": {
    "id": 1,
    "fingerprint": "fp_xxx",
    "risk_score": 82.5,
    "trust_level": 80
  }
}
```

---

### 风控策略管理API

#### 获取当前策略版本

**接口地址**：`GET /api/v1/risk/strategy/version`

**认证要求**：无需认证

**响应示例**：

```json
{
  "code": 0,
  "message": "查询成功",
  "data": {
    "version": {
      "id": 1,
      "version": "v1.0.0",
      "strategy_type": "default",
      "is_active": true,
      "published_at": "2024-01-01T00:00:00Z"
    },
    "rules": [
      {
        "id": 1,
        "name": "IP频率限制",
        "rule_type": "rate_limit",
        "action": "captcha",
        "priority": 100,
        "weight": 0.15,
        "enabled": true
      }
    ]
  }
}
```

#### 获取策略版本历史

**接口地址**：`GET /api/v1/risk/strategy/versions`

**认证要求**：无需认证

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| limit | int | 否 | 返回数量限制 (默认20) |

#### 创建新策略版本

**接口地址**：`POST /api/v1/risk/strategy/version`

**认证要求**：无需认证

**请求参数**：

```json
{
  "base_version": "v1.0.0",
  "new_version": "v1.1.0",
  "description": "优化风控规则"
}
```

#### 更新策略规则

**接口地址**：`PUT /api/v1/risk/strategy/rule/:id`

**认证要求**：无需认证

**请求参数**：

```json
{
  "name": "新规则名称",
  "condition": "ip_request_count > threshold",
  "action": "block",
  "parameters": {"threshold": 100},
  "priority": 90,
  "weight": 0.2,
  "enabled": true
}
```

#### 发布策略版本

**接口地址**：`POST /api/v1/risk/strategy/version/:id/publish`

**认证要求**：无需认证

#### 回滚策略版本

**接口地址**：`POST /api/v1/risk/strategy/version/:id/rollback`

**认证要求**：无需认证

#### 获取版本更新历史

**接口地址**：`GET /api/v1/risk/strategy/version/:id/updates`

**认证要求**：无需认证

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| limit | int | 否 | 返回数量限制 (默认50) |

#### 导出策略版本

**接口地址**：`GET /api/v1/risk/strategy/version/:id/export`

**认证要求**：无需认证

**响应**：JSON格式的策略配置文件

#### 导入策略版本

**接口地址**：`POST /api/v1/risk/strategy/version/import`

**认证要求**：无需认证

**请求参数**：

```json
{
  "json_data": "{...}"  // 导出的策略JSON
}
```

#### 评估风险规则

**接口地址**：`POST /api/v1/risk/strategy/evaluate`

**认证要求**：无需认证

**请求参数**：

```json
{
  "risk_context": {
    "ip_request_count": 150,
    "mouse_speed": 2500,
    "is_vpn": true
  }
}
```

**响应示例**：

```json
{
  "code": 0,
  "message": "评估成功",
  "data": {
    "action": "captcha",
    "risk_score": 45.5,
    "triggered_rules": ["IP频率限制", "VPN/代理检测"]
  }
}
```

---

### 风控监控API

#### 获取监控指标

**接口地址**：`GET /api/v1/risk/monitoring/metrics`

**认证要求**：无需认证

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| type | string | 否 | 指标类型: risk/performance/block (默认risk) |
| range | string | 否 | 时间范围: 1h/6h/24h/7d (默认1h) |

**响应示例**：

```json
{
  "code": 0,
  "message": "查询成功",
  "data": {
    "risk_score_avg": 72.5,
    "risk_score_max": 95.0,
    "risk_score_min": 30.0
  }
}
```

#### 获取风险指标

**接口地址**：`GET /api/v1/risk/monitoring/risk-metrics`

**认证要求**：无需认证

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start_time | string | 否 | 开始时间 (RFC3339格式) |
| end_time | string | 否 | 结束时间 (RFC3339格式) |

**响应示例**：

```json
{
  "code": 0,
  "message": "查询成功",
  "data": {
    "metrics": {
      "avg_risk_score": 68.5,
      "total_requests": 50000,
      "blocked_requests": 1500,
      "block_rate": 0.03,
      "false_positives": 45,
      "false_positive_rate": 0.03,
      "avg_response_time_ms": 8.5
    },
    "risk_distribution": {
      "low": 35000,
      "medium": 10000,
      "high": 4000,
      "critical": 1000
    },
    "action_distribution": {
      "allow": 40000,
      "captcha": 8000,
      "review": 1500,
      "block": 500
    },
    "period": {
      "start": "2024-05-14T00:00:00Z",
      "end": "2024-05-15T00:00:00Z"
    }
  }
}
```

#### 获取策略性能

**接口地址**：`GET /api/v1/risk/monitoring/strategy-performance`

**认证要求**：无需认证

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| strategy_name | string | 否 | 策略名称 |
| start_time | string | 否 | 开始时间 |
| end_time | string | 否 | 结束时间 |

**响应示例**：

```json
{
  "code": 0,
  "message": "查询成功",
  "data": {
    "effectiveness": 0.85,
    "false_positive_rate": 0.02,
    "total_hits": 1500,
    "trends": [
      {
        "timestamp": "2024-05-15T00:00:00Z",
        "value": 0.82,
        "unit": "rate"
      }
    ]
  }
}
```

#### 获取模型性能

**接口地址**：`GET /api/v1/risk/monitoring/model-performance`

**认证要求**：无需认证

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| model_type | string | 否 | 模型类型 (默认drp) |
| start_time | string | 否 | 开始时间 |
| end_time | string | 否 | 结束时间 |

**响应示例**：

```json
{
  "code": 0,
  "message": "查询成功",
  "data": {
    "accuracy": 0.92,
    "precision": 0.88,
    "recall": 0.95,
    "f1_score": 0.91,
    "avg_latency_ms": 5.2,
    "accuracy_trend": []
  }
}
```

#### 获取活跃告警

**接口地址**：`GET /api/v1/risk/monitoring/alerts`

**认证要求**：无需认证

**响应示例**：

```json
{
  "code": 0,
  "message": "查询成功",
  "data": [
    {
      "id": 1,
      "alert_type": "threshold",
      "alert_name": "risk_score告警",
      "severity": "medium",
      "message": "risk_score指标低于阈值: 当前值35.0, 阈值40.0",
      "status": "active",
      "created_at": "2024-05-15T10:00:00Z"
    }
  ]
}
```

#### 确认告警

**接口地址**：`POST /api/v1/risk/monitoring/alerts/:id/acknowledge`

**认证要求**：无需认证

#### 解决告警

**接口地址**：`POST /api/v1/risk/monitoring/alerts/:id/resolve`

**认证要求**：无需认证

#### 生成监控报告

**接口地址**：`POST /api/v1/risk/monitoring/reports`

**认证要求**：无需认证

**请求参数**：

```json
{
  "report_type": "risk",
  "start_time": "2024-05-01T00:00:00Z",
  "end_time": "2024-05-15T00:00:00Z"
}
```

**响应示例**：

```json
{
  "code": 0,
  "message": "报告生成成功",
  "data": {
    "id": 1,
    "report_type": "risk",
    "report_name": "risk报告_20240501_20240515",
    "period_start": "2024-05-01T00:00:00Z",
    "period_end": "2024-05-15T00:00:00Z",
    "summary": "平均风险评分: 68.50; 拦截率: 3.00%; 误报率: 3.00%;",
    "generated_at": "2024-05-15T12:00:00Z"
  }
}
```

#### 获取监控报告列表

**接口地址**：`GET /api/v1/risk/monitoring/reports`

**认证要求**：无需认证

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| report_type | string | 否 | 报告类型 |
| limit | int | 否 | 返回数量限制 (默认10) |

---

### 深度强化学习(DRL) API

#### 获取DRL策略状态

**接口地址**：`GET /api/v1/risk/drl/status`

**认证要求**：无需认证

**响应示例**：

```json
{
  "code": 0,
  "message": "查询成功",
  "data": {
    "current_performance": 5.2,
    "outcomes_summary": {
      "total_requests": 10000,
      "successful": 9500,
      "failed": 500,
      "accuracy": 0.95,
      "avg_latency_ms": 8.5,
      "action_counts": {
        "allow": 8000,
        "captcha": 1500,
        "review": 300,
        "block": 200
      },
      "action_accuracy": {
        "allow": 0.98,
        "captcha": 0.92,
        "review": 0.85,
        "block": 0.88
      },
      "current_exploration": 0.05,
      "policy_performance": 5.2
    }
  }
}
```

#### 记录DRL结果

**接口地址**：`POST /api/v1/risk/drl/outcome`

**认证要求**：无需认证

**请求参数**：

```json
{
  "session_id": "string",
  "action": "string",    // allow/captcha/review/block/challenge
  "success": boolean,
  "latency_ms": 5
}
```

#### 训练DRL模型

**接口地址**：`POST /api/v1/risk/drl/train`

**认证要求**：无需认证

**请求参数**：

```json
{
  "batch_size": 32
}
```

**响应示例**：

```json
{
  "code": 0,
  "message": "模型训练成功"
}
```

---

### 错误码

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 400 | 参数错误 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

---

### 性能指标

| 指标 | 目标值 | 说明 |
|------|--------|------|
| 响应时间 | < 10ms | 99分位 |
| 可用性 | 99.9% | 全年可用性 |
| 并发处理 | 10000 QPS | 每秒请求数 |
| 准确率 | > 90% | 风险识别准确率 |
| 误报率 | < 5% | 正常请求被误拦比例 |

---

### 使用示例

#### Go语言示例

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

type RiskAssessmentRequest struct {
    Fingerprint string                 `json:"fingerprint"`
    IPAddress   string                 `json:"ip_address"`
    SessionID   string                 `json:"session_id"`
    DeviceInfo  map[string]interface{} `json:"device_info"`
    BehaviorData map[string]interface{} `json:"behavior_data"`
}

type RiskAssessmentResponse struct {
    RiskScore      float64  `json:"risk_score"`
    RiskLevel      string   `json:"risk_level"`
    Action         string   `json:"action"`
    Factors        []string `json:"factors"`
    ProcessingTime int64    `json:"processing_time_ms"`
}

func main() {
    req := RiskAssessmentRequest{
        Fingerprint: "test_fingerprint_001",
        IPAddress:   "192.168.1.100",
        SessionID:   "session_001",
        DeviceInfo: map[string]interface{}{
            "user_agent":    "Mozilla/5.0 Chrome/91.0",
            "screen_res":    "1920x1080",
        },
        BehaviorData: map[string]interface{}{
            "score": 85.0,
        },
    }

    jsonData, _ := json.Marshal(req)
    resp, err := http.Post(
        "http://localhost:8080/api/v1/risk/assess",
        "application/json",
        bytes.NewBuffer(jsonData),
    )
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)

    fmt.Printf("Risk Score: %.2f\n", result["data"].(map[string]interface{})["risk_score"])
}
```

#### JavaScript示例

```javascript
async function assessRisk(fingerprint, ipAddress) {
    const response = await fetch('/api/v1/risk/assess', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            fingerprint,
            ip_address: ipAddress,
            session_id: 'session_' + Date.now(),
            device_info: {
                user_agent: navigator.userAgent,
                screen_resolution: `${screen.width}x${screen.height}`,
            }
        })
    });

    const result = await response.json();
    return result.data;
}

// 使用示例
const riskResult = await assessRisk('fp_xxx', '192.168.1.100');
console.log('Risk Level:', riskResult.risk_level);
console.log('Action:', riskResult.action);
```

---

## 附录

### A. 风险等级说明

| 等级 | 评分范围 | 说明 | 建议动作 |
|------|----------|------|----------|
| 低 (low) | 80-100 | 风险极低，可信用户 | 直接放行 |
| 中 (medium) | 60-79 | 风险中等 | 验证码挑战 |
| 高 (high) | 40-59 | 风险较高 | 人工审核 |
| 严重 (critical) | 0-39 | 风险极高 | 直接拦截 |

### B. 支持的动作

| 动作 | 说明 |
|------|------|
| allow | 允许访问 |
| captcha | 要求完成验证码 |
| review | 人工审核 |
| block | 阻止访问 |
| challenge | 额外验证挑战 |

### C. 规则类型

| 类型 | 说明 |
|------|------|
| rate_limit | 频率限制规则 |
| behavior | 行为分析规则 |
| device_fingerprint | 设备指纹规则 |
| network | 网络特征规则 |
| geo | 地理位置规则 |
| blacklist | 黑名单规则 |

### D. 告警级别

| 级别 | 说明 | 处理时效 |
|------|------|----------|
| critical | 严重告警 | 即时处理 |
| high | 高优先级 | 1小时内 |
| medium | 中优先级 | 4小时内 |
| low | 低优先级 | 24小时内 |

---

**文档版本**: v15.0
**最后更新**: 2024-05-18
**维护团队**: AI风控引擎开发组
