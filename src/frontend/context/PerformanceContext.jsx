import React, { createContext, useContext, useState, useEffect, useCallback, useMemo } from 'react';
import usePerformanceMonitor from '../hooks/usePerformanceMonitor';

const PerformanceContext = createContext(null);

export const usePerformance = () => {
  const context = useContext(PerformanceContext);
  if (!context) {
    throw new Error('usePerformance must be used within PerformanceProvider');
  }
  return context;
};

export const PerformanceProvider = ({ children, options = {} }) => {
  const [isMonitoring, setIsMonitoring] = useState(true);
  const [alerts, setAlerts] = useState([]);
  const [history, setHistory] = useState([]);

  const handleMetricChange = useCallback((metric, value) => {
    setHistory(prev => {
      const newEntry = {
        metric,
        value,
        timestamp: Date.now()
      };
      return [...prev.slice(-99), newEntry];
    });
  }, []);

  const handleThresholdExceeded = useCallback((exceededThresholds) => {
    setAlerts(prev => [...prev.slice(-9), ...exceededThresholds]);
  }, []);

  const {
    metrics,
    performanceScore,
    performanceRating,
    isSlowConnection,
    loadingState,
    getReport,
    logMetrics
  } = usePerformanceMonitor({
    enableLCP: true,
    enableFID: true,
    enableCLS: true,
    enableTTFB: true,
    reportInterval: options.reportInterval || 5000,
    onMetricChange: handleMetricChange,
    onThresholdExceeded: handleThresholdExceeded
  });

  const clearAlerts = useCallback(() => {
    setAlerts([]);
  }, []);

  const exportMetrics = useCallback(() => {
    const report = getReport();
    const dataStr = JSON.stringify(report, null, 2);
    const dataUri = 'data:application/json;charset=utf-8,' + encodeURIComponent(dataStr);

    const exportName = `performance-report-${Date.now()}.json`;
    const linkElement = document.createElement('a');
    linkElement.setAttribute('href', dataUri);
    linkElement.setAttribute('download', exportName);
    linkElement.click();

    return report;
  }, [getReport]);

  const value = useMemo(() => ({
    metrics,
    performanceScore,
    performanceRating,
    isSlowConnection,
    loadingState,
    isMonitoring,
    alerts,
    history,
    setIsMonitoring,
    clearAlerts,
    exportMetrics,
    logMetrics,
    getReport
  }), [
    metrics,
    performanceScore,
    performanceRating,
    isSlowConnection,
    loadingState,
    isMonitoring,
    alerts,
    history,
    clearAlerts,
    exportMetrics,
    logMetrics
  ]);

  return (
    <PerformanceContext.Provider value={value}>
      {children}
    </PerformanceContext.Provider>
  );
};

export default PerformanceProvider;
