import React, { useState, useEffect } from 'react';

const PWAInstallPrompt = () => {
  const [deferredPrompt, setDeferredPrompt] = useState(null);
  const [isVisible, setIsVisible] = useState(false);
  const [isDismissed, setIsDismissed] = useState(false);

  useEffect(() => {
    const isInstalled = window.matchMedia('(display-mode: standalone)').matches;
    const wasDismissed = localStorage.getItem('pwaInstallDismissed');

    if (isInstalled || wasDismissed) {
      return;
    }

    const handleBeforeInstallPrompt = (e) => {
      e.preventDefault();
      setDeferredPrompt(e);
      setIsVisible(true);
    };

    window.addEventListener('beforeinstallprompt', handleBeforeInstallPrompt);

    return () => {
      window.removeEventListener('beforeinstallprompt', handleBeforeInstallPrompt);
    };
  }, []);

  const handleInstall = async () => {
    if (!deferredPrompt) {
      return;
    }

    deferredPrompt.prompt();

    const { outcome } = await deferredPrompt.userChoice;
    console.log(`User response to the install prompt: ${outcome}`);

    setDeferredPrompt(null);
    setIsVisible(false);

    if (outcome === 'accepted') {
      localStorage.setItem('pwaInstallDismissed', 'true');
    }
  };

  const handleDismiss = () => {
    setIsVisible(false);
    setIsDismissed(true);
    localStorage.setItem('pwaInstallDismissed', 'true');
  };

  if (!isVisible) {
    return null;
  }

  return (
    <div
      style={{
        position: 'fixed',
        bottom: '20px',
        left: '50%',
        transform: 'translateX(-50%)',
        backgroundColor: '#1890ff',
        color: 'white',
        padding: '16px 24px',
        borderRadius: '12px',
        boxShadow: '0 4px 12px rgba(0, 0, 0, 0.15)',
        zIndex: 9999,
        maxWidth: '90vw',
        width: '400px',
        animation: 'slideUp 0.3s ease-out'
      }}
    >
      <style>
        {`
          @keyframes slideUp {
            from {
              transform: translateX(-50%) translateY(100%);
              opacity: 0;
            }
            to {
              transform: translateX(-50%) translateY(0);
              opacity: 1;
            }
          }
        `}
      </style>
      <div style={{ display: 'flex', alignItems: 'flex-start', gap: '12px' }}>
        <div style={{ flex: 1 }}>
          <h3 style={{ margin: '0 0 8px 0', fontSize: '18px', fontWeight: '600' }}>
            安装 HJTPX 应用
          </h3>
          <p style={{ margin: '0 0 16px 0', fontSize: '14px', lineHeight: '1.5', opacity: 0.95 }}>
            添加到主屏幕以获得更好的移动端体验和更快的访问速度
          </p>
          <div style={{ display: 'flex', gap: '8px' }}>
            <button
              onClick={handleInstall}
              style={{
                flex: 1,
                padding: '10px 20px',
                backgroundColor: 'white',
                color: '#1890ff',
                border: 'none',
                borderRadius: '8px',
                fontSize: '14px',
                fontWeight: '600',
                cursor: 'pointer',
                transition: 'transform 0.2s'
              }}
              onMouseOver={(e) => (e.target.style.transform = 'scale(1.02)')}
              onMouseOut={(e) => (e.target.style.transform = 'scale(1)')}
            >
              安装
            </button>
            <button
              onClick={handleDismiss}
              style={{
                padding: '10px 20px',
                backgroundColor: 'transparent',
                color: 'white',
                border: '1px solid rgba(255, 255, 255, 0.5)',
                borderRadius: '8px',
                fontSize: '14px',
                cursor: 'pointer',
                transition: 'background-color 0.2s'
              }}
              onMouseOver={(e) => (e.target.style.backgroundColor = 'rgba(255, 255, 255, 0.1)')}
              onMouseOut={(e) => (e.target.style.backgroundColor = 'transparent')}
            >
              稍后
            </button>
          </div>
        </div>
        <button
          onClick={handleDismiss}
          style={{
            backgroundColor: 'transparent',
            border: 'none',
            color: 'white',
            cursor: 'pointer',
            padding: '4px',
            fontSize: '20px',
            lineHeight: 1,
            opacity: 0.8,
            transition: 'opacity 0.2s'
          }}
          onMouseOver={(e) => (e.target.style.opacity = 1)}
          onMouseOut={(e) => (e.target.style.opacity = 0.8)}
        >
          ×
        </button>
      </div>
    </div>
  );
};

export default PWAInstallPrompt;
