import type { Component } from 'vue';

export interface CaptchaConfig {
  apiKey: string;
  apiSecret: string;
  serverUrl: string;
}

export interface CaptchaResult {
  token: string;
  expiresAt: number;
}

export interface CaptchaButtonProps {
  scene?: string;
  text?: string;
  size?: 'small' | 'medium' | 'large';
  theme?: 'light' | 'dark';
  disabled?: boolean;
}

export interface CaptchaDialogProps {
  visible: boolean;
  type?: 'slider' | 'click' | 'rotate' | 'puzzle' | 'text' | 'icon';
  title?: string;
  targetImage?: string;
  sliderImage?: string;
  onSuccess?: (token: string) => void;
  onError?: (error: Error) => void;
  onClose?: () => void;
}

export interface CaptchaSliderProps {
  targetImage?: string;
  sliderImage?: string;
  onSuccess?: (token: string) => void;
  onError?: (error: Error) => void;
}

export interface CaptchaState {
  isVisible: boolean;
  isLoading: boolean;
  token: string | null;
  error: Error | null;
}

export interface UseCaptchaReturn {
  verify: (scene?: string) => Promise<string>;
  config: CaptchaConfig;
}

export interface UseCaptchaStateReturn {
  show: () => void;
  hide: () => void;
  setLoading: (loading: boolean) => void;
  setToken: (token: string) => void;
  setError: (error: Error) => void;
  reset: () => void;
  isVisible: Readonly<import('vue').Ref<boolean>>;
  isLoading: Readonly<import('vue').Ref<boolean>>;
  token: Readonly<import('vue').Ref<string | null>>;
  error: Readonly<import('vue').Ref<Error | null>>;
}

declare module '@vue/runtime-core' {
  interface GlobalComponents {
    CaptchaButton: Component<CaptchaButtonProps>;
    CaptchaDialog: Component<CaptchaDialogProps>;
    CaptchaSlider: Component<CaptchaSliderProps>;
  }
}
