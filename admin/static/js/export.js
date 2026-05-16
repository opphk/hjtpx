let exportModal, currentExportType = 'csv';

document.addEventListener('DOMContentLoaded', () => {
    initExportModal();
    setupEventListeners();
});

function initExportModal() {
    const modalEl = document.getElementById('exportModal');
    if (modalEl) {
        exportModal = new bootstrap.Modal(modalEl);
    }

    const exportRange = document.getElementById('exportRange');
    if (exportRange) {
        exportRange.addEventListener('change', (e) => {
            const customRange = document.getElementById('customDateRange');
            if (e.target.value === 'custom') {
                customRange.classList.remove('d-none');
            } else {
                customRange.classList.add('d-none');
            }
        });
    }

    const confirmExportBtn = document.getElementById('confirmExportBtn');
    if (confirmExportBtn) {
        confirmExportBtn.addEventListener('click', handleExport);
    }
}

function setupEventListeners() {
    const exportBtn = document.getElementById('exportDashboardBtn');
    if (exportBtn) {
        exportBtn.addEventListener('click', () => {
            if (exportModal) {
                exportModal.show();
            }
        });
    }

    document.querySelectorAll('input[name="exportFormat"]').forEach(radio => {
        radio.addEventListener('change', (e) => {
            currentExportType = e.target.value;
        });
    });
}

async function handleExport() {
    const format = currentExportType;
    const range = document.getElementById('exportRange')?.value || 'month';
    const startDate = document.getElementById('exportStartDate')?.value;
    const endDate = document.getElementById('exportEndDate')?.value;

    const exportData = collectExportData(range, startDate, endDate);

    if (format === 'csv' || format === 'excel') {
        exportAsCSV(exportData);
    } else if (format === 'pdf') {
        exportAsPDF(exportData);
    }

    if (exportModal) {
        exportModal.hide();
    }
}

function collectExportData(range, startDate, endDate) {
    const data = {
        title: '管理后台数据报告',
        exportTime: new Date().toLocaleString('zh-CN'),
        range: range,
        dateRange: {
            start: startDate || getRangeStartDate(range),
            end: endDate || new Date().toISOString().slice(0, 10)
        },
        summary: {
            totalUsers: document.getElementById('totalUsers')?.textContent || '0',
            totalApps: document.getElementById('totalApps')?.textContent || '0',
            totalRequests: document.getElementById('totalRequests')?.textContent || '0',
            totalErrors: document.getElementById('totalErrors')?.textContent || '0',
            totalSuccess: document.getElementById('totalSuccess')?.textContent || '0',
            totalFail: document.getElementById('totalFail')?.textContent || '0',
            totalRisk: document.getElementById('totalRisk')?.textContent || '0'
        },
        trends: [],
        riskDistribution: [],
        topApps: []
    };

    return data;
}

function getRangeStartDate(range) {
    const now = new Date();
    switch (range) {
        case 'today':
            return now.toISOString().slice(0, 10);
        case 'week':
            return new Date(now - 7 * 24 * 60 * 60 * 1000).toISOString().slice(0, 10);
        case 'month':
            return new Date(now - 30 * 24 * 60 * 60 * 1000).toISOString().slice(0, 10);
        default:
            return new Date(now - 30 * 24 * 60 * 60 * 1000).toISOString().slice(0, 10);
    }
}

function exportAsCSV(data) {
    let csvContent = '\ufeff';
    csvContent += '管理后台数据报告\n\n';
    csvContent += `导出时间,${data.exportTime}\n`;
    csvContent += `数据范围,${data.range}\n`;
    csvContent += `开始日期,${data.dateRange.start}\n`;
    csvContent += `结束日期,${data.dateRange.end}\n\n`;
    csvContent += '指标,数值\n';
    csvContent += `总用户数,${data.summary.totalUsers}\n`;
    csvContent += `应用总数,${data.summary.totalApps}\n`;
    csvContent += `请求总数,${data.summary.totalRequests}\n`;
    csvContent += `错误总数,${data.summary.totalErrors}\n`;
    csvContent += `验证成功,${data.summary.totalSuccess}\n`;
    csvContent += `验证失败,${data.summary.totalFail}\n`;
    csvContent += `风险预警,${data.summary.totalRisk}\n`;

    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8' });
    downloadBlob(blob, `dashboard_report_${data.dateRange.start}_${data.dateRange.end}.csv`);
}

function exportAsPDF(data) {
    const pdfContent = generatePDFHTML(data);
    const blob = new Blob([pdfContent], { type: 'application/pdf' });
    downloadBlob(blob, `dashboard_report_${data.dateRange.start}_${data.dateRange.end}.html`);
}

function generatePDFHTML(data) {
    return `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>管理后台数据报告</title>
    <style>
        body { font-family: "Microsoft YaHei", Arial, sans-serif; padding: 40px; color: #333; }
        h1 { color: #1a1a2e; border-bottom: 3px solid #c9a96e; padding-bottom: 10px; }
        h2 { color: #b8924e; margin-top: 30px; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 12px; text-align: left; }
        th { background: #f5f5f5; font-weight: bold; }
        .summary-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 15px; margin: 20px 0; }
        .summary-item { background: #f9f9f9; padding: 15px; border-radius: 8px; text-align: center; }
        .summary-value { font-size: 24px; font-weight: bold; color: #1a1a2e; }
        .summary-label { color: #666; font-size: 12px; margin-top: 5px; }
        .footer { margin-top: 40px; text-align: center; color: #999; font-size: 12px; }
        @media print { body { padding: 20px; } }
    </style>
</head>
<body>
    <h1>🛡️ ${data.title}</h1>
    <p><strong>导出时间:</strong> ${data.exportTime}</p>
    <p><strong>数据范围:</strong> ${data.dateRange.start} 至 ${data.dateRange.end}</p>
    
    <h2>📊 数据概览</h2>
    <div class="summary-grid">
        <div class="summary-item">
            <div class="summary-value">${data.summary.totalUsers}</div>
            <div class="summary-label">总用户数</div>
        </div>
        <div class="summary-item">
            <div class="summary-value">${data.summary.totalApps}</div>
            <div class="summary-label">应用总数</div>
        </div>
        <div class="summary-item">
            <div class="summary-value">${data.summary.totalRequests}</div>
            <div class="summary-label">请求总数</div>
        </div>
        <div class="summary-item">
            <div class="summary-value">${data.summary.totalErrors}</div>
            <div class="summary-label">错误总数</div>
        </div>
    </div>
    
    <h2>✅ 验证统计</h2>
    <table>
        <tr><th>指标</th><th>数值</th></tr>
        <tr><td>验证成功</td><td>${data.summary.totalSuccess}</td></tr>
        <tr><td>验证失败</td><td>${data.summary.totalFail}</td></tr>
        <tr><td>风险预警</td><td>${data.summary.totalRisk}</td></tr>
    </table>
    
    <div class="footer">
        <p>本报告由墨盾验证管理系统自动生成</p>
        <p>如有问题请联系技术支持</p>
    </div>
</body>
</html>`;
}

function downloadBlob(blob, filename) {
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    link.style.display = 'none';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
}

function exportStatsReport() {
    const dateRange = document.getElementById('dateRange')?.value || '30d';
    const reportData = {
        exportTime: new Date().toLocaleString('zh-CN'),
        dateRange: dateRange,
        summary: {
            totalRequests: document.getElementById('statTotalRequests')?.textContent || '0',
            avgResponse: document.getElementById('statAvgResponse')?.textContent || '0ms',
            successRate: document.getElementById('statSuccessRate')?.textContent || '0%',
            activeUsers: document.getElementById('statActiveUsers')?.textContent || '0'
        }
    };

    let csvContent = '\ufeff';
    csvContent += '统计分析报表\n\n';
    csvContent += `导出时间,${reportData.exportTime}\n`;
    csvContent += `时间范围,${reportData.dateRange}\n\n`;
    csvContent += '指标,数值\n';
    csvContent += `总请求量,${reportData.summary.totalRequests}\n`;
    csvContent += `平均响应时间,${reportData.summary.avgResponse}\n`;
    csvContent += `成功率,${reportData.summary.successRate}\n`;
    csvContent += `活跃用户,${reportData.summary.activeUsers}\n`;

    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8' });
    downloadBlob(blob, `stats_report_${reportData.dateRange}_${new Date().toISOString().slice(0, 10)}.csv`);
}
