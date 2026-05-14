# CaptchaX 行为验证系统

CaptchaX 是一款高性能、易部署的开源行为验证码系统，提供滑块验证、点选验证和拼图验证三种验证方式，有效防止自动化攻击和机器人恶意行为。

## 核心特性

- **三种验证模式**：滑块验证、点选验证、拼图验证
- **高性能**：基于 Redis 的分布式缓存，毫秒级响应
- **安全防护**：内置 IP 限流、黑名单/白名单机制、风险评分引擎
- **易于集成**：提供 RESTful API，支持前后端快速接入
- **管理后台**：可视化配置面板，支持统计分析和实时监控
- **容器化部署**：支持 Docker 一键部署

## 技术架构

```
┌─────────────────────────────────────────────────────────────┐
│                         前端应用                             │
└─────────────────────────┬───────────────────────────────────┘
                          │ HTTP/HTTPS
┌─────────────────────────▼───────────────────────────────────┐
│                     CaptchaX Server                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │  API 服务    │  │  管理后台    │  │  静态资源    │          │
│  └──────┬──────┘  └──────┬──────┘  └─────────────┘          │
│         │                │                                    │
│  ┌──────▼────────────────▼──────┐                           │
│  │         Captcha Service       │                           │
│  │   ┌─────────┐ ┌─────────┐   │                           │
│  │   │ Slider  │ │  Click  │   │                           │
│  │   │ Puzzle  │ │ Engine  │   │                           │
│  │   └─────────┘ └─────────┘   │                           │
│  └─────────────────────────────┘                           │
└─────────────────────────────────────────────────────────────┘
                          │
         ┌────────────────┼────────────────┐
         │                │                │
    ┌────▼────┐     ┌────▼────┐     ┌────▼────┐
    │  Redis  │     │ Postgres│     │  文件   │
    │  缓存    │     │  数据库  │     │  存储   │
    └─────────┘     └─────────┘     └─────────┘
```

## 快速开始

### 环境要求

- Go 1.21+
- Redis 6.0+
- PostgreSQL 13+
- Docker & Docker Compose (可选)

### Docker 部署（推荐）

```bash
# 克隆项目
git clone https://github.com/your-org/captchax.git
cd captchax

# 启动服务
docker-compose up -d

# 访问管理后台
open http://localhost:8080/admin/login
```

默认管理员账号：`admin` / `admin123`

### 手动部署

```bash
# 1. 安装依赖
go mod download

# 2. 配置数据库
# 编辑 config/config.yaml

# 3. 运行数据库迁移
psql -h localhost -U postgres -d captcha_db -f migrations/001_initial_schema.sql

# 4. 启动服务
go run cmd/server/main.go
go run cmd/admin/main.go
```

## API 概览

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/v1/captcha/slider` | POST | 生成滑块验证码 |
| `/api/v1/captcha/slider/verify` | POST | 验证滑块验证码 |
| `/api/v1/captcha/click` | POST | 生成点选验证码 |
| `/api/v1/captcha/click/verify` | POST | 验证点选验证码 |
| `/api/v1/captcha/puzzle` | POST | 生成拼图验证码 |
| `/api/v1/captcha/puzzle/verify` | POST | 验证拼图验证码 |

详细文档请参阅 [API 文档](docs/API.md)。

## SDK 接入

前端接入示例：

```html
<!-- 引入 CaptchaX SDK -->
<script src="/static/captchax.js"></script>

<script>
const captcha = new CaptchaX({
  appId: 'your-app-id',
  serverUrl: 'https://your-captchax-server.com',
  onSuccess: function(token) {
    // 提交表单，将 token 发送到后端验证
    console.log('验证成功，token:', token);
  },
  onError: function(error) {
    console.error('验证失败:', error);
  }
});

// 渲染验证码
captcha.render('#captcha-container');
</script>
```

详细接入指南请参阅 [SDK 文档](docs/SDK.md)。

## 配置说明

主要配置项（`config/config.yaml`）：

```yaml
server:
  host: "0.0.0.0"
  port: 8080

database:
  host: "localhost"
  port: 5432
  user: "captcha_admin"
  password: "your-password"
  dbname: "captcha_db"

redis:
  host: "localhost"
  port: 6379
  password: ""

captcha:
  expire_minutes: 5      # 验证码有效期
  max_attempts: 3        # 最大验证次数
  width: 200             # 滑块宽度
  height: 80             # 滑块高度
  slider_size: 50        # 滑块大小
  tolerance: 5           # 容差范围

admin:
  jwt_secret: "your-secret"
  token_ttl_seconds: 86400
```

## 目录结构

```
captchax/
├── cmd/                    # 命令行入口
│   ├── admin/             # 管理后台服务
│   └── server/            # API 服务
├── config/                # 配置文件
├── docs/                  # 文档目录
├── internal/              # 内部包
│   ├── admin/             # 管理后台
│   ├── api/               # API 处理
│   ├── captcha/           # 验证码核心
│   │   ├── slider/        # 滑块验证
│   │   ├── click/         # 点选验证
│   │   └── puzzle/        # 拼图验证
│   ├── middleware/         # 中间件
│   ├── model/             # 数据模型
│   ├── repository/        # 数据访问层
│   ├── risk/              # 风险控制
│   └── service/           # 业务服务
├── migrations/            # 数据库迁移
├── pkg/                    # 公共包
├── templates/              # HTML 模板
└── web/                   # 前端资源
    ├── static/            # 静态资源
    └── templates/         # 前端模板
```

## 安全特性

- **IP 限流**：防止暴力破解
- **黑名单/白名单**：灵活的 IP 管理
- **风险评分**：多维度行为分析
- **JWT 认证**：安全的会话管理
- **防自动化**：图像混淆、轨迹检测

## 部署指南

详细的部署说明请参阅：

- [部署文档](docs/DEPLOY.md) - 环境配置、Docker 部署、手动部署
- [管理后台使用手册](docs/ADMIN.md) - 登录、配置、统计分析
- [常见问题](docs/FAQ.md) - FAQ 和故障排除

## 贡献指南

欢迎提交 Issue 和 Pull Request！

## 开源协议

MIT License
