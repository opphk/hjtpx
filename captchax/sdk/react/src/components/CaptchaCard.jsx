import { useState, useCallback, useMemo } from 'react';
import './CaptchaCard.css';

/**
 * @typedef {Object} CaptchaCardProps
 * @property {string} [title]
 * @property {React.ReactNode} children
 * @property {React.ReactNode} [footer]
 * @property {'light' | 'dark'} [theme='light']
 * @property {boolean} [bordered=true]
 * @property {boolean} [hoverable=false]
 * @property {Function} [onRefresh]
 * @property {string} [className]
 */

/**
 * CaptchaCard - 验证码卡片组件
 * 提供卡片样式的验证码容器
 * @param {CaptchaCardProps} props
 * @returns {JSX.Element}
 */
export const CaptchaCard = ({
  title,
  children,
  footer,
  theme = 'light',
  bordered = true,
  hoverable = false,
  onRefresh,
  className = ''
}) => {
  const cardClasses = useMemo(() => {
    const classes = [
      'captcha-card',
      `captcha-card--${theme}`
    ];

    if (bordered) classes.push('captcha-card--bordered');
    if (hoverable) classes.push('captcha-card--hoverable');
    if (className) classes.push(className);

    return classes.join(' ');
  }, [theme, bordered, hoverable, className]);

  return (
    <div className={cardClasses}>
      {title && (
        <div className="captcha-card__header">
          <h4 className="captcha-card__title">{title}</h4>
          {onRefresh && (
            <button
              type="button"
              className="captcha-card__refresh"
              onClick={onRefresh}
              aria-label="刷新"
            >
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path
                  fill="currentColor"
                  d="M17.65 6.35A7.958 7.958 0 0012 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08A5.99 5.99 0 0112 18c-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"
                />
              </svg>
            </button>
          )}
        </div>
      )}

      <div className="captcha-card__body">
        {children}
      </div>

      {footer && (
        <div className="captcha-card__footer">
          {footer}
        </div>
      )}
    </div>
  );
};

export default CaptchaCard;
