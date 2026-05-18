// 风控规则引擎 - 扩展功能
const RiskRulesEngine = {
    templates: [],
    rules: [],
    performanceData: {},
    triggerHistory: [],
    auditLogs: [],
    
    async init() {
        await this.loadTemplates();
        await this.loadRules();
        await this.loadPerformanceData();
        this.bindEvents();
    },
    
    // 加载规则模板
    async loadTemplates() {
        try {
            const response = await fetch('/admin/api/risk-templates');
            const result = await response.json();
            if (result.code === 0) {
                this.templates = result.data.items || [];
                this.renderTemplates();
            }
        } catch (error) {
            console.error('加载规则模板失败:', error);
            this.loadDefaultTemplates();
        }
    },
    
    loadDefaultTemplates() {
        this.templates = [
            {
                id: 'extreme_speed',
                name: '极端速度检测',
                description: '检测异常快速的滑动行为',
                rule_type: 'behavior',
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
                rule_type: 'behavior',
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
                rule_type: 'behavior',
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
                rule_type: 'rate_limit',
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
                rule_type: 'device_fingerprint',
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
                rule_type: 'ip_block',
                severity: 'high',
                conditions: [
                    { field: 'is_proxy', operator: 'eq', value: true }
                ],
                action: 'block'
            }
        ];
        this.renderTemplates();
    },
    
    renderTemplates() {
        const container = document.getElementById('templatesContainer');
        if (!container) return;
        
        container.innerHTML = this.templates.map(template => `
            <div class="card mb-2">
                <div class="card-body">
                    <div class="d-flex justify-content-between align-items-start">
                        <div>
                            <h6 class="mb-1">${this.escapeHtml(template.name)}</h6>
                            <p class="text-muted small mb-2">${this.escapeHtml(template.description)}</p>
                            <span class="badge bg-${this.getSeverityColor(template.severity)} me-1">
                                ${this.getSeverityText(template.severity)}
                            </span>
                            <span class="badge bg-secondary">${template.rule_type}</span>
                        </div>
                        <div class="btn-group btn-group-sm">
                            <button class="btn btn-outline-primary" onclick="RiskRulesEngine.applyTemplate('${template.id}')">
                                <i class="fas fa-check"></i> 应用
                            </button>
                            <button class="btn btn-outline-secondary" onclick="RiskRulesEngine.viewTemplate('${template.id}')">
                                <i class="fas fa-eye"></i>
                            </button>
                        </div>
                    </div>
                </div>
            </div>
        `).join('');
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
            }
        } catch (error) {
            console.error('应用模板失败:', error);
            alert('应用模板失败，请重试');
        }
    },
    
    viewTemplate(templateId) {
        const template = this.templates.find(t => t.id === templateId || t.id === parseInt(templateId));
        if (!template) return;
        
        alert(`模板详情:\n名称: ${template.name}\n描述: ${template.description}\n类型: ${template.rule_type}\n严重度: ${template.severity}`);
    },
    
    // 加载规则列表
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
            <tr>
                <td><input type="checkbox" class="rule-checkbox" data-id="${rule.id}"></td>
                <td>
                    <strong>${this.escapeHtml(rule.name)}</strong>
                    <div class="text-muted small">${this.escapeHtml(rule.description || '')}</div>
                </td>
                <td><span class="badge bg-secondary">${rule.rule_type}</span></td>
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
    
    // 性能分析
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
    
    // 触发历史
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
    
    // 审计日志
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
    
    // 工具方法
    getSeverityColor(severity) {
        const colors = { low: 'success', medium: 'warning', high: 'danger' };
        return colors[severity] || 'secondary';
    },
    
    getSeverityText(severity) {
        const texts = { low: '低', medium: '中', high: '高' };
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
    
    escapeHtml(text) {
        if (text === null || text === undefined) return '';
        const div = document.createElement('div');
        div.textContent = String(text);
        return div.innerHTML;
    },
    
    bindEvents() {
        // 绑定按钮事件
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
    }
};

// 页面加载完成后初始化
document.addEventListener('DOMContentLoaded', () => {
    RiskRulesEngine.init();
});
