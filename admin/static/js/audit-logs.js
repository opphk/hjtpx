let auditCurrentPage = 1;
let auditPageSize = 50;
let auditLogs = [];
let currentAuditView = 'table';
let selectedAuditLogs = new Set();
let operationTrendChart = null;
let operationTypeChart = null;

document.addEventListener('DOMContentLoaded', () => {
    initAuditLogs();
    setupAuditEventListeners();
    loadAuditSummary();
    loadAuditLogs();
});

function initAuditLogs() {
    initCharts();
    updateAuditPageDisplay();
}

function setupAuditEventListeners() {
    const searchBtn = document.getElementById('searchAuditBtn');
    if (searchBtn) {
        searchBtn.addEventListener('click', () => {
            auditCurrentPage = 1;
            loadAuditLogs();
        });
    }

    const exportBtn = document.getElementById('exportAuditBtn');
    if (exportBtn) {
        exportBtn.addEventListener('click', () => showExportAuditModal());
    }

    const filterBtn = document.getElementById('filterAuditBtn');
    if (filterBtn) {
        filterBtn.addEventListener('click', () => showAuditFilterModal());
    }

    const applyFilterBtn = document.getElementById('applyFilterBtn');
    if (applyFilterBtn) {
        applyFilterBtn.addEventListener('click', () => applyAuditFilters());
    }

    const resetFilterBtn = document.getElementById('resetFilterBtn');
    if (resetFilterBtn) {
        resetFilterBtn.addEventListener('click', () => resetAuditFilters());
    }

    const confirmExportBtn = document.getElementById('confirmExportAuditBtn');
    if (confirmExportBtn) {
        confirmExportBtn.addEventListener('click', () => exportAuditLogs());
    }

    const timeRangeSelect = document.getElementById('auditTimeRange');
    if (timeRangeSelect) {
        timeRangeSelect.addEventListener('change', (e) => {
            const customRange = document.getElementById('customTimeRange');
            if (e.target.value === 'custom') {
                customRange.classList.remove('d-none');
            } else {
                customRange.classList.add('d-none');
            }
        });
    }

    const viewButtons = document.querySelectorAll('[data-view]');
    viewButtons.forEach(btn => {
        btn.addEventListener('click', (e) => {
            viewButtons.forEach(b => b.classList.remove('active'));
            e.target.classList.add('active');
            switchAuditView(e.target.dataset.view);
        });
    });

    const prevBtn = document.getElementById('prevAuditPageBtn');
    if (prevBtn) {
        prevBtn.addEventListener('click', () => changeAuditPage(auditCurrentPage - 1));
    }

    const nextBtn = document.getElementById('nextAuditPageBtn');
    if (nextBtn) {
        nextBtn.addEventListener('click', () => changeAuditPage(auditCurrentPage + 1));
    }

    const pageInput = document.getElementById('auditPageInput');
    if (pageInput) {
        pageInput.addEventListener('change', (e) => {
            const page = parseInt(e.target.value);
            if (page >= 1 && page <= parseInt(document.getElementById('auditTotalPages')?.textContent || '1')) {
                changeAuditPage(page);
            }
        });
    }

    const pageSizeSelect = document.getElementById('auditPageSize');
    if (pageSizeSelect) {
        pageSizeSelect.addEventListener('change', (e) => {
            auditPageSize = parseInt(e.target.value);
            auditCurrentPage = 1;
            loadAuditLogs();
        });
    }

    document.querySelectorAll('[data-filter]').forEach(btn => {
        btn.addEventListener('click', (e) => {
            document.querySelectorAll('[data-filter]').forEach(b => b.classList.remove('active'));
            e.target.classList.add('active');
            applyQuickFilter(e.target.dataset.filter);
        });
    });
}

function initCharts() {
    initOperationTrendChart();
    initOperationTypeChart();
}

function initOperationTrendChart() {
    const ctx = document.getElementById('operationTrendChart');
    if (!ctx) return;

    operationTrendChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: [],
            datasets: [{
                label: '操作次数',
                data: [],
                borderColor: '#3b82f6',
                backgroundColor: 'rgba(59, 130, 246, 0.1)',
                fill: true,
                tension: 0.4
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: { display: false }
            },
            scales: {
                y: { beginAtZero: true }
            }
        }
    });
}

function initOperationTypeChart() {
    const ctx = document.getElementById('operationTypeChart');
    if (!ctx) return;

    operationTypeChart = new Chart(ctx, {
        type: 'doughnut',
        data: {
            labels: [],
            datasets: [{
                data: [],
                backgroundColor: [
                    '#3b82f6',
                    '#10b981',
                    '#f59e0b',
                    '#ef4444',
                    '#8b5cf6',
                    '#ec4899',
                    '#14b8a6',
                    '#f97316'
                ],
                borderWidth: 2,
                borderColor: '#fff'
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    position: 'bottom',
                    labels: {
                        padding: 10,
                        usePointStyle: true
                    }
                }
            }
        }
    });
}

async function loadAuditSummary() {
    const mockSummary = getMockAuditSummary();
    
    try {
        const result = await auth.request('/admin/api/audit/summary');
        if (result.code === 0) {
            updateAuditSummary(result.data);
        } else {
            updateAuditSummary(mockSummary);
        }
    } catch (error) {
        updateAuditSummary(mockSummary);
    }
}

function getMockAuditSummary() {
    return {
        totalOperations: 12567,
        dangerousOperations: 23,
        activeUsers: 45,
        avgResponseTime: '125ms'
    };
}

function updateAuditSummary(summary) {
    const totalEl = document.getElementById('totalOperations');
    const dangerousEl = document.getElementById('dangerousOperations');
    const usersEl = document.getElementById('activeUsers');
    const responseEl = document.getElementById('avgResponseTime');

    if (totalEl) totalEl.textContent = formatAuditNumber(summary.totalOperations);
    if (dangerousEl) dangerousEl.textContent = formatAuditNumber(summary.dangerousOperations);
    if (usersEl) usersEl.textContent = formatAuditNumber(summary.activeUsers);
    if (responseEl) responseEl.textContent = summary.avgResponseTime;
}

async function loadAuditLogs() {
    const filters = getAuditFilters();
    const mockLogs = getMockAuditLogs();

    try {
        const params = new URLSearchParams({
            page: auditCurrentPage,
            size: auditPageSize,
            ...filters
        });

        const result = await auth.request(`/admin/api/audit/logs?${params.toString()}`);
        if (result.code === 0) {
            auditLogs = result.data.logs || [];
            renderAuditPagination(result.data.total || auditLogs.length);
            updateAuditLogsCount(result.data.total || auditLogs.length);
        } else {
            auditLogs = filterAuditLogs(mockLogs, filters);
            renderAuditPagination(auditLogs.length);
            updateAuditLogsCount(auditLogs.length);
        }
    } catch (error) {
        auditLogs = filterAuditLogs(mockLogs, filters);
        renderAuditPagination(auditLogs.length);
        updateAuditLogsCount(auditLogs.length);
    }

    renderAuditLogs();
    updateCharts();
}

function getAuditFilters() {
    return {
        operationType: document.getElementById('operationType')?.value || '',
        operatorName: document.getElementById('operatorName')?.value || '',
        resourceType: document.getElementById('resourceType')?.value || '',
        timeRange: document.getElementById('auditTimeRange')?.value || '24h',
        keyword: document.getElementById('auditKeyword')?.value || ''
    };
}

function getMockAuditLogs() {
    const operations = ['create', 'update', 'delete', 'enable', 'disable', 'login', 'logout', 'export', 'config'];
    const resources = ['app', 'rule', 'user', 'blacklist', 'config', 'system'];
    const results = ['success', 'failed', 'partial'];
    
    return Array.from({ length: 100 }, (_, i) => {
        const op = operations[Math.floor(Math.random() * operations.length)];
        const res = resources[Math.floor(Math.random() * resources.length)];
        const result = results[Math.floor(Math.random() * results.length)];
        const isDangerous = ['delete', 'disable', 'config'].includes(op);
        
        return {
            id: i + 1,
            timestamp: new Date(Date.now() - Math.random() * 7 * 24 * 60 * 60 * 1000).toISOString(),
            operator: `admin_${Math.floor(Math.random() * 10) + 1}`,
            operationType: op,
            resourceType: res,
            resourceId: `res_${Math.floor(Math.random() * 1000)}`,
            ip: `192.168.${Math.floor(Math.random() * 255)}.${Math.floor(Math.random() * 255)}`,
            result: result,
            riskLevel: isDangerous && result === 'failed' ? 'high' : 'low',
            details: `操作详情: ${op} ${res}`
        };
    });
}

function filterAuditLogs(logs, filters) {
    return logs.filter(log => {
        if (filters.operationType && log.operationType !== filters.operationType) return false;
        if (filters.operatorName && !log.operator.toLowerCase().includes(filters.operatorName.toLowerCase())) return false;
        if (filters.resourceType && log.resourceType !== filters.resourceType) return false;
        if (filters.keyword && !log.details.toLowerCase().includes(filters.keyword.toLowerCase())) return false;
        return true;
    });
}

function renderAuditLogs() {
    if (currentAuditView === 'table') {
        renderAuditTable();
    } else if (currentAuditView === 'timeline') {
        renderAuditTimeline();
    } else {
        renderAuditCards();
    }
}

function renderAuditTable() {
    const tbody = document.getElementById('auditTableBody');
    if (!tbody) return;

    document.getElementById('auditTableView')?.classList.remove('d-none');
    document.getElementById('auditTimelineView')?.classList.add('d-none');
    document.getElementById('auditCardView')?.classList.add('d-none');

    tbody.innerHTML = auditLogs.map(log => `
        <tr class="${selectedAuditLogs.has(log.id) ? 'table-primary' : ''} ${log.riskLevel === 'high' ? 'table-danger' : ''}">
            <td><small class="text-muted">${formatAuditDate(log.timestamp)}</small></td>
            <td><strong>${escapeHtml(log.operator)}</strong></td>
            <td><span class="badge ${getAuditOperationBadge(log.operationType)}">${getAuditOperationText(log.operationType)}</span></td>
            <td><span class="badge bg-secondary">${getResourceTypeText(log.resourceType)}</span></td>
            <td><code class="small">${log.resourceId}</code></td>
            <td><small>${log.ip}</small></td>
            <td><span class="badge ${getAuditResultBadge(log.result)}">${getAuditResultText(log.result)}</span></td>
            <td>
                <div class="btn-group btn-group-sm">
                    <button class="btn btn-outline-secondary" onclick="showAuditDetail(${log.id})" title="详情"><i class="fas fa-eye"></i></button>
                    <button class="btn btn-outline-info" onclick="copyAuditLog(${log.id})" title="复制"><i class="fas fa-copy"></i></button>
                </div>
            </td>
        </tr>
    `).join('');
}

function renderAuditTimeline() {
    const container = document.getElementById('auditTimelineView');
    if (!container) return;

    container.classList.remove('d-none');
    document.getElementById('auditTableView')?.classList.add('d-none');
    document.getElementById('auditCardView')?.classList.add('d-none');

    const groupedLogs = groupLogsByDate(auditLogs);
    
    container.innerHTML = Object.entries(groupedLogs).map(([date, logs]) => `
        <div class="timeline-item">
            <div class="timeline-date">${date}</div>
            <div class="timeline-events">
                ${logs.map(log => `
                    <div class="timeline-event ${log.riskLevel === 'high' ? 'border-danger' : ''}">
                        <div class="timeline-time">${formatAuditTime(log.timestamp)}</div>
                        <div class="timeline-content">
                            <strong>${escapeHtml(log.operator)}</strong>
                            ${getAuditOperationText(log.operationType)}
                            <span class="text-muted">${getResourceTypeText(log.resourceType)}</span>
                            <code>${log.resourceId}</code>
                        </div>
                    </div>
                `).join('')}
            </div>
        </div>
    `).join('');
}

function renderAuditCards() {
    const container = document.getElementById('auditCardView');
    if (!container) return;

    container.classList.remove('d-none');
    document.getElementById('auditTableView')?.classList.add('d-none');
    document.getElementById('auditTimelineView')?.classList.add('d-none');

    container.innerHTML = `<div class="row g-3 p-3">${auditLogs.map(log => `
        <div class="col-md-6 col-lg-4">
            <div class="card border ${log.riskLevel === 'high' ? 'border-danger' : ''}">
                <div class="card-header bg-transparent py-2 d-flex justify-content-between align-items-center">
                    <strong>${escapeHtml(log.operator)}</strong>
                    <span class="badge ${getAuditResultBadge(log.result)}">${getAuditResultText(log.result)}</span>
                </div>
                <div class="card-body py-2">
                    <p class="card-text small mb-1">
                        <span class="badge ${getAuditOperationBadge(log.operationType)}">${getAuditOperationText(log.operationType)}</span>
                        <span class="badge bg-secondary">${getResourceTypeText(log.resourceType)}</span>
                    </p>
                    <p class="card-text small text-muted mb-1">
                        <i class="fas fa-hashtag me-1"></i>${log.resourceId}
                    </p>
                    <p class="card-text small text-muted mb-0">
                        <i class="fas fa-clock me-1"></i>${formatAuditDate(log.timestamp)}
                    </p>
                </div>
                <div class="card-footer bg-transparent py-2">
                    <button class="btn btn-sm btn-outline-primary w-100" onclick="showAuditDetail(${log.id})">
                        <i class="fas fa-eye me-1"></i>查看详情
                    </button>
                </div>
            </div>
        </div>
    `).join('')}</div>`;
}

function groupLogsByDate(logs) {
    return logs.reduce((groups, log) => {
        const date = formatAuditDate(log.timestamp).split(' ')[0];
        if (!groups[date]) groups[date] = [];
        groups[date].push(log);
        return groups;
    }, {});
}

function switchAuditView(view) {
    currentAuditView = view;
    renderAuditLogs();
}

function showAuditDetail(logId) {
    const log = auditLogs.find(l => l.id === logId);
    if (!log) return;

    const content = document.getElementById('auditDetailContent');
    if (content) {
        content.innerHTML = `
            <div class="row">
                <div class="col-md-6">
                    <table class="table table-borderless">
                        <tr><td class="text-muted">日志ID</td><td>${log.id}</td></tr>
                        <tr><td class="text-muted">时间</td><td>${formatAuditDate(log.timestamp)}</td></tr>
                        <tr><td class="text-muted">操作者</td><td><strong>${escapeHtml(log.operator)}</strong></td></tr>
                        <tr><td class="text-muted">IP地址</td><td>${log.ip}</td></tr>
                    </table>
                </div>
                <div class="col-md-6">
                    <table class="table table-borderless">
                        <tr><td class="text-muted">操作类型</td><td><span class="badge ${getAuditOperationBadge(log.operationType)}">${getAuditOperationText(log.operationType)}</span></td></tr>
                        <tr><td class="text-muted">资源类型</td><td><span class="badge bg-secondary">${getResourceTypeText(log.resourceType)}</span></td></tr>
                        <tr><td class="text-muted">资源ID</td><td><code>${log.resourceId}</code></td></tr>
                        <tr><td class="text-muted">结果</td><td><span class="badge ${getAuditResultBadge(log.result)}">${getAuditResultText(log.result)}</span></td></tr>
                    </table>
                </div>
            </div>
            <div class="alert ${log.riskLevel === 'high' ? 'alert-danger' : 'alert-info'} mt-3">
                <i class="fas fa-info-circle me-1"></i>
                <strong>风险等级:</strong> ${log.riskLevel === 'high' ? '高风险' : '低风险'}
            </div>
            <div class="mt-3">
                <h6>详细信息</h6>
                <pre class="bg-light p-3 rounded" style="max-height: 200px; overflow: auto;">${escapeHtml(log.details || '无详细信息')}</pre>
            </div>
        `;
    }

    currentLogForCopy = log;
    const modal = new bootstrap.Modal(document.getElementById('auditDetailModal'));
    modal.show();
}

function copyAuditLog(logId) {
    const log = auditLogs.find(l => l.id === logId);
    if (!log) return;

    const text = JSON.stringify(log, null, 2);
    navigator.clipboard.writeText(text).then(() => {
        showAuditToast('日志已复制到剪贴板', 'success');
    }).catch(() => {
        showAuditToast('复制失败', 'danger');
    });
}

document.getElementById('copyAuditBtn')?.addEventListener('click', () => {
    if (currentLogForCopy) {
        copyAuditLog(currentLogForCopy.id);
    }
});

function updateCharts() {
    updateOperationTrendChart();
    updateOperationTypeChart();
}

function updateOperationTrendChart() {
    if (!operationTrendChart || auditLogs.length === 0) return;

    const groupedByHour = groupLogsByHour(auditLogs);
    operationTrendChart.data.labels = Object.keys(groupedByHour);
    operationTrendChart.data.datasets[0].data = Object.values(groupedByHour);
    operationTrendChart.update();
}

function updateOperationTypeChart() {
    if (!operationTypeChart || auditLogs.length === 0) return;

    const groupedByType = groupLogsByOperationType(auditLogs);
    operationTypeChart.data.labels = Object.keys(groupedByType).map(getAuditOperationText);
    operationTypeChart.data.datasets[0].data = Object.values(groupedByType);
    operationTypeChart.update();
}

function groupLogsByHour(logs) {
    return logs.reduce((groups, log) => {
        const hour = new Date(log.timestamp).toLocaleTimeString('zh-CN', { hour: '2-digit' });
        groups[hour] = (groups[hour] || 0) + 1;
        return groups;
    }, {});
}

function groupLogsByOperationType(logs) {
    return logs.reduce((groups, log) => {
        groups[log.operationType] = (groups[log.operationType] || 0) + 1;
        return groups;
    }, {});
}

function renderAuditPagination(total) {
    const totalPages = Math.ceil(total / auditPageSize);
    const pagination = document.getElementById('auditPagination');
    const totalPagesEl = document.getElementById('auditTotalPages');
    const currentPageEl = document.getElementById('currentPage');

    if (totalPagesEl) totalPagesEl.textContent = totalPages;
    if (currentPageEl) currentPageEl.textContent = auditCurrentPage;

    if (!pagination || totalPages <= 1) {
        if (pagination) pagination.innerHTML = '';
        return;
    }

    let html = '<div class="btn-group btn-group-sm">';
    html += `<button class="btn btn-outline-secondary" onclick="changeAuditPage(${auditCurrentPage - 1})" ${auditCurrentPage === 1 ? 'disabled' : ''}>上一页</button>`;

    const startPage = Math.max(1, auditCurrentPage - 2);
    const endPage = Math.min(totalPages, auditCurrentPage + 2);

    for (let i = startPage; i <= endPage; i++) {
        html += `<button class="btn ${i === auditCurrentPage ? 'btn-primary' : 'btn-outline-secondary'}" onclick="changeAuditPage(${i})">${i}</button>`;
    }

    html += `<button class="btn btn-outline-secondary" onclick="changeAuditPage(${auditCurrentPage + 1})" ${auditCurrentPage === totalPages ? 'disabled' : ''}>下一页</button>`;
    html += '</div>';

    pagination.innerHTML = html;
}

function changeAuditPage(page) {
    const totalPages = parseInt(document.getElementById('auditTotalPages')?.textContent || '1');
    if (page < 1 || page > totalPages) return;
    
    auditCurrentPage = page;
    updateAuditPageDisplay();
    loadAuditLogs();
}

function updateAuditPageDisplay() {
    const pageInput = document.getElementById('auditPageInput');
    if (pageInput) pageInput.value = auditCurrentPage;
}

function updateAuditLogsCount(total) {
    const countEl = document.getElementById('auditLogsCount');
    if (countEl) countEl.textContent = total;
}

function showAuditFilterModal() {
    const modal = new bootstrap.Modal(document.getElementById('auditFilterModal'));
    modal.show();
}

function applyAuditFilters() {
    const filters = {
        operator: document.getElementById('filterOperator')?.value || '',
        ip: document.getElementById('filterIP')?.value || '',
        resourceId: document.getElementById('filterResourceId')?.value || '',
        result: document.getElementById('filterResult')?.value || '',
        riskLevel: document.getElementById('filterRiskLevel')?.value || ''
    };

    if (filters.operator) document.getElementById('operatorName').value = filters.operator;
    if (filters.result) document.getElementById('operationType').value = filters.result;

    auditCurrentPage = 1;
    loadAuditLogs();
    
    bootstrap.Modal.getInstance(document.getElementById('auditFilterModal'))?.hide();
    showAuditToast('筛选条件已应用', 'success');
}

function resetAuditFilters() {
    document.getElementById('filterOperator').value = '';
    document.getElementById('filterIP').value = '';
    document.getElementById('filterResourceId').value = '';
    document.getElementById('filterResult').value = '';
    document.getElementById('filterRiskLevel').value = '';
    
    document.getElementById('operatorName').value = '';
    document.getElementById('operationType').value = '';
    document.getElementById('resourceType').value = '';
    document.getElementById('auditKeyword').value = '';
    document.getElementById('auditTimeRange').value = '24h';
    
    auditCurrentPage = 1;
    loadAuditLogs();
    
    showAuditToast('筛选条件已重置', 'info');
}

function applyQuickFilter(filter) {
    const typeMap = {
        'all': '',
        'dangerous': 'delete',
        'failed': 'failed',
        'login': 'login',
        'config': 'config'
    };

    const resultMap = {
        'all': '',
        'dangerous': '',
        'failed': 'failed',
        'login': '',
        'config': ''
    };

    document.getElementById('operationType').value = typeMap[filter] || '';
    auditCurrentPage = 1;
    loadAuditLogs();
}

function showExportAuditModal() {
    const modal = new bootstrap.Modal(document.getElementById('exportAuditModal'));
    modal.show();
}

function exportAuditLogs() {
    const format = document.querySelector('input[name="auditExportFormat"]:checked')?.value || 'csv';
    const scope = document.querySelector('input[name="auditExportScope"]:checked')?.value || 'current';
    
    const exportLogs = scope === 'all' ? auditLogs : auditLogs;
    
    if (exportLogs.length === 0) {
        showAuditToast('没有可导出的日志', 'warning');
        return;
    }

    const timestamp = new Date().toISOString().slice(0, 10);
    
    if (format === 'json') {
        exportAuditAsJSON(exportLogs, `audit_logs_${timestamp}`);
    } else {
        exportAuditAsCSV(exportLogs, `audit_logs_${timestamp}.csv`);
    }

    bootstrap.Modal.getInstance(document.getElementById('exportAuditModal'))?.hide();
    showAuditToast(`已导出 ${exportLogs.length} 条审计日志`, 'success');
}

function exportAuditAsCSV(logs, filename) {
    const headers = ['ID', '时间', '操作者', '操作类型', '资源类型', '资源ID', 'IP地址', '结果', '风险等级', '详情'];
    const rows = logs.map(log => [
        log.id,
        formatAuditDate(log.timestamp),
        log.operator,
        log.operationType,
        log.resourceType,
        log.resourceId,
        log.ip,
        log.result,
        log.riskLevel,
        `"${log.details || ''}"`
    ]);

    const csvContent = [headers.join(','), ...rows.map(r => r.join(','))].join('\n');
    downloadFile(csvContent, filename, 'text/csv;charset=utf-8');
}

function exportAuditAsJSON(logs, filename) {
    const jsonData = {
        exportTime: new Date().toISOString(),
        totalCount: logs.length,
        logs: logs
    };
    const jsonString = JSON.stringify(jsonData, null, 2);
    downloadFile(jsonString, `${filename}.json`, 'application/json;charset=utf-8');
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

function getAuditOperationBadge(type) {
    const map = {
        'create': 'bg-success',
        'update': 'bg-primary',
        'delete': 'bg-danger',
        'enable': 'bg-success',
        'disable': 'bg-warning',
        'login': 'bg-info',
        'logout': 'bg-secondary',
        'export': 'bg-purple',
        'import': 'bg-cyan',
        'config': 'bg-orange'
    };
    return map[type] || 'bg-secondary';
}

function getAuditOperationText(type) {
    const map = {
        'create': '创建',
        'update': '更新',
        'delete': '删除',
        'enable': '启用',
        'disable': '禁用',
        'login': '登录',
        'logout': '登出',
        'export': '导出',
        'import': '导入',
        'config': '配置'
    };
    return map[type] || type;
}

function getAuditResultBadge(result) {
    const map = {
        'success': 'bg-success',
        'failed': 'bg-danger',
        'partial': 'bg-warning'
    };
    return map[result] || 'bg-secondary';
}

function getAuditResultText(result) {
    const map = {
        'success': '成功',
        'failed': '失败',
        'partial': '部分成功'
    };
    return map[result] || result;
}

function getResourceTypeText(type) {
    const map = {
        'app': '应用',
        'rule': '规则',
        'user': '用户',
        'blacklist': '黑名单',
        'config': '配置',
        'system': '系统'
    };
    return map[type] || type;
}

function formatAuditDate(dateStr) {
    if (!dateStr) return '-';
    const date = new Date(dateStr);
    if (isNaN(date.getTime())) return dateStr;

    return date.toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
    });
}

function formatAuditTime(dateStr) {
    if (!dateStr) return '-';
    const date = new Date(dateStr);
    if (isNaN(date.getTime())) return dateStr;

    return date.toLocaleTimeString('zh-CN', {
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
    });
}

function formatAuditNumber(num) {
    if (num >= 1000000) {
        return (num / 1000000).toFixed(1) + 'M';
    } else if (num >= 1000) {
        return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
}

function showAuditToast(message, type = 'info') {
    const container = document.getElementById('auditToastContainer') || createAuditToastContainer();
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

function createAuditToastContainer() {
    const container = document.createElement('div');
    container.id = 'auditToastContainer';
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
