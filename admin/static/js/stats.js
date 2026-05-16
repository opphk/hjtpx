
let requestTrendChart = null;
let requestDistributionChart = null;
let appRankingChart = null;
let errorDistributionChart = null;
let currentPeriod = 'day';

document.addEventListener('DOMContentLoaded', function() {
    if (!Auth.requireAuth()) {
        return;
    }

    const user = Auth.getCurrentUser();
    if (user && user.username) {
        Auth.updateUserDisplay(user.username);
    }

    initCharts();
    loadStatistics();
    initEventListeners();
});

function initEventListeners() {
    const periodButtons = document.querySelectorAll('.period-selector button');
    periodButtons.forEach(button => {
        button.addEventListener('click', function() {
            periodButtons.forEach(btn => btn.classList.remove('active'));
            this.classList.add('active');
            currentPeriod = this.getAttribute('data-period');
            loadRequestTrendData(currentPeriod);
        });
    });

    const refreshBtn = document.getElementById('refreshStatsBtn');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', function() {
            loadStatistics();
            Auth.showToast('统计数据已刷新', 'success');
        });
    }
}

function initCharts() {
    initRequestTrendChart();
    initRequestDistributionChart();
    initAppRankingChart();
    initErrorDistributionChart();
}

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

function initRequestDistributionChart() {
    const ctx = document.getElementById('requestDistributionChart');
    if (!ctx) return;

    if (requestDistributionChart) {
        requestDistributionChart.destroy();
    }

    requestDistributionChart = new Chart(ctx, {
        type: 'doughnut',
        data: {
            labels: ['iOS', 'Android', 'Web', '其他'],
            datasets: [{
                data: [300, 250, 200, 50],
                backgroundColor: [
                    'rgba(59, 130, 246, 0.8)',
                    'rgba(34, 197, 94, 0.8)',
                    'rgba(168, 85, 247, 0.8)',
                    'rgba(107, 114, 128, 0.8)'
                ],
                borderWidth: 0
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: true,
            plugins: {
                legend: {
                    position: 'bottom',
                }
            }
        }
    });
}

function initAppRankingChart() {
    const ctx = document.getElementById('appRankingChart');
    if (!ctx) return;

    if (appRankingChart) {
        appRankingChart.destroy();
    }

    appRankingChart = new Chart(ctx, {
        type: 'bar',
        data: {
            labels: [],
            datasets: [{
                label: '请求量',
                data: [],
                backgroundColor: 'rgba(59, 130, 246, 0.8)',
                borderColor: 'rgb(59, 130, 246)',
                borderWidth: 1
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            indexAxis: 'y',
            plugins: {
                legend: {
                    display: false,
                }
            },
            scales: {
                y: {
                    beginAtZero: true,
                    grid: {
                        display: false,
                    }
                },
                x: {
                    grid: {
                        drawBorder: false,
                    }
                }
            }
        }
    });
}

function initErrorDistributionChart() {
    const ctx = document.getElementById('errorDistributionChart');
    if (!ctx) return;

    if (errorDistributionChart) {
        errorDistributionChart.destroy();
    }

    errorDistributionChart = new Chart(ctx, {
        type: 'polarArea',
        data: {
            labels: ['超时错误', '认证失败', '权限不足', '参数错误', '服务器错误'],
            datasets: [{
                data: [120, 80, 60, 45, 30],
                backgroundColor: [
                    'rgba(239, 68, 68, 0.8)',
                    'rgba(245, 158, 11, 0.8)',
                    'rgba(59, 130, 246, 0.8)',
                    'rgba(168, 85, 247, 0.8)',
                    'rgba(107, 114, 128, 0.8)'
                ],
                borderWidth: 0
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    position: 'right',
                }
            }
        }
    });
}

async function loadStatistics() {
    try {
        await Promise.all([
            loadSummaryStats(),
            loadRequestTrendData(currentPeriod),
            loadRequestDistributionData(),
            loadAppRankingData(),
            loadErrorDistributionData(),
            loadDetailedTable()
        ]);
    } catch (error) {
        console.error('加载统计数据失败:', error);
        loadMockStatistics();
    }
}

async function loadSummaryStats() {
    try {
        const response = await fetch('/admin/api/stats/summary');
        if (!response.ok) throw new Error('获取统计数据失败');

        const data = await response.json();

        updateStatCard('totalUsers', data.totalUsers, data.usersTrend);
        updateStatCard('totalApps', data.totalApps, data.appsTrend);
        updateStatCard('totalRequests', data.totalRequests, data.requestsTrend);
        updateLatencyCard(data.avgLatency, data.latencyTrend);
    } catch (error) {
        console.error('加载摘要统计数据失败:', error);
        updateStatCard('totalUsers', 1234, 12.5);
        updateStatCard('totalApps', 56, 8.3);
        updateStatCard('totalRequests', 89765, 15.2);
        updateLatencyCard(45, -5.2);
    }
}

function updateStatCard(elementId, value, trend) {
    const valueElement = document.getElementById(elementId);
    const trendElement = document.getElementById(elementId.replace('total', '').toLowerCase() + 'Trend');

    if (valueElement) {
        valueElement.textContent = value.toLocaleString();
    }

    if (trendElement) {
        const isPositive = trend >= 0;
        trendElement.className = `trend ${isPositive ? 'up' : 'down'}`;
        trendElement.innerHTML = `<i class="fas fa-arrow-${isPositive ? 'up' : 'down'} me-1"></i>${isPositive ? '+' : ''}${Math.abs(trend)}% 较上周`;
    }
}

function updateLatencyCard(value, trend) {
    const valueElement = document.getElementById('avgLatency');
    const trendElement = document.getElementById('latencyTrend');

    if (valueElement) {
        valueElement.textContent = `${value}ms`;
    }

    if (trendElement) {
        const isPositive = trend >= 0;
        trendElement.className = `trend ${isPositive ? 'down' : 'up'}`;
        trendElement.innerHTML = `<i class="fas fa-arrow-${isPositive ? 'up' : 'down'} me-1"></i>${isPositive ? '+' : ''}${Math.abs(trend)}% 较上周`;
    }
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
        loadMockRequestTrendData(period);
    }
}

function loadMockRequestTrendData(period) {
    const labels = [];
    const successData = [];
    const failedData = [];

    let count = period === 'day' ? 24 : period === 'week' ? 7 : 30;

    for (let i = 0; i < count; i++) {
        if (period === 'day') {
            labels.push(`${i}:00`);
        } else if (period === 'week') {
            const date = new Date();
            date.setDate(date.getDate() - (count - 1 - i));
            labels.push(`${date.getMonth() + 1}/${date.getDate()}`);
        } else {
            const date = new Date();
            date.setDate(date.getDate() - (count - 1 - i));
            labels.push(`${date.getMonth() + 1}/${date.getDate()}`);
        }
        successData.push(Math.floor(Math.random() * 1000) + 500);
        failedData.push(Math.floor(Math.random() * 50) + 10);
    }

    if (requestTrendChart) {
        requestTrendChart.data.labels = labels;
        requestTrendChart.data.datasets[0].data = successData;
        requestTrendChart.data.datasets[1].data = failedData;
        requestTrendChart.update();
    }
}

async function loadRequestDistributionData() {
    try {
        const response = await fetch('/admin/api/stats/request-distribution');
        if (!response.ok) throw new Error('获取请求分布失败');

        const data = await response.json();

        if (requestDistributionChart) {
            requestDistributionChart.data.labels = data.labels;
            requestDistributionChart.data.datasets[0].data = data.values;
            requestDistributionChart.update();
        }
    } catch (error) {
        console.error('加载请求分布数据失败:', error);
        if (requestDistributionChart) {
            requestDistributionChart.data.datasets[0].data = [300, 250, 200, 50];
            requestDistributionChart.update();
        }
    }
}

async function loadAppRankingData() {
    try {
        const response = await fetch('/admin/api/stats/app-ranking');
        if (!response.ok) throw new Error('获取应用排名失败');

        const data = await response.json();

        if (appRankingChart) {
            appRankingChart.data.labels = data.labels;
            appRankingChart.data.datasets[0].data = data.values;
            appRankingChart.update();
        }
    } catch (error) {
        console.error('加载应用排名数据失败:', error);
        if (appRankingChart) {
            appRankingChart.data.labels = ['应用A', '应用B', '应用C', '应用D', '应用E'];
            appRankingChart.data.datasets[0].data = [1200, 980, 750, 520, 380];
            appRankingChart.update();
        }
    }
}

async function loadErrorDistributionData() {
    try {
        const response = await fetch('/admin/api/stats/error-distribution');
        if (!response.ok) throw new Error('获取错误分布失败');

        const data = await response.json();

        if (errorDistributionChart) {
            errorDistributionChart.data.labels = data.labels;
            errorDistributionChart.data.datasets[0].data = data.values;
            errorDistributionChart.update();
        }
    } catch (error) {
        console.error('加载错误分布数据失败:', error);
        if (errorDistributionChart) {
            errorDistributionChart.data.datasets[0].data = [120, 80, 60, 45, 30];
            errorDistributionChart.update();
        }
    }
}

async function loadDetailedTable() {
    try {
        const response = await fetch('/admin/api/stats/detailed');
        if (!response.ok) throw new Error('获取详细数据失败');

        const data = await response.json();
        renderStatsTable(data);
    } catch (error) {
        console.error('加载详细数据失败:', error);
        loadMockDetailedTable();
    }
}

function loadMockDetailedTable() {
    const mockData = [
        { time: '2024-01-15', success: 1250, failed: 45, successRate: 96.5, avgLatency: 42, p99Latency: 120 },
        { time: '2024-01-14', success: 1180, failed: 38, successRate: 96.9, avgLatency: 38, p99Latency: 115 },
        { time: '2024-01-13', success: 1320, failed: 52, successRate: 96.2, avgLatency: 45, p99Latency: 125 },
        { time: '2024-01-12', success: 1090, failed: 30, successRate: 97.3, avgLatency: 35, p99Latency: 108 },
        { time: '2024-01-11', success: 1255, failed: 42, successRate: 96.8, avgLatency: 40, p99Latency: 118 }
    ];
    renderStatsTable(mockData);
}

function renderStatsTable(data) {
    const tbody = document.getElementById('statsTableBody');
    if (!tbody) return;

    tbody.innerHTML = '';

    if (data.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="6" class="text-center text-muted py-5">
                    暂无数据
                </td>
            </tr>
        `;
        return;
    }

    data.forEach(row => {
        const tr = document.createElement('tr');
        const successRateClass = row.successRate >= 95 ? 'text-success' : row.successRate >= 90 ? 'text-warning' : 'text-danger';

        tr.innerHTML = `
            <td><small class="text-muted">${row.time}</small></td>
            <td>${row.success.toLocaleString()}</td>
            <td class="text-danger">${row.failed.toLocaleString()}</td>
            <td class="${successRateClass} fw-medium">${row.successRate.toFixed(1)}%</td>
            <td>${row.avgLatency}ms</td>
            <td>${row.p99Latency}ms</td>
        `;

        tbody.appendChild(tr);
    });
}

function loadMockStatistics() {
    updateStatCard('totalUsers', 1234, 12.5);
    updateStatCard('totalApps', 56, 8.3);
    updateStatCard('totalRequests', 89765, 15.2);
    updateLatencyCard(45, -5.2);

    loadMockRequestTrendData(currentPeriod);

    if (requestDistributionChart) {
        requestDistributionChart.data.datasets[0].data = [300, 250, 200, 50];
        requestDistributionChart.update();
    }

    if (appRankingChart) {
        appRankingChart.data.labels = ['应用A', '应用B', '应用C', '应用D', '应用E'];
        appRankingChart.data.datasets[0].data = [1200, 980, 750, 520, 380];
        appRankingChart.update();
    }

    if (errorDistributionChart) {
        errorDistributionChart.data.datasets[0].data = [120, 80, 60, 45, 30];
        errorDistributionChart.update();
    }

    loadMockDetailedTable();
}

window.Stats = {
    loadStatistics: loadStatistics,
    loadRequestTrendData: loadRequestTrendData
};
