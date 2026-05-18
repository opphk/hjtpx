document.addEventListener('DOMContentLoaded', function() {
    console.log('用户端已加载');
    
    initPageLoadProgress();
    initPerformanceMonitoring();
    initEnhancedErrorHandling();
    initSmoothTransitions();
    initAccessibilityFeatures();
    initCoreWebVitalsMonitoring();
    initResourcePreloading();

    const navLinks = document.querySelectorAll('nav a');
    navLinks.forEach(link => {
        link.addEventListener('click', function(e) {
            navLinks.forEach(l => l.classList.remove('active'));
            this.classList.add('active');
        });
    });

    const buttons = document.querySelectorAll('.btn');
    buttons.forEach(btn => {
        btn.addEventListener('mouseenter', function() {
            if (!this.classList.contains('no-animation')) {
                this.style.transform = 'scale(1.05)';
            }
        });
        btn.addEventListener('mouseleave', function() {
            if (!this.classList.contains('no-animation')) {
                this.style.transform = 'scale(1)';
            }
        });
        btn.addEventListener('focus', function() {
            this.style.outline = '2px solid #c9a96e';
            this.style.outlineOffset = '2px';
        });
        btn.addEventListener('blur', function() {
            this.style.outline = '';
            this.style.outlineOffset = '';
        });
    });

    injectCaptchaStyles();
});

function initPageLoadProgress() {
    const progressBar = document.createElement('div');
    progressBar.id = 'page-load-progress';
    progressBar.innerHTML = `
        <div class="page-load-progress-bar"></div>
        <div class="page-load-progress-glow"></div>
        <div class="page-load-progress-text" role="status" aria-live="polite">加载中...</div>
    `;
    progressBar.style.cssText = `
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 4px;
        z-index: 99999;
        background: rgba(201, 169, 110, 0.1);
        overflow: hidden;
    `;
    
    const innerBar = progressBar.querySelector('.page-load-progress-bar');
    innerBar.style.cssText = `
        height: 100%;
        background: linear-gradient(90deg, #c9a96e, #d4b87a, #c9a96e);
        background-size: 200% 100%;
        width: 0%;
        transition: width 0.3s cubic-bezier(0.4, 0, 0.2, 1);
        animation: shimmer 2s linear infinite;
    `;
    
    const glow = progressBar.querySelector('.page-load-progress-glow');
    glow.style.cssText = `
        position: absolute;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background: linear-gradient(90deg, transparent, rgba(201, 169, 110, 0.3), transparent);
        animation: pulse-glow 1.5s ease-in-out infinite;
    `;
    
    const text = progressBar.querySelector('.page-load-progress-text');
    text.style.cssText = `
        position: absolute;
        top: 100%;
        left: 50%;
        transform: translateX(-50%);
        font-size: 12px;
        color: #c9a96e;
        white-space: nowrap;
        margin-top: 8px;
        font-weight: 500;
    `;
    
    document.body.appendChild(progressBar);
    
    const style = document.createElement('style');
    style.textContent = `
        @keyframes shimmer {
            0% { background-position: -200% 0; }
            100% { background-position: 200% 0; }
        }
        @keyframes pulse-glow {
            0%, 100% { opacity: 0; transform: translateX(-100%); }
            50% { opacity: 1; }
            100% { transform: translateX(100%); }
        }
    `;
    document.head.appendChild(style);
    
    let progress = 0;
    let lastProgressTime = performance.now();
    
    const updateProgress = () => {
        const now = performance.now();
        const elapsed = now - lastProgressTime;
        
        if (elapsed > 50) {
            progress += Math.random() * 12;
            if (progress > 90) progress = 90;
            innerBar.style.width = progress + '%';
            
            const statusTexts = ['初始化...', '加载资源...', '准备验证...', '即将完成...'];
            const statusIndex = Math.min(Math.floor(progress / 25), statusTexts.length - 1);
            text.textContent = statusTexts[statusIndex];
            
            lastProgressTime = now;
        }
        
        if (progress < 90) {
            requestAnimationFrame(updateProgress);
        }
    };
    
    requestAnimationFrame(updateProgress);
    
    const handleLoadComplete = () => {
        innerBar.style.width = '100%';
        text.textContent = '加载完成';
        
        setTimeout(() => {
            progressBar.style.opacity = '0';
            progressBar.style.transition = 'opacity 0.4s ease';
            
            setTimeout(() => {
                if (progressBar.parentNode) {
                    progressBar.parentNode.removeChild(progressBar);
                }
            }, 400);
        }, 200);
    };
    
    if (document.readyState === 'complete') {
        handleLoadComplete();
    } else {
        window.addEventListener('load', handleLoadComplete);
    }
}

function initPerformanceMonitoring() {
    if (!window.PerformanceObserver) return;
    
    try {
        const observer = new PerformanceObserver((list) => {
            list.getEntries().forEach((entry) => {
                if (entry.entryType === 'navigation') {
                    const loadTime = Math.round(entry.loadEventEnd - entry.startTime);
                    const domContentLoadedTime = Math.round(entry.domContentLoadedEventEnd - entry.startTime);
                    
                    console.log('页面加载时间:', loadTime + 'ms');
                    console.log('DOM加载时间:', domContentLoadedTime + 'ms');
                    
                    updatePerformanceMetrics({
                        pageLoad: loadTime,
                        domLoad: domContentLoadedTime
                    });
                    
                    if (loadTime > 500) {
                        console.warn('页面加载时间超过500ms，建议优化');
                    }
                }
                if (entry.entryType === 'resource') {
                    if (entry.duration > 1000) {
                        console.warn('资源加载较慢:', entry.name, Math.round(entry.duration) + 'ms');
                    }
                }
            });
        });
        observer.observe({ entryTypes: ['navigation', 'resource'] });
    } catch (e) {
        console.log('性能监控不可用');
    }
}

function initCoreWebVitalsMonitoring() {
    if (!window.PerformanceObserver) return;
    
    try {
        const observer = new PerformanceObserver((entryList) => {
            for (const entry of entryList.getEntries()) {
                const metricName = entry.name;
                const value = entry.value;
                
                console.log(`${metricName}: ${value.toFixed(2)}ms`);
                
                if (metricName === 'LCP') {
                    if (value > 2500) {
                        console.warn('LCP 较慢:', value.toFixed(2) + 'ms');
                    }
                } else if (metricName === 'FID' || metricName === 'INP') {
                    if (value > 200) {
                        console.warn('交互响应较慢:', value.toFixed(2) + 'ms');
                    }
                } else if (metricName === 'CLS') {
                    if (value > 0.25) {
                        console.warn('布局偏移较大:', value.toFixed(2));
                    }
                }
            }
        });
        
        observer.observe({ type: 'measure', buffered: true });
        
        const paintObserver = new PerformanceObserver((entryList) => {
            for (const entry of entryList.getEntries()) {
                console.log(`${entry.name}: ${entry.startTime.toFixed(2)}ms`);
            }
        });
        
        paintObserver.observe({ type: 'paint', buffered: true });
    } catch (e) {
        console.log('Core Web Vitals 监控不可用');
    }
}

function initResourcePreloading() {
    const criticalStylesheets = [
        'https://cdn.bootcdn.net/ajax/libs/twitter-bootstrap/5.3.8/css/bootstrap.min.css',
        'https://cdn.bootcdn.net/ajax/libs/font-awesome/6.5.1/css/all.min.css'
    ];
    
    criticalStylesheets.forEach((url, index) => {
        const link = document.createElement('link');
        link.rel = 'preload';
        link.href = url;
        link.as = 'style';
        link.crossOrigin = 'anonymous';
        link.setAttribute('data-preload-index', index);
        document.head.appendChild(link);
        
        link.onload = () => {
            link.rel = 'stylesheet';
        };
    });
    
    const fontLinks = document.querySelectorAll('link[rel="preconnect"]');
    fontLinks.forEach(link => {
        link.setAttribute('crossorigin', 'anonymous');
    });
}

function updatePerformanceMetrics(data) {
    const pageLoadEl = document.getElementById('pageLoadTime');
    const fpsEl = document.getElementById('fpsMetric');
    
    if (pageLoadEl) {
        pageLoadEl.textContent = data.pageLoad + 'ms';
        pageLoadEl.className = 'metric-value ' + (data.pageLoad < 500 ? 'good' : data.pageLoad < 1000 ? 'warning' : 'bad');
    }
    
    if (fpsEl) {
        const fps = Math.round(1000 / (data.domLoad / 60));
        fpsEl.textContent = fps + ' FPS';
        fpsEl.className = 'metric-value ' + (fps >= 55 ? 'good' : fps >= 30 ? 'warning' : 'bad');
    }
}

function initEnhancedErrorHandling() {
    const originalError = window.onerror;
    window.onerror = function(message, source, lineno, colno, error) {
        const errorInfo = {
            message: message,
            source: source,
            line: lineno,
            col: colno,
            stack: error ? error.stack : '',
            timestamp: new Date().toISOString(),
            userAgent: navigator.userAgent
        };
        console.error('页面错误:', errorInfo);
        
        const errorCategory = categorizeError(message);
        showEnhancedErrorToast(errorCategory.message, errorCategory.details);
        
        if (originalError) {
            return originalError.apply(this, arguments);
        }
        return false;
    };

    window.addEventListener('unhandledrejection', function(event) {
        console.error('未处理的Promise拒绝:', event.reason);
        const reason = event.reason;
        let message = '网络请求失败，请检查网络连接后重试';
        let details = '';
        
        if (reason && typeof reason === 'object') {
            if (reason.status === 401) {
                message = '会话已过期，请重新登录';
            } else if (reason.status === 403) {
                message = '权限不足，无法访问该资源';
            } else if (reason.status === 429) {
                message = '请求过于频繁，请稍后重试';
            } else if (reason.status === 500) {
                message = '服务器内部错误，请稍后重试';
            } else if (reason.message) {
                details = reason.message;
            }
        }
        
        showEnhancedErrorToast(message, details);
    });

    window.addEventListener('online', function() {
        showEnhancedToast('网络已恢复', 'success');
    });

    window.addEventListener('offline', function() {
        showEnhancedErrorToast('网络连接已断开，请检查网络设置');
    });

    setupFormValidationFeedback();
    setupButtonClickFeedback();
}

function categorizeError(message) {
    if (message.includes('NetworkError') || message.includes('Failed to fetch')) {
        return {
            message: '网络请求失败，请检查网络连接',
            details: '可能是网络不稳定或服务器暂时不可用'
        };
    }
    if (message.includes('SyntaxError')) {
        return {
            message: '脚本解析错误',
            details: '页面脚本加载异常，请刷新重试'
        };
    }
    if (message.includes('TypeError')) {
        return {
            message: '运行时错误',
            details: '页面功能暂时不可用，请刷新重试'
        };
    }
    if (message.includes('ReferenceError')) {
        return {
            message: '资源引用错误',
            details: '页面资源加载异常，请刷新重试'
        };
    }
    return {
        message: '页面发生错误，请刷新重试',
        details: message
    };
}

function setupFormValidationFeedback() {
    const forms = document.querySelectorAll('form');
    forms.forEach(form => {
        form.addEventListener('submit', function(e) {
            const requiredFields = form.querySelectorAll('[required]');
            let hasError = false;
            
            requiredFields.forEach(field => {
                if (!field.value || field.value.trim() === '') {
                    hasError = true;
                    field.classList.add('validation-error');
                    field.setAttribute('aria-invalid', 'true');
                    field.setAttribute('aria-describedby', field.id + '-error');
                    
                    const errorId = field.id + '-error';
                    let errorEl = document.getElementById(errorId);
                    if (!errorEl) {
                        errorEl = document.createElement('span');
                        errorEl.id = errorId;
                        errorEl.className = 'validation-error-message';
                        errorEl.setAttribute('role', 'alert');
                        errorEl.setAttribute('aria-live', 'assertive');
                        errorEl.textContent = getFieldLabel(field) + '是必填项';
                        field.parentNode.appendChild(errorEl);
                    }
                } else {
                    field.classList.remove('validation-error');
                    field.setAttribute('aria-invalid', 'false');
                    const errorId = field.id + '-error';
                    const errorEl = document.getElementById(errorId);
                    if (errorEl) errorEl.remove();
                }
            });
            
            if (hasError) {
                e.preventDefault();
                showEnhancedToast('请填写必填字段', 'warning');
            }
        });
        
        form.addEventListener('input', function(e) {
            const target = e.target;
            if (target.hasAttribute('required')) {
                if (target.value && target.value.trim() !== '') {
                    target.classList.remove('validation-error');
                    target.setAttribute('aria-invalid', 'false');
                    const errorId = target.id + '-error';
                    const errorEl = document.getElementById(errorId);
                    if (errorEl) errorEl.remove();
                }
            }
        });
    });
}

function getFieldLabel(field) {
    const label = document.querySelector(`label[for="${field.id}"]`);
    if (label) return label.textContent;
    const placeholder = field.getAttribute('placeholder');
    if (placeholder) return placeholder;
    return '此字段';
}

function setupButtonClickFeedback() {
    const buttons = document.querySelectorAll('button, .btn, [role="button"]');
    buttons.forEach(btn => {
        btn.addEventListener('click', function(e) {
            if (this.classList.contains('no-feedback')) return;
            
            const originalText = this.innerHTML;
            const isSubmit = this.type === 'submit' || this.classList.contains('btn-submit');
            
            if (isSubmit && !this.classList.contains('is-loading')) {
                this.classList.add('is-loading');
                this.disabled = true;
                this.innerHTML = '<i class="fas fa-spinner fa-spin"></i> 处理中...';
                this.setAttribute('aria-busy', 'true');
                
                setTimeout(() => {
                    this.classList.remove('is-loading');
                    this.disabled = false;
                    this.innerHTML = originalText;
                    this.setAttribute('aria-busy', 'false');
                }, 3000);
            }
        });
    });
}

function showEnhancedErrorToast(message, details) {
    showEnhancedToast(message, 'error', details);
}

function showEnhancedToast(message, type = 'info', details, options = {}) {
    const existing = document.querySelector('.enhanced-toast');
    if (existing) existing.remove();
    
    const config = {
        duration: type === 'error' ? 8000 : 4000,
        showRetry: type === 'error',
        showUndo: false,
        onRetry: () => location.reload(),
        onUndo: null,
        ...options
    };
    
    const toast = document.createElement('div');
    toast.className = `enhanced-toast enhanced-toast-${type}`;
    toast.setAttribute('role', 'alert');
    toast.setAttribute('aria-live', type === 'error' ? 'assertive' : 'polite');
    toast.setAttribute('aria-atomic', 'true');
    toast.setAttribute('data-toast-type', type);
    
    const icons = {
        success: 'fa-check-circle',
        error: 'fa-exclamation-circle',
        warning: 'fa-exclamation-triangle',
        info: 'fa-info-circle',
        confirm: 'fa-question-circle'
    };
    
    const colors = {
        success: 'linear-gradient(135deg, #28a745, #20c997)',
        error: 'linear-gradient(135deg, #dc3545, #fd7e14)',
        warning: 'linear-gradient(135deg, #ffc107, #fd7e14)',
        info: 'linear-gradient(135deg, #17a2b8, #007bff)',
        confirm: 'linear-gradient(135deg, #0d6efd, #6610f2)'
    };
    
    const actionButtons = [];
    if (config.showRetry) {
        actionButtons.push(`<button class="toast-action toast-retry" aria-label="重试"><i class="fas fa-redo me-1"></i>重试</button>`);
    }
    if (config.showUndo && config.onUndo) {
        actionButtons.push(`<button class="toast-action toast-undo" aria-label="撤销"><i class="fas fa-undo me-1"></i>撤销</button>`);
    }
    if (options.customActions) {
        options.customActions.forEach(action => {
            actionButtons.push(`<button class="toast-action toast-custom" aria-label="${action.label}" data-action="${action.name}"><i class="fas ${action.icon} me-1"></i>${action.label}</button>`);
        });
    }
    
    toast.innerHTML = `
        <div class="toast-icon">
            <i class="fas ${icons[type] || icons.info}"></i>
        </div>
        <div class="toast-content">
            <p class="toast-message">${message}</p>
            ${details ? `<p class="toast-details">${details}</p>` : ''}
            ${actionButtons.length > 0 ? `<div class="toast-actions">${actionButtons.join('')}</div>` : ''}
        </div>
        <button class="toast-dismiss" aria-label="关闭通知">
            <i class="fas fa-times"></i>
        </button>
    `;
    
    toast.style.cssText = `
        position: fixed;
        bottom: ${options.position?.bottom || '20px'};
        right: ${options.position?.right || '20px'};
        left: ${options.position?.left || 'auto'};
        background: ${colors[type] || colors.info};
        color: white;
        padding: 14px 16px;
        border-radius: 12px;
        display: flex;
        align-items: flex-start;
        gap: 12px;
        z-index: 99999;
        box-shadow: 0 8px 24px rgba(0,0,0,0.25);
        max-width: 380px;
        animation: toastSlideIn 0.4s cubic-bezier(0.175, 0.885, 0.32, 1.275);
        backdrop-filter: blur(8px);
        border: 1px solid rgba(255,255,255,0.1);
    `;
    
    injectToastStyles();
    
    const dismissBtn = toast.querySelector('.toast-dismiss');
    dismissBtn.addEventListener('click', () => dismissToast(toast));
    
    const retryBtn = toast.querySelector('.toast-retry');
    if (retryBtn) {
        retryBtn.addEventListener('click', () => {
            dismissToast(toast);
            config.onRetry();
        });
    }
    
    const undoBtn = toast.querySelector('.toast-undo');
    if (undoBtn && config.onUndo) {
        undoBtn.addEventListener('click', () => {
            dismissToast(toast);
            config.onUndo();
        });
    }
    
    const customButtons = toast.querySelectorAll('.toast-custom');
    customButtons.forEach(btn => {
        btn.addEventListener('click', () => {
            const actionName = btn.dataset.action;
            dismissToast(toast);
            options.customActions.find(a => a.name === actionName)?.onClick?.();
        });
    });
    
    document.body.appendChild(toast);
    
    if (config.duration > 0) {
        setTimeout(() => {
            if (toast.parentElement) {
                dismissToast(toast);
            }
        }, config.duration);
    }
    
    return toast;
}

function injectToastStyles() {
    if (document.getElementById('enhanced-toast-styles')) {
        return;
    }
    
    const style = document.createElement('style');
    style.id = 'enhanced-toast-styles';
    style.textContent = `
        @keyframes toastSlideIn {
            from {
                transform: translateX(120%);
                opacity: 0;
            }
            to {
                transform: translateX(0);
                opacity: 1;
            }
        }
        @keyframes toastSlideOut {
            from {
                transform: translateX(0);
                opacity: 1;
            }
            to {
                transform: translateX(120%);
                opacity: 0;
            }
        }
        @keyframes toastFadeIn {
            from { opacity: 0; transform: translateY(20px); }
            to { opacity: 1; transform: translateY(0); }
        }
        .enhanced-toast {
            animation: toastFadeIn 0.3s ease;
        }
        .enhanced-toast .toast-icon {
            flex-shrink: 0;
            width: 36px;
            height: 36px;
            background: rgba(255,255,255,0.15);
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 18px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.15);
        }
        .enhanced-toast .toast-content {
            flex: 1;
            min-width: 0;
        }
        .enhanced-toast .toast-message {
            margin: 0;
            font-weight: 600;
            font-size: 14px;
            line-height: 1.4;
        }
        .enhanced-toast .toast-details {
            margin: 6px 0 0;
            font-size: 12px;
            opacity: 0.85;
            font-family: -apple-system, BlinkMacSystemFont, sans-serif;
            line-height: 1.4;
        }
        .enhanced-toast .toast-actions {
            display: flex;
            gap: 8px;
            margin-top: 10px;
        }
        .enhanced-toast .toast-action {
            flex-shrink: 0;
            background: rgba(255,255,255,0.2);
            border: 1px solid rgba(255,255,255,0.3);
            border-radius: 6px;
            padding: 6px 12px;
            color: white;
            font-size: 12px;
            font-weight: 500;
            cursor: pointer;
            transition: all 0.2s ease;
            display: inline-flex;
            align-items: center;
            gap: 4px;
        }
        .enhanced-toast .toast-action:hover {
            background: rgba(255,255,255,0.3);
            transform: translateY(-1px);
            box-shadow: 0 2px 8px rgba(0,0,0,0.2);
        }
        .enhanced-toast .toast-action:focus-visible {
            outline: 2px solid rgba(255,255,255,0.8);
            outline-offset: 2px;
        }
        .enhanced-toast .toast-dismiss {
            flex-shrink: 0;
            background: rgba(255,255,255,0.15);
            border: none;
            border-radius: 50%;
            width: 28px;
            height: 28px;
            display: flex;
            align-items: center;
            justify-content: center;
            color: white;
            cursor: pointer;
            transition: all 0.2s ease;
        }
        .enhanced-toast .toast-dismiss:hover {
            background: rgba(255,255,255,0.25);
            transform: scale(1.1);
        }
        .enhanced-toast .toast-dismiss:focus-visible {
            outline: 2px solid rgba(255,255,255,0.8);
            outline-offset: 2px;
        }
        .enhanced-toast.error {
            border-left: 4px solid #dc3545;
        }
        .enhanced-toast.success {
            border-left: 4px solid #28a745;
        }
        .enhanced-toast.warning {
            border-left: 4px solid #ffc107;
        }
        @media (max-width: 576px) {
            .enhanced-toast {
                left: 12px !important;
                right: 12px !important;
                bottom: 12px !important;
                max-width: none;
                border-radius: 10px;
            }
            .enhanced-toast .toast-actions {
                flex-wrap: wrap;
            }
            .enhanced-toast .toast-action {
                flex: 1;
                min-width: calc(50% - 4px);
                justify-content: center;
            }
        }
        @media (prefers-reduced-motion: reduce) {
            .enhanced-toast {
                animation: none;
                opacity: 1;
            }
            .enhanced-toast .toast-action:hover {
                transform: none;
            }
        }
        .validation-error {
            border-color: #dc3545 !important;
            box-shadow: 0 0 0 2px rgba(220, 53, 69, 0.2) !important;
        }
        .validation-error-message {
            display: block;
            color: #dc3545;
            font-size: 12px;
            margin-top: 4px;
            padding: 4px 8px;
            background: rgba(220, 53, 69, 0.05);
            border-radius: 4px;
        }
        button.is-loading {
            pointer-events: none;
            opacity: 0.7;
        }
    `;
    document.head.appendChild(style);
}

function dismissToast(toast) {
    toast.style.animation = 'toastSlideOut 0.3s ease forwards';
    setTimeout(() => {
        if (toast.parentElement) {
            toast.remove();
        }
    }, 300);
}

function initSmoothTransitions() {
    const elements = document.querySelectorAll('.card, .btn, .nav-link, .demo-card');
    elements.forEach(el => {
        el.style.transition = 'transform 0.25s cubic-bezier(0.4, 0, 0.2, 1), box-shadow 0.25s cubic-bezier(0.4, 0, 0.2, 1), opacity 0.25s cubic-bezier(0.4, 0, 0.2, 1)';
    });
    
    if ('IntersectionObserver' in window) {
        const observer = new IntersectionObserver((entries) => {
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    entry.target.classList.add('animate-in');
                    observer.unobserve(entry.target);
                }
            });
        }, { 
            threshold: 0.1,
            rootMargin: '0px 0px -50px 0px'
        });
        
        document.querySelectorAll('.animate-on-scroll').forEach(el => {
            el.classList.add('animate-prepare');
            observer.observe(el);
        });
    }
    
    const animStyle = document.createElement('style');
    animStyle.textContent = `
        .animate-prepare { 
            opacity: 0; 
            transform: translateY(30px); 
            transition: opacity 0s, transform 0s;
        }
        .animate-in { 
            animation: animateIn 0.6s cubic-bezier(0.4, 0, 0.2, 1) forwards; 
        }
        @keyframes animateIn {
            to { 
                opacity: 1; 
                transform: translateY(0); 
            }
        }
        .animate-in-delay-1 { animation-delay: 0.1s; }
        .animate-in-delay-2 { animation-delay: 0.2s; }
        .animate-in-delay-3 { animation-delay: 0.3s; }
        .animate-in-delay-4 { animation-delay: 0.4s; }
        
        .slide-in-left {
            animation: slideInLeft 0.5s cubic-bezier(0.4, 0, 0.2, 1) forwards;
        }
        @keyframes slideInLeft {
            from { opacity: 0; transform: translateX(-30px); }
            to { opacity: 1; transform: translateX(0); }
        }
        
        .fade-in-up {
            animation: fadeInUp 0.5s cubic-bezier(0.4, 0, 0.2, 1) forwards;
        }
        @keyframes fadeInUp {
            from { opacity: 0; transform: translateY(20px); }
            to { opacity: 1; transform: translateY(0); }
        }
        
        .scale-in {
            animation: scaleIn 0.4s cubic-bezier(0.4, 0, 0.2, 1) forwards;
        }
        @keyframes scaleIn {
            from { opacity: 0; transform: scale(0.95); }
            to { opacity: 1; transform: scale(1); }
        }
    `;
    document.head.appendChild(animStyle);
}

function initAccessibilityFeatures() {
    initSkipLinks();
    initKeyboardNavigation();
    initReducedMotionSupport();
    initFocusManagement();
}

function initSkipLinks() {
    const skipLinks = document.querySelectorAll('.skip-link');
    skipLinks.forEach(link => {
        link.addEventListener('click', (e) => {
            e.preventDefault();
            const targetId = link.getAttribute('href');
            const target = document.querySelector(targetId);
            if (target) {
                target.setAttribute('tabindex', '-1');
                target.focus();
                target.addEventListener('blur', function removeTabindex() {
                    target.removeAttribute('tabindex');
                    target.removeEventListener('blur', removeTabindex);
                });
            }
        });
    });
}

function initKeyboardNavigation() {
    const focusableElements = document.querySelectorAll(
        'a, button, input, select, textarea, [tabindex]:not([tabindex="-1"]), [role="button"]'
    );
    
    focusableElements.forEach(el => {
        if (!el.hasAttribute('tabindex')) {
            el.setAttribute('tabindex', '0');
        }
    });
    
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            const modals = document.querySelectorAll('.modal.show, .captcha-loading-overlay:not([hidden])');
            modals.forEach(modal => {
                if (modal.close) {
                    modal.close();
                } else {
                    modal.setAttribute('hidden', '');
                }
            });
            
            const activeElement = document.activeElement;
            if (activeElement && activeElement.blur) {
                activeElement.blur();
            }
        }
    });
}

function initReducedMotionSupport() {
    const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)');
    
    const applyReducedMotion = (reduce) => {
        if (reduce) {
            document.documentElement.classList.add('reduce-motion');
        } else {
            document.documentElement.classList.remove('reduce-motion');
        }
    };
    
    applyReducedMotion(prefersReducedMotion.matches);
    
    prefersReducedMotion.addEventListener('change', (e) => {
        applyReducedMotion(e.matches);
    });
    
    const reducedMotionStyle = document.createElement('style');
    reducedMotionStyle.textContent = `
        .reduce-motion *,
        .reduce-motion *::before,
        .reduce-motion *::after {
            animation-duration: 0.01ms !important;
            animation-iteration-count: 1 !important;
            transition-duration: 0.01ms !important;
            scroll-behavior: auto !important;
        }
        .reduce-motion .animate-prepare {
            opacity: 1;
            transform: none;
        }
        .reduce-motion .animate-in {
            animation: none;
            opacity: 1;
            transform: none;
        }
    `;
    document.head.appendChild(reducedMotionStyle);
}

function initFocusManagement() {
    document.addEventListener('focus', (e) => {
        const target = e.target;
        if (target.classList.contains('btn') || target.tagName === 'BUTTON') {
            target.style.outline = '2px solid #c9a96e';
            target.style.outlineOffset = '2px';
        }
    }, true);
    
    document.addEventListener('blur', (e) => {
        const target = e.target;
        if (target.classList.contains('btn') || target.tagName === 'BUTTON') {
            target.style.outline = '';
            target.style.outlineOffset = '';
        }
    }, true);
}

function injectCaptchaStyles() {
    if (document.getElementById('captcha-dynamic-styles')) {
        return;
    }

    const styleSheet = document.createElement('style');
    styleSheet.id = 'captcha-dynamic-styles';
    styleSheet.textContent = `
        .captcha-container {
            background: #fff;
            border-radius: 12px;
            box-shadow: 0 4px 24px rgba(0,0,0,0.08);
            overflow: hidden;
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            border: 1px solid rgba(201, 169, 110, 0.1);
        }
        .captcha-header {
            background: linear-gradient(135deg, #c9a96e 0%, #d4b87a 100%);
            color: white;
            padding: 20px;
            text-align: center;
        }
        .captcha-header h3 {
            margin: 0 0 5px 0;
            font-size: 20px;
            font-weight: 600;
        }
        .captcha-header p {
            margin: 0;
            font-size: 14px;
            opacity: 0.9;
        }
        .captcha-body {
            padding: 20px;
        }
        .captcha-tabs {
            display: flex;
            gap: 8px;
            margin-bottom: 15px;
            border-bottom: 2px solid #f0f0f0;
            padding-bottom: 10px;
            overflow-x: auto;
            scrollbar-width: none;
        }
        .captcha-tabs::-webkit-scrollbar {
            display: none;
        }
        .captcha-tab {
            flex: 0 0 auto;
            padding: 10px 16px;
            border: none;
            background: transparent;
            color: #666;
            font-size: 14px;
            font-weight: 500;
            cursor: pointer;
            border-radius: 6px;
            transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 8px;
            white-space: nowrap;
            min-width: 80px;
        }
        .captcha-tab .tab-icon {
            font-size: 14px;
        }
        .captcha-tab:hover {
            background: rgba(201, 169, 110, 0.1);
        }
        .captcha-tab.active {
            background: linear-gradient(135deg, #c9a96e 0%, #d4b87a 100%);
            color: white;
            box-shadow: 0 2px 8px rgba(201, 169, 110, 0.3);
        }
        .captcha-tab:focus-visible {
            outline: 2px solid #c9a96e;
            outline-offset: 2px;
        }
        .captcha-content {
            display: none;
        }
        .captcha-content.active {
            display: block;
            animation: fadeIn 0.3s ease;
        }
        @keyframes fadeIn {
            from { opacity: 0; }
            to { opacity: 1; }
        }
        .captcha-loading-overlay {
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: rgba(255,255,255,0.98);
            display: flex;
            align-items: center;
            justify-content: center;
            z-index: 20;
            backdrop-filter: blur(8px);
            border-radius: 8px;
        }
        .captcha-loading-container {
            text-align: center;
            padding: 20px;
        }
        .loading-animation-pulse {
            margin-bottom: 15px;
        }
        .loading-dots {
            display: flex;
            gap: 8px;
            justify-content: center;
        }
        .loading-dots span {
            width: 10px;
            height: 10px;
            background: #c9a96e;
            border-radius: 50%;
            animation: loading-bounce 1.4s infinite ease-in-out both;
        }
        .loading-dots span:nth-child(1) { animation-delay: -0.32s; }
        .loading-dots span:nth-child(2) { animation-delay: -0.16s; }
        .loading-dots span:nth-child(3) { animation-delay: 0s; }
        .loading-dots span:nth-child(4) { animation-delay: 0.16s; }
        .loading-dots span:nth-child(5) { animation-delay: 0.32s; }
        @keyframes loading-bounce {
            0%, 80%, 100% { 
                transform: scale(0);
                opacity: 0.4;
            }
            40% { 
                transform: scale(1);
                opacity: 1;
            }
        }
        .loading-progress-bar {
            width: 200px;
            height: 4px;
            background: #f0f0f0;
            border-radius: 2px;
            overflow: hidden;
            margin: 15px auto;
            box-shadow: inset 0 1px 2px rgba(0,0,0,0.1);
        }
        .loading-progress-fill {
            height: 100%;
            background: linear-gradient(90deg, #c9a96e 0%, #d4b87a 100%);
            width: 0%;
            transition: width 0.3s cubic-bezier(0.4, 0, 0.2, 1);
            border-radius: 2px;
        }
        .loading-message {
            color: #666;
            font-size: 14px;
            font-weight: 500;
        }
        .captcha-image-wrapper {
            position: relative;
            width: 100%;
            max-width: 360px;
            margin: 0 auto 15px;
            border-radius: 8px;
            overflow: hidden;
            background: #f8f9fa;
            border: 1px solid #e9ecef;
        }
        .captcha-canvas {
            display: block;
            width: 100%;
            height: auto;
        }
        .captcha-background-layer {
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            pointer-events: none;
        }
        .captcha-puzzle {
            position: absolute;
            top: 0;
            left: 0;
            width: 50px;
            height: 50px;
            pointer-events: none;
            z-index: 2;
            transition: transform 0.1s ease;
        }
        .puzzle-piece-square {
            width: 50px;
            height: 50px;
            background: rgba(255,255,255,0.4);
            border: 2px solid rgba(255,255,255,0.9);
            box-shadow: 0 2px 8px rgba(0,0,0,0.25);
            backdrop-filter: blur(4px);
        }
        .puzzle-piece-circle {
            width: 50px;
            height: 50px;
            background: rgba(255,255,255,0.4);
            border: 2px solid rgba(255,255,255,0.9);
            border-radius: 50%;
            box-shadow: 0 2px 8px rgba(0,0,0,0.25);
            backdrop-filter: blur(4px);
        }
        .puzzle-piece-triangle {
            width: 0;
            height: 0;
            border-left: 25px solid transparent;
            border-right: 25px solid transparent;
            border-bottom: 43px solid rgba(255,255,255,0.5);
            background: transparent;
            filter: drop-shadow(0 2px 8px rgba(0,0,0,0.25));
        }
        .puzzle-piece-diamond {
            width: 50px;
            height: 50px;
            background: rgba(255,255,255,0.4);
            border: 2px solid rgba(255,255,255,0.9);
            transform: rotate(45deg);
            margin: 5px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.25);
            backdrop-filter: blur(4px);
        }
        .puzzle-piece-hexagon {
            width: 50px;
            height: 28.87px;
            background: rgba(255,255,255,0.4);
            border: 2px solid rgba(255,255,255,0.9);
            position: relative;
            margin-top: 10px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.25);
            backdrop-filter: blur(4px);
        }
        .captcha-refresh {
            position: absolute;
            top: 8px;
            right: 8px;
            width: 34px;
            height: 34px;
            border: none;
            background: rgba(255,255,255,0.95);
            border-radius: 50%;
            cursor: pointer;
            font-size: 16px;
            display: flex;
            align-items: center;
            justify-content: center;
            transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
            z-index: 10;
            color: #666;
            box-shadow: 0 2px 8px rgba(0,0,0,0.15);
        }
        .captcha-refresh:hover {
            background: white;
            transform: rotate(180deg);
            box-shadow: 0 4px 12px rgba(0,0,0,0.2);
        }
        .captcha-refresh:focus-visible {
            outline: 2px solid #c9a96e;
            outline-offset: 2px;
        }
        .captcha-slider-container {
            position: relative;
            width: 100%;
            max-width: 360px;
            height: 46px;
            margin: 0 auto;
            background: #f5f5f5;
            border-radius: 23px;
            overflow: hidden;
            border: 1px solid #e9ecef;
            transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
            touch-action: pan-y pinch-zoom;
        }
        .captcha-slider-container.is-dragging {
            border-color: #c9a96e;
            box-shadow: 0 0 0 3px rgba(201, 169, 110, 0.15);
        }
        .captcha-slider-container.error-flash {
            animation: error-flash-animation 0.5s ease;
        }
        @keyframes error-flash-animation {
            0%, 100% { background: #f5f5f5; }
            25%, 75% { background: #fff5f5; }
            50% { background: #ffe0e0; }
        }
        .captcha-slider-track {
            position: absolute;
            left: 2px;
            top: 2px;
            height: 42px;
            width: 0;
            background: linear-gradient(90deg, #c9a96e 0%, #d4b87a 100%);
            border-radius: 21px;
            transition: width 0.08s linear;
        }
        .captcha-slider-text {
            position: absolute;
            width: 100%;
            text-align: center;
            line-height: 46px;
            font-size: 14px;
            color: #666;
            pointer-events: none;
            z-index: 1;
            font-weight: 500;
        }
        .captcha-slider-hint {
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 6px;
            margin-top: 10px;
            font-size: 12px;
            color: #999;
        }
        .captcha-slider-hint .hint-icon {
            color: #c9a96e;
        }
        .captcha-slider-button {
            position: absolute;
            left: 2px;
            top: 2px;
            width: 42px;
            height: 42px;
            background: white;
            border-radius: 50%;
            cursor: grab;
            display: flex;
            align-items: center;
            justify-content: center;
            box-shadow: 0 2px 10px rgba(0,0,0,0.15);
            transition: left 0.08s linear, transform 0.2s cubic-bezier(0.4, 0, 0.2, 1), background 0.2s ease, box-shadow 0.2s ease;
            z-index: 2;
            color: #c9a96e;
            touch-action: none;
            user-select: none;
            -webkit-user-select: none;
        }
        .captcha-slider-button svg {
            width: 18px;
            height: 18px;
        }
        .captcha-slider-button:hover:not(.dragging) {
            transform: scale(1.08);
            box-shadow: 0 4px 14px rgba(0,0,0,0.2);
        }
        .captcha-slider-button.dragging {
            cursor: grabbing;
            transform: scale(1.12);
            box-shadow: 0 6px 20px rgba(201, 169, 110, 0.4);
        }
        .captcha-slider-button.verifying {
            animation: pulse-verifying 1.2s infinite;
        }
        @keyframes pulse-verifying {
            0%, 100% { box-shadow: 0 2px 10px rgba(201, 169, 110, 0.3); }
            50% { box-shadow: 0 4px 20px rgba(201, 169, 110, 0.6); }
        }
        .captcha-slider-button.success {
            background: #28a745;
            color: white;
            animation: success-bounce 0.6s cubic-bezier(0.175, 0.885, 0.32, 1.275);
        }
        @keyframes success-bounce {
            0% { transform: scale(1); }
            50% { transform: scale(1.2); }
            100% { transform: scale(1); }
        }
        .captcha-slider-button.error {
            background: #dc3545;
            color: white;
            animation: shake 0.5s ease-in-out;
        }
        @keyframes shake {
            0%, 100% { transform: translateX(0); }
            20% { transform: translateX(-8px); }
            40% { transform: translateX(8px); }
            60% { transform: translateX(-6px); }
            80% { transform: translateX(6px); }
        }
        .captcha-click-hint {
            text-align: center;
            padding: 12px;
            background: #f8f9fa;
            border-radius: 8px;
            margin-bottom: 12px;
            font-size: 14px;
            color: #333;
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 10px;
            border: 1px solid #e9ecef;
        }
        .captcha-click-hint .hint-icon {
            color: #c9a96e;
            font-size: 16px;
        }
        .captcha-click-grid {
            position: relative;
            display: inline-block;
            border-radius: 8px;
            overflow: hidden;
        }
        .captcha-click-image {
            display: block;
            width: 100%;
            max-width: 360px;
            border-radius: 8px;
        }
        .captcha-click-marker {
            position: absolute;
            width: 30px;
            height: 30px;
            background: linear-gradient(135deg, #c9a96e 0%, #d4b87a 100%);
            border: 2px solid white;
            border-radius: 50%;
            color: white;
            font-size: 14px;
            font-weight: 600;
            display: flex;
            align-items: center;
            justify-content: center;
            transform: translate(-50%, -50%);
            cursor: pointer;
            box-shadow: 0 3px 12px rgba(201, 169, 110, 0.4);
            animation: marker-pop 0.35s cubic-bezier(0.175, 0.885, 0.32, 1.275);
            transition: transform 0.2s ease, background 0.2s ease;
            z-index: 10;
        }
        .captcha-click-marker:hover {
            transform: translate(-50%, -50%) scale(1.15);
            background: linear-gradient(135deg, #b8954f 0%, #c9a96e 100%);
        }
        .captcha-click-marker.success-marker {
            background: #28a745;
            animation: marker-success 0.5s ease;
        }
        @keyframes marker-success {
            0% { transform: translate(-50%, -50%) scale(1); }
            50% { transform: translate(-50%, -50%) scale(1.3); }
            100% { transform: translate(-50%, -50%) scale(1); }
        }
        @keyframes marker-pop {
            0% { transform: translate(-50%, -50%) scale(0); opacity: 0; }
            50% { transform: translate(-50%, -50%) scale(1.2); }
            100% { transform: translate(-50%, -50%) scale(1); opacity: 1; }
        }
        .captcha-click-progress {
            text-align: center;
            margin: 12px 0;
            font-size: 14px;
            color: #666;
        }
        .count-badge {
            display: inline-block;
            min-width: 28px;
            padding: 3px 10px;
            background: #f0f0f0;
            border-radius: 14px;
            font-weight: 600;
            transition: all 0.25s ease;
        }
        .count-badge.partial {
            background: rgba(201, 169, 110, 0.15);
            color: #c9a96e;
        }
        .count-badge.complete {
            background: rgba(40, 167, 69, 0.15);
            color: #28a745;
        }
        .captcha-actions {
            display: flex;
            gap: 12px;
            justify-content: center;
            margin-top: 15px;
        }
        .captcha-btn {
            padding: 11px 32px;
            border: none;
            border-radius: 8px;
            font-size: 14px;
            font-weight: 500;
            cursor: pointer;
            transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
            display: inline-flex;
            align-items: center;
            gap: 8px;
            min-height: 44px;
        }
        .captcha-btn i {
            font-size: 14px;
        }
        .captcha-btn-primary {
            background: linear-gradient(135deg, #c9a96e 0%, #d4b87a 100%);
            color: white;
            box-shadow: 0 2px 8px rgba(201, 169, 110, 0.3);
        }
        .captcha-btn-primary:hover {
            opacity: 0.95;
            transform: translateY(-2px);
            box-shadow: 0 4px 14px rgba(201, 169, 110, 0.45);
        }
        .captcha-btn-primary:focus-visible {
            outline: 2px solid #c9a96e;
            outline-offset: 2px;
        }
        .captcha-btn-secondary {
            background: #f5f5f5;
            color: #666;
            border: 1px solid #e9ecef;
        }
        .captcha-btn-secondary:hover {
            background: #e9ecef;
            border-color: #dee2e6;
        }
        .captcha-btn-secondary:focus-visible {
            outline: 2px solid #c9a96e;
            outline-offset: 2px;
        }
        .captcha-result {
            text-align: center;
            padding: 14px;
            margin-top: 15px;
            border-radius: 8px;
            font-size: 14px;
            font-weight: 500;
            display: none;
            border: 1px solid;
        }
        .captcha-result.show {
            display: block;
            animation: resultFadeIn 0.35s cubic-bezier(0.4, 0, 0.2, 1);
        }
        @keyframes resultFadeIn {
            from { opacity: 0; transform: translateY(-10px); }
            to { opacity: 1; transform: translateY(0); }
        }
        .captcha-result.success {
            background: rgba(40, 167, 69, 0.08);
            color: #28a745;
            border-color: rgba(40, 167, 69, 0.2);
        }
        .captcha-result.error {
            background: rgba(220, 53, 69, 0.08);
            color: #dc3545;
            border-color: rgba(220, 53, 69, 0.2);
        }
        .captcha-footer {
            padding: 14px 20px;
            background: #fafafa;
            border-top: 1px solid #f0f0f0;
        }
        .captcha-security-badge {
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 8px;
            font-size: 12px;
            color: #28a745;
        }
        .captcha-security-badge i {
            font-size: 14px;
        }
        .captcha-image-skeleton,
        .captcha-click-skeleton {
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: linear-gradient(135deg, #f0f0f0 0%, #e8e8e8 50%, #f0f0f0 100%);
            background-size: 200% 100%;
            display: none;
            overflow: hidden;
            z-index: 5;
            border-radius: 8px;
            animation: skeleton-shimmer 2s linear infinite;
        }
        .captcha-image-skeleton.active,
        .captcha-click-skeleton.active {
            display: block;
        }
        @keyframes skeleton-shimmer {
            0% { background-position: -200% 0; }
            100% { background-position: 200% 0; }
        }
        .skeleton-shimmer {
            position: absolute;
            top: 0;
            left: -100%;
            width: 100%;
            height: 100%;
            background: linear-gradient(
                90deg,
                transparent 0%,
                rgba(255,255,255,0.5) 50%,
                transparent 100%
            );
            animation: shimmer-move 1.5s infinite;
        }
        @keyframes shimmer-move {
            0% { left: -100%; }
            100% { left: 100%; }
        }
        .success-particle {
            animation: particle-fly 0.6s ease-out forwards;
        }
        @keyframes particle-fly {
            to { opacity: 0; }
        }
        .error-shake {
            animation: error-shake-animation 0.5s ease;
        }
        @keyframes error-shake-animation {
            0%, 100% { transform: translateX(0); }
            20% { transform: translateX(-6px); }
            40% { transform: translateX(6px); }
            60% { transform: translateX(-4px); }
            80% { transform: translateX(4px); }
        }
        @media (max-width: 576px) {
            .captcha-container {
                border-radius: 10px;
                margin: 0 12px;
            }
            .captcha-header {
                padding: 16px;
            }
            .captcha-header h3 {
                font-size: 18px;
            }
            .captcha-body {
                padding: 16px;
            }
            .captcha-tabs {
                gap: 6px;
            }
            .captcha-tab {
                padding: 8px 12px;
                font-size: 13px;
                min-width: 70px;
            }
            .captcha-image-wrapper {
                max-width: 100%;
                margin-bottom: 12px;
            }
            .captcha-slider-container {
                max-width: 100%;
                height: 44px;
                border-radius: 22px;
            }
            .captcha-slider-button {
                width: 40px;
                height: 40px;
            }
            .captcha-slider-track {
                height: 40px;
            }
            .captcha-slider-text {
                line-height: 44px;
                font-size: 13px;
            }
            .captcha-click-image {
                max-width: 100%;
            }
            .captcha-click-marker {
                width: 32px;
                height: 32px;
                font-size: 15px;
            }
            .captcha-btn {
                padding: 10px 20px;
                font-size: 13px;
                min-height: 44px;
            }
            .captcha-actions {
                flex-direction: column;
            }
            .captcha-btn {
                width: 100%;
                justify-content: center;
            }
            .captcha-refresh {
                width: 36px;
                height: 36px;
                font-size: 17px;
            }
        }
        @media (max-width: 360px) {
            .captcha-tab .tab-icon {
                display: none;
            }
            .captcha-slider-hint {
                font-size: 11px;
            }
            .captcha-header {
                padding: 12px;
            }
            .captcha-body {
                padding: 12px;
            }
        }
        @media (prefers-reduced-motion: reduce) {
            *,
            *::before,
            *::after {
                animation-duration: 0.01ms !important;
                animation-iteration-count: 1 !important;
                transition-duration: 0.01ms !important;
            }
        }
        .visually-hidden {
            position: absolute;
            width: 1px;
            height: 1px;
            padding: 0;
            margin: -1px;
            overflow: hidden;
            clip: rect(0, 0, 0, 0);
            white-space: nowrap;
            border: 0;
        }
        @media (prefers-contrast: high) {
            .captcha-tab.active {
                border: 2px solid #c9a96e;
            }
            .captcha-slider-button {
                border: 2px solid #c9a96e;
            }
            .captcha-container {
                border: 2px solid #c9a96e;
            }
        }
        [data-high-contrast="true"] .captcha-tab.active {
            border: 3px solid #ffd700;
            background: #000;
        }
        [data-high-contrast="true"] .captcha-slider-button {
            border: 3px solid #ffd700;
        }
        [data-high-contrast="true"] .captcha-container {
            border: 3px solid #ffd700;
        }
    `;

    document.head.appendChild(styleSheet);
}