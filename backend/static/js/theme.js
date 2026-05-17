(function() {
    'use strict';

    const THEME_STORAGE_KEY = 'theme';
    const THEME_ATTRIBUTE = 'data-bs-theme';
    const DEFAULT_THEME = 'light';

    function getStoredTheme() {
        return localStorage.getItem(THEME_STORAGE_KEY);
    }

    function setStoredTheme(theme) {
        localStorage.setItem(THEME_STORAGE_KEY, theme);
    }

    function getSystemTheme() {
        return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    }

    function getInitialTheme() {
        const stored = getStoredTheme();
        if (stored === 'auto' || !stored) {
            return getSystemTheme();
        }
        return stored;
    }

    function applyTheme(theme) {
        document.documentElement.setAttribute(THEME_ATTRIBUTE, theme);
        document.querySelector('meta[name="theme-color"]')?.setAttribute('content', theme === 'dark' ? '#212529' : '#0d6efd');
    }

    function updateThemeIcon(theme, iconElement) {
        if (iconElement) {
            iconElement.className = theme === 'dark' ? 'fas fa-sun' : 'fas fa-moon';
        }
    }

    function toggleTheme() {
        const current = document.documentElement.getAttribute(THEME_ATTRIBUTE);
        const newTheme = current === 'dark' ? 'light' : 'dark';
        applyTheme(newTheme);
        setStoredTheme(newTheme);
        
        const iconElement = document.getElementById('themeIcon');
        updateThemeIcon(newTheme, iconElement);
        
        if (typeof showToast === 'function') {
            showToast(`已切换到${newTheme === 'dark' ? '深色' : '浅色'}模式`, 'success');
        }
        
        return newTheme;
    }

    function initThemeToggle() {
        const toggleBtn = document.getElementById('themeToggle');
        const iconElement = document.getElementById('themeIcon');
        
        if (toggleBtn) {
            toggleBtn.addEventListener('click', toggleTheme);
            toggleBtn.addEventListener('keydown', function(e) {
                if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    toggleTheme();
                }
            });
        }
    }

    function initSystemThemeListener() {
        window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', function(e) {
            if (getStoredTheme() === 'auto' || !getStoredTheme()) {
                const newTheme = e.matches ? 'dark' : 'light';
                applyTheme(newTheme);
                const iconElement = document.getElementById('themeIcon');
                updateThemeIcon(newTheme, iconElement);
            }
        });
    }

    function initTheme() {
        const initialTheme = getInitialTheme();
        applyTheme(initialTheme);
        
        const iconElement = document.getElementById('themeIcon');
        updateThemeIcon(initialTheme, iconElement);
        
        initThemeToggle();
        initSystemThemeListener();
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', initTheme);
    } else {
        initTheme();
    }

    window.ThemeManager = {
        getTheme: function() {
            return document.documentElement.getAttribute(THEME_ATTRIBUTE);
        },
        setTheme: function(theme) {
            if (theme === 'dark' || theme === 'light' || theme === 'auto') {
                if (theme === 'auto') {
                    applyTheme(getSystemTheme());
                } else {
                    applyTheme(theme);
                }
                setStoredTheme(theme);
                const iconElement = document.getElementById('themeIcon');
                updateThemeIcon(document.documentElement.getAttribute(THEME_ATTRIBUTE), iconElement);
            }
        },
        toggleTheme: toggleTheme
    };
})();
