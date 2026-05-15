import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import SEO, { getSEOMeta } from '../components/SEO';
import LogList from '../components/LogList';
import LogFilter from '../components/LogFilter';
import Pagination from '../components/ui/Pagination';
import Loading from '../components/ui/Loading';
import Alert from '../components/ui/Alert';
import { formatDateTime } from '../i18n/dateFormat';

const LogsPage = () => {
  const { t } = useTranslation();
  const [logs, setLogs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [currentPage, setCurrentPage] = useState(1);
  const [totalLogs, setTotalLogs] = useState(0);
  const seoMeta = getSEOMeta('logs');
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
        setError(errorData.error || t('logs.fetchFailed'));
      }
    } catch (err) {
      setError(t('users.networkError'));
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
        setError(t('logs.exportFailed'));
      }
    } catch (err) {
      setError(t('users.networkError'));
    }
  };

  const handleExportJSON = () => handleExportLogs('json');
  const handleExportCSV = () => handleExportLogs('csv');

  const handleClearOldLogs = async () => {
    if (!window.confirm(t('logs.clearConfirm'))) return;

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
        Alert.success(t('logs.clearSuccess'));
        fetchLogs();
      } else {
        const errorData = await response.json();
        setError(errorData.error || t('logs.clearFailed'));
      }
    } catch (err) {
      setError(t('users.networkError'));
    }
  };

  return (
    <>
      <SEO {...seoMeta} />
      <div className="logs-page">
      <div className="page-header">
        <div>
          <h1>{t('logs.title')}</h1>
          <p>{t('logs.description')}</p>
        </div>
        <div className="header-actions">
          <button className="btn btn-secondary" onClick={() => setShowStats(!showStats)}>
            {showStats ? t('logs.hideStats') : t('logs.showStats')}
          </button>
          <div className="export-dropdown">
            <button className="btn btn-secondary dropdown-toggle">
              {t('logs.exportLogs')} ▾
            </button>
            <div className="export-dropdown-menu">
              <button onClick={handleExportCSV}>{t('logs.exportCSV')}</button>
              <button onClick={handleExportJSON}>{t('logs.exportJSON')}</button>
            </div>
          </div>
          <button className="btn btn-danger" onClick={handleClearOldLogs}>
            {t('logs.clearOldLogs')}
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
            <div className="stat-label">{t('logs.totalLogs')}</div>
          </div>
          <div className="stat-card">
            <div className="stat-value">{stats.last24h}</div>
            <div className="stat-label">{t('logs.last24h')}</div>
          </div>
          <div className="stat-card">
            <div className="stat-value">{stats.byLevel.error || 0}</div>
            <div className="stat-label">{t('logs.errorCount')}</div>
          </div>
          <div className="stat-card">
            <div className="stat-value">{stats.byLevel.warn || 0}</div>
            <div className="stat-label">{t('logs.warnCount')}</div>
          </div>
          <div className="stat-card">
            <div className="stat-value">{stats.byType.error || 0}</div>
            <div className="stat-label">{t('logs.errorType')}</div>
          </div>
          <div className="stat-card">
            <div className="stat-value">{stats.byType.security || 0}</div>
            <div className="stat-label">{t('logs.securityType')}</div>
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
              <h3>{t('logs.logDetails')}</h3>
              <button className="close-btn" onClick={() => setSelectedLog(null)}>×</button>
            </div>
            <div className="log-detail-body">
              <div className="log-detail-item">
                <label>{t('logs.time')}:</label>
                <span>{formatDateTime(selectedLog.timestamp)}</span>
              </div>
              <div className="log-detail-item">
                <label>{t('logs.type')}:</label>
                <span className={`log-type ${selectedLog.type}`}>{selectedLog.type}</span>
              </div>
              <div className="log-detail-item">
                <label>{t('logs.level')}:</label>
                <span className={`log-level ${selectedLog.level}`}>{selectedLog.level}</span>
              </div>
              <div className="log-detail-item">
                <label>{t('logs.user')}:</label>
                <span>{selectedLog.user_id || t('logs.system')}</span>
              </div>
              <div className="log-detail-item">
                <label>{t('logs.action')}:</label>
                <span>{selectedLog.action}</span>
              </div>
              {selectedLog.ip && (
                <div className="log-detail-item">
                  <label>{t('logs.ipAddress')}:</label>
                  <span>{selectedLog.ip}</span>
                </div>
              )}
              {selectedLog.user_agent && (
                <div className="log-detail-item">
                  <label>{t('logs.userAgent')}:</label>
                  <span className="user-agent">{selectedLog.user_agent}</span>
                </div>
              )}
              {selectedLog.details && (
                <div className="log-detail-item full-width">
                  <label>{t('logs.details')}:</label>
                  <pre className="log-details">{JSON.stringify(selectedLog.details, null, 2)}</pre>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
    </>
  );
};

export default LogsPage;Page;
