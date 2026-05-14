import React from 'react';

const Loading = ({ 
  size = 'medium', 
  text = '加载中...',
  fullScreen = false,
  className = ''
}) => {
  const loadingClasses = [
    'loading',
    `loading-${size}`,
    fullScreen ? 'loading-fullscreen' : '',
    className
  ].filter(Boolean).join(' ');

  if (fullScreen) {
    return (
      <div className={loadingClasses}>
        <div className="loading-spinner"></div>
        {text && <p className="loading-text">{text}</p>}
      </div>
    );
  }

  return (
    <div className={loadingClasses}>
      <div className="loading-spinner"></div>
      {text && <p className="loading-text">{text}</p>}
    </div>
  );
};

export default Loading;
