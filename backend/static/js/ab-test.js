document.addEventListener('DOMContentLoaded', function() {
    initTheme();
    loadApps();
    loadExperiments();
    loadStats();
});

let currentPage = 1;
let pageSize = 10;
let experiments = [];
let applications = [];

function initTheme() {
    const themeToggle = document.getElementById('themeToggle');
    const themeIcon = document.getElementById('themeIcon');
    const html = document.documentElement;
    const savedTheme = localStorage.getItem('theme') || 'light';
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    
    const initialTheme = savedTheme === 'auto' ? (prefersDark ? 'dark' : 'light') : savedTheme;
    html.setAttribute('data-bs-theme', initialTheme);
    updateThemeIcon(initialTheme);

    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
        if (localStorage.getItem('theme') === 'auto') {
            const newTheme = e.matches ? 'dark' : 'light';
            html.setAttribute('data-bs-theme', newTheme);
            updateThemeIcon(newTheme);
        }
    });

    themeToggle.addEventListener('click', toggleTheme);
}

function toggleTheme() {
    const html = document.documentElement;
    const currentTheme = html.getAttribute('data-bs-theme');
    const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
    html.setAttribute('data-bs-theme', newTheme);
    localStorage.setItem('theme', newTheme);
    updateThemeIcon(newTheme);
}

function updateThemeIcon(theme) {
    const themeIcon = document.getElementById('themeIcon');
    themeIcon.className = theme === 'dark' ? 'fas fa-sun' : 'fas fa-moon';
}

async function loadApps() {
    try {
        const response = await fetch('/api/v1/admin/applications');
        if (response.ok) {
            const data = await response.json();
            applications = data.data || [];
            const appFilter = document.getElementById('appFilter');
            const expApp = document.getElementById('expApp');
            
            appFilter.innerHTML = '<option value="">所有应用</option>';
            expApp.innerHTML = '<option value="">选择应用</option>';
            
            applications.forEach(app => {
                appFilter.innerHTML += `<option value="${app.id}">${app.name}</option>`;
                expApp.innerHTML += `<option value="${app.id}">${app.name}</option>`;
            });
        }
    } catch (error) {
        console.error('加载应用列表失败:', error);
        showToast('加载应用列表失败', 'error');
    }
}

async function loadStats() {
    try {
        const response = await fetch('/api/v1/admin/ab-tests/summary');
        if (response.ok) {
            const data = await response.json();
            document.getElementById('statTotal').textContent = data.total || 0;
            document.getElementById('statRunning').textContent = data.running || 0;
            document.getElementById('statStopped').textContent = data.stopped || 0;
            document.getElementById('statDraft').textContent = data.draft || 0;
        }
    } catch (error) {
        console.error('加载统计数据失败:', error);
    }
}

async function loadExperiments() {
    try {
        const search = document.getElementById('searchInput').value;
        const status = document.getElementById('statusFilter').value;
        const appId = document.getElementById('appFilter').value;
        
        let url = `/api/v1/admin/ab-tests?page=${currentPage}&page_size=${pageSize}`;
        if (search) url += `&keyword=${encodeURIComponent(search)}`;
        if (status) url += `&status=${status}`;
        if (appId) url += `&application_id=${appId}`;
        
        const response = await fetch(url);
        if (response.ok) {
            const data = await response.json();
            experiments = data.data || [];
            renderExperiments();
            renderPagination(data.total || 0);
        }
    } catch (error) {
        console.error('加载实验列表失败:', error);
        showToast('加载实验列表失败', 'error');
    }
}

function renderExperiments() {
    const container = document.getElementById('experimentsContainer');
    
    if (experiments.length === 0) {
        container.innerHTML = `
            <div class="col-12">
                <div class="card border-0 shadow-sm text-center py-5">
                    <div class="card-body">
                        <i class="fas fa-flask text-muted fs-1 mb-3"></i>
                        <h5 class="text-muted">暂无实验</h5>
                        <p class="text-muted mb-4">点击 "创建新实验" 按钮开始</p>
                        <button class="btn btn-primary" onclick="openCreateModal()">
                            <i class="fas fa-plus me-2"></i>创建新实验
                        </button>
                    </div>
                </div>
            </div>
        `;
        return;
    }
    
    container.innerHTML = experiments.map(exp => {
        const statusClass = exp.status === 'running' ? 'status-running' : 
                           exp.status === 'stopped' ? 'status-stopped' : 'status-draft';
        const statusText = exp.status === 'running' ? '进行中' : 
                          exp.status === 'stopped' ? '已停止' : '草稿';
        
        return `
            <div class="col-12 col-lg-6">
                <div class="card border-0 shadow-sm card-hover h-100">
                    <div class="card-body">
                        <div class="d-flex justify-content-between align-items-start mb-3">
                            <div>
                                <h5 class="fw-bold mb-1">${escapeHtml(exp.name)}</h5>
                                <span class="badge ${statusClass} text-white small">${statusText}</span>
                            </div>
                            <div class="dropdown">
                                <button class="btn btn-link text-muted p-0" type="button" data-bs-toggle="dropdown">
                                    <i class="fas fa-ellipsis-v"></i>
                                </button>
                                <ul class="dropdown-menu">
                                    <li><a class="dropdown-item" onclick="viewExperiment(${exp.id})"><i class="fas fa-eye me-2"></i>查看详情</a></li>
                                    ${exp.status === 'draft' ? `<li><a class="dropdown-item" onclick="startExperiment(${exp.id})"><i class="fas fa-play me-2"></i>启动实验</a></li>` : ''}
                                    ${exp.status === 'running' ? `<li><a class="dropdown-item" onclick="stopExperiment(${exp.id})"><i class="fas fa-stop me-2"></i>停止实验</a></li>` : ''}
                                    ${exp.status === 'draft' ? `<li><a class="dropdown-item" onclick="editExperiment(${exp.id})"><i class="fas fa-edit me-2"></i>编辑</a></li>` : ''}
                                    <li><hr class="dropdown-divider"></li>
                                    <li><a class="dropdown-item text-danger" onclick="deleteExperiment(${exp.id})"><i class="fas fa-trash me-2"></i>删除</a></li>
                                </ul>
                            </div>
                        </div>
                        
                        ${exp.description ? `<p class="text-muted small mb-3">${escapeHtml(exp.description)}</p>` : ''}
                        
                        <div class="mb-3">
                            <small class="text-muted">变体（${(exp.variants || []).length}）</small>
                            <div class="mt-2">
                                ${(exp.variants || []).map(v => `
                                    <div class="variant-card bg-light rounded p-2 mb-2">
                                        <div class="d-flex justify-content-between align-items-center">
                                            <div>
                                                <span class="fw-semibold">${escapeHtml(v.name)}</span>
                                                ${v.is_control ? '<span class="badge bg-primary ms-1">对照</span>' : ''}
                                            </div>
                                            <span class="text-muted small">${v.traffic_percent}% 流量</span>
                                        </div>
                                    </div>
                                `).join('')}
                            </div>
                        </div>
                        
                        <div class="d-flex justify-content-between align-items-center text-muted small">
                            <span><i class="fas fa-calendar-alt me-1"></i>${formatDate(exp.created_at)}</span>
                            ${exp.application ? `<span><i class="fas fa-folder me-1"></i>${escapeHtml(exp.application.name)}</span>` : ''}
                        </div>
                    </div>
                </div>
            </div>
        `;
    }).join('');
}

function renderPagination(total) {
    const container = document.getElementById('paginationContainer');
    const totalPages = Math.ceil(total / pageSize);
    
    if (totalPages <= 1) {
        container.innerHTML = '';
        return;
    }
    
    let html = '<nav><ul class="pagination">';
    
    html += `<li class="page-item ${currentPage === 1 ? 'disabled' : ''}">
        <a class="page-link" href="#" onclick="goToPage(${currentPage - 1}); return false;"><i class="fas fa-chevron-left"></i></a>
    </li>`;
    
    for (let i = 1; i <= totalPages; i++) {
        if (i === 1 || i === totalPages || (i >= currentPage - 2 && i <= currentPage + 2)) {
            html += `<li class="page-item ${i === currentPage ? 'active' : ''}">
                <a class="page-link" href="#" onclick="goToPage(${i}); return false;">${i}</a>
            </li>`;
        } else if (i === currentPage - 3 || i === currentPage + 3) {
            html += `<li class="page-item disabled"><span class="page-link">...</span></li>`;
        }
    }
    
    html += `<li class="page-item ${currentPage === totalPages ? 'disabled' : ''}">
        <a class="page-link" href="#" onclick="goToPage(${currentPage + 1}); return false;"><i class="fas fa-chevron-right"></i></a>
    </li>`;
    
    html += '</ul></nav>';
    container.innerHTML = html;
}

function goToPage(page) {
    currentPage = page;
    loadExperiments();
}

function refreshData() {
    loadExperiments();
    loadStats();
    showToast('数据已刷新', 'success');
}

function filterExperiments() {
    currentPage = 1;
    loadExperiments();
}

function openCreateModal() {
    document.getElementById('createForm').reset();
    document.getElementById('variantsContainer').innerHTML = '';
    
    addVariant(true);
    addVariant(false);
    
    const modal = new bootstrap.Modal(document.getElementById('createModal'));
    modal.show();
}

let variantIndex = 0;

function addVariant(isControl = false) {
    const container = document.getElementById('variantsContainer');
    const index = variantIndex++;
    
    const html = `
        <div class="variant-card bg-light rounded p-3 mb-3" id="variant-${index}">
            <div class="d-flex justify-content-between align-items-center mb-3">
                <h6 class="fw-bold mb-0"><i class="fas fa-layer-group me-2"></i>变体 ${index + 1}</h6>
                ${!isControl ? `<button type="button" class="btn btn-sm btn-outline-danger" onclick="removeVariant(${index})">
                    <i class="fas fa-times"></i>
                </button>` : ''}
            </div>
            <div class="row g-3">
                <div class="col-md-5">
                    <label class="form-label small fw-semibold">名称 <span class="text-danger">*</span></label>
                    <input type="text" class="form-control variant-name" value="${isControl ? '对照组' : '变体 ' + (index + 1)}" required>
                </div>
                <div class="col-md-3">
                    <label class="form-label small fw-semibold">流量占比 (%) <span class="text-danger">*</span></label>
                    <input type="number" class="form-control variant-traffic" min="0" max="100" value="${50}" required>
                </div>
                <div class="col-md-4">
                    <label class="form-label small fw-semibold">类型</label>
                    <div class="form-check mt-2">
                        <input class="form-check-input variant-control" type="checkbox" ${isControl ? 'checked' : ''} ${isControl ? 'disabled' : ''}>
                        <label class="form-check-label small">作为对照组</label>
                    </div>
                </div>
                <div class="col-12">
                    <label class="form-label small fw-semibold">描述</label>
                    <textarea class="form-control variant-description" rows="2" placeholder="描述此变体的特性"></textarea>
                </div>
            </div>
        </div>
    `;
    
    container.insertAdjacentHTML('beforeend', html);
}

function removeVariant(index) {
    const element = document.getElementById(`variant-${index}`);
    if (element) {
        element.remove();
    }
}

async function saveExperiment() {
    const name = document.getElementById('expName').value.trim();
    const description = document.getElementById('expDescription').value.trim();
    const applicationId = parseInt(document.getElementById('expApp').value);
    
    if (!name) {
        showToast('请输入实验名称', 'warning');
        return;
    }
    if (!applicationId) {
        showToast('请选择关联应用', 'warning');
        return;
    }
    
    const variants = [];
    const variantCards = document.querySelectorAll('[id^="variant-"]');
    let totalTraffic = 0;
    let hasControl = false;
    
    variantCards.forEach(card => {
        const vName = card.querySelector('.variant-name').value.trim();
        const vTraffic = parseInt(card.querySelector('.variant-traffic').value) || 0;
        const vControl = card.querySelector('.variant-control').checked;
        const vDescription = card.querySelector('.variant-description').value.trim();
        
        if (!vName) {
            showToast('请填写所有变体的名称', 'warning');
            throw new Error('变体名称不能为空');
        }
        
        variants.push({
            name: vName,
            traffic_percent: vTraffic,
            is_control: vControl,
            description: vDescription
        });
        
        totalTraffic += vTraffic;
        if (vControl) hasControl = true;
    });
    
    if (variants.length < 2) {
        showToast('至少需要两个变体', 'warning');
        return;
    }
    if (totalTraffic !== 100) {
        showToast(`流量总和必须为100%，当前为${totalTraffic}%`, 'warning');
        return;
    }
    if (!hasControl) {
        variants[0].is_control = true;
    }
    
    try {
        const response = await fetch('/api/v1/admin/ab-tests', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                name,
                description,
                application_id: applicationId,
                variants
            })
        });
        
        if (response.ok) {
            bootstrap.Modal.getInstance(document.getElementById('createModal')).hide();
            showToast('实验创建成功', 'success');
            refreshData();
        } else {
            const data = await response.json();
            showToast(data.message || '创建实验失败', 'error');
        }
    } catch (error) {
        console.error('创建实验失败:', error);
        showToast('创建实验失败', 'error');
    }
}

async function viewExperiment(id) {
    try {
        const [expResponse, reportResponse] = await Promise.all([
            fetch(`/api/v1/admin/ab-tests/${id}`),
            fetch(`/api/v1/admin/ab-tests/${id}/report`)
        ]);
        
        if (expResponse.ok && reportResponse.ok) {
            const experiment = await expResponse.json();
            const report = await reportResponse.json();
            renderViewModal(experiment.data, report);
            
            const modal = new bootstrap.Modal(document.getElementById('viewModal'));
            modal.show();
        }
    } catch (error) {
        console.error('获取实验详情失败:', error);
        showToast('获取实验详情失败', 'error');
    }
}

function renderViewModal(experiment, report) {
    const title = document.getElementById('viewModalTitle');
    const body = document.getElementById('viewModalBody');
    
    title.innerHTML = `<i class="fas fa-info-circle me-2"></i>${escapeHtml(experiment.name)}`;
    
    const statusText = experiment.status === 'running' ? '进行中' : 
                      experiment.status === 'stopped' ? '已停止' : '草稿';
    const statusClass = experiment.status === 'running' ? 'text-success' : 
                        experiment.status === 'stopped' ? 'text-secondary' : 'text-warning';
    
    body.innerHTML = `
        <div class="row mb-4">
            <div class="col-md-8">
                <p class="text-muted">${escapeHtml(experiment.description || '暂无描述')}</p>
                <div class="d-flex gap-3 mt-3">
                    <span class="badge bg-primary"><i class="fas fa-folder me-1"></i>${experiment.application?.name || '未知应用'}</span>
                    <span class="badge ${statusClass}"><i class="fas fa-circle me-1"></i>${statusText}</span>
                </div>
            </div>
            <div class="col-md-4 text-md-end">
                <small class="text-muted">创建于</small>
                <p class="mb-0 fw-semibold">${formatDate(experiment.created_at)}</p>
                ${experiment.start_date ? `<small class="text-muted">启动于 ${formatDate(experiment.start_date)}</small>` : ''}
            </div>
        </div>
        
        <hr>
        
        <h6 class="fw-bold mb-3"><i class="fas fa-chart-bar me-2"></i>实验数据</h6>
        <div class="row g-3 mb-4">
            <div class="col-md-3">
                <div class="card border-0 bg-light">
                    <div class="card-body text-center">
                        <h4 class="fw-bold text-primary">${report.total_visitors || 0}</h4>
                        <small class="text-muted">总访问量</small>
                    </div>
                </div>
            </div>
        </div>
        
        <h6 class="fw-bold mb-3"><i class="fas fa-layer-group me-2"></i>变体表现</h6>
        <div class="row g-3">
            ${(report.variants || []).map(v => {
                const controlRate = report.variants.find(x => x.is_control)?.conversion_rate || 0;
                const improvement = controlRate > 0 ? ((v.conversion_rate - controlRate) / controlRate * 100).toFixed(1) : 0;
                
                return `
                    <div class="col-md-6">
                        <div class="card border-0 ${v.is_control ? 'border-start border-4 border-primary' : ''}">
                            <div class="card-body">
                                <div class="d-flex justify-content-between align-items-center mb-2">
                                    <h6 class="fw-bold mb-0">${escapeHtml(v.variant_name)} ${v.is_control ? '<span class="badge bg-primary ms-1">对照</span>' : ''}</h6>
                                    <span class="text-muted small">${v.traffic_percent}% 流量</span>
                                </div>
                                <div class="row text-center mb-3">
                                    <div class="col-6">
                                        <div class="fw-bold fs-5">${v.visitors}</div>
                                        <div class="text-muted small">访问</div>
                                    </div>
                                    <div class="col-6">
                                        <div class="fw-bold fs-5">${v.conversions}</div>
                                        <div class="text-muted small">转化</div>
                                    </div>
                                </div>
                                <div class="mb-2">
                                    <div class="d-flex justify-content-between small mb-1">
                                        <span>转化率</span>
                                        <span class="fw-bold">${(v.conversion_rate || 0).toFixed(2)}%</span>
                                    </div>
                                    <div class="progress" style="height: 8px;">
                                        <div class="progress-bar progress-bar-conversion" style="width: ${Math.min(v.conversion_rate * 10, 100)}%"></div>
                                    </div>
                                </div>
                                ${!v.is_control ? `
                                    <div class="small ${improvement >= 0 ? 'text-success' : 'text-danger'}">
                                        <i class="fas fa-${improvement >= 0 ? 'arrow-up' : 'arrow-down'} me-1"></i>
                                        ${Math.abs(improvement)}% ${improvement >= 0 ? '提升' : '下降'}
                                        ${v.confidence > 95 ? '<span class="badge bg-success ms-1">显著</span>' : ''}
                                    </div>
                                    <div class="small text-muted">置信度: ${(v.confidence || 0).toFixed(1)}%</div>
                                ` : ''}
                            </div>
                        </div>
                    </div>
                `;
            }).join('')}
        </div>
        
        ${report.recommendations && report.recommendations.length > 0 ? `
            <hr>
            <h6 class="fw-bold mb-3"><i class="fas fa-lightbulb me-2"></i>建议</h6>
            <div class="alert alert-info">
                <ul class="mb-0">
                    ${report.recommendations.map(r => `<li>${escapeHtml(r)}</li>`).join('')}
                </ul>
            </div>
        ` : ''}
    `;
}

async function startExperiment(id) {
    showConfirmModal('启动实验', '确定要启动此实验吗？启动后将无法编辑配置。', async () => {
        try {
            const response = await fetch(`/api/v1/admin/ab-tests/${id}/start`, { method: 'POST' });
            if (response.ok) {
                showToast('实验已启动', 'success');
                refreshData();
            } else {
                const data = await response.json();
                showToast(data.message || '启动实验失败', 'error');
            }
        } catch (error) {
            console.error('启动实验失败:', error);
            showToast('启动实验失败', 'error');
        }
    });
}

async function stopExperiment(id) {
    showConfirmModal('停止实验', '确定要停止此实验吗？', async () => {
        try {
            const response = await fetch(`/api/v1/admin/ab-tests/${id}/stop`, { method: 'POST' });
            if (response.ok) {
                showToast('实验已停止', 'success');
                refreshData();
            } else {
                const data = await response.json();
                showToast(data.message || '停止实验失败', 'error');
            }
        } catch (error) {
            console.error('停止实验失败:', error);
            showToast('停止实验失败', 'error');
        }
    });
}

async function deleteExperiment(id) {
    showConfirmModal('删除实验', '确定要删除此实验吗？此操作不可恢复。', async () => {
        try {
            const response = await fetch(`/api/v1/admin/ab-tests/${id}`, { method: 'DELETE' });
            if (response.ok) {
                showToast('实验已删除', 'success');
                refreshData();
            } else {
                const data = await response.json();
                showToast(data.message || '删除实验失败', 'error');
            }
        } catch (error) {
            console.error('删除实验失败:', error);
            showToast('删除实验失败', 'error');
        }
    });
}

function showConfirmModal(title, message, callback) {
    document.getElementById('confirmModalTitle').innerHTML = `<i class="fas fa-exclamation-triangle me-2"></i>${title}`;
    document.getElementById('confirmModalBody').textContent = message;
    
    const confirmBtn = document.getElementById('confirmBtn');
    const newBtn = confirmBtn.cloneNode(true);
    confirmBtn.parentNode.replaceChild(newBtn, confirmBtn);
    
    newBtn.addEventListener('click', async () => {
        bootstrap.Modal.getInstance(document.getElementById('confirmModal')).hide();
        await callback();
    });
    
    const modal = new bootstrap.Modal(document.getElementById('confirmModal'));
    modal.show();
}

function editExperiment(id) {
    showToast('编辑功能开发中', 'info');
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function formatDate(dateStr) {
    if (!dateStr) return '-';
    const date = new Date(dateStr);
    return date.toLocaleDateString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
    });
}

function showToast(message, type = 'info') {
    const container = document.getElementById('toastContainer');
    
    const colors = {
        success: 'bg-success text-white',
        error: 'bg-danger text-white',
        warning: 'bg-warning text-dark',
        info: 'bg-primary text-white'
    };
    
    const icons = {
        success: 'fa-check-circle',
        error: 'fa-exclamation-circle',
        warning: 'fa-exclamation-triangle',
        info: 'fa-info-circle'
    };
    
    const toastId = 'toast-' + Date.now();
    const toastHtml = `
        <div id="${toastId}" class="toast ${colors[type]} fade show" role="alert">
            <div class="toast-header">
                <i class="fas ${icons[type]} me-2"></i>
                <strong class="me-auto">提示</strong>
                <button type="button" class="btn-close btn-close-white ms-2 mb-1" data-bs-dismiss="toast"></button>
            </div>
            <div class="toast-body">${escapeHtml(message)}</div>
        </div>
    `;
    
    container.insertAdjacentHTML('beforeend', toastHtml);
    const toastEl = document.getElementById(toastId);
    const toast = new bootstrap.Toast(toastEl, { delay: 4000 });
    toast.show();
    toastEl.addEventListener('hidden.bs.toast', () => toastEl.remove());
}
