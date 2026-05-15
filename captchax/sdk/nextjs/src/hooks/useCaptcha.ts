'use client';

import { useState, useCallback } from 'react';
import { useCaptchaContext } from '../app/CaptchaProvider';

interface UseCaptchaOptions {
  scene?: string;
  onSuccess?: (token: string) => void;
  onError?: (error: Error) => void;
}

interface UseCaptchaReturn {
  token: string | null;
  loading: boolean;
  error: Error | null;
  verify: () => Promise<string | null>;
  reset: () => void;
}

export function useCaptcha(options: UseCaptchaOptions = {}): UseCaptchaReturn {
  const { scene = 'default', onSuccess, onError } = options;
  
  const [token, setToken] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  let captchaContext;
  try {
    captchaContext = useCaptchaContext();
  } catch {
    captchaContext = null;
  }

  const verify = useCallback(async () => {
    setLoading(true);
    setError(null);
    
    try {
      const resultToken = await captchaContext?.verify(scene);
      
      if (resultToken) {
        setToken(resultToken);
        onSuccess?.(resultToken);
        return resultToken;
      } else {
        throw new Error('Verification failed');
      }
    } catch (err) {
      const error = err instanceof Error ? err : new Error('Verification failed');
      setError(error);
      onError?.(error);
      return null;
    } finally {
      setLoading(false);
    }
  }, [scene, onSuccess, onError, captchaContext]);

  const reset = useCallback(() => {
    setToken(null);
    setError(null);
  }, []);

  return { token, loading, error, verify, reset };
}

export default useCaptcha;
