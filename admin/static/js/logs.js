let logCurrentPage = 1;
let logPageSize = 20;
let currentLogs = [];
let currentDisplay = 'list';
let autoRefreshInterval = null;
let currentLogForCopy = null;

document.addEventListener('DOMContentLoaded', () => {
    loadLogsSummary();
    loadLogs();
    setupEventListeners();
    initDefaultDates();
});

function initDefaultDates() {
    const now = new Date();
    const weekAgo = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);

    const formatDate = (date) => {
        const pad = (n) => String(n).padStart(2, '0');
        return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
    };

    const startDateInput = document.getElementById('startDate');
    const endDateInput = document.getElementById('endDate');

    if (startDateInput) startDateInput.value = formatDate(weekAgo);
    if (endDateInput) endDateInput.value = formatDate(now);
}

function setupEventListeners() {
    const searchBtn = document.getElementById('searchLogsBtn');
    if (searchBtn) {
        searchBtn.addEventListener('click', () => {
            logCurrentPage = 1;
            loadLogs();
        });
    }

    const exportBtn = document.getElementById('exportBtn');
    if (exportBtn) {
        exportBtn.addEventListener('click', exportLogs);
    }

    const clearBtn = document.getElementById('clearLogsBtn');
    if (clearBtn) {
        clearBtn.addEventListener('click', () => {
            const modal = new bootstrap.Modal(document.getElementById('clearLogsModal'));
            modal.show();
        });
    }

    const confirmClearBtn = document.getElementById('confirmClearBtn');
    if (confirmClearBtn) {
        confirmClearBtn.addEventListener('click', handleClearLogs);
    }

    const copyLogBtn = document.getElementById('copyLogBtn');
    if (copyLogBtn) {
        copyLogBtn.addEventListener('click', () => {
            if (currentLogForCopy) {
                navigator.clipboard.writeText(JSON.stringify(currentLogForCopy, null, 2))
                    .then(() => showToast('日志已复制到剪贴板', 'success'))
                    .catch(() => showToast('复制失败', 'danger'));
            }
        });
    }

    const autoRefreshSwitch = document.getElementById('autoRefreshLogs');
    if (autoRefreshSwitch) {
        autoRefreshSwitch.addEventListener('change', (e) => {
            if (e.target.checked) {
                startAutoRefresh();
            } else {
                stopAutoRefresh();
            }
        });
    }

    const levelButtons = document.querySelectorAll('[data-level]');
    levelButtons.forEach(btn => {
        btn.addEventListener('click', (e) => {
            levelButtons.forEach(b => b.classList.remove('active'));
            e.target.classList.add('active');

            const level = e.target.dataset.level;
            const levelSelect = document.getElementById('logLevel');
            if (level === 'all') {
                levelSelect.value = '';
            } else {
                levelSelect.value = level;
            }

            logCurrentPage = 1;
            loadLogs();
        });
    });

    const displayButtons = document.querySelectorAll('[data-display]');
    displayButtons.forEach(btn => {
        btn.addEventListener('click', (e) => {
            displayButtons.forEach(b => b.classList.remove('active'));
            e.target.classList.add('active');
            currentDisplay = e.target.dataset.display;
            renderLogs(currentLogs);
        });
    });
}

function startAutoRefresh() {
    stopAutoRefresh();
    autoRefreshInterval = setInterval(() => {
        loadLogs(false);
    }, 10000);
}

function stopAutoRefresh() {
    if (autoRefreshInterval) {
        clearInterval(autoRefreshInterval);
        autoRefreshInterval = null;
    }
}

async function loadLogsSummary() {
    const mockSummary = getMockLogsSummary();

    try {
        const data = await auth.request('/admin/logs/summary');
        if (data.code === 0) {
            updateLogsSummary(data.data);
        } else {
            updateLogsSummary(mockSummary);
        }
    } catch (error) {
        updateLogsSummary(mockSummary);
    }
}

function getMockLogsSummary() {
    return {
        total: 123456,
        errors: 2345,
        warnings: 5678,
        today: 1234
    };
}

function updateLogsSummary(summary) {
    const totalEl = document.getElementById('totalLogCount');
    const errorEl = document.getElementById('errorLogCount');
    const warningEl = document.getElementById('warningLogCount');
    const todayEl = document.getElementById('todayLogCount');

    if (totalEl) totalEl.textContent = formatNumber(summary.total);
    if (errorEl) errorEl.textContent = formatNumber(summary.errors);
    if (warningEl) warningEl.textContent = formatNumber(summary.warnings);
    if (todayEl) todayEl.textContent = formatNumber(summary.today);
}

function formatNumber(num) {
    if (num >= 1000000) {
        return (num / 1000000).toFixed(1) + 'M';
    } else if (num >= 1000) {
        return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
}

async function loadLogs(showLoading = true) {
    const level = document.getElementById('logLevel')?.value || '';
    const source = document.getElementById('logSource')?.value || '';
    const startDate = document.getElementById('startDate')?.value || '';
    const endDate = document.getElementById('endDate')?.value || '';
    const keyword = document.getElementById('keyword')?.value || '';
    const mockLogs = getMockLogs();

    try {
        const params = new URLSearchParams({
            page: logCurrentPage,
            size: logPageSize,
            level, source, startDate, endDate, keyword
        });

        const result = await auth.request(`/admin/logs?${params.toString()}`);
        if (result.code === 0) {
            currentLogs = result.data.list || [];
            renderLogsPagination(result.data.total || currentLogs.length);
            renderLogsCount(result.data.total || currentLogs.length);
        } else {
            currentLogs = filterLogs(mockLogs, level, source, keyword);
            renderLogsPagination(currentLogs.length);
            renderLogsCount(currentLogs.length);
        }
    } catch (error) {
        currentLogs = filterLogs(mockLogs, level, source, keyword);
        renderLogsPagination(currentLogs.length);
        renderLogsCount(currentLogs.length);
    }

    renderLogs(currentLogs);
    if (showLoading) {
        loadLogsSummary();
    }
}

function getMockLogs() {
    const sources = ['auth', 'captcha', 'api', 'db', 'cache'];
    const levels = ['debug', 'info', 'info', 'info', 'warning', 'error'];
    const messages = {
        'auth': ['用户登录成功', '用户登录失败', 'Token验证通过', 'Token已过期', '权限检查通过'],
        'captcha': ['验证码生成成功', '验证码验证成功', '验证码验证失败', '验证码已过期', '验证码类型不支持'],
        'api': ['API请求处理成功', 'API请求参数错误', 'API请求频率超限', 'API认证失败', 'API限流触发'],
        'db': ['数据库连接成功', '数据库查询超时', '数据库事务回滚', '数据库连接池满', '数据库写入成功'],
        'cache': ['缓存命中', '缓存未命中', '缓存写入成功', '缓存过期清理', 'Redis连接成功']
    };

    const logs = [];
    for (let i = 0; i < 50; i++) {
        const source = sources[Math.floor(Math.random() * sources.length)];
        const level = levels[Math.floor(Math.random() * levels.length)];
        const sourceMessages = messages[source];
        const message = sourceMessages[Math.floor(Math.random() * sourceMessages.length)];

        logs.push({
            id: `log_${String(i + 1).padStart(6, '0')}`,
            level: level,
            message: message,
            timestamp: getRandomTime(),
            source: source,
            details: {
                ip: `192.168.${Math.floor(Math.random() * 255)}.${Math.floor(Math.random() * 255)}`,
                userId: `user_${Math.floor(Math.random() * 10000)}`,
                duration: `${Math.floor(Math.random() * 500)}ms`,
                requestId: `req_${Math.random().toString(36).substring(2, 10)}`
            }
        });
    }

    return logs.sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp));
}

function getRandomTime() {
    const now = new Date();
    const offset = Math.floor(Math.random() * 7 * 24 * 60 * 60 * 1000);
    const date = new Date(now.getTime() - offset);

    const pad = (n) => String(n).padStart(2, '0');
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
}

function filterLogs(logs, level, source, keyword) {
    return logs.filter(log => {
        if (level && log.level !== level) return false;
        if (source && log.source !== source) return false;
        if (keyword && !log.message.toLowerCase().includes(keyword.toLowerCase())) return false;
        return true;
    });
}

function renderLogs(logs) {
    const container = document.getElementById('logsList');
    if (!container) return;

    if (logs.length === 0) {
        container.innerHTML = `
            <div class="text-center py-5 text-muted">
                <i class="fas fa-inbox fa-3x mb-3"></i>
                <p>暂无日志记录</p>
            </div>
        `;
        return;
    }

    if (currentDisplay === 'list') {
        renderLogsList(container, logs);
    } else {
        renderLogsCompact(container, logs);
    }
}

function renderLogsList(container, logs) {
    container.innerHTML = `<div class="list-group list-group-flush">${logs.map(log => `
        <div class="list-group-item log-item" onclick="showLogDetail('${log.id}')">
            <div class="d-flex align-items-center justify-content-between">
                <div class="d-flex align-items-center gap-3">
                    <span class="badge ${getLevelBadgeClass(log.level)}">${getLevelText(log.level)}</span>
                    <small class="text-muted">${log.timestamp}</small>
                    <span class="badge bg-secondary">${getSourceText(log.source)}</span>
                </div>
                <div class="d-flex align-items-center gap-2">
                    <small class="text-muted">${escapeHtml(log.message)}</small>
                    <i class="fas fa-chevron-right text-muted"></i>
                </div>
            </div>
        </div>
    `).join('')}</div>`;
}

function renderLogsCompact(container, logs) {
    container.innerHTML = `<div class="p-3">${logs.map(log => `
        <div class="d-flex align-items-start gap-3 py-2 border-bottom log-item" onclick="showLogDetail('${log.id}')" style="cursor: pointer;">
            <span class="badge ${getLevelBadgeClass(log.level)} mt-1">${getLevelText(log.level)}</span>
            <div class="flex-grow-1 min-w-0">
                <div class="d-flex justify-content-between align-items-center mb-1">
                    <small class="text-muted">${log.timestamp}</small>
                    <span class="badge bg-light text-dark">${getSourceText(log.source)}</span>
                </div>
                <div class="text-truncate">${escapeHtml(log.message)}</div>
            </div>
        </div>
    `).join('')}</div>`;
}

function showLogDetail(logId) {
    const log = currentLogs.find(l => l.id === logId);
    if (!log) return;

    currentLogForCopy = log;

    const modal = document.getElementById('logDetailModal');
    const content = document.getElementById('logDetailContent');

    const levelClass = log.level === 'error' ? 'text-danger' : log.level === 'warning' ? 'text-warning' : 'text-info';

    content.innerHTML = `
        <div class="mb-3">
            <div class="d-flex align-items-center gap-3 mb-2">
                <span class="badge ${getLevelBadgeClass(log.level)} fs-6">${getLevelText(log.level)}</span>
                <span class="badge bg-secondary">${getSourceText(log.source)}</span>
                <small class="text-muted ms-auto">${log.id}</small>
            </div>
            <h6 class="${levelClass} mb-3">${escapeHtml(log.message)}</h6>
            <div class="text-muted mb-3">
                <i class="fas fa-clock me-1"></i>${log.timestamp}
            </div>
        </div>

        <div class="card bg-light">
            <div class="card-header py-2">
                <h6 class="mb-0"><i class="fas fa-info-circle me-2"></i>详细信息</h6>
            </div>
            <div class="card-body py-2">
                <pre class="mb-0" style="max-height: 300px; overflow: auto;"><code>${formatJson(log.details)}</code></pre>
            </div>
        </div>

        ${log.level === 'error' || log.level === 'warning' ? `
        <div class="alert alert-warning mt-3 mb-0">
            <i class="fas fa-exclamation-triangle me-1"></i>
            这是一条${log.level === 'error' ? '错误' : '警告'}日志，建议及时处理。
        </div>
        ` : ''}
    `;

    const bsModal = new bootstrap.Modal(modal);
    bsModal.show();
}

function formatJson(obj) {
    try {
        return JSON.stringify(obj, null, 2);
    } catch (e) {
        return String(obj);
    }
}

function renderLogsCount(total) {
    const countEl = document.getElementById('logsCount');
    const pageEl = document.getElementById('currentPage');

    if (countEl) countEl.textContent = formatNumber(total);
    if (pageEl) pageEl.textContent = logCurrentPage;
}

function renderLogsPagination(total) {
    const pagination = document.getElementById('pagination');
    if (!pagination) return;

    const totalPages = Math.ceil(total / logPageSize);
    if (totalPages <= 1) {
        pagination.innerHTML = '';
        return;
    }

    let html = '<div class="d-flex justify-content-between align-items-center">';
    html += `<span class="text-muted">每页 ${logPageSize} 条</span>`;
    html += '<div class="btn-group btn-group-sm">';
    html += `<button class="btn btn-outline-secondary" onclick="changeLogPage(${logCurrentPage - 1})" ${logCurrentPage === 1 ? 'disabled' : ''}>上一页</button>`;

    const startPage = Math.max(1, logCurrentPage - 2);
    const endPage = Math.min(totalPages, logCurrentPage + 2);

    if (startPage > 1) {
        html += `<button class="btn btn-outline-secondary" onclick="changeLogPage(1)">1</button>`;
        if (startPage > 2) html += `<button class="btn btn-outline-secondary" disabled>...</button>`;
    }

    for (let i = startPage; i <= endPage; i++) {
        html += `<button class="btn ${i === logCurrentPage ? 'btn-primary' : 'btn-outline-secondary'}" onclick="changeLogPage(${i})">${i}</button>`;
    }

    if (endPage < totalPages) {
        if (endPage < totalPages - 1) html += `<button class="btn btn-outline-secondary" disabled>...</button>`;
        html += `<button class="btn btn-outline-secondary" onclick="changeLogPage(${totalPages})">${totalPages}</button>`;
    }

    html += `<button class="btn btn-outline-secondary" onclick="changeLogPage(${logCurrentPage + 1})" ${logCurrentPage === totalPages ? 'disabled' : ''}>下一页</button>`;
    html += '</div></div>';

    pagination.innerHTML = html;
}

function changeLogPage(page) {
    logCurrentPage = page;
    loadLogs();
}

function exportLogs() {
    const logsToExport = currentLogs.length > 0 ? currentLogs : getMockLogs();

    const csvContent = [
        ['ID', '级别', '消息', '来源', '时间', '详情'].join(','),
        ...logsToExport.map(log => [
            log.id,
            log.level,
            `"${log.message.replace(/"/g, '""')}"`,
            log.source,
            log.timestamp,
            `"${JSON.stringify(log.details).replace(/"/g, '""')}"`
        ].join(','))
    ].join('\n');

    downloadFile(csvContent, `logs_${new Date().toISOString().slice(0, 10)}.csv`, 'text/csv;charset=utf-8');
    showToast(`成功导出 ${logsToExport.length} 条日志`, 'success');
}

function downloadFile(content, filename, mimeType) {
    const blob = new Blob(['\ufeff' + content], { type: mimeType });
    const link = document.createElement('a');
    const url = URL.createObjectURL(blob);
    link.setAttribute('href', url);
    link.setAttribute('download', filename);
    link.style.visibility = 'hidden';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
}

async function handleClearLogs() {
    const clearRange = document.getElementById('clearRange')?.value || '7d';

    try {
        await auth.request('/admin/logs/clear', {
            method: 'POST',
            body: JSON.stringify({ range: clearRange })
        });
        showToast('日志清理成功', 'success');
        bootstrap.Modal.getInstance(document.getElementById('clearLogsModal'))?.hide();
        loadLogs();
        loadLogsSummary();
    } catch (error) {
        showToast('清理失败', 'danger');
    }
}

function getLevelBadgeClass(level) {
    const map = {
        'debug': 'bg-secondary',
        'info': 'bg-info',
        'warning': 'bg-warning text-dark',
        'error': 'bg-danger'
    };
    return map[level] || 'bg-secondary';
}

function getLevelText(level) {
    const map = {
        'debug': 'DEBUG',
        'info': 'INFO',
        'warning': 'WARNING',
        'error': 'ERROR'
    };
    return map[level] || level.toUpperCase();
}

function getSourceText(source) {
    const map = {
        'auth': '认证服务',
        'captcha': '验证码',
        'api': 'API网关',
        'db': '数据库',
        'cache': '缓存'
    };
    return map[source] || source;
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
