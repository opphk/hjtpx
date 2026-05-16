# E2E 测试框架

这是基于 Playwright 的行为验证系统端到端测试框架。

## 快速开始

### 安装依赖

```bash
cd e2e
npm install
```

### 安装浏览器

```bash
npx playwright install --with-deps
```

## 运行测试

### 运行所有测试

```bash
npm test
```

### 运行特定测试

```bash
# 仅运行前端测试
npx playwright test tests/frontend/

# 仅运行管理端测试
npx playwright test tests/admin/

# 仅运行验证码API测试
npx playwright test tests/api/
```

### 使用 UI 模式运行

```bash
npm run test:ui
```

### 调试模式运行

```bash
npm run test:debug
```

### 特定浏览器运行

```bash
# Chrome
npm run test:chromium

# Firefox
npm run test:firefox

# WebKit
npm run test:webkit
```

## 测试报告

### 查看测试报告

```bash
npm run report
```

## CI/CD 集成

本测试框架已配置 GitHub Actions 工作流。在以下情况会自动运行测试：

- 推送到 main/master 分支
- 创建 PR 到 main/master 分支
- 手动触发 (workflow_dispatch)

## 项目结构

```
e2e/
├── tests/              # 测试文件
│   ├── frontend/       # 用户端测试
│   ├── admin/          # 管理端测试
│   ├── api/            # API 测试
│   └── captcha/        # 验证码前端测试
├── fixtures/           # 测试夹具
├── utils/              # 工具函数
├── playwright.config.ts # Playwright 配置
└── package.json
```

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| BASE_URL | http://localhost:8080 | 测试的基础 URL |
