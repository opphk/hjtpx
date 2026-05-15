
const axios = require('axios');

const CAPTCHA_CONFIG = {
  appId: process.env.CAPTCHA_APP_ID || 'hjtpx-app',
  serverUrl: process.env.CAPTCHA_SERVER_URL || 'http://localhost:8080',
  timeout: parseInt(process.env.CAPTCHA_TIMEOUT, 10) || 10000,
  enabled: process.env.CAPTCHA_ENABLED !== 'false',
};

class CaptchaXClient {
  constructor(config) {
    this.appId = config.appId;
    this.serverUrl = config.serverUrl.replace(/\/$/, '');
    this.timeout = config.timeout || 10000;
    this.enabled = config.enabled;
    this.client = axios.create({
      baseURL: this.serverUrl,
      timeout: this.timeout,
      headers: {
        'Content-Type': 'application/json',
      },
    });
  }

  async createSliderCaptcha(options = {}) {
    if (!this.enabled) {
      return { id: 'mock-id', image: 'mock-image', targetX: 100, targetY: 0 };
    }
    const response = await this.client.post('/api/v1/captcha/slider', {
      app_id: this.appId,
      width: options.width || 200,
      height: options.height || 80,
      client_info: options.clientInfo,
    });

    const data = response.data;
    if (data.code !== 200) {
      throw new Error(data.message || '创建验证码失败');
    }

    return data.data;
  }

  async createClickCaptcha(options = {}) {
    if (!this.enabled) {
      return { id: 'mock-id', image: 'mock-image', targetChars: ['A', 'B', 'C', 'D'] };
    }
    const response = await this.client.post('/api/v1/captcha/click', {
      app_id: this.appId,
      char_count: options.charCount || 4,
      client_info: options.clientInfo,
    });

    const data = response.data;
    if (data.code !== 200) {
      throw new Error(data.message || '创建验证码失败');
    }

    return data.data;
  }

  async createPuzzleCaptcha(options = {}) {
    if (!this.enabled) {
      return { id: 'mock-id', image: 'mock-image', targetX: 150, targetY: 50 };
    }
    const response = await this.client.post('/api/v1/captcha/puzzle', {
      app_id: this.appId,
      width: options.width || 300,
      height: options.height || 150,
      client_info: options.clientInfo,
    });

    const data = response.data;
    if (data.code !== 200) {
      throw new Error(data.message || '创建验证码失败');
    }

    return data.data;
  }

  async verifySlider(captchaId, targetX, targetY = 0) {
    if (!this.enabled) {
      return { success: true, token: 'mock-valid-token' };
    }
    const response = await this.client.post('/api/v1/captcha/slider/verify', {
      captcha_id: captchaId,
      target_x: targetX,
      target_y: targetY,
    });

    const data = response.data;
    if (data.code !== 200) {
      throw new Error(data.message || '验证失败');
    }

    return data.data;
  }

  async verifyClick(captchaId, clicks) {
    if (!this.enabled) {
      return { success: true, token: 'mock-valid-token' };
    }
    const response = await this.client.post('/api/v1/captcha/click/verify', {
      captcha_id: captchaId,
      clicks: clicks,
    });

    const data = response.data;
    if (data.code !== 200) {
      throw new Error(data.message || '验证失败');
    }

    return data.data;
  }

  async verifyPuzzle(captchaId, targetX, targetY = 0) {
    if (!this.enabled) {
      return { success: true, token: 'mock-valid-token' };
    }
    const response = await this.client.post('/api/v1/captcha/puzzle/verify', {
      captcha_id: captchaId,
      target_x: targetX,
      target_y: targetY,
    });

    const data = response.data;
    if (data.code !== 200) {
      throw new Error(data.message || '验证失败');
    }

    return data.data;
  }

  verifyToken(token) {
    if (!this.enabled) {
      return true;
    }
    try {
      const parts = token.split('.');
      if (parts.length !== 3) {
        return false;
      }
      return true;
    } catch (error) {
      return false;
    }
  }
}

const captchaClient = new CaptchaXClient(CAPTCHA_CONFIG);

const createCaptcha = async (type = 'slider', options = {}) =&gt; {
  switch (type) {
    case 'slider':
      return await captchaClient.createSliderCaptcha(options);
    case 'click':
      return await captchaClient.createClickCaptcha(options);
    case 'puzzle':
      return await captchaClient.createPuzzleCaptcha(options);
    default:
      throw new Error(`不支持的验证码类型: ${type}`);
  }
};

const verifyCaptcha = async (type, params) =&gt; {
  switch (type) {
    case 'slider':
      return await captchaClient.verifySlider(params.captchaId, params.targetX, params.targetY);
    case 'click':
      return await captchaClient.verifyClick(params.captchaId, params.clicks);
    case 'puzzle':
      return await captchaClient.verifyPuzzle(params.captchaId, params.targetX, params.targetY);
    default:
      throw new Error(`不支持的验证码类型: ${type}`);
  }
};

const verifyToken = (token) =&gt; {
  return captchaClient.verifyToken(token);
};

const isEnabled = () =&gt; {
  return CAPTCHA_CONFIG.enabled;
};

module.exports = {
  captchaClient,
  createCaptcha,
  verifyCaptcha,
  verifyToken,
  isEnabled,
};

