import { useEffect, useCallback } from 'react';
import { recordPageLoad, recordApiCall, recordUserInteraction, recordMetric } from '../utils/sentry';

export function usePerformanceMonitoring() {
  const measurePageLoad = useCallback(() => {
    if (typeof window === 'undefined' || !window.performance) return;

    const timing = window.performance.timing;
    if (!timing) return;

    const pageLoadMetrics = {
      ttfb: timing.responseStart - timing.requestStart,
      domContentLoaded: timing.domContentLoadedEventEnd - timing.navigationStart,
      domComplete: timing.domComplete - timing.navigationStart,
      loadComplete: timing.loadEventEnd - timing.navigationStart,
      firstPaint: null,
      firstContentfulPaint: null,
      largestContentfulPaint: null,
    };

    const paintEntries = window.performance.getEntriesByType('paint');
    paintEntries.forEach(entry => {
      if (entry.name === 'first-paint') {
        pageLoadMetrics.firstPaint = entry.startTime;
      }
      if (entry.name === 'first-contentful-paint') {
        pageLoadMetrics.firstContentfulPaint = entry.startTime;
      }
    });

    const lcpEntries = window.performance.getEntriesByType('largest-contentful-paint');
    if (lcpEntries.length > 0) {
      const lcp = lcpEntries[lcpEntries.length - 1];
      pageLoadMetrics.largestContentfulPaint = lcp.startTime;
    }

    const totalLoadTime = pageLoadMetrics.loadComplete;
    recordPageLoad(window.location.pathname, totalLoadTime);
    
    Object.entries(pageLoadMetrics).forEach(([key, value]) => {
      if (value !== null) {
        recordMetric(`page.${key}`, value, 'ms');
      }
    });

    return pageLoadMetrics;
  }, []);

  const trackApiCall = useCallback((endpoint, method, status, duration) => {
    recordApiCall(endpoint, method, status, duration);
    recordMetric(`api.${endpoint}`, duration, 'ms');
    if (status >= 400) {
      recordMetric('api.errors', 1, 'count');
    }
  }, []);

  const trackInteraction = useCallback((element, action) => {
    if (!element) return;
    recordUserInteraction(element, action);
    recordMetric('user.interactions', 1, 'count');
  }, []);

  const getMemoryUsage = useCallback(() => {
    if (typeof window === 'undefined' || !window.performance?.memory) return null;
    const memory = window.performance.memory;
    return {
      usedJSHeapSize: memory.usedJSHeapSize,
      totalJSHeapSize: memory.totalJSHeapSize,
      jsHeapSizeLimit: memory.jsHeapSizeLimit,
      usagePercent: ((memory.usedJSHeapSize / memory.jsHeapSizeLimit) * 100).toFixed(2),
    };
  }, []);

  useEffect(() => {
    if (typeof window === 'undefined') return;
    measurePageLoad();
    const handleLoad = () => setTimeout(measurePageLoad, 0);
    if (document.readyState === 'complete') {
      handleLoad();
    } else {
      window.addEventListener('load', handleLoad);
      return () => window.removeEventListener('load', handleLoad);
    }
  }, [measurePageLoad]);

  useEffect(() => {
    if (typeof window === 'undefined') return;
    const observer = new PerformanceObserver((list) => {
      for (const entry of list.getEntries()) {
        if (entry.entryType === 'largest-contentful-paint') {
          recordMetric('page.lcp', entry.startTime, 'ms');
        }
      }
    });
    observer.observe({ entryTypes: ['largest-contentful-paint'] });
    return () => observer.disconnect();
  }, []);

  return {
    measurePageLoad,
    trackApiCall,
    trackInteraction,
    getMemoryUsage,
  };
}
