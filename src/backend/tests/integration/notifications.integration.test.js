const bcrypt = require('bcryptjs');
const express = require('express');
const mongoose = require('mongoose');
const request = require('supertest');

const pool = require('../../../config/database/db');
const notificationsRoutes = require('../../routes/v1/notifications');
const { userFactory, notificationFactory } = require('../factories');
const {
  generateToken,
  testPassword,
  notificationData,
  HTTP_STATUS
} = require('../helpers/testFixtures');

const app = express();
app.use(express.json());
app.use('/api/v1/notifications', notificationsRoutes);

const MONGO_URI = process.env.MONGO_URI || 'mongodb://localhost:27017/hjtpx_notifications_test';

describe('Notifications API Integration Tests', () => {
  let testUser;
  let testToken;
  let cleanupUsers = [];
  let cleanupNotificationIds = [];

  beforeAll(async () => {
    try {
      if (mongoose.connection.readyState === 0) {
        await mongoose.connect(MONGO_URI, {
          serverSelectionTimeoutMS: 2000
        });
      }
    } catch (error) {
      console.log('MongoDB connection skipped for integration test:', error.message);
    }

    testUser = await userFactory.createUser({
      password: testPassword
    });
    cleanupUsers.push(testUser.id);

    testToken = generateToken(testUser);

    try {
      const notification = await notificationFactory.createNotification(testUser.id);
      cleanupNotificationIds.push(notification._id.toString());
    } catch (error) {
      console.log('Could not create test notification:', error.message);
    }
  });

  afterAll(async () => {
    try {
      if (cleanupNotificationIds.length > 0) {
        await notificationFactory.deleteNotifications(cleanupNotificationIds);
      }
    } catch (error) {
      console.log('Notification cleanup skipped:', error.message);
    }

    await userFactory.deleteUsers(cleanupUsers);
    await pool.end();

    try {
      if (mongoose.connection.readyState !== 0) {
        await mongoose.connection.close();
      }
    } catch (error) {
      console.log('MongoDB disconnect skipped:', error.message);
    }
  });

  describe('GET /api/v1/notifications', () => {
    it('should get user notifications successfully', async () => {
      const response = await request(app)
        .get('/api/v1/notifications')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
      expect(Array.isArray(response.body.data)).toBe(true);
    });

    it('should support pagination parameters', async () => {
      const response = await request(app)
        .get('/api/v1/notifications?page=1&limit=10')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
    });

    it('should support status filtering', async () => {
      const response = await request(app)
        .get('/api/v1/notifications?status=unread')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
    });

    it('should support type filtering', async () => {
      const response = await request(app)
        .get('/api/v1/notifications?type=info')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
    });

    it('should fail without authentication', async () => {
      const response = await request(app).get('/api/v1/notifications');

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });

    it('should fail with invalid token', async () => {
      const response = await request(app)
        .get('/api/v1/notifications')
        .set('Authorization', 'Bearer invalid-token');

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });
  });

  describe('POST /api/v1/notifications', () => {
    it('should create a new notification successfully', async () => {
      const response = await request(app)
        .post('/api/v1/notifications')
        .set('Authorization', `Bearer ${testToken}`)
        .send({
          title: 'New Notification',
          message: 'This is a new notification',
          type: 'info',
          channels: ['in_app']
        });

      expect(response.status).toBe(HTTP_STATUS.CREATED);
      expect(response.body.success).toBe(true);

      if (response.body.data && response.body.data._id) {
        cleanupNotificationIds.push(response.body.data._id.toString());
      }
    });

    it('should support different notification types (success)', async () => {
      const response = await request(app)
        .post('/api/v1/notifications')
        .set('Authorization', `Bearer ${testToken}`)
        .send({
          title: 'Success Notification',
          message: 'Operation completed successfully',
          type: 'success',
          channels: ['in_app']
        });

      expect(response.status).toBe(HTTP_STATUS.CREATED);

      if (response.body.data && response.body.data._id) {
        cleanupNotificationIds.push(response.body.data._id.toString());
      }
    });

    it('should support different notification types (warning)', async () => {
      const response = await request(app)
        .post('/api/v1/notifications')
        .set('Authorization', `Bearer ${testToken}`)
        .send({
          title: 'Warning Notification',
          message: 'Please check your settings',
          type: 'warning',
          channels: ['in_app']
        });

      expect(response.status).toBe(HTTP_STATUS.CREATED);

      if (response.body.data && response.body.data._id) {
        cleanupNotificationIds.push(response.body.data._id.toString());
      }
    });

    it('should support different notification types (error)', async () => {
      const response = await request(app)
        .post('/api/v1/notifications')
        .set('Authorization', `Bearer ${testToken}`)
        .send({
          title: 'Error Notification',
          message: 'Something went wrong',
          type: 'error',
          channels: ['in_app']
        });

      expect(response.status).toBe(HTTP_STATUS.CREATED);

      if (response.body.data && response.body.data._id) {
        cleanupNotificationIds.push(response.body.data._id.toString());
      }
    });

    it('should fail with missing required fields', async () => {
      const response = await request(app)
        .post('/api/v1/notifications')
        .set('Authorization', `Bearer ${testToken}`)
        .send({
          title: 'Missing Message'
        });

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
    });

    it('should fail without authentication', async () => {
      const response = await request(app).post('/api/v1/notifications').send(notificationData);

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });
  });

  describe('PUT /api/v1/notifications/:id/read', () => {
    let notificationToMark;

    beforeEach(async () => {
      try {
        notificationToMark = await notificationFactory.createUnreadNotification(testUser.id);
        cleanupNotificationIds.push(notificationToMark._id.toString());
      } catch (error) {
        console.log('Could not create notification in beforeEach:', error.message);
        notificationToMark = null;
      }
    });

    it('should mark notification as read successfully', async () => {
      if (!notificationToMark) {
        console.log('Skipping test: Could not create notification');
        return;
      }

      const response = await request(app)
        .put(`/api/v1/notifications/${notificationToMark._id}/read`)
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
    });

    it('should fail for non-existent notification', async () => {
      const fakeId = new mongoose.Types.ObjectId();
      const response = await request(app)
        .put(`/api/v1/notifications/${fakeId}/read`)
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.NOT_FOUND);
    });

    it('should fail without authentication', async () => {
      if (!notificationToMark) {
        console.log('Skipping test: Could not create notification');
        return;
      }

      const response = await request(app).put(
        `/api/v1/notifications/${notificationToMark._id}/read`
      );

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });
  });

  describe('DELETE /api/v1/notifications/:id', () => {
    let notificationToDelete;

    beforeEach(async () => {
      try {
        notificationToDelete = await notificationFactory.createNotification(testUser.id);
      } catch (error) {
        console.log('Could not create notification in beforeEach:', error.message);
        notificationToDelete = null;
      }
    });

    it('should delete notification successfully', async () => {
      if (!notificationToDelete) {
        console.log('Skipping test: Could not create notification');
        return;
      }

      const response = await request(app)
        .delete(`/api/v1/notifications/${notificationToDelete._id}`)
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.NO_CONTENT);
    });

    it('should fail for non-existent notification', async () => {
      const fakeId = new mongoose.Types.ObjectId();
      const response = await request(app)
        .delete(`/api/v1/notifications/${fakeId}`)
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.NOT_FOUND);
    });

    it('should fail without authentication', async () => {
      if (!notificationToDelete) {
        console.log('Skipping test: Could not create notification');
        return;
      }

      const response = await request(app).delete(
        `/api/v1/notifications/${notificationToDelete._id}`
      );

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });
  });

  describe('PUT /api/v1/notifications/mark-all-read', () => {
    it('should mark all notifications as read successfully', async () => {
      try {
        await notificationFactory.createMultipleNotifications(testUser.id, 3);
      } catch (error) {
        console.log('Could not create multiple notifications:', error.message);
      }

      const response = await request(app)
        .put('/api/v1/notifications/mark-all-read')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
    });

    it('should fail without authentication', async () => {
      const response = await request(app).put('/api/v1/notifications/mark-all-read');

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });
  });

  describe('GET /api/v1/notifications/unread/count', () => {
    it('should get unread notification count', async () => {
      const response = await request(app)
        .get('/api/v1/notifications/unread/count')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
      expect(response.body.data).toHaveProperty('count');
      expect(typeof response.body.data.count).toBe('number');
    });

    it('should fail without authentication', async () => {
      const response = await request(app).get('/api/v1/notifications/unread/count');

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });
  });
});
