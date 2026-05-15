let userGrowthChart, requestChart, appDistributionChart, errorRateChart;

document.addEventListener('DOMContentLoaded', () => {
    initCharts();
    setupEventListeners();
    loadStatsData();
});

function setupEventListeners() {
    const refreshBtn = document.getElementById('refreshBtn');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', loadStatsData);
    }

    const dateRange = document.getElementById('dateRange');
    if (dateRange) {
        dateRange.addEventListener('change', loadStatsData);
    }
}

async function loadStatsData() {
    const mockData = getMockStatsData();
    let data = mockData;

    try {
        const dateRange = document.getElementById('dateRange')?.value || '30d';
        const result = await auth.request(`/stats?range=${dateRange}`);
        if (result.code === 0) {
            data = result.data;
        }
    } catch (error) {
    }

    updateCharts(data);
}

function getMockStatsData() {
    return {
        userGrowth: {
            labels: ['1月', '2月', '3月', '4月', '5月', '6月', '7月'],
            data: [1200, 1900, 3000, 5000, 7200, 9800, 12456]
        },
        requests: {
            labels: ['1月', '2月', '3月', '4月', '5月', '6月', '7月'],
            data: [650000, 780000, 920000, 1100000, 1300000, 1600000, 1900000]
        },
        appDistribution: {
            labels: ['Web应用', '移动应用', '桌面应用', 'API服务', '其他'],
            data: [45, 30, 15, 8, 2]
        },
        errorRate: {
            labels: ['1月', '2月', '3月', '4月', '5月', '6月', '7月'],
            data: [5.2, 4.8, 3.5, 2.8, 2.1, 1.8, 1.5]
        }
    };
}

function initCharts() {
    const chartOptions = {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
            legend: {
                display: true,
                position: 'bottom'
            }
        }
    };

    const userGrowthCtx = document.getElementById('userGrowthChart');
    if (userGrowthCtx) {
        userGrowthChart = new Chart(userGrowthCtx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [{
                    label: '用户增长',
                    data: [],
                    borderColor: '#3b82f6',
                    backgroundColor: 'rgba(59, 130, 246, 0.1)',
                    fill: true,
                    tension: 0.4
                }]
            },
            options: {
                ...chartOptions,
                scales: {
                    y: {
                        beginAtZero: true
                    }
                }
            }
        });
    }

    const requestCtx = document.getElementById('requestChart');
    if (requestCtx) {
        requestChart = new Chart(requestCtx, {
            type: 'bar',
            data: {
                labels: [],
                datasets: [{
                    label: '请求量',
                    data: [],
                    backgroundColor: 'rgba(16, 185, 129, 0.8)',
                    borderColor: '#10b981',
                    borderWidth: 1
                }]
            },
            options: {
                ...chartOptions,
                scales: {
                    y: {
                        beginAtZero: true
                    }
                }
            }
        });
    }

    const appDistributionCtx = document.getElementById('appDistributionChart');
    if (appDistributionCtx) {
        appDistributionChart = new Chart(appDistributionCtx, {
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
                        '#64748b'
                    ]
                }]
            },
            options: chartOptions
        });
    }

    const errorRateCtx = document.getElementById('errorRateChart');
    if (errorRateCtx) {
        errorRateChart = new Chart(errorRateCtx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [{
                    label: '错误率 (%)',
                    data: [],
                    borderColor: '#ef4444',
                    backgroundColor: 'rgba(239, 68, 68, 0.1)',
                    fill: true,
                    tension: 0.4
                }]
            },
            options: {
                ...chartOptions,
                scales: {
                    y: {
                        beginAtZero: true,
                        max: 10
                    }
                }
            }
        });
    }
}

function updateCharts(data) {
    if (userGrowthChart && data.userGrowth) {
        userGrowthChart.data.labels = data.userGrowth.labels;
        userGrowthChart.data.datasets[0].data = data.userGrowth.data;
        userGrowthChart.update();
    }

    if (requestChart && data.requests) {
        requestChart.data.labels = data.requests.labels;
        requestChart.data.datasets[0].data = data.requests.data;
        requestChart.update();
    }

    if (appDistributionChart && data.appDistribution) {
        appDistributionChart.data.labels = data.appDistribution.labels;
        appDistributionChart.data.datasets[0].data = data.appDistribution.data;
        appDistributionChart.update();
    }

    if (errorRateChart && data.errorRate) {
        errorRateChart.data.labels = data.errorRate.labels;
        errorRateChart.data.datasets[0].data = data.errorRate.data;
        errorRateChart.update();
    }
}
