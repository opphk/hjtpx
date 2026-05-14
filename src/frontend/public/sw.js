const CACHE_VERSION = 'v2';
const STATIC_CACHE = `hjtpx-static-${CACHE_VERSION}`;
const DYNAMIC_CACHE = `hjtpx-dynamic-${CACHE_VERSION}`;
const IMAGE_CACHE = `hjtpx-images-${CACHE_VERSION}`;
const API_CACHE = `hjtpx-api-${CACHE_VERSION}`;

const STATIC_ASSETS = [
  '/',
  '/index.html',
  '/manifest.json',
  '/offline.html'
];

const CACHE_STRATEGIES = {
  cacheFirst: 'cache-first',
  networkFirst: 'network-first',
  staleWhileRevalidate: 'stale-while-revalidate',
  networkOnly: 'network-only',
  cacheOnly: 'cache-only'
};

const OFFLINE_PAGE = '/offline.html';

self.addEventListener('install', (event) => {
  console.log('[SW] Installing service worker...');

  event.waitUntil(
    caches.open(STATIC_CACHE)
      .then((cache) => {
        console.log('[SW] Caching static assets');
        return cache.addAll(STATIC_ASSETS);
      })
      .then(() => {
        console.log('[SW] Skip waiting to activate immediately');
        return self.skipWaiting();
      })
  );
});

self.addEventListener('activate', (event) => {
  console.log('[SW] Activating service worker...');

  event.waitUntil(
    caches.keys()
      .then((keys) => {
        return Promise.all(
          keys
            .filter((key) =>
              key.startsWith('hjtpx-') &&
              key !== STATIC_CACHE &&
              key !== DYNAMIC_CACHE &&
              key !== IMAGE_CACHE &&
              key !== API_CACHE
            )
            .map((key) => {
              console.log('[SW] Deleting old cache:', key);
              return caches.delete(key);
            })
        );
      })
      .then(() => {
        console.log('[SW] Claiming clients');
        return self.clients.claim();
      })
  );
});

self.addEventListener('fetch', (event) => {
  const { request } = event;
  const url = new URL(request.url);

  if (request.method !== 'GET') {
    return;
  }

  if (url.origin === location.origin) {
    if (isStaticAsset(url.pathname)) {
      event.respondWith(cacheFirst(request, STATIC_CACHE));
    } else if (isAPIRequest(url.pathname)) {
      if (isCriticalAPI(url.pathname)) {
        event.respondWith(networkFirst(request, API_CACHE));
      } else {
        event.respondWith(staleWhileRevalidate(request, API_CACHE));
      }
    } else if (isImageRequest(request)) {
      event.respondWith(staleWhileRevalidate(request, IMAGE_CACHE));
    } else if (isFontRequest(request)) {
      event.respondWith(cacheFirst(request, STATIC_CACHE));
    } else {
      event.respondWith(networkFirst(request, DYNAMIC_CACHE));
    }
  } else {
    if (isExternalImage(url)) {
      event.respondWith(staleWhileRevalidate(request, IMAGE_CACHE));
    } else {
      event.respondWith(networkFirst(request, DYNAMIC_CACHE));
    }
  }
});

function isStaticAsset(pathname) {
  const staticExtensions = [
    '.js',
    '.jsx',
    '.css',
    '.scss',
    '.woff',
    '.woff2',
    '.ttf',
    '.eot',
    '.svg',
    '.png',
    '.jpg',
    '.jpeg',
    '.gif',
    '.ico',
    '.webp',
    '.avif'
  ];
  return staticExtensions.some(ext => pathname.endsWith(ext));
}

function isAPIRequest(pathname) {
  return pathname.startsWith('/api/');
}

function isCriticalAPI(pathname) {
  const criticalEndpoints = [
    '/api/v1/auth/me',
    '/api/v1/users/profile',
    '/api/v1/health',
    '/api/v1/notifications'
  ];
  return criticalEndpoints.some(endpoint => pathname.startsWith(endpoint));
}

function isImageRequest(request) {
  return request.destination === 'image';
}

function isFontRequest(request) {
  const url = new URL(request.url);
  return url.pathname.match(/\.(woff|woff2|ttf|eot)$/);
}

function isExternalImage(url) {
  const imageHosts = ['images.unsplash.com', 'picsum.photos', 'via.placeholder.com'];
  return imageHosts.some(host => url.hostname.includes(host));
}

async function cacheFirst(request, cacheName) {
  const cachedResponse = await caches.match(request);

  if (cachedResponse) {
    return cachedResponse;
  }

  try {
    const networkResponse = await fetch(request);

    if (networkResponse.ok) {
      const cache = await caches.open(cacheName);
      cache.put(request, networkResponse.clone());
    }

    return networkResponse;
  } catch (error) {
    return caches.match(OFFLINE_PAGE);
  }
}

async function networkFirst(request, cacheName) {
  try {
    const networkResponse = await fetch(request);

    if (networkResponse.ok) {
      const cache = await caches.open(cacheName);
      cache.put(request, networkResponse.clone());
    }

    return networkResponse;
  } catch (error) {
    const cachedResponse = await caches.match(request);
    return cachedResponse || createOfflineResponse();
  }
}

async function staleWhileRevalidate(request, cacheName) {
  const cache = await caches.open(cacheName);
  const cachedResponse = await cache.match(request);

  const fetchPromise = fetch(request)
    .then((networkResponse) => {
      if (networkResponse.ok) {
        cache.put(request, networkResponse.clone());
      }
      return networkResponse;
    })
    .catch(() => null);

  return cachedResponse || fetchPromise || createOfflineResponse();
}

function networkOnly(request) {
  return fetch(request);
}

function createOfflineResponse() {
  return new Response(
    JSON.stringify({
      success: false,
      error: {
        code: 'OFFLINE',
        message: 'You are currently offline. Please check your internet connection.',
        suggestion: 'Try again when you have a stable internet connection.'
      }
    }),
    {
      status: 503,
      headers: { 'Content-Type': 'application/json' }
    }
  );
}

self.addEventListener('sync', (event) => {
  console.log('[SW] Background sync:', event.tag);

  if (event.tag === 'sync-data') {
    event.waitUntil(syncData());
  }

  if (event.tag === 'sync-notifications') {
    event.waitUntil(syncNotifications());
  }

  if (event.tag === 'sync-analytics') {
    event.waitUntil(syncAnalytics());
  }
});

async function syncData() {
  try {
    const db = await openDatabase();
    const tx = db.transaction('pendingRequests', 'readonly');
    const store = tx.objectStore('pendingRequests');
    const requests = await getAllFromStore(store);

    for (const request of requests) {
      try {
        await fetch(request.url, {
          method: request.method,
          headers: request.headers,
          body: request.body
        });

        const deleteTx = db.transaction('pendingRequests', 'readwrite');
        deleteTx.objectStore('pendingRequests').delete(request.id);
      } catch (error) {
        console.error('[SW] Sync failed for request:', request.id, error);
      }
    }
  } catch (error) {
    console.error('[SW] Sync error:', error);
  }
}

async function syncNotifications() {
  try {
    const response = await fetch('/api/v1/notifications/sync', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        lastSync: localStorage.getItem('lastNotificationSync')
      })
    });

    if (response.ok) {
      const data = await response.json();
      localStorage.setItem('lastNotificationSync', new Date().toISOString());

      self.registration.showNotification(data.title, {
        body: data.body,
        icon: '/icon-192.png',
        badge: '/badge-72.png',
        tag: 'sync-notification'
      });
    }
  } catch (error) {
    console.error('[SW] Notification sync failed:', error);
  }
}

async function syncAnalytics() {
  try {
    const pendingAnalytics = await getPendingAnalytics();

    if (pendingAnalytics.length > 0) {
      await fetch('/api/v1/analytics/batch', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ events: pendingAnalytics })
      });

      await clearPendingAnalytics();
    }
  } catch (error) {
    console.error('[SW] Analytics sync failed:', error);
  }
}

function openDatabase() {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open('hjtpx-sync', 1);
    request.onerror = () => reject(request.error);
    request.onsuccess = () => resolve(request.result);
    request.onupgradeneeded = (event) => {
      const db = event.target.result;
      if (!db.objectStoreNames.contains('pendingRequests')) {
        db.createObjectStore('pendingRequests', { keyPath: 'id', autoIncrement: true });
      }
      if (!db.objectStoreNames.contains('pendingAnalytics')) {
        db.createObjectStore('pendingAnalytics', { keyPath: 'id', autoIncrement: true });
      }
    };
  });
}

function getAllFromStore(store) {
  return new Promise((resolve, reject) => {
    const request = store.getAll();
    request.onerror = () => reject(request.error);
    request.onsuccess = () => resolve(request.result);
  });
}

async function getPendingAnalytics() {
  const db = await openDatabase();
  const tx = db.transaction('pendingAnalytics', 'readonly');
  const store = tx.objectStore('pendingAnalytics');
  return getAllFromStore(store);
}

async function clearPendingAnalytics() {
  const db = await openDatabase();
  const tx = db.transaction('pendingAnalytics', 'readwrite');
  const store = tx.objectStore('pendingAnalytics');
  store.clear();
}

self.addEventListener('push', (event) => {
  console.log('[SW] Push notification received');

  let data = {
    title: 'HJTPX',
    body: 'You have a new notification',
    icon: '/icon-192.png',
    badge: '/badge-72.png',
    tag: 'default',
    data: {}
  };

  if (event.data) {
    try {
      const payload = event.data.json();
      data = { ...data, ...payload };
    } catch (e) {
      data.body = event.data.text();
    }
  }

  const options = {
    body: data.body,
    icon: data.icon || '/icon-192.png',
    badge: data.badge || '/badge-72.png',
    tag: data.tag || 'default',
    data: data.data || {},
    vibrate: [100, 50, 100],
    actions: data.actions || [],
    requireInteraction: data.requireInteraction || false,
    silent: data.silent || false
  };

  event.waitUntil(
    self.registration.showNotification(data.title, options)
  );
});

self.addEventListener('notificationclick', (event) => {
  console.log('[SW] Notification clicked');

  event.notification.close();

  const notificationData = event.notification.data || {};
  const urlToOpen = notificationData.url || '/';

  event.waitUntil(
    clients.matchAll({ type: 'window', includeUncontrolled: true })
      .then((clientList) => {
        for (const client of clientList) {
          if (client.url === urlToOpen && 'focus' in client) {
            return client.focus();
          }
          if (client.url.includes(urlToOpen) && 'focus' in client) {
            return client.focus();
          }
        }

        if (clients.openWindow) {
          return clients.openWindow(urlToOpen);
        }
      })
  );
});

self.addEventListener('notificationclose', (event) => {
  console.log('[SW] Notification closed');

  const notificationData = event.notification.data || {};

  if (notificationData.trackingId) {
    fetch('/api/v1/analytics/notification', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        trackingId: notificationData.trackingId,
        action: 'dismissed',
        timestamp: new Date().toISOString()
      })
    }).catch(() => {});
  }
});

self.addEventListener('message', (event) => {
  console.log('[SW] Message received:', event.data);

  if (event.data.action === 'skipWaiting') {
    self.skipWaiting();
  }

  if (event.data.action === 'clearCache') {
    event.waitUntil(
      caches.keys()
        .then((keys) => Promise.all(keys.map((key) => caches.delete(key))))
        .then(() => ({ success: true }))
        .then((result) => event.source.postMessage(result))
    );
  }

  if (event.data.action === 'getCacheStatus') {
    event.waitUntil(
      getCacheStatus()
        .then((stats) => event.source.postMessage({ action: 'cacheStatus', stats }))
    );
  }

  if (event.data.action === 'prefetch') {
    event.waitUntil(
      prefetchAssets(event.data.urls)
        .then(() => event.source.postMessage({ action: 'prefetchComplete' }))
        .catch((error) => event.source.postMessage({ action: 'prefetchFailed', error: error.message }))
    );
  }

  if (event.data.action === 'requestNotificationPermission') {
    requestNotificationPermission()
      .then((result) => event.source.postMessage({ action: 'notificationPermissionResult', ...result }))
      .catch((error) => event.source.postMessage({ action: 'notificationPermissionError', error: error.message }));
  }
});

async function getCacheStatus() {
  const cacheNames = await caches.keys();
  const stats = {};

  for (const name of cacheNames) {
    if (name.startsWith('hjtpx-')) {
      const cache = await caches.open(name);
      const keys = await cache.keys();
      stats[name] = {
        count: keys.length,
        urls: keys.map(k => k.url).slice(0, 10)
      };
    }
  }

  return stats;
}

async function prefetchAssets(urls) {
  const cache = await caches.open(STATIC_CACHE);

  for (const url of urls) {
    try {
      const response = await fetch(url);
      if (response.ok) {
        await cache.put(url, response);
      }
    } catch (error) {
      console.error('[SW] Prefetch failed for:', url, error);
    }
  }
}

async function requestNotificationPermission() {
  try {
    const permission = await self.registration.pushManager.permissionState({
      userVisibleOnly: true,
      applicationServerKey: urlBase64ToUint8Array(self.registration.vapidPublicKey || '')
    });

    if (permission === 'granted') {
      return { granted: true, permission };
    } else {
      return { granted: false, permission, error: 'Push permission not granted' };
    }
  } catch (error) {
    return { granted: false, error: error.message };
  }
}

function urlBase64ToUint8Array(base64String) {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
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

self.addEventListener('periodicsync', (event) => {
  if (event.tag === 'sync-content') {
    event.waitUntil(syncContent());
  }
});

async function syncContent() {
  console.log('[SW] Periodic sync triggered');

  try {
    const response = await fetch('/api/v1/content/updates', {
      headers: {
        'Cache-Control': 'no-cache'
      }
    });

    if (response.ok) {
      const data = await response.json();

      if (data.updates) {
        const cache = await caches.open(DYNAMIC_CACHE);
        for (const update of data.updates) {
          const updateResponse = new Response(JSON.stringify(update.data), {
            headers: { 'Content-Type': 'application/json' }
          });
          await cache.put(update.url, updateResponse);
        }
      }
    }
  } catch (error) {
    console.error('[SW] Content sync failed:', error);
  }
}
