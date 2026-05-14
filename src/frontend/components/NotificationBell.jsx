import React, { useState, useEffect, useCallback } from 'react';
import PropTypes from 'prop-types';
import axios from 'axios';
import socketService from '../../services/socketService';

const NotificationBell = ({ userId, maxVisible = 5, onNotificationClick }) => {
  const [notifications, setNotifications] = useState([]);
  const [unreadCount, setUnreadCount] = useState(0);
  const [isOpen, setIsOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const fetchNotifications = useCallback(async () => {
    if (!userId) return;

    setLoading(true);
    setError(null);

    try {
      const response = await axios.get(`/api/v1/notifications?limit=${maxVisible}`, {
        headers: {
          Authorization: `Bearer ${localStorage.getItem('token')}`
        }
      });

      if (response.data.success) {
        setNotifications(response.data.data);
      }
    } catch (err) {
      setError('Failed to load notifications');
      console.error('Error fetching notifications:', err);
    } finally {
      setLoading(false);
    }
  }, [userId, maxVisible]);

  const fetchUnreadCount = useCallback(async () => {
    if (!userId) return;

    try {
      const response = await axios.get('/api/v1/notifications/unread-count', {
        headers: {
          Authorization: `Bearer ${localStorage.getItem('token')}`
        }
      });

      if (response.data.success) {
        setUnreadCount(response.data.data.count);
      }
    } catch (err) {
      console.error('Error fetching unread count:', err);
    }
  }, [userId]);

  useEffect(() => {
    fetchUnreadCount();
    const interval = setInterval(fetchUnreadCount, 30000);
    return () => clearInterval(interval);
  }, [fetchUnreadCount]);

  useEffect(() => {
    if (isOpen) {
      fetchNotifications();
    }
  }, [isOpen, fetchNotifications]);

  useEffect(() => {
    const token = localStorage.getItem('token');
    if (!token) return;

    socketService.connect(token);

    socketService.onNotification((notification) => {
      setNotifications((prev) => [notification, ...prev]);
      setUnreadCount((prev) => prev + 1);

      if (onNotificationClick) {
        onNotificationClick(notification);
      }
    });

    socketService.onPresenceUpdate((update) => {
      console.log('Presence update received:', update);
    });

    return () => {
    };
  }, [onNotificationClick]);

  const handleMarkAsRead = async (notificationId) => {
    try {
      await axios.put(
        `/api/v1/notifications/${notificationId}/read`,
        {},
        {
          headers: {
            Authorization: `Bearer ${localStorage.getItem('token')}`
          }
        }
      );

      setNotifications((prev) =>
        prev.map((n) =>
          n.id === notificationId ? { ...n, status: 'read' } : n
        )
      );
      setUnreadCount((prev) => Math.max(0, prev - 1));

      if (onNotificationClick) {
        const notification = notifications.find((n) => n.id === notificationId);
        onNotificationClick(notification);
      }
    } catch (err) {
      console.error('Error marking notification as read:', err);
    }
  };

  const handleMarkAllAsRead = async () => {
    try {
      await axios.put(
        '/api/v1/notifications/read-all',
        {},
        {
          headers: {
            Authorization: `Bearer ${localStorage.getItem('token')}`
          }
        }
      );

      setNotifications((prev) =>
        prev.map((n) => ({ ...n, status: 'read' }))
      );
      setUnreadCount(0);
    } catch (err) {
      console.error('Error marking all as read:', err);
    }
  };

  const getNotificationIcon = (type) => {
    switch (type) {
      case 'success':
        return '✓';
      case 'warning':
        return '⚠';
      case 'error':
        return '✕';
      case 'info':
        return 'ℹ';
      case 'system':
        return '⚙';
      case 'message':
        return '✉';
      case 'reminder':
        return '⏰';
      case 'alert':
        return '🚨';
      default:
        return '•';
    }
  };

  const getPriorityClass = (priority) => {
    switch (priority) {
      case 'urgent':
        return 'notification-urgent';
      case 'high':
        return 'notification-high';
      case 'low':
        return 'notification-low';
      default:
        return 'notification-normal';
    }
  };

  const formatTime = (timestamp) => {
    const date = new Date(timestamp);
    const now = new Date();
    const diff = now - date;

    if (diff < 60000) return 'Just now';
    if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
    if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
    return date.toLocaleDateString();
  };

  return (
    <div className="notification-bell">
      <button
        className="notification-bell-button"
        onClick={() => setIsOpen(!isOpen)}
        aria-label={`Notifications ${unreadCount > 0 ? `(${unreadCount} unread)` : ''}`}
      >
        <span className="notification-icon">🔔</span>
        {unreadCount > 0 && (
          <span className="notification-badge">
            {unreadCount > 99 ? '99+' : unreadCount}
          </span>
        )}
      </button>

      {isOpen && (
        <div className="notification-dropdown">
          <div className="notification-header">
            <h3>Notifications</h3>
            {unreadCount > 0 && (
              <button
                className="mark-all-read"
                onClick={handleMarkAllAsRead}
              >
                Mark all as read
              </button>
            )}
          </div>

          <div className="notification-list">
            {loading && <div className="notification-loading">Loading...</div>}

            {error && <div className="notification-error">{error}</div>}

            {!loading && !error && notifications.length === 0 && (
              <div className="notification-empty">No notifications</div>
            )}

            {!loading && !error && notifications.map((notification) => (
              <div
                key={notification.id}
                className={`notification-item ${getPriorityClass(notification.priority)} ${
                  notification.status === 'unread' ? 'unread' : ''
                }`}
                onClick={() => handleMarkAsRead(notification.id)}
              >
                <div className="notification-item-icon">
                  {getNotificationIcon(notification.type)}
                </div>
                <div className="notification-item-content">
                  <div className="notification-item-title">
                    {notification.title}
                  </div>
                  <div className="notification-item-message">
                    {notification.message}
                  </div>
                  <div className="notification-item-time">
                    {formatTime(notification.created_at)}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

NotificationBell.propTypes = {
  userId: PropTypes.oneOfType([PropTypes.string, PropTypes.number]).isRequired,
  maxVisible: PropTypes.number,
  onNotificationClick: PropTypes.func
};

NotificationBell.defaultProps = {
  maxVisible: 5,
  onNotificationClick: null
};

export default NotificationBell;
