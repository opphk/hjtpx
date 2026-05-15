'use client';

import { useState, useEffect, useRef, ReactNode } from 'react';

interface CaptchaDialogProps {
  open: boolean;
  onClose: () => void;
  onSuccess: (token: string) => void;
  onError?: (error: Error) => void;
  scene?: string;
  type?: 'slider' | 'click' | 'puzzle' | 'rotate' | 'text' | 'icon';
  serverUrl?: string;
  apiKey?: string;
  title?: string;
  width?: number | string;
  height?: number | string;
}

export function CaptchaDialog({
  open,
  onClose,
  onSuccess,
  onError,
  scene = 'default',
  type = 'slider',
  serverUrl = 'https://api.captchax.com',
  apiKey = '',
  title = '安全验证',
  width = 400,
  height = 300
}: CaptchaDialogProps) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [challenge, setChallenge] = useState<unknown>(null);
  const dialogRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (open && !challenge) {
      fetchChallenge();
    }
  }, [open]);

  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && open) {
        onClose();
      }
    };
    document.addEventListener('keydown', handleEscape);
    return () => document.removeEventListener('keydown', handleEscape);
  }, [open, onClose]);

  const fetchChallenge = async () => {
    if (!apiKey) {
      setError('API key is required');
      return;
    }

    setLoading(true);
    setError(null);
    try {
      const response = await fetch(`${serverUrl}/api/v1/captcha/${type}`, {
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
      setError(err instanceof Error ? err.message : 'Failed to load challenge');
      onError?.(err instanceof Error ? err : new Error('Failed to load challenge'));
    } finally {
      setLoading(false);
    }
  };

  const handleVerify = async (verifyData: unknown) => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch(`${serverUrl}/api/v1/captcha/${type}/verify`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          scene,
          apiKey,
          ...verifyData,
          timestamp: Date.now()
        })
      });
      const data = await response.json();
      if (data.success) {
        onSuccess(data.token);
        onClose();
      } else {
        setError(data.error || 'Verification failed');
        setChallenge(null);
        fetchChallenge();
      }
    } catch (err) {
      const error = err instanceof Error ? err : new Error('Verification failed');
      setError(error.message);
      onError?.(error);
    } finally {
      setLoading(false);
    }
  };

  if (!open) return null;

  return (
    <div 
      style={{
        position: 'fixed',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        backgroundColor: 'rgba(0, 0, 0, 0.5)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 1000
      }}
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div
        ref={dialogRef}
        style={{
          backgroundColor: 'white',
          borderRadius: '12px',
          boxShadow: '0 20px 25px -5px rgba(0, 0, 0, 0.1)',
          width: typeof width === 'number' ? `${width}px` : width,
          height: typeof height === 'number' ? `${height}px` : height,
          maxWidth: '90vw',
          maxHeight: '90vh',
          overflow: 'hidden',
          display: 'flex',
          flexDirection: 'column'
        }}
      >
        <div style={{
          padding: '16px 20px',
          borderBottom: '1px solid #e5e7eb',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center'
        }}>
          <h3 style={{ margin: 0, fontSize: '18px', fontWeight: 600 }}>{title}</h3>
          <button
            onClick={onClose}
            style={{
              background: 'none',
              border: 'none',
              fontSize: '24px',
              cursor: 'pointer',
              color: '#6b7280',
              padding: '0',
              lineHeight: 1
            }}
          >
            ×
          </button>
        </div>
        
        <div style={{ flex: 1, padding: '20px', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          {loading && <div>加载中...</div>}
          {error && <div style={{ color: '#ef4444', textAlign: 'center' }}>{error}</div>}
          {!loading && !error && challenge && (
            <div style={{ width: '100%', textAlign: 'center' }}>
              <p>验证类型: {type}</p>
              <p>场景: {scene}</p>
              <button
                onClick={() => handleVerify({})}
                style={{
                  marginTop: '20px',
                  padding: '10px 30px',
                  backgroundColor: '#4F46E5',
                  color: 'white',
                  border: 'none',
                  borderRadius: '6px',
                  cursor: 'pointer'
                }}
              >
                模拟验证
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default CaptchaDialog;
