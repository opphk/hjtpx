import React, { useState, useRef, useEffect } from 'react';
import CaptchaXService from './captchaService';
import './CaptchaX.css';

const SliderCaptcha = ({
  serviceOptions = {},
  onSuccess,
  onError,
  onRefresh,
  width = 320,
  height = 160,
}) => {
  const [captcha, setCaptcha] = useState(null);
  const [loading, setLoading] = useState(true);
  const [verifying, setVerifying] = useState(false);
  const [status, setStatus] = useState('idle');
  const [sliderX, setSliderX] = useState(0);
  const [isDragging, setIsDragging] = useState(false);
  const [dragStartX, setDragStartX] = useState(0);
  const sliderTrackRef = useRef(null);
  const serviceRef = useRef(null);

  useEffect(() => {
    serviceRef.current = new CaptchaXService(serviceOptions);
    loadCaptcha();
  }, [serviceOptions]);

  const loadCaptcha = async () => {
    setLoading(true);
    setStatus('idle');
    setSliderX(0);
    try {
      const data = await serviceRef.current.createSliderCaptcha({ width, height });
      setCaptcha(data);
    } catch (error) {
      setStatus('error');
      onError?.(error);
    } finally {
      setLoading(false);
    }
  };

  const handleRefresh = () => {
    loadCaptcha();
    onRefresh?.();
  };

  const handleMouseDown = (e) => {
    if (status !== 'idle' || loading) return;
    setIsDragging(true);
    setDragStartX(e.clientX);
  };

  const handleMouseMove = (e) => {
    if (!isDragging) return;
    const deltaX = e.clientX - dragStartX;
    const maxX = width - 50;
    const newX = Math.max(0, Math.min(maxX, deltaX));
    setSliderX(newX);
  };

  const handleMouseUp = async () => {
    if (!isDragging) return;
    setIsDragging(false);
    await verifyCaptcha();
  };

  const handleTouchStart = (e) => {
    if (status !== 'idle' || loading) return;
    setIsDragging(true);
    setDragStartX(e.touches[0].clientX);
  };

  const handleTouchMove = (e) => {
    if (!isDragging) return;
    const deltaX = e.touches[0].clientX - dragStartX;
    const maxX = width - 50;
    const newX = Math.max(0, Math.min(maxX, deltaX));
    setSliderX(newX);
  };

  const handleTouchEnd = async () => {
    if (!isDragging) return;
    setIsDragging(false);
    await verifyCaptcha();
  };

  const verifyCaptcha = async () => {
    if (!captcha) return;
    setVerifying(true);
    try {
      const result = await serviceRef.current.verifySlider(
        captcha.id,
        Math.round(sliderX),
        0
      );
      if (result.success) {
        setStatus('success');
        onSuccess?.(result);
      } else {
        setStatus('fail');
        onError?.(new Error(result.message || '验证失败'));
      }
    } catch (error) {
      setStatus('error');
      onError?.(error);
    } finally {
      setVerifying(false);
    }
  };

  useEffect(() => {
    if (isDragging) {
      window.addEventListener('mousemove', handleMouseMove);
      window.addEventListener('mouseup', handleMouseUp);
      window.addEventListener('touchmove', handleTouchMove);
      window.addEventListener('touchend', handleTouchEnd);
    } else {
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('mouseup', handleMouseUp);
      window.removeEventListener('touchmove', handleTouchMove);
      window.removeEventListener('touchend', handleTouchEnd);
    }
    return () => {
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('mouseup', handleMouseUp);
      window.removeEventListener('touchmove', handleTouchMove);
      window.removeEventListener('touchend', handleTouchEnd);
    };
  }, [isDragging]);

  const getStatusMessage = () => {
    switch (status) {
      case 'success':
        return '验证成功';
      case 'fail':
        return '验证失败，请重试';
      case 'error':
        return '网络错误，请重试';
      default:
        return '向右拖动滑块完成拼图';
    }
  };

  return (
    <div className="captcha-container" style={{ width }}>
      <div className="captcha-image-wrapper" style={{ height }}>
        {loading ? (
          <div className="captcha-loading">
            <div className="captcha-spinner"></div>
            <span>加载中...</span>
          </div>
        ) : captcha ? (
          <>
            <img
              src={`data:image/png;base64,${captcha.background_b64}`}
              alt="验证码背景"
              className="captcha-background"
            />
            <img
              src={`data:image/png;base64,${captcha.slider_b64}`}
              alt="滑块"
              className="captcha-slider-piece"
              style={{
                left: sliderX,
                top: (captcha.target_y || 0) + 'px',
              }}
            />
            {status === 'success' && (
              <div className="captcha-overlay captcha-overlay-success">
                <span className="captcha-checkmark">✓</span>
              </div>
            )}
            {status === 'fail' && (
              <div className="captcha-overlay captcha-overlay-fail">
                <span className="captcha-cross">✗</span>
              </div>
            )}
          </>
        ) : (
          <div className="captcha-error">加载失败</div>
        )}
        <button
          className="captcha-refresh-btn"
          onClick={handleRefresh}
          disabled={loading || verifying}
          title="刷新验证码"
        >
          🔄
        </button>
      </div>

      <div className="captcha-slider-wrapper">
        <div
          ref={sliderTrackRef}
          className={`captcha-slider-track ${isDragging ? 'dragging' : ''} ${status}`}
        >
          <div
            className="captcha-slider-progress"
            style={{ width: sliderX }}
          ></div>
          <span className="captcha-slider-tip">{getStatusMessage()}</span>
          <button
            className={`captcha-slider-button ${isDragging ? 'dragging' : ''} ${status}`}
            style={{ left: sliderX }}
            onMouseDown={handleMouseDown}
            onTouchStart={handleTouchStart}
            disabled={loading || verifying || status === 'success'}
          >
            {status === 'success' ? '✓' : '→'}
          </button>
        </div>
      </div>
    </div>
  );
};

export default SliderCaptcha;
