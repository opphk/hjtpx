import React, { memo, useMemo } from 'react';
import PropTypes from 'prop-types';

const Input = memo(({
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
  const inputClasses = useMemo(() => [
    'form-input',
    error ? 'input-error' : '',
    disabled ? 'input-disabled' : '',
    className
  ].filter(Boolean).join(' '), [error, disabled, className]);

  const errorId = useMemo(() => error ? `${name}-error` : undefined, [error, name]);

  const describedByIds = useMemo(() => {
    const ids = [ariaDescribedBy, errorId].filter(Boolean);
    return ids.length > 0 ? ids.join(' ') : undefined;
  }, [ariaDescribedBy, errorId]);

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
});

Input.displayName = 'Input';

Input.propTypes = {
  label: PropTypes.string,
  type: PropTypes.string,
  name: PropTypes.string.isRequired,
  value: PropTypes.oneOfType([PropTypes.string, PropTypes.number]),
  onChange: PropTypes.func,
  placeholder: PropTypes.string,
  error: PropTypes.string,
  disabled: PropTypes.bool,
  required: PropTypes.bool,
  className: PropTypes.string,
  'aria-label': PropTypes.string,
  'aria-describedby': PropTypes.string,
};

export default Input;
