let enhancedLogCurrentPage = 1;
let enhancedLogPageSize = 50;
let currentEnhancedLogs = [];
let autoRefreshEnabled = false;
let autoRefreshInterval = null;
let wsConnection = null;
let currentFilters = {};
let searchHistory = [];
let selectedLogs = new Set();

const ENHANCED_CONFIG = {
    AUTO_REFRESH_INTERVAL: 10000,
    MAX_SEARCH_HISTORY: 20,
    MAX_SELECTED_LOGS: 100,
    DEBOUNCE_DELAY: 300,
    ANIMATION_DURATION: 200
};

document.addEventListener('DOMContentLoaded', () => {
    initEnhancedFeatures();
    setupWebSocketConnection();
    setupEventListeners();
    setupKeyboardShortcuts();
    loadLogs();
    loadLogStatistics();
    startAutoRefresh();
});

function initEnhancedFeatures() {
    initDateTimePickers();
    initAdvancedFilters();
    loadSearchHistory();
    initChartVisualization();
}

function setupEventListeners() {
    const searchInput = document.getElementById('enhancedSearchInput');
    if (searchInput) {
        searchInput.addEventListener('input', debounce(performEnhancedSearch, ENHANCED_CONFIG.DEBOUNCE_DELAY));
        searchInput.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                performEnhancedSearch();
            }
        });
    }

    const advancedSearchBtn = document.getElementById('toggleAdvancedSearch');
    if (advancedSearchBtn) {
        advancedSearchBtn.addEventListener('click', toggleAdvancedSearchPanel);
    }

    const realtimeBtn = document.getElementById('realtimeMonitorBtn');
    if (realtimeBtn) {
        realtimeBtn.addEventListener('click', toggleRealtimeMonitor);
    }

    const alertSettingsBtn = document.getElementById('alertSettingsBtn');
    if (alertSettingsBtn) {
        alertSettingsBtn.addEventListener('click', showAlertSettingsModal);
    }

    const exportDropdown = document.querySelector('[data-toggle="dropdown"]');
    if (exportDropdown) {
        document.querySelectorAll('.dropdown-item').forEach(item => {
            item.addEventListener('click', (e) => {
                const format = e.target.closest('a').textContent.trim().toLowerCase();
                exportLogs(format);
            });
        });
    }
}

function setupWebSocketConnection() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/v1/admin/monitoring/ws`;

    try {
        wsConnection = new WebSocket(wsUrl);

        wsConnection.onopen = () => {
            updateConnectionStatus(true);
            console.log('Enhanced WebSocket connected');
        };

        wsConnection.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                handleEnhancedWebSocketMessage(data);
            } catch (e) {
                console.error('Error parsing WebSocket message:', e);
            }
        };

        wsConnection.onclose = () => {
            updateConnectionStatus(false);
            console.log('WebSocket disconnected, reconnecting in 5s...');
            setTimeout(setupWebSocketConnection, 5000);
        };

        wsConnection.onerror = (error) => {
            console.error('WebSocket error:', error);
            updateConnectionStatus(false);
        };
    } catch (error) {
        console.error('Failed to create WebSocket connection:', error);
    }
}

function handleEnhancedWebSocketMessage(data) {
    switch(data.type) {
        case 'initial':
            handleInitialData(data);
            break;
        case 'metrics':
            handleRealTimeMetrics(data.metrics);
            break;
        case 'alert':
            handleNewAlert(data.alert);
            break;
        case 'log':
            handleNewLog(data.log);
            break;
    }
}

function handleInitialData(data) {
    if (data.systemMetrics) {
        updateDashboardMetrics(data.systemMetrics);
    }
    if (data.recentAlerts) {
        updateAlertsList(data.recentAlerts);
    }
}

function handleRealTimeMetrics(metrics) {
    if (!metrics) return;

    const metricsDisplay = document.getElementById('realtimeMetrics');
    if (metricsDisplay) {
        animateValue('cpuMetric', metrics.cpu_usage);
        animateValue('memoryMetric', metrics.memory_usage);
        animateValue('diskMetric', metrics.disk_usage);
        animateValue('requestCountMetric', metrics.request_count);
    }

    updateChartData(metrics);
}

function handleNewAlert(alert) {
    showNotification('warning', `新告警: ${alert.name}`, alert.message);

    const alertBadge = document.getElementById('activeAlertBadge');
    if (alertBadge) {
        const currentCount = parseInt(alertBadge.textContent) || 0;
        alertBadge.textContent = currentCount + 1;
        alertBadge.style.display = 'inline';
    }

    refreshAlertsList();
}

function handleNewLog(log) {
    prependLogToTable(log);
    updateLogCount();
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

function performEnhancedSearch() {
    const searchInput = document.getElementById('enhancedSearchInput');
    if (!searchInput) return;

    const query = searchInput.value.trim();
    if (query) {
        addToSearchHistory(query);
    }

    enhancedLogCurrentPage = 1;
    loadLogs(true);
}

function loadLogs(showLoading = true) {
    const filters = buildFilterParams();

    if (showLoading) {
        showLoadingIndicator();
    }

    const params = new URLSearchParams(filters);

    fetch(`/admin/api/logs/search?${params.toString()}`, {
        headers: {
            'Authorization': 'Bearer ' + localStorage.getItem('admin_token'),
            'Content-Type': 'application/json'
        }
    })
    .then(response => {
        if (!response.ok) throw new Error('Network error');
        return response.json();
    })
    .then(result => {
        if (result.code === 0) {
            currentEnhancedLogs = result.data.results || [];
            renderEnhancedTable(currentEnhancedLogs);
            renderPagination(result.data.total || 0);
            updateLogStatistics(result.data);
            hideLoadingIndicator();
        } else {
            throw new Error(result.message || 'Load failed');
        }
    })
    .catch(error => {
        console.error('Load logs failed:', error);
        showNotification('error', '加载失败', error.message);
        hideLoadingIndicator();
    });
}

function buildFilterParams() {
    const filters = {
        page: enhancedLogCurrentPage,
        page_size: enhancedLogPageSize,
        full_text_search: true
    };

    const searchInput = document.getElementById('enhancedSearchInput');
    if (searchInput && searchInput.value.trim()) {
        filters.query = searchInput.value.trim();
    }

    const status = document.getElementById('filterStatus')?.value;
    if (status) filters.status = status;

    const level = document.getElementById('filterLevel')?.value;
    if (level) filters.level = level;

    const logType = document.getElementById('filterLogType')?.value;
    if (logType) filters.log_type = logType;

    const startDate = document.getElementById('filterStartDate')?.value;
    if (startDate) filters.start_date = startDate;

    const endDate = document.getElementById('filterEndDate')?.value;
    if (endDate) filters.end_date = endDate;

    const userId = document.getElementById('filterUserId')?.value;
    if (userId) filters.user_id = userId;

    const ipAddress = document.getElementById('filterIpAddress')?.value;
    if (ipAddress) filters.ip_address = ipAddress;

    const resourceType = document.getElementById('filterResourceType')?.value;
    if (resourceType) filters.resource_type = resourceType;

    return filters;
}

function renderEnhancedTable(logs) {
    const tbody = document.getElementById('enhancedLogsTableBody');
    if (!tbody) return;

    if (logs.length === 0) {
        tbody.innerHTML = '<tr><td colspan="10" class="text-center">暂无数据</td></tr>';
        return;
    }

    tbody.innerHTML = logs.map(log => createEnhancedLogRow(log)).join('');

    attachRowEventListeners();
}

function createEnhancedLogRow(log) {
    const timestamp = new Date(log.timestamp || log.created_at).toLocaleString('zh-CN');
    const levelClass = getLevelClass(log.level);
    const isSelected = selectedLogs.has(log.id);

    return `
        <tr class="log-row ${isSelected ? 'selected' : ''}" data-id="${log.id}">
            <td>
                <input type="checkbox" class="log-checkbox" data-id="${log.id}" ${isSelected ? 'checked' : ''}>
            </td>
            <td>${timestamp}</td>
            <td><span class="badge badge-${levelClass}">${log.level || 'info'}</span></td>
            <td>${log.log_type || 'unknown'}</td>
            <td>${log.username || '-'}</td>
            <td>${log.ip_address || '-'}</td>
            <td class="log-message" title="${log.action || log.message || ''}">
                ${truncateText(log.action || log.message || '-', 50)}
            </td>
            <td>${log.status || '-'}</td>
            <td>${log.duration || '-'}</td>
            <td>
                <div class="btn-group btn-group-sm">
                    <button class="btn btn-info btn-sm" onclick="viewLogDetails('${log.id}')" title="查看详情">
                        <i class="fas fa-eye"></i>
                    </button>
                    <button class="btn btn-secondary btn-sm" onclick="copyLogToClipboard('${log.id}')" title="复制">
                        <i class="fas fa-copy"></i>
                    </button>
                </div>
            </td>
        </tr>
    `;
}

function attachRowEventListeners() {
    document.querySelectorAll('.log-checkbox').forEach(checkbox => {
        checkbox.addEventListener('change', (e) => {
            const logId = parseInt(e.target.dataset.id);
            if (e.target.checked) {
                selectedLogs.add(logId);
            } else {
                selectedLogs.delete(logId);
            }
            updateSelectedCount();
        });
    });

    document.querySelectorAll('.log-row').forEach(row => {
        row.addEventListener('click', (e) => {
            if (e.target.type !== 'checkbox') {
                viewLogDetails(row.dataset.id);
            }
        });
    });
}

function getLevelClass(level) {
    const levelMap = {
        'info': 'primary',
        'warning': 'warning',
        'error': 'danger',
        'critical': 'danger',
        'debug': 'secondary'
    };
    return levelMap[level?.toLowerCase()] || 'primary';
}

function truncateText(text, maxLength) {
    if (!text) return '-';
    if (text.length <= maxLength) return text;
    return text.substring(0, maxLength) + '...';
}

function viewLogDetails(logId) {
    const log = currentEnhancedLogs.find(l => l.id === logId || l.log_id === logId);
    if (!log) {
        showNotification('error', '未找到日志', '日志详情不存在');
        return;
    }

    showLogDetailModal(log);
}

function showLogDetailModal(log) {
    const modal = document.getElementById('logDetailModal') || createLogDetailModal();
    const content = document.getElementById('logDetailContent');

    const details = `
        <div class="log-detail-section">
            <h6>基本信息</h6>
            <div class="row">
                <div class="col-md-6">
                    <p><strong>日志ID:</strong> ${log.id || log.log_id || '-'}</p>
                    <p><strong>时间戳:</strong> ${new Date(log.timestamp || log.created_at).toLocaleString('zh-CN')}</p>
                    <p><strong>级别:</strong> <span class="badge badge-${getLevelClass(log.level)}">${log.level || 'info'}</span></p>
                    <p><strong>类型:</strong> ${log.log_type || log.category || '-'}</p>
                </div>
                <div class="col-md-6">
                    <p><strong>用户名:</strong> ${log.username || '-'}</p>
                    <p><strong>IP地址:</strong> ${log.ip_address || '-'}</p>
                    <p><strong>用户代理:</strong> ${log.user_agent || '-'}</p>
                    <p><strong>状态:</strong> ${log.status || '-'}</p>
                </div>
            </div>
        </div>

        <div class="log-detail-section">
            <h6>操作详情</h6>
            <p><strong>操作:</strong> ${log.action || log.message || '-'}</p>
            ${log.error_message ? `<p><strong>错误信息:</strong> <span class="text-danger">${log.error_message}</span></p>` : ''}
            ${log.duration ? `<p><strong>耗时:</strong> ${log.duration}ms</p>` : ''}
        </div>

        <div class="log-detail-section">
            <h6>上下文</h6>
            <pre class="bg-light p-3 rounded"><code>${JSON.stringify(log.context || {}, null, 2)}</code></pre>
        </div>

        ${log.metadata ? `
        <div class="log-detail-section">
            <h6>元数据</h6>
            <pre class="bg-light p-3 rounded"><code>${typeof log.metadata === 'string' ? log.metadata : JSON.stringify(log.metadata, null, 2)}</code></pre>
        </div>
        ` : ''}
    `;

    content.innerHTML = details;
    $('#logDetailModal').modal('show');
}

function createLogDetailModal() {
    const modalHTML = `
        <div class="modal fade" id="logDetailModal" tabindex="-1" role="dialog">
            <div class="modal-dialog modal-lg" role="document">
                <div class="modal-content">
                    <div class="modal-header">
                        <h5 class="modal-title">日志详情</h5>
                        <button type="button" class="close" data-dismiss="modal">
                            <span>&times;</span>
                        </button>
                    </div>
                    <div class="modal-body" id="logDetailContent">
                    </div>
                    <div class="modal-footer">
                        <button type="button" class="btn btn-secondary" data-dismiss="modal">关闭</button>
                        <button type="button" class="btn btn-primary" onclick="copyCurrentLog()">复制</button>
                    </div>
                </div>
            </div>
        </div>
    `;
    document.body.insertAdjacentHTML('beforeend', modalHTML);
    return document.getElementById('logDetailModal');
}

function copyLogToClipboard(logId) {
    const log = currentEnhancedLogs.find(l => l.id === logId || l.log_id === logId);
    if (!log) return;

    const text = JSON.stringify(log, null, 2);
    navigator.clipboard.writeText(text).then(() => {
        showNotification('success', '复制成功', '日志已复制到剪贴板');
    }).catch(err => {
        console.error('Copy failed:', err);
        showNotification('error', '复制失败', err.message);
    });
}

function copyCurrentLog() {
    if (currentEnhancedLogs.length > 0) {
        copyLogToClipboard(currentEnhancedLogs[0].id || currentEnhancedLogs[0].log_id);
    }
}

function updateSelectedCount() {
    const countEl = document.getElementById('selectedLogsCount');
    if (countEl) {
        countEl.textContent = selectedLogs.size;
    }
}

function loadSearchHistory() {
    const saved = localStorage.getItem('logSearchHistory');
    if (saved) {
        try {
            searchHistory = JSON.parse(saved);
            renderSearchHistory();
        } catch (e) {
            console.error('Failed to load search history:', e);
        }
    }
}

function addToSearchHistory(query) {
    if (!query || query.trim() === '') return;

    searchHistory = searchHistory.filter(item => item !== query);
    searchHistory.unshift(query);

    if (searchHistory.length > ENHANCED_CONFIG.MAX_SEARCH_HISTORY) {
        searchHistory = searchHistory.slice(0, ENHANCED_CONFIG.MAX_SEARCH_HISTORY);
    }

    localStorage.setItem('logSearchHistory', JSON.stringify(searchHistory));
    renderSearchHistory();
}

function renderSearchHistory() {
    const container = document.getElementById('searchHistoryList');
    if (!container) return;

    if (searchHistory.length === 0) {
        container.innerHTML = '<div class="dropdown-item">暂无搜索历史</div>';
        return;
    }

    container.innerHTML = searchHistory.map(query => `
        <a class="dropdown-item" href="#" onclick="applySearchHistory('${query}')">
            <i class="fas fa-history"></i> ${query}
        </a>
    `).join('');
}

function applySearchHistory(query) {
    const searchInput = document.getElementById('enhancedSearchInput');
    if (searchInput) {
        searchInput.value = query;
        performEnhancedSearch();
    }
}

function toggleAdvancedSearchPanel() {
    const panel = document.getElementById('advancedSearchPanel');
    if (panel) {
        panel.classList.toggle('d-none');
    }
}

function toggleRealtimeMonitor() {
    autoRefreshEnabled = !autoRefreshEnabled;

    const btn = document.getElementById('realtimeMonitorBtn');
    if (btn) {
        if (autoRefreshEnabled) {
            btn.classList.add('btn-success');
            btn.innerHTML = '<i class="fas fa-stop"></i> 停止监控';
            startAutoRefresh();
        } else {
            btn.classList.remove('btn-success');
            btn.innerHTML = '<i class="fas fa-broadcast-tower"></i> 实时监控';
            stopAutoRefresh();
        }
    }
}

function startAutoRefresh() {
    if (autoRefreshInterval) {
        clearInterval(autoRefreshInterval);
    }

    if (autoRefreshEnabled) {
        autoRefreshInterval = setInterval(() => {
            loadLogs(false);
            loadLogStatistics();
        }, ENHANCED_CONFIG.AUTO_REFRESH_INTERVAL);
    }
}

function stopAutoRefresh() {
    if (autoRefreshInterval) {
        clearInterval(autoRefreshInterval);
        autoRefreshInterval = null;
    }
}

function showAlertSettingsModal() {
    const modal = document.getElementById('alertSettingsModal') || createAlertSettingsModal();
    loadAlertRules();
    $('#alertSettingsModal').modal('show');
}

function createAlertSettingsModal() {
    const modalHTML = `
        <div class="modal fade" id="alertSettingsModal" tabindex="-1" role="dialog">
            <div class="modal-dialog modal-lg" role="document">
                <div class="modal-content">
                    <div class="modal-header">
                        <h5 class="modal-title">告警设置</h5>
                        <button type="button" class="close" data-dismiss="modal">
                            <span>&times;</span>
                        </button>
                    </div>
                    <div class="modal-body">
                        <ul class="nav nav-tabs" id="alertTabs" role="tablist">
                            <li class="nav-item">
                                <a class="nav-link active" id="rules-tab" data-toggle="tab" href="#rules">告警规则</a>
                            </li>
                            <li class="nav-item">
                                <a class="nav-link" id="channels-tab" data-toggle="tab" href="#channels">通知渠道</a>
                            </li>
                            <li class="nav-item">
                                <a class="nav-link" id="history-tab" data-toggle="tab" href="#history">历史记录</a>
                            </li>
                        </ul>
                        <div class="tab-content mt-3">
                            <div class="tab-pane fade show active" id="rules">
                                <div id="alertRulesList"></div>
                            </div>
                            <div class="tab-pane fade" id="channels">
                                <div id="alertChannelsList"></div>
                            </div>
                            <div class="tab-pane fade" id="history">
                                <div id="alertHistoryList"></div>
                            </div>
                        </div>
                    </div>
                    <div class="modal-footer">
                        <button type="button" class="btn btn-secondary" data-dismiss="modal">关闭</button>
                    </div>
                </div>
            </div>
        </div>
    `;
    document.body.insertAdjacentHTML('beforeend', modalHTML);
    return document.getElementById('alertSettingsModal');
}

function loadAlertRules() {
    fetch('/admin/api/alerts/rules', {
        headers: {
            'Authorization': 'Bearer ' + localStorage.getItem('admin_token')
        }
    })
    .then(response => response.json())
    .then(result => {
        if (result.code === 0) {
            renderAlertRules(result.data || []);
        }
    })
    .catch(error => {
        console.error('Load alert rules failed:', error);
    });
}

function renderAlertRules(rules) {
    const container = document.getElementById('alertRulesList');
    if (!container) return;

    if (rules.length === 0) {
        container.innerHTML = '<p class="text-muted">暂无告警规则</p>';
        return;
    }

    container.innerHTML = rules.map(rule => `
        <div class="card mb-2">
            <div class="card-body">
                <div class="d-flex justify-content-between align-items-center">
                    <div>
                        <h6>${rule.name}</h6>
                        <small class="text-muted">${rule.description}</small>
                    </div>
                    <div>
                        <span class="badge badge-${getSeverityClass(rule.severity)}">${rule.severity}</span>
                        <button class="btn btn-sm ${rule.enabled ? 'btn-success' : 'btn-secondary'}" onclick="toggleAlertRule('${rule.id}')">
                            ${rule.enabled ? '已启用' : '已禁用'}
                        </button>
                    </div>
                </div>
            </div>
        </div>
    `).join('');
}

function getSeverityClass(severity) {
    const map = {
        'critical': 'danger',
        'warning': 'warning',
        'info': 'info'
    };
    return map[severity?.toLowerCase()] || 'info';
}

function toggleAlertRule(ruleId) {
    fetch(`/admin/api/alerts/rules/${ruleId}/toggle`, {
        method: 'POST',
        headers: {
            'Authorization': 'Bearer ' + localStorage.getItem('admin_token')
        }
    })
    .then(() => {
        loadAlertRules();
        showNotification('success', '更新成功', '告警规则状态已更新');
    })
    .catch(error => {
        console.error('Toggle rule failed:', error);
        showNotification('error', '更新失败', error.message);
    });
}

function initDateTimePickers() {
    const now = new Date();
    const weekAgo = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);

    const formatDate = (date) => {
        const pad = (n) => String(n).padStart(2, '0');
        return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
    };

    const startDateInput = document.getElementById('filterStartDate');
    const endDateInput = document.getElementById('filterEndDate');

    if (startDateInput) startDateInput.value = formatDate(weekAgo);
    if (endDateInput) endDateInput.value = formatDate(now);
}

function initAdvancedFilters() {
    const advancedFilters = document.getElementById('advancedFilters');
    if (advancedFilters) {
        advancedFilters.classList.add('d-none');
    }
}

function initChartVisualization() {
    const ctx = document.getElementById('logTrendChart');
    if (ctx) {
        new Chart(ctx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [{
                    label: '日志数量',
                    data: [],
                    borderColor: '#3b82f6',
                    backgroundColor: 'rgba(59, 130, 246, 0.1)',
                    tension: 0.4
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        display: true,
                        position: 'top'
                    }
                },
                scales: {
                    y: {
                        beginAtZero: true
                    }
                }
            }
        });
    }
}

function updateChartData(metrics) {
    const ctx = document.getElementById('logTrendChart');
    if (!ctx) return;

    const chart = ctx.chart;
    if (!chart) return;

    const now = new Date().toLocaleTimeString('zh-CN');
    chart.data.labels.push(now);
    chart.data.datasets[0].data.push(metrics.request_count || 0);

    if (chart.data.labels.length > 30) {
        chart.data.labels.shift();
        chart.data.datasets[0].data.shift();
    }

    chart.update();
}

function updateDashboardMetrics(metrics) {
    if (!metrics) return;

    if (metrics.cpu_usage !== undefined) {
        animateValue('cpuMetric', metrics.cpu_usage);
    }
    if (metrics.memory_usage !== undefined) {
        animateValue('memoryMetric', metrics.memory_usage);
    }
    if (metrics.disk_usage !== undefined) {
        animateValue('diskMetric', metrics.disk_usage);
    }
}

function animateValue(elementId, newValue) {
    const element = document.getElementById(elementId);
    if (!element) return;

    element.textContent = typeof newValue === 'number' ? newValue.toFixed(1) + '%' : newValue;
}

function showLoadingIndicator() {
    const loader = document.getElementById('logLoader');
    if (loader) {
        loader.style.display = 'block';
    }
}

function hideLoadingIndicator() {
    const loader = document.getElementById('logLoader');
    if (loader) {
        loader.style.display = 'none';
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

function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

function setupKeyboardShortcuts() {
    document.addEventListener('keydown', (e) => {
        if (e.ctrlKey || e.metaKey) {
            switch(e.key) {
                case 'f':
                    e.preventDefault();
                    const searchInput = document.getElementById('enhancedSearchInput');
                    if (searchInput) searchInput.focus();
                    break;
                case 'r':
                    e.preventDefault();
                    loadLogs();
                    break;
                case 'e':
                    e.preventDefault();
                    exportLogs('csv');
                    break;
            }
        }

        if (e.key === 'Escape') {
            const modal = document.querySelector('.modal.show');
            if (modal) {
                $(modal).modal('hide');
            }
        }
    });
}

function exportLogs(format) {
    const params = buildFilterParams();
    delete params.page;
    delete params.page_size;

    const url = `/admin/api/logs/export?${new URLSearchParams(params).toString()}&format=${format}`;

    fetch(url, {
        headers: {
            'Authorization': 'Bearer ' + localStorage.getItem('admin_token')
        }
    })
    .then(response => response.blob())
    .then(blob => {
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `logs_${new Date().toISOString()}.${format}`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        window.URL.revokeObjectURL(url);

        showNotification('success', '导出成功', `日志已导出为${format.toUpperCase()}格式`);
    })
    .catch(error => {
        console.error('Export failed:', error);
        showNotification('error', '导出失败', error.message);
    });
}

function updateLogCount() {
    const countEl = document.getElementById('totalLogsCount');
    if (countEl) {
        countEl.textContent = currentEnhancedLogs.length;
    }
}

function prependLogToTable(log) {
    const tbody = document.getElementById('enhancedLogsTableBody');
    if (!tbody) return;

    const row = document.createElement('tr');
    row.className = 'log-row new-log';
    row.dataset.id = log.id || log.log_id;
    row.innerHTML = createEnhancedLogRow(log);

    tbody.insertBefore(row, tbody.firstChild);

    setTimeout(() => {
        row.classList.remove('new-log');
    }, ENHANCED_CONFIG.ANIMATION_DURATION);

    if (tbody.children.length > enhancedLogPageSize) {
        tbody.removeChild(tbody.lastChild);
    }
}

function refreshAlertsList() {
    fetch('/admin/api/alerts?status=firing', {
        headers: {
            'Authorization': 'Bearer ' + localStorage.getItem('admin_token')
        }
    })
    .then(response => response.json())
    .then(result => {
        if (result.code === 0) {
            updateAlertsList(result.data || []);
        }
    })
    .catch(error => {
        console.error('Refresh alerts failed:', error);
    });
}

function updateAlertsList(alerts) {
    const container = document.getElementById('activeAlertsList');
    if (!container) return;

    if (alerts.length === 0) {
        container.innerHTML = '<p class="text-muted">暂无活跃告警</p>';
        return;
    }

    container.innerHTML = alerts.map(alert => `
        <div class="alert-item alert-${alert.severity}">
            <div class="d-flex justify-content-between">
                <div>
                    <strong>${alert.name}</strong>
                    <p class="mb-0">${alert.message}</p>
                    <small class="text-muted">${new Date(alert.timestamp).toLocaleString('zh-CN')}</small>
                </div>
                <button class="btn btn-sm btn-primary" onclick="resolveAlert('${alert.id}')">
                    处理
                </button>
            </div>
        </div>
    `).join('');
}

function resolveAlert(alertId) {
    fetch(`/admin/api/alerts/${alertId}/resolve`, {
        method: 'POST',
        headers: {
            'Authorization': 'Bearer ' + localStorage.getItem('admin_token')
        }
    })
    .then(() => {
        refreshAlertsList();
        showNotification('success', '处理成功', '告警已处理');
    })
    .catch(error => {
        console.error('Resolve alert failed:', error);
        showNotification('error', '处理失败', error.message);
    });
}

function renderPagination(total) {
    const pagination = document.getElementById('enhancedPagination');
    if (!pagination) return;

    const totalPages = Math.ceil(total / enhancedLogPageSize);

    if (totalPages <= 1) {
        pagination.innerHTML = '';
        return;
    }

    let html = '';

    if (enhancedLogCurrentPage > 1) {
        html += `<button class="btn btn-sm btn-default" onclick="goToPage(${enhancedLogCurrentPage - 1})">上一页</button>`;
    }

    for (let i = 1; i <= totalPages; i++) {
        if (i === 1 || i === totalPages || (i >= enhancedLogCurrentPage - 2 && i <= enhancedLogCurrentPage + 2)) {
            html += `<button class="btn btn-sm ${i === enhancedLogCurrentPage ? 'btn-primary' : 'btn-default'}" onclick="goToPage(${i})">${i}</button>`;
        } else if (i === enhancedLogCurrentPage - 3 || i === enhancedLogCurrentPage + 3) {
            html += '<span class="btn btn-sm btn-default disabled">...</span>';
        }
    }

    if (enhancedLogCurrentPage < totalPages) {
        html += `<button class="btn btn-sm btn-default" onclick="goToPage(${enhancedLogCurrentPage + 1})">下一页</button>`;
    }

    pagination.innerHTML = html;
}

function goToPage(page) {
    enhancedLogCurrentPage = page;
    loadLogs();
}

function loadLogStatistics() {
    fetch('/admin/api/logs/statistics', {
        headers: {
            'Authorization': 'Bearer ' + localStorage.getItem('admin_token')
        }
    })
    .then(response => response.json())
    .then(result => {
        if (result.code === 0) {
            updateLogStatistics(result.data);
        }
    })
    .catch(error => {
        console.error('Load statistics failed:', error);
    });
}

function updateLogStatistics(stats) {
    if (!stats) return;

    const elements = {
        successCount: stats.success_count,
        failedCount: stats.failed_count,
        blockedCount: stats.blocked_count,
        avgDuration: stats.avg_duration ? `${stats.avg_duration.toFixed(0)}ms` : '0ms',
        avgRiskScore: stats.avg_risk_score ? stats.avg_risk_score.toFixed(1) : '0.0',
        lastUpdate: new Date().toLocaleTimeString('zh-CN')
    };

    Object.keys(elements).forEach(id => {
        const el = document.getElementById(id);
        if (el) {
            el.textContent = elements[id];
        }
    });
}
