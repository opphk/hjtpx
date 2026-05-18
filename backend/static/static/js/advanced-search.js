// 高级搜索功能模块
class AdvancedSearch {
    constructor(entityType, config = {}) {
        this.entityType = entityType;
        this.config = {
            apiBase: '/api/admin',
            ...config
        };
        this.conditions = [];
        this.sortOptions = [];
        this.page = 1;
        this.pageSize = 20;
        this.savedSearches = [];
        this.fieldConfigs = this.getFieldConfigs();
    }

    // 获取不同实体的字段配置
    getFieldConfigs() {
        const configs = {
            logs: [
                { name: 'session_id', label: '会话ID', type: 'text' },
                { name: 'status', label: '状态', type: 'select', options: ['success', 'failed', 'pending'] },
                { name: 'captcha_type', label: '验证码类型', type: 'select', options: ['slider', 'click', 'voice'] },
                { name: 'risk_score', label: '风险分数', type: 'number' },
                { name: 'ip_address', label: 'IP地址', type: 'text' },
                { name: 'user_agent', label: 'User Agent', type: 'text' },
                { name: 'duration', label: '耗时(ms)', type: 'number' },
                { name: 'created_at', label: '创建时间', type: 'date' }
            ],
            applications: [
                { name: 'name', label: '应用名称', type: 'text' },
                { name: 'user_id', label: '用户ID', type: 'number' },
                { name: 'domain', label: '域名', type: 'text' },
                { name: 'website', label: '网站', type: 'text' },
                { name: 'is_active', label: '是否激活', type: 'boolean' },
                { name: 'created_at', label: '创建时间', type: 'date' }
            ],
            blacklist: [
                { name: 'target', label: '目标', type: 'text' },
                { name: 'type', label: '类型', type: 'select', options: ['ip', 'email', 'phone', 'device'] },
                { name: 'source', label: '来源', type: 'text' },
                { name: 'reason', label: '原因', type: 'text' },
                { name: 'action', label: '操作', type: 'select', options: ['block', 'warn'] },
                { name: 'status', label: '状态', type: 'select', options: ['active', 'inactive'] },
                { name: 'created_at', label: '创建时间', type: 'date' }
            ]
        };
        return configs[this.entityType] || [];
    }

    // 获取操作符列表
    getOperators(fieldType) {
        const operators = {
            text: [
                { value: 'eq', label: '等于' },
                { value: 'contains', label: '包含' },
                { value: 'starts_with', label: '开头为' },
                { value: 'ends_with', label: '结尾为' }
            ],
            number: [
                { value: 'eq', label: '等于' },
                { value: 'ne', label: '不等于' },
                { value: 'gt', label: '大于' },
                { value: 'gte', label: '大于等于' },
                { value: 'lt', label: '小于' },
                { value: 'lte', label: '小于等于' },
                { value: 'between', label: '在...之间' }
            ],
            select: [
                { value: 'eq', label: '等于' },
                { value: 'ne', label: '不等于' },
                { value: 'in', label: '在列表中' }
            ],
            boolean: [
                { value: 'eq', label: '等于' }
            ],
            date: [
                { value: 'eq', label: '等于' },
                { value: 'gt', label: '晚于' },
                { value: 'lt', label: '早于' },
                { value: 'between', label: '在...之间' }
            ]
        };
        return operators[fieldType] || operators.text;
    }

    // 添加搜索条件
    addCondition(field, operator, value) {
        this.conditions.push({ field, operator, value });
        this.renderConditions();
    }

    // 删除搜索条件
    removeCondition(index) {
        this.conditions.splice(index, 1);
        this.renderConditions();
    }

    // 清空所有条件
    clearConditions() {
        this.conditions = [];
        this.renderConditions();
    }

    // 添加排序选项
    addSortOption(field, order = 'desc') {
        this.sortOptions.push({ field, order });
        this.renderSortOptions();
    }

    // 删除排序选项
    removeSortOption(index) {
        this.sortOptions.splice(index, 1);
        this.renderSortOptions();
    }

    // 构建搜索查询
    buildQuery() {
        return {
            conditions: this.conditions,
            sort: this.sortOptions,
            page: this.page,
            pageSize: this.pageSize
        };
    }

    // 执行搜索
    async search() {
        try {
            const query = this.buildQuery();
            const endpoint = `${this.config.apiBase}/${this.entityType}/advanced-search`;
            const response = await fetch(endpoint, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(query)
            });
            const result = await response.json();
            
            if (result.code === 0) {
                this.onSearchSuccess(result.data);
            } else {
                this.onSearchError(result.message);
            }
        } catch (error) {
            this.onSearchError(error.message);
        }
    }

    // 搜索成功回调
    onSearchSuccess(data) {
        console.log('搜索成功:', data);
        // 子类可以重写此方法
    }

    // 搜索失败回调
    onSearchError(message) {
        console.error('搜索失败:', message);
        alert('搜索失败: ' + message);
    }

    // 保存搜索
    async saveSearch(name, description = '') {
        try {
            const query = this.buildQuery();
            const endpoint = `${this.config.apiBase}/${this.entityType}/save-search`;
            const response = await fetch(endpoint, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ name, description, query })
            });
            const result = await response.json();
            
            if (result.code === 0) {
                alert('保存成功!');
                await this.loadSavedSearches();
            } else {
                alert('保存失败: ' + result.message);
            }
        } catch (error) {
            alert('保存失败: ' + error.message);
        }
    }

    // 加载保存的搜索
    async loadSavedSearches() {
        try {
            const endpoint = `${this.config.apiBase}/${this.entityType}/saved-searches`;
            const response = await fetch(endpoint);
            const result = await response.json();
            
            if (result.code === 0) {
                this.savedSearches = result.data;
                this.renderSavedSearches();
            }
        } catch (error) {
            console.error('加载保存的搜索失败:', error);
        }
    }

    // 删除保存的搜索
    async deleteSavedSearch(id) {
        if (!confirm('确定要删除这个搜索吗?')) return;
        
        try {
            const endpoint = `${this.config.apiBase}/${this.entityType}/saved-searches/${id}`;
            const response = await fetch(endpoint, { method: 'DELETE' });
            const result = await response.json();
            
            if (result.code === 0) {
                alert('删除成功!');
                await this.loadSavedSearches();
            } else {
                alert('删除失败: ' + result.message);
            }
        } catch (error) {
            alert('删除失败: ' + error.message);
        }
    }

    // 应用保存的搜索
    applySavedSearch(savedSearch) {
        try {
            const query = JSON.parse(savedSearch.query);
            this.conditions = query.conditions || [];
            this.sortOptions = query.sort || [];
            this.page = query.page || 1;
            this.pageSize = query.pageSize || 20;
            
            this.renderConditions();
            this.renderSortOptions();
            this.search();
        } catch (error) {
            console.error('应用保存的搜索失败:', error);
            alert('应用搜索失败');
        }
    }

    // 渲染搜索条件
    renderConditions(containerId = 'search-conditions') {
        const container = document.getElementById(containerId);
        if (!container) return;
        
        container.innerHTML = this.conditions.map((cond, index) => {
            const fieldConfig = this.fieldConfigs.find(f => f.name === cond.field);
            return `
                <div class="condition-item mb-2 p-2 bg-light rounded">
                    <span class="me-2">${fieldConfig?.label || cond.field}</span>
                    <span class="me-2 badge bg-secondary">${this.getOperatorLabel(cond.operator)}</span>
                    <span class="me-2">${this.formatValue(cond.value)}</span>
                    <button class="btn btn-sm btn-danger" onclick="search.removeCondition(${index})">
                        <i class="bi bi-trash"></i>
                    </button>
                </div>
            `;
        }).join('');
    }

    // 渲染排序选项
    renderSortOptions(containerId = 'sort-options') {
        const container = document.getElementById(containerId);
        if (!container) return;
        
        container.innerHTML = this.sortOptions.map((sort, index) => `
            <div class="sort-item mb-2 p-2 bg-light rounded">
                <span class="me-2">${sort.field}</span>
                <span class="me-2 badge bg-info">${sort.order === 'asc' ? '升序' : '降序'}</span>
                <button class="btn btn-sm btn-danger" onclick="search.removeSortOption(${index})">
                    <i class="bi bi-trash"></i>
                </button>
            </div>
        `).join('');
    }

    // 渲染保存的搜索列表
    renderSavedSearches(containerId = 'saved-searches') {
        const container = document.getElementById(containerId);
        if (!container) return;
        
        container.innerHTML = this.savedSearches.map(search => `
            <div class="saved-search-item mb-2 p-2 border rounded">
                <div class="d-flex justify-content-between align-items-center">
                    <div>
                        <strong>${search.name}</strong>
                        ${search.description ? `<small class="text-muted d-block">${search.description}</small>` : ''}
                    </div>
                    <div>
                        <button class="btn btn-sm btn-primary me-2" onclick='search.applySavedSearch(${JSON.stringify(search).replace(/'/g, "\\'")})'>
                            应用
                        </button>
                        <button class="btn btn-sm btn-danger" onclick="search.deleteSavedSearch(${search.id})">
                            删除
                        </button>
                    </div>
                </div>
            </div>
        `).join('');
    }

    // 获取操作符标签
    getOperatorLabel(operator) {
        const labels = {
            eq: '等于',
            ne: '不等于',
            gt: '大于',
            gte: '大于等于',
            lt: '小于',
            lte: '小于等于',
            contains: '包含',
            starts_with: '开头为',
            ends_with: '结尾为',
            in: '在列表中',
            not_in: '不在列表中',
            is_null: '为空',
            is_not_null: '不为空',
            between: '在...之间'
        };
        return labels[operator] || operator;
    }

    // 格式化值
    formatValue(value) {
        if (Array.isArray(value)) {
            return value.join(' ~ ');
        }
        return String(value);
    }

    // 渲染添加条件表单
    renderAddConditionForm(containerId = 'add-condition-form') {
        const container = document.getElementById(containerId);
        if (!container) return;
        
        container.innerHTML = `
            <div class="row g-2">
                <div class="col-md-3">
                    <select class="form-select" id="condition-field">
                        <option value="">选择字段</option>
                        ${this.fieldConfigs.map(field => `
                            <option value="${field.name}" data-type="${field.type}">${field.label}</option>
                        `).join('')}
                    </select>
                </div>
                <div class="col-md-2">
                    <select class="form-select" id="condition-operator" disabled>
                        <option value="">选择操作符</option>
                    </select>
                </div>
                <div class="col-md-5" id="condition-value-container">
                    <input type="text" class="form-control" id="condition-value" placeholder="输入值" disabled>
                </div>
                <div class="col-md-2">
                    <button class="btn btn-primary w-100" id="add-condition-btn" disabled>
                        <i class="bi bi-plus"></i> 添加
                    </button>
                </div>
            </div>
        `;
        
        this.bindAddConditionEvents();
    }

    // 绑定添加条件事件
    bindAddConditionEvents() {
        const fieldSelect = document.getElementById('condition-field');
        const operatorSelect = document.getElementById('condition-operator');
        const valueContainer = document.getElementById('condition-value-container');
        const addBtn = document.getElementById('add-condition-btn');
        
        fieldSelect.addEventListener('change', () => {
            const selectedOption = fieldSelect.options[fieldSelect.selectedIndex];
            const fieldType = selectedOption.getAttribute('data-type');
            const fieldName = fieldSelect.value;
            
            if (!fieldName) {
                operatorSelect.disabled = true;
                addBtn.disabled = true;
                return;
            }
            
            // 更新操作符选项
            const operators = this.getOperators(fieldType);
            operatorSelect.innerHTML = `
                <option value="">选择操作符</option>
                ${operators.map(op => `<option value="${op.value}">${op.label}</option>`).join('')}
            `;
            operatorSelect.disabled = false;
            
            // 更新值输入框
            this.updateValueInput(fieldType, valueContainer);
        });
        
        operatorSelect.addEventListener('change', () => {
            const operator = operatorSelect.value;
            const valueInput = document.getElementById('condition-value');
            
            if (!operator) {
                valueInput.disabled = true;
                addBtn.disabled = true;
                return;
            }
            
            valueInput.disabled = false;
            addBtn.disabled = false;
        });
        
        addBtn.addEventListener('click', () => {
            const field = fieldSelect.value;
            const operator = operatorSelect.value;
            let value = this.getValueFromInput();
            
            if (!field || !operator) return;
            
            this.addCondition(field, operator, value);
            
            // 重置表单
            fieldSelect.value = '';
            operatorSelect.innerHTML = '<option value="">选择操作符</option>';
            operatorSelect.disabled = true;
            valueContainer.innerHTML = '<input type="text" class="form-control" id="condition-value" placeholder="输入值" disabled>';
            addBtn.disabled = true;
        });
    }

    // 更新值输入框
    updateValueInput(fieldType, container) {
        switch (fieldType) {
            case 'select':
                const fieldConfig = this.fieldConfigs.find(f => f.name === document.getElementById('condition-field').value);
                container.innerHTML = `
                    <select class="form-select" id="condition-value">
                        <option value="">选择值</option>
                        ${(fieldConfig?.options || []).map(opt => `<option value="${opt}">${opt}</option>`).join('')}
                    </select>
                `;
                break;
            case 'boolean':
                container.innerHTML = `
                    <select class="form-select" id="condition-value">
                        <option value="true">是</option>
                        <option value="false">否</option>
                    </select>
                `;
                break;
            case 'date':
                container.innerHTML = `
                    <div class="row g-2">
                        <div class="col-md-6">
                            <input type="date" class="form-control" id="condition-value-start">
                        </div>
                        <div class="col-md-6">
                            <input type="date" class="form-control" id="condition-value-end" style="display:none;">
                        </div>
                    </div>
                `;
                break;
            case 'number':
                container.innerHTML = `
                    <div class="row g-2">
                        <div class="col-md-6">
                            <input type="number" class="form-control" id="condition-value-start" placeholder="数值">
                        </div>
                        <div class="col-md-6">
                            <input type="number" class="form-control" id="condition-value-end" placeholder="结束值" style="display:none;">
                        </div>
                    </div>
                `;
                break;
            default:
                container.innerHTML = '<input type="text" class="form-control" id="condition-value" placeholder="输入值">';
        }
        
        // 监听操作符变化来显示/隐藏第二个输入框
        document.getElementById('condition-operator').addEventListener('change', (e) => {
            const operator = e.target.value;
            const endInput = document.getElementById('condition-value-end');
            if (endInput) {
                endInput.style.display = operator === 'between' ? 'block' : 'none';
            }
        });
    }

    // 从输入获取值
    getValueFromInput() {
        const operator = document.getElementById('condition-operator').value;
        
        if (operator === 'between') {
            const startInput = document.getElementById('condition-value-start');
            const endInput = document.getElementById('condition-value-end');
            return [startInput?.value || '', endInput?.value || ''];
        }
        
        const singleInput = document.getElementById('condition-value');
        const startInput = document.getElementById('condition-value-start');
        const input = singleInput || startInput;
        
        return input ? input.value : '';
    }

    // 初始化
    async init() {
        await this.loadSavedSearches();
        this.renderAddConditionForm();
    }
}

// 全局实例（在页面中使用时需要创建具体实例）
let search = null;

// 初始化高级搜索
function initAdvancedSearch(entityType) {
    search = new AdvancedSearch(entityType);
    search.init();
    return search;
}
