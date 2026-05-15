const PUSH_NOTIFICATION_CONFIG = {
  VAPID_PUBLIC_KEY: import.meta.env.VITE_VAPID_PUBLIC_KEY || '',
  API_ENDPOINT: '/api/v1/notifications',
  SUBSCRIPTION_ENDPOINT: '/api/v1/notifications/subscribe',
  UNSUBSCRIBE_ENDPOINT: '/api/v1/notifications/unsubscribe'
};

class PushNotificationService {
  constructor() {
    this.isSupported = 'Notification' in window && 
                       'serviceWorker' in navigator && 
                       'PushManager' in window;
    this.permission = this.isSupported ? Notification.permission : 'denied';
  }

  async initialize() {
    if (!this.isSupported) {
      console.warn('Push notifications are not supported');
      return false;
    }

    this.permission = Notification.permission;
    
    if (this.permission === 'granted') {
      await this.ensureSubscription();
    }

    return true;
  }

  async requestPermission() {
    if (!this.isSupported) {
      throw new Error('Push notifications are not supported');
    }

    if (this.permission === 'granted') {
      return true;
    }

    if (this.permission === 'denied') {
      throw new Error('Notification permission is denied');
    }

    const result = await Notification.requestPermission();
    this.permission = result;

    if (result === 'granted') {
      await this.subscribe();
      return true;
    }

    return false;
  }

  async ensureSubscription() {
    try {
      const registration = await navigator.serviceWorker.ready;
      const existingSubscription = await registration.pushManager.getSubscription();
      
      if (!existingSubscription) {
        await this.subscribe();
      }
      
      return existingSubscription;
    } catch (error) {
      console.error('Error checking subscription:', error);
      return null;
    }
  }

  async subscribe() {
    if (!PUSH_NOTIFICATION_CONFIG.VAPID_PUBLIC_KEY) {
      console.warn('VAPID public key not configured');
      return null;
    }

    try {
      const registration = await navigator.serviceWorker.ready;
      
      const subscription = await registration.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: this.urlBase64ToUint8Array(PUSH_NOTIFICATION_CONFIG.VAPID_PUBLIC_KEY)
      });

      const subscriptionData = this.extractSubscriptionData(subscription);
      
      const response = await fetch(PUSH_NOTIFICATION_CONFIG.SUBSCRIPTION_ENDPOINT, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(subscriptionData)
      });

      if (!response.ok) {
        throw new Error('Failed to send subscription to server');
      }

      console.log('Push notification subscription successful');
      return subscription;
    } catch (error) {
      console.error('Push notification subscription failed:', error);
      throw error;
    }
  }

  async unsubscribe() {
    try {
      const registration = await navigator.serviceWorker.ready;
      const subscription = await registration.pushManager.getSubscription();
      
      if (subscription) {
        await subscription.unsubscribe();
        
        const subscriptionData = this.extractSubscriptionData(subscription);
        
        await fetch(PUSH_NOTIFICATION_CONFIG.UNSUBSCRIBE_ENDPOINT, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify(subscriptionData)
        });

        console.log('Push notification unsubscription successful');
      }
      
      return true;
    } catch (error) {
      console.error('Push notification unsubscription failed:', error);
      throw error;
    }
  }

  extractSubscriptionData(subscription) {
    const json = subscription.toJSON();
    return {
      endpoint: json.endpoint,
      keys: {
        p256dh: json.keys.p256dh,
        auth: json.keys.auth
      }
    };
  }

  urlBase64ToUint8Array(base64String) {
    const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
    const base64 = (base64String + padding)
      .replace(/-/g, '+')
      .replace(/_/g, '/');
    const rawData = window.atob(base64);
    return Uint8Array.from([...rawData].map(char => char.charCodeAt(0)));
  }

  async sendLocalNotification(title, options = {}) {
    if (this.permission !== 'granted') {
      throw new Error('Notification permission not granted');
    }

    const notificationOptions = {
      icon: options.icon || '/favicon.png',
      badge: options.badge || '/favicon.png',
      vibrate: options.vibrate || [100, 50, 100],
      tag: options.tag || 'local-' + Date.now(),
      requireInteraction: options.requireInteraction || false,
      silent: options.silent || false,
      body: options.body || '',
      data: options.data || {},
      actions: options.actions || [],
      dir: options.dir || 'ltr',
      lang: options.lang || 'zh-CN',
      ...options
    };

    if ('serviceWorker' in navigator && navigator.serviceWorker.controller) {
      navigator.serviceWorker.controller.postMessage({
        action: 'showNotification',
        title: title,
        ...notificationOptions
      });
    } else {
      new Notification(title, notificationOptions);
    }
  }

  async sendTestNotification() {
    return this.sendLocalNotification('测试通知', {
      body: '推送通知功能正常工作！',
      tag: 'test-notification',
      data: { type: 'test' }
    });
  }

  async getSubscriptionStatus() {
    if (!this.isSupported) {
      return { supported: false };
    }

    try {
      const registration = await navigator.serviceWorker.ready;
      const subscription = await registration.pushManager.getSubscription();
      
      return {
        supported: true,
        permission: this.permission,
        subscribed: !!subscription,
        subscription: subscription ? this.extractSubscriptionData(subscription) : null
      };
    } catch (error) {
      console.error('Error getting subscription status:', error);
      return {
        supported: true,
        permission: this.permission,
        subscribed: false,
        error: error.message
      };
    }
  }

  onMessage(callback) {
    if ('serviceWorker' in navigator) {
      navigator.serviceWorker.addEventListener('message', (event) => {
        if (event.data && (event.data.action === 'notificationClick' || 
                          event.data.action === 'notificationAction')) {
          callback(event.data);
        }
      });
    }
  }

  async getNotificationHistory(limit = 20) {
    try {
      const response = await fetch(`${PUSH_NOTIFICATION_CONFIG.API_ENDPOINT}/history?limit=${limit}`);
      
      if (!response.ok) {
        throw new Error('Failed to fetch notification history');
      }

      return await response.json();
    } catch (error) {
      console.error('Error fetching notification history:', error);
      return { notifications: [] };
    }
  }

  async markAsRead(notificationId) {
    try {
      const response = await fetch(`${PUSH_NOTIFICATION_CONFIG.API_ENDPOINT}/${notificationId}/read`, {
        method: 'POST'
      });
      
      return response.ok;
    } catch (error) {
      console.error('Error marking notification as read:', error);
      return false;
    }
  }

  async markAllAsRead() {
    try {
      const response = await fetch(`${PUSH_NOTIFICATION_CONFIG.API_ENDPOINT}/read-all`, {
        method: 'POST'
      });
      
      return response.ok;
    } catch (error) {
      console.error('Error marking all notifications as read:', error);
      return false;
    }
  }
}

export const pushNotificationService = new PushNotificationService();
export default pushNotificationService;
