const { logInfo: loggerInfo, logWarning: loggerWarn } = require('../middleware/logger');
const WebSocketServer = require('../websocket');

const notificationService = require('./notificationService');

class WebSocketService {
  constructor() {
    this.io = null;
    this.presenceInterval = null;
    this.reconnectAttempts = new Map();
    this.maxReconnectAttempts = 5;
    this.reconnectDelay = 1000;
  }

  initialize(httpServer) {
    if (this.io) {
      loggerWarn('WebSocket server already initialized');
      return;
    }

    this.io = new WebSocketServer(httpServer);

    this.setupNotificationHandlers();

    this.startPresenceChecker();

    loggerInfo('WebSocket service initialized');
  }

  setupNotificationHandlers() {
    this.io.broadcast = (event, data, room) => {
      this.io.io.to(room || 'notifications').emit(event, data);
    };
  }

  startPresenceChecker() {
    this.presenceInterval = setInterval(() => {
      this.checkUserPresence();
    }, 30000);
  }

  checkUserPresence() {
    try {
      const stats = this.io.getConnectionStats();
      const onlineUsers = stats.onlineUsers;

      if (onlineUsers > 0) {
        logger.debug('Presence check', {
          onlineUsers,
          totalConnections: stats.totalConnections
        });
      }
    } catch (error) {
      logger.error('Error in presence checker', { error: error.message });
    }
  }

  async sendNotification(userId, notification) {
    try {
      const savedNotification = await notificationService.createNotification({
        userId,
        type: notification.type,
        title: notification.title,
        message: notification.message,
        data: notification.data,
        priority: notification.priority || 'normal'
      });

      const sent = this.io.sendToUser(userId, 'notification', {
        id: savedNotification._id,
        type: notification.type,
        title: notification.title,
        message: notification.message,
        data: notification.data,
        priority: notification.priority || 'normal',
        timestamp: new Date()
      });

      if (!sent) {
        logger.warn('User not connected, notification queued', { userId });
      }

      return savedNotification;
    } catch (error) {
      logger.error('Error sending notification', {
        userId,
        error: error.message
      });
      throw error;
    }
  }

  async sendBulkNotifications(userIds, notification) {
    const results = await Promise.all(
      userIds.map(userId => this.sendNotification(userId, notification))
    );
    return results;
  }

  async sendToRole(role, notification) {
    try {
      const User = require('../models/User');
      const users = await User.find({ role, isActive: true });

      await this.sendBulkNotifications(
        users.map(u => u._id),
        notification
      );

      logger.info('Bulk notification sent to role', {
        role,
        recipientCount: users.length
      });
    } catch (error) {
      logger.error('Error sending notification to role', {
        role,
        error: error.message
      });
      throw error;
    }
  }

  broadcastNotification(notification, room = 'notifications') {
    this.io.broadcast(
      'notification',
      {
        type: notification.type,
        title: notification.title,
        message: notification.message,
        data: notification.data,
        priority: notification.priority || 'normal',
        timestamp: new Date()
      },
      room
    );
  }

  async pushDataUpdate(userId, entityType, entityId, action, data) {
    const sent = this.io.sendToUser(userId, 'data:update', {
      entityType,
      entityId,
      action,
      data,
      timestamp: new Date()
    });

    if (!sent) {
      logger.debug('User not connected for data update', { userId, entityType });
    }

    return sent;
  }

  async broadcastDataUpdate(entityType, entityId, action, data, room = null) {
    this.io.broadcast(
      'data:update',
      {
        entityType,
        entityId,
        action,
        data,
        timestamp: new Date()
      },
      room
    );
  }

  notifyUserOnline(userId) {
    this.io.broadcastUserOnlineStatus(userId, true);
  }

  notifyUserOffline(userId) {
    this.io.broadcastUserOnlineStatus(userId, false);
  }

  getOnlineUsers() {
    return this.io.getOnlineUsers();
  }

  getConnectionStats() {
    return this.io.getConnectionStats();
  }

  getDetailedMetrics() {
    return this.io.getDetailedMetrics();
  }

  isUserOnline(userId) {
    const onlineUsers = this.getOnlineUsers();
    return onlineUsers.some(user => user.userId === userId);
  }

  getUserSockets(userId) {
    const clients = this.io.getConnectedClients();
    return clients.filter(client => client.userId === userId);
  }

  handleReconnection(socketId, userId) {
    const attempts = this.reconnectAttempts.get(userId) || 0;

    if (attempts < this.maxReconnectAttempts) {
      this.reconnectAttempts.set(userId, attempts + 1);

      setTimeout(
        () => {
          this.io.sendToUser(userId, 'reconnected', {
            socketId,
            timestamp: new Date()
          });
        },
        this.reconnectDelay * (attempts + 1)
      );
    } else {
      logger.warn('Max reconnection attempts reached', { userId });
      this.reconnectAttempts.delete(userId);
    }
  }

  close() {
    if (this.presenceInterval) {
      clearInterval(this.presenceInterval);
    }

    if (this.io) {
      this.io.close();
      this.io = null;
    }

    logger.info('WebSocket service closed');
  }
}

const websocketService = new WebSocketService();

module.exports = websocketService;
