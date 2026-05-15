# HJTPX GraphQL API 文档

## 概述

GraphQL API 提供了一种灵活的数据查询和变更接口，支持实时订阅功能。

## 端点信息

- **HTTP 端点**: `/graphql`
- **WebSocket 端点**: `/graphql` (用于订阅)
- **开发环境 Playground**: `/playground`

## 认证

GraphQL API 使用 JWT Token 进行认证。在请求头中添加：

```http
Authorization: Bearer <your-jwt-token>
```

## 类型定义

### 枚举类型

#### Role (用户角色)
```graphql
enum Role {
  admin      # 管理员
  user       # 普通用户
  moderator  # 版主
}
```

#### NotificationType (通知类型)
```graphql
enum NotificationType {
  info      # 信息
  success   # 成功
  warning   # 警告
  error     # 错误
  system    # 系统
  message   # 消息
  reminder  # 提醒
  alert     # 警报
}
```

#### Priority (优先级)
```graphql
enum Priority {
  low    # 低
  normal # 普通
  high   # 高
  urgent # 紧急
}
```

#### NotificationStatus (通知状态)
```graphql
enum NotificationStatus {
  unread    # 未读
  read      # 已读
  archived  # 已归档
}
```

#### Channel (通知渠道)
```graphql
enum Channel {
  in_app  # 应用内
  email   # 邮件
  sms     # 短信
  push    # 推送
}
```

### 对象类型

#### User (用户)
```graphql
type User {
  id: ID!                      # 用户ID
  email: String!               # 邮箱
  name: String!                # 名称
  role: Role!                  # 角色
  created_at: String!          # 创建时间
  updated_at: String           # 更新时间
  notifications: [Notification] # 用户通知列表
  unreadNotificationsCount: Int  # 未读通知数量
}
```

#### Notification (通知)
```graphql
type Notification {
  id: ID!                  # 通知ID
  userId: ID!              # 用户ID
  type: NotificationType!  # 通知类型
  title: String!           # 标题
  message: String!         # 消息内容
  data: JSON               # 附加数据
  priority: Priority!      # 优先级
  status: NotificationStatus!  # 状态
  readAt: String           # 阅读时间
  expiresAt: String        # 过期时间
  actionUrl: String        # 操作链接
  actionLabel: String      # 操作标签
  channels: [Channel!]!   # 通知渠道
  metadata: JSON           # 元数据
  createdAt: String!       # 创建时间
  updatedAt: String!       # 更新时间
  user: User               # 关联用户
}
```

#### Pagination (分页信息)
```graphql
type Pagination {
  page: Int!   # 当前页
  limit: Int!  # 每页数量
  total: Int!  # 总数
  pages: Int!  # 总页数
}
```

#### NotificationsResponse (通知列表响应)
```graphql
type NotificationsResponse {
  notifications: [Notification!]!  # 通知列表
  pagination: Pagination!          # 分页信息
}
```

#### AuthPayload (认证响应)
```graphql
type AuthPayload {
  token: String!  # JWT Token
  user: User!     # 用户信息
}
```

### 标量类型

#### JSON
自定义 JSON 标量类型，用于存储任意 JSON 数据。

## 查询 (Query)

### 获取所有用户 (需要 admin 权限)

```graphql
query GetUsers($limit: Int, $offset: Int) {
  users(limit: $limit, offset: $offset) {
    id
    email
    name
    role
    created_at
  }
}
```

**变量:**
```json
{
  "limit": 10,
  "offset": 0
}
```

### 获取单个用户

```graphql
query GetUser($id: ID!) {
  user(id: $id) {
    id
    email
    name
    role
    created_at
  }
}
```

**变量:**
```json
{
  "id": "1"
}
```

### 获取当前用户

```graphql
query GetMe {
  me {
    id
    email
    name
    role
    created_at
    unreadNotificationsCount
  }
}
```

### 获取通知列表

```graphql
query GetNotifications(
  $status: NotificationStatus,
  $type: NotificationType,
  $page: Int,
  $limit: Int,
  $sortBy: String,
  $order: String
) {
  notifications(
    status: $status,
    type: $type,
    page: $page,
    limit: $limit,
    sortBy: $sortBy,
    order: $order
  ) {
    notifications {
      id
      title
      message
      type
      status
      priority
      createdAt
    }
    pagination {
      page
      limit
      total
      pages
    }
  }
}
```

**变量:**
```json
{
  "status": "unread",
  "type": "info",
  "page": 1,
  "limit": 20,
  "sortBy": "createdAt",
  "order": "desc"
}
```

### 获取单个通知

```graphql
query GetNotification($id: ID!) {
  notification(id: $id) {
    id
    title
    message
    type
    status
    priority
    data
    createdAt
    user {
      id
      name
      email
    }
  }
}
```

**变量:**
```json
{
  "id": "60d21b4667d0d8992e610c85"
}
```

### 获取未读通知数量

```graphql
query GetUnreadCount {
  unreadNotificationsCount
}
```

## 变更 (Mutation)

### 用户注册

```graphql
mutation Register($email: String!, $name: String!, $password: String!) {
  register(email: $email, name: $name, password: $password) {
    token
    user {
      id
      email
      name
      role
    }
  }
}
```

**变量:**
```json
{
  "email": "user@example.com",
  "name": "用户名",
  "password": "password123"
}
```

### 用户登录

```graphql
mutation Login($email: String!, $password: String!) {
  login(email: $email, password: $password) {
    token
    user {
      id
      email
      name
      role
    }
  }
}
```

**变量:**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

### 创建用户 (需要 admin 权限)

```graphql
mutation CreateUser(
  $email: String!,
  $name: String!,
  $password: String!,
  $role: Role
) {
  createUser(
    email: $email,
    name: $name,
    password: $password,
    role: $role
  ) {
    id
    email
    name
    role
    created_at
  }
}
```

**变量:**
```json
{
  "email": "newuser@example.com",
  "name": "新用户",
  "password": "password123",
  "role": "user"
}
```

### 更新用户

```graphql
mutation UpdateUser(
  $id: ID!,
  $email: String,
  $name: String,
  $password: String,
  $role: Role
) {
  updateUser(
    id: $id,
    email: $email,
    name: $name,
    password: $password,
    role: $role
  ) {
    id
    email
    name
    role
    created_at
  }
}
```

**变量:**
```json
{
  "id": "1",
  "name": "更新后的名称"
}
```

### 删除用户 (需要 admin 权限)

```graphql
mutation DeleteUser($id: ID!) {
  deleteUser(id: $id)
}
```

**变量:**
```json
{
  "id": "1"
}
```

### 创建通知

```graphql
mutation CreateNotification(
  $userId: ID!,
  $type: NotificationType!,
  $title: String!,
  $message: String!,
  $priority: Priority,
  $actionUrl: String,
  $actionLabel: String,
  $channels: [Channel!]
) {
  createNotification(
    userId: $userId,
    type: $type,
    title: $title,
    message: $message,
    priority: $priority,
    actionUrl: $actionUrl,
    actionLabel: $actionLabel,
    channels: $channels
  ) {
    id
    title
    message
    type
    status
    priority
    createdAt
  }
}
```

**变量:**
```json
{
  "userId": "1",
  "type": "info",
  "title": "新通知",
  "message": "这是一条测试通知",
  "priority": "normal"
}
```

### 标记通知为已读

```graphql
mutation MarkAsRead($id: ID!) {
  markNotificationAsRead(id: $id) {
    id
    status
    readAt
  }
}
```

**变量:**
```json
{
  "id": "60d21b4667d0d8992e610c85"
}
```

### 标记所有通知为已读

```graphql
mutation MarkAllAsRead {
  markAllNotificationsAsRead
}
```

### 删除通知

```graphql
mutation DeleteNotification($id: ID!) {
  deleteNotification(id: $id)
}
```

**变量:**
```json
{
  "id": "60d21b4667d0d8992e610c85"
}
```

## 订阅 (Subscription)

### 订阅新通知

```graphql
subscription OnNotificationCreated($userId: ID) {
  notificationCreated(userId: $userId) {
    id
    title
    message
    type
    priority
    createdAt
  }
}
```

**变量:**
```json
{
  "userId": "1"
}
```

### 订阅通知更新

```graphql
subscription OnNotificationUpdated($userId: ID!) {
  notificationUpdated(userId: $userId) {
    id
    status
    readAt
    updatedAt
  }
}
```

### 订阅通知删除

```graphql
subscription OnNotificationDeleted($userId: ID!) {
  notificationDeleted(userId: $userId)
}
```

### 订阅用户更新

```graphql
subscription OnUserUpdated {
  userUpdated {
    id
    name
    role
    updatedAt
  }
}
```

## 错误处理

GraphQL API 返回以下错误代码：

| 代码 | 说明 |
|------|------|
| `BAD_USER_INPUT` | 输入参数错误 |
| `AUTHENTICATION_ERROR` | 需要认证 |
| `FORBIDDEN` | 权限不足 |
| `INTERNAL_ERROR` | 服务器内部错误 |

## 使用示例

### 使用 curl

**查询:**
```bash
curl -X POST http://localhost:3000/graphql \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-token>" \
  -d '{
    "query": "{ me { id email name role } }"
  }'
```

**变更:**
```bash
curl -X POST http://localhost:3000/graphql \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-token>" \
  -d '{
    "query": "mutation { login(email: \"user@example.com\", password: \"password\") { token user { id name } } }"
  }'
```

### 使用 JavaScript

```javascript
const fetchGraphQL = async (query, variables = {}) => {
  const response = await fetch('/graphql', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${localStorage.getItem('token')}`
    },
    body: JSON.stringify({ query, variables })
  });
  
  const { data, errors } = await response.json();
  if (errors) {
    throw new Error(errors[0].message);
  }
  return data;
};

// 获取当前用户
const me = await fetchGraphQL(`
  query {
    me {
      id
      email
      name
      role
    }
  }
`);

// 登录
const login = await fetchGraphQL(`
  mutation Login($email: String!, $password: String!) {
    login(email: $email, password: $password) {
      token
      user {
        id
        name
      }
    }
  }
`, { email: 'user@example.com', password: 'password123' });
```

## 开发工具

### GraphQL Playground

在开发环境中，访问 `/playground` 可以使用交互式 GraphQL Playground。

### 内省

支持 GraphQL 内省，可通过以下查询获取完整的 schema：

```graphql
{
  __schema {
    types {
      name
      fields {
        name
        type {
          name
          kind
        }
      }
    }
  }
}
```

## 性能优化

API 使用了以下优化：

1. **DataLoader** - 批量加载和缓存，避免 N+1 查询问题
2. **查询缓存** - 使用 Redis 缓存常用查询结果
3. **索引** - 数据库索引优化查询性能

## 安全考虑

1. 所有敏感操作都需要认证
2. 管理员操作需要 admin 角色权限
3. 用户只能访问自己的资源
4. 输入参数经过验证和清理
