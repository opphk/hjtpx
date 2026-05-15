# Sentry 配置指南

## 概述

本项目已集成 Sentry 错误追踪系统，支持 Node.js 主项目和 Go 语言 captchax 子项目。Sentry 提供了强大的错误追踪、性能监控和告警功能。

## 1. Sentry 配置更新

### 1.1 核心功能特性

- **错误追踪**: 自动捕获未处理的异常和错误
- **性能监控**: 追踪 HTTP 请求、数据库查询、WebSocket 连接等
- **自定义分组**: 基于错误类型、端点、环境进行智能分组
- **告警通知**: 支持 Slack、邮件等多渠道通知
- **源码映射**: 支持 sourcemaps 上传，便于错误调试

### 1.2 环境变量配置

在 `.env.production` 或 `.env.staging` 中配置：

```env
# Sentry 配置
SENTRY_DSN=https://your-dsn@sentry.io/project-id
SENTRY_ENVIRONMENT=production
SENTRY_RELEASE=hjtpx@1.0.0
SENTRY_TRACES_SAMPLE_RATE=0.1
SENTRY_PROFILES_SAMPLE_RATE=0.05
SENTRY_DEBUG=false
SENTRY_SPOTLIGHT=false
```

## 2. Sentry 配置详情

### 2.1 集成配置

[Sentry 配置文件](file:///workspace/hjtpx/src/backend/config/sentry.js) 包含以下功能：

- **HTTP 集成**: 自动追踪所有 HTTP 请求
- **Express 集成**: 追踪 Express.js 路由和中间件
- **MongoDB 集成**: 追踪 Mongoose 数据库操作
- **性能分析**: 使用 `@sentry/profiling-node` 进行性能分析

### 2.2 错误过滤

配置了以下错误过滤规则：

```javascript
ignoreErrors: [
  'Network Error',
  'ECONNREFUSED',
  'ETIMEDOUT',
  'timeout',
  'Navigation aborted',
]
```

### 2.3 忽略的 URL

以下浏览器扩展相关的请求不会被上报：

```javascript
denyUrls: [
  /chrome-extension:\/\//i,
  /safari-extension:\/\//i,
  /moz-extension:\/\//i,
]
```

## 3. 错误分组策略

### 3.1 自定义指纹

使用 `beforeSend` 回调为特定错误设置自定义指纹：

- **ValidationError**: `['validation-error', '错误类型']`
- **CSRF Token Error**: `['csrf-token-error']`
- **Network Error**: 自动标记 `network_error: true` 标签

### 3.2 错误严重性

不同类型的错误被标记为不同的严重级别：

- **Network Errors**: `Info` 级别（避免告警疲劳）
- **Validation Errors**: `Warning` 级别
- **Security Errors (CSRF)**: `Error` 级别
- **Database Errors**: `Error` 级别

### 3.3 性能监控过滤

以下端点的事务不会被追踪：

- `/health`
- `/ping`
- `/favicon`

## 4. 性能监控

### 4.1 采样率策略

根据环境自动调整采样率：

| 环境 | tracesSampleRate | profilesSampleRate |
|------|------------------|--------------------|
| production | 0.1 (10%) | 0.05 (5%) |
| staging | 0.3 (30%) | 0.1 (10%) |
| development | 1.0 (100%) | 0.5 (50%) |

### 4.2 慢请求监控

当请求响应时间超过 1 秒时，会自动添加性能面包屑：

```javascript
if (duration > 1000) {
  Sentry.addBreadcrumb({
    category: 'performance',
    message: `Slow request: ${req.method} ${req.path}`,
    level: Sentry.Severity.Warning,
    data: { duration, method, path, status },
  });
}
```

### 4.3 数据库查询监控

通过 `captureDatabaseError` 函数追踪数据库错误：

```javascript
const { captureDatabaseError } = require('./config/sentry');

try {
  await User.find({});
} catch (error) {
  captureDatabaseError(error, 'find', 'users');
}
```

### 4.4 WebSocket 监控

追踪 WebSocket 连接错误：

```javascript
const { captureWebSocketError } = require('./config/sentry');

captureWebSocketError(error, 'connection', socketId);
```

### 4.5 GraphQL 监控

追踪 GraphQL 查询错误：

```javascript
const { captureGraphQLError } = require('./config/sentry');

captureGraphQLError(error, 'getUsers', variables);
```

## 5. 告警规则

### 5.1 告警配置文件

[Sentry 告警规则](file:///workspace/hjtpx/monitoring/alerts.yml) 包含以下告警：

| 告警名称 | 条件 | 通知渠道 | 节流时间 |
|---------|------|---------|---------|
| High Error Rate | 5分钟内 > 100 个错误 | Slack #alerts | 15m |
| Performance Degradation | P95 > 3s | Slack #performance-alerts | 5m |
| New Error Type | 首次出现 | Email | 1h |
| Database Connection Errors | ECONNREFUSED | Slack #database-alerts | 10m |
| Authentication Errors Spike | > 50 个认证错误 | Slack #security-alerts | 5m |
| Error Rate Spike | 10分钟内 > 20 个错误 | Slack + Email | 10m |
| API Latency Degradation | P99 > 5s | Slack #performance-alerts | 15m |
| Memory Usage High | > 512 MB | Slack #performance-alerts | 30m |

### 5.2 告警配置说明

告警规则支持以下组件：

- **condition**: 触发条件（如错误数量、响应时间等）
- **filters**: 过滤条件（如环境、错误类型等）
- **action**: 通知动作（Slack、Email）
- **throttle**: 节流时间（避免告警疲劳）

## 6. 源码映射 (Source Maps)

### 6.1 Sentry CLI 配置

[.sentryclirc](file:///workspace/hjtpx/.sentryclirc) 配置文件：

```ini
[auth]
token=%SENTRY_AUTH_TOKEN%

[defaults]
url=https://sentry.io
org=%SENTRY_ORG%
project=hjtpx-api
```

### 6.2 CI 中的 Sourcemaps 上传

在 GitHub Actions 中自动执行：

```yaml
- name: Create Sentry Release
  run: |
    sentry-cli releases new "$SENTRY_RELEASE"
    sentry-cli releases set-commits "$SENTRY_RELEASE" --auto
    sentry-cli releases finalize "$SENTRY_RELEASE"

- name: Upload Source Maps
  run: npm run sentry:sourcemaps
  env:
    SENTRY_AUTH_TOKEN: ${{ secrets.SENTRY_AUTH_TOKEN }}
```

### 6.3 NPM 脚本

```json
{
  "scripts": {
    "sentry:test": "node scripts/test-sentry.js",
    "sentry:sourcemaps": "node scripts/upload-sourcemaps.js",
    "sentry:release": "sentry-cli releases new $SENTRY_RELEASE",
    "sentry:deploy": "sentry-cli releases deploys $SENTRY_RELEASE production"
  }
}
```

## 7. 测试 Sentry 集成

### 7.1 运行测试

```bash
# 运行 Sentry 相关测试
npm test -- tests/unit/sentry.test.js

# 测试 Sentry 连接
npm run sentry:test
```

### 7.2 测试覆盖

单元测试包含以下场景：

- ✅ Sentry 初始化（有无 DSN）
- ✅ 环境配置（production/staging/development）
- ✅ 错误过滤（health check、validation error、CSRF）
- ✅ 性能监控（慢请求检测）
- ✅ 错误捕获（database、WebSocket、GraphQL）
- ✅ 用户上下文设置
- ✅ 性能标签和指标

## 8. 最佳实践

### 8.1 错误上报指南

```javascript
// ✅ 推荐：添加上下文信息
Sentry.captureException(error, {
  tags: {
    feature: 'user-authentication',
    environment: NODE_ENV,
  },
  extra: {
    userId: user.id,
    requestPath: req.path,
  },
});

// ❌ 避免：包含敏感信息
Sentry.captureException(error, {
  extra: {
    password: user.password,  // 敏感信息
    apiKey: 'sk_live_xxx',    // 密钥
  },
});
```

### 8.2 敏感信息清理

`sanitizeVariables` 函数自动清理以下敏感字段：

- `password`
- `token`
- `secret`
- `apiKey`
- `api_key`
- `authorization`

### 8.3 性能采样策略

- **生产环境**: 采样率控制在 10-30%，降低 Sentry 使用成本
- **预发布环境**: 采样率 30-50%，便于测试监控功能
- **开发环境**: 采样率 100%，获得完整追踪数据

### 8.4 告警优化

- **避免告警疲劳**: 为每个告警设置合理的节流时间
- **分级告警**: 区分 Critical、Warning、Info 级别
- **多渠道通知**: 重要告警使用 Slack + Email 双渠道

## 9. 故障排除

### 9.1 错误没有上报

1. 检查 DSN 是否正确配置
2. 验证环境变量是否加载：`console.log(process.env.SENTRY_DSN)`
3. 查看控制台日志是否有 Sentry 初始化信息
4. 运行 `npm run sentry:test` 测试连接

### 9.2 Sourcemaps 不工作

1. 确认 sourcemaps 已生成（在构建过程中）
2. 检查 release 版本是否匹配
3. 验证文件路径是否正确（相对于项目根目录）
4. 在 Sentry 控制台检查上传的 sourcemaps

### 9.3 告警未触发

1. 检查告警规则配置是否正确
2. 验证环境变量（`SENTRY_ENVIRONMENT`）
3. 确认通知渠道配置（Slack webhook、Email）
4. 检查节流时间（throttle）是否已过

## 10. 环境配置示例

### 10.1 开发环境 (.env)

```env
NODE_ENV=development
SENTRY_DSN=https://test@sentry.io/project-id
SENTRY_ENVIRONMENT=development
SENTRY_RELEASE=hjtpx@local
SENTRY_TRACES_SAMPLE_RATE=1.0
SENTRY_PROFILES_SAMPLE_RATE=0.5
SENTRY_DEBUG=true
```

### 10.2 生产环境 (.env.production)

```env
NODE_ENV=production
SENTRY_DSN=https://xxx@sentry.io/xxx
SENTRY_ENVIRONMENT=production
SENTRY_RELEASE=hjtpx@1.0.0
SENTRY_TRACES_SAMPLE_RATE=0.1
SENTRY_PROFILES_SAMPLE_RATE=0.05
SENTRY_DEBUG=false
```

## 11. 完整文件索引

| 文件 | 说明 |
|------|------|
| [src/backend/config/sentry.js](file:///workspace/hjtpx/src/backend/config/sentry.js) | Sentry 核心配置 |
| [.sentryclirc](file:///workspace/hjtpx/.sentryclirc) | Sentry CLI 配置 |
| [monitoring/alerts.yml](file:///workspace/hjtpx/monitoring/alerts.yml) | 告警规则配置 |
| [scripts/test-sentry.js](file:///workspace/hjtpx/scripts/test-sentry.js) | Sentry 连接测试脚本 |
| [scripts/upload-sourcemaps.js](file:///workspace/hjtpx/scripts/upload-sourcemaps.js) | Sourcemaps 上传脚本 |
| [.github/workflows/ci.yml](file:///workspace/hjtpx/.github/workflows/ci.yml) | CI 中的 Sentry 集成 |
| [tests/unit/sentry.test.js](file:///workspace/hjtpx/tests/unit/sentry.test.js) | Sentry 单元测试 |

## 12. GitHub Secrets 配置

在 GitHub 仓库的 Settings → Secrets 中配置以下密钥：

| Secret 名称 | 说明 |
|------------|------|
| `SENTRY_DSN` | Sentry Data Source Name |
| `SENTRY_AUTH_TOKEN` | Sentry API 认证令牌 |
| `SENTRY_ORG` | Sentry 组织名称 |
| `SENTRY_PROJECT` | Sentry 项目名称 |
| `SLACK_WEBHOOK_URL` | Slack Webhook URL（用于告警通知） |
