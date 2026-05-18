/**
 * @fileoverview 墨盾验证 - 统一UI组件库
 * @description 提供验证码、按钮、模态框、提示等UI组件，支持主题定制和无障碍访问
 * @version 1.0.0
 * @license MIT
 */

(function(global) {
    'use strict';

    /**
     * @namespace UIComponents
     * @description 统一UI组件库命名空间
     */
    const UIComponents = {};

    /**
     * 设计规范常量
     * @namespace DesignTokens
     */
    const DesignTokens = {
        colors: {
            primary: '#c9a96e',
            primaryLight: '#d4b87a',
            primaryDark: '#b8954f',
            success: '#28a745',
            successLight: 'rgba(40, 167, 69, 0.1)',
            danger: '#dc3545',
            dangerLight: 'rgba(220, 53, 69, 0.1)',
            warning: '#ffc107',
            info: '#17a2b8',
            dark: '#1a1a2e',
            darkSecondary: '#2d2d44',
            light: '#f8f9fa',
            white: '#ffffff',
            gray100: '#f8f9fa',
            gray200: '#e9ecef',
            gray300: '#dee2e6',
            gray400: '#ced4da',
            gray500: '#adb5bd',
            gray600: '#6c757d',
            gray700: '#495057',
            gray800: '#343a40',
            gray900: '#212529',
            textPrimary: '#1a1a2e',
            textSecondary: '#6c757d',
            textMuted: '#adb5bd',
            border: 'rgba(201, 169, 110, 0.2)',
            overlay: 'rgba(26, 26, 46, 0.8)'
        },
        fonts: {
            familyBase: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif",
            familyMono: "'SFMono-Regular', Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace",
            familySerif: "'Noto Serif SC', 'STSong', 'SimSun', serif",
            sizeXs: '0.75rem',
            sizeSm: '0.875rem',
            sizeBase: '1rem',
            sizeLg: '1.125rem',
            sizeXl: '1.25rem',
            size2xl: '1.5rem',
            size3xl: '1.875rem',
            size4xl: '2.25rem',
            weightNormal: 400,
            weightMedium: 500,
            weightSemiBold: 600,
            weightBold: 700
        },
        spacing: {
            xs: '0.25rem',
            sm: '0.5rem',
            md: '1rem',
            lg: '1.5rem',
            xl: '2rem',
            xxl: '3rem',
            xxxl: '4rem'
        },
        borderRadius: {
            sm: '0.25rem',
            md: '0.375rem',
            lg: '0.5rem',
            xl: '0.75rem',
            xxl: '1rem',
            full: '9999px'
        },
        shadows: {
            sm: '0 1px 2px rgba(0, 0, 0, 0.05)',
            md: '0 4px 6px rgba(0, 0, 0, 0.1)',
            lg: '0 10px 15px rgba(0, 0, 0, 0.1)',
            xl: '0 20px 25px rgba(0, 0, 0, 0.15)',
            primary: '0 4px 12px rgba(201, 169, 110, 0.3)',
            primaryLg: '0 8px 20px rgba(201, 169, 110, 0.4)'
        },
        transitions: {
            fast: '0.15s ease',
            normal: '0.3s ease',
            slow: '0.5s ease',
            bounce: '0.3s cubic-bezier(0.4, 0, 0.2, 1)'
        },
        zIndex: {
            dropdown: 1000,
            sticky: 1020,
            fixed: 1030,
            modalBackdrop: 1040,
            modal: 1050,
            popover: 1060,
            tooltip: 1070,
            toast: 1090
        }
    };

    /**
     * @class ThemeManager
     * @description 主题管理器，支持CSS变量和预设主题切换
     */
    class ThemeManager {
        /**
         * @constructor
         * @param {Object} options - 配置选项
         * @param {string} [options.defaultTheme='light'] - 默认主题
         * @param {string} [options.storageKey='ui-theme'] - 本地存储键名
         */
        constructor(options = {}) {
            this.defaultTheme = options.defaultTheme || 'light';
            this.storageKey = options.storageKey || 'ui-theme';
            this.currentTheme = this.getStoredTheme() || this.defaultTheme;
            this.listeners = [];
            this.cssVariables = {};
            this.init();
        }

        /**
         * 初始化主题系统
         * @private
         */
        init() {
            this.injectCSSVariables();
            this.applyTheme(this.currentTheme);
            this.watchSystemPreference();
        }

        /**
         * 注入CSS变量到根元素
         * @private
         */
        injectCSSVariables() {
            const root = document.documentElement;
            const tokens = DesignTokens.colors;

            for (const [key, value] of Object.entries(tokens)) {
                root.style.setProperty(`--ui-color-${this.toKebabCase(key)}`, value);
            }
            root.style.setProperty('--ui-font-base', DesignTokens.fonts.familyBase);
            root.style.setProperty('--ui-font-mono', DesignTokens.fonts.familyMono);
            root.style.setProperty('--ui-font-serif', DesignTokens.fonts.familySerif);
        }

        /**
         * 驼峰转连字符
         * @private
         * @param {string} str - 驼峰字符串
         * @returns {string} 连字符字符串
         */
        toKebabCase(str) {
            return str.replace(/([a-z])([A-Z])/g, '$1-$2').toLowerCase();
        }

        /**
         * 获取存储的主题
         * @returns {string|null} 存储的主题名称
         */
        getStoredTheme() {
            try {
                return localStorage.getItem(this.storageKey);
            } catch (e) {
                return null;
            }
        }

        /**
         * 存储主题
         * @param {string} theme - 主题名称
         */
        storeTheme(theme) {
            try {
                localStorage.setItem(this.storageKey, theme);
            } catch (e) {
                console.warn('Theme storage failed:', e);
            }
        }

        /**
         * 应用主题
         * @param {string} theme - 主题名称
         */
        applyTheme(theme) {
            this.currentTheme = theme;
            this.storeTheme(theme);

            document.documentElement.setAttribute('data-ui-theme', theme);

            const themeConfig = this.getThemeConfig(theme);
            Object.entries(themeConfig).forEach(([key, value]) => {
                document.documentElement.style.setProperty(key, value);
            });

            this.notifyListeners(theme);
        }

        /**
         * 获取主题配置
         * @private
         * @param {string} theme - 主题名称
         * @returns {Object} CSS变量配置
         */
        getThemeConfig(theme) {
            const configs = {
                light: {
                    '--ui-bg-primary': DesignTokens.colors.white,
                    '--ui-bg-secondary': DesignTokens.colors.gray100,
                    '--ui-text-primary': DesignTokens.colors.textPrimary,
                    '--ui-text-secondary': DesignTokens.colors.textSecondary,
                    '--ui-border-color': DesignTokens.colors.border,
                    '--ui-shadow': DesignTokens.shadows.md
                },
                dark: {
                    '--ui-bg-primary': DesignTokens.colors.darkSecondary,
                    '--ui-bg-secondary': DesignTokens.colors.dark,
                    '--ui-text-primary': DesignTokens.colors.white,
                    '--ui-text-secondary': DesignTokens.colors.gray400,
                    '--ui-border-color': 'rgba(255, 255, 255, 0.1)',
                    '--ui-shadow': '0 4px 6px rgba(0, 0, 0, 0.3)'
                },
                highContrast: {
                    '--ui-bg-primary': DesignTokens.colors.white,
                    '--ui-bg-secondary': DesignTokens.colors.gray900,
                    '--ui-text-primary': DesignTokens.colors.black,
                    '--ui-text-secondary': DesignTokens.colors.gray900,
                    '--ui-border-color': DesignTokens.colors.black,
                    '--ui-color-primary': '#ffd700',
                    '--ui-color-success': '#00ff00',
                    '--ui-color-danger': '#ff0000'
                }
            };

            return configs[theme] || configs.light;
        }

        /**
         * 监听系统偏好变化
         * @private
         */
        watchSystemPreference() {
            const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
            mediaQuery.addEventListener('change', (e) => {
                if (!this.getStoredTheme()) {
                    this.applyTheme(e.matches ? 'dark' : 'light');
                }
            });
        }

        /**
         * 切换主题
         * @param {string} [theme] - 目标主题，不传则切换
         */
        toggle(theme) {
            if (theme) {
                this.applyTheme(theme);
            } else {
                const nextTheme = this.currentTheme === 'light' ? 'dark' : 'light';
                this.applyTheme(nextTheme);
            }
        }

        /**
         * 获取当前主题
         * @returns {string} 当前主题名称
         */
        getCurrentTheme() {
            return this.currentTheme;
        }

        /**
         * 添加主题变化监听器
         * @param {Function} callback - 回调函数
         */
        addListener(callback) {
            if (typeof callback === 'function') {
                this.listeners.push(callback);
            }
        }

        /**
         * 移除主题变化监听器
         * @param {Function} callback - 回调函数
         */
        removeListener(callback) {
            this.listeners = this.listeners.filter(cb => cb !== callback);
        }

        /**
         * 通知监听器
         * @private
         * @param {string} theme - 新主题
         */
        notifyListeners(theme) {
            this.listeners.forEach(callback => {
                try {
                    callback(theme);
                } catch (e) {
                    console.error('Theme listener error:', e);
                }
            });
        }
    }

    UIComponents.ThemeManager = ThemeManager;

    /**
     * @class BaseComponent
     * @description 组件基类，提供通用功能
     */
    class BaseComponent {
        /**
         * @constructor
         * @param {HTMLElement|string} container - 容器元素或选择器
         * @param {Object} options - 配置选项
         */
        constructor(container, options = {}) {
            this.container = typeof container === 'string'
                ? document.querySelector(container)
                : container;

            if (!this.container) {
                throw new Error('Container element not found');
            }

            this.options = { ...this.getDefaultOptions(), ...options };
            this.isInitialized = false;
            this.eventHandlers = {};
        }

        /**
         * 获取默认选项
         * @protected
         * @returns {Object} 默认选项
         */
        getDefaultOptions() {
            return {};
        }

        /**
         * 初始化组件
         * @protected
         */
        init() {
            this.isInitialized = true;
        }

        /**
         * 销毁组件
         */
        destroy() {
            this.removeAllEventListeners();
            this.isInitialized = false;
        }

        /**
         * 添加事件监听
         * @protected
         * @param {string} event - 事件名
         * @param {Function} handler - 处理函数
         * @param {Object} options - 事件选项
         */
        on(event, handler, options = {}) {
            if (!this.eventHandlers[event]) {
                this.eventHandlers[event] = [];
            }
            this.eventHandlers[event].push(handler);
            this.container.addEventListener(event, handler, options);
        }

        /**
         * 移除事件监听
         * @protected
         * @param {string} event - 事件名
         * @param {Function} handler - 处理函数
         */
        off(event, handler) {
            if (handler) {
                this.container.removeEventListener(event, handler);
                if (this.eventHandlers[event]) {
                    this.eventHandlers[event] = this.eventHandlers[event].filter(h => h !== handler);
                }
            }
        }

        /**
         * 移除所有事件监听
         * @protected
         */
        removeAllEventListeners() {
            Object.entries(this.eventHandlers).forEach(([event, handlers]) => {
                handlers.forEach(handler => {
                    this.container.removeEventListener(event, handler);
                });
            });
            this.eventHandlers = {};
        }

        /**
         * 触发事件
         * @protected
         * @param {string} event - 事件名
         * @param {*} data - 事件数据
         */
        emit(event, data) {
            const customEvent = new CustomEvent(event, {
                detail: data,
                bubbles: true,
                cancelable: true
            });
            this.container.dispatchEvent(customEvent);
        }

        /**
         * 创建带样式标签
         * @protected
         * @param {string} tag - HTML标签
         * @param {Object} attrs - 属性
         * @param {string} [text=''] - 文本内容
         * @returns {HTMLElement} 创建的元素
         */
        createElement(tag, attrs = {}, text = '') {
            const el = document.createElement(tag);
            Object.entries(attrs).forEach(([key, value]) => {
                if (key === 'className') {
                    el.className = value;
                } else if (key === 'style' && typeof value === 'object') {
                    Object.assign(el.style, value);
                } else if (key.startsWith('data-') || key.startsWith('aria-') || key === 'role') {
                    el.setAttribute(key, value);
                } else {
                    el[key] = value;
                }
            });
            if (text) {
                el.textContent = text;
            }
            return el;
        }
    }

    /**
     * @class CaptchaUI
     * @description 验证码UI组件，提供加载、错误、成功等状态的UI展示
     * @extends BaseComponent
     */
    class CaptchaUI extends BaseComponent {
        /**
         * @constructor
         * @param {HTMLElement|string} container - 容器元素或选择器
         * @param {Object} options - 配置选项
         * @param {string} [options.type='slider'] - 验证码类型
         * @param {Function} [options.onSuccess] - 验证成功回调
         * @param {Function} [options.onError] - 验证失败回调
         * @param {Function} [options.onRefresh] - 刷新回调
         * @param {string} [options.language='zh-CN'] - 语言
         */
        constructor(container, options = {}) {
            super(container, options);
            this.type = options.type || 'slider';
            this.state = 'idle';
            this.loadingTimer = null;
            this.animationDuration = 300;
            this.init();
        }

        /**
         * @inheritdoc
         */
        getDefaultOptions() {
            return {
                type: 'slider',
                language: 'zh-CN',
                showLoading: true,
                loadingText: '加载中...',
                errorText: '验证失败',
                successText: '验证成功',
                retryText: '重试',
                refreshText: '刷新',
                onSuccess: null,
                onError: null,
                onRefresh: null
            };
        }

        /**
         * @inheritdoc
         */
        init() {
            super.init();
            this.render();
            this.bindEvents();
            this.emit('captcha:init', { type: this.type });
        }

        /**
         * 渲染组件UI
         * @private
         */
        render() {
            this.container.innerHTML = '';
            this.container.setAttribute('role', 'application');
            this.container.setAttribute('aria-label', `${this.getTypeName()} 验证区域`);

            const wrapper = this.createElement('div', {
                className: 'captcha-ui-wrapper',
                style: {
                    position: 'relative',
                    minHeight: '160px',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center'
                }
            });

            this.contentArea = this.createElement('div', {
                className: 'captcha-ui-content',
                style: {
                    textAlign: 'center',
                    padding: '1rem'
                }
            });

            this.iconElement = this.createElement('i', {
                className: 'fas fa-shield-alt',
                style: {
                    fontSize: '2.5rem',
                    color: DesignTokens.colors.primary,
                    marginBottom: '0.5rem'
                },
                'aria-hidden': 'true'
            });

            this.textElement = this.createElement('p', {
                className: 'captcha-ui-text',
                style: {
                    margin: '0.5rem 0 0',
                    color: DesignTokens.colors.textSecondary,
                    fontSize: DesignTokens.fonts.sizeSm
                }
            }, this.options.loadingText);

            this.contentArea.appendChild(this.iconElement);
            this.contentArea.appendChild(this.textElement);

            this.spinnerElement = this.createElement('div', {
                className: 'captcha-ui-spinner',
                style: {
                    display: 'none',
                    position: 'absolute',
                    top: '50%',
                    left: '50%',
                    transform: 'translate(-50%, -50%)'
                }
            });

            const spinner = this.createElement('div', {
                style: {
                    width: '2rem',
                    height: '2rem',
                    border: '0.2rem solid rgba(201, 169, 110, 0.2)',
                    borderTopColor: DesignTokens.colors.primary,
                    borderRadius: '50%',
                    animation: 'captcha-ui-spin 0.8s linear infinite'
                }
            });

            this.spinnerElement.appendChild(spinner);
            wrapper.appendChild(this.contentArea);
            wrapper.appendChild(this.spinnerElement);
            this.container.appendChild(wrapper);

            this.injectKeyframes();
        }

        /**
         * 注入CSS动画
         * @private
         */
        injectKeyframes() {
            if (document.getElementById('captcha-ui-keyframes')) return;

            const style = document.createElement('style');
            style.id = 'captcha-ui-keyframes';
            style.textContent = `
                @keyframes captcha-ui-spin {
                    to { transform: rotate(360deg); }
                }
                @keyframes captcha-ui-fade-in {
                    from { opacity: 0; transform: translateY(10px); }
                    to { opacity: 1; transform: translateY(0); }
                }
                @keyframes captcha-ui-shake {
                    0%, 100% { transform: translateX(0); }
                    25% { transform: translateX(-5px); }
                    75% { transform: translateX(5px); }
                }
                @keyframes captcha-ui-bounce {
                    0%, 100% { transform: translateY(0); }
                    50% { transform: translateY(-10px); }
                }
            `;
            document.head.appendChild(style);
        }

        /**
         * 绑定事件
         * @private
         */
        bindEvents() {
            this.on('captcha:loading', () => this.showLoading());
            this.on('captcha:loaded', () => this.hideLoading());
            this.on('captcha:success', (e) => this.showSuccess(e.detail));
            this.on('captcha:error', (e) => this.showError(e.detail));
        }

        /**
         * 获取类型名称
         * @private
         * @returns {string} 类型名称
         */
        getTypeName() {
            const names = {
                slider: '滑块',
                click: '点选',
                passive: '无感',
                rotation: '旋转',
                gesture: '手势'
            };
            return names[this.type] || '验证码';
        }

        /**
         * 显示加载状态
         * @param {string} [text] - 加载文本
         */
        showLoading(text) {
            if (!this.options.showLoading) return;

            this.state = 'loading';
            this.contentArea.style.opacity = '0.5';
            this.spinnerElement.style.display = 'block';
            this.textElement.textContent = text || this.options.loadingText;

            this.container.setAttribute('aria-busy', 'true');
            this.container.setAttribute('aria-live', 'polite');
        }

        /**
         * 隐藏加载状态
         */
        hideLoading() {
            this.state = 'idle';
            this.contentArea.style.opacity = '1';
            this.spinnerElement.style.display = 'none';
            this.container.setAttribute('aria-busy', 'false');
        }

        /**
         * 显示成功状态
         * @param {Object} data - 成功数据
         */
        showSuccess(data) {
            this.state = 'success';
            this.hideLoading();

            this.iconElement.className = 'fas fa-check-circle';
            this.iconElement.style.color = DesignTokens.colors.success;
            this.textElement.textContent = this.options.successText;

            this.container.classList.add('captcha-ui-success');
            this.animateElement(this.container, 'captcha-ui-bounce');

            if (typeof this.options.onSuccess === 'function') {
                this.options.onSuccess(data);
            }

            this.emit('success', data);
        }

        /**
         * 显示错误状态
         * @param {Object} data - 错误数据
         */
        showError(data) {
            this.state = 'error';
            this.hideLoading();

            this.iconElement.className = 'fas fa-times-circle';
            this.iconElement.style.color = DesignTokens.colors.danger;
            this.textElement.textContent = data?.message || this.options.errorText;

            this.container.classList.add('captcha-ui-error');
            this.animateElement(this.container, 'captcha-ui-shake');

            setTimeout(() => {
                this.container.classList.remove('captcha-ui-error');
            }, this.animationDuration);

            if (typeof this.options.onError === 'function') {
                this.options.onError(data);
            }

            this.emit('error', data);
        }

        /**
         * 重置状态
         */
        reset() {
            this.state = 'idle';
            this.hideLoading();

            this.iconElement.className = 'fas fa-shield-alt';
            this.iconElement.style.color = DesignTokens.colors.primary;
            this.textElement.textContent = this.options.loadingText;

            this.container.classList.remove('captcha-ui-success', 'captcha-ui-error');

            if (typeof this.options.onRefresh === 'function') {
                this.options.onRefresh();
            }

            this.emit('refresh');
        }

        /**
         * 元素动画
         * @private
         * @param {HTMLElement} el - 元素
         * @param {string} animationName - 动画名称
         */
        animateElement(el, animationName) {
            el.style.animation = `${animationName} ${this.animationDuration}ms ease`;
            setTimeout(() => {
                el.style.animation = '';
            }, this.animationDuration);
        }

        /**
         * 设置语言
         * @param {string} lang - 语言代码
         */
        setLanguage(lang) {
            this.options.language = lang;
            const translations = {
                'zh-CN': {
                    loading: '加载中...',
                    error: '验证失败',
                    success: '验证成功',
                    retry: '重试',
                    refresh: '刷新'
                },
                'en-US': {
                    loading: 'Loading...',
                    error: 'Verification failed',
                    success: 'Verification successful',
                    retry: 'Retry',
                    refresh: 'Refresh'
                }
            };

            const trans = translations[lang] || translations['zh-CN'];
            Object.assign(this.options, trans);

            if (this.state === 'loading') {
                this.textElement.textContent = this.options.loadingText;
            } else if (this.state === 'error') {
                this.textElement.textContent = this.options.errorText;
            }
        }

        /**
         * @inheritdoc
         */
        destroy() {
            this.container.innerHTML = '';
            super.destroy();
        }
    }

    UIComponents.CaptchaUI = CaptchaUI;

    /**
     * @class SliderCaptcha
     * @description 滑块验证码组件
     * @extends BaseComponent
     */
    class SliderCaptcha extends BaseComponent {
        /**
         * @constructor
         * @param {HTMLElement|string} container - 容器元素或选择器
         * @param {Object} options - 配置选项
         * @param {string} [options.apiBase='/api/v1'] - API基础路径
         * @param {Function} [options.onSuccess] - 成功回调
         * @param {Function} [options.onError] - 失败回调
         */
        constructor(container, options = {}) {
            super(container, options);
            this.apiBase = options.apiBase || '/api/v1';
            this.isDragging = false;
            this.startX = 0;
            this.currentX = 0;
            this.startTime = 0;
            this.trajectoryData = [];
            this.challengeId = '';
            this.targetPosition = 0;
            this.sliderWidth = 0;
            this.init();
        }

        /**
         * @inheritdoc
         */
        getDefaultOptions() {
            return {
                apiBase: '/api/v1',
                sliderHeight: 44,
                tolerance: 10,
                maxAttempts: 3,
                animationDuration: 300,
                onSuccess: null,
                onError: null,
                onRefresh: null
            };
        }

        /**
         * @inheritdoc
         */
        init() {
            super.init();
            this.loadCaptcha();
            this.bindEvents();
        }

        /**
         * 加载验证码
         * @private
         */
        async loadCaptcha() {
            this.emit('loading');

            try {
                const response = await fetch(`${this.apiBase}/captcha/slider?t=${Date.now()}`);
                const data = await response.json();

                if (data.code === 0 || data.success) {
                    const captchaData = data.data || data;
                    this.challengeId = captchaData.challenge_id;
                    this.targetPosition = captchaData.target_position || 0;
                    this.render(captchaData);
                    this.emit('loaded');
                } else {
                    throw new Error(data.message || 'Failed to load captcha');
                }
            } catch (error) {
                console.error('Captcha load error:', error);
                this.emit('error', { message: '加载失败，请重试' });
            }
        }

        /**
         * 渲染组件
         * @private
         * @param {Object} data - 验证码数据
         */
        render(data) {
            this.container.innerHTML = '';
            this.container.setAttribute('role', 'slider');
            this.container.setAttribute('aria-label', '滑块验证，请拖动滑块完成验证');
            this.container.setAttribute('aria-valuemin', 0);
            this.container.setAttribute('aria-valuemax', 100);
            this.container.setAttribute('aria-valuenow', 0);

            const wrapper = this.createElement('div', {
                className: 'slider-captcha-wrapper',
                style: {
                    width: '100%',
                    maxWidth: '320px'
                }
            });

            const imageContainer = this.createElement('div', {
                className: 'slider-captcha-image-container',
                style: {
                    position: 'relative',
                    borderRadius: DesignTokens.borderRadius.lg,
                    overflow: 'hidden',
                    background: DesignTokens.colors.gray200
                }
            });

            const backgroundImage = this.createElement('img', {
                className: 'slider-captcha-bg',
                src: data.background_image || data.bgImage || '',
                alt: '验证码背景图',
                style: {
                    width: '100%',
                    display: 'block'
                }
            });

            const sliderImage = this.createElement('img', {
                className: 'slider-captcha-slider',
                src: data.slider_image || data.sliderImage || '',
                alt: '滑块图片',
                style: {
                    position: 'absolute',
                    top: 0,
                    left: '0',
                    height: '100%',
                    cursor: 'grab'
                }
            });

            this.sliderImageElement = sliderImage;
            this.backgroundImageElement = backgroundImage;

            imageContainer.appendChild(backgroundImage);
            imageContainer.appendChild(sliderImage);
            wrapper.appendChild(imageContainer);

            const sliderContainer = this.createElement('div', {
                className: 'slider-captcha-container',
                style: {
                    position: 'relative',
                    height: `${this.options.sliderHeight}px`,
                    background: DesignTokens.colors.gray100,
                    borderRadius: DesignTokens.borderRadius.full,
                    marginTop: '0.75rem',
                    boxShadow: `inset 0 2px 4px ${DesignTokens.colors.gray300}`,
                    overflow: 'hidden'
                },
                tabindex: '0',
                role: 'slider',
                'aria-label': '拖动滑块'
            });

            this.sliderContainerElement = sliderContainer;

            const track = this.createElement('div', {
                className: 'slider-captcha-track',
                style: {
                    position: 'absolute',
                    left: '4px',
                    top: '4px',
                    height: 'calc(100% - 8px)',
                    width: '0',
                    background: `linear-gradient(90deg, ${DesignTokens.colors.primary}, ${DesignTokens.colors.primaryLight})`,
                    borderRadius: DesignTokens.borderRadius.full,
                    transition: 'width 0.05s linear'
                }
            });

            const sliderButton = this.createElement('div', {
                className: 'slider-captcha-button',
                style: {
                    position: 'absolute',
                    left: '4px',
                    top: '4px',
                    width: `${this.options.sliderHeight - 8}px`,
                    height: `${this.options.sliderHeight - 8}px`,
                    background: DesignTokens.colors.white,
                    borderRadius: '50%',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    boxShadow: DesignTokens.shadows.md,
                    cursor: 'grab',
                    color: DesignTokens.colors.primary,
                    fontSize: '1rem',
                    transition: 'left 0.05s linear'
                },
                'aria-hidden': 'true'
            });

            const arrowIcon = this.createElement('i', {
                className: 'fas fa-arrow-right',
                'aria-hidden': 'true'
            });

            sliderButton.appendChild(arrowIcon);
            this.sliderButtonElement = sliderButton;

            const hintText = this.createElement('span', {
                className: 'slider-captcha-hint',
                style: {
                    position: 'absolute',
                    width: '100%',
                    textAlign: 'center',
                    lineHeight: `${this.options.sliderHeight}px`,
                    fontSize: DesignTokens.fonts.sizeSm,
                    color: DesignTokens.colors.gray600,
                    pointerEvents: 'none'
                }
            }, '拖动滑块完成验证');

            this.hintTextElement = hintText;

            sliderContainer.appendChild(track);
            sliderContainer.appendChild(sliderButton);
            sliderContainer.appendChild(hintText);
            wrapper.appendChild(sliderContainer);

            const refreshButton = this.createElement('button', {
                className: 'slider-captcha-refresh',
                type: 'button',
                style: {
                    position: 'absolute',
                    right: '8px',
                    top: '50%',
                    transform: 'translateY(-50%)',
                    background: 'transparent',
                    border: 'none',
                    cursor: 'pointer',
                    color: DesignTokens.colors.gray600,
                    fontSize: '1rem',
                    padding: '4px'
                },
                'aria-label': '刷新验证码'
            });

            const refreshIcon = this.createElement('i', {
                className: 'fas fa-sync-alt',
                'aria-hidden': 'true'
            });

            refreshButton.appendChild(refreshIcon);
            sliderContainer.appendChild(refreshButton);

            this.trackElement = track;
            this.container.appendChild(wrapper);
        }

        /**
         * 绑定事件
         * @private
         */
        bindEvents() {
            this.startHandler = this.handleDragStart.bind(this);
            this.moveHandler = this.handleDragMove.bind(this);
            this.endHandler = this.handleDragEnd.bind(this);

            this.sliderButtonElement.addEventListener('mousedown', this.startHandler);
            this.sliderButtonElement.addEventListener('touchstart', this.startHandler, { passive: false });

            this.sliderContainerElement.addEventListener('mousedown', (e) => {
                if (e.target === this.sliderContainerElement) {
                    this.startHandler(e);
                }
            });

            document.addEventListener('mousemove', this.moveHandler);
            document.addEventListener('touchmove', this.moveHandler, { passive: false });

            document.addEventListener('mouseup', this.endHandler);
            document.addEventListener('touchend', this.endHandler);

            this.sliderContainerElement.addEventListener('keydown', (e) => {
                if (e.key === 'ArrowRight') {
                    e.preventDefault();
                    this.adjustSlider(5);
                } else if (e.key === 'ArrowLeft') {
                    e.preventDefault();
                    this.adjustSlider(-5);
                } else if (e.key === 'Enter') {
                    e.preventDefault();
                    this.submitVerification();
                }
            });

            const refreshBtn = this.container.querySelector('.slider-captcha-refresh');
            if (refreshBtn) {
                refreshBtn.addEventListener('click', () => this.refresh());
            }
        }

        /**
         * 处理拖动开始
         * @private
         * @param {Event} e - 事件对象
         */
        handleDragStart(e) {
            if (this.state === 'success') return;

            this.isDragging = true;
            this.startX = e.type === 'mousedown' ? e.clientX : e.touches[0].clientX;
            this.currentX = this.startX;
            this.startTime = Date.now();
            this.trajectoryData = [{
                x: 0,
                y: 0,
                t: 0
            }];

            this.sliderButtonElement.style.cursor = 'grabbing';
            this.sliderButtonElement.style.transition = 'none';
            this.trackElement.style.transition = 'none';

            e.preventDefault();
        }

        /**
         * 处理拖动移动
         * @private
         * @param {Event} e - 事件对象
         */
        handleDragMove(e) {
            if (!this.isDragging) return;

            const clientX = e.type === 'mousemove' ? e.clientX : e.touches[0].clientX;
            const deltaX = clientX - this.startX;
            const maxWidth = this.sliderContainerElement.offsetWidth - this.sliderButtonElement.offsetWidth - 8;

            this.currentX = Math.max(4, Math.min(deltaX + 4, maxWidth + 4));
            const progress = (this.currentX - 4) / maxWidth;

            this.sliderButtonElement.style.left = `${this.currentX}px`;
            this.trackElement.style.width = `${this.currentX}px`;

            const sliderOffset = progress * this.backgroundImageElement.offsetWidth * 0.7;
            this.sliderImageElement.style.left = `${sliderOffset}px`;

            this.hintTextElement.style.opacity = '0';
            this.container.setAttribute('aria-valuenow', Math.round(progress * 100));

            this.trajectoryData.push({
                x: deltaX,
                y: 0,
                t: Date.now() - this.startTime
            });

            e.preventDefault();
        }

        /**
         * 处理拖动结束
         * @private
         * @param {Event} e - 事件对象
         */
        handleDragEnd(e) {
            if (!this.isDragging) return;

            this.isDragging = false;
            this.sliderButtonElement.style.cursor = 'grab';
            this.sliderButtonElement.style.transition = `left ${this.options.animationDuration}ms ease`;
            this.trackElement.style.transition = `width ${this.options.animationDuration}ms ease`;

            this.submitVerification();
        }

        /**
         * 调整滑块位置
         * @private
         * @param {number} delta - 调整量
         */
        adjustSlider(delta) {
            const maxWidth = this.sliderContainerElement.offsetWidth - this.sliderButtonElement.offsetWidth - 8;
            const currentLeft = parseFloat(this.sliderButtonElement.style.left) || 4;
            const newLeft = Math.max(4, Math.min(currentLeft + delta, maxWidth + 4));

            this.sliderButtonElement.style.left = `${newLeft}px`;
            this.trackElement.style.width = `${newLeft}px`;

            const progress = (newLeft - 4) / maxWidth;
            const sliderOffset = progress * this.backgroundImageElement.offsetWidth * 0.7;
            this.sliderImageElement.style.left = `${sliderOffset}px`;

            this.container.setAttribute('aria-valuenow', Math.round(progress * 100));
        }

        /**
         * 提交验证
         * @private
         */
        async submitVerification() {
            if (!this.challengeId) return;

            const maxWidth = this.sliderContainerElement.offsetWidth - this.sliderButtonElement.offsetWidth - 8;
            const progress = (this.currentX - 4) / maxWidth;
            const sliderPosition = Math.round(progress * 100);

            this.emit('verifying');

            try {
                const response = await fetch(`${this.apiBase}/captcha/slider/verify`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        challenge_id: this.challengeId,
                        slider_position: sliderPosition,
                        trajectory: this.trajectoryData
                    })
                });

                const data = await response.json();

                if (data.code === 0 || data.success) {
                    this.state = 'success';
                    this.showSuccess();
                } else {
                    this.handleError();
                }
            } catch (error) {
                console.error('Verification error:', error);
                this.handleError();
            }
        }

        /**
         * 显示成功状态
         * @private
         */
        showSuccess() {
            const icon = this.sliderButtonElement.querySelector('i');
            if (icon) {
                icon.className = 'fas fa-check';
            }
            this.sliderButtonElement.style.background = DesignTokens.colors.success;
            this.sliderButtonElement.style.color = DesignTokens.colors.white;

            this.hintTextElement.textContent = '验证成功';
            this.hintTextElement.style.color = DesignTokens.colors.success;
            this.hintTextElement.style.opacity = '1';

            this.container.setAttribute('aria-valuenow', 100);

            if (typeof this.options.onSuccess === 'function') {
                this.options.onSuccess({
                    challenge_id: this.challengeId,
                    position: 100
                });
            }

            this.emit('success', { challenge_id: this.challengeId });
        }

        /**
         * 处理验证失败
         * @private
         */
        handleError() {
            this.sliderButtonElement.style.background = DesignTokens.colors.danger;
            this.sliderButtonElement.style.color = DesignTokens.colors.white;

            const icon = this.sliderButtonElement.querySelector('i');
            if (icon) {
                icon.className = 'fas fa-times';
            }

            this.hintTextElement.textContent = '验证失败，点击重试';
            this.hintTextElement.style.color = DesignTokens.colors.danger;
            this.hintTextElement.style.opacity = '1';

            this.animateElement(this.container, 'slider-captcha-shake');

            setTimeout(() => {
                this.resetSlider();
            }, 1000);

            if (typeof this.options.onError === 'function') {
                this.options.onError({ message: '验证失败' });
            }

            this.emit('error', { message: '验证失败' });
        }

        /**
         * 重置滑块
         * @private
         */
        resetSlider() {
            this.sliderButtonElement.style.left = '4px';
            this.trackElement.style.width = '0';
            this.sliderImageElement.style.left = '0';
            this.sliderImageElement.style.transition = 'left 0.3s ease';

            const icon = this.sliderButtonElement.querySelector('i');
            if (icon) {
                icon.className = 'fas fa-arrow-right';
            }

            this.sliderButtonElement.style.background = DesignTokens.colors.white;
            this.sliderButtonElement.style.color = DesignTokens.colors.primary;

            this.hintTextElement.textContent = '拖动滑块完成验证';
            this.hintTextElement.style.color = DesignTokens.colors.gray600;

            this.container.setAttribute('aria-valuenow', 0);

            setTimeout(() => {
                this.sliderImageElement.style.transition = 'none';
            }, 300);
        }

        /**
         * 刷新验证码
         */
        refresh() {
            this.state = 'idle';
            this.loadCaptcha();

            if (typeof this.options.onRefresh === 'function') {
                this.options.onRefresh();
            }
        }

        /**
         * 元素动画
         * @private
         * @param {HTMLElement} el - 元素
         * @param {string} animationName - 动画名称
         */
        animateElement(el, animationName) {
            el.style.animation = `${animationName} 0.5s ease`;
            setTimeout(() => {
                el.style.animation = '';
            }, 500);
        }

        /**
         * @inheritdoc
         */
        destroy() {
            document.removeEventListener('mousemove', this.moveHandler);
            document.removeEventListener('touchmove', this.moveHandler);
            document.removeEventListener('mouseup', this.endHandler);
            document.removeEventListener('touchend', this.endHandler);
            super.destroy();
        }
    }

    UIComponents.SliderCaptcha = SliderCaptcha;

    /**
     * @class Button
     * @description 按钮组件
     */
    class Button {
        /**
         * @constructor
         * @param {HTMLElement|string} container - 容器元素或选择器
         * @param {Object} options - 配置选项
         */
        constructor(container, options = {}) {
            this.container = typeof container === 'string'
                ? document.querySelector(container)
                : container;

            if (!this.container) {
                throw new Error('Container element not found');
            }

            this.options = { ...this.getDefaultOptions(), ...options };
            this.state = 'idle';
            this.init();
        }

        /**
         * 获取默认选项
         * @returns {Object} 默认选项
         */
        getDefaultOptions() {
            return {
                variant: 'primary',
                size: 'md',
                disabled: false,
                loading: false,
                block: false,
                icon: null,
                iconPosition: 'left',
                text: '',
                onClick: null,
                type: 'button'
            };
        }

        /**
         * 初始化按钮
         */
        init() {
            this.render();
            this.bindEvents();
        }

        /**
         * 渲染按钮
         * @private
         */
        render() {
            this.container.innerHTML = '';
            this.container.className = `ui-button ui-button-${this.options.variant} ui-button-${this.options.size}`;
            this.container.setAttribute('role', 'button');

            if (this.options.disabled || this.options.loading) {
                this.container.setAttribute('disabled', 'true');
                this.container.setAttribute('aria-disabled', 'true');
            }

            if (this.options.block) {
                this.container.classList.add('ui-button-block');
                this.container.style.display = 'block';
                this.container.style.width = '100%';
            }

            const btn = document.createElement('button');
            btn.type = this.options.type;
            btn.className = 'btn';
            btn.disabled = this.options.disabled || this.options.loading;

            const variantClasses = {
                primary: 'btn-danger',
                secondary: 'btn-secondary',
                outline: 'btn-outline-secondary',
                ghost: 'btn-ghost',
                link: 'btn-link'
            };

            btn.className = `btn ${variantClasses[this.options.variant] || 'btn-danger'}`;

            const sizeClasses = {
                sm: 'btn-sm',
                md: '',
                lg: 'btn-lg'
            };

            btn.className += ` ${sizeClasses[this.options.size] || ''}`;

            btn.style.cssText = `
                border-radius: ${DesignTokens.borderRadius.md};
                font-weight: ${DesignTokens.fonts.weightMedium};
                transition: ${DesignTokens.transitions.bounce};
                display: inline-flex;
                align-items: center;
                justify-content: center;
                gap: 0.5rem;
            `;

            if (this.options.icon && this.options.iconPosition === 'left') {
                const iconEl = document.createElement('i');
                iconEl.className = this.options.icon;
                iconEl.setAttribute('aria-hidden', 'true');
                btn.appendChild(iconEl);
            }

            if (this.options.text) {
                const textNode = document.createTextNode(this.options.text);
                btn.appendChild(textNode);
            } else if (this.options.loading) {
                const spinner = document.createElement('span');
                spinner.className = 'ui-button-spinner';
                spinner.innerHTML = '<i class="fas fa-spinner fa-spin"></i>';
                spinner.style.marginRight = '0.5rem';
                btn.insertBefore(spinner, btn.firstChild);
            }

            if (this.options.icon && this.options.iconPosition === 'right') {
                const iconEl = document.createElement('i');
                iconEl.className = this.options.icon;
                iconEl.setAttribute('aria-hidden', 'true');
                btn.appendChild(iconEl);
            }

            this.buttonElement = btn;
            this.container.appendChild(btn);
        }

        /**
         * 绑定事件
         * @private
         */
        bindEvents() {
            this.buttonElement.addEventListener('click', (e) => {
                if (this.options.disabled || this.options.loading) {
                    e.preventDefault();
                    return;
                }

                if (typeof this.options.onClick === 'function') {
                    this.options.onClick(e);
                }
            });
        }

        /**
         * 设置加载状态
         * @param {boolean} loading - 是否加载中
         */
        setLoading(loading) {
            this.options.loading = loading;
            this.buttonElement.disabled = loading;

            if (loading) {
                this.buttonElement.setAttribute('aria-busy', 'true');
                if (!this.buttonElement.querySelector('.ui-button-spinner')) {
                    const spinner = document.createElement('span');
                    spinner.className = 'ui-button-spinner';
                    spinner.innerHTML = '<i class="fas fa-spinner fa-spin"></i>';
                    spinner.style.marginRight = '0.5rem';
                    this.buttonElement.insertBefore(spinner, this.buttonElement.firstChild);
                }
            } else {
                this.buttonElement.setAttribute('aria-busy', 'false');
                const spinner = this.buttonElement.querySelector('.ui-button-spinner');
                if (spinner) {
                    spinner.remove();
                }
            }
        }

        /**
         * 设置禁用状态
         * @param {boolean} disabled - 是否禁用
         */
        setDisabled(disabled) {
            this.options.disabled = disabled;
            this.buttonElement.disabled = disabled;
            this.container.setAttribute('aria-disabled', disabled ? 'true' : 'false');
        }

        /**
         * 设置按钮文本
         * @param {string} text - 按钮文本
         */
        setText(text) {
            this.options.text = text;
            const textNode = Array.from(this.buttonElement.childNodes).find(
                node => node.nodeType === Node.TEXT_NODE
            );
            if (textNode) {
                textNode.textContent = text;
            } else {
                this.buttonElement.appendChild(document.createTextNode(text));
            }
        }

        /**
         * 获取按钮元素
         * @returns {HTMLButtonElement} 按钮元素
         */
        getElement() {
            return this.buttonElement;
        }

        /**
         * 销毁按钮
         */
        destroy() {
            this.container.innerHTML = '';
        }
    }

    UIComponents.Button = Button;

    /**
     * @class Modal
     * @description 模态框组件
     */
    class Modal {
        /**
         * @constructor
         * @param {Object} options - 配置选项
         */
        constructor(options = {}) {
            this.options = { ...this.getDefaultOptions(), ...options };
            this.isOpen = false;
            this.currentFocus = null;
            this.previousActiveElement = null;
            this.create();
        }

        /**
         * 获取默认选项
         * @returns {Object} 默认选项
         */
        getDefaultOptions() {
            return {
                title: '',
                content: '',
                size: 'md',
                closable: true,
                closeOnEscape: true,
                closeOnBackdrop: true,
                showHeader: true,
                showFooter: true,
                footerButtons: [],
                onOpen: null,
                onClose: null,
                onConfirm: null,
                centered: false,
                scrollable: false
            };
        }

        /**
         * 创建模态框
         * @private
         */
        create() {
            this.backdrop = document.createElement('div');
            this.backdrop.className = 'ui-modal-backdrop';
            this.backdrop.style.cssText = `
                position: fixed;
                top: 0;
                left: 0;
                right: 0;
                bottom: 0;
                background: ${DesignTokens.colors.overlay};
                z-index: ${DesignTokens.zIndex.modalBackdrop};
                opacity: 0;
                transition: opacity ${DesignTokens.transitions.normal};
            `;

            this.element = document.createElement('div');
            this.element.className = 'ui-modal';
            this.element.setAttribute('role', 'dialog');
            this.element.setAttribute('aria-modal', 'true');
            this.element.setAttribute('aria-labelledby', 'ui-modal-title');
            this.element.style.cssText = `
                position: fixed;
                top: 0;
                left: 0;
                right: 0;
                bottom: 0;
                z-index: ${DesignTokens.zIndex.modal};
                display: flex;
                align-items: ${this.options.centered ? 'center' : 'flex-start'};
                justify-content: center;
                padding: 1rem;
                opacity: 0;
                visibility: hidden;
                transition: opacity ${DesignTokens.transitions.normal}, visibility ${DesignTokens.transitions.normal};
            `;

            const sizeMap = {
                sm: '400px',
                md: '500px',
                lg: '700px',
                xl: '900px',
                full: '100%'
            };

            this.dialog = document.createElement('div');
            this.dialog.className = 'ui-modal-dialog';
            this.dialog.style.cssText = `
                background: ${DesignTokens.colors.white};
                border-radius: ${DesignTokens.borderRadius.xl};
                box-shadow: ${DesignTokens.shadows.xl};
                width: 100%;
                max-width: ${sizeMap[this.options.size] || sizeMap.md};
                max-height: ${this.options.scrollable ? '90vh' : 'none'};
                overflow: ${this.options.scrollable ? 'auto' : 'hidden'};
                display: flex;
                flex-direction: column;
                transform: translateY(-20px);
                transition: transform ${DesignTokens.transitions.normal};
            `;

            if (this.options.showHeader) {
                this.header = document.createElement('div');
                this.header.className = 'ui-modal-header';
                this.header.style.cssText = `
                    padding: 1.25rem 1.5rem;
                    border-bottom: 1px solid ${DesignTokens.colors.border};
                    display: flex;
                    align-items: center;
                    justify-content: space-between;
                `;

                const title = document.createElement('h5');
                title.id = 'ui-modal-title';
                title.className = 'ui-modal-title';
                title.textContent = this.options.title;
                title.style.cssText = `
                    margin: 0;
                    font-size: ${DesignTokens.fonts.sizeLg};
                    font-weight: ${DesignTokens.fonts.weightSemiBold};
                    color: ${DesignTokens.colors.textPrimary};
                `;

                this.header.appendChild(title);

                if (this.options.closable) {
                    const closeBtn = document.createElement('button');
                    closeBtn.type = 'button';
                    closeBtn.className = 'ui-modal-close';
                    closeBtn.setAttribute('aria-label', '关闭');
                    closeBtn.style.cssText = `
                        background: transparent;
                        border: none;
                        font-size: 1.25rem;
                        cursor: pointer;
                        color: ${DesignTokens.colors.gray500};
                        padding: 0.25rem;
                        line-height: 1;
                        transition: color ${DesignTokens.transitions.fast};
                    `;
                    closeBtn.innerHTML = '&times;';
                    closeBtn.addEventListener('click', () => this.close());
                    closeBtn.addEventListener('mouseenter', () => {
                        closeBtn.style.color = DesignTokens.colors.textPrimary;
                    });
                    closeBtn.addEventListener('mouseleave', () => {
                        closeBtn.style.color = DesignTokens.colors.gray500;
                    });
                    this.header.appendChild(closeBtn);
                }

                this.dialog.appendChild(this.header);
            }

            this.body = document.createElement('div');
            this.body.className = 'ui-modal-body';
            this.body.style.cssText = `
                padding: 1.5rem;
                flex: 1;
                overflow-y: auto;
            `;
            this.body.innerHTML = this.options.content;
            this.dialog.appendChild(this.body);

            if (this.options.showFooter) {
                this.footer = document.createElement('div');
                this.footer.className = 'ui-modal-footer';
                this.footer.style.cssText = `
                    padding: 1rem 1.5rem;
                    border-top: 1px solid ${DesignTokens.colors.border};
                    display: flex;
                    align-items: center;
                    justify-content: flex-end;
                    gap: 0.75rem;
                `;

                if (this.options.footerButtons.length === 0) {
                    const cancelBtn = document.createElement('button');
                    cancelBtn.type = 'button';
                    cancelBtn.className = 'btn btn-secondary';
                    cancelBtn.textContent = '取消';
                    cancelBtn.style.cssText = `
                        padding: 0.5rem 1rem;
                        border-radius: ${DesignTokens.borderRadius.md};
                    `;
                    cancelBtn.addEventListener('click', () => this.close());
                    this.footer.appendChild(cancelBtn);

                    const confirmBtn = document.createElement('button');
                    confirmBtn.type = 'button';
                    confirmBtn.className = 'btn btn-danger';
                    confirmBtn.textContent = '确定';
                    confirmBtn.style.cssText = `
                        padding: 0.5rem 1rem;
                        border-radius: ${DesignTokens.borderRadius.md};
                    `;
                    confirmBtn.addEventListener('click', () => {
                        if (typeof this.options.onConfirm === 'function') {
                            this.options.onConfirm();
                        }
                        this.close();
                    });
                    this.footer.appendChild(confirmBtn);
                } else {
                    this.options.footerButtons.forEach(btn => {
                        const button = document.createElement('button');
                        button.type = 'button';
                        button.className = `btn btn-${btn.variant || 'secondary'}`;
                        button.textContent = btn.text || '';
                        button.style.cssText = `
                            padding: 0.5rem 1rem;
                            border-radius: ${DesignTokens.borderRadius.md};
                        `;
                        button.addEventListener('click', () => {
                            if (typeof btn.onClick === 'function') {
                                btn.onClick();
                            }
                            if (btn.closeOnClick !== false) {
                                this.close();
                            }
                        });
                        this.footer.appendChild(button);
                    });
                }

                this.dialog.appendChild(this.footer);
            }

            this.element.appendChild(this.dialog);
            document.body.appendChild(this.backdrop);
            document.body.appendChild(this.element);

            this.bindEvents();
        }

        /**
         * 绑定事件
         * @private
         */
        bindEvents() {
            if (this.options.closeOnBackdrop) {
                this.backdrop.addEventListener('click', () => this.close());
            }

            if (this.options.closeOnEscape) {
                this.escapeHandler = (e) => {
                    if (e.key === 'Escape' && this.isOpen) {
                        this.close();
                    }
                };
                document.addEventListener('keydown', this.escapeHandler);
            }
        }

        /**
         * 打开模态框
         */
        open() {
            this.previousActiveElement = document.activeElement;
            this.isOpen = true;

            document.body.style.overflow = 'hidden';

            this.backdrop.style.opacity = '1';
            this.element.style.visibility = 'visible';
            this.element.style.opacity = '1';
            this.dialog.style.transform = 'translateY(0)';

            this.currentFocus = this.dialog.querySelector('button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])');
            if (this.currentFocus) {
                this.currentFocus.focus();
            }

            if (typeof this.options.onOpen === 'function') {
                this.options.onOpen();
            }

            this.emit('open');
        }

        /**
         * 关闭模态框
         */
        close() {
            if (!this.isOpen) return;

            this.isOpen = false;
            document.body.style.overflow = '';

            this.backdrop.style.opacity = '0';
            this.element.style.opacity = '0';
            this.element.style.visibility = 'hidden';
            this.dialog.style.transform = 'translateY(-20px)';

            if (this.previousActiveElement) {
                this.previousActiveElement.focus();
            }

            setTimeout(() => {
                if (typeof this.options.onClose === 'function') {
                    this.options.onClose();
                }
                this.emit('close');
            }, 300);
        }

        /**
         * 设置内容
         * @param {string} content - HTML内容
         */
        setContent(content) {
            this.body.innerHTML = content;
        }

        /**
         * 获取内容元素
         * @returns {HTMLElement} 内容元素
         */
        getBody() {
            return this.body;
        }

        /**
         * 设置标题
         * @param {string} title - 标题
         */
        setTitle(title) {
            const titleEl = this.dialog.querySelector('#ui-modal-title');
            if (titleEl) {
                titleEl.textContent = title;
            }
        }

        /**
         * 触发事件
         * @private
         * @param {string} eventName - 事件名
         */
        emit(eventName) {
            const event = new CustomEvent(`modal:${eventName}`, {
                detail: { modal: this }
            });
            this.element.dispatchEvent(event);
        }

        /**
         * 销毁模态框
         */
        destroy() {
            if (this.escapeHandler) {
                document.removeEventListener('keydown', this.escapeHandler);
            }
            this.backdrop.remove();
            this.element.remove();
        }

        /**
         * 静态方法：显示确认对话框
         * @static
         * @param {Object} options - 配置选项
         * @returns {Modal} 模态框实例
         */
        static confirm(options) {
            const modal = new Modal({
                title: options.title || '确认',
                content: `<p>${options.message || '确定要执行此操作吗？'}</p>`,
                closable: false,
                closeOnBackdrop: false,
                closeOnEscape: false,
                footerButtons: [
                    {
                        text: options.cancelText || '取消',
                        variant: 'secondary',
                        onClick: () => {
                            if (typeof options.onCancel === 'function') {
                                options.onCancel();
                            }
                        }
                    },
                    {
                        text: options.confirmText || '确定',
                        variant: 'primary',
                        onClick: () => {
                            if (typeof options.onConfirm === 'function') {
                                options.onConfirm();
                            }
                        }
                    }
                ]
            });
            modal.open();
            return modal;
        }

        /**
         * 静态方法：显示警告对话框
         * @static
         * @param {Object} options - 配置选项
         * @returns {Modal} 模态框实例
         */
        static alert(options) {
            const modal = new Modal({
                title: options.title || '提示',
                content: `<p>${options.message || ''}</p>`,
                closable: true,
                showFooter: true,
                footerButtons: [
                    {
                        text: options.buttonText || '确定',
                        variant: 'primary',
                        onClick: () => {
                            if (typeof options.onClose === 'function') {
                                options.onClose();
                            }
                        }
                    }
                ]
            });
            modal.open();
            return modal;
        }
    }

    UIComponents.Modal = Modal;

    /**
     * @class Toast
     * @description 提示组件
     */
    class Toast {
        /**
         * @constructor
         */
        constructor() {
            this.toasts = [];
            this.container = null;
            this.createContainer();
        }

        /**
         * 创建容器
         * @private
         */
        createContainer() {
            this.container = document.getElementById('ui-toast-container');
            if (this.container) return;

            this.container = document.createElement('div');
            this.container.id = 'ui-toast-container';
            this.container.style.cssText = `
                position: fixed;
                top: 1rem;
                right: 1rem;
                z-index: ${DesignTokens.zIndex.toast};
                display: flex;
                flex-direction: column;
                gap: 0.5rem;
                max-width: 400px;
                pointer-events: none;
            `;
            document.body.appendChild(this.container);
        }

        /**
         * 显示提示
         * @param {Object|string} options - 配置选项或消息文本
         * @returns {HTMLElement} Toast元素
         */
        show(options) {
            if (typeof options === 'string') {
                options = { message: options };
            }

            const toast = document.createElement('div');
            toast.className = `ui-toast ui-toast-${options.type || 'info'}`;
            toast.setAttribute('role', 'alert');
            toast.setAttribute('aria-live', 'polite');

            const iconMap = {
                success: 'fa-check-circle',
                error: 'fa-times-circle',
                warning: 'fa-exclamation-triangle',
                info: 'fa-info-circle'
            };

            const colorMap = {
                success: DesignTokens.colors.success,
                error: DesignTokens.colors.danger,
                warning: DesignTokens.colors.warning,
                info: DesignTokens.colors.info
            };

            toast.style.cssText = `
                background: ${DesignTokens.colors.white};
                border-radius: ${DesignTokens.borderRadius.lg};
                box-shadow: ${DesignTokens.shadows.lg};
                padding: 1rem 1.25rem;
                display: flex;
                align-items: flex-start;
                gap: 0.75rem;
                pointer-events: auto;
                animation: toast-slide-in 0.3s ease;
                border-left: 4px solid ${colorMap[options.type] || colorMap.info};
            `;

            const icon = document.createElement('i');
            icon.className = `fas ${iconMap[options.type] || iconMap.info}`;
            icon.setAttribute('aria-hidden', 'true');
            icon.style.cssText = `
                font-size: 1.25rem;
                color: ${colorMap[options.type] || colorMap.info};
                flex-shrink: 0;
            `;

            const content = document.createElement('div');
            content.style.cssText = `
                flex: 1;
                min-width: 0;
            `;

            if (options.title) {
                const title = document.createElement('div');
                title.style.cssText = `
                    font-weight: ${DesignTokens.fonts.weightSemiBold};
                    color: ${DesignTokens.colors.textPrimary};
                    margin-bottom: 0.25rem;
                `;
                title.textContent = options.title;
                content.appendChild(title);
            }

            const message = document.createElement('div');
            message.style.cssText = `
                color: ${DesignTokens.colors.textSecondary};
                font-size: ${DesignTokens.fonts.sizeSm};
                line-height: 1.5;
            `;
            message.textContent = options.message || '';
            content.appendChild(message);

            if (options.dismissible !== false) {
                const closeBtn = document.createElement('button');
                closeBtn.type = 'button';
                closeBtn.className = 'ui-toast-close';
                closeBtn.setAttribute('aria-label', '关闭');
                closeBtn.style.cssText = `
                    background: transparent;
                    border: none;
                    cursor: pointer;
                    color: ${DesignTokens.colors.gray500};
                    font-size: 1rem;
                    padding: 0;
                    flex-shrink: 0;
                    transition: color ${DesignTokens.transitions.fast};
                `;
                closeBtn.innerHTML = '&times;';
                closeBtn.addEventListener('click', () => this.hide(toast));
                closeBtn.addEventListener('mouseenter', () => {
                    closeBtn.style.color = DesignTokens.colors.textPrimary;
                });
                closeBtn.addEventListener('mouseleave', () => {
                    closeBtn.style.color = DesignTokens.colors.gray500;
                });
                toast.appendChild(closeBtn);
            }

            toast.appendChild(icon);
            toast.appendChild(content);
            this.container.appendChild(toast);

            const toastData = { element: toast, timer: null };
            this.toasts.push(toastData);

            if (options.duration !== 0) {
                const duration = options.duration || 3000;
                toastData.timer = setTimeout(() => {
                    this.hide(toast);
                }, duration);
            }

            return toast;
        }

        /**
         * 隐藏提示
         * @param {HTMLElement} toast - Toast元素
         */
        hide(toast) {
            if (!toast || !toast.parentNode) return;

            const toastData = this.toasts.find(t => t.element === toast);
            if (toastData && toastData.timer) {
                clearTimeout(toastData.timer);
            }

            toast.style.animation = 'toast-slide-out 0.3s ease forwards';
            setTimeout(() => {
                toast.remove();
                this.toasts = this.toasts.filter(t => t.element !== toast);
            }, 300);
        }

        /**
         * 显示成功提示
         * @param {string} message - 消息
         * @param {Object} [options] - 其他选项
         * @returns {HTMLElement} Toast元素
         */
        success(message, options = {}) {
            return this.show({ ...options, type: 'success', message });
        }

        /**
         * 显示错误提示
         * @param {string} message - 消息
         * @param {Object} [options] - 其他选项
         * @returns {HTMLElement} Toast元素
         */
        error(message, options = {}) {
            return this.show({ ...options, type: 'error', message });
        }

        /**
         * 显示警告提示
         * @param {string} message - 消息
         * @param {Object} [options] - 其他选项
         * @returns {HTMLElement} Toast元素
         */
        warning(message, options = {}) {
            return this.show({ ...options, type: 'warning', message });
        }

        /**
         * 显示信息提示
         * @param {string} message - 消息
         * @param {Object} [options] - 其他选项
         * @returns {HTMLElement} Toast元素
         */
        info(message, options = {}) {
            return this.show({ ...options, type: 'info', message });
        }

        /**
         * 隐藏所有提示
         */
        hideAll() {
            [...this.toasts].forEach(toastData => {
                this.hide(toastData.element);
            });
        }

        /**
         * 销毁
         */
        destroy() {
            this.hideAll();
            if (this.container && this.container.parentNode) {
                this.container.remove();
            }
        }
    }

    Toast.DEFAULTS = {
        duration: 3000,
        type: 'info',
        dismissible: true
    };

    UIComponents.Toast = Toast;

    UIComponents.DesignTokens = DesignTokens;

    UIComponents.version = '1.0.0';

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = UIComponents;
    } else {
        global.UIComponents = UIComponents;
        global.DesignTokens = DesignTokens;
        global.ThemeManager = ThemeManager;
        global.CaptchaUI = CaptchaUI;
        global.SliderCaptcha = SliderCaptcha;
        global.UIButton = Button;
        global.UIModal = Modal;
        global.UIToast = Toast;
    }

})(typeof window !== 'undefined' ? window : this);
