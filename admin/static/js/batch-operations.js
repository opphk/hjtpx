let selectedItems = [];
let batchProgress = { success: 0, failed: 0, pending: 0, total: 0 };
let isExecuting = false;

document.addEventListener('DOMContentLoaded', () => {
    setupEventListeners();
    updateSelectedCount();
});

function setupEventListeners() {
    document.querySelectorAll('.item-card').forEach(card => {
        card.addEventListener('click', (e) => {
            if (e.target.type !== 'checkbox') {
                toggleItemSelection(card.dataset.id);
            }
        });

        const checkbox = card.querySelector('.item-checkbox');
        if (checkbox) {
            checkbox.addEventListener('change', () => {
                toggleItemSelection(card.dataset.id);
            });
        }
    });

    document.getElementById('selectAllApps')?.addEventListener('change', (e) => {
        const isChecked = e.target.checked;
        document.querySelectorAll('.item-select').forEach(checkbox => {
            checkbox.checked = isChecked;
        });

        if (isChecked) {
            selectedItems = Array.from(document.querySelectorAll('.item-card')).map(c => c.dataset.id);
        } else {
            selectedItems = [];
        }

        updateSelectedCount();
        updateItemCards();
    });

    document.getElementById('startBatchBtn')?.addEventListener('click', startBatchOperation);
    document.getElementById('applyConfigBtn')?.addEventListener('click', applyBatchConfig);
    document.getElementById('startExportBtn')?.addEventListener('click', startBatchExport);
    document.getElementById('refreshCacheBtn')?.addEventListener('click', () => confirmAction('refreshCache', '确定要刷新所有应用的缓存吗？'));
    document.getElementById('resetKeysBtn')?.addEventListener('click', () => confirmAction('resetKeys', '确定要重置选中应用的密钥吗？'));
    document.getElementById('syncConfigBtn')?.addEventListener('click', () => confirmAction('syncConfig', '确定要同步配置吗？'));
    document.getElementById('clearLogsBtn')?.addEventListener('click', () => confirmAction('clearLogs', '确定要清理日志吗？此操作不可恢复！'));
}

function toggleItemSelection(id) {
    const index = selectedItems.indexOf(id);
    if (index === -1) {
        selectedItems.push(id);
    } else {
        selectedItems.splice(index, 1);
    }

    updateSelectedCount();
    updateItemCards();
}

function updateSelectedCount() {
    const countEl = document.getElementById('selectedCount');
    if (countEl) {
        countEl.textContent = selectedItems.length;
    }
}

function updateItemCards() {
    document.querySelectorAll('.item-card').forEach(card => {
        if (selectedItems.includes(card.dataset.id)) {
            card.classList.add('selected');
            card.querySelector('.item-checkbox').checked = true;
        } else {
            card.classList.remove('selected');
            card.querySelector('.item-checkbox').checked = false;
        }
    });
}

function startBatchOperation() {
    if (selectedItems.length === 0) {
        showToast('请先选择要操作的项目', 'warning');
        return;
    }

    batchProgress = {
        success: 0,
        failed: 0,
        pending: selectedItems.length,
        total: selectedItems.length
    };

    isExecuting = true;
    updateProgressUI();

    simulateBatchExecution();
}

function simulateBatchExecution() {
    if (!isExecuting || batchProgress.pending === 0) {
        isExecuting = false;
        showToast(`批量操作完成！成功: ${batchProgress.success}, 失败: ${batchProgress.failed}`, batchProgress.failed > 0 ? 'warning' : 'success');
        return;
    }

    setTimeout(() => {
        const success = Math.random() > 0.1;
        batchProgress.pending--;
        if (success) {
            batchProgress.success++;
        } else {
            batchProgress.failed++;
        }

        updateProgressUI();
        addLogEntry(success ? 'success' : 'error', `应用 ${selectedItems[batchProgress.total - batchProgress.pending]} ${success ? '更新成功' : '更新失败'}`);

        simulateBatchExecution();
    }, 500 + Math.random() * 500);
}

function updateProgressUI() {
    const successEl = document.getElementById('successCount');
    const failedEl = document.getElementById('failedCount');
    const pendingEl = document.getElementById('pendingCount');
    const progressBar = document.getElementById('batchProgress');
    const progressPercent = document.getElementById('progressPercent');

    if (successEl) successEl.textContent = batchProgress.success;
    if (failedEl) failedEl.textContent = batchProgress.failed;
    if (pendingEl) pendingEl.textContent = batchProgress.pending;

    const percent = Math.round(((batchProgress.total - batchProgress.pending) / batchProgress.total) * 100);
    if (progressBar) progressBar.style.width = `${percent}%`;
    if (progressPercent) progressPercent.textContent = `${percent}%`;
}

function addLogEntry(type, message) {
    const logContainer = document.getElementById('operationLog');
    if (!logContainer) return;

    const time = new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' });

    const entry = document.createElement('div');
    entry.className = 'log-entry';
    entry.innerHTML = `
        <span class="log-time">${time}</span>
        <span class="log-status ${type}"></span>
        <span>${escapeHtml(message)}</span>
    `;

    logContainer.insertBefore(entry, logContainer.firstChild);

    if (logContainer.children.length > 20) {
        logContainer.removeChild(logContainer.lastChild);
    }
}

function applyBatchConfig() {
    const securityLevel = document.getElementById('batchSecurityLevel')?.value;
    const timeout = document.getElementById('batchTimeout')?.value;
    const retry = document.getElementById('batchRetry')?.value;
    const notifyEmail = document.getElementById('notifyEmail')?.checked;
    const notifyWebhook = document.getElementById('notifyWebhook')?.checked;

    if (selectedItems.length === 0) {
        showToast('请先选择要配置的应用', 'warning');
        return;
    }

    if (!securityLevel && !timeout && !retry && !notifyEmail && !notifyWebhook) {
        showToast('请至少选择一项配置进行修改', 'warning');
        return;
    }

    const configSummary = [];
    if (securityLevel) configSummary.push(`安全级别: ${securityLevel}`);
    if (timeout) configSummary.push(`超时时间: ${timeout}秒`);
    if (retry) configSummary.push(`自动重试: ${retry}次`);
    if (notifyEmail) configSummary.push('邮件通知: 开启');
    if (notifyWebhook) configSummary.push('Webhook: 开启');

    showToast(`配置已应用到 ${selectedItems.length} 个应用: ${configSummary.join(', ')}`, 'success');
}

function startBatchExport() {
    const exportType = document.getElementById('exportType')?.value;
    const exportFormat = document.querySelector('input[name="exportFormat"]:checked')?.value;
    const exportApps = document.getElementById('exportApps')?.value;

    showToast(`正在导出 ${exportApps === 'all' ? '全部应用' : exportApps === 'selected' ? `${selectedItems.length} 个选中应用` : '活跃应用'} 的 ${exportType} 数据 (${exportFormat.toUpperCase()})`, 'info');

    setTimeout(() => {
        const content = generateExportContent(exportType);
        const filename = `batch_export_${exportType}_${formatDate(new Date())}.${exportFormat === 'excel' ? 'csv' : exportFormat}`;
        downloadFile(content, filename, getMimeType(exportFormat));
        showToast('导出完成！文件已开始下载', 'success');
    }, 1500);
}

function generateExportContent(type) {
    const apps = [
        { id: 1, name: '用户中心', status: 'active', requests: 15678 },
        { id: 2, name: '支付系统', status: 'active', requests: 23456 },
        { id: 3, name: '消息推送', status: 'suspended', requests: 8765 },
        { id: 4, name: '数据分析', status: 'active', requests: 12345 },
        { id: 5, name: '文件存储', status: 'inactive', requests: 0 }
    ];

    let headers, rows;

    switch (type) {
        case 'applications':
            headers = ['ID', '名称', '状态', '日请求量'];
            rows = apps.map(a => [a.id, a.name, a.status, a.requests]);
            break;
        case 'logs':
            headers = ['时间', '操作', '应用', '结果'];
            rows = [
                ['2024-01-15 14:30:15', '配置更新', '用户中心', '成功'],
                ['2024-01-15 14:30:14', '配置更新', '支付系统', '成功'],
                ['2024-01-15 14:30:13', '配置更新', '消息推送', '失败']
            ];
            break;
        case 'stats':
            headers = ['应用', '总请求', '成功率', '平均响应'];
            rows = apps.map(a => [a.name, a.requests * 30, '98.5%', '125ms']);
            break;
        case 'security':
            headers = ['时间', '事件', '风险等级', '处理方式'];
            rows = [
                ['2024-01-15 14:30:00', '暴力破解', '高', '拦截'],
                ['2024-01-15 14:25:00', '异常IP', '中', '警告'],
                ['2024-01-15 14:20:00', '频率超限', '低', '限制']
            ];
            break;
        default:
            headers = ['ID', '数据'];
            rows = apps.map(a => [a.id, a.name]);
    }

    return [headers.join(','), ...rows.map(r => r.join(','))].join('\n');
}

function getMimeType(format) {
    switch (format) {
        case 'csv': return 'text/csv;charset=utf-8';
        case 'json': return 'application/json';
        case 'excel': return 'text/csv;charset=utf-8';
        default: return 'text/plain';
    }
}

function confirmAction(action, message) {
    const modal = document.getElementById('confirmModal');
    const body = document.getElementById('confirmModalBody');
    const confirmBtn = document.getElementById('confirmActionBtn');

    if (body) {
        body.innerHTML = `<p>${escapeHtml(message)}</p>`;
    }

    if (confirmBtn) {
        confirmBtn.onclick = () => {
            executeAction(action);
            bootstrap.Modal.getInstance(modal)?.hide();
        };
    }

    const bsModal = new bootstrap.Modal(modal);
    bsModal.show();
}

function executeAction(action) {
    let message = '';

    switch (action) {
        case 'refreshCache':
            message = '缓存刷新任务已启动';
            addLogEntry('success', '批量缓存刷新完成');
            break;
        case 'resetKeys':
            message = `正在重置 ${selectedItems.length} 个应用的密钥...`;
            addLogEntry('pending', message);
            setTimeout(() => {
                addLogEntry('success', '密钥重置完成');
            }, 2000);
            break;
        case 'syncConfig':
            message = '配置同步任务已启动';
            addLogEntry('success', '配置同步完成');
            break;
        case 'clearLogs':
            message = '日志清理任务已启动';
            addLogEntry('success', '日志清理完成，已删除 12,345 条记录');
            break;
    }

    showToast(message, 'info');
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

function formatDate(date) {
    return date.toISOString().slice(0, 10).replace(/-/g, '');
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
