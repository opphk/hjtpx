import React from 'react';

function Loading({ 
  size = 'medium', 
  fullScreen = false, 
  text = 'Loading...',
  'aria-label': ariaLabel,
  role = 'status'
}) {
  const spinnerSizes = {
    small: '20px',
    medium: '40px',
    large: '60px'
  };

  const spinnerSize = spinnerSizes[size] || spinnerSizes.medium;

  const containerClass = fullScreen ? 'loading-fullscreen' : 'loading-inline';

  return (
    <div 
      className={containerClass}
      role={role}
      aria-label={ariaLabel || text}
      aria-live="polite"
      aria-busy={true}
    >
      <div 
        className="loading-spinner" 
        style={{ width: spinnerSize, height: spinnerSize }}
        aria-hidden="true"
      >
        <div className="spinner"></div>
      </div>
      {text && <p className="loading-text">{text}</p>}
    </div>
  );
}

export default React.memo(Loading);
