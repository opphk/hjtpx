(function() {
  'use strict';

  const PERFORMANCE_CONFIG = {
    lazyLoadThreshold: 200,
    preloadBatchSize: 3,
    imageMaxWidth: 800,
    imageMaxHeight: 800,
    compressionQuality: 0.8,
    debounceDelay: 250,
    throttleDelay: 100
  };

  class PerformanceOptimizer {
    constructor() {
      this.observer = null;
      this.imageObserver = null;
      this.loadedImages = new Set();
      this.processedElements = new Set();
    }

    init() {
      console.log('[Performance] 初始化性能优化模块');
      this.setupLazyLoading();
      this.setupImageOptimization();
      this.setupResourceHints();
      this.setupCodeSplitting();
      this.setupMemoryManagement();
      this.setupPerformanceMonitoring();
      this.injectPerformanceStyles();
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
          threshold: 0.01
        });

        const lazyElements = document.querySelectorAll('[data-lazy], [data-src], [data-bg]');
        lazyElements.forEach(el => this.imageObserver.observe(el));
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
      const images = document.querySelectorAll('img[data-optimize]');

      images.forEach(img => {
        if (img.complete) {
          this.optimizeImage(img);
        } else {
          img.addEventListener('load', () => this.optimizeImage(img));
        }
      });
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

      const optimizedSrc = canvas.toDataURL('image/jpeg', PERFORMANCE_CONFIG.compressionQuality);

      img.src = optimizedSrc;
      console.log(`[Performance] 图片压缩: ${width}x${height} -> ${newWidth}x${newHeight}`);
    }

    setupResourceHints() {
      const preconnectLinks = [
        { href: 'https://cdn.bootcdn.net', crossorigin: true },
        { href: 'https://cdn.bootcdn.net', crossorigin: true }
      ];

      preconnectLinks.forEach(config => {
        const link = document.createElement('link');
        link.rel = 'preconnect';
        link.href = config.href;
        if (config.crossorigin) {
          link.crossOrigin = 'anonymous';
        }
        document.head.appendChild(link);
      });

      const dnsPrefetchLinks = [
        'https://cdn.bootcdn.net'
      ];

      dnsPrefetchLinks.forEach(href => {
        const link = document.createElement('link');
        link.rel = 'dns-prefetch';
        link.href = href;
        document.head.appendChild(link);
      });
    }

    setupCodeSplitting() {
      const scriptPaths = [
        '/static/js/captcha.js',
        '/static/js/main.js',
        '/static/js/pwa.js'
      ];

      scriptPaths.forEach(path => {
        const existing = document.querySelector(`script[src="${path}"]`);
        if (existing && !existing.hasAttribute('data-split')) {
          existing.setAttribute('data-split', 'true');
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
      });
    }

    cleanupEventListeners() {
      const deadElements = document.querySelectorAll('.dead');
      deadElements.forEach(el => {
        const clone = el.cloneNode(true);
        el.parentNode.replaceChild(clone, el);
      });
    }

    cleanupUnusedImages() {
      const images = document.querySelectorAll('img');
      images.forEach(img => {
        const rect = img.getBoundingClientRect();
        if (rect.top > window.innerHeight || rect.bottom < 0) {
          if (!img.hasAttribute('data-keep')) {
            img.setAttribute('data-keep', 'true');
          }
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
            if (entry.entryType === 'navigation') {
              this.handleNavigationMetrics(entry);
            }
            if (entry.entryType === 'resource') {
              this.handleResourceMetrics(entry);
            }
            if (entry.entryType === 'paint') {
              this.handlePaintMetrics(entry);
            }
          });
        });

        this.observer.observe({ entryTypes: ['navigation', 'resource', 'paint'] });

      } catch (e) {
        console.warn('[Performance] 性能监控初始化失败:', e);
      }
    }

    handleNavigationMetrics(entry) {
      const metrics = {
        domContentLoaded: entry.domContentLoadedEventEnd - entry.startTime,
        loadComplete: entry.loadEventEnd - entry.startTime,
        firstByte: entry.responseStart - entry.requestStart,
        dnsLookup: entry.domainLookupEnd - entry.domainLookupStart,
        tcpConnection: entry.connectEnd - entry.connectStart
      };

      console.log('[Performance] 导航性能指标:', metrics);

      if (metrics.loadComplete > 5000) {
        console.warn('[Performance] 页面加载时间过长:', metrics.loadComplete + 'ms');
      }
    }

    handleResourceMetrics(entry) {
      if (entry.duration > 2000) {
        console.warn('[Performance] 资源加载缓慢:', entry.name, entry.duration + 'ms');
      }
    }

    handlePaintMetrics(entry) {
      if (entry.name === 'first-contentful-paint') {
        console.log('[Performance] 首次内容绘制:', entry.startTime + 'ms');
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
        [data-bg] {
          opacity: 0;
          transition: opacity 0.3s ease;
        }

        [data-lazy].lazy-loaded,
        [data-src].lazy-loaded,
        [data-bg].lazy-loaded {
          opacity: 1;
        }

        .perf-placeholder {
          background: linear-gradient(
            90deg,
            #f0f0f0 0%,
            #e0e0e0 50%,
            #f0f0f0 100%
          );
          background-size: 200% 100%;
          animation: perf-shimmer 1.5s ease-in-out infinite;
        }

        @keyframes perf-shimmer {
          0% { background-position: 200% 0; }
          100% { background-position: -200% 0; }
        }

        @media (prefers-reduced-motion: reduce) {
          [data-lazy],
          [data-src],
          [data-bg],
          .perf-placeholder {
            transition: none;
            animation: none;
          }
        }

        .perf-critical {
          content-visibility: auto;
          contain-intrinsic-size: 1px 1000px;
        }
      `;

      document.head.appendChild(style);
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
