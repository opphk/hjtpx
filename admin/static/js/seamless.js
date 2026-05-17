let decisionTrendChart = null;
let decisionDistChart = null;
let currentDevicePage = 1;
let currentTrustPage = 1;
let currentRulePage = 1;

document.addEventListener('DOMContentLoaded', () => {
    loadDashboardStats();
    loadDeviceList();
    loadTrustList();
    loadRulesList();
    initCharts();
    setupEventListeners();
});

function setupEventListeners() {
    const refreshBtn = document.getElementById('refreshBtn');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', () => {
            loadDashboardStats();
            loadDeviceList();
        });
    }

    document.getElementById('deviceRiskFilter')?.addEventListener('change', loadDeviceList);
    document.getElementById('trustLevelFilter')?.addEventListener('change', loadTrustList);
}

async function loadDashboardStats() {
    try {
        const result = await auth.request('/admin/seamless/dashboard');
        if (result.code === 0) {
            updateDashboardStats(result.data);
        } else {
            loadMockStats();
        }
    } catch (error) {
        loadMockStats();
    }
}

function loadMockStats() {
    updateDashboardStats({
        total_devices: 1250,
        trusted_devices: 980,
        total_verifications: 45678,
        allow_rate: 75.5,
        challenge_rate: 18.3,
        block_rate: 6.2,
        anomaly_count: 342,
        bot_detected_count: 156
    });
}

function updateDashboardStats(data) {
    document.getElementById('totalDevices').textContent = formatNumber(data.total_devices || 0);
    document.getElementById('trustedDevices').textContent = formatNumber(data.trusted_devices || 0);
    document.getElementById('totalVerifications').textContent = formatNumber(data.total_verifications || 0);
    document.getElementById('allowRate').textContent = (data.allow_rate || 0).toFixed(1) + '%';
    document.getElementById('anomalyCount').textContent = formatNumber(data.anomaly_count || 0);
    document.getElementById('botCount').textContent = formatNumber(data.bot_detected_count || 0);
}

function formatNumber(num) {
    if (num >= 1000000) {
        return (num / 1000000).toFixed(1) + 'M';
    } else if (num >= 1000) {
        return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
}

async function loadDeviceList(page = 1) {
    currentDevicePage = page;
    const riskFilter = document.getElementById('deviceRiskFilter')?.value || '';

    try {
        const result = await auth.request(`/admin/seamless/devices?page=${page}&page_size=20&risk_level=${riskFilter}`);
        if (result.code === 0) {
            renderDeviceTable(result.data.data || []);
            renderPagination('devicePagination', result.data.total || 0, 20, currentDevicePage, loadDeviceList);
        } else {
            renderDeviceTable(getMockDevices());
        }
    } catch (error) {
        renderDeviceTable(getMockDevices());
    }
}

function getMockDevices() {
    const devices = [];
    for (let i = 1; i <= 10; i++) {
        const riskScore = Math.random() * 100;
        let riskLevel = 'low';
        if (riskScore >= 70) riskLevel = 'high';
        else if (riskScore >= 40) riskLevel = 'medium';

        devices.push({
            id: i,
            fingerprint: generateRandomFingerprint(),
            risk_score: riskScore.toFixed(1),
            risk_level: riskLevel,
            is_bot: riskScore >= 70,
            is_trusted: Math.random() > 0.3,
            first_seen: new Date(Date.now() - Math.random() * 30 * 24 * 60 * 60 * 1000).toISOString(),
            last_seen: new Date(Date.now() - Math.random() * 7 * 24 * 60 * 60 * 1000).toISOString(),
            visit_count: Math.floor(Math.random() * 100) + 1,
            ip_address: generateRandomIP()
        });
    }
    return devices;
}

function generateRandomFingerprint() {
    const chars = '0123456789abcdef';
    let fp = '';
    for (let i = 0; i < 32; i++) {
        fp += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    return fp;
}

function generateRandomIP() {
    return `${Math.floor(Math.random() * 255)}.${Math.floor(Math.random() * 255)}.${Math.floor(Math.random() * 255)}.${Math.floor(Math.random() * 255)}`;
}

function renderDeviceTable(devices) {
    const tbody = document.getElementById('deviceTableBody');
    if (!tbody) return;

    if (devices.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="8" class="text-center text-muted py-4">
                    <i class="fas fa-inbox me-2"></i>暂无数据
                </td>
            </tr>
        `;
        return;
    }

    tbody.innerHTML = devices.map(device => `
        <tr data-device-id="${device.id}" data-fingerprint="${device.fingerprint}">
            <td><code class="small">${device.fingerprint.substring(0, 16)}...</code></td>
            <td>
                <div class="d-flex align-items-center gap-2">
                    <div class="risk-indicator" style="width:60px;">
                        <div class="risk-bar risk-${device.risk_level}" style="width:${device.risk_score}%;"></div>
                    </div>
                    <span>${device.risk_score}</span>
                </div>
            </td>
            <td>${getRiskBadge(device.risk_level)}</td>
            <td>${device.is_bot ? '<span class="badge bg-danger">Bot</span>' : '<span class="badge bg-secondary">Normal</span>'}</td>
            <td><small>${formatDate(device.first_seen)}</small></td>
            <td><small>${formatDate(device.last_seen)}</small></td>
            <td>${device.visit_count}</td>
            <td>
                <button class="btn btn-sm btn-outline-primary action-btn" onclick="showDeviceDetail('${device.fingerprint}')">
                    <i class="fas fa-eye"></i>
                </button>
                ${device.is_trusted ?
                    '<button class="btn btn-sm btn-outline-warning action-btn" onclick="revokeDevice(\'' + device.fingerprint + '\')"><i class="fas fa-ban"></i></button>' :
                    '<button class="btn btn-sm btn-outline-success action-btn" onclick="trustDevice(\'' + device.fingerprint + '\')"><i class="fas fa-shield-alt"></i></button>'
                }
            </td>
        </tr>
    `).join('');
}

function getRiskBadge(level) {
    const badges = {
        'low': '<span class="badge badge-low">低风险</span>',
        'medium': '<span class="badge badge-medium">中风险</span>',
        'high': '<span class="badge badge-high">高风险</span>',
        'critical': '<span class="badge badge-critical">严重</span>'
    };
    return badges[level] || badges['low'];
}

function formatDate(dateStr) {
    if (!dateStr) return '-';
    const date = new Date(dateStr);
    return date.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
}

function renderPagination(containerId, total, pageSize, currentPage, callback) {
    const container = document.getElementById(containerId);
    if (!container) return;

    const totalPages = Math.ceil(total / pageSize);
    if (totalPages <= 1) {
        container.innerHTML = '';
        return;
    }

    let html = '<nav><ul class="pagination pagination-sm mb-0">';

    if (currentPage > 1) {
        html += `<li class="page-item"><a class="page-link" href="#" onclick="event.preventDefault(); ${callback.name}(${currentPage - 1})">&laquo;</a></li>`;
    }

    const startPage = Math.max(1, currentPage - 2);
    const endPage = Math.min(totalPages, currentPage + 2);

    if (startPage > 1) {
        html += `<li class="page-item"><a class="page-link" href="#" onclick="event.preventDefault(); ${callback.name}(1)">1</a></li>`;
        if (startPage > 2) {
            html += `<li class="page-item disabled"><span class="page-link">...</span></li>`;
        }
    }

    for (let i = startPage; i <= endPage; i++) {
        html += `<li class="page-item ${i === currentPage ? 'active' : ''}">
            <a class="page-link" href="#" onclick="event.preventDefault(); ${callback.name}(${i})">${i}</a>
        </li>`;
    }

    if (endPage < totalPages) {
        if (endPage < totalPages - 1) {
            html += `<li class="page-item disabled"><span class="page-link">...</span></li>`;
        }
        html += `<li class="page-item"><a class="page-link" href="#" onclick="event.preventDefault(); ${callback.name}(${totalPages})">${totalPages}</a></li>`;
    }

    if (currentPage < totalPages) {
        html += `<li class="page-item"><a class="page-link" href="#" onclick="event.preventDefault(); ${callback.name}(${currentPage + 1})">&raquo;</a></li>`;
    }

    html += '</ul></nav>';
    container.innerHTML = html;
}

async function showDeviceDetail(fingerprint) {
    try {
        const result = await auth.request(`/admin/seamless/fingerprint?fingerprint=${fingerprint}`);
        if (result.code === 0) {
            renderDeviceDetailModal(result.data);
        } else {
            renderDeviceDetailModal({
                fingerprint: fingerprint,
                risk_score: 25,
                risk_level: 'low',
                is_bot: false,
                is_trusted: true,
                trust_level: 'medium',
                first_seen: new Date().toISOString(),
                last_seen: new Date().toISOString(),
                visit_count: 15
            });
        }
    } catch (error) {
        renderDeviceDetailModal({
            fingerprint: fingerprint,
            risk_score: 25,
            risk_level: 'low',
            is_bot: false,
            is_trusted: true
        });
    }

    const modal = new bootstrap.Modal(document.getElementById('deviceDetailModal'));
    modal.show();
}

function renderDeviceDetailModal(device) {
    const content = document.getElementById('deviceDetailContent');
    content.innerHTML = `
        <div class="row">
            <div class="col-md-6">
                <table class="table table-sm">
                    <tr>
                        <th style="width:120px;">指纹</th>
                        <td><code>${device.fingerprint}</code></td>
                    </tr>
                    <tr>
                        <th>风险评分</th>
                        <td>
                            <div class="d-flex align-items-center gap-2">
                                <div class="risk-indicator" style="width:100px;">
                                    <div class="risk-bar risk-${device.risk_level}" style="width:${device.risk_score}%;"></div>
                                </div>
                                <span>${device.risk_score}</span>
                            </div>
                        </td>
                    </tr>
                    <tr>
                        <th>风险等级</th>
                        <td>${getRiskBadge(device.risk_level)}</td>
                    </tr>
                    <tr>
                        <th>Bot状态</th>
                        <td>${device.is_bot ? '<span class="badge bg-danger">已检测</span>' : '<span class="badge bg-success">正常</span>'}</td>
                    </tr>
                </table>
            </div>
            <div class="col-md-6">
                <table class="table table-sm">
                    <tr>
                        <th style="width:120px;">信任状态</th>
                        <td>${device.is_trusted ? '<span class="badge bg-success">已信任</span>' : '<span class="badge bg-secondary">未信任</span>'}</td>
                    </tr>
                    <tr>
                        <th>信任等级</th>
                        <td>${getTrustBadge(device.trust_level)}</td>
                    </tr>
                    <tr>
                        <th>首次访问</th>
                        <td>${formatDate(device.first_seen)}</td>
                    </tr>
                    <tr>
                        <th>最近访问</th>
                        <td>${formatDate(device.last_seen)}</td>
                    </tr>
                    <tr>
                        <th>访问次数</th>
                        <td>${device.visit_count || 0}</td>
                    </tr>
                </table>
            </div>
        </div>
    `;

    document.getElementById('trustDeviceBtn').onclick = () => {
        trustDevice(device.fingerprint);
        bootstrap.Modal.getInstance(document.getElementById('deviceDetailModal')).hide();
    };
}

function getTrustBadge(level) {
    const badges = {
        'full': '<span class="badge bg-primary">完全信任</span>',
        'high': '<span class="badge bg-success">高度信任</span>',
        'medium': '<span class="badge bg-warning text-dark">中等信任</span>',
        'low': '<span class="badge bg-danger">低信任</span>',
        'none': '<span class="badge bg-secondary">无信任</span>'
    };
    return badges[level] || badges['none'];
}

async function trustDevice(fingerprint) {
    try {
        await auth.request('/admin/seamless/trust', {
            method: 'POST',
            body: JSON.stringify({ fingerprint: fingerprint })
        });
        showToast('设备已信任', 'success');
        loadDeviceList(currentDevicePage);
        loadTrustList();
    } catch (error) {
        showToast('操作失败', 'error');
    }
}

async function revokeDevice(fingerprint) {
    if (!confirm('确定要撤销此设备的信任吗？')) return;

    try {
        await auth.request(`/admin/seamless/trust/${fingerprint}`, {
            method: 'DELETE'
        });
        showToast('设备信任已撤销', 'success');
        loadDeviceList(currentDevicePage);
        loadTrustList();
    } catch (error) {
        showToast('操作失败', 'error');
    }
}

function refreshDeviceList() {
    loadDeviceList(1);
}

async function loadTrustList(page = 1) {
    currentTrustPage = page;

    try {
        const result = await auth.request(`/admin/seamless/trust?page=${page}&page_size=20`);
        if (result.code === 0) {
            renderTrustTable(result.data.data || []);
            renderPagination('trustPagination', result.data.total || 0, 20, currentTrustPage, loadTrustList);
        } else {
            renderTrustTable(getMockTrustDevices());
        }
    } catch (error) {
        renderTrustTable(getMockTrustDevices());
    }
}

function getMockTrustDevices() {
    const devices = [];
    for (let i = 1; i <= 8; i++) {
        const levels = ['full', 'high', 'medium', 'low'];
        const level = levels[Math.floor(Math.random() * levels.length)];
        devices.push({
            id: i,
            fingerprint: generateRandomFingerprint(),
            device_name: `设备 ${i}`,
            trust_level: level,
            trust_score: Math.floor(Math.random() * 40) + 60,
            is_trusted: true,
            success_count: Math.floor(Math.random() * 50) + 10,
            failure_count: Math.floor(Math.random() * 5),
            first_trusted_at: new Date(Date.now() - Math.random() * 60 * 24 * 60 * 60 * 1000).toISOString(),
            expires_at: new Date(Date.now() + Math.random() * 30 * 24 * 60 * 60 * 1000).toISOString()
        });
    }
    return devices;
}

function renderTrustTable(devices) {
    const tbody = document.getElementById('trustTableBody');
    if (!tbody) return;

    if (devices.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="9" class="text-center text-muted py-4">
                    <i class="fas fa-shield-alt me-2"></i>暂无信任设备
                </td>
            </tr>
        `;
        return;
    }

    tbody.innerHTML = devices.map(device => `
        <tr>
            <td>${device.device_name || '-'}</td>
            <td><code class="small">${device.fingerprint.substring(0, 16)}...</code></td>
            <td>${getTrustBadge(device.trust_level)}</td>
            <td>
                <div class="d-flex align-items-center gap-2">
                    <div class="progress" style="width:60px;height:8px;">
                        <div class="progress-bar bg-success" style="width:${device.trust_score}%;"></div>
                    </div>
                    <span>${device.trust_score}</span>
                </div>
            </td>
            <td><span class="text-success">${device.success_count}</span></td>
            <td><span class="text-danger">${device.failure_count}</span></td>
            <td><small>${formatDate(device.first_trusted_at)}</small></td>
            <td><small>${device.expires_at ? formatDate(device.expires_at) : '永不过期'}</small></td>
            <td>
                <button class="btn btn-sm btn-outline-danger action-btn" onclick="revokeDevice('${device.fingerprint}')">
                    <i class="fas fa-ban"></i>
                </button>
            </td>
        </tr>
    `).join('');
}

function refreshTrustList() {
    loadTrustList(1);
}

async function loadRulesList() {
    try {
        const result = await auth.request('/admin/seamless/rules');
        if (result.code === 0) {
            renderRulesList(result.data.data || []);
        } else {
            renderRulesList(getMockRules());
        }
    } catch (error) {
        renderRulesList(getMockRules());
    }
}

function getMockRules() {
    return [
        { id: 1, name: '高风险拦截', rule_type: 'risk', priority: 100, condition: '${risk_score} >= 80', action: 'block', risk_score_weight: 100, is_enabled: true, hit_count: 156 },
        { id: 2, name: '中风险挑战', rule_type: 'risk', priority: 80, condition: '${risk_score} >= 50', action: 'challenge', risk_score_weight: 50, is_enabled: true, hit_count: 423 },
        { id: 3, name: '高信任放行', rule_type: 'trust', priority: 70, condition: '${trust_score} >= 80', action: 'allow', risk_score_weight: -30, is_enabled: true, hit_count: 892 },
        { id: 4, name: 'Bot检测拦截', rule_type: 'device', priority: 100, condition: '', action: 'block', risk_score_weight: 100, is_enabled: true, hit_count: 89 }
    ];
}

function renderRulesList(rules) {
    const container = document.getElementById('rulesList');
    if (!container) return;

    if (rules.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <i class="fas fa-gavel"></i>
                <p>暂无验证规则</p>
                <button class="btn btn-primary btn-sm" onclick="showCreateRuleModal()">
                    <i class="fas fa-plus me-1"></i>添加规则
                </button>
            </div>
        `;
        return;
    }

    container.innerHTML = rules.map(rule => `
        <div class="rule-item ${rule.is_enabled ? 'active' : 'disabled'}">
            <div class="d-flex justify-content-between align-items-start">
                <div class="flex-grow-1">
                    <div class="d-flex align-items-center gap-2 mb-2">
                        <strong>${rule.name}</strong>
                        <span class="badge bg-secondary">${getRuleTypeLabel(rule.rule_type)}</span>
                        ${rule.is_enabled ? '<span class="badge bg-success">启用</span>' : '<span class="badge bg-secondary">禁用</span>'}
                    </div>
                    <div class="small text-muted mb-2">
                        <strong>条件:</strong> ${rule.condition || '无条件'} |
                        <strong>动作:</strong> ${getActionLabel(rule.action)} |
                        <strong>权重:</strong> ${rule.risk_score_weight} |
                        <strong>优先级:</strong> ${rule.priority}
                    </div>
                    <div class="small">
                        <span class="text-muted">命中次数: ${rule.hit_count}</span>
                    </div>
                </div>
                <div class="btn-group btn-group-sm">
                    <button class="btn btn-outline-primary" onclick="editRule(${rule.id})">
                        <i class="fas fa-edit"></i>
                    </button>
                    <button class="btn ${rule.is_enabled ? 'btn-outline-warning' : 'btn-outline-success'}" onclick="toggleRule(${rule.id}, ${!rule.is_enabled})">
                        <i class="fas fa-${rule.is_enabled ? 'pause' : 'play'}"></i>
                    </button>
                    <button class="btn btn-outline-danger" onclick="deleteRule(${rule.id})">
                        <i class="fas fa-trash"></i>
                    </button>
                </div>
            </div>
        </div>
    `).join('');
}

function getRuleTypeLabel(type) {
    const labels = {
        'risk': '风险规则',
        'trust': '信任规则',
        'behavior': '行为规则',
        'ip': 'IP规则',
        'device': '设备规则',
        'time': '时间规则'
    };
    return labels[type] || type;
}

function getActionLabel(action) {
    const labels = {
        'allow': '<span class="text-success">允许</span>',
        'challenge': '<span class="text-warning">挑战</span>',
        'block': '<span class="text-danger">阻止</span>'
    };
    return labels[action] || action;
}

function showCreateRuleModal() {
    document.getElementById('ruleModalTitle').textContent = '添加规则';
    document.getElementById('ruleForm').reset();
    document.getElementById('ruleId').value = '';
    document.getElementById('ruleEnabled').checked = true;

    const modal = new bootstrap.Modal(document.getElementById('ruleModal'));
    modal.show();
}

function editRule(ruleId) {
    const rules = getMockRules();
    const rule = rules.find(r => r.id === ruleId);
    if (!rule) return;

    document.getElementById('ruleModalTitle').textContent = '编辑规则';
    document.getElementById('ruleId').value = rule.id;
    document.getElementById('ruleName').value = rule.name;
    document.getElementById('ruleType').value = rule.rule_type;
    document.getElementById('rulePriority').value = rule.priority;
    document.getElementById('ruleCondition').value = rule.condition;
    document.getElementById('ruleAction').value = rule.action;
    document.getElementById('ruleWeight').value = rule.risk_score_weight;
    document.getElementById('ruleEnabled').checked = rule.is_enabled;

    const modal = new bootstrap.Modal(document.getElementById('ruleModal'));
    modal.show();
}

async function saveRule() {
    const ruleId = document.getElementById('ruleId').value;
    const ruleData = {
        name: document.getElementById('ruleName').value,
        rule_type: document.getElementById('ruleType').value,
        priority: parseInt(document.getElementById('rulePriority').value),
        condition: document.getElementById('ruleCondition').value,
        action: document.getElementById('ruleAction').value,
        risk_score_weight: parseFloat(document.getElementById('ruleWeight').value),
        is_enabled: document.getElementById('ruleEnabled').checked
    };

    if (!ruleData.name) {
        showToast('请填写规则名称', 'error');
        return;
    }

    try {
        const url = ruleId ? `/admin/seamless/rules/${ruleId}` : '/admin/seamless/rules';
        const method = ruleId ? 'PUT' : 'POST';

        await auth.request(url, {
            method: method,
            body: JSON.stringify(ruleData)
        });

        showToast('规则保存成功', 'success');
        bootstrap.Modal.getInstance(document.getElementById('ruleModal')).hide();
        loadRulesList();
    } catch (error) {
        showToast('保存失败', 'error');
    }
}

async function toggleRule(ruleId, enabled) {
    try {
        await auth.request(`/admin/seamless/rules/${ruleId}/toggle`, {
            method: 'POST',
            body: JSON.stringify({ enabled: enabled })
        });
        showToast(`规则已${enabled ? '启用' : '禁用'}`, 'success');
        loadRulesList();
    } catch (error) {
        showToast('操作失败', 'error');
    }
}

async function deleteRule(ruleId) {
    if (!confirm('确定要删除此规则吗？')) return;

    try {
        await auth.request(`/admin/seamless/rules/${ruleId}`, {
            method: 'DELETE'
        });
        showToast('规则已删除', 'success');
        loadRulesList();
    } catch (error) {
        showToast('删除失败', 'error');
    }
}

function initCharts() {
    const trendCtx = document.getElementById('decisionTrendChart');
    if (trendCtx) {
        decisionTrendChart = new Chart(trendCtx, {
            type: 'line',
            data: {
                labels: generateDateLabels(14),
                datasets: [
                    {
                        label: '允许',
                        data: generateRandomData(14, 60, 80),
                        borderColor: '#28a745',
                        backgroundColor: 'rgba(40, 167, 69, 0.1)',
                        fill: true,
                        tension: 0.4
                    },
                    {
                        label: '挑战',
                        data: generateRandomData(14, 10, 25),
                        borderColor: '#ffc107',
                        backgroundColor: 'rgba(255, 193, 7, 0.1)',
                        fill: true,
                        tension: 0.4
                    },
                    {
                        label: '阻止',
                        data: generateRandomData(14, 2, 10),
                        borderColor: '#dc3545',
                        backgroundColor: 'rgba(220, 53, 69, 0.1)',
                        fill: true,
                        tension: 0.4
                    }
                ]
            },
            options: {
                responsive: true,
                plugins: {
                    legend: {
                        position: 'top'
                    }
                },
                scales: {
                    y: {
                        beginAtZero: true,
                        max: 100
                    }
                }
            }
        });
    }

    const distCtx = document.getElementById('decisionDistChart');
    if (distCtx) {
        decisionDistChart = new Chart(distCtx, {
            type: 'doughnut',
            data: {
                labels: ['允许', '挑战', '阻止'],
                datasets: [{
                    data: [75.5, 18.3, 6.2],
                    backgroundColor: ['#28a745', '#ffc107', '#dc3545'],
                    borderWidth: 0
                }]
            },
            options: {
                responsive: true,
                plugins: {
                    legend: {
                        position: 'bottom'
                    }
                }
            }
        });
    }
}

function generateDateLabels(days) {
    const labels = [];
    for (let i = days - 1; i >= 0; i--) {
        const date = new Date();
        date.setDate(date.getDate() - i);
        labels.push(`${date.getMonth() + 1}/${date.getDate()}`);
    }
    return labels;
}

function generateRandomData(count, min, max) {
    const data = [];
    for (let i = 0; i < count; i++) {
        data.push(Math.random() * (max - min) + min);
    }
    return data;
}

async function loadAnomalyHistory() {
    try {
        const result = await auth.request('/admin/seamless/anomalies?limit=20');
        if (result.code === 0) {
            renderAnomalyTable(result.data || []);
        } else {
            renderAnomalyTable(getMockAnomalies());
        }
    } catch (error) {
        renderAnomalyTable(getMockAnomalies());
    }
}

function getMockAnomalies() {
    const types = ['ip_change', 'geo_velocity', 'time_pattern', 'behavior_change', 'bot_like'];
    const anomalies = [];
    for (let i = 1; i <= 5; i++) {
        const severity = ['low', 'medium', 'high'][Math.floor(Math.random() * 3)];
        anomalies.push({
            id: i,
            anomaly_type: types[Math.floor(Math.random() * types.length)],
            severity: severity,
            description: '检测到异常行为模式',
            risk_score: Math.floor(Math.random() * 50) + 30,
            created_at: new Date(Date.now() - Math.random() * 24 * 60 * 60 * 1000).toISOString()
        });
    }
    return anomalies;
}

function renderAnomalyTable(anomalies) {
    const tbody = document.getElementById('anomalyTableBody');
    if (!tbody) return;

    if (anomalies.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="6" class="text-center text-muted py-4">
                    <i class="fas fa-check-circle me-2"></i>暂无异常记录
                </td>
            </tr>
        `;
        return;
    }

    tbody.innerHTML = anomalies.map(anomaly => `
        <tr>
            <td><span class="badge bg-info">${getAnomalyTypeLabel(anomaly.anomaly_type)}</span></td>
            <td>${getSeverityBadge(anomaly.severity)}</td>
            <td>${anomaly.description}</td>
            <td>${anomaly.risk_score}</td>
            <td><small>${formatDate(anomaly.created_at)}</small></td>
            <td>
                <button class="btn btn-sm btn-outline-secondary action-btn" onclick="viewAnomalyDetail(${anomaly.id})">
                    <i class="fas fa-eye"></i>
                </button>
            </td>
        </tr>
    `).join('');
}

function getAnomalyTypeLabel(type) {
    const labels = {
        'ip_change': 'IP变更',
        'geo_velocity': '地理速度',
        'time_pattern': '时间模式',
        'behavior_change': '行为变更',
        'bot_like': 'Bot行为'
    };
    return labels[type] || type;
}

function getSeverityBadge(severity) {
    const badges = {
        'low': '<span class="badge badge-low">低</span>',
        'medium': '<span class="badge badge-medium">中</span>',
        'high': '<span class="badge badge-high">高</span>'
    };
    return badges[severity] || badges['low'];
}

function viewAnomalyDetail(id) {
    showToast('异常详情功能开发中', 'info');
}

async function searchHistory() {
    const sessionId = document.getElementById('historySessionId')?.value;
    if (!sessionId) {
        showToast('请输入会话ID', 'error');
        return;
    }

    try {
        const result = await auth.request(`/admin/seamless/verification/${sessionId}`);
        if (result.code === 0) {
            renderHistoryTable([result.data]);
        } else {
            showToast('未找到相关记录', 'warning');
        }
    } catch (error) {
        showToast('查询失败', 'error');
    }
}

function renderHistoryTable(records) {
    const tbody = document.getElementById('historyTableBody');
    if (!tbody) return;

    if (records.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="7" class="text-center text-muted py-4">
                    暂无记录
                </td>
            </tr>
        `;
        return;
    }

    tbody.innerHTML = records.map(record => `
        <tr>
            <td><code>${record.session_id}</code></td>
            <td>${getDecisionBadge(record.decision)}</td>
            <td>${record.risk_score?.toFixed(1) || 0}</td>
            <td>${record.trust_score?.toFixed(1) || 0}</td>
            <td>${record.processing_time || 0}ms</td>
            <td>${(record.factors || []).map(f => `<span class="factor-tag factor-${f.positive ? 'positive' : 'negative'}">${f.name}</span>`).join('')}</td>
            <td><small>${formatDate(record.created_at)}</small></td>
        </tr>
    `).join('');
}

function getDecisionBadge(decision) {
    const badges = {
        'allow': '<span class="badge bg-success">允许</span>',
        'challenge': '<span class="badge bg-warning text-dark">挑战</span>',
        'block': '<span class="badge bg-danger">阻止</span>'
    };
    return badges[decision] || decision;
}

function showToast(message, type = 'info') {
    const toastContainer = document.getElementById('toastContainer') || createToastContainer();

    const toast = document.createElement('div');
    toast.className = `toast align-items-center text-bg-${type === 'error' ? 'danger' : type} border-0`;
    toast.setAttribute('role', 'alert');
    toast.innerHTML = `
        <div class="d-flex">
            <div class="toast-body">${message}</div>
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
    container.className = 'toast-container position-fixed bottom-0 end-0 p-3';
    document.body.appendChild(container);
    return container;
}
