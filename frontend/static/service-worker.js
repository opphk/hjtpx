const CACHE_NAME = 'hjtpx-captcha-v1';
const STATIC_CACHE_NAME = 'hjtpx-static-v1';
const DYNAMIC_CACHE_NAME = 'hjtpx-dynamic-v1';
const IMAGE_CACHE_NAME = 'hjtpx-images-v1';

const STATIC_ASSETS = [
  '/',
  '/captcha.html',
  '/static/js/main.js',
  '/static/js/captcha.js',
  '/static/css/captcha-ui-unified.css',
  '/static/manifest.json',
  '/offline.html',
];

const CACHE_STRATEGIES = {
  staleWhileRevalidate: 'stale-while-revalidate',
  cacheFirst: 'cache-first',
  networkFirst: 'network-first',
  cacheOnly: 'cache-only',
  networkOnly: 'network-only',
};

self.addEventListener('install', (event) => {
  console.log('[ServiceWorker] Installing...');

  event.waitUntil(
    caches.open(STATIC_CACHE_NAME)
      .then((cache) => {
        console.log('[ServiceWorker] Caching static assets');
        return cache.addAll(STATIC_ASSETS);
      })
      .then(() => {
        console.log('[ServiceWorker] Skip waiting');
        return self.skipWaiting();
      })
      .catch((error) => {
        console.error('[ServiceWorker] Cache failed:', error);
      })
  );
});

self.addEventListener('activate', (event) => {
  console.log('[ServiceWorker] Activating...');

  event.waitUntil(
    caches.keys()
      .then((cacheNames) => {
        return Promise.all(
          cacheNames
            .filter((cacheName) => {
              return cacheName.startsWith('hjtpx-') &&
                     cacheName !== STATIC_CACHE_NAME &&
                     cacheName !== DYNAMIC_CACHE_NAME &&
                     cacheName !== IMAGE_CACHE_NAME;
            })
            .map((cacheName) => {
              console.log('[ServiceWorker] Deleting old cache:', cacheName);
              return caches.delete(cacheName);
            })
        );
      })
      .then(() => {
        console.log('[ServiceWorker] Claiming clients');
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

  if (url.pathname.startsWith('/api/')) {
    event.respondWith(networkFirstStrategy(request));
  } else if (url.pathname.match(/\.(jpg|png|gif|webp|svg)$/)) {
    event.respondWith(staleWhileRevalidateStrategy(request, IMAGE_CACHE_NAME));
  } else if (url.pathname.match(/\.(js|css)$/)) {
    event.respondWith(staleWhileRevalidateStrategy(request, STATIC_CACHE_NAME));
  } else if (url.pathname.match(/\.(woff|woff2|ttf|otf)$/)) {
    event.respondWith(cacheFirstStrategy(request, STATIC_CACHE_NAME));
  } else {
    event.respondWith(networkFirstStrategy(request));
  }
});

async function cacheFirstStrategy(request, cacheName) {
  const cache = await caches.open(cacheName);
  const cachedResponse = await cache.match(request);

  if (cachedResponse) {
    return cachedResponse;
  }

  try {
    const networkResponse = await fetch(request);
    if (networkResponse.ok) {
      cache.put(request, networkResponse.clone());
    }
    return networkResponse;
  } catch (error) {
    console.error('[ServiceWorker] Fetch failed:', error);
    return new Response('Offline', { status: 503 });
  }
}

async function networkFirstStrategy(request) {
  try {
    const networkResponse = await fetch(request);

    if (networkResponse.ok) {
      const cache = await caches.open(DYNAMIC_CACHE_NAME);
      cache.put(request, networkResponse.clone());
    }

    return networkResponse;
  } catch (error) {
    console.error('[ServiceWorker] Network failed:', error);
    const cachedResponse = await caches.match(request);

    if (cachedResponse) {
      return cachedResponse;
    }

    if (request.mode === 'navigate') {
      return caches.match('/offline.html');
    }

    return new Response('Offline', { status: 503 });
  }
}

async function staleWhileRevalidateStrategy(request, cacheName) {
  const cache = await caches.open(cacheName);
  const cachedResponse = await cache.match(request);

  const fetchPromise = fetch(request)
    .then((networkResponse) => {
      if (networkResponse.ok) {
        cache.put(request, networkResponse.clone());
      }
      return networkResponse;
    })
    .catch(() => {
      return null;
    });

  return cachedResponse || fetchPromise;
}

self.addEventListener('sync', (event) => {
  console.log('[ServiceWorker] Background sync:', event.tag);

  if (event.tag === 'sync-captcha-data') {
    event.waitUntil(syncCaptchaData());
  }
});

async function syncCaptchaData() {
  try {
    const clients = await self.clients.matchAll();
    clients.forEach((client) => {
      client.postMessage({
        type: 'SYNC_COMPLETE',
        timestamp: Date.now(),
      });
    });
  } catch (error) {
    console.error('[ServiceWorker] Sync failed:', error);
  }
}

self.addEventListener('message', (event) => {
  console.log('[ServiceWorker] Message received:', event.data);

  if (event.data && event.data.type === 'SKIP_WAITING') {
    self.skipWaiting();
  }

  if (event.data && event.data.type === 'CLEAR_CACHE') {
    event.waitUntil(
      caches.keys().then((cacheNames) => {
        return Promise.all(
          cacheNames.map((cacheName) => caches.delete(cacheName))
        );
      })
    );
  }

  if (event.data && event.data.type === 'CACHE_URLS') {
    event.waitUntil(
      caches.open(DYNAMIC_CACHE_NAME).then((cache) => {
        return cache.addAll(event.data.urls);
      })
    );
  }

  if (event.data && event.data.type === 'GET_VERSION') {
    event.ports[0].postMessage({
      version: CACHE_NAME,
    });
  }
});

self.addEventListener('push', (event) => {
  console.log('[ServiceWorker] Push received');

  let data = {
    title: '验证码通知',
    body: '您有新的验证请求',
    icon: '/static/icons/icon-192x192.png',
    badge: '/static/icons/badge-72x72.png',
    tag: 'captcha-notification',
  };

  if (event.data) {
    try {
      data = { ...data, ...event.data.json() };
    } catch (error) {
      console.error('[ServiceWorker] Push data parse failed:', error);
    }
  }

  const options = {
    body: data.body,
    icon: data.icon,
    badge: data.badge,
    tag: data.tag,
    vibrate: [100, 50, 100],
    data: {
      dateOfArrival: Date.now(),
      primaryKey: data.tag,
    },
    actions: [
      { action: 'verify', title: '立即验证' },
      { action: 'dismiss', title: '忽略' },
    ],
  };

  event.waitUntil(
    self.registration.showNotification(data.title, options)
  );
});

self.addEventListener('notificationclick', (event) => {
  console.log('[ServiceWorker] Notification click:', event.action);
  event.notification.close();

  if (event.action === 'verify') {
    event.waitUntil(
      clients.openWindow('/captcha.html')
    );
  }
});

self.addEventListener('notificationclose', (event) => {
  console.log('[ServiceWorker] Notification closed');
});

self.addEventListener('error', (error) => {
  console.error('[ServiceWorker] Error:', error);
});

self.addEventListener('unhandledrejection', (event) => {
  console.error('[ServiceWorker] Unhandled promise rejection:', event.reason);
});
