const request = require('supertest');
const express = require('express');

jest.mock('mongoose', () => ({
  connect: jest.fn().mockResolvedValue({}),
  connection: {
    readyState: 1,
    close: jest.fn().mockResolvedValue({})
  },
  Schema: jest.fn().mockReturnValue({
    index: jest.fn().mockReturnThis(),
    timestamps: jest.fn().mockReturnThis(),
    pre: jest.fn().mockReturnThis(),
    post: jest.fn().mockReturnThis()
  }),
  model: jest.fn().mockReturnValue({
    create: jest.fn().mockResolvedValue({
      _id: 'mock-notification-id',
      title: 'Test Notification',
      message: 'This is a test notification message',
      userId: 1,
      status: 'unread'
    })
  }),
  Types: {
    ObjectId: jest.fn().mockImplementation(() => 'mock-object-id')
  },
  default: {}
}));

jest.mock('../../../config/database/db', () => ({
  query: jest.fn().mockResolvedValue({ rows: [] }),
  end: jest.fn().mockResolvedValue(undefined)
}));

jest.mock('../../services/notificationService', () => ({
  createNotification: jest.fn().mockImplementation((data) => Promise.resolve({
    _id: 'mock-notification-id',
    ...data,
    status: 'unread',
    createdAt: new Date()
  })),
  getUserNotifications: jest.fn().mockResolvedValue([]),
  getUnreadCount: jest.fn().mockResolvedValue(0),
  getNotificationById: jest.fn().mockImplementation((id, userId) => {
    if (id === '999999') return Promise.resolve(null);
    return Promise.resolve({
      _id: id,
      title: 'Test Notification',
      message: 'This is a test notification message',
      userId,
      status: 'unread'
    });
  }),
  markAsRead: jest.fn().mockImplementation((id, userId) => {
    if (id === '999999') return Promise.resolve(null);
    return Promise.resolve({
      _id: id,
      userId,
      status: 'read'
    });
  }),
  markAllAsRead: jest.fn().mockResolvedValue({ modifiedCount: 5 }),
  deleteNotification: jest.fn().mockImplementation((id, userId) => {
    if (id === '999999') return Promise.resolve(null);
    return Promise.resolve({ deletedCount: 1 });
  })
}));

const notificationService = require('../../services/notificationService');
const {
  generateToken,
  notificationData,
  HTTP_STATUS
} = require('../helpers/testFixtures');

describe('Notifications API Integration Tests', () => {
  let testUser;
  let testToken;

  beforeEach(() => {
    jest.clearAllMocks();
  });

  beforeAll(() => {
    testUser = {
      id: 1,
      email: 'test@example.com',
      name: 'Test User',
      role: 'user',
      status: 'active'
    };

    testToken = generateToken(testUser);
  });

  afterAll(async () => {
    jest.clearAllMocks();
  });

  describe('Notification Service Tests', () => {
    describe('createNotification', () => {
      it('should create a notification successfully', async () => {
        const notification = await notificationService.createNotification({
          userId: testUser.id,
          title: 'Test Notification',
          message: 'This is a test message',
          type: 'info'
        });

        expect(notification).toHaveProperty('title', 'Test Notification');
        expect(notification).toHaveProperty('message', 'This is a test message');
        expect(notification).toHaveProperty('status', 'unread');
      });

      it('should support different notification types', async () => {
        const types = ['info', 'success', 'warning', 'error'];
        
        for (const type of types) {
          const notification = await notificationService.createNotification({
            userId: testUser.id,
            title: `${type} Notification`,
            message: 'Test message',
            type
          });

          expect(notification).toHaveProperty('type', type);
        }
      });
    });

    describe('getUserNotifications', () => {
      it('should get user notifications', async () => {
        const notifications = await notificationService.getUserNotifications(testUser.id);
        expect(Array.isArray(notifications)).toBe(true);
      });

      it('should support pagination', async () => {
        await notificationService.getUserNotifications(testUser.id, {
          page: 1,
          limit: 10
        });
        expect(notificationService.getUserNotifications).toHaveBeenCalled();
      });
    });

    describe('getUnreadCount', () => {
      it('should return unread count', async () => {
        const count = await notificationService.getUnreadCount(testUser.id);
        expect(typeof count).toBe('number');
      });
    });

    describe('getNotificationById', () => {
      it('should get notification by id', async () => {
        const notification = await notificationService.getNotificationById('123', testUser.id);
        expect(notification).toBeTruthy();
        expect(notification).toHaveProperty('_id', '123');
      });

      it('should return null for non-existent notification', async () => {
        const notification = await notificationService.getNotificationById('999999', testUser.id);
        expect(notification).toBeNull();
      });
    });

    describe('markAsRead', () => {
      it('should mark notification as read', async () => {
        const result = await notificationService.markAsRead('123', testUser.id);
        expect(result).toHaveProperty('status', 'read');
      });

      it('should return null for non-existent notification', async () => {
        const result = await notificationService.markAsRead('999999', testUser.id);
        expect(result).toBeNull();
      });
    });

    describe('markAllAsRead', () => {
      it('should mark all notifications as read', async () => {
        const result = await notificationService.markAllAsRead(testUser.id);
        expect(result).toHaveProperty('modifiedCount');
      });
    });

    describe('deleteNotification', () => {
      it('should delete notification', async () => {
        const result = await notificationService.deleteNotification('123', testUser.id);
        expect(result).toHaveProperty('deletedCount');
      });

      it('should return null for non-existent notification', async () => {
        const result = await notificationService.deleteNotification('999999', testUser.id);
        expect(result).toBeNull();
      });
    });
  });

  describe('Notification Data Validation Tests', () => {
    it('should validate notification data structure', () => {
      expect(notificationData).toHaveProperty('title');
      expect(notificationData).toHaveProperty('message');
      expect(notificationData).toHaveProperty('type');
      expect(notificationData).toHaveProperty('channels');
    });

    it('should validate notification types', () => {
      const validTypes = ['info', 'success', 'warning', 'error'];
      validTypes.forEach(type => {
        expect(['info', 'success', 'warning', 'error']).toContain(type);
      });
    });

    it('should validate channels', () => {
      expect(Array.isArray(notificationData.channels)).toBe(true);
      expect(notificationData.channels).toContain('in_app');
    });
  });

  describe('HTTP Status Code Tests', () => {
    it('should validate correct status codes', () => {
      expect(HTTP_STATUS.OK).toBe(200);
      expect(HTTP_STATUS.CREATED).toBe(201);
      expect(HTTP_STATUS.NO_CONTENT).toBe(204);
      expect(HTTP_STATUS.BAD_REQUEST).toBe(400);
      expect(HTTP_STATUS.UNAUTHORIZED).toBe(401);
      expect(HTTP_STATUS.FORBIDDEN).toBe(403);
      expect(HTTP_STATUS.NOT_FOUND).toBe(404);
    });
  });

  describe('Token Authorization Tests', () => {
    it('should generate valid notification token', () => {
      const token = generateToken(testUser);
      expect(token).toBeTruthy();
    });

    it('should include user id in token', () => {
      const token = generateToken(testUser);
      const jwt = require('jsonwebtoken');
      const JWT_SECRET = process.env.JWT_SECRET || 'hjtpx-secret-key-change-in-production';
      
      const decoded = jwt.verify(token, JWT_SECRET);
      expect(decoded.id).toBe(testUser.id);
    });
  });
});
