import React from 'react';

const Input = ({ 
  label,
  type = 'text',
  name,
  value,
  onChange,
  placeholder,
  error,
  disabled = false,
  required = false,
  className = '',
  'aria-label': ariaLabel,
  'aria-describedby': ariaDescribedBy,
  ...props
}) => {
  const inputClasses = [
    'form-input',
    error ? 'input-error' : '',
    disabled ? 'input-disabled' : '',
    className
  ].filter(Boolean).join(' ');
  
  const errorId = error ? `${name}-error` : undefined;
  
  const describedByIds = [
    ariaDescribedBy,
    errorId
  ].filter(Boolean).join(' ') || undefined;

  return (
    <div className="form-group">
      {label && (
        <label htmlFor={name} className="form-label">
          {label}
          {required && <span className="required">*</span>}
        </label>
      )}
      <input
        type={type}
        id={name}
        name={name}
        value={value}
        onChange={onChange}
        placeholder={placeholder}
        disabled={disabled}
        required={required}
        className={inputClasses}
        aria-label={ariaLabel}
        aria-describedby={describedByIds}
        aria-invalid={!!error}
        aria-required={required}
        aria-disabled={disabled}
        {...props}
      />
      {error && (
        <span className="error-text" id={errorId} role="alert">
          {error}
        </span>
      )}
    </div>
  );
};

export default Input;
