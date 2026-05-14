import React from 'react';
import { useTranslation } from 'react-i18next';
import { format } from 'date-fns';

const NotificationBell = ({ unreadCount = 0, onClick }) => {
  const { t } = useTranslation();

  return (
    <div className="notification-bell" onClick={onClick}>
      <span className="bell-icon">🔔</span>
      {unreadCount > 0 && (
        <span className="badge">{unreadCount > 99 ? '99+' : unreadCount}</span>
      )}
    </div>
  );
};

const NotificationItem = ({ notification, onMarkRead, onDelete }) => {
  const { t } = useTranslation();

  const getIcon = () => {
    switch (notification.type) {
      case 'system': return '⚙️';
      case 'user': return '👤';
      case 'security': return '🔒';
      case 'promotion': return '🎉';
      default: return '📢';
    }
  };

  const formatDate = (dateString) => {
    try {
      return format(new Date(dateString), 'MMM dd, yyyy HH:mm');
    } catch {
      return dateString;
    }
  };

  return (
    <div
      className={`notification-item ${notification.read ? 'read' : 'unread'} ${notification.type}`}
    >
      <div className="notification-icon">{getIcon()}</div>
      <div className="notification-content">
        <h4>{notification.title}</h4>
        <p>{notification.message}</p>
        <span className="notification-time">{formatDate(notification.createdAt)}</span>
      </div>
      <div className="notification-actions">
        {!notification.read && (
          <button onClick={() => onMarkRead(notification.id)} className="btn-mark-read">
            ✓
          </button>
        )}
        <button onClick={() => onDelete(notification.id)} className="btn-delete">
          ×
        </button>
      </div>
    </div>
  );
};

const NotificationList = ({ notifications = [], onMarkRead, onDelete, onLoadMore }) => {
  const { t } = useTranslation();

  if (notifications.length === 0) {
    return (
      <div className="notification-list empty">
        <p>{t('notifications.noNotifications')}</p>
      </div>
    );
  }

  return (
    <div className="notification-list">
      {notifications.map((notification) => (
        <NotificationItem
          key={notification.id}
          notification={notification}
          onMarkRead={onMarkRead}
          onDelete={onDelete}
        />
      ))}
      {onLoadMore && (
        <button onClick={onLoadMore} className="load-more">
          {t('common.loading')}
        </button>
      )}
    </div>
  );
};

const NotificationSettings = ({ settings, onChange }) => {
  const { t } = useTranslation();

  const channels = [
    { key: 'email', label: t('notifications.settings.email'), icon: '📧' },
    { key: 'sms', label: t('notifications.settings.sms'), icon: '📱' },
    { key: 'push', label: t('notifications.settings.push'), icon: '🔔' },
    { key: 'inApp', label: t('notifications.settings.inApp'), icon: '💬' }
  ];

  const notificationTypes = [
    { key: 'system', label: t('notifications.types.system') },
    { key: 'user', label: t('notifications.types.user') },
    { key: 'security', label: t('notifications.types.security') },
    { key: 'promotion', label: t('notifications.types.promotion') }
  ];

  const handleChannelToggle = (channel) => {
    const updated = {
      ...settings,
      channels: {
        ...settings.channels,
        [channel]: !settings.channels[channel]
      }
    };
    onChange(updated);
  };

  const handleTypeToggle = (type) => {
    const updated = {
      ...settings,
      types: {
        ...settings.types,
        [type]: !settings.types[type]
      }
    };
    onChange(updated);
  };

  return (
    <div className="notification-settings">
      <h3>{t('notifications.settings.title')}</h3>

      <div className="settings-section">
        <h4>通知渠道</h4>
        {channels.map((channel) => (
          <label key={channel.key} className="setting-item">
            <input
              type="checkbox"
              checked={settings.channels?.[channel.key] ?? true}
              onChange={() => handleChannelToggle(channel.key)}
            />
            <span className="channel-icon">{channel.icon}</span>
            <span>{channel.label}</span>
          </label>
        ))}
      </div>

      <div className="settings-section">
        <h4>通知类型</h4>
        {notificationTypes.map((type) => (
          <label key={type.key} className="setting-item">
            <input
              type="checkbox"
              checked={settings.types?.[type.key] ?? true}
              onChange={() => handleTypeToggle(type.key)}
            />
            <span>{type.label}</span>
          </label>
        ))}
      </div>
    </div>
  );
};

const NotificationCenter = ({ isOpen, onClose, notifications, onMarkRead, onMarkAllRead, onDelete, settings, onSettingsChange }) => {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = React.useState('notifications');

  if (!isOpen) return null;

  const unreadCount = notifications.filter(n => !n.read).length;

  return (
    <div className="notification-center-overlay" onClick={onClose}>
      <div className="notification-center" onClick={(e) => e.stopPropagation()}>
        <div className="center-header">
          <h2>{t('notifications.title')}</h2>
          <button onClick={onClose} className="close-button">×</button>
        </div>

        <div className="center-tabs">
          <button
            className={activeTab === 'notifications' ? 'active' : ''}
            onClick={() => setActiveTab('notifications')}
          >
            {t('notifications.title')}
            {unreadCount > 0 && <span className="tab-badge">{unreadCount}</span>}
          </button>
          <button
            className={activeTab === 'settings' ? 'active' : ''}
            onClick={() => setActiveTab('settings')}
          >
            {t('notifications.settings.title')}
          </button>
        </div>

        <div className="center-content">
          {activeTab === 'notifications' && (
            <>
              {unreadCount > 0 && (
                <div className="mark-all-container">
                  <button onClick={onMarkAllRead} className="mark-all-btn">
                    {t('notifications.markAllRead')}
                  </button>
                </div>
              )}
              <NotificationList
                notifications={notifications}
                onMarkRead={onMarkRead}
                onDelete={onDelete}
              />
            </>
          )}

          {activeTab === 'settings' && (
            <NotificationSettings
              settings={settings}
              onChange={onSettingsChange}
            />
          )}
        </div>
      </div>
    </div>
  );
};

export {
  NotificationBell,
  NotificationItem,
  NotificationList,
  NotificationSettings,
  NotificationCenter
};
export default NotificationBell;
