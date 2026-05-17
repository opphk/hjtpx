const ErrorHandler = {
    config: {
        maxRetries: 3,
        retryDelay: 1000,
        retryBackoff: 2,
        enableNotifications: true,
        enableRecovery: true,
        enableLogging: true,
        errorThreshold: 5,
        errorWindow: 60000
    },
    errors: [],
    errorCounts: new Map(),
    retryQueue: [],
    handlers: new Map(),
    recoveryStrategies: new Map(),
    isRecovering: false,

    async init(options = {}) {
        this.config = { ...this.config, ...options };
        this.setupGlobalErrorHandlers();
        this.setupUnhandledRejectionHandler();
        this.setupErrorRecovery();
        this.registerDefaultHandlers();
        this.setupErrorNotifications();
        this.setupErrorBoundaries();
        return this;
    },

    setupGlobalErrorHandlers() {
        window.addEventListener('error', (event) => {
            this.handleError(event.error || new Error(event.message), {
                type: 'window',
                filename: event.filename,
                lineno: event.lineno,
                colno: event.colno,
                message: event.message
            });
        });

        window.addEventListener('unhandledrejection', (event) => {
            this.handleError(event.reason, {
                type: 'promise',
                promise: true
            });
        });
    },

    setupUnhandledRejectionHandler() {
        const originalFetch = window.fetch;
        const self = this;

        window.fetch = async function(...args) {
            try {
                const response = await originalFetch.apply(this, args);
                if (!response.ok && response.status >= 400) {
                    throw new HttpError(response.status, response.statusText, response.url);
                }
                return response;
            } catch (error) {
                if (!(error instanceof AbortError) && !(error instanceof TypeError && error.message.includes('fetch'))) {
                    self.handleError(error, { type: 'fetch', url: args[0] });
                }
                throw error;
            }
        };
    },

    setupErrorRecovery() {
        this.recoveryStrategies.set('network', {
            detect: (error) => error.message.includes('network') || error.message.includes('fetch'),
            strategy: async () => {
                if (navigator.onLine) {
                    await this.delay(1000);
                    return true;
                }
                return false;
            }
        });

        this.recoveryStrategies.set('timeout', {
            detect: (error) => error.message.includes('timeout') || error.name === 'AbortError',
            strategy: async () => {
                await this.delay(2000);
                return true;
            }
        });

        this.recoveryStrategies.set('server', {
            detect: (error) => error instanceof HttpError && error.status >= 500,
            strategy: async () => {
                await this.delay(5000);
                return true;
            }
        });

        this.recoveryStrategies.set('client', {
            detect: (error) => error instanceof HttpError && error.status >= 400 && error.status < 500,
            strategy: async () => {
                return false;
            }
        });
    },

    registerDefaultHandlers() {
        this.registerHandler('validation', (error, context) => ({
            title: '验证失败',
            message: error.message || '验证未通过，请重试',
            type: 'error',
            action: 'refresh',
            icon: 'exclamation-circle'
        }));

        this.registerHandler('network', (error, context) => ({
            title: '网络错误',
            message: '网络连接失败，请检查网络设置',
            type: 'error',
            action: 'retry',
            icon: 'wifi-slash'
        }));

        this.registerHandler('timeout', (error, context) => ({
            title: '请求超时',
            message: '服务器响应超时，请稍后重试',
            type: 'warning',
            action: 'retry',
            icon: 'clock'
        }));

        this.registerHandler('server', (error, context) => ({
            title: '服务器错误',
            message: '服务器繁忙，请稍后再试',
            type: 'error',
            action: 'wait',
            icon: 'server'
        }));

        this.registerHandler('client', (error, context) => ({
            title: '请求错误',
            message: error.message || '请求处理失败',
            type: 'error',
            action: 'dismiss',
            icon: 'exclamation-triangle'
        }));

        this.registerHandler('unknown', (error, context) => ({
            title: '操作失败',
            message: '发生了未知错误，请稍后重试',
            type: 'error',
            action: 'retry',
            icon: 'bomb'
        }));
    },

    registerHandler(type, handler) {
        this.handlers.set(type, handler);
    },

    registerRecoveryStrategy(type, strategy) {
        this.recoveryStrategies.set(type, strategy);
    },

    handleError(error, context = {}) {
        const errorInfo = this.createErrorInfo(error, context);
        this.errors.push(errorInfo);
        this.pruneErrors();
        this.updateErrorCounts(errorInfo.type);
        this.logError(errorInfo);

        if (this.shouldShowNotification(errorInfo)) {
            this.showNotification(errorInfo);
        }

        this.dispatchEvent('error:handled', errorInfo);

        return errorInfo;
    },

    createErrorInfo(error, context = {}) {
        const errorType = this.classifyError(error);
        const handler = this.handlers.get(errorType);
        const handlerResult = handler ? handler(error, context) : this.handlers.get('unknown')(error, context);

        return {
            id: this.generateErrorId(),
            type: errorType,
            error: error instanceof Error ? error : new Error(String(error)),
            message: error.message || String(error),
            stack: error.stack,
            timestamp: Date.now(),
            context,
            handlerResult,
            retryable: this.isRetryable(error),
            recoverable: this.isRecoverable(error),
            count: this.errorCounts.get(errorType) || 1
        };
    },

    classifyError(error) {
        if (error instanceof HttpError) {
            if (error.status >= 500) return 'server';
            if (error.status >= 400) return 'client';
        }

        const message = (error.message || '').toLowerCase();

        if (message.includes('valid') || message.includes('verify') || message.includes('验证')) {
            return 'validation';
        }
        if (message.includes('network') || message.includes('fetch') || message.includes('connection')) {
            return 'network';
        }
        if (message.includes('timeout') || message.includes('aborted')) {
            return 'timeout';
        }

        return 'unknown';
    },

    isRetryable(error) {
        const nonRetryable = ['validation', 'client'];
        return !nonRetryable.includes(this.classifyError(error));
    },

    isRecoverable(error) {
        const errorType = this.classifyError(error);
        return this.recoveryStrategies.has(errorType);
    },

    async recover(errorInfo) {
        if (this.isRecovering) {
            return { success: false, reason: 'recovery_in_progress' };
        }

        const errorType = errorInfo.type;
        const strategy = this.recoveryStrategies.get(errorType);

        if (!strategy) {
            return { success: false, reason: 'no_strategy' };
        }

        this.isRecovering = true;
        this.dispatchEvent('recovery:start', { errorType });

        try {
            const success = await strategy.strategy();
            this.isRecovering = false;

            if (success) {
                this.dispatchEvent('recovery:success', { errorType });
                return { success: true };
            } else {
                this.dispatchEvent('recovery:failed', { errorType });
                return { success: false, reason: 'strategy_failed' };
            }
        } catch (e) {
            this.isRecovering = false;
            this.dispatchEvent('recovery:error', { errorType, error: e.message });
            return { success: false, reason: e.message };
        }
    },

    async retry(operation, options = {}) {
        const config = {
            maxRetries: options.maxRetries || this.config.maxRetries,
            delay: options.delay || this.config.retryDelay,
            backoff: options.backoff || this.config.retryBackoff,
            exponential: options.exponential !== false,
            shouldRetry: options.shouldRetry || (() => true)
        };

        let lastError;

        for (let attempt = 0; attempt <= config.maxRetries; attempt++) {
            if (attempt > 0) {
                await this.delay(config.delay);

                if (config.exponential) {
                    config.delay *= config.backoff;
                }
            }

            try {
                const result = await operation(attempt);
                if (attempt > 0) {
                    this.dispatchEvent('retry:success', { attempt, result });
                }
                return result;
            } catch (error) {
                lastError = error;

                if (!config.shouldRetry(error, attempt)) {
                    break;
                }

                if (attempt < config.maxRetries) {
                    this.dispatchEvent('retry:attempt', {
                        attempt: attempt + 1,
                        maxRetries: config.maxRetries,
                        error: error.message
                    });
                }
            }
        }

        this.dispatchEvent('retry:exhausted', {
            attempts: config.maxRetries + 1,
            error: lastError.message
        });

        throw lastError;
    },

    setupErrorNotifications() {
        this.notificationContainer = null;
        this.createNotificationContainer();
    },

    createNotificationContainer() {
        if (this.notificationContainer) return;

        this.notificationContainer = document.createElement('div');
        this.notificationContainer.id = 'error-notifications';
        this.notificationContainer.setAttribute('role', 'region');
        this.notificationContainer.setAttribute('aria-label', '错误通知');
        this.notificationContainer.className = 'error-notification-container';
        this.notificationContainer.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            z-index: 10000;
            display: flex;
            flex-direction: column;
            gap: 10px;
            max-width: 400px;
            width: 100%;
            pointer-events: none;
        `;
        document.body.appendChild(this.notificationContainer);
    },

    shouldShowNotification(errorInfo) {
        if (!this.config.enableNotifications) return false;

        const count = this.errorCounts.get(errorInfo.type) || 0;
        if (count > this.config.errorThreshold) {
            return false;
        }

        return true;
    },

    showNotification(errorInfo) {
        if (!this.notificationContainer) {
            this.createNotificationContainer();
        }

        const notification = document.createElement('div');
        notification.className = `error-notification error-notification-${errorInfo.handlerResult.type}`;
        notification.setAttribute('role', 'alert');
        notification.setAttribute('aria-live', 'polite');
        notification.style.cssText = `
            background: white;
            border-radius: 8px;
            padding: 16px;
            box-shadow: 0 4px 12px rgba(0,0,0,0.15);
            display: flex;
            align-items: flex-start;
            gap: 12px;
            animation: errorSlideIn 0.3s ease;
            pointer-events: auto;
            border-left: 4px solid ${this.getTypeColor(errorInfo.handlerResult.type)};
        `;

        const iconHtml = this.getIconHtml(errorInfo.handlerResult.icon, errorInfo.handlerResult.type);
        const actionHtml = this.getActionHtml(errorInfo, notification);

        notification.innerHTML = `
            <div class="error-notification-icon" style="flex-shrink: 0;">
                ${iconHtml}
            </div>
            <div class="error-notification-content" style="flex: 1; min-width: 0;">
                <div class="error-notification-title" style="font-weight: 600; margin-bottom: 4px; color: #333;">
                    ${errorInfo.handlerResult.title}
                </div>
                <div class="error-notification-message" style="font-size: 14px; color: #666; line-height: 1.5;">
                    ${errorInfo.handlerResult.message}
                </div>
                ${actionHtml}
            </div>
            <button class="error-notification-close" aria-label="关闭通知" style="
                background: none;
                border: none;
                padding: 4px;
                cursor: pointer;
                color: #999;
                flex-shrink: 0;
            ">
                <i class="fas fa-times"></i>
            </button>
        `;

        const closeBtn = notification.querySelector('.error-notification-close');
        closeBtn.addEventListener('click', () => this.dismissNotification(notification));

        const actionBtn = notification.querySelector('[data-action]');
        if (actionBtn) {
            actionBtn.addEventListener('click', () => {
                this.handleNotificationAction(errorInfo, actionBtn.dataset.action);
                this.dismissNotification(notification);
            });
        }

        if (actionBtn) {
            actionBtn.addEventListener('click', () => this.handleNotificationAction(errorInfo, actionBtn.dataset.action));
        }

        this.notificationContainer.appendChild(notification);

        if (errorInfo.handlerResult.type !== 'warning') {
            setTimeout(() => {
                this.dismissNotification(notification);
            }, 5000);
        }

        this.announceToScreenReader(`${errorInfo.handlerResult.title}，${errorInfo.handlerResult.message}`);
    },

    getTypeColor(type) {
        const colors = {
            error: '#dc3545',
            warning: '#ffc107',
            info: '#17a2b8',
            success: '#28a745'
        };
        return colors[type] || colors.error;
    },

    getIconHtml(icon, type) {
        const color = this.getTypeColor(type);
        return `<i class="fas fa-${icon}" style="font-size: 20px; color: ${color};"></i>`;
    },

    getActionHtml(errorInfo, notification) {
        const action = errorInfo.handlerResult.action;
        if (!action || action === 'dismiss') return '';

        const actionConfig = {
            retry: { label: '重试', icon: 'redo', class: 'btn-primary' },
            refresh: { label: '刷新', icon: 'sync', class: 'btn-primary' },
            wait: { label: '等待', icon: 'clock', class: 'btn-secondary' }
        };

        const config = actionConfig[action];
        if (!config) return '';

        return `
            <button class="btn btn-sm ${config.class} mt-2" data-action="${action}" style="
                padding: 6px 12px;
                font-size: 13px;
                border-radius: 4px;
                border: none;
                cursor: pointer;
            ">
                <i class="fas fa-${config.icon} me-1"></i>
                ${config.label}
            </button>
        `;
    },

    handleNotificationAction(errorInfo, action) {
        switch (action) {
            case 'retry':
                if (errorInfo.retryable) {
                    this.dispatchEvent('action:retry', errorInfo);
                }
                break;
            case 'refresh':
                window.location.reload();
                break;
            case 'wait':
                setTimeout(() => {
                    this.dispatchEvent('action:retry', errorInfo);
                }, 3000);
                break;
            case 'dismiss':
                break;
        }
    },

    dismissNotification(notification) {
        notification.style.animation = 'errorSlideOut 0.3s ease forwards';
        setTimeout(() => {
            if (notification.parentNode) {
                notification.parentNode.removeChild(notification);
            }
        }, 300);
    },

    announceToScreenReader(message) {
        const announcer = document.createElement('div');
        announcer.setAttribute('role', 'status');
        announcer.setAttribute('aria-live', 'polite');
        announcer.setAttribute('aria-atomic', 'true');
        announcer.className = 'sr-only';
        announcer.textContent = message;
        document.body.appendChild(announcer);
        setTimeout(() => announcer.remove(), 1000);
    },

    updateErrorCounts(type) {
        const count = (this.errorCounts.get(type) || 0) + 1;
        this.errorCounts.set(type, count);

        setTimeout(() => {
            const currentCount = this.errorCounts.get(type) || 1;
            this.errorCounts.set(type, currentCount - 1);
            if (this.errorCounts.get(type) === 0) {
                this.errorCounts.delete(type);
            }
        }, this.config.errorWindow);
    },

    pruneErrors() {
        const maxErrors = 100;
        if (this.errors.length > maxErrors) {
            this.errors = this.errors.slice(-maxErrors);
        }
    },

    logError(errorInfo) {
        if (!this.config.enableLogging) return;

        const logPrefix = `[${errorInfo.type.toUpperCase()}]`;
        const logMessage = `${logPrefix} ${errorInfo.message}`;

        if (errorInfo.handlerResult.type === 'error') {
            console.error(logMessage, {
                error: errorInfo.error,
                stack: errorInfo.stack,
                context: errorInfo.context
            });
        } else if (errorInfo.handlerResult.type === 'warning') {
            console.warn(logMessage, errorInfo.context);
        } else {
            console.info(logMessage, errorInfo.context);
        }
    },

    setupErrorBoundaries() {
        const errorBoundaryElements = document.querySelectorAll('[data-error-boundary]');

        errorBoundaryElements.forEach(element => {
            this.wrapWithErrorBoundary(element);
        });
    },

    wrapWithErrorBoundary(element) {
        const originalContent = element.innerHTML;

        const originalErrorHandler = element.onerror;
        element.addEventListener('error', (event) => {
            event.preventDefault();
            this.handleComponentError(event.error || new Error(event.message), element);
        });

        element.addEventListener('unhandledrejection', (event) => {
            event.preventDefault();
            this.handleComponentError(event.reason, element);
        });
    },

    handleComponentError(error, component) {
        const errorInfo = this.handleError(error, { type: 'component', component: component.id || component.className });

        component.innerHTML = `
            <div class="component-error" style="
                padding: 20px;
                text-align: center;
                background: #fff3cd;
                border: 1px solid #ffc107;
                border-radius: 8px;
            ">
                <i class="fas fa-exclamation-triangle text-warning mb-2" style="font-size: 24px;"></i>
                <p style="margin: 0 0 10px; color: #856404;">
                    组件加载失败
                </p>
                <button class="btn btn-sm btn-warning" onclick="location.reload()">
                    刷新页面
                </button>
            </div>
        `;
    },

    createFallbackContent(type) {
        const fallbacks = {
            image: `
                <div class="fallback-image" style="
                    display: flex;
                    align-items: center;
                    justify-content: center;
                    background: #f0f0f0;
                    color: #999;
                    font-size: 14px;
                    min-height: 100px;
                ">
                    <i class="fas fa-image me-2"></i>图片加载失败
                </div>
            `,
            script: `
                <div class="fallback-script" style="
                    padding: 20px;
                    background: #f8f8f8;
                    border: 1px dashed #ccc;
                    border-radius: 4px;
                    text-align: center;
                    color: #666;
                ">
                    <i class="fas fa-exclamation-circle me-2"></i>
                    脚本加载失败，部分功能可能不可用
                </div>
            `,
            component: `
                <div class="fallback-component" style="
                    padding: 20px;
                    text-align: center;
                    background: #f0f0f0;
                    border-radius: 4px;
                ">
                    <i class="fas fa-sync fa-spin me-2"></i>
                    加载中...
                </div>
            `
        };

        return fallbacks[type] || fallbacks.component;
    },

    generateErrorId() {
        return `err_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    },

    delay(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    },

    dispatchEvent(eventName, detail = {}) {
        const event = new CustomEvent(`errorhandler:${eventName}`, { detail });
        document.dispatchEvent(event);
    },

    on(eventName, handler) {
        document.addEventListener(`errorhandler:${eventName}`, (e) => handler(e.detail));
    },

    off(eventName, handler) {
        document.removeEventListener(`errorhandler:${eventName}`, (e) => handler(e.detail));
    },

    getErrors(type = null) {
        if (type) {
            return this.errors.filter(e => e.type === type);
        }
        return this.errors;
    },

    getErrorStats() {
        const stats = {
            total: this.errors.length,
            byType: {},
            recent: this.errors.slice(-10)
        };

        this.errors.forEach(error => {
            stats.byType[error.type] = (stats.byType[error.type] || 0) + 1;
        });

        return stats;
    },

    clearErrors() {
        this.errors = [];
        this.errorCounts.clear();
        this.dispatchEvent('errors:cleared');
    }
};

class HttpError extends Error {
    constructor(status, statusText, url) {
        super(`HTTP ${status}: ${statusText}`);
        this.name = 'HttpError';
        this.status = status;
        this.statusText = statusText;
        this.url = url;
    }
}

if (typeof window !== 'undefined') {
    window.ErrorHandler = ErrorHandler;
    window.HttpError = HttpError;
}
