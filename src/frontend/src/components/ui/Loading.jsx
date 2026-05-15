import React from 'react';

const Loading = ({ 
  size = 'medium', 
  text = '加载中...',
  fullScreen = false,
  className = '',
  role = 'status',
  'aria-label': ariaLabel,
  'aria-live': ariaLive = 'polite',
  ...props
}) => {
  const loadingClasses = [
    'loading',
    `loading-${size}`,
    fullScreen ? 'loading-fullscreen' : '',
    className
  ].filter(Boolean).join(' ');

  if (fullScreen) {
    return (
      <div 
        className={loadingClasses}
        role={role}
        aria-label={ariaLabel || text}
        aria-live={ariaLive}
        aria-busy={true}
        {...props}
      >
        <div className="loading-spinner" aria-hidden="true"></div>
        {text && <p className="loading-text">{text}</p>}
      </div>
    );
  }

  return (
    <div 
      className={loadingClasses}
      role={role}
      aria-label={ariaLabel || text}
      aria-live={ariaLive}
      aria-busy={true}
      {...props}
    >
      <div className="loading-spinner" aria-hidden="true"></div>
      {text && <p className="loading-text">{text}</p>}
    </div>
  );
};

export default Loading;
