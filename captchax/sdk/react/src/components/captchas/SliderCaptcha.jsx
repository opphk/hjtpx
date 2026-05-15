import { useState, useCallback, useMemo, useRef, useEffect } from 'react';
import { SLIDER_CONFIG } from '../../utils/constants';
import './SliderCaptcha.css';

/**
 * @typedef {Object} SliderCaptchaProps
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
 * SliderCaptcha - 滑块验证码组件
 * 拖动滑块到目标位置完成验证
 * @param {SliderCaptchaProps} props
 * @returns {JSX.Element}
 */
export const SliderCaptcha = ({
  data,
  onSuccess,
  onRefresh,
  onError,
  status = 'ready',
  theme = 'light',
  width = SLIDER_CONFIG.DEFAULT_WIDTH,
  height = SLIDER_CONFIG.DEFAULT_HEIGHT
}) => {
  const [sliderPosition, setSliderPosition] = useState(0);
  const [isDragging, setIsDragging] = useState(false);
  const [isVerified, setIsVerified] = useState(false);
  const [error, setError] = useState(null);

  const trackRef = useRef(null);
  const startXRef = useRef(0);
  const startPositionRef = useRef(0);

  const targetX = useMemo(() => {
    if (data?.targetX !== undefined) {
      return data.targetX;
    }
    return Math.floor(width * 0.7);
  }, [data, width]);

  const handleDragStart = useCallback((e) => {
    if (isVerified || isDragging) return;

    e.preventDefault();
    setIsDragging(true);
    setError(null);

    const clientX = e.type.includes('touch')
      ? e.touches[0].clientX
      : e.clientX;

    startXRef.current = clientX;
    startPositionRef.current = sliderPosition;
  }, [isVerified, isDragging, sliderPosition]);

  const handleDrag = useCallback((e) => {
    if (!isDragging) return;

    const clientX = e.type.includes('touch')
      ? e.touches[0].clientX
      : e.clientX;

    const deltaX = clientX - startXRef.current;
    const maxPosition = width - SLIDER_CONFIG.THUMB_SIZE;
    const newPosition = Math.max(0, Math.min(
      startPositionRef.current + deltaX,
      maxPosition
    ));

    setSliderPosition(newPosition);
  }, [isDragging, width]);

  const handleDragEnd = useCallback(() => {
    if (!isDragging) return;

    setIsDragging(false);

    const tolerance = SLIDER_CONFIG.TOLERANCE;
    const accuracy = Math.abs(sliderPosition - targetX);

    if (accuracy <= tolerance) {
      setIsVerified(true);
      onSuccess?.({
        type: 'slider',
        position: sliderPosition,
        target: targetX,
        accuracy
      });
    } else {
      setError('验证失败，请重试');
      setSliderPosition(0);
      onError?.({
        position: sliderPosition,
        target: targetX,
        accuracy
      });
    }
  }, [isDragging, sliderPosition, targetX, onSuccess, onError]);

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
      setSliderPosition(0);
      setIsVerified(false);
      setError(null);
    }
  }, [status]);

  const containerClasses = useMemo(() => {
    const classes = [
      'slider-captcha',
      `slider-captcha--${theme}`
    ];

    if (isVerified) classes.push('slider-captcha--verified');
    if (isDragging) classes.push('slider-captcha--dragging');

    return classes.join(' ');
  }, [theme, isVerified, isDragging]);

  const targetImage = data?.targetImage;
  const sliderImage = data?.sliderImage;

  return (
    <div className={containerClasses} style={{ width, height }}>
      <div className="slider-captcha__image-container">
        {targetImage ? (
          <img src={targetImage} alt="Background" className="slider-captcha__bg" />
        ) : (
          <div className="slider-captcha__bg-placeholder">
            <div className="slider-captcha__puzzle-hole" style={{ left: targetX }}>
              <svg viewBox="0 0 40 40" width="40" height="40">
                <path fill="#1890ff" d="M15 10v20h-5v-5c0-2.76 2.24-5 5-5h0zm10 5c0 2.76-2.24 5-5 5v5h5V10h-5v5h0z" opacity="0.5"/>
              </svg>
            </div>
          </div>
        )}

        {isVerified && (
          <div className="slider-captcha__success-overlay">
            <svg viewBox="0 0 24 24" width="48" height="48">
              <path fill="#52c41a" d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
            </svg>
          </div>
        )}
      </div>

      <div
        ref={trackRef}
        className="slider-captcha__track"
        onMouseDown={handleDragStart}
        onTouchStart={handleDragStart}
      >
        <div
          className={`slider-captcha__thumb ${isVerified ? 'verified' : ''}`}
          style={{ left: sliderPosition }}
        >
          {isVerified ? (
            <svg viewBox="0 0 24 24" width="20" height="20">
              <path fill="currentColor" d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
            </svg>
          ) : (
            <>
              <svg viewBox="0 0 24 24" width="20" height="20">
                <path fill="currentColor" d="M15.5 5H11l5 7-5 7h4.5l5-7z"/>
              </svg>
              <svg viewBox="0 0 24 24" width="20" height="20" style={{ transform: 'scaleX(-1)' }}>
                <path fill="currentColor" d="M15.5 5H11l5 7-5 7h4.5l5-7z"/>
              </svg>
            </>
          )}
        </div>

        <span className="slider-captcha__hint">
          {isVerified ? '验证成功' : '拖动滑块完成验证'}
        </span>
      </div>

      {error && (
        <div className="slider-captcha__error">
          <span>{error}</span>
          <button onClick={onRefresh} className="slider-captcha__retry">
            重新验证
          </button>
        </div>
      )}

      {!isVerified && !error && (
        <button
          onClick={onRefresh}
          className="slider-captcha__refresh-btn"
          aria-label="刷新"
        >
          <svg viewBox="0 0 24 24" width="16" height="16">
            <path fill="currentColor" d="M17.65 6.35A7.958 7.958 0 0012 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08A5.99 5.99 0 0112 18c-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"/>
          </svg>
        </button>
      )}
    </div>
  );
};

export default SliderCaptcha;
