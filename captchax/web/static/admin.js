(function() {
    'use strict';

    const API_BASE = '/api/admin';
    const POLL_INTERVAL = 30000;
    const POLL_INTERVAL_FAST = 10000;

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
                    throw new Error(data.message || '请求失败');
                }

                return data;
            } catch (error) {
                console.error('API Error:', error);
                throw error;
            }
        },

        auth: {
            async login(username, password) {
                return ApiClient.request('/auth/login', {
                    method: 'POST',
                    body: { username, password }
                });
            },

            async logout() {
                return ApiClient.request('/auth/logout', {
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
                return ApiClient.request('/stats/dashboard', {
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
                    method: 'PUT',
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
            },

            async batchDelete(ids) {
                return ApiClient.request('/whitelist/batch-delete', {
                    method: 'POST',
                    body: { ids }
                });
            },

            async batchAdd(entries) {
                return ApiClient.request('/whitelist/batch-add', {
                    method: 'POST',
                    body: { entries }
                });
            },

            async export(format = 'csv') {
                return ApiClient.request(`/whitelist/export?format=${format}`, {
                    method: 'GET'
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
            },

            async unban(id) {
                return ApiClient.request(`/blacklist/${id}/unban`, {
                    method: 'POST'
                });
            },

            async batchDelete(ids) {
                return ApiClient.request('/blacklist/batch-delete', {
                    method: 'POST',
                    body: { ids }
                });
            },

            async batchUnban(ids) {
                return ApiClient.request('/blacklist/batch-unban', {
                    method: 'POST',
                    body: { ids }
                });
            },

            async batchBan(ids, reason) {
                return ApiClient.request('/blacklist/batch-ban', {
                    method: 'POST',
                    body: { ids, reason }
                });
            },

            async export(format = 'csv') {
                return ApiClient.request(`/blacklist/export?format=${format}`, {
                    method: 'GET'
                });
            }
        }
    };

    window.AdminApi = ApiClient;

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

        showLoading(element, text = '加载中...') {
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

        confirm(message, title = '确认操作') {
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
                            <button class="cancel-btn px-4 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition">取消</button>
                            <button class="confirm-btn px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition">确定</button>
                        </div>
                    </div>
                `;

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
            if (num >= 1000000) {
                return (num / 1000000).toFixed(1) + 'M';
            }
            if (num >= 1000) {
                return (num / 1000).toFixed(1) + 'K';
            }
            return num.toString();
        },

        formatDate(dateString) {
            if (!dateString) return '-';
            const date = new Date(dateString);
            return date.toLocaleString('zh-CN', {
                year: 'numeric',
                month: '2-digit',
                day: '2-digit',
                hour: '2-digit',
                minute: '2-digit'
            });
        },

        formatDuration(seconds) {
            if (seconds < 60) return seconds + '秒';
            if (seconds < 3600) return Math.floor(seconds / 60) + '分钟';
            const hours = Math.floor(seconds / 3600);
            const mins = Math.floor((seconds % 3600) / 60);
            return `${hours}小时${mins}分钟`;
        }
    };

    window.AdminUI = UIController;

    const FormValidator = {
        rules: {
            required: (value) => value && value.trim() !== '',
            email: (value) => !value || /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value),
            ip: (value) => !value || /^(\d{1,3}\.){3}\d{1,3}(\/\d{1,2})?$/.test(value),
            ipv4: (value) => !value || /^(\d{1,3}\.){3}\d{1,3}$/.test(value),
            ipv4Cidr: (value) => !value || /^(\d{1,3}\.){3}\d{1,3}\/\d{1,2}$/.test(value),
            number: (value) => !value || !isNaN(parseFloat(value)),
            minLength: (value, min) => !value || value.length >= min,
            maxLength: (value, max) => !value || value.length <= max,
            range: (value, min, max) => {
                const num = parseFloat(value);
                return !isNaN(num) && num >= min && num <= max;
            }
        },

        messages: {
            required: '此字段为必填项',
            email: '请输入有效的邮箱地址',
            ip: '请输入有效的IP地址或CIDR格式',
            ipv4: '请输入有效的IPv4地址',
            ipv4Cidr: '请输入有效的CIDR格式地址',
            number: '请输入有效的数字',
            minLength: (min) => `最少需要 ${min} 个字符`,
            maxLength: (max) => `最多允许 ${max} 个字符`,
            range: (min, max) => `数值必须在 ${min} 到 ${max} 之间`
        },

        validateField(input, rules) {
            const value = input.value;
            const errors = [];

            for (const rule of rules) {
                if (typeof rule === 'string') {
                    if (!this.rules[rule](value)) {
                        errors.push(this.messages[rule]);
                    }
                } else if (typeof rule === 'object') {
                    const { type, params, message } = rule;
                    if (type === 'minLength' && !this.rules.minLength(value, params[0])) {
                        errors.push(message || this.messages.minLength(params[0]));
                    } else if (type === 'maxLength' && !this.rules.maxLength(value, params[0])) {
                        errors.push(message || this.messages.maxLength(params[0]));
                    } else if (type === 'range' && !this.rules.range(value, params[0], params[1])) {
                        errors.push(message || this.messages.range(params[0], params[1]));
                    } else if (type && !this.rules[type]) {
                        errors.push(message || `无效的验证规则: ${type}`);
                    } else if (type && !this.rules[type](value)) {
                        errors.push(message || this.messages[type] || '验证失败');
                    }
                }
            }

            return errors;
        },

        showError(input, errors) {
            const wrapper = input.closest('.form-group') || input.parentElement;
            const existingError = wrapper.querySelector('.error-message');

            input.classList.add('border-red-500');
            input.classList.remove('border-gray-300');

            if (existingError) {
                existingError.remove();
            }

            if (errors.length > 0) {
                const errorEl = document.createElement('p');
                errorEl.className = 'error-message text-red-500 text-sm mt-1';
                errorEl.textContent = errors[0];
                wrapper.appendChild(errorEl);
            }
        },

        clearError(input) {
            const wrapper = input.closest('.form-group') || input.parentElement;
            const existingError = wrapper.querySelector('.error-message');

            input.classList.remove('border-red-500');
            input.classList.add('border-gray-300');

            if (existingError) {
                existingError.remove();
            }
        },

        validateForm(form) {
            const inputs = form.querySelectorAll('[data-validate]');
            let isValid = true;

            inputs.forEach(input => {
                const rulesStr = input.dataset.validate;
                if (!rulesStr) return;

                try {
                    const rules = JSON.parse(rulesStr);
                    const errors = this.validateField(input, rules);

                    if (errors.length > 0) {
                        isValid = false;
                        this.showError(input, errors);
                    } else {
                        this.clearError(input);
                    }
                } catch (e) {
                    console.error('Invalid validate rule:', e);
                }
            });

            return isValid;
        },

        initFormValidation(form) {
            const inputs = form.querySelectorAll('[data-validate]');

            inputs.forEach(input => {
                input.addEventListener('blur', () => {
                    const rulesStr = input.dataset.validate;
                    if (!rulesStr) return;

                    try {
                        const rules = JSON.parse(rulesStr);
                        const errors = this.validateField(input, rules);

                        if (errors.length > 0) {
                            this.showError(input, errors);
                        } else {
                            this.clearError(input);
                        }
                    } catch (e) {
                        console.error('Invalid validate rule:', e);
                    }
                });

                input.addEventListener('input', () => {
                    const wrapper = input.closest('.form-group') || input.parentElement;
                    const existingError = wrapper.querySelector('.error-message');
                    if (existingError && input.classList.contains('border-red-500')) {
                        this.clearError(input);
                    }
                });
            });
        }
    };

    window.FormValidator = FormValidator;

    const ExportHelper = {
        downloadCSV(data, filename, columns) {
            if (!data || data.length === 0) {
                UIController.showToast('没有数据可导出', 'warning');
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
        },

        async exportTable(tableId, filename, columns) {
            const tbody = document.querySelector(`#${tableId} tbody`);
            if (!tbody) {
                UIController.showToast('表格不存在', 'error');
                return;
            }

            const rows = Array.from(tbody.querySelectorAll('tr'));
            const data = rows.map(row => {
                const item = {};
                columns.forEach((col, idx) => {
                    const cell = row.cells[idx];
                    if (cell) {
                        const text = cell.textContent.trim();
                        item[col.key] = text;
                    }
                });
                return item;
            }).filter(row => Object.values(row).some(v => v && v !== '-'));

            if (data.length === 0) {
                UIController.showToast('没有数据可导出', 'warning');
                return;
            }

            this.downloadCSV(data, filename, columns);
            UIController.showToast(`已导出 ${data.length} 条数据`, 'success');
        },

        async exportFromAPI(apiFn, filename, columns) {
            try {
                const data = await apiFn();
                if (Array.isArray(data)) {
                    this.downloadCSV(data, filename, columns);
                    UIController.showToast(`已导出 ${data.length} 条数据`, 'success');
                } else if (data.data && Array.isArray(data.data)) {
                    this.downloadCSV(data.data, filename, columns);
                    UIController.showToast(`已导出 ${data.data.length} 条数据`, 'success');
                } else {
                    UIController.showToast('数据格式不正确', 'error');
                }
            } catch (error) {
                UIController.showToast('导出失败: ' + error.message, 'error');
            }
        }
    };

    window.ExportHelper = ExportHelper;

    const ChartManager = {
        charts: {},

        initTrendChart(canvasId, data, options = {}) {
            const ctx = document.getElementById(canvasId);
            if (!ctx) return;

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
                            label: '验证成功',
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
                            label: '拦截',
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
            if (!ctx) return;

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

        initResultChart(canvasId, data, options = {}) {
            const ctx = document.getElementById(canvasId);
            if (!ctx) return;

            if (this.charts[canvasId]) {
                this.charts[canvasId].destroy();
            }

            const colorMap = {
                '通过': '#10b981',
                '失败': '#ef4444',
                '拦截': '#f59e0b',
                '通过率': '#3b82f6'
            };

            this.charts[canvasId] = new Chart(ctx, {
                type: options.type || 'bar',
                data: {
                    labels: data.map(d => d.result || d.label),
                    datasets: [{
                        label: options.label || '数量',
                        data: data.map(d => d.count || d.value),
                        backgroundColor: data.map(d => colorMap[d.result] || colorMap[d.label] || '#3b82f6'),
                        borderRadius: 6,
                        borderSkipped: false
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
                            padding: 12
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

        initBarChart(canvasId, data, options = {}) {
            const ctx = document.getElementById(canvasId);
            if (!ctx) return;

            if (this.charts[canvasId]) {
                this.charts[canvasId].destroy();
            }

            this.charts[canvasId] = new Chart(ctx, {
                type: 'bar',
                data: {
                    labels: data.labels || data.map(d => d.label),
                    datasets: [{
                        label: options.label || '数据',
                        data: data.values || data.map(d => d.value),
                        backgroundColor: options.colors || 'rgba(59, 130, 246, 0.5)',
                        borderColor: options.borderColors || 'rgba(59, 130, 246, 1)',
                        borderWidth: 2
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                        legend: {
                            display: false
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

        updateChart(canvasId, newData, type = 'trend') {
            const chart = this.charts[canvasId];
            if (!chart) return;

            if (type === 'trend') {
                chart.data.labels = newData.map(d => d.time);
                chart.data.datasets[0].data = newData.map(d => d.verified);
                chart.data.datasets[1].data = newData.map(d => d.rejected);
            } else if (type === 'distribution') {
                chart.data.labels = newData.map(d => d.label || d.type);
                chart.data.datasets[0].data = newData.map(d => d.count || d.value);
            } else if (type === 'bar') {
                chart.data.labels = newData.labels || newData.map(d => d.label);
                chart.data.datasets[0].data = newData.values || newData.map(d => d.value);
            }

            chart.update('active');
        },

        destroyChart(canvasId) {
            if (this.charts[canvasId]) {
                this.charts[canvasId].destroy();
                delete this.charts[canvasId];
            }
        },

        destroyAll() {
            Object.keys(this.charts).forEach(id => this.destroyChart(id));
        }
    };

    window.AdminCharts = ChartManager;

    const Pagination = {
        currentPage: 1,
        pageSize: 20,
        total: 0,
        container: null,
        onPageChange: null,
        maxButtons: 5,

        init(containerId, pageSize, callback) {
            this.container = document.getElementById(containerId);
            this.pageSize = pageSize;
            this.onPageChange = callback;
            this.currentPage = 1;
            this.render();
        },

        setTotal(total) {
            this.total = total;
            this.render();
        },

        setPage(page) {
            const totalPages = Math.ceil(this.total / this.pageSize) || 1;
            this.currentPage = Math.max(1, Math.min(page, totalPages));
            this.render();
            if (this.onPageChange) {
                this.onPageChange(this.currentPage);
            }
        },

        setPageSize(size) {
            this.pageSize = size;
            this.currentPage = 1;
            this.render();
            if (this.onPageChange) {
                this.onPageChange(this.currentPage);
            }
        },

        goToFirst() {
            this.setPage(1);
        },

        goToLast() {
            this.setPage(Math.ceil(this.total / this.pageSize) || 1);
        },

        prev() {
            if (this.currentPage > 1) {
                this.setPage(this.currentPage - 1);
            }
        },

        next() {
            const totalPages = Math.ceil(this.total / this.pageSize) || 1;
            if (this.currentPage < totalPages) {
                this.setPage(this.currentPage + 1);
            }
        },

        render() {
            if (!this.container) return;

            const totalPages = Math.ceil(this.total / this.pageSize) || 1;
            if (totalPages <= 1 && this.total === 0) {
                this.container.innerHTML = '';
                return;
            }

            const startRecord = (this.currentPage - 1) * this.pageSize + 1;
            const endRecord = Math.min(this.currentPage * this.pageSize, this.total);

            let html = `<div class="flex flex-col sm:flex-row items-center justify-between gap-4">
                <div class="text-sm text-gray-600">
                    显示 ${startRecord}-${endRecord} 条，共 ${this.total} 条
                </div>
                <div class="flex items-center space-x-1">`;

            html += `<button class="px-3 py-1.5 rounded ${this.currentPage === 1 ? 'bg-gray-100 text-gray-400 cursor-not-allowed' : 'bg-gray-200 hover:bg-gray-300 text-gray-700'}" ${this.currentPage === 1 ? 'disabled' : ''} onclick="AdminUI.Pagination?.goToFirst()" title="首页">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 19l-7-7 7-7m8 14l-7-7 7-7"></path></svg>
            </button>`;

            html += `<button class="px-3 py-1.5 rounded ${this.currentPage === 1 ? 'bg-gray-100 text-gray-400 cursor-not-allowed' : 'bg-gray-200 hover:bg-gray-300 text-gray-700'}" ${this.currentPage === 1 ? 'disabled' : ''} onclick="AdminUI.Pagination?.prev()">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"></path></svg>
            </button>`;

            let startPage = Math.max(1, this.currentPage - Math.floor(this.maxButtons / 2));
            let endPage = Math.min(totalPages, startPage + this.maxButtons - 1);

            if (endPage - startPage < this.maxButtons - 1) {
                startPage = Math.max(1, endPage - this.maxButtons + 1);
            }

            if (startPage > 1) {
                html += `<button class="px-3 py-1.5 rounded bg-gray-200 hover:bg-gray-300 text-gray-700" onclick="AdminUI.Pagination?.setPage(1)">1</button>`;
                if (startPage > 2) {
                    html += '<span class="px-2 text-gray-400">...</span>';
                }
            }

            for (let i = startPage; i <= endPage; i++) {
                html += `<button class="px-3 py-1.5 rounded ${i === this.currentPage ? 'bg-blue-500 text-white' : 'bg-gray-200 hover:bg-gray-300 text-gray-700'}" onclick="AdminUI.Pagination?.setPage(${i})">${i}</button>`;
            }

            if (endPage < totalPages) {
                if (endPage < totalPages - 1) {
                    html += '<span class="px-2 text-gray-400">...</span>';
                }
                html += `<button class="px-3 py-1.5 rounded bg-gray-200 hover:bg-gray-300 text-gray-700" onclick="AdminUI.Pagination?.setPage(${totalPages})">${totalPages}</button>`;
            }

            html += `<button class="px-3 py-1.5 rounded ${this.currentPage === totalPages ? 'bg-gray-100 text-gray-400 cursor-not-allowed' : 'bg-gray-200 hover:bg-gray-300 text-gray-700'}" ${this.currentPage === totalPages ? 'disabled' : ''} onclick="AdminUI.Pagination?.next()">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"></path></svg>
            </button>`;

            html += `<button class="px-3 py-1.5 rounded ${this.currentPage === totalPages ? 'bg-gray-100 text-gray-400 cursor-not-allowed' : 'bg-gray-200 hover:bg-gray-300 text-gray-700'}" ${this.currentPage === totalPages ? 'disabled' : ''} onclick="AdminUI.Pagination?.goToLast()" title="末页">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 5l7 7-7 7M5 5l7 7-7 7"></path></svg>
            </button>`;

            html += `</div><div class="flex items-center space-x-2">
                <span class="text-sm text-gray-500">每页</span>
                <select class="px-2 py-1 border border-gray-300 rounded text-sm" onchange="AdminUI.Pagination?.setPageSize(parseInt(this.value))">
                    <option value="10" ${this.pageSize === 10 ? 'selected' : ''}>10</option>
                    <option value="20" ${this.pageSize === 20 ? 'selected' : ''}>20</option>
                    <option value="50" ${this.pageSize === 50 ? 'selected' : ''}>50</option>
                    <option value="100" ${this.pageSize === 100 ? 'selected' : ''}>100</option>
                </select>
                <span class="text-sm text-gray-500">条</span>
            </div></div>`;

            this.container.innerHTML = html;
        }
    };

    UIController.Pagination = Pagination;

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
        },

        refreshAll() {
            Object.keys(this.callbacks).forEach(key => {
                if (this.callbacks[key]) {
                    this.callbacks[key]();
                }
            });
        },

        setInterval(key, interval) {
            if (this.timers[key] && this.intervals[key] !== interval) {
                this.intervals[key] = interval;
                this.stop(key);
                this.start(key, this.callbacks[key], interval);
            }
        }
    };

    window.RealtimeUpdater = RealtimeUpdater;

    const BatchOperations = {
        selectedItems: new Set(),

        init(tableId, checkboxClass = 'row-checkbox') {
            this.tableId = tableId;
            this.checkboxClass = checkboxClass;
            this.selectedItems.clear();
            this.bindEvents();
        },

        bindEvents() {
            const table = document.getElementById(this.tableId);
            if (!table) return;

            table.addEventListener('change', (e) => {
                if (e.target.classList.contains(this.checkboxClass)) {
                    const id = e.target.dataset.id;
                    if (e.target.checked) {
                        this.selectedItems.add(id);
                    } else {
                        this.selectedItems.delete(id);
                    }
                    this.updateMasterCheckbox();
                    this.updateActionBar();
                }
            });

            const masterCheckbox = table.querySelector('.master-checkbox');
            if (masterCheckbox) {
                masterCheckbox.addEventListener('change', (e) => {
                    this.toggleAll(e.target.checked);
                });
            }
        },

        toggleAll(checked) {
            const checkboxes = document.querySelectorAll(`#${this.tableId} .${this.checkboxClass}`);
            checkboxes.forEach(cb => {
                cb.checked = checked;
                if (checked) {
                    this.selectedItems.add(cb.dataset.id);
                } else {
                    this.selectedItems.delete(cb.dataset.id);
                }
            });
            this.updateActionBar();
        },

        updateMasterCheckbox() {
            const table = document.getElementById(this.tableId);
            if (!table) return;

            const checkboxes = table.querySelectorAll(`.${this.checkboxClass}`);
            const masterCheckbox = table.querySelector('.master-checkbox');

            if (masterCheckbox && checkboxes.length > 0) {
                const allChecked = Array.from(checkboxes).every(cb => cb.checked);
                const someChecked = Array.from(checkboxes).some(cb => cb.checked);

                masterCheckbox.checked = allChecked;
                masterCheckbox.indeterminate = someChecked && !allChecked;
            }
        },

        updateActionBar() {
            const actionBar = document.getElementById('batch-action-bar');
            const countEl = document.getElementById('selected-count');

            if (actionBar) {
                actionBar.classList.toggle('hidden', this.selectedItems.size === 0);
            }
            if (countEl) {
                countEl.textContent = this.selectedItems.size;
            }
        },

        getSelected() {
            return Array.from(this.selectedItems);
        },

        clearSelection() {
            this.selectedItems.clear();
            const checkboxes = document.querySelectorAll(`#${this.tableId} .${this.checkboxClass}`);
            checkboxes.forEach(cb => {
                cb.checked = false;
            });
            this.updateMasterCheckbox();
            this.updateActionBar();
        },

        selectAll(items) {
            items.forEach(item => this.selectedItems.add(item.id || item));
            this.updateMasterCheckbox();
            this.updateActionBar();
        }
    };

    window.BatchOperations = BatchOperations;

    document.addEventListener('DOMContentLoaded', function() {
        const loginForm = document.getElementById('login-form');
        if (loginForm) {
            loginForm.addEventListener('submit', async function(e) {
                e.preventDefault();

                const username = document.getElementById('username');
                const password = document.getElementById('password');
                const submitBtn = document.getElementById('login-btn');
                const errorDiv = document.getElementById('login-error');

                if (!username.value.trim() || !password.value) {
                    errorDiv.textContent = '请输入用户名和密码';
                    errorDiv.classList.remove('hidden');
                    return;
                }

                UIController.showLoading(submitBtn);
                errorDiv.classList.add('hidden');

                try {
                    const result = await ApiClient.auth.login(username.value.trim(), password.value);
                    localStorage.setItem('admin_token', result.token);
                    window.location.href = '/admin/dashboard';
                } catch (error) {
                    errorDiv.textContent = error.message || '登录失败，请检查用户名和密码';
                    errorDiv.classList.remove('hidden');
                } finally {
                    UIController.hideLoading(submitBtn);
                }
            });
        }

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

        document.querySelectorAll('[data-page]').forEach(link => {
            link.addEventListener('click', function(e) {
                e.preventDefault();
                const page = this.dataset.page;
                window.location.href = `/admin/${page}`;
            });
        });

        const logoutBtn = document.getElementById('logout-btn');
        if (logoutBtn) {
            logoutBtn.addEventListener('click', async function() {
                const confirmed = await UIController.confirm('确定要退出登录吗？');
                if (confirmed) {
                    try {
                        await ApiClient.auth.logout();
                    } catch (e) {}
                    localStorage.removeItem('admin_token');
                    window.location.href = '/admin/login';
                }
            });
        }
    });

    window.addEventListener('beforeunload', function() {
        AdminCharts.destroyAll();
        RealtimeUpdater.stopAll();
    });

    window.addEventListener('hashchange', function() {
        RealtimeUpdater.refreshAll();
    });

})();
