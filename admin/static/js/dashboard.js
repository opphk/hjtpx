class DashboardManager {
    constructor() {
        this.refreshInterval = 30000;
        this.autoRefreshEnabled = false;
        this.refreshTimer = null;
        this.realtimeCharts = [];
        this.notifications = [];
        this.init();
    }

    init() {
        this.loadDashboardData();
        this.initRealtimeUpdates();
        this.initNotifications();
    }

    async loadDashboardData() {
        await Promise.all([
            this.loadStats(),
            this.loadCharts(),
            this.loadRecentActivity(),
            this.loadQuickStats()
        ]);
    }

    async loadStats() {
        try {
            const data = await auth.request('/admin/dashboard/stats');
            if (data.code === 0 && data.data) {
                this.updateStats(data.data);
            } else {
                this.loadMockStats();
            }
        } catch (error) {
            this.loadMockStats();
        }
    }

    loadMockStats() {
        const mockStats = {
            totalUsers: 12456,
            totalApps: 156,
            totalRequests: 8234567,
            totalErrors: 1234,
            successRate: 99.2,
            avgResponseTime: 45,
            activeUsers: 892,
            qps: 1250
        };
        this.updateStats(mockStats);
    }

    updateStats(stats) {
        this.animateNumber('totalUsers', stats.totalUsers || 0);
        this.animateNumber('totalApps', stats.totalApps || 0);
        this.animateNumber('totalRequests', stats.totalRequests || 0);
        this.animateNumber('totalErrors', stats.totalErrors || 0);

        if (document.getElementById('successRate')) {
            this.animateNumber('successRate', stats.successRate || 0, { suffix: '%', decimals: 1 });
        }
        if (document.getElementById('avgResponseTime')) {
            this.animateNumber('avgResponseTime', stats.avgResponseTime || 0, { suffix: 'ms' });
        }
        if (document.getElementById('activeUsers')) {
            this.animateNumber('activeUsers', stats.activeUsers || 0);
        }
        if (document.getElementById('qps')) {
            this.animateNumber('qps', stats.qps || 0);
        }

        this.updateGrowthIndicators(stats);
    }

    updateGrowthIndicators(stats) {
        const elements = {
            'userGrowth': stats.userGrowth || 12.5,
            'appGrowth': stats.appGrowth || 8.2,
            'requestGrowth': stats.requestGrowth || 23.1,
            'errorChange': stats.errorChange || -5.7
        };

        for (const [id, value] of Object.entries(elements)) {
            const el = document.getElementById(id);
            if (el) {
                const isPositive = value > 0;
                const isGood = id === 'errorChange' ? value < 0 : isPositive;
                el.textContent = (value > 0 ? '+' : '') + value.toFixed(1) + '%';
                el.className = isGood ? 'text-success' : 'text-danger';
            }
        }
    }

    animateNumber(elementId, target, options = {}) {
        const element = document.getElementById(elementId);
        if (!element) return;

        const duration = options.duration || 1500;
        const start = 0;
        const startTime = performance.now();
        const decimals = options.decimals || 0;
        const suffix = options.suffix || '';
        const prefix = options.prefix || '';

        const easeOutQuart = (x) => 1 - Math.pow(1 - x, 4);

        const animate = (currentTime) => {
            const elapsed = currentTime - startTime;
            const progress = Math.min(elapsed / duration, 1);
            const easedProgress = easeOutQuart(progress);
            const current = start + (target - start) * easedProgress;

            let formattedValue;
            if (target >= 1000000) {
                formattedValue = (current / 1000000).toFixed(1) + 'M';
            } else if (target >= 1000 && !options.noFormat) {
                formattedValue = (current / 1000).toFixed(1) + 'K';
            } else {
                formattedValue = current.toFixed(decimals);
            }

            element.textContent = prefix + formattedValue + suffix;

            if (progress < 1) {
                requestAnimationFrame(animate);
            }
        };

        requestAnimationFrame(animate);
    }

    async loadCharts() {
        try {
            const data = await auth.request('/admin/dashboard/charts');
            if (data.code === 0 && data.data) {
                this.renderCharts(data.data);
            } else {
                this.renderMockCharts();
            }
        } catch (error) {
            this.renderMockCharts();
        }
    }

    renderCharts(chartData) {
        if (chartData.trend24h) {
            this.renderTrendChart(chartData.trend24h);
        }
        if (chartData.typeDistribution) {
            this.renderTypeDistributionChart(chartData.typeDistribution);
        }
        if (chartData.riskDistribution) {
            this.renderRiskDistributionChart(chartData.riskDistribution);
        }
    }

    renderMockCharts() {
        const hours = [];
        const now = new Date();
        for (let i = 23; i >= 0; i--) {
            const hour = new Date(now.getTime() - i * 3600000);
            hours.push(`${hour.getHours()}:00`);
        }

        const trendData = {
            labels: hours,
            datasets: [{
                label: '验证次数',
                data: hours.map(() => Math.floor(Math.random() * 5000) + 2000),
                borderColor: 'rgba(59, 130, 246, 1)',
                backgroundColor: 'rgba(59, 130, 246, 0.1)',
                fill: true,
                tension: 0.4
            }]
        };

        this.renderTrendChart(trendData);

        const typeData = {
            labels: ['滑块验证', '点选验证', '图形验证', '行为验证'],
            datasets: [{
                data: [35, 25, 20, 20],
                backgroundColor: [
                    'rgba(59, 130, 246, 0.8)',
                    'rgba(16, 185, 129, 0.8)',
                    'rgba(245, 158, 11, 0.8)',
                    'rgba(139, 92, 246, 0.8)'
                ]
            }]
        };

        this.renderTypeDistributionChart(typeData);

        const riskData = {
            labels: ['低风险', '中风险', '高风险', '极高风险'],
            datasets: [{
                label: '风险分布',
                data: [65, 20, 10, 5],
                backgroundColor: [
                    'rgba(16, 185, 129, 0.8)',
                    'rgba(245, 158, 11, 0.8)',
                    'rgba(249, 115, 22, 0.8)',
                    'rgba(239, 68, 68, 0.8)'
                ]
            }]
        };

        this.renderRiskDistributionChart(riskData);

        const comparisonData = {
            labels: ['1月', '2月', '3月', '4月', '5月', '6月'],
            datasets: [
                {
                    label: '本周',
                    data: [12000, 15000, 18000, 22000, 25000, 28000],
                    borderColor: 'rgba(59, 130, 246, 1)',
                    backgroundColor: 'rgba(59, 130, 246, 0.1)',
                    fill: true,
                    tension: 0.4
                },
                {
                    label: '上周',
                    data: [10000, 12000, 14000, 16000, 18000, 20000],
                    borderColor: 'rgba(139, 92, 246, 1)',
                    backgroundColor: 'rgba(139, 92, 246, 0.1)',
                    fill: true,
                    tension: 0.4
                }
            ]
        };

        this.renderComparisonChart(comparisonData);
    }

    renderTrendChart(data) {
        const canvas = document.getElementById('trendChart');
        if (!canvas) return;

        const container = canvas.parentElement;
        canvas.remove();

        const newCanvas = document.createElement('canvas');
        newCanvas.id = 'trendChart';
        container.appendChild(newCanvas);

        chartManager.createLineChart('trendChart', data, {
            showLegend: false,
            chartOptions: {
                plugins: {
                    title: {
                        display: true,
                        text: '24小时验证趋势',
                        font: { size: 16, weight: 'bold' }
                    }
                }
            }
        });
    }

    renderTypeDistributionChart(data) {
        const canvas = document.getElementById('typeDistributionChart');
        if (!canvas) return;

        const container = canvas.parentElement;
        canvas.remove();

        const newCanvas = document.createElement('canvas');
        newCanvas.id = 'typeDistributionChart';
        container.appendChild(newCanvas);

        chartManager.createPieChart('typeDistributionChart', data, {
            legendPosition: 'bottom',
            cutout: '50%'
        });
    }

    renderRiskDistributionChart(data) {
        const canvas = document.getElementById('riskDistributionChart');
        if (!canvas) return;

        const container = canvas.parentElement;
        canvas.remove();

        const newCanvas = document.createElement('canvas');
        newCanvas.id = 'riskDistributionChart';
        container.appendChild(newCanvas);

        chartManager.createBarChart('riskDistributionChart', data);
    }

    renderComparisonChart(data) {
        const canvas = document.getElementById('comparisonChart');
        if (!canvas) return;

        const container = canvas.parentElement;
        canvas.remove();

        const newCanvas = document.createElement('canvas');
        newCanvas.id = 'comparisonChart';
        container.appendChild(newCanvas);

        chartManager.createLineChart('comparisonChart', data, {
            chartOptions: {
                plugins: {
                    title: {
                        display: true,
                        text: '7天趋势对比',
                        font: { size: 16, weight: 'bold' }
                    }
                }
            }
        });
    }

    async loadRecentActivity() {
        try {
            const data = await auth.request('/admin/dashboard/activity');
            if (data.code === 0 && data.data) {
                this.renderActivityTable(data.data);
            } else {
                this.renderMockActivity();
            }
        } catch (error) {
            this.renderMockActivity();
        }
    }

    renderMockActivity() {
        const mockActivities = [
            { time: '2024-01-15 14:32:18', event: '用户登录', user: 'admin', status: 'success' },
            { time: '2024-01-15 14:28:45', event: '创建应用', user: 'developer1', status: 'success' },
            { time: '2024-01-15 14:25:12', event: 'API请求失败', user: 'app_001', status: 'error' },
            { time: '2024-01-15 14:20:33', event: '更新配置', user: 'admin', status: 'success' },
            { time: '2024-01-15 14:15:09', event: '用户注册', user: 'new_user', status: 'success' },
            { time: '2024-01-15 14:10:42', event: '删除应用', user: 'admin', status: 'warning' },
            { time: '2024-01-15 14:05:18', event: '批量导出', user: 'analyst', status: 'success' }
        ];
        this.renderActivityTable(mockActivities);
    }

    renderActivityTable(activities) {
        const tbody = document.getElementById('recentActivity');
        if (!tbody) return;

        tbody.innerHTML = activities.map(activity => `
            <tr>
                <td><small>${activity.time}</small></td>
                <td>${activity.event}</td>
                <td><span class="badge bg-secondary">${activity.user}</span></td>
                <td><span class="badge ${this.getStatusBadgeClass(activity.status)}">${this.getStatusText(activity.status)}</span></td>
            </tr>
        `).join('');
    }

    getStatusBadgeClass(status) {
        const classMap = {
            success: 'bg-success',
            error: 'bg-danger',
            warning: 'bg-warning text-dark',
            pending: 'bg-info',
            info: 'bg-primary'
        };
        return classMap[status] || 'bg-secondary';
    }

    getStatusText(status) {
        const textMap = {
            success: '成功',
            error: '失败',
            warning: '警告',
            pending: '处理中',
            info: '信息'
        };
        return textMap[status] || status;
    }

    async loadQuickStats() {
        try {
            const data = await auth.request('/admin/dashboard/quick-stats');
            if (data.code === 0 && data.data) {
                this.updateQuickStats(data.data);
            }
        } catch (error) {
            console.warn('Failed to load quick stats');
        }
    }

    updateQuickStats(stats) {
        if (stats.topApps) {
            this.renderTopApps(stats.topApps);
        }
        if (stats.systemStatus) {
            this.updateSystemStatus(stats.systemStatus);
        }
    }

    renderTopApps(apps) {
        const container = document.getElementById('topApps');
        if (!container) return;

        container.innerHTML = apps.slice(0, 5).map((app, index) => `
            <div class="d-flex justify-content-between align-items-center mb-2">
                <span class="badge bg-${['primary', 'success', 'info', 'warning', 'secondary'][index]}">${index + 1}</span>
                <span class="flex-grow-1 mx-2">${app.name}</span>
                <small class="text-muted">${this.formatNumber(app.requests)} 请求</small>
            </div>
        `).join('');
    }

    updateSystemStatus(status) {
        const elements = {
            'dbStatus': status.database,
            'redisStatus': status.redis,
            'apiStatus': status.api,
            'storageStatus': status.storage
        };

        for (const [id, value] of Object.entries(elements)) {
            const el = document.getElementById(id);
            if (el) {
                const badge = el.querySelector('.badge');
                if (badge) {
                    badge.className = `badge bg-${value === 'healthy' || value === 'up' ? 'success' : value === 'degraded' ? 'warning' : 'danger'} rounded-pill`;
                }
            }
        }
    }

    initRealtimeUpdates() {
        this.startRealtimeUpdates();
    }

    startRealtimeUpdates() {
        if (this.refreshTimer) {
            clearInterval(this.refreshTimer);
        }

        this.refreshTimer = setInterval(() => {
            if (this.autoRefreshEnabled) {
                this.loadDashboardData();
            }
        }, this.refreshInterval);
    }

    stopRealtimeUpdates() {
        if (this.refreshTimer) {
            clearInterval(this.refreshTimer);
            this.refreshTimer = null;
        }
    }

    toggleAutoRefresh() {
        this.autoRefreshEnabled = !this.autoRefreshEnabled;
        const btn = document.getElementById('autoRefreshBtn');
        if (btn) {
            btn.className = this.autoRefreshEnabled ? 'btn btn-success btn-sm' : 'btn btn-outline-secondary btn-sm';
            btn.innerHTML = this.autoRefreshEnabled ? '<i class="fas fa-pause me-1"></i>暂停刷新' : '<i class="fas fa-play me-1"></i>自动刷新';
        }
        this.showNotification('自动刷新已' + (this.autoRefreshEnabled ? '启用' : '禁用'), this.autoRefreshEnabled ? 'success' : 'info');
    }

    initNotifications() {
        this.loadNotifications();
    }

    async loadNotifications() {
        try {
            const data = await auth.request('/admin/notifications');
            if (data.code === 0 && data.data) {
                this.notifications = data.data;
                this.updateNotificationBadge();
            }
        } catch (error) {
            this.notifications = [];
        }
    }

    updateNotificationBadge() {
        const badge = document.getElementById('notificationBadge');
        if (badge) {
            const unreadCount = this.notifications.filter(n => !n.read).length;
            badge.textContent = unreadCount;
            badge.style.display = unreadCount > 0 ? 'inline' : 'none';
        }
    }

    showNotification(message, type = 'info') {
        const container = document.getElementById('notificationContainer') || this.createNotificationContainer();

        const notification = document.createElement('div');
        notification.className = `alert alert-${type} alert-dismissible fade show`;
        notification.style.cssText = 'position: fixed; top: 80px; right: 20px; z-index: 9999; min-width: 300px;';
        notification.innerHTML = `
            <i class="fas fa-${this.getNotificationIcon(type)} me-2"></i>
            ${message}
            <button type="button" class="btn-close" data-bs-dismiss="alert"></button>
        `;

        container.appendChild(notification);

        setTimeout(() => {
            notification.classList.remove('show');
            setTimeout(() => notification.remove(), 150);
        }, 3000);
    }

    createNotificationContainer() {
        const container = document.createElement('div');
        container.id = 'notificationContainer';
        container.style.cssText = 'position: fixed; top: 80px; right: 20px; z-index: 9999;';
        document.body.appendChild(container);
        return container;
    }

    getNotificationIcon(type) {
        const iconMap = {
            success: 'check-circle',
            danger: 'exclamation-circle',
            warning: 'exclamation-triangle',
            info: 'info-circle',
            primary: 'info-circle'
        };
        return iconMap[type] || 'info-circle';
    }

    formatNumber(num) {
        if (num >= 1000000) {
            return (num / 1000000).toFixed(1) + 'M';
        } else if (num >= 1000) {
            return (num / 1000).toFixed(1) + 'K';
        }
        return num.toString();
    }

    refresh() {
        this.loadDashboardData();
        this.showNotification('数据已刷新', 'success');
    }

    destroy() {
        this.stopRealtimeUpdates();
        chartManager.destroyAllCharts();
    }
}

const dashboardManager = new DashboardManager();
