const CACHE_VERSION = 'v2.2.0';
const CACHE_NAME = `hjtpx-${CACHE_VERSION}`;
const STATIC_CACHE = `hjtpx-static-${CACHE_VERSION}`;
const DYNAMIC_CACHE = `hjtpx-dynamic-${CACHE_VERSION}`;
const IMAGE_CACHE = `hjtpx-images-${CACHE_VERSION}`;
const API_CACHE = `hjtpx-api-${CACHE_VERSION}`;

const STATIC_ASSETS = [
  '/',
  '/index.html',
  '/offline.html',
  '/manifest.json',
  '/favicon.png'
];

const CACHE_STRATEGIES = {
  cacheFirst: 'cache-first',
  networkFirst: 'network-first',
  staleWhileRevalidate: 'stale-while-revalidate',
  networkOnly: 'network-only'
};

const MAX_CACHE_AGE = {
  static: 7 * 24 * 60 * 60 * 1000,
  dynamic: 24 * 60 * 60 * 1000,
  images: 30 * 24 * 60 * 60 * 1000,
  api: 5 * 60 * 1000
};

const OFFLINE_URL = '/offline.html';
const API_TIMEOUT = 5000;

self.addEventListener('install', (event) => {
  console.log('[SW] Installing service worker...', CACHE_VERSION);
  
  event.waitUntil(
    caches.open(STATIC_CACHE)
      .then((cache) => {
        console.log('[SW] Caching static assets');
        return cache.addAll(STATIC_ASSETS);
      })
      .then(() => self.skipWaiting())
  );
});

self.addEventListener('activate', (event) => {
  console.log('[SW] Activating service worker...');
  
  event.waitUntil(
    caches.keys()
      .then((keys) => {
        return Promise.all(
          keys
            .filter((key) => !key.includes(CACHE_VERSION))
            .map((key) => {
              console.log('[SW] Deleting old cache:', key);
              return caches.delete(key);
            })
        );
      })
      .then(() => {
        return self.clients.claim();
      })
      .then(() => {
        console.log('[SW] Service worker activated');
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
    if (isHTMLRequest(request)) {
      event.respondWith(networkFirst(request, STATIC_CACHE, '/offline.html'));
    } else if (isStaticAsset(url.pathname)) {
      event.respondWith(cacheFirst(request, STATIC_CACHE));
    } else if (isAPIRequest(url.pathname)) {
      event.respondWith(networkFirst(request, API_CACHE));
    } else if (isImageRequest(request)) {
      event.respondWith(staleWhileRevalidate(request, IMAGE_CACHE));
    } else {
      event.respondWith(networkFirst(request, DYNAMIC_CACHE));
    }
  } else if (isCDNRequest(url)) {
    event.respondWith(staleWhileRevalidate(request, DYNAMIC_CACHE));
  }
});

function isHTMLRequest(request) {
  return request.destination === 'document';
}

function isStaticAsset(pathname) {
  const staticExtensions = ['.js', '.css', '.woff', '.woff2', '.ttf', '.eot', '.svg', '.png', '.jpg', '.jpeg', '.gif', '.webp', '.avif', '.ico'];
  return staticExtensions.some(ext => pathname.endsWith(ext)) || pathname.includes('/assets/');
}

function isAPIRequest(pathname) {
  return pathname.startsWith('/api/') || pathname.startsWith('/socket.io/');
}

function isImageRequest(request) {
  return request.destination === 'image';
}

function isCDNRequest(url) {
  return url.hostname.includes('unpkg.com') || 
         url.hostname.includes('jsdelivr.net') || 
         url.hostname.includes('cdnjs.cloudflare.com') ||
         url.hostname.includes('cdn.');
}

async function cacheFirst(request, cacheName) {
  const cache = await caches.open(cacheName);
  const cachedResponse = await cache.match(request);
  
  if (cachedResponse) {
    const cacheDate = cachedResponse.headers.get('sw-cache-date');
    if (cacheDate) {
      const age = Date.now() - parseInt(cacheDate, 10);
      if (age > MAX_CACHE_AGE.static) {
        cache.delete(request);
        try {
          const networkResponse = await fetch(request);
          
          if (networkResponse.ok) {
            const headers = new Headers(networkResponse.headers);
            headers.append('sw-cache-date', Date.now().toString());
            const responseToCache = new Response(await networkResponse.clone().blob(), {
              status: networkResponse.status,
              statusText: networkResponse.statusText,
              headers: headers
            });
            cache.put(request, responseToCache);
          }
          
          return networkResponse;
        } catch (error) {
          console.error('[SW] Cache first failed:', error);
          return cachedResponse;
        }
      }
    }
    return cachedResponse;
  }

  try {
    const networkResponse = await fetch(request);
    
    if (networkResponse.ok) {
      const headers = new Headers(networkResponse.headers);
      headers.append('sw-cache-date', Date.now().toString());
      const responseToCache = new Response(await networkResponse.clone().blob(), {
        status: networkResponse.status,
        statusText: networkResponse.statusText,
        headers: headers
      });
      cache.put(request, responseToCache);
    }
    
    return networkResponse;
  } catch (error) {
    console.error('[SW] Cache first failed:', error);
    return caches.match('/offline.html');
  }
}

async function networkFirst(request, cacheName, fallback = null) {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), API_TIMEOUT);
  
  try {
    const networkResponse = await fetch(request, { signal: controller.signal });
    clearTimeout(timeoutId);
    
    if (networkResponse.ok) {
      const cache = await caches.open(cacheName);
      const headers = new Headers(networkResponse.headers);
      headers.append('sw-cache-date', Date.now().toString());
      const responseToCache = new Response(await networkResponse.clone().blob(), {
        status: networkResponse.status,
        statusText: networkResponse.statusText,
        headers: headers
      });
      cache.put(request, responseToCache);
    }
    
    return networkResponse;
  } catch (error) {
    clearTimeout(timeoutId);
    const cachedResponse = await caches.match(request);
    if (cachedResponse) {
      return cachedResponse;
    }
    if (fallback) {
      return caches.match(fallback);
    }
    return createOfflineResponse();
  }
}

async function staleWhileRevalidate(request, cacheName) {
  const cache = await caches.open(cacheName);
  const cachedResponse = await cache.match(request);

  const fetchPromise = fetch(request)
    .then((networkResponse) => {
      if (networkResponse.ok) {
        const headers = new Headers(networkResponse.headers);
        headers.append('sw-cache-date', Date.now().toString());
        const responseToCache = new Response(await networkResponse.clone().blob(), {
          status: networkResponse.status,
          statusText: networkResponse.statusText,
          headers: headers
        });
        cache.put(request, responseToCache);
      }
      return networkResponse;
    })
    .catch((error) => {
      console.warn('[SW] Stale while revalidate fetch failed:', error);
      return null;
    });

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
        message: '您当前处于离线状态，请检查网络连接'
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
  
  if (event.tag === 'sync-notifications') {
    event.waitUntil(syncNotifications());
  } else if (event.tag === 'sync-user-data') {
    event.waitUntil(syncUserData());
  } else if (event.tag.startsWith('sync-')) {
    event.waitUntil(syncData(event.tag));
  }
});

async function syncNotifications() {
  try {
    console.log('[SW] Syncing notifications...');
    const response = await fetch('/api/v1/notifications/unread', {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json'
      }
    });
    
    if (response.ok) {
      const data = await response.json();
      if (data.notifications && data.notifications.length > 0) {
        self.registration.showNotification('HJTPX 系统', {
          body: `您有 ${data.notifications.length} 条未读通知`,
          icon: '/favicon.png',
          badge: '/favicon.png',
          tag: 'sync-notification',
          vibrate: [100, 50, 100]
        });
      }
    }
  } catch (error) {
    console.error('[SW] Notification sync failed:', error);
  }
}

async function syncUserData() {
  try {
    console.log('[SW] Syncing user data...');
    const db = await openDatabase();
    const tx = db.transaction('pendingRequests', 'readonly');
    const store = tx.objectStore('pendingRequests');
    const requests = await getAllFromStore(store);

    for (const request of requests) {
      try {
        const response = await fetch(request.url, {
          method: request.method,
          headers: request.headers,
          body: request.body
        });
        
        if (response.ok) {
          const deleteTx = db.transaction('pendingRequests', 'readwrite');
          deleteTx.objectStore('pendingRequests').delete(request.id);
        }
      } catch (error) {
        console.error('[SW] Sync failed for request:', request.id, error);
      }
    }
  } catch (error) {
    console.error('[SW] Sync error:', error);
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

self.addEventListener('push', (event) => {
  console.log('[SW] Push notification received');
  
  let data = {
    title: 'HJTPX 系统',
    body: '您有一条新通知',
    icon: '/favicon.png',
    badge: '/favicon.png'
  };

  if (event.data) {
    try {
      data = { ...data, ...event.data.json() };
    } catch (e) {
      data.body = event.data.text();
    }
  }

  const options = {
    body: data.body,
    icon: data.icon || '/favicon.png',
    badge: data.badge || '/favicon.png',
    tag: data.tag || 'default-' + Date.now(),
    data: data.data || {},
    vibrate: data.vibrate || [100, 50, 100],
    renotify: data.renotify !== false,
    requireInteraction: data.requireInteraction || false,
    actions: data.actions || [],
    dir: data.dir || 'ltr',
    lang: data.lang || 'zh-CN'
  };

  event.waitUntil(
    self.registration.showNotification(data.title, options)
  );
});

self.addEventListener('notificationclick', (event) => {
  console.log('[SW] Notification clicked');
  
  event.notification.close();

  const notificationData = event.notification.data || {};
  const targetUrl = notificationData.url || '/';
  const action = event.action;

  event.waitUntil(
    clients.matchAll({ type: 'window', includeUncontrolled: true })
      .then(async (clientList) => {
        if (action && notificationData.actionUrls && notificationData.actionUrls[action]) {
          const actionUrl = notificationData.actionUrls[action];
          
          for (const client of clientList) {
            if (client.url === actionUrl && 'focus' in client) {
              await client.focus();
              client.postMessage({
                action: 'notificationAction',
                type: action,
                notification: notificationData
              });
              return;
            }
          }
          
          if (clients.openWindow) {
            const newClient = await clients.openWindow(actionUrl);
            if (newClient) {
              newClient.postMessage({
                action: 'notificationAction',
                type: action,
                notification: notificationData
              });
            }
            return;
          }
        }

        for (const client of clientList) {
          if ('focus' in client) {
            await client.focus();
            client.postMessage({
              action: 'notificationClick',
              notification: notificationData
            });
            return;
          }
        }
        
        if (clients.openWindow) {
          const newClient = await clients.openWindow(targetUrl);
          if (newClient) {
            newClient.postMessage({
              action: 'notificationClick',
              notification: notificationData
            });
          }
        }
      })
  );
});

self.addEventListener('notificationclose', (event) => {
  console.log('[SW] Notification closed');
  
  if (event.notification.data && event.notification.data.id) {
    console.log('[SW] Notification ID:', event.notification.data.id);
  }
});

self.addEventListener('periodicsync', (event) => {
  if (event.tag === 'sync-content') {
    event.waitUntil(syncContent());
  }
});

async function syncContent() {
  try {
    console.log('[SW] Periodic sync content...');
    const cache = await caches.open(DYNAMIC_CACHE);
    const criticalRoutes = ['/', '/dashboard', '/api/v1/user/profile'];
    
    for (const route of criticalRoutes) {
      try {
        const response = await fetch(route);
        if (response.ok) {
          await cache.put(route, response.clone());
          console.log('[SW] Periodically synced:', route);
        }
      } catch (error) {
        console.error('[SW] Periodic sync failed for:', route, error);
      }
    }
    
    notifyClientsAboutSync(true);
  } catch (error) {
    console.error('[SW] Periodic sync error:', error);
    notifyClientsAboutSync(false);
  }
}

function notifyClientsAboutSync(success) {
  clients.matchAll({ type: 'window', includeUncontrolled: true })
    .then((clientList) => {
      clientList.forEach((client) => {
        client.postMessage({
          action: 'periodicSyncComplete',
          success: success,
          timestamp: Date.now()
        });
      });
    });
}

self.addEventListener('message', (event) => {
  console.log('[SW] Message received:', event.data);
  
  if (event.data.action === 'skipWaiting') {
    self.skipWaiting();
  }
  
  if (event.data.action === 'clearCache') {
    event.waitUntil(
      caches.keys()
        .then((keys) => Promise.all(keys.map((key) => caches.delete(key))))
        .then(() => event.source.postMessage({ action: 'cacheCleared', success: true }))
    );
  }
  
  if (event.data.action === 'getCacheStatus') {
    event.waitUntil(
      caches.keys()
        .then(async (keys) => {
          const stats = await Promise.all(
            keys.map(async (key) => {
              const cache = await caches.open(key);
              const cacheKeys = await cache.keys();
              return { name: key, count: cacheKeys.length };
            })
          );
          return stats;
        })
        .then((stats) => {
          event.source.postMessage({ action: 'cacheStatus', stats });
        })
    );
  }
  
  if (event.data.action === 'registerSync') {
    event.waitUntil(
      self.registration.sync.register(event.data.tag || 'sync-data')
        .then(() => {
          event.source.postMessage({ action: 'syncRegistered', success: true });
        })
        .catch((error) => {
          event.source.postMessage({ action: 'syncFailed', error: error.message });
        })
    );
  }
  
  if (event.data.action === 'showNotification') {
    event.waitUntil(
      self.registration.showNotification(event.data.title, {
        body: event.data.body,
        icon: event.data.icon || '/favicon.png',
        badge: event.data.badge || '/favicon.png',
        tag: event.data.tag || 'custom-' + Date.now(),
        data: event.data.data || {},
        vibrate: event.data.vibrate || [100, 50, 100],
        requireInteraction: event.data.requireInteraction || false,
        actions: event.data.actions || []
      })
    );
  }
  
  if (event.data.action === 'getVersion') {
    event.source.postMessage({
      action: 'version',
      version: CACHE_VERSION
    });
  }
  
  if (event.data.action === 'prefetch') {
    event.waitUntil(
      prefetchResources(event.data.urls)
        .then(() => {
          event.source.postMessage({ action: 'prefetchComplete', success: true });
        })
        .catch((error) => {
          event.source.postMessage({ action: 'prefetchFailed', error: error.message });
        })
    );
  }
  
  if (event.data.action === 'updateAvailable') {
    event.waitUntil(
      clients.matchAll({ type: 'window', includeUncontrolled: true })
        .then((clientList) => {
          clientList.forEach((client) => {
            client.postMessage({
              action: 'updateAvailable',
              version: CACHE_VERSION
            });
          });
        })
    );
  }
});

async function prefetchResources(urls) {
  const cache = await caches.open(STATIC_CACHE);
  
  for (const url of urls) {
    try {
      const response = await fetch(url);
      if (response.ok) {
        await cache.put(url, response);
        console.log('[SW] Prefetched:', url);
      }
    } catch (error) {
      console.error('[SW] Prefetch failed for:', url, error);
    }
  }
}
