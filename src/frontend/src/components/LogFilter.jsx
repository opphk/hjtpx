import React, { useState } from 'react';

const LogFilter = ({ filters, onFilterChange }) => {
  const [localFilters, setLocalFilters] = useState(filters);

  const handleChange = (key, value) => {
    setLocalFilters(prev => ({
      ...prev,
      [key]: value
    }));
  };

  const handleApply = () => {
    onFilterChange(localFilters);
  };

  const handleReset = () => {
    const resetFilters = {
      type: '',
      level: '',
      startDate: '',
      endDate: '',
      search: ''
    };
    setLocalFilters(resetFilters);
    onFilterChange(resetFilters);
  };

  return (
    <div className="log-filter">
      <div className="filter-row">
        <div className="filter-item">
          <input
            type="text"
            placeholder="搜索日志内容..."
            value={localFilters.search}
            onChange={(e) => handleChange('search', e.target.value)}
            className="search-input"
          />
        </div>

        <div className="filter-item">
          <label htmlFor="log-type-filter" className="filter-label">日志类型</label>
          <select
            id="log-type-filter"
            value={localFilters.type}
            onChange={(e) => handleChange('type', e.target.value)}
            className="form-select"
            aria-label="按日志类型筛选"
          >
            <option value="">所有类型</option>
            <option value="operation">操作</option>
            <option value="error">错误</option>
            <option value="security">安全</option>
            <option value="system">系统</option>
          </select>
        </div>

        <div className="filter-item">
          <label htmlFor="log-level-filter" className="filter-label">日志级别</label>
          <select
            id="log-level-filter"
            value={localFilters.level}
            onChange={(e) => handleChange('level', e.target.value)}
            className="form-select"
            aria-label="按日志级别筛选"
          >
            <option value="">所有级别</option>
            <option value="error">错误</option>
            <option value="warn">警告</option>
            <option value="info">信息</option>
            <option value="debug">调试</option>
          </select>
        </div>
      </div>

      <div className="filter-row">
        <div className="filter-item">
          <label className="filter-label">开始日期</label>
          <input
            type="date"
            value={localFilters.startDate}
            onChange={(e) => handleChange('startDate', e.target.value)}
            className="date-input"
          />
        </div>

        <div className="filter-item">
          <label className="filter-label">结束日期</label>
          <input
            type="date"
            value={localFilters.endDate}
            onChange={(e) => handleChange('endDate', e.target.value)}
            className="date-input"
          />
        </div>

        <div className="filter-actions">
          <button className="btn btn-primary" onClick={handleApply}>
            应用筛选
          </button>
          <button className="btn btn-secondary" onClick={handleReset}>
            重置
          </button>
        </div>
      </div>
    </div>
  );
};

export default LogFilter;
