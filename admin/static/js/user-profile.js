let charts = {};
let currentPage = 1;
let pageSize = 10;
let currentProfile = null;

document.addEventListener('DOMContentLoaded', async function() {
    setupEventListeners();
    await loadProfiles();
    await loadUserOptions();
});

function setupEventListeners() {
    document.getElementById('refreshBtn').addEventListener('click', refreshAll);
    document.getElementById('searchBtn').addEventListener('click', loadProfiles);
    document.getElementById('searchInput').addEventListener('keypress', (e) => {
        if (e.key === 'Enter') loadProfiles();
    });
    document.getElementById('trustLevelFilter').addEventListener('change', loadProfiles);
    document.getElementById('riskLevelFilter').addEventListener('change', loadProfiles);
    document.getElementById('compareBtn').addEventListener('click', compareUsers);
}

async function refreshAll() {
    await Promise.all([
        loadProfiles(),
        loadUserOptions()
    ]);
}

async function loadProfiles() {
    try {
        const trustLevel = document.getElementById('trustLevelFilter').value;
        const riskLevel = document.getElementById('riskLevelFilter').value;
        
        let url = `/api/v1/admin/user-profiles?page=${currentPage}&page_size=${pageSize}`;
        if (trustLevel) url += `&trust_level=${trustLevel}`;
        if (riskLevel) url += `&risk_level=${riskLevel}`;
        
        const data = await auth.request(url);
        if (data.code === 0 && data.data) {
            renderProfiles(data.data);
            renderSummary(data.data);
        } else {
            renderProfiles([]);
            renderSummary([]);
        }
    } catch (error) {
        console.error('Failed to load profiles:', error);
        renderProfiles([]);
        renderSummary([]);
    }
}

function renderSummary(profiles) {
    const container = document.getElementById('summaryMetrics');
    
    const total = profiles.length || 0;
    const highTrust = profiles.filter(p => p.trust_level === '非常高' || p.trust_level === '高').length || 0;
    const mediumRisk = profiles.filter(p => p.risk_level === '中' || p.risk_level === '高').length || 0;
    const lowRisk = profiles.filter(p => p.risk_level === '低').length || 0;
    
    container.innerHTML = `
        <div class="col-md-3">
            <div class="card metric-card primary">
                <div class="card-body">
                    <div class="text-muted small">总用户数</div>
                    <div class="fs-3 fw-bold">${total}</div>
                </div>
            </div>
        </div>
        <div class="col-md-3">
            <div class="card metric-card success">
                <div class="card-body">
                    <div class="text-muted small">高可信用户</div>
                    <div class="fs-3 fw-bold">${highTrust}</div>
                </div>
            </div>
        </div>
        <div class="col-md-3">
            <div class="card metric-card warning">
                <div class="card-body">
                    <div class="text-muted small">中等风险</div>
                    <div class="fs-3 fw-bold">${mediumRisk}</div>
                </div>
            </div>
        </div>
        <div class="col-md-3">
            <div class="card metric-card danger">
                <div class="card-body">
                    <div class="text-muted small">低风险</div>
                    <div class="fs-3 fw-bold">${lowRisk}</div>
                </div>
            </div>
        </div>
    `;
}

function renderProfiles(profiles) {
    const container = document.getElementById('profileList');
    
    if (profiles.length === 0) {
        container.innerHTML = `
            <div class="text-center py-5">
                <i class="fas fa-user-circle text-muted" style="font-size: 3rem;"></i>
                <p class="text-muted mt-3">暂无用户数据</p>
            </div>
        `;
        return;
    }
    
    container.innerHTML = profiles.map(profile => {
        const riskClass = getRiskClass(profile.risk_level);
        const trustClass = getTrustClass(profile.trust_level);
        
        return `
            <div class="profile-card p-4 mb-3" onclick="viewProfile(${profile.user_id})">
                <div class="row g-3">
                    <div class="col-md-6">
                        <div class="d-flex align-items-center gap-3">
                            <div class="bg-primary text-white rounded-circle d-flex align-items-center justify-content-center" style="width:50px;height:50px;font-weight:700;font-size:1.5rem;">
                                ${profile.username ? profile.username.charAt(0).toUpperCase() : 'U'}
                            </div>
                            <div>
                                <h5 class="mb-1">${escapeHtml(profile.username || 'Unknown')}</h5>
                                <div class="text-muted small">用户ID: ${profile.user_id}</div>
                            </div>
                        </div>
                    </div>
                    <div class="col-md-3">
                        <div class="mb-2">
                            <span class="text-muted small">信任等级</span>
                            <div>
                                <span class="trust-badge ${trustClass}">${profile.trust_level || '未知'}</span>
                            </div>
                        </div>
                    </div>
                    <div class="col-md-3">
                        <div class="mb-2">
                            <span class="text-muted small">风险等级</span>
                            <div>
                                <span class="risk-badge ${riskClass}">${profile.risk_level || '未知'}</span>
                            </div>
                        </div>
                    </div>
                </div>
                <div class="row g-3 mt-2">
                    <div class="col-md-3">
                        <div class="text-muted small">成功率</div>
                        <div class="fw-bold">${(profile.success_rate || 0).toFixed(1)}%</div>
                    </div>
                    <div class="col-md-3">
                        <div class="text-muted small">会话数</div>
                        <div class="fw-bold">${profile.total_sessions || 0}</div>
                    </div>
                    <div class="col-md-3">
                        <div class="text-muted small">设备数</div>
                        <div class="fw-bold">${profile.device_count || 0}</div>
                    </div>
                    <div class="col-md-3">
                        <div class="text-muted small">标签</div>
                        <div>
                            ${(profile.tags || []).slice(0, 2).map(tag => 
                                `<span class="badge bg-secondary me-1">${escapeHtml(tag)}</span>`
                            ).join('')}
                        </div>
                    </div>
                </div>
            </div>
        `;
    }).join('');
}

function getRiskClass(level) {
    const classes = {
        '极高': 'risk-very-high',
        '高': 'risk-high',
        '中': 'risk-medium',
        '低': 'risk-low'
    };
    return classes[level] || 'risk-medium';
}

function getTrustClass(level) {
    const classes = {
        '非常高': 'trust-very-high',
        '高': 'trust-high',
        '中': 'trust-medium',
        '低': 'trust-low',
        '非常低': 'trust-very-low'
    };
    return classes[level] || 'trust-medium';
}

async function viewProfile(userId) {
    try {
        const data = await auth.request(`/api/v1/admin/user-profiles/${userId}`);
        if (data.code === 0 && data.data) {
            currentProfile = data.data;
            renderProfileDetail(data.data);
            
            document.getElementById('detail-tab').click();
        } else {
            alert('加载用户画像失败');
        }
    } catch (error) {
        console.error('Failed to load profile:', error);
        alert('加载用户画像失败');
    }
}

function renderProfileDetail(profile) {
    renderBasicInfo(profile);
    renderRiskProfile(profile);
    renderActivityMetrics(profile);
    renderDeviceProfile(profile);
    renderActivityTimeline(profile);
}

function renderBasicInfo(profile) {
    const container = document.getElementById('basicInfo');
    container.innerHTML = `
        <div class="mb-3">
            <div class="text-muted small">用户名</div>
            <div class="fw-bold">${escapeHtml(profile.username || 'Unknown')}</div>
        </div>
        <div class="mb-3">
            <div class="text-muted small">邮箱</div>
            <div class="fw-bold">${escapeHtml(profile.email || 'N/A')}</div>
        </div>
        <div class="mb-3">
            <div class="text-muted small">注册时间</div>
            <div class="fw-bold">${formatDate(profile.created_at)}</div>
        </div>
        <div class="mb-3">
            <div class="text-muted small">信任等级</div>
            <div>
                <span class="trust-badge ${getTrustClass(profile.profile_data?.trust_level)}">
                    ${profile.profile_data?.trust_level || '未知'}
                </span>
            </div>
        </div>
        <div>
            <div class="text-muted small">标签</div>
            <div>
                ${(profile.profile_data?.tags || []).map(tag => 
                    `<span class="badge bg-primary me-1 mb-1">${escapeHtml(tag)}</span>`
                ).join('')}
            </div>
        </div>
    `;
}

function renderRiskProfile(profile) {
    const container = document.getElementById('riskProfile');
    const risk = profile.risk_profile || {};
    
    container.innerHTML = `
        <div class="mb-3">
            <div class="text-muted small">风险评分</div>
            <div class="fs-3 fw-bold">${(risk.risk_score || 0).toFixed(1)}</div>
            <div class="progress mt-2" style="height: 8px;">
                <div class="progress-bar bg-danger" style="width: ${risk.risk_score || 0}%"></div>
            </div>
        </div>
        <div class="mb-3">
            <div class="text-muted small">风险等级</div>
            <div>
                <span class="risk-badge ${getRiskClass(risk.risk_level)}">${risk.risk_level || '未知'}</span>
            </div>
        </div>
        ${(risk.risk_factors || []).length > 0 ? `
            <div class="mb-3">
                <div class="text-muted small">风险因素</div>
                <ul class="list-unstyled">
                    ${risk.risk_factors.map(factor => `
                        <li class="mb-2">
                            <i class="fas fa-exclamation-triangle text-warning me-2"></i>
                            ${escapeHtml(factor.factor)}
                            <span class="badge bg-secondary ms-2">${factor.severity}</span>
                        </li>
                    `).join('')}
                </ul>
            </div>
        ` : ''}
        ${(risk.recommendations || []).length > 0 ? `
            <div>
                <div class="text-muted small">建议</div>
                <ul class="list-unstyled">
                    ${risk.recommendations.map(rec => `
                        <li class="mb-1 text-success">
                            <i class="fas fa-check-circle me-2"></i>
                            ${escapeHtml(rec)}
                        </li>
                    `).join('')}
                </ul>
            </div>
        ` : ''}
    `;
}

function renderActivityMetrics(profile) {
    const container = document.getElementById('activityMetrics');
    const data = profile.profile_data || {};
    
    container.innerHTML = `
        <div class="row g-3">
            <div class="col-md-3">
                <div class="text-muted small">总会话数</div>
                <div class="fs-4 fw-bold">${data.total_sessions || 0}</div>
            </div>
            <div class="col-md-3">
                <div class="text-muted small">验证码总数</div>
                <div class="fs-4 fw-bold">${data.total_captchas || 0}</div>
            </div>
            <div class="col-md-3">
                <div class="text-muted small">成功率</div>
                <div class="fs-4 fw-bold text-success">${(data.success_rate || 0).toFixed(1)}%</div>
            </div>
            <div class="col-md-3">
                <div class="text-muted small">平均解题时间</div>
                <div class="fs-4 fw-bold">${(data.avg_solve_time || 0).toFixed(0)}ms</div>
            </div>
        </div>
        <hr>
        <div class="row g-3">
            <div class="col-md-4">
                <div class="text-muted small">首次活动</div>
                <div class="fw-bold">${data.first_activity ? formatDate(data.first_activity) : 'N/A'}</div>
            </div>
            <div class="col-md-4">
                <div class="text-muted small">最近活动</div>
                <div class="fw-bold">${data.last_activity ? formatDate(data.last_activity) : 'N/A'}</div>
            </div>
            <div class="col-md-4">
                <div class="text-muted small">活跃天数</div>
                <div class="fw-bold">${data.activity_days || 0}</div>
            </div>
        </div>
    `;
}

function renderDeviceProfile(profile) {
    const container = document.getElementById('deviceProfile');
    const device = profile.device_profile || {};
    const devices = device.all_devices || [];
    
    if (devices.length === 0) {
        container.innerHTML = '<p class="text-muted mb-0">暂无设备数据</p>';
        return;
    }
    
    container.innerHTML = devices.map(dev => {
        const icon = getDeviceIcon(dev.device_type);
        return `
            <div class="d-flex align-items-center gap-3 mb-3 p-3 border rounded">
                <div class="device-icon bg-primary text-white">
                    <i class="fas fa-${icon}"></i>
                </div>
                <div class="flex-grow-1">
                    <div class="fw-bold">${escapeHtml(dev.device_type || 'Unknown')}</div>
                    <div class="text-muted small">
                        ${escapeHtml(dev.os || 'Unknown')} · ${escapeHtml(dev.browser || 'Unknown')}
                    </div>
                    <div class="text-muted small">
                        使用 ${dev.usage_count} 次 · 
                        ${dev.is_trusted ? '<span class="text-success">已信任</span>' : '<span class="text-warning">未信任</span>'}
                    </div>
                </div>
            </div>
        `;
    }).join('');
}

function getDeviceIcon(type) {
    const icons = {
        'Desktop': 'desktop',
        'Mobile': 'mobile-alt',
        'Tablet': 'tablet-alt'
    };
    return icons[type] || 'question';
}

function renderActivityTimeline(profile) {
    const container = document.getElementById('activityTimeline');
    const timeline = profile.activity_timeline || [];
    
    if (timeline.length === 0) {
        container.innerHTML = '<p class="text-muted mb-0">暂无活动时间线</p>';
        return;
    }
    
    container.innerHTML = timeline.slice(0, 20).map(item => `
        <div class="timeline-item ${item.risk_level}">
            <div class="fw-bold">${escapeHtml(item.event_type)}</div>
            <div class="text-muted small">${formatDateTime(item.timestamp)}</div>
            <div>${escapeHtml(item.description)}</div>
        </div>
    `).join('');
}

async function loadUserOptions() {
    try {
        const data = await auth.request('/api/v1/admin/user-profiles?page=1&page_size=100');
        if (data.code === 0 && data.data) {
            const users = data.data;
            
            const select1 = document.getElementById('compareUser1');
            const select2 = document.getElementById('compareUser2');
            
            const options = users.map(user => 
                `<option value="${user.user_id}">${escapeHtml(user.username)} (ID: ${user.user_id})</option>`
            ).join('');
            
            select1.innerHTML = '<option value="">选择第一个用户</option>' + options;
            select2.innerHTML = '<option value="">选择第二个用户</option>' + options;
        }
    } catch (error) {
        console.error('Failed to load user options:', error);
    }
}

async function compareUsers() {
    const user1Id = document.getElementById('compareUser1').value;
    const user2Id = document.getElementById('compareUser2').value;
    
    if (!user1Id || !user2Id) {
        alert('请选择两个用户进行对比');
        return;
    }
    
    if (user1Id === user2Id) {
        alert('请选择不同的用户进行对比');
        return;
    }
    
    try {
        const data = await auth.request(`/api/v1/admin/user-profiles/compare/${user1Id}/${user2Id}`);
        if (data.code === 0 && data.data) {
            renderComparisonResult(data.data);
        } else {
            alert('加载对比数据失败');
        }
    } catch (error) {
        console.error('Failed to compare users:', error);
        alert('加载对比数据失败');
    }
}

function renderComparisonResult(comparison) {
    const container = document.getElementById('comparisonResult');
    
    container.innerHTML = `
        <div class="row g-4">
            <div class="col-md-6">
                <div class="card shadow-sm h-100">
                    <div class="card-header bg-primary text-white">
                        <h5 class="mb-0">用户1: ${escapeHtml(comparison.user1.username)}</h5>
                    </div>
                    <div class="card-body">
                        <div class="mb-2">
                            <span class="text-muted">信任等级:</span>
                            <span class="trust-badge ${getTrustClass(comparison.user1.trust_level)} ms-2">${comparison.user1.trust_level}</span>
                        </div>
                        <div class="mb-2">
                            <span class="text-muted">风险等级:</span>
                            <span class="risk-badge ${getRiskClass(comparison.user1.risk_level)} ms-2">${comparison.user1.risk_level}</span>
                        </div>
                        <div class="mb-2">
                            <span class="text-muted">成功率:</span>
                            <span class="fw-bold ms-2">${(comparison.user1.success_rate || 0).toFixed(1)}%</span>
                        </div>
                        <div>
                            <span class="text-muted">会话数:</span>
                            <span class="fw-bold ms-2">${comparison.user1.total_sessions || 0}</span>
                        </div>
                    </div>
                </div>
            </div>
            <div class="col-md-6">
                <div class="card shadow-sm h-100">
                    <div class="card-header bg-info text-white">
                        <h5 class="mb-0">用户2: ${escapeHtml(comparison.user2.username)}</h5>
                    </div>
                    <div class="card-body">
                        <div class="mb-2">
                            <span class="text-muted">信任等级:</span>
                            <span class="trust-badge ${getTrustClass(comparison.user2.trust_level)} ms-2">${comparison.user2.trust_level}</span>
                        </div>
                        <div class="mb-2">
                            <span class="text-muted">风险等级:</span>
                            <span class="risk-badge ${getRiskClass(comparison.user2.risk_level)} ms-2">${comparison.user2.risk_level}</span>
                        </div>
                        <div class="mb-2">
                            <span class="text-muted">成功率:</span>
                            <span class="fw-bold ms-2">${(comparison.user2.success_rate || 0).toFixed(1)}%</span>
                        </div>
                        <div>
                            <span class="text-muted">会话数:</span>
                            <span class="fw-bold ms-2">${comparison.user2.total_sessions || 0}</span>
                        </div>
                    </div>
                </div>
            </div>
        </div>
        
        <div class="row g-4 mt-3">
            <div class="col-md-4">
                <div class="card shadow-sm">
                    <div class="card-header bg-success text-white">
                        <h5 class="mb-0"><i class="fas fa-check-circle me-2"></i>相似之处</h5>
                    </div>
                    <div class="card-body">
                        ${(comparison.similarities || []).length > 0 ? 
                            comparison.similarities.map(sim => `
                                <div class="mb-2">
                                    <i class="fas fa-check text-success me-2"></i>
                                    ${escapeHtml(sim)}
                                </div>
                            `).join('') : 
                            '<p class="text-muted mb-0">暂无相似之处</p>'
                        }
                    </div>
                </div>
            </div>
            <div class="col-md-4">
                <div class="card shadow-sm">
                    <div class="card-header bg-warning text-dark">
                        <h5 class="mb-0"><i class="fas fa-exclamation-circle me-2"></i>差异之处</h5>
                    </div>
                    <div class="card-body">
                        ${(comparison.differences || []).length > 0 ? 
                            comparison.differences.map(diff => `
                                <div class="mb-2">
                                    <i class="fas fa-arrows-alt-h text-warning me-2"></i>
                                    ${escapeHtml(diff)}
                                </div>
                            `).join('') : 
                            '<p class="text-muted mb-0">暂无明显差异</p>'
                        }
                    </div>
                </div>
            </div>
            <div class="col-md-4">
                <div class="card shadow-sm">
                    <div class="card-header bg-primary text-white">
                        <h5 class="mb-0"><i class="fas fa-lightbulb me-2"></i>建议</h5>
                    </div>
                    <div class="card-body">
                        ${(comparison.recommendations || []).length > 0 ? 
                            comparison.recommendations.map(rec => `
                                <div class="mb-2">
                                    <i class="fas fa-info-circle text-primary me-2"></i>
                                    ${escapeHtml(rec)}
                                </div>
                            `).join('') : 
                            '<p class="text-muted mb-0">暂无建议</p>'
                        }
                    </div>
                </div>
            </div>
        </div>
    `;
}

function formatDate(dateStr) {
    if (!dateStr) return 'N/A';
    const date = new Date(dateStr);
    return date.toLocaleDateString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit'
    });
}

function formatDateTime(dateStr) {
    if (!dateStr) return 'N/A';
    const date = new Date(dateStr);
    return date.toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
    });
}

function escapeHtml(text) {
    if (text === null || text === undefined) return '';
    const div = document.createElement('div');
    div.textContent = String(text);
    return div.innerHTML;
}
