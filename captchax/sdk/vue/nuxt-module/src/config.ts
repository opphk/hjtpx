export interface CaptchaModuleConfig {
  apiKey?: string;
  apiSecret?: string;
  serverUrl?: string;
  enabled?: boolean;
}

export const captchaModuleConfig: CaptchaModuleConfig = {
  apiKey: '',
  apiSecret: '',
  serverUrl: 'https://api.captchax.com',
  enabled: true
};
