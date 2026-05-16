/**
 * 主脚本 - 用户端全局管理
 * 包含：模块初始化、事件处理、性能监控、用户行为追踪、错误处理
 */

(function() {
    'use strict';

    const App = {
        version: '1.0.0',
        modules: {},
        initialized: false,
        config: {
            apiBase: '/api/v1',
            debug: false,
            enableAnalytics: true,
            enablePerformance: true,
            scrollThreshold: 100
        },

        init: function(options = {}) {
            if (this.initialized) {
                console.warn('App已初始化，请勿重复调用');
                return this;
            }

            this.config = { ...this.config, ...options };

            this.performance.start('app-init');

            this.initErrorHandling();
            this.initModules();
            this.initNavigation();
            this.initAnimations();
            this.initAccessibility();
            this.initPerformanceMonitoring();
            this.initAnalytics();

            this.initialized = true;

            this.performance.end('app-init');
            console.log(`[App] 初始化完成，耗时: ${this.performance.getEntries().find(e => e.name === 'app-init')?.duration.toFixed(2)}ms`);
        },

        initErrorHandling: function() {
            window.addEventListener('error', (event) => {
                this.handleError(event.error, 'Uncaught Error');
            });

            window.addEventListener('unhandledrejection', (event) => {
                this.handleError(event.reason, 'Unhandled Promise Rejection');
            });
        },

        handleError: function(error, source) {
            const errorInfo = {
                message: error?.message || '未知错误',
                stack: error?.stack || '',
                source: source,
                url: window.location.href,
                timestamp: new Date().toISOString(),
                userAgent: navigator.userAgent
            };

            if (this.config.debug) {
                console.error('[Error]', errorInfo);
            }

            if (typeof window.__errorCollector !== 'undefined') {
                window.__errorCollector.push(errorInfo);
            }
        },

        initModules: function() {
            this.modules = {
                utils: window.Utils,
                notification: window.notification,
                captcha: window.Captcha
            };
        },

        initNavigation: function() {
            const nav = document.querySelector('nav');
            if (!nav) return;

            const navLinks = nav.querySelectorAll('.nav-link');
            const currentPath = window.location.pathname;

            navLinks.forEach(link => {
                const href = link.getAttribute('href');
                if (href === currentPath || (href === '/' && currentPath === '/')) {
                    link.classList.add('active');
                    link.setAttribute('aria-current', 'page');
                }

                link.addEventListener('click', (e) => {
                    navLinks.forEach(l => {
                        l.classList.remove('active');
                        l.removeAttribute('aria-current');
                    });
                    link.classList.add('active');
                    link.setAttribute('aria-current', 'page');
                });
            });

            const mobileMenuBtn = nav.querySelector('.navbar-toggler');
            const mobileMenu = nav.querySelector('.navbar-collapse');

            if (mobileMenuBtn && mobileMenu) {
                mobileMenuBtn.addEventListener('click', () => {
                    const isExpanded = mobileMenuBtn.getAttribute('aria-expanded') === 'true';
                    mobileMenuBtn.setAttribute('aria-expanded', !isExpanded);
                    mobileMenu.classList.toggle('show');
                });
            }
        },

        initAnimations: function() {
            this.initScrollAnimations();
            this.initButtonAnimations();
            this.initCounterAnimations();
            this.initLazyLoading();
        },

        initScrollAnimations: function() {
            const observerOptions = {
                root: null,
                rootMargin: '0px',
                threshold: 0.1
            };

            const observer = new IntersectionObserver((entries) => {
                entries.forEach(entry => {
                    if (entry.isIntersecting) {
                        entry.target.classList.add('animate-in');
                        observer.unobserve(entry.target);
                    }
                });
            }, observerOptions);

            document.querySelectorAll('.animate-on-scroll').forEach(el => {
                observer.observe(el);
            });
        },

        initButtonAnimations: function() {
            document.querySelectorAll('.btn').forEach(btn => {
                btn.addEventListener('mouseenter', function() {
                    this.style.transform = 'translateY(-2px)';
                    this.style.boxShadow = '0 4px 12px rgba(0, 0, 0, 0.15)';
                });

                btn.addEventListener('mouseleave', function() {
                    this.style.transform = 'translateY(0)';
                    this.style.boxShadow = '';
                });

                btn.addEventListener('mousedown', function() {
                    this.style.transform = 'translateY(0) scale(0.98)';
                });

                btn.addEventListener('mouseup', function() {
                    this.style.transform = 'translateY(-2px)';
                });
            });
        },

        initCounterAnimations: function() {
            const counters = document.querySelectorAll('[data-counter]');
            if (counters.length === 0) return;

            const observerOptions = {
                threshold: 0.5
            };

            const observer = new IntersectionObserver((entries) => {
                entries.forEach(entry => {
                    if (entry.isIntersecting) {
                        this.animateCounter(entry.target);
                        observer.unobserve(entry.target);
                    }
                });
            }, observerOptions);

            counters.forEach(counter => {
                observer.observe(counter);
            });
        },

        animateCounter: function(element) {
            const target = parseInt(element.dataset.counter);
            const duration = 2000;
            const startTime = performance.now();

            const animate = (currentTime) => {
                const elapsed = currentTime - startTime;
                const progress = Math.min(elapsed / duration, 1);

                const easeOutQuart = 1 - Math.pow(1 - progress, 4);
                const current = Math.floor(target * easeOutQuart);

                element.textContent = current.toLocaleString();

                if (progress < 1) {
                    requestAnimationFrame(animate);
                } else {
                    element.textContent = target.toLocaleString();
                }
            };

            requestAnimationFrame(animate);
        },

        initLazyLoading: function() {
            if ('IntersectionObserver' in window) {
                const imageObserver = new IntersectionObserver((entries) => {
                    entries.forEach(entry => {
                        if (entry.isIntersecting) {
                            const img = entry.target;
                            const src = img.dataset.src;
                            if (src) {
                                img.src = src;
                                img.removeAttribute('data-src');
                                imageObserver.unobserve(img);
                            }
                        }
                    });
                });

                document.querySelectorAll('img[data-src]').forEach(img => {
                    imageObserver.observe(img);
                });
            }
        },

        initAccessibility: function() {
            this.initKeyboardNavigation();
            this.initFocusManagement();
            this.initSkipLinks();
            this.initScreenReaderAnnouncements();
        },

        initKeyboardNavigation: function() {
            document.addEventListener('keydown', (e) => {
                if (e.key === 'Tab') {
                    document.body.classList.add('keyboard-nav');
                }
            });

            document.addEventListener('mousedown', () => {
                document.body.classList.remove('keyboard-nav');
            });

            document.querySelectorAll('a, button, input, select, textarea').forEach(el => {
                el.addEventListener('focus', function() {
                    this.classList.add('focus-visible');
                });
                el.addEventListener('blur', function() {
                    this.classList.remove('focus-visible');
                });
            });
        },

        initFocusManagement: function() {
            const focusableElements = 'a[href], area[href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), button:not([disabled]), [tabindex="0"]';

            document.querySelectorAll('.modal').forEach(modal => {
                modal.addEventListener('keydown', (e) => {
                    if (e.key === 'Escape') {
                        const closeBtn = modal.querySelector('[data-bs-dismiss="modal"], .btn-close');
                        if (closeBtn) closeBtn.click();
                    }

                    if (e.key === 'Tab') {
                        const focusableContent = modal.querySelectorAll(focusableElements);
                        const firstElement = focusableContent[0];
                        const lastElement = focusableContent[focusableContent.length - 1];

                        if (e.shiftKey && document.activeElement === firstElement) {
                            e.preventDefault();
                            lastElement.focus();
                        } else if (!e.shiftKey && document.activeElement === lastElement) {
                            e.preventDefault();
                            firstElement.focus();
                        }
                    }
                });
            });
        },

        initSkipLinks: function() {
            const skipLink = document.createElement('a');
            skipLink.href = '#main-content';
            skipLink.textContent = '跳到主要内容';
            skipLink.className = 'skip-link';
            skipLink.style.cssText = 'position:absolute;top:-40px;left:0;background:#000;color:#fff;padding:8px;z-index:10000;transition:top 0.3s;';

            skipLink.addEventListener('focus', () => {
                skipLink.style.top = '0';
            });

            skipLink.addEventListener('blur', () => {
                skipLink.style.top = '-40px';
            });

            document.body.insertBefore(skipLink, document.body.firstChild);

            const main = document.querySelector('main, #main, [role="main"]');
            if (main) {
                main.id = 'main-content';
            }
        },

        initScreenReaderAnnouncements: function() {
            const announcer = document.createElement('div');
            announcer.id = 'sr-announcer';
            announcer.setAttribute('aria-live', 'polite');
            announcer.setAttribute('aria-atomic', 'true');
            announcer.style.cssText = 'position:absolute;width:1px;height:1px;padding:0;margin:-1px;overflow:hidden;clip:rect(0,0,0,0);white-space:nowrap;border:0;';
            document.body.appendChild(announcer);

            this.announce = function(message, priority = 'polite') {
                announcer.setAttribute('aria-live', priority);
                announcer.textContent = '';
                setTimeout(() => {
                    announcer.textContent = message;
                }, 100);
            };
        },

        initPerformanceMonitoring: function() {
            if (!this.config.enablePerformance) return;

            this.performance = {
                marks: {},
                measures: [],

                start: function(name) {
                    this.marks[name] = performance.now();
                },

                end: function(name) {
                    if (!this.marks[name]) return 0;
                    const duration = performance.now() - this.marks[name];
                    delete this.marks[name];
                    return duration;
                },

                measure: function(name, startMark, endMark) {
                    const start = this.marks[startMark] || performance.now();
                    const end = endMark ? (this.marks[endMark] || performance.now()) : performance.now();
                    const duration = end - start;
                    this.measures.push({ name, duration, timestamp: Date.now() });
                    return duration;
                },

                getEntries: function() {
                    return [...this.measures];
                },

                getTiming: function() {
                    const timing = performance.timing || {};
                    return {
                        dns: timing.domainLookupEnd - timing.domainLookupStart,
                        tcp: timing.connectEnd - timing.connectStart,
                        ttfb: timing.responseStart - timing.requestStart,
                        domParse: timing.domInteractive - timing.responseEnd,
                        domReady: timing.domContentLoadedEventEnd - timing.navigationStart,
                        loadComplete: timing.loadEventEnd - timing.navigationStart
                    };
                }
            };

            window.addEventListener('load', () => {
                setTimeout(() => {
                    const timing = this.performance.getTiming();
                    console.log('[Performance]', timing);

                    if (typeof window.__metricsCollector !== 'undefined') {
                        Object.entries(timing).forEach(([key, value]) => {
                            window.__metricsCollector.push({
                                metric: `page.${key}`,
                                value,
                                timestamp: Date.now()
                            });
                        });
                    }
                }, 0);
            });
        },

        initAnalytics: function() {
            if (!this.config.enableAnalytics) return;

            this.analytics = {
                pageviews: 0,
                events: [],

                trackPageview: function(path, title) {
                    this.pageviews++;
                    this.events.push({
                        type: 'pageview',
                        path: path || window.location.pathname,
                        title: title || document.title,
                        referrer: document.referrer,
                        timestamp: Date.now()
                    });

                    if (typeof window.__analyticsCollector !== 'undefined') {
                        window.__analyticsCollector.push({
                            type: 'pageview',
                            path: path,
                            title: title
                        });
                    }
                },

                trackEvent: function(category, action, label, value) {
                    this.events.push({
                        type: 'event',
                        category,
                        action,
                        label,
                        value,
                        timestamp: Date.now()
                    });

                    if (typeof window.__analyticsCollector !== 'undefined') {
                        window.__analyticsCollector.push({
                            type: 'event',
                            category,
                            action,
                            label,
                            value
                        });
                    }
                },

                trackTiming: function(category, variable, time, label) {
                    this.events.push({
                        type: 'timing',
                        category,
                        variable,
                        time,
                        label,
                        timestamp: Date.now()
                    });
                }
            };

            this.analytics.trackPageview();

            document.querySelectorAll('a, button').forEach(el => {
                el.addEventListener('click', (e) => {
                    const category = el.tagName.toLowerCase();
                    const action = e.type;
                    const label = el.textContent.trim() || el.id || el.className;

                    this.analytics.trackEvent(category, action, label);
                });
            });

            let scrollTimeout;
            window.addEventListener('scroll', () => {
                if (scrollTimeout) return;
                scrollTimeout = setTimeout(() => {
                    const scrollPercent = Math.round(
                        (window.scrollY / (document.body.scrollHeight - window.innerHeight)) * 100
                    );

                    if (scrollPercent >= this.config.scrollThreshold) {
                        this.analytics.trackEvent('scroll', 'depth', `${scrollPercent}%`);
                    }

                    scrollTimeout = null;
                }, 100);
            }, { passive: true });
        },

        trackUserBehavior: function(action, details = {}) {
            const behavior = {
                action,
                details,
                timestamp: Date.now(),
                url: window.location.href,
                userAgent: navigator.userAgent,
                screenWidth: window.innerWidth,
                screenHeight: window.innerHeight
            };

            if (typeof window.__behaviorCollector !== 'undefined') {
                window.__behaviorCollector.push(behavior);
            }
        },

        debounce: function(func, wait) {
            let timeout;
            return function executedFunction(...args) {
                const later = () => {
                    clearTimeout(timeout);
                    func(...args);
                };
                clearTimeout(timeout);
                timeout = setTimeout(later, wait);
            };
        },

        throttle: function(func, limit) {
            let inThrottle;
            return function(...args) {
                if (!inThrottle) {
                    func.apply(this, args);
                    inThrottle = true;
                    setTimeout(() => inThrottle = false, limit);
                }
            };
        },

        getModule: function(name) {
            return this.modules[name];
        },

        hasModule: function(name) {
            return name in this.modules;
        },

        registerModule: function(name, module) {
            this.modules[name] = module;
        },

        config: {}
    };

    window.App = App;

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', () => {
            App.init();
        });
    } else {
        App.init();
    }
})();
