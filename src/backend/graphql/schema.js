const gql = require('graphql-tag');

const typeDefs = gql`
  # 枚举类型
  enum Role {
    admin
    user
    moderator
  }

  enum NotificationType {
    info
    success
    warning
    error
    system
    message
    reminder
    alert
  }

  enum Priority {
    low
    normal
    high
    urgent
  }

  enum NotificationStatus {
    unread
    read
    archived
  }

  enum Channel {
    in_app
    email
    sms
    push
  }

  # 分页信息
  type Pagination {
    page: Int!
    limit: Int!
    total: Int!
    pages: Int!
  }

  # 用户类型
  type User {
    id: ID!
    email: String!
    name: String!
    role: Role!
    created_at: String!
    updated_at: String
    notifications: [Notification]
    unreadNotificationsCount: Int
  }

  # 通知类型
  type Notification {
    id: ID!
    userId: ID!
    type: NotificationType!
    title: String!
    message: String!
    data: JSON
    priority: Priority!
    status: NotificationStatus!
    readAt: String
    expiresAt: String
    actionUrl: String
    actionLabel: String
    channels: [Channel!]!
    metadata: JSON
    createdAt: String!
    updatedAt: String!
    user: User
  }

  # 通知列表响应
  type NotificationsResponse {
    notifications: [Notification!]!
    pagination: Pagination!
  }

  # 查询
  type Query {
    # 用户查询
    users: [User!]!
    user(id: ID!): User
    me: User
    
    # 通知查询
    notifications(
      status: NotificationStatus
      type: NotificationType
      page: Int = 1
      limit: Int = 20
      sortBy: String = "createdAt"
      order: String = "desc"
    ): NotificationsResponse!
    notification(id: ID!): Notification
    unreadNotificationsCount: Int!
  }

  # 变更
  type Mutation {
    # 用户变更
    createUser(email: String!, name: String!, password: String!, role: Role = user): User!
    updateUser(id: ID!, email: String, name: String, password: String, role: Role): User
    deleteUser(id: ID!): Boolean!
    
    # 通知变更
    createNotification(
      userId: ID!
      type: NotificationType!
      title: String!
      message: String!
      priority: Priority = normal
      actionUrl: String
      actionLabel: String
      channels: [Channel!] = [in_app]
    ): Notification!
    markNotificationAsRead(id: ID!): Notification
    markAllNotificationsAsRead: Boolean!
  }

  # 自定义标量类型
  scalar JSON
`;

module.exports = typeDefs;
