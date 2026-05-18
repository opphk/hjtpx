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
                ...options
            };

            this.isDragging = false;
            this.startX = 0;
            this.currentX = 0;
            this.maxX = 0;
            this.init();
        }

        init() {
            this.setupUI();
            this.bindEvents();
        }

        setupUI() {
            const container = document.createElement('div');
            container.className = 'captcha-slider-container';
            container.innerHTML = `
                <div class="captcha-slider-track"></div>
                <div class="captcha-slider-text">向右滑动完成验证</div>
                <div class="captcha-slider-button">
                    <i class="fas fa-chevron-right"></i>
                </div>
                <div class="captcha-slider-hint">
                    <i class="fas fa-info-circle"></i>
                    <span>按住滑块拖动到最右侧</span>
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

        bindEvents() {
            // 鼠标事件
            this.button.addEventListener('mousedown', (e) => this.startDrag(e));
            document.addEventListener('mousemove', (e) => this.drag(e));
            document.addEventListener('mouseup', (e) => this.endDrag(e));
            
            // 触摸事件
            this.button.addEventListener('touchstart', (e) => this.startDrag(e), { passive: false });
            document.addEventListener('touchmove', (e) => this.drag(e), { passive: false });
            document.addEventListener('touchend', (e) => this.endDrag(e));

            // 键盘事件
            this.container.setAttribute('tabindex', '0');
            this.container.addEventListener('keydown', (e) => this.handleKeyboard(e));
        }

        startDrag(e) {
            e.preventDefault();
            this.isDragging = true;
            this.startX = e.type === 'mousedown' ? e.clientX : e.touches[0].clientX;
            this.maxX = this.container.offsetWidth - this.button.offsetWidth - 4;
            this.button.classList.add('dragging');
            this.text.textContent = '滑动中...';
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
        }

        endDrag(e) {
            if (!this.isDragging) return;
            this.isDragging = false;
            this.button.classList.remove('dragging');
            
            const progress = this.currentX / this.maxX;
            
            if (progress > 0.95) {
                this.showSuccess();
            } else if (progress > 0.1) {
                this.showError();
            } else {
                this.reset();
            }
        }

        handleKeyboard(e) {
            if (e.key === 'ArrowRight') {
                e.preventDefault();
                this.currentX = Math.min(this.currentX + 10, this.maxX);
                this.updateUI();
                
                if (this.currentX >= this.maxX) {
                    this.showSuccess();
                }
            } else if (e.key === 'ArrowLeft') {
                e.preventDefault();
                this.currentX = Math.max(this.currentX - 10, 0);
                this.updateUI();
            } else if (e.key === 'Enter' && this.currentX >= this.maxX * 0.95) {
                e.preventDefault();
                this.showSuccess();
            }
        }

        updateUI() {
            const progress = this.currentX / this.maxX;
            this.button.style.left = (this.currentX + 2) + 'px';
            this.track.style.width = (progress * this.maxX) + 'px';
        }

        showSuccess() {
            this.container.classList.add('success');
            this.text.textContent = '验证成功!';
            this.button.innerHTML = '<i class="fas fa-check"></i>';
            
            if (this.options.onSuccess) {
                this.options.onSuccess();
            }
            
            setTimeout(() => {
                if (this.options.onRefresh) {
                    this.options.onRefresh();
                }
                this.reset();
            }, 1500);
        }

        showError() {
            this.container.classList.add('error');
            this.text.textContent = '验证失败，请重试';
            this.button.innerHTML = '<i class="fas fa-times"></i>';
            
            if (this.options.onError) {
                this.options.onError();
            }
            
            setTimeout(() => {
                this.reset();
            }, 1000);
        }

        reset() {
            this.container.classList.remove('success', 'error');
            this.currentX = 0;
            this.updateUI();
            this.text.textContent = '向右滑动完成验证';
            this.button.innerHTML = '<i class="fas fa-chevron-right"></i>';
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
                onSuccess: null,
                onError: null,
                ...options
            };

            this.selectedPoints = [];
            this.init();
        }

        init() {
            this.setupUI();
            this.bindEvents();
        }

        setupUI() {
            const container = document.createElement('div');
            container.className = 'captcha-click-grid';
            container.innerHTML = `
                <div class="captcha-click-hint-panel">
                    <div class="hint-text">
                        <i class="fas fa-lightbulb"></i>
                        <span>请依次点击图中的文字</span>
                    </div>
                </div>
                <div class="captcha-click-progress">
                    <span>已选择: </span>
                    <span class="count-badge">0</span>
                    <span>/</span>
                    <span>${this.options.maxPoints}</span>
                </div>
            `;
            
            this.element.innerHTML = '';
            this.element.appendChild(container);
            
            this.container = container;
            this.progressBadge = container.querySelector('.count-badge');
        }

        bindEvents() {
            this.container.addEventListener('click', (e) => this.handleClick(e));
        }

        handleClick(e) {
            if (this.selectedPoints.length >= this.options.maxPoints) {
                return;
            }

            const rect = this.container.getBoundingClientRect();
            const x = e.clientX - rect.left;
            const y = e.clientY - rect.top;
            
            const point = { x, y, index: this.selectedPoints.length + 1 };
            this.selectedPoints.push(point);
            
            this.addMarker(point);
            this.updateProgress();
            
            if (this.selectedPoints.length === this.options.maxPoints) {
                setTimeout(() => this.verify(), 500);
            }
        }

        addMarker(point) {
            const marker = document.createElement('div');
            marker.className = 'captcha-click-marker';
            marker.style.left = point.x + 'px';
            marker.style.top = point.y + 'px';
            marker.textContent = point.index;
            marker.dataset.index = point.index;
            
            marker.addEventListener('click', () => this.removePoint(point.index));
            
            this.container.appendChild(marker);
            
            setTimeout(() => {
                marker.classList.add('selected');
            }, 50);
        }

        removePoint(index) {
            const marker = this.container.querySelector(`.captcha-click-marker[data-index="${index}"]`);
            if (marker) {
                marker.remove();
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
                }
            });
        }

        updateProgress() {
            this.progressBadge.textContent = this.selectedPoints.length;
            
            if (this.selectedPoints.length === this.options.maxPoints) {
                this.progressBadge.classList.add('complete');
            } else {
                this.progressBadge.classList.remove('complete');
            }
        }

        async verify() {
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

        reset() {
            this.selectedPoints = [];
            const markers = this.container.querySelectorAll('.captcha-click-marker');
            markers.forEach(marker => marker.remove());
            this.progressBadge.textContent = '0';
            this.progressBadge.classList.remove('complete');
        }
    }

    // 全局成功庆祝动画
    function showSuccessCelebration() {
        const confettiContainer = document.createElement('div');
        confettiContainer.className = 'captcha-success-confetti';
        document.body.appendChild(confettiContainer);
        
        const colors = ['#c9a96e', '#d4b87a', '#28a745', '#ffc107', '#dc3545'];
        
        for (let i = 0; i < 50; i++) {
            const confetti = document.createElement('div');
            confetti.className = 'confetti-piece';
            confetti.style.left = Math.random() * 100 + '%';
            confetti.style.backgroundColor = colors[Math.floor(Math.random() * colors.length)];
            confetti.style.animationDelay = Math.random() * 2 + 's';
            confetti.style.animationDuration = (2 + Math.random() * 2) + 's';
            confettiContainer.appendChild(confetti);
        }
        
        setTimeout(() => {
            confettiContainer.remove();
        }, 5000);
    }

    // 增强的错误提示
    function showEnhancedError(message, container) {
        const errorHint = document.createElement('div');
        errorHint.className = 'captcha-error-hint';
        errorHint.innerHTML = `
            <i class="fas fa-exclamation-circle"></i>
            <div class="captcha-error-hint-text">
                <div class="captcha-error-hint-title">验证失败</div>
                <div class="captcha-error-hint-desc">${message}</div>
            </div>
        `;
        
        if (container) {
            container.appendChild(errorHint);
            
            setTimeout(() => {
                errorHint.remove();
            }, 3000);
        }
        
        return errorHint;
    }

    // 增强的成功提示
    function showEnhancedSuccess(message, container) {
        const successHint = document.createElement('div');
        successHint.className = 'captcha-success-hint';
        successHint.innerHTML = `
            <i class="fas fa-check-circle"></i>
            <div class="captcha-success-hint-text">
                <div class="captcha-success-hint-title">验证成功</div>
                <div class="captcha-success-hint-desc">${message}</div>
            </div>
        `;
        
        if (container) {
            container.appendChild(successHint);
            
            setTimeout(() => {
                successHint.remove();
            }, 3000);
        }
        
        return successHint;
    }

    // 性能监控
    class PerformanceMonitor {
        constructor() {
            this.metrics = {
                pageLoadTime: 0,
                fcp: 0,
                lcp: 0
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
                    }, 100);
                });
            }
        }

        updateMetricsDisplay() {
            const metricsEl = document.getElementById('performanceMetrics');
            if (metricsEl) {
                const loadTimeEl = document.getElementById('pageLoadTime');
                if (loadTimeEl) {
                    loadTimeEl.textContent = this.metrics.pageLoadTime + 'ms';
                    loadTimeEl.className = 'metric-value ' + this.getMetricClass(this.metrics.pageLoadTime, 2000);
                }
            }
        }

        getMetricClass(value, threshold) {
            if (value < threshold * 0.5) return 'good';
            if (value < threshold) return 'warning';
            return 'bad';
        }
    }

    // 导出到全局
    window.CaptchaUIPack = {
        EnhancedSliderCaptcha,
        EnhancedClickCaptcha,
        CDNResourceMonitor,
        PerformanceMonitor,
        showSuccessCelebration,
        showEnhancedError,
        showEnhancedSuccess
    };

    // 初始化
    document.addEventListener('DOMContentLoaded', () => {
        // 监控CDN资源
        new CDNResourceMonitor();
        
        // 初始化性能监控
        new PerformanceMonitor();
        
        // 自动初始化增强的滑块验证
        document.querySelectorAll('[data-captcha-slider]').forEach(el => {
            new EnhancedSliderCaptcha(el);
        });
        
        // 自动初始化增强的点选验证
        document.querySelectorAll('[data-captcha-click]').forEach(el => {
            new EnhancedClickCaptcha(el);
        });
    });

})();
