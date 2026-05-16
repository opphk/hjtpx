/**
 * 行为验证系统 JavaScript SDK
 */

class CaptchaClient {
  /**
   * 创建验证码客户端
   * @param {string} baseURL - API基础URL
   * @param {Object} options - 配置选项
   * @param {number} [options.timeout=30000] - 超时时间（毫秒）
   * @param {string} [options.apiKey] - API密钥
   */
  constructor(baseURL, options = {}) {
    this.baseURL = baseURL;
    this.timeout = options.timeout || 30000;
    this.apiKey = options.apiKey;
  }

  /**
   * 发送请求
   * @param {string} method - HTTP方法
   * @param {string} path - API路径
   * @param {Object} [data] - 请求数据
   * @param {Object} [params] - URL参数
   * @returns {Promise<Object>}
   * @private
   */
  async _request(method, path, data = null, params = null) {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.timeout);

    try {
      const url = new URL(`${this.baseURL}${path}`);
      
      if (params) {
        Object.entries(params).forEach(([key, value]) => {
          if (value !== undefined && value !== null) {
            url.searchParams.append(key, value);
          }
        });
      }

      const options = {
        method,
        signal: controller.signal,
        headers: {
          'Content-Type': 'application/json',
        },
      };

      if (this.apiKey) {
        options.headers['X-API-Key'] = this.apiKey;
      }

      if (data) {
        options.body = JSON.stringify(data);
      }

      const response = await fetch(url.toString(), options);
      clearTimeout(timeoutId);

      const result = await response.json();

      if (result.code !== 0) {
        throw new Error(result.message || 'API request failed');
      }

      return result.data;
    } catch (error) {
      clearTimeout(timeoutId);
      throw error;
    }
  }

  /**
   * 获取滑块验证码
   * @param {Object} options - 配置选项
   * @param {number} [options.width=320] - 图片宽度
   * @param {number} [options.height=160] - 图片高度
   * @param {number} [options.tolerance=8] - 容差值
   * @returns {Promise<Object>}
   */
  async getSliderCaptcha(options = {}) {
    const { width = 320, height = 160, tolerance = 8 } = options;
    return await this._request('GET', '/api/v1/captcha/slider', null, {
      width,
      height,
      tolerance,
    });
  }

  /**
   * 验证滑块验证码
   * @param {Object} data - 验证数据
   * @param {string} data.session_id - 会话ID
   * @param {number} data.x - X坐标
   * @param {number} [data.y] - Y坐标
   * @param {Array<Object>} [data.trajectory] - 轨迹数据
   * @returns {Promise<Object>}
   */
  async verifyCaptcha(data) {
    return await this._request('POST', '/api/v1/captcha/verify', data);
  }

  /**
   * 获取点击验证码
   * @returns {Promise<Object>}
   */
  async getClickCaptcha() {
    return await this._request('GET', '/api/v1/captcha/click');
  }

  /**
   * 获取手势验证码
   * @returns {Promise<Object>}
   */
  async getGestureCaptcha() {
    return await this._request('GET', '/api/v1/captcha/gesture');
  }

  /**
   * 验证手势验证码
   * @param {Object} data - 验证数据
   * @param {string} data.session_id - 会话ID
   * @param {Array<number>} data.pattern - 手势模式
   * @returns {Promise<Object>}
   */
  async verifyGestureCaptcha(data) {
    return await this._request('POST', '/api/v1/captcha/gesture/verify', data);
  }

  /**
   * 获取用户认证API
   * @returns {UserAuth}
   */
  auth() {
    return new UserAuth(this);
  }

  /**
   * 获取环境检测API
   * @returns {Environment}
   */
  env() {
    return new Environment(this);
  }

  /**
   * 记录鼠标/触摸轨迹
   * @param {Function} onTrajectory - 轨迹回调
   * @returns {Object} - 控制对象
   */
  recordTrajectory(onTrajectory) {
    let points = [];
    let startTime = Date.now();

    const handleMove = (event) => {
      const point = {
        x: event.clientX,
        y: event.clientY,
        t: Date.now() - startTime,
      };
      points.push(point);
      
      if (onTrajectory) {
        onTrajectory(points);
      }
    };

    window.addEventListener('mousemove', handleMove);
    window.addEventListener('touchmove', (e) => {
      const touch = e.touches[0];
      handleMove({ clientX: touch.clientX, clientY: touch.clientY });
    });

    return {
      getPoints: () => [...points],
      reset: () => {
        points = [];
        startTime = Date.now();
      },
      stop: () => {
        window.removeEventListener('mousemove', handleMove);
        return [...points];
      },
    };
  }
}

/**
 * 用户认证API
 */
class UserAuth {
  /**
   * @param {CaptchaClient} client - 验证码客户端
   */
  constructor(client) {
    this.client = client;
    this._token = null;
    this._refreshToken = null;
  }

  /**
   * 设置访问令牌
   * @param {string} token - 访问令牌
   */
  setToken(token) {
    this._token = token;
    this.client.apiKey = token;
  }

  /**
   * 用户注册
   * @param {Object} data - 注册数据
   * @param {string} data.username - 用户名
   * @param {string} data.email - 邮箱
   * @param {string} data.password - 密码
   * @returns {Promise<Object>}
   */
  async register(data) {
    return await this.client._request('POST', '/api/v1/auth/register', data);
  }

  /**
   * 用户登录
   * @param {Object} data - 登录数据
   * @param {string} data.username - 用户名
   * @param {string} data.password - 密码
   * @param {string} [data.captcha_token] - 验证码令牌
   * @returns {Promise<Object>}
   */
  async login(data) {
    const result = await this.client._request('POST', '/api/v1/auth/login', data);
    if (result.access_token) {
      this._token = result.access_token;
      this._refreshToken = result.refresh_token;
      this.client.apiKey = result.access_token;
    }
    return result;
  }

  /**
   * 刷新访问令牌
   * @param {string} [refreshToken] - 刷新令牌
   * @returns {Promise<Object>}
   */
  async refreshToken(refreshToken = this._refreshToken) {
    const result = await this.client._request('POST', '/api/v1/auth/refresh', {
      refresh_token: refreshToken,
    });
    if (result.access_token) {
      this._token = result.access_token;
      if (result.refresh_token) {
        this._refreshToken = result.refresh_token;
      }
      this.client.apiKey = result.access_token;
    }
    return result;
  }

  /**
   * 用户登出
   * @returns {Promise<void>}
   */
  async logout() {
    await this.client._request('POST', '/api/v1/auth/logout');
    this._token = null;
    this._refreshToken = null;
    this.client.apiKey = null;
  }

  /**
   * 验证邮箱
   * @param {string} token - 验证令牌
   * @returns {Promise<Object>}
   */
  async verifyEmail(token) {
    return await this.client._request('GET', '/api/v1/auth/verify-email', null, { token });
  }

  /**
   * 重新发送验证邮件
   * @param {string} email - 邮箱
   * @returns {Promise<Object>}
   */
  async resendVerification(email) {
    return await this.client._request('POST', '/api/v1/auth/resend-verification', { email });
  }

  /**
   * 请求重置密码
   * @param {string} email - 邮箱
   * @returns {Promise<Object>}
   */
  async requestPasswordReset(email) {
    return await this.client._request('POST', '/api/v1/auth/request-password-reset', { email });
  }

  /**
   * 重置密码
   * @param {Object} data - 重置数据
   * @param {string} data.token - 重置令牌
   * @param {string} data.new_password - 新密码
   * @returns {Promise<Object>}
   */
  async resetPassword(data) {
    return await this.client._request('POST', '/api/v1/auth/reset-password', data);
  }
}

/**
 * 环境检测API
 */
class Environment {
  /**
   * @param {CaptchaClient} client - 验证码客户端
   */
  constructor(client) {
    this.client = client;
  }

  /**
   * 获取检测脚本
   * @param {string} [callback] - 回调函数名（JSONP）
   * @returns {Promise<string>}
   */
  async getDetectionScript(callback) {
    const params = callback ? { callback } : null;
    return await this.client._request('GET', '/api/v1/detect/script', null, params);
  }

  /**
   * 注入并执行检测脚本
   * @param {string} [callback] - 回调函数名
   * @returns {Promise<Object>}
   */
  async injectDetectionScript(callback) {
    const script = await this.getDetectionScript(callback);
    
    return new Promise((resolve, reject) => {
      const scriptElement = document.createElement('script');
      scriptElement.textContent = script;
      
      if (callback) {
        window[callback] = (data) => {
          resolve(data);
          document.body.removeChild(scriptElement);
          delete window[callback];
        };
      }

      scriptElement.onerror = (error) => {
        reject(error);
        if (callback) delete window[callback];
      };

      document.body.appendChild(scriptElement);

      if (!callback) {
        setTimeout(() => resolve({}), 100);
      }
    });
  }

  /**
   * 提交检测数据
   * @param {Object} data - 检测数据
   * @returns {Promise<Object>}
   */
  async submitDetection(data) {
    return await this.client._request('POST', '/api/v1/detect/submit', data);
  }

  /**
   * 执行完整的环境检测
   * @returns {Promise<Object>}
   */
  async performFullCheck() {
    const detectionData = this.collectBrowserData();
    return await this.client._request('POST', '/api/v1/detect/check', detectionData);
  }

  /**
   * 收集浏览器指纹数据
   * @returns {Object}
   */
  collectBrowserData() {
    const data = {
      user_agent: navigator.userAgent,
      language: navigator.language,
      platform: navigator.platform,
      screen_width: screen.width,
      screen_height: screen.height,
      color_depth: screen.colorDepth,
      pixel_ratio: window.devicePixelRatio,
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      timezone_offset: new Date().getTimezoneOffset(),
    };

    // WebGL信息
    try {
      const canvas = document.createElement('canvas');
      const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
      if (gl) {
        const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
        data.webgl_vendor = debugInfo ? gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL) : '';
        data.webgl_renderer = debugInfo ? gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL) : '';
      }
    } catch (e) {}

    // 插件信息
    try {
      data.plugins = Array.from(navigator.plugins || []).map(p => p.name).join(',');
    } catch (e) {}

    // 字体检测（简单版本）
    data.fonts = this.detectFonts();

    // Canvas指纹
    data.canvas_hash = this.getCanvasFingerprint();

    // 音频指纹
    data.audio_fingerprint = this.getAudioFingerprint();

    // WebDriver检测
    data.is_webdriver = navigator.webdriver;

    return data;
  }

  /**
   * 检测常用字体
   * @returns {Array<string>}
   */
  detectFonts() {
    const baseFonts = ['monospace', 'sans-serif', 'serif'];
    const testFonts = [
      'Arial', 'Helvetica', 'Times New Roman', 'Courier New',
      'Verdana', 'Georgia', 'Comic Sans MS', 'Impact',
      'Trebuchet MS', 'Palatino', 'Garamond', 'Bookman',
    ];
    const detected = [];
    const testString = 'mmmmmmmmmmlli';
    const testSize = '72px';

    const canvas = document.createElement('canvas');
    const ctx = canvas.getContext('2d');

    baseFonts.forEach(baseFont => {
      ctx.font = `${testSize} ${baseFont}`;
      const baseWidth = ctx.measureText(testString).width;

      testFonts.forEach(testFont => {
        if (detected.includes(testFont)) return;
        
        ctx.font = `${testSize} '${testFont}', ${baseFont}`;
        const testWidth = ctx.measureText(testString).width;
        
        if (testWidth !== baseWidth) {
          detected.push(testFont);
        }
      });
    });

    return detected;
  }

  /**
   * 获取Canvas指纹
   * @returns {string}
   */
  getCanvasFingerprint() {
    try {
      const canvas = document.createElement('canvas');
      const ctx = canvas.getContext('2d');
      
      ctx.textBaseline = 'alphabetic';
      ctx.fillStyle = '#f60';
      ctx.fillRect(125, 1, 62, 20);
      ctx.fillStyle = '#069';
      ctx.font = '11pt Arial';
      ctx.fillText('Cwm fjordbank glyphs vext quiz, 😃', 2, 15);
      ctx.fillStyle = 'rgba(102, 204, 0, 0.7)';
      ctx.font = '18pt Arial';
      ctx.fillText('Cwm fjordbank glyphs vext quiz, 😃', 4, 45);
      
      return canvas.toDataURL();
    } catch (e) {
      return '';
    }
  }

  /**
   * 获取音频指纹
   * @returns {string}
   */
  getAudioFingerprint() {
    try {
      const offlineContext = new (window.OfflineAudioContext || window.webkitOfflineAudioContext)(1, 44100, 44100);
      const oscillator = offlineContext.createOscillator();
      const compressor = offlineContext.createDynamicsCompressor();
      
      oscillator.type = 'triangle';
      oscillator.frequency.setValueAtTime(10000, offlineContext.currentTime);
      compressor.threshold.setValueAtTime(-50, offlineContext.currentTime);
      compressor.knee.setValueAtTime(40, offlineContext.currentTime);
      compressor.ratio.setValueAtTime(12, offlineContext.currentTime);
      compressor.attack.setValueAtTime(0, offlineContext.currentTime);
      compressor.release.setValueAtTime(0.25, offlineContext.currentTime);
      
      oscillator.connect(compressor);
      compressor.connect(offlineContext.destination);
      oscillator.start();
      
      return 'audio_supported';
    } catch (e) {
      return 'audio_not_supported';
    }
  }
}

// 导出
if (typeof module !== 'undefined' && module.exports) {
  module.exports = { CaptchaClient, UserAuth, Environment };
} else if (typeof window !== 'undefined') {
  window.CaptchaClient = CaptchaClient;
}

// 使用示例
if (typeof document !== 'undefined') {
  /**
   * 示例：滑块验证码完整流程
   */
  async function sliderCaptchaExample() {
    const client = new CaptchaClient('http://localhost:8080');
    
    try {
      // 1. 获取验证码
      const captcha = await client.getSliderCaptcha({
        width: 320,
        height: 160,
        tolerance: 8,
      });
      
      console.log('验证码获取成功:', captcha.session_id);
      
      // 2. 记录用户滑动轨迹
      const trajectoryRecorder = client.recordTrajectory();
      
      // 3. 模拟用户滑动（实际应用中应该是用户真实操作）
      const result = await client.verifyCaptcha({
        session_id: captcha.session_id,
        x: 185,
        y: captcha.secret_y,
        trajectory: trajectoryRecorder.stop(),
      });
      
      console.log('验证结果:', result);
      
      if (result.success) {
        alert('验证成功！');
      } else {
        alert('验证失败：' + result.message);
      }
    } catch (error) {
      console.error('验证出错:', error);
    }
  }

  /**
   * 示例：用户登录流程
   */
  async function loginExample() {
    const client = new CaptchaClient('http://localhost:8080');
    const auth = client.auth();
    
    try {
      const loginResult = await auth.login({
        username: 'testuser',
        password: 'password123',
      });
      
      console.log('登录成功:', loginResult);
    } catch (error) {
      console.error('登录失败:', error);
    }
  }
}
