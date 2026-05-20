let realtimeManager = {
    ws: null,
    isConnected: false,
    reconnectAttempts: 0,
    maxReconnectAttempts: 10,
    reconnectDelay: 3000,
    dataBuffer: [],
    maxBufferSize: 100,
    updateInterval: null,
    lastUpdateTime: null,
    stats: {
        totalRequests: 0,
        passRate: 0,
        failRate: 0,
        blockedRequests: 0,
        avgResponseTime: 0,
        currentQPS: 0
    }
};

const WS_UPDATE_INTERVAL = 5000;
const POLLING_INTERVAL = 5000;

function initRealtimeManager() {
    initWebSocketConnection();
    initPollingFallback();
    setupRealtimeEventListeners();
    startRealtimeUpdates();
}

function initWebSocketConnection() {
    if (typeof WebSocket === 'undefined') {
        console.warn('WebSocket 不可用，切换到轮询模式');
        return;
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/v1/admin/dashboard/ws`;

    try {
        realtimeManager.ws = new WebSocket(wsUrl);

        realtimeManager.ws.onopen = function() {
            realtimeManager.isConnected = true;
            realtimeManager.reconnectAttempts = 0;
            updateConnectionStatus(true);
            console.log('实时数据连接已建立');
            sendPing();
        };

        realtimeManager.ws.onmessage = function(event) {
            try {
                const data = JSON.parse(event.data);
                handleRealtimeMessage(data);
            } catch (e) {
                console.error('解析实时数据失败:', e);
            }
        };

        realtimeManager.ws.onerror = function(error) {
            console.error('WebSocket 错误:', error);
            realtimeManager.isConnected = false;
            updateConnectionStatus(false);
        };

        realtimeManager.ws.onclose = function() {
            realtimeManager.isConnected = false;
            updateConnectionStatus(false);
            console.log(`WebSocket 连接已断开，${realtimeManager.reconnectDelay / 1000}秒后尝试重连...`);
            attemptReconnect();
        };

    } catch (e) {
        console.error('WebSocket 初始化失败:', e);
        realtimeManager.isConnected = false;
        updateConnectionStatus(false);
    }
}

function attemptReconnect() {
    if (realtimeManager.reconnectAttempts >= realtimeManager.maxReconnectAttempts) {
        console.error('达到最大重连次数，切换到轮询模式');
        return;
    }

    realtimeManager.reconnectAttempts++;
    const delay = realtimeManager.reconnectDelay * Math.pow(1.5, realtimeManager.reconnectAttempts - 1);

    console.log(`第 ${realtimeManager.reconnectAttempts} 次重连尝试，延迟 ${delay / 1000} 秒...`);

    setTimeout(() => {
        if (!realtimeManager.isConnected) {
            initWebSocketConnection();
        }
    }, delay);
}

function sendPing() {
    if (realtimeManager.ws && realtimeManager.ws.readyState === WebSocket.OPEN) {
        realtimeManager.ws.send(JSON.stringify({ type: 'ping' }));
    }
}

function handleRealtimeMessage(data) {
    realtimeManager.lastUpdateTime = new Date();

    if (data.type === 'metrics') {
        updateRealtimeStats(data.payload);
    } else if (data.type === 'verification') {
        addVerificationRecord(data.payload);
    } else if (data.type === 'alert') {
        handleAlert(data.payload);
    } else if (data.type === 'pong') {
        console.log('收到服务器 Pong 响应');
    }

    addToBuffer(data);
    updateDashboardUI();
}

function updateRealtimeStats(stats) {
    if (!stats) return;

    realtimeManager.stats.totalRequests = stats.total_requests || 0;
    realtimeManager.stats.passRate = stats.pass_rate || 0;
    realtimeManager.stats.failRate = stats.fail_rate || 0;
    realtimeManager.stats.blockedRequests = stats.blocked_requests || 0;
    realtimeManager.stats.avgResponseTime = stats.avg_response_time || 0;
    realtimeManager.stats.currentQPS = stats.requests_per_second || 0;

    updateStatsDisplay();
    updateProgressBars();
    updateRealtimeChart(stats.requests_per_second || 0);
}

function updateStatsDisplay() {
    const elements = {
        'totalRequests': realtimeManager.stats.totalRequests,
        'passRate': realtimeManager.stats.passRate.toFixed(1),
        'failRate': realtimeManager.stats.failRate.toFixed(1),
        'avgResponseTime': realtimeManager.stats.avgResponseTime
    };

    Object.entries(elements).forEach(([id, value]) => {
        const element = document.getElementById(id);
        if (element) {
            if (id === 'passRate' || id === 'failRate') {
                element.textContent = value + '%';
            } else if (id === 'avgResponseTime') {
                element.textContent = value + 'ms';
            } else {
                element.textContent = formatNumber(value);
            }
        }
    });

    const currentQPSEl = document.getElementById('currentQPS');
    if (currentQPSEl) {
        currentQPSEl.textContent = realtimeManager.stats.currentQPS.toFixed(0) + ' QPS';
    }

    const currentQPSDisplay = document.getElementById('currentQPSDisplay');
    if (currentQPSDisplay) {
        currentQPSDisplay.textContent = formatNumber(realtimeManager.stats.currentQPS.toFixed(0));
    }
}

function updateProgressBars() {
    const passRate = realtimeManager.stats.passRate;
    const passRateProgressBar = document.getElementById('passRateProgressBar');
    if (passRateProgressBar) {
        passRateProgressBar.style.width = Math.min(passRate, 100) + '%';
    }

    const passRateProgress = document.getElementById('passRateProgress');
    if (passRateProgress) {
        passRateProgress.style.width = Math.min(passRate, 100) + '%';
    }

    const blockRate = (realtimeManager.stats.blockedRequests / realtimeManager.stats.totalRequests * 100) || 0;
    const blockRateProgressBar = document.getElementById('blockRateProgressBar');
    if (blockRateProgressBar) {
        blockRateProgressBar.style.width = Math.min(blockRate, 100) + '%';
    }

    const responseTimeProgress = document.getElementById('responseTimeProgress');
    if (responseTimeProgress) {
        responseTimeProgress.style.width = Math.min(realtimeManager.stats.avgResponseTime / 2, 100) + '%';
    }
}

function updateRealtimeChart(qps) {
    if (typeof Chart !== 'undefined' && window.dashboardCharts && window.dashboardCharts.instances && window.dashboardCharts.instances.realtime) {
        return;
    }

    if (typeof echarts !== 'undefined' && window.realtimeChart) {
        const now = new Date();
        const timeLabel = now.toLocaleTimeString('zh-CN', {
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit'
        });

        window.realtimeDataPoints = window.realtimeDataPoints || [];
        window.realtimeDataPoints.push({ time: timeLabel, value: qps });

        if (window.realtimeDataPoints.length > 60) {
            window.realtimeDataPoints.shift();
        }

        window.realtimeChart.setOption({
            xAxis: {
                data: window.realtimeDataPoints.map(p => p.time)
            },
            series: [{
                data: window.realtimeDataPoints.map(p => p.value)
            }]
        }, false);
    }
}

function addVerificationRecord(record) {
    const tbody = document.getElementById('recentVerifications');
    if (!tbody || tbody.rows.length >= 10) return;

    const row = tbody.insertRow(0);
    const time = new Date(record.timestamp || Date.now()).toLocaleTimeString('zh-CN');
    const statusClass = record.status === 'success' ? 'badge-success' : 'badge-danger';
    const statusText = record.status === 'success' ? '成功' : '失败';

    row.innerHTML = `
        <td><small>${time}</small></td>
        <td><small>${escapeHtml(record.app || '-')}</small></td>
        <td><small>${escapeHtml(record.type || '-')}</small></td>
        <td><span class="badge ${statusClass}">${statusText}</span></td>
        <td><small>${record.response_time || 0}ms</small></td>
    `;

    row.className = 'fade-in';
    row.style.animation = 'fadeIn 0.5s ease';

    if (tbody.rows.length > 10) {
        tbody.deleteRow(10);
    }
}

function handleAlert(alert) {
    const type = alert.type || 'info';
    const message = alert.message || '收到新通知';
    const title = alert.title || '提醒';

    showRealtimeAlert(title, message, type);
}

function showRealtimeAlert(title, message, type = 'info') {
    const alertClass = {
        info: 'alert-info',
        warning: 'alert-warning',
        error: 'alert-danger',
        success: 'alert-success'
    }[type] || 'alert-info';

    const alertContainer = document.getElementById('alertsContainer') || createAlertContainer();

    const alertHtml = `
        <div class="alert ${alertClass} alert-dismissible fade show" role="alert">
            <strong>${escapeHtml(title)}</strong> ${escapeHtml(message)}
            <button type="button" class="close" data-dismiss="alert" aria-label="关闭">
                <span aria-hidden="true">&times;</span>
            </button>
        </div>
    `;

    alertContainer.insertAdjacentHTML('beforeend', alertHtml);

    setTimeout(() => {
        const alerts = alertContainer.querySelectorAll('.alert');
        if (alerts.length > 5) {
            alerts[0].remove();
        }
    }, 10000);
}

function createAlertContainer() {
    const container = document.createElement('div');
    container.id = 'alertsContainer';
    container.className = 'alerts-container position-fixed top-0 right-0 p-3';
    container.style.zIndex = '9999';
    container.style.maxWidth = '400px';
    document.body.appendChild(container);
    return container;
}

function initPollingFallback() {
    if (realtimeManager.isConnected) return;

    console.log('初始化轮询模式作为后备');

    realtimeManager.updateInterval = setInterval(async () => {
        await pollForUpdates();
    }, POLLING_INTERVAL);
}

async function pollForUpdates() {
    try {
        const response = await fetch('/admin/api/dashboard/realtime', {
            method: 'GET',
            headers: {
                'Cache-Control': 'no-cache'
            }
        });

        if (!response.ok) throw new Error('轮询请求失败');

        const data = await response.json();
        if (data.code === 0) {
            handleRealtimeMessage({ type: 'metrics', payload: data.data });
        }
    } catch (error) {
        console.error('轮询更新失败:', error);
    }
}

function startRealtimeUpdates() {
    setInterval(() => {
        if (!realtimeManager.isConnected) {
            updateConnectionStatus(false);
        }
    }, 10000);
}

function setupRealtimeEventListeners() {
    const refreshBtn = document.getElementById('refreshBtn');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', async () => {
            await pollForUpdates();
            showToast('数据已更新', 'success');
        });
    }

    document.addEventListener('visibilitychange', () => {
        if (document.hidden) {
            pauseRealtimeUpdates();
        } else {
            resumeRealtimeUpdates();
        }
    });
}

function pauseRealtimeUpdates() {
    if (realtimeManager.ws) {
        realtimeManager.ws.close();
    }
    console.log('实时更新已暂停');
}

function resumeRealtimeUpdates() {
    if (!realtimeManager.isConnected) {
        initWebSocketConnection();
        if (!realtimeManager.isConnected) {
            initPollingFallback();
        }
    }
    console.log('实时更新已恢复');
}

function updateConnectionStatus(connected) {
    const wsStatus = document.getElementById('wsStatus');
    const liveIndicator = document.getElementById('liveIndicator');

    if (wsStatus) {
        if (connected) {
            wsStatus.className = 'badge badge-success';
            wsStatus.innerHTML = '<i class="fas fa-wifi mr-1"></i>已连接';
        } else {
            wsStatus.className = 'badge badge-danger';
            wsStatus.innerHTML = '<i class="fas fa-wifi mr-1"></i>已断开';
        }
    }

    if (liveIndicator) {
        if (connected) {
            liveIndicator.innerHTML = '<i class="fas fa-circle text-success"></i> 实时';
        } else {
            liveIndicator.innerHTML = '<i class="fas fa-circle text-secondary"></i> 轮询';
        }
    }

    realtimeManager.isConnected = connected;
}

function addToBuffer(data) {
    realtimeManager.dataBuffer.push({
        time: new Date(),
        data: data
    });

    if (realtimeManager.dataBuffer.length > realtimeManager.maxBufferSize) {
        realtimeManager.dataBuffer.shift();
    }
}

function updateDashboardUI() {
    if (realtimeManager.lastUpdateTime) {
        const lastUpdateEl = document.getElementById('lastUpdateTime');
        if (lastUpdateEl) {
            lastUpdateEl.textContent = realtimeManager.lastUpdateTime.toLocaleTimeString('zh-CN');
        }
    }
}

function getRealtimeStats() {
    return { ...realtimeManager.stats };
}

function getRealtimeBuffer() {
    return [...realtimeManager.dataBuffer];
}

function getConnectionStatus() {
    return {
        isConnected: realtimeManager.isConnected,
        reconnectAttempts: realtimeManager.reconnectAttempts,
        lastUpdateTime: realtimeManager.lastUpdateTime
    };
}

function showToast(message, type = 'info') {
    const toast = document.createElement('div');
    toast.className = `alert alert-${type} alert-dismissible fade show position-fixed`;
    toast.style.cssText = 'top: 20px; right: 20px; z-index: 9999; min-width: 250px;';
    toast.innerHTML = `
        ${escapeHtml(message)}
        <button type="button" class="close" data-dismiss="alert" aria-label="关闭">
            <span aria-hidden="true">&times;</span>
        </button>
    `;
    document.body.appendChild(toast);

    setTimeout(() => {
        toast.remove();
    }, 3000);
}

function escapeHtml(text) {
    if (text === null || text === undefined) return '';
    const div = document.createElement('div');
    div.textContent = String(text);
    return div.innerHTML;
}

function formatNumber(num) {
    if (num >= 1000000) {
        return (num / 1000000).toFixed(1) + 'M';
    } else if (num >= 1000) {
        return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
}

document.addEventListener('DOMContentLoaded', function() {
    if (document.getElementById('dashboardContent') || document.querySelector('[data-realtime]')) {
        setTimeout(initRealtimeManager, 100);
    }
});

window.realtimeManager = realtimeManager;
window.getRealtimeStats = getRealtimeStats;
window.getRealtimeBuffer = getRealtimeBuffer;
window.getConnectionStatus = getConnectionStatus;
window.pauseRealtimeUpdates = pauseRealtimeUpdates;
window.resumeRealtimeUpdates = resumeRealtimeUpdates;
