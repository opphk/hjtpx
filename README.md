# 行为验证系统 (Behavior Verification System)

一个高性能、高安全性的行为验证系统，前后端均使用Go语言开发，目标是超越极验、易盾、五秒盾等现有产品。

## ✨ 核心特性

- 🔐 **多种验证方式**：滑块验证、点选验证、图形验证码
- 🤖 **AI行为分析**：智能识别机器人行为，风险评分系统
- 📊 **完善的管理端**：验证数据统计、配置管理、日志查询
- 🚀 **高性能架构**：基于Gin框架，支持高并发
- 🔒 **安全可靠**：JWT认证、数据加密、防暴力破解

## 🛠️ 技术栈

- **后端**：Go 1.25+ + Gin + GORM
- **数据库**：PostgreSQL + Redis
- **前端**：原生HTML5 + CSS3 + JavaScript
- **图表**：Chart.js

## 📁 项目结构

```
hjtpx/
├── backend/                    # 后端服务
│   ├── cmd/                    # 程序入口
│   ├── internal/               # 内部代码
│   │   └── api/
│   │       ├── handler/        # API处理器
│   │       ├── middleware/     # 中间件
│   │       ├── router/         # 路由
│   │       └── service/        # 业务逻辑
│   └── pkg/                    # 公共包
├── frontend/                   # 用户端
│   ├── static/
│   └── templates/
├── admin/                      # 管理端
│   ├── static/
│   └── templates/
└── 开发核心.md
```

## 🚀 快速开始

### 环境要求

- Go 1.25+
- PostgreSQL 12+
- Redis 6+

### 安装依赖

```bash
cd backend
go mod download
```

### 配置

修改配置文件或设置环境变量：

```bash
# 数据库配置
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=yourpassword
POSTGRES_DB=verification

# Redis配置
REDIS_HOST=localhost
REDIS_PORT=6379

# JWT配置
JWT_SECRET=your-secret-key
JWT_EXPIRE_HOURS=24
```

### 运行服务

```bash
cd backend
go run ./cmd/server
```

服务将在 http://localhost:8080 启动

## 📡 API接口文档

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

## 👤 默认账号

- **用户名**：admin
- **密码**：admin123

## 🎯 核心功能

### 1. 滑块验证

- 动态生成带缺口的背景图
- 实时验证滑动位置
- 行为数据收集和分析

### 2. 点选验证

- 随机生成目标字符
- 支持多种难度级别
- 智能干扰项生成

### 3. 图形验证码

- 支持数字、字母、混合模式
- 干扰线和噪点
- 可配置长度和复杂度

### 4. 行为分析

- 鼠标轨迹分析
- 点击模式识别
- 风险评分系统
- 实时风险判断

### 5. 管理功能

- 验证数据统计
- 图表可视化
- 应用配置管理
- 日志查询和分析

## 📊 开发进度

- ✅ 项目初始化
- ✅ 数据库设计与搭建
- ✅ 后端基础框架搭建
- ✅ 用户端API开发
- ✅ 管理端API开发
- ✅ 验证核心算法开发
- ✅ 用户端界面开发
- ✅ 管理端界面开发
- 🔄 集成测试
- ⏳ 部署上线

## 🔐 安全特性

- JWT Token认证
- 接口访问频率限制
- SQL注入防护
- XSS攻击防护
- 数据加密存储
- 操作日志记录

## 📈 性能指标

- 响应时间：< 200ms
- 支持并发：10000+ QPS
- 可用性：99.9%
- 验证码有效期：5分钟

## 📝 许可证

MIT License

## 🤝 贡献

欢迎提交Issue和Pull Request！

## 📧 联系方式

如有问题，请联系：3395587255@qq.com

---

**GitHub仓库**：https://github.com/opphk/behavior-verification-system
