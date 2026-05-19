class UIXEngine {
    constructor(config = {}) {
        this.config = {
            theme: 'modern',
            breakpoints: {
                xs: { min: 0, max: 575 },
                sm: { min: 576, max: 767 },
                md: { min: 768, max: 991 },
                lg: { min: 992, max: 1199 },
                xl: { min: 1200, max: 1399 },
                xxl: { min: 1400, max: 9999 }
            },
            animations: {
                enabled: true,
                duration: 300,
                easing: 'cubic-bezier(0.4, 0, 0.2, 1)'
            },
            accessibility: {
                highContrast: false,
                largeText: false,
                reduceMotion: false,
                screenReader: false,
                keyboardOnly: false,
                focusIndicators: true,
                colorBlindMode: 'none'
            },
            ...config
        };

        this.currentBreakpoint = this.detectBreakpoint();
        this.theme = {};
        this.initialized = false;
        this.performanceMetrics = {
            fps: 60,
            lastFrame: 0,
            frames: []
        };
    }

    init() {
        if (this.initialized) return;

        this.loadTheme(this.config.theme);
        this.setupResponsiveHandler();
        this.setupAccessibilityFeatures();
        this.startPerformanceMonitoring();
        this.injectStyles();

        this.initialized = true;
        console.log('UIX Engine initialized');
    }

    loadTheme(themeId) {
        const themes = {
            modern: {
                primary: '#c9a96e',
                secondary: '#1a1a2e',
                accent: '#0dcaf0',
                background: '#ffffff',
                surface: '#f8f9fa',
                text: '#1a1a2e',
                border: '#e9ecef',
                success: '#28a745',
                warning: '#ffc107',
                error: '#dc3545',
                gradient: ['#c9a96e', '#0dcaf0'],
                borderRadius: '8px'
            },
            elegant: {
                primary: '#6c5ce7',
                secondary: '#2d3436',
                accent: '#fd79a8',
                background: '#fdfbf7',
                surface: '#ffffff',
                text: '#2d3436',
                border: '#dfe6e9',
                success: '#00b894',
                warning: '#fdcb6e',
                error: '#e17055',
                gradient: ['#6c5ce7', '#fd79a8'],
                borderRadius: '12px'
            },
            minimal: {
                primary: '#0984e3',
                secondary: '#636e72',
                accent: '#00cec9',
                background: '#f5f6fa',
                surface: '#ffffff',
                text: '#2d3436',
                border: '#dcdde1',
                success: '#00b894',
                warning: '#fdcb6e',
                error: '#d63031',
                gradient: ['#0984e3', '#00cec9'],
                borderRadius: '4px'
            },
            vibrant: {
                primary: '#e84393',
                secondary: '#2d3436',
                accent: '#fdcb6e',
                background: '#ffeaa7',
                surface: '#ffffff',
                text: '#2d3436',
                border: '#fab1a0',
                success: '#00b894',
                warning: '#fdcb6e',
                error: '#d63031',
                gradient: ['#e84393', '#fdcb6e'],
                borderRadius: '16px'
            }
        };

        this.theme = themes[themeId] || themes.modern;
        this.applyTheme();
    }

    applyTheme() {
        const root = document.documentElement;
        root.style.setProperty('--ui-primary', this.theme.primary);
        root.style.setProperty('--ui-secondary', this.theme.secondary);
        root.style.setProperty('--ui-accent', this.theme.accent);
        root.style.setProperty('--ui-background', this.theme.background);
        root.style.setProperty('--ui-surface', this.theme.surface);
        root.style.setProperty('--ui-text', this.theme.text);
        root.style.setProperty('--ui-border', this.theme.border);
        root.style.setProperty('--ui-success', this.theme.success);
        root.style.setProperty('--ui-warning', this.theme.warning);
        root.style.setProperty('--ui-error', this.theme.error);
        root.style.setProperty('--ui-gradient', `linear-gradient(135deg, ${this.theme.gradient[0]}, ${this.theme.gradient[1]})`);
        root.style.setProperty('--ui-border-radius', this.theme.borderRadius);
    }

    detectBreakpoint() {
        const width = window.innerWidth;
        const breakpoints = this.config.breakpoints;

        for (const [name, bp] of Object.entries(breakpoints)) {
            if (width >= bp.min && width <= bp.max) {
                return name;
            }
        }
        return 'md';
    }

    setupResponsiveHandler() {
        let resizeTimeout;
        window.addEventListener('resize', () => {
            clearTimeout(resizeTimeout);
            resizeTimeout = setTimeout(() => {
                const newBreakpoint = this.detectBreakpoint();
                if (newBreakpoint !== this.currentBreakpoint) {
                    this.currentBreakpoint = newBreakpoint;
                    this.onBreakpointChange(newBreakpoint);
                }
            }, 100);
        });
    }

    onBreakpointChange(breakpoint) {
        document.body.setAttribute('data-breakpoint', breakpoint);
        this.dispatchEvent('breakpointchange', { breakpoint });
    }

    setupAccessibilityFeatures() {
        const acc = this.config.accessibility;

        if (acc.highContrast) {
            document.body.classList.add('high-contrast');
        }

        if (acc.largeText) {
            document.body.classList.add('large-text');
        }

        if (acc.reduceMotion) {
            document.body.classList.add('reduce-motion');
        }

        if (acc.colorBlindMode !== 'none') {
            document.body.setAttribute('data-colorblind', acc.colorBlindMode);
            this.applyColorBlindFilter(acc.colorBlindMode);
        }

        this.setupKeyboardNavigation();
        this.setupScreenReaderSupport();
    }

    applyColorBlindFilter(mode) {
        const filters = {
            protanopia: 'url(#protanopia)',
            deuteranopia: 'url(#deuteranopia)',
            tritanopia: 'url(#tritanopia)'
        };

        if (filters[mode]) {
            document.body.style.filter = filters[mode];
        }
    }

    setupKeyboardNavigation() {
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Tab') {
                document.body.classList.add('keyboard-nav');
            }
        });

        document.addEventListener('mousedown', () => {
            document.body.classList.remove('keyboard-nav');
        });
    }

    setupScreenReaderSupport() {
        const srButton = document.querySelector('.skip-link, [role="button"]');
        if (srButton) {
            srButton.addEventListener('focus', () => {
                srButton.classList.add('sr-focus');
            });
        }
    }

    injectStyles() {
        if (document.getElementById('uix-styles')) return;

        const style = document.createElement('style');
        style.id = 'uix-styles';
        style.textContent = `
            :root {
                --ui-primary: #c9a96e;
                --ui-secondary: #1a1a2e;
                --ui-accent: #0dcaf0;
                --ui-background: #ffffff;
                --ui-surface: #f8f9fa;
                --ui-text: #1a1a2e;
                --ui-border: #e9ecef;
                --ui-success: #28a745;
                --ui-warning: #ffc107;
                --ui-error: #dc3545;
                --ui-gradient: linear-gradient(135deg, #c9a96e, #0dcaf0);
                --ui-border-radius: 8px;
            }

            .ui-component {
                transition: all var(--ui-animation-duration, 0.3s) var(--ui-animation-easing, cubic-bezier(0.4, 0, 0.2, 1));
            }

            .ui-fade-in {
                animation: uiFadeIn 0.3s ease-out;
            }

            .ui-slide-up {
                animation: uiSlideUp 0.3s ease-out;
            }

            .ui-scale-in {
                animation: uiScaleIn 0.3s ease-out;
            }

            @keyframes uiFadeIn {
                from { opacity: 0; }
                to { opacity: 1; }
            }

            @keyframes uiSlideUp {
                from { transform: translateY(20px); opacity: 0; }
                to { transform: translateY(0); opacity: 1; }
            }

            @keyframes uiScaleIn {
                from { transform: scale(0.9); opacity: 0; }
                to { transform: scale(1); opacity: 1; }
            }

            .ui-shake {
                animation: uiShake 0.5s ease-in-out;
            }

            @keyframes uiShake {
                0%, 100% { transform: translateX(0); }
                25% { transform: translateX(-10px); }
                75% { transform: translateX(10px); }
            }

            .ui-pulse {
                animation: uiPulse 2s ease-in-out infinite;
            }

            @keyframes uiPulse {
                0%, 100% { transform: scale(1); }
                50% { transform: scale(1.05); }
            }

            .high-contrast .ui-component {
                border: 2px solid #000 !important;
                outline: 3px solid #0000ff !important;
            }

            .large-text body {
                font-size: 125% !important;
            }

            .reduce-motion * {
                animation-duration: 0.001ms !important;
                transition-duration: 0.001ms !important;
            }

            .keyboard-nav :focus-visible {
                outline: 3px solid var(--ui-primary);
                outline-offset: 2px;
            }

            .sr-only {
                position: absolute;
                width: 1px;
                height: 1px;
                padding: 0;
                margin: -1px;
                overflow: hidden;
                clip: rect(0, 0, 0, 0);
                white-space: nowrap;
                border-width: 0;
            }

            .ui-glass {
                background: rgba(255, 255, 255, 0.1);
                backdrop-filter: blur(10px);
                border: 1px solid rgba(255, 255, 255, 0.2);
            }

            .ui-shadow-sm {
                box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
            }

            .ui-shadow-md {
                box-shadow: 0 4px 8px rgba(0, 0, 0, 0.15);
            }

            .ui-shadow-lg {
                box-shadow: 0 8px 16px rgba(0, 0, 0, 0.2);
            }
        `;
        document.head.appendChild(style);
    }

    startPerformanceMonitoring() {
        if (!this.config.animations.enabled) return;

        const measureFPS = () => {
            const now = performance.now();
            const delta = now - this.performanceMetrics.lastFrame;
            this.performanceMetrics.lastFrame = now;

            if (delta > 0) {
                const fps = 1000 / delta;
                this.performanceMetrics.frames.push(fps);

                if (this.performanceMetrics.frames.length > 60) {
                    this.performanceMetrics.frames.shift();
                }

                const avgFPS = this.performanceMetrics.frames.reduce((a, b) => a + b, 0) / this.performanceMetrics.frames.length;
                this.performanceMetrics.fps = Math.round(avgFPS);
            }

            requestAnimationFrame(measureFPS);
        };

        requestAnimationFrame(measureFPS);
    }

    animate(element, animation, duration = 300) {
        if (!this.config.animations.enabled || this.config.accessibility.reduceMotion) {
            return Promise.resolve();
        }

        return new Promise(resolve => {
            element.classList.add(`ui-${animation}`);
            setTimeout(() => {
                element.classList.remove(`ui-${animation}`);
                resolve();
            }, duration);
        });
    }

    transition(element, properties, duration = 300) {
        if (!this.config.animations.enabled) {
            Object.assign(element.style, properties);
            return Promise.resolve();
        }

        return new Promise(resolve => {
            const originalStyles = {};
            for (const key of Object.keys(properties)) {
                originalStyles[key] = element.style[key];
            }

            element.style.transition = `all ${duration}ms ${this.config.animations.easing}`;
            Object.assign(element.style, properties);

            setTimeout(() => {
                element.style.transition = '';
                resolve();
            }, duration);
        });
    }

    dispatchEvent(name, detail) {
        window.dispatchEvent(new CustomEvent(`uix:${name}`, { detail }));
    }

    getMetrics() {
        return {
            ...this.performanceMetrics,
            breakpoint: this.currentBreakpoint,
            theme: this.config.theme,
            accessibility: this.config.accessibility
        };
    }

    destroy() {
        document.getElementById('uix-styles')?.remove();
        this.initialized = false;
    }
}

class UIXComponents {
    constructor(engine) {
        this.engine = engine;
    }

    createButton(config = {}) {
        const defaults = {
            text: 'Button',
            type: 'primary',
            size: 'md',
            disabled: false,
            icon: null,
            loading: false,
            onClick: null
        };

        const config_ = { ...defaults, ...config };

        const button = document.createElement('button');
        button.className = `ui-btn ui-btn-${config_.type} ui-btn-${config_.size}`;
        button.disabled = config_.disabled;
        button.setAttribute('role', 'button');

        if (config_.icon) {
            const icon = document.createElement('i');
            icon.className = config_.icon;
            button.appendChild(icon);
        }

        const text = document.createElement('span');
        text.textContent = config_.text;
        button.appendChild(text);

        if (config_.loading) {
            button.classList.add('ui-btn-loading');
            const spinner = document.createElement('span');
            spinner.className = 'ui-spinner';
            button.appendChild(spinner);
        }

        if (config_.onClick) {
            button.addEventListener('click', config_.onClick);
        }

        this.applyStyles(button);
        return button;
    }

    createCard(config = {}) {
        const defaults = {
            title: '',
            content: '',
            footer: '',
            variant: 'default',
            hoverable: false,
            clickable: false
        };

        const config_ = { ...defaults, ...config };

        const card = document.createElement('div');
        card.className = `ui-card ui-card-${config_.variant}`;

        if (config_.hoverable) card.classList.add('ui-card-hoverable');
        if (config_.clickable) card.classList.add('ui-card-clickable');

        if (config_.title) {
            const header = document.createElement('div');
            header.className = 'ui-card-header';
            header.textContent = config_.title;
            card.appendChild(header);
        }

        if (config_.content) {
            const body = document.createElement('div');
            body.className = 'ui-card-body';
            body.textContent = config_.content;
            card.appendChild(body);
        }

        if (config_.footer) {
            const footer = document.createElement('div');
            footer.className = 'ui-card-footer';
            footer.textContent = config_.footer;
            card.appendChild(footer);
        }

        this.applyStyles(card);
        return card;
    }

    createInput(config = {}) {
        const defaults = {
            type: 'text',
            placeholder: '',
            value: '',
            disabled: false,
            error: '',
            label: '',
            helper: '',
            onChange: null
        };

        const config_ = { ...defaults, ...config };

        const wrapper = document.createElement('div');
        wrapper.className = 'ui-input-wrapper';

        if (config_.label) {
            const label = document.createElement('label');
            label.className = 'ui-label';
            label.textContent = config_.label;
            label.setAttribute('for', `input-${Date.now()}`);
            wrapper.appendChild(label);
        }

        const input = document.createElement('input');
        input.type = config_.type;
        input.className = 'ui-input';
        input.placeholder = config_.placeholder;
        input.value = config_.value;
        input.disabled = config_.disabled;
        input.setAttribute('aria-invalid', config_.error ? 'true' : 'false');

        if (config_.error) {
            input.classList.add('ui-input-error');
        }

        if (config_.onChange) {
            input.addEventListener('input', (e) => config_.onChange(e.target.value));
        }

        wrapper.appendChild(input);

        if (config_.error) {
            const error = document.createElement('span');
            error.className = 'ui-error-text';
            error.textContent = config_.error;
            wrapper.appendChild(error);
        } else if (config_.helper) {
            const helper = document.createElement('span');
            helper.className = 'ui-helper-text';
            helper.textContent = config_.helper;
            wrapper.appendChild(helper);
        }

        return wrapper;
    }

    createToast(config = {}) {
        const defaults = {
            type: 'info',
            title: '',
            message: '',
            duration: 5000,
            closable: true
        };

        const config_ = { ...defaults, ...config };

        const icons = {
            success: 'fa-check-circle',
            error: 'fa-exclamation-circle',
            warning: 'fa-exclamation-triangle',
            info: 'fa-info-circle'
        };

        const toast = document.createElement('div');
        toast.className = `ui-toast ui-toast-${config_.type}`;

        const icon = document.createElement('i');
        icon.className = `fas ${icons[config_.type]}`;
        toast.appendChild(icon);

        const content = document.createElement('div');
        content.className = 'ui-toast-content';

        if (config_.title) {
            const title = document.createElement('div');
            title.className = 'ui-toast-title';
            title.textContent = config_.title;
            content.appendChild(title);
        }

        const message = document.createElement('div');
        message.className = 'ui-toast-message';
        message.textContent = config_.message;
        content.appendChild(message);

        toast.appendChild(content);

        if (config_.closable) {
            const close = document.createElement('button');
            close.className = 'ui-toast-close';
            close.innerHTML = '<i class="fas fa-times"></i>';
            close.setAttribute('aria-label', 'Close');
            close.addEventListener('click', () => this.removeToast(toast));
            toast.appendChild(close);
        }

        document.body.appendChild(toast);

        requestAnimationFrame(() => {
            toast.classList.add('ui-toast-show');
        });

        if (config_.duration > 0) {
            setTimeout(() => this.removeToast(toast), config_.duration);
        }

        return toast;
    }

    removeToast(toast) {
        toast.classList.remove('ui-toast-show');
        setTimeout(() => toast.remove(), 300);
    }

    createProgress(config = {}) {
        const defaults = {
            value: 0,
            max: 100,
            showLabel: true,
            variant: 'default',
            animated: true
        };

        const config_ = { ...defaults, ...config };

        const wrapper = document.createElement('div');
        wrapper.className = `ui-progress ui-progress-${config_.variant}`;

        const track = document.createElement('div');
        track.className = 'ui-progress-track';

        const fill = document.createElement('div');
        fill.className = 'ui-progress-fill';
        const percentage = (config_.value / config_.max) * 100;
        fill.style.width = `${percentage}%`;

        if (config_.animated) {
            fill.classList.add('ui-progress-animated');
        }

        track.appendChild(fill);
        wrapper.appendChild(track);

        if (config_.showLabel) {
            const label = document.createElement('span');
            label.className = 'ui-progress-label';
            label.textContent = `${Math.round(percentage)}%`;
            wrapper.appendChild(label);
        }

        return wrapper;
    }

    applyStyles(element) {
        element.style.setProperty('--ui-primary', this.engine.theme.primary);
        element.style.setProperty('--ui-secondary', this.engine.theme.secondary);
        element.style.setProperty('--ui-border-radius', this.engine.theme.borderRadius);
    }
}

window.UIXEngine = UIXEngine;
window.UIXComponents = UIXComponents;
