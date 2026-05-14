import React, { useEffect } from 'react';

function Alert({
  type = 'info',
  message,
  onClose,
  autoClose = true,
  autoCloseTime = 3000,
  className = '',
  role = 'alert',
  politeness = 'assertive',
  title
}) {
  const alertId = `alert-${Math.random().toString(36).substr(2, 9)}`;
  
  useEffect(() => {
    if (autoClose && autoCloseTime > 0) {
      const timer = setTimeout(() => {
        if (onClose) {
          onClose();
        }
      }, autoCloseTime);

      return () => clearTimeout(timer);
    }
  }, [autoClose, autoCloseTime, onClose]);

  const alertClass = `alert alert-${type} ${className}`.trim();
  const alertTitleId = title ? `${alertId}-title` : undefined;

  const getAlertRole = () => {
    switch (type) {
      case 'error':
      case 'danger':
        return 'alert';
      case 'success':
        return 'status';
      case 'warning':
        return 'alert';
      default:
        return 'status';
    }
  };

  return (
    <div 
      className={alertClass} 
      role={role || getAlertRole()}
      aria-live={politeness}
      aria-atomic="true"
      id={alertId}
    >
      <div className="alert-content">
        {title && (
          <strong id={alertTitleId} className="alert-title">
            {title}
          </strong>
        )}
        <span className="alert-message" aria-labelledby={alertTitleId}>
          {message}
        </span>
        {onClose && (
          <button
            type="button"
            className="alert-close"
            onClick={onClose}
            aria-label="Dismiss alert"
            aria-describedby={alertId}
          >
            &times;
          </button>
        )}
      </div>
    </div>
  );
}

export default React.memo(Alert);
