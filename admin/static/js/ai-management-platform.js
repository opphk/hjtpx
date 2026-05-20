class AIManagementPlatform {
    constructor() {
        this.currentTab = 'models';
        this.models = [];
        this.experiments = [];
        this.abTests = [];
        this.monitors = [];
        this.charts = {};
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.loadInitialData();
        this.setupTabs();
    }

    setupEventListeners() {
        document.getElementById('refreshBtn')?.addEventListener('click', () => this.refreshData());
        document.getElementById('createModelBtn')?.addEventListener('click', () => this.showCreateModelModal());
        document.getElementById('createExperimentBtn')?.addEventListener('click', () => this.showCreateExperimentModal());
        document.getElementById('createABTestBtn')?.addEventListener('click', () => this.showCreateABTestModal());
    }

    setupTabs() {
        const tabs = document.querySelectorAll('.nav-tabs .nav-link');
        tabs.forEach(tab => {
            tab.addEventListener('click', (e) => {
                e.preventDefault();
                const tabId = tab.getAttribute('data-tab');
                this.switchTab(tabId);
            });
        });
    }

    switchTab(tabId) {
        this.currentTab = tabId;
        
        document.querySelectorAll('.nav-tabs .nav-link').forEach(t => t.classList.remove('active'));
        document.querySelector(`[data-tab="${tabId}"]`)?.classList.add('active');
        
        document.querySelectorAll('.tab-pane').forEach(pane => pane.classList.remove('show', 'active'));
        document.getElementById(tabId)?.classList.add('show', 'active');
        
        this.loadTabData(tabId);
    }

    loadInitialData() {
        this.loadTabData('models');
    }

    async loadTabData(tabId) {
        switch (tabId) {
            case 'models':
                await this.loadModels();
                break;
            case 'experiments':
                await this.loadExperiments();
                break;
            case 'abtesting':
                await this.loadABTests();
                break;
            case 'monitoring':
                await this.loadMonitoring();
                break;
        }
    }

    async loadModels() {
        try {
            const response = await fetch('/api/v1/ai/models');
            if (response.ok) {
                this.models = await response.json();
            } else {
                this.models = this.getMockModels();
            }
        } catch (error) {
            console.error('Failed to load models:', error);
            this.models = this.getMockModels();
        }
        this.renderModels();
    }

    getMockModels() {
        return [
            {
                id: 1,
                name: 'Captcha Classifier',
                version: 'v2.1.0',
                description: '图像验证码分类模型',
                status: 'deployed',
                type: 'classification',
                createdAt: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
                metadata: { accuracy: 0.98, latency: 45.5 }
            },
            {
                id: 2,
                name: 'Risk Detector',
                version: 'v1.3.2',
                description: '风险检测模型',
                status: 'deployed',
                type: 'detection',
                createdAt: new Date(Date.now() - 60 * 24 * 60 * 60 * 1000).toISOString(),
                metadata: { precision: 0.95, recall: 0.92 }
            },
            {
                id: 3,
                name: 'Behavior Analyzer',
                version: 'v0.8.0',
                description: '行为分析模型（测试版）',
                status: 'draft',
                type: 'analysis',
                createdAt: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString()
            }
        ];
    }

    renderModels() {
        const container = document.getElementById('modelsList');
        if (!container) return;

        container.innerHTML = this.models.map(model => `
            <div class="card mb-3 model-card" data-id="${model.id}">
                <div class="card-body">
                    <div class="d-flex justify-content-between align-items-start">
                        <div>
                            <h5 class="card-title mb-1">
                                <i class="fas fa-brain me-2 text-primary"></i>
                                ${model.name}
                            </h5>
                            <h6 class="card-subtitle text-muted mb-2">
                                v${model.version} • ${model.type}
                            </h6>
                            <p class="card-text text-muted small">${model.description}</p>
                        </div>
                        <div class="text-end">
                            <span class="badge ${this.getStatusBadgeClass(model.status)}">
                                ${this.getStatusText(model.status)}
                            </span>
                        </div>
                    </div>
                    ${model.metadata ? `
                    <div class="mt-3 d-flex gap-4">
                        ${Object.entries(model.metadata).map(([key, value]) => `
                            <div class="text-center">
                                <div class="fw-bold">${typeof value === 'number' ? (value * 100).toFixed(1) + '%' : value}</div>
                                <div class="text-muted small">${key}</div>
                            </div>
                        `).join('')}
                    </div>
                    ` : ''}
                    <div class="mt-3 d-flex gap-2">
                        <button class="btn btn-sm btn-outline-primary" onclick="aiPlatform.viewModel(${model.id})">
                            <i class="fas fa-eye"></i> 查看
                        </button>
                        <button class="btn btn-sm btn-outline-success" onclick="aiPlatform.deployModel(${model.id})">
                            <i class="fas fa-rocket"></i> 部署
                        </button>
                        <button class="btn btn-sm btn-outline-danger" onclick="aiPlatform.deleteModel(${model.id})">
                            <i class="fas fa-trash"></i> 删除
                        </button>
                    </div>
                </div>
            </div>
        `).join('');
    }

    async loadExperiments() {
        try {
            const response = await fetch('/api/v1/ai/experiments');
            if (response.ok) {
                this.experiments = await response.json();
            } else {
                this.experiments = this.getMockExperiments();
            }
        } catch (error) {
            console.error('Failed to load experiments:', error);
            this.experiments = this.getMockExperiments();
        }
        this.renderExperiments();
    }

    getMockExperiments() {
        return [
            {
                id: 1,
                name: 'Captcha Model Optimization',
                description: '优化验证码分类模型准确率和性能',
                type: 'optimization',
                status: 'running',
                createdBy: 'admin',
                createdAt: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
                tags: ['captcha', 'classification', 'optimization']
            },
            {
                id: 2,
                name: 'Risk Detection Model v2',
                description: '开发新一代风险检测模型',
                type: 'research',
                status: 'completed',
                createdBy: 'researcher',
                createdAt: new Date(Date.now() - 60 * 24 * 60 * 60 * 1000).toISOString(),
                tags: ['risk', 'detection', 'research']
            }
        ];
    }

    renderExperiments() {
        const container = document.getElementById('experimentsList');
        if (!container) return;

        container.innerHTML = this.experiments.map(exp => `
            <div class="card mb-3 experiment-card" data-id="${exp.id}">
                <div class="card-body">
                    <div class="d-flex justify-content-between align-items-start">
                        <div>
                            <h5 class="card-title mb-1">
                                <i class="fas fa-flask me-2 text-success"></i>
                                ${exp.name}
                            </h5>
                            <h6 class="card-subtitle text-muted mb-2">
                                ${exp.type} • ${exp.createdBy}
                            </h6>
                            <p class="card-text text-muted small">${exp.description}</p>
                            ${exp.tags ? `
                            <div class="mt-2">
                                ${exp.tags.map(tag => `<span class="badge bg-secondary me-1">${tag}</span>`).join('')}
                            </div>
                            ` : ''}
                        </div>
                        <div class="text-end">
                            <span class="badge ${this.getStatusBadgeClass(exp.status)}">
                                ${this.getStatusText(exp.status)}
                            </span>
                        </div>
                    </div>
                    <div class="mt-3 d-flex gap-2">
                        <button class="btn btn-sm btn-outline-primary" onclick="aiPlatform.viewExperiment(${exp.id})">
                            <i class="fas fa-eye"></i> 查看
                        </button>
                        <button class="btn btn-sm btn-outline-success" onclick="aiPlatform.startExperiment(${exp.id})">
                            <i class="fas fa-play"></i> 开始
                        </button>
                    </div>
                </div>
            </div>
        `).join('');
    }

    async loadABTests() {
        try {
            const response = await fetch('/api/v1/ai/ab-tests');
            if (response.ok) {
                this.abTests = await response.json();
            } else {
                this.abTests = this.getMockABTests();
            }
        } catch (error) {
            console.error('Failed to load A/B tests:', error);
            this.abTests = this.getMockABTests();
        }
        this.renderABTests();
    }

    getMockABTests() {
        return [
            {
                id: 1,
                name: 'Captcha Model v2.1 vs v2.0',
                description: '测试新版验证码分类模型性能',
                status: 'running',
                modelId: 1,
                createdAt: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString()
            },
            {
                id: 2,
                name: 'Risk Detector Threshold Test',
                description: '测试不同阈值下的风险检测效果',
                status: 'completed',
                modelId: 2,
                createdAt: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString()
            }
        ];
    }

    renderABTests() {
        const container = document.getElementById('abTestsList');
        if (!container) return;

        container.innerHTML = this.abTests.map(test => `
            <div class="card mb-3 abtest-card" data-id="${test.id}">
                <div class="card-body">
                    <div class="d-flex justify-content-between align-items-start">
                        <div>
                            <h5 class="card-title mb-1">
                                <i class="fas fa-code-branch me-2 text-warning"></i>
                                ${test.name}
                            </h5>
                            <p class="card-text text-muted small">${test.description}</p>
                        </div>
                        <div class="text-end">
                            <span class="badge ${this.getStatusBadgeClass(test.status)}">
                                ${this.getStatusText(test.status)}
                            </span>
                        </div>
                    </div>
                    <div class="mt-3 d-flex gap-2">
                        <button class="btn btn-sm btn-outline-primary" onclick="aiPlatform.viewABTest(${test.id})">
                            <i class="fas fa-chart-bar"></i> 分析
                        </button>
                        ${test.status === 'running' ? `
                        <button class="btn btn-sm btn-outline-danger" onclick="aiPlatform.stopABTest(${test.id})">
                            <i class="fas fa-stop"></i> 停止
                        </button>
                        ` : ''}
                    </div>
                </div>
            </div>
        `).join('');
    }

    async loadMonitoring() {
        try {
            const response = await fetch('/api/v1/ai/monitors');
            if (response.ok) {
                this.monitors = await response.json();
            } else {
                this.monitors = this.getMockMonitors();
            }
        } catch (error) {
            console.error('Failed to load monitors:', error);
            this.monitors = this.getMockMonitors();
        }
        this.renderMonitoring();
        this.initMonitoringCharts();
    }

    getMockMonitors() {
        return [
            {
                id: 1,
                modelId: 1,
                modelName: 'Captcha Classifier',
                status: 'healthy',
                lastCheck: new Date().toISOString(),
                cpuUsage: 34.5,
                memoryUsage: 42.8,
                throughput: 1250.0,
                avgLatency: 45.2,
                p99Latency: 120.5,
                errorRate: 0.001,
                requestCount: 1500000
            },
            {
                id: 2,
                modelId: 2,
                modelName: 'Risk Detector',
                status: 'warning',
                lastCheck: new Date().toISOString(),
                cpuUsage: 78.2,
                memoryUsage: 65.5,
                throughput: 890.0,
                avgLatency: 78.5,
                p99Latency: 250.3,
                errorRate: 0.008,
                requestCount: 890000
            }
        ];
    }

    renderMonitoring() {
        const container = document.getElementById('monitoringOverview');
        if (!container) return;

        container.innerHTML = this.monitors.map(monitor => `
            <div class="card mb-3 monitor-card" data-id="${monitor.id}">
                <div class="card-body">
                    <div class="d-flex justify-content-between align-items-start mb-3">
                        <div>
                            <h5 class="card-title mb-1">
                                <i class="fas fa-heartbeat me-2 ${monitor.status === 'healthy' ? 'text-success' : monitor.status === 'warning' ? 'text-warning' : 'text-danger'}"></i>
                                ${monitor.modelName}
                            </h5>
                        </div>
                        <div class="text-end">
                            <span class="badge ${this.getStatusBadgeClass(monitor.status)}">
                                ${this.getStatusText(monitor.status)}
                            </span>
                        </div>
                    </div>
                    <div class="row g-3">
                        <div class="col-md-2 col-4">
                            <div class="text-center">
                                <div class="fw-bold text-primary">${monitor.throughput.toFixed(0)}</div>
                                <div class="text-muted small">QPS</div>
                            </div>
                        </div>
                        <div class="col-md-2 col-4">
                            <div class="text-center">
                                <div class="fw-bold text-success">${monitor.avgLatency.toFixed(1)}ms</div>
                                <div class="text-muted small">延迟</div>
                            </div>
                        </div>
                        <div class="col-md-2 col-4">
                            <div class="text-center">
                                <div class="fw-bold text-info">${(monitor.errorRate * 100).toFixed(3)}%</div>
                                <div class="text-muted small">错误率</div>
                            </div>
                        </div>
                        <div class="col-md-2 col-4">
                            <div class="text-center">
                                <div class="fw-bold">${monitor.cpuUsage.toFixed(1)}%</div>
                                <div class="text-muted small">CPU</div>
                            </div>
                        </div>
                        <div class="col-md-2 col-4">
                            <div class="text-center">
                                <div class="fw-bold">${monitor.memoryUsage.toFixed(1)}%</div>
                                <div class="text-muted small">内存</div>
                            </div>
                        </div>
                        <div class="col-md-2 col-4">
                            <div class="text-center">
                                <div class="fw-bold">${(monitor.requestCount / 1000).toFixed(0)}K</div>
                                <div class="text-muted small">请求</div>
                            </div>
                        </div>
                    </div>
                    <div class="mt-3">
                        <button class="btn btn-sm btn-outline-primary" onclick="aiPlatform.viewMonitor(${monitor.modelId})">
                            <i class="fas fa-chart-line"></i> 详细监控
                        </button>
                    </div>
                </div>
            </div>
        `).join('');
    }

    initMonitoringCharts() {
        this.initLatencyChart();
        this.initThroughputChart();
        this.initErrorRateChart();
    }

    initLatencyChart() {
        const ctx = document.getElementById('latencyChart');
        if (!ctx || typeof Chart === 'undefined') return;

        const labels = [];
        const data = [];
        for (let i = 24; i >= 0; i--) {
            const d = new Date();
            d.setHours(d.getHours() - i);
            labels.push(d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }));
            data.push(40 + Math.random() * 20);
        }

        this.charts.latency = new Chart(ctx, {
            type: 'line',
            data: {
                labels: labels,
                datasets: [{
                    label: '平均延迟 (ms)',
                    data: data,
                    borderColor: 'rgb(75, 192, 192)',
                    backgroundColor: 'rgba(75, 192, 192, 0.1)',
                    tension: 0.4,
                    fill: true
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        display: false
                    }
                },
                scales: {
                    y: {
                        beginAtZero: true,
                        title: {
                            display: true,
                            text: '延迟 (ms)'
                        }
                    }
                }
            }
        });
    }

    initThroughputChart() {
        const ctx = document.getElementById('throughputChart');
        if (!ctx || typeof Chart === 'undefined') return;

        const labels = [];
        const data = [];
        for (let i = 24; i >= 0; i--) {
            const d = new Date();
            d.setHours(d.getHours() - i);
            labels.push(d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }));
            data.push(1000 + Math.random() * 500);
        }

        this.charts.throughput = new Chart(ctx, {
            type: 'line',
            data: {
                labels: labels,
                datasets: [{
                    label: 'QPS',
                    data: data,
                    borderColor: 'rgb(54, 162, 235)',
                    backgroundColor: 'rgba(54, 162, 235, 0.1)',
                    tension: 0.4,
                    fill: true
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        display: false
                    }
                },
                scales: {
                    y: {
                        beginAtZero: true,
                        title: {
                            display: true,
                            text: 'QPS'
                        }
                    }
                }
            }
        });
    }

    initErrorRateChart() {
        const ctx = document.getElementById('errorRateChart');
        if (!ctx || typeof Chart === 'undefined') return;

        const labels = [];
        const data = [];
        for (let i = 24; i >= 0; i--) {
            const d = new Date();
            d.setHours(d.getHours() - i);
            labels.push(d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }));
            data.push(0.05 + Math.random() * 0.1);
        }

        this.charts.errorRate = new Chart(ctx, {
            type: 'line',
            data: {
                labels: labels,
                datasets: [{
                    label: '错误率 (%)',
                    data: data,
                    borderColor: 'rgb(255, 99, 132)',
                    backgroundColor: 'rgba(255, 99, 132, 0.1)',
                    tension: 0.4,
                    fill: true
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        display: false
                    }
                },
                scales: {
                    y: {
                        beginAtZero: true,
                        title: {
                            display: true,
                            text: '错误率 (%)'
                        }
                    }
                }
            }
        });
    }

    getStatusBadgeClass(status) {
        const classes = {
            'deployed': 'bg-success',
            'draft': 'bg-secondary',
            'running': 'bg-primary',
            'completed': 'bg-info',
            'healthy': 'bg-success',
            'warning': 'bg-warning',
            'error': 'bg-danger'
        };
        return classes[status] || 'bg-secondary';
    }

    getStatusText(status) {
        const texts = {
            'deployed': '已部署',
            'draft': '草稿',
            'running': '运行中',
            'completed': '已完成',
            'healthy': '健康',
            'warning': '警告',
            'error': '错误'
        };
        return texts[status] || status;
    }

    showCreateModelModal() {
        const modal = new bootstrap.Modal(document.getElementById('createModelModal'));
        modal.show();
    }

    showCreateExperimentModal() {
        const modal = new bootstrap.Modal(document.getElementById('createExperimentModal'));
        modal.show();
    }

    showCreateABTestModal() {
        const modal = new bootstrap.Modal(document.getElementById('createABTestModal'));
        modal.show();
    }

    async refreshData() {
        await this.loadTabData(this.currentTab);
    }

    viewModel(id) {
        console.log('View model:', id);
    }

    async deployModel(id) {
        console.log('Deploy model:', id);
    }

    async deleteModel(id) {
        console.log('Delete model:', id);
    }

    viewExperiment(id) {
        console.log('View experiment:', id);
    }

    async startExperiment(id) {
        console.log('Start experiment:', id);
    }

    viewABTest(id) {
        console.log('View A/B test:', id);
    }

    async stopABTest(id) {
        console.log('Stop A/B test:', id);
    }

    viewMonitor(modelId) {
        console.log('View monitor:', modelId);
    }
}

let aiPlatform;
document.addEventListener('DOMContentLoaded', () => {
    aiPlatform = new AIManagementPlatform();
});
