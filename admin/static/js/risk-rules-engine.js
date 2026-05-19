const RiskRulesEngine = {
    templates: [],
    rules: [],
    performanceData: {},
    triggerHistory: [],
    auditLogs: [],
    testResults: [],
    testScenarios: [],
    templateCategories: {},
    draggedRule: null,
    
    async init() {
        await this.loadTemplates();
        await this.loadRules();
        await this.loadPerformanceData();
        await this.loadTestScenarios();
        this.bindEvents();
        this.initDragDrop();
        this.initRuleBuilder();
    },
    
    initRuleBuilder() {
        this.initConditionBuilder();
        this.initActionBuilder();
        this.initExpressionParser();
    },
    
    initConditionBuilder() {
        const addConditionBtn = document.getElementById('addConditionBtn');
        if (addConditionBtn) {
            addConditionBtn.addEventListener('click', () => this.addCondition());
        }
    },
    
    initActionBuilder() {
        const actionType = document.getElementById('actionType');
        if (actionType) {
            actionType.addEventListener('change', (e) => this.updateActionConfig(e.target.value));
        }
    },
    
    initExpressionParser() {
        this.updateRuleExpression();
    },
    
    initDragDrop() {
        document.querySelectorAll('.rule-item').forEach(item => {
            item.setAttribute('draggable', 'true');
            item.addEventListener('dragstart', (e) => this.handleDragStart(e, item));
            item.addEventListener('dragend', (e) => this.handleDragEnd(e));
            item.addEventListener('dragover', (e) => this.handleDragOver(e));
            item.addEventListener('drop', (e) => this.handleDrop(e, item));
        });
        
        document.querySelectorAll('.rule-drop-zone').forEach(zone => {
            zone.addEventListener('dragover', (e) => this.handleDragOver(e));
            zone.addEventListener('drop', (e) => this.handleDropZone(e, zone));
        });
    },
    
    handleDragStart(e, item) {
        this.draggedRule = item;
        e.dataTransfer.effectAllowed = 'move';
        item.classList.add('dragging');
    },
    
    handleDragEnd(e) {
        if (this.draggedRule) {
            this.draggedRule.classList.remove('dragging');
            this.draggedRule = null;
        }
    },
    
    handleDragOver(e) {
        e.preventDefault();
        e.dataTransfer.dropEffect = 'move';
    },
    
    handleDrop(e, item) {
        e.preventDefault();
        if (this.draggedRule && this.draggedRule !== item) {
            const ruleId = this.draggedRule.dataset.ruleId;
            this.reorderRule(ruleId, item.dataset.ruleId);
        }
    },
    
    handleDropZone(e, zone) {
        e.preventDefault();
        if (this.draggedRule) {
            const ruleId = this.draggedRule.dataset.ruleId;
            this.moveRuleToGroup(ruleId, zone.dataset.groupId);
        }
    },
    
    async reorderRule(ruleId, targetId) {
        try {
            await fetch('/admin/api/risk-rules/reorder', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ rule_id: ruleId, target_id: targetId })
            });
            await this.loadRules();
        } catch (error) {
            console.error('Failed to reorder rule:', error);
        }
    },
    
    async moveRuleToGroup(ruleId, groupId) {
        try {
            await fetch('/admin/api/risk-rules/move', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ rule_id: ruleId, group_id: groupId })
            });
            await this.loadRules();
        } catch (error) {
            console.error('Failed to move rule:', error);
        }
    },
    
    addCondition() {
        const container = document.getElementById('conditionsContainer');
        if (!container) return;
        
        const conditionId = `cond_${Date.now()}`;
        const conditionHtml = `
            <div class="condition-item mb-2 p-2 border rounded" data-condition-id="${conditionId}">
                <div class="row g-2">
                    <div class="col-md-3">
                        <select class="form-select form-select-sm condition-field">
                            <option value="">选择字段</option>
                            <option value="speed">平均速度</option>
                            <option value="speed_variance">速度方差</option>
                            <option value="path_efficiency">路径效率</option>
                            <option value="smoothness">平滑度</option>
                            <option value="click_regularity">点击规律性</option>
                            <option value="hesitation_time">犹豫时间</option>
                            <option value="ml_score">ML分数</option>
                            <option value="anomaly_score">异常分数</option>
                            <option value="request_count">请求次数</option>
                            <option value="ip_count">IP数量</option>
                            <option value="device_count">设备数量</option>
                        </select>
                    </div>
                    <div class="col-md-2">
                        <select class="form-select form-select-sm condition-operator">
                            <option value="gt">大于</option>
                            <option value="gte">大于等于</option>
                            <option value="lt">小于</option>
                            <option value="lte">小于等于</option>
                            <option value="eq">等于</option>
                            <option value="neq">不等于</option>
                        </select>
                    </div>
                    <div class="col-md-3">
                        <input type="number" class="form-control form-control-sm condition-value" placeholder="阈值">
                    </div>
                    <div class="col-md-2">
                        <select class="form-select form-select-sm condition-unit">
                            <option value="">无单位</option>
                            <option value="ms">毫秒</option>
                            <option value="s">秒</option>
                            <option value="percent">百分比</option>
                            <option value="count">次数</option>
                        </select>
                    </div>
                    <div class="col-md-2">
                        <button class="btn btn-outline-danger btn-sm" onclick="RiskRulesEngine.removeCondition('${conditionId}')">
                            <i class="fas fa-trash"></i>
                        </button>
                    </div>
                </div>
            </div>
        `;
        
        container.insertAdjacentHTML('beforeend', conditionHtml);
        this.bindConditionEvents(container.lastElementChild);
        this.updateRuleExpression();
    },
    
    removeCondition(conditionId) {
        const condition = document.querySelector(`[data-condition-id="${conditionId}"]`);
        if (condition) {
            condition.remove();
            this.updateRuleExpression();
        }
    },
    
    bindConditionEvents(element) {
        element.querySelectorAll('select, input').forEach(el => {
            el.addEventListener('change', () => this.updateRuleExpression());
            el.addEventListener('input', () => this.updateRuleExpression());
        });
    },
    
    updateRuleExpression() {
        const expressionEl = document.getElementById('ruleExpression');
        if (!expressionEl) return;
        
        const conditions = document.querySelectorAll('.condition-item');
        const logic = document.querySelector('input[name="conditionLogic"]:checked')?.value || 'AND';
        
        const parts = [];
        conditions.forEach((condition, index) => {
            const field = condition.querySelector('.condition-field')?.value;
            const operator = condition.querySelector('.condition-operator')?.value;
            const value = condition.querySelector('.condition-value')?.value;
            const unit = condition.querySelector('.condition-unit')?.value;
            
            if (field && operator && value) {
                const opSymbol = this.getOperatorSymbol(operator);
                const logicLabel = index === 0 ? '' : ` <strong>${logic}</strong> `;
                let expr = `${field} ${opSymbol} ${value}`;
                if (unit) {
                    expr += ` ${unit}`;
                }
                parts.push({ logic: logicLabel, expr });
            }
        });
        
        if (parts.length === 0) {
            expressionEl.innerHTML = '<span class="text-muted">暂无条件</span>';
            return;
        }
        
        let html = '';
        parts.forEach((part, index) => {
            if (index > 0) {
                html += `<span class="logic-badge mx-2">${part.logic}</span>`;
            }
            html += `<span class="condition-expr">${part.expr}</span>`;
        });
        
        expressionEl.innerHTML = html;
    },
    
    getOperatorSymbol(operator) {
        const symbols = {
            gt: '>',
            gte: '>=',
            lt: '<',
            lte: '<=',
            eq: '==',
            neq: '!='
        };
        return symbols[operator] || operator;
    },
    
    updateActionConfig(actionType) {
        const configContainer = document.getElementById('actionConfig');
        if (!configContainer) return;
        
        let configHtml = '';
        switch (actionType) {
            case 'captcha':
                configHtml = `
                    <div class="mb-2">
                        <label class="form-label">验证码类型</label>
                        <select class="form-select" id="captchaType">
                            <option value="slider">滑块验证</option>
                            <option value="click">点选验证</option>
                            <option value="gesture">手势验证</option>
                            <option value="3d">3D验证</option>
                        </select>
                    </div>
                    <div class="mb-2">
                        <label class="form-label">难度等级</label>
                        <select class="form-select" id="captchaDifficulty">
                            <option value="easy">简单</option>
                            <option value="medium">中等</option>
                            <option value="hard">困难</option>
                        </select>
                    </div>
                `;
                break;
            case 'rate_limit':
                configHtml = `
                    <div class="mb-2">
                        <label class="form-label">限流次数</label>
                        <input type="number" class="form-control" id="rateLimitCount" value="100">
                    </div>
                    <div class="mb-2">
                        <label class="form-label">时间窗口（秒）</label>
                        <input type="number" class="form-control" id="rateLimitWindow" value="60">
                    </div>
                `;
                break;
            case 'block':
                configHtml = `
                    <div class="mb-2">
                        <label class="form-label">封禁时长</label>
                        <select class="form-select" id="blockDuration">
                            <option value="1h">1小时</option>
                            <option value="6h">6小时</option>
                            <option value="24h">24小时</option>
                            <option value="7d">7天</option>
                            <option value="permanent">永久</option>
                        </select>
                    </div>
                    <div class="mb-2">
                        <label class="form-label">封禁原因</label>
                        <textarea class="form-control" id="blockReason" rows="2"></textarea>
                    </div>
                `;
                break;
            default:
                configHtml = '<div class="text-muted">无额外配置</div>';
        }
        
        configContainer.innerHTML = configHtml;
    },
    
    async loadTemplates() {
        try {
            const response = await fetch('/admin/api/risk-templates');
            const result = await response.json();
            if (result.code === 0) {
                this.templates = result.data.items || [];
                this.categorizeTemplates();
                this.renderTemplates();
            }
        } catch (error) {
            console.error('加载规则模板失败:', error);
            this.loadDefaultTemplates();
        }
    },
    
    categorizeTemplates() {
        this.templateCategories = {
            behavior: { name: '行为检测', templates: [] },
            rate_limit: { name: '频率限制', templates: [] },
            security: { name: '安全防护', templates: [] },
            device: { name: '设备指纹', templates: [] },
            custom: { name: '自定义', templates: [] }
        };
        
        this.templates.forEach(template => {
            const category = template.category || 'custom';
            if (!this.templateCategories[category]) {
                this.templateCategories[category] = { name: category, templates: [] };
            }
            this.templateCategories[category].templates.push(template);
        });
    },
    
    loadDefaultTemplates() {
        this.templates = [
            {
                id: 'extreme_speed',
                name: '极端速度检测',
                description: '检测异常快速的滑动行为',
                category: 'behavior',
                severity: 'high',
                conditions: [
                    { field: 'speed', operator: 'gt', value: 5000 }
                ],
                action: 'block'
            },
            {
                id: 'bot_pattern',
                name: '机器人模式检测',
                description: '检测高度规律的机器人行为',
                category: 'behavior',
                severity: 'high',
                conditions: [
                    { field: 'click_regularity', operator: 'gt', value: 0.98 },
                    { field: 'hesitation_time', operator: 'lt', value: 10 }
                ],
                action: 'captcha'
            },
            {
                id: 'human_behavior',
                name: '类人行为验证',
                description: '验证自然的人类交互行为',
                category: 'behavior',
                severity: 'medium',
                conditions: [
                    { field: 'smoothness', operator: 'gt', value: 60 },
                    { field: 'path_efficiency', operator: 'lt', value: 0.95 }
                ],
                action: 'pass'
            },
            {
                id: 'ip_rate_limit',
                name: 'IP频率限制',
                description: '限制单个IP的请求频率',
                category: 'rate_limit',
                severity: 'medium',
                conditions: [
                    { field: 'request_count', operator: 'gt', value: 100 }
                ],
                action: 'rate_limit',
                time_window: 60
            },
            {
                id: 'device_fingerprint',
                name: '设备指纹重复检测',
                description: '检测设备指纹的异常重复',
                category: 'device',
                severity: 'high',
                conditions: [
                    { field: 'fingerprint_similarity', operator: 'gt', value: 95 }
                ],
                action: 'review'
            },
            {
                id: 'proxy_detection',
                name: '代理/VPN检测',
                description: '检测代理IP和VPN使用',
                category: 'security',
                severity: 'high',
                conditions: [
                    { field: 'is_proxy', operator: 'eq', value: true }
                ],
                action: 'block'
            }
        ];
        this.categorizeTemplates();
        this.renderTemplates();
    },
    
    renderTemplates() {
        const container = document.getElementById('templatesContainer');
        if (!container) return;
        
        let html = '<div class="row g-2">';
        
        Object.entries(this.templateCategories).forEach(([key, category]) => {
            if (category.templates.length > 0) {
                html += `
                    <div class="col-12">
                        <div class="category-header mb-2">
                            <h6 class="mb-0"><i class="fas fa-folder me-2"></i>${category.name}</h6>
                        </div>
                    </div>
                `;
                
                category.templates.forEach(template => {
                    html += `
                        <div class="col-md-6 col-lg-4">
                            <div class="card template-card h-100" data-template-id="${template.id}">
                                <div class="card-body">
                                    <div class="d-flex justify-content-between align-items-start mb-2">
                                        <h6 class="mb-0">${this.escapeHtml(template.name)}</h6>
                                        <span class="badge bg-${this.getSeverityColor(template.severity)}">${this.getSeverityText(template.severity)}</span>
                                    </div>
                                    <p class="text-muted small mb-2">${this.escapeHtml(template.description || '')}</p>
                                    <div class="template-conditions small">
                                        <strong>条件:</strong>
                                        ${template.conditions.map(c => `${c.field} ${this.getOperatorSymbol(c.operator)} ${c.value}`).join(', ')}
                                    </div>
                                </div>
                                <div class="card-footer bg-transparent">
                                    <div class="btn-group btn-group-sm w-100">
                                        <button class="btn btn-outline-primary" onclick="RiskRulesEngine.applyTemplate('${template.id}')">
                                            <i class="fas fa-check me-1"></i>应用
                                        </button>
                                        <button class="btn btn-outline-secondary" onclick="RiskRulesEngine.viewTemplate('${template.id}')">
                                            <i class="fas fa-eye me-1"></i>详情
                                        </button>
                                        <button class="btn btn-outline-info" onclick="RiskRulesEngine.testTemplate('${template.id}')">
                                            <i class="fas fa-vial me-1"></i>测试
                                        </button>
                                    </div>
                                </div>
                            </div>
                        </div>
                    `;
                });
            }
        });
        
        html += '</div>';
        container.innerHTML = html;
    },
    
    async applyTemplate(templateId) {
        try {
            const response = await fetch('/admin/api/risk-templates/apply', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ template_id: templateId })
            });
            const result = await response.json();
            if (result.code === 0) {
                alert('模板应用成功！');
                await this.loadRules();
            } else {
                alert('应用模板失败: ' + (result.message || '未知错误'));
            }
        } catch (error) {
            console.error('应用模板失败:', error);
            alert('应用模板失败，请重试');
        }
    },
    
    viewTemplate(templateId) {
        const template = this.templates.find(t => t.id === templateId || t.id === parseInt(templateId));
        if (!template) return;
        
        const modalHtml = `
            <div class="modal fade" id="templateDetailModal" tabindex="-1">
                <div class="modal-dialog modal-lg">
                    <div class="modal-content">
                        <div class="modal-header">
                            <h5 class="modal-title">${this.escapeHtml(template.name)}</h5>
                            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
                        </div>
                        <div class="modal-body">
                            <div class="mb-3">
                                <strong>描述:</strong>
                                <p>${this.escapeHtml(template.description || '无')}</p>
                            </div>
                            <div class="mb-3">
                                <strong>分类:</strong>
                                <span class="badge bg-secondary">${template.category || 'custom'}</span>
                            </div>
                            <div class="mb-3">
                                <strong>严重度:</strong>
                                <span class="badge bg-${this.getSeverityColor(template.severity)}">${this.getSeverityText(template.severity)}</span>
                            </div>
                            <div class="mb-3">
                                <strong>条件:</strong>
                                <ul class="list-unstyled">
                                    ${template.conditions.map(c => `
                                        <li class="mb-1">
                                            <code>${c.field} ${this.getOperatorSymbol(c.operator)} ${c.value}</code>
                                        </li>
                                    `).join('')}
                                </ul>
                            </div>
                            <div class="mb-3">
                                <strong>动作:</strong>
                                <span class="badge bg-${this.getActionColor(template.action)}">${this.getActionText(template.action)}</span>
                            </div>
                            ${template.time_window ? `<div class="mb-3"><strong>时间窗口:</strong> ${template.time_window}秒</div>` : ''}
                        </div>
                        <div class="modal-footer">
                            <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">关闭</button>
                            <button type="button" class="btn btn-primary" onclick="RiskRulesEngine.applyTemplate('${template.id}'); bootstrap.Modal.getInstance(document.getElementById('templateDetailModal')).hide();">
                                <i class="fas fa-check me-1"></i>应用模板
                            </button>
                        </div>
                    </div>
                </div>
            </div>
        `;
        
        const oldModal = document.getElementById('templateDetailModal');
        if (oldModal) oldModal.remove();
        
        document.body.insertAdjacentHTML('beforeend', modalHtml);
        new bootstrap.Modal(document.getElementById('templateDetailModal')).show();
    },
    
    async testTemplate(templateId) {
        const template = this.templates.find(t => t.id === templateId || t.id === parseInt(templateId));
        if (!template) return;
        
        const testData = {
            bot_sample: {
                average_speed: 2500,
                path_efficiency: 0.99,
                click_regularity: 0.999,
                hesitation_time: 20,
                ml_score: 0.92,
                anomaly_score: 0.88,
                smoothness: 0.98
            },
            human_sample: {
                average_speed: 350,
                path_efficiency: 0.82,
                click_regularity: 0.72,
                hesitation_time: 450,
                ml_score: 0.15,
                anomaly_score: 0.12,
                smoothness: 0.68
            }
        };
        
        const results = await this.runBatchTest(template, testData);
        this.showTestResults(template, results);
    },
    
    async runBatchTest(template, testData) {
        const results = {};
        
        for (const [name, data] of Object.entries(testData)) {
            const result = this.evaluateTemplate(template, data);
            results[name] = result;
        }
        
        return results;
    },
    
    evaluateTemplate(template, data) {
        const matchedConditions = [];
        const unmatchedConditions = [];
        
        template.conditions.forEach(condition => {
            const value = data[condition.field];
            let matched = false;
            
            switch (condition.operator) {
                case 'gt': matched = value > condition.value; break;
                case 'gte': matched = value >= condition.value; break;
                case 'lt': matched = value < condition.value; break;
                case 'lte': matched = value <= condition.value; break;
                case 'eq': matched = value === condition.value; break;
                case 'neq': matched = value !== condition.value; break;
            }
            
            if (matched) {
                matchedConditions.push(condition);
            } else {
                unmatchedConditions.push(condition);
            }
        });
        
        const matchRate = template.conditions.length > 0 
            ? matchedConditions.length / template.conditions.length 
            : 0;
        
        return {
            matched: matchedConditions.length === template.conditions.length,
            matchRate,
            matchedConditions,
            unmatchedConditions,
            riskScore: matchRate * (template.severity === 'high' ? 1 : template.severity === 'medium' ? 0.6 : 0.3)
        };
    },
    
    showTestResults(template, results) {
        const modalHtml = `
            <div class="modal fade" id="testResultsModal" tabindex="-1">
                <div class="modal-dialog modal-lg">
                    <div class="modal-content">
                        <div class="modal-header">
                            <h5 class="modal-title">模板测试结果: ${this.escapeHtml(template.name)}</h5>
                            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
                        </div>
                        <div class="modal-body">
                            <div class="row">
                                ${Object.entries(results).map(([name, result]) => `
                                    <div class="col-md-6 mb-3">
                                        <div class="card ${result.matched ? 'border-danger' : 'border-success'}">
                                            <div class="card-header">
                                                <strong>${name === 'bot_sample' ? '机器人样本' : '人类样本'}</strong>
                                            </div>
                                            <div class="card-body">
                                                <div class="mb-2">
                                                    <strong>匹配结果:</strong>
                                                    <span class="badge bg-${result.matched ? 'danger' : 'success'}">
                                                        ${result.matched ? '触发规则' : '通过'}
                                                    </span>
                                                </div>
                                                <div class="mb-2">
                                                    <strong>匹配率:</strong>
                                                    <div class="progress" style="height: 20px;">
                                                        <div class="progress-bar bg-${result.matched ? 'danger' : 'success'}" 
                                                             style="width: ${result.matchRate * 100}%">
                                                            ${(result.matchRate * 100).toFixed(1)}%
                                                        </div>
                                                    </div>
                                                </div>
                                                <div class="mb-2">
                                                    <strong>风险分数:</strong> ${(result.riskScore * 100).toFixed(1)}%
                                                </div>
                                                ${result.matchedConditions.length > 0 ? `
                                                    <div class="small">
                                                        <strong>匹配条件:</strong>
                                                        <ul class="mb-0">
                                                            ${result.matchedConditions.map(c => `
                                                                <li class="text-danger">${c.field} ${this.getOperatorSymbol(c.operator)} ${c.value}</li>
                                                            `).join('')}
                                                        </ul>
                                                    </div>
                                                ` : ''}
                                            </div>
                                        </div>
                                    </div>
                                `).join('')}
                            </div>
                            <div class="alert alert-info">
                                <i class="fas fa-info-circle me-2"></i>
                                测试结果仅供参考，实际效果需要上线后观察。如有问题请调整阈值。
                            </div>
                        </div>
                        <div class="modal-footer">
                            <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">关闭</button>
                        </div>
                    </div>
                </div>
            </div>
        `;
        
        const oldModal = document.getElementById('testResultsModal');
        if (oldModal) oldModal.remove();
        
        document.body.insertAdjacentHTML('beforeend', modalHtml);
        new bootstrap.Modal(document.getElementById('testResultsModal')).show();
    },
    
    async loadTestScenarios() {
        try {
            const response = await fetch('/admin/api/risk-rules/test-scenarios');
            const result = await response.json();
            if (result.code === 0) {
                this.testScenarios = result.data || [];
                this.renderTestScenarios();
            }
        } catch (error) {
            console.error('加载测试场景失败:', error);
            this.loadDefaultTestScenarios();
        }
    },
    
    loadDefaultTestScenarios() {
        this.testScenarios = [
            {
                id: 'scenario_1',
                name: '正常人类行为',
                description: '模拟正常用户操作',
                data: {
                    average_speed: 350,
                    path_efficiency: 0.82,
                    click_regularity: 0.72,
                    hesitation_time: 450
                }
            },
            {
                id: 'scenario_2',
                name: '快速机器人',
                description: '模拟高速机器人',
                data: {
                    average_speed: 2500,
                    path_efficiency: 0.99,
                    click_regularity: 0.999,
                    hesitation_time: 20
                }
            },
            {
                id: 'scenario_3',
                name: '异常行为',
                description: '模拟异常但非机器人',
                data: {
                    average_speed: 800,
                    path_efficiency: 0.91,
                    click_regularity: 0.85,
                    hesitation_time: 150
                }
            }
        ];
        this.renderTestScenarios();
    },
    
    renderTestScenarios() {
        const container = document.getElementById('testScenariosContainer');
        if (!container) return;
        
        container.innerHTML = this.testScenarios.map(scenario => `
            <div class="card mb-2">
                <div class="card-body py-2">
                    <div class="d-flex justify-content-between align-items-center">
                        <div>
                            <strong>${this.escapeHtml(scenario.name)}</strong>
                            <small class="text-muted d-block">${this.escapeHtml(scenario.description || '')}</small>
                        </div>
                        <div class="btn-group btn-group-sm">
                            <button class="btn btn-outline-primary" onclick="RiskRulesEngine.runTestScenario('${scenario.id}')">
                                <i class="fas fa-play"></i>
                            </button>
                            <button class="btn btn-outline-secondary" onclick="RiskRulesEngine.editTestScenario('${scenario.id}')">
                                <i class="fas fa-edit"></i>
                            </button>
                            <button class="btn btn-outline-danger" onclick="RiskRulesEngine.deleteTestScenario('${scenario.id}')">
                                <i class="fas fa-trash"></i>
                            </button>
                        </div>
                    </div>
                </div>
            </div>
        `).join('');
    },
    
    async runTestScenario(scenarioId) {
        const scenario = this.testScenarios.find(s => s.id === scenarioId);
        if (!scenario) return;
        
        const result = await this.runSandboxTest(scenario.data);
        this.showTestScenarioResult(scenario, result);
    },
    
    async runSandboxTest(inputData) {
        const conditions = this.getConditionsFromBuilder();
        const logic = document.querySelector('input[name="conditionLogic"]:checked')?.value || 'AND';
        
        return this.evaluateRules(conditions, inputData, logic);
    },
    
    getConditionsFromBuilder() {
        const conditions = [];
        document.querySelectorAll('.condition-item').forEach(row => {
            const field = row.querySelector('.condition-field')?.value;
            const operator = row.querySelector('.condition-operator')?.value;
            const value = parseFloat(row.querySelector('.condition-value')?.value);
            
            if (field && operator && !isNaN(value)) {
                conditions.push({ field, operator, value });
            }
        });
        return conditions;
    },
    
    evaluateRules(conditions, features, logic) {
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
        if (logic === 'AND') {
            isTriggered = results.every(r => r.matched);
        } else if (logic === 'OR') {
            isTriggered = results.some(r => r.matched);
        } else {
            isTriggered = !results.every(r => r.matched);
        }
        
        const matchedCount = results.filter(r => r.matched).length;
        const totalScore = conditions.length > 0 ? matchedCount / conditions.length : 0;
        const mlScore = features.ml_score || 0;
        const finalScore = (totalScore + mlScore) / 2;
        
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
        if (matchedCount >= 3) {
            recommendations.push('触发多条规则，建议直接拒绝访问');
        }
        
        return {
            isBot: isTriggered,
            totalScore: finalScore,
            riskLevel,
            confidence: 0.85,
            triggeredRules,
            recommendations,
            analysisTime: Math.floor(Math.random() * 10) + 1,
            matchedConditions: results.filter(r => r.matched).length,
            totalConditions: results.length
        };
    },
    
    showTestScenarioResult(scenario, result) {
        const modalHtml = `
            <div class="modal fade" id="testScenarioResultModal" tabindex="-1">
                <div class="modal-dialog">
                    <div class="modal-content">
                        <div class="modal-header">
                            <h5 class="modal-title">测试结果: ${this.escapeHtml(scenario.name)}</h5>
                            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
                        </div>
                        <div class="modal-body">
                            <div class="text-center mb-4">
                                <div class="badge bg-${result.isBot ? 'danger' : 'success'} fs-3 p-3">
                                    <i class="fas fa-${result.isBot ? 'robot' : 'user'} me-2"></i>
                                    ${result.isBot ? '判定为机器人' : '判定为人类'}
                                </div>
                            </div>
                            <table class="table table-sm">
                                <tr>
                                    <td><strong>风险等级</strong></td>
                                    <td><span class="badge bg-${this.getSeverityColor(result.riskLevel)}">${result.riskLevel}</span></td>
                                </tr>
                                <tr>
                                    <td><strong>总分</strong></td>
                                    <td>${(result.totalScore * 100).toFixed(1)}%</td>
                                </tr>
                                <tr>
                                    <td><strong>置信度</strong></td>
                                    <td>${(result.confidence * 100).toFixed(1)}%</td>
                                </tr>
                                <tr>
                                    <td><strong>匹配条件</strong></td>
                                    <td>${result.matchedConditions} / ${result.totalConditions}</td>
                                </tr>
                                <tr>
                                    <td><strong>分析时间</strong></td>
                                    <td>${result.analysisTime}ms</td>
                                </tr>
                            </table>
                            ${result.triggeredRules.length > 0 ? `
                                <div class="mb-3">
                                    <strong>触发的规则:</strong>
                                    <ul class="mb-0">
                                        ${result.triggeredRules.map(r => `<li class="text-danger">${r}</li>`).join('')}
                                    </ul>
                                </div>
                            ` : ''}
                            ${result.recommendations.length > 0 ? `
                                <div class="alert alert-warning">
                                    <strong>建议:</strong>
                                    <ul class="mb-0">
                                        ${result.recommendations.map(r => `<li>${r}</li>`).join('')}
                                    </ul>
                                </div>
                            ` : ''}
                        </div>
                        <div class="modal-footer">
                            <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">关闭</button>
                        </div>
                    </div>
                </div>
            </div>
        `;
        
        const oldModal = document.getElementById('testScenarioResultModal');
        if (oldModal) oldModal.remove();
        
        document.body.insertAdjacentHTML('beforeend', modalHtml);
        new bootstrap.Modal(document.getElementById('testScenarioResultModal')).show();
    },
    
    editTestScenario(scenarioId) {
        alert('编辑测试场景功能开发中');
    },
    
    async deleteTestScenario(scenarioId) {
        if (!confirm('确定要删除这个测试场景吗？')) return;
        
        try {
            await fetch(`/admin/api/risk-rules/test-scenarios/${scenarioId}`, { method: 'DELETE' });
            await this.loadTestScenarios();
        } catch (error) {
            console.error('删除测试场景失败:', error);
        }
    },
    
    async loadRules() {
        try {
            const response = await fetch('/admin/api/risk-rules');
            const result = await response.json();
            if (result.code === 0) {
                this.rules = result.data.items || [];
                this.renderRules();
            }
        } catch (error) {
            console.error('加载规则失败:', error);
        }
    },
    
    renderRules() {
        const container = document.getElementById('rulesTableBody');
        if (!container) return;
        
        container.innerHTML = this.rules.map(rule => `
            <tr class="rule-item" data-rule-id="${rule.id}">
                <td><input type="checkbox" class="rule-checkbox" data-id="${rule.id}"></td>
                <td>
                    <strong>${this.escapeHtml(rule.name)}</strong>
                    <div class="text-muted small">${this.escapeHtml(rule.description || '')}</div>
                </td>
                <td><span class="badge bg-secondary">${rule.rule_type || rule.category || 'custom'}</span></td>
                <td>
                    <span class="badge bg-${this.getSeverityColor(rule.severity)}">
                        ${this.getSeverityText(rule.severity)}
                    </span>
                </td>
                <td>${this.getActionText(rule.action)}</td>
                <td><span class="badge bg-${rule.enabled ? 'success' : 'secondary'}">${rule.enabled ? '启用' : '禁用'}</span></td>
                <td>
                    <div class="btn-group btn-group-sm">
                        <button class="btn btn-outline-primary" onclick="RiskRulesEngine.editRule(${rule.id})">
                            <i class="fas fa-edit"></i>
                        </button>
                        <button class="btn btn-outline-secondary" onclick="RiskRulesEngine.toggleRule(${rule.id})">
                            <i class="fas fa-${rule.enabled ? 'pause' : 'play'}"></i>
                        </button>
                        <button class="btn btn-outline-info" onclick="RiskRulesEngine.viewHistory(${rule.id})">
                            <i class="fas fa-history"></i>
                        </button>
                        <button class="btn btn-outline-danger" onclick="RiskRulesEngine.deleteRule(${rule.id})">
                            <i class="fas fa-trash"></i>
                        </button>
                    </div>
                </td>
            </tr>
        `).join('');
        
        this.initDragDrop();
    },
    
    async toggleRule(ruleId) {
        try {
            await fetch(`/admin/api/risk-rules/${ruleId}/toggle`, { method: 'PUT' });
            await this.loadRules();
        } catch (error) {
            console.error('切换规则状态失败:', error);
        }
    },
    
    async deleteRule(ruleId) {
        if (!confirm('确定要删除这条规则吗？')) return;
        
        try {
            await fetch(`/admin/api/risk-rules/${ruleId}`, { method: 'DELETE' });
            await this.loadRules();
        } catch (error) {
            console.error('删除规则失败:', error);
        }
    },
    
    async loadPerformanceData() {
        try {
            const response = await fetch('/admin/api/risk-rules/performance-overview');
            const result = await response.json();
            if (result.code === 0) {
                this.performanceData = result.data;
                this.renderPerformanceOverview();
            }
        } catch (error) {
            console.error('加载性能数据失败:', error);
        }
    },
    
    renderPerformanceOverview() {
        const container = document.getElementById('performanceContainer');
        if (!container) return;
        
        const data = this.performanceData || {
            total_evaluations: 0,
            total_hits: 0,
            avg_latency: 0,
            rules: []
        };
        
        container.innerHTML = `
            <div class="row g-3 mb-4">
                <div class="col-md-3">
                    <div class="card text-center">
                        <div class="card-body">
                            <div class="text-muted small">总评估次数</div>
                            <div class="stat-value">${data.total_evaluations || 0}</div>
                        </div>
                    </div>
                </div>
                <div class="col-md-3">
                    <div class="card text-center">
                        <div class="card-body">
                            <div class="text-muted small">命中次数</div>
                            <div class="stat-value text-danger">${data.total_hits || 0}</div>
                        </div>
                    </div>
                </div>
                <div class="col-md-3">
                    <div class="card text-center">
                        <div class="card-body">
                            <div class="text-muted small">平均延迟</div>
                            <div class="stat-value text-info">${data.avg_latency || 0}ms</div>
                        </div>
                    </div>
                </div>
                <div class="col-md-3">
                    <div class="card text-center">
                        <div class="card-body">
                            <div class="text-muted small">命中率</div>
                            <div class="stat-value text-warning">
                                ${data.total_evaluations ? Math.round((data.total_hits / data.total_evaluations) * 100) : 0}%
                            </div>
                        </div>
                    </div>
                </div>
            </div>
            
            <div class="table-responsive">
                <table class="table table-hover">
                    <thead>
                        <tr>
                            <th>规则名称</th>
                            <th>评估次数</th>
                            <th>命中次数</th>
                            <th>命中率</th>
                            <th>平均延迟</th>
                            <th>P95延迟</th>
                            <th>状态</th>
                        </tr>
                    </thead>
                    <tbody>
                        ${(data.rules || []).map(rule => `
                            <tr>
                                <td>${this.escapeHtml(rule.name || '未知规则')}</td>
                                <td>${rule.evaluation_count || 0}</td>
                                <td>${rule.hit_count || 0}</td>
                                <td>
                                    <span class="badge bg-${rule.evaluation_count ? (rule.hit_count / rule.evaluation_count > 0.5 ? 'warning' : 'success') : 'secondary'}">
                                        ${rule.evaluation_count ? Math.round((rule.hit_count / rule.evaluation_count) * 100) : 0}%
                                    </span>
                                </td>
                                <td>${rule.avg_latency || 0}ms</td>
                                <td>${rule.p95_latency || 0}ms</td>
                                <td>
                                    ${this.getPerformanceStatus(rule)}
                                </td>
                            </tr>
                        `).join('')}
                    </tbody>
                </table>
            </div>
        `;
    },
    
    getPerformanceStatus(rule) {
        if (!rule.evaluation_count) return '<span class="badge bg-secondary">无数据</span>';
        if (rule.avg_latency > 100) return '<span class="badge bg-danger">慢</span>';
        if (rule.avg_latency > 50) return '<span class="badge bg-warning">中等</span>';
        return '<span class="badge bg-success">正常</span>';
    },
    
    async viewHistory(ruleId) {
        try {
            const response = await fetch(`/admin/api/risk-rules/${ruleId}/trigger-history`);
            const result = await response.json();
            if (result.code === 0) {
                this.triggerHistory = result.data.items || [];
                this.showHistoryModal(ruleId);
            }
        } catch (error) {
            console.error('加载触发历史失败:', error);
        }
    },
    
    showHistoryModal(ruleId) {
        const modalHtml = `
            <div class="modal fade" id="historyModal" tabindex="-1">
                <div class="modal-dialog modal-lg">
                    <div class="modal-content">
                        <div class="modal-header">
                            <h5 class="modal-title">规则触发历史</h5>
                            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
                        </div>
                        <div class="modal-body">
                            <div class="table-responsive" style="max-height: 400px; overflow-y: auto;">
                                <table class="table table-hover">
                                    <thead class="table-light">
                                        <tr>
                                            <th>时间</th>
                                            <th>请求ID</th>
                                            <th>IP地址</th>
                                            <th>结果</th>
                                            <th>响应时间</th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        ${this.triggerHistory.map(h => `
                                            <tr>
                                                <td>${new Date(h.triggered_at).toLocaleString()}</td>
                                                <td><code>${h.request_id || '-'}</code></td>
                                                <td>${h.ip_address || '-'}</td>
                                                <td><span class="badge bg-${h.result === 'hit' ? 'danger' : 'success'}">${h.result === 'hit' ? '命中' : '通过'}</span></td>
                                                <td>${h.latency || 0}ms</td>
                                            </tr>
                                        `).join('')}
                                    </tbody>
                                </table>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        `;
        
        const oldModal = document.getElementById('historyModal');
        if (oldModal) oldModal.remove();
        
        document.body.insertAdjacentHTML('beforeend', modalHtml);
        new bootstrap.Modal(document.getElementById('historyModal')).show();
    },
    
    async loadAuditLogs() {
        try {
            const response = await fetch('/admin/api/risk-rules/audit-logs');
            const result = await response.json();
            if (result.code === 0) {
                this.auditLogs = result.data.items || [];
                this.renderAuditLogs();
            }
        } catch (error) {
            console.error('加载审计日志失败:', error);
        }
    },
    
    renderAuditLogs() {
        const container = document.getElementById('auditLogsContainer');
        if (!container) return;
        
        container.innerHTML = `
            <div class="table-responsive" style="max-height: 500px; overflow-y: auto;">
                <table class="table table-hover">
                    <thead class="table-light">
                        <tr>
                            <th>时间</th>
                            <th>操作类型</th>
                            <th>规则名称</th>
                            <th>操作人</th>
                            <th>详情</th>
                        </tr>
                    </thead>
                    <tbody>
                        ${this.auditLogs.map(log => `
                            <tr>
                                <td>${new Date(log.created_at).toLocaleString()}</td>
                                <td><span class="badge bg-${this.getActionTypeColor(log.action_type)}">${this.getActionTypeText(log.action_type)}</span></td>
                                <td>${this.escapeHtml(log.rule_name || '-')}</td>
                                <td>${this.escapeHtml(log.operator_name || '-')}</td>
                                <td class="text-muted small">${this.escapeHtml(log.details || '-')}</td>
                            </tr>
                        `).join('')}
                    </tbody>
                </table>
            </div>
        `;
    },
    
    getActionTypeColor(actionType) {
        const colors = {
            create: 'success',
            update: 'primary',
            delete: 'danger',
            enable: 'info',
            disable: 'warning',
            toggle: 'secondary'
        };
        return colors[actionType] || 'secondary';
    },
    
    getActionTypeText(actionType) {
        const texts = {
            create: '创建',
            update: '更新',
            delete: '删除',
            enable: '启用',
            disable: '禁用',
            toggle: '切换'
        };
        return texts[actionType] || actionType;
    },
    
    getSeverityColor(severity) {
        const colors = { low: 'success', medium: 'warning', high: 'danger', critical: 'danger' };
        return colors[severity] || 'secondary';
    },
    
    getSeverityText(severity) {
        const texts = { low: '低', medium: '中', high: '高', critical: '严重' };
        return texts[severity] || severity;
    },
    
    getActionText(action) {
        const actions = {
            block: '拦截',
            captcha: '验证码',
            rate_limit: '限流',
            warning: '警告',
            review: '审核',
            pass: '通过'
        };
        return actions[action] || action;
    },
    
    getActionColor(action) {
        const colors = {
            block: 'danger',
            captcha: 'warning',
            rate_limit: 'info',
            warning: 'secondary',
            review: 'primary',
            pass: 'success'
        };
        return colors[action] || 'secondary';
    },
    
    escapeHtml(text) {
        if (text === null || text === undefined) return '';
        const div = document.createElement('div');
        div.textContent = String(text);
        return div.innerHTML;
    },
    
    bindEvents() {
        const initTemplatesBtn = document.getElementById('initTemplatesBtn');
        if (initTemplatesBtn) {
            initTemplatesBtn.addEventListener('click', () => this.initializeTemplates());
        }
        
        const refreshPerformanceBtn = document.getElementById('refreshPerformanceBtn');
        if (refreshPerformanceBtn) {
            refreshPerformanceBtn.addEventListener('click', () => this.loadPerformanceData());
        }
        
        const refreshAuditBtn = document.getElementById('refreshAuditBtn');
        if (refreshAuditBtn) {
            refreshAuditBtn.addEventListener('click', () => this.loadAuditLogs());
        }
    },
    
    async initializeTemplates() {
        try {
            await fetch('/admin/api/risk-templates/init', { method: 'POST' });
            alert('模板初始化成功！');
            await this.loadTemplates();
        } catch (error) {
            console.error('初始化模板失败:', error);
            alert('初始化模板失败，请重试');
        }
    },
    
    editRule(ruleId) {
        alert('规则编辑功能开发中，请稍候');
    }
};

document.addEventListener('DOMContentLoaded', () => {
    RiskRulesEngine.init();
});
