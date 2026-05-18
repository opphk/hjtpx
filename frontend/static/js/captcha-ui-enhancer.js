// 验证码UI优化脚本 - 增强交互和动画效果

(function() {
    'use strict';

    // BootCDN资源加载监控
    class CDNResourceMonitor {
        constructor() {
            this.resources = [
                'https://cdn.bootcdn.net/ajax/libs/twitter-bootstrap/5.3.8/css/bootstrap.min.css',
                'https://cdn.bootcdn.net/ajax/libs/font-awesome/6.5.1/css/all.min.css'
            ];
            this.failedResources = [];
            this.init();
        }

        init() {
            this.resources.forEach(url => {
                this.checkResource(url);
            });
        }

        checkResource(url) {
            const link = document.createElement('link');
            link.rel = 'stylesheet';
            link.href = url;
            
            link.onload = () => {
                console.log(`✓ BootCDN资源加载成功: ${url}`);
            };
            
            link.onerror = () => {
                console.error(`✗ BootCDN资源加载失败: ${url}`);
                this.failedResources.push(url);
                this.showResourceError(url);
            };
            
            document.head.appendChild(link);
        }

        showResourceError(url) {
            console.warn('部分BootCDN资源加载失败,可能影响页面样式');
        }

        getFailedResources() {
            return this.failedResources;
        }
    }

    // 性能监控增强
    class EnhancedPerformanceMonitor {
        constructor() {
            this.metrics = {
                pageLoadTime: 0,
                fcp: 0,
                lcp: 0,
                fid: 0,
                cls: 0
            };
            this.init();
        }

        init() {
            if (window.performance) {
                const timing = window.performance.timing;
                this.metrics.pageLoadTime = timing.loadEventEnd - timing.navigationStart;
                
                window.addEventListener('load', () => {
                    setTimeout(() => {
                        this.updateMetricsDisplay();
                        this.observeCLS();
                    }, 100);
                });

                if (window.PerformanceObserver) {
                    this.observeFID();
                    this.observeLCP();
                }
            }
        }

        observeFID() {
            try {
                const fidObserver = new PerformanceObserver((list) => {
                    for (const entry of list.getEntries()) {
                        this.metrics.fid = entry.processingStart - entry.startTime;
                    }
                });
                fidObserver.observe({ type: 'first-input', buffered: true });
            } catch (e) {
                console.log('FID observer not supported');
            }
        }

        observeLCP() {
            try {
                const lcpObserver = new PerformanceObserver((list) => {
                    const entries = list.getEntries();
                    const lastEntry = entries[entries.length - 1];
                    this.metrics.lcp = lastEntry.renderTime || lastEntry.loadTime;
                });
                lcpObserver.observe({ type: 'largest-contentful-paint', buffered: true });
            } catch (e) {
                console.log('LCP observer not supported');
            }
        }

        observeCLS() {
            try {
                let clsValue = 0;
                const clsObserver = new PerformanceObserver((list) => {
                    for (const entry of list.getEntries()) {
                        if (!entry.hadRecentInput) {
                            clsValue += entry.value;
                        }
                    }
                    this.metrics.cls = clsValue;
                });
                clsObserver.observe({ type: 'layout-shift', buffered: true });
            } catch (e) {
                console.log('CLS observer not supported');
            }
        }

        updateMetricsDisplay() {
            const metricsEl = document.getElementById('performanceMetrics');
            if (metricsEl) {
                metricsEl.classList.add('show');
                
                const loadTimeEl = document.getElementById('pageLoadTime');
                if (loadTimeEl) {
                    loadTimeEl.textContent = this.metrics.pageLoadTime + 'ms';
                    loadTimeEl.className = 'metric-value ' + this.getMetricClass(this.metrics.pageLoadTime, 2000);
                }

                const responseTimeEl = document.getElementById('verifyResponseTime');
                if (responseTimeEl) {
                    const responseTime = Math.round(this.metrics.fid);
                    responseTimeEl.textContent = responseTime > 0 ? responseTime + 'ms' : '--';
                    responseTimeEl.className = 'metric-value ' + this.getMetricClass(responseTime, 100);
                }

                const fpsEl = document.getElementById('fpsMetric');
                if (fpsEl) {
                    fpsEl.textContent = this.getCurrentFPS();
                    fpsEl.className = 'metric-value ' + this.getMetricClass(60 - this.getCurrentFPS(), 15);
                }
            }
        }

        getMetricClass(value, threshold) {
            if (value < threshold * 0.5) return 'good';
            if (value < threshold) return 'warning';
            return 'bad';
        }

        getCurrentFPS() {
            return Math.round(1000 / (this.metrics.fid || 16.67));
        }

        getAllMetrics() {
            return { ...this.metrics };
        }
    }

    // 增强的滑块验证组件
    class EnhancedSliderCaptcha {
        constructor(element, options = {}) {
            this.element = typeof element === 'string' 
                ? document.querySelector(element) 
                : element;
            
            if (!this.element) return;

            this.options = {
                onSuccess: null,
                onError: null,
                onRefresh: null,
                onDragStart: null,
                onDrag: null,
                onDragEnd: null,
                validationThreshold: 0.95,
                animationDuration: 300,
                touchOptimized: true,
                keyboardEnabled: true,
                ...options
            };

            this.isDragging = false;
            this.startX = 0;
            this.currentX = 0;
            this.maxX = 0;
            this.startTime = 0;
            this.trajectoryData = [];
            this.init();
        }

        init() {
            this.setupUI();
            this.bindEvents();
            this.setupAccessibility();
        }

        setupUI() {
            const container = document.createElement('div');
            container.className = 'captcha-slider-container';
            container.innerHTML = `
                <div class="captcha-slider-track"></div>
                <div class="captcha-slider-text" data-i18n="sliderText">向右滑动完成验证</div>
                <div class="captcha-slider-button" role="slider" aria-label="滑块验证" aria-valuemin="0" aria-valuemax="100" aria-valuenow="0" tabindex="0">
                    <i class="fas fa-chevron-right" aria-hidden="true"></i>
                </div>
                <div class="captcha-slider-hint" role="tooltip">
                    <i class="fas fa-info-circle" aria-hidden="true"></i>
                    <span data-i18n="sliderHint">按住滑块拖动到最右侧</span>
                </div>
            `;
            
            this.element.innerHTML = '';
            this.element.appendChild(container);
            
            this.container = container;
            this.track = container.querySelector('.captcha-slider-track');
            this.text = container.querySelector('.captcha-slider-text');
            this.button = container.querySelector('.captcha-slider-button');
            this.hint = container.querySelector('.captcha-slider-hint');
        }

        setupAccessibility() {
            if (!this.options.keyboardEnabled) return;
            
            this.container.setAttribute('role', 'slider');
            this.container.setAttribute('aria-valuemin', '0');
            this.container.setAttribute('aria-valuemax', '100');
            this.container.setAttribute('aria-valuenow', '0');
            this.container.setAttribute('aria-label', '滑块验证码，请拖动完成验证');
            this.container.setAttribute('tabindex', '0');
        }

        bindEvents() {
            // 鼠标事件
            this.button.addEventListener('mousedown', (e) => this.startDrag(e));
            document.addEventListener('mousemove', (e) => this.drag(e));
            document.addEventListener('mouseup', (e) => this.endDrag(e));
            
            // 触摸事件 - 优化移动端体验
            if (this.options.touchOptimized) {
                this.button.addEventListener('touchstart', (e) => this.startDrag(e), { passive: false });
                document.addEventListener('touchmove', (e) => this.drag(e), { passive: false });
                document.addEventListener('touchend', (e) => this.endDrag(e));
                document.addEventListener('touchcancel', (e) => this.endDrag(e));
            }

            // 键盘事件
            if (this.options.keyboardEnabled) {
                this.container.addEventListener('keydown', (e) => this.handleKeyboard(e));
            }

            // 点击重置
            this.text.addEventListener('click', () => this.reset());
        }

        startDrag(e) {
            e.preventDefault();
            e.stopPropagation();
            
            this.isDragging = true;
            this.startX = e.type === 'mousedown' ? e.clientX : e.touches[0].clientX;
            this.startTime = Date.now();
            this.trajectoryData = [{ x: this.startX, t: this.startTime }];
            this.maxX = this.container.offsetWidth - this.button.offsetWidth - 4;
            
            this.button.classList.add('dragging');
            this.button.style.cursor = 'grabbing';
            this.text.textContent = this.getTranslation('dragging');
            
            this.container.setAttribute('aria-grabbed', 'true');
            this.container.setAttribute('aria-valuenow', '0');
            
            if (this.options.onDragStart) {
                this.options.onDragStart({ timestamp: this.startTime });
            }
            
            // 防止文本选择和页面滚动
            document.body.style.userSelect = 'none';
            document.body.style.webkitUserSelect = 'none';
        }

        drag(e) {
            if (!this.isDragging) return;
            e.preventDefault();
            
            const clientX = e.type === 'mousemove' ? e.clientX : e.touches[0].clientX;
            const deltaX = clientX - this.startX;
            const progress = Math.max(0, Math.min(deltaX / this.maxX, 1));
            
            this.currentX = progress * this.maxX;
            this.button.style.left = (this.currentX + 2) + 'px';
            this.track.style.width = (progress * this.maxX) + 'px';
            
            const currentTime = Date.now();
            this.trajectoryData.push({ x: clientX, t: currentTime });
            
            const percent = Math.round(progress * 100);
            this.container.setAttribute('aria-valuenow', percent.toString());
            this.text.textContent = `${percent}%`;
            
            if (this.options.onDrag) {
                this.options.onDrag({
                    progress: progress,
                    percent: percent,
                    trajectory: this.trajectoryData
                });
            }
        }

        endDrag(e) {
            if (!this.isDragging) return;
            
            this.isDragging = false;
            this.button.classList.remove('dragging');
            this.button.style.cursor = 'grab';
            
            this.container.setAttribute('aria-grabbed', 'false');
            
            // 恢复文本选择
            document.body.style.userSelect = '';
            document.body.style.webkitUserSelect = '';
            
            const progress = this.currentX / this.maxX;
            const duration = Date.now() - this.startTime;
            
            if (this.options.onDragEnd) {
                this.options.onDragEnd({
                    progress: progress,
                    duration: duration,
                    trajectory: this.trajectoryData
                });
            }
            
            if (progress >= this.options.validationThreshold) {
                this.showSuccess();
            } else if (progress > 0.1) {
                this.showError();
            } else {
                this.reset();
            }
        }

        handleKeyboard(e) {
            const step = 10;
            let newProgress = this.currentX / this.maxX;
            
            switch(e.key) {
                case 'ArrowRight':
                case 'ArrowUp':
                    e.preventDefault();
                    newProgress = Math.min(newProgress + step / 100, 1);
                    break;
                case 'ArrowLeft':
                case 'ArrowDown':
                    e.preventDefault();
                    newProgress = Math.max(newProgress - step / 100, 0);
                    break;
                case 'Home':
                    e.preventDefault();
                    newProgress = 0;
                    break;
                case 'End':
                    e.preventDefault();
                    newProgress = 1;
                    break;
                case 'Enter':
                case ' ':
                    e.preventDefault();
                    if (newProgress >= this.options.validationThreshold) {
                        this.showSuccess();
                    }
                    return;
                default:
                    return;
            }
            
            this.currentX = newProgress * this.maxX;
            this.updateUI();
            
            const percent = Math.round(newProgress * 100);
            this.container.setAttribute('aria-valuenow', percent.toString());
            this.text.textContent = `${percent}%`;
            
            if (newProgress >= this.options.validationThreshold) {
                setTimeout(() => this.showSuccess(), 200);
            }
        }

        updateUI() {
            const progress = this.currentX / this.maxX;
            this.button.style.left = (this.currentX + 2) + 'px';
            this.track.style.width = (progress * this.maxX) + 'px';
        }

        showSuccess() {
            this.container.classList.add('success');
            this.text.textContent = this.getTranslation('success');
            this.button.innerHTML = '<i class="fas fa-check" aria-hidden="true"></i>';
            this.button.classList.add('success');
            
            this.container.setAttribute('aria-valuenow', '100');
            
            if (this.options.onSuccess) {
                this.options.onSuccess({
                    duration: Date.now() - this.startTime,
                    trajectory: this.trajectoryData
                });
            }
            
            setTimeout(() => {
                if (this.options.onRefresh) {
                    this.options.onRefresh();
                }
                this.reset();
            }, this.options.animationDuration + 500);
        }

        showError() {
            this.container.classList.add('error');
            this.text.textContent = this.getTranslation('error');
            this.button.innerHTML = '<i class="fas fa-times" aria-hidden="true"></i>';
            this.button.classList.add('error');
            
            if (this.options.onError) {
                this.options.onError({
                    progress: this.currentX / this.maxX
                });
            }
            
            setTimeout(() => {
                this.reset();
            }, this.options.animationDuration + 500);
        }

        reset() {
            this.container.classList.remove('success', 'error');
            this.button.classList.remove('success', 'error');
            this.currentX = 0;
            this.updateUI();
            this.text.textContent = this.getTranslation('sliderText');
            this.button.innerHTML = '<i class="fas fa-chevron-right" aria-hidden="true"></i>';
            this.container.setAttribute('aria-valuenow', '0');
        }

        getTranslation(key) {
            const translations = {
                'sliderText': '向右滑动完成验证',
                'dragging': '滑动中...',
                'success': '验证成功!',
                'error': '验证失败，请重试'
            };
            return translations[key] || key;
        }

        destroy() {
            this.element.innerHTML = '';
            this.element = null;
        }

        refresh() {
            this.reset();
        }

        enable() {
            this.container.removeAttribute('disabled');
            this.button.removeAttribute('disabled');
        }

        disable() {
            this.container.setAttribute('disabled', 'true');
            this.button.setAttribute('disabled', 'true');
        }
    }

    // 增强的点选验证组件
    class EnhancedClickCaptcha {
        constructor(element, options = {}) {
            this.element = typeof element === 'string' 
                ? document.querySelector(element) 
                : element;
            
            if (!this.element) return;

            this.options = {
                maxPoints: 3,
                imageUrl: null,
                onSuccess: null,
                onError: null,
                onPointSelect: null,
                animationEnabled: true,
                ...options
            };

            this.selectedPoints = [];
            this.markers = [];
            this.init();
        }

        init() {
            this.setupUI();
            this.bindEvents();
            this.loadImage();
        }

        setupUI() {
            const container = document.createElement('div');
            container.className = 'captcha-click-grid';
            container.innerHTML = `
                <div class="captcha-click-image-wrapper" style="position: relative; background: #f0f0f0; border-radius: var(--captcha-radius-sm); overflow: hidden;">
                    <div class="captcha-image-skeleton active" id="clickSkeleton"></div>
                    <img class="captcha-click-image" alt="点选验证码图片" style="display: none; width: 100%; height: auto;" />
                </div>
                <div class="captcha-click-hint-panel">
                    <div class="hint-text">
                        <i class="fas fa-lightbulb" aria-hidden="true"></i>
                        <span data-i18n="clickHint">请依次点击图中的文字</span>
                    </div>
                </div>
                <div class="captcha-click-progress" role="status" aria-live="polite">
                    <span>已选择: </span>
                    <span class="count-badge" id="clickCountBadge">0</span>
                    <span>/</span>
                    <span>${this.options.maxPoints}</span>
                </div>
            `;
            
            this.element.innerHTML = '';
            this.element.appendChild(container);
            
            this.container = container;
            this.image = container.querySelector('.captcha-click-image');
            this.skeleton = container.querySelector('.captcha-image-skeleton');
            this.progressBadge = container.querySelector('#clickCountBadge');
        }

        bindEvents() {
            this.container.addEventListener('click', (e) => this.handleClick(e));
        }

        loadImage() {
            if (this.options.imageUrl) {
                this.image.onload = () => {
                    this.image.style.display = 'block';
                    this.skeleton.classList.remove('active');
                };
                this.image.onerror = () => {
                    this.skeleton.classList.remove('active');
                    this.showErrorHint('图片加载失败');
                };
                this.image.src = this.options.imageUrl;
            } else {
                setTimeout(() => {
                    this.skeleton.classList.remove('active');
                    this.image.style.display = 'block';
                    this.image.src = 'data:image/svg+xml,' + encodeURIComponent(this.generatePlaceholderSVG());
                }, 500);
            }
        }

        generatePlaceholderSVG() {
            return `<svg xmlns="http://www.w3.org/2000/svg" width="400" height="300" viewBox="0 0 400 300">
                <rect fill="#f5f5f5" width="400" height="300"/>
                <text x="200" y="150" text-anchor="middle" fill="#999" font-family="Arial" font-size="14">点击验证图片</text>
            </svg>`;
        }

        handleClick(e) {
            if (e.target.classList.contains('captcha-click-marker')) return;
            
            if (this.selectedPoints.length >= this.options.maxPoints) {
                return;
            }

            const rect = this.container.querySelector('.captcha-click-image-wrapper').getBoundingClientRect();
            const x = e.clientX - rect.left;
            const y = e.clientY - rect.top;
            
            // 检查是否在图片范围内
            if (x < 0 || y < 0 || x > rect.width || y > rect.height) return;
            
            const point = { 
                x: (x / rect.width) * 100, 
                y: (y / rect.height) * 100,
                index: this.selectedPoints.length + 1 
            };
            this.selectedPoints.push(point);
            
            this.addMarker(point);
            this.updateProgress();
            
            if (this.options.onPointSelect) {
                this.options.onPointSelect(point);
            }
            
            if (this.selectedPoints.length === this.options.maxPoints) {
                setTimeout(() => this.verify(), 500);
            }
        }

        addMarker(point) {
            const marker = document.createElement('div');
            marker.className = 'captcha-click-marker';
            marker.style.left = point.x + '%';
            marker.style.top = point.y + '%';
            marker.textContent = point.index;
            marker.dataset.index = point.index;
            marker.setAttribute('role', 'button');
            marker.setAttribute('aria-label', `已选择第 ${point.index} 个点，点击可删除`);
            
            marker.addEventListener('click', (e) => {
                e.stopPropagation();
                this.removePoint(point.index);
            });
            
            this.container.querySelector('.captcha-click-image-wrapper').appendChild(marker);
            this.markers.push(marker);
            
            if (this.options.animationEnabled) {
                setTimeout(() => {
                    marker.classList.add('selected');
                }, 50);
            } else {
                marker.classList.add('selected');
            }
        }

        removePoint(index) {
            const marker = this.container.querySelector(`.captcha-click-marker[data-index="${index}"]`);
            if (marker) {
                if (this.options.animationEnabled) {
                    marker.style.opacity = '0';
                    marker.style.transform = 'translate(-50%, -50%) scale(0)';
                    setTimeout(() => marker.remove(), 200);
                } else {
                    marker.remove();
                }
            }
            
            this.selectedPoints = this.selectedPoints.filter(p => p.index !== index);
            this.renumberMarkers();
            this.updateProgress();
        }

        renumberMarkers() {
            this.selectedPoints.forEach((point, idx) => {
                point.index = idx + 1;
                const marker = this.container.querySelector(`.captcha-click-marker[data-index="${idx + 1}"]`);
                if (marker) {
                    marker.textContent = idx + 1;
                    marker.setAttribute('aria-label', `已选择第 ${idx + 1} 个点，点击可删除`);
                }
            });
        }

        updateProgress() {
            this.progressBadge.textContent = this.selectedPoints.length;
            
            if (this.selectedPoints.length === this.options.maxPoints) {
                this.progressBadge.classList.add('complete');
                this.progressBadge.parentElement.style.color = 'var(--captcha-success)';
            } else {
                this.progressBadge.classList.remove('complete');
                this.progressBadge.parentElement.style.color = '';
            }
        }

        verify() {
            try {
                if (this.options.onSuccess) {
                    this.options.onSuccess(this.selectedPoints);
                }
            } catch (error) {
                if (this.options.onError) {
                    this.options.onError(error);
                }
            }
            
            setTimeout(() => this.reset(), 1500);
        }

        showErrorHint(message) {
            const errorHint = document.createElement('div');
            errorHint.className = 'captcha-error-hint';
            errorHint.innerHTML = `
                <i class="fas fa-exclamation-circle" aria-hidden="true"></i>
                <div class="captcha-error-hint-text">
                    <div class="captcha-error-hint-title">验证失败</div>
                    <div class="captcha-error-hint-desc">${message}</div>
                </div>
            `;
            this.element.appendChild(errorHint);
            
            setTimeout(() => errorHint.remove(), 3000);
        }

        reset() {
            this.selectedPoints = [];
            this.markers.forEach(marker => marker.remove());
            this.markers = [];
            this.progressBadge.textContent = '0';
            this.progressBadge.classList.remove('complete');
            this.progressBadge.parentElement.style.color = '';
        }

        destroy() {
            this.element.innerHTML = '';
            this.element = null;
        }

        refresh() {
            this.reset();
            this.loadImage();
        }
    }

    // 全局成功庆祝动画
    function showSuccessCelebration(options = {}) {
        const config = {
            particleCount: options.particleCount || 50,
            colors: options.colors || ['#c9a96e', '#d4b87a', '#28a745', '#ffc107', '#dc3545'],
            duration: options.duration || 3000,
            ...options
        };
        
        const confettiContainer = document.createElement('div');
        confettiContainer.className = 'captcha-confetti';
        confettiContainer.setAttribute('aria-hidden', 'true');
        document.body.appendChild(confettiContainer);
        
        for (let i = 0; i < config.particleCount; i++) {
            const confetti = document.createElement('div');
            confetti.className = 'captcha-confetti-piece';
            confetti.style.left = Math.random() * 100 + '%';
            confetti.style.backgroundColor = config.colors[Math.floor(Math.random() * config.colors.length)];
            confetti.style.animationDelay = Math.random() * 2 + 's';
            confetti.style.animationDuration = (2 + Math.random() * 2) + 's';
            
            if (Math.random() > 0.5) {
                confetti.style.borderRadius = '50%';
            }
            
            confettiContainer.appendChild(confetti);
        }
        
        setTimeout(() => {
            confettiContainer.remove();
        }, config.duration);
    }

    // 增强的错误提示
    function showEnhancedError(message, container, options = {}) {
        const config = {
            duration: options.duration || 5000,
            showRetry: options.showRetry !== false,
            title: options.title || '验证失败',
            onRetry: options.onRetry || null,
            ...options
        };
        
        const errorHint = document.createElement('div');
        errorHint.className = 'captcha-error-hint';
        errorHint.setAttribute('role', 'alert');
        errorHint.setAttribute('aria-live', 'assertive');
        
        let htmlContent = `
            <i class="fas fa-exclamation-circle" aria-hidden="true"></i>
            <div class="captcha-error-hint-text">
                <div class="captcha-error-hint-title">${config.title}</div>
                <div class="captcha-error-hint-desc">${message}</div>
        `;
        
        if (config.showRetry) {
            htmlContent += `
                <div class="captcha-error-hint-action">
                    <button type="button" aria-label="重试验证码">
                        <i class="fas fa-redo me-1" aria-hidden="true"></i>重试
                    </button>
                </div>
            `;
        }
        
        htmlContent += '</div>';
        errorHint.innerHTML = htmlContent;
        
        if (config.showRetry) {
            const retryBtn = errorHint.querySelector('button');
            retryBtn.addEventListener('click', () => {
                if (config.onRetry) {
                    config.onRetry();
                }
                errorHint.remove();
            });
        }
        
        if (container) {
            container.appendChild(errorHint);
            
            setTimeout(() => {
                errorHint.style.opacity = '0';
                errorHint.style.transform = 'translateY(-10px)';
                setTimeout(() => errorHint.remove(), 300);
            }, config.duration);
        }
        
        return errorHint;
    }

    // 增强的成功提示
    function showEnhancedSuccess(message, container, options = {}) {
        const config = {
            duration: options.duration || 3000,
            title: options.title || '验证成功',
            ...options
        };
        
        const successHint = document.createElement('div');
        successHint.className = 'captcha-success-hint';
        successHint.setAttribute('role', 'status');
        successHint.setAttribute('aria-live', 'polite');
        successHint.innerHTML = `
            <i class="fas fa-check-circle" aria-hidden="true"></i>
            <div class="captcha-success-hint-text">
                <div class="captcha-success-hint-title">${config.title}</div>
                <div class="captcha-success-hint-desc">${message}</div>
            </div>
        `;
        
        if (container) {
            container.appendChild(successHint);
            
            setTimeout(() => {
                successHint.style.opacity = '0';
                successHint.style.transform = 'translateY(10px)';
                setTimeout(() => successHint.remove(), 300);
            }, config.duration);
        }
        
        return successHint;
    }

    // Toast通知系统
    class ToastNotification {
        constructor() {
            this.container = null;
            this.init();
        }

        init() {
            this.container = document.createElement('div');
            this.container.className = 'captcha-toast-container';
            this.container.setAttribute('role', 'region');
            this.container.setAttribute('aria-label', '通知');
            document.body.appendChild(this.container);
        }

        show(message, type = 'info', options = {}) {
            const toast = document.createElement('div');
            toast.className = `captcha-toast ${type}`;
            
            const icons = {
                success: 'fa-check-circle',
                error: 'fa-exclamation-circle',
                warning: 'fa-exclamation-triangle',
                info: 'fa-info-circle'
            };
            
            toast.innerHTML = `
                <div class="captcha-toast-icon">
                    <i class="fas ${icons[type]}" aria-hidden="true"></i>
                </div>
                <div class="captcha-toast-content">
                    <div class="captcha-toast-title">${options.title || ''}</div>
                    <div class="captcha-toast-message">${message}</div>
                </div>
                <button class="captcha-toast-close" aria-label="关闭通知">
                    <i class="fas fa-times" aria-hidden="true"></i>
                </button>
            `;
            
            toast.querySelector('.captcha-toast-close').addEventListener('click', () => {
                this.dismiss(toast);
            });
            
            this.container.appendChild(toast);
            
            // 自动消失
            const duration = options.duration || 3000;
            if (duration > 0) {
                setTimeout(() => this.dismiss(toast), duration);
            }
            
            return toast;
        }

        dismiss(toast) {
            toast.classList.add('removing');
            setTimeout(() => toast.remove(), 300);
        }

        success(message, title, options) {
            return this.show(message, 'success', { title, ...options });
        }

        error(message, title, options) {
            return this.show(message, 'error', { title, ...options });
        }

        warning(message, title, options) {
            return this.show(message, 'warning', { title, ...options });
        }

        info(message, title, options) {
            return this.show(message, 'info', { title, ...options });
        }
    }

    // 导出到全局
    window.CaptchaUIPack = {
        EnhancedSliderCaptcha,
        EnhancedClickCaptcha,
        CDNResourceMonitor,
        EnhancedPerformanceMonitor,
        ToastNotification,
        showSuccessCelebration,
        showEnhancedError,
        showEnhancedSuccess
    };

    // 初始化
    document.addEventListener('DOMContentLoaded', () => {
        // 监控CDN资源
        new CDNResourceMonitor();
        
        // 初始化性能监控
        const perfMonitor = new EnhancedPerformanceMonitor();
        window.captchaPerfMonitor = perfMonitor;
        
        // 初始化Toast通知
        const toast = new ToastNotification();
        window.captchaToast = toast;
        
        // 自动初始化增强的滑块验证
        document.querySelectorAll('[data-captcha-slider]').forEach(el => {
            new EnhancedSliderCaptcha(el);
        });
        
        // 自动初始化增强的点选验证
        document.querySelectorAll('[data-captcha-click]').forEach(el => {
            new EnhancedClickCaptcha(el);
        });
        
        // 注册Service Worker (如果可用)
        if ('serviceWorker' in navigator) {
            navigator.serviceWorker.register('/sw.js')
                .then(registration => {
                    console.log('Service Worker注册成功:', registration.scope);
                })
                .catch(error => {
                    console.log('Service Worker注册失败:', error);
                });
        }
    });

})();
