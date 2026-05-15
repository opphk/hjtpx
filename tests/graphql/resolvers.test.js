const { GraphQLScalarType, Kind } = require('graphql');
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

describe('GraphQL Resolvers', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Query Resolvers', () => {
    describe('users', () => {
      test('should return all users', async () => {
        const mockUsers = [
          { id: '1', email: 'user1@test.com', name: 'User 1', role: 'user' },
          { id: '2', email: 'user2@test.com', name: 'User 2', role: 'admin' }
        ];
        userService.getAllUsers.mockResolvedValue(mockUsers);

        const result = await resolvers.Query.users();

        expect(result).toEqual(mockUsers);
        expect(userService.getAllUsers).toHaveBeenCalledTimes(1);
      });

      test('should handle errors', async () => {
        userService.getAllUsers.mockRejectedValue(new Error('Database error'));

        await expect(resolvers.Query.users()).rejects.toThrow('Database error');
      });
    });

    describe('user', () => {
      test('should return user by id', async () => {
        const mockUser = { id: '1', email: 'user@test.com', name: 'Test User' };
        userService.getUserById.mockResolvedValue(mockUser);

        const result = await resolvers.Query.user(null, { id: '1' });

        expect(result).toEqual(mockUser);
        expect(userService.getUserById).toHaveBeenCalledWith('1');
      });

      test('should return null for non-existent user', async () => {
        userService.getUserById.mockResolvedValue(null);

        const result = await resolvers.Query.user(null, { id: '999' });

        expect(result).toBeNull();
      });
    });

    describe('me', () => {
      test('should return current user', async () => {
        const mockUser = { id: '1', email: 'user@test.com', name: 'Test User' };
        userService.getUserById.mockResolvedValue(mockUser);
        const context = { user: { id: '1', email: 'user@test.com', role: 'user' } };

        const result = await resolvers.Query.me(null, null, context);

        expect(result).toEqual(mockUser);
        expect(userService.getUserById).toHaveBeenCalledWith('1');
      });

      test('should throw error if not authenticated', async () => {
        const context = { user: null };

        await expect(resolvers.Query.me(null, null, context))
          .rejects.toThrow('Not authenticated');
      });
    });

    describe('notifications', () => {
      test('should return user notifications', async () => {
        const mockNotifications = {
          notifications: [
            {
              _id: { toString: () => '1' },
              toObject: () => ({ id: '1', title: 'Test', type: 'info' })
            }
          ],
          pagination: { page: 1, limit: 20, total: 1, pages: 1 }
        };
        Notification.getUserNotifications.mockResolvedValue(mockNotifications);
        const context = { user: { id: '1' } };

        const result = await resolvers.Query.notifications(null, { page: 1, limit: 20 }, context);

        expect(result.notifications).toHaveLength(1);
        expect(result.pagination).toEqual(mockNotifications.pagination);
        expect(Notification.getUserNotifications).toHaveBeenCalledWith('1', { page: 1, limit: 20 });
      });

      test('should throw error if not authenticated', async () => {
        const context = { user: null };

        await expect(resolvers.Query.notifications(null, {}, context))
          .rejects.toThrow('Not authenticated');
      });
    });

    describe('unreadNotificationsCount', () => {
      test('should return unread count', async () => {
        Notification.getUnreadCount.mockResolvedValue(5);
        const context = { user: { id: '1' } };

        const result = await resolvers.Query.unreadNotificationsCount(null, null, context);

        expect(result).toBe(5);
        expect(Notification.getUnreadCount).toHaveBeenCalledWith('1');
      });

      test('should throw error if not authenticated', async () => {
        const context = { user: null };

        await expect(resolvers.Query.unreadNotificationsCount(null, null, context))
          .rejects.toThrow('Not authenticated');
      });
    });
  });

  describe('Mutation Resolvers', () => {
    describe('createUser', () => {
      test('should create user with admin privileges', async () => {
        const mockUser = { id: '1', email: 'new@test.com', name: 'New User' };
        userService.createUser.mockResolvedValue(mockUser);
        const context = { user: { id: '1', role: 'admin' } };

        const result = await resolvers.Mutation.createUser(
          null,
          { email: 'new@test.com', name: 'New User', password: 'password123' },
          context
        );

        expect(result).toEqual(mockUser);
        expect(userService.createUser).toHaveBeenCalledWith({
          email: 'new@test.com',
          name: 'New User',
          password: 'password123'
        });
      });

      test('should throw error if not authorized', async () => {
        const context = { user: { id: '1', role: 'user' } };

        await expect(resolvers.Mutation.createUser(
          null,
          { email: 'new@test.com', name: 'New User', password: 'password123' },
          context
        )).rejects.toThrow('Not authorized');
      });

      test('should throw error if not authenticated', async () => {
        const context = { user: null };

        await expect(resolvers.Mutation.createUser(
          null,
          { email: 'new@test.com', name: 'New User', password: 'password123' },
          context
        )).rejects.toThrow('Not authorized');
      });
    });

    describe('updateUser', () => {
      test('should update user by admin', async () => {
        const mockUser = { id: '1', email: 'updated@test.com', name: 'Updated User' };
        userService.updateUser.mockResolvedValue(mockUser);
        const context = { user: { id: '1', role: 'admin' } };

        const result = await resolvers.Mutation.updateUser(
          null,
          { id: '2', email: 'updated@test.com', name: 'Updated User' },
          context
        );

        expect(result).toEqual(mockUser);
        expect(userService.updateUser).toHaveBeenCalledWith('2', {
          email: 'updated@test.com',
          name: 'Updated User'
        });
      });

      test('should update own profile', async () => {
        const mockUser = { id: '1', email: 'updated@test.com', name: 'Updated User' };
        userService.updateUser.mockResolvedValue(mockUser);
        const context = { user: { id: '1', role: 'user' } };

        const result = await resolvers.Mutation.updateUser(
          null,
          { id: '1', name: 'Updated User' },
          context
        );

        expect(result).toEqual(mockUser);
      });

      test('should not allow non-admin to change role', async () => {
        const context = { user: { id: '1', role: 'user' } };

        await resolvers.Mutation.updateUser(
          null,
          { id: '1', role: 'admin' },
          context
        );

        expect(userService.updateUser).toHaveBeenCalledWith('1', {});
      });

      test('should throw error if not authenticated', async () => {
        const context = { user: null };

        await expect(resolvers.Mutation.updateUser(
          null,
          { id: '1', name: 'Updated' },
          context
        )).rejects.toThrow('Not authenticated');
      });
    });

    describe('deleteUser', () => {
      test('should delete user by admin', async () => {
        userService.deleteUser.mockResolvedValue(true);
        const context = { user: { id: '1', role: 'admin' } };

        const result = await resolvers.Mutation.deleteUser(null, { id: '2' }, context);

        expect(result).toBe(true);
        expect(userService.deleteUser).toHaveBeenCalledWith('2');
      });

      test('should throw error if not admin', async () => {
        const context = { user: { id: '1', role: 'user' } };

        await expect(resolvers.Mutation.deleteUser(null, { id: '2' }, context))
          .rejects.toThrow('Not authorized');
      });
    });

    describe('createNotification', () => {
      test('should create notification', async () => {
        const mockNotification = {
          _id: { toString: () => '1' },
          toObject: () => ({ id: '1', title: 'Test', message: 'Test message' })
        };
        Notification.createNotification.mockResolvedValue(mockNotification);
        const context = { user: { id: '1' } };

        const result = await resolvers.Mutation.createNotification(
          null,
          {
            type: 'info',
            title: 'Test',
            message: 'Test message',
            priority: 'normal'
          },
          context
        );

        expect(result.id).toBe('1');
        expect(Notification.createNotification).toHaveBeenCalled();
      });

      test('should throw error if not authenticated', async () => {
        const context = { user: null };

        await expect(resolvers.Mutation.createNotification(
          null,
          { type: 'info', title: 'Test', message: 'Test message' },
          context
        )).rejects.toThrow('Not authenticated');
      });
    });

    describe('markNotificationAsRead', () => {
      test('should mark notification as read', async () => {
        const mockNotification = {
          _id: { toString: () => '1' },
          toObject: () => ({ id: '1', status: 'read' })
        };
        Notification.markAsRead.mockResolvedValue({ modifiedCount: 1 });
        Notification.findById.mockResolvedValue(mockNotification);
        const context = { user: { id: '1' } };

        const result = await resolvers.Mutation.markNotificationAsRead(
          null,
          { id: '1' },
          context
        );

        expect(result.id).toBe('1');
        expect(Notification.markAsRead).toHaveBeenCalledWith('1', '1');
      });

      test('should return null if notification not found', async () => {
        Notification.markAsRead.mockResolvedValue({ modifiedCount: 0 });
        const context = { user: { id: '1' } };

        const result = await resolvers.Mutation.markNotificationAsRead(
          null,
          { id: '999' },
          context
        );

        expect(result).toBeNull();
      });
    });

    describe('markAllNotificationsAsRead', () => {
      test('should mark all notifications as read', async () => {
        Notification.markAllAsRead.mockResolvedValue({ modifiedCount: 5 });
        const context = { user: { id: '1' } };

        const result = await resolvers.Mutation.markAllNotificationsAsRead(null, null, context);

        expect(result).toBe(true);
        expect(Notification.markAllAsRead).toHaveBeenCalledWith('1');
      });

      test('should throw error if not authenticated', async () => {
        const context = { user: null };

        await expect(resolvers.Mutation.markAllNotificationsAsRead(null, null, context))
          .rejects.toThrow('Not authenticated');
      });
    });
  });

  describe('Field Resolvers', () => {
    describe('User.notifications', () => {
      test('should return user notifications', async () => {
        const mockNotifications = {
          notifications: [
            {
              _id: { toString: () => '1' },
              toObject: () => ({ id: '1', title: 'Test' })
            }
          ]
        };
        Notification.getUserNotifications.mockResolvedValue(mockNotifications);

        const result = await resolvers.User.notifications({ id: '1' });

        expect(result).toHaveLength(1);
        expect(result[0].id).toBe('1');
        expect(Notification.getUserNotifications).toHaveBeenCalledWith('1', { limit: 10 });
      });
    });

    describe('User.unreadNotificationsCount', () => {
      test('should return unread count', async () => {
        Notification.getUnreadCount.mockResolvedValue(3);

        const result = await resolvers.User.unreadNotificationsCount({ id: '1' });

        expect(result).toBe(3);
        expect(Notification.getUnreadCount).toHaveBeenCalledWith('1');
      });
    });

    describe('Notification.user', () => {
      test('should return notification user', async () => {
        const mockUser = { id: '1', email: 'user@test.com' };
        userService.getUserById.mockResolvedValue(mockUser);

        const result = await resolvers.Notification.user({ userId: '1' });

        expect(result).toEqual(mockUser);
        expect(userService.getUserById).toHaveBeenCalledWith('1');
      });
    });
  });

  describe('JSON Scalar', () => {
    test('should serialize JSON values', () => {
      const jsonScalar = resolvers.JSON;
      const testData = { key: 'value', nested: { foo: 'bar' } };

      expect(jsonScalar.serialize(testData)).toEqual(testData);
    });

    test('should parse JSON values', () => {
      const jsonScalar = resolvers.JSON;
      const testData = { key: 'value' };

      expect(jsonScalar.parseValue(testData)).toEqual(testData);
    });

    test('should parse JSON literals', () => {
      const jsonScalar = resolvers.JSON;
      
      const stringLiteral = { kind: Kind.STRING, value: 'test' };
      expect(jsonScalar.parseLiteral(stringLiteral)).toBe('test');

      const intLiteral = { kind: Kind.INT, value: '42' };
      expect(jsonScalar.parseLiteral(intLiteral)).toBe(42);

      const floatLiteral = { kind: Kind.FLOAT, value: '3.14' };
      expect(jsonScalar.parseLiteral(floatLiteral)).toBe(3.14);

      const boolLiteral = { kind: Kind.BOOLEAN, value: true };
      expect(jsonScalar.parseLiteral(boolLiteral)).toBe(true);

      const nullLiteral = { kind: Kind.NULL, value: null };
      expect(jsonScalar.parseLiteral(nullLiteral)).toBe(null);
    });

    test('should parse object literals', () => {
      const jsonScalar = resolvers.JSON;
      
      const objectLiteral = {
        kind: Kind.OBJECT,
        fields: [
          { name: { value: 'key' }, value: { kind: Kind.STRING, value: 'value' } }
        ]
      };

      const result = jsonScalar.parseLiteral(objectLiteral);
      expect(result).toEqual({ key: 'value' });
    });

    test('should parse list literals', () => {
      const jsonScalar = resolvers.JSON;
      
      const listLiteral = {
        kind: Kind.LIST,
        values: [
          { kind: Kind.STRING, value: 'item1' },
          { kind: Kind.STRING, value: 'item2' }
        ]
      };

      const result = jsonScalar.parseLiteral(listLiteral);
      expect(result).toEqual(['item1', 'item2']);
    });
  });
});
