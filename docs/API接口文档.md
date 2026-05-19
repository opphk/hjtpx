# API 接口文档 v15.0

## 目录

1. [概述](#概述)
2. [认证](#认证)
3. [用户端 API](#用户端-api)
   - [滑块验证码](#滑块验证码)
   - [点选验证码](#点选验证码)
   - [旋转验证码](#旋转验证码)
   - [手势验证码](#手势验证码)
   - [拼图验证码](#拼图验证码)
   - [图形验证码](#图形验证码)
   - [连连看验证码](#连连看验证码)
   - [3D验证码](#3d验证码)
   - [语音验证码](#语音验证码)
   - [无感验证](#无感验证)
   - [环境检测](#环境检测)
4. [v15.0 新增接口](#v150-新增接口)
5. [管理端 API](#管理端-api)
6. [错误码](#错误码)
7. [示例](#示例)

---

## v15.0 新增接口

### 增强行为分析接口
#### 获取高级行为分析报告
```
POST /api/v1/behavior/advanced-analyze
Content-Type: application/json

{
  "session_id": "sess_1715000000000_1234",
  "trajectory_data": [...],
  "context": {
    "captcha_type": "slider",
    "device_id": "device_123456"
  }
}
```

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "human_probability": 0.95,
    "risk_score": 12.5,
    "anomaly_patterns": [
      {
        "type": "linear_movement",
        "severity": "low",
        "description": "轨迹过于线性"
      }
    ],
    "features": {
      "dtw_distance": 15.2,
      "entropy": 0.87,
      "velocity_variance": 3.2
    }
  }
}
```

---

### 增强环境检测接口
#### 高级环境检测
```
POST /api/v1/detect/enhanced
Content-Type: application/json

{
  "fingerprint": {
    "canvas": "canvas_hash",
    "webgl": "webgl_hash",
    "audio": "audio_hash",
    "fonts": [...],
    "plugins": [...],
    "navigator": {...},
    "webrtc": {...}
  },
  "behavior_signals": {
    "mouse_movements": [...],
    "keyboard_events": [...]
  }
}
```

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "is_human": true,
    "is_proxy": false,
    "is_vpn": false,
    "is_tor": false,
    "is_emulator": false,
    "is_headless": false,
    "is_real_browser": true,
    "fingerprint_id": "fp_1234567890",
    "risk_score": 8.5,
    "trust_level": "high",
    "details": {
      "canvas_modified": false,
      "webgl_renderer_valid": true,
      "timezone_consistent": true,
      "webrtc_no_leak": true
    }
  }
}
```

---

### 多级缓存接口
#### 缓存预热
```
POST /api/v1/cache/warmup
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "type": "all",
  "scope": "application:12345"
}
```

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "status": "in_progress",
    "total_items": 1000,
    "processed": 250,
    "estimated_time": 30
  }
}
```

#### 获取缓存状态
```
GET /api/v1/cache/status
Authorization: Bearer <admin_token>
```

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "l1": {
      "size": 950,
      "max_size": 1000,
      "hit_rate": 0.95,
      "misses": 50
    },
    "l2": {
      "size": 9500,
      "max_size": 10000,
      "hit_rate": 0.88,
      "misses": 1200
    },
    "redis": {
      "connected": true,
      "memory_usage": 256,
      "memory_max": 512,
      "hit_rate": 0.92
    }
  }
}
```

---

### 自适应难度接口
#### 获取自适应难度配置
```
GET /api/v1/adaptive-difficulty/config?app_id=12345
```

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "enabled": true,
    "mode": "auto",
    "current_level": "medium",
    "history": [
      { "level": "easy", "timestamp": "2026-05-17T10:00:00Z" },
      { "level": "medium", "timestamp": "2026-05-17T10:05:00Z" }
    ]
  }
}
```

#### 更新自适应难度配置
```
PUT /api/v1/adaptive-difficulty/config
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "app_id": "12345",
  "enabled": true,
  "mode": "auto",
  "min_level": "easy",
  "max_level": "expert",
  "thresholds": {
    "easy": 10,
    "medium": 25,
    "hard": 50,
    "expert": 75
  }
}
```

---

### A/B测试接口
#### 创建A/B测试
```
POST /api/v1/ab-test/create
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "name": "验证码样式优化测试",
  "description": "测试新滑块样式的验证效果",
  "variants": [
    { "name": "control", "weight": 50, "config": {} },
    { "name": "variant_a", "weight": 50, "config": { "style": "modern" } }
  ],
  "start_time": "2026-05-18T00:00:00Z",
  "end_time": "2026-05-25T00:00:00Z"
}
```

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "test_id": "ab_test_123",
    "status": "created"
  }
}
```

#### 获取A/B测试结果
```
GET /api/v1/ab-test/result?test_id=ab_test_123
Authorization: Bearer <admin_token>
```

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "test_id": "ab_test_123",
    "status": "running",
    "variants": {
      "control": {
        "impressions": 1000,
        "success_rate": 0.85,
        "avg_time": 3.5,
        "risk_score_avg": 12.3
      },
      "variant_a": {
        "impressions": 1000,
        "success_rate": 0.88,
        "avg_time": 3.2,
        "risk_score_avg": 11.8
      }
    },
    "winning": "variant_a",
    "confidence": 0.95
  }
}
```

---

### 系统指标接口
#### 获取实时系统指标
```
GET /api/v1/system/metrics
Authorization: Bearer <admin_token>
```

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "qps": { "current": 1250, "avg_5m": 1180, "peak": 1500 },
    "latency": { "p50": 25, "p95": 45, "p99": 78 },
    "cpu": { "usage": 45.2, "cores": 8 },
    "memory": { "used": 2048, "total": 4096, "percent": 50 },
    "database": { "connections": 45, "max_connections": 100, "query_latency": 12 },
    "redis": { "memory": 256, "clients": 120, "hit_rate": 0.95 },
    "errors": { "rate_5m": 0.001, "total_5m": 15 }
  }
}
```

---

### 风险规则接口
#### 获取风险规则列表
```
GET /api/v1/risk-rules
Authorization: Bearer <admin_token>
```

**响应示例**:
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
        "enabled": true,
        "threshold": 5,
        "action": "challenge",
        "description": "1秒内点击超过5次"
      },
      {
        "id": 2,
        "name": "异常轨迹",
        "type": "trajectory",
        "enabled": true,
        "threshold": 3.5,
        "action": "reject",
        "description": "DTW距离超过阈值"
      }
    ],
    "total": 15
  }
}
```

#### 添加风险规则
```
POST /api/v1/risk-rules
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "name": "新风险规则",
  "type": "custom",
  "enabled": true,
  "threshold": 50,
  "action": "challenge",
  "conditions": [
    { "field": "risk_score", "operator": ">", "value": 50 }
  ]
}
```

---

### 备份与恢复接口
#### 创建数据备份
```
POST /api/v1/backup/create
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "type": "full",
  "include_logs": true,
  "compress": true
}
```

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "backup_id": "backup_123456",
    "status": "in_progress",
    "size_estimate": 2048
  }
}
```

#### 获取备份列表
```
GET /api/v1/backup/list
Authorization: Bearer <admin_token>
```

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "backups": [
      {
        "id": "backup_123456",
        "type": "full",
        "size": 2048,
        "created_at": "2026-05-18T00:00:00Z",
        "status": "completed"
      }
    ]
  }
}
```

---

## 目录

1. [概述](#概述)
2. [认证](#认证)
3. [用户端 API](#用户端-api)
   - [滑块验证码](#滑块验证码)
   - [点选验证码](#点选验证码)
   - [旋转验证码](#旋转验证码)
   - [手势验证码](#手势验证码)
   - [拼图验证码](#拼图验证码)
   - [图形验证码](#图形验证码)
   - [连连看验证码](#连连看验证码)
   - [3D验证码](#3d验证码)
   - [语音验证码](#语音验证码)
   - [无感验证](#无感验证)
   - [环境检测](#环境检测)
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

### 连连看验证码

#### 生成连连看验证码

```
POST /captcha/lianliankan/create
```

**请求参数：**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| difficulty | string | 否 | 难度级别：easy, medium, hard, expert |
| pairs | int | 否 | 配对数量（4-12），默认6 |

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "session_id": "sess_lianliankan_xxx",
    "background_image": "data:image/png;base64...",
    "icons": [
      {"id": 1, "icon": "🍎", "pairs": [0, 5]},
      {"id": 2, "icon": "🍊", "pairs": [1, 6]},
      {"id": 3, "icon": "🍋", "pairs": [2, 7]}
    ],
    "grid_size": {"rows": 4, "cols": 6},
    "difficulty": "medium",
    "pairs": 6
  }
}
```

#### 验证连连看

```
POST /captcha/lianliankan/verify
```

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "success": true,
    "accuracy": 1.0,
    "risk_score": 8.5,
    "captcha_pass": true
  }
}
```

---

### 3D验证码

#### 生成3D验证码

```
POST /captcha/3d/create
```

**请求参数：**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| difficulty | string | 否 | 难度级别：easy, medium, hard, expert |
| model_type | string | 否 | 模型类型：cube, sphere, cylinder, cone |

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "session_id": "sess_3d_xxx",
    "model_data": {
      "type": "cube",
      "vertices": [...],
      "faces": [...]
    },
    "target_rotation": {"x": 45, "y": 90, "z": 0},
    "background_image": "data:image/png;base64...",
    "difficulty": "medium",
    "model_type": "cube"
  }
}
```

#### 验证3D验证码

```
POST /captcha/3d/verify
```

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "success": true,
    "rotation_diff": 3.5,
    "risk_score": 7.2,
    "captcha_pass": true
  }
}
```

---

### 语音验证码
### 3D验证码
### Emoji验证码
### 生物特征验证接口
### 白标定制接口
### 高级分析接口
### Webhook接口
5. [错误码](#错误码)
6. [示例](#示例)

---

## Emoji验证码

### 生成Emoji验证码

```
POST /captcha/emoji/create
```

**请求参数：**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| difficulty | string | 否 | 难度级别：easy, medium, hard, expert |
| count | int | 否 | Emoji数量（3-6），默认4 |

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "session_id": "sess_emoji_xxx",
    "background_image": "data:image/png;base64...",
    "emojis": [
      {"id": 1, "emoji": "🐱", "target": true, "x": 50, "y": 50},
      {"id": 2, "emoji": "🐶", "target": false, "x": 150, "y": 50},
      {"id": 3, "emoji": "🐰", "target": true, "x": 250, "y": 50},
      {"id": 4, "emoji": "🐼", "target": false, "x": 100, "y": 150}
    ],
    "target_emojis": ["🐱", "🐰"],
    "difficulty": "medium",
    "count": 4
  }
}
```

### 验证Emoji验证码

```
POST /captcha/emoji/verify
Content-Type: application/json

{
  "session_id": "sess_emoji_xxx",
  "selected_ids": [1, 3],
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
    "accuracy": 1.0,
    "risk_score": 9.2,
    "captcha_pass": true
  }
}
```

---

## 生物特征验证接口

### 注册生物特征档案

```
POST /api/v1/biometrics/register
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "user_id": "user_123456",
  "keyboard_sample": {
    "typing_speed": 85.5,
    "total_chars": 256,
    "dwell_time_avg": 120.3,
    "flight_time_avg": 80.5,
    "error_rate": 0.02
  },
  "mouse_sample": {
    "movement_speed_avg": 250.5,
    "click_frequency": 3.2,
    "scroll_behavior": "normal"
  }
}
```

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "profile_id": "profile_123456",
    "enrollment_status": "completed",
    "confidence_score": 0.92
  }
}
```

### 验证生物特征

```
POST /api/v1/biometrics/verify
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "user_id": "user_123456",
  "keyboard_sample": {
    "typing_speed": 84.8,
    "total_chars": 200,
    "dwell_time_avg": 118.5,
    "flight_time_avg": 82.1,
    "error_rate": 0.015
  }
}
```

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "match": true,
    "confidence": 0.88,
    "risk_score": 5.5
  }
}
```

---

## 白标定制接口

### 获取白标配置

```
GET /api/v1/whitelabel/config
Authorization: Bearer <admin_token>
```

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "brand_name": "YourBrand",
    "primary_color": "#1890ff",
    "success_color": "#52c41a",
    "warning_color": "#faad14",
    "danger_color": "#f5222d",
    "logo_url": "https://example.com/logo.png",
    "favicon_url": "https://example.com/favicon.ico",
    "custom_css": ".captcha-container { border-radius: 8px; }",
    "is_enabled": true
  }
}
```

### 更新白标配置

```
PUT /api/v1/whitelabel/config
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "brand_name": "NewBrand",
  "primary_color": "#722ed1",
  "custom_css": ".captcha-container { border-radius: 12px; }",
  "is_enabled": true
}
```

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "message": "whitelabel config updated successfully",
    "config": {
      "brand_name": "NewBrand",
      "primary_color": "#722ed1",
      "is_enabled": true
    }
  }
}
```

### 获取白标CSS

```
GET /api/v1/whitelabel/css
```

返回动态生成的CSS样式。

---

## 高级分析接口

### 行为轨迹分析

```
POST /api/v1/behavior/analyze
Content-Type: application/json

{
  "session_id": "sess_123456",
  "trajectory_data": [
    {"x": 0, "y": 100, "timestamp": 1715000001000, "event": "move"},
    {"x": 50, "y": 102, "timestamp": 1715000001050, "event": "move"},
    {"x": 100, "y": 98, "timestamp": 1715000001100, "event": "move"},
    {"x": 150, "y": 101, "timestamp": 1715000001150, "event": "move"}
  ],
  "captcha_type": "slider",
  "device_id": "device_123456"
}
```

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "human_probability": 0.92,
    "risk_score": 15.5,
    "features": {
      "dtw_distance": 12.3,
      "velocity_variance": 2.8,
      "acceleration_pattern": "natural",
      "entropy": 0.85
    },
    "anomalies": []
  }
}
```

### 意图识别

```
POST /api/v1/behavior/intent
Content-Type: application/json

{
  "session_id": "sess_123456",
  "behavior_sequence": [
    {"event": "page_load", "timestamp": 1715000000000},
    {"event": "mouse_move", "timestamp": 1715000000500},
    {"event": "focus_input", "timestamp": 1715000002000},
    {"event": "type", "timestamp": 1715000002500}
  ],
  "context": {
    "page_type": "login",
    "user_authenticated": false
  }
}
```

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "primary_intent": "user_login",
    "confidence": 0.89,
    "secondary_intents": ["form_filling"],
    "risk_level": "low"
  }
}
```

---

## Webhook接口

### 配置Webhook

```
POST /api/v1/webhooks
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "url": "https://your-server.com/webhook",
  "events": ["verification.success", "verification.failed", "risk.detected"],
  "secret": "your-webhook-secret",
  "enabled": true
}
```

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "webhook_id": "wh_123456",
    "url": "https://your-server.com/webhook",
    "events": ["verification.success", "verification.failed", "risk.detected"],
    "status": "active",
    "created_at": "2026-05-18T10:00:00Z"
  }
}
```

### 测试Webhook

```
POST /api/v1/webhooks/:id/test
Authorization: Bearer <admin_token>
```

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "webhook_id": "wh_123456",
    "test_result": "delivered",
    "response_time_ms": 150,
    "status_code": 200
  }
}
```

### Webhook事件 payload 示例

```json
{
  "event": "verification.success",
  "timestamp": "2026-05-18T10:30:00Z",
  "data": {
    "session_id": "sess_123456",
    "captcha_type": "slider",
    "risk_score": 12.5,
    "user_id": "user_123456",
    "ip_address": "192.168.1.100",
    "user_agent": "Mozilla/5.0..."
  }
}
```

### 语音验证码

#### 生成语音验证码

```
POST /captcha/voice/create
```

**请求参数：**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| mode | string | 否 | 语音模式：number, letter, mixed |
| count | int | 否 | 字符数量（4-8），默认4 |

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "session_id": "sess_voice_xxx",
    "audio_url": "data:audio/mp3;base64,...",
    "text": "A3B7",
    "duration": 3,
    "mode": "mixed",
    "count": 4
  }
}
```

#### 验证语音验证码

```
POST /captcha/voice/verify
```

**响应示例：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "success": true,
    "risk_score": 5.8,
    "captcha_pass": true
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

### 错误码总览

系统采用分层错误码设计，错误码格式为 `XXYYY`：
- `XX`：错误类别（01-99）
- `YYY`：具体错误编号（001-999）

| 错误码 | 说明 | HTTP状态码 | 处理建议 |
|--------|------|-----------|----------|
| **成功** |
| 0 | 成功 | 200 | - |
| **验证错误 (10xxx)** |
| 10001 | 验证失败 | 200 | 检查验证参数是否正确 |
| 10002 | Session过期 | 200 | 重新获取验证码 |
| 10003 | 参数错误 | 400 | 检查请求参数格式 |
| 10004 | 验证码类型不支持 | 400 | 使用支持的验证码类型 |
| 10005 | 验证码已过期 | 200 | 重新获取验证码 |
| 10006 | 验证过于频繁 | 200 | 稍后重试 |
| 10007 | 验证次数超限 | 200 | 超过最大验证次数 |
| 10008 | 行为风险过高 | 200 | 触发风控规则 |
| 10009 | 环境检测异常 | 200 | 环境可能被修改 |
| 10010 | Token无效 | 200 | 重新获取验证Token |
| **认证错误 (20xxx)** |
| 20001 | 认证失败 | 401 | 检查认证信息 |
| 20002 | Token无效 | 401 | 重新登录获取Token |
| 20003 | Token过期 | 401 | 刷新Token或重新登录 |
| 20004 | 签名无效 | 401 | 检查签名算法和密钥 |
| 20005 | 签名过期 | 401 | 重新生成签名 |
| 20006 | 权限不足 | 403 | 联系管理员授权 |
| 20007 | 账户被禁用 | 403 | 联系管理员启用账户 |
| 20008 | 账户被锁定 | 403 | 等待解锁或联系管理员 |
| 20009 | 登录失败次数超限 | 403 | 稍后重试 |
| 20010 | MFA验证失败 | 401 | 检查MFA验证码 |
| **资源错误 (30xxx)** |
| 30001 | 资源不存在 | 404 | 检查请求的资源ID |
| 30002 | 资源已存在 | 409 | 避免重复创建 |
| 30003 | 资源已过期 | 410 | 重新创建资源 |
| 30004 | 资源被占用 | 409 | 等待资源释放 |
| **限流错误 (40xxx)** |
| 40001 | 接口限流 | 429 | 降低请求频率 |
| 40002 | 全局限流 | 429 | 全局请求过多，稍后重试 |
| 40003 | IP限流 | 429 | 当前IP请求过于频繁 |
| 40004 | 用户限流 | 429 | 当前用户请求过于频繁 |
| 40005 | 应用限流 | 429 | 当前应用请求过于频繁 |
| 40006 | 并发限制 | 429 | 等待并发请求完成 |
| **服务器错误 (50xxx)** |
| 50001 | 服务器内部错误 | 500 | 联系技术支持 |
| 50002 | 服务暂不可用 | 503 | 服务维护中，稍后重试 |
| 50003 | 数据库错误 | 500 | 检查数据库连接 |
| 50004 | 缓存错误 | 500 | 检查Redis连接 |
| 50005 | 验证码生成失败 | 500 | 重试获取验证码 |
| 50006 | 配置错误 | 500 | 检查系统配置 |
| 50007 | 文件上传失败 | 500 | 检查文件大小和格式 |
| 50008 | 任务执行失败 | 500 | 重试或联系技术支持 |
| **v15.0 新增错误码** |
| 10011 | A/B测试无效 | 400 | 检查A/B测试配置 |
| 10012 | 备份任务不存在 | 404 | 检查备份ID |
| 10013 | 缓存预热失败 | 500 | 检查缓存服务状态 |
| 10014 | 规则冲突 | 409 | 检查规则配置是否冲突 |
| 40007 | 多级缓存错误 | 500 | 检查缓存层状态 |
| 50009 | 高级分析失败 | 500 | 重试或联系技术支持 |

### 错误码详细说明

#### 10001 - 验证失败

**原因分析**：
- 用户输入的验证答案不正确
- 轨迹数据被判定为机器人行为
- 环境检测发现异常

**处理建议**：
```javascript
// 前端处理示例
if (error.code === 10001) {
    // 提示用户重新验证
    showToast('验证失败，请重试');
    // 重新获取验证码
    refreshCaptcha();
}
```

```go
// Go语言后端处理示例
if err.Code == 10001 {
    log.Printf("验证码验证失败: session=%s", sessionID)
    return c.JSON(400, gin.H{
        "code": 10001,
        "message": "验证失败",
        "data": gin.H{
            "retry_allowed": true,
            "remaining_attempts": remainingAttempts,
        },
    })
}
```

#### 10002 - Session过期

**原因分析**：
- 验证码Session超时（默认5分钟）
- Session被服务端清除
- 并发验证导致Session失效

**处理建议**：
```javascript
if (error.code === 10002) {
    // 重新获取验证码
    getNewCaptcha();
}
```

```go
if err.Code == 10002 {
    // 生成新的验证码
    newCaptcha, _ := client.GenerateSliderCaptcha()
    return c.JSON(200, gin.H{
        "code": 10002,
        "message": "Session已过期，已自动刷新",
        "data": newCaptcha,
    })
}
```

#### 10003 - 参数错误

**原因分析**：
- 必填参数缺失
- 参数格式不正确
- 参数值超出有效范围

**详细错误响应示例**：
```json
{
  "code": 10003,
  "message": "参数错误",
  "data": {
    "field": "session_id",
    "reason": "会话ID不能为空",
    "expected": "string",
    "received": "null"
  }
}
```

#### 10004 - 验证码类型不支持

**原因分析**：
- 使用了未启用的验证码类型
- 应用未配置该验证码类型

**处理建议**：
```go
if err.Code == 10004 {
    // 获取应用支持的验证码类型
    app, _ := client.GetApplication(appID)
    supportedTypes := app.SupportedCaptchaTypes
    return c.JSON(400, gin.H{
        "code": 10004,
        "message": "不支持的验证码类型",
        "data": gin.H{
            "supported_types": supportedTypes,
            "requested_type": requestedType,
        },
    })
}
```

#### 10005 - 验证码已过期

**原因分析**：
- 验证码超过有效期（默认300秒）
- 用户操作时间过长

**处理建议**：
```javascript
if (error.code === 10005) {
    alert('验证码已过期，请重新获取');
    captchaWidget.reset();
}
```

#### 10006 - 验证过于频繁

**原因分析**：
- 单个用户在短时间内验证次数过多
- 触发了频率限制规则

**处理建议**：
```go
if err.Code == 10006 {
    retryAfter := time.Now().Add(30 * time.Second)
    c.Header("Retry-After", "30")
    return c.JSON(429, gin.H{
        "code": 10006,
        "message": "验证过于频繁，请稍后再试",
        "data": gin.H{
            "retry_after": retryAfter.Unix(),
            "wait_seconds": 30,
        },
    })
}
```

#### 10007 - 验证次数超限

**原因分析**：
- 单个验证码验证次数超过限制（默认3次）
- 多次失败后被临时封禁

**处理建议**：
```javascript
if (error.code === 10007) {
    alert('验证次数已用完，请重新获取验证码');
    // 禁用验证按钮
    verifyButton.disabled = true;
    // 等待一段时间后重新启用
    setTimeout(() => {
        verifyButton.disabled = false;
        refreshCaptcha();
    }, 60000);
}
```

#### 10008 - 行为风险过高

**原因分析**：
- 轨迹数据表现出机器人特征
- 检测到自动化工具特征
- 行为模式异常

**处理建议**：
```go
if err.Code == 10008 {
    // 记录风控事件
    logSecurityEvent("high_risk_behavior", gin.H{
        "session_id": sessionID,
        "risk_score": riskScore,
        "user_id": userID,
        "ip": clientIP,
    })
    // 触发额外验证或直接拒绝
    return c.JSON(200, gin.H{
        "code": 10008,
        "message": "检测到异常行为，请进行额外验证",
        "data": gin.H{
            "action": "require_mfa",
            "risk_level": "high",
        },
    })
}
```

#### 20003 - Token过期

**原因分析**：
- JWT Token超过有效期
- Token被刷新导致旧Token失效

**处理建议**：
```javascript
if (error.code === 20003) {
    // 尝试刷新Token
    refreshToken().then(() => {
        // 重试原请求
        retryOriginalRequest();
    }).catch(() => {
        // 刷新失败，重新登录
        redirectToLogin();
    });
}
```

#### 40001 - 接口限流

**原因分析**：
- 单个接口请求频率过高
- 触发了服务端速率限制

**处理建议**：
```javascript
if (error.code === 40001) {
    // 从响应头获取重试时间
    const retryAfter = response.headers['retry-after'] || 60;
    // 延迟后重试
    setTimeout(retryRequest, retryAfter * 1000);
}
```

### 错误响应格式

#### 标准错误响应
```json
{
  "code": 10001,
  "message": "验证失败",
  "data": null
}
```

#### 详细错误响应
```json
{
  "code": 10003,
  "message": "参数错误",
  "data": {
    "field": "session_id",
    "reason": "会话ID不能为空",
    "expected": "string",
    "received": "null"
  }
}
```

#### 限流错误响应
```json
{
  "code": 40001,
  "message": "请求过于频繁",
  "data": {
    "retry_after": 60,
    "limit": 100,
    "window": "1 minute",
    "remaining": 0
  }
}
```

#### 认证错误响应
```json
{
  "code": 20003,
  "message": "Token已过期",
  "data": {
    "token_type": "access_token",
    "expired_at": "2026-05-18T12:00:00Z",
    "refresh_token": "your-refresh-token"
  }
}
```

### 错误日志级别

| 错误码范围 | 日志级别 | 说明 |
|-----------|---------|------|
| 10xxx | WARN | 业务验证错误，需要关注但非紧急 |
| 20xxx | WARN | 认证错误，可能存在安全风险 |
| 30xxx | INFO | 资源相关错误 |
| 40xxx | WARN | 限流触发，正常的流量控制 |
| 50xxx | ERROR | 服务器错误，需要立即处理 |

---

**最后更新**: 2026-05-18
**当前版本**: v14.0

---

## 示例

### Java SDK 示例

```java
package com.example;

import com.hjtpx.captcha.client.CaptchaClient;
import com.hjtpx.captcha.client.CaptchaClientConfig;
import com.hjtpx.captcha.model.*;
import java.util.Arrays;
import java.util.List;

public class CaptchaIntegration {
    public static void main(String[] args) {
        // 配置客户端
        CaptchaClientConfig config = new CaptchaClientConfig();
        config.setBaseUrl("http://localhost:8080");
        config.setApiKey("your-api-key");
        config.setSecretKey("your-secret-key");
        config.setTimeout(30000);

        try (CaptchaClient client = new CaptchaClient(config)) {
            // 1. 获取滑块验证码
            SliderCaptchaResponse captcha = client.getSliderCaptcha(320, 160, 5);
            System.out.println("Session ID: " + captcha.getSessionId());
            
            // 2. 验证滑块位置
            List<TrajectoryPoint> trajectory = generateTrajectory(160);
            VerifyCaptchaResponse verifyResp = client.verifySliderCaptcha(
                captcha.getSessionId(),
                160,  // X坐标
                160,  // Y坐标
                trajectory  // 轨迹数据
            );
            
            if (verifyResp.isSuccess()) {
                System.out.println("验证通过！Token: " + verifyResp.getToken());
            }
        } catch (Exception e) {
            e.printStackTrace();
        }
    }
    
    private static List<TrajectoryPoint> generateTrajectory(int targetY) {
        long baseTime = System.currentTimeMillis();
        return Arrays.asList(
            new TrajectoryPoint(0, targetY, baseTime - 1000),
            new TrajectoryPoint(30, targetY + 2, baseTime - 800),
            new TrajectoryPoint(60, targetY - 1, baseTime - 600),
            new TrajectoryPoint(100, targetY + 3, baseTime - 400),
            new TrajectoryPoint(140, targetY - 2, baseTime - 200),
            new TrajectoryPoint(160, targetY, baseTime)
        );
    }
}
```

### Python SDK 示例

```python
from hjtpx import CaptchaClient
from hjtpx.exceptions import CaptchaError, NetworkError

def main():
    client = CaptchaClient(
        endpoint="http://localhost:8080",
        api_key="your-api-key",
        timeout=30
    )
    
    try:
        # 获取滑块验证码
        captcha = client.get_slider_captcha(
            width=320,
            height=160,
            tolerance=5
        )
        print(f"Session ID: {captcha['session_id']}")
        
        # 生成模拟轨迹
        trajectory = generate_trajectory(captcha['secret_y'])
        
        # 验证
        result = client.verify_slider_captcha(
            session_id=captcha['session_id'],
            x=160,
            y=captcha['secret_y'],
            trajectory=trajectory
        )
        
        if result['success']:
            print(f"验证通过！风险分数: {result.get('risk_score', 0)}")
            
    except CaptchaError as e:
        print(f"验证码错误: {e.code} - {e.message}")
    except NetworkError as e:
        print(f"网络错误: {e}")
    finally:
        client.close()

def generate_trajectory(target_y):
    import time
    base_time = int(time.time() * 1000)
    trajectory = []
    for i in range(6):
        x = i * 30
        y = target_y + (i % 3 - 1) * 2
        trajectory.append({
            'x': x,
            'y': y,
            'timestamp': base_time + i * 200 - 1000
        })
    return trajectory

if __name__ == "__main__":
    main()
```

### PHP SDK 示例

```php
<?php
require_once 'vendor/autoload.php';

use Hjtpx\CaptchaClient;
use Hjtpx\Exception\CaptchaException;

$client = new CaptchaClient([
    'base_url' => 'http://localhost:8080',
    'api_key' => 'your-api-key',
    'api_secret' => 'your-api-secret',
    'timeout' => 30
]);

try {
    // 获取点选验证码
    $captcha = $client->getClickCaptcha([
        'mode' => 'number',
        'shuffle' => true,
        'points' => 3
    ]);
    
    echo "Session ID: " . $captcha['session_id'] . "\n";
    echo "提示: " . $captcha['hint'] . "\n";
    
    // 用户按顺序点击
    $points = [
        [100, 100],
        [200, 150],
        [150, 200]
    ];
    $clickSequence = [0, 1, 2];
    
    // 验证
    $result = $client->verifyClickCaptcha(
        $captcha['session_id'],
        $points,
        $clickSequence
    );
    
    if ($result['success']) {
        echo "验证通过！\n";
    }
    
} catch (CaptchaException $e) {
    echo "验证码错误: " . $e->getMessage() . "\n";
    echo "错误码: " . $e->getCode() . "\n";
} finally {
    $client->close();
}
```

### C# SDK 示例

```csharp
using Hjtpx.Captcha.Sdk;
using Hjtpx.Captcha.Sdk.Models;
using Hjtpx.Captcha.Sdk.Exceptions;

class Program
{
    static async Task Main(string[] args)
    {
        var config = new CaptchaClientConfig
        {
            BaseUrl = "http://localhost:8080",
            ApiKey = "your-api-key",
            ApiSecret = "your-api-secret",
            Timeout = 30000
        };

        using var client = new CaptchaClient(config);
        
        try
        {
            // 获取滑块验证码
            var sliderCaptcha = await client.GetSliderCaptchaAsync(320, 160, 5);
            Console.WriteLine($"Session ID: {sliderCaptcha.SessionId}");
            
            // 生成轨迹
            var trajectory = GenerateTrajectory(160);
            
            // 验证
            var result = await client.VerifySliderCaptchaAsync(
                sliderCaptcha.SessionId,
                160,  // X坐标
                160,  // Y坐标
                trajectory
            );
            
            if (result.Success)
            {
                Console.WriteLine($"验证通过！Token: {result.Token}");
            }
        }
        catch (ApiException ex)
        {
            Console.WriteLine($"API错误: {ex.Code} - {ex.Message}");
        }
        catch (NetworkException ex)
        {
            Console.WriteLine($"网络错误: {ex.Message}");
        }
    }
    
    private static List<TrajectoryPoint> GenerateTrajectory(int targetY)
    {
        var points = new List<TrajectoryPoint>();
        long baseTime = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds();
        
        for (int i = 0; i < 6; i++)
        {
            points.Add(new TrajectoryPoint
            {
                X = i * 30,
                Y = targetY + (i % 3 - 1) * 2,
                Timestamp = baseTime + i * 200 - 1000
            });
        }
        
        return points;
    }
}
```

### Ruby SDK 示例

```ruby
require 'hjtpx'

client = Hjtpx::CaptchaClient.new(
  base_url: 'http://localhost:8080',
  api_key: 'your-api-key',
  timeout: 30
)

begin
  # 获取滑块验证码
  captcha = client.get_slider_captcha(
    width: 320,
    height: 160,
    tolerance: 5
  )
  
  puts "Session ID: #{captcha[:session_id]}"
  
  # 生成轨迹
  trajectory = generate_trajectory(captcha[:secret_y])
  
  # 验证
  result = client.verify_slider_captcha(
    session_id: captcha[:session_id],
    x: 160,
    y: captcha[:secret_y],
    trajectory: trajectory
  )
  
  if result[:success]
    puts "验证通过！Token: #{result[:token]}"
  end
  
rescue Hjtpx::CaptchaError => e
  puts "验证码错误: #{e.code} - #{e.message}"
ensure
  client.close
end

def generate_trajectory(target_y)
  base_time = Time.now.to_i * 1000
  trajectory = []
  
  6.times do |i|
    trajectory << {
      x: i * 30,
      y: target_y + (i % 3 - 1) * 2,
      timestamp: base_time + i * 200 - 1000
    }
  end
  
  trajectory
end
```

### Go SDK 完整集成示例

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/hjtpx/hjtpx/sdk/go"
)

func main() {
    // 创建高级客户端（推荐生产环境使用）
    cfg := &captcha.Config{
        BaseURL:        "http://localhost:8080",
        MaxRetries:     3,
        HTTPTimeout:    10 * time.Second,
        MaxIdleConns:   10,
        MaxOpenConns:   100,
        RetryDelay:     100 * time.Millisecond,
    }

    client := captcha.NewCaptchaClient("your-app-id", "your-app-secret", cfg)
    defer client.Close()

    ctx := context.Background()

    // 1. 获取滑块验证码
    slider, err := client.GenerateSliderCaptcha()
    if err != nil {
        log.Fatalf("获取滑块验证码失败: %v", err)
    }
    fmt.Printf("滑块验证码 SessionID: %s\n", slider.ChallengeID)
    fmt.Printf("背景图: %s...\n", slider.BackgroundImage[:50])

    // 2. 模拟用户滑动轨迹
    trajectory := generateTrajectory(160)

    // 3. 验证滑块
    verifyResult, err := client.VerifySliderCaptcha(slider.ChallengeID, "160")
    if err != nil {
        log.Fatalf("验证失败: %v", err)
    }
    
    fmt.Printf("验证结果: %v\n", verifyResult.Success)
    fmt.Printf("风险分数: %.2f\n", verifyResult.Score)
    fmt.Printf("风险等级: %s\n", verifyResult.RiskLevel)

    // 4. 获取统计数据
    stats := client.GetStats()
    fmt.Printf("总请求数: %d\n", stats.TotalRequests)
    fmt.Printf("成功率: %.2f%%\n", stats.SuccessRate)
    fmt.Printf("重试次数: %d\n", stats.RetriedRequests)
}

func generateTrajectory(targetY int) []captcha.TrajectoryPoint {
    baseTime := time.Now().UnixMilli()
    points := make([]captcha.TrajectoryPoint, 0, 10)
    
    for i := 0; i < 10; i++ {
        x := float64(i * 16)
        y := float64(targetY) + float64(i%3-1) * 2
        timestamp := baseTime + int64(i*50)
        
        points = append(points, captcha.TrajectoryPoint{
            X:        x,
            Y:        y,
            Timestamp: timestamp,
        })
    }
    
    return points
}
```

---

## 使用场景示例

### 场景1：用户注册集成

#### 前端实现
```html
<!DOCTYPE html>
<html>
<head>
    <title>用户注册</title>
    <script src="http://localhost:8080/static/js/captcha.js"></script>
</head>
<body>
    <form id="registerForm">
        <input type="text" name="username" placeholder="用户名" required>
        <input type="email" name="email" placeholder="邮箱" required>
        <input type="password" name="password" placeholder="密码" required>
        <div id="captchaContainer"></div>
        <input type="hidden" id="captchaToken">
        <button type="submit">注册</button>
    </form>

    <script>
        // 初始化验证码
        HJTPXCaptcha.init({
            container: '#captchaContainer',
            apiServer: 'http://localhost:8080',
            captchaType: 'slider',
            onVerify: function(result) {
                if (result.success) {
                    document.getElementById('captchaToken').value = result.token;
                    console.log('验证成功，Token:', result.token);
                }
            },
            onError: function(error) {
                console.error('验证错误:', error);
            }
        });

        // 表单提交
        document.getElementById('registerForm').onsubmit = async function(e) {
            e.preventDefault();
            
            const token = document.getElementById('captchaToken').value;
            if (!token) {
                alert('请先完成验证码');
                return;
            }

            const formData = new FormData(this);
            formData.append('captcha_token', token);

            try {
                const response = await fetch('/api/register', {
                    method: 'POST',
                    body: formData
                });
                const result = await response.json();
                
                if (result.success) {
                    alert('注册成功！');
                    window.location.href = '/login';
                } else {
                    alert('注册失败: ' + result.message);
                }
            } catch (error) {
                console.error('注册错误:', error);
            }
        };
    </script>
</body>
</html>
```

#### 后端验证
```go
package main

import (
    "encoding/json"
    "net/http"
)

type RegisterRequest struct {
    Username    string `json:"username"`
    Email      string `json:"email"`
    Password   string `json:"password"`
    CaptchaToken string `json:"captcha_token"`
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
    var req RegisterRequest
    json.NewDecoder(r.Body).Decode(&req)

    // 1. 验证验证码Token
    verifyURL := "http://localhost:8080/api/v1/captcha/verify-token"
    resp, err := http.Post(verifyURL, "application/json", 
        strings.NewReader(fmt.Sprintf(`{"token":"%s"}`, req.CaptchaToken)))
    
    if err != nil || resp.StatusCode != 200 {
        http.Error(w, "验证码验证失败", http.StatusBadRequest)
        return
    }

    var verifyResult map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&verifyResult)

    if verifyResult["success"] != true {
        http.Error(w, "验证码验证失败", http.StatusBadRequest)
        return
    }

    // 2. 创建用户
    // ... 用户创建逻辑
}
```

### 场景2：登录保护集成

```javascript
// 登录失败重试限制
const MAX_LOGIN_ATTEMPTS = 5;
const LOCKOUT_DURATION = 15 * 60 * 1000; // 15分钟

class LoginManager {
    constructor() {
        this.attempts = new Map();
    }

    async login(username, password) {
        // 检查是否被锁定
        if (this.isLocked(username)) {
            const remainingTime = this.getRemainingLockoutTime(username);
            throw new Error(`账户已被锁定，请在 ${Math.ceil(remainingTime / 60000)} 分钟后重试`);
        }

        try {
            // 获取验证码
            const captcha = await HJTPXCaptcha.getCaptcha();
            
            // 执行登录
            const result = await this.executeLogin(username, password, captcha.token);
            
            // 登录成功，清除记录
            this.attempts.delete(username);
            return result;
            
        } catch (error) {
            // 登录失败，记录次数
            this.recordFailedAttempt(username);
            
            if (this.attempts.get(username) >= MAX_LOGIN_ATTEMPTS) {
                throw new Error('登录失败次数过多，账户已被临时锁定');
            }
            
            throw error;
        }
    }

    recordFailedAttempt(username) {
        const attempts = this.attempts.get(username) || 0;
        this.attempts.set(username, attempts + 1);
        
        if (attempts === 0) {
            // 首次失败，设置锁定时间
            setTimeout(() => this.attempts.delete(username), LOCKOUT_DURATION);
        }
    }

    isLocked(username) {
        const attempts = this.attempts.get(username) || 0;
        return attempts >= MAX_LOGIN_ATTEMPTS;
    }

    getRemainingLockoutTime(username) {
        // 返回剩余锁定时间
        return LOCKOUT_DURATION;
    }
}
```

### 场景3：无感验证集成

```javascript
// 无感验证集成示例
class SeamlessVerification {
    constructor() {
        this.trustLevel = null;
        this.deviceFingerprint = null;
    }

    async init() {
        // 收集设备指纹
        this.deviceFingerprint = await this.collectFingerprint();
        
        // 收集行为数据
        this.behaviorData = await this.collectBehaviorData();
        
        // 检查信任级别
        await this.checkTrustLevel();
    }

    async collectFingerprint() {
        const data = {
            canvas: await this.getCanvasFingerprint(),
            webgl: this.getWebGLFingerprint(),
            fonts: await this.detectFonts(),
            timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
            language: navigator.language,
            platform: navigator.platform,
            screen: `${screen.width}x${screen.height}`,
            colorDepth: screen.colorDepth
        };

        return await this.hashFingerprint(JSON.stringify(data));
    }

    async collectBehaviorData() {
        const data = {
            mouseMovements: this.trackMouseMovements(),
            keystrokeDynamics: this.trackKeystrokes(),
            scrollBehavior: this.trackScroll(),
            clickPatterns: this.trackClicks()
        };

        return data;
    }

    async checkTrustLevel() {
        const response = await fetch('http://localhost:8080/api/v1/seamless/check', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                device_fingerprint: this.deviceFingerprint,
                behavior_sequence: this.behaviorData.mouseMovements.slice(0, 10)
            })
        });

        const result = await response.json();
        this.trustLevel = result.data.trust_level;
        
        return {
            requiresCaptcha: result.data.requires_captcha,
            trustLevel: result.data.trust_level,
            riskScore: result.data.risk_score
        };
    }

    shouldRequireCaptcha() {
        // 高信任级别不需要验证码
        if (this.trustLevel === 'high') {
            return false;
        }
        
        // 中信任级别需要验证码
        if (this.trustLevel === 'medium') {
            return true;
        }
        
        // 低信任级别强制验证码
        return true;
    }
}
```

### 场景4：批量操作保护

```go
// 批量操作速率限制示例
package main

import (
    "sync"
    "time"
)

type RateLimiter struct {
    mu       sync.Mutex
    requests map[string][]time.Time
    limit    int
    window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
    return &RateLimiter{
        requests: make(map[string][]time.Time),
        limit:    limit,
        window:   window,
    }
}

func (r *RateLimiter) Allow(key string) bool {
    r.mu.Lock()
    defer r.mu.Unlock()

    now := time.Now()
    cutoff := now.Add(-r.window)

    // 清理过期记录
    var validRequests []time.Time
    for _, t := range r.requests[key] {
        if t.After(cutoff) {
            validRequests = append(validRequests, t)
        }
    }

    if len(validRequests) >= r.limit {
        r.requests[key] = validRequests
        return false
    }

    r.requests[key] = append(validRequests, now)
    return true
}

func (r *RateLimiter) GetRemaining(key string) int {
    r.mu.Lock()
    defer r.mu.Unlock()

    now := time.Now()
    cutoff := now.Add(-r.window)

    count := 0
    for _, t := range r.requests[key] {
        if t.After(cutoff) {
            count++
        }
    }

    return r.limit - count
}

func (r *RateLimiter) GetResetTime(key string) time.Duration {
    r.mu.Lock()
    defer r.mu.Unlock()

    if len(r.requests[key]) == 0 {
        return 0
    }

    oldest := r.requests[key][0]
    return time.Until(oldest.Add(r.window))
}

// 使用示例
func handleBatchOperation(w http.ResponseWriter, r *http.Request) {
    limiter := NewRateLimiter(100, time.Minute)
    
    clientIP := getClientIP(r)
    
    if !limiter.Allow(clientIP) {
        remaining := limiter.GetRemaining(clientIP)
        resetTime := limiter.GetResetTime(clientIP)
        
        w.Header().Set("X-RateLimit-Limit", "100")
        w.Header().Set("X-RateLimit-Remaining", string(rune(remaining)))
        w.Header().Set("X-RateLimit-Reset", resetTime.String())
        
        http.Error(w, "请求过于频繁", http.StatusTooManyRequests)
        return
    }

    // 处理批量操作
    // ...
}
```

### 场景5：敏感操作二次验证

```javascript
// 敏感操作需要MFA二次验证
class SensitiveOperationHandler {
    constructor() {
        this.sensitiveOperations = [
            'password_change',
            'email_change',
            'phone_change',
            'withdrawal',
            'transfer'
        ];
    }

    async execute(operation, data) {
        // 检查是否需要MFA
        if (this.requiresMFA(operation)) {
            // 获取MFA验证码
            const mfaToken = await this.getMFAToken();
            
            // 验证MFA
            const mfaValid = await this.verifyMFA(mfaToken);
            
            if (!mfaValid) {
                throw new Error('MFA验证失败');
            }
        }

        // 执行操作
        return await this.performOperation(operation, data);
    }

    requiresMFA(operation) {
        return this.sensitiveOperations.includes(operation);
    }

    async getMFAToken() {
        // 弹出MFA验证界面
        return new Promise((resolve) => {
            HJTPXMFA.show({
                method: 'totp', // TOTP、短信、邮箱
                onVerify: (token) => {
                    resolve(token);
                },
                onCancel: () => {
                    resolve(null);
                }
            });
        });
    }

    async verifyMFA(token) {
        const response = await fetch('/api/v1/mfa/verify', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${getAccessToken()}`
            },
            body: JSON.stringify({
                token: token,
                action: 'sensitive_operation'
            })
        });

        return response.ok;
    }
}
```

### 场景6：生物特征验证集成

```go
// Go语言生物特征验证示例
package main

import (
    "fmt"
    "time"
)

type KeyboardSample struct {
    TypingSpeed   float64
    TotalChars    int
    DwellTimeAvg  float64
    FlightTimeAvg float64
    ErrorRate     float64
}

type MouseSample struct {
    MovementSpeedAvg float64
    ClickFrequency   float64
    ScrollBehavior   string
}

func main() {
    // 1. 注册生物特征档案
    keyboardSample := &KeyboardSample{
        TypingSpeed:   85.5,
        TotalChars:    256,
        DwellTimeAvg:  120.3,
        FlightTimeAvg: 80.5,
        ErrorRate:     0.02,
    }

    mouseSample := &MouseSample{
        MovementSpeedAvg: 250.5,
        ClickFrequency:   3.2,
        ScrollBehavior:   "normal",
    }

    profile, err := biometricsService.RegisterProfile("user_123456", keyboardSample, mouseSample)
    if err != nil {
        fmt.Printf("注册失败: %v\n", err)
        return
    }
    fmt.Printf("生物特征档案已注册: %s\n", profile.ID)

    // 2. 验证生物特征
    verifySample := &KeyboardSample{
        TypingSpeed:   84.8,
        TotalChars:    200,
        DwellTimeAvg:  118.5,
        FlightTimeAvg: 82.1,
        ErrorRate:     0.015,
    }

    result, err := biometricsService.Verify("user_123456", verifySample, nil)
    if err != nil {
        fmt.Printf("验证失败: %v\n", err)
        return
    }

    if result.Match {
        fmt.Printf("生物特征匹配成功，置信度: %.2f%%\n", result.Confidence*100)
    } else {
        fmt.Printf("生物特征不匹配，可信度: %.2f%%\n", result.Confidence*100)
    }
}
```

### 场景7：Webhook集成

```go
// Go语言Webhook处理示例
package main

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "io"
    "net/http"
)

type WebhookEvent struct {
    Event     string      `json:"event"`
    Timestamp string      `json:"timestamp"`
    Data      interface{} `json:"data"`
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
    // 1. 验证签名
    signature := r.Header.Get("X-Webhook-Signature")
    body, _ := io.ReadAll(r.Body)
    
    if !verifySignature(body, signature, "your-webhook-secret") {
        http.Error(w, "Invalid signature", http.StatusUnauthorized)
        return
    }

    // 2. 解析事件
    var event WebhookEvent
    if err := json.Unmarshal(body, &event); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    // 3. 处理不同类型的事件
    switch event.Event {
    case "verification.success":
        handleVerificationSuccess(event.Data)
    case "verification.failed":
        handleVerificationFailed(event.Data)
    case "risk.detected":
        handleRiskDetected(event.Data)
    default:
        fmt.Printf("未知事件类型: %s\n", event.Event)
    }

    w.WriteHeader(http.StatusOK)
}

func verifySignature(payload []byte, signature, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    expected := hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(signature), []byte(expected))
}
```

### 场景8：白标定制集成

```javascript
// 前端白标定制示例
class WhitelabelCaptcha {
    constructor() {
        this.config = null;
    }

    async loadConfig() {
        const response = await fetch('/api/v1/whitelabel/config', {
            headers: {
                'Authorization': `Bearer ${getAdminToken()}`
            }
        });
        this.config = await response.json();
        this.applyStyles();
    }

    applyStyles() {
        if (!this.config || !this.config.data) return;

        const { primary_color, success_color, custom_css } = this.config.data;

        // 应用主色调
        document.documentElement.style.setProperty('--captcha-primary', primary_color);
        document.documentElement.style.setProperty('--captcha-success', success_color);

        // 应用自定义CSS
        if (custom_css) {
            const style = document.createElement('style');
            style.textContent = custom_css;
            document.head.appendChild(style);
        }

        // 更新品牌名称
        if (this.config.data.brand_name) {
            document.title = `${this.config.data.brand_name} - Captcha`;
        }
    }

    async updateConfig(updates) {
        const response = await fetch('/api/v1/whitelabel/config', {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${getAdminToken()}`
            },
            body: JSON.stringify(updates)
        });

        if (response.ok) {
            await this.loadConfig();
        }
    }
}

// 使用示例
const captcha = new WhitelabelCaptcha();
await captcha.loadConfig();

// 更新配置
await captcha.updateConfig({
    brand_name: 'MyBrand',
    primary_color: '#722ed1',
    custom_css: '.captcha-container { border-radius: 12px; }'
});
```

### 场景9：批量验证处理

```go
// Go语言批量验证示例
package main

import (
    "fmt"
    "sync"
    "time"
)

type CaptchaBatch struct {
    client *CaptchaClient
    results chan *VerifyResult
    errors  chan error
}

func NewBatch(client *CaptchaClient) *CaptchaBatch {
    return &CaptchaBatch{
        client:  client,
        results: make(chan *VerifyResult, 100),
        errors:  make(chan error, 100),
    }
}

func (b *CaptchaBatch) ProcessBatch(sessions []string) map[string]*VerifyResult {
    var wg sync.WaitGroup
    results := make(map[string]*VerifyResult)
    var mu sync.Mutex

    for _, sessionID := range sessions {
        wg.Add(1)
        go func(sid string) {
            defer wg.Done()

            result, err := b.client.VerifySliderCaptcha(sid, "160")
            if err != nil {
                b.errors <- fmt.Errorf("session %s: %v", sid, err)
                return
            }

            mu.Lock()
            results[sid] = result
            mu.Unlock()
        }(sessionID)
    }

    wg.Wait()
    close(b.results)
    close(b.errors)

    return results
}

// 使用示例
func main() {
    client := captcha.NewCaptchaClient("app-id", "app-secret", nil)
    defer client.Close()

    batch := NewBatch(client)

    sessions := []string{
        "sess_001", "sess_002", "sess_003",
        "sess_004", "sess_005",
    }

    start := time.Now()
    results := batch.ProcessBatch(sessions)
    duration := time.Since(start)

    fmt.Printf("批量处理完成，耗时: %v\n", duration)
    fmt.Printf("成功处理: %d/%d\n", len(results), len(sessions))

    for sid, result := range results {
        fmt.Printf("Session %s: 成功=%v, 风险分数=%.2f\n",
            sid, result.Success, result.Score)
    }
}
```

### 场景10：高并发场景处理

```go
// Go语言高并发验证码处理
package main

import (
    "context"
    "fmt"
    "golang.org/x/time/rate"
    "sync"
    "time"
)

type RateLimitedClient struct {
    client  *CaptchaClient
    limiter *rate.Limiter
    results chan *VerifyResult
    errors  chan error
}

func NewRateLimitedClient(qps float64) *RateLimitedClient {
    return &RateLimitedClient{
        client:  captcha.NewCaptchaClient("app-id", "app-secret", nil),
        limiter: rate.NewLimiter(rate.Limit(qps), int(qps)),
        results: make(chan *VerifyResult, 1000),
        errors:  make(chan error, 100),
    }
}

func (c *RateLimitedClient) ProcessWithLimit(ctx context.Context, sessions []string) {
    var wg sync.WaitGroup

    for _, sessionID := range sessions {
        select {
        case <-ctx.Done():
            return
        default:
        }

        wg.Add(1)
        go func(sid string) {
            defer wg.Done()

            // 限速控制
            if err := c.limiter.Wait(ctx); err != nil {
                c.errors <- err
                return
            }

            result, err := c.client.VerifySliderCaptcha(sid, "160")
            if err != nil {
                c.errors <- err
                return
            }

            c.results <- result
        }(sessionID)
    }

    wg.Wait()
}

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // 每秒处理100个请求
    client := NewRateLimitedClient(100)

    sessions := make([]string, 1000)
    for i := range sessions {
        sessions[i] = fmt.Sprintf("sess_%04d", i)
    }

    start := time.Now()
    client.ProcessWithLimit(ctx, sessions)
    duration := time.Since(start)

    fmt.Printf("高并发处理完成，耗时: %v\n", duration)
    fmt.Printf("处理速率: %.2f req/s\n", float64(1000)/duration.Seconds())
}
```

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

---

**最后更新**: 2026-05-18
**当前版本**: v15.0
