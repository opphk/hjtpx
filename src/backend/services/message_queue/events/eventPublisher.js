const connectionManager = require('../connectionManager');
const config = require('../../config/messageQueue');
const { Event, createEvent } = require('./eventTypes');
const { v4: uuidv4 } = require('uuid');

class EventPublisher {
  constructor() {
    this.channels = new Map();
    this.isConnected = false;
  }

  async connect() {
    if (this.isConnected) {
      return;
    }

    this.client = connectionManager.getClient();
    this.isConnected = true;
    console.log('[EventPublisher] Connected');
  }

  async publish(event, options = {}) {
    if (!this.isConnected) {
      await this.connect();
    }

    const channel = options.channel || `${config.events.pubSubPrefix}${event.type}`;
    const serializedEvent = this.serializeEvent(event);

    try {
      await this.client.publish(channel, serializedEvent);
      console.log(`[EventPublisher] Event published: ${event.type} to ${channel}`);
      return {
        eventId: event.id,
        channel,
        timestamp: event.timestamp
      };
    } catch (error) {
      console.error(`[EventPublisher] Failed to publish event:`, error);
      throw error;
    }
  }

  async publishBatch(events, options = {}) {
    const results = [];
    for (const event of events) {
      try {
        const result = await this.publish(event, options);
        results.push({ success: true, ...result });
      } catch (error) {
        results.push({ success: false, error: error.message });
      }
    }
    return results;
  }

  serializeEvent(event) {
    if (event instanceof Event) {
      return JSON.stringify(event.toJSON());
    }
    return JSON.stringify(event);
  }

  async publishUserCreated(user, options = {}) {
    const event = createEvent('user.created', {
      userId: user._id || user.id,
      username: user.username,
      email: user.email,
      createdAt: user.createdAt
    }, { source: 'user-service', ...options });

    return await this.publish(event);
  }

  async publishUserUpdated(user, changes, options = {}) {
    const event = createEvent('user.updated', {
      userId: user._id || user.id,
      changes,
      updatedAt: new Date().toISOString()
    }, { source: 'user-service', ...options });

    return await this.publish(event);
  }

  async publishUserLoggedIn(user, metadata = {}, options = {}) {
    const event = createEvent('user.logged_in', {
      userId: user._id || user.id,
      username: user.username,
      timestamp: new Date().toISOString(),
      ...metadata
    }, { source: 'auth-service', ...options });

    return await this.publish(event);
  }

  async publishNotificationCreated(notification, options = {}) {
    const event = createEvent('notification.created', {
      notificationId: notification._id || notification.id,
      userId: notification.userId,
      type: notification.type,
      title: notification.title,
      message: notification.message
    }, { source: 'notification-service', ...options });

    return await this.publish(event);
  }

  async publishExportCompleted(exportId, userId, downloadUrl, options = {}) {
    const event = createEvent('export.completed', {
      exportId,
      userId,
      downloadUrl,
      completedAt: new Date().toISOString()
    }, { source: 'export-service', ...options });

    return await this.publish(event);
  }

  async publishSecurityEvent(eventType, details, options = {}) {
    const event = createEvent('security.event', {
      eventType,
      details,
      timestamp: new Date().toISOString()
    }, { source: 'security-service', severity: details.severity || 'medium', ...options });

    return await this.publish(event);
  }

  async healthCheck() {
    try {
      if (!this.isConnected) {
        await this.connect();
      }
      await this.client.ping();
      return {
        healthy: true,
        connected: this.isConnected,
        channels: this.channels.size
      };
    } catch (error) {
      return {
        healthy: false,
        error: error.message
      };
    }
  }
}

const eventPublisher = new EventPublisher();

module.exports = eventPublisher;
