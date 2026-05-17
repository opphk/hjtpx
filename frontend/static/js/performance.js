const PerformanceOptimizer = {
    config: {
        targetLoadTime: 1000,
        enableResourceHints: true,
        enableLazyLoading: true,
        enableCompression: true,
        resourcePriority: {
            high: ['bootstrap.min.css', 'main.css'],
            medium: ['captcha.js', 'i18n.js'],
            low: ['analytics.js', 'third-party.js']
        }
    },
    metrics: {
        pageLoadTime: 0,
        firstContentfulPaint: 0,
        largestContentfulPaint: 0,
        firstInputDelay: 0,
        cumulativeLayoutShift: 0,
        domContentLoaded: 0,
        domInteractive: 0,
        resourceTimings: [],
        customTimings: new Map()
    },
    observers: [],

    async init(options = {}) {
        this.config = { ...this.config, ...options };
        this.setupPerformanceObserver();
        this.setupResourceHints();
        this.setupLazyLoading();
        this.injectCriticalCSS();
        this.setupCompressionHints();
        this.setupFPSMonitoring();
        this.startMetricsCollection();
        return this;
    },

    setupPerformanceObserver() {
        if ('PerformanceObserver' in window) {
            this.setupNavigationObserver();
            this.setupPaintObserver();
            this.setupResourceObserver();
            this.setupLayoutShiftObserver();
            this.setupLongTaskObserver();
        }
        this.setupFirstInputDelay();
    },

    setupNavigationObserver() {
        try {
            const navObserver = new PerformanceObserver((list) => {
                for (const entry of list.getEntries()) {
                    if (entry.entryType === 'navigation') {
                        this.metrics.domContentLoaded = entry.domContentLoadedEventEnd;
                        this.metrics.domInteractive = entry.domInteractive;
                        this.metrics.pageLoadTime = entry.loadEventEnd;
                        this.updateLoadStatus();
                    }
                }
            });
            navObserver.observe({ entryTypes: ['navigation'] });
            this.observers.push(navObserver);
        } catch (e) {
            console.warn('Navigation observer not supported');
        }
    },

    setupPaintObserver() {
        try {
            const paintObserver = new PerformanceObserver((list) => {
                for (const entry of list.getEntries()) {
                    if (entry.name === 'first-contentful-paint') {
                        this.metrics.firstContentfulPaint = entry.startTime;
                    }
                    this.metrics.paintEntries = this.metrics.paintEntries || [];
                    this.metrics.paintEntries.push({
                        name: entry.name,
                        startTime: entry.startTime
                    });
                }
            });
            paintObserver.observe({ entryTypes: ['paint'] });
            this.observers.push(paintObserver);
        } catch (e) {
            console.warn('Paint observer not supported');
        }
    },

    setupResourceObserver() {
        try {
            const resourceObserver = new PerformanceObserver((list) => {
                for (const entry of list.getEntries()) {
                    this.metrics.resourceTimings.push({
                        name: entry.name,
                        duration: entry.duration,
                        transferSize: entry.transferSize,
                        initiatorType: entry.initiatorType,
                        startTime: entry.startTime
                    });
                }
            });
            resourceObserver.observe({ entryTypes: ['resource'] });
            this.observers.push(resourceObserver);
        } catch (e) {
            console.warn('Resource observer not supported');
        }
    },

    setupLayoutShiftObserver() {
        try {
            let layoutShiftValue = 0;
            const shiftObserver = new PerformanceObserver((list) => {
                for (const entry of list.getEntries()) {
                    if (!entry.hadRecentInput) {
                        layoutShiftValue += entry.value;
                        this.metrics.cumulativeLayoutShift = layoutShiftValue;
                    }
                }
            });
            shiftObserver.observe({ entryTypes: ['layout-shift'] });
            this.observers.push(shiftObserver);
        } catch (e) {
            console.warn('Layout shift observer not supported');
        }
    },

    setupLongTaskObserver() {
        try {
            const longTaskObserver = new PerformanceObserver((list) => {
                for (const entry of list.getEntries()) {
                    this.dispatchEvent('longtask', {
                        duration: entry.duration,
                        startTime: entry.startTime,
                        attribution: entry.attribution
                    });
                }
            });
            longTaskObserver.observe({ entryTypes: ['longtask'] });
            this.observers.push(longTaskObserver);
        } catch (e) {
            console.warn('Long task observer not supported');
        }
    },

    setupFirstInputDelay() {
        const firstInputDelay = new PerformanceObserver((list) => {
            for (const entry of list.getEntries()) {
                this.metrics.firstInputDelay = entry.processingStart - entry.startTime;
            }
        });

        try {
            firstInputDelay.observe({ entryTypes: ['first-input'] });
            this.observers.push(firstInputDelay);
        } catch (e) {
            this.measureFirstInputDelay();
        }
    },

    measureFirstInputDelay() {
        const firstInputTime = parseInt(localStorage.getItem('firstInputTime') || '0');
        if (firstInputTime === 0) {
            document.addEventListener('pointerdown', () => {
                if (!this.metrics.firstInputDelay) {
                    const delay = performance.now() - (parseInt(localStorage.getItem('firstInputTime') || '0') || performance.now());
                    this.metrics.firstInputDelay = delay;
                }
            }, { once: true });
        }
    },

    setupResourceHints() {
        if (!this.config.enableResourceHints) return;

        const resources = document.querySelectorAll('link[href], script[src], img[src]');

        resources.forEach(resource => {
            if (resource.tagName === 'LINK' && resource.rel === 'stylesheet') {
                resource.setAttribute('rel', 'preload');
                resource.setAttribute('as', 'style');
            }

            if (resource.tagName === 'SCRIPT') {
                resource.setAttribute('fetchpriority', 'low');
            }

            if (resource.tagName === 'IMG') {
                resource.setAttribute('fetchpriority', this.getImagePriority(resource));
            }
        });

        this.injectDNS_prefetch();
        this.injectPreconnect();
    },

    getImagePriority(img) {
        const aboveFold = this.isAboveFold(img);
        return aboveFold ? 'high' : 'low';
    },

    isAboveFold(element) {
        const rect = element.getBoundingClientRect();
        return rect.top < window.innerHeight && rect.bottom > 0;
    },

    injectDNS_prefetch() {
        const domains = ['cdn.bootcdn.net', 'cdnjs.cloudflare.com'];

        domains.forEach(domain => {
            const link = document.createElement('link');
            link.rel = 'dns-prefetch';
            link.href = `https://${domain}`;
            link.crossOrigin = 'anonymous';
            document.head.appendChild(link);
        });
    },

    injectPreconnect() {
        const domains = [
            { href: 'https://cdn.bootcdn.net', crossOrigin: 'anonymous' },
            { href: 'https://cdnjs.cloudflare.com', crossOrigin: 'anonymous' }
        ];

        domains.forEach(({ href, crossOrigin }) => {
            const link = document.createElement('link');
            link.rel = 'preconnect';
            link.href = href;
            link.crossOrigin = crossOrigin;
            document.head.appendChild(link);
        });
    },

    setupLazyLoading() {
        if (!this.config.enableLazyLoading) return;

        const images = document.querySelectorAll('img[src]');
        images.forEach(img => {
            if (!img.hasAttribute('loading')) {
                img.setAttribute('loading', 'lazy');
            }

            if (!img.hasAttribute('decoding')) {
                img.setAttribute('decoding', 'async');
            }
        });

        const iframes = document.querySelectorAll('iframe[src]');
        iframes.forEach(iframe => {
            if (!iframe.hasAttribute('loading')) {
                iframe.setAttribute('loading', 'lazy');
            }
        });
    },

    injectCriticalCSS() {
        const criticalStyles = `
            .loading-critical { opacity: 0; }
            .loading-critical.loaded { opacity: 1; transition: opacity 0.3s ease; }
            .skeleton { background: linear-gradient(90deg, #f0f0f0 25%, #e0e0e0 50%, #f0f0f0 75%); background-size: 200% 100%; animation: skeleton-loading 1.5s infinite; }
            @keyframes skeleton-loading { 0% { background-position: 200% 0; } 100% { background-position: -200% 0; } }
        `;

        const style = document.createElement('style');
        style.textContent = criticalStyles;
        document.head.appendChild(style);
    },

    setupCompressionHints() {
        if (!this.config.enableCompression) return;

        const links = document.querySelectorAll('link[href]');
        links.forEach(link => {
            if (link.as === 'style') {
                link.setAttribute('crossorigin', 'anonymous');
            }
        });
    },

    setupFPSMonitoring() {
        let lastTime = performance.now();
        let frames = 0;
        let lastFrameTime = performance.now();
        this.metrics.fps = 60;
        this.metrics.frameDrops = 0;

        const measureFPS = () => {
            const currentTime = performance.now();
            frames++;

            if (currentTime >= lastTime + 1000) {
                const fps = Math.round((frames * 1000) / (currentTime - lastTime));
                this.metrics.fps = fps;
                frames = 0;
                lastTime = currentTime;

                if (fps < 30) {
                    this.dispatchEvent('fps:low', { fps });
                }
            }

            const deltaTime = currentTime - lastFrameTime;
            if (deltaTime > 20) {
                this.metrics.frameDrops++;
            }
            lastFrameTime = currentTime;

            requestAnimationFrame(measureFPS);
        };

        requestAnimationFrame(measureFPS);
    },

    startMetricsCollection() {
        window.addEventListener('load', () => {
            setTimeout(() => {
                this.collectMetrics();
            }, 0);
        });
    },

    collectMetrics() {
        const timing = performance.timing;
        const navigation = performance.getEntriesByType('navigation')[0];

        this.metrics.pageLoadTime = timing.loadEventEnd - timing.navigationStart;
        this.metrics.domContentLoaded = timing.domContentLoadedEventEnd - timing.navigationStart;
        this.metrics.domInteractive = timing.domInteractive - timing.navigationStart;

        if (navigation) {
            this.metrics.transferSize = navigation.transferSize || 0;
            this.metrics.encodedBodySize = navigation.encodedBodySize || 0;
            this.metrics.decodedBodySize = navigation.decodedBodySize || 0;
        }

        this.calculateWebVitals();
        this.updatePerformanceMetrics();
        this.dispatchEvent('metrics:collected', this.metrics);
    },

    calculateWebVitals() {
        this.metrics.webVitals = {
            lcp: this.getLargestContentfulPaint(),
            fid: this.metrics.firstInputDelay,
            cls: this.metrics.cumulativeLayoutShift,
            fcp: this.metrics.firstContentfulPaint,
            ttfb: this.getTimeToFirstByte()
        };
    },

    getLargestContentfulPaint() {
        const entries = performance.getEntriesByType('largest-contentful-paint');
        if (entries.length > 0) {
            return entries[entries.length - 1].startTime;
        }
        return 0;
    },

    getTimeToFirstByte() {
        const entries = performance.getEntriesByType('navigation');
        if (entries.length > 0) {
            return entries[0].responseStart;
        }
        return 0;
    },

    updatePerformanceMetrics() {
        const metricsElement = document.getElementById('performance-metrics');
        if (!metricsElement) return;

        const loadTimeClass = this.getLoadTimeClass(this.metrics.pageLoadTime);
        const fpsClass = this.getFPSClass(this.metrics.fps);

        metricsElement.innerHTML = `
            <div class="metric-item">
                <span class="metric-label">页面加载</span>
                <span class="metric-value ${loadTimeClass}">${Math.round(this.metrics.pageLoadTime)}ms</span>
            </div>
            <div class="metric-item">
                <span class="metric-label">首次内容绘制</span>
                <span class="metric-value ${this.getLoadTimeClass(this.metrics.firstContentfulPaint)}">${Math.round(this.metrics.firstContentfulPaint)}ms</span>
            </div>
            <div class="metric-item">
                <span class="metric-label">FPS</span>
                <span class="metric-value ${fpsClass}">${this.metrics.fps}</span>
            </div>
            <div class="metric-item">
                <span class="metric-label">CLS</span>
                <span class="metric-value">${this.metrics.cumulativeLayoutShift.toFixed(3)}</span>
            </div>
        `;
    },

    getLoadTimeClass(time) {
        if (time < 500) return 'good';
        if (time < 1000) return 'warning';
        return 'bad';
    },

    getFPSClass(fps) {
        if (fps >= 55) return 'good';
        if (fps >= 30) return 'warning';
        return 'bad';
    },

    startTiming(label) {
        this.metrics.customTimings.set(label, {
            startTime: performance.now(),
            marks: []
        });
    },

    endTiming(label) {
        const timing = this.metrics.customTimings.get(label);
        if (timing) {
            timing.endTime = performance.now();
            timing.duration = timing.endTime - timing.startTime;
            this.dispatchEvent('timing:end', {
                label,
                duration: timing.duration
            });
        }
        return timing?.duration || 0;
    },

    mark(name) {
        performance.mark(name);
        const timing = this.metrics.customTimings.get('current');
        if (timing) {
            timing.marks.push({
                name,
                time: performance.now()
            });
        }
    },

    measure(name, startMark, endMark) {
        try {
            performance.measure(name, startMark, endMark);
        } catch (e) {
            console.warn('Performance measure failed:', e);
        }
    },

    getResourceStats() {
        const stats = {
            total: 0,
            byType: {},
            totalSize: 0,
            critical: [],
            nonCritical: []
        };

        this.metrics.resourceTimings.forEach(resource => {
            stats.total++;
            stats.byType[resource.initiatorType] = (stats.byType[resource.initiatorType] || 0) + 1;
            stats.totalSize += resource.transferSize || 0;

            if (this.isCriticalResource(resource.name)) {
                stats.critical.push(resource);
            } else {
                stats.nonCritical.push(resource);
            }
        });

        return stats;
    },

    isCriticalResource(url) {
        const criticalPatterns = [
            'bootstrap.min.css',
            'main.css',
            'captcha',
            'i18n'
        ];
        return criticalPatterns.some(pattern => url.includes(pattern));
    },

    optimizeImages() {
        const images = document.querySelectorAll('img');

        images.forEach(img => {
            const src = img.src || img.dataset.src;

            if (img.dataset.optimized) return;

            if (img.dataset.srcset) {
                img.srcset = this.optimizeSrcSet(img.dataset.srcset);
            }

            if (img.dataset.sizes) {
                img.sizes = this.optimizeSizes(img.dataset.sizes);
            }

            img.dataset.optimized = 'true';
        });
    },

    optimizeSrcSet(srcset) {
        return srcset;
    },

    optimizeSizes(sizes) {
        return sizes;
    },

    async prefetchResources(urls, priority = 'low') {
        if (!('IntersectionObserver' in window)) {
            urls.forEach(url => {
                const link = document.createElement('link');
                link.rel = 'prefetch';
                link.href = url;
                document.head.appendChild(link);
            });
            return;
        }

        const observer = new IntersectionObserver((entries) => {
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    const url = entry.target.dataset.href;
                    const link = document.createElement('link');
                    link.rel = priority === 'high' ? 'preload' : 'prefetch';
                    link.href = url;
                    document.head.appendChild(link);
                    observer.unobserve(entry.target);
                }
            });
        });

        urls.forEach(url => {
            const dummy = document.createElement('div');
            dummy.dataset.href = url;
            dummy.style.display = 'none';
            document.body.appendChild(dummy);
            observer.observe(dummy);
            setTimeout(() => dummy.remove(), 60000);
        });
    },

    createResourceHint(type, urls) {
        urls.forEach(url => {
            const link = document.createElement('link');
            link.rel = type;
            link.href = url;
            if (type === 'preload') {
                link.as = this.getResourceType(url);
            }
            document.head.appendChild(link);
        });
    },

    getResourceType(url) {
        if (url.endsWith('.css')) return 'style';
        if (url.endsWith('.js')) return 'script';
        if (url.match(/\.(jpg|jpeg|png|gif|webp|avif)/i)) return 'image';
        if (url.match(/\.(woff|woff2|ttf|otf)/i)) return 'font';
        return 'fetch';
    },

    getMetrics() {
        return {
            ...this.metrics,
            resourceStats: this.getResourceStats(),
            performanceRating: this.getPerformanceRating()
        };
    },

    getPerformanceRating() {
        const score = {
            loadTime: this.getLoadTimeScore(),
            fps: this.getFPSScore(),
            cls: this.getCLSScore(),
            resources: this.getResourceScore()
        };
        score.total = (score.loadTime + score.fps + score.cls + score.resources) / 4;
        return score;
    },

    getLoadTimeScore() {
        const time = this.metrics.pageLoadTime;
        if (time < 500) return 100;
        if (time < 1000) return 90;
        if (time < 2000) return 70;
        if (time < 3000) return 50;
        return 30;
    },

    getFPSScore() {
        const fps = this.metrics.fps;
        if (fps >= 55) return 100;
        if (fps >= 45) return 80;
        if (fps >= 30) return 60;
        return 40;
    },

    getCLSScore() {
        const cls = this.metrics.cumulativeLayoutShift;
        if (cls < 0.05) return 100;
        if (cls < 0.1) return 80;
        if (cls < 0.25) return 60;
        return 40;
    },

    getResourceScore() {
        const stats = this.getResourceStats();
        if (stats.totalSize < 100000) return 100;
        if (stats.totalSize < 500000) return 80;
        if (stats.totalSize < 1000000) return 60;
        return 40;
    },

    dispatchEvent(eventName, detail = {}) {
        const event = new CustomEvent(`performance:${eventName}`, { detail });
        document.dispatchEvent(event);
    },

    on(eventName, handler) {
        document.addEventListener(`performance:${eventName}`, (e) => handler(e.detail));
    },

    off(eventName, handler) {
        document.removeEventListener(`performance:${eventName}`, (e) => handler(e.detail));
    },

    generateReport() {
        return {
            timestamp: new Date().toISOString(),
            pageLoadTime: `${this.metrics.pageLoadTime.toFixed(2)}ms`,
            targetLoadTime: `${this.config.targetLoadTime}ms`,
            targetMet: this.metrics.pageLoadTime < this.config.targetLoadTime,
            webVitals: this.metrics.webVitals,
            resourceStats: this.getResourceStats(),
            performanceRating: this.getPerformanceRating()
        };
    }
};

if (typeof window !== 'undefined') {
    window.PerformanceOptimizer = PerformanceOptimizer;
}
