import React, { useState, useEffect } from 'react';
import LogList from '../components/LogList';
import LogFilter from '../components/LogFilter';
import Pagination from '../components/ui/Pagination';
import Loading from '../components/ui/Loading';
import Alert from '../components/ui/Alert';

const LogsPage = () => {
  const [logs, setLogs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [currentPage, setCurrentPage] = useState(1);
  const [totalLogs, setTotalLogs] = useState(0);
  const [filters, setFilters] = useState({
    type: '',
    level: '',
    startDate: '',
    endDate: '',
    search: ''
  });
  const [selectedLog, setSelectedLog] = useState(null);
  const [stats, setStats] = useState({
    total: 0,
    byLevel: {},
    byType: {},
    last24h: 0
  });
  const [showStats, setShowStats] = useState(true);

  const pageSize = 20;

  useEffect(() => {
    fetchLogs();
    fetchStats();
  }, [currentPage, filters]);

  const fetchStats = async () => {
    try {
      const token = localStorage.getItem('authToken');
      const queryParams = new URLSearchParams({
        ...Object.fromEntries(
          Object.entries(filters).filter(([_, value]) => value !== '')
        ),
        limit: '1000'
      });

      const response = await fetch(`/api/v1/admin/logs?${queryParams}`, {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });

      if (response.ok) {
        const data = await response.json();
        const logData = data.logs || [];

        const byLevel = {};
        const byType = {};
        let last24h = 0;
        const now = Date.now();
        const dayAgo = now - 24 * 60 * 60 * 1000;

        logData.forEach(log => {
          byLevel[log.level] = (byLevel[log.level] || 0) + 1;
          byType[log.type] = (byType[log.type] || 0) + 1;

          const logTime = new Date(log.timestamp).getTime();
          if (logTime > dayAgo) {
            last24h++;
          }
        });

        setStats({
          total: data.total || 0,
          byLevel,
          byType,
          last24h
        });
      }
    } catch (err) {
      console.error('Failed to fetch stats:', err);
    }
  };

  const fetchLogs = async () => {
    setLoading(true);
    setError('');

    try {
      const token = localStorage.getItem('authToken');
      const queryParams = new URLSearchParams({
        page: currentPage,
        limit: pageSize,
        ...Object.fromEntries(
          Object.entries(filters).filter(([_, value]) => value !== '')
        )
      });

      const response = await fetch(`/api/v1/admin/logs?${queryParams}`, {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });

      if (response.ok) {
        const data = await response.json();
        setLogs(data.logs || []);
        setTotalLogs(data.total || 0);
      } else {
        const errorData = await response.json();
        setError(errorData.error || '获取日志列表失败');
      }
    } catch (err) {
      setError('网络错误，请稍后重试');
    } finally {
      setLoading(false);
    }
  };

  const handleFilterChange = (newFilters) => {
    setFilters(newFilters);
    setCurrentPage(1);
  };

  const handleExportLogs = async (format = 'csv') => {
    try {
      const token = localStorage.getItem('authToken');
      const queryParams = new URLSearchParams({
        ...Object.fromEntries(
          Object.entries(filters).filter(([_, value]) => value !== '')
        ),
        format
      });

      const response = await fetch(`/api/v1/admin/logs/export?${queryParams}`, {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });

      if (response.ok) {
        const contentType = response.headers.get('Content-Type');
        let blob;
        let filename;

        if (format === 'json' || contentType.includes('application/json')) {
          const data = await response.json();
          blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
          filename = `logs_${new Date().toISOString().split('T')[0]}.json`;
        } else {
          blob = await response.blob();
          filename = `logs_${new Date().toISOString().split('T')[0]}.csv`;
        }

        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        window.URL.revokeObjectURL(url);
      } else {
        setError('导出日志失败');
      }
    } catch (err) {
      setError('导出失败，请稍后重试');
    }
  };

  const handleExportJSON = () => handleExportLogs('json');
  const handleExportCSV = () => handleExportLogs('csv');

  const handleClearOldLogs = async () => {
    if (!window.confirm('确定要清除30天前的日志吗？此操作不可撤销。')) return;

    try {
      const token = localStorage.getItem('authToken');
      const response = await fetch('/api/v1/admin/logs/clear', {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({ days: 30 })
      });

      if (response.ok) {
        setError('');
        Alert.success('旧日志已清除');
        fetchLogs();
      } else {
        const errorData = await response.json();
        setError(errorData.error || '清除日志失败');
      }
    } catch (err) {
      setError('网络错误，请稍后重试');
    }
  };

  return (
    <div className="logs-page">
      <div className="page-header">
        <div>
          <h1>日志管理</h1>
          <p>查看和搜索系统操作日志、错误日志</p>
        </div>
        <div className="header-actions">
          <button className="btn btn-secondary" onClick={() => setShowStats(!showStats)}>
            {showStats ? '隐藏统计' : '显示统计'}
          </button>
          <div className="export-dropdown">
            <button className="btn btn-secondary dropdown-toggle">
              导出日志 ▾
            </button>
            <div className="export-dropdown-menu">
              <button onClick={handleExportCSV}>导出 CSV</button>
              <button onClick={handleExportJSON}>导出 JSON</button>
            </div>
          </div>
          <button className="btn btn-danger" onClick={handleClearOldLogs}>
            清除旧日志
          </button>
        </div>
      </div>

      {error && (
        <Alert
          type="error"
          message={error}
          closable
          onClose={() => setError('')}
        />
      )}

      {showStats && (
        <div className="log-stats">
          <div className="stat-card">
            <div className="stat-value">{stats.total}</div>
            <div className="stat-label">总日志数</div>
          </div>
          <div className="stat-card">
            <div className="stat-value">{stats.last24h}</div>
            <div className="stat-label">24小时内</div>
          </div>
          <div className="stat-card">
            <div className="stat-value">{stats.byLevel.error || 0}</div>
            <div className="stat-label">错误数</div>
          </div>
          <div className="stat-card">
            <div className="stat-value">{stats.byLevel.warn || 0}</div>
            <div className="stat-label">警告数</div>
          </div>
          <div className="stat-card">
            <div className="stat-value">{stats.byType.error || 0}</div>
            <div className="stat-label">错误类型</div>
          </div>
          <div className="stat-card">
            <div className="stat-value">{stats.byType.security || 0}</div>
            <div className="stat-label">安全类型</div>
          </div>
        </div>
      )}

      <LogFilter filters={filters} onFilterChange={handleFilterChange} />

      <LogList
        logs={logs}
        loading={loading}
        onViewDetails={setSelectedLog}
        selectedLog={selectedLog}
      />

      {!loading && logs.length > 0 && (
        <Pagination
          current={currentPage}
          total={totalLogs}
          pageSize={pageSize}
          onChange={setCurrentPage}
        />
      )}

      {selectedLog && (
        <div className="log-detail-modal" onClick={() => setSelectedLog(null)}>
          <div className="log-detail-content" onClick={(e) => e.stopPropagation()}>
            <div className="log-detail-header">
              <h3>日志详情</h3>
              <button className="close-btn" onClick={() => setSelectedLog(null)}>×</button>
            </div>
            <div className="log-detail-body">
              <div className="log-detail-item">
                <label>时间:</label>
                <span>{new Date(selectedLog.timestamp).toLocaleString('zh-CN')}</span>
              </div>
              <div className="log-detail-item">
                <label>类型:</label>
                <span className={`log-type ${selectedLog.type}`}>{selectedLog.type}</span>
              </div>
              <div className="log-detail-item">
                <label>级别:</label>
                <span className={`log-level ${selectedLog.level}`}>{selectedLog.level}</span>
              </div>
              <div className="log-detail-item">
                <label>用户:</label>
                <span>{selectedLog.user_id || '系统'}</span>
              </div>
              <div className="log-detail-item">
                <label>操作:</label>
                <span>{selectedLog.action}</span>
              </div>
              {selectedLog.ip && (
                <div className="log-detail-item">
                  <label>IP地址:</label>
                  <span>{selectedLog.ip}</span>
                </div>
              )}
              {selectedLog.user_agent && (
                <div className="log-detail-item">
                  <label>用户代理:</label>
                  <span className="user-agent">{selectedLog.user_agent}</span>
                </div>
              )}
              {selectedLog.details && (
                <div className="log-detail-item full-width">
                  <label>详细信息:</label>
                  <pre className="log-details">{JSON.stringify(selectedLog.details, null, 2)}</pre>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default LogsPage;
