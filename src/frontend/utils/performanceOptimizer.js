export class PerformanceOptimizer {
  constructor(config = {}) {
    this.config = {
      enablePrefetch: config.enablePrefetch !== false,
      enablePreload: config.enablePreload !== false,
      enableBundleAnalysis: config.enableBundleAnalysis || false,
      ...config
    };

    this.preloadedResources = new Set();
    this.prefetchedLinks = new Set();
  }

  prefetchRoute(routePath) {
    if (!this.config.enablePrefetch || typeof document === 'undefined') return;

    const link = document.createElement('link');
    link.rel = 'prefetch';
    link.href = routePath;
    link.as = 'fetch';

    if (!this.prefetchedLinks.has(routePath)) {
      document.head.appendChild(link);
      this.prefetchedLinks.add(routePath);
    }
  }

  preloadResource(resourcePath, options = {}) {
    if (!this.config.enablePreload || typeof document === 'undefined') return;

    const { as = 'script', type, crossOrigin } = options;

    if (this.preloadedResources.has(resourcePath)) {
      return;
    }

    const link = document.createElement('link');
    link.rel = 'preload';
    link.href = resourcePath;
    link.as = as;

    if (type) link.type = type;
    if (crossOrigin) link.crossOrigin = crossOrigin;

    document.head.appendChild(link);
    this.preloadedResources.add(resourcePath);
  }

  optimizeImages(images) {
    if (typeof window === 'undefined') return images;

    const connection = navigator.connection || navigator.mozConnection || navigator.webkitConnection;

    if (connection) {
      const effectiveType = connection.effectiveType;
      const saveData = connection.saveData;

      if (saveData || effectiveType === '2g' || effectiveType === 'slow-2g') {
        return images.map(img => ({
          ...img,
          src: img.lowQualitySrc || img.src,
          srcSet: undefined
        }));
      }

      if (effectiveType === '3g') {
        return images.map(img => ({
          ...img,
          srcSet: img.mediumQualitySrcSet || img.srcSet
        }));
      }
    }

    return images;
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

  memoize(func) {
    const cache = new Map();
    return function(...args) {
      const key = JSON.stringify(args);
      if (cache.has(key)) {
        return cache.get(key);
      }
      const result = func.apply(this, args);
      cache.set(key, result);
      return result;
    };
  }
}

export class BundleAnalyzer {
  constructor() {
    this.bundleStats = {
      totalSize: 0,
      chunks: [],
      modules: new Map()
    };
  }

  analyzeBundle(buildStats) {
    if (!buildStats || !buildStats.assets) {
      return this.bundleStats;
    }

    let totalSize = 0;
    const chunks = [];

    buildStats.assets.forEach(asset => {
      totalSize += asset.size;
      chunks.push({
        name: asset.name,
        size: asset.size,
        sizeFormatted: this.formatBytes(asset.size)
      });
    });

    if (buildStats.chunks) {
      buildStats.chunks.forEach(chunk => {
        const existingChunk = chunks.find(c => c.name === chunk.file);
        if (existingChunk) {
          existingChunk.modules = chunk.modules?.length || 0;
        }
      });
    }

    this.bundleStats = {
      totalSize,
      totalSizeFormatted: this.formatBytes(totalSize),
      chunks: chunks.sort((a, b) => b.size - a.size),
      analyzedAt: Date.now()
    };

    return this.bundleStats;
  }

  formatBytes(bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  }

  getLargestChunks(limit = 5) {
    return this.bundleStats.chunks.slice(0, limit);
  }

  identifyLargeChunks(thresholdKB = 100) {
    const thresholdBytes = thresholdKB * 1024;
    return this.bundleStats.chunks.filter(chunk => chunk.size > thresholdBytes);
  }

  suggestOptimizations() {
    const suggestions = [];
    const largeChunks = this.identifyLargeChunks();

    if (largeChunks.length > 0) {
      suggestions.push({
        type: 'code_splitting',
        message: `Found ${largeChunks.length} chunks larger than 100KB`,
        chunks: largeChunks.map(c => c.name),
        recommendation: 'Consider splitting these chunks further using dynamic imports'
      });
    }

    if (this.bundleStats.totalSize > 500 * 1024) {
      suggestions.push({
        type: 'bundle_size',
        message: `Total bundle size (${this.bundleStats.totalSizeFormatted}) exceeds recommended 500KB`,
        recommendation: 'Consider implementing tree shaking, removing unused code, or lazy loading'
      });
    }

    return suggestions;
  }

  exportReport() {
    return {
      ...this.bundleStats,
      suggestions: this.suggestOptimizations()
    };
  }
}

export class ResourceOptimizer {
  constructor() {
    this.resourceHints = new Map();
  }

  addDnsPrefetch(domain) {
    if (typeof document === 'undefined') return;

    const existing = document.querySelector(`link[rel="dns-prefetch"][href*="${domain}"]`);
    if (existing) return;

    const link = document.createElement('link');
    link.rel = 'dns-prefetch';
    link.href = `//${domain}`;
    document.head.appendChild(link);
  }

  addPreconnect(domain, options = {}) {
    if (typeof document === 'undefined') return;

    const existing = document.querySelector(`link[rel="preconnect"][href*="${domain}"]`);
    if (existing) return;

    const link = document.createElement('link');
    link.rel = 'preconnect';
    link.href = `//${domain}`;
    link.crossOrigin = options.crossOrigin || 'anonymous';

    document.head.appendChild(link);
  }

  prioritizeCriticalResources(urls) {
    urls.forEach(url => {
      const link = document.createElement('link');
      link.rel = 'preload';
      link.href = url;
      link.as = this.determineResourceType(url);
      document.head.appendChild(link);
    });
  }

  determineResourceType(url) {
    const extension = url.split('.').pop().toLowerCase();

    const typeMap = {
      js: 'script',
      css: 'style',
      png: 'image',
      jpg: 'image',
      jpeg: 'image',
      gif: 'image',
      svg: 'image',
      webp: 'image',
      woff: 'font',
      woff2: 'font',
      ttf: 'font',
      eot: 'font'
    };

    return typeMap[extension] || 'fetch';
  }

  deferNonCriticalScripts() {
    if (typeof document === 'undefined') return;

    const scripts = document.querySelectorAll('script:not([type="module"]):not([async])');

    scripts.forEach(script => {
      if (!script.hasAttribute('data-critical')) {
        script.setAttribute('defer', '');
      }
    });
  }
}

export default {
  PerformanceOptimizer,
  BundleAnalyzer,
  ResourceOptimizer
};
