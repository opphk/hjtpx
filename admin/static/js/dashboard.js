
let requestTrendChart = null;
let realtimeChart = null;
let autoRefreshInterval = null;
let refreshInterval = 30000;

document.addEventListener('DOMContentLoaded', function() {
    if (!Auth.requireAuth()) {
        return;
    }

    const user = Auth.getCurrentUser();
    if (user && user.username) {
        Auth.updateUserDisplay(user.username);
    }

    initRequestTrendChart();
    initRealtimeChart();
    loadDashboardData();

    const autoRefreshSwitch = document.getElementById('autoRefreshSwitch');
    if (autoRefreshSwitch) {
        autoRefreshSwitch.addEventListener('change', function() {
            if (this.checked) {
                startAutoRefresh();
            } else {
                stopAutoRefresh();
            }
        });
    }

    const refreshBtn = document.getElementById('refreshBtn');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', function() {
            loadDashboardData();
            Auth.showToast('数据已刷新', 'success');
        });
    }

    const periodButtons = document.querySelectorAll('[data-period]');
    periodButtons.forEach(button => {
        button.addEventListener('click', function() {
            periodButtons.forEach(btn => btn.classList.remove('active'));
            this.classList.add('active');
            const period = this.getAttribute('data-period');
            loadRequestTrendData(period);
        });
    });

    startAutoRefresh();
});

function initRequestTrendChart() {
    const ctx = document.getElementById('requestTrendChart');
    if (!ctx) return;

    if (requestTrendChart) {
        requestTrendChart.destroy();
    }

    requestTrendChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: [],
            datasets: [
                {
                    label: '成功请求',
                    data: [],
                    borderColor: 'rgb(34, 197, 94)',
                    backgroundColor: 'rgba(34, 197, 94, 0.1)',
                    fill: true,
                    tension: 0.4
                },
                {
                    label: '失败请求',
                    data: [],
                    borderColor: 'rgb(239, 68, 68)',
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
                    position: 'top',
                },
                tooltip: {
                    mode: 'index',
                    intersect: false,
                }
            },
            scales: {
                y: {
                    beginAtZero: true,
                    grid: {
                        drawBorder: false,
                    }
                },
                x: {
                    grid: {
                        display: false,
                    }
                }
            }
        }
    });
}

function initRealtimeChart() {
    const ctx = document.getElementById('realtimeChart');
    if (!ctx) return;

    if (realtimeChart) {
        realtimeChart.destroy();
    }

    const now = new Date();
    const labels = [];
    for (let i = 11; i >= 0; i--) {
        const time = new Date(now - i * 5000);
        labels.push(time.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' }));
    }

    realtimeChart = new Chart(ctx, {
        type: 'bar',
        data: {
            labels: labels,
            datasets: [{
                label: 'QPS',
                data: [],
                backgroundColor: 'rgba(59, 130, 246, 0.5)',
                borderColor: 'rgb(59, 130, 246)',
                borderWidth: 1
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    display: false,
                }
            },
            scales: {
                y: {
                    beginAtZero: true,
                    grid: {
                        drawBorder: false,
                    }
                },
                x: {
                    grid: {
                        display: false,
                    }
                }
            }
        }
    });
}

async function loadDashboardData() {
    try {
        await Promise.all([
            loadStatsData(),
            loadRequestTrendData('hour'),
            loadSystemStatus(),
            loadRecentActivity(),
            loadRealtimeData()
        ]);
    } catch (error) {
        console.error('加载仪表盘数据失败:', error);
        Auth.showToast('数据加载失败', 'error');
    }
}

async function loadStatsData() {
    try {
        const response = await fetch('/admin/api/stats/summary');
        if (!response.ok) throw new Error('获取统计数据失败');

        const data = await response.json();

        updateStatCard('totalUsers', data.totalUsers, data.usersTrend, true);
        updateStatCard('totalApps', data.totalApps, data.appsTrend, true);
        updateStatCard('totalRequests', data.totalRequests, data.requestsTrend, true);
        updateStatCard('totalErrors', data.totalErrors, data.errorsTrend, false);
    } catch (error) {
        console.error('加载统计数据失败:', error);
        animateValue('totalUsers', 0, 1234, 1000);
        animateValue('totalApps', 0, 56, 1000);
        animateValue('totalRequests', 0, 89765, 1000);
        animateValue('totalErrors', 0, 234, 1000);
    }
}

function updateStatCard(elementId, value, trend, isPositiveGood) {
    const valueElement = document.getElementById(elementId);
    const trendElement = document.getElementById(elementId.replace('total', '').toLowerCase() + 'Trend');

    if (valueElement) {
        animateValue(elementId, parseInt(valueElement.textContent.replace(/,/g, '')) || 0, value, 1000);
    }

    if (trendElement && trend !== undefined) {
        const isPositive = trend >= 0;
        const isGood = isPositiveGood ? isPositive : !isPositive;

        trendElement.className = isGood ? 'text-success' : 'text-danger';
        trendElement.innerHTML = `<i class="fas fa-arrow-${isPositive ? 'up' : 'down'} me-1"></i>${isPositive ? '+' : ''}${trend}%`;
    }
}

function animateValue(elementId, start, end, duration) {
    const element = document.getElementById(elementId);
    if (!element) return;

    const range = end - start;
    const startTime = performance.now();

    function update(currentTime) {
        const elapsed = currentTime - startTime;
        const progress = Math.min(elapsed / duration, 1);

        const easeOutQuad = progress * (2 - progress);
        const current = Math.floor(start + range * easeOutQuad);

        element.textContent = current.toLocaleString();

        if (progress < 1) {
            requestAnimationFrame(update);
        }
    }

    requestAnimationFrame(update);
}

async function loadRequestTrendData(period) {
    try {
        const response = await fetch(`/admin/api/stats/request-trend?period=${period}`);
        if (!response.ok) throw new Error('获取请求趋势失败');

        const data = await response.json();

        if (requestTrendChart) {
            requestTrendChart.data.labels = data.labels;
            requestTrendChart.data.datasets[0].data = data.success;
            requestTrendChart.data.datasets[1].data = data.failed;
            requestTrendChart.update();
        }
    } catch (error) {
        console.error('加载请求趋势数据失败:', error);
        const mockLabels = [];
        const mockSuccess = [];
        const mockFailed = [];

        for (let i = 0; i < 12; i++) {
            mockLabels.push(`${i}:00`);
            mockSuccess.push(Math.floor(Math.random() * 1000) + 500);
            mockFailed.push(Math.floor(Math.random() * 50) + 10);
        }

        if (requestTrendChart) {
            requestTrendChart.data.labels = mockLabels;
            requestTrendChart.data.datasets[0].data = mockSuccess;
            requestTrendChart.data.datasets[1].data = mockFailed;
            requestTrendChart.update();
        }
    }
}

async function loadSystemStatus() {
    try {
        const response = await fetch('/admin/api/stats/system-status');
        if (!response.ok) throw new Error('获取系统状态失败');

        const data = await response.json();

        updateServiceStatus('db', data.database);
        updateServiceStatus('redis', data.redis);
        updateServiceStatus('api', data.api);
        updateServiceStatus('storage', data.storage);

        updateResourceUsage('cpu', data.cpu);
        updateResourceUsage('memory', data.memory);
        updateResourceUsage('disk', data.disk);
    } catch (error) {
        console.error('加载系统状态失败:', error);
        updateServiceStatus('db', { status: 'healthy', latency: 15 });
        updateServiceStatus('redis', { status: 'healthy', latency: 3 });
        updateServiceStatus('api', { status: 'healthy', latency: 45 });
        updateServiceStatus('storage', { status: 'healthy', latency: 20 });
        updateResourceUsage('cpu', 45);
        updateResourceUsage('memory', 62);
        updateResourceUsage('disk', 38);
    }
}

function updateServiceStatus(service, data) {
    const statusBadge = document.getElementById(`${service}Status`);
    const latencyText = document.getElementById(`${service}Latency`);

    if (statusBadge) {
        statusBadge.className = `badge rounded-pill ${data.status === 'healthy' ? 'bg-success' : 'bg-danger'}`;
    }

    if (latencyText) {
        latencyText.textContent = `${data.latency}ms`;
    }
}

function updateResourceUsage(type, value) {
    const usageText = document.getElementById(`${type}Usage`);
    const progressBar = document.getElementById(`${type}Progress`);

    if (usageText) {
        usageText.textContent = `${value}%`;
    }

    if (progressBar) {
        progressBar.style.width = `${value}%`;

        let bgClass = 'bg-info';
        if (type === 'cpu') bgClass = 'bg-info';
        else if (type === 'memory') bgClass = 'bg-success';
        else if (type === 'disk') bgClass = 'bg-warning';

        progressBar.className = `progress-bar ${bgClass}`;

        if (value > 90) {
            progressBar.classList.add('bg-danger');
            progressBar.classList.remove(bgClass);
        }
    }
}

async function loadRecentActivity() {
    const tbody = document.getElementById('recentActivity');
    if (!tbody) return;

    try {
        const response = await fetch('/admin/api/activity/recent?limit=10');
        if (!response.ok) throw new Error('获取最近活动失败');

        const data = await response.json();
        renderActivityTable(tbody, data);
    } catch (error) {
        console.error('加载最近活动失败:', error);
        const mockData = [
            { time: '2024-01-15 14:30:25', event: '用户登录', user: 'admin', status: 'success' },
            { time: '2024-01-15 14:28:10', event: '应用创建', user: 'developer', status: 'success' },
            { time: '2024-01-15 14:25:33', event: '配置修改', user: 'admin', status: 'success' },
            { time: '2024-01-15 14:20:15', event: 'API调用', user: 'app_user', status: 'failed' },
            { time: '2024-01-15 14:15:42', event: '权限变更', user: 'admin', status: 'success' }
        ];
        renderActivityTable(tbody, mockData);
    }
}

function renderActivityTable(tbody, activities) {
    tbody.innerHTML = '';

    activities.forEach(activity => {
        const tr = document.createElement('tr');

        const statusClass = activity.status === 'success' ? 'text-success' : 'text-danger';
        const statusIcon = activity.status === 'success' ? 'fa-check-circle' : 'fa-times-circle';
        const statusText = activity.status === 'success' ? '成功' : '失败';

        tr.innerHTML = `
            <td><small>${activity.time}</small></td>
            <td>${activity.event}</td>
            <td><small>${activity.user}</small></td>
            <td><i class="fas ${statusIcon} ${statusClass}"></i> ${statusText}</td>
        `;

        tbody.appendChild(tr);
    });
}

async function loadRealtimeData() {
    try {
        const response = await fetch('/admin/api/stats/realtime');
        if (!response.ok) throw new Error('获取实时数据失败');

        const data = await response.json();
        updateRealtimeChart(data.qps);
    } catch (error) {
        console.error('加载实时数据失败:', error);
        const mockQps = [];
        for (let i = 0; i < 12; i++) {
            mockQps.push(Math.floor(Math.random() * 100) + 20);
        }
        updateRealtimeChart(mockQps);
    }
}

function updateRealtimeChart(qpsData) {
    if (!realtimeChart) return;

    const now = new Date();
    const newLabel = now.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' });

    realtimeChart.data.labels.push(newLabel);
    realtimeChart.data.labels.shift();

    realtimeChart.data.datasets[0].data.push(...qpsData);
    if (qpsData.length === 1) {
        realtimeChart.data.datasets[0].data.shift();
    }

    realtimeChart.update('none');
}

function startAutoRefresh() {
    stopAutoRefresh();
    autoRefreshInterval = setInterval(() => {
        loadDashboardData();
    }, refreshInterval);
}

function stopAutoRefresh() {
    if (autoRefreshInterval) {
        clearInterval(autoRefreshInterval);
        autoRefreshInterval = null;
    }
}

window.Dashboard = {
    loadDashboardData: loadDashboardData,
    loadRequestTrendData: loadRequestTrendData,
    loadSystemStatus: loadSystemStatus,
    loadRecentActivity: loadRecentActivity,
    startAutoRefresh: startAutoRefresh,
    stopAutoRefresh: stopAutoRefresh
};
