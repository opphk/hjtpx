import React, { createContext, useContext, useEffect, useState, useCallback } from 'react';
import { usePerformanceMetrics, useNetworkStatus } from '../hooks/usePerformance';

const PerformanceMonitorContext = createContext(null);

export const usePerformanceMonitor = () => {
  const context = useContext(PerformanceMonitorContext);
  if (!context) {
    throw new Error('usePerformanceMonitor must be used within PerformanceMonitorProvider');
  }
  return context;
};

export const PerformanceMonitor = ({ 
  children, 
  enabled = process.env.NODE_ENV === 'development',
  onThresholdExceeded,
  thresholds = {
    lcp: 2500,
    fcp: 1800,
    cls: 0.1,
    fid: 100,
    ttfb: 800
  }
}) => {
  const { metrics, webVitalsSupported, isLoading } = usePerformanceMetrics();
  const { isOnline, effectiveType, downlink } = useNetworkStatus();
  const [warnings, setWarnings] = useState([]);
  const [showPanel, setShowPanel] = useState(false);

  useEffect(() => {
    if (!enabled || isLoading) return;

    const newWarnings = [];

    if (metrics.lcp && metrics.lcp > thresholds.lcp) {
      newWarnings.push({
        type: 'LCP',
        value: metrics.lcp,
        threshold: thresholds.lcp,
        message: 'Largest Contentful Paint exceeded threshold'
      });
    }

    if (metrics.fcp && metrics.fcp > thresholds.fcp) {
      newWarnings.push({
        type: 'FCP',
        value: metrics.fcp,
        threshold: thresholds.fcp,
        message: 'First Contentful Paint exceeded threshold'
      });
    }

    if (metrics.cls && metrics.cls > thresholds.cls) {
      newWarnings.push({
        type: 'CLS',
        value: metrics.cls,
        threshold: thresholds.cls,
        message: 'Cumulative Layout Shift exceeded threshold'
      });
    }

    if (metrics.ttfb && metrics.ttfb > thresholds.ttfb) {
      newWarnings.push({
        type: 'TTFB',
        value: metrics.ttfb,
        threshold: thresholds.ttfb,
        message: 'Time to First Byte exceeded threshold'
      });
    }

    setWarnings(newWarnings);

    if (newWarnings.length > 0 && onThresholdExceeded) {
      onThresholdExceeded(newWarnings);
    }
  }, [metrics, thresholds, enabled, isLoading, onThresholdExceeded]);

  const getScore = useCallback((metric, value) => {
    if (!value) return 'unknown';
    
    const scores = {
      lcp: { good: 2500, needsImprovement: 4000 },
      fcp: { good: 1800, needsImprovement: 3000 },
      cls: { good: 0.1, needsImprovement: 0.25 },
      ttfb: { good: 800, needsImprovement: 1800 }
    };

    const thresholds = scores[metric];
    if (!thresholds) return 'unknown';

    if (value <= thresholds.good) return 'good';
    if (value <= thresholds.needsImprovement) return 'needs-improvement';
    return 'poor';
  }, []);

  const formatValue = useCallback((metric, value) => {
    if (!value) return 'N/A';
    
    if (['lcp', 'fcp', 'ttfb', 'fid'].includes(metric)) {
      return `${(value / 1000).toFixed(2)}s`;
    }
    if (metric === 'cls') {
      return value.toFixed(3);
    }
    return value;
  }, []);

  const value = {
    metrics,
    webVitalsSupported,
    isLoading,
    isOnline,
    effectiveType,
    downlink,
    warnings,
    showPanel,
    setShowPanel,
    getScore,
    formatValue,
    thresholds
  };

  return (
    <PerformanceMonitorContext.Provider value={value}>
      {children}
      {enabled && showPanel && <PerformancePanel />}
    </PerformanceMonitorContext.Provider>
  );
};

const PerformancePanel = () => {
  const {
    metrics,
    webVitalsSupported,
    isLoading,
    isOnline,
    effectiveType,
    downlink,
    warnings,
    setShowPanel,
    getScore,
    formatValue
  } = usePerformanceMonitor();

  const getScoreColor = (score) => {
    switch (score) {
      case 'good': return '#4caf50';
      case 'needs-improvement': return '#ff9800';
      case 'poor': return '#f44336';
      default: return '#9e9e9e';
    }
  };

  return (
    <div style={{
      position: 'fixed',
      bottom: 20,
      right: 20,
      width: 320,
      backgroundColor: 'white',
      borderRadius: 12,
      boxShadow: '0 4px 20px rgba(0, 0, 0, 0.15)',
      fontFamily: 'system-ui, -apple-system, sans-serif',
      fontSize: 12,
      zIndex: 9999,
      overflow: 'hidden'
    }}>
      <div style={{
        padding: '12px 16px',
        backgroundColor: '#f5f5f5',
        borderBottom: '1px solid #e0e0e0',
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        cursor: 'move'
      }}>
        <div style={{ fontWeight: 600, color: '#333' }}>
          ⚡ Performance Monitor
        </div>
        <button
          onClick={() => setShowPanel(false)}
          style={{
            background: 'none',
            border: 'none',
            fontSize: 18,
            cursor: 'pointer',
            color: '#666',
            padding: 0,
            width: 24,
            height: 24,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center'
          }}
        >
          ×
        </button>
      </div>

      <div style={{ padding: 16 }}>
        {isLoading ? (
          <div style={{ textAlign: 'center', padding: 20, color: '#666' }}>
            Loading metrics...
          </div>
        ) : (
          <>
            <div style={{ marginBottom: 16 }}>
              <div style={{ 
                fontSize: 11, 
                color: '#666', 
                marginBottom: 8,
                textTransform: 'uppercase',
                letterSpacing: '0.5px'
              }}>
                Network Status
              </div>
              <div style={{ display: 'flex', gap: 8 }}>
                <div style={{
                  flex: 1,
                  padding: '8px 12px',
                  backgroundColor: isOnline ? '#e8f5e9' : '#ffebee',
                  borderRadius: 6,
                  textAlign: 'center',
                  fontWeight: 500,
                  color: isOnline ? '#2e7d32' : '#c62828'
                }}>
                  {isOnline ? '🟢 Online' : '🔴 Offline'}
                </div>
                <div style={{
                  flex: 1,
                  padding: '8px 12px',
                  backgroundColor: '#e3f2fd',
                  borderRadius: 6,
                  textAlign: 'center',
                  fontWeight: 500,
                  color: '#1565c0'
                }}>
                  {effectiveType?.toUpperCase() || '4G'}
                </div>
              </div>
            </div>

            <div style={{ marginBottom: 16 }}>
              <div style={{ 
                fontSize: 11, 
                color: '#666', 
                marginBottom: 8,
                textTransform: 'uppercase',
                letterSpacing: '0.5px'
              }}>
                Core Web Vitals
              </div>
              
              {[
                { key: 'fcp', label: 'FCP' },
                { key: 'lcp', label: 'LCP' },
                { key: 'cls', label: 'CLS' },
                { key: 'ttfb', label: 'TTFB' },
                { key: 'fid', label: 'FID' }
              ].map(metric => {
                const value = metrics[metric.key];
                const score = getScore(metric.key, value);
                
                return (
                  <div key={metric.key} style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    padding: '6px 0',
                    borderBottom: '1px solid #f0f0f0'
                  }}>
                    <span style={{ fontWeight: 500, color: '#333' }}>
                      {metric.label}
                    </span>
                    <span style={{
                      fontWeight: 600,
                      color: getScoreColor(score),
                      backgroundColor: `${getScoreColor(score)}20`,
                      padding: '2px 8px',
                      borderRadius: 4
                    }}>
                      {formatValue(metric.key, value)}
                    </span>
                  </div>
                );
              })}
            </div>

            {warnings.length > 0 && (
              <div style={{
                padding: 12,
                backgroundColor: '#fff3e0',
                borderRadius: 6,
                border: '1px solid #ffb74d'
              }}>
                <div style={{
                  fontSize: 11,
                  color: '#e65100',
                  fontWeight: 600,
                  marginBottom: 8
                }}>
                  ⚠️ Warnings
                </div>
                {warnings.map((warning, i) => (
                  <div key={i} style={{
                    fontSize: 11,
                    color: '#bf360c',
                    marginTop: 4
                  }}>
                    {warning.type}: {formatValue(warning.type.toLowerCase(), warning.value)} exceeds {formatValue(warning.type.toLowerCase(), warning.threshold)}
                  </div>
                ))}
              </div>
            )}

            {webVitalsSupported && (
              <div style={{
                marginTop: 12,
                fontSize: 10,
                color: '#999',
                textAlign: 'center'
              }}>
                Web Vitals API supported ✅
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
};

export default PerformanceMonitor;
