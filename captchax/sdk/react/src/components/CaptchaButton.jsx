import { useState, useCallback, useMemo } from 'react';
import { useCaptcha } from '../CaptchaProvider';
import { CAPTCHA_STATUS } from '../utils/constants';
import './CaptchaButton.css';

/**
 * @typedef {'small' | 'medium' | 'large'} ButtonSize
 * @typedef {'light' | 'dark'} Theme
 */

/**
 * @typedef {Object} CaptchaButtonProps
 * @property {React.ReactNode} children
 * @property {string} [scene='default']
 * @property {'slider' | 'click' | 'rotate' | 'puzzle' | 'text' | 'icon'} [captchaType='slider']
 * @property {(token: string) => void} [onSuccess]
 * @property {(error: Error) => void} [onError]
 * @property {Theme} [theme='light']
 * @property {ButtonSize} [size='medium']
 * @property {boolean} [disabled=false]
 * @property {string} [className]
 * @property {Object} [buttonProps]
 */

/**
 * CaptchaButton - 验证码按钮组件
 * 提供点击触发验证的功能，支持多种验证码类型
 * @param {CaptchaButtonProps} props
 * @returns {JSX.Element}
 */
export const CaptchaButton = ({
  children,
  scene = 'default',
  captchaType = 'slider',
  onSuccess,
  onError,
  theme = 'light',
  size = 'medium',
  disabled = false,
  className = '',
  ...buttonProps
}) => {
  const { verify, loading, error, clearError } = useCaptcha();
  const [status, setStatus] = useState(CAPTCHA_STATUS.IDLE);
  const [showCaptcha, setShowCaptcha] = useState(false);

  const handleClick = useCallback(async () => {
    if (disabled || loading) return;

    setStatus(CAPTCHA_STATUS.VERIFYING);
    clearError?.();

    try {
      const token = await verify(scene);
      setStatus(CAPTCHA_STATUS.SUCCESS);
      onSuccess?.(token);

      setTimeout(() => {
        setStatus(CAPTCHA_STATUS.IDLE);
      }, 2000);
    } catch (err) {
      setStatus(CAPTCHA_STATUS.FAILED);
      onError?.(err);

      setTimeout(() => {
        setStatus(CAPTCHA_STATUS.IDLE);
      }, 3000);
    }
  }, [disabled, loading, verify, scene, onSuccess, onError, clearError]);

  const buttonClasses = useMemo(() => {
    const classes = [
      'captcha-button',
      `captcha-button--${size}`,
      `captcha-button--${theme}`,
      `captcha-button--${status}`
    ];

    if (className) classes.push(className);
    if (loading) classes.push('captcha-button--loading');
    if (disabled) classes.push('captcha-button--disabled');

    return classes.join(' ');
  }, [size, theme, status, className, loading, disabled]);

  const getButtonText = () => {
    if (loading) return '验证中...';
    if (status === CAPTCHA_STATUS.SUCCESS) return '验证成功';
    if (status === CAPTCHA_STATUS.FAILED) return '验证失败';
    if (error) return '点击重试';
    return children;
  };

  return (
    <div className="captcha-button-wrapper">
      <button
        type="button"
        className={buttonClasses}
        onClick={handleClick}
        disabled={disabled || loading}
        {...buttonProps}
      >
        <span className="captcha-button__text">{getButtonText()}</span>
        {loading && (
          <span className="captcha-button__spinner">
            <svg viewBox="0 0 24 24" className="spinner-icon">
              <circle
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                strokeWidth="3"
                fill="none"
                strokeDasharray="31.416"
                strokeLinecap="round"
              />
            </svg>
          </span>
        )}
      </button>
      {error && (
        <div className="captcha-button__error">
          {error}
        </div>
      )}
    </div>
  );
};

export default CaptchaButton;
