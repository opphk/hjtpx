let blacklistPage = 1;
let whitelistPage = 1;
let blacklistPageSize = 10;
let whitelistPageSize = 10;
let currentBlacklist = [];
let currentWhitelist = [];
let currentView = 'table';
let selectedBlacklist = new Set();
let activeTab = 'blacklist';

document.addEventListener('DOMContentLoaded', () => {
    setupTabs();
    loadBlacklistSummary();
    loadBlacklist();
    setupEventListeners();
});

function setupTabs() {
    const tabLinks = document.querySelectorAll('[data-tab]');
    tabLinks.forEach(link => {
        link.addEventListener('click', (e) => {
            e.preventDefault();
            tabLinks.forEach(l => l.classList.remove('active'));
            link.classList.add('active');
            activeTab = link.dataset.tab;
            
            if (activeTab === 'blacklist') {
                document.getElementById('blacklistSection').classList.remove('d-none');
                document.getElementById('whitelistSection').classList.add('d-none');
                loadBlacklist();
            } else {
                document.getElementById('blacklistSection').classList.add('d-none');
                document.getElementById('whitelistSection').classList.remove('d-none');
                loadWhitelist();
            }
        });
    });
}

function setupEventListeners() {
    const addBtn = document.getElementById('addBlacklistBtn');
    if (addBtn) {
        addBtn.addEventListener('click', () => {
            if (activeTab === 'blacklist') {
                openBlacklistModal();
            } else {
                openWhitelistModal();
            }
        });
    }

    const importBtn = document.getElementById('importBlacklistBtn');
    if (importBtn) {
        importBtn.addEventListener('click', () => {
            if (activeTab === 'blacklist') {
                const modal = new bootstrap.Modal(document.getElementById('importModal'));
                modal.show();
            } else {
                const modal = new bootstrap.Modal(document.getElementById('whitelistImportModal'));
                modal.show();
            }
        });
    }

    const confirmImportBtn = document.getElementById('confirmImportBtn');
    if (confirmImportBtn) {
        confirmImportBtn.addEventListener('click', handleImport);
    }

    const confirmWhitelistImportBtn = document.getElementById('confirmWhitelistImportBtn');
    if (confirmWhitelistImportBtn) {
        confirmWhitelistImportBtn.addEventListener('click', handleWhitelistImport);
    }

    const searchBtn = document.getElementById('searchBlacklistBtn');
    if (searchBtn) {
        searchBtn.addEventListener('click', () => {
            blacklistPage = 1;
            whitelistPage = 1;
            if (activeTab === 'blacklist') {
                loadBlacklist();
            } else {
                loadWhitelist();
            }
        });
    }

    const selectAllCheckbox = document.getElementById('selectAllBlacklist');
    if (selectAllCheckbox) {
        selectAllCheckbox.addEventListener('change', (e) => {
            const checkboxes = document.querySelectorAll('.bl-checkbox');
            checkboxes.forEach(cb => {
                cb.checked = e.target.checked;
                if (e.target.checked) {
                    selectedBlacklist.add(parseInt(cb.dataset.id));
                } else {
                    selectedBlacklist.delete(parseInt(cb.dataset.id));
                }
            });
            updateSelectedCount();
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

    const whitelistForm = document.getElementById('whitelistForm');
    if (whitelistForm) {
        whitelistForm.addEventListener('submit', handleWhitelistSubmit);
    }

    setupBatchActions();
}

function setupBatchActions() {
    const batchDeleteBtn = document.getElementById('batchDeleteBtn');
    if (batchDeleteBtn) {
        batchDeleteBtn.addEventListener('click', handleBatchDelete);
    }

    const batchUnblockBtn = document.getElementById('batchUnblockBtn');
    if (batchUnblockBtn) {
        batchUnblockBtn.addEventListener('click', handleBatchUnblock);
    }

    const batchExportBtn = document.getElementById('batchExportBtn');
    if (batchExportBtn) {
        batchExportBtn.addEventListener('click', handleBatchExport);
    }

    const batchWhitelistBtn = document.getElementById('batchWhitelistBtn');
    if (batchWhitelistBtn) {
        batchWhitelistBtn.addEventListener('click', handleBatchWhitelist);
    }
}

function updateSelectedCount() {
    const countEl = document.getElementById('selectedCount');
    if (countEl) {
        countEl.textContent = selectedBlacklist.size;
    }
}

async function handleBatchDelete() {
    if (selectedBlacklist.size === 0) {
        showToast('请先选择要删除的记录', 'warning');
        return;
    }

    if (!confirm(`确定要删除选中的 ${selectedBlacklist.size} 条记录吗？`)) {
        return;
    }

    try {
        const ids = Array.from(selectedBlacklist);
        await auth.request('/admin/blacklist/batch-delete', {
            method: 'POST',
            body: JSON.stringify({ ids })
        });
        showToast(`成功删除 ${ids.length} 条记录`, 'success');
        selectedBlacklist.clear();
        loadBlacklist();
        loadBlacklistSummary();
    } catch (error) {
        showToast('批量删除失败', 'danger');
    }
}

async function handleBatchUnblock() {
    if (selectedBlacklist.size === 0) {
        showToast('请先选择要解封的记录', 'warning');
        return;
    }

    if (!confirm(`确定要解封选中的 ${selectedBlacklist.size} 条记录吗？`)) {
        return;
    }

    try {
        const ids = Array.from(selectedBlacklist);
        await auth.request('/admin/blacklist/batch-unblock', {
            method: 'POST',
            body: JSON.stringify({ ids })
        });
        showToast(`成功解封 ${ids.length} 条记录`, 'success');
        selectedBlacklist.clear();
        loadBlacklist();
        loadBlacklistSummary();
    } catch (error) {
        showToast('批量解封失败', 'danger');
    }
}

function handleBatchExport() {
    if (selectedBlacklist.size === 0) {
        showToast('请先选择要导出的记录', 'warning');
        return;
    }

    const selectedItems = currentBlacklist.filter(item => selectedBlacklist.has(item.id));
    const dataStr = JSON.stringify(selectedItems, null, 2);
    const blob = new Blob([dataStr], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `blacklist_export_${new Date().toISOString().slice(0,10)}.json`;
    link.click();
    URL.revokeObjectURL(url);
    showToast(`成功导出 ${selectedItems.length} 条记录`, 'success');
}

async function handleBatchWhitelist() {
    if (selectedBlacklist.size === 0) {
        showToast('请先选择要移入白名单的记录', 'warning');
        return;
    }

    if (!confirm(`确定要将选中的 ${selectedBlacklist.size} 条记录移入白名单吗？`)) {
        return;
    }

    try {
        const ids = Array.from(selectedBlacklist);
        await auth.request('/admin/blacklist/move-to-whitelist', {
            method: 'POST',
            body: JSON.stringify({ ids })
        });
        showToast(`成功移入白名单 ${ids.length} 条记录`, 'success');
        selectedBlacklist.clear();
        loadBlacklist();
        loadWhitelist();
        loadBlacklistSummary();
    } catch (error) {
        showToast('批量移入白名单失败', 'danger');
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

async function loadWhitelist() {
    const type = document.getElementById('whitelistType')?.value || '';
    const keyword = document.getElementById('whitelistKeyword')?.value || '';
    const mockData = getMockWhitelist();

    try {
        const params = new URLSearchParams({
            page: whitelistPage,
            size: whitelistPageSize,
            type, keyword
        });

        const result = await auth.request(`/admin/whitelist?${params.toString()}`);
        if (result.code === 0) {
            currentWhitelist = result.data.list || [];
            renderWhitelistPagination(result.data.total || currentWhitelist.length);
            renderWhitelistCount(result.data.total || currentWhitelist.length);
        } else {
            currentWhitelist = filterWhitelist(mockData, type, keyword);
            renderWhitelistPagination(currentWhitelist.length);
            renderWhitelistCount(currentWhitelist.length);
        }
    } catch (error) {
        currentWhitelist = filterWhitelist(mockData, type, keyword);
        renderWhitelistPagination(currentWhitelist.length);
        renderWhitelistCount(currentWhitelist.length);
    }

    renderWhitelist();
}

function getMockWhitelist() {
    return [
        {
            id: 1, target: '192.168.1.1', type: 'ip', source: 'manual',
            reason: '内部测试IP', expiration: 'permanent', status: 'active',
            createdBy: 'admin', createdAt: '2025-01-01 10:00:00'
        },
        {
            id: 2, target: 'trusted_user_001', type: 'user_id', source: 'manual',
            reason: 'VIP用户', expiration: 'permanent', status: 'active',
            createdBy: 'admin', createdAt: '2025-01-15 14:30:00'
        }
    ];
}

function filterWhitelist(list, type, keyword) {
    return list.filter(item => {
        if (type && item.type !== type) return false;
        if (keyword && !item.target.toLowerCase().includes(keyword.toLowerCase())) return false;
        return true;
    });
}

function renderWhitelist() {
    const tbody = document.getElementById('whitelistTableBody');
    if (!tbody) return;

    tbody.innerHTML = currentWhitelist.map(item => `
        <tr>
            <td><input type="checkbox" class="wl-checkbox" data-id="${item.id}"></td>
            <td>
                <strong>${escapeHtml(item.target)}</strong>
                <br><small class="text-muted">ID: ${item.id}</small>
            </td>
            <td><span class="badge ${getTypeBadgeClass(item.type)}">${getTypeText(item.type)}</span></td>
            <td><small>${escapeHtml(item.reason)}</small></td>
            <td><small>${item.expiration === 'permanent' ? '永久' : item.expiration}</small></td>
            <td><span class="badge bg-success">生效中</span></td>
            <td>
                <div class="btn-group btn-group-sm">
                    <button class="btn btn-outline-secondary" onclick="viewWhitelistDetail(${item.id})" title="查看"><i class="fas fa-eye"></i></button>
                    <button class="btn btn-outline-danger" onclick="deleteWhitelistItem(${item.id})" title="删除"><i class="fas fa-trash"></i></button>
                </div>
            </td>
        </tr>
    `).join('');
}

function renderWhitelistCount(total) {
    const countEl = document.getElementById('whitelistCount');
    if (countEl) countEl.textContent = total;
}

function renderWhitelistPagination(total) {
    const pagination = document.getElementById('whitelistPagination');
    if (!pagination) return;

    const totalPages = Math.ceil(total / whitelistPageSize);
    if (totalPages <= 1) {
        pagination.innerHTML = '';
        return;
    }

    let html = '<div class="d-flex justify-content-between align-items-center">';
    html += `<span class="text-muted">第 ${whitelistPage} / ${totalPages} 页，共 ${total} 条</span>`;
    html += '<div class="btn-group btn-group-sm">';
    html += `<button class="btn btn-outline-secondary" onclick="changeWhitelistPage(${whitelistPage - 1})" ${whitelistPage === 1 ? 'disabled' : ''}>上一页</button>`;

    const startPage = Math.max(1, whitelistPage - 2);
    const endPage = Math.min(totalPages, whitelistPage + 2);

    if (startPage > 1) {
        html += `<button class="btn btn-outline-secondary" onclick="changeWhitelistPage(1)">1</button>`;
        if (startPage > 2) html += `<button class="btn btn-outline-secondary" disabled>...</button>`;
    }

    for (let i = startPage; i <= endPage; i++) {
        html += `<button class="btn ${i === whitelistPage ? 'btn-primary' : 'btn-outline-secondary'}" onclick="changeWhitelistPage(${i})">${i}</button>`;
    }

    if (endPage < totalPages) {
        if (endPage < totalPages - 1) html += `<button class="btn btn-outline-secondary" disabled>...</button>`;
        html += `<button class="btn btn-outline-secondary" onclick="changeWhitelistPage(${totalPages})">${totalPages}</button>`;
    }

    html += `<button class="btn btn-outline-secondary" onclick="changeWhitelistPage(${whitelistPage + 1})" ${whitelistPage === totalPages ? 'disabled' : ''}>下一页</button>`;
    html += '</div></div>';

    pagination.innerHTML = html;
}

function changeWhitelistPage(page) {
    whitelistPage = page;
    loadWhitelist();
}

function openWhitelistModal(item = null) {
    const modal = document.getElementById('whitelistModal');
    const title = document.getElementById('whitelistModalTitle');
    const form = document.getElementById('whitelistForm');

    if (item) {
        title.textContent = '编辑白名单';
        document.getElementById('whitelistId').value = item.id;
        document.getElementById('wlType').value = item.type;
        document.getElementById('wlValue').value = item.target;
        document.getElementById('wlReason').value = item.reason;
    } else {
        title.textContent = '添加白名单';
        form.reset();
        document.getElementById('whitelistId').value = '';
    }

    const bsModal = new bootstrap.Modal(modal);
    bsModal.show();
}

async function handleWhitelistSubmit(e) {
    e.preventDefault();

    const id = document.getElementById('whitelistId').value;

    const data = {
        type: document.getElementById('wlType').value,
        target: document.getElementById('wlValue').value,
        reason: document.getElementById('wlReason').value
    };

    try {
        if (id) {
            await auth.request(`/admin/whitelist/${id}`, { method: 'PUT', body: JSON.stringify(data) });
            showToast('白名单更新成功', 'success');
        } else {
            await auth.request('/admin/whitelist', { method: 'POST', body: JSON.stringify(data) });
            showToast('白名单添加成功', 'success');
        }

        bootstrap.Modal.getInstance(document.getElementById('whitelistModal'))?.hide();
        loadWhitelist();
    } catch (error) {
        showToast('保存失败', 'danger');
    }
}

async function handleWhitelistImport() {
    const fileInput = document.getElementById('whitelistImportFile');
    const importType = document.getElementById('whitelistImportType').value;
    const importReason = document.getElementById('whitelistImportReason').value;

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
            await auth.request('/admin/whitelist/import', { method: 'POST', body: JSON.stringify(data) });
            showToast(`成功导入 ${targets.length} 条记录`, 'success');
            bootstrap.Modal.getInstance(document.getElementById('whitelistImportModal'))?.hide();
            loadWhitelist();
        } catch (error) {
            showToast('导入失败', 'danger');
        }
    };

    reader.readAsText(file);
}

async function deleteWhitelistItem(id) {
    if (!confirm('确定要删除这条白名单记录吗？')) return;

    try {
        await auth.request(`/admin/whitelist/${id}`, { method: 'DELETE' });
        showToast('删除成功', 'success');
        loadWhitelist();
    } catch (error) {
        showToast('删除失败', 'danger');
    }
}

function viewWhitelistDetail(id) {
    const item = currentWhitelist.find(i => i.id === id);
    if (!item) return;

    const modal = document.getElementById('whitelistDetailModal');
    const content = document.getElementById('whitelistDetailContent');

    content.innerHTML = `
        <table class="table table-borderless">
            <tr><td class="text-muted" style="width: 100px;">ID</td><td>${item.id}</td></tr>
            <tr><td class="text-muted">目标</td><td><code>${escapeHtml(item.target)}</code></td></tr>
            <tr><td class="text-muted">类型</td><td><span class="badge ${getTypeBadgeClass(item.type)}">${getTypeText(item.type)}</span></td></tr>
            <tr><td class="text-muted">来源</td><td><span class="badge bg-secondary">${getSourceText(item.source)}</span></td></tr>
            <tr><td class="text-muted">原因</td><td>${escapeHtml(item.reason)}</td></tr>
            <tr><td class="text-muted">过期时间</td><td>${item.expiration === 'permanent' ? '永久' : item.expiration}</td></tr>
            <tr><td class="text-muted">添加人</td><td>${item.createdBy}</td></tr>
            <tr><td class="text-muted">添加时间</td><td>${item.createdAt}</td></tr>
        </table>
    `;

    const bsModal = new bootstrap.Modal(modal);
    bsModal.show();
}
