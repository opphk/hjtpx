class AIManagementPlatformV2 {
    constructor() {
        this.models = [];
        this.trainingJobs = [];
        this.deployments = [];
        this.experiments = [];
        this.alerts = [];
        this.currentTab = 'models';
        this.charts = {};
        this.init();
    }

    init() {
        this.loadMockData();
        this.setupEventListeners();
        this.initCharts();
        this.renderModels();
        this.updateStatistics();
    }

    setupEventListeners() {
        document.querySelectorAll('.sidebar .nav-link').forEach(link => {
            link.addEventListener('click', (e) => {
                e.preventDefault();
                const tabId = link.getAttribute('data-tab');
                this.switchTab(tabId);
            });
        });

        document.querySelectorAll('[data-filter]').forEach(btn => {
            btn.addEventListener('click', (e) => {
                document.querySelectorAll('[data-filter]').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                this.renderModels(btn.getAttribute('data-filter'));
            });
        });
    }

    loadMockData() {
        this.models = [
            {
                id: 1,
                name: 'Captcha Classification v3',
                version: 'v3.2.1',
                type: 'classification',
                framework: 'pytorch',
                description: '验证码图像分类模型，支持多种验证码类型',
                status: 'deployed',
                metrics: {
                    accuracy: 0.98,
                    precision: 0.97,
                    recall: 0.98,
                    f1Score: 0.975
                },
                createdAt: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
                deployedAt: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString()
            },
            {
                id: 2,
                name: 'Risk Detection Model',
                version: 'v2.0.5',
                type: 'detection',
                framework: 'tensorflow',
                description: '用户行为风险检测模型',
                status: 'training',
                metrics: {
                    accuracy: 0.95,
                    precision: 0.94,
                    recall: 0.96
                },
                progress: 75,
                createdAt: new Date(Date.now() - 15 * 24 * 60 * 60 * 1000).toISOString()
            },
            {
                id: 3,
                name: 'Bot Behavior Analyzer',
                version: 'v1.5.0',
                type: 'classification',
                framework: 'xgboost',
                description: '机器人行为分析模型',
                status: 'draft',
                createdAt: new Date(Date.now() - 5 * 24 * 60 * 60 * 1000).toISOString()
            },
            {
                id: 4,
                name: 'Trajectory Prediction',
                version: 'v0.9.0',
                type: 'nlp',
                framework: 'pytorch',
                description: '用户轨迹预测模型（测试版）',
                status: 'deployed',
                metrics: {
                    accuracy: 0.92,
                    precision: 0.91,
                    recall: 0.93
                },
                createdAt: new Date(Date.now() - 45 * 24 * 60 * 60 * 1000).toISOString()
            }
        ];

        this.trainingJobs = [
            {
                id: 1,
                modelId: 2,
                modelName: 'Risk Detection Model',
                status: 'running',
                progress: 75,
                currentEpoch: 75,
                maxEpochs: 100,
                metrics: {
                    trainLoss: 0.15,
                    valLoss: 0.18,
                    trainAcc: 0.95,
                    valAcc: 0.94
                },
                startedAt: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString()
            }
        ];

        this.deployments = [
            {
                id: 1,
                modelId: 1,
                modelName: 'Captcha Classification v3',
                environment: 'production',
                status: 'running',
                replicas: 3,
                readyReplicas: 3,
                endpoints: {
                    publicURL: 'https://captcha-classifier.hjtpx.com',
                    apiVersion: 'v1'
                },
                metrics: {
                    requestsTotal: 1500000,
                    avgLatencyMs: 45.2,
                    errorRate: 0.002,
                    cpuUtilization: 35.5,
                    memoryUsageGB: 4.2
                }
            },
            {
                id: 2,
                modelId: 4,
                modelName: 'Trajectory Prediction',
                environment: 'production',
                status: 'running',
                replicas: 2,
                readyReplicas: 2,
                endpoints: {
                    publicURL: 'https://trajectory.hjtpx.com',
                    apiVersion: 'v1'
                },
                metrics: {
                    requestsTotal: 850000,
                    avgLatencyMs: 62.8,
                    errorRate: 0.005,
                    cpuUtilization: 28.3,
                    memoryUsageGB: 3.8
                }
            }
        ];

        this.experiments = [
            {
                id: 1,
                name: 'Optimize Classification Layers',
                type: 'hyperparameter',
                modelId: 1,
                status: 'completed',
                metrics: {
                    accuracy: 0.985,
                    improvement: 0.5
                },
                createdAt: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString()
            },
            {
                id: 2,
                name: 'Data Augmentation Test',
                type: 'data',
                modelId: 1,
                status: 'running',
                metrics: {
                    accuracy: 0.972,
                    improvement: -0.8
                },
                createdAt: new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString()
            }
        ];

        this.alerts = [
            {
                id: 1,
                modelId: 1,
                type: 'performance',
                severity: 'warning',
                title: '延迟升高',
                description: '模型平均延迟超过阈值',
                status: 'active',
                metrics: {
                    avgLatencyMs: 120,
                    threshold: 100
                },
                triggeredAt: new Date(Date.now() - 30 * 60 * 1000).toISOString()
            }
        ];
    }

    switchTab(tabId) {
        this.currentTab = tabId;
        
        document.querySelectorAll('.sidebar .nav-link').forEach(link => {
            link.classList.remove('active');
            if (link.getAttribute('data-tab') === tabId) {
                link.classList.add('active');
            }
        });

        this.loadTabData(tabId);
    }

    loadTabData(tabId) {
        switch (tabId) {
            case 'models':
                this.renderModels();
                break;
            case 'training':
                this.renderTrainingJobs();
                break;
            case 'deployment':
                this.renderDeployments();
                break;
            case 'experiments':
                this.renderExperiments();
                break;
            case 'monitoring':
                this.renderMonitoring();
                break;
            case 'ab-testing':
                this.renderABTesting();
                break;
        }
    }

    renderModels(filter = 'all') {
        const container = document.getElementById('modelsList');
        if (!container) return;

        let filteredModels = this.models;
        if (filter !== 'all') {
            filteredModels = this.models.filter(m => m.status === filter);
        }

        container.innerHTML = filteredModels.map(model => `
            <div class="col-md-6 mb-3">
                <div class="card model-card ${model.status}">
                    <div class="card-body">
                        <div class="d-flex justify-content-between align-items-start mb-3">
                            <div>
                                <h5 class="card-title mb-1">
                                    <i class="fas fa-brain me-2 text-primary"></i>
                                    ${model.name}
                                </h5>
                                <h6 class="text-muted mb-2">
                                    ${model.version} • ${model.type} • ${model.framework}
                                </h6>
                            </div>
                            <span class="badge badge-${model.status}">${this.getStatusText(model.status)}</span>
                        </div>
                        
                        <p class="text-muted small mb-3">${model.description}</p>
                        
                        ${model.metrics ? `
                        <div class="row mb-3">
                            <div class="col-4 text-center">
                                <div class="fw-bold text-primary">${(model.metrics.accuracy * 100).toFixed(1)}%</div>
                                <small class="text-muted">准确率</small>
                            </div>
                            <div class="col-4 text-center">
                                <div class="fw-bold text-success">${(model.metrics.precision * 100).toFixed(1)}%</div>
                                <small class="text-muted">精确率</small>
                            </div>
                            <div class="col-4 text-center">
                                <div class="fw-bold text-info">${(model.metrics.recall * 100).toFixed(1)}%</div>
                                <small class="text-muted">召回率</small>
                            </div>
                        </div>
                        ` : ''}
                        
                        ${model.status === 'training' && model.progress ? `
                        <div class="mb-3">
                            <div class="d-flex justify-content-between mb-1">
                                <small class="text-muted">训练进度</small>
                                <small class="text-muted">${model.progress}%</small>
                            </div>
                            <div class="progress">
                                <div class="progress-bar" style="width: ${model.progress}%"></div>
                            </div>
                        </div>
                        ` : ''}
                        
                        <div class="d-flex gap-2">
                            <button class="btn btn-sm btn-outline-primary" onclick="aiPlatform.viewModel(${model.id})">
                                <i class="fas fa-eye"></i> 详情
                            </button>
                            ${model.status === 'draft' ? `
                            <button class="btn btn-sm btn-outline-success" onclick="aiPlatform.trainModel(${model.id})">
                                <i class="fas fa-play"></i> 训练
                            </button>
                            ` : ''}
                            ${model.status === 'trained' ? `
                            <button class="btn btn-sm btn-outline-success" onclick="aiPlatform.deployModel(${model.id})">
                                <i class="fas fa-cloud-upload-alt"></i> 部署
                            </button>
                            ` : ''}
                            <button class="btn btn-sm btn-outline-danger" onclick="aiPlatform.deleteModel(${model.id})">
                                <i class="fas fa-trash"></i> 删除
                            </button>
                        </div>
                    </div>
                </div>
            </div>
        `).join('');
    }

    renderTrainingJobs() {
        console.log('Rendering training jobs:', this.trainingJobs);
    }

    renderDeployments() {
        console.log('Rendering deployments:', this.deployments);
    }

    renderExperiments() {
        console.log('Rendering experiments:', this.experiments);
    }

    renderMonitoring() {
        console.log('Rendering monitoring data');
    }

    renderABTesting() {
        console.log('Rendering A/B testing');
    }

    initCharts() {
        this.initPerformanceChart();
        this.initResourceChart();
    }

    initPerformanceChart() {
        const ctx = document.getElementById('performanceChart');
        if (!ctx || typeof Chart === 'undefined') return;

        const labels = [];
        const accuracyData = [];
        const latencyData = [];

        for (let i = 30; i >= 0; i--) {
            const d = new Date();
            d.setDate(d.getDate() - i);
            labels.push(d.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' }));
            accuracyData.push(0.95 + Math.random() * 0.03);
            latencyData.push(40 + Math.random() * 20);
        }

        this.charts.performance = new Chart(ctx, {
            type: 'line',
            data: {
                labels: labels,
                datasets: [
                    {
                        label: '准确率',
                        data: accuracyData,
                        borderColor: 'rgb(75, 192, 192)',
                        backgroundColor: 'rgba(75, 192, 192, 0.1)',
                        tension: 0.4,
                        yAxisID: 'y'
                    },
                    {
                        label: '延迟 (ms)',
                        data: latencyData,
                        borderColor: 'rgb(255, 99, 132)',
                        backgroundColor: 'rgba(255, 99, 132, 0.1)',
                        tension: 0.4,
                        yAxisID: 'y1'
                    }
                ]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                interaction: {
                    mode: 'index',
                    intersect: false,
                },
                plugins: {
                    legend: {
                        display: true,
                        position: 'top',
                    }
                },
                scales: {
                    y: {
                        type: 'linear',
                        display: true,
                        position: 'left',
                        title: {
                            display: true,
                            text: '准确率'
                        },
                        min: 0.9,
                        max: 1.0
                    },
                    y1: {
                        type: 'linear',
                        display: true,
                        position: 'right',
                        title: {
                            display: true,
                            text: '延迟 (ms)'
                        },
                        grid: {
                            drawOnChartArea: false,
                        }
                    }
                }
            }
        });
    }

    initResourceChart() {
        const ctx = document.getElementById('resourceChart');
        if (!ctx || typeof Chart === 'undefined') return;

        this.charts.resource = new Chart(ctx, {
            type: 'doughnut',
            data: {
                labels: ['CPU使用率', '内存使用率', 'GPU使用率', '剩余'],
                datasets: [{
                    data: [35.5, 42.0, 35.0, 47.5],
                    backgroundColor: [
                        'rgba(255, 99, 132, 0.8)',
                        'rgba(54, 162, 235, 0.8)',
                        'rgba(255, 206, 86, 0.8)',
                        'rgba(75, 192, 192, 0.2)'
                    ],
                    borderWidth: 0
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        position: 'right',
                    }
                },
                cutout: '60%'
            }
        });
    }

    updateStatistics() {
        document.getElementById('totalModels').textContent = this.models.length;
        document.getElementById('deployedModels').textContent = this.models.filter(m => m.status === 'deployed').length;
        document.getElementById('trainingJobs').textContent = this.trainingJobs.filter(j => j.status === 'running').length;
        document.getElementById('activeAlerts').textContent = this.alerts.filter(a => a.status === 'active').length;
    }

    getStatusText(status) {
        const texts = {
            'deployed': '已部署',
            'training': '训练中',
            'draft': '草稿',
            'trained': '已训练'
        };
        return texts[status] || status;
    }

    showCreateModelModal() {
        const modal = new bootstrap.Modal(document.getElementById('createModelModal'));
        modal.show();
    }

    createModel() {
        const name = document.getElementById('modelName').value;
        const type = document.getElementById('modelType').value;
        const framework = document.getElementById('modelFramework').value;
        const description = document.getElementById('modelDescription').value;

        if (!name) {
            alert('请输入模型名称');
            return;
        }

        const newModel = {
            id: this.models.length + 1,
            name: name,
            version: 'v0.1.0',
            type: type,
            framework: framework,
            description: description,
            status: 'draft',
            createdAt: new Date().toISOString()
        };

        this.models.push(newModel);
        this.renderModels();
        this.updateStatistics();

        bootstrap.Modal.getInstance(document.getElementById('createModelModal')).hide();
        
        document.getElementById('modelName').value = '';
        document.getElementById('modelDescription').value = '';
    }

    viewModel(id) {
        console.log('View model:', id);
    }

    trainModel(id) {
        console.log('Train model:', id);
    }

    deployModel(id) {
        console.log('Deploy model:', id);
    }

    deleteModel(id) {
        if (confirm('确定要删除这个模型吗？')) {
            this.models = this.models.filter(m => m.id !== id);
            this.renderModels();
            this.updateStatistics();
        }
    }
}

let aiPlatform;
document.addEventListener('DOMContentLoaded', () => {
    aiPlatform = new AIManagementPlatformV2();
});
