import React from 'react';
import Loading from './Loading';

function Button({
  children,
  type = 'button',
  variant = 'primary',
  size = 'medium',
  loading = false,
  disabled = false,
  onClick,
  className = '',
  'aria-label': ariaLabel,
  'aria-describedby': ariaDescribedBy,
  ...props
}) {
  const baseClass = 'btn';
  const variantClass = `btn-${variant}`;
  const sizeClass = `btn-${size}`;
  const classes = [baseClass, variantClass, sizeClass, className].filter(Boolean).join(' ');

  const isDisabled = disabled || loading;
  const buttonLabel = ariaLabel || (typeof children === 'string' ? children : null);

  const handleClick = (e) => {
    if (!isDisabled && onClick) {
      onClick(e);
    }
  };

  const handleKeyDown = (e) => {
    if (e.key === 'Enter' || e.key === ' ') {
      if (!isDisabled && onClick) {
        e.preventDefault();
        onClick(e);
      }
    }
  };

  return (
    <button
      type={type}
      className={classes}
      onClick={handleClick}
      onKeyDown={handleKeyDown}
      disabled={isDisabled}
      aria-label={buttonLabel}
      aria-describedby={ariaDescribedBy}
      aria-disabled={isDisabled}
      aria-busy={loading}
      tabIndex={isDisabled ? -1 : 0}
      {...props}
    >
      {loading ? <Loading size="small" aria-hidden="true" /> : children}
    </button>
  );
}

export default React.memo(Button);
