const CACHE_NAME = 'hjtpx-v2.1.0';
const CAPTCHA_CACHE = 'hjtpx-captcha-v2';
const OFFLINE_VERIFICATION = 'hjtpx-offline-v2';
const IMAGE_CACHE = 'hjtpx-images-v1';
const FONT_CACHE = 'hjtpx-fonts-v1';
const API_CACHE = 'hjtpx-api-v1';

const ASSETS_TO_CACHE = [
  '/',
  '/captcha',
  '/static/manifest.json',
  '/static/js/pwa.js',
  '/static/js/i18n.js',
  '/static/js/theme.js',
  '/static/js/captcha.js',
  '/static/js/main.js',
  '/static/js/mobile-optimization.js',
  '/static/js/performance-optimization.js',
  '/static/js/gesture-handler.js',
  '/static/js/trace.js',
  '/static/css/captcha-ui-optimized.css'
];

const CDN_ASSETS = [
  'https://cdn.bootcdn.net/ajax/libs/twitter-bootstrap/5.3.8/css/bootstrap.min.css',
  'https://cdn.bootcdn.net/ajax/libs/font-awesome/6.5.1/css/all.min.css',
  'https://cdn.bootcdn.net/ajax/libs/twitter-bootstrap/5.3.8/js/bootstrap.bundle.min.js'
];

const API_CACHE_RULES = [
  { pattern: /\/api\/v1\/captcha\/(slider|click|rotation)/, maxAge: 3600 },
  { pattern: /\/api\/v1\/health/, maxAge: 300 },
  { pattern: /\/api\/v1\/captcha\/verify/, maxAge: 0 }
];

self.addEventListener('install', (event) => {
  console.log('[ServiceWorker] 安装 Service Worker v2.1');
  event.waitUntil(
    Promise.all([
      caches.open(CACHE_NAME).then((cache) => {
        console.log('[ServiceWorker] 缓存核心资源');
        return cache.addAll(ASSETS_TO_CACHE).catch((err) => {
          console.warn('[ServiceWorker] 核心资源缓存失败，继续安装:', err);
        });
      }),
      caches.open(CAPTCHA_CACHE).then(() => {
        console.log('[ServiceWorker] 验证码缓存就绪');
        return Promise.resolve();
      }),
      caches.open(OFFLINE_VERIFICATION).then(() => {
        console.log('[ServiceWorker] 离线验证缓存就绪');
        return Promise.resolve();
      }),
      caches.open(IMAGE_CACHE).then(() => {
        console.log('[ServiceWorker] 图片缓存就绪');
        return Promise.resolve();
      }),
      caches.open(FONT_CACHE).then(() => {
        console.log('[ServiceWorker] 字体缓存就绪');
        return Promise.resolve();
      }),
      caches.open(API_CACHE).then(() => {
        console.log('[ServiceWorker] API缓存就绪');
        return Promise.resolve();
      })
    ]).then(() => {
      console.log('[ServiceWorker] 缓存初始化完成');
      return self.skipWaiting();
    })
  );
});

self.addEventListener('activate', (event) => {
  console.log('[ServiceWorker] 激活 Service Worker v2');
  event.waitUntil(
    caches.keys().then((cacheNames) => {
      return Promise.all(
        cacheNames.map((cacheName) => {
          const validCaches = [CACHE_NAME, CAPTCHA_CACHE, OFFLINE_VERIFICATION, IMAGE_CACHE];
          if (!validCaches.includes(cacheName)) {
            console.log('[ServiceWorker] 清理旧缓存:', cacheName);
            return caches.delete(cacheName);
          }
        }).filter(Boolean)
      );
    }).then(() => {
      console.log('[ServiceWorker] 缓存清理完成');
      return self.clients.claim();
    })
  );
});

self.addEventListener('fetch', (event) => {
  const request = event.request;

  if (request.method !== 'GET') {
    return;
  }

  if (isApiRequest(request.url)) {
    event.respondWith(handleApiRequest(request));
    return;
  }

  if (isImageRequest(request.url)) {
    event.respondWith(handleImageRequest(request));
    return;
  }

  if (CDN_ASSETS.some(url => request.url === url)) {
    event.respondWith(handleCDNRequest(request));
    return;
  }

  if (request.url.startsWith('http')) {
    event.respondWith(handleDefaultRequest(request));
  }
});

function isApiRequest(url) {
  return url.includes('/api/');
}

function isImageRequest(url) {
  return /\.(jpg|jpeg|png|gif|webp|svg)(\?|$)/.test(url);
}

async function handleApiRequest(request) {
  const cacheRule = API_CACHE_RULES.find(rule => rule.pattern.test(request.url));

  if (request.url.includes('/api/v1/captcha/verify')) {
    return handleVerificationRequest(request);
  }

  try {
    const networkResponse = await fetch(request);

    if (networkResponse.ok) {
      const cache = await caches.open(CAPTCHA_CACHE);
      cache.put(request, networkResponse.clone());

      if (cacheRule && cacheRule.maxAge > 0) {
        const headers = new Headers(networkResponse.headers);
        headers.set('Cache-Timestamp', Date.now().toString());
        headers.set('Cache-MaxAge', cacheRule.maxAge.toString());

        const responseWithHeaders = new Response(await networkResponse.clone().blob(), {
          status: networkResponse.status,
          statusText: networkResponse.statusText,
          headers: headers
        });
        cache.put(request, responseWithHeaders);
      }
    }

    return networkResponse;
  } catch (error) {
    console.log('[ServiceWorker] API请求失败，尝试缓存:', error);
    const cache = await caches.open(CAPTCHA_CACHE);
    const cachedResponse = await cache.match(request);

    if (cachedResponse) {
      const cacheTimestamp = parseInt(cachedResponse.headers.get('Cache-Timestamp') || '0');
      const maxAge = parseInt(cachedResponse.headers.get('Cache-MaxAge') || '3600') * 1000;

      if (Date.now() - cacheTimestamp < maxAge) {
        console.log('[ServiceWorker] 使用缓存的API响应');
        return cachedResponse;
      }
    }

    return new Response(JSON.stringify({
      success: false,
      error: '网络请求失败'
    }), {
      headers: { 'Content-Type': 'application/json' },
      status: 503
    });
  }
}

async function handleVerificationRequest(request) {
  try {
    const networkResponse = await fetch(request);
    return networkResponse;
  } catch (error) {
    console.log('[ServiceWorker] 验证请求失败:', error);
    return handleOfflineVerification(request);
  }
}

async function handleImageRequest(request) {
  try {
    const networkResponse = await fetch(request);

    if (networkResponse.ok) {
      const cache = await caches.open(IMAGE_CACHE);
      cache.put(request, networkResponse.clone());
    }

    return networkResponse;
  } catch (error) {
    const cache = await caches.open(IMAGE_CACHE);
    const cachedResponse = await cache.match(request);

    if (cachedResponse) {
      return cachedResponse;
    }

    return new Response('', {
      status: 503
    });
  }
}

async function handleCDNRequest(request) {
  try {
    const networkResponse = await fetch(request);
    if (networkResponse.ok) {
      const cache = await caches.open(CACHE_NAME);
      cache.put(request, networkResponse.clone());
    }
    return networkResponse;
  } catch (error) {
    const cache = await caches.open(CACHE_NAME);
    const cachedResponse = await cache.match(request);
    if (cachedResponse) {
      return cachedResponse;
    }
    return new Response('CDN resource unavailable offline', {
      status: 503
    });
  }
}

async function handleDefaultRequest(request) {
  try {
    const networkResponse = await fetch(request);
    if (networkResponse.ok) {
      const cache = await caches.open(CACHE_NAME);
      cache.put(request, networkResponse.clone());
    }
    return networkResponse;
  } catch (error) {
    const cache = await caches.open(CACHE_NAME);
    const cachedResponse = await cache.match(request);
    if (cachedResponse) {
      return cachedResponse;
    }
    return new Response('您当前离线', {
      status: 503,
      headers: { 'Content-Type': 'text/plain; charset=utf-8' }
    });
  }
}

async function handleOfflineVerification(request) {
  try {
    const cache = await caches.open(OFFLINE_VERIFICATION);
    const storedData = await cache.match('offline-verification-data');

    if (storedData) {
      const verificationData = await storedData.json();
      const requestClone = request.clone();
      const requestBody = await requestClone.json().catch(() => null);

      if (requestBody && verificationData.expected) {
        const isValid = verifyOffline(requestBody, verificationData.expected);
        return new Response(JSON.stringify({
          success: isValid,
          message: isValid ? '离线验证成功' : '离线验证失败',
          offline: true,
          verification_id: verificationData.verification_id
        }), {
          headers: { 'Content-Type': 'application/json' }
        });
      }
    }
  } catch (e) {
    console.error('[ServiceWorker] 离线验证错误:', e);
  }

  return new Response(JSON.stringify({
    success: false,
    error: '离线验证不可用',
    offline: true
  }), {
    headers: { 'Content-Type': 'application/json' },
    status: 503
  });
}

function verifyOffline(userInput, expected) {
  if (expected.type === 'slider') {
    const tolerance = expected.tolerance || 5;
    return Math.abs(userInput.x - expected.x) <= tolerance;
  }

  if (expected.type === 'click') {
    if (!Array.isArray(userInput.clicks) || !Array.isArray(expected.points)) {
      return false;
    }
    if (userInput.clicks.length !== expected.points.length) {
      return false;
    }

    const tolerance = expected.tolerance || 15;
    for (let i = 0; i < expected.points.length; i++) {
      const dx = Math.abs(userInput.clicks[i].x - expected.points[i].x);
      const dy = Math.abs(userInput.clicks[i].y - expected.points[i].y);
      if (dx > tolerance || dy > tolerance) {
        return false;
      }
    }
    return true;
  }

  if (expected.type === 'rotation') {
    const tolerance = expected.tolerance || 10;
    return Math.abs(userInput.angle - expected.angle) <= tolerance;
  }

  return false;
}

self.addEventListener('message', (event) => {
  if (event.data && event.data.type === 'STORE_OFFLINE_DATA') {
    console.log('[ServiceWorker] 存储离线验证数据');
    caches.open(OFFLINE_VERIFICATION).then((cache) => {
      cache.put('offline-verification-data', new Response(JSON.stringify({
        expected: event.data.expected,
        verification_id: event.data.verification_id || generateVerificationId(),
        timestamp: Date.now()
      }), {
        headers: { 'Content-Type': 'application/json' }
      }));
    });
  }

  if (event.data && event.data.type === 'CLEAR_CACHE') {
    console.log('[ServiceWorker] 清理缓存');
    caches.keys().then((cacheNames) => {
      return Promise.all(cacheNames.map((cacheName) => caches.delete(cacheName)));
    });
  }

  if (event.data && event.data.type === 'SKIP_WAITING') {
    self.skipWaiting();
  }

  if (event.data && event.data.type === 'GET_CACHE_STATUS') {
    caches.keys().then((cacheNames) => {
      event.source.postMessage({
        type: 'CACHE_STATUS',
        caches: cacheNames
      });
    });
  }
});

self.addEventListener('sync', (event) => {
  console.log('[ServiceWorker] 收到后台同步事件:', event.tag);

  if (event.tag === 'sync-verification') {
    event.waitUntil(syncOfflineVerifications());
  }

  if (event.tag === 'sync-failed-requests') {
    event.waitUntil(syncFailedRequests());
  }
});

async function syncOfflineVerifications() {
  try {
    console.log('[ServiceWorker] 同步离线验证数据');
    const cache = await caches.open(OFFLINE_VERIFICATION);
    const storedData = await cache.match('offline-verification-data');

    if (storedData) {
      const data = await storedData.json();
      const response = await fetch('/api/v1/captcha/verify', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data)
      });

      if (response.ok) {
        console.log('[ServiceWorker] 离线数据同步成功');
        await cache.delete('offline-verification-data');
      } else {
        console.log('[ServiceWorker] 离线数据同步失败，将重试');
      }
    }
  } catch (error) {
    console.error('[ServiceWorker] 离线数据同步失败:', error);
    throw error;
  }
}

async function syncFailedRequests() {
  try {
    console.log('[ServiceWorker] 同步失败请求');
    const cache = await caches.open(API_CACHE);
    const failedRequests = await cache.match('failed-requests');

    if (failedRequests) {
      const requests = await failedRequests.json();

      for (const requestData of requests) {
        try {
          const response = await fetch(requestData.url, {
            method: requestData.method,
            headers: requestData.headers,
            body: requestData.body
          });

          if (response.ok) {
            console.log('[ServiceWorker] 请求重试成功:', requestData.url);
          }
        } catch (error) {
          console.error('[ServiceWorker] 请求重试失败:', requestData.url, error);
        }
      }

      await cache.delete('failed-requests');
    }
  } catch (error) {
    console.error('[ServiceWorker] 同步失败请求错误:', error);
  }
}

function storeFailedRequest(request) {
  caches.open(API_CACHE).then(cache => {
    return cache.match('failed-requests').then(failedRequests => {
      let requests = [];

      if (failedRequests) {
        failedRequests.json().then(data => {
          requests = data;
        });
      }

      requests.push(request);

      return cache.put('failed-requests', new Response(JSON.stringify(requests), {
        headers: { 'Content-Type': 'application/json' }
      }));
    });
  });
}

self.addEventListener('push', (event) => {
  if (!event.data) {
    return;
  }

  try {
    const data = event.data.json();
    const options = {
      body: data.body || '验证码相关通知',
      icon: '/static/icons/icon-192x192.png',
      badge: '/static/icons/badge-72x72.png',
      vibrate: [100, 50, 100],
      data: {
        url: data.url || '/'
      },
      actions: [
        { action: 'open', title: '查看' },
        { action: 'close', title: '关闭' }
      ]
    };

    event.waitUntil(
      self.registration.showNotification(data.title || '墨盾验证', options)
    );
  } catch (error) {
    console.error('[ServiceWorker] 推送通知解析失败:', error);
  }
});

self.addEventListener('notificationclick', (event) => {
  event.notification.close();

  if (event.action === 'close') {
    return;
  }

  const url = event.notification.data?.url || '/';

  event.waitUntil(
    clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clientList) => {
      for (const client of clientList) {
        if (client.url === url && 'focus' in client) {
          return client.focus();
        }
      }
      if (clients.openWindow) {
        return clients.openWindow(url);
      }
    })
  );
});

self.addEventListener('notificationclose', (event) => {
  console.log('[ServiceWorker] 通知被关闭');
});

function generateVerificationId() {
  return 'offline_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
}
