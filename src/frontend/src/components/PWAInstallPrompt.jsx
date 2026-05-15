import React, { useState, useEffect } from 'react';

const PWAInstallPrompt = ({ onInstalled, delay = 5000 }) => {
  const [deferredPrompt, setDeferredPrompt] = useState(null);
  const [showInstallPrompt, setShowInstallPrompt] = useState(false);
  const [isInstalled, setIsInstalled] = useState(false);
  const [isDismissed, setIsDismissed] = useState(false);
  const [installingStep, setInstallingStep] = useState('idle');

  useEffect(() => {
    const handleBeforeInstallPrompt = (e) => {
      e.preventDefault();
      setDeferredPrompt(e);
      setIsDismissed(false);
      
      setTimeout(() => {
        if (!isInstalled && !isDismissed) {
          setShowInstallPrompt(true);
        }
      }, delay);
    };

    const handleAppInstalled = () => {
      setShowInstallPrompt(false);
      setDeferredPrompt(null);
      setIsInstalled(true);
      setInstallingStep('idle');
      
      if (onInstalled) {
        onInstalled();
      }
    };

    window.addEventListener('beforeinstallprompt', handleBeforeInstallPrompt);
    window.addEventListener('appinstalled', handleAppInstalled);

    if (window.matchMedia('(display-mode: standalone)').matches) {
      setIsInstalled(true);
      setShowInstallPrompt(false);
    }

    if (localStorage.getItem('pwa-install-dismissed')) {
      setIsDismissed(true);
    }

    return () => {
      window.removeEventListener('beforeinstallprompt', handleBeforeInstallPrompt);
      window.removeEventListener('appinstalled', handleAppInstalled);
    };
  }, [delay, isInstalled, isDismissed, onInstalled]);

  const handleInstallClick = async () => {
    if (!deferredPrompt) return;

    setInstallingStep('installing');

    try {
      deferredPrompt.prompt();
      const { outcome } = await deferredPrompt.userChoice;
      
      if (outcome === 'accepted') {
        setShowInstallPrompt(false);
        setDeferredPrompt(null);
        setIsInstalled(true);
        setInstallingStep('success');
        localStorage.setItem('pwa-install-dismissed', 'false');
      } else {
        setInstallingStep('idle');
        setShowInstallPrompt(false);
      }
    } catch (error) {
      console.error('PWA 安装失败:', error);
      setInstallingStep('error');
      setTimeout(() => {
        setInstallingStep('idle');
      }, 3000);
    }
  };

  const handleDismiss = () => {
    setShowInstallPrompt(false);
    setIsDismissed(true);
    localStorage.setItem('pwa-install-dismissed', 'true');
    localStorage.setItem('pwa-install-dismissed-time', Date.now().toString());
  };

  const handleRemindLater = () => {
    setShowInstallPrompt(false);
    localStorage.removeItem('pwa-install-dismissed');
    localStorage.removeItem('pwa-install-dismissed-time');
    
    setTimeout(() => {
      if (!isInstalled && deferredPrompt) {
        setShowInstallPrompt(true);
      }
    }, 24 * 60 * 60 * 1000);
  };

  if (!showInstallPrompt || isInstalled) {
    return null;
  }

  const getInstallButtonText = () => {
    switch (installingStep) {
      case 'installing':
        return '安装中...';
      case 'success':
        return '安装成功!';
      case 'error':
        return '安装失败';
      default:
        return '立即安装';
    }
  };

  const isInstalling = installingStep === 'installing' || installingStep === 'success' || installingStep === 'error';

  return (
    <div style={styles.overlay}>
      <div style={styles.container}>
        <div style={styles.banner}>
          <div style={styles.iconContainer}>
            <span style={styles.icon}>📱</span>
          </div>
          <div style={styles.content}>
            <h3 style={styles.title}>安装 HJTPX 应用</h3>
            <p style={styles.description}>添加到主屏幕，随时随地快速访问</p>
            <div style={styles.features}>
              <span style={styles.feature}>⚡ 更快访问</span>
              <span style={styles.feature}>📴 离线使用</span>
              <span style={styles.feature}>🔔 推送通知</span>
            </div>
          </div>
          <button style={styles.dismissBtn} onClick={handleDismiss} aria-label="关闭">
            ✕
          </button>
        </div>
        <div style={styles.buttons}>
          <button 
            style={styles.cancelBtn} 
            onClick={handleRemindLater}
            disabled={isInstalling}
          >
            稍后提醒
          </button>
          <button 
            style={{
              ...styles.installBtn,
              ...(isInstalling ? styles.installBtnDisabled : {})
            }} 
            onClick={handleInstallClick}
            disabled={isInstalling}
          >
            {getInstallButtonText()}
          </button>
        </div>
      </div>
    </div>
  );
};

const styles = {
  overlay: {
    position: 'fixed',
    bottom: 0,
    left: 0,
    right: 0,
    backgroundColor: 'rgba(0, 0, 0, 0.5)',
    zIndex: 9999,
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'flex-end',
    padding: '0 0 20px 0'
  },
  container: {
    width: '90%',
    maxWidth: 480,
    background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
    borderRadius: 20,
    boxShadow: '0 20px 60px rgba(0, 0, 0, 0.3)',
    padding: 20,
    color: 'white',
    animation: 'slideUp 0.4s ease-out'
  },
  banner: {
    display: 'flex',
    alignItems: 'flex-start',
    marginBottom: 16,
    position: 'relative'
  },
  iconContainer: {
    marginRight: 16,
    marginTop: 4
  },
  icon: {
    fontSize: 48
  },
  content: {
    flex: 1
  },
  title: {
    margin: 0,
    fontSize: 20,
    fontWeight: 700,
    color: 'white',
    marginBottom: 4
  },
  description: {
    margin: 0,
    fontSize: 14,
    color: 'rgba(255, 255, 255, 0.9)',
    marginBottom: 8
  },
  features: {
    display: 'flex',
    gap: 8,
    flexWrap: 'wrap'
  },
  feature: {
    fontSize: 12,
    backgroundColor: 'rgba(255, 255, 255, 0.2)',
    padding: '4px 8px',
    borderRadius: 12,
    color: 'white'
  },
  dismissBtn: {
    position: 'absolute',
    top: -8,
    right: -8,
    background: 'rgba(255, 255, 255, 0.2)',
    border: 'none',
    width: 28,
    height: 28,
    borderRadius: '50%',
    cursor: 'pointer',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    fontSize: 14,
    color: 'white',
    transition: 'background-color 0.2s'
  },
  buttons: {
    display: 'flex',
    gap: 12
  },
  cancelBtn: {
    flex: 1,
    padding: '12px 20px',
    border: '2px solid rgba(255, 255, 255, 0.3)',
    background: 'transparent',
    color: 'white',
    borderRadius: 12,
    cursor: 'pointer',
    fontSize: 15,
    fontWeight: 600,
    transition: 'all 0.2s'
  },
  installBtn: {
    flex: 1,
    padding: '12px 20px',
    border: 'none',
    background: 'white',
    color: '#667eea',
    borderRadius: 12,
    cursor: 'pointer',
    fontSize: 15,
    fontWeight: 700,
    transition: 'all 0.2s',
    boxShadow: '0 4px 12px rgba(0, 0, 0, 0.2)'
  },
  installBtnDisabled: {
    opacity: 0.6,
    cursor: 'not-allowed'
  }
};

export default PWAInstallPrompt;
