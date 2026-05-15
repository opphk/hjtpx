import { useState, useCallback, useMemo, useRef, useEffect } from 'react';
import './PuzzleCaptcha.css';

/**
 * @typedef {Object} PuzzleCaptchaProps
 * @property {Object} [data]
 * @property {Function} [onSuccess]
 * @property {Function} [onRefresh]
 * @property {Function} [onError]
 * @property {string} [status='ready']
 * @property {'light' | 'dark'} [theme='light']
 * @property {number} [width=300]
 */

/**
 * PuzzleCaptcha - 拼图验证码组件
 * 将滑块拼入正确位置完成验证
 * @param {PuzzleCaptchaProps} props
 * @returns {JSX.Element}
 */
export const PuzzleCaptcha = ({
  data,
  onSuccess,
  onRefresh,
  onError,
  status = 'ready',
  theme = 'light',
  width = 300
}) => {
  const [sliderPosition, setSliderPosition] = useState(0);
  const [isDragging, setIsDragging] = useState(false);
  const [isVerified, setIsVerified] = useState(false);
  const [error, setError] = useState(null);

  const containerRef = useRef(null);
  const startXRef = useRef(0);
  const startPositionRef = useRef(0);

  const targetX = data?.targetX ?? Math.floor(width * 0.7);
  const puzzleSize = 40;
  const tolerance = 5;

  const handleDragStart = useCallback((e) => {
    if (isVerified) return;

    e.preventDefault();
    setIsDragging(true);

    const clientX = e.type.includes('touch')
      ? e.touches[0].clientX
      : e.clientX;

    startXRef.current = clientX;
    startPositionRef.current = sliderPosition;
  }, [isVerified, sliderPosition]);

  const handleDrag = useCallback((e) => {
    if (!isDragging) return;

    const clientX = e.type.includes('touch')
      ? e.touches[0].clientX
      : e.clientX;

    const deltaX = clientX - startXRef.current;
    const maxPosition = width - puzzleSize - 10;
    const newPosition = Math.max(10, Math.min(
      startPositionRef.current + deltaX,
      maxPosition
    ));

    setSliderPosition(newPosition);
  }, [isDragging, width]);

  const handleDragEnd = useCallback(() => {
    if (!isDragging) return;

    setIsDragging(false);

    const accuracy = Math.abs(sliderPosition - targetX);

    if (accuracy <= tolerance) {
      setIsVerified(true);
      onSuccess?.({
        type: 'puzzle',
        position: sliderPosition,
        target: targetX,
        accuracy
      });
    } else {
      setError('位置不正确，请重试');
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
      'puzzle-captcha',
      `puzzle-captcha--${theme}`
    ];

    if (isVerified) classes.push('puzzle-captcha--verified');

    return classes.join(' ');
  }, [theme, isVerified]);

  return (
    <div className={containerClasses} style={{ width }}>
      <div className="puzzle-captcha__header">
        <span className="puzzle-captcha__hint">
          将拼图移动到正确位置
        </span>
      </div>

      <div ref={containerRef} className="puzzle-captcha__container">
        <div className="puzzle-captcha__background">
          {data?.backgroundImage ? (
            <img src={data.backgroundImage} alt="Background" />
          ) : (
            <div className="puzzle-captcha__bg-placeholder">
              <div className="puzzle-captcha__scene">
                <div className="puzzle-captcha__mountain"/>
                <div className="puzzle-captcha__sun"/>
              </div>
            </div>
          )}
        </div>

        <div className="puzzle-captcha__hole" style={{ left: targetX }}>
          <svg viewBox="0 0 40 40" width="40" height="40">
            <path
              fill="#ffffff"
              stroke="#d9d9d9"
              strokeWidth="2"
              d="M10 5h20v20h-5c0-8-5-15-15-15v-5zm20 0v5c0 8-5 15-15 15h-5V5z"
            />
          </svg>
        </div>

        <div
          className="puzzle-captcha__piece"
          style={{
            left: isVerified ? targetX : sliderPosition,
            transition: isDragging ? 'none' : 'left 0.3s ease'
          }}
        >
          {data?.pieceImage ? (
            <img src={data.pieceImage} alt="Puzzle piece" />
          ) : (
            <div className="puzzle-captcha__piece-placeholder">
              <svg viewBox="0 0 40 40" width="40" height="40">
                <path
                  fill="#1890ff"
                  stroke="#ffffff"
                  strokeWidth="2"
                  d="M10 5h20v20h-5c0-8-5-15-15-15v-5zm20 0v5c0 8-5 15-15 15h-5V5z"
                />
              </svg>
            </div>
          )}
        </div>

        {isVerified && (
          <div className="puzzle-captcha__success-overlay">
            <svg viewBox="0 0 24 24" width="48" height="48">
              <path fill="#52c41a" d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
            </svg>
          </div>
        )}
      </div>

      <div
        className={`puzzle-captcha__slider ${isDragging ? 'dragging' : ''}`}
        onMouseDown={handleDragStart}
        onTouchStart={handleDragStart}
      >
        <div className="puzzle-captcha__slider-track">
          <div
            className="puzzle-captcha__slider-thumb"
            style={{ left: sliderPosition }}
          >
            <svg viewBox="0 0 24 24" width="20" height="20">
              <path fill="currentColor" d="M15.5 5H11l5 7-5 7h4.5l5-7z"/>
              <path fill="currentColor" d="M8.5 5H4l5 7-5 7h4.5l5-7z"/>
            </svg>
          </div>
        </div>
        <span className="puzzle-captcha__hint-text">
          {isVerified ? '验证成功' : '拖动滑块'}
        </span>
      </div>

      {error && (
        <div className="puzzle-captcha__error">
          <span>{error}</span>
          <button onClick={() => {
            setSliderPosition(0);
            setError(null);
          }}>重新验证</button>
        </div>
      )}
    </div>
  );
};

export default PuzzleCaptcha;
