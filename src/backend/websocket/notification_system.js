let logger;
try {
  const loggerModule = require('../middleware/logger');
  logger = loggerModule.logger || loggerModule;
} catch (error) {
  logger = {
    info: () => {},
    error: () => {},
    warn: () => {},
    debug: () => {}
  };
}

if (!logger.info) logger.info = () => {};
if (!logger.error) logger.error = () => {};
if (!logger.warn) logger.warn = () => {};
if (!logger.debug) logger.debug = () => {};

let Notification;
try {
  Notification = require('../models/Notification');
} catch (error) {
  Notification = {
    createNotification: async () => ({ _id: 'mock-id' }),
    getUserNotifications: async () => ({ notifications: [], pagination: {} }),
    markAsRead: async () => ({ modifiedCount: 1 }),
    markAllAsRead: async () => ({ modifiedCount: 1 })
  };
}

let cacheService;
try {
  cacheService = require('../services/cacheService');
} catch (error) {
  cacheService = {
    get: async () => null,
    set: async () => true,
    del: async () => true
  };
}

class NotificationSystem {
  constructor() {
    this.pendingNotifications = new Map();
    this.notificationHistory = [];
    this.maxHistorySize = 500;
    this.notificationHandlers = new Map();
    this.stats = {
      totalNotifications: 0,
      sentNotifications: 0,
      failedNotifications: 0,
      queuedNotifications: 0,
      byType: {},
      byPriority: {}
    };
  }

  initialize(io, connectionManager) {
    this.io = io;
    this.connectionManager = connectionManager;
    this.setupDefaultHandlers();
    logger.info('Notification system initialized');
  }

  setupDefaultHandlers() {
    this.registerHandler('info', this.handleInfoNotification.bind(this));
    this.registerHandler('success', this.handleSuccessNotification.bind(this));
    this.registerHandler('warning', this.handleWarningNotification.bind(this));
    this.registerHandler('error', this.handleErrorNotification.bind(this));
    this.registerHandler('system', this.handleSystemNotification.bind(this));
    this.registerHandler('message', this.handleMessageNotification.bind(this));
    this.registerHandler('reminder', this.handleReminderNotification.bind(this));
    this.registerHandler('alert', this.handleAlertNotification.bind(this));
  }

  registerHandler(type, handler) {
    this.notificationHandlers.set(type, handler);
    logger.debug(`Notification handler registered for type: ${type}`);
  }

  async sendToUser(userId, notification) {
    this.stats.totalNotifications++;

    const notificationData = {
      id: this.generateNotificationId(),
      userId,
      type: notification.type || 'info',
      title: notification.title,
      message: notification.message,
      data: notification.data || {},
      priority: notification.priority || 'normal',
      channels: notification.channels || ['in_app'],
      actionUrl: notification.actionUrl,
      actionLabel: notification.actionLabel,
      metadata: notification.metadata || {},
      createdAt: new Date(),
      status: 'pending'
    };

    try {
      const savedNotification = await this.persistNotification(notificationData);
      notificationData._id = savedNotification._id;
      notificationData.status = 'sent';

      const handler = this.notificationHandlers.get(notificationData.type);
      if (handler) {
        await handler(notificationData);
      }

      this.emitToUser(userId, 'notification', notificationData);

      this.addToHistory(notificationData);
      this.updateStats(notificationData, 'sent');

      logger.info('Notification sent', {
        notificationId: notificationData._id,
        userId,
        type: notificationData.type,
        priority: notificationData.priority
      });

      return notificationData;
    } catch (error) {
      notificationData.status = 'failed';
      notificationData.error = error.message;
      this.updateStats(notificationData, 'failed');

      logger.error('Failed to send notification', {
        userId,
        type: notificationData.type,
        error: error.message
      });

      this.queueNotification(userId, notificationData);
      throw error;
    }
  }

  async sendToUsers(userIds, notification) {
    const results = await Promise.allSettled(
      userIds.map(userId => this.sendToUser(userId, notification))
    );

    const successful = results.filter(r => r.status === 'fulfilled').length;
    const failed = results.filter(r => r.status === 'rejected').length;

    logger.info('Bulk notification sent', {
      total: userIds.length,
      successful,
      failed,
      type: notification.type
    });

    return results.map((result, index) => ({
      userId: userIds[index],
      success: result.status === 'fulfilled',
      notification: result.status === 'fulfilled' ? result.value : null,
      error: result.status === 'rejected' ? result.reason.message : null
    }));
  }

  broadcast(notification, room = 'notifications') {
    this.stats.totalNotifications++;

    const notificationData = {
      id: this.generateNotificationId(),
      type: notification.type || 'info',
      title: notification.title,
      message: notification.message,
      data: notification.data || {},
      priority: notification.priority || 'normal',
      channels: notification.channels || ['in_app'],
      metadata: notification.metadata || {},
      createdAt: new Date(),
      broadcast: true,
      room
    };

    try {
      this.io.to(room).emit('notification', notificationData);

      this.addToHistory(notificationData);
      this.updateStats(notificationData, 'sent');

      logger.info('Notification broadcast', {
        notificationId: notificationData.id,
        room,
        type: notificationData.type
      });

      return notificationData;
    } catch (error) {
      this.updateStats(notificationData, 'failed');
      logger.error('Failed to broadcast notification', {
        error: error.message,
        room
      });
      throw error;
    }
  }

  emitToUser(userId, event, data) {
    const userConnections = this.connectionManager.getUserConnections(userId);

    if (userConnections.length === 0) {
      logger.debug('User not connected, notification queued', { userId });
      this.queueNotification(userId, { ...data, event });
      return false;
    }

    userConnections.forEach(connection => {
      this.io.to(connection.socketId).emit(event, data);
    });

    return true;
  }

  async persistNotification(notification) {
    try {
      const notificationDoc = new Notification({
        userId: notification.userId,
        type: notification.type,
        title: notification.title,
        message: notification.message,
        data: notification.data,
        priority: notification.priority,
        channels: notification.channels,
        actionUrl: notification.actionUrl,
        actionLabel: notification.actionLabel,
        metadata: notification.metadata
      });

      return await notificationDoc.save();
    } catch (error) {
      logger.error('Failed to persist notification', { error: error.message });
      throw error;
    }
  }

  queueNotification(userId, notification) {
    if (!this.pendingNotifications.has(userId)) {
      this.pendingNotifications.set(userId, []);
    }

    this.pendingNotifications.get(userId).push({
      ...notification,
      queuedAt: new Date()
    });

    this.stats.queuedNotifications++;

    logger.debug('Notification queued', {
      userId,
      notificationId: notification.id,
      queueSize: this.pendingNotifications.get(userId).length
    });
  }

  async deliverQueuedNotifications(userId) {
    const queued = this.pendingNotifications.get(userId);

    if (!queued || queued.length === 0) {
      return [];
    }

    const delivered = [];

    for (const notification of queued) {
      try {
        if (notification.event) {
          this.emitToUser(userId, notification.event, notification);
        }
        delivered.push(notification);
      } catch (error) {
        logger.error('Failed to deliver queued notification', {
          userId,
          notificationId: notification.id,
          error: error.message
        });
      }
    }

    this.pendingNotifications.set(userId, []);
    this.stats.queuedNotifications -= delivered.length;

    logger.info('Queued notifications delivered', {
      userId,
      deliveredCount: delivered.length
    });

    return delivered;
  }

  addToHistory(notification) {
    this.notificationHistory.push({
      ...notification,
      deliveredAt: new Date()
    });

    if (this.notificationHistory.length > this.maxHistorySize) {
      this.notificationHistory = this.notificationHistory.slice(-this.maxHistorySize);
    }
  }

  updateStats(notification, status) {
    if (status === 'sent') {
      this.stats.sentNotifications++;
    } else if (status === 'failed') {
      this.stats.failedNotifications++;
    }

    const type = notification.type || 'unknown';
    if (!this.stats.byType[type]) {
      this.stats.byType[type] = { sent: 0, failed: 0 };
    }
    this.stats.byType[type][status + 'Notifications']++;

    const priority = notification.priority || 'normal';
    if (!this.stats.byPriority[priority]) {
      this.stats.byPriority[priority] = { sent: 0, failed: 0 };
    }
    this.stats.byPriority[priority][status + 'Notifications']++;
  }

  async getNotificationHistory(userId, options = {}) {
    const { limit = 50, offset = 0 } = options;

    try {
      const notifications = await Notification.getUserNotifications(userId, {
        ...options,
        limit,
        page: Math.floor(offset / limit) + 1
      });

      return notifications;
    } catch (error) {
      logger.error('Failed to get notification history', {
        userId,
        error: error.message
      });
      throw error;
    }
  }

  async markAsRead(notificationId, userId) {
    try {
      const notification = await Notification.markAsRead(notificationId, userId);

      if (notification.modifiedCount > 0) {
        this.emitToUser(userId, 'notification:read', {
          notificationId,
          readAt: new Date()
        });

        logger.debug('Notification marked as read', { notificationId, userId });
      }

      return notification;
    } catch (error) {
      logger.error('Failed to mark notification as read', {
        notificationId,
        userId,
        error: error.message
      });
      throw error;
    }
  }

  async markAllAsRead(userId) {
    try {
      const result = await Notification.markAllAsRead(userId);

      if (result.modifiedCount > 0) {
        this.emitToUser(userId, 'notification:allRead', {
          count: result.modifiedCount,
          readAt: new Date()
        });

        logger.debug('All notifications marked as read', {
          userId,
          count: result.modifiedCount
        });
      }

      return result;
    } catch (error) {
      logger.error('Failed to mark all notifications as read', {
        userId,
        error: error.message
      });
      throw error;
    }
  }

  handleInfoNotification(notification) {
    logger.debug('Handling info notification', { notificationId: notification.id });
  }

  handleSuccessNotification(notification) {
    logger.debug('Handling success notification', { notificationId: notification.id });
  }

  handleWarningNotification(notification) {
    logger.warn('Warning notification sent', {
      notificationId: notification.id,
      title: notification.title
    });
  }

  handleErrorNotification(notification) {
    logger.error('Error notification sent', {
      notificationId: notification.id,
      title: notification.title,
      message: notification.message
    });
  }

  handleSystemNotification(notification) {
    logger.info('System notification sent', {
      notificationId: notification.id,
      title: notification.title
    });
  }

  handleMessageNotification(notification) {
    logger.debug('Message notification sent', {
      notificationId: notification.id,
      metadata: notification.metadata
    });
  }

  handleReminderNotification(notification) {
    logger.debug('Reminder notification sent', {
      notificationId: notification.id,
      data: notification.data
    });
  }

  handleAlertNotification(notification) {
    logger.warn('Alert notification sent', {
      notificationId: notification.id,
      title: notification.title,
      priority: notification.priority
    });
  }

  generateNotificationId() {
    return `notif_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  getStats() {
    return {
      ...this.stats,
      pendingDelivery: Array.from(this.pendingNotifications.entries()).map(
        ([userId, notifications]) => ({
          userId,
          count: notifications.length
        })
      ),
      recentHistory: this.notificationHistory.slice(-10).map(n => ({
        id: n.id,
        type: n.type,
        userId: n.userId,
        deliveredAt: n.deliveredAt
      }))
    };
  }

  cleanup() {
    this.pendingNotifications.clear();
    this.notificationHistory = [];
    this.notificationHandlers.clear();
    this.stats = {
      totalNotifications: 0,
      sentNotifications: 0,
      failedNotifications: 0,
      queuedNotifications: 0,
      byType: {},
      byPriority: {}
    };
    logger.info('Notification system cleaned up');
  }
}

const notificationSystem = new NotificationSystem();

module.exports = notificationSystem;
module.exports.NotificationSystem = NotificationSystem;
