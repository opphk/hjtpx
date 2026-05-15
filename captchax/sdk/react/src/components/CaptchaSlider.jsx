import { useState, useCallback, useMemo, useRef, useEffect } from 'react';
import './CaptchaSlider.css';

/**
 * @typedef {Object} CaptchaSliderProps
 * @property {number} [width=300]
 * @property {number} [height=150]
 * @property {string} [targetImage]
 * @property {string} [sliderImage]
 * @property {number} [targetPosition=0.5]
 * @property {Function} [onSuccess]
 * @property {Function} [onError]
 * @property {Function} [onChange]
 * @property {'light' | 'dark'} [theme='light']
 */

/**
 * CaptchaSlider - 滑块验证码组件
 * 提供拖动滑块完成验证的功能
 * @param {CaptchaSliderProps} props
 * @returns {JSX.Element}
 */
export const CaptchaSlider = ({
  width = 300,
  height = 150,
  targetImage,
  sliderImage,
  targetPosition = 0.5,
  onSuccess,
  onError,
  onChange,
  theme = 'light'
}) => {
  const [sliderPosition, setSliderPosition] = useState(0);
  const [isDragging, setIsDragging] = useState(false);
  const [isVerified, setIsVerified] = useState(false);
  const trackRef = useRef(null);
  const startXRef = useRef(0);
  const currentXRef = useRef(0);

  const targetX = useMemo(() => {
    return Math.floor(width * targetPosition);
  }, [width, targetPosition]);

  const handleDragStart = useCallback((e) => {
    if (isVerified || isDragging) return;

    e.preventDefault();
    setIsDragging(true);
    startXRef.current = e.type.includes('touch')
      ? e.touches[0].clientX
      : e.clientX;
    currentXRef.current = sliderPosition;
  }, [isVerified, isDragging, sliderPosition]);

  const handleDrag = useCallback((e) => {
    if (!isDragging) return;

    const clientX = e.type.includes('touch')
      ? e.touches[0].clientX
      : e.clientX;

    const deltaX = clientX - startXRef.current;
    const newPosition = Math.max(0, Math.min(
      currentXRef.current + deltaX,
      width - 40
    ));

    setSliderPosition(newPosition);
    onChange?.(newPosition);
  }, [isDragging, width, onChange]);

  const handleDragEnd = useCallback(() => {
    if (!isDragging) return;

    setIsDragging(false);

    const tolerance = 5;
    const isCorrect = Math.abs(sliderPosition - targetX) <= tolerance;

    if (isCorrect) {
      setIsVerified(true);
      onSuccess?.({
        position: sliderPosition,
        target: targetX,
        accuracy: Math.abs(sliderPosition - targetX)
      });
    } else {
      setSliderPosition(0);
      onError?.({
        position: sliderPosition,
        target: targetX,
        accuracy: Math.abs(sliderPosition - targetX)
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

  const reset = useCallback(() => {
    setSliderPosition(0);
    setIsVerified(false);
  }, []);

  const containerClasses = useMemo(() => {
    const classes = [
      'captcha-slider',
      `captcha-slider--${theme}`
    ];

    if (isVerified) classes.push('captcha-slider--verified');
    if (isDragging) classes.push('captcha-slider--dragging');

    return classes.join(' ');
  }, [theme, isVerified, isDragging]);

  return (
    <div
      className={containerClasses}
      style={{ width, height }}
      ref={trackRef}
    >
      <div className="captcha-slider__background">
        {targetImage ? (
          <img src={targetImage} alt="Captcha background" />
        ) : (
          <div className="captcha-slider__placeholder">
            拖动滑块完成验证
          </div>
        )}
        <div
          className="captcha-slider__target"
          style={{ left: targetX }}
        >
          {sliderImage ? (
            <img src={sliderImage} alt="Slider" />
          ) : (
            <svg viewBox="0 0 40 40" width="40" height="40">
              <path
                fill="#1890ff"
                d="M30.59 4.87l-1.42-1.42-6.37 6.37-6.37-6.37-1.42 1.42 6.37 6.37-6.37 6.37 1.42 1.42 6.37-6.37 6.37 6.37 1.42-1.42-6.37-6.37z"
                transform="rotate(45 20 20)"
              />
            </svg>
          )}
        </div>
      </div>

      <div
        className={`captcha-slider__track ${isDragging ? 'active' : ''}`}
        onMouseDown={handleDragStart}
        onTouchStart={handleDragStart}
      >
        <div
          className={`captcha-slider__thumb ${
            isVerified ? 'verified' : ''
          } ${isDragging ? 'dragging' : ''}`}
          style={{ left: sliderPosition }}
        >
          {isVerified ? (
            <svg viewBox="0 0 24 24" width="20" height="20">
              <path fill="currentColor" d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
            </svg>
          ) : (
            <svg viewBox="0 0 24 24" width="20" height="20">
              <path
                fill="currentColor"
                d="M15.5 5H11l5 7-5 7h4.5l5-7z"
              />
              <path
                fill="currentColor"
                d="M8.5 5H4l5 7-5 7h4.5l5-7z"
              />
            </svg>
          )}
        </div>
        <div className="captcha-slider__track-bg">
          {!isVerified && (
            <span className="captcha-slider__hint">
              拖动滑块完成验证
            </span>
          )}
        </div>
      </div>

      {isVerified && (
        <div className="captcha-slider__success">
          验证成功
        </div>
      )}
    </div>
  );
};

export default CaptchaSlider;
