# HJTPX 新阶段开发实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 基于HJTPX现有开发核心文档，执行新阶段的20个开发任务，完善测试框架、优化性能、增强安全性和完善国际化功能。

**Architecture:** 采用前后端分离架构，前端使用React + Vite，后端使用Express.js，数据库使用PostgreSQL，缓存使用Redis。遵循TDD开发模式，每个功能先写测试再实现。

**Tech Stack:** Node.js, Express.js, React, PostgreSQL, Redis, Jest, Playwright, Docker

---

## 项目信息

- **项目路径**: `/workspace/hjtpx`
- **主仓库**: `opphk/hjtpx`
- **目标仓库**: `hjtpx1` (需创建PR)
- **开发模式**: TDD (测试驱动开发)

---

## 大任务 1: API集成测试完善

### 任务说明
完善 API 集成测试，覆盖所有端点，添加测试数据工厂和fixture

### Files
- Create: `src/backend/tests/integration/auth.integration.test.js`
- Create: `src/backend/tests/integration/users.integration.test.js`
- Create: `src/backend/tests/integration/notifications.integration.test.js`
- Create: `src/backend/tests/integration/files.integration.test.js`
- Create: `src/backend/tests/factories/userFactory.js`
- Create: `src/backend/tests/factories/notificationFactory.js`
- Modify: `src/backend/tests/setup.js`
- Test: `src/backend/tests/integration/*.test.js`

### SubTasks
- [ ] 1.1: 创建认证API集成测试文件
- [ ] 1.2: 创建用户管理API集成测试文件
- [ ] 1.3: 创建通知API集成测试文件
- [ ] 1.4: 创建文件管理API集成测试文件
- [ ] 1.5: 创建测试数据工厂
- [ ] 1.6: 运行所有集成测试验证

---

## 大任务 2: 前端E2E测试框架

### 任务说明
配置 Playwright 测试框架，创建登录、注册、用户管理流程 E2E 测试

### Files
- Create: `src/frontend/tests/e2e/login.spec.js`
- Create: `src/frontend/tests/e2e/register.spec.js`
- Create: `src/frontend/tests/e2e/user-management.spec.js`
- Modify: `playwright.config.js`
- Modify: `.github/workflows/ci.yml`
- Test: `src/frontend/tests/e2e/*.spec.js`

### SubTasks
- [ ] 2.1: 配置Playwright测试框架
- [ ] 2.2: 创建登录流程E2E测试
- [ ] 2.3: 创建注册流程E2E测试
- [ ] 2.4: 创建用户管理E2E测试
- [ ] 2.5: 配置CI中的E2E测试运行
- [ ] 2.6: 运行所有E2E测试验证

---

## 大任务 3: 数据库迁移脚本优化

### 任务说明
审查现有迁移脚本完整性，添加回滚脚本，优化性能，添加状态追踪

### Files
- Create: `migrations/009-migration-tracking.up.sql`
- Create: `migrations/009-migration-tracking.down.sql`
- Modify: `scripts/migrate.js`
- Modify: `scripts/run-migrations.js`
- Test: `scripts/test-migrations.sh`

### SubTasks
- [ ] 3.1: 审查现有迁移脚本完整性
- [ ] 3.2: 添加迁移追踪表
- [ ] 3.3: 优化迁移执行脚本
- [ ] 3.4: 测试迁移回滚功能
- [ ] 3.5: 编写迁移文档

---

## 大任务 4: Redis缓存策略优化

### 任务说明
分析现有缓存使用情况，优化会话缓存，实现 API 响应缓存，配置失效策略，添加监控指标

### Files
- Create: `src/backend/services/sessionCacheService.js`
- Modify: `src/backend/services/cacheService.js`
- Modify: `src/backend/middleware/cacheMiddleware.js`
- Create: `src/backend/config/cacheConfig.js`
- Test: `src/backend/tests/services/cacheService.test.js`

### SubTasks
- [ ] 4.1: 分析现有缓存使用情况
- [ ] 4.2: 实现会话缓存优化
- [ ] 4.3: 实现API响应缓存
- [ ] 4.4: 配置缓存失效策略
- [ ] 4.5: 添加缓存监控指标
- [ ] 4.6: 测试缓存功能

---

## 大任务 5: WebSocket压力测试

### 任务说明
配置 WebSocket 测试环境，编写并发连接测试，测试消息广播性能，优化心跳机制

### Files
- Create: `src/backend/websocket/websocketService.js`
- Create: `tests/websocket/load-test.js`
- Modify: `src/backend/websocket/index.js`
- Test: `tests/websocket/*.test.js`

### SubTasks
- [ ] 5.1: 配置WebSocket测试环境
- [ ] 5.2: 编写并发连接测试脚本
- [ ] 5.3: 测试消息广播性能
- [ ] 5.4: 优化WebSocket心跳机制
- [ ] 5.5: 添加WebSocket监控
- [ ] 5.6: 运行压力测试验证

---

## 大任务 6: 前端性能优化

### 任务说明
分析前端性能瓶颈，实现路由级代码分割，图片懒加载，优化 bundle 大小

### Files
- Modify: `src/frontend/src/App.jsx`
- Create: `src/frontend/src/components/LazyImage.jsx`
- Modify: `vite.config.js`
- Create: `docs/frontend/performance-analysis.md`
- Test: `Lighthouse Performance Score > 90`

### SubTasks
- [ ] 6.1: 分析前端性能瓶颈
- [ ] 6.2: 实现路由级代码分割
- [ ] 6.3: 实现图片懒加载组件
- [ ] 6.4: 优化bundle大小
- [ ] 6.5: 测试Lighthouse性能分数

---

## 大任务 7: 安全漏洞扫描与修复

### 任务说明
集成 npm audit，扫描并修复依赖漏洞，添加 CSP 策略，配置安全响应头

### Files
- Modify: `package.json`
- Modify: `src/backend/middleware/securityHeaders.js`
- Create: `src/backend/middleware/cspMiddleware.js`
- Create: `tests/security/dependency-audit.test.js`
- Test: `npm audit`

### SubTasks
- [ ] 7.1: 运行npm audit安全扫描
- [ ] 7.2: 修复高危依赖漏洞
- [ ] 7.3: 添加CSP内容安全策略
- [ ] 7.4: 配置安全响应头
- [ ] 7.5: 编写安全测试用例

---

## 大任务 8: API版本控制完善

### 任务说明
设计 API 版本迁移策略，实现版本协商，添加弃用警告

### Files
- Create: `src/backend/middleware/apiVersionNegotiation.js`
- Create: `src/backend/middleware/deprecationWarning.js`
- Modify: `src/backend/routes/v1/index.js`
- Modify: `src/backend/routes/v2/index.js`
- Create: `docs/api/versioning-strategy.md`
- Test: `tests/versioning/*.test.js`

### SubTasks
- [ ] 8.1: 设计API版本迁移策略
- [ ] 8.2: 实现API版本协商中间件
- [ ] 8.3: 添加版本弃用警告
- [ ] 8.4: 编写版本迁移指南
- [ ] 8.5: 测试版本共存功能

---

## 大任务 9: 前端国际化完善

### 任务说明
审计现有 i18n 覆盖范围，添加更多语言支持，实现动态语言切换

### Files
- Create: `src/frontend/src/i18n/locales/pt.json`
- Create: `src/frontend/src/i18n/locales/it.json`
- Create: `src/frontend/src/i18n/locales/nl.json`
- Modify: `src/frontend/src/i18n/index.js`
- Modify: `src/frontend/src/components/LanguageSwitcher.jsx`
- Test: `tests/i18n/*.test.js`

### SubTasks
- [ ] 9.1: 审计现有i18n覆盖范围
- [ ] 9.2: 添加葡萄牙语、意大利语、荷兰语支持
- [ ] 9.3: 实现动态语言切换
- [ ] 9.4: 添加日期时间本地化
- [ ] 9.5: 优化翻译文件加载

---

## 大任务 10: 后端日志聚合

### 任务说明
配置结构化日志格式，实现日志分级管理，添加请求追踪 ID，添加敏感信息过滤

### Files
- Modify: `src/backend/config/logging.js`
- Modify: `src/backend/utils/logger.js`
- Create: `src/backend/middleware/requestId.js`
- Create: `src/backend/middleware/sanitizeLogs.js`
- Test: `tests/logger/*.test.js`

### SubTasks
- [ ] 10.1: 配置结构化日志格式
- [ ] 10.2: 实现日志分级管理
- [ ] 10.3: 添加请求追踪ID中间件
- [ ] 10.4: 配置日志输出格式化
- [ ] 10.5: 添加敏感信息过滤

---

## 大任务 11: 移动端PWA优化

### 任务说明
完善 Service Worker，添加离线缓存策略，实现推送通知，优化 Manifest 配置

### Files
- Modify: `src/frontend/public/sw.js`
- Modify: `src/frontend/public/manifest.json`
- Create: `src/frontend/src/services/pushNotification.js`
- Create: `src/frontend/src/components/PWAInstallPrompt.jsx`
- Test: `PWA Lighthouse Score`

### SubTasks
- [ ] 11.1: 完善Service Worker
- [ ] 11.2: 添加离线缓存策略
- [ ] 11.3: 实现推送通知服务
- [ ] 11.4: 优化Manifest配置
- [ ] 11.5: 添加PWA安装提示组件

---

## 大任务 12: 数据库连接池优化

### 任务说明
分析当前连接池配置，优化连接池参数，添加连接健康检查，实现连接泄漏检测

### Files
- Modify: `src/backend/config/database/dbPoolManager.js`
- Create: `src/backend/services/connectionHealthCheck.js`
- Create: `src/backend/services/connectionLeakDetector.js`
- Test: `tests/database/connection-pool.test.js`

### SubTasks
- [ ] 12.1: 分析当前连接池配置
- [ ] 12.2: 优化连接池参数
- [ ] 12.3: 添加连接健康检查服务
- [ ] 12.4: 实现连接泄漏检测
- [ ] 12.5: 配置连接池监控

---

## 大任务 13: 前端组件库文档

### 任务说明
配置 Storybook 文档工具，编写基础组件文档，添加组件示例代码

### Files
- Modify: `.storybook/main.js`
- Modify: `.storybook/preview.js`
- Create: `src/frontend/src/stories/Button.stories.jsx`
- Create: `src/frontend/src/stories/Input.stories.jsx`
- Create: `src/frontend/src/stories/Modal.stories.jsx`
- Test: `Storybook build`

### SubTasks
- [ ] 13.1: 配置Storybook文档工具
- [ ] 13.2: 编写Button组件文档
- [ ] 13.3: 编写Input组件文档
- [ ] 13.4: 编写Modal组件文档
- [ ] 13.5: 构建Storybook文档网站

---

## 大任务 14: 后端API文档自动更新

### 任务说明
配置 Swagger 自动生成，添加 API 变更检测，实现文档版本管理

### Files
- Modify: `src/backend/config/swagger.js`
- Create: `scripts/check-api-changes.js`
- Create: `scripts/generate-swagger.js`
- Create: `docs/api/versions/`
- Test: `scripts/validate-swagger.js`

### SubTasks
- [ ] 14.1: 配置Swagger自动生成
- [ ] 14.2: 添加API变更检测脚本
- [ ] 14.3: 实现文档版本管理
- [ ] 14.4: 添加API使用统计
- [ ] 14.5: 配置文档CI检查

---

## 大任务 15: CI/CD测试覆盖率检查

### 任务说明
配置覆盖率阈值检查，添加覆盖率下降告警，生成覆盖率趋势报告

### Files
- Modify: `jest.config.js`
- Create: `scripts/coverage-analyzer.js`
- Create: `scripts/coverage-alert.js`
- Modify: `.github/workflows/ci.yml`
- Test: `npm run test:coverage`

### SubTasks
- [ ] 15.1: 配置覆盖率阈值检查
- [ ] 15.2: 添加覆盖率下降告警
- [ ] 15.3: 生成覆盖率趋势报告
- [ ] 15.4: 配置分支覆盖率要求
- [ ] 15.5: 集成到PR检查流程

---

## 大任务 16: 前端无障碍访问优化

### 任务说明
添加 ARIA 标签，优化键盘导航，添加屏幕阅读器支持，配置 a11y 测试

### Files
- Modify: `src/frontend/src/components/*.jsx`
- Create: `src/frontend/__tests__/a11y.test.js`
- Modify: `jest.config.js`
- Modify: `src/frontend/src/styles/global.css`
- Test: `axe-core accessibility tests`

### SubTasks
- [ ] 16.1: 添加ARIA标签到所有组件
- [ ] 16.2: 优化键盘导航
- [ ] 16.3: 添加屏幕阅读器支持
- [ ] 16.4: 配置a11y测试
- [ ] 16.5: 添加颜色对比度优化

---

## 大任务 17: 后端错误追踪系统

### 任务说明
集成 Sentry 错误追踪，配置错误分组，添加性能监控，配置告警规则

### Files
- Modify: `src/backend/config/sentry.js`
- Create: `.sentryclirc`
- Modify: `src/index.js`
- Test: `Sentry integration tests`

### SubTasks
- [ ] 17.1: 集成Sentry错误追踪
- [ ] 17.2: 配置错误分组
- [ ] 17.3: 添加性能监控
- [ ] 17.4: 配置告警规则
- [ ] 17.5: 集成源码映射

---

## 大任务 18: 数据库备份恢复自动化

### 任务说明
配置自动备份脚本，实现增量备份，添加备份验证，实现定时恢复演练

### Files
- Modify: `scripts/backup.sh`
- Create: `scripts/backup-incremental.sh`
- Create: `scripts/verify-backup.sh`
- Create: `scripts/restore-drill.sh`
- Create: `docs/database/backup-restore.md`
- Test: `scripts/test-backup.sh`

### SubTasks
- [ ] 18.1: 配置自动备份脚本
- [ ] 18.2: 实现增量备份
- [ ] 18.3: 添加备份验证
- [ ] 18.4: 实现定时恢复演练
- [ ] 18.5: 编写备份恢复文档

---

## 大任务 19: 前端SEO优化

### 任务说明
添加 Meta 标签，配置 Open Graph，添加结构化数据，优化页面标题

### Files
- Modify: `src/frontend/index.html`
- Create: `src/frontend/src/components/StructuredData.jsx`
- Modify: `src/frontend/src/pages/*.jsx`
- Create: `src/frontend/public/robots.txt`
- Test: `SEO Lighthouse Score`

### SubTasks
- [ ] 19.1: 添加Meta标签到所有页面
- [ ] 19.2: 配置Open Graph
- [ ] 19.3: 添加结构化数据组件
- [ ] 19.4: 优化页面标题
- [ ] 19.5: 配置robots.txt

---

## 大任务 20: 后端GraphQL API

### 任务说明
配置 Apollo Server，定义 GraphQL Schema，实现查询和变更，添加 GraphQL Playground

### Files
- Create: `src/backend/graphql/schema.js`
- Create: `src/backend/graphql/resolvers.js`
- Create: `src/backend/config/apollo.js`
- Modify: `src/index.js`
- Test: `tests/graphql/*.test.js`

### SubTasks
- [ ] 20.1: 配置Apollo Server
- [ ] 20.2: 定义GraphQL Schema
- [ ] 20.3: 实现查询和变更
- [ ] 20.4: 添加GraphQL Playground
- [ ] 20.5: 测试GraphQL API

---

## 执行顺序建议

### Phase 1: 基础测试框架 (可并行)
- 大任务 1: API集成测试完善
- 大任务 2: 前端E2E测试框架
- 大任务 3: 数据库迁移脚本优化

### Phase 2: 性能与缓存 (可并行)
- 大任务 4: Redis缓存策略优化
- 大任务 5: WebSocket压力测试
- 大任务 6: 前端性能优化

### Phase 3: 安全与监控 (可并行)
- 大任务 7: 安全漏洞扫描与修复
- 大任务 10: 后端日志聚合
- 大任务 17: 后端错误追踪系统

### Phase 4: 国际化与SEO (可并行)
- 大任务 8: API版本控制完善
- 大任务 9: 前端国际化完善
- 大任务 19: 前端SEO优化

### Phase 5: PWA与文档 (可并行)
- 大任务 11: 移动端PWA优化
- 大任务 13: 前端组件库文档
- 大任务 14: 后端API文档自动更新

### Phase 6: 高级功能 (可并行)
- 大任务 12: 数据库连接池优化
- 大任务 15: CI/CD测试覆盖率检查
- 大任务 16: 前端无障碍访问优化
- 大任务 18: 数据库备份恢复自动化
- 大任务 20: 后端GraphQL API

---

## 提交规范

使用 Conventional Commits 规范：

```
feat: 新功能
fix: 错误修复
docs: 文档更新
style: 代码格式
refactor: 重构
perf: 性能优化
test: 测试相关
chore: 构建/工具
```

示例：
```
feat(tests): add API integration tests

feat(e2e): add Playwright E2E tests for login flow

fix(redis): optimize cache invalidation strategy
```

---

## 验证检查清单

- [ ] 所有API端点有集成测试
- [ ] Playwright E2E测试通过
- [ ] 数据库迁移脚本完整
- [ ] Redis缓存正常工作
- [ ] WebSocket压力测试通过
- [ ] Lighthouse性能分数 > 90
- [ ] 安全扫描无高危漏洞
- [ ] API版本控制正常
- [ ] 国际化覆盖 > 90%
- [ ] 日志系统正常工作
- [ ] PWA功能完整
- [ ] 无障碍测试通过
- [ ] GraphQL API可用
