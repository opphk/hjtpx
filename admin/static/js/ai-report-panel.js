let trendForecastChart, anomalyChart, reportChart;
let currentReportType = 'executive';
let refreshTimer = null;
let isAutoRefresh = false;

const AUTO_REFRESH_INTERVAL = 60000;
const API_BASE = '/api/v1';

document.addEventListener('DOMContentLoaded', () => {
    initializeAIReportPanel();
    setupEventListeners();
    loadInitialData();
});

function initializeAIReportPanel() {
    initializeCharts();
    setupAutoRefresh();
}

function initializeCharts() {
    const chartOptions = {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
            legend: {
                display: true,
                position: 'top'
            },
            tooltip: {
                enabled: true,
                mode: 'index',
                intersect: false
            }
        },
        scales: {
            y: {
                beginAtZero: true
            }
        }
    };

    const trendCtx = document.getElementById('trendForecastChart');
    if (trendCtx) {
        trendForecastChart = new Chart(trendCtx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [{
                    label: 'Historical',
                    data: [],
                    borderColor: 'rgb(75, 192, 192)',
                    backgroundColor: 'rgba(75, 192, 192, 0.2)',
                    tension: 0.1
                }, {
                    label: 'Forecast',
                    data: [],
                    borderColor: 'rgb(255, 99, 132)',
                    backgroundColor: 'rgba(255, 99, 132, 0.2)',
                    borderDash: [5, 5],
                    tension: 0.1
                }, {
                    label: 'Upper Bound',
                    data: [],
                    borderColor: 'rgba(255, 99, 132, 0.3)',
                    fill: false,
                    pointRadius: 0,
                    borderDash: [2, 2]
                }, {
                    label: 'Lower Bound',
                    data: [],
                    borderColor: 'rgba(255, 99, 132, 0.3)',
                    fill: '-1',
                    pointRadius: 0,
                    borderDash: [2, 2]
                }]
            },
            options: chartOptions
        });
    }

    const anomalyCtx = document.getElementById('anomalyChart');
    if (anomalyCtx) {
        anomalyChart = new Chart(anomalyCtx, {
            type: 'bar',
            data: {
                labels: [],
                datasets: [{
                    label: 'Value',
                    data: [],
                    backgroundColor: []
                }]
            },
            options: {
                ...chartOptions,
                onClick: (event, elements) => {
                    if (elements.length > 0) {
                        const index = elements[0].index;
                        showAnomalyDetails(index);
                    }
                }
            }
        });
    }
}

function setupEventListeners() {
    const refreshBtn = document.getElementById('refreshBtn');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', loadInitialData);
    }

    const autoRefreshBtn = document.getElementById('autoRefreshBtn');
    if (autoRefreshBtn) {
        autoRefreshBtn.addEventListener('click', toggleAutoRefresh);
    }

    const reportTypeSelect = document.getElementById('reportTypeSelect');
    if (reportTypeSelect) {
        reportTypeSelect.addEventListener('change', (e) => {
            currentReportType = e.target.value;
            generateNLReport();
        });
    }

    const timeRangeSelect = document.getElementById('timeRangeSelect');
    if (timeRangeSelect) {
        timeRangeSelect.addEventListener('change', loadInitialData);
    }

    const exportBtn = document.getElementById('exportReportBtn');
    if (exportBtn) {
        exportBtn.addEventListener('click', exportReport);
    }

    const queryInput = document.getElementById('nlQueryInput');
    if (queryInput) {
        queryInput.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                processNLQuery();
            }
        });
    }

    const submitQueryBtn = document.getElementById('submitQueryBtn');
    if (submitQueryBtn) {
        submitQueryBtn.addEventListener('click', processNLQuery);
    }

    document.querySelectorAll('[data-metric]').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const metric = e.target.dataset.metric;
            loadTrendForecast(metric);
        });
    });
}

function setupAutoRefresh() {
    const autoRefreshToggle = document.getElementById('autoRefreshToggle');
    if (autoRefreshToggle) {
        autoRefreshToggle.addEventListener('change', (e) => {
            isAutoRefresh = e.target.checked;
            if (isAutoRefresh) {
                startAutoRefresh();
            } else {
                stopAutoRefresh();
            }
        });
    }
}

function startAutoRefresh() {
    if (refreshTimer) {
        clearInterval(refreshTimer);
    }
    refreshTimer = setInterval(loadInitialData, AUTO_REFRESH_INTERVAL);
    showToast('Auto-refresh enabled', 'info');
}

function stopAutoRefresh() {
    if (refreshTimer) {
        clearInterval(refreshTimer);
        refreshTimer = null;
    }
    showToast('Auto-refresh disabled', 'info');
}

function loadInitialData() {
    loadDashboardStats();
    loadTrendForecast('requests');
    loadAnomalyDetection();
    loadReportSummary();
}

async function loadDashboardStats() {
    try {
        const response = await fetch(`${API_BASE}/ai-report/stats`);
        const data = await response.json();

        updateStatCard('totalRequests', data.totalRequests);
        updateStatCard('successRate', `${(data.successRate * 100).toFixed(2)}%`);
        updateStatCard('anomaliesDetected', data.anomaliesDetected);
        updateStatCard('predictionsCount', data.predictionsCount);

        if (data.alerts && data.alerts.length > 0) {
            displayAlerts(data.alerts);
        }
    } catch (error) {
        console.error('Failed to load dashboard stats:', error);
        showToast('Failed to load dashboard stats', 'error');
    }
}

function updateStatCard(cardId, value) {
    const card = document.getElementById(cardId);
    if (card) {
        const valueElement = card.querySelector('.stat-value');
        if (valueElement) {
            valueElement.textContent = value;
        }
    }
}

function displayAlerts(alerts) {
    const alertsContainer = document.getElementById('alertsContainer');
    if (!alertsContainer) return;

    alertsContainer.innerHTML = alerts.map(alert => `
        <div class="alert alert-${getAlertSeverity(alert.level)} alert-dismissible fade show" role="alert">
            <strong>${alert.resource}:</strong> ${alert.message}
            <button type="button" class="btn-close" data-bs-dismiss="alert"></button>
        </div>
    `).join('');
}

function getAlertSeverity(level) {
    const severityMap = {
        'critical': 'danger',
        'high': 'danger',
        'medium': 'warning',
        'low': 'info'
    };
    return severityMap[level] || 'info';
}

async function loadTrendForecast(metric = 'requests') {
    try {
        const timeRange = getSelectedTimeRange();
        const response = await fetch(`${API_BASE}/ai-report/forecast?metric=${metric}&horizon=${timeRange.horizon}`);
        const forecast = await response.json();

        updateTrendChart(forecast);
        displayForecastDetails(forecast);
    } catch (error) {
        console.error('Failed to load trend forecast:', error);
        showToast('Failed to load trend forecast', 'error');
    }
}

function getSelectedTimeRange() {
    const select = document.getElementById('timeRangeSelect');
    const value = select ? select.value : '24h';

    const ranges = {
        '1h': { hours: 1, horizon: '1h' },
        '6h': { hours: 6, horizon: '6h' },
        '24h': { hours: 24, horizon: '24h' },
        '7d': { hours: 168, horizon: '7d' },
        '30d': { hours: 720, horizon: '30d' }
    };

    return ranges[value] || ranges['24h'];
}

function updateTrendChart(forecast) {
    if (!trendForecastChart) return;

    const labels = forecast.forecastPoints.map(p => formatTimestamp(p.timestamp));
    const actualData = forecast.forecastPoints.map(p => p.value);
    const upperBound = forecast.forecastPoints.map(p => p.upperBound);
    const lowerBound = forecast.forecastPoints.map(p => p.lowerBound);

    trendForecastChart.data.labels = labels;
    trendForecastChart.data.datasets[0].data = actualData.slice(0, Math.floor(actualData.length / 2));
    trendForecastChart.data.datasets[1].data = actualData.slice(Math.floor(actualData.length / 2));
    trendForecastChart.data.datasets[2].data = upperBound.slice(Math.floor(actualData.length / 2));
    trendForecastChart.data.datasets[3].data = lowerBound.slice(Math.floor(actualData.length / 2));

    trendForecastChart.update();
}

function displayForecastDetails(forecast) {
    const detailsContainer = document.getElementById('forecastDetails');
    if (!detailsContainer) return;

    detailsContainer.innerHTML = `
        <div class="row">
            <div class="col-md-3">
                <div class="stat-card">
                    <h6>Current Value</h6>
                    <p class="stat-value">${forecast.currentValue.toFixed(2)}</p>
                </div>
            </div>
            <div class="col-md-3">
                <div class="stat-card">
                    <h6>Predicted Value</h6>
                    <p class="stat-value">${forecast.predictedValue.toFixed(2)}</p>
                </div>
            </div>
            <div class="col-md-3">
                <div class="stat-card">
                    <h6>Change</h6>
                    <p class="stat-value ${forecast.changePercent > 0 ? 'text-success' : 'text-danger'}">
                        ${forecast.changePercent > 0 ? '+' : ''}${forecast.changePercent.toFixed(2)}%
                    </p>
                </div>
            </div>
            <div class="col-md-3">
                <div class="stat-card">
                    <h6>Trend</h6>
                    <p class="stat-value">
                        <span class="badge bg-${getTrendBadgeClass(forecast.trend)}">${forecast.trend}</span>
                    </p>
                </div>
            </div>
        </div>
        <div class="mt-3">
            <h6>Recommendations</h6>
            <ul class="list-group">
                ${forecast.recommendations.map(rec => `
                    <li class="list-group-item">
                        <i class="fas fa-lightbulb text-warning me-2"></i>
                        ${rec}
                    </li>
                `).join('')}
            </ul>
        </div>
    `;
}

function getTrendBadgeClass(trend) {
    const classes = {
        'increasing': 'success',
        'decreasing': 'danger',
        'stable': 'secondary',
        'fluctuating': 'warning'
    };
    return classes[trend] || 'secondary';
}

async function loadAnomalyDetection() {
    try {
        const response = await fetch(`${API_BASE}/ai-report/anomalies`);
        const anomalies = await response.json();

        updateAnomalyChart(anomalies);
        displayAnomalyList(anomalies);
    } catch (error) {
        console.error('Failed to load anomaly detection:', error);
        showToast('Failed to load anomaly detection', 'error');
    }
}

function updateAnomalyChart(anomalies) {
    if (!anomalyChart) return;

    const labels = anomalies.map(a => formatTimestamp(a.timestamp));
    const values = anomalies.map(a => a.value);
    const colors = anomalies.map(a => getAnomalyColor(a.severity));

    anomalyChart.data.labels = labels;
    anomalyChart.data.datasets[0].data = values;
    anomalyChart.data.datasets[0].backgroundColor = colors;

    anomalyChart.update();
}

function getAnomalyColor(severity) {
    const colors = {
        'critical': 'rgba(255, 99, 132, 0.8)',
        'high': 'rgba(255, 159, 64, 0.8)',
        'medium': 'rgba(255, 205, 86, 0.8)',
        'low': 'rgba(75, 192, 192, 0.8)'
    };
    return colors[severity] || colors['low'];
}

function displayAnomalyList(anomalies) {
    const listContainer = document.getElementById('anomalyList');
    if (!listContainer) return;

    if (anomalies.length === 0) {
        listContainer.innerHTML = '<p class="text-muted">No anomalies detected</p>';
        return;
    }

    listContainer.innerHTML = anomalies.map(anomaly => `
        <div class="card mb-2">
            <div class="card-body">
                <div class="d-flex justify-content-between align-items-center">
                    <div>
                        <span class="badge bg-${getAnomalyBadgeColor(anomaly.severity)}">${anomaly.severity}</span>
                        <strong>${anomaly.metric}</strong>
                    </div>
                    <small class="text-muted">${formatTimestamp(anomaly.timestamp)}</small>
                </div>
                <p class="mb-1 mt-2">${anomaly.description}</p>
                <div class="d-flex justify-content-between align-items-center">
                    <small class="text-muted">
                        Score: ${(anomaly.score * 100).toFixed(1)}% |
                        Confidence: ${(anomaly.confidence * 100).toFixed(1)}%
                    </small>
                    <button class="btn btn-sm btn-outline-primary" onclick="showAnomalyDetails('${anomaly.anomalyId}')">
                        View Details
                    </button>
                </div>
            </div>
        </div>
    `).join('');
}

function getAnomalyBadgeColor(severity) {
    const colors = {
        'critical': 'danger',
        'high': 'warning',
        'medium': 'info',
        'low': 'secondary'
    };
    return colors[severity] || 'secondary';
}

function showAnomalyDetails(anomalyId) {
    fetch(`${API_BASE}/ai-report/anomaly/${anomalyId}`)
        .then(response => response.json())
        .then(anomaly => {
            const modal = new bootstrap.Modal(document.getElementById('anomalyDetailsModal'));
            const content = document.getElementById('anomalyDetailsContent');

            content.innerHTML = `
                <div class="row">
                    <div class="col-md-6">
                        <h6>Basic Information</h6>
                        <table class="table table-sm">
                            <tr>
                                <td>Metric</td>
                                <td><strong>${anomaly.metric}</strong></td>
                            </tr>
                            <tr>
                                <td>Severity</td>
                                <td><span class="badge bg-${getAnomalyBadgeColor(anomaly.severity)}">${anomaly.severity}</span></td>
                            </tr>
                            <tr>
                                <td>Score</td>
                                <td>${(anomaly.score * 100).toFixed(1)}%</td>
                            </tr>
                            <tr>
                                <td>Confidence</td>
                                <td>${(anomaly.confidence * 100).toFixed(1)}%</td>
                            </tr>
                            <tr>
                                <td>Timestamp</td>
                                <td>${formatTimestamp(anomaly.timestamp)}</td>
                            </tr>
                        </table>
                    </div>
                    <div class="col-md-6">
                        <h6>Impact Assessment</h6>
                        <table class="table table-sm">
                            <tr>
                                <td>Level</td>
                                <td><span class="badge bg-${getImpactBadgeColor(anomaly.impact.level)}">${anomaly.impact.level}</span></td>
                            </tr>
                            <tr>
                                <td>Affected Users</td>
                                <td>${anomaly.impact.affectedUsers.toLocaleString()}</td>
                            </tr>
                            <tr>
                                <td>Affected Transactions</td>
                                <td>${anomaly.impact.affectedTransactions.toLocaleString()}</td>
                            </tr>
                            <tr>
                                <td>Estimated Downtime</td>
                                <td>${anomaly.impact.estimatedDowntime}</td>
                            </tr>
                            <tr>
                                <td>Financial Impact</td>
                                <td>$${anomaly.impact.financialImpact.toLocaleString()}</td>
                            </tr>
                        </table>
                    </div>
                </div>
                <div class="mt-3">
                    <h6>Root Cause</h6>
                    <p>${anomaly.rootCause}</p>
                </div>
                <div class="mt-3">
                    <h6>Recommendations</h6>
                    <ul class="list-group">
                        ${anomaly.recommendations.map(rec => `
                            <li class="list-group-item">
                                <i class="fas fa-check-circle text-success me-2"></i>
                                ${rec}
                            </li>
                        `).join('')}
                    </ul>
                </div>
                <div class="mt-3">
                    <h6>Correlated Events</h6>
                    <ul class="list-group">
                        ${anomaly.correlations.map(corr => `
                            <li class="list-group-item">
                                <div class="d-flex justify-content-between">
                                    <span>${corr.eventType}: ${corr.description}</span>
                                    <small class="text-muted">${(corr.correlation * 100).toFixed(0)}%</small>
                                </div>
                            </li>
                        `).join('')}
                    </ul>
                </div>
            `;

            modal.show();
        })
        .catch(error => {
            console.error('Failed to load anomaly details:', error);
            showToast('Failed to load anomaly details', 'error');
        });
}

function getImpactBadgeColor(level) {
    const colors = {
        'critical': 'danger',
        'high': 'warning',
        'medium': 'info',
        'low': 'secondary'
    };
    return colors[level] || 'secondary';
}

async function loadReportSummary() {
    try {
        const response = await fetch(`${API_BASE}/ai-report/summary?type=${currentReportType}`);
        const summary = await response.json();

        displayReportSummary(summary);
    } catch (error) {
        console.error('Failed to load report summary:', error);
        showToast('Failed to load report summary', 'error');
    }
}

function displayReportSummary(summary) {
    const summaryContainer = document.getElementById('reportSummary');
    if (!summaryContainer) return;

    summaryContainer.innerHTML = `
        <div class="card">
            <div class="card-header">
                <h5 class="mb-0">${summary.title}</h5>
            </div>
            <div class="card-body">
                <p class="lead">${summary.summary}</p>
                <hr>
                <div class="row">
                    ${summary.keyMetrics ? Object.entries(summary.keyMetrics).map(([key, value]) => `
                        <div class="col-md-3 mb-2">
                            <div class="border rounded p-2 text-center">
                                <h6>${formatMetricName(key)}</h6>
                                <p class="mb-0 fw-bold">${formatMetricValue(key, value)}</p>
                            </div>
                        </div>
                    `).join('') : ''}
                </div>
                <hr>
                <h6>Key Insights</h6>
                <div class="row">
                    ${summary.insights.map(insight => `
                        <div class="col-md-6 mb-2">
                            <div class="card ${getInsightCardClass(insight.type)}">
                                <div class="card-body">
                                    <h6>${insight.title}</h6>
                                    <p class="mb-0 small">${insight.description}</p>
                                    ${insight.value ? `<span class="badge bg-primary">${formatMetricValue(insight.metric, insight.value)}</span>` : ''}
                                </div>
                            </div>
                        </div>
                    `).join('')}
                </div>
            </div>
        </div>
    `;
}

function formatMetricName(name) {
    return name.split('_').map(word => word.charAt(0).toUpperCase() + word.slice(1)).join(' ');
}

function formatMetricValue(metric, value) {
    if (metric && metric.includes('rate')) {
        return `${(value * 100).toFixed(2)}%`;
    }
    if (metric && metric.includes('latency')) {
        return `${value.toFixed(2)}ms`;
    }
    if (typeof value === 'number') {
        return value.toLocaleString();
    }
    return value;
}

function getInsightCardClass(type) {
    const classes = {
        'positive': 'border-success',
        'negative': 'border-danger',
        'warning': 'border-warning',
        'info': 'border-info'
    };
    return classes[type] || 'border-secondary';
}

async function generateNLReport() {
    try {
        const response = await fetch(`${API_BASE}/ai-report/generate`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                reportType: currentReportType,
                timeRange: getSelectedTimeRange(),
                metrics: ['requests', 'success_rate', 'latency_p99', 'error_rate'],
                language: 'zh-CN',
                format: 'full'
            })
        });

        const report = await response.json();
        displayFullReport(report);
    } catch (error) {
        console.error('Failed to generate report:', error);
        showToast('Failed to generate report', 'error');
    }
}

function displayFullReport(report) {
    const reportContainer = document.getElementById('fullReportContainer');
    if (!reportContainer) return;

    reportContainer.innerHTML = `
        <div class="card">
            <div class="card-header d-flex justify-content-between align-items-center">
                <h5 class="mb-0">${report.title}</h5>
                <div>
                    <button class="btn btn-sm btn-outline-primary me-2" onclick="exportReport('pdf')">
                        <i class="fas fa-file-pdf"></i> Export PDF
                    </button>
                    <button class="btn btn-sm btn-outline-success" onclick="exportReport('csv')">
                        <i class="fas fa-file-csv"></i> Export CSV
                    </button>
                </div>
            </div>
            <div class="card-body">
                ${report.sections.map(section => `
                    <div class="mb-4">
                        <h5>${section.title}</h5>
                        <p>${section.content}</p>
                    </div>
                `).join('')}

                ${report.charts && report.charts.length > 0 ? `
                    <hr>
                    <h5>Visualizations</h5>
                    <div class="row">
                        ${report.charts.map(chart => `
                            <div class="col-md-6 mb-3">
                                <div class="card">
                                    <div class="card-body">
                                        <h6>${chart.title}</h6>
                                        <canvas id="chart_${chart.id}"></canvas>
                                    </div>
                                </div>
                            </div>
                        `).join('')}
                    </div>
                ` : ''}

                ${report.tables && report.tables.length > 0 ? `
                    <hr>
                    <h5>Data Tables</h5>
                    ${report.tables.map(table => `
                        <div class="mb-3">
                            <h6>${table.title}</h6>
                            <table class="table table-striped">
                                <thead>
                                    <tr>
                                        ${table.headers.map(h => `<th>${h}</th>`).join('')}
                                    </tr>
                                </thead>
                                <tbody>
                                    ${table.rows.map(row => `
                                        <tr>
                                            ${row.map(cell => `<td>${cell}</td>`).join('')}
                                        </tr>
                                    `).join('')}
                                </tbody>
                            </table>
                        </div>
                    `).join('')}
                ` : ''}

                ${report.comparisons && report.comparisons.length > 0 ? `
                    <hr>
                    <h5>Period Comparisons</h5>
                    <table class="table">
                        <thead>
                            <tr>
                                <th>Metric</th>
                                <th>Current</th>
                                <th>Previous</th>
                                <th>Change</th>
                                <th>Trend</th>
                            </tr>
                        </thead>
                        <tbody>
                            ${report.comparisons.map(comp => `
                                <tr>
                                    <td>${comp.metric}</td>
                                    <td>${formatMetricValue(comp.metric, comp.current)}</td>
                                    <td>${formatMetricValue(comp.metric, comp.previous)}</td>
                                    <td class="${comp.changePercent > 0 ? 'text-success' : 'text-danger'}">
                                        ${comp.changePercent > 0 ? '+' : ''}${comp.changePercent.toFixed(2)}%
                                    </td>
                                    <td>
                                        <i class="fas fa-arrow-${comp.trend === 'up' ? 'up text-success' : 'down text-danger'}"></i>
                                    </td>
                                </tr>
                            `).join('')}
                        </tbody>
                    </table>
                ` : ''}
            </div>
            <div class="card-footer text-muted">
                Generated at: ${formatTimestamp(report.generatedAt)} | Model: ${report.modelVersion}
            </div>
        </div>
    `;

    report.charts.forEach(chart => {
        const ctx = document.getElementById(`chart_${chart.id}`);
        if (ctx) {
            renderChart(ctx, chart);
        }
    });
}

function renderChart(canvas, chartConfig) {
    new Chart(canvas, {
        type: chartConfig.type,
        data: chartConfig.data,
        options: {
            responsive: true,
            plugins: {
                legend: {
                    display: true
                }
            }
        }
    });
}

async function processNLQuery() {
    const queryInput = document.getElementById('nlQueryInput');
    const query = queryInput.value.trim();

    if (!query) {
        showToast('Please enter a query', 'warning');
        return;
    }

    try {
        const response = await fetch(`${API_BASE}/ai-report/query`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                query: query,
                timeRange: getSelectedTimeRange()
            })
        });

        const result = await response.json();
        displayQueryResult(result);
    } catch (error) {
        console.error('Failed to process query:', error);
        showToast('Failed to process query', 'error');
    }
}

function displayQueryResult(result) {
    const resultsContainer = document.getElementById('queryResults');
    if (!resultsContainer) return;

    resultsContainer.innerHTML = `
        <div class="card">
            <div class="card-header">
                <h6 class="mb-0">Query Results</h6>
            </div>
            <div class="card-body">
                <div class="mb-3">
                    <strong>Explanation:</strong>
                    <p>${result.explanation}</p>
                </div>

                ${result.results ? `
                    <div class="mb-3">
                        <strong>Results:</strong>
                        <pre class="bg-light p-3 rounded"><code>${JSON.stringify(result.results, null, 2)}</code></pre>
                    </div>
                ` : ''}

                ${result.visualizations && result.visualizations.length > 0 ? `
                    <div class="mb-3">
                        <strong>Visualizations:</strong>
                        ${result.visualizations.map(viz => `
                            <canvas id="viz_${result.queryId}_${viz.id}"></canvas>
                        `).join('')}
                    </div>
                ` : ''}

                ${result.relatedQueries && result.relatedQueries.length > 0 ? `
                    <div>
                        <strong>Related Queries:</strong>
                        <div class="mt-2">
                            ${result.relatedQueries.map(q => `
                                <button class="btn btn-sm btn-outline-secondary me-2 mb-2" onclick="executeRelatedQuery('${q}')">
                                    ${q}
                                </button>
                            `).join('')}
                        </div>
                    </div>
                ` : ''}
            </div>
        </div>
    `;

    if (result.visualizations) {
        result.visualizations.forEach(viz => {
            const canvas = document.getElementById(`viz_${result.queryId}_${viz.id}`);
            if (canvas) {
                renderChart(canvas, viz);
            }
        });
    }
}

function executeRelatedQuery(query) {
    const queryInput = document.getElementById('nlQueryInput');
    queryInput.value = query;
    processNLQuery();
}

async function exportReport(format = 'json') {
    try {
        const response = await fetch(`${API_BASE}/ai-report/export?format=${format}`);
        const blob = await response.blob();

        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `ai-report-${Date.now()}.${format}`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        window.URL.revokeObjectURL(url);

        showToast(`Report exported as ${format.toUpperCase()}`, 'success');
    } catch (error) {
        console.error('Failed to export report:', error);
        showToast('Failed to export report', 'error');
    }
}

function formatTimestamp(timestamp) {
    const date = new Date(timestamp);
    return date.toLocaleString();
}

function showToast(message, type = 'info') {
    const toastContainer = document.getElementById('toastContainer') || createToastContainer();

    const toast = document.createElement('div');
    toast.className = `toast align-items-center text-white bg-${type === 'error' ? 'danger' : type} border-0`;
    toast.setAttribute('role', 'alert');
    toast.innerHTML = `
        <div class="d-flex">
            <div class="toast-body">
                ${message}
            </div>
            <button type="button" class="btn-close btn-close-white me-2 m-auto" data-bs-dismiss="toast"></button>
        </div>
    `;

    toastContainer.appendChild(toast);
    const bsToast = new bootstrap.Toast(toast);
    bsToast.show();

    toast.addEventListener('hidden.bs.toast', () => {
        toast.remove();
    });
}

function createToastContainer() {
    const container = document.createElement('div');
    container.id = 'toastContainer';
    container.className = 'toast-container position-fixed top-0 end-0 p-3';
    document.body.appendChild(container);
    return container;
}
