'use client';

import { useState, useCallback } from 'react';

interface UseCaptchaVerifyOptions {
  scene?: string;
  onSuccess?: (token: string) => void;
  onError?: (error: Error) => void;
  serverUrl?: string;
  apiKey?: string;
}

interface UseCaptchaVerifyReturn {
  token: string | null;
  loading: boolean;
  error: Error | null;
  verify: () => Promise<string | null>;
  reset: () => void;
  isVerified: boolean;
}

export function useCaptchaVerify(
  options: UseCaptchaVerifyOptions = {}
): UseCaptchaVerifyReturn {
  const { 
    scene = 'default', 
    onSuccess, 
    onError,
    serverUrl = 'https://api.captchax.com',
    apiKey = ''
  } = options;
  
  const [token, setToken] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [isVerified, setIsVerified] = useState(false);

  const verify = useCallback(async () => {
    if (!apiKey) {
      const error = new Error('API key is required');
      setError(error);
      onError?.(error);
      return null;
    }

    setLoading(true);
    setError(null);
    
    try {
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
      
      if (data.success) {
        setToken(data.token);
        setIsVerified(true);
        onSuccess?.(data.token);
        return data.token;
      } else {
        throw new Error(data.error || 'Verification failed');
      }
    } catch (err) {
      const error = err instanceof Error ? err : new Error('Verification failed');
      setError(error);
      onError?.(error);
      return null;
    } finally {
      setLoading(false);
    }
  }, [scene, onSuccess, onError, serverUrl, apiKey]);

  const reset = useCallback(() => {
    setToken(null);
    setError(null);
    setIsVerified(false);
  }, []);

  return { token, loading, error, verify, reset, isVerified };
}

export default useCaptchaVerify;
