const CACHE_NAME = 'hjtpx-v1.0.0';
const STATIC_CACHE = 'hjtpx-static-v1';
const DYNAMIC_CACHE = 'hjtpx-dynamic-v1';
const IMAGE_CACHE = 'hjtpx-images-v1';

const STATIC_ASSETS = [
  '/',
  '/index.html',
  '/manifest.json',
  '/favicon.png',
  '/static/js/main.js',
  '/static/css/main.css'
];

const API_CACHE_NAME = 'hjtpx-api-v1';
const OFFLINE_API_PATHS = [
  '/api/v1/health',
  '/api/v1/users',
  '/api/v1/notifications'
];

self.addEventListener('install', (event) => {
  console.log('[SW] Installing Service Worker...', event);
  
  event.waitUntil(
    Promise.all([
      caches.open(STATIC_CACHE).then((cache) => {
        console.log('[SW] Precaching static assets');
        return cache.addAll(STATIC_ASSETS).catch((error) => {
          console.warn('[SW] Failed to cache some static assets:', error);
        });
      }),
      caches.open(DYNAMIC_CACHE).then((cache) => {
        console.log('[SW] Dynamic cache initialized');
        return cache;
      }),
      caches.open(IMAGE_CACHE).then((cache) => {
        console.log('[SW] Image cache initialized');
        return cache;
      }),
      caches.open(API_CACHE_NAME).then((cache) => {
        console.log('[SW] API cache initialized');
        return cache;
      })
    ]).then(() => {
      console.log('[SW] Skip waiting');
      return self.skipWaiting();
    })
  );
});

self.addEventListener('activate', (event) => {
  console.log('[SW] Activating Service Worker...', event);
  
  event.waitUntil(
    caches.keys().then((cacheNames) => {
      return Promise.all(
        cacheNames
          .filter((cacheName) => {
            return (
              cacheName.startsWith('hjtpx-') &&
              ![
                STATIC_CACHE,
                DYNAMIC_CACHE,
                IMAGE_CACHE,
                API_CACHE_NAME
              ].includes(cacheName)
            );
          })
          .map((cacheName) => {
            console.log('[SW] Deleting old cache:', cacheName);
            return caches.delete(cacheName);
          })
      );
    }).then(() => {
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
    if (STATIC_ASSETS.some(asset => request.url.includes(asset))) {
      event.respondWith(cacheFirst(request, STATIC_CACHE));
      return;
    }

    if (isImageRequest(request)) {
      event.respondWith(staleWhileRevalidate(request, IMAGE_CACHE));
      return;
    }

    if (url.pathname.startsWith('/api/')) {
      event.respondWith(networkFirst(request, API_CACHE_NAME));
      return;
    }

    event.respondWith(networkFirst(request, DYNAMIC_CACHE));
  } else {
    event.respondWith(networkFirst(request, DYNAMIC_CACHE));
  }
});

function isImageRequest(request) {
  const imageExtensions = ['.jpg', '.jpeg', '.png', '.gif', '.svg', '.webp', '.ico'];
  return imageExtensions.some(ext => request.url.includes(ext));
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
    console.error('[SW] Cache first fetch failed:', error);
    return caches.match('/offline.html');
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
    console.log('[SW] Network failed, trying cache:', request.url);
    const cachedResponse = await caches.match(request);
    if (cachedResponse) {
      return cachedResponse;
    }

    if (request.destination === 'document') {
      return caches.match('/offline.html');
    }

    return new Response('Offline content not available', {
      status: 503,
      statusText: 'Service Unavailable',
      headers: { 'Content-Type': 'text/plain' }
    });
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
    .catch((error) => {
      console.log('[SW] Stale while revalidate fetch failed:', error);
    });

  return cachedResponse || fetchPromise;
}

self.addEventListener('message', (event) => {
  console.log('[SW] Message received:', event.data);

  if (event.data && event.data.type === 'SKIP_WAITING') {
    self.skipWaiting();
  }

  if (event.data && event.data.type === 'CLEAR_CACHE') {
    event.waitUntil(
      caches.keys().then((cacheNames) => {
        return Promise.all(
          cacheNames
            .filter(name => name.startsWith('hjtpx-'))
            .map(name => caches.delete(name))
        );
      }).then(() => {
        console.log('[SW] All caches cleared');
        event.ports[0].postMessage({ success: true });
      })
    );
  }

  if (event.data && event.data.type === 'GET_CACHE_SIZE') {
    event.waitUntil(
      getCacheSize().then(size => {
        event.ports[0].postMessage({ cacheSize: size });
      })
    );
  }

  if (event.data && event.data.type === 'CACHE_URLS') {
    const { urls } = event.data;
    event.waitUntil(
      cacheUrls(urls).then(() => {
        event.ports[0].postMessage({ success: true });
      })
    );
  }
});

async function getCacheSize() {
  let totalSize = 0;
  const cacheNames = await caches.keys();

  for (const cacheName of cacheNames) {
    if (cacheName.startsWith('hjtpx-')) {
      const cache = await caches.open(cacheName);
      const keys = await cache.keys();

      for (const request of keys) {
        const response = await cache.match(request);
        if (response) {
          const blob = await response.clone().blob();
          totalSize += blob.size;
        }
      }
    }
  }

  return totalSize;
}

async function cacheUrls(urls) {
  const cache = await caches.open(DYNAMIC_CACHE);
  for (const url of urls) {
    try {
      const response = await fetch(url);
      if (response.ok) {
        await cache.put(url, response);
      }
    } catch (error) {
      console.error(`[SW] Failed to cache URL: ${url}`, error);
    }
  }
}

self.addEventListener('push', (event) => {
  console.log('[SW] Push notification received:', event);

  let data = {
    title: 'HJTPX 通知',
    body: '您有一条新消息',
    icon: '/favicon.png',
    badge: '/favicon.png',
    tag: 'default',
    renotify: true,
    requireInteraction: false,
    data: {}
  };

  try {
    if (event.data) {
      const payload = event.data.json();
      data = {
        ...data,
        ...payload
      };
    }
  } catch (error) {
    console.error('[SW] Error parsing push data:', error);
  }

  const options = {
    body: data.body,
    icon: data.icon,
    badge: data.badge,
    tag: data.tag,
    renotify: data.renotify,
    requireInteraction: data.requireInteraction,
    data: data.data,
    vibrate: [200, 100, 200],
    actions: [
      {
        action: 'view',
        title: '查看',
        icon: '/favicon.png'
      },
      {
        action: 'dismiss',
        title: '忽略',
        icon: '/favicon.png'
      }
    ]
  };

  event.waitUntil(
    self.registration.showNotification(data.title, options)
  );
});

self.addEventListener('notificationclick', (event) => {
  console.log('[SW] Notification clicked:', event);
  event.notification.close();

  if (event.action === 'dismiss') {
    return;
  }

  event.waitUntil(
    clients.matchAll({ type: 'window', includeUncontrolled: true })
      .then((clientList) => {
        if (clientList.length > 0) {
          let client = clientList[0];
          for (let i = 0; i < clientList.length; i++) {
            if (clientList[i].focused) {
              client = clientList[i];
              break;
            }
          }
          return client.focus();
        }
        return clients.openWindow('/');
      })
      .then((client) => {
        if (client) {
          client.postMessage({
            type: 'NOTIFICATION_CLICK',
            data: event.notification.data
          });
        }
      })
  );
});

self.addEventListener('notificationclose', (event) => {
  console.log('[SW] Notification closed:', event);
});

self.addEventListener('sync', (event) => {
  console.log('[SW] Background sync:', event);

  if (event.tag === 'sync-data') {
    event.waitUntil(syncData());
  }

  if (event.tag === 'sync-notifications') {
    event.waitUntil(syncNotifications());
  }
});

async function syncData() {
  console.log('[SW] Syncing data...');
}

async function syncNotifications() {
  console.log('[SW] Syncing notifications...');
}

self.addEventListener('error', (event) => {
  console.error('[SW] Error:', event.error);
});

self.addEventListener('unhandledrejection', (event) => {
  console.error('[SW] Unhandled promise rejection:', event.reason);
});
