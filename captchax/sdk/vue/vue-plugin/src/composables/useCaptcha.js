import { inject, readonly } from 'vue';

export const useCaptcha = () => {
  const config = inject('captchaConfig');
  
  const verify = async (scene = 'default') => {
    if (!config?.apiKey) {
      throw new Error('CaptchaX API key is not configured');
    }
    
    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        reject(new Error('Verification timeout'));
      }, 30000);
      
      const eventSource = new EventSource(`${config.serverUrl}/api/verify?scene=${scene}&apiKey=${config.apiKey}`);
      
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
    config: readonly(config)
  };
};
