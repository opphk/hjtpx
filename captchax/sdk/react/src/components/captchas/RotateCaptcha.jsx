import { useState, useCallback, useMemo, useRef, useEffect } from 'react';
import './RotateCaptcha.css';

/**
 * @typedef {Object} RotateCaptchaProps
 * @property {Object} [data]
 * @property {Function} [onSuccess]
 * @property {Function} [onRefresh]
 * @property {Function} [onError]
 * @property {string} [status='ready']
 * @property {'light' | 'dark'} [theme='light']
 * @property {number} [width=300]
 */

/**
 * RotateCaptcha - 旋转验证码组件
 * 拖动滑块旋转图片到正确角度完成验证
 * @param {RotateCaptchaProps} props
 * @returns {JSX.Element}
 */
export const RotateCaptcha = ({
  data,
  onSuccess,
  onRefresh,
  onError,
  status = 'ready',
  theme = 'light',
  width = 300
}) => {
  const [rotation, setRotation] = useState(0);
  const [isDragging, setIsDragging] = useState(false);
  const [isVerified, setIsVerified] = useState(false);
  const [error, setError] = useState(null);

  const containerRef = useRef(null);
  const startAngleRef = useRef(0);
  const startRotationRef = useRef(0);

  const targetRotation = data?.targetRotation ?? 45;
  const tolerance = 10;

  const handleDragStart = useCallback((e) => {
    if (isVerified) return;

    e.preventDefault();
    setIsDragging(true);

    const rect = containerRef.current.getBoundingClientRect();
    const centerX = rect.left + rect.width / 2;
    const centerY = rect.top + rect.height / 2;

    const clientX = e.type.includes('touch')
      ? e.touches[0].clientX
      : e.clientX;
    const clientY = e.type.includes('touch')
      ? e.touches[0].clientY
      : e.clientY;

    const angle = Math.atan2(clientY - centerY, clientX - centerX) * (180 / Math.PI);
    startAngleRef.current = angle;
    startRotationRef.current = rotation;
  }, [isVerified, rotation]);

  const handleDrag = useCallback((e) => {
    if (!isDragging) return;

    const rect = containerRef.current.getBoundingClientRect();
    const centerX = rect.left + rect.width / 2;
    const centerY = rect.top + rect.height / 2;

    const clientX = e.type.includes('touch')
      ? e.touches[0].clientX
      : e.clientX;
    const clientY = e.type.includes('touch')
      ? e.touches[0].clientY
      : e.clientY;

    const angle = Math.atan2(clientY - centerY, clientX - centerX) * (180 / Math.PI);
    const deltaAngle = angle - startAngleRef.current;
    const newRotation = startRotationRef.current + deltaAngle;

    setRotation(newRotation);
  }, [isDragging]);

  const handleDragEnd = useCallback(() => {
    if (!isDragging) return;

    setIsDragging(false);

    const normalizedRotation = ((rotation % 360) + 360) % 360;
    const normalizedTarget = ((targetRotation % 360) + 360) % 360;

    let minDiff = Math.abs(normalizedRotation - normalizedTarget);
    minDiff = Math.min(minDiff, 360 - minDiff);

    if (minDiff <= tolerance) {
      setIsVerified(true);
      onSuccess?.({
        type: 'rotate',
        rotation: normalizedRotation,
        target: normalizedTarget,
        accuracy: minDiff
      });
    } else {
      setError('角度不正确，请重试');
      setRotation(0);
      onError?.({
        rotation: normalizedRotation,
        target: normalizedTarget,
        accuracy: minDiff
      });
    }
  }, [isDragging, rotation, targetRotation, onSuccess, onError]);

  useEffect(() => {
    if (isDragging) {
      document.addEventListener('mousemove', handleDrag);
      document.addEventListener('mouseup', handleDragEnd);
      document.addEventListener('touchmove', handleDrag, { passive: false });
      document.addEventListener('touchend', handleDragEnd);
    }

    return () => {
      document.removeEventListener('mousemove', handleDrag);
      document.removeEventListener('mouseup', handleDragEnd);
      document.removeEventListener('touchmove', handleDrag);
      document.removeEventListener('touchend', handleDragEnd);
    };
  }, [isDragging, handleDrag, handleDragEnd]);

  useEffect(() => {
    if (status === 'ready') {
      setRotation(0);
      setIsVerified(false);
      setError(null);
    }
  }, [status]);

  const containerClasses = useMemo(() => {
    const classes = [
      'rotate-captcha',
      `rotate-captcha--${theme}`
    ];

    if (isVerified) classes.push('rotate-captcha--verified');

    return classes.join(' ');
  }, [theme, isVerified]);

  return (
    <div className={containerClasses} style={{ width }}>
      <div className="rotate-captcha__header">
        <span className="rotate-captcha__hint">
          拖动下方滑块旋转图片至正常角度
        </span>
      </div>

      <div
        ref={containerRef}
        className="rotate-captcha__image-container"
      >
        <div
          className="rotate-captcha__image"
          style={{ transform: `rotate(${rotation}deg)` }}
        >
          {data?.image ? (
            <img src={data.image} alt="Rotate captcha" />
          ) : (
            <div className="rotate-captcha__placeholder">
              <svg viewBox="0 0 100 100" width="100" height="100">
                <rect x="20" y="20" width="60" height="60" fill="#1890ff" opacity="0.8"/>
                <circle cx="50" cy="50" r="10" fill="#ffffff"/>
                <path d="M50 30 L55 40 L45 40 Z" fill="#ffffff"/>
              </svg>
            </div>
          )}
        </div>

        {isVerified && (
          <div className="rotate-captcha__success-overlay">
            <svg viewBox="0 0 24 24" width="48" height="48">
              <path fill="#52c41a" d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
            </svg>
          </div>
        )}
      </div>

      <div
        className={`rotate-captcha__slider ${isDragging ? 'dragging' : ''}`}
        onMouseDown={handleDragStart}
        onTouchStart={handleDragStart}
      >
        <div className="rotate-captcha__slider-track">
          <div
            className="rotate-captcha__slider-thumb"
            style={{ transform: `rotate(${rotation}deg)` }}
          >
            <svg viewBox="0 0 24 24" width="20" height="20">
              <path fill="currentColor" d="M15.55 5.55L11 1v3.07C7.06 4.56 4 7.92 4 12s3.05 7.44 7 7.93v-2.02c-2.84-.48-5-2.94-5-5.91s2.16-5.43 5-5.91V10l4.55-4.45zM19.93 11c-.17-1.39-.72-2.73-1.62-3.89l-1.42 1.42c.54.75.88 1.6 1.02 2.47h2.02zM13 17.9v2.02c1.39-.17 2.74-.71 3.9-1.6l-1.44-1.44c-.75.54-1.59.89-2.46 1.02zm3.89-2.42l1.42 1.41c.9-1.16 1.45-2.5 1.62-3.89h-2.02c-.14.87-.48 1.72-1.02 2.48z"/>
            </svg>
          </div>
        </div>
        <div className="rotate-captcha__angle-display">
          {Math.round(rotation)}°
        </div>
      </div>

      {error && (
        <div className="rotate-captcha__error">
          <span>{error}</span>
          <button onClick={() => {
            setRotation(0);
            setError(null);
          }}>重新验证</button>
        </div>
      )}
    </div>
  );
};

export default RotateCaptcha;
