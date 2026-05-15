'use client';

import { useState, useCallback } from 'react';

interface CaptchaButtonProps {
  children?: React.ReactNode;
  scene?: string;
  onSuccess?: (token: string) => void;
  onError?: (error: Error) => void;
  text?: string;
  disabled?: boolean;
  className?: string;
  style?: React.CSSProperties;
  serverUrl?: string;
  apiKey?: string;
}

async function verifyCaptchaClient(
  scene: string,
  serverUrl: string,
  apiKey: string
): Promise<string> {
  const response = await fetch(`${serverUrl}/api/v1/captcha/verify`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ 
      scene,
      apiKey,
      timestamp: Date.now()
    })
  });
  
  const data = await response.json();
  
  if (!data.success) {
    throw new Error(data.error || 'Verification failed');
  }
  
  return data.token;
}

export function CaptchaButton({
  children,
  scene = 'default',
  onSuccess,
  onError,
  text = '验证',
  disabled = false,
  className = '',
  style,
  serverUrl = 'https://api.captchax.com',
  apiKey = ''
}: CaptchaButtonProps) {
  const [loading, setLoading] = useState(false);

  const handleClick = useCallback(async () => {
    if (!apiKey) {
      const error = new Error('API key is required');
      onError?.(error);
      return;
    }

    setLoading(true);
    try {
      const token = await verifyCaptchaClient(scene, serverUrl, apiKey);
      onSuccess?.(token);
    } catch (error) {
      onError?.(error instanceof Error ? error : new Error('Verification failed'));
    } finally {
      setLoading(false);
    }
  }, [scene, onSuccess, onError, serverUrl, apiKey]);

  return (
    <button
      className={`captcha-button ${className} ${loading ? 'captcha-button-loading' : ''}`}
      onClick={handleClick}
      disabled={disabled || loading}
      style={{
        padding: '10px 20px',
        borderRadius: '6px',
        border: 'none',
        backgroundColor: '#4F46E5',
        color: 'white',
        cursor: disabled || loading ? 'not-allowed' : 'pointer',
        opacity: disabled || loading ? 0.6 : 1,
        transition: 'all 0.2s ease',
        fontSize: '14px',
        fontWeight: 500,
        ...style
      }}
    >
      {loading ? '验证中...' : (children || text)}
    </button>
  );
}

export default CaptchaButton;
