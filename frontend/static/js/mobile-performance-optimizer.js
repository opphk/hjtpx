const MobilePerformanceOptimizer = (function() {
    'use strict';

    class ImageOptimizer {
        constructor() {
            this.quality = 0.8;
            this.maxWidth = 1920;
            this.maxHeight = 1920;
            this.supportedFormats = ['webp', 'jpeg', 'png'];
            this.observer = null;
            this.init();
        }

        init() {
            if ('IntersectionObserver' in window) {
                this.setupLazyLoading();
            }
            this.setupResponsiveImages();
            this.optimizeExistingImages();
        }

        setupLazyLoading() {
            this.observer = new IntersectionObserver((entries) => {
                entries.forEach(entry => {
                    if (entry.isIntersecting) {
                        this.loadImage(entry.target);
                        this.observer.unobserve(entry.target);
                    }
                });
            }, {
                rootMargin: '50px 0px',
                threshold: 0.1,
            });

            const lazyImages = document.querySelectorAll('img[data-src]');
            lazyImages.forEach(img => this.observer.observe(img));
        }

        setupResponsiveImages() {
            const images = document.querySelectorAll('img[data-srcset]');

            images.forEach(img => {
                const srcset = img.dataset.srcset;
                if (srcset) {
                    img.srcset = srcset;
                }

                const sizes = img.dataset.sizes;
                if (sizes) {
                    img.sizes = sizes;
                }
            });
        }

        optimizeExistingImages() {
            const images = document.querySelectorAll('img:not([data-optimized])');

            images.forEach(img => {
                img.setAttribute('data-optimized', 'true');

                if (!img.loading) {
                    img.loading = 'lazy';
                }

                if (!img.decoding) {
                    img.decoding = 'async';
                }

                if (img.naturalWidth === 0) {
                    img.addEventListener('load', () => {
                        this.optimizeImageDimensions(img);
                    });
                } else {
                    this.optimizeImageDimensions(img);
                }
            });
        }

        optimizeImageDimensions(img) {
            const maxWidth = window.innerWidth * window.devicePixelRatio;
            const maxHeight = window.innerHeight * window.devicePixelRatio;

            if (img.naturalWidth > maxWidth || img.naturalHeight > maxHeight) {
                const scale = Math.min(maxWidth / img.naturalWidth, maxHeight / img.naturalHeight);
                img.style.maxWidth = (img.naturalWidth * scale) + 'px';
                img.style.maxHeight = (img.naturalHeight * scale) + 'px';
            }
        }

        loadImage(img) {
            if (img.dataset.src) {
                img.src = img.dataset.src;
                img.removeAttribute('data-src');
            }
        }

        createResponsiveSrcset(imageUrl, widths = [320, 640, 960, 1280, 1920]) {
            const urlObj = new URL(imageUrl);
            const basePath = urlObj.pathname;
            const ext = basePath.split('.').pop();

            return widths
                .map(width => {
                    const newUrl = `${urlObj.origin}${basePath.replace(`.${ext}`, `_${width}w.${ext}`)} ${width}w`;
                    return newUrl;
                })
                .join(', ');
        }

        detectBestFormat() {
            if (typeof document === 'undefined') return 'jpeg';

            const canvas = document.createElement('canvas');
            canvas.width = 1;
            canvas.height = 1;

            if (canvas.toDataURL('image/webp').indexOf('data:image/webp') === 0) {
                return 'webp';
            }

            return 'jpeg';
        }

        destroy() {
            if (this.observer) {
                this.observer.disconnect();
            }
        }
    }

    class NetworkOptimizer {
        constructor() {
            this.prefetchedUrls = new Set();
            this.dnsPrefetchCache = new Map();
            this.requestQueue = [];
            this.maxConcurrentRequests = 4;
            this.init();
        }

        init() {
            this.setupPrefetching();
            this.setupDnsPrefetch();
            this.setupRequestBatching();
        }

        setupPrefetching() {
            const links = document.querySelectorAll('a[href^="/"], a[href^="./"]');

            links.forEach(link => {
                link.addEventListener('mouseenter', () => {
                    this.prefetchLink(link.href);
                }, { passive: true });

                link.addEventListener('touchstart', () => {
                    this.prefetchLink(link.href);
                }, { passive: true });
            });
        }

        setupDnsPrefetch() {
            const hostnames = new Set();

            document.querySelectorAll('img[src], link[href], script[src]').forEach(element => {
                const url = element.src || element.href;
                if (url) {
                    try {
                        const hostname = new URL(url).hostname;
                        hostnames.add(hostname);
                    } catch (e) {
                    }
                }
            });

            hostnames.forEach(hostname => {
                const link = document.createElement('link');
                link.rel = 'dns-prefetch';
                link.href = `//${hostname}`;
                document.head.appendChild(link);
            });
        }

        setupRequestBatching() {
            const batchButtons = document.querySelectorAll('[data-batch-request]');

            batchButtons.forEach(button => {
                button.addEventListener('click', () => {
                    const batchConfig = JSON.parse(button.dataset.batchRequest);
                    this.batchRequest(batchConfig);
                });
            });
        }

        prefetchLink(url) {
            if (this.prefetchedUrls.has(url)) {
                return;
            }

            this.prefetchedUrls.add(url);

            const link = document.createElement('link');
            link.rel = 'prefetch';
            link.href = url;
            link.as = 'document';
            document.head.appendChild(link);

            setTimeout(() => {
                link.remove();
            }, 30000);
        }

        prefetchDns(hostname) {
            if (this.dnsPrefetchCache.has(hostname)) {
                return;
            }

            this.dnsPrefetchCache.set(hostname, true);

            const link = document.createElement('link');
            link.rel = 'dns-prefetch';
            link.href = `//${hostname}`;
            document.head.appendChild(link);

            const preconnect = document.createElement('link');
            preconnect.rel = 'preconnect';
            preconnect.href = `//${hostname}`;
            preconnect.crossOrigin = 'anonymous';
            document.head.appendChild(preconnect);
        }

        batchRequest(configs) {
            const batchedRequests = configs.map(config => {
                return this.requestQueue.push({
                    url: config.url,
                    options: config.options || {},
                    resolve: null,
                    reject: null,
                });
            });

            this.processQueue();

            return Promise.all(
                this.requestQueue.slice(-batchedRequests.length).map(req => {
                    return new Promise((resolve, reject) => {
                        req.resolve = resolve;
                        req.reject = reject;
                    });
                })
            );
        }

        processQueue() {
            const batch = this.requestQueue.splice(0, this.maxConcurrentRequests);

            batch.forEach(request => {
                fetch(request.url, request.options)
                    .then(response => {
                        if (request.resolve) {
                            request.resolve(response);
                        }
                    })
                    .catch(error => {
                        if (request.reject) {
                            request.reject(error);
                        }
                    });
            });

            if (this.requestQueue.length > 0) {
                setTimeout(() => this.processQueue(), 100);
            }
        }

        preloadResource(url, type = 'fetch') {
            const link = document.createElement('link');
            link.rel = 'preload';
            link.href = url;
            link.as = type;
            document.head.appendChild(link);

            return new Promise((resolve, reject) => {
                link.onload = () => resolve(link);
                link.onerror = () => reject(new Error(`Failed to preload: ${url}`));
            });
        }
    }

    class RenderOptimizer {
        constructor() {
            this.frameRate = 60;
            this.lastFrameTime = 0;
            this.animationQueue = [];
            this.isAnimating = false;
            this.init();
        }

        init() {
            this.optimizeCss();
            this.setupRequestAnimationFrame();
            this.optimizeReflow();
            this.deferNonCriticalRendering();
        }

        optimizeCss() {
            const criticalCSS = `
                .captcha-container { opacity: 0; }
                .captcha-ready .captcha-container { opacity: 1; transition: opacity 0.3s ease-in; }
            `;

            const style = document.createElement('style');
            style.textContent = criticalCSS;
            document.head.insertBefore(style, document.head.firstChild);
        }

        setupRequestAnimationFrame() {
            const raf = window.requestAnimationFrame;

            window.requestAnimationFrame = (callback) => {
                return raf((timestamp) => {
                    if (timestamp - this.lastFrameTime < (1000 / this.frameRate)) {
                        return;
                    }

                    this.lastFrameTime = timestamp;
                    callback(timestamp);
                });
            };
        }

        optimizeReflow() {
            const batchUpdates = [];
            let rafId = null;

            const processBatch = () => {
                batchUpdates.forEach(update => {
                    try {
                        update.callback();
                    } catch (error) {
                        console.error('Batch update error:', error);
                    }
                });
                batchUpdates.length = 0;
                rafId = null;
            };

            window.batchDOMUpdate = (callback) => {
                batchUpdates.push({ callback });

                if (!rafId) {
                    rafId = requestAnimationFrame(processBatch);
                }
            };
        }

        deferNonCriticalRendering() {
            const deferElements = document.querySelectorAll('[data-defer]');

            deferElements.forEach(element => {
                const content = element.innerHTML;
                element.innerHTML = '';

                const observer = new IntersectionObserver((entries) => {
                    entries.forEach(entry => {
                        if (entry.isIntersecting) {
                            element.innerHTML = content;
                            observer.unobserve(element);
                        }
                    });
                }, {
                    rootMargin: '100px 0px',
                });

                observer.observe(element);
            });
        }

        addToAnimationQueue(animation) {
            this.animationQueue.push(animation);

            if (!this.isAnimating) {
                this.startAnimationLoop();
            }
        }

        startAnimationLoop() {
            this.isAnimating = true;

            const animate = (timestamp) => {
                if (this.animationQueue.length === 0) {
                    this.isAnimating = false;
                    return;
                }

                this.animationQueue.forEach(animation => {
                    try {
                        animation.update(timestamp);
                    } catch (error) {
                        console.error('Animation error:', error);
                    }
                });

                this.animationQueue = this.animationQueue.filter(animation => !animation.completed);

                requestAnimationFrame(animate);
            };

            requestAnimationFrame(animate);
        }
    }

    class MemoryOptimizer {
        constructor() {
            this.objectPool = new Map();
            this.eventListenerCache = new Map();
            this.init();
        }

        init() {
            this.setupCleanupListeners();
            this.optimizeEventListeners();
        }

        setupCleanupListeners() {
            document.addEventListener('visibilitychange', () => {
                if (document.hidden) {
                    this.cleanupOnHide();
                } else {
                    this.restoreOnShow();
                }
            });

            window.addEventListener('beforeunload', () => {
                this.cleanupOnUnload();
            });
        }

        optimizeEventListeners() {
            const eventTypes = ['click', 'touchstart', 'touchmove', 'touchend'];

            eventTypes.forEach(eventType => {
                const wrappedHandler = this.throttle(
                    this.handleEvent.bind(this, eventType),
                    100
                );

                document.addEventListener(eventType, wrappedHandler, {
                    passive: true,
                    capture: false,
                });

                this.eventListenerCache.set(eventType, wrappedHandler);
            });
        }

        handleEvent(eventType, event) {
        }

        throttle(func, wait) {
            let timeout;
            let previous = 0;

            return function executedFunction(...args) {
                const now = Date.now();
                const remaining = wait - (now - previous);

                if (remaining <= 0 || remaining > wait) {
                    if (timeout) {
                        clearTimeout(timeout);
                        timeout = null;
                    }
                    previous = now;
                    func.apply(this, args);
                } else if (!timeout) {
                    timeout = setTimeout(() => {
                        previous = Date.now();
                        timeout = null;
                        func.apply(this, args);
                    }, remaining);
                }
            };
        }

        cleanupOnHide() {
            const canvases = document.querySelectorAll('canvas');
            canvases.forEach(canvas => {
                const ctx = canvas.getContext('2d');
                if (ctx) {
                    ctx.clearRect(0, 0, canvas.width, canvas.height);
                }
            });

            const images = document.querySelectorAll('img:not([src])');
            images.forEach(img => img.src = '');
        }

        restoreOnShow() {
            console.log('Page restored from hidden state');
        }

        cleanupOnUnload() {
            Object.keys(this.objectPool).forEach(key => {
                this.objectPool.get(key)?.clear?.();
                this.objectPool.delete(key);
            });

            this.eventListenerCache.forEach((handler, eventType) => {
                document.removeEventListener(eventType, handler);
            });
            this.eventListenerCache.clear();
        }

        createObjectPool(createFn, initialSize = 10) {
            const pool = [];

            for (let i = 0; i < initialSize; i++) {
                pool.push(createFn());
            }

            return {
                acquire() {
                    return pool.length > 0 ? pool.pop() : createFn();
                },
                release(obj) {
                    if (pool.length < 100) {
                        pool.push(obj);
                    }
                },
                clear() {
                    pool.length = 0;
                },
                size() {
                    return pool.length;
                },
            };
        }
    }

    class BatteryOptimizer {
        constructor() {
            this.isLowPowerMode = false;
            this.batteryLevel = 1.0;
            this.init();
        }

        async init() {
            if ('getBattery' in navigator) {
                try {
                    const battery = await navigator.getBattery();
                    this.batteryLevel = battery.level;

                    battery.addEventListener('levelchange', () => {
                        this.batteryLevel = battery.level;
                        this.adaptToBatteryLevel();
                    });

                    battery.addEventListener('chargingchange', () => {
                        this.adaptToBatteryLevel();
                    });

                    this.adaptToBatteryLevel();
                } catch (error) {
                    console.warn('Battery API not available:', error);
                }
            }

            this.detectLowPowerMode();
        }

        detectLowPowerMode() {
            if ('matchMedia' in window) {
                const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)');
                this.isLowPowerMode = prefersReducedMotion.matches;

                prefersReducedMotion.addEventListener('change', (e) => {
                    this.isLowPowerMode = e.matches;
                    this.adaptToPowerMode();
                });
            }
        }

        adaptToBatteryLevel() {
            if (this.batteryLevel < 0.2) {
                this.enableBatterySavingMode();
            } else if (this.batteryLevel > 0.5) {
                this.disableBatterySavingMode();
            }
        }

        adaptToPowerMode() {
            if (this.isLowPowerMode) {
                this.enableBatterySavingMode();
            } else {
                this.disableBatterySavingMode();
            }
        }

        enableBatterySavingMode() {
            document.documentElement.classList.add('battery-saving');

            const animations = document.querySelectorAll('[style*="animation"]');
            animations.forEach(el => {
                el.style.animationPlayState = 'paused';
            });
        }

        disableBatterySavingMode() {
            document.documentElement.classList.remove('battery-saving');

            const animations = document.querySelectorAll('[style*="animation"]');
            animations.forEach(el => {
                el.style.animationPlayState = 'running';
            });
        }

        getStatus() {
            return {
                batteryLevel: this.batteryLevel,
                isLowPowerMode: this.isLowPowerMode,
            };
        }
    }

    let instance = null;

    return {
        init: function() {
            if (!instance) {
                instance = {
                    imageOptimizer: new ImageOptimizer(),
                    networkOptimizer: new NetworkOptimizer(),
                    renderOptimizer: new RenderOptimizer(),
                    memoryOptimizer: new MemoryOptimizer(),
                    batteryOptimizer: new BatteryOptimizer(),
                };
            }
            return instance;
        },

        getOptimizer: function(name) {
            if (!instance) {
                this.init();
            }
            return instance[name];
        },

        getAllOptimizers: function() {
            if (!instance) {
                this.init();
            }
            return instance;
        },
    };
})();

if (typeof module !== 'undefined' && module.exports) {
    module.exports = MobilePerformanceOptimizer;
}

if (typeof window !== 'undefined') {
    window.MobilePerformanceOptimizer = MobilePerformanceOptimizer;
}
