/**
 * HJTPX 移动端性能优化器
 * 提供移动端性能优化功能，包括图片懒加载、渲染优化、内存管理等
 */

class MobilePerformanceOptimizer {
    constructor(options = {}) {
        this.config = {
            enableImageOptimization: options.enableImageOptimization !== false,
            enableRenderingOptimization: options.enableRenderingOptimization !== false,
            enableMemoryOptimization: options.enableMemoryOptimization !== false,
            enableNetworkOptimization: options.enableNetworkOptimization !== false,
            lazyLoadThreshold: options.lazyLoadThreshold || 100,
            maxCacheSize: options.maxCacheSize || 50 * 1024 * 1024,
            fpsTarget: options.fpsTarget || 60,
            ...options
        };

        this.observer = null;
        this.animationFrameId = null;
        this.cachedImages = new Map();
        this.performanceMetrics = this.initMetrics();
        this.isLowPowerMode = false;
        this.batteryLevel = 1.0;
        
        this.init();
    }

    init() {
        if (typeof window === 'undefined') return;

        this.detectBatteryStatus();
        this.detectLowPowerMode();
        
        if (this.config.enableRenderingOptimization) {
            this.setupRenderingOptimization();
        }
        
        if (this.config.enableMemoryOptimization) {
            this.setupMemoryOptimization();
        }
        
        if (this.config.enableNetworkOptimization) {
            this.setupNetworkOptimization();
        }
        
        if (this.config.enableImageOptimization && 'IntersectionObserver' in window) {
            this.setupLazyLoading();
        }
        
        this.startPerformanceMonitoring();
    }

    detectBatteryStatus() {
        if ('getBattery' in navigator) {
            navigator.getBattery().then(battery => {
                this.batteryLevel = battery.level;
                this.isLowPowerMode = battery.level < 0.2;
                
                battery.addEventListener('levelchange', () => {
                    this.batteryLevel = battery.level;
                    this.isLowPowerMode = battery.level < 0.2;
                    this.adjustPerformanceForBattery();
                });
            });
        }
    }

    detectLowPowerMode() {
        if (matchMedia) {
            const prefersReducedMotion = matchMedia('(prefers-reduced-motion: reduce)');
            
            if (prefersReducedMotion.matches) {
                this.config.fpsTarget = 30;
                this.config.enableAnimations = false;
            }
            
            prefersReducedMotion.addEventListener('change', (e) => {
                if (e.matches) {
                    this.config.fpsTarget = 30;
                    this.config.enableAnimations = false;
                } else {
                    this.config.fpsTarget = 60;
                    this.config.enableAnimations = true;
                }
            });
        }
    }

    adjustPerformanceForBattery() {
        if (this.isLowPowerMode) {
            this.config.fpsTarget = 30;
            this.config.enableAnimations = false;
            this.config.enableImageOptimization = false;
            
            this.reduceRenderingQuality();
        } else if (this.batteryLevel < 0.5) {
            this.config.fpsTarget = 45;
        } else {
            this.config.fpsTarget = 60;
            this.config.enableAnimations = true;
            this.config.enableImageOptimization = true;
        }
    }

    reduceRenderingQuality() {
        document.documentElement.style.setProperty('--animation-duration', '0s');
        document.documentElement.style.setProperty('--transition-duration', '0s');
    }

    setupRenderingOptimization() {
        this.usePassiveEventListeners();
        this.optimizeScrollPerformance();
        this.setupWillChangeHints();
    }

    usePassiveEventListeners() {
        if (!this.supportsPassiveEvents()) return;

        const originalAddEventListener = EventTarget.prototype.addEventListener;
        
        EventTarget.prototype.addEventListener = function(type, listener, options) {
            if (type === 'touchmove' || type === 'touchstart' || type === 'wheel' || type === 'scroll') {
                if (typeof options === 'boolean') {
                    options = { passive: true };
                } else if (typeof options === 'object') {
                    options.passive = options.passive !== false;
                }
            }
            
            return originalAddEventListener.call(this, type, listener, options);
        };
    }

    supportsPassiveEvents() {
        let supportsPassive = false;
        
        try {
            const opts = Object.defineProperty({}, 'passive', {
                get: function() {
                    supportsPassive = true;
                }
            });
            
            window.addEventListener('test', null, opts);
        } catch (e) {
            supportsPassive = false;
        }
        
        return supportsPassive;
    }

    optimizeScrollPerformance() {
        let lastScrollY = window.scrollY;
        let ticking = false;

        const updateScrollPosition = () => {
            lastScrollY = window.scrollY;
            
            requestAnimationFrame(() => {
                this.performanceMetrics.scrollEvents++;
            });
            
            ticking = false;
        };

        window.addEventListener('scroll', () => {
            lastScrollY = window.scrollY;
            
            if (!ticking) {
                requestAnimationFrame(updateScrollPosition);
                ticking = true;
            }
        }, { passive: true });
    }

    setupWillChangeHints() {
        const animatedElements = document.querySelectorAll('.captcha-animated, .slider-button, .slider-track');
        
        animatedElements.forEach(element => {
            element.style.willChange = 'transform, opacity';
        });
    }

    setupMemoryOptimization() {
        this.setupCleanupOnVisibilityChange();
        this.setupImageCacheCleanup();
        this.cleanupEventListeners();
    }

    setupCleanupOnVisibilityChange() {
        document.addEventListener('visibilitychange', () => {
            if (document.hidden) {
                this.pauseAnimations();
                this.releaseUnusedResources();
            } else {
                this.resumeAnimations();
            }
        });
    }

    pauseAnimations() {
        if (!this.config.enableAnimations) return;

        document.querySelectorAll('video, audio, canvas').forEach(media => {
            if (media.pause) {
                media.pause();
            }
        });
    }

    resumeAnimations() {
        if (!this.config.enableAnimations) return;

        document.querySelectorAll('video, audio').forEach(media => {
            if (media.play) {
                media.play().catch(() => {});
            }
        });
    }

    releaseUnusedResources() {
        this.cachedImages.forEach((cacheEntry, key) => {
            if (Date.now() - cacheEntry.timestamp > 60000) {
                this.cachedImages.delete(key);
            }
        });
    }

    setupImageCacheCleanup() {
        setInterval(() => {
            let totalSize = 0;
            
            this.cachedImages.forEach((cacheEntry) => {
                totalSize += cacheEntry.size;
            });
            
            if (totalSize > this.config.maxCacheSize) {
                this.cleanOldestCacheEntries();
            }
        }, 30000);
    }

    cleanOldestCacheEntries() {
        const entries = Array.from(this.cachedImages.entries())
            .sort((a, b) => a[1].timestamp - b[1].timestamp);
        
        const toRemove = Math.ceil(entries.length * 0.3);
        
        for (let i = 0; i < toRemove && i < entries.length; i++) {
            const img = entries[i][1].image;
            if (img && img.src) {
                img.src = '';
            }
            this.cachedImages.delete(entries[i][0]);
        }
    }

    cleanupEventListeners() {
        const cleanup = () => {
            this.performanceMetrics.memoryUsage = this.getMemoryUsage();
        };

        if ('onpagehide' in window) {
            window.addEventListener('pagehide', cleanup);
        } else {
            window.addEventListener('unload', cleanup);
        }
    }

    getMemoryUsage() {
        if (performance.memory) {
            return {
                usedJSHeapSize: performance.memory.usedJSHeapSize,
                totalJSHeapSize: performance.memory.totalJSHeapSize,
                jsHeapSizeLimit: performance.memory.jsHeapSizeLimit,
            };
        }
        return null;
    }

    setupNetworkOptimization() {
        this.setupDnsPrefetch();
        this.setupPreconnect();
        this.optimizeRequestBatching();
    }

    setupDnsPrefetch() {
        const prefetchLinks = document.querySelectorAll('link[rel="dns-prefetch"]');
        prefetchLinks.forEach(link => {
            const href = link.href;
            if (href) {
                const url = new URL(href, window.location.origin);
                const hostname = url.hostname;
                
                const anchor = document.createElement('a');
                anchor.href = `https://${hostname}`;
                anchor rel = 'preconnect';
            }
        });
    }

    setupPreconnect() {
        const preconnectUrls = [
            window.location.origin,
        ];
        
        preconnectUrls.forEach(url => {
            const link = document.createElement('link');
            link.rel = 'preconnect';
            link.href = url;
            link.crossOrigin = 'anonymous';
            document.head.appendChild(link);
        });
    }

    optimizeRequestBatching() {
        this.pendingRequests = [];
        this.requestBatchTimeout = null;
    }

    batchRequest(request) {
        return new Promise((resolve, reject) => {
            this.pendingRequests.push({ request, resolve, reject });
            
            if (!this.requestBatchTimeout) {
                this.requestBatchTimeout = setTimeout(() => {
                    this.flushRequestBatch();
                }, 100);
            }
        });
    }

    flushRequestBatch() {
        const batch = this.pendingRequests.splice(0);
        this.requestBatchTimeout = null;
        
        batch.forEach(({ request, resolve, reject }) => {
            fetch(request.url, request.options)
                .then(resolve)
                .catch(reject);
        });
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
            rootMargin: `${this.config.lazyLoadThreshold}px`,
            threshold: 0.01,
        });

        document.querySelectorAll('img[data-src]').forEach(img => {
            this.observer.observe(img);
        });
    }

    loadImage(img) {
        const src = img.dataset.src;
        
        if (this.cachedImages.has(src)) {
            img.src = src;
            img.classList.remove('lazy-loading');
            img.classList.add('lazy-loaded');
            return;
        }

        img.classList.add('lazy-loading');

        const tempImage = new Image();
        
        tempImage.onload = () => {
            img.src = src;
            img.classList.remove('lazy-loading');
            img.classList.add('lazy-loaded');
            
            this.cachedImages.set(src, {
                image: tempImage,
                size: tempImage.src.length * 2,
                timestamp: Date.now(),
            });
        };

        tempImage.onerror = () => {
            img.classList.remove('lazy-loading');
            img.classList.add('lazy-error');
        };

        tempImage.src = src;
    }

    startPerformanceMonitoring() {
        this.monitorFPS();
        this.monitorMemoryUsage();
    }

    monitorFPS() {
        let frameCount = 0;
        let lastTime = performance.now();

        const measureFPS = () => {
            const currentTime = performance.now();
            frameCount++;

            if (currentTime >= lastTime + 1000) {
                this.performanceMetrics.fps = frameCount;
                frameCount = 0;
                lastTime = currentTime;

                if (this.performanceMetrics.fps < this.config.fpsTarget) {
                    this.onLowFPS(this.performanceMetrics.fps);
                }
            }

            this.animationFrameId = requestAnimationFrame(measureFPS);
        };

        this.animationFrameId = requestAnimationFrame(measureFPS);
    }

    onLowFPS(fps) {
        this.performanceMetrics.lowFPSCount++;
        
        if (this.performanceMetrics.lowFPSCount > 10) {
            this.reduceRenderingQuality();
        }
    }

    monitorMemoryUsage() {
        setInterval(() => {
            const memory = this.getMemoryUsage();
            if (memory) {
                this.performanceMetrics.memoryUsage = memory;
                
                const usageRatio = memory.usedJSHeapSize / memory.jsHeapSizeLimit;
                
                if (usageRatio > 0.9) {
                    this.triggerGarbageCollection();
                }
            }
        }, 5000);
    }

    triggerGarbageCollection() {
        this.releaseUnusedResources();
        
        if (this.cachedImages.size > 10) {
            this.cleanOldestCacheEntries();
        }
    }

    initMetrics() {
        return {
            fps: 60,
            scrollEvents: 0,
            touchEvents: 0,
            memoryUsage: null,
            lowFPSCount: 0,
            imagesLoaded: 0,
            networkRequests: 0,
        };
    }

    getMetrics() {
        return { ...this.performanceMetrics };
    }

    optimizeCanvasRendering(canvas) {
        const ctx = canvas.getContext('2d');
        
        ctx.imageSmoothingEnabled = true;
        ctx.imageSmoothingQuality = 'high';
        
        const dpr = window.devicePixelRatio || 1;
        
        return {
            scale: dpr,
            width: canvas.width,
            height: canvas.height,
        };
    }

    debounce(func, wait) {
        let timeout;
        
        return function executedFunction(...args) {
            const later = () => {
                clearTimeout(timeout);
                func(...args);
            };
            
            clearTimeout(timeout);
            timeout = setTimeout(later, wait);
        };
    }

    throttle(func, limit) {
        let inThrottle;
        
        return function executedFunction(...args) {
            if (!inThrottle) {
                func(...args);
                inThrottle = true;
                setTimeout(() => inThrottle = false, limit);
            }
        };
    }

    destroy() {
        if (this.animationFrameId) {
            cancelAnimationFrame(this.animationFrameId);
        }

        if (this.observer) {
            this.observer.disconnect();
        }

        this.cachedImages.clear();
        
        this.performanceMetrics = this.initMetrics();
    }
}

class TouchPerformanceAnalyzer {
    constructor() {
        this.touchEvents = [];
        this.maxHistory = 100;
        this.lastFrameTime = 0;
        this.frameCount = 0;
    }

    recordTouch(touch) {
        const now = performance.now();
        
        this.touchEvents.push({
            x: touch.clientX,
            y: touch.clientY,
            timestamp: now,
            type: touch.type,
        });

        if (this.touchEvents.length > this.maxHistory) {
            this.touchEvents.shift();
        }

        this.analyzeFrame(now);
    }

    analyzeFrame(now) {
        const dt = now - this.lastFrameTime;
        
        if (dt > 0) {
            const fps = 1000 / dt;
            
            if (fps < 30) {
                this.onLowFrameRate(fps);
            }
        }
        
        this.lastFrameTime = now;
        this.frameCount++;
    }

    onLowFrameRate(fps) {
        console.warn(`Low frame rate detected: ${fps.toFixed(1)} FPS`);
    }

    getAnalysis() {
        if (this.touchEvents.length < 2) {
            return null;
        }

        const intervals = [];
        for (let i = 1; i < this.touchEvents.length; i++) {
            intervals.push(
                this.touchEvents[i].timestamp - this.touchEvents[i - 1].timestamp
            );
        }

        const avgInterval = intervals.reduce((a, b) => a + b, 0) / intervals.length;
        const variance = intervals.reduce((sum, val) => {
            return sum + Math.pow(val - avgInterval, 2);
        }, 0) / intervals.length;

        return {
            totalEvents: this.touchEvents.length,
            averageInterval: avgInterval,
            variance: variance,
            isSmooth: variance < 100,
        };
    }

    clear() {
        this.touchEvents = [];
        this.frameCount = 0;
    }
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = { MobilePerformanceOptimizer, TouchPerformanceAnalyzer };
}

if (typeof window !== 'undefined') {
    window.MobilePerformanceOptimizer = MobilePerformanceOptimizer;
    window.TouchPerformanceAnalyzer = TouchPerformanceAnalyzer;
}
