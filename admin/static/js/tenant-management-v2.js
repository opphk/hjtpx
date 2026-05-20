let tenantTable;
let currentTenantId = null;
let quotaChart = null;
let usageChart = null;

const API_BASE = '/api/v1/tenant-v2';

document.addEventListener('DOMContentLoaded', () => {
    initializeTenantManagement();
    setupEventListeners();
    loadTenantList();
});

function initializeTenantManagement() {
    initializeTenantTable();
    initializeCharts();
}

function initializeTenantTable() {
    const tableElement = document.getElementById('tenantTable');
    if (!tableElement) return;

    tenantTable = new DataTable(tableElement, {
        ajax: {
            url: `${API_BASE}/list`,
            dataSrc: function(json) {
                return json.data || [];
            }
        },
        columns: [
            { data: 'tenantId', title: 'ID' },
            { data: 'tenantCode', title: 'Code' },
            { data: 'tenantName', title: 'Name' },
            { data: 'plan', title: 'Plan' },
            { data: 'tier', title: 'Tier' },
            { data: 'status', title: 'Status' },
            { 
                data: 'quota.maxUsers',
                title: 'Users',
                render: function(data, type, row) {
                    return `${row.currentUsers || 0} / ${data}`;
                }
            },
            {
                data: null,
                title: 'Quota Usage',
                render: function(data, type, row) {
                    const usage = row.quotaUsagePercent || 0;
                    return `
                        <div class="progress" style="height: 20px;">
                            <div class="progress-bar ${getQuotaProgressClass(usage)}" 
                                 role="progressbar" 
                                 style="width: ${Math.min(usage, 100)}%">
                                ${usage.toFixed(0)}%
                            </div>
                        </div>
                    `;
                }
            },
            {
                data: null,
                title: 'Actions',
                orderable: false,
                render: function(data, type, row) {
                    return `
                        <div class="btn-group btn-group-sm">
                            <button class="btn btn-outline-primary" onclick="viewTenantDetails(${row.tenantId})">
                                <i class="fas fa-eye"></i>
                            </button>
                            <button class="btn btn-outline-warning" onclick="editTenantQuota(${row.tenantId})">
                                <i class="fas fa-edit"></i>
                            </button>
                            <button class="btn btn-outline-danger" onclick="deleteTenant(${row.tenantId})">
                                <i class="fas fa-trash"></i>
                            </button>
                        </div>
                    `;
                }
            }
        ],
        order: [[0, 'desc']],
        pageLength: 25,
        dom: 'Bfrtip',
        buttons: [
            'copy', 'csv', 'excel', 'pdf', 'print'
        ]
    });
}

function initializeCharts() {
    const quotaCtx = document.getElementById('quotaChart');
    if (quotaCtx) {
        quotaChart = new Chart(quotaCtx, {
            type: 'radar',
            data: {
                labels: ['Users', 'Applications', 'Storage', 'Bandwidth', 'Webhooks', 'Rules'],
                datasets: [{
                    label: 'Quota',
                    data: [0, 0, 0, 0, 0, 0],
                    borderColor: 'rgb(75, 192, 192)',
                    backgroundColor: 'rgba(75, 192, 192, 0.2)'
                }, {
                    label: 'Usage',
                    data: [0, 0, 0, 0, 0, 0],
                    borderColor: 'rgb(255, 99, 132)',
                    backgroundColor: 'rgba(255, 99, 132, 0.2)'
                }]
            },
            options: {
                responsive: true,
                scales: {
                    r: {
                        beginAtZero: true
                    }
                }
            }
        });
    }

    const usageCtx = document.getElementById('usageChart');
    if (usageCtx) {
        usageChart = new Chart(usageCtx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [{
                    label: 'API Requests',
                    data: [],
                    borderColor: 'rgb(75, 192, 192)',
                    tension: 0.1
                }, {
                    label: 'Success Rate',
                    data: [],
                    borderColor: 'rgb(54, 162, 235)',
                    tension: 0.1,
                    yAxisID: 'y1'
                }]
            },
            options: {
                responsive: true,
                interaction: {
                    mode: 'index',
                    intersect: false
                },
                scales: {
                    y: {
                        type: 'linear',
                        display: true,
                        position: 'left'
                    },
                    y1: {
                        type: 'linear',
                        display: true,
                        position: 'right',
                        min: 0,
                        max: 100
                    }
                }
            }
        });
    }
}

function setupEventListeners() {
    const createTenantBtn = document.getElementById('createTenantBtn');
    if (createTenantBtn) {
        createTenantBtn.addEventListener('click', showCreateTenantModal);
    }

    const searchInput = document.getElementById('tenantSearch');
    if (searchInput) {
        searchInput.addEventListener('input', debounce(searchTenants, 300));
    }

    const planFilter = document.getElementById('planFilter');
    if (planFilter) {
        planFilter.addEventListener('change', filterTenants);
    }

    const statusFilter = document.getElementById('statusFilter');
    if (statusFilter) {
        statusFilter.addEventListener('change', filterTenants);
    }

    document.querySelectorAll('.nav-tabs .nav-link').forEach(tab => {
        tab.addEventListener('shown.bs.tab', handleTabSwitch);
    });

    const refreshBtn = document.getElementById('refreshTenantsBtn');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', loadTenantList);
    }
}

function handleTabSwitch(event) {
    const tabId = event.target.getAttribute('data-tab');
    
    switch(tabId) {
        case 'overview':
            loadDashboardOverview();
            break;
        case 'tenants':
            loadTenantList();
            break;
        case 'quotas':
            loadQuotaOverview();
            break;
        case 'permissions':
            loadPermissionsOverview();
            break;
        case 'analytics':
            loadCrossTenantAnalytics();
            break;
        case 'self-service':
            loadSelfServicePortal();
            break;
    }
}

async function loadTenantList() {
    try {
        const response = await fetch(`${API_BASE}/list`);
        const data = await response.json();

        if (tenantTable) {
            tenantTable.clear();
            tenantTable.rows.add(data.data || []);
            tenantTable.draw();
        }

        updateTenantStats(data.stats);
    } catch (error) {
        console.error('Failed to load tenant list:', error);
        showToast('Failed to load tenant list', 'error');
    }
}

function updateTenantStats(stats) {
    if (!stats) return;

    updateStatCard('totalTenants', stats.totalTenants);
    updateStatCard('activeTenants', stats.activeTenants);
    updateStatCard('totalUsers', stats.totalUsers);
    updateStatCard('totalQuotaUsage', `${stats.averageQuotaUsage?.toFixed(1) || 0}%`);
}

function updateStatCard(cardId, value) {
    const card = document.getElementById(cardId);
    if (card) {
        const valueElement = card.querySelector('.stat-value');
        if (valueElement) {
            valueElement.textContent = value;
        }
    }
}

function getQuotaProgressClass(usage) {
    if (usage >= 100) return 'bg-danger';
    if (usage >= 80) return 'bg-warning';
    return 'bg-success';
}

async function searchTenants(query) {
    if (!query) {
        loadTenantList();
        return;
    }

    try {
        const response = await fetch(`${API_BASE}/search?q=${encodeURIComponent(query)}`);
        const data = await response.json();

        if (tenantTable) {
            tenantTable.clear();
            tenantTable.rows.add(data.data || []);
            tenantTable.draw();
        }
    } catch (error) {
        console.error('Failed to search tenants:', error);
        showToast('Failed to search tenants', 'error');
    }
}

async function filterTenants() {
    const plan = document.getElementById('planFilter')?.value;
    const status = document.getElementById('statusFilter')?.value;

    const params = new URLSearchParams();
    if (plan) params.append('plan', plan);
    if (status) params.append('status', status);

    try {
        const response = await fetch(`${API_BASE}/list?${params.toString()}`);
        const data = await response.json();

        if (tenantTable) {
            tenantTable.clear();
            tenantTable.rows.add(data.data || []);
            tenantTable.draw();
        }
    } catch (error) {
        console.error('Failed to filter tenants:', error);
        showToast('Failed to filter tenants', 'error');
    }
}

function showCreateTenantModal() {
    const modal = new bootstrap.Modal(document.getElementById('createTenantModal'));
    resetCreateTenantForm();
    modal.show();
}

function resetCreateTenantForm() {
    const form = document.getElementById('createTenantForm');
    if (form) {
        form.reset();
    }
}

async function createTenant(event) {
    event.preventDefault();

    const form = event.target;
    const formData = new FormData(form);
    const tenantData = Object.fromEntries(formData);

    try {
        const response = await fetch(`${API_BASE}/create`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(tenantData)
        });

        if (!response.ok) {
            throw new Error('Failed to create tenant');
        }

        const result = await response.json();
        showToast('Tenant created successfully', 'success');

        bootstrap.Modal.getInstance(document.getElementById('createTenantModal')).hide();
        loadTenantList();
    } catch (error) {
        console.error('Failed to create tenant:', error);
        showToast('Failed to create tenant', 'error');
    }
}

async function viewTenantDetails(tenantId) {
    try {
        const response = await fetch(`${API_BASE}/${tenantId}`);
        const tenant = await response.json();

        currentTenantId = tenantId;
        displayTenantDetails(tenant);
        displayQuotaChart(tenant.quota);
        displayUsageChart(tenant.stats);

        const modal = new bootstrap.Modal(document.getElementById('tenantDetailsModal'));
        modal.show();
    } catch (error) {
        console.error('Failed to load tenant details:', error);
        showToast('Failed to load tenant details', 'error');
    }
}

function displayTenantDetails(tenant) {
    const detailsContainer = document.getElementById('tenantDetailsContent');
    if (!detailsContainer) return;

    detailsContainer.innerHTML = `
        <div class="row">
            <div class="col-md-6">
                <h6>Basic Information</h6>
                <table class="table table-sm">
                    <tr>
                        <td>Tenant ID</td>
                        <td><strong>${tenant.tenantId}</strong></td>
                    </tr>
                    <tr>
                        <td>Code</td>
                        <td>${tenant.tenantCode}</td>
                    </tr>
                    <tr>
                        <td>Name</td>
                        <td>${tenant.tenantName}</td>
                    </tr>
                    <tr>
                        <td>Plan</td>
                        <td><span class="badge bg-${getPlanBadgeColor(tenant.plan)}">${tenant.plan}</span></td>
                    </tr>
                    <tr>
                        <td>Tier</td>
                        <td><span class="badge bg-${getTierBadgeColor(tenant.tier)}">${tenant.tier}</span></td>
                    </tr>
                    <tr>
                        <td>Status</td>
                        <td><span class="badge bg-${tenant.status === 'active' ? 'success' : 'secondary'}">${tenant.status}</span></td>
                    </tr>
                </table>
            </div>
            <div class="col-md-6">
                <h6>Isolation Settings</h6>
                <table class="table table-sm">
                    <tr>
                        <td>Isolated DB</td>
                        <td>
                            <i class="fas fa-${tenant.isolatedDB ? 'check text-success' : 'times text-danger'}"></i>
                        </td>
                    </tr>
                    <tr>
                        <td>Isolated Cache</td>
                        <td>
                            <i class="fas fa-${tenant.isolatedCache ? 'check text-success' : 'times text-danger'}"></i>
                        </td>
                    </tr>
                    <tr>
                        <td>Isolated Storage</td>
                        <td>
                            <i class="fas fa-${tenant.isolatedStorage ? 'check text-success' : 'times text-danger'}"></i>
                        </td>
                    </tr>
                    <tr>
                        <td>Data Residency</td>
                        <td>${tenant.dataResidency || 'default'}</td>
                    </tr>
                </table>
            </div>
        </div>
        <div class="row mt-3">
            <div class="col-md-12">
                <h6>Compliance Certifications</h6>
                <div>
                    ${(tenant.complianceCerts || []).map(cert => `
                        <span class="badge bg-info me-2">${cert}</span>
                    `).join('')}
                </div>
            </div>
        </div>
        <div class="row mt-3">
            <div class="col-md-12">
                <h6>Features</h6>
                <div class="row">
                    ${Object.entries(tenant.features || {}).map(([feature, enabled]) => `
                        <div class="col-md-4 mb-2">
                            <i class="fas fa-${enabled ? 'check text-success' : 'times text-danger'} me-2"></i>
                            ${formatFeatureName(feature)}
                        </div>
                    `).join('')}
                </div>
            </div>
        </div>
    `;
}

function formatFeatureName(name) {
    return name.split('_').map(word => word.charAt(0).toUpperCase() + word.slice(1)).join(' ');
}

function getPlanBadgeColor(plan) {
    const colors = {
        'free': 'secondary',
        'starter': 'info',
        'pro': 'primary',
        'enterprise': 'warning'
    };
    return colors[plan] || 'secondary';
}

function getTierBadgeColor(tier) {
    const colors = {
        'basic': 'secondary',
        'standard': 'info',
        'professional': 'primary',
        'enterprise': 'warning'
    };
    return colors[tier] || 'secondary';
}

function displayQuotaChart(quota) {
    if (!quotaChart || !quota) return;

    const quotaData = [
        quota.maxUsers || 0,
        quota.maxApplications || 0,
        (quota.maxStorage || 0) / (1024 * 1024 * 1024),
        (quota.maxBandwidth || 0) / (1024 * 1024 * 1024),
        quota.maxWebhooks || 0,
        quota.maxRules || 0
    ];

    quotaChart.data.datasets[0].data = quotaData;
    quotaChart.update();
}

function displayUsageChart(stats) {
    if (!usageChart || !stats) return;

    const labels = stats.trendData?.map(p => formatTimestamp(p.timestamp)) || [];
    const apiRequests = stats.trendData?.map(p => p.value) || [];
    const successRates = stats.trendData?.map(p => (stats.successRate || 0) * 100) || [];

    usageChart.data.labels = labels;
    usageChart.data.datasets[0].data = apiRequests;
    usageChart.data.datasets[1].data = successRates;
    usageChart.update();
}

async function editTenantQuota(tenantId) {
    try {
        const response = await fetch(`${API_BASE}/${tenantId}`);
        const tenant = await response.json();

        populateQuotaEditForm(tenant);
        
        const modal = new bootstrap.Modal(document.getElementById('editQuotaModal'));
        modal.show();
    } catch (error) {
        console.error('Failed to load tenant for quota edit:', error);
        showToast('Failed to load tenant', 'error');
    }
}

function populateQuotaEditForm(tenant) {
    const form = document.getElementById('editQuotaForm');
    if (!form) return;

    form.elements.tenantId.value = tenant.tenantId;
    form.elements.maxUsers.value = tenant.quota?.maxUsers || 0;
    form.elements.maxApplications.value = tenant.quota?.maxApplications || 0;
    form.elements.maxAPIRequests.value = tenant.quota?.maxAPIRequests || 0;
    form.elements.maxStorage.value = tenant.quota?.maxStorage || 0;
    form.elements.maxBandwidth.value = tenant.quota?.maxBandwidth || 0;
    form.elements.maxWebhooks.value = tenant.quota?.maxWebhooks || 0;
    form.elements.maxRules.value = tenant.quota?.maxRules || 0;
}

async function updateQuota(event) {
    event.preventDefault();

    const form = event.target;
    const formData = new FormData(form);
    const tenantId = formData.get('tenantId');
    const quotaData = {
        maxUsers: parseInt(formData.get('maxUsers')),
        maxApplications: parseInt(formData.get('maxApplications')),
        maxAPIRequests: parseInt(formData.get('maxAPIRequests')),
        maxStorage: parseInt(formData.get('maxStorage')),
        maxBandwidth: parseInt(formData.get('maxBandwidth')),
        maxWebhooks: parseInt(formData.get('maxWebhooks')),
        maxRules: parseInt(formData.get('maxRules'))
    };

    try {
        const response = await fetch(`${API_BASE}/${tenantId}/quota`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(quotaData)
        });

        if (!response.ok) {
            throw new Error('Failed to update quota');
        }

        showToast('Quota updated successfully', 'success');
        bootstrap.Modal.getInstance(document.getElementById('editQuotaModal')).hide();
        loadTenantList();
    } catch (error) {
        console.error('Failed to update quota:', error);
        showToast('Failed to update quota', 'error');
    }
}

async function deleteTenant(tenantId) {
    if (!confirm('Are you sure you want to delete this tenant? This action cannot be undone.')) {
        return;
    }

    try {
        const response = await fetch(`${API_BASE}/${tenantId}`, {
            method: 'DELETE'
        });

        if (!response.ok) {
            throw new Error('Failed to delete tenant');
        }

        showToast('Tenant deleted successfully', 'success');
        loadTenantList();
    } catch (error) {
        console.error('Failed to delete tenant:', error);
        showToast('Failed to delete tenant', 'error');
    }
}

async function loadDashboardOverview() {
    try {
        const response = await fetch(`${API_BASE}/dashboard`);
        const data = await response.json();

        updateTenantStats(data.stats);
        displayTopTenants(data.topTenants);
        displayQuotaAlerts(data.alerts);
    } catch (error) {
        console.error('Failed to load dashboard overview:', error);
    }
}

function displayTopTenants(tenants) {
    const container = document.getElementById('topTenantsContainer');
    if (!container) return;

    container.innerHTML = tenants?.map(tenant => `
        <div class="d-flex justify-content-between align-items-center mb-2">
            <div>
                <strong>${tenant.tenantName}</strong>
                <br>
                <small class="text-muted">${tenant.tenantCode}</small>
            </div>
            <div class="text-end">
                <div class="badge bg-primary">${tenant.plan}</div>
                <br>
                <small>${tenant.totalUsers} users</small>
            </div>
        </div>
    `).join('') || '<p class="text-muted">No tenants found</p>';
}

function displayQuotaAlerts(alerts) {
    const container = document.getElementById('quotaAlertsContainer');
    if (!container) return;

    if (!alerts || alerts.length === 0) {
        container.innerHTML = '<p class="text-success"><i class="fas fa-check-circle"></i> All quotas within limits</p>';
        return;
    }

    container.innerHTML = alerts.map(alert => `
        <div class="alert alert-${getAlertClass(alert.level)} alert-dismissible fade show" role="alert">
            <strong>${alert.resource}:</strong> ${alert.message}
            <button type="button" class="btn-close" data-bs-dismiss="alert"></button>
        </div>
    `).join('');
}

function getAlertClass(level) {
    return level === 'critical' ? 'danger' : level === 'warning' ? 'warning' : 'info';
}

async function loadQuotaOverview() {
    try {
        const response = await fetch(`${API_BASE}/quotas`);
        const quotas = await response.json();

        displayQuotaOverviewTable(quotas);
    } catch (error) {
        console.error('Failed to load quota overview:', error);
    }
}

function displayQuotaOverviewTable(quotas) {
    const container = document.getElementById('quotaOverviewTable');
    if (!container) return;

    container.innerHTML = `
        <table class="table table-striped">
            <thead>
                <tr>
                    <th>Tenant</th>
                    <th>Users</th>
                    <th>Applications</th>
                    <th>API Requests</th>
                    <th>Storage</th>
                    <th>Actions</th>
                </tr>
            </thead>
            <tbody>
                ${quotas.map(quota => `
                    <tr>
                        <td>${quota.tenantName}</td>
                        <td>${quota.currentUsers} / ${quota.maxUsers}</td>
                        <td>${quota.currentApps} / ${quota.maxApplications}</td>
                        <td>${formatNumber(quota.currentRequests)} / ${formatNumber(quota.maxAPIRequests)}</td>
                        <td>${formatBytes(quota.maxStorage)}</td>
                        <td>
                            <button class="btn btn-sm btn-outline-primary" onclick="editTenantQuota(${quota.tenantId})">
                                Edit
                            </button>
                        </td>
                    </tr>
                `).join('')}
            </tbody>
        </table>
    `;
}

function formatNumber(num) {
    return num?.toLocaleString() || '0';
}

function formatBytes(bytes) {
    if (!bytes) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

async function loadPermissionsOverview() {
    try {
        const response = await fetch(`${API_BASE}/permissions`);
        const permissions = await response.json();

        displayPermissionsMatrix(permissions);
    } catch (error) {
        console.error('Failed to load permissions overview:', error);
    }
}

function displayPermissionsMatrix(permissions) {
    const container = document.getElementById('permissionsMatrix');
    if (!container) return;

    container.innerHTML = `
        <table class="table table-bordered">
            <thead>
                <tr>
                    <th>Permission</th>
                    <th>Free</th>
                    <th>Starter</th>
                    <th>Pro</th>
                    <th>Enterprise</th>
                </tr>
            </thead>
            <tbody>
                ${Object.entries(permissions.matrix).map(([perm, plans]) => `
                    <tr>
                        <td>${formatPermissionName(perm)}</td>
                        <td class="${plans.free ? 'table-success' : 'table-danger'}">
                            <i class="fas fa-${plans.free ? 'check' : 'times'}"></i>
                        </td>
                        <td class="${plans.starter ? 'table-success' : 'table-danger'}">
                            <i class="fas fa-${plans.starter ? 'check' : 'times'}"></i>
                        </td>
                        <td class="${plans.pro ? 'table-success' : 'table-danger'}">
                            <i class="fas fa-${plans.pro ? 'check' : 'times'}"></i>
                        </td>
                        <td class="${plans.enterprise ? 'table-success' : 'table-danger'}">
                            <i class="fas fa-${plans.enterprise ? 'check' : 'times'}"></i>
                        </td>
                    </tr>
                `).join('')}
            </tbody>
        </table>
    `;
}

function formatPermissionName(name) {
    return name.split('_').map(word => word.charAt(0).toUpperCase() + word.slice(1)).join(' ');
}

async function loadCrossTenantAnalytics() {
    try {
        const response = await fetch(`${API_BASE}/analytics/cross-tenant`);
        const analytics = await response.json();

        displayCrossTenantCharts(analytics);
    } catch (error) {
        console.error('Failed to load cross-tenant analytics:', error);
    }
}

function displayCrossTenantCharts(analytics) {
    const container = document.getElementById('crossTenantAnalytics');
    if (!container) return;

    container.innerHTML = `
        <div class="row">
            <div class="col-md-6">
                <div class="card">
                    <div class="card-body">
                        <h6>Requests by Plan</h6>
                        <canvas id="requestsByPlanChart"></canvas>
                    </div>
                </div>
            </div>
            <div class="col-md-6">
                <div class="card">
                    <div class="card-body">
                        <h6>Users by Plan</h6>
                        <canvas id="usersByPlanChart"></canvas>
                    </div>
                </div>
            </div>
        </div>
        <div class="row mt-3">
            <div class="col-md-12">
                <div class="card">
                    <div class="card-body">
                        <h6>Trend Analysis</h6>
                        <canvas id="trendAnalysisChart"></canvas>
                    </div>
                </div>
            </div>
        </div>
    `;

    renderAnalyticsCharts(analytics);
}

function renderAnalyticsCharts(analytics) {
    if (analytics.requestsByPlan) {
        new Chart(document.getElementById('requestsByPlanChart'), {
            type: 'doughnut',
            data: {
                labels: Object.keys(analytics.requestsByPlan),
                datasets: [{
                    data: Object.values(analytics.requestsByPlan),
                    backgroundColor: [
                        '#6c757d', '#0dcaf0', '#0d6efd', '#ffc107'
                    ]
                }]
            }
        });
    }

    if (analytics.usersByPlan) {
        new Chart(document.getElementById('usersByPlanChart'), {
            type: 'pie',
            data: {
                labels: Object.keys(analytics.usersByPlan),
                datasets: [{
                    data: Object.values(analytics.usersByPlan),
                    backgroundColor: [
                        '#6c757d', '#0dcaf0', '#0d6efd', '#ffc107'
                    ]
                }]
            }
        });
    }
}

async function loadSelfServicePortal() {
    try {
        const response = await fetch(`${API_BASE}/self-service/portal`);
        const portal = await response.json();

        displaySelfServicePortal(portal);
    } catch (error) {
        console.error('Failed to load self-service portal:', error);
    }
}

function displaySelfServicePortal(portal) {
    const container = document.getElementById('selfServicePortal');
    if (!container) return;

    container.innerHTML = `
        <div class="row">
            <div class="col-md-3">
                <div class="card text-center">
                    <div class="card-body">
                        <i class="fas fa-tachometer-alt fa-3x mb-3"></i>
                        <h6>Dashboard</h6>
                        <a href="${portal.dashboardUrl}" class="btn btn-sm btn-primary">Open</a>
                    </div>
                </div>
            </div>
            <div class="col-md-3">
                <div class="card text-center">
                    <div class="card-body">
                        <i class="fas fa-cog fa-3x mb-3"></i>
                        <h6>Settings</h6>
                        <a href="${portal.settingsUrl}" class="btn btn-sm btn-primary">Open</a>
                    </div>
                </div>
            </div>
            <div class="col-md-3">
                <div class="card text-center">
                    <div class="card-body">
                        <i class="fas fa-credit-card fa-3x mb-3"></i>
                        <h6>Billing</h6>
                        <a href="${portal.billingUrl}" class="btn btn-sm btn-primary">Open</a>
                    </div>
                </div>
            </div>
            <div class="col-md-3">
                <div class="card text-center">
                    <div class="card-body">
                        <i class="fas fa-headset fa-3x mb-3"></i>
                        <h6>Support</h6>
                        <a href="${portal.supportUrl}" class="btn btn-sm btn-primary">Open</a>
                    </div>
                </div>
            </div>
        </div>
        <div class="row mt-3">
            <div class="col-md-12">
                <h6>Available Actions</h6>
                <div class="list-group">
                    ${portal.availableActions.map(action => `
                        <a href="#" class="list-group-item list-group-item-action">
                            <i class="fas fa-arrow-right me-2"></i>
                            ${formatActionName(action)}
                        </a>
                    `).join('')}
                </div>
            </div>
        </div>
    `;
}

function formatActionName(action) {
    return action.split('_').map(word => word.charAt(0).toUpperCase() + word.slice(1)).join(' ');
}

function formatTimestamp(timestamp) {
    const date = new Date(timestamp);
    return date.toLocaleString();
}

function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

function showToast(message, type = 'info') {
    const toastContainer = document.getElementById('toastContainer') || createToastContainer();

    const toast = document.createElement('div');
    toast.className = `toast align-items-center text-white bg-${type === 'error' ? 'danger' : type} border-0`;
    toast.setAttribute('role', 'alert');
    toast.innerHTML = `
        <div class="d-flex">
            <div class="toast-body">
                ${message}
            </div>
            <button type="button" class="btn-close btn-close-white me-2 m-auto" data-bs-dismiss="toast"></button>
        </div>
    `;

    toastContainer.appendChild(toast);
    const bsToast = new bootstrap.Toast(toast);
    bsToast.show();

    toast.addEventListener('hidden.bs.toast', () => {
        toast.remove();
    });
}

function createToastContainer() {
    const container = document.createElement('div');
    container.id = 'toastContainer';
    container.className = 'toast-container position-fixed top-0 end-0 p-3';
    document.body.appendChild(container);
    return container;
}
