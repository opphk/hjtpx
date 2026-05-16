/**
 * 前端安全增强模块
 * 提供XSS防护、CSRF Token管理和输入验证功能
 */

class SecurityUtils {
    /**
     * HTML转义
     * @param {string} text - 待转义文本
     * @returns {string} 转义后的文本
     */
    static escapeHTML(text) {
        if (!text) return '';
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    /**
     * HTML属性转义
     * @param {string} text - 待转义文本
     * @returns {string} 转义后的文本
     */
    static escapeHTMLAttribute(text) {
        if (!text) return '';
        const map = {
            '&': '&amp;',
            '<': '&lt;',
            '>': '&gt;',
            '"': '&quot;',
            "'": '&#39;',
            ' ': '%20'
        };
        return String(text).replace(/[&<>"' ]/g, c => map[c]);
    }

    /**
     * URL转义
     * @param {string} text - 待转义文本
     * @returns {string} 转义后的文本
     */
    static escapeURL(text) {
        if (!text) return '';
        return encodeURIComponent(text);
    }

    /**
     * JavaScript转义
     * @param {string} text - 待转义文本
     * @returns {string} 转义后的文本
     */
    static escapeJavaScript(text) {
        if (!text) return '';
        const map = {
            '\\': '\\\\',
            '"': '\\"',
            "'": "\\'",
            '\n': '\\n',
            '\r': '\\r',
            '\t': '\\t',
            '<': '\\x3c',
            '>': '\\x3e'
        };
        return String(text).replace(/[\\"'\<\>\n\r\t]/g, c => map[c] || c);
    }

    /**
     * 过滤危险HTML标签
     * @param {string} html - 待过滤HTML
     * @returns {string} 过滤后的HTML
     */
    static stripDangerousTags(html) {
        if (!html) return '';
        let result = html;
        
        const dangerousPatterns = [
            /<\s*script[^>]*>[\s\S]*?<\/script>/gi,
            /<\s*iframe[^>]*>[\s\S]*?<\/iframe>/gi,
            /<\s*object[^>]*>[\s\S]*?<\/object>/gi,
            /<\s*embed[^>]*>/gi,
            /<\s*applet[^>]*>[\s\S]*?<\/applet>/gi,
            /on\w+\s*=\s*["'][^"']*["']/gi,
            /javascript\s*:/gi,
            /data\s*:/gi
        ];

        dangerousPatterns.forEach(pattern => {
            result = result.replace(pattern, '');
        });

        return result;
    }

    /**
     * 验证邮箱格式
     * @param {string} email - 邮箱地址
     * @returns {boolean} 是否有效
     */
    static isValidEmail(email) {
        if (!email) return false;
        const pattern = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/;
        return pattern.test(email);
    }

    /**
     * 验证手机号格式
     * @param {string} phone - 手机号
     * @returns {boolean} 是否有效
     */
    static isValidPhone(phone) {
        if (!phone) return false;
        const pattern = /^1[3-9]\d{9}$/;
        return pattern.test(phone);
    }

    /**
     * 验证URL格式
     * @param {string} url - URL
     * @returns {boolean} 是否有效
     */
    static isValidURL(url) {
        if (!url) return false;
        try {
            const parsed = new URL(url);
            return ['http:', 'https:', 'mailto:'].includes(parsed.protocol);
        } catch {
            return false;
        }
    }

    /**
     * 验证IP地址
     * @param {string} ip - IP地址
     * @returns {boolean} 是否有效
     */
    static isValidIP(ip) {
        if (!ip) return false;
        const ipv4Pattern = /^(\d{1,3}\.){3}\d{1,3}$/;
        if (!ipv4Pattern.test(ip)) return false;
        
        const parts = ip.split('.');
        return parts.every(part => {
            const num = parseInt(part, 10);
            return num >= 0 && num <= 255;
        });
    }

    /**
     * 验证用户名格式（字母、数字、下划线，3-20个字符）
     * @param {string} username - 用户名
     * @returns {boolean} 是否有效
     */
    static isValidUsername(username) {
        if (!username) return false;
        const pattern = /^[a-zA-Z0-9_]{3,20}$/;
        return pattern.test(username);
    }

    /**
     * 验证密码强度
     * @param {string} password - 密码
     * @returns {object} {valid: boolean, strength: number, message: string}
     */
    static validatePasswordStrength(password) {
        if (!password) {
            return { valid: false, strength: 0, message: '密码不能为空' };
        }

        let strength = 0;
        const checks = {
            length: password.length >= 8,
            lower: /[a-z]/.test(password),
            upper: /[A-Z]/.test(password),
            digit: /\d/.test(password),
            special: /[!@#$%^&*(),.?":{}|<>]/.test(password)
        };

        Object.values(checks).forEach(passed => {
            if (passed) strength += 20;
        });

        let message = '';
        let valid = true;

        if (strength < 40) {
            message = '密码强度：弱';
            valid = false;
        } else if (strength < 60) {
            message = '密码强度：中等';
        } else if (strength < 80) {
            message = '密码强度：良好';
        } else {
            message = '密码强度：强';
        }

        return { valid, strength, message };
    }

    /**
     * 脱敏手机号
     * @param {string} phone - 手机号
     * @returns {string} 脱敏后的手机号
     */
    static maskPhone(phone) {
        if (!phone || phone.length < 11) return phone;
        return phone.substring(0, 3) + '****' + phone.substring(7);
    }

    /**
     * 脱敏邮箱
     * @param {string} email - 邮箱
     * @returns {string} 脱敏后的邮箱
     */
    static maskEmail(email) {
        if (!email || !email.includes('@')) return email;
        const [username, domain] = email.split('@');
        if (username.length <= 2) {
            return '**@' + domain;
        }
        return username.substring(0, 2) + '***@' + domain;
    }

    /**
     * 脱敏银行卡号
     * @param {string} cardNumber - 卡号
     * @returns {string} 脱敏后的卡号
     */
    static maskBankCard(cardNumber) {
        if (!cardNumber || cardNumber.length < 8) return cardNumber;
        return cardNumber.substring(0, 4) + ' **** **** ' + cardNumber.substring(cardNumber.length - 4);
    }

    /**
     * 检测潜在XSS攻击
     * @param {string} input - 输入内容
     * @returns {boolean} 是否包含XSS风险
     */
    static detectXSS(input) {
        if (!input) return false;
        
        const xssPatterns = [
            /<script[^>]*>/i,
            /javascript\s*:/i,
            /on\w+\s*=/i,
            /<\s*iframe/i,
            /<\s*object/i,
            /<\s*embed/i,
            /expression\s*\(/i,
            /data\s*:/i
        ];

        return xssPatterns.some(pattern => pattern.test(input));
    }

    /**
     * 过滤输入内容
     * @param {string} input - 输入内容
     * @param {object} options - 选项
     * @returns {string} 过滤后的内容
     */
    static sanitizeInput(input, options = {}) {
        if (!input) return '';
        
        let result = input;
        
        if (options.stripTags !== false) {
            result = this.stripDangerousTags(result);
        }
        
        if (options.escapeHTML) {
            result = this.escapeHTML(result);
        }
        
        if (options.maxLength) {
            result = result.substring(0, options.maxLength);
        }
        
        return result.trim();
    }

    /**
     * 生成随机字符串
     * @param {number} length - 长度
     * @param {string} charset - 字符集
     * @returns {string} 随机字符串
     */
    static generateRandomString(length = 16, charset = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789') {
        let result = '';
        const randomValues = new Uint32Array(length);
        crypto.getRandomValues(randomValues);
        
        for (let i = 0; i < length; i++) {
            result += charset[randomValues[i] % charset.length];
        }
        
        return result;
    }

    /**
     * 安全地设置HTML内容
     * @param {HTMLElement} element - DOM元素
     * @param {string} html - HTML内容
     */
    static safeSetHTML(element, html) {
        if (!element || !html) return;
        
        const sanitized = this.stripDangerousTags(html);
        element.textContent = sanitized;
    }

    /**
     * 清理表单数据
     * @param {HTMLFormElement} form - 表单元素
     * @returns {object} 清理后的表单数据
     */
    static sanitizeFormData(form) {
        const formData = new FormData(form);
        const data = {};
        
        for (const [key, value] of formData.entries()) {
            if (typeof value === 'string') {
                data[key] = this.sanitizeInput(value);
            } else {
                data[key] = value;
            }
        }
        
        return data;
    }
}

/**
 * CSRF Token管理器
 */
class CSRFTokenManager {
    constructor(options = {}) {
        this.tokenName = options.tokenName || '_csrf';
        this.headerName = options.headerName || 'X-CSRF-Token';
        this.cookieName = options.cookieName || 'csrf_token';
        this.token = null;
    }

    /**
     * 获取当前Token
     * @returns {string|null} CSRF Token
     */
    getToken() {
        if (this.token) {
            return this.token;
        }

        const metaToken = document.querySelector('meta[name="csrf-token"]');
        if (metaToken) {
            this.token = metaToken.content;
            return this.token;
        }

        const cookieValue = this.getCookie(this.cookieName);
        if (cookieValue) {
            this.token = cookieValue;
            return this.token;
        }

        return null;
    }

    /**
     * 设置Token
     * @param {string} token - CSRF Token
     */
    setToken(token) {
        this.token = token;
    }

    /**
     * 从Cookie获取值
     * @param {string} name - Cookie名称
     * @returns {string|null}
     */
    getCookie(name) {
        const value = `; ${document.cookie}`;
        const parts = value.split(`; ${name}=`);
        if (parts.length === 2) {
            return parts.pop().split(';').shift();
        }
        return null;
    }

    /**
     * 添加Token到请求头
     * @param {RequestInfo} request - Fetch请求
     * @returns {Request} 带Token的请求
     */
    addToRequest(request) {
        const token = this.getToken();
        if (!token) {
            console.warn('CSRF token not found');
            return request;
        }

        const headers = new Headers(request.headers || {});
        headers.set(this.headerName, token);

        return new Request(request, { headers });
    }

    /**
     * 添加Token到表单数据
     * @param {FormData} formData - 表单数据
     */
    addToFormData(formData) {
        const token = this.getToken();
        if (token) {
            formData.append(this.tokenName, token);
        }
    }

    /**
     * 从服务器获取新Token
     * @param {string} url - 获取Token的URL
     * @returns {Promise<string>} 新Token
     */
    async refreshToken(url = '/api/csrf-token') {
        try {
            const response = await fetch(url);
            if (response.ok) {
                const data = await response.json();
                if (data.token) {
                    this.setToken(data.token);
                    return data.token;
                }
            }
            throw new Error('Failed to refresh CSRF token');
        } catch (error) {
            console.error('Error refreshing CSRF token:', error);
            throw error;
        }
    }

    /**
     * 验证响应中的Token
     * @param {Response} response - Fetch响应
     * @returns {boolean} Token是否有效
     */
    validateResponse(response) {
        const newToken = response.headers.get(this.headerName);
        if (newToken) {
            this.setToken(newToken);
            return true;
        }
        return false;
    }
}

/**
 * 输入验证器
 */
class InputValidator {
    constructor() {
        this.rules = new Map();
        this.errors = new Map();
    }

    /**
     * 添加验证规则
     * @param {string} field - 字段名
     * @param {Array} fieldRules - 字段规则
     */
    addRule(field, fieldRules) {
        this.rules.set(field, fieldRules);
    }

    /**
     * 验证单个字段
     * @param {string} field - 字段名
     * @param {*} value - 字段值
     * @returns {string|null} 错误信息
     */
    validateField(field, value) {
        const rules = this.rules.get(field);
        if (!rules) return null;

        for (const rule of rules) {
            if (rule.required && (value === undefined || value === null || value === '')) {
                return rule.message || `${field}不能为空`;
            }

            if (rule.type === 'email' && value && !SecurityUtils.isValidEmail(value)) {
                return rule.message || `${field}邮箱格式不正确`;
            }

            if (rule.type === 'phone' && value && !SecurityUtils.isValidPhone(value)) {
                return rule.message || `${field}手机号格式不正确`;
            }

            if (rule.type === 'url' && value && !SecurityUtils.isValidURL(value)) {
                return rule.message || `${field}URL格式不正确`;
            }

            if (rule.type === 'ip' && value && !SecurityUtils.isValidIP(value)) {
                return rule.message || `${field}IP地址格式不正确`;
            }

            if (rule.minLength && value && value.length < rule.minLength) {
                return rule.message || `${field}长度不能少于${rule.minLength}个字符`;
            }

            if (rule.maxLength && value && value.length > rule.maxLength) {
                return rule.message || `${field}长度不能超过${rule.maxLength}个字符`;
            }

            if (rule.pattern && value && !rule.pattern.test(value)) {
                return rule.message || `${field}格式不正确`;
            }

            if (rule.custom && typeof rule.custom === 'function') {
                const result = rule.custom(value);
                if (!result) {
                    return rule.message || `${field}验证失败`;
                }
            }
        }

        return null;
    }

    /**
     * 验证所有字段
     * @param {object} data - 要验证的数据
     * @returns {boolean} 是否通过验证
     */
    validate(data) {
        this.errors.clear();
        let isValid = true;

        for (const [field, value] of Object.entries(data)) {
            const error = this.validateField(field, value);
            if (error) {
                this.errors.set(field, error);
                isValid = false;
            }
        }

        return isValid;
    }

    /**
     * 获取第一个错误
     * @returns {string|null}
     */
    getFirstError() {
        const errors = Array.from(this.errors.values());
        return errors.length > 0 ? errors[0] : null;
    }

    /**
     * 获取所有错误
     * @returns {Map}
     */
    getErrors() {
        return this.errors;
    }

    /**
     * 清除错误
     */
    clearErrors() {
        this.errors.clear();
    }
}

/**
 * 安全请求工具
 */
class SecureRequest {
    constructor(csrfManager) {
        this.csrfManager = csrfManager || new CSRFTokenManager();
    }

    /**
     * 发送安全GET请求
     * @param {string} url - 请求URL
     * @param {object} options - 选项
     * @returns {Promise}
     */
    async get(url, options = {}) {
        const request = new Request(url, {
            method: 'GET',
            credentials: options.credentials || 'same-origin',
            headers: options.headers || {}
        });

        return this.execute(request);
    }

    /**
     * 发送安全POST请求
     * @param {string} url - 请求URL
     * @param {object} data - 请求数据
     * @param {object} options - 选项
     * @returns {Promise}
     */
    async post(url, data, options = {}) {
        const headers = new Headers(options.headers || {});
        headers.set('Content-Type', 'application/json');

        const request = new Request(url, {
            method: 'POST',
            credentials: options.credentials || 'same-origin',
            headers,
            body: JSON.stringify(data)
        });

        return this.execute(request);
    }

    /**
     * 发送安全表单请求
     * @param {string} url - 请求URL
     * @param {FormData} formData - 表单数据
     * @param {object} options - 选项
     * @returns {Promise}
     */
    async postForm(url, formData, options = {}) {
        this.csrfManager.addToFormData(formData);

        const request = new Request(url, {
            method: 'POST',
            credentials: options.credentials || 'same-origin',
            body: formData
        });

        return this.execute(request);
    }

    /**
     * 执行请求
     * @param {Request} request - 请求对象
     * @returns {Promise}
     */
    async execute(request) {
        const securedRequest = this.csrfManager.addToRequest(request);

        try {
            const response = await fetch(securedRequest);
            
            this.csrfManager.validateResponse(response);

            if (!response.ok) {
                const error = await response.json().catch(() => ({}));
                throw new Error(error.message || `HTTP ${response.status}`);
            }

            return response.json();
        } catch (error) {
            if (error.name === 'TypeError') {
                throw new Error('网络请求失败，请检查网络连接');
            }
            throw error;
        }
    }
}

/**
 * XSS防护中间件
 */
class XSSProtection {
    constructor(options = {}) {
        this.enabled = options.enabled !== false;
        this.strict = options.strict || false;
    }

    /**
     * 防护所有表单输入
     */
    protectForms() {
        if (!this.enabled) return;

        document.querySelectorAll('form').forEach(form => {
            form.addEventListener('submit', (e) => {
                const inputs = form.querySelectorAll('input, textarea');
                inputs.forEach(input => {
                    if (input.type !== 'submit' && input.type !== 'button') {
                        input.value = SecurityUtils.sanitizeInput(input.value);
                    }
                });
            });
        });
    }

    /**
     * 防护特定元素
     * @param {string} selector - CSS选择器
     */
    protectElements(selector) {
        if (!this.enabled) return;

        document.querySelectorAll(selector).forEach(element => {
            const originalSetter = Object.getOwnPropertyDescriptor(
                element.tagName === 'INPUT' ? HTMLInputElement.prototype : HTMLElement.prototype,
                'textContent'
            );

            Object.defineProperty(element, 'textContent', {
                set(value) {
                    originalSetter.set.call(this, SecurityUtils.sanitizeInput(value));
                },
                get: originalSetter.get
            });
        });
    }

    /**
     * 阻止危险事件
     */
    blockDangerousEvents() {
        if (!this.enabled || !this.strict) return;

        const dangerousEvents = ['click', 'mouseover', 'focus', 'blur', 'change', 'submit'];

        dangerousEvents.forEach(eventType => {
            document.addEventListener(eventType, (e) => {
                const target = e.target;
                if (target.hasAttribute(`on${eventType}`)) {
                    const handler = target.getAttribute(`on${eventType}`);
                    if (SecurityUtils.detectXSS(handler)) {
                        e.preventDefault();
                        console.warn(`Blocked dangerous ${eventType} handler`);
                    }
                }
            }, true);
        });
    }
}

// 导出到全局
if (typeof window !== 'undefined') {
    window.SecurityUtils = SecurityUtils;
    window.CSRFTokenManager = CSRFTokenManager;
    window.InputValidator = InputValidator;
    window.SecureRequest = SecureRequest;
    window.XSSProtection = XSSProtection;
}

export {
    SecurityUtils,
    CSRFTokenManager,
    InputValidator,
    SecureRequest,
    XSSProtection
};
