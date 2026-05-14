import React from 'react';

const Alert = ({ 
  type = 'info', 
  message, 
  description,
  closable = false,
  onClose,
  className = ''
}) => {
  const alertClasses = [
    'alert',
    `alert-${type}`,
    className
  ].filter(Boolean).join(' ');

  return (
    <div className={alertClasses}>
      <div className="alert-content">
        <span className="alert-message">{message}</span>
        {description && (
          <span className="alert-description">{description}</span>
        )}
      </div>
      {closable && (
        <button className="alert-close" onClick={onClose}>
          ×
        </button>
      )}
    </div>
  );
};

export default Alert;
