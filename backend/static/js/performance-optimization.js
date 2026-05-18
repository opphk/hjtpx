(function() {
  'use strict';

  const PERFORMANCE_CONFIG = {
    lazyLoadThreshold: 300,
    preloadBatchSize: 3,
    imageMaxWidth: 800,
    imageMaxHeight: 800,
    compressionQuality: 0.85,
    debounceDelay: 200,
    throttleDelay: 100,
    criticalRenderingBudget: 500,
    maxResourceDuration: 1000,
    criticalCssThreshold: 2000,
    preloadPriority: ['script', 'style', 'font', 'image']
  };

  class PerformanceOptimizer {
    constructor() {
      this.observer = null;
      this.imageObserver = null;
      this.loadedImages = new Set();
      this.processedElements = new Set();
      this.performanceMetrics = {
        startTime: performance.now(),
        navigationStart: 0,
        domContentLoaded: 0,
        loadComplete: 0,
        firstPaint: 0,
        firstContentfulPaint: 0,
        lcp: 0,
        fid: 0,
        cls: 0,
        resourceCount: 0,
        slowResources: []
      };
    }

    init() {
      console.log('[Performance] 初始化性能优化模块');
      
      this.performanceMetrics.startTime = performance.now();
      this.performanceMetrics.navigationStart = performance.timing?.navigationStart || 0;

      this.setupEarlyOptimizations();
      this.setupResourceHints();
      this.setupCriticalResourcePreloading();
      this.setupLazyLoading();
      this.setupImageOptimization();
      this.setupScriptDeferring();
      this.setupMemoryManagement();
      this.setupPerformanceMonitoring();
      this.injectPerformanceStyles();
      this.setupPreloadCriticalAssets();
    }

    setupEarlyOptimizations() {
      this.setupRequestIdleCallbackPolyfill();
      this.setupPerformanceMarks();
      this.setupInlineCriticalCss();
      this.setupFontPreloading();
    }

    setupInlineCriticalCss() {
      if (document.getElementById('critical-css')) {
        return;
      }

      const criticalCss = `
        .captcha-container { display: block; }
        .captcha-slider-button { touch-action: none; user-select: none; }
        .captcha-loading-overlay { display: none; }
        .captcha-canvas { max-width: 100%; height: auto; }
        @keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }
        .captcha-content.active { animation: fadeIn 0.3s ease; }
        @media (max-width: 576px) {
          .captcha-container { margin: 0; border-radius: 0; }
          .captcha-slider-container { height: 50px; }
          .captcha-slider-button { width: 46px; height: 46px; }
        }
      `;

      const style = document.createElement('style');
      style.id = 'critical-css';
      style.textContent = criticalCss;
      document.head.insertBefore(style, document.head.firstChild);
      console.log('[Performance] 关键CSS已内联');
    }

    setupFontPreloading() {
      const fontUrls = [
        'https://cdn.bootcdn.net/ajax/libs/font-awesome/6.5.1/webfonts/fa-solid-900.woff2',
        'https://cdn.bootcdn.net/ajax/libs/font-awesome/6.5.1/webfonts/fa-regular-400.woff2'
      ];

      fontUrls.forEach((url, index) => {
        const link = document.createElement('link');
        link.rel = 'preload';
        link.href = url;
        link.as = 'font';
        link.type = 'font/woff2';
        link.crossOrigin = 'anonymous';
        link.setAttribute('data-font-preload', index);
        document.head.appendChild(link);
      });

      const fontDisplayStyle = document.createElement('style');
      fontDisplayStyle.textContent = `
        @font-face {
          font-family: 'Font Awesome 6 Solid';
          font-display: swap;
          src: url('https://cdn.bootcdn.net/ajax/libs/font-awesome/6.5.1/webfonts/fa-solid-900.woff2') format('woff2');
        }
        @font-face {
          font-family: 'Font Awesome 6 Regular';
          font-display: swap;
          src: url('https://cdn.bootcdn.net/ajax/libs/font-awesome/6.5.1/webfonts/fa-regular-400.woff2') format('woff2');
        }
      `;
      document.head.appendChild(fontDisplayStyle);
      console.log('[Performance] 字体预加载已配置');
    }

    setupRequestIdleCallbackPolyfill() {
      if (!window.requestIdleCallback) {
        window.requestIdleCallback = function(cb) {
          return setTimeout(() => cb({ timeRemaining: () => Infinity }), 1);
        };
        window.cancelIdleCallback = function(id) {
          clearTimeout(id);
        };
      }
    }

    setupPerformanceMarks() {
      if ('performance' in window) {
        performance.mark('performance-optimizer-start');
      }
    }

    setupCriticalResourcePreloading() {
      const criticalResources = [
        { type: 'style', href: 'https://cdn.bootcdn.net/ajax/libs/twitter-bootstrap/5.3.8/css/bootstrap.min.css' },
        { type: 'style', href: 'https://cdn.bootcdn.net/ajax/libs/font-awesome/6.5.1/css/all.min.css' }
      ];

      criticalResources.forEach((resource, index) => {
        requestIdleCallback(() => {
          const link = document.createElement('link');
          link.rel = 'preload';
          link.href = resource.href;
          link.as = resource.type;
          link.crossOrigin = 'anonymous';
          link.onload = () => {
            link.rel = resource.type === 'style' ? 'stylesheet' : 'preload';
          };
          document.head.appendChild(link);
        }, { timeout: 50 + index * 20 });
      });
    }

    setupResourceHints() {
      const preconnectLinks = [
        { href: 'https://cdn.bootcdn.net', crossorigin: 'anonymous' }
      ];

      preconnectLinks.forEach(config => {
        const existing = document.querySelector(`link[rel="preconnect"][href="${config.href}"]`);
        if (!existing) {
          const link = document.createElement('link');
          link.rel = 'preconnect';
          link.href = config.href;
          if (config.crossorigin) {
            link.crossOrigin = config.crossorigin;
          }
          document.head.insertBefore(link, document.head.firstChild);
        }
      });

      const dnsPrefetchLinks = [
        'https://cdn.bootcdn.net'
      ];

      dnsPrefetchLinks.forEach(href => {
        const existing = document.querySelector(`link[rel="dns-prefetch"][href="${href}"]`);
        if (!existing) {
          const link = document.createElement('link');
          link.rel = 'dns-prefetch';
          link.href = href;
          document.head.appendChild(link);
        }
      });
    }

    setupPreloadCriticalAssets() {
      const preloadConfig = [
        { rel: 'preload', href: '/static/js/main.js', as: 'script', priority: 'high' },
        { rel: 'preload', href: '/static/js/captcha.js', as: 'script', priority: 'high' },
        { rel: 'preload', href: '/static/js/mobile-optimization.js', as: 'script', priority: 'medium' },
        { rel: 'preload', href: '/static/js/performance-optimization.js', as: 'script', priority: 'medium' }
      ];

      preloadConfig.forEach((config, index) => {
        const existing = document.querySelector(`link[rel="${config.rel}"][href="${config.href}"]`);
        if (!existing) {
          requestIdleCallback(() => {
            const link = document.createElement('link');
            link.rel = config.rel;
            link.href = config.href;
            link.as = config.as;
            if (config.priority === 'high') {
              link.setAttribute('fetchpriority', 'high');
            }
            document.head.appendChild(link);
          }, { timeout: 50 + index * 15 });
        }
      });

      this.setupResourcePrioritization();
    }

    setupResourcePrioritization() {
      const nonBlockingStylesheets = document.querySelectorAll('link[rel="stylesheet"]:not([href*="bootstrap"]):not([href*="font-awesome"])');
      nonBlockingStylesheets.forEach(link => {
        const href = link.getAttribute('href');
        if (href && !href.startsWith('data:')) {
          link.setAttribute('media', 'print');
          link.setAttribute('onload', "this.media='all'");
        }
      });

      const asyncScripts = document.querySelectorAll('script:not([async]):not([defer]):not([src*="main.js"]):not([src*="captcha.js"])');
      asyncScripts.forEach(script => {
        if (!script.hasAttribute('data-critical')) {
          script.setAttribute('async', '');
        }
      });
      console.log('[Performance] 资源优先级已优化');
    }

    setupLazyLoading() {
      if ('IntersectionObserver' in window) {
        this.imageObserver = new IntersectionObserver((entries) => {
          entries.forEach(entry => {
            if (entry.isIntersecting) {
              this.loadLazyElement(entry.target);
              this.imageObserver.unobserve(entry.target);
            }
          });
        }, {
          rootMargin: `${PERFORMANCE_CONFIG.lazyLoadThreshold}px`,
          threshold: 0.01,
          trackVisibility: true,
          delay: 100
        });

        const lazyElements = document.querySelectorAll('[data-lazy], [data-src], [data-bg], img[data-srcset]');
        lazyElements.forEach(el => {
          if (!this.processedElements.has(el)) {
            this.processedElements.add(el);
            this.imageObserver.observe(el);
          }
        });
      }
    }

    loadLazyElement(element) {
      if (element.hasAttribute('data-src')) {
        const src = element.getAttribute('data-src');
        if (!this.loadedImages.has(src)) {
          element.src = src;
          element.removeAttribute('data-src');
          this.loadedImages.add(src);
          element.classList.add('lazy-loaded');
        }
      }

      if (element.hasAttribute('data-bg')) {
        const bg = element.getAttribute('data-bg');
        element.style.backgroundImage = `url(${bg})`;
        element.removeAttribute('data-bg');
        element.classList.add('lazy-loaded');
      }

      if (element.hasAttribute('data-srcset')) {
        const srcset = element.getAttribute('data-srcset');
        if (srcset) {
          element.srcset = srcset;
          element.removeAttribute('data-srcset');
          element.classList.add('lazy-loaded');
        }
      }

      if (element.hasAttribute('data-lazy')) {
        const lazyContent = element.getAttribute('data-lazy');
        try {
          const content = JSON.parse(lazyContent);
          if (content.innerHTML) {
            element.innerHTML = content.innerHTML;
          }
          if (content.src) {
            element.src = content.src;
          }
        } catch (e) {
          console.warn('[Performance] 懒加载内容解析失败:', e);
        }
        element.removeAttribute('data-lazy');
        element.classList.add('lazy-loaded');
      }
    }

    setupImageOptimization() {
      const images = document.querySelectorAll('img[data-optimize], img[loading="lazy"]');

      images.forEach(img => {
        if (img.complete) {
          this.optimizeImage(img);
        } else {
          img.addEventListener('load', () => this.optimizeImage(img), { once: true });
        }
      });

      this.setupResponsiveImages();
    }

    optimizeImage(img) {
      const src = img.src;

      if (src.startsWith('data:') || src.startsWith('blob:')) {
        return;
      }

      if (img.hasAttribute('data-optimized')) {
        return;
      }

      img.setAttribute('data-optimized', 'true');

      const canvas = document.createElement('canvas');
      const ctx = canvas.getContext('2d');

      const width = img.naturalWidth || img.width;
      const height = img.naturalHeight || img.height;

      let newWidth = width;
      let newHeight = height;

      if (width > PERFORMANCE_CONFIG.imageMaxWidth) {
        newHeight = (height / width) * PERFORMANCE_CONFIG.imageMaxWidth;
        newWidth = PERFORMANCE_CONFIG.imageMaxWidth;
      }

      if (newHeight > PERFORMANCE_CONFIG.imageMaxHeight) {
        newWidth = (newWidth / newHeight) * PERFORMANCE_CONFIG.imageMaxHeight;
        newHeight = PERFORMANCE_CONFIG.imageMaxHeight;
      }

      if (newWidth === width && newHeight === height) {
        return;
      }

      canvas.width = newWidth;
      canvas.height = newHeight;
      ctx.drawImage(img, 0, 0, newWidth, newHeight);

      const format = img.dataset.format || 'jpeg';
      const optimizedSrc = canvas.toDataURL(`image/${format}`, PERFORMANCE_CONFIG.compressionQuality);

      img.src = optimizedSrc;
      img.width = newWidth;
      img.height = newHeight;
      
      console.log(`[Performance] 图片压缩: ${width}x${height} -> ${newWidth}x${newHeight}`);
    }

    setupResponsiveImages() {
      const images = document.querySelectorAll('img[data-src]');
      images.forEach(img => {
        const src = img.getAttribute('data-src');
        if (src) {
          const srcset = this.generateSrcset(src);
          if (srcset) {
            img.setAttribute('data-srcset', srcset);
          }
        }
      });
    }

    generateSrcset(src) {
      if (!src.includes('.')) return null;
      
      const extIndex = src.lastIndexOf('.');
      const base = src.substring(0, extIndex);
      const ext = src.substring(extIndex);
      
      const sizes = [1, 1.5, 2, 3];
      return sizes.map(size => {
        const suffix = size === 1 ? '' : `@${size}x`;
        return `${base}${suffix}${ext} ${size}x`;
      }).join(', ');
    }

    setupScriptDeferring() {
      const nonCriticalScripts = document.querySelectorAll('script[data-defer]');
      nonCriticalScripts.forEach(script => {
        script.defer = true;
        script.async = false;
      });

      this.deferNonCriticalScripts();
    }

    deferNonCriticalScripts() {
      const scriptsToDefer = [
        '/static/js/captcha.js',
        '/static/js/pwa.js',
        '/static/js/trace.js',
        '/static/js/detector.js'
      ];

      scriptsToDefer.forEach(src => {
        const script = document.querySelector(`script[src="${src}"]`);
        if (script && !script.hasAttribute('data-deferred')) {
          script.setAttribute('data-deferred', 'true');
          script.setAttribute('defer', '');
        }
      });
    }

    setupMemoryManagement() {
      const cleanup = this.debounce(() => {
        this.cleanupEventListeners();
        this.cleanupUnusedImages();
        this.cleanupStyles();
      }, 5000);

      if (document.readyState === 'complete') {
        cleanup();
      } else {
        window.addEventListener('load', cleanup);
      }

      window.addEventListener('beforeunload', () => {
        this.cleanupEventListeners();
        this.cleanupObserver();
        this.clearMemoryCache();
      });

      this.setupMemoryWarningHandler();
    }

    setupMemoryWarningHandler() {
      if ('MemoryManager' in window && 'addEventListener' in window.MemoryManager) {
        window.MemoryManager.addEventListener('memorypressure', (e) => {
          if (e.detail.level === 'critical') {
            this.clearMemoryCache();
            this.cleanupUnusedImages();
          }
        });
      }
    }

    clearMemoryCache() {
      this.loadedImages.clear();
      this.processedElements.clear();
      console.log('[Performance] 内存缓存已清理');
    }

    cleanupEventListeners() {
      const deadElements = document.querySelectorAll('.dead, [data-dead]');
      deadElements.forEach(el => {
        const clone = el.cloneNode(true);
        el.parentNode.replaceChild(clone, el);
      });
    }

    cleanupUnusedImages() {
      const images = document.querySelectorAll('img');
      images.forEach(img => {
        const rect = img.getBoundingClientRect();
        const isOffscreen = rect.top > window.innerHeight * 2 || rect.bottom < -window.innerHeight;
        
        if (isOffscreen && !img.hasAttribute('data-keep') && !img.hasAttribute('data-src')) {
          img.setAttribute('data-src', img.src);
          img.src = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg"%3E%3C/svg%3E';
          img.setAttribute('data-offloaded', 'true');
        }
      });
    }

    cleanupStyles() {
      const styles = document.querySelectorAll('style[data-dynamic]');
      styles.forEach(style => {
        if (!document.body.contains(style)) {
          style.remove();
        }
      });
    }

    cleanupObserver() {
      if (this.imageObserver) {
        this.imageObserver.disconnect();
      }
      if (this.observer) {
        this.observer.disconnect();
      }
    }

    setupPerformanceMonitoring() {
      if (!window.PerformanceObserver) {
        return;
      }

      try {
        this.observer = new PerformanceObserver((list) => {
          list.getEntries().forEach(entry => {
            this.handlePerformanceEntry(entry);
          });
        });

        this.observer.observe({ entryTypes: ['navigation', 'resource', 'paint', 'measure', 'largest-contentful-paint'] });

        this.setupCoreWebVitalsMonitoring();
        
      } catch (e) {
        console.warn('[Performance] 性能监控初始化失败:', e);
      }
    }

    setupCoreWebVitalsMonitoring() {
      if (!('PerformanceObserver' in window)) return;

      const lcpObserver = new PerformanceObserver((entryList) => {
        const entries = entryList.getEntries();
        const lcpEntry = entries[entries.length - 1];
        if (lcpEntry) {
          this.performanceMetrics.lcp = lcpEntry.startTime;
          console.log('[Performance] LCP:', lcpEntry.startTime.toFixed(2) + 'ms');
          
          if (lcpEntry.startTime > 2500) {
            console.warn('[Performance] LCP 较慢:', lcpEntry.startTime.toFixed(2) + 'ms');
          }
        }
      });

      lcpObserver.observe({ type: 'largest-contentful-paint', buffered: true });

      const fidObserver = new PerformanceObserver((entryList) => {
        const entries = entryList.getEntries();
        const fidEntry = entries[0];
        if (fidEntry) {
          this.performanceMetrics.fid = fidEntry.processingStart - fidEntry.startTime;
          console.log('[Performance] FID:', this.performanceMetrics.fid.toFixed(2) + 'ms');
        }
      });

      fidObserver.observe({ type: 'first-input', buffered: true });

      const clsObserver = new PerformanceObserver((entryList) => {
        entryList.getEntries().forEach(entry => {
          if (!entry.hadRecentInput) {
            this.performanceMetrics.cls += entry.value;
            console.log('[Performance] CLS:', this.performanceMetrics.cls.toFixed(2));
            
            if (this.performanceMetrics.cls > 0.25) {
              console.warn('[Performance] CLS 较大:', this.performanceMetrics.cls.toFixed(2));
            }
          }
        });
      });

      clsObserver.observe({ type: 'layout-shift', buffered: true });

      const inpObserver = new PerformanceObserver((entryList) => {
        entryList.getEntries().forEach(entry => {
          const duration = entry.processingEnd - entry.startTime;
          if (!this.performanceMetrics.inp || duration > this.performanceMetrics.inp) {
            this.performanceMetrics.inp = duration;
            console.log('[Performance] INP:', this.performanceMetrics.inp.toFixed(2) + 'ms');
            
            if (duration > 200) {
              console.warn('[Performance] INP 较慢:', duration.toFixed(2) + 'ms');
            }
          }
        });
      });

      inpObserver.observe({ type: 'interaction', buffered: true });

      this.setupCodeSplitting();
    }

    setupCodeSplitting() {
      const modules = {
        trace: '/static/js/trace.js',
        pwa: '/static/js/pwa.js',
        detector: '/static/js/detector.js',
        '3dcaptcha': '/static/js/3dcaptcha.js',
        seamless: '/static/js/seamless.js',
        gesture: '/static/js/gesture-handler.js',
        theme: '/static/js/theme.js',
        i18n: '/static/js/i18n.js',
        mfa: '/static/js/mfa.js',
        crypto: '/static/js/crypto-utils.js'
      };

      const loadModule = (name) => {
        return new Promise((resolve, reject) => {
          const script = document.createElement('script');
          script.src = modules[name];
          script.onload = () => {
            console.log(`[Performance] 模块 ${name} 已加载`);
            resolve();
          };
          script.onerror = () => {
            console.error(`[Performance] 模块 ${name} 加载失败`);
            reject(new Error(`Failed to load ${name}`));
          };
          document.body.appendChild(script);
        });
      };

      window.loadCaptchaModule = loadModule;
      window.captchaModules = modules;

      requestIdleCallback(() => {
        const deferredModules = ['i18n', 'theme'];
        deferredModules.forEach(module => {
          setTimeout(() => loadModule(module), 1000);
        });
      });

      console.log('[Performance] 代码分割已配置');
    }

    handlePerformanceEntry(entry) {
      switch (entry.entryType) {
        case 'navigation':
          this.handleNavigationMetrics(entry);
          break;
        case 'resource':
          this.handleResourceMetrics(entry);
          break;
        case 'paint':
          this.handlePaintMetrics(entry);
          break;
        case 'largest-contentful-paint':
          this.handleLCPMetrics(entry);
          break;
      }
    }

    handleNavigationMetrics(entry) {
      const metrics = {
        domContentLoaded: entry.domContentLoadedEventEnd - entry.startTime,
        loadComplete: entry.loadEventEnd - entry.startTime,
        firstByte: entry.responseStart - entry.requestStart,
        dnsLookup: entry.domainLookupEnd - entry.domainLookupStart,
        tcpConnection: entry.connectEnd - entry.connectStart,
        ttfB: entry.responseStart - entry.navigationStart,
        tti: entry.domContentLoadedEventEnd - entry.navigationStart
      };

      this.performanceMetrics.domContentLoaded = metrics.domContentLoaded;
      this.performanceMetrics.loadComplete = metrics.loadComplete;

      console.log('[Performance] 导航性能指标:', metrics);

      if (metrics.loadComplete > PERFORMANCE_CONFIG.criticalRenderingBudget) {
        console.warn('[Performance] 页面加载时间超过预算:', metrics.loadComplete.toFixed(2) + 'ms');
        this.showPerformanceWarning(metrics);
      }

      if (metrics.ttfB > 300) {
        console.warn('[Performance] TTFB 较慢:', metrics.ttfB.toFixed(2) + 'ms');
      }
    }

    handleResourceMetrics(entry) {
      this.performanceMetrics.resourceCount++;

      if (entry.duration > PERFORMANCE_CONFIG.maxResourceDuration) {
        this.performanceMetrics.slowResources.push({
          name: entry.name,
          duration: entry.duration,
          type: entry.initiatorType
        });
        console.warn('[Performance] 资源加载缓慢:', entry.name, entry.duration.toFixed(2) + 'ms');
      }
    }

    handlePaintMetrics(entry) {
      if (entry.name === 'first-paint') {
        this.performanceMetrics.firstPaint = entry.startTime;
        console.log('[Performance] 首次绘制:', entry.startTime.toFixed(2) + 'ms');
      }
      if (entry.name === 'first-contentful-paint') {
        this.performanceMetrics.firstContentfulPaint = entry.startTime;
        console.log('[Performance] 首次内容绘制:', entry.startTime.toFixed(2) + 'ms');
        
        if (entry.startTime > 1800) {
          console.warn('[Performance] FCP 较慢:', entry.startTime.toFixed(2) + 'ms');
        }
      }
    }

    handleLCPMetrics(entry) {
      this.performanceMetrics.lcp = entry.startTime;
      console.log('[Performance] LCP:', entry.startTime.toFixed(2) + 'ms');
    }

    showPerformanceWarning(metrics) {
      if (typeof showEnhancedToast === 'function') {
        const loadTime = Math.round(metrics.loadComplete);
        if (loadTime > PERFORMANCE_CONFIG.criticalRenderingBudget) {
          showEnhancedToast(
            `页面加载时间 ${loadTime}ms，建议优化`,
            'warning',
            `TTFB: ${Math.round(metrics.ttfB)}ms | FCP: ${Math.round(metrics.firstContentfulPaint || 0)}ms`
          );
        }
      }
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
      return function(...args) {
        if (!inThrottle) {
          func.apply(this, args);
          inThrottle = true;
          setTimeout(() => inThrottle = false, limit);
        }
      };
    }

    injectPerformanceStyles() {
      if (document.getElementById('performance-optimization-styles')) {
        return;
      }

      const style = document.createElement('style');
      style.id = 'performance-optimization-styles';
      style.textContent = `
        [data-lazy],
        [data-src],
        [data-bg],
        img[data-srcset] {
          opacity: 0;
          transition: opacity 0.3s ease;
        }

        [data-lazy].lazy-loaded,
        [data-src].lazy-loaded,
        [data-bg].lazy-loaded,
        img[data-srcset].lazy-loaded {
          opacity: 1;
        }

        .perf-placeholder {
          background: linear-gradient(
            90deg,
            #f5f5f5 0%,
            #e8e8e8 50%,
            #f5f5f5 100%
          );
          background-size: 200% 100%;
          animation: perf-shimmer 1.5s ease-in-out infinite;
        }

        @keyframes perf-shimmer {
          0% { background-position: 200% 0; }
          100% { background-position: -200% 0; }
        }

        .perf-critical {
          content-visibility: auto;
          contain-intrinsic-size: 1px 1000px;
        }

        .perf-optimized-image {
          image-rendering: -webkit-optimize-contrast;
          image-rendering: crisp-edges;
        }

        @media (prefers-reduced-motion: reduce) {
          [data-lazy],
          [data-src],
          [data-bg],
          img[data-srcset],
          .perf-placeholder {
            transition: none;
            animation: none;
          }
        }

        @media (prefers-color-scheme: dark) {
          .perf-placeholder {
            background: linear-gradient(
              90deg,
              #2a2a2a 0%,
              #333 50%,
              #2a2a2a 100%
            );
          }
        }

        img {
          max-width: 100%;
          height: auto;
        }

        .perf-blur-up {
          filter: blur(5px);
          transition: filter 0.3s ease;
        }

        .perf-blur-up.loaded {
          filter: blur(0);
        }
      `;

      document.head.appendChild(style);
    }

    getPerformanceReport() {
      return {
        ...this.performanceMetrics,
        timestamp: Date.now(),
        userAgent: navigator.userAgent
      };
    }
  }

  const performanceOptimizer = new PerformanceOptimizer();

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => performanceOptimizer.init());
  } else {
    performanceOptimizer.init();
  }

  window.PerformanceOptimizer = performanceOptimizer;
})();