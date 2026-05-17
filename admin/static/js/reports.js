let previewChart = null;
let currentChartType = 'table';
let selectedExportFormat = 'csv';
let selectedTemplate = null;

const mockReportData = {
    summary: { customReports: 12, exports: 156, schedules: 5, history: 89 },
    preview: [
        { date: '2024-01-15', app: '用户中心', requests: 15678, success: 15234, fail: 444, rate: '97.2%', avgResponse: '125ms' },
        { date: '2024-01-14', app: '支付系统', requests: 23456, success: 22890, fail: 566, rate: '97.6%', avgResponse: '118ms' },
        { date: '2024-01-13', app: '消息推送', requests: 8765, success: 8654, fail: 111, rate: '98.7%', avgResponse: '98ms' },
        { date: '2024-01-12', app: '数据分析', requests: 12345, success: 12089, fail: 256, rate: '97.9%', avgResponse: '132ms' },
        { date: '2024-01-11', app: '社交平台', requests: 45678, success: 44890, fail: 788, rate: '98.3%', avgResponse: '145ms' },
        { date: '2024-01-10', app: '电商后台', requests: 34567, success: 33901, fail: 666, rate: '98.1%', avgResponse: '108ms' },
        { date: '2024-01-09', app: '游戏中心', requests: 28901, success: 28345, fail: 556, rate: '98.1%', avgResponse: '115ms' }
    ],
    templates: {
        daily: { name: '日报表', description: '每日运营数据汇总', dimensions: ['时间', '应用'], metrics: ['请求量', '成功数', '失败数', '成功率'], chart: 'bar' },
        weekly: { name: '周报表', description: '每周数据趋势分析', dimensions: ['时间', '应用'], metrics: ['请求量', '成功数', '成功率'], chart: 'line' },
        monthly: { name: '月报表', description: '月度综合运营报告', dimensions: ['时间', '应用', '地区'], metrics: ['请求量', '成功数', '失败数', '成功率', '平均响应'], chart: 'mixed' },
        security: { name: '安全报告', description: '安全事件与风险分析', dimensions: ['时间', '风险等级'], metrics: ['攻击次数', '拦截次数', '风险评分'], chart: 'bar' },
        performance: { name: '性能报告', description: '系统性能与响应分析', dimensions: ['时间', '应用'], metrics: ['平均响应', 'P95响应', 'P99响应'], chart: 'line' },
        user: { name: '用户报告', description: '用户行为与增长分析', dimensions: ['时间'], metrics: ['新增用户', '活跃用户', '留存率'], chart: 'line' },
        application: { name: '应用报告', description: '各应用使用情况对比', dimensions: ['应用'], metrics: ['请求量', '成功率', '平均响应'], chart: 'pie' }
    }
};

document.addEventListener('DOMContentLoaded', () => {
    initStats();
    initPreviewTable();
    setupEventListeners();
});

function setupEventListeners() {
    document.getElementById('refreshBtn')?.addEventListener('click', () => {
        initStats();
        initPreviewTable();
    });

    document.querySelectorAll('[data-chart]').forEach(btn => {
        btn.addEventListener('click', (e) => {
            document.querySelectorAll('[data-chart]').forEach(b => b.classList.remove('active'));
            e.target.classList.add('active');
            switchPreviewChart(e.target.dataset.chart);
        });
    });

    document.querySelectorAll('.export-format-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('.export-format-btn').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            selectedExportFormat = btn.dataset.format;
        });
    });

    document.querySelectorAll('.template-card[data-template]').forEach(card => {
        card.addEventListener('click', () => {
            document.querySelectorAll('.template-card').forEach(c => c.classList.remove('active'));
            card.classList.add('active');
            showTemplateDetail(card.dataset.template);
        });
    });

    document.getElementById('exportTimeRange')?.addEventListener('change', (e) => {
        const customRange = document.getElementById('customDateRange');
        if (e.target.value === 'custom') {
            customRange.classList.remove('d-none');
        } else {
            customRange.classList.add('d-none');
        }
        estimateExportSize();
    });

    document.getElementById('exportDataSource')?.addEventListener('change', estimateExportSize);
    document.getElementById('exportStartDate')?.addEventListener('change', estimateExportSize);
    document.getElementById('exportEndDate')?.addEventListener('change', estimateExportSize);

    document.getElementById('startExportBtn')?.addEventListener('click', startExport);
    document.getElementById('exportReportBtn')?.addEventListener('click', exportReport);
    document.getElementById('saveReportBtn')?.addEventListener('click', () => {
        const modal = new bootstrap.Modal(document.getElementById('saveReportModal'));
        modal.show();
    });
    document.getElementById('confirmSaveReport')?.addEventListener('click', saveReport);
    document.getElementById('printPreviewBtn')?.addEventListener('click', printPreview);
    document.getElementById('createScheduleBtn')?.addEventListener('click', createSchedule);
}

function initStats() {
    document.getElementById('customReportsCount').textContent = mockReportData.summary.customReports;
    document.getElementById('exportCount').textContent = mockReportData.summary.exports;
    document.getElementById('scheduleCount').textContent = mockReportData.summary.schedules;
    document.getElementById('historyCount').textContent = mockReportData.summary.history;
}

function initPreviewTable() {
    const tbody = document.getElementById('previewTableBody');
    if (!tbody) return;

    tbody.innerHTML = mockReportData.preview.map(row => `
        <tr>
            <td>${escapeHtml(row.date)}</td>
            <td>${escapeHtml(row.app)}</td>
            <td class="text-primary fw-bold">${formatNumber(row.requests)}</td>
            <td class="text-success">${formatNumber(row.success)}</td>
            <td class="text-danger">${formatNumber(row.fail)}</td>
            <td><span class="badge ${parseFloat(row.rate) >= 97 ? 'bg-success' : 'bg-warning'}">${row.rate}</span></td>
            <td>${row.avgResponse}</td>
        </tr>
    `).join('');
}

function switchPreviewChart(type) {
    currentChartType = type;
    const preview = document.getElementById('reportPreview');
    const chart = document.getElementById('previewChart');

    if (type === 'table') {
        preview.style.display = 'block';
        chart.style.display = 'none';
        return;
    }

    preview.style.display = 'none';
    chart.style.display = 'block';

    if (previewChart) {
        previewChart.destroy();
    }

    const labels = mockReportData.preview.map(r => r.date);
    const data = mockReportData.preview.map(r => r.requests);

    const chartType = type === 'pie' ? 'pie' : type === 'line' ? 'line' : 'bar';
    const backgroundColor = type === 'pie' ? ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899', '#06b6d4'] : 'rgba(59, 130, 246, 0.8)';

    previewChart = new Chart(chart, {
        type: chartType,
        data: {
            labels: labels,
            datasets: [{
                label: '请求量',
                data: data,
                backgroundColor: backgroundColor,
                borderColor: '#3b82f6',
                borderWidth: type === 'line' ? 2 : 0,
                fill: type === 'line',
                tension: 0.4
            }]
        },
        options: getChartOptions(chartType)
    });
}

function getChartOptions(type) {
    const base = {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
            legend: { display: type === 'pie', position: 'bottom' },
            tooltip: {
                backgroundColor: 'rgba(0, 0, 0, 0.8)',
                padding: 12,
                cornerRadius: 8
            }
        }
    };

    if (type === 'line' || type === 'bar') {
        base.scales = {
            x: { grid: { display: false } },
            y: { beginAtZero: true, grid: { color: 'rgba(0, 0, 0, 0.05)' } }
        };
    }

    return base;
}

function showTemplateDetail(templateKey) {
    const template = mockReportData.templates[templateKey];
    if (!template) return;

    const detail = document.getElementById('templateDetail');
    detail.innerHTML = `
        <h5 class="mb-4"><i class="fas fa-info-circle me-2" style="color: var(--gold);"></i>${escapeHtml(template.name)} 详情</h5>
        <div class="row">
            <div class="col-md-6">
                <div class="mb-3">
                    <label class="form-label fw-bold">模板描述</label>
                    <p>${escapeHtml(template.description)}</p>
                </div>
                <div class="mb-3">
                    <label class="form-label fw-bold">包含维度</label>
                    <div>${template.dimensions.map(d => `<span class="dimension-tag">${escapeHtml(d)}</span>`).join('')}</div>
                </div>
                <div class="mb-3">
                    <label class="form-label fw-bold">包含指标</label>
                    <div>${template.metrics.map(m => `<span class="metric-tag">${escapeHtml(m)}</span>`).join('')}</div>
                </div>
            </div>
            <div class="col-md-6">
                <label class="form-label fw-bold">图表类型</label>
                <div class="d-flex gap-2 mb-3">
                    <span class="badge bg-primary">${getChartTypeName(template.chart)}</span>
                </div>
                <canvas id="templatePreviewChart" height="200"></canvas>
            </div>
        </div>
        <div class="mt-4">
            <button class="btn-gold" onclick="applyTemplate('${templateKey}')"><i class="fas fa-check me-2"></i>应用此模板</button>
            <button class="btn btn-outline-secondary" onclick="previewTemplate('${templateKey}')"><i class="fas fa-eye me-2"></i>预览</button>
        </div>
    `;

    setTimeout(() => renderTemplateChart(template), 100);
}

function renderTemplateChart(template) {
    const ctx = document.getElementById('templatePreviewChart');
    if (!ctx) return;

    new Chart(ctx, {
        type: template.chart === 'mixed' ? 'bar' : template.chart,
        data: {
            labels: ['周一', '周二', '周三', '周四', '周五', '周六', '周日'],
            datasets: [{
                label: template.metrics[0],
                data: generateRandomData(7, 5000, 20000),
                backgroundColor: 'rgba(59, 130, 246, 0.8)',
                borderColor: '#3b82f6',
                borderWidth: 1
            }]
        },
        options: getChartOptions(template.chart === 'mixed' ? 'bar' : template.chart)
    });
}

function applyTemplate(templateKey) {
    showToast(`已应用 "${mockReportData.templates[templateKey]?.name}" 模板`, 'success');
    document.querySelector('[data-bs-target="#custom-report"]')?.click();
}

function previewTemplate(templateKey) {
    const template = mockReportData.templates[templateKey];
    if (template) {
        showToast(`预览 "${template.name}" 模板`, 'info');
    }
}

function estimateExportSize() {
    const dataSource = document.getElementById('exportDataSource')?.value || 'requests';
    const timeRange = document.getElementById('exportTimeRange')?.value || '30d';
    const estimatedEl = document.getElementById('estimatedSize');

    let multiplier = 1;
    switch (timeRange) {
        case 'today': case 'yesterday': multiplier = 1/30; break;
        case '7d': multiplier = 7/30; break;
        case '90d': multiplier = 3; break;
        case 'custom': multiplier = 2; break;
    }

    const baseSize = dataSource === 'logs' ? 50 : dataSource === 'requests' ? 100 : 20;
    const estimated = Math.round(baseSize * multiplier);

    if (estimatedEl) {
        estimatedEl.value = estimated > 1000 ? `${(estimated/1000).toFixed(1)} MB` : `${estimated} KB`;
    }
}

async function startExport() {
    const modal = new bootstrap.Modal(document.getElementById('exportModal'));
    modal.show();

    setTimeout(() => {
        const modalBody = document.getElementById('exportModalBody');
        modalBody.innerHTML = `
            <div class="text-success mb-3"><i class="fas fa-check-circle fa-3x"></i></div>
            <p class="fw-bold">导出完成！</p>
            <p class="text-muted">文件已准备就绪，即将下载...</p>
        `;

        setTimeout(() => {
            modal.hide();
            downloadExportFile();
        }, 1500);
    }, 2000);
}

function downloadExportFile() {
    const dataSource = document.getElementById('exportDataSource')?.value || 'requests';
    const format = selectedExportFormat;
    const timeRange = document.getElementById('exportTimeRange')?.value || '30d';

    let content, filename, mimeType;

    if (format === 'csv') {
        content = generateCSVData();
        filename = `${dataSource}_export_${timeRange}_${formatDate(new Date())}.csv`;
        mimeType = 'text/csv;charset=utf-8';
    } else if (format === 'json') {
        content = JSON.stringify(mockReportData.preview, null, 2);
        filename = `${dataSource}_export_${timeRange}_${formatDate(new Date())}.json`;
        mimeType = 'application/json';
    } else if (format === 'excel') {
        content = generateCSVData();
        filename = `${dataSource}_export_${timeRange}_${formatDate(new Date())}.xlsx`;
        mimeType = 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet';
    } else {
        content = generateCSVData();
        filename = `${dataSource}_export_${timeRange}_${formatDate(new Date())}.pdf`;
        mimeType = 'application/pdf';
    }

    downloadFile(content, filename, mimeType);
    showToast(`文件 "${filename}" 已开始下载`, 'success');
}

function generateCSVData() {
    const headers = ['日期', '应用', '请求量', '成功数', '失败数', '成功率', '平均响应'];
    const rows = mockReportData.preview.map(row => [
        row.date, row.app, row.requests, row.success, row.fail, row.rate, row.avgResponse
    ]);

    return [headers.join(','), ...rows.map(r => r.join(','))].join('\n');
}

function exportReport() {
    downloadExportFile();
}

function saveReport() {
    const name = document.getElementById('reportName')?.value;
    const description = document.getElementById('reportDescription')?.value;

    if (!name) {
        showToast('请输入报表名称', 'warning');
        return;
    }

    showToast(`报表 "${name}" 已保存`, 'success');
    bootstrap.Modal.getInstance(document.getElementById('saveReportModal'))?.hide();

    document.getElementById('reportName').value = '';
    document.getElementById('reportDescription').value = '';

    let count = parseInt(document.getElementById('customReportsCount').textContent) || 0;
    document.getElementById('customReportsCount').textContent = count + 1;
}

function printPreview() {
    const preview = document.getElementById('reportPreview');
    if (preview) {
        const printWindow = window.open('', '_blank');
        printWindow.document.write(`
            <html>
            <head>
                <title>报表预览</title>
                <link rel="stylesheet" href="https://cdn.bootcdn.net/ajax/libs/twitter-bootstrap/5.3.8/css/bootstrap.min.css">
                <style>
                    body { padding: 20px; }
                    table { width: 100%; border-collapse: collapse; }
                    th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
                    th { background: #f8f9fa; }
                </style>
            </head>
            <body>
                <h3>自定义报表</h3>
                <p>生成时间: ${new Date().toLocaleString('zh-CN')}</p>
                ${preview.innerHTML}
            </body>
            </html>
        `);
        printWindow.document.close();
        printWindow.print();
    }
}

function createSchedule() {
    const name = document.getElementById('scheduleName')?.value;
    const frequency = document.getElementById('scheduleFrequency')?.value;
    const time = document.getElementById('scheduleTime')?.value;

    if (!name) {
        showToast('请输入任务名称', 'warning');
        return;
    }

    const tbody = document.getElementById('scheduleListBody');
    if (tbody) {
        const row = document.createElement('tr');
        row.innerHTML = `
            <td><i class="fas fa-clock me-2 text-warning"></i>${escapeHtml(name)}</td>
            <td>每${getFrequencyText(frequency)} ${time}</td>
            <td>即将执行</td>
            <td><span class="schedule-badge"><i class="fas fa-play"></i>运行中</span></td>
            <td>
                <button class="btn btn-sm btn-link text-primary"><i class="fas fa-edit"></i></button>
                <button class="btn btn-sm btn-link text-danger"><i class="fas fa-trash"></i></button>
            </td>
        `;
        tbody.insertBefore(row, tbody.firstChild);
    }

    showToast(`定时任务 "${name}" 已创建`, 'success');

    document.getElementById('scheduleName').value = '';

    let count = parseInt(document.getElementById('scheduleCount').textContent) || 0;
    document.getElementById('scheduleCount').textContent = count + 1;
}

function getFrequencyText(freq) {
    const map = { daily: '天', weekly: '周', monthly: '月' };
    return map[freq] || freq;
}

function getChartTypeName(type) {
    const map = { bar: '柱状图', line: '折线图', pie: '饼图', mixed: '混合图' };
    return map[type] || type;
}

function generateRandomData(count, min, max) {
    return Array.from({ length: count }, () => Math.floor(Math.random() * (max - min) + min));
}

function formatNumber(num) {
    if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
    if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
    return num.toString();
}

function formatDate(date) {
    return date.toISOString().slice(0, 10).replace(/-/g, '');
}

function downloadFile(content, filename, mimeType) {
    const blob = new Blob(['\ufeff' + content], { type: mimeType });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.style.display = 'none';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
}

function showToast(message, type = 'info') {
    const container = document.getElementById('toastContainer') || createToastContainer();
    const toast = document.createElement('div');
    toast.className = `toast align-items-center text-white bg-${type} border-0`;
    toast.setAttribute('role', 'alert');
    toast.innerHTML = `
        <div class="d-flex">
            <div class="toast-body">${escapeHtml(message)}</div>
            <button type="button" class="btn-close btn-close-white me-2 m-auto" data-bs-dismiss="toast"></button>
        </div>
    `;
    container.appendChild(toast);
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
