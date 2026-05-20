let chartInstances = {};
const CHART_COLORS = {
    primary: 'rgba(0, 123, 255, 1)',
    primaryLight: 'rgba(0, 123, 255, 0.2)',
    success: 'rgba(40, 167, 69, 1)',
    successLight: 'rgba(40, 167, 69, 0.2)',
    danger: 'rgba(220, 53, 69, 1)',
    dangerLight: 'rgba(220, 53, 69, 0.2)',
    warning: 'rgba(255, 193, 7, 1)',
    warningLight: 'rgba(255, 193, 7, 0.2)',
    info: 'rgba(23, 162, 184, 1)',
    infoLight: 'rgba(23, 162, 184, 0.2)'
};

const CHART_UPDATE_INTERVAL = 5000;
let chartUpdateTimer = null;
let realtimeChartData = [];

function initDashboardCharts() {
    if (typeof Chart === 'undefined') {
        console.error('Chart.js 未加载');
        return;
    }

    Chart.defaults.global.animation.duration = 1000;
    Chart.defaults.global.animation.easing = 'easeOutQuart';
    Chart.defaults.global.responsive = true;
    Chart.defaults.global.maintainAspectRatio = false;

    initAllCharts();
    startChartUpdates();
    setupChartInteractions();

    window.addEventListener('resize', debounce(() => {
        Object.values(chartInstances).forEach(chart => {
            if (chart) chart.resize();
        });
    }, 250));
}

function initAllCharts() {
    initTrendChart();
    initRiskDistributionChart();
    initCaptchaTypeChart();
    initRealtimeChart();
    initAppUsageChart();
}

function initTrendChart() {
    const canvas = document.getElementById('trendChartCanvas');
    if (!canvas) return;

    const ctx = canvas.getContext('2d');

    const labels = generateTimeLabels(24);
    const data = generateMockTrendData(24);

    chartInstances.trend = new Chart(ctx, {
        type: 'line',
        data: {
            labels: labels,
            datasets: [{
                label: '验证请求数',
                data: data,
                borderColor: CHART_COLORS.primary,
                backgroundColor: CHART_COLORS.primaryLight,
                fill: true,
                tension: 0.4,
                pointRadius: 2,
                pointHoverRadius: 5
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    display: true,
                    position: 'top'
                },
                tooltip: {
                    mode: 'index',
                    intersect: false,
                    backgroundColor: 'rgba(0,0,0,0.8)',
                    titleColor: '#fff',
                    bodyColor: '#fff',
                    padding: 12,
                    displayColors: true
                }
            },
            scales: {
                x: {
                    display: true,
                    scaleLabel: {
                        display: true,
                        labelString: '时间'
                    },
                    grid: {
                        display: false
                    }
                },
                y: {
                    display: true,
                    scaleLabel: {
                        display: true,
                        labelString: '请求数'
                    },
                    beginAtZero: true,
                    grid: {
                        color: 'rgba(0,0,0,0.05)'
                    }
                }
            },
            interaction: {
                mode: 'nearest',
                axis: 'x',
                intersect: false
            }
        }
    });

    chartInstances.trend.canvas.id = 'trendChart';
}

function initRiskDistributionChart() {
    const canvas = document.getElementById('pieChart');
    if (!canvas) return;

    const ctx = canvas.getContext('2d');

    chartInstances.riskDistribution = new Chart(ctx, {
        type: 'doughnut',
        data: {
            labels: ['低风险', '中风险', '高风险', '极高风险'],
            datasets: [{
                data: [3000, 1500, 500, 100],
                backgroundColor: [
                    CHART_COLORS.success,
                    CHART_COLORS.warning,
                    'rgba(253, 126, 20, 1)',
                    CHART_COLORS.danger
                ],
                borderWidth: 2,
                borderColor: '#fff'
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    display: true,
                    position: 'bottom',
                    labels: {
                        padding: 15,
                        usePointStyle: true
                    }
                },
                tooltip: {
                    callbacks: {
                        label: function(context) {
                            const total = context.dataset.data.reduce((a, b) => a + b, 0);
                            const percentage = ((context.raw / total) * 100).toFixed(1);
                            return `${context.label}: ${context.raw} (${percentage}%)`;
                        }
                    }
                }
            },
            cutout: '60%'
        }
    });
}

function initCaptchaTypeChart() {
    const canvas = document.getElementById('captchaTypeChart');
    if (!canvas) return;

    const ctx = canvas.getContext('2d');

    chartInstances.captchaType = new Chart(ctx, {
        type: 'bar',
        data: {
            labels: ['滑动验证', '点选验证', '图形验证', '文字验证'],
            datasets: [{
                label: '使用次数',
                data: [2500, 1500, 1000, 500],
                backgroundColor: [
                    CHART_COLORS.primary,
                    CHART_COLORS.success,
                    CHART_COLORS.warning,
                    CHART_COLORS.info
                ],
                borderRadius: 4,
                borderSkipped: false
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
                    mode: 'index',
                    intersect: false
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
                        color: 'rgba(0,0,0,0.05)'
                    }
                }
            }
        }
    });
}

function initRealtimeChart() {
    const canvas = document.getElementById('realtimeChartCanvas');
    if (!canvas) return;

    const ctx = canvas.getContext('2d');

    realtimeChartData = Array(60).fill(0).map(() => ({
        time: new Date(Date.now() - (60 - realtimeChartData.length - 1) * 1000),
        value: Math.floor(Math.random() * 50) + 30
    }));

    chartInstances.realtime = new Chart(ctx, {
        type: 'line',
        data: {
            labels: realtimeChartData.map(d => formatTime(d.time)),
            datasets: [{
                label: '实时QPS',
                data: realtimeChartData.map(d => d.value),
                borderColor: CHART_COLORS.success,
                backgroundColor: CHART_COLORS.successLight,
                fill: true,
                tension: 0.4,
                pointRadius: 0,
                pointHoverRadius: 4
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    display: true,
                    position: 'top'
                },
                tooltip: {
                    mode: 'index',
                    intersect: false
                }
            },
            scales: {
                x: {
                    display: true,
                    grid: {
                        display: false
                    },
                    ticks: {
                        maxTicksLimit: 10
                    }
                },
                y: {
                    display: true,
                    beginAtZero: true,
                    grid: {
                        color: 'rgba(0,0,0,0.05)'
                    }
                }
            },
            animation: {
                duration: 0
            }
        }
    });
}

function initAppUsageChart() {
    const canvas = document.getElementById('appUsageChart');
    if (!canvas) return;

    const ctx = canvas.getContext('2d');

    chartInstances.appUsage = new Chart(ctx, {
        type: 'pie',
        data: {
            labels: ['App_Web', 'App_Mobile', 'App_API', 'App_Admin', 'App_Test'],
            datasets: [{
                data: [3500, 2800, 2100, 1500, 800],
                backgroundColor: [
                    CHART_COLORS.primary,
                    CHART_COLORS.success,
                    CHART_COLORS.warning,
                    CHART_COLORS.info,
                    'rgba(139, 92, 246, 1)'
                ],
                borderWidth: 2,
                borderColor: '#fff'
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    display: true,
                    position: 'bottom',
                    labels: {
                        padding: 15,
                        usePointStyle: true
                    }
                },
                tooltip: {
                    callbacks: {
                        label: function(context) {
                            const total = context.dataset.data.reduce((a, b) => a + b, 0);
                            const percentage = ((context.raw / total) * 100).toFixed(1);
                            return `${context.label}: ${context.raw} (${percentage}%)`;
                        }
                    }
                }
            }
        }
    });
}

function startChartUpdates() {
    chartUpdateTimer = setInterval(() => {
        updateRealtimeChartData();
        updateTrendChartData();
    }, CHART_UPDATE_INTERVAL);
}

function updateRealtimeChartData() {
    if (!chartInstances.realtime) return;

    const newValue = Math.floor(Math.random() * 50) + 30;
    const now = new Date();

    realtimeChartData.push({ time: now, value: newValue });

    if (realtimeChartData.length > 60) {
        realtimeChartData.shift();
    }

    chartInstances.realtime.data.labels = realtimeChartData.map(d => formatTime(d.time));
    chartInstances.realtime.data.datasets[0].data = realtimeChartData.map(d => d.value);
    chartInstances.realtime.update('none');

    const currentQPSEl = document.getElementById('currentQPS');
    if (currentQPSEl) {
        currentQPSEl.textContent = newValue + ' QPS';
    }
}

function updateTrendChartData() {
    if (!chartInstances.trend) return;

    const now = new Date();
    const hour = now.getHours();
    const currentLabel = `${hour}:00`;

    const newValue = Math.floor(Math.random() * 500) + 100;

    chartInstances.trend.data.labels.push(currentLabel);
    chartInstances.trend.data.datasets[0].data.push(newValue);

    if (chartInstances.trend.data.labels.length > 24) {
        chartInstances.trend.data.labels.shift();
        chartInstances.trend.data.datasets[0].data.shift();
    }

    chartInstances.trend.update('none');
}

function generateTimeLabels(count) {
    const labels = [];
    const now = new Date();
    for (let i = count - 1; i >= 0; i--) {
        const time = new Date(now.getTime() - i * 3600000);
        labels.push(`${time.getHours()}:00`);
    }
    return labels;
}

function generateMockTrendData(count) {
    return Array.from({ length: count }, () => Math.floor(Math.random() * 500) + 100);
}

function formatTime(date) {
    return date.toLocaleTimeString('zh-CN', {
        hour: '2-digit',
        minute: '2-digit'
    });
}

function setupChartInteractions() {
    Object.keys(chartInstances).forEach(chartName => {
        const chart = chartInstances[chartName];
        if (chart) {
            chart.canvas.addEventListener('click', () => {
                showChartDetail(chartName);
            });

            chart.canvas.style.cursor = 'pointer';
            chart.canvas.title = '点击查看详情';
        }
    });
}

function showChartDetail(chartName) {
    const chart = chartInstances[chartName];
    if (!chart) return;

    const titles = {
        trend: '验证请求趋势',
        riskDistribution: '风险分布详情',
        captchaType: '验证码类型分布',
        realtime: '实时QPS监控',
        appUsage: '应用使用分布'
    };

    const modal = new bootstrap.Modal(document.getElementById('chartDetailModal'));
    document.getElementById('chartDetailTitle').innerHTML = `<i class="fas fa-chart-bar"></i> ${titles[chartName] || '图表详情'}`;

    const data = chart.data.datasets[0].data;
    const total = data.reduce((a, b) => a + b, 0);
    const avg = (total / data.length).toFixed(2);
    const max = Math.max(...data);
    const min = Math.min(...data);

    document.getElementById('chartDetailContent').innerHTML = `
        <div class="row text-center">
            <div class="col-md-4 mb-3">
                <h5 class="text-primary">${formatNumber(total)}</h5>
                <small class="text-muted">总计</small>
            </div>
            <div class="col-md-4 mb-3">
                <h5 class="text-success">${formatNumber(avg)}</h5>
                <small class="text-muted">平均值</small>
            </div>
            <div class="col-md-4 mb-3">
                <h5 class="text-danger">${formatNumber(max)}</h5>
                <small class="text-muted">最大值</small>
            </div>
        </div>
        <div class="text-center mt-3">
            <small class="text-muted">更新时间: ${new Date().toLocaleString('zh-CN')}</small>
        </div>
    `;

    modal.show();
}

function exportChartAsImage(chartName, filename) {
    const chart = chartInstances[chartName];
    if (!chart) return;

    const url = chart.toBase64Image('image/png', 1);
    const link = document.createElement('a');
    link.download = filename + '.png';
    link.href = url;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
}

function exportChartData(chartName) {
    const chart = chartInstances[chartName];
    if (!chart) return;

    const data = {
        labels: chart.data.labels,
        datasets: chart.data.datasets.map(ds => ({
            label: ds.label,
            data: ds.data
        })),
        exportTime: new Date().toISOString()
    };

    const jsonContent = JSON.stringify(data, null, 2);
    const blob = new Blob([jsonContent], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.download = `chart_data_${Date.now()}.json`;
    link.href = url;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
}

function refreshAllCharts() {
    Object.values(chartInstances).forEach(chart => {
        if (chart) {
            chart.update();
        }
    });
}

function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

document.addEventListener('DOMContentLoaded', function() {
    if (document.querySelector('[id$="Chart"]') || document.querySelector('[id*="Chart"]')) {
        setTimeout(initDashboardCharts, 100);
    }
});

window.dashboardCharts = {
    instances: chartInstances,
    init: initDashboardCharts,
    refresh: refreshAllCharts,
    exportImage: exportChartAsImage,
    exportData: exportChartData
};
