let rulesPage = 1;
let rulesPageSize = 10;
let currentRules = [];
let currentView = 'table';

document.addEventListener('DOMContentLoaded', () => {
    loadRiskRulesSummary();
    loadRiskRules();
    setupEventListeners();
});

function setupEventListeners() {
    const addRuleBtn = document.getElementById('addRuleBtn');
    if (addRuleBtn) {
        addRuleBtn.addEventListener('click', () => openRuleModal());
    }

    const searchBtn = document.getElementById('searchRulesBtn');
    if (searchBtn) {
        searchBtn.addEventListener('click', () => {
            rulesPage = 1;
            loadRiskRules();
        });
    }

    const selectAllCheckbox = document.getElementById('selectAllRules');
    if (selectAllCheckbox) {
        selectAllCheckbox.addEventListener('change', (e) => {
            const checkboxes = document.querySelectorAll('.rule-checkbox');
            checkboxes.forEach(cb => cb.checked = e.target.checked);
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

    const ruleForm = document.getElementById('ruleForm');
    if (ruleForm) {
        ruleForm.addEventListener('submit', handleRuleSubmit);
    }
}

async function loadRiskRulesSummary() {
    const mockSummary = getMockRulesSummary();

    try {
        const data = await auth.request('/admin/risk-rules/summary');
        if (data.code === 0) {
            updateRulesSummary(data.data);
        } else {
            updateRulesSummary(mockSummary);
        }
    } catch (error) {
        updateRulesSummary(mockSummary);
    }
}

function getMockRulesSummary() {
    return {
        totalRules: 24,
        activeRules: 18,
        blockedToday: 1234,
        riskAlerts: 56,
        blockRate: 2.5
    };
}

function updateRulesSummary(summary) {
    const totalEl = document.getElementById('totalRules');
    const activeEl = document.getElementById('activeRules');
    const blockedEl = document.getElementById('blockedRequests');
    const alertsEl = document.getElementById('riskAlerts');
    const rateEl = document.getElementById('blockRate');

    if (totalEl) totalEl.textContent = summary.totalRules;
    if (activeEl) activeEl.textContent = summary.activeRules;
    if (blockedEl) blockedEl.textContent = formatNumber(summary.blockedToday);
    if (alertsEl) alertsEl.textContent = summary.riskAlerts;
    if (rateEl) rateEl.textContent = `${summary.blockRate}%`;
}

function formatNumber(num) {
    if (num >= 1000000) {
        return (num / 1000000).toFixed(1) + 'M';
    } else if (num >= 1000) {
        return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
}

async function loadRiskRules() {
    const typeFilter = document.getElementById('ruleTypeFilter')?.value || '';
    const statusFilter = document.getElementById('ruleStatusFilter')?.value || '';
    const keyword = document.getElementById('ruleKeyword')?.value || '';
    const mockRules = getMockRules(typeFilter, statusFilter, keyword);

    try {
        const params = new URLSearchParams({
            page: rulesPage,
            size: rulesPageSize,
            type: typeFilter,
            status: statusFilter,
            keyword: keyword
        });

        const result = await auth.request(`/admin/risk-rules?${params.toString()}`);
        if (result.code === 0) {
            currentRules = result.data.list || [];
            renderRulesPagination(result.data.total || currentRules.length);
            renderRulesCount(result.data.total || currentRules.length);
        } else {
            currentRules = filterRules(mockRules, typeFilter, statusFilter, keyword);
            renderRulesPagination(currentRules.length);
            renderRulesCount(currentRules.length);
        }
    } catch (error) {
        currentRules = filterRules(mockRules, typeFilter, statusFilter, keyword);
        renderRulesPagination(currentRules.length);
        renderRulesCount(currentRules.length);
    }

    renderRules();
}

function getMockRules() {
    return [
        {
            id: 1,
            name: 'IP频率限制',
            type: 'rate_limit',
            description: '限制单个IP的请求频率',
            condition: '请求次数 > 100',
            action: 'captcha',
            priority: 10,
            enabled: true,
            hitCount: 1234,
            apps: ['all']
        },
        {
            id: 2,
            name: '恶意IP封禁',
            type: 'ip_block',
            description: '封禁已知恶意IP段',
            condition: 'IP in 黑名单',
            action: 'block',
            priority: 100,
            enabled: true,
            hitCount: 567,
            apps: ['all']
        },
        {
            id: 3,
            name: '异常行为检测',
            type: 'behavior',
            description: '检测异常用户行为模式',
            condition: '行为分数 < 60',
            action: 'captcha',
            priority: 5,
            enabled: true,
            hitCount: 89,
            apps: ['1', '2']
        },
        {
            id: 4,
            name: '设备指纹识别',
            type: 'device_fingerprint',
            description: '识别重复设备',
            condition: '设备重复率 > 80%',
            action: 'warning',
            priority: 3,
            enabled: false,
            hitCount: 23,
            apps: ['1']
        },
        {
            id: 5,
            name: '会话劫持检测',
            type: 'behavior',
            description: '检测会话异常',
            condition: 'IP变更 + UA变更',
            action: 'review',
            priority: 8,
            enabled: true,
            hitCount: 12,
            apps: ['1', '2', '3']
        },
        {
            id: 6,
            name: '批量注册限制',
            type: 'rate_limit',
            description: '限制批量注册行为',
            condition: '注册次数 > 10/分钟',
            action: 'block',
            priority: 10,
            enabled: true,
            hitCount: 345,
            apps: ['1']
        },
        {
            id: 7,
            name: '爬虫识别',
            type: 'behavior',
            description: '识别爬虫访问',
            condition: '请求特征匹配',
            action: 'captcha',
            priority: 5,
            enabled: true,
            hitCount: 678,
            apps: ['all']
        },
        {
            id: 8,
            name: '暴力破解防护',
            type: 'rate_limit',
            description: '防止暴力破解密码',
            condition: '失败次数 > 5/10分钟',
            action: 'block',
            priority: 10,
            enabled: true,
            hitCount: 234,
            apps: ['1']
        }
    ];
}

function filterRules(rules, type, status, keyword) {
    return rules.filter(rule => {
        if (type && rule.type !== type) return false;
        if (status === 'enabled' && !rule.enabled) return false;
        if (status === 'disabled' && rule.enabled) return false;
        if (keyword && !rule.name.toLowerCase().includes(keyword.toLowerCase())) return false;
        return true;
    });
}

function renderRules() {
    if (currentView === 'table') {
        renderRulesTable();
    } else {
        renderRulesCards();
    }
}

function renderRulesTable() {
    const tbody = document.getElementById('rulesTableBody');
    if (!tbody) return;

    tbody.innerHTML = currentRules.map(rule => `
        <tr>
            <td><input type="checkbox" class="rule-checkbox" data-id="${rule.id}"></td>
            <td>
                <strong>${escapeHtml(rule.name)}</strong>
                ${rule.description ? `<br><small class="text-muted">${escapeHtml(rule.description)}</small>` : ''}
            </td>
            <td><span class="badge ${getTypeBadgeClass(rule.type)}">${getTypeText(rule.type)}</span></td>
            <td><code>${escapeHtml(rule.condition)}</code></td>
            <td><span class="badge ${getActionBadgeClass(rule.action)}">${getActionText(rule.action)}</span></td>
            <td><span class="badge ${getPriorityBadgeClass(rule.priority)}">${getPriorityText(rule.priority)}</span></td>
            <td>
                <div class="form-check form-switch">
                    <input class="form-check-input" type="checkbox" role="switch" ${rule.enabled ? 'checked' : ''} onchange="toggleRule(${rule.id}, this.checked)">
                </div>
            </td>
            <td>
                <div class="btn-group btn-group-sm">
                    <button class="btn btn-outline-secondary" onclick="viewRuleDetail(${rule.id})" title="查看详情"><i class="fas fa-eye"></i></button>
                    <button class="btn btn-outline-primary" onclick="editRule(${rule.id})" title="编辑"><i class="fas fa-edit"></i></button>
                    <button class="btn btn-outline-danger" onclick="deleteRule(${rule.id})" title="删除"><i class="fas fa-trash"></i></button>
                </div>
            </td>
        </tr>
    `).join('');
}

function renderRulesCards() {
    const container = document.getElementById('rulesCardView');
    if (!container) return;

    container.classList.remove('d-none');
    document.getElementById('rulesTableView')?.classList.add('d-none');

    container.innerHTML = `<div class="row g-3 p-3">${currentRules.map(rule => `
        <div class="col-md-6 col-lg-4">
            <div class="card border">
                <div class="card-header bg-transparent d-flex justify-content-between align-items-center py-2">
                    <span class="badge ${getTypeBadgeClass(rule.type)}">${getTypeText(rule.type)}</span>
                    <div class="form-check form-switch mb-0">
                        <input class="form-check-input" type="checkbox" ${rule.enabled ? 'checked' : ''} onchange="toggleRule(${rule.id}, this.checked)">
                    </div>
                </div>
                <div class="card-body py-2">
                    <h6 class="card-title mb-1">${escapeHtml(rule.name)}</h6>
                    <p class="card-text small text-muted mb-2">${escapeHtml(rule.condition)}</p>
                    <div class="d-flex justify-content-between align-items-center">
                        <span class="badge ${getActionBadgeClass(rule.action)}">${getActionText(rule.action)}</span>
                        <small class="text-muted">命中 ${rule.hitCount} 次</small>
                    </div>
                </div>
                <div class="card-footer bg-transparent py-2">
                    <div class="btn-group btn-group-sm w-100">
                        <button class="btn btn-outline-secondary" onclick="viewRuleDetail(${rule.id})"><i class="fas fa-eye me-1"></i>详情</button>
                        <button class="btn btn-outline-primary" onclick="editRule(${rule.id})"><i class="fas fa-edit me-1"></i>编辑</button>
                        <button class="btn btn-outline-danger" onclick="deleteRule(${rule.id})"><i class="fas fa-trash me-1"></i>删除</button>
                    </div>
                </div>
            </div>
        </div>
    `).join('')}</div>`;
}

function switchView(view) {
    currentView = view;
    renderRules();
}

function renderRulesCount(total) {
    const countEl = document.getElementById('rulesCount');
    if (countEl) countEl.textContent = total;
}

function renderRulesPagination(total) {
    const pagination = document.getElementById('rulesPagination');
    if (!pagination) return;

    const totalPages = Math.ceil(total / rulesPageSize);
    if (totalPages <= 1) {
        pagination.innerHTML = '';
        return;
    }

    let html = '<div class="d-flex justify-content-between align-items-center">';
    html += `<span class="text-muted">第 ${rulesPage} / ${totalPages} 页，共 ${total} 条</span>`;
    html += '<div class="btn-group btn-group-sm">';

    html += `<button class="btn btn-outline-secondary" onclick="changeRulesPage(${rulesPage - 1})" ${rulesPage === 1 ? 'disabled' : ''}>上一页</button>`;

    const startPage = Math.max(1, rulesPage - 2);
    const endPage = Math.min(totalPages, rulesPage + 2);

    if (startPage > 1) {
        html += `<button class="btn btn-outline-secondary" onclick="changeRulesPage(1)">1</button>`;
        if (startPage > 2) {
            html += `<button class="btn btn-outline-secondary" disabled>...</button>`;
        }
    }

    for (let i = startPage; i <= endPage; i++) {
        html += `<button class="btn ${i === rulesPage ? 'btn-primary' : 'btn-outline-secondary'}" onclick="changeRulesPage(${i})">${i}</button>`;
    }

    if (endPage < totalPages) {
        if (endPage < totalPages - 1) {
            html += `<button class="btn btn-outline-secondary" disabled>...</button>`;
        }
        html += `<button class="btn btn-outline-secondary" onclick="changeRulesPage(${totalPages})">${totalPages}</button>`;
    }

    html += `<button class="btn btn-outline-secondary" onclick="changeRulesPage(${rulesPage + 1})" ${rulesPage === totalPages ? 'disabled' : ''}>下一页</button>`;
    html += '</div></div>';

    pagination.innerHTML = html;
}

function changeRulesPage(page) {
    rulesPage = page;
    loadRiskRules();
}

function openRuleModal(rule = null) {
    const modal = document.getElementById('ruleModal');
    const title = document.getElementById('ruleModalTitle');
    const form = document.getElementById('ruleForm');

    if (rule) {
        title.textContent = '编辑规则';
        document.getElementById('ruleId').value = rule.id;
        document.getElementById('ruleName').value = rule.name;
        document.getElementById('ruleDescription').value = rule.description || '';
        document.getElementById('ruleType').value = rule.type;
        document.getElementById('ruleAction').value = rule.action;
        document.getElementById('rulePriority').value = rule.priority;
        document.getElementById('ruleEnabled').checked = rule.enabled;
    } else {
        title.textContent = '添加规则';
        form.reset();
        document.getElementById('ruleId').value = '';
        document.getElementById('ruleEnabled').checked = true;
    }

    const bsModal = new bootstrap.Modal(modal);
    bsModal.show();
}

async function handleRuleSubmit(e) {
    e.preventDefault();

    const ruleId = document.getElementById('ruleId').value;
    const ruleData = {
        name: document.getElementById('ruleName').value,
        description: document.getElementById('ruleDescription').value,
        type: document.getElementById('ruleType').value,
        condition: {
            field: document.getElementById('conditionField').value,
            operator: document.getElementById('conditionOperator').value,
            value: document.getElementById('conditionValue').value,
            timeWindow: parseInt(document.getElementById('timeWindow').value),
            triggerCount: parseInt(document.getElementById('triggerCount').value)
        },
        action: document.getElementById('ruleAction').value,
        priority: parseInt(document.getElementById('rulePriority').value),
        enabled: document.getElementById('ruleEnabled').checked,
        apps: Array.from(document.getElementById('ruleApps').selectedOptions).map(o => o.value)
    };

    try {
        if (ruleId) {
            await auth.request(`/admin/risk-rules/${ruleId}`, {
                method: 'PUT',
                body: JSON.stringify(ruleData)
            });
            showToast('规则更新成功', 'success');
        } else {
            await auth.request('/admin/risk-rules', {
                method: 'POST',
                body: JSON.stringify(ruleData)
            });
            showToast('规则创建成功', 'success');
        }

        bootstrap.Modal.getInstance(document.getElementById('ruleModal'))?.hide();
        loadRiskRules();
        loadRiskRulesSummary();
    } catch (error) {
        showToast('保存规则失败', 'danger');
    }
}

function editRule(ruleId) {
    const rule = currentRules.find(r => r.id === ruleId);
    if (rule) {
        openRuleModal(rule);
    }
}

async function deleteRule(ruleId) {
    if (!confirm('确定要删除这条规则吗？')) return;

    try {
        await auth.request(`/admin/risk-rules/${ruleId}`, {
            method: 'DELETE'
        });
        showToast('规则删除成功', 'success');
        loadRiskRules();
        loadRiskRulesSummary();
    } catch (error) {
        showToast('删除规则失败', 'danger');
    }
}

async function toggleRule(ruleId, enabled) {
    try {
        await auth.request(`/admin/risk-rules/${ruleId}/toggle`, {
            method: 'POST',
            body: JSON.stringify({ enabled })
        });
        showToast(`规则已${enabled ? '启用' : '禁用'}`, 'success');
        loadRiskRulesSummary();
    } catch (error) {
        loadRiskRules();
        showToast('切换状态失败', 'danger');
    }
}

function viewRuleDetail(ruleId) {
    const rule = currentRules.find(r => r.id === ruleId);
    if (!rule) return;

    const modal = document.getElementById('ruleDetailModal');
    const content = document.getElementById('ruleDetailContent');

    content.innerHTML = `
        <table class="table table-borderless">
            <tr><td class="text-muted" style="width: 120px;">规则名称</td><td><strong>${escapeHtml(rule.name)}</strong></td></tr>
            <tr><td class="text-muted">规则类型</td><td><span class="badge ${getTypeBadgeClass(rule.type)}">${getTypeText(rule.type)}</span></td></tr>
            <tr><td class="text-muted">规则描述</td><td>${rule.description ? escapeHtml(rule.description) : '-'}</td></tr>
            <tr><td class="text-muted">触发条件</td><td><code>${escapeHtml(rule.condition)}</code></td></tr>
            <tr><td class="text-muted">处置方式</td><td><span class="badge ${getActionBadgeClass(rule.action)}">${getActionText(rule.action)}</span></td></tr>
            <tr><td class="text-muted">优先级</td><td><span class="badge ${getPriorityBadgeClass(rule.priority)}">${getPriorityText(rule.priority)}</span></td></tr>
            <tr><td class="text-muted">状态</td><td>${rule.enabled ? '<span class="badge bg-success">启用</span>' : '<span class="badge bg-secondary">禁用</span>'}</td></tr>
            <tr><td class="text-muted">命中次数</td><td><span class="text-danger fw-bold">${rule.hitCount}</span> 次</td></tr>
        </table>
    `;

    const bsModal = new bootstrap.Modal(modal);
    bsModal.show();
}

function getTypeBadgeClass(type) {
    const map = {
        'rate_limit': 'bg-info',
        'ip_block': 'bg-danger',
        'behavior': 'bg-warning',
        'device_fingerprint': 'bg-primary',
        'custom': 'bg-secondary'
    };
    return map[type] || 'bg-secondary';
}

function getTypeText(type) {
    const map = {
        'rate_limit': '频率限制',
        'ip_block': 'IP封禁',
        'behavior': '行为分析',
        'device_fingerprint': '设备指纹',
        'custom': '自定义'
    };
    return map[type] || type;
}

function getActionBadgeClass(action) {
    const map = {
        'block': 'bg-danger',
        'captcha': 'bg-warning',
        'rate_limit': 'bg-info',
        'warning': 'bg-secondary',
        'review': 'bg-primary'
    };
    return map[action] || 'bg-secondary';
}

function getActionText(action) {
    const map = {
        'block': '直接拦截',
        'captcha': '验证码',
        'rate_limit': '限流',
        'warning': '警告',
        'review': '人工审核'
    };
    return map[action] || action;
}

function getPriorityBadgeClass(priority) {
    if (priority >= 100) return 'bg-danger';
    if (priority >= 10) return 'bg-warning';
    if (priority >= 5) return 'bg-info';
    return 'bg-secondary';
}

function getPriorityText(priority) {
    if (priority >= 100) return '紧急';
    if (priority >= 10) return '高';
    if (priority >= 5) return '中';
    return '低';
}

function showToast(message, type = 'info') {
    const toastContainer = document.getElementById('toastContainer') || createToastContainer();
    const toast = document.createElement('div');
    toast.className = `toast align-items-center text-white bg-${type} border-0`;
    toast.setAttribute('role', 'alert');
    toast.innerHTML = `
        <div class="d-flex">
            <div class="toast-body">${escapeHtml(message)}</div>
            <button type="button" class="btn-close btn-close-white me-2 m-auto" data-bs-dismiss="toast"></button>
        </div>
    `;
    toastContainer.appendChild(toast);
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
