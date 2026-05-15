const {
  createLoaders,
  createUserLoader,
  createUsersLoader,
  createNotificationLoader,
  createUserNotificationsLoader,
  createUnreadCountLoader
} = require('../../src/backend/graphql/loaders');

jest.mock('../../src/backend/services/userService');
jest.mock('../../src/backend/models/Notification');

const userService = require('../../src/backend/services/userService');
const Notification = require('../../src/backend/models/Notification');

describe('GraphQL DataLoaders', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('createUserLoader', () => {
    it('should batch load users', async () => {
      const mockUsers = [
        { id: '1', email: 'user1@example.com', name: 'User 1' },
        { id: '2', email: 'user2@example.com', name: 'User 2' }
      ];
      
      userService.getUserById
        .mockResolvedValueOnce(mockUsers[0])
        .mockResolvedValueOnce(mockUsers[1]);

      const loader = createUserLoader();
      const results = await Promise.all([
        loader.load('1'),
        loader.load('2')
      ]);

      expect(results).toHaveLength(2);
      expect(results[0]).toEqual(mockUsers[0]);
      expect(results[1]).toEqual(mockUsers[1]);
    });

    it('should deduplicate requests for same user', async () => {
      const mockUser = { id: '1', email: 'user1@example.com', name: 'User 1' };
      userService.getUserById.mockResolvedValue(mockUser);

      const loader = createUserLoader();
      const results = await Promise.all([
        loader.load('1'),
        loader.load('1')
      ]);

      expect(userService.getUserById).toHaveBeenCalledTimes(1);
      expect(results).toHaveLength(2);
      expect(results[0]).toEqual(mockUser);
      expect(results[1]).toEqual(mockUser);
    });

    it('should return null for non-existent user', async () => {
      userService.getUserById.mockResolvedValue(null);

      const loader = createUserLoader();
      const result = await loader.load('999');

      expect(result).toBeNull();
    });
  });

  describe('createNotificationLoader', () => {
    it('should batch load notifications', async () => {
      const mockNotifications = [
        { _id: 'notif-1', title: 'Notification 1', toObject: function() { return this; } },
        { _id: 'notif-2', title: 'Notification 2', toObject: function() { return this; } }
      ];

      Notification.find.mockReturnValue({
        sort: jest.fn().mockResolvedValue(mockNotifications)
      });

      const loader = createNotificationLoader();
      const results = await Promise.all([
        loader.load('notif-1'),
        loader.load('notif-2')
      ]);

      expect(results).toHaveLength(2);
      expect(results[0]).toHaveProperty('id', 'notif-1');
      expect(results[1]).toHaveProperty('id', 'notif-2');
    });

    it('should return null for non-existent notification', async () => {
      Notification.find.mockReturnValue({
        sort: jest.fn().mockResolvedValue([])
      });

      const loader = createNotificationLoader();
      const result = await loader.load('notif-999');

      expect(result).toBeNull();
    });
  });

  describe('createUserNotificationsLoader', () => {
    it('should batch load notifications for multiple users', async () => {
      const mockNotifications = [
        { _id: 'notif-1', userId: '1', title: 'Notification 1', toObject: function() { return this; } },
        { _id: 'notif-2', userId: '2', title: 'Notification 2', toObject: function() { return this; } }
      ];

      Notification.find.mockReturnValue({
        sort: jest.fn().mockReturnValue({
          limit: jest.fn().mockResolvedValue(mockNotifications)
        })
      });

      const loader = createUserNotificationsLoader();
      const results = await Promise.all([
        loader.load('1'),
        loader.load('2')
      ]);

      expect(results).toHaveLength(2);
      expect(results[0]).toHaveLength(1);
      expect(results[1]).toHaveLength(1);
    });
  });

  describe('createUnreadCountLoader', () => {
    it('should batch load unread counts for multiple users', async () => {
      const mockCounts = [
        { _id: '1', count: 5 },
        { _id: '2', count: 3 }
      ];

      Notification.aggregate.mockResolvedValue(mockCounts);

      const loader = createUnreadCountLoader();
      const results = await Promise.all([
        loader.load('1'),
        loader.load('2')
      ]);

      expect(results).toHaveLength(2);
      expect(results[0]).toBe(5);
      expect(results[1]).toBe(3);
    });

    it('should return 0 for users with no unread notifications', async () => {
      Notification.aggregate.mockResolvedValue([]);

      const loader = createUnreadCountLoader();
      const result = await loader.load('1');

      expect(result).toBe(0);
    });
  });

  describe('createLoaders', () => {
    it('should create all loaders', () => {
      const loaders = createLoaders();

      expect(loaders).toHaveProperty('user');
      expect(loaders).toHaveProperty('users');
      expect(loaders).toHaveProperty('notification');
      expect(loaders).toHaveProperty('userNotifications');
      expect(loaders).toHaveProperty('unreadCount');
    });

    it('should return independent loader instances', () => {
      const loaders = createLoaders();

      expect(loaders.user).not.toBe(loaders.users);
      expect(loaders.notification).not.toBe(loaders.userNotifications);
    });
  });
});
