import { useState, useCallback, useMemo, useRef, useEffect } from 'react';
import './CaptchaInput.css';

/**
 * @typedef {Object} CaptchaInputProps
 * @property {string} [value]
 * @property {(value: string) => void} [onChange]
 * @property {(value: string) => void} [onComplete]
 * @property {number} [length=4]
 * @property {boolean} [autoFocus=false]
 * @property {'light' | 'dark'} [theme='light']
 * @property {string} [placeholder='请输入验证码']
 * @property {boolean} [disabled=false]
 * @property {string} [className]
 */

/**
 * CaptchaInput - 验证码输入框组件
 * 提供验证码输入功能，支持多个字符输入框
 * @param {CaptchaInputProps} props
 * @returns {JSX.Element}
 */
export const CaptchaInput = ({
  value = '',
  onChange,
  onComplete,
  length = 4,
  autoFocus = false,
  theme = 'light',
  placeholder = '请输入验证码',
  disabled = false,
  className = ''
}) => {
  const [focusedIndex, setFocusedIndex] = useState(-1);
  const inputRefs = useRef([]);

  const handleChange = useCallback((index, char) => {
    const newValue = value.split('');
    newValue[index] = char;

    if (char && index < length - 1) {
      inputRefs.current[index + 1]?.focus();
    }

    const result = newValue.join('').slice(0, length);
    onChange?.(result);

    if (result.length === length) {
      onComplete?.(result);
    }
  }, [value, length, onChange, onComplete]);

  const handleKeyDown = useCallback((index, e) => {
    if (e.key === 'Backspace' && !value[index] && index > 0) {
      inputRefs.current[index - 1]?.focus();
    } else if (e.key === 'ArrowLeft' && index > 0) {
      inputRefs.current[index - 1]?.focus();
    } else if (e.key === 'ArrowRight' && index < length - 1) {
      inputRefs.current[index + 1]?.focus();
    }
  }, [value, length]);

  const handlePaste = useCallback((e) => {
    e.preventDefault();
    const pastedData = e.clipboardData.getData('text').slice(0, length);
    onChange?.(pastedData);
    if (pastedData.length === length) {
      onComplete?.(pastedData);
    }
  }, [length, onChange, onComplete]);

  useEffect(() => {
    if (autoFocus && inputRefs.current[0]) {
      inputRefs.current[0].focus();
    }
  }, [autoFocus]);

  const containerClasses = useMemo(() => {
    const classes = [
      'captcha-input',
      `captcha-input--${theme}`
    ];

    if (disabled) classes.push('captcha-input--disabled');
    if (className) classes.push(className);

    return classes.join(' ');
  }, [theme, disabled, className]);

  return (
    <div className={containerClasses}>
      <div className="captcha-input__container">
        {Array.from({ length }).map((_, index) => (
          <input
            key={index}
            ref={el => inputRefs.current[index] = el}
            type="text"
            inputMode="numeric"
            pattern="[0-9]*"
            maxLength={1}
            value={value[index] || ''}
            onChange={(e) => handleChange(index, e.target.value)}
            onKeyDown={(e) => handleKeyDown(index, e)}
            onFocus={() => setFocusedIndex(index)}
            onBlur={() => setFocusedIndex(-1)}
            onPaste={index === 0 ? handlePaste : undefined}
            placeholder={index === 0 ? placeholder : ''}
            disabled={disabled}
            className={`captcha-input__cell ${
              focusedIndex === index ? 'captcha-input__cell--focused' : ''
            } ${value[index] ? 'captcha-input__cell--filled' : ''}`}
            aria-label={`验证码第 ${index + 1} 位`}
          />
        ))}
      </div>
    </div>
  );
};

export default CaptchaInput;
