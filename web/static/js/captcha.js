/**
 * @fileoverview 验证码前端交互模块
 * 
 * 功能特性：
 * - Canvas图片渲染
 * - 滑块拖拽交互
 * - 验证状态UI反馈
 * - 中英文国际化支持
 * - 响应式布局（PC/移动端）
 * - 无障碍访问支持
 * 
 * @module CaptchaValidator
 * @version 1.0.0
 * @author HJTPX Team
 * @license MIT
 */

(function() {
    'use strict';

    /**
     * 国际化翻译配置
     * @type {Object.<string, Object.<string, string>>}
     */
    const i18n = {
        'zh-CN': {
            title: '安全验证',
            tip: '按住滑块，拖动完成拼图',
            verifying: '验证中...',
            success: '验证成功',
            failed: '验证失败',
            expired: '验证码已过期',
            refresh: '刷新',
            loadError: '加载失败，请重试'
        },
        'en-US': {
            title: 'Security Verification',
            tip: 'Hold the slider and drag to complete',
            verifying: 'Verifying...',
            success: 'Verification Success',
            failed: 'Verification Failed',
            expired: 'Captcha Expired',
            refresh: 'Refresh',
            loadError: 'Load failed, please retry'
        }
    };

    /**
     * 验证码验证器类
     * 
     * 负责管理验证码的创建、渲染、拖拽交互和验证流程
     * 
     * @class CaptchaValidator
     * @example
     * const validator = new CaptchaValidator();
     * validator.onSuccess((result) => {
     *     console.log('验证成功', result);
     * });
     */
    class CaptchaValidator {
        /**
         * 创建验证码验证器实例
         * 
         * 初始化所有必要的状态和配置，包括：
         * - 会话ID
         * - 目标位置
         * - 拖拽状态
         * - 语言设置
         * 
         * @constructor
         */
        constructor() {
            this.sessionId = null;
            this.targetX = 0;
            this.targetY = 0;
            this.puzzleY = 0;
            this.tolerance = 10;
            this.isDragging = false;
            this.startX = 0;
            this.currentX = 0;
            this.maxDragX = 0;
            this.traceData = [];
            this.bgCanvas = null;
            this.bgCtx = null;
            this.currentLang = this.getLanguage();
            this.isVerifying = false;
            this.isVerified = false;
            
            this.init();
        }

        /**
         * 获取当前语言设置
         * 
         * 从localStorage读取保存的语言偏好，默认为简体中文
         * 
         * @returns {string} 语言代码，如 'zh-CN' 或 'en-US'
         * @method getLanguage
         */
        getLanguage() {
            const savedLang = localStorage.getItem('captcha-lang') || 'zh-CN';
            return savedLang;
        }

        /**
         * 获取翻译文本
         * 
         * 根据当前语言设置获取对应的翻译文本
         * 
         * @param {string} key - 翻译键名
         * @returns {string} 翻译后的文本，如果未找到则返回键名
         * @method t
         */
        t(key) {
            const translations = i18n[this.currentLang] || i18n['zh-CN'];
            return translations[key] || key;
        }

        /**
         * 初始化验证码验证器
         * 
         * 执行完整的初始化流程：
         * 1. 初始化DOM元素引用
         * 2. 初始化Canvas画布
         * 3. 绑定事件监听器
         * 4. 初始化语言切换器
         * 5. 创建验证码
         * 
         * @async
         * @returns {Promise<void>}
         * @method init
         */
        async init() {
            this.initElements();
            this.initCanvas();
            this.bindEvents();
            this.initLanguageSwitcher();
            await this.createCaptcha();
        }

        /**
         * 初始化DOM元素引用
         * 
         * 获取并缓存所有必要的DOM元素引用，避免重复查询
         * 
         * @method initElements
         */
        initElements() {
            this.bgCanvas = document.getElementById('bgCanvas');
            this.bgCtx = this.bgCanvas ? this.bgCanvas.getContext('2d') : null;
            this.sliderPiece = document.getElementById('sliderPiece');
            this.sliderThumb = document.getElementById('sliderThumb');
            this.sliderTrack = document.getElementById('sliderTrack');
            this.sliderProgress = document.getElementById('sliderProgress');
            this.sliderTip = document.getElementById('sliderTip');
            this.statusText = document.getElementById('statusText');
            this.refreshBtn = document.getElementById('refreshBtn');
            this.loadingOverlay = document.getElementById('loadingOverlay');
            this.imageWrapper = document.getElementById('imageWrapper');
        }

        /**
         * 初始化Canvas画布
         * 
         * 设置Canvas的宽度和高度，与容器尺寸保持一致
         * 
         * @method initCanvas
         */
        initCanvas() {
            if (!this.bgCanvas || !this.bgCtx) return;
            
            const wrapper = this.imageWrapper;
            if (wrapper) {
                this.bgCanvas.width = wrapper.offsetWidth;
                this.bgCanvas.height = wrapper.offsetHeight;
            }
        }

        /**
         * 初始化语言切换器
         * 
         * 为语言切换按钮绑定点击事件，支持：
         * - 中英文切换
         * - localStorage持久化
         * - 自动应用保存的语言设置
         * 
         * @method initLanguageSwitcher
         */
        initLanguageSwitcher() {
            const langBtns = document.querySelectorAll('.captcha-lang-btn');
            langBtns.forEach(btn => {
                btn.addEventListener('click', (e) => {
                    const lang = e.target.getAttribute('data-lang');
                    this.switchLanguage(lang);
                    
                    langBtns.forEach(b => {
                        b.classList.remove('active');
                        b.setAttribute('aria-pressed', 'false');
                    });
                    e.target.classList.add('active');
                    e.target.setAttribute('aria-pressed', 'true');
                });
            });

            const savedLang = localStorage.getItem('captcha-lang') || 'zh-CN';
            const langBtn = document.querySelector(`.captcha-lang-btn[data-lang="${savedLang}"]`);
            if (langBtn) {
                langBtn.click();
            }
        }

        /**
         * 切换界面语言
         * 
         * 更新当前语言设置并刷新界面上的所有翻译文本
         * 
         * @param {string} lang - 目标语言代码
         * @returns {void}
         * @method switchLanguage
         */
        switchLanguage(lang) {
            this.currentLang = lang;
            localStorage.setItem('captcha-lang', lang);
            document.documentElement.lang = lang;

            document.querySelectorAll('[data-i18n]').forEach(el => {
                const key = el.getAttribute('data-i18n');
                if (this.t(key) !== key) {
                    el.textContent = this.t(key);
                }
            });
        }

        /**
         * 创建新验证码
         * 
         * 调用后端API创建新的验证码会话，包括：
         * - 重置滑块状态
         * - 获取验证码图片
         * - 渲染验证码界面
         * 
         * @async
         * @returns {Promise<void>}
         * @method createCaptcha
         */
        async createCaptcha() {
            this.showLoading(true);
            this.resetSlider();
            this.hideStatus();

            try {
                const response = await fetch('/api/v1/captcha/slider', {
                    method: 'GET',
                    headers: {
                        'Content-Type': 'application/json'
                    }
                });

                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }

                const result = await response.json();
                
                if (result.session_id) {
                    this.sessionId = result.session_id;
                    this.targetX = result.target_x || 0;
                    this.targetY = result.target_y || 0;
                    this.puzzleY = result.puzzle_y || result.target_y || 0;
                    this.tolerance = result.tolerance || 10;
                    
                    await this.renderImages(result);
                }
            } catch (error) {
                console.error('创建验证码失败:', error);
                this.showTip(this.t('loadError'));
            } finally {
                this.showLoading(false);
            }
        }

        /**
         * 渲染验证码图片
         * 
         * 在Canvas上绘制验证码背景图，并在指定位置放置滑块图片
         * 
         * @async
         * @param {Object} data - 验证码数据，包含image_url和puzzle_image
         * @returns {Promise<void>}
         * @method renderImages
         */
        async renderImages(data) {
            if (!this.bgCtx) return;

            const bgCanvas = this.bgCanvas;
            const ctx = this.bgCtx;
            
            bgCanvas.width = 340;
            bgCanvas.height = 180;

            if (data.image_url && data.puzzle_image) {
                const bgImg = new Image();
                bgImg.crossOrigin = 'anonymous';
                
                await new Promise((resolve, reject) => {
                    bgImg.onload = () => {
                        ctx.drawImage(bgImg, 0, 0, 340, 180);
                        resolve();
                    };
                    bgImg.onerror = () => {
                        this.drawFallbackBackground();
                        resolve();
                    };
                    bgImg.src = data.image_url;
                });

                if (this.sliderPiece) {
                    this.sliderPiece.style.backgroundImage = `url(${data.puzzle_image})`;
                    this.sliderPiece.style.top = `${this.puzzleY}px`;
                    this.sliderPiece.style.left = '0px';
                }
            } else {
                this.drawFallbackBackground();
            }

            this.maxDragX = 340 - 52;
        }

        /**
         * 绘制备用背景
         * 
         * 当验证码图片加载失败时，绘制一个渐变背景作为备选方案
         * 
         * @returns {void}
         * @method drawFallbackBackground
         */
        drawFallbackBackground() {
            if (!this.bgCtx) return;
            
            const ctx = this.bgCtx;
            const width = 340;
            const height = 180;
            
            const gradient = ctx.createLinearGradient(0, 0, width, height);
            gradient.addColorStop(0, '#667eea');
            gradient.addColorStop(0.5, '#764ba2');
            gradient.addColorStop(1, '#f093fb');
            ctx.fillStyle = gradient;
            ctx.fillRect(0, 0, width, height);

            ctx.fillStyle = 'rgba(255, 255, 255, 0.1)';
            for (let i = 0; i < 20; i++) {
                const x = Math.random() * width;
                const y = Math.random() * height;
                const r = 20 + Math.random() * 40;
                ctx.beginPath();
                ctx.arc(x, y, r, 0, Math.PI * 2);
                ctx.fill();
            }
        }

        /**
         * 绑定事件监听器
         * 
         * 为滑块和轨道绑定鼠标/触摸事件，包括：
         * - 开始拖拽 (mousedown/touchstart)
         * - 拖拽中 (mousemove/touchmove)
         * - 结束拖拽 (mouseup/touchend)
         * - 键盘导航 (keydown)
         * - 窗口调整 (resize)
         * 
         * @returns {void}
         * @method bindEvents
         */
        bindEvents() {
            if (this.sliderThumb) {
                this.sliderThumb.addEventListener('mousedown', (e) => this.startDrag(e));
                this.sliderThumb.addEventListener('touchstart', (e) => this.startDrag(e), { passive: false });
            }

            if (this.sliderTrack) {
                this.sliderTrack.addEventListener('mousedown', (e) => this.startDrag(e));
                this.sliderTrack.addEventListener('touchstart', (e) => this.startDrag(e), { passive: false });
            }

            document.addEventListener('mousemove', (e) => this.onDrag(e));
            document.addEventListener('touchmove', (e) => this.onDrag(e), { passive: false });

            document.addEventListener('mouseup', (e) => this.endDrag(e));
            document.addEventListener('touchend', (e) => this.endDrag(e));

            if (this.refreshBtn) {
                this.refreshBtn.addEventListener('click', () => this.refresh());
            }

            if (this.sliderTrack) {
                this.sliderTrack.addEventListener('keydown', (e) => this.handleKeyboard(e));
            }

            window.addEventListener('resize', () => this.handleResize());
        }

        /**
         * 开始拖拽操作
         * 
         * 记录拖拽起始位置，初始化轨迹数据
         * 
         * @param {MouseEvent|TouchEvent} e - 鼠标或触摸事件
         * @returns {void}
         * @method startDrag
         */
        startDrag(e) {
            if (this.isVerifying || this.isVerified) return;
            
            e.preventDefault();
            this.isDragging = true;
            
            const clientX = e.type.includes('mouse') ? e.clientX : e.touches[0].clientX;
            this.startX = clientX - this.currentX;
            
            this.traceData = [{
                event: 'start',
                x: clientX,
                timestamp: Date.now()
            }];

            if (this.sliderThumb) {
                this.sliderThumb.style.cursor = 'grabbing';
            }

            this.updateTip(this.t('tip'));
        }

        /**
         * 处理拖拽移动
         * 
         * 实时更新滑块位置，记录移动轨迹
         * 
         * @param {MouseEvent|TouchEvent} e - 鼠标或触摸事件
         * @returns {void}
         * @method onDrag
         */
        onDrag(e) {
            if (!this.isDragging) return;
            
            e.preventDefault();
            
            const clientX = e.type.includes('mouse') ? e.clientX : e.touches[0].clientX;
            const diff = clientX - this.startX;
            
            const newX = Math.max(0, Math.min(diff, this.maxDragX));
            this.currentX = newX;

            if (this.sliderThumb) {
                this.sliderThumb.style.left = `${newX}px`;
            }

            if (this.sliderPiece) {
                this.sliderPiece.style.left = `${newX}px`;
            }

            if (this.sliderProgress) {
                const progress = (newX / this.maxDragX) * 100;
                this.sliderProgress.style.width = `${progress}%`;
            }

            if (this.sliderTrack) {
                const progress = Math.round((newX / this.maxDragX) * 100);
                this.sliderTrack.setAttribute('aria-valuenow', progress.toString());
            }

            this.traceData.push({
                event: 'move',
                x: clientX,
                timestamp: Date.now()
            });
        }

        /**
         * 结束拖拽操作
         * 
         * 记录结束事件，触发验证码验证
         * 
         * @param {MouseEvent|TouchEvent} e - 鼠标或触摸事件
         * @returns {void}
         * @method endDrag
         */
        endDrag(e) {
            if (!this.isDragging) return;
            
            this.isDragging = false;

            const clientX = e.type.includes('mouse') ? e.clientX : 
                           (e.changedTouches ? e.changedTouches[0].clientX : 0);

            this.traceData.push({
                event: 'end',
                x: clientX,
                timestamp: Date.now()
            });

            if (this.sliderThumb) {
                this.sliderThumb.style.cursor = 'grab';
            }

            if (this.currentX > 5) {
                this.verify();
            }
        }

        /**
         * 处理键盘导航
         * 
         * 支持使用方向键控制滑块位置，空格或回车键提交验证
         * 
         * @param {KeyboardEvent} e - 键盘事件
         * @returns {void}
         * @method handleKeyboard
         */
        handleKeyboard(e) {
            if (this.isVerifying || this.isVerified) return;
            
            const step = 10;
            let newX = this.currentX;

            switch(e.key) {
                case 'ArrowRight':
                case 'ArrowUp':
                    e.preventDefault();
                    newX = Math.min(this.currentX + step, this.maxDragX);
                    break;
                case 'ArrowLeft':
                case 'ArrowDown':
                    e.preventDefault();
                    newX = Math.max(this.currentX - step, 0);
                    break;
                case 'Enter':
                case ' ':
                    e.preventDefault();
                    if (this.currentX > 5) {
                        this.verify();
                    }
                    return;
                default:
                    return;
            }

            this.currentX = newX;

            if (this.sliderThumb) {
                this.sliderThumb.style.left = `${newX}px`;
            }

            if (this.sliderPiece) {
                this.sliderPiece.style.left = `${newX}px`;
            }

            if (this.sliderProgress) {
                const progress = (newX / this.maxDragX) * 100;
                this.sliderProgress.style.width = `${progress}%`;
            }

            if (this.sliderTrack) {
                const progress = Math.round((newX / this.maxDragX) * 100);
                this.sliderTrack.setAttribute('aria-valuenow', progress.toString());
            }
        }

        /**
         * 提交验证码验证
         * 
         * 将滑块位置和轨迹数据发送到后端进行验证
         * 
         * @async
         * @returns {Promise<void>}
         * @method verify
         */
        async verify() {
            if (this.isVerifying || !this.sessionId) return;
            
            this.isVerifying = true;
            this.showVerifying();

            try {
                const response = await fetch('/api/v1/captcha/verify', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        session_id: this.sessionId,
                        type: 'slider',
                        x: Math.round(this.currentX + 26),
                        y: Math.round(this.puzzleY + 26),
                        behavior_data: this.traceData.map(t => ({
                            x: t.x,
                            y: 0,
                            timestamp: t.timestamp,
                            event: t.event
                        }))
                    })
                });

                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }

                const result = await response.json();

                if (result.success) {
                    this.showSuccess();
                } else {
                    this.showFailed();
                }
            } catch (error) {
                console.error('验证失败:', error);
                this.showFailed();
            } finally {
                this.isVerifying = false;
            }
        }

        /**
         * 显示加载状态
         * 
         * @param {boolean} show - 是否显示加载状态
         * @returns {void}
         * @method showLoading
         */
        showLoading(show) {
            if (!this.loadingOverlay) return;
            
            if (show) {
                this.loadingOverlay.classList.add('show');
            } else {
                this.loadingOverlay.classList.remove('show');
            }
        }

        /**
         * 显示验证中状态
         * 
         * 更新UI显示验证正在进行中
         * 
         * @returns {void}
         * @method showVerifying
         */
        showVerifying() {
            if (!this.statusText || !this.sliderThumb) return;

            this.updateTip(this.t('verifying'));
            
            this.statusText.innerHTML = `<i class="fas fa-spinner fa-spin"></i> ${this.t('verifying')}`;
            this.statusText.className = 'captcha-status verifying show';
            
            if (this.refreshBtn) {
                this.refreshBtn.classList.add('loading');
            }
        }

        /**
         * 显示验证成功状态
         * 
         * 更新UI显示验证成功，触发成功动画
         * 
         * @returns {void}
         * @method showSuccess
         */
        showSuccess() {
            if (!this.statusText || !this.sliderThumb || !this.imageWrapper) return;

            this.statusText.innerHTML = `<i class="fas fa-check-circle"></i> ${this.t('success')}`;
            this.statusText.className = 'captcha-status success show';
            
            this.sliderThumb.classList.add('success');
            this.sliderThumb.innerHTML = '<i class="fas fa-check"></i>';
            
            this.imageWrapper.classList.add('captcha-success-bounce');
            setTimeout(() => {
                this.imageWrapper.classList.remove('captcha-success-bounce');
            }, 600);

            this.isVerified = true;
            
            if (this.refreshBtn) {
                this.refreshBtn.classList.remove('loading');
            }

            if (typeof this.onSuccessCallback === 'function') {
                this.onSuccessCallback({
                    session_id: this.sessionId,
                    risk_score: 0
                });
            }
        }

        /**
         * 显示验证失败状态
         * 
         * 更新UI显示验证失败，触发失败动画，
         * 1.5秒后自动刷新验证码
         * 
         * @returns {void}
         * @method showFailed
         */
        showFailed() {
            if (!this.statusText || !this.sliderThumb || !this.imageWrapper) return;

            this.statusText.innerHTML = `<i class="fas fa-times-circle"></i> ${this.t('failed')}`;
            this.statusText.className = 'captcha-status failed show';
            
            this.sliderThumb.classList.add('failed');
            this.sliderThumb.innerHTML = '<i class="fas fa-times"></i>';
            
            this.imageWrapper.classList.add('captcha-error-shake');
            setTimeout(() => {
                this.imageWrapper.classList.remove('captcha-error-shake');
            }, 500);

            if (this.refreshBtn) {
                this.refreshBtn.classList.remove('loading');
            }

            setTimeout(() => {
                this.refresh();
            }, 1500);

            if (typeof this.onErrorCallback === 'function') {
                this.onErrorCallback({
                    message: this.t('failed')
                });
            }
        }

        /**
         * 隐藏状态文本
         * 
         * @returns {void}
         * @method hideStatus
         */
        hideStatus() {
            if (!this.statusText) return;
            
            this.statusText.className = 'captcha-status';
            this.statusText.style.display = 'none';
        }

        /**
         * 更新提示文本
         * 
         * @param {string} text - 新的提示文本
         * @returns {void}
         * @method updateTip
         */
        updateTip(text) {
            if (!this.sliderTip) return;
            
            const tipSpan = this.sliderTip.querySelector('span');
            if (tipSpan) {
                tipSpan.textContent = text;
            } else {
                this.sliderTip.innerHTML = `<i class="fas fa-hand-pointer" aria-hidden="true"></i> <span>${text}</span>`;
            }
        }

        /**
         * 显示提示信息
         * 
         * @param {string} text - 提示文本
         * @returns {void}
         * @method showTip
         */
        showTip(text) {
            if (!this.sliderTip) return;
            
            this.sliderTip.innerHTML = `<i class="fas fa-info-circle" aria-hidden="true"></i> <span>${text}</span>`;
        }

        /**
         * 重置滑块状态
         * 
         * 将滑块恢复到初始位置，清除所有状态数据
         * 
         * @returns {void}
         * @method resetSlider
         */
        resetSlider() {
            this.currentX = 0;
            this.isDragging = false;
            this.isVerifying = false;
            this.isVerified = false;
            this.traceData = [];

            if (this.sliderThumb) {
                this.sliderThumb.style.left = '2px';
                this.sliderThumb.className = 'slider-thumb';
                this.sliderThumb.innerHTML = '<i class="fas fa-chevron-right"></i>';
            }

            if (this.sliderPiece) {
                this.sliderPiece.style.left = '0px';
            }

            if (this.sliderProgress) {
                this.sliderProgress.style.width = '0';
            }

            if (this.sliderTrack) {
                this.sliderTrack.setAttribute('aria-valuenow', '0');
            }

            this.updateTip(this.t('tip'));
        }

        /**
         * 刷新验证码
         * 
         * 重新创建新的验证码，重置所有状态
         * 
         * @async
         * @returns {Promise<void>}
         * @method refresh
         */
        async refresh() {
            if (this.refreshBtn) {
                this.refreshBtn.classList.add('loading');
                const icon = this.refreshBtn.querySelector('i');
                if (icon) {
                    icon.classList.add('fa-spin');
                }
            }

            await this.createCaptcha();

            if (this.refreshBtn) {
                this.refreshBtn.classList.remove('loading');
                const icon = this.refreshBtn.querySelector('i');
                if (icon) {
                    icon.classList.remove('fa-spin');
                }
            }

            if (typeof this.onRefreshCallback === 'function') {
                this.onRefreshCallback();
            }
        }

        /**
         * 处理窗口大小变化
         * 
         * 重新初始化Canvas尺寸
         * 
         * @returns {void}
         * @method handleResize
         */
        handleResize() {
            this.initCanvas();
        }

        /**
         * 销毁验证码验证器
         * 
         * 清除所有状态，停止所有事件监听
         * 
         * @returns {void}
         * @method destroy
         */
        destroy() {
            this.isDragging = false;
            this.isVerifying = false;
            this.isVerified = false;
        }

        /**
         * 注册验证成功回调
         * 
         * @param {Function} callback - 回调函数，接收验证结果对象
         * @returns {void}
         * @method onSuccess
         */
        onSuccess(callback) {
            if (typeof callback === 'function') {
                this.onSuccessCallback = callback;
            }
        }

        /**
         * 注册验证失败回调
         * 
         * @param {Function} callback - 回调函数，接收错误对象
         * @returns {void}
         * @method onError
         */
        onError(callback) {
            if (typeof callback === 'function') {
                this.onErrorCallback = callback;
            }
        }

        /**
         * 注册刷新回调
         * 
         * @param {Function} callback - 回调函数
         * @returns {void}
         * @method onRefresh
         */
        onRefresh(callback) {
            if (typeof callback === 'function') {
                this.onRefreshCallback = callback;
            }
        }
    }

    /**
     * 将CaptchaValidator导出到全局作用域
     */
    if (typeof window !== 'undefined') {
        window.CaptchaValidator = CaptchaValidator;
    }

    /**
     * DOM准备就绪后初始化验证码
     */
    document.addEventListener('DOMContentLoaded', function() {
        const container = document.querySelector('.captcha-container');
        if (container) {
            const validator = new CaptchaValidator();
            
            validator.onSuccess(function(result) {
                console.log('验证码验证成功:', result);
            });
            
            validator.onError(function(error) {
                console.log('验证码验证失败:', error);
            });
            
            window.captchaValidator = validator;
        }
    });

    /**
     * 支持CommonJS模块导出
     */
    if (typeof module !== 'undefined' && module.exports) {
        module.exports = CaptchaValidator;
    }
})();
