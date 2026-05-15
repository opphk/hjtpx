# HJTPX API 版本控制策略

## 1. 概述

本文档描述了 HJTPX 项目的 API 版本控制策略、迁移计划和最佳实践。v2 版本引入了增强的版本协商机制和改进的弃用警告系统。

## 2. 版本控制方法

### 2.1 版本标识

- **格式**: 语义化版本 `v{major}`（如 `v1`, `v2`）
- **策略**: 仅在引入破坏性变更时增加主版本号
- **示例**:
  - `v1` - 当前弃用版本（2026-07-01 废弃）
  - `v2` - 最新稳定版本

### 2.2 版本指定方式

支持四种方式指定 API 版本（按优先级从高到低）：

1. **URL 路径前缀**（推荐）: `/api/v1/users`
2. **Accept Header**: `Accept: application/vnd.hjtpx.v1+json`
3. **自定义 Header**: `X-API-Version: v1`
4. **Prefer Header**: `Prefer: version=v1`

### 2.3 默认版本

- 当未指定版本时，默认使用最新稳定版本（当前为 `v2`）

### 2.4 版本协商流程

```
Client Request
     ↓
URL Path Check (/api/v1/*)
     ↓ (if not found)
Accept Header Check (application/vnd.hjtpx.v1+json)
     ↓ (if not found)
X-API-Version Header Check
     ↓ (if not found)
Prefer Header Check (version=v1)
     ↓ (if not found)
Default Version (v2)
```

### 2.5 版本协商响应头

| 响应头 | 说明 | 示例 |
|--------|------|------|
| X-API-Version | 当前使用版本 | v2 |
| X-API-Version-Status | 版本状态 | stable/deprecated |
| X-API-Supported-Versions | 支持的所有版本 | v1, v2 |
| X-API-Latest-Version | 最新稳定版本 | v2 |
| X-API-Version-Negotiated | 是否进行了版本协商 | true |
| X-API-Original-Version | 原始请求版本 | v1 |

## 3. 版本迁移策略

### 3.1 版本生命周期

| 阶段 | 描述 | 持续时间 |
|------|------|----------|
| 开发 | 新功能开发，可能不稳定 | 直到发布 |
| 稳定 | 推荐使用，接收安全更新 | 至少 12 个月 |
| 弃用 | 仍可用，但建议升级 | 6 个月 |
| 废弃 | 不再维护，移除路由 | - |

### 3.2 版本发布计划

- **稳定版本**: 每 12-18 个月发布一个新主版本
- **弃用警告**: 新版本发布后，旧版本立即标记为弃用
- **废弃时间**: 弃用 6 个月后正式废弃

### 3.3 当前版本状态

| 版本 | 状态 | 弃用日期 | 废弃日期 |
|------|------|----------|----------|
| v1 | deprecated | 2026-01-01 | 2026-07-01 |
| v2 | stable | - | - |

## 4. 破坏性变更定义

当以下情况发生时，需创建新的主版本：

- 删除或重命名 API 端点
- 删除或重命名请求参数
- 更改响应数据结构
- 更改 HTTP 方法或状态码
- 移除或更改认证方式

### 4.1 v1 到 v2 的破坏性变更

- 移除了遗留的身份验证端点
- 更改了用户端点的响应格式
- 移除了已弃用的字段

## 5. 弃用策略

### 5.1 警告机制

在响应头中添加弃用警告:
- `Warning: 299 - "API v1 is deprecated. Please upgrade to v2"`
- `X-API-Deprecation-Date: 2026-01-01`
- `X-API-Sunset-Date: 2026-07-01`
- `X-API-Migration-Guide: /docs/v1-migration-guide.md`
- `X-API-Breaking-Changes: 3`
- `X-API-Days-Until-Sunset: 47`

### 5.2 紧急弃用警告

当废弃日期在 30 天内时，会触发紧急警告：

```
Warning: 299 - "API v1 will be sunset in 15 days. Urgent upgrade required."
```

### 5.3 响应体弃用信息

v1 响应体包含完整的弃用信息：

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

### 5.4 通知机制

- API 文档更新
- 开发人员邮件通知
- 仪表盘警告提示
- 响应头实时警告

## 6. 回滚策略

- 保留旧版本路由直到完全废弃
- 支持同时运行多个版本
- 详细的日志记录，便于追踪
- 版本协商自动降级

## 7. 文档

- 每个版本有独立的 API 文档
- 维护版本变更日志
- 提供详细的迁移指南
- 版本协商最佳实践

## 8. 测试

### 8.1 版本共存测试

支持多个 API 版本同时运行和测试，确保平滑迁移。

### 8.2 测试覆盖

- 版本协商功能测试
- 弃用警告测试
- 响应格式测试
- 性能基准测试

## 附录

- [v1 到 v2 迁移指南](./v1-migration-guide.md)
- [API 变更日志](../CHANGELOG.md)
