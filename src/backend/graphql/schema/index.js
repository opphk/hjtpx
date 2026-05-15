const { gql } = require('graphql-tag');

const typeDefs = gql`
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

  type Pagination {
    page: Int!
    limit: Int!
    total: Int!
    pages: Int!
  }

  type User {
    id: ID!
    email: String!
    name: String!
    role: Role!
    created_at: String!
    updated_at: String
    notifications(limit: Int, status: NotificationStatus): [Notification]
    unreadNotificationsCount: Int
  }

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

  type NotificationsResponse {
    notifications: [Notification!]!
    pagination: Pagination!
  }

  type AuthPayload {
    token: String!
    user: User!
  }

  type Query {
    users(limit: Int, offset: Int): [User!]!
    user(id: ID!): User
    me: User

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

  type Mutation {
    createUser(email: String!, name: String!, password: String!, role: Role = user): User!
    updateUser(id: ID!, email: String, name: String, password: String, role: Role): User
    deleteUser(id: ID!): Boolean!

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
    deleteNotification(id: ID!): Boolean!

    login(email: String!, password: String!): AuthPayload!
    register(email: String!, name: String!, password: String!): AuthPayload!
  }

  type Subscription {
    notificationCreated(userId: ID): Notification!
    notificationUpdated(userId: ID!): Notification!
    notificationDeleted(userId: ID!): ID!
    userUpdated: User!
  }

  scalar JSON
`;

module.exports = typeDefs;
