let rateLimitData = {
    adaptive: null,
    distributed: null,
    smart: null,
    tokenBucket: null
};

document.addEventListener('DOMContentLoaded', function() {
    loadRateLimitConfig();
    initCharts();
    setupEventListeners();
});

function setupEventListeners() {
    document.getElementById('refreshBtn')?.addEventListener('click', loadRateLimitConfig);
    document.getElementById('adaptiveConfigForm')?.addEventListener('submit', updateAdaptiveConfig);
    document.getElementById('distributedConfigForm')?.addEventListener('submit', updateDistributedConfig);
    document.getElementById('smartConfigForm')?.addEventListener('submit', updateSmartConfig);
    document.getElementById('resetKeyBtn')?.addEventListener('click', resetDistributedKey);
}

function initCharts() {
    const ctx = document.getElementById('rateLimitChart');
    if (ctx) {
        window.rateLimitChart = new Chart(ctx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [
                    {
                        label: '请求数',
                        data: [],
                        borderColor: '#3b82f6',
                        backgroundColor: 'rgba(59, 130, 246, 0.1)',
                        fill: true,
                        tension: 0.4
                    },
                    {
                        label: '拒绝数',
                        data: [],
                        borderColor: '#ef4444',
                        backgroundColor: 'rgba(239, 68, 68, 0.1)',
                        fill: true,
                        tension: 0.4
                    }
                ]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        position: 'top',
                    }
                },
                scales: {
                    y: {
                        beginAtZero: true,
                        grid: {
                            color: 'rgba(0, 0, 0, 0.05)'
                        }
                    },
                    x: {
                        grid: {
                            display: false
                        }
                    }
                }
            }
        });
    }
}

async function loadRateLimitConfig() {
    try {
        showLoading(true);
        
        const [adaptiveRes, distributedRes, smartRes, tokenBucketRes] = await Promise.all([
            fetch('/api/v1/admin/rate-limit/adaptive/stats'),
            fetch('/api/v1/admin/rate-limit/distributed/stats'),
            fetch('/api/v1/admin/rate-limit/smart/stats'),
            fetch('/api/v1/admin/rate-limit/token-bucket/stats')
        ]);

        const [adaptive, distributed, smart, tokenBucket] = await Promise.all([
            adaptiveRes.json(),
            distributedRes.json(),
            smartRes.json(),
            tokenBucketRes.json()
        ]);

        rateLimitData = { adaptive, distributed, smart, tokenBucket };
        
        updateAdaptiveUI(adaptive);
        updateDistributedUI(distributed);
        updateSmartUI(smart);
        updateTokenBucketUI(tokenBucket);
        
        showLoading(false);
    } catch (error) {
        console.error('Failed to load rate limit config:', error);
        showError('加载限流配置失败');
        showLoading(false);
    }
}

function updateAdaptiveUI(data) {
    if (!data.data) return;
    
    const stats = data.data;
    document.getElementById('adaptiveLoadLevel').textContent = stats.load_level || 'normal';
    document.getElementById('adaptiveBucketCount').textContent = stats.bucket_count || 0;
    document.getElementById('adaptiveLoadFactor').textContent = ((stats.load_factor || 0) * 100).toFixed(1) + '%';
    document.getElementById('adaptiveBaseRate').textContent = stats.base_rate || 0;
    document.getElementById('adaptiveBaseCapacity').textContent = stats.base_capacity || 0;
}

function updateDistributedUI(data) {
    if (!data.data) return;
    
    const stats = data.data;
    document.getElementById('distNodeId').textContent = stats.node_id || 'unknown';
    document.getElementById('distMaxRequests').textContent = stats.max_requests || 0;
    document.getElementById('distWindowSecs').textContent = stats.window_secs || 0;
    document.getElementById('distRedisEnabled').textContent = stats.redis_enabled ? '已启用' : '未启用';
    document.getElementById('distLocalCounters').textContent = stats.local_counters || 0;
}

function updateSmartUI(data) {
    if (!data.data) return;
    
    const stats = data.data;
    document.getElementById('smartTotalClients').textContent = stats.total_clients || 0;
    document.getElementById('smartTotalRequests').textContent = stats.total_requests || 0;
    document.getElementById('smartHitRate').textContent = ((stats.hit_rate || 0) * 100).toFixed(2) + '%';
    document.getElementById('smartHotspotCount').textContent = stats.hotspot_count || 0;
    
    const tierDist = stats.tier_distribution || {};
    const tierTable = document.getElementById('tierDistributionTable');
    if (tierTable) {
        tierTable.innerHTML = Object.entries(tierDist).map(([tier, count]) => 
            `<tr><td>${tier}</td><td>${count}</td></tr>`
        ).join('');
    }
}

function updateTokenBucketUI(data) {
    if (!data.data) return;
    
    const stats = data.data.global_stats || {};
    const buckets = data.data.buckets || [];
    
    document.getElementById('tbBucketCount').textContent = stats.bucket_count || 0;
    document.getElementById('tbTotalTokens').textContent = (stats.total_tokens || 0).toFixed(2);
    document.getElementById('tbRedisEnabled').textContent = stats.redis_enabled ? '已启用' : '未启用';
    
    const bucketTable = document.getElementById('bucketListTable');
    if (bucketTable) {
        bucketTable.innerHTML = buckets.slice(0, 20).map(bucket => 
            `<tr>
                <td>${escapeHtml(bucket.key || '')}</td>
                <td>${(bucket.tokens || 0).toFixed(2)}</td>
                <td>${bucket.capacity || 0}</td>
                <td>${((bucket.token_usage || 0)).toFixed(1)}%</td>
                <td>${bucket.allowed_requests || 0}</td>
                <td>${bucket.rejected_requests || 0}</td>
            </tr>`
        ).join('');
    }
}

async function updateAdaptiveConfig(e) {
    e.preventDefault();
    
    const form = e.target;
    const data = {
        base_rate: parseFloat(form.base_rate.value),
        base_capacity: parseFloat(form.base_capacity.value),
        min_capacity: parseFloat(form.min_capacity.value),
        max_capacity: parseFloat(form.max_capacity.value),
        high_load_threshold: parseFloat(form.high_load_threshold.value),
        critical_load_threshold: parseFloat(form.critical_load_threshold.value)
    };
    
    try {
        showLoading(true);
        const res = await fetch('/api/v1/admin/rate-limit/adaptive/config', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });
        
        if (res.ok) {
            showSuccess('自适应限流配置已更新');
            loadRateLimitConfig();
        } else {
            showError('更新配置失败');
        }
    } catch (error) {
        console.error('Failed to update adaptive config:', error);
        showError('更新配置失败');
    } finally {
        showLoading(false);
    }
}

async function updateDistributedConfig(e) {
    e.preventDefault();
    
    const form = e.target;
    const data = {
        type: form.algorithm_type.value,
        max_requests: parseInt(form.max_requests.value),
        window_secs: parseInt(form.window_secs.value),
        consistency_mode: form.consistency_mode.checked
    };
    
    try {
        showLoading(true);
        const res = await fetch('/api/v1/admin/rate-limit/distributed/config', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });
        
        if (res.ok) {
            showSuccess('分布式限流配置已更新');
            loadRateLimitConfig();
        } else {
            showError('更新配置失败');
        }
    } catch (error) {
        console.error('Failed to update distributed config:', error);
        showError('更新配置失败');
    } finally {
        showLoading(false);
    }
}

async function updateSmartConfig(e) {
    e.preventDefault();
    
    const form = e.target;
    const data = {
        default_requests_per_min: parseInt(form.default_requests_per_min.value),
        default_burst_limit: parseInt(form.default_burst_limit.value),
        enable_adaptive_limit: form.enable_adaptive_limit.checked,
        enable_risk_based_limit: form.enable_risk_based_limit.checked,
        enable_hotspot_detection: form.enable_hotspot_detection.checked,
        enable_predictive_limit: form.enable_predictive_limit.checked
    };
    
    try {
        showLoading(true);
        const res = await fetch('/api/v1/admin/rate-limit/smart/config', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });
        
        if (res.ok) {
            showSuccess('智能限流配置已更新');
            loadRateLimitConfig();
        } else {
            showError('更新配置失败');
        }
    } catch (error) {
        console.error('Failed to update smart config:', error);
        showError('更新配置失败');
    } finally {
        showLoading(false);
    }
}

async function resetDistributedKey() {
    const key = document.getElementById('resetKeyInput')?.value;
    if (!key) {
        showError('请输入要重置的键');
        return;
    }
    
    try {
        showLoading(true);
        const res = await fetch(`/api/v1/admin/rate-limit/distributed/reset?key=${encodeURIComponent(key)}`, {
            method: 'POST'
        });
        
        if (res.ok) {
            showSuccess('键已重置');
            loadRateLimitConfig();
        } else {
            showError('重置失败');
        }
    } catch (error) {
        console.error('Failed to reset key:', error);
        showError('重置失败');
    } finally {
        showLoading(false);
    }
}

function showLoading(show) {
    const overlay = document.getElementById('loadingOverlay');
    if (overlay) {
        overlay.style.display = show ? 'flex' : 'none';
    }
}

function showSuccess(message) {
    showToast(message, 'success');
}

function showError(message) {
    showToast(message, 'danger');
}

function showToast(message, type) {
    const container = document.getElementById('toastContainer') || createToastContainer();
    const toast = document.createElement('div');
    toast.className = `alert alert-${type} alert-toast`;
    toast.innerHTML = `<i class="fas fa-${type === 'success' ? 'check' : 'exclamation'}-circle"></i> ${escapeHtml(message)}`;
    container.appendChild(toast);
    setTimeout(() => toast.remove(), 3000);
}

function createToastContainer() {
    const container = document.createElement('div');
    container.id = 'toastContainer';
    container.style.cssText = 'position: fixed; top: 20px; right: 20px; z-index: 9999;';
    document.body.appendChild(container);
    return container;
}

function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}
