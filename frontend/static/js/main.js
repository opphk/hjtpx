document.addEventListener('DOMContentLoaded', function() {
    console.log('用户端已加载');

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
            this.style.transform = 'scale(1.05)';
        });
        btn.addEventListener('mouseleave', function() {
            this.style.transform = 'scale(1)';
        });
    });

    initErrorHandling();
    initRetryMechanism();
    initImageOptimization();
    initPerformanceMetrics();
    
    injectCaptchaStyles();
});

window.addEventListener('error', function(e) {
    console.error('Page error:', e.error);
    showErrorHint('页面发生错误，请刷新重试', 'error');
});

window.addEventListener('unhandledrejection', function(e) {
    console.error('Unhandled promise rejection:', e.reason);
    showErrorHint('网络请求失败，请检查网络连接', 'warning');
});

function initErrorHandling() {
    const originalFetch = window.fetch;
    window.fetch = async function(url, options) {
        try {
            const response = await originalFetch.apply(this, arguments);
            if (!response.ok && response.status >= 400) {
                handleFetchError(url, response.status);
            }
            return response;
        } catch (error) {
            handleNetworkError(url, error);
            throw error;
        }
    };
}

function handleFetchError(url, status) {
    let errorMsg = '请求失败';
    let errorType = 'warning';
    
    switch(status) {
        case 400:
            errorMsg = '请求参数错误';
            errorType = 'error';
            break;
        case 401:
            errorMsg = '未授权，请重新登录';
            errorType = 'error';
            break;
        case 403:
            errorMsg = '无权限访问';
            errorType = 'error';
            break;
        case 404:
            errorMsg = '请求的资源不存在';
            errorType = 'warning';
            break;
        case 429:
            errorMsg = '请求过于频繁，请稍后再试';
            errorType = 'warning';
            break;
        case 500:
            errorMsg = '服务器内部错误';
            errorType = 'error';
            break;
        case 502:
        case 503:
            errorMsg = '服务暂时不可用';
            errorType = 'error';
            break;
        default:
            errorMsg = `请求失败 (${status})`;
    }
    
    showErrorHint(errorMsg, errorType);
}

function handleNetworkError(url, error) {
    if (!navigator.onLine) {
        showErrorHint('网络连接已断开，请检查网络', 'error');
    } else {
        showErrorHint('网络连接失败，请稍后重试', 'error');
    }
}

function showErrorHint(message, type) {
    const errorContainer = document.getElementById('captcha-error-container') || createErrorContainer();
    
    const errorHtml = `
        <div class="captcha-error-hint captcha-error-${type}" role="alert" aria-live="assertive">
            <i class="fas fa-${type === 'error' ? 'exclamation-circle' : type === 'warning' ? 'exclamation-triangle' : 'info-circle'}" aria-hidden="true"></i>
            <div class="captcha-error-hint-text">
                <div class="captcha-error-hint-title">${type === 'error' ? '错误' : type === 'warning' ? '警告' : '提示'}</div>
                <div class="captcha-error-hint-desc">${message}</div>
            </div>
        </div>
    `;
    
    errorContainer.innerHTML = errorHtml;
    errorContainer.style.display = 'block';
    
    setTimeout(() => {
        errorContainer.style.display = 'none';
    }, 5000);
}

function createErrorContainer() {
    const container = document.createElement('div');
    container.id = 'captcha-error-container';
    container.className = 'captcha-error-container';
    container.style.cssText = 'position: fixed; top: 80px; right: 20px; z-index: 9999; max-width: 400px;';
    document.body.appendChild(container);
    return container;
}

function initRetryMechanism() {
    window.captchaRetryConfig = {
        maxRetries: 3,
        retryDelay: 1000,
        backoffMultiplier: 2,
        retryCount: {},
        lastRetryTime: {}
    };
}

window.retryWithBackoff = async function(asyncFn, context, customOptions = {}) {
    const config = window.captchaRetryConfig;
    const options = { ...config, ...customOptions };
    const contextKey = context || 'default';
    
    if (!options.retryCount[contextKey]) {
        options.retryCount[contextKey] = 0;
    }
    
    while (options.retryCount[contextKey] < options.maxRetries) {
        try {
            const result = await asyncFn();
            options.retryCount[contextKey] = 0;
            return result;
        } catch (error) {
            options.retryCount[contextKey]++;
            
            if (options.retryCount[contextKey] >= options.maxRetries) {
                options.retryCount[contextKey] = 0;
                throw error;
            }
            
            const delay = options.retryDelay * Math.pow(options.backoffMultiplier, options.retryCount[contextKey] - 1);
            options.lastRetryTime[contextKey] = Date.now();
            
            console.log(`重试 ${options.retryCount[contextKey]}/${options.maxRetries}，等待 ${delay}ms`);
            
            await new Promise(resolve => setTimeout(resolve, delay));
        }
    }
};

window.safeRequest = async function(url, options = {}) {
    const defaultOptions = {
        method: 'GET',
        headers: { 'Content-Type': 'application/json' },
        timeout: 10000,
        ...options
    };
    
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), defaultOptions.timeout);
    
    try {
        const response = await window.retryWithBackoff(
            () => fetch(url, { ...defaultOptions, signal: controller.signal }),
            url,
            { maxRetries: 2 }
        );
        
        clearTimeout(timeoutId);
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        
        return await response.json();
    } catch (error) {
        clearTimeout(timeoutId);
        
        if (error.name === 'AbortError') {
            showErrorHint('请求超时，请稍后重试', 'warning');
        }
        
        throw error;
    }
};

function initImageOptimization() {
    const images = document.querySelectorAll('img[loading="lazy"]');
    images.forEach(img => {
        if ('IntersectionObserver' in window) {
            const observer = new IntersectionObserver((entries, obs) => {
                entries.forEach(entry => {
                    if (entry.isIntersecting) {
                        const target = entry.target;
                        if (target.dataset.src) {
                            target.src = target.dataset.src;
                            target.removeAttribute('data-src');
                        }
                        obs.unobserve(target);
                    }
                });
            }, { rootMargin: '50px 0px', threshold: 0.1 });
            
            observer.observe(img);
        }
    });
}

function initPerformanceMetrics() {
    if (!window.PerformanceObserver) return;
    
    try {
        const perfObserver = new PerformanceObserver((list) => {
            for (const entry of list.getEntries()) {
                if (entry.entryType === 'navigation') {
                    const timing = entry;
                    console.log('页面加载时间:', timing.loadEventEnd - timing.fetchStart, 'ms');
                    console.log('DOM加载时间:', timing.domContentLoadedEventEnd - timing.fetchStart, 'ms');
                    console.log('完整加载时间:', timing.loadEventEnd - timing.navigationStart, 'ms');
                }
            }
        });
        
        perfObserver.observe({ entryTypes: ['navigation'] });
    } catch (e) {
        console.log('Performance metrics not available');
    }
}

function preloadCriticalResources() {
    const criticalResources = [
        'https://cdn.bootcdn.net/ajax/libs/twitter-bootstrap/5.3.8/css/bootstrap.min.css',
        'https://cdn.bootcdn.net/ajax/libs/font-awesome/6.5.1/css/all.min.css'
    ];
    
    criticalResources.forEach(href => {
        const link = document.createElement('link');
        link.rel = 'preload';
        link.as = 'style';
        link.href = href;
        document.head.appendChild(link);
    });
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
            box-shadow: 0 4px 20px rgba(0,0,0,0.08);
            overflow: hidden;
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
        }
        .captcha-header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
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
            gap: 10px;
            margin-bottom: 15px;
            border-bottom: 2px solid #f0f0f0;
            padding-bottom: 10px;
        }
        .captcha-tab {
            flex: 1;
            padding: 10px 20px;
            border: none;
            background: transparent;
            color: #666;
            font-size: 14px;
            font-weight: 500;
            cursor: pointer;
            border-radius: 6px;
            transition: all 0.3s ease;
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 8px;
        }
        .captcha-tab .tab-icon {
            font-size: 16px;
        }
        .captcha-tab:hover {
            background: #f5f5f5;
        }
        .captcha-tab.active {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
        }
        .captcha-content {
            display: none;
        }
        .captcha-content.active {
            display: block;
        }
        .captcha-loading-overlay {
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: rgba(255,255,255,0.95);
            display: flex;
            align-items: center;
            justify-content: center;
            z-index: 20;
            backdrop-filter: blur(4px);
        }
        .captcha-loading-container {
            text-align: center;
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
            width: 12px;
            height: 12px;
            background: #667eea;
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
                opacity: 0.5;
            }
            40% { 
                transform: scale(1);
                opacity: 1;
            }
        }
        .loading-progress-bar {
            width: 200px;
            height: 4px;
            background: #e0e0e0;
            border-radius: 2px;
            overflow: hidden;
            margin: 15px auto;
        }
        .loading-progress-fill {
            height: 100%;
            background: linear-gradient(90deg, #667eea 0%, #764ba2 100%);
            width: 0%;
            transition: width 0.3s ease;
        }
        .loading-message {
            color: #666;
            font-size: 14px;
        }
        .captcha-image-wrapper {
            position: relative;
            width: 100%;
            max-width: 360px;
            margin: 0 auto 15px;
            border-radius: 8px;
            overflow: hidden;
            background: #f5f5f5;
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
            background: rgba(255,255,255,0.3);
            border: 2px solid rgba(255,255,255,0.8);
            box-shadow: 0 2px 8px rgba(0,0,0,0.2);
        }
        .puzzle-piece-circle {
            width: 50px;
            height: 50px;
            background: rgba(255,255,255,0.3);
            border: 2px solid rgba(255,255,255,0.8);
            border-radius: 50%;
            box-shadow: 0 2px 8px rgba(0,0,0,0.2);
        }
        .puzzle-piece-triangle {
            width: 0;
            height: 0;
            border-left: 25px solid transparent;
            border-right: 25px solid transparent;
            border-bottom: 43px solid rgba(255,255,255,0.4);
            background: transparent;
        }
        .puzzle-piece-diamond {
            width: 50px;
            height: 50px;
            background: rgba(255,255,255,0.3);
            border: 2px solid rgba(255,255,255,0.8);
            transform: rotate(45deg);
            margin: 5px;
        }
        .puzzle-piece-hexagon {
            width: 50px;
            height: 28.87px;
            background: rgba(255,255,255,0.3);
            border: 2px solid rgba(255,255,255,0.8);
            position: relative;
            margin-top: 10px;
        }
        .captcha-refresh {
            position: absolute;
            top: 8px;
            right: 8px;
            width: 32px;
            height: 32px;
            border: none;
            background: rgba(255,255,255,0.9);
            border-radius: 50%;
            cursor: pointer;
            font-size: 16px;
            display: flex;
            align-items: center;
            justify-content: center;
            transition: all 0.3s ease;
            z-index: 10;
            color: #666;
        }
        .captcha-refresh:hover {
            background: white;
            transform: rotate(180deg);
        }
        .captcha-refresh:focus {
            outline: 2px solid #667eea;
            outline-offset: 2px;
        }
        .captcha-slider-container {
            position: relative;
            width: 100%;
            max-width: 360px;
            height: 44px;
            margin: 0 auto;
            background: #f5f5f5;
            border-radius: 22px;
            overflow: hidden;
            border: 1px solid #e0e0e0;
            transition: all 0.3s ease;
        }
        .captcha-slider-container.is-dragging {
            border-color: #667eea;
            box-shadow: 0 0 0 3px rgba(102,126,234,0.1);
        }
        .captcha-slider-container.error-flash {
            animation: error-flash-animation 0.5s ease;
        }
        @keyframes error-flash-animation {
            0%, 100% { background: #f5f5f5; }
            50% { background: #fff2f0; }
        }
        .captcha-slider-track {
            position: absolute;
            left: 2px;
            top: 2px;
            height: 40px;
            width: 0;
            background: linear-gradient(90deg, #667eea 0%, #764ba2 100%);
            border-radius: 20px;
            transition: width 0.1s ease;
        }
        .captcha-slider-text {
            position: absolute;
            width: 100%;
            text-align: center;
            line-height: 44px;
            font-size: 14px;
            color: #666;
            pointer-events: none;
            z-index: 1;
        }
        .captcha-slider-hint {
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 6px;
            margin-top: 8px;
            font-size: 12px;
            color: #999;
        }
        .captcha-slider-hint .hint-icon {
            color: #667eea;
        }
        .captcha-slider-button {
            position: absolute;
            left: 2px;
            top: 2px;
            width: 40px;
            height: 40px;
            background: white;
            border-radius: 50%;
            cursor: grab;
            display: flex;
            align-items: center;
            justify-content: center;
            box-shadow: 0 2px 8px rgba(0,0,0,0.15);
            transition: left 0.1s ease, transform 0.2s ease, background 0.3s ease;
            z-index: 2;
            color: #667eea;
            touch-action: none;
        }
        .captcha-slider-button:hover {
            transform: scale(1.05);
        }
        .captcha-slider-button.dragging {
            cursor: grabbing;
            transform: scale(1.1);
            box-shadow: 0 4px 12px rgba(102,126,234,0.4);
        }
        .captcha-slider-button.verifying {
            animation: pulse-verifying 1s infinite;
        }
        @keyframes pulse-verifying {
            0%, 100% { box-shadow: 0 2px 8px rgba(102,126,234,0.3); }
            50% { box-shadow: 0 2px 16px rgba(102,126,234,0.6); }
        }
        .captcha-slider-button.success {
            background: #52c41a;
            color: white;
            animation: success-bounce 0.5s ease;
        }
        @keyframes success-bounce {
            0% { transform: scale(1); }
            50% { transform: scale(1.2); }
            100% { transform: scale(1); }
        }
        .captcha-slider-button.error {
            background: #ff4d4f;
            color: white;
            animation: shake 0.5s ease-in-out;
        }
        @keyframes shake {
            0%, 100% { transform: translateX(0); }
            25% { transform: translateX(-10px); }
            75% { transform: translateX(10px); }
        }
        .captcha-click-hint {
            text-align: center;
            padding: 10px;
            background: #f8f8f8;
            border-radius: 6px;
            margin-bottom: 10px;
            font-size: 14px;
            color: #333;
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 8px;
        }
        .captcha-click-hint .hint-icon {
            color: #667eea;
        }
        .captcha-click-grid {
            position: relative;
            display: inline-block;
        }
        .captcha-click-image {
            display: block;
            width: 100%;
            max-width: 360px;
            border-radius: 8px;
        }
        .captcha-click-marker {
            position: absolute;
            width: 28px;
            height: 28px;
            background: #667eea;
            border: 2px solid white;
            border-radius: 50%;
            color: white;
            font-size: 14px;
            font-weight: bold;
            display: flex;
            align-items: center;
            justify-content: center;
            transform: translate(-50%, -50%);
            cursor: pointer;
            box-shadow: 0 2px 8px rgba(0,0,0,0.2);
            animation: marker-pop 0.3s cubic-bezier(0.175, 0.885, 0.32, 1.275);
            transition: transform 0.2s ease, background 0.2s ease;
            z-index: 10;
        }
        .captcha-click-marker:hover {
            transform: translate(-50%, -50%) scale(1.1);
            background: #764ba2;
        }
        .captcha-click-marker.success-marker {
            background: #52c41a;
            animation: marker-success 0.5s ease;
        }
        @keyframes marker-success {
            0% { transform: translate(-50%, -50%) scale(1); }
            50% { transform: translate(-50%, -50%) scale(1.3); }
            100% { transform: translate(-50%, -50%) scale(1); }
        }
        @keyframes marker-pop {
            0% { transform: translate(-50%, -50%) scale(0); }
            50% { transform: translate(-50%, -50%) scale(1.2); }
            100% { transform: translate(-50%, -50%) scale(1); }
        }
        .captcha-click-progress {
            text-align: center;
            margin: 10px 0;
            font-size: 14px;
            color: #666;
        }
        .count-badge {
            display: inline-block;
            min-width: 24px;
            padding: 2px 8px;
            background: #f0f0f0;
            border-radius: 12px;
            font-weight: 600;
            transition: all 0.3s ease;
        }
        .count-badge.partial {
            background: #e6f7ff;
            color: #1890ff;
        }
        .count-badge.complete {
            background: #f6ffed;
            color: #52c41a;
        }
        .captcha-actions {
            display: flex;
            gap: 10px;
            justify-content: center;
            margin-top: 15px;
        }
        .captcha-btn {
            padding: 10px 30px;
            border: none;
            border-radius: 6px;
            font-size: 14px;
            font-weight: 500;
            cursor: pointer;
            transition: all 0.3s ease;
            display: inline-flex;
            align-items: center;
            gap: 6px;
        }
        .captcha-btn i {
            font-size: 14px;
        }
        .captcha-btn-primary {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
        }
        .captcha-btn-primary:hover {
            opacity: 0.9;
            transform: translateY(-2px);
            box-shadow: 0 4px 12px rgba(102,126,234,0.4);
        }
        .captcha-btn-primary:focus {
            outline: 2px solid #667eea;
            outline-offset: 2px;
        }
        .captcha-btn-secondary {
            background: #f5f5f5;
            color: #666;
        }
        .captcha-btn-secondary:hover {
            background: #e8e8e8;
        }
        .captcha-result {
            text-align: center;
            padding: 12px;
            margin-top: 15px;
            border-radius: 6px;
            font-size: 14px;
            font-weight: 500;
            display: none;
        }
        .captcha-result.show {
            display: block;
            animation: fadeIn 0.3s ease;
        }
        .captcha-result.success {
            background: #f6ffed;
            color: #52c41a;
            border: 1px solid #b7eb8f;
        }
        .captcha-result.error {
            background: #fff2f0;
            color: #ff4d4f;
            border: 1px solid #ffccc7;
        }
        @keyframes fadeIn {
            from { opacity: 0; transform: translateY(-10px); }
            to { opacity: 1; transform: translateY(0); }
        }
        .captcha-footer {
            padding: 12px 20px;
            background: #fafafa;
            border-top: 1px solid #f0f0f0;
        }
        .captcha-security-badge {
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 6px;
            font-size: 12px;
            color: #52c41a;
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
            background: #f0f0f0;
            display: none;
            overflow: hidden;
            z-index: 5;
            border-radius: 8px;
        }
        .captcha-image-skeleton.active,
        .captcha-click-skeleton.active {
            display: block;
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
                rgba(255,255,255,0.4) 50%,
                transparent 100%
            );
            animation: shimmer 1.5s infinite;
        }
        @keyframes shimmer {
            0% { left: -100%; }
            100% { left: 100%; }
        }
        .success-particle {
            animation: particle-fly 0.6s ease-out forwards;
        }
        @keyframes particle-fly {
            to {
                opacity: 0;
            }
        }
        .error-shake {
            animation: error-shake-animation 0.5s ease;
        }
        @keyframes error-shake-animation {
            0%, 100% { transform: translateX(0); }
            25% { transform: translateX(-5px); }
            75% { transform: translateX(5px); }
        }
        @media (max-width: 576px) {
            .captcha-container {
                border-radius: 8px;
                margin: 0 10px;
            }
            .captcha-header {
                padding: 15px;
            }
            .captcha-header h3 {
                font-size: 18px;
            }
            .captcha-body {
                padding: 15px;
            }
            .captcha-tabs {
                flex-direction: column;
                gap: 5px;
            }
            .captcha-tab {
                padding: 8px 16px;
                font-size: 13px;
            }
            .captcha-image-wrapper {
                max-width: 100%;
                margin-bottom: 10px;
            }
            .captcha-slider-container {
                max-width: 100%;
                height: 40px;
                border-radius: 20px;
            }
            .captcha-slider-button {
                width: 36px;
                height: 36px;
            }
            .captcha-slider-track {
                height: 36px;
            }
            .captcha-slider-text {
                line-height: 40px;
                font-size: 13px;
            }
            .captcha-click-image {
                max-width: 100%;
            }
            .captcha-btn {
                padding: 8px 20px;
                font-size: 13px;
            }
            .captcha-actions {
                flex-direction: column;
            }
            .captcha-btn {
                width: 100%;
                justify-content: center;
            }
        }
        @media (max-width: 360px) {
            .captcha-tab .tab-icon {
                display: none;
            }
            .captcha-slider-hint {
                font-size: 11px;
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
        
        .gpu-accelerated {
            transform: translateZ(0);
            -webkit-transform: translateZ(0);
            -moz-transform: translateZ(0);
            -ms-transform: translateZ(0);
            -o-transform: translateZ(0);
            backface-visibility: hidden;
            -webkit-backface-visibility: hidden;
            -moz-backface-visibility: hidden;
            -ms-backface-visibility: hidden;
            perspective: 1000px;
            -webkit-perspective: 1000px;
            will-change: transform, opacity;
            -webkit-will-change: transform, opacity;
        }
        
        .captcha-loading-overlay {
            will-change: opacity;
        }
        
        .captcha-slider-container {
            will-change: transform;
        }
        
        .captcha-slider-button {
            will-change: left, transform;
        }
        
        .captcha-click-marker {
            will-change: transform, opacity;
        }
        
        .loading-dots span {
            transform: translateZ(0);
            -webkit-transform: translateZ(0);
        }
        
        @keyframes loading-bounce {
            0%, 80%, 100% { 
                transform: scale(0);
                opacity: 0.5;
            }
            40% { 
                transform: scale(1);
                opacity: 1;
            }
        }
        
        .captcha-error-container {
            will-change: transform, opacity;
        }
        
        .captcha-error-hint {
            transform: translateX(100%);
            opacity: 0;
            animation: slideInError 0.3s ease forwards;
        }
        
        @keyframes slideInError {
            to {
                transform: translateX(0);
                opacity: 1;
            }
        }
        
        .captcha-progress-container {
            will-change: width;
        }
        
        .captcha-progress-fill {
            will-change: width;
        }
        
        .skeleton-shimmer {
            will-change: transform;
        }
        
        .captcha-refresh {
            will-change: transform;
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
    `;

    document.head.appendChild(styleSheet);
}
