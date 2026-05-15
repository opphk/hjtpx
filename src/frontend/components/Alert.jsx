import React, { useEffect } from 'react';

function Alert({
  type = 'info',
  message,
  onClose,
  autoClose = true,
  autoCloseTime = 3000,
  className = '',
  role,
  'aria-label': ariaLabel,
  'aria-live': ariaLive,
  ...props
}) {
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
  
  const defaultAriaLive = type === 'error' || type === 'warning' ? 'assertive' : 'polite';
  const defaultRole = type === 'error' || type === 'warning' ? 'alert' : 'status';

  return (
    <div 
      className={alertClass} 
      role={role || defaultRole}
      aria-label={ariaLabel}
      aria-live={ariaLive || defaultAriaLive}
      aria-atomic="true"
      {...props}
    >
      <div className="alert-content">
        <span className="alert-message">{message}</span>
        {onClose && (
          <button
            type="button"
            className="alert-close"
            onClick={onClose}
            aria-label="关闭提示"
          >
            &times;
          </button>
        )}
      </div>
    </div>
  );
}

export default React.memo(Alert);
