import { useState, useEffect } from 'react';

const usePushNotifications = () => {
  const [isSupported, setIsSupported] = useState(false);
  const [permission, setPermission] = useState('default');
  const [subscription, setSubscription] = useState(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    if ('Notification' in window && 'serviceWorker' in navigator && 'PushManager' in window) {
      setIsSupported(true);
      setPermission(Notification.permission);
      checkExistingSubscription();
    } else {
      setIsSupported(false);
      setIsLoading(false);
    }
  }, []);

  const checkExistingSubscription = async () => {
    try {
      const registration = await navigator.serviceWorker.ready;
      const existingSubscription = await registration.pushManager.getSubscription();
      setSubscription(existingSubscription);
      setIsLoading(false);
    } catch (error) {
      console.error('检查订阅状态失败:', error);
      setError(error.message);
      setIsLoading(false);
    }
  };

  const requestPermission = async () => {
    if (!isSupported) {
      console.log('通知功能不被支持');
      return false;
    }

    try {
      const result = await Notification.requestPermission();
      setPermission(result);
      if (result === 'granted') {
        console.log('通知权限已授予');
        return true;
      } else if (result === 'denied') {
        console.log('通知权限被拒绝');
        return false;
      } else {
        console.log('通知权限请求被取消');
        return false;
      }
    } catch (error) {
      console.error('请求通知权限失败:', error);
      setError(error.message);
      return false;
    }
  };

  const subscribeToPush = async (publicKey) => {
    if (!isSupported || permission !== 'granted') {
      console.log('通知功能未启用');
      return null;
    }

    if (!publicKey) {
      console.error('VAPID 公钥未提供');
      setError('VAPID 公钥未提供');
      return null;
    }

    try {
      const registration = await navigator.serviceWorker.ready;
      const existingSubscription = await registration.pushManager.getSubscription();
      
      if (existingSubscription) {
        console.log('使用现有订阅');
        setSubscription(existingSubscription);
        return existingSubscription;
      }

      const options = {
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToUint8Array(publicKey)
      };

      const newSubscription = await registration.pushManager.subscribe(options);
      setSubscription(newSubscription);
      console.log('推送订阅成功');
      return newSubscription;
    } catch (error) {
      console.error('订阅推送通知失败:', error);
      setError(error.message);
      return null;
    }
  };

  const unsubscribeFromPush = async () => {
    if (!subscription) {
      return true;
    }

    try {
      await subscription.unsubscribe();
      setSubscription(null);
      console.log('推送订阅已取消');
      return true;
    } catch (error) {
      console.error('取消订阅推送通知失败:', error);
      setError(error.message);
      return false;
    }
  };

  const sendLocalNotification = (title, options = {}) => {
    if (permission !== 'granted') {
      console.log('通知权限未授予');
      return null;
    }

    if ('Notification' in window) {
      const notification = new Notification(title, {
        icon: options.icon || '/favicon.png',
        badge: options.badge || '/favicon.png',
        vibrate: options.vibrate || [100, 50, 100],
        tag: options.tag || 'local-' + Date.now(),
        requireInteraction: options.requireInteraction || false,
        silent: options.silent || false,
        ...options
      });
      return notification;
    }
    return null;
  };

  const sendNotificationViaServiceWorker = (title, options = {}) => {
    if (!('serviceWorker' in navigator) || !navigator.serviceWorker.controller) {
      console.error('Service Worker 未就绪');
      return;
    }

    navigator.serviceWorker.controller.postMessage({
      action: 'showNotification',
      title: title,
      body: options.body || '',
      icon: options.icon || '/favicon.png',
      badge: options.badge || '/favicon.png',
      tag: options.tag || 'sw-' + Date.now(),
      data: options.data || {},
      vibrate: options.vibrate || [100, 50, 100],
      requireInteraction: options.requireInteraction || false,
      actions: options.actions || []
    });
  };

  const urlBase64ToUint8Array = (base64String) => {
    const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
    const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');
    const rawData = window.atob(base64);
    return Uint8Array.from([...rawData].map((char) => char.charCodeAt(0)));
  };

  const getSubscriptionInfo = async () => {
    if (!subscription) {
      return null;
    }

    const json = subscription.toJSON();
    return {
      endpoint: json.endpoint,
      keys: {
        p256dh: json.keys.p256dh,
        auth: json.keys.auth
      }
    };
  };

  const clearError = () => {
    setError(null);
  };

  return {
    isSupported,
    permission,
    subscription,
    isLoading,
    error,
    requestPermission,
    subscribeToPush,
    unsubscribeFromPush,
    sendLocalNotification,
    sendNotificationViaServiceWorker,
    getSubscriptionInfo,
    clearError
  };
};

export default usePushNotifications;
