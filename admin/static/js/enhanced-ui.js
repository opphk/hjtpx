/**
 * 用户交互体验增强模块
 * 功能：表单验证、加载状态、错误提示增强
 */
document.addEventListener('DOMContentLoaded', function() {
    initEnhancedFormValidation();
    initEnhancedLoadingStates();
    initEnhancedErrorHandling();
    initEnhancedNotifications();
    initEnhancedTooltips();
    initEnhancedConfirmDialogs();
});

/**
 * 增强表单验证
 */
function initEnhancedFormValidation() {
    document.querySelectorAll('form').forEach(form => {
        const inputs = form.querySelectorAll('input, select, textarea');
        
        inputs.forEach(input => {
            input.addEventListener('blur', function() {
                validateField(this);
            });
            
            input.addEventListener('input', function() {
                if (this.classList.contains('is-invalid')) {
                    validateField(this);
                }
            });
        });
        
        form.addEventListener('submit', function(e) {
            if (!validateForm(this)) {
                e.preventDefault();
                showValidationErrors();
            }
        });
    });
}

function validateField(field) {
    const value = field.value.trim();
    const type = field.type;
    const required = field.hasAttribute('required');
    const minLength = field.getAttribute('minlength');
    const maxLength = field.getAttribute('maxlength');
    const pattern = field.getAttribute('pattern');
    
    clearFieldError(field);
    
    if (required && !value) {
        showFieldError(field, '此字段为必填项');
        return false;
    }
    
    if (value) {
        if (type === 'email' && !isValidEmail(value)) {
            showFieldError(field, '请输入有效的邮箱地址');
            return false;
        }
        
        if (type === 'url' && !isValidUrl(value)) {
            showFieldError(field, '请输入有效的URL地址');
            return false;
        }
        
        if (type === 'tel' && !isValidPhone(value)) {
            showFieldError(field, '请输入有效的电话号码');
            return false;
        }
        
        if (minLength && value.length < parseInt(minLength)) {
            showFieldError(field, `至少需要${minLength}个字符`);
            return false;
        }
        
        if (maxLength && value.length > parseInt(maxLength)) {
            showFieldError(field, `最多只能输入${maxLength}个字符`);
            return false;
        }
        
        if (pattern) {
            const regex = new RegExp(pattern);
            if (!regex.test(value)) {
                showFieldError(field, '输入格式不正确');
                return false;
            }
        }
        
        if (field.id === 'password' && value.length < 8) {
            showFieldError(field, '密码至少需要8个字符');
            return false;
        }
        
        if (field.id === 'password' && !/(?=.*[a-z])(?=.*[A-Z])(?=.*\d)/.test(value)) {
            showFieldError(field, '密码必须包含大小写字母和数字');
            return false;
        }
        
        if (field.id === 'confirmPassword') {
            const password = document.getElementById('password');
            if (password && value !== password.value) {
                showFieldError(field, '两次输入的密码不一致');
                return false;
            }
        }
    }
    
    markFieldValid(field);
    return true;
}

function validateForm(form) {
    let isValid = true;
    const inputs = form.querySelectorAll('input, select, textarea');
    
    inputs.forEach(input => {
        if (!validateField(input)) {
            isValid = false;
        }
    });
    
    return isValid;
}

function showFieldError(field, message) {
    field.classList.remove('is-valid');
    field.classList.add('is-invalid');
    
    let errorDiv = field.parentElement.querySelector('.invalid-feedback');
    if (!errorDiv) {
        errorDiv = document.createElement('div');
        errorDiv.className = 'invalid-feedback';
        field.parentElement.appendChild(errorDiv);
    }
    errorDiv.textContent = message;
}

function clearFieldError(field) {
    field.classList.remove('is-invalid', 'is-valid');
    const errorDiv = field.parentElement.querySelector('.invalid-feedback');
    if (errorDiv) {
        errorDiv.remove();
    }
}

function markFieldValid(field) {
    field.classList.remove('is-invalid');
    field.classList.add('is-valid');
}

function showValidationErrors() {
    const invalidFields = document.querySelectorAll('.is-invalid');
    if (invalidFields.length > 0) {
        invalidFields[0].focus();
        showEnhancedToast(`有${invalidFields.length}个字段验证失败，请检查输入`, 'error');
    }
}

function isValidEmail(email) {
    return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
}

function isValidUrl(url) {
    try {
        new URL(url);
        return true;
    } catch {
        return false;
    }
}

function isValidPhone(phone) {
    return /^1[3-9]\d{9}$/.test(phone.replace(/\s/g, ''));
}

/**
 * 增强加载状态
 */
function initEnhancedLoadingStates() {
    document.querySelectorAll('button[type="submit"], .btn-primary, .btn-success').forEach(btn => {
        if (!btn.classList.contains('no-loading')) {
            btn.addEventListener('click', function(e) {
                if (this.type === 'submit' || this.classList.contains('ajax-submit')) {
                    showButtonLoading(this);
                }
            });
        }
    });
    
    document.querySelectorAll('.ajax-link, [data-ajax="true"]').forEach(link => {
        link.addEventListener('click', function(e) {
            showElementLoading(this);
        });
    });
}

function showButtonLoading(button) {
    if (button.dataset.loading === 'true') return;
    
    const originalText = button.innerHTML;
    button.dataset.originalText = originalText;
    button.dataset.loading = 'true';
    button.disabled = true;
    
    button.innerHTML = `
        <span class="spinner-border spinner-border-sm mr-2" role="status" aria-hidden="true"></span>
        处理中...
    `;
    
    setTimeout(() => {
        resetButtonLoading(button);
    }, 10000);
}

function resetButtonLoading(button) {
    if (button.dataset.loading === 'true') {
        button.innerHTML = button.dataset.originalText;
        button.disabled = false;
        button.dataset.loading = 'false';
    }
}

function showElementLoading(element) {
    element.classList.add('loading');
}

function hideElementLoading(element) {
    element.classList.remove('loading');
}

function showGlobalLoading() {
    let overlay = document.getElementById('globalLoadingOverlay');
    if (!overlay) {
        overlay = document.createElement('div');
        overlay.id = 'globalLoadingOverlay';
        overlay.className = 'global-loading-overlay';
        overlay.innerHTML = `
            <div class="global-loading-spinner">
                <div class="spinner">
                    <div class="spinner-inner"></div>
                </div>
                <p>加载中，请稍候...</p>
            </div>
        `;
        document.body.appendChild(overlay);
    }
    overlay.classList.add('show');
}

function hideGlobalLoading() {
    const overlay = document.getElementById('globalLoadingOverlay');
    if (overlay) {
        overlay.classList.remove('show');
    }
}

/**
 * 增强错误处理
 */
function initEnhancedErrorHandling() {
    window.addEventListener('error', function(e) {
        logError('JavaScript Error', e.error);
    });
    
    window.addEventListener('unhandledrejection', function(e) {
        logError('Unhandled Promise Rejection', e.reason);
    });
}

function logError(type, error) {
    const errorLog = {
        type: type,
        message: error.message || error,
        stack: error.stack,
        url: window.location.href,
        timestamp: new Date().toISOString()
    };
    
    console.error(`[${errorLog.type}]`, errorLog.message);
    
    if (window.location.hostname !== 'localhost') {
        sendErrorLog(errorLog);
    }
}

function sendErrorLog(errorLog) {
    try {
        fetch('/api/v1/admin/error-log', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(errorLog),
            timeout: 5000
        }).catch(() => {});
    } catch (e) {}
}

function handleAjaxError(jqxhr, settings, thrownError) {
    let errorTitle = '请求失败';
    let errorMessage = '服务器未能处理您的请求';
    let errorSuggestion = '请检查网络连接后重试';
    let errorType = 'error';
    
    switch(jqxhr.status) {
        case 400:
            errorTitle = '请求参数错误';
            errorMessage = '您提交的数据格式不正确';
            errorSuggestion = '请检查输入内容后重试';
            break;
        case 401:
            errorTitle = '未授权访问';
            errorMessage = '请先登录后再继续操作';
            errorSuggestion = '页面将跳转到登录页面';
            setTimeout(() => window.location.href = '/admin/login', 2000);
            break;
        case 403:
            errorTitle = '权限不足';
            errorMessage = '您没有执行此操作的权限';
            errorSuggestion = '请联系管理员获取相应权限';
            break;
        case 404:
            errorTitle = '资源不存在';
            errorMessage = '请求的资源未找到';
            errorSuggestion = '请检查URL是否正确';
            break;
        case 408:
            errorTitle = '请求超时';
            errorMessage = '服务器响应时间过长';
            errorSuggestion = '请检查网络连接后重试';
            break;
        case 429:
            errorTitle = '请求过于频繁';
            errorMessage = '您的操作频率过高';
            errorSuggestion = '请稍后再试';
            break;
        case 500:
            errorTitle = '服务器内部错误';
            errorMessage = '服务器遇到问题无法完成请求';
            errorSuggestion = '请稍后重试';
            break;
        case 502:
        case 503:
        case 504:
            errorTitle = '服务暂不可用';
            errorMessage = '服务器正在维护或负载过高';
            errorSuggestion = '请稍后再试';
            break;
        default:
            if (jqxhr.responseJSON && jqxhr.responseJSON.message) {
                errorMessage = jqxhr.responseJSON.message;
            }
    }
    
    showEnhancedToast(`${errorTitle}: ${errorMessage}`, errorType);
}

function retryAction(action, maxRetries = 3) {
    let retries = 0;
    
    function attempt() {
        try {
            action();
        } catch (error) {
            retries++;
            if (retries < maxRetries) {
                showEnhancedToast(`操作失败，正在重试 (${retries}/${maxRetries})`, 'warning');
                setTimeout(attempt, 1000 * retries);
            } else {
                showEnhancedToast('操作失败，请稍后重试', 'error');
            }
        }
    }
    
    attempt();
}

/**
 * 增强通知系统
 */
function initEnhancedNotifications() {
    window.showEnhancedToast = function(message, type = 'info', duration = 3000) {
        const container = getToastContainer();
        const toastId = 'toast-' + Date.now() + '-' + Math.random().toString(36).substr(2, 9);
        
        const icons = {
            success: 'fa-check-circle',
            error: 'fa-exclamation-circle',
            warning: 'fa-exclamation-triangle',
            info: 'fa-info-circle'
        };
        
        const colors = {
            success: '#28a745',
            error: '#dc3545',
            warning: '#ffc107',
            info: '#17a2b8'
        };
        
        const toast = document.createElement('div');
        toast.id = toastId;
        toast.className = 'enhanced-toast';
        toast.innerHTML = `
            <div class="enhanced-toast-content">
                <div class="enhanced-toast-icon" style="color: ${colors[type]}">
                    <i class="fas ${icons[type]}"></i>
                </div>
                <div class="enhanced-toast-message">${escapeHtml(message)}</div>
                <button class="enhanced-toast-close" onclick="closeEnhancedToast('${toastId}')">
                    <i class="fas fa-times"></i>
                </button>
            </div>
            <div class="enhanced-toast-progress" style="background-color: ${colors[type]}"></div>
        `;
        
        container.appendChild(toast);
        
        requestAnimationFrame(() => {
            toast.classList.add('show');
        });
        
        if (duration > 0) {
            setTimeout(() => {
                closeEnhancedToast(toastId);
            }, duration);
        }
        
        return toastId;
    };
}

function getToastContainer() {
    let container = document.getElementById('enhancedToastContainer');
    if (!container) {
        container = document.createElement('div');
        container.id = 'enhancedToastContainer';
        container.className = 'enhanced-toast-container';
        document.body.appendChild(container);
        
        const style = document.createElement('style');
        style.textContent = `
            .enhanced-toast-container {
                position: fixed;
                top: 20px;
                right: 20px;
                z-index: 10000;
                max-width: 400px;
            }
            .enhanced-toast {
                background: white;
                border-radius: 8px;
                box-shadow: 0 4px 12px rgba(0,0,0,0.15);
                margin-bottom: 10px;
                overflow: hidden;
                transform: translateX(100%);
                opacity: 0;
                transition: all 0.3s ease;
            }
            .enhanced-toast.show {
                transform: translateX(0);
                opacity: 1;
            }
            .enhanced-toast-content {
                display: flex;
                align-items: center;
                padding: 15px;
                gap: 12px;
            }
            .enhanced-toast-icon {
                font-size: 24px;
                flex-shrink: 0;
            }
            .enhanced-toast-message {
                flex: 1;
                font-size: 14px;
                color: #333;
                line-height: 1.5;
            }
            .enhanced-toast-close {
                background: none;
                border: none;
                color: #999;
                cursor: pointer;
                padding: 5px;
                transition: color 0.2s;
            }
            .enhanced-toast-close:hover {
                color: #333;
            }
            .enhanced-toast-progress {
                height: 3px;
                width: 100%;
                animation: toastProgress 3s linear forwards;
            }
            @keyframes toastProgress {
                from { width: 100%; }
                to { width: 0%; }
            }
            @media (max-width: 576px) {
                .enhanced-toast-container {
                    left: 10px;
                    right: 10px;
                    max-width: none;
                }
            }
        `;
        document.head.appendChild(style);
    }
    return container;
}

function closeEnhancedToast(toastId) {
    const toast = document.getElementById(toastId);
    if (toast) {
        toast.classList.remove('show');
        setTimeout(() => toast.remove(), 300);
    }
}

function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

/**
 * 增强提示工具
 */
function initEnhancedTooltips() {
    const tooltipStyle = document.createElement('style');
    tooltipStyle.textContent = `
        .enhanced-tooltip {
            position: absolute;
            background: rgba(0, 0, 0, 0.9);
            color: white;
            padding: 8px 12px;
            border-radius: 6px;
            font-size: 13px;
            max-width: 250px;
            z-index: 9999;
            pointer-events: none;
            opacity: 0;
            transition: opacity 0.2s ease;
        }
        .enhanced-tooltip.show {
            opacity: 1;
        }
        .enhanced-tooltip::after {
            content: '';
            position: absolute;
            border: 6px solid transparent;
        }
        .enhanced-tooltip.top::after {
            top: 100%;
            left: 50%;
            transform: translateX(-50%);
            border-top-color: rgba(0, 0, 0, 0.9);
        }
        .enhanced-tooltip.bottom::after {
            bottom: 100%;
            left: 50%;
            transform: translateX(-50%);
            border-bottom-color: rgba(0, 0, 0, 0.9);
        }
        .enhanced-tooltip.left::after {
            left: 100%;
            top: 50%;
            transform: translateY(-50%);
            border-left-color: rgba(0, 0, 0, 0.9);
        }
        .enhanced-tooltip.right::after {
            right: 100%;
            top: 50%;
            transform: translateY(-50%);
            border-right-color: rgba(0, 0, 0, 0.9);
        }
    `;
    document.head.appendChild(tooltipStyle);
    
    document.querySelectorAll('[data-tooltip]').forEach(element => {
        element.addEventListener('mouseenter', showTooltip);
        element.addEventListener('mouseleave', hideTooltip);
    });
}

function showTooltip(e) {
    const element = e.currentTarget;
    const text = element.dataset.tooltip;
    const placement = element.dataset.tooltipPlacement || 'top';
    
    let tooltip = document.getElementById('enhancedTooltip');
    if (!tooltip) {
        tooltip = document.createElement('div');
        tooltip.id = 'enhancedTooltip';
        tooltip.className = 'enhanced-tooltip';
        document.body.appendChild(tooltip);
    }
    
    tooltip.textContent = text;
    tooltip.className = `enhanced-tooltip ${placement}`;
    
    const rect = element.getBoundingClientRect();
    const tooltipRect = tooltip.getBoundingClientRect();
    
    let top, left;
    
    switch(placement) {
        case 'top':
            top = rect.top - tooltipRect.height - 10 + window.scrollY;
            left = rect.left + (rect.width - tooltipRect.width) / 2;
            break;
        case 'bottom':
            top = rect.bottom + 10 + window.scrollY;
            left = rect.left + (rect.width - tooltipRect.width) / 2;
            break;
        case 'left':
            top = rect.top + (rect.height - tooltipRect.height) / 2 + window.scrollY;
            left = rect.left - tooltipRect.width - 10;
            break;
        case 'right':
            top = rect.top + (rect.height - tooltipRect.height) / 2 + window.scrollY;
            left = rect.right + 10;
            break;
    }
    
    tooltip.style.top = top + 'px';
    tooltip.style.left = left + 'px';
    
    requestAnimationFrame(() => {
        tooltip.classList.add('show');
    });
}

function hideTooltip() {
    const tooltip = document.getElementById('enhancedTooltip');
    if (tooltip) {
        tooltip.classList.remove('show');
    }
}

/**
 * 增强确认对话框
 */
function initEnhancedConfirmDialogs() {
    window.showEnhancedConfirm = function(options) {
        const defaults = {
            title: '确认操作',
            message: '确定要执行此操作吗？',
            confirmText: '确认',
            cancelText: '取消',
            confirmClass: 'btn-danger',
            onConfirm: () => {},
            onCancel: () => {}
        };
        
        const settings = { ...defaults, ...options };
        
        const modalId = 'enhancedConfirmModal';
        let modal = document.getElementById(modalId);
        
        if (!modal) {
            modal = document.createElement('div');
            modal.id = modalId;
            modal.className = 'modal fade';
            modal.innerHTML = `
                <div class="modal-dialog" role="document">
                    <div class="modal-content">
                        <div class="modal-header">
                            <h5 class="modal-title"></h5>
                            <button type="button" class="close" data-dismiss="modal">
                                <span>&times;</span>
                            </button>
                        </div>
                        <div class="modal-body">
                            <p></p>
                        </div>
                        <div class="modal-footer">
                            <button type="button" class="btn btn-secondary" data-dismiss="modal"></button>
                            <button type="button" class="btn"></button>
                        </div>
                    </div>
                </div>
            `;
            document.body.appendChild(modal);
        }
        
        modal.querySelector('.modal-title').textContent = settings.title;
        modal.querySelector('.modal-body p').textContent = settings.message;
        modal.querySelector('.modal-footer .btn-secondary').textContent = settings.cancelText;
        modal.querySelector('.modal-footer .btn').textContent = settings.confirmText;
        modal.querySelector('.modal-footer .btn').className = `btn ${settings.confirmClass}`;
        
        const confirmBtn = modal.querySelector('.modal-footer .btn');
        const newConfirmBtn = confirmBtn.cloneNode(true);
        confirmBtn.parentNode.replaceChild(newConfirmBtn, confirmBtn);
        
        newConfirmBtn.addEventListener('click', function() {
            $('#' + modalId).modal('hide');
            settings.onConfirm();
        });
        
        $('#' + modalId).modal('show');
    };
}

window.showConfirm = window.showEnhancedConfirm;

/**
 * 辅助函数
 */
function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

function throttle(func, limit) {
    let inThrottle;
    return function(...args) {
        if (!inThrottle) {
            func.apply(this, args);
            inThrottle = true;
            setTimeout(() => inThrottle = false, limit);
        }
    };
}

function copyToClipboard(text, successMessage = '已复制到剪贴板') {
    if (navigator.clipboard) {
        navigator.clipboard.writeText(text).then(() => {
            showEnhancedToast(successMessage, 'success');
        }).catch(() => {
            fallbackCopyToClipboard(text, successMessage);
        });
    } else {
        fallbackCopyToClipboard(text, successMessage);
    }
}

function fallbackCopyToClipboard(text, successMessage) {
    const textarea = document.createElement('textarea');
    textarea.value = text;
    textarea.style.position = 'fixed';
    textarea.style.opacity = '0';
    document.body.appendChild(textarea);
    textarea.select();
    try {
        document.execCommand('copy');
        showEnhancedToast(successMessage, 'success');
    } catch (err) {
        showEnhancedToast('复制失败', 'error');
    }
    document.body.removeChild(textarea);
}

function isEmpty(value) {
    if (value === null || value === undefined) return true;
    if (typeof value === 'string') return value.trim() === '';
    if (Array.isArray(value)) return value.length === 0;
    if (typeof value === 'object') return Object.keys(value).length === 0;
    return false;
}

function formatFileSize(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
}

function parseQueryString(queryString) {
    const params = {};
    const pairs = (queryString || window.location.search.slice(1)).split('&');
    pairs.forEach(pair => {
        const [key, value] = pair.split('=');
        if (key) {
            params[decodeURIComponent(key)] = decodeURIComponent(value || '');
        }
    });
    return params;
}

function buildQueryString(params) {
    return Object.keys(params)
        .filter(key => params[key] !== '' && params[key] !== null && params[key] !== undefined)
        .map(key => encodeURIComponent(key) + '=' + encodeURIComponent(params[key]))
        .join('&');
}
