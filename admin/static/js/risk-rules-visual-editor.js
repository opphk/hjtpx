class RiskRuleEditor {
    constructor() {
        this.rules = [];
        this.selectedComponent = null;
        this.ruleCounter = 0;
        this.init();
    }
    
    init() {
        this.setupDragAndDrop();
        this.setupEventListeners();
    }
    
    setupDragAndDrop() {
        const editor = document.getElementById('ruleEditor');
        
        // 拖拽开始
        document.querySelectorAll('[draggable="true"]').forEach(item => {
            item.addEventListener('dragstart', (e) => {
                e.dataTransfer.setData('type', item.dataset.type);
                e.dataTransfer.setData('field', item.dataset.field);
                e.dataTransfer.effectAllowed = 'copy';
            });
        });
        
        // 拖拽结束
        editor.addEventListener('dragover', (e) => {
            e.preventDefault();
            e.dataTransfer.dropEffect = 'copy';
        });
        
        // 放置
        editor.addEventListener('drop', (e) => {
            e.preventDefault();
            const type = e.dataTransfer.getData('type');
            const field = e.dataTransfer.getData('field');
            
            if (type) {
                this.addComponent(type, field);
            }
        });
    }
    
    setupEventListeners() {
        // 点击组件时选中
        document.getElementById('ruleTree').addEventListener('click', (e) => {
            const component = e.target.closest('.condition-block, .action-block');
            if (component) {
                this.selectComponent(component);
            }
        });
        
        // 删除按钮
        document.getElementById('ruleTree').addEventListener('click', (e) => {
            if (e.target.classList.contains('delete-btn')) {
                e.target.closest('.condition-block, .action-block').remove();
                this.updatePreview();
            }
        });
    }
    
    addComponent(type, field) {
        const ruleTree = document.getElementById('ruleTree');
        const id = `rule-${++this.ruleCounter}`;
        
        let html = '';
        
        switch(type) {
            case 'condition':
                html = this.createConditionBlock(id, field);
                break;
            case 'operator':
                html = this.createOperatorBlock(id);
                break;
            case 'action':
                html = this.createActionBlock(id);
                break;
            case 'group':
                html = this.createGroupBlock(id);
                break;
            case 'threshold':
                html = this.createThresholdBlock(id, field);
                break;
        }
        
        ruleTree.insertAdjacentHTML('beforeend', html);
        this.updatePreview();
    }
    
    createConditionBlock(id, field) {
        return `
            <div class="condition-block" id="${id}" data-type="condition">
                <div class="d-flex align-items-center">
                    <span class="drag-handle"><i class="fas fa-grip-vertical"></i></span>
                    <select class="form-select form-select-sm" style="width: auto;">
                        <option value="field">字段</option>
                        <option value="value">值</option>
                    </select>
                    <span class="operator-select">${field || '选择字段'}</span>
                    <select class="form-select form-select-sm" style="width: auto;">
                        <option value="eq">等于</option>
                        <option value="ne">不等于</option>
                        <option value="gt">大于</option>
                        <option value="lt">小于</option>
                        <option value="gte">大于等于</option>
                        <option value="lte">小于等于</option>
                        <option value="contains">包含</option>
                        <option value="regex">正则匹配</option>
                    </select>
                    <input type="text" class="form-control form-control-sm" style="width: 150px;" placeholder="值">
                    <button class="btn btn-sm btn-outline-danger delete-btn ml-2">
                        <i class="fas fa-times"></i>
                    </button>
                </div>
            </div>
        `;
    }
    
    createOperatorBlock(id) {
        return `
            <div class="text-center my-2">
                <span class="badge bg-primary operator-select">AND</span>
            </div>
        `;
    }
    
    createActionBlock(id) {
        return `
            <div class="action-block" id="${id}" data-type="action">
                <div class="d-flex align-items-center">
                    <span class="drag-handle"><i class="fas fa-grip-vertical"></i></span>
                    <span class="me-2">执行动作:</span>
                    <select class="form-select form-select-sm" style="width: auto;">
                        <option value="block">阻止</option>
                        <option value="challenge">挑战验证</option>
                        <option value="allow">允许</option>
                        <option value="log">记录日志</option>
                        <option value="notify">发送通知</option>
                    </select>
                    <button class="btn btn-sm btn-outline-danger delete-btn ml-2">
                        <i class="fas fa-times"></i>
                    </button>
                </div>
            </div>
        `;
    }
    
    createGroupBlock(id) {
        return `
            <div class="condition-block border border-info" id="${id}" data-type="group">
                <div class="d-flex align-items-center mb-2">
                    <span class="drag-handle"><i class="fas fa-grip-vertical"></i></span>
                    <span class="badge bg-info">条件组</span>
                    <select class="form-select form-select-sm ms-2" style="width: auto;">
                        <option value="and">全部满足 (AND)</option>
                        <option value="or">任一满足 (OR)</option>
                    </select>
                    <button class="btn btn-sm btn-outline-danger delete-btn ml-auto">
                        <i class="fas fa-times"></i>
                    </button>
                </div>
                <div class="ms-4">
                    <!-- 子条件将在这里 -->
                </div>
            </div>
        `;
    }
    
    createThresholdBlock(id, field) {
        return `
            <div class="condition-block" id="${id}" data-type="threshold">
                <div class="d-flex align-items-center">
                    <span class="drag-handle"><i class="fas fa-grip-vertical"></i></span>
                    <span>阈值:</span>
                    <select class="form-select form-select-sm ms-2" style="width: 150px;">
                        <option value="request_count">请求次数</option>
                        <option value="failure_rate">失败率</option>
                        <option value="response_time">响应时间</option>
                        <option value="error_rate">错误率</option>
                    </select>
                    <select class="form-select form-select-sm ms-2" style="width: auto;">
                        <option value="gt">大于</option>
                        <option value="lt">小于</option>
                        <option value="eq">等于</option>
                    </select>
                    <input type="number" class="form-control form-control-sm ms-2" style="width: 100px;" placeholder="阈值">
                    <span class="ms-2">在</span>
                    <input type="number" class="form-control form-control-sm ms-2" style="width: 80px;" placeholder="时间">
                    <select class="form-select form-select-sm ms-2" style="width: auto;">
                        <option value="minute">分钟内</option>
                        <option value="hour">小时内</option>
                        <option value="day">天内</option>
                    </select>
                    <button class="btn btn-sm btn-outline-danger delete-btn ml-2">
                        <i class="fas fa-times"></i>
                    </button>
                </div>
            </div>
        `;
    }
    
    selectComponent(component) {
        // 移除之前的选中状态
        document.querySelectorAll('.selected').forEach(el => {
            el.classList.remove('selected');
        });
        
        // 添加选中状态
        component.classList.add('selected');
        this.selectedComponent = component;
        
        // 更新属性面板
        this.updatePropertyPanel(component);
    }
    
    updatePropertyPanel(component) {
        const panel = document.getElementById('propertyPanel');
        const type = component.dataset.type;
        
        let html = '';
        
        switch(type) {
            case 'condition':
                html = `
                    <div class="mb-3">
                        <label>字段类型</label>
                        <select class="form-select">
                            <option value="string">字符串</option>
                            <option value="number">数字</option>
                            <option value="boolean">布尔值</option>
                        </select>
                    </div>
                    <div class="mb-3">
                        <label>必填</label>
                        <input type="checkbox" checked>
                    </div>
                `;
                break;
            case 'action':
                html = `
                    <div class="mb-3">
                        <label>动作优先级</label>
                        <input type="number" class="form-control" value="1">
                    </div>
                `;
                break;
        }
        
        panel.innerHTML = html;
    }
    
    updatePreview() {
        const ruleTree = document.getElementById('ruleTree');
        const components = ruleTree.querySelectorAll('.condition-block, .action-block');
        
        let code = '{\n  "rules": [\n';
        
        components.forEach((comp, index) => {
            const type = comp.dataset.type;
            code += `    {\n`;
            code += `      "type": "${type}",\n`;
            
            if (type === 'condition') {
                code += `      "field": "example_field",\n`;
                code += `      "operator": "eq",\n`;
                code += `      "value": "example_value"\n`;
            } else if (type === 'action') {
                code += `      "action": "block"\n`;
            }
            
            code += `    }`;
            if (index < components.length - 1) {
                code += ',';
            }
            code += '\n';
        });
        
        code += '  ]\n}';
        
        document.getElementById('ruleCode').textContent = code;
    }
    
    exportRule() {
        const ruleTree = document.getElementById('ruleTree');
        const components = ruleTree.querySelectorAll('.condition-block, .action-block');
        
        const rules = [];
        components.forEach(comp => {
            const type = comp.dataset.type;
            rules.push({
                type: type,
                data: this.extractComponentData(comp)
            });
        });
        
        return JSON.stringify({rules: rules}, null, 2);
    }
    
    extractComponentData(component) {
        const selects = component.querySelectorAll('select');
        const inputs = component.querySelectorAll('input');
        
        const data = {};
        
        selects.forEach((select, index) => {
            data[`field${index}`] = select.value;
        });
        
        inputs.forEach((input, index) => {
            data[`value${index}`] = input.value;
        });
        
        return data;
    }
}

// 全局函数
let editor;

document.addEventListener('DOMContentLoaded', () => {
    editor = new RiskRuleEditor();
});

function saveRule() {
    const rule = editor.exportRule();
    console.log('保存规则:', rule);
    alert('规则已保存');
}

function testRule() {
    const rule = editor.exportRule();
    console.log('测试规则:', rule);
    alert('规则测试中...');
}

function clearRule() {
    if (confirm('确定要清空所有规则吗？')) {
        document.getElementById('ruleTree').innerHTML = '';
        editor.updatePreview();
    }
}
