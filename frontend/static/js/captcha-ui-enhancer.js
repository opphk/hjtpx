/**
 * Captcha UI Enhancer - 验证码UI增强组件
 * 提供更好的用户体验，包括加载状态、错误提示、无障碍支持等
 */

(function() {
    'use strict';

    // 错误消息管理器
    window.CaptchaErrorManager = {
        // 显示错误消息
        show: function(type, title, message, suggestion, actions) {
            var errorEl = document.getElementById(type + 'ErrorMessage');
            if (!errorEl) return null;

            var titleEl = document.getElementById(type + 'ErrorTitle');
            var textEl = document.getElementById(type + 'ErrorText');
            var suggestionTextEl = document.getElementById(type + 'ErrorSuggestionText');
            var suggestionEl = document.getElementById(type + 'ErrorSuggestion');

            if (titleEl) titleEl.textContent = title || '操作失败';
            if (textEl) textEl.textContent = message || '请稍后重试';

            if (suggestionTextEl) {
                suggestionTextEl.textContent = suggestion || '';
                if (suggestionEl) {
                    suggestionEl.style.display = suggestion ? 'block' : 'none';
                }
            }

            // 添加错误恢复操作
            var actionsContainer = errorEl.querySelector('.captcha-error-actions');
            if (actionsContainer) {
                actionsContainer.remove();
            }

            if (actions && actions.length > 0) {
                actionsContainer = document.createElement('div');
                actionsContainer.className = 'captcha-error-actions';

                actions.forEach(function(action) {
                    var btn = document.createElement('button');
                    btn.className = 'captcha-error-action-btn' + (action.secondary ? ' secondary' : '');
                    btn.innerHTML = '<i class="' + (action.icon || 'fas fa-redo') + '"></i>' + action.label;
                    btn.onclick = action.onClick;
                    actionsContainer.appendChild(btn);
                });

                errorEl.querySelector('.captcha-error-message-content').appendChild(actionsContainer);
            }

            errorEl.style.display = 'flex';
            errorEl.setAttribute('role', 'alert');
            errorEl.setAttribute('aria-live', 'assertive');

            return errorEl;
        },

        // 隐藏错误消息
        hide: function(type) {
            var errorEl = document.getElementById(type + 'ErrorMessage');
            if (errorEl) {
                errorEl.style.display = 'none';
            }
        },

        // 获取错误建议
        getSuggestion: function(errorCode) {
            var suggestions = {
                'timeout': {
                    title: '网络连接超时',
                    suggestion: '请检查网络连接后重试',
                    actions: [
                        { label: '重试', icon: 'fas fa-redo', onClick: function() { window.refreshCaptcha(); } }
                    ]
                },
                'network': {
                    title: '网络连接失败',
                    suggestion: '请检查您的网络设置，确保网络畅通',
                    actions: [
                        { label: '重试', icon: 'fas fa-redo', onClick: function() { window.refreshCaptcha(); } },
                        { label: '检查网络', icon: 'fas fa-wifi', onClick: function() { window.open('https://www.google.com', '_blank'); } }
                    ]
                },
                'invalid': {
                    title: '验证码无效',
                    suggestion: '验证码可能已过期或无效，请刷新获取新的验证码',
                    actions: [
                        { label: '刷新验证码', icon: 'fas fa-sync', onClick: function() { window.refreshCaptcha(); } }
                    ]
                },
                'expired': {
                    title: '验证码已过期',
                    suggestion: '验证码已过期，请刷新获取新的验证码',
                    actions: [
                        { label: '获取新验证码', icon: 'fas fa-sync', onClick: function() { window.refreshCaptcha(); } }
                    ]
                },
                'retry': {
                    title: '操作过于频繁',
                    suggestion: '请稍后重试，避免频繁操作',
                    actions: [
                        { label: '稍后重试', icon: 'fas fa-clock', onClick: function() { window.CaptchaToast.info('请5秒后重试'); } }
                    ]
                },
                'server': {
                    title: '服务器错误',
                    suggestion: '服务器暂时不可用，请稍后重试',
                    actions: [
                        { label: '重试', icon: 'fas fa-redo', onClick: function() { window.refreshCaptcha(); } }
                    ]
                },
                'default': {
                    title: '操作失败',
                    suggestion: '请稍后重试，如果问题持续存在请联系支持',
                    actions: [
                        { label: '重试', icon: 'fas fa-redo', onClick: function() { window.refreshCaptcha(); } }
                    ]
                }
            };

            return suggestions[errorCode] || suggestions['default'];
        }
    };

    // 加载状态管理器
    window.CaptchaLoadingManager = {
        show: function(container, type) {
            type = type || 'spinner';

            var loadingEl = document.createElement('div');
            loadingEl.className = 'captcha-loading-' + type;
            loadingEl.setAttribute('role', 'status');
            loadingEl.setAttribute('aria-label', '加载中');

            if (type === 'spinner') {
                loadingEl.innerHTML = '<div class="captcha-loading-spinner"></div>';
            } else if (type === 'dots') {
                loadingEl.innerHTML = '<div class="captcha-loading-dots"><span></span><span></span><span></span></div>';
            } else if (type === 'wave') {
                loadingEl.innerHTML = '<div class="captcha-loading-wave"><div></div><div></div><div></div><div></div><div></div></div>';
            } else if (type === 'pulse') {
                loadingEl.innerHTML = '<div class="captcha-loading-pulse"><div></div><div></div><div></div><div></div><div></div><div></div><div></div><div></div></div>';
            }

            container.appendChild(loadingEl);
            container.classList.add('captcha-loading');

            return loadingEl;
        },

        hide: function(container) {
            var loadingEl = container.querySelector('[role="status"]');
            if (loadingEl) {
                loadingEl.style.opacity = '0';
                loadingEl.style.transform = 'scale(0.8)';
                setTimeout(function() {
                    if (loadingEl.parentNode) {
                        loadingEl.parentNode.removeChild(loadingEl);
                    }
                }, 300);
            }
            container.classList.remove('captcha-loading');
        }
    };

    // 骨架屏管理器
    window.CaptchaSkeletonManager = {
        create: function(container, type) {
            type = type || 'text';

            var skeletonEl = document.createElement('div');
            skeletonEl.className = 'captcha-skeleton';

            if (type === 'text') {
                skeletonEl.className += ' captcha-skeleton-text';
            } else if (type === 'title') {
                skeletonEl.className += ' captcha-skeleton-title';
            } else if (type === 'avatar') {
                skeletonEl.className += ' captcha-skeleton-avatar';
            } else if (type === 'button') {
                skeletonEl.className += ' captcha-skeleton-button';
            } else if (type === 'image') {
                skeletonEl.className += ' captcha-skeleton-image';
            } else if (type === 'paragraph') {
                skeletonEl.className += ' captcha-skeleton-paragraph';
                skeletonEl.innerHTML = '<span></span><span></span><span></span><span></span>';
            }

            container.appendChild(skeletonEl);
            return skeletonEl;
        },

        remove: function(skeletonEl) {
            if (skeletonEl && skeletonEl.parentNode) {
                skeletonEl.parentNode.removeChild(skeletonEl);
            }
        }
    };

    // 性能监控增强
    window.CaptchaPerformanceMonitor = {
        metrics: {
            pageLoad: 0,
            firstContentfulPaint: 0,
            domInteractive: 0,
            largestContentfulPaint: 0
        },

        init: function() {
            var self = this;

            if (window.PerformanceObserver) {
                // 首次内容绘制
                var fcpObserver = new PerformanceObserver(function(list) {
                    for (var entry of list.getEntries()) {
                        if (entry.name === 'first-contentful-paint') {
                            self.metrics.firstContentfulPaint = Math.round(entry.startTime);
                        }
                    }
                });
                fcpObserver.observe({ entryTypes: ['paint'] });

                // 最大内容绘制
                var lcpObserver = new PerformanceObserver(function(list) {
                    var entries = list.getEntries();
                    var lastEntry = entries[entries.length - 1];
                    self.metrics.largestContentfulPaint = Math.round(lastEntry.startTime);
                });
                lcpObserver.observe({ entryTypes: ['largest-contentful-paint'] });

                // DOM交互时间
                if (document.readyState === 'complete') {
                    self.metrics.domInteractive = Math.round(performance.timing.domInteractive - performance.timing.navigationStart);
                } else {
                    window.addEventListener('load', function() {
                        self.metrics.domInteractive = Math.round(performance.timing.domInteractive - performance.timing.navigationStart);
                    });
                }
            }

            window.addEventListener('load', function() {
                self.metrics.pageLoad = Math.round(performance.timing.loadEventEnd - performance.timing.navigationStart);
                self.updateDisplay();
            });
        },

        updateDisplay: function() {
            var pageLoadEl = document.getElementById('pageLoadTime');
            if (pageLoadEl) {
                pageLoadEl.textContent = this.metrics.pageLoad + 'ms';
                pageLoadEl.className = 'metric-value ' + (this.metrics.pageLoad < 2000 ? 'good' : this.metrics.pageLoad < 4000 ? 'warning' : 'bad');
            }
        },

        getMetrics: function() {
            return this.metrics;
        }
    };

    // 初始化性能监控
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', function() {
            window.CaptchaPerformanceMonitor.init();
        });
    } else {
        window.CaptchaPerformanceMonitor.init();
    }

    // 无障碍增强
    window.CaptchaAccessibilityEnhancer = {
        // 焦点管理
        focusFirst: function(container) {
            var focusable = container.querySelectorAll('button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])');
            if (focusable.length > 0) {
                focusable[0].focus();
            }
        },

        // 陷阱焦点（模态框中使用）
        trapFocus: function(container) {
            var focusable = container.querySelectorAll('button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])');
            var firstFocusable = focusable[0];
            var lastFocusable = focusable[focusable.length - 1];

            container.addEventListener('keydown', function(e) {
                if (e.key === 'Tab') {
                    if (e.shiftKey) {
                        if (document.activeElement === firstFocusable) {
                            e.preventDefault();
                            lastFocusable.focus();
                        }
                    } else {
                        if (document.activeElement === lastFocusable) {
                            e.preventDefault();
                            firstFocusable.focus();
                        }
                    }
                }
            });
        },

        // 区域公告
        announce: function(message, priority) {
            priority = priority || 'polite';

            var announcer = document.createElement('div');
            announcer.setAttribute('aria-live', priority);
            announcer.setAttribute('aria-atomic', 'true');
            announcer.className = 'sr-only';
            announcer.textContent = message;
            document.body.appendChild(announcer);

            setTimeout(function() {
                if (announcer.parentNode) {
                    announcer.parentNode.removeChild(announcer);
                }
            }, 1000);

            return announcer;
        }
    };

    // 网络状态监听
    window.CaptchaNetworkMonitor = {
        isOnline: true,

        init: function() {
            var self = this;
            this.isOnline = navigator.onLine;

            window.addEventListener('online', function() {
                self.isOnline = true;
                self.onStatusChange(true);
            });

            window.addEventListener('offline', function() {
                self.isOnline = false;
                self.onStatusChange(false);
            });
        },

        onStatusChange: function(online) {
            var statusEl = document.getElementById('networkStatus');
            if (statusEl) {
                if (!online) {
                    statusEl.innerHTML = '<i class="fas fa-wifi me-1"></i><span>网络连接已断开，请检查网络</span>';
                    statusEl.classList.add('show');
                    statusEl.setAttribute('role', 'alert');
                    statusEl.setAttribute('aria-live', 'assertive');

                    window.CaptchaAccessibilityEnhancer.announce('网络连接已断开', 'assertive');
                } else {
                    statusEl.innerHTML = '<i class="fas fa-wifi me-1"></i><span>网络连接已恢复</span>';
                    statusEl.classList.add('show');

                    setTimeout(function() {
                        statusEl.classList.remove('show');
                    }, 3000);

                    window.CaptchaAccessibilityEnhancer.announce('网络连接已恢复', 'polite');
                }
            }
        }
    };

    // 初始化网络监控
    window.CaptchaNetworkMonitor.init();

    // 导出API
    window.CaptchaUI = Object.assign(window.CaptchaUI || {}, {
        error: window.CaptchaErrorManager,
        loading: window.CaptchaLoadingManager,
        skeleton: window.CaptchaSkeletonManager,
        performance: window.CaptchaPerformanceMonitor,
        accessibility: window.CaptchaAccessibilityEnhancer,
        network: window.CaptchaNetworkMonitor
    });

})();
