# 行为验证系统 v2.0 开发规格说明

## 目标
开发一个完整的、高性能的行为验证系统（行为验证码），使用Go语言编写前后端，目标是超越极验、易盾、五秒盾等现有产品。

## 核心功能变更

### 新增功能
- **增强滑块验证码**: 真实图片背景、多样化拼图块、轨迹采集、速度检测
- **增强点选验证码**: 多种验证模式（汉字/字母/数字）、图片背景、点击时序验证
- **增强图形验证码**: 优化字体渲染、扭曲变形效果、复杂干扰线
- **增强行为分析**: 鼠标轨迹平滑、速度分析、路径相似度检测、键盘模式分析
- **用户管理**: 注册/登录/JWT刷新/密码重置/邮箱验证
- **应用管理**: 应用CRUD/密钥管理/配置管理
- **日志分析**: 验证日志存储/查询/统计/趋势分析
- **API限流**: IP限流/用户限流/应用限流/黑白名单
- **安全加固**: CSRF防护/XSS过滤/请求签名/数据加密
- **前端增强**: 加载动画/成功反馈/移动端适配/无障碍支持
- **管理端增强**: 实时数据/可视化图表/风控规则配置
- **性能优化**: Redis缓存/数据库连接池/异步任务队列/CDN集成
- **部署运维**: Docker部署/健康检查/日志收集/配置热更新
- **测试与SDK**: 单元测试/集成测试/Go SDK/API文档

### 改进项
- 验证码算法优化
- 风险评分算法优化
- UI/UX优化
- 性能优化

## 技术要求
- **CSS规范**: 所有CSS必须从bootcdn.cn加载，禁止编写自定义CSS文件
- **CDN资源**:
  - Bootstrap 5 CSS: `https://cdn.bootcdn.net/ajax/libs/twitter-bootstrap/5.3.8/css/bootstrap.min.css`
  - Bootstrap 5 JS: `https://cdn.bootcdn.net/ajax/libs/twitter-bootstrap/5.3.8/js/bootstrap.bundle.min.js`
  - Font Awesome 6: `https://cdn.bootcdn.net/ajax/libs/font-awesome/6.5.1/css/all.min.css`
  - Chart.js: `https://cdn.bootcdn.net/ajax/libs/Chart.js/4.4.0/chart.umd.min.js`
- **数据库**: PostgreSQL + Redis
- **语言**: Go (前后端)

## 架构
```
hjtpx/
├── backend/                    # 后端服务 (Go + Gin + GORM)
│   ├── cmd/                   # 程序入口
│   ├── internal/              # 内部代码
│   │   └── api/
│   │       ├── handler/       # API处理器
│   │       ├── middleware/     # 中间件
│   │       ├── router/        # 路由
│   │       └── service/       # 业务逻辑
│   └── pkg/                   # 公共包
│       ├── config/            # 配置
│       ├── database/          # 数据库
│       ├── models/            # 数据模型
│       ├── postgres/          # PostgreSQL
│       ├── redis/             # Redis
│       ├── response/          # 响应
│       └── jwt/              # JWT认证
├── frontend/                   # 前端（用户端）
│   ├── static/
│   └── templates/
├── admin/                      # 管理端
│   ├── static/
│   └── templates/
├── .trae/                      # 规划文档
│   └── specs/
└── 开发核心.md
```

## 验收标准
1. 所有验证码功能正常运行
2. 行为分析准确识别机器人行为
3. API限流和安全防护有效
4. 前端界面美观、响应式
5. 管理端功能完整
6. 单元测试覆盖率 > 70%
7. 部署文档完整
