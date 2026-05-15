import { useRuntimeConfig } from '#imports';

interface CaptchaConfig {
  apiKey: string;
  apiSecret: string;
  serverUrl: string;
}

export const useCaptcha = () => {
  const config = useRuntimeConfig();
  const captchaConfig = config.captcha as CaptchaConfig;
  
  const verify = async (scene: string = 'default'): Promise<string> => {
    if (!captchaConfig?.apiKey) {
      throw new Error('CaptchaX API key is not configured');
    }
    
    if (typeof window === 'undefined') {
      throw new Error('CaptchaX verify must be called on client side');
    }
    
    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        reject(new Error('Verification timeout'));
      }, 30000);
      
      const eventSource = new EventSource(
        `${captchaConfig.serverUrl}/api/verify?scene=${scene}&apiKey=${captchaConfig.apiKey}`
      );
      
      eventSource.onmessage = (event) => {
        clearTimeout(timeout);
        eventSource.close();
        
        try {
          const data = JSON.parse(event.data);
          if (data.token) {
            resolve(data.token);
          } else if (data.error) {
            reject(new Error(data.error));
          }
        } catch (error) {
          reject(new Error('Failed to parse verification response'));
        }
      };
      
      eventSource.onerror = () => {
        clearTimeout(timeout);
        eventSource.close();
        reject(new Error('Verification connection failed'));
      };
    });
  };
  
  return { 
    verify, 
    config: captchaConfig 
  };
};
