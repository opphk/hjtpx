const express = require('express');
const router = express.Router();
const notificationService = require('../services/notificationService');
const { authMiddleware } = require('../middleware/auth');
const { validate, notificationSchemas } = require('../middleware/validator');

router.use(authMiddleware);

router.get('/', async (req, res, next) => {
  try {
    const { status, type, page, limit, sortBy, order } = req.query;

    const result = await notificationService.getUserNotifications(
      req.user.id,
      {
        status,
        type,
        page: parseInt(page) || 1,
        limit: parseInt(limit) || 20,
        sortBy: sortBy || 'createdAt',
        order: order || 'desc'
      }
    );

    res.json({
      success: true,
      data: result.notifications,
      pagination: result.pagination
    });
  } catch (error) {
    next(error);
  }
});

router.get('/unread-count', async (req, res, next) => {
  try {
    const count = await notificationService.getUnreadCount(req.user.id);

    res.json({
      success: true,
      data: { count }
    });
  } catch (error) {
    next(error);
  }
});

router.get('/:id', async (req, res, next) => {
  try {
    const notification = await notificationService.getNotificationById(
      req.params.id,
      req.user.id
    );

    if (!notification) {
      return res.status(404).json({
        success: false,
        error: 'Notification not found'
      });
    }

    res.json({
      success: true,
      data: notification
    });
  } catch (error) {
    next(error);
  }
});

router.post('/', validate(notificationSchemas.create), async (req, res, next) => {
  try {
    const notification = await notificationService.createNotification({
      ...req.body,
      userId: req.user.id
    });

    res.status(201).json({
      success: true,
      data: notification
    });
  } catch (error) {
    next(error);
  }
});

router.put('/:id/read', async (req, res, next) => {
  try {
    const result = await notificationService.markAsRead(
      req.params.id,
      req.user.id
    );

    res.json({
      success: true,
      data: result
    });
  } catch (error) {
    next(error);
  }
});

router.put('/read-all', async (req, res, next) => {
  try {
    const result = await notificationService.markAllAsRead(req.user.id);

    res.json({
      success: true,
      data: result
    });
  } catch (error) {
    next(error);
  }
});

router.delete('/:id', async (req, res, next) => {
  try {
    await notificationService.deleteNotification(req.params.id, req.user.id);

    res.json({
      success: true,
      message: 'Notification deleted'
    });
  } catch (error) {
    next(error);
  }
});

router.delete('/', async (req, res, next) => {
  try {
    const result = await notificationService.deleteOldNotifications(
      req.user.id,
      parseInt(req.query.daysOld) || 30
    );

    res.json({
      success: true,
      data: result
    });
  } catch (error) {
    next(error);
  }
});

module.exports = router;
