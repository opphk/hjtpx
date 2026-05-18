(function() {
  'use strict';

  const PWA_CONFIG = {
    serviceWorkerPath: '/static/service-worker.js',
    manifestPath: '/static/manifest.json',
    offlineDetectionInterval: 1000,
    cacheRefreshInterval: 3600000,
    updateCheckInterval: 60000
  };

  let isOnline = navigator.onLine;
  let registration = null;
  let deferredPrompt = null;
  let updateAvailable = false;
  let notificationPermission = 'default';

  const PWA = {
    init: function() {
      console.log('[PWA] 初始化 PWA 模块 v2');
      this.registerServiceWorker();
      this.setupOnlineOfflineEvents();
      this.setupInstallPrompt();
      this.injectOfflineIndicator();
      this.setupNotificationPermission();
      this.setupPeriodicUpdateCheck();
      this.setupBadgeSupport();
      this.setupShareTarget();
    },

    registerServiceWorker: async function() {
      if (!('serviceWorker' in navigator)) {
        console.warn('[PWA] 浏览器不支持 Service Worker');
        return;
      }

      try {
        registration = await navigator.serviceWorker.register(PWA_CONFIG.serviceWorkerPath, {
          scope: '/',
          updateViaCache: 'none'
        });
        console.log('[PWA] Service Worker 注册成功:', registration.scope);

        registration.addEventListener('updatefound', () => {
          console.log('[PWA] 发现 Service Worker 更新');
          const newWorker = registration.installing;
          
          newWorker.addEventListener('statechange', () => {
            switch (newWorker.state) {
              case 'installing':
                console.log('[PWA] 正在安装新的 Service Worker');
                break;
              case 'installed':
                console.log('[PWA] Service Worker 安装完成');
                if (navigator.serviceWorker.controller) {
                  updateAvailable = true;
                  this.showUpdateNotification();
                }
                break;
              case 'activating':
                console.log('[PWA] 正在激活新的 Service Worker');
                break;
              case 'activated':
                console.log('[PWA] Service Worker 已激活');
                break;
              case 'redundant':
                console.warn('[PWA] Service Worker 已废弃');
                break;
            }
          });
        });

        if (registration.waiting) {
          updateAvailable = true;
          this.showUpdateNotification();
        }

      } catch (error) {
        console.error('[PWA] Service Worker 注册失败:', error);
      }
    },

    setupOnlineOfflineEvents: function() {
      const checkConnection = () => {
        const status = navigator.onLine;
        if (status !== isOnline) {
          isOnline = status;
          this.updateOfflineIndicator();
          this.dispatchConnectionEvent(status);
          
          if (status) {
            this.syncPendingData();
          }
        }
      };

      window.addEventListener('online', checkConnection);
      window.addEventListener('offline', checkConnection);

      setInterval(checkConnection, PWA_CONFIG.offlineDetectionInterval);
    },

    dispatchConnectionEvent: function(online) {
      const event = new CustomEvent('connectionchange', {
        detail: { online: online },
        bubbles: true
      });
      document.dispatchEvent(event);
    },

    setupInstallPrompt: function() {
      window.addEventListener('beforeinstallprompt', (e) => {
        console.log('[PWA] 收到安装提示事件');
        e.preventDefault();
        deferredPrompt = e;
        this.showInstallButton();
        
        e.userChoice.then((choiceResult) => {
          console.log('[PWA] 用户安装选择:', choiceResult.outcome);
          deferredPrompt = null;
        });
      });

      window.addEventListener('appinstalled', () => {
        console.log('[PWA] 应用已安装');
        deferredPrompt = null;
        this.hideInstallButton();
        this.showInstallSuccessNotification();
      });
    },

    injectOfflineIndicator: function() {
      const indicator = document.createElement('div');
      indicator.id = 'pwa-offline-indicator';
      indicator.innerHTML = `
        <div class="pwa-indicator-content">
          <i class="fas fa-wifi-slash me-2"></i>
          <span>您当前处于离线状态</span>
          <button id="pwa-retry-btn" class="ml-2 btn btn-sm btn-outline-light">重试</button>
        </div>
      `;
      
      const style = document.createElement('style');
      style.textContent = `
        #pwa-offline-indicator {
          position: fixed;
          top: 0;
          left: 0;
          right: 0;
          background: linear-gradient(135deg, #dc3545 0%, #c82333 100%);
          color: white;
          text-align: center;
          padding: 10px;
          font-size: 14px;
          z-index: 9999;
          transform: translateY(-100%);
          transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1);
          box-shadow: 0 2px 10px rgba(0,0,0,0.2);
        }
        
        #pwa-offline-indicator.show {
          transform: translateY(0);
        }
        
        #pwa-offline-indicator .pwa-indicator-content {
          display: flex;
          align-items: center;
          justify-content: center;
          gap: 8px;
        }
        
        #pwa-retry-btn {
          padding: 4px 12px;
          border-radius: 12px;
          font-size: 12px;
        }
        
        #pwa-offline-indicator.online {
          background: linear-gradient(135deg, #198754 0%, #15803d 100%);
        }
        
        #pwa-offline-indicator.online i {
          content: '\\f1eb';
        }
      `;
      document.head.appendChild(style);
      
      document.body.appendChild(indicator);
      
      document.getElementById('pwa-retry-btn').addEventListener('click', () => {
        this.forceRefresh();
      });
      
      this.updateOfflineIndicator();
    },

    updateOfflineIndicator: function() {
      const indicator = document.getElementById('pwa-offline-indicator');
      if (!indicator) return;
      
      if (isOnline) {
        indicator.classList.remove('show');
        setTimeout(() => {
          indicator.classList.remove('online');
        }, 300);
      } else {
        indicator.classList.add('show');
      }
    },

    showInstallButton: function() {
      if (document.getElementById('pwa-install-button')) {
        return;
      }

      const button = document.createElement('button');
      button.id = 'pwa-install-button';
      button.innerHTML = '<i class="fas fa-download me-2"></i>安装应用';
      
      const style = document.createElement('style');
      style.textContent = `
        #pwa-install-button {
          position: fixed;
          bottom: 20px;
          right: 20px;
          background: linear-gradient(135deg, #0d6efd 0%, #0b5ed7 100%);
          color: white;
          border: none;
          padding: 14px 28px;
          border-radius: 30px;
          font-size: 14px;
          font-weight: 500;
          cursor: pointer;
          box-shadow: 0 4px 20px rgba(13, 110, 253, 0.4);
          z-index: 9998;
          transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
          display: flex;
          align-items: center;
          gap: 8px;
        }
        
        #pwa-install-button:hover {
          transform: translateY(-2px);
          box-shadow: 0 6px 25px rgba(13, 110, 253, 0.5);
        }
        
        #pwa-install-button:active {
          transform: scale(0.98);
        }
        
        #pwa-install-button.installing {
          background: linear-gradient(135deg, #6c757d 0%, #5c636a 100%);
          pointer-events: none;
        }
        
        @media (max-width: 576px) {
          #pwa-install-button {
            bottom: calc(env(safe-area-inset-bottom) + 20px);
            left: 20px;
            right: 20px;
            justify-content: center;
          }
        }
      `;
      document.head.appendChild(style);
      
      button.addEventListener('click', () => {
        this.promptInstall();
      });
      
      document.body.appendChild(button);
    },

    hideInstallButton: function() {
      const button = document.getElementById('pwa-install-button');
      if (button) {
        button.style.opacity = '0';
        button.style.transform = 'translateY(20px)';
        setTimeout(() => button.remove(), 300);
      }
    },

    promptInstall: async function() {
      if (!deferredPrompt) {
        console.warn('[PWA] 没有可用的安装提示');
        return;
      }

      const button = document.getElementById('pwa-install-button');
      if (button) {
        button.classList.add('installing');
        button.innerHTML = '<i class="fas fa-spinner fa-spin me-2"></i>安装中...';
      }

      deferredPrompt.prompt();
      
      const { outcome } = await deferredPrompt.userChoice;
      console.log('[PWA] 用户选择:', outcome);
      deferredPrompt = null;
      
      if (button) {
        button.classList.remove('installing');
        button.innerHTML = '<i class="fas fa-download me-2"></i>安装应用';
      }
      
      if (outcome === 'accepted') {
        this.hideInstallButton();
      }
    },

    showInstallSuccessNotification: function() {
      if (!('Notification' in window)) return;
      
      if (notificationPermission === 'granted') {
        new Notification('墨盾验证', {
          body: '应用已成功安装到您的设备',
          icon: '/static/icons/icon-192x192.png',
          badge: '/static/icons/badge-72x72.png',
          data: { url: '/' }
        });
      } else {
        this.showToast('应用已成功安装', 'success');
      }
    },

    showUpdateNotification: function() {
      if (document.getElementById('pwa-update-notification')) {
        return;
      }

      const notification = document.createElement('div');
      notification.id = 'pwa-update-notification';
      notification.innerHTML = `
        <div class="pwa-update-content">
          <div class="pwa-update-icon">
            <i class="fas fa-refresh"></i>
          </div>
          <div class="pwa-update-text">
            <strong>新版本可用!</strong>
            <p class="mb-0 small">点击刷新以获取最新内容</p>
          </div>
          <div class="pwa-update-actions">
            <button id="pwa-refresh-btn" class="btn btn-primary btn-sm">刷新</button>
            <button id="pwa-defer-btn" class="btn btn-outline-secondary btn-sm ms-2">稍后</button>
          </div>
        </div>
      `;
      
      const style = document.createElement('style');
      style.textContent = `
        #pwa-update-notification {
          position: fixed;
          bottom: 80px;
          left: 20px;
          right: 20px;
          background: white;
          border-radius: 12px;
          box-shadow: 0 4px 20px rgba(0,0,0,0.15);
          padding: 16px;
          z-index: 9999;
          animation: slideUp 0.3s cubic-bezier(0.4, 0, 0.2, 1);
          max-width: 400px;
        }
        
        @keyframes slideUp {
          from {
            transform: translateY(100%);
            opacity: 0;
          }
          to {
            transform: translateY(0);
            opacity: 1;
          }
        }
        
        .pwa-update-content {
          display: flex;
          align-items: center;
          gap: 12px;
        }
        
        .pwa-update-icon {
          width: 40px;
          height: 40px;
          background: linear-gradient(135deg, #198754 0%, #15803d 100%);
          border-radius: 10px;
          display: flex;
          align-items: center;
          justify-content: center;
          color: white;
          font-size: 18px;
        }
        
        .pwa-update-text {
          flex: 1;
        }
        
        .pwa-update-text strong {
          color: #1f2937;
          font-size: 15px;
        }
        
        .pwa-update-text p {
          color: #6b7280;
        }
        
        .pwa-update-actions {
          display: flex;
          gap: 8px;
        }
        
        @media (max-width: 576px) {
          #pwa-update-notification {
            bottom: calc(env(safe-area-inset-bottom) + 80px);
          }
        }
      `;
      document.head.appendChild(style);
      
      document.body.appendChild(notification);
      
      document.getElementById('pwa-refresh-btn').addEventListener('click', () => {
        this.activateUpdate();
      });
      
      document.getElementById('pwa-defer-btn').addEventListener('click', () => {
        notification.style.opacity = '0';
        notification.style.transform = 'translateY(20px)';
        setTimeout(() => notification.remove(), 300);
      });
    },

    activateUpdate: function() {
      if (registration && registration.waiting) {
        registration.waiting.postMessage({ type: 'SKIP_WAITING' });
        
        navigator.serviceWorker.addEventListener('controllerchange', () => {
          window.location.reload();
        });
      } else {
        window.location.reload();
      }
    },

    forceRefresh: function() {
      if (registration) {
        registration.update();
      }
      window.location.reload();
    },

    setupNotificationPermission: async function() {
      if (!('Notification' in window)) {
        console.warn('[PWA] 浏览器不支持通知');
        return;
      }

      notificationPermission = await Notification.requestPermission();
      console.log('[PWA] 通知权限:', notificationPermission);
    },

    setupPeriodicUpdateCheck: function() {
      setInterval(() => {
        if (registration) {
          registration.update();
        }
      }, PWA_CONFIG.updateCheckInterval);
    },

    setupBadgeSupport: function() {
      if (!('setAppBadge' in navigator)) {
        console.warn('[PWA] 浏览器不支持应用徽章');
        return;
      }

      this.setBadge(0);
    },

    setBadge: function(count) {
      if ('setAppBadge' in navigator) {
        navigator.setAppBadge(count).catch(() => {});
      }
    },

    clearBadge: function() {
      if ('clearAppBadge' in navigator) {
        navigator.clearAppBadge().catch(() => {});
      }
    },

    setupShareTarget: function() {
      if (!('share' in navigator)) {
        console.warn('[PWA] 浏览器不支持 Web Share API');
        return;
      }

      const shareButton = document.createElement('button');
      shareButton.id = 'pwa-share-button';
      shareButton.innerHTML = '<i class="fas fa-share-alt"></i>';
      shareButton.style.cssText = `
        position: fixed;
        bottom: 20px;
        left: 20px;
        width: 56px;
        height: 56px;
        background: linear-gradient(135deg, #0d6efd 0%, #0b5ed7 100%);
        color: white;
        border: none;
        border-radius: 50%;
        cursor: pointer;
        box-shadow: 0 4px 15px rgba(13, 110, 253, 0.4);
        z-index: 9997;
        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
        font-size: 18px;
        display: flex;
        align-items: center;
        justify-content: center;
      `;
      
      shareButton.addEventListener('click', () => {
        this.shareApp();
      });
      
      document.body.appendChild(shareButton);
    },

    shareApp: async function() {
      if (!('share' in navigator)) return;

      try {
        await navigator.share({
          title: '墨盾验证',
          text: '新一代智能行为验证系统',
          url: window.location.href
        });
      } catch (error) {
        console.log('[PWA] 分享取消或失败:', error);
      }
    },

    storeOfflineVerificationData: function(expectedData) {
      if (registration && registration.active) {
        registration.active.postMessage({
          type: 'STORE_OFFLINE_DATA',
          expected: expectedData,
          timestamp: Date.now()
        });
        console.log('[PWA] 已发送离线验证数据到 Service Worker');
      }
    },

    syncPendingData: async function() {
      if (!registration) return;

      try {
        await registration.sync.register('sync-verification');
        console.log('[PWA] 后台同步已注册');
      } catch (error) {
        console.error('[PWA] 后台同步注册失败:', error);
      }
    },

    subscribeToPush: async function() {
      if (!('PushManager' in window)) {
        console.warn('[PWA] 浏览器不支持推送通知');
        return null;
      }

      try {
        const subscription = await registration.pushManager.subscribe({
          userVisibleOnly: true,
          applicationServerKey: this.urlBase64ToUint8Array('YOUR_VAPID_PUBLIC_KEY')
        });
        
        await this.sendSubscriptionToServer(subscription);
        console.log('[PWA] 已订阅推送通知');
        return subscription;
      } catch (error) {
        console.error('[PWA] 推送订阅失败:', error);
        return null;
      }
    },

    sendSubscriptionToServer: async function(subscription) {
      try {
        await fetch('/api/v1/push/subscribe', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(subscription)
        });
      } catch (error) {
        console.error('[PWA] 发送订阅信息失败:', error);
      }
    },

    urlBase64ToUint8Array: function(base64String) {
      const padding = '='.repeat((4 - base64String.length % 4) % 4);
      const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');
      const rawData = window.atob(base64);
      return Uint8Array.from([...rawData].map(char => char.charCodeAt(0)));
    },

    showToast: function(message, type = 'info') {
      const toast = document.createElement('div');
      toast.className = `pwa-toast pwa-toast-${type}`;
      toast.textContent = message;
      
      const style = document.createElement('style');
      style.textContent = `
        .pwa-toast {
          position: fixed;
          bottom: 100px;
          left: 50%;
          transform: translateX(-50%);
          padding: 12px 24px;
          border-radius: 8px;
          color: white;
          font-size: 14px;
          z-index: 10000;
          animation: toastFadeIn 0.3s ease;
        }
        
        @keyframes toastFadeIn {
          from { opacity: 0; transform: translateX(-50%) translateY(20px); }
          to { opacity: 1; transform: translateX(-50%) translateY(0); }
        }
        
        .pwa-toast-success {
          background: linear-gradient(135deg, #198754 0%, #15803d 100%);
        }
        
        .pwa-toast-warning {
          background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
        }
        
        .pwa-toast-error {
          background: linear-gradient(135deg, #dc3545 0%, #c82333 100%);
        }
        
        .pwa-toast-info {
          background: linear-gradient(135deg, #0d6efd 0%, #0b5ed7 100%);
        }
      `;
      document.head.appendChild(style);
      
      document.body.appendChild(toast);
      
      setTimeout(() => {
        toast.style.opacity = '0';
        toast.style.transform = 'translateX(-50%) translateY(20px)';
        setTimeout(() => toast.remove(), 300);
      }, 3000);
    },

    isOffline: function() {
      return !isOnline;
    },

    getAppVersion: async function() {
      try {
        const response = await fetch('/static/manifest.json');
        const manifest = await response.json();
        return manifest.version || '1.0.0';
      } catch {
        return '1.0.0';
      }
    },

    getCacheStatus: async function() {
      if (!('caches' in window)) return null;
      
      try {
        const cacheNames = await caches.keys();
        const cacheInfo = [];
        
        for (const name of cacheNames) {
          const cache = await caches.open(name);
          const keys = await cache.keys();
          cacheInfo.push({
            name: name,
            entries: keys.length
          });
        }
        
        return cacheInfo;
      } catch (error) {
        console.error('[PWA] 获取缓存状态失败:', error);
        return null;
      }
    },

    clearCache: async function() {
      if (!('caches' in window)) return;
      
      try {
        const cacheNames = await caches.keys();
        await Promise.all(cacheNames.map(name => caches.delete(name)));
        console.log('[PWA] 缓存已清理');
        
        if (registration) {
          registration.active?.postMessage({ type: 'CLEAR_CACHE' });
        }
        
        return true;
      } catch (error) {
        console.error('[PWA] 清理缓存失败:', error);
        return false;
      }
    }
  };

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => PWA.init());
  } else {
    PWA.init();
  }

  window.PWA = PWA;
})();