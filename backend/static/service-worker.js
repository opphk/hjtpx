// Service Worker for HJTPX v15.0 - Enhanced PWA Support
const CACHE_NAME = 'hjtpx-v15.0';
const OFFLINE_URL = '/offline.html';

const STATIC_CACHE = 'hjtpx-static-v15.0';
const DYNAMIC_CACHE = 'hjtpx-dynamic-v15.0';
const IMAGE_CACHE = 'hjtpx-images-v15.0';
const API_CACHE = 'hjtpx-api-v15.0';

const PRECACHE_ASSETS = [
    '/',
    '/offline.html',
    '/manifest.json',
    '/static/css/mobile.css',
    '/static/css/desktop.css',
    '/static/js/main.js',
    '/static/js/i18n.js',
    '/static/js/captcha.js',
    '/static/js/mobile-optimization.js',
    '/static/images/logo.png',
    '/static/images/icon-192.png',
    '/static/images/icon-512.png',
    '/static/images/splash.png',
    '/translations/zh-CN.json',
    '/translations/en-US.json',
    '/translations/ja-JP.json',
    '/translations/ko-KR.json'
];

const CACHE_STRATEGIES = {
    static: {
        pattern: /\.(js|css|png|jpg|jpeg|gif|svg|ico|woff|woff2|ttf|eot)$/,
        strategy: 'cacheFirst',
        cacheName: STATIC_CACHE,
        maxAge: 7 * 24 * 60 * 60 * 1000
    },
    images: {
        pattern: /\.(png|jpg|jpeg|gif|svg|webp|avif)$/,
        strategy: 'cacheFirst',
        cacheName: IMAGE_CACHE,
        maxAge: 30 * 24 * 60 * 60 * 1000
    },
    api: {
        pattern: /\/api\//,
        strategy: 'networkFirst',
        cacheName: API_CACHE,
        maxAge: 5 * 60 * 1000,
        networkTimeout: 3000
    },
    dynamic: {
        pattern: /.*/,
        strategy: 'staleWhileRevalidate',
        cacheName: DYNAMIC_CACHE,
        maxAge: 24 * 60 * 60 * 1000
    }
};

const SYNC_TAGS = {
    backgroundSync: 'hjtpx-sync',
    periodicSync: 'hjtpx-periodic-sync'
};

installEvent();
activateEvent();
fetchEvent();
syncEvent();
messageEvent();
periodicSyncEvent();

function installEvent() {
    self.addEventListener('install', (event) => {
        console.log('[ServiceWorker] Installing...');
        
        event.waitUntil(
            caches.open(STATIC_CACHE)
                .then((cache) => {
                    console.log('[ServiceWorker] Precaching assets');
                    return cache.addAll(PRECACHE_ASSETS);
                })
                .then(() => {
                    console.log('[ServiceWorker] Skip waiting');
                    return self.skipWaiting();
                })
        );
    });
}

function activateEvent() {
    self.addEventListener('activate', (event) => {
        console.log('[ServiceWorker] Activating...');
        
        event.waitUntil(
            caches.keys()
                .then((cacheNames) => {
                    return Promise.all(
                        cacheNames
                            .filter((cacheName) => {
                                return ![
                                    STATIC_CACHE,
                                    DYNAMIC_CACHE,
                                    IMAGE_CACHE,
                                    API_CACHE
                                ].includes(cacheName);
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
}

function fetchEvent() {
    self.addEventListener('fetch', (event) => {
        const { request } = event;
        const url = new URL(request.url);
        
        if (url.origin !== location.origin && !request.url.startsWith(self.location.origin)) {
            return;
        }
        
        const strategy = getCacheStrategy(request);
        
        switch (strategy.strategy) {
            case 'cacheFirst':
                event.respondWith(cacheFirst(request, strategy));
                break;
            case 'networkFirst':
                event.respondWith(networkFirst(request, strategy));
                break;
            case 'staleWhileRevalidate':
                event.respondWith(staleWhileRevalidate(request, strategy));
                break;
            default:
                event.respondWith(networkFirst(request, strategy));
        }
    });
}

function getCacheStrategy(request) {
    const url = request.url;
    
    for (const [name, config] of Object.entries(CACHE_STRATEGIES)) {
        if (config.pattern.test(url)) {
            return config;
        }
    }
    
    return CACHE_STRATEGIES.dynamic;
}

async function cacheFirst(request, strategy) {
    const cachedResponse = await caches.match(request);
    
    if (cachedResponse) {
        const cacheTime = cachedResponse.headers.get('sw-cache-time');
        if (cacheTime) {
            const age = Date.now() - parseInt(cacheTime);
            if (age < strategy.maxAge) {
                return cachedResponse;
            }
        } else {
            return cachedResponse;
        }
    }
    
    try {
        const networkResponse = await fetch(request);
        
        if (networkResponse.ok) {
            const cache = await caches.open(strategy.cacheName);
            const responseToCache = networkResponse.clone();
            
            const headers = new Headers(responseToCache.headers);
            headers.append('sw-cache-time', Date.now().toString());
            
            const cachedResponse = new Response(await responseToCache.blob(), {
                status: responseToCache.status,
                statusText: responseToCache.statusText,
                headers: headers
            });
            
            cache.put(request, cachedResponse);
        }
        
        return networkResponse;
    } catch (error) {
        console.error('[ServiceWorker] Cache first fetch failed:', error);
        return caches.match(OFFLINE_URL);
    }
}

async function networkFirst(request, strategy) {
    const cache = await caches.open(strategy.cacheName);
    
    try {
        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), strategy.networkTimeout || 3000);
        
        const networkResponse = await fetch(request, { signal: controller.signal });
        clearTimeout(timeoutId);
        
        if (networkResponse.ok) {
            const responseToCache = networkResponse.clone();
            
            const headers = new Headers(responseToCache.headers);
            headers.append('sw-cache-time', Date.now().toString());
            
            const cachedResponse = new Response(await responseToCache.blob(), {
                status: responseToCache.status,
                statusText: responseToCache.statusText,
                headers: headers
            });
            
            cache.put(request, cachedResponse);
        }
        
        return networkResponse;
    } catch (error) {
        console.log('[ServiceWorker] Network first, falling back to cache');
        
        const cachedResponse = await cache.match(request);
        if (cachedResponse) {
            return cachedResponse;
        }
        
        return caches.match(OFFLINE_URL);
    }
}

async function staleWhileRevalidate(request, strategy) {
    const cache = await caches.open(strategy.cacheName);
    const cachedResponse = await cache.match(request);
    
    const fetchPromise = fetch(request)
        .then((networkResponse) => {
            if (networkResponse.ok) {
                const headers = new Headers(networkResponse.headers);
                headers.append('sw-cache-time', Date.now().toString());
                
                const cachedResponse = new Response(await networkResponse.blob(), {
                    status: networkResponse.status,
                    statusText: networkResponse.statusText,
                    headers: headers
                });
                
                cache.put(request, cachedResponse);
            }
            return networkResponse;
        })
        .catch((error) => {
            console.error('[ServiceWorker] Stale while revalidate fetch failed:', error);
        });
    
    return cachedResponse || fetchPromise;
}

function syncEvent() {
    self.addEventListener('sync', (event) => {
        console.log('[ServiceWorker] Sync event:', event.tag);
        
        if (event.tag === SYNC_TAGS.backgroundSync) {
            event.waitUntil(doBackgroundSync());
        }
    });
}

async function doBackgroundSync() {
    try {
        const cache = await caches.open(DYNAMIC_CACHE);
        const requests = await cache.keys();
        
        const pendingRequests = requests.filter((request) => {
            return request.method === 'POST' || request.method === 'PUT';
        });
        
        for (const request of pendingRequests) {
            try {
                const response = await fetch(request);
                if (response.ok) {
                    await cache.delete(request);
                    console.log('[ServiceWorker] Synced request:', request.url);
                }
            } catch (error) {
                console.error('[ServiceWorker] Sync failed for:', request.url, error);
            }
        }
        
        notifyClients({ type: 'SYNC_COMPLETE' });
    } catch (error) {
        console.error('[ServiceWorker] Background sync failed:', error);
    }
}

function messageEvent() {
    self.addEventListener('message', (event) => {
        console.log('[ServiceWorker] Message received:', event.data);
        
        switch (event.data.type) {
            case 'SKIP_WAITING':
                self.skipWaiting();
                break;
                
            case 'GET_VERSION':
                event.ports[0].postMessage({ version: CACHE_NAME });
                break;
                
            case 'CLEAR_CACHE':
                event.waitUntil(clearAllCaches());
                event.ports[0].postMessage({ success: true });
                break;
                
            case 'CACHE_URLS':
                event.waitUntil(cacheUrls(event.data.urls));
                event.ports[0].postMessage({ success: true });
                break;
                
            case 'GET_CACHE_SIZE':
                event.waitUntil(getCacheSize().then((size) => {
                    event.ports[0].postMessage({ size: size });
                }));
                break;
                
            case 'PREFETCH':
                event.waitUntil(prefetchAssets(event.data.assets));
                event.ports[0].postMessage({ success: true });
                break;
        }
    });
}

async function clearAllCaches() {
    const cacheNames = await caches.keys();
    return Promise.all(
        cacheNames.map((cacheName) => caches.delete(cacheName))
    );
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
            console.error('[ServiceWorker] Failed to cache URL:', url, error);
        }
    }
}

async function getCacheSize() {
    const cacheNames = await caches.keys();
    let totalSize = 0;
    
    for (const cacheName of cacheNames) {
        const cache = await caches.open(cacheName);
        const requests = await cache.keys();
        
        for (const request of requests) {
            const response = await cache.match(request);
            if (response) {
                const blob = await response.clone().blob();
                totalSize += blob.size;
            }
        }
    }
    
    return totalSize;
}

async function prefetchAssets(assets) {
    const cache = await caches.open(STATIC_CACHE);
    
    for (const asset of assets) {
        try {
            const response = await fetch(asset);
            if (response.ok) {
                await cache.put(asset, response);
                console.log('[ServiceWorker] Prefetched:', asset);
            }
        } catch (error) {
            console.error('[ServiceWorker] Prefetch failed:', asset, error);
        }
    }
}

function periodicSyncEvent() {
    if ('periodicSync' in self.registration) {
        self.addEventListener('periodicsync', (event) => {
            console.log('[ServiceWorker] Periodic sync:', event.tag);
            
            if (event.tag === SYNC_TAGS.periodicSync) {
                event.waitUntil(doPeriodicSync());
            }
        });
    }
}

async function doPeriodicSync() {
    try {
        const cache = await caches.open(API_CACHE);
        
        const requests = await cache.keys();
        
        for (const request of requests) {
            try {
                const response = await fetch(request);
                if (response.ok) {
                    await cache.put(request, response);
                }
            } catch (error) {
                console.error('[ServiceWorker] Periodic sync refresh failed:', error);
            }
        }
        
        console.log('[ServiceWorker] Periodic sync completed');
    } catch (error) {
        console.error('[ServiceWorker] Periodic sync failed:', error);
    }
}

async function notifyClients(message) {
    const clients = await self.clients.matchAll({
        type: 'window',
        includeUncontrolled: true
    });
    
    for (const client of clients) {
        client.postMessage(message);
    }
}

self.addEventListener('notificationclick', (event) => {
    console.log('[ServiceWorker] Notification clicked:', event);
    
    event.notification.close();
    
    event.waitUntil(
        self.clients.matchAll({ type: 'window', includeUncontrolled: true })
            .then((clientList) => {
                for (const client of clientList) {
                    if (client.url.includes(self.location.origin) && 'focus' in client) {
                        return client.focus();
                    }
                }
                
                if (self.clients.openWindow) {
                    return self.clients.openWindow('/');
                }
            })
    );
});

self.addEventListener('notificationclose', (event) => {
    console.log('[ServiceWorker] Notification closed');
});

self.addEventListener('push', (event) => {
    console.log('[ServiceWorker] Push received');
    
    let data = {
        title: 'HJTPX',
        body: 'You have a new notification',
        icon: '/static/images/icon-192.png',
        badge: '/static/images/badge.png'
    };
    
    if (event.data) {
        try {
            data = { ...data, ...event.data.json() };
        } catch (error) {
            data.body = event.data.text();
        }
    }
    
    event.waitUntil(
        self.registration.showNotification(data.title, {
            body: data.body,
            icon: data.icon,
            badge: data.badge,
            tag: data.tag || 'hjtpx-notification',
            data: data.data
        })
    );
});

console.log('[ServiceWorker] Script loaded');
