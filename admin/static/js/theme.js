(function() {
  'use strict';

  const ThemeManager = {
    STORAGE_KEY: 'adminTheme',
    THEMES: {
      LIGHT: 'light',
      DARK: 'dark',
      AUTO: 'auto'
    },
    currentTheme: null,
    initialized: false,

    init: function() {
      if (this.initialized) {
        return;
      }

      this.applySavedTheme();
      this.setupEventListeners();
      this.watchSystemPreference();
      this.initialized = true;

      console.log('ThemeManager initialized with theme:', this.getCurrentTheme());
    },

    applySavedTheme: function() {
      const savedTheme = localStorage.getItem(this.STORAGE_KEY) || this.THEMES.AUTO;
      const resolvedTheme = this.resolveTheme(savedTheme);
      
      this.setTheme(resolvedTheme, false);
      this.currentTheme = savedTheme;
      this.updateThemeToggleIcon(resolvedTheme);
    },

    resolveTheme: function(theme) {
      if (theme === this.THEMES.AUTO) {
        return window.matchMedia('(prefers-color-scheme: dark)').matches 
          ? this.THEMES.DARK 
          : this.THEMES.LIGHT;
      }
      return theme;
    },

    setTheme: function(theme, save = true) {
      const resolvedTheme = this.resolveTheme(theme);
      
      document.documentElement.setAttribute('data-theme', resolvedTheme);
      
      if (save) {
        localStorage.setItem(this.STORAGE_KEY, theme);
        this.currentTheme = theme;
        this.updateThemeToggleIcon(resolvedTheme);
      }

      this.updateChartsTheme(resolvedTheme);
      this.updateBootstrapComponents(resolvedTheme);
      
      window.dispatchEvent(new CustomEvent('themechanged', { 
        detail: { 
          theme: resolvedTheme, 
          savedTheme: theme 
        } 
      }));
    },

    getCurrentTheme: function() {
      return document.documentElement.getAttribute('data-theme') || this.THEMES.LIGHT;
    },

    toggleTheme: function() {
      const currentResolved = this.getCurrentTheme();
      const newTheme = currentResolved === this.THEMES.DARK 
        ? this.THEMES.LIGHT 
        : this.THEMES.DARK;
      
      this.setTheme(newTheme, true);
      this.showThemeNotification(newTheme);
    },

    setLightTheme: function() {
      this.setTheme(this.THEMES.LIGHT, true);
      this.showThemeNotification(this.THEMES.LIGHT);
    },

    setDarkTheme: function() {
      this.setTheme(this.THEMES.DARK, true);
      this.showThemeNotification(this.THEMES.DARK);
    },

    setAutoTheme: function() {
      this.setTheme(this.THEMES.AUTO, true);
      const resolvedTheme = this.resolveTheme(this.THEMES.AUTO);
      this.showThemeNotification(resolvedTheme, true);
    },

    setupEventListeners: function() {
      const themeToggle = document.getElementById('themeToggle');
      if (themeToggle) {
        themeToggle.addEventListener('click', () => this.toggleTheme());
        
        themeToggle.addEventListener('keydown', (e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            this.toggleTheme();
          }
        });

        themeToggle.setAttribute('role', 'button');
        themeToggle.setAttribute('aria-label', '切换深色/浅色主题');
        themeToggle.setAttribute('tabindex', '0');
      }

      document.querySelectorAll('[data-theme-toggle]').forEach(btn => {
        btn.addEventListener('click', () => {
          const theme = btn.getAttribute('data-theme-toggle');
          if (theme === 'light') {
            this.setLightTheme();
          } else if (theme === 'dark') {
            this.setDarkTheme();
          } else if (theme === 'auto') {
            this.setAutoTheme();
          }
        });
      });

      window.addEventListener('storage', (e) => {
        if (e.key === this.STORAGE_KEY) {
          this.applySavedTheme();
        }
      });
    },

    watchSystemPreference: function() {
      const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
      
      mediaQuery.addEventListener('change', (e) => {
        const savedTheme = localStorage.getItem(this.STORAGE_KEY);
        if (!savedTheme || savedTheme === this.THEMES.AUTO) {
          this.setTheme(this.THEMES.AUTO, false);
          this.updateThemeToggleIcon(e.matches ? this.THEMES.DARK : this.THEMES.LIGHT);
        }
      });
    },

    updateThemeToggleIcon: function(theme) {
      const themeIcon = document.getElementById('themeIcon');
      if (themeIcon) {
        themeIcon.classList.remove('fa-moon', 'fa-sun', 'fa-adjust');
        
        if (theme === this.THEMES.DARK) {
          themeIcon.classList.add('fa-sun');
          themeIcon.setAttribute('aria-label', '当前为深色模式，点击切换到浅色模式');
        } else {
          themeIcon.classList.add('fa-moon');
          themeIcon.setAttribute('aria-label', '当前为浅色模式，点击切换到深色模式');
        }
      }

      const themeToggle = document.getElementById('themeToggle');
      if (themeToggle) {
        themeToggle.setAttribute('aria-pressed', theme === this.THEMES.DARK ? 'true' : 'false');
      }
    },

    updateChartsTheme: function(theme) {
      if (typeof echarts !== 'undefined') {
        const charts = ['trendChart', 'pieChart', 'captchaTypeChart', 'realtimeChart', 'miniChart'];
        
        charts.forEach(chartName => {
          const chart = window[chartName];
          if (chart && typeof chart.setOption === 'function') {
            const isDark = theme === this.THEMES.DARK;
            const textColor = isDark ? '#e9ecef' : '#666';
            const bgColor = isDark ? '#2d3238' : '#ffffff';
            
            chart.setOption({
              textStyle: {
                color: textColor
              },
              backgroundColor: 'transparent'
            }, false);
          }
        });
      }
    },

    updateBootstrapComponents: function(theme) {
      const isDark = theme === this.THEMES.DARK;
      
      document.querySelectorAll('.modal').forEach(modal => {
        if (isDark) {
          modal.classList.add('theme-dark');
        } else {
          modal.classList.remove('theme-dark');
        }
      });

      document.querySelectorAll('.dropdown-menu').forEach(dropdown => {
        if (isDark) {
          dropdown.classList.add('theme-dark');
        } else {
          dropdown.classList.remove('theme-dark');
        }
      });
    },

    showThemeNotification: function(theme, isAuto = false) {
      const themeName = theme === this.THEMES.DARK ? '深色' : '浅色';
      const message = isAuto 
        ? `已切换到${themeName}模式（系统偏好）` 
        : `已切换到${themeName}模式`;
      
      if (typeof showAdminToast === 'function') {
        showAdminToast(message, 'success');
      } else if (typeof Swal !== 'undefined') {
        Swal.fire({
          toast: true,
          position: 'top-end',
          icon: 'success',
          title: message,
          showConfirmButton: false,
          timer: 2000,
          timerProgressBar: true
        });
      }
    },

    getThemeColors: function() {
      const theme = this.getCurrentTheme();
      const isDark = theme === this.THEMES.DARK;
      
      return {
        isDark: isDark,
        primary: isDark ? '#4a9eff' : '#007bff',
        success: isDark ? '#2fd56a' : '#28a745',
        warning: isDark ? '#ffc107' : '#ffc107',
        danger: isDark ? '#ff4757' : '#dc3545',
        info: isDark ? '#17a2b8' : '#17a2b8',
        background: isDark ? '#1a1d21' : '#f4f6f9',
        card: isDark ? '#2d3238' : '#ffffff',
        text: isDark ? '#e9ecef' : '#333333',
        border: isDark ? '#3d434a' : '#dee2e6',
        sidebar: isDark ? '#1a1d21' : '#ffffff',
        navbar: isDark ? '#2d3238' : '#ffffff',
        chart: {
          grid: isDark ? '#3d434a' : '#e0e0e0',
          axis: isDark ? '#e9ecef' : '#666666',
          tooltip: isDark ? 'rgba(45, 50, 56, 0.9)' : 'rgba(0, 0, 0, 0.8)'
        }
      };
    },

    isDarkMode: function() {
      return this.getCurrentTheme() === this.THEMES.DARK;
    },

    isLightMode: function() {
      return this.getCurrentTheme() === this.THEMES.LIGHT;
    },

    getSavedThemePreference: function() {
      return localStorage.getItem(this.STORAGE_KEY) || this.THEMES.AUTO;
    },

    resetToSystemPreference: function() {
      this.setAutoTheme();
    },

    addThemeChangeListener: function(callback) {
      if (typeof callback === 'function') {
        window.addEventListener('themechanged', (e) => callback(e.detail));
      }
    },

    removeThemeChangeListener: function(callback) {
      if (typeof callback === 'function') {
        window.removeEventListener('themechanged', (e) => callback(e.detail));
      }
    },

    destroy: function() {
      this.initialized = false;
      this.currentTheme = null;
      console.log('ThemeManager destroyed');
    }
  };

  if (typeof window !== 'undefined') {
    window.ThemeManager = ThemeManager;
  }

  if (typeof define === 'function' && define.amd) {
    define([], function() {
      return ThemeManager;
    });
  }

  if (typeof module !== 'undefined' && module.exports) {
    module.exports = ThemeManager;
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => ThemeManager.init());
  } else {
    ThemeManager.init();
  }
})();
