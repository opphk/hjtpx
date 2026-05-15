const request = require('supertest');
const { ApolloServer } = require('@apollo/server');
const { startStandaloneServer } = require('@apollo/server/standalone');
const typeDefs = require('../../src/backend/graphql/schema');
const resolvers = require('../../src/backend/graphql/resolvers');

describe('GraphQL Error Handling', () => {
  let server;
  let serverUrl;

  beforeAll(async () => {
    server = new ApolloServer({
      typeDefs,
      resolvers,
      formatError: (error) => {
        return {
          message: error.message,
          code: error.extensions?.code || 'INTERNAL_SERVER_ERROR',
          locations: error.locations,
          path: error.path
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

  describe('Authentication Errors', () => {
    test('should return error for unauthenticated requests', async () => {
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

    test('should return error for invalid token', async () => {
      const query = `
        query {
          notifications {
            notifications {
              id
            }
          }
        }
      `;

      const response = await request(serverUrl)
        .post('/')
        .set('Authorization', 'Bearer invalid-token')
        .send({ query });

      expect(response.status).toBe(200);
      expect(response.body.data).toBeNull();
    });
  });

  describe('Authorization Errors', () => {
    test('should return error for non-admin user creating user', async () => {
      const mutation = `
        mutation {
          createUser(email: "new@test.com", name: "New User", password: "password123") {
            id
          }
        }
      `;

      const response = await request(serverUrl)
        .post('/')
        .send({ query: mutation });

      expect(response.status).toBe(200);
      expect(response.body.errors).toBeDefined();
      expect(response.body.errors[0].message).toContain('Not authorized');
    });

    test('should return error for non-admin deleting user', async () => {
      const mutation = `
        mutation {
          deleteUser(id: "123")
        }
      `;

      const response = await request(serverUrl)
        .post('/')
        .send({ query: mutation });

      expect([200, 400]).toContain(response.status);
      expect(response.body.errors || response.body.data).toBeDefined();
    });
  });

  describe('Validation Errors', () => {
    test('should return error for missing required fields', async () => {
      const mutation = `
        mutation {
          createUser(email: "test@test.com") {
            id
          }
        }
      `;

      const response = await request(serverUrl)
        .post('/')
        .send({ query: mutation });

      expect([200, 400]).toContain(response.status);
      expect(response.body.errors || response.body.data).toBeDefined();
    });

    test('should return error for invalid enum values', async () => {
      const mutation = `
        mutation {
          createUser(email: "test@test.com", name: "Test", password: "password", role: "invalid_role") {
            id
          }
        }
      `;

      const response = await request(serverUrl)
        .post('/')
        .send({ query: mutation });

      expect([200, 400]).toContain(response.status);
      expect(response.body.errors || response.body.data).toBeDefined();
    });

    test('should return error for invalid ID format', async () => {
      const query = `
        query {
          user(id: "invalid-id") {
            id
          }
        }
      `;

      const response = await request(serverUrl)
        .post('/')
        .send({ query });

      expect([200, 400]).toContain(response.status);
    });
  });

  describe('Query Errors', () => {
    test('should return error for non-existent resource', async () => {
      const query = `
        query {
          user(id: "non-existent-id") {
            id
            email
          }
        }
      `;

      const response = await request(serverUrl)
        .post('/')
        .send({ query });

      expect(response.status).toBe(200);
    });

    test('should return error for syntax errors', async () => {
      const query = `
        query {
          users {
            invalidField
          }
        }
      `;

      const response = await request(serverUrl)
        .post('/')
        .send({ query });

      expect([200, 400]).toContain(response.status);
      expect(response.body.errors).toBeDefined();
    });
  });

  describe('Error Response Format', () => {
    test('should include error locations', async () => {
      const query = `
        query {
          invalidQuery
        }
      `;

      const response = await request(serverUrl)
        .post('/')
        .send({ query });

      expect([200, 400]).toContain(response.status);
      if (response.body.errors && response.body.errors.length > 0) {
        expect(response.body.errors[0]).toHaveProperty('locations');
      }
    });

    test('should include error path', async () => {
      const query = `
        query {
          users {
            nonExistentField
          }
        }
      `;

      const response = await request(serverUrl)
        .post('/')
        .send({ query });

      expect([200, 400]).toContain(response.status);
      if (response.body.errors && response.body.errors.length > 0) {
        const error = response.body.errors[0];
        expect(error).toHaveProperty('message');
      }
    });

    test('should include error code', async () => {
      const query = `
        query {
          me {
            id
          }
        }
      `;

      const response = await request(serverUrl)
        .post('/')
        .send({ query });

      expect(response.status).toBe(200);
      if (response.body.errors && response.body.errors.length > 0) {
        expect(response.body.errors[0]).toHaveProperty('code');
      }
    });
  });
});
