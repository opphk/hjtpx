/**
 * CaptchaX Node.js Express 后端集成示例
 */

const express = require('express');
const axios = require('axios');

const app = express();

const CAPTCHA_CONFIG = {
  appId: 'example-app',
  serverUrl: 'http://localhost:8080',
  timeout: 10000,
};

app.use(express.json());

app.use((req, res, next) => {
  res.header('Access-Control-Allow-Origin', '*');
  res.header('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS');
  res.header('Access-Control-Allow-Headers', 'Content-Type, Authorization, X-Captcha-Token');

  if (req.method === 'OPTIONS') {
    return res.sendStatus(204);
  }

  next();
});

class CaptchaXClient {
  constructor(config) {
    this.appId = config.appId;
    this.serverUrl = config.serverUrl.replace(/\/$/, '');
    this.timeout = config.timeout || 10000;
    this.client = axios.create({
      baseURL: this.serverUrl,
      timeout: this.timeout,
      headers: {
        'Content-Type': 'application/json',
      },
    });
  }

  async createSliderCaptcha(options = {}) {
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

const requireCaptcha = (req, res, next) => {
  const token = req.headers['x-captcha-token'];

  if (!token) {
    return res.status(400).json({
      success: false,
      message: '验证码 Token 不能为空',
      code: 400,
    });
  }

  try {
    const valid = captchaClient.verifyToken(token);
    if (!valid) {
      return res.status(400).json({
        success: false,
        message: '验证码验证失败',
        code: 400,
      });
    }
  } catch (error) {
    console.error('验证码验证异常:', error);
    return res.status(500).json({
      success: false,
      message: '验证码服务错误',
      code: 500,
    });
  }

  next();
};

app.get('/health', (req, res) => {
  res.json({
    status: 'healthy',
    service: 'captcha-example-backend',
  });
});

app.post('/api/captcha/create', async (req, res) => {
  const { type, width, height, charCount, clientInfo } = req.body;
  const captchaType = type || 'slider';

  try {
    let result;

    switch (captchaType) {
      case 'slider':
        result = await captchaClient.createSliderCaptcha({ width, height, clientInfo });
        break;
      case 'click':
        result = await captchaClient.createClickCaptcha({ charCount, clientInfo });
        break;
      case 'puzzle':
        result = await captchaClient.createPuzzleCaptcha({ width, height, clientInfo });
        break;
      default:
        return res.status(400).json({
          success: false,
          message: `不支持的验证码类型: ${captchaType}`,
        });
    }

    res.json({
      success: true,
      data: result,
    });
  } catch (error) {
    console.error('创建验证码失败:', error);
    res.status(500).json({
      success: false,
      message: error.message,
    });
  }
});

app.post('/api/captcha/verify', async (req, res) => {
  const { type, captchaId, targetX, targetY, clicks } = req.body;

  if (!captchaId) {
    return res.status(400).json({
      success: false,
      message: 'captcha_id 不能为空',
    });
  }

  try {
    let result;

    switch (type) {
      case 'slider':
        result = await captchaClient.verifySlider(captchaId, targetX || 0, targetY || 0);
        break;
      case 'click':
        if (!clicks || !clicks.length) {
          return res.status(400).json({
            success: false,
            message: 'clicks 不能为空',
          });
        }
        result = await captchaClient.verifyClick(captchaId, clicks);
        break;
      case 'puzzle':
        result = await captchaClient.verifyPuzzle(captchaId, targetX || 0, targetY || 0);
        break;
      default:
        return res.status(400).json({
          success: false,
          message: `不支持的验证码类型: ${type}`,
        });
    }

    res.json({
      success: true,
      data: result,
    });
  } catch (error) {
    console.error('验证失败:', error);
    res.status(500).json({
      success: false,
      message: error.message,
    });
  }
});

app.post('/api/login', requireCaptcha, (req, res) => {
  const { username, password } = req.body;

  if (!username || !password) {
    return res.status(400).json({
      success: false,
      message: '用户名和密码不能为空',
    });
  }

  if (username === 'admin' && password === 'admin123') {
    return res.json({
      success: true,
      message: '登录成功',
      token: 'mock-jwt-token',
    });
  }

  res.status(401).json({
    success: false,
    message: '用户名或密码错误',
  });
});

app.post('/api/register', requireCaptcha, (req, res) => {
  const { username, password, email } = req.body;

  if (!username || !password || !email) {
    return res.status(400).json({
      success: false,
      message: '缺少必填字段',
    });
  }

  res.json({
    success: true,
    message: '注册成功',
  });
});

const PORT = process.env.PORT || 8081;

app.listen(PORT, () => {
  console.log(`服务器启动在 http://localhost:${PORT}`);
});
