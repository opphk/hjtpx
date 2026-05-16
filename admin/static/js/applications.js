
let currentPage = 1;
const pageSize = 10;
let totalApps = 0;
let appList = [];
let selectedApps = new Set();

document.addEventListener('DOMContentLoaded', function() {
    if (!Auth.requireAuth()) {
        return;
    }

    const user = Auth.getCurrentUser();
    if (user && user.username) {
        Auth.updateUserDisplay(user.username);
    }

    loadApplications();
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
                loadApplications();
            }, 500);
        });
    }

    const statusFilter = document.getElementById('statusFilter');
    if (statusFilter) {
        statusFilter.addEventListener('change', function() {
            currentPage = 1;
            loadApplications();
        });
    }

    const platformFilter = document.getElementById('platformFilter');
    if (platformFilter) {
        platformFilter.addEventListener('change', function() {
            currentPage = 1;
            loadApplications();
        });
    }

    const resetBtn = document.getElementById('resetFilterBtn');
    if (resetBtn) {
        resetBtn.addEventListener('click', function() {
            document.getElementById('searchInput').value = '';
            document.getElementById('statusFilter').value = '';
            document.getElementById('platformFilter').value = '';
            currentPage = 1;
            loadApplications();
        });
    }

    const addAppBtn = document.getElementById('addAppBtn');
    if (addAppBtn) {
        addAppBtn.addEventListener('click', function() {
            openAppModal();
        });
    }

    const saveAppBtn = document.getElementById('saveAppBtn');
    if (saveAppBtn) {
        saveAppBtn.addEventListener('click', function() {
            saveApplication();
        });
    }

    const selectAllCheckbox = document.getElementById('selectAll');
    if (selectAllCheckbox) {
        selectAllCheckbox.addEventListener('change', function() {
            const checkboxes = document.querySelectorAll('.app-checkbox');
            checkboxes.forEach(checkbox => {
                checkbox.checked = this.checked;
                const appId = checkbox.getAttribute('data-id');
                if (this.checked) {
                    selectedApps.add(appId);
                } else {
                    selectedApps.delete(appId);
                }
            });
            updateBulkActions();
        });
    }
}

async function loadApplications() {
    try {
        const searchQuery = document.getElementById('searchInput')?.value || '';
        const statusFilter = document.getElementById('statusFilter')?.value || '';
        const platformFilter = document.getElementById('platformFilter')?.value || '';

        const params = new URLSearchParams({
            page: currentPage,
            pageSize: pageSize,
            search: searchQuery,
            status: statusFilter,
            platform: platformFilter
        });

        const response = await fetch(`/admin/api/applications?${params}`);
        if (!response.ok) throw new Error('获取应用列表失败');

        const data = await response.json();
        appList = data.apps || [];
        totalApps = data.total || 0;

        renderApplicationTable(appList);
        renderPagination();
        updateTotalCount();
    } catch (error) {
        console.error('加载应用列表失败:', error);
        loadMockApplications();
    }
}

function loadMockApplications() {
    const mockApps = [
        {
            id: 1,
            name: '示例应用1',
            appId: 'app_001',
            platform: 'ios',
            status: 'active',
            createdAt: '2024-01-15 10:30:00',
            requestCount: 12345
        },
        {
            id: 2,
            name: '测试应用',
            appId: 'app_002',
            platform: 'android',
            status: 'active',
            createdAt: '2024-01-14 15:20:00',
            requestCount: 8765
        },
        {
            id: 3,
            name: 'Web应用',
            appId: 'app_003',
            platform: 'web',
            status: 'pending',
            createdAt: '2024-01-13 09:15:00',
            requestCount: 2345
        },
        {
            id: 4,
            name: '企业应用',
            appId: 'app_004',
            platform: 'ios',
            status: 'inactive',
            createdAt: '2024-01-12 14:45:00',
            requestCount: 5678
        },
        {
            id: 5,
            name: '移动应用',
            appId: 'app_005',
            platform: 'android',
            status: 'active',
            createdAt: '2024-01-11 11:30:00',
            requestCount: 9876
        }
    ];

    appList = mockApps;
    totalApps = mockApps.length;

    renderApplicationTable(appList);
    renderPagination();
    updateTotalCount();
}

function renderApplicationTable(apps) {
    const tbody = document.getElementById('appTableBody');
    if (!tbody) return;

    tbody.innerHTML = '';

    if (apps.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="8" class="text-center text-muted py-5">
                    <i class="fas fa-inbox fa-3x mb-3"></i>
                    <p class="mb-0">暂无应用数据</p>
                </td>
            </tr>
        `;
        return;
    }

    apps.forEach(app => {
        const tr = document.createElement('tr');
        tr.setAttribute('data-id', app.id);

        const platformIcon = getPlatformIcon(app.platform);
        const platformClass = getPlatformClass(app.platform);
        const statusBadge = getStatusBadge(app.status);

        tr.innerHTML = `
            <td><input type="checkbox" class="app-checkbox" data-id="${app.id}"></td>
            <td>
                <div class="d-flex align-items-center gap-2">
                    <div class="app-icon ${platformClass}">
                        ${platformIcon}
                    </div>
                    <div>
                        <div class="fw-medium">${app.name}</div>
                    </div>
                </div>
            </td>
            <td><code class="small">${app.appId}</code></td>
            <td>${app.platform.toUpperCase()}</td>
            <td>${statusBadge}</td>
            <td><small class="text-muted">${app.createdAt}</small></td>
            <td>${app.requestCount.toLocaleString()}</td>
            <td>
                <div class="btn-group btn-group-sm">
                    <button class="btn btn-outline-primary" onclick="viewAppDetail(${app.id})" title="查看详情">
                        <i class="fas fa-eye"></i>
                    </button>
                    <button class="btn btn-outline-secondary" onclick="editApp(${app.id})" title="编辑">
                        <i class="fas fa-edit"></i>
                    </button>
                    <button class="btn btn-outline-danger" onclick="deleteApp(${app.id})" title="删除">
                        <i class="fas fa-trash"></i>
                    </button>
                </div>
            </td>
        `;

        tbody.appendChild(tr);

        const checkbox = tr.querySelector('.app-checkbox');
        checkbox.addEventListener('change', function() {
            if (this.checked) {
                selectedApps.add(app.id.toString());
            } else {
                selectedApps.delete(app.id.toString());
            }
            updateBulkActions();
        });

        tr.addEventListener('click', function(e) {
            if (e.target.type !== 'checkbox') {
                viewAppDetail(app.id);
            }
        });
    });
}

function getPlatformIcon(platform) {
    const icons = {
        ios: '<i class="fab fa-apple"></i>',
        android: '<i class="fab fa-android"></i>',
        web: '<i class="fas fa-globe"></i>'
    };
    return icons[platform] || '<i class="fas fa-mobile-alt"></i>';
}

function getPlatformClass(platform) {
    const classes = {
        ios: 'bg-dark',
        android: 'bg-success',
        web: 'bg-primary'
    };
    return classes[platform] || 'bg-secondary';
}

function getStatusBadge(status) {
    const badges = {
        active: '<span class="badge bg-success">正常</span>',
        inactive: '<span class="badge bg-secondary">停用</span>',
        pending: '<span class="badge bg-warning text-dark">待审核</span>'
    };
    return badges[status] || '<span class="badge bg-secondary">未知</span>';
}

function renderPagination() {
    const pagination = document.getElementById('pagination');
    if (!pagination) return;

    pagination.innerHTML = '';

    const totalPages = Math.ceil(totalApps / pageSize);
    if (totalPages <= 1) return;

    const prevLi = document.createElement('li');
    prevLi.className = `page-item ${currentPage === 1 ? 'disabled' : ''}`;
    prevLi.innerHTML = `<a class="page-link" href="#" aria-label="上一页">&laquo;</a>`;
    if (currentPage > 1) {
        prevLi.addEventListener('click', function(e) {
            e.preventDefault();
            currentPage--;
            loadApplications();
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
            loadApplications();
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
            loadApplications();
        });
    }
    pagination.appendChild(nextLi);
}

function updateTotalCount() {
    const totalCountEl = document.getElementById('totalCount');
    if (totalCountEl) {
        totalCountEl.textContent = totalApps;
    }
}

function updateBulkActions() {
    const selectAllCheckbox = document.getElementById('selectAll');
    if (selectAllCheckbox) {
        const checkboxes = document.querySelectorAll('.app-checkbox');
        selectAllCheckbox.checked = checkboxes.length > 0 && selectedApps.size === checkboxes.length;
    }
}

function openAppModal(app = null) {
    const modal = new bootstrap.Modal(document.getElementById('appModal'));
    const modalLabel = document.getElementById('appModalLabel');
    const form = document.getElementById('appForm');

    form.reset();
    document.getElementById('appId').value = '';

    if (app) {
        modalLabel.textContent = '编辑应用';
        document.getElementById('appId').value = app.id;
        document.getElementById('appName').value = app.name;
        document.getElementById('appDescription').value = app.description || '';
        document.getElementById('appPlatform').value = app.platform;
        document.getElementById('appBundleId').value = app.bundleId || '';
        document.getElementById('appStatus').value = app.status;
    } else {
        modalLabel.textContent = '新建应用';
    }

    modal.show();
}

async function saveApplication() {
    const form = document.getElementById('appForm');
    if (!form.checkValidity()) {
        form.reportValidity();
        return;
    }

    const appId = document.getElementById('appId').value;
    const appData = {
        name: document.getElementById('appName').value,
        description: document.getElementById('appDescription').value,
        platform: document.getElementById('appPlatform').value,
        bundleId: document.getElementById('appBundleId').value,
        status: document.getElementById('appStatus').value
    };

    try {
        const url = appId ? `/admin/api/applications/${appId}` : '/admin/api/applications';
        const method = appId ? 'PUT' : 'POST';

        const response = await fetch(url, {
            method: method,
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(appData)
        });

        if (!response.ok) throw new Error('保存应用失败');

        const result = await response.json();

        bootstrap.Modal.getInstance(document.getElementById('appModal')).hide();

        Auth.showToast(appId ? '应用更新成功' : '应用创建成功', 'success');
        loadApplications();
    } catch (error) {
        console.error('保存应用失败:', error);
        Auth.showToast('保存失败，请重试', 'error');
    }
}

async function viewAppDetail(appId) {
    try {
        const response = await fetch(`/admin/api/applications/${appId}`);
        if (!response.ok) throw new Error('获取应用详情失败');

        const app = await response.json();
        showAppDetailModal(app);
    } catch (error) {
        console.error('获取应用详情失败:', error);
        const app = appList.find(a => a.id === appId);
        if (app) {
            showAppDetailModal(app);
        }
    }
}

function showAppDetailModal(app) {
    const modal = new bootstrap.Modal(document.getElementById('appDetailModal'));
    const content = document.getElementById('appDetailContent');

    const statusClass = app.status === 'active' ? 'success' : app.status === 'inactive' ? 'secondary' : 'warning';
    const statusText = app.status === 'active' ? '正常' : app.status === 'inactive' ? '停用' : '待审核';

    content.innerHTML = `
        <div class="row">
            <div class="col-md-6">
                <div class="mb-3">
                    <label class="text-muted small">应用名称</label>
                    <div class="fw-medium">${app.name}</div>
                </div>
                <div class="mb-3">
                    <label class="text-muted small">AppID</label>
                    <div><code>${app.appId}</code></div>
                </div>
                <div class="mb-3">
                    <label class="text-muted small">平台</label>
                    <div>${app.platform.toUpperCase()}</div>
                </div>
                <div class="mb-3">
                    <label class="text-muted small">状态</label>
                    <div><span class="badge bg-${statusClass}">${statusText}</span></div>
                </div>
            </div>
            <div class="col-md-6">
                <div class="mb-3">
                    <label class="text-muted small">创建时间</label>
                    <div>${app.createdAt}</div>
                </div>
                <div class="mb-3">
                    <label class="text-muted small">请求量</label>
                    <div class="fw-medium">${(app.requestCount || 0).toLocaleString()}</div>
                </div>
                ${app.bundleId ? `
                <div class="mb-3">
                    <label class="text-muted small">Bundle ID</label>
                    <div><code>${app.bundleId}</code></div>
                </div>
                ` : ''}
                ${app.description ? `
                <div class="mb-3">
                    <label class="text-muted small">应用描述</label>
                    <div>${app.description}</div>
                </div>
                ` : ''}
            </div>
        </div>
        <hr>
        <div class="row">
            <div class="col-12">
                <h6>统计信息</h6>
                <div class="row mt-3">
                    <div class="col-md-4">
                        <div class="text-center">
                            <div class="h4 mb-1">${(app.requestCount || 0).toLocaleString()}</div>
                            <small class="text-muted">总请求量</small>
                        </div>
                    </div>
                    <div class="col-md-4">
                        <div class="text-center">
                            <div class="h4 mb-1">${((app.successRate || 98.5)).toFixed(1)}%</div>
                            <small class="text-muted">成功率</small>
                        </div>
                    </div>
                    <div class="col-md-4">
                        <div class="text-center">
                            <div class="h4 mb-1">${(app.avgLatency || 45)}ms</div>
                            <small class="text-muted">平均延迟</small>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    `;

    modal.show();
}

function editApp(appId) {
    const app = appList.find(a => a.id === appId);
    if (app) {
        openAppModal(app);
    }
}

function deleteApp(appId) {
    Auth.showConfirmDialog(
        '删除应用',
        '确定要删除该应用吗？此操作不可恢复。',
        async function() {
            try {
                const response = await fetch(`/admin/api/applications/${appId}`, {
                    method: 'DELETE'
                });

                if (!response.ok) throw new Error('删除应用失败');

                Auth.showToast('应用已删除', 'success');
                loadApplications();
            } catch (error) {
                console.error('删除应用失败:', error);
                Auth.showToast('删除失败，请重试', 'error');
            }
        }
    );
}

window.Applications = {
    loadApplications: loadApplications,
    viewAppDetail: viewAppDetail,
    editApp: editApp,
    deleteApp: deleteApp
};
