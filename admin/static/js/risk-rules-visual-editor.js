// 风险规则可视化编辑器
let ruleEditor = {
    currentRule: null,
    rules: [],
    availableFields: [],
    availableOperators: ['=', '!=', '>', '<', '>=', '<=', 'CONTAINS', 'MATCHES', 'IN', 'NOT IN'],
    
    init: function() {
        this.loadAvailableFields();
        this.bindEvents();
        this.initMonacoEditor();
        this.loadRules();
    },
    
    loadAvailableFields: function() {
        this.availableFields = [
            { name: 'ip', type: 'string', description: 'IP地址' },
            { name: 'user_agent', type: 'string', description: 'User Agent' },
            { name: 'country', type: 'string', description: '国家' },
            { name: 'city', type: 'string', description: '城市' },
            { name: 'device_type', type: 'string', description: '设备类型' },
            { name: 'browser', type: 'string', description: '浏览器' },
            { name: 'os', type: 'string', description: '操作系统' },
            { name: 'request_count', type: 'number', description: '请求次数' },
            { name: 'fail_count', type: 'number', description: '失败次数' },
            { name: 'session_count', type: 'number', description: '会话数量' },
            { name: 'risk_score', type: 'number', description: '风险评分' },
            { name: 'timestamp', type: 'number', description: '时间戳' },
            { name: 'email_domain', type: 'string', description: '邮箱域名' },
            { name: 'referer', type: 'string', description: '来源页面' },
            { name: 'url_path', type: 'string', description: 'URL路径' },
            { name: 'method', type: 'string', description: 'HTTP方法' }
        ];
        
        this.renderFieldSelect();
    },
    
    renderFieldSelect: function() {
        const fieldSelect = document.getElementById('conditionField');
        if (!fieldSelect) return;
        
        fieldSelect.innerHTML = this.availableFields.map(field => 
            `<option value="${field.name}" data-type="${field.type}">${field.name} (${field.description})</option>`
        ).join('');
    },
    
    bindEvents: function() {
        document.getElementById('addConditionBtn').addEventListener('click', () => this.addCondition());
        document.getElementById('addActionBtn').addEventListener('click', () => this.addAction());
        document.getElementById('saveRuleBtn').addEventListener('click', () => this.saveRule());
        document.getElementById('previewRuleBtn').addEventListener('click', () => this.previewRule());
        document.getElementById('testRuleBtn').addEventListener('click', () => this.testRule());
        document.getElementById('validateDSLBtn').addEventListener('click', () => this.validateDSL());
        document.getElementById('generateFromDSLBtn').addEventListener('click', () => this.generateFromDSL());
        
        document.getElementById('conditionField').addEventListener('change', (e) => this.onFieldChange(e));
    },
    
    initMonacoEditor: function() {
        const container = document.getElementById('dslEditor');
        if (!container) return;
        
        require.config({ paths: { vs: 'https://cdn.bootcdn.net/ajax/libs/monaco-editor/0.45.0/min/vs' } });
        require(['vs/editor/editor.main'], () => {
            window.dslEditorInstance = monaco.editor.create(container, {
                value: this.getDefaultDSLTemplate(),
                language: 'plaintext',
                theme: 'vs-dark',
                minimap: { enabled: false },
                automaticLayout: true,
                lineNumbers: 'on',
                scrollBeyondLastLine: false,
                wordWrap: 'on'
            });
        });
    },
    
    getDefaultDSLTemplate: function() {
        return `RULE "示例规则"
WHEN 
  ip IN ["192.168.1.1", "10.0.0.1"] 
  AND fail_count > 5 
  AND risk_score >= 50
THEN 
  BLOCK("检测到可疑IP地址和异常失败次数");
  ADD SCORE 30;
  ALERT("高风险请求");
SCORE 50;`;
    },
    
    addCondition: function() {
        const field = document.getElementById('conditionField').value;
        const operator = document.getElementById('conditionOperator').value;
        const value = document.getElementById('conditionValue').value;
        const logic = document.getElementById('conditionLogic').value;
        
        if (!field || !operator || value === '') {
            alert('请填写完整的条件');
            return;
        }
        
        const conditionsContainer = document.getElementById('conditionsList');
        const conditionHtml = this.renderConditionRow({
            field, operator, value, logic
        });
        
        conditionsContainer.insertAdjacentHTML('beforeend', conditionHtml);
        this.updateDSLPreview();
    },
    
    renderConditionRow: function(condition) {
        return `
            <div class="rule-condition-row" data-field="${condition.field}">
                <span class="condition-logic-label">${condition.logic}</span>
                <select class="form-select form-select-sm" style="width: auto;" disabled>
                    <option value="AND" ${condition.logic === 'AND' ? 'selected' : ''}>AND</option>
                    <option value="OR" ${condition.logic === 'OR' ? 'selected' : ''}>OR</option>
                </select>
                <span class="badge bg-primary">${condition.field}</span>
                <span class="badge bg-secondary">${condition.operator}</span>
                <span class="badge bg-info">${condition.value}</span>
                <button type="button" class="btn btn-outline-danger btn-sm" onclick="ruleEditor.removeCondition(this)">
                    <i class="fas fa-trash"></i>
                </button>
            </div>
        `;
    },
    
    removeCondition: function(btn) {
        btn.closest('.rule-condition-row').remove();
        this.updateDSLPreview();
    },
    
    addAction: function() {
        const actionType = document.getElementById('actionType').value;
        const actionValue = document.getElementById('actionValue').value;
        
        if (!actionType || !actionValue) {
            alert('请填写完整的动作');
            return;
        }
        
        const actionsContainer = document.getElementById('actionsList');
        const actionHtml = this.renderActionRow({
            type: actionType,
            value: actionValue
        });
        
        actionsContainer.insertAdjacentHTML('beforeend', actionHtml);
        this.updateDSLPreview();
    },
    
    renderActionRow: function(action) {
        const actionIcons = {
            'block': 'fa-ban',
            'alert': 'fa-bell',
            'add_score': 'fa-plus-circle',
            'set': 'fa-cog'
        };
        
        return `
            <div class="rule-action-row mb-2 p-2 border rounded">
                <div class="d-flex align-items-center gap-2">
                    <i class="fas ${actionIcons[action.type] || 'fa-cog'} text-primary"></i>
                    <span class="badge bg-primary">${action.type}</span>
                    <span class="flex-grow-1">${action.value}</span>
                    <button type="button" class="btn btn-outline-danger btn-sm" onclick="ruleEditor.removeAction(this)">
                        <i class="fas fa-trash"></i>
                    </button>
                </div>
            </div>
        `;
    },
    
    removeAction: function(btn) {
        btn.closest('.rule-action-row').remove();
        this.updateDSLPreview();
    },
    
    onFieldChange: function(e) {
        const selectedOption = e.target.selectedOptions[0];
        const fieldType = selectedOption.dataset.type;
        
        const operatorSelect = document.getElementById('conditionOperator');
        operatorSelect.innerHTML = this.availableOperators.map(op => 
            `<option value="${op}">${op}</option>`
        ).join('');
        
        const valueInput = document.getElementById('conditionValue');
        if (fieldType === 'number') {
            valueInput.type = 'number';
            valueInput.placeholder = '输入数值';
        } else {
            valueInput.type = 'text';
            valueInput.placeholder = '输入值';
        }
    },
    
    updateDSLPreview: function() {
        const conditions = this.collectConditions();
        const actions = this.collectActions();
        const name = document.getElementById('ruleName').value || '未命名规则';
        const score = parseInt(document.getElementById('ruleScore').value) || 0;
        
        const dsl = this.generateDSL(name, conditions, actions, score);
        
        if (window.dslEditorInstance) {
            window.dslEditorInstance.setValue(dsl);
        }
    },
    
    collectConditions: function() {
        const conditions = [];
        const rows = document.querySelectorAll('#conditionsList .rule-condition-row');
        
        rows.forEach((row, index) => {
            const logic = index === 0 ? '' : row.querySelector('.condition-logic-label').textContent;
            conditions.push({
                logic: logic,
                field: row.dataset.field,
                operator: row.querySelector('.badge.bg-secondary').textContent,
                value: row.querySelector('.badge.bg-info').textContent
            });
        });
        
        return conditions;
    },
    
    collectActions: function() {
        const actions = [];
        const rows = document.querySelectorAll('#actionsList .rule-action-row');
        
        rows.forEach(row => {
            actions.push({
                type: row.querySelector('.badge.bg-primary').textContent,
                value: row.querySelector('.flex-grow-1').textContent
            });
        });
        
        return actions;
    },
    
    generateDSL: function(name, conditions, actions, score) {
        let dsl = `RULE "${name}"\n`;
        dsl += "WHEN\n";
        
        conditions.forEach((cond, index) => {
            if (index > 0) {
                dsl += `  ${cond.logic} `;
            }
            dsl += `  ${cond.field} ${cond.operator} ${cond.value}`;
            if (index < conditions.length - 1) {
                dsl += '\n';
            }
        });
        
        dsl += '\nTHEN\n';
        actions.forEach((action, index) => {
            dsl += `  ${action.type.toUpperCase()}(${action.value})`;
            if (index < actions.length - 1) {
                dsl += ';\n';
            }
        });
        
        if (score > 0) {
            dsl += `\nSCORE ${score};`;
        }
        
        return dsl;
    },
    
    saveRule: async function() {
        const dsl = window.dslEditorInstance ? window.dslEditorInstance.getValue() : '';
        
        try {
            const data = await auth.request('/api/v1/admin/risk-rules', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    name: document.getElementById('ruleName').value,
                    description: document.getElementById('ruleDescription').value,
                    dsl: dsl,
                    enabled: document.getElementById('ruleEnabled').checked,
                    priority: parseInt(document.getElementById('rulePriority').value) || 0,
                    risk_score: parseInt(document.getElementById('ruleScore').value) || 0
                })
            });
            
            if (data.code === 0) {
                alert('规则保存成功');
                this.loadRules();
            } else {
                alert('保存失败: ' + (data.message || '未知错误'));
            }
        } catch (error) {
            console.error('保存规则失败:', error);
            alert('保存规则失败');
        }
    },
    
    previewRule: function() {
        const dsl = window.dslEditorInstance ? window.dslEditorInstance.getValue() : '';
        
        try {
            const parsed = this.parseDSL(dsl);
            this.renderPreview(parsed);
        } catch (error) {
            alert('DSL解析错误: ' + error.message);
        }
    },
    
    parseDSL: function(dsl) {
        const rule = {
            name: '',
            conditions: [],
            actions: [],
            score: 0
        };
        
        const lines = dsl.split('\n');
        let currentSection = '';
        
        lines.forEach(line => {
            line = line.trim();
            if (!line) return;
            
            if (line.startsWith('RULE')) {
                const match = line.match(/RULE\s+"([^"]+)"/);
                if (match) {
                    rule.name = match[1];
                }
            } else if (line.startsWith('WHEN')) {
                currentSection = 'WHEN';
            } else if (line.startsWith('THEN')) {
                currentSection = 'THEN';
            } else if (line.startsWith('SCORE')) {
                const match = line.match(/SCORE\s+(\d+)/);
                if (match) {
                    rule.score = parseInt(match[1]);
                }
            } else if (currentSection === 'WHEN') {
                const condition = this.parseConditionLine(line);
                if (condition) {
                    rule.conditions.push(condition);
                }
            } else if (currentSection === 'THEN') {
                const action = this.parseActionLine(line);
                if (action) {
                    rule.actions.push(action);
                }
            }
        });
        
        return rule;
    },
    
    parseConditionLine: function(line) {
        const operators = ['>=', '<=', '!=', '=', '>', '<', 'CONTAINS', 'MATCHES', 'IN'];
        
        for (const op of operators) {
            const idx = line.indexOf(op);
            if (idx !== -1) {
                const field = line.substring(0, idx).trim();
                const value = line.substring(idx + op.length).trim().replace(/^["'\[\]]+|[ "'\[\]]+$/g, '');
                
                return { field, operator: op, value };
            }
        }
        
        return null;
    },
    
    parseActionLine: function(line) {
        const blockMatch = line.match(/BLOCK\((.+)\)/);
        if (blockMatch) {
            return { type: 'block', value: blockMatch[1] };
        }
        
        const alertMatch = line.match(/ALERT\((.+)\)/);
        if (alertMatch) {
            return { type: 'alert', value: alertMatch[1] };
        }
        
        const scoreMatch = line.match(/ADD\s+SCORE\s+(\d+)/);
        if (scoreMatch) {
            return { type: 'add_score', value: scoreMatch[1] };
        }
        
        return null;
    },
    
    renderPreview: function(rule) {
        const previewContainer = document.getElementById('rulePreview');
        if (!previewContainer) return;
        
        previewContainer.innerHTML = `
            <div class="alert alert-info">
                <h6>规则预览: ${rule.name}</h6>
                <hr>
                <p><strong>条件 (${rule.conditions.length}):</strong></p>
                <ul>
                    ${rule.conditions.map(c => `<li>${c.field} ${c.operator} ${c.value}</li>`).join('')}
                </ul>
                <p><strong>动作 (${rule.actions.length}):</strong></p>
                <ul>
                    ${rule.actions.map(a => `<li>${a.type}: ${a.value}</li>`).join('')}
                </ul>
                <p><strong>风险评分:</strong> ${rule.score}</p>
            </div>
        `;
    },
    
    testRule: async function() {
        const dsl = window.dslEditorInstance ? window.dslEditorInstance.getValue() : '';
        const testContext = document.getElementById('testContext').value;
        
        if (!testContext) {
            alert('请输入测试上下文');
            return;
        }
        
        let context;
        try {
            context = JSON.parse(testContext);
        } catch (e) {
            alert('测试上下文必须是有效的JSON');
            return;
        }
        
        try {
            const data = await auth.request('/api/v1/admin/risk-rules/test', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    dsl: dsl,
                    context: context
                })
            });
            
            if (data.code === 0) {
                this.renderTestResult(data.data);
            } else {
                alert('测试失败: ' + (data.message || '未知错误'));
            }
        } catch (error) {
            console.error('测试规则失败:', error);
            alert('测试规则失败');
        }
    },
    
    renderTestResult: function(result) {
        const resultContainer = document.getElementById('testResult');
        if (!resultContainer) return;
        
        const resultClass = result.matched ? 'alert-success' : 'alert-secondary';
        
        resultContainer.innerHTML = `
            <div class="alert ${resultClass}">
                <h6>测试结果: ${result.matched ? '匹配' : '不匹配'}</h6>
                <hr>
                <p><strong>规则:</strong> ${result.rule_name}</p>
                <p><strong>得分:</strong> ${result.score}</p>
                ${result.factors && result.factors.length > 0 ? `
                    <p><strong>匹配因素:</strong></p>
                    <ul>
                        ${result.factors.map(f => `
                            <li>
                                ${f.field}: ${f.matched ? '✓ 匹配' : '✗ 不匹配'}
                                (预期: ${f.expected}, 实际: ${f.actual})
                            </li>
                        `).join('')}
                    </ul>
                ` : ''}
                ${result.actions && result.actions.length > 0 ? `
                    <p><strong>触发的动作:</strong></p>
                    <ul>
                        ${result.actions.map(a => `<li>${a.type}: ${JSON.stringify(a.value)}</li>`).join('')}
                    </ul>
                ` : ''}
            </div>
        `;
    },
    
    validateDSL: function() {
        const dsl = window.dslEditorInstance ? window.dslEditorInstance.getValue() : '';
        
        try {
            const parsed = this.parseDSL(dsl);
            
            if (!parsed.name) {
                throw new Error('规则名称不能为空');
            }
            
            if (parsed.conditions.length === 0) {
                throw new Error('规则必须至少有一个条件');
            }
            
            if (parsed.actions.length === 0) {
                throw new Error('规则必须至少有一个动作');
            }
            
            for (const field of this.availableFields) {
                if (parsed.conditions.some(c => c.field === field.name)) {
                    continue;
                }
            }
            
            document.getElementById('validationResult').innerHTML = `
                <div class="alert alert-success">
                    <i class="fas fa-check-circle me-2"></i>
                    DSL语法验证通过
                </div>
            `;
        } catch (error) {
            document.getElementById('validationResult').innerHTML = `
                <div class="alert alert-danger">
                    <i class="fas fa-times-circle me-2"></i>
                    验证失败: ${error.message}
                </div>
            `;
        }
    },
    
    generateFromDSL: function() {
        const dsl = window.dslEditorInstance ? window.dslEditorInstance.getValue() : '';
        
        try {
            const parsed = this.parseDSL(dsl);
            
            document.getElementById('ruleName').value = parsed.name;
            document.getElementById('ruleScore').value = parsed.score;
            
            const conditionsContainer = document.getElementById('conditionsList');
            conditionsContainer.innerHTML = '';
            
            parsed.conditions.forEach((cond, index) => {
                const logic = index === 0 ? '' : (cond.logic || 'AND');
                const conditionHtml = this.renderConditionRow({
                    ...cond,
                    logic: logic
                });
                conditionsContainer.insertAdjacentHTML('beforeend', conditionHtml);
            });
            
            const actionsContainer = document.getElementById('actionsList');
            actionsContainer.innerHTML = '';
            
            parsed.actions.forEach(action => {
                const actionHtml = this.renderActionRow(action);
                actionsContainer.insertAdjacentHTML('beforeend', actionHtml);
            });
            
            document.getElementById('validationResult').innerHTML = `
                <div class="alert alert-success">
                    <i class="fas fa-check-circle me-2"></i>
                    成功从DSL生成规则编辑器内容
                </div>
            `;
        } catch (error) {
            alert('生成失败: ' + error.message);
        }
    },
    
    loadRules: async function() {
        try {
            const data = await auth.request('/api/v1/admin/risk-rules?page_size=100');
            if (data.code === 0 && data.data) {
                this.rules = data.data.data || [];
                this.renderRulesList();
            }
        } catch (error) {
            console.error('加载规则失败:', error);
        }
    },
    
    renderRulesList: function() {
        const container = document.getElementById('rulesList');
        if (!container) return;
        
        if (this.rules.length === 0) {
            container.innerHTML = '<p class="text-muted">暂无规则</p>';
            return;
        }
        
        container.innerHTML = this.rules.map(rule => `
            <div class="rule-item mb-3 p-3 border rounded" onclick="ruleEditor.editRule(${rule.id})">
                <div class="d-flex justify-content-between align-items-center">
                    <div>
                        <h6>${rule.name || '未命名规则'}</h6>
                        <small class="text-muted">${rule.description || '无描述'}</small>
                    </div>
                    <div>
                        <span class="badge ${rule.enabled ? 'bg-success' : 'bg-secondary'}">
                            ${rule.enabled ? '启用' : '禁用'}
                        </span>
                        <span class="badge bg-primary ms-2">评分: ${rule.risk_score || 0}</span>
                    </div>
                </div>
            </div>
        `).join('');
    },
    
    editRule: function(ruleId) {
        const rule = this.rules.find(r => r.id === ruleId);
        if (!rule) return;
        
        this.currentRule = rule;
        
        document.getElementById('ruleName').value = rule.name || '';
        document.getElementById('ruleDescription').value = rule.description || '';
        document.getElementById('ruleEnabled').checked = rule.enabled !== false;
        document.getElementById('rulePriority').value = rule.priority || 0;
        document.getElementById('ruleScore').value = rule.risk_score || 0;
        
        if (window.dslEditorInstance && rule.dsl) {
            window.dslEditorInstance.setValue(rule.dsl);
        }
        
        $('#ruleEditorModal').modal('show');
    }
};

document.addEventListener('DOMContentLoaded', function() {
    ruleEditor.init();
});
