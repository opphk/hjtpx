let blacklistPage = 1;
let blacklistPageSize = 10;
let currentBlacklist = [];
let currentView = 'table';
let selectedBlacklistItems = new Set();

document.addEventListener('DOMContentLoaded', () => {
    loadBlacklistSummary();
    loadBlacklist();
    setupEventListeners();
});

function setupEventListeners() {
    const addBtn = document.getElementById('addBlacklistBtn');
    if (addBtn) {
        addBtn.addEventListener('click', () => openBlacklistModal());
    }

    const importBtn = document.getElementById('importBlacklistBtn');
    if (importBtn) {
        importBtn.addEventListener('click', () => {
            const modal = new bootstrap.Modal(document.getElementById('importModal'));
            modal.show();
        });
    }

    const confirmImportBtn = document.getElementById('confirmImportBtn');
    if (confirmImportBtn) {
        confirmImportBtn.addEventListener('click', handleImport);
    }

    const searchBtn = document.getElementById('searchBlacklistBtn');
    if (searchBtn) {
        searchBtn.addEventListener('click', () => {
            blacklistPage = 1;
            loadBlacklist();
        });
    }

    const selectAllCheckbox = document.getElementById('selectAllBlacklist');
    if (selectAllCheckbox) {
        selectAllCheckbox.addEventListener('change', (e) => {
            const checkboxes = document.querySelectorAll('.bl-checkbox');
            checkboxes.forEach(cb => cb.checked = e.target.checked);
            toggleSelectAll(e.target.checked);
        });
    }

    const viewButtons = document.querySelectorAll('[data-view]');
    viewButtons.forEach(btn => {
        btn.addEventListener('click', (e) => {
            viewButtons.forEach(b => b.classList.remove('active'));
            e.target.classList.add('active');
            switchView(e.target.dataset.view);
        });
    });

    const expirationSelect = document.getElementById('blExpiration');
    if (expirationSelect) {
        expirationSelect.addEventListener('change', (e) => {
            const customGroup = document.getElementById('customExpirationGroup');
            if (e.target.value === 'custom') {
                customGroup.classList.remove('d-none');
            } else {
                customGroup.classList.add('d-none');
            }
        });
    }

    const form = document.getElementById('blacklistForm');
    if (form) {
        form.addEventListener('submit', handleBlacklistSubmit);
    }

    const batchDeleteBtn = document.getElementById('batchDeleteBtn');
    if (batchDeleteBtn) {
        batchDeleteBtn.addEventListener('click', () => batchDeleteBlacklist());
    }

    const batchUnblockBtn = document.getElementById('batchUnblockBtn');
    if (batchUnblockBtn) {
        batchUnblockBtn.addEventListener('click', () => batchUnblockBlacklist());
    }

    const batchUpdateBtn = document.getElementById('batchUpdateBtn');
    if (batchUpdateBtn) {
        batchUpdateBtn.addEventListener('click', () => showBatchUpdateModal());
    }
}

async function loadBlacklistSummary() {
    const mockSummary = getMockBlacklistSummary();

    try {
        const data = await auth.request('/admin/blacklist/summary');
        if (data.code === 0) {
            updateBlacklistSummary(data.data);
        } else {
            updateBlacklistSummary(mockSummary);
        }
    } catch (error) {
        updateBlacklistSummary(mockSummary);
    }
}

function getMockBlacklistSummary() {
    return {
        total: 1234,
        todayAdded: 56,
        autoUnblocked: 23,
        totalBlocked: 5678
    };
}

function updateBlacklistSummary(summary) {
    const totalEl = document.getElementById('totalBlacklist');
    const todayEl = document.getElementById('todayAdded');
    const autoEl = document.getElementById('autoUnblocked');
    const blockedEl = document.getElementById('totalBlocked');

    if (totalEl) totalEl.textContent = formatNumber(summary.total);
    if (todayEl) todayEl.textContent = formatNumber(summary.todayAdded);
    if (autoEl) autoEl.textContent = formatNumber(summary.autoUnblocked);
    if (blockedEl) blockedEl.textContent = formatNumber(summary.totalBlocked);
}

function formatNumber(num) {
    if (num >= 1000000) {
        return (num / 1000000).toFixed(1) + 'M';
    } else if (num >= 1000) {
        return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
}

async function loadBlacklist() {
    const type = document.getElementById('blacklistType')?.value || '';
    const source = document.getElementById('blacklistSource')?.value || '';
    const status = document.getElementById('blacklistStatus')?.value || '';
    const keyword = document.getElementById('blacklistKeyword')?.value || '';
    const mockData = getMockBlacklist();

    try {
        const params = new URLSearchParams({
            page: blacklistPage,
            size: blacklistPageSize,
            type, source, status, keyword
        });

        const result = await auth.request(`/admin/blacklist?${params.toString()}`);
        if (result.code === 0) {
            currentBlacklist = result.data.list || [];
            renderBlacklistPagination(result.data.total || currentBlacklist.length);
            renderBlacklistCount(result.data.total || currentBlacklist.length);
        } else {
            currentBlacklist = filterBlacklist(mockData, type, source, status, keyword);
            renderBlacklistPagination(currentBlacklist.length);
            renderBlacklistCount(currentBlacklist.length);
        }
    } catch (error) {
        currentBlacklist = filterBlacklist(mockData, type, source, status, keyword);
        renderBlacklistPagination(currentBlacklist.length);
        renderBlacklistCount(currentBlacklist.length);
    }

    renderBlacklist();
}

function getMockBlacklist() {
    return [
        {
            id: 1, target: '192.168.1.100', type: 'ip', source: 'auto',
            reason: '检测到恶意扫描行为', expiration: '2025-06-01', status: 'active',
            action: 'block', hitCount: 234, apps: ['all'], createdBy: 'system',
            createdAt: '2025-05-01 10:00:00'
        },
        {
            id: 2, target: 'user_malicious_001', type: 'user_id', source: 'manual',
            reason: '多次违规操作', expiration: 'permanent', status: 'active',
            action: 'block', hitCount: 567, apps: ['1', '2'], createdBy: 'admin',
            createdAt: '2025-04-15 14:30:00'
        },
        {
            id: 3, target: 'device_fp_abc123', type: 'device_id', source: 'auto',
            reason: '设备指纹异常', expiration: '2025-05-20', status: 'active',
            action: 'captcha', hitCount: 89, apps: ['all'], createdBy: 'system',
            createdAt: '2025-05-10 09:15:00'
        },
        {
            id: 4, target: '138****8888', type: 'phone', source: 'import',
            reason: '批量注册账号', expiration: '2025-07-01', status: 'active',
            action: 'block', hitCount: 1234, apps: ['1'], createdBy: 'admin',
            createdAt: '2025-05-05 11:20:00'
        },
        {
            id: 5, target: 'spam@example.com', type: 'email', source: 'manual',
            reason: '垃圾邮件发送源', expiration: '2025-08-01', status: 'active',
            action: 'review', hitCount: 45, apps: ['all'], createdBy: 'admin',
            createdAt: '2025-04-20 16:45:00'
        },
        {
            id: 6, target: '10.0.0.0/24', type: 'ip', source: 'auto',
            reason: 'IP段异常流量', expiration: '2025-05-18', status: 'expired',
            action: 'block', hitCount: 789, apps: ['all'], createdBy: 'system',
            createdAt: '2025-05-10 08:00:00'
        },
        {
            id: 7, target: 'bad_user_007', type: 'user_id', source: 'manual',
            reason: '违反使用协议', expiration: '2025-06-15', status: 'active',
            action: 'block', hitCount: 321, apps: ['2', '3'], createdBy: 'admin',
            createdAt: '2025-05-12 13:00:00'
        },
        {
            id: 8, target: '138****9999', type: 'phone', source: 'import',
            reason: '批量营销电话', expiration: 'permanent', status: 'unblocked',
            action: 'captcha', hitCount: 56, apps: ['1'], createdBy: 'admin',
            createdAt: '2025-03-01 10:00:00'
        }
    ];
}

function filterBlacklist(list, type, source, status, keyword) {
    return list.filter(item => {
        if (type && item.type !== type) return false;
        if (source && item.source !== source) return false;
        if (status && item.status !== status) return false;
        if (keyword && !item.target.toLowerCase().includes(keyword.toLowerCase())) return false;
        return true;
    });
}

function renderBlacklist() {
    if (currentView === 'table') {
        renderBlacklistTable();
    } else {
        renderBlacklistCards();
    }
}

function renderBlacklistTable() {
    const tbody = document.getElementById('blacklistTableBody');
    if (!tbody) return;

    document.getElementById('blacklistTableView')?.classList.remove('d-none');
    document.getElementById('blacklistCardView')?.classList.add('d-none');

    tbody.innerHTML = currentBlacklist.map(item => `
        <tr class="${item.status !== 'active' ? 'text-muted' : ''}">
            <td><input type="checkbox" class="bl-checkbox" data-id="${item.id}"></td>
            <td>
                <strong>${escapeHtml(item.target)}</strong>
                <br><small class="text-muted">ID: ${item.id}</small>
            </td>
            <td><span class="badge ${getTypeBadgeClass(item.type)}">${getTypeText(item.type)}</span></td>
            <td><span class="badge bg-secondary">${getSourceText(item.source)}</span></td>
            <td><small>${escapeHtml(item.reason)}</small></td>
            <td><small>${item.expiration === 'permanent' ? '永久' : item.expiration}</small></td>
            <td><span class="badge ${getStatusBadgeClass(item.status)}">${getStatusText(item.status)}</span></td>
            <td>
                <div class="btn-group btn-group-sm">
                    <button class="btn btn-outline-secondary" onclick="viewBlacklistDetail(${item.id})" title="查看"><i class="fas fa-eye"></i></button>
                    ${item.status === 'active' ? `
                        <button class="btn btn-outline-success" onclick="unblockItem(${item.id})" title="解封"><i class="fas fa-unlock"></i></button>
                    ` : ''}
                    <button class="btn btn-outline-danger" onclick="deleteBlacklistItem(${item.id})" title="删除"><i class="fas fa-trash"></i></button>
                </div>
            </td>
        </tr>
    `).join('');
}

function renderBlacklistCards() {
    const container = document.getElementById('blacklistCardView');
    if (!container) return;

    container.classList.remove('d-none');
    document.getElementById('blacklistTableView')?.classList.add('d-none');

    container.innerHTML = `<div class="row g-3 p-3">${currentBlacklist.map(item => `
        <div class="col-md-6 col-lg-4">
            <div class="card border ${item.status !== 'active' ? 'border-secondary' : 'border-danger'}">
                <div class="card-header bg-transparent d-flex justify-content-between align-items-center py-2">
                    <span class="badge ${getTypeBadgeClass(item.type)}">${getTypeText(item.type)}</span>
                    <span class="badge ${getStatusBadgeClass(item.status)}">${getStatusText(item.status)}</span>
                </div>
                <div class="card-body py-2">
                    <h6 class="card-title mb-1 text-truncate" title="${escapeHtml(item.target)}">${escapeHtml(item.target)}</h6>
                    <p class="card-text small text-muted mb-2 text-truncate">${escapeHtml(item.reason)}</p>
                    <div class="d-flex justify-content-between align-items-center">
                        <small class="text-muted">命中 ${item.hitCount} 次</small>
                        ${item.expiration !== 'permanent' ? `<small class="text-muted">${item.expiration}</small>` : '<small class="text-danger">永久</small>'}
                    </div>
                </div>
                <div class="card-footer bg-transparent py-2">
                    <div class="btn-group btn-group-sm w-100">
                        <button class="btn btn-outline-secondary" onclick="viewBlacklistDetail(${item.id})"><i class="fas fa-eye me-1"></i>详情</button>
                        ${item.status === 'active' ? `
                            <button class="btn btn-outline-success" onclick="unblockItem(${item.id})"><i class="fas fa-unlock me-1"></i>解封</button>
                        ` : ''}
                        <button class="btn btn-outline-danger" onclick="deleteBlacklistItem(${item.id})"><i class="fas fa-trash me-1"></i>删除</button>
                    </div>
                </div>
            </div>
        </div>
    `).join('')}</div>`;
}

function switchView(view) {
    currentView = view;
    renderBlacklist();
}

function renderBlacklistCount(total) {
    const countEl = document.getElementById('blacklistCount');
    if (countEl) countEl.textContent = total;
}

function renderBlacklistPagination(total) {
    const pagination = document.getElementById('blacklistPagination');
    if (!pagination) return;

    const totalPages = Math.ceil(total / blacklistPageSize);
    if (totalPages <= 1) {
        pagination.innerHTML = '';
        return;
    }

    let html = '<div class="d-flex justify-content-between align-items-center">';
    html += `<span class="text-muted">第 ${blacklistPage} / ${totalPages} 页，共 ${total} 条</span>`;
    html += '<div class="btn-group btn-group-sm">';
    html += `<button class="btn btn-outline-secondary" onclick="changeBlacklistPage(${blacklistPage - 1})" ${blacklistPage === 1 ? 'disabled' : ''}>上一页</button>`;

    const startPage = Math.max(1, blacklistPage - 2);
    const endPage = Math.min(totalPages, blacklistPage + 2);

    if (startPage > 1) {
        html += `<button class="btn btn-outline-secondary" onclick="changeBlacklistPage(1)">1</button>`;
        if (startPage > 2) html += `<button class="btn btn-outline-secondary" disabled>...</button>`;
    }

    for (let i = startPage; i <= endPage; i++) {
        html += `<button class="btn ${i === blacklistPage ? 'btn-primary' : 'btn-outline-secondary'}" onclick="changeBlacklistPage(${i})">${i}</button>`;
    }

    if (endPage < totalPages) {
        if (endPage < totalPages - 1) html += `<button class="btn btn-outline-secondary" disabled>...</button>`;
        html += `<button class="btn btn-outline-secondary" onclick="changeBlacklistPage(${totalPages})">${totalPages}</button>`;
    }

    html += `<button class="btn btn-outline-secondary" onclick="changeBlacklistPage(${blacklistPage + 1})" ${blacklistPage === totalPages ? 'disabled' : ''}>下一页</button>`;
    html += '</div></div>';

    pagination.innerHTML = html;
}

function changeBlacklistPage(page) {
    blacklistPage = page;
    loadBlacklist();
}

function openBlacklistModal(item = null) {
    const modal = document.getElementById('blacklistModal');
    const title = document.getElementById('blacklistModalTitle');
    const form = document.getElementById('blacklistForm');

    if (item) {
        title.textContent = '编辑黑名单';
        document.getElementById('blacklistId').value = item.id;
        document.getElementById('blType').value = item.type;
        document.getElementById('blValue').value = item.target;
        document.getElementById('blReason').value = item.reason;
        document.getElementById('blAction').value = item.action;
        document.getElementById('blNote').value = item.note || '';
    } else {
        title.textContent = '添加黑名单';
        form.reset();
        document.getElementById('blacklistId').value = '';
    }

    const bsModal = new bootstrap.Modal(modal);
    bsModal.show();
}

async function handleBlacklistSubmit(e) {
    e.preventDefault();

    const id = document.getElementById('blacklistId').value;
    const expirationSelect = document.getElementById('blExpiration').value;
    let expiration = expirationSelect;

    if (expiration === 'custom') {
        expiration = document.getElementById('blCustomExpiration').value;
    }

    const data = {
        type: document.getElementById('blType').value,
        target: document.getElementById('blValue').value,
        reason: document.getElementById('blReason').value,
        action: document.getElementById('blAction').value,
        expiration: expiration,
        apps: Array.from(document.getElementById('blApps').selectedOptions).map(o => o.value),
        note: document.getElementById('blNote').value
    };

    try {
        if (id) {
            await auth.request(`/admin/blacklist/${id}`, { method: 'PUT', body: JSON.stringify(data) });
            showToast('黑名单更新成功', 'success');
        } else {
            await auth.request('/admin/blacklist', { method: 'POST', body: JSON.stringify(data) });
            showToast('黑名单添加成功', 'success');
        }

        bootstrap.Modal.getInstance(document.getElementById('blacklistModal'))?.hide();
        loadBlacklist();
        loadBlacklistSummary();
    } catch (error) {
        showToast('保存失败', 'danger');
    }
}

async function handleImport() {
    const fileInput = document.getElementById('importFile');
    const importType = document.getElementById('importType').value;
    const importReason = document.getElementById('importReason').value;

    if (!fileInput.files.length) {
        showToast('请选择要导入的文件', 'warning');
        return;
    }

    const file = fileInput.files[0];
    const reader = new FileReader();

    reader.onload = async (e) => {
        const content = e.target.result;
        const targets = content.split(/\r?\n/).filter(line => line.trim());

        const data = {
            type: importType,
            targets: targets,
            reason: importReason
        };

        try {
            await auth.request('/admin/blacklist/import', { method: 'POST', body: JSON.stringify(data) });
            showToast(`成功导入 ${targets.length} 条记录`, 'success');
            bootstrap.Modal.getInstance(document.getElementById('importModal'))?.hide();
            loadBlacklist();
            loadBlacklistSummary();
        } catch (error) {
            showToast('导入失败', 'danger');
        }
    };

    reader.readAsText(file);
}

async function unblockItem(id) {
    if (!confirm('确定要解封这条记录吗？')) return;

    try {
        await auth.request(`/admin/blacklist/${id}/unblock`, { method: 'POST' });
        showToast('解封成功', 'success');
        loadBlacklist();
        loadBlacklistSummary();
    } catch (error) {
        showToast('解封失败', 'danger');
    }
}

async function deleteBlacklistItem(id) {
    if (!confirm('确定要删除这条黑名单记录吗？')) return;

    try {
        await auth.request(`/admin/blacklist/${id}`, { method: 'DELETE' });
        showToast('删除成功', 'success');
        loadBlacklist();
        loadBlacklistSummary();
    } catch (error) {
        showToast('删除失败', 'danger');
    }
}

function viewBlacklistDetail(id) {
    const item = currentBlacklist.find(i => i.id === id);
    if (!item) return;

    const modal = document.getElementById('detailModal');
    const content = document.getElementById('detailContent');

    content.innerHTML = `
        <table class="table table-borderless">
            <tr><td class="text-muted" style="width: 100px;">ID</td><td>${item.id}</td></tr>
            <tr><td class="text-muted">目标</td><td><code>${escapeHtml(item.target)}</code></td></tr>
            <tr><td class="text-muted">类型</td><td><span class="badge ${getTypeBadgeClass(item.type)}">${getTypeText(item.type)}</span></td></tr>
            <tr><td class="text-muted">来源</td><td><span class="badge bg-secondary">${getSourceText(item.source)}</span></td></tr>
            <tr><td class="text-muted">原因</td><td>${escapeHtml(item.reason)}</td></tr>
            <tr><td class="text-muted">处置方式</td><td><span class="badge ${getActionBadgeClass(item.action)}">${getActionText(item.action)}</span></td></tr>
            <tr><td class="text-muted">过期时间</td><td>${item.expiration === 'permanent' ? '永久' : item.expiration}</td></tr>
            <tr><td class="text-muted">状态</td><td><span class="badge ${getStatusBadgeClass(item.status)}">${getStatusText(item.status)}</span></td></tr>
            <tr><td class="text-muted">命中次数</td><td class="text-danger fw-bold">${item.hitCount}</td></tr>
            <tr><td class="text-muted">添加人</td><td>${item.createdBy}</td></tr>
            <tr><td class="text-muted">添加时间</td><td>${item.createdAt}</td></tr>
        </table>
    `;

    const bsModal = new bootstrap.Modal(modal);
    bsModal.show();
}

function getTypeBadgeClass(type) {
    const map = {
        'ip': 'bg-danger',
        'user_id': 'bg-primary',
        'device_id': 'bg-info',
        'phone': 'bg-warning',
        'email': 'bg-secondary'
    };
    return map[type] || 'bg-secondary';
}

function getTypeText(type) {
    const map = {
        'ip': 'IP地址',
        'user_id': '用户ID',
        'device_id': '设备ID',
        'phone': '手机号',
        'email': '邮箱'
    };
    return map[type] || type;
}

function getSourceText(source) {
    const map = {
        'manual': '手动',
        'auto': '自动',
        'import': '导入'
    };
    return map[source] || source;
}

function getActionBadgeClass(action) {
    const map = {
        'block': 'bg-danger',
        'captcha': 'bg-warning',
        'review': 'bg-info'
    };
    return map[action] || 'bg-secondary';
}

function getActionText(action) {
    const map = {
        'block': '拦截',
        'captcha': '验证码',
        'review': '审核'
    };
    return map[action] || action;
}

function getStatusBadgeClass(status) {
    const map = {
        'active': 'bg-success',
        'expired': 'bg-secondary',
        'unblocked': 'bg-warning'
    };
    return map[status] || 'bg-secondary';
}

function getStatusText(status) {
    const map = {
        'active': '生效中',
        'expired': '已过期',
        'unblocked': '已解封'
    };
    return map[status] || status;
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

function toggleSelectAll(selectAll) {
    if (selectAll) {
        currentBlacklist.forEach(item => selectedBlacklistItems.add(item.id));
    } else {
        selectedBlacklistItems.clear();
    }
    updateBatchToolbar();
}

function toggleBlacklistSelection(id) {
    if (selectedBlacklistItems.has(id)) {
        selectedBlacklistItems.delete(id);
    } else {
        selectedBlacklistItems.add(id);
    }
    updateBatchToolbar();
}

function updateBatchToolbar() {
    const batchToolbar = document.getElementById('batchToolbar');
    const selectedCount = document.getElementById('selectedBlacklistCount');
    
    if (batchToolbar) {
        if (selectedBlacklistItems.size > 0) {
            batchToolbar.classList.remove('d-none');
        } else {
            batchToolbar.classList.add('d-none');
        }
    }
    
    if (selectedCount) {
        selectedCount.textContent = selectedBlacklistItems.size;
    }
}

async function batchDeleteBlacklist() {
    if (selectedBlacklistItems.size === 0) {
        showToast('请先选择要删除的记录', 'warning');
        return;
    }
    
    if (!confirm(`确定要删除选中的 ${selectedBlacklistItems.size} 条黑名单记录吗？此操作不可恢复。`)) {
        return;
    }
    
    try {
        const deletePromises = Array.from(selectedBlacklistItems).map(id => 
            auth.request(`/admin/blacklist/${id}`, { method: 'DELETE' })
        );
        
        await Promise.all(deletePromises);
        showToast(`成功删除 ${selectedBlacklistItems.size} 条记录`, 'success');
        selectedBlacklistItems.clear();
        loadBlacklist();
        loadBlacklistSummary();
    } catch (error) {
        showToast('批量删除失败', 'danger');
    }
}

async function batchUnblockBlacklist() {
    if (selectedBlacklistItems.size === 0) {
        showToast('请先选择要解封的记录', 'warning');
        return;
    }
    
    if (!confirm(`确定要解封选中的 ${selectedBlacklistItems.size} 条记录吗？`)) {
        return;
    }
    
    try {
        const unblockPromises = Array.from(selectedBlacklistItems).map(id => 
            auth.request(`/admin/blacklist/${id}/unblock`, { method: 'POST' })
        );
        
        await Promise.all(unblockPromises);
        showToast(`成功解封 ${selectedBlacklistItems.size} 条记录`, 'success');
        selectedBlacklistItems.clear();
        loadBlacklist();
        loadBlacklistSummary();
    } catch (error) {
        showToast('批量解封失败', 'danger');
    }
}

function showBatchUpdateModal() {
    if (selectedBlacklistItems.size === 0) {
        showToast('请先选择要更新的记录', 'warning');
        return;
    }
    
    const modal = new bootstrap.Modal(document.getElementById('batchUpdateModal'));
    document.getElementById('batchUpdateCount').textContent = selectedBlacklistItems.size;
    modal.show();
}

async function applyBatchUpdate() {
    const newAction = document.getElementById('batchNewAction')?.value;
    const newExpiration = document.getElementById('batchNewExpiration')?.value;
    
    if (!newAction && !newExpiration) {
        showToast('请至少选择一个要更新的字段', 'warning');
        return;
    }
    
    try {
        const updateData = {};
        if (newAction) updateData.action = newAction;
        if (newExpiration) updateData.expiration = newExpiration;
        
        const updatePromises = Array.from(selectedBlacklistItems).map(id => 
            auth.request(`/admin/blacklist/${id}`, {
                method: 'PUT',
                body: JSON.stringify(updateData)
            })
        );
        
        await Promise.all(updatePromises);
        showToast(`成功更新 ${selectedBlacklistItems.size} 条记录`, 'success');
        
        bootstrap.Modal.getInstance(document.getElementById('batchUpdateModal'))?.hide();
        selectedBlacklistItems.clear();
        loadBlacklist();
        loadBlacklistSummary();
    } catch (error) {
        showToast('批量更新失败', 'danger');
    }
}

function exportBlacklist() {
    const exportItems = selectedBlacklistItems.size > 0 ? 
        currentBlacklist.filter(item => selectedBlacklistItems.has(item.id)) : 
        currentBlacklist;
    
    if (exportItems.length === 0) {
        showToast('没有可导出的记录', 'warning');
        return;
    }
    
    const csvContent = [
        ['ID', '目标', '类型', '来源', '原因', '处置方式', '过期时间', '状态', '命中次数', '添加时间'].join(','),
        ...exportItems.map(item => [
            item.id,
            `"${escapeHtml(item.target)}"`,
            item.type,
            item.source,
            `"${escapeHtml(item.reason)}"`,
            item.action,
            item.expiration === 'permanent' ? '永久' : item.expiration,
            item.status,
            item.hitCount,
            item.createdAt
        ].join(','))
    ].join('\n');
    
    downloadFile(csvContent, `blacklist_${new Date().toISOString().slice(0,10)}.csv`, 'text/csv;charset=utf-8');
    showToast(`已导出 ${exportItems.length} 条记录`, 'success');
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

function clearBlacklistSelection() {
    selectedBlacklistItems.clear();
    const checkboxes = document.querySelectorAll('.bl-checkbox');
    checkboxes.forEach(cb => cb.checked = false);
    updateBatchToolbar();
    showToast('已取消选择', 'info');
}
