import React, { useEffect } from 'react';

function Alert({
  type = 'info',
  message,
  onClose,
  autoClose = true,
  autoCloseTime = 3000,
  className = ''
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

  return (
    <div className={alertClass} role="alert">
      <div className="alert-content">
        <span className="alert-message">{message}</span>
        {onClose && (
          <button
            type="button"
            className="alert-close"
            onClick={onClose}
            aria-label="Close"
          >
            &times;
          </button>
        )}
      </div>
    </div>
  );
}

export default Alert;
