import { useState, useEffect, useRef, useCallback } from 'react';

export const usePerformanceMonitor = (options = {}) => {
  const {
    enableLCP = true,
    enableFID = true,
    enableCLS = true,
    enableTTFB = true,
    reportInterval = 10000,
    onMetricChange = null,
    onThresholdExceeded = null
  } = options;

  const [metrics, setMetrics] = useState({
    LCP: null,
    FID: null,
    CLS: null,
    TTFB: null,
    FCP: null,
    TT: null
  });

  const [performanceScore, setPerformanceScore] = useState(null);
  const [isSlowConnection, setIsSlowConnection] = useState(false);
  const [loadingState, setLoadingState] = useState('loading');
  const [resourceTimings, setResourceTimings] = useState([]);
  const metricsRef = useRef({});
  const thresholds = {
    LCP: { good: 2500, poor: 4000 },
    FID: { good: 100, poor: 300 },
    CLS: { good: 0.1, poor: 0.25 },
    TTFB: { good: 800, poor: 1800 }
  };

  const calculatePerformanceScore = useCallback((currentMetrics) => {
    let score = 100;

    if (currentMetrics.LCP !== null) {
      if (currentMetrics.LCP > thresholds.LCP.poor) score -= 30;
      else if (currentMetrics.LCP > thresholds.LCP.good) score -= 15;
    }

    if (currentMetrics.FID !== null) {
      if (currentMetrics.FID > thresholds.FID.poor) score -= 20;
      else if (currentMetrics.FID > thresholds.FID.good) score -= 10;
    }

    if (currentMetrics.CLS !== null) {
      if (currentMetrics.CLS > thresholds.CLS.poor) score -= 30;
      else if (currentMetrics.CLS > thresholds.CLS.good) score -= 15;
    }

    return Math.max(0, score);
  }, []);

  const checkThresholds = useCallback((currentMetrics) => {
    const exceededThresholds = [];

    Object.entries(currentMetrics).forEach(([metric, value]) => {
      if (value !== null && thresholds[metric]) {
        if (value > thresholds[metric].poor) {
          exceededThresholds.push({
            metric,
            value,
            threshold: thresholds[metric].poor,
            severity: 'critical'
          });
        } else if (value > thresholds[metric].good) {
          exceededThresholds.push({
            metric,
            value,
            threshold: thresholds[metric].good,
            severity: 'warning'
          });
        }
      }
    });

    if (exceededThresholds.length > 0 && onThresholdExceeded) {
      onThresholdExceeded(exceededThresholds);
    }

    return exceededThresholds;
  }, [onThresholdExceeded]);

  useEffect(() => {
    if (!('PerformanceObserver' in window)) {
      setLoadingState('unsupported');
      return;
    }

    const observers = [];

    if (enableLCP && 'PerformanceObserver' in window) {
      try {
        const lcpObserver = new PerformanceObserver((entryList) => {
          const entries = entryList.getEntries();
          const lastEntry = entries[entries.length - 1];

          if (lastEntry) {
            const lcpTime = lastEntry.startTime;
            setMetrics(prev => {
              const newMetrics = { ...prev, LCP: lcpTime };
              metricsRef.current = newMetrics;
              return newMetrics;
            });

            if (onMetricChange) {
              onMetricChange('LCP', lcpTime);
            }
          }
        });

        lcpObserver.observe({ type: 'largest-contentful-paint', buffered: true });
        observers.push(lcpObserver);
      } catch (e) {
        console.warn('LCP observer not supported');
      }
    }

    if (enableFID && 'PerformanceObserver' in window) {
      try {
        const fidObserver = new PerformanceObserver((entryList) => {
          const entries = entryList.getEntries();
          entries.forEach(entry => {
            if (entry.duration > 0) {
              setMetrics(prev => {
                const newMetrics = { ...prev, FID: entry.processingStart - entry.startTime };
                metricsRef.current = newMetrics;
                return newMetrics;
              });

              if (onMetricChange) {
                onMetricChange('FID', entry.processingStart - entry.startTime);
              }
            }
          });
        });

        fidObserver.observe({ type: 'first-input', buffered: true });
        observers.push(fidObserver);
      } catch (e) {
        console.warn('FID observer not supported');
      }
    }

    if (enableCLS && 'PerformanceObserver' in window) {
      try {
        let clsValue = 0;
        let clsEntries = [];

        const clsObserver = new PerformanceObserver((entryList) => {
          for (const entry of entryList.getEntries()) {
            if (!entry.hadRecentInput) {
              clsEntries.push(entry);
              clsValue += entry.value;
            }
          }
        });

        clsObserver.observe({ type: 'layout-shift', buffered: true });

        setTimeout(() => {
          setMetrics(prev => {
            const newMetrics = { ...prev, CLS: clsValue };
            metricsRef.current = newMetrics;
            return newMetrics;
          });
        }, 1000);

        observers.push(clsObserver);
      } catch (e) {
        console.warn('CLS observer not supported');
      }
    }

    if (enableTTFB && 'PerformanceObserver' in window) {
      try {
        const ttfbObserver = new PerformanceObserver((entryList) => {
          for (const entry of entryList.getEntries()) {
            if (entry.responseStart > 0) {
              const ttfb = entry.responseStart - entry.requestStart;
              setMetrics(prev => {
                const newMetrics = { ...prev, TTFB: ttfb };
                metricsRef.current = newMetrics;
                return newMetrics;
              });

              if (onMetricChange) {
                onMetricChange('TTFB', ttfb);
              }
            }
          }
        });

        ttfbObserver.observe({ type: 'navigation', buffered: true });
        observers.push(ttfbObserver);
      } catch (e) {
        console.warn('TTFB observer not supported');
      }
    }

    const navigationTiming = performance.getEntriesByType('navigation')[0];
    if (navigationTiming) {
      if (navigationTiming.loadEventEnd > 0) {
        setMetrics(prev => ({
          ...prev,
          TT: navigationTiming.loadEventEnd - navigationTiming.startTime
        }));
      }
    }

    const fcpObserver = new PerformanceObserver((entryList) => {
      const entries = entryList.getEntries();
      entries.forEach(entry => {
        if (entry.name === 'first-contentful-paint') {
          setMetrics(prev => {
            const newMetrics = { ...prev, FCP: entry.startTime };
            metricsRef.current = newMetrics;
            return newMetrics;
          });
        }
      });
    });

    try {
      fcpObserver.observe({ type: 'paint', buffered: true });
      observers.push(fcpObserver);
    } catch (e) {}

    const connection = navigator.connection || navigator.mozConnection || navigator.webkitConnection;
    if (connection) {
      const handleConnectionChange = () => {
        setIsSlowConnection(
          connection.effectiveType === '2g' ||
          connection.effectiveType === 'slow-2g' ||
          (connection.saveData === true)
        );
      };

      connection.addEventListener('change', handleConnectionChange);
      handleConnectionChange();
    }

    const handleLoad = () => {
      setLoadingState('loaded');

      const resourceEntries = performance.getEntriesByType('resource');
      setResourceTimings(resourceEntries);

      const updatedMetrics = metricsRef.current;
      const score = calculatePerformanceScore(updatedMetrics);
      setPerformanceScore(score);

      checkThresholds(updatedMetrics);
    };

    if (document.readyState === 'complete') {
      handleLoad();
    } else {
      window.addEventListener('load', handleLoad);
    }

    return () => {
      observers.forEach(observer => observer.disconnect());
      window.removeEventListener('load', handleLoad);
    };
  }, [enableLCP, enableFID, enableCLS, enableTTFB, onMetricChange, calculatePerformanceScore, checkThresholds]);

  useEffect(() => {
    if (!reportInterval || reportInterval <= 0) return;

    const interval = setInterval(() => {
      if (Object.keys(metricsRef.current).length > 0) {
        const score = calculatePerformanceScore(metricsRef.current);
        setPerformanceScore(score);
        checkThresholds(metricsRef.current);
      }
    }, reportInterval);

    return () => clearInterval(interval);
  }, [reportInterval, calculatePerformanceScore, checkThresholds]);

  const getScoreRating = useCallback(() => {
    if (performanceScore === null) return 'loading';
    if (performanceScore >= 90) return 'good';
    if (performanceScore >= 50) return 'needs-improvement';
    return 'poor';
  }, [performanceScore]);

  const getMetricRating = useCallback((metricName) => {
    const value = metrics[metricName];
    if (value === null) return 'loading';
    if (!thresholds[metricName]) return 'unknown';
    if (value <= thresholds[metricName].good) return 'good';
    if (value <= thresholds[metricName].poor) return 'needs-improvement';
    return 'poor';
  }, [metrics]);

  const getReport = useCallback(() => {
    return {
      metrics,
      performanceScore,
      performanceRating: getScoreRating(),
      isSlowConnection,
      resourceTimings: resourceTimings.slice(0, 50),
      timestamp: Date.now()
    };
  }, [metrics, performanceScore, getScoreRating, isSlowConnection, resourceTimings]);

  const logMetrics = useCallback(() => {
    const report = getReport();
    console.log('Performance Report:', report);
    return report;
  }, [getReport]);

  return {
    metrics,
    performanceScore,
    performanceRating: getScoreRating(),
    isSlowConnection,
    loadingState,
    resourceTimings: resourceTimings.slice(0, 50),
    getScoreRating,
    getMetricRating,
    getReport,
    logMetrics
  };
};

export default usePerformanceMonitor;
