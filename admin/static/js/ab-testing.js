let charts = {};
let currentPage = 1;
let pageSize = 10;
let applications = [];
let currentTestId = null;
let variantColors = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899'];

document.addEventListener('DOMContentLoaded', async function() {
    setupEventListeners();
    await loadApplications();
    await loadSummary();
    await loadTests();
    initializeDefaultVariants();
});

function setupEventListeners() {
    // 刷新按钮
    document.getElementById('refreshBtn').addEventListener('click', refreshAll);
    
    // 过滤和搜索
    document.getElementById('statusFilter').addEventListener('change', loadTests);
    document.getElementById('applicationFilter').addEventListener('change', loadTests);
    document.getElementById('searchBtn').addEventListener('click', loadTests);
    document.getElementById('searchInput').addEventListener('keypress', (e) => {
        if (e.key === 'Enter') loadTests();
    });
    
    // 创建测试
    document.getElementById('addVariantBtn').addEventListener('click', addVariant);
    document.getElementById('saveTestBtn').addEventListener('click', saveTest);
    
    // 测试选择
    document.getElementById('analyticsTestSelect').addEventListener('change', loadTestAnalytics);
    
    // Tab切换
    const tabs = document.querySelectorAll('#abtestTabs button');
    tabs.forEach(tab => {
        tab.addEventListener('shown.bs.tab', handleTabChange);
    });
}

async function refreshAll() {
    await Promise.all([
        loadSummary(),
        loadTests(),
        loadApplications()
    ]);
}

function handleTabChange(event) {
    if (event.target.id === 'analytics-tab') {
        populateTestSelect();
    }
}

async function loadApplications() {
    try {
        const data = await auth.request('/api/v1/admin/applications?page_size=100');
        if (data.code === 0 && data.data) {
            applications = data.data.data || [];
            populateApplicationSelects();
        }
    } catch (error) {
        console.error('Failed to load applications:', error);
    }
}

function populateApplicationSelects() {
    const filterSelect = document.getElementById('applicationFilter');
    const formSelect = document.getElementById('testApplication');
    
    const options = applications.map(app => 
        `<option value="${app.id}">${escapeHtml(app.name)}</option>`
    ).join('');
    
    if (filterSelect) {
        filterSelect.innerHTML = '<option value="">全部应用</option>' + options;
    }
    if (formSelect) {
        formSelect.innerHTML = '<option value="">选择应用</option>' + options;
    }
}

async function loadSummary() {
    try {
        const data = await auth.request('/api/v1/admin/ab-testing/summary?application_id=');
        if (data.code === 0 && data.data) {
            renderSummary(data.data);
        } else {
            renderSummary(getMockSummary());
        }
    } catch (error) {
        renderSummary(getMockSummary());
    }
}

function getMockSummary() {
    return {
        total: 5,
        running: 2,
        stopped: 2,
        draft: 1
    };
}

function renderSummary(summary) {
    const container = document.getElementById('summaryMetrics');
    container.innerHTML = `
        <div class="col-md-3">
            <div class="card metric-card primary">
                <div class="card-body">
                    <div class="text-muted small">总测试</div>
                    <div class="fs-3 fw-bold">${summary.total || 0}</div>
                </div>
            </div>
        </div>
        <div class="col-md-3">
            <div class="card metric-card success">
                <div class="card-body">
                    <div class="text-muted small">运行中</div>
                    <div class="fs-3 fw-bold">${summary.running || 0}</div>
                </div>
            </div>
        </div>
        <div class="col-md-3">
            <div class="card metric-card warning">
                <div class="card-body">
                    <div class="text-muted small">已停止</div>
                    <div class="fs-3 fw-bold">${summary.stopped || 0}</div>
                </div>
            </div>
        </div>
        <div class="col-md-3">
            <div class="card metric-card danger">
                <div class="card-body">
                    <div class="text-muted small">草稿</div>
                    <div class="fs-3 fw-bold">${summary.draft || 0}</div>
                </div>
            </div>
        </div>
    `;
}

async function loadTests() {
    try {
        const status = document.getElementById('statusFilter').value;
        const appId = document.getElementById('applicationFilter').value;
        const keyword = document.getElementById('searchInput').value;
        
        let url = `/api/v1/admin/ab-testing?page=${currentPage}&page_size=${pageSize}`;
        if (status) url += `&status=${status}`;
        if (appId) url += `&application_id=${appId}`;
        if (keyword) url += `&keyword=${encodeURIComponent(keyword)}`;
        
        const data = await auth.request(url);
        if (data.code === 0 && data.data) {
            renderTests(data.data);
        } else {
            renderTests(getMockTests());
        }
    } catch (error) {
        renderTests(getMockTests());
    }
}

function getMockTests() {
    const now = new Date();
    return {
        data: [
            {
                id: 1,
                name: '登录页按钮颜色测试',
                description: '测试蓝色和绿色按钮对点击率的影响',
                application_id: 1,
                status: 'running',
                start_date: new Date(now - 7 * 24 * 60 * 60 * 1000).toISOString(),
                created_at: new Date(now - 10 * 24 * 60 * 60 * 1000).toISOString(),
                variants: [
                    { id: 1, name: '控制组（蓝色）', is_control: true, traffic_percent: 50 },
                    { id: 2, name: '变体A（绿色）', is_control: false, traffic_percent: 50 }
                ]
            },
            {
                id: 2,
                name: '表单布局优化',
                description: '测试单栏和双栏布局的完成率',
                application_id: 1,
                status: 'draft',
                created_at: new Date(now - 2 * 24 * 60 * 60 * 1000).toISOString(),
                variants: [
                    { id: 3, name: '单栏布局', is_control: true, traffic_percent: 50 },
                    { id: 4, name: '双栏布局', is_control: false, traffic_percent: 50 }
                ]
            },
            {
                id: 3,
                name: '验证码类型对比',
                description: '对比滑块验证码和点击验证码',
                application_id: 2,
                status: 'stopped',
                start_date: new Date(now - 30 * 24 * 60 * 60 * 1000).toISOString(),
                end_date: new Date(now - 7 * 24 * 60 * 60 * 1000).toISOString(),
                created_at: new Date(now - 35 * 24 * 60 * 60 * 1000).toISOString(),
                variants: [
                    { id: 5, name: '滑块验证码', is_control: true, traffic_percent: 50 },
                    { id: 6, name: '点击验证码', is_control: false, traffic_percent: 50 }
                ]
            }
        ],
        total: 3,
        page: 1,
        page_size: 10
    };
}

function renderTests(result) {
    const container = document.getElementById('testsList');
    const tests = result.data || [];
    
    if (tests.length === 0) {
        container.innerHTML = `
            <div class="text-center py-5">
                <i class="fas fa-flask text-muted" style="font-size: 3rem;"></i>
                <p class="text-muted mt-3">暂无测试，点击"新建测试"开始</p>
            </div>
        `;
        return;
    }
    
    container.innerHTML = tests.map(test => {
        const app = applications.find(a => a.id === test.application_id);
        const statusClass = `test-status-${test.status}`;
        const statusText = {
            'draft': '草稿',
            'running': '运行中',
            'stopped': '已停止'
        }[test.status] || test.status;
        
        return `
            <div class="test-card p-4 mb-3" onclick="viewTest(${test.id})">
                <div class="d-flex justify-content-between align-items-start">
                    <div class="flex-grow-1">
                        <div class="d-flex align-items-center gap-2 mb-2">
                            <h5 class="mb-0">${escapeHtml(test.name)}</h5>
                            <span class="badge ${statusClass} text-white">${statusText}</span>
                        </div>
                        <p class="text-muted mb-2">${escapeHtml(test.description || '无描述')}</p>
                        <div class="d-flex gap-4 text-muted small">
                            <span><i class="fas fa-mobile-alt me-1"></i>${app ? escapeHtml(app.name) : '未知应用'}</span>
                            <span><i class="fas fa-calendar me-1"></i>${formatDate(test.created_at)}</span>
                            ${test.start_date ? `<span><i class="fas fa-play me-1"></i>开始于 ${formatDate(test.start_date)}</span>` : ''}
                        </div>
                    </div>
                    <div class="text-end">
                        <div class="text-muted small mb-2">${test.variants?.length || 0} 个变体</div>
                        <div class="d-flex gap-2 justify-content-end">
                            ${test.variants?.map((v, i) => `
                                <div class="text-center">
                                    <div class="small text-muted">${escapeHtml(v.name)}</div>
                                    <div class="fw-bold">${v.traffic_percent}%</div>
                                </div>
                            `).join('') || ''}
                        </div>
                    </div>
                </div>
            </div>
        `;
    }).join('');
    
    renderPagination(result);
}

function renderPagination(result) {
    const container = document.getElementById('testsPagination');
    const totalPages = Math.ceil(result.total / pageSize);
    
    if (totalPages <= 1) {
        container.innerHTML = '';
        return;
    }
    
    let pages = '';
    for (let i = 1; i <= totalPages; i++) {
        const active = i === currentPage ? 'active' : '';
        pages += `<li class="page-item ${active}"><a class="page-link" href="#" onclick="goToPage(${i}); return false;">${i}</a></li>`;
    }
    
    container.innerHTML = `
        <ul class="pagination justify-content-center">
            <li class="page-item ${currentPage === 1 ? 'disabled' : ''}">
                <a class="page-link" href="#" onclick="goToPage(${currentPage - 1}); return false;">
                    <i class="fas fa-chevron-left"></i>
                </a>
            </li>
            ${pages}
            <li class="page-item ${currentPage === totalPages ? 'disabled' : ''}">
                <a class="page-link" href="#" onclick="goToPage(${currentPage + 1}); return false;">
                    <i class="fas fa-chevron-right"></i>
                </a>
            </li>
        </ul>
    `;
}

function goToPage(page) {
    currentPage = page;
    loadTests();
}

function initializeDefaultVariants() {
    const container = document.getElementById('variantsContainer');
    container.innerHTML = `
        <div class="variant-form mb-3" data-index="0">
            <div class="row g-3">
                <div class="col-md-4">
                    <label class="form-label">变体名称 <span class="text-danger">*</span></label>
                    <input type="text" class="form-control variant-name" value="控制组" required>
                </div>
                <div class="col-md-2">
                    <label class="form-label">流量占比 <span class="text-danger">*</span></label>
                    <input type="number" class="form-control variant-traffic" value="50" min="0" max="100" required>
                </div>
                <div class="col-md-1 d-flex align-items-end">
                    <div class="form-check">
                        <input class="form-check-input variant-control" type="checkbox" checked disabled>
                        <label class="form-check-label small">控制组</label>
                    </div>
                </div>
                <div class="col-md-4">
                    <label class="form-label">配置（JSON）</label>
                    <textarea class="form-control variant-config font-monospace" rows="1" placeholder='{"color": "blue"}'></textarea>
                </div>
                <div class="col-md-1 d-flex align-items-end">
                </div>
            </div>
            <div class="mt-2">
                <input type="text" class="form-control form-control-sm variant-description" placeholder="变体描述">
            </div>
        </div>
        <div class="variant-form mb-3" data-index="1">
            <div class="row g-3">
                <div class="col-md-4">
                    <label class="form-label">变体名称 <span class="text-danger">*</span></label>
                    <input type="text" class="form-control variant-name" value="变体A" required>
                </div>
                <div class="col-md-2">
                    <label class="form-label">流量占比 <span class="text-danger">*</span></label>
                    <input type="number" class="form-control variant-traffic" value="50" min="0" max="100" required>
                </div>
                <div class="col-md-1 d-flex align-items-end">
                    <div class="form-check">
                        <input class="form-check-input variant-control" type="checkbox">
                        <label class="form-check-label small">控制组</label>
                    </div>
                </div>
                <div class="col-md-4">
                    <label class="form-label">配置（JSON）</label>
                    <textarea class="form-control variant-config font-monospace" rows="1" placeholder='{"color": "green"}'></textarea>
                </div>
                <div class="col-md-1 d-flex align-items-end">
                    <button type="button" class="btn btn-outline-danger btn-sm" onclick="removeVariant(this)">
                        <i class="fas fa-trash"></i>
                    </button>
                </div>
            </div>
            <div class="mt-2">
                <input type="text" class="form-control form-control-sm variant-description" placeholder="变体描述">
            </div>
        </div>
    `;
}

function addVariant() {
    const container = document.getElementById('variantsContainer');
    const forms = container.querySelectorAll('.variant-form');
    const index = forms.length;
    
    const form = document.createElement('div');
    form.className = 'variant-form mb-3';
    form.dataset.index = index;
    form.innerHTML = `
        <div class="row g-3">
            <div class="col-md-4">
                <label class="form-label">变体名称 <span class="text-danger">*</span></label>
                <input type="text" class="form-control variant-name" value="变体${String.fromCharCode(65 + index - 1)}" required>
            </div>
            <div class="col-md-2">
                <label class="form-label">流量占比 <span class="text-danger">*</span></label>
                <input type="number" class="form-control variant-traffic" value="0" min="0" max="100" required>
            </div>
            <div class="col-md-1 d-flex align-items-end">
                <div class="form-check">
                    <input class="form-check-input variant-control" type="checkbox">
                    <label class="form-check-label small">控制组</label>
                </div>
            </div>
            <div class="col-md-4">
                <label class="form-label">配置（JSON）</label>
                <textarea class="form-control variant-config font-monospace" rows="1" placeholder='{"key": "value"}'></textarea>
            </div>
            <div class="col-md-1 d-flex align-items-end">
                <button type="button" class="btn btn-outline-danger btn-sm" onclick="removeVariant(this)">
                    <i class="fas fa-trash"></i>
                </button>
            </div>
        </div>
        <div class="mt-2">
            <input type="text" class="form-control form-control-sm variant-description" placeholder="变体描述">
        </div>
    `;
    
    container.appendChild(form);
}

function removeVariant(btn) {
    const form = btn.closest('.variant-form');
    if (form) {
        form.remove();
    }
}

async function saveTest() {
    const name = document.getElementById('testName').value;
    const description = document.getElementById('testDescription').value;
    const applicationId = document.getElementById('testApplication').value;
    const configText = document.getElementById('testConfig').value;
    
    if (!name || !applicationId) {
        alert('请填写必填字段');
        return;
    }
    
    // 收集变体
    const variants = [];
    const forms = document.querySelectorAll('.variant-form');
    let totalTraffic = 0;
    
    forms.forEach((form, index) => {
        const variantName = form.querySelector('.variant-name').value;
        const traffic = parseInt(form.querySelector('.variant-traffic').value) || 0;
        const isControl = form.querySelector('.variant-control').checked;
        const config = form.querySelector('.variant-config').value;
        const desc = form.querySelector('.variant-description').value;
        
        if (!variantName) {
            alert(`请填写第 ${index + 1} 个变体的名称`);
            return;
        }
        
        totalTraffic += traffic;
        
        let variantConfig = {};
        if (config) {
            try {
                variantConfig = JSON.parse(config);
            } catch (e) {
                alert(`第 ${index + 1} 个变体的配置JSON格式错误`);
                return;
            }
        }
        
        variants.push({
            name: variantName,
            is_control: isControl,
            traffic_percent: traffic,
            config: variantConfig,
            description: desc
        });
    });
    
    if (totalTraffic !== 100) {
        alert(`流量占比总和必须为100%，当前为${totalTraffic}%`);
        return;
    }
    
    // 确保有一个控制组
    const hasControl = variants.some(v => v.is_control);
    if (!hasControl && variants.length > 0) {
        variants[0].is_control = true;
    }
    
    let testConfig = {};
    if (configText) {
        try {
            testConfig = JSON.parse(configText);
        } catch (e) {
            alert('测试配置JSON格式错误');
            return;
        }
    }
    
    try {
        const data = await auth.request('/api/v1/admin/ab-testing', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                name,
                description,
                application_id: parseInt(applicationId),
                variants,
                config: testConfig
            })
        });
        
        if (data.code === 0) {
            const modal = bootstrap.Modal.getInstance(document.getElementById('createTestModal'));
            modal.hide();
            await loadTests();
            await loadSummary();
            alert('测试创建成功！');
        } else {
            alert(data.message || '创建失败');
        }
    } catch (error) {
        console.error('Failed to save test:', error);
        alert('创建失败，请重试');
    }
}

async function viewTest(testId) {
    currentTestId = testId;
    
    try {
        const [testData, reportData] = await Promise.all([
            auth.request(`/api/v1/admin/ab-testing/${testId}`),
            auth.request(`/api/v1/admin/ab-testing/${testId}/report`)
        ]);
        
        if (testData.code === 0 && testData.data) {
            renderTestDetail(testData.data, reportData.data);
        } else {
            const mockTest = getMockTests().data.find(t => t.id === testId);
            renderTestDetail(mockTest, getMockReport(testId));
        }
    } catch (error) {
        const mockTest = getMockTests().data.find(t => t.id === testId);
        renderTestDetail(mockTest, getMockReport(testId));
    }
    
    const modal = new bootstrap.Modal(document.getElementById('viewTestModal'));
    modal.show();
}

function renderTestDetail(test, report) {
    document.getElementById('viewTestTitle').textContent = test.name;
    
    const startBtn = document.getElementById('startTestBtn');
    const stopBtn = document.getElementById('stopTestBtn');
    
    startBtn.classList.toggle('d-none', test.status !== 'draft');
    stopBtn.classList.toggle('d-none', test.status !== 'running');
    
    startBtn.onclick = () => startTest(test.id);
    stopBtn.onclick = () => stopTest(test.id);
    
    const content = document.getElementById('viewTestContent');
    const app = applications.find(a => a.id === test.application_id);
    
    let variantsHtml = '';
    if (report && report.variants) {
        variantsHtml = report.variants.map((v, i) => `
            <div class="border rounded p-3 mb-3">
                <div class="d-flex justify-content-between align-items-center mb-2">
                    <div class="d-flex align-items-center gap-2">
                        <h6 class="mb-0">
                            <span style="color: ${variantColors[i]};">●</span>
                            ${escapeHtml(v.variant_name)}
                            ${v.is_control ? '<span class="badge bg-secondary ms-1">控制组</span>' : ''}
                            ${report.winning_variant === v.variant_id ? '<span class="winner-badge ms-1"><i class="fas fa-trophy me-1"></i>优胜</span>' : ''}
                        </h6>
                    </div>
                    <div class="text-end">
                        <div class="fw-bold">${v.traffic_percent || test.variants?.find(tv => tv.id === v.variant_id)?.traffic_percent || 0}% 流量</div>
                    </div>
                </div>
                <div class="row g-3 text-center">
                    <div class="col-md-3">
                        <div class="text-muted small">访客数</div>
                        <div class="fw-bold fs-5">${formatLargeNumber(v.visitors || 0)}</div>
                    </div>
                    <div class="col-md-3">
                        <div class="text-muted small">转化数</div>
                        <div class="fw-bold fs-5">${formatLargeNumber(v.conversions || 0)}</div>
                    </div>
                    <div class="col-md-3">
                        <div class="text-muted small">转化率</div>
                        <div class="fw-bold fs-5 text-primary">${(v.conversion_rate || 0).toFixed(2)}%</div>
                    </div>
                    <div class="col-md-3">
                        <div class="text-muted small">提升/置信度</div>
                        <div class="fw-bold fs-5 ${v.improvement >= 0 ? 'text-success' : 'text-danger'}">
                            ${v.improvement >= 0 ? '+' : ''}${(v.improvement || 0).toFixed(2)}%
                            ${v.confidence ? `<span class="small text-muted">/${(v.confidence).toFixed(1)}%</span>` : ''}
                        </div>
                    </div>
                </div>
            </div>
        `).join('');
    } else if (test.variants) {
        variantsHtml = test.variants.map((v, i) => `
            <div class="border rounded p-3 mb-3">
                <div class="d-flex justify-content-between align-items-center">
                    <div class="d-flex align-items-center gap-2">
                        <h6 class="mb-0">
                            <span style="color: ${variantColors[i]};">●</span>
                            ${escapeHtml(v.name)}
                            ${v.is_control ? '<span class="badge bg-secondary ms-1">控制组</span>' : ''}
                        </h6>
                    </div>
                    <div class="text-end">
                        <div class="fw-bold">${v.traffic_percent}% 流量</div>
                    </div>
                </div>
                <div class="text-muted small mt-2">${escapeHtml(v.description || '无描述')}</div>
            </div>
        `).join('');
    }
    
    content.innerHTML = `
        <div class="mb-4">
            <h6>基本信息</h6>
            <div class="row g-3">
                <div class="col-md-4">
                    <div class="text-muted small">状态</div>
                    <div class="fw-bold">${{draft:'草稿',running:'运行中',stopped:'已停止'}[test.status] || test.status}</div>
                </div>
                <div class="col-md-4">
                    <div class="text-muted small">关联应用</div>
                    <div class="fw-bold">${app ? escapeHtml(app.name) : '未知'}</div>
                </div>
                <div class="col-md-4">
                    <div class="text-muted small">创建时间</div>
                    <div class="fw-bold">${formatDate(test.created_at)}</div>
                </div>
            </div>
            ${test.description ? `<div class="mt-3"><div class="text-muted small">描述</div><div>${escapeHtml(test.description)}</div></div>` : ''}
        </div>
        <div>
            <h6>变体配置</h6>
            ${variantsHtml}
        </div>
        ${report && report.recommendations ? `
            <div class="mt-4">
                <h6><i class="fas fa-lightbulb text-warning me-2"></i>智能推荐</h6>
                <div class="mt-2">
                    ${report.recommendations.map(rec => `
                        <div class="alert alert-info py-2 mb-2">
                            <i class="fas fa-info-circle me-2"></i>${escapeHtml(rec)}
                        </div>
                    `).join('')}
                </div>
            </div>
        ` : ''}
    `;
}

async function startTest(testId) {
    if (!confirm('确定要启动这个测试吗？')) return;
    
    try {
        const data = await auth.request(`/api/v1/admin/ab-testing/${testId}/start`, {
            method: 'POST'
        });
        
        if (data.code === 0) {
            const modal = bootstrap.Modal.getInstance(document.getElementById('viewTestModal'));
            modal.hide();
            await loadTests();
            await loadSummary();
            alert('测试已启动！');
        } else {
            alert(data.message || '启动失败');
        }
    } catch (error) {
        alert('启动失败，请重试');
    }
}

async function stopTest(testId) {
    if (!confirm('确定要停止这个测试吗？')) return;
    
    try {
        const data = await auth.request(`/api/v1/admin/ab-testing/${testId}/stop`, {
            method: 'POST'
        });
        
        if (data.code === 0) {
            const modal = bootstrap.Modal.getInstance(document.getElementById('viewTestModal'));
            modal.hide();
            await loadTests();
            await loadSummary();
            alert('测试已停止！');
        } else {
            alert(data.message || '停止失败');
        }
    } catch (error) {
        alert('停止失败，请重试');
    }
}

function populateTestSelect() {
    const select = document.getElementById('analyticsTestSelect');
    const tests = getMockTests().data;
    
    select.innerHTML = '<option value="">选择一个测试查看分析</option>' + 
        tests.map(t => `<option value="${t.id}">${escapeHtml(t.name)}</option>`).join('');
}

async function loadTestAnalytics() {
    const testId = document.getElementById('analyticsTestSelect').value;
    if (!testId) {
        document.getElementById('analyticsContent').classList.add('d-none');
        return;
    }
    
    try {
        const data = await auth.request(`/api/v1/admin/ab-testing/${testId}/report`);
        if (data.code === 0 && data.data) {
            renderAnalytics(data.data);
        } else {
            renderAnalytics(getMockReport(parseInt(testId)));
        }
        
        await Promise.all([
            loadComparisonAnalysis(testId),
            loadTestRecommendations(testId)
        ]);
    } catch (error) {
        renderAnalytics(getMockReport(parseInt(testId)));
    }
}

function getMockReport(testId) {
    const tests = getMockTests().data;
    const test = tests.find(t => t.id === testId);
    if (!test) return null;
    
    return {
        test_id: test.id,
        test_name: test.name,
        status: test.status,
        start_date: test.start_date,
        end_date: test.end_date,
        total_visitors: Math.floor(10000 + Math.random() * 50000),
        winning_variant: test.status === 'stopped' ? test.variants[1]?.id : null,
        variants: test.variants.map((v, i) => {
            const baseVisitors = Math.floor(5000 + Math.random() * 20000);
            const visitors = Math.floor(baseVisitors * (v.traffic_percent / 50));
            const baseRate = 8 + Math.random() * 5;
            const conversionRate = i === 0 ? baseRate : baseRate + (Math.random() * 6 - 2);
            const conversions = Math.floor(visitors * conversionRate / 100);
            
            return {
                variant_id: v.id,
                variant_name: v.name,
                is_control: v.is_control,
                visitors,
                conversions,
                conversion_rate: conversionRate,
                improvement: i === 0 ? 0 : (conversionRate - baseRate) / baseRate * 100,
                confidence: 70 + Math.random() * 25
            };
        }),
        recommendations: [
            '测试正在进行中，继续收集数据以获得更可靠的结果',
            '建议每个变体至少收集1000个访客后再做判断',
            '如果置信度达到95%以上，可以考虑结束测试'
        ]
    };
}

function renderAnalytics(report) {
    if (!report) return;
    
    document.getElementById('analyticsContent').classList.remove('d-none');
    
    // 测试概览
    const overview = document.getElementById('testOverview');
    overview.innerHTML = `
        <div class="col-md-3">
            <div class="text-muted small">总访客</div>
            <div class="fw-bold fs-4">${formatLargeNumber(report.total_visitors)}</div>
        </div>
        <div class="col-md-3">
            <div class="text-muted small">状态</div>
            <div class="fw-bold fs-4">${{draft:'草稿',running:'运行中',stopped:'已停止'}[report.status] || report.status}</div>
        </div>
        <div class="col-md-3">
            <div class="text-muted small">开始时间</div>
            <div class="fw-bold fs-4">${report.start_date ? formatDate(report.start_date) : '-'}</div>
        </div>
        <div class="col-md-3">
            <div class="text-muted small">结束时间</div>
            <div class="fw-bold fs-4">${report.end_date ? formatDate(report.end_date) : '-'}</div>
        </div>
    `;
    
    // 变体对比
    const comparison = document.getElementById('variantsComparison');
    comparison.innerHTML = `
        <div class="table-responsive">
            <table class="table table-hover">
                <thead>
                    <tr>
                        <th>变体</th>
                        <th class="text-center">访客</th>
                        <th class="text-center">转化</th>
                        <th class="text-center">转化率</th>
                        <th class="text-center">提升</th>
                        <th class="text-center">置信度</th>
                    </tr>
                </thead>
                <tbody>
                    ${report.variants.map((v, i) => `
                        <tr>
                            <td>
                                <span style="color: ${variantColors[i]};">●</span>
                                <strong>${escapeHtml(v.variant_name)}</strong>
                                ${v.is_control ? '<span class="badge bg-secondary ms-1">控制组</span>' : ''}
                                ${report.winning_variant === v.variant_id ? '<span class="winner-badge ms-1"><i class="fas fa-trophy"></i></span>' : ''}
                            </td>
                            <td class="text-center">${formatLargeNumber(v.visitors)}</td>
                            <td class="text-center">${formatLargeNumber(v.conversions)}</td>
                            <td class="text-center"><strong>${v.conversion_rate.toFixed(2)}%</strong></td>
                            <td class="text-center">
                                <span class="${v.improvement >= 0 ? 'text-success' : 'text-danger'}">
                                    ${v.improvement >= 0 ? '+' : ''}${v.improvement.toFixed(2)}%
                                </span>
                            </td>
                            <td class="text-center">
                                <span class="badge ${v.confidence >= 95 ? 'bg-success' : v.confidence >= 90 ? 'bg-warning' : 'bg-secondary'}">
                                    ${v.confidence.toFixed(1)}%
                                </span>
                            </td>
                        </tr>
                    `).join('')}
                </tbody>
            </table>
        </div>
    `;
    
    // 渲染图表
    renderConversionRateChart(report);
    renderTrafficDistributionChart(report);
    
    // 推荐
    const recommendations = document.getElementById('recommendationsList');
    if (report.recommendations && report.recommendations.length > 0) {
        recommendations.innerHTML = report.recommendations.map(rec => `
            <div class="alert alert-info py-2 mb-2">
                <i class="fas fa-info-circle me-2"></i>${escapeHtml(rec)}
            </div>
        `).join('');
    } else {
        recommendations.innerHTML = '<p class="text-muted mb-0">暂无推荐</p>';
    }
}

function renderConversionRateChart(report) {
    const ctx = document.getElementById('conversionRateChart');
    if (!ctx) return;
    
    if (charts.conversionRate) {
        charts.conversionRate.destroy();
    }
    
    // 生成模拟日期数据
    const labels = [];
    const now = new Date();
    for (let i = 13; i >= 0; i--) {
        const date = new Date(now);
        date.setDate(date.getDate() - i);
        labels.push(date.toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit' }));
    }
    
    const datasets = report.variants.map((v, i) => ({
        label: v.variant_name,
        data: labels.map(() => {
            const variance = (Math.random() - 0.5) * 2;
            return Math.max(0, v.conversion_rate + variance);
        }),
        borderColor: variantColors[i],
        backgroundColor: variantColors[i] + '20',
        tension: 0.4,
        fill: false
    }));
    
    charts.conversionRate = new Chart(ctx, {
        type: 'line',
        data: { labels, datasets },
        options: getChartOptions('line')
    });
}

function renderTrafficDistributionChart(report) {
    const ctx = document.getElementById('trafficDistributionChart');
    if (!ctx) return;
    
    if (charts.trafficDistribution) {
        charts.trafficDistribution.destroy();
    }
    
    const totalVisitors = report.variants.reduce((sum, v) => sum + v.visitors, 0);
    
    charts.trafficDistribution = new Chart(ctx, {
        type: 'doughnut',
        data: {
            labels: report.variants.map(v => v.variant_name),
            datasets: [{
                data: report.variants.map(v => v.visitors),
                backgroundColor: report.variants.map((v, i) => variantColors[i]),
                borderWidth: 2,
                borderColor: '#fff'
            }]
        },
        options: getChartOptions('doughnut')
    });
}

function getChartOptions(type) {
    const options = {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
            legend: {
                display: type !== 'line',
                position: 'bottom'
            }
        }
    };
    
    if (type === 'line' || type === 'bar') {
        options.scales = {
            y: {
                beginAtZero: true
            }
        };
    }
    
    return options;
}

function formatDate(dateStr) {
    if (!dateStr) return '-';
    const date = new Date(dateStr);
    return date.toLocaleDateString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit'
    });
}

function formatLargeNumber(num) {
    if (num >= 1000000) {
        return (num / 1000000).toFixed(1) + 'M';
    } else if (num >= 1000) {
        return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
}

function escapeHtml(text) {
    if (text === null || text === undefined) return '';
    const div = document.createElement('div');
    div.textContent = String(text);
    return div.innerHTML;
}

async function loadComparisonAnalysis(testId) {
    try {
        const data = await auth.request(`/api/v1/admin/ab-tests/${testId}/compare`);
        if (data.code === 0 && data.data && data.data.length > 0) {
            renderComparisonAnalysis(data.data);
        } else {
            document.getElementById('comparisonAnalysis').innerHTML = `
                <div class="text-center py-3">
                    <p class="text-muted">暂无对比数据</p>
                </div>
            `;
        }
    } catch (error) {
        console.error('Failed to load comparison analysis:', error);
        document.getElementById('comparisonAnalysis').innerHTML = `
            <div class="text-center py-3">
                <p class="text-muted">加载对比分析失败</p>
            </div>
        `;
    }
}

function renderComparisonAnalysis(comparisons) {
    const container = document.getElementById('comparisonAnalysis');
    
    container.innerHTML = comparisons.map(comp => {
        const isPositive = comp.relative_diff > 0;
        const significanceClass = comp.statistical_significance ? 'success' : 'secondary';
        const significanceText = comp.statistical_significance ? '统计显著' : '不显著';
        
        return `
            <div class="card mb-3 border-${significanceClass}">
                <div class="card-body">
                    <div class="d-flex justify-content-between align-items-center mb-3">
                        <h6 class="mb-0">
                            <span class="badge bg-primary me-2">${escapeHtml(comp.variant1.variant_name)}</span>
                            <i class="fas fa-arrows-alt-h mx-2"></i>
                            <span class="badge bg-info me-2">${escapeHtml(comp.variant2.variant_name)}</span>
                        </h6>
                        <span class="badge bg-${significanceClass}">${significanceText}</span>
                    </div>
                    <div class="row g-3 mb-3">
                        <div class="col-md-3">
                            <div class="text-muted small">相对差异</div>
                            <div class="fw-bold fs-5 ${isPositive ? 'text-success' : 'text-danger'}">
                                ${isPositive ? '+' : ''}${comp.relative_diff.toFixed(2)}%
                            </div>
                        </div>
                        <div class="col-md-3">
                            <div class="text-muted small">绝对差异</div>
                            <div class="fw-bold fs-5 ${comp.absolute_diff > 0 ? 'text-success' : 'text-danger'}">
                                ${comp.absolute_diff > 0 ? '+' : ''}${comp.absolute_diff.toFixed(2)}%
                            </div>
                        </div>
                        <div class="col-md-6">
                            <div class="text-muted small">结论</div>
                            <div class="fw-bold">${escapeHtml(comp.conclusion)}</div>
                        </div>
                    </div>
                    <div class="progress" style="height: 8px;">
                        <div class="progress-bar bg-primary" style="width: 50%"></div>
                        <div class="progress-bar ${isPositive ? 'bg-success' : 'bg-danger'}" 
                             style="width: ${Math.min(Math.abs(comp.relative_diff), 50)}%"></div>
                    </div>
                </div>
            </div>
        `;
    }).join('');
}

async function loadTestAnalytics(testId) {
    const testIdNum = parseInt(testId);
    if (!testIdNum) return;
    
    await Promise.all([
        loadTestAnalyticsByPeriod(testIdNum, '7d'),
        loadComparisonAnalysis(testIdNum)
    ]);
}

async function loadTestAnalyticsByPeriod(testId, period) {
    const tests = getMockTests().data;
    const test = tests.find(t => t.id === testId);
    if (!test || !test.variants) return;
    
    const variantAnalyticsPromises = test.variants.map(variant => {
        return loadVariantAnalytics(testId, variant.id, period);
    });
    
    await Promise.all(variantAnalyticsPromises);
}

async function loadVariantAnalytics(testId, variantId, period) {
    try {
        const data = await auth.request(`/api/v1/admin/ab-tests/${testId}/variant/${variantId}/analytics?period=${period}`);
        if (data.code === 0 && data.data) {
            console.log('Variant analytics loaded:', data.data);
        }
    } catch (error) {
        console.error('Failed to load variant analytics:', error);
    }
}

async function loadTestRecommendations(testId) {
    try {
        const data = await auth.request(`/api/v1/admin/ab-tests/${testId}/recommendations`);
        if (data.code === 0 && data.data && data.data.length > 0) {
            renderAdvancedRecommendations(data.data);
        }
    } catch (error) {
        console.error('Failed to load recommendations:', error);
    }
}

function renderAdvancedRecommendations(recommendations) {
    const container = document.getElementById('recommendationsList');
    
    const impactOrder = { high: 1, medium: 2, low: 3 };
    recommendations.sort((a, b) => impactOrder[a.impact] - impactOrder[b.impact]);
    
    container.innerHTML = recommendations.map(rec => {
        const impactColors = {
            high: 'danger',
            medium: 'warning',
            low: 'info'
        };
        
        const typeIcons = {
            data_quality: 'fa-database',
            timing: 'fa-clock',
            statistical: 'fa-chart-line',
            next_steps: 'fa-arrow-right'
        };
        
        return `
            <div class="alert alert-${impactColors[rec.impact] || 'info'} py-2 mb-2">
                <div class="d-flex align-items-center">
                    <i class="fas ${typeIcons[rec.type] || 'fa-info-circle'} me-2"></i>
                    <div class="flex-grow-1">
                        <strong>${escapeHtml(rec.title)}</strong>
                        <div class="small mt-1">${escapeHtml(rec.content)}</div>
                    </div>
                    <span class="badge bg-${impactColors[rec.impact] || 'secondary'} ms-2">
                        ${rec.impact === 'high' ? '高优先级' : rec.impact === 'medium' ? '中优先级' : '低优先级'}
                    </span>
                </div>
            </div>
        `;
    }).join('');
}