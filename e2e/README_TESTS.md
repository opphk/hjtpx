# E2E 测试文档 v15.0

## 概述

本文档描述了 HJTPX v15.0 系统的端到端（End-to-End）测试方案。E2E测试覆盖了从前端界面到后端API的完整用户流程，确保系统各组件之间的正确集成。

## 测试范围

### 覆盖的功能模块

1. **验证码功能测试**
   - 滑块验证码
   - 点击验证码
   - 手势验证码
   - 连连看验证码
   - 3D验证码
   - 语音验证码
   - 无感验证

2. **用户界面测试**
   - 首页功能
   - 验证码页面
   - 登录/注册流程
   - 响应式设计
   - 无障碍功能

3. **管理后台测试**
   - 管理员登录
   - 仪表盘
   - 应用管理
   - 日志查看
   - 黑名单管理
   - 统计分析

4. **API接口测试**
   - 验证码API
   - 认证API
   - 管理API

5. **无障碍测试**
   - ARIA标签
   - 键盘导航
   - 屏幕阅读器兼容性
   - 焦点管理

6. **跨浏览器测试**
   - Chrome
   - Firefox
   - Safari (WebKit)
   - 移动浏览器

## 测试环境要求

- Node.js 18+
- npm 9+
- Playwright 1.40+
- Docker（用于启动测试服务）

## 快速开始

### 1. 安装依赖

```bash
cd e2e
npm install
```

### 2. 安装浏览器

```bash
npm run install:browsers
```

### 3. 启动测试服务

在另一个终端启动服务：

```bash
cd ..
docker-compose up -d
```

### 4. 运行测试

```bash
# 运行所有测试
npm test

# 运行特定测试
npm run test:chromium

# 运行带UI的测试
npm run test:ui

# 生成测试报告
npm run test:report
```

## 测试命令

### 基础命令

```bash
# 运行所有测试
npm test

# 带UI运行测试
npm run test:ui

# 调试模式运行
npm run test:debug

# 显示测试报告
npm run report
```

### 浏览器特定测试

```bash
# Chrome
npm run test:chromium

# Firefox
npm run test:firefox

# WebKit
npm run test:webkit

# 有头模式（显示浏览器）
npm run test:headed
```

### 测试类型

```bash
# 冒烟测试
npm run test:smoke

# 回归测试
npm run test:regression

# 跨浏览器测试
npm run test:cross-browser

# 移动设备测试
npm run test:mobile
```

### 完整测试流程

```bash
# 1. 安装浏览器
npm run install:browsers

# 2. 运行所有测试
npm run test:all

# 3. 生成报告
npm run test:report

# 4. 查看报告
npm run report
```

## 测试项目配置

### Playwright 配置

编辑 `playwright.config.ts` 自定义测试配置：

```typescript
export default defineConfig({
  testDir: './tests',
  fullyParallel: false,
  retries: 2,
  workers: 2,
  reporter: [
    ['html', { open: 'never' }],
    ['json', { outputFile: 'test-results/results.json' }],
    ['list'],
  ],
  use: {
    baseURL: 'http://localhost:8080',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
});
```

### 环境变量

```bash
# 测试基础URL
BASE_URL=http://localhost:8080

# CI模式
CI=true

# 慢动作模式（用于调试）
SLOW_MO=100

# 调试模式
DEBUG=true
```

## 编写测试

### 测试文件结构

```
tests/
├── frontend/
│   ├── home.spec.ts
│   ├── captcha-pages.spec.ts
│   ├── accessibility.spec.ts
│   └── ...
├── admin/
│   ├── auth.spec.ts
│   ├── dashboard.spec.ts
│   └── ...
└── api/
    └── captcha.spec.ts
```

### 测试模板

```typescript
import { test, expect } from '@playwright/test';

test.describe('功能模块', () => {
  
  test.beforeEach(async ({ page }) => {
    // 每个测试前执行
    await page.goto('/');
  });

  test('@smoke 测试场景1', async ({ page }) => {
    // 测试步骤
    await page.click('#selector');
    
    // 断言
    await expect(page.locator('.result')).toBeVisible();
  });

  test('@regression 测试场景2', async ({ page }) => {
    // 测试代码
  });

  test('@accessibility 无障碍测试', async ({ page }) => {
    // 无障碍测试代码
  });
});
```

### 标签说明

- `@smoke`: 冒烟测试，快速验证核心功能
- `@regression`: 回归测试，全面验证功能
- `@accessibility`: 无障碍测试
- `@security`: 安全测试
- `@performance`: 性能测试

## 测试报告

### 报告位置

```
test-results/
├── html/                    # HTML报告
│   └── index.html
├── results.json           # JSON结果
└── screenshots/           # 失败截图
    └── ...
```

### 查看报告

```bash
# 在浏览器中打开HTML报告
open test-results/html/index.html

# 或使用Playwright查看
npm run report
```

### CI集成报告

在 CI 环境中，测试结果会自动上传到测试报告服务。

## 跨浏览器测试

### 支持的浏览器

- **桌面浏览器**
  - Chrome (最新)
  - Firefox (最新)
  - Safari (WebKit) (最新)

- **移动设备**
  - iPhone 12
  - iPad Pro
  - Pixel 5
  - Samsung Galaxy S10

### 运行跨浏览器测试

```bash
# 运行所有浏览器
npm run test:cross-browser

# 运行移动设备测试
npm run test:mobile

# 运行特定浏览器
npm run test:chromium
npm run test:firefox
npm run test:webkit
```

## 无障碍测试

### 测试内容

1. **ARIA标签**
   - 验证所有交互元素有适当的ARIA标签
   - 检查aria-describedby关联
   - 验证aria-live区域

2. **键盘导航**
   - Tab键导航顺序
   - Enter/Space激活按钮
   - Escape关闭模态框

3. **焦点管理**
   - 焦点是否可见
   - 焦点是否正确移动
   - 是否 trap 在模态框中

4. **屏幕阅读器**
   - 文本替代
   - 语义结构
   - 动态内容通知

### 运行无障碍测试

```bash
# 运行所有测试（包括无障碍）
npm test

# 只运行无障碍测试
npm test --grep="@accessibility"
```

## 持续集成

### GitHub Actions

在 `.github/workflows/e2e-tests.yml` 中配置：

```yaml
name: E2E Tests

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main, develop]

jobs:
  e2e-tests:
    runs-on: ubuntu-latest
    
    services:
      postgres:
        image: postgres:14
        env:
          POSTGRES_USER: hjtpx_test
          POSTGRES_PASSWORD: hjtpx_test_password
          POSTGRES_DB: hjtpx_test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
      
      redis:
        image: redis:7-alpine
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '18'
      
      - name: Install dependencies
        run: |
          cd e2e
          npm install
      
      - name: Install browsers
        run: |
          cd e2e
          npm run install:browsers
      
      - name: Start services
        run: |
          docker-compose up -d
          sleep 30
      
      - name: Run E2E tests
        run: |
          cd e2e
          npm test
      
      - name: Upload test results
        uses: actions/upload-artifact@v3
        if: always()
        with:
          name: e2e-test-results
          path: e2e/test-results/
```

## 故障排查

### 测试失败

1. **检查服务状态**
   ```bash
   docker-compose ps
   curl http://localhost:8080/health
   ```

2. **查看详细日志**
   ```bash
   docker-compose logs -f
   ```

3. **重新运行测试**
   ```bash
   # 清理后重新运行
   rm -rf test-results
   npm test
   ```

### 浏览器问题

1. **浏览器未安装**
   ```bash
   npm run install:browsers
   ```

2. **浏览器启动失败**
   ```bash
   # 检查系统依赖
   playwright install-deps
   ```

### 超时问题

1. **增加超时时间**
   编辑 `playwright.config.ts`:
   ```typescript
   timeout: 120000,
   ```

2. **慢动作模式**
   ```bash
   SLOW_MO=100 npm test
   ```

## 最佳实践

1. **测试独立性**
   - 每个测试应该独立运行
   - 不依赖其他测试的结果
   - 清理测试数据

2. **清晰的测试名称**
   - 使用描述性的测试名称
   - 包含@tag标签
   - 说明测试目的

3. **适当的等待**
   - 使用自动等待
   - 避免硬编码等待时间
   - 使用waitForSelector

4. **有意义的断言**
   - 使用具体的断言消息
   - 验证关键行为
   - 不仅验证元素存在

5. **错误处理**
   - 捕获异常
   - 提供详细错误信息
   - 截图和日志

## 相关文档

- [API文档](../docs/API接口文档.md)
- [开发者指南](../docs/开发者指南.md)
- [故障排查手册](../docs/故障排查手册.md)
- [部署文档](../docs/部署文档.md)

---

**最后更新**: 2026-05-19
**当前版本**: v15.0
