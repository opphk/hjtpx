const resolvers = require('../../src/backend/graphql/resolvers');
const { subscriptionResolver, pubsub } = require('../../src/backend/graphql/subscriptions');

jest.mock('../../src/backend/services/userService');
jest.mock('../../src/backend/services/authService');
jest.mock('../../src/backend/models/Notification');

const userService = require('../../src/backend/services/userService');
const authService = require('../../src/backend/services/authService');
const Notification = require('../../src/backend/models/Notification');

describe('GraphQL Resolvers', () => {
  let context;
  const mockUser = {
    id: '1',
    email: 'test@example.com',
    name: 'Test User',
    role: 'user',
    created_at: new Date().toISOString()
  };

  const mockAdminUser = {
    id: '2',
    email: 'admin@example.com',
    name: 'Admin User',
    role: 'admin',
    created_at: new Date().toISOString()
  };

  beforeEach(() => {
    jest.clearAllMocks();
    context = {
      user: mockUser,
      loaders: {
        user: { load: jest.fn() },
        users: { load: jest.fn() },
        notification: { load: jest.fn() },
        userNotifications: { load: jest.fn() },
        unreadCount: { load: jest.fn() }
      },
      pubsub
    };
  });

  describe('Query Resolvers', () => {
    describe('users', () => {
      it('should return all users for admin', async () => {
        context.user = mockAdminUser;
        userService.getAllUsers.mockResolvedValue([mockUser, mockAdminUser]);

        const result = await resolvers.Query.users(null, {}, context);

        expect(userService.getAllUsers).toHaveBeenCalledWith({ limit: 100, offset: 0 });
        expect(result).toHaveLength(2);
        expect(result[0]).toHaveProperty('id');
        expect(result[0]).toHaveProperty('email');
      });

      it('should throw error for non-admin users', async () => {
        context.user = mockUser;

        await expect(resolvers.Query.users(null, {}, context))
          .rejects.toThrow('Admin privileges required');
      });

      it('should return users with pagination', async () => {
        context.user = mockAdminUser;
        userService.getAllUsers.mockResolvedValue([mockUser]);

        const result = await resolvers.Query.users(null, { limit: 10, offset: 0 }, context);

        expect(userService.getAllUsers).toHaveBeenCalledWith({ limit: 10, offset: 0 });
        expect(result).toHaveLength(1);
      });
    });

    describe('user', () => {
      it('should return user by id', async () => {
        userService.getUserById.mockResolvedValue(mockUser);

        const result = await resolvers.Query.user(null, { id: '1' }, context);

        expect(userService.getUserById).toHaveBeenCalledWith('1');
        expect(result).toHaveProperty('id', '1');
        expect(result).toHaveProperty('email', mockUser.email);
      });

      it('should return null for non-existent user', async () => {
        userService.getUserById.mockResolvedValue(null);

        const result = await resolvers.Query.user(null, { id: '999' }, context);

        expect(result).toBeNull();
      });
    });

    describe('me', () => {
      it('should return current user', async () => {
        userService.getUserById.mockResolvedValue(mockUser);

        const result = await resolvers.Query.me(null, {}, context);

        expect(userService.getUserById).toHaveBeenCalledWith(mockUser.id);
        expect(result).toHaveProperty('id', mockUser.id);
      });

      it('should throw error when not authenticated', async () => {
        context.user = null;

        await expect(resolvers.Query.me(null, {}, context))
          .rejects.toThrow('Authentication required');
      });
    });

    describe('notifications', () => {
      const mockNotification = {
        _id: 'notif-1',
        userId: '1',
        type: 'info',
        title: 'Test Notification',
        message: 'Test message',
        status: 'unread',
        priority: 'normal',
        createdAt: new Date(),
        toObject: function() { return this; }
      };

      it('should return user notifications', async () => {
        Notification.getUserNotifications.mockResolvedValue({
          notifications: [mockNotification],
          pagination: { page: 1, limit: 20, total: 1, pages: 1 }
        });

        const result = await resolvers.Query.notifications(null, {}, context);

        expect(Notification.getUserNotifications).toHaveBeenCalledWith(mockUser.id, {});
        expect(result.notifications).toHaveLength(1);
        expect(result.pagination).toHaveProperty('total', 1);
      });

      it('should filter notifications by status', async () => {
        Notification.getUserNotifications.mockResolvedValue({
          notifications: [mockNotification],
          pagination: { page: 1, limit: 20, total: 1, pages: 1 }
        });

        const result = await resolvers.Query.notifications(
          null,
          { status: 'unread' },
          context
        );

        expect(Notification.getUserNotifications).toHaveBeenCalledWith(mockUser.id, { status: 'unread' });
      });
    });

    describe('unreadNotificationsCount', () => {
      it('should return unread count', async () => {
        Notification.getUnreadCount.mockResolvedValue(5);

        const result = await resolvers.Query.unreadNotificationsCount(null, {}, context);

        expect(Notification.getUnreadCount).toHaveBeenCalledWith(mockUser.id);
        expect(result).toBe(5);
      });
    });
  });

  describe('Mutation Resolvers', () => {
    describe('createUser', () => {
      it('should create user for admin', async () => {
        context.user = mockAdminUser;
        const newUser = { ...mockUser, id: '3', email: 'new@example.com' };
        userService.createUser.mockResolvedValue(newUser);

        const result = await resolvers.Mutation.createUser(
          null,
          { email: 'new@example.com', name: 'New User', password: 'password123' },
          context
        );

        expect(userService.createUser).toHaveBeenCalled();
        expect(result).toHaveProperty('email', 'new@example.com');
      });

      it('should throw error for non-admin', async () => {
        context.user = mockUser;

        await expect(
          resolvers.Mutation.createUser(
            null,
            { email: 'new@example.com', name: 'New User', password: 'password123' },
            context
          )
        ).rejects.toThrow('Admin privileges required');
      });
    });

    describe('updateUser', () => {
      it('should update own user', async () => {
        const updatedUser = { ...mockUser, name: 'Updated Name' };
        userService.updateUser.mockResolvedValue(updatedUser);

        const result = await resolvers.Mutation.updateUser(
          null,
          { id: mockUser.id, name: 'Updated Name' },
          context
        );

        expect(userService.updateUser).toHaveBeenCalledWith(mockUser.id, { name: 'Updated Name' });
        expect(result).toHaveProperty('name', 'Updated Name');
      });

      it('should allow admin to update any user', async () => {
        context.user = mockAdminUser;
        const updatedUser = { ...mockUser, name: 'Admin Updated' };
        userService.updateUser.mockResolvedValue(updatedUser);

        const result = await resolvers.Mutation.updateUser(
          null,
          { id: mockUser.id, name: 'Admin Updated' },
          context
        );

        expect(result).toHaveProperty('name', 'Admin Updated');
      });
    });

    describe('deleteUser', () => {
      it('should delete user for admin', async () => {
        context.user = mockAdminUser;
        userService.deleteUser.mockResolvedValue(true);

        const result = await resolvers.Mutation.deleteUser(null, { id: '1' }, context);

        expect(userService.deleteUser).toHaveBeenCalledWith('1');
        expect(result).toBe(true);
      });

      it('should throw error for non-admin', async () => {
        context.user = mockUser;

        await expect(
          resolvers.Mutation.deleteUser(null, { id: '1' }, context)
        ).rejects.toThrow('Admin privileges required');
      });
    });

    describe('createNotification', () => {
      const mockNotification = {
        _id: 'notif-2',
        userId: mockUser.id,
        type: 'info',
        title: 'New Notification',
        message: 'Test',
        status: 'unread',
        priority: 'normal',
        channels: ['in_app'],
        createdAt: new Date(),
        toObject: function() { return this; }
      };

      it('should create notification for current user', async () => {
        Notification.createNotification.mockResolvedValue(mockNotification);

        const result = await resolvers.Mutation.createNotification(
          null,
          {
            userId: mockUser.id,
            type: 'info',
            title: 'New Notification',
            message: 'Test'
          },
          context
        );

        expect(Notification.createNotification).toHaveBeenCalled();
        expect(result).toHaveProperty('title', 'New Notification');
      });
    });

    describe('markNotificationAsRead', () => {
      it('should mark notification as read', async () => {
        const updatedNotification = {
          _id: 'notif-1',
          status: 'read',
          readAt: new Date(),
          toObject: function() { return this; }
        };
        Notification.markAsRead.mockResolvedValue({ modifiedCount: 1 });
        Notification.findById.mockResolvedValue(updatedNotification);

        const result = await resolvers.Mutation.markNotificationAsRead(
          null,
          { id: 'notif-1' },
          context
        );

        expect(Notification.markAsRead).toHaveBeenCalledWith('notif-1', mockUser.id);
        expect(result).toHaveProperty('status', 'read');
      });

      it('should return null if notification not found', async () => {
        Notification.markAsRead.mockResolvedValue({ modifiedCount: 0 });

        const result = await resolvers.Mutation.markNotificationAsRead(
          null,
          { id: 'notif-999' },
          context
        );

        expect(result).toBeNull();
      });
    });

    describe('markAllNotificationsAsRead', () => {
      it('should mark all notifications as read', async () => {
        Notification.markAllAsRead.mockResolvedValue({ modifiedCount: 5 });

        const result = await resolvers.Mutation.markAllNotificationsAsRead(null, {}, context);

        expect(Notification.markAllAsRead).toHaveBeenCalledWith(mockUser.id);
        expect(result).toBe(true);
      });
    });

    describe('login', () => {
      it('should login user with valid credentials', async () => {
        authService.authenticate = jest.fn().mockResolvedValue(mockUser);
        authService.generateToken = jest.fn().mockReturnValue('test-token');

        const result = await resolvers.Mutation.login(
          null,
          { email: 'test@example.com', password: 'password123' },
          context
        );

        expect(authService.authenticate).toHaveBeenCalledWith('test@example.com', 'password123');
        expect(result).toHaveProperty('token', 'test-token');
        expect(result.user).toHaveProperty('id', mockUser.id);
      });

      it('should throw error for invalid credentials', async () => {
        authService.authenticate = jest.fn().mockResolvedValue(null);

        await expect(
          resolvers.Mutation.login(
            null,
            { email: 'test@example.com', password: 'wrong' },
            context
          )
        ).rejects.toThrow('Invalid credentials');
      });
    });

    describe('register', () => {
      it('should register new user', async () => {
        userService.getUserByEmail.mockResolvedValue(null);
        userService.createUser.mockResolvedValue(mockUser);
        authService.validatePassword = jest.fn().mockReturnValue(true);
        authService.generateToken = jest.fn().mockReturnValue('test-token');

        const result = await resolvers.Mutation.register(
          null,
          { email: 'new@example.com', name: 'New User', password: 'password123' },
          context
        );

        expect(result).toHaveProperty('token', 'test-token');
        expect(result.user).toHaveProperty('email', mockUser.email);
      });

      it('should throw error if user exists', async () => {
        userService.getUserByEmail.mockResolvedValue(mockUser);

        await expect(
          resolvers.Mutation.register(
            null,
            { email: 'test@example.com', name: 'New User', password: 'password123' },
            context
          )
        ).rejects.toThrow('User already exists');
      });
    });
  });

  describe('Subscription Resolvers', () => {
    describe('notificationCreated', () => {
      it('should have subscribe function defined', () => {
        expect(subscriptionResolver.notificationCreated).toBeDefined();
        expect(subscriptionResolver.notificationCreated.subscribe).toBeDefined();
        expect(typeof subscriptionResolver.notificationCreated.subscribe).toBe('function');
      });
    });

    describe('notificationUpdated', () => {
      it('should have subscribe function defined', () => {
        expect(subscriptionResolver.notificationUpdated).toBeDefined();
        expect(subscriptionResolver.notificationUpdated.subscribe).toBeDefined();
        expect(typeof subscriptionResolver.notificationUpdated.subscribe).toBe('function');
      });
    });

    describe('userUpdated', () => {
      it('should have subscribe function defined', () => {
        expect(subscriptionResolver.userUpdated).toBeDefined();
        expect(subscriptionResolver.userUpdated.subscribe).toBeDefined();
        expect(typeof subscriptionResolver.userUpdated.subscribe).toBe('function');
      });
    });
  });

  describe('Field Resolvers', () => {
    describe('User.notifications', () => {
      it('should resolve user notifications field', async () => {
        const mockNotification = {
          _id: 'notif-1',
          userId: '1',
          type: 'info',
          title: 'Test',
          toObject: function() { return this; }
        };
        Notification.getUserNotifications.mockResolvedValue({
          notifications: [mockNotification],
          pagination: { page: 1, limit: 10, total: 1, pages: 1 }
        });

        const result = await resolvers.User.notifications(mockUser, { limit: 10 }, context);

        expect(Notification.getUserNotifications).toHaveBeenCalledWith(mockUser.id, { status: undefined, limit: 10 });
      });
    });

    describe('User.unreadNotificationsCount', () => {
      it('should resolve unread count', async () => {
        Notification.getUnreadCount.mockResolvedValue(3);

        const result = await resolvers.User.unreadNotificationsCount(mockUser, {}, context);

        expect(Notification.getUnreadCount).toHaveBeenCalledWith(mockUser.id);
        expect(result).toBe(3);
      });
    });
  });
});
