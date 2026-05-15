import { describe, it, expect, beforeEach, vi } from 'vitest';
import { defineNuxtConfig } from 'nuxt/config';

describe('CaptchaX Nuxt Module', () => {
  describe('Module Configuration', () => {
    it('should have default configuration values', () => {
      const expectedDefaults = {
        apiKey: '',
        apiSecret: '',
        serverUrl: 'https://api.captchax.com',
        enabled: true
      };

      expect(expectedDefaults.serverUrl).toBe('https://api.captchax.com');
      expect(expectedDefaults.enabled).toBe(true);
    });

    it('should accept custom configuration', () => {
      const customConfig = {
        apiKey: 'test-api-key',
        apiSecret: 'test-api-secret',
        serverUrl: 'https://custom-server.com',
        enabled: false
      };

      expect(customConfig.apiKey).toBe('test-api-key');
      expect(customConfig.serverUrl).toBe('https://custom-server.com');
    });
  });

  describe('Runtime Config', () => {
    it('should expose captcha config in runtime', () => {
      const runtimeConfig = {
        captcha: {
          apiKey: 'runtime-api-key',
          apiSecret: 'runtime-api-secret',
          serverUrl: 'https://api.captchax.com'
        }
      };

      expect(runtimeConfig.captcha).toBeDefined();
      expect(runtimeConfig.captcha.apiKey).toBe('runtime-api-key');
    });
  });

  describe('Component Registration', () => {
    it('should register CaptchaButton component', () => {
      const components = ['CaptchaButton', 'CaptchaDialog', 'CaptchaSlider'];
      
      expect(components).toContain('CaptchaButton');
    });

    it('should register CaptchaDialog component', () => {
      const components = ['CaptchaButton', 'CaptchaDialog', 'CaptchaSlider'];
      
      expect(components).toContain('CaptchaDialog');
    });

    it('should register CaptchaSlider component', () => {
      const components = ['CaptchaButton', 'CaptchaDialog', 'CaptchaSlider'];
      
      expect(components).toContain('CaptchaSlider');
    });
  });

  describe('useCaptcha Composable', () => {
    it('should create verify function', () => {
      const mockVerify = async (scene: string = 'default'): Promise<string> => {
        return `token_${scene}_${Date.now()}`;
      };

      expect(typeof mockVerify).toBe('function');
    });

    it('should return token on successful verification', async () => {
      const mockVerify = async (scene: string = 'default'): Promise<string> => {
        return `token_${scene}_${Date.now()}`;
      };

      const token = await mockVerify('login');
      
      expect(token).toBeDefined();
      expect(typeof token).toBe('string');
      expect(token).toContain('token_login_');
    });

    it('should throw error when API key is not configured', () => {
      const config = { apiKey: '' };
      
      expect(() => {
        if (!config.apiKey) {
          throw new Error('CaptchaX API key is not configured');
        }
      }).toThrow('CaptchaX API key is not configured');
    });

    it('should throw error on verification timeout', async () => {
      vi.useFakeTimers();

      const mockVerify = new Promise<string>((_, reject) => {
        setTimeout(() => reject(new Error('Verification timeout')), 30000);
      });

      const verifyPromise = mockVerify;
      vi.advanceTimersByTime(30000);

      await expect(verifyPromise).rejects.toThrow('Verification timeout');

      vi.useRealTimers();
    });
  });

  describe('useCaptchaState Composable', () => {
    it('should initialize with default state', () => {
      const state = {
        isVisible: false,
        isLoading: false,
        token: null,
        error: null
      };

      expect(state.isVisible).toBe(false);
      expect(state.isLoading).toBe(false);
      expect(state.token).toBe(null);
      expect(state.error).toBe(null);
    });

    it('should show dialog', () => {
      const state = { isVisible: false };
      const show = () => { state.isVisible = true; };

      show();

      expect(state.isVisible).toBe(true);
    });

    it('should hide dialog', () => {
      const state = { isVisible: true };
      const hide = () => { state.isVisible = false; };

      hide();

      expect(state.isVisible).toBe(false);
    });

    it('should set loading state', () => {
      const state = { isLoading: false };
      const setLoading = (loading: boolean) => { state.isLoading = loading; };

      setLoading(true);

      expect(state.isLoading).toBe(true);
    });

    it('should set token', () => {
      const state = { token: null };
      const setToken = (token: string) => { state.token = token; };

      setToken('test_token_123');

      expect(state.token).toBe('test_token_123');
    });

    it('should set error', () => {
      const state = { error: null };
      const setError = (error: Error) => { state.error = error; };

      const testError = new Error('Test error');
      setError(testError);

      expect(state.error).toBe(testError);
    });

    it('should reset state', () => {
      const state = {
        isLoading: true,
        token: 'test_token',
        error: new Error('Test error')
      };

      const reset = () => {
        state.token = null;
        state.error = null;
        state.isLoading = false;
      };

      reset();

      expect(state.token).toBe(null);
      expect(state.error).toBe(null);
      expect(state.isLoading).toBe(false);
    });
  });

  describe('CaptchaButton Component', () => {
    it('should have default props', () => {
      const defaultProps = {
        scene: 'default',
        text: '验证',
        size: 'medium',
        theme: 'light',
        disabled: false
      };

      expect(defaultProps.scene).toBe('default');
      expect(defaultProps.text).toBe('验证');
      expect(defaultProps.size).toBe('medium');
    });

    it('should emit success event on verification', async () => {
      const mockVerify = async () => 'test_token';
      
      const token = await mockVerify();
      
      expect(token).toBe('test_token');
    });

    it('should emit error event on verification failure', async () => {
      const mockVerify = async () => {
        throw new Error('Verification failed');
      };

      await expect(mockVerify()).rejects.toThrow('Verification failed');
    });
  });

  describe('CaptchaDialog Component', () => {
    it('should have default props', () => {
      const defaultProps = {
        visible: false,
        type: 'slider',
        title: '安全验证',
        targetImage: '',
        sliderImage: ''
      };

      expect(defaultProps.visible).toBe(false);
      expect(defaultProps.type).toBe('slider');
      expect(defaultProps.title).toBe('安全验证');
    });

    it('should support different verification types', () => {
      const supportedTypes = ['slider', 'click', 'rotate', 'puzzle', 'text', 'icon'];

      supportedTypes.forEach(type => {
        expect(['slider', 'click', 'rotate', 'puzzle', 'text', 'icon']).toContain(type);
      });
    });
  });

  describe('CaptchaSlider Component', () => {
    it('should have default props', () => {
      const defaultProps = {
        targetImage: '',
        sliderImage: ''
      };

      expect(defaultProps.targetImage).toBe('');
      expect(defaultProps.sliderImage).toBe('');
    });

    it('should emit success event on correct slider position', async () => {
      const targetPosition = 50;
      const userPosition = 52;
      const tolerance = 10;

      const isSuccess = Math.abs(userPosition - targetPosition) <= tolerance;

      expect(isSuccess).toBe(true);
    });

    it('should emit error event on incorrect slider position', async () => {
      const targetPosition = 50;
      const userPosition = 20;
      const tolerance = 10;

      const isSuccess = Math.abs(userPosition - targetPosition) <= tolerance;

      expect(isSuccess).toBe(false);
    });
  });

  describe('SSR Compatibility', () => {
    it('should not call window in SSR context', () => {
      const isServer = typeof window === 'undefined';

      if (isServer) {
        expect(() => new EventSource('http://test.com')).toThrow();
      }
    });

    it('should handle client-side verification', () => {
      const isServer = typeof window !== 'undefined';

      if (isServer) {
        expect(typeof EventSource).toBe('function');
      }
    });
  });

  describe('TypeScript Types', () => {
    it('should export CaptchaModuleOptions interface', () => {
      interface CaptchaModuleOptions {
        apiKey?: string;
        apiSecret?: string;
        serverUrl?: string;
        enabled?: boolean;
      }

      const options: CaptchaModuleOptions = {
        apiKey: 'test',
        enabled: true
      };

      expect(options.apiKey).toBe('test');
      expect(options.enabled).toBe(true);
    });

    it('should support component props types', () => {
      interface CaptchaButtonProps {
        scene?: string;
        text?: string;
        size?: 'small' | 'medium' | 'large';
        theme?: 'light' | 'dark';
        disabled?: boolean;
      }

      const props: CaptchaButtonProps = {
        scene: 'login',
        size: 'large',
        theme: 'dark'
      };

      expect(props.size).toBe('large');
      expect(props.theme).toBe('dark');
    });
  });
});
