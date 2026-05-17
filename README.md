# HJTPX - 行为验证系统

## 项目介绍

HJTPX是一个高性能、高安全性的行为验证系统，采用前后端分离架构，前后端均使用Go语言开发。目标是超越极验、易盾、五秒盾等现有产品。

## 核心功能

- 滑块验证码
- 点选验证码
- 旋转验证码
- 拼图验证码
- 手势验证码
- AI行为分析
- 环境检测
- 无感验证
- 管理后台

## 技术栈

- 后端：Go + Gin + GORM
- 数据库：PostgreSQL + Redis
- 前端：HTML5 + JavaScript + Bootstrap 5
- UI框架：AdminLTE 3.2
- 监控：Prometheus + Grafana + Loki

## 快速开始

### 环境要求

- Go 1.21+
- PostgreSQL 12+
- Redis 6+

### 安装部署

1. 克隆代码
```bash
git clone https://github.com/opphk/hjtpx.git
cd hjtpx
```

2. 配置数据库
```bash
cp .env.example .env
# 编辑 .env 文件配置数据库信息
```

3. 使用 Docker Compose 启动
```bash
docker-compose up -d
```

### 默认访问地址

| 服务 | 地址 | 说明 |
|------|------|------|
| 应用服务 | http://localhost:8080 | API 服务 |
| 用户端 | http://localhost | 前端页面 |
| 管理后台 | http://localhost/admin | 管理后台 |
| 健康检查 | http://localhost:8080/health | 健康检查端点 |

### 默认账号

- 用户名：admin
- 密码：admin123

## 项目结构

```
hjtpx/
├── backend/                    # 后端服务
│   ├── cmd/                   # 程序入口
│   ├── internal/              # 内部代码
│   │   └── api/
│   │       ├── handler/       # API处理器
│   │       ├── middleware/    # 中间件
│   │       ├── router/        # 路由
│   │       └── service/       # 业务逻辑
│   └── pkg/                   # 公共包
├── frontend/                   # 用户端
├── admin/                      # 管理端
├── sdk/                        # 多语言SDK
│   ├── go/                     # Go SDK
│   ├── python/                 # Python SDK
│   └── nodejs/                 # Node.js SDK
├── docs/                       # 文档
├── e2e/                        # 端到端测试
├── monitoring/                 # 监控配置
├── nginx/                      # Nginx配置
└── scripts/                    # 部署脚本
```

## API文档

详细API文档请查看 [API.md](docs/API接口文档.md)

## 性能指标

- QPS: >8000
- P99延迟: <80ms
- 缓存命中率: >95%
- 机器人识别准确率: >99%

## 安全特性

- JWT Token认证
- HMAC-SHA256签名验证
- 接口访问频率限制
- 防重放攻击机制
- CSRF/XSS/SQL注入防护
- IP白名单/黑名单
- DDoS防护
- OWASP Top 10安全测试通过

## 开发文档

- [开发核心.md](开发核心.md) - 开发进度和计划
- [API接口文档](docs/API接口文档.md) - 详细API文档
- [部署文档](docs/部署文档.md) - 部署指南
- [配置说明](docs/配置说明.md) - 配置详解
- [安全设计](docs/安全设计.md) - 安全架构
- [性能调优指南](docs/性能调优指南.md) - 性能优化
- [故障排查手册](docs/故障排查手册.md) - 问题解决

## 许可证

MIT License

---

**GitHub仓库**：https://github.com/opphk/hjtpx
