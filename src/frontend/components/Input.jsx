import React, { forwardRef } from 'react';

const Input = forwardRef(({
  type = 'text',
  name,
  label,
  value,
  onChange,
  error,
  placeholder,
  required = false,
  disabled = false,
  className = '',
  'aria-label': ariaLabel,
  'aria-describedby': ariaDescribedBy,
  helpText,
  ...props
}, ref) => {
  const inputId = name || `input-${Math.random().toString(36).substr(2, 9)}`;
  const errorId = `${inputId}-error`;
  const helpId = `${inputId}-help`;
  const classes = ['input-wrapper', error ? 'input-error' : '', className].filter(Boolean).join(' ');

  const describedByIds = [
    error ? errorId : null,
    helpText ? helpId : null,
    ariaDescribedBy || null
  ].filter(Boolean).join(' ') || undefined;

  const handleChange = (e) => {
    if (onChange) {
      onChange(e);
    }
  };

  const handleKeyDown = (e) => {
    if (e.key === 'Enter' && onChange) {
      e.preventDefault();
    }
  };

  return (
    <div className={classes}>
      {label && (
        <label htmlFor={inputId} className="input-label">
          {label}
          {required && <span className="required" aria-hidden="true">*</span>}
          {required && <span className="visually-hidden">(required)</span>}
        </label>
      )}
      <input
        ref={ref}
        type={type}
        id={inputId}
        name={name}
        value={value}
        onChange={handleChange}
        onKeyDown={handleKeyDown}
        placeholder={placeholder}
        required={required}
        disabled={disabled}
        className="input-field"
        aria-invalid={!!error}
        aria-describedby={describedByIds}
        aria-required={required}
        aria-label={ariaLabel}
        tabIndex={disabled ? -1 : 0}
        {...props}
      />
      {helpText && !error && (
        <span id={helpId} className="input-help-text">
          {helpText}
        </span>
      )}
      {error && (
        <span id={errorId} className="input-error-message" role="alert" aria-live="polite">
          {error}
        </span>
      )}
    </div>
  );
});

Input.displayName = 'Input';

export default React.memo(Input);
