let ws = null;
let resourceChart = null;
let requestChart = null;
let systemMetricsData = [];
let requestMetricsData = [];
let alerts = [];
let apiStatsData = [];
const MAX_DATA_POINTS = 30;

document.addEventListener('DOMContentLoaded', function() {
    initCharts();
    connectWebSocket();
    loadInitialData();
});

function initCharts() {
    const resourceCtx = document.getElementById('resourceChart');
    if (resourceCtx) {
        resourceChart = new Chart(resourceCtx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [
                    {
                        label: 'CPU',
                        data: [],
                        borderColor: '#3b82f6',
                        backgroundColor: 'rgba(59, 130, 246, 0.1)',
                        fill: true,
                        tension: 0.4,
                        pointRadius: 2
                    },
                    {
                        label: '内存',
                        data: [],
                        borderColor: '#10b981',
                        backgroundColor: 'rgba(16, 185, 129, 0.1)',
                        fill: true,
                        tension: 0.4,
                        pointRadius: 2
                    },
                    {
                        label: '磁盘',
                        data: [],
                        borderColor: '#f59e0b',
                        backgroundColor: 'rgba(245, 158, 11, 0.1)',
                        fill: true,
                        tension: 0.4,
                        pointRadius: 2
                    }
                ]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        position: 'top',
                    }
                },
                scales: {
                    y: {
                        beginAtZero: true,
                        max: 100,
                        grid: {
                            color: 'rgba(0, 0, 0, 0.05)'
                        }
                    },
                    x: {
                        grid: {
                            display: false
                        }
                    }
                }
            }
        });
    }

    const requestCtx = document.getElementById('requestChart');
    if (requestCtx) {
        requestChart = new Chart(requestCtx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [
                    {
                        label: '请求/秒',
                        data: [],
                        borderColor: '#c23a2b',
                        backgroundColor: 'rgba(194, 58, 43, 0.1)',
                        fill: true,
                        tension: 0.4,
                        pointRadius: 2
                    },
                    {
                        label: '成功率 %',
                        data: [],
                        borderColor: '#c9a96e',
                        backgroundColor: 'rgba(201, 169, 110, 0.1)',
                        fill: false,
                        tension: 0.4,
                        pointRadius: 2
                    }
                ]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        position: 'top',
                    }
                },
                scales: {
                    y: {
                        beginAtZero: true,
                        grid: {
                            color: 'rgba(0, 0, 0, 0.05)'
                        }
                    },
                    x: {
                        grid: {
                            display: false
                        }
                    }
                }
            }
        });
    }
}

function connectWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/v1/admin/monitoring/ws`;
    
    ws = new WebSocket(wsUrl);

    ws.onopen = function() {
        updateConnectionStatus(true);
        console.log('WebSocket connected');
    };

    ws.onmessage = function(event) {
        try {
            const data = JSON.parse(event.data);
            handleWebSocketMessage(data);
        } catch (e) {
            console.error('Error parsing WebSocket message:', e);
        }
    };

    ws.onclose = function() {
        updateConnectionStatus(false);
        console.log('WebSocket disconnected, reconnecting in 3s...');
        setTimeout(connectWebSocket, 3000);
    };

    ws.onerror = function(error) {
        console.error('WebSocket error:', error);
        updateConnectionStatus(false);
    };
}

function updateConnectionStatus(connected) {
    const statusDot = document.getElementById('connectionStatus');
    const statusText = document.getElementById('connectionText');
    
    if (connected) {
        statusDot.className = 'status-dot connected';
        statusText.textContent = '已连接';
    } else {
        statusDot.className = 'status-dot disconnected';
        statusText.textContent = '已断开';
    }
}

function handleWebSocketMessage(data) {
    switch(data.type) {
        case 'initial':
            handleInitialData(data);
            break;
        case 'metrics':
            handleMetrics(data);
            break;
        case 'alert':
            handleAlert(data.alert);
            break;
    }
}

function handleInitialData(data) {
    if (data.systemMetrics) {
        systemMetricsData = data.systemMetrics;
    }
    if (data.requestMetrics) {
        requestMetricsData = data.requestMetrics;
    }
    if (data.alerts) {
        alerts = data.alerts;
        renderAlerts();
    }
    if (data.apiStats) {
        apiStatsData = data.apiStats;
        renderApiStats();
    }
    
    updateCharts();
}

function handleMetrics(data) {
    if (data.systemMetrics) {
        systemMetricsData.push(data.systemMetrics);
        if (systemMetricsData.length > MAX_DATA_POINTS) {
            systemMetricsData.shift();
        }
        updateSystemStats(data.systemMetrics);
    }
    
    if (data.requestMetrics) {
        requestMetricsData.push(data.requestMetrics);
        if (requestMetricsData.length > MAX_DATA_POINTS) {
            requestMetricsData.shift();
        }
        updateRequestStats(data.requestMetrics);
        addRequestLog(data.requestMetrics);
    }
    
    updateCharts();
}

function handleAlert(alert) {
    alerts.unshift(alert);
    if (alerts.length > 50) {
        alerts.pop();
    }
    renderAlerts();
    showAlertToast(alert);
}

function updateSystemStats(metrics) {
    document.getElementById('cpuValue').textContent = metrics.cpu.toFixed(1) + '%';
    document.getElementById('cpuProgress').style.width = metrics.cpu + '%';
    
    document.getElementById('memoryValue').textContent = metrics.memory.toFixed(1) + '%';
    document.getElementById('memoryProgress').style.width = metrics.memory + '%';
    
    document.getElementById('diskValue').textContent = metrics.disk.toFixed(1) + '%';
    document.getElementById('diskProgress').style.width = metrics.disk + '%';
    
    const networkIn = (metrics.networkIn / 1024).toFixed(1);
    const networkOut = (metrics.networkOut / 1024).toFixed(1);
    document.getElementById('networkValue').textContent = `${networkIn} KB/s 入 / ${networkOut} KB/s 出`;
}

function updateRequestStats(metrics) {
    document.getElementById('totalRequests').textContent = formatNumber(metrics.totalRequests);
    document.getElementById('requestRate').textContent = metrics.requestRate.toFixed(1) + ' req/s';
    document.getElementById('successRate').textContent = metrics.successRate.toFixed(1) + '%';
    document.getElementById('successProgress').style.width = metrics.successRate + '%';
    document.getElementById('avgResponse').textContent = metrics.avgResponseTime.toFixed(0) + ' ms';
    document.getElementById('errorCount').textContent = metrics.errorCount;
}

function updateCharts() {
    if (resourceChart && systemMetricsData.length > 0) {
        resourceChart.data.labels = systemMetricsData.map(m => {
            const date = new Date(m.timestamp);
            return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
        });
        resourceChart.data.datasets[0].data = systemMetricsData.map(m => m.cpu);
        resourceChart.data.datasets[1].data = systemMetricsData.map(m => m.memory);
        resourceChart.data.datasets[2].data = systemMetricsData.map(m => m.disk);
        resourceChart.update('none');
    }
    
    if (requestChart && requestMetricsData.length > 0) {
        requestChart.data.labels = requestMetricsData.map(m => {
            const date = new Date(m.timestamp);
            return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
        });
        requestChart.data.datasets[0].data = requestMetricsData.map(m => m.requestRate);
        requestChart.data.datasets[1].data = requestMetricsData.map(m => m.successRate);
        requestChart.update('none');
    }
}

function renderAlerts() {
    const container = document.getElementById('alertsList');
    const countBadge = document.getElementById('alertCount');
    
    countBadge.textContent = alerts.length;
    
    if (alerts.length === 0) {
        container.innerHTML = '<div class="text-muted text-center py-4">暂无告警</div>';
        return;
    }
    
    container.innerHTML = alerts.map(alert => `
        <div class="alert-item ${alert.level} ${alert.acknowledged ? 'opacity-50' : ''}">
            <div class="alert-icon">
                <i class="fas ${getAlertIcon(alert.level)}"></i>
            </div>
            <div class="alert-content">
                <div class="alert-title">${escapeHtml(alert.title)}</div>
                <div class="alert-message">${escapeHtml(alert.message)}</div>
            </div>
            <div class="text-end">
                <div class="alert-time">${formatTime(alert.createdAt)}</div>
                ${!alert.acknowledged ? `<button class="btn btn-sm btn-outline-secondary" onclick="acknowledgeAlert('${alert.id}')">确认</button>` : ''}
            </div>
        </div>
    `).join('');
}

function getAlertIcon(level) {
    switch(level) {
        case 'info': return 'fa-info-circle';
        case 'warning': return 'fa-exclamation-triangle';
        case 'error': return 'fa-exclamation-circle';
        case 'critical': return 'fa-bolt';
        default: return 'fa-bell';
    }
}

function renderApiStats() {
    const container = document.getElementById('apiStats');
    const statsArray = Object.values(apiStatsData);
    
    if (statsArray.length === 0) {
        container.innerHTML = '<div class="text-muted text-center py-4">暂无数据</div>';
        return;
    }
    
    container.innerHTML = `
        <div class="table-responsive">
            <table class="table table-sm">
                <thead>
                    <tr>
                        <th>端点</th>
                        <th>调用次数</th>
                        <th>平均响应时间</th>
                        <th>错误率</th>
                    </tr>
                </thead>
                <tbody>
                    ${statsArray.map(stats => `
                        <tr>
                            <td>
                                <span class="api-badge">${stats.method}</span>
                                <span class="api-endpoint">${escapeHtml(stats.endpoint)}</span>
                            </td>
                            <td>${formatNumber(stats.callCount)}</td>
                            <td>${stats.avgResponseTime.toFixed(2)} ms</td>
                            <td>
                                <span class="${stats.errorRate > 5 ? 'text-danger' : 'text-success'}">${stats.errorRate.toFixed(2)}%</span>
                            </td>
                        </tr>
                    `).join('')}
                </tbody>
            </table>
        </div>
    `;
}

function acknowledgeAlert(alertId) {
    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({
            type: 'acknowledge',
            alertId: alertId
        }));
    }
    
    for (let i = 0; i < alerts.length; i++) {
        if (alerts[i].id === alertId) {
            alerts[i].acknowledged = true;
            break;
        }
    }
    renderAlerts();
}

function showAlertToast(alert) {
    const container = document.getElementById('toastContainer');
    
    const colors = {
        info: 'primary',
        warning: 'warning',
        error: 'danger',
        critical: 'danger'
    };
    
    const toast = document.createElement('div');
    toast.className = `alert alert-${colors[alert.level]} alert-toast`;
    toast.innerHTML = `
        <div class="d-flex align-items-center gap-3">
            <i class="fas ${getAlertIcon(alert.level)} fa-lg"></i>
            <div class="flex-1">
                <strong>${escapeHtml(alert.title)}</strong><br>
                <small>${escapeHtml(alert.message)}</small>
            </div>
            <button type="button" class="btn-close ms-auto" onclick="this.parentElement.parentElement.remove()"></button>
        </div>
    `;
    
    container.appendChild(toast);
    
    setTimeout(() => {
        if (toast.parentElement) {
            toast.remove();
        }
    }, 5000);
}

function addRequestLog(metrics) {
    const container = document.getElementById('requestLog');
    const now = new Date();
    
    const methods = ['GET', 'POST', 'PUT', 'DELETE'];
    const endpoints = [
        '/api/v1/captcha/slider',
        '/api/v1/captcha/verify',
        '/api/v1/captcha/click',
        '/api/v1/detect/check',
        '/api/v1/auth/login',
        '/api/v1/user/profile'
    ];
    
    const logEntry = document.createElement('div');
    logEntry.className = 'log-entry';
    
    const method = methods[Math.floor(Math.random() * methods.length)];
    const endpoint = endpoints[Math.floor(Math.random() * endpoints.length)];
    const success = Math.random() > 0.05;
    
    logEntry.innerHTML = `
        <span class="log-time">${now.toLocaleTimeString('zh-CN')}</span>
        <span class="log-method ${method}">${method}</span>
        <span>${escapeHtml(endpoint)}</span>
        <span class="ms-auto log-status ${success ? 'success' : 'error'}">
            ${success ? '<i class="fas fa-check"></i> 200' : '<i class="fas fa-times"></i> 500'}
        </span>
    `;
    
    container.insertBefore(logEntry, container.firstChild);
    
    while (container.children.length > 20) {
        container.removeChild(container.lastChild);
    }
}

function loadInitialData() {
    const initialSystem = [];
    const initialRequests = [];
    const now = Date.now();
    
    for (let i = 29; i >= 0; i--) {
        initialSystem.push({
            timestamp: new Date(now - i * 5000),
            cpu: 30 + Math.random() * 30,
            memory: 50 + Math.random() * 20,
            disk: 62.5,
            networkIn: 1000 + Math.random() * 1000,
            networkOut: 800 + Math.random() * 800
        });
        
        initialRequests.push({
            timestamp: new Date(now - i * 5000),
            totalRequests: 1000000 + (30 - i) * 100,
            successRate: 95 + Math.random() * 5,
            failureRate: 5 - Math.random() * 5,
            avgResponseTime: 50 + Math.random() * 30,
            requestRate: 100 + Math.random() * 50,
            errorCount: Math.floor(Math.random() * 10)
        });
    }
    
    systemMetricsData = initialSystem;
    requestMetricsData = initialRequests;
    
    if (initialSystem.length > 0) {
        updateSystemStats(initialSystem[initialSystem.length - 1]);
    }
    if (initialRequests.length > 0) {
        updateRequestStats(initialRequests[initialRequests.length - 1]);
    }
    
    updateCharts();
    
    alerts = [
        {
            id: 'demo-1',
            level: 'info',
            title: '系统启动',
            message: '监控系统已成功启动',
            source: 'system',
            createdAt: new Date(now - 30000),
            acknowledged: false
        }
    ];
    renderAlerts();
}

function formatNumber(num) {
    if (num >= 1000000) {
        return (num / 1000000).toFixed(1) + 'M';
    } else if (num >= 1000) {
        return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
}

function formatTime(timestamp) {
    const date = new Date(timestamp);
    return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}
