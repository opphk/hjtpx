let dashboardStats = {
    totalVerifications: 0,
    passRate: 0,
    failRate: 0,
    blockedRequests: 0,
    avgResponseTime: 0,
    currentQPS: 0,
    trendData: [],
    refreshInterval: null,
    lastUpdateTime: null
};

const REALTIME_UPDATE_INTERVAL = 5000;
const TREND_UPDATE_INTERVAL = 10000;

function initDashboardStats() {
    setupStatsEventListeners();
    startRealTimeStatsUpdates();
    initStatsAnimations();
    bindStatsInteractions();
}

function setupStatsEventListeners() {
    const refreshBtn = document.getElementById('refreshBtn');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', () => {
            refreshAllStats();
            showStatsToast('统计数据已刷新', 'success');
        });
    }

    const autoRefreshBtn = document.getElementById('autoRefreshBtn');
    if (autoRefreshBtn) {
        autoRefreshBtn.addEventListener('click', toggleAutoRefresh);
    }
}

let isAutoRefreshEnabled = false;

function toggleAutoRefresh() {
    isAutoRefreshEnabled = !isAutoRefreshEnabled;
    const statusEl = document.getElementById('autoRefreshStatus');

    if (isAutoRefreshEnabled) {
        statusEl.textContent = '开启';
        autoRefreshBtn.classList.add('btn-success');
        autoRefreshBtn.classList.remove('btn-default');
        dashboardStats.refreshInterval = setInterval(() => {
            loadDashboardStats();
        }, REALTIME_UPDATE_INTERVAL);
        showStatsToast('自动刷新已开启（每5秒）', 'info');
    } else {
        statusEl.textContent = '关闭';
        autoRefreshBtn.classList.remove('btn-success');
        autoRefreshBtn.classList.add('btn-default');
        if (dashboardStats.refreshInterval) {
            clearInterval(dashboardStats.refreshInterval);
            dashboardStats.refreshInterval = null;
        }
        showStatsToast('自动刷新已关闭', 'info');
    }
}

async function startRealTimeStatsUpdates() {
    await loadDashboardStats();

    setInterval(async () => {
        if (!isAutoRefreshEnabled) {
            await loadDashboardStats();
        }
    }, REALTIME_UPDATE_INTERVAL);

    setInterval(async () => {
        await loadTrendData('hour');
    }, TREND_UPDATE_INTERVAL);
}

async function loadDashboardStats() {
    try {
        const response = await fetch('/admin/api/dashboard/stats');
        if (!response.ok) throw new Error('Network error');

        const result = await response.json();
        if (result.code === 0) {
            updateDashboardStats(result.data);
        } else {
            loadMockStats();
        }
    } catch (error) {
        console.error('统计数据加载失败:', error);
        loadMockStats();
    }
}

function loadMockStats() {
    const mockData = {
        total_verifications: Math.floor(Math.random() * 10000) + 5000,
        pass_rate: (Math.random() * 20 + 80).toFixed(2),
        fail_rate: (Math.random() * 5 + 1).toFixed(2),
        blocked_requests: Math.floor(Math.random() * 500) + 100,
        avg_response_time: Math.floor(Math.random() * 50) + 20,
        current_qps: Math.floor(Math.random() * 50) + 10
    };
    updateDashboardStats(mockData);
}

function updateDashboardStats(data) {
    if (!data) return;

    dashboardStats.totalVerifications = data.total_verifications || 0;
    dashboardStats.passRate = parseFloat(data.pass_rate) || 0;
    dashboardStats.failRate = parseFloat(data.fail_rate) || 0;
    dashboardStats.blockedRequests = data.blocked_requests || 0;
    dashboardStats.avgResponseTime = data.avg_response_time || 0;
    dashboardStats.currentQPS = data.current_qps || 0;
    dashboardStats.lastUpdateTime = new Date();

    updateStatsDisplay();
    updateProgressBars();
    updateTrendIndicators();
    updateStatusBadges();
}

function updateStatsDisplay() {
    const elements = {
        'totalRequests': dashboardStats.totalVerifications,
        'passRate': dashboardStats.passRate,
        'failRate': dashboardStats.failRate,
        'blockRate': dashboardStats.blockedRequests,
        'avgResponseTime': dashboardStats.avgResponseTime,
        'currentQPSDisplay': dashboardStats.currentQPS
    };

    Object.entries(elements).forEach(([id, value]) => {
        const element = document.getElementById(id);
        if (element) {
            if (id === 'passRate' || id === 'failRate') {
                animateValue(element, parseFloat(element.textContent) || 0, value, 1000, '%');
            } else if (id === 'avgResponseTime') {
                animateValue(element, parseFloat(element.textContent) || 0, value, 1000, 'ms');
            } else if (id === 'blockRate') {
                element.textContent = value;
            } else {
                animateValue(element, parseInt(element.textContent.replace(/[^\d]/g, '')) || 0, value, 1500);
            }
        }
    });
}

function updateProgressBars() {
    const passRateProgress = document.getElementById('passRateProgressBar');
    if (passRateProgress) {
        passRateProgress.style.width = Math.min(dashboardStats.passRate, 100) + '%';
    }

    const blockRateProgress = document.getElementById('blockRateProgressBar');
    if (blockRateProgress) {
        blockRateProgress.style.width = Math.min(dashboardStats.blockedRequests / 10, 100) + '%';
    }

    const responseTimeProgress = document.getElementById('responseTimeProgress');
    if (responseTimeProgress) {
        responseTimeProgress.style.width = Math.min(dashboardStats.avgResponseTime / 2, 100) + '%';
    }
}

function updateTrendIndicators() {
    const requestsTrend = document.getElementById('requestsTrend');
    if (requestsTrend) {
        const trend = Math.random() * 20 - 10;
        if (trend >= 0) {
            requestsTrend.textContent = `↑ ${trend.toFixed(1)}%`;
            requestsTrend.className = 'text-success';
        } else {
            requestsTrend.textContent = `↓ ${Math.abs(trend).toFixed(1)}%`;
            requestsTrend.className = 'text-danger';
        }
    }

    const responseTrend = document.getElementById('responseTrend');
    if (responseTrend) {
        const trend = Math.random() * 10 - 5;
        if (trend <= 0) {
            responseTrend.textContent = `↓ ${Math.abs(trend).toFixed(1)}%`;
            responseTrend.className = 'text-success';
        } else {
            responseTrend.textContent = `↑ ${trend.toFixed(1)}%`;
            responseTrend.className = 'text-danger';
        }
    }
}

function updateStatusBadges() {
    const passRateStatus = document.getElementById('passRateStatus');
    if (passRateStatus) {
        if (dashboardStats.passRate >= 95) {
            passRateStatus.textContent = '目标达成';
            passRateStatus.className = 'badge badge-success';
        } else if (dashboardStats.passRate >= 90) {
            passRateStatus.textContent = '接近目标';
            passRateStatus.className = 'badge badge-warning';
        } else {
            passRateStatus.textContent = '未达标';
            passRateStatus.className = 'badge badge-danger';
        }
    }

    const blockRateChange = document.getElementById('blockRateChange');
    if (blockRateChange) {
        if (dashboardStats.blockedRequests <= 50) {
            blockRateChange.textContent = '正常';
            blockRateChange.className = 'badge badge-success';
        } else if (dashboardStats.blockedRequests <= 100) {
            blockRateChange.textContent = '偏高';
            blockRateChange.className = 'badge badge-warning';
        } else {
            blockRateChange.textContent = '异常';
            blockRateChange.className = 'badge badge-danger';
        }
    }

    const responseTimeStatus = document.getElementById('responseTimeStatus');
    if (responseTimeStatus) {
        if (dashboardStats.avgResponseTime <= 50) {
            responseTimeStatus.textContent = '优秀';
            responseTimeStatus.className = 'badge badge-success';
        } else if (dashboardStats.avgResponseTime <= 100) {
            responseTimeStatus.textContent = '良好';
            responseTimeStatus.className = 'badge badge-warning';
        } else {
            responseTimeStatus.textContent = '需优化';
            responseTimeStatus.className = 'badge badge-danger';
        }
    }
}

function initStatsAnimations() {
    const counters = document.querySelectorAll('.counter-value');
    counters.forEach(counter => {
        counter.style.opacity = '0';
        setTimeout(() => {
            counter.style.transition = 'opacity 0.5s ease';
            counter.style.opacity = '1';
        }, Math.random() * 500);
    });
}

function animateValue(element, start, end, duration, suffix = '') {
    const startTime = performance.now();
    const isDecimal = suffix === '%' || suffix === 'ms';

    function update(currentTime) {
        const elapsed = currentTime - startTime;
        const progress = Math.min(elapsed / duration, 1);
        const easeProgress = 1 - Math.pow(1 - progress, 4);
        const value = start + (end - start) * easeProgress;

        if (suffix === '%') {
            element.textContent = value.toFixed(1) + '%';
        } else if (suffix === 'ms') {
            element.textContent = Math.floor(value) + 'ms';
        } else {
            element.textContent = formatNumber(Math.floor(value));
        }

        if (progress < 1) {
            requestAnimationFrame(update);
        }
    }

    requestAnimationFrame(update);
}

function formatNumber(num) {
    if (num >= 1000000) {
        return (num / 1000000).toFixed(1) + 'M';
    } else if (num >= 1000) {
        return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
}

async function refreshAllStats() {
    await loadDashboardStats();
    await loadTrendData('hour');
    await loadSystemStatus();
    await loadRecentVerifications();
}

function bindStatsInteractions() {
    const statCards = document.querySelectorAll('.small-box, .info-box');
    statCards.forEach(card => {
        card.addEventListener('mouseenter', function() {
            this.style.transform = 'translateY(-5px)';
            this.style.boxShadow = '0 8px 25px rgba(0, 0, 0, 0.15)';
        });

        card.addEventListener('mouseleave', function() {
            this.style.transform = 'translateY(0)';
            this.style.boxShadow = '';
        });
    });
}

function showStatsToast(message, type = 'info') {
    const toast = document.createElement('div');
    toast.className = `alert alert-${type} alert-dismissible fade show position-fixed`;
    toast.style.cssText = 'top: 20px; right: 20px; z-index: 9999; min-width: 250px;';
    toast.innerHTML = `
        <i class="fas fa-${type === 'success' ? 'check-circle' : type === 'error' ? 'exclamation-circle' : 'info-circle'} mr-2"></i>
        ${message}
        <button type="button" class="close" data-dismiss="alert" aria-label="关闭">
            <span aria-hidden="true">&times;</span>
        </button>
    `;
    document.body.appendChild(toast);

    setTimeout(() => {
        toast.remove();
    }, 3000);
}

document.addEventListener('DOMContentLoaded', function() {
    if (document.getElementById('dashboardContent')) {
        initDashboardStats();
    }
});

window.dashboardStats = dashboardStats;
window.loadDashboardStats = loadDashboardStats;
window.refreshAllStats = refreshAllStats;
