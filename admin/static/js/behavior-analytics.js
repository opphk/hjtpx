let behaviorData = null;
let currentTrajectoryIndex = 0;
let isPlaying = false;
let animationFrameId = null;
let riskChart, interactionChart, speedChart;
let sankeyChart, heatmapEcharts, radarChart;

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
    
    // 绘制高级分析图表
    drawSankeyChart(behaviorData.sankeyData);
    drawHeatmapEcharts(behaviorData.heatmap);
    drawRadarChart(behaviorData.radarData);
    
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

// 绘制桑基图
function drawSankeyChart(sankeyData) {
    if (!sankeyChart) {
        sankeyChart = echarts.init(document.getElementById('sankeyChart'));
        window.addEventListener('resize', () => {
            sankeyChart.resize();
        });
    }
    
    // 默认数据
    const defaultData = {
        nodes: [
            { name: '进入页面' },
            { name: '浏览内容' },
            { name: '点击验证码' },
            { name: '验证成功' },
            { name: '验证失败' },
            { name: '重新验证' },
            { name: '离开页面' }
        ],
        links: [
            { source: '进入页面', target: '浏览内容', value: 1000 },
            { source: '浏览内容', target: '点击验证码', value: 800 },
            { source: '点击验证码', target: '验证成功', value: 600 },
            { source: '点击验证码', target: '验证失败', value: 200 },
            { source: '验证失败', target: '重新验证', value: 150 },
            { source: '验证失败', target: '离开页面', value: 50 },
            { source: '重新验证', target: '验证成功', value: 120 },
            { source: '重新验证', target: '离开页面', value: 30 },
            { source: '验证成功', target: '离开页面', value: 720 },
            { source: '浏览内容', target: '离开页面', value: 200 }
        ]
    };
    
    const data = sankeyData || defaultData;
    
    const option = {
        tooltip: {
            trigger: 'item',
            triggerOn: 'mousemove'
        },
        series: [
            {
                type: 'sankey',
                data: data.nodes,
                links: data.links,
                emphasis: {
                    focus: 'adjacency'
                },
                lineStyle: {
                    color: 'gradient',
                    curveness: 0.5
                },
                label: {
                    color: '#333'
                }
            }
        ]
    };
    
    sankeyChart.setOption(option);
}

// 绘制ECharts热力图
function drawHeatmapEcharts(heatmapData) {
    if (!heatmapEcharts) {
        heatmapEcharts = echarts.init(document.getElementById('heatmapEcharts'));
        window.addEventListener('resize', () => {
            heatmapEcharts.resize();
        });
    }
    
    // 处理数据
    const data = heatmapData || [];
    const points = data.map(point => [point.x, point.y, point.intensity]);
    
    const option = {
        tooltip: {
            position: 'top',
            formatter: function (params) {
                return `坐标: (${params.data[0]}, ${params.data[1]})<br/>强度: ${(params.data[2] * 100).toFixed(1)}%`;
            }
        },
        grid: {
            height: '80%',
            top: '10%'
        },
        xAxis: {
            type: 'value',
            min: 0,
            max: 1920,
            splitNumber: 10,
            axisLine: { onZero: false }
        },
        yAxis: {
            type: 'value',
            min: 0,
            max: 1080,
            splitNumber: 10,
            axisLine: { onZero: false }
        },
        visualMap: {
            min: 0,
            max: 1,
            calculable: true,
            orient: 'horizontal',
            left: 'center',
            bottom: '0%',
            inRange: {
                color: ['#313695', '#4575b4', '#74add1', '#abd9e9', '#e0f3f8', '#ffffbf', '#fee090', '#fdae61', '#f46d43', '#d73027', '#a50026']
            }
        },
        series: [{
            name: '点击强度',
            type: 'heatmap',
            data: points,
            label: {
                show: false
            },
            emphasis: {
                itemStyle: {
                    shadowBlur: 10,
                    shadowColor: 'rgba(0, 0, 0, 0.5)'
                }
            },
            blurSize: 30,
            pointSize: 20
        }]
    };
    
    heatmapEcharts.setOption(option);
}

// 绘制雷达图
function drawRadarChart(radarData) {
    if (!radarChart) {
        radarChart = echarts.init(document.getElementById('radarChart'));
        window.addEventListener('resize', () => {
            radarChart.resize();
        });
    }
    
    // 默认数据
    const defaultData = {
        indicator: [
            { name: '鼠标移动速度', max: 100 },
            { name: '点击间隔', max: 100 },
            { name: '轨迹直线度', max: 100 },
            { name: '操作频率', max: 100 },
            { name: '响应时间', max: 100 },
            { name: '一致性', max: 100 }
        ],
        data: [
            {
                value: [20, 15, 25, 30, 20, 25],
                name: '正常用户',
                itemStyle: { color: '#28a745' },
                areaStyle: { opacity: 0.3 }
            },
            {
                value: [75, 80, 85, 70, 80, 90],
                name: '高风险用户',
                itemStyle: { color: '#dc3545' },
                areaStyle: { opacity: 0.3 }
            }
        ]
    };
    
    const data = radarData || defaultData;
    
    const option = {
        tooltip: {},
        legend: {
            data: data.data.map(item => item.name),
            bottom: '0%'
        },
        radar: {
            indicator: data.indicator,
            center: ['50%', '55%'],
            radius: '65%'
        },
        series: [{
            name: '风险分析',
            type: 'radar',
            data: data.data
        }]
    };
    
    radarChart.setOption(option);
}
