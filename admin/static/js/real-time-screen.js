let ws = null;
let wsConnected = false;
let realtimeChart = null;
let captchaTypeChart = null;
let riskDistributionChart = null;
let topAppsChart = null;
let realtimeData = [];
const MAX_DATA_POINTS = 60;
let chartsInitialized = false;
const WS_RECONNECT_DELAY = 3000;

document.addEventListener('DOMContentLoaded', function() {
    initECharts();
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

function initECharts() {
    initRealtimeChart();
    initCaptchaTypeChart();
    initRiskDistributionChart();
    initTopAppsChart();
    chartsInitialized = true;
}

function initRealtimeChart() {
    const container = document.getElementById('realtimeChart');
    if (!container) return;

    realtimeChart = echarts.init(container);
    window.addEventListener('resize', () => realtimeChart.resize());

    realtimeData = Array(MAX_DATA_POINTS).fill(0).map((_, i) => ({
        time: formatTime(new Date(Date.now() - (MAX_DATA_POINTS - i) * 1000)),
        requests: Math.floor(Math.random() * 100) + 50,
        success: Math.floor(Math.random() * 90) + 45
    }));

    updateRealtimeChartInit();
}

function updateRealtimeChartInit() {
    if (!realtimeChart) return;

    realtimeChart.setOption({
        xAxis: {
            type: 'category',
            data: realtimeData.map(p => p.time),
            axisLabel: { color: 'rgba(255,255,255,0.5)', rotate: 45 },
            boundaryGap: false
        },
        yAxis: {
            type: 'value',
            axisLabel: { color: 'rgba(255,255,255,0.5)' },
            splitLine: { lineStyle: { color: 'rgba(255,255,255,0.1)' } }
        },
        series: [
            {
                name: '请求数',
                data: realtimeData.map(p => p.requests),
                type: 'line',
                smooth: true,
                areaStyle: {
                    color: {
                        type: 'linear',
                        x: 0, y: 0, x2: 0, y2: 1,
                        colorStops: [
                            { offset: 0, color: 'rgba(0, 212, 255, 0.3)' },
                            { offset: 1, color: 'rgba(0, 212, 255, 0.05)' }
                        ]
                    }
                },
                lineStyle: { color: '#00d4ff', width: 2 },
                itemStyle: { color: '#00d4ff' },
                symbol: 'circle',
                symbolSize: 4
            },
            {
                name: '成功数',
                data: realtimeData.map(p => p.success),
                type: 'line',
                smooth: true,
                areaStyle: {
                    color: {
                        type: 'linear',
                        x: 0, y: 0, x2: 0, y2: 1,
                        colorStops: [
                            { offset: 0, color: 'rgba(40, 167, 69, 0.3)' },
                            { offset: 1, color: 'rgba(40, 167, 69, 0.05)' }
                        ]
                    }
                },
                lineStyle: { color: '#28a745', width: 2 },
                itemStyle: { color: '#28a745' },
                symbol: 'circle',
                symbolSize: 4
            }
        ],
        tooltip: {
            trigger: 'axis',
            backgroundColor: 'rgba(0,0,0,0.8)',
            textStyle: { color: '#fff' }
        },
        legend: {
            data: ['请求数', '成功数'],
            textStyle: { color: 'rgba(255,255,255,0.7)' }
        },
        grid: { left: '3%', right: '4%', bottom: '15%', containLabel: true },
        animation: false
    });
}

function initCaptchaTypeChart() {
    const container = document.getElementById('captchaTypeChart');
    if (!container) return;

    captchaTypeChart = echarts.init(container);
    window.addEventListener('resize', () => captchaTypeChart.resize());

    captchaTypeChart.setOption({
        series: [{
            type: 'pie',
            radius: ['45%', '70%'],
            data: [
                { value: 35, name: '滑块验证' },
                { value: 25, name: '点击验证' },
                { value: 20, name: '手势验证' },
                { value: 20, name: '拼图验证' }
            ],
            label: {
                show: true,
                formatter: '{b}: {c}%',
                color: 'rgba(255,255,255,0.7)'
            },
            emphasis: {
                label: {
                    show: true,
                    fontSize: 14,
                    fontWeight: 'bold'
                }
            }
        }],
        color: ['#00d4ff', '#28a745', '#ffc107', '#dc3545'],
        tooltip: {
            trigger: 'item',
            formatter: '{b}: {c}% ({d}%)'
        },
        legend: {
            orient: 'vertical',
            left: 'left',
            textStyle: { color: 'rgba(255,255,255,0.7)' }
        }
    });
}

function initRiskDistributionChart() {
    const container = document.getElementById('riskDistributionChart');
    if (!container) return;

    riskDistributionChart = echarts.init(container);
    window.addEventListener('resize', () => riskDistributionChart.resize());

    riskDistributionChart.setOption({
        series: [{
            type: 'pie',
            radius: '65%',
            data: [
                { value: 70, name: '低风险' },
                { value: 20, name: '中风险' },
                { value: 10, name: '高风险' }
            ],
            label: {
                show: true,
                formatter: '{b}: {c}%',
                color: 'rgba(255,255,255,0.7)'
            },
            emphasis: {
                label: {
                    show: true,
                    fontSize: 14,
                    fontWeight: 'bold'
                }
            }
        }],
        color: ['#28a745', '#ffc107', '#dc3545'],
        tooltip: {
            trigger: 'item',
            formatter: '{b}: {c}% ({d}%)'
        },
        legend: {
            orient: 'vertical',
            right: 'right',
            textStyle: { color: 'rgba(255,255,255,0.7)' }
        }
    });
}

function initTopAppsChart() {
    const container = document.getElementById('topAppsChart');
    if (!container) return;

    topAppsChart = echarts.init(container);
    window.addEventListener('resize', () => topAppsChart.resize());

    topAppsChart.setOption({
        xAxis: {
            type: 'value',
            axisLabel: { color: 'rgba(255,255,255,0.5)' },
            splitLine: { lineStyle: { color: 'rgba(255,255,255,0.1)' } }
        },
        yAxis: {
            type: 'category',
            data: ['应用A', '应用B', '应用C', '应用D', '应用E'],
            axisLabel: { color: 'rgba(255,255,255,0.5)' },
            axisLine: { lineStyle: { color: 'rgba(255,255,255,0.1)' } }
        },
        series: [{
            type: 'bar',
            data: [5000, 4200, 3800, 2900, 2100],
            itemStyle: {
                color: {
                    type: 'linear',
                    x: 0, y: 0, x2: 1, y2: 0,
                    colorStops: [
                        { offset: 0, color: '#00d4ff' },
                        { offset: 1, color: '#007bff' }
                    ]
                },
                borderRadius: [0, 4, 4, 0]
            },
            barWidth: '60%'
        }],
        tooltip: {
            trigger: 'axis',
            axisPointer: { type: 'shadow' },
            backgroundColor: 'rgba(0,0,0,0.8)',
            textStyle: { color: '#fff' }
        },
        grid: { left: '3%', right: '8%', bottom: '3%', containLabel: true }
    });
}

function initWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/v1/admin/monitoring/ws`;

    try {
        ws = new WebSocket(wsUrl);

        ws.onopen = function() {
            wsConnected = true;
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
            wsConnected = false;
            updateConnectionStatus(false);
        };

        ws.onclose = function() {
            wsConnected = false;
            updateConnectionStatus(false);
            console.log('WebSocket disconnected, reconnecting in', WS_RECONNECT_DELAY, 'ms...');
            setTimeout(initWebSocket, WS_RECONNECT_DELAY);
        };
    } catch (e) {
        console.error('Failed to initialize WebSocket:', e);
        wsConnected = false;
        updateConnectionStatus(false);
        setTimeout(initWebSocket, WS_RECONNECT_DELAY);
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

    updateRealtimeChartData(data);

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
    if (!circle || !valueEl) return;

    const circumference = 251.2;
    const offset = circumference - (value / 100) * circumference;
    circle.style.strokeDashoffset = offset;
    valueEl.textContent = value.toFixed(1) + '%';
}

function updateRealtimeChartData(data) {
    if (!realtimeChart) return;

    const now = new Date();
    const timeLabel = formatTime(now);

    realtimeData.push({
        time: timeLabel,
        requests: data.requests || 0,
        success: data.success_count || 0
    });

    if (realtimeData.length > MAX_DATA_POINTS) {
        realtimeData.shift();
    }

    realtimeChart.setOption({
        xAxis: {
            data: realtimeData.map(p => p.time)
        },
        series: [
            { data: realtimeData.map(p => p.requests) },
            { data: realtimeData.map(p => p.success) }
        ]
    }, false);
}

function updateCaptchaTypeChart(data) {
    if (!captchaTypeChart) return;
    captchaTypeChart.setOption({
        series: [{
            data: [
                { value: data.slider || 0, name: '滑块验证' },
                { value: data.click || 0, name: '点击验证' },
                { value: data.gesture || 0, name: '手势验证' },
                { value: data.jigsaw || 0, name: '拼图验证' }
            ]
        }]
    }, false);
}

function updateRiskDistributionChart(data) {
    if (!riskDistributionChart) return;
    riskDistributionChart.setOption({
        series: [{
            data: [
                { value: data.low || 0, name: '低风险' },
                { value: data.medium || 0, name: '中风险' },
                { value: data.high || 0, name: '高风险' }
            ]
        }]
    }, false);
}

function updateTopAppsChart(data) {
    if (!topAppsChart) return;

    const labels = data.map(item => item.name);
    const values = data.map(item => item.requests);

    topAppsChart.setOption({
        yAxis: {
            data: labels
        },
        series: [{
            data: values
        }]
    }, false);
}

function updateDeviceList(devices) {
    const container = document.getElementById('deviceList');
    if (!container) return;
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
    if (!container) return;

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

function formatTime(date) {
    return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}
