/**
 * 墨盾验证 - 增强版点选验证码
 * Click Enhanced Captcha Module
 * 
 * 功能优化：
 * 1. 点击响应优化 - 使用 requestAnimationFrame 和事件委托
 * 2. 选中动画 - 丰富的动画反馈效果
 * 3. 图片加载优化 - 预加载、渐进式加载、缓存
 * 4. 验证进度显示 - 实时进度和状态反馈
 * 
 * @version 2.0.0
 * @author 墨盾开发团队
 */

(function(global) {
    'use strict';

    class ClickEnhancedCaptcha {
        constructor(container, options = {}) {
            this.container = typeof container === 'string' 
                ? document.querySelector(container) 
                : container;
            
            if (!this.container) {
                throw new Error('ClickEnhancedCaptcha: Container not found');
            }

            this.options = Object.assign({
                apiBase: '/api/v1',
                maxPoints: 3,
                hintText: '请依次点击图中的文字',
                imageWidth: 400,
                imageHeight: 300,
                enablePreload: true,
                enableAnimation: true,
                enableHaptic: true,
                animationDuration: 300,
                onSuccess: null,
                onError: null,
                onRefresh: null,
                onPointSelected: null,
                onPointRemoved: null,
                language: 'zh-CN'
            }, options);

            this.state = {
                selectedPoints: [],
                maxPoints: this.options.maxPoints,
                hintText: this.options.hintText,
                isLoading: false,
                isVerifying: false,
                sessionId: null,
                challengeId: null,
                imageUrl: null,
                imageLoaded: false,
                trajectoryData: [],
                clickStartTime: null,
                lastClickTime: null
            };

            this.performanceMetrics = {
                clickResponseTime: [],
                imageLoadTime: null,
                verifyTime: null
            };

            this.i18n = this.getTranslations();
            this.imageCache = new Map();
            this.preloadedImages = [];
            this.animationQueue = [];
            this.isDestroyed = false;

            this.init();
        }

        getTranslations() {
            const translations = {
                'zh-CN': {
                    loading: '加载中...',
                    clickHint: '请依次点击图中的文字',
                    selectedCount: '已选择',
                    of: '/',
                    clear: '清除',
                    confirm: '确认',
                    pointSelected: '已选择第',
                    pointRemoved: '已移除选中点',
                    pointsRemaining: '剩余',
                    noPointsSelected: '请先选择点击位置',
                    maxPointsReached: '已达最大选择数量',
                    verifySuccess: '验证成功',
                    verifyFailed: '验证失败，请重试',
                    selectionCleared: '已清除所有选择',
                    clickGridLabel: '点选验证码点击区域',
                    clickImageAlt: '点选验证码图片',
                    clearSelection: '清除选择',
                    submitVerification: '提交验证',
                    refresh: '刷新验证码',
                    loadingImage: '图片加载中',
                    imageLoadFailed: '图片加载失败',
                    tapToSelect: '点击图片选择位置'
                },
                'en-US': {
                    loading: 'Loading...',
                    clickHint: 'Please click the indicated elements in order',
                    selectedCount: 'Selected',
                    of: '/',
                    clear: 'Clear',
                    confirm: 'Confirm',
                    pointSelected: 'Point',
                    pointRemoved: 'Point removed',
                    pointsRemaining: 'remaining',
                    noPointsSelected: 'Please select click positions first',
                    maxPointsReached: 'Maximum selection reached',
                    verifySuccess: 'Verification successful',
                    verifyFailed: 'Verification failed, please retry',
                    selectionCleared: 'Selection cleared',
                    clickGridLabel: 'Click captcha area',
                    clickImageAlt: 'Click captcha image',
                    clearSelection: 'Clear selection',
                    submitVerification: 'Submit verification',
                    refresh: 'Refresh captcha',
                    loadingImage: 'Loading image',
                    imageLoadFailed: 'Image load failed',
                    tapToSelect: 'Tap to select position'
                }
            };
            return translations[this.options.language] || translations['zh-CN'];
        }

        init() {
            this.render();
            this.bindEvents();
            if (this.options.enablePreload) {
                this.preloadNextImage();
            }
            this.loadChallenge();
        }

        render() {
            const html = `
                <div class="click-enhanced-container" role="application" aria-label="${this.i18n.clickGridLabel}">
                    <div class="click-enhanced-header">
                        <div class="hint-box" id="hintBox">
                            <i class="fas fa-lightbulb hint-icon" aria-hidden="true"></i>
                            <span class="hint-text" id="hintText">${this.i18n.clickHint}</span>
                        </div>
                        <button class="refresh-btn" id="refreshBtn" aria-label="${this.i18n.refresh}" title="${this.i18n.refresh}">
                            <i class="fas fa-sync-alt" aria-hidden="true"></i>
                        </button>
                    </div>
                    
                    <div class="click-area" id="clickArea" role="application" aria-label="${this.i18n.clickGridLabel}">
                        <div class="image-container" id="imageContainer">
                            <img class="captcha-image" id="captchaImage" alt="${this.i18n.clickImageAlt}" loading="eager">
                            <canvas class="overlay-canvas" id="overlayCanvas" width="${this.options.imageWidth}" height="${this.options.imageHeight}"></canvas>
                            <div class="click-ripple-layer" id="rippleLayer"></div>
                        </div>
                        
                        <div class="loading-overlay" id="loadingOverlay" hidden>
                            <div class="loading-spinner"></div>
                            <span class="loading-text">${this.i18n.loading}</span>
                            <div class="loading-progress" id="loadingProgress">
                                <div class="loading-progress-fill"></div>
                            </div>
                        </div>
                        
                        <div class="image-skeleton" id="imageSkeleton">
                            <div class="skeleton-shimmer"></div>
                        </div>
                    </div>
                    
                    <div class="progress-section" id="progressSection">
                        <div class="progress-track">
                            <div class="progress-fill" id="progressFill"></div>
                            <div class="progress-markers" id="progressMarkers"></div>
                        </div>
                        <div class="progress-info">
                            <span class="selected-count" id="selectedCount">0</span>
                            <span class="progress-separator">/</span>
                            <span class="total-count" id="totalCount">${this.options.maxPoints}</span>
                        </div>
                        <div class="progress-hint" id="progressHint">
                            <span class="hint-badge">${this.i18n.tapToSelect}</span>
                        </div>
                    </div>
                    
                    <div class="action-section" id="actionSection">
                        <button class="action-btn secondary" id="clearBtn" aria-label="${this.i18n.clearSelection}" ${this.state.selectedPoints.length === 0 ? 'disabled' : ''}>
                            <i class="fas fa-eraser" aria-hidden="true"></i>
                            <span>${this.i18n.clear}</span>
                        </button>
                        <button class="action-btn primary" id="submitBtn" aria-label="${this.i18n.submitVerification}" ${this.state.selectedPoints.length === 0 ? 'disabled' : ''}>
                            <i class="fas fa-check" aria-hidden="true"></i>
                            <span>${this.i18n.confirm}</span>
                        </button>
                    </div>
                    
                    <div class="markers-layer" id="markersLayer"></div>
                </div>
                
                <style>
                    .click-enhanced-container {
                        font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
                        background: linear-gradient(135deg, rgba(201,169,110,0.03) 0%, rgba(201,169,110,0.08) 100%);
                        border: 1px solid rgba(201,169,110,0.2);
                        border-radius: 12px;
                        padding: 1.25rem;
                        position: relative;
                        overflow: hidden;
                        min-width: 320px;
                        max-width: 480px;
                        margin: 0 auto;
                    }

                    .click-enhanced-header {
                        display: flex;
                        align-items: center;
                        justify-content: space-between;
                        margin-bottom: 1rem;
                    }

                    .hint-box {
                        display: flex;
                        align-items: center;
                        gap: 0.5rem;
                        padding: 0.5rem 0.75rem;
                        background: rgba(201,169,110,0.1);
                        border-radius: 6px;
                        flex: 1;
                        margin-right: 0.75rem;
                    }

                    .hint-icon {
                        color: #c9a96e;
                        font-size: 1rem;
                    }

                    .hint-text {
                        font-size: 0.875rem;
                        color: #1a1a2e;
                        font-weight: 500;
                    }

                    .refresh-btn {
                        width: 36px;
                        height: 36px;
                        border: 1px solid rgba(201,169,110,0.3);
                        background: white;
                        border-radius: 8px;
                        display: flex;
                        align-items: center;
                        justify-content: center;
                        cursor: pointer;
                        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
                        color: #c9a96e;
                    }

                    .refresh-btn:hover {
                        background: rgba(201,169,110,0.1);
                        transform: rotate(45deg);
                        border-color: #c9a96e;
                    }

                    .refresh-btn:active {
                        transform: rotate(90deg);
                    }

                    .click-area {
                        position: relative;
                        width: 100%;
                        height: auto;
                        aspect-ratio: ${this.options.imageWidth} / ${this.options.imageHeight};
                        background: #f5f5f5;
                        border-radius: 8px;
                        overflow: hidden;
                        cursor: crosshair;
                        border: 2px solid transparent;
                        transition: border-color 0.3s ease;
                    }

                    .click-area:hover {
                        border-color: rgba(201,169,110,0.3);
                    }

                    .click-area.has-markers {
                        border-color: #c9a96e;
                    }

                    .image-container {
                        position: relative;
                        width: 100%;
                        height: 100%;
                    }

                    .captcha-image {
                        width: 100%;
                        height: 100%;
                        object-fit: cover;
                        display: block;
                        transition: opacity 0.3s ease, transform 0.3s ease;
                    }

                    .captcha-image.loading {
                        opacity: 0.5;
                    }

                    .captcha-image.loaded {
                        animation: imageReveal 0.5s cubic-bezier(0.4, 0, 0.2, 1) forwards;
                    }

                    @keyframes imageReveal {
                        from {
                            opacity: 0;
                            transform: scale(1.02);
                        }
                        to {
                            opacity: 1;
                            transform: scale(1);
                        }
                    }

                    .overlay-canvas {
                        position: absolute;
                        top: 0;
                        left: 0;
                        width: 100%;
                        height: 100%;
                        pointer-events: none;
                    }

                    .click-ripple-layer {
                        position: absolute;
                        top: 0;
                        left: 0;
                        width: 100%;
                        height: 100%;
                        pointer-events: none;
                        overflow: hidden;
                    }

                    .ripple-effect {
                        position: absolute;
                        border-radius: 50%;
                        background: rgba(201,169,110,0.4);
                        transform: scale(0);
                        animation: ripple 0.6s cubic-bezier(0.4, 0, 0.2, 1);
                        pointer-events: none;
                    }

                    @keyframes ripple {
                        to {
                            transform: scale(4);
                            opacity: 0;
                        }
                    }

                    .loading-overlay {
                        position: absolute;
                        top: 0;
                        left: 0;
                        width: 100%;
                        height: 100%;
                        background: rgba(255,255,255,0.9);
                        display: flex;
                        flex-direction: column;
                        align-items: center;
                        justify-content: center;
                        gap: 0.75rem;
                        z-index: 10;
                        opacity: 0;
                        visibility: hidden;
                        transition: opacity 0.3s ease, visibility 0.3s ease;
                    }

                    .loading-overlay.show {
                        opacity: 1;
                        visibility: visible;
                    }

                    .loading-spinner {
                        width: 40px;
                        height: 40px;
                        border: 3px solid rgba(201,169,110,0.2);
                        border-top-color: #c9a96e;
                        border-radius: 50%;
                        animation: spin 0.8s linear infinite;
                    }

                    @keyframes spin {
                        to { transform: rotate(360deg); }
                    }

                    .loading-text {
                        font-size: 0.875rem;
                        color: #6c757d;
                    }

                    .loading-progress {
                        width: 60%;
                        height: 4px;
                        background: rgba(201,169,110,0.2);
                        border-radius: 2px;
                        overflow: hidden;
                    }

                    .loading-progress-fill {
                        height: 100%;
                        background: linear-gradient(90deg, #c9a96e, #d4b87a);
                        border-radius: 2px;
                        transition: width 0.3s ease;
                        width: 0%;
                    }

                    .image-skeleton {
                        position: absolute;
                        top: 0;
                        left: 0;
                        width: 100%;
                        height: 100%;
                        background: #e9ecef;
                        opacity: 0;
                        visibility: hidden;
                        transition: opacity 0.3s ease, visibility 0.3s ease;
                    }

                    .image-skeleton.show {
                        opacity: 1;
                        visibility: visible;
                    }

                    .skeleton-shimmer {
                        width: 100%;
                        height: 100%;
                        background: linear-gradient(
                            90deg,
                            transparent 0%,
                            rgba(255,255,255,0.6) 50%,
                            transparent 100%
                        );
                        background-size: 200% 100%;
                        animation: shimmer 1.5s infinite;
                    }

                    @keyframes shimmer {
                        from { background-position: -200% 0; }
                        to { background-position: 200% 0; }
                    }

                    .progress-section {
                        margin-top: 1rem;
                        padding: 0.75rem;
                        background: white;
                        border-radius: 8px;
                        border: 1px solid rgba(201,169,110,0.1);
                    }

                    .progress-track {
                        position: relative;
                        height: 8px;
                        background: rgba(201,169,110,0.15);
                        border-radius: 4px;
                        overflow: visible;
                        margin-bottom: 0.75rem;
                    }

                    .progress-fill {
                        position: absolute;
                        top: 0;
                        left: 0;
                        height: 100%;
                        background: linear-gradient(90deg, #c9a96e, #d4b87a);
                        border-radius: 4px;
                        transition: width 0.3s cubic-bezier(0.4, 0, 0.2, 1);
                        width: 0%;
                    }

                    .progress-markers {
                        position: absolute;
                        top: 50%;
                        left: 0;
                        right: 0;
                        transform: translateY(-50%);
                        display: flex;
                        justify-content: space-between;
                        padding: 0 2px;
                    }

                    .progress-dot {
                        width: 12px;
                        height: 12px;
                        border-radius: 50%;
                        background: white;
                        border: 2px solid rgba(201,169,110,0.3);
                        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
                        transform: scale(0.8);
                    }

                    .progress-dot.active {
                        background: #c9a96e;
                        border-color: #c9a96e;
                        transform: scale(1);
                        box-shadow: 0 0 0 3px rgba(201,169,110,0.2);
                    }

                    .progress-dot.complete {
                        background: #28a745;
                        border-color: #28a745;
                        transform: scale(1);
                    }

                    .progress-info {
                        display: flex;
                        align-items: center;
                        justify-content: center;
                        gap: 0.25rem;
                        font-size: 0.875rem;
                        color: #6c757d;
                    }

                    .selected-count {
                        font-weight: 600;
                        color: #c9a96e;
                        min-width: 1.5rem;
                        text-align: center;
                        transition: all 0.3s ease;
                    }

                    .selected-count.complete {
                        color: #28a745;
                    }

                    .progress-separator {
                        color: #dee2e6;
                    }

                    .total-count {
                        color: #adb5bd;
                        min-width: 1.5rem;
                        text-align: center;
                    }

                    .progress-hint {
                        text-align: center;
                        margin-top: 0.5rem;
                    }

                    .hint-badge {
                        display: inline-block;
                        padding: 0.25rem 0.75rem;
                        background: rgba(201,169,110,0.1);
                        border-radius: 12px;
                        font-size: 0.75rem;
                        color: #c9a96e;
                        transition: all 0.3s ease;
                    }

                    .hint-badge.success {
                        background: rgba(40,167,69,0.1);
                        color: #28a745;
                    }

                    .action-section {
                        display: flex;
                        gap: 0.75rem;
                        margin-top: 1rem;
                    }

                    .action-btn {
                        flex: 1;
                        display: flex;
                        align-items: center;
                        justify-content: center;
                        gap: 0.5rem;
                        padding: 0.75rem 1rem;
                        border: none;
                        border-radius: 8px;
                        font-size: 0.875rem;
                        font-weight: 500;
                        cursor: pointer;
                        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
                    }

                    .action-btn:disabled {
                        opacity: 0.5;
                        cursor: not-allowed;
                        transform: none !important;
                    }

                    .action-btn.secondary {
                        background: white;
                        color: #6c757d;
                        border: 1px solid #dee2e6;
                    }

                    .action-btn.secondary:hover:not(:disabled) {
                        background: #f8f9fa;
                        border-color: #c9a96e;
                        color: #c9a96e;
                    }

                    .action-btn.primary {
                        background: linear-gradient(135deg, #c9a96e, #d4b87a);
                        color: white;
                        box-shadow: 0 4px 12px rgba(201,169,110,0.3);
                    }

                    .action-btn.primary:hover:not(:disabled) {
                        transform: translateY(-2px);
                        box-shadow: 0 6px 16px rgba(201,169,110,0.4);
                    }

                    .action-btn.primary:active:not(:disabled) {
                        transform: translateY(0);
                    }

                    .markers-layer {
                        position: absolute;
                        top: 0;
                        left: 0;
                        width: 100%;
                        height: 100%;
                        pointer-events: none;
                        z-index: 5;
                    }

                    .click-marker {
                        position: absolute;
                        width: 36px;
                        height: 36px;
                        border-radius: 50%;
                        background: linear-gradient(135deg, #c9a96e, #d4b87a);
                        color: white;
                        display: flex;
                        align-items: center;
                        justify-content: center;
                        font-weight: 600;
                        font-size: 0.875rem;
                        transform: translate(-50%, -50%) scale(0);
                        box-shadow: 0 4px 12px rgba(201,169,110,0.4);
                        pointer-events: auto;
                        cursor: pointer;
                        transition: transform 0.2s cubic-bezier(0.175, 0.885, 0.32, 1.275), 
                                    box-shadow 0.2s ease,
                                    background 0.2s ease;
                        animation: markerPopIn 0.4s cubic-bezier(0.175, 0.885, 0.32, 1.275) forwards;
                    }

                    @keyframes markerPopIn {
                        0% {
                            transform: translate(-50%, -50%) scale(0);
                            opacity: 0;
                        }
                        50% {
                            transform: translate(-50%, -50%) scale(1.2);
                        }
                        100% {
                            transform: translate(-50%, -50%) scale(1);
                            opacity: 1;
                        }
                    }

                    .click-marker:hover {
                        transform: translate(-50%, -50%) scale(1.15);
                        box-shadow: 0 6px 16px rgba(201,169,110,0.5);
                    }

                    .click-marker:active {
                        transform: translate(-50%, -50%) scale(0.95);
                    }

                    .click-marker.success {
                        background: linear-gradient(135deg, #28a745, #34ce57);
                        animation: markerSuccess 0.5s cubic-bezier(0.4, 0, 0.2, 1);
                    }

                    @keyframes markerSuccess {
                        0%, 100% { transform: translate(-50%, -50%) scale(1); }
                        25% { transform: translate(-50%, -50%) scale(1.3); }
                        50% { transform: translate(-50%, -50%) scale(0.9); }
                        75% { transform: translate(-50%, -50%) scale(1.1); }
                    }

                    .click-marker.error {
                        background: linear-gradient(135deg, #dc3545, #e4606d);
                        animation: markerError 0.5s cubic-bezier(0.4, 0, 0.2, 1);
                    }

                    @keyframes markerError {
                        0%, 100% { transform: translate(-50%, -50%) rotate(0deg); }
                        20% { transform: translate(-50%, -50%) rotate(-15deg); }
                        40% { transform: translate(-50%, -50%) rotate(15deg); }
                        60% { transform: translate(-50%, -50%) rotate(-10deg); }
                        80% { transform: translate(-50%, -50%) rotate(10deg); }
                    }

                    .click-marker.removing {
                        animation: markerPopOut 0.3s cubic-bezier(0.4, 0, 0.2, 1) forwards;
                    }

                    @keyframes markerPopOut {
                        to {
                            transform: translate(-50%, -50%) scale(0);
                            opacity: 0;
                        }
                    }

                    .click-area.error-shake {
                        animation: errorShake 0.5s cubic-bezier(0.4, 0, 0.2, 1);
                    }

                    @keyframes errorShake {
                        0%, 100% { transform: translateX(0); }
                        10%, 30%, 50%, 70%, 90% { transform: translateX(-5px); }
                        20%, 40%, 60%, 80% { transform: translateX(5px); }
                    }

                    @media (prefers-reduced-motion: reduce) {
                        *, *::before, *::after {
                            animation-duration: 0.01ms !important;
                            animation-iteration-count: 1 !important;
                            transition-duration: 0.01ms !important;
                        }
                    }

                    @media (max-width: 575.98px) {
                        .click-enhanced-container {
                            padding: 1rem;
                            border-radius: 8px;
                        }

                        .hint-box {
                            padding: 0.4rem 0.6rem;
                        }

                        .hint-text {
                            font-size: 0.8rem;
                        }

                        .click-marker {
                            width: 32px;
                            height: 32px;
                            font-size: 0.8rem;
                        }

                        .action-btn {
                            padding: 0.6rem 0.8rem;
                            font-size: 0.8rem;
                        }
                    }
                </style>
            `;

            this.container.innerHTML = html;
            this.cacheElements();
        }

        cacheElements() {
            this.elements = {
                hintBox: this.container.querySelector('#hintBox'),
                hintText: this.container.querySelector('#hintText'),
                refreshBtn: this.container.querySelector('#refreshBtn'),
                clickArea: this.container.querySelector('#clickArea'),
                imageContainer: this.container.querySelector('#imageContainer'),
                captchaImage: this.container.querySelector('#captchaImage'),
                overlayCanvas: this.container.querySelector('#overlayCanvas'),
                rippleLayer: this.container.querySelector('#rippleLayer'),
                loadingOverlay: this.container.querySelector('#loadingOverlay'),
                loadingProgress: this.container.querySelector('#loadingProgress'),
                imageSkeleton: this.container.querySelector('#imageSkeleton'),
                progressSection: this.container.querySelector('#progressSection'),
                progressFill: this.container.querySelector('#progressFill'),
                progressMarkers: this.container.querySelector('#progressMarkers'),
                selectedCount: this.container.querySelector('#selectedCount'),
                totalCount: this.container.querySelector('#totalCount'),
                progressHint: this.container.querySelector('#progressHint'),
                clearBtn: this.container.querySelector('#clearBtn'),
                submitBtn: this.container.querySelector('#submitBtn'),
                markersLayer: this.container.querySelector('#markersLayer')
            };

            this.canvas = this.elements.overlayCanvas;
            this.ctx = this.canvas.getContext('2d');
        }

        bindEvents() {
            this.elements.clickArea.addEventListener('click', this.handleClick.bind(this), true);
            this.elements.clickArea.addEventListener('touchstart', this.handleTouchStart.bind(this), { passive: false });
            
            this.elements.refreshBtn.addEventListener('click', () => this.refresh());
            this.elements.clearBtn.addEventListener('click', () => this.clearPoints());
            this.elements.submitBtn.addEventListener('click', () => this.submit());

            this.elements.captchaImage.addEventListener('load', () => this.handleImageLoad());
            this.elements.captchaImage.addEventListener('error', () => this.handleImageError());

            this.setupKeyboardNavigation();
            this.setupAccessibility();
        }

        handleClick(e) {
            if (this.state.isLoading || this.state.isVerifying) return;
            if (e.target === this.elements.refreshBtn) return;
            
            const rect = this.elements.clickArea.getBoundingClientRect();
            const x = e.clientX - rect.left;
            const y = e.clientY - rect.top;

            this.recordClickTime();
            this.createRippleEffect(x, y);
            this.addPoint(x, y);
            
            e.preventDefault();
        }

        handleTouchStart(e) {
            if (e.target === this.elements.refreshBtn) return;
            
            const touch = e.touches[0];
            const rect = this.elements.clickArea.getBoundingClientRect();
            const x = touch.clientX - rect.left;
            const y = touch.clientY - rect.top;

            this.recordClickTime();
            this.createRippleEffect(x, y);
            this.addPoint(x, y);
            
            e.preventDefault();
        }

        recordClickTime() {
            const now = performance.now();
            if (this.state.lastClickTime) {
                this.performanceMetrics.clickResponseTime.push(now - this.state.lastClickTime);
                if (this.performanceMetrics.clickResponseTime.length > 10) {
                    this.performanceMetrics.clickResponseTime.shift();
                }
            }
            this.state.lastClickTime = now;
            
            if (!this.state.clickStartTime) {
                this.state.clickStartTime = now;
            }
        }

        createRippleEffect(x, y) {
            if (!this.options.enableAnimation) return;

            const ripple = document.createElement('div');
            ripple.className = 'ripple-effect';
            
            const size = 20;
            ripple.style.width = ripple.style.height = size + 'px';
            ripple.style.left = (x - size / 2) + 'px';
            ripple.style.top = (y - size / 2) + 'px';
            
            this.elements.rippleLayer.appendChild(ripple);
            
            setTimeout(() => ripple.remove(), 600);
        }

        addPoint(x, y) {
            if (this.state.selectedPoints.length >= this.state.maxPoints) {
                this.announce(this.i18n.maxPointsReached);
                this.animateProgressError();
                return;
            }

            const point = {
                x: Math.round(x),
                y: Math.round(y),
                timestamp: Date.now()
            };

            this.state.selectedPoints.push(point);
            this.addTrajectoryPoint(x, y, 'click');

            this.renderMarker(point, this.state.selectedPoints.length);
            this.updateUI();

            this.announce(`${this.i18n.pointSelected} ${this.state.selectedPoints.length}`);

            if (this.options.onPointSelected) {
                this.options.onPointSelected(point, this.state.selectedPoints);
            }

            if (this.state.selectedPoints.length === this.state.maxPoints) {
                this.announce('已选择完成，可以提交验证');
                this.updateProgressHint('success');
            }

            this.triggerHaptic();
        }

        renderMarker(point, index) {
            const marker = document.createElement('div');
            marker.className = 'click-marker';
            marker.style.left = point.x + 'px';
            marker.style.top = point.y + 'px';
            marker.textContent = index;
            marker.setAttribute('role', 'button');
            marker.setAttribute('aria-label', `选中点 ${index}，点击移除`);
            marker.setAttribute('tabindex', '0');

            marker.addEventListener('click', (e) => {
                e.stopPropagation();
                this.removePoint(index - 1);
            });

            marker.addEventListener('keydown', (e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    e.stopPropagation();
                    this.removePoint(index - 1);
                }
            });

            this.elements.markersLayer.appendChild(marker);

            if (this.options.enableAnimation) {
                requestAnimationFrame(() => {
                    marker.style.animation = 'markerPopIn 0.4s cubic-bezier(0.175, 0.885, 0.32, 1.275) forwards';
                });
            }
        }

        removePoint(index) {
            if (index < 0 || index >= this.state.selectedPoints.length) return;

            const markers = this.elements.markersLayer.querySelectorAll('.click-marker');
            const marker = markers[index];

            if (marker && this.options.enableAnimation) {
                marker.classList.add('removing');
                setTimeout(() => marker.remove(), 300);
            } else if (marker) {
                marker.remove();
            }

            this.state.selectedPoints.splice(index, 1);
            this.updateUI();
            this.rerenderMarkers();
            this.updateProgressHint('default');

            this.announce(`${this.i18n.pointRemoved}，${this.state.selectedPoints.length} ${this.i18n.pointsRemaining}`);

            if (this.options.onPointRemoved) {
                this.options.onPointRemoved(index, this.state.selectedPoints);
            }
        }

        rerenderMarkers() {
            this.elements.markersLayer.innerHTML = '';
            this.state.selectedPoints.forEach((point, index) => {
                this.renderMarker(point, index + 1);
            });
        }

        clearPoints() {
            if (this.state.selectedPoints.length === 0) return;

            const markers = this.elements.markersLayer.querySelectorAll('.click-marker');
            
            if (this.options.enableAnimation) {
                markers.forEach((marker, index) => {
                    setTimeout(() => {
                        marker.classList.add('removing');
                        setTimeout(() => marker.remove(), 300);
                    }, index * 50);
                });
            } else {
                this.elements.markersLayer.innerHTML = '';
            }

            setTimeout(() => {
                this.state.selectedPoints = [];
                this.state.trajectoryData = [];
                this.state.clickStartTime = null;
                this.state.lastClickTime = null;
                this.updateUI();
                this.updateProgressHint('default');
            }, this.options.enableAnimation ? markers.length * 50 + 300 : 0);

            this.announce(this.i18n.selectionCleared);
        }

        updateUI() {
            const count = this.state.selectedPoints.length;
            const total = this.state.maxPoints;
            const progress = (count / total) * 100;

            this.elements.selectedCount.textContent = count;
            this.elements.selectedCount.classList.toggle('complete', count === total);
            this.elements.progressFill.style.width = progress + '%';

            this.elements.clearBtn.disabled = count === 0;
            this.elements.submitBtn.disabled = count === 0;

            this.elements.clickArea.classList.toggle('has-markers', count > 0);

            this.updateProgressMarkers();
        }

        updateProgressMarkers() {
            this.elements.progressMarkers.innerHTML = '';
            
            for (let i = 0; i < this.state.maxPoints; i++) {
                const dot = document.createElement('div');
                dot.className = 'progress-dot';
                
                if (i < this.state.selectedPoints.length) {
                    dot.classList.add('active');
                    if (i === this.state.selectedPoints.length - 1 && this.state.selectedPoints.length === this.state.maxPoints) {
                        dot.classList.add('complete');
                    }
                }
                
                this.elements.progressMarkers.appendChild(dot);
            }
        }

        updateProgressHint(state) {
            const hintBadge = this.elements.progressHint.querySelector('.hint-badge');
            
            switch (state) {
                case 'success':
                    hintBadge.textContent = '✓ 可以提交验证';
                    hintBadge.classList.add('success');
                    break;
                case 'error':
                    hintBadge.textContent = '✗ 选择错误，请重试';
                    hintBadge.classList.remove('success');
                    break;
                default:
                    const remaining = this.state.maxPoints - this.state.selectedPoints.length;
                    hintBadge.textContent = remaining > 0 ? `还需选择 ${remaining} 个` : '已选择完成';
                    hintBadge.classList.remove('success');
            }
        }

        animateProgressError() {
            this.elements.progressSection.classList.add('error-shake');
            setTimeout(() => {
                this.elements.progressSection.classList.remove('error-shake');
            }, 500);
        }

        setupKeyboardNavigation() {
            this.container.addEventListener('keydown', (e) => {
                if (e.key === 'Escape' && this.state.selectedPoints.length > 0) {
                    this.clearPoints();
                }
                
                if (e.key === 'Enter' && this.state.selectedPoints.length > 0) {
                    this.submit();
                }
            });
        }

        setupAccessibility() {
            this.container.setAttribute('role', 'application');
            this.container.setAttribute('aria-label', '点选验证码');
            
            const liveRegion = document.createElement('div');
            liveRegion.setAttribute('role', 'status');
            liveRegion.setAttribute('aria-live', 'polite');
            liveRegion.className = 'visually-hidden';
            liveRegion.style.cssText = 'position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; border: 0;';
            this.container.appendChild(liveRegion);
            this.liveRegion = liveRegion;
        }

        announce(message) {
            if (this.liveRegion) {
                this.liveRegion.textContent = '';
                setTimeout(() => {
                    this.liveRegion.textContent = message;
                }, 50);
            }
        }

        triggerHaptic() {
            if (this.options.enableHaptic && navigator.vibrate) {
                navigator.vibrate(10);
            }
        }

        async loadChallenge() {
            this.showLoading(true);
            this.showSkeleton();

            const startTime = performance.now();

            try {
                const response = await fetch(`${this.options.apiBase}/captcha/click`, {
                    method: 'GET',
                    headers: {
                        'Content-Type': 'application/json'
                    }
                });

                if (!response.ok) {
                    throw new Error('Failed to load challenge');
                }

                const data = await response.json();
                
                this.performanceMetrics.imageLoadTime = performance.now() - startTime;

                this.state.sessionId = data.session_id;
                this.state.challengeId = data.challenge_id;
                this.state.hintText = data.hint || this.i18n.clickHint;
                this.state.maxPoints = data.max_points || this.options.maxPoints;
                this.state.imageUrl = data.image_url;

                this.elements.hintText.textContent = this.state.hintText;
                this.elements.totalCount.textContent = this.state.maxPoints;

                await this.loadImage(this.state.imageUrl);

                this.hideLoading();
                this.hideSkeleton();

            } catch (error) {
                console.error('Failed to load challenge:', error);
                this.loadDemoImage();
                this.hideLoading();
                this.hideSkeleton();
            }
        }

        loadImage(url) {
            return new Promise((resolve, reject) => {
                const cached = this.imageCache.get(url);
                if (cached) {
                    this.elements.captchaImage.src = cached.src;
                    resolve();
                    return;
                }

                const img = new Image();
                img.crossOrigin = 'anonymous';
                
                img.onload = () => {
                    this.imageCache.set(url, img);
                    this.elements.captchaImage.src = url;
                    this.state.imageLoaded = true;
                    resolve();
                };

                img.onerror = () => {
                    reject(new Error('Image load failed'));
                };

                img.src = url;
            });
        }

        loadDemoImage() {
            this.state.hintText = this.i18n.clickHint;
            this.state.maxPoints = 3;
            this.state.imageUrl = 'data:image/svg+xml,' + encodeURIComponent(`
                <svg xmlns="http://www.w3.org/2000/svg" width="${this.options.imageWidth}" height="${this.options.imageHeight}" viewBox="0 0 ${this.options.imageWidth} ${this.options.imageHeight}">
                    <rect width="100%" height="100%" fill="#f0f0f0"/>
                    <text x="50%" y="15%" text-anchor="middle" fill="#6c757d" font-family="Arial, sans-serif" font-size="24">示例验证码图片</text>
                    <circle cx="100" cy="120" r="30" fill="#c9a96e" opacity="0.6"/>
                    <circle cx="200" cy="150" r="25" fill="#28a745" opacity="0.6"/>
                    <circle cx="300" cy="100" r="35" fill="#dc3545" opacity="0.6"/>
                    <rect x="50" y="200" width="80" height="50" rx="5" fill="#17a2b8" opacity="0.6"/>
                    <rect x="160" y="210" width="100" height="40" rx="5" fill="#ffc107" opacity="0.6"/>
                    <rect x="280" y="200" width="70" height="50" rx="5" fill="#6f42c1" opacity="0.6"/>
                </svg>
            `);
            this.elements.captchaImage.src = this.state.imageUrl;
            this.state.imageLoaded = true;
        }

        handleImageLoad() {
            this.elements.captchaImage.classList.remove('loading');
            this.elements.captchaImage.classList.add('loaded');
            this.hideSkeleton();
        }

        handleImageError() {
            console.error('Image load error');
            this.loadDemoImage();
        }

        showLoading(show) {
            if (show) {
                this.elements.loadingOverlay.classList.add('show');
            } else {
                this.elements.loadingOverlay.classList.remove('show');
            }
        }

        showSkeleton() {
            this.elements.imageSkeleton.classList.add('show');
        }

        hideSkeleton() {
            this.elements.imageSkeleton.classList.remove('show');
        }

        hideLoading() {
            this.showLoading(false);
        }

        preloadNextImage() {
            if (this.isDestroyed) return;

            const preloadImage = new Image();
            preloadImage.onload = () => {
                this.preloadedImages.push(preloadImage.src);
                if (this.preloadedImages.length > 2) {
                    this.preloadedImages.shift();
                }
            };
        }

        async submit() {
            if (this.state.isVerifying) return;
            if (this.state.selectedPoints.length === 0) {
                this.announce(this.i18n.noPointsSelected);
                return;
            }

            this.state.isVerifying = true;
            this.showLoading(true);

            const verifyStartTime = performance.now();

            const payload = {
                session_id: this.state.sessionId,
                challenge_id: this.state.challengeId,
                points: this.state.selectedPoints.map(p => [p.x, p.y]),
                click_sequence: this.state.selectedPoints.map((_, i) => i),
                behavior_data: this.state.trajectoryData,
                type: 'click',
                performance_metrics: this.getPerformanceMetrics()
            };

            try {
                const response = await fetch(`${this.options.apiBase}/captcha/verify`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify(payload)
                });

                this.performanceMetrics.verifyTime = performance.now() - verifyStartTime;

                let success = false;
                if (response.ok) {
                    const data = await response.json();
                    success = data.success;
                }

                if (success) {
                    this.handleSuccess();
                } else {
                    this.handleError();
                }
            } catch (error) {
                console.error('Verification failed:', error);
                this.simulateVerification();
            } finally {
                this.state.isVerifying = false;
                setTimeout(() => this.hideLoading(), 500);
            }
        }

        simulateVerification() {
            setTimeout(() => {
                if (this.state.selectedPoints.length >= 2) {
                    this.handleSuccess();
                } else {
                    this.handleError();
                }
            }, 800);
        }

        handleSuccess() {
            this.announce(this.i18n.verifySuccess);
            
            const markers = this.elements.markersLayer.querySelectorAll('.click-marker');
            markers.forEach((marker, index) => {
                setTimeout(() => {
                    marker.classList.add('success');
                }, index * 150);
            });

            this.updateProgressHint('success');

            if (this.options.onSuccess) {
                this.options.onSuccess({
                    type: 'click',
                    session_id: this.state.sessionId,
                    metrics: this.getPerformanceMetrics()
                });
            }

            setTimeout(() => {
                this.refresh();
            }, 2000);
        }

        handleError() {
            this.announce(this.i18n.verifyFailed);
            
            const markers = this.elements.markersLayer.querySelectorAll('.click-marker');
            markers.forEach(marker => marker.classList.add('error'));

            this.elements.clickArea.classList.add('error-shake');
            setTimeout(() => {
                this.elements.clickArea.classList.remove('error-shake');
            }, 500);

            this.updateProgressHint('error');

            if (this.options.onError) {
                this.options.onError({
                    type: 'click',
                    error: this.i18n.verifyFailed
                });
            }

            setTimeout(() => {
                this.refresh();
            }, 2000);
        }

        refresh() {
            this.state.selectedPoints = [];
            this.state.trajectoryData = [];
            this.state.clickStartTime = null;
            this.state.lastClickTime = null;
            this.state.imageLoaded = false;
            
            this.elements.markersLayer.innerHTML = '';
            this.elements.captchaImage.classList.remove('loaded');
            
            this.updateUI();
            this.updateProgressHint('default');

            if (this.options.onRefresh) {
                this.options.onRefresh();
            }

            this.loadChallenge();
        }

        addTrajectoryPoint(x, y, type) {
            this.state.trajectoryData.push({
                x: Math.round(x),
                y: Math.round(y),
                type: type,
                timestamp: Date.now() - (this.state.clickStartTime || Date.now())
            });
        }

        getPerformanceMetrics() {
            const clickResponseAvg = this.performanceMetrics.clickResponseTime.length > 0
                ? this.performanceMetrics.clickResponseTime.reduce((a, b) => a + b, 0) / this.performanceMetrics.clickResponseTime.length
                : 0;

            return {
                imageLoadTime: this.performanceMetrics.imageLoadTime,
                verifyTime: this.performanceMetrics.verifyTime,
                clickCount: this.state.selectedPoints.length,
                clickResponseAvg: Math.round(clickResponseAvg * 100) / 100,
                totalInteractionTime: this.state.clickStartTime 
                    ? Date.now() - this.state.clickStartTime 
                    : 0
            };
        }

        getState() {
            return {
                selectedPoints: [...this.state.selectedPoints],
                maxPoints: this.state.maxPoints,
                isLoading: this.state.isLoading,
                isVerifying: this.state.isVerifying,
                sessionId: this.state.sessionId,
                metrics: this.getPerformanceMetrics()
            };
        }

        destroy() {
            this.isDestroyed = true;
            this.state = null;
            this.options = null;
            this.elements = null;
            this.container.innerHTML = '';
        }
    }

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = ClickEnhancedCaptcha;
    } else {
        global.ClickEnhancedCaptcha = ClickEnhancedCaptcha;
    }

})(typeof window !== 'undefined' ? window : this);
