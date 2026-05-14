import { ref, onMounted, onUnmounted, readonly } from 'vue';

interface CaptchaResult {
  token: string;
  captchaId: string;
}

interface CaptchaConfig {
  appId: string;
  serverUrl: string;
  container: HTMLElement | string;
  onReady?: () => void;
  onSuccess?: (result: CaptchaResult) => void;
  onError?: (error: Error) => void;
  onClose?: () => void;
  onRefresh?: () => void;
}

interface CaptchaXInstance {
  render: () => void;
  reset: () => void;
  destroy: () => void;
  verify: () => Promise<CaptchaResult>;
  on: (event: string, callback: (result?: CaptchaResult) => void) => void;
  off: (event: string, callback: (result?: CaptchaResult) => void) => void;
}

declare global {
  interface Window {
    CaptchaX: new (config: CaptchaConfig) => CaptchaXInstance;
  }
}

export function useCaptcha(config: {
  appId: string;
  serverUrl: string;
  container: HTMLElement | string;
  onSuccess?: (token: string) => void;
  onError?: (error: Error) => void;
}) {
  const isReady = ref(false);
  const isVerified = ref(false);
  const isLoading = ref(true);
  const error = ref<string | null>(null);
  const token = ref<string | null>(null);

  let captchaInstance: CaptchaXInstance | null = null;

  const initCaptcha = () => {
    if (typeof window === 'undefined' || !window.CaptchaX) {
      error.value = 'CaptchaX SDK 未加载';
      isLoading.value = false;
      return;
    }

    const container =
      typeof config.container === 'string'
        ? (document.querySelector(config.container) as HTMLElement)
        : config.container;

    if (!container) {
      error.value = '验证码容器未找到';
      isLoading.value = false;
      return;
    }

    try {
      captchaInstance = new window.CaptchaX({
        appId: config.appId,
        serverUrl: config.serverUrl,
        container: container,
        onReady: () => {
          console.log('验证码已就绪');
          isReady.value = true;
          isLoading.value = false;
        },
        onSuccess: (result: CaptchaResult) => {
          console.log('验证成功:', result);
          isVerified.value = true;
          token.value = result.token;
          error.value = null;
          config.onSuccess?.(result.token);
        },
        onError: (err: Error) => {
          console.error('验证失败:', err);
          error.value = err.message || '验证失败';
          isVerified.value = false;
          token.value = null;
          config.onError?.(err);
        },
        onClose: () => {
          console.log('验证码已关闭');
          isVerified.value = false;
        },
        onRefresh: () => {
          console.log('验证码已刷新');
          isVerified.value = false;
          token.value = null;
          error.value = null;
        },
      });

      captchaInstance.render();
    } catch (err) {
      console.error('初始化验证码失败:', err);
      error.value = '验证码加载失败';
      isLoading.value = false;
    }
  };

  const reset = () => {
    if (captchaInstance) {
      captchaInstance.reset();
      isVerified.value = false;
      token.value = null;
      error.value = null;
    }
  };

  const verify = async (): Promise<CaptchaResult | null> => {
    if (!captchaInstance) {
      error.value = '验证码未初始化';
      return null;
    }

    try {
      isLoading.value = true;
      const result = await captchaInstance.verify();
      return result;
    } catch (err) {
      console.error('手动验证失败:', err);
      error.value = '验证失败';
      return null;
    } finally {
      isLoading.value = false;
    }
  };

  const destroy = () => {
    if (captchaInstance) {
      captchaInstance.destroy();
      captchaInstance = null;
    }
  };

  onMounted(() => {
    initCaptcha();
  });

  onUnmounted(() => {
    destroy();
  });

  return {
    isReady: readonly(isReady),
    isVerified: readonly(isVerified),
    isLoading: readonly(isLoading),
    error: readonly(error),
    token: readonly(token),
    reset,
    verify,
    destroy,
  };
}

export default useCaptcha;
