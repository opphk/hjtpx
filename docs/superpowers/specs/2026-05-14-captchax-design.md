# 行为验证系统 CaptchaX Spec

## Why
需要开发一个超越极验、易盾、五秒盾的现代化行为验证系统，提供更安全、更智能、更易用的验证码服务。

## What Changes
- 从 Node.js/Express 切换到 Go 语言后端
- 从 React 切换到 Go+HTML/JS 前端（使用 Go 原生模板）
- 前后端统一使用 Go 语言开发
- 完整的行为验证系统（滑块、点选、拼图等）
- 智能风控引擎
- 用户端 SDK 和管理后台

## Impact
- 新建 captcha 服务模块
- 创建 Go 后端 API
- 创建 Go 前端 UI
- 集成 Redis 缓存
- 集成 PostgreSQL 数据存储

## ADDED Requirements

### Requirement: 滑块验证码
系统 SHALL 提供滑块验证码，支持拖动滑块完成验证。

#### Scenario: 正常验证
- **WHEN** 用户发起验证请求
- **THEN** 返回带有缺口的背景图和滑块图片

### Requirement: 点选验证码
系统 SHALL 提供点选验证码，支持按顺序点击汉字/图标完成验证。

#### Scenario: 正常验证
- **WHEN** 用户发起点选验证请求
- **THEN** 返回包含多个目标物的图片和点击顺序提示

### Requirement: 智能风控
系统 SHALL 提供智能风控，根据用户行为判断是否为机器人。

#### Scenario: 风险检测
- **WHEN** 用户行为异常（如：过快、过慢、规律性）
- **THEN** 提升验证难度或直接拦截

### Requirement: 用户端 SDK
系统 SHALL 提供前端 SDK，方便客户集成验证码。

#### Scenario: SDK 集成
- **WHEN** 客户网站引入 SDK
- **THEN** 可以通过简单配置展示验证码组件

### Requirement: 管理后台
系统 SHALL 提供管理后台，支持配置验证类型、查看统计、管理黑白名单。

#### Scenario: 管理员登录
- **WHEN** 管理员访问管理后台
- **THEN** 可以配置系统参数、查看验证统计

## Architecture

```
CaptchaX/
├── cmd/
│   ├── server/           # 后端服务入口
│   └── admin/            # 管理后台入口
├── internal/
│   ├── captcha/          # 验证码核心逻辑
│   │   ├── slider/       # 滑块验证
│   │   ├── click/        # 点选验证
│   │   ├── puzzle/       # 拼图验证
│   │   └── util/         # 验证码工具
│   ├── api/              # API 接口
│   ├── middleware/       # 中间件
│   ├── model/            # 数据模型
│   ├── repository/       # 数据访问层
│   ├── service/          # 业务逻辑
│   ├── risk/             # 风控引擎
│   └── admin/            # 管理后台
├── pkg/
│   ├── cache/            # Redis 缓存
│   ├── database/         # PostgreSQL
│   └── response/         # 响应处理
├── web/                  # 前端资源
│   ├── static/           # 静态文件
│   └── templates/        # HTML 模板
├── config/               # 配置文件
└── main.go              # 程序入口
```

## Tech Stack
- **后端**: Go 1.21+, Gin 框架
- **前端**: Go html/template, Vanilla JS
- **数据库**: PostgreSQL 16
- **缓存**: Redis 7
- **图像处理**: Go 标准库 + imaging

## API Endpoints

### 验证接口
- `POST /api/v1/captcha/slider` - 获取滑块验证码
- `POST /api/v1/captcha/slider/verify` - 校验滑块
- `POST /api/v1/captcha/click` - 获取点选验证码
- `POST /api/v1/captcha/click/verify` - 校验点选
- `POST /api/v1/captcha/puzzle` - 获取拼图验证码
- `POST /api/v1/captcha/puzzle/verify` - 校验拼图

### 管理接口
- `POST /admin/login` - 管理员登录
- `GET /admin/dashboard` - 仪表盘
- `GET /admin/stats` - 统计数据
- `POST /admin/config` - 修改配置
- `GET /admin/whitelist` - 白名单管理
- `POST /admin/whitelist` - 添加白名单
- `DELETE /admin/whitelist/:id` - 删除白名单
- `GET /admin/blacklist` - 黑名单管理

## Database Schema

### captcha_logs 表
- id: 主键
- captcha_type: 验证码类型
- client_id: 客户端ID
- ip: IP地址
- user_agent: 用户代理
- result: 验证结果
- duration: 验证耗时
- risk_score: 风险评分
- created_at: 创建时间

### captcha_config 表
- id: 主键
- key: 配置键
- value: 配置值
- updated_at: 更新时间
