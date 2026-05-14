import React, { useState, useEffect } from 'react';
import Button from './ui/Button';
import Loading from './ui/Loading';

const SystemConfig = ({ onSuccess, onError }) => {
  const [config, setConfig] = useState({
    site_name: 'HJTPX',
    site_url: 'http://localhost:3000',
    maintenance_mode: false,
    max_users: 1000,
    session_timeout: 3600,
    api_rate_limit: 100,
    log_level: 'info',
    log_retention_days: 30,
    cache_enabled: true,
    cache_ttl: 300,
    upload_max_size: 10485760,
    allowed_file_types: '.jpg,.png,.pdf,.doc,.docx',
    email_verification_required: true,
    password_min_length: 6
  });
  const [originalConfig, setOriginalConfig] = useState(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [errors, setErrors] = useState({});
  const [hasChanges, setHasChanges] = useState(false);

  useEffect(() => {
    fetchConfig();
  }, []);

  const fetchConfig = async () => {
    setLoading(true);
    try {
      const token = localStorage.getItem('authToken');
      const response = await fetch('/api/v1/admin/settings/system', {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });

      if (response.ok) {
        const data = await response.json();
        setConfig(data.config || config);
        setOriginalConfig(data.config || config);
      }
    } catch (err) {
      console.error('Failed to fetch config:', err);
    } finally {
      setLoading(false);
    }
  };

  const validate = () => {
    const newErrors = {};

    if (!config.site_name.trim()) {
      newErrors.site_name = '网站名称不能为空';
    }

    if (!config.site_url.trim()) {
      newErrors.site_url = '网站URL不能为空';
    } else {
      try {
        new URL(config.site_url);
      } catch {
        newErrors.site_url = '请输入有效的URL地址';
      }
    }

    if (config.max_users < 1) {
      newErrors.max_users = '最大用户数必须大于0';
    }

    if (config.session_timeout < 60) {
      newErrors.session_timeout = '会话超时时间至少60秒';
    }

    if (config.password_min_length < 6) {
      newErrors.password_min_length = '密码最小长度不能小于6';
    }

    if (config.api_rate_limit < 1) {
      newErrors.api_rate_limit = 'API速率限制必须大于0';
    }

    if (config.log_retention_days < 1) {
      newErrors.log_retention_days = '日志保留天数必须大于0';
    }

    if (config.cache_ttl < 60) {
      newErrors.cache_ttl = '缓存TTL至少60秒';
    }

    if (config.upload_max_size < 1024) {
      newErrors.upload_max_size = '上传文件大小至少1KB';
    }

    if (!config.allowed_file_types.trim()) {
      newErrors.allowed_file_types = '请输入允许的文件类型';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleChange = (key, value) => {
    setConfig(prev => ({
      ...prev,
      [key]: value
    }));

    if (errors[key]) {
      setErrors(prev => ({
        ...prev,
        [key]: ''
      }));
    }

    if (originalConfig) {
      setHasChanges(JSON.stringify(config) !== JSON.stringify(originalConfig));
    }
  };

  const handleSave = async () => {
    if (!validate()) {
      onError('请修正表单错误后再保存');
      return;
    }

    setSaving(true);
    try {
      const token = localStorage.getItem('authToken');
      const response = await fetch('/api/v1/admin/settings/system', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify(config)
      });

      if (response.ok) {
        setOriginalConfig(config);
        setHasChanges(false);
        onSuccess('系统配置已保存');
      } else {
        const errorData = await response.json();
        onError(errorData.error || '保存失败');
      }
    } catch (err) {
      onError('网络错误，请稍后重试');
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return <Loading text="加载配置..." />;
  }

  return (
    <div className="config-section">
      {hasChanges && (
        <div className="config-changes-indicator">
          <span>配置已修改，但尚未保存</span>
        </div>
      )}

      <div className="config-card">
        <h3>基本信息</h3>
        <div className="config-form">
          <div className="form-group">
            <label className="form-label">网站名称</label>
            <input
              type="text"
              value={config.site_name}
              onChange={(e) => handleChange('site_name', e.target.value)}
              className={`form-input ${errors.site_name ? 'input-error' : ''}`}
            />
            {errors.site_name && <span className="error-text">{errors.site_name}</span>}
          </div>
          <div className="form-group">
            <label className="form-label">网站URL</label>
            <input
              type="url"
              value={config.site_url}
              onChange={(e) => handleChange('site_url', e.target.value)}
              className={`form-input ${errors.site_url ? 'input-error' : ''}`}
            />
            {errors.site_url && <span className="error-text">{errors.site_url}</span>}
          </div>
          <div className="config-item">
            <div className="config-item-info">
              <span className="config-item-label">维护模式</span>
              <span className="config-item-desc">启用后普通用户无法访问</span>
            </div>
            <button
              className={`toggle-switch ${config.maintenance_mode ? 'active' : ''}`}
              onClick={() => handleChange('maintenance_mode', !config.maintenance_mode)}
            >
              <span className="toggle-slider"></span>
            </button>
          </div>
        </div>
      </div>

      <div className="config-card">
        <h3>用户设置</h3>
        <div className="config-form">
          <div className="form-group">
            <label className="form-label">最大用户数</label>
            <input
              type="number"
              value={config.max_users}
              onChange={(e) => handleChange('max_users', parseInt(e.target.value))}
              className={`form-input ${errors.max_users ? 'input-error' : ''}`}
              min="1"
            />
            {errors.max_users && <span className="error-text">{errors.max_users}</span>}
          </div>
          <div className="form-group">
            <label className="form-label">会话超时 (秒)</label>
            <input
              type="number"
              value={config.session_timeout}
              onChange={(e) => handleChange('session_timeout', parseInt(e.target.value))}
              className={`form-input ${errors.session_timeout ? 'input-error' : ''}`}
              min="60"
            />
            {errors.session_timeout && <span className="error-text">{errors.session_timeout}</span>}
          </div>
          <div className="form-group">
            <label className="form-label">密码最小长度</label>
            <input
              type="number"
              value={config.password_min_length}
              onChange={(e) => handleChange('password_min_length', parseInt(e.target.value))}
              className={`form-input ${errors.password_min_length ? 'input-error' : ''}`}
              min="6"
            />
            {errors.password_min_length && <span className="error-text">{errors.password_min_length}</span>}
          </div>
          <div className="config-item">
            <div className="config-item-info">
              <span className="config-item-label">邮箱验证</span>
              <span className="config-item-desc">注册时需要邮箱验证</span>
            </div>
            <button
              className={`toggle-switch ${config.email_verification_required ? 'active' : ''}`}
              onClick={() => handleChange('email_verification_required', !config.email_verification_required)}
            >
              <span className="toggle-slider"></span>
            </button>
          </div>
        </div>
      </div>

      <div className="config-card">
        <h3>API 设置</h3>
        <div className="config-form">
          <div className="form-group">
            <label className="form-label">API 速率限制 (请求/分钟)</label>
            <input
              type="number"
              value={config.api_rate_limit}
              onChange={(e) => handleChange('api_rate_limit', parseInt(e.target.value))}
              className={`form-input ${errors.api_rate_limit ? 'input-error' : ''}`}
              min="1"
            />
            {errors.api_rate_limit && <span className="error-text">{errors.api_rate_limit}</span>}
          </div>
        </div>
      </div>

      <div className="config-card">
        <h3>日志设置</h3>
        <div className="config-form">
          <div className="form-group">
            <label className="form-label">日志级别</label>
            <select
              value={config.log_level}
              onChange={(e) => handleChange('log_level', e.target.value)}
              className="form-select"
            >
              <option value="error">Error</option>
              <option value="warn">Warn</option>
              <option value="info">Info</option>
              <option value="debug">Debug</option>
            </select>
          </div>
          <div className="form-group">
            <label className="form-label">日志保留天数</label>
            <input
              type="number"
              value={config.log_retention_days}
              onChange={(e) => handleChange('log_retention_days', parseInt(e.target.value))}
              className={`form-input ${errors.log_retention_days ? 'input-error' : ''}`}
              min="1"
            />
            {errors.log_retention_days && <span className="error-text">{errors.log_retention_days}</span>}
          </div>
        </div>
      </div>

      <div className="config-card">
        <h3>缓存设置</h3>
        <div className="config-form">
          <div className="config-item">
            <div className="config-item-info">
              <span className="config-item-label">启用缓存</span>
              <span className="config-item-desc">启用系统缓存提高性能</span>
            </div>
            <button
              className={`toggle-switch ${config.cache_enabled ? 'active' : ''}`}
              onClick={() => handleChange('cache_enabled', !config.cache_enabled)}
            >
              <span className="toggle-slider"></span>
            </button>
          </div>
          <div className="form-group">
            <label className="form-label">缓存 TTL (秒)</label>
            <input
              type="number"
              value={config.cache_ttl}
              onChange={(e) => handleChange('cache_ttl', parseInt(e.target.value))}
              className={`form-input ${errors.cache_ttl ? 'input-error' : ''}`}
              min="60"
              disabled={!config.cache_enabled}
            />
            {errors.cache_ttl && <span className="error-text">{errors.cache_ttl}</span>}
          </div>
        </div>
      </div>

      <div className="config-card">
        <h3>文件上传设置</h3>
        <div className="config-form">
          <div className="form-group">
            <label className="form-label">最大文件大小 (字节)</label>
            <input
              type="number"
              value={config.upload_max_size}
              onChange={(e) => handleChange('upload_max_size', parseInt(e.target.value))}
              className={`form-input ${errors.upload_max_size ? 'input-error' : ''}`}
              min="1024"
            />
            {errors.upload_max_size && <span className="error-text">{errors.upload_max_size}</span>}
          </div>
          <div className="form-group">
            <label className="form-label">允许的文件类型</label>
            <input
              type="text"
              value={config.allowed_file_types}
              onChange={(e) => handleChange('allowed_file_types', e.target.value)}
              className={`form-input ${errors.allowed_file_types ? 'input-error' : ''}`}
              placeholder=".jpg,.png,.pdf"
            />
            {errors.allowed_file_types && <span className="error-text">{errors.allowed_file_types}</span>}
          </div>
        </div>
      </div>

      <div className="config-actions">
        <Button
          variant="primary"
          onClick={handleSave}
          loading={saving}
          disabled={!hasChanges}
        >
          保存配置
        </Button>
        {hasChanges && (
          <Button
            variant="secondary"
            onClick={() => {
              if (originalConfig) {
                setConfig(originalConfig);
                setHasChanges(false);
                setErrors({});
              }
            }}
          >
            放弃更改
          </Button>
        )}
      </div>
    </div>
  );
};

export default SystemConfig;
