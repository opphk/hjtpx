/**
 * 通知系统
 * 提供成功、错误、警告、info等类型的消息通知
 * 支持自动消失、手动关闭、自定义样式等功能
 */

class NotificationManager {
    constructor(options = {}) {
        this.container = null;
        this.notifications = new Map();
        this.idCounter = 0;

        this.defaults = {
            duration: 3000,
            maxVisible: 5,
            position: 'top-right',
            stack: true,
            animation: true,
            closeButton: true,
            progressBar: true,
            pauseOnHover: true,
            ...options
        };

        this.init();
    }

    init() {
        if (this.container) return;

        this.container = document.createElement('div');
        this.container.id = 'notification-container';
        this.container.setAttribute('role', 'region');
        this.container.setAttribute('aria-label', '通知区域');
        this.container.setAttribute('aria-live', 'polite');

        this.applyPosition(this.defaults.position);

        document.body.appendChild(this.container);

        this.injectStyles();
    }

    injectStyles() {
        if (document.getElementById('notification-styles')) return;

        const style = document.createElement('style');
        style.id = 'notification-styles';
        style.textContent = `
            #notification-container {
                position: fixed;
                z-index: 9999;
                display: flex;
                flex-direction: column;
                gap: 10px;
                max-width: 400px;
                width: calc(100% - 20px);
                font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
                pointer-events: none;
            }

            #notification-container.top-right {
                top: 20px;
                right: 20px;
                align-items: flex-end;
            }

            #notification-container.top-left {
                top: 20px;
                left: 20px;
                align-items: flex-start;
            }

            #notification-container.top-center {
                top: 20px;
                left: 50%;
                transform: translateX(-50%);
                align-items: center;
            }

            #notification-container.bottom-right {
                bottom: 20px;
                right: 20px;
                align-items: flex-end;
                flex-direction: column-reverse;
            }

            #notification-container.bottom-left {
                bottom: 20px;
                left: 20px;
                align-items: flex-start;
                flex-direction: column-reverse;
            }

            #notification-container.bottom-center {
                bottom: 20px;
                left: 50%;
                transform: translateX(-50%);
                align-items: center;
                flex-direction: column-reverse;
            }

            .notification {
                display: flex;
                align-items: flex-start;
                gap: 12px;
                padding: 16px;
                border-radius: 8px;
                box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
                background: white;
                pointer-events: auto;
                opacity: 0;
                transform: translateX(100%);
                transition: all 0.3s ease;
                width: 100%;
                position: relative;
                overflow: hidden;
            }

            .notification.show {
                opacity: 1;
                transform: translateX(0);
            }

            .notification.hide {
                opacity: 0;
                transform: translateX(100%);
            }

            #notification-container.top-right .notification.show,
            #notification-container.bottom-right .notification.show {
                transform: translateX(0);
            }

            #notification-container.top-left .notification.show,
            #notification-container.bottom-left .notification.show {
                transform: translateX(0);
            }

            #notification-container.top-center .notification.show,
            #notification-container.bottom-center .notification.show {
                transform: translateY(-20px);
            }

            .notification-icon {
                flex-shrink: 0;
                width: 24px;
                height: 24px;
                display: flex;
                align-items: center;
                justify-content: center;
                font-size: 20px;
            }

            .notification-content {
                flex: 1;
                min-width: 0;
            }

            .notification-title {
                font-weight: 600;
                font-size: 14px;
                color: #1a1a1a;
                margin-bottom: 4px;
                line-height: 1.4;
            }

            .notification-message {
                font-size: 14px;
                color: #666;
                line-height: 1.5;
                word-wrap: break-word;
            }

            .notification-close {
                flex-shrink: 0;
                width: 24px;
                height: 24px;
                border: none;
                background: transparent;
                color: #999;
                cursor: pointer;
                display: flex;
                align-items: center;
                justify-content: center;
                border-radius: 4px;
                transition: all 0.2s ease;
                font-size: 18px;
                line-height: 1;
            }

            .notification-close:hover {
                background: #f5f5f5;
                color: #666;
            }

            .notification-progress {
                position: absolute;
                bottom: 0;
                left: 0;
                height: 3px;
                background: rgba(0, 0, 0, 0.1);
                width: 100%;
            }

            .notification-progress-bar {
                height: 100%;
                transition: width 0.1s linear;
            }

            .notification.success {
                border-left: 4px solid #10b981;
            }

            .notification.success .notification-icon {
                color: #10b981;
            }

            .notification.success .notification-progress-bar {
                background: #10b981;
            }

            .notification.error {
                border-left: 4px solid #ef4444;
            }

            .notification.error .notification-icon {
                color: #ef4444;
            }

            .notification.error .notification-progress-bar {
                background: #ef4444;
            }

            .notification.warning {
                border-left: 4px solid #f59e0b;
            }

            .notification.warning .notification-icon {
                color: #f59e0b;
            }

            .notification.warning .notification-progress-bar {
                background: #f59e0b;
            }

            .notification.info {
                border-left: 4px solid #3b82f6;
            }

            .notification.info .notification-icon {
                color: #3b82f6;
            }

            .notification.info .notification-progress-bar {
                background: #3b82f6;
            }

            @media (max-width: 480px) {
                #notification-container {
                    max-width: none;
                    width: calc(100% - 20px);
                }

                .notification {
                    padding: 12px;
                }
            }

            @keyframes notificationSlideIn {
                from {
                    opacity: 0;
                    transform: translateX(100%);
                }
                to {
                    opacity: 1;
                    transform: translateX(0);
                }
            }

            @keyframes notificationSlideOut {
                from {
                    opacity: 1;
                    transform: translateX(0);
                }
                to {
                    opacity: 0;
                    transform: translateX(100%);
                }
            }

            @keyframes notificationFadeIn {
                from {
                    opacity: 0;
                    transform: translateY(-20px);
                }
                to {
                    opacity: 1;
                    transform: translateY(0);
                }
            }

            @keyframes notificationFadeOut {
                from {
                    opacity: 1;
                    transform: translateY(0);
                }
                to {
                    opacity: 0;
                    transform: translateY(-20px);
                }
            }
        `;

        document.head.appendChild(style);
    }

    applyPosition(position) {
        this.container.className = position;
    }

    createIcon(type) {
        const icons = {
            success: '<i class="fas fa-check-circle" aria-hidden="true"></i>',
            error: '<i class="fas fa-times-circle" aria-hidden="true"></i>',
            warning: '<i class="fas fa-exclamation-triangle" aria-hidden="true"></i>',
            info: '<i class="fas fa-info-circle" aria-hidden="true"></i>'
        };

        return icons[type] || icons.info;
    }

    createNotification(type, message, options = {}) {
        const id = ++this.idCounter;
        const config = {
            ...this.defaults,
            ...options
        };

        const notification = document.createElement('div');
        notification.className = `notification ${type}`;
        notification.id = `notification-${id}`;
        notification.setAttribute('role', 'alert');
        notification.setAttribute('aria-atomic', 'true');

        const icon = this.createIcon(type);

        notification.innerHTML = `
            <div class="notification-icon">${icon}</div>
            <div class="notification-content">
                ${config.title ? `<div class="notification-title">${this.escapeHtml(config.title)}</div>` : ''}
                <div class="notification-message">${this.escapeHtml(message)}</div>
            </div>
            ${config.closeButton ? '<button class="notification-close" aria-label="关闭通知">&times;</button>' : ''}
            ${config.progressBar && config.duration > 0 ? '<div class="notification-progress"><div class="notification-progress-bar"></div></div>' : ''}
        `;

        const data = {
            id,
            type,
            element: notification,
            timer: null,
            progressTimer: null,
            startTime: Date.now(),
            duration: config.duration,
            config
        };

        if (config.closeButton) {
            const closeBtn = notification.querySelector('.notification-close');
            closeBtn.addEventListener('click', () => {
                this.dismiss(id);
            });
        }

        if (config.pauseOnHover && config.duration > 0) {
            notification.addEventListener('mouseenter', () => {
                if (data.timer) {
                    clearTimeout(data.timer);
                    data.timer = null;
                }
                if (data.progressTimer) {
                    clearInterval(data.progressTimer);
                    data.progressTimer = null;
                }
            });

            notification.addEventListener('mouseleave', () => {
                if (data.timer === null && config.duration > 0) {
                    const remaining = data.duration - (Date.now() - data.startTime);
                    if (remaining > 0) {
                        this.startTimer(data, remaining);
                        this.startProgress(data, remaining);
                    }
                }
            });
        }

        this.notifications.set(id, data);
        this.container.appendChild(notification);

        requestAnimationFrame(() => {
            notification.classList.add('show');
        });

        if (config.duration > 0) {
            this.startTimer(data, config.duration);
            this.startProgress(data, config.duration);
        }

        this.enforceMaxVisible();

        return id;
    }

    startTimer(data, duration) {
        data.timer = setTimeout(() => {
            this.dismiss(data.id);
        }, duration);
    }

    startProgress(data, duration) {
        const progressBar = data.element.querySelector('.notification-progress-bar');
        if (!progressBar) return;

        const startWidth = 100;
        const startTime = Date.now();

        data.progressTimer = setInterval(() => {
            const elapsed = Date.now() - data.startTime;
            const remaining = data.duration - elapsed;
            const percent = Math.max(0, (remaining / data.duration) * 100);

            progressBar.style.width = `${percent}%`;

            if (remaining <= 0) {
                clearInterval(data.progressTimer);
                data.progressTimer = null;
            }
        }, 50);
    }

    enforceMaxVisible() {
        if (!this.defaults.stack) return;

        const notifications = Array.from(this.notifications.values());
        if (notifications.length <= this.defaults.maxVisible) return;

        const toRemove = notifications.slice(0, notifications.length - this.defaults.maxVisible);
        toRemove.forEach(data => {
            this.dismiss(data.id);
        });
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    success(message, options = {}) {
        return this.createNotification('success', message, {
            ...options,
            title: options.title || '操作成功'
        });
    }

    error(message, options = {}) {
        return this.createNotification('error', message, {
            ...options,
            title: options.title || '操作失败',
            duration: options.duration !== 0 ? (options.duration || 5000) : 0
        });
    }

    warning(message, options = {}) {
        return this.createNotification('warning', message, {
            ...options,
            title: options.title || '警告'
        });
    }

    info(message, options = {}) {
        return this.createNotification('info', message, {
            ...options,
            title: options.title || '提示'
        });
    }

    dismiss(id) {
        const data = this.notifications.get(id);
        if (!data) return;

        if (data.timer) {
            clearTimeout(data.timer);
            data.timer = null;
        }

        if (data.progressTimer) {
            clearInterval(data.progressTimer);
            data.progressTimer = null;
        }

        data.element.classList.remove('show');
        data.element.classList.add('hide');

        setTimeout(() => {
            if (data.element.parentNode) {
                data.element.parentNode.removeChild(data.element);
            }
            this.notifications.delete(id);
        }, 300);
    }

    dismissAll() {
        const ids = Array.from(this.notifications.keys());
        ids.forEach(id => {
            this.dismiss(id);
        });
    }

    getNotification(id) {
        return this.notifications.get(id);
    }

    getAllNotifications() {
        return Array.from(this.notifications.values());
    }

    setPosition(position) {
        this.defaults.position = position;
        this.applyPosition(position);
    }

    setDefaultDuration(duration) {
        this.defaults.duration = duration;
    }

    setMaxVisible(count) {
        this.defaults.maxVisible = count;
        this.enforceMaxVisible();
    }

    destroy() {
        this.dismissAll();

        if (this.container && this.container.parentNode) {
            this.container.parentNode.removeChild(this.container);
            this.container = null;
        }

        const style = document.getElementById('notification-styles');
        if (style && style.parentNode) {
            style.parentNode.removeChild(style);
        }
    }
}

const Notification = NotificationManager;

const notification = new NotificationManager();

document.addEventListener('DOMContentLoaded', function() {
    window.Notification = Notification;
    window.notification = notification;

    window.showNotification = function(type, message, options) {
        return notification.createNotification(type, message, options);
    };

    window.notify = {
        success: (message, options) => notification.success(message, options),
        error: (message, options) => notification.error(message, options),
        warning: (message, options) => notification.warning(message, options),
        info: (message, options) => notification.info(message, options),
        dismiss: (id) => notification.dismiss(id),
        dismissAll: () => notification.dismissAll()
    };
});
