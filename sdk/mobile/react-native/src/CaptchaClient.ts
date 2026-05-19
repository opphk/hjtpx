export interface SliderCaptchaResult {
  sessionId: string;
  backgroundImage: string;
  sliderImage: string;
}

export interface ClickCaptchaResult {
  sessionId: string;
  backgroundImage: string;
  targetCount: number;
}

export interface VerifyResult {
  success: boolean;
  score: number;
  message: string;
}

export interface CaptchaConfig {
  baseUrl: string;
  appId: string;
  appSecret: string;
  timeout?: number;
}

class CaptchaClient {
  private baseUrl: string;
  private appId: string;
  private appSecret: string;
  private timeout: number;

  constructor(config: CaptchaConfig) {
    this.baseUrl = config.baseUrl;
    this.appId = config.appId;
    this.appSecret = config.appSecret;
    this.timeout = config.timeout || 30000;
  }

  private async fetch(url: string, body: object): Promise<any> {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.timeout);

    try {
      const response = await fetch(url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'User-Agent': 'HjtpxCaptcha-ReactNative/1.0',
        },
        body: JSON.stringify(body),
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      clearTimeout(timeoutId);
      throw error;
    }
  }

  async generateSliderCaptcha(
    width: number = 320,
    height: number = 200
  ): Promise<SliderCaptchaResult> {
    const response = await this.fetch(`${this.baseUrl}/api/captcha/slider`, {
      app_id: this.appId,
      captcha_type: 'slider',
      width,
      height,
    });

    return {
      sessionId: response.session_id,
      backgroundImage: this.baseUrl + response.background_image,
      sliderImage: this.baseUrl + response.slider_image,
    };
  }

  async verifySliderCaptcha(
    sessionId: string,
    x: number
  ): Promise<VerifyResult> {
    const response = await this.fetch(
      `${this.baseUrl}/api/captcha/verify/slider`,
      {
        session_id: sessionId,
        app_id: this.appId,
        x,
      }
    );

    return {
      success: response.success,
      score: response.score || 0,
      message: response.message || '',
    };
  }

  async generateClickCaptcha(count: number = 4): Promise<ClickCaptchaResult> {
    const response = await this.fetch(`${this.baseUrl}/api/captcha/click`, {
      app_id: this.appId,
      captcha_type: 'click',
      count,
    });

    return {
      sessionId: response.session_id,
      backgroundImage: this.baseUrl + response.background_image,
      targetCount: response.target_count,
    };
  }

  async verifyClickCaptcha(
    sessionId: string,
    xCoords: number[],
    yCoords: number[]
  ): Promise<VerifyResult> {
    const response = await this.fetch(
      `${this.baseUrl}/api/captcha/verify/click`,
      {
        session_id: sessionId,
        app_id: this.appId,
        x_coords: xCoords,
        y_coords: yCoords,
      }
    );

    return {
      success: response.success,
      score: response.score || 0,
      message: response.message || '',
    };
  }
}

export default CaptchaClient;
