import { useState, useEffect, useRef, useCallback } from 'react';

export function usePerformanceMetrics() {
  const [metrics, setMetrics] = useState({
    fcp: null,
    lcp: null,
    fid: null,
    cls: null,
    ttfb: null
  });
  const [webVitalsSupported, setWebVitalsSupported] = useState(false);

  useEffect(() => {
    if (typeof window === 'undefined') return;

    const observer = new PerformanceObserver((list) => {
      for (const entry of list.getEntries()) {
        setMetrics(prev => ({
          ...prev,
          [entry.name]: entry.value
        }));
      }
    });

    try {
      observer.observe({ entryTypes: ['paint', 'largest-contentful-paint', 'first-input', 'layout-shift', 'navigation'] });
      setWebVitalsSupported(true);

      const navEntry = performance.getEntriesByType('navigation')[0];
      if (navEntry) {
        setMetrics(prev => ({
          ...prev,
          ttfb: navEntry.responseStart - navEntry.requestStart,
          fcp: navEntry.domContentLoadedEventEnd - navEntry.requestStart
        }));
      }
    } catch (e) {
      console.warn('Performance metrics not fully supported:', e);
    }

    return () => observer.disconnect();
  }, []);

  return { metrics, webVitalsSupported };
}

export function useNetworkStatus() {
  const [isOnline, setIsOnline] = useState(
    typeof navigator !== 'undefined' ? navigator.onLine : true
  );
  const [effectiveType, setEffectiveType] = useState(
    typeof navigator !== 'undefined' && navigator.connection
      ? navigator.connection.effectiveType
      : '4g'
  );

  useEffect(() => {
    if (typeof window === 'undefined') return;

    const handleOnline = () => setIsOnline(true);
    const handleOffline = () => setIsOnline(false);

    const connection = navigator.connection;
    const handleConnectionChange = () => {
      if (connection) {
        setEffectiveType(connection.effectiveType);
      }
    };

    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);

    if (connection) {
      connection.addEventListener('change', handleConnectionChange);
    }

    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
      if (connection) {
        connection.removeEventListener('change', handleConnectionChange);
      }
    };
  }, []);

  return { isOnline, effectiveType };
}

export function useLazyLoad(options = {}) {
  const {
    threshold = 0.1,
    rootMargin = '50px',
    triggerOnce = true
  } = options;

  const [isVisible, setIsVisible] = useState(false);
  const [isLoaded, setIsLoaded] = useState(false);
  const elementRef = useRef(null);
  const observerRef = useRef(null);

  useEffect(() => {
    if (!elementRef.current) return;

    observerRef.current = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setIsVisible(true);

          if (triggerOnce) {
            observerRef.current?.disconnect();
          }
        }
      },
      { threshold, rootMargin }
    );

    observerRef.current.observe(elementRef.current);

    return () => {
      observerRef.current?.disconnect();
    };
  }, [threshold, rootMargin, triggerOnce]);

  const handleLoad = useCallback(() => {
    setIsLoaded(true);
  }, []);

  return {
    ref: elementRef,
    isVisible,
    isLoaded,
    handleLoad
  };
}

export function useDebounce(value, delay = 300) {
  const [debouncedValue, setDebouncedValue] = useState(value);

  useEffect(() => {
    const handler = setTimeout(() => {
      setDebouncedValue(value);
    }, delay);

    return () => {
      clearTimeout(handler);
    };
  }, [value, delay]);

  return debouncedValue;
}

export function useThrottle(value, limit = 300) {
  const [throttledValue, setThrottledValue] = useState(value);
  const lastRan = useRef(Date.now());

  useEffect(() => {
    const handler = setTimeout(() => {
      if (Date.now() - lastRan.current >= limit) {
        setThrottledValue(value);
        lastRan.current = Date.now();
      }
    }, limit - (Date.now() - lastRan.current));

    return () => clearTimeout(handler);
  }, [value, limit]);

  return throttledValue;
}

export function useRAF(callback) {
  const requestRef = useRef();
  const previousTimeRef = useRef();

  const animate = useCallback((time) => {
    if (previousTimeRef.current !== undefined) {
      const deltaTime = time - previousTimeRef.current;
      callback(deltaTime);
    }
    previousTimeRef.current = time;
    requestRef.current = requestAnimationFrame(animate);
  }, [callback]);

  useEffect(() => {
    requestRef.current = requestAnimationFrame(animate);
    return () => cancelAnimationFrame(requestRef.current);
  }, [animate]);
}

export default {
  usePerformanceMetrics,
  useNetworkStatus,
  useLazyLoad,
  useDebounce,
  useThrottle,
  useRAF
};
