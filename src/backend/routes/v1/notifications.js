const express = require('express');
const router = express.Router();
const notificationService = require('../../services/notificationService');
const { authenticateToken } = require('../../middleware/auth');

router.get('/', authenticateToken, async (req, res) => {
  try {
    const { page = 1, limit = 20, status, type } = req.query;
    const userId = req.user.id;

    const notifications = await notificationService.getUserNotifications(userId, {
      page: parseInt(page),
      limit: parseInt(limit),
      status,
      type
    });

    res.success(notifications);
  } catch (error) {
    res.error(error.message, 500, 'NOTIFICATION_ERROR');
  }
});

router.get('/unread/count', authenticateToken, async (req, res) => {
  try {
    const userId = req.user.id;
    const count = await notificationService.getUnreadCount(userId);
    res.success({ count });
  } catch (error) {
    res.error(error.message, 500, 'NOTIFICATION_ERROR');
  }
});

router.get('/:id', authenticateToken, async (req, res) => {
  try {
    const { id } = req.params;
    const userId = req.user.id;

    const notification = await notificationService.getNotificationById(id, userId);
    if (!notification) {
      return res.notFound('Notification not found');
    }

    res.success(notification);
  } catch (error) {
    res.error(error.message, 500, 'NOTIFICATION_ERROR');
  }
});

router.put('/:id/read', authenticateToken, async (req, res) => {
  try {
    const { id } = req.params;
    const userId = req.user.id;

    const result = await notificationService.markAsRead(id, userId);
    if (!result) {
      return res.notFound('Notification not found');
    }
    res.success(result);
  } catch (error) {
    res.error(error.message, 500, 'NOTIFICATION_ERROR');
  }
});

router.put('/mark-all-read', authenticateToken, async (req, res) => {
  try {
    const userId = req.user.id;
    const result = await notificationService.markAllAsRead(userId);
    res.success(result);
  } catch (error) {
    res.error(error.message, 500, 'NOTIFICATION_ERROR');
  }
});

router.put('/read-all', authenticateToken, async (req, res) => {
  try {
    const userId = req.user.id;
    const result = await notificationService.markAllAsRead(userId);
    res.success(result);
  } catch (error) {
    res.error(error.message, 500, 'NOTIFICATION_ERROR');
  }
});

router.delete('/:id', authenticateToken, async (req, res) => {
  try {
    const { id } = req.params;
    const userId = req.user.id;

    const result = await notificationService.deleteNotification(id, userId);
    if (!result) {
      return res.notFound('Notification not found');
    }
    res.noContent();
  } catch (error) {
    res.error(error.message, 500, 'NOTIFICATION_ERROR');
  }
});

router.post('/', authenticateToken, async (req, res) => {
  try {
    const { title, message, type, channels } = req.body;
    const userId = req.user.id;

    if (!title || !message) {
      return res.badRequest('Missing required fields: title and message');
    }

    const notification = await notificationService.createNotification({
      userId,
      title,
      message,
      type: type || 'info',
      channels: channels || ['in_app']
    });

    res.created(notification);
  } catch (error) {
    res.error(error.message, 500, 'NOTIFICATION_ERROR');
  }
});

router.post('/send', authenticateToken, async (req, res) => {
  try {
    const { userId, title, message, type, channels } = req.body;

    if (!userId || !title || !message) {
      return res.badRequest('Missing required fields');
    }

    const notification = await notificationService.createNotification({
      userId,
      title,
      message,
      type: type || 'system',
      channels: channels || ['in_app']
    });

    res.created(notification);
  } catch (error) {
    res.error(error.message, 500, 'NOTIFICATION_ERROR');
  }
});

module.exports = router;
