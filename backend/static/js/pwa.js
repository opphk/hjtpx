(function() {
  'use strict';

  const PWA_CONFIG = {
    serviceWorkerPath: '/static/service-worker.js',
    manifestPath: '/static/manifest.json',
    offlineDetectionInterval: 1000
  };

  let isOnline = navigator.onLine;
  let registration = null;
  let deferredPrompt = null;

  const PWA = {
    init: function() {
      console.log('[PWA] 初始化 PWA 模块');
      this.registerServiceWorker();
      this.setupOnlineOfflineEvents();
      this.setupInstallPrompt();
      this.injectOfflineIndicator();
    },

    registerServiceWorker: async function() {
      if (!('serviceWorker' in navigator)) {
        console.warn('[PWA] 浏览器不支持 Service Worker');
        return;
      }

      try {
        registration = await navigator.serviceWorker.register(PWA_CONFIG.serviceWorkerPath);
        console.log('[PWA] Service Worker 注册成功:', registration.scope);

        registration.addEventListener('updatefound', () => {
          console.log('[PWA] 发现 Service Worker 更新');
          const newWorker = registration.installing;
          newWorker.addEventListener('statechange', () => {
            if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
              console.log('[PWA] 新内容可用，请刷新页面');
              this.showUpdateNotification();
            }
          });
        });
      } catch (error) {
        console.error('[PWA] Service Worker 注册失败:', error);
      }
    },

    setupOnlineOfflineEvents: function() {
      window.addEventListener('online', () => {
        isOnline = true;
        this.updateOfflineIndicator();
        console.log('[PWA] 网络已恢复');
      });

      window.addEventListener('offline', () => {
        isOnline = false;
        this.updateOfflineIndicator();
        console.log('[PWA] 网络已断开');
      });
    },

    setupInstallPrompt: function() {
      window.addEventListener('beforeinstallprompt', (e) => {
        console.log('[PWA] 收到安装提示事件');
        e.preventDefault();
        deferredPrompt = e;
        this.showInstallButton();
      });

      window.addEventListener('appinstalled', () => {
        console.log('[PWA] 应用已安装');
        deferredPrompt = null;
      });
    },

    injectOfflineIndicator: function() {
      const indicator = document.createElement('div');
      indicator.id = 'pwa-offline-indicator';
      indicator.style.cssText = `
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        background: #dc3545;
        color: white;
        text-align: center;
        padding: 8px;
        font-size: 14px;
        z-index: 9999;
        transform: translateY(-100%);
        transition: transform 0.3s ease;
      `;
      indicator.innerHTML = `
        <i class="fas fa-wifi-slash me-2"></i>
        您当前处于离线状态
      `;
      document.body.appendChild(indicator);
      
      this.updateOfflineIndicator();
    },

    updateOfflineIndicator: function() {
      const indicator = document.getElementById('pwa-offline-indicator');
      if (!indicator) return;
      
      indicator.style.transform = isOnline ? 'translateY(-100%)' : 'translateY(0)';
    },

    showInstallButton: function() {
      const button = document.createElement('button');
      button.id = 'pwa-install-button';
      button.innerHTML = '<i class="fas fa-download me-2"></i>安装应用';
      button.style.cssText = `
        position: fixed;
        bottom: 20px;
        right: 20px;
        background: #0d6efd;
        color: white;
        border: none;
        padding: 12px 24px;
        border-radius: 30px;
        font-size: 14px;
        cursor: pointer;
        box-shadow: 0 4px 12px rgba(13, 110, 253, 0.3);
        z-index: 9998;
        transition: all 0.3s ease;
      `;
      
      button.addEventListener('click', () => {
        this.promptInstall();
      });
      
      button.addEventListener('mouseenter', () => {
        button.style.transform = 'scale(1.05)';
      });
      
      button.addEventListener('mouseleave', () => {
        button.style.transform = 'scale(1)';
      });
      
      document.body.appendChild(button);
    },

    promptInstall: async function() {
      if (!deferredPrompt) {
        console.warn('[PWA] 没有可用的安装提示');
        return;
      }

      deferredPrompt.prompt();
      const { outcome } = await deferredPrompt.userChoice;
      console.log('[PWA] 用户选择:', outcome);
      deferredPrompt = null;
      
      const button = document.getElementById('pwa-install-button');
      if (button) {
        button.remove();
      }
    },

    showUpdateNotification: function() {
      const notification = document.createElement('div');
      notification.style.cssText = `
        position: fixed;
        bottom: 20px;
        left: 20px;
        background: #198754;
        color: white;
        padding: 16px 24px;
        border-radius: 8px;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
        z-index: 9999;
      `;
      notification.innerHTML = `
        <div class="d-flex align-items-center gap-3">
          <div>
            <strong>新版本可用!</strong>
            <p class="mb-0 small">点击刷新以获取最新内容</p>
          </div>
          <button id="pwa-refresh-btn" class="btn btn-light btn-sm">刷新</button>
          <button id="pwa-close-btn" class="btn btn-outline-light btn-sm">×</button>
        </div>
      `;
      
      document.body.appendChild(notification);
      
      document.getElementById('pwa-refresh-btn').addEventListener('click', () => {
        window.location.reload();
      });
      
      document.getElementById('pwa-close-btn').addEventListener('click', () => {
        notification.remove();
      });
    },

    storeOfflineVerificationData: function(expectedData) {
      if (registration && registration.active) {
        registration.active.postMessage({
          type: 'STORE_OFFLINE_DATA',
          expected: expectedData
        });
        console.log('[PWA] 已发送离线验证数据到 Service Worker');
      }
    },

    isOffline: function() {
      return !isOnline;
    }
  };

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => PWA.init());
  } else {
    PWA.init();
  }

  window.PWA = PWA;
})();
