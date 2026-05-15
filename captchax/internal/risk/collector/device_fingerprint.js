/**
 * @fileoverview 设备指纹采集器
 * @description 采集Canvas指纹、WebGL指纹、字体指纹、浏览器环境信息
 * @module captchax/internal/risk/collector/device_fingerprint
 */

'use strict';

class DeviceFingerprint {
  constructor(config = {}) {
    this.config = {
      hashAlgorithm: config.hashAlgorithm || 'md5',
      components: {
        canvas: config.components?.canvas !== false,
        webgl: config.components?.webgl !== false,
        fonts: config.components?.fonts !== false,
        browser: config.components?.browser !== false,
        screen: config.components?.screen !== false,
        timezone: config.components?.timezone !== false,
        language: config.components?.language !== false
      },
      ...config
    };
    
    this.cachedFingerprint = null;
    this.cachedHash = null;
  }

  async collect() {
    if (this.cachedFingerprint) {
      return this.cachedFingerprint;
    }

    const fingerprint = {
      timestamp: Date.now(),
      hash: null,
      components: {}
    };

    if (this.config.components.canvas) {
      fingerprint.components.canvas = await this.getCanvasFingerprint();
    }

    if (this.config.components.webgl) {
      fingerprint.components.webgl = this.getWebGLFingerprint();
    }

    if (this.config.components.fonts) {
      fingerprint.components.fonts = this.getFontFingerprint();
    }

    if (this.config.components.browser) {
      fingerprint.components.browser = this.getBrowserInfo();
    }

    if (this.config.components.screen) {
      fingerprint.components.screen = this.getScreenInfo();
    }

    if (this.config.components.timezone) {
      fingerprint.components.timezone = this.getTimezoneInfo();
    }

    if (this.config.components.language) {
      fingerprint.components.language = this.getLanguageInfo();
    }

    fingerprint.hash = this.calculateHash(fingerprint.components);
    this.cachedFingerprint = fingerprint;
    this.cachedHash = fingerprint.hash;
    
    return fingerprint;
  }

  async getCanvasFingerprint() {
    try {
      const canvas = document.createElement('canvas');
      canvas.width = 200;
      canvas.height = 50;
      const ctx = canvas.getContext('2d');
      
      if (!ctx) return null;

      ctx.textBaseline = 'top';
      ctx.font = '14px Arial';
      ctx.fillStyle = '#f60';
      ctx.fillRect(125, 1, 62, 20);
      ctx.fillStyle = '#069';
      ctx.fillText('CaptchaX Fingerprint', 2, 15);
      ctx.fillStyle = 'rgba(102, 204, 0, 0.7)';
      ctx.fillText('CaptchaX Fingerprint', 4, 17);
      
      const dataUrl = canvas.toDataURL();
      return this.hashString(dataUrl);
    } catch (error) {
      console.error('Canvas fingerprint error:', error);
      return null;
    }
  }

  getWebGLFingerprint() {
    try {
      const canvas = document.createElement('canvas');
      const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
      
      if (!gl) return null;

      const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
      
      const result = {
        vendor: gl.getParameter(gl.VENDOR),
        renderer: gl.getParameter(gl.RENDERER),
        version: gl.getParameter(gl.VERSION),
        shadingLanguageVersion: gl.getParameter(gl.SHADING_LANGUAGE_VERSION)
      };

      if (debugInfo) {
        result.unmaskedVendor = gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL);
        result.unmaskedRenderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
      }

      const parameters = [
        gl.MAX_TEXTURE_SIZE,
        gl.MAX_CUBE_MAP_TEXTURE_SIZE,
        gl.MAX_VERTEX_ATTRIBS,
        gl.MAX_VERTEX_UNIFORM_VECTORS,
        gl.MAX_VARYING_VECTORS,
        gl.MAX_COMBINED_TEXTURE_IMAGE_UNITS,
        gl.MAX_FRAGMENT_UNIFORM_VECTORS
      ];

      result.parameters = parameters.map(p => gl.getParameter(p));

      const shaderTest = this.testWebGLShaders(gl);
      result.shaders = shaderTest;

      result.hash = this.hashString(JSON.stringify(result));
      
      return result;
    } catch (error) {
      console.error('WebGL fingerprint error:', error);
      return null;
    }
  }

  testWebGLShaders(gl) {
    const result = { fragment: null, vertex: null };
    
    try {
      const fragmentShader = gl.createShader(gl.FRAGMENT_SHADER);
      gl.shaderSource(fragmentShader, `
        precision mediump float;
        void main(void) {
          gl_FragColor = vec4(1.0, 0.0, 0.0, 1.0);
        }
      `);
      gl.compileShader(fragmentShader);
      result.fragment = gl.getShaderParameter(fragmentShader, gl.COMPILE_STATUS);
      
      const vertexShader = gl.createShader(gl.VERTEX_SHADER);
      gl.shaderSource(vertexShader, `
        attribute vec3 position;
        void main(void) {
          gl_Position = vec4(position, 1.0);
        }
      `);
      gl.compileShader(vertexShader);
      result.vertex = gl.getShaderParameter(vertexShader, gl.VERTEX_SHADER);
    } catch (e) {
      console.error('WebGL shader test error:', e);
    }
    
    return result;
  }

  getFontFingerprint() {
    const baseFonts = ['monospace', 'sans-serif', 'serif'];
    const testString = 'mmmmmmmmmmlli';
    const testSize = '72px';
    
    const canvas = document.createElement('canvas');
    const ctx = canvas.getContext('2d');
    
    if (!ctx) return null;

    const getWidth = (fontFamily) => {
      ctx.font = `${testSize} ${fontFamily}`;
      return ctx.measureText(testString).width;
    };

    const baseWidths = baseFonts.map(font => getWidth(font));
    
    const testFonts = [
      'Arial', 'Arial Black', 'Comic Sans MS', 'Courier New', 'Georgia',
      'Impact', 'Times New Roman', 'Trebuchet MS', 'Verdana', 'Palatino',
      'Lucida Console', 'Lucida Sans Unicode', 'Tahoma', 'Geneva', 'Helvetica'
    ];

    const availableFonts = [];
    
    for (const font of testFonts) {
      const detected = baseFonts.some((baseFont, index) => {
        return getWidth(`'${font}', ${baseFont}`) !== baseWidths[index];
      });
      
      if (detected) {
        availableFonts.push(font);
      }
    }

    return {
      availableFonts,
      count: availableFonts.length,
      hash: this.hashString(availableFonts.sort().join(','))
    };
  }

  getBrowserInfo() {
    const ua = navigator.userAgent;
    
    const parseUserAgent = (ua) => {
      const browsers = {
        Chrome: /Chrome\/([\d.]+)/,
        Firefox: /Firefox\/([\d.]+)/,
        Safari: /Version\/([\d.]+).*Safari/,
        Edge: /Edg\/([\d.]+)/,
        Opera: /OPR\/([\d.]+)/,
        IE: /MSIE\s([\d.]+)/,
        'IE Mobile': /IEMobile\/([\d.]+)/
      };

      for (const [name, regex] of Object.entries(browsers)) {
        const match = ua.match(regex);
        if (match) {
          return { name, version: match[1] };
        }
      }
      
      return { name: 'Unknown', version: '0' };
    };

    const browser = parseUserAgent(ua);

    const parseOS = (ua) => {
      const systems = {
        'Windows': /Windows NT ([\d.]+)/,
        'Mac OS': /Mac OS X ([\d._]+)/,
        'Linux': /Linux/,
        'Android': /Android ([\d.]+)/,
        'iOS': /OS ([\d_]+)/
      };

      for (const [name, regex] of Object.entries(systems)) {
        const match = ua.match(regex);
        if (match) {
          if (name === 'Windows') {
            const ntVersion = parseFloat(match[1]);
            let windowsVersion = 'Unknown';
            if (ntVersion === 10) windowsVersion = '10';
            else if (ntVersion === 6.3) windowsVersion = '8.1';
            else if (ntVersion === 6.2) windowsVersion = '8';
            else if (ntVersion === 6.1) windowsVersion = '7';
            else if (ntVersion === 6.0) windowsVersion = 'Vista';
            return `${name} ${windowsVersion}`;
          }
          return name + (match[1] ? ' ' + match[1].replace(/_/g, '.') : '');
        }
      }
      
      return 'Unknown';
    };

    const platform = navigator.platform || 'Unknown';
    const os = parseOS(ua);
    const mobile = /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(ua);

    const plugins = this.getPlugins();
    const cookiesEnabled = navigator.cookieEnabled;
    const doNotTrack = navigator.doNotTrack === '1' || navigator.doNotTrack === 'yes';
    const javaEnabled = navigator.javaEnabled ? navigator.javaEnabled() : false;
    
    const touchPoints = navigator.maxTouchPoints || 0;
    const hardwareConcurrency = navigator.hardwareConcurrency || 0;
    const deviceMemory = navigator.deviceMemory || 0;

    return {
      userAgent: ua,
      browser: browser.name,
      browserVersion: browser.version,
      os,
      platform,
      mobile,
      plugins,
      cookiesEnabled,
      doNotTrack,
      javaEnabled,
      touchPoints,
      hardwareConcurrency,
      deviceMemory,
      webdriver: navigator.webdriver || false,
      languages: navigator.languages || [navigator.language],
      hash: this.hashString(`${browser.name}${browser.version}${os}${platform}`)
    };
  }

  getPlugins() {
    try {
      const plugins = [];
      for (let i = 0; i < navigator.plugins.length; i++) {
        const plugin = navigator.plugins[i];
        plugins.push({
          name: plugin.name,
          filename: plugin.filename,
          description: plugin.description
        });
      }
      return plugins;
    } catch (error) {
      return [];
    }
  }

  getScreenInfo() {
    return {
      width: screen.width,
      height: screen.height,
      colorDepth: screen.colorDepth,
      pixelDepth: screen.pixelDepth,
      availWidth: screen.availWidth,
      availHeight: screen.availHeight,
      innerWidth: window.innerWidth,
      innerHeight: window.innerHeight,
      devicePixelRatio: window.devicePixelRatio || 1,
      orientation: screen.orientation ? screen.orientation.type : 'unknown'
    };
  }

  getTimezoneInfo() {
    try {
      const offset = new Date().getTimezoneOffset();
      const tz = Intl.DateTimeFormat().resolvedOptions().timeZone;
      const offsetHours = -offset / 60;
      const offsetString = `UTC${offsetHours >= 0 ? '+' : ''}${offsetHours}`;
      
      const jan = new Date(2024, 0, 1);
      const jul = new Date(2024, 6, 1);
      const janOffset = new Date(jan).getTimezoneOffset();
      const julOffset = new Date(jul).getTimezoneOffset();
      const isDST = janOffset !== julOffset;

      return {
        timezone: tz,
        offset,
        offsetHours,
        offsetString,
        isDST,
        hash: this.hashString(`${tz}${offset}`)
      };
    } catch (error) {
      return {
        timezone: 'Unknown',
        offset: 0,
        isDST: false
      };
    }
  }

  getLanguageInfo() {
    return {
      language: navigator.language || navigator.userLanguage,
      languages: navigator.languages || [],
      systemLanguage: navigator.systemLanguage || '',
      browserLanguage: navigator.browserLanguage || ''
    };
  }

  calculateHash(components) {
    const str = JSON.stringify(components);
    return this.hashString(str);
  }

  hashString(str) {
    let hash = 0;
    if (str.length === 0) return hash.toString(16);
    
    for (let i = 0; i < str.length; i++) {
      const char = str.charCodeAt(i);
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash;
    }
    
    return Math.abs(hash).toString(16).padStart(8, '0');
  }

  getHash() {
    if (!this.cachedHash) {
      this.collect();
    }
    return this.cachedHash;
  }

  clearCache() {
    this.cachedFingerprint = null;
    this.cachedHash = null;
  }

  static generateId() {
    const timestamp = Date.now().toString(36);
    const randomPart = Math.random().toString(36).substring(2, 15);
    return `${timestamp}-${randomPart}`;
  }
}

module.exports = DeviceFingerprint;
