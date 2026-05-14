# Changelog

所有重要的项目变更都将记录在此文件中。

格式遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/) 规范。

## [2.0.0] - 2026-05-14

### Added

#### 新功能
- **批量验证 API**：新增 `/api/v2/captcha/batch/verify` 端点，支持一次请求验证多个验证码
- **场景管理 API**：新增完整的场景管理功能，支持创建、查询、更新、删除验证场景
- **Webhook 回调**：支持注册 Webhook 回调地址，接收验证完成事件通知
- **OpenAPI 规范**：提供完整的 OpenAPI 3.0 规范文档 (`docs/openapi.yaml`)

#### SDK 更新
- **Python SDK**：新增异步方法支持 (`create_slider_captcha_async`)
- **Go SDK**：新增 Gin 框架中间件示例
- **Java SDK**：新增 Spring Boot 集成示例

#### 安全增强
- **请求去重**：通过 `X-Deduplication-ID` 头支持幂等操作
- **响应压缩**：支持 gzip 和 Brotli 压缩
- **缓存策略**：支持 ETag 和条件请求

### Changed

- **API 响应格式**：统一使用标准响应结构 `{code, message, data}`
- **错误码体系**：扩展错误码，支持更精确的错误定位
- **文档结构**：重新组织文档目录，新增多语言 SDK 指南

### Deprecated

- **v1 API 部分端点**：将在 v3.0 中移除，请迁移至 v2 API

### Fixed

- 修复高并发下 Redis 连接池耗尽问题
- 修复验证码过期后仍可验证的问题
- 修复某些图片格式显示异常的问题

---

## [1.5.0] - 2026-04-01

### Added

- **管理后台白名单/黑名单 API**：支持通过 API 管理 IP 白名单和黑名单
- **统计分析 API**：新增验证统计数据查询接口
- **Redis Cluster 支持**：支持 Redis Cluster 部署模式

### Changed

- **性能优化**：改进图片生成算法，减少 CPU 占用
- **缓存优化**：实现多级缓存策略，提升命中率

### Fixed

- 修复内存泄漏问题
- 修复数据库连接泄漏问题

---

## [1.4.0] - 2026-03-01

### Added

- **点选验证码**：支持中文字符点选验证
- **拼图验证码**：新增拼图验证模式
- **风险评分引擎**：基于多维度行为分析的风险评分系统

### Changed

- **前端组件**：重构前端组件，支持多种验证类型
- **错误处理**：统一错误处理机制

---

## [1.3.0] - 2026-02-01

### Added

- **滑块验证码**：基础滑块验证功能
- **管理后台**：可视化配置面板
- **JWT 认证**：管理后台 JWT 认证

### Changed

- **数据库优化**：添加索引，优化查询性能
- **日志系统**：改进日志记录，便于问题排查

---

## [1.2.0] - 2026-01-01

### Added

- **Redis 缓存**：验证码结果缓存
- **IP 限流**：基于 IP 的请求限流
- **Circuit Breaker**：熔断保护机制

### Changed

- **架构优化**：引入中间件层，统一处理横切关注点

---

## [1.1.0] - 2025-12-01

### Added

- **会话管理**：验证码会话管理
- **负载均衡**：基础负载均衡支持
- **健康检查**：服务健康检查端点

---

## [1.0.0] - 2025-11-01

### Added

- 初始版本发布
- 基础 API 功能
- Docker 部署支持
- 基础管理功能

---

## [Unreleased]

### Planned

- [ ] Kubernetes Operator 支持
- [ ] 更多 SDK 语言支持 (Rust, Swift, Kotlin)
- [ ] 机器学习驱动的风险分析
- [ ] 多租户支持
- [ ] 国际化 (i18n)

---

## 版本号规则

版本号遵循 [语义化版本 2.0.0](https://semver.org/lang/zh-CN/) 规范：

- **MAJOR**: 主版本号 - 包含不兼容的 API 变更
- **MINOR**: 次版本号 - 保持向后兼容的功能新增
- **PATCH**: 修订号 - 保持向后兼容的问题修复

## 更新日志类型

- **Added**: 新功能
- **Changed**: 功能变更
- **Deprecated**: 已弃用功能
- **Removed**: 已移除功能
- **Fixed**: 问题修复
- **Security**: 安全相关修复

---

## 迁移指南

### 从 v1 迁移到 v2

#### API 变更

1. **基础 URL 变更**
   - v1: `http://localhost:8080/api/v1/captcha/slider`
   - v2: `http://localhost:8080/api/v2/captcha/slider`

2. **响应格式统一**
   ```json
   // v1 (不一致)
   {"success": true}

   // v2 (统一)
   {"code": 200, "message": "success", "data": {...}}
   ```

3. **新增必需头**
   - 推荐添加 `X-Request-ID` 用于请求追踪

#### 认证变更

- 管理后台 API 现在需要 JWT Token 认证
- Token 通过 Cookie 或 `Authorization` 头传递

#### 缓存策略

- v2 支持 HTTP 缓存机制
- 推荐使用 ETag 进行条件请求

---

## 如何使用本文件

1. **查找特定版本**：搜索 `[version]` 或版本号
2. **了解新功能**：查看 `Added` 部分
3. **问题排查**：查看 `Fixed` 部分
4. **迁移准备**：查看变更和弃用说明
