import { useState, useCallback, useEffect, useMemo } from 'react';
import { useCaptcha } from '../CaptchaProvider';
import { SliderCaptcha } from './captchas/SliderCaptcha';
import { ClickCaptcha } from './captchas/ClickCaptcha';
import { RotateCaptcha } from './captchas/RotateCaptcha';
import { PuzzleCaptcha } from './captchas/PuzzleCaptcha';
import { TextCaptcha } from './captchas/TextCaptcha';
import { IconCaptcha } from './captchas/IconCaptcha';
import './CaptchaDialog.css';

/**
 * @typedef {'slider' | 'click' | 'rotate' | 'puzzle' | 'text' | 'icon'} CaptchaType
 */

/**
 * @typedef {Object} CaptchaDialogProps
 * @property {boolean} visible
 * @property {CaptchaType} [type='slider']
 * @property {string} [scene='default']
 * @property {(token: string) => void} [onSuccess]
 * @property {(error: Error) => void} [onError]
 * @property {() => void} onClose
 * @property {boolean} [maskClosable=true]
 * @property {'light' | 'dark'} [theme='light']
 * @property {number} [width=350]
 */

const CAPTCHA_COMPONENTS = {
  slider: SliderCaptcha,
  click: ClickCaptcha,
  rotate: RotateCaptcha,
  puzzle: PuzzleCaptcha,
  text: TextCaptcha,
  icon: IconCaptcha
};

/**
 * CaptchaDialog - 验证码弹窗组件
 * 提供模态框形式的验证码交互
 * @param {CaptchaDialogProps} props
 * @returns {JSX.Element | null}
 */
export const CaptchaDialog = ({
  visible,
  type = 'slider',
  scene = 'default',
  onSuccess,
  onError,
  onClose,
  maskClosable = true,
  theme = 'light',
  width = 350
}) => {
  const { verify, refresh, loading } = useCaptcha();
  const [captchaData, setCaptchaData] = useState(null);
  const [status, setStatus] = useState('idle');
  const [error, setError] = useState(null);

  const handleSuccess = useCallback(async (result) => {
    setStatus('success');
    try {
      const token = await verify(scene, result);
      onSuccess?.(token);
      setTimeout(() => {
        onClose?.();
        setStatus('idle');
      }, 1500);
    } catch (err) {
      setStatus('failed');
      setError(err.message);
      onError?.(err);
    }
  }, [verify, scene, onSuccess, onError, onClose]);

  const handleRefresh = useCallback(async () => {
    setStatus('loading');
    setError(null);
    try {
      const data = await refresh(scene);
      setCaptchaData(data);
      setStatus('ready');
    } catch (err) {
      setStatus('failed');
      setError(err.message);
      onError?.(err);
    }
  }, [refresh, scene, onError]);

  useEffect(() => {
    if (visible) {
      handleRefresh();
    }
  }, [visible]);

  useEffect(() => {
    const handleEsc = (e) => {
      if (e.key === 'Escape' && visible) {
        onClose?.();
      }
    };

    if (visible) {
      document.addEventListener('keydown', handleEsc);
      document.body.style.overflow = 'hidden';
    }

    return () => {
      document.removeEventListener('keydown', handleEsc);
      document.body.style.overflow = '';
    };
  }, [visible, onClose]);

  const handleMaskClick = useCallback((e) => {
    if (maskClosable && e.target === e.currentTarget) {
      onClose?.();
    }
  }, [maskClosable, onClose]);

  const CaptchaComponent = CAPTCHA_COMPONENTS[type];

  const dialogClasses = useMemo(() => {
    return [
      'captcha-dialog',
      `captcha-dialog--${theme}`,
      visible ? 'captcha-dialog--visible' : 'captcha-dialog--hidden'
    ].join(' ');
  }, [theme, visible]);

  if (!visible && !captchaData) return null;

  return (
    <div
      className={dialogClasses}
      onClick={handleMaskClick}
      role="dialog"
      aria-modal="true"
      aria-labelledby="captcha-dialog-title"
    >
      <div className="captcha-dialog__content" style={{ width }}>
        <div className="captcha-dialog__header">
          <h3 id="captcha-dialog-title" className="captcha-dialog__title">
            安全验证
          </h3>
          <button
            type="button"
            className="captcha-dialog__close"
            onClick={onClose}
            aria-label="关闭"
          >
            <svg viewBox="0 0 24 24" width="20" height="20">
              <path
                fill="currentColor"
                d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"
              />
            </svg>
          </button>
        </div>

        <div className="captcha-dialog__body">
          {status === 'loading' && (
            <div className="captcha-dialog__loading">
              <div className="spinner"></div>
              <span>加载中...</span>
            </div>
          )}

          {status === 'failed' && error && (
            <div className="captcha-dialog__error">
              <span>{error}</span>
              <button onClick={handleRefresh}>重试</button>
            </div>
          )}

          {(status === 'ready' || status === 'success') && CaptchaComponent && (
            <CaptchaComponent
              data={captchaData}
              onSuccess={handleSuccess}
              onRefresh={handleRefresh}
              status={status}
              theme={theme}
            />
          )}
        </div>

        <div className="captcha-dialog__footer">
          <span className="captcha-dialog__tip">
            完成验证后即可继续操作
          </span>
        </div>
      </div>
    </div>
  );
};

export default CaptchaDialog;
