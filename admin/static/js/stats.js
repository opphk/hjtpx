let userGrowthChart, requestTrendChart, requestTypeChart, appDistributionChart, errorRateChart, geoDistributionChart;
let currentChartType = 'line';

document.addEventListener('DOMContentLoaded', () => {
    initAllCharts();
    setupEventListeners();
    loadStatsData();
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
}

function switchChartType(type) {
    currentChartType = type;
    if (requestTrendChart) {
        requestTrendChart.config.type = type;
        requestTrendChart.update();
    }
}

async function loadStatsData() {
    const dateRange = document.getElementById('dateRange')?.value || '30d';
    const comparePeriod = document.getElementById('comparePeriod')?.value || '';
    const mockData = getMockStatsData(dateRange);

    try {
        const result = await auth.request(`/admin/stats?range=${dateRange}&compare=${comparePeriod}`);
        if (result.code === 0) {
            updateAllStats(result.data);
        } else {
            updateAllStats(mockData);
        }
    } catch (error) {
        updateAllStats(mockData);
    }
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
        baseOptions.scales = {
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
    const format = prompt('请选择导出格式 (1: CSV, 2: JSON, 3: Excel):', '1');
    
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

    if (format === '2' || format === 'json') {
        exportAsJSON(reportData, `stats_report_${dateRange}_${new Date().toISOString().slice(0,10)}`);
    } else if (format === '3' || format === 'excel') {
        exportAsCSVEnhanced(reportData, `stats_report_${dateRange}_${new Date().toISOString().slice(0,10)}.csv`);
    } else {
        exportAsCSV(reportData, `stats_report_${dateRange}_${new Date().toISOString().slice(0,10)}.csv`);
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
