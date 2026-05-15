import React from 'react';

const Alert = ({ 
  type = 'info', 
  message, 
  description,
  closable = false,
  onClose,
  className = '',
  role = 'alert',
  'aria-label': ariaLabel,
  'aria-live': ariaLive = type === 'error' || type === 'warning' ? 'assertive' : 'polite',
  ...props
}) => {
  const alertClasses = [
    'alert',
    `alert-${type}`,
    className
  ].filter(Boolean).join(' ');
  
  const alertId = React.useId();
  const messageId = `${alertId}-message`;
  const descriptionId = `${alertId}-description`;
  
  const getRoleFromType = () => {
    if (role) return role;
    switch (type) {
      case 'error':
        return 'alert';
      case 'warning':
        return 'alert';
      case 'success':
        return 'status';
      default:
        return 'status';
    }
  };

  return (
    <div 
      className={alertClasses}
      role={getRoleFromType()}
      aria-label={ariaLabel}
      aria-live={ariaLive}
      aria-atomic="true"
      aria-labelledby={!ariaLabel && message ? messageId : undefined}
      aria-describedby={description ? descriptionId : undefined}
      {...props}
    >
      <div className="alert-content">
        <span className="alert-message" id={messageId}>{message}</span>
        {description && (
          <span className="alert-description" id={descriptionId}>{description}</span>
        )}
      </div>
      {closable && (
        <button 
          className="alert-close" 
          onClick={onClose}
          aria-label="关闭提示"
          type="button"
        >
          ×
        </button>
      )}
    </div>
  );
};

export default Alert;
