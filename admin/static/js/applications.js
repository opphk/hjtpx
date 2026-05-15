let currentPage = 1;
let pageSize = 10;

document.addEventListener('DOMContentLoaded', () => {
    loadApplications();
    setupEventListeners();
});

function setupEventListeners() {
    const createAppBtn = document.getElementById('createAppBtn');
    if (createAppBtn) {
        createAppBtn.addEventListener('click', () => openAppModal());
    }

    const closeModal = document.getElementById('closeModal');
    if (closeModal) {
        closeModal.addEventListener('click', () => closeAppModal());
    }

    const cancelBtn = document.getElementById('cancelBtn');
    if (cancelBtn) {
        cancelBtn.addEventListener('click', () => closeAppModal());
    }

    const appForm = document.getElementById('appForm');
    if (appForm) {
        appForm.addEventListener('submit', handleAppSubmit);
    }

    const searchBtn = document.getElementById('searchBtn');
    if (searchBtn) {
        searchBtn.addEventListener('click', () => {
            currentPage = 1;
            loadApplications();
        });
    }
}

async function loadApplications() {
    const mockApps = getMockApplications();
    let apps = mockApps;

    try {
        const keyword = document.getElementById('searchApp')?.value || '';
        const status = document.getElementById('appStatus')?.value || '';
        const result = await auth.request(`/apps?page=${currentPage}&size=${pageSize}&keyword=${encodeURIComponent(keyword)}&status=${status}`);
        if (result.code === 0) {
            apps = result.data.list || [];
            renderPagination(result.data.total || apps.length);
        } else {
            renderPagination(apps.length);
        }
    } catch (error) {
        renderPagination(apps.length);
    }

    renderApplications(apps);
}

function getMockApplications() {
    return [
        { id: 'app_001', name: '用户中心', secret: 'sk_abc123def456', status: 'active', createdAt: '2024-01-01 10:00:00' },
        { id: 'app_002', name: '支付系统', secret: 'sk_xyz789123', status: 'active', createdAt: '2024-01-05 14:30:00' },
        { id: 'app_003', name: '消息推送', secret: 'sk_mno456789', status: 'inactive', createdAt: '2024-01-10 09:15:00' },
        { id: 'app_004', name: '数据分析', secret: 'sk_pqr012345', status: 'active', createdAt: '2024-01-12 16:45:00' },
        { id: 'app_005', name: '文件存储', secret: 'sk_stu678901', status: 'suspended', createdAt: '2024-01-15 11:20:00' }
    ];
}

function renderApplications(apps) {
    const tbody = document.getElementById('appsTableBody');
    if (!tbody) return;

    tbody.innerHTML = apps.map(app => `
        <tr>
            <td>${app.id}</td>
            <td>${app.name}</td>
            <td><code class="secret-code">${maskSecret(app.secret)}</code></td>
            <td><span class="status ${app.status}">${getStatusText(app.status)}</span></td>
            <td>${app.createdAt}</td>
            <td>
                <button class="btn btn-sm" onclick="editApp('${app.id}')">编辑</button>
                <button class="btn btn-sm btn-danger" onclick="deleteApp('${app.id}')">删除</button>
            </td>
        </tr>
    `).join('');
}

function maskSecret(secret) {
    if (!secret) return '';
    return secret.substring(0, 4) + '...' + secret.substring(secret.length - 4);
}

function getStatusText(status) {
    const map = {
        active: '活跃',
        inactive: '停用',
        suspended: '暂停'
    };
    return map[status] || status;
}

function renderPagination(total) {
    const pagination = document.getElementById('pagination');
    if (!pagination) return;

    const totalPages = Math.ceil(total / pageSize);
    let html = '';

    if (totalPages > 1) {
        html += `<button class="btn btn-sm" onclick="changePage(${currentPage - 1})" ${currentPage === 1 ? 'disabled' : ''}>上一页</button>`;
        
        for (let i = 1; i <= totalPages; i++) {
            html += `<button class="btn btn-sm ${i === currentPage ? 'btn-primary' : ''}" onclick="changePage(${i})">${i}</button>`;
        }
        
        html += `<button class="btn btn-sm" onclick="changePage(${currentPage + 1})" ${currentPage === totalPages ? 'disabled' : ''}>下一页</button>`;
    }

    pagination.innerHTML = html;
}

function changePage(page) {
    currentPage = page;
    loadApplications();
}

function openAppModal(app = null) {
    const modal = document.getElementById('appModal');
    const title = document.getElementById('modalTitle');
    const form = document.getElementById('appForm');

    if (app) {
        title.textContent = '编辑应用';
        document.getElementById('appId').value = app.id;
        document.getElementById('appName').value = app.name;
        document.getElementById('appDescription').value = app.description || '';
        document.getElementById('appStatusSelect').value = app.status;
    } else {
        title.textContent = '创建应用';
        form.reset();
        document.getElementById('appId').value = '';
    }

    modal.classList.remove('hidden');
}

function closeAppModal() {
    const modal = document.getElementById('appModal');
    modal.classList.add('hidden');
}

async function handleAppSubmit(e) {
    e.preventDefault();

    const appId = document.getElementById('appId').value;
    const appData = {
        name: document.getElementById('appName').value,
        description: document.getElementById('appDescription').value,
        status: document.getElementById('appStatusSelect').value
    };

    try {
        if (appId) {
            await auth.request(`/apps/${appId}`, {
                method: 'PUT',
                body: JSON.stringify(appData)
            });
        } else {
            await auth.request('/apps', {
                method: 'POST',
                body: JSON.stringify(appData)
            });
        }

        closeAppModal();
        loadApplications();
    } catch (error) {
        closeAppModal();
        loadApplications();
    }
}

function editApp(appId) {
    const mockApps = getMockApplications();
    const app = mockApps.find(a => a.id === appId);
    if (app) {
        openAppModal(app);
    }
}

async function deleteApp(appId) {
    if (!confirm('确定要删除这个应用吗？')) return;

    try {
        await auth.request(`/apps/${appId}`, {
            method: 'DELETE'
        });
    } catch (error) {
    }

    loadApplications();
}
