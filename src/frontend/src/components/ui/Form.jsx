import React, { memo, useState, useCallback, useMemo, useRef, useEffect } from 'react';
import PropTypes from 'prop-types';
import Input from './Input';
import Button from './Button';

const Form = memo(({
  fields = [],
  initialValues = {},
  validationSchema = {},
  onSubmit,
  submitText = '提交',
  loading = false,
  className = '',
  children,
  'aria-label': ariaLabel,
}) => {
  const [values, setValues] = useState(() => {
    const initial = { ...initialValues };
    fields.forEach(field => {
      if (!(field.name in initial)) {
        initial[field.name] = field.type === 'checkbox' ? false : '';
      }
    });
    return initial;
  });
  const [errors, setErrors] = useState({});
  const [touched, setTouched] = useState({});
  const formRef = useRef(null);
  const previousFocusRef = useRef(null);

  const handleChange = useCallback((e) => {
    const { name, value, type, checked } = e.target;
    const newValue = type === 'checkbox' ? checked : value;

    setValues(prev => ({ ...prev, [name]: newValue }));

    if (errors[name]) {
      setErrors(prev => ({ ...prev, [name]: '' }));
    }
  }, [errors]);

  const handleBlur = useCallback((e) => {
    const { name } = e.target;
    setTouched(prev => ({ ...prev, [name]: true }));

    if (validationSchema[name]) {
      const error = validateField(name, values[name]);
      setErrors(prev => ({ ...prev, [name]: error }));
    }
  }, [values, validationSchema]);

  const validateField = useCallback((name, value) => {
    const validator = validationSchema[name];
    if (!validator) return '';

    if (typeof validator === 'function') {
      return validator(value, values);
    }

    if (typeof validator === 'object') {
      const { required, minLength, maxLength, pattern, message } = validator;

      if (required && (!value || (typeof value === 'string' && !value.trim()))) {
        return message || `${name}不能为空`;
      }

      if (minLength && value.length < minLength) {
        return message || `${name}至少${minLength}个字符`;
      }

      if (maxLength && value.length > maxLength) {
        return message || `${name}最多${maxLength}个字符`;
      }

      if (pattern && !pattern.test(value)) {
        return message || `${name}格式不正确`;
      }
    }

    return '';
  }, [values]);

  const validate = useCallback(() => {
    const newErrors = {};
    let isValid = true;

    fields.forEach(field => {
      const error = validateField(field.name, values[field.name]);
      if (error) {
        newErrors[field.name] = error;
        isValid = false;
      }
    });

    setErrors(newErrors);
    setTouched(
      fields.reduce((acc, field) => ({ ...acc, [field.name]: true }), {})
    );

    return isValid;
  }, [fields, values, validateField]);

  const handleSubmit = useCallback((e) => {
    e.preventDefault();

    if (validate()) {
      onSubmit(values);
    }
  }, [validate, values, onSubmit]);

  const reset = useCallback(() => {
    setValues(() => {
      const initial = { ...initialValues };
      fields.forEach(field => {
        if (!(field.name in initial)) {
          initial[field.name] = field.type === 'checkbox' ? false : '';
        }
      });
      return initial;
    });
    setErrors({});
    setTouched({});
  }, [initialValues, fields]);

  const fieldErrors = useMemo(() => {
    return fields.reduce((acc, field) => ({
      ...acc,
      [field.name]: touched[field.name] ? errors[field.name] : ''
    }), {});
  }, [fields, touched, errors]);

  useEffect(() => {
    if (formRef.current) {
      const firstError = formRef.current.querySelector('.input-error');
      if (firstError) {
        firstError.focus();
      }
    }
  }, [errors]);

  const getFieldProps = useCallback((fieldName) => ({
    name: fieldName,
    value: values[fieldName] || '',
    onChange: handleChange,
    onBlur: handleBlur,
    error: fieldErrors[fieldName],
  }), [values, handleChange, handleBlur, fieldErrors]);

  return (
    <form
      ref={formRef}
      onSubmit={handleSubmit}
      className={className}
      aria-label={ariaLabel}
      noValidate
    >
      {fields.map(field => (
        <Input
          key={field.name}
          {...field}
          {...getFieldProps(field.name)}
        />
      ))}

      {children}

      <Button
        type="submit"
        loading={loading}
        disabled={loading}
      >
        {submitText}
      </Button>
    </form>
  );
});

Form.displayName = 'Form';

Form.propTypes = {
  fields: PropTypes.arrayOf(
    PropTypes.shape({
      name: PropTypes.string.isRequired,
      label: PropTypes.string,
      type: PropTypes.string,
      placeholder: PropTypes.string,
      required: PropTypes.bool,
      disabled: PropTypes.bool,
      validation: PropTypes.oneOfType([
        PropTypes.func,
        PropTypes.shape({
          required: PropTypes.bool,
          minLength: PropTypes.number,
          maxLength: PropTypes.number,
          pattern: PropTypes.instanceOf(RegExp),
          message: PropTypes.string,
        }),
      ]),
    })
  ),
  initialValues: PropTypes.object,
  validationSchema: PropTypes.objectOf(
    PropTypes.oneOfType([
      PropTypes.func,
      PropTypes.shape({
        required: PropTypes.bool,
        minLength: PropTypes.number,
        maxLength: PropTypes.number,
        pattern: PropTypes.instanceOf(RegExp),
        message: PropTypes.string,
      }),
    ])
  ),
  onSubmit: PropTypes.func.isRequired,
  submitText: PropTypes.string,
  loading: PropTypes.bool,
  className: PropTypes.string,
  children: PropTypes.node,
  'aria-label': PropTypes.string,
};

export default Form;
