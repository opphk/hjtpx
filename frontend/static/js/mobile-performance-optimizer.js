const MobilePerformanceOptimizer = (function() {
    'use strict';

    class PerformanceMonitor {
        constructor() {
            this.metrics = {
                firstPaint: 0,
                firstContentfulPaint: 0,
                domContentLoaded: 0,
                loadComplete: 0,
                firstInputDelay: 0,
                largestContentfulPaint: 0,
                cumulativeLayoutShift: 0,
                timeToInteractive: 0
            };
            this.observers = [];
            this.init();
        }

        init() {
            this.setupPaintObserver();
            this.setupNavigationObserver();
            this.setupInputDelayObserver();
            this.setupLCPObserver();
            this.setupCLObserver();
            this.recordInitialMetrics();
            this.setupPerformanceReporting();
        }

        setupPaintObserver() {
            if ('PerformanceObserver' in window) {
                try {
                    const observer = new PerformanceObserver((list) => {
                        for (const entry of list.getEntries()) {
                            if (entry.name === 'first-paint') {
                                this.metrics.firstPaint = entry.startTime;
                            } else if (entry.name === 'first-contentful-paint') {
                                this.metrics.firstContentfulPaint = entry.startTime;
                            }
                        }
                    });
                    observer.observe({ entryTypes: ['paint'] });
                    this.observers.push(observer);
                } catch (e) {
                    console.warn('Paint observer not supported:', e);
                }
            }
        }

        setupNavigationObserver() {
            if ('PerformanceObserver' in window) {
                try {
                    const observer = new PerformanceObserver((list) => {
                        const entries = list.getEntries();
                        if (entries.length > 0) {
                            const navEntry = entries[0];
                            this.metrics.domContentLoaded = navEntry.domContentLoadedEventStart;
                            this.metrics.loadComplete = navEntry.loadEventEnd;
                            this.metrics.timeToInteractive = navEntry.domInteractive;
                        }
                    });
                    observer.observe({ entryTypes: ['navigation'] });
                    this.observers.push(observer);
                } catch (e) {
                    console.warn('Navigation observer not supported:', e);
                }
            }
        }

        setupInputDelayObserver() {
            if ('PerformanceObserver' in window && 'EventCounts' in window) {
                try {
                    const observer = new PerformanceObserver((list) => {
                        for (const entry of list.getEntries()) {
                            if (entry.processingStart > entry.startTime) {
                                this.metrics.firstInputDelay = entry.processingStart - entry.startTime;
                            }
                        }
                    });
                    observer.observe({ entryTypes: ['first-input'], buffered: true });
                    this.observers.push(observer);
                } catch (e) {
                    console.warn('First input observer not supported:', e);
                }
            }
        }

        setupLCPObserver() {
            if ('PerformanceObserver' in window) {
                try {
                    const observer = new PerformanceObserver((list) => {
                        const entries = list.getEntries();
                        if (entries.length > 0) {
                            this.metrics.largestContentfulPaint = entries[entries.length - 1].startTime;
                        }
                    });
                    observer.observe({ entryTypes: ['largest-contentful-paint'], buffered: true });
                    this.observers.push(observer);
                } catch (e) {
                    console.warn('LCP observer not supported:', e);
                }
            }
        }

        setupCLObserver() {
            if ('PerformanceObserver' in window) {
                try {
                    const observer = new PerformanceObserver((list) => {
                        for (const entry of list.getEntries()) {
                            if (!entry.hadRecentInput) {
                                this.metrics.cumulativeLayoutShift += entry.value;
                            }
                        }
                    });
                    observer.observe({ entryTypes: ['layout-shift'], buffered: true });
                    this.observers.push(observer);
                } catch (e) {
                    console.warn('CLS observer not supported:', e);
                }
            }
        }

        recordInitialMetrics() {
            if ('getEntriesByType' in performance) {
                const paintEntries = performance.getEntriesByType('paint');
                paintEntries.forEach(entry => {
                    if (entry.name === 'first-paint') {
                        this.metrics.firstPaint = entry.startTime;
                    } else if (entry.name === 'first-contentful-paint') {
                        this.metrics.firstContentfulPaint = entry.startTime;
                    }
                });

                const navEntries = performance.getEntriesByType('navigation');
                if (navEntries.length > 0) {
                    const navEntry = navEntries[0];
                    this.metrics.domContentLoaded = navEntry.domContentLoadedEventStart;
                    this.metrics.loadComplete = navEntry.loadEventEnd;
                }
            }
        }

        setupPerformanceReporting() {
            document.addEventListener('DOMContentLoaded', () => {
                this.metrics.domContentLoaded = performance.now();
            });

            window.addEventListener('load', () => {
                this.metrics.loadComplete = performance.now();
                this.reportMetrics();
            });
        }

        reportMetrics() {
            console.group('📊 Performance Metrics');
            console.log('First Paint:', this.metrics.firstPaint.toFixed(2) + 'ms');
            console.log('First Contentful Paint:', this.metrics.firstContentfulPaint.toFixed(2) + 'ms');
            console.log('DOM Content Loaded:', this.metrics.domContentLoaded.toFixed(2) + 'ms');
            console.log('Load Complete:', this.metrics.loadComplete.toFixed(2) + 'ms');
            console.log('First Input Delay:', this.metrics.firstInputDelay.toFixed(2) + 'ms');
            console.log('Largest Contentful Paint:', this.metrics.largestContentfulPaint.toFixed(2) + 'ms');
            console.log('Cumulative Layout Shift:', this.metrics.cumulativeLayoutShift.toFixed(4));
            console.log('Time to Interactive:', this.metrics.timeToInteractive.toFixed(2) + 'ms');
            console.groupEnd();

            this.sendMetricsToServer();
        }

        sendMetricsToServer() {
            if (typeof fetch !== 'undefined') {
                fetch('/api/performance-metrics', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        metrics: this.metrics,
                        timestamp: Date.now(),
                        url: window.location.href,
                        userAgent: navigator.userAgent
                    })
                }).catch(() => {});
            }
        }

        getMetrics() {
            return { ...this.metrics };
        }

        destroy() {
            this.observers.forEach(observer => {
                observer.disconnect();
            });
            this.observers = [];
        }
    }

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

    class InteractionOptimizer {
        constructor() {
            this.interactionMetrics = {
                totalInteractions: 0,
                averageResponseTime: 0,
                longestResponseTime: 0,
                interactionHistory: []
            };
            this.responseTimeThreshold = 100;
            this.observers = [];
            this.init();
        }

        init() {
            this.setupInteractionTracking();
            this.setupTouchOptimization();
            this.setupClickOptimization();
            this.setupScrollOptimization();
            this.setupAnimationOptimization();
        }

        setupInteractionTracking() {
            const interactionTypes = ['click', 'touchstart', 'touchend', 'keypress'];

            interactionTypes.forEach(type => {
                document.addEventListener(type, (event) => {
                    const startTime = performance.now();

                    const measureResponse = () => {
                        const endTime = performance.now();
                        const responseTime = endTime - startTime;

                        this.interactionMetrics.totalInteractions++;
                        this.interactionMetrics.interactionHistory.push({
                            type,
                            responseTime,
                            timestamp: endTime
                        });

                        if (this.interactionMetrics.interactionHistory.length > 100) {
                            this.interactionMetrics.interactionHistory.shift();
                        }

                        this.updateAverageResponseTime();

                        if (responseTime > this.interactionMetrics.longestResponseTime) {
                            this.interactionMetrics.longestResponseTime = responseTime;
                        }
                    };

                    requestAnimationFrame(measureResponse);
                }, { passive: true });
            });
        }

        setupTouchOptimization() {
            if ('ontouchstart' in window) {
                document.addEventListener('touchstart', () => {
                    document.body.classList.add('touch-enabled');
                }, { passive: true });

                document.addEventListener('touchend', () => {
                    document.body.classList.remove('touch-active');
                }, { passive: true });
            }
        }

        setupClickOptimization() {
            const clickableElements = document.querySelectorAll('button, a, [role="button"]');

            clickableElements.forEach(element => {
                if (!element.hasAttribute('data-optimized')) {
                    element.setAttribute('data-optimized', 'true');

                    element.addEventListener('click', (e) => {
                        this.handleClickFeedback(e.target);
                    }, { passive: true });
                }
            });
        }

        handleClickFeedback(element) {
            const rect = element.getBoundingClientRect();
            const x = rect.left + rect.width / 2;
            const y = rect.top + rect.height / 2;

            const ripple = document.createElement('span');
            ripple.className = 'click-ripple';
            ripple.style.cssText = `
                position: fixed;
                left: ${x}px;
                top: ${y}px;
                width: 0;
                height: 0;
                border-radius: 50%;
                background: rgba(13, 110, 253, 0.3);
                pointer-events: none;
                transform: translate(-50%, -50%);
                transition: width 0.3s ease-out, height 0.3s ease-out, opacity 0.3s ease-out;
            `;

            document.body.appendChild(ripple);

            requestAnimationFrame(() => {
                ripple.style.width = '100px';
                ripple.style.height = '100px';
                ripple.style.opacity = '0';
            });

            setTimeout(() => {
                ripple.remove();
            }, 300);
        }

        setupScrollOptimization() {
            let ticking = false;

            window.addEventListener('scroll', () => {
                if (!ticking) {
                    requestAnimationFrame(() => {
                        this.handleScrollOptimization();
                        ticking = false;
                    });
                    ticking = true;
                }
            }, { passive: true });
        }

        handleScrollOptimization() {
            const scrollTop = window.pageYOffset || document.documentElement.scrollTop;
            const scrollHeight = document.documentElement.scrollHeight - window.innerHeight;
            const scrollPercentage = (scrollTop / scrollHeight) * 100;

            document.body.dataset.scrollProgress = scrollPercentage;
        }

        setupAnimationOptimization() {
            if ('matchMedia' in window) {
                const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)');

                if (prefersReducedMotion.matches) {
                    this.disableAnimations();
                }

                prefersReducedMotion.addEventListener('change', (e) => {
                    if (e.matches) {
                        this.disableAnimations();
                    } else {
                        this.enableAnimations();
                    }
                });
            }
        }

        disableAnimations() {
            document.documentElement.style.setProperty('--animation-duration', '0.01ms');
            document.documentElement.style.setProperty('--transition-duration', '0.01ms');
            document.body.classList.add('reduced-motion');
        }

        enableAnimations() {
            document.documentElement.style.removeProperty('--animation-duration');
            document.documentElement.style.removeProperty('--transition-duration');
            document.body.classList.remove('reduced-motion');
        }

        updateAverageResponseTime() {
            const history = this.interactionMetrics.interactionHistory;
            if (history.length === 0) return;

            const total = history.reduce((sum, interaction) => sum + interaction.responseTime, 0);
            this.interactionMetrics.averageResponseTime = total / history.length;
        }

        getMetrics() {
            return { ...this.interactionMetrics };
        }

        isResponsive() {
            return this.interactionMetrics.averageResponseTime < this.responseTimeThreshold;
        }

        destroy() {
            this.observers.forEach(observer => {
                observer.disconnect();
            });
            this.observers = [];
        }
    }

    let instance = null;

    return {
        init: function(options = {}) {
            if (!instance) {
                instance = {
                    performanceMonitor: new PerformanceMonitor(),
                    imageOptimizer: new ImageOptimizer(),
                    networkOptimizer: new NetworkOptimizer(),
                    renderOptimizer: new RenderOptimizer(),
                    memoryOptimizer: new MemoryOptimizer(),
                    batteryOptimizer: new BatteryOptimizer(),
                    interactionOptimizer: new InteractionOptimizer(),
                };
            }

            if (options.autoReport) {
                window.addEventListener('load', () => {
                    setTimeout(() => {
                        const metrics = instance.performanceMonitor.getMetrics();
                        instance.performanceMonitor.reportMetrics();
                    }, 2000);
                });
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

        getMetrics: function() {
            if (!instance) {
                this.init();
            }
            return instance.performanceMonitor.getMetrics();
        }
    };
})();

if (typeof module !== 'undefined' && module.exports) {
    module.exports = MobilePerformanceOptimizer;
}

if (typeof window !== 'undefined') {
    window.MobilePerformanceOptimizer = MobilePerformanceOptimizer;
}
