let userGrowthChart, requestTrendChart, requestTypeChart, appDistributionChart, errorRateChart, geoDistributionChart;
let currentChartType = 'line';
let refreshTimer = null;
let isAutoRefresh = false;
let advancedStatsTimer = null;
const AUTO_REFRESH_INTERVAL = 60000;
const ADVANCED_STATS_INTERVAL = 30000;

let predictionChart = null;
let anomalyChart = null;
let breakdownChart = null;

document.addEventListener('DOMContentLoaded', () => {
    initAllCharts();
    setupEventListeners();
    loadStatsData();
    initAdvancedStats();
    setupAdvancedStatsRefresh();
});

function setupEventListeners() {
    const refreshBtn = document.getElementById('refreshBtn');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', loadStatsData);
    }

    const applyFilterBtn = document.getElementById('applyFilterBtn');
    if (applyFilterBtn) {
        applyFilterBtn.addEventListener('click', loadStatsData);
    }

    const dateRange = document.getElementById('dateRange');
    if (dateRange) {
        dateRange.addEventListener('change', loadStatsData);
    }

    const comparePeriod = document.getElementById('comparePeriod');
    if (comparePeriod) {
        comparePeriod.addEventListener('change', loadStatsData);
    }

    const chartTypeButtons = document.querySelectorAll('[data-chart-type]');
    chartTypeButtons.forEach(btn => {
        btn.addEventListener('click', (e) => {
            chartTypeButtons.forEach(b => b.classList.remove('active'));
            e.target.classList.add('active');
            const type = e.target.dataset.chartType || e.target.closest('button').dataset.chartType;
            if (type) {
                switchChartType(type);
            }
        });
    });

    const exportBtn = document.getElementById('exportStatsBtn');
    if (exportBtn) {
        exportBtn.addEventListener('click', exportStatsReport);
    }

    const customReportBtn = document.getElementById('customReportBtn');
    if (customReportBtn) {
        customReportBtn.addEventListener('click', generateCustomReport);
    }

    const predictionBtn = document.getElementById('predictionBtn');
    if (predictionBtn) {
        predictionBtn.addEventListener('click', loadPredictionData);
    }

    const anomalyBtn = document.getElementById('anomalyBtn');
    if (anomalyBtn) {
        anomalyBtn.addEventListener('click', loadAnomalyData);
    }

    setupAutoRefresh();
}

function setupAutoRefresh() {
    const autoRefreshBtn = document.createElement('button');
    autoRefreshBtn.id = 'autoRefreshBtn';
    autoRefreshBtn.className = 'btn btn-outline-secondary btn-sm ms-2';
    autoRefreshBtn.innerHTML = '<i class="fas fa-sync me-1"></i>自动刷新';
    
    const refreshBtn = document.getElementById('refreshBtn');
    if (refreshBtn && refreshBtn.parentElement) {
        refreshBtn.parentElement.appendChild(autoRefreshBtn);
        
        autoRefreshBtn.addEventListener('click', () => {
            isAutoRefresh = !isAutoRefresh;
            if (isAutoRefresh) {
                autoRefreshBtn.classList.add('btn-success');
                autoRefreshBtn.classList.remove('btn-outline-secondary');
                refreshTimer = setInterval(loadStatsData, AUTO_REFRESH_INTERVAL);
                showToast('自动刷新已开启（每分钟）', 'info');
            } else {
                autoRefreshBtn.classList.remove('btn-success');
                autoRefreshBtn.classList.add('btn-outline-secondary');
                if (refreshTimer) {
                    clearInterval(refreshTimer);
                    refreshTimer = null;
                }
                showToast('自动刷新已关闭', 'info');
            }
        });
    }
}

function switchChartType(type) {
    currentChartType = type;
    if (requestTrendChart) {
        requestTrendChart.config.type = type;
        requestTrendChart.update();
    }
}

function loadStatsData() {
    animateValue('totalRequests', 0, 82560, 1500);
    document.getElementById('successRate').textContent = '92.5%';
    document.getElementById('avgResponse').textContent = '45ms';
    animateValue('activeUsers', 0, 12580, 1500);
    
    document.getElementById('requestsProgress').style.width = '75%';
    document.getElementById('successProgress').style.width = '92.5%';
    document.getElementById('responseProgress').style.width = '45%';
    document.getElementById('usersProgress').style.width = '60%';
    
    const requestsChange = 12.5;
    const successChange = 2.3;
    const responseChange = -8.2;
    const usersChange = 15.8;
    
    document.getElementById('requestsChange').innerHTML = `<span class="trend-up"><i class="fas fa-arrow-up"></i> +${requestsChange}%</span>`;
    document.getElementById('successChange').innerHTML = `<span class="trend-up"><i class="fas fa-arrow-up"></i> +${successChange}%</span>`;
    document.getElementById('responseChange').innerHTML = `<span class="trend-down"><i class="fas fa-arrow-down"></i> ${responseChange}%</span>`;
    document.getElementById('usersChange').innerHTML = `<span class="trend-up"><i class="fas fa-arrow-up"></i> +${usersChange}%</span>`;
    
    updateTrendBadge('requestsTrendBadge', requestsChange);
    updateTrendBadge('successTrendBadge', successChange);
    updateTrendBadge('responseTrendBadge', responseChange, true);
    updateTrendBadge('usersTrendBadge', usersChange);
    
    loadAdditionalStats();
    loadStatsTable();
}

function loadAdditionalStats() {
    animateValue('totalBlockRate', 0, 5.2, 1000, '%');
    animateValue('peakQPS', 0, 1523, 1200);
    animateValue('avgRiskScore', 0, 23.5, 1000);
    animateValue('totalRevenue', 0, 45680, 1500, '¥');
    
    document.getElementById('blockRateProgress').style.width = '5.2%';
    document.getElementById('qpsProgress').style.width = '76%';
    document.getElementById('riskProgress').style.width = '23.5%';
    document.getElementById('revenueProgress').style.width = '68%';
    
    updateTrendBadge('revenueTrendBadge', 18.5);
}

function updateTrendBadge(elementId, change, isInverse = false) {
    const el = document.getElementById(elementId);
    if (!el) return;
    
    const isPositive = change >= 0;
    const displayValue = Math.abs(change).toFixed(1);
    
    if (isPositive === !isInverse) {
        el.innerHTML = `<i class="fas fa-arrow-up"></i> ${displayValue}%`;
        el.className = 'badge badge-success';
    } else {
        el.innerHTML = `<i class="fas fa-arrow-down"></i> ${displayValue}%`;
        el.className = 'badge badge-danger';
    }
}

function animateValue(elementId, start, end, duration, prefix = '') {
    const element = document.getElementById(elementId);
    if (!element) return;
    
    const startTime = performance.now();
    
    function update(currentTime) {
        const elapsed = currentTime - startTime;
        const progress = Math.min(elapsed / duration, 1);
        const easeProgress = 1 - Math.pow(1 - progress, 4);
        const value = start + (end - start) * easeProgress;
        
        if (prefix === '%') {
            element.textContent = value.toFixed(1) + '%';
        } else if (prefix === '¥') {
            element.textContent = '¥' + Math.floor(value).toLocaleString();
        } else {
            element.textContent = formatNumber(Math.floor(value));
        }
        
        if (progress < 1) {
            requestAnimationFrame(update);
        }
    }
    
    requestAnimationFrame(update);
}

function getMockStatsData(range) {
    const labels = generateLabels(range);
    const dataCount = labels.length;

    return {
        summary: {
            totalRequests: 8234567,
            avgResponseTime: 125,
            successRate: 98.5,
            activeUsers: 12456
        },
        changes: {
            requests: 23.5,
            responseTime: -12.3,
            successRate: 2.1,
            activeUsers: 15.8
        },
        requestTrend: {
            labels: labels,
            data: generateRandomData(dataCount, 5000, 20000)
        },
        requestType: {
            labels: ['滑块验证', '点选验证', '旋转验证', '拼图验证', '文字识别'],
            data: [45, 25, 15, 10, 5]
        },
        userGrowth: {
            labels: labels,
            data: generateGrowthData(dataCount, 8000, 15000)
        },
        appDistribution: {
            labels: ['Web应用', '移动应用', '桌面应用', 'API服务', '其他'],
            data: [42, 28, 15, 10, 5]
        },
        errorRate: {
            labels: labels,
            data: generateRandomData(dataCount, 0.5, 3.5)
        },
        geoDistribution: {
            labels: ['北京', '上海', '广东', '浙江', '江苏', '四川', '其他'],
            data: [25, 20, 18, 12, 10, 8, 7]
        }
    };
}

function generateLabels(range) {
    const count = range === '7d' ? 7 : range === '30d' ? 30 : range === '90d' ? 12 : 12;
    const labels = [];

    if (range === '7d') {
        for (let i = 6; i >= 0; i--) {
            const date = new Date(Date.now() - i * 24 * 60 * 60 * 1000);
            labels.push(`${date.getMonth() + 1}/${date.getDate()}`);
        }
    } else if (range === '30d') {
        for (let i = 29; i >= 0; i--) {
            const date = new Date(Date.now() - i * 24 * 60 * 60 * 1000);
            labels.push(`${date.getMonth() + 1}/${date.getDate()}`);
        }
    } else if (range === '90d') {
        for (let i = 0; i < 12; i++) {
            labels.push(`第${i + 1}周`);
        }
    } else {
        for (let i = 11; i >= 0; i--) {
            const date = new Date(Date.now() - i * 30 * 24 * 60 * 60 * 1000);
            labels.push(`${date.getFullYear()}/${date.getMonth() + 1}`);
        }
    }

    return labels;
}

function generateRandomData(count, min, max) {
    return Array.from({ length: count }, () => Math.floor(Math.random() * (max - min) + min));
}

function generateGrowthData(count, start, end) {
    const step = (end - start) / count;
    return Array.from({ length: count }, (_, i) => Math.floor(start + step * i + Math.random() * 100));
}

function updateAllStats(data) {
    updateSummaryCards(data.summary, data.changes);
    updateCharts(data);
}

function updateSummaryCards(summary, changes) {
    const totalRequestsEl = document.getElementById('statTotalRequests');
    const avgResponseEl = document.getElementById('statAvgResponse');
    const successRateEl = document.getElementById('statSuccessRate');
    const activeUsersEl = document.getElementById('statActiveUsers');

    if (totalRequestsEl) totalRequestsEl.textContent = formatLargeNumber(summary.totalRequests);
    if (avgResponseEl) avgResponseEl.textContent = `${summary.avgResponseTime}ms`;
    if (successRateEl) successRateEl.textContent = `${summary.successRate.toFixed(1)}%`;
    if (activeUsersEl) activeUsersEl.textContent = formatLargeNumber(summary.activeUsers);

    updateChangeIndicator('statRequestsChange', changes.requests);
    updateChangeIndicator('statResponseChange', changes.responseTime, true);
    updateChangeIndicator('statSuccessChange', changes.successRate);
    updateChangeIndicator('statUsersChange', changes.activeUsers);
}

function updateChangeIndicator(elementId, value, isInverse = false) {
    const el = document.getElementById(elementId);
    if (!el) return;

    const isPositive = value >= 0;
    const isGood = isInverse ? !isPositive : isPositive;
    const displayValue = Math.abs(value).toFixed(1);

    el.className = isGood ? 'text-success' : 'text-danger';
    el.innerHTML = `<i class="fas fa-arrow-${isPositive ? 'up' : 'down'} me-1"></i>${isPositive ? '+' : '-'}${displayValue}%`;
}

function formatLargeNumber(num) {
    if (num >= 1000000) {
        return (num / 1000000).toFixed(1) + 'M';
    } else if (num >= 1000) {
        return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
}

function updateCharts(data) {
    updateRequestTrendChart(data.requestTrend);
    updateRequestTypeChart(data.requestType);
    updateUserGrowthChart(data.userGrowth);
    updateAppDistributionChart(data.appDistribution);
    updateErrorRateChart(data.errorRate);
    updateGeoDistributionChart(data.geoDistribution);
    updateStatisticsSummary(data);
}

function updateStatisticsSummary(data) {
    const summaryEl = document.getElementById('statisticsSummary');
    if (!summaryEl) return;
    
    const avgRequests = data.requestTrend.data.reduce((a, b) => a + b, 0) / data.requestTrend.data.length;
    const peakRequests = Math.max(...data.requestTrend.data);
    const totalSuccessRate = data.requestType.data.reduce((acc, rate, idx) => {
        return acc + (rate * data.requestType.data[idx] / 100);
    }, 0);
    
    summaryEl.innerHTML = `
        <div class="row">
            <div class="col-md-4">
                <div class="text-center">
                    <h5>平均请求量</h5>
                    <p class="text-primary">${formatLargeNumber(Math.round(avgRequests))}</p>
                </div>
            </div>
            <div class="col-md-4">
                <div class="text-center">
                    <h5>峰值请求量</h5>
                    <p class="text-success">${formatLargeNumber(peakRequests)}</p>
                </div>
            </div>
            <div class="col-md-4">
                <div class="text-center">
                    <h5>总体成功率</h5>
                    <p class="text-info">${totalSuccessRate.toFixed(1)}%</p>
                </div>
            </div>
        </div>
    `;
}

function aggregateDataByPeriod(data, period) {
    if (period === 'hour') {
        return aggregateHourly(data);
    } else if (period === 'day') {
        return aggregateDaily(data);
    } else if (period === 'week') {
        return aggregateWeekly(data);
    } else if (period === 'month') {
        return aggregateMonthly(data);
    }
    return data;
}

function aggregateHourly(data) {
    return data.map(item => ({
        time: item.time,
        requests: item.requests,
        avgLatency: item.avgLatency || 0,
        successRate: item.successRate || 0
    }));
}

function aggregateDaily(data) {
    const grouped = {};
    data.forEach(item => {
        const date = item.time.split(' ')[0];
        if (!grouped[date]) {
            grouped[date] = { requests: 0, latencySum: 0, count: 0 };
        }
        grouped[date].requests += item.requests;
        if (item.avgLatency) {
            grouped[date].latencySum += item.avgLatency;
            grouped[date].count++;
        }
    });
    
    return Object.entries(grouped).map(([date, values]) => ({
        time: date,
        requests: values.requests,
        avgLatency: values.count > 0 ? Math.round(values.latencySum / values.count) : 0
    }));
}

function aggregateWeekly(data) {
    const grouped = {};
    data.forEach(item => {
        const date = new Date(item.time);
        const weekStart = getWeekStart(date);
        const weekKey = weekStart.toISOString().split('T')[0];
        
        if (!grouped[weekKey]) {
            grouped[weekKey] = { requests: 0, latencySum: 0, count: 0 };
        }
        grouped[weekKey].requests += item.requests;
        if (item.avgLatency) {
            grouped[weekKey].latencySum += item.avgLatency;
            grouped[weekKey].count++;
        }
    });
    
    return Object.entries(grouped).map(([week, values]) => ({
        time: week,
        requests: values.requests,
        avgLatency: values.count > 0 ? Math.round(values.latencySum / values.count) : 0
    }));
}

function aggregateMonthly(data) {
    const grouped = {};
    data.forEach(item => {
        const month = item.time.substring(0, 7);
        if (!grouped[month]) {
            grouped[month] = { requests: 0, latencySum: 0, count: 0 };
        }
        grouped[month].requests += item.requests;
        if (item.avgLatency) {
            grouped[month].latencySum += item.avgLatency;
            grouped[month].count++;
        }
    });
    
    return Object.entries(grouped).map(([month, values]) => ({
        time: month,
        requests: values.requests,
        avgLatency: values.count > 0 ? Math.round(values.latencySum / values.count) : 0
    }));
}

function getWeekStart(date) {
    const d = new Date(date);
    const day = d.getDay();
    const diff = d.getDate() - day + (day === 0 ? -6 : 1);
    return new Date(d.setDate(diff));
}

function calculateTrend(data) {
    if (data.length < 2) return { trend: 'stable', changePercent: 0 };
    
    const firstHalf = data.slice(0, Math.floor(data.length / 2));
    const secondHalf = data.slice(Math.floor(data.length / 2));
    
    const firstAvg = firstHalf.reduce((a, b) => a + b, 0) / firstHalf.length;
    const secondAvg = secondHalf.reduce((a, b) => a + b, 0) / secondHalf.length;
    
    const changePercent = firstAvg > 0 ? ((secondAvg - firstAvg) / firstAvg * 100) : 0;
    
    let trend = 'stable';
    if (changePercent > 10) trend = 'up';
    else if (changePercent < -10) trend = 'down';
    
    return { trend, changePercent };
}

function initAllCharts() {
    initRequestTrendChart();
    initRequestTypeChart();
    initUserGrowthChart();
    initAppDistributionChart();
    initErrorRateChart();
    initGeoDistributionChart();
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
                pointRadius: 2,
                pointHoverRadius: 6
            }]
        },
        options: getChartOptions('line')
    });
}

function initRequestTypeChart() {
    const ctx = document.getElementById('requestTypeChart');
    if (!ctx) return;

    requestTypeChart = new Chart(ctx, {
        type: 'doughnut',
        data: {
            labels: [],
            datasets: [{
                data: [],
                backgroundColor: [
                    '#3b82f6',
                    '#10b981',
                    '#f59e0b',
                    '#ef4444',
                    '#8b5cf6'
                ],
                borderWidth: 2,
                borderColor: '#fff'
            }]
        },
        options: getChartOptions('doughnut')
    });
}

function initUserGrowthChart() {
    const ctx = document.getElementById('userGrowthChart');
    if (!ctx) return;

    userGrowthChart = new Chart(ctx, {
        type: 'bar',
        data: {
            labels: [],
            datasets: [{
                label: '用户增长',
                data: [],
                backgroundColor: 'rgba(16, 185, 129, 0.8)',
                borderColor: '#10b981',
                borderWidth: 1,
                borderRadius: 4
            }]
        },
        options: getChartOptions('bar')
    });
}

function initAppDistributionChart() {
    const ctx = document.getElementById('appDistributionChart');
    if (!ctx) return;

    appDistributionChart = new Chart(ctx, {
        type: 'pie',
        data: {
            labels: [],
            datasets: [{
                data: [],
                backgroundColor: [
                    '#3b82f6',
                    '#10b981',
                    '#f59e0b',
                    '#ef4444',
                    '#8b5cf6'
                ],
                borderWidth: 2,
                borderColor: '#fff'
            }]
        },
        options: getChartOptions('pie')
    });
}

function initErrorRateChart() {
    const ctx = document.getElementById('errorRateChart');
    if (!ctx) return;

    errorRateChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: [],
            datasets: [{
                label: '错误率 (%)',
                data: [],
                borderColor: '#ef4444',
                backgroundColor: 'rgba(239, 68, 68, 0.1)',
                fill: true,
                tension: 0.4,
                pointRadius: 2,
                pointBackgroundColor: '#ef4444'
            }]
        },
        options: {
            ...getChartOptions('line'),
            scales: {
                ...getChartOptions('line').scales,
                y: {
                    ...getChartOptions('line').scales.y,
                    beginAtZero: true,
                    max: 10
                }
            }
        }
    });
}

function initGeoDistributionChart() {
    const ctx = document.getElementById('geoDistributionChart');
    if (!ctx) return;

    geoDistributionChart = new Chart(ctx, {
        type: 'bar',
        data: {
            labels: [],
            datasets: [{
                label: '请求占比 (%)',
                data: [],
                backgroundColor: [
                    'rgba(59, 130, 246, 0.8)',
                    'rgba(16, 185, 129, 0.8)',
                    'rgba(245, 158, 11, 0.8)',
                    'rgba(239, 68, 68, 0.8)',
                    'rgba(139, 92, 246, 0.8)',
                    'rgba(236, 72, 153, 0.8)',
                    'rgba(107, 114, 128, 0.8)'
                ],
                borderWidth: 0,
                borderRadius: 4
            }]
        },
        options: getChartOptions('bar')
    });
}

function getChartOptions(type) {
    const baseOptions = {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
            legend: {
                display: type !== 'line' && type !== 'bar',
                position: 'bottom',
                labels: {
                    padding: 15,
                    usePointStyle: true,
                    pointStyle: 'circle',
                    font: {
                        size: 12
                    }
                }
            },
            tooltip: {
                backgroundColor: 'rgba(0, 0, 0, 0.8)',
                padding: 12,
                titleFont: { size: 14, weight: 'bold' },
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
                    },
                    title: function(context) {
                        if (context && context[0]) {
                            return context[0].label || '';
                        }
                        return '';
                    }
                }
            }
        },
        animation: {
            duration: 800,
            easing: 'easeOutQuart'
        },
        interaction: {
            intersect: false,
            mode: 'index'
        }
    };

    if (type === 'line' || type === 'bar') {
        baseOptions.scales = {
            x: {
                grid: {
                    display: type === 'bar',
                    color: 'rgba(0, 0, 0, 0.05)'
                },
                ticks: {
                    maxRotation: 45,
                    minRotation: 0,
                    font: {
                        size: 11
                    }
                }
            },
            y: {
                beginAtZero: true,
                grid: {
                    color: 'rgba(0, 0, 0, 0.05)'
                },
                ticks: {
                    font: {
                        size: 11
                    },
                    callback: function(value) {
                        return formatChartValue(value);
                    }
                }
            }
        };
    }

    return baseOptions;
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

function updateRequestTrendChart(data) {
    if (!requestTrendChart || !data) return;
    requestTrendChart.data.labels = data.labels;
    requestTrendChart.data.datasets[0].data = data.data;
    requestTrendChart.config.type = currentChartType;
    requestTrendChart.update();
}

function updateRequestTypeChart(data) {
    if (!requestTypeChart || !data) return;
    requestTypeChart.data.labels = data.labels;
    requestTypeChart.data.datasets[0].data = data.data;
    requestTypeChart.update();
}

function updateUserGrowthChart(data) {
    if (!userGrowthChart || !data) return;
    userGrowthChart.data.labels = data.labels;
    userGrowthChart.data.datasets[0].data = data.data;
    userGrowthChart.update();
}

function updateAppDistributionChart(data) {
    if (!appDistributionChart || !data) return;
    appDistributionChart.data.labels = data.labels;
    appDistributionChart.data.datasets[0].data = data.data;
    appDistributionChart.update();
}

function updateErrorRateChart(data) {
    if (!errorRateChart || !data) return;
    errorRateChart.data.labels = data.labels;
    errorRateChart.data.datasets[0].data = data.data;
    errorRateChart.update();
}

function updateGeoDistributionChart(data) {
    if (!geoDistributionChart || !data) return;
    geoDistributionChart.data.labels = data.labels;
    geoDistributionChart.data.datasets[0].data = data.data;
    geoDistributionChart.update();
}

function exportStatsReport() {
    const dateRange = document.getElementById('dateRange')?.value || '30d';
    const formatOptions = [
        { value: '1', label: 'CSV', description: '通用格式，适合Excel和数据分析工具' },
        { value: '2', label: 'JSON', description: '结构化数据，适合程序处理' },
        { value: '3', label: 'Excel增强版', description: '包含详细说明和分析' }
    ];
    
    let formatHtml = '<div class="form-group"><label>选择导出格式：</label><select id="exportFormatSelect" class="form-control">';
    formatOptions.forEach(opt => {
        formatHtml += `<option value="${opt.value}">${opt.label} - ${opt.description}</option>`;
    });
    formatHtml += '</select></div>';
    
    const modalHtml = `
        <div class="modal fade" id="exportModal" tabindex="-1" role="dialog">
            <div class="modal-dialog" role="document">
                <div class="modal-content">
                    <div class="modal-header">
                        <h5 class="modal-title"><i class="fas fa-download"></i> 导出统计数据</h5>
                        <button type="button" class="close" data-dismiss="modal">
                            <span>&times;</span>
                        </button>
                    </div>
                    <div class="modal-body">
                        ${formatHtml}
                        <div class="alert alert-info mt-3">
                            <i class="fas fa-info-circle"></i>
                            <strong>提示：</strong>导出的数据将包含当前选择时间范围内的所有统计数据
                        </div>
                    </div>
                    <div class="modal-footer">
                        <button type="button" class="btn btn-secondary" data-dismiss="modal">取消</button>
                        <button type="button" class="btn btn-primary" onclick="confirmExport()">
                            <i class="fas fa-download"></i> 确认导出
                        </button>
                    </div>
                </div>
            </div>
        </div>
    `;
    
    const modalContainer = document.createElement('div');
    modalContainer.innerHTML = modalHtml;
    document.body.appendChild(modalContainer);
    
    $('#exportModal').modal('show');
    
    $('#exportModal').on('hidden.bs.modal', function() {
        document.body.removeChild(modalContainer);
    });
}

function confirmExport() {
    const dateRange = document.getElementById('dateRange')?.value || '30d';
    const format = document.getElementById('exportFormatSelect')?.value || '1';
    
    $('#exportModal').modal('hide');
    
    const reportData = {
        exportTime: new Date().toLocaleString('zh-CN'),
        dateRange: dateRange,
        summary: {
            totalRequests: document.getElementById('statTotalRequests')?.textContent || '0',
            avgResponse: document.getElementById('statAvgResponse')?.textContent || '0ms',
            successRate: document.getElementById('statSuccessRate')?.textContent || '0%',
            activeUsers: document.getElementById('statActiveUsers')?.textContent || '0'
        },
        changes: {
            requests: parseFloat(document.getElementById('statRequestsChange')?.textContent.replace(/[^0-9.-]/g, '')) || 0,
            responseTime: parseFloat(document.getElementById('statResponseChange')?.textContent.replace(/[^0-9.-]/g, '')) || 0,
            successRate: parseFloat(document.getElementById('statSuccessChange')?.textContent.replace(/[^0-9.-]/g, '')) || 0,
            activeUsers: parseFloat(document.getElementById('statUsersChange')?.textContent.replace(/[^0-9.-]/g, '')) || 0
        }
    };

    const timestamp = new Date().toISOString().slice(0,10);
    
    if (format === '2' || format === 'json') {
        exportAsJSON(reportData, `stats_report_${dateRange}_${timestamp}`);
    } else if (format === '3' || format === 'excel') {
        exportAsCSVEnhanced(reportData, `stats_report_${dateRange}_${timestamp}.csv`);
    } else {
        exportAsCSV(reportData, `stats_report_${dateRange}_${timestamp}.csv`);
    }
}

function exportAsCSV(reportData, filename) {
    const csvContent = [
        ['统计分析报表'].join(','),
        [''].join(','),
        ['导出时间', reportData.exportTime].join(','),
        ['时间范围', reportData.dateRange].join(','),
        [''].join(','),
        ['指标', '数值', '变化'].join(','),
        ['总请求量', reportData.summary.totalRequests, formatChange(reportData.changes.requests)].join(','),
        ['平均响应时间', reportData.summary.avgResponse, formatChange(reportData.changes.responseTime, true)].join(','),
        ['成功率', reportData.summary.successRate, formatChange(reportData.changes.successRate)].join(','),
        ['活跃用户', reportData.summary.activeUsers, formatChange(reportData.changes.activeUsers)].join(',')
    ].join('\n');

    downloadFile(csvContent, filename, 'text/csv;charset=utf-8');
    showToast('CSV报表导出成功', 'success');
}

function exportAsCSVEnhanced(reportData, filename) {
    const csvContent = [
        ['墨盾验证 - 统计分析报表'].join(','),
        [''].join(','),
        ['报表信息'].join(','),
        ['导出时间', reportData.exportTime].join(','),
        ['时间范围', reportData.dateRange].join(','),
        ['生成版本', 'v11.0'].join(','),
        [''].join(','),
        ['核心指标汇总'].join(','),
        ['指标名称', '当前值', '环比变化', '变化率', '评估'].join(','),
        ['总请求量', reportData.summary.totalRequests, formatChange(reportData.changes.requests), `${Math.abs(reportData.changes.requests).toFixed(1)}%`, reportData.changes.requests >= 0 ? '正向' : '负向'].join(','),
        ['平均响应时间', reportData.summary.avgResponse, formatChange(reportData.changes.responseTime, true), `${Math.abs(reportData.changes.responseTime).toFixed(1)}%`, reportData.changes.responseTime <= 0 ? '正向' : '负向'].join(','),
        ['成功率', reportData.summary.successRate, formatChange(reportData.changes.successRate), `${Math.abs(reportData.changes.successRate).toFixed(1)}%`, reportData.changes.successRate >= 0 ? '正向' : '负向'].join(','),
        ['活跃用户', reportData.summary.activeUsers, formatChange(reportData.changes.activeUsers), `${Math.abs(reportData.changes.activeUsers).toFixed(1)}%`, reportData.changes.activeUsers >= 0 ? '正向' : '负向'].join(','),
        [''].join(','),
        ['图表数据（请求量趋势）'].join(','),
        ...generateChartCSVData(),
        [''].join(','),
        ['详细说明'].join(','),
        ['1. 总请求量反映系统整体负载情况'].join(','),
        ['2. 平均响应时间越低，用户体验越好'].join(','),
        ['3. 成功率是服务质量的核心指标'].join(','),
        ['4. 活跃用户数体现产品用户规模'].join(',')
    ].join('\n');

    downloadFile(csvContent, filename, 'text/csv;charset=utf-8');
    showToast('增强报表导出成功', 'success');
}

function exportAsJSON(reportData, filename) {
    const jsonData = {
        report: {
            title: '墨盾验证 - 统计分析报表',
            version: 'v11.0',
            exportTime: reportData.exportTime,
            dateRange: reportData.dateRange,
            summary: reportData.summary,
            changes: reportData.changes,
            chartData: {
                requestTrend: requestTrendChart?.data ? {
                    labels: requestTrendChart.data.labels,
                    values: requestTrendChart.data.datasets[0]?.data
                } : null,
                requestType: requestTypeChart?.data ? {
                    labels: requestTypeChart.data.labels,
                    values: requestTypeChart.data.datasets[0]?.data
                } : null
            },
            metadata: {
                generatedBy: '墨盾验证管理后台',
                environment: window.location.hostname
            }
        }
    };

    const jsonString = JSON.stringify(jsonData, null, 2);
    downloadFile(jsonString, `${filename}.json`, 'application/json;charset=utf-8');
    showToast('JSON报表导出成功', 'success');
}

function generateChartCSVData() {
    const rows = [];
    if (requestTrendChart?.data) {
        rows.push(['日期', '请求量']);
        requestTrendChart.data.labels.forEach((label, i) => {
            const value = requestTrendChart.data.datasets[0]?.data[i] || 0;
            rows.push([label, value]);
        });
    }
    return rows;
}

function formatChange(value, isInverse = false) {
    const prefix = value > 0 ? '+' : '';
    const isPositive = isInverse ? value < 0 : value > 0;
    return prefix + value.toFixed(1) + '%';
}

function downloadFile(content, filename, mimeType) {
    const blob = new Blob(['\ufeff' + content], { type: mimeType });
    const link = document.createElement('a');
    const url = URL.createObjectURL(blob);
    link.setAttribute('href', url);
    link.setAttribute('download', filename);
    link.style.visibility = 'hidden';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
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

function initAdvancedStats() {
    initPredictionChart();
    initAnomalyChart();
    initBreakdownChart();
    loadAdvancedStats();
}

function setupAdvancedStatsRefresh() {
    const autoRefreshBtn = document.getElementById('autoRefreshBtn');
    if (autoRefreshBtn && autoRefreshBtn.parentElement) {
        const advancedRefreshBtn = document.createElement('button');
        advancedRefreshBtn.id = 'advancedRefreshBtn';
        advancedRefreshBtn.className = 'btn btn-outline-info btn-sm ms-2';
        advancedRefreshBtn.innerHTML = '<i class="fas fa-brain me-1"></i>智能刷新';

        autoRefreshBtn.parentElement.appendChild(advancedRefreshBtn);

        advancedRefreshBtn.addEventListener('click', () => {
            if (advancedStatsTimer) {
                clearInterval(advancedStatsTimer);
                advancedStatsTimer = null;
                advancedRefreshBtn.classList.remove('btn-info');
                advancedRefreshBtn.classList.add('btn-outline-info');
                showToast('智能刷新已关闭', 'info');
            } else {
                advancedStatsTimer = setInterval(loadAdvancedStats, ADVANCED_STATS_INTERVAL);
                advancedRefreshBtn.classList.add('btn-info');
                advancedRefreshBtn.classList.remove('btn-outline-info');
                showToast('智能刷新已开启（每30秒）', 'info');
            }
        });
    }
}

function initPredictionChart() {
    const ctx = document.getElementById('predictionChart');
    if (!ctx) return;

    predictionChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: [],
            datasets: [
                {
                    label: '实际值',
                    data: [],
                    borderColor: '#3b82f6',
                    backgroundColor: 'rgba(59, 130, 246, 0.1)',
                    fill: true,
                    tension: 0.4,
                    pointRadius: 3,
                    pointHoverRadius: 6
                },
                {
                    label: '预测值',
                    data: [],
                    borderColor: '#10b981',
                    backgroundColor: 'rgba(16, 185, 129, 0.1)',
                    fill: true,
                    tension: 0.4,
                    borderDash: [5, 5],
                    pointRadius: 3,
                    pointHoverRadius: 6
                },
                {
                    label: '置信区间上限',
                    data: [],
                    borderColor: 'rgba(16, 185, 129, 0.3)',
                    backgroundColor: 'rgba(16, 185, 129, 0.1)',
                    fill: '+1',
                    tension: 0.4,
                    borderDash: [2, 2],
                    pointRadius: 0
                },
                {
                    label: '置信区间下限',
                    data: [],
                    borderColor: 'rgba(16, 185, 129, 0.3)',
                    backgroundColor: 'transparent',
                    tension: 0.4,
                    borderDash: [2, 2],
                    pointRadius: 0
                }
            ]
        },
        options: {
            ...getChartOptions('line'),
            plugins: {
                ...getChartOptions('line').plugins,
                title: {
                    display: true,
                    text: '趋势预测分析',
                    font: { size: 16, weight: 'bold' }
                }
            }
        }
    });
}

function initAnomalyChart() {
    const ctx = document.getElementById('anomalyChart');
    if (!ctx) return;

    anomalyChart = new Chart(ctx, {
        type: 'scatter',
        data: {
            labels: [],
            datasets: [
                {
                    label: '正常',
                    data: [],
                    backgroundColor: 'rgba(59, 130, 246, 0.6)',
                    pointRadius: 6,
                    pointHoverRadius: 8
                },
                {
                    label: '异常',
                    data: [],
                    backgroundColor: 'rgba(239, 68, 68, 0.8)',
                    pointRadius: 8,
                    pointHoverRadius: 10
                }
            ]
        },
        options: {
            ...getChartOptions('scatter'),
            plugins: {
                ...getChartOptions('scatter').plugins,
                title: {
                    display: true,
                    text: '异常检测',
                    font: { size: 16, weight: 'bold' }
                }
            }
        }
    });
}

function initBreakdownChart() {
    const ctx = document.getElementById('breakdownChart');
    if (!ctx) return;

    breakdownChart = new Chart(ctx, {
        type: 'radar',
        data: {
            labels: ['滑块验证', '点选验证', '旋转验证', '拼图验证', '语音验证'],
            datasets: [
                {
                    label: '成功率',
                    data: [95, 92, 88, 85, 90],
                    backgroundColor: 'rgba(59, 130, 246, 0.2)',
                    borderColor: '#3b82f6',
                    pointBackgroundColor: '#3b82f6'
                },
                {
                    label: '平均响应时间',
                    data: [40, 55, 65, 70, 50],
                    backgroundColor: 'rgba(16, 185, 129, 0.2)',
                    borderColor: '#10b981',
                    pointBackgroundColor: '#10b981'
                }
            ]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    position: 'bottom',
                    labels: {
                        padding: 15,
                        usePointStyle: true
                    }
                },
                title: {
                    display: true,
                    text: '多维度分析',
                    font: { size: 16, weight: 'bold' }
                }
            },
            scales: {
                r: {
                    beginAtZero: true,
                    max: 100
                }
            }
        }
    });
}

function loadAdvancedStats() {
    loadBreakdownData();
    loadRealtimeMetrics();
    updateAdvancedStatsCards();
}

function loadBreakdownData() {
    const breakdownData = {
        labels: ['滑块验证', '点选验证', '旋转验证', '拼图验证', '语音验证'],
        successRates: [95, 92, 88, 85, 90],
        avgLatency: [40, 55, 65, 70, 50]
    };

    if (breakdownChart) {
        breakdownChart.data.datasets[0].data = breakdownData.successRates;
        breakdownChart.data.datasets[1].data = breakdownData.avgLatency;
        breakdownChart.update();
    }

    updateBreakdownCards(breakdownData);
}

function updateBreakdownCards(data) {
    const container = document.getElementById('breakdownCards');
    if (!container) return;

    container.innerHTML = '';

    data.labels.forEach((label, index) => {
        const card = document.createElement('div');
        card.className = 'col-md-4 mb-3';
        card.innerHTML = `
            <div class="card">
                <div class="card-body">
                    <h6 class="card-title">${label}</h6>
                    <div class="d-flex justify-content-between">
                        <div>
                            <small class="text-muted">成功率</small>
                            <div class="fw-bold text-success">${data.successRates[index]}%</div>
                        </div>
                        <div>
                            <small class="text-muted">平均延迟</small>
                            <div class="fw-bold text-primary">${data.avgLatency[index]}ms</div>
                        </div>
                    </div>
                </div>
            </div>
        `;
        container.appendChild(card);
    });
}

function loadRealtimeMetrics() {
    const metrics = {
        currentQPS: 7500 + Math.floor(Math.random() * 500),
        peakQPS: 8500 + Math.floor(Math.random() * 200),
        activeUsers: 15000 + Math.floor(Math.random() * 1000),
        averageLatency: 45 + Math.random() * 10,
        successRate: 0.94 + Math.random() * 0.02,
        blockedRate: 0.01 + Math.random() * 0.02
    };

    updateRealtimeMetricsDisplay(metrics);
}

function updateRealtimeMetricsDisplay(metrics) {
    const qpsEl = document.getElementById('realtimeQPS');
    const peakQpsEl = document.getElementById('realtimePeakQPS');
    const usersEl = document.getElementById('realtimeActiveUsers');
    const latencyEl = document.getElementById('realtimeLatency');
    const successEl = document.getElementById('realtimeSuccessRate');
    const blockedEl = document.getElementById('realtimeBlockedRate');

    if (qpsEl) animateValue('realtimeQPS', parseInt(qpsEl.textContent.replace(/,/g, '')) || 0, metrics.currentQPS, 500);
    if (peakQpsEl) peakQpsEl.textContent = formatNumber(metrics.peakQPS);
    if (usersEl) animateValue('realtimeActiveUsers', parseInt(usersEl.textContent.replace(/,/g, '')) || 0, metrics.activeUsers, 500);
    if (latencyEl) latencyEl.textContent = metrics.averageLatency.toFixed(1) + 'ms';
    if (successEl) successEl.textContent = (metrics.successRate * 100).toFixed(1) + '%';
    if (blockedEl) blockedEl.textContent = (metrics.blockedRate * 100).toFixed(1) + '%';
}

function updateAdvancedStatsCards() {
    const stats = {
        totalVerifications: 1000000 + Math.floor(Math.random() * 10000),
        successRate: 0.95,
        peakQPS: 8500,
        avgLatency: 45.5
    };

    animateValue('advancedTotalVerifications', 0, stats.totalVerifications, 1000);
    document.getElementById('advancedSuccessRate').textContent = (stats.successRate * 100).toFixed(1) + '%';
    document.getElementById('advancedPeakQPS').textContent = formatNumber(stats.peakQPS);
    document.getElementById('advancedAvgLatency').textContent = stats.avgLatency + 'ms';
}

function loadPredictionData() {
    const predictions = generateMockPredictions();
    updatePredictionChart(predictions);
    displayPredictions(predictions);
}

function generateMockPredictions() {
    const now = new Date();
    const historical = [];
    const forecast = [];
    const confidenceUpper = [];
    const confidenceLower = [];

    for (let i = -7; i <= 0; i++) {
        const date = new Date(now);
        date.setDate(date.getDate() + i);
        historical.push({
            x: date.toLocaleDateString('zh-CN'),
            y: 10000 + Math.floor(Math.random() * 2000) + i * 500
        });
    }

    for (let i = 1; i <= 7; i++) {
        const date = new Date(now);
        date.setDate(date.getDate() + i);
        const baseValue = 12000 + i * 300;
        const predictedValue = baseValue + Math.floor(Math.random() * 500);

        forecast.push({
            x: date.toLocaleDateString('zh-CN'),
            y: predictedValue
        });

        confidenceUpper.push({
            x: date.toLocaleDateString('zh-CN'),
            y: predictedValue + 1500
        });

        confidenceLower.push({
            x: date.toLocaleDateString('zh-CN'),
            y: predictedValue - 1500
        });
    }

    return { historical, forecast, confidenceUpper, confidenceLower };
}

function updatePredictionChart(predictions) {
    if (!predictionChart) return;

    const allLabels = [...predictions.historical.map(p => p.x), ...predictions.forecast.map(p => p.x)];
    predictionChart.data.labels = allLabels;

    predictionChart.data.datasets[0].data = [
        ...predictions.historical.map(p => ({ x: p.x, y: p.y })),
        ...Array(predictions.forecast.length).fill(null)
    ];

    predictionChart.data.datasets[1].data = [
        ...Array(predictions.historical.length).fill(null),
        ...predictions.forecast.map(p => ({ x: p.x, y: p.y }))
    ];

    predictionChart.data.datasets[2].data = [
        ...Array(predictions.historical.length).fill(null),
        ...predictions.confidenceUpper.map(p => ({ x: p.x, y: p.y }))
    ];

    predictionChart.data.datasets[3].data = [
        ...Array(predictions.historical.length).fill(null),
        ...predictions.confidenceLower.map(p => ({ x: p.x, y: p.y }))
    ];

    predictionChart.update();
}

function displayPredictions(predictions) {
    const container = document.getElementById('predictionsList');
    if (!container) return;

    container.innerHTML = '<h6 class="mb-3"><i class="fas fa-chart-line me-2"></i>未来7天预测</h6>';

    predictions.forecast.forEach((pred, index) => {
        const date = new Date();
        date.setDate(date.getDate() + index + 1);

        const confidence = (0.95 - (index + 1) * 0.03).toFixed(2);
        const card = document.createElement('div');
        card.className = 'card mb-2';
        card.innerHTML = `
            <div class="card-body py-2">
                <div class="d-flex justify-content-between align-items-center">
                    <div>
                        <strong>${date.toLocaleDateString('zh-CN', { weekday: 'short', month: 'short', day: 'numeric' })}</strong>
                        <span class="badge bg-success ms-2">置信度: ${(parseFloat(confidence) * 100).toFixed(0)}%</span>
                    </div>
                    <div class="text-end">
                        <div class="fw-bold">${formatNumber(pred.y)}</div>
                        <small class="text-muted">次验证</small>
                    </div>
                </div>
            </div>
        `;
        container.appendChild(card);
    });
}

function loadAnomalyData() {
    const anomalies = generateMockAnomalies();
    updateAnomalyChart(anomalies);
    displayAnomalies(anomalies);
}

function generateMockAnomalies() {
    const normal = [];
    const anomalies = [];

    for (let i = 0; i < 50; i++) {
        normal.push({
            x: Math.random() * 100,
            y: Math.random() * 100
        });
    }

    for (let i = 0; i < 5; i++) {
        anomalies.push({
            x: 80 + Math.random() * 20,
            y: 80 + Math.random() * 20,
            severity: Math.random() > 0.5 ? 'high' : 'medium',
            type: ['流量激增', '异常模式', '失败率升高'][Math.floor(Math.random() * 3)]
        });
    }

    return { normal, anomalies };
}

function updateAnomalyChart(anomalies) {
    if (!anomalyChart) return;

    anomalyChart.data.datasets[0].data = anomalies.normal;
    anomalyChart.data.datasets[1].data = anomalies.anomalies;
    anomalyChart.update();
}

function displayAnomalies(anomalies) {
    const container = document.getElementById('anomaliesList');
    if (!container) return;

    container.innerHTML = '<h6 class="mb-3"><i class="fas fa-exclamation-triangle me-2"></i>检测到的异常</h6>';

    if (anomalies.anomalies.length === 0) {
        container.innerHTML += '<div class="alert alert-success">未检测到异常情况</div>';
        return;
    }

    anomalies.anomalies.forEach((anomaly, index) => {
        const severityClass = anomaly.severity === 'high' ? 'bg-danger' : 'bg-warning';
        const card = document.createElement('div');
        card.className = 'card mb-2';
        card.innerHTML = `
            <div class="card-body py-2">
                <div class="d-flex justify-content-between align-items-center">
                    <div>
                        <span class="badge ${severityClass} me-2">${anomaly.type}</span>
                        <small class="text-muted">严重程度: ${anomaly.severity === 'high' ? '高' : '中'}</small>
                    </div>
                    <button class="btn btn-sm btn-outline-primary" onclick="analyzeAnomaly(${index})">
                        分析详情
                    </button>
                </div>
            </div>
        `;
        container.appendChild(card);
    });
}

function analyzeAnomaly(index) {
    showToast(`正在分析异常 #${index + 1}...`, 'info');
}

function generateCustomReport() {
    const config = {
        title: prompt('请输入报表标题:', '自定义统计报表'),
        metrics: prompt('请输入要包含的指标（逗号分隔）:', 'total,successRate,avgLatency,peakQPS'),
        dateRange: document.getElementById('dateRange')?.value || '30d',
        format: prompt('导出格式 (json/csv):', 'json')
    };

    if (!config.title || !config.metrics) {
        showToast('报表生成已取消', 'warning');
        return;
    }

    const reportData = {
        title: config.title,
        generatedAt: new Date().toLocaleString('zh-CN'),
        dateRange: config.dateRange,
        metrics: config.metrics.split(',').map(m => m.trim()),
        data: {
            totalVerifications: document.getElementById('statTotalRequests')?.textContent || '0',
            successRate: document.getElementById('statSuccessRate')?.textContent || '0%',
            avgLatency: document.getElementById('statAvgResponse')?.textContent || '0ms',
            peakQPS: document.getElementById('advancedPeakQPS')?.textContent || '0'
        }
    };

    if (config.format === 'csv') {
        exportCustomReportAsCSV(reportData, `custom_report_${new Date().toISOString().slice(0,10)}`);
    } else {
        exportCustomReportAsJSON(reportData, `custom_report_${new Date().toISOString().slice(0,10)}`);
    }
}

function exportCustomReportAsJSON(reportData, filename) {
    const jsonString = JSON.stringify(reportData, null, 2);
    downloadFile(jsonString, `${filename}.json`, 'application/json;charset=utf-8');
    showToast('自定义报表已导出为JSON', 'success');
}

function exportCustomReportAsCSV(reportData, filename) {
    const csvContent = [
        [reportData.title].join(','),
        [''].join(','),
        ['生成时间', reportData.generatedAt].join(','),
        ['时间范围', reportData.dateRange].join(','),
        [''].join(','),
        ['指标名称', '数值'].join(','),
        ...reportData.metrics.map(metric => {
            const key = metric.toLowerCase();
            const value = reportData.data[key] || 'N/A';
            return [metric, value].join(',');
        })
    ].join('\n');

    downloadFile(csvContent, `${filename}.csv`, 'text/csv;charset=utf-8');
    showToast('自定义报表已导出为CSV', 'success');
}

function formatNumber(num) {
    if (typeof num !== 'number') {
        num = parseInt(num) || 0;
    }
    return num.toLocaleString('zh-CN');
}
