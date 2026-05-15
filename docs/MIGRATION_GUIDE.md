# API v1 到 v2 迁移指南

## 概述

本文档提供从 HJTPX API v1 迁移到 v2 的完整指南。v2 是一个改进的版本，提供了更好的错误处理、分页功能和响应格式。

## 迁移时间表

| 阶段 | 日期 | 说明 |
|------|------|------|
| v2 发布 | 2026-01-01 | v2 正式发布，v1 开始弃用 |
| v1 弃用期结束 | 2026-07-01 | v1 将不再被支持 |

## 主要变更

### 1. 响应格式变化

#### v1 格式
```json
{
  "success": true,
  "data": {
    "version": "v1",
    "name": "HJTPX API v1"
  }
}
```

#### v2 格式
```json
{
  "success": true,
  "data": {
    "version": "v2",
    "name": "HJTPX API v2"
  },
  "meta": {
    "version": "v2",
    "timestamp": "2026-05-14T00:00:00.000Z"
  }
}
```

### 2. 分页功能

#### v1 (无分页)
```http
GET /api/v1/users
```

#### v2 (默认分页)
```http
GET /api/v2/users?page=1&limit=10
```

响应中包含 pagination 信息：
```json
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
  "meta": {...}
}
```

### 3. 用户详情增强

v2 的用户详情包含额外的 profile 信息：

#### v1
```json
{
  "id": 1,
  "name": "John Doe",
  "email": "john@example.com",
  "role": "user"
}
```

#### v2
```json
{
  "id": 1,
  "name": "John Doe",
  "email": "john@example.com",
  "role": "user",
  "created_at": "2024-01-01",
  "profile": {
    "avatar": null,
    "bio": null,
    "location": null
  }
}
```

### 4. 健康检查增强

v2 健康检查提供更详细的信息，包括服务状态和系统指标。

## 迁移步骤

### 步骤 1: 更新 API 端点

将所有 API 调用从 `/api/v1` 更新为 `/api/v2`。

### 步骤 2: 更新响应解析代码

确保代码能正确解析 v2 的响应格式，特别是 `meta` 字段。

### 步骤 3: 处理分页

对于返回列表的端点，添加分页支持。

### 步骤 4: 测试

在测试环境全面测试所有功能。

## 版本指定方式

### 方式 1: URL 路径（推荐）
```http
GET /api/v2/users
```

### 方式 2: Accept Header
```http
GET /api/users
Accept: application/vnd.hjtpx.v2+json
```

### 方式 3: 自定义 Header
```http
GET /api/users
X-API-Version: v2
```

## 弃用警告

使用 v1 时，响应会包含弃用警告头：

```
Warning: 299 - "API v1 is deprecated. Please upgrade to v2"
X-API-Deprecation-Date: 2026-01-01
X-API-Sunset-Date: 2026-07-01
```

## 支持

如有迁移问题，请联系开发团队。
