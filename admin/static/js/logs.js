let logCurrentPage = 1;
let logPageSize = 20;
let currentLogs = [];
let currentDisplay = 'list';
let autoRefreshInterval = null;
let currentLogForCopy = null;
let selectedLogs = new Set();
const LOG_AUTO_REFRESH_INTERVAL = 30000;

document.addEventListener('DOMContentLoaded', () => {
    initDefaultDates();
    setupEventListeners();
    loadLogs();
    loadLogStatistics();
});

function setupEventListeners() {
    const searchBtn = document.getElementById('searchBtn');
    if (searchBtn) {
        searchBtn.addEventListener('click', () => {
            logCurrentPage = 1;
            loadLogs();
        });
    }

    const resetBtn = document.getElementById('resetSearchBtn');
    if (resetBtn) {
        resetBtn.addEventListener('click', resetFilters);
    }

    const autoRefreshBtn = document.getElementById('autoRefreshLogs');
    if (autoRefreshBtn) {
        autoRefreshBtn.addEventListener('click', toggleAutoRefresh);
    }

    const selectAllCheckbox = document.getElementById('selectAllLogs');
    if (selectAllCheckbox) {
        selectAllCheckbox.addEventListener('change', (e) => {
            const checkboxes = document.querySelectorAll('.log-checkbox');
            checkboxes.forEach(cb => {
                cb.checked = e.target.checked;
                if (e.target.checked) {
                    selectedLogs.add(parseInt(cb.dataset.id));
                } else {
                    selectedLogs.delete(parseInt(cb.dataset.id));
                }
            });
            updateSelectedCount();
        });
    }
}

function toggleAutoRefresh() {
    if (autoRefreshInterval) {
        clearInterval(autoRefreshInterval);
        autoRefreshInterval = null;
        const statusEl = document.getElementById('autoRefreshStatus');
        if (statusEl) statusEl.textContent = '自动刷新';
        showToast('自动刷新已关闭', 'info');
    } else {
        autoRefreshInterval = setInterval(() => {
            loadLogs(false);
        }, LOG_AUTO_REFRESH_INTERVAL);
        const statusEl = document.getElementById('autoRefreshStatus');
        if (statusEl) statusEl.textContent = '已开启';
        showToast('自动刷新已开启（每30秒）', 'info');
    }
}

function resetFilters() {
    document.getElementById('filterStatus').value = '';
    document.getElementById('filterRiskLevel').value = '';
    document.getElementById('filterCaptchaType').value = '';
    initDefaultDates();
    logCurrentPage = 1;
    loadLogs();
}

async function loadLogStatistics() {
    try {
        const response = await fetch('/admin/api/logs/statistics');
        if (!response.ok) throw new Error('Network error');
        
        const result = await response.json();
        if (result.code === 0) {
            updateLogStatistics(result.data);
        }
    } catch (error) {
        console.error('Load log statistics failed:', error);
    }
}

function updateLogStatistics(stats) {
    const alertEl = document.getElementById('logStatsAlert');
    const infoEl = document.getElementById('logStatsInfo');
    
    if (alertEl && infoEl && stats) {
        alertEl.style.display = 'block';
        infoEl.innerHTML = `总记录: ${stats.total_count || 0} | 成功: ${stats.success_count || 0} | 失败: ${stats.failed_count || 0} | 平均风险评分: ${(stats.avg_risk_score || 0).toFixed(2)}`;
    }
}

function updateSelectedCount() {
    const selectedEl = document.getElementById('selectedLogs');
    if (selectedEl) {
        selectedEl.textContent = selectedLogs.size;
    }
}

function initDefaultDates() {
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

function loadLogs(showLoading = true) {
    const filters = {
        page: logCurrentPage,
        page_size: logPageSize
    };

    const status = document.getElementById('filterStatus')?.value;
    if (status) filters.status = status;

    const riskLevel = document.getElementById('filterRiskLevel')?.value;
    if (riskLevel) filters.risk_level = riskLevel;

    const captchaType = document.getElementById('filterCaptchaType')?.value;
    if (captchaType) filters.captcha_type = captchaType;

    const startDate = document.getElementById('filterStartDate')?.value;
    if (startDate) filters.start_date = startDate;

    const endDate = document.getElementById('filterEndDate')?.value;
    if (endDate) filters.end_date = endDate;

    const params = new URLSearchParams(filters);
    
    fetch(`/admin/api/logs?${params.toString()}`, {
        headers: {
            'Authorization': 'Bearer ' + localStorage.getItem('admin_token')
        }
    })
    .then(response => response.json())
    .then(result => {
        if (result.code === 0) {
            currentLogs = result.data.logs || [];
            renderTable(currentLogs);
            renderPagination(result.data.total || 0);
            updateTotalLogs(result.data.total || 0);
        } else {
            renderTable([]);
            renderPagination(0);
            updateTotalLogs(0);
        }
    })
    .catch(error => {
        console.error('Load logs failed:', error);
        renderTable([]);
        renderPagination(0);
        updateTotalLogs(0);
    });
}

function renderTable(logs) {
    const tbody = document.getElementById('logsTableBody');
    if (!tbody) return;
    tbody.innerHTML = '';

    if (logs.length === 0) {
        tbody.innerHTML = '<tr><td colspan="11" class="text-center">暂无数据</td></tr>';
        return;
    }

    logs.forEach(log => {
        const statusClass = getStatusBadgeClass(log.status || log.result);
        const statusText = getStatusText(log.status || log.result);
        const riskClass = getRiskBadgeClass(log.risk_level);
        const riskText = getRiskLevelText(log.risk_level);

        const sessionId = log.session_id || '';
        const displaySessionId = sessionId.length > 16 ? sessionId.substring(0, 16) + '...' : sessionId;
        
        const captchaTypeText = getCaptchaTypeText(log.captcha_type);

        const row = `
            <tr>
                <td><input type="checkbox" class="log-checkbox" data-id="${log.id}" onchange="toggleLogSelection(${log.id})"></td>
                <td>${log.id}</td>
                <td title="${sessionId}">${displaySessionId}</td>
                <td><span class="badge badge-info">${captchaTypeText}</span></td>
                <td><span class="badge badge-${statusClass}">${statusText}</span></td>
                <td><span class="badge badge-${riskClass}">${riskText}</span></td>
                <td>${(log.risk_score || 0).toFixed(1)}</td>
                <td>${log.ip_address || '-'}</td>
                <td>${log.duration || 0}ms</td>
                <td>${formatDate(log.created_at)}</td>
                <td>
                    <div class="btn-group btn-group-sm">
                        <button class="btn btn-info" onclick="showDetail(${log.id})" title="详情">
                            <i class="fas fa-eye"></i>
                        </button>
                        <button class="btn btn-secondary" onclick="copyLogSession('${sessionId}')" title="复制会话ID">
                            <i class="fas fa-copy"></i>
                        </button>
                    </div>
                </td>
            </tr>
        `;
        tbody.innerHTML += row;
    });
}

function toggleLogSelection(id) {
    if (selectedLogs.has(id)) {
        selectedLogs.delete(id);
    } else {
        selectedLogs.add(id);
    }
    updateSelectedCount();
}

function updateTotalLogs(total) {
    const totalEl = document.getElementById('totalLogs');
    if (totalEl) totalEl.textContent = total;
}

function getCaptchaTypeText(type) {
    const map = {
        'slider': '滑块',
        'click': '点选',
        'image': '图片',
        'voice': '语音',
        'gesture': '手势'
    };
    return map[type] || type || '-';
}

function exportLogs(format) {
    const params = [];
    const status = document.getElementById('filterStatus')?.value;
    if (status) params.push('status=' + encodeURIComponent(status));

    const riskLevel = document.getElementById('filterRiskLevel')?.value;
    if (riskLevel) params.push('risk_level=' + encodeURIComponent(riskLevel));

    const startDate = document.getElementById('filterStartDate')?.value;
    if (startDate) params.push('start_date=' + encodeURIComponent(startDate));

    const endDate = document.getElementById('filterEndDate')?.value;
    if (endDate) params.push('end_date=' + encodeURIComponent(endDate));
    
    params.push('format=' + encodeURIComponent(format));

    const queryString = params.length > 0 ? '?' + params.join('&') : '';
    window.location.href = '/admin/api/logs/export' + queryString;
    showToast(`正在导出${format.toUpperCase()}格式...`, 'info');
}

function copyLogSession(sessionId) {
    navigator.clipboard.writeText(sessionId).then(() => {
        showToast('会话ID已复制', 'success');
    }).catch(err => {
        console.error('Copy failed:', err);
        showToast('复制失败', 'error');
    });
}

function showToast(message, type = 'info') {
    const container = document.getElementById('toastContainer') || createToastContainer();
    const toast = document.createElement('div');
    toast.className = `toast align-items-center text-white bg-${type} border-0`;
    toast.setAttribute('role', 'alert');
    toast.innerHTML = `
        <div class="d-flex">
            <div class="toast-body">${escapeHtml(message)}</div>
            <button type="button" class="btn-close btn-close-white me-2 m-auto" data-bs-dismiss="toast"></button>
        </div>
    `;
    container.appendChild(toast);
    const bsToast = new bootstrap.Toast(toast);
    bsToast.show();
    toast.addEventListener('hidden.bs.toast', () => toast.remove());
}

function createToastContainer() {
    const container = document.createElement('div');
    container.id = 'toastContainer';
    container.className = 'toast-container position-fixed top-0 end-0 p-3';
    container.style.zIndex = '9999';
    document.body.appendChild(container);
    return container;
}

function escapeHtml(text) {
    if (text === null || text === undefined) return '';
    const div = document.createElement('div');
    div.textContent = String(text);
    return div.innerHTML;
}
