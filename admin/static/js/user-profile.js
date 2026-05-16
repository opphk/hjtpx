let behaviorRadarChart, deviceConsistencyChart, ipHistoryChart;

document.addEventListener('DOMContentLoaded', () => {
    initProfileCharts();
    setupEventListeners();
});

function initProfileCharts() {
    initDeviceConsistencyChart();
    initIpHistoryChart();
}

function initDeviceConsistencyChart() {
    const ctx = document.getElementById('deviceConsistencyChart');
    if (!ctx) return;

    deviceConsistencyChart = new Chart(ctx, {
        type: 'doughnut',
        data: {
            labels: ['正常设备', '可疑设备', '新设备'],
            datasets: [{
                data: [85, 10, 5],
                backgroundColor: ['#10b981', '#f59e0b', '#3b82f6'],
                borderWidth: 0
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            cutout: '70%',
            plugins: {
                legend: {
                    position: 'bottom',
                    labels: { padding: 15, usePointStyle: true }
                }
            }
        }
    });
}

function initIpHistoryChart() {
    const ctx = document.getElementById('ipHistoryChart');
    if (!ctx) return;

    ipHistoryChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: ['第1周', '第2周', '第3周', '第4周'],
            datasets: [{
                label: 'IP地址数',
                data: [3, 4, 5, 5],
                borderColor: '#8b5cf6',
                backgroundColor: 'rgba(139, 92, 246, 0.1)',
                fill: true,
                tension: 0.4
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: { display: false }
            },
            scales: {
                y: {
                    beginAtZero: true,
                    ticks: { stepSize: 1 }
                }
            }
        }
    });
}

function setupEventListeners() {
    const searchBtn = document.getElementById('searchProfileBtn');
    if (searchBtn) {
        searchBtn.addEventListener('click', searchUserProfile);
    }

    const searchInput = document.getElementById('searchInput');
    if (searchInput) {
        searchInput.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                searchUserProfile();
            }
        });
    }

    const exportBtn = document.getElementById('exportProfileBtn');
    if (exportBtn) {
        exportBtn.addEventListener('click', exportUserProfileReport);
    }
}

function searchUserProfile() {
    const searchValue = document.getElementById('searchInput')?.value;
    if (!searchValue) {
        alert('请输入用户ID或IP地址');
        return;
    }
    
    console.log('搜索用户画像:', searchValue);
    loadUserProfileData(searchValue);
}

function loadUserProfileData(userId) {
    console.log('加载用户画像数据:', userId);
    
    const mockData = {
        userId: userId,
        riskLevel: Math.random() > 0.5 ? 'low' : 'medium',
        verifyCount: Math.floor(Math.random() * 5000) + 500,
        successRate: (Math.random() * 10 + 90).toFixed(1),
        lastActive: '刚刚'
    };

    const userIdEl = document.getElementById('userId');
    const riskBadgeEl = document.getElementById('userRiskBadge');
    const verifyCountEl = document.getElementById('userVerifyCount');
    const successRateEl = document.getElementById('userSuccessRate');
    const lastActiveEl = document.getElementById('userLastActive');

    if (userIdEl) userIdEl.textContent = mockData.userId;
    if (verifyCountEl) verifyCountEl.textContent = formatNumber(mockData.verifyCount);
    if (successRateEl) successRateEl.textContent = mockData.successRate + '%';
    if (lastActiveEl) lastActiveEl.textContent = mockData.lastActive;
    
    if (riskBadgeEl) {
        const levelMap = {
            low: { text: '低风险', class: 'bg-success' },
            medium: { text: '中风险', class: 'bg-warning' },
            high: { text: '高风险', class: 'bg-danger' },
            critical: { text: '极高风险', class: 'bg-danger' }
        };
        const level = levelMap[mockData.riskLevel];
        riskBadgeEl.textContent = level.text;
        riskBadgeEl.className = `badge ${level.class}`;
    }
}

function formatNumber(num) {
    if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
    if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
    return num.toString();
}

function exportUserProfileReport() {
    const userId = document.getElementById('userId')?.textContent || 'unknown';
    
    let csvContent = '\ufeff';
    csvContent += '用户画像报告\n\n';
    csvContent += `用户ID,${userId}\n`;
    csvContent += `导出时间,${new Date().toLocaleString('zh-CN')}\n\n`;
    csvContent += '基本信息\n';
    csvContent += `风险等级,${document.getElementById('userRiskBadge')?.textContent || '未知'}\n`;
    csvContent += `验证次数,${document.getElementById('userVerifyCount')?.textContent || '0'}\n`;
    csvContent += `成功率,${document.getElementById('userSuccessRate')?.textContent || '0%'}\n`;
    csvContent += `最后活动,${document.getElementById('userLastActive')?.textContent || '未知'}\n`;

    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `user_profile_${userId}_${new Date().toISOString().slice(0, 10)}.csv`;
    link.click();
    URL.revokeObjectURL(url);
}
