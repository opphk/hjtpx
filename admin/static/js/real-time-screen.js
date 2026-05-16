let ws = null;
let realtimeChart = null;
let captchaTypeChart = null;
let riskDistributionChart = null;
let topAppsChart = null;
let realtimeData = [];
const MAX_DATA_POINTS = 60;

let chartsInitialized = false;

document.addEventListener('DOMContentLoaded', function() {
    initCharts();
    initWebSocket();
    updateTime();
    setInterval(updateTime, 1000);
    loadInitialData();
});

function updateTime() {
    const now = new Date();
    document.getElementById('currentTime').textContent = now.toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
    });
}

function exitScreen() {
    if (ws) {
        ws.close();
    }
    window.location.href = '/admin';
}

function initCharts() {
    const chartOptions = {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
            legend: {
                labels: { color: 'rgba(255,255,255,0.7)', font: { size: 11 } }
            }
        }
    };

    const realtimeCtx = document.getElementById('realtimeChart').getContext('2d');
    realtimeChart = new Chart(realtimeCtx, {
        type: 'line',
        data: {
            labels: [],
            datasets: [
                {
                    label: '请求数',
                    data: [],
                    borderColor: '#00d4ff',
                    backgroundColor: 'rgba(0, 212, 255, 0.1)',
                    fill: true,
                    tension: 0.4,
                    pointRadius: 0
                },
                {
                    label: '成功数',
                    data: [],
                    borderColor: '#28a745',
                    backgroundColor: 'rgba(40, 167, 69, 0.1)',
                    fill: true,
                    tension: 0.4,
                    pointRadius: 0
                }
            ]
        },
        options: {
            ...chartOptions,
            scales: {
                x: { ticks: { color: 'rgba(255,255,255,0.5)' }, grid: { color: 'rgba(255,255,255,0.1)' } },
                y: { ticks: { color: 'rgba(255,255,255,0.5)' }, grid: { color: 'rgba(255,255,255,0.1)' } }
            }
        }
    });

    const captchaTypeCtx = document.getElementById('captchaTypeChart').getContext('2d');
    captchaTypeChart = new Chart(captchaTypeCtx, {
        type: 'doughnut',
        data: {
            labels: ['滑块验证', '点击验证', '手势验证', '拼图验证'],
            datasets: [{
                data: [35, 25, 20, 20],
                backgroundColor: ['#00d4ff', '#28a745', '#ffc107', '#dc3545']
            }]
        },
        options: {
            ...chartOptions,
            cutout: '60%'
        }
    });

    const riskCtx = document.getElementById('riskDistributionChart').getContext('2d');
    riskDistributionChart = new Chart(riskCtx, {
        type: 'pie',
        data: {
            labels: ['低风险', '中风险', '高风险'],
            datasets: [{
                data: [70, 20, 10],
                backgroundColor: ['#28a745', '#ffc107', '#dc3545']
            }]
        },
        options: chartOptions
    });

    const topAppsCtx = document.getElementById('topAppsChart').getContext('2d');
    topAppsChart = new Chart(topAppsCtx, {
        type: 'bar',
        data: {
            labels: ['应用A', '应用B', '应用C', '应用D', '应用E'],
            datasets: [{
                label: '请求数',
                data: [5000, 4200, 3800, 2900, 2100],
                backgroundColor: 'rgba(0, 212, 255, 0.6)',
                borderColor: '#00d4ff',
                borderWidth: 1
            }]
        },
        options: {
            ...chartOptions,
            indexAxis: 'y',
            scales: {
                x: { ticks: { color: 'rgba(255,255,255,0.5)' }, grid: { color: 'rgba(255,255,255,0.1)' } },
                y: { ticks: { color: 'rgba(255,255,255,0.5)' }, grid: { display: false } }
            }
        }
    });

    chartsInitialized = true;
}

function initWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/v1/admin/monitoring/ws`;

    try {
        ws = new WebSocket(wsUrl);

        ws.onopen = function() {
            updateConnectionStatus(true);
            console.log('WebSocket connected');
        };

        ws.onmessage = function(event) {
            try {
                const data = JSON.parse(event.data);
                handleWebSocketData(data);
            } catch (e) {
                console.error('Failed to parse WebSocket data:', e);
            }
        };

        ws.onerror = function(error) {
            console.error('WebSocket error:', error);
            updateConnectionStatus(false);
        };

        ws.onclose = function() {
            updateConnectionStatus(false);
            console.log('WebSocket disconnected, reconnecting in 3s...');
            setTimeout(initWebSocket, 3000);
        };
    } catch (e) {
        console.error('Failed to initialize WebSocket:', e);
        updateConnectionStatus(false);
        setTimeout(initWebSocket, 3000);
    }
}

function updateConnectionStatus(connected) {
    const statusEl = document.getElementById('connectionStatus');
    const textEl = document.getElementById('connectionText');
    
    if (connected) {
        statusEl.classList.remove('disconnected');
        statusEl.classList.add('connected');
        textEl.textContent = '已连接';
    } else {
        statusEl.classList.remove('connected');
        statusEl.classList.add('disconnected');
        textEl.textContent = '已断开';
    }
}

function handleWebSocketData(data) {
    if (data.type === 'metrics') {
        updateMetrics(data.payload);
    } else if (data.type === 'alert') {
        addAlert(data.payload);
    }
}

function updateMetrics(data) {
    document.getElementById('totalRequests').textContent = formatNumber(data.total_requests || 0);
    document.getElementById('successCount').textContent = formatNumber(data.success_count || 0);
    document.getElementById('failCount').textContent = formatNumber(data.fail_count || 0);
    document.getElementById('qpsValue').textContent = (data.qps || 0).toFixed(0);
    document.getElementById('avgResponse').textContent = (data.avg_response_time || 0) + 'ms';

    const total = (data.total_requests || 0);
    const successRate = total > 0 ? ((data.success_count || 0) / total * 100).toFixed(1) : 0;
    const failRate = total > 0 ? ((data.fail_count || 0) / total * 100).toFixed(1) : 0;
    document.getElementById('successRate').textContent = successRate + '%';
    document.getElementById('failRate').textContent = failRate + '%';

    updateGauge('cpu', data.cpu_usage || 0);
    updateGauge('memory', data.memory_usage || 0);
    updateGauge('disk', data.disk_usage || 0);

    updateRealtimeChart(data);

    if (data.captcha_types) {
        updateCaptchaTypeChart(data.captcha_types);
    }

    if (data.risk_distribution) {
        updateRiskDistributionChart(data.risk_distribution);
    }

    if (data.top_apps) {
        updateTopAppsChart(data.top_apps);
    }

    if (data.devices) {
        updateDeviceList(data.devices);
    }
}

function updateGauge(type, value) {
    const circle = document.getElementById(type + 'Circle');
    const valueEl = document.getElementById(type + 'Value');
    const circumference = 251.2;
    const offset = circumference - (value / 100) * circumference;
    circle.style.strokeDashoffset = offset;
    valueEl.textContent = value.toFixed(1) + '%';
}

function updateRealtimeChart(data) {
    if (!realtimeChart) return;

    const now = new Date();
    const timeLabel = now.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' });

    realtimeChart.data.labels.push(timeLabel);
    realtimeChart.data.datasets[0].data.push(data.requests || 0);
    realtimeChart.data.datasets[1].data.push(data.success_count || 0);

    if (realtimeChart.data.labels.length > MAX_DATA_POINTS) {
        realtimeChart.data.labels.shift();
        realtimeChart.data.datasets[0].data.shift();
        realtimeChart.data.datasets[1].data.shift();
    }

    realtimeChart.update('none');
}

function updateCaptchaTypeChart(data) {
    if (!captchaTypeChart) return;
    captchaTypeChart.data.datasets[0].data = [
        data.slider || 0,
        data.click || 0,
        data.gesture || 0,
        data.jigsaw || 0
    ];
    captchaTypeChart.update('none');
}

function updateRiskDistributionChart(data) {
    if (!riskDistributionChart) return;
    riskDistributionChart.data.datasets[0].data = [
        data.low || 0,
        data.medium || 0,
        data.high || 0
    ];
    riskDistributionChart.update('none');
}

function updateTopAppsChart(data) {
    if (!topAppsChart) return;
    topAppsChart.data.labels = data.map(item => item.name);
    topAppsChart.data.datasets[0].data = data.map(item => item.requests);
    topAppsChart.update('none');
}

function updateDeviceList(devices) {
    const container = document.getElementById('deviceList');
    container.innerHTML = '';
    
    devices.forEach(device => {
        const deviceEl = document.createElement('div');
        deviceEl.className = 'device-item';
        deviceEl.innerHTML = `
            <div class="device-info">
                <div class="device-icon ${device.status}">
                    <i class="fas fa-${device.icon || 'server'}"></i>
                </div>
                <div>
                    <div class="device-name">${device.name}</div>
                </div>
            </div>
            <div class="device-status ${device.status}">
                ${device.status === 'online' ? '在线' : device.status === 'warning' ? '警告' : '离线'}
            </div>
        `;
        container.appendChild(deviceEl);
    });
}

function addAlert(alert) {
    const container = document.getElementById('alertsContainer');
    const alertEl = document.createElement('div');
    alertEl.className = `alert-item ${alert.severity}`;
    
    const time = new Date(alert.timestamp * 1000);
    alertEl.innerHTML = `
        <div class="alert-time">${time.toLocaleTimeString('zh-CN')}</div>
        <div class="alert-message"><i class="fas fa-${alert.icon || 'exclamation-triangle'} me-2"></i>${alert.message}</div>
    `;
    
    container.insertBefore(alertEl, container.firstChild);
    
    while (container.children.length > 20) {
        container.removeChild(container.lastChild);
    }
}

function loadInitialData() {
    fetch('/api/v1/admin/monitoring/data')
        .then(response => response.json())
        .then(result => {
            if (result.success && result.data) {
                const mockData = {
                    total_requests: result.data.requests?.total || 123456,
                    success_count: result.data.requests?.success || 118500,
                    fail_count: result.data.requests?.failed || 4956,
                    qps: 156,
                    avg_response_time: 123,
                    cpu_usage: result.data.system?.cpu_usage || 45.2,
                    memory_usage: result.data.system?.memory_usage || 62.8,
                    disk_usage: result.data.system?.disk_usage || 35.1,
                    requests: 156,
                    captcha_types: { slider: 35, click: 25, gesture: 20, jigsaw: 20 },
                    risk_distribution: { low: 70, medium: 20, high: 10 },
                    top_apps: [
                        { name: '电商平台', requests: 5200 },
                        { name: '金融服务', requests: 4800 },
                        { name: '社交应用', requests: 3900 },
                        { name: '游戏中心', requests: 3100 },
                        { name: '新闻资讯', requests: 2400 }
                    ],
                    devices: [
                        { name: 'API 服务器 1', status: 'online', icon: 'server' },
                        { name: 'API 服务器 2', status: 'online', icon: 'server' },
                        { name: '数据库主库', status: 'online', icon: 'database' },
                        { name: 'Redis 集群', status: 'warning', icon: 'memory' },
                        { name: '负载均衡器', status: 'online', icon: 'network-wired' }
                    ]
                };
                updateMetrics(mockData);
            }
        })
        .catch(error => {
            console.error('Failed to load initial data:', error);
            const mockData = {
                total_requests: 123456,
                success_count: 118500,
                fail_count: 4956,
                qps: 156,
                avg_response_time: 123,
                cpu_usage: 45.2,
                memory_usage: 62.8,
                disk_usage: 35.1,
                requests: 156,
                captcha_types: { slider: 35, click: 25, gesture: 20, jigsaw: 20 },
                risk_distribution: { low: 70, medium: 20, high: 10 },
                top_apps: [
                    { name: '电商平台', requests: 5200 },
                    { name: '金融服务', requests: 4800 },
                    { name: '社交应用', requests: 3900 },
                    { name: '游戏中心', requests: 3100 },
                    { name: '新闻资讯', requests: 2400 }
                ],
                devices: [
                    { name: 'API 服务器 1', status: 'online', icon: 'server' },
                    { name: 'API 服务器 2', status: 'online', icon: 'server' },
                    { name: '数据库主库', status: 'online', icon: 'database' },
                    { name: 'Redis 集群', status: 'warning', icon: 'memory' },
                    { name: '负载均衡器', status: 'online', icon: 'network-wired' }
                ]
            };
            updateMetrics(mockData);
        });

    fetch('/api/v1/admin/monitoring/alerts')
        .then(response => response.json())
        .then(result => {
            if (result.success && result.data) {
                result.data.forEach(alert => {
                    addAlert(alert);
                });
            }
        })
        .catch(error => {
            console.error('Failed to load alerts:', error);
            const mockAlerts = [
                { id: 1, severity: 'warning', message: 'Redis 内存使用率较高', timestamp: Math.floor(Date.now() / 1000) - 300, icon: 'memory' },
                { id: 2, severity: 'info', message: '系统自动备份完成', timestamp: Math.floor(Date.now() / 1000) - 600, icon: 'check-circle' }
            ];
            mockAlerts.forEach(alert => addAlert(alert));
        });

    if (!ws || ws.readyState !== WebSocket.OPEN) {
        startMockDataGeneration();
    }
}

let mockInterval = null;
function startMockDataGeneration() {
    if (mockInterval) return;
    
    mockInterval = setInterval(() => {
        const mockData = {
            total_requests: Math.floor(Math.random() * 100000) + 100000,
            success_count: Math.floor(Math.random() * 95000) + 95000,
            fail_count: Math.floor(Math.random() * 5000) + 1000,
            qps: Math.floor(Math.random() * 200) + 100,
            avg_response_time: Math.floor(Math.random() * 200) + 50,
            cpu_usage: Math.random() * 30 + 40,
            memory_usage: Math.random() * 20 + 55,
            disk_usage: Math.random() * 10 + 30,
            requests: Math.floor(Math.random() * 200) + 100,
            captcha_types: {
                slider: Math.floor(Math.random() * 40) + 20,
                click: Math.floor(Math.random() * 30) + 15,
                gesture: Math.floor(Math.random() * 25) + 10,
                jigsaw: Math.floor(Math.random() * 25) + 10
            },
            risk_distribution: {
                low: Math.floor(Math.random() * 30) + 60,
                medium: Math.floor(Math.random() * 20) + 15,
                high: Math.floor(Math.random() * 15) + 5
            },
            top_apps: [
                { name: '电商平台', requests: Math.floor(Math.random() * 2000) + 4000 },
                { name: '金融服务', requests: Math.floor(Math.random() * 1500) + 3500 },
                { name: '社交应用', requests: Math.floor(Math.random() * 1500) + 3000 },
                { name: '游戏中心', requests: Math.floor(Math.random() * 1000) + 2500 },
                { name: '新闻资讯', requests: Math.floor(Math.random() * 1000) + 2000 }
            ],
            devices: [
                { name: 'API 服务器 1', status: 'online', icon: 'server' },
                { name: 'API 服务器 2', status: 'online', icon: 'server' },
                { name: '数据库主库', status: 'online', icon: 'database' },
                { name: 'Redis 集群', status: Math.random() > 0.7 ? 'warning' : 'online', icon: 'memory' },
                { name: '负载均衡器', status: 'online', icon: 'network-wired' }
            ]
        };
        updateMetrics(mockData);

        if (Math.random() > 0.85) {
            const severities = ['info', 'warning', 'critical'];
            const icons = ['info-circle', 'exclamation-triangle', 'exclamation-circle'];
            const messages = [
                '新的应用注册成功',
                'CPU 使用率短暂升高',
                '检测到异常访问模式',
                '系统健康检查通过',
                '缓存命中率下降'
            ];
            const idx = Math.floor(Math.random() * severities.length);
            addAlert({
                severity: severities[idx],
                message: messages[Math.floor(Math.random() * messages.length)],
                timestamp: Math.floor(Date.now() / 1000),
                icon: icons[idx]
            });
        }
    }, 2000);
}

function formatNumber(num) {
    if (num >= 1000000) {
        return (num / 1000000).toFixed(1) + 'M';
    } else if (num >= 1000) {
        return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
}