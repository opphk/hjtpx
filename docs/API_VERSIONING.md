# HJTPX API 版本控制与迁移指南

## 目录

1. [概述](#1-概述)
2. [版本控制方法](#2-版本控制方法)
3. [版本协商机制](#3-版本协商机制)
4. [版本迁移策略](#4-版本迁移策略)
5. [弃用警告机制](#5-弃用警告机制)
6. [版本共存测试](#6-版本共存测试)
7. [最佳实践](#7-最佳实践)
8. [故障排除](#8-故障排除)

---

## 1. 概述

HJTPX API 采用语义化版本控制策略，支持多版本共存和灵活的版本协商机制。本文档详细说明了 API 版本控制的实现方式、迁移策略和最佳实践。

### 1.1 当前版本状态

| 版本 | 状态 | 弃用日期 | 废弃日期 | 说明 |
|------|------|----------|----------|------|
| v1 | deprecated | 2026-01-01 | 2026-07-01 | 已弃用，建议迁移到 v2 |
| v2 | stable | - | - | 当前推荐版本 |

### 1.2 版本生命周期

```
开发(Development) → 稳定(Stable) → 弃用(Deprecated) → 废弃(Sunset)
     ↓                    ↓                ↓                ↓
   功能开发            推荐使用         警告通知          停止服务
   可能不稳定          安全更新         建议升级          路由移除
```

| 阶段 | 描述 | 持续时间 |
|------|------|----------|
| 开发 | 新功能开发，可能不稳定 | 直到发布 |
| 稳定 | 推荐使用，接收安全更新 | 至少 12 个月 |
| 弃用 | 仍可用，但建议升级 | 6 个月 |
| 废弃 | 不再维护，移除路由 | - |

---

## 2. 版本控制方法

### 2.1 版本标识格式

- **格式**: `v{major}`（如 `v1`, `v2`）
- **策略**: 仅在引入破坏性变更时增加主版本号
- **向后兼容**: 同一主版本内保持向后兼容

### 2.2 破坏性变更定义

以下情况需要创建新的主版本：

- 删除或重命名 API 端点
- 删除或重命名请求参数
- 更改响应数据结构
- 更改 HTTP 方法或状态码
- 移除或更改认证方式
- 移除或重命名响应字段

### 2.3 v1 到 v2 的破坏性变更

| 变更类型 | v1 | v2 | 影响 |
|----------|-----|-----|------|
| 认证方式 | Basic Auth | JWT Token | 需要更新认证逻辑 |
| 响应格式 | 扁平结构 | 嵌套结构(data/meta) | 需要更新解析逻辑 |
| 分页 | 不支持 | 默认支持 | 需要添加分页参数 |
| 用户字段 | 基础字段 | 扩展字段(profile) | 需要适配字段变化 |

---

## 3. 版本协商机制

### 3.1 支持的协商方式

支持五种方式指定 API 版本（按优先级从高到低）：

#### 1. URL 路径前缀（推荐）

```http
GET /api/v2/users
GET /api/v1/health
```

#### 2. Accept-Version Header

```http
GET /api/users
Accept-Version: v2
```

#### 3. Accept Header（MIME 类型）

```http
GET /api/users
Accept: application/vnd.hjtpx.v2+json
```

#### 4. 自定义 X-API-Version Header

```http
GET /api/users
X-API-Version: v2
```

#### 5. Prefer Header

```http
GET /api/users
Prefer: version=v2
```

#### 6. Query 参数

```http
GET /api/users?api-version=v2
```

### 3.2 版本协商流程

```
Client Request
     ↓
URL Path Check (/api/v1/*)
     ↓ (if not found)
Accept-Version Header Check
     ↓ (if not found)
Accept Header Check (application/vnd.hjtpx.v1+json)
     ↓ (if not found)
X-API-Version Header Check
     ↓ (if not found)
Prefer Header Check (version=v1)
     ↓ (if not found)
Query Parameter Check (?api-version=v1)
     ↓ (if not found)
Default Version (v2)
```

### 3.3 版本协商响应头

| 响应头 | 说明 | 示例 |
|--------|------|------|
| `X-API-Version` | 当前使用版本 | `v2` |
| `X-API-Version-Status` | 版本状态 | `stable` / `deprecated` |
| `X-API-Supported-Versions` | 支持的所有版本 | `v1, v2` |
| `X-API-Latest-Version` | 最新稳定版本 | `v2` |
| `X-API-Version-Negotiated` | 是否进行了版本协商 | `true` |
| `X-API-Original-Version` | 原始请求版本 | `v1` |
| `X-API-Version-Upgrade` | 版本升级提示 | `Version v1 not available. Using v2.` |

### 3.4 版本协商示例

#### cURL 示例

```bash
# URL 路径方式
curl -H "Authorization: Bearer TOKEN" \
  http://localhost:3000/api/v2/users

# Accept-Version Header 方式
curl -H "Authorization: Bearer TOKEN" \
  -H "Accept-Version: v2" \
  http://localhost:3000/api/users

# Accept Header 方式
curl -H "Authorization: Bearer TOKEN" \
  -H "Accept: application/vnd.hjtpx.v2+json" \
  http://localhost:3000/api/users

# X-API-Version Header 方式
curl -H "Authorization: Bearer TOKEN" \
  -H "X-API-Version: v2" \
  http://localhost:3000/api/users

# Query 参数方式
curl -H "Authorization: Bearer TOKEN" \
  "http://localhost:3000/api/users?api-version=v2"
```

#### JavaScript/TypeScript 示例

```typescript
// 使用 URL 路径
const response = await fetch('/api/v2/users', {
  headers: { 'Authorization': 'Bearer TOKEN' }
});

// 使用 Accept-Version Header
const response = await fetch('/api/users', {
  headers: {
    'Authorization': 'Bearer TOKEN',
    'Accept-Version': 'v2'
  }
});

// 使用 Accept Header
const response = await fetch('/api/users', {
  headers: {
    'Authorization': 'Bearer TOKEN',
    'Accept': 'application/vnd.hjtpx.v2+json'
  }
});
```

---

## 4. 版本迁移策略

### 4.1 迁移前准备

1. **评估影响范围**
   - 识别所有使用 v1 API 的客户端
   - 评估代码变更量
   - 制定测试计划

2. **阅读迁移指南**
   - 详细阅读 [v1 迁移指南](./v1-migration-guide.md)
   - 了解所有破坏性变更
   - 准备回滚方案

3. **环境准备**
   - 在开发环境验证 v2 API
   - 准备测试数据
   - 配置监控和告警

### 4.2 迁移步骤

#### 步骤 1: 更新 API 基础 URL

```javascript
// v1
const API_BASE = 'https://api.example.com/api/v1';

// v2
const API_BASE = 'https://api.example.com/api/v2';
```

#### 步骤 2: 更新认证方式

```javascript
// v1 - Basic Auth
const response = await fetch('/api/v1/users', {
  headers: {
    'Authorization': 'Basic ' + btoa(username + ':' + password)
  }
});

// v2 - JWT Token
const response = await fetch('/api/v2/users', {
  headers: {
    'Authorization': 'Bearer ' + jwtToken
  }
});
```

#### 步骤 3: 适配响应格式

```javascript
// v1 响应格式
{
  "id": 1,
  "name": "John Doe",
  "email": "john@example.com"
}

// v2 响应格式
{
  "success": true,
  "data": {
    "id": 1,
    "name": "John Doe",
    "email": "john@example.com",
    "profile": {
      "avatar": null,
      "bio": null,
      "location": null
    }
  },
  "meta": {
    "version": "v2",
    "timestamp": "2024-01-01T00:00:00.000Z"
  }
}
```

#### 步骤 4: 添加分页支持

```javascript
// v2 分页请求
const response = await fetch('/api/v2/users?page=1&limit=10', {
  headers: { 'Authorization': 'Bearer TOKEN' }
});

// v2 分页响应
{
  "success": true,
  "data": {
    "users": [...],
    "pagination": {
      "page": 1,
      "limit": 10,
      "total": 100,
      "total_pages": 10
    }
  },
  "meta": { ... }
}
```

### 4.3 迁移检查清单

- [ ] 更新所有 API 调用 URL
- [ ] 迁移认证机制到 JWT
- [ ] 更新响应解析逻辑
- [ ] 添加分页参数支持
- [ ] 更新错误处理逻辑
- [ ] 测试所有 API 端点
- [ ] 更新文档和代码注释
- [ ] 部署到生产环境
- [ ] 监控 API 使用情况

---

## 5. 弃用警告机制

### 5.1 弃用响应头

当使用已弃用的 v1 API 时，响应会包含以下头信息：

```http
Deprecation: API version v1 is deprecated since 2026-01-01
Warning: 299 - "API v1 is deprecated. Please upgrade to v2."
X-API-Deprecation-Date: 2026-01-01
X-API-Sunset-Date: 2026-07-01
X-API-Migration-Guide: /docs/v1-migration-guide.md
X-API-Breaking-Changes: 3
X-API-Days-Until-Sunset: 47
Link: </docs/v1-migration-guide.md>; rel="deprecation", <v2>; rel="successor-version"
```

### 5.2 紧急弃用警告

当废弃日期在 30 天内时，会触发紧急警告：

```http
Warning: 299 - "API v1 will be sunset in 15 days. Urgent upgrade required."
```

### 5.3 响应体弃用信息

v1 响应体包含完整的弃用信息：

```json
{
  "success": true,
  "data": { ... },
  "deprecation": {
    "deprecated": true,
    "currentVersion": "v1",
    "latestVersion": "v2",
    "deprecationDate": "2026-01-01",
    "sunsetDate": "2026-07-01",
    "migrationGuide": "/docs/v1-migration-guide.md",
    "breakingChanges": [
      "Removed legacy authentication endpoints",
      "Changed response format for user endpoints",
      "Removed deprecated fields"
    ],
    "migrationSteps": [
      {
        "step": 1,
        "title": "Update API Base URL",
        "description": "Change API base URL from /api/v1 to /api/v2",
        "action": "Replace /api/v1/ with /api/v2/ in all API calls"
      }
    ],
    "estimatedMigrationTime": "5 hours"
  }
}
```

### 5.4 客户端处理建议

```javascript
// 检查弃用警告
function checkDeprecationWarning(response) {
  const deprecation = response.headers.get('Deprecation');
  const warning = response.headers.get('Warning');
  const daysUntilSunset = response.headers.get('X-API-Days-Until-Sunset');

  if (deprecation) {
    console.warn('API Deprecation Warning:', deprecation);
    
    if (daysUntilSunset && parseInt(daysUntilSunset) <= 30) {
      console.error(`URGENT: API will be sunset in ${daysUntilSunset} days!`);
      // 发送告警通知
      notifyTeam('API sunset imminent');
    }
  }
}

// 自动检测版本并迁移
async function makeApiRequest(endpoint, options = {}) {
  const response = await fetch(endpoint, options);
  
  checkDeprecationWarning(response);
  
  const data = await response.json();
  
  if (data.deprecation) {
    console.warn('This API version is deprecated:', data.deprecation.message);
    console.log('Migration guide:', data.deprecation.migrationGuide);
  }
  
  return data;
}
```

---

## 6. 版本共存测试

### 6.1 测试策略

确保 v1 和 v2 可以同时正常运行：

1. **并行测试**: 同时测试两个版本的所有端点
2. **隔离测试**: 确保版本之间互不影响
3. **协商测试**: 验证各种协商方式正常工作
4. **弃用测试**: 验证弃用警告正确返回

### 6.2 运行版本共存测试

```bash
# 运行所有版本相关测试
npm test -- tests/versioning/

# 运行特定测试文件
npm test -- tests/versioning/api-version.test.js

# 运行版本协商测试
npm test -- tests/api/versioning.test.js
```

### 6.3 手动测试示例

```bash
# 测试 v1 和 v2 同时工作
curl http://localhost:3000/api/v1/health
curl http://localhost:3000/api/v2/health

# 测试版本协商
curl -H "Accept-Version: v1" http://localhost:3000/api/health
curl -H "Accept-Version: v2" http://localhost:3000/api/health

# 测试弃用警告
curl -i http://localhost:3000/api/v1/health
```

---

## 7. 最佳实践

### 7.1 客户端最佳实践

1. **明确指定版本**
   - 始终明确指定 API 版本
   - 不要依赖默认版本
   - 使用 URL 路径方式最可靠

2. **处理弃用警告**
   - 监控响应头中的弃用警告
   - 设置告警机制
   - 及时规划迁移

3. **优雅降级**
   - 实现版本协商失败的处理逻辑
   - 提供用户友好的错误提示
   - 记录版本协商事件

### 7.2 服务端最佳实践

1. **版本控制中间件**
   - 保持版本配置集中管理
   - 使用中间件统一处理版本协商
   - 记录版本使用统计

2. **弃用管理**
   - 提前通知客户端弃用计划
   - 提供详细的迁移指南
   - 设置合理的弃用期限

3. **文档维护**
   - 每个版本独立文档
   - 清晰的变更日志
   - 完整的迁移指南

### 7.3 版本发布检查清单

- [ ] 新版本功能完整测试
- [ ] 旧版本回归测试
- [ ] 版本协商机制测试
- [ ] 弃用警告测试
- [ ] 文档更新
- [ ] 客户端通知
- [ ] 监控配置
- [ ] 回滚方案准备

---

## 8. 故障排除

### 8.1 常见问题

#### Q: 版本协商失败怎么办？

A: 检查以下几点：
1. 确认请求头格式正确
2. 确认版本号格式正确（v1, v2）
3. 查看响应头中的 `X-API-Version` 确认实际使用的版本
4. 检查服务器日志

#### Q: 如何强制使用特定版本？

A: 使用 URL 路径方式最可靠：
```http
GET /api/v2/users
```

#### Q: 弃用警告没有显示？

A: 检查以下几点：
1. 确认使用的是 v1 API
2. 检查响应头中的 `Deprecation` 字段
3. 查看响应体中的 `deprecation` 对象
4. 确认中间件正确配置

#### Q: 如何同时测试多个版本？

A: 使用不同的 URL 路径或请求头：
```bash
# 同时测试 v1 和 v2
curl http://localhost:3000/api/v1/users &
curl http://localhost:3000/api/v2/users &
wait
```

### 8.2 调试技巧

```bash
# 查看所有响应头
curl -i http://localhost:3000/api/v1/health

# 详细查看版本协商过程
curl -v -H "Accept-Version: v1" http://localhost:3000/api/health

# 检查支持的版本
curl -I http://localhost:3000/api/health | grep X-API
```

### 8.3 联系支持

如有问题，请联系：
- 技术支持: support@hjtpx.com
- API 文档: /api-docs
- 问题反馈: /feedback

---

## 附录

### A. 相关文档

- [v1 迁移指南](./v1-migration-guide.md)
- [API 文档](./api/)
- [变更日志](../CHANGELOG.md)

### B. 版本历史

| 版本 | 发布日期 | 状态 | 说明 |
|------|----------|------|------|
| v1 | 2025-01-01 | Deprecated | 初始版本 |
| v2 | 2026-01-01 | Stable | 增强版本，改进响应格式 |

### C. HTTP 状态码

| 状态码 | 说明 |
|--------|------|
| 200 | 请求成功 |
| 400 | 请求参数错误 |
| 401 | 未授权 |
| 404 | 资源不存在 |
| 429 | 请求过于频繁 |
| 500 | 服务器内部错误 |

---

*文档版本: 2.0.0*
*最后更新: 2026-05-15*
