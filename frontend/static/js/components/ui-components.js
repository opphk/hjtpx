(function(globalContext) {
    'use strict';

    var UIModule = (function() {
        var styleInjected = false;

        function injectBaseStyles() {
            if (document.getElementById('captcha-ui-base-styles')) {
                return;
            }

            var style = document.createElement('style');
            style.id = 'captcha-ui-base-styles';
            style.textContent = `
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
                .captcha-tab:hover {
                    background: rgba(201, 169, 110, 0.1);
                }
                .captcha-tab.active {
                    background: linear-gradient(135deg, #c9a96e 0%, #d4b87a 100%);
                    color: white;
                    box-shadow: 0 2px 8px rgba(201, 169, 110, 0.3);
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
                .captcha-btn-secondary {
                    background: #f5f5f5;
                    color: #666;
                    border: 1px solid #e9ecef;
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
                @keyframes resultFadeIn {
                    from { opacity: 0; transform: translateY(-10px); }
                    to { opacity: 1; transform: translateY(0); }
                }
                @media (max-width: 576px) {
                    .captcha-container {
                        border-radius: 10px;
                        margin: 0 12px;
                    }
                    .captcha-header {
                        padding: 16px;
                    }
                    .captcha-body {
                        padding: 16px;
                    }
                }
            `;

            document.head.appendChild(style);
            styleInjected = true;
        }

        function createToast(message, type, options) {
            options = options || {};
            type = type || 'info';

            var existing = document.querySelector('.captcha-toast');
            if (existing) existing.remove();

            var toast = document.createElement('div');
            toast.className = 'captcha-toast captcha-toast-' + type;
            toast.setAttribute('role', 'alert');
            toast.setAttribute('aria-live', type === 'error' ? 'assertive' : 'polite');

            var icons = {
                success: 'fa-check-circle',
                error: 'fa-exclamation-circle',
                warning: 'fa-exclamation-triangle',
                info: 'fa-info-circle'
            };

            toast.innerHTML = '<i class="fas ' + (icons[type] || icons.info) + '"></i><span>' + message + '</span>';

            toast.style.cssText = [
                'position: fixed',
                'bottom: 20px',
                'right: 20px',
                'background: ' + getToastColor(type),
                'color: white',
                'padding: 14px 20px',
                'border-radius: 8px',
                'display: flex',
                'align-items: center',
                'gap: 10px',
                'z-index: 99999',
                'box-shadow: 0 4px 12px rgba(0,0,0,0.15)',
                'animation: toastSlideIn 0.3s ease'
            ].join(';');

            injectToastStyles();

            document.body.appendChild(toast);

            var duration = options.duration || (type === 'error' ? 5000 : 3000);
            setTimeout(function() {
                if (toast.parentElement) {
                    toast.style.animation = 'toastSlideOut 0.3s ease forwards';
                    setTimeout(function() {
                        if (toast.parentElement) toast.remove();
                    }, 300);
                }
            }, duration);

            return toast;
        }

        function getToastColor(type) {
            var colors = {
                success: 'linear-gradient(135deg, #28a745, #20c997)',
                error: 'linear-gradient(135deg, #dc3545, #fd7e14)',
                warning: 'linear-gradient(135deg, #ffc107, #fd7e14)',
                info: 'linear-gradient(135deg, #17a2b8, #007bff)'
            };
            return colors[type] || colors.info;
        }

        function injectToastStyles() {
            if (document.getElementById('captcha-toast-styles')) {
                return;
            }

            var style = document.createElement('style');
            style.id = 'captcha-toast-styles';
            style.textContent = `
                @keyframes toastSlideIn {
                    from { transform: translateX(120%); opacity: 0; }
                    to { transform: translateX(0); opacity: 1; }
                }
                @keyframes toastSlideOut {
                    from { transform: translateX(0); opacity: 1; }
                    to { transform: translateX(120%); opacity: 0; }
                }
                .captcha-toast i {
                    font-size: 18px;
                }
                .captcha-toast span {
                    font-size: 14px;
                    font-weight: 500;
                }
            `;

            document.head.appendChild(style);
        }

        function createLoadingOverlay(message) {
            message = message || '加载中...';

            var overlay = document.createElement('div');
            overlay.className = 'captcha-loading-overlay';
            overlay.innerHTML = '<div class="loading-dots"><span></span><span></span><span></span></div><p>' + message + '</p>';

            overlay.style.cssText = [
                'position: absolute',
                'top: 0',
                'left: 0',
                'right: 0',
                'bottom: 0',
                'background: rgba(255,255,255,0.98)',
                'display: flex',
                'flex-direction: column',
                'align-items: center',
                'justify-content: center',
                'z-index: 20',
                'border-radius: 8px'
            ].join(';');

            injectLoadingStyles();

            return overlay;
        }

        function injectLoadingStyles() {
            if (document.getElementById('captcha-loading-styles')) {
                return;
            }

            var style = document.createElement('style');
            style.id = 'captcha-loading-styles';
            style.textContent = `
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
                @keyframes loading-bounce {
                    0%, 80%, 100% { transform: scale(0); opacity: 0.4; }
                    40% { transform: scale(1); opacity: 1; }
                }
                .captcha-loading-overlay p {
                    margin-top: 15px;
                    color: #666;
                    font-size: 14px;
                }
            `;

            document.head.appendChild(style);
        }

        function showLoading(container, message) {
            var overlay = createLoadingOverlay(message);
            container.style.position = 'relative';
            container.appendChild(overlay);
            return overlay;
        }

        function hideLoading(overlay) {
            if (overlay && overlay.parentElement) {
                overlay.style.transition = 'opacity 0.3s ease';
                overlay.style.opacity = '0';
                setTimeout(function() {
                    if (overlay.parentElement) {
                        overlay.parentElement.removeChild(overlay);
                    }
                }, 300);
            }
        }

        function setElementState(element, state) {
            if (!element) return;

            var states = {
                loading: function() {
                    element.disabled = true;
                    element.setAttribute('data-original-text', element.innerHTML);
                    element.innerHTML = '<i class="fas fa-spinner fa-spin"></i> 处理中...';
                },
                success: function() {
                    element.disabled = false;
                    element.classList.add('success');
                    var originalText = element.getAttribute('data-original-text');
                    if (originalText) {
                        element.innerHTML = '<i class="fas fa-check"></i> ' + originalText;
                    }
                },
                error: function() {
                    element.disabled = false;
                    element.classList.add('error');
                    var originalText = element.getAttribute('data-original-text');
                    if (originalText) {
                        element.innerHTML = originalText;
                    }
                },
                reset: function() {
                    element.disabled = false;
                    element.classList.remove('success', 'error');
                    var originalText = element.getAttribute('data-original-text');
                    if (originalText) {
                        element.innerHTML = originalText;
                    }
                }
            };

            if (states[state]) {
                states[state]();
            }
        }

        function addAccessibilityAttributes(element, type) {
            if (!element) return;

            switch(type) {
                case 'button':
                    element.setAttribute('role', 'button');
                    if (!element.hasAttribute('tabindex')) {
                        element.setAttribute('tabindex', '0');
                    }
                    break;
                case 'image':
                    element.setAttribute('role', 'img');
                    if (!element.hasAttribute('alt')) {
                        element.setAttribute('alt', '验证码图片');
                    }
                    break;
                case 'input':
                    if (!element.hasAttribute('aria-label')) {
                        element.setAttribute('aria-label', '验证码输入框');
                    }
                    break;
            }
        }

        function injectAnimationStyles() {
            if (document.getElementById('captcha-animation-styles')) {
                return;
            }

            var style = document.createElement('style');
            style.id = 'captcha-animation-styles';
            style.textContent = `
                .animate-in {
                    animation: animateIn 0.6s cubic-bezier(0.4, 0, 0.2, 1) forwards;
                }
                @keyframes animateIn {
                    to { opacity: 1; transform: translateY(0); }
                }
                .animate-prepare {
                    opacity: 0;
                    transform: translateY(30px);
                }
                @keyframes shake {
                    0%, 100% { transform: translateX(0); }
                    20% { transform: translateX(-8px); }
                    40% { transform: translateX(8px); }
                    60% { transform: translateX(-6px); }
                    80% { transform: translateX(6px); }
                }
                .error-shake {
                    animation: shake 0.5s ease-in-out;
                }
                @keyframes success-bounce {
                    0% { transform: scale(1); }
                    50% { transform: scale(1.2); }
                    100% { transform: scale(1); }
                }
                .success-bounce {
                    animation: success-bounce 0.6s cubic-bezier(0.175, 0.885, 0.32, 1.275);
                }
                @media (prefers-reduced-motion: reduce) {
                    *, *::before, *::after {
                        animation-duration: 0.01ms !important;
                        transition-duration: 0.01ms !important;
                    }
                }
            `;

            document.head.appendChild(style);
        }

        return {
            injectBaseStyles: injectBaseStyles,
            createToast: createToast,
            showLoading: showLoading,
            hideLoading: hideLoading,
            setElementState: setElementState,
            addAccessibilityAttributes: addAccessibilityAttributes,
            injectAnimationStyles: injectAnimationStyles
        };
    })();

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = UIModule;
    } else {
        globalContext.UIModule = UIModule;
    }

})(typeof window !== 'undefined' ? window : this);
