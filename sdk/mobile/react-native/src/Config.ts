export interface CaptchaConfig {
  width: number;
  height: number;
  enableHapticFeedback: boolean;
  enableSoundEffect: boolean;
  sliderTrackHeight: number;
  sliderThumbSize: number;
  timeout: number;
}

export const defaultConfig: CaptchaConfig = {
  width: 320,
  height: 200,
  enableHapticFeedback: true,
  enableSoundEffect: false,
  sliderTrackHeight: 4,
  sliderThumbSize: 50,
  timeout: 30000,
};

export class ConfigManager {
  private config: CaptchaConfig;

  constructor(config: Partial<CaptchaConfig> = {}) {
    this.config = { ...defaultConfig, ...config };
  }

  getConfig(): CaptchaConfig {
    return { ...this.config };
  }

  updateConfig(updates: Partial<CaptchaConfig>): void {
    this.config = { ...this.config, ...updates };
  }

  resetConfig(): void {
    this.config = { ...defaultConfig };
  }
}

export const configManager = new ConfigManager();
