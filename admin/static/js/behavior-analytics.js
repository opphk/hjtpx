let behaviorData = null;
let currentTrajectoryIndex = 0;
let isPlaying = false;
let animationFrameId = null;
let riskChart, interactionChart, speedChart;

document.addEventListener('DOMContentLoaded', () => {
    initializeEventListeners();
    loadBehaviorData();
});

function initializeEventListeners() {
    // 筛选和刷新
    document.getElementById('refreshBtn').addEventListener('click', loadBehaviorData);
    document.getElementById('applyFilterBtn').addEventListener('click', loadBehaviorData);
    
    // 轨迹控制
    document.getElementById('prevTrajectory').addEventListener('click', showPrevTrajectory);
    document.getElementById('nextTrajectory').addEventListener('click', showNextTrajectory);
    document.getElementById('playTrajectory').addEventListener('click', togglePlay);
    document.getElementById('trajectorySelect').addEventListener('change', (e) => {
        if (e.target.value) {
            currentTrajectoryIndex = parseInt(e.target.value);
            drawTrajectory(behaviorData.trajectories[currentTrajectoryIndex], false);
        }
    });
}

async function loadBehaviorData() {
    try {
        const period = document.getElementById('dateRange').value;
        const result = await auth.request(`/admin/behavior-analytics?period=${period}`);
        if (result.code === 0) {
            behaviorData = result.data;
            updateUI();
        } else {
            console.error('Failed to load behavior data');
        }
    } catch (error) {
        console.error('Error loading behavior data:', error);
    }
}

function updateUI() {
    if (!behaviorData) return;
    
    // 更新概览卡片
    document.getElementById('statTotalSessions').textContent = formatNumber(behaviorData.summary.totalSessions);
    document.getElementById('statTotalInteractions').textContent = formatNumber(behaviorData.summary.totalInteractions);
    document.getElementById('statAvgDuration').textContent = behaviorData.summary.avgSessionDuration.toFixed(1);
    document.getElementById('statHighRiskUsers').textContent = behaviorData.summary.highRiskUsers;
    
    // 绘制热力图
    drawHeatmap(behaviorData.heatmap);
    
    // 绘制图表
    drawRiskChart(behaviorData.riskDistribution);
    drawInteractionChart(behaviorData.summary);
    drawSpeedChart();
    
    // 更新轨迹选择
    updateTrajectorySelect();
    
    // 绘制第一个轨迹
    if (behaviorData.trajectories.length > 0) {
        currentTrajectoryIndex = 0;
        drawTrajectory(behaviorData.trajectories[0], false);
    }
    
    // 更新异常表
    updateAnomalyTable(behaviorData.anomalies);
}

function formatNumber(num) {
    if (num >= 1000000) {
        return (num / 1000000).toFixed(1) + 'M';
    } else if (num >= 1000) {
        return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
}

function drawHeatmap(points) {
    const canvas = document.getElementById('heatmapCanvas');
    const ctx = canvas.getContext('2d');
    
    // 调整canvas尺寸
    const rect = canvas.getBoundingClientRect();
    canvas.width = rect.width;
    canvas.height = 400;
    
    // 清空画布
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    
    // 绘制热力点
    points.forEach(point => {
        // 将坐标映射到canvas尺寸
        const x = (point.x / 1920) * canvas.width;
        const y = (point.y / 1080) * canvas.height;
        const intensity = point.intensity;
        
        // 创建径向渐变
        const radius = 30;
        const gradient = ctx.createRadialGradient(x, y, 0, x, y, radius);
        const color = getHeatColor(intensity);
        
        gradient.addColorStop(0, color.replace('1)', '0.8)'));
        gradient.addColorStop(0.5, color.replace('1)', '0.4)'));
        gradient.addColorStop(1, color.replace('1)', '0)'));
        
        ctx.fillStyle = gradient;
        ctx.beginPath();
        ctx.arc(x, y, radius, 0, 2 * Math.PI);
        ctx.fill();
    });
}

function getHeatColor(intensity) {
    if (intensity < 0.2) {
        return 'rgba(0, 255, 255, 1)';
    } else if (intensity < 0.4) {
        return 'rgba(0, 255, 0, 1)';
    } else if (intensity < 0.6) {
        return 'rgba(255, 255, 0, 1)';
    } else if (intensity < 0.8) {
        return 'rgba(255, 165, 0, 1)';
    } else {
        return 'rgba(255, 0, 0, 1)';
    }
}

function drawRiskChart(distribution) {
    const ctx = document.getElementById('riskChart').getContext('2d');
    
    if (riskChart) {
        riskChart.destroy();
    }
    
    riskChart = new Chart(ctx, {
        type: 'bar',
        data: {
            labels: distribution.map(d => d.range),
            datasets: [{
                label: '用户数量',
                data: distribution.map(d => d.count),
                backgroundColor: [
                    'rgba(40, 167, 69, 0.8)',
                    'rgba(255, 193, 7, 0.8)',
                    'rgba(253, 126, 20, 0.8)',
                    'rgba(220, 53, 69, 0.8)',
                    'rgba(128, 0, 32, 0.8)'
                ],
                borderColor: [
                    'rgba(40, 167, 69, 1)',
                    'rgba(255, 193, 7, 1)',
                    'rgba(253, 126, 20, 1)',
                    'rgba(220, 53, 69, 1)',
                    'rgba(128, 0, 32, 1)'
                ],
                borderWidth: 1,
                borderRadius: 4
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: { display: false },
                tooltip: {
                    callbacks: {
                        label: function(context) {
                            const d = distribution[context.dataIndex];
                            return `用户数: ${d.count}, 占比: ${d.percentage}%`;
                        }
                    }
                }
            },
            scales: {
                y: { beginAtZero: true }
            }
        }
    });
}

function drawInteractionChart(summary) {
    const ctx = document.getElementById('interactionChart').getContext('2d');
    
    if (interactionChart) {
        interactionChart.destroy();
    }
    
    interactionChart = new Chart(ctx, {
        type: 'doughnut',
        data: {
            labels: ['点击次数', '键盘事件'],
            datasets: [{
                data: [summary.clickCount, summary.keyboardEventCount],
                backgroundColor: [
                    'rgba(59, 130, 246, 0.8)',
                    'rgba(16, 185, 129, 0.8)'
                ],
                borderWidth: 2,
                borderColor: '#fff'
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    position: 'bottom'
                }
            }
        }
    });
}

function drawSpeedChart() {
    const ctx = document.getElementById('speedChart').getContext('2d');
    
    if (speedChart) {
        speedChart.destroy();
    }
    
    // 生成模拟趋势数据
    const labels = ['周一', '周二', '周三', '周四', '周五', '周六', '周日'];
    const data = labels.map(() => 3 + Math.random() * 3);
    
    speedChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: labels,
            datasets: [{
                label: '平均鼠标速度',
                data: data,
                borderColor: '#3b82f6',
                backgroundColor: 'rgba(59, 130, 246, 0.1)',
                fill: true,
                tension: 0.4,
                pointRadius: 4,
                pointHoverRadius: 6
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: { legend: { display: false } },
            scales: { y: { beginAtZero: true } }
        }
    });
}

function updateTrajectorySelect() {
    const select = document.getElementById('trajectorySelect');
    select.innerHTML = '<option value="">选择会话...</option>';
    
    behaviorData.trajectories.forEach((trajectory, index) => {
        const option = document.createElement('option');
        option.value = index;
        option.textContent = `${trajectory.userId} - ${trajectory.sessionId}`;
        select.appendChild(option);
    });
}

function drawTrajectory(trajectory, animate) {
    const canvas = document.getElementById('trajectoryCanvas');
    const ctx = canvas.getContext('2d');
    
    const rect = canvas.getBoundingClientRect();
    canvas.width = rect.width;
    canvas.height = 400;
    
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    
    if (!trajectory || trajectory.points.length === 0) return;
    
    const points = trajectory.points;
    
    if (animate) {
        animateTrajectory(ctx, points, 0);
    } else {
        // 直接绘制完整轨迹
        drawTrajectoryLine(ctx, points, 1);
        drawTrajectoryPoints(ctx, points);
    }
}

function drawTrajectoryLine(ctx, points, progress) {
    const endIndex = Math.floor(points.length * progress);
    
    ctx.strokeStyle = '#3b82f6';
    ctx.lineWidth = 2;
    ctx.beginPath();
    
    for (let i = 0; i < endIndex; i++) {
        const x = (points[i].x / 1920) * ctx.canvas.width;
        const y = (points[i].y / 1080) * ctx.canvas.height;
        
        if (i === 0) {
            ctx.moveTo(x, y);
        } else {
            ctx.lineTo(x, y);
        }
    }
    
    ctx.stroke();
}

function drawTrajectoryPoints(ctx, points) {
    points.forEach((point, index) => {
        const x = (point.x / 1920) * ctx.canvas.width;
        const y = (point.y / 1080) * ctx.canvas.height;
        
        if (point.event === 'click') {
            // 点击点用红色
            ctx.fillStyle = '#ef4444';
            ctx.beginPath();
            ctx.arc(x, y, 6, 0, 2 * Math.PI);
            ctx.fill();
        } else {
            // 普通点用蓝色
            ctx.fillStyle = '#3b82f6';
            ctx.beginPath();
            ctx.arc(x, y, 3, 0, 2 * Math.PI);
            ctx.fill();
        }
    });
}

function animateTrajectory(ctx, points, index) {
    if (index >= points.length || !isPlaying) {
        if (isPlaying) {
            // 绘制完整轨迹和点
            drawTrajectoryLine(ctx, points, 1);
            drawTrajectoryPoints(ctx, points);
            isPlaying = false;
            document.getElementById('playTrajectory').innerHTML = '<i class="fas fa-play"></i>';
        }
        return;
    }
    
    ctx.clearRect(0, 0, ctx.canvas.width, ctx.canvas.height);
    
    // 绘制到当前点的轨迹
    drawTrajectoryLine(ctx, points, (index + 1) / points.length);
    
    // 绘制当前点之前的点击点
    for (let i = 0; i <= index; i++) {
        if (points[i].event === 'click') {
            const x = (points[i].x / 1920) * ctx.canvas.width;
            const y = (points[i].y / 1080) * ctx.canvas.height;
            ctx.fillStyle = '#ef4444';
            ctx.beginPath();
            ctx.arc(x, y, 6, 0, 2 * Math.PI);
            ctx.fill();
        }
    }
    
    // 绘制当前点
    const currentX = (points[index].x / 1920) * ctx.canvas.width;
    const currentY = (points[index].y / 1080) * ctx.canvas.height;
    ctx.fillStyle = '#ef4444';
    ctx.beginPath();
    ctx.arc(currentX, currentY, 8, 0, 2 * Math.PI);
    ctx.fill();
    
    animationFrameId = requestAnimationFrame(() => {
        setTimeout(() => animateTrajectory(ctx, points, index + 1), 30);
    });
}

function togglePlay() {
    if (!behaviorData || behaviorData.trajectories.length === 0) return;
    
    isPlaying = !isPlaying;
    const btn = document.getElementById('playTrajectory');
    
    if (isPlaying) {
        btn.innerHTML = '<i class="fas fa-pause"></i>';
        drawTrajectory(behaviorData.trajectories[currentTrajectoryIndex], true);
    } else {
        btn.innerHTML = '<i class="fas fa-play"></i>';
        if (animationFrameId) {
            cancelAnimationFrame(animationFrameId);
        }
    }
}

function showPrevTrajectory() {
    if (!behaviorData || behaviorData.trajectories.length === 0) return;
    
    isPlaying = false;
    document.getElementById('playTrajectory').innerHTML = '<i class="fas fa-play"></i>';
    
    currentTrajectoryIndex = (currentTrajectoryIndex - 1 + behaviorData.trajectories.length) % behaviorData.trajectories.length;
    document.getElementById('trajectorySelect').value = currentTrajectoryIndex;
    drawTrajectory(behaviorData.trajectories[currentTrajectoryIndex], false);
}

function showNextTrajectory() {
    if (!behaviorData || behaviorData.trajectories.length === 0) return;
    
    isPlaying = false;
    document.getElementById('playTrajectory').innerHTML = '<i class="fas fa-play"></i>';
    
    currentTrajectoryIndex = (currentTrajectoryIndex + 1) % behaviorData.trajectories.length;
    document.getElementById('trajectorySelect').value = currentTrajectoryIndex;
    drawTrajectory(behaviorData.trajectories[currentTrajectoryIndex], false);
}

function updateAnomalyTable(anomalies) {
    const tbody = document.getElementById('anomalyTableBody');
    tbody.innerHTML = '';
    
    anomalies.forEach(anomaly => {
        const tr = document.createElement('tr');
        tr.innerHTML = `
            <td>${anomaly.type}</td>
            <td>${anomaly.description}</td>
            <td><span class="badge severity-${anomaly.severity}">${anomaly.severity}</span></td>
            <td>${anomaly.count}</td>
            <td>${anomaly.affectedUsers}</td>
        `;
        tbody.appendChild(tr);
    });
}

function exportData(format) {
    if (!behaviorData) return;
    
    const period = document.getElementById('dateRange').value;
    const url = `/admin/behavior-analytics/export?format=${format}&period=${period}`;
    
    // 创建临时链接来下载
    const link = document.createElement('a');
    link.href = url;
    link.download = `behavior_analytics_${period}.${format}`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
}
