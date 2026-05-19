const PWAManager = (function() {
    'use strict';

    class PWAManager {
        constructor() {
            this.registration = null;
            this.updateAvailable = false;
            this.swVersion = '1.0.0';
            this.isOnline = navigator.onLine;
            this.notificationsEnabled = false;

            this.init();
        }

        init() {
            if (typeof window === 'undefined' || !('serviceWorker' in navigator)) {
                console.warn('Service Worker not supported');
                return;
            }

            this.setupOnlineStatusListener();
            this.registerServiceWorker();
            this.setupMessageListener();
            this.checkNotificationPermission();
        }

        async registerServiceWorker() {
            try {
                this.registration = await navigator.serviceWorker.register('/service-worker.js', {
                    scope: '/',
                });

                console.log('Service Worker registered:', this.registration.scope);

                this.registration.addEventListener('updatefound', () => {
                    const newWorker = this.registration.installing;

                    newWorker.addEventListener('statechange', () => {
                        if (newWorker.state === 'installed') {
                            if (navigator.serviceWorker.controller) {
                                this.updateAvailable = true;
                                this.notifyUpdateAvailable();
                            } else {
                                console.log('Service Worker installed for the first time');
                            }
                        }
                    });
                });

                if (this.registration.active) {
                    console.log('Service Worker is active');
                }
            } catch (error) {
                console.error('Service Worker registration failed:', error);
            }
        }

        setupOnlineStatusListener() {
            window.addEventListener('online', () => {
                this.isOnline = true;
                this.notifyStatusChange(true);
                console.log('Network connection restored');
            });

            window.addEventListener('offline', () => {
                this.isOnline = false;
                this.notifyStatusChange(false);
                console.log('Network connection lost');
            });
        }

        setupMessageListener() {
            if ('serviceWorker' in navigator) {
                navigator.serviceWorker.addEventListener('message', (event) => {
                    this.handleServiceWorkerMessage(event.data);
                });
            }
        }

        handleServiceWorkerMessage(message) {
            switch (message.type) {
                case 'SYNC_COMPLETE':
                    console.log('Background sync complete');
                    this.emit('syncComplete', message);
                    break;

                case 'CACHE_UPDATED':
                    console.log('Cache updated:', message.url);
                    this.emit('cacheUpdated', message);
                    break;

                case 'UPDATE_AVAILABLE':
                    this.updateAvailable = true;
                    this.emit('updateAvailable', message);
                    break;

                default:
                    console.log('Unknown message type:', message.type);
            }
        }

        notifyUpdateAvailable() {
            this.emit('updateAvailable', {
                version: this.swVersion,
            });

            const shouldUpdate = confirm(
                '有新版本可用，是否立即更新？\n\n点击"确定"立即更新，或点击"取消"稍后更新。'
            );

            if (shouldUpdate) {
                this.applyUpdate();
            }
        }

        async applyUpdate() {
            if (!this.registration || !this.registration.waiting) {
                console.warn('No waiting Service Worker found');
                return;
            }

            this.registration.waiting.postMessage({ type: 'SKIP_WAITING' });

            this.registration.waiting.addEventListener('statechange', (event) => {
                if (event.target.state === 'activated') {
                    console.log('Service Worker updated');
                    window.location.reload();
                }
            });
        }

        setupOnlineStatusListener() {
            window.addEventListener('online', () => {
                this.isOnline = true;
                this.emit('online');
            });

            window.addEventListener('offline', () => {
                this.isOnline = false;
                this.emit('offline');
            });
        }

        async checkNotificationPermission() {
            if (!('Notification' in window)) {
                console.warn('Notifications not supported');
                return;
            }

            if (Notification.permission === 'granted') {
                this.notificationsEnabled = true;
            } else if (Notification.permission !== 'denied') {
                const permission = await Notification.requestPermission();
                this.notificationsEnabled = permission === 'granted';
            }
        }

        async enableNotifications() {
            if (!('Notification' in window)) {
                console.warn('Notifications not supported');
                return false;
            }

            if (Notification.permission === 'granted') {
                this.notificationsEnabled = true;
                return true;
            }

            if (Notification.permission !== 'denied') {
                const permission = await Notification.requestPermission();
                this.notificationsEnabled = permission === 'granted';
                return this.notificationsEnabled;
            }

            return false;
        }

        async sendNotification(title, options = {}) {
            if (!this.notificationsEnabled) {
                await this.enableNotifications();
            }

            if (!this.notificationsEnabled) {
                console.warn('Notifications not enabled');
                return null;
            }

            const defaultOptions = {
                icon: '/static/icons/icon-192x192.png',
                badge: '/static/icons/badge-72x72.png',
                vibrate: [100, 50, 100],
                requireInteraction: false,
                silent: false,
            };

            return new Notification(title, { ...defaultOptions, ...options });
        }

        async subscribeToPush() {
            if (!('PushManager' in window)) {
                console.warn('Push not supported');
                return null;
            }

            try {
                const subscription = await this.registration.pushManager.subscribe({
                    userVisibleOnly: true,
                    applicationServerKey: this.urlBase64ToUint8Array(
                        'BEl62iUYgUivxIkv69yViEuiBIa-Ib9-SkvMeAtA3LFgDzkrxZJjSgSnfckjBJuBkr3qBUYIHBQFLXYp5Nksh8U'
                    ),
                });

                console.log('Push subscription:', subscription);
                return subscription;
            } catch (error) {
                console.error('Push subscription failed:', error);
                return null;
            }
        }

        urlBase64ToUint8Array(base64String) {
            const padding = '='.repeat((4 - base64String.length % 4) % 4);
            const base64 = (base64String + padding)
                .replace(/-/g, '+')
                .replace(/_/g, '/');

            const rawData = window.atob(base64);
            const outputArray = new Uint8Array(rawData.length);

            for (let i = 0; i < rawData.length; ++i) {
                outputArray[i] = rawData.charCodeAt(i);
            }
            return outputArray;
        }

        async cacheUrls(urls) {
            if (!this.registration || !this.registration.active) {
                console.warn('Service Worker not ready');
                return false;
            }

            try {
                this.registration.active.postMessage({
                    type: 'CACHE_URLS',
                    urls: urls,
                });
                return true;
            } catch (error) {
                console.error('Cache URLs failed:', error);
                return false;
            }
        }

        async clearCache() {
            if (!this.registration || !this.registration.active) {
                console.warn('Service Worker not ready');
                return false;
            }

            try {
                this.registration.active.postMessage({ type: 'CLEAR_CACHE' });
                return true;
            } catch (error) {
                console.error('Clear cache failed:', error);
                return false;
            }
        }

        notifyStatusChange(isOnline) {
            this.emit(isOnline ? 'online' : 'offline');
        }

        getStatus() {
            return {
                isOnline: this.isOnline,
                updateAvailable: this.updateAvailable,
                notificationsEnabled: this.notificationsEnabled,
                swVersion: this.swVersion,
                registration: this.registration,
            };
        }

        addEventListener(event, callback) {
            if (!this.listeners) {
                this.listeners = {};
            }
            if (!this.listeners[event]) {
                this.listeners[event] = [];
            }
            this.listeners[event].push(callback);
        }

        removeEventListener(event, callback) {
            if (!this.listeners || !this.listeners[event]) {
                return;
            }
            const index = this.listeners[event].indexOf(callback);
            if (index > -1) {
                this.listeners[event].splice(index, 1);
            }
        }

        emit(event, data) {
            if (!this.listeners || !this.listeners[event]) {
                return;
            }
            this.listeners[event].forEach(callback => {
                try {
                    callback(data);
                } catch (error) {
                    console.error('Event handler error:', error);
                }
            });
        }

        destroy() {
            this.removeEventListener = () => {};
            this.emit = () => {};
            this.listeners = {};
        }
    }

    let instance = null;

    return {
        init: function() {
            if (!instance) {
                instance = new PWAManager();
            }
            return instance;
        },

        getInstance: function() {
            return instance;
        },

        getStatus: function() {
            return instance ? instance.getStatus() : null;
        },

        sendNotification: function(title, options) {
            return instance ? instance.sendNotification(title, options) : null;
        },

        cacheUrls: function(urls) {
            return instance ? instance.cacheUrls(urls) : Promise.resolve(false);
        },

        clearCache: function() {
            return instance ? instance.clearCache() : Promise.resolve(false);
        },
    };
})();

if (typeof module !== 'undefined' && module.exports) {
    module.exports = PWAManager;
}

if (typeof window !== 'undefined') {
    window.PWAManager = PWAManager;
}
