import React, { memo, useEffect, useState } from 'react';
import usePerformanceMonitor from '../hooks/usePerformanceMonitor';

const PerformanceMonitor = memo(({
  showOnDev = true,
  position = 'bottom-right',
  refreshInterval = 5000,
  showDetails = true,
  onScoreChange = null
}) => {
  const [isExpanded, setIsExpanded] = useState(false);
  const [isVisible, setIsVisible] = useState(false);

  const {
    metrics,
    performanceScore,
    performanceRating,
    isSlowConnection,
    loadingState,
    getMetricRating,
    getReport
  } = usePerformanceMonitor({
    enableLCP: true,
    enableFID: true,
    enableCLS: true,
    enableTTFB: true,
    reportInterval: refreshInterval
  });

  useEffect(() => {
    if (showOnDev || process.env.NODE_ENV === 'production') {
      setIsVisible(true);
    }
  }, [showOnDev]);

  useEffect(() => {
    if (onScoreChange && performanceScore !== null) {
      onScoreChange(performanceScore);
    }
  }, [performanceScore, onScoreChange]);

  if (!isVisible) return null;

  const getRatingColor = (rating) => {
    switch (rating) {
      case 'good':
        return '#10b981';
      case 'needs-improvement':
        return '#f59e0b';
      case 'poor':
        return '#ef4444';
      default:
        return '#6b7280';
    }
  };

  const getRatingLabel = (rating) => {
    switch (rating) {
      case 'good':
        return '优秀';
      case 'needs-improvement':
        return '需改进';
      case 'poor':
        return '较差';
      default:
        return '加载中';
    }
  };

  const formatMetric = (name, value) => {
    if (value === null) return '--';
    if (name === 'CLS') return value.toFixed(4);
    return `${Math.round(value)}ms`;
  };

  const styles = {
    container: {
      position: 'fixed',
      [position.includes('bottom') ? 'bottom' : 'top']: position.includes('right') ? '20px' : 'auto',
      [position.includes('top') ? 'top' : 'bottom']: position.includes('top') ? '20px' : 'auto',
      [position.includes('left') ? 'left' : 'right']: position.includes('left') ? '20px' : 'auto',
      zIndex: 9999,
      fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
      fontSize: '12px'
    },
    badge: {
      display: 'flex',
      alignItems: 'center',
      gap: '8px',
      padding: '8px 12px',
      backgroundColor: 'white',
      borderRadius: '8px',
      boxShadow: '0 4px 12px rgba(0, 0, 0, 0.15)',
      cursor: 'pointer',
      transition: 'all 0.2s ease'
    },
    scoreCircle: {
      width: '36px',
      height: '36px',
      borderRadius: '50%',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      color: 'white',
      fontWeight: 'bold',
      fontSize: '14px'
    },
    label: {
      display: 'flex',
      flexDirection: 'column',
      gap: '2px'
    },
    title: {
      fontWeight: '600',
      color: '#1f2937'
    },
    subtitle: {
      color: '#6b7280',
      fontSize: '11px'
    },
    expandedPanel: {
      position: 'absolute',
      [position.includes('bottom') ? 'bottom' : 'top']: position.includes('bottom') ? '50px' : 'auto',
      [position.includes('top') ? 'top' : 'bottom']: position.includes('top') ? '50px' : 'auto',
      [position.includes('left') ? 'left' : 'right']: '0px',
      width: '280px',
      backgroundColor: 'white',
      borderRadius: '12px',
      boxShadow: '0 8px 24px rgba(0, 0, 0, 0.2)',
      padding: '16px',
      marginTop: position.includes('bottom') ? '0' : '10px',
      marginBottom: position.includes('top') ? '0' : '10px'
    },
    metricRow: {
      display: 'flex',
      justifyContent: 'space-between',
      alignItems: 'center',
      padding: '8px 0',
      borderBottom: '1px solid #f3f4f6'
    },
    metricLabel: {
      color: '#374151',
      fontWeight: '500'
    },
    metricValue: {
      fontFamily: 'monospace',
      color: '#1f2937'
    },
    badgeDot: {
      width: '8px',
      height: '8px',
      borderRadius: '50%',
      animation: loadingState === 'loading' ? 'pulse 1.5s infinite' : 'none'
    }
  };

  return (
    <div style={styles.container}>
      <div
        style={styles.badge}
        onClick={() => setIsExpanded(!isExpanded)}
        onMouseEnter={(e) => e.currentTarget.style.transform = 'scale(1.02)'}
        onMouseLeave={(e) => e.currentTarget.style.transform = 'scale(1)'}
      >
        <div
          style={{
            ...styles.scoreCircle,
            backgroundColor: getRatingColor(performanceRating)
          }}
        >
          {performanceScore !== null ? Math.round(performanceScore) : '--'}
        </div>
        <div style={styles.label}>
          <span style={styles.title}>性能得分</span>
          <span style={styles.subtitle}>
            {getRatingLabel(performanceRating)}
            {isSlowConnection && ' • 慢速网络'}
          </span>
        </div>
        <div
          style={{
            ...styles.badgeDot,
            backgroundColor: loadingState === 'loading' ? '#f59e0b' : getRatingColor(performanceRating)
          }}
        />
      </div>

      {isExpanded && showDetails && (
        <div style={styles.expandedPanel}>
          <div style={{ marginBottom: '12px', fontWeight: '600', color: '#1f2937' }}>
            性能指标
          </div>

          {[
            { name: 'LCP', label: '最大内容绘制', description: '加载性能' },
            { name: 'FID', label: '首次输入延迟', description: '交互性' },
            { name: 'CLS', label: '布局偏移', description: '视觉稳定性' },
            { name: 'TTFB', label: '首字节时间', description: '服务器响应' }
          ].map(({ name, label, description }) => {
            const value = metrics[name];
            const rating = getMetricRating(name);

            return (
              <div key={name} style={styles.metricRow}>
                <div>
                  <div style={styles.metricLabel}>{label}</div>
                  <div style={{ fontSize: '10px', color: '#9ca3af' }}>{description}</div>
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <span style={styles.metricValue}>{formatMetric(name, value)}</span>
                  <div
                    style={{
                      width: '8px',
                      height: '8px',
                      borderRadius: '50%',
                      backgroundColor: getRatingColor(rating)
                    }}
                  />
                </div>
              </div>
            );
          })}

          <button
            onClick={() => {
              const report = getReport();
              console.log('Performance Report:', report);
              alert('性能报告已打印到控制台');
            }}
            style={{
              marginTop: '12px',
              width: '100%',
              padding: '8px',
              backgroundColor: '#3b82f6',
              color: 'white',
              border: 'none',
              borderRadius: '6px',
              cursor: 'pointer',
              fontSize: '12px',
              fontWeight: '500'
            }}
          >
            查看完整报告
          </button>
        </div>
      )}

      <style>
        {`
          @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.5; }
          }
        `}
      </style>
    </div>
  );
});

PerformanceMonitor.displayName = 'PerformanceMonitor';

export default PerformanceMonitor;
