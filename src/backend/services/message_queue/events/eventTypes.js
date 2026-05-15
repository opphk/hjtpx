const EventTypes = {
  USER_CREATED: 'user.created',
  USER_UPDATED: 'user.updated',
  USER_DELETED: 'user.deleted',
  USER_LOGGED_IN: 'user.logged_in',
  USER_LOGGED_OUT: 'user.logged_out',
  USER_PASSWORD_CHANGED: 'user.password_changed',
  USER_EMAIL_VERIFIED: 'user.email_verified',

  NOTIFICATION_CREATED: 'notification.created',
  NOTIFICATION_SENT: 'notification.sent',
  NOTIFICATION_READ: 'notification.read',
  NOTIFICATION_DELETED: 'notification.deleted',

  EXPORT_REQUESTED: 'export.requested',
  EXPORT_COMPLETED: 'export.completed',
  EXPORT_FAILED: 'export.failed',

  EMAIL_SENT: 'email.sent',
  EMAIL_FAILED: 'email.failed',

  SECURITY_EVENT: 'security.event',
  SECURITY_LOGIN_FAILED: 'security.login_failed',
  SECURITY_SUSPICIOUS_ACTIVITY: 'security.suspicious_activity',

  ANALYTICS_EVENT: 'analytics.event',
  ANALYTICS_TRACKED: 'analytics.tracked',

  SYSTEM_ERROR: 'system.error',
  SYSTEM_WARNING: 'system.warning',

  CACHE_CLEARED: 'cache.cleared',
  CACHE_WARMED: 'cache.warmed'
};

const EventCategories = {
  USER: 'user',
  NOTIFICATION: 'notification',
  EXPORT: 'export',
  EMAIL: 'email',
  SECURITY: 'security',
  ANALYTICS: 'analytics',
  SYSTEM: 'system',
  CACHE: 'cache'
};

class Event {
  constructor(type, data, options = {}) {
    this.id = options.id || require('uuid').v4();
    this.type = type;
    this.data = data;
    this.timestamp = new Date().toISOString();
    this.source = options.source || 'application';
    this.correlationId = options.correlationId || require('uuid').v4();
    this.causationId = options.causationId || null;
    this.metadata = options.metadata || {};
    this.category = this.getCategory(type);
    this.version = options.version || '1.0';
  }

  getCategory(type) {
    const [prefix] = type.split('.');
    return EventCategories[prefix.toUpperCase()] || 'unknown';
  }

  toJSON() {
    return {
      id: this.id,
      type: this.type,
      data: this.data,
      timestamp: this.timestamp,
      source: this.source,
      correlationId: this.correlationId,
      causationId: this.causationId,
      metadata: this.metadata,
      category: this.category,
      version: this.version
    };
  }

  toBuffer() {
    return Buffer.from(JSON.stringify(this.toJSON()));
  }

  static fromBuffer(buffer) {
    const data = JSON.parse(buffer.toString());
    return new Event(data.type, data.data, {
      id: data.id,
      source: data.source,
      correlationId: data.correlationId,
      causationId: data.causationId,
      metadata: data.metadata,
      version: data.version
    });
  }
}

const EventPriorities = {
  low: 1,
  normal: 5,
  high: 8,
  critical: 10
};

function createEvent(type, data, options = {}) {
  return new Event(type, data, options);
}

function createUserEvent(type, userId, userData, options = {}) {
  return createEvent(type, {
    userId,
    user: userData
  }, {
    source: 'user-service',
    ...options
  });
}

function createNotificationEvent(type, notificationId, notificationData, options = {}) {
  return createEvent(type, {
    notificationId,
    notification: notificationData
  }, {
    source: 'notification-service',
    ...options
  });
}

function createSecurityEvent(type, details, options = {}) {
  return createEvent(type, details, {
    source: 'security-service',
    metadata: {
      ...options.metadata,
      severity: details.severity || 'medium'
    }
  });
}

module.exports = {
  EventTypes,
  EventCategories,
  Event,
  EventPriorities,
  createEvent,
  createUserEvent,
  createNotificationEvent,
  createSecurityEvent
};
