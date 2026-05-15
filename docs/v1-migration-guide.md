# HJTPX API v1 迁移指南

## 1. 概述

本文档详细说明如何从 HJTPX API v1 迁移到 v2。v2 版本带来了增强的响应格式、改进的错误处理、默认分页功能和更好的性能优化。

## 2. 迁移时间表

### 2.1 重要日期

| 阶段 | 日期 | 说明 |
|------|------|------|
| v2 发布 | 2026-01-01 | v2 正式发布 |
| v1 弃用开始 | 2026-01-01 | v1 标记为弃用，开始接收弃用警告 |
| v1 废弃日期 | 2026-07-01 | v1 将不再可用 |

### 2.2 迁移窗口

**剩余时间**: 请查看响应头 `X-API-Days-Until-Sunset` 获取具体天数。

**建议**: 在 2026-06-01 前完成迁移，以确保有充足的测试和缓冲时间。

## 3. 主要破坏性变更

### 3.1 移除的端点

v1 中以下端点在 v2 中已被移除：

```javascript
// v1 中存在但 v2 中已移除
DELETE /api/v1/auth/logout
POST /api/v1/users/bulk-create
GET /api/v1/analytics/legacy
```

**替代方案**: 使用 v2 的标准 RESTful 端点或查看新的分析 API。

### 3.2 响应格式变更

#### 3.2.1 标准响应格式

**v1 响应格式**:
```json
{
  "success": true,
  "data": {
    "id": 1,
    "name": "Test User",
    "email": "test@example.com"
  }
}
```

**v2 响应格式**:
```json
{
  "success": true,
  "data": {
    "id": 1,
    "name": "Test User",
    "email": "test@example.com",
    "created_at": "2026-01-15T10:30:00.000Z"
  },
  "meta": {
    "version": "v2",
    "timestamp": "2026-05-15T12:00:00.000Z",
    "requestId": "req_abc123"
  }
}
```

#### 3.2.2 用户对象变更

**v1 用户对象**:
```javascript
{
  id: 1,
  name: "John Doe",
  email: "john@example.com",
  role: "user"
}
```

**v2 用户对象**:
```javascript
{
  id: 1,
  name: "John Doe",
  email: "john@example.com",
  role: "user",
  created_at: "2026-01-15T10:30:00.000Z",
  profile: {
    avatar: "https://example.com/avatar.jpg",
    bio: "Software Developer",
    location: "San Francisco"
  }
}
```

### 3.3 分页变更

#### 3.3.1 默认分页

v1 返回完整列表，v2 默认启用分页。

**v1 请求**:
```http
GET /api/v1/users
```

**v1 响应** (完整列表):
```json
{
  "success": true,
  "data": [
    { "id": 1, "name": "User 1" },
    { "id": 2, "name": "User 2" },
    { "id": 3, "name": "User 3" }
  ]
}
```

**v2 请求**:
```http
GET /api/v2/users?page=1&limit=10
```

**v2 响应** (分页格式):
```json
{
  "success": true,
  "data": {
    "users": [
      { "id": 1, "name": "User 1" }
    ],
    "pagination": {
      "page": 1,
      "limit": 10,
      "total": 100,
      "total_pages": 10,
      "has_next": true,
      "has_prev": false
    }
  },
  "meta": {
    "version": "v2",
    "timestamp": "2026-05-15T12:00:00.000Z"
  }
}
```

#### 3.3.2 分页参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| page | integer | 1 | 当前页码 |
| limit | integer | 10 | 每页项目数 (最大 100) |
| sort | string | 'created_at' | 排序字段 |
| order | string | 'desc' | 排序方向 (asc/desc) |

## 4. 弃用字段

### 4.1 废弃的请求参数

| v1 参数 | v2 替代 | 说明 |
|---------|---------|------|
| `include_deleted` | 使用 `filter` | 过滤已删除项目 |
| `raw_response` | 默认返回结构化数据 | 不再支持原始响应 |
| `compact` | 默认启用 | v2 始终返回紧凑格式 |

### 4.2 废弃的响应字段

| v1 字段 | v2 状态 | 说明 |
|---------|---------|------|
| `legacy_id` | 已移除 | 使用 `id` 替代 |
| `metadata.raw` | 已移除 | 使用 `meta` 替代 |
| `status_text` | 已移除 | HTTP 状态码已足够 |

## 5. 迁移步骤

### 5.1 准备工作

```bash
# 1. 备份当前配置
cp config/api.config.js config/api.config.js.backup

# 2. 创建测试环境
docker-compose -f docker-compose.test.yml up -d

# 3. 检查当前使用的 API 版本
curl -I https://api.example.com/api/users
# 查看 X-API-Version 响应头
```

### 5.2 代码更新

#### 5.2.1 更新 API 客户端配置

```javascript
// 更新 API 配置
const apiConfig = {
  baseURL: '/api/v2',
  headers: {
    'Accept': 'application/vnd.hjtpx.v2+json',
    'Content-Type': 'application/json'
  },
  timeout: 10000
};

// 使用新的分页参数
const fetchUsers = async (page = 1, limit = 10) => {
  const response = await axios.get('/api/v2/users', {
    params: { page, limit, sort: 'created_at', order: 'desc' }
  });
  return response.data.data;
};
```

#### 5.2.2 更新响应解析逻辑

```javascript
// v1 响应解析
const getUserData = (response) => response.data;

// v2 响应解析
const getUserData = (response) => {
  const { data, meta } = response;
  
  if (meta?.version !== 'v2') {
    console.warn('Unexpected API version:', meta?.version);
  }
  
  return data;
};

// 处理分页响应
const getPaginatedData = (response) => {
  const { users, pagination } = response.data;
  return {
    items: users,
    page: pagination.page,
    totalPages: pagination.total_pages,
    hasNext: pagination.has_next,
    hasPrev: pagination.has_prev
  };
};
```

#### 5.2.3 添加错误处理

```javascript
// v2 增强的错误处理
const handleApiError = (error) => {
  if (error.response) {
    const { status, data } = error.response;
    
    switch (status) {
      case 400:
        return handleValidationError(data);
      case 401:
        return handleAuthError(data);
      case 404:
        return handleNotFoundError(data);
      case 429:
        return handleRateLimitError(data);
      default:
        return handleGenericError(data);
    }
  }
  
  if (error.deprecation?.deprecated) {
    console.warn('Using deprecated API:', error.deprecation.message);
  }
};
```

### 5.3 测试验证

#### 5.3.1 功能测试

```bash
# 测试基本端点
curl -X GET "http://localhost:3000/api/v2/health" \
  -H "Accept: application/vnd.hjtpx.v2+json"

# 测试分页
curl -X GET "http://localhost:3000/api/v2/users?page=1&limit=5" \
  -H "Accept: application/vnd.hjtpx.v2+json"

# 测试弃用警告（v1）
curl -X GET "http://localhost:3000/api/v1/users" -I
```

#### 5.3.2 自动化测试

```javascript
// test/migration.test.js
describe('API v2 Migration Tests', () => {
  test('should return v2 response format', async () => {
    const response = await request(app).get('/api/v2/users');
    
    expect(response.status).toBe(200);
    expect(response.body).toHaveProperty('meta');
    expect(response.body.meta.version).toBe('v2');
    expect(response.body).toHaveProperty('data.users');
    expect(response.body.data).toHaveProperty('pagination');
  });

  test('should include deprecation headers for v1', async () => {
    const response = await request(app).get('/api/v1/users');
    
    expect(response.headers['warning']).toBeDefined();
    expect(response.headers['x-api-deprecation-date']).toBeDefined();
    expect(response.headers['x-api-sunset-date']).toBeDefined();
    expect(response.headers['x-api-migration-guide']).toBeDefined();
  });

  test('should negotiate version correctly', async () => {
    const response = await request(app)
      .get('/api/users')
      .set('Accept', 'application/vnd.hjtpx.v2+json');
    
    expect(response.headers['x-api-version']).toBe('v2');
    expect(response.headers['x-api-latest-version']).toBe('v2');
  });
});
```

### 5.4 生产部署

```bash
# 1. 部署到预生产环境
kubectl apply -f k8s/api-v2-deployment.yml

# 2. 运行完整测试套件
npm test -- --grep "v2"

# 3. 监控错误率和性能指标
kubectl logs -f deployment/api-v2 | grep ERROR

# 4. 切换流量（蓝绿部署）
kubectl patch service api-service -p '{"spec":{"selector":{"version":"v2"}}}'

# 5. 监控迁移进度
watch -n 5 'curl -s http://api.example.com/metrics | grep api_version'
```

## 6. 版本协商

### 6.1 多种版本指定方式

v2 支持多种版本协商方式，客户端可以根据需要选择：

#### 6.1.1 URL 路径方式（推荐）

```http
GET /api/v2/users
GET /api/v1/users
```

#### 6.1.2 Accept Header 方式

```http
GET /api/users
Accept: application/vnd.hjtpx.v2+json
```

#### 6.1.3 自定义 Header 方式

```http
GET /api/users
X-API-Version: v2
```

#### 6.1.4 Prefer Header 方式

```http
GET /api/users
Prefer: version=v2
```

### 6.2 版本协商响应头

| 响应头 | 说明 | 示例 |
|--------|------|------|
| X-API-Version | 当前使用版本 | v2 |
| X-API-Version-Status | 版本状态 | stable/deprecated |
| X-API-Supported-Versions | 支持的所有版本 | v1, v2 |
| X-API-Latest-Version | 最新稳定版本 | v2 |
| X-API-Version-Negotiated | 是否进行了版本协商 | true/false |
| X-API-Original-Version | 原始请求版本（如有协商） | v1 |

## 7. 弃用警告详解

### 7.1 响应头中的弃用信息

使用 v1 时，会收到以下弃用警告头：

```
HTTP/1.1 200 OK
Warning: 299 - "API v1 is deprecated. Please upgrade to v2"
X-API-Deprecation-Date: 2026-01-01
X-API-Sunset-Date: 2026-07-01
X-API-Migration-Guide: /docs/v1-migration-guide.md
X-API-Breaking-Changes: 3
X-API-Days-Until-Sunset: 47
X-API-Version: v1
X-API-Version-Status: deprecated
```

### 7.2 响应体中的弃用信息

v1 响应体也会包含弃用信息：

```json
{
  "success": true,
  "data": {...},
  "deprecation": {
    "deprecated": true,
    "message": "API v1 is deprecated. Please upgrade to the latest version.",
    "currentVersion": "v1",
    "latestVersion": "v2",
    "deprecationDate": "2026-01-01",
    "sunsetDate": "2026-07-01",
    "migrationGuide": "/docs/v1-migration-guide.md",
    "breakingChanges": [
      "Removed legacy authentication endpoints",
      "Changed response format for user endpoints",
      "Removed deprecated fields"
    ]
  }
}
```

## 8. 性能考虑

### 8.1 v2 性能优化

v2 包含以下性能改进：

- 更高效的响应序列化
- 默认启用 gzip 压缩
- 优化的数据库查询
- 改进的缓存策略

### 8.2 迁移后验证

```bash
# 性能基准测试
npm run benchmark -- --version v2

# 监控关键指标
curl -X GET "http://api.example.com/metrics" | jq '.api_performance'
```

## 9. 回滚策略

### 9.1 回滚触发条件

在以下情况下应考虑回滚：

- 错误率超过 1%
- 响应时间增加超过 50%
- 关键业务功能失败

### 9.2 回滚步骤

```bash
# 1. 立即切换回 v1
kubectl patch service api-service -p '{"spec":{"selector":{"version":"v1"}}}'

# 2. 监控稳定性
kubectl logs -f deployment/api-v1 | grep ERROR

# 3. 分析问题并修复
# 4. 准备重新部署 v2
```

## 10. 获取帮助

### 10.1 联系方式

- 技术支持邮箱: support@hjtpx.example.com
- Slack 频道: #api-migration
- 紧急情况: +1-xxx-xxxx-xxxx

### 10.2 资源链接

- [API 文档](../docs/API_VERSIONING.md)
- [变更日志](../CHANGELOG.md)
- [状态页面](https://status.hjtpx.example.com)

## 11. 检查清单

迁移完成前请确认以下项目：

- [ ] 所有 API 调用已更新到 v2
- [ ] 响应解析逻辑已适配 v2 格式
- [ ] 分页逻辑已正确实现
- [ ] 错误处理已更新
- [ ] 测试套件已通过
- [ ] 性能指标已验证
- [ ] 文档已更新
- [ ] 团队已培训

---

**最后更新**: 2026-05-15  
**版本**: 1.0.0  
**维护者**: HJTPX 开发团队
