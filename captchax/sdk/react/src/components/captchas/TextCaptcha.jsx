import { useState, useCallback, useMemo } from 'react';
import './TextCaptcha.css';

/**
 * @typedef {Object} TextCaptchaProps
 * @property {Object} [data]
 * @property {Function} [onSuccess]
 * @property {Function} [onRefresh]
 * @property {Function} [onError]
 * @property {string} [status='ready']
 * @property {'light' | 'dark'} [theme='light']
 * @property {number} [length=4]
 */

/**
 * TextCaptcha - 文字验证码组件
 * 输入显示的文字完成验证
 * @param {TextCaptchaProps} props
 * @returns {JSX.Element}
 */
export const TextCaptcha = ({
  data,
  onSuccess,
  onRefresh,
  onError,
  status = 'ready',
  theme = 'light',
  length = 4
}) => {
  const [inputValue, setInputValue] = useState('');
  const [isVerified, setIsVerified] = useState(false);
  const [error, setError] = useState('');

  const captchaText = data?.text || 'ABCD';
  const hint = data?.hint || '请输入上方验证码';

  const handleInputChange = useCallback((e) => {
    const value = e.target.value.toUpperCase().slice(0, length);
    setInputValue(value);
    setError('');

    if (value.length === length) {
      if (value === captchaText) {
        setIsVerified(true);
        onSuccess?.({
          type: 'text',
          input: value,
          expected: captchaText
        });
      } else {
        setError('验证码错误');
        setInputValue('');
        onError?.({
          input: value,
          expected: captchaText
        });
      }
    }
  }, [captchaText, length, onSuccess, onError]);

  const handleSubmit = useCallback((e) => {
    e.preventDefault();

    if (inputValue.length < length) {
      setError(`请输入${length}位验证码`);
      return;
    }

    if (inputValue.toUpperCase() === captchaText) {
      setIsVerified(true);
      onSuccess?.({
        type: 'text',
        input: inputValue,
        expected: captchaText
      });
    } else {
      setError('验证码错误');
      setInputValue('');
      onError?.({
        input: inputValue,
        expected: captchaText
      });
    }
  }, [inputValue, captchaText, length, onSuccess, onError]);

  const containerClasses = useMemo(() => {
    const classes = [
      'text-captcha',
      `text-captcha--${theme}`
    ];

    if (isVerified) classes.push('text-captcha--verified');

    return classes.join(' ');
  }, [theme, isVerified]);

  return (
    <div className={containerClasses}>
      <div className="text-captcha__header">
        <span className="text-captcha__hint">{hint}</span>
      </div>

      <div className="text-captcha__display">
        <div className="text-captcha__code">
          {captchaText.split('').map((char, index) => (
            <span
              key={index}
              className="text-captcha__char"
              style={{
                transform: `rotate(${(Math.random() - 0.5) * 30}deg)`,
                animationDelay: `${index * 0.1}s`
              }}
            >
              {char}
            </span>
          ))}
        </div>

        <button
          type="button"
          className="text-captcha__refresh"
          onClick={onRefresh}
          aria-label="刷新验证码"
        >
          <svg viewBox="0 0 24 24" width="20" height="20">
            <path fill="currentColor" d="M17.65 6.35A7.958 7.958 0 0012 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08A5.99 5.99 0 0112 18c-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"/>
          </svg>
        </button>
      </div>

      <form className="text-captcha__form" onSubmit={handleSubmit}>
        <div className="text-captcha__input-wrapper">
          {Array.from({ length }).map((_, index) => (
            <div
              key={index}
              className={`text-captcha__cell ${
                inputValue[index] ? 'filled' : ''
              }`}
            >
              {inputValue[index] || ''}
            </div>
          ))}
          <input
            type="text"
            className="text-captcha__input"
            value={inputValue}
            onChange={handleInputChange}
            maxLength={length}
            placeholder=""
            autoComplete="off"
            autoCapitalize="characters"
          />
        </div>

        <button
          type="submit"
          className="text-captcha__submit"
          disabled={inputValue.length < length || isVerified}
        >
          {isVerified ? '验证成功' : '确认'}
        </button>
      </form>

      {error && (
        <div className="text-captcha__error">
          {error}
        </div>
      )}
    </div>
  );
};

export default TextCaptcha;
