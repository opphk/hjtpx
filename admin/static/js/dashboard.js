document.addEventListener('DOMContentLoaded', async () => {
    await loadDashboardStats();
    await loadRecentActivity();
});

async function loadDashboardStats() {
    const mockData = {
        totalUsers: 12456,
        totalApps: 156,
        totalRequests: 8234567,
        totalErrors: 1234
    };

    try {
        const data = await auth.request('/dashboard/stats');
        if (data.code === 0) {
            updateStats(data.data);
        } else {
            updateStats(mockData);
        }
    } catch (error) {
        updateStats(mockData);
    }
}

function updateStats(stats) {
    animateNumber('totalUsers', stats.totalUsers);
    animateNumber('totalApps', stats.totalApps);
    animateNumber('totalRequests', stats.totalRequests);
    animateNumber('totalErrors', stats.totalErrors);
}

function animateNumber(elementId, target) {
    const element = document.getElementById(elementId);
    const duration = 1500;
    const start = 0;
    const startTime = performance.now();

    function update(currentTime) {
        const elapsed = currentTime - startTime;
        const progress = Math.min(elapsed / duration, 1);
        const current = Math.floor(start + (target - start) * easeOutQuart(progress));
        element.textContent = formatNumber(current);

        if (progress < 1) {
            requestAnimationFrame(update);
        }
    }

    requestAnimationFrame(update);
}

function easeOutQuart(x) {
    return 1 - Math.pow(1 - x, 4);
}

function formatNumber(num) {
    if (num >= 1000000) {
        return (num / 1000000).toFixed(1) + 'M';
    } else if (num >= 1000) {
        return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
}

async function loadRecentActivity() {
    const mockActivities = [
        { time: '2024-01-15 14:32:18', event: '用户登录', user: 'admin', status: 'success' },
        { time: '2024-01-15 14:28:45', event: '创建应用', user: 'developer1', status: 'success' },
        { time: '2024-01-15 14:25:12', event: 'API请求失败', user: 'app_001', status: 'error' },
        { time: '2024-01-15 14:20:33', event: '更新配置', user: 'admin', status: 'success' },
        { time: '2024-01-15 14:15:09', event: '用户注册', user: 'new_user', status: 'success' }
    ];

    let activities = mockActivities;

    try {
        const data = await auth.request('/dashboard/activity');
        if (data.code === 0 && data.data) {
            activities = data.data;
        }
    } catch (error) {
    }

    renderActivityTable(activities);
}

function renderActivityTable(activities) {
    const tbody = document.getElementById('recentActivity');
    tbody.innerHTML = activities.map(activity => `
        <tr>
            <td>${activity.time}</td>
            <td>${activity.event}</td>
            <td>${activity.user}</td>
            <td><span class="status ${activity.status}">${getStatusText(activity.status)}</span></td>
        </tr>
    `).join('');
}

function getStatusText(status) {
    const map = {
        success: '成功',
        error: '失败',
        pending: '处理中',
        warning: '警告'
    };
    return map[status] || status;
}
