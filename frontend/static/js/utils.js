/**
 * 通用工具库
 * 提供格式化、验证、网络请求、存储、设备检测、性能监控等功能
 * 所有方法均为静态方法，通过 Utils 对象调用
 */

const Utils = {
    version: '1.0.0',
    author: 'Frontend Team',

    /**
     * 格式化日期
     * @param {Date|string|number} date - 日期对象、字符串或时间戳
     * @param {string} format - 格式化字符串，默认 'YYYY-MM-DD HH:mm:ss'
     * @returns {string} 格式化后的日期字符串
     */
    formatDate: function(date, format = 'YYYY-MM-DD HH:mm:ss') {
        if (!date) return '';

        const d = date instanceof Date ? date : new Date(date);
        if (isNaN(d.getTime())) return '';

        const year = d.getFullYear();
        const month = String(d.getMonth() + 1).padStart(2, '0');
        const day = String(d.getDate()).padStart(2, '0');
        const hours = String(d.getHours()).padStart(2, '0');
        const minutes = String(d.getMinutes()).padStart(2, '0');
        const seconds = String(d.getSeconds()).padStart(2, '0');

        return format
            .replace('YYYY', year)
            .replace('MM', month)
            .replace('DD', day)
            .replace('HH', hours)
            .replace('mm', minutes)
            .replace('ss', seconds);
    },

    /**
     * 格式化数字
     * @param {number} num - 要格式化的数字
     * @param {Object} options - 配置选项
     * @param {number} options.decimals - 小数位数，默认0
     * @param {string} options.separator - 千分位分隔符，默认','
     * @param {string} options.decPoint - 小数点符号，默认'.'
     * @returns {string} 格式化后的数字字符串
     */
    formatNumber: function(num, options = {}) {
        if (num === null || num === undefined || isNaN(num)) return '0';

        const {
            decimals = 0,
            separator = ',',
            decPoint = '.'
        } = options;

        const fixed = Number(num).toFixed(decimals);
        const parts = fixed.split('.');
        const integerPart = parts[0].replace(/\B(?=(\d{3})+(?!\d))/g, separator);

        return decimals > 0 ? integerPart + decPoint + parts[1] : integerPart;
    },

    /**
     * 格式化文件大小
     * @param {number} bytes - 字节数
     * @param {number} decimals - 小数位数，默认2
     * @returns {string} 格式化后的大小字符串
     */
    formatFileSize: function(bytes, decimals = 2) {
        if (bytes === 0) return '0 Bytes';

        const k = 1024;
        const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));

        return parseFloat((bytes / Math.pow(k, i)).toFixed(decimals)) + ' ' + sizes[i];
    },

    /**
     * 验证邮箱
     * @param {string} email - 邮箱地址
     * @returns {boolean} 是否为有效邮箱
     */
    validateEmail: function(email) {
        if (!email || typeof email !== 'string') return false;

        const pattern = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/;
        return pattern.test(email.trim());
    },

    /**
     * 验证手机号
     * @param {string} phone - 手机号
     * @param {string} locale - 地区，默认'zh-CN'
     * @returns {boolean} 是否为有效手机号
     */
    validatePhone: function(phone, locale = 'zh-CN') {
        if (!phone || typeof phone !== 'string') return false;

        const patterns = {
            'zh-CN': /^1[3-9]\d{9}$/,
            'zh-TW': /^09\d{9}$/,
            'en-US': /^1?\d{10}$/,
            'default': /^\d{10,15}$/
        };

        const pattern = patterns[locale] || patterns['default'];
        return pattern.test(phone.replace(/\s/g, ''));
    },

    /**
     * 验证URL
     * @param {string} url - URL地址
     * @returns {boolean} 是否为有效URL
     */
    validateUrl: function(url) {
        if (!url || typeof url !== 'string') return false;

        try {
            new URL(url);
            return true;
        } catch {
            return false;
        }
    },

    /**
     * 验证身份证号（中国）
     * @param {string} idCard - 身份证号
     * @returns {boolean} 是否为有效身份证号
     */
    validateIdCard: function(idCard) {
        if (!idCard || typeof idCard !== 'string') return false;

        const pattern = /^[1-9]\d{5}(18|19|20)\d{2}(0[1-9]|1[0-2])(0[1-9]|[12]\d|3[01])\d{3}[\dXx]$/;
        if (!pattern.test(idCard)) return false;

        const weights = [7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2];
        const checkCodes = ['1', '0', 'X', '9', '8', '7', '6', '5', '4', '3', '2'];

        let sum = 0;
        for (let i = 0; i < 17; i++) {
            sum += parseInt(idCard[i]) * weights[i];
        }

        const checkCode = checkCodes[sum % 11];
        return checkCode === idCard[17].toUpperCase();
    },

    /**
     * 防抖函数
     * @param {Function} func - 要执行的函数
     * @param {number} wait - 等待时间（毫秒）
     * @param {boolean} immediate - 是否立即执行
     * @returns {Function} 防抖后的函数
     */
    debounce: function(func, wait = 300, immediate = false) {
        let timeout;
        return function(...args) {
            const context = this;
            clearTimeout(timeout);

            if (immediate && !timeout) {
                func.apply(context, args);
            }

            timeout = setTimeout(() => {
                if (!immediate) {
                    func.apply(context, args);
                }
                timeout = null;
            }, wait);
        };
    },

    /**
     * 节流函数
     * @param {Function} func - 要执行的函数
     * @param {number} wait - 间隔时间（毫秒）
     * @param {Object} options - 配置选项
     * @param {boolean} options.leading - 是否在开始时执行
     * @param {boolean} options.trailing - 是否在结束时执行
     * @returns {Function} 节流后的函数
     */
    throttle: function(func, wait = 300, options = {}) {
        let timeout = null;
        let previous = 0;
        const { leading = true, trailing = true } = options;

        return function(...args) {
            const context = this;
            const now = Date.now();

            if (!previous && !leading) previous = now;

            const remaining = wait - (now - previous);
            if (remaining <= 0 || remaining > wait) {
                if (timeout) {
                    clearTimeout(timeout);
                    timeout = null;
                }
                previous = now;
                func.apply(context, args);
            } else if (!timeout && trailing) {
                timeout = setTimeout(() => {
                    previous = leading ? Date.now() : 0;
                    timeout = null;
                    func.apply(context, args);
                }, remaining);
            }
        };
    },

    /**
     * 深拷贝
     * @param {*} obj - 要拷贝的对象
     * @param {WeakMap} [cache] - 缓存（用于处理循环引用）
     * @returns {*} 拷贝后的对象
     */
    deepClone: function(obj, cache = new WeakMap()) {
        if (obj === null || typeof obj !== 'object') return obj;
        if (obj instanceof Date) return new Date(obj.getTime());
        if (obj instanceof RegExp) return new RegExp(obj.source, obj.flags);
        if (obj instanceof Map) {
            const cloned = new Map();
            cache.set(obj, cloned);
            obj.forEach((value, key) => {
                cloned.set(key, this.deepClone(value, cache));
            });
            return cloned;
        }
        if (obj instanceof Set) {
            const cloned = new Set();
            cache.set(obj, cloned);
            obj.forEach(value => {
                cloned.add(this.deepClone(value, cache));
            });
            return cloned;
        }
        if (cache.has(obj)) return cache.get(obj);

        const cloned = Array.isArray(obj) ? [] : {};
        cache.set(obj, cloned);

        for (const key in obj) {
            if (Object.prototype.hasOwnProperty.call(obj, key)) {
                cloned[key] = this.deepClone(obj[key], cache);
            }
        }

        return cloned;
    },

    /**
     * 生成随机字符串
     * @param {number} length - 长度
     * @param {string} chars - 字符集
     * @returns {string} 随机字符串
     */
    randomString: function(length = 32, chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789') {
        let result = '';
        for (let i = 0; i < length; i++) {
            result += chars.charAt(Math.floor(Math.random() * chars.length));
        }
        return result;
    },

    /**
     * 生成UUID
     * @returns {string} UUID字符串
     */
    uuid: function() {
        return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
            const r = Math.random() * 16 | 0;
            const v = c === 'x' ? r : (r & 0x3 | 0x8);
            return v.toString(16);
        });
    },

    /**
     * 获取URL参数
     * @param {string} [name] - 参数名，不传则返回所有参数
     * @returns {string|Object} 参数值或参数对象
     */
    getUrlParam: function(name) {
        const params = {};
        const searchParams = new URLSearchParams(window.location.search);

        searchParams.forEach((value, key) => {
            params[key] = value;
        });

        return name ? (params[name] || '') : params;
    },

    /**
     * 设置URL参数
     * @param {string|Object} name - 参数名或参数对象
     * @param {string} [value] - 参数值
     * @param {boolean} [replace] - 是否替换历史记录
     */
    setUrlParam: function(name, value, replace = false) {
        const url = new URL(window.location.href);
        const params = new URLSearchParams(url.search);

        if (typeof name === 'object') {
            Object.entries(name).forEach(([k, v]) => {
                params.set(k, v);
            });
        } else {
            params.set(name, value);
        }

        url.search = params.toString();
        const newUrl = url.toString();

        if (replace) {
            history.replaceState(null, '', newUrl);
        } else {
            history.pushState(null, '', newUrl);
        }
    },

    /**
     * 异步AJAX请求
     * @param {string} url - 请求URL
     * @param {Object} options - 配置选项
     * @returns {Promise<Object>} 响应数据
     */
    ajax: function(url, options = {}) {
        const defaults = {
            method: 'GET',
            headers: {},
            data: null,
            timeout: 30000,
            credentials: 'same-origin',
            cache: 'default'
        };

        const config = { ...defaults, ...options };

        return new Promise((resolve, reject) => {
            const xhr = new XMLHttpRequest();

            xhr.open(config.method.toUpperCase(), url, true);

            xhr.timeout = config.timeout;

            if (config.credentials) {
                xhr.withCredentials = config.credentials === 'include';
            }

            Object.entries(config.headers).forEach(([key, value]) => {
                xhr.setRequestHeader(key, value);
            });

            xhr.onload = function() {
                if (xhr.status >= 200 && xhr.status < 300) {
                    try {
                        const response = JSON.parse(xhr.responseText);
                        resolve(response);
                    } catch {
                        resolve(xhr.responseText);
                    }
                } else {
                    reject(new Error(`Request failed with status ${xhr.status}: ${xhr.statusText}`));
                }
            };

            xhr.onerror = function() {
                reject(new Error('Network error'));
            };

            xhr.ontimeout = function() {
                reject(new Error('Request timeout'));
            };

            if (config.data) {
                if (typeof config.data === 'object' && !(config.data instanceof FormData)) {
                    xhr.setRequestHeader('Content-Type', 'application/json');
                    xhr.send(JSON.stringify(config.data));
                } else {
                    xhr.send(config.data);
                }
            } else {
                xhr.send();
            }
        });
    },

    /**
     * Fetch JSON数据
     * @param {string} url - 请求URL
     * @param {Object} options - 配置选项
     * @returns {Promise<Object>} JSON响应数据
     */
    fetchJSON: async function(url, options = {}) {
        const defaults = {
            method: 'GET',
            headers: {
                'Content-Type': 'application/json',
                'Accept': 'application/json'
            },
            credentials: 'same-origin',
            cache: 'default'
        };

        const config = { ...defaults, ...options };

        if (config.data && config.method.toUpperCase() === 'GET') {
            const params = new URLSearchParams(config.data);
            url += (url.includes('?') ? '&' : '?') + params.toString();
            delete config.data;
        }

        try {
            const response = await fetch(url, config);

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            return await response.json();
        } catch (error) {
            console.error('Fetch error:', error);
            throw error;
        }
    },

    /**
     * 检测是否为移动设备
     * @returns {boolean} 是否为移动设备
     */
    isMobile: function() {
        return /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent);
    },

    /**
     * 检测是否为平板设备
     * @returns {boolean} 是否为平板设备
     */
    isTablet: function() {
        const ua = navigator.userAgent;
        return /iPad|Android/i.test(ua) && !/Mobile/i.test(ua);
    },

    /**
     * 检测是否为iOS设备
     * @returns {boolean} 是否为iOS设备
     */
    isIOS: function() {
        return /iPad|iPhone|iPod/.test(navigator.userAgent);
    },

    /**
     * 检测是否为Android设备
     * @returns {boolean} 是否为Android设备
     */
    isAndroid: function() {
        return /Android/.test(navigator.userAgent);
    },

    /**
     * 检测是否为微信浏览器
     * @returns {boolean} 是否为微信浏览器
     */
    isWeChat: function() {
        return /MicroMessenger/i.test(navigator.userAgent);
    },

    /**
     * 获取浏览器信息
     * @returns {Object} 浏览器信息对象
     */
    getBrowser: function() {
        const ua = navigator.userAgent;
        const browser = {};

        if (ua.indexOf('Firefox') > -1) {
            browser.name = 'Firefox';
            browser.version = ua.match(/Firefox\/([\d.]+)/)?.[1] || '';
        } else if (ua.indexOf('Chrome') > -1 && ua.indexOf('Edg') === -1) {
            browser.name = 'Chrome';
            browser.version = ua.match(/Chrome\/([\d.]+)/)?.[1] || '';
        } else if (ua.indexOf('Safari') > -1 && ua.indexOf('Chrome') === -1) {
            browser.name = 'Safari';
            browser.version = ua.match(/Version\/([\d.]+)/)?.[1] || '';
        } else if (ua.indexOf('Edg') > -1) {
            browser.name = 'Edge';
            browser.version = ua.match(/Edg\/([\d.]+)/)?.[1] || '';
        } else if (ua.indexOf('Opera') > -1 || ua.indexOf('OPR') > -1) {
            browser.name = 'Opera';
            browser.version = ua.match(/(?:Opera|OPR)\/([\d.]+)/)?.[1] || '';
        } else {
            browser.name = 'Unknown';
            browser.version = '';
        }

        browser.isMobile = this.isMobile();
        browser.isTablet = this.isTablet();
        browser.isIOS = this.isIOS();
        browser.isAndroid = this.isAndroid();
        browser.isWeChat = this.isWeChat();

        return browser;
    },

    /**
     * 获取操作系统信息
     * @returns {string} 操作系统名称
     */
    getOS: function() {
        const ua = navigator.userAgent;
        if (ua.indexOf('Win') > -1) return 'Windows';
        if (ua.indexOf('Mac') > -1) return 'macOS';
        if (ua.indexWith('Linux') > -1) return 'Linux';
        if (ua.indexOf('Android') > -1) return 'Android';
        if (ua.indexOf('iOS') > -1 || ua.indexOf('iPhone') > -1 || ua.indexOf('iPad') > -1) return 'iOS';
        return 'Unknown';
    },

    /**
     * 获取屏幕信息
     * @returns {Object} 屏幕信息对象
     */
    getScreen: function() {
        return {
            width: window.screen.width,
            height: window.screen.height,
            availWidth: window.screen.availWidth,
            availHeight: window.screen.availHeight,
            colorDepth: window.screen.colorDepth,
            pixelRatio: window.devicePixelRatio
        };
    },

    /**
     * 性能监控
     */
    performance: {
        marks: {},

        /**
         * 开始计时
         * @param {string} name - 标记名称
         */
        start: function(name) {
            this.marks[name] = {
                startTime: performance.now(),
                endTime: null
            };
        },

        /**
         * 结束计时
         * @param {string} name - 标记名称
         * @returns {number} 耗时（毫秒）
         */
        end: function(name) {
            if (!this.marks[name]) {
                console.warn(`Performance mark "${name}" not found`);
                return 0;
            }

            this.marks[name].endTime = performance.now();
            return this.marks[name].endTime - this.marks[name].startTime;
        },

        /**
         * 获取所有性能记录
         * @returns {Array} 性能记录数组
         */
        getEntries: function() {
            return Object.entries(this.marks).map(([name, mark]) => ({
                name,
                duration: mark.endTime ? mark.endTime - mark.startTime : null,
                startTime: mark.startTime,
                endTime: mark.endTime
            }));
        },

        /**
         * 获取Navigation Timing
         * @returns {Object} 导航时间信息
         */
        getTiming: function() {
            const timing = performance.timing || {};
            return {
                dns: timing.domainLookupEnd - timing.domainLookupStart,
                tcp: timing.connectEnd - timing.connectStart,
                ssl: timing.secureConnectionStart > 0 ? timing.connectEnd - timing.secureConnectionStart : 0,
                ttfb: timing.responseStart - timing.requestStart,
                download: timing.responseEnd - timing.responseStart,
                domParse: timing.domInteractive - timing.responseEnd,
                domReady: timing.domContentLoadedEventEnd - timing.navigationStart,
                loadComplete: timing.loadEventEnd - timing.navigationStart
            };
        },

        /**
         * 获取资源加载信息
         * @returns {Array} 资源加载信息数组
         */
        getResources: function() {
            const resources = performance.getEntriesByType('resource') || [];
            return resources.map(resource => ({
                name: resource.name,
                type: resource.initiatorType,
                duration: resource.duration,
                size: resource.transferSize || 0,
                dns: resource.domainLookupEnd - resource.domainLookupStart,
                tcp: resource.connectEnd - resource.connectStart,
                ttfb: resource.responseStart - resource.requestStart
            }));
        }
    },

    /**
     * 性能测量装饰器
     * @param {string} name - 测量名称
     * @param {Function} fn - 要执行的函数
     * @returns {*} 函数执行结果
     */
    measure: function(name, fn) {
        this.performance.start(name);
        try {
            const result = fn();
            const duration = this.performance.end(name);
            console.log(`[Performance] ${name}: ${duration.toFixed(2)}ms`);
            return result;
        } catch (error) {
            const duration = this.performance.end(name);
            console.error(`[Performance] ${name} failed after ${duration.toFixed(2)}ms:`, error);
            throw error;
        }
    },

    /**
     * 上报性能指标
     * @param {string} metric - 指标名称
     * @param {number} value - 指标值
     * @param {Object} [tags] - 标签
     */
    report: function(metric, value, tags = {}) {
        const payload = {
            metric,
            value,
            tags,
            timestamp: Date.now(),
            url: window.location.href,
            userAgent: navigator.userAgent
        };

        if (typeof window.__metricsCollector !== 'undefined') {
            window.__metricsCollector.push(payload);
        }

        console.log('[Metrics]', payload);
    },

    /**
     * 本地存储管理器
     */
    storage: {
        /**
         * 获取存储数据
         * @param {string} key - 键名
         * @param {*} defaultValue - 默认值
         * @returns {*} 存储的值
         */
        get: function(key, defaultValue = null) {
            try {
                const item = localStorage.getItem(key);
                if (item === null) return defaultValue;

                try {
                    return JSON.parse(item);
                } catch {
                    return item;
                }
            } catch (error) {
                console.error('Storage get error:', error);
                return defaultValue;
            }
        },

        /**
         * 设置存储数据
         * @param {string} key - 键名
         * @param {*} value - 值
         * @returns {boolean} 是否成功
         */
        set: function(key, value) {
            try {
                const data = typeof value === 'string' ? value : JSON.stringify(value);
                localStorage.setItem(key, data);
                return true;
            } catch (error) {
                console.error('Storage set error:', error);
                return false;
            }
        },

        /**
         * 移除存储数据
         * @param {string} key - 键名
         * @returns {boolean} 是否成功
         */
        remove: function(key) {
            try {
                localStorage.removeItem(key);
                return true;
            } catch (error) {
                console.error('Storage remove error:', error);
                return false;
            }
        },

        /**
         * 清空所有存储
         * @returns {boolean} 是否成功
         */
        clear: function() {
            try {
                localStorage.clear();
                return true;
            } catch (error) {
                console.error('Storage clear error:', error);
                return false;
            }
        },

        /**
         * 检查存储键是否存在
         * @param {string} key - 键名
         * @returns {boolean} 是否存在
         */
        has: function(key) {
            return localStorage.getItem(key) !== null;
        },

        /**
         * 获取所有存储键
         * @returns {Array} 键数组
         */
        keys: function() {
            return Object.keys(localStorage);
        }
    },

    /**
     * Session存储管理器
     */
    session: {
        get: function(key, defaultValue = null) {
            try {
                const item = sessionStorage.getItem(key);
                if (item === null) return defaultValue;

                try {
                    return JSON.parse(item);
                } catch {
                    return item;
                }
            } catch (error) {
                console.error('Session get error:', error);
                return defaultValue;
            }
        },

        set: function(key, value) {
            try {
                const data = typeof value === 'string' ? value : JSON.stringify(value);
                sessionStorage.setItem(key, data);
                return true;
            } catch (error) {
                console.error('Session set error:', error);
                return false;
            }
        },

        remove: function(key) {
            try {
                sessionStorage.removeItem(key);
                return true;
            } catch (error) {
                console.error('Session remove error:', error);
                return false;
            }
        },

        clear: function() {
            try {
                sessionStorage.clear();
                return true;
            } catch (error) {
                console.error('Session clear error:', error);
                return false;
            }
        }
    },

    /**
     * Cookie管理器
     */
    cookie: {
        /**
         * 获取Cookie值
         * @param {string} name - Cookie名称
         * @returns {string|null} Cookie值
         */
        get: function(name) {
            const match = document.cookie.match(new RegExp('(^| )' + name + '=([^;]+)'));
            return match ? decodeURIComponent(match[2]) : null;
        },

        /**
         * 设置Cookie
         * @param {string} name - Cookie名称
         * @param {string} value - Cookie值
         * @param {Object} [options] - 配置选项
         */
        set: function(name, value, options = {}) {
            const {
                expires = 30,
                path = '/',
                domain = '',
                secure = false,
                sameSite = 'Lax'
            } = options;

            let cookieString = `${encodeURIComponent(name)}=${encodeURIComponent(value)}`;

            if (expires) {
                const date = new Date();
                date.setTime(date.getTime() + expires * 24 * 60 * 60 * 1000);
                cookieString += `; expires=${date.toUTCString()}`;
            }

            if (path) {
                cookieString += `; path=${path}`;
            }

            if (domain) {
                cookieString += `; domain=${domain}`;
            }

            if (secure) {
                cookieString += '; secure';
            }

            cookieString += `; SameSite=${sameSite}`;

            document.cookie = cookieString;
        },

        /**
         * 删除Cookie
         * @param {string} name - Cookie名称
         * @param {Object} [options] - 配置选项
         */
        remove: function(name, options = {}) {
            this.set(name, '', { ...options, expires: -1 });
        }
    },

    /**
     * 事件管理器
     */
    events: {
        listeners: {},

        /**
         * 注册事件监听
         * @param {string} event - 事件名
         * @param {Function} callback - 回调函数
         * @param {Object} [context] - 上下文
         */
        on: function(event, callback, context = null) {
            if (!this.listeners[event]) {
                this.listeners[event] = [];
            }

            this.listeners[event].push({ callback, context });
        },

        /**
         * 触发事件
         * @param {string} event - 事件名
         * @param {*} data - 事件数据
         */
        emit: function(event, data = null) {
            if (!this.listeners[event]) return;

            this.listeners[event].forEach(({ callback, context }) => {
                try {
                    callback.call(context, data);
                } catch (error) {
                    console.error(`Event handler error for "${event}":`, error);
                }
            });
        },

        /**
         * 移除事件监听
         * @param {string} event - 事件名
         * @param {Function} [callback] - 回调函数，不传则移除所有
         */
        off: function(event, callback = null) {
            if (!callback) {
                delete this.listeners[event];
            } else {
                this.listeners[event] = this.listeners[event].filter(
                    item => item.callback !== callback
                );
            }
        },

        /**
         * 只执行一次的事件监听
         * @param {string} event - 事件名
         * @param {Function} callback - 回调函数
         * @param {Object} [context] - 上下文
         */
        once: function(event, callback, context = null) {
            const wrapper = (data) => {
                this.off(event, wrapper);
                callback.call(context, data);
            };
            this.on(event, wrapper, context);
        }
    },

    /**
     * 复制文本到剪贴板
     * @param {string} text - 要复制的文本
     * @returns {Promise<boolean>} 是否成功
     */
    copyToClipboard: async function(text) {
        try {
            if (navigator.clipboard && navigator.clipboard.writeText) {
                await navigator.clipboard.writeText(text);
                return true;
            }

            const textarea = document.createElement('textarea');
            textarea.value = text;
            textarea.style.position = 'fixed';
            textarea.style.opacity = '0';
            document.body.appendChild(textarea);
            textarea.select();
            document.execCommand('copy');
            document.body.removeChild(textarea);
            return true;
        } catch (error) {
            console.error('Copy to clipboard failed:', error);
            return false;
        }
    },

    /**
     * 滚动到指定元素
     * @param {string|Element} target - 元素选择器或元素
     * @param {Object} [options] - 配置选项
     */
    scrollTo: function(target, options = {}) {
        const {
            offset = 0,
            duration = 500,
            easing = 'swing'
        } = options;

        const element = typeof target === 'string' ? document.querySelector(target) : target;
        if (!element) return;

        const startPosition = window.pageYOffset;
        const targetPosition = element.getBoundingClientRect().top + startPosition - offset;
        const distance = targetPosition - startPosition;
        let startTime = null;

        const ease = (t, b, c, d) => {
            t /= d / 2;
            if (t < 1) return c / 2 * t * t + b;
            t--;
            return -c / 2 * (t * (t - 2) - 1) + b;
        };

        const animate = (currentTime) => {
            if (startTime === null) startTime = currentTime;
            const elapsed = currentTime - startTime;
            const progress = Math.min(elapsed / duration, 1);
            const position = startPosition + distance * ease(progress, 0, 1, 1);

            window.scrollTo(0, position);

            if (elapsed < duration) {
                requestAnimationFrame(animate);
            }
        };

        requestAnimationFrame(animate);
    },

    /**
     * 延迟执行
     * @param {number} ms - 延迟毫秒数
     * @returns {Promise<void>}
     */
    sleep: function(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    },

    /**
     * 判断是否为有效数字
     * @param {*} value - 要检查的值
     * @returns {boolean} 是否为有效数字
     */
    isNumeric: function(value) {
        return !isNaN(parseFloat(value)) && isFinite(value);
    },

    /**
     * 判断是否为空值（null, undefined, '', [], {}）
     * @param {*} value - 要检查的值
     * @returns {boolean} 是否为空
     */
    isEmpty: function(value) {
        if (value === null || value === undefined) return true;
        if (typeof value === 'string') return value.trim() === '';
        if (Array.isArray(value)) return value.length === 0;
        if (typeof value === 'object') return Object.keys(value).length === 0;
        return false;
    },

    /**
     * 安全地获取嵌套属性
     * @param {Object} obj - 对象
     * @param {string} path - 属性路径，如 'a.b.c'
     * @param {*} defaultValue - 默认值
     * @returns {*} 属性值
     */
    get: function(obj, path, defaultValue = undefined) {
        return path.split('.').reduce((acc, part) => {
            return acc && acc[part] !== undefined ? acc[part] : defaultValue;
        }, obj);
    },

    /**
     * 安全地设置嵌套属性
     * @param {Object} obj - 对象
     * @param {string} path - 属性路径
     * @param {*} value - 要设置的值
     */
    set: function(obj, path, value) {
        const parts = path.split('.');
        const lastIndex = parts.length - 1;

        parts.reduce((acc, part, index) => {
            if (index === lastIndex) {
                acc[part] = value;
                return value;
            }
            if (!acc[part] || typeof acc[part] !== 'object') {
                acc[part] = {};
            }
            return acc[part];
        }, obj);
    },

    /**
     * 生成密码强度指数
     * @param {string} password - 密码
     * @returns {Object} 强度信息
     */
    passwordStrength: function(password) {
        if (!password) return { score: 0, level: 'empty' };

        let score = 0;
        const checks = {
            length: password.length >= 8,
            lowercase: /[a-z]/.test(password),
            uppercase: /[A-Z]/.test(password),
            number: /\d/.test(password),
            special: /[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]/.test(password)
        };

        Object.values(checks).forEach(passed => {
            if (passed) score++;
        });

        const levels = ['empty', 'weak', 'fair', 'good', 'strong', 'excellent'];
        return {
            score,
            level: levels[Math.min(score, levels.length - 1)],
            checks
        };
    }
};

document.addEventListener('DOMContentLoaded', function() {
    window.Utils = Utils;
});
