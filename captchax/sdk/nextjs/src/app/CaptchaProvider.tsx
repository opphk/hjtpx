'use client';

import { createContext, useContext, useState, useCallback, ReactNode } from 'react';

interface CaptchaContextValue {
  verify: (scene?: string) => Promise<string>;
  getChallenge: (scene?: string) => Promise<unknown>;
  config: {
    apiKey: string;
    serverUrl: string;
  };
}

const CaptchaContext = createContext<CaptchaContextValue | null>(null);

interface CaptchaProviderProps {
  children: ReactNode;
  apiKey: string;
  serverUrl?: string;
}

export function CaptchaProvider({ 
  children, 
  apiKey,
  serverUrl = 'https://api.captchax.com'
}: CaptchaProviderProps) {
  const [tokenCache, setTokenCache] = useState<Map<string, string>>(new Map());

  const verify = useCallback(async (scene: string = 'default'): Promise<string> => {
    try {
      const response = await fetch(`${serverUrl}/api/v1/captcha/${scene}/verify`, {
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
      
      const token = data.token;
      setTokenCache(prev => new Map(prev).set(scene, token));
      return token;
    } catch (error) {
      throw error instanceof Error ? error : new Error('Verification failed');
    }
  }, [apiKey, serverUrl]);

  const getChallenge = useCallback(async (scene: string = 'default'): Promise<unknown> => {
    try {
      const response = await fetch(`${serverUrl}/api/v1/captcha/challenge`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ 
          scene,
          apiKey,
          timestamp: Date.now()
        })
      });
      
      return await response.json();
    } catch (error) {
      throw error instanceof Error ? error : new Error('Failed to get challenge');
    }
  }, [apiKey, serverUrl]);

  return (
    <CaptchaContext.Provider value={{ verify, getChallenge, config: { apiKey, serverUrl } }}>
      {children}
    </CaptchaContext.Provider>
  );
}

export function useCaptchaContext(): CaptchaContextValue {
  const context = useContext(CaptchaContext);
  if (!context) {
    throw new Error('useCaptchaContext must be used within CaptchaProvider');
  }
  return context;
}

export { CaptchaContext };
export type { CaptchaProviderProps };
