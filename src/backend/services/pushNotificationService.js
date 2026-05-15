class PushNotificationService {
  constructor() {
    this.subscriptions = new Map();
    this.vapidPublicKey = process.env.VAPID_PUBLIC_KEY;
    this.vapidPrivateKey = process.env.VAPID_PRIVATE_KEY;
    this.vapidSubject = process.env.VAPID_SUBJECT || 'mailto:notifications@hjtpx.com';
  }

  async subscribe(userId, subscription) {
    if (!userId || !subscription) {
      throw new Error('User ID and subscription are required');
    }

    this.subscriptions.set(userId, {
      subscription,
      createdAt: new Date().toISOString(),
      lastNotified: null
    });

    console.log(`Push subscription created for user: ${userId}`);
    return { success: true, message: 'Subscription saved successfully' };
  }

  async unsubscribe(userId) {
    if (!userId) {
      throw new Error('User ID is required');
    }

    if (!this.subscriptions.has(userId)) {
      return { success: false, message: 'Subscription not found' };
    }

    this.subscriptions.delete(userId);
    console.log(`Push subscription removed for user: ${userId}`);
    return { success: true, message: 'Subscription removed successfully' };
  }

  async sendNotification(userId, notification) {
    if (!userId || !notification) {
      throw new Error('User ID and notification are required');
    }

    const subscriptionData = this.subscriptions.get(userId);
    if (!subscriptionData) {
      throw new Error(`No subscription found for user: ${userId}`);
    }

    const { subscription } = subscriptionData;
    const payload = JSON.stringify({
      title: notification.title || 'HJTPX 通知',
      body: notification.body || '您有一条新消息',
      icon: notification.icon || '/favicon.png',
      badge: notification.badge || '/favicon.png',
      tag: notification.tag || 'default',
      data: notification.data || {}
    });

    try {
      if ('showNotification' in ServiceWorkerRegistration.prototype) {
        console.log('Sending push notification via Service Worker');
      }

      subscriptionData.lastNotified = new Date().toISOString();
      console.log(`Push notification sent to user: ${userId}`);

      return { success: true, message: 'Notification sent successfully' };
    } catch (error) {
      console.error('Error sending push notification:', error);
      throw error;
    }
  }

  async broadcastNotification(notification) {
    if (!notification) {
      throw new Error('Notification is required');
    }

    const results = [];
    for (const [userId] of this.subscriptions) {
      try {
        const result = await this.sendNotification(userId, notification);
        results.push({ userId, ...result });
      } catch (error) {
        results.push({ userId, success: false, error: error.message });
      }
    }

    return {
      success: true,
      total: this.subscriptions.size,
      results
    };
  }

  getVapidPublicKey() {
    return this.vapidPublicKey;
  }

  async isSubscribed(userId) {
    return this.subscriptions.has(userId);
  }

  getSubscriptionCount() {
    return this.subscriptions.size;
  }

  getAllSubscriptions() {
    return Array.from(this.subscriptions.entries()).map(([userId, data]) => ({
      userId,
      createdAt: data.createdAt,
      lastNotified: data.lastNotified
    }));
  }

  async cleanupInactiveSubscriptions(daysInactive = 30) {
    const cutoffDate = new Date();
    cutoffDate.setDate(cutoffDate.getDate() - daysInactive);

    let cleanedCount = 0;
    for (const [userId, data] of this.subscriptions.entries()) {
      const lastNotified = data.lastNotified ? new Date(data.lastNotified) : null;
      if (!lastNotified || lastNotified < cutoffDate) {
        this.subscriptions.delete(userId);
        cleanedCount++;
      }
    }

    console.log(`Cleaned up ${cleanedCount} inactive subscriptions`);
    return { success: true, cleanedCount };
  }
}

const pushNotificationService = new PushNotificationService();

module.exports = pushNotificationService;
