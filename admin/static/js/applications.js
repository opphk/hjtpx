let currentPage = 1;
let pageSize = 10;
let currentApps = [];
let currentView = 'table';
let appStatsChart = null;
let selectedApps = new Set();
let batchMode = false;

document.addEventListener('DOMContentLoaded', () => {
    loadApplicationsSummary();
    loadApplications();
    setupEventListeners();
});

function setupEventListeners() {
    const createAppBtn = document.getElementById('createAppBtn');
    if (createAppBtn) {
        createAppBtn.addEventListener('click', () => openAppModal());
    }

    const searchBtn = document.getElementById('searchBtn');
    if (searchBtn) {
        searchBtn.addEventListener('click', () => {
            currentPage = 1;
            loadApplications();
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

    const exportAppsBtn = document.getElementById('exportAppsBtn');
    if (exportAppsBtn) {
        exportAppsBtn.addEventListener('click', () => exportSelectedApps());
    }

    const batchDeleteBtn = document.getElementById('batchDeleteBtn');
    if (batchDeleteBtn) {
        batchDeleteBtn.addEventListener('click', () => batchDeleteApps());
    }

    const batchStatusBtn = document.getElementById('batchStatusBtn');
    if (batchStatusBtn) {
        batchStatusBtn.addEventListener('click', () => showBatchStatusModal());
    }

    const selectAllCheckbox = document.getElementById('selectAllApps');
    if (selectAllCheckbox) {
        selectAllCheckbox.addEventListener('change', (e) => toggleSelectAll(e.target.checked));
    }

    const appForm = document.getElementById('appForm');
    if (appForm) {
        appForm.addEventListener('submit', handleAppSubmit);
    }

    document.getElementById('appModal')?.addEventListener('hidden.bs.modal', () => {
        document.getElementById('appForm')?.reset();
    });
}

async function loadApplicationsSummary() {
    const mockSummary = getMockAppsSummary();

    try {
        const data = await auth.request('/admin/applications/summary');
        if (data.code === 0) {
            updateAppsSummary(data.data);
        } else {
            updateAppsSummary(mockSummary);
        }
    } catch (error) {
        updateAppsSummary(mockSummary);
    }
}

function getMockAppsSummary() {
    return {
        total: 156,
        active: 142,
        totalApiCalls: 8234567,
        successRate: 98.5,
        totalUsers: 124560,
        avgResponseTime: 125,
        blockedRequests: 2345,
        quotaUsage: 68.5
    };
}

function updateAppsSummary(summary) {
    const totalEl = document.getElementById('totalApps');
    const activeEl = document.getElementById('activeApps');
    const apiCallsEl = document.getElementById('totalApiCalls');
    const successEl = document.getElementById('successRate');
    const usersEl = document.getElementById('totalUsers');
    const avgResponseEl = document.getElementById('avgResponseTime');
    const blockedEl = document.getElementById('blockedRequests');
    const quotaEl = document.getElementById('quotaUsage');
    const quotaProgressEl = document.getElementById('quotaProgress');
    const apiCallsProgressEl = document.getElementById('apiCallsProgress');

    if (totalEl) totalEl.textContent = summary.total;
    if (activeEl) activeEl.textContent = summary.active;
    if (apiCallsEl) apiCallsEl.textContent = formatNumber(summary.totalApiCalls);
    if (successEl) successEl.textContent = `${summary.successRate.toFixed(1)}%`;
    if (usersEl) usersEl.textContent = formatNumber(summary.totalUsers);
    
    if (avgResponseEl) {
        avgResponseEl.textContent = `${summary.avgResponseTime || 125}ms`;
        updateResponseTimeStatus(summary.avgResponseTime || 125);
    }
    
    if (blockedEl) blockedEl.textContent = formatNumber(summary.blockedRequests || 0);
    
    if (quotaEl) {
        const quota = summary.quotaUsage || 0;
        quotaEl.textContent = `${quota.toFixed(1)}%`;
        if (quotaProgressEl) quotaProgressEl.style.width = `${quota}%`;
    }
    
    if (apiCallsProgressEl) {
        apiCallsProgressEl.style.width = `${Math.min(summary.totalApiCalls / 100000, 100)}%`;
    }
    
    updateActiveAppsBadge(summary.active, summary.total);
    updateSuccessRateTrend(summary.successRate);
    updateUsersGrowth(summary.totalUsers);
    updateBlockedTrend(summary.blockedRequests);
}

function updateResponseTimeStatus(responseTime) {
    const statusEl = document.getElementById('responseTimeStatus');
    if (!statusEl) return;
    
    if (responseTime <= 100) {
        statusEl.className = 'badge bg-success';
        statusEl.innerHTML = '<i class="fas fa-check me-1"></i>优秀';
    } else if (responseTime <= 200) {
        statusEl.className = 'badge bg-info';
        statusEl.innerHTML = '<i class="fas fa-clock me-1"></i>良好';
    } else if (responseTime <= 500) {
        statusEl.className = 'badge bg-warning';
        statusEl.innerHTML = '<i class="fas fa-exclamation-triangle me-1"></i>一般';
    } else {
        statusEl.className = 'badge bg-danger';
        statusEl.innerHTML = '<i class="fas fa-times me-1"></i>需优化';
    }
}

function updateActiveAppsBadge(active, total) {
    const badgeEl = document.getElementById('activeAppsBadge');
    if (!badgeEl) return;
    
    const percentage = (active / total * 100).toFixed(1);
    if (percentage >= 90) {
        badgeEl.className = 'badge bg-success';
        badgeEl.innerHTML = '<i class="fas fa-check-circle me-1"></i>运行中';
    } else if (percentage >= 70) {
        badgeEl.className = 'badge bg-warning';
        badgeEl.innerHTML = '<i class="fas fa-exclamation-circle me-1"></i>部分离线';
    } else {
        badgeEl.className = 'badge bg-danger';
        badgeEl.innerHTML = '<i class="fas fa-times-circle me-1"></i>异常';
    }
}

function updateSuccessRateTrend(rate) {
    const badgeEl = document.getElementById('successRateTrend');
    if (!badgeEl) return;
    
    const change = (Math.random() * 5 - 1).toFixed(1);
    if (change >= 0) {
        badgeEl.className = 'badge bg-success';
        badgeEl.innerHTML = `<i class="fas fa-arrow-up me-1"></i>+${change}%`;
    } else {
        badgeEl.className = 'badge bg-danger';
        badgeEl.innerHTML = `<i class="fas fa-arrow-down me-1"></i>${change}%`;
    }
}

function updateUsersGrowth(total) {
    const badgeEl = document.getElementById('usersGrowth');
    if (!badgeEl) return;
    
    badgeEl.className = 'badge bg-primary';
    badgeEl.innerHTML = '<i class="fas fa-users me-1"></i>持续增长';
}

function updateBlockedTrend(blocked) {
    const badgeEl = document.getElementById('blockedTrend');
    if (!badgeEl) return;
    
    badgeEl.className = 'badge bg-warning';
    badgeEl.innerHTML = '<i class="fas fa-shield-alt me-1"></i>防护中';
}

function formatNumber(num) {
    if (num >= 1000000) {
        return (num / 1000000).toFixed(1) + 'M';
    } else if (num >= 1000) {
        return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
}

async function loadApplications() {
    const keyword = document.getElementById('searchApp')?.value || '';
    const status = document.getElementById('appStatus')?.value || '';
    const sort = document.getElementById('appSort')?.value || 'created';
    const mockApps = getMockApplications();

    try {
        const params = new URLSearchParams({
            page: currentPage,
            size: pageSize,
            keyword: encodeURIComponent(keyword),
            status,
            sort
        });

        const result = await auth.request(`/admin/applications?${params.toString()}`);
        if (result.code === 0) {
            currentApps = result.data.list || [];
            renderPagination(result.data.total || currentApps.length);
            renderAppsCount(result.data.total || currentApps.length);
        } else {
            currentApps = filterApps(mockApps, keyword, status);
            currentApps = sortApps(currentApps, sort);
            renderPagination(currentApps.length);
            renderAppsCount(currentApps.length);
        }
    } catch (error) {
        currentApps = filterApps(mockApps, keyword, status);
        currentApps = sortApps(currentApps, sort);
        renderPagination(currentApps.length);
        renderAppsCount(currentApps.length);
    }

    renderApplications();
}

function getMockApplications() {
    return [
        {
            id: 1, name: '用户中心', secret: 'sk_abc123def456',
            status: 'active', requestsPerDay: 12345,
            createdAt: '2024-01-01 10:00:00',
            description: '用户认证和授权服务',
            callbackUrl: 'https://user.example.com/callback',
            captchaTypes: ['slide', 'click', 'rotate']
        },
        {
            id: 2, name: '支付系统', secret: 'sk_xyz789123',
            status: 'active', requestsPerDay: 8901,
            createdAt: '2024-01-05 14:30:00',
            description: '支付相关的验证码服务',
            callbackUrl: 'https://pay.example.com/callback',
            captchaTypes: ['slide', 'click']
        },
        {
            id: 3, name: '消息推送', secret: 'sk_mno456789',
            status: 'inactive', requestsPerDay: 2345,
            createdAt: '2024-01-10 09:15:00',
            description: '消息推送服务',
            callbackUrl: 'https://push.example.com/callback',
            captchaTypes: ['slide']
        },
        {
            id: 4, name: '数据分析', secret: 'sk_pqr012345',
            status: 'active', requestsPerDay: 5678,
            createdAt: '2024-01-12 16:45:00',
            description: '数据分析平台',
            callbackUrl: 'https://data.example.com/callback',
            captchaTypes: ['slide', 'rotate']
        },
        {
            id: 5, name: '文件存储', secret: 'sk_stu678901',
            status: 'suspended', requestsPerDay: 0,
            createdAt: '2024-01-15 11:20:00',
            description: '文件存储服务',
            callbackUrl: 'https://storage.example.com/callback',
            captchaTypes: ['click']
        },
        {
            id: 6, name: '社交平台', secret: 'sk_vwx234567',
            status: 'active', requestsPerDay: 15678,
            createdAt: '2024-01-20 08:00:00',
            description: '社交平台服务',
            callbackUrl: 'https://social.example.com/callback',
            captchaTypes: ['slide', 'click', 'rotate']
        },
        {
            id: 7, name: '电商后台', secret: 'sk_yza890123',
            status: 'active', requestsPerDay: 9876,
            createdAt: '2024-02-01 10:30:00',
            description: '电商后台管理',
            callbackUrl: 'https://mall.example.com/callback',
            captchaTypes: ['slide', 'click']
        },
        {
            id: 8, name: '游戏中心', secret: 'sk_bcd456789',
            status: 'active', requestsPerDay: 7654,
            createdAt: '2024-02-10 14:00:00',
            description: '游戏中心服务',
            callbackUrl: 'https://game.example.com/callback',
            captchaTypes: ['slide', 'rotate']
        }
    ];
}

function filterApps(apps, keyword, status) {
    return apps.filter(app => {
        if (status && app.status !== status) return false;
        if (keyword) {
            const kw = keyword.toLowerCase();
            if (!app.name.toLowerCase().includes(kw) && !String(app.id).includes(kw)) {
                return false;
            }
        }
        return true;
    });
}

function sortApps(apps, sort) {
    return apps.sort((a, b) => {
        switch (sort) {
            case 'requests':
                return b.requestsPerDay - a.requestsPerDay;
            case 'name':
                return a.name.localeCompare(b.name);
            default:
                return new Date(b.createdAt) - new Date(a.createdAt);
        }
    });
}

function renderApplications() {
    if (currentView === 'table') {
        renderApplicationsTable();
    } else {
        renderApplicationsCards();
    }
}

function renderApplicationsTable() {
    const tbody = document.getElementById('appsTableBody');
    if (!tbody) return;

    document.getElementById('appsTableView')?.classList.remove('d-none');
    document.getElementById('appsCardView')?.classList.add('d-none');

    tbody.innerHTML = currentApps.map(app => {
        const successRate = (Math.random() * 5 + 95).toFixed(1);
        const responseTime = Math.floor(Math.random() * 100) + 80;
        const quotaUsage = Math.floor(Math.random() * 30) + 50;
        
        return `
        <tr class="${selectedApps.has(app.id) ? 'table-primary' : ''}">
            <td>
                <input type="checkbox" class="form-check-input app-checkbox" 
                    data-id="${app.id}" ${selectedApps.has(app.id) ? 'checked' : ''}
                    onchange="toggleAppSelection(${app.id})">
            </td>
            <td>${app.id}</td>
            <td>
                <strong>${escapeHtml(app.name)}</strong>
                ${app.description ? `<br><small class="text-muted">${escapeHtml(app.description)}</small>` : ''}
            </td>
            <td>
                <code class="secret-code">${maskSecret(app.secret)}</code>
                <button class="btn btn-sm btn-link p-0 ms-1" onclick="copySecret('${app.secret}')" title="复制">
                    <i class="fas fa-copy"></i>
                </button>
            </td>
            <td><span class="badge ${getStatusBadgeClass(app.status)}">${getStatusText(app.status)}</span></td>
            <td>
                ${formatNumber(app.requestsPerDay)}
                <div class="progress mt-1" style="height: 3px; width: 60px;">
                    <div class="progress-bar bg-info" style="width: ${Math.min(app.requestsPerDay / 200, 100)}%"></div>
                </div>
            </td>
            <td>
                <span class="badge ${successRate >= 98 ? 'bg-success' : successRate >= 95 ? 'bg-warning' : 'bg-danger'}">
                    ${successRate}%
                </span>
            </td>
            <td>
                <span class="badge ${responseTime <= 150 ? 'bg-success' : responseTime <= 300 ? 'bg-warning' : 'bg-danger'}">
                    ${responseTime}ms
                </span>
            </td>
            <td>
                <div class="d-flex align-items-center">
                    <span class="mr-2">${quotaUsage}%</span>
                    <div class="progress flex-grow-1" style="height: 4px; width: 60px;">
                        <div class="progress-bar ${quotaUsage >= 90 ? 'bg-danger' : quotaUsage >= 70 ? 'bg-warning' : 'bg-success'}" 
                             style="width: ${quotaUsage}%"></div>
                    </div>
                </div>
            </td>
            <td><small class="text-muted">${app.keyStatus || '有效'}</small></td>
            <td><small class="text-muted">${app.createdAt}</small></td>
            <td>
                <div class="btn-group btn-group-sm">
                    <button class="btn btn-outline-secondary" onclick="viewAppDetail(${app.id})" title="详情"><i class="fas fa-eye"></i></button>
                    <button class="btn btn-outline-info" onclick="viewAppStats(${app.id})" title="统计"><i class="fas fa-chart-line"></i></button>
                    <button class="btn btn-outline-primary" onclick="editApp(${app.id})" title="编辑"><i class="fas fa-edit"></i></button>
                    <button class="btn btn-outline-danger" onclick="deleteApp(${app.id})" title="删除"><i class="fas fa-trash"></i></button>
                </div>
            </td>
        </tr>
    `}).join('');

    updateBatchToolbar();
}

function renderApplicationsCards() {
    const container = document.getElementById('appsCardView');
    if (!container) return;

    container.classList.remove('d-none');
    document.getElementById('appsTableView')?.classList.add('d-none');

    container.innerHTML = `<div class="row g-3 p-3">${currentApps.map(app => `
        <div class="col-md-6 col-lg-4">
            <div class="card border">
                <div class="card-header bg-transparent d-flex justify-content-between align-items-center py-2">
                    <strong>${escapeHtml(app.name)}</strong>
                    <span class="badge ${getStatusBadgeClass(app.status)}">${getStatusText(app.status)}</span>
                </div>
                <div class="card-body py-2">
                    <p class="card-text small text-muted mb-2">${escapeHtml(app.description || '暂无描述')}</p>
                    <div class="d-flex justify-content-between align-items-center">
                        <small class="text-muted">ID: ${app.id}</small>
                        <small class="text-muted">${formatNumber(app.requestsPerDay)}/日</small>
                    </div>
                </div>
                <div class="card-footer bg-transparent py-2">
                    <div class="btn-group btn-group-sm w-100">
                        <button class="btn btn-outline-secondary" onclick="viewAppDetail(${app.id})"><i class="fas fa-eye me-1"></i>详情</button>
                        <button class="btn btn-outline-info" onclick="viewAppStats(${app.id})"><i class="fas fa-chart-line me-1"></i>统计</button>
                        <button class="btn btn-outline-primary" onclick="editApp(${app.id})"><i class="fas fa-edit me-1"></i>编辑</button>
                    </div>
                </div>
            </div>
        </div>
    `).join('')}</div>`;
}

function switchView(view) {
    currentView = view;
    renderApplications();
}

function copySecret(secret) {
    navigator.clipboard.writeText(secret).then(() => {
        showToast('密钥已复制到剪贴板', 'success');
    }).catch(() => {
        showToast('复制失败', 'danger');
    });
}

function renderAppsCount(total) {
    const countEl = document.getElementById('appsCount');
    if (countEl) countEl.textContent = total;
}

function renderPagination(total) {
    const pagination = document.getElementById('pagination');
    if (!pagination) return;

    const totalPages = Math.ceil(total / pageSize);
    if (totalPages <= 1) {
        pagination.innerHTML = '';
        return;
    }

    let html = '<div class="d-flex justify-content-between align-items-center">';
    html += `<span class="text-muted">第 ${currentPage} / ${totalPages} 页，共 ${total} 条</span>`;
    html += '<div class="btn-group btn-group-sm">';
    html += `<button class="btn btn-outline-secondary" onclick="changePage(${currentPage - 1})" ${currentPage === 1 ? 'disabled' : ''}>上一页</button>`;

    const startPage = Math.max(1, currentPage - 2);
    const endPage = Math.min(totalPages, currentPage + 2);

    if (startPage > 1) {
        html += `<button class="btn btn-outline-secondary" onclick="changePage(1)">1</button>`;
        if (startPage > 2) html += `<button class="btn btn-outline-secondary" disabled>...</button>`;
    }

    for (let i = startPage; i <= endPage; i++) {
        html += `<button class="btn ${i === currentPage ? 'btn-primary' : 'btn-outline-secondary'}" onclick="changePage(${i})">${i}</button>`;
    }

    if (endPage < totalPages) {
        if (endPage < totalPages - 1) html += `<button class="btn btn-outline-secondary" disabled>...</button>`;
        html += `<button class="btn btn-outline-secondary" onclick="changePage(${totalPages})">${totalPages}</button>`;
    }

    html += `<button class="btn btn-outline-secondary" onclick="changePage(${currentPage + 1})" ${currentPage === totalPages ? 'disabled' : ''}>下一页</button>`;
    html += '</div></div>';

    pagination.innerHTML = html;
}

function changePage(page) {
    currentPage = page;
    loadApplications();
}

function openAppModal(app = null) {
    const modal = document.getElementById('appModal');
    const title = document.getElementById('modalTitle');

    if (app) {
        title.textContent = '编辑应用';
        document.getElementById('appId').value = app.id;
        document.getElementById('appName').value = app.name;
        document.getElementById('appDescription').value = app.description || '';
        document.getElementById('appCallbackUrl').value = app.callbackUrl || '';
        document.getElementById('appStatusSelect').value = app.status;

        document.getElementById('captchaSlide').checked = app.captchaTypes?.includes('slide') || false;
        document.getElementById('captchaClick').checked = app.captchaTypes?.includes('click') || false;
        document.getElementById('captchaRotate').checked = app.captchaTypes?.includes('rotate') || false;
    } else {
        title.textContent = '创建应用';
        document.getElementById('appId').value = '';
        document.getElementById('appForm')?.reset();
        document.getElementById('captchaSlide').checked = true;
    }

    const bsModal = new bootstrap.Modal(modal);
    bsModal.show();
}

async function handleAppSubmit(e) {
    e.preventDefault();

    const appId = document.getElementById('appId').value;
    const captchaTypes = [];
    if (document.getElementById('captchaSlide').checked) captchaTypes.push('slide');
    if (document.getElementById('captchaClick').checked) captchaTypes.push('click');
    if (document.getElementById('captchaRotate').checked) captchaTypes.push('rotate');

    const appData = {
        name: document.getElementById('appName').value,
        description: document.getElementById('appDescription').value,
        callback_url: document.getElementById('appCallbackUrl').value,
        status: document.getElementById('appStatusSelect').value,
        captcha_types: captchaTypes
    };

    try {
        if (appId) {
            await auth.request(`/admin/applications/${appId}`, {
                method: 'PUT',
                body: JSON.stringify(appData)
            });
            showToast('应用更新成功', 'success');
        } else {
            await auth.request('/admin/applications', {
                method: 'POST',
                body: JSON.stringify(appData)
            });
            showToast('应用创建成功', 'success');
        }

        bootstrap.Modal.getInstance(document.getElementById('appModal'))?.hide();
        loadApplications();
        loadApplicationsSummary();
    } catch (error) {
        showToast('保存应用失败', 'danger');
    }
}

function editApp(appId) {
    const app = currentApps.find(a => a.id === appId);
    if (app) {
        openAppModal(app);
    }
}

async function deleteApp(appId) {
    if (!confirm('确定要删除这个应用吗？此操作不可恢复。')) return;

    try {
        await auth.request(`/admin/applications/${appId}`, {
            method: 'DELETE'
        });
        showToast('应用删除成功', 'success');
        loadApplications();
        loadApplicationsSummary();
    } catch (error) {
        showToast('删除应用失败', 'danger');
    }
}

function viewAppDetail(appId) {
    const app = currentApps.find(a => a.id === appId);
    if (!app) return;

    const modal = document.getElementById('appDetailModal');
    const content = document.getElementById('appDetailContent');

    content.innerHTML = `
        <div class="row">
            <div class="col-md-6">
                <table class="table table-borderless">
                    <tr><td class="text-muted">应用ID</td><td>${app.id}</td></tr>
                    <tr><td class="text-muted">应用名称</td><td><strong>${escapeHtml(app.name)}</strong></td></tr>
                    <tr><td class="text-muted">应用密钥</td><td><code>${app.secret}</code> <button class="btn btn-sm btn-link p-0" onclick="copySecret('${app.secret}')"><i class="fas fa-copy"></i></button></td></tr>
                    <tr><td class="text-muted">状态</td><td><span class="badge ${getStatusBadgeClass(app.status)}">${getStatusText(app.status)}</span></td></tr>
                    <tr><td class="text-muted">创建时间</td><td>${app.createdAt}</td></tr>
                </table>
            </div>
            <div class="col-md-6">
                <table class="table table-borderless">
                    <tr><td class="text-muted">描述</td><td>${escapeHtml(app.description || '-')}</td></tr>
                    <tr><td class="text-muted">回调URL</td><td><small>${escapeHtml(app.callbackUrl || '-')}</small></td></tr>
                    <tr><td class="text-muted">验证码类型</td><td>${app.captchaTypes?.map(t => `<span class="badge bg-info me-1">${getCaptchaTypeText(t)}</span>`).join('') || '-'}</td></tr>
                    <tr><td class="text-muted">日请求量</td><td class="text-primary fw-bold">${formatNumber(app.requestsPerDay)}</td></tr>
                </table>
            </div>
        </div>
        <div class="alert alert-info mt-3">
            <i class="fas fa-key me-1"></i>
            请妥善保管应用密钥，泄露后可能导致安全问题。
        </div>
    `;

    const bsModal = new bootstrap.Modal(modal);
    bsModal.show();
}

function viewAppStats(appId) {
    const app = currentApps.find(a => a.id === appId);
    if (!app) return;

    const modal = document.getElementById('appStatsModal');
    const bsModal = new bootstrap.Modal(modal);
    bsModal.show();

    document.getElementById('modalTodayRequests').textContent = formatNumber(app.requestsPerDay);
    document.getElementById('modalSuccessRate').textContent = '98.5%';
    document.getElementById('modalAvgResponse').textContent = '125ms';
    document.getElementById('modalActiveUsers').textContent = formatNumber(Math.floor(app.requestsPerDay * 0.8));

    setTimeout(() => initAppStatsChart(app), 100);
}

function initAppStatsChart(app) {
    const ctx = document.getElementById('appStatsChart');
    if (!ctx) return;

    if (appStatsChart) {
        appStatsChart.destroy();
    }

    const labels = Array.from({ length: 7 }, (_, i) => {
        const date = new Date(Date.now() - (6 - i) * 24 * 60 * 60 * 1000);
        return `${date.getMonth() + 1}/${date.getDate()}`;
    });

    const data = labels.map(() => Math.floor(app.requestsPerDay * (0.8 + Math.random() * 0.4)));

    appStatsChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: labels,
            datasets: [{
                label: '请求量',
                data: data,
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

function getStatusBadgeClass(status) {
    const map = {
        'active': 'bg-success',
        'inactive': 'bg-secondary',
        'suspended': 'bg-warning'
    };
    return map[status] || 'bg-secondary';
}

function getStatusText(status) {
    const map = {
        'active': '活跃',
        'inactive': '停用',
        'suspended': '暂停'
    };
    return map[status] || status;
}

function getCaptchaTypeText(type) {
    const map = {
        'slide': '滑块',
        'click': '点选',
        'rotate': '旋转'
    };
    return map[type] || type;
}

function maskSecret(secret) {
    if (!secret) return '';
    return secret.substring(0, 4) + '...' + secret.substring(secret.length - 4);
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

function toggleAppSelection(appId) {
    if (selectedApps.has(appId)) {
        selectedApps.delete(appId);
    } else {
        selectedApps.add(appId);
    }
    updateBatchToolbar();
    updateTableRowStyles();
}

function toggleSelectAll(selectAll) {
    if (selectAll) {
        currentApps.forEach(app => selectedApps.add(app.id));
    } else {
        selectedApps.clear();
    }
    updateBatchToolbar();
    updateTableRowStyles();
}

function updateTableRowStyles() {
    document.querySelectorAll('.app-checkbox').forEach(checkbox => {
        const row = checkbox.closest('tr');
        const appId = parseInt(checkbox.dataset.id);
        if (selectedApps.has(appId)) {
            row.classList.add('table-primary');
        } else {
            row.classList.remove('table-primary');
        }
        checkbox.checked = selectedApps.has(appId);
    });
}

function updateBatchToolbar() {
    const batchToolbar = document.getElementById('batchToolbar');
    const selectedCount = document.getElementById('selectedAppCount');
    
    if (batchToolbar) {
        if (selectedApps.size > 0) {
            batchToolbar.classList.remove('d-none');
        } else {
            batchToolbar.classList.add('d-none');
        }
    }
    
    if (selectedCount) {
        selectedCount.textContent = selectedApps.size;
    }
}

async function exportSelectedApps() {
    const exportApps = selectedApps.size > 0 ? 
        currentApps.filter(app => selectedApps.has(app.id)) : 
        currentApps;
    
    if (exportApps.length === 0) {
        showToast('没有可导出的应用', 'warning');
        return;
    }
    
    const csvContent = [
        ['应用ID', '应用名称', '描述', '状态', '日请求量', '创建时间', '密钥状态'].join(','),
        ...exportApps.map(app => [
            app.id,
            `"${escapeHtml(app.name)}"`,
            `"${escapeHtml(app.description || '')}"`,
            app.status,
            app.requestsPerDay,
            app.createdAt,
            app.keyStatus || '有效'
        ].join(','))
    ].join('\n');
    
    downloadFile(csvContent, `applications_${new Date().toISOString().slice(0,10)}.csv`, 'text/csv;charset=utf-8');
    showToast(`已导出 ${exportApps.length} 个应用`, 'success');
}

async function batchDeleteApps() {
    if (selectedApps.size === 0) {
        showToast('请先选择要删除的应用', 'warning');
        return;
    }
    
    if (!confirm(`确定要删除选中的 ${selectedApps.size} 个应用吗？此操作不可恢复。`)) {
        return;
    }
    
    try {
        const deletePromises = Array.from(selectedApps).map(appId => 
            auth.request(`/admin/applications/${appId}`, { method: 'DELETE' })
        );
        
        await Promise.all(deletePromises);
        showToast(`成功删除 ${selectedApps.size} 个应用`, 'success');
        selectedApps.clear();
        loadApplications();
        loadApplicationsSummary();
    } catch (error) {
        showToast('批量删除失败', 'danger');
    }
}

function showBatchStatusModal() {
    if (selectedApps.size === 0) {
        showToast('请先选择要修改的应用', 'warning');
        return;
    }
    
    const modal = new bootstrap.Modal(document.getElementById('batchStatusModal'));
    modal.show();
}

async function applyBatchStatus() {
    const newStatus = document.getElementById('batchNewStatus')?.value;
    if (!newStatus) {
        showToast('请选择新状态', 'warning');
        return;
    }
    
    try {
        const updatePromises = Array.from(selectedApps).map(appId => 
            auth.request(`/admin/applications/${appId}`, {
                method: 'PUT',
                body: JSON.stringify({ status: newStatus })
            })
        );
        
        await Promise.all(updatePromises);
        showToast(`成功更新 ${selectedApps.size} 个应用的状态`, 'success');
        
        bootstrap.Modal.getInstance(document.getElementById('batchStatusModal'))?.hide();
        selectedApps.clear();
        loadApplications();
        loadApplicationsSummary();
    } catch (error) {
        showToast('批量更新失败', 'danger');
    }
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
