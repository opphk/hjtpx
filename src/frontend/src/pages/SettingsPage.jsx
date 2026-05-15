import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import SEO, { getSEOMeta } from '../components/SEO';
import SystemConfig from '../components/SystemConfig';
import FeatureFlags from '../components/FeatureFlags';
import Alert from '../components/ui/Alert';

const SettingsPage = () => {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState('system');
  const [success, setSuccess] = useState('');
  const [error, setError] = useState('');
  const seoMeta = getSEOMeta('settings');

  const handleSuccess = (message) => {
    setSuccess(message);
    setError('');
  };

  const handleError = (message) => {
    setError(message);
    setSuccess('');
  };

  const renderContent = () => {
    switch (activeTab) {
      case 'system':
        return (
          <SystemConfig
            onSuccess={handleSuccess}
            onError={handleError}
          />
        );
      case 'features':
        return (
          <FeatureFlags
            onSuccess={handleSuccess}
            onError={handleError}
          />
        );
      case 'notifications':
        return (
          <NotificationsConfig
            onSuccess={handleSuccess}
            onError={handleError}
          />
        );
      default:
        return null;
    }
  };

  return (
    <>
      <SEO {...seoMeta} />
      <div className="settings-page">
      <div className="page-header">
        <div>
          <h1>{t('settings.title')}</h1>
          <p>{t('settings.description')}</p>
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

      {success && (
        <Alert
          type="success"
          message={success}
          closable
          onClose={() => setSuccess('')}
        />
      )}

      <div className="settings-tabs">
        <button
          className={`tab-button ${activeTab === 'system' ? 'active' : ''}`}
          onClick={() => setActiveTab('system')}
        >
          {t('settings.systemConfig')}
        </button>
        <button
          className={`tab-button ${activeTab === 'features' ? 'active' : ''}`}
          onClick={() => setActiveTab('features')}
        >
          {t('settings.features')}
        </button>
        <button
          className={`tab-button ${activeTab === 'notifications' ? 'active' : ''}`}
          onClick={() => setActiveTab('notifications')}
        >
          {t('settings.notifications')}
        </button>
      </div>

      <div className="settings-content">
        {renderContent()}
      </div>
    </div>
    </>
  );
};

const NotificationsConfig = ({ onSuccess, onError }) => {
  const { t } = useTranslation();
  const [config, setConfig] = useState({
    email_notifications: true,
    push_notifications: true,
    in_app_notifications: true,
    notification_frequency: 'realtime',
    email_for_important: true,
    daily_digest: false,
    weekly_report: false
  });
  const [loading, setLoading] = useState(false);

  const handleToggle = (key) => {
    setConfig(prev => ({
      ...prev,
      [key]: !prev[key]
    }));
  };

  const handleSelectChange = (key, value) => {
    setConfig(prev => ({
      ...prev,
      [key]: value
    }));
  };

  const handleSave = async () => {
    setLoading(true);
    try {
      const token = localStorage.getItem('authToken');
      const response = await fetch('/api/v1/admin/settings/notifications', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify(config)
      });

      if (response.ok) {
        onSuccess(t('settings.saveSuccess'));
      } else {
        const errorData = await response.json();
        onError(errorData.error || t('settings.saveFailed'));
      }
    } catch (err) {
      onError(t('users.networkError'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="config-section">
      <div className="config-card">
        <h3>{t('settings.notificationChannels')}</h3>
        <div className="config-item">
          <div className="config-item-info">
            <span className="config-item-label">{t('settings.emailNotifications')}</span>
            <span className="config-item-desc">{t('users.description')}</span>
          </div>
          <button
            className={`toggle-switch ${config.email_notifications ? 'active' : ''}`}
            onClick={() => handleToggle('email_notifications')}
          >
            <span className="toggle-slider"></span>
          </button>
        </div>
        <div className="config-item">
          <div className="config-item-info">
            <span className="config-item-label">{t('settings.pushNotifications')}</span>
            <span className="config-item-desc">{t('logs.description')}</span>
          </div>
          <button
            className={`toggle-switch ${config.push_notifications ? 'active' : ''}`}
            onClick={() => handleToggle('push_notifications')}
          >
            <span className="toggle-slider"></span>
          </button>
        </div>
        <div className="config-item">
          <div className="config-item-info">
            <span className="config-item-label">{t('settings.inAppNotifications')}</span>
            <span className="config-item-desc">{t('audit.description')}</span>
          </div>
          <button
            className={`toggle-switch ${config.in_app_notifications ? 'active' : ''}`}
            onClick={() => handleToggle('in_app_notifications')}
          >
            <span className="toggle-slider"></span>
          </button>
        </div>
      </div>

      <div className="config-card">
        <h3>{t('settings.notificationSettings')}</h3>
        <div className="config-item">
          <div className="config-item-info">
            <span className="config-item-label">{t('settings.notificationFrequency')}</span>
            <span className="config-item-desc">{t('common.apply')}</span>
          </div>
          <select
            value={config.notification_frequency}
            onChange={(e) => handleSelectChange('notification_frequency', e.target.value)}
            className="form-select"
          >
            <option value="realtime">{t('settings.realtime')}</option>
            <option value="hourly">{t('settings.hourly')}</option>
            <option value="daily">{t('settings.daily')}</option>
          </select>
        </div>
        <div className="config-item">
          <div className="config-item-info">
            <span className="config-item-label">{t('settings.importantEmail')}</span>
            <span className="config-item-desc">{t('common.confirm')}</span>
          </div>
          <button
            className={`toggle-switch ${config.email_for_important ? 'active' : ''}`}
            onClick={() => handleToggle('email_for_important')}
          >
            <span className="toggle-slider"></span>
          </button>
        </div>
      </div>

      <div className="config-card">
        <h3>{t('settings.reportSettings')}</h3>
        <div className="config-item">
          <div className="config-item-info">
            <span className="config-item-label">{t('settings.dailyDigest')}</span>
            <span className="config-item-desc">{t('common.save')}</span>
          </div>
          <button
            className={`toggle-switch ${config.daily_digest ? 'active' : ''}`}
            onClick={() => handleToggle('daily_digest')}
          >
            <span className="toggle-slider"></span>
          </button>
        </div>
        <div className="config-item">
          <div className="config-item-info">
            <span className="config-item-label">{t('settings.weeklyReport')}</span>
            <span className="config-item-desc">{t('common.cancel')}</span>
          </div>
          <button
            className={`toggle-switch ${config.weekly_report ? 'active' : ''}`}
            onClick={() => handleToggle('weekly_report')}
          >
            <span className="toggle-slider"></span>
          </button>
        </div>
      </div>

      <div className="config-actions">
        <button
          className="btn btn-primary"
          onClick={handleSave}
          disabled={loading}
        >
          {loading ? t('common.loading') : t('settings.saveConfig')}
        </button>
      </div>
    </div>
  );
};

export default SettingsPage;
