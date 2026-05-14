# CaptchaX 行为验证系统

<p align="center">
  <img src="https://img.shields.io/badge/Version-2.0.0-blue.svg" alt="Version">
  <img src="https://img.shields.io/badge/License-MIT-green.svg" alt="License">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8.svg" alt="Go">
  <img src="https://img.shields.io/badge/Docker-Ready-2496ED.svg" alt="Docker">
</p>

CaptchaX 是一款高性能、易部署的开源行为验证码系统，提供滑块验证、点选验证和拼图验证三种验证方式，有效防止自动化攻击和机器人恶意行为。

## 核心特性

| 特性 | 说明 |
|------|------|
| **三种验证模式** | 滑块验证、点选验证、拼图验证 |
| **高性能** | 基于 Redis 的分布式缓存，毫秒级响应 |
| **安全防护** | 内置 IP 限流、黑名单/白名单、风险评分引擎 |
| **易于集成** | 提供 RESTful API，支持前后端快速接入 |
| **多语言 SDK** | Go、Python、Java、JavaScript、.NET、PHP、Ruby |
| **管理后台** | 可视化配置面板，支持统计分析和实时监控 |
| **容器化部署** | 支持 Docker 和 Kubernetes 一键部署 |

## 快速开始

### 环境要求

| 软件 | 版本要求 |
|------|----------|
| Go | 1.21+ |
| Redis | 6.0+ |
| PostgreSQL | 13+ |
| Docker | 20.10+ |

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

## 功能概览

### 验证类型

#### 滑块验证
用户通过拖动滑块到正确位置完成验证，适用于登录、注册等场景。

#### 点选验证
用户需要按正确顺序点击指定字符，支持中文字符，适用于高安全场景。

#### 拼图验证
用户将拼图块拖动到正确位置，提供更直观的验证体验。

### 安全特性

- **IP 限流**：防止暴力破解和 DDoS 攻击
- **黑名单/白名单**：灵活的 IP 管理
- **风险评分**：多维度行为分析
- **JWT 认证**：安全的会话管理
- **防自动化**：图像混淆、轨迹检测
- **Circuit Breaker**：熔断保护机制

## API 概览

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/v1/captcha/slider` | POST | 生成滑块验证码 |
| `/api/v1/captcha/slider/verify` | POST | 验证滑块验证码 |
| `/api/v1/captcha/click` | POST | 生成点选验证码 |
| `/api/v1/captcha/click/verify` | POST | 验证点选验证码 |
| `/api/v1/captcha/puzzle` | POST | 生成拼图验证码 |
| `/api/v1/captcha/puzzle/verify` | POST | 验证拼图验证码 |
| `/api/v2/captcha/batch/verify` | POST | 批量验证 |
| `/api/v2/captcha/scenarios` | GET/POST | 场景管理 |
| `/api/v2/captcha/webhook` | POST | Webhook 注册 |

详细文档请参阅 [API 文档](docs/API.md)。

## SDK 接入

### 前端接入示例

```html
<!-- 引入 CaptchaX SDK -->
<script src="/static/captchax.js"></script>

<script>
const captcha = new CaptchaX({
  appId: 'your-app-id',
  serverUrl: 'https://your-captchax-server.com',
  container: '#captcha-container',

  onSuccess: function(result) {
    console.log('验证成功，token:', result.token);
    submitForm(result.token);
  },

  onError: function(error) {
    console.error('验证失败:', error);
  }
});

// 渲染验证码
captcha.render();
</script>
```

### 后端验证示例

```python
from flask import Flask, request, jsonify
from captchax import CaptchaXClient

app = Flask(__name__)
client = CaptchaXClient(app_id='your-app-id', server_url='http://localhost:8080')

@app.route('/api/login', methods=['POST'])
def login():
    token = request.json.get('captchaToken')

    if not client.verify_token(token):
        return jsonify({'error': '验证失败'}), 400

    # 处理登录逻辑...
    return jsonify({'success': True})
```

详细接入指南请参阅 [SDK 文档](docs/SDK.md)。

## 项目结构

```
captchax/
├── cmd/                    # 命令行入口
│   ├── admin/             # 管理后台服务
│   └── server/            # API 服务
├── config/                # 配置文件
├── docs/                  # 文档目录
│   ├── API.md            # API 接口文档
│   ├── SDK.md            # SDK 使用指南
│   ├── DEPLOY.md          # 部署文档
│   ├── FAQ.md             # 常见问题
│   ├── OPS.md             # 运维手册
│   └── openapi.yaml       # OpenAPI 规范
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
├── pkg/                   # 公共包
├── sdk/                   # 多语言 SDK
│   ├── go/               # Go SDK
│   ├── python/            # Python SDK
│   ├── java/              # Java SDK
│   ├── javascript/         # JavaScript SDK
│   ├── dotnet/            # .NET SDK
│   ├── php/               # PHP SDK
│   └── ruby/              # Ruby SDK
├── examples/             # 示例代码
│   ├── frontend/         # 前端示例
│   └── backend/          # 后端示例
├── templates/             # HTML 模板
├── web/                   # 前端资源
│   ├── static/            # 静态资源
│   └── templates/         # 前端模板
├── systemd/               # systemd 配置
├── scripts/               # 脚本
├── tests/                 # 测试
├── Dockerfile
├── docker-compose.yml
└── README.md
```

## 部署指南

### Docker Compose

```bash
docker-compose up -d
```

### Kubernetes

使用 Helm 部署：

```bash
helm repo add captchax https://charts.captchax.io
helm install captchax captchax/captchax -n captchax --create-namespace
```

详细部署说明请参阅 [部署文档](docs/DEPLOY.md)。

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

## 性能基准

| 指标 | 数值 |
|------|------|
| 单次验证响应时间 | < 50ms |
| 并发支持 | 10000+ QPS |
| 内存占用 | < 200MB |
| 缓存命中率 | > 95% |

## 文档

| 文档 | 说明 |
|------|------|
| [API 文档](docs/API.md) | 完整的 API 接口文档 |
| [SDK 指南](docs/SDK.md) | 各语言 SDK 使用指南 |
| [部署文档](docs/DEPLOY.md) | Docker、K8s 部署指南 |
| [运维手册](docs/OPS.md) | 监控、日志、故障排查 |
| [常见问题](docs/FAQ.md) | FAQ 和故障排除 |
| [贡献指南](CONTRIBUTING.md) | 如何贡献代码 |

## 更新日志

详细更新记录请参阅 [CHANGELOG.md](CHANGELOG.md)。

## 贡献指南

欢迎提交 Issue 和 Pull Request！

- 阅读 [贡献指南](CONTRIBUTING.md)
- 提交 Bug 请创建 [Issue](https://github.com/your-org/captchax/issues)
- 提交功能请先讨论

## 开源协议

MIT License - 详见 [LICENSE](LICENSE) 文件。

## 联系方式

- GitHub Issues: [https://github.com/your-org/captchax/issues](https://github.com/your-org/captchax/issues)
- 邮箱: support@example.com
- 讨论组: [GitHub Discussions](https://github.com/your-org/captchax/discussions)

---

<p align="center">
  如果 CaptchaX 对您有帮助，请给我们一个 ⭐️
</p>
