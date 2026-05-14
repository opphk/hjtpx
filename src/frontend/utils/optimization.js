import { lazy, Suspense } from 'react';

export function lazyImport(lazyFn, fallback = null) {
  const LazyComponent = lazy(lazyFn);
  return function LazyWrapper(props) {
    return (
      <Suspense fallback={fallback}>
        <LazyComponent {...props} />
      </Suspense>
    );
  };
}

export function createLazyComponent(importFn, LoadingComponent = null) {
  const LazyComponent = lazy(importFn);
  return function WrappedComponent(props) {
    return (
      <Suspense fallback={LoadingComponent}>
        <LazyComponent {...props} />
      </Suspense>
    );
  };
}

export function memoize(fn, maxSize = 100) {
  const cache = new Map();
  return function memoized(...args) {
    const key = JSON.stringify(args);
    if (cache.has(key)) {
      return cache.get(key);
    }
    const result = fn.apply(this, args);
    if (cache.size >= maxSize) {
      const firstKey = cache.keys().next().value;
      cache.delete(firstKey);
    }
    cache.set(key, result);
    return result;
  };
}

export function debounce(fn, delay = 300) {
  let timeoutId;
  return function debounced(...args) {
    clearTimeout(timeoutId);
    timeoutId = setTimeout(() => fn.apply(this, args), delay);
  };
}

export function throttle(fn, limit = 300) {
  let inThrottle;
  return function throttled(...args) {
    if (!inThrottle) {
      fn.apply(this, args);
      inThrottle = true;
      setTimeout(() => (inThrottle = false), limit);
    }
  };
}

export function createImageLoader() {
  const loadedImages = new Set();
  const loadingImages = new Map();

  return {
    async loadImage(src) {
      if (loadedImages.has(src)) {
        return true;
      }

      if (loadingImages.has(src)) {
        return loadingImages.get(src);
      }

      const promise = new Promise((resolve, reject) => {
        const img = new Image();
        img.onload = () => {
          loadedImages.add(src);
          loadingImages.delete(src);
          resolve(true);
        };
        img.onerror = () => {
          loadingImages.delete(src);
          reject(new Error(`Failed to load image: ${src}`));
        };
        img.src = src;
      });

      loadingImages.set(src, promise);
      return promise;
    },

    preloadImages(sources) {
      return Promise.allSettled(sources.map(src => this.loadImage(src)));
    },

    isLoaded(src) {
      return loadedImages.has(src);
    },

    clearCache() {
      loadedImages.clear();
      loadingImages.clear();
    }
  };
}

export function optimizeCallback(fn, deps = []) {
  let lastCall = 0;
  let lastResult;
  let lastDeps = [];

  return function optimizedCallback(...args) {
    const now = Date.now();
    const depsChanged = deps.some((dep, i) => dep !== lastDeps[i]);

    if (now - lastCall < 16 || depsChanged) {
      lastCall = now;
      lastResult = fn.apply(this, args);
      lastDeps = deps;
    }

    return lastResult;
  };
}

export function createIntersectionObserver(callback, options = {}) {
  const defaultOptions = {
    root: null,
    rootMargin: '50px',
    threshold: 0.1,
    ...options
  };

  let observer = null;

  return {
    observe(element) {
      if (!observer) {
        observer = new IntersectionObserver((entries) => {
          entries.forEach(entry => callback(entry));
        }, defaultOptions);
      }
      observer.observe(element);
    },

    unobserve(element) {
      if (observer) {
        observer.unobserve(element);
      }
    },

    disconnect() {
      if (observer) {
        observer.disconnect();
        observer = null;
      }
    }
  };
}

export function batchUpdates(callback) {
  let updates = [];
  let scheduled = false;

  return function batched(...args) {
    updates.push(() => callback.apply(this, args));

    if (!scheduled) {
      scheduled = true;
      Promise.resolve().then(() => {
        const currentUpdates = updates;
        updates = [];
        scheduled = false;
        currentUpdates.forEach(update => update());
      });
    }
  };
}

export function generatePreloadHints(assets) {
  return assets.map(asset => {
    if (asset.type === 'script') {
      return {
        rel: 'preload',
        as: 'script',
        href: asset.src,
        crossOrigin: asset.crossOrigin
      };
    }
    if (asset.type === 'style') {
      return {
        rel: 'preload',
        as: 'style',
        href: asset.href
      };
    }
    if (asset.type === 'image') {
      return {
        rel: 'preload',
        as: 'image',
        href: asset.src,
        ...(asset.imagesrcset && { imagesrcset: asset.imagesrcset })
      };
    }
    return null;
  }).filter(Boolean);
}

export function createResourceCache(maxSize = 50) {
  const cache = new Map();

  return {
    get(key) {
      if (cache.has(key)) {
        const entry = cache.get(key);
        entry.lastAccessed = Date.now();
        return entry.value;
      }
      return null;
    },

    set(key, value) {
      if (cache.size >= maxSize) {
        let oldestKey = null;
        let oldestTime = Infinity;

        for (const [k, v] of cache.entries()) {
          if (v.lastAccessed < oldestTime) {
            oldestTime = v.lastAccessed;
            oldestKey = k;
          }
        }

        if (oldestKey) {
          cache.delete(oldestKey);
        }
      }

      cache.set(key, {
        value,
        lastAccessed: Date.now()
      });
    },

    has(key) {
      return cache.has(key);
    },

    invalidate(key) {
      cache.delete(key);
    },

    clear() {
      cache.clear();
    },

    size() {
      return cache.size;
    }
  };
}

export default {
  lazyImport,
  createLazyComponent,
  memoize,
  debounce,
  throttle,
  createImageLoader,
  optimizeCallback,
  createIntersectionObserver,
  batchUpdates,
  generatePreloadHints,
  createResourceCache
};
