let trajectoryData = null;
let currentTrajectoryIndex = 0;
let isPlaying = false;
let animationFrameId = null;
let speedEcharts, accelerationEcharts, comparisonEcharts;
let trajectoryCanvas, trajectoryCtx;

document.addEventListener('DOMContentLoaded', () => {
    initializeTrajectoryVisualization();
    loadTrajectoryData();
});

function initializeTrajectoryVisualization() {
    trajectoryCanvas = document.getElementById('trajectoryCanvas');
    if (trajectoryCanvas) {
        trajectoryCtx = trajectoryCanvas.getContext('2d');
        resizeCanvas();
        window.addEventListener('resize', resizeCanvas);
    }

    document.getElementById('refreshTrajectory')?.addEventListener('click', loadTrajectoryData);
    document.getElementById('playTrajectoryBtn')?.addEventListener('click', togglePlayback);
    document.getElementById('prevTrajectoryBtn')?.addEventListener('click', showPrevTrajectory);
    document.getElementById('nextTrajectoryBtn')?.addEventListener('click', showNextTrajectory);
    document.getElementById('trajectorySpeed')?.addEventListener('input', updatePlaybackSpeed);
    document.getElementById('compareTrajectoryBtn')?.addEventListener('click', showComparisonView);
    document.getElementById('exportTrajectoryBtn')?.addEventListener('click', exportTrajectoryData);
    document.getElementById('trajectorySelector')?.addEventListener('change', selectTrajectory);

    initializeEChartsInstances();
}

function resizeCanvas() {
    if (!trajectoryCanvas) return;
    const container = trajectoryCanvas.parentElement;
    const rect = container.getBoundingClientRect();
    trajectoryCanvas.width = rect.width;
    trajectoryCanvas.height = 400;
    if (trajectoryData) {
        drawTrajectory(trajectoryData.trajectories[currentTrajectoryIndex], false);
    }
}

function initializeEChartsInstances() {
    if (document.getElementById('speedChart')) {
        speedEcharts = echarts.init(document.getElementById('speedChart'));
        window.addEventListener('resize', () => speedEcharts.resize());
    }
    if (document.getElementById('accelerationChart')) {
        accelerationEcharts = echarts.init(document.getElementById('accelerationChart'));
        window.addEventListener('resize', () => accelerationEcharts.resize());
    }
    if (document.getElementById('comparisonChart')) {
        comparisonEcharts = echarts.init(document.getElementById('comparisonChart'));
        window.addEventListener('resize', () => comparisonEcharts.resize());
    }
}

async function loadTrajectoryData() {
    try {
        const response = await fetch('/admin/behavior-analytics');
        const result = await response.json();
        if (result.code === 0) {
            trajectoryData = result.data;
            updateTrajectorySelector();
            if (trajectoryData.trajectories && trajectoryData.trajectories.length > 0) {
                currentTrajectoryIndex = 0;
                drawTrajectory(trajectoryData.trajectories[0], false);
                drawFeatureCharts(trajectoryData.trajectories[0]);
            }
        }
    } catch (error) {
        console.error('加载轨迹数据失败:', error);
        loadDemoData();
    }
}

function loadDemoData() {
    trajectoryData = {
        trajectories: [
            {
                id: 'demo_001',
                userId: 'user_12345',
                sessionId: 'session_abcde',
                points: generateDemoTrajectory(100),
                features: {
                    avgSpeed: 2.5,
                    maxSpeed: 8.3,
                    avgAcceleration: 0.4,
                    maxAcceleration: 2.1,
                    smoothness: 0.85,
                    totalDistance: 1250,
                    directDistance: 980,
                    pauseCount: 3,
                    pathRatio: 1.27
                }
            }
        ]
    };
    updateTrajectorySelector();
    if (trajectoryData.trajectories.length > 0) {
        drawTrajectory(trajectoryData.trajectories[0], false);
        drawFeatureCharts(trajectoryData.trajectories[0]);
    }
}

function generateDemoTrajectory(pointCount) {
    const points = [];
    let x = 100, y = 200;
    const targetX = 700, targetY = 350;
    let timestamp = Date.now();

    for (let i = 0; i < pointCount; i++) {
        const progress = i / pointCount;
        const targetXPos = x + (targetX - x) * progress + (Math.random() - 0.5) * 20;
        const targetYPos = y + (targetY - y) * progress + (Math.random() - 0.5) * 20;
        x = targetXPos;
        y = targetYPos;

        points.push({
            x: x,
            y: y,
            timestamp: timestamp,
            event: i % 15 === 0 ? 'click' : 'move',
            velocity: 1 + Math.random() * 3,
            acceleration: (Math.random() - 0.5) * 0.5
        });

        timestamp += 15 + Math.random() * 20;
    }

    return points;
}

function updateTrajectorySelector() {
    const selector = document.getElementById('trajectorySelector');
    if (!selector || !trajectoryData) return;

    selector.innerHTML = '';
    trajectoryData.trajectories.forEach((traj, index) => {
        const option = document.createElement('option');
        option.value = index;
        option.textContent = `${traj.userId} - ${traj.sessionId.substring(0, 8)}`;
        selector.appendChild(option);
    });
}

function selectTrajectory(e) {
    const index = parseInt(e.target.value);
    if (!isNaN(index) && trajectoryData.trajectories[index]) {
        currentTrajectoryIndex = index;
        stopPlayback();
        drawTrajectory(trajectoryData.trajectories[index], false);
        drawFeatureCharts(trajectoryData.trajectories[index]);
    }
}

function drawTrajectory(trajectory, animate) {
    if (!trajectoryCanvas || !trajectoryCtx || !trajectory) return;

    const ctx = trajectoryCtx;
    ctx.clearRect(0, 0, trajectoryCanvas.width, trajectoryCanvas.height);

    const points = trajectory.points;
    if (!points || points.length === 0) return;

    drawTrajectoryGrid(ctx);
    drawTrajectoryPath(ctx, points);
    drawTrajectoryPoints(ctx, points);

    if (trajectory.features) {
        updateFeatureDisplay(trajectory.features);
    }
}

function drawTrajectoryGrid(ctx) {
    ctx.strokeStyle = '#e5e7eb';
    ctx.lineWidth = 1;

    const gridSize = 50;
    for (let x = 0; x < ctx.canvas.width; x += gridSize) {
        ctx.beginPath();
        ctx.moveTo(x, 0);
        ctx.lineTo(x, ctx.canvas.height);
        ctx.stroke();
    }

    for (let y = 0; y < ctx.canvas.height; y += gridSize) {
        ctx.beginPath();
        ctx.moveTo(0, y);
        ctx.lineTo(ctx.canvas.width, y);
        ctx.stroke();
    }
}

function drawTrajectoryPath(ctx, points) {
    if (points.length < 2) return;

    ctx.strokeStyle = '#3b82f6';
    ctx.lineWidth = 2;
    ctx.lineCap = 'round';
    ctx.lineJoin = 'round';

    ctx.beginPath();
    const firstPoint = normalizePoint(points[0], ctx.canvas.width, ctx.canvas.height);
    ctx.moveTo(firstPoint.x, firstPoint.y);

    for (let i = 1; i < points.length; i++) {
        const point = normalizePoint(points[i], ctx.canvas.width, ctx.canvas.height);
        ctx.lineTo(point.x, point.y);
    }

    ctx.stroke();

    const gradient = ctx.createLinearGradient(0, 0, ctx.canvas.width, 0);
    gradient.addColorStop(0, '#3b82f6');
    gradient.addColorStop(0.5, '#10b981');
    gradient.addColorStop(1, '#f59e0b');
    ctx.strokeStyle = gradient;
    ctx.stroke();
}

function drawTrajectoryPoints(ctx, points) {
    points.forEach((point, index) => {
        const normalized = normalizePoint(point, ctx.canvas.width, ctx.canvas.height);

        if (point.event === 'click') {
            ctx.fillStyle = '#ef4444';
            ctx.beginPath();
            ctx.arc(normalized.x, normalized.y, 8, 0, Math.PI * 2);
            ctx.fill();

            ctx.strokeStyle = '#ffffff';
            ctx.lineWidth = 2;
            ctx.stroke();
        } else {
            const size = 3 + (index / points.length) * 2;
            ctx.fillStyle = `rgba(59, 130, 246, ${0.5 + (index / points.length) * 0.5})`;
            ctx.beginPath();
            ctx.arc(normalized.x, normalized.y, size, 0, Math.PI * 2);
            ctx.fill();
        }
    });
}

function normalizePoint(point, canvasWidth, canvasHeight) {
    let maxX = 1920, maxY = 1080;
    const points = trajectoryData?.trajectories[currentTrajectoryIndex]?.points || [];
    if (points.length > 0) {
        maxX = Math.max(...points.map(p => p.x));
        maxY = Math.max(...points.map(p => p.y));
    }

    return {
        x: (point.x / maxX) * canvasWidth * 0.9 + canvasWidth * 0.05,
        y: (point.y / maxY) * canvasHeight * 0.9 + canvasHeight * 0.05
    };
}

function updateFeatureDisplay(features) {
    document.getElementById('avgSpeed').textContent = features.avgSpeed?.toFixed(2) || '0.00';
    document.getElementById('maxSpeed').textContent = features.maxSpeed?.toFixed(2) || '0.00';
    document.getElementById('avgAcceleration').textContent = features.avgAcceleration?.toFixed(2) || '0.00';
    document.getElementById('maxAcceleration').textContent = features.maxAcceleration?.toFixed(2) || '0.00';
    document.getElementById('smoothness').textContent = features.smoothness?.toFixed(2) || '0.00';
    document.getElementById('totalDistance').textContent = features.totalDistance?.toFixed(0) || '0';
    document.getElementById('directDistance').textContent = features.directDistance?.toFixed(0) || '0';
    document.getElementById('pauseCount').textContent = features.pauseCount || '0';
    document.getElementById('pathRatio').textContent = features.pathRatio?.toFixed(2) || '0.00';
}

function drawFeatureCharts(trajectory) {
    if (!trajectory.points || trajectory.points.length < 2) return;

    const points = trajectory.points;
    const speeds = calculateSpeeds(points);
    const accelerations = calculateAccelerations(speeds);

    drawSpeedChart(speeds, points);
    drawAccelerationChart(accelerations, points);
}

function calculateSpeeds(points) {
    const speeds = [];
    for (let i = 1; i < points.length; i++) {
        const prev = points[i - 1];
        const curr = points[i];
        const dx = curr.x - prev.x;
        const dy = curr.y - prev.y;
        const distance = Math.sqrt(dx * dx + dy * dy);
        const time = (curr.timestamp - prev.timestamp) / 1000;
        const speed = time > 0 ? distance / time : 0;
        speeds.push(speed);
    }
    return speeds;
}

function calculateAccelerations(speeds) {
    const accelerations = [];
    for (let i = 1; i < speeds.length; i++) {
        const ds = speeds[i] - speeds[i - 1];
        accelerations.push(Math.abs(ds));
    }
    return accelerations;
}

function drawSpeedChart(speeds, points) {
    if (!speedEcharts) return;

    const option = {
        title: {
            text: '速度曲线',
            left: 'center',
            textStyle: { fontSize: 14, fontWeight: 'normal' }
        },
        tooltip: {
            trigger: 'axis',
            formatter: function(params) {
                return `速度: ${params[0].value.toFixed(2)} px/ms`;
            }
        },
        grid: {
            left: '3%',
            right: '4%',
            bottom: '3%',
            containLabel: true
        },
        xAxis: {
            type: 'category',
            boundaryGap: false,
            data: points.slice(1).map((p, i) => i)
        },
        yAxis: {
            type: 'value',
            name: '速度'
        },
        series: [{
            name: '速度',
            type: 'line',
            smooth: true,
            data: speeds,
            areaStyle: {
                color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                    { offset: 0, color: 'rgba(59, 130, 246, 0.5)' },
                    { offset: 1, color: 'rgba(59, 130, 246, 0.1)' }
                ])
            },
            lineStyle: { color: '#3b82f6', width: 2 },
            itemStyle: { color: '#3b82f6' }
        }]
    };

    speedEcharts.setOption(option);
}

function drawAccelerationChart(accelerations, points) {
    if (!accelerationEcharts) return;

    const option = {
        title: {
            text: '加速度曲线',
            left: 'center',
            textStyle: { fontSize: 14, fontWeight: 'normal' }
        },
        tooltip: {
            trigger: 'axis',
            formatter: function(params) {
                return `加速度: ${params[0].value.toFixed(3)} px/ms²`;
            }
        },
        grid: {
            left: '3%',
            right: '4%',
            bottom: '3%',
            containLabel: true
        },
        xAxis: {
            type: 'category',
            boundaryGap: false,
            data: points.slice(2).map((p, i) => i)
        },
        yAxis: {
            type: 'value',
            name: '加速度'
        },
        series: [{
            name: '加速度',
            type: 'line',
            smooth: true,
            data: accelerations,
            areaStyle: {
                color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                    { offset: 0, color: 'rgba(239, 68, 68, 0.5)' },
                    { offset: 1, color: 'rgba(239, 68, 68, 0.1)' }
                ])
            },
            lineStyle: { color: '#ef4444', width: 2 },
            itemStyle: { color: '#ef4444' }
        }]
    };

    accelerationEcharts.setOption(option);
}

function drawHeatmapECharts(heatmapData) {
    if (!document.getElementById('heatmapEcharts')) return;

    const heatmapChart = echarts.init(document.getElementById('heatmapEcharts'));
    window.addEventListener('resize', () => heatmapChart.resize());

    const points = heatmapData.map(p => [p.x, p.y, p.intensity]);

    const option = {
        title: {
            text: '轨迹热力图',
            left: 'center'
        },
        tooltip: {
            position: 'top',
            formatter: function(params) {
                return `坐标: (${params.data[0]}, ${params.data[1]})<br/>强度: ${(params.data[2] * 100).toFixed(1)}%`;
            }
        },
        grid: {
            height: '70%',
            top: '15%'
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
            bottom: '5%',
            inRange: {
                color: ['#313695', '#4575b4', '#74add1', '#abd9e9', '#ffffbf', '#fee090', '#fdae61', '#f46d43', '#d73027', '#a50026']
            }
        },
        series: [{
            name: '点击强度',
            type: 'heatmap',
            data: points,
            label: { show: false },
            emphasis: {
                itemStyle: {
                    shadowBlur: 10,
                    shadowColor: 'rgba(0, 0, 0, 0.5)'
                }
            },
            blurSize: 30,
            pointSize: 25
        }]
    };

    heatmapChart.setOption(option);
}

function togglePlayback() {
    if (!trajectoryData || trajectoryData.trajectories.length === 0) return;

    isPlaying = !isPlaying;
    const btn = document.getElementById('playTrajectoryBtn');

    if (isPlaying) {
        btn.innerHTML = '<i class="fas fa-pause"></i>';
        animateTrajectory();
    } else {
        btn.innerHTML = '<i class="fas fa-play"></i>';
        stopPlayback();
    }
}

function stopPlayback() {
    isPlaying = false;
    if (animationFrameId) {
        cancelAnimationFrame(animationFrameId);
        animationFrameId = null;
    }
    const btn = document.getElementById('playTrajectoryBtn');
    if (btn) btn.innerHTML = '<i class="fas fa-play"></i>';
}

function animateTrajectory() {
    if (!isPlaying || !trajectoryCanvas || !trajectoryCtx) return;

    const trajectory = trajectoryData.trajectories[currentTrajectoryIndex];
    const points = trajectory.points;
    const speed = parseFloat(document.getElementById('trajectorySpeed')?.value || 1);

    let currentIndex = 0;

    function animate() {
        if (!isPlaying) return;

        trajectoryCtx.clearRect(0, 0, trajectoryCanvas.width, trajectoryCanvas.height);
        drawTrajectoryGrid(trajectoryCtx);

        const visiblePoints = points.slice(0, currentIndex + 1);
        if (visiblePoints.length > 1) {
            drawTrajectoryPath(trajectoryCtx, visiblePoints);
        }

        for (let i = 0; i <= currentIndex; i++) {
            const point = points[i];
            const normalized = normalizePoint(point, trajectoryCanvas.width, trajectoryCanvas.height);

            if (point.event === 'click') {
                trajectoryCtx.fillStyle = '#ef4444';
                trajectoryCtx.beginPath();
                trajectoryCtx.arc(normalized.x, normalized.y, 8, 0, Math.PI * 2);
                trajectoryCtx.fill();
                trajectoryCtx.strokeStyle = '#ffffff';
                trajectoryCtx.lineWidth = 2;
                trajectoryCtx.stroke();
            }
        }

        if (currentIndex >= points.length - 1) {
            isPlaying = false;
            const btn = document.getElementById('playTrajectoryBtn');
            if (btn) btn.innerHTML = '<i class="fas fa-play"></i>';
            return;
        }

        currentIndex++;
        animationFrameId = requestAnimationFrame(() => {
            setTimeout(animate, 50 / speed);
        });
    }

    animate();
}

function updatePlaybackSpeed(e) {
    const speed = parseFloat(e.target.value);
    document.getElementById('speedValue').textContent = speed.toFixed(1) + 'x';
}

function showPrevTrajectory() {
    if (!trajectoryData || trajectoryData.trajectories.length === 0) return;

    stopPlayback();
    currentTrajectoryIndex = (currentTrajectoryIndex - 1 + trajectoryData.trajectories.length) % trajectoryData.trajectories.length;
    document.getElementById('trajectorySelector').value = currentTrajectoryIndex;
    drawTrajectory(trajectoryData.trajectories[currentTrajectoryIndex], false);
    drawFeatureCharts(trajectoryData.trajectories[currentTrajectoryIndex]);
}

function showNextTrajectory() {
    if (!trajectoryData || trajectoryData.trajectories.length === 0) return;

    stopPlayback();
    currentTrajectoryIndex = (currentTrajectoryIndex + 1) % trajectoryData.trajectories.length;
    document.getElementById('trajectorySelector').value = currentTrajectoryIndex;
    drawTrajectory(trajectoryData.trajectories[currentTrajectoryIndex], false);
    drawFeatureCharts(trajectoryData.trajectories[currentTrajectoryIndex]);
}

function showComparisonView() {
    if (!trajectoryData || trajectoryData.trajectories.length < 2) {
        alert('需要至少2条轨迹才能进行对比分析');
        return;
    }

    const comparisonModal = document.getElementById('comparisonModal');
    if (comparisonModal) {
        comparisonModal.style.display = 'flex';
        drawComparisonChart();
    }
}

function drawComparisonChart() {
    if (!comparisonEcharts || trajectoryData.trajectories.length < 2) return;

    const seriesData = trajectoryData.trajectories.slice(0, 3).map((traj, index) => {
        const speeds = calculateSpeeds(traj.points);
        return {
            name: `轨迹 ${index + 1}`,
            type: 'line',
            smooth: true,
            data: speeds,
            lineStyle: { width: 2 },
            itemStyle: {}
        };
    });

    const colors = ['#3b82f6', '#10b981', '#f59e0b'];
    seriesData.forEach((s, i) => {
        s.lineStyle.color = colors[i];
        s.itemStyle.color = colors[i];
    });

    const option = {
        title: {
            text: '轨迹速度对比',
            left: 'center'
        },
        tooltip: {
            trigger: 'axis'
        },
        legend: {
            data: seriesData.map(s => s.name),
            bottom: 0
        },
        grid: {
            left: '3%',
            right: '4%',
            bottom: '15%',
            containLabel: true
        },
        xAxis: {
            type: 'category',
            boundaryGap: false,
            data: Array.from({length: 50}, (_, i) => i)
        },
        yAxis: {
            type: 'value',
            name: '速度'
        },
        series: seriesData
    };

    comparisonEcharts.setOption(option);
}

function closeComparisonModal() {
    const modal = document.getElementById('comparisonModal');
    if (modal) modal.style.display = 'none';
}

function exportTrajectoryData() {
    if (!trajectoryData) return;

    const trajectory = trajectoryData.trajectories[currentTrajectoryIndex];
    const exportData = {
        id: trajectory.id,
        userId: trajectory.userId,
        sessionId: trajectory.sessionId,
        points: trajectory.points,
        features: trajectory.features,
        exportedAt: new Date().toISOString()
    };

    const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `trajectory_${trajectory.id}_${Date.now()}.json`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
}

function clearTrajectoryCanvas() {
    if (!trajectoryCtx) return;
    trajectoryCtx.clearRect(0, 0, trajectoryCanvas.width, trajectoryCanvas.height);
    drawTrajectoryGrid(trajectoryCtx);
}

function resetTrajectoryView() {
    stopPlayback();
    currentTrajectoryIndex = 0;
    if (trajectoryData && trajectoryData.trajectories.length > 0) {
        drawTrajectory(trajectoryData.trajectories[0], false);
        drawFeatureCharts(trajectoryData.trajectories[0]);
    } else {
        clearTrajectoryCanvas();
    }
}
