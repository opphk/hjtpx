import React from 'react';

function Loading({ size = 'medium', fullScreen = false, text = 'Loading...' }) {
  const spinnerSizes = {
    small: '20px',
    medium: '40px',
    large: '60px'
  };

  const spinnerSize = spinnerSizes[size] || spinnerSizes.medium;

  const containerClass = fullScreen ? 'loading-fullscreen' : 'loading-inline';

  return (
    <div className={containerClass}>
      <div className="loading-spinner" style={{ width: spinnerSize, height: spinnerSize }}>
        <div className="spinner"></div>
      </div>
      {text && <p className="loading-text">{text}</p>}
    </div>
  );
}

export default Loading;
