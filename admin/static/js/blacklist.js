
let currentPage = 1;
const pageSize = 10;
let totalItems = 0;
let blacklist = [];
let selectedItems = new Set();

document.addEventListener('DOMContentLoaded', function() {
    if (!Auth.requireAuth()) {
        return;
    }

    const user = Auth.getCurrentUser();
    if (user && user.username) {
        Auth.updateUserDisplay(user.username);
    }

    loadBlacklist();
    initEventListeners();
});

function initEventListeners() {
    const searchInput = document.getElementById('searchInput');
    if (searchInput) {
        let searchTimeout;
        searchInput.addEventListener('input', function() {
            clearTimeout(searchTimeout);
            searchTimeout = setTimeout(() => {
                currentPage = 1;
                loadBlacklist();
            }, 500);
        });
    }

    const typeFilter = document.getElementById('typeFilter');
    if (typeFilter) {
        typeFilter.addEventListener('change', function() {
            currentPage = 1;
            loadBlacklist();
        });
    }

    const statusFilter = document.getElementById('statusFilter');
    if (statusFilter) {
        statusFilter.addEventListener('change', function() {
            currentPage = 1;
            loadBlacklist();
        });
    }

    const resetBtn = document.getElementById('resetFilterBtn');
    if (resetBtn) {
        resetBtn.addEventListener('click', function() {
            document.getElementById('searchInput').value = '';
            document.getElementById('typeFilter').value = '';
            document.getElementById('statusFilter').value = '';
            currentPage = 1;
            loadBlacklist();
        });
    }

    const addBtn = document.getElementById('addBlacklistBtn');
    if (addBtn) {
        addBtn.addEventListener('click', function() {
            openBlacklistModal();
        });
    }

    const saveBtn = document.getElementById('saveBlacklistBtn');
    if (saveBtn) {
        saveBtn.addEventListener('click', function() {
            saveBlacklist();
        });
    }

    const selectAllCheckbox = document.getElementById('selectAll');
    if (selectAllCheckbox) {
        selectAllCheckbox.addEventListener('change', function() {
            const checkboxes = document.querySelectorAll('.blacklist-checkbox');
            checkboxes.forEach(checkbox => {
                checkbox.checked = this.checked;
                const id = checkbox.getAttribute('data-id');
                if (this.checked) {
                    selectedItems.add(id);
                } else {
                    selectedItems.delete(id);
                }
            });
        });
    }
}

async function loadBlacklist() {
    try {
        const searchQuery = document.getElementById('searchInput')?.value || '';
        const typeFilter = document.getElementById('typeFilter')?.value || '';
        const statusFilter = document.getElementById('statusFilter')?.value || '';

        const params = new URLSearchParams({
            page: currentPage,
            pageSize: pageSize,
            search: searchQuery,
            type: typeFilter,
            status: statusFilter
        });

        const response = await fetch(`/admin/api/blacklist?${params}`);
        if (!response.ok) throw new Error('获取黑名单失败');

        const data = await response.json();
        blacklist = data.items || [];
        totalItems = data.total || 0;

        renderBlacklistTable(blacklist);
        renderPagination();
        updateTotalCount();
    } catch (error) {
        console.error('加载黑名单失败:', error);
        loadMockBlacklist();
    }
}

function loadMockBlacklist() {
    const mockData = [
        {
            id: 1,
            type: 'ip',
            value: '192.168.1.100',
            reason: '异常请求行为',
            createdAt: '2024-01-15 10:30:00',
            expireAt: '2024-02-15 10:30:00',
            status: 'active',
            operator: 'admin'
        },
        {
            id: 2,
            type: 'device',
            value: 'device_abc123',
            reason: '设备指纹异常',
            createdAt: '2024-01-14 15:20:00',
            expireAt: null,
            status: 'active',
            operator: 'admin'
        },
        {
            id: 3,
            type: 'user',
            value: 'user_456',
            reason: '违规操作',
            createdAt: '2024-01-13 09:15:00',
            expireAt: '2024-03-13 09:15:00',
            status: 'active',
            operator: 'admin'
        },
        {
            id: 4,
            type: 'app',
            value: 'app_test',
            reason: '应用违规',
            createdAt: '2024-01-12 14:45:00',
            expireAt: '2024-01-12 14:45:00',
            status: 'expired',
            operator: 'admin'
        },
        {
            id: 5,
            type: 'ip',
            value: '10.0.0.50',
            reason: 'DDoS攻击',
            createdAt: '2024-01-11 11:30:00',
            expireAt: null,
            status: 'active',
            operator: 'admin'
        }
    ];

    blacklist = mockData;
    totalItems = mockData.length;

    renderBlacklistTable(blacklist);
    renderPagination();
    updateTotalCount();
}

function renderBlacklistTable(items) {
    const tbody = document.getElementById('blacklistTableBody');
    if (!tbody) return;

    tbody.innerHTML = '';

    if (items.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="9" class="text-center text-muted py-5">
                    <i class="fas fa-ban fa-3x mb-3"></i>
                    <p class="mb-0">暂无黑名单数据</p>
                </td>
            </tr>
        `;
        return;
    }

    items.forEach(item => {
        const tr = document.createElement('tr');
        tr.setAttribute('data-id', item.id);

        const typeIcon = getTypeIcon(item.type);
        const typeClass = getTypeClass(item.type);
        const statusBadge = getStatusBadge(item.status);

        tr.innerHTML = `
            <td><input type="checkbox" class="blacklist-checkbox" data-id="${item.id}"></td>
            <td>
                <div class="d-flex align-items-center gap-2">
                    <div class="type-icon ${typeClass}">${typeIcon}</div>
                    <span>${getTypeName(item.type)}</span>
                </div>
            </td>
            <td><code>${item.value}</code></td>
            <td><small>${item.reason}</small></td>
            <td><small class="text-muted">${item.createdAt}</small></td>
            <td><small class="text-muted">${item.expireAt || '永久'}</small></td>
            <td>${statusBadge}</td>
            <td><small>${item.operator}</small></td>
            <td>
                <div class="btn-group btn-group-sm">
                    <button class="btn btn-outline-primary" onclick="viewBlacklistDetail(${item.id})" title="查看详情">
                        <i class="fas fa-eye"></i>
                    </button>
                    <button class="btn btn-outline-danger" onclick="removeBlacklist(${item.id})" title="删除">
                        <i class="fas fa-trash"></i>
                    </button>
                </div>
            </td>
        `;

        tbody.appendChild(tr);

        const checkbox = tr.querySelector('.blacklist-checkbox');
        checkbox.addEventListener('change', function() {
            if (this.checked) {
                selectedItems.add(item.id.toString());
            } else {
                selectedItems.delete(item.id.toString());
            }
        });
    });
}

function getTypeIcon(type) {
    const icons = {
        ip: '<i class="fas fa-network-wired"></i>',
        device: '<i class="fas fa-mobile-alt"></i>',
        user: '<i class="fas fa-user"></i>',
        app: '<i class="fas fa-rocket"></i>'
    };
    return icons[type] || '<i class="fas fa-question"></i>';
}

function getTypeClass(type) {
    return `type-${type}`;
}

function getTypeName(type) {
    const names = {
        ip: 'IP地址',
        device: '设备ID',
        user: '用户ID',
        app: '应用ID'
    };
    return names[type] || type;
}

function getStatusBadge(status) {
    const badges = {
        active: '<span class="badge bg-danger">生效中</span>',
        expired: '<span class="badge bg-secondary">已过期</span>'
    };
    return badges[status] || '<span class="badge bg-secondary">未知</span>';
}

function renderPagination() {
    const pagination = document.getElementById('pagination');
    if (!pagination) return;

    pagination.innerHTML = '';

    const totalPages = Math.ceil(totalItems / pageSize);
    if (totalPages <= 1) return;

    const prevLi = document.createElement('li');
    prevLi.className = `page-item ${currentPage === 1 ? 'disabled' : ''}`;
    prevLi.innerHTML = `<a class="page-link" href="#" aria-label="上一页">&laquo;</a>`;
    if (currentPage > 1) {
        prevLi.addEventListener('click', function(e) {
            e.preventDefault();
            currentPage--;
            loadBlacklist();
        });
    }
    pagination.appendChild(prevLi);

    const maxVisible = 5;
    let startPage = Math.max(1, currentPage - Math.floor(maxVisible / 2));
    let endPage = Math.min(totalPages, startPage + maxVisible - 1);

    if (endPage - startPage + 1 < maxVisible) {
        startPage = Math.max(1, endPage - maxVisible + 1);
    }

    for (let i = startPage; i <= endPage; i++) {
        const li = document.createElement('li');
        li.className = `page-item ${i === currentPage ? 'active' : ''}`;
        li.innerHTML = `<a class="page-link" href="#">${i}</a>`;
        li.addEventListener('click', function(e) {
            e.preventDefault();
            currentPage = i;
            loadBlacklist();
        });
        pagination.appendChild(li);
    }

    const nextLi = document.createElement('li');
    nextLi.className = `page-item ${currentPage === totalPages ? 'disabled' : ''}`;
    nextLi.innerHTML = `<a class="page-link" href="#" aria-label="下一页">&raquo;</a>`;
    if (currentPage < totalPages) {
        nextLi.addEventListener('click', function(e) {
            e.preventDefault();
            currentPage++;
            loadBlacklist();
        });
    }
    pagination.appendChild(nextLi);
}

function updateTotalCount() {
    const totalCountEl = document.getElementById('totalCount');
    if (totalCountEl) {
        totalCountEl.textContent = totalItems;
    }
}

function openBlacklistModal() {
    const modal = new bootstrap.Modal(document.getElementById('blacklistModal'));
    const form = document.getElementById('blacklistForm');

    form.reset();
    document.getElementById('blacklistType').value = '';
    document.getElementById('blacklistValue').value = '';
    document.getElementById('blacklistReason').value = '';
    document.getElementById('blacklistExpire').value = '';

    modal.show();
}

async function saveBlacklist() {
    const form = document.getElementById('blacklistForm');
    if (!form.checkValidity()) {
        form.reportValidity();
        return;
    }

    const blacklistData = {
        type: document.getElementById('blacklistType').value,
        value: document.getElementById('blacklistValue').value,
        reason: document.getElementById('blacklistReason').value,
        expireAt: document.getElementById('blacklistExpire').value || null
    };

    try {
        const response = await fetch('/admin/api/blacklist', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(blacklistData)
        });

        if (!response.ok) throw new Error('添加黑名单失败');

        bootstrap.Modal.getInstance(document.getElementById('blacklistModal')).hide();

        Auth.showToast('黑名单添加成功', 'success');
        loadBlacklist();
    } catch (error) {
        console.error('添加黑名单失败:', error);
        Auth.showToast('添加失败，请重试', 'error');
    }
}

async function viewBlacklistDetail(id) {
    try {
        const response = await fetch(`/admin/api/blacklist/${id}`);
        if (!response.ok) throw new Error('获取黑名单详情失败');

        const item = await response.json();
        showDetailModal(item);
    } catch (error) {
        console.error('获取黑名单详情失败:', error);
        const item = blacklist.find(b => b.id === id);
        if (item) {
            showDetailModal(item);
        }
    }
}

function showDetailModal(item) {
    const modal = new bootstrap.Modal(document.getElementById('blacklistDetailModal'));
    const content = document.getElementById('blacklistDetailContent');

    const statusClass = item.status === 'active' ? 'danger' : 'secondary';
    const statusText = item.status === 'active' ? '生效中' : '已过期';

    content.innerHTML = `
        <div class="mb-3">
            <label class="text-muted small">类型</label>
            <div class="d-flex align-items-center gap-2">
                <div class="type-icon ${getTypeClass(item.type)}">${getTypeIcon(item.type)}</div>
                <span>${getTypeName(item.type)}</span>
            </div>
        </div>
        <div class="mb-3">
            <label class="text-muted small">目标值</label>
            <div><code>${item.value}</code></div>
        </div>
        <div class="mb-3">
            <label class="text-muted small">原因</label>
            <div>${item.reason}</div>
        </div>
        <div class="mb-3">
            <label class="text-muted small">添加时间</label>
            <div>${item.createdAt}</div>
        </div>
        <div class="mb-3">
            <label class="text-muted small">过期时间</label>
            <div>${item.expireAt || '永久'}</div>
        </div>
        <div class="mb-3">
            <label class="text-muted small">状态</label>
            <div><span class="badge bg-${statusClass}">${statusText}</span></div>
        </div>
        <div class="mb-3">
            <label class="text-muted small">操作人</label>
            <div>${item.operator}</div>
        </div>
    `;

    modal.show();
}

function removeBlacklist(id) {
    Auth.showConfirmDialog(
        '删除黑名单',
        '确定要删除该黑名单记录吗？',
        async function() {
            try {
                const response = await fetch(`/admin/api/blacklist/${id}`, {
                    method: 'DELETE'
                });

                if (!response.ok) throw new Error('删除黑名单失败');

                Auth.showToast('黑名单已删除', 'success');
                loadBlacklist();
            } catch (error) {
                console.error('删除黑名单失败:', error);
                Auth.showToast('删除失败，请重试', 'error');
            }
        }
    );
}

window.Blacklist = {
    loadBlacklist: loadBlacklist,
    viewBlacklistDetail: viewBlacklistDetail,
    removeBlacklist: removeBlacklist
};
