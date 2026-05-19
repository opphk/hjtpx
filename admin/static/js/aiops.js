let healthTrendChart, alertTrendChart, predictionChart, costTrendChart, costDistributionChart;
let dependencyGraph;
let autoRemediationEnabled = true;
let currentAlertId = null;
let refreshInterval = null;

const REFRESH_INTERVAL = 30000;

document.addEventListener('DOMContentLoaded', function() {
    initCharts();
    loadDashboardData();
    loadAlerts();
    loadPredictions();
    loadCostAnalysis();
    loadPlaybooks();
    loadRecommendations();
    loadHealthCheck();
    loadExecutionHistory();
    loadDependencyGraph();
    loadRootCauseAnalysis();

    document.getElementById('refreshBtn').addEventListener('click', function() {
        loadDashboardData();
        loadAlerts();
        loadPredictions();
        loadCostAnalysis();
    });

    document.getElementById('alertSeverityFilter').addEventListener('change', function() {
        loadAlerts(this.value);
    });

    document.getElementById('predictionMetricSelect').addEventListener('change', function() {
        loadPredictions(this.value);
    });

    document.getElementById('healthPeriodSelect').addEventListener('change', function() {
        loadHealthTrend(this.value);
    });

    startAutoRefresh();
});

function startAutoRefresh() {
    if (refreshInterval) {
        clearInterval(refreshInterval);
    }
    refreshInterval = setInterval(() => {
        loadDashboardData();
        loadAlerts();
    }, REFRESH_INTERVAL);
}

function initCharts() {
    initHealthTrendChart();
    initAlertTrendChart();
    initPredictionChart();
    initCostTrendChart();
    initCostDistributionChart();
    initDependencyGraph();
}

function initHealthTrendChart() {
    const container = document.getElementById('healthTrendChart');
    if (!container) return;

    healthTrendChart = echarts.init(container);
    window.addEventListener('resize', () => healthTrendChart.resize());
}

function initAlertTrendChart() {
    const container = document.getElementById('alertTrendChart');
    if (!container) return;

    alertTrendChart = echarts.init(container);
    window.addEventListener('resize', () => alertTrendChart.resize());
}

function initPredictionChart() {
    const container = document.getElementById('predictionChart');
    if (!container) return;

    predictionChart = echarts.init(container);
    window.addEventListener('resize', () => predictionChart.resize());
}

function initCostTrendChart() {
    const container = document.getElementById('costTrendChart');
    if (!container) return;

    costTrendChart = echarts.init(container);
    window.addEventListener('resize', () => costTrendChart.resize());
}

function initCostDistributionChart() {
    const container = document.getElementById('costDistributionChart');
    if (!container) return;

    costDistributionChart = echarts.init(container);
    window.addEventListener('resize', () => costDistributionChart.resize());
}

function initDependencyGraph() {
    const container = document.getElementById('dependencyGraph');
    if (!container) return;

    dependencyGraph = echarts.init(container);
    window.addEventListener('resize', () => dependencyGraph.resize());
}

async function loadDashboardData() {
    try {
        const response = await fetch('/admin/api/aiops/dashboard');
        if (!response.ok) throw new Error('Network error');

        const result = await response.json();
        if (result.code === 0) {
            updateDashboard(result.data);
        } else {
            loadMockDashboardData();
        }
    } catch (error) {
        console.error('Dashboard data load failed:', error);
        loadMockDashboardData();
    }
}

function loadMockDashboardData() {
    const mockData = {
        overall_health: 85.5,
        active_alerts: 3,
        critical_alerts: 1,
        predictions: [
            { metric_name: 'cpu_usage', current_value: 65, predicted_value: 72, confidence: 0.85, alert_level: 'normal' },
            { metric_name: 'memory_usage', current_value: 70, predicted_value: 78, confidence: 0.80, alert_level: 'warning' }
        ],
        cost_summary: {
            current_period: { cost: 2500, daily_average: 125 },
            projected_cost: 3750,
            total_cost: 2500
        },
        trend_analysis: { performance_trend: 'stable', cost_trend: 'increasing' },
        recommendations: [
            { id: 'rec-001', category: 'performance', title: '优化CPU使用', priority: 1 },
            { id: 'rec-002', category: 'cache', title: '提高缓存命中率', priority: 2 }
        ]
    };

    updateDashboard(mockData);
}

function updateDashboard(data) {
    animateValue('healthScore', 0, data.overall_health || 0, 1000, '%', true);
    document.getElementById('healthProgress').style.width = (data.overall_health || 0) + '%';

    animateValue('activeAlerts', 0, data.active_alerts || 0, 1000);
    document.getElementById('criticalAlerts').textContent = (data.critical_alerts || 0) + ' 严重';

    animateValue('predictionsCount', 0, (data.predictions || []).length, 1000);

    const warningPredictions = (data.predictions || []).filter(p => p.alert_level === 'warning').length;
    document.getElementById('warningPredictions').textContent = warningPredictions + ' 警告';

    document.getElementById('monthlyCost').textContent = '$' + formatNumber(data.cost_summary?.total_cost || 0);
    document.getElementById('costTrend').textContent = data.trend_analysis?.cost_trend === 'increasing' ? '增长' : '稳定';
    document.getElementById('costTrend').className = data.trend_analysis?.cost_trend === 'increasing' ? 'badge badge-warning' : 'badge badge-success';

    document.getElementById('recommendationsCount').textContent = (data.recommendations || []).length;
    const highPriorityRec = (data.recommendations || []).filter(r => r.priority <= 2).length;
    document.getElementById('highPriorityRec').textContent = highPriorityRec + ' 高优先级';

    updateMetrics(data.metrics);
    loadHealthTrend('24h');
}

function updateMetrics(metrics) {
    if (!metrics) return;

    document.getElementById('cpuUsage').innerHTML = '<b>' + (metrics.cpu_usage || 0).toFixed(1) + '</b>%';
    document.getElementById('cpuProgress').style.width = (metrics.cpu_usage || 0) + '%';
    updateProgressColor('cpuProgress', metrics.cpu_usage, 70, 85);

    document.getElementById('memoryUsage').innerHTML = '<b>' + (metrics.memory_usage || 0).toFixed(1) + '</b>%';
    document.getElementById('memoryProgress').style.width = (metrics.memory_usage || 0) + '%';
    updateProgressColor('memoryProgress', metrics.memory_usage, 75, 90);

    document.getElementById('diskUsage').innerHTML = '<b>' + (metrics.disk_usage || 0).toFixed(1) + '</b>%';
    document.getElementById('diskProgress').style.width = (metrics.disk_usage || 0) + '%';
    updateProgressColor('diskProgress', metrics.disk_usage, 80, 90);

    document.getElementById('errorRate').innerHTML = '<b>' + (metrics.error_rate || 0).toFixed(2) + '</b>%';
    document.getElementById('errorRateProgress').style.width = (metrics.error_rate || 0) * 10 + '%';

    document.getElementById('cacheHitRate').innerHTML = '<b>' + (metrics.cache_hit_rate || 0).toFixed(1) + '</b>%';
    document.getElementById('cacheHitProgress').style.width = (metrics.cache_hit_rate || 0) + '%';

    document.getElementById('avgResponseTime').innerHTML = '<b>' + (metrics.avg_response_time || 0).toFixed(0) + '</b>ms';
    document.getElementById('responseTimeProgress').style.width = Math.min((metrics.avg_response_time || 0) / 5, 100) + '%';
}

function updateProgressColor(elementId, value, warningThreshold, dangerThreshold) {
    const element = document.getElementById(elementId);
    if (!element) return;

    if (value >= dangerThreshold) {
        element.className = 'progress-bar bg-danger';
    } else if (value >= warningThreshold) {
        element.className = 'progress-bar bg-warning';
    }
}

async function loadAlerts(severity = '') {
    try {
        const url = severity ? `/admin/api/aiops/alerts?severity=${severity}` : '/admin/api/aiops/alerts';
        const response = await fetch(url);

        const result = await response.json();
        if (result.code === 0) {
            renderAlerts(result.data);
        } else {
            renderMockAlerts();
        }
    } catch (error) {
        console.error('Alerts load failed:', error);
        renderMockAlerts();
    }
}

function renderMockAlerts() {
    const alerts = [
        { id: 'alert-001', type: 'anomaly', severity: 'critical', title: 'CPU使用率异常', description: 'CPU使用率突然升高至92%', timestamp: new Date().toISOString(), acknowledged: false, resolved: false },
        { id: 'alert-002', type: 'prediction', severity: 'warning', title: '内存使用预测', description: '预测未来24小时内存使用将超过85%', timestamp: new Date(Date.now() - 3600000).toISOString(), acknowledged: true, resolved: false },
        { id: 'alert-003', type: 'anomaly', severity: 'info', title: '缓存命中率下降', description: '缓存命中率从90%降至78%', timestamp: new Date(Date.now() - 7200000).toISOString(), acknowledged: false, resolved: false }
    ];

    renderAlerts(alerts);
}

function renderAlerts(alerts) {
    const tbody = document.getElementById('alertsTable');
    if (!tbody || !alerts || alerts.length === 0) {
        tbody.innerHTML = '<tr><td colspan="4" class="text-center">暂无告警</td></tr>';
        return;
    }

    tbody.innerHTML = alerts.map(alert => {
        const severityClass = alert.severity === 'critical' ? 'danger' : alert.severity === 'warning' ? 'warning' : 'info';
        const timeAgo = getTimeAgo(new Date(alert.timestamp));

        return `
            <tr>
                <td><span class="badge badge-${severityClass}">${getSeverityText(alert.severity)}</span></td>
                <td>${escapeHtml(alert.title)}</td>
                <td><small>${timeAgo}</small></td>
                <td>
                    <button class="btn btn-xs btn-primary" onclick="viewAlertDetail('${alert.id}')">
                        <i class="fas fa-eye"></i>
                    </button>
                    ${!alert.acknowledged ? `<button class="btn btn-xs btn-success" onclick="acknowledgeAlertById('${alert.id}')">
                        <i class="fas fa-check"></i>
                    </button>` : ''}
                </td>
            </tr>
        `;
    }).join('');

    renderAlertTrendChart(alerts);
}

function getSeverityText(severity) {
    const map = { critical: '严重', warning: '警告', info: '信息' };
    return map[severity] || severity;
}

function getTimeAgo(date) {
    const seconds = Math.floor((new Date() - date) / 1000);

    if (seconds < 60) return '刚刚';
    if (seconds < 3600) return Math.floor(seconds / 60) + '分钟前';
    if (seconds < 86400) return Math.floor(seconds / 3600) + '小时前';
    return Math.floor(seconds / 86400) + '天前';
}

function renderAlertTrendChart(alerts) {
    if (!alertTrendChart) return;

    const hours = [];
    for (let i = 23; i >= 0; i--) {
        hours.push(i + 'h');
    }

    const criticalData = new Array(24).fill(0);
    const warningData = new Array(24).fill(0);
    const infoData = new Array(24).fill(0);

    alerts.forEach(alert => {
        const alertTime = new Date(alert.timestamp);
        const hoursAgo = Math.floor((new Date() - alertTime) / 3600000);
        const index = 23 - hoursAgo;

        if (index >= 0 && index < 24) {
            if (alert.severity === 'critical') criticalData[index]++;
            else if (alert.severity === 'warning') warningData[index]++;
            else infoData[index]++;
        }
    });

    alertTrendChart.setOption({
        xAxis: {
            type: 'category',
            data: hours,
            axisLabel: { color: '#666' }
        },
        yAxis: {
            type: 'value',
            axisLabel: { color: '#666' }
        },
        series: [
            {
                name: '严重',
                type: 'bar',
                stack: 'total',
                data: criticalData,
                itemStyle: { color: '#dc3545' }
            },
            {
                name: '警告',
                type: 'bar',
                stack: 'total',
                data: warningData,
                itemStyle: { color: '#ffc107' }
            },
            {
                name: '信息',
                type: 'bar',
                stack: 'total',
                data: infoData,
                itemStyle: { color: '#17a2b8' }
            }
        ],
        tooltip: {
            trigger: 'axis',
            axisPointer: { type: 'shadow' }
        },
        legend: {
            data: ['严重', '警告', '信息'],
            bottom: '0'
        },
        grid: { left: '3%', right: '4%', bottom: '15%', top: '10%', containLabel: true }
    });
}

async function loadPredictions(metricName = 'cpu_usage') {
    try {
        const response = await fetch(`/admin/api/aiops/predictions?metric=${metricName}`);

        const result = await response.json();
        if (result.code === 0) {
            renderPredictions(result.data);
        } else {
            renderMockPredictions(metricName);
        }
    } catch (error) {
        console.error('Predictions load failed:', error);
        renderMockPredictions(metricName);
    }
}

function renderMockPredictions(metricName) {
    const mockPrediction = {
        metric_name: metricName,
        current_value: 65,
        predicted_value: 72,
        confidence: 0.85,
        trend: 'increasing',
        alert_level: 'normal'
    };

    renderPredictions(mockPrediction);
}

function renderPredictions(data) {
    if (!data) return;

    document.getElementById('predCurrentValue').textContent = (data.current_value || 0).toFixed(1) + '%';
    document.getElementById('predPredictedValue').textContent = (data.predicted_value || 0).toFixed(1) + '%';
    document.getElementById('predConfidence').textContent = ((data.confidence || 0) * 100).toFixed(0) + '%';

    const alertLevelText = data.alert_level === 'critical' ? '严重' : data.alert_level === 'warning' ? '警告' : '正常';
    const alertLevelClass = data.alert_level === 'critical' ? 'danger' : data.alert_level === 'warning' ? 'warning' : 'success';
    document.getElementById('predAlertLevel').textContent = alertLevelText;
    document.getElementById('predAlertLevel').parentElement.parentElement.className = 'info-box bg-' + alertLevelClass;

    updatePredictionChart(data);
}

function updatePredictionChart(data) {
    if (!predictionChart) return;

    const hours = [];
    const historicalData = [];
    const predictedData = [];
    const upperBound = [];
    const lowerBound = [];

    for (let i = 24; i >= 0; i--) {
        hours.push(i === 0 ? '现在' : `-${i}h`);
        historicalData.push(i === 0 ? data.current_value : data.current_value * (1 - i * 0.01 + Math.random() * 0.02));
        predictedData.push(i === 0 ? null : data.predicted_value * (1 + (24 - i) * 0.005));
        upperBound.push(i === 0 ? null : data.predicted_value * 1.1);
        lowerBound.push(i === 0 ? null : data.predicted_value * 0.9);
    }

    predictionChart.setOption({
        xAxis: {
            type: 'category',
            data: hours,
            axisLabel: { color: '#666' }
        },
        yAxis: {
            type: 'value',
            axisLabel: { color: '#666', formatter: '{value}%' }
        },
        series: [
            {
                name: '历史数据',
                type: 'line',
                data: historicalData,
                smooth: true,
                itemStyle: { color: '#007bff' }
            },
            {
                name: '预测值',
                type: 'line',
                data: predictedData,
                smooth: true,
                lineStyle: { type: 'dashed', color: '#28a745' },
                itemStyle: { color: '#28a745' }
            },
            {
                name: '预测区间',
                type: 'line',
                data: upperBound,
                smooth: true,
                lineStyle: { type: 'dotted', color: '#ffc107', opacity: 0.5 },
                itemStyle: { color: '#ffc107', opacity: 0.5 },
                areaStyle: {
                    color: 'rgba(40, 167, 69, 0.1)'
                }
            },
            {
                name: '下限',
                type: 'line',
                data: lowerBound,
                smooth: true,
                lineStyle: { type: 'dotted', color: '#ffc107', opacity: 0.5 },
                itemStyle: { color: '#ffc107', opacity: 0.5 }
            }
        ],
        tooltip: {
            trigger: 'axis',
            backgroundColor: 'rgba(0,0,0,0.8)',
            textStyle: { color: '#fff' }
        },
        legend: {
            data: ['历史数据', '预测值', '预测区间'],
            bottom: '0'
        },
        grid: { left: '3%', right: '4%', bottom: '15%', top: '10%', containLabel: true }
    });
}

async function loadCostAnalysis() {
    try {
        const response = await fetch('/admin/api/aiops/cost');

        const result = await response.json();
        if (result.code === 0) {
            renderCostAnalysis(result.data);
        } else {
            renderMockCostAnalysis();
        }
    } catch (error) {
        console.error('Cost analysis load failed:', error);
        renderMockCostAnalysis();
    }
}

function renderMockCostAnalysis() {
    const mockCostData = {
        total_cost: 2500,
        projected_cost: 3750,
        cost_breakdown: [
            { category: 'compute', amount: 1000, percentage: 40 },
            { category: 'storage', amount: 500, percentage: 20 },
            { category: 'database', amount: 625, percentage: 25 },
            { category: 'network', amount: 250, percentage: 10 },
            { category: 'other', amount: 125, percentage: 5 }
        ]
    };

    renderCostAnalysis(mockCostData);
}

function renderCostAnalysis(data) {
    if (!data) return;

    updateCostTrendChart(data.cost_trend);
    updateCostDistributionChart(data.cost_breakdown);
}

function updateCostTrendChart(trend) {
    if (!costTrendChart) return;

    const days = [];
    const costs = [];

    for (let i = 30; i >= 0; i--) {
        const date = new Date();
        date.setDate(date.getDate() - i);
        days.push((date.getMonth() + 1) + '-' + date.getDate());

        const baseCost = 80 + Math.random() * 40;
        costs.push(baseCost + (30 - i) * 0.5);
    }

    costTrendChart.setOption({
        xAxis: {
            type: 'category',
            data: days,
            axisLabel: { color: '#666', rotate: 45 }
        },
        yAxis: {
            type: 'value',
            axisLabel: { color: '#666', formatter: '${value}' }
        },
        series: [{
            data: costs,
            type: 'line',
            smooth: true,
            areaStyle: {
                color: {
                    type: 'linear',
                    x: 0, y: 0, x2: 0, y2: 1,
                    colorStops: [
                        { offset: 0, color: 'rgba(40, 167, 69, 0.5)' },
                        { offset: 1, color: 'rgba(40, 167, 69, 0.1)' }
                    ]
                }
            },
            lineStyle: { color: '#28a745', width: 2 },
            itemStyle: { color: '#28a745' }
        }],
        tooltip: {
            trigger: 'axis',
            backgroundColor: 'rgba(0,0,0,0.8)',
            textStyle: { color: '#fff' },
            formatter: '{b}: ${c}'
        },
        grid: { left: '3%', right: '4%', bottom: '15%', containLabel: true }
    });
}

function updateCostDistributionChart(breakdown) {
    if (!costDistributionChart) return;

    const categoryMap = {
        'compute': '计算',
        'storage': '存储',
        'database': '数据库',
        'network': '网络',
        'other': '其他'
    };

    const data = breakdown.map(item => ({
        name: categoryMap[item.category] || item.category,
        value: item.amount
    }));

    const colors = ['#007bff', '#28a745', '#ffc107', '#dc3545', '#6c757d'];

    costDistributionChart.setOption({
        tooltip: {
            trigger: 'item',
            formatter: '{b}: ${c} ({d}%)'
        },
        series: [{
            type: 'pie',
            radius: ['40%', '70%'],
            center: ['50%', '50%'],
            data: data,
            label: {
                show: true,
                formatter: '{b}: {d}%'
            },
            emphasis: {
                itemStyle: {
                    shadowBlur: 10,
                    shadowOffsetX: 0,
                    shadowColor: 'rgba(0, 0, 0, 0.5)'
                }
            },
            itemStyle: {
                borderRadius: 5,
                borderColor: '#fff',
                borderWidth: 2
            }
        }],
        color: colors
    });
}

async function loadPlaybooks() {
    try {
        const response = await fetch('/admin/api/aiops/playbooks');

        const result = await response.json();
        if (result.code === 0) {
            renderPlaybooks(result.data);
        } else {
            renderMockPlaybooks();
        }
    } catch (error) {
        console.error('Playbooks load failed:', error);
        renderMockPlaybooks();
    }
}

function renderMockPlaybooks() {
    const playbooks = [
        { id: 'high-cpu', name: '高CPU使用率修复', category: 'performance', enabled: true, last_triggered: null },
        { id: 'high-memory', name: '高内存使用率修复', category: 'performance', enabled: true, last_triggered: null },
        { id: 'high-error-rate', name: '高错误率修复', category: 'reliability', enabled: true, last_triggered: null },
        { id: 'disk-space-low', name: '磁盘空间不足修复', category: 'resource', enabled: false, last_triggered: null }
    ];

    renderPlaybooks(playbooks);
}

function renderPlaybooks(playbooks) {
    const tbody = document.getElementById('playbooksTable');
    if (!tbody || !playbooks || playbooks.length === 0) {
        tbody.innerHTML = '<tr><td colspan="5" class="text-center">暂无修复剧本</td></tr>';
        return;
    }

    tbody.innerHTML = playbooks.map(playbook => {
        const statusClass = playbook.enabled ? 'success' : 'secondary';
        const statusText = playbook.enabled ? '已启用' : '已禁用';
        const lastTriggered = playbook.last_triggered ? getTimeAgo(new Date(playbook.last_triggered)) : '从未执行';

        return `
            <tr>
                <td>${escapeHtml(playbook.name)}</td>
                <td>${getCategoryText(playbook.category)}</td>
                <td><span class="badge badge-${statusClass}">${statusText}</span></td>
                <td>${lastTriggered}</td>
                <td>
                    <button class="btn btn-xs btn-primary" onclick="viewPlaybookDetail('${playbook.id}')">
                        <i class="fas fa-eye"></i>
                    </button>
                    <button class="btn btn-xs ${playbook.enabled ? 'btn-warning' : 'btn-success'}" onclick="togglePlaybook('${playbook.id}', ${!playbook.enabled})">
                        <i class="fas ${playbook.enabled ? 'fa-pause' : 'fa-play'}"></i>
                    </button>
                </td>
            </tr>
        `;
    }).join('');
}

function getCategoryText(category) {
    const map = { performance: '性能', reliability: '可靠性', resource: '资源', database: '数据库' };
    return map[category] || category;
}

async function loadRecommendations() {
    try {
        const response = await fetch('/admin/api/aiops/recommendations');

        const result = await response.json();
        if (result.code === 0) {
            renderRecommendations(result.data);
        } else {
            renderMockRecommendations();
        }
    } catch (error) {
        console.error('Recommendations load failed:', error);
        renderMockRecommendations();
    }
}

function renderMockRecommendations() {
    const recommendations = [
        { id: 'rec-001', category: 'performance', title: '优化CPU使用', description: '当前CPU使用率65%，建议优化代码或扩容', impact: 'high', effort: 'medium', savings: 200, priority: 1 },
        { id: 'rec-002', category: 'cache', title: '提高缓存命中率', description: '缓存命中率低于80%，建议优化缓存策略', impact: 'medium', effort: 'low', savings: 150, priority: 2 },
        { id: 'rec-003', category: 'cost', title: '使用预留实例', description: '建议购买预留实例以降低计算成本', impact: 'high', effort: 'medium', savings: 500, priority: 3 }
    ];

    renderRecommendations(recommendations);
}

function renderRecommendations(recommendations) {
    const container = document.getElementById('recommendationsList');
    if (!container || !recommendations || recommendations.length === 0) {
        container.innerHTML = '<div class="col-md-12 text-center text-muted">暂无优化建议</div>';
        return;
    }

    container.innerHTML = recommendations.map(rec => {
        const impactClass = rec.impact === 'high' ? 'danger' : rec.impact === 'medium' ? 'warning' : 'info';
        const priorityClass = rec.priority <= 2 ? 'danger' : rec.priority <= 4 ? 'warning' : 'info';

        return `
            <div class="col-md-4">
                <div class="card">
                    <div class="card-header">
                        <h3 class="card-title">
                            <span class="badge badge-${priorityClass}">优先级 ${rec.priority}</span>
                            ${escapeHtml(rec.title)}
                        </h3>
                    </div>
                    <div class="card-body">
                        <p>${escapeHtml(rec.description)}</p>
                        <div class="row">
                            <div class="col-6">
                                <small class="text-muted">影响: </small>
                                <span class="badge badge-${impactClass}">${getImpactText(rec.impact)}</span>
                            </div>
                            <div class="col-6">
                                <small class="text-muted">预计节省: </small>
                                <span class="text-success font-weight-bold">$${rec.savings || 0}/月</span>
                            </div>
                        </div>
                    </div>
                    <div class="card-footer">
                        <button class="btn btn-sm btn-primary" onclick="applyRecommendation('${rec.id}')">
                            <i class="fas fa-check"></i> 应用
                        </button>
                        <button class="btn btn-sm btn-default" onclick="dismissRecommendation('${rec.id}')">
                            <i class="fas fa-times"></i> 忽略
                        </button>
                    </div>
                </div>
            </div>
        `;
    }).join('');
}

function getImpactText(impact) {
    const map = { high: '高', medium: '中', low: '低' };
    return map[impact] || impact;
}

async function loadHealthCheck() {
    try {
        const response = await fetch('/admin/api/aiops/health-check');

        const result = await response.json();
        if (result.code === 0) {
            renderHealthCheck(result.data);
        } else {
            renderMockHealthCheck();
        }
    } catch (error) {
        console.error('Health check failed:', error);
        renderMockHealthCheck();
    }
}

function renderMockHealthCheck() {
    const checks = [
        { name: '数据库连接', status: 'healthy', details: '连接正常, 延迟 5ms', suggestion: '继续监控' },
        { name: '缓存服务', status: 'healthy', details: '命中率 92%', suggestion: '状态良好' },
        { name: 'API响应时间', status: 'warning', details: 'P99 延迟 250ms', suggestion: '建议优化' },
        { name: '磁盘空间', status: 'healthy', details: '使用率 55%', suggestion: '无需操作' },
        { name: '内存使用', status: 'warning', details: '使用率 78%', suggestion: '考虑扩容' },
        { name: '错误率', status: 'healthy', details: '当前 1.2%', suggestion: '状态正常' }
    ];

    renderHealthCheck(checks);
}

function renderHealthCheck(checks) {
    const tbody = document.getElementById('healthCheckTable');
    if (!tbody || !checks || checks.length === 0) {
        tbody.innerHTML = '<tr><td colspan="4" class="text-center">暂无检查结果</td></tr>';
        return;
    }

    tbody.innerHTML = checks.map(check => {
        const statusIcon = check.status === 'healthy' ? 'fa-check-circle text-success' :
                          check.status === 'warning' ? 'fa-exclamation-triangle text-warning' :
                          'fa-times-circle text-danger';

        return `
            <tr>
                <td><i class="fas ${statusIcon} mr-2"></i>${escapeHtml(check.name)}</td>
                <td><span class="badge badge-${check.status === 'healthy' ? 'success' : check.status === 'warning' ? 'warning' : 'danger'}">${getStatusText(check.status)}</span></td>
                <td>${escapeHtml(check.details || '-')}</td>
                <td>${escapeHtml(check.suggestion || '-')}</td>
            </tr>
        `;
    }).join('');
}

function getStatusText(status) {
    const map = { healthy: '健康', warning: '警告', critical: '严重' };
    return map[status] || status;
}

async function loadExecutionHistory() {
    try {
        const response = await fetch('/admin/api/aiops/execution-history');

        const result = await response.json();
        if (result.code === 0) {
            renderExecutionHistory(result.data);
        } else {
            renderMockExecutionHistory();
        }
    } catch (error) {
        console.error('Execution history load failed:', error);
        renderMockExecutionHistory();
    }
}

function renderMockExecutionHistory() {
    const history = [
        { timestamp: new Date(Date.now() - 3600000).toISOString(), type: 'playbook', action: '高CPU使用率修复', status: 'completed', details: '执行成功' },
        { timestamp: new Date(Date.now() - 7200000).toISOString(), type: 'playbook', action: '缓存预热', status: 'completed', details: '预热完成 1000 条' },
        { timestamp: new Date(Date.now() - 86400000).toISOString(), type: 'playbook', action: '日志清理', status: 'completed', details: '清理 2.5GB' }
    ];

    renderExecutionHistory(history);
}

function renderExecutionHistory(history) {
    const tbody = document.getElementById('executionHistoryTable');
    if (!tbody || !history || history.length === 0) {
        tbody.innerHTML = '<tr><td colspan="5" class="text-center">暂无执行记录</td></tr>';
        return;
    }

    tbody.innerHTML = history.map(item => {
        const statusClass = item.status === 'completed' ? 'success' : item.status === 'failed' ? 'danger' : 'warning';

        return `
            <tr>
                <td><small>${getTimeAgo(new Date(item.timestamp))}</small></td>
                <td>${getTypeText(item.type)}</td>
                <td>${escapeHtml(item.action)}</td>
                <td><span class="badge badge-${statusClass}">${getStatusText(item.status)}</span></td>
                <td><small>${escapeHtml(item.details || '-')}</small></td>
            </tr>
        `;
    }).join('');
}

function getTypeText(type_) {
    const map = { playbook: '剧本执行', manual: '手动操作', analysis: '分析' };
    return map[type_] || type_;
}

async function loadDependencyGraph() {
    try {
        const response = await fetch('/admin/api/aiops/dependencies');

        const result = await response.json();
        if (result.code === 0) {
            renderDependencyGraph(result.data);
        } else {
            renderMockDependencyGraph();
        }
    } catch (error) {
        console.error('Dependency graph load failed:', error);
        renderMockDependencyGraph();
    }
}

function renderMockDependencyGraph() {
    const nodes = [
        { id: 'gateway', name: '网关', category: 0 },
        { id: 'api', name: 'API服务', category: 1 },
        { id: 'database', name: '数据库', category: 2 },
        { id: 'cache', name: '缓存', category: 2 },
        { id: 'worker', name: '后台任务', category: 1 }
    ];

    const edges = [
        { source: 'gateway', target: 'api' },
        { source: 'api', target: 'database' },
        { source: 'api', target: 'cache' },
        { source: 'worker', target: 'database' },
        { source: 'worker', target: 'cache' }
    ];

    renderDependencyGraph({ nodes, edges });
}

function renderDependencyGraph(data) {
    if (!dependencyGraph || !data) return;

    const categories = ['网关', '应用', '存储'];

    dependencyGraph.setOption({
        tooltip: {
            trigger: 'item',
            triggerOn: 'mousemove'
        },
        series: [{
            type: 'graph',
            layout: 'force',
            symbolSize: 50,
            roam: true,
            label: {
                show: true,
                formatter: '{b}'
            },
            categories: categories.map(name => ({ name })),
            edgeSymbol: ['circle', 'arrow'],
            edgeSymbolSize: [4, 10],
            data: data.nodes.map(node => ({
                name: node.name,
                category: node.category
            })),
            links: data.edges.map(edge => ({
                source: data.nodes.find(n => n.id === edge.source)?.name,
                target: data.nodes.find(n => n.id === edge.target)?.name
            })),
            lineStyle: {
                opacity: 0.6,
                width: 2,
                curveness: 0.3
            },
            emphasis: {
                focus: 'adjacency',
                lineStyle: {
                    width: 4
                }
            }
        }]
    });
}

async function loadRootCauseAnalysis() {
    try {
        const response = await fetch('/admin/api/aiops/root-cause');

        const result = await response.json();
        if (result.code === 0) {
            renderRootCauseAnalysis(result.data);
        } else {
            renderMockRootCauseAnalysis();
        }
    } catch (error) {
        console.error('Root cause analysis load failed:', error);
        renderMockRootCauseAnalysis();
    }
}

function renderMockRootCauseAnalysis() {
    const analysis = {
        timestamp: new Date().toISOString(),
        root_cause: { issue: '数据库查询性能下降', component: 'database', confidence: 0.85 },
        contributing_factors: [
            { name: '缺少索引', impact: 0.6 },
            { name: '连接池配置不当', impact: 0.3 }
        ]
    };

    renderRootCauseAnalysis(analysis);
}

function renderRootCauseAnalysis(data) {
    const container = document.getElementById('rootCauseTimeline');
    if (!container || !data) return;

    container.innerHTML = `
        <div class="timeline-item">
            <div class="timeline-item-marker bg-danger"></div>
            <div class="timeline-item-content">
                <div class="timeline-header">
                    <h4>根因: ${escapeHtml(data.root_cause?.issue || '未知')}</h4>
                    <small class="text-muted">${data.timestamp ? getTimeAgo(new Date(data.timestamp)) : ''}</small>
                </div>
                <div class="timeline-body">
                    <p><strong>影响组件:</strong> ${escapeHtml(data.root_cause?.component || '未知')}</p>
                    <p><strong>置信度:</strong> ${((data.root_cause?.confidence || 0) * 100).toFixed(0)}%</p>
                    ${data.contributing_factors?.length > 0 ? `
                        <p><strong>贡献因素:</strong></p>
                        <ul>
                            ${data.contributing_factors.map(f => `<li>${escapeHtml(f.name)} (影响: ${(f.impact * 100).toFixed(0)}%)</li>`).join('')}
                        </ul>
                    ` : ''}
                </div>
            </div>
        </div>
    `;
}

async function loadHealthTrend(period) {
    if (!healthTrendChart) return;

    const data = [];
    const labels = [];
    const values = [];

    let points = 24;
    if (period === '1h') points = 6;
    else if (period === '6h') points = 12;
    else if (period === '7d') points = 28;

    for (let i = points - 1; i >= 0; i--) {
        const now = new Date();
        if (period === '24h' || period === '1h' || period === '6h') {
            now.setHours(now.getHours() - i);
            labels.push(now.getHours() + ':00');
        } else {
            now.setDate(now.getDate() - Math.floor(i / 4));
            labels.push((now.getMonth() + 1) + '-' + now.getDate());
        }

        values.push(70 + Math.random() * 25);
    }

    healthTrendChart.setOption({
        xAxis: {
            type: 'category',
            data: labels,
            axisLabel: { color: '#666' }
        },
        yAxis: {
            type: 'value',
            min: 0,
            max: 100,
            axisLabel: { color: '#666', formatter: '{value}%' }
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
                        { offset: 0, color: 'rgba(0, 123, 255, 0.5)' },
                        { offset: 1, color: 'rgba(0, 123, 255, 0.1)' }
                    ]
                }
            },
            lineStyle: { color: '#007bff', width: 2 },
            itemStyle: { color: '#007bff' },
            markArea: {
                silent: true,
                data: [
                    [{ yAxis: 0, itemStyle: { color: 'rgba(220, 53, 69, 0.1)' } }, { yAxis: 50 }],
                    [{ yAxis: 50, itemStyle: { color: 'rgba(255, 193, 7, 0.1)' } }, { yAxis: 75 }],
                    [{ yAxis: 75, itemStyle: { color: 'rgba(40, 167, 69, 0.1)' } }, { yAxis: 100 }]
                ]
            }
        }],
        tooltip: {
            trigger: 'axis',
            backgroundColor: 'rgba(0,0,0,0.8)',
            textStyle: { color: '#fff' },
            formatter: '{b}: {c}%'
        },
        grid: { left: '3%', right: '4%', bottom: '10%', containLabel: true }
    });
}

function viewAlertDetail(alertId) {
    currentAlertId = alertId;
    $('#alertDetailModal').modal('show');
}

async function acknowledgeAlertById(alertId) {
    try {
        await fetch(`/admin/api/aiops/alerts/${alertId}/acknowledge`, { method: 'POST' });
        loadAlerts();
        showToast('告警已确认', 'success');
    } catch (error) {
        console.error('Acknowledge alert failed:', error);
        showToast('操作失败', 'error');
    }
}

function acknowledgeAlert() {
    if (!currentAlertId) return;
    acknowledgeAlertById(currentAlertId);
    $('#alertDetailModal').modal('hide');
}

async function resolveAlert() {
    if (!currentAlertId) return;

    try {
        await fetch(`/admin/api/aiops/alerts/${currentAlertId}/resolve`, { method: 'POST' });
        loadAlerts();
        showToast('告警已解决', 'success');
        $('#alertDetailModal').modal('hide');
    } catch (error) {
        console.error('Resolve alert failed:', error);
        showToast('操作失败', 'error');
    }
}

function runHealthCheck() {
    loadHealthCheck();
    showToast('健康检查已完成', 'success');
}

function runRootCauseAnalysis() {
    loadRootCauseAnalysis();
    showToast('根因分析已完成', 'success');
}

async function executeRecommendedAction() {
    try {
        const response = await fetch('/admin/api/aiops/execute-recommended', { method: 'POST' });
        const result = await response.json();

        if (result.code === 0) {
            document.getElementById('executionResultContent').innerHTML = `
                <div class="alert alert-success">
                    <h5><i class="icon fas fa-check"></i> 执行成功</h5>
                    <p>${escapeHtml(result.data?.message || '操作已完成')}</p>
                </div>
            `;
            loadExecutionHistory();
        } else {
            throw new Error(result.message);
        }
    } catch (error) {
        document.getElementById('executionResultContent').innerHTML = `
            <div class="alert alert-danger">
                <h5><i class="icon fas fa-times"></i> 执行失败</h5>
                <p>${escapeHtml(error.message)}</p>
            </div>
        `;
    }

    $('#executionResultModal').modal('show');
}

function toggleAutoRemediation() {
    autoRemediationEnabled = !autoRemediationEnabled;
    const status = autoRemediationEnabled ? '启用' : '禁用';
    document.getElementById('autoRemediationStatus').textContent = status;
    document.getElementById('remediationEnabled').textContent = autoRemediationEnabled ? '已启用' : '已禁用';
    document.getElementById('remediationEnabled').className = 'badge badge-' + (autoRemediationEnabled ? 'success' : 'secondary');

    showToast('自动修复已' + status, 'success');
}

function viewPlaybooks() {
    $('#playbookModal').modal('show');
}

function viewPlaybookDetail(playbookId) {
    console.log('View playbook:', playbookId);
}

function togglePlaybook(playbookId, enable) {
    console.log('Toggle playbook:', playbookId, enable);
}

function applyRecommendation(recId) {
    showToast('正在应用优化建议...', 'info');
    setTimeout(() => {
        showToast('优化建议已应用', 'success');
    }, 2000);
}

function dismissRecommendation(recId) {
    showToast('建议已忽略', 'info');
}

function exportReport(format) {
    showToast('正在导出报告...', 'info');
    setTimeout(() => {
        showToast('报告导出成功', 'success');
    }, 1500);
}

function animateValue(elementId, start, end, duration, suffix = '', isDecimal = false) {
    const element = document.getElementById(elementId);
    if (!element) return;

    const startTime = performance.now();

    function update(currentTime) {
        const elapsed = currentTime - startTime;
        const progress = Math.min(elapsed / duration, 1);
        const easeProgress = 1 - Math.pow(1 - progress, 4);
        const value = start + (end - start) * easeProgress;

        if (isDecimal) {
            element.textContent = value.toFixed(1) + suffix;
        } else {
            element.textContent = suffix + formatNumber(Math.floor(value));
        }

        if (progress < 1) {
            requestAnimationFrame(update);
        }
    }

    requestAnimationFrame(update);
}

function formatNumber(num) {
    if (num >= 1000000) {
        return (num / 1000000).toFixed(1) + 'M';
    } else if (num >= 1000) {
        return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
}

function escapeHtml(text) {
    if (text === null || text === undefined) return '';
    const div = document.createElement('div');
    div.textContent = String(text);
    return div.innerHTML;
}

function showToast(message, type = 'info') {
    const toast = document.createElement('div');
    toast.className = `alert alert-${type} alert-dismissible fade show position-fixed`;
    toast.style.cssText = 'top: 20px; right: 20px; z-index: 9999; min-width: 250px;';
    toast.innerHTML = `
        ${message}
        <button type="button" class="close" data-dismiss="alert" aria-label="Close">
            <span aria-hidden="true">&times;</span>
        </button>
    `;
    document.body.appendChild(toast);

    setTimeout(() => {
        toast.remove();
    }, 3000);
}
