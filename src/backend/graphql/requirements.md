# HJTPX GraphQL API 需求文档

## 1. 概述

为 HJTPX 项目添加 GraphQL API 支持，提供灵活的数据查询和变更能力。

## 2. 数据模型

### 2.1 用户 (User)
- 存储在 PostgreSQL 数据库
- 字段：id, email, name, role, created_at, updated_at

### 2.2 通知 (Notification)
- 存储在 MongoDB 数据库
- 字段：_id, userId, type, title, message, data, priority, status, readAt, expiresAt, actionUrl, actionLabel, channels, metadata, createdAt, updatedAt

## 3. 功能需求

### 3.1 查询 (Queries)
- 查询所有用户 (admin)
- 查询单个用户 (admin 或本人)
- 查询当前用户
- 查询用户的通知列表
- 查询通知详情
- 查询用户未读通知数量

### 3.2 变更 (Mutations)
- 创建用户 (admin)
- 更新用户 (admin 或本人)
- 删除用户 (admin)
- 创建通知
- 标记通知为已读
- 标记所有通知为已读

## 4. 技术实现

- 使用 Apollo Server 作为 GraphQL 服务器
- 集成到现有 Express 应用
- 启用 GraphQL Playground 用于开发测试
