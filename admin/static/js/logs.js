
let currentPage = 1;
const pageSize = 20;
let totalLogs = 0;
let logs = [];
let autoScroll = true;
let selectedLog = null;

document.addEventListener('DOMContentLoaded', function() {
    if (!Auth.requireAuth()) {
        return;
    }

    const user = Auth.getCurrentUser();
    if (user && user.username) {
        Auth.updateUserDisplay(user.username);
    }

    loadApps();
    initDefaultTimeRange();
    loadLogs();
    initEventListeners();
});

function initEventListeners() {
    const searchBtn = document.getElementById('searchLogsBtn');
    if (searchBtn) {
        searchBtn.addEventListener('click', function() {
            currentPage = 1;
            loadLogs();
        });
    }

    const resetBtn = document.getElementById('resetLogsBtn');
    if (resetBtn) {
        resetBtn.addEventListener('click', function() {
            resetFilters();
        });
    }

    const refreshBtn = document.getElementById('refreshLogsBtn');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', function() {
            loadLogs();
            Auth.showToast('日志已刷新', 'success');
        });
    }

    const autoScrollBtn = document.getElementById('autoScrollBtn');
    if (autoScrollBtn) {
        autoScrollBtn.addEventListener('click', function() {
            autoScroll = !autoScroll;
            this.classList.toggle('active', autoScroll);
        });
    }

    const exportBtn = document.getElementById('exportLogsBtn');
    if (exportBtn) {
        exportBtn.addEventListener('click', function() {
            exportLogs();
        });
    }

    const copyBtn = document.getElementById('copyLogBtn');
    if (copyBtn) {
        copyBtn.addEventListener('click', function() {
            copyLogContent();
        });
    }

    document.getElementById('levelFilter').addEventListener('change', function() {
        currentPage = 1;
        loadLogs();
    });

    document.getElementById('appFilter').addEventListener('change', function() {
        currentPage = 1;
        loadLogs();
    });

    let searchTimeout;
    document.getElementById('searchKeyword').addEventListener('input', function() {
        clearTimeout(searchTimeout);
        searchTimeout = setTimeout(() => {
            currentPage = 1;
            loadLogs();
        }, 500);
    });

    document.getElementById('startTime').addEventListener('change', function() {
        currentPage = 1;
        loadLogs();
    });

    document.getElementById('endTime').addEventListener('change', function() {
        currentPage = 1;
        loadLogs();
    });
}

function initDefaultTimeRange() {
    const endTime = new Date();
    const startTime = new Date();
    startTime.setHours(startTime.getHours() - 24);

    document.getElementById('startTime').value = formatDateTimeLocal(startTime);
    document.getElementById('endTime').value = formatDateTimeLocal(endTime);
}

function formatDateTimeLocal(date) {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    const hours = String(date.getHours()).padStart(2, '0');
    const minutes = String(date.getMinutes()).padStart(2, '0');
    return `${year}-${month}-${day}T${hours}:${minutes}`;
}

async function loadApps() {
    try {
        const response = await fetch('/admin/api/applications?pageSize=100');
        if (!response.ok) throw new Error('获取应用列表失败');

        const data = await response.json();
        populateAppFilter(data.apps || []);
    } catch (error) {
        console.error('加载应用列表失败:', error);
        populateAppFilter([
            { id: 1, name: '示例应用1', appId: 'app_001' },
            { id: 2, name: '测试应用', appId: 'app_002' }
        ]);
    }
}

function populateAppFilter(apps) {
    const select = document.getElementById('appFilter');
    select.innerHTML = '<option value="">全部应用</option>';

    apps.forEach(app => {
        const option = document.createElement('option');
        option.value = app.appId;
        option.textContent = app.name;
        select.appendChild(option);
    });
}

async function loadLogs() {
    try {
        const level = document.getElementById('levelFilter').value;
        const appId = document.getElementById('appFilter').value;
        const keyword = document.getElementById('searchKeyword').value;
        const startTime = document.getElementById('startTime').value;
        const endTime = document.getElementById('endTime').value;

        const params = new URLSearchParams({
            page: currentPage,
            pageSize: pageSize
        });

        if (level) params.append('level', level);
        if (appId) params.append('appId', appId);
        if (keyword) params.append('keyword', keyword);
        if (startTime) params.append('startTime', startTime);
        if (endTime) params.append('endTime', endTime);

        const response = await fetch(`/admin/api/logs?${params}`);
        if (!response.ok) throw new Error('获取日志失败');

        const data = await response.json();
        logs = data.logs || [];
        totalLogs = data.total || 0;

        renderLogsTable(logs);
        renderPagination();
        updateLogStats();
    } catch (error) {
        console.error('加载日志失败:', error);
        loadMockLogs();
    }
}

function loadMockLogs() {
    const mockLogs = [
        {
            id: 1,
            level: 'info',
            app: 'app_001',
            module: 'UserService',
            message: '用户登录成功',
            time: '2024-01-15 14:30:25',
            details: {
                userId: 'user_123',
                ip: '192.168.1.100',
                userAgent: 'Mozilla/5.0...'
            }
        },
        {
            id: 2,
            level: 'warning',
            app: 'app_002',
            module: 'APIController',
            message: '请求频率过高',
            time: '2024-01-15 14:29:10',
            details: {
                ip: '10.0.0.50',
                count: 120,
                threshold: 100
            }
        },
        {
            id: 3,
            level: 'error',
            app: 'app_001',
            module: 'DatabaseService',
            message: '数据库连接超时',
            time: '2024-01-15 14:28:45',
            details: {
                error: 'Connection timeout after 30000ms',
                host: 'db.example.com',
                port: 5432
            }
        },
        {
            id: 4,
            level: 'info',
            app: 'app_003',
            module: 'AuthService',
            message: 'Token验证成功',
            time: '2024-01-15 14:28:20',
            details: {
                tokenId: 'tok_abc123',
                expiresIn: 3600
            }
        },
        {
            id: 5,
            level: 'debug',
            app: 'app_002',
            module: 'CacheService',
            message: '缓存命中',
            time: '2024-01-15 14:27:55',
            details: {
                key: 'user_profile_123',
                ttl: 300
            }
        }
    ];

    logs = mockLogs;
    totalLogs = mockLogs.length;

    renderLogsTable(logs);
    renderPagination();
    updateLogStats();
}

function renderLogsTable(logItems) {
    const tbody = document.getElementById('logsTableBody');
    if (!tbody) return;

    tbody.innerHTML = '';

    if (logItems.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="6" class="text-center text-muted py-5">
                    <i class="fas fa-clipboard-list fa-3x mb-3"></i>
                    <p class="mb-0">暂无日志数据</p>
                </td>
            </tr>
        `;
        return;
    }

    logItems.forEach(log => {
        const tr = document.createElement('tr');
        tr.setAttribute('data-id', log.id);

        const levelClass = getLevelClass(log.level);
        const levelName = getLevelName(log.level);

        tr.innerHTML = `
            <td><small class="text-muted">${log.time}</small></td>
            <td><span class="log-level ${levelClass}">${levelName}</span></td>
            <td><small>${log.app}</small></td>
            <td><small>${log.module}</small></td>
            <td><small>${log.message}</small></td>
            <td>
                <button class="btn btn-outline-primary btn-sm" onclick="viewLogDetail(${log.id})">
                    <i class="fas fa-eye"></i>
                </button>
            </td>
        `;

        tbody.appendChild(tr);
    });

    if (autoScroll) {
        scrollToTop();
    }
}

function getLevelClass(level) {
    const classes = {
        info: 'log-info',
        warning: 'log-warning',
        error: 'log-error',
        debug: 'log-debug'
    };
    return classes[level] || 'log-info';
}

function getLevelName(level) {
    const names = {
        info: 'INFO',
        warning: 'WARN',
        error: 'ERROR',
        debug: 'DEBUG'
    };
    return names[level] || level.toUpperCase();
}

function renderPagination() {
    const pagination = document.getElementById('pagination');
    if (!pagination) return;

    pagination.innerHTML = '';

    const totalPages = Math.ceil(totalLogs / pageSize);
    if (totalPages <= 1) return;

    const prevLi = document.createElement('li');
    prevLi.className = `page-item ${currentPage === 1 ? 'disabled' : ''}`;
    prevLi.innerHTML = `<a class="page-link" href="#">&laquo;</a>`;
    if (currentPage > 1) {
        prevLi.addEventListener('click', function(e) {
            e.preventDefault();
            currentPage--;
            loadLogs();
        });
    }
    pagination.appendChild(prevLi);

    for (let i = 1; i <= totalPages && i <= 5; i++) {
        const li = document.createElement('li');
        li.className = `page-item ${i === currentPage ? 'active' : ''}`;
        li.innerHTML = `<a class="page-link" href="#">${i}</a>`;
        li.addEventListener('click', function(e) {
            e.preventDefault();
            currentPage = i;
            loadLogs();
        });
        pagination.appendChild(li);
    }

    const nextLi = document.createElement('li');
    nextLi.className = `page-item ${currentPage === totalPages ? 'disabled' : ''}`;
    nextLi.innerHTML = `<a class="page-link" href="#">&raquo;</a>`;
    if (currentPage < totalPages) {
        nextLi.addEventListener('click', function(e) {
            e.preventDefault();
            currentPage++;
            loadLogs();
        });
    }
    pagination.appendChild(nextLi);
}

function updateLogStats() {
    const totalEl = document.getElementById('totalLogs');
    const startEl = document.getElementById('logStart');
    const endEl = document.getElementById('logEnd');

    if (totalEl) totalEl.textContent = totalLogs;

    const start = (currentPage - 1) * pageSize + 1;
    const end = Math.min(currentPage * pageSize, totalLogs);

    if (startEl) startEl.textContent = totalLogs > 0 ? start : 0;
    if (endEl) endEl.textContent = end;
}

function viewLogDetail(logId) {
    const log = logs.find(l => l.id === logId);
    if (!log) return;

    selectedLog = log;
    showLogDetailModal(log);
}

function showLogDetailModal(log) {
    const modal = new bootstrap.Modal(document.getElementById('logDetailModal'));

    const levelClass = getLevelClass(log.level);
    const levelName = getLevelName(log.level);

    document.getElementById('detailLevel').className = `log-level ${levelClass}`;
    document.getElementById('detailLevel').textContent = levelName;
    document.getElementById('detailTime').textContent = log.time;
    document.getElementById('detailApp').textContent = log.app;
    document.getElementById('detailModule').textContent = log.module;

    const contentEl = document.getElementById('logDetailContent');
    const pre = contentEl.querySelector('pre');

    if (typeof log.details === 'object') {
        pre.textContent = JSON.stringify(log.details, null, 2);
    } else {
        pre.textContent = log.details || log.message;
    }

    modal.show();
}

function copyLogContent() {
    if (!selectedLog) return;

    const content = typeof selectedLog.details === 'object'
        ? JSON.stringify(selectedLog.details, null, 2)
        : selectedLog.details || selectedLog.message;

    navigator.clipboard.writeText(content).then(() => {
        Auth.showToast('日志内容已复制', 'success');
    }).catch(err => {
        console.error('复制失败:', err);
        Auth.showToast('复制失败', 'error');
    });
}

function resetFilters() {
    document.getElementById('levelFilter').value = '';
    document.getElementById('appFilter').value = '';
    document.getElementById('searchKeyword').value = '';
    initDefaultTimeRange();
    currentPage = 1;
    loadLogs();
}

async function exportLogs() {
    try {
        const level = document.getElementById('levelFilter').value;
        const appId = document.getElementById('appFilter').value;
        const keyword = document.getElementById('searchKeyword').value;
        const startTime = document.getElementById('startTime').value;
        const endTime = document.getElementById('endTime').value;

        const params = new URLSearchParams();
        if (level) params.append('level', level);
        if (appId) params.append('appId', appId);
        if (keyword) params.append('keyword', keyword);
        if (startTime) params.append('startTime', startTime);
        if (endTime) params.append('endTime', endTime);
        params.append('export', 'true');

        const response = await fetch(`/admin/api/logs?${params}`);
        if (!response.ok) throw new Error('导出失败');

        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `logs_${new Date().toISOString().slice(0, 10)}.json`;
        document.body.appendChild(a);
        a.click();
        window.URL.revokeObjectURL(url);
        document.body.removeChild(a);

        Auth.showToast('日志导出成功', 'success');
    } catch (error) {
        console.error('导出日志失败:', error);
        Auth.showToast('导出失败，请重试', 'error');
    }
}

function scrollToTop() {
    const container = document.querySelector('.table-responsive');
    if (container) {
        container.scrollTop = 0;
    }
}

window.Logs = {
    loadLogs: loadLogs,
    viewLogDetail: viewLogDetail,
    exportLogs: exportLogs,
    resetFilters: resetFilters
};
