(function() {
    'use strict';

    const API_BASE = '/api/admin';

    const ApiClient = {
        async request(endpoint, options = {}) {
            const url = API_BASE + endpoint;
            const config = {
                headers: {
                    'Content-Type': 'application/json',
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
            async list(page = 1, pageSize = 20, search = '') {
                return ApiClient.request(`/whitelist?page=${page}&page_size=${pageSize}&search=${encodeURIComponent(search)}`, {
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
            async list(page = 1, pageSize = 20, search = '') {
                return ApiClient.request(`/blacklist?page=${page}&page_size=${pageSize}&search=${encodeURIComponent(search)}`, {
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

            toast.className = `${colors[type]} text-white px-6 py-3 rounded-lg shadow-lg mb-2 transform transition-all duration-300 translate-x-full`;
            toast.textContent = message;
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
            container.className = 'fixed top-4 right-4 z-50';
            document.body.appendChild(container);
            return container;
        },

        showLoading(element) {
            if (!element) return;
            element.dataset.originalContent = element.innerHTML;
            element.disabled = true;
            element.innerHTML = '<svg class="animate-spin h-5 w-5 inline mr-2" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>加载中...';
        },

        hideLoading(element) {
            if (!element || !element.dataset.originalContent) return;
            element.disabled = false;
            element.innerHTML = element.dataset.originalContent;
            delete element.dataset.originalContent;
        },

        confirm(message) {
            return new Promise((resolve) => {
                const overlay = document.createElement('div');
                overlay.className = 'fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50';
                overlay.innerHTML = `
                    <div class="bg-white rounded-xl p-6 max-w-md mx-4 shadow-2xl">
                        <p class="text-gray-800 text-lg mb-6">${message}</p>
                        <div class="flex justify-end space-x-3">
                            <button class="cancel-btn px-4 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition">取消</button>
                            <button class="confirm-btn px-4 py-2 bg-red-500 text-white rounded-lg hover:bg-red-600 transition">确定</button>
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
            });
        },

        formatNumber(num) {
            if (num >= 1000000) {
                return (num / 1000000).toFixed(1) + 'M';
            }
            if (num >= 1000) {
                return (num / 1000).toFixed(1) + 'K';
            }
            return num.toString();
        },

        formatDate(dateString) {
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

    const ChartManager = {
        charts: {},

        initTrendChart(canvasId, data) {
            const ctx = document.getElementById(canvasId);
            if (!ctx) return;

            if (this.charts[canvasId]) {
                this.charts[canvasId].destroy();
            }

            const labels = data.map(d => d.time);
            const verifiedData = data.map(d => d.verified);
            const rejectedData = data.map(d => d.rejected);

            this.charts[canvasId] = new Chart(ctx, {
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
                            tension: 0.4
                        },
                        {
                            label: '拦截',
                            data: rejectedData,
                            borderColor: '#ef4444',
                            backgroundColor: 'rgba(239, 68, 68, 0.1)',
                            fill: true,
                            tension: 0.4
                        }
                    ]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                        legend: {
                            position: 'top'
                        }
                    },
                    scales: {
                        y: {
                            beginAtZero: true
                        }
                    }
                }
            });
        },

        initDistributionChart(canvasId, data) {
            const ctx = document.getElementById(canvasId);
            if (!ctx) return;

            if (this.charts[canvasId]) {
                this.charts[canvasId].destroy();
            }

            this.charts[canvasId] = new Chart(ctx, {
                type: 'doughnut',
                data: {
                    labels: data.map(d => d.type),
                    datasets: [{
                        data: data.map(d => d.count),
                        backgroundColor: [
                            '#3b82f6',
                            '#8b5cf6',
                            '#ec4899',
                            '#f59e0b',
                            '#10b981'
                        ]
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                        legend: {
                            position: 'right'
                        }
                    }
                }
            });
        },

        initResultChart(canvasId, data) {
            const ctx = document.getElementById(canvasId);
            if (!ctx) return;

            if (this.charts[canvasId]) {
                this.charts[canvasId].destroy();
            }

            this.charts[canvasId] = new Chart(ctx, {
                type: 'bar',
                data: {
                    labels: data.map(d => d.result),
                    datasets: [{
                        label: '数量',
                        data: data.map(d => d.count),
                        backgroundColor: data.map(d => d.result === '通过' ? '#10b981' : '#ef4444')
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
                            beginAtZero: true
                        }
                    }
                }
            });
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

        init(containerId, pageSize, callback) {
            this.container = document.getElementById(containerId);
            this.pageSize = pageSize;
            this.onPageChange = callback;
            this.render();
        },

        setTotal(total) {
            this.total = total;
            this.render();
        },

        setPage(page) {
            this.currentPage = page;
            this.render();
            if (this.onPageChange) {
                this.onPageChange(page);
            }
        },

        render() {
            if (!this.container) return;

            const totalPages = Math.ceil(this.total / this.pageSize);
            if (totalPages <= 1) {
                this.container.innerHTML = '';
                return;
            }

            let html = '<div class="flex items-center justify-center space-x-2">';

            html += `<button class="px-3 py-1 rounded ${this.currentPage === 1 ? 'bg-gray-300 cursor-not-allowed' : 'bg-gray-200 hover:bg-gray-300'}" ${this.currentPage === 1 ? 'disabled' : ''} onclick="AdminUI.Pagination?.prev()">上一页</button>`;

            const maxButtons = 5;
            let startPage = Math.max(1, this.currentPage - Math.floor(maxButtons / 2));
            let endPage = Math.min(totalPages, startPage + maxButtons - 1);

            if (endPage - startPage < maxButtons - 1) {
                startPage = Math.max(1, endPage - maxButtons + 1);
            }

            if (startPage > 1) {
                html += `<button class="px-3 py-1 rounded bg-gray-200 hover:bg-gray-300" onclick="AdminUI.Pagination?.setPage(1)">1</button>`;
                if (startPage > 2) {
                    html += '<span class="px-2">...</span>';
                }
            }

            for (let i = startPage; i <= endPage; i++) {
                html += `<button class="px-3 py-1 rounded ${i === this.currentPage ? 'bg-blue-500 text-white' : 'bg-gray-200 hover:bg-gray-300'}" onclick="AdminUI.Pagination?.setPage(${i})">${i}</button>`;
            }

            if (endPage < totalPages) {
                if (endPage < totalPages - 1) {
                    html += '<span class="px-2">...</span>';
                }
                html += `<button class="px-3 py-1 rounded bg-gray-200 hover:bg-gray-300" onclick="AdminUI.Pagination?.setPage(${totalPages})">${totalPages}</button>`;
            }

            html += `<button class="px-3 py-1 rounded ${this.currentPage === totalPages ? 'bg-gray-300 cursor-not-allowed' : 'bg-gray-200 hover:bg-gray-300'}" ${this.currentPage === totalPages ? 'disabled' : ''} onclick="AdminUI.Pagination?.next()">下一页</button>`;

            html += `<span class="ml-4 text-gray-600">共 ${this.total} 条</span>`;

            html += '</div>';

            this.container.innerHTML = html;
        },

        prev() {
            if (this.currentPage > 1) {
                this.setPage(this.currentPage - 1);
            }
        },

        next() {
            const totalPages = Math.ceil(this.total / this.pageSize);
            if (this.currentPage < totalPages) {
                this.setPage(this.currentPage + 1);
            }
        }
    };

    UIController.Pagination = Pagination;

    document.addEventListener('DOMContentLoaded', function() {
        const loginForm = document.getElementById('login-form');
        if (loginForm) {
            loginForm.addEventListener('submit', async function(e) {
                e.preventDefault();

                const username = document.getElementById('username').value.trim();
                const password = document.getElementById('password').value;
                const submitBtn = document.getElementById('login-btn');
                const errorDiv = document.getElementById('login-error');

                if (!username || !password) {
                    errorDiv.textContent = '请输入用户名和密码';
                    errorDiv.classList.remove('hidden');
                    return;
                }

                UIController.showLoading(submitBtn);
                errorDiv.classList.add('hidden');

                try {
                    const result = await ApiClient.auth.login(username, password);
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
    });

})();
