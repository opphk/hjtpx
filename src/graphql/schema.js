const { gql } = require('graphql-tag');

const typeDefs = gql`
  type User {
    id: ID!
    username: String!
    email: String!
    role: String
    createdAt: String
    updatedAt: String
  }

  type HealthStatus {
    status: String!
    timestamp: String!
    uptime: Float
    environment: String!
    version: String!
  }

  type ServiceInfo {
    name: String!
    status: String!
    version: String!
  }

  type SystemInfo {
    nodeVersion: String!
    platform: String!
    environment: String!
    totalMemory: Float
    freeMemory: Float
    cpus: Int
  }

  type DatabaseInfo {
    status: String!
    type: String!
    connection: String
  }

  type DetailedHealth {
    status: String!
    timestamp: String!
    services: [ServiceInfo!]!
    system: SystemInfo!
    database: DatabaseInfo!
  }

  type Query {
    user(id: ID!): User
    users(limit: Int, offset: Int): [User!]!
    health: HealthStatus!
    detailedHealth: DetailedHealth!
    version: String!
    environment: String!
  }

  input CreateUserInput {
    username: String!
    email: String!
    role: String
  }

  input UpdateUserInput {
    username: String
    email: String
    role: String
  }

  type Mutation {
    createUser(input: CreateUserInput!): User!
    updateUser(id: ID!, input: UpdateUserInput!): User
    deleteUser(id: ID!): Boolean!
  }
`;

module.exports = typeDefs;
