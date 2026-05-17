const ValidationFlow = {
    config: {
        apiBase: '/api/v1',
        maxRetries: 3,
        retryDelay: 1000,
        timeout: 10000,
        cacheEnabled: true,
        cacheTTL: 300000
    },
    cache: new Map(),
    pendingRequests: new Map(),
    state: {
        currentStep: 0,
        totalSteps: 0,
        isValidating: false,
        validationHistory: []
    },

    async init(options = {}) {
        this.config = { ...this.config, ...options };
        this.setupInterceptors();
        this.setupProgressTracking();
        this.setupStepOptimization();
        this.setupErrorOptimization();
        this.setupKeyboardShortcuts();
        return this;
    },

    setupInterceptors() {
        const originalFetch = window.fetch;
        const self = this;

        window.fetch = async function(...args) {
            const [url, options = {}] = args;
            const cacheKey = self.generateCacheKey(url, options);

            if (self.config.cacheEnabled && options.method === 'GET') {
                const cached = self.getFromCache(cacheKey);
                if (cached) {
                    self.dispatchEvent('cache:hit', { url, data: cached });
                    return cached;
                }
            }

            const requestId = self.generateRequestId();
            self.pendingRequests.set(requestId, { url, startTime: Date.now() });

            try {
                const controller = new AbortController();
                const timeoutId = setTimeout(() => controller.abort(), self.config.timeout);
                options.signal = controller.signal;

                self.dispatchEvent('request:start', { requestId, url });
                const response = await originalFetch(...[url, options]);
                clearTimeout(timeoutId);

                if (!response.ok) {
                    throw new Error(`HTTP ${response.status}: ${response.statusText}`);
                }

                const data = await response.clone().json();

                if (self.config.cacheEnabled && options.method === 'GET' && data) {
                    self.setToCache(cacheKey, data);
                }

                self.pendingRequests.delete(requestId);
                self.dispatchEvent('request:complete', {
                    requestId,
                    url,
                    duration: Date.now() - self.pendingRequests.get(requestId)?.startTime
                });

                return data;
            } catch (error) {
                self.pendingRequests.delete(requestId);
                self.dispatchEvent('request:error', { requestId, url, error: error.message });
                throw error;
            }
        };
    },

    setupProgressTracking() {
        this.state.progressCallbacks = [];
    },

    onProgress(callback) {
        this.state.progressCallbacks.push(callback);
    },

    updateProgress(current, total, message = '') {
        this.state.currentStep = current;
        this.state.totalSteps = total;
        const percentage = Math.round((current / total) * 100);

        this.state.progressCallbacks.forEach(cb => cb({
            current,
            total,
            percentage,
            message
        }));

        this.dispatchEvent('progress:update', { current, total, percentage, message });
        return percentage;
    },

    setupStepOptimization() {
        this.stepDefinitions = new Map();
        this.stepValidators = new Map();
    },

    registerStep(stepId, config) {
        this.stepDefinitions.set(stepId, {
            id: stepId,
            name: config.name || stepId,
            description: config.description || '',
            validate: config.validate || (() => Promise.resolve(true)),
            onEnter: config.onEnter || (() => {}),
            onExit: config.onExit || (() => {}),
            skipIf: config.skipIf || (() => false),
            priority: config.priority || 0,
            timeout: config.timeout || 30000
        });
        return this;
    },

    registerStepValidator(stepId, validator) {
        this.stepValidators.set(stepId, validator);
    },

    async executeFlow(steps, context = {}) {
        const executionContext = {
            ...context,
            errors: [],
            warnings: [],
            startTime: Date.now(),
            stepResults: new Map()
        };

        const sortedSteps = steps
            .map(id => this.stepDefinitions.get(id))
            .filter(Boolean)
            .sort((a, b) => a.priority - b.priority);

        const totalSteps = sortedSteps.length;
        let currentStep = 0;

        this.state.isValidating = true;
        this.dispatchEvent('flow:start', { totalSteps });

        for (const step of sortedSteps) {
            currentStep++;
            this.updateProgress(currentStep, totalSteps, `执行步骤: ${step.name}`);

            try {
                const shouldSkip = await this.evaluateCondition(step.skipIf, executionContext);
                if (shouldSkip) {
                    executionContext.warnings.push(`步骤 ${step.name} 已跳过`);
                    this.dispatchEvent('step:skip', { step: step.id });
                    continue;
                }

                step.onEnter(executionContext);
                this.dispatchEvent('step:enter', { step: step.id, stepName: step.name });

                const validator = this.stepValidators.get(step.id) || step.validate;
                const startTime = Date.now();
                const result = await this.executeWithTimeout(
                    validator(executionContext),
                    step.timeout
                );

                const duration = Date.now() - startTime;
                executionContext.stepResults.set(step.id, {
                    success: true,
                    duration,
                    result
                });

                step.onExit(executionContext, result);
                this.dispatchEvent('step:complete', {
                    step: step.id,
                    duration,
                    result
                });

                if (!result) {
                    executionContext.errors.push(`步骤 ${step.name} 验证失败`);
                }
            } catch (error) {
                const errorInfo = {
                    step: step.id,
                    error: error.message,
                    timestamp: Date.now()
                };
                executionContext.errors.push(errorInfo);
                executionContext.stepResults.set(step.id, {
                    success: false,
                    error: error.message
                });

                this.dispatchEvent('step:error', errorInfo);
                this.handleStepError(step, error, executionContext);
            }
        }

        this.state.isValidating = false;
        this.state.validationHistory.push({
            timestamp: Date.now(),
            duration: Date.now() - executionContext.startTime,
            success: executionContext.errors.length === 0,
            errors: executionContext.errors.length,
            steps: executionContext.stepResults.size
        });

        this.updateProgress(totalSteps, totalSteps, '流程完成');
        this.dispatchEvent('flow:complete', executionContext);

        return {
            success: executionContext.errors.length === 0,
            context: executionContext,
            duration: Date.now() - executionContext.startTime
        };
    },

    async executeWithTimeout(promise, timeout) {
        return Promise.race([
            promise,
            new Promise((_, reject) =>
                setTimeout(() => reject(new Error(`操作超时 (${timeout}ms)`)), timeout)
            )
        ]);
    },

    async evaluateCondition(condition, context) {
        if (typeof condition === 'function') {
            return condition(context);
        }
        return !!condition;
    },

    handleStepError(step, error, context) {
        const retryStrategy = this.getRetryStrategy(step.id);
        if (retryStrategy) {
            this.scheduleRetry(step, error, context, retryStrategy);
        }
    },

    getRetryStrategy(stepId) {
        return {
            maxRetries: this.config.maxRetries,
            retryDelay: this.config.retryDelay,
            backoffMultiplier: 2
        };
    },

    async scheduleRetry(step, error, context, strategy) {
        let retryCount = context.retryCount || 0;

        if (retryCount >= strategy.maxRetries) {
            this.dispatchEvent('retry:exhausted', {
                step: step.id,
                error,
                attempts: retryCount
            });
            return;
        }

        const delay = strategy.retryDelay * Math.pow(strategy.backoffMultiplier, retryCount);
        context.retryCount = retryCount + 1;

        await new Promise(resolve => setTimeout(resolve, delay));

        this.dispatchEvent('retry:attempt', {
            step: step.id,
            attempt: context.retryCount,
            delay
        });
    },

    setupErrorOptimization() {
        this.errorHandlers = new Map();
        this.errorPatterns = new Map();
        this.registerDefaultErrorHandlers();
    },

    registerDefaultErrorHandlers() {
        this.registerErrorHandler('network', (error) => ({
            title: '网络错误',
            message: '网络连接失败，请检查网络设置',
            action: 'retry',
            severity: 'error'
        }));

        this.registerErrorHandler('timeout', (error) => ({
            title: '请求超时',
            message: '服务器响应超时，请稍后重试',
            action: 'retry',
            severity: 'warning'
        }));

        this.registerErrorHandler('validation', (error) => ({
            title: '验证失败',
            message: error.message || '验证未通过，请重试',
            action: 'refresh',
            severity: 'error'
        }));

        this.registerErrorHandler('server', (error) => ({
            title: '服务器错误',
            message: '服务器繁忙，请稍后再试',
            action: 'wait',
            severity: 'error'
        }));
    },

    registerErrorHandler(type, handler) {
        this.errorHandlers.set(type, handler);
    },

    registerErrorPattern(pattern, type) {
        this.errorPatterns.set(pattern, type);
    },

    getErrorInfo(error) {
        const errorType = this.classifyError(error);
        const handler = this.errorHandlers.get(errorType);

        if (handler) {
            return handler(error);
        }

        return {
            title: '操作失败',
            message: error.message || '发生了未知错误',
            action: 'retry',
            severity: 'error'
        };
    },

    classifyError(error) {
        const message = (error.message || '').toLowerCase();

        for (const [pattern, type] of this.errorPatterns) {
            if (message.includes(pattern)) {
                return type;
            }
        }

        if (message.includes('network') || message.includes('fetch')) {
            return 'network';
        }
        if (message.includes('timeout')) {
            return 'timeout';
        }
        if (message.includes('valid') || message.includes('verify')) {
            return 'validation';
        }
        if (message.includes('500') || message.includes('503')) {
            return 'server';
        }

        return 'unknown';
    },

    showOptimizedError(error, container) {
        const errorInfo = this.getErrorInfo(error);
        const errorEl = document.createElement('div');
        errorEl.className = `validation-error validation-error-${errorInfo.severity}`;
        errorEl.setAttribute('role', 'alert');
        errorEl.setAttribute('aria-live', 'polite');

        errorEl.innerHTML = `
            <div class="validation-error-icon">
                <i class="fas fa-${this.getSeverityIcon(errorInfo.severity)}" aria-hidden="true"></i>
            </div>
            <div class="validation-error-content">
                <div class="validation-error-title">${errorInfo.title}</div>
                <div class="validation-error-message">${errorInfo.message}</div>
            </div>
            ${errorInfo.action ? `
                <button class="validation-error-action btn btn-sm btn-${errorInfo.severity === 'warning' ? 'warning' : 'danger'}" data-action="${errorInfo.action}">
                    <i class="fas fa-${this.getActionIcon(errorInfo.action)} me-1"></i>
                    ${this.getActionLabel(errorInfo.action)}
                </button>
            ` : ''}
            <button class="validation-error-close" aria-label="关闭">
                <i class="fas fa-times" aria-hidden="true"></i>
            </button>
        `;

        if (container) {
            container.appendChild(errorEl);
            this.announceToScreenReader(errorInfo.message);
        }

        const closeBtn = errorEl.querySelector('.validation-error-close');
        closeBtn?.addEventListener('click', () => {
            errorEl.classList.add('validation-error-hiding');
            setTimeout(() => errorEl.remove(), 300);
        });

        const actionBtn = errorEl.querySelector('.validation-error-action');
        actionBtn?.addEventListener('click', () => {
            this.handleErrorAction(errorInfo.action, error);
        });

        setTimeout(() => {
            errorEl.classList.add('validation-error-visible');
        }, 10);

        if (errorInfo.severity !== 'warning') {
            setTimeout(() => {
                if (errorEl.parentNode) {
                    errorEl.classList.add('validation-error-hiding');
                    setTimeout(() => errorEl.remove(), 300);
                }
            }, 5000);
        }

        return errorEl;
    },

    getSeverityIcon(severity) {
        const icons = {
            error: 'exclamation-circle',
            warning: 'exclamation-triangle',
            info: 'info-circle',
            success: 'check-circle'
        };
        return icons[severity] || icons.error;
    },

    getActionIcon(action) {
        const icons = {
            retry: 'redo',
            refresh: 'sync',
            wait: 'clock',
            dismiss: 'times'
        };
        return icons[action] || icons.retry;
    },

    getActionLabel(action) {
        const labels = {
            retry: '重试',
            refresh: '刷新',
            wait: '等待',
            dismiss: '关闭'
        };
        return labels[action] || labels.retry;
    },

    handleErrorAction(action, error) {
        switch (action) {
            case 'retry':
                this.dispatchEvent('action:retry', { error });
                break;
            case 'refresh':
                window.location.reload();
                break;
            case 'wait':
                setTimeout(() => {
                    this.dispatchEvent('action:retry', { error });
                }, 3000);
                break;
        }
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

    setupKeyboardShortcuts() {
        document.addEventListener('keydown', (e) => {
            if (e.ctrlKey || e.metaKey) {
                switch (e.key) {
                    case 'r':
                        e.preventDefault();
                        this.dispatchEvent('shortcut:retry');
                        break;
                    case 'Enter':
                        e.preventDefault();
                        this.dispatchEvent('shortcut:submit');
                        break;
                }
            }

            if (e.key === 'Escape') {
                this.dispatchEvent('shortcut:cancel');
            }
        });
    },

    generateCacheKey(url, options) {
        return `${url}_${JSON.stringify(options || {})}`;
    },

    getFromCache(key) {
        const cached = this.cache.get(key);
        if (cached && Date.now() - cached.timestamp < this.config.cacheTTL) {
            return cached.data;
        }
        this.cache.delete(key);
        return null;
    },

    setToCache(key, data) {
        this.cache.set(key, {
            data,
            timestamp: Date.now()
        });

        setTimeout(() => {
            this.cache.delete(key);
        }, this.config.cacheTTL);
    },

    generateRequestId() {
        return `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    },

    dispatchEvent(eventName, detail = {}) {
        const event = new CustomEvent(`validation:${eventName}`, { detail });
        document.dispatchEvent(event);
    },

    on(eventName, handler) {
        document.addEventListener(`validation:${eventName}`, (e) => handler(e.detail));
    },

    off(eventName, handler) {
        document.removeEventListener(`validation:${eventName}`, (e) => handler(e.detail));
    },

    clearCache() {
        this.cache.clear();
        this.dispatchEvent('cache:clear');
    },

    getStats() {
        return {
            cacheSize: this.cache.size,
            pendingRequests: this.pendingRequests.size,
            validationHistory: this.state.validationHistory.slice(-10),
            totalValidations: this.state.validationHistory.length
        };
    }
};

if (typeof window !== 'undefined') {
    window.ValidationFlow = ValidationFlow;
}
