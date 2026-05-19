/**
 * 滑块验证码增强版 - slider_enhanced.js
 * 
 * 优化内容：
 * 1. 优化拖动手感：使用requestAnimationFrame和CSS transform
 * 2. 滑块动画效果：添加流畅的动画过渡
 * 3. 图片加载优化：预加载、占位符、渐进式显示
 * 4. 错误反馈动画：多种错误动画效果
 * 
 * 依赖：captcha.js (Captcha类)
 */

class SliderCaptchaEnhanced {
    constructor(container, options = {}) {
        this.container = typeof container === 'string' 
            ? document.getElementById(container) 
            : container;
        
        if (!this.container) {
            console.error('SliderCaptchaEnhanced: 容器未找到');
            return;
        }

        this.options = {
            apiBase: options.apiBase || '/api/v1',
            theme: options.theme || 'light',
            enableParticles: options.enableParticles !== false,
            enableHaptic: options.enableHaptic !== false,
            enableSound: options.enableSound || false,
            animationDuration: options.animationDuration || 300,
            dragSmoothing: options.dragSmoothing || 0.85,
            onSuccess: options.onSuccess || null,
            onError: options.onError || null,
            onReady: options.onReady || null,
            ...options
        };

        this.state = {
            isDragging: false,
            isAnimating: false,
            isLoading: false,
            isLoaded: false,
            currentX: 0,
            targetX: 0,
            startX: 0,
            startY: 0,
            maxX: 0,
            velocity: 0,
            lastX: 0,
            lastTime: 0,
            imageLoaded: false,
            puzzleLoaded: false
        };

        this.animationFrame = null;
        this.touchIdentifier = null;
        this.lastMoveTime = 0;
        this.moveThrottle = 16;

        this.elements = {};
        this.eventHandlers = {};
        this.particles = [];

        this.init();
    }

    init() {
        this.createStructure();
        this.bindEvents();
        this.preloadImages();
        
        if (this.options.onReady) {
            this.options.onReady(this);
        }
    }

    createStructure() {
        const wrapper = document.createElement('div');
        wrapper.className = 'slider-enhanced-wrapper';
        wrapper.innerHTML = `
            <div class="slider-enhanced-container">
                <div class="slider-enhanced-image-container">
                    <div class="slider-enhanced-image-placeholder">
                        <div class="slider-enhanced-spinner"></div>
                        <span class="slider-enhanced-loading-text">加载中...</span>
                    </div>
                    <img class="slider-enhanced-image" alt="验证码图片" />
                    <div class="slider-enhanced-puzzle">
                        <div class="slider-enhanced-puzzle-shine"></div>
                    </div>
                    <div class="slider-enhanced-guide-arrow">
                        <i class="fas fa-arrow-right"></i>
                    </div>
                </div>
                <div class="slider-enhanced-slider-container">
                    <div class="slider-enhanced-slider-track">
                        <div class="slider-enhanced-slider-progress"></div>
                    </div>
                    <div class="slider-enhanced-slider-button" tabindex="0" role="slider" 
                         aria-label="滑块验证" aria-valuemin="0" aria-valuemax="100" aria-valuenow="0">
                        <i class="fas fa-arrows-alt-h"></i>
                    </div>
                    <div class="slider-enhanced-slider-text">拖动滑块完成验证</div>
                </div>
                <div class="slider-enhanced-feedback"></div>
            </div>
            <div class="slider-enhanced-particles-container"></div>
        `;

        this.container.appendChild(wrapper);
        this.elements.wrapper = wrapper;
        this.elements.container = wrapper.querySelector('.slider-enhanced-container');
        this.elements.imageContainer = wrapper.querySelector('.slider-enhanced-image-container');
        this.elements.imagePlaceholder = wrapper.querySelector('.slider-enhanced-image-placeholder');
        this.elements.image = wrapper.querySelector('.slider-enhanced-image');
        this.elements.puzzle = wrapper.querySelector('.slider-enhanced-puzzle');
        this.elements.puzzleShine = wrapper.querySelector('.slider-enhanced-puzzle-shine');
        this.elements.guideArrow = wrapper.querySelector('.slider-enhanced-guide-arrow');
        this.elements.sliderContainer = wrapper.querySelector('.slider-enhanced-slider-container');
        this.elements.sliderTrack = wrapper.querySelector('.slider-enhanced-slider-track');
        this.elements.sliderProgress = wrapper.querySelector('.slider-enhanced-slider-progress');
        this.elements.sliderButton = wrapper.querySelector('.slider-enhanced-slider-button');
        this.elements.sliderText = wrapper.querySelector('.slider-enhanced-slider-text');
        this.elements.feedback = wrapper.querySelector('.slider-enhanced-feedback');
        this.elements.particlesContainer = wrapper.querySelector('.slider-enhanced-particles-container');

        this.addStyles();
    }

    addStyles() {
        const styleId = 'slider-enhanced-styles';
        if (document.getElementById(styleId)) return;

        const style = document.createElement('style');
        style.id = styleId;
        style.textContent = `
            .slider-enhanced-wrapper {
                width: 100%;
                max-width: 320px;
                margin: 0 auto;
                position: relative;
                font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            }

            .slider-enhanced-container {
                background: white;
                border-radius: 12px;
                padding: 16px;
                box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
                transition: box-shadow 0.3s ease;
            }

            .slider-enhanced-container:hover {
                box-shadow: 0 6px 24px rgba(0, 0, 0, 0.15);
            }

            .slider-enhanced-image-container {
                position: relative;
                width: 100%;
                height: 160px;
                background: #f0f0f0;
                border-radius: 8px;
                overflow: hidden;
                margin-bottom: 12px;
            }

            .slider-enhanced-image-placeholder {
                position: absolute;
                top: 0;
                left: 0;
                right: 0;
                bottom: 0;
                display: flex;
                flex-direction: column;
                align-items: center;
                justify-content: center;
                background: linear-gradient(135deg, #f8f9fa 0%, #e9ecef 100%);
                z-index: 2;
                transition: opacity 0.3s ease;
            }

            .slider-enhanced-image-placeholder.hidden {
                opacity: 0;
                pointer-events: none;
            }

            .slider-enhanced-spinner {
                width: 32px;
                height: 32px;
                border: 3px solid rgba(201, 169, 110, 0.2);
                border-top-color: #c9a96e;
                border-radius: 50%;
                animation: sliderEnhancedSpin 0.8s linear infinite;
            }

            @keyframes sliderEnhancedSpin {
                to { transform: rotate(360deg); }
            }

            .slider-enhanced-loading-text {
                margin-top: 8px;
                font-size: 12px;
                color: #6c757d;
            }

            .slider-enhanced-image {
                width: 100%;
                height: 100%;
                object-fit: cover;
                opacity: 0;
                transition: opacity 0.5s ease;
            }

            .slider-enhanced-image.loaded {
                opacity: 1;
            }

            .slider-enhanced-puzzle {
                position: absolute;
                top: 0;
                width: 40px;
                height: 40px;
                background: rgba(201, 169, 110, 0.3);
                border: 2px solid #c9a96e;
                border-radius: 4px;
                display: flex;
                align-items: center;
                justify-content: center;
                z-index: 3;
                transition: transform 0.05s linear, box-shadow 0.3s ease;
                will-change: transform;
            }

            .slider-enhanced-puzzle::before {
                content: '';
                position: absolute;
                top: 50%;
                left: 50%;
                transform: translate(-50%, -50%);
                width: 12px;
                height: 12px;
                background: #c9a96e;
                border-radius: 50%;
            }

            .slider-enhanced-puzzle-shine {
                position: absolute;
                top: 0;
                left: -100%;
                width: 100%;
                height: 100%;
                background: linear-gradient(
                    90deg,
                    transparent,
                    rgba(255, 255, 255, 0.4),
                    transparent
                );
                animation: puzzleShine 2s ease-in-out infinite;
            }

            @keyframes puzzleShine {
                0%, 100% { left: -100%; }
                50% { left: 100%; }
            }

            .slider-enhanced-guide-arrow {
                position: absolute;
                top: 50%;
                right: 8px;
                transform: translateY(-50%);
                color: #c9a96e;
                font-size: 20px;
                animation: guideArrowPulse 1.5s ease-in-out infinite;
                z-index: 4;
            }

            @keyframes guideArrowPulse {
                0%, 100% { opacity: 0.5; transform: translateY(-50%) translateX(0); }
                50% { opacity: 1; transform: translateY(-50%) translateX(5px); }
            }

            .slider-enhanced-slider-container {
                position: relative;
                width: 100%;
                height: 44px;
                background: #f0f0f0;
                border-radius: 22px;
                overflow: hidden;
                cursor: pointer;
            }

            .slider-enhanced-slider-track {
                position: absolute;
                top: 2px;
                left: 2px;
                height: calc(100% - 4px);
                width: 0;
                background: linear-gradient(90deg, #c9a96e 0%, #d4b87a 100%);
                border-radius: 20px;
                transition: width 0.05s linear;
            }

            .slider-enhanced-slider-progress {
                position: absolute;
                top: 0;
                left: 0;
                height: 100%;
                width: 100%;
                background: repeating-linear-gradient(
                    90deg,
                    transparent,
                    transparent 8px,
                    rgba(255, 255, 255, 0.2) 8px,
                    rgba(255, 255, 255, 0.2) 16px
                );
            }

            .slider-enhanced-slider-button {
                position: absolute;
                top: 2px;
                left: 2px;
                width: 40px;
                height: 40px;
                background: white;
                border-radius: 50%;
                display: flex;
                align-items: center;
                justify-content: center;
                box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
                cursor: grab;
                transition: transform 0.2s ease, box-shadow 0.2s ease;
                z-index: 5;
                will-change: transform;
            }

            .slider-enhanced-slider-button:hover {
                transform: scale(1.05);
                box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
            }

            .slider-enhanced-slider-button:active,
            .slider-enhanced-slider-button.dragging {
                cursor: grabbing;
                transform: scale(0.95);
                box-shadow: 0 2px 6px rgba(0, 0, 0, 0.15);
            }

            .slider-enhanced-slider-button.verifying {
                animation: sliderEnhancedVerify 0.6s ease-in-out infinite;
            }

            @keyframes sliderEnhancedVerify {
                0%, 100% { transform: scale(1); }
                50% { transform: scale(1.1); }
            }

            .slider-enhanced-slider-button.success {
                background: #28a745;
                color: white;
                animation: sliderEnhancedSuccess 0.5s ease-out;
            }

            @keyframes sliderEnhancedSuccess {
                0% { transform: scale(1); }
                50% { transform: scale(1.2); }
                100% { transform: scale(1); }
            }

            .slider-enhanced-slider-button.error {
                background: #dc3545;
                color: white;
                animation: sliderEnhancedError 0.5s ease-in-out;
            }

            @keyframes sliderEnhancedError {
                0%, 100% { transform: translateX(0); }
                20%, 60% { transform: translateX(-8px); }
                40%, 80% { transform: translateX(8px); }
            }

            .slider-enhanced-slider-button.shake {
                animation: sliderEnhancedShake 0.5s ease-in-out;
            }

            @keyframes sliderEnhancedShake {
                0%, 100% { transform: translateX(0); }
                10%, 30%, 50%, 70%, 90% { transform: translateX(-5px); }
                20%, 40%, 60%, 80% { transform: translateX(5px); }
            }

            .slider-enhanced-slider-button.bounce {
                animation: sliderEnhancedBounce 0.6s ease-out;
            }

            @keyframes sliderEnhancedBounce {
                0%, 100% { transform: translateY(0); }
                30% { transform: translateY(-10px); }
                50% { transform: translateY(-5px); }
                70% { transform: translateY(-3px); }
            }

            .slider-enhanced-slider-button i {
                color: #c9a96e;
                font-size: 16px;
            }

            .slider-enhanced-slider-button.success i,
            .slider-enhanced-slider-button.error i {
                color: white;
            }

            .slider-enhanced-slider-text {
                position: absolute;
                width: 100%;
                text-align: center;
                line-height: 44px;
                font-size: 13px;
                color: #6c757d;
                pointer-events: none;
                transition: opacity 0.3s ease;
            }

            .slider-enhanced-feedback {
                position: absolute;
                bottom: -30px;
                left: 50%;
                transform: translateX(-50%);
                padding: 4px 12px;
                border-radius: 12px;
                font-size: 12px;
                opacity: 0;
                transition: opacity 0.3s ease, transform 0.3s ease;
                white-space: nowrap;
            }

            .slider-enhanced-feedback.show {
                opacity: 1;
                transform: translateX(-50%) translateY(-5px);
            }

            .slider-enhanced-feedback.success {
                background: rgba(40, 167, 69, 0.1);
                color: #28a745;
                border: 1px solid rgba(40, 167, 69, 0.2);
            }

            .slider-enhanced-feedback.error {
                background: rgba(220, 53, 69, 0.1);
                color: #dc3545;
                border: 1px solid rgba(220, 53, 69, 0.2);
            }

            .slider-enhanced-particles-container {
                position: absolute;
                top: 0;
                left: 0;
                right: 0;
                bottom: 0;
                pointer-events: none;
                z-index: 10;
            }

            .slider-enhanced-particle {
                position: absolute;
                width: 8px;
                height: 8px;
                border-radius: 50%;
                pointer-events: none;
            }

            @keyframes particleFadeOut {
                0% { opacity: 1; transform: scale(1); }
                100% { opacity: 0; transform: scale(0); }
            }

            .slider-enhanced-loading-overlay {
                position: absolute;
                top: 0;
                left: 0;
                right: 0;
                bottom: 0;
                background: rgba(255, 255, 255, 0.8);
                display: flex;
                align-items: center;
                justify-content: center;
                z-index: 20;
                opacity: 0;
                visibility: hidden;
                transition: opacity 0.3s ease, visibility 0.3s ease;
            }

            .slider-enhanced-loading-overlay.show {
                opacity: 1;
                visibility: visible;
            }

            [data-theme="dark"] .slider-enhanced-container {
                background: #2d2d44;
            }

            [data-theme="dark"] .slider-enhanced-image-container {
                background: #1a1a2e;
            }

            [data-theme="dark"] .slider-enhanced-image-placeholder {
                background: linear-gradient(135deg, #2d2d44 0%, #1a1a2e 100%);
            }

            [data-theme="dark"] .slider-enhanced-slider-container {
                background: #1a1a2e;
            }

            @media (prefers-reduced-motion: reduce) {
                .slider-enhanced-spinner,
                .slider-enhanced-puzzle-shine,
                .slider-enhanced-guide-arrow,
                .slider-enhanced-slider-button.verifying,
                .slider-enhanced-slider-button.success,
                .slider-enhanced-slider-button.error,
                .slider-enhanced-slider-button.shake,
                .slider-enhanced-slider-button.bounce,
                .slider-enhanced-particle {
                    animation: none !important;
                    transition: none !important;
                }
            }
        `;
        document.head.appendChild(style);
    }

    bindEvents() {
        const button = this.elements.sliderButton;
        const container = this.elements.sliderContainer;

        const handleStart = (e) => {
            if (this.state.isDragging || this.state.isLoading) return;
            
            e.preventDefault();
            this.state.isDragging = true;
            
            const touch = e.touches ? e.touches[0] : e;
            this.touchIdentifier = touch.identifier;
            this.state.startX = touch.clientX;
            this.state.startY = touch.clientY;
            this.state.lastX = touch.clientX;
            this.state.lastTime = performance.now();
            this.state.velocity = 0;
            
            button.classList.add('dragging');
            this.elements.sliderText.textContent = '滑动中...';
            this.elements.guideArrow.style.opacity = '0';
            
            this.triggerHaptic();
        };

        const handleMove = (e) => {
            if (!this.state.isDragging) return;
            
            e.preventDefault();
            
            const touch = e.touches 
                ? Array.from(e.touches).find(t => t.identifier === this.touchIdentifier)
                : e;
            
            if (!touch) return;
            
            const now = performance.now();
            const deltaTime = now - this.state.lastTime;
            
            if (deltaTime < this.moveThrottle) return;
            
            const deltaX = touch.clientX - this.state.startX;
            const deltaY = touch.clientY - this.state.startY;
            
            if (Math.abs(deltaY) > Math.abs(deltaX) && Math.abs(deltaY) > 10) {
                this.handleDragCancel();
                return;
            }
            
            const velocity = (touch.clientX - this.state.lastX) / deltaTime;
            this.state.velocity = velocity * this.options.dragSmoothing + this.state.velocity * (1 - this.options.dragSmoothing);
            
            this.state.targetX = Math.max(0, Math.min(deltaX, this.state.maxX));
            
            const smoothedX = this.state.currentX + (this.state.targetX - this.state.currentX) * 0.3;
            this.state.currentX = smoothedX;
            
            this.updateSliderPosition(this.state.currentX);
            
            this.state.lastX = touch.clientX;
            this.state.lastTime = now;
        };

        const handleEnd = (e) => {
            if (!this.state.isDragging) return;
            
            this.state.isDragging = false;
            button.classList.remove('dragging');
            
            const hasMoved = this.state.currentX > 10;
            
            if (hasMoved) {
                this.elements.sliderText.textContent = '验证中...';
                this.performVerification();
            } else {
                this.elements.sliderText.textContent = '拖动滑块完成验证';
                this.elements.guideArrow.style.opacity = '1';
            }
        };

        button.addEventListener('mousedown', handleStart);
        button.addEventListener('touchstart', handleStart, { passive: false });
        
        document.addEventListener('mousemove', handleMove);
        document.addEventListener('touchmove', handleMove, { passive: false });
        
        document.addEventListener('mouseup', handleEnd);
        document.addEventListener('touchend', handleEnd);
        document.addEventListener('touchcancel', handleEnd);

        button.addEventListener('keydown', (e) => {
            if (this.state.isLoading) return;
            
            let delta = 0;
            switch (e.key) {
                case 'ArrowRight':
                case 'ArrowUp':
                    delta = 20;
                    break;
                case 'ArrowLeft':
                case 'ArrowDown':
                    delta = -20;
                    break;
                case 'Home':
                    delta = -this.state.currentX;
                    break;
                case 'End':
                    delta = this.state.maxX - this.state.currentX;
                    break;
                default:
                    return;
            }
            
            e.preventDefault();
            
            if (e.key === 'Enter' || e.key === ' ') {
                if (this.state.currentX > 10) {
                    this.performVerification();
                }
                return;
            }
            
            this.state.currentX = Math.max(0, Math.min(this.state.currentX + delta, this.state.maxX));
            this.updateSliderPosition(this.state.currentX);
            
            const progress = Math.round((this.state.currentX / this.state.maxX) * 100);
            button.setAttribute('aria-valuenow', progress);
        });

        this.eventHandlers = { handleStart, handleMove, handleEnd };
    }

    updateSliderPosition(x) {
        const button = this.elements.sliderButton;
        const track = this.elements.sliderTrack;
        const puzzle = this.elements.puzzle;
        
        requestAnimationFrame(() => {
            button.style.transform = `translateX(${x}px)`;
            track.style.width = `${x}px`;
            puzzle.style.transform = `translateX(${x}px)`;
        });
    }

    async preloadImages() {
        this.state.isLoading = true;
        this.showImagePlaceholder();
        
        try {
            const response = await fetch(`${this.options.apiBase}/captcha/slider?t=${Date.now()}`);
            
            if (!response.ok) {
                throw new Error('加载失败');
            }
            
            const data = await response.json();
            const imageData = data.data || data;
            
            if (imageData.image && imageData.puzzle_image) {
                await Promise.all([
                    this.loadImage(imageData.image),
                    this.loadImage(imageData.puzzle_image)
                ]);
                
                this.elements.image.src = imageData.image;
                this.elements.image.onload = () => {
                    this.state.imageLoaded = true;
                    this.hideImagePlaceholder();
                    this.animateImageIn();
                };
                
                this.sessionId = imageData.session_id || imageData.challenge_id;
                this.targetX = imageData.target_x || imageData.x || 100;
                this.targetY = imageData.target_y || imageData.y || 0;
                
                this.state.maxX = this.elements.sliderContainer.offsetWidth - 44;
                this.positionPuzzle(this.targetX, this.targetY);
                this.positionGuideArrow(this.targetX);
                
                this.state.isLoaded = true;
            } else {
                throw new Error('数据格式错误');
            }
        } catch (error) {
            console.error('SliderCaptchaEnhanced: 图片加载失败', error);
            this.showFeedback('加载失败，点击重试', 'error');
            this.elements.imagePlaceholder.querySelector('.slider-enhanced-loading-text').textContent = '加载失败';
            
            setTimeout(() => {
                this.preloadImages();
            }, 3000);
        }
    }

    loadImage(src) {
        return new Promise((resolve, reject) => {
            const img = new Image();
            img.onload = resolve;
            img.onerror = reject;
            img.src = src;
        });
    }

    showImagePlaceholder() {
        this.elements.imagePlaceholder.classList.remove('hidden');
        this.elements.image.classList.remove('loaded');
    }

    hideImagePlaceholder() {
        this.elements.imagePlaceholder.classList.add('hidden');
    }

    animateImageIn() {
        this.elements.image.classList.add('loaded');
        
        this.elements.puzzle.style.opacity = '0';
        this.elements.puzzle.style.transform = `translateX(${this.state.currentX}px) scale(0.8)`;
        
        setTimeout(() => {
            this.elements.puzzle.style.transition = 'opacity 0.3s ease, transform 0.3s ease';
            this.elements.puzzle.style.opacity = '1';
            this.elements.puzzle.style.transform = `translateX(${this.state.currentX}px) scale(1)`;
            
            setTimeout(() => {
                this.elements.puzzle.style.transition = 'transform 0.05s linear';
            }, 300);
        }, 100);
    }

    positionPuzzle(x, y) {
        this.elements.puzzle.style.left = `${x}px`;
        this.elements.puzzle.style.top = `${y}px`;
    }

    positionGuideArrow(targetX) {
        const containerWidth = this.elements.imageContainer.offsetWidth;
        const arrowX = Math.min(targetX + 50, containerWidth - 40);
        this.elements.guideArrow.style.right = `${containerWidth - arrowX - 20}px`;
    }

    showFeedback(message, type = 'info') {
        this.elements.feedback.textContent = message;
        this.elements.feedback.className = `slider-enhanced-feedback show ${type}`;
        
        setTimeout(() => {
            this.elements.feedback.classList.remove('show');
        }, 3000);
    }

    performVerification() {
        if (this.state.isLoading) return;
        
        this.state.isLoading = true;
        this.elements.sliderButton.classList.add('verifying');
        
        const payload = {
            session_id: this.sessionId,
            x: Math.round(this.state.currentX),
            y: this.targetY,
            type: 'slider',
            velocity: this.state.velocity
        };
        
        this.verifyWithServer(payload);
    }

    async verifyWithServer(payload) {
        try {
            const response = await fetch(`${this.options.apiBase}/captcha/verify-v2`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });
            
            let success = false;
            if (response.ok) {
                const data = await response.json();
                success = data.success || data.data?.success;
            }
            
            this.handleVerificationResult(success);
        } catch (error) {
            console.error('SliderCaptchaEnhanced: 验证请求失败', error);
            this.handleVerificationResult(false);
        }
    }

    handleVerificationResult(success) {
        this.state.isLoading = false;
        this.elements.sliderButton.classList.remove('verifying');
        
        if (success) {
            this.handleSuccess();
        } else {
            this.handleError();
        }
    }

    handleSuccess() {
        this.elements.sliderButton.classList.add('success');
        this.elements.sliderButton.innerHTML = '<i class="fas fa-check"></i>';
        this.showFeedback('验证成功', 'success');
        
        this.triggerHaptic('success');
        this.spawnParticles('#28a745', 20);
        
        if (this.options.enableParticles) {
            this.animateSuccessSequence();
        }
        
        if (this.options.onSuccess) {
            this.options.onSuccess({
                type: 'slider',
                session_id: this.sessionId
            });
        }
        
        this.disableInteraction();
    }

    handleError() {
        this.elements.sliderButton.classList.add('error');
        this.elements.sliderButton.innerHTML = '<i class="fas fa-times"></i>';
        this.showFeedback('验证失败', 'error');
        
        this.triggerHaptic('error');
        this.animateErrorSequence();
        
        if (this.options.onError) {
            this.options.onError({
                type: 'slider',
                error: '验证失败'
            });
        }
        
        setTimeout(() => {
            this.reset();
        }, 1500);
    }

    animateSuccessSequence() {
        const finalX = this.state.currentX;
        
        this.animateToPosition(finalX, 400, 'easeOutBack', () => {
            this.elements.sliderButton.classList.add('bounce');
            
            setTimeout(() => {
                this.elements.sliderButton.classList.remove('bounce');
            }, 600);
        });
    }

    animateErrorSequence() {
        const originalX = this.state.currentX;
        
        this.animateShake(() => {
            this.animateToPosition(0, 300, 'easeOutCubic', () => {
                this.elements.sliderButton.classList.remove('error');
                this.elements.sliderButton.innerHTML = '<i class="fas fa-arrows-alt-h"></i>';
                this.elements.sliderText.textContent = '拖动滑块完成验证';
                this.elements.guideArrow.style.opacity = '1';
            });
        });
    }

    animateShake(callback) {
        this.elements.sliderButton.classList.add('shake');
        
        setTimeout(() => {
            this.elements.sliderButton.classList.remove('shake');
            if (callback) callback();
        }, 500);
    }

    animateToPosition(targetX, duration, easing = 'linear', callback) {
        const startX = this.state.currentX;
        const startTime = performance.now();
        
        const easeFunctions = {
            linear: (t) => t,
            easeOutCubic: (t) => 1 - Math.pow(1 - t, 3),
            easeOutBack: (t) => {
                const c1 = 1.70158;
                const c3 = c1 + 1;
                return 1 + c3 * Math.pow(t - 1, 3) + c1 * Math.pow(t - 1, 2);
            }
        };
        
        const ease = easeFunctions[easing] || easeFunctions.linear;
        
        const animate = (currentTime) => {
            const elapsed = currentTime - startTime;
            const progress = Math.min(elapsed / duration, 1);
            const easedProgress = ease(progress);
            
            this.state.currentX = startX + (targetX - startX) * easedProgress;
            this.updateSliderPosition(this.state.currentX);
            
            if (progress < 1) {
                requestAnimationFrame(animate);
            } else {
                this.state.currentX = targetX;
                if (callback) callback();
            }
        };
        
        requestAnimationFrame(animate);
    }

    reset() {
        this.state.currentX = 0;
        this.state.targetX = 0;
        this.state.velocity = 0;
        this.state.isLoading = false;
        
        this.animateToPosition(0, 300, 'easeOutCubic');
        
        this.elements.sliderButton.classList.remove('error', 'verifying', 'success');
        this.elements.sliderButton.innerHTML = '<i class="fas fa-arrows-alt-h"></i>';
        this.elements.sliderText.textContent = '拖动滑块完成验证';
        this.elements.guideArrow.style.opacity = '1';
    }

    handleDragCancel() {
        this.state.isDragging = false;
        this.elements.sliderButton.classList.remove('dragging');
        this.elements.sliderText.textContent = '拖动滑块完成验证';
        
        this.animateToPosition(0, 200, 'easeOutCubic');
    }

    disableInteraction() {
        this.elements.sliderButton.style.pointerEvents = 'none';
        this.elements.sliderContainer.style.cursor = 'not-allowed';
    }

    enableInteraction() {
        this.elements.sliderButton.style.pointerEvents = 'auto';
        this.elements.sliderContainer.style.cursor = 'pointer';
    }

    spawnParticles(color, count = 10) {
        const button = this.elements.sliderButton;
        const rect = button.getBoundingClientRect();
        const containerRect = this.elements.wrapper.getBoundingClientRect();
        
        for (let i = 0; i < count; i++) {
            const particle = document.createElement('div');
            particle.className = 'slider-enhanced-particle';
            particle.style.background = color;
            particle.style.left = `${rect.left - containerRect.left + rect.width / 2}px`;
            particle.style.top = `${rect.top - containerRect.top + rect.height / 2}px`;
            
            const angle = (Math.PI * 2 * i) / count;
            const velocity = 50 + Math.random() * 50;
            const vx = Math.cos(angle) * velocity;
            const vy = Math.sin(angle) * velocity;
            
            this.elements.particlesContainer.appendChild(particle);
            
            let opacity = 1;
            let x = 0;
            let y = 0;
            let scale = 1;
            
            const animate = () => {
                opacity -= 0.02;
                x += vx * 0.02;
                y += vy * 0.02 + 2;
                scale -= 0.02;
                
                if (opacity <= 0) {
                    particle.remove();
                    return;
                }
                
                particle.style.opacity = opacity;
                particle.style.transform = `translate(${x}px, ${y}px) scale(${Math.max(0, scale)})`;
                
                requestAnimationFrame(animate);
            };
            
            requestAnimationFrame(animate);
        }
    }

    triggerHaptic(type = 'light') {
        if (!this.options.enableHaptic || !navigator.vibrate) return;
        
        switch (type) {
            case 'light':
                navigator.vibrate(10);
                break;
            case 'success':
                navigator.vibrate([50, 30, 50]);
                break;
            case 'error':
                navigator.vibrate([100, 50, 100]);
                break;
        }
    }

    refresh() {
        this.reset();
        this.enableInteraction();
        this.preloadImages();
    }

    destroy() {
        if (this.animationFrame) {
            cancelAnimationFrame(this.animationFrame);
        }
        
        Object.values(this.eventHandlers).forEach(handler => {
            document.removeEventListener('mousemove', handler);
            document.removeEventListener('mouseup', handler);
            document.removeEventListener('touchmove', handler);
            document.removeEventListener('touchend', handler);
        });
        
        if (this.elements.wrapper && this.elements.wrapper.parentNode) {
            this.elements.wrapper.parentNode.removeChild(this.elements.wrapper);
        }
        
        const style = document.getElementById('slider-enhanced-styles');
        if (style) {
            style.remove();
        }
    }

    getState() {
        return { ...this.state };
    }

    setOption(key, value) {
        if (this.options.hasOwnProperty(key)) {
            this.options[key] = value;
        }
    }
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = SliderCaptchaEnhanced;
}
