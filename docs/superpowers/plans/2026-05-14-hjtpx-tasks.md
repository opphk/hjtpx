# HJTPX 下一阶段开发 - 任务分解清单

## [ ] 大任务 1: API集成测试完善
- **Priority**: P0
- **Depends On**: None
- **Description**: 完善 API 集成测试，覆盖所有端点，添加测试数据工厂和 fixture
- **Acceptance Criteria Addressed**: AC-1
- **Test Requirements**:
  - `programmatic` 1.1: 认证 API 端点集成测试通过
  - `programmatic` 1.2: 用户管理 API 端点集成测试通过
  - `programmatic` 1.3: 通知 API 集成测试通过
  - `programmatic` 1.4: 文件管理 API 集成测试通过
  - `programmatic` 1.5: 测试数据工厂正常工作
- **Notes**: 测试文件位于 `tests/integration/`

### 子任务分解
- [ ] 子任务 1.1: 完成所有认证 API 端点的集成测试
  - 文件: `tests/integration/auth.test.js`
  - 测试: 登录、注册、验证、刷新、登出
- [ ] 子任务 1.2: 完成所有用户管理 API 端点的集成测试
  - 文件: `tests/integration/users.test.js`
  - 测试: 创建、读取、更新、删除用户
- [ ] 子任务 1.3: 完成通知和消息 API 的集成测试
  - 文件: `tests/integration/notifications.test.js`
- [ ] 子任务 1.4: 完成文件管理 API 的集成测试
  - 文件: `tests/integration/files.test.js`
- [ ] 子任务 1.5: 添加测试数据工厂和 fixture
  - 文件: `tests/helpers/factories.js`

---

## [ ] 大任务 2: 前端E2E测试框架
- **Priority**: P0
- **Depends On**: None
- **Description**: 配置 Playwright 测试框架，创建登录、注册、用户管理流程 E2E 测试
- **Acceptance Criteria Addressed**: AC-2
- **Test Requirements**:
  - `programmatic` 2.1: Playwright 测试框架配置完成
  - `programmatic` 2.2: 登录流程 E2E 测试通过
  - `programmatic` 2.3: 注册流程 E2E 测试通过
  - `programmatic` 2.4: 用户管理 E2E 测试通过
  - `programmatic` 2.5: CI 中的 E2E 测试配置完成
- **Notes**: 测试文件位于 `src/frontend/tests/e2e/`

### 子任务分解
- [ ] 子任务 2.1: 配置 Playwright 测试框架
  - 文件: `playwright.config.js`
- [ ] 子任务 2.2: 创建登录流程 E2E 测试
  - 文件: `src/frontend/tests/e2e/login.spec.js`
- [ ] 子任务 2.3: 创建注册流程 E2E 测试
  - 文件: `src/frontend/tests/e2e/register.spec.js`
- [ ] 子任务 2.4: 创建用户管理 E2E 测试
  - 文件: `src/frontend/tests/e2e/user-management.spec.js`
- [ ] 子任务 2.5: 配置 CI 中的 E2E 测试运行
  - 文件: `.github/workflows/ci.yml`

---

## [ ] 大任务 3: 数据库迁移脚本优化
- **Priority**: P0
- **Depends On**: None
- **Description**: 审查现有迁移脚本完整性，添加回滚脚本，优化性能，添加状态追踪
- **Acceptance Criteria Addressed**: AC-3
- **Test Requirements**:
  - `programmatic` 3.1: 迁移脚本完整性审查通过
  - `programmatic` 3.2: 回滚脚本正常工作
  - `programmatic` 3.3: 迁移执行性能优化
  - `programmatic` 3.4: 迁移状态追踪正常
  - `programmatic` 3.5: 迁移文档完整
- **Notes**: 迁移文件位于 `migrations/`

### 子任务分解
- [ ] 子任务 3.1: 审查现有迁移脚本完整性
  - 文件: `migrations/`
- [ ] 子任务 3.2: 添加迁移回滚脚本
  - 文件: `scripts/migrate-rollback.js`
- [ ] 子任务 3.3: 优化迁移执行性能
  - 文件: `scripts/migrate.js`
- [ ] 子任务 3.4: 添加迁移状态追踪
  - 文件: `migrations/000-migration-state.sql`
- [ ] 子任务 3.5: 编写迁移文档
  - 文件: `docs/database/migrations.md`

---

## [ ] 大任务 4: Redis缓存策略优化
- **Priority**: P0
- **Depends On**: None
- **Description**: 分析现有缓存使用情况，优化会话缓存，实现 API 响应缓存，配置失效策略，添加监控指标
- **Acceptance Criteria Addressed**: AC-4
- **Test Requirements**:
  - `programmatic` 4.1: 缓存使用分析报告完成
  - `programmatic` 4.2: 会话缓存优化生效
  - `programmatic` 4.3: API 响应缓存正常工作
  - `programmatic` 4.4: 缓存失效策略配置完成
  - `programmatic` 4.5: 缓存监控指标正常
- **Notes**: 缓存相关代码位于 `src/backend/middleware/cacheMiddleware.js` 和 `src/backend/services/cacheService.js`

### 子任务分解
- [ ] 子任务 4.1: 分析现有缓存使用情况
  - 文件: `docs/cache/analysis.md`
- [ ] 子任务 4.2: 实现会话缓存优化
  - 文件: `src/backend/services/sessionService.js`
- [ ] 子任务 4.3: 实现 API 响应缓存
  - 文件: `src/backend/middleware/cacheMiddleware.js`
- [ ] 子任务 4.4: 配置缓存失效策略
  - 文件: `src/backend/config/cache.js`
- [ ] 子任务 4.5: 添加缓存监控指标
  - 文件: `src/backend/services/metricsService.js`

---

## [ ] 大任务 5: WebSocket压力测试
- **Priority**: P0
- **Depends On**: None
- **Description**: 配置 WebSocket 测试环境，编写并发连接测试，测试消息广播性能，优化心跳机制，添加监控
- **Acceptance Criteria Addressed**: AC-5
- **Test Requirements**:
  - `programmatic` 5.1: WebSocket 测试环境配置完成
  - `programmatic` 5.2: 并发连接测试通过（1000+）
  - `programmatic` 5.3: 消息广播性能测试通过（<100ms）
  - `programmatic` 5.4: 心跳机制优化生效
  - `programmatic` 5.5: WebSocket 监控正常
- **Notes**: WebSocket 相关代码位于 `src/backend/websocket/`

### 子任务分解
- [ ] 子任务 5.1: 配置 WebSocket 测试环境
  - 文件: `tests/performance/websocket-test.js`
- [ ] 子任务 5.2: 编写并发连接测试脚本
  - 文件: `tests/performance/websocket-concurrent.js`
- [ ] 子任务 5.3: 测试消息广播性能
  - 文件: `tests/performance/websocket-broadcast.js`
- [ ] 子任务 5.4: 优化 WebSocket 心跳机制
  - 文件: `src/backend/websocket/index.js`
- [ ] 子任务 5.5: 添加 WebSocket 监控
  - 文件: `src/backend/services/metricsService.js`

---

## [ ] 大任务 6: 前端性能优化
- **Priority**: P1
- **Depends On**: 大任务 2
- **Description**: 分析前端性能瓶颈，实现路由级代码分割，图片懒加载，Gzip 压缩，优化 bundle 大小
- **Acceptance Criteria Addressed**: AC-6
- **Test Requirements**:
  - `human-judgment` 6.1: 性能瓶颈分析报告完成
  - `programmatic` 6.2: 路由级代码分割正常工作
  - `programmatic` 6.3: 图片懒加载正常工作
  - `programmatic` 6.4: Gzip 压缩配置完成
  - `human-judgment` 6.5: Lighthouse 分数 > 90
- **Notes**: 前端代码位于 `src/frontend/`

### 子任务分解
- [ ] 子任务 6.1: 分析前端性能瓶颈
  - 文件: `docs/performance/analysis.md`
- [ ] 子任务 6.2: 实现路由级代码分割
  - 文件: `src/frontend/src/App.jsx`
- [ ] 子任务 6.3: 实现图片懒加载
  - 文件: `src/frontend/src/components/LazyImage.jsx`
- [ ] 子任务 6.4: 配置 Gzip 压缩
  - 文件: `nginx.conf`
- [ ] 子任务 6.5: 优化 bundle 大小
  - 文件: `src/frontend/vite.config.js`

---

## [ ] 大任务 7: 安全漏洞扫描与修复
- **Priority**: P1
- **Depends On**: None
- **Description**: 集成 npm audit，扫描并修复依赖漏洞，添加 CSP 策略，配置安全响应头，编写安全测试
- **Acceptance Criteria Addressed**: AC-7
- **Test Requirements**:
  - `programmatic` 7.1: npm audit 扫描完成
  - `programmatic` 7.2: 所有高危漏洞已修复
  - `programmatic` 7.3: CSP 策略配置完成
  - `programmatic` 7.4: 安全响应头配置完成
  - `programmatic` 7.5: 安全测试用例通过
- **Notes**: 安全相关中间件位于 `src/backend/middleware/securityHardening.js`

### 子任务分解
- [ ] 子任务 7.1: 集成 npm audit 安全扫描
  - 文件: `package.json`
- [ ] 子任务 7.2: 扫描并修复依赖漏洞
  - 文件: `package.json`, `package-lock.json`
- [ ] 子任务 7.3: 添加 CSP 内容安全策略
  - 文件: `src/backend/middleware/securityHeaders.js`
- [ ] 子任务 7.4: 配置安全响应头
  - 文件: `src/backend/middleware/helmet.js`
- [ ] 子任务 7.5: 编写安全测试用例
  - 文件: `tests/security/security.test.js`

---

## [ ] 大任务 8: API版本控制完善
- **Priority**: P1
- **Depends On**: None
- **Description**: 设计 API 版本迁移策略，实现版本协商，添加弃用警告，编写迁移指南，配置版本共存测试
- **Acceptance Criteria Addressed**: AC-8
- **Test Requirements**:
  - `programmatic` 8.1: 版本迁移策略文档完成
  - `programmatic` 8.2: API 版本协商正常工作
  - `programmatic` 8.3: 版本弃用警告正常
  - `programmatic` 8.4: 版本迁移指南完整
  - `programmatic` 8.5: 版本共存测试通过
- **Notes**: API 路由位于 `src/backend/routes/`

### 子任务分解
- [ ] 子任务 8.1: 设计 API 版本迁移策略
  - 文件: `docs/api/versioning-strategy.md`
- [ ] 子任务 8.2: 实现 API 版本协商
  - 文件: `src/backend/middleware/apiVersion.js`
- [ ] 子任务 8.3: 添加版本弃用警告
  - 文件: `src/backend/middleware/deprecationWarning.js`
- [ ] 子任务 8.4: 编写版本迁移指南
  - 文件: `docs/api/migration-guide.md`
- [ ] 子任务 8.5: 配置版本共存测试
  - 文件: `tests/integration/api-version.test.js`

---

## [ ] 大任务 9: 前端国际化完善
- **Priority**: P1
- **Depends On**: None
- **Description**: 审计现有 i18n 覆盖范围，添加更多语言支持，实现动态语言切换，添加日期时间本地化，优化翻译加载
- **Acceptance Criteria Addressed**: AC-9
- **Test Requirements**:
  - `human-judgment` 9.1: i18n 覆盖范围审计完成
  - `human-judgment` 9.2: 新增语言支持完成
  - `programmatic` 9.3: 动态语言切换正常工作
  - `human-judgment` 9.4: 日期时间本地化正常
  - `programmatic` 9.5: 翻译文件加载优化
- **Notes**: 国际化相关代码位于 `src/frontend/src/i18n/`

### 子任务分解
- [ ] 子任务 9.1: 审计现有 i18n 覆盖范围
  - 文件: `docs/i18n/coverage-audit.md`
- [ ] 子任务 9.2: 添加更多语言支持
  - 文件: `src/frontend/src/i18n/locales/`
- [ ] 子任务 9.3: 实现动态语言切换
  - 文件: `src/frontend/src/components/LanguageSwitcher.jsx`
- [ ] 子任务 9.4: 添加日期时间本地化
  - 文件: `src/frontend/src/utils/date.js`
- [ ] 子任务 9.5: 优化翻译文件加载
  - 文件: `src/frontend/src/i18n/index.js`

---

## [ ] 大任务 10: 后端日志聚合
- **Priority**: P1
- **Depends On**: None
- **Description**: 配置结构化日志格式，实现日志分级管理，添加请求追踪 ID，配置日志输出格式化，添加敏感信息过滤
- **Acceptance Criteria Addressed**: AC-10
- **Test Requirements**:
  - `programmatic` 10.1: 结构化日志格式配置完成
  - `programmatic` 10.2: 日志分级管理正常
  - `programmatic` 10.3: 请求追踪 ID 正常工作
  - `programmatic` 10.4: 日志输出格式化完成
  - `programmatic` 10.5: 敏感信息过滤正常
- **Notes**: 日志相关代码位于 `src/backend/config/logging.js`

### 子任务分解
- [ ] 子任务 10.1: 配置结构化日志格式
  - 文件: `src/backend/config/logging.js`
- [ ] 子任务 10.2: 实现日志分级管理
  - 文件: `src/backend/utils/logger.js`
- [ ] 子任务 10.3: 添加请求追踪 ID
  - 文件: `src/backend/middleware/requestId.js`
- [ ] 子任务 10.4: 配置日志输出格式化
  - 文件: `src/backend/utils/logger.js`
- [ ] 子任务 10.5: 添加敏感信息过滤
  - 文件: `src/backend/middleware/sanitizeLogs.js`

---

## [ ] 大任务 11: 移动端PWA优化
- **Priority**: P2
- **Depends On**: 大任务 6
- **Description**: 完善 Service Worker，添加离线缓存策略，实现推送通知，优化 Manifest 配置，添加 PWA 安装提示
- **Acceptance Criteria Addressed**: AC-11
- **Test Requirements**:
  - `human-judgment` 11.1: Service Worker 完善
  - `human-judgment` 11.2: 离线缓存策略生效
  - `human-judgment` 11.3: 推送通知正常工作
  - `human-judgment` 11.4: Manifest 配置优化
  - `human-judgment` 11.5: PWA 安装提示正常
- **Notes**: PWA 相关代码位于 `src/frontend/public/`

### 子任务分解
- [ ] 子任务 11.1: 完善 Service Worker
  - 文件: `src/frontend/public/sw.js`
- [ ] 子任务 11.2: 添加离线缓存策略
  - 文件: `src/frontend/public/sw.js`
- [ ] 子任务 11.3: 实现推送通知
  - 文件: `src/frontend/src/services/pushNotification.js`
- [ ] 子任务 11.4: 优化 Manifest 配置
  - 文件: `src/frontend/public/manifest.json`
- [ ] 子任务 11.5: 添加 PWA 安装提示
  - 文件: `src/frontend/src/components/PWAInstallPrompt.jsx`

---

## [ ] 大任务 12: 数据库连接池优化
- **Priority**: P2
- **Depends On**: 大任务 3
- **Description**: 分析当前连接池配置，优化连接池参数，添加连接健康检查，实现连接泄漏检测，配置连接池监控
- **Acceptance Criteria Addressed**: AC-12
- **Test Requirements**:
  - `programmatic` 12.1: 连接池配置分析完成
  - `programmatic` 12.2: 连接池参数优化生效
  - `programmatic` 12.3: 连接健康检查正常
  - `programmatic` 12.4: 连接泄漏检测正常
  - `programmatic` 12.5: 连接池监控正常
- **Notes**: 数据库连接池位于 `config/database/db.js` 和 `src/backend/config/database/dbPoolManager.js`

### 子任务分解
- [ ] 子任务 12.1: 分析当前连接池配置
  - 文件: `docs/database/pool-analysis.md`
- [ ] 子任务 12.2: 优化连接池参数
  - 文件: `src/backend/config/database/dbPoolManager.js`
- [ ] 子任务 12.3: 添加连接健康检查
  - 文件: `src/backend/config/database/dbPoolManager.js`
- [ ] 子任务 12.4: 实现连接泄漏检测
  - 文件: `src/backend/config/database/dbPoolManager.js`
- [ ] 子任务 12.5: 配置连接池监控
  - 文件: `src/backend/services/metricsService.js`

---

## [ ] 大任务 13: 前端组件库文档
- **Priority**: P2
- **Depends On**: 大任务 6
- **Description**: 配置 Storybook 文档工具，编写基础组件文档，添加组件示例代码，配置 Props 表格生成，部署组件文档网站
- **Acceptance Criteria Addressed**: AC-13
- **Test Requirements**:
  - `human-judgment` 13.1: Storybook 配置完成
  - `human-judgment` 13.2: 基础组件文档完整
  - `human-judgment` 13.3: 组件示例代码正常
  - `human-judgment` 13.4: Props 表格生成正常
  - `human-judgment` 13.5: 组件文档网站可访问
- **Notes**: 组件位于 `src/frontend/src/components/`

### 子任务分解
- [ ] 子任务 13.1: 配置 Storybook 文档工具
  - 文件: `.storybook/main.js`, `.storybook/preview.js`
- [ ] 子任务 13.2: 编写基础组件文档
  - 文件: `src/frontend/src/components/**/*.stories.jsx`
- [ ] 子任务 13.3: 添加组件示例代码
  - 文件: `src/frontend/src/components/**/*.stories.jsx`
- [ ] 子任务 13.4: 配置 Props 表格生成
  - 文件: `.storybook/preview.js`
- [ ] 子任务 13.5: 部署组件文档网站
  - 文件: `package.json`, `.github/workflows/deploy-docs.yml`

---

## [ ] 大任务 14: 后端API文档自动更新
- **Priority**: P2
- **Depends On**: 大任务 1
- **Description**: 配置 Swagger 自动生成，添加 API 变更检测，实现文档版本管理，添加 API 使用统计，配置文档 CI 检查
- **Acceptance Criteria Addressed**: AC-14
- **Test Requirements**:
  - `programmatic` 14.1: Swagger 自动生成配置完成
  - `programmatic` 14.2: API 变更检测正常
  - `programmatic` 14.3: 文档版本管理正常
  - `programmatic` 14.4: API 使用统计正常
  - `programmatic` 14.5: 文档 CI 检查通过
- **Notes**: API 文档配置位于 `src/backend/config/swagger.js`

### 子任务分解
- [ ] 子任务 14.1: 配置 Swagger 自动生成
  - 文件: `src/backend/config/swagger.js`
- [ ] 子任务 14.2: 添加 API 变更检测
  - 文件: `scripts/check-api-changes.js`
- [ ] 子任务 14.3: 实现文档版本管理
  - 文件: `docs/api/versions/`
- [ ] 子任务 14.4: 添加 API 使用统计
  - 文件: `src/backend/middleware/apiUsage.js`
- [ ] 子任务 14.5: 配置文档 CI 检查
  - 文件: `.github/workflows/check-docs.yml`

---

## [ ] 大任务 15: CI/CD测试覆盖率检查
- **Priority**: P2
- **Depends On**: 大任务 1, 大任务 2
- **Description**: 配置覆盖率阈值检查，添加覆盖率下降告警，生成覆盖率趋势报告，配置分支覆盖率要求，集成到 PR 检查流程
- **Acceptance Criteria Addressed**: AC-15
- **Test Requirements**:
  - `programmatic` 15.1: 覆盖率阈值配置完成
  - `programmatic` 15.2: 覆盖率下降告警正常
  - `programmatic` 15.3: 覆盖率趋势报告生成
  - `programmatic` 15.4: 分支覆盖率要求配置完成
  - `programmatic` 15.5: PR 检查流程集成完成
- **Notes**: CI/CD 配置位于 `.github/workflows/`

### 子任务分解
- [ ] 子任务 15.1: 配置覆盖率阈值检查
  - 文件: `jest.config.js`
- [ ] 子任务 15.2: 添加覆盖率下降告警
  - 文件: `.github/workflows/ci.yml`
- [ ] 子任务 15.3: 生成覆盖率趋势报告
  - 文件: `scripts/generate-coverage-trend.js`
- [ ] 子任务 15.4: 配置分支覆盖率要求
  - 文件: `jest.config.js`
- [ ] 子任务 15.5: 集成到 PR 检查流程
  - 文件: `.github/workflows/ci.yml`

---

## [ ] 大任务 16: 前端无障碍访问优化
- **Priority**: P2
- **Depends On**: 大任务 6
- **Description**: 添加 ARIA 标签，优化键盘导航，添加屏幕阅读器支持，配置 a11y 测试，添加颜色对比度优化
- **Acceptance Criteria Addressed**: AC-16
- **Test Requirements**:
  - `human-judgment` 16.1: ARIA 标签完整
  - `human-judgment` 16.2: 键盘导航正常
  - `human-judgment` 16.3: 屏幕阅读器支持正常
  - `programmatic` 16.4: a11y 测试通过
  - `human-judgment` 16.5: 颜色对比度优化
- **Notes**: 组件位于 `src/frontend/src/components/`

### 子任务分解
- [ ] 子任务 16.1: 添加 ARIA 标签
  - 文件: `src/frontend/src/components/**/*.jsx`
- [ ] 子任务 16.2: 优化键盘导航
  - 文件: `src/frontend/src/components/**/*.jsx`
- [ ] 子任务 16.3: 添加屏幕阅读器支持
  - 文件: `src/frontend/src/components/**/*.jsx`
- [ ] 子任务 16.4: 配置 a11y 测试
  - 文件: `jest.config.js`, `src/frontend/__tests__/a11y.test.js`
- [ ] 子任务 16.5: 添加颜色对比度优化
  - 文件: `src/frontend/src/styles/global.css`

---

## [ ] 大任务 17: 后端错误追踪系统
- **Priority**: P2
- **Depends On**: 大任务 10
- **Description**: 集成 Sentry 错误追踪，配置错误分组，添加性能监控，配置告警规则，集成源码映射
- **Acceptance Criteria Addressed**: AC-17
- **Test Requirements**:
  - `programmatic` 17.1: Sentry 集成完成
  - `programmatic` 17.2: 错误分组配置完成
  - `programmatic` 17.3: 性能监控正常
  - `programmatic` 17.4: 告警规则配置完成
  - `programmatic` 17.5: 源码映射集成完成
- **Notes**: 错误追踪相关代码

### 子任务分解
- [ ] 子任务 17.1: 集成 Sentry 错误追踪
  - 文件: `src/backend/config/sentry.js`
- [ ] 子任务 17.2: 配置错误分组
  - 文件: `src/backend/config/sentry.js`
- [ ] 子任务 17.3: 添加性能监控
  - 文件: `src/backend/config/sentry.js`
- [ ] 子任务 17.4: 配置告警规则
  - 文件: `.sentryclirc`
- [ ] 子任务 17.5: 集成源码映射
  - 文件: `webpack.config.js` (如需要)

---

## [ ] 大任务 18: 数据库备份恢复自动化
- **Priority**: P2
- **Depends On**: 大任务 3, 大任务 12
- **Description**: 配置自动备份脚本，实现增量备份，添加备份验证，实现定时恢复演练，编写备份恢复文档
- **Acceptance Criteria Addressed**: AC-18
- **Test Requirements**:
  - `programmatic` 18.1: 自动备份脚本配置完成
  - `programmatic` 18.2: 增量备份正常工作
  - `programmatic` 18.3: 备份验证正常
  - `programmatic` 18.4: 定时恢复演练完成
  - `programmatic` 18.5: 备份恢复文档完整
- **Notes**: 备份脚本位于 `scripts/`

### 子任务分解
- [ ] 子任务 18.1: 配置自动备份脚本
  - 文件: `scripts/backup.sh`
- [ ] 子任务 18.2: 实现增量备份
  - 文件: `scripts/backup-incremental.sh`
- [ ] 子任务 18.3: 添加备份验证
  - 文件: `scripts/verify-backup.sh`
- [ ] 子任务 18.4: 实现定时恢复演练
  - 文件: `scripts/test-restore.sh`
- [ ] 子任务 18.5: 编写备份恢复文档
  - 文件: `docs/database/backup-restore.md`

---

## [ ] 大任务 19: 前端SEO优化
- **Priority**: P2
- **Depends On**: 大任务 6
- **Description**: 添加 Meta 标签，配置 Open Graph，添加结构化数据，优化页面标题，添加 robots.txt
- **Acceptance Criteria Addressed**: AC-19
- **Test Requirements**:
  - `human-judgment` 19.1: Meta 标签完整
  - `human-judgment` 19.2: Open Graph 配置完成
  - `human-judgment` 19.3: 结构化数据添加完成
  - `human-judgment` 19.4: 页面标题优化完成
  - `human-judgment` 19.5: robots.txt 配置完成
- **Notes**: SEO 相关代码位于 `src/frontend/index.html` 和组件中

### 子任务分解
- [ ] 子任务 19.1: 添加 Meta 标签
  - 文件: `src/frontend/src/pages/**/*.jsx`
- [ ] 子任务 19.2: 配置 Open Graph
  - 文件: `src/frontend/index.html`
- [ ] 子任务 19.3: 添加结构化数据
  - 文件: `src/frontend/src/components/StructuredData.jsx`
- [ ] 子任务 19.4: 优化页面标题
  - 文件: `src/frontend/src/pages/**/*.jsx`
- [ ] 子任务 19.5: 添加 robots.txt
  - 文件: `src/frontend/public/robots.txt`

---

## [ ] 大任务 20: 后端GraphQL API
- **Priority**: P2
- **Depends On**: 大任务 1
- **Description**: 评估 GraphQL 需求，配置 Apollo Server，定义 GraphQL Schema，实现查询和变更，添加 GraphQL Playground
- **Acceptance Criteria Addressed**: AC-20
- **Test Requirements**:
  - `programmatic` 20.1: GraphQL 需求评估完成
  - `programmatic` 20.2: Apollo Server 配置完成
  - `programmatic` 20.3: GraphQL Schema 定义完成
  - `programmatic` 20.4: 查询和变更实现完成
  - `programmatic` 20.5: GraphQL Playground 可访问
- **Notes**: GraphQL 相关代码

### 子任务分解
- [ ] 子任务 20.1: 评估 GraphQL 需求
  - 文件: `docs/api/graphql-requirements.md`
- [ ] 子任务 20.2: 配置 Apollo Server
  - 文件: `src/backend/config/apollo.js`
- [ ] 子任务 20.3: 定义 GraphQL Schema
  - 文件: `src/backend/graphql/schema.graphql`
- [ ] 子任务 20.4: 实现查询和变更
  - 文件: `src/backend/graphql/resolvers.js`
- [ ] 子任务 20.5: 添加 GraphQL Playground
  - 文件: `src/backend/config/apollo.js`
