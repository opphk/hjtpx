import React, { useEffect, useState } from 'react';
import * as VITE from 'vite';

const BundleAnalyzer = () => {
  const [bundleStats, setBundleStats] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    const analyzeBundle = async () => {
      try {
        const statsUrl = '/stats.json';
        const response = await fetch(statsUrl);
        
        if (!response.ok) {
          throw new Error('Bundle stats not available');
        }
        
        const stats = await response.json();
        setBundleStats(processStats(stats));
        setLoading(false);
      } catch (err) {
        setError(err.message);
        setLoading(false);
        generateMockStats();
      }
    };

    analyzeBundle();
  }, []);

  const processStats = (stats) => {
    const modules = Object.keys(stats.modules || {});
    const totalSize = modules.reduce((acc, mod) => {
      return acc + (stats.modules[mod].size || 0);
    }, 0);

    const chunks = stats.chunks || [];
    const chunkSizes = chunks.map(chunk => ({
      name: chunk.names?.[0] || 'unnamed',
      size: chunk.size || 0,
      sizeFormatted: formatBytes(chunk.size || 0)
    }));

    return {
      totalSize,
      totalSizeFormatted: formatBytes(totalSize),
      moduleCount: modules.length,
      chunks: chunkSizes,
      assetCount: stats.assets?.length || 0
    };
  };

  const generateMockStats = () => {
    setBundleStats({
      totalSize: 245678,
      totalSizeFormatted: '239.9 KB',
      moduleCount: 42,
      chunks: [
        { name: 'main', size: 125000, sizeFormatted: '122.1 KB' },
        { name: 'vendor', size: 89000, sizeFormatted: '86.9 KB' },
        { name: 'react', size: 45000, sizeFormatted: '43.9 KB' },
        { name: 'charts', size: 32000, sizeFormatted: '31.3 KB' }
      ],
      assetCount: 8
    });
  };

  const formatBytes = (bytes) => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  if (loading) {
    return <div>加载Bundle分析中...</div>;
  }

  if (error) {
    console.warn('Bundle stats error:', error);
  }

  if (!bundleStats) {
    return null;
  }

  return (
    <div className="bundle-analyzer">
      <h3>📦 Bundle 分析报告</h3>
      
      <div className="stats-overview">
        <div className="stat-card">
          <div className="stat-label">总大小</div>
          <div className="stat-value">{bundleStats.totalSizeFormatted}</div>
        </div>
        
        <div className="stat-card">
          <div className="stat-label">模块数</div>
          <div className="stat-value">{bundleStats.moduleCount}</div>
        </div>
        
        <div className="stat-card">
          <div className="stat-label">代码块数</div>
          <div className="stat-value">{bundleStats.chunks.length}</div>
        </div>
        
        <div className="stat-card">
          <div className="stat-label">资源文件数</div>
          <div className="stat-value">{bundleStats.assetCount}</div>
        </div>
      </div>

      <div className="chunks-breakdown">
        <h4>代码块详情</h4>
        <table>
          <thead>
            <tr>
              <th>代码块</th>
              <th>大小</th>
              <th>占比</th>
            </tr>
          </thead>
          <tbody>
            {bundleStats.chunks.map((chunk, index) => (
              <tr key={index}>
                <td>{chunk.name}</td>
                <td>{chunk.sizeFormatted}</td>
                <td>
                  {((chunk.size / bundleStats.totalSize) * 100).toFixed(1)}%
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="optimization-tips">
        <h4>💡 优化建议</h4>
        <ul>
          {bundleStats.totalSize > 500000 && (
            <li>Bundle大小超过500KB，建议进一步拆分</li>
          )}
          {bundleStats.chunks.length < 5 && (
            <li>考虑添加更多代码分割点以提高加载性能</li>
          )}
          {bundleStats.chunks.some(c => c.size > 150000) && (
            <li>某些代码块过大，考虑懒加载</li>
          )}
          <li>启用Gzip/Brotli压缩</li>
          <li>使用CDN加速资源加载</li>
        </ul>
      </div>

      <div className="build-info">
        <small>
          生成时间: {new Date().toLocaleString('zh-CN')}
          {error && <span> (使用模拟数据)</span>}
        </small>
      </div>
    </div>
  );
};

export default BundleAnalyzer;
