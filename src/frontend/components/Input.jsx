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
  ...props
}, ref) => {
  const inputId = name || `input-${Math.random().toString(36).substr(2, 9)}`;
  const errorId = error ? `${inputId}-error` : undefined;
  const classes = ['input-wrapper', error ? 'input-error' : '', className].filter(Boolean).join(' ');
  
  const describedByIds = [ariaDescribedBy, errorId].filter(Boolean).join(' ') || undefined;

  const handleChange = (e) => {
    if (onChange) {
      onChange(e);
    }
  };

  return (
    <div className={classes}>
      {label && (
        <label htmlFor={inputId} className="input-label">
          {label}
          {required && <span className="required">*</span>}
        </label>
      )}
      <input
        ref={ref}
        type={type}
        id={inputId}
        name={name}
        value={value}
        onChange={handleChange}
        placeholder={placeholder}
        required={required}
        disabled={disabled}
        className="input-field"
        aria-label={ariaLabel}
        aria-describedby={describedByIds}
        aria-invalid={!!error}
        aria-required={required}
        aria-disabled={disabled}
        {...props}
      />
      {error && (
        <span 
          className="input-error-message" 
          id={errorId}
          role="alert"
        >
          {error}
        </span>
      )}
    </div>
  );
});

Input.displayName = 'Input';

export default React.memo(Input);
