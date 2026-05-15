# HJTPX GraphQL API 实现文档

## 概述

本项目已成功实现后端GraphQL API，使用Apollo Server作为GraphQL服务器，提供灵活的数据查询和变更能力。

## 实现详情

### 1. Apollo Server 配置
**文件**: [src/backend/config/apollo.js](file:///workspace/hjtpx/src/backend/config/apollo.js)

**功能**:
- 配置Apollo Server实例
- 集成Express中间件
- 配置GraphQL Playground（开发环境启用）
- 错误格式化和日志记录
- 认证上下文解析
- 性能监控插件
- 追踪配置

**环境变量**:
- `GRAPHQL_PLAYGROUND`: 启用/禁用Playground
- `GRAPHQL_INTROSPECTION`: 启用/禁用内省
- `GRAPHQL_ENDPOINT`: GraphQL端点路径
- `GRAPHQL_LOG_RESPONSES`: 记录响应日志

### 2. GraphQL Schema
**文件**: [src/backend/graphql/schema.js](file:///workspace/hjtpx/src/backend/graphql/schema.js)

**定义类型**:
- **枚举类型**:
  - `Role`: admin, user, moderator
  - `NotificationType`: info, success, warning, error, system, message, reminder, alert
  - `Priority`: low, normal, high, urgent
  - `NotificationStatus`: unread, read, archived
  - `Channel`: in_app, email, sms, push

- **对象类型**:
  - `User`: 用户对象（id, email, name, role, created_at, updated_at）
  - `Notification`: 通知对象（完整字段支持）
  - `Pagination`: 分页信息
  - `NotificationsResponse`: 通知列表响应

- **标量类型**:
  - `JSON`: 自定义JSON标量类型

**查询(Query)**:
```graphql
users: [User!]!
user(id: ID!): User
me: User
notifications(status, type, page, limit, sortBy, order): NotificationsResponse!
notification(id: ID!): Notification
unreadNotificationsCount: Int!
```

**变更(Mutation)**:
```graphql
createUser(email, name, password, role): User!
updateUser(id, email, name, password, role): User
deleteUser(id): Boolean!
createNotification(userId, type, title, message, priority, actionUrl, actionLabel, channels): Notification!
markNotificationAsRead(id): Notification
markAllNotificationsAsRead: Boolean!
```

### 3. Resolvers 实现
**文件**: [src/backend/graphql/resolvers.js](file:///workspace/hjtpx/src/backend/graphql/resolvers.js)

**功能**:
- 查询解析器（Query resolvers）
- 变更解析器（Mutation resolvers）
- 字段解析器（Field resolvers）
- JSON标量类型实现
- 完整的错误处理
- 认证和授权检查

### 4. GraphQL Playground 配置
**特性**:
- 开发环境自动启用
- 深色主题
- 自定义编辑器设置
- 查询历史记录
- 预配置查询示例
- Schema轮询（开发环境）
- 追踪扩展支持

### 5. 测试覆盖
**文件位置**: [tests/graphql/](file:///workspace/hjtpx/tests/graphql/)

**测试文件**:
- [schema.test.js](file:///workspace/hjtpx/tests/graphql/schema.test.js): Schema验证测试
- [resolvers.test.js](file:///workspace/hjtpx/tests/graphql/resolvers.test.js): Resolver单元测试
- [error-handling.test.js](file:///workspace/hjtpx/tests/graphql/error-handling.test.js): 错误处理测试
- [performance.test.js](file:///workspace/hjtpx/tests/graphql/performance.test.js): 性能测试

**测试统计**:
- 测试套件: 4个通过
- 测试用例: 62个通过
- 覆盖范围: Schema、Resolvers、错误处理、性能

## 测试运行

```bash
# 运行所有GraphQL测试
npm run test:graphql

# 运行单个测试文件
npm run test:graphql -- schema.test.js

# 查看测试覆盖
npm run test:coverage -- --testPathPattern=graphql
```

## 性能基准

基于测试结果：
- 并发查询 (10x100用户): ~608ms
- 大数据集查询 (1000用户): ~239ms
- 深度嵌套查询 (50用户x10通知): ~85ms
- 复杂过滤查询: ~33ms
- 批量变更 (20个): ~621ms
- 连续快速变更 (50个): ~598ms
- 内存使用: 无内存泄漏（100次查询）
- 平均响应时间: ~10.70ms

## 安全特性

1. **认证**: JWT Token验证
2. **授权**: 
   - 用户查询需要认证
   - 管理功能需要admin角色
   - 用户只能修改自己的资料
3. **输入验证**: 必填字段和类型检查
4. **错误处理**: 安全的错误信息返回

## 集成方式

```javascript
const { createApolloServer, startApolloServer } = require('./config/apollo');
const express = require('express');

const app = express();

// 创建并启动Apollo Server
const server = createApolloServer();
await startApolloServer(server, app);
```

## 环境配置

在 `.env` 文件中配置：

```env
GRAPHQL_PLAYGROUND=true
GRAPHQL_INTROSPECTION=true
GRAPHQL_ENDPOINT=/graphql
GRAPHQL_LOG_RESPONSES=false
NODE_ENV=development
```

## 未来扩展

建议的后续功能：
1. 添加订阅(Subscriptions)支持实时通知
2. 实现数据加载器(DataLoader)优化N+1查询
3. 添加查询复杂度限制
4. 实现查询缓存
5. 添加批量查询支持
6. 实现分页游标(Pagination Cursors)
7. 添加字段级权限控制

## 相关文档

- [Apollo Server文档](https://www.apollographql.com/docs/apollo-server/)
- [GraphQL规范](https://graphql.org/)
- [项目需求文档](file:///workspace/hjtpx/src/backend/graphql/requirements.md)
