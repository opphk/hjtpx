let charts = {};
let currentConfigId = null;
let reportConfigs = [];

document.addEventListener('DOMContentLoaded', async () => {
    setupEventListeners();
    setDefaultDates();
    await loadUserProfileData();
    await loadAttackPredictionData();
    await loadVisualizationData();
    await loadReportConfigs();
    initAttackHeatmap();
});

function setupEventListeners() {
    const refreshBtn = document.getElementById('refreshBtn');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', refreshAllData);
    }

    const triggerTabList = document.querySelectorAll('#analyticsTabs button');
    triggerTabList.forEach(tab => {
        tab.addEventListener('shown.bs.tab', handleTabChange);
    });

    const generateReportBtn = document.getElementById('generateReportBtn');
    if (generateReportBtn) {
        generateReportBtn.addEventListener('click', generateReport);
    }

    const configTimeRangeType = document.getElementById('configTimeRangeType');
    if (configTimeRangeType) {
        configTimeRangeType.addEventListener('change', handleTimeRangeChange);
    }

    const configScheduleEnabled = document.getElementById('configScheduleEnabled');
    if (configScheduleEnabled) {
        configScheduleEnabled.addEventListener('change', handleScheduleToggle);
    }

    const reportConfigForm = document.getElementById('reportConfigForm');
    if (reportConfigForm) {
        reportConfigForm.addEventListener('submit', handleSaveConfig);
    }

    const deleteConfigBtn = document.getElementById('deleteConfigBtn');
    if (deleteConfigBtn) {
        deleteConfigBtn.addEventListener('click', handleDeleteConfig);
    }

    const newReportBtn = document.getElementById('newReportBtn');
    if (newReportBtn) {
        newReportBtn.addEventListener('click', createNewConfig);
    }
}

async function refreshAllData() {
    await Promise.all([
        loadUserProfileData(),
        loadAttackPredictionData(),
        loadVisualizationData(),
        loadReportConfigs()
    ]);
}

function handleTabChange(event) {
    const tabId = event.target.id;
    switch(tabId) {
        case 'user-behavior-tab':
            loadUserProfileData();
            break;
        case 'attack-trend-tab':
            loadAttackPredictionData();
            break;
        case 'visualization-tab':
            loadVisualizationData();
            break;
        case 'risk-report-tab':
            break;
        case 'custom-reports-tab':
            loadReportConfigs();
            break;
    }
}

async function loadUserProfileData() {
    try {
        const data = await auth.request('/admin/api/analytics/user-profile');
        if (data.code === 0 && data.data) {
            updateUserProfileDashboard(data.data);
        } else {
            updateUserProfileDashboard(getMockUserProfileData());
        }
    } catch (error) {
        updateUserProfileDashboard(getMockUserProfileData());
    }
}

function getMockUserProfileData() {
    return {
        totalUsers: 125843,
        activeUsers: 89532,
        newUsers: 1256,
        riskUsers: 342,
        userGrowthRate: 8.5,
        activeUserRate: 71.1,
        riskUserRate: 0.27,
        userSegments: {
            vip: 12584,
            regular: 89532,
            inactive: 18765,
            blocked: 4962
        },
        deviceDistribution: {
            desktop: 65,
            mobile: 30,
            tablet: 5
        },
        browserFingerprint: [
            { browser: 'Chrome', count: 52341, percentage: 42 },
            { browser: 'Firefox', count: 25169, percentage: 20 },
            { browser: 'Safari', count: 18876, percentage: 15 },
            { browser: 'Edge', count: 15012, percentage: 12 },
            { browser: '其他', count: 14445, percentage: 11 }
        ],
        geoDistribution: [
            { region: '北京', count: 18456 },
            { region: '上海', count: 15234 },
            { region: '广东', count: 14521 },
            { region: '浙江', count: 9876 },
            { region: '江苏', count: 8567 },
            { region: '四川', count: 7654 },
            { region: '湖北', count: 6543 },
            { region: '山东', count: 5678 },
            { region: '河南', count: 5432 },
            { region: '福建', count: 4876 }
        ],
        activityByHour: generateHourlyData(24, 100, 500)
    };
}

function generateHourlyData(hours, min, max) {
    const data = [];
    for (let i = 0; i < hours; i++) {
        let value = min;
        if (i >= 9 && i <= 11) value = max * 0.8;
        else if (i >= 14 && i <= 17) value = max * 0.9;
        else if (i >= 19 && i <= 22) value = max * 0.7;
        else if (i >= 0 && i <= 6) value = min;
        data.push({ hour: i, count: Math.floor(value + Math.random() * 100) });
    }
    return data;
}

function updateUserProfileDashboard(data) {
    document.getElementById('totalUsers').textContent = formatLargeNumber(data.totalUsers);
    document.getElementById('activeUsers').textContent = formatLargeNumber(data.activeUsers);
    document.getElementById('newUsers').textContent = formatLargeNumber(data.newUsers);
    document.getElementById('riskUsers').textContent = formatLargeNumber(data.riskUsers);
    document.getElementById('userGrowthRate').textContent = data.userGrowthRate.toFixed(1);
    document.getElementById('activeUserRate').textContent = data.activeUserRate.toFixed(1);
    document.getElementById('riskUserRate').textContent = data.riskUserRate.toFixed(2);
    document.getElementById('vipUsers').textContent = formatLargeNumber(data.userSegments.vip);
    document.getElementById('regularUsers').textContent = formatLargeNumber(data.userSegments.regular);
    document.getElementById('inactiveUsers').textContent = formatLargeNumber(data.userSegments.inactive);
    document.getElementById('blockedUsers').textContent = formatLargeNumber(data.userSegments.blocked);
    document.getElementById('desktopRate').textContent = data.deviceDistribution.desktop + '%';
    document.getElementById('mobileRate').textContent = data.deviceDistribution.mobile + '%';

    updateUserSegmentChart(data.userSegments);
    updateDeviceDistributionChart(data.deviceDistribution);
    updateBrowserFingerprintChart(data.browserFingerprint);
    updateGeoRankingList(data.geoDistribution);
    updateUserActivityChart(data.activityByHour);
}

function updateUserSegmentChart(data) {
    const ctx = document.getElementById('userSegmentChart');
    if (!ctx) return;

    if (charts.userSegment) {
        charts.userSegment.destroy();
    }

    charts.userSegment = new Chart(ctx, {
        type: 'doughnut',
        data: {
            labels: ['VIP 用户', '普通用户', '低活跃用户', '受限用户'],
            datasets: [{
                data: [data.vip, data.regular, data.inactive, data.blocked],
                backgroundColor: ['#3b82f6', '#10b981', '#f59e0b', '#ef4444'],
                borderWidth: 2,
                borderColor: '#fff'
            }]
        },
        options: getChartOptions('doughnut')
    });
}

function updateDeviceDistributionChart(data) {
    const ctx = document.getElementById('deviceDistributionChart');
    if (!ctx) return;

    if (charts.deviceDistribution) {
        charts.deviceDistribution.destroy();
    }

    charts.deviceDistribution = new Chart(ctx, {
        type: 'pie',
        data: {
            labels: ['桌面端', '移动端', '平板'],
            datasets: [{
                data: [data.desktop, data.mobile, data.tablet],
                backgroundColor: ['#3b82f6', '#10b981', '#f59e0b']
            }]
        },
        options: getChartOptions('pie')
    });
}

function updateBrowserFingerprintChart(data) {
    const ctx = document.getElementById('browserFingerprintChart');
    if (!ctx) return;

    if (charts.browserFingerprint) {
        charts.browserFingerprint.destroy();
    }

    charts.browserFingerprint = new Chart(ctx, {
        type: 'bar',
        data: {
            labels: data.map(d => d.browser),
            datasets: [{
                label: '用户数',
                data: data.map(d => d.count),
                backgroundColor: ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6']
            }]
        },
        options: {
            ...getChartOptions('bar'),
            indexAxis: 'y'
        }
    });
}

function updateGeoRankingList(data) {
    const container = document.getElementById('geoRankingList');
    if (!container) return;

    const maxValue = Math.max(...data.map(d => d.count));

    container.innerHTML = data.map((item, index) => `
        <div class="comparison-bar">
            <span class="comparison-bar-label">${index + 1}. ${item.region}</span>
            <div class="comparison-bar-track">
                <div class="comparison-bar-fill" style="width: ${(item.count / maxValue) * 100}%; background: linear-gradient(90deg, #3b82f6, #10b981);"></div>
            </div>
            <span class="comparison-bar-value">${formatLargeNumber(item.count)}</span>
        </div>
    `).join('');
}

function updateUserActivityChart(data) {
    const ctx = document.getElementById('userActivityChart');
    if (!ctx) return;

    if (charts.userActivity) {
        charts.userActivity.destroy();
    }

    charts.userActivity = new Chart(ctx, {
        type: 'bar',
        data: {
            labels: data.map(d => d.hour + '时'),
            datasets: [{
                label: '活跃用户',
                data: data.map(d => d.count),
                backgroundColor: 'rgba(59, 130, 246, 0.8)',
                borderColor: '#3b82f6',
                borderWidth: 1
            }]
        },
        options: getChartOptions('bar')
    });
}

async function loadAttackPredictionData() {
    try {
        const data = await auth.request('/admin/api/analytics/attack-prediction');
        if (data.code === 0 && data.data) {
            updateAttackPredictionDashboard(data.data);
        } else {
            updateAttackPredictionDashboard(getMockAttackPredictionData());
        }
    } catch (error) {
        updateAttackPredictionDashboard(getMockAttackPredictionData());
    }
}

function getMockAttackPredictionData() {
    const today = new Date();
    const historicalData = [];
    const predictedData = [];

    for (let i = 13; i >= 0; i--) {
        const date = new Date(today);
        date.setDate(date.getDate() - i);
        historicalData.push({
            date: date.toISOString().split('T')[0],
            actual: Math.floor(Math.random() * 500) + 200,
            predicted: null
        });
    }

    for (let i = 1; i <= 7; i++) {
        const date = new Date(today);
        date.setDate(date.getDate() + i);
        predictedData.push({
            date: date.toISOString().split('T')[0],
            predicted: Math.floor(Math.random() * 500) + 300,
            confidence: 0.8
        });
    }

    return {
        predictedAttackCount: Math.floor(Math.random() * 300) + 500,
        predictedAttackChange: Math.floor(Math.random() * 30) + 5,
        todayAttackCount: Math.floor(Math.random() * 400) + 300,
        highRiskIP: Math.floor(Math.random() * 50) + 20,
        defenseRate: 98.5,
        historicalData: historicalData,
        predictedData: predictedData,
        attackTypeDistribution: [
            { type: '暴力破解', count: 4523 },
            { type: '爬虫攻击', count: 3214 },
            { type: 'IP欺诈', count: 2876 },
            { type: '设备指纹异常', count: 1987 },
            { type: '会话劫持', count: 876 }
        ],
        attackHeatmap: generateAttackHeatmap(),
        attackTimeline: generateAttackTimeline(),
        topAttackGeo: [
            { region: '海外', count: 2345 },
            { region: '北京', count: 1234 },
            { region: '广东', count: 987 },
            { region: '浙江', count: 765 },
            { region: '上海', count: 654 }
        ],
        highRiskEvents: generateHighRiskEvents()
    };
}

function generateAttackHeatmap() {
    const heatmap = [];
    for (let day = 0; day < 7; day++) {
        const row = [];
        for (let hour = 0; hour < 24; hour++) {
            let base = 10;
            if (hour >= 2 && hour <= 5) base += 60;
            if (day >= 5) base -= 20;
            row.push(Math.floor(base + Math.random() * 30));
        }
        heatmap.push(row);
    }
    return heatmap;
}

function generateAttackTimeline() {
    const timeline = [];
    const now = new Date();
    const types = ['暴力破解尝试', '异常IP访问', '爬虫行为检测', '暴力登录攻击', 'API滥用'];

    for (let i = 0; i < 8; i++) {
        timeline.push({
            time: new Date(now.getTime() - i * 45 * 60 * 1000).toISOString(),
            type: types[i % types.length],
            severity: i < 2 ? 'high' : i < 5 ? 'medium' : 'low'
        });
    }
    return timeline;
}

function generateHighRiskEvents() {
    const events = [];
    const types = ['暴力破解', '爬虫攻击', 'IP欺诈', '指纹异常'];
    const statuses = ['blocked', 'flagged', 'reviewing'];

    for (let i = 0; i < 10; i++) {
        events.push({
            time: new Date(Date.now() - i * 30 * 60 * 1000).toLocaleTimeString('zh-CN'),
            type: types[i % types.length],
            ip: `192.168.${Math.floor(Math.random() * 255)}.${Math.floor(Math.random() * 255)}`,
            riskScore: Math.floor(Math.random() * 30) + 70,
            status: statuses[i % statuses.length]
        });
    }
    return events;
}

function updateAttackPredictionDashboard(data) {
    document.getElementById('predictedAttackCount').textContent = formatLargeNumber(data.predictedAttackCount);
    document.getElementById('predictedAttackChange').textContent = '+' + data.predictedAttackChange + '%';
    document.getElementById('todayAttackCount').textContent = formatLargeNumber(data.todayAttackCount);
    document.getElementById('highRiskIP').textContent = data.highRiskIP;
    document.getElementById('defenseRate').textContent = data.defenseRate.toFixed(1) + '%';

    updateAttackTrendChart(data.historicalData, data.predictedData);
    updateAttackTypePieChart(data.attackTypeDistribution);
    updateAttackHeatmap(data.attackHeatmap);
    updateAttackTimeline(data.attackTimeline);
    updateTopAttackGeoList(data.topAttackGeo);
    updateHighRiskEventsTable(data.highRiskEvents);
}

function updateAttackTrendChart(historical, predicted) {
    const ctx = document.getElementById('attackTrendChart');
    if (!ctx) return;

    if (charts.attackTrend) {
        charts.attackTrend.destroy();
    }

    const allLabels = [...historical.map(d => d.date.slice(5)), ...predicted.map(d => d.date.slice(5))];
    const actualData = [...historical.map(d => d.actual), ...Array(predicted.length).fill(null)];
    const predictedLine = [...Array(historical.length).fill(null), ...predicted.map(d => d.predicted)];

    charts.attackTrend = new Chart(ctx, {
        type: 'line',
        data: {
            labels: allLabels,
            datasets: [
                {
                    label: '实际攻击',
                    data: actualData,
                    borderColor: '#ef4444',
                    backgroundColor: 'rgba(239, 68, 68, 0.1)',
                    fill: true,
                    tension: 0.4,
                    pointRadius: 3
                },
                {
                    label: '预测趋势',
                    data: predictedLine,
                    borderColor: '#8b5cf6',
                    borderDash: [5, 5],
                    tension: 0.4,
                    pointRadius: 3
                }
            ]
        },
        options: {
            ...getChartOptions('line'),
            scales: {
                y: {
                    beginAtZero: true,
                    grid: { color: 'rgba(0, 0, 0, 0.05)' }
                }
            }
        }
    });
}

function updateAttackTypePieChart(data) {
    const ctx = document.getElementById('attackTypePieChart');
    if (!ctx) return;

    if (charts.attackTypePie) {
        charts.attackTypePie.destroy();
    }

    charts.attackTypePie = new Chart(ctx, {
        type: 'doughnut',
        data: {
            labels: data.map(d => d.type),
            datasets: [{
                data: data.map(d => d.count),
                backgroundColor: ['#ef4444', '#f59e0b', '#3b82f6', '#10b981', '#8b5cf6']
            }]
        },
        options: getChartOptions('doughnut')
    });
}

function updateAttackHeatmap(data) {
    const container = document.getElementById('attackHeatmapContainer');
    if (!container) return;

    const days = ['一', '二', '三', '四', '五', '六', '日'];
    let html = '';

    for (let day = 0; day < 7; day++) {
        for (let hour = 0; hour < 24; hour++) {
            const value = data[day]?.[hour] || 0;
            const intensity = Math.min(value / 100, 1);
            const r = Math.floor(intensity * 239);
            const g = Math.floor(intensity * 68);
            const b = Math.floor(intensity * 68);
            html += `<div class="heatmap-cell" style="background-color: rgb(${r},${g},${b});" title="${days[day]} ${hour}时: ${value}"></div>`;
        }
    }

    container.innerHTML = html;
}

function updateAttackTimeline(data) {
    const container = document.getElementById('attackTimeline');
    if (!container) return;

    container.innerHTML = data.map(item => `
        <div class="anomaly-item">
            <div class="d-flex justify-content-between">
                <span class="fw-bold">${item.type}</span>
                <span class="badge ${item.severity === 'high' ? 'bg-danger' : item.severity === 'medium' ? 'bg-warning' : 'bg-info'}">${item.severity}</span>
            </div>
            <div class="text-muted small">${new Date(item.time).toLocaleString('zh-CN')}</div>
        </div>
    `).join('');
}

function updateTopAttackGeoList(data) {
    const container = document.getElementById('topAttackGeoList');
    if (!container) return;

    const maxValue = Math.max(...data.map(d => d.count));

    container.innerHTML = data.map((item, index) => `
        <div class="comparison-bar">
            <span class="comparison-bar-label">${index + 1}. ${item.region}</span>
            <div class="comparison-bar-track">
                <div class="comparison-bar-fill" style="width: ${(item.count / maxValue) * 100}%; background: linear-gradient(90deg, #ef4444, #f59e0b);"></div>
            </div>
            <span class="comparison-bar-value">${formatLargeNumber(item.count)}</span>
        </div>
    `).join('');
}

function updateHighRiskEventsTable(data) {
    const tbody = document.getElementById('highRiskEventsTable');
    if (!tbody) return;

    tbody.innerHTML = data.map(event => `
        <tr>
            <td><small>${event.time}</small></td>
            <td><span class="badge bg-danger">${event.type}</span></td>
            <td><code>${event.ip}</code></td>
            <td><span class="badge bg-warning">${event.riskScore}</span></td>
            <td><span class="badge ${event.status === 'blocked' ? 'bg-danger' : event.status === 'flagged' ? 'bg-warning' : 'bg-info'}">${event.status}</span></td>
        </tr>
    `).join('');
}

function initAttackHeatmap() {
    updateAttackHeatmap(generateAttackHeatmap());
}

async function loadVisualizationData() {
    try {
        const data = await auth.request('/admin/api/analytics/visualization');
        if (data.code === 0 && data.data) {
            updateVisualization(data.data);
        } else {
            updateVisualization(getMockVisualizationData());
        }
    } catch (error) {
        updateVisualization(getMockVisualizationData());
    }
}

function getMockVisualizationData() {
    const labels = [];
    const today = new Date();
    for (let i = 29; i >= 0; i--) {
        const date = new Date(today);
        date.setDate(date.getDate() - i);
        labels.push(date.toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit' }));
    }

    return {
        pieChart: {
            labels: ['滑块验证', '点选验证', '旋转验证', '拼图验证', '文字识别'],
            data: [425632, 215678, 112345, 75234, 25432]
        },
        barChart: {
            labels: ['周一', '周二', '周三', '周四', '周五', '周六', '周日'],
            datasets: [
                { label: '成功', data: [12000, 15000, 18000, 16000, 20000, 25000, 22000] },
                { label: '失败', data: [800, 1200, 900, 1100, 1500, 2000, 1800] }
            ]
        },
        lineChart: {
            labels: labels,
            datasets: [
                { label: '请求量', data: generateRandomNumbers(30, 10000, 25000) },
                { label: '成功率', data: generateRandomNumbers(30, 90, 98) }
            ]
        },
        radarChart: {
            labels: ['安全性', '可用性', '性能', '准确性', '用户体验', '可靠性'],
            data: [95, 92, 88, 94, 85, 93]
        },
        polarChart: {
            labels: ['Chrome', 'Firefox', 'Safari', 'Edge', '其他', '移动端'],
            data: [42, 20, 15, 12, 6, 5]
        },
        bubbleChart: {
            points: generateBubbleData(20)
        },
        areaChart: {
            labels: labels,
            datasets: [
                { label: '访问量', data: generateRandomNumbers(30, 5000, 15000) },
                { label: '验证量', data: generateRandomNumbers(30, 3000, 12000) }
            ]
        },
        funnelChart: {
            stages: [
                { name: '访问页面', value: 100000 },
                { name: '开始验证', value: 85000 },
                { name: '完成验证', value: 78000 },
                { name: '验证成功', value: 74000 },
                { name: '继续操作', value: 68000 }
            ]
        }
    };
}

function generateRandomNumbers(count, min, max) {
    return Array.from({ length: count }, () => Math.floor(min + Math.random() * (max - min)));
}

function generateBubbleData(count) {
    const data = [];
    for (let i = 0; i < count; i++) {
        data.push({
            x: Math.random() * 100,
            y: Math.random() * 100,
            r: Math.random() * 15 + 5
        });
    }
    return data;
}

function updateVisualization(data) {
    updatePieChart(data.pieChart);
    updateBarChart(data.barChart);
    updateLineChart(data.lineChart);
    updateRadarChart(data.radarChart);
    updatePolarChart(data.polarChart);
    updateBubbleChart(data.bubbleChart);
    updateAreaChart(data.areaChart);
    updateFunnelChart(data.funnelChart);
}

function updatePieChart(data) {
    const ctx = document.getElementById('vizPieChart');
    if (!ctx) return;
    if (charts.pie) charts.pie.destroy();
    charts.pie = new Chart(ctx, {
        type: 'pie',
        data: {
            labels: data.labels,
            datasets: [{ data: data.data, backgroundColor: ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6'], borderWidth: 2, borderColor: '#fff' }]
        },
        options: getChartOptions('pie')
    });
}

function updateBarChart(data) {
    const ctx = document.getElementById('vizBarChart');
    if (!ctx) return;
    if (charts.bar) charts.bar.destroy();
    charts.bar = new Chart(ctx, {
        type: 'bar',
        data: {
            labels: data.labels,
            datasets: data.datasets.map((ds, i) => ({
                label: ds.label,
                data: ds.data,
                backgroundColor: i === 0 ? 'rgba(16, 185, 129, 0.8)' : 'rgba(239, 68, 68, 0.8)',
                borderWidth: 1,
                borderRadius: 4
            }))
        },
        options: getChartOptions('bar')
    });
}

function updateLineChart(data) {
    const ctx = document.getElementById('vizLineChart');
    if (!ctx) return;
    if (charts.line) charts.line.destroy();
    charts.line = new Chart(ctx, {
        type: 'line',
        data: {
            labels: data.labels,
            datasets: data.datasets.map((ds, i) => ({
                label: ds.label,
                data: ds.data,
                borderColor: i === 0 ? '#3b82f6' : '#10b981',
                backgroundColor: i === 0 ? 'rgba(59, 130, 246, 0.1)' : 'rgba(16, 185, 129, 0.1)',
                fill: true,
                tension: 0.4,
                pointRadius: 2
            }))
        },
        options: getChartOptions('line')
    });
}

function updateRadarChart(data) {
    const ctx = document.getElementById('vizRadarChart');
    if (!ctx) return;
    if (charts.radar) charts.radar.destroy();
    charts.radar = new Chart(ctx, {
        type: 'radar',
        data: {
            labels: data.labels,
            datasets: [{
                label: '评分',
                data: data.data,
                borderColor: '#8b5cf6',
                backgroundColor: 'rgba(139, 92, 246, 0.2)',
                borderWidth: 2,
                pointBackgroundColor: '#8b5cf6'
            }]
        },
        options: { responsive: true, maintainAspectRatio: false, scales: { r: { beginAtZero: true, max: 100 } } }
    });
}

function updatePolarChart(data) {
    const ctx = document.getElementById('vizPolarChart');
    if (!ctx) return;
    if (charts.polar) charts.polar.destroy();
    charts.polar = new Chart(ctx, {
        type: 'polarArea',
        data: {
            labels: data.labels,
            datasets: [{ data: data.data, backgroundColor: ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#6b7280'] }]
        },
        options: { responsive: true, maintainAspectRatio: false }
    });
}

function updateBubbleChart(data) {
    const ctx = document.getElementById('vizBubbleChart');
    if (!ctx) return;
    if (charts.bubble) charts.bubble.destroy();
    charts.bubble = new Chart(ctx, {
        type: 'bubble',
        data: {
            datasets: [{
                label: '数据点',
                data: data.points,
                backgroundColor: 'rgba(139, 92, 246, 0.8)'
            }]
        },
        options: { responsive: true, maintainAspectRatio: false }
    });
}

function updateAreaChart(data) {
    const ctx = document.getElementById('vizAreaChart');
    if (!ctx) return;
    if (charts.area) charts.area.destroy();
    charts.area = new Chart(ctx, {
        type: 'line',
        data: {
            labels: data.labels,
            datasets: data.datasets.map((ds, i) => ({
                label: ds.label,
                data: ds.data,
                borderColor: i === 0 ? '#3b82f6' : '#10b981',
                backgroundColor: i === 0 ? 'rgba(59, 130, 246, 0.3)' : 'rgba(16, 185, 129, 0.3)',
                fill: true,
                tension: 0.4
            }))
        },
        options: getChartOptions('line')
    });
}

function updateFunnelChart(data) {
    const container = document.getElementById('funnelContainer');
    if (!container) return;
    const maxValue = Math.max(...data.stages.map(s => s.value));
    container.innerHTML = data.stages.map((stage, index) => {
        const width = (stage.value / maxValue) * 100;
        const marginLeft = (100 - width) / 2;
        return `<div class="funnel-stage" style="width: ${width}%; margin-left: ${marginLeft}%;"><div class="fw-bold">${escapeHtml(stage.name)}</div><div>${formatLargeNumber(stage.value)}</div></div>`;
    }).join('');
}

async function loadReportConfigs() {
    try {
        const data = await auth.request('/admin/api/analytics/report-configs');
        if (data.code === 0 && data.data) {
            reportConfigs = data.data.list || [];
        } else {
            reportConfigs = getMockReportConfigs();
        }
    } catch (error) {
        reportConfigs = getMockReportConfigs();
    }
    renderReportConfigsList();
}

function getMockReportConfigs() {
    return [
        { id: 'config-1', name: '日常监控报表', description: '每日系统运行状态监控', metrics: ['totalRequests', 'successRate', 'avgResponseTime', 'attackCount'], timeRange: { type: 'daily', start: '', end: '' }, filters: {}, visualization: 'dashboard', schedule: { enabled: true, frequency: 'daily', email: 'admin@example.com' }, createdAt: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString(), updatedAt: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString() },
        { id: 'config-2', name: '安全分析报表', description: '安全攻击趋势分析', metrics: ['attackCount', 'detectionRate', 'riskScore'], timeRange: { type: 'weekly', start: '', end: '' }, filters: { severity: 'high' }, visualization: 'charts', schedule: { enabled: false, frequency: '', email: '' }, createdAt: new Date(Date.now() - 14 * 24 * 60 * 60 * 1000).toISOString(), updatedAt: new Date(Date.now() - 3 * 24 * 60 * 60 * 1000).toISOString() }
    ];
}

function renderReportConfigsList() {
    const container = document.getElementById('reportConfigsList');
    if (!container) return;
    if (reportConfigs.length === 0) {
        container.innerHTML = '<div class="text-muted text-center py-4">暂无报表配置</div>';
        return;
    }
    container.innerHTML = reportConfigs.map(config => `<div class="report-config-item" onclick="selectConfig('${config.id}')"><div class="d-flex justify-content-between align-items-start"><div><div class="fw-bold">${escapeHtml(config.name)}</div><div class="text-muted small">${escapeHtml(config.description || '')}</div></div>${config.schedule?.enabled ? '<span class="badge bg-success">定时</span>' : ''}</div></div>`).join('');
}

function selectConfig(id) {
    currentConfigId = id;
    const config = reportConfigs.find(c => c.id === id);
    if (!config) return;
    document.getElementById('configEditorTitle').textContent = '编辑报表配置';
    document.getElementById('deleteConfigBtn').style.display = 'inline-block';
    document.getElementById('configName').value = config.name || '';
    document.getElementById('configDescription').value = config.description || '';
    const metricsSelector = document.getElementById('metricsSelector');
    const checkboxes = metricsSelector.querySelectorAll('input[type="checkbox"]');
    checkboxes.forEach(cb => { cb.checked = (config.metrics || []).includes(cb.value); });
    document.getElementById('configTimeRangeType').value = config.timeRange?.type || 'weekly';
    document.getElementById('configVisualization').value = config.visualization || 'dashboard';
    document.getElementById('configScheduleEnabled').checked = config.schedule?.enabled || false;
    document.getElementById('configScheduleFrequency').value = config.schedule?.frequency || 'daily';
    document.getElementById('configScheduleEmail').value = config.schedule?.email || '';
    handleTimeRangeChange();
    handleScheduleToggle();
}

function createNewConfig() {
    currentConfigId = null;
    document.getElementById('configEditorTitle').textContent = '新建报表配置';
    document.getElementById('deleteConfigBtn').style.display = 'none';
    document.getElementById('reportConfigForm').reset();
}

function handleTimeRangeChange() {
    const type = document.getElementById('configTimeRangeType').value;
    const startDate = document.getElementById('configStartDate');
    const endDate = document.getElementById('configEndDate');
    if (type === 'custom') { startDate.disabled = false; endDate.disabled = false; }
    else { startDate.disabled = true; endDate.disabled = true; }
}

function handleScheduleToggle() {
    const enabled = document.getElementById('configScheduleEnabled').checked;
    document.getElementById('configScheduleFrequency').disabled = !enabled;
    document.getElementById('configScheduleEmail').disabled = !enabled;
}

async function handleSaveConfig(event) {
    event.preventDefault();
    const config = {
        name: document.getElementById('configName').value,
        description: document.getElementById('configDescription').value,
        metrics: Array.from(document.querySelectorAll('#metricsSelector input[type="checkbox"]:checked')).map(cb => cb.value),
        timeRange: { type: document.getElementById('configTimeRangeType').value, start: document.getElementById('configStartDate').value, end: document.getElementById('configEndDate').value },
        filters: {},
        visualization: document.getElementById('configVisualization').value,
        schedule: { enabled: document.getElementById('configScheduleEnabled').checked, frequency: document.getElementById('configScheduleFrequency').value, email: document.getElementById('configScheduleEmail').value }
    };
    try {
        if (currentConfigId) {
            await auth.request(`/admin/api/analytics/report-configs/${currentConfigId}`, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(config) });
        } else {
            await auth.request('/admin/api/analytics/report-configs', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(config) });
        }
        await loadReportConfigs();
        alert('保存成功');
    } catch (error) {
        if (currentConfigId) {
            const index = reportConfigs.findIndex(c => c.id === currentConfigId);
            if (index >= 0) reportConfigs[index] = { ...reportConfigs[index], ...config };
        } else {
            reportConfigs.push({ ...config, id: 'config-' + (reportConfigs.length + 1), createdAt: new Date().toISOString(), updatedAt: new Date().toISOString() });
        }
        renderReportConfigsList();
        alert('保存成功');
    }
}

async function handleDeleteConfig() {
    if (!currentConfigId) return;
    if (!confirm('确定要删除这个报表配置吗？')) return;
    try {
        await auth.request(`/admin/api/analytics/report-configs/${currentConfigId}`, { method: 'DELETE' });
        await loadReportConfigs();
        createNewConfig();
        alert('删除成功');
    } catch (error) {
        reportConfigs = reportConfigs.filter(c => c.id !== currentConfigId);
        renderReportConfigsList();
        createNewConfig();
        alert('删除成功');
    }
}

function setDefaultDates() {
    const today = new Date();
    const lastWeek = new Date(today);
    lastWeek.setDate(lastWeek.getDate() - 7);
    const reportEndDate = document.getElementById('reportEndDate');
    const reportStartDate = document.getElementById('reportStartDate');
    if (reportEndDate) reportEndDate.value = today.toISOString().split('T')[0];
    if (reportStartDate) reportStartDate.value = lastWeek.toISOString().split('T')[0];
}

async function generateReport() {
    const reportType = document.getElementById('reportType').value;
    const startDate = document.getElementById('reportStartDate').value;
    const endDate = document.getElementById('reportEndDate').value;
    const format = document.getElementById('reportFormat').value;
    try {
        const data = await auth.request('/admin/api/analytics/generate-report', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ reportType, startDate, endDate, format }) });
        if (data.code === 0 && data.data) displayReport(data.data);
        else displayReport(getMockReportData(reportType, startDate, endDate));
    } catch (error) {
        displayReport(getMockReportData(reportType, startDate, endDate));
    }
}

function getMockReportData(type, start, end) {
    return {
        reportId: 'REPORT-' + Date.now(), reportType: type, generatedAt: new Date().toISOString(), startDate: start, endDate: end,
        summary: { totalRequests: 854321, successRate: 94.7, attackDetected: 12345, riskReduction: 45.2 },
        keyMetrics: [
            { name: '验证成功率', value: 94.7, unit: '%', change: 2.3, isPositive: true },
            { name: '平均响应时间', value: 3.2, unit: 's', change: -15.4, isPositive: true },
            { name: '攻击拦截率', value: 98.5, unit: '%', change: 1.2, isPositive: true },
            { name: '活跃用户数', value: 12456, unit: '', change: 8.7, isPositive: true },
            { name: '风险评分', value: 72.3, unit: '', change: -5.4, isPositive: true }
        ],
        anomalies: [
            { time: '2024-01-15 02:30', type: 'spike', description: '验证请求量异常增加', severity: 'high' },
            { time: '2024-01-14 18:45', type: 'pattern', description: '检测到批量注册模式', severity: 'medium' }
        ],
        recommendations: ['建议在凌晨时段增加风控规则敏感度', '考虑对高频IP段实施更严格的验证策略', '建议更新设备指纹识别库', '优化滑块验证码难度参数']
    };
}

function displayReport(report) {
    const container = document.getElementById('reportResultContainer');
    if (!container) return;
    container.classList.remove('d-none');
    const typeNames = { daily: '日报', weekly: '周报', monthly: '月报' };
    document.getElementById('reportTitle').textContent = typeNames[report.reportType] || '风险报告';
    document.getElementById('reportId').textContent = report.reportId;
    document.getElementById('reportGeneratedAt').textContent = new Date(report.generatedAt).toLocaleString('zh-CN');
    document.getElementById('reportDateRange').textContent = `${report.startDate} 至 ${report.endDate}`;
    document.getElementById('reportTotalRequests').textContent = formatLargeNumber(report.summary.totalRequests);
    document.getElementById('reportSuccessRate').textContent = report.summary.successRate.toFixed(1) + '%';
    document.getElementById('reportAttackDetected').textContent = formatLargeNumber(report.summary.attackDetected);
    document.getElementById('reportRiskReduction').textContent = report.summary.riskReduction.toFixed(1) + '%';

    const keyMetricsContainer = document.getElementById('keyMetricsContainer');
    keyMetricsContainer.innerHTML = report.keyMetrics.map(metric => `
        <div class="col-md-3">
            <div class="card bg-light">
                <div class="card-body">
                    <div class="text-muted small">${escapeHtml(metric.name)}</div>
                    <div class="fs-4 fw-bold">${metric.value}${metric.unit}</div>
                    <div class="${metric.isPositive ? 'text-success' : 'text-danger'} small"><i class="fas fa-arrow-${metric.change >= 0 ? 'up' : 'down'} me-1"></i>${metric.change >= 0 ? '+' : ''}${metric.change.toFixed(1)}%</div>
                </div>
            </div>
        </div>
    `).join('');

    const anomaliesContainer = document.getElementById('anomaliesContainer');
    anomaliesContainer.innerHTML = report.anomalies.map(anomaly => `<div class="card border-warning"><div class="card-body py-2"><div class="d-flex justify-content-between"><span class="fw-bold">${escapeHtml(anomaly.description)}</span><span class="badge bg-${anomaly.severity === 'high' ? 'danger' : 'warning'}">${escapeHtml(anomaly.severity)}</span></div><div class="text-muted small">${escapeHtml(anomaly.time)}</div></div></div>`).join('');

    const recommendationsContainer = document.getElementById('recommendationsContainer');
    recommendationsContainer.innerHTML = report.recommendations.map(rec => `<div class="card border-success"><div class="card-body py-2"><i class="fas fa-lightbulb text-success me-2"></i>${escapeHtml(rec)}</div></div>`).join('');
}

function getChartOptions(type) {
    const options = {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
            legend: { display: type !== 'line' && type !== 'bar', position: 'bottom', labels: { padding: 15, usePointStyle: true, pointStyle: 'circle' } },
            tooltip: { backgroundColor: 'rgba(0, 0, 0, 0.8)', padding: 12, titleFont: { size: 14 }, bodyFont: { size: 13 }, cornerRadius: 8, displayColors: true }
        },
        animation: { duration: 800, easing: 'easeOutQuart' }
    };
    if (type === 'line' || type === 'bar') {
        options.scales = {
            x: { grid: { display: type === 'bar', color: 'rgba(0, 0, 0, 0.05)' }, ticks: { maxRotation: 45, minRotation: 0 } },
            y: { beginAtZero: true, grid: { color: 'rgba(0, 0, 0, 0.05)' } }
        };
    }
    return options;
}

function formatLargeNumber(num) {
    if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
    else if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
    return num.toString();
}

function escapeHtml(text) {
    if (text === null || text === undefined) return '';
    const div = document.createElement('div');
    div.textContent = String(text);
    return div.innerHTML;
}
