let riskMonitorChart, threatTrendChart, geoDistributionChart;
let updateTimer = null;

document.addEventListener('DOMContentLoaded', () => {
    initRiskMonitorCharts();
    startRealTimeUpdates();
});

function initRiskMonitorCharts() {
    initThreatTrendChart();
    initGeoDistributionChart();
}

function initThreatTrendChart() {
    const ctx = document.getElementById('threatTrendChart');
    if (!ctx) return;

    const labels = Array.from({ length: 24 }, (_, i) => `${i}:00`);
    const threatData = Array.from({ length: 24 }, () => Math.floor(Math.random() * 50));
    const blockedData = Array.from({ length: 24 }, () => Math.floor(Math.random() * 40));

    riskMonitorChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: labels,
            datasets: [{
                label: '威胁检测',
                data: threatData,
                borderColor: '#ef4444',
                backgroundColor: 'rgba(239, 68, 68, 0.1)',
                fill: true,
                tension: 0.4
            }, {
                label: '成功拦截',
                data: blockedData,
                borderColor: '#10b981',
                backgroundColor: 'rgba(16, 185, 129, 0.1)',
                fill: true,
                tension: 0.4
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    position: 'top',
                    labels: { usePointStyle: true }
                }
            },
            scales: {
                y: {
                    beginAtZero: true,
                    grid: { color: 'rgba(0, 0, 0, 0.05)' }
                }
            }
        }
    });
}

function initGeoDistributionChart() {
    const ctx = document.getElementById('geoDistributionChart');
    if (!ctx) return;

    geoDistributionChart = new Chart(ctx, {
        type: 'bar',
        data: {
            labels: ['北京', '上海', '广东', '浙江', '江苏', '四川', '境外'],
            datasets: [{
                label: '攻击来源占比 (%)',
                data: [25, 20, 18, 12, 10, 8, 7],
                backgroundColor: [
                    'rgba(59, 130, 246, 0.8)',
                    'rgba(16, 185, 129, 0.8)',
                    'rgba(245, 158, 11, 0.8)',
                    'rgba(239, 68, 68, 0.8)',
                    'rgba(139, 92, 246, 0.8)',
                    'rgba(236, 72, 153, 0.8)',
                    'rgba(107, 114, 128, 0.8)'
                ],
                borderWidth: 0,
                borderRadius: 4
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            indexAxis: 'y',
            plugins: {
                legend: { display: false }
            },
            scales: {
                x: {
                    beginAtZero: true,
                    max: 30,
                    grid: { color: 'rgba(0, 0, 0, 0.05)' }
                }
            }
        }
    });
}

function startRealTimeUpdates() {
    updateTimer = setInterval(() => {
        updateRiskStats();
    }, 3000);
}

function updateRiskStats() {
    const threatCount = document.getElementById('threatCount');
    const blockedCount = document.getElementById('blockedCount');
    const safeCount = document.getElementById('safeCount');

    if (threatCount) {
        threatCount.textContent = formatNumber(parseInt(threatCount.textContent) + Math.floor(Math.random() * 5));
    }
    if (blockedCount) {
        blockedCount.textContent = formatNumber(parseInt(blockedCount.textContent.replace(/[^\d]/g, '')) + Math.floor(Math.random() * 10));
    }
    if (safeCount) {
        safeCount.textContent = formatNumber(parseInt(safeCount.textContent.replace(/[^\d]/g, '')) + Math.floor(Math.random() * 50));
    }
}

function formatNumber(num) {
    if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
    if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
    return num.toString();
}

function stopUpdates() {
    if (updateTimer) {
        clearInterval(updateTimer);
        updateTimer = null;
    }
}
