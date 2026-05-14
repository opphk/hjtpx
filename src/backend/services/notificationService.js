const Notification = require('../models/Notification');

async function createNotification(data) {
  const notification = await Notification.createNotification(data);
  await sendToChannels(notification);
  return notification;
}

async function createBulkNotifications(notifications) {
  const created = await Notification.createBulkNotifications(notifications);
  created.forEach(notification => sendToChannels(notification));
  return created;
}

async function getUserNotifications(userId, options = {}) {
  return Notification.getUserNotifications(userId, options);
}

async function getUnreadCount(userId) {
  return Notification.getUnreadCount(userId);
}

async function getNotificationById(notificationId, userId) {
  return Notification.findOne({ _id: notificationId, userId });
}

async function markAsRead(notificationId, userId) {
  return Notification.markAsRead(notificationId, userId);
}

async function markAllAsRead(userId) {
  return Notification.markAllAsRead(userId);
}

async function deleteNotification(notificationId, userId) {
  return Notification.deleteOne({ _id: notificationId, userId });
}

async function deleteOldNotifications(userId, daysOld = 30) {
  const cutoffDate = new Date();
  cutoffDate.setDate(cutoffDate.getDate() - daysOld);

  return Notification.deleteMany({
    userId,
    status: { $in: ['read', 'archived'] },
    createdAt: { $lt: cutoffDate }
  });
}

async function sendToChannels(notification) {
  for (const channel of notification.channels) {
    try {
      switch (channel) {
        case 'email':
          await sendEmailNotification(notification);
          break;
        case 'push':
          await sendPushNotification(notification);
          break;
        case 'sms':
          await sendSMSNotification(notification);
          break;
        case 'in_app':
        default:
          break;
      }
    } catch (error) {
      console.error(`Failed to send notification via ${channel}:`, error);
    }
  }
}

async function sendEmailNotification(notification) {
  console.log(`[Email] Sending notification to user ${notification.userId}: ${notification.title}`);
  return { sent: true };
}

async function sendPushNotification(notification) {
  console.log(`[Push] Sending notification to user ${notification.userId}: ${notification.title}`);
  return { sent: true };
}

async function sendSMSNotification(notification) {
  console.log(`[SMS] Sending notification to user ${notification.userId}: ${notification.message}`);
  return { sent: true };
}

async function archiveOldNotifications(daysOld = 30) {
  return Notification.archiveOld(daysOld);
}

module.exports = {
  createNotification,
  createBulkNotifications,
  getUserNotifications,
  getUnreadCount,
  getNotificationById,
  markAsRead,
  markAllAsRead,
  deleteNotification,
  deleteOldNotifications,
  archiveOldNotifications,
  sendToChannels
};
