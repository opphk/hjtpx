# 墨盾验证码系统 API 文档

## 增强型滑块验证码 API (v15.0)

### 1. 生成增强型滑块验证码

**接口地址**: `POST /api/v1/captcha/slider/enhanced/generate`

**请求头**:
```json
{
  "Content-Type": "application/json",
  "X-Fingerprint": "用户指纹标识（可选）"
}
```

**请求体**:
```json
{
  "width": 320,
  "height": 200,
  "slider_width": 50,
  "slider_height": 50,
  "difficulty": 1,
  "mode": "standard"
}
```

**参数说明**:
- `width` (int, optional): 验证码图片宽度，默认 320
- `height` (int, optional): 验证码图片高度，默认 200
- `slider_width` (int, optional): 滑块宽度，默认 50
- `slider_height` (int, optional): 滑块高度，默认 50
- `difficulty` (int, optional): 难度等级，1-5，默认随机
  - 1: 简单 - 障碍物少，阻力低
  - 2: 中等 - 障碍物中等，阻力适中
  - 3: 困难 - 障碍物多，阻力较高
  - 4: 极难 - 障碍物很多，阻力很高
  - 5: 地狱 - 最大难度
- `mode` (string, optional): 验证模式
  - `standard`: 标准模式
  - `dual_track`: 双轨模式
  - `multi_obstacle`: 多障碍模式
  - `chaos`: 混沌模式

**成功响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "session_id": "enhanced_1718707200000_1234",
    "background_url": "data:image/png;base64,...",
    "slider_url": "data:image/png;base64,...",
    "gap_x": 150,
    "gap_y": 100,
    "expires_in": 300,
    "expires_at": 1718707500,
    "obstacles": [
      {
        "type": "barrier",
        "x": 80,
        "y": 50,
        "width": 40,
        "height": 40,
        "rotation": 0
      }
    ],
    "trajectory_hint": {
      "suggested_speed": 0.75,
      "path_complexity": 3,
      "hints": [
        {"x": 50, "y": 100},
        {"x": 100, "y": 98},
        {"x": 150, "y": 100}
      ]
    },
    "resistance_level": 2,
    "difficulty": 2,
    "track_info": {
      "upper_track_y": 50,
      "lower_track_y": 150,
      "has_obstacles": true,
      "track_width": 100
    }
  }
}
```

**响应字段说明**:
- `session_id`: 会话 ID，用于后续验证
- `background_url`: 验证码背景图片（Base64 编码）
- `slider_url`: 滑块图片（Base64 编码）
- `gap_x`: 缺口 X 坐标
- `gap_y`: 缺口 Y 坐标
- `expires_in`: 有效期（秒）
- `expires_at`: 过期时间戳
- `obstacles`: 障碍物列表
- `trajectory_hint`: 轨迹提示
- `resistance_level`: 阻力等级（1-5）
- `difficulty`: 难度等级
- `track_info`: 轨道信息（双轨模式下使用）

---

### 2. 验证增强型滑块验证码

**接口地址**: `POST /api/v1/captcha/slider/enhanced/verify`

**请求头**:
```json
{
  "Content-Type": "application/json"
}
```

**请求体**:
```json
{
  "session_id": "enhanced_1718707200000_1234",
  "position_x": 150,
  "position_y": 100,
  "trajectory": [
    {
      "x": 0,
      "y": 25,
      "timestamp": 0,
      "pressure": 0.5,
      "tilt_x": 0,
      "tilt_y": 0
    },
    {
      "x": 10,
      "y": 26,
      "timestamp": 30,
      "pressure": 0.6,
      "tilt_x": 0.1,
      "tilt_y": 0.2
    }
  ],
  "drag_duration": 1500,
  "resistance_level": 2,
  "difficulty": 2,
  "obstacles": [
    {
      "type": "barrier",
      "x": 80,
      "y": 50,
      "width": 40,
      "height": 40,
      "rotation": 0
    }
  ],
  "track_mode": "dual_track"
}
```

**参数说明**:
- `session_id` (string, required): 会话 ID
- `position_x` (int, required): 滑块最终 X 位置
- `position_y` (int, required): 滑块最终 Y 位置
- `trajectory` (array, required): 拖拽轨迹点列表
  - `x`: X 坐标
  - `y`: Y 坐标
  - `timestamp`: 时间戳（毫秒）
  - `pressure`: 压力值（可选，0-1）
  - `tilt_x`: X 轴倾斜（可选）
  - `tilt_y`: Y 轴倾斜（可选）
- `drag_duration` (int64): 拖拽总时长（毫秒）
- `resistance_level` (int): 阻力等级
- `difficulty` (int): 难度等级
- `obstacles` (array): 障碍物列表
- `track_mode` (string): 轨道模式

**成功响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "success": true,
    "message": "验证成功",
    "score": 85.5,
    "position_diff": 3,
    "trajectory_analysis": {
      "is_human": true,
      "confidence": 0.92,
      "anomaly_score": 0.08,
      "speed_profile": {
        "average_speed": 85.5,
        "max_speed": 150.2,
        "min_speed": 20.3,
        "speed_variance": 35.6,
        "speed_trend": "decelerating"
      },
      "acceleration": {
        "average_acceleration": 2.5,
        "max_acceleration": 8.2,
        "min_acceleration": -5.1,
        "jerk_magnitude": 3.2
      },
      "direction_changes": [0.05, 0.08, 0.12, 0.15],
      "pattern": "natural",
      "features": {
        "total_distance": 285.5,
        "direct_distance": 265.0,
        "efficiency": 0.93,
        "curvature": 0.85,
        "sinuosity": 1.08,
        "x_variation": 15.5,
        "y_variation": 8.2,
        "dwell_time": 250,
        "move_time": 1250
      },
      "neural_score": 0.88,
      "lstm_confidence": 0.85,
      "attention_score": 0.82
    },
    "risk_assessment": {
      "overall_risk": 0.15,
      "risk_factors": [
        {
          "type": "ml_detection",
          "score": 0.12,
          "weight": 0.3,
          "detail": "机器学习模型检测正常"
        }
      ],
      "recommendation": "allow",
      "ml_risk_score": 0.12
    },
    "track_validation": {
      "is_valid": true,
      "track_score": 0.95,
      "obstacles_hit": 0,
      "path_quality": 0.95
    }
  }
}
```

**响应字段说明**:
- `success`: 验证是否成功
- `message`: 验证消息
- `score`: 综合分数（0-100）
- `position_diff`: 位置偏差（像素）
- `trajectory_analysis`: 轨迹分析详情
  - `is_human`: 是否判定为人类操作
  - `confidence`: 置信度
  - `anomaly_score`: 异常分数
  - `speed_profile`: 速度特征
  - `acceleration`: 加速度特征
  - `direction_changes`: 方向变化列表
  - `pattern`: 轨迹模式（linear/curved/zigzag/natural/hesitant）
  - `features`: 详细特征
  - `neural_score`: 神经网络分数
  - `lstm_confidence`: LSTM 置信度
  - `attention_score`: 注意力机制分数
- `risk_assessment`: 风险评估
  - `overall_risk`: 总体风险（0-1）
  - `risk_factors`: 风险因素列表
  - `recommendation`: 建议（allow/review/block）
  - `ml_risk_score`: ML 风险分数
- `track_validation`: 轨道验证（双轨模式）
  - `is_valid`: 是否有效
  - `track_score`: 轨道分数
  - `obstacles_hit`: 碰撞障碍物数
  - `path_quality`: 路径质量

---

### 3. 获取验证码状态

**接口地址**: `GET /api/v1/captcha/slider/enhanced/status/:session_id`

**路径参数**:
- `session_id` (string, required): 会话 ID

**成功响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "session_id": "enhanced_1718707200000_1234",
    "status": "pending",
    "verify_count": 0,
    "max_attempts": 3,
    "created_at": "2024-06-18T10:00:00Z",
    "expired_at": "2024-06-18T10:05:00Z",
    "gap_x": 150,
    "gap_y": 100
  }
}
```

---

### 4. 检查验证码有效性

**接口地址**: `GET /api/v1/captcha/slider/enhanced/check/:session_id`

**路径参数**:
- `session_id` (string, required): 会话 ID

**成功响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "valid": true,
    "message": ""
  }
}
```

---

### 5. 获取推荐难度

**接口地址**: `GET /api/v1/captcha/slider/enhanced/difficulty`

**请求头**:
```json
{
  "X-Fingerprint": "用户指纹"
}
```

**成功响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "difficulty": 2,
    "fingerprint": "用户指纹"
  }
}
```

---

## 增强功能说明

### 1. 多层障碍物系统

增强型滑块验证码支持多层障碍物，包括：
- **barrier**: 条形障碍物
- **zigzag**: 锯齿形障碍物
- **curve**: 曲线形障碍物
- **bump**: 凸起障碍物
- **hole**: 空洞障碍物
- **trap**: 陷阱障碍物

### 2. 双轨滑块模式

双轨模式下，系统会生成两条轨道（上层和下层），用户可以选择不同的轨道进行验证。

### 3. 智能缺口检测

使用深度学习算法进行缺口检测，支持：
- 边缘检测（Canny 算子）
- 投影分析
- 自适应平滑

### 4. 自适应阻力系统

根据难度等级和用户指纹动态调整滑块阻力：
- 1 级: 平滑移动
- 2 级: 轻微阻力
- 3 级: 中等阻力
- 4 级: 较高阻力
- 5 级: 高阻力

### 5. 轨迹预测模型

使用多种机器学习模型分析轨迹：
- **神经网络**: 基础轨迹分类
- **LSTM**: 时序数据分析
- **注意力机制**: 关键点识别
- **特征提取**: 多维度特征分析

### 6. 风险评估系统

综合评估用户行为风险，包括：
- 速度异常检测
- 轨迹异常检测
- 机械行为检测
- 机器学习综合评估

---

## 错误代码

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 1001 | 参数错误 |
| 1002 | 会话不存在 |
| 1003 | 验证码已过期 |
| 1004 | 验证次数已用完 |
| 1005 | 服务器错误 |

---

## 使用示例

### 前端集成示例

```javascript
// 1. 生成验证码
async function generateCaptcha() {
    const response = await fetch('/api/v1/captcha/slider/enhanced/generate', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({
            difficulty: 2,
            mode: 'dual_track'
        })
    });
    
    const result = await response.json();
    
    if (result.code === 0) {
        const data = result.data;
        // 显示验证码图片
        document.getElementById('captcha').src = data.background_url;
        // 保存会话 ID
        sessionStorage.setItem('captcha_session', data.session_id);
    }
}

// 2. 验证验证码
async function verifyCaptcha(positionX, positionY, trajectory) {
    const sessionId = sessionStorage.getItem('captcha_session');
    
    const response = await fetch('/api/v1/captcha/slider/enhanced/verify', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({
            session_id: sessionId,
            position_x: positionX,
            position_y: positionY,
            trajectory: trajectory,
            drag_duration: Date.now() - startTime,
            resistance_level: 2,
            difficulty: 2
        })
    });
    
    const result = await response.json();
    
    if (result.data.success) {
        console.log('验证成功！');
    } else {
        console.log('验证失败：' + result.data.message);
    }
}
```

---

## 版本历史

| 版本 | 日期 | 说明 |
|------|------|------|
| v15.0 | 2024-06-18 | 初始版本，增强型滑块验证码 |

---

## 联系我们

如有问题，请联系技术支持团队。
