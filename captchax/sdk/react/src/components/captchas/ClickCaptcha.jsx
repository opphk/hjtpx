import { useState, useCallback, useMemo, useEffect } from 'react';
import './ClickCaptcha.css';

/**
 * @typedef {Object} ClickCaptchaProps
 * @property {Object} [data]
 * @property {Function} [onSuccess]
 * @property {Function} [onRefresh]
 * @property {Function} [onError]
 * @property {string} [status='ready']
 * @property {'light' | 'dark'} [theme='light']
 * @property {number} [width=300]
 * @property {number} [height=150]
 */

/**
 * ClickCaptcha - 点击验证码组件
 * 根据提示点击正确位置的图片区域完成验证
 * @param {ClickCaptchaProps} props
 * @returns {JSX.Element}
 */
export const ClickCaptcha = ({
  data,
  onSuccess,
  onRefresh,
  onError,
  status = 'ready',
  theme = 'light',
  width = 300,
  height = 150
}) => {
  const [clicks, setClicks] = useState([]);
  const [isVerified, setIsVerified] = useState(false);
  const [error, setError] = useState(null);

  const targets = data?.targets || [
    { x: 45, y: 35 },
    { x: 72, y: 58 },
    { x: 28, y: 72 }
  ];

  const hint = data?.hint || '请依次点击: 第1处 第2处 第3处';

  const handleImageClick = useCallback((e) => {
    if (isVerified || status !== 'ready') return;

    const rect = e.currentTarget.getBoundingClientRect();
    const x = ((e.clientX - rect.left) / rect.width) * 100;
    const y = ((e.clientY - rect.top) / rect.height) * 100;

    const newClicks = [...clicks, { x, y, id: Date.now() }];
    setClicks(newClicks);

    if (newClicks.length >= targets.length) {
      const accuracy = calculateAccuracy(newClicks, targets);
      const tolerance = 8;

      if (accuracy <= tolerance) {
        setIsVerified(true);
        onSuccess?.({
          type: 'click',
          clicks: newClicks,
          targets,
          accuracy
        });
      } else {
        setError('验证失败，请重试');
        setClicks([]);
        onError?.({
          clicks: newClicks,
          targets,
          accuracy
        });
      }
    }
  }, [isVerified, status, clicks, targets, onSuccess, onError]);

  const calculateAccuracy = (clicks, targets) => {
    if (clicks.length !== targets.length) return Infinity;

    let totalDistance = 0;
    for (let i = 0; i < clicks.length; i++) {
      const dx = clicks[i].x - targets[i].x;
      const dy = clicks[i].y - targets[i].y;
      totalDistance += Math.sqrt(dx * dx + dy * dy);
    }

    return totalDistance / clicks.length;
  };

  const handleReset = useCallback(() => {
    setClicks([]);
    setIsVerified(false);
    setError(null);
  }, []);

  useEffect(() => {
    if (status === 'ready') {
      handleReset();
    }
  }, [status]);

  const containerClasses = useMemo(() => {
    const classes = [
      'click-captcha',
      `click-captcha--${theme}`
    ];

    if (isVerified) classes.push('click-captcha--verified');

    return classes.join(' ');
  }, [theme, isVerified]);

  return (
    <div className={containerClasses} style={{ width, height }}>
      <div className="click-captcha__header">
        <span className="click-captcha__hint">{hint}</span>
        <span className="click-captcha__progress">
          {clicks.length} / {targets.length}
        </span>
      </div>

      <div
        className="click-captcha__image-container"
        onClick={handleImageClick}
      >
        {data?.image ? (
          <img src={data.image} alt="Click captcha" className="click-captcha__image" />
        ) : (
          <div className="click-captcha__placeholder">
            <div className="click-captcha__grid">
              {[...Array(9)].map((_, i) => (
                <div key={i} className="click-captcha__cell" />
              ))}
            </div>
          </div>
        )}

        {clicks.map((click, index) => (
          <div
            key={click.id}
            className="click-captcha__marker"
            style={{ left: `${click.x}%`, top: `${click.y}%` }}
          >
            {index + 1}
          </div>
        ))}

        {isVerified && (
          <div className="click-captcha__success-overlay">
            <svg viewBox="0 0 24 24" width="48" height="48">
              <path fill="#52c41a" d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
            </svg>
          </div>
        )}

        {!isVerified && (
          <div className="click-captcha__refresh-btn" onClick={(e) => {
            e.stopPropagation();
            onRefresh?.();
          }}>
            <svg viewBox="0 0 24 24" width="16" height="16">
              <path fill="currentColor" d="M17.65 6.35A7.958 7.958 0 0012 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08A5.99 5.99 0 0112 18c-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"/>
            </svg>
          </div>
        )}
      </div>

      {error && (
        <div className="click-captcha__error">
          <span>{error}</span>
          <button onClick={handleReset}>重新验证</button>
        </div>
      )}
    </div>
  );
};

export default ClickCaptcha;
