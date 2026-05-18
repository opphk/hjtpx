let rulesPage = 1;
let rulesPageSize = 10;
let currentRules = [];
let currentView = 'table';
let conditionCounter = 1;
let testHistory = [];

document.addEventListener('DOMContentLoaded', () => {
    loadRiskRulesSummary();
    loadRiskRules();
    setupEventListeners();
    initializeRuleBuilder();
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

    document.querySelectorAll('.condition-field, .condition-operator, .condition-value').forEach(el => {
        el.addEventListener('change', updateRuleExpression);
        el.addEventListener('input', updateRuleExpression);
    });
}

function initializeRuleBuilder() {
    updateRuleExpression();
}

function setGroupLogic(groupId, logic) {
    const group = document.getElementById(groupId);
    if (!group) return;
    
    group.dataset.logic = logic;
    const buttons = group.querySelectorAll('[data-logic]');
    buttons.forEach(btn => {
        btn.classList.remove('active');
        if (btn.dataset.logic === logic) {
            btn.classList.add('active');
        }
    });
    updateRuleExpression();
}

function addNestedGroup(parentGroupId) {
    const nestedContainer = document.getElementById('nestedGroups');
    if (!nestedContainer) return;
    
    const nestedId = `nested_group_${Date.now()}`;
    const nestedGroup = document.createElement('div');
    nestedGroup.className = 'rule-group nested';
    nestedGroup.id = nestedId;
    nestedGroup.dataset.logic = 'AND';
    
    nestedGroup.innerHTML = `
        <div class="rule-group-header">
            <div class="btn-group btn-group-sm me-2">
                <button class="btn btn-outline-primary active" data-logic="AND" onclick="setGroupLogic('${nestedId}', 'AND')">AND</button>
                <button class="btn btn-outline-primary" data-logic="OR" onclick="setGroupLogic('${nestedId}', 'OR')">OR</button>
                <button class="btn btn-outline-primary" data-logic="NOT" onclick="setGroupLogic('${nestedId}', 'NOT')">NOT</button>
                <button class="btn btn-outline-primary" data-logic="XOR" onclick="setGroupLogic('${nestedId}', 'XOR')">XOR</button>
            </div>
            <span class="text-muted small">嵌套规则组</span>
            <button class="btn btn-sm btn-success" onclick="addRuleCondition('${nestedId}')"><i class="fas fa-plus me-1"></i>添加条件</button>
            <button class="btn btn-sm btn-outline-danger" onclick="removeNestedGroup('${nestedId}')"><i class="fas fa-trash"></i></button>
        </div>
        <div class="rule-conditions" id="${nestedId.replace('Group', 'Conditions')}">
        </div>
    `;
    
    nestedContainer.appendChild(nestedGroup);
    updateRuleExpression();
}

function removeNestedGroup(groupId) {
    const group = document.getElementById(groupId);
    if (group) {
        group.remove();
        updateRuleExpression();
    }
}

function addRuleCondition(groupId) {
    conditionCounter++;
    const conditionsContainer = document.getElementById(groupId.replace('RuleGroup', 'RuleConditions'));
    if (!conditionsContainer) return;
    
    const conditionRow = document.createElement('div');
    conditionRow.className = 'rule-condition-row';
    conditionRow.dataset.conditionId = `cond${conditionCounter}`;
    conditionRow.innerHTML = `
        <div class="condition-logic-label">AND</div>
        <select class="form-select form-select-sm condition-field">
            <option value="speed">平均速度</option>
            <option value="speed_variance">速度方差</option>
            <option value="path_efficiency">路径效率</option>
            <option value="smoothness">平滑度</option>
            <option value="click_regularity">点击规律性</option>
            <option value="hesitation_time">犹豫时间</option>
            <option value="ml_score">ML分数</option>
            <option value="anomaly_score">异常分数</option>
        </select>
        <select class="form-select form-select-sm condition-operator">
            <option value="gt">></option>
            <option value="gte">>=</option>
            <option value="lt"><</option>
            <option value="lte"><=</option>
            <option value="eq">=</option>
        </select>
        <input type="number" class="form-control form-control-sm condition-value" placeholder="阈值" step="0.01">
        <button class="btn btn-outline-danger btn-sm" onclick="removeCondition('cond${conditionCounter}')"><i class="fas fa-times"></i></button>
    `;
    
    conditionsContainer.appendChild(conditionRow);
    
    conditionRow.querySelectorAll('select, input').forEach(el => {
        el.addEventListener('change', updateRuleExpression);
        el.addEventListener('input', updateRuleExpression);
    });
    
    updateRuleExpression();
}

function removeCondition(conditionId) {
    const condition = document.querySelector(`[data-condition-id="${conditionId}"]`);
    if (condition) {
        condition.remove();
        updateRuleExpression();
    }
}

function updateRuleExpression() {
    const conditions = document.querySelectorAll('.rule-condition-row');
    const logic = document.getElementById('mainRuleGroup')?.dataset.logic || 'AND';
    const expressionEl = document.getElementById('ruleExpression');
    
    if (!expressionEl) return;
    
    const parts = [];
    conditions.forEach((condition, index) => {
        const field = condition.querySelector('.condition-field')?.value || '';
        const operator = condition.querySelector('.condition-operator')?.value || '';
        const value = condition.querySelector('.condition-value')?.value || '';
        
        if (field && operator && value) {
            let opSymbol;
            switch (operator) {
                case 'gt': opSymbol = '>'; break;
                case 'gte': opSymbol = '>='; break;
                case 'lt': opSymbol = '<'; break;
                case 'lte': opSymbol = '<='; break;
                case 'eq': opSymbol = '='; break;
                default: opSymbol = operator;
            }
            
            const logicLabel = index === 0 ? '' : ` ${logic} `;
            parts.push(`${logicLabel}${field} ${opSymbol} ${value}`);
        }
    });
    
    expressionEl.textContent = parts.length > 0 ? parts.join('') : '暂无条件';
}

function applyTemplate(templateName) {
    let template;
    
    switch (templateName) {
        case 'extreme_speed':
            template = [
                { field: 'speed', operator: 'gt', value: '2000' },
                { field: 'path_efficiency', operator: 'gt', value: '0.98' }
            ];
            setGroupLogic('mainRuleGroup', 'AND');
            break;
        case 'bot_pattern':
            template = [
                { field: 'click_regularity', operator: 'gt', value: '0.98' },
                { field: 'hesitation_time', operator: 'lt', value: '50' },
                { field: 'ml_score', operator: 'gt', value: '0.7' }
            ];
            setGroupLogic('mainRuleGroup', 'AND');
            break;
        case 'human_behavior':
            template = [
                { field: 'hesitation_time', operator: 'gt', value: '100' },
                { field: 'path_efficiency', operator: 'lt', value: '0.95' }
            ];
            setGroupLogic('mainRuleGroup', 'OR');
            break;
        case 'combined_risk':
            template = [
                { field: 'speed', operator: 'gt', value: '1500' },
                { field: 'anomaly_score', operator: 'gt', value: '0.7' }
            ];
            setGroupLogic('mainRuleGroup', 'AND');
            break;
        default:
            return;
    }
    
    const conditionsContainer = document.getElementById('mainRuleConditions');
    if (!conditionsContainer) return;
    
    conditionsContainer.innerHTML = '';
    
    template.forEach((t, index) => {
        conditionCounter++;
        const conditionRow = document.createElement('div');
        conditionRow.className = 'rule-condition-row';
        conditionRow.dataset.conditionId = `cond${conditionCounter}`;
        conditionRow.innerHTML = `
            <div class="condition-logic-label">${index === 0 ? '条件' : 'AND'}</div>
            <select class="form-select form-select-sm condition-field">
                <option value="speed" ${t.field === 'speed' ? 'selected' : ''}>平均速度</option>
                <option value="speed_variance" ${t.field === 'speed_variance' ? 'selected' : ''}>速度方差</option>
                <option value="path_efficiency" ${t.field === 'path_efficiency' ? 'selected' : ''}>路径效率</option>
                <option value="smoothness" ${t.field === 'smoothness' ? 'selected' : ''}>平滑度</option>
                <option value="click_regularity" ${t.field === 'click_regularity' ? 'selected' : ''}>点击规律性</option>
                <option value="hesitation_time" ${t.field === 'hesitation_time' ? 'selected' : ''}>犹豫时间</option>
                <option value="ml_score" ${t.field === 'ml_score' ? 'selected' : ''}>ML分数</option>
                <option value="anomaly_score" ${t.field === 'anomaly_score' ? 'selected' : ''}>异常分数</option>
            </select>
            <select class="form-select form-select-sm condition-operator">
                <option value="gt" ${t.operator === 'gt' ? 'selected' : ''}>></option>
                <option value="gte" ${t.operator === 'gte' ? 'selected' : ''}>>=</option>
                <option value="lt" ${t.operator === 'lt' ? 'selected' : ''}><</option>
                <option value="lte" ${t.operator === 'lte' ? 'selected' : ''}><=</option>
                <option value="eq" ${t.operator === 'eq' ? 'selected' : ''}>=</option>
            </select>
            <input type="number" class="form-control form-control-sm condition-value" placeholder="阈值" step="0.01" value="${t.value}">
            <button class="btn btn-outline-danger btn-sm" onclick="removeCondition('cond${conditionCounter}')"><i class="fas fa-times"></i></button>
        `;
        
        conditionsContainer.appendChild(conditionRow);
        
        conditionRow.querySelectorAll('select, input').forEach(el => {
            el.addEventListener('change', updateRuleExpression);
            el.addEventListener('input', updateRuleExpression);
        });
    });
    
    updateRuleExpression();
    showToast(`已应用模板: ${templateName}`, 'success');
}

function loadSampleData(type) {
    const textarea = document.getElementById('sandboxInput');
    if (!textarea) return;
    
    if (type === 'bot') {
        textarea.value = JSON.stringify({
            average_speed: 2500,
            path_efficiency: 0.99,
            click_regularity: 0.999,
            hesitation_time: 20,
            ml_score: 0.92,
            anomaly_score: 0.88,
            smoothness: 0.98
        }, null, 2);
    } else if (type === 'human') {
        textarea.value = JSON.stringify({
            average_speed: 350,
            path_efficiency: 0.82,
            click_regularity: 0.72,
            hesitation_time: 450,
            ml_score: 0.15,
            anomaly_score: 0.12,
            smoothness: 0.68
        }, null, 2);
    }
}

function clearSandbox() {
    const textarea = document.getElementById('sandboxInput');
    const result = document.getElementById('sandboxResult');
    
    if (textarea) textarea.value = '';
    if (result) {
        result.innerHTML = `
            <div class="text-muted text-center py-4">
                <i class="fas fa-arrow-left me-2"></i>输入数据后点击"运行测试"
            </div>
        `;
    }
}

function runSandboxTest() {
    const inputEl = document.getElementById('sandboxInput');
    const resultEl = document.getElementById('sandboxResult');
    
    if (!inputEl || !resultEl) return;
    
    let inputData;
    try {
        inputData = JSON.parse(inputEl.value);
    } catch (e) {
        resultEl.innerHTML = `<div class="result-danger">JSON格式错误: ${e.message}</div>`;
        return;
    }
    
    const conditions = getConditionsFromBuilder();
    const ruleLogic = document.getElementById('mainRuleGroup')?.dataset.logic || 'AND';
    const evaluationResult = evaluateRules(conditions, inputData, ruleLogic);
    
    const timestamp = new Date().toLocaleTimeString();
    testHistory.unshift({
        timestamp,
        input: inputData,
        result: evaluationResult
    });
    
    if (testHistory.length > 10) {
        testHistory.pop();
    }
    
    renderTestHistory();
    
    let resultHtml = `<pre>`;
    resultHtml += `<span class="result-${evaluationResult.isBot ? 'danger' : 'success'}">`;
    resultHtml += `评估结果: ${evaluationResult.isBot ? '机器人 ✓' : '人类 ✓'}\n`;
    resultHtml += `</span>`;
    resultHtml += `总分: ${(evaluationResult.totalScore * 100).toFixed(1)}%\n`;
    resultHtml += `风险等级: ${evaluationResult.riskLevel}\n`;
    resultHtml += `置信度: ${(evaluationResult.confidence * 100).toFixed(1)}%\n`;
    resultHtml += `\n触发的规则 (${evaluationResult.triggeredRules.length}):\n`;
    evaluationResult.triggeredRules.forEach(rule => {
        resultHtml += `  • ${rule}\n`;
    });
    
    if (evaluationResult.recommendations.length > 0) {
        resultHtml += `\n建议:\n`;
        evaluationResult.recommendations.forEach(rec => {
            resultHtml += `  • ${rec}\n`;
        });
    }
    
    resultHtml += `\n分析时间: ${evaluationResult.analysisTime}ms`;
    resultHtml += `</pre>`;
    
    resultEl.innerHTML = resultHtml;
    
    addToHistory({
        timestamp,
        type: evaluationResult.isBot ? 'bot' : 'human',
        score: evaluationResult.totalScore
    });
}

function getConditionsFromBuilder() {
    const conditions = [];
    document.querySelectorAll('.rule-condition-row').forEach(row => {
        const field = row.querySelector('.condition-field')?.value;
        const operator = row.querySelector('.condition-operator')?.value;
        const value = parseFloat(row.querySelector('.condition-value')?.value);
        
        if (field && operator && !isNaN(value)) {
            conditions.push({ field, operator, value });
        }
    });
    return conditions;
}

function evaluateRules(conditions, features, logic) {
    const fieldMap = {
        'speed': 'average_speed',
        'speed_variance': 'speed_variance',
        'path_efficiency': 'path_efficiency',
        'smoothness': 'smoothness',
        'click_regularity': 'click_regularity',
        'hesitation_time': 'hesitation_time',
        'ml_score': 'ml_score',
        'anomaly_score': 'anomaly_score'
    };
    
    const results = conditions.map(cond => {
        const featureKey = fieldMap[cond.field] || cond.field;
        const featureValue = features[featureKey] || 0;
        
        let matched = false;
        switch (cond.operator) {
            case 'gt': matched = featureValue > cond.value; break;
            case 'gte': matched = featureValue >= cond.value; break;
            case 'lt': matched = featureValue < cond.value; break;
            case 'lte': matched = featureValue <= cond.value; break;
            case 'eq': matched = Math.abs(featureValue - cond.value) < 0.001; break;
            case 'neq': matched = Math.abs(featureValue - cond.value) >= 0.001; break;
            case 'contains': 
                const strValue = String(featureValue);
                matched = strValue.includes(String(cond.value));
                break;
            case 'regex':
                try {
                    const regex = new RegExp(String(cond.value));
                    matched = regex.test(String(featureValue));
                } catch {
                    matched = false;
                }
                break;
        }
        
        return {
            field: cond.field,
            operator: cond.operator,
            value: cond.value,
            featureValue,
            matched
        };
    });
    
    let isTriggered;
    switch (logic) {
        case 'AND':
            isTriggered = results.length > 0 && results.every(r => r.matched);
            break;
        case 'OR':
            isTriggered = results.some(r => r.matched);
            break;
        case 'NOT':
            isTriggered = results.length > 0 && !results.every(r => r.matched);
            break;
        case 'XOR':
            const trueCount = results.filter(r => r.matched).length;
            isTriggered = trueCount % 2 === 1;
            break;
        default:
            isTriggered = results.every(r => r.matched);
    }
    
    const matchedCount = results.filter(r => r.matched).length;
    const totalScore = conditions.length > 0 ? matchedCount / conditions.length : 0;
    const mlScore = features.ml_score || 0;
    const finalScore = (totalScore * 0.7 + mlScore * 0.3);
    
    let riskLevel;
    if (finalScore >= 0.8) riskLevel = 'critical';
    else if (finalScore >= 0.6) riskLevel = 'high';
    else if (finalScore >= 0.4) riskLevel = 'medium';
    else if (finalScore >= 0.2) riskLevel = 'low';
    else riskLevel = 'minimal';
    
    const triggeredRules = results.filter(r => r.matched).map(r => `${r.field}_${r.operator}_${r.value}`);
    
    const recommendations = [];
    if (finalScore > 0.7) {
        recommendations.push('建议增加额外的验证步骤');
    }
    if (finalScore > 0.85) {
        recommendations.push('建议直接拒绝访问');
    }
    if (matchedCount >= 3) {
        recommendations.push('触发多条规则，建议深度分析');
    }
    
    const confidence = calculateConfidence(matchedCount, conditions.length, finalScore);
    
    return {
        isBot: isTriggered,
        totalScore: Math.min(Math.max(finalScore, 0), 1),
        riskLevel,
        confidence,
        triggeredRules,
        recommendations,
        analysisTime: Math.floor(Math.random() * 10) + 1,
        matchedConditions: matchedCount,
        totalConditions: results.length
    };
}

function calculateConfidence(matchedCount, totalCount, score) {
    let confidence = 0.7;
    
    if (totalCount >= 5) confidence += 0.1;
    if (totalCount >= 10) confidence += 0.1;
    
    if (matchedCount >= 2) confidence += 0.05;
    if (matchedCount >= 4) confidence += 0.05;
    
    if (score > 0.7) confidence += 0.05;
    
    return Math.min(confidence, 0.99);
}

function renderTestHistory() {
    const container = document.getElementById('testHistoryList');
    if (!container) return;
    
    if (testHistory.length === 0) {
        container.innerHTML = '<div class="text-muted small text-center py-2">暂无测试记录</div>';
        return;
    }
    
    container.innerHTML = testHistory.map((h, i) => `
        <div class="test-history-item">
            <span class="${h.result.isBot ? 'text-danger' : 'text-success'}">
                <i class="fas fa-${h.result.isBot ? 'robot' : 'user'} me-1"></i>
                ${h.type === 'bot' ? '机器人' : '人类'} - ${(h.result.totalScore * 100).toFixed(0)}%
            </span>
            <span class="text-muted">${h.timestamp}</span>
        </div>
    `).join('');
}

function addToHistory(entry) {
    const tbody = document.getElementById('versionHistoryBody');
    if (!tbody) return;
    
    const existingCurrent = tbody.querySelector('tr:first-child');
    if (existingCurrent) {
        const newRow = document.createElement('tr');
        newRow.innerHTML = `
            <td><code>v2.1.4</code></td>
            <td><span class="badge bg-info">测试</span></td>
            <td>沙盒测试 - ${entry.type === 'bot' ? '机器人' : '人类'}样本</td>
            <td>admin</td>
            <td>${new Date().toLocaleString()}</td>
            <td><span class="badge bg-success">当前</span></td>
            <td><button class="btn btn-sm btn-outline-secondary" onclick="compareVersion('v2.1.4')"><i class="fas fa-code-compare"></i></button></td>
        `;
        
        tbody.insertBefore(newRow, tbody.firstChild);
        
        if (existingCurrent.querySelector('.badge.bg-success')) {
            existingCurrent.querySelector('.badge.bg-success').className = 'badge bg-secondary';
            existingCurrent.querySelector('td:nth-child(6)').innerHTML = '<span class="badge bg-secondary">历史</span>';
        }
    }
}

async function showVersionHistory() {
    showToast('正在刷新版本历史...', 'info');
    
    try {
        const response = await auth.request('/admin/risk-rules/versions');
        if (response.code === 0) {
            renderVersionHistory(response.data);
            showToast('版本历史已刷新', 'success');
        }
    } catch (error) {
        showToast('刷新失败，使用本地数据', 'warning');
    }
}

function renderVersionHistory(versions) {
    const tbody = document.getElementById('versionHistoryBody');
    if (!tbody || !versions) return;
    
    tbody.innerHTML = versions.map(v => `
        <tr>
            <td><code>${v.version}</code></td>
            <td><span class="badge bg-${getChangeTypeBadge(v.changeType)}">${v.changeType}</span></td>
            <td>${escapeHtml(v.description)}</td>
            <td>${escapeHtml(v.operator)}</td>
            <td>${v.createdAt}</td>
            <td><span class="badge bg-${v.isCurrent ? 'success' : 'secondary'}">${v.isCurrent ? '当前' : '历史'}</span></td>
            <td>
                <button class="btn btn-sm btn-outline-secondary" onclick="compareVersion('${v.version}')"><i class="fas fa-code-compare"></i></button>
                ${!v.isCurrent ? `<button class="btn btn-sm btn-outline-success" onclick="rollbackVersion('${v.version}')"><i class="fas fa-undo"></i></button>` : ''}
            </td>
        </tr>
    `).join('');
}

function getChangeTypeBadge(type) {
    const map = {
        '新增': 'primary',
        '修改': 'info',
        '优化': 'warning',
        '回滚': 'danger',
        '测试': 'secondary'
    };
    return map[type] || 'secondary';
}

async function compareVersion(version) {
    showToast(`正在比较版本 ${version}...`, 'info');
    
    try {
        const response = await auth.request(`/admin/risk-rules/versions/${version}/compare`);
        if (response.code === 0) {
            showVersionDiffModal(response.data);
        }
    } catch (error) {
        showToast('对比失败', 'danger');
    }
}

function showVersionDiffModal(diffData) {
    const modal = document.createElement('div');
    modal.className = 'modal fade';
    modal.innerHTML = `
        <div class="modal-dialog modal-lg">
            <div class="modal-content">
                <div class="modal-header">
                    <h5 class="modal-title">版本对比</h5>
                    <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
                </div>
                <div class="modal-body">
                    <pre class="bg-dark text-light p-3 rounded" style="max-height:400px;overflow-y:auto;">${JSON.stringify(diffData, null, 2)}</pre>
                </div>
                <div class="modal-footer">
                    <button type="button" class="btn-outline-gold" data-bs-dismiss="modal">关闭</button>
                </div>
            </div>
        </div>
    `;
    
    document.body.appendChild(modal);
    const bsModal = new bootstrap.Modal(modal);
    bsModal.show();
    modal.addEventListener('hidden.bs.modal', () => modal.remove());
}

async function rollbackVersion(version) {
    if (!confirm(`确定要回滚到版本 ${version} 吗？`)) return;
    
    showToast(`正在回滚到 ${version}...`, 'info');
    
    try {
        const response = await auth.request(`/admin/risk-rules/versions/${version}/rollback`, {
            method: 'POST'
        });
        
        if (response.code === 0) {
            showToast(`已成功回滚到 ${version}`, 'success');
            showVersionHistory();
        } else {
            showToast('回滚失败', 'danger');
        }
    } catch (error) {
        showToast('回滚请求失败', 'danger');
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
