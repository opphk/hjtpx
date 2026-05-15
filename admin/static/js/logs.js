let logCurrentPage = 1;
let logPageSize = 20;

document.addEventListener('DOMContentLoaded', () => {
    loadLogs();
    setupLogEventListeners();
});

function setupLogEventListeners() {
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

    const closeLogModal = document.getElementById('closeLogModal');
    if (closeLogModal) {
        closeLogModal.addEventListener('click', closeLogDetailModal);
    }

    const closeLogDetailBtn = document.getElementById('closeLogDetailBtn');
    if (closeLogDetailBtn) {
        closeLogDetailBtn.addEventListener('click', closeLogDetailModal);
    }
}

async function loadLogs() {
    const mockLogs = getMockLogs();
    let logs = mockLogs;

    try {
        const level = document.getElementById('logLevel')?.value || '';
        const startDate = document.getElementById('startDate')?.value || '';
        const endDate = document.getElementById('endDate')?.value || '';
        const keyword = document.getElementById('keyword')?.value || '';
        
        const params = new URLSearchParams({
            page: logCurrentPage,
            size: logPageSize,
            level,
            startDate,
            endDate,
            keyword
        });

        const result = await auth.request(`/logs?${params.toString()}`);
        if (result.code === 0) {
            logs = result.data.list || [];
            renderLogPagination(result.data.total || logs.length);
        } else {
            renderLogPagination(logs.length);
        }
    } catch (error) {
        renderLogPagination(logs.length);
    }

    renderLogs(logs);
}

function getMockLogs() {
    return [
        { id: 'log_001', level: 'info', message: '用户登录成功', timestamp: '2024-01-15 14:32:18', source: 'auth_service', details: '{\"user_id\": 1, \"ip\": \"192.168.1.100\"}' },
        { id: 'log_002', level: 'error', message: '数据库连接失败', timestamp: '2024-01-15 14:30:05', source: 'db_service', details: '{\"error\": \"connection timeout\", \"host\": \"db.example.com\"}' },
        { id: 'log_003', level: 'warning', message: 'API请求频率过高', timestamp: '2024-01-15 14:28:45', source: 'api_gateway', details: '{\"app_id\": \"app_001\", \"rate\": \"1000/min\"}' },
        { id: 'log_004', level: 'info', message: '新用户注册', timestamp: '2024-01-15 14:25:12', source: 'user_service', details: '{\"user_id\": 12345, \"email\": \"user@example.com\"}' },
        { id: 'log_005', level: 'debug', message: '缓存命中', timestamp: '2024-01-15 14:20:33', source: 'cache_service', details: '{\"key\": \"user:12345\", \"ttl\": 3600}' },
        { id: 'log_006', level: 'error', message: '支付回调处理失败', timestamp: '2024-01-15 14:15:09', source: 'payment_service', details: '{\"order_id\": \"ORD_001\", \"error\": \"invalid signature\"}' }
    ];
}

function renderLogs(logs) {
    const container = document.getElementById('logsList');
    const countEl = document.getElementById('logsCount');

    if (countEl) {
        countEl.textContent = `共 ${logs.length} 条日志`;
    }

    if (!container) return;

    container.innerHTML = logs.map(log => `
        <div class="log-item log-${log.level}" onclick="showLogDetail('${log.id}')">
            <div class="log-header">
                <span class="log-level ${log.level}">${getLevelText(log.level)}</span>
                <span class="log-time">${log.timestamp}</span>
                <span class="log-source">${log.source}</span>
            </div>
            <div class="log-message">${escapeHtml(log.message)}</div>
        </div>
    `).join('');
}

function getLevelText(level) {
    const map = {
        debug: 'DEBUG',
        info: 'INFO',
        warning: 'WARNING',
        error: 'ERROR'
    };
    return map[level] || level;
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function renderLogPagination(total) {
    const pagination = document.getElementById('pagination');
    if (!pagination) return;

    const totalPages = Math.ceil(total / logPageSize);
    let html = '';

    if (totalPages > 1) {
        html += `<button class="btn btn-sm" onclick="changeLogPage(${logCurrentPage - 1})" ${logCurrentPage === 1 ? 'disabled' : ''}>上一页</button>`;
        
        for (let i = 1; i <= totalPages; i++) {
            html += `<button class="btn btn-sm ${i === logCurrentPage ? 'btn-primary' : ''}" onclick="changeLogPage(${i})">${i}</button>`;
        }
        
        html += `<button class="btn btn-sm" onclick="changeLogPage(${logCurrentPage + 1})" ${logCurrentPage === totalPages ? 'disabled' : ''}>下一页</button>`;
    }

    pagination.innerHTML = html;
}

function changeLogPage(page) {
    logCurrentPage = page;
    loadLogs();
}

function showLogDetail(logId) {
    const mockLogs = getMockLogs();
    const log = mockLogs.find(l => l.id === logId);
    if (!log) return;

    const modal = document.getElementById('logDetailModal');
    const content = document.getElementById('logDetailContent');

    content.innerHTML = `
        <div class="log-detail-field">
            <label>日志ID:</label>
            <span>${log.id}</span>
        </div>
        <div class="log-detail-field">
            <label>级别:</label>
            <span class="log-level ${log.level}">${getLevelText(log.level)}</span>
        </div>
        <div class="log-detail-field">
            <label>时间:</label>
            <span>${log.timestamp}</span>
        </div>
        <div class="log-detail-field">
            <label>来源:</label>
            <span>${log.source}</span>
        </div>
        <div class="log-detail-field">
            <label>消息:</label>
            <span>${escapeHtml(log.message)}</span>
        </div>
        <div class="log-detail-field">
            <label>详情:</label>
            <pre><code>${formatJson(log.details)}</code></pre>
        </div>
    `;

    modal.classList.remove('hidden');
}

function formatJson(jsonStr) {
    try {
        return JSON.stringify(JSON.parse(jsonStr), null, 2);
    } catch (e) {
        return jsonStr;
    }
}

function closeLogDetailModal() {
    const modal = document.getElementById('logDetailModal');
    modal.classList.add('hidden');
}

function exportLogs() {
    const mockLogs = getMockLogs();
    const csvContent = [
        ['ID', '级别', '消息', '时间', '来源'].join(','),
        ...mockLogs.map(log => [
            log.id,
            log.level,
            `"${log.message.replace(/"/g, '""')}"`,
            log.timestamp,
            log.source
        ].join(','))
    ].join('\n');

    const blob = new Blob(['\ufeff' + csvContent], { type: 'text/csv;charset=utf-8;' });
    const link = document.createElement('a');
    const url = URL.createObjectURL(blob);
    link.setAttribute('href', url);
    link.setAttribute('download', `logs_${new Date().toISOString().slice(0, 10)}.csv`);
    link.style.visibility = 'hidden';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
}
