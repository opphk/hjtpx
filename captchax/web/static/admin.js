(function() {
    'use strict';

    const API_BASE = '/admin/api';
    const POLL_INTERVAL = 30000;
    const POLL_INTERVAL_FAST = 10000;

    // API Client
    const ApiClient = {
        async request(endpoint, options = {}) {
            const url = API_BASE + endpoint;
            const token = localStorage.getItem('admin_token');
            const config = {
                headers: {
                    'Content-Type': 'application/json',
                    ...(token ? { 'Authorization': `Bearer ${token}` } : {}),
                    ...options.headers
                },
                ...options
            };

            if (options.body && typeof options.body === 'object') {
                config.body = JSON.stringify(options.body);
            }

            try {
                const response = await fetch(url, config);
                const data = await response.json();

                if (!response.ok) {
                    throw new Error(data.message || 'Request failed');
                }

                return data;
            } catch (error) {
                console.error('API Error:', error);
                throw error;
            }
        },

        auth: {
            async login(username, password) {
                return ApiClient.request('/login', {
                    method: 'POST',
                    body: { username, password }
                });
            },

            async logout() {
                return ApiClient.request('/logout', {
                    method: 'POST'
                });
            },

            async checkSession() {
                return ApiClient.request('/auth/session', {
                    method: 'GET'
                });
            }
        },

        stats: {
            async getDashboard() {
                return ApiClient.request('/dashboard', {
                    method: 'GET'
                });
            },

            async getTrend(hours = 24) {
                return ApiClient.request(`/stats/trend?hours=${hours}`, {
                    method: 'GET'
                });
            },

            async getCaptchaDistribution() {
                return ApiClient.request('/stats/captcha-distribution', {
                    method: 'GET'
                });
            },

            async getResultStats() {
                return ApiClient.request('/stats/results', {
                    method: 'GET'
                });
            },

            async getIpRanking(limit = 20) {
                return ApiClient.request(`/stats/ip-ranking?limit=${limit}`, {
                    method: 'GET'
                });
            },

            async exportData(format = 'csv', type = 'stats') {
                return ApiClient.request(`/stats/export?format=${format}&type=${type}`, {
                    method: 'GET'
                });
            }
        },

        config: {
            async get() {
                return ApiClient.request('/config', {
                    method: 'GET'
                });
            },

            async update(config) {
                return ApiClient.request('/config', {
                    method: 'POST',
                    body: config
                });
            }
        },

        whitelist: {
            async list(page = 1, pageSize = 20, search = '', type = '') {
                let url = `/whitelist?page=${page}&page_size=${pageSize}&search=${encodeURIComponent(search)}`;
                if (type) url += `&type=${encodeURIComponent(type)}`;
                return ApiClient.request(url, {
                    method: 'GET'
                });
            },

            async add(entry) {
                return ApiClient.request('/whitelist', {
                    method: 'POST',
                    body: entry
                });
            },

            async delete(id) {
                return ApiClient.request(`/whitelist/${id}`, {
                    method: 'DELETE'
                });
            }
        },

        blacklist: {
            async list(page = 1, pageSize = 20, search = '', status = '', type = '') {
                let url = `/blacklist?page=${page}&page_size=${pageSize}&search=${encodeURIComponent(search)}`;
                if (status) url += `&status=${encodeURIComponent(status)}`;
                if (type) url += `&type=${encodeURIComponent(type)}`;
                return ApiClient.request(url, {
                    method: 'GET'
                });
            },

            async add(entry) {
                return ApiClient.request('/blacklist', {
                    method: 'POST',
                    body: entry
                });
            },

            async delete(id) {
                return ApiClient.request(`/blacklist/${id}`, {
                    method: 'DELETE'
                });
            }
        },

        analytics: {
            async getOverview(range = '7d') {
                return ApiClient.request(`/analytics/overview?range=${range}`, {
                    method: 'GET'
                });
            },

            async getTrends(range = '7d', interval = '', startDate = '', endDate = '') {
                let url = `/analytics/trends?range=${range}`;
                if (interval) url += `&interval=${interval}`;
                if (startDate) url += `&start_date=${startDate}`;
                if (endDate) url += `&end_date=${endDate}`;
                return ApiClient.request(url, {
                    method: 'GET'
                });
            },

            async getDistribution(range = '7d', groupBy = 'type') {
                return ApiClient.request(`/analytics/distribution?range=${range}&group_by=${groupBy}`, {
                    method: 'GET'
                });
            },

            async getGeo(range = '7d', limit = 20) {
                return ApiClient.request(`/analytics/geo?range=${range}&limit=${limit}`, {
                    method: 'GET'
                });
            },

            async getDevices(range = '7d') {
                return ApiClient.request(`/analytics/devices?range=${range}`, {
                    method: 'GET'
                });
            },

            async getRisk(range = '7d') {
                return ApiClient.request(`/analytics/risk?range=${range}`, {
                    method: 'GET'
                });
            }
        }
    };

    window.AdminApi = ApiClient;

    // i18n Helper
    const I18nHelper = {
        initialized: false,
        currentLocale: 'zh-CN',
        translations: {},

        async init() {
            try {
                const locales = ['zh-CN', 'en', 'ja', 'ko'];
                const promises = locales.map(locale => 
                    fetch(`/i18n/${locale}.json`)
                        .then(res => res.ok ? res.json() : null)
                        .catch(() => null)
                );

                const results = await Promise.all(promises);
                locales.forEach((locale, i) => {
                    if (results[i]) {
                        this.translations[locale] = results[i];
                    }
                });

                const stored = localStorage.getItem('admin_locale');
                if (stored && this.translations[stored]) {
                    this.currentLocale = stored;
                } else {
                    const browserLang = navigator.language || navigator.userLanguage;
                    for (const locale of locales) {
                        if (browserLang.startsWith(locale)) {
                            this.currentLocale = locale;
                            break;
                        }
                    }
                }

                this.initialized = true;
                this.applyTranslations();
            } catch (e) {
                console.warn('i18n init failed:', e);
            }
        },

        t(key, params = {}) {
            const keys = key.split('.');
            let value = this.translations[this.currentLocale];
            
            for (const k of keys) {
                if (value && typeof value === 'object') {
                    value = value[k];
                } else {
                    value = undefined;
                    break;
                }
            }

            if (typeof value !== 'string') {
                value = key;
            }

            return value.replace(/\{(\w+)\}/g, (match, k) => 
                params[k] !== undefined ? params[k] : match
            );
        },

        setLocale(locale) {
            if (this.translations[locale]) {
                this.currentLocale = locale;
                localStorage.setItem('admin_locale', locale);
                document.documentElement.lang = locale;
                this.applyTranslations();
            }
        },

        applyTranslations() {
            document.querySelectorAll('[data-i18n]').forEach(el => {
                const key = el.getAttribute('data-i18n');
                el.textContent = this.t(key);
            });

            document.querySelectorAll('[data-i18n-attr]').forEach(el => {
                try {
                    const attrs = JSON.parse(el.getAttribute('data-i18n-attr'));
                    Object.entries(attrs).forEach(([attr, key]) => {
                        el.setAttribute(attr, this.t(key));
                    });
                } catch (e) {}
            });
        },

        initLanguageSwitcher() {
            const container = document.getElementById('language-switcher');
            if (!container) return;

            const locales = [
                { code: 'zh-CN', name: '简体中文', flag: '🇨🇳' },
                { code: 'en', name: 'English', flag: '🇺🇸' },
                { code: 'ja', name: '日本語', flag: '🇯🇵' },
                { code: 'ko', name: '한국어', flag: '🇰🇷' }
            ];

            container.innerHTML = `
                <div class="relative">
                    <button id="lang-toggle" class="flex items-center space-x-2 px-3 py-2 rounded-lg hover:bg-gray-100 transition">
                        <span id="current-flag">${locales.find(l => l.code === this.currentLocale)?.flag || '🌐'}</span>
                        <span class="text-sm" id="current-lang-name">${locales.find(l => l.code === this.currentLocale)?.name || 'Language'}</span>
                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
                        </svg>
                    </button>
                    <div id="lang-dropdown" class="hidden absolute right-0 mt-2 w-48 bg-white rounded-lg shadow-lg border py-1 z-50">
                        ${locales.map(l => `
                            <button data-lang="${l.code}" class="w-full flex items-center space-x-3 px-4 py-2 hover:bg-gray-100 transition text-left ${l.code === this.currentLocale ? 'bg-blue-50 text-blue-600' : ''}">
                                <span class="text-lg">${l.flag}</span>
                                <span class="text-sm">${l.name}</span>
                            </button>
                        `).join('')}
                    </div>
                </div>
            `;

            const toggle = document.getElementById('lang-toggle');
            const dropdown = document.getElementById('lang-dropdown');

            toggle.addEventListener('click', (e) => {
                e.stopPropagation();
                dropdown.classList.toggle('hidden');
            });

            dropdown.addEventListener('click', (e) => {
                const btn = e.target.closest('[data-lang]');
                if (btn) {
                    const locale = btn.dataset.lang;
                    this.setLocale(locale);
                    const flag = locales.find(l => l.code === locale)?.flag;
                    const name = locales.find(l => l.code === locale)?.name;
                    document.getElementById('current-flag').textContent = flag;
                    document.getElementById('current-lang-name').textContent = name;
                    dropdown.querySelectorAll('[data-lang]').forEach(b => {
                        b.classList.remove('bg-blue-50', 'text-blue-600');
                        if (b.dataset.lang === locale) {
                            b.classList.add('bg-blue-50', 'text-blue-600');
                        }
                    });
                    dropdown.classList.add('hidden');
                }
            });

            document.addEventListener('click', () => dropdown.classList.add('hidden'));
        }
    };

    window.I18nHelper = I18nHelper;

    // UI Controller
    const UIController = {
        showToast(message, type = 'info') {
            const container = document.getElementById('toast-container') || this.createToastContainer();
            const toast = document.createElement('div');
            const colors = {
                success: 'bg-green-500',
                error: 'bg-red-500',
                warning: 'bg-yellow-500',
                info: 'bg-blue-500'
            };
            const icons = {
                success: '<svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path></svg>',
                error: '<svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path></svg>',
                warning: '<svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path></svg>',
                info: '<svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>'
            };

            toast.className = `${colors[type]} text-white px-4 py-3 rounded-lg shadow-lg mb-2 transform transition-all duration-300 translate-x-full flex items-center`;
            toast.innerHTML = icons[type] + message;
            container.appendChild(toast);

            requestAnimationFrame(() => {
                toast.classList.remove('translate-x-full');
            });

            setTimeout(() => {
                toast.classList.add('translate-x-full');
                setTimeout(() => toast.remove(), 300);
            }, 3000);
        },

        createToastContainer() {
            const container = document.createElement('div');
            container.id = 'toast-container';
            container.className = 'fixed top-4 right-4 z-50 flex flex-col';
            document.body.appendChild(container);
            return container;
        },

        showLoading(element, text = 'Loading...') {
            if (!element) return;
            element.dataset.originalContent = element.innerHTML;
            element.disabled = true;
            element.classList.add('opacity-75', 'cursor-not-allowed');
            element.innerHTML = `<svg class="animate-spin h-5 w-5 inline mr-2" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>${text}`;
        },

        hideLoading(element) {
            if (!element || !element.dataset.originalContent) return;
            element.disabled = false;
            element.classList.remove('opacity-75', 'cursor-not-allowed');
            element.innerHTML = element.dataset.originalContent;
            delete element.dataset.originalContent;
        },

        confirm(message, title = 'Confirm') {
            return new Promise((resolve) => {
                const overlay = document.createElement('div');
                overlay.className = 'fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50';
                overlay.innerHTML = `
                    <div class="bg-white rounded-xl p-6 max-w-md mx-4 shadow-2xl">
                        <div class="flex items-center mb-4">
                            <div class="w-10 h-10 bg-blue-100 rounded-full flex items-center justify-center mr-3">
                                <svg class="w-5 h-5 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                                </svg>
                            </div>
                            <h3 class="text-lg font-semibold text-gray-800">${title}</h3>
                        </div>
                        <p class="text-gray-600 mb-6">${message}</p>
                        <div class="flex justify-end space-x-3">
                            <button class="cancel-btn px-4 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition" data-i18n="common.cancel">Cancel</button>
                            <button class="confirm-btn px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition" data-i18n="common.confirm">Confirm</button>
                        </div>
                    </div>
                `;

                if (I18nHelper.initialized) {
                    I18nHelper.applyTranslations();
                }

                document.body.appendChild(overlay);

                overlay.querySelector('.cancel-btn').onclick = () => {
                    overlay.remove();
                    resolve(false);
                };

                overlay.querySelector('.confirm-btn').onclick = () => {
                    overlay.remove();
                    resolve(true);
                };

                overlay.addEventListener('click', (e) => {
                    if (e.target === overlay) {
                        overlay.remove();
                        resolve(false);
                    }
                });
            });
        },

        formatNumber(num) {
            if (num === null || num === undefined) return '0';
            return new Intl.NumberFormat().format(num);
        },

        formatDate(dateString) {
            if (!dateString) return '-';
            const date = new Date(dateString);
            return date.toLocaleString(I18nHelper.currentLocale);
        },

        formatDuration(seconds) {
            if (seconds < 60) return seconds + 's';
            if (seconds < 3600) return Math.floor(seconds / 60) + 'm';
            const hours = Math.floor(seconds / 3600);
            const mins = Math.floor((seconds % 3600) / 60);
            return `${hours}h ${mins}m`;
        }
    };

    window.AdminUI = UIController;

    // Chart Manager
    const ChartManager = {
        charts: {},

        initTrendChart(canvasId, data, options = {}) {
            const ctx = document.getElementById(canvasId);
            if (!ctx || !window.Chart) return;

            if (this.charts[canvasId]) {
                this.charts[canvasId].destroy();
            }

            const labels = data.map(d => d.time);
            const verifiedData = data.map(d => d.verified);
            const rejectedData = data.map(d => d.rejected);

            const chartConfig = {
                type: 'line',
                data: {
                    labels: labels,
                    datasets: [
                        {
                            label: I18nHelper.t('admin.charts.verified') || 'Verified',
                            data: verifiedData,
                            borderColor: '#10b981',
                            backgroundColor: 'rgba(16, 185, 129, 0.1)',
                            fill: true,
                            tension: 0.4,
                            pointRadius: options.pointRadius || 3,
                            pointHoverRadius: options.pointHoverRadius || 6,
                            borderWidth: options.borderWidth || 2
                        },
                        {
                            label: I18nHelper.t('admin.charts.rejected') || 'Rejected',
                            data: rejectedData,
                            borderColor: '#ef4444',
                            backgroundColor: 'rgba(239, 68, 68, 0.1)',
                            fill: true,
                            tension: 0.4,
                            pointRadius: options.pointRadius || 3,
                            pointHoverRadius: options.pointHoverRadius || 6,
                            borderWidth: options.borderWidth || 2
                        }
                    ]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    interaction: {
                        intersect: false,
                        mode: 'index'
                    },
                    plugins: {
                        legend: {
                            position: 'top',
                            labels: {
                                usePointStyle: true,
                                padding: 20
                            }
                        },
                        tooltip: {
                            backgroundColor: 'rgba(0, 0, 0, 0.8)',
                            padding: 12,
                            titleFont: { size: 14 },
                            bodyFont: { size: 13 },
                            displayColors: true,
                            callbacks: {
                                label: function(context) {
                                    let label = context.dataset.label || '';
                                    if (label) {
                                        label += ': ';
                                    }
                                    if (context.parsed.y !== null) {
                                        label += AdminUI.formatNumber(context.parsed.y);
                                    }
                                    return label;
                                }
                            }
                        }
                    },
                    scales: {
                        x: {
                            grid: {
                                display: false
                            }
                        },
                        y: {
                            beginAtZero: true,
                            ticks: {
                                callback: function(value) {
                                    return AdminUI.formatNumber(value);
                                }
                            }
                        }
                    },
                    animation: {
                        duration: 750,
                        easing: 'easeOutQuart'
                    }
                }
            };

            if (options.onClick) {
                chartConfig.options.onClick = options.onClick;
            }

            this.charts[canvasId] = new Chart(ctx, chartConfig);
            return this.charts[canvasId];
        },

        initDistributionChart(canvasId, data, options = {}) {
            const ctx = document.getElementById(canvasId);
            if (!ctx || !window.Chart) return;

            if (this.charts[canvasId]) {
                this.charts[canvasId].destroy();
            }

            const colors = options.colors || [
                '#3b82f6',
                '#8b5cf6',
                '#ec4899',
                '#f59e0b',
                '#10b981',
                '#ef4444'
            ];

            const chartConfig = {
                type: options.type || 'doughnut',
                data: {
                    labels: data.map(d => d.label || d.type),
                    datasets: [{
                        data: data.map(d => d.count || d.value),
                        backgroundColor: colors.slice(0, data.length),
                        borderWidth: 2,
                        borderColor: '#ffffff'
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                        legend: {
                            position: options.legendPosition || 'right',
                            labels: {
                                usePointStyle: true,
                                padding: 15
                            }
                        },
                        tooltip: {
                            backgroundColor: 'rgba(0, 0, 0, 0.8)',
                            padding: 12,
                            callbacks: {
                                label: function(context) {
                                    const total = context.dataset.data.reduce((a, b) => a + b, 0);
                                    const percentage = ((context.parsed / total) * 100).toFixed(1);
                                    return `${context.label}: ${AdminUI.formatNumber(context.parsed)} (${percentage}%)`;
                                }
                            }
                        }
                    },
                    animation: {
                        animateRotate: true,
                        animateScale: true
                    }
                }
            };

            if (options.onClick) {
                chartConfig.options.onClick = (evt, elements) => {
                    if (elements.length > 0) {
                        const index = elements[0].index;
                        options.onClick(data[index], index);
                    }
                };
            }

            this.charts[canvasId] = new Chart(ctx, chartConfig);
            return this.charts[canvasId];
        },

        initBarChart(canvasId, data, options = {}) {
            const ctx = document.getElementById(canvasId);
            if (!ctx || !window.Chart) return;

            if (this.charts[canvasId]) {
                this.charts[canvasId].destroy();
            }

            this.charts[canvasId] = new Chart(ctx, {
                type: 'bar',
                data: {
                    labels: data.labels || data.map(d => d.label),
                    datasets: [{
                        label: options.label || 'Data',
                        data: data.values || data.map(d => d.value),
                        backgroundColor: options.colors || 'rgba(59, 130, 246, 0.5)',
                        borderColor: options.borderColors || 'rgba(59, 130, 246, 1)',
                        borderWidth: 2,
                        borderRadius: 6
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                        legend: {
                            display: false
                        },
                        tooltip: {
                            backgroundColor: 'rgba(0, 0, 0, 0.8)',
                            padding: 12,
                            callbacks: {
                                label: function(context) {
                                    return AdminUI.formatNumber(context.parsed.y);
                                }
                            }
                        }
                    },
                    scales: {
                        y: {
                            beginAtZero: true,
                            ticks: {
                                callback: function(value) {
                                    return AdminUI.formatNumber(value);
                                }
                            }
                        }
                    }
                }
            });
        },

        destroyAll() {
            Object.keys(this.charts).forEach(id => {
                if (this.charts[id]) {
                    this.charts[id].destroy();
                    delete this.charts[id];
                }
            });
        }
    };

    window.AdminCharts = ChartManager;

    // Export Helper
    const ExportHelper = {
        downloadCSV(data, filename, columns) {
            if (!data || data.length === 0) {
                UIController.showToast('No data to export', 'warning');
                return;
            }

            const headers = columns.map(col => col.title);
            const rows = data.map(item =>
                columns.map(col => {
                    let value = item[col.key];
                    if (col.formatter && typeof col.formatter === 'function') {
                        value = col.formatter(value, item);
                    }
                    if (value === null || value === undefined) value = '';
                    value = String(value).replace(/"/g, '""');
                    if (value.includes(',') || value.includes('\n') || value.includes('"')) {
                        value = `"${value}"`;
                    }
                    return value;
                })
            );

            const csvContent = [headers.join(','), ...rows.map(r => r.join(','))].join('\n');
            const BOM = '\uFEFF';
            const blob = new Blob([BOM + csvContent], { type: 'text/csv;charset=utf-8' });

            this.downloadBlob(blob, filename + '.csv');
        },

        downloadBlob(blob, filename) {
            const url = URL.createObjectURL(blob);
            const link = document.createElement('a');
            link.href = url;
            link.download = filename;
            document.body.appendChild(link);
            link.click();
            document.body.removeChild(link);
            URL.revokeObjectURL(url);
        },

        downloadJSON(data, filename) {
            const jsonContent = JSON.stringify(data, null, 2);
            const blob = new Blob([jsonContent], { type: 'application/json' });
            this.downloadBlob(blob, filename + '.json');
        }
    };

    window.ExportHelper = ExportHelper;

    // Realtime Updater
    const RealtimeUpdater = {
        timers: {},
        callbacks: {},
        intervals: {},

        start(key, callback, interval = POLL_INTERVAL) {
            this.stop(key);
            if (callback) {
                callback();
                this.intervals[key] = interval;
                this.callbacks[key] = callback;
                this.timers[key] = setInterval(callback, interval);
            }
        },

        stop(key) {
            if (this.timers[key]) {
                clearInterval(this.timers[key]);
                delete this.timers[key];
                delete this.callbacks[key];
                delete this.intervals[key];
            }
        },

        stopAll() {
            Object.keys(this.timers).forEach(key => this.stop(key));
        },

        refresh(key) {
            if (this.callbacks[key]) {
                this.callbacks[key]();
            }
        }
    };

    window.RealtimeUpdater = RealtimeUpdater;

    // Initialization
    async function init() {
        await I18nHelper.init();
        I18nHelper.initLanguageSwitcher();
        initEventListeners();
    }

    function initEventListeners() {
        // Login form
        const loginForm = document.getElementById('login-form');
        if (loginForm) {
            loginForm.addEventListener('submit', async function(e) {
                e.preventDefault();

                const username = document.getElementById('username');
                const password = document.getElementById('password');
                const submitBtn = document.getElementById('login-btn');
                const errorDiv = document.getElementById('login-error');

                if (!username.value.trim() || !password.value) {
                    errorDiv.textContent = I18nHelper.t('admin.form.loginRequired') || 'Please enter username and password';
                    errorDiv.classList.remove('hidden');
                    return;
                }

                UIController.showLoading(submitBtn, I18nHelper.t('common.loading') || 'Loading...');
                errorDiv.classList.add('hidden');

                try {
                    const result = await ApiClient.auth.login(username.value.trim(), password.value);
                    localStorage.setItem('admin_token', result.token);
                    window.location.href = '/admin/dashboard';
                } catch (error) {
                    errorDiv.textContent = error.message || I18nHelper.t('admin.form.loginError') || 'Login failed';
                    errorDiv.classList.remove('hidden');
                } finally {
                    UIController.hideLoading(submitBtn);
                }
            });
        }

        // Mobile menu
        const sidebar = document.getElementById('sidebar');
        const mobileMenuBtn = document.getElementById('mobile-menu-btn');
        if (mobileMenuBtn && sidebar) {
            mobileMenuBtn.addEventListener('click', function() {
                sidebar.classList.toggle('-translate-x-full');
            });

            sidebar.addEventListener('click', function(e) {
                if (e.target === sidebar && !sidebar.classList.contains('-translate-x-full')) {
                    sidebar.classList.add('-translate-x-full');
                }
            });
        }

        // Logout button
        const logoutBtn = document.getElementById('logout-btn');
        if (logoutBtn) {
            logoutBtn.addEventListener('click', async function() {
                const confirmed = await UIController.confirm(
                    I18nHelper.t('admin.logoutConfirm') || 'Are you sure you want to logout?',
                    I18nHelper.t('admin.logout') || 'Logout'
                );
                if (confirmed) {
                    try {
                        await ApiClient.auth.logout();
                    } catch (e) {}
                    localStorage.removeItem('admin_token');
                    window.location.href = '/admin/login';
                }
            });
        }

        // Refresh button
        const refreshBtn = document.getElementById('refresh-btn');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', function() {
                RealtimeUpdater.refresh('dashboard');
                UIController.showToast(I18nHelper.t('common.dataRefreshed') || 'Data refreshed', 'success');
            });
        }
    }

    // Page-specific functions (used in templates)
    function generateMockTrendData() {
        const data = [];
        for (let i = 23; i >= 0; i--) {
            const time = new Date();
            time.setHours(time.getHours() - i);
            data.push({
                time: time.getHours() + ':00',
                verified: Math.floor(Math.random() * 500) + 200,
                rejected: Math.floor(Math.random() * 50) + 10
            });
        }
        return data;
    }

    function generateMockDistribution() {
        return [
            { type: 'slider', label: 'Slider', count: 5420 },
            { type: 'click', label: 'Click', count: 3890 },
            { type: 'rotate', label: 'Rotate', count: 2560 }
        ];
    }

    function generateMockIpRanking() {
        const ips = [];
        const statuses = ['normal', 'normal', 'normal', 'suspicious', 'suspicious', 'blocked'];
        for (let i = 0; i < 10; i++) {
            const total = Math.floor(Math.random() * 5000) + 100;
            const verified = Math.floor(total * (0.8 + Math.random() * 0.2));
            ips.push({
                ip: `192.168.${Math.floor(Math.random() * 255)}.${Math.floor(Math.random() * 255)}`,
                total: total,
                verified: verified,
                successRate: ((verified / total) * 100).toFixed(1),
                status: statuses[Math.floor(Math.random() * statuses.length)]
            });
        }
        return ips.sort((a, b) => b.total - a.total);
    }

    window.generateMockTrendData = generateMockTrendData;
    window.generateMockDistribution = generateMockDistribution;
    window.generateMockIpRanking = generateMockIpRanking;

    // Init on DOM ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }

    window.addEventListener('beforeunload', function() {
        AdminCharts.destroyAll();
        RealtimeUpdater.stopAll();
    });

})();
