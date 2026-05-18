const CACHE_NAME = 'hjtpx-v1.0.0';
const ASSETS_TO_CACHE = [
  '/',
  '/captcha',
  '/static/manifest.json',
  '/static/js/pwa.js',
  '/static/js/i18n.js',
  '/static/js/theme.js',
  '/static/js/captcha.js',
  '/static/js/environment-detector.js'
];

const CDN_ASSETS = [
  'https://cdn.bootcdn.net/ajax/libs/twitter-bootstrap/5.3.8/css/bootstrap.min.css',
  'https://cdn.bootcdn.net/ajax/libs/font-awesome/6.5.1/css/all.min.css',
  'https://cdn.bootcdn.net/ajax/libs/twitter-bootstrap/5.3.8/js/bootstrap.bundle.min.js'
];

const CAPTCHA_CACHE = 'hjtpx-captcha-v1';
const OFFLINE_VERIFICATION = 'hjtpx-offline-verification-v1';

self.addEventListener('install', (event) => {
  console.log('[ServiceWorker] 安装 Service Worker');
  event.waitUntil(
    Promise.all([
      caches.open(CACHE_NAME).then((cache) => {
        console.log('[ServiceWorker] 缓存应用资源');
        return cache.addAll(ASSETS_TO_CACHE).catch((err) => {
          console.warn('[ServiceWorker] 部分资源缓存失败，但继续安装:', err);
        });
      }),
      caches.open(CAPTCHA_CACHE).then((cache) => {
        console.log('[ServiceWorker] 准备验证码缓存空间');
        return Promise.resolve();
      }),
      caches.open(OFFLINE_VERIFICATION).then((cache) => {
        console.log('[ServiceWorker] 准备离线验证缓存空间');
        return Promise.resolve();
      })
    ]).then(() => {
      console.log('[ServiceWorker] 所有缓存初始化完成');
      return self.skipWaiting();
    })
  );
});

self.addEventListener('activate', (event) => {
  console.log('[ServiceWorker] 激活 Service Worker');
  event.waitUntil(
    caches.keys().then((cacheNames) => {
      return Promise.all(
        cacheNames.map((cacheName) => {
          if (cacheName !== CACHE_NAME && cacheName !== CAPTCHA_CACHE && cacheName !== OFFLINE_VERIFICATION) {
            console.log('[ServiceWorker] 删除旧缓存:', cacheName);
            return caches.delete(cacheName);
          }
        })
      );
    }).then(() => self.clients.claim())
  );
});

self.addEventListener('fetch', (event) => {
  const request = event.request;
  
  if (request.method !== 'GET') {
    return;
  }

  if (request.url.includes('/api/captcha') || request.url.includes('/api/verify')) {
    event.respondWith(handleCaptchaRequest(request));
    return;
  }

  if (CDN_ASSETS.some(url => request.url === url)) {
    event.respondWith(handleCDNRequest(request));
    return;
  }

  event.respondWith(handleDefaultRequest(request));
});

async function handleCaptchaRequest(request) {
  try {
    const networkResponse = await fetch(request);
    if (networkResponse.ok && request.url.includes('/api/captcha')) {
      const cache = await caches.open(CAPTCHA_CACHE);
      cache.put(request, networkResponse.clone());
    }
    return networkResponse;
  } catch (error) {
    console.log('[ServiceWorker] 网络请求失败，尝试使用缓存:', error);
    
    if (request.url.includes('/api/verify')) {
      return handleOfflineVerification(request);
    }
    
    const cache = await caches.open(CAPTCHA_CACHE);
    const cachedResponse = await cache.match(request);
    if (cachedResponse) {
      return cachedResponse;
    }
    
    return new Response(JSON.stringify({
      success: false,
      error: '无法连接到服务器，且无离线验证码可用'
    }), {
      headers: { 'Content-Type': 'application/json' },
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
    console.log('[ServiceWorker] CDN 请求失败，使用缓存:', error);
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
    console.log('[ServiceWorker] 请求失败，使用缓存:', error);
    const cache = await caches.open(CACHE_NAME);
    const cachedResponse = await cache.match(request);
    if (cachedResponse) {
      return cachedResponse;
    }
    return new Response('您当前离线，且页面未缓存', {
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
          offline: true
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
  
  return false;
}

self.addEventListener('message', (event) => {
  if (event.data && event.data.type === 'STORE_OFFLINE_DATA') {
    console.log('[ServiceWorker] 存储离线验证数据');
    caches.open(OFFLINE_VERIFICATION).then((cache) => {
      cache.put('offline-verification-data', new Response(JSON.stringify({
        expected: event.data.expected,
        timestamp: Date.now()
      }), {
        headers: { 'Content-Type': 'application/json' }
      }));
    });
  }
  
  if (event.data && event.data.type === 'SKIP_WAITING') {
    self.skipWaiting();
  }
});
