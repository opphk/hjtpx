import React, { useState, useEffect, Suspense, lazy } from 'react';

const LazyChart = lazy(() => import('./components/Chart'));

const PerformanceOptimization = () => {
  const [showChart, setShowChart] = useState(false);
  const [metrics, setMetrics] = useState({
    bundleSize: 0,
    loadTime: 0,
    firstPaint: 0,
    domContentLoaded: 0
  });

  useEffect(() => {
    if (window.performance) {
      const perfData = window.performance.timing;
      const metrics = {
        loadTime: perfData.loadEventEnd - perfData.navigationStart,
        firstPaint: perfData.loadEventEnd - perfData.navigationStart,
        domContentLoaded: perfData.domContentLoadedEventEnd - perfData.navigationStart,
        bundleSize: getBundleSize()
      };
      setMetrics(metrics);
    }
  }, []);

  const getBundleSize = () => {
    const scripts = document.querySelectorAll('script[src]');
    let totalSize = 0;
    scripts.forEach(script => {
      const size = script.getAttribute('data-size');
      if (size) {
        totalSize += parseInt(size, 10);
      }
    });
    return totalSize;
  };

  const toggleChart = () => {
    setShowChart(!showChart);
  };

  return (
    <div className="performance-container">
      <h2>性能优化指标</h2>
      
      <div className="metrics-grid">
        <div className="metric-card">
          <h3>Bundle 大小</h3>
          <p className="metric-value">{(metrics.bundleSize / 1024).toFixed(2)} KB</p>
        </div>
        
        <div className="metric-card">
          <h3>页面加载时间</h3>
          <p className="metric-value">{metrics.loadTime} ms</p>
        </div>
        
        <div className="metric-card">
          <h3>DOM 加载时间</h3>
          <p className="metric-value">{metrics.domContentLoaded} ms</p>
        </div>
      </div>

      <button onClick={toggleChart}>
        {showChart ? '隐藏图表' : '显示图表'}
      </button>

      {showChart && (
        <Suspense fallback={<div>加载图表中...</div>}>
          <LazyChart />
        </Suspense>
      )}

      <BundleAnalyzer />
    </div>
  );
};

const BundleAnalyzer = () => {
  const [stats, setStats] = useState(null);

  useEffect(() => {
    const loadStats = async () => {
      try {
        const response = await fetch('/stats.json');
        const data = await response.json();
        setStats(data);
      } catch (error) {
        console.error('Failed to load bundle stats:', error);
      }
    };

    if (window.location.search.includes('analyze=true')) {
      loadStats();
    }
  }, []);

  if (!stats) {
    return null;
  }

  return (
    <div className="bundle-analyzer">
      <h3>Bundle 分析</h3>
      <pre>{JSON.stringify(stats, null, 2)}</pre>
    </div>
  );
};

export default PerformanceOptimization;
