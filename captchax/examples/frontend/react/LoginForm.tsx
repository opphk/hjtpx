import React, { useState, useRef, useEffect } from 'react';

interface CaptchaVerifyProps {
  appId: string;
  serverUrl: string;
  onVerify: (token: string) => void;
  theme?: 'light' | 'dark';
}

interface CaptchaResult {
  token: string;
  captchaId: string;
}

interface CaptchaXInstance {
  render: () => void;
  reset: () => void;
  destroy: () => void;
  verify: () => Promise<CaptchaResult>;
  on: (event: string, callback: (result?: CaptchaResult) => void) => void;
  off: (event: string, callback: (result?: CaptchaResult) => void) => void;
}

declare global {
  interface Window {
    CaptchaX: new (config: {
      appId: string;
      serverUrl: string;
      container: HTMLElement | string;
      onReady?: () => void;
      onSuccess?: (result: CaptchaResult) => void;
      onError?: (error: Error) => void;
      onClose?: () => void;
      onRefresh?: () => void;
    }) => CaptchaXInstance;
  }
}

export const CaptchaVerify: React.FC<CaptchaVerifyProps> = ({
  appId,
  serverUrl,
  onVerify,
  theme = 'light',
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const captchaRef = useRef<CaptchaXInstance | null>(null);
  const [isVerified, setIsVerified] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!containerRef.current) return;

    const container = containerRef.current;

    const handleSuccess = (result: CaptchaResult) => {
      console.log('验证成功:', result);
      setIsVerified(true);
      setError(null);
      onVerify(result.token);
    };

    const handleError = (err: Error) => {
      console.error('验证失败:', err);
      setError(err.message || '验证失败，请重试');
      setIsVerified(false);
    };

    const handleReady = () => {
      console.log('验证码已就绪');
      setIsLoading(false);
    };

    const handleClose = () => {
      console.log('验证码已关闭');
      setIsVerified(false);
    };

    const handleRefresh = () => {
      console.log('验证码已刷新');
      setIsVerified(false);
      setError(null);
    };

    try {
      if (window.CaptchaX) {
        captchaRef.current = new window.CaptchaX({
          appId,
          serverUrl,
          container,
          onReady: handleReady,
          onSuccess: handleSuccess,
          onError: handleError,
          onClose: handleClose,
          onRefresh: handleRefresh,
        });

        captchaRef.current.render();
      } else {
        setError('CaptchaX SDK 未加载');
        setIsLoading(false);
      }
    } catch (err) {
      console.error('初始化验证码失败:', err);
      setError('验证码加载失败');
      setIsLoading(false);
    }

    return () => {
      if (captchaRef.current) {
        captchaRef.current.destroy();
        captchaRef.current = null;
      }
    };
  }, [appId, serverUrl, onVerify]);

  const handleReset = () => {
    if (captchaRef.current) {
      captchaRef.current.reset();
      setIsVerified(false);
      setError(null);
    }
  };

  const handleManualVerify = async () => {
    if (!captchaRef.current) return;

    try {
      setIsLoading(true);
      const result = await captchaRef.current.verify();
      console.log('手动验证结果:', result);
      setIsVerified(true);
      onVerify(result.token);
    } catch (err) {
      console.error('手动验证失败:', err);
      setError('验证失败，请重试');
    } finally {
      setIsLoading(false);
    }
  };

  if (isVerified) {
    return (
      <div
        style={{
          padding: '16px',
          backgroundColor: theme === 'dark' ? '#1f1f1f' : '#f6ffed',
          border: `1px solid ${theme === 'dark' ? '#434343' : '#b7eb8f'}`,
          borderRadius: '8px',
          color: theme === 'dark' ? '#52c41a' : '#52c41a',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <span style={{ fontSize: '18px' }}>✓</span>
          <span>验证成功</span>
        </div>
        <button
          onClick={handleReset}
          style={{
            padding: '4px 12px',
            backgroundColor: 'transparent',
            border: `1px solid ${theme === 'dark' ? '#434343' : '#d9d9d9'}`,
            borderRadius: '4px',
            cursor: 'pointer',
            color: theme === 'dark' ? '#fff' : '#333',
          }}
        >
          重新验证
        </button>
      </div>
    );
  }

  return (
    <div
      style={{
        backgroundColor: theme === 'dark' ? '#1f1f1f' : '#fff',
        padding: '16px',
        borderRadius: '8px',
        border: `1px solid ${error ? '#ff4d4f' : theme === 'dark' ? '#434343' : '#d9d9d9'}`,
      }}
    >
      {isLoading && (
        <div
          style={{
            padding: '20px',
            textAlign: 'center',
            color: theme === 'dark' ? '#999' : '#666',
          }}
        >
          加载中...
        </div>
      )}

      {error && (
        <div
          style={{
            padding: '12px',
            marginBottom: '16px',
            backgroundColor: theme === 'dark' ? '#2a1a1a' : '#fff2f0',
            border: '1px solid #ffccc7',
            borderRadius: '4px',
            color: '#ff4d4f',
          }}
        >
          {error}
        </div>
      )}

      <div ref={containerRef} style={{ minHeight: '200px' }} />

      <div style={{ marginTop: '16px' }}>
        <button
          onClick={handleManualVerify}
          disabled={isLoading}
          style={{
            width: '100%',
            padding: '10px',
            backgroundColor: '#1890ff',
            color: '#fff',
            border: 'none',
            borderRadius: '4px',
            cursor: isLoading ? 'not-allowed' : 'pointer',
            opacity: isLoading ? 0.6 : 1,
          }}
        >
          手动验证
        </button>
      </div>
    </div>
  );
};

export default CaptchaVerify;
