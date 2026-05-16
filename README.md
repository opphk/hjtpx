# 行为验证系统 (Behavior Verification System)

一个高性能、高安全性的行为验证系统，采用前后端分离架构，支持多种验证码类型和智能行为分析，目标是超越极验、易盾、五秒盾等现有产品。

## 核心特性

- **多种验证码类型**：滑块验证、点选验证、图形验证码
- **智能行为分析**：AI驱动的鼠标轨迹分析、风险评分系统
- **完善管理端**：验证数据统计、配置管理、日志查询、风控规则
- **高性能架构**：基于Go/Gin框架，支持高并发场景
- **安全可靠**：JWT认证、数据加密、防暴力破解、多层防护
- **完整SDK支持**：提供Go SDK，便于快速集成
- **容器化部署**：Docker/Docker Compose一键部署

## 技术栈

### 后端
- **语言**：Go 1.25+
- **框架**：Gin Web Framework
- **ORM**：GORM
- **数据库**：PostgreSQL 12+ / Redis 6+
- **认证**：JWT (JSON Web Token)

### 前端
- **用户端**：原生 HTML5 + CSS3 + JavaScript
- **管理端**：Bootstrap 5 + Font Awesome 6
- **图表**：Chart.js 4.x
- **CDN**：BootCDN (稳定、快速)

### 基础设施
- **Web服务器**：Nginx 1.25
- **监控**：Prometheus + Grafana + Loki
- **日志收集**：Promtail + Loki
- **容器化**：Docker + Docker Compose

## 项目结构

```
hjtpx/
├── backend/                    # 后端服务
│   ├── cmd/                    # 程序入口
│   │   └── api/
│   │       └── main.go         # API服务入口
│   ├── internal/               # 内部代码
│   │   └── api/
│   │       ├── handler/        # API处理器
│   │       ├── middleware/     # 中间件
│   │       ├── router/         # 路由配置
│   │       └── service/        # 业务逻辑
│   └── pkg/                    # 公共包
│       ├── config/            # 配置管理
│       ├── database/          # 数据库连接
│       ├── models/            # 数据模型
│       ├── postgres/          # PostgreSQL
│       ├── redis/             # Redis
│       ├── response/          # 统一响应
│       ├── jwt/               # JWT认证
│       ├── crypto/            # 加密工具
│       ├── metrics/           # 指标收集
│       └── storage/           # 存储服务
├── frontend/                   # 用户端
│   ├── static/
│   │   └── js/
│   │       ├── captcha.js     # 验证码前端逻辑
│   │       └── main.js        # 主逻辑
│   └── templates/
│       ├── captcha.html       # 验证码页面
│       └── home.html          # 首页
├── admin/                      # 管理端
│   ├── static/
│   │   └── js/
│   │       ├── dashboard.js   # 仪表盘
│   │       ├── stats.js       # 统计图表
│   │       ├── applications.js # 应用管理
│   │       ├── logs.js        # 日志查询
│   │       ├── blacklist.js   # 黑名单管理
│   │       ├── risk-rules.js  # 风控规则
│   │       ├── auth.js        # 认证
│   │       └── main.js        # 主逻辑
│   └── templates/
│       ├── base.html          # 基础模板
│       ├── login.html         # 登录页
│       ├── dashboard.html     # 仪表盘
│       ├── stats.html         # 统计页
│       ├── applications.html  # 应用管理
│       ├── logs.html          # 日志查询
│       ├── blacklist.html     # 黑名单
│       └── risk-rules.html    # 风控规则
├── sdk/                        # 客户端SDK
│   └── go/
│       ├── captcha.go         # 验证码SDK
│       ├── captcha_test.go    # SDK测试
│       └── examples/          # 使用示例
├── monitoring/                  # 监控配置
│   ├── prometheus/            # Prometheus配置
│   ├── grafana/               # Grafana配置
│   ├── loki/                  # Loki配置
│   └── promtail/              # Promtail配置
├── nginx/                      # Nginx配置
│   ├── nginx.conf             # 主配置
│   └── conf.d/
│       └── default.conf       # 默认站点配置
├── scripts/                    # 部署脚本
│   ├── deploy.sh              # 部署脚本
│   ├── backup.sh              # 备份脚本
│   ├── update.sh              # 更新脚本
│   └── health-check.sh        # 健康检查
├── .env.example               # 环境变量示例
├── docker-compose.yml         # Docker编排
├── Dockerfile                 # 容器构建
└── README.md
```

## 快速开始

### 环境要求

- **Go**: 1.25+
- **PostgreSQL**: 12+
- **Redis**: 6+
- **Docker**: 20.10+ (可选)

### 方式一：Docker部署（推荐）

1. 克隆项目
```bash
git clone https://github.com/opphk/behavior-verification-system.git
cd behavior-verification-system
```

2. 复制环境变量文件
```bash
cp .env.example .env
```

3. 修改 `.env` 文件配置（必填项）
```bash
# 数据库密码
POSTGRES_PASSWORD=your-secure-password

# Redis密码
REDIS_PASSWORD=your-redis-password

# JWT密钥（至少32字符）
JWT_SECRET=your-very-secure-jwt-secret-key-min-32-chars

# Grafana密码
GRAFANA_ADMIN_PASSWORD=your-grafana-password
```

4. 启动服务
```bash
docker-compose up -d
```

5. 访问服务
- 用户端验证码：`http://localhost:8080`
- 管理后台：`http://localhost:8080/admin`
- Prometheus：`http://localhost:9090`
- Grafana：`http://localhost:3000`

### 方式二：本地开发

1. 克隆项目
```bash
git clone https://github.com/opphk/behavior-verification-system.git
cd behavior-verification-system
```

2. 安装依赖
```bash
cd backend
go mod download
```

3. 配置环境变量或配置文件

**方式A：环境变量**
```bash
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5432
export POSTGRES_USER=postgres
export POSTGRES_PASSWORD=yourpassword
export POSTGRES_DB=verification

export REDIS_HOST=localhost
export REDIS_PORT=6379
export REDIS_PASSWORD=

export JWT_SECRET=your-secret-key
export JWT_EXPIRE_HOURS=24
```

**方式B：配置文件**
```bash
cp backend/config/config.yaml.example backend/config/config.yaml
# 编辑 backend/config/config.yaml
```

4. 初始化数据库
```bash
# 创建数据库
psql -U postgres -c "CREATE DATABASE verification;"

# 执行初始化脚本
psql -U postgres -d verification -f scripts/init-db.sql
```

5. 运行服务
```bash
cd backend
go run ./cmd/api/main.go
```

6. 访问服务
- 服务地址：`http://localhost:8080`
- 用户端：`http://localhost:8080`
- 管理后台：`http://localhost:8080/admin`
- 健康检查：`http://localhost:8080/health`

## 默认账号

- **用户名**：`admin`
- **密码**：`admin123`

> ⚠️ **重要**：生产环境请务必修改默认密码！

## API接口文档

### 用户端验证API

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/v1/captcha/slider/generate` | 生成滑块验证码 |
| POST | `/api/v1/captcha/slider/verify` | 验证滑块 |
| POST | `/api/v1/captcha/click/generate` | 生成点选验证码 |
| POST | `/api/v1/captcha/click/verify` | 验证点选 |
| GET | `/api/v1/captcha/image` | 生成图形验证码 |
| POST | `/api/v1/captcha/image/verify` | 验证图形验证码 |

### 管理端API

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/v1/auth/login` | 管理员登录 |
| POST | `/api/v1/admin/logout` | 管理员登出 |
| GET | `/api/v1/admin/stats/verification` | 验证统计 |
| GET | `/api/v1/admin/stats/chart` | 图表数据 |
| GET | `/api/v1/admin/applications` | 应用列表 |
| POST | `/api/v1/admin/applications` | 创建应用 |
| PUT | `/api/v1/admin/applications/:id` | 更新应用 |
| DELETE | `/api/v1/admin/applications/:id` | 删除应用 |
| GET | `/api/v1/admin/logs` | 验证日志 |
| GET | `/api/v1/admin/logs/:id` | 日志详情 |

## 🛠️ Go SDK 使用指南

### 安装

```bash
go get github.com/hjtpx/hjtpx/sdk/go
```

### 快速开始

```go
package main

import (
    "fmt"
    captcha "github.com/hjtpx/hjtpx/sdk/go"
)

func main() {
    client := captcha.NewClient(
        captcha.WithEndpoint("http://localhost:8080"),
        captcha.WithAPIKey("your-api-key"),
        captcha.WithAPISecret("your-api-secret"),
    )

    resp, err := client.GenerateImageCaptcha(&captcha.ImageCaptchaRequest{
        Type:  captcha.CaptchaTypeMixed,
        Count: 4,
    })
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    fmt.Printf("Challenge ID: %s\n", resp.ChallengeID)

    verifyResp, err := client.VerifyImageCaptcha(&captcha.VerifyImageCaptchaRequest{
        ChallengeID: resp.ChallengeID,
        Answer:     "user-input",
    })
    if err != nil {
        fmt.Printf("Verification error: %v\n", err)
        return
    }

    fmt.Printf("Verification result: %v\n", verifyResp.Success)
}
```

### 客户端配置

```go
client := captcha.NewClient(
    captcha.WithEndpoint("http://localhost:8080"),
    captcha.WithAPIKey("your-api-key"),
    captcha.WithAPISecret("your-api-secret"),
    captcha.WithAppID("your-app-id"),
    captcha.WithAppSecret("your-app-secret"),
    captcha.WithTimeout(30 * time.Second),
    captcha.WithDebugMode(true),
    captcha.WithSignatureKey("your-signature-key"),
    captcha.WithRetryConfig(&captcha.RetryConfig{
        MaxRetries:     3,
        InitialDelay:   100 * time.Millisecond,
        MaxDelay:       5 * time.Second,
        BackoffFactor:  2.0,
        RetryableCodes: []int{429, 500, 502, 503, 504},
    }),
)
```

### 图形验证码

```go
req := &captcha.ImageCaptchaRequest{
    Type:      captcha.CaptchaTypeMixed,
    Count:     6,
    NoiseMode: 3,
    LineMode:  2,
}

resp, err := client.GenerateImageCaptcha(req)
if err != nil {
    log.Fatal(err)
}

imageData, err := client.ExtractBase64Image(resp.Image)
if err != nil {
    log.Fatal(err)
}

err = os.WriteFile("captcha.png", imageData, 0644)

verifyResp, err := client.VerifyImageCaptcha(&captcha.VerifyImageCaptchaRequest{
    ChallengeID: resp.ChallengeID,
    Answer:     "a1b2c3",
})
```

### 滑块验证码

```go
req := &captcha.SliderCaptchaRequest{
    Width:  300,
    Height: 200,
}

resp, err := client.GetSliderCaptcha(req)
if err != nil {
    log.Fatal(err)
}

bgData, _ := client.ExtractBase64Image(resp.BackgroundImage)
sliderData, _ := client.ExtractBase64Image(resp.SliderImage)

verifyResp, err := client.VerifyCaptcha(&captcha.VerifyCaptchaRequest{
    ChallengeID: resp.ChallengeID,
    Action:     "slide",
    Data: map[string]interface{}{
        "offset": 120,
        "trajectory": []map[string]interface{}{
            {"x": 0, "y": 0, "timestamp": 0},
            {"x": 30, "y": 5, "timestamp": 100},
            {"x": 60, "y": 10, "timestamp": 200},
        },
    },
})
```

### 点选验证码

```go
req := &captcha.ClickCaptchaRequest{
    Width:     400,
    Height:    300,
    IconCount: 9,
}

resp, err := client.GetClickCaptcha(req)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Target index: %d\n", resp.TargetIndex)
fmt.Printf("Icon positions: %v\n", resp.IconPositions)

verifyResp, err := client.VerifyCaptcha(&captcha.VerifyCaptchaRequest{
    ChallengeID: resp.ChallengeID,
    Action:     "click",
    Data: map[string]interface{}{
        "click_index": resp.TargetIndex,
    },
})
```

### 请求签名

```go
client := captcha.NewClient(
    captcha.WithSignatureKey("your-signature-key"),
)

client.EnableSignature(true)

signature := client.GenerateSignature("POST", "/api/v1/captcha/verify",
    map[string]string{"key": "value"},
    []byte(`{"test":"data"}`))

fmt.Printf("Signature: %s\n", signature)
```

### 错误处理

```go
_, err := client.VerifyImageCaptcha(nil)
if err != nil {
    if sdkErr, ok := err.(*captcha.SDKError); ok {
        fmt.Printf("Error code: %d\n", sdkErr.Code)
        fmt.Printf("Error message: %s\n", sdkErr.Message)
        
        switch {
        case errors.Is(sdkErr, captcha.ErrNilRequest):
            fmt.Println("Request is nil")
        case errors.Is(sdkErr, captcha.ErrMissingChallenge):
            fmt.Println("Challenge ID is missing")
        case errors.Is(sdkErr, captcha.ErrMissingAnswer):
            fmt.Println("Answer is missing")
        }
    }
}
```

### 重试配置

```go
retryConfig := captcha.DefaultRetryConfig()
retryConfig.MaxRetries = 5
retryConfig.InitialDelay = 200 * time.Millisecond
retryConfig.MaxDelay = 10 * time.Second
retryConfig.BackoffFactor = 2.0
retryConfig.RetryableCodes = []int{429, 500, 502, 503, 504}

client := captcha.NewClient(
    captcha.WithRetryConfig(retryConfig),
)

delay := retryConfig.NextDelay(0)
fmt.Printf("Delay for attempt 0: %v\n", delay)

shouldRetry := retryConfig.ShouldRetry(500)
fmt.Printf("Should retry 500: %v\n", shouldRetry)
```

### Mock服务器

```go
mock := captcha.NewMockServer(18080)
if err := mock.Start(); err != nil {
    log.Fatal(err)
}
defer mock.Stop()

time.Sleep(100 * time.Millisecond)

client := captcha.NewClient(
    captcha.WithEndpoint("http://localhost:18080"),
)

resp, err := client.GenerateImageCaptcha(nil)
if err != nil {
    log.Fatal(err)
}

mock.SetCorrectAnswer("test-answer")

verifyResp, err := client.VerifyImageCaptcha(&captcha.VerifyImageCaptchaRequest{
    ChallengeID: "test-id",
    Answer:     "test-answer",
})
fmt.Printf("Result: %v\n", verifyResp.Success)
fmt.Printf("Verify calls: %d\n", mock.VerifyCalls)
```

### 环境变量配置

```bash
export CAPTCHA_ENDPOINT="http://localhost:8080"
export CAPTCHA_API_KEY="your-api-key"
export CAPTCHA_API_SECRET="your-api-secret"
```

```go
client := captcha.NewClient(
    captcha.WithEndpoint(os.Getenv("CAPTCHA_ENDPOINT")),
    captcha.WithAPIKey(os.Getenv("CAPTCHA_API_KEY")),
    captcha.WithAPISecret(os.Getenv("CAPTCHA_API_SECRET")),
    captcha.WithTimeout(30 * time.Second),
    captcha.WithRetryConfig(&captcha.RetryConfig{
        MaxRetries:     3,
        InitialDelay:   100 * time.Millisecond,
        MaxDelay:       5 * time.Second,
        BackoffFactor:  2.0,
        RetryableCodes: []int{429, 500, 502, 503, 504},
    }),
)
```

### SDK API 参考

#### 类型定义

| 类型 | 描述 |
|------|------|
| `SDKError` | 自定义错误类型，包含错误码和消息 |
| `RetryConfig` | 重试配置 |
| `ImageCaptchaRequest` | 图形验证码请求 |
| `ImageCaptchaResponse` | 图形验证码响应 |
| `SliderCaptchaRequest` | 滑块验证码请求 |
| `SliderCaptchaResponse` | 滑块验证码响应 |
| `ClickCaptchaRequest` | 点选验证码请求 |
| `ClickCaptchaResponse` | 点选验证码响应 |
| `VerifyCaptchaRequest` | 验证请求 |
| `VerifyCaptchaResponse` | 验证响应 |

#### 客户端选项

| 选项 | 描述 |
|------|------|
| `WithAPIKey(key)` | 设置API密钥 |
| `WithAPISecret(secret)` | 设置API密钥 |
| `WithAppID(id)` | 设置应用ID |
| `WithAppSecret(secret)` | 设置应用密钥 |
| `WithEndpoint(endpoint)` | 设置API端点 |
| `WithTimeout(timeout)` | 设置请求超时时间 |
| `WithDebugMode(debug)` | 启用调试模式 |
| `WithSignatureKey(key)` | 设置签名密钥 |
| `WithRetryConfig(config)` | 设置重试配置 |

### 用户认证API

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/v1/auth/register` | 用户注册 |
| POST | `/api/v1/auth/login` | 用户登录 |
| POST | `/api/v1/auth/logout` | 用户登出 |
| POST | `/api/v1/auth/refresh` | 刷新Token |
| POST | `/api/v1/auth/request-password-reset` | 请求密码重置 |
| POST | `/api/v1/auth/reset-password` | 重置密码 |
| GET | `/api/v1/auth/verify-email` | 验证邮箱 |

**用户登录**
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "user@example.com",
  "password": "password123"
}
```

响应示例：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_at": "2024-05-16T12:00:00Z"
  }
}
```

### 用户资料API

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v1/user/profile` | 获取用户资料 |
| PUT | `/api/v1/user/profile` | 更新用户资料 |
| POST | `/api/v1/user/change-password` | 修改密码 |

### 管理端API

> ⚠️ 需要管理员认证

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/v1/auth/login` | 管理员登录 |
| POST | `/api/v1/admin/logout` | 管理员登出 |
| GET | `/api/v1/admin/stats/verification` | 验证统计 |
| GET | `/api/v1/admin/stats/chart` | 图表数据 |
| GET | `/api/v1/admin/stats/trend` | 趋势数据 |
| GET | `/api/v1/admin/stats/realtime` | 实时监控 |
| GET | `/api/v1/admin/stats/risk-distribution` | 风险分布 |
| GET | `/api/v1/admin/applications` | 应用列表 |
| POST | `/api/v1/admin/applications` | 创建应用 |
| PUT | `/api/v1/admin/applications/:id` | 更新应用 |
| DELETE | `/api/v1/admin/applications/:id` | 删除应用 |
| GET | `/api/v1/admin/logs` | 验证日志 |
| GET | `/api/v1/admin/logs/:id` | 日志详情 |
| GET | `/api/v1/admin/logs/export` | 导出日志CSV |
| GET | `/api/v1/admin/blacklist` | 黑名单列表 |
| POST | `/api/v1/admin/blacklist` | 添加黑名单 |
| DELETE | `/api/v1/admin/blacklist/:id` | 删除黑名单 |

**验证统计**
```http
GET /api/v1/admin/stats/verification
Authorization: Bearer <admin_token>
```

响应示例：
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

## 核心功能

### 1. 滑块验证

- 动态生成带缺口的SVG背景图
- 实时验证滑动位置（容差±10px）
- 鼠标轨迹采集与分析
- 行为数据完整性校验

### 2. 点选验证

- 支持多种字符模式（数字、字母、中文、混合）
- 可配置目标点数量（2-6个）
- 顺序打乱功能
- 点击时序验证

### 3. 图形验证码

- 支持多种字符类型组合
- 可配置字符数量
- 噪点干扰
- 干扰线覆盖
- 扭曲变形效果

### 4. 行为分析

- **鼠标轨迹分析**：
  - 总移动距离
  - 平均/最大/最小速度
  - 路径效率
  - 方向变化次数
  - 轨迹平滑度

- **点击模式分析**：
  - 点击间隔规律性
  - 点击速度稳定性
  - 点击位置聚类

- **风险评分**：
  - 综合风险分数（0-100）
  - 多维度风险因子
  - Bot行为识别

### 5. 管理功能

- **仪表盘**：实时验证数据概览
- **统计图表**：多维度数据可视化
- **应用管理**：创建/配置/管理应用
- **日志查询**：完整的验证日志
- **黑名单**：IP/用户黑名单管理
- **风控规则**：配置风险阈值和规则

## 安全特性

- JWT Token认证
- 接口访问频率限制（IP/用户/应用级别）
- SQL注入防护
- XSS攻击防护
- CSRF跨站请求伪造防护
- API签名验证
- 数据加密存储
- 操作日志审计
- IP白名单/黑名单
- CDN回源认证

## 使用示例

### Go SDK 使用示例

```go
package main

import (
    "fmt"
    "log"

    "github.com/opphk/behavior-verification-system/sdk/go/captcha"
)

func main() {
    // 创建客户端
    client := captcha.NewClient(
        captcha.WithEndpoint("http://localhost:8080"),
        captcha.WithTimeout(30 * time.Second),
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

    // 模拟用户验证（滑块位置）
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
    fmt.Printf("验证结果: %v, 风险分数: %.2f\n", verifyResp.Success, verifyResp.Score)
}
```

### JavaScript 前端集成

```html
<!DOCTYPE html>
<html>
<head>
    <title>滑块验证示例</title>
</head>
<body>
    <div id="captcha-container"></div>
    <button onclick="initCaptcha()">加载验证码</button>

    <script>
        async function initCaptcha() {
            // 获取验证码
            const resp = await fetch('/api/v1/captcha/slider');
            const { data } = await resp.json();

            // 显示验证码图片
            document.getElementById('captcha-container').innerHTML = `
                <img src="${data.image_url}" />
                <p>请拖动滑块完成验证</p>
            `;

            // 收集行为数据
            const behaviorData = [];
            let isDragging = false;

            document.querySelector('img').addEventListener('mousedown', (e) => {
                isDragging = true;
                behaviorData.push({
                    x: e.offsetX,
                    y: e.offsetY,
                    timestamp: Date.now(),
                    event: 'mousedown'
                });
            });

            document.addEventListener('mousemove', (e) => {
                if (isDragging) {
                    behaviorData.push({
                        x: e.clientX,
                        y: e.clientY,
                        timestamp: Date.now(),
                        event: 'move'
                    });
                }
            });

            document.addEventListener('mouseup', async (e) => {
                if (isDragging) {
                    isDragging = false;
                    behaviorData.push({
                        x: e.clientX,
                        y: e.clientY,
                        timestamp: Date.now(),
                        event: 'mouseup'
                    });

                    // 提交验证
                    const verifyResp = await fetch('/api/v1/captcha/verify', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({
                            session_id: data.session_id,
                            type: 'slider',
                            x: e.clientX,
                            y: data.puzzle_y,
                            behavior_data: behaviorData
                        })
                    });

                    const result = await verifyResp.json();
                    if (result.data.success) {
                        alert('验证成功！');
                    } else {
                        alert('验证失败: ' + result.data.message);
                    }
                }
            });
        }
    </script>
</body>
</html>
```

## 性能指标

- **响应时间**：< 200ms（P99）
- **并发支持**：10000+ QPS
- **可用性**：99.9%
- **验证码有效期**：5分钟
- **会话清理**：自动清理10分钟以上的过期会话

## 开发进度

| 功能 | 状态 |
|------|------|
| 项目初始化 | ✅ 完成 |
| 数据库设计与搭建 | ✅ 完成 |
| 后端基础框架搭建 | ✅ 完成 |
| 用户端API开发 | ✅ 完成 |
| 管理端API开发 | ✅ 完成 |
| 验证核心算法开发 | ✅ 完成 |
| 用户端界面开发 | ✅ 完成 |
| 管理端界面开发 | ✅ 完成 |
| 滑块验证码增强 | ✅ 完成 |
| 点选验证码增强 | ✅ 完成 |
| 图形验证码增强 | ✅ 完成 |
| 行为分析算法 | ✅ 完成 |
| 用户认证系统 | ✅ 完成 |
| 应用管理系统 | ✅ 完成 |
| 日志分析统计 | ✅ 完成 |
| API限流 | ✅ 完成 |
| 安全加固 | ✅ 完成 |
| Docker部署 | ✅ 完成 |
| 监控配置 | ✅ 完成 |
| Go SDK | ✅ 完成 |
| 集成测试 | 🔄 进行中 |
| 部署上线 | ⏳ 待开始 |

## 贡献指南

欢迎提交Issue和Pull Request！

### 提交规范

请遵循 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

```
feat: 新功能
fix: 修复Bug
docs: 文档更新
style: 代码格式（不影响功能）
refactor: 重构
perf: 性能优化
test: 测试相关
chore: 构建/工具相关
```

### 开发流程

1. Fork 本仓库
2. 创建特性分支 `git checkout -b feat/your-feature`
3. 提交更改 `git commit -m 'feat: add new feature'`
4. 推送分支 `git push origin feat/your-feature`
5. 创建 Pull Request

### 代码规范

- Go代码遵循 `gofmt` 格式化
- 变量命名清晰、见名知意
- 关键函数/方法添加注释
- 提交前运行测试

## 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

## 联系方式

- **邮箱**：3395587255@qq.com
- **GitHub**：https://github.com/opphk/behavior-verification-system

## 致谢

感谢以下开源项目：

- [Gin](https://github.com/gin-gonic/gin) - Go Web框架
- [GORM](https://gorm.io/) - Go ORM库
- [Bootstrap](https://getbootstrap.com/) - 前端框架
- [Chart.js](https://www.chartjs.org/) - 图表库
- [Prometheus](https://prometheus.io/) - 监控系统
- [Grafana](https://grafana.com/) - 可视化平台

---

**GitHub仓库**：https://github.com/opphk/behavior-verification-system
