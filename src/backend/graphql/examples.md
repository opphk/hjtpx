# HJTPX GraphQL API 使用示例

## 访问 GraphQL Playground

访问 `http://localhost:3000/graphql` 打开 GraphQL Playground。

## 示例查询

### 1. 查询所有用户 (需要 admin 权限)

```graphql
query GetAllUsers {
  users {
    id
    email
    name
    role
    created_at
  }
}
```

### 2. 查询单个用户

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

### 3. 查询当前用户信息 (需要认证)

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

### 4. 查询通知列表 (需要认证)

```graphql
query GetNotifications($status: NotificationStatus, $type: NotificationType) {
  notifications(status: $status, type: $type) {
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
  "type": null
}
```

### 5. 查询通知详情 (需要认证)

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

### 6. 查询未读通知数量 (需要认证)

```graphql
query GetUnreadCount {
  unreadNotificationsCount
}
```

## 示例变更 (Mutations)

### 1. 创建新用户 (需要 admin 权限)

```graphql
mutation CreateUser($email: String!, $name: String!, $password: String!, $role: Role) {
  createUser(email: $email, name: $name, password: $password, role: $role) {
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
  "name": "New User",
  "password": "password123",
  "role": "user"
}
```

### 2. 更新用户信息 (需要认证)

```graphql
mutation UpdateUser($id: ID!, $email: String, $name: String, $password: String) {
  updateUser(id: $id, email: $email, name: $name, password: $password) {
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
  "name": "Updated Name"
}
```

### 3. 删除用户 (需要 admin 权限)

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

### 4. 创建通知 (需要认证)

```graphql
mutation CreateNotification(
  $userId: ID!,
  $type: NotificationType!,
  $title: String!,
  $message: String!,
  $priority: Priority,
  $actionUrl: String,
  $actionLabel: String
) {
  createNotification(
    userId: $userId,
    type: $type,
    title: $title,
    message: $message,
    priority: $priority,
    actionUrl: $actionUrl,
    actionLabel: $actionLabel
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

### 5. 标记通知为已读 (需要认证)

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

### 6. 标记所有通知为已读 (需要认证)

```graphql
mutation MarkAllAsRead {
  markAllNotificationsAsRead
}
```

## 认证

对于需要认证的操作，需要在 HTTP 请求头中添加:

```
Authorization: Bearer <your-jwt-token>
```

在 GraphQL Playground 中，可以在左下角的 "HTTP HEADERS" 面板中添加:

```json
{
  "Authorization": "Bearer <your-jwt-token>"
}
```
