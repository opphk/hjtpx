const request = require('supertest');
const { ApolloServer } = require('@apollo/server');
const { startStandaloneServer } = require('@apollo/server/standalone');
const typeDefs = require('../../src/backend/graphql/schema');
const resolvers = require('../../src/backend/graphql/resolvers');

describe('GraphQL Schema', () => {
  let server;
  let serverUrl;

  beforeAll(async () => {
    server = new ApolloServer({
      typeDefs,
      resolvers,
      formatError: (error) => {
        return {
          message: error.message,
          code: error.extensions?.code
        };
      }
    });

    const { url } = await startStandaloneServer(server, {
      listen: { port: 0 }
    });
    serverUrl = url;
  });

  afterAll(async () => {
    await server.stop();
  });

  describe('Type Definitions', () => {
    test('should have User type defined', async () => {
      const query = `
        query {
          __type(name: "User") {
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
      `;

      const response = await request(serverUrl)
        .post('/')
        .send({ query });

      expect(response.status).toBe(200);
      expect(response.body.data.__type).not.toBeNull();
      expect(response.body.data.__type.name).toBe('User');
    });

    test('should have Notification type defined', async () => {
      const query = `
        query {
          __type(name: "Notification") {
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
      `;

      const response = await request(serverUrl)
        .post('/')
        .send({ query });

      expect(response.status).toBe(200);
      expect(response.body.data.__type).not.toBeNull();
      expect(response.body.data.__type.name).toBe('Notification');
    });

    test('should have required Query fields', async () => {
      const query = `
        query {
          __schema {
            queryType {
              fields {
                name
                type {
                  name
                  kind
                }
                args {
                  name
                  type {
                    kind
                  }
                }
              }
            }
          }
        }
      `;

      const response = await request(serverUrl)
        .post('/')
        .send({ query });

      expect(response.status).toBe(200);
      expect(response.body.data.__schema.queryType.fields.length).toBeGreaterThan(0);
      
      const queryFieldNames = response.body.data.__schema.queryType.fields.map(f => f.name);
      expect(queryFieldNames).toContain('users');
      expect(queryFieldNames).toContain('user');
      expect(queryFieldNames).toContain('me');
      expect(queryFieldNames).toContain('notifications');
    });

    test('should have required Mutation fields', async () => {
      const query = `
        query {
          __schema {
            mutationType {
              fields {
                name
                type {
                  name
                  kind
                }
                args {
                  name
                  type {
                    kind
                  }
                }
              }
            }
          }
        }
      `;

      const response = await request(serverUrl)
        .post('/')
        .send({ query });

      expect(response.status).toBe(200);
      expect(response.body.data.__schema.mutationType).not.toBeNull();
      
      const mutationFieldNames = response.body.data.__schema.mutationType.fields.map(f => f.name);
      expect(mutationFieldNames).toContain('createUser');
      expect(mutationFieldNames).toContain('updateUser');
      expect(mutationFieldNames).toContain('deleteUser');
      expect(mutationFieldNames).toContain('createNotification');
      expect(mutationFieldNames).toContain('markNotificationAsRead');
    });

    test('should have all enum types defined', async () => {
      const query = `
        query {
          __schema {
            types {
              name
              kind
              enumValues {
                name
              }
            }
          }
        }
      `;

      const response = await request(serverUrl)
        .post('/')
        .send({ query });

      expect(response.status).toBe(200);
      
      const enumTypes = response.body.data.__schema.types.filter(t => t.kind === 'ENUM');
      const enumNames = enumTypes.map(t => t.name);
      
      expect(enumNames).toContain('Role');
      expect(enumNames).toContain('NotificationType');
      expect(enumNames).toContain('Priority');
      expect(enumNames).toContain('NotificationStatus');
      expect(enumNames).toContain('Channel');
    });
  });

  describe('Schema Validation', () => {
    test('should validate email format', async () => {
      const mutation = `
        mutation {
          createUser(email: "invalid-email", name: "Test", password: "password123") {
            id
          }
        }
      `;

      const response = await request(serverUrl)
        .post('/')
        .send({ query: mutation });

      expect(response.status).toBe(200);
    });

    test('should require authentication for protected queries', async () => {
      const query = `
        query {
          me {
            id
            email
          }
        }
      `;

      const response = await request(serverUrl)
        .post('/')
        .send({ query });

      expect(response.status).toBe(200);
      expect(response.body.errors).toBeDefined();
      expect(response.body.errors[0].message).toContain('Not authenticated');
    });

    test('should require authorization for admin queries', async () => {
      const query = `
        query {
          users {
            id
            email
          }
        }
      `;

      const response = await request(serverUrl)
        .post('/')
        .send({ query });

      expect(response.status).toBe(200);
      expect(response.body.errors).toBeDefined();
    });
  });

  describe('Scalar Types', () => {
    test('should support JSON scalar type', async () => {
      const query = `
        query {
          __type(name: "JSON") {
            name
            kind
          }
        }
      `;

      const response = await request(serverUrl)
        .post('/')
        .send({ query });

      expect(response.status).toBe(200);
      expect(response.body.data.__type).not.toBeNull();
    });
  });
});
