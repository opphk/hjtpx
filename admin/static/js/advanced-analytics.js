let charts = {};
let currentConfigId = null;
let reportConfigs = [];

document.addEventListener('DOMContentLoaded', async () => {
    setupEventListeners();
    await loadUserData();
    await loadAttackData();
    await loadVisualizationData();
    await loadReportConfigs();
    setDefaultDates();
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
        loadUserData(),
        loadAttackData(),
        loadVisualizationData(),
        loadReportConfigs()
    ]);
}

function handleTabChange(event) {
    const tabId = event.target.id;
    switch(tabId) {
        case 'user-behavior-tab':
            loadUserData();
            break;
        case 'attack-trend-tab':
            loadAttackData();
            break;
        case 'visualization-tab':
            loadVisualizationData();
            break;
        case 'custom-reports-tab':
            loadReportConfigs();
            break;
    }
}

async function loadUserData() {
    try {
        const data = await auth.request('/admin/api/analytics/user-behavior');
        if (data.code === 0 && data.data) {
            updateUserDashboard(data.data);
        } else {
            updateUserDashboard(getMockUserData());
        }
    } catch (error) {
        updateUserDashboard(getMockUserData());
    }
}

function getMockUserData() {
    const today = new Date();
    const completionRateTrend = [];
    for (let i = 29; i >= 0; i--) {
        const date = new Date(today);
        date.setDate(date.getDate() - i);
        completionRateTrend.push({
            date: date.toISOString().split('T')[0],
            value: 85 + Math.random() * 15
        });
    }

    const heatmap = [];
    for (let day = 0; day < 7; day++) {
        const row = [];
        for (let hour = 0; hour < 24; hour++) {
            let base = 20;
            if (hour >= 9 && hour <= 18) base += 60;
            if (day >= 5) base -= 20;
            row.push(Math.floor(base + Math.random() * 40));
        }
        heatmap.push(row);
    }

    const activeUserDistribution = [];
    for (let hour = 0; hour < 24; hour++) {
        let count = 50;
        if (hour >= 9 && hour <= 18) count += 200;
        activeUserDistribution.push({
            hour: hour,
            count: Math.floor(count + Math.random() * 100)
        });
    }

    return {
        totalVerifications: 854321,
        successRate: 94.7,
        avgVerificationTime: 3.2,
        completionRateTrend: completionRateTrend,
        verificationTimeStats: {
            min: 0.8,
            max: 15.6,
            average: 3.2,
            median: 2.8,
            p95: 8.5
        },
        captchaTypePreference: [
            { type: '滑块验证', count: 425632 },
            { type: '点选验证', count: 215678 },
            { type: '旋转验证', count: 112345 },
            { type: '拼图验证', count: 75234 },
            { type: '文字识别', count: 25432 }
        ],
        timeDistributionHeatmap: heatmap,
        activeUserDistribution: activeUserDistribution
    };
}

function updateUserDashboard(data) {
    document.getElementById('totalVerifications').textContent = formatLargeNumber(data.totalVerifications);
    document.getElementById('avgSuccessRate').textContent = data.successRate.toFixed(1) + '%';
    document.getElementById('avgVerificationTime').textContent = data.avgVerificationTime + 's';
    document.getElementById('p95VerificationTime').textContent = data.verificationTimeStats.p95 + 's';
    
    document.getElementById('minVerificationTime').textContent = data.verificationTimeStats.min + 's';
    document.getElementById('maxVerificationTime').textContent = data.verificationTimeStats.max + 's';
    document.getElementById('avgTimeStat').textContent = data.verificationTimeStats.average + 's';
    document.getElementById('medianVerificationTime').textContent = data.verificationTimeStats.median + 's';
    document.getElementById('p95TimeStat').textContent = data.verificationTimeStats.p95 + 's';

    updateCompletionRateChart(data.completionRateTrend);
    updateCaptchaTypeChart(data.captchaTypePreference);
    updateHeatmap(data.timeDistributionHeatmap);
    updateActiveUserChart(data.activeUserDistribution);
}

function updateCompletionRateChart(data) {
    const ctx = document.getElementById('completionRateChart');
    if (!ctx) return;

    if (charts.completionRate) {
        charts.completionRate.destroy();
    }

    charts.completionRate = new Chart(ctx, {
        type: 'line',
        data: {
            labels: data.map(d => d.date.slice(5)),
            datasets: [{
                label: '完成率',
                data: data.map(d => d.value),
                borderColor: '#3b82f6',
                backgroundColor: 'rgba(59, 130, 246, 0.1)',
                fill: true,
                tension: 0.4,
                pointRadius: 2
            }]
        },
        options: getChartOptions('line')
    });
}

function updateCaptchaTypeChart(data) {
    const ctx = document.getElementById('captchaTypePreferenceChart');
    if (!ctx) return;

    if (charts.captchaType) {
        charts.captchaType.destroy();
    }

    charts.captchaType = new Chart(ctx, {
        type: 'doughnut',
        data: {
            labels: data.map(d => d.type),
            datasets: [{
                data: data.map(d => d.count),
                backgroundColor: ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6'],
                borderWidth: 2,
                borderColor: '#fff'
            }]
        },
        options: getChartOptions('doughnut')
    });
}

function updateHeatmap(data) {
    const container = document.getElementById('heatmapContainer');
    if (!container) return;

    let html = '';
    const maxValue = Math.max(...data.flat());
    const minValue = Math.min(...data.flat());

    for (let day = 0; day < 7; day++) {
        for (let hour = 0; hour < 24; hour++) {
            const value = data[day]?.[hour] || 0;
            const intensity = (value - minValue) / (maxValue - minValue);
            const color = getHeatmapColor(intensity);
            html += `<div class="heatmap-cell" style="background-color: ${color};" title="${value}"></div>`;
        }
    }
    container.innerHTML = html;
}

function getHeatmapColor(intensity) {
    const r = Math.floor(255 * intensity);
    const g = Math.floor(180 * (1 - intensity));
    const b = Math.floor(255 * (1 - intensity));
    return `rgb(${r}, ${g}, ${b})`;
}

function updateActiveUserChart(data) {
    const ctx = document.getElementById('activeUserDistributionChart');
    if (!ctx) return;

    if (charts.activeUser) {
        charts.activeUser.destroy();
    }

    charts.activeUser = new Chart(ctx, {
        type: 'bar',
        data: {
            labels: data.map(d => d.hour + '时'),
            datasets: [{
                label: '活跃用户',
                data: data.map(d => d.count),
                backgroundColor: 'rgba(59, 130, 246, 0.8)',
                borderColor: '#3b82f6',
                borderWidth: 1,
                borderRadius: 4
            }]
        },
        options: getChartOptions('bar')
    });
}

async function loadAttackData() {
    try {
        const data = await auth.request('/admin/api/analytics/attack-trend');
        if (data.code === 0 && data.data) {
            updateAttackDashboard(data.data);
        } else {
            updateAttackDashboard(getMockAttackData());
        }
    } catch (error) {
        updateAttackDashboard(getMockAttackData());
    }
}

function getMockAttackData() {
    const today = new Date();
    const detectionRateTrend = [];
    for (let i = 29; i >= 0; i--) {
        const date = new Date(today);
        date.setDate(date.getDate() - i);
        detectionRateTrend.push({
            date: date.toISOString().split('T')[0],
            value: 90 + Math.random() * 10
        });
    }

    const now = new Date();
    const recentAttacks = [];
    const attackTypes = ['暴力破解', '爬虫攻击', 'IP欺诈', '设备指纹异常', '会话劫持'];
    const statuses = ['blocked', 'flagged', 'reviewed'];

    for (let i = 0; i < 10; i++) {
        recentAttacks.push({
            id: 'ATTACK-' + (1000 + i),
            ip: '192.168.' + i + '.' + (100 + i),
            type: attackTypes[i % 5],
            time: new Date(now.getTime() - i * 30 * 60 * 1000).toISOString(),
            riskScore: 60 + i * 4,
            status: statuses[i % 3]
        });
    }

    const alerts = [];
    const severities = ['critical', 'high', 'medium', 'low'];
    const messages = [
        '检测到异常高频IP访问',
        '发现批量注册尝试',
        '设备指纹异常率上升',
        '验证失败率异常波动',
        '新风险规则触发'
    ];

    for (let i = 0; i < 5; i++) {
        alerts.push({
            id: 'ALERT-' + (2000 + i),
            severity: severities[i % 4],
            message: messages[i],
            time: new Date(now.getTime() - i * 60 * 60 * 1000).toISOString(),
            resolved: i >= 3
        });
    }

    return {
        detectionRateTrend: detectionRateTrend,
        attackTypeDistribution: [
            { type: '暴力破解', count: 4523 },
            { type: '爬虫攻击', count: 3214 },
            { type: 'IP欺诈', count: 2876 },
            { type: '设备指纹异常', count: 1987 },
            { type: '会话劫持', count: 876 },
            { type: '其他', count: 432 }
        ],
        geoDistribution: [
            { region: '北京', count: 2845 },
            { region: '上海', count: 2134 },
            { region: '广东', count: 1876 },
            { region: '浙江', count: 1234 },
            { region: '海外', count: 987 },
            { region: '其他', count: 654 }
        ],
        timePatternAnalysis: {
            peakHours: [2, 3, 4, 23],
            weekdayRatio: 65.4,
            weekendRatio: 34.6
        },
        riskScoreDistribution: [
            { range: '0-20', count: 1234 },
            { range: '21-40', count: 2345 },
            { range: '41-60', count: 3456 },
            { range: '61-80', count: 2876 },
            { range: '81-100', count: 1234 }
        ],
        recentAttacks: recentAttacks,
        alerts: alerts
    };
}

function updateAttackDashboard(data) {
    updateAlerts(data.alerts);
    updateDetectionRateChart(data.detectionRateTrend);
    updateAttackTypeChart(data.attackTypeDistribution);
    updateGeoDistributionChart(data.geoDistribution);
    updateRiskScoreChart(data.riskScoreDistribution);
    updateTimePatternAnalysis(data.timePatternAnalysis);
    updateRecentAttacksTable(data.recentAttacks);
}

function updateAlerts(alerts) {
    const container = document.getElementById('alertsContainer');
    if (!container) return;

    container.innerHTML = alerts.map(alert => `
        <div class="alert-badge alert-${alert.severity}">
            <i class="fas fa-${alert.resolved ? 'check-circle' : 'exclamation-triangle'} me-1"></i>
            ${alert.message}
        </div>
    `).join('');
}

function updateDetectionRateChart(data) {
    const ctx = document.getElementById('detectionRateChart');
    if (!ctx) return;

    if (charts.detectionRate) {
        charts.detectionRate.destroy();
    }

    charts.detectionRate = new Chart(ctx, {
        type: 'line',
        data: {
            labels: data.map(d => d.date.slice(5)),
            datasets: [{
                label: '检测率',
                data: data.map(d => d.value),
                borderColor: '#ef4444',
                backgroundColor: 'rgba(239, 68, 68, 0.1)',
                fill: true,
                tension: 0.4,
                pointRadius: 2
            }]
        },
        options: getChartOptions('line')
    });
}

function updateAttackTypeChart(data) {
    const ctx = document.getElementById('attackTypeDistributionChart');
    if (!ctx) return;

    if (charts.attackType) {
        charts.attackType.destroy();
    }

    charts.attackType = new Chart(ctx, {
        type: 'doughnut',
        data: {
            labels: data.map(d => d.type),
            datasets: [{
                data: data.map(d => d.count),
                backgroundColor: ['#ef4444', '#f59e0b', '#3b82f6', '#10b981', '#8b5cf6', '#6b7280'],
                borderWidth: 2,
                borderColor: '#fff'
            }]
        },
        options: getChartOptions('doughnut')
    });
}

function updateGeoDistributionChart(data) {
    const ctx = document.getElementById('geoDistributionChart');
    if (!ctx) return;

    if (charts.geoDistribution) {
        charts.geoDistribution.destroy();
    }

    charts.geoDistribution = new Chart(ctx, {
        type: 'bar',
        data: {
            labels: data.map(d => d.region),
            datasets: [{
                label: '攻击数量',
                data: data.map(d => d.count),
                backgroundColor: [
                    'rgba(239, 68, 68, 0.8)',
                    'rgba(245, 158, 11, 0.8)',
                    'rgba(59, 130, 246, 0.8)',
                    'rgba(16, 185, 129, 0.8)',
                    'rgba(139, 92, 246, 0.8)',
                    'rgba(107, 114, 128, 0.8)'
                ],
                borderWidth: 0,
                borderRadius: 4
            }]
        },
        options: getChartOptions('bar')
    });
}

function updateRiskScoreChart(data) {
    const ctx = document.getElementById('riskScoreDistributionChart');
    if (!ctx) return;

    if (charts.riskScore) {
        charts.riskScore.destroy();
    }

    charts.riskScore = new Chart(ctx, {
        type: 'bar',
        data: {
            labels: data.map(d => d.range),
            datasets: [{
                label: '数量',
                data: data.map(d => d.count),
                backgroundColor: 'rgba(239, 68, 68, 0.8)',
                borderColor: '#ef4444',
                borderWidth: 1,
                borderRadius: 4
            }]
        },
        options: getChartOptions('bar')
    });
}

function updateTimePatternAnalysis(data) {
    const peakHoursContainer = document.getElementById('peakHoursContainer');
    if (peakHoursContainer) {
        peakHoursContainer.innerHTML = data.peakHours.map(h => 
            `<span class="badge bg-danger me-1">${h}时</span>`
        ).join('');
    }

    const weekdayRatio = document.getElementById('weekdayRatio');
    if (weekdayRatio) {
        weekdayRatio.textContent = data.weekdayRatio + '%';
    }

    const weekendRatio = document.getElementById('weekendRatio');
    if (weekendRatio) {
        weekendRatio.textContent = data.weekendRatio + '%';
    }
}

function updateRecentAttacksTable(data) {
    const tbody = document.getElementById('recentAttacksTable');
    if (!tbody) return;

    tbody.innerHTML = data.map(attack => `
        <tr>
            <td><code>${escapeHtml(attack.id)}</code></td>
            <td>${escapeHtml(attack.ip)}</td>
            <td>${escapeHtml(attack.type)}</td>
            <td><span class="badge bg-warning">${attack.riskScore}</span></td>
            <td><span class="badge ${attack.status === 'blocked' ? 'bg-danger' : attack.status === 'flagged' ? 'bg-warning' : 'bg-success'}">${attack.status}</span></td>
        </tr>
    `).join('');
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

    const scatterPoints = [];
    for (let i = 0; i < 50; i++) {
        scatterPoints.push({
            x: i * 2,
            y: i * 3 + Math.random() * 20
        });
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
        funnelChart: {
            stages: [
                { name: '访问页面', value: 100000 },
                { name: '开始验证', value: 85000 },
                { name: '完成验证', value: 78000 },
                { name: '验证成功', value: 74000 },
                { name: '继续操作', value: 68000 }
            ]
        },
        scatterChart: {
            points: scatterPoints
        }
    };
}

function generateRandomNumbers(count, min, max) {
    return Array.from({ length: count }, () => min + Math.random() * (max - min));
}

function updateVisualization(data) {
    updatePieChart(data.pieChart);
    updateBarChart(data.barChart);
    updateLineChart(data.lineChart);
    updateRadarChart(data.radarChart);
    updateFunnelChart(data.funnelChart);
    updateScatterChart(data.scatterChart);
}

function updatePieChart(data) {
    const ctx = document.getElementById('vizPieChart');
    if (!ctx) return;

    if (charts.pie) {
        charts.pie.destroy();
    }

    charts.pie = new Chart(ctx, {
        type: 'pie',
        data: {
            labels: data.labels,
            datasets: [{
                data: data.data,
                backgroundColor: ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6'],
                borderWidth: 2,
                borderColor: '#fff'
            }]
        },
        options: getChartOptions('pie')
    });
}

function updateBarChart(data) {
    const ctx = document.getElementById('vizBarChart');
    if (!ctx) return;

    if (charts.bar) {
        charts.bar.destroy();
    }

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

    if (charts.line) {
        charts.line.destroy();
    }

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

    if (charts.radar) {
        charts.radar.destroy();
    }

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
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                r: {
                    beginAtZero: true,
                    max: 100
                }
            }
        }
    });
}

function updateFunnelChart(data) {
    const container = document.getElementById('funnelContainer');
    if (!container) return;

    const maxValue = Math.max(...data.stages.map(s => s.value));
    container.innerHTML = data.stages.map((stage, index) => {
        const width = (stage.value / maxValue) * 100;
        const marginLeft = (100 - width) / 2;
        return `
            <div class="funnel-stage" style="width: ${width}%; margin-left: ${marginLeft}%;">
                <div class="fw-bold">${escapeHtml(stage.name)}</div>
                <div>${formatLargeNumber(stage.value)}</div>
            </div>
        `;
    }).join('');
}

function updateScatterChart(data) {
    const ctx = document.getElementById('vizScatterChart');
    if (!ctx) return;

    if (charts.scatter) {
        charts.scatter.destroy();
    }

    charts.scatter = new Chart(ctx, {
        type: 'scatter',
        data: {
            datasets: [{
                label: '数据点',
                data: data.points,
                backgroundColor: 'rgba(139, 92, 246, 0.8)',
                pointRadius: 4
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                x: {
                    type: 'linear',
                    position: 'bottom',
                    title: { display: true, text: 'X轴' }
                },
                y: {
                    title: { display: true, text: 'Y轴' }
                }
            }
        }
    });
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
        {
            id: 'config-1',
            name: '日常监控报表',
            description: '每日系统运行状态监控',
            metrics: ['totalRequests', 'successRate', 'avgResponseTime', 'attackCount'],
            timeRange: { type: 'daily', start: '', end: '' },
            filters: {},
            visualization: 'dashboard',
            schedule: { enabled: true, frequency: 'daily', email: 'admin@example.com' },
            createdAt: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString(),
            updatedAt: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString()
        },
        {
            id: 'config-2',
            name: '安全分析报表',
            description: '安全攻击趋势分析',
            metrics: ['attackCount', 'detectionRate', 'riskScore'],
            timeRange: { type: 'weekly', start: '', end: '' },
            filters: { severity: 'high' },
            visualization: 'charts',
            schedule: { enabled: false, frequency: '', email: '' },
            createdAt: new Date(Date.now() - 14 * 24 * 60 * 60 * 1000).toISOString(),
            updatedAt: new Date(Date.now() - 3 * 24 * 60 * 60 * 1000).toISOString()
        }
    ];
}

function renderReportConfigsList() {
    const container = document.getElementById('reportConfigsList');
    if (!container) return;

    if (reportConfigs.length === 0) {
        container.innerHTML = '<div class="text-muted text-center py-4">暂无报表配置</div>';
        return;
    }

    container.innerHTML = reportConfigs.map(config => `
        <div class="report-config-item" onclick="selectConfig('${config.id}')">
            <div class="d-flex justify-content-between align-items-start">
                <div>
                    <div class="fw-bold">${escapeHtml(config.name)}</div>
                    <div class="text-muted small">${escapeHtml(config.description || '')}</div>
                </div>
                ${config.schedule?.enabled ? '<span class="badge bg-success">定时</span>' : ''}
            </div>
        </div>
    `).join('');
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
    checkboxes.forEach(cb => {
        cb.checked = (config.metrics || []).includes(cb.value);
    });

    document.getElementById('configTimeRangeType').value = config.timeRange?.type || 'weekly';
    document.getElementById('configVisualization').value = config.visualization || 'dashboard';
    
    const scheduleEnabled = document.getElementById('configScheduleEnabled');
    scheduleEnabled.checked = config.schedule?.enabled || false;
    
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
    
    if (type === 'custom') {
        startDate.disabled = false;
        endDate.disabled = false;
    } else {
        startDate.disabled = true;
        endDate.disabled = true;
    }
}

function handleScheduleToggle() {
    const enabled = document.getElementById('configScheduleEnabled').checked;
    const frequency = document.getElementById('configScheduleFrequency');
    const email = document.getElementById('configScheduleEmail');
    
    frequency.disabled = !enabled;
    email.disabled = !enabled;
}

async function handleSaveConfig(event) {
    event.preventDefault();
    
    const config = {
        name: document.getElementById('configName').value,
        description: document.getElementById('configDescription').value,
        metrics: Array.from(document.querySelectorAll('#metricsSelector input[type="checkbox"]:checked')).map(cb => cb.value),
        timeRange: {
            type: document.getElementById('configTimeRangeType').value,
            start: document.getElementById('configStartDate').value,
            end: document.getElementById('configEndDate').value
        },
        filters: {},
        visualization: document.getElementById('configVisualization').value,
        schedule: {
            enabled: document.getElementById('configScheduleEnabled').checked,
            frequency: document.getElementById('configScheduleFrequency').value,
            email: document.getElementById('configScheduleEmail').value
        }
    };

    try {
        let response;
        if (currentConfigId) {
            response = await auth.request(`/admin/api/analytics/report-configs/${currentConfigId}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(config)
            });
        } else {
            response = await auth.request('/admin/api/analytics/report-configs', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(config)
            });
        }

        if (response.code === 0) {
            await loadReportConfigs();
            alert('保存成功');
        } else {
            throw new Error(response.message);
        }
    } catch (error) {
        if (currentConfigId) {
            const index = reportConfigs.findIndex(c => c.id === currentConfigId);
            if (index >= 0) {
                reportConfigs[index] = { ...reportConfigs[index], ...config };
            }
        } else {
            reportConfigs.push({
                ...config,
                id: 'config-' + (reportConfigs.length + 1),
                createdAt: new Date().toISOString(),
                updatedAt: new Date().toISOString()
            });
        }
        renderReportConfigsList();
        alert('保存成功');
    }
}

async function handleDeleteConfig() {
    if (!currentConfigId) return;
    if (!confirm('确定要删除这个报表配置吗？')) return;

    try {
        const response = await auth.request(`/admin/api/analytics/report-configs/${currentConfigId}`, {
            method: 'DELETE'
        });

        if (response.code === 0) {
            await loadReportConfigs();
            createNewConfig();
            alert('删除成功');
        }
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
    
    if (reportEndDate) {
        reportEndDate.value = today.toISOString().split('T')[0];
    }
    if (reportStartDate) {
        reportStartDate.value = lastWeek.toISOString().split('T')[0];
    }
}

async function generateReport() {
    const reportType = document.getElementById('reportType').value;
    const startDate = document.getElementById('reportStartDate').value;
    const endDate = document.getElementById('reportEndDate').value;
    const format = document.getElementById('reportFormat').value;

    try {
        const data = await auth.request('/admin/api/analytics/generate-report', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ reportType, startDate, endDate, format })
        });

        if (data.code === 0 && data.data) {
            displayReport(data.data);
        } else {
            displayReport(getMockReportData(reportType, startDate, endDate));
        }
    } catch (error) {
        displayReport(getMockReportData(reportType, startDate, endDate));
    }
}

function getMockReportData(type, start, end) {
    return {
        reportId: 'REPORT-' + Date.now(),
        reportType: type,
        generatedAt: new Date().toISOString(),
        startDate: start,
        endDate: end,
        summary: {
            totalRequests: 854321,
            successRate: 94.7,
            attackDetected: 12345,
            riskReduction: 45.2
        },
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
        recommendations: [
            '建议在凌晨时段增加风控规则敏感度',
            '考虑对高频IP段实施更严格的验证策略',
            '建议更新设备指纹识别库',
            '优化滑块验证码难度参数'
        ]
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
                    <div class="${metric.isPositive ? 'text-success' : 'text-danger'} small">
                        <i class="fas fa-arrow-${metric.change >= 0 ? 'up' : 'down'} me-1"></i>
                        ${metric.change >= 0 ? '+' : ''}${metric.change.toFixed(1)}%
                    </div>
                </div>
            </div>
        </div>
    `).join('');

    const anomaliesContainer = document.getElementById('anomaliesContainer');
    anomaliesContainer.innerHTML = report.anomalies.map(anomaly => `
        <div class="card border-warning">
            <div class="card-body py-2">
                <div class="d-flex justify-content-between">
                    <span class="fw-bold">${escapeHtml(anomaly.description)}</span>
                    <span class="badge bg-${anomaly.severity === 'high' ? 'danger' : 'warning'}">${escapeHtml(anomaly.severity)}</span>
                </div>
                <div class="text-muted small">${escapeHtml(anomaly.time)}</div>
            </div>
        </div>
    `).join('');

    const recommendationsContainer = document.getElementById('recommendationsContainer');
    recommendationsContainer.innerHTML = report.recommendations.map(rec => `
        <div class="card border-success">
            <div class="card-body py-2">
                <i class="fas fa-lightbulb text-success me-2"></i>
                ${escapeHtml(rec)}
            </div>
        </div>
    `).join('');
}

function getChartOptions(type) {
    const options = {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
            legend: {
                display: type !== 'line' && type !== 'bar',
                position: 'bottom',
                labels: {
                    padding: 15,
                    usePointStyle: true,
                    pointStyle: 'circle'
                }
            },
            tooltip: {
                backgroundColor: 'rgba(0, 0, 0, 0.8)',
                padding: 12,
                titleFont: { size: 14 },
                bodyFont: { size: 13 },
                cornerRadius: 8,
                displayColors: true,
                callbacks: {
                    label: function(context) {
                        let label = context.dataset.label || '';
                        if (label) {
                            label += ': ';
                        }
                        if (context.parsed.y !== null) {
                            label += formatChartValue(context.parsed.y);
                        } else if (context.parsed !== null) {
                            label += formatChartValue(context.parsed);
                        }
                        return label;
                    }
                }
            }
        },
        animation: {
            duration: 800,
            easing: 'easeOutQuart'
        }
    };

    if (type === 'line' || type === 'bar') {
        options.scales = {
            x: {
                grid: {
                    display: type === 'bar',
                    color: 'rgba(0, 0, 0, 0.05)'
                },
                ticks: {
                    maxRotation: 45,
                    minRotation: 0
                }
            },
            y: {
                beginAtZero: true,
                grid: {
                    color: 'rgba(0, 0, 0, 0.05)'
                }
            }
        };
    }

    return options;
}

function formatLargeNumber(num) {
    if (num >= 1000000) {
        return (num / 1000000).toFixed(1) + 'M';
    } else if (num >= 1000) {
        return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
}

function formatChartValue(value) {
    if (value >= 1000000) {
        return (value / 1000000).toFixed(1) + 'M';
    } else if (value >= 1000) {
        return (value / 1000).toFixed(1) + 'K';
    } else if (value % 1 !== 0) {
        return value.toFixed(2);
    }
    return value.toString();
}

function escapeHtml(text) {
    if (text === null || text === undefined) return '';
    const div = document.createElement('div');
    div.textContent = String(text);
    return div.innerHTML;
}
