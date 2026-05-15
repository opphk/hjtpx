import { useState, useEffect } from 'react';

const useServiceWorker = () => {
  const [isSupported, setIsSupported] = useState(false);
  const [registration, setRegistration] = useState(null);
  const [updateAvailable, setUpdateAvailable] = useState(false);
  const [isOnline, setIsOnline] = useState(navigator.onLine);
  const [cacheStatus, setCacheStatus] = useState(null);
  const [swVersion, setSwVersion] = useState(null);
  const [isUpdating, setIsUpdating] = useState(false);
  const [updateError, setUpdateError] = useState(null);

  useEffect(() => {
    if ('serviceWorker' in navigator) {
      setIsSupported(true);
      registerServiceWorker();
    }

    const handleOnline = () => setIsOnline(true);
    const handleOffline = () => setIsOnline(false);

    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);

    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
    };
  }, []);

  const registerServiceWorker = async () => {
    try {
      const reg = await navigator.serviceWorker.register('/sw.js', {
        scope: '/'
      });
      setRegistration(reg);

      reg.addEventListener('updatefound', () => {
        const newWorker = reg.installing;
        
        if (newWorker) {
          newWorker.addEventListener('statechange', () => {
            if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
              setUpdateAvailable(true);
              setUpdateError(null);
            }
          });
        }
      });

      reg.addEventListener('controllerchange', () => {
        console.log('Service Worker 更新完成');
        setUpdateAvailable(false);
        setIsUpdating(false);
        window.location.reload();
      });

      const existingWorker = reg.active;
      if (existingWorker && existingWorker.state === 'activated') {
        getServiceWorkerVersion();
      }

      console.log('Service Worker 注册成功:', reg.scope);
    } catch (error) {
      console.error('Service Worker 注册失败:', error);
      setUpdateError(error.message);
    }
  };

  const getServiceWorkerVersion = () => {
    if (navigator.serviceWorker.controller) {
      navigator.serviceWorker.controller.postMessage({ action: 'getVersion' });
      
      const handleMessage = (event) => {
        if (event.data.action === 'version') {
          setSwVersion(event.data.version);
          navigator.serviceWorker.removeEventListener('message', handleMessage);
        }
      };
      
      navigator.serviceWorker.addEventListener('message', handleMessage);
    }
  };

  const updateServiceWorker = () => {
    if (registration) {
      setIsUpdating(true);
      setUpdateError(null);
      registration.update().catch((error) => {
        console.error('更新 Service Worker 失败:', error);
        setUpdateError(error.message);
        setIsUpdating(false);
      });
    }
  };

  const skipWaiting = () => {
    if (registration && registration.waiting) {
      registration.waiting.postMessage({ action: 'skipWaiting' });
    }
  };

  const clearCache = async () => {
    if (navigator.serviceWorker.controller) {
      return new Promise((resolve) => {
        const handleMessage = (event) => {
          if (event.data.action === 'cacheCleared') {
            navigator.serviceWorker.removeEventListener('message', handleMessage);
            resolve(event.data.success);
          }
        };
        
        navigator.serviceWorker.addEventListener('message', handleMessage);
        navigator.serviceWorker.controller.postMessage({ action: 'clearCache' });

        setTimeout(() => {
          navigator.serviceWorker.removeEventListener('message', handleMessage);
          resolve(false);
        }, 5000);
      });
    }
    return false;
  };

  const getCacheStatus = async () => {
    if (navigator.serviceWorker.controller) {
      return new Promise((resolve) => {
        const handleMessage = (event) => {
          if (event.data.action === 'cacheStatus') {
            navigator.serviceWorker.removeEventListener('message', handleMessage);
            setCacheStatus(event.data.stats);
            resolve(event.data.stats);
          }
        };
        
        navigator.serviceWorker.addEventListener('message', handleMessage);
        navigator.serviceWorker.controller.postMessage({ action: 'getCacheStatus' });

        setTimeout(() => {
          navigator.serviceWorker.removeEventListener('message', handleMessage);
          resolve(null);
        }, 5000);
      });
    }
    return null;
  };

  const registerSync = async (tag = 'sync-data') => {
    if (registration && 'SyncManager' in window) {
      try {
        await registration.sync.register(tag);
        console.log('后台同步注册成功');
        return true;
      } catch (error) {
        console.error('后台同步注册失败:', error);
        return false;
      }
    }
    return false;
  };

  const prefetchResources = async (urls) => {
    if (!navigator.serviceWorker.controller) {
      console.error('Service Worker 未就绪');
      return false;
    }

    return new Promise((resolve, reject) => {
      const handleMessage = (event) => {
        if (event.data.action === 'prefetchComplete') {
          navigator.serviceWorker.removeEventListener('message', handleMessage);
          resolve(true);
        } else if (event.data.action === 'prefetchFailed') {
          navigator.serviceWorker.removeEventListener('message', handleMessage);
          reject(new Error(event.data.error));
        }
      };
      
      navigator.serviceWorker.addEventListener('message', handleMessage);
      navigator.serviceWorker.controller.postMessage({
        action: 'prefetch',
        urls: urls
      });

      setTimeout(() => {
        navigator.serviceWorker.removeEventListener('message', handleMessage);
        resolve(false);
      }, 30000);
    });
  };

  const showCustomNotification = (title, options = {}) => {
    if (!navigator.serviceWorker.controller) {
      console.error('Service Worker 未就绪');
      return;
    }

    navigator.serviceWorker.controller.postMessage({
      action: 'showNotification',
      title: title,
      ...options
    });
  };

  return {
    isSupported,
    registration,
    updateAvailable,
    isOnline,
    cacheStatus,
    swVersion,
    isUpdating,
    updateError,
    updateServiceWorker,
    skipWaiting,
    clearCache,
    getCacheStatus,
    registerSync,
    prefetchResources,
    showCustomNotification,
    getServiceWorkerVersion
  };
};

export default useServiceWorker;
