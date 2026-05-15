import { createContext, useContext, useState, useCallback, useMemo } from 'react';
import axios from 'axios';

const CaptchaContext = createContext(null);

const defaultConfig = {
  apiServer: process.env.CAPTCHA_API_SERVER || 'http://localhost:8080',
  timeout: 10000,
  retryAttempts: 3,
  language: 'zh-CN'
};

export const CaptchaProvider = ({
  children,
  config = {}
}) => {
  const mergedConfig = useMemo(
    () => ({ ...defaultConfig, ...config }),
    [config]
  );

  const [captchaInstance, setCaptchaInstance] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const verify = useCallback(async (scene = 'default', options = {}) => {
    setLoading(true);
    setError(null);

    try {
      const fingerprint = await generateFingerprint();
      const token = crypto.randomUUID();

      const response = await axios.post(
        `${mergedConfig.apiServer}/api/captcha/verify`,
        {
          scene,
          token,
          fingerprint,
          ...options
        },
        {
          timeout: mergedConfig.timeout
        }
      );

      if (response.data.success) {
        return response.data.token;
      }

      throw new Error(response.data.message || 'Verification failed');
    } catch (err) {
      const errorMessage = err.response?.data?.message || err.message || 'Verification error';
      setError(errorMessage);
      throw new Error(errorMessage);
    } finally {
      setLoading(false);
    }
  }, [mergedConfig]);

  const refresh = useCallback(async (scene = 'default') => {
    setLoading(true);
    try {
      const fingerprint = await generateFingerprint();
      const response = await axios.post(
        `${mergedConfig.apiServer}/api/captcha/refresh`,
        {
          scene,
          fingerprint
        },
        {
          timeout: mergedConfig.timeout
        }
      );

      if (response.data.success) {
        setCaptchaInstance(response.data);
        return response.data;
      }
      throw new Error(response.data.message || 'Refresh failed');
    } catch (err) {
      const errorMessage = err.response?.data?.message || err.message || 'Refresh error';
      setError(errorMessage);
      throw new Error(errorMessage);
    } finally {
      setLoading(false);
    }
  }, [mergedConfig]);

  const value = useMemo(
    () => ({
      verify,
      refresh,
      config: mergedConfig,
      captchaInstance,
      setCaptchaInstance,
      loading,
      error,
      clearError: () => setError(null)
    }),
    [verify, refresh, mergedConfig, captchaInstance, loading, error]
  );

  return (
    <CaptchaContext.Provider value={value}>
      {children}
    </CaptchaContext.Provider>
  );
};

const generateFingerprint = async () => {
  const canvas = document.createElement('canvas');
  const ctx = canvas.getContext('2d');
  canvas.width = 200;
  canvas.height = 50;
  ctx.textBaseline = 'top';
  ctx.font = '14px Arial';
  ctx.fillStyle = '#f60';
  ctx.fillRect(125, 1, 62, 20);
  ctx.fillStyle = '#069';
  ctx.fillText('CaptchaX', 2, 15);
  ctx.fillStyle = 'rgba(102, 204, 0, 0.7)';
  ctx.fillText('CaptchaX', 4, 17);

  const dataUrl = canvas.toDataURL();
  const hash = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(dataUrl));
  const hashArray = Array.from(new Uint8Array(hash));
  return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
};

export const useCaptcha = () => {
  const context = useContext(CaptchaContext);
  if (!context) {
    throw new Error('useCaptcha must be used within a CaptchaProvider');
  }
  return context;
};

export default CaptchaContext;
