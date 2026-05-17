let trendChart, riskChart, appChart, captchaTypeChart, hourlyChart, realtimeChart, errorChart;
let updateInterval = null;
let currentInterval = 5000;
let currentChartType = 'line';
let realtimeData = [];

const mockData = {
    totalRequests: 8234567,
    totalSuccess: 8085894,
    avgResponse: 125,
    totalRisk: 45678,
    trendData: { labels: generateTimeLabels(24), data: [] },
    riskData: { labels: ['低风险', '中风险', '高风险', '严重'], data: [65, 25, 8, 2] },
    appData: [
        { name: '用户中心', requests: 15678, change: 12.5 },
        { name: '支付系统', requests: 23456, change: 8.3 },
        { name: '消息推送', requests: 8765, change: -2.1 },
        { name: '数据分析', requests: 12345, change: 5.7 },
        { name: '社交平台', requests: 45678, change: 15.2 }
    ],
    geoData: [
        { name: '北京', value: 35 },
        { name: '上海', value: 25 },
        { name: '广东', value: 18 },
        { name: '浙江', value: 12 },
        { name: '江苏', value: 10 }
    ],
    captchaTypeData: { labels: ['滑块验证', '点选验证', '旋转验证', '拼图验证'], data: [45, 30, 15, 10] },
    hourlyData: { labels: generateHourLabels(), data: [] },
    errorData: { labels: ['超时', '参数错误', '签名验证', '频率限制', '其他'], data: [] }
};

document.addEventListener('DOMContentLoaded', () => {
    initMockData();
    initAllCharts();
    setupEventListeners();
    startRealtimeUpdate();
    animateNumbers();
});

function initMockData() {
    mockData.trendData.data = Array.from({ length: 24 }, () => Math.floor(Math.random() * 5000) + 10000);
    mockData.hourlyData.data = Array.from({ length: 24 }, () => Math.floor(Math.random() * 3000) + 500);
    mockData.errorData.data = Array.from({ length: 5 }, () => Math.floor(Math.random() * 500) + 100);
    realtimeData = Array.from({ length: 30 }, (_, i) => ({
        time: new Date(Date.now() - (30 - i) * 1000),
        value: Math.floor(Math.random() * 200) + 100
    }));
}

function initAllCharts() {
    initTrendChart();
    initRiskChart();
    initAppChart();
    initCaptchaTypeChart();
    initHourlyChart();
    initRealtimeChart();
    initErrorChart();
    updateAppTable();
    updateGeoDistribution();
}

function setupEventListeners() {
    document.querySelectorAll('.interval-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            document.querySelectorAll('.interval-btn').forEach(b => b.classList.remove('active'));
            e.target.classList.add('active');
            const interval = e.target.dataset.interval;
            currentInterval = parseInterval(interval);
            restartRealtimeUpdate();
        });
    });

    document.getElementById('refreshBtn')?.addEventListener('click', () => {
        refreshAllData();
    });

    document.querySelectorAll('[data-type]').forEach(btn => {
        btn.addEventListener('click', (e) => {
            document.querySelectorAll('[data-type]').forEach(b => b.classList.remove('active'));
            e.target.classList.add('active');
            currentChartType = e.target.dataset.type;
            updateTrendChart();
        });
    });

    document.querySelectorAll('[data-view]').forEach(btn => {
        btn.addEventListener('click', (e) => {
            document.querySelectorAll('[data-view]').forEach(b => b.classList.remove('active'));
            e.target.classList.add('active');
            const view = e.target.dataset.view;
            toggleAppView(view);
        });
    });

    document.getElementById('fullscreenBtn')?.addEventListener('click', toggleFullscreen);
}

function parseInterval(interval) {
    if (interval === '5s') return 5000;
    if (interval === '10s') return 10000;
    if (interval === '30s') return 30000;
    if (interval === '1m') return 60000;
    return 5000;
}

function startRealtimeUpdate() {
    stopRealtimeUpdate();
    updateInterval = setInterval(updateRealtime, currentInterval);
}

function stopRealtimeUpdate() {
    if (updateInterval) {
        clearInterval(updateInterval);
        updateInterval = null;
    }
}

function restartRealtimeUpdate() {
    startRealtimeUpdate();
}

function updateRealtime() {
    const newValue = Math.floor(Math.random() * 200) + 100;
    realtimeData.push({ time: new Date(), value: newValue });
    if (realtimeData.length > 30) {
        realtimeData.shift();
    }

    if (realtimeChart) {
        realtimeChart.data.labels = realtimeData.map(d => formatTime(d.time));
        realtimeChart.data.datasets[0].data = realtimeData.map(d => d.value);
        realtimeChart.update('none');
    }

    updateMetrics();
}

function updateMetrics() {
    mockData.totalRequests += Math.floor(Math.random() * 100);
    mockData.totalSuccess += Math.floor(Math.random() * 95);
    mockData.totalRisk += Math.floor(Math.random() * 10);

    document.getElementById('totalRequests').textContent = formatLargeNumber(mockData.totalRequests);
    document.getElementById('totalSuccess').textContent = formatLargeNumber(mockData.totalSuccess);
    document.getElementById('totalRisk').textContent = formatLargeNumber(mockData.totalRisk);
}

function refreshAllData() {
    initMockData();
    initAllCharts();
    animateNumbers();
    showToast('数据已刷新', 'success');
}

function animateNumbers() {
    animateValue('totalRequests', 0, mockData.totalRequests, 1500);
    animateValue('totalSuccess', 0, mockData.totalSuccess, 1500);
    animateValue('avgResponse', 0, mockData.avgResponse, 1000, 'ms');
}

function animateValue(elementId, start, end, duration, suffix = '') {
    const element = document.getElementById(elementId);
    if (!element) return;

    const startTime = performance.now();
    const diff = end - start;

    function update(currentTime) {
        const elapsed = currentTime - startTime;
        const progress = Math.min(elapsed / duration, 1);
        const eased = 1 - Math.pow(1 - progress, 4);
        const current = Math.floor(start + diff * eased);

        element.textContent = formatLargeNumber(current) + suffix;

        if (progress < 1) {
            requestAnimationFrame(update);
        }
    }

    requestAnimationFrame(update);
}

function initTrendChart() {
    const ctx = document.getElementById('trendChart');
    if (!ctx) return;

    trendChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: mockData.trendData.labels,
            datasets: [{
                label: '请求量',
                data: mockData.trendData.data,
                borderColor: '#3b82f6',
                backgroundColor: 'rgba(59, 130, 246, 0.1)',
                fill: true,
                tension: 0.4,
                pointRadius: 3,
                pointHoverRadius: 6,
                borderWidth: 2
            }]
        },
        options: getChartOptions('line')
    });
}

function updateTrendChart() {
    if (!trendChart) return;

    let type = currentChartType;
    let fill = type === 'area' || type === 'line';

    trendChart.config.type = type === 'area' ? 'line' : type;
    trendChart.data.datasets[0].fill = fill;
    trendChart.data.datasets[0].backgroundColor = fill ? 'rgba(59, 130, 246, 0.1)' : 'transparent';
    trendChart.update();
}

function initRiskChart() {
    const ctx = document.getElementById('riskChart');
    if (!ctx) return;

    riskChart = new Chart(ctx, {
        type: 'doughnut',
        data: {
            labels: mockData.riskData.labels,
            datasets: [{
                data: mockData.riskData.data,
                backgroundColor: ['#10b981', '#f59e0b', '#f97316', '#ef4444'],
                borderWidth: 2,
                borderColor: '#fff'
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    position: 'bottom',
                    labels: { padding: 15, usePointStyle: true }
                }
            },
            cutout: '60%'
        }
    });
}

function initAppChart() {
    const ctx = document.getElementById('appChart');
    if (!ctx) return;

    appChart = new Chart(ctx, {
        type: 'bar',
        data: {
            labels: mockData.appData.map(a => a.name),
            datasets: [{
                label: '请求量',
                data: mockData.appData.map(a => a.requests),
                backgroundColor: [
                    'rgba(59, 130, 246, 0.8)',
                    'rgba(16, 185, 129, 0.8)',
                    'rgba(245, 158, 11, 0.8)',
                    'rgba(139, 92, 246, 0.8)',
                    'rgba(236, 72, 153, 0.8)'
                ],
                borderRadius: 6
            }]
        },
        options: getChartOptions('bar')
    });
}

function toggleAppView(view) {
    const chartView = document.getElementById('appComparisonChart');
    const tableView = document.getElementById('appComparisonTable');

    if (view === 'chart') {
        chartView.classList.remove('d-none');
        tableView.classList.add('d-none');
    } else {
        chartView.classList.add('d-none');
        tableView.classList.remove('d-none');
    }
}

function updateAppTable() {
    const tbody = document.getElementById('appTableBody');
    if (!tbody) return;

    const total = mockData.appData.reduce((sum, a) => sum + a.requests, 0);

    tbody.innerHTML = mockData.appData.map(app => {
        const percent = ((app.requests / total) * 100).toFixed(1);
        const trendClass = app.change >= 0 ? 'text-success' : 'text-danger';
        const trendIcon = app.change >= 0 ? 'fa-arrow-up' : 'fa-arrow-down';

        return `
            <tr>
                <td>${escapeHtml(app.name)}</td>
                <td class="fw-bold">${formatLargeNumber(app.requests)}</td>
                <td>
                    <div class="comparison-bar">
                        <div class="comparison-fill" style="width: ${percent}%; background: #3b82f6;">${percent}%</div>
                    </div>
                </td>
                <td class="${trendClass}"><i class="fas ${trendIcon} me-1"></i>${Math.abs(app.change)}%</td>
            </tr>
        `;
    }).join('');
}

function updateGeoDistribution() {
    const container = document.getElementById('geoDistributionList');
    if (!container) return;

    const maxValue = Math.max(...mockData.geoData.map(g => g.value));

    container.innerHTML = mockData.geoData.map((geo, index) => {
        const percent = (geo.value / maxValue) * 100;
        const colors = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6'];

        return `
            <div class="mb-3">
                <div class="d-flex justify-content-between mb-1">
                    <span class="fw-bold"><i class="fas fa-map-marker-alt me-2" style="color: ${colors[index]};"></i>${escapeHtml(geo.name)}</span>
                    <span class="text-muted">${geo.value}%</span>
                </div>
                <div class="comparison-bar">
                    <div class="comparison-fill" style="width: ${percent}%; background: ${colors[index]};">${geo.value}%</div>
                </div>
            </div>
        `;
    }).join('');
}

function initCaptchaTypeChart() {
    const ctx = document.getElementById('captchaTypeChart');
    if (!ctx) return;

    captchaTypeChart = new Chart(ctx, {
        type: 'pie',
        data: {
            labels: mockData.captchaTypeData.labels,
            datasets: [{
                data: mockData.captchaTypeData.data,
                backgroundColor: ['#3b82f6', '#10b981', '#f59e0b', '#ef4444'],
                borderWidth: 2,
                borderColor: '#fff'
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: { display: false }
            }
        }
    });

    updateCaptchaTypeLegend();
}

function updateCaptchaTypeLegend() {
    const container = document.getElementById('captchaTypeLegend');
    if (!container) return;

    const colors = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444'];

    container.innerHTML = mockData.captchaTypeData.labels.map((label, i) => `
        <div class="d-flex align-items-center mb-2">
            <span style="width: 12px; height: 12px; background: ${colors[i]}; border-radius: 3px; margin-right: 8px;"></span>
            <span class="small">${escapeHtml(label)}</span>
            <span class="ms-auto fw-bold">${mockData.captchaTypeData.data[i]}%</span>
        </div>
    `).join('');
}

function initHourlyChart() {
    const ctx = document.getElementById('hourlyChart');
    if (!ctx) return;

    hourlyChart = new Chart(ctx, {
        type: 'bar',
        data: {
            labels: mockData.hourlyData.labels,
            datasets: [{
                label: '请求量',
                data: mockData.hourlyData.data,
                backgroundColor: 'rgba(59, 130, 246, 0.6)',
                borderRadius: 4
            }]
        },
        options: getChartOptions('bar')
    });
}

function initRealtimeChart() {
    const ctx = document.getElementById('realtimeChart');
    if (!ctx) return;

    realtimeChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: realtimeData.map(d => formatTime(d.time)),
            datasets: [{
                label: '请求/秒',
                data: realtimeData.map(d => d.value),
                borderColor: '#10b981',
                backgroundColor: 'rgba(16, 185, 129, 0.1)',
                fill: true,
                tension: 0.4,
                borderWidth: 2,
                pointRadius: 2
            }]
        },
        options: {
            ...getChartOptions('line'),
            animation: { duration: 300 }
        }
    });
}

function initErrorChart() {
    const ctx = document.getElementById('errorChart');
    if (!ctx) return;

    errorChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: mockData.errorData.labels,
            datasets: [{
                label: '错误数',
                data: mockData.errorData.data,
                borderColor: '#ef4444',
                backgroundColor: 'rgba(239, 68, 68, 0.1)',
                fill: true,
                tension: 0.4,
                pointBackgroundColor: '#ef4444'
            }]
        },
        options: getChartOptions('line')
    });
}

function getChartOptions(type) {
    const base = {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
            legend: { display: false },
            tooltip: {
                backgroundColor: 'rgba(0, 0, 0, 0.8)',
                padding: 12,
                cornerRadius: 8,
                displayColors: false
            }
        },
        animation: { duration: 500 }
    };

    if (type === 'line' || type === 'bar') {
        base.scales = {
            x: {
                grid: { display: false },
                ticks: { maxRotation: 0 }
            },
            y: {
                beginAtZero: true,
                grid: { color: 'rgba(0, 0, 0, 0.05)' }
            }
        };
    }

    return base;
}

function toggleFullscreen() {
    const container = document.getElementById('realtimeChartContainer');
    if (!document.fullscreenElement) {
        container.requestFullscreen?.() || container.webkitRequestFullscreen?.();
    } else {
        document.exitFullscreen?.() || document.webkitExitFullscreen?.();
    }
}

function generateTimeLabels(count) {
    const labels = [];
    const now = new Date();
    for (let i = count - 1; i >= 0; i--) {
        const date = new Date(now - i * 60 * 60 * 1000);
        labels.push(`${date.getHours()}:00`);
    }
    return labels;
}

function generateHourLabels() {
    return Array.from({ length: 24 }, (_, i) => `${String(i).padStart(2, '0')}:00`);
}

function formatTime(date) {
    return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

function formatLargeNumber(num) {
    if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
    if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
    return num.toString();
}

function showToast(message, type = 'info') {
    const container = document.getElementById('toastContainer') || createToastContainer();
    const toast = document.createElement('div');
    toast.className = `toast align-items-center text-white bg-${type} border-0`;
    toast.setAttribute('role', 'alert');
    toast.innerHTML = `
        <div class="d-flex">
            <div class="toast-body">${escapeHtml(message)}</div>
            <button type="button" class="btn-close btn-close-white me-2 m-auto" data-bs-dismiss="toast"></button>
        </div>
    `;
    container.appendChild(toast);
    const bsToast = new bootstrap.Toast(toast);
    bsToast.show();
    toast.addEventListener('hidden.bs.toast', () => toast.remove());
}

function createToastContainer() {
    const container = document.createElement('div');
    container.id = 'toastContainer';
    container.className = 'toast-container position-fixed top-0 end-0 p-3';
    container.style.zIndex = '9999';
    document.body.appendChild(container);
    return container;
}

function escapeHtml(text) {
    if (text === null || text === undefined) return '';
    const div = document.createElement('div');
    div.textContent = String(text);
    return div.innerHTML;
}
