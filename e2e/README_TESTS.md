# E2E测试使用指南

## 快速开始

### 1. 安装依赖

```bash
cd /workspace/hjtpx/e2e
npm install
npx playwright install --with-deps
```

### 2. 配置环境

确保后端服务正在运行在 `http://localhost:8080`

### 3. 运行测试

```bash
# 运行所有测试
npm test

# 运行特定测试文件
npx playwright test tests/frontend/captcha-pages.spec.ts

# 运行特定测试
npx playwright test tests/frontend/captcha-pages.spec.ts --grep "滑块"

# 调试模式
npm run test:debug

# UI模式
npm run test:ui
```

### 4. 生成测试报告

```bash
# 运行测试并生成完整报告
npm run test:all

# 仅生成报告（需要先运行测试）
npm run test:report

# 查看HTML报告
npm run report
```

## 测试结构

```
e2e/
├── tests/
│   ├── admin/          # 管理端测试
│   │   ├── admin-pages.spec.ts
│   │   ├── auth.spec.ts
│   │   └── dashboard.spec.ts
│   ├── api/            # API测试
│   │   └── captcha.spec.ts
│   └── frontend/       # 前端测试
│       ├── captcha-pages.spec.ts
│       ├── click-captcha.spec.ts
│       ├── home.spec.ts
│       └── home-page.spec.ts
├── utils/              # 工具类
│   ├── api-helper.ts
│   ├── test-data.ts
│   ├── test-helpers.ts
│   └── report-generator.ts
└── fixtures/           # 测试fixtures
    └── test-fixtures.ts
```

## 测试覆盖范围

### 滑块验证码测试
- 页面加载测试
- 元素可见性测试
- API生成测试
- 拖拽交互测试
- 验证流程测试
- 错误处理测试
- 并发测试
- UI交互测试
- 控制台错误检测

### 点选验证码测试
- 页面加载测试
- 单点验证测试
- 多点验证测试
- UI元素测试
- 点击交互测试
- 错误处理测试
- 并发测试
- 控制台错误检测

### 管理端测试
- 登录/登出功能
- 页面加载测试
- 控制台错误检测
- 认证测试
- 导航测试
- API数据获取测试
- 页面重定向测试

## 报告说明

测试完成后，会在以下位置生成报告：

- `test-results/html/index.html` - Playwright HTML报告
- `test-results/e2e-test-report.html` - 自定义HTML报告
- `test-results/e2e-test-report.md` - Markdown报告
- `test-results/e2e-test-report.json` - JSON报告
- `test-screenshots/` - 测试截图目录

## 常见问题

### Q: 测试失败时怎么办？

A: 
1. 检查后端服务是否正常运行
2. 查看Playwright HTML报告获取详细错误信息
3. 查看测试截图了解失败时的页面状态
4. 使用调试模式重新运行失败的测试

### Q: 如何添加新的测试用例？

A: 
1. 在相应的测试文件中添加新的 `test` 块
2. 使用 `TestHelpers` 工具类进行截图和日志记录
3. 使用 `ApiHelper` 进行API调用
4. 遵循现有的测试命名和结构规范

### Q: 如何跳过某些测试？

A: 
```typescript
test.skip('跳过的测试', async ({ page }) => {
  // ...
});
```

### Q: 如何只运行失败的测试？

A: 
```bash
npx playwright test --grep "@failed"
```

## 最佳实践

1. **使用有意义的测试名称**：清晰描述测试内容
2. **添加适当的等待**：使用 `waitForLoadState('networkidle')`
3. **捕获关键状态**：使用 `takeScreenshot` 记录重要状态
4. **清理测试数据**：在 `afterEach` 中清理创建的数据
5. **分离关注点**：UI测试和API测试分开

## 报告查看

```bash
# 直接打开HTML报告
open test-results/html/index.html

# 查看控制台报告
cat test-results/results.json | jq

# 查看Markdown报告
cat test-results/e2e-test-report.md
```
