let logCurrentPage = 1;
let logPageSize = 20;
let currentLogs = [];
let currentDisplay = 'list';
let autoRefreshInterval = null;
let currentLogForCopy = null;
let selectedLogs = new Set();
let savedSearches = [];
let currentAnalytics = null;
const LOG_AUTO_REFRESH_INTERVAL = 30000;

document.addEventListener('DOMContentLoaded', () => {
    initDefaultDates();
    setupEventListeners();
    loadLogs();
    loadLogStatistics();
    loadSavedSearches();
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

    const analyticsBtn = document.getElementById('viewAnalyticsBtn');
    if (analyticsBtn) {
        analyticsBtn.addEventListener('click', showAnalytics);
    }

    const exportCSVBtn = document.getElementById('exportCSVBtn');
    if (exportCSVBtn) {
        exportCSVBtn.addEventListener('click', () => exportLogs('csv'));
    }

    const exportJSONBtn = document.getElementById('exportJSONBtn');
    if (exportJSONBtn) {
        exportJSONBtn.addEventListener('click', () => exportLogs('json'));
    }

    const streamExportBtn = document.getElementById('streamExportBtn');
    if (streamExportBtn) {
        streamExportBtn.addEventListener('click', () => streamExportLogs('csv'));
    }

    const sortBySelect = document.getElementById('sortBy');
    if (sortBySelect) {
        sortBySelect.addEventListener('change', () => {
            logCurrentPage = 1;
            loadLogs();
        });
    }

    const sortOrderSelect = document.getElementById('sortOrder');
    if (sortOrderSelect) {
        sortOrderSelect.addEventListener('change', () => {
            logCurrentPage = 1;
            loadLogs();
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
    document.getElementById('filterIPAddress').value = '';
    document.getElementById('filterMinRiskScore').value = '';
    document.getElementById('filterMaxRiskScore').value = '';
    document.getElementById('sortBy').value = 'created_at';
    document.getElementById('sortOrder').value = 'desc';
    initDefaultDates();
    logCurrentPage = 1;
    loadLogs();
}

async function loadSavedSearches() {
    try {
        const response = await fetch('/admin/api/logs/saved-searches', {
            headers: {
                'Authorization': 'Bearer ' + localStorage.getItem('admin_token')
            }
        });
        if (!response.ok) throw new Error('Network error');
        
        const result = await response.json();
        if (result.code === 0) {
            savedSearches = result.data || [];
            renderSavedSearches();
        }
    } catch (error) {
        console.error('Load saved searches failed:', error);
    }
}

function renderSavedSearches() {
    const container = document.getElementById('savedSearchesContainer');
    if (!container) return;
    
    if (savedSearches.length === 0) {
        container.innerHTML = '<p class="text-muted">暂无保存的搜索</p>';
        return;
    }
    
    container.innerHTML = savedSearches.map(search => `
        <div class="saved-search-item d-flex align-items-center justify-content-between mb-2">
            <div>
                <strong>${escapeHtml(search.name)}</strong>
                <small class="text-muted d-block">${escapeHtml(search.description || '')}</small>
            </div>
            <div>
                <button class="btn btn-sm btn-primary me-1" onclick="applySavedSearch(${search.id})">
                    <i class="fas fa-search"></i> 应用
                </button>
                <button class="btn btn-sm btn-danger" onclick="deleteSavedSearch(${search.id})">
                    <i class="fas fa-trash"></i>
                </button>
            </div>
        </div>
    `).join('');
}

async function applySavedSearch(id) {
    const search = savedSearches.find(s => s.id === id);
    if (!search) return;
    
    const query = search.query || {};
    
    if (query.status) document.getElementById('filterStatus').value = query.status;
    if (query.captcha_type) document.getElementById('filterCaptchaType').value = query.captcha_type;
    if (query.risk_level) document.getElementById('filterRiskLevel').value = query.risk_level;
    if (query.start_date) document.getElementById('filterStartDate').value = query.start_date;
    if (query.end_date) document.getElementById('filterEndDate').value = query.end_date;
    if (query.min_risk_score) document.getElementById('filterMinRiskScore').value = query.min_risk_score;
    if (query.max_risk_score) document.getElementById('filterMaxRiskScore').value = query.max_risk_score;
    
    logCurrentPage = 1;
    loadLogs();
    showToast('已应用保存的搜索条件', 'success');
}

async function saveCurrentSearch() {
    const name = prompt('请输入保存的搜索名称：');
    if (!name) return;
    
    const description = prompt('请输入搜索描述（可选）：') || '';
    
    const filters = getCurrentFilters();
    
    try {
        const response = await fetch('/admin/api/logs/saved-searches', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': 'Bearer ' + localStorage.getItem('admin_token')
            },
            body: JSON.stringify({
                name: name,
                description: description,
                query: filters
            })
        });
        
        const result = await response.json();
        if (result.code === 0) {
            showToast('搜索条件已保存', 'success');
            loadSavedSearches();
        } else {
            showToast('保存失败', 'error');
        }
    } catch (error) {
        console.error('Save search failed:', error);
        showToast('保存失败', 'error');
    }
}

async function deleteSavedSearch(id) {
    if (!confirm('确定要删除此保存的搜索吗？')) return;
    
    try {
        const response = await fetch(`/admin/api/logs/saved-searches/${id}`, {
            method: 'DELETE',
            headers: {
                'Authorization': 'Bearer ' + localStorage.getItem('admin_token')
            }
        });
        
        const result = await response.json();
        if (result.code === 0) {
            showToast('已删除保存的搜索', 'success');
            loadSavedSearches();
        } else {
            showToast('删除失败', 'error');
        }
    } catch (error) {
        console.error('Delete saved search failed:', error);
        showToast('删除失败', 'error');
    }
}

function getCurrentFilters() {
    return {
        status: document.getElementById('filterStatus')?.value || '',
        captcha_type: document.getElementById('filterCaptchaType')?.value || '',
        risk_level: document.getElementById('filterRiskLevel')?.value || '',
        ip_address: document.getElementById('filterIPAddress')?.value || '',
        start_date: document.getElementById('filterStartDate')?.value || '',
        end_date: document.getElementById('filterEndDate')?.value || '',
        min_risk_score: document.getElementById('filterMinRiskScore')?.value || '',
        max_risk_score: document.getElementById('filterMaxRiskScore')?.value || '',
        sort_by: document.getElementById('sortBy')?.value || 'created_at',
        sort_order: document.getElementById('sortOrder')?.value || 'desc'
    };
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
        infoEl.innerHTML = `
            总记录: ${stats.total_count || 0} | 
            成功: ${stats.success_count || 0} | 
            失败: ${stats.failed_count || 0} | 
            平均风险评分: ${(stats.avg_risk_score || 0).toFixed(2)}
            ${stats.cache_hit ? '<span class="badge bg-success ms-2">缓存命中</span>' : ''}
        `;
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
    const filters = getCurrentFilters();
    filters.page = logCurrentPage;
    filters.page_size = logPageSize;

    const params = new URLSearchParams(filters);
    
    if (showLoading) {
        const tbody = document.getElementById('logsTableBody');
        if (tbody) {
            tbody.innerHTML = '<tr><td colspan="12" class="text-center"><i class="fas fa-spinner fa-spin"></i> 加载中...</td></tr>';
        }
    }
    
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
            updateQueryStats(result.data.stats);
        } else {
            renderTable([]);
            renderPagination(0);
            updateTotalLogs(0);
        }
    })
    .catch(error => {
        console.error('Load logs failed:', error);
        showToast('加载日志失败', 'error');
        renderTable([]);
        renderPagination(0);
        updateTotalLogs(0);
    });
}

function updateQueryStats(stats) {
    if (!stats) return;
    
    const queryTimeEl = document.getElementById('queryTime');
    if (queryTimeEl) {
        queryTimeEl.textContent = `查询时间: ${stats.query_time_ms?.toFixed(2) || 0}ms`;
    }
    
    const cacheStatusEl = document.getElementById('cacheStatus');
    if (cacheStatusEl) {
        cacheStatusEl.textContent = stats.cache_hit ? '缓存命中' : '数据库查询';
        cacheStatusEl.className = stats.cache_hit ? 'badge bg-success' : 'badge bg-info';
    }
}

function renderTable(logs) {
    const tbody = document.getElementById('logsTableBody');
    if (!tbody) return;
    tbody.innerHTML = '';

    if (logs.length === 0) {
        tbody.innerHTML = '<tr><td colspan="12" class="text-center">暂无数据</td></tr>';
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

function renderPagination(total) {
    const pagination = document.getElementById('logsPagination');
    if (!pagination) return;
    
    const totalPages = Math.ceil(total / logPageSize);
    let html = '';
    
    if (totalPages <= 1) {
        pagination.innerHTML = '';
        return;
    }
    
    html += `<li class="page-item ${logCurrentPage === 1 ? 'disabled' : ''}">
        <a class="page-link" href="#" onclick="goToPage(${logCurrentPage - 1}); return false;">上一页</a>
    </li>`;
    
    const maxPages = 5;
    let startPage = Math.max(1, logCurrentPage - Math.floor(maxPages / 2));
    let endPage = Math.min(totalPages, startPage + maxPages - 1);
    
    if (endPage - startPage < maxPages - 1) {
        startPage = Math.max(1, endPage - maxPages + 1);
    }
    
    if (startPage > 1) {
        html += `<li class="page-item"><a class="page-link" href="#" onclick="goToPage(1); return false;">1</a></li>`;
        if (startPage > 2) {
            html += `<li class="page-item disabled"><span class="page-link">...</span></li>`;
        }
    }
    
    for (let i = startPage; i <= endPage; i++) {
        html += `<li class="page-item ${i === logCurrentPage ? 'active' : ''}">
            <a class="page-link" href="#" onclick="goToPage(${i}); return false;">${i}</a>
        </li>`;
    }
    
    if (endPage < totalPages) {
        if (endPage < totalPages - 1) {
            html += `<li class="page-item disabled"><span class="page-link">...</span></li>`;
        }
        html += `<li class="page-item"><a class="page-link" href="#" onclick="goToPage(${totalPages}); return false;">${totalPages}</a></li>`;
    }
    
    html += `<li class="page-item ${logCurrentPage === totalPages ? 'disabled' : ''}">
        <a class="page-link" href="#" onclick="goToPage(${logCurrentPage + 1}); return false;">下一页</a>
    </li>`;
    
    pagination.innerHTML = html;
}

function goToPage(page) {
    logCurrentPage = page;
    loadLogs();
}

function getCaptchaTypeText(type) {
    const map = {
        'slider': '滑块',
        'click': '点选',
        'image': '图片',
        'voice': '语音',
        'gesture': '手势',
        '3d': '3D旋转',
        'lianliankan': '连连看'
    };
    return map[type] || type || '-';
}

function getStatusBadgeClass(status) {
    const map = {
        'success': 'success',
        'failed': 'danger',
        'pending': 'warning',
        'blocked': 'dark'
    };
    return map[status] || 'secondary';
}

function getStatusText(status) {
    const map = {
        'success': '成功',
        'failed': '失败',
        'pending': '待处理',
        'blocked': '阻止'
    };
    return map[status] || status || '-';
}

function getRiskBadgeClass(level) {
    const map = {
        'low': 'success',
        'medium': 'warning',
        'high': 'danger',
        'critical': 'dark'
    };
    return map[level] || 'secondary';
}

function getRiskLevelText(level) {
    const map = {
        'low': '低风险',
        'medium': '中风险',
        'high': '高风险',
        'critical': '极高风险'
    };
    return map[level] || level || '-';
}

function exportLogs(format) {
    const filters = getCurrentFilters();
    delete filters.page;
    delete filters.page_size;
    delete filters.sort_by;
    delete filters.sort_order;
    
    const params = new URLSearchParams(filters);
    params.append('format', format);
    params.append('include_stats', 'true');

    showToast(`正在导出${format.toUpperCase()}格式...`, 'info');
    window.location.href = `/admin/api/logs/export?${params.toString()}`;
}

function streamExportLogs(format) {
    const filters = getCurrentFilters();
    delete filters.page;
    delete filters.page_size;
    
    const params = new URLSearchParams(filters);
    params.append('format', format);

    showToast('正在流式导出数据...', 'info');
    window.location.href = `/admin/api/logs/export/stream?${params.toString()}`;
}

function copyLogSession(sessionId) {
    navigator.clipboard.writeText(sessionId).then(() => {
        showToast('会话ID已复制', 'success');
    }).catch(err => {
        console.error('Copy failed:', err);
        showToast('复制失败', 'error');
    });
}

async function showDetail(id) {
    try {
        const response = await fetch(`/admin/api/logs/${id}`, {
            headers: {
                'Authorization': 'Bearer ' + localStorage.getItem('admin_token')
            }
        });
        
        const result = await response.json();
        if (result.code === 0) {
            showLogDetailModal(result.data);
        } else {
            showToast('获取详情失败', 'error');
        }
    } catch (error) {
        console.error('Load log detail failed:', error);
        showToast('获取详情失败', 'error');
    }
}

function showLogDetailModal(log) {
    const modalContent = `
        <div class="modal fade" id="logDetailModal" tabindex="-1">
            <div class="modal-dialog modal-lg">
                <div class="modal-content">
                    <div class="modal-header">
                        <h5 class="modal-title">日志详情 #${log.id}</h5>
                        <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
                    </div>
                    <div class="modal-body">
                        <div class="row">
                            <div class="col-md-6">
                                <p><strong>会话ID:</strong> <code>${escapeHtml(log.session_id || '')}</code></p>
                                <p><strong>验证码类型:</strong> ${getCaptchaTypeText(log.captcha_type)}</p>
                                <p><strong>状态:</strong> <span class="badge bg-${getStatusBadgeClass(log.status)}">${getStatusText(log.status)}</span></p>
                                <p><strong>风险等级:</strong> <span class="badge bg-${getRiskBadgeClass(log.risk_level)}">${getRiskLevelText(log.risk_level)}</span></p>
                            </div>
                            <div class="col-md-6">
                                <p><strong>风险评分:</strong> ${(log.risk_score || 0).toFixed(2)}</p>
                                <p><strong>处理时长:</strong> ${log.duration || 0}ms</p>
                                <p><strong>IP地址:</strong> ${log.ip_address || '-'}</p>
                                <p><strong>时间:</strong> ${formatDate(log.created_at)}</p>
                            </div>
                        </div>
                        <div class="mt-3">
                            <p><strong>User Agent:</strong></p>
                            <pre class="bg-light p-2 rounded" style="white-space: pre-wrap; word-break: break-all;">${escapeHtml(log.user_agent || '-')}</pre>
                        </div>
                        ${log.analysis_result ? `
                        <div class="mt-3">
                            <p><strong>分析结果:</strong></p>
                            <pre class="bg-light p-2 rounded" style="white-space: pre-wrap; word-break: break-all;">${escapeHtml(log.analysis_result)}</pre>
                        </div>
                        ` : ''}
                    </div>
                    <div class="modal-footer">
                        <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">关闭</button>
                        <button type="button" class="btn btn-primary" onclick="copyLogSession('${log.session_id || ''}')">
                            <i class="fas fa-copy"></i> 复制会话ID
                        </button>
                    </div>
                </div>
            </div>
        </div>
    `;
    
    const existingModal = document.getElementById('logDetailModal');
    if (existingModal) {
        existingModal.remove();
    }
    
    document.body.insertAdjacentHTML('beforeend', modalContent);
    const modal = new bootstrap.Modal(document.getElementById('logDetailModal'));
    modal.show();
    
    document.getElementById('logDetailModal').addEventListener('hidden.bs.modal', function() {
        this.remove();
    });
}

async function showAnalytics() {
    const modalContent = `
        <div class="modal fade" id="analyticsModal" tabindex="-1">
            <div class="modal-dialog modal-xl">
                <div class="modal-content">
                    <div class="modal-header">
                        <h5 class="modal-title">日志分析</h5>
                        <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
                    </div>
                    <div class="modal-body">
                        <div class="text-center">
                            <i class="fas fa-spinner fa-spin fa-2x"></i>
                            <p class="mt-2">加载分析数据中...</p>
                        </div>
                    </div>
                    <div class="modal-footer">
                        <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">关闭</button>
                    </div>
                </div>
            </div>
        </div>
    `;
    
    const existingModal = document.getElementById('analyticsModal');
    if (existingModal) {
        existingModal.remove();
    }
    
    document.body.insertAdjacentHTML('beforeend', modalContent);
    const modal = new bootstrap.Modal(document.getElementById('analyticsModal'));
    modal.show();
    
    try {
        const filters = getCurrentFilters();
        const params = new URLSearchParams({
            start_date: filters.start_date,
            end_date: filters.end_date
        });
        
        const response = await fetch(`/admin/api/logs/analytics?${params.toString()}`, {
            headers: {
                'Authorization': 'Bearer ' + localStorage.getItem('admin_token')
            }
        });
        
        const result = await response.json();
        if (result.code === 0) {
            currentAnalytics = result.data;
            renderAnalyticsContent(result.data);
        } else {
            showToast('获取分析数据失败', 'error');
        }
    } catch (error) {
        console.error('Load analytics failed:', error);
        showToast('获取分析数据失败', 'error');
    }
    
    document.getElementById('analyticsModal').addEventListener('hidden.bs.modal', function() {
        this.remove();
    });
}

function renderAnalyticsContent(data) {
    const modalBody = document.querySelector('#analyticsModal .modal-body');
    
    if (!modalBody) return;
    
    const typeBreakdownHtml = Object.entries(data.type_breakdown || {}).map(([type, count]) => 
        `<tr><td>${getCaptchaTypeText(type)}</td><td>${count}</td><td>${((count / data.total_count) * 100).toFixed(1)}%</td></tr>`
    ).join('');
    
    const riskDistHtml = Object.entries(data.risk_distribution || {}).map(([level, count]) => 
        `<tr><td><span class="badge bg-${getRiskBadgeClass(level)}">${getRiskLevelText(level)}</span></td><td>${count}</td></tr>`
    ).join('');
    
    modalBody.innerHTML = `
        <div class="row">
            <div class="col-md-3">
                <div class="card bg-primary text-white">
                    <div class="card-body text-center">
                        <h3>${data.total_count || 0}</h3>
                        <p class="mb-0">总验证数</p>
                    </div>
                </div>
            </div>
            <div class="col-md-3">
                <div class="card bg-success text-white">
                    <div class="card-body text-center">
                        <h3>${data.success_count || 0}</h3>
                        <p class="mb-0">成功数</p>
                    </div>
                </div>
            </div>
            <div class="col-md-3">
                <div class="card bg-danger text-white">
                    <div class="card-body text-center">
                        <h3>${data.failed_count || 0}</h3>
                        <p class="mb-0">失败数</p>
                    </div>
                </div>
            </div>
            <div class="col-md-3">
                <div class="card bg-info text-white">
                    <div class="card-body text-center">
                        <h3>${(data.success_rate || 0).toFixed(1)}%</h3>
                        <p class="mb-0">成功率</p>
                    </div>
                </div>
            </div>
        </div>
        
        <div class="row mt-4">
            <div class="col-md-6">
                <h5>类型分布</h5>
                <table class="table table-sm">
                    <thead><tr><th>类型</th><th>数量</th><th>占比</th></tr></thead>
                    <tbody>${typeBreakdownHtml || '<tr><td colspan="3">暂无数据</td></tr>'}</tbody>
                </table>
            </div>
            <div class="col-md-6">
                <h5>风险分布</h5>
                <table class="table table-sm">
                    <thead><tr><th>风险等级</th><th>数量</th></tr></thead>
                    <tbody>${riskDistHtml || '<tr><td colspan="2">暂无数据</td></tr>'}</tbody>
                </table>
            </div>
        </div>
        
        <div class="row mt-4">
            <div class="col-md-4">
                <p><strong>平均风险评分:</strong> ${(data.avg_risk_score || 0).toFixed(2)}</p>
            </div>
            <div class="col-md-4">
                <p><strong>平均处理时长:</strong> ${(data.avg_duration || 0).toFixed(0)}ms</p>
            </div>
            <div class="col-md-4">
                <p><strong>待处理数:</strong> ${data.pending_count || 0}</p>
            </div>
        </div>
        
        ${data.top_ips && data.top_ips.length > 0 ? `
        <div class="row mt-4">
            <div class="col-12">
                <h5>Top 10 IP地址</h5>
                <table class="table table-sm table-hover">
                    <thead><tr><th>IP地址</th><th>访问次数</th></tr></thead>
                    <tbody>
                        ${data.top_ips.slice(0, 10).map(ip => 
                            `<tr><td>${escapeHtml(ip.ip_address)}</td><td>${ip.count}</td></tr>`
                        ).join('')}
                    </tbody>
                </table>
            </div>
        </div>
        ` : ''}
    `;
}

function formatDate(dateStr) {
    if (!dateStr) return '-';
    const date = new Date(dateStr);
    if (isNaN(date.getTime())) return dateStr;
    
    const pad = n => String(n).padStart(2, '0');
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
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
