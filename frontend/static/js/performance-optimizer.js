const FrontendPerformanceOptimizer = (function() {
    'use strict';

    class CodeSplitter {
        constructor() {
            this.moduleCache = new Map();
            this.loadingModules = new Set();
            this.dependencyGraph = new Map();
        }

        registerModule(name, dependencies = [], factory) {
            this.dependencyGraph.set(name, {
                dependencies,
                factory,
                instance: null,
                loaded: false
            });
        }

        async loadModule(name) {
            if (this.moduleCache.has(name)) {
                const module = this.moduleCache.get(name);
                return module.loaded ? module.instance : this.initializeModule(name);
            }

            if (this.loadingModules.has(name)) {
                return new Promise((resolve) => {
                    const checkLoaded = setInterval(() => {
                        if (this.moduleCache.has(name) && this.moduleCache.get(name).loaded) {
                            clearInterval(checkLoaded);
                            resolve(this.moduleCache.get(name).instance);
                        }
                    }, 10);
                });
            }

            this.loadingModules.add(name);

            try {
                const moduleInfo = this.dependencyGraph.get(name);
                if (!moduleInfo) {
                    throw new Error(`Module ${name} not found`);
                }

                for (const dep of moduleInfo.dependencies) {
                    if (!this.moduleCache.has(dep) || !this.moduleCache.get(dep).loaded) {
                        await this.loadModule(dep);
                    }
                }

                const instance = await this.initializeModule(name);
                this.loadingModules.delete(name);
                return instance;
            } catch (error) {
                this.loadingModules.delete(name);
                throw error;
            }
        }

        async initializeModule(name) {
            const moduleInfo = this.dependencyGraph.get(name);
            if (!moduleInfo) {
                throw new Error(`Module ${name} not found`);
            }

            const deps = moduleInfo.dependencies.map(dep => {
                return this.moduleCache.get(dep)?.instance;
            });

            const instance = await moduleInfo.factory(...deps);
            this.moduleCache.set(name, {
                ...moduleInfo,
                instance,
                loaded: true
            });

            return instance;
        }

        preloadModule(name) {
            if (!this.loadingModules.has(name)) {
                this.loadModule(name).catch(err => {
                    console.warn(`Failed to preload module ${name}:`, err);
                });
            }
        }

        isLoaded(name) {
            return this.moduleCache.has(name) && this.moduleCache.get(name).loaded;
        }

        getModule(name) {
            return this.moduleCache.get(name)?.instance;
        }

        clearCache() {
            this.moduleCache.clear();
            this.dependencyGraph.forEach((module, name) => {
                module.loaded = false;
                module.instance = null;
            });
        }
    }

    class ScriptLoader {
        constructor() {
            this.loadedScripts = new Map();
            this.loadingScripts = new Map();
            this.scriptPromises = new Map();
        }

        loadScript(src, options = {}) {
            const {
                async = true,
                defer = true,
                crossOrigin = 'anonymous',
                integrity = '',
                module = false
            } = options;

            if (this.loadedScripts.has(src)) {
                return Promise.resolve(this.loadedScripts.get(src));
            }

            if (this.scriptPromises.has(src)) {
                return this.scriptPromises.get(src);
            }

            const promise = new Promise((resolve, reject) => {
                if (this.loadingScripts.has(src)) {
                    resolve(this.loadingScripts.get(src));
                    return;
                }

                const script = document.createElement('script');
                script.src = src;
                script.async = async;
                script.defer = defer;

                if (crossOrigin) {
                    script.crossOrigin = crossOrigin;
                }

                if (integrity) {
                    script.integrity = integrity;
                }

                if (module) {
                    script.type = 'module';
                }

                script.onload = () => {
                    this.loadedScripts.set(src, script);
                    this.loadingScripts.delete(src);
                    resolve(script);
                };

                script.onerror = () => {
                    this.loadingScripts.delete(src);
                    reject(new Error(`Failed to load script: ${src}`));
                };

                this.loadingScripts.set(src, script);
                document.head.appendChild(script);
            });

            this.scriptPromises.set(src, promise);
            return promise;
        }

        preloadScript(src, options = {}) {
            const link = document.createElement('link');
            link.rel = 'preload';
            link.as = 'script';
            link.href = src;

            if (options.crossOrigin) {
                link.crossOrigin = options.crossOrigin;
            }

            document.head.appendChild(link);
        }

        loadScriptsInParallel(scripts, maxConcurrent = 4) {
            const queue = [...scripts];
            const results = [];
            let index = 0;

            const loadNext = async () => {
                if (queue.length === 0) return;

                const src = queue.shift();
                const result = await this.loadScript(src, scripts[index]?.options || {});
                results.push(result);
                index++;

                if (queue.length > 0) {
                    await loadNext();
                }
            };

            const workers = Array(Math.min(maxConcurrent, scripts.length))
                .fill(null)
                .map(() => loadNext());

            return Promise.all(workers).then(() => results);
        }

        isLoaded(src) {
            return this.loadedScripts.has(src);
        }
    }

    class PerformanceOptimizer {
        constructor() {
            this.metrics = {
                firstPaint: 0,
                firstContentfulPaint: 0,
                firstMeaningfulPaint: 0,
                domContentLoaded: 0,
                loadComplete: 0,
                resourceLoadTimes: new Map(),
                apiCallTimes: new Map(),
                interactionTimes: [],
                memoryUsage: []
            };

            this.config = {
                enableCodeSplitting: true,
                enableLazyLoading: true,
                enablePrefetching: true,
                enableResourceHints: true,
                enableCaching: true,
                prefetchThreshold: 0.8,
                lazyLoadThreshold: 0.2,
                criticalCssInline: true,
                scriptDefer: true,
                maxConcurrentScripts: 4,
                enableModulePreload: true
            };

            this.observers = [];
            this.codeSplitter = new CodeSplitter();
            this.scriptLoader = new ScriptLoader();
            this.init();
        }

        init() {
            if (typeof window === 'undefined') return;

            this.setupPerformanceObserver();
            this.setupResourceHints();
            this.setupLazyLoading();
            this.setupCodeSplitting();
            this.optimizeCriticalPath();
            this.setupServiceWorker();
            this.recordInitialMetrics();
            this.setupInteractionTracking();
            this.setupMemoryMonitoring();

            document.addEventListener('DOMContentLoaded', () => {
                this.metrics.domContentLoaded = performance.now();
            });

            window.addEventListener('load', () => {
                this.metrics.loadComplete = performance.now();
                this.optimizeImages();
                this.prefetchCriticalResources();
            });
        }

        setupPerformanceObserver() {
            if ('PerformanceObserver' in window) {
                try {
                    const paintObserver = new PerformanceObserver((list) => {
                        for (const entry of list.getEntries()) {
                            if (entry.name === 'first-paint') {
                                this.metrics.firstPaint = entry.startTime;
                            } else if (entry.name === 'first-contentful-paint') {
                                this.metrics.firstContentfulPaint = entry.startTime;
                            }
                        }
                    });
                    paintObserver.observe({ entryTypes: ['paint'] });

                    const resourceObserver = new PerformanceObserver((list) => {
                        for (const entry of list.getEntries()) {
                            this.metrics.resourceLoadTimes.set(entry.name, {
                                duration: entry.duration,
                                size: entry.transferSize || 0,
                                type: entry.initiatorType
                            });
                        }
                    });
                    resourceObserver.observe({ entryTypes: ['resource'] });

                    this.observers.push(paintObserver, resourceObserver);
                } catch (e) {
                    console.warn('Performance Observer setup failed:', e);
                }
            }
        }

        setupResourceHints() {
            if (!this.config.enableResourceHints) return;

            const hints = [
                { rel: 'dns-prefetch', href: '//cdn.example.com', crossorigin: true },
                { rel: 'preconnect', href: '//cdn.example.com', crossorigin: true },
                { rel: 'preload', as: 'script', href: '/static/js/main.js' },
                { rel: 'preload', as: 'style', href: '/static/css/main.css' }
            ];

            hints.forEach(hint => {
                const link = document.createElement('link');
                link.rel = hint.rel;
                if (hint.href) link.href = hint.href;
                if (hint.as) link.as = hint.as;
                if (hint.crossorigin) link.crossOrigin = 'anonymous';
                document.head.appendChild(link);
            });
        }

        setupLazyLoading() {
            if (!this.config.enableLazyLoading) return;

            if ('IntersectionObserver' in window) {
                const lazyImages = document.querySelectorAll('img[data-src]');
                const imageObserver = new IntersectionObserver((entries) => {
                    entries.forEach(entry => {
                        if (entry.isIntersecting) {
                            const img = entry.target;
                            if (img.dataset.src) {
                                img.src = img.dataset.src;
                                img.removeAttribute('data-src');
                                imageObserver.unobserve(img);
                            }
                        }
                    });
                }, {
                    rootMargin: '50px 0px',
                    threshold: this.config.lazyLoadThreshold
                });

                lazyImages.forEach(img => imageObserver.observe(img));
                this.observers.push(imageObserver);
            }

            const lazyComponents = document.querySelectorAll('[data-lazy-component]');
            lazyComponents.forEach(component => {
                this.setupComponentLazyLoad(component);
            });
        }

        setupComponentLazyLoad(component) {
            const observer = new IntersectionObserver((entries) => {
                entries.forEach(entry => {
                    if (entry.isIntersecting) {
                        const componentName = component.dataset.lazyComponent;
                        this.loadComponent(componentName, component);
                        observer.unobserve(component);
                    }
                });
            }, {
                rootMargin: '100px 0px'
            });

            observer.observe(component);
        }

        loadComponent(name, container) {
            const componentPath = `/static/js/components/${name}.js`;

            this.loadScript(componentPath).then(() => {
                if (window.Components && window.Components[name]) {
                    window.Components[name].init(container);
                }
            }).catch(err => {
                console.error(`Failed to load component ${name}:`, err);
            });
        }

        setupCodeSplitting() {
            if (!this.config.enableCodeSplitting) return;

            const routes = this.detectRoutes();
            routes.forEach(route => {
                this.setupRoutePrefetch(route);
            });

            this.setupClickPrefetching();
        }

        detectRoutes() {
            const routes = [];
            const links = document.querySelectorAll('a[href^="/"]');

            links.forEach(link => {
                const href = link.getAttribute('href');
                if (href && !routes.includes(href)) {
                    routes.push(href);
                }
            });

            return routes;
        }

        setupRoutePrefetch(route) {
            const link = document.createElement('link');
            link.rel = 'prefetch';
            link.href = route;
            link.as = 'document';
            document.head.appendChild(link);
        }

        setupClickPrefetching() {
            const links = document.querySelectorAll('a[href^="/"]');

            links.forEach(link => {
                link.addEventListener('mouseenter', () => {
                    const href = link.getAttribute('href');
                    if (href) {
                        this.prefetchRoute(href);
                    }
                }, { passive: true });
            });
        }

        prefetchRoute(route) {
            const link = document.createElement('link');
            link.rel = 'prefetch';
            link.href = route;
            link.as = 'document';
            document.head.appendChild(link);
        }

        optimizeCriticalPath() {
            this.inlineCriticalCSS();
            this.deferNonCriticalScripts();
            this.optimizeFontLoading();
        }

        inlineCriticalCSS() {
            if (!this.config.criticalCssInline) return;

            const criticalCSS = this.extractCriticalCSS();
            if (criticalCSS) {
                const style = document.createElement('style');
                style.textContent = criticalCSS;
                document.head.insertBefore(style, document.head.firstChild);
            }
        }

        extractCriticalCSS() {
            return `
                body { margin: 0; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; }
                .captcha-container { visibility: hidden; }
                .captcha-loaded .captcha-container { visibility: visible; }
            `;
        }

        deferNonCriticalScripts() {
            if (!this.config.scriptDefer) return;

            const scripts = document.querySelectorAll('script[src]');
            scripts.forEach(script => {
                if (!script.hasAttribute('data-critical')) {
                    script.setAttribute('defer', '');
                }
            });
        }

        optimizeFontLoading() {
            const fonts = document.querySelectorAll('link[rel="stylesheet"]');
            fonts.forEach(font => {
                if (font.href.includes('fonts.googleapis.com')) {
                    font.setAttribute('rel', 'preload');
                    font.setAttribute('as', 'style');
                    font.setAttribute('rel', 'stylesheet');
                }
            });
        }

        setupServiceWorker() {
            if ('serviceWorker' in navigator && this.config.enableCaching) {
                navigator.serviceWorker.register('/service-worker.js').then(registration => {
                    console.log('Service Worker registered:', registration.scope);
                }).catch(err => {
                    console.warn('Service Worker registration failed:', err);
                });
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
            }
        }

        optimizeImages() {
            const images = document.querySelectorAll('img');

            images.forEach(img => {
                if (!img.hasAttribute('loading')) {
                    img.setAttribute('loading', 'lazy');
                }

                if (img.hasAttribute('data-srcset')) {
                    img.setAttribute('srcset', img.dataset.srcset);
                }

                if (img.hasAttribute('data-sizes')) {
                    img.setAttribute('sizes', img.dataset.sizes);
                }
            });
        }

        prefetchCriticalResources() {
            if (!this.config.enablePrefetching) return;

            const criticalPaths = ['/api/user', '/api/config', '/static/js/main.js'];
            criticalPaths.forEach(path => {
                this.prefetchResource(path);
            });
        }

        prefetchResource(url) {
            const link = document.createElement('link');
            link.rel = 'prefetch';
            link.href = url;
            link.as = this.getResourceType(url);
            document.head.appendChild(link);
        }

        getResourceType(url) {
            if (url.endsWith('.js')) return 'script';
            if (url.endsWith('.css')) return 'style';
            if (url.endsWith('.jpg') || url.endsWith('.png') || url.endsWith('.webp')) return 'image';
            return 'fetch';
        }

        loadScript(src, async = true) {
            return new Promise((resolve, reject) => {
                const script = document.createElement('script');
                script.src = src;
                script.async = async;

                script.onload = () => resolve(script);
                script.onerror = () => reject(new Error(`Failed to load script: ${src}`));

                document.head.appendChild(script);
            });
        }

        loadStylesheet(href) {
            return new Promise((resolve, reject) => {
                const link = document.createElement('link');
                link.rel = 'stylesheet';
                link.href = href;

                link.onload = () => resolve(link);
                link.onerror = () => reject(new Error(`Failed to load stylesheet: ${href}`));

                document.head.appendChild(link);
            });
        }

        getMetrics() {
            return {
                ...this.metrics,
                resourceLoadTimes: Array.from(this.metrics.resourceLoadTimes.entries()),
                apiCallTimes: Array.from(this.metrics.apiCallTimes.entries())
            };
        }

        getLoadTime() {
            return this.metrics.loadComplete - this.metrics.firstPaint;
        }

        getFirstContentfulPaint() {
            return this.metrics.firstContentfulPaint;
        }

        getLargestContentfulPaint() {
            if ('PerformanceObserver' in window) {
                return new Promise(resolve => {
                    try {
                        const observer = new PerformanceObserver((list) => {
                            const entries = list.getEntries();
                            const lastEntry = entries[entries.length - 1];
                            resolve(lastEntry.startTime);
                            observer.disconnect();
                        });
                        observer.observe({ entryTypes: ['largest-contentful-paint'] });
                    } catch (e) {
                        resolve(0);
                    }
                });
            }
            return Promise.resolve(0);
        }

        getCumulativeLayoutShift() {
            if ('PerformanceObserver' in window) {
                return new Promise(resolve => {
                    try {
                        const observer = new PerformanceObserver((list) => {
                            let totalScore = 0;
                            for (const entry of list.getEntries()) {
                                if (!entry.hadRecentInput) {
                                    totalScore += entry.value;
                                }
                            }
                            resolve(totalScore);
                            observer.disconnect();
                        });
                        observer.observe({ entryTypes: ['layout-shift'] });
                    } catch (e) {
                        resolve(0);
                    }
                });
            }
            return Promise.resolve(0);
        }

        setupInteractionTracking() {
            const interactionTypes = ['click', 'keypress', 'touchstart', 'scroll'];

            interactionTypes.forEach(type => {
                document.addEventListener(type, () => {
                    this.metrics.interactionTimes.push({
                        type,
                        time: performance.now()
                    });
                }, { passive: true });
            });

            if ('EventTiming' in window) {
                try {
                    const observer = new PerformanceObserver((list) => {
                        for (const entry of list.getEntries()) {
                            if (entry.interactionId) {
                                this.metrics.interactionTimes.push({
                                    type: entry.name,
                                    duration: entry.duration,
                                    time: entry.startTime
                                });
                            }
                        }
                    });
                    observer.observe({ entryTypes: ['event'] });
                    this.observers.push(observer);
                } catch (e) {
                    console.warn('Event timing not supported:', e);
                }
            }
        }

        setupMemoryMonitoring() {
            if (performance.memory) {
                setInterval(() => {
                    this.metrics.memoryUsage.push({
                        usedJSHeapSize: performance.memory.usedJSHeapSize,
                        totalJSHeapSize: performance.memory.totalJSHeapSize,
                        jsHeapSizeLimit: performance.memory.jsHeapSizeLimit,
                        timestamp: Date.now()
                    });

                    if (this.metrics.memoryUsage.length > 100) {
                        this.metrics.memoryUsage.shift();
                    }
                }, 5000);
            }
        }

        getMemoryUsage() {
            if (performance.memory) {
                return {
                    used: (performance.memory.usedJSHeapSize / 1048576).toFixed(2) + ' MB',
                    total: (performance.memory.totalJSHeapSize / 1048576).toFixed(2) + ' MB',
                    limit: (performance.memory.jsHeapSizeLimit / 1048576).toFixed(2) + ' MB'
                };
            }
            return null;
        }

        reportPerformance() {
            const metrics = this.getMetrics();
            console.group('🚀 Performance Report');
            console.log('First Paint:', metrics.firstPaint.toFixed(2) + 'ms');
            console.log('First Contentful Paint:', metrics.firstContentfulPaint.toFixed(2) + 'ms');
            console.log('DOM Content Loaded:', metrics.domContentLoaded.toFixed(2) + 'ms');
            console.log('Load Complete:', metrics.loadComplete.toFixed(2) + 'ms');
            console.log('Total Load Time:', this.getLoadTime().toFixed(2) + 'ms');

            const memoryUsage = this.getMemoryUsage();
            if (memoryUsage) {
                console.log('Memory Usage:', memoryUsage);
            }

            console.log('Interaction Count:', metrics.interactionTimes.length);
            console.groupEnd();

            if (typeof fetch !== 'undefined') {
                fetch('/api/metrics', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        metrics: metrics,
                        timestamp: Date.now(),
                        url: window.location.href,
                        memoryUsage: memoryUsage
                    })
                }).catch(() => {});
            }
        }

        destroy() {
            this.observers.forEach(observer => {
                observer.disconnect();
            });
            this.observers = [];
        }
    }

    class ResourceOptimizer {
        constructor() {
            this.cache = new Map();
            this.preloadedResources = new Set();
        }

        preload(url, options = {}) {
            if (this.preloadedResources.has(url)) {
                return Promise.resolve();
            }

            const { type = 'auto', as = null } = options;

            return new Promise((resolve, reject) => {
                const link = document.createElement('link');
                link.rel = 'preload';

                if (type === 'fetch' || as === 'fetch') {
                    link.rel = 'preload';
                    link.href = url;
                    link.as = 'fetch';
                    link.crossOrigin = 'anonymous';
                } else if (type === 'image' || url.match(/\.(jpg|png|gif|webp|svg)$/)) {
                    link.as = 'image';
                    link.href = url;
                } else if (type === 'script' || url.endsWith('.js')) {
                    link.as = 'script';
                    link.href = url;
                } else if (type === 'style' || url.endsWith('.css')) {
                    link.as = 'style';
                    link.href = url;
                } else {
                    link.href = url;
                }

                link.onload = () => {
                    this.preloadedResources.add(url);
                    resolve();
                };

                link.onerror = () => reject(new Error(`Failed to preload: ${url}`));

                document.head.appendChild(link);
            });
        }

        prefetch(url) {
            const link = document.createElement('link');
            link.rel = 'prefetch';
            link.href = url;
            link.as = this.guessResourceType(url);
            document.head.appendChild(link);
        }

        guessResourceType(url) {
            if (url.match(/\.(jpg|png|gif|webp|svg)$/)) return 'image';
            if (url.match(/\.js$/)) return 'script';
            if (url.match(/\.css$/)) return 'style';
            if (url.match(/\.html?$/)) return 'document';
            return 'fetch';
        }

        cacheResource(url, data) {
            this.cache.set(url, {
                data,
                timestamp: Date.now(),
                ttl: 5 * 60 * 1000
            });
        }

        getCachedResource(url) {
            const cached = this.cache.get(url);
            if (!cached) return null;

            if (Date.now() - cached.timestamp > cached.ttl) {
                this.cache.delete(url);
                return null;
            }

            return cached.data;
        }

        clearCache() {
            this.cache.clear();
        }
    }

    class CSSOptimizer {
        constructor() {
            this.loadedStylesheets = new Set();
        }

        async loadStylesheet(href, options = {}) {
            const { media = 'all', preload = true } = options;

            if (this.loadedStylesheets.has(href)) {
                return Promise.resolve();
            }

            if (preload) {
                const preloadLink = document.createElement('link');
                preloadLink.rel = 'preload';
                preloadLink.as = 'style';
                preloadLink.href = href;
                document.head.appendChild(preloadLink);
            }

            return new Promise((resolve, reject) => {
                const link = document.createElement('link');
                link.rel = 'stylesheet';
                link.href = href;
                link.media = media;

                link.onload = () => {
                    this.loadedStylesheets.add(href);
                    resolve(link);
                };

                link.onerror = () => reject(new Error(`Failed to load stylesheet: ${href}`));

                document.head.appendChild(link);
            });
        }

        async loadStylesheetsInParallel(stylesheets, maxConcurrent = 3) {
            const results = [];
            const batches = [];

            for (let i = 0; i < stylesheets.length; i += maxConcurrent) {
                batches.push(stylesheets.slice(i, i + maxConcurrent));
            }

            for (const batch of batches) {
                const batchResults = await Promise.all(
                    batch.map(stylesheet => this.loadStylesheet(stylesheet.href, stylesheet.options || {}))
                );
                results.push(...batchResults);
            }

            return results;
        }

        inlineCriticalCSS(css, id = 'critical-css') {
            let style = document.getElementById(id);

            if (!style) {
                style = document.createElement('style');
                style.id = id;
                document.head.insertBefore(style, document.head.firstChild);
            }

            style.textContent = css;
        }

        deferStylesheet(href) {
            const link = document.createElement('link');
            link.rel = 'stylesheet';
            link.href = href;
            link.media = 'print';
            link.onload = () => {
                link.media = 'all';
            };
            document.head.appendChild(link);
        }

        async loadStylesheetOnIdle(href, options = {}) {
            if ('requestIdleCallback' in window) {
                return new Promise((resolve, reject) => {
                    requestIdleCallback(async () => {
                        try {
                            await this.loadStylesheet(href, options);
                            resolve();
                        } catch (error) {
                            reject(error);
                        }
                    }, { timeout: 3000 });
                });
            } else {
                return this.loadStylesheet(href, options);
            }
        }
    }

    class LazyLoader {
        constructor() {
            this.observer = null;
            this.loadedElements = new Set();
            this.loadHandlers = new Map();
        }

        init(options = {}) {
            const defaultOptions = {
                root: null,
                rootMargin: '50px 0px',
                threshold: 0.2
            };

            this.options = { ...defaultOptions, ...options };

            if ('IntersectionObserver' in window) {
                this.observer = new IntersectionObserver(
                    this.handleIntersection.bind(this),
                    this.options
                );
            }
        }

        handleIntersection(entries) {
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    this.loadElement(entry.target);
                    this.observer.unobserve(entry.target);
                }
            });
        }

        loadElement(element) {
            const id = this.getElementId(element);
            if (this.loadedElements.has(id)) return;

            const handler = this.loadHandlers.get(id);
            if (handler) {
                handler(element);
                this.loadedElements.add(id);
                this.loadHandlers.delete(id);
            }
        }

        register(id, handler) {
            this.loadHandlers.set(id, handler);
        }

        observe(element) {
            if (this.observer) {
                this.observer.observe(element);
            }
        }

        getElementId(element) {
            return element.id || element.dataset.lazyId || Math.random().toString(36).substr(2, 9);
        }
    }

    let optimizer = null;
    let resourceOptimizer = null;
    let lazyLoader = null;
    let cssOptimizer = null;
    let codeSplitter = null;
    let scriptLoader = null;

    function init(options = {}) {
        optimizer = new PerformanceOptimizer();
        resourceOptimizer = new ResourceOptimizer();
        lazyLoader = new LazyLoader();
        cssOptimizer = new CSSOptimizer();
        codeSplitter = new CodeSplitter();
        scriptLoader = new ScriptLoader();
        lazyLoader.init();

        if (options.autoReport) {
            window.addEventListener('load', () => {
                setTimeout(() => optimizer.reportPerformance(), 1000);
            });
        }

        return {
            optimizer,
            resourceOptimizer,
            lazyLoader,
            cssOptimizer,
            codeSplitter,
            scriptLoader
        };
    }

    return {
        init,
        getOptimizer: () => optimizer,
        getResourceOptimizer: () => resourceOptimizer,
        getLazyLoader: () => lazyLoader,
        getCSSOptimizer: () => cssOptimizer,
        getCodeSplitter: () => codeSplitter,
        getScriptLoader: () => scriptLoader,
        PerformanceOptimizer,
        ResourceOptimizer,
        LazyLoader,
        CSSOptimizer,
        CodeSplitter,
        ScriptLoader
    };
})();

if (typeof module !== 'undefined' && module.exports) {
    module.exports = FrontendPerformanceOptimizer;
}

if (typeof window !== 'undefined') {
    window.PerformanceOptimizer = FrontendPerformanceOptimizer;

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', () => {
            FrontendPerformanceOptimizer.init({ autoReport: true });
        });
    } else {
        FrontendPerformanceOptimizer.init({ autoReport: true });
    }
}
