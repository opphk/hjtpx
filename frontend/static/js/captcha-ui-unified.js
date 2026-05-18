/**
 * HJTPX Captcha UI Enhancer
 * Provides unified toast notifications, loading animations, and UI interactions
 */
(function() {
    'use strict';

    const ToastManager = {
        container: null,
        init: function() {
            if (!this.container) {
                this.container = document.createElement('div');
                this.container.className = 'captcha-toast-container';
                this.container.id = 'captchaToastContainer';
                document.body.appendChild(this.container);
            }
            return this.container;
        },
        show: function(options) {
            const defaults = {
                type: 'info',
                title: '',
                message: '',
                duration: 5000,
                closable: true,
                onClose: null
            };
            const config = Object.assign({}, defaults, options);

            this.init();

            const icons = {
                success: 'fa-check-circle',
                error: 'fa-exclamation-circle',
                warning: 'fa-exclamation-triangle',
                info: 'fa-info-circle'
            };

            const toast = document.createElement('div');
            toast.className = `captcha-toast ${config.type}`;
            toast.innerHTML = `
                <div class="captcha-toast-icon">
                    <i class="fas ${icons[config.type] || icons.info}"></i>
                </div>
                <div class="captcha-toast-content">
                    ${config.title ? `<div class="captcha-toast-title">${config.title}</div>` : ''}
                    ${config.message ? `<div class="captcha-toast-message">${config.message}</div>` : ''}
                </div>
                ${config.closable ? `
                <button class="captcha-toast-close" aria-label="Close">
                    <i class="fas fa-times"></i>
                </button>
                ` : ''}
            `;

            if (config.closable) {
                const closeBtn = toast.querySelector('.captcha-toast-close');
                closeBtn.addEventListener('click', () => this.remove(toast, config.onClose));
            }

            this.container.appendChild(toast);

            if (config.duration > 0) {
                setTimeout(() => this.remove(toast, config.onClose), config.duration);
            }

            return toast;
        },
        remove: function(toast, callback) {
            if (!toast || !toast.parentNode) return;

            toast.classList.add('removing');
            setTimeout(() => {
                if (toast.parentNode) {
                    toast.parentNode.removeChild(toast);
                }
                if (typeof callback === 'function') {
                    callback();
                }
            }, 300);
        },
        success: function(message, title, duration) {
            return this.show({ type: 'success', message, title, duration });
        },
        error: function(message, title, duration) {
            return this.show({ type: 'error', message, title, duration: duration || 8000 });
        },
        warning: function(message, title, duration) {
            return this.show({ type: 'warning', message, title, duration });
        },
        info: function(message, title, duration) {
            return this.show({ type: 'info', message, title, duration });
        }
    };

    const LoadingManager = {
        overlay: null,
        show: function(message) {
            if (!this.overlay) {
                this.overlay = document.createElement('div');
                this.overlay.className = 'captcha-loading-overlay';
                this.overlay.innerHTML = `
                    <div class="captcha-loading-container text-center">
                        <div class="captcha-loading-spinner"></div>
                        <div class="loading-message mt-3">${message || 'Loading...'}</div>
                    </div>
                `;
                document.body.appendChild(this.overlay);
            }

            const msgEl = this.overlay.querySelector('.loading-message');
            if (msgEl) msgEl.textContent = message || 'Loading...';

            requestAnimationFrame(() => {
                this.overlay.classList.add('show');
            });
        },
        hide: function() {
            if (this.overlay) {
                this.overlay.classList.remove('show');
            }
        },
        updateMessage: function(message) {
            if (this.overlay) {
                const msgEl = this.overlay.querySelector('.loading-message');
                if (msgEl) msgEl.textContent = message;
            }
        }
    };

    const AnimationManager = {
        success: function(element) {
            if (!element) return;
            element.classList.add('captcha-success-animation');
            setTimeout(() => {
                element.classList.remove('captcha-success-animation');
            }, 600);
        },
        error: function(element) {
            if (!element) return;
            element.classList.add('captcha-error-animation');
            setTimeout(() => {
                element.classList.remove('captcha-error-animation');
            }, 500);
        },
        confetti: function() {
            const container = document.createElement('div');
            container.className = 'captcha-confetti';
            document.body.appendChild(container);

            const colors = ['#c9a96e', '#28a745', '#0dcaf0', '#ffc107', '#dc3545'];
            for (let i = 0; i < 50; i++) {
                const piece = document.createElement('div');
                piece.className = 'captcha-confetti-piece';
                piece.style.left = Math.random() * 100 + '%';
                piece.style.backgroundColor = colors[Math.floor(Math.random() * colors.length)];
                piece.style.animationDelay = Math.random() * 2 + 's';
                piece.style.animationDuration = (2 + Math.random() * 2) + 's';
                container.appendChild(piece);
            }

            setTimeout(() => container.remove(), 4000);
        },
        refresh: function(button) {
            if (!button) return;
            const icon = button.querySelector('i') || button;
            icon.classList.add('fa-spin');
            setTimeout(() => {
                icon.classList.remove('fa-spin');
            }, 1000);
        }
    };

    const CaptchaUI = {
        toast: ToastManager,
        loading: LoadingManager,
        animation: AnimationManager,

        init: function() {
            console.log('Captcha UI Enhancer initialized');
        },

        showSuccessToast: function(message, title) {
            return ToastManager.success(message, title);
        },

        showErrorToast: function(message, title) {
            return ToastManager.error(message, title);
        },

        showWarningToast: function(message, title) {
            return ToastManager.warning(message, title);
        },

        showInfoToast: function(message, title) {
            return ToastManager.info(message, title);
        },

        showLoading: function(message) {
            LoadingManager.show(message);
        },

        hideLoading: function() {
            LoadingManager.hide();
        },

        animateSuccess: function(element) {
            AnimationManager.success(element);
        },

        animateError: function(element) {
            AnimationManager.error(element);
        },

        showConfetti: function() {
            AnimationManager.confetti();
        },

        refreshButton: function(button) {
            AnimationManager.refresh(button);
        },

        handleCaptchaSuccess: function(element) {
            AnimationManager.success(element);
            AnimationManager.confetti();
            ToastManager.success('Verification successful!', 'Success');
        },

        handleCaptchaError: function(element, message) {
            AnimationManager.error(element);
            ToastManager.error(message || 'Verification failed, please try again', 'Error');
        },

        createLoadingSpinner: function(size) {
            const spinner = document.createElement('div');
            spinner.className = 'captcha-loading-spinner';
            if (size) {
                spinner.style.width = size + 'px';
                spinner.style.height = size + 'px';
            }
            return spinner;
        },

        createResultBanner: function(type, message) {
            const banner = document.createElement('div');
            banner.className = `captcha-result-banner ${type}`;
            banner.innerHTML = `
                <i class="fas ${type === 'success' ? 'fa-check-circle' : 'fa-times-circle'}"></i>
                <span>${message}</span>
            `;
            return banner;
        },

        createHint: function(type, title, description) {
            const hint = document.createElement('div');
            hint.className = `captcha-hint ${type}`;
            hint.innerHTML = `
                <div class="captcha-hint-icon">
                    <i class="fas ${type === 'success' ? 'fa-check-circle' : 'fa-exclamation-circle'}"></i>
                </div>
                <div class="captcha-hint-text">
                    ${title ? `<div class="captcha-hint-title">${title}</div>` : ''}
                    ${description ? `<div class="captcha-hint-desc">${description}</div>` : ''}
                </div>
            `;
            return hint;
        },

        createProgressBar: function() {
            const container = document.createElement('div');
            container.className = 'captcha-progress-container';
            container.innerHTML = `
                <div class="captcha-progress-bar">
                    <div class="captcha-progress-fill" style="width: 0%"></div>
                </div>
                <div class="captcha-progress-text">Loading...</div>
            `;
            return {
                element: container,
                setProgress: function(percent, text) {
                    const fill = container.querySelector('.captcha-progress-fill');
                    const label = container.querySelector('.captcha-progress-text');
                    if (fill) fill.style.width = percent + '%';
                    if (label) label.textContent = text || percent + '%';
                },
                show: function() {
                    container.style.display = 'block';
                },
                hide: function() {
                    container.style.display = 'none';
                }
            };
        }
    };

    if (typeof window !== 'undefined') {
        window.CaptchaUI = CaptchaUI;
        window.ToastManager = ToastManager;
        window.LoadingManager = LoadingManager;
        window.AnimationManager = AnimationManager;
    }

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = CaptchaUI;
    }

    document.addEventListener('DOMContentLoaded', function() {
        CaptchaUI.init();
    });

})();
