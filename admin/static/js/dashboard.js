let dashboardChart, trafficChart, riskDistributionChart, responseTimeChart, regionChart;
let refreshInterval = null;
let selectedTimeRange = '1h';

document.addEventListener('DOMContentLoaded', () => {
    initializeDashboard();
    loadDashboardData();
    startAutoRefresh();
});

function initializeDashboard() {
    document.getElementById('timeRangeSelector')?.addEventListener('change', (e) => {
        selectedTimeRange = e.target.value;
        loadDashboardData();
    });

    document.getElementById('manualRefresh')?.addEventListener('click', loadDashboardData);
    document.getElementById('exportDashboardBtn')?.addEventListener('click', exportDashboardData);

    initCharts();
}

function initCharts() {
    if (document.getElementById('dashboardChart')) {
        dashboardChart = echarts.init(document.getElementById('dashboardChart'));
        window.addEventListener('resize', () => dashboardChart.resize());
    }

    if (document.getElementById('trafficChart')) {
        trafficChart = echarts.init(document.getElementById('trafficChart'));
        window.addEventListener('resize', () => trafficChart.resize());
    }

    if (document.getElementById('riskDistributionChart')) {
        riskDistributionChart = echarts.init(document.getElementById('riskDistributionChart'));
        window.addEventListener('resize', () => riskDistributionChart.resize());
    }

    if (document.getElementById('responseTimeChart')) {
        responseTimeChart = echarts.init(document.getElementById('responseTimeChart'));
        window.addEventListener('resize', () => responseTimeChart.resize());
    }

    if (document.getElementById('regionChart')) {
        regionChart = echarts.init(document.getElementById('regionChart'));
        window.addEventListener('resize', () => regionChart.resize());
    }
}

async function loadDashboardData() {
    try {
        const response = await fetch(`/admin/dashboard?range=${selectedTimeRange}`);
        const result = await response.json();
        if (result.code === 0) {
            updateDashboard(result.data);
        }
    } catch (error) {
        console.error('加载仪表盘数据失败:', error);
        loadDemoDashboardData();
    }
}

function loadDemoDashboardData() {
    const data = generateDemoDashboardData();
    updateDashboard(data);
}

function generateDemoDashboardData() {
    const now = Date.now();
    const hours = [];
    const requestCounts = [];
    const riskCounts = [];
    const responseTimes = [];
    const successRates = [];

    for (let i = 23; i >= 0; i--) {
        const hour = new Date(now - i * 3600000);
        hours.push(`${hour.getHours().toString().padStart(2, '0')}:00`);
        requestCounts.push(1200 + Math.floor(Math.random() * 800));
        riskCounts.push(Math.floor(Math.random() * 150));
        responseTimes.push(50 + Math.floor(Math.random() * 80));
        successRates.push(95 + Math.random() * 4.5);
    }

    return {
        totalRequests: 45680,
        totalRequestsChange: 12.5,
        validRequests: 38920,
        validRequestsChange: 8.3,
        invalidRequests: 6760,
        invalidRequestsChange: -5.2,
        riskScore: 23.5,
        riskScoreChange: -3.2,
        anomalyRate: 2.1,
        anomalyRateChange: 1.8,
        realtimeMetrics: {
            currentQPS: 1256,
            avgResponseTime: 68,
            peakQPS: 2340,
            minResponseTime: 25,
            errorRate: 0.8,
            concurrentUsers: 4520
        },
        trafficData: {
            labels: hours,
            requests: requestCounts,
            risks: riskCounts
        },
        riskDistribution: [
            { name: '低风险', value: 6520, percent: 78.5 },
            { name: '中风险', value: 1280, percent: 15.4 },
            { name: '高风险', value: 450, percent: 5.4 },
            { name: '严重风险', value: 60, percent: 0.7 }
        ],
        responseTimeData: {
            labels: hours.slice(-12),
            avg: responseTimes.slice(-12),
            p95: responseTimes.slice(-12).map(t => t * 1.5),
            p99: responseTimes.slice(-12).map(t => t * 2)
        },
        regionData: [
            { name: '北京', value: 12580 },
            { name: '上海', value: 9860 },
            { name: '广州', value: 7650 },
            { name: '深圳', value: 6890 },
            { name: '杭州', value: 5430 },
            { name: '成都', value: 4280 },
            { name: '武汉', value: 3650 },
            { name: '其他', value: 1300 }
        ],
        topEndpoints: [
            { endpoint: '/api/auth/login', requests: 8520, risk: 15.2, avgTime: 85 },
            { endpoint: '/api/query/search', requests: 6320, risk: 28.5, avgTime: 120 },
            { endpoint: '/api/data/submit', requests: 5890, risk: 35.8, avgTime: 95 },
            { endpoint: '/api/user/profile', requests: 4560, risk: 8.3, avgTime: 45 },
            { endpoint: '/api/report/generate', requests: 3280, risk: 42.1, avgTime: 256 }
        ],
        recentAlerts: [
            { id: 'ALT001', type: '异常流量', severity: 'high', time: '2分钟前', message: '某IP请求频率异常，触发限流' },
            { id: 'ALT002', type: '风险行为', severity: 'medium', time: '15分钟前', message: '检测到异常轨迹模式' },
            { id: 'ALT003', type: '系统告警', severity: 'low', time: '1小时前', message: '响应时间超过阈值' },
            { id: 'ALT004', type: '安全威胁', severity: 'critical', time: '2小时前', message: '检测到暴力破解攻击' },
            { id: 'ALT005', type: '性能告警', severity: 'medium', time: '3小时前', message: '数据库查询缓慢' }
        ]
    };
}

function updateDashboard(data) {
    updateSummaryCards(data);
    updateRealtimeMetrics(data.realtimeMetrics);
    drawDashboardChart(data);
    drawTrafficChart(data.trafficData);
    drawRiskDistributionChart(data.riskDistribution);
    drawResponseTimeChart(data.responseTimeData);
    drawRegionChart(data.regionData);
    updateTopEndpoints(data.topEndpoints);
    updateRecentAlerts(data.recentAlerts);
}

function updateSummaryCards(data) {
    updateCard('totalRequests', data.totalRequests, data.totalRequestsChange);
    updateCard('validRequests', data.validRequests, data.validRequestsChange);
    updateCard('invalidRequests', data.invalidRequests, data.invalidRequestsChange);
    updateCard('riskScore', data.riskScore, data.riskScoreChange, 'score');
    updateCard('anomalyRate', data.anomalyRate, data.anomalyRateChange, 'percent');
}

function updateCard(id, value, change, format = 'number') {
    const card = document.getElementById(id);
    if (!card) return;

    let formattedValue;
    if (format === 'number') {
        formattedValue = value.toLocaleString();
    } else if (format === 'score') {
        formattedValue = value.toFixed(1);
    } else if (format === 'percent') {
        formattedValue = value.toFixed(1) + '%';
    }

    card.querySelector('.card-value').textContent = formattedValue;

    const changeEl = card.querySelector('.card-change');
    if (changeEl) {
        const icon = change >= 0 ? 'fa-arrow-up' : 'fa-arrow-down';
        const color = change >= 0 ? 'text-green-500' : 'text-red-500';
        changeEl.innerHTML = `<i class="fas ${icon} ${color}"></i> ${Math.abs(change).toFixed(1)}%`;
    }
}

function updateRealtimeMetrics(metrics) {
    if (!metrics) return;

    document.getElementById('currentQPS')?.textContent = metrics.currentQPS.toLocaleString();
    document.getElementById('avgResponseTime')?.textContent = metrics.avgResponseTime + 'ms';
    document.getElementById('peakQPS')?.textContent = metrics.peakQPS.toLocaleString();
    document.getElementById('minResponseTime')?.textContent = metrics.minResponseTime + 'ms';
    document.getElementById('errorRate')?.textContent = metrics.errorRate.toFixed(2) + '%';
    document.getElementById('concurrentUsers')?.textContent = metrics.concurrentUsers.toLocaleString();
}

function drawDashboardChart(data) {
    if (!dashboardChart) return;

    const option = {
        backgroundColor: 'transparent',
        title: {
            text: '实时风险仪表盘',
            left: 'center',
            textStyle: { fontSize: 16, fontWeight: 'bold', color: '#fff' }
        },
        series: [
            {
                type: 'gauge',
                startAngle: 90,
                endAngle: -270,
                pointer: {
                    show: true,
                    length: '60%',
                    itemStyle: { color: '#ef4444' }
                },
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
                splitLine: { length: 15, lineStyle: { color: '#9ca3af', width: 2 } },
                axisTick: { show: false },
                axisLabel: {
                    color: '#9ca3af',
                    distance: 20,
                    formatter: function(value) {
                        if (value === 0) return '0';
                        if (value === 50) return '50';
                        if (value === 100) return '100';
                        return '';
                    }
                },
                title: {
                    offsetCenter: [0, '75%'],
                    textStyle: { fontSize: 14, color: '#9ca3af' }
                },
                detail: {
                    valueAnimation: true,
                    formatter: '{value}',
                    offsetCenter: [0, '50%'],
                    textStyle: { fontSize: 48, fontWeight: 'bold', color: '#fff' }
                },
                data: [{ value: data.riskScore || 23.5, name: '风险评分' }]
            }
        ]
    };

    dashboardChart.setOption(option);
}

function drawTrafficChart(data) {
    if (!trafficChart || !data) return;

    const option = {
        backgroundColor: 'transparent',
        title: {
            text: '流量趋势',
            left: 'center',
            textStyle: { fontSize: 14, fontWeight: 'normal', color: '#9ca3af' }
        },
        tooltip: {
            trigger: 'axis',
            backgroundColor: 'rgba(0,0,0,0.8)',
            textStyle: { color: '#fff' },
            axisPointer: { type: 'cross' }
        },
        legend: {
            data: ['请求数', '风险数'],
            bottom: 0,
            textStyle: { color: '#9ca3af' }
        },
        grid: {
            left: '3%',
            right: '4%',
            bottom: '15%',
            top: '15%',
            containLabel: true
        },
        xAxis: {
            type: 'category',
            data: data.labels.slice(-12),
            axisLabel: { color: '#9ca3af', fontSize: 10, rotate: 45 },
            axisLine: { lineStyle: { color: '#374151' } }
        },
        yAxis: [
            {
                type: 'value',
                name: '请求数',
                nameTextStyle: { color: '#9ca3af' },
                axisLabel: { color: '#9ca3af' },
                splitLine: { lineStyle: { color: 'rgba(255,255,255,0.1)' } }
            },
            {
                type: 'value',
                name: '风险数',
                nameTextStyle: { color: '#9ca3af' },
                axisLabel: { color: '#9ca3af' },
                splitLine: { show: false }
            }
        ],
        series: [
            {
                name: '请求数',
                type: 'line',
                smooth: true,
                data: data.requests.slice(-12),
                lineStyle: { color: '#3b82f6', width: 3 },
                areaStyle: {
                    color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                        { offset: 0, color: 'rgba(59, 130, 246, 0.4)' },
                        { offset: 1, color: 'rgba(59, 130, 246, 0.05)' }
                    ])
                },
                symbol: 'circle',
                symbolSize: 6
            },
            {
                name: '风险数',
                type: 'bar',
                yAxisIndex: 1,
                data: data.risks.slice(-12),
                itemStyle: {
                    color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                        { offset: 0, color: '#ef4444' },
                        { offset: 1, color: '#dc2626' }
                    ]),
                    borderRadius: [4, 4, 0, 0]
                }
            }
        ]
    };

    trafficChart.setOption(option);
}

function drawRiskDistributionChart(data) {
    if (!riskDistributionChart || !data) return;

    const colors = ['#10b981', '#f59e0b', '#f97316', '#ef4444'];

    const option = {
        backgroundColor: 'transparent',
        title: {
            text: '风险等级分布',
            left: 'center',
            textStyle: { fontSize: 14, fontWeight: 'normal', color: '#9ca3af' }
        },
        tooltip: {
            trigger: 'item',
            backgroundColor: 'rgba(0,0,0,0.8)',
            textStyle: { color: '#fff' },
            formatter: function(params) {
                return `${params.name}: ${params.value} (${params.data.percent}%)`;
            }
        },
        legend: {
            orient: 'horizontal',
            bottom: 0,
            textStyle: { color: '#9ca3af' }
        },
        series: [
            {
                name: '风险分布',
                type: 'pie',
                radius: ['45%', '70%'],
                center: ['50%', '45%'],
                avoidLabelOverlap: false,
                itemStyle: {
                    borderRadius: 8,
                    borderColor: '#1f2937',
                    borderWidth: 2
                },
                label: {
                    show: true,
                    color: '#9ca3af',
                    formatter: '{b}: {d}%'
                },
                emphasis: {
                    label: { show: true, fontSize: 14, fontWeight: 'bold', color: '#fff' },
                    itemStyle: { shadowBlur: 10, shadowOffsetX: 0, shadowColor: 'rgba(0, 0, 0, 0.5)' }
                },
                labelLine: { show: true },
                data: data.map((item, index) => ({
                    ...item,
                    itemStyle: { color: colors[index] }
                }))
            }
        ]
    };

    riskDistributionChart.setOption(option);
}

function drawResponseTimeChart(data) {
    if (!responseTimeChart || !data) return;

    const option = {
        backgroundColor: 'transparent',
        title: {
            text: '响应时间分布',
            left: 'center',
            textStyle: { fontSize: 14, fontWeight: 'normal', color: '#9ca3af' }
        },
        tooltip: {
            trigger: 'axis',
            backgroundColor: 'rgba(0,0,0,0.8)',
            textStyle: { color: '#fff' },
            formatter: function(params) {
                let result = `${params[0].axisValue}<br/>`;
                params.forEach(param => {
                    result += `${param.seriesName}: ${param.value}ms<br/>`;
                });
                return result;
            }
        },
        legend: {
            data: ['平均响应时间', 'P95响应时间', 'P99响应时间'],
            bottom: 0,
            textStyle: { color: '#9ca3af' }
        },
        grid: {
            left: '3%',
            right: '4%',
            bottom: '15%',
            top: '15%',
            containLabel: true
        },
        xAxis: {
            type: 'category',
            data: data.labels,
            axisLabel: { color: '#9ca3af', fontSize: 10, rotate: 45 },
            axisLine: { lineStyle: { color: '#374151' } }
        },
        yAxis: {
            type: 'value',
            name: '响应时间(ms)',
            nameTextStyle: { color: '#9ca3af' },
            axisLabel: { color: '#9ca3af' },
            splitLine: { lineStyle: { color: 'rgba(255,255,255,0.1)' } }
        },
        series: [
            {
                name: '平均响应时间',
                type: 'line',
                smooth: true,
                data: data.avg,
                lineStyle: { color: '#10b981', width: 2 },
                symbol: 'circle',
                symbolSize: 5
            },
            {
                name: 'P95响应时间',
                type: 'line',
                smooth: true,
                data: data.p95,
                lineStyle: { color: '#f59e0b', width: 2, type: 'dashed' },
                symbol: 'circle',
                symbolSize: 5
            },
            {
                name: 'P99响应时间',
                type: 'line',
                smooth: true,
                data: data.p99,
                lineStyle: { color: '#ef4444', width: 2, type: 'dotted' },
                symbol: 'circle',
                symbolSize: 5
            }
        ]
    };

    responseTimeChart.setOption(option);
}

function drawRegionChart(data) {
    if (!regionChart || !data) return;

    const option = {
        backgroundColor: 'transparent',
        title: {
            text: '地域分布',
            left: 'center',
            textStyle: { fontSize: 14, fontWeight: 'normal', color: '#9ca3af' }
        },
        tooltip: {
            trigger: 'axis',
            backgroundColor: 'rgba(0,0,0,0.8)',
            textStyle: { color: '#fff' },
            axisPointer: { type: 'shadow' },
            formatter: function(params) {
                const param = params[0];
                return `${param.name}: ${param.value.toLocaleString()}`;
            }
        },
        grid: {
            left: '3%',
            right: '4%',
            bottom: '3%',
            top: '15%',
            containLabel: true
        },
        xAxis: {
            type: 'value',
            name: '请求数',
            nameTextStyle: { color: '#9ca3af' },
            axisLabel: { color: '#9ca3af' },
            splitLine: { lineStyle: { color: 'rgba(255,255,255,0.1)' } }
        },
        yAxis: {
            type: 'category',
            data: data.map(item => item.name).reverse(),
            axisLabel: { color: '#9ca3af' },
            axisLine: { lineStyle: { color: '#374151' } }
        },
        series: [
            {
                type: 'bar',
                data: data.map(item => item.value).reverse(),
                itemStyle: {
                    color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [
                        { offset: 0, color: '#3b82f6' },
                        { offset: 1, color: '#1d4ed8' }
                    ]),
                    borderRadius: [0, 4, 4, 0]
                },
                barWidth: '60%',
                emphasis: {
                    itemStyle: {
                        color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [
                            { offset: 0, color: '#60a5fa' },
                            { offset: 1, color: '#3b82f6' }
                        ])
                    }
                }
            }
        ]
    };

    regionChart.setOption(option);
}

function updateTopEndpoints(endpoints) {
    const tbody = document.getElementById('topEndpointsTable');
    if (!tbody || !endpoints) return;

    tbody.innerHTML = endpoints.map((ep, index) => `
        <tr>
            <td>${index + 1}</td>
            <td class="endpoint-name">${ep.endpoint}</td>
            <td>${ep.requests.toLocaleString()}</td>
            <td><span class="risk-badge ${getRiskBadgeClass(ep.risk)}">${ep.risk.toFixed(1)}</span></td>
            <td>${ep.avgTime}ms</td>
        </tr>
    `).join('');
}

function getRiskBadgeClass(risk) {
    if (risk >= 70) return 'risk-critical';
    if (risk >= 50) return 'risk-high';
    if (risk >= 30) return 'risk-medium';
    return 'risk-low';
}

function updateRecentAlerts(alerts) {
    const container = document.getElementById('recentAlerts');
    if (!container || !alerts) return;

    container.innerHTML = alerts.map(alert => `
        <div class="alert-item ${alert.severity}">
            <div class="alert-icon">${getSeverityIcon(alert.severity)}</div>
            <div class="alert-content">
                <div class="alert-header">
                    <span class="alert-id">${alert.id}</span>
                    <span class="alert-type">${alert.type}</span>
                    <span class="severity-badge ${alert.severity}">${getSeverityText(alert.severity)}</span>
                </div>
                <p class="alert-message">${alert.message}</p>
                <span class="alert-time">${alert.time}</span>
            </div>
        </div>
    `).join('');
}

function getSeverityIcon(severity) {
    const icons = {
        critical: '<i class="fas fa-exclamation-triangle"></i>',
        high: '<i class="fas fa-exclamation-circle"></i>',
        medium: '<i class="fas fa-info-circle"></i>',
        low: '<i class="fas fa-info"></i>'
    };
    return icons[severity] || icons.low;
}

function getSeverityText(severity) {
    const texts = {
        critical: '严重',
        high: '高',
        medium: '中',
        low: '低'
    };
    return texts[severity] || '低';
}

function startAutoRefresh() {
    if (refreshInterval) {
        clearInterval(refreshInterval);
    }
    refreshInterval = setInterval(loadDashboardData, 10000);
}

function stopAutoRefresh() {
    if (refreshInterval) {
        clearInterval(refreshInterval);
        refreshInterval = null;
    }
}

function exportDashboardData() {
    const data = {
        timestamp: new Date().toISOString(),
        timeRange: selectedTimeRange,
        summary: {
            totalRequests: document.getElementById('totalRequests')?.querySelector('.card-value').textContent,
            validRequests: document.getElementById('validRequests')?.querySelector('.card-value').textContent,
            invalidRequests: document.getElementById('invalidRequests')?.querySelector('.card-value').textContent,
            riskScore: document.getElementById('riskScore')?.querySelector('.card-value').textContent,
            anomalyRate: document.getElementById('anomalyRate')?.querySelector('.card-value').textContent
        }
    };

    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `dashboard_export_${Date.now()}.json`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
}

function handleCardClick(cardId) {
    console.log('Card clicked:', cardId);
}

function toggleFullscreen() {
    const container = document.getElementById('dashboardContainer');
    if (!container) return;

    if (document.fullscreenElement) {
        document.exitFullscreen();
    } else {
        container.requestFullscreen();
    }
}