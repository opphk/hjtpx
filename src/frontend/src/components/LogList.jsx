import React from 'react';
import Loading from './ui/Loading';

const LogList = ({ logs, loading, onViewDetails, selectedLog }) => {
  const tableId = React.useId();
  const captionId = `${tableId}-caption`;

  const getLevelBadge = (level) => {
    const levelMap = {
      error: { label: '错误', className: 'level-error' },
      warn: { label: '警告', className: 'level-warn' },
      info: { label: '信息', className: 'level-info' },
      debug: { label: '调试', className: 'level-debug' }
    };
    const levelInfo = levelMap[level] || levelMap.info;
    return <span className={`log-level-badge ${levelInfo.className}`}>{levelInfo.label}</span>;
  };

  const getTypeBadge = (type) => {
    const typeMap = {
      operation: { label: '操作', className: 'type-operation' },
      error: { label: '错误', className: 'type-error' },
      security: { label: '安全', className: 'type-security' },
      system: { label: '系统', className: 'type-system' }
    };
    const typeInfo = typeMap[type] || typeMap.operation;
    return <span className={`log-type-badge ${typeInfo.className}`}>{typeInfo.label}</span>;
  };

  const handleRowKeyDown = (e, log) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      onViewDetails(log);
    }
  };

  if (loading && logs.length === 0) {
    return (
      <Loading 
        text="加载日志..." 
        aria-label="正在加载日志列表"
      />
    );
  }

  if (!loading && logs.length === 0) {
    return (
      <div 
        className="empty-state"
        role="status"
        aria-live="polite"
        aria-label="暂无日志数据"
      >
        <p>暂无日志数据</p>
      </div>
    );
  }

  return (
    <div className="log-list-container" role="region" aria-label="日志列表">
      <table 
        className="log-table"
        role="grid"
        aria-label="日志数据表"
      >
        <caption id={captionId} className="sr-only">
          系统日志列表，包含时间、级别、类型、用户、操作、IP地址和详情
        </caption>
        <thead role="rowgroup">
          <tr role="row">
            <th role="columnheader" scope="col">时间</th>
            <th role="columnheader" scope="col">级别</th>
            <th role="columnheader" scope="col">类型</th>
            <th role="columnheader" scope="col">用户</th>
            <th role="columnheader" scope="col">操作</th>
            <th role="columnheader" scope="col">IP地址</th>
            <th role="columnheader" scope="col">
              <span className="sr-only">操作</span>
            </th>
          </tr>
        </thead>
        <tbody role="rowgroup">
          {logs.map((log, index) => (
            <tr 
              key={log.id || index}
              role="row"
              className={selectedLog?.id === log.id ? 'selected' : ''}
              onClick={() => onViewDetails(log)}
              onKeyDown={(e) => handleRowKeyDown(e, log)}
              tabIndex={0}
              aria-selected={selectedLog?.id === log.id}
              aria-label={`日志条目 ${index + 1}，${log.action || '操作'}`}
            >
              <td className="log-time" role="gridcell">
                {new Date(log.timestamp).toLocaleString('zh-CN', {
                  year: 'numeric',
                  month: '2-digit',
                  day: '2-digit',
                  hour: '2-digit',
                  minute: '2-digit'
                })}
              </td>
              <td role="gridcell">{getLevelBadge(log.level)}</td>
              <td role="gridcell">{getTypeBadge(log.type)}</td>
              <td className="log-user" role="gridcell">
                {log.user_id ? (
                  <span className="user-badge">{log.user_id}</span>
                ) : (
                  <span className="system-badge">系统</span>
                )}
              </td>
              <td className="log-action" role="gridcell">{log.action}</td>
              <td className="log-ip" role="gridcell">{log.ip || '-'}</td>
              <td className="log-details-btn" role="gridcell">
                <button 
                  className="btn btn-small btn-secondary" 
                  onClick={(e) => {
                    e.stopPropagation();
                    onViewDetails(log);
                  }}
                  aria-label={`查看日志详情 ${log.action || '操作'}`}
                >
                  查看
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

export default LogList;
