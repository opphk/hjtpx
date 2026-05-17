// 主题切换功能
(function() {
    'use strict';

    const ThemeManager = {
        currentTheme: 'light',
        storageKey: 'hjtpx-theme',

        init: function() {
            this.currentTheme = this.getStoredTheme() || 'light';
            this.applyTheme(this.currentTheme);
            this.setupToggle();
        },

        getStoredTheme: function() {
            try {
                return localStorage.getItem(this.storageKey);
            } catch (e) {
                return null;
            }
        },

        storeTheme: function(theme) {
            try {
                localStorage.setItem(this.storageKey, theme);
            } catch (e) {
                console.warn('Could not store theme preference:', e);
            }
        },

        applyTheme: function(theme) {
            const body = document.body;
            body.classList.remove('theme-light', 'theme-dark', 'theme-blue');
            body.classList.add('theme-' + theme);
            this.currentTheme = theme;
            this.storeTheme(theme);

            // 触发主题变化事件
            const event = new CustomEvent('themeChange', { detail: { theme: theme } });
            document.dispatchEvent(event);
        },

        toggleTheme: function() {
            const themes = ['light', 'dark', 'blue'];
            const currentIndex = themes.indexOf(this.currentTheme);
            const nextIndex = (currentIndex + 1) % themes.length;
            this.applyTheme(themes[nextIndex]);
        },

        setupToggle: function() {
            const toggleBtn = document.getElementById('theme-toggle');
            if (toggleBtn) {
                toggleBtn.addEventListener('click', () => {
                    this.toggleTheme();
                    this.updateToggleIcon(toggleBtn);
                });
            }
        },

        updateToggleIcon: function(btn) {
            const icons = {
                light: '☀️',
                dark: '🌙',
                blue: '💙'
            };
            if (btn.innerHTML = icons[this.currentTheme] || '🎨';
        },

        getTheme: function() {
            return this.currentTheme;
        },

        isDarkMode: function() {
            return this.currentTheme === 'dark';
        }
    };

    // 动画效果增强
    const AnimationEnhancer = {
        init: function() {
            this.setupSmoothScroll();
            this.setupHoverEffects();
            this.setupLoadAnimations();
        },

        setupSmoothScroll: function() {
            document.querySelectorAll('a[href^="#"]').forEach(anchor => {
                anchor.addEventListener('click', function (e) {
                    e.preventDefault();
                    const target = document.querySelector(this.getAttribute('href'));
                    if (target) {
                        target.scrollIntoView({
                            behavior: 'smooth',
                            block: 'start'
                        });
                    }
                });
            });
        },

        setupHoverEffects: function() {
            document.querySelectorAll('.btn, .card, button').forEach(el => {
                el.addEventListener('mouseenter', function() {
                    this.style.transform = 'translateY(-2px)';
                    this.style.transition = 'all 0.3s ease';
                });
                el.addEventListener('mouseleave', function() {
                    this.style.transform = 'translateY(0)';
                });
            });
        },

        setupLoadAnimations: function() {
            const observer = new IntersectionObserver((entries) => {
                entries.forEach(entry => {
                    if (entry.isIntersecting) {
                    entry.target.classList.add('animate-in');
                }
            });
        }, { threshold: 0.1 });

            document.querySelectorAll('.animate-on-load').forEach(el => {
                observer.observe(el);
            });
        }
    };

    // 无障碍支持
    const Accessibility = {
        init: function() {
            this.setupKeyboardNavigation();
            this.setupFocusIndicators();
        },

        setupKeyboardNavigation: function() {
            document.addEventListener('keydown', function(e) {
                if (e.key === 'Escape') {
                    const modals = document.querySelectorAll('.modal.show');
                    modals.forEach(modal => {
                        modal.classList.remove('show');
                    });
                }
            });
        },

        setupFocusIndicators: function() {
            document.addEventListener('keydown', function(e) {
                if (e.key === 'Tab') {
                    document.body.classList.add('user-is-tabbing');
                }
            });
            document.addEventListener('mousedown', function() {
                document.body.classList.remove('user-is-tabbing');
            });
        }
    };

    // 初始化
    document.addEventListener('DOMContentLoaded', function() {
        ThemeManager.init();
        AnimationEnhancer.init();
        Accessibility.init();
    });

    // 导出到全局
    window.ThemeManager = ThemeManager;
    window.AnimationEnhancer = AnimationEnhancer;
    window.Accessibility = Accessibility;

})();
