let requestTrendChart, realtimeChart;
let autoRefreshInterval = null;
let realtimeDataPoints = [];
let previousStats = null;
const REALTIME_UPDATE_INTERVAL = 5000;
const MAX_REALTIME_POINTS = 20;

document.addEventListener('DOMContentLoaded', async () => {
    initCharts();
    setupEventListeners();
    await loadDashboardStats();
    await loadSystemStatus();
    loadRecentActivity();
    startAutoRefresh();
});

function setupEventListeners() {
    const refreshBtn = document.getElementById('refreshBtn');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', async () => {
            await loadDashboardStats();
            await loadSystemStatus();
            loadRecentActivity();
        });
    }

    const autoRefreshSwitch = document.getElementById('autoRefreshSwitch');
    if (autoRefreshSwitch) {
        autoRefreshSwitch.addEventListener('change', (e) => {
            if (e.target.checked) {
                startAutoRefresh();
            } else {
                stopAutoRefresh();
            }
        });
    }

    const periodButtons = document.querySelectorAll('[data-period]');
    periodButtons.forEach(btn => {
        btn.addEventListener('click', async (e) => {
            periodButtons.forEach(b => b.classList.remove('active'));
            e.target.classList.add('active');
            const period = e.target.dataset.period;
            await loadRequestTrendData(period);
        });
    });
}

function startAutoRefresh() {
    stopAutoRefresh();
    autoRefreshInterval = setInterval(async () => {
        await loadDashboardStats();
        await loadSystemStatus();
    }, REALTIME_UPDATE_INTERVAL);
}

function stopAutoRefresh() {
    if (autoRefreshInterval) {
        clearInterval(autoRefreshInterval);
        autoRefreshInterval = null;
    }
}

async function loadDashboardStats() {
    const mockData = getMockDashboardStats();

    try {
        const data = await auth.request('/admin/dashboard/stats');
        if (data.code === 0) {
            updateStats(data.data);
        } else {
            updateStats(mockData);
        }
    } catch (error) {
        updateStats(mockData);
    }

    previousStats = {
        totalUsers: parseInt(document.getElementById('totalUsers').textContent.replace(/[^\d]/g, '')) || 0,
        totalApps: parseInt(document.getElementById('totalApps').textContent.replace(/[^\d]/g, '')) || 0,
        totalRequests: parseInt(document.getElementById('totalRequests').textContent.replace(/[^\d]/g, '')) || 0,
        totalErrors: parseInt(document.getElementById('totalErrors').textContent.replace(/[^\d]/g, '')) || 0
    };

    await loadRequestTrendData('hour');
    updateRealtimeChart(mockData.requestsPerMinute || 0);
}

function getMockDashboardStats() {
    const baseUsers = 12456;
    const baseApps = 156;
    const baseRequests = 8234567;
    const baseErrors = 1234;

    return {
        totalUsers: baseUsers + Math.floor(Math.random() * 50),
        totalApps: baseApps + Math.floor(Math.random() * 3),
        totalRequests: baseRequests + Math.floor(Math.random() * 1000),
        totalErrors: Math.max(0, baseErrors + Math.floor(Math.random() * 50) - 25),
        requestsPerMinute: Math.floor(Math.random() * 100) + 50,
        userGrowth: 12.5,
        appGrowth: 8.2,
        requestGrowth: 23.1,
        errorGrowth: -5.7,
        systemStatus: {
            database: { status: 'healthy', latency: Math.floor(Math.random() * 50) + 10 },
            redis: { status: 'healthy', latency: Math.floor(Math.random() * 10) + 1 },
            api: { status: 'healthy', latency: Math.floor(Math.random() * 100) + 20 },
            storage: { status: 'healthy', latency: Math.floor(Math.random() * 30) + 5 }
        },
        resourceUsage: {
            cpu: Math.floor(Math.random() * 40) + 20,
            memory: Math.floor(Math.random() * 30) + 40,
            disk: Math.floor(Math.random() * 20) + 50
        }
    };
}

function updateStats(stats) {
    animateNumber('totalUsers', stats.totalUsers);
    animateNumber('totalApps', stats.totalApps);
    animateNumber('totalRequests', stats.totalRequests);
    animateNumber('totalErrors', stats.totalErrors);

    updateTrend('usersTrend', stats.userGrowth || 0);
    updateTrend('appsTrend', stats.appGrowth || 0);
    updateTrend('requestsTrend', stats.requestGrowth || 0);
    updateTrend('errorsTrend', stats.errorGrowth || 0, true);

    if (stats.systemStatus) {
        updateSystemStatus(stats.systemStatus);
    }

    if (stats.resourceUsage) {
        updateResourceUsage(stats.resourceUsage);
    }
}

function updateTrend(elementId, value, isInverse = false) {
    const el = document.getElementById(elementId);
    if (!el) return;

    const isPositive = value >= 0;
    const displayValue = Math.abs(value).toFixed(1);
    const iconClass = (isPositive === !isInverse) ? 'fa-arrow-up' : 'fa-arrow-down';
    const colorClass = (isPositive === !isInverse) ? 'text-success' : 'text-danger';

    el.className = colorClass;
    el.innerHTML = `<i class="fas ${iconClass} me-1"></i>${isPositive ? '+' : '-'}${displayValue}%`;
}

function updateSystemStatus(status) {
    const services = ['db', 'redis', 'api', 'storage'];
    services.forEach(service => {
        const statusEl = document.getElementById(`${service}Status`);
        const latencyEl = document.getElementById(`${service}Latency`);

        if (statusEl && status[service]) {
            const isHealthy = status[service].status === 'healthy';
            statusEl.className = `badge rounded-pill ${isHealthy ? 'bg-success' : 'bg-danger'}`;
        }

        if (latencyEl && status[service]) {
            latencyEl.textContent = `${status[service].latency}ms`;
        }
    });
}

function updateResourceUsage(usage) {
    if (usage.cpu !== undefined) {
        document.getElementById('cpuUsage').textContent = `${usage.cpu}%`;
        document.getElementById('cpuProgress').style.width = `${usage.cpu}%`;
        const cpuBar = document.getElementById('cpuProgress');
        cpuBar.className = `progress-bar ${usage.cpu > 80 ? 'bg-danger' : usage.cpu > 60 ? 'bg-warning' : 'bg-info'}`;
    }

    if (usage.memory !== undefined) {
        document.getElementById('memUsage').textContent = `${usage.memory}%`;
        document.getElementById('memProgress').style.width = `${usage.memory}%`;
        const memBar = document.getElementById('memProgress');
        memBar.className = `progress-bar ${usage.memory > 80 ? 'bg-danger' : usage.memory > 60 ? 'bg-warning' : 'bg-success'}`;
    }

    if (usage.disk !== undefined) {
        document.getElementById('diskUsage').textContent = `${usage.disk}%`;
        document.getElementById('diskProgress').style.width = `${usage.disk}%`;
        const diskBar = document.getElementById('diskProgress');
        diskBar.className = `progress-bar ${usage.disk > 90 ? 'bg-danger' : usage.disk > 70 ? 'bg-warning' : 'bg-warning'}`;
    }
}

function animateNumber(elementId, target) {
    const element = document.getElementById(elementId);
    if (!element) return;

    const currentText = element.textContent;
    const current = parseInt(currentText.replace(/[^\d]/g, '')) || 0;
    const duration = 1000;
    const startTime = performance.now();

    function update(currentTime) {
        const elapsed = currentTime - startTime;
        const progress = Math.min(elapsed / duration, 1);
        const easeProgress = easeOutQuart(progress);
        const value = Math.floor(current + (target - current) * easeProgress);
        element.textContent = formatNumber(value);

        if (progress < 1) {
            requestAnimationFrame(update);
        }
    }

    requestAnimationFrame(update);
}

function easeOutQuart(x) {
    return 1 - Math.pow(1 - x, 4);
}

function formatNumber(num) {
    if (num >= 1000000) {
        return (num / 1000000).toFixed(1) + 'M';
    } else if (num >= 1000) {
        return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
}

async function loadSystemStatus() {
    try {
        const data = await auth.request('/admin/dashboard/system-status');
        if (data.code === 0) {
            updateSystemStatus(data.data.status || {});
            updateResourceUsage(data.data.resourceUsage || {});
        }
    } catch (error) {
        const mockStatus = getMockDashboardStats();
        updateSystemStatus(mockStatus.systemStatus);
        updateResourceUsage(mockStatus.resourceUsage);
    }
}

function initCharts() {
    initRequestTrendChart();
    initRealtimeChart();
}

function initRequestTrendChart() {
    const ctx = document.getElementById('requestTrendChart');
    if (!ctx) return;

    requestTrendChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: [],
            datasets: [{
                label: '请求量',
                data: [],
                borderColor: '#3b82f6',
                backgroundColor: 'rgba(59, 130, 246, 0.1)',
                fill: true,
                tension: 0.4,
                pointRadius: 3,
                pointHoverRadius: 6
            }]
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
                    display: false
                },
                tooltip: {
                    backgroundColor: 'rgba(0, 0, 0, 0.8)',
                    padding: 12,
                    titleFont: { size: 14 },
                    bodyFont: { size: 13 },
                    displayColors: false
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
                    grid: {
                        color: 'rgba(0, 0, 0, 0.05)'
                    }
                }
            }
        }
    });
}

function initRealtimeChart() {
    const ctx = document.getElementById('realtimeChart');
    if (!ctx) return;

    realtimeDataPoints = Array(MAX_REALTIME_POINTS).fill(0).map((_, i) => ({
        x: new Date(Date.now() - (MAX_REALTIME_POINTS - i) * 5000),
        y: Math.floor(Math.random() * 50) + 30
    }));

    realtimeChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: realtimeDataPoints.map(p => formatTime(p.x)),
            datasets: [{
                label: '请求/秒',
                data: realtimeDataPoints.map(p => p.y),
                borderColor: '#10b981',
                backgroundColor: 'rgba(16, 185, 129, 0.1)',
                fill: true,
                tension: 0.4,
                pointRadius: 2
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
                    displayColors: false
                }
            },
            scales: {
                x: {
                    display: false
                },
                y: {
                    beginAtZero: true,
                    grid: {
                        color: 'rgba(0, 0, 0, 0.05)'
                    }
                }
            },
            animation: {
                duration: 300
            }
        }
    });
}

function formatTime(date) {
    return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

async function loadRequestTrendData(period) {
    const mockData = getMockTrendData(period);

    try {
        const data = await auth.request(`/admin/dashboard/request-trend?period=${period}`);
        if (data.code === 0) {
            updateRequestTrendChart(data.data);
        } else {
            updateRequestTrendChart(mockData);
        }
    } catch (error) {
        updateRequestTrendChart(mockData);
    }
}

function getMockTrendData(period) {
    let labels, data;

    if (period === 'hour') {
        labels = Array.from({ length: 24 }, (_, i) => `${i}:00`);
        data = Array.from({ length: 24 }, () => Math.floor(Math.random() * 5000) + 1000);
    } else if (period === 'day') {
        labels = ['周一', '周二', '周三', '周四', '周五', '周六', '周日'];
        data = [12000, 15000, 18000, 16000, 20000, 25000, 22000];
    } else {
        labels = Array.from({ length: 7 }, (_, i) => `第${i + 1}周`);
        data = [85000, 92000, 105000, 98000, 120000, 135000, 145000];
    }

    return { labels, data };
}

function updateRequestTrendChart(data) {
    if (!requestTrendChart || !data) return;

    requestTrendChart.data.labels = data.labels;
    requestTrendChart.data.datasets[0].data = data.data;
    requestTrendChart.update('none');
}

function updateRealtimeChart(requestsPerMinute) {
    if (!realtimeChart) return;

    const now = new Date();
    realtimeDataPoints.push({ x: now, y: requestsPerMinute });

    if (realtimeDataPoints.length > MAX_REALTIME_POINTS) {
        realtimeDataPoints.shift();
    }

    realtimeChart.data.labels = realtimeDataPoints.map(p => formatTime(p.x));
    realtimeChart.data.datasets[0].data = realtimeDataPoints.map(p => p.y);
    realtimeChart.update('none');
}

async function loadRecentActivity() {
    const mockActivities = getMockActivities();

    try {
        const data = await auth.request('/admin/dashboard/activity');
        if (data.code === 0 && data.data) {
            renderActivityTable(data.data);
        } else {
            renderActivityTable(mockActivities);
        }
    } catch (error) {
        renderActivityTable(mockActivities);
    }
}

function getMockActivities() {
    const activities = [
        { time: getRelativeTime(0), event: '用户登录', user: 'admin', status: 'success' },
        { time: getRelativeTime(3), event: '创建应用', user: 'developer1', status: 'success' },
        { time: getRelativeTime(5), event: 'API请求失败', user: 'app_001', status: 'error' },
        { time: getRelativeTime(8), event: '更新配置', user: 'admin', status: 'success' },
        { time: getRelativeTime(12), event: '用户注册', user: 'new_user', status: 'success' },
        { time: getRelativeTime(15), event: '验证码校验', user: 'user_123', status: 'success' },
        { time: getRelativeTime(18), event: '批量导出', user: 'admin', status: 'success' },
        { time: getRelativeTime(22), event: '权限变更', user: 'super_admin', status: 'success' }
    ];
    return activities;
}

function getRelativeTime(minutesAgo) {
    const date = new Date(Date.now() - minutesAgo * 60 * 1000);
    return date.toLocaleString('zh-CN', {
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
    }).replace(/\//g, '-');
}

function renderActivityTable(activities) {
    const tbody = document.getElementById('recentActivity');
    if (!tbody) return;

    tbody.innerHTML = activities.slice(0, 8).map(activity => `
        <tr>
            <td><small class="text-muted">${activity.time}</small></td>
            <td>${escapeHtml(activity.event)}</td>
            <td><code>${escapeHtml(activity.user)}</code></td>
            <td><span class="badge ${getStatusBadgeClass(activity.status)}">${getStatusText(activity.status)}</span></td>
        </tr>
    `).join('');
}

function getStatusBadgeClass(status) {
    const map = {
        success: 'bg-success',
        error: 'bg-danger',
        pending: 'bg-warning',
        warning: 'bg-warning'
    };
    return map[status] || 'bg-secondary';
}

function getStatusText(status) {
    const map = {
        success: '成功',
        error: '失败',
        pending: '处理中',
        warning: '警告'
    };
    return map[status] || status;
}

function escapeHtml(text) {
    if (text === null || text === undefined) return '';
    const div = document.createElement('div');
    div.textContent = String(text);
    return div.innerHTML;
}

function initHeatmapChart() {
    const chartDom = document.getElementById('heatmapChart');
    if (!chartDom) return;

    heatmapChart = echarts.init(chartDom);

    const hours = [];
    for (let i = 0; i < 24; i++) {
        hours.push(String(i).padStart(2, '0') + ':00');
    }

    const days = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];

    const data = [];
    for (let i = 0; i < 7; i++) {
        for (let j = 0; j < 24; j++) {
            data.push([j, i, Math.floor(Math.random() * 100)]);
        }
    }

    const option = {
        tooltip: {
            position: 'top',
            formatter: function(params) {
                return hours[params.value[0]] + '<br>' + days[params.value[1]] + ': ' + params.value[2] + ' 次';
            }
        },
        grid: {
            left: '3%',
            right: '10%',
            top: '10%',
            bottom: '15%',
            containLabel: true
        },
        xAxis: {
            type: 'category',
            data: hours,
            splitArea: { show: true },
            axisLabel: {
                interval: 2,
                fontSize: 10
            }
        },
        yAxis: {
            type: 'category',
            data: days,
            splitArea: { show: true }
        },
        visualMap: {
            min: 0,
            max: 100,
            calculable: true,
            orient: 'vertical',
            right: '0',
            top: 'center',
            inRange: {
                color: ['#e8f5e9', '#c8e6c9', '#a5d6a7', '#81c784', '#66bb6a', '#4caf50', '#43a047', '#388e3c', '#2e7d32', '#1b5e20']
            }
        },
        series: [{
            name: '验证请求',
            type: 'heatmap',
            data: data,
            label: {
                show: false
            },
            emphasis: {
                itemStyle: {
                    shadowBlur: 10,
                    shadowColor: 'rgba(0, 0, 0, 0.5)'
                }
            }
        }]
    };

    heatmapChart.setOption(option);
}

function initRadarChart() {
    const chartDom = document.getElementById('radarChart');
    if (!chartDom) return;

    radarChart = echarts.init(chartDom);

    const option = {
        tooltip: {
            trigger: 'item'
        },
        legend: {
            data: ['系统性能'],
            bottom: 10
        },
        radar: {
            indicator: [
                { name: '安全性', max: 100 },
                { name: '响应速度', max: 100 },
                { name: '用户体验', max: 100 },
                { name: '准确性', max: 100 },
                { name: '稳定性', max: 100 }
            ],
            radius: '65%',
            axisName: {
                color: '#666',
                fontSize: 12
            },
            splitArea: {
                areaStyle: {
                    color: ['rgba(59, 130, 246, 0.05)', 'rgba(59, 130, 246, 0.1)', 'rgba(59, 130, 246, 0.15)', 'rgba(59, 130, 246, 0.2)', 'rgba(59, 130, 246, 0.25)']
                }
            },
            axisLine: {
                lineStyle: {
                    color: 'rgba(59, 130, 246, 0.3)'
                }
            },
            splitLine: {
                lineStyle: {
                    color: 'rgba(59, 130, 246, 0.3)'
                }
            }
        },
        series: [{
            type: 'radar',
            data: [{
                value: [80, 75, 85, 90, 95],
                name: '系统性能',
                areaStyle: {
                    color: 'rgba(59, 130, 246, 0.3)'
                },
                lineStyle: {
                    color: '#3b82f6',
                    width: 2
                },
                itemStyle: {
                    color: '#3b82f6'
                }
            }]
        }]
    };

    radarChart.setOption(option);
}

function initSankeyChart() {
    const chartDom = document.getElementById('sankeyChart');
    if (!chartDom) return;

    sankeyChart = echarts.init(chartDom);

    const option = {
        tooltip: {
            trigger: 'item',
            triggerOn: 'mousemove'
        },
        series: [{
            type: 'sankey',
            layout: 'none',
            emphasis: {
                focus: 'adjacency'
            },
            nodeAlign: 'left',
            lineStyle: {
                color: 'gradient',
                curveness: 0.5
            },
            data: [
                { name: '总请求', itemStyle: { color: '#3b82f6' } },
                { name: '滑动验证', itemStyle: { color: '#10b981' } },
                { name: '点选验证', itemStyle: { color: '#f59e0b' } },
                { name: '图片验证', itemStyle: { color: '#ef4444' } },
                { name: '文字验证', itemStyle: { color: '#8b5cf6' } },
                { name: '成功', itemStyle: { color: '#22c55e' } },
                { name: '失败', itemStyle: { color: '#f97316' } },
                { name: '拦截', itemStyle: { color: '#ef4444' } }
            ],
            links: [
                { source: '总请求', target: '滑动验证', value: 1000 },
                { source: '总请求', target: '点选验证', value: 500 },
                { source: '总请求', target: '图片验证', value: 300 },
                { source: '总请求', target: '文字验证', value: 200 },
                { source: '滑动验证', target: '成功', value: 800 },
                { source: '滑动验证', target: '失败', value: 150 },
                { source: '滑动验证', target: '拦截', value: 50 },
                { source: '点选验证', target: '成功', value: 400 },
                { source: '点选验证', target: '失败', value: 80 },
                { source: '点选验证', target: '拦截', value: 20 },
                { source: '图片验证', target: '成功', value: 250 },
                { source: '图片验证', target: '失败', value: 40 },
                { source: '图片验证', target: '拦截', value: 10 },
                { source: '文字验证', target: '成功', value: 180 },
                { source: '文字验证', target: '失败', value: 15 },
                { source: '文字验证', target: '拦截', value: 5 }
            ],
            itemStyle: {
                borderWidth: 1,
                borderColor: '#aaa'
            },
            label: {
                fontSize: 11
            }
        }]
    };

    sankeyChart.setOption(option);
}

async function loadHeatmapData() {
    try {
        const response = await fetch('/admin/api/heatmap');
        if (!response.ok) throw new Error('Failed to load heatmap data');

        const result = await response.json();
        if (result.code === 0 && result.data) {
            updateHeatmapChart(result.data);
        } else {
            loadMockHeatmapData();
        }
    } catch (error) {
        console.error('加载热力图数据失败:', error);
        loadMockHeatmapData();
    }
}

function loadMockHeatmapData() {
    const hours = [];
    for (let i = 0; i < 24; i++) {
        hours.push(String(i).padStart(2, '0') + ':00');
    }
    const days = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];

    const data = [];
    for (let i = 0; i < 7; i++) {
        for (let j = 0; j < 24; j++) {
            const baseValue = (j >= 9 && j <= 18) ? 80 : 30;
            const randomValue = Math.floor(Math.random() * 50);
            data.push([j, i, baseValue + randomValue]);
        }
    }

    updateHeatmapChart({ hours, days, values: data, max_value: 130 });
}

function updateHeatmapChart(data) {
    if (!heatmapChart || !data) return;

    const chartData = [];
    for (let i = 0; i < data.days.length; i++) {
        for (let j = 0; j < data.hours.length; j++) {
            chartData.push([j, i, data.values[i] ? data.values[i][j] : 0]);
        }
    }

    heatmapChart.setOption({
        xAxis: {
            data: data.hours
        },
        yAxis: {
            data: data.days
        },
        visualMap: {
            max: data.max_value || 100
        },
        series: [{
            data: chartData
        }]
    });
}

async function loadRadarData() {
    try {
        const response = await fetch('/admin/api/radar');
        if (!response.ok) throw new Error('Failed to load radar data');

        const result = await response.json();
        if (result.code === 0 && result.data) {
            updateRadarChart(result.data);
        } else {
            loadMockRadarData();
        }
    } catch (error) {
        console.error('加载雷达图数据失败:', error);
        loadMockRadarData();
    }
}

function loadMockRadarData() {
    updateRadarChart({
        indicators: [
            { name: '安全性', max: 100 },
            { name: '响应速度', max: 100 },
            { name: '用户体验', max: 100 },
            { name: '准确性', max: 100 },
            { name: '稳定性', max: 100 }
        ],
        values: [80, 75, 85, 90, 95]
    });
}

function updateRadarChart(data) {
    if (!radarChart || !data) return;

    radarChart.setOption({
        radar: {
            indicator: data.indicators.map(ind => ({
                name: ind.name,
                max: ind.max || 100
            }))
        },
        series: [{
            data: [{
                value: data.values,
                name: '系统性能',
                areaStyle: {
                    color: 'rgba(59, 130, 246, 0.3)'
                }
            }]
        }]
    });
}

async function loadSankeyData() {
    try {
        const response = await fetch('/admin/api/sankey');
        if (!response.ok) throw new Error('Failed to load sankey data');

        const result = await response.json();
        if (result.code === 0 && result.data) {
            updateSankeyChart(result.data);
        } else {
            loadMockSankeyData();
        }
    } catch (error) {
        console.error('加载桑基图数据失败:', error);
        loadMockSankeyData();
    }
}

function loadMockSankeyData() {
    updateSankeyChart({
        nodes: [
            { name: '总请求' },
            { name: '滑动验证' },
            { name: '点选验证' },
            { name: '图片验证' },
            { name: '文字验证' },
            { name: '成功' },
            { name: '失败' },
            { name: '拦截' }
        ],
        links: [
            { source: 0, target: 1, value: 1000 },
            { source: 0, target: 2, value: 500 },
            { source: 0, target: 3, value: 300 },
            { source: 0, target: 4, value: 200 },
            { source: 1, target: 5, value: 800 },
            { source: 1, target: 6, value: 150 },
            { source: 1, target: 7, value: 50 },
            { source: 2, target: 5, value: 400 },
            { source: 2, target: 6, value: 80 },
            { source: 2, target: 7, value: 20 },
            { source: 3, target: 5, value: 250 },
            { source: 3, target: 6, value: 40 },
            { source: 3, target: 7, value: 10 },
            { source: 4, target: 5, value: 180 },
            { source: 4, target: 6, value: 15 },
            { source: 4, target: 7, value: 5 }
        ]
    });
}

function updateSankeyChart(data) {
    if (!sankeyChart || !data) return;

    sankeyChart.setOption({
        series: [{
            data: data.nodes,
            links: data.links
        }]
    });
}

async function loadAdvancedAnalytics(period) {
    try {
        const response = await fetch(`/admin/api/advanced-analytics?period=${period}`);
        if (!response.ok) throw new Error('Failed to load advanced analytics');

        const result = await response.json();
        if (result.code === 0 && result.data) {
            updateAdvancedCharts(result.data);
        } else {
            loadMockAdvancedAnalytics(period);
        }
    } catch (error) {
        console.error('加载高级分析数据失败:', error);
        loadMockAdvancedAnalytics(period);
    }
}

function loadMockAdvancedAnalytics(period) {
    let labels, data;

    if (period === 'hour') {
        labels = Array.from({ length: 24 }, (_, i) => `${i}:00`);
        data = Array.from({ length: 24 }, () => Math.floor(Math.random() * 5000) + 1000);
    } else if (period === 'day') {
        labels = ['周一', '周二', '周三', '周四', '周五', '周六', '周日'];
        data = [12000, 15000, 18000, 16000, 20000, 25000, 22000];
    } else {
        labels = Array.from({ length: 7 }, (_, i) => `第${i + 1}周`);
        data = [85000, 92000, 105000, 98000, 120000, 135000, 145000];
    }

    updateAdvancedCharts({ hourly_trend: { labels, data } });
}

function updateAdvancedCharts(data) {
    if (requestTrendChart && data.hourly_trend) {
        requestTrendChart.setOption({
            xAxis: {
                data: data.hourly_trend.labels || data.hourly_trend.map(t => t.time)
            },
            series: [{
                data: data.hourly_trend.data || data.hourly_trend.map(t => t.requests)
            }]
        });
    }
}

async function loadCustomRangeData(startDate, endDate) {
    try {
        const response = await fetch(`/admin/api/custom-range?start_date=${startDate}&end_date=${endDate}`);
        if (!response.ok) throw new Error('Failed to load custom range data');

        const result = await response.json();
        if (result.code === 0 && result.data) {
            updateCustomRangeDashboard(result.data);
        } else {
            showNotification('加载自定义时间范围数据失败', 'error');
        }
    } catch (error) {
        console.error('加载自定义时间范围数据失败:', error);
        showNotification('加载自定义时间范围数据失败', 'error');
    }
}

function updateCustomRangeDashboard(data) {
    if (data.total_count !== undefined) {
        animateValue('totalRequests', 0, data.total_count, 1000);
    }

    if (data.success_rate !== undefined) {
        document.getElementById('passRate').textContent = data.success_rate.toFixed(1) + '%';
    }

    if (data.daily_trend && requestTrendChart) {
        requestTrendChart.setOption({
            xAxis: {
                data: data.daily_trend.map(t => t.time)
            },
            series: [{
                data: data.daily_trend.map(t => t.requests)
            }]
        });
    }

    showNotification('时间范围数据已更新', 'success');
}

function initRequestTrendChart() {
    const ctx = document.getElementById('requestTrendChart');
    if (!ctx) return;

    requestTrendChart = echarts.init(ctx);

    const option = {
        tooltip: {
            trigger: 'axis',
            axisPointer: {
                type: 'cross'
            }
        },
        grid: {
            left: '3%',
            right: '4%',
            bottom: '3%',
            containLabel: true
        },
        xAxis: {
            type: 'category',
            data: [],
            boundaryGap: false
        },
        yAxis: {
            type: 'value',
            axisLabel: {
                formatter: function(value) {
                    if (value >= 1000) {
                        return (value / 1000).toFixed(1) + 'K';
                    }
                    return value;
                }
            }
        },
        series: [{
            name: '请求量',
            type: 'line',
            smooth: true,
            data: [],
            areaStyle: {
                color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                    { offset: 0, color: 'rgba(59, 130, 246, 0.5)' },
                    { offset: 1, color: 'rgba(59, 130, 246, 0.1)' }
                ])
            },
            lineStyle: {
                color: '#3b82f6',
                width: 2
            },
            itemStyle: {
                color: '#3b82f6'
            }
        }]
    };

    requestTrendChart.setOption(option);
}

function initRealtimeChart() {
    const ctx = document.getElementById('realtimeChart');
    if (!ctx) return;

    realtimeChart = echarts.init(ctx);

    const option = {
        tooltip: {
            trigger: 'axis',
            axisPointer: {
                type: 'line'
            }
        },
        grid: {
            left: '3%',
            right: '4%',
            bottom: '3%',
            containLabel: true
        },
        xAxis: {
            type: 'category',
            data: [],
            show: false
        },
        yAxis: {
            type: 'value',
            axisLabel: {
                formatter: function(value) {
                    if (value >= 1000) {
                        return (value / 1000).toFixed(1) + 'K';
                    }
                    return value;
                }
            }
        },
        series: [{
            name: '请求/秒',
            type: 'line',
            smooth: true,
            data: [],
            areaStyle: {
                color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                    { offset: 0, color: 'rgba(16, 185, 129, 0.5)' },
                    { offset: 1, color: 'rgba(16, 185, 129, 0.1)' }
                ])
            },
            lineStyle: {
                color: '#10b981',
                width: 2
            },
            itemStyle: {
                color: '#10b981'
            }
        }]
    };

    realtimeChart.setOption(option);
    startRealtimeUpdates();
}

function startRealtimeUpdates() {
    setInterval(function() {
        const now = new Date();
        const timeString = now.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
        const randomValue = Math.floor(Math.random() * 50) + 30;

        realtimeChart.addData(0, randomValue);
    }, REALTIME_UPDATE_INTERVAL);
}

window.addEventListener('resize', function() {
    if (heatmapChart) heatmapChart.resize();
    if (radarChart) radarChart.resize();
    if (sankeyChart) sankeyChart.resize();
    if (requestTrendChart) requestTrendChart.resize();
    if (realtimeChart) realtimeChart.resize();
});
