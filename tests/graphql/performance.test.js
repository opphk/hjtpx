const { ApolloServer } = require('@apollo/server');
const { startStandaloneServer } = require('@apollo/server/standalone');
const typeDefs = require('../../src/backend/graphql/schema');
const resolvers = require('../../src/backend/graphql/resolvers');

jest.mock('../../src/backend/services/userService', () => ({
  getAllUsers: jest.fn(),
  getUserById: jest.fn(),
  createUser: jest.fn(),
  updateUser: jest.fn(),
  deleteUser: jest.fn()
}));

jest.mock('../../src/backend/models/Notification', () => ({
  getUserNotifications: jest.fn(),
  findOne: jest.fn(),
  findById: jest.fn(),
  getUnreadCount: jest.fn(),
  createNotification: jest.fn(),
  markAsRead: jest.fn(),
  markAllAsRead: jest.fn()
}));

const userService = require('../../src/backend/services/userService');
const Notification = require('../../src/backend/models/Notification');

describe('GraphQL Performance', () => {
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
      listen: { port: 0 },
      context: async () => ({
        user: { id: '1', email: 'test@test.com', role: 'user' }
      })
    });
    serverUrl = url;
  });

  afterAll(async () => {
    await server.stop();
  });

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Query Performance', () => {
    test('should handle multiple concurrent queries', async () => {
      const mockUsers = Array(100).fill(null).map((_, i) => ({
        id: `${i}`,
        email: `user${i}@test.com`,
        name: `User ${i}`,
        role: 'user'
      }));
      userService.getAllUsers.mockResolvedValue(mockUsers);

      const query = `
        query {
          users {
            id
            email
            name
            role
          }
        }
      `;

      const startTime = Date.now();
      const results = await Promise.all(
        Array(10).fill(null).map(() =>
          fetch(`${serverUrl}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ query })
          }).then(res => res.json())
        )
      );
      const duration = Date.now() - startTime;

      results.forEach(result => {
        expect(result.data).toBeDefined();
        expect(result.data.users).toHaveLength(100);
      });

      console.log(`Concurrent queries (10x100 users) completed in ${duration}ms`);
      expect(duration).toBeLessThan(5000);
    });

    test('should handle large result sets efficiently', async () => {
      const mockUsers = Array(1000).fill(null).map((_, i) => ({
        id: `${i}`,
        email: `user${i}@test.com`,
        name: `User ${i}`,
        role: i % 10 === 0 ? 'admin' : 'user'
      }));
      userService.getAllUsers.mockResolvedValue(mockUsers);

      const query = `
        query {
          users {
            id
            email
            name
            role
          }
        }
      `;

      const startTime = Date.now();
      const response = await fetch(`${serverUrl}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ query })
      });
      const result = await response.json();
      const duration = Date.now() - startTime;

      expect(result.data.users).toHaveLength(1000);
      console.log(`Large result set (1000 users) query completed in ${duration}ms`);
      expect(duration).toBeLessThan(2000);
    });

    test('should handle deep nested queries efficiently', async () => {
      const mockUsers = Array(50).fill(null).map((_, i) => ({
        id: `${i}`,
        email: `user${i}@test.com`,
        name: `User ${i}`,
        role: 'user'
      }));
      userService.getAllUsers.mockResolvedValue(mockUsers);

      const mockNotifications = Array(10).fill(null).map((_, i) => ({
        _id: { toString: () => `${i}` },
        toObject: () => ({
          id: `${i}`,
          title: `Notification ${i}`,
          message: `Message ${i}`,
          type: 'info'
        })
      }));
      Notification.getUserNotifications.mockResolvedValue({
        notifications: mockNotifications,
        pagination: { page: 1, limit: 10, total: 10, pages: 1 }
      });

      const query = `
        query {
          users {
            id
            email
            name
            role
            notifications {
              id
              title
              message
              type
            }
            unreadNotificationsCount
          }
        }
      `;

      const startTime = Date.now();
      const response = await fetch(`${serverUrl}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ query })
      });
      const result = await response.json();
      const duration = Date.now() - startTime;

      expect(result.data).toBeDefined();
      console.log(`Deep nested query (50 users x 10 notifications) completed in ${duration}ms`);
      expect(duration).toBeLessThan(3000);
    });

    test('should handle complex filtering efficiently', async () => {
      const mockNotifications = Array(100).fill(null).map((_, i) => ({
        _id: { toString: () => `${i}` },
        toObject: () => ({
          id: `${i}`,
          title: `Notification ${i}`,
          message: `Message ${i}`,
          type: i % 2 === 0 ? 'info' : 'warning',
          status: i % 3 === 0 ? 'unread' : 'read'
        })
      }));
      Notification.getUserNotifications.mockResolvedValue({
        notifications: mockNotifications,
        pagination: { page: 1, limit: 20, total: 100, pages: 5 }
      });

      const query = `
        query {
          notifications(
            status: unread
            type: info
            page: 1
            limit: 20
            sortBy: "createdAt"
            order: "desc"
          ) {
            notifications {
              id
              title
              message
              type
              status
            }
            pagination {
              page
              limit
              total
              pages
            }
          }
        }
      `;

      const startTime = Date.now();
      const response = await fetch(`${serverUrl}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ query })
      });
      const result = await response.json();
      const duration = Date.now() - startTime;

      expect(result.data.notifications.notifications).toBeDefined();
      expect(result.data.notifications.pagination).toBeDefined();
      console.log(`Complex filtering query completed in ${duration}ms`);
      expect(duration).toBeLessThan(1000);
    });
  });

  describe('Mutation Performance', () => {
    test('should handle batched mutations efficiently', async () => {
      userService.createUser.mockImplementation(async (args) => ({
        id: Math.random().toString(36).substr(2, 9),
        ...args
      }));

      const mutation = `
        mutation CreateUser($email: String!, $name: String!, $password: String!) {
          createUser(email: $email, name: $name, password: $password) {
            id
            email
            name
          }
        }
      `;

      const startTime = Date.now();
      const results = await Promise.all(
        Array(20).fill(null).map((_, i) =>
          fetch(`${serverUrl}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              query: mutation,
              variables: {
                email: `user${i}@test.com`,
                name: `User ${i}`,
                password: 'password123'
              }
            })
          }).then(res => res.json())
        )
      );
      const duration = Date.now() - startTime;

      results.forEach(result => {
        expect(result.data || result.errors).toBeDefined();
      });

      console.log(`Batched mutations (20) completed in ${duration}ms`);
      expect(duration).toBeLessThan(5000);
    });

    test('should handle rapid sequential mutations', async () => {
      let notificationCount = 0;
      Notification.createNotification.mockImplementation(async () => {
        notificationCount++;
        return {
          _id: { toString: () => `${notificationCount}` },
          toObject: () => ({
            id: `${notificationCount}`,
            title: `Notification ${notificationCount}`,
            message: `Message ${notificationCount}`
          })
        };
      });

      const mutation = `
        mutation {
          createNotification(
            userId: "1"
            type: info
            title: "Test"
            message: "Test message"
          ) {
            id
            title
          }
        }
      `;

      const startTime = Date.now();
      for (let i = 0; i < 50; i++) {
        await fetch(`${serverUrl}`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ query: mutation })
        });
      }
      const duration = Date.now() - startTime;

      console.log(`Rapid sequential mutations (50) completed in ${duration}ms`);
      expect(duration).toBeLessThan(10000);
    });
  });

  describe('Memory Usage', () => {
    test('should not leak memory on repeated queries', async () => {
      const mockUsers = Array(100).fill(null).map((_, i) => ({
        id: `${i}`,
        email: `user${i}@test.com`,
        name: `User ${i}`,
        role: 'user'
      }));
      userService.getAllUsers.mockResolvedValue(mockUsers);

      const query = `
        query {
          users {
            id
            email
            name
          }
        }
      `;

      const initialMemory = process.memoryUsage().heapUsed;

      for (let i = 0; i < 100; i++) {
        await fetch(`${serverUrl}`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ query })
        });
      }

      const finalMemory = process.memoryUsage().heapUsed;
      const memoryIncrease = finalMemory - initialMemory;
      const memoryIncreaseMB = memoryIncrease / 1024 / 1024;

      console.log(`Memory increase after 100 queries: ${memoryIncreaseMB.toFixed(2)} MB`);
      expect(memoryIncreaseMB).toBeLessThan(50);
    });
  });

  describe('Response Time', () => {
    test('should maintain consistent response times', async () => {
      const mockUsers = Array(10).fill(null).map((_, i) => ({
        id: `${i}`,
        email: `user${i}@test.com`,
        name: `User ${i}`,
        role: 'user'
      }));
      userService.getAllUsers.mockResolvedValue(mockUsers);

      const query = `
        query {
          users {
            id
            email
          }
        }
      `;

      const responseTimes = [];
      for (let i = 0; i < 20; i++) {
        const startTime = Date.now();
        await fetch(`${serverUrl}`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ query })
        });
        responseTimes.push(Date.now() - startTime);
      }

      const avgTime = responseTimes.reduce((a, b) => a + b, 0) / responseTimes.length;
      const maxTime = Math.max(...responseTimes);
      const minTime = Math.min(...responseTimes);

      console.log(`Response times - Avg: ${avgTime.toFixed(2)}ms, Min: ${minTime}ms, Max: ${maxTime}ms`);
      
      expect(avgTime).toBeLessThan(100);
      expect(maxTime).toBeLessThan(500);
    });
  });
});
