let enhancedMonitoring = {
    ws: null,
    charts: {},
    metricsData: {
        cpu: [],
        memory: [],
        disk: [],
        network: [],
        requests: [],
        errors: []
    },
    config: {
        WS_RECONNECT_INTERVAL: 5000,
        METRICS_UPDATE_INTERVAL: 5000,
        CHART_MAX_POINTS: 60,
        ENABLE_REALTIME: true
    },
    alerts: [],
    thresholds: {
        cpu: { warning: 70, critical: 90 },
        memory: { warning: 75, critical: 90 },
        disk: { warning: 80, critical: 95 },
        errorRate: { warning: 5, critical: 10 }
    }
};

document.addEventListener('DOMContentLoaded', () => {
    initEnhancedMonitoring();
    setupWebSocket();
    setupEventListeners();
    loadInitialData();
});

function initEnhancedMonitoring() {
    initCharts();
    initGaugeCharts();
    initAlertTable();
    loadThresholds();
}

function initCharts() {
    const chartConfig = {
        responsive: true,
        maintainAspectRatio: false,
        animation: {
            duration: 300
        },
        plugins: {
            legend: {
                position: 'top',
                labels: {
                    boxWidth: 12,
                    padding: 15
                }
            },
            tooltip: {
                mode: 'index',
                intersect: false
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
    };

    const cpuCtx = document.getElementById('cpuChart');
    if (cpuCtx) {
        enhancedMonitoring.charts.cpu = new Chart(cpuCtx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [{
                    label: 'CPU使用率',
                    data: [],
                    borderColor: '#3b82f6',
                    backgroundColor: 'rgba(59, 130, 246, 0.1)',
                    fill: true,
                    tension: 0.4,
                    pointRadius: 2
                }]
            },
            options: chartConfig
        });
    }

    const memoryCtx = document.getElementById('memoryChart');
    if (memoryCtx) {
        enhancedMonitoring.charts.memory = new Chart(memoryCtx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [{
                    label: '内存使用率',
                    data: [],
                    borderColor: '#10b981',
                    backgroundColor: 'rgba(16, 185, 129, 0.1)',
                    fill: true,
                    tension: 0.4,
                    pointRadius: 2
                }]
            },
            options: chartConfig
        });
    }

    const networkCtx = document.getElementById('networkChart');
    if (networkCtx) {
        enhancedMonitoring.charts.network = new Chart(networkCtx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [
                    {
                        label: '入站流量',
                        data: [],
                        borderColor: '#8b5cf6',
                        backgroundColor: 'rgba(139, 92, 246, 0.1)',
                        fill: true,
                        tension: 0.4
                    },
                    {
                        label: '出站流量',
                        data: [],
                        borderColor: '#ec4899',
                        backgroundColor: 'rgba(236, 72, 153, 0.1)',
                        fill: true,
                        tension: 0.4
                    }
                ]
            },
            options: chartConfig
        });
    }

    const requestCtx = document.getElementById('requestChart');
    if (requestCtx) {
        enhancedMonitoring.charts.requests = new Chart(requestCtx, {
            type: 'bar',
            data: {
                labels: [],
                datasets: [
                    {
                        label: '成功请求',
                        data: [],
                        backgroundColor: 'rgba(16, 185, 129, 0.8)'
                    },
                    {
                        label: '失败请求',
                        data: [],
                        backgroundColor: 'rgba(239, 68, 68, 0.8)'
                    }
                ]
            },
            options: {
                ...chartConfig,
                scales: {
                    ...chartConfig.scales,
                    y: {
                        ...chartConfig.scales.y,
                        stacked: true
                    }
                }
            }
        });
    }
}

function initGaugeCharts() {
    const gaugeConfig = {
        angle: 0.15,
        lineWidth: 0.3,
        pointer: {
            length: 0.6,
            strokeWidth: 0.03,
            color: '#000000'
        },
        limitMin: false,
        limitMax: false,
        colorStart: '#6fad01',
        colorStop: '#8fc800',
        strokeColor: '#E0E0E0',
        generateGradient: true,
        highDpiSupport: true
    };

    const cpuGauge = document.getElementById('cpuGauge');
    if (cpuGauge) {
        enhancedMonitoring.gauges = enhancedMonitoring.gauges || {};
        enhancedMonitoring.gauges.cpu = new Gauge(cpuGauge).setOptions(gaugeConfig);
        enhancedMonitoring.gauges.cpu.maxValue = 100;
        enhancedMonitoring.gauges.cpu.set(0);
    }

    const memoryGauge = document.getElementById('memoryGauge');
    if (memoryGauge) {
        enhancedMonitoring.gauges = enhancedMonitoring.gauges || {};
        enhancedMonitoring.gauges.memory = new Gauge(memoryGauge).setOptions(gaugeConfig);
        enhancedMonitoring.gauges.memory.maxValue = 100;
        enhancedMonitoring.gauges.memory.set(0);
    }
}

function initAlertTable() {
    loadAlerts();
}

function setupWebSocket() {
    if (!enhancedMonitoring.config.ENABLE_REALTIME) {
        return;
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/v1/admin/monitoring/ws`;

    try {
        enhancedMonitoring.ws = new WebSocket(wsUrl);

        enhancedMonitoring.ws.onopen = () => {
            updateConnectionStatus(true);
            console.log('Enhanced monitoring WebSocket connected');
            loadInitialData();
        };

        enhancedMonitoring.ws.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                handleEnhancedMessage(data);
            } catch (e) {
                console.error('Error parsing WebSocket message:', e);
            }
        };

        enhancedMonitoring.ws.onclose = () => {
            updateConnectionStatus(false);
            console.log('WebSocket disconnected, reconnecting in 5s...');
            setTimeout(setupWebSocket, enhancedMonitoring.config.WS_RECONNECT_INTERVAL);
        };

        enhancedMonitoring.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
            updateConnectionStatus(false);
        };
    } catch (error) {
        console.error('Failed to create WebSocket:', error);
    }
}

function handleEnhancedMessage(data) {
    switch(data.type) {
        case 'initial':
            handleInitialData(data);
            break;
        case 'metrics':
            handleMetricsUpdate(data.metrics);
            break;
        case 'alert':
            handleNewAlert(data.alert);
            break;
        case 'system_metrics':
            handleSystemMetrics(data);
            break;
    }
}

function handleInitialData(data) {
    if (data.systemMetrics) {
        updateAllMetrics(data.systemMetrics);
    }

    if (data.recentAlerts) {
        enhancedMonitoring.alerts = data.recentAlerts;
        renderAlertTable();
    }
}

function handleMetricsUpdate(metrics) {
    if (!metrics) return;

    updateChartData('cpu', metrics.cpu_usage);
    updateChartData('memory', metrics.memory_usage);
    updateChartData('network', metrics.network_in, metrics.network_out);

    if (metrics.request_count !== undefined) {
        updateChartData('requests', metrics.request_count, metrics.error_count);
    }

    updateMetricsDisplay(metrics);
    checkThresholds(metrics);
}

function handleSystemMetrics(data) {
    if (data.cpu) updateChartData('cpu', data.cpu);
    if (data.memory) updateChartData('memory', data.memory);
    if (data.network) {
        updateChartData('network', data.network.in, data.network.out);
    }

    updateMetricsDisplay(data);
    checkThresholds(data);
}

function handleNewAlert(alert) {
    enhancedMonitoring.alerts.unshift(alert);

    if (enhancedMonitoring.alerts.length > 50) {
        enhancedMonitoring.alerts.pop();
    }

    renderAlertTable();
    showAlertNotification(alert);

    updateAlertBadge();
}

function updateChartData(chartName, ...values) {
    const chart = enhancedMonitoring.charts[chartName];
    if (!chart) return;

    const now = new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });

    chart.data.labels.push(now);
    if (chart.data.labels.length > enhancedMonitoring.config.CHART_MAX_POINTS) {
        chart.data.labels.shift();
    }

    if (values.length === 1) {
        chart.data.datasets[0].data.push(values[0]);
        if (chart.data.datasets[0].data.length > enhancedMonitoring.config.CHART_MAX_POINTS) {
            chart.data.datasets[0].data.shift();
        }
    } else if (values.length === 2) {
        chart.data.datasets[0].data.push(values[0]);
        if (chart.data.datasets[1]) {
            chart.data.datasets[1].data.push(values[1]);
        }
        if (chart.data.datasets[0].data.length > enhancedMonitoring.config.CHART_MAX_POINTS) {
            chart.data.datasets[0].data.shift();
            if (chart.data.datasets[1]) {
                chart.data.datasets[1].data.shift();
            }
        }
    }

    chart.update('none');
}

function updateMetricsDisplay(metrics) {
    const elements = {
        cpuValue: metrics.cpu_usage,
        memoryValue: metrics.memory_usage,
        diskValue: metrics.disk_usage,
        networkInValue: formatBytes(metrics.network_in),
        networkOutValue: formatBytes(metrics.network_out),
        requestCount: formatNumber(metrics.request_count),
        errorCount: formatNumber(metrics.error_count),
        successRate: metrics.success_rate ? `${metrics.success_rate.toFixed(2)}%` : '0%',
        avgResponseTime: metrics.avg_response_time ? `${metrics.avg_response_time.toFixed(0)}ms` : '0ms',
        activeConnections: metrics.active_connections || 0,
        uptime: formatUptime(metrics.uptime)
    };

    Object.keys(elements).forEach(key => {
        const el = document.getElementById(key);
        if (el) {
            el.textContent = elements[key];
        }
    });

    if (enhancedMonitoring.gauges) {
        if (enhancedMonitoring.gauges.cpu && metrics.cpu_usage !== undefined) {
            enhancedMonitoring.gauges.cpu.set(metrics.cpu_usage);
        }
        if (enhancedMonitoring.gauges.memory && metrics.memory_usage !== undefined) {
            enhancedMonitoring.gauges.memory.set(metrics.memory_usage);
        }
    }

    updateStatusIndicator(metrics);
}

function updateStatusIndicator(metrics) {
    const indicator = document.getElementById('systemHealthIndicator');
    if (!indicator) return;

    let status = 'healthy';
    let statusClass = 'success';
    let statusText = '系统正常';

    if (metrics.success_rate < 90) {
        status = 'critical';
        statusClass = 'danger';
        statusText = '系统异常';
    } else if (metrics.success_rate < 95) {
        status = 'warning';
        statusClass = 'warning';
        statusText = '性能下降';
    }

    if (metrics.cpu_usage > 90 || metrics.memory_usage > 90) {
        status = 'critical';
        statusClass = 'danger';
        statusText = '资源紧张';
    } else if (metrics.cpu_usage > 70 || metrics.memory_usage > 75) {
        status = 'warning';
        statusClass = 'warning';
        statusText = '资源偏高';
    }

    indicator.className = `status-indicator status-${statusClass}`;
    indicator.textContent = statusText;
}

function checkThresholds(metrics) {
    const newAlerts = [];

    if (metrics.cpu_usage > enhancedMonitoring.thresholds.cpu.critical) {
        newAlerts.push({
            name: 'CPU使用率过高',
            severity: 'critical',
            message: `CPU使用率达到 ${metrics.cpu_usage.toFixed(1)}%`,
            value: metrics.cpu_usage,
            threshold: enhancedMonitoring.thresholds.cpu.critical
        });
    } else if (metrics.cpu_usage > enhancedMonitoring.thresholds.cpu.warning) {
        newAlerts.push({
            name: 'CPU使用率偏高',
            severity: 'warning',
            message: `CPU使用率达到 ${metrics.cpu_usage.toFixed(1)}%`,
            value: metrics.cpu_usage,
            threshold: enhancedMonitoring.thresholds.cpu.warning
        });
    }

    if (metrics.memory_usage > enhancedMonitoring.thresholds.memory.critical) {
        newAlerts.push({
            name: '内存使用率过高',
            severity: 'critical',
            message: `内存使用率达到 ${metrics.memory_usage.toFixed(1)}%`,
            value: metrics.memory_usage,
            threshold: enhancedMonitoring.thresholds.memory.critical
        });
    } else if (metrics.memory_usage > enhancedMonitoring.thresholds.memory.warning) {
        newAlerts.push({
            name: '内存使用率偏高',
            severity: 'warning',
            message: `内存使用率达到 ${metrics.memory_usage.toFixed(1)}%`,
            value: metrics.memory_usage,
            threshold: enhancedMonitoring.thresholds.memory.warning
        });
    }

    if (metrics.error_rate > enhancedMonitoring.thresholds.errorRate.critical) {
        newAlerts.push({
            name: '错误率过高',
            severity: 'critical',
            message: `错误率达到 ${metrics.error_rate.toFixed(2)}%`,
            value: metrics.error_rate,
            threshold: enhancedMonitoring.thresholds.errorRate.critical
        });
    }

    newAlerts.forEach(alert => {
        const existing = enhancedMonitoring.alerts.find(a =>
            a.name === alert.name && a.severity === alert.severity && a.status === 'firing'
        );

        if (!existing) {
            handleNewAlert({
                id: `alert-${Date.now()}`,
                ...alert,
                timestamp: new Date(),
                status: 'firing',
                source: 'monitoring'
            });
        }
    });
}

function updateConnectionStatus(connected) {
    const statusDot = document.getElementById('connectionStatus');
    const statusText = document.getElementById('connectionStatusText');

    if (connected) {
        if (statusDot) statusDot.className = 'status-dot connected';
        if (statusText) statusText.textContent = '已连接';
    } else {
        if (statusDot) statusDot.className = 'status-dot disconnected';
        if (statusText) statusText.textContent = '已断开';
    }
}

function renderAlertTable() {
    const tbody = document.getElementById('alertTableBody');
    if (!tbody) return;

    if (enhancedMonitoring.alerts.length === 0) {
        tbody.innerHTML = '<tr><td colspan="5" class="text-center text-muted">暂无告警</td></tr>';
        return;
    }

    tbody.innerHTML = enhancedMonitoring.alerts.map(alert => `
        <tr class="alert-row alert-${alert.severity}">
            <td>
                <span class="badge badge-${getSeverityBadgeClass(alert.severity)}">
                    ${alert.severity}
                </span>
            </td>
            <td>${alert.name}</td>
            <td>${alert.message}</td>
            <td>${new Date(alert.timestamp).toLocaleString('zh-CN')}</td>
            <td>
                <button class="btn btn-sm btn-primary" onclick="handleAlert('${alert.id}')">
                    处理
                </button>
                <button class="btn btn-sm btn-secondary" onclick="muteAlert('${alert.id}')">
                    静音
                </button>
            </td>
        </tr>
    `).join('');
}

function getSeverityBadgeClass(severity) {
    const map = {
        'critical': 'danger',
        'warning': 'warning',
        'info': 'info'
    };
    return map[severity?.toLowerCase()] || 'info';
}

function showAlertNotification(alert) {
    if (!('Notification' in window)) return;

    if (Notification.permission === 'granted') {
        new Notification(`告警: ${alert.name}`, {
            body: alert.message,
            icon: '/static/img/alert-icon.png'
        });
    } else if (Notification.permission !== 'denied') {
        Notification.requestPermission().then(permission => {
            if (permission === 'granted') {
                new Notification(`告警: ${alert.name}`, {
                    body: alert.message,
                    icon: '/static/img/alert-icon.png'
                });
            }
        });
    }
}

function updateAlertBadge() {
    const badge = document.getElementById('activeAlertBadge');
    if (!badge) return;

    const activeCount = enhancedMonitoring.alerts.filter(a => a.status === 'firing').length;
    if (activeCount > 0) {
        badge.textContent = activeCount;
        badge.style.display = 'inline';
    } else {
        badge.style.display = 'none';
    }
}

function setupEventListeners() {
    const refreshBtn = document.getElementById('refreshMetrics');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', () => {
            loadInitialData();
            showNotification('success', '刷新成功', '指标数据已更新');
        });
    }

    const exportBtn = document.getElementById('exportMetrics');
    if (exportBtn) {
        exportBtn.addEventListener('click', exportMetrics);
    }

    const alertSettingsBtn = document.getElementById('alertSettingsBtn');
    if (alertSettingsBtn) {
        alertSettingsBtn.addEventListener('click', showAlertSettings);
    }
}

function loadInitialData() {
    fetch('/admin/api/monitoring/stats', {
        headers: {
            'Authorization': 'Bearer ' + localStorage.getItem('admin_token')
        }
    })
    .then(response => response.json())
    .then(result => {
        if (result.code === 0) {
            updateAllMetrics(result.data);
        }
    })
    .catch(error => {
        console.error('Load initial data failed:', error);
    });

    loadAlerts();
}

function loadAlerts() {
    fetch('/admin/api/alerts?status=firing', {
        headers: {
            'Authorization': 'Bearer ' + localStorage.getItem('admin_token')
        }
    })
    .then(response => response.json())
    .then(result => {
        if (result.code === 0) {
            enhancedMonitoring.alerts = result.data || [];
            renderAlertTable();
            updateAlertBadge();
        }
    })
    .catch(error => {
        console.error('Load alerts failed:', error);
    });
}

function handleAlert(alertId) {
    fetch(`/admin/api/alerts/${alertId}/resolve`, {
        method: 'POST',
        headers: {
            'Authorization': 'Bearer ' + localStorage.getItem('admin_token')
        }
    })
    .then(() => {
        const alert = enhancedMonitoring.alerts.find(a => a.id === alertId);
        if (alert) {
            alert.status = 'resolved';
        }
        renderAlertTable();
        updateAlertBadge();
        showNotification('success', '处理成功', '告警已处理');
    })
    .catch(error => {
        console.error('Handle alert failed:', error);
        showNotification('error', '处理失败', error.message);
    });
}

function muteAlert(alertId) {
    const alert = enhancedMonitoring.alerts.find(a => a.id === alertId);
    if (alert) {
        alert.muted = true;
        renderAlertTable();
        showNotification('info', '已静音', '告警已静音');
    }
}

function loadThresholds() {
    fetch('/admin/api/monitoring/thresholds', {
        headers: {
            'Authorization': 'Bearer ' + localStorage.getItem('admin_token')
        }
    })
    .then(response => response.json())
    .then(result => {
        if (result.data) {
            enhancedMonitoring.thresholds = result.data;
        }
    })
    .catch(error => {
        console.error('Load thresholds failed:', error);
    });
}

function showAlertSettings() {
    const modal = document.getElementById('alertSettingsModal') || createAlertSettingsModal();
    $('#alertSettingsModal').modal('show');
}

function createAlertSettingsModal() {
    const modalHTML = `
        <div class="modal fade" id="alertSettingsModal" tabindex="-1" role="dialog">
            <div class="modal-dialog" role="document">
                <div class="modal-content">
                    <div class="modal-header">
                        <h5 class="modal-title">告警阈值设置</h5>
                        <button type="button" class="close" data-dismiss="modal">
                            <span>&times;</span>
                        </button>
                    </div>
                    <div class="modal-body">
                        <form id="thresholdForm">
                            <div class="form-group">
                                <label>CPU使用率 - 警告阈值</label>
                                <input type="number" class="form-control" id="cpuWarning" value="${enhancedMonitoring.thresholds.cpu.warning}">
                            </div>
                            <div class="form-group">
                                <label>CPU使用率 - 严重阈值</label>
                                <input type="number" class="form-control" id="cpuCritical" value="${enhancedMonitoring.thresholds.cpu.critical}">
                            </div>
                            <div class="form-group">
                                <label>内存使用率 - 警告阈值</label>
                                <input type="number" class="form-control" id="memoryWarning" value="${enhancedMonitoring.thresholds.memory.warning}">
                            </div>
                            <div class="form-group">
                                <label>内存使用率 - 严重阈值</label>
                                <input type="number" class="form-control" id="memoryCritical" value="${enhancedMonitoring.thresholds.memory.critical}">
                            </div>
                        </form>
                    </div>
                    <div class="modal-footer">
                        <button type="button" class="btn btn-secondary" data-dismiss="modal">取消</button>
                        <button type="button" class="btn btn-primary" onclick="saveThresholds()">保存</button>
                    </div>
                </div>
            </div>
        </div>
    `;
    document.body.insertAdjacentHTML('beforeend', modalHTML);
    return document.getElementById('alertSettingsModal');
}

function saveThresholds() {
    enhancedMonitoring.thresholds.cpu.warning = parseInt(document.getElementById('cpuWarning').value);
    enhancedMonitoring.thresholds.cpu.critical = parseInt(document.getElementById('cpuCritical').value);
    enhancedMonitoring.thresholds.memory.warning = parseInt(document.getElementById('memoryWarning').value);
    enhancedMonitoring.thresholds.memory.critical = parseInt(document.getElementById('memoryCritical').value);

    fetch('/admin/api/monitoring/thresholds', {
        method: 'POST',
        headers: {
            'Authorization': 'Bearer ' + localStorage.getItem('admin_token'),
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(enhancedMonitoring.thresholds)
    })
    .then(() => {
        $('#alertSettingsModal').modal('hide');
        showNotification('success', '保存成功', '告警阈值已更新');
    })
    .catch(error => {
        console.error('Save thresholds failed:', error);
        showNotification('error', '保存失败', error.message);
    });
}

function exportMetrics() {
    fetch('/admin/api/monitoring/export', {
        headers: {
            'Authorization': 'Bearer ' + localStorage.getItem('admin_token')
        }
    })
    .then(response => response.blob())
    .then(blob => {
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `metrics_${new Date().toISOString()}.json`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        window.URL.revokeObjectURL(url);

        showNotification('success', '导出成功', '监控数据已导出');
    })
    .catch(error => {
        console.error('Export failed:', error);
        showNotification('error', '导出失败', error.message);
    });
}

function updateAllMetrics(data) {
    if (!data) return;

    if (data.metrics) {
        updateMetricsDisplay(data.metrics);
        if (data.metrics.cpu_usage) {
            updateChartData('cpu', data.metrics.cpu_usage);
        }
        if (data.metrics.memory_usage) {
            updateChartData('memory', data.metrics.memory_usage);
        }
        if (data.metrics.network_in && data.metrics.network_out) {
            updateChartData('network', data.metrics.network_in, data.metrics.network_out);
        }
    } else {
        updateMetricsDisplay(data);
    }
}

function formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return (bytes / Math.pow(k, i)).toFixed(2) + ' ' + sizes[i];
}

function formatNumber(num) {
    if (num >= 1000000) {
        return (num / 1000000).toFixed(1) + 'M';
    } else if (num >= 1000) {
        return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
}

function formatUptime(seconds) {
    if (!seconds) return '0秒';

    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);

    if (days > 0) {
        return `${days}天 ${hours}小时`;
    } else if (hours > 0) {
        return `${hours}小时 ${minutes}分钟`;
    } else {
        return `${minutes}分钟`;
    }
}

function showNotification(type, title, message) {
    const toast = document.createElement('div');
    toast.className = `toast-notification toast-${type}`;
    toast.innerHTML = `
        <div class="toast-header">
            <strong class="mr-auto">${title}</strong>
            <button type="button" class="ml-2 mb-1 close" onclick="this.parentElement.parentElement.remove()">
                <span>&times;</span>
            </button>
        </div>
        <div class="toast-body">${message}</div>
    `;

    document.body.appendChild(toast);

    setTimeout(() => {
        toast.classList.add('show');
    }, 100);

    setTimeout(() => {
        toast.classList.remove('show');
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}
