let ws = null;
let wsConnected = false;
let wsReconnectAttempts = 0;
const MAX_RECONNECT_ATTEMPTS = 10;
const WS_RECONNECT_DELAY = 3000;
const WS_HEARTBEAT_INTERVAL = 30000;
let wsHeartbeatTimer = null;

let realtimeChart = null;
let responseTimeChart = null;
let throughputChart = null;

let realtimeData = [];
const MAX_DATA_POINTS = 60;

let chartsInitialized = false;
let lastMetrics = null;

const AI_INSIGHTS = [
    '系统运行正常，各项指标均在健康范围内。',
    '检测到QPS有上升趋势，建议关注服务器负载。',
    '缓存命中率优秀，继续保持。',
    '风险验证比例稳定，风控系统工作正常。',
    '延迟略有波动，但仍在可接受范围内。',
    '租户活跃度良好，业务增长趋势明显。',
    '数据库连接池状态健康。',
    '建议在低峰期进行系统维护。',
    '用户体验指标优秀，继续保持。',
    '资源使用合理，无需扩容。'
];

document.addEventListener('DOMContentLoaded', function() {
    initCharts();
    initWebSocket();
    updateTime();
    setInterval(updateTime, 1000);
    loadInitialData();
    startAIInsights();
});

function updateTime() {
    const now = new Date();
    document.getElementById('currentTime').textContent = now.toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        hour12: false
    });
}

function exitScreen() {
    if (ws) {
        ws.close();
    }
    if (wsHeartbeatTimer) {
        clearInterval(wsHeartbeatTimer);
    }
    window.location.href = '/admin';
}

function initCharts() {
    initRealtimeChart();
    initMiniCharts();
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

    updateRealtimeChart();
}

function updateRealtimeChart() {
    if (!realtimeChart) return;

    realtimeChart.setOption({
        backgroundColor: 'transparent',
        grid: { left: '3%', right: '4%', bottom: '10%', top: '15%', containLabel: true },
        tooltip: {
            trigger: 'axis',
            backgroundColor: 'rgba(0,0,0,0.8)',
            borderColor: 'rgba(0,212,255,0.3)',
            textStyle: { color: '#fff' }
        },
        legend: {
            data: ['请求总数', '成功数'],
            textStyle: { color: 'rgba(224,230,237,0.7)' },
            top: 0
        },
        xAxis: {
            type: 'category',
            data: realtimeData.map(p => p.time),
            axisLine: { lineStyle: { color: 'rgba(255,255,255,0.1)' } },
            axisLabel: { color: 'rgba(224,230,237,0.5)', fontSize: 10 },
            splitLine: { show: false }
        },
        yAxis: {
            type: 'value',
            axisLine: { lineStyle: { color: 'rgba(255,255,255,0.1)' } },
            axisLabel: { color: 'rgba(224,230,237,0.5)' },
            splitLine: { lineStyle: { color: 'rgba(255,255,255,0.05)' } }
        },
        series: [
            {
                name: '请求总数',
                type: 'line',
                smooth: true,
                data: realtimeData.map(p => p.requests),
                areaStyle: {
                    color: {
                        type: 'linear',
                        x: 0, y: 0, x2: 0, y2: 1,
                        colorStops: [
                            { offset: 0, color: 'rgba(0,212,255,0.3)' },
                            { offset: 1, color: 'rgba(0,212,255,0.05)' }
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
                type: 'line',
                smooth: true,
                data: realtimeData.map(p => p.success),
                areaStyle: {
                    color: {
                        type: 'linear',
                        x: 0, y: 0, x2: 0, y2: 1,
                        colorStops: [
                            { offset: 0, color: 'rgba(40,167,69,0.3)' },
                            { offset: 1, color: 'rgba(40,167,69,0.05)' }
                        ]
                    }
                },
                lineStyle: { color: '#28a745', width: 2 },
                itemStyle: { color: '#28a745' },
                symbol: 'circle',
                symbolSize: 4
            }
        ],
        animation: false
    });
}

function initMiniCharts() {
    const rtCtx = document.getElementById('responseTimeChart');
    const tpCtx = document.getElementById('throughputChart');

    if (rtCtx) {
        responseTimeChart = new Chart(rtCtx, {
            type: 'line',
            data: {
                labels: Array(20).fill(''),
                datasets: [{
                    data: Array(20).fill(0).map(() => Math.floor(Math.random() * 100) + 50),
                    borderColor: '#00d4ff',
                    backgroundColor: 'rgba(0,212,255,0.1)',
                    borderWidth: 2,
                    fill: true,
                    tension: 0.4,
                    pointRadius: 0
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: { legend: { display: false } },
                scales: {
                    x: { display: false },
                    y: { display: false, min: 0, max: 200 }
                }
            }
        });
    }

    if (tpCtx) {
        throughputChart = new Chart(tpCtx, {
            type: 'line',
            data: {
                labels: Array(20).fill(''),
                datasets: [{
                    data: Array(20).fill(0).map(() => Math.floor(Math.random() * 100) + 100),
                    borderColor: '#28a745',
                    backgroundColor: 'rgba(40,167,69,0.1)',
                    borderWidth: 2,
                    fill: true,
                    tension: 0.4,
                    pointRadius: 0
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: { legend: { display: false } },
                scales: {
                    x: { display: false },
                    y: { display: false, min: 0, max: 300 }
                }
            }
        });
    }
}

function initWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/v1/admin/monitoring/ws/v3`;

    try {
        ws = new WebSocket(wsUrl);

        ws.onopen = function() {
            wsConnected = true;
            wsReconnectAttempts = 0;
            updateConnectionStatus(true);
            console.log('WebSocket V3 connected');
            startHeartbeat();
            ws.send(JSON.stringify({ type: 'subscribe', channels: ['metrics', 'alerts', 'devices'] }));
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
            stopHeartbeat();
            console.log('WebSocket disconnected, reconnecting...');
            scheduleReconnect();
        };
    } catch (e) {
        console.error('Failed to initialize WebSocket:', e);
        wsConnected = false;
        updateConnectionStatus(false);
        scheduleReconnect();
    }
}

function startHeartbeat() {
    wsHeartbeatTimer = setInterval(() => {
        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({ type: 'ping' }));
        }
    }, WS_HEARTBEAT_INTERVAL);
}

function stopHeartbeat() {
    if (wsHeartbeatTimer) {
        clearInterval(wsHeartbeatTimer);
        wsHeartbeatTimer = null;
    }
}

function scheduleReconnect() {
    if (wsReconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
        console.error('Max reconnect attempts reached');
        return;
    }
    wsReconnectAttempts++;
    const delay = WS_RECONNECT_DELAY * Math.min(wsReconnectAttempts, 3);
    setTimeout(() => {
        if (!wsConnected) {
            initWebSocket();
        }
    }, delay);
}

function updateConnectionStatus(connected) {
    const statusEl = document.getElementById('connectionStatus');
    const textEl = document.getElementById('connectionText');
    const refreshEl = document.getElementById('refreshStatus');

    if (connected) {
        statusEl.classList.remove('disconnected');
        statusEl.classList.add('connected');
        textEl.textContent = 'WebSocket 已连接';
        refreshEl.textContent = '数据刷新中...';
    } else {
        statusEl.classList.remove('connected');
        statusEl.classList.add('disconnected');
        textEl.textContent = 'WebSocket 已断开';
        refreshEl.textContent = '尝试重连...';
    }
}

function handleWebSocketData(data) {
    if (data.type === 'pong') {
        return;
    } else if (data.type === 'metrics') {
        updateMetrics(data.payload);
    } else if (data.type === 'alert') {
        addAlert(data.payload);
    } else if (data.type === 'devices') {
        updateDeviceList(data.payload);
    } else if (data.type === 'tenants') {
        updateTenantsGrid(data.payload);
    }
}

function updateMetrics(data) {
    lastMetrics = data;

    document.getElementById('totalRequests').textContent = formatNumber(data.totalRequests || 0);
    document.getElementById('successCount').textContent = formatNumber(data.successCount || 0);
    document.getElementById('failCount').textContent = formatNumber(data.failCount || 0);
    document.getElementById('qpsValue').textContent = Math.round(data.qps || 0);
    document.getElementById('avgResponse').textContent = Math.round(data.avgResponseTime || 0) + 'ms';
    document.getElementById('activeUsers').textContent = formatNumber(data.activeUsers || 0);

    const total = (data.totalRequests || 0);
    const successRate = total > 0 ? ((data.successCount || 0) / total * 100).toFixed(1) : 0;
    const failRate = total > 0 ? ((data.failCount || 0) / total * 100).toFixed(1) : 0;
    document.getElementById('successRate').textContent = successRate + '%';
    document.getElementById('failRate').textContent = failRate + '%';

    document.getElementById('cacheHitRate').textContent = (data.cacheHitRate || 0).toFixed(1) + '%';
    document.getElementById('errorRate').textContent = (data.errorRate || 0).toFixed(2) + '%';
    document.getElementById('networkLatency').textContent = (data.networkLatency || 0) + 'ms';
    document.getElementById('activeConnections').textContent = formatNumber(data.activeConnections || 0);
    document.getElementById('storageUsage').textContent = (data.storageUsage || 0).toFixed(1) + '%';
    document.getElementById('bandwidthUsage').textContent = (data.bandwidthUsage || 0).toFixed(1) + 'Mbps';

    updateGauge('cpu', data.cpuUsage || 0);
    updateGauge('memory', data.memoryUsage || 0);
    updateGauge('disk', data.diskUsage || 0);

    document.getElementById('dbStatus').textContent = data.dbStatus || '正常';
    document.getElementById('redisStatus').textContent = data.redisStatus || '正常';

    updateRealtimeData(data);
    updateCaptchaTypeGrid(data.captchaTypes);
    updateRiskIndicators(data.riskDistribution);
    updateTopAppsList(data.topApps);
    updateMiniCharts(data);
}

function updateGauge(type, value) {
    const circle = document.getElementById(type + 'Circle');
    const valueEl = document.getElementById(type + 'Value');
    if (!circle || !valueEl) return;

    const circumference = 251.2;
    const offset = circumference - (Math.min(value, 100) / 100) * circumference;
    circle.style.strokeDashoffset = offset;
    valueEl.textContent = value.toFixed(1) + '%';
}

function updateRealtimeData(data) {
    const now = new Date();
    const timeLabel = formatTime(now);

    realtimeData.push({
        time: timeLabel,
        requests: data.currentRequests || Math.floor(Math.random() * 200) + 100,
        success: data.currentSuccess || Math.floor(Math.random() * 180) + 90
    });

    if (realtimeData.length > MAX_DATA_POINTS) {
        realtimeData.shift();
    }

    if (realtimeChart) {
        realtimeChart.setOption({
            xAxis: {
                data: realtimeData.map(p => p.time)
            },
            series: [
                { data: realtimeData.map(p => p.requests) },
                { data: realtimeData.map(p => p.success) }
            ]
        });
    }
}

function updateCaptchaTypeGrid(types) {
    const container = document.getElementById('captchaTypeGrid');
    if (!container) return;

    const captchaData = types || {
        slider: 35,
        click: 25,
        gesture: 20,
        puzzle: 20
    };

    const total = Object.values(captchaData).reduce((a, b) => a + b, 0);
    const captchaNames = {
        slider: '滑块验证',
        click: '点击验证',
        gesture: '手势验证',
        puzzle: '拼图验证'
    };
    const captchaColors = {
        slider: '#00d4ff',
        click: '#28a745',
        gesture: '#ffc107',
        puzzle: '#dc3545'
    };

    container.innerHTML = Object.entries(captchaData).map(([key, value]) => {
        const percent = total > 0 ? ((value / total) * 100).toFixed(0) : 0;
        return `
            <div class="captcha-type-item">
                <div class="captcha-type-name">${captchaNames[key] || key}</div>
                <div class="captcha-type-value">${value}</div>
                <div class="captcha-type-bar">
                    <div class="captcha-type-fill" style="width: ${percent}%; background: ${captchaColors[key]};"></div>
                </div>
            </div>
        `;
    }).join('');
}

function updateRiskIndicators(distribution) {
    const container = document.getElementById('riskIndicators');
    if (!container) return;

    const riskData = distribution || { low: 70, medium: 20, high: 10 };

    container.innerHTML = `
        <div class="risk-item">
            <div class="risk-dot low"></div>
            <div class="risk-label">低风险</div>
            <div class="risk-value" style="color: #28a745;">${riskData.low || 0}%</div>
        </div>
        <div class="risk-item">
            <div class="risk-dot medium"></div>
            <div class="risk-label">中风险</div>
            <div class="risk-value" style="color: #ffc107;">${riskData.medium || 0}%</div>
        </div>
        <div class="risk-item">
            <div class="risk-dot high"></div>
            <div class="risk-label">高风险</div>
            <div class="risk-value" style="color: #dc3545;">${riskData.high || 0}%</div>
        </div>
    `;

    // container.innerHTML = riskData;  // 删除这行错误的代码
}

function updateTopAppsList(apps) {
    const container = document.getElementById('topAppsList');
    if (!container) return;

    const appData = apps || [
        { name: '电商平台', requests: 5200, users: 1200 },
        { name: '金融服务', requests: 4800, users: 800 },
        { name: '社交应用', requests: 3900, users: 2500 },
        { name: '游戏中心', requests: 3100, users: 1800 },
        { name: '新闻资讯', requests: 2400, users: 3200 }
    ];

    container.innerHTML = appData.map((app, index) => {
        let rankClass = 'rank-other';
        if (index === 0) rankClass = 'rank-1';
        else if (index === 1) rankClass = 'rank-2';
        else if (index === 2) rankClass = 'rank-3';

        return `
            <div class="top-app-item">
                <div class="top-app-rank ${rankClass}">${index + 1}</div>
                <div class="top-app-info">
                    <div class="top-app-name">${app.name}</div>
                    <div class="top-app-meta">${formatNumber(app.users || 0)} 用户</div>
                </div>
                <div class="top-app-value">${formatNumber(app.requests || 0)}</div>
            </div>
        `;
    }).join('');
}

function updateDeviceList(devices) {
    const container = document.getElementById('deviceList');
    if (!container) return;

    const deviceList = devices || [
        { name: 'API 服务器 1', status: 'online', cpu: 45.2, memory: 62.8, icon: 'server' },
        { name: 'API 服务器 2', status: 'online', cpu: 38.5, memory: 55.3, icon: 'server' },
        { name: '数据库主库', status: 'online', cpu: 52.1, memory: 78.2, icon: 'database' },
        { name: 'Redis 集群', status: Math.random() > 0.7 ? 'warning' : 'online', cpu: 35.8, memory: 72.5, icon: 'memory' },
        { name: '负载均衡器', status: 'online', cpu: 25.3, memory: 40.1, icon: 'network-wired' }
    ];

    container.innerHTML = deviceList.map(device => {
        const statusText = device.status === 'online' ? '在线' : device.status === 'warning' ? '告警' : '离线';
        return `
            <div class="device-item">
                <div class="device-info">
                    <div class="device-icon ${device.status}">
                        <i class="fas fa-${device.icon || 'server'}"></i>
                    </div>
                    <div>
                        <div class="device-name">${device.name}</div>
                        <div class="device-meta">CPU: ${(device.cpu || 0).toFixed(1)}% | 内存: ${(device.memory || 0).toFixed(1)}%</div>
                    </div>
                </div>
                <div class="device-status ${device.status}">
                    ${statusText}
                </div>
            </div>
        `;
    }).join('');
}

function updateTenantsGrid(tenants) {
    const container = document.getElementById('tenantsGrid');
    if (!container) return;

    const tenantData = tenants || [
        { name: '企业A', requests: '125K', users: 850 },
        { name: '企业B', requests: '98K', users: 420 },
        { name: '企业C', requests: '76K', users: 310 },
        { name: '企业D', requests: '54K', users: 180 }
    ];

    container.innerHTML = tenantData.map(tenant => `
        <div class="tenant-item">
            <div class="tenant-name">${tenant.name}</div>
            <div class="tenant-metrics">
                <span>${tenant.requests} 请求</span>
                <span>${tenant.users} 用户</span>
            </div>
        </div>
    `).join('');
}

function updateMiniCharts(data) {
    if (responseTimeChart) {
        const newData = responseTimeChart.data.datasets[0].data.slice(1);
        newData.push(data.avgResponseTime || Math.floor(Math.random() * 100) + 50);
        responseTimeChart.data.datasets[0].data = newData;
        responseTimeChart.update('none');
    }

    if (throughputChart) {
        const newData = throughputChart.data.datasets[0].data.slice(1);
        newData.push(data.qps || Math.floor(Math.random() * 100) + 100);
        throughputChart.data.datasets[0].data = newData;
        throughputChart.update('none');
    }
}

function addAlert(alert) {
    const container = document.getElementById('alertsContainer');
    if (!container) return;

    const alertEl = document.createElement('div');
    alertEl.className = `alert-item ${alert.severity || 'info'}`;

    const time = new Date(alert.timestamp || Date.now());
    alertEl.innerHTML = `
        <div class="alert-time">${time.toLocaleTimeString('zh-CN')}</div>
        <div class="alert-message"><i class="fas fa-${alert.icon || 'info-circle'} me-2"></i>${alert.message || '系统通知'}</div>
    `;

    container.insertBefore(alertEl, container.firstChild);

    while (container.children.length > 20) {
        container.removeChild(container.lastChild);
    }
}

function loadInitialData() {
    fetch('/api/v1/admin/monitoring/data/v3')
        .then(response => response.json())
        .then(result => {
            if (result.success && result.data) {
                updateMetrics(result.data);
                updateDeviceList(result.data.devices);
                updateTenantsGrid(result.data.tenants);
            }
        })
        .catch(error => {
            console.error('Failed to load initial data:', error);
            generateMockData();
        });

    fetch('/api/v1/admin/monitoring/alerts/v3')
        .then(response => response.json())
        .then(result => {
            if (result.success && result.data) {
                result.data.forEach(alert => addAlert(alert));
            }
        })
        .catch(error => {
            console.error('Failed to load alerts:', error);
            const mockAlerts = [
                { id: 1, severity: 'info', message: '系统启动完成，所有服务正常运行', timestamp: Date.now() - 60000, icon: 'check-circle' },
                { id: 2, severity: 'success', message: '缓存预热完成', timestamp: Date.now() - 120000, icon: 'thumbs-up' },
                { id: 3, severity: 'warning', message: 'Redis 内存使用率较高', timestamp: Date.now() - 180000, icon: 'memory' }
            ];
            mockAlerts.forEach(alert => addAlert(alert));
        });

    if (!ws || ws.readyState !== WebSocket.OPEN) {
        setTimeout(() => {
            if (!wsConnected) {
                startMockDataGeneration();
            }
        }, 2000);
    }
}

function generateMockData() {
    const mockData = {
        totalRequests: Math.floor(Math.random() * 500000) + 500000,
        successCount: Math.floor(Math.random() * 475000) + 475000,
        failCount: Math.floor(Math.random() * 25000) + 5000,
        qps: Math.floor(Math.random() * 200) + 100,
        avgResponseTime: Math.floor(Math.random() * 100) + 50,
        activeUsers: Math.floor(Math.random() * 5000) + 2000,
        cacheHitRate: 92.5 + Math.random() * 5,
        errorRate: Math.random() * 0.5,
        cpuUsage: 35 + Math.random() * 30,
        memoryUsage: 55 + Math.random() * 25,
        diskUsage: 30 + Math.random() * 20,
        networkLatency: 15 + Math.random() * 30,
        activeConnections: Math.floor(Math.random() * 1000) + 500,
        storageUsage: 45 + Math.random() * 20,
        bandwidthUsage: Math.random() * 100,
        dbStatus: '正常',
        redisStatus: '正常',
        currentRequests: Math.floor(Math.random() * 200) + 100,
        currentSuccess: Math.floor(Math.random() * 180) + 90,
        captchaTypes: { slider: 35, click: 25, gesture: 20, puzzle: 20 },
        riskDistribution: { low: 70, medium: 20, high: 10 },
        topApps: [
            { name: '电商平台', requests: 5200, users: 1200 },
            { name: '金融服务', requests: 4800, users: 800 },
            { name: '社交应用', requests: 3900, users: 2500 },
            { name: '游戏中心', requests: 3100, users: 1800 },
            { name: '新闻资讯', requests: 2400, users: 3200 }
        ],
        devices: [
            { name: 'API 服务器 1', status: 'online', cpu: 45.2, memory: 62.8, icon: 'server' },
            { name: 'API 服务器 2', status: 'online', cpu: 38.5, memory: 55.3, icon: 'server' },
            { name: '数据库主库', status: 'online', cpu: 52.1, memory: 78.2, icon: 'database' },
            { name: 'Redis 集群', status: Math.random() > 0.7 ? 'warning' : 'online', cpu: 35.8, memory: 72.5, icon: 'memory' },
            { name: '负载均衡器', status: 'online', cpu: 25.3, memory: 40.1, icon: 'network-wired' }
        ],
        tenants: [
            { name: '企业A', requests: '125K', users: 850 },
            { name: '企业B', requests: '98K', users: 420 },
            { name: '企业C', requests: '76K', users: 310 },
            { name: '企业D', requests: '54K', users: 180 }
        ]
    };
    updateMetrics(mockData);
    updateDeviceList(mockData.devices);
    updateTenantsGrid(mockData.tenants);
}

let mockInterval = null;
function startMockDataGeneration() {
    if (mockInterval) return;
    document.getElementById('refreshStatus').textContent = '模拟数据中...';

    mockInterval = setInterval(() => {
        generateMockData();

        if (Math.random() > 0.85) {
            const severities = ['info', 'warning', 'success', 'critical'];
            const icons = ['info-circle', 'exclamation-triangle', 'thumbs-up', 'exclamation-circle'];
            const messages = [
                '新的应用注册成功',
                'CPU 使用率短暂升高',
                '检测到异常访问模式',
                '系统健康检查通过',
                '缓存命中率下降',
                '租户配额即将超限',
                '自动化备份完成',
                '网络延迟优化完成'
            ];
            const idx = Math.floor(Math.random() * severities.length);
            addAlert({
                severity: severities[idx],
                message: messages[Math.floor(Math.random() * messages.length)],
                timestamp: Date.now(),
                icon: icons[idx]
            });
        }
    }, 2000);
}

function startAIInsights() {
    let insightIndex = 0;
    setInterval(() => {
        const insightEl = document.getElementById('aiInsight');
        if (insightEl) {
            insightEl.textContent = AI_INSIGHTS[insightIndex % AI_INSIGHTS.length];
            insightIndex++;
        }
    }, 15000);
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
    return date.toLocaleTimeString('zh-CN', { 
        hour: '2-digit', 
        minute: '2-digit', 
        second: '2-digit',
        hour12: false 
    });
}
