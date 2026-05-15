'use client';

import { useState, useRef, useCallback, useEffect } from 'react';

interface CaptchaSliderProps {
  onSuccess: (token: string) => void;
  onError?: (error: Error) => void;
  onClose?: () => void;
  scene?: string;
  serverUrl?: string;
  apiKey?: string;
  width?: number;
  height?: number;
}

interface SliderState {
  dragging: boolean;
  x: number;
  startX: number;
}

export function CaptchaSlider({
  onSuccess,
  onError,
  onClose,
  scene = 'default',
  serverUrl = 'https://api.captchax.com',
  apiKey = '',
  width = 300,
  height = 150
}: CaptchaSliderProps) {
  const [loading, setLoading] = useState(false);
  const [challenge, setChallenge] = useState<{
    backgroundImage?: string;
    sliderImage?: string;
    targetX?: number;
    targetY?: number;
  } | null>(null);
  const [sliderState, setSliderState] = useState<SliderState>({
    dragging: false,
    x: 0,
    startX: 0
  });
  const [error, setError] = useState<string | null>(null);
  
  const containerRef = useRef<HTMLDivElement>(null);
  const sliderRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    fetchChallenge();
  }, []);

  const fetchChallenge = async () => {
    if (!apiKey) {
      setError('API key is required');
      return;
    }

    setLoading(true);
    setError(null);
    try {
      const response = await fetch(`${serverUrl}/api/v1/captcha/slider`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ 
          scene,
          apiKey,
          timestamp: Date.now()
        })
      });
      const data = await response.json();
      if (data.success) {
        setChallenge(data);
      } else {
        setError(data.error || 'Failed to load challenge');
      }
    } catch (err) {
      const error = err instanceof Error ? err : new Error('Failed to load challenge');
      setError(error.message);
      onError?.(error);
    } finally {
      setLoading(false);
    }
  };

  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    setSliderState(prev => ({
      ...prev,
      dragging: true,
      startX: e.clientX
    }));
  }, []);

  const handleTouchStart = useCallback((e: React.TouchEvent) => {
    const touch = e.touches[0];
    setSliderState(prev => ({
      ...prev,
      dragging: true,
      startX: touch.clientX
    }));
  }, []);

  useEffect(() => {
    if (!sliderState.dragging) return;

    const handleMouseMove = (e: MouseEvent) => {
      const deltaX = e.clientX - sliderState.startX;
      const clampedX = Math.max(0, Math.min(deltaX, width - 40));
      setSliderState(prev => ({ ...prev, x: clampedX }));
    };

    const handleTouchMove = (e: TouchEvent) => {
      const touch = e.touches[0];
      const deltaX = touch.clientX - sliderState.startX;
      const clampedX = Math.max(0, Math.min(deltaX, width - 40));
      setSliderState(prev => ({ ...prev, x: clampedX }));
    };

    const handleEnd = async () => {
      if (!sliderState.dragging) return;
      
      setLoading(true);
      try {
        const response = await fetch(`${serverUrl}/api/v1/captcha/slider/verify`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            scene,
            apiKey,
            offsetX: sliderState.x,
            timestamp: Date.now()
          })
        });
        const data = await response.json();
        
        if (data.success) {
          onSuccess(data.token);
        } else {
          setError(data.error || 'Verification failed');
          setSliderState({ dragging: false, x: 0, startX: 0 });
          fetchChallenge();
        }
      } catch (err) {
        const error = err instanceof Error ? err : new Error('Verification failed');
        onError?.(error);
        setError(error.message);
      } finally {
        setLoading(false);
        setSliderState(prev => ({ ...prev, dragging: false }));
      }
    };

    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleEnd);
    document.addEventListener('touchmove', handleTouchMove);
    document.addEventListener('touchend', handleEnd);

    return () => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleEnd);
      document.removeEventListener('touchmove', handleTouchMove);
      document.removeEventListener('touchend', handleEnd);
    };
  }, [sliderState.dragging, sliderState.startX, width, scene, apiKey, serverUrl, onSuccess, onError]);

  return (
    <div
      ref={containerRef}
      style={{
        width: `${width}px`,
        backgroundColor: '#f9fafb',
        borderRadius: '8px',
        padding: '16px',
        boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1)'
      }}
    >
      <div
        style={{
          width: `${width}px`,
          height: `${height}px`,
          backgroundColor: '#e5e7eb',
          borderRadius: '6px',
          position: 'relative',
          overflow: 'hidden',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center'
        }}
      >
        {loading && <div style={{ color: '#6b7280' }}>加载中...</div>}
        {error && <div style={{ color: '#ef4444' }}>{error}</div>}
        {!loading && !error && challenge && (
          <>
            {challenge.backgroundImage && (
              <img
                src={challenge.backgroundImage}
                alt="background"
                style={{
                  width: '100%',
                  height: '100%',
                  objectFit: 'cover',
                  position: 'absolute'
                }}
              />
            )}
            <div
              style={{
                position: 'absolute',
                left: `${sliderState.x}px`,
                top: '50%',
                transform: 'translateY(-50%)',
                width: '40px',
                height: '40px',
                backgroundColor: 'white',
                borderRadius: '4px',
                boxShadow: '0 2px 4px rgba(0,0,0,0.2)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                cursor: sliderState.dragging ? 'grabbing' : 'grab'
              }}
              onMouseDown={handleMouseDown}
              onTouchStart={handleTouchStart}
              ref={sliderRef}
            >
              <span style={{ fontSize: '20px' }}>→</span>
            </div>
          </>
        )}
      </div>

      <div
        style={{
          marginTop: '12px',
          height: '4px',
          backgroundColor: '#d1d5db',
          borderRadius: '2px',
          overflow: 'hidden'
        }}
      >
        <div
          style={{
            height: '100%',
            width: `${(sliderState.x / (width - 40)) * 100}%`,
            backgroundColor: sliderState.dragging ? '#4F46E5' : '#d1d5db',
            transition: sliderState.dragging ? 'none' : 'width 0.2s ease'
          }}
        />
      </div>

      <p style={{ marginTop: '8px', fontSize: '12px', color: '#6b7280', textAlign: 'center' }}>
        {sliderState.dragging ? '拖动滑块完成验证' : '请拖动滑块到正确位置'}
      </p>

      {onClose && (
        <button
          onClick={onClose}
          style={{
            marginTop: '12px',
            width: '100%',
            padding: '8px',
            backgroundColor: 'white',
            border: '1px solid #d1d5db',
            borderRadius: '6px',
            cursor: 'pointer'
          }}
        >
          取消
        </button>
      )}
    </div>
  );
}

export default CaptchaSlider;
