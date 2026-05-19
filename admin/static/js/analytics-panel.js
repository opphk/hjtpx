const ANALYTICS = (function() {
    let charts = {};
    let reportConfigs = [];
    let currentConfigId = null;
    let realtimeDataBuffer = [];
    let wsConnection = null;
    let isConnected = false;

    const init = function() {
        setupEventListeners();
        initRealtimeDashboard();
        initCustomReports();
        initDataExport();
        initPredictiveAnalytics();
        updateCurrentTime();
        setInterval(updateCurrentTime, 1000);
        initWebSocket();
        startRealtimeSimulation();
    };

    const setupEventListeners = function() {
        document.querySelectorAll('.nav-link[data-panel]').forEach(el => {
            el.addEventListener('click', function() {
                switchPanel(this.dataset.panel);
            });
        });

        document.getElementById('refreshBtn').addEventListener('click', refreshAllData);

        document.getElementById('configTimeRangeType').addEventListener('change', function() {
            const type = this.value;
            document.getElementById('configStartDate').disabled = type !== 'custom';
            document.getElementById('configEndDate').disabled = type !== 'custom';
        });

        document.getElementById('configScheduleEnabled').addEventListener('change', function() {
            const enabled = this.checked;
            document.getElementById('scheduleSettings').classList.toggle('d-none', !enabled);
        });

        document.getElementById('reportConfigForm').addEventListener('submit', handleSaveConfig);

        document.getElementById('deleteConfigBtn').addEventListener('click', handleDeleteConfig);

        document.getElementById('newReportBtn').addEventListener('click', createNewConfig);

        document.querySelectorAll('.export-format-btn').forEach(el => {
            el.addEventListener('click', function() {
                document.querySelectorAll('.export-format-btn').forEach(e => e.classList.remove('active'));
                this.classList.add('active');
            });
        });

        document.getElementById('exportBtn').addEventListener('click', handleExport);

        document.getElementById('modalConfirmBtn').addEventListener('click', handleModalConfirm);
    };

    const switchPanel = function(panelId) {
        document.querySelectorAll('.nav-link[data-panel]').forEach(el => el.classList.remove('active'));
        document.querySelector(`.nav-link[data-panel="${panelId}"]`).classList.add('active');

        document.querySelectorAll('[id$="-panel"]').forEach(el => el.classList.add('d-none'));
        document.getElementById(`${panelId}-panel`).classList.remove('d-none');

        if (panelId === 'realtime') {
            refreshRealtimeData();
        } else if (panelId === 'predictive') {
            refreshPredictiveData();
        }
    };

    const updateCurrentTime = function() {
        const now = new Date();
        document.getElementById('currentTime').textContent = now.toLocaleTimeString('zh-CN');
    };

    const initWebSocket = function() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/api/v1/admin/analytics/ws`;

        try {
            wsConnection = new WebSocket(wsUrl);

            wsConnection.onopen = function() {
                isConnected = true;
                console.log('WebSocket connected');
            };

            wsConnection.onmessage = function(event) {
                try {
                    const data = JSON.parse(event.data);
                    handleRealtimeData(data);
                } catch (e) {
                    console.error('Parse WebSocket data failed:', e);
                }
            };

            wsConnection.onerror = function(error) {
                console.error('WebSocket error:', error);
                isConnected = false;
            };

            wsConnection.onclose = function() {
                isConnected = false;
                setTimeout(initWebSocket, 3000);
            };
        } catch (e) {
            console.error('WebSocket init failed:', e);
        }
    };

    const handleRealtimeData = function(data) {
        if (data.type === 'metrics') {
            updateRealtimeMetrics(data.payload);
        } else if (data.type === 'chart') {
            updateRealtimeChart(data.payload);
        }
    };

    const startRealtimeSimulation = function() {
        setInterval(function() {
            if (!isConnected) {
                simulateRealtimeUpdate();
            }
        }, 3000);
    };

    const simulateRealtimeUpdate = function() {
        const payload = {
            totalRequests: Math.floor(854321 + Math.random() * 1000),
            successRate: (94.5 + Math.random() * 3).toFixed(1),
            avgResponseTime: Math.floor(50 + Math.random() * 30),
            blockedAttacks: Math.floor(12345 + Math.random() * 100),
            requestTrend: generateRequestTrendData()
        };
        updateRealtimeMetrics(payload);
        updateRealtimeChartData(payload.requestTrend);
    };

    const generateRequestTrendData = function() {
        const data = [];
        const now = new Date();
        for (let i = 19; i >= 0; i--) {
            const time = new Date(now.getTime() - i * 60000);
            data.push({
                time: time.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }),
                value: Math.floor(1000 + Math.random() * 2000)
            });
        }
        return data;
    };

    const initRealtimeDashboard = function() {
        initRealtimeRequestChart();
        initRealtimeTypeChart();
        initResponseTimeChart();
        initGeoDistributionChart();
        loadMockRealtimeData();
    };

    const initRealtimeRequestChart = function() {
        const ctx = document.getElementById('realtime-request-chart');
        if (!ctx) return;

        charts.realtimeRequest = new Chart(ctx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [{
                    label: '请求量',
                    data: [],
                    borderColor: '#3b82f6',
                    backgroundColor: 'rgba(59, 130, 246, 0.1)',
                    fill: true,
                    tension: 0.4,
                    pointRadius: 3,
                    pointHoverRadius: 6
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: { display: false },
                    tooltip: {
                        backgroundColor: 'rgba(0, 0, 0, 0.8)',
                        padding: 12,
                        cornerRadius: 8
                    }
                },
                scales: {
                    x: {
                        grid: { display: false },
                        ticks: { maxRotation: 45 }
                    },
                    y: {
                        beginAtZero: true,
                        grid: { color: 'rgba(0, 0, 0, 0.05)' }
                    }
                },
                animation: { duration: 500 }
            }
        });
    };

    const initRealtimeTypeChart = function() {
        const ctx = document.getElementById('realtime-type-chart');
        if (!ctx) return;

        charts.realtimeType = new Chart(ctx, {
            type: 'doughnut',
            data: {
                labels: ['滑块验证', '点选验证', '旋转验证', '拼图验证', '语音验证'],
                datasets: [{
                    data: [425632, 215678, 112345, 75234, 25432],
                    backgroundColor: ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6'],
                    borderWidth: 2,
                    borderColor: '#fff'
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        position: 'bottom',
                        labels: { padding: 15, usePointStyle: true }
                    }
                },
                cutout: '60%'
            }
        });
    };

    const initResponseTimeChart = function() {
        const ctx = document.getElementById('response-time-chart');
        if (!ctx) return;

        charts.responseTime = new Chart(ctx, {
            type: 'bar',
            data: {
                labels: ['<50ms', '50-100ms', '100-200ms', '200-500ms', '>500ms'],
                datasets: [{
                    label: '请求数',
                    data: [4523, 3214, 1876, 876, 234],
                    backgroundColor: [
                        'rgba(16, 185, 129, 0.8)',
                        'rgba(59, 130, 246, 0.8)',
                        'rgba(245, 158, 11, 0.8)',
                        'rgba(239, 68, 68, 0.8)',
                        'rgba(139, 92, 246, 0.8)'
                    ],
                    borderRadius: 6
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: { legend: { display: false } },
                scales: {
                    x: { grid: { display: false } },
                    y: { beginAtZero: true }
                }
            }
        });
    };

    const initGeoDistributionChart = function() {
        const ctx = document.getElementById('geo-distribution-chart');
        if (!ctx) return;

        charts.geoDistribution = new Chart(ctx, {
            type: 'bar',
            data: {
                labels: ['北京', '上海', '广州', '深圳', '杭州', '成都', '武汉', '西安', '南京', '重庆'],
                datasets: [{
                    label: '请求数',
                    data: [3500, 3200, 2800, 2600, 2100, 1800, 1600, 1400, 1300, 1200],
                    backgroundColor: 'rgba(59, 130, 246, 0.8)',
                    borderRadius: 6
                }]
            },
            options: {
                indexAxis: 'y',
                responsive: true,
                maintainAspectRatio: false,
                plugins: { legend: { display: false } },
                scales: {
                    x: { beginAtZero: true },
                    y: { reverse: true }
                }
            }
        });
    };

    const loadMockRealtimeData = function() {
        updateRealtimeMetrics({
            totalRequests: 854321,
            successRate: '94.7',
            avgResponseTime: 58,
            blockedAttacks: 12345
        });
        updateRealtimeChartData(generateRequestTrendData());
    };

    const updateRealtimeMetrics = function(data) {
        document.getElementById('rt-total-requests').textContent = formatNumber(data.totalRequests);
        document.getElementById('rt-success-rate').textContent = data.successRate + '%';
        document.getElementById('rt-response-time').textContent = data.avgResponseTime + 'ms';
        document.getElementById('rt-blocked-attacks').textContent = formatNumber(data.blockedAttacks);
    };

    const updateRealtimeChartData = function(data) {
        if (!charts.realtimeRequest) return;
        charts.realtimeRequest.data.labels = data.map(d => d.time);
        charts.realtimeRequest.data.datasets[0].data = data.map(d => d.value);
        charts.realtimeRequest.update('none');
    };

    const refreshRealtimeData = function() {
        loadMockRealtimeData();
        showToast('数据已刷新', 'success');
    };

    const refreshAllData = function() {
        refreshRealtimeData();
        refreshCustomReports();
        refreshExportHistory();
        refreshPredictiveData();
    };

    const initCustomReports = function() {
        loadReportConfigs();
    };

    const loadReportConfigs = function() {
        reportConfigs = getMockReportConfigs();
        renderReportConfigsList();
    };

    const getMockReportConfigs = function() {
        return [
            {
                id: 'config-1',
                name: '日常监控报表',
                description: '每日系统运行状态监控',
                metrics: ['totalRequests', 'successRate', 'responseTime', 'attackCount'],
                timeRange: { type: 'daily' },
                visualization: 'dashboard',
                refreshRate: '5m',
                schedule: { enabled: true, frequency: 'daily', time: '09:00', email: 'admin@example.com' },
                createdAt: '2024-01-10',
                updatedAt: '2024-01-15'
            },
            {
                id: 'config-2',
                name: '安全分析报表',
                description: '安全攻击趋势分析',
                metrics: ['attackCount', 'detectionRate', 'riskScore'],
                timeRange: { type: 'weekly' },
                visualization: 'charts',
                refreshRate: '15m',
                schedule: { enabled: false },
                createdAt: '2024-01-08',
                updatedAt: '2024-01-14'
            },
            {
                id: 'config-3',
                name: '性能监控报表',
                description: '系统性能指标监控',
                metrics: ['responseTime', 'totalRequests'],
                timeRange: { type: 'monthly' },
                visualization: 'table',
                refreshRate: '1h',
                schedule: { enabled: true, frequency: 'weekly', time: '08:00', email: 'dev@example.com' },
                createdAt: '2024-01-05',
                updatedAt: '2024-01-12'
            }
        ];
    };

    const renderReportConfigsList = function() {
        const container = document.getElementById('reportConfigsList');
        if (!container) return;

        if (reportConfigs.length === 0) {
            container.innerHTML = '<div class="text-center text-muted py-4">暂无报表配置</div>';
            return;
        }

        container.innerHTML = reportConfigs.map(config => `
            <div class="config-item ${currentConfigId === config.id ? 'active' : ''}" onclick="ANALYTICS.selectConfig('${config.id}')">
                <div class="d-flex justify-content-between align-items-start">
                    <div>
                        <div class="config-item-title">${escapeHtml(config.name)}</div>
                        <div class="config-item-desc">${escapeHtml(config.description || '')}</div>
                    </div>
                    ${config.schedule?.enabled ? '<span class="schedule-badge">定时</span>' : ''}
                </div>
            </div>
        `).join('');
    };

    const selectConfig = function(id) {
        currentConfigId = id;
        const config = reportConfigs.find(c => c.id === id);
        if (!config) return;

        document.getElementById('configEditorTitle').textContent = '编辑报表配置';
        document.getElementById('deleteConfigBtn').style.display = 'inline-block';

        document.getElementById('configName').value = config.name;
        document.getElementById('configDescription').value = config.description || '';

        document.querySelectorAll('#metricsSelector input').forEach(cb => {
            cb.checked = config.metrics.includes(cb.value);
        });

        document.getElementById('configTimeRangeType').value = config.timeRange?.type || 'weekly';
        document.getElementById('configVisualization').value = config.visualization || 'dashboard';
        document.getElementById('configRefreshRate').value = config.refreshRate || '5m';

        document.getElementById('configScheduleEnabled').checked = config.schedule?.enabled || false;
        document.getElementById('scheduleSettings').classList.toggle('d-none', !config.schedule?.enabled);

        document.getElementById('configScheduleFrequency').value = config.schedule?.frequency || 'daily';
        document.getElementById('configScheduleTime').value = config.schedule?.time || '09:00';
        document.getElementById('configScheduleEmail').value = config.schedule?.email || '';

        renderReportConfigsList();
    };

    const createNewConfig = function() {
        currentConfigId = null;
        document.getElementById('configEditorTitle').textContent = '新建报表配置';
        document.getElementById('deleteConfigBtn').style.display = 'none';
        document.getElementById('reportConfigForm').reset();
        document.getElementById('scheduleSettings').classList.add('d-none');
        renderReportConfigsList();
    };

    const handleSaveConfig = function(e) {
        e.preventDefault();

        const config = {
            id: currentConfigId || 'config-' + Date.now(),
            name: document.getElementById('configName').value,
            description: document.getElementById('configDescription').value,
            metrics: Array.from(document.querySelectorAll('#metricsSelector input:checked')).map(cb => cb.value),
            timeRange: {
                type: document.getElementById('configTimeRangeType').value,
                start: document.getElementById('configStartDate').value,
                end: document.getElementById('configEndDate').value
            },
            visualization: document.getElementById('configVisualization').value,
            refreshRate: document.getElementById('configRefreshRate').value,
            schedule: {
                enabled: document.getElementById('configScheduleEnabled').checked,
                frequency: document.getElementById('configScheduleFrequency').value,
                time: document.getElementById('configScheduleTime').value,
                email: document.getElementById('configScheduleEmail').value
            },
            createdAt: currentConfigId ? reportConfigs.find(c => c.id === currentConfigId)?.createdAt || new Date().toISOString().split('T')[0] : new Date().toISOString().split('T')[0],
            updatedAt: new Date().toISOString().split('T')[0]
        };

        if (!config.name) {
            showToast('请输入报表名称', 'warning');
            return;
        }

        if (currentConfigId) {
            const index = reportConfigs.findIndex(c => c.id === currentConfigId);
            if (index >= 0) {
                reportConfigs[index] = config;
            }
        } else {
            reportConfigs.push(config);
        }

        renderReportConfigsList();
        showToast('保存成功', 'success');
    };

    const handleDeleteConfig = function() {
        if (!currentConfigId) return;

        showModal('确认删除', '确定要删除这个报表配置吗？此操作不可撤销。', function() {
            reportConfigs = reportConfigs.filter(c => c.id !== currentConfigId);
            createNewConfig();
            showToast('删除成功', 'success');
        });
    };

    const refreshCustomReports = function() {
        loadReportConfigs();
    };

    const initDataExport = function() {
        const today = new Date();
        const lastWeek = new Date(today);
        lastWeek.setDate(lastWeek.getDate() - 7);

        document.getElementById('exportStartDate').value = lastWeek.toISOString().split('T')[0];
        document.getElementById('exportEndDate').value = today.toISOString().split('T')[0];

        loadExportHistory();
    };

    const loadExportHistory = function() {
        const history = getMockExportHistory();
        const tbody = document.getElementById('exportHistoryTable');
        if (!tbody) return;

        tbody.innerHTML = history.map(item => `
            <tr>
                <td><i class="fas fa-file-${getFormatIcon(item.format)} me-2"></i>${escapeHtml(item.filename)}</td>
                <td><span class="badge bg-secondary">${getFormatLabel(item.format)}</span></td>
                <td>${escapeHtml(item.size)}</td>
                <td>${escapeHtml(item.time)}</td>
                <td>
                    <button class="btn btn-sm btn-outline-primary" onclick="ANALYTICS.downloadExport('${item.filename}')">
                        <i class="fas fa-download"></i>
                    </button>
                </td>
            </tr>
        `).join('');
    };

    const getMockExportHistory = function() {
        return [
            { filename: 'report_summary_20240115.xlsx', format: 'excel', size: '256 KB', time: '2024-01-15 09:00' },
            { filename: 'report_attack_20240114.csv', format: 'csv', size: '128 KB', time: '2024-01-14 18:30' },
            { filename: 'report_performance_20240113.pdf', format: 'pdf', size: '512 KB', time: '2024-01-13 10:15' },
            { filename: 'report_user_20240112.json', format: 'json', size: '340 KB', time: '2024-01-12 14:45' },
            { filename: 'report_summary_20240111.xlsx', format: 'excel', size: '280 KB', time: '2024-01-11 09:00' }
        ];
    };

    const getFormatIcon = function(format) {
        const icons = { excel: 'excel', csv: 'csv', pdf: 'pdf', json: 'code' };
        return icons[format] || 'file';
    };

    const getFormatLabel = function(format) {
        const labels = { excel: 'Excel', csv: 'CSV', pdf: 'PDF', json: 'JSON' };
        return labels[format] || format;
    };

    const handleExport = function() {
        const format = document.querySelector('.export-format-btn.active').dataset.format;
        const reportType = document.getElementById('exportReportType').value;
        const startDate = document.getElementById('exportStartDate').value;
        const endDate = document.getElementById('exportEndDate').value;
        const fields = Array.from(document.querySelectorAll('#data-export-panel input[type="checkbox"]:checked')).map(cb => cb.value);

        if (!startDate || !endDate) {
            showToast('请选择日期范围', 'warning');
            return;
        }

        const progressBar = document.getElementById('exportProgress');
        const progressText = document.getElementById('exportProgressText');
        const exportBtn = document.getElementById('exportBtn');

        progressBar.classList.remove('d-none');
        exportBtn.disabled = true;

        let progress = 0;
        const interval = setInterval(() => {
            progress += Math.random() * 15;
            if (progress >= 100) {
                clearInterval(interval);
                progress = 100;
                progressText.textContent = '生成完成，正在下载...';

                setTimeout(() => {
                    generateExportFile(format, reportType, startDate, endDate, fields);
                    progressBar.classList.add('d-none');
                    exportBtn.disabled = false;
                }, 500);
            } else {
                progressText.textContent = `正在处理数据... ${Math.floor(progress)}%`;
            }
            document.querySelector('.progress-bar').style.width = progress + '%';
        }, 200);
    };

    const generateExportFile = function(format, reportType, startDate, endDate, fields) {
        const mockData = generateMockExportData(fields, 100);

        switch (format) {
            case 'excel':
                exportToExcel(mockData, reportType);
                break;
            case 'csv':
                exportToCSV(mockData, reportType);
                break;
            case 'json':
                exportToJSON(mockData, reportType);
                break;
            case 'pdf':
                exportToPDF(mockData, reportType);
                break;
        }
    };

    const generateMockExportData = function(fields, count) {
        const data = [];
        const types = ['滑块验证', '点选验证', '旋转验证', '拼图验证'];
        const results = ['success', 'failed'];

        for (let i = 0; i < count; i++) {
            const row = {};
            if (fields.includes('timestamp')) row.timestamp = new Date(Date.now() - i * 300000).toISOString();
            if (fields.includes('requestId')) row.requestId = 'REQ-' + String(i).padStart(8, '0');
            if (fields.includes('userId')) row.userId = 'USER-' + String(Math.floor(Math.random() * 10000)).padStart(5, '0');
            if (fields.includes('captchaType')) row.captchaType = types[Math.floor(Math.random() * types.length)];
            if (fields.includes('result')) row.result = results[Math.floor(Math.random() * results.length)];
            if (fields.includes('responseTime')) row.responseTime = Math.floor(30 + Math.random() * 100);
            if (fields.includes('ipAddress')) row.ipAddress = `192.168.${Math.floor(Math.random() * 256)}.${Math.floor(Math.random() * 256)}`;
            if (fields.includes('riskScore')) row.riskScore = Math.floor(Math.random() * 100);
            data.push(row);
        }

        return data;
    };

    const exportToExcel = function(data, reportType) {
        const worksheet = XLSX.utils.json_to_sheet(data);
        const workbook = XLSX.utils.book_new();
        XLSX.utils.book_append_sheet(workbook, worksheet, 'Data');
        XLSX.writeFile(workbook, `report_${reportType}_${Date.now()}.xlsx`);
        showToast('Excel文件已下载', 'success');
    };

    const exportToCSV = function(data, reportType) {
        const csv = convertToCSV(data);
        downloadFile(csv, `report_${reportType}_${Date.now()}.csv`, 'text/csv');
        showToast('CSV文件已下载', 'success');
    };

    const exportToJSON = function(data, reportType) {
        const json = JSON.stringify(data, null, 2);
        downloadFile(json, `report_${reportType}_${Date.now()}.json`, 'application/json');
        showToast('JSON文件已下载', 'success');
    };

    const exportToPDF = function(data, reportType) {
        const content = generatePDFContent(data, reportType);
        downloadFile(content, `report_${reportType}_${Date.now()}.txt`, 'text/plain');
        showToast('PDF文件已生成（演示模式）', 'success');
    };

    const convertToCSV = function(data) {
        if (!data.length) return '';
        const headers = Object.keys(data[0]);
        return [headers.join(','), ...data.map(row => headers.map(h => `"${row[h] || ''}"`).join(','))].join('\n');
    };

    const downloadFile = function(content, filename, type) {
        const blob = new Blob([content], { type: type });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
    };

    const generatePDFContent = function(data, reportType) {
        return `
========================================
            数据分析报告
========================================
报表类型: ${reportType}
生成时间: ${new Date().toLocaleString('zh-CN')}
数据行数: ${data.length}

========================================
数据摘要:
- 成功验证: ${data.filter(d => d.result === 'success').length}
- 失败验证: ${data.filter(d => d.result === 'failed').length}
- 平均响应时间: ${data.reduce((sum, d) => sum + (d.responseTime || 0), 0) / data.length}ms

========================================
数据详情:
${data.slice(0, 10).map((row, i) => `
${i + 1}. ${row.requestId} - ${row.captchaType} - ${row.result} - ${row.responseTime}ms
`).join('')}

========================================
报告结束
        `.trim();
    };

    const downloadExport = function(filename) {
        showToast(`正在下载 ${filename}`, 'info');
    };

    const refreshExportHistory = function() {
        loadExportHistory();
    };

    const initPredictiveAnalytics = function() {
        initForecastRequestChart();
        initForecastSuccessChart();
        loadMockPredictiveData();
    };

    const initForecastRequestChart = function() {
        const ctx = document.getElementById('forecast-request-chart');
        if (!ctx) return;

        const days = generateDayLabels();
        const historicalData = [820, 932, 901, 934, 1290, 1330, 1320, 1250];
        const forecastData = [null, null, null, null, null, null, null, null, 1280, 1350, 1420];

        charts.forecastRequest = new Chart(ctx, {
            type: 'line',
            data: {
                labels: days,
                datasets: [
                    {
                        label: '历史数据',
                        data: historicalData,
                        borderColor: '#3b82f6',
                        backgroundColor: 'rgba(59, 130, 246, 0.1)',
                        fill: true,
                        tension: 0.4,
                        pointRadius: 4
                    },
                    {
                        label: '预测数据',
                        data: forecastData,
                        borderColor: '#f59e0b',
                        backgroundColor: 'rgba(245, 158, 11, 0.1)',
                        fill: true,
                        tension: 0.4,
                        borderDash: [5, 5],
                        pointRadius: 4
                    }
                ]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: { position: 'bottom' },
                    tooltip: { backgroundColor: 'rgba(0, 0, 0, 0.8)' }
                },
                scales: {
                    x: { grid: { display: false } },
                    y: { beginAtZero: true }
                }
            }
        });
    };

    const initForecastSuccessChart = function() {
        const ctx = document.getElementById('forecast-success-chart');
        if (!ctx) return;

        const days = ['周一', '周二', '周三', '周四', '周五', '周六', '周日'];
        const historical = [94.2, 94.8, 95.1, 94.5, 95.3, 95.8, 95.5];
        const forecast = [95.2, 95.5, 95.8, 96.0, 96.2, 96.5, 96.8];

        charts.forecastSuccess = new Chart(ctx, {
            type: 'bar',
            data: {
                labels: days,
                datasets: [
                    {
                        label: '历史成功率',
                        data: historical,
                        backgroundColor: 'rgba(59, 130, 246, 0.7)',
                        borderRadius: 6
                    },
                    {
                        label: '预测成功率',
                        data: forecast,
                        backgroundColor: 'rgba(245, 158, 11, 0.7)',
                        borderRadius: 6
                    }
                ]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: { legend: { position: 'bottom' } },
                scales: {
                    x: { grid: { display: false } },
                    y: { min: 90, max: 100 }
                }
            }
        });
    };

    const generateDayLabels = function() {
        const labels = [];
        for (let i = 7; i >= 0; i--) {
            const date = new Date();
            date.setDate(date.getDate() - i);
            labels.push(`${date.getMonth() + 1}/${date.getDate()}`);
        }
        for (let i = 1; i <= 3; i++) {
            const date = new Date();
            date.setDate(date.getDate() + i);
            labels.push(`${date.getMonth() + 1}/${date.getDate()}`);
        }
        return labels;
    };

    const loadMockPredictiveData = function() {
        document.getElementById('pred-requests').textContent = '1.2M';
        document.getElementById('pred-success-rate').textContent = '96.5%';
        document.getElementById('pred-attacks').textContent = '8,520';
        document.getElementById('pred-risk-level').textContent = '中等';

        renderAnomalyPredictions();
        renderSmartRecommendations();
    };

    const renderAnomalyPredictions = function() {
        const container = document.getElementById('anomalyPredictions');
        if (!container) return;

        const anomalies = [
            { time: '明天 02:00-04:00', type: 'spike', description: '预测请求量异常峰值', severity: 'high' },
            { time: '后天 10:00-12:00', type: 'pattern', description: '检测到潜在攻击模式', severity: 'medium' },
            { time: '本周六 14:00-16:00', type: 'trend', description: '成功率可能下降', severity: 'low' }
        ];

        container.innerHTML = anomalies.map(a => `
            <div class="alert-badge alert-${a.severity}">
                <i class="fas fa-${getSeverityIcon(a.severity)}"></i>
                <div>
                    <div class="font-medium">${escapeHtml(a.description)}</div>
                    <div class="text-xs">${escapeHtml(a.time)}</div>
                </div>
            </div>
        `).join('');
    };

    const getSeverityIcon = function(severity) {
        const icons = { high: 'alert-triangle', medium: 'alert-circle', low: 'info-circle' };
        return icons[severity] || 'info-circle';
    };

    const renderSmartRecommendations = function() {
        const container = document.getElementById('smartRecommendations');
        if (!container) return;

        const recommendations = [
            { priority: 'high', text: '建议在凌晨时段增加风控规则敏感度，预计可降低30%的攻击成功率' },
            { priority: 'medium', text: '考虑对高频IP段实施更严格的验证策略' },
            { priority: 'medium', text: '建议更新设备指纹识别库以提升检测准确率' },
            { priority: 'low', text: '优化滑块验证码难度参数以平衡安全性与用户体验' }
        ];

        container.innerHTML = recommendations.map(r => `
            <div class="card mb-2 ${r.priority === 'high' ? 'border-danger' : r.priority === 'medium' ? 'border-warning' : 'border-info'}">
                <div class="card-body py-3">
                    <div class="d-flex align-items-start gap-3">
                        <i class="fas fa-lightbulb ${r.priority === 'high' ? 'text-danger' : r.priority === 'medium' ? 'text-warning' : 'text-info'} mt-1"></i>
                        <div>
                            <div class="text-sm font-medium">${escapeHtml(r.text)}</div>
                            <div class="text-xs text-muted mt-1">优先级: ${getPriorityLabel(r.priority)}</div>
                        </div>
                    </div>
                </div>
            </div>
        `).join('');
    };

    const getPriorityLabel = function(priority) {
        const labels = { high: '高', medium: '中', low: '低' };
        return labels[priority] || priority;
    };

    const refreshPredictiveData = function() {
        loadMockPredictiveData();
        showToast('预测数据已刷新', 'success');
    };

    const showToast = function(message, type = 'info') {
        const toast = document.createElement('div');
        toast.className = `alert alert-${type} alert-dismissible fade show position-fixed`;
        toast.style.cssText = 'top: 80px; right: 20px; z-index: 9999; min-width: 250px;';
        toast.innerHTML = `
            ${message}
            <button type="button" class="btn-close" data-bs-dismiss="alert"></button>
        `;
        document.body.appendChild(toast);

        setTimeout(() => toast.remove(), 3000);
    };

    const showModal = function(title, body, onConfirm) {
        document.getElementById('modalTitle').textContent = title;
        document.getElementById('modalBody').textContent = body;
        document.getElementById('modal').classList.add('show');
        document.getElementById('modalConfirmBtn').onclick = function() {
            closeModal();
            onConfirm();
        };
    };

    const closeModal = function() {
        document.getElementById('modal').classList.remove('show');
    };

    const handleModalConfirm = function() {
        closeModal();
    };

    const logout = function() {
        showModal('确认退出', '确定要退出登录吗？', function() {
            window.location.href = '/admin/login';
        });
    };

    const formatNumber = function(num) {
        if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
        if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
        return num.toString();
    };

    const escapeHtml = function(text) {
        if (!text) return '';
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    };

    return {
        init,
        selectConfig
    };
})();

document.addEventListener('DOMContentLoaded', ANALYTICS.init);