const mongoose = require('mongoose');

const Notification = require('../../models/Notification');

const defaultNotificationAttributes = {
  type: 'info',
  title: 'Test Notification',
  message: 'This is a test notification message',
  status: 'unread',
  channels: ['in_app'],
  priority: 'medium'
};

async function createNotification(userId, overrides = {}) {
  const attributes = {
    ...defaultNotificationAttributes,
    userId,
    ...overrides
  };

  try {
    const notification = await Notification.create(attributes);
    return notification;
  } catch (error) {
    console.error('Error creating notification:', error);
    throw error;
  }
}

async function createMultipleNotifications(userId, count, overrides = {}) {
  const notifications = [];
  for (let i = 0; i < count; i++) {
    const notification = await createNotification(userId, {
      ...overrides,
      title: `${overrides.title || 'Test Notification'} ${i + 1}`
    });
    notifications.push(notification);
  }
  return notifications;
}

async function createReadNotification(userId, overrides = {}) {
  return createNotification(userId, {
    ...overrides,
    status: 'read'
  });
}

async function createUnreadNotification(userId, overrides = {}) {
  return createNotification(userId, {
    ...overrides,
    status: 'unread'
  });
}

async function deleteNotification(notificationId) {
  try {
    await Notification.findByIdAndDelete(notificationId);
  } catch (error) {
    console.error('Error deleting notification:', error);
  }
}

async function deleteNotifications(notificationIds) {
  for (const notificationId of notificationIds) {
    await deleteNotification(notificationId);
  }
}

async function deleteUserNotifications(userId) {
  try {
    await Notification.deleteMany({ userId });
  } catch (error) {
    console.error('Error deleting user notifications:', error);
  }
}

const NOTIFICATION_TYPES = ['info', 'success', 'warning', 'error'];
const NOTIFICATION_PRIORITIES = ['low', 'medium', 'high', 'urgent'];

module.exports = {
  createNotification,
  createMultipleNotifications,
  createReadNotification,
  createUnreadNotification,
  deleteNotification,
  deleteNotifications,
  deleteUserNotifications,
  defaultNotificationAttributes,
  NOTIFICATION_TYPES,
  NOTIFICATION_PRIORITIES
};
