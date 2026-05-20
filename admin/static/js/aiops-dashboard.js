let healthScoreGauge;
let anomalyDetectionChart;
let faultLocalizationChart;
let predictiveMaintenanceChart;
let knowledgeGraphVisualization;

let refreshTimer = null;
let isAutoRefresh = false;
let currentTab = 'overview';

const API_BASE = '/api/v1/aiops';
const AUTO_REFRESH_INTERVAL = 30000;

document.addEventListener('DOMContentLoaded', () => {
    initializeAIOpsDashboard();
    setupEventListeners();
    loadDashboardData();
});

function initializeAIOpsDashboard() {
    initializeHealthScoreGauge();
    initializeAnomalyChart();
    initializeFaultChart();
    initializeMaintenanceChart();
    initializeKnowledgeGraph();
}

function initializeHealthScoreGauge() {
    const ctx = document.getElementById('healthScoreGauge');
    if (!ctx) return;

    healthScoreGauge = new Chart(ctx, {
        type: 'doughnut',
        data: {
            labels: ['Healthy', 'Issues'],
            datasets: [{
                data: [100, 0],
                backgroundColor: [
                    'rgba(75, 192, 192, 0.8)',
                    'rgba(255, 99, 132, 0.8)'
                ],
                borderWidth: 0
            }]
        },
        options: {
            circumference: 180,
            rotation: 270,
            responsive: true,
            maintainAspectRatio: false,
            cutout: '70%',
            plugins: {
                legend: {
                    display: false
                },
                tooltip: {
                    enabled: false
                }
            }
        }
    });
}

function initializeAnomalyChart() {
    const ctx = document.getElementById('anomalyDetectionChart');
    if (!ctx) return;

    anomalyDetectionChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: [],
            datasets: [{
                label: 'Anomaly Score',
                data: [],
                borderColor: 'rgb(255, 99, 132)',
                backgroundColor: 'rgba(255, 99, 132, 0.2)',
                tension: 0.1,
                fill: true
            }, {
                label: 'Threshold',
                data: [],
                borderColor: 'rgba(255, 159, 64, 0.8)',
                borderDash: [5, 5],
                pointRadius: 0,
                fill: false
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                y: {
                    beginAtZero: true,
                    max: 1
                }
            },
            onClick: (event, elements) => {
                if (elements.length > 0) {
                    const index = elements[0].index;
                    showAnomalyDetails(index);
                }
            }
        }
    });
}

function initializeFaultChart() {
    const ctx = document.getElementById('faultLocalizationChart');
    if (!ctx) return;

    faultLocalizationChart = new Chart(ctx, {
        type: 'bar',
        data: {
            labels: [],
            datasets: [{
                label: 'Probability',
                data: [],
                backgroundColor: []
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                y: {
                    beginAtZero: true,
                    max: 1
                }
            },
            plugins: {
                legend: {
                    display: false
                }
            }
        }
    });
}

function initializeMaintenanceChart() {
    const ctx = document.getElementById('predictiveMaintenanceChart');
    if (!ctx) return;

    predictiveMaintenanceChart = new Chart(ctx, {
        type: 'radar',
        data: {
            labels: ['CPU', 'Memory', 'Disk', 'Network', 'Latency', 'Errors'],
            datasets: [{
                label: 'Current',
                data: [0, 0, 0, 0, 0, 0],
                borderColor: 'rgb(75, 192, 192)',
                backgroundColor: 'rgba(75, 192, 192, 0.2)'
            }, {
                label: 'Threshold',
                data: [0.8, 0.85, 0.9, 0.7, 0.6, 0.5],
                borderColor: 'rgba(255, 99, 132, 0.8)',
                backgroundColor: 'rgba(255, 99, 132, 0.1)'
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                r: {
                    beginAtZero: true,
                    max: 1
                }
            }
        }
    });
}

function initializeKnowledgeGraph() {
    const container = document.getElementById('knowledgeGraphContainer');
    if (!container) return;

    container.innerHTML = `
        <div id="kg-canvas" style="width: 100%; height: 400px; background: #f8f9fa; border-radius: 8px;">
            <div class="text-center text-muted p-5">
                <i class="fas fa-project-diagram fa-3x mb-3"></i>
                <p>Knowledge Graph Visualization</p>
                <p class="small">Click on nodes to explore relationships</p>
            </div>
        </div>
    `;
}

function setupEventListeners() {
    document.querySelectorAll('.aiops-nav-link').forEach(link => {
        link.addEventListener('click', (e) => {
            e.preventDefault();
            const tabId = e.target.getAttribute('data-tab');
            switchTab(tabId);
        });
    });

    const refreshBtn = document.getElementById('refreshDashboardBtn');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', loadDashboardData);
    }

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

    const detectAnomalyBtn = document.getElementById('detectAnomalyBtn');
    if (detectAnomalyBtn) {
        detectAnomalyBtn.addEventListener('click', runAnomalyDetection);
    }

    const localizeFaultBtn = document.getElementById('localizeFaultBtn');
    if (localizeFaultBtn) {
        localizeFaultBtn.addEventListener('click', runFaultLocalization);
    }

    const predictMaintenanceBtn = document.getElementById('predictMaintenanceBtn');
    if (predictMaintenanceBtn) {
        predictMaintenanceBtn.addEventListener('click', runPredictiveMaintenance);
    }

    const queryKnowledgeGraphBtn = document.getElementById('queryKnowledgeGraphBtn');
    if (queryKnowledgeGraphBtn) {
        queryKnowledgeGraphBtn.addEventListener('click', queryKnowledgeGraph);
    }

    const metricSelect = document.getElementById('metricSelect');
    if (metricSelect) {
        metricSelect.addEventListener('change', (e) => {
            loadMetricAnomalies(e.target.value);
        });
    }

    document.querySelectorAll('.severity-filter').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const severity = e.target.dataset.severity;
            filterAnomaliesBySeverity(severity);
        });
    });
}

function switchTab(tabId) {
    currentTab = tabId;

    document.querySelectorAll('.aiops-nav-link').forEach(link => {
        link.classList.remove('active');
    });
    document.querySelector(`[data-tab="${tabId}"]`)?.classList.add('active');

    document.querySelectorAll('.tab-content').forEach(content => {
        content.classList.remove('show', 'active');
    });
    document.getElementById(`${tabId}Tab`)?.classList.add('show', 'active');

    switch (tabId) {
        case 'overview':
            loadDashboardData();
            break;
        case 'anomalies':
            loadAnomaliesList();
            break;
        case 'faults':
            loadFaultsList();
            break;
        case 'maintenance':
            loadMaintenancePredictions();
            break;
        case 'knowledge':
            loadKnowledgeGraph();
            break;
        case 'incidents':
            loadIncidents();
            break;
    }
}

function startAutoRefresh() {
    if (refreshTimer) {
        clearInterval(refreshTimer);
    }
    refreshTimer = setInterval(loadDashboardData, AUTO_REFRESH_INTERVAL);
    showToast('Auto-refresh enabled', 'info');
}

function stopAutoRefresh() {
    if (refreshTimer) {
        clearInterval(refreshTimer);
        refreshTimer = null;
    }
    showToast('Auto-refresh disabled', 'info');
}

async function loadDashboardData() {
    try {
        const response = await fetch(`${API_BASE}/dashboard`);
        const dashboard = await response.json();

        updateHealthScore(dashboard.healthScore);
        updateDashboardStats(dashboard);
        updateAnomalyChart(dashboard.recentAnomalies);
        updateMaintenanceChart(dashboard.maintenanceIndicators);
    } catch (error) {
        console.error('Failed to load dashboard data:', error);
        showToast('Failed to load dashboard data', 'error');
    }
}

function updateHealthScore(score) {
    if (healthScoreGauge) {
        healthScoreGauge.data.datasets[0].data = [score, 100 - score];
        healthScoreGauge.update();
    }

    const scoreElement = document.getElementById('healthScoreValue');
    if (scoreElement) {
        scoreElement.textContent = score.toFixed(1);
        scoreElement.className = `display-4 fw-bold ${getHealthScoreColor(score)}`;
    }
}

function getHealthScoreColor(score) {
    if (score >= 90) return 'text-success';
    if (score >= 70) return 'text-warning';
    return 'text-danger';
}

function updateDashboardStats(dashboard) {
    updateStatCard('activeIncidents', dashboard.activeIncidents);
    updateStatCard('anomaliesDetected', dashboard.anomaliesDetected);
    updateStatCard('predictions', dashboard.predictions);
    updateStatCard('resolvedToday', dashboard.resolvedToday);

    if (dashboard.topIssues && dashboard.topIssues.length > 0) {
        displayTopIssues(dashboard.topIssues);
    }

    if (dashboard.recentEvents && dashboard.recentEvents.length > 0) {
        displayRecentEvents(dashboard.recentEvents);
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

function displayTopIssues(issues) {
    const container = document.getElementById('topIssuesList');
    if (!container) return;

    container.innerHTML = issues.map(issue => `
        <div class="list-group-item d-flex justify-content-between align-items-center">
            <div>
                <h6 class="mb-0">${issue.title}</h6>
                <small class="text-muted">${issue.affectedCount} affected</small>
            </div>
            <div>
                <span class="badge bg-${getPriorityColor(issue.priority)}">${issue.priority}</span>
                <span class="badge bg-${getStatusColor(issue.status)}">${issue.status}</span>
            </div>
        </div>
    `).join('');
}

function getPriorityColor(priority) {
    const colors = {
        'critical': 'danger',
        'high': 'warning',
        'medium': 'info',
        'low': 'secondary'
    };
    return colors[priority] || 'secondary';
}

function getStatusColor(status) {
    const colors = {
        'investigating': 'warning',
        'identified': 'info',
        'monitoring': 'primary',
        'resolved': 'success'
    };
    return colors[status] || 'secondary';
}

function displayRecentEvents(events) {
    const container = document.getElementById('recentEventsList');
    if (!container) return;

    container.innerHTML = events.map(event => `
        <div class="list-group-item">
            <div class="d-flex w-100 justify-content-between">
                <h6 class="mb-1">
                    <span class="badge bg-${getEventTypeColor(event.type)} me-2">${event.type}</span>
                    ${event.description}
                </h6>
                <small class="text-muted">${formatTimestamp(event.timestamp)}</small>
            </div>
        </div>
    `).join('');
}

function getEventTypeColor(type) {
    const colors = {
        'incident': 'danger',
        'alert': 'warning',
        'maintenance': 'info',
        'change': 'primary'
    };
    return colors[type] || 'secondary';
}

function updateAnomalyChart(anomalies) {
    if (!anomalyDetectionChart) return;

    const labels = anomalies.map(a => formatTimestamp(a.timestamp));
    const scores = anomalies.map(a => a.score);
    const threshold = anomalies.map(() => 0.7);

    anomalyDetectionChart.data.labels = labels;
    anomalyDetectionChart.data.datasets[0].data = scores;
    anomalyDetectionChart.data.datasets[1].data = threshold;
    anomalyDetectionChart.update();
}

function updateMaintenanceChart(indicators) {
    if (!predictiveMaintenanceChart || !indicators) return;

    const values = [
        indicators.cpu_usage / 100,
        indicators.memory_usage / 100,
        indicators.disk_usage / 100,
        indicators.network_usage / 100,
        indicators.latency / 1000,
        indicators.error_rate * 10
    ];

    predictiveMaintenanceChart.data.datasets[0].data = values;
    predictiveMaintenanceChart.update();
}

async function loadAnomaliesList() {
    try {
        const response = await fetch(`${API_BASE}/anomalies`);
        const anomalies = await response.json();

        displayAnomaliesTable(anomalies);
    } catch (error) {
        console.error('Failed to load anomalies:', error);
        showToast('Failed to load anomalies', 'error');
    }
}

function displayAnomaliesTable(anomalies) {
    const container = document.getElementById('anomaliesTableContainer');
    if (!container) return;

    container.innerHTML = `
        <table class="table table-hover">
            <thead>
                <tr>
                    <th>ID</th>
                    <th>Metric</th>
                    <th>Severity</th>
                    <th>Score</th>
                    <th>Root Cause</th>
                    <th>Timestamp</th>
                    <th>Actions</th>
                </tr>
            </thead>
            <tbody>
                ${anomalies.map(anomaly => `
                    <tr>
                        <td>${anomaly.anomalyId}</td>
                        <td><strong>${anomaly.metric}</strong></td>
                        <td><span class="badge bg-${getSeverityColor(anomaly.severity)}">${anomaly.severity}</span></td>
                        <td>
                            <div class="progress" style="width: 100px;">
                                <div class="progress-bar bg-${getSeverityColor(anomaly.severity)}" 
                                     style="width: ${anomaly.score * 100}%"></div>
                            </div>
                            ${(anomaly.score * 100).toFixed(0)}%
                        </td>
                        <td>${anomaly.rootCause}</td>
                        <td>${formatTimestamp(anomaly.timestamp)}</td>
                        <td>
                            <button class="btn btn-sm btn-outline-primary" onclick="showAnomalyDetails('${anomaly.anomalyId}')">
                                Details
                            </button>
                        </td>
                    </tr>
                `).join('')}
            </tbody>
        </table>
    `;
}

function getSeverityColor(severity) {
    const colors = {
        'critical': 'danger',
        'high': 'warning',
        'medium': 'info',
        'low': 'secondary'
    };
    return colors[severity] || 'secondary';
}

async function runAnomalyDetection() {
    const metric = document.getElementById('metricSelect')?.value || 'requests';

    try {
        showToast('Running anomaly detection...', 'info');

        const response = await fetch(`${API_BASE}/detect`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ metric })
        });

        const result = await response.json();
        showToast('Anomaly detection completed', 'success');
        displayAnomalyResult(result);
        loadAnomaliesList();
    } catch (error) {
        console.error('Anomaly detection failed:', error);
        showToast('Anomaly detection failed', 'error');
    }
}

function displayAnomalyResult(result) {
    const container = document.getElementById('anomalyResultContainer');
    if (!container) return;

    container.innerHTML = `
        <div class="card">
            <div class="card-header d-flex justify-content-between align-items-center">
                <h6 class="mb-0">Detection Result</h6>
                <span class="badge bg-${getSeverityColor(result.severity)}">${result.severity}</span>
            </div>
            <div class="card-body">
                <div class="row">
                    <div class="col-md-6">
                        <p><strong>Metric:</strong> ${result.metric}</p>
                        <p><strong>Score:</strong> ${(result.score * 100).toFixed(1)}%</p>
                        <p><strong>Confidence:</strong> ${(result.confidence * 100).toFixed(1)}%</p>
                    </div>
                    <div class="col-md-6">
                        <p><strong>Description:</strong></p>
                        <p>${result.description}</p>
                    </div>
                </div>
                <hr>
                <h6>Root Cause</h6>
                <p>${result.rootCause}</p>
                <hr>
                <h6>Recommendations</h6>
                <ul class="list-group">
                    ${result.recommendations.map(rec => `
                        <li class="list-group-item">
                            <i class="fas fa-lightbulb text-warning me-2"></i>
                            ${rec}
                        </li>
                    `).join('')}
                </ul>
            </div>
        </div>
    `;
}

async function showAnomalyDetails(anomalyId) {
    try {
        const response = await fetch(`${API_BASE}/anomaly/${anomalyId}`);
        const anomaly = await response.json();

        displayAnomalyDetailsModal(anomaly);
    } catch (error) {
        console.error('Failed to load anomaly details:', error);
        showToast('Failed to load anomaly details', 'error');
    }
}

function displayAnomalyDetailsModal(anomaly) {
    const modal = new bootstrap.Modal(document.getElementById('anomalyDetailsModal'));
    const content = document.getElementById('anomalyDetailsContent');

    content.innerHTML = `
        <div class="row">
            <div class="col-md-6">
                <h6>Basic Information</h6>
                <table class="table table-sm">
                    <tr><td>ID</td><td><strong>${anomaly.anomalyId}</strong></td></tr>
                    <tr><td>Metric</td><td>${anomaly.metric}</td></tr>
                    <tr><td>Severity</td><td><span class="badge bg-${getSeverityColor(anomaly.severity)}">${anomaly.severity}</span></td></tr>
                    <tr><td>Score</td><td>${(anomaly.score * 100).toFixed(1)}%</td></tr>
                    <tr><td>Confidence</td><td>${(anomaly.confidence * 100).toFixed(1)}%</td></tr>
                </table>
            </div>
            <div class="col-md-6">
                <h6>Impact Assessment</h6>
                <table class="table table-sm">
                    <tr><td>Level</td><td><span class="badge bg-${getSeverityColor(anomaly.impact.level)}">${anomaly.impact.level}</span></td></tr>
                    <tr><td>Score</td><td>${(anomaly.impact.score * 100).toFixed(0)}%</td></tr>
                    <tr><td>Affected Users</td><td>${anomaly.impact.affectedUsers.toLocaleString()}</td></tr>
                    <tr><td>Downtime</td><td>${anomaly.impact.estimatedDowntime}</td></tr>
                </table>
            </div>
        </div>
        <div class="mt-3">
            <h6>Description</h6>
            <p>${anomaly.description}</p>
        </div>
        <div class="mt-3">
            <h6>Root Cause</h6>
            <p>${anomaly.rootCause}</p>
        </div>
        <div class="mt-3">
            <h6>Affected Entities</h6>
            <div class="d-flex flex-wrap gap-2">
                ${anomaly.affectedEntities.map(entity => `
                    <span class="badge bg-secondary">${entity}</span>
                `).join('')}
            </div>
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
            <h6>Correlations</h6>
            <ul class="list-group">
                ${anomaly.correlations.map(corr => `
                    <li class="list-group-item">
                        <div class="d-flex justify-content-between">
                            <span><strong>${corr.eventType}</strong>: ${corr.description}</span>
                            <span class="badge bg-info">${(corr.correlation * 100).toFixed(0)}%</span>
                        </div>
                    </li>
                `).join('')}
            </ul>
        </div>
    `;

    modal.show();
}

async function loadFaultsList() {
    try {
        const response = await fetch(`${API_BASE}/faults`);
        const faults = await response.json();

        displayFaultsTable(faults);
    } catch (error) {
        console.error('Failed to load faults:', error);
        showToast('Failed to load faults', 'error');
    }
}

function displayFaultsTable(faults) {
    const container = document.getElementById('faultsTableContainer');
    if (!container) return;

    container.innerHTML = `
        <table class="table table-hover">
            <thead>
                <tr>
                    <th>ID</th>
                    <th>Root Cause</th>
                    <th>Confidence</th>
                    <th>Propagation Path</th>
                    <th>Estimated Resolution</th>
                    <th>Actions</th>
                </tr>
            </thead>
            <tbody>
                ${faults.map(fault => `
                    <tr>
                        <td>${fault.faultId}</td>
                        <td>${fault.rootCause?.component || 'Unknown'}</td>
                        <td>
                            <div class="progress" style="width: 100px;">
                                <div class="progress-bar bg-success" style="width: ${fault.confidence * 100}%"></div>
                            </div>
                            ${(fault.confidence * 100).toFixed(0)}%
                        </td>
                        <td>
                            <small>${fault.propagationPath?.join(' → ') || 'N/A'}</small>
                        </td>
                        <td>${fault.estimatedResolutionTime}</td>
                        <td>
                            <button class="btn btn-sm btn-outline-primary" onclick="showFaultDetails('${fault.faultId}')">
                                Details
                            </button>
                        </td>
                    </tr>
                `).join('')}
            </tbody>
        </table>
    `;
}

async function runFaultLocalization() {
    try {
        showToast('Running fault localization...', 'info');

        const response = await fetch(`${API_BASE}/localize`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                symptoms: [
                    { type: 'latency', description: 'High API latency', severity: 'high', entity: 'api-gateway' },
                    { type: 'errors', description: 'Increased error rate', severity: 'critical', entity: 'core-api' }
                ]
            })
        });

        const result = await response.json();
        showToast('Fault localization completed', 'success');
        displayFaultResult(result);
        loadFaultsList();
    } catch (error) {
        console.error('Fault localization failed:', error);
        showToast('Fault localization failed', 'error');
    }
}

function displayFaultResult(result) {
    const container = document.getElementById('faultResultContainer');
    if (!container) return;

    container.innerHTML = `
        <div class="card">
            <div class="card-header">
                <h6 class="mb-0">Fault Localization Result</h6>
            </div>
            <div class="card-body">
                <div class="row">
                    <div class="col-md-6">
                        <h6>Root Cause</h6>
                        ${result.rootCause ? `
                            <p><strong>Component:</strong> ${result.rootCause.component}</p>
                            <p><strong>Probability:</strong> ${(result.rootCause.probability * 100).toFixed(1)}%</p>
                            <h6>Evidence</h6>
                            <ul class="list-group">
                                ${result.rootCause.evidence.map(ev => `
                                    <li class="list-group-item">
                                        <strong>${ev.type}</strong>: ${ev.description}
                                        <br><small class="text-muted">Weight: ${(ev.weight * 100).toFixed(0)}%</small>
                                    </li>
                                `).join('')}
                            </ul>
                        ` : '<p class="text-muted">No root cause identified</p>'}
                    </div>
                    <div class="col-md-6">
                        <h6>Propagation Path</h6>
                        <div class="d-flex align-items-center flex-wrap">
                            ${result.propagationPath.map((node, i) => `
                                <span class="badge bg-primary me-2">${node}</span>
                                ${i < result.propagationPath.length - 1 ? '<i class="fas fa-arrow-right me-2"></i>' : ''}
                            `).join('')}
                        </div>
                        <hr>
                        <h6>Candidates</h6>
                        <ul class="list-group">
                            ${result.candidates.map(cand => `
                                <li class="list-group-item d-flex justify-content-between align-items-center">
                                    ${cand.component}
                                    <span class="badge bg-${cand.probability > 0.5 ? 'success' : 'secondary'}">
                                        ${(cand.probability * 100).toFixed(0)}%
                                    </span>
                                </li>
                            `).join('')}
                        </ul>
                    </div>
                </div>
                <hr>
                <h6>Recommended Actions</h6>
                <ul class="list-group">
                    ${result.recommendedActions.map(action => `
                        <li class="list-group-item">
                            <i class="fas fa-wrench text-primary me-2"></i>
                            ${action}
                        </li>
                    `).join('')}
                </ul>
            </div>
        </div>
    `;

    updateFaultChart(result.candidates);
}

function updateFaultChart(candidates) {
    if (!faultLocalizationChart) return;

    const labels = candidates.map(c => c.component);
    const probabilities = candidates.map(c => c.probability);
    const colors = probabilities.map(p => p > 0.5 ? 'rgba(75, 192, 192, 0.8)' : 'rgba(255, 159, 64, 0.8)');

    faultLocalizationChart.data.labels = labels;
    faultLocalizationChart.data.datasets[0].data = probabilities;
    faultLocalizationChart.data.datasets[0].backgroundColor = colors;
    faultLocalizationChart.update();
}

async function showFaultDetails(faultId) {
    try {
        const response = await fetch(`${API_BASE}/fault/${faultId}`);
        const fault = await response.json();

        displayFaultDetailsModal(fault);
    } catch (error) {
        console.error('Failed to load fault details:', error);
        showToast('Failed to load fault details', 'error');
    }
}

function displayFaultDetailsModal(fault) {
    const modal = new bootstrap.Modal(document.getElementById('faultDetailsModal'));
    const content = document.getElementById('faultDetailsContent');

    content.innerHTML = `
        <div class="row">
            <div class="col-md-12">
                <h6>Fault ID</h6>
                <p>${fault.faultId}</p>
            </div>
        </div>
        <div class="row mt-3">
            <div class="col-md-6">
                <h6>Symptoms</h6>
                <ul class="list-group">
                    ${fault.symptoms.map(sym => `
                        <li class="list-group-item">
                            <strong>${sym.type}</strong>: ${sym.description}
                            <br><small class="text-muted">${sym.entity} - ${sym.severity}</small>
                        </li>
                    `).join('')}
                </ul>
            </div>
            <div class="col-md-6">
                <h6>Fault Candidates</h6>
                <ul class="list-group">
                    ${fault.candidates.map(cand => `
                        <li class="list-group-item d-flex justify-content-between align-items-center ${cand.excluded ? 'opacity-50' : ''}">
                            <div>
                                <strong>${cand.component}</strong>
                                ${cand.excluded ? `<span class="badge bg-secondary ms-2">Excluded</span>` : ''}
                                ${cand.exclusionReason ? `<br><small class="text-muted">${cand.exclusionReason}</small>` : ''}
                            </div>
                            <span class="badge bg-primary">${(cand.probability * 100).toFixed(0)}%</span>
                        </li>
                    `).join('')}
                </ul>
            </div>
        </div>
    `;

    modal.show();
}

async function loadMaintenancePredictions() {
    try {
        const response = await fetch(`${API_BASE}/maintenance`);
        const predictions = await response.json();

        displayMaintenanceTable(predictions);
    } catch (error) {
        console.error('Failed to load maintenance predictions:', error);
        showToast('Failed to load maintenance predictions', 'error');
    }
}

function displayMaintenanceTable(predictions) {
    const container = document.getElementById('maintenanceTableContainer');
    if (!container) return;

    container.innerHTML = `
        <table class="table table-hover">
            <thead>
                <tr>
                    <th>Prediction ID</th>
                    <th>Component</th>
                    <th>Type</th>
                    <th>Risk Level</th>
                    <th>Time to Failure</th>
                    <th>Probability</th>
                    <th>Actions</th>
                </tr>
            </thead>
            <tbody>
                ${predictions.map(pred => `
                    <tr>
                        <td>${pred.predictionId}</td>
                        <td><strong>${pred.component}</strong></td>
                        <td>${pred.predictionType}</td>
                        <td><span class="badge bg-${getRiskLevelColor(pred.riskLevel)}">${pred.riskLevel}</span></td>
                        <td>${pred.timeToFailure}</td>
                        <td>
                            <div class="progress" style="width: 100px;">
                                <div class="progress-bar bg-${getRiskLevelColor(pred.riskLevel)}" style="width: ${pred.probability * 100}%"></div>
                            </div>
                            ${(pred.probability * 100).toFixed(0)}%
                        </td>
                        <td>
                            <button class="btn btn-sm btn-outline-primary" onclick="showMaintenanceDetails('${pred.predictionId}')">
                                Details
                            </button>
                        </td>
                    </tr>
                `).join('')}
            </tbody>
        </table>
    `;
}

function getRiskLevelColor(level) {
    const colors = {
        'critical': 'danger',
        'high': 'warning',
        'medium': 'info',
        'low': 'secondary'
    };
    return colors[level] || 'secondary';
}

async function runPredictiveMaintenance() {
    const component = document.getElementById('componentSelect')?.value || 'database';

    try {
        showToast('Running predictive maintenance analysis...', 'info');

        const response = await fetch(`${API_BASE}/predict`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ component })
        });

        const result = await response.json();
        showToast('Prediction completed', 'success');
        displayMaintenanceResult(result);
        loadMaintenancePredictions();
    } catch (error) {
        console.error('Predictive maintenance failed:', error);
        showToast('Predictive maintenance failed', 'error');
    }
}

function displayMaintenanceResult(result) {
    const container = document.getElementById('maintenanceResultContainer');
    if (!container) return;

    container.innerHTML = `
        <div class="card">
            <div class="card-header d-flex justify-content-between align-items-center">
                <h6 class="mb-0">Prediction Result</h6>
                <span class="badge bg-${getRiskLevelColor(result.riskLevel)}">${result.riskLevel}</span>
            </div>
            <div class="card-body">
                <div class="row">
                    <div class="col-md-4">
                        <div class="text-center">
                            <h6>Probability</h6>
                            <p class="display-4 fw-bold text-${getRiskLevelColor(result.riskLevel)}">
                                ${(result.probability * 100).toFixed(0)}%
                            </p>
                        </div>
                    </div>
                    <div class="col-md-4">
                        <div class="text-center">
                            <h6>Time to Failure</h6>
                            <p class="display-6 fw-bold">${result.timeToFailure}</p>
                        </div>
                    </div>
                    <div class="col-md-4">
                        <div class="text-center">
                            <h6>Confidence</h6>
                            <p class="display-6 fw-bold">${(result.confidence * 100).toFixed(0)}%</p>
                        </div>
                    </div>
                </div>
                <hr>
                <h6>Maintenance Indicators</h6>
                <div class="row">
                    ${result.indicators.map(ind => `
                        <div class="col-md-3 mb-2">
                            <div class="card ${ind.status === 'warning' ? 'border-warning' : ind.status === 'critical' ? 'border-danger' : ''}">
                                <div class="card-body text-center">
                                    <h6>${ind.name}</h6>
                                    <p class="mb-0">
                                        <strong>${ind.value}</strong> ${ind.unit}
                                    </p>
                                    <small class="text-muted">Threshold: ${ind.threshold}</small>
                                    <br>
                                    <span class="badge bg-${ind.status === 'normal' ? 'success' : ind.status === 'warning' ? 'warning' : 'danger'}">
                                        ${ind.status}
                                    </span>
                                </div>
                            </div>
                        </div>
                    `).join('')}
                </div>
                <hr>
                <h6>Recommended Actions</h6>
                <ul class="list-group">
                    ${result.recommendedActions.map(action => `
                        <li class="list-group-item">
                            <i class="fas fa-tools text-primary me-2"></i>
                            ${action}
                        </li>
                    `).join('')}
                </ul>
                <hr>
                <h6>Suggested Maintenance Window</h6>
                <div class="alert alert-info">
                    <p><strong>Earliest Start:</strong> ${formatTimestamp(result.maintenanceWindow.earliestStart)}</p>
                    <p><strong>Latest End:</strong> ${formatTimestamp(result.maintenanceWindow.latestEnd)}</p>
                    <p><strong>Duration:</strong> ${result.maintenanceWindow.duration}</p>
                    <p><strong>Downtime Required:</strong> ${result.maintenanceWindow.downtimeRequired ? 'Yes' : 'No'}</p>
                    <p><strong>Impact:</strong> ${result.maintenanceWindow.impact}</p>
                </div>
            </div>
        </div>
    `;
}

async function loadKnowledgeGraph() {
    try {
        const response = await fetch(`${API_BASE}/knowledge-graph?depth=2`);
        const graph = await response.json();

        displayKnowledgeGraph(graph);
    } catch (error) {
        console.error('Failed to load knowledge graph:', error);
        showToast('Failed to load knowledge graph', 'error');
    }
}

function displayKnowledgeGraph(graph) {
    const container = document.getElementById('knowledgeGraphContent');
    if (!container) return;

    container.innerHTML = `
        <div class="row">
            <div class="col-md-8">
                <div id="kg-visualization" style="height: 500px; background: #f8f9fa; border-radius: 8px;"></div>
            </div>
            <div class="col-md-4">
                <div class="card">
                    <div class="card-header">
                        <h6 class="mb-0">Summary</h6>
                    </div>
                    <div class="card-body">
                        <p>${graph.summary}</p>
                        <hr>
                        <h6>Nodes (${graph.nodes.length})</h6>
                        <ul class="list-group">
                            ${graph.nodes.map(node => `
                                <li class="list-group-item d-flex justify-content-between align-items-center">
                                    <div>
                                        <strong>${node.name}</strong>
                                        <br><small class="text-muted">${node.type}</small>
                                    </div>
                                    <button class="btn btn-sm btn-outline-primary" onclick="showNodeDetails('${node.id}')">
                                        Details
                                    </button>
                                </li>
                            `).join('')}
                        </ul>
                    </div>
                </div>
            </div>
        </div>
    `;

    renderKnowledgeGraphVisualization(graph);
}

function renderKnowledgeGraphVisualization(graph) {
    const canvas = document.getElementById('kg-visualization');
    if (!canvas) return;

    const ctx = canvas.getContext('2d');
    const centerX = canvas.width / 2;
    const centerY = canvas.height / 2;

    ctx.clearRect(0, 0, canvas.width, canvas.height);

    const angleStep = (2 * Math.PI) / graph.nodes.length;
    graph.nodes.forEach((node, i) => {
        const x = centerX + 150 * Math.cos(angleStep * i);
        const y = centerY + 150 * Math.sin(angleStep * i);

        ctx.beginPath();
        ctx.arc(x, y, 30, 0, 2 * Math.PI);
        ctx.fillStyle = getNodeColor(node.type);
        ctx.fill();
        ctx.fillStyle = '#000';
        ctx.textAlign = 'center';
        ctx.fillText(node.name.substring(0, 10), x, y + 50);
    });
}

function getNodeColor(type) {
    const colors = {
        'service': 'rgba(75, 192, 192, 0.8)',
        'database': 'rgba(255, 99, 132, 0.8)',
        'cache': 'rgba(255, 159, 64, 0.8)',
        'queue': 'rgba(153, 102, 255, 0.8)'
    };
    return colors[type] || 'rgba(201, 203, 207, 0.8)';
}

async function queryKnowledgeGraph() {
    const queryType = document.getElementById('kgQueryType')?.value || 'dependencies';
    const entity = document.getElementById('kgEntity')?.value || 'service';

    try {
        showToast('Querying knowledge graph...', 'info');

        const response = await fetch(`${API_BASE}/knowledge-graph/query`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                queryType,
                entities: [entity],
                relationships: ['depends_on', 'connects_to'],
                depth: 2
            })
        });

        const result = await response.json();
        showToast('Query completed', 'success');
        displayKnowledgeGraph(result);
    } catch (error) {
        console.error('Knowledge graph query failed:', error);
        showToast('Query failed', 'error');
    }
}

async function loadIncidents() {
    try {
        const response = await fetch(`${API_BASE}/incidents`);
        const incidents = await response.json();

        displayIncidentsList(incidents);
    } catch (error) {
        console.error('Failed to load incidents:', error);
        showToast('Failed to load incidents', 'error');
    }
}

function displayIncidentsList(incidents) {
    const container = document.getElementById('incidentsListContainer');
    if (!container) return;

    container.innerHTML = incidents.map(incident => `
        <div class="card mb-2">
            <div class="card-body">
                <div class="d-flex justify-content-between align-items-center">
                    <div>
                        <h6 class="mb-0">${incident.title}</h6>
                        <small class="text-muted">${incident.incidentId}</small>
                    </div>
                    <div>
                        <span class="badge bg-${getStatusColor(incident.status)}">${incident.status}</span>
                        <span class="badge bg-${getPriorityColor(incident.priority)}">${incident.priority}</span>
                    </div>
                </div>
                <div class="mt-2">
                    <small class="text-muted">
                        Created: ${formatTimestamp(incident.createdAt)} |
                        Updated: ${formatTimestamp(incident.updatedAt)}
                    </small>
                </div>
                <div class="mt-2">
                    <button class="btn btn-sm btn-outline-primary" onclick="showIncidentContext('${incident.incidentId}')">
                        View Context
                    </button>
                    <button class="btn btn-sm btn-outline-success" onclick="autoResolveIncident('${incident.incidentId}')">
                        Auto Resolve
                    </button>
                </div>
            </div>
        </div>
    `).join('');
}

async function showIncidentContext(incidentId) {
    try {
        const response = await fetch(`${API_BASE}/incident/${incidentId}/context`);
        const context = await response.json();

        displayIncidentContextModal(context);
    } catch (error) {
        console.error('Failed to load incident context:', error);
        showToast('Failed to load incident context', 'error');
    }
}

function displayIncidentContextModal(context) {
    const modal = new bootstrap.Modal(document.getElementById('incidentContextModal'));
    const content = document.getElementById('incidentContextContent');

    content.innerHTML = `
        <div class="row">
            <div class="col-md-12">
                <h6>Timeline</h6>
                <div class="list-group">
                    ${context.timeline.map(event => `
                        <div class="list-group-item">
                            <div class="d-flex justify-content-between">
                                <div>
                                    <span class="badge bg-${getEventTypeColor(event.type)} me-2">${event.type}</span>
                                    ${event.description}
                                </div>
                                <small class="text-muted">${formatTimestamp(event.timestamp)}</small>
                            </div>
                        </div>
                    `).join('')}
                </div>
            </div>
        </div>
        <div class="row mt-3">
            <div class="col-md-6">
                <h6>Related Changes</h6>
                <ul class="list-group">
                    ${context.relatedChanges.map(change => `
                        <li class="list-group-item">
                            <strong>${change.type}</strong>: ${change.description}
                            <br><small class="text-muted">${formatTimestamp(change.timestamp)} by ${change.changeBy}</small>
                        </li>
                    `).join('')}
                </ul>
            </div>
            <div class="col-md-6">
                <h6>Related Alerts</h6>
                <ul class="list-group">
                    ${context.relatedAlerts.map(alert => `
                        <li class="list-group-item d-flex justify-content-between align-items-center">
                            <div>
                                <strong>${alert.title}</strong>
                                <br><small class="text-muted">${formatTimestamp(alert.timestamp)}</small>
                            </div>
                            <span class="badge bg-${getSeverityColor(alert.severity)}">${alert.severity}</span>
                        </li>
                    `).join('')}
                </ul>
            </div>
        </div>
    `;

    modal.show();
}

async function autoResolveIncident(incidentId) {
    if (!confirm('Attempt to automatically resolve this incident?')) {
        return;
    }

    try {
        const response = await fetch(`${API_BASE}/incident/${incidentId}/auto-resolve`, {
            method: 'POST'
        });

        const result = await response.json();

        if (result.autoResolved) {
            showToast('Incident auto-resolved successfully', 'success');
        } else {
            showToast('Auto-resolution not possible. Manual intervention required.', 'warning');
            displayResolutionActions(result);
        }

        loadIncidents();
    } catch (error) {
        console.error('Auto-resolution failed:', error);
        showToast('Auto-resolution failed', 'error');
    }
}

function displayResolutionActions(result) {
    const container = document.getElementById('resolutionActionsContainer');
    if (!container) return;

    container.innerHTML = `
        <div class="card mt-3">
            <div class="card-header">
                <h6 class="mb-0">Suggested Resolution Actions</h6>
            </div>
            <div class="card-body">
                <ul class="list-group">
                    ${result.actions.map(action => `
                        <li class="list-group-item d-flex justify-content-between align-items-center">
                            <div>
                                <strong>${action.type}</strong>: ${action.description}
                                <br><small class="text-muted">Target: ${action.target}</small>
                            </div>
                            <div>
                                <span class="badge bg-${getRiskLevelColor(action.riskLevel)}">${action.riskLevel} risk</span>
                                <button class="btn btn-sm btn-primary" onclick="executeAction('${action.actionId}')">
                                    Execute
                                </button>
                            </div>
                        </li>
                    `).join('')}
                </ul>
                ${result.rollbackPlan ? `
                    <div class="mt-3">
                        <h6>Rollback Plan</h6>
                        <pre class="bg-light p-3">${result.rollbackPlan}</pre>
                    </div>
                ` : ''}
            </div>
        </div>
    `;
}

function filterAnomaliesBySeverity(severity) {
    document.querySelectorAll('.severity-filter').forEach(btn => {
        btn.classList.remove('active');
    });
    document.querySelector(`[data-severity="${severity}"]`)?.classList.add('active');

    loadAnomaliesList().then(anomalies => {
        const filtered = anomalies.filter(a => a.severity === severity);
        displayAnomaliesTable(filtered);
    });
}

function loadMetricAnomalies(metric) {
    loadAnomaliesList();
}

function formatTimestamp(timestamp) {
    if (!timestamp) return 'N/A';
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
