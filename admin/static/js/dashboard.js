let requestTrendChart, realtimeChart, distributionChart, riskGaugeChart;
let ws = null;
let wsConnected = false;
let autoRefreshInterval = null;
let realtimeDataPoints = [];
let previousStats = null;
let chartUpdateThrottle = null;
let dataBuffer = [];
const REALTIME_UPDATE_INTERVAL = 5000;
const MAX_REALTIME_POINTS = 60;
const WS_RECONNECT_DELAY = 3000;
const AUTO_REFRESH_INTERVAL = 30000;
const CHART_UPDATE_THROTTLE_MS = 1000;
let isAutoRefreshEnabled = false;
const chartCache = new Map();
const memoryMonitor = {
    startTime: performance.now(),
    updateCount: 0,
    lastMemoryCheck: 0
};

document.addEventListener('DOMContentLoaded', async () => {
    startMemoryMonitor();
    initECharts();
    initWebSocket();
    setupEventListeners();
    await loadDashboardStats();
    await loadSystemStatus();
    loadRecentActivity();
    startAutoRefresh();
    await loadExtendedStats();
    initAdvancedCharts();
});

function startMemoryMonitor() {
    setInterval(() => {
        if (performance.memory) {
            const used = Math.round(performance.memory.usedJSHeapSize / 1048576);
            const limit = Math.round(performance.memory.jsHeapSizeLimit / 1048576);
            console.log(`Memory: ${used}MB / ${limit}MB`);
        }
        memoryMonitor.updateCount++;
    }, 30000);
}

function toggleAutoRefresh() {
    isAutoRefreshEnabled = !isAutoRefreshEnabled;
    const statusEl = document.getElementById('autoRefreshStatus');
    const btnEl = document.getElementById('autoRefreshBtn');
    
    if (isAutoRefreshEnabled) {
        statusEl.textContent = '开启';
        btnEl.classList.add('btn-success');
        btnEl.classList.remove('btn-default');
        autoRefreshInterval = setInterval(() => {
            loadDashboardStats();
            loadSystemStatus();
        }, AUTO_REFRESH_INTERVAL);
        showToast('自动刷新已开启（每30秒）', 'info');
    } else {
        statusEl.textContent = '关闭';
        btnEl.classList.remove('btn-success');
        btnEl.classList.add('btn-default');
        if (autoRefreshInterval) {
            clearInterval(autoRefreshInterval);
            autoRefreshInterval = null;
        }
        showToast('自动刷新已关闭', 'info');
    }
}

async function loadExtendedStats() {
    try {
        const response = await fetch('/admin/api/dashboard/extended');
        if (!response.ok) throw new Error('Network error');
        
        const result = await response.json();
        if (result.code === 0) {
            updateExtendedStats(result.data);
        } else {
            loadMockExtendedStats();
        }
    } catch (error) {
        console.error('Extended stats load failed:', error);
        loadMockExtendedStats();
    }
}

function loadMockExtendedStats() {
    updateExtendedStats({
        total_users: Math.floor(Math.random() * 5000) + 8000,
        total_apps: Math.floor(Math.random() * 50) + 100,
        current_qps: Math.floor(Math.random() * 50) + 20,
        error_rate: (Math.random() * 2 + 0.5).toFixed(2),
        user_growth: (Math.random() * 15 + 5).toFixed(1),
        app_growth: (Math.random() * 10 + 2).toFixed(1),
        error_growth: (Math.random() * 5 - 3).toFixed(1)
    });
}

function updateExtendedStats(data) {
    const totalUsersEl = document.getElementById('totalUsers');
    const totalAppsEl = document.getElementById('totalApps');
    const currentQPSEl = document.getElementById('currentQPSDisplay');
    const errorRateEl = document.getElementById('errorRate');
    const userGrowthEl = document.getElementById('userGrowth');
    const appGrowthEl = document.getElementById('appGrowth');
    const errorTrendEl = document.getElementById('errorTrend');
    
    if (totalUsersEl) {
        animateNumber('totalUsers', data.total_users || 0);
        if (userGrowthEl) {
            const growth = parseFloat(data.user_growth || 0);
            userGrowthEl.textContent = growth >= 0 ? `↑ ${growth}%` : `↓ ${Math.abs(growth)}%`;
            userGrowthEl.className = growth >= 0 ? 'text-success' : 'text-danger';
        }
    }
    
    if (totalAppsEl) {
        animateNumber('totalApps', data.total_apps || 0);
        if (appGrowthEl) {
            const growth = parseFloat(data.app_growth || 0);
            appGrowthEl.textContent = growth >= 0 ? `↑ ${growth}%` : `↓ ${Math.abs(growth)}%`;
            appGrowthEl.className = growth >= 0 ? 'text-success' : 'text-danger';
        }
    }
    
    if (currentQPSEl) {
        animateNumber('currentQPSDisplay', data.current_qps || 0);
    }
    
    if (errorRateEl) {
        errorRateEl.textContent = (data.error_rate || 0) + '%';
        if (errorTrendEl) {
            const growth = parseFloat(data.error_growth || 0);
            errorTrendEl.textContent = growth <= 0 ? `↓ ${Math.abs(growth)}%` : `↑ ${growth}%`;
            errorTrendEl.className = growth <= 0 ? 'text-success' : 'text-danger';
        }
    }
}

function initECharts() {
    initRequestTrendChart();
    initRealtimeChart();
    initDistributionChart();
    initRiskGaugeChart();
}

function initAdvancedCharts() {
    const distributionContainer = document.getElementById('distributionChart');
    if (distributionContainer && !distributionChart) {
        distributionChart = echarts.init(distributionContainer);
        updateDistributionChart(getMockDistributionData());
    }

    const riskContainer = document.getElementById('riskGaugeChart');
    if (riskContainer && !riskGaugeChart) {
        riskGaugeChart = echarts.init(riskContainer);
        updateRiskGaugeChart(45);
    }

    window.addEventListener('resize', () => {
        if (distributionChart) distributionChart.resize();
        if (riskGaugeChart) riskGaugeChart.resize();
    });
}

function initDistributionChart() {
    const container = document.getElementById('distributionChart');
    if (!container) return;

    distributionChart = echarts.init(container);
    updateDistributionChart(getMockDistributionData());
}

function initRiskGaugeChart() {
    const container = document.getElementById('riskGaugeChart');
    if (!container) return;

    riskGaugeChart = echarts.init(container);
    updateRiskGaugeChart(50);
}

function updateDistributionChart(data) {
    if (!distributionChart) return;

    const option = {
        tooltip: {
            trigger: 'item',
            formatter: '{b}: {c} ({d}%)'
        },
        legend: {
            orient: 'vertical',
            left: 'left',
            textStyle: { color: '#666' }
        },
        series: [{
            name: 'Distribution',
            type: 'pie',
            radius: ['40%', '70%'],
            avoidLabelOverlap: false,
            itemStyle: {
                borderRadius: 10,
                borderColor: '#fff',
                borderWidth: 2
            },
            label: {
                show: true,
                formatter: '{b}\n{c}'
            },
            emphasis: {
                label: {
                    show: true,
                    fontSize: 14,
                    fontWeight: 'bold'
                }
            },
            data: data
        }],
        animation: true,
        animationDuration: 1000,
        animationEasing: 'cubicOut'
    };

    distributionChart.setOption(option);
}

function updateRiskGaugeChart(value) {
    if (!riskGaugeChart) return;

    const option = {
        series: [{
            type: 'gauge',
            startAngle: 180,
            endAngle: 0,
            min: 0,
            max: 100,
            splitNumber: 5,
            axisLine: {
                lineStyle: {
                    width: 20,
                    color: [
                        [0.3, '#10b981'],
                        [0.7, '#f59e0b'],
                        [1, '#ef4444']
                    ]
                }
            },
            pointer: {
                icon: 'path://M12.8,0.7l12,40.1H0.7L12.8,0.7z',
                length: '60%',
                width: 8,
                offsetCenter: [0, '-10%'],
                itemStyle: {
                    color: '#666'
                }
            },
            axisTick: {
                length: 8,
                lineStyle: { color: 'auto', width: 1 }
            },
            splitLine: {
                length: 15,
                lineStyle: { color: 'auto', width: 2 }
            },
            axisLabel: {
                color: '#666',
                fontSize: 10,
                distance: -40
            },
            title: {
                show: true,
                offsetCenter: [0, '30%'],
                fontSize: 12,
                color: '#666'
            },
            detail: {
                valueAnimation: true,
                formatter: function(value) {
                    return Math.round(value);
                },
                color: '#333',
                fontSize: 24,
                offsetCenter: [0, '-20%']
            },
            data: [{
                value: value,
                name: 'Risk Score'
            }]
        }]
    };

    riskGaugeChart.setOption(option, false);
}

function getMockDistributionData() {
    return [
        { value: 45, name: 'Slider', itemStyle: { color: '#3b82f6' } },
        { value: 25, name: 'Click', itemStyle: { color: '#10b981' } },
        { value: 15, name: '3D', itemStyle: { color: '#f59e0b' } },
        { value: 10, name: 'Voice', itemStyle: { color: '#8b5cf6' } },
        { value: 5, name: 'Other', itemStyle: { color: '#6b7280' } }
    ];
}

function initRequestTrendChart() {
    const container = document.getElementById('requestTrendChart');
    if (!container) return;

    requestTrendChart = echarts.init(container);
    window.addEventListener('resize', () => requestTrendChart.resize());
}

function initRealtimeChart() {
    const container = document.getElementById('realtimeChart');
    if (!container) return;

    realtimeDataPoints = Array(MAX_REALTIME_POINTS).fill(0).map((_, i) => ({
        time: formatTime(new Date(Date.now() - (MAX_REALTIME_POINTS - i) * 5000)),
        value: Math.floor(Math.random() * 50) + 30
    }));

    realtimeChart = echarts.init(container);
    window.addEventListener('resize', () => realtimeChart.resize());

    updateRealtimeChartInit();
}

function updateRealtimeChartInit() {
    if (!realtimeChart) return;

    realtimeChart.setOption({
        xAxis: {
            type: 'category',
            data: realtimeDataPoints.map(p => p.time),
            axisLabel: { color: '#666', rotate: 45 },
            boundaryGap: false
        },
        yAxis: {
            type: 'value',
            axisLabel: { color: '#666' }
        },
        series: [{
            data: realtimeDataPoints.map(p => p.value),
            type: 'line',
            smooth: true,
            areaStyle: {
                color: {
                    type: 'linear',
                    x: 0, y: 0, x2: 0, y2: 1,
                    colorStops: [
                        { offset: 0, color: 'rgba(16, 185, 129, 0.3)' },
                        { offset: 1, color: 'rgba(16, 185, 129, 0.05)' }
                    ]
                }
            },
            lineStyle: { color: '#10b981', width: 2 },
            itemStyle: { color: '#10b981' },
            symbol: 'circle',
            symbolSize: 4
        }],
        tooltip: {
            trigger: 'axis',
            backgroundColor: 'rgba(0,0,0,0.8)',
            textStyle: { color: '#fff' }
        },
        grid: { left: '3%', right: '4%', bottom: '15%', containLabel: true },
        animation: false
    });
}

function initWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/v1/admin/dashboard/ws`;

    try {
        ws = new WebSocket(wsUrl);

        ws.onopen = function() {
            wsConnected = true;
            updateWsStatus(true);
            console.log('WebSocket connected');
        };

        ws.onmessage = function(event) {
            try {
                const data = JSON.parse(event.data);
                handleRealtimeData(data);
            } catch (e) {
                console.error('Parse WebSocket data failed:', e);
            }
        };

        ws.onerror = function(error) {
            console.error('WebSocket error:', error);
            wsConnected = false;
            updateWsStatus(false);
        };

        ws.onclose = function() {
            wsConnected = false;
            updateWsStatus(false);
            console.log('WebSocket disconnected, reconnecting in', WS_RECONNECT_DELAY, 'ms...');
            setTimeout(initWebSocket, WS_RECONNECT_DELAY);
        };
    } catch (e) {
        console.error('WebSocket init failed:', e);
        wsConnected = false;
        updateWsStatus(false);
        setTimeout(initWebSocket, WS_RECONNECT_DELAY);
    }
}

function updateWsStatus(connected) {
    const statusEl = document.getElementById('wsStatus');
    if (!statusEl) return;

    if (connected) {
        statusEl.className = 'badge badge-success';
        statusEl.innerHTML = '<i class="fas fa-wifi mr-1"></i>已连接';
    } else {
        statusEl.className = 'badge badge-danger';
        statusEl.innerHTML = '<i class="fas fa-wifi mr-1"></i>已断开';
    }
}

function handleRealtimeData(data) {
    if (data.type === 'metrics') {
        updateRealtimeMetrics(data.payload);
    } else if (data.type === 'activity') {
        addActivityRow(data.payload);
    } else if (data.type === 'stats') {
        updateStats(data.payload);
    }
}

function updateRealtimeMetrics(data) {
    if (data.total_requests !== undefined) {
        animateNumber('totalRequests', data.total_requests);
    }

    if (data.requests_per_second !== undefined) {
        document.getElementById('currentQPS').textContent = data.requests_per_second.toFixed(0) + ' QPS';
        updateRealtimeChart(data.requests_per_second);
    }

    if (data.system_status) {
        updateSystemStatus(data.system_status);
    }

    if (data.resource_usage) {
        updateResourceUsage(data.resource_usage);
    }
}

function addActivityRow(data) {
    const tbody = document.getElementById('recentActivity');
    if (!tbody) return;

    const row = document.createElement('tr');
    row.innerHTML = `
        <td><small class="text-muted">${data.time || formatTime(new Date())}</small></td>
        <td>${escapeHtml(data.event || '-')}</td>
        <td><code>${escapeHtml(data.user || '-')}</code></td>
        <td><span class="badge ${getStatusBadgeClass(data.status)}">${getStatusText(data.status)}</span></td>
    `;

    tbody.insertBefore(row, tbody.firstChild);

    while (tbody.children.length > 8) {
        tbody.removeChild(tbody.lastChild);
    }
}

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
    let labels, dataValues;

    if (period === 'hour') {
        labels = Array.from({ length: 24 }, (_, i) => `${i}:00`);
        dataValues = Array.from({ length: 24 }, () => Math.floor(Math.random() * 5000) + 1000);
    } else if (period === 'day') {
        labels = ['周一', '周二', '周三', '周四', '周五', '周六', '周日'];
        dataValues = [12000, 15000, 18000, 16000, 20000, 25000, 22000];
    } else {
        labels = Array.from({ length: 7 }, (_, i) => `第${i + 1}周`);
        dataValues = [85000, 92000, 105000, 98000, 120000, 135000, 145000];
    }

    return { labels, data: dataValues };
}

function updateRequestTrendChart(data) {
    if (!requestTrendChart || !data) return;

    const labels = data.labels || [];
    const values = data.data || [];

    requestTrendChart.setOption({
        xAxis: {
            type: 'category',
            data: labels,
            axisLabel: { color: '#666', rotate: 45 }
        },
        yAxis: {
            type: 'value',
            axisLabel: { color: '#666' }
        },
        series: [{
            data: values,
            type: 'line',
            smooth: true,
            areaStyle: {
                color: {
                    type: 'linear',
                    x: 0, y: 0, x2: 0, y2: 1,
                    colorStops: [
                        { offset: 0, color: 'rgba(59, 130, 246, 0.3)' },
                        { offset: 1, color: 'rgba(59, 130, 246, 0.05)' }
                    ]
                }
            },
            lineStyle: { color: '#3b82f6', width: 2 },
            itemStyle: { color: '#3b82f6' },
            symbol: 'circle',
            symbolSize: 4
        }],
        tooltip: {
            trigger: 'axis',
            backgroundColor: 'rgba(0,0,0,0.8)',
            textStyle: { color: '#fff' },
            padding: 12
        },
        grid: { left: '3%', right: '4%', bottom: '15%', containLabel: true }
    }, false);
}

function updateRealtimeChart(value) {
    if (!realtimeChart) return;

    dataBuffer.push({ value: value, timestamp: Date.now() });

    if (dataBuffer.length > 10) {
        const avg = dataBuffer.reduce((sum, item) => sum + item.value, 0) / dataBuffer.length;
        dataBuffer = [];
        value = Math.round(avg);
    }

    if (chartUpdateThrottle) return;
    
    chartUpdateThrottle = setTimeout(() => {
        chartUpdateThrottle = null;
        
        const now = new Date();
        const timeLabel = formatTime(now);

        realtimeDataPoints.push({ time: timeLabel, value: value });

        if (realtimeDataPoints.length > MAX_REALTIME_POINTS) {
            realtimeDataPoints.shift();
        }

        const cachedOption = chartCache.get('realtime');
        if (cachedOption) {
            cachedOption.xAxis.data = realtimeDataPoints.map(p => p.time);
            cachedOption.series[0].data = realtimeDataPoints.map(p => p.value);
            realtimeChart.setOption(cachedOption, false);
        } else {
            realtimeChart.setOption({
                xAxis: {
                    data: realtimeDataPoints.map(p => p.time)
                },
                series: [{
                    data: realtimeDataPoints.map(p => p.value)
                }]
            }, false);
        }
    }, CHART_UPDATE_THROTTLE_MS);
}

function clearChartCache() {
    chartCache.clear();
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
