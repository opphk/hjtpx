const { consumerManager } = require('./consumers/streamConsumer');
const { producerManager } = require('./producers/streamProducer');

class NotificationQueueService {
  constructor() {
    this.queueName = 'notification';
  }

  async sendNotification(userId, notification, options = {}) {
    return await producerManager.send(
      this.queueName,
      {
        userId,
        notification,
        options
      },
      {
        type: 'push_notification',
        priority: notification.priority || 0,
        correlationId: options.correlationId
      }
    );
  }

  async sendBulkNotifications(userIds, notification, options = {}) {
    const messages = userIds.map(userId => ({
      userId,
      notification,
      options
    }));

    return await producerManager.sendBatch(
      this.queueName,
      messages.map(msg => ({
        ...msg,
        type: 'push_notification'
      })),
      options
    );
  }

  async sendPushNotification(userId, title, body, data = {}, options = {}) {
    return await this.sendNotification(
      userId,
      {
        type: 'push',
        title,
        body,
        data,
        priority: options.priority || 0
      },
      options
    );
  }

  async sendInAppNotification(userId, notification) {
    return await this.sendNotification(userId, {
      type: 'in_app',
      ...notification
    });
  }

  async sendEmailNotification(userId, notification) {
    return await this.sendNotification(userId, {
      type: 'email',
      ...notification
    });
  }

  async scheduleNotification(userId, notification, scheduledTime, options = {}) {
    const delay = new Date(scheduledTime).getTime() - Date.now();
    if (delay <= 0) {
      throw new Error('Scheduled time must be in the future');
    }

    return await producerManager.sendWithDelay(
      this.queueName,
      {
        userId,
        notification,
        options
      },
      delay,
      {
        type: 'push_notification',
        priority: notification.priority || 0
      }
    );
  }

  async startConsumer(options = {}) {
    const consumer = await consumerManager.createConsumer(this.queueName, options);

    consumer.registerHandler('push_notification', async message => {
      const notificationService = require('../notificationService');
      const Notification = require('../../models/Notification');

      const { userId, notification, options: msgOptions } = message.payload;

      if (notification.type === 'push' || notification.type === 'in_app') {
        await notificationService.createNotification(userId, {
          type: notification.type,
          title: notification.title,
          message: notification.body || notification.message,
          data: notification.data
        });
      }

      if (notification.type === 'email') {
        const emailQueueService = require('./emailQueueService');
        await emailQueueService.sendNotificationEmail(
          { _id: userId, email: notification.email },
          notification
        );
      }

      if (msgOptions?.broadcast) {
        const WebSocketService = require('../websocketService');
        WebSocketService.broadcastToUser(userId, {
          type: 'notification',
          data: notification
        });
      }

      console.log(`[NotificationQueue] Notification sent to user ${userId}`);
    });

    return consumer;
  }
}

const notificationQueueService = new NotificationQueueService();

module.exports = notificationQueueService;
