const request = require('supertest');
const express = require('express');
const mongoose = require('mongoose');

const mockNotificationService = {
  getUserNotifications: jest.fn(),
  getUnreadCount: jest.fn(),
  getNotificationById: jest.fn(),
  markAsRead: jest.fn(),
  markAllAsRead: jest.fn(),
  deleteNotification: jest.fn(),
  createNotification: jest.fn()
};

jest.mock('../../services/notificationService', () => mockNotificationService);

const notificationsRoutes = require('../../routes/v1/notifications');
const {
  generateToken,
  HTTP_STATUS
} = require('../helpers/testFixtures');

const app = express();
app.use(express.json());
app.use('/api/v1/notifications', notificationsRoutes);

describe('Notifications API Unit Tests', () => {
  const mockUser = {
    id: 1,
    email: 'test@example.com',
    name: 'Test User',
    role: 'user'
  };

  let testToken;

  const mockNotification = {
    _id: new mongoose.Types.ObjectId(),
    userId: mockUser.id,
    title: 'Test Notification',
    message: 'This is a test notification',
    type: 'info',
    status: 'unread',
    channels: ['in_app'],
    createdAt: new Date(),
    updatedAt: new Date()
  };

  beforeEach(() => {
    jest.clearAllMocks();
    testToken = generateToken(mockUser);
    
    mockNotificationService.getUserNotifications.mockResolvedValue([mockNotification]);
    mockNotificationService.getUnreadCount.mockResolvedValue(5);
    mockNotificationService.getNotificationById.mockResolvedValue(mockNotification);
    mockNotificationService.markAsRead.mockResolvedValue({ ...mockNotification, status: 'read' });
    mockNotificationService.markAllAsRead.mockResolvedValue({ modifiedCount: 5 });
    mockNotificationService.deleteNotification.mockResolvedValue(true);
    mockNotificationService.createNotification.mockResolvedValue(mockNotification);
  });

  describe('GET /api/v1/notifications', () => {
    it('should get user notifications successfully', async () => {
      const response = await request(app)
        .get('/api/v1/notifications')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body).toHaveProperty('success');
    });

    it('should support pagination parameters', async () => {
      const response = await request(app)
        .get('/api/v1/notifications?page=1&limit=10')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(mockNotificationService.getUserNotifications).toHaveBeenCalledWith(
        mockUser.id,
        expect.objectContaining({ page: 1, limit: 10 })
      );
    });

    it('should support status filtering', async () => {
      const response = await request(app)
        .get('/api/v1/notifications?status=unread')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(mockNotificationService.getUserNotifications).toHaveBeenCalledWith(
        mockUser.id,
        expect.objectContaining({ status: 'unread' })
      );
    });

    it('should fail without authentication', async () => {
      const response = await request(app).get('/api/v1/notifications');

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });
  });

  describe('GET /api/v1/notifications/unread/count', () => {
    it('should get unread notification count', async () => {
      const response = await request(app)
        .get('/api/v1/notifications/unread/count')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body).toHaveProperty('success');
      expect(response.body.data).toHaveProperty('count');
    });

    it('should fail without authentication', async () => {
      const response = await request(app).get('/api/v1/notifications/unread/count');

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });
  });

  describe('GET /api/v1/notifications/:id', () => {
    it('should get notification by id successfully', async () => {
      const notificationId = mockNotification._id.toString();

      const response = await request(app)
        .get(`/api/v1/notifications/${notificationId}`)
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body).toHaveProperty('success');
    });

    it('should return 404 for non-existent notification', async () => {
      mockNotificationService.getNotificationById.mockResolvedValue(null);
      const fakeId = new mongoose.Types.ObjectId().toString();

      const response = await request(app)
        .get(`/api/v1/notifications/${fakeId}`)
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.NOT_FOUND);
    });

    it('should fail without authentication', async () => {
      const notificationId = mockNotification._id.toString();

      const response = await request(app)
        .get(`/api/v1/notifications/${notificationId}`);

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });
  });

  describe('PUT /api/v1/notifications/:id/read', () => {
    it('should mark notification as read successfully', async () => {
      const notificationId = mockNotification._id.toString();

      const response = await request(app)
        .put(`/api/v1/notifications/${notificationId}/read`)
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body).toHaveProperty('success');
    });

    it('should return 404 for non-existent notification', async () => {
      mockNotificationService.markAsRead.mockRejectedValue(new Error('Not found'));
      const fakeId = new mongoose.Types.ObjectId().toString();

      const response = await request(app)
        .put(`/api/v1/notifications/${fakeId}/read`)
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.INTERNAL_SERVER_ERROR);
    });

    it('should fail without authentication', async () => {
      const notificationId = mockNotification._id.toString();

      const response = await request(app)
        .put(`/api/v1/notifications/${notificationId}/read`);

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });
  });

  describe('PUT /api/v1/notifications/read-all', () => {
    it('should mark all notifications as read successfully', async () => {
      const response = await request(app)
        .put('/api/v1/notifications/read-all')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body).toHaveProperty('success');
    });

    it('should fail without authentication', async () => {
      const response = await request(app)
        .put('/api/v1/notifications/read-all');

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });
  });

  describe('DELETE /api/v1/notifications/:id', () => {
    it('should delete notification successfully', async () => {
      const notificationId = mockNotification._id.toString();

      const response = await request(app)
        .delete(`/api/v1/notifications/${notificationId}`)
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body).toHaveProperty('success');
    });

    it('should return 404 for non-existent notification', async () => {
      mockNotificationService.deleteNotification.mockRejectedValue(new Error('Not found'));
      const fakeId = new mongoose.Types.ObjectId().toString();

      const response = await request(app)
        .delete(`/api/v1/notifications/${fakeId}`)
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.INTERNAL_SERVER_ERROR);
    });

    it('should fail without authentication', async () => {
      const notificationId = mockNotification._id.toString();

      const response = await request(app)
        .delete(`/api/v1/notifications/${notificationId}`);

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });
  });

  describe('POST /api/v1/notifications/send', () => {
    it('should send notification successfully', async () => {
      const response = await request(app)
        .post('/api/v1/notifications/send')
        .set('Authorization', `Bearer ${testToken}`)
        .send({
          userId: mockUser.id,
          title: 'New Notification',
          message: 'This is a new notification',
          type: 'info',
          channels: ['in_app']
        });

      expect(response.status).toBe(HTTP_STATUS.CREATED);
      expect(response.body).toHaveProperty('success');
    });

    it('should fail with missing required fields', async () => {
      const response = await request(app)
        .post('/api/v1/notifications/send')
        .set('Authorization', `Bearer ${testToken}`)
        .send({
          title: 'Missing Message'
        });

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
    });

    it('should fail without authentication', async () => {
      const response = await request(app)
        .post('/api/v1/notifications/send')
        .send({
          userId: mockUser.id,
          title: 'New Notification',
          message: 'This is a new notification'
        });

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });
  });
});
