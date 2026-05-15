class CaptchaXService {
  constructor(options = {}) {
    this.appId = options.appId || 'hjtpx-app';
    this.serverUrl = options.serverUrl || 'http://localhost:8080';
    this.timeout = options.timeout || 10000;
  }

  async request(endpoint, options = {}) {
    const url = `${this.serverUrl}${endpoint}`;
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.timeout);

    try {
      const response = await fetch(url, {
        method: options.method || 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...options.headers,
        },
        body: options.body ? JSON.stringify(options.body) : undefined,
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();
      return data;
    } catch (error) {
      clearTimeout(timeoutId);
      if (error.name === 'AbortError') {
        throw new Error('请求超时');
      }
      throw error;
    }
  }

  async createSliderCaptcha(options = {}) {
    const response = await this.request('/api/v1/captcha/slider', {
      body: {
        app_id: this.appId,
        width: options.width || 320,
        height: options.height || 160,
        client_info: options.clientInfo,
      },
    });

    if (response.code !== 200) {
      throw new Error(response.message || '获取验证码失败');
    }

    return response.data;
  }

  async verifySlider(captchaId, targetX, targetY) {
    const response = await this.request('/api/v1/captcha/slider/verify', {
      body: {
        captcha_id: captchaId,
        target_x: targetX,
        target_y: targetY,
      },
    });

    if (response.code !== 200) {
      throw new Error(response.message || '验证失败');
    }

    return response.data;
  }

  async createClickCaptcha(options = {}) {
    const response = await this.request('/api/v1/captcha/click', {
      body: {
        app_id: this.appId,
        char_count: options.charCount || 4,
        client_info: options.clientInfo,
      },
    });

    if (response.code !== 200) {
      throw new Error(response.message || '获取验证码失败');
    }

    return response.data;
  }

  async verifyClick(captchaId, clicks) {
    const response = await this.request('/api/v1/captcha/click/verify', {
      body: {
        captcha_id: captchaId,
        clicks: clicks,
      },
    });

    if (response.code !== 200) {
      throw new Error(response.message || '验证失败');
    }

    return response.data;
  }

  async createPuzzleCaptcha(options = {}) {
    const response = await this.request('/api/v1/captcha/puzzle', {
      body: {
        app_id: this.appId,
        width: options.width || 320,
        height: options.height || 160,
        client_info: options.clientInfo,
      },
    });

    if (response.code !== 200) {
      throw new Error(response.message || '获取验证码失败');
    }

    return response.data;
  }

  async verifyPuzzle(captchaId, targetX, targetY) {
    const response = await this.request('/api/v1/captcha/puzzle/verify', {
      body: {
        captcha_id: captchaId,
        target_x: targetX,
        target_y: targetY,
      },
    });

    if (response.code !== 200) {
      throw new Error(response.message || '验证失败');
    }

    return response.data;
  }
}

export default CaptchaXService;
