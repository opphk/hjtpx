const express = require('express');
const router = express.Router();
const notificationService = require('../services/notificationService');
const emailService = require('../services/emailService');
const smsService = require('../services/smsService');
const { authenticateToken } = require('../middleware/auth');

/**
 * @swagger
 * /api/v1/notifications:
 *   get:
 *     summary: Get user notifications
 *     tags: [Notifications]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: query
 *         name: page
 *         schema:
 *           type: integer
 *           default: 1
 *       - in: query
 *         name: limit
 *         schema:
 *           type: integer
 *           default: 20
 *       - in: query
 *         name: status
 *         schema:
 *           type: string
 *           enum: [unread, read, archived]
 *     responses:
 *       200:
 *         description: List of notifications
 */
router.get('/', authenticateToken, async (req, res) => {
  try {
    const { page = 1, limit = 20, status } = req.query;
    const userId = req.user.id;

    const notifications = await notificationService.getUserNotifications(userId, {
      page: parseInt(page),
      limit: parseInt(limit),
      status
    });

    res.success(notifications);
  } catch (error) {
    res.error(error.message, 500, 'NOTIFICATION_ERROR');
  }
});

/**
 * @swagger
 * /api/v1/notifications/unread/count:
 *   get:
 *     summary: Get unread notification count
 *     tags: [Notifications]
 *     security:
 *       - bearerAuth: []
 *     responses:
 *       200:
 *         description: Unread count
 */
router.get('/unread/count', authenticateToken, async (req, res) => {
  try {
    const userId = req.user.id;
    const count = await notificationService.getUnreadCount(userId);
    res.success({ count });
  } catch (error) {
    res.error(error.message, 500, 'NOTIFICATION_ERROR');
  }
});

/**
 * @swagger
 * /api/v1/notifications/{id}:
 *   get:
 *     summary: Get notification by ID
 *     tags: [Notifications]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: Notification details
 *       404:
 *         description: Notification not found
 */
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

/**
 * @swagger
 * /api/v1/notifications/{id}/read:
 *   put:
 *     summary: Mark notification as read
 *     tags: [Notifications]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: Notification marked as read
 */
router.put('/:id/read', authenticateToken, async (req, res) => {
  try {
    const { id } = req.params;
    const userId = req.user.id;

    const result = await notificationService.markAsRead(id, userId);
    res.success(result);
  } catch (error) {
    res.error(error.message, 500, 'NOTIFICATION_ERROR');
  }
});

/**
 * @swagger
 * /api/v1/notifications/read-all:
 *   put:
 *     summary: Mark all notifications as read
 *     tags: [Notifications]
 *     security:
 *       - bearerAuth: []
 *     responses:
 *       200:
 *         description: All notifications marked as read
 */
router.put('/read-all', authenticateToken, async (req, res) => {
  try {
    const userId = req.user.id;
    const result = await notificationService.markAllAsRead(userId);
    res.success(result);
  } catch (error) {
    res.error(error.message, 500, 'NOTIFICATION_ERROR');
  }
});

/**
 * @swagger
 * /api/v1/notifications/{id}:
 *   delete:
 *     summary: Delete notification
 *     tags: [Notifications]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: Notification deleted
 */
router.delete('/:id', authenticateToken, async (req, res) => {
  try {
    const { id } = req.params;
    const userId = req.user.id;

    await notificationService.deleteNotification(id, userId);
    res.success({ message: 'Notification deleted' });
  } catch (error) {
    res.error(error.message, 500, 'NOTIFICATION_ERROR');
  }
});

/**
 * @swagger
 * /api/v1/notifications/send:
 *   post:
 *     summary: Send notification
 *     tags: [Notifications]
 *     security:
 *       - bearerAuth: []
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             properties:
 *               userId:
 *                 type: string
 *               title:
 *                 type: string
 *               message:
 *                 type: string
 *               type:
 *                 type: string
 *               channels:
 *                 type: array
 *                 items:
 *                   type: string
 *     responses:
 *       201:
 *         description: Notification sent
 */
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
