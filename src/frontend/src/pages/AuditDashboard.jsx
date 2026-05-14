import React, { useState, useEffect } from 'react';
import Loading from '../components/ui/Loading';
import Alert from '../components/ui/Alert';
import Pagination from '../components/ui/Pagination';

const AuditDashboard = () => {
  const [auditLogs, setAuditLogs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [currentPage, setCurrentPage] = useState(1);
  const [totalLogs, setTotalLogs] = useState(0);
  const [filters, setFilters] = useState({
    userId: '',
    action: '',
    startDate: '',
    endDate: ''
  });
  const [stats, setStats] = useState({
    totalAudits: 0,
    todayAudits: 0,
    userChanges: 0,
    systemChanges: 0,
    byAction: {},
    byUser: {}
  });

  const pageSize = 20;

  useEffect(() => {
    fetchAuditLogs();
    fetchStats();
  }, [currentPage, filters]);

  const fetchStats = async () => {
    try {
      const token = localStorage.getItem('authToken');
      const queryParams = new URLSearchParams({
        limit: '1000'
      });

      const response = await fetch(`/api/v1/admin/audit?${queryParams}`, {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });

      if (response.ok) {
        const data = await response.json();
        const logs = data.audit_logs || [];

        const byAction = {};
        const byUser = {};
        let todayCount = 0;
        const today = new Date().toDateString();

        logs.forEach(log => {
          byAction[log.action] = (byAction[log.action] || 0) + 1;
          byUser[log.user_id] = (byUser[log.user_id] || 0) + 1;

          const logDate = new Date(log.created_at).toDateString();
          if (logDate === today) {
            todayCount++;
          }
        });

        setStats({
          totalAudits: data.total || 0,
          todayAudits: todayCount,
          userChanges: logs.filter(l => l.action.includes('user') || l.action.includes('User')).length,
          systemChanges: logs.filter(l => l.action.includes('system') || l.action.includes('System')).length,
          byAction,
          byUser
        });
      }
    } catch (err) {
      console.error('Failed to fetch audit stats:', err);
    }
  };

  const fetchAuditLogs = async () => {
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

      const response = await fetch(`/api/v1/admin/audit?${queryParams}`, {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });

      if (response.ok) {
        const data = await response.json();
        setAuditLogs(data.audit_logs || []);
        setTotalLogs(data.total || 0);
      } else {
        const errorData = await response.json();
        setError(errorData.error || '获取审计日志失败');
      }
    } catch (err) {
      setError('网络错误，请稍后重试');
    } finally {
      setLoading(false);
    }
  };

  const handleFilterChange = (key, value) => {
    setFilters(prev => ({ ...prev, [key]: value }));
    setCurrentPage(1);
  };

  const handleGenerateReport = async (format = 'csv') => {
    try {
      const token = localStorage.getItem('authToken');
      const queryParams = new URLSearchParams({
        ...Object.fromEntries(
          Object.entries(filters).filter(([_, value]) => value !== '')
        ),
        format,
        report: 'true'
      });

      const response = await fetch(`/api/v1/admin/audit/export?${queryParams}`, {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });

      if (response.ok) {
        const contentType = response.headers.get('Content-Type');
        let blob;
        let filename;

        if (format === 'json') {
          const data = await response.json();
          blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
          filename = `audit_report_${new Date().toISOString().split('T')[0]}.json`;
        } else {
          blob = await response.blob();
          filename = `audit_report_${new Date().toISOString().split('T')[0]}.csv`;
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
        setError('生成报告失败');
      }
    } catch (err) {
      setError('生成报告失败，请稍后重试');
    }
  };

  const getActionBadge = (action) => {
    if (action.includes('create') || action.includes('CREATE')) {
      return <span className="action-badge action-create">创建</span>;
    }
    if (action.includes('update') || action.includes('UPDATE') || action.includes('edit') || action.includes('EDIT')) {
      return <span className="action-badge action-update">更新</span>;
    }
    if (action.includes('delete') || action.includes('DELETE')) {
      return <span className="action-badge action-delete">删除</span>;
    }
    if (action.includes('login') || action.includes('LOGIN') || action.includes('logout') || action.includes('LOGOUT')) {
      return <span className="action-badge action-login">登录</span>;
    }
    return <span className="action-badge action-other">其他</span>;
  };

  if (loading && auditLogs.length === 0) {
    return <Loading text="加载审计数据..." />;
  }

  return (
    <div className="audit-dashboard">
      <div className="page-header">
        <div>
          <h1>审计仪表板</h1>
          <p>监控用户活动、系统变更和安全事件</p>
        </div>
        <div className="header-actions">
          <button className="btn btn-secondary" onClick={() => handleGenerateReport('csv')}>
            导出报告 CSV
          </button>
          <button className="btn btn-secondary" onClick={() => handleGenerateReport('json')}>
            导出报告 JSON
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

      <div className="audit-stats">
        <div className="stat-card">
          <div className="stat-icon stat-icon-total">📋</div>
          <div className="stat-content">
            <div className="stat-value">{stats.totalAudits}</div>
            <div className="stat-label">总审计记录</div>
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-icon stat-icon-today">📅</div>
          <div className="stat-content">
            <div className="stat-value">{stats.todayAudits}</div>
            <div className="stat-label">今日审计</div>
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-icon stat-icon-users">👥</div>
          <div className="stat-content">
            <div className="stat-value">{stats.userChanges}</div>
            <div className="stat-label">用户变更</div>
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-icon stat-icon-system">⚙️</div>
          <div className="stat-content">
            <div className="stat-value">{stats.systemChanges}</div>
            <div className="stat-label">系统变更</div>
          </div>
        </div>
      </div>

      <div className="audit-filters">
        <div className="filter-row">
          <div className="filter-item">
            <label className="filter-label">用户ID</label>
            <input
              type="text"
              value={filters.userId}
              onChange={(e) => handleFilterChange('userId', e.target.value)}
              className="form-input"
              placeholder="输入用户ID"
            />
          </div>
          <div className="filter-item">
            <label className="filter-label">操作类型</label>
            <select
              value={filters.action}
              onChange={(e) => handleFilterChange('action', e.target.value)}
              className="form-select"
            >
              <option value="">所有操作</option>
              <option value="create">创建</option>
              <option value="update">更新</option>
              <option value="delete">删除</option>
              <option value="login">登录</option>
              <option value="logout">登出</option>
            </select>
          </div>
          <div className="filter-item">
            <label className="filter-label">开始日期</label>
            <input
              type="date"
              value={filters.startDate}
              onChange={(e) => handleFilterChange('startDate', e.target.value)}
              className="date-input"
            />
          </div>
          <div className="filter-item">
            <label className="filter-label">结束日期</label>
            <input
              type="date"
              value={filters.endDate}
              onChange={(e) => handleFilterChange('endDate', e.target.value)}
              className="date-input"
            />
          </div>
        </div>
        <div className="filter-row">
          <div className="filter-actions">
            <button
              className="btn btn-primary"
              onClick={fetchAuditLogs}
            >
              应用筛选
            </button>
            <button
              className="btn btn-secondary"
              onClick={() => {
                setFilters({
                  userId: '',
                  action: '',
                  startDate: '',
                  endDate: ''
                });
                setCurrentPage(1);
              }}
            >
              重置
            </button>
          </div>
        </div>
      </div>

      {Object.keys(stats.byAction).length > 0 && (
        <div className="audit-charts">
          <div className="chart-card">
            <h3>操作类型分布</h3>
            <div className="chart-content">
              {Object.entries(stats.byAction).slice(0, 5).map(([action, count]) => (
                <div key={action} className="chart-item">
                  <div className="chart-label">{action}</div>
                  <div className="chart-bar">
                    <div
                      className="chart-bar-fill"
                      style={{
                        width: `${(count / Math.max(...Object.values(stats.byAction))) * 100}%`
                      }}
                    ></div>
                  </div>
                  <div className="chart-value">{count}</div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      <div className="audit-logs">
        <h3>审计日志列表</h3>
        <div className="table-container">
          <table className="audit-table">
            <thead>
              <tr>
                <th>时间</th>
                <th>用户</th>
                <th>操作</th>
                <th>详情</th>
                <th>IP地址</th>
              </tr>
            </thead>
            <tbody>
              {auditLogs.map((log, index) => (
                <tr key={log.id || index}>
                  <td className="audit-time">
                    {new Date(log.created_at).toLocaleString('zh-CN')}
                  </td>
                  <td className="audit-user">
                    {log.user_id || <span className="system-badge">系统</span>}
                  </td>
                  <td className="audit-action">
                    {getActionBadge(log.action)}
                    <span className="action-text">{log.action}</span>
                  </td>
                  <td className="audit-details">
                    {log.details ? (
                      <details>
                        <summary>查看详情</summary>
                        <pre>{JSON.stringify(log.details, null, 2)}</pre>
                      </details>
                    ) : '-'}
                  </td>
                  <td className="audit-ip">{log.ip || '-'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {!loading && auditLogs.length > 0 && (
          <Pagination
            current={currentPage}
            total={totalLogs}
            pageSize={pageSize}
            onChange={setCurrentPage}
          />
        )}

        {!loading && auditLogs.length === 0 && (
          <div className="empty-state">
            <p>暂无审计数据</p>
          </div>
        )}
      </div>
    </div>
  );
};

export default AuditDashboard;
