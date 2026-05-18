/**
 * 行为验证系统 JavaScript SDK - 浏览器端完整版
 *
 * 提供浏览器环境下完整的验证码功能，支持多种集成模式
 */

class CaptchaClient {
  /**
   * 创建验证码客户端
   * @param {string} baseURL - API基础URL
   * @param {Object} options - 配置选项
   */
  constructor(baseURL, options = {}) {
    this.baseURL = baseURL;
    this.timeout = options.timeout || 30000;
    this.apiKey = options.apiKey;
    this.retryCount = options.retryCount || 3;
    this.retryDelay = options.retryDelay || 1000;
    this._token = null;
  }

  /**
   * 发送请求
   * @private
   */
  async _request(method, path, data = null, params = null) {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.timeout);

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

    if (this._token) {
      options.headers['Authorization'] = `Bearer ${this._token}`;
    }

    if (data) {
      options.body = JSON.stringify(data);
    }

    try {
      const response = await fetch(url.toString(), options);
      clearTimeout(timeoutId);

      if (!response.ok) {
        throw this._createError(response.status, response.statusText);
      }

      const result = await response.json();

      if (result.code !== 0) {
        throw new Error(result.message || 'API request failed');
      }

      return result.data;
    } catch (error) {
      clearTimeout(timeoutId);

      if (error.name === 'AbortError') {
        throw new Error('Request timeout');
      }

      throw error;
    }
  }

  /**
   * 创建错误对象
   * @private
   */
  _createError(status, message) {
    const error = new Error(message || 'Request failed');
    error.status = status;
    return error;
  }

  /**
   * 获取滑块验证码
   * @param {Object} options - 配置选项
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
   * @returns {Promise<Object>}
   */
  async verifySliderCaptcha(data) {
    return await this._request('POST', '/api/v1/captcha/verify', {
      type: 'slider',
      ...data,
    });
  }

  /**
   * 获取点击验证码
   * @param {Object} options - 配置选项
   * @returns {Promise<Object>}
   */
  async getClickCaptcha(options = {}) {
    const { mode = 'number', shuffle = true, points = 3 } = options;
    return await this._request('GET', '/api/v1/captcha/click', null, {
      mode,
      shuffle: shuffle.toString(),
      points: points.toString(),
    });
  }

  /**
   * 验证点击验证码
   * @param {Object} data - 验证数据
   * @returns {Promise<Object>}
   */
  async verifyClickCaptcha(data) {
    return await this._request('POST', '/api/v1/captcha/verify', {
      type: 'click',
      ...data,
    });
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
   * @returns {Promise<Object>}
   */
  async verifyGestureCaptcha(data) {
    return await this._request('POST', '/api/v1/captcha/gesture/verify', data);
  }

  /**
   * 获取图形验证码
   * @param {Object} options - 配置选项
   * @returns {Promise<Object>}
   */
  async getImageCaptcha(options = {}) {
    const { type = 'mixed', count = 4 } = options;
    return await this._request('GET', '/api/v1/captcha/image', null, {
      type,
      count: count.toString(),
    });
  }

  /**
   * 验证图形验证码
   * @param {string} challengeId - 挑战ID
   * @param {string} answer - 用户答案
   * @returns {Promise<Object>}
   */
  async verifyImageCaptcha(challengeId, answer) {
    return await this._request('POST', '/api/v1/captcha/image/verify', {
      challenge_id: challengeId,
      answer,
    });
  }

  /**
   * 通用验证方法
   * @param {Object} data - 验证数据
   * @returns {Promise<Object>}
   */
  async verifyCaptcha(data) {
    return await this._request('POST', '/api/v1/captcha/verify', data);
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
   * 设置访问令牌
   * @param {string} token - 访问令牌
   */
  setToken(token) {
    this._token = token;
  }

  /**
   * 记录鼠标/触摸轨迹
   * @param {Function} onTrajectory - 轨迹回调
   * @param {HTMLElement} element - 监听元素，默认document.body
   * @returns {Object} - 控制对象
   */
  recordTrajectory(onTrajectory, element = document.body) {
    let points = [];
    let startTime = Date.now();
    let isRecording = false;

    const handleMove = (event) => {
      if (!isRecording) return;

      const clientX = event.touches ? event.touches[0].clientX : event.clientX;
      const clientY = event.touches ? event.touches[0].clientY : event.clientY;

      const point = {
        x: clientX,
        y: clientY,
        t: Date.now() - startTime,
      };
      points.push(point);

      if (onTrajectory) {
        onTrajectory([...points]);
      }
    };

    const startRecording = () => {
      isRecording = true;
      startTime = Date.now();
      points = [];
    };

    const stopRecording = () => {
      isRecording = false;
      return [...points];
    };

    element.addEventListener('mousemove', handleMove);
    element.addEventListener('touchmove', handleMove, { passive: true });

    return {
      getPoints: () => [...points],
      start: startRecording,
      stop: stopRecording,
      reset: () => {
        points = [];
        startTime = Date.now();
      },
      isRecording: () => isRecording,
      destroy: () => {
        element.removeEventListener('mousemove', handleMove);
        element.removeEventListener('touchmove', handleMove);
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
    this.client.setToken(token);
  }

  /**
   * 用户注册
   * @param {Object} data - 注册数据
   * @returns {Promise<Object>}
   */
  async register(data) {
    return await this.client._request('POST', '/api/v1/auth/register', data);
  }

  /**
   * 用户登录
   * @param {Object} data - 登录数据
   * @returns {Promise<Object>}
   */
  async login(data) {
    const result = await this.client._request('POST', '/api/v1/auth/login', data);
    if (result.access_token) {
      this._token = result.access_token;
      this._refreshToken = result.refresh_token;
      this.client.setToken(result.access_token);
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
      this.client.setToken(result.access_token);
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
    this.client.setToken(null);
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
   * 请求重置密码
   * @param {string} email - 邮箱
   * @returns {Promise<Object>}
   */
  async requestPasswordReset(email) {
    return await this.client._request('POST', '/api/v1/auth/request-password-reset', { email });
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

    try {
      const canvas = document.createElement('canvas');
      const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
      if (gl) {
        const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
        data.webgl_vendor = debugInfo ? gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL) : '';
        data.webgl_renderer = debugInfo ? gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL) : '';
      }
    } catch (e) {}

    try {
      data.plugins = Array.from(navigator.plugins || []).map(p => p.name).join(',');
    } catch (e) {}

    data.fonts = this.detectFonts();
    data.canvas_hash = this.getCanvasFingerprint();
    data.audio_fingerprint = this.getAudioFingerprint();
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

/**
 * UI组件：滑块验证码
 */
class SliderCaptchaWidget {
  /**
   * 创建滑块验证码组件
   * @param {HTMLElement} container - 容器元素
   * @param {CaptchaClient} client - 验证码客户端
   * @param {Object} options - 配置选项
   */
  constructor(container, client, options = {}) {
    this.container = container;
    this.client = client;
    this.options = options;
    this.sessionId = null;
    this.secretY = null;
    this.isVerified = false;

    this._init();
  }

  async _init() {
    try {
      const captcha = await this.client.getSliderCaptcha({
        width: this.options.width || 320,
        height: this.options.height || 160,
        tolerance: this.options.tolerance || 8,
      });

      this.sessionId = captcha.session_id;
      this.secretY = captcha.secret_y;

      this._render(captcha);
      this._bindEvents();
    } catch (error) {
      this.container.innerHTML = `<div class="captcha-error">加载验证码失败: ${error.message}</div>`;
    }
  }

  _render(captcha) {
    this.container.innerHTML = `
      <div class="captcha-slider-widget">
        <div class="captcha-image-container">
          <img src="${captcha.image_url}" alt="验证码背景" class="captcha-bg" />
          <div class="captcha-slider-track">
            <div class="captcha-slider-thumb"></div>
          </div>
        </div>
        <div class="captcha-controls">
          <span class="captcha-hint">拖动滑块完成拼图</span>
          <button class="captcha-refresh-btn">刷新</button>
        </div>
      </div>
    `;

    this.sliderElement = this.container.querySelector('.captcha-slider-thumb');
    this.trackElement = this.container.querySelector('.captcha-slider-track');
    this.refreshBtn = this.container.querySelector('.captcha-refresh-btn');
  }

  _bindEvents() {
    let isDragging = false;
    let startX = 0;
    let currentX = 0;

    const handleStart = (e) => {
      if (this.isVerified) return;
      isDragging = true;
      startX = e.type === 'touchstart' ? e.touches[0].clientX : e.clientX;
      this.sliderElement.classList.add('dragging');
    };

    const handleMove = (e) => {
      if (!isDragging) return;
      e.preventDefault();

      const clientX = e.type === 'touchmove' ? e.touches[0].clientX : e.clientX;
      currentX = clientX - startX;

      const maxX = this.trackElement.offsetWidth - this.sliderElement.offsetWidth;
      currentX = Math.max(0, Math.min(currentX, maxX));

      this.sliderElement.style.left = `${currentX}px`;
    };

    const handleEnd = async () => {
      if (!isDragging) return;
      isDragging = false;
      this.sliderElement.classList.remove('dragging');

      const targetX = Math.round(currentX);

      try {
        const result = await this.client.verifySliderCaptcha({
          session_id: this.sessionId,
          x: targetX,
          y: this.secretY,
        });

        if (result.success) {
          this._onSuccess(result);
        } else {
          this._onFail(result.message);
        }
      } catch (error) {
        this._onFail(error.message);
      }
    };

    this.sliderElement.addEventListener('mousedown', handleStart);
    this.sliderElement.addEventListener('touchstart', handleStart, { passive: true });

    document.addEventListener('mousemove', handleMove);
    document.addEventListener('touchmove', handleMove, { passive: false });

    document.addEventListener('mouseup', handleEnd);
    document.addEventListener('touchend', handleEnd);

    this.refreshBtn.addEventListener('click', () => {
      this._init();
    });
  }

  _onSuccess(result) {
    this.isVerified = true;
    this.container.querySelector('.captcha-slider-thumb').classList.add('success');
    this.container.querySelector('.captcha-hint').textContent = '验证成功！';

    if (this.options.onSuccess) {
      this.options.onSuccess(result);
    }
  }

  _onFail(message) {
    this.container.querySelector('.captcha-hint').textContent = message || '验证失败，请重试';
    this.sliderElement.style.left = '0';

    if (this.options.onFail) {
      this.options.onFail(message);
    }
  }

  /**
   * 重新加载验证码
   */
  reload() {
    this.isVerified = false;
    this._init();
  }
}

/**
 * UI组件：点击验证码
 */
class ClickCaptchaWidget {
  /**
   * 创建点击验证码组件
   * @param {HTMLElement} container - 容器元素
   * @param {CaptchaClient} client - 验证码客户端
   * @param {Object} options - 配置选项
   */
  constructor(container, client, options = {}) {
    this.container = container;
    this.client = client;
    this.options = options;
    this.sessionId = null;
    this.hintOrder = [];
    this.clicks = [];

    this._init();
  }

  async _init() {
    try {
      const captcha = await this.client.getClickCaptcha({
        mode: this.options.mode || 'number',
        shuffle: this.options.shuffle !== false,
        points: this.options.points || 3,
      });

      this.sessionId = captcha.session_id;
      this.hintOrder = captcha.hint_order || [];
      this.clicks = [];

      this._render(captcha);
      this._bindEvents();
    } catch (error) {
      this.container.innerHTML = `<div class="captcha-error">加载验证码失败: ${error.message}</div>`;
    }
  }

  _render(captcha) {
    this.container.innerHTML = `
      <div class="captcha-click-widget">
        <div class="captcha-image-container">
          <img src="${captcha.image_url}" alt="验证码" class="captcha-img" />
        </div>
        <div class="captcha-hint-text">请按顺序点击: ${captcha.hint || ''}</div>
        <div class="captcha-controls">
          <button class="captcha-verify-btn">验证</button>
          <button class="captcha-refresh-btn">刷新</button>
        </div>
      </div>
    `;

    this.imgElement = this.container.querySelector('.captcha-img');
    this.verifyBtn = this.container.querySelector('.captcha-verify-btn');
    this.refreshBtn = this.container.querySelector('.captcha-refresh-btn');
  }

  _bindEvents() {
    this.imgElement.addEventListener('click', (e) => {
      const rect = this.imgElement.getBoundingClientRect();
      const x = Math.round(e.clientX - rect.left);
      const y = Math.round(e.clientY - rect.top);

      this.clicks.push([x, y]);

      this._drawClickMarker(e.clientX - rect.left, e.clientY - rect.top);
    });

    this.verifyBtn.addEventListener('click', async () => {
      if (this.clicks.length === 0) {
        return;
      }

      try {
        const result = await this.client.verifyClickCaptcha({
          session_id: this.sessionId,
          points: this.clicks,
          click_sequence: this.hintOrder,
        });

        if (result.success) {
          this._onSuccess(result);
        } else {
          this._onFail(result.message);
        }
      } catch (error) {
        this._onFail(error.message);
      }
    });

    this.refreshBtn.addEventListener('click', () => {
      this._init();
    });
  }

  _drawClickMarker(x, y) {
    const marker = document.createElement('div');
    marker.className = 'click-marker';
    marker.style.left = `${x}px`;
    marker.style.top = `${y}px`;
    marker.textContent = this.clicks.length;
    this.imgElement.parentElement.appendChild(marker);
  }

  _onSuccess(result) {
    this.verifyBtn.disabled = true;
    this.container.querySelector('.captcha-hint-text').textContent = '验证成功！';

    if (this.options.onSuccess) {
      this.options.onSuccess(result);
    }
  }

  _onFail(message) {
    this.container.querySelector('.captcha-hint-text').textContent = message || '验证失败，请重试';
    this.clicks = [];

    const markers = this.container.querySelectorAll('.click-marker');
    markers.forEach(m => m.remove());

    if (this.options.onFail) {
      this.options.onFail(message);
    }
  }

  /**
   * 重新加载验证码
   */
  reload() {
    this._init();
  }
}

// 导出
if (typeof module !== 'undefined' && module.exports) {
  module.exports = {
    CaptchaClient,
    UserAuth,
    Environment,
    SliderCaptchaWidget,
    ClickCaptchaWidget,
  };
} else if (typeof window !== 'undefined') {
  window.CaptchaClient = CaptchaClient;
  window.UserAuth = UserAuth;
  window.Environment = Environment;
  window.SliderCaptchaWidget = SliderCaptchaWidget;
  window.ClickCaptchaWidget = ClickCaptchaWidget;
}
