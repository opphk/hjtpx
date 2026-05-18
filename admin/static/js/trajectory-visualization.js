let trajectoryData = null;
let currentTrajectoryIndex = 0;
let isPlaying = false;
let animationFrameId = null;
let speedEcharts, accelerationEcharts, comparisonEcharts, heatmapEcharts, scatter3DEcharts;
let trajectoryCanvas, trajectoryCtx;
let heatmapData = [];
let playSpeed = 1;
let selectedPoints = [];

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
        trajectoryCanvas.addEventListener('click', handleCanvasClick);
        trajectoryCanvas.addEventListener('mousemove', handleCanvasHover);
    }

    document.getElementById('refreshTrajectory')?.addEventListener('click', loadTrajectoryData);
    document.getElementById('playTrajectoryBtn')?.addEventListener('click', togglePlayback);
    document.getElementById('prevTrajectoryBtn')?.addEventListener('click', showPrevTrajectory);
    document.getElementById('nextTrajectoryBtn')?.addEventListener('click', showNextTrajectory);
    document.getElementById('trajectorySpeed')?.addEventListener('input', updatePlaybackSpeed);
    document.getElementById('compareTrajectoryBtn')?.addEventListener('click', showComparisonView);
    document.getElementById('exportTrajectoryBtn')?.addEventListener('click', exportTrajectoryData);
    document.getElementById('trajectorySelector')?.addEventListener('change', selectTrajectory);
    document.getElementById('showHeatmapBtn')?.addEventListener('click', toggleHeatmap);
    document.getElementById('show3DViewBtn')?.addEventListener('click', toggle3DView);
    document.getElementById('clearSelectionBtn')?.addEventListener('click', clearSelection);
    document.getElementById('analyzeBtn')?.addEventListener('click', analyzeTrajectory);

    initializeEChartsInstances();
}

function resizeCanvas() {
    if (!trajectoryCanvas) return;
    const container = trajectoryCanvas.parentElement;
    const rect = container.getBoundingClientRect();
    trajectoryCanvas.width = rect.width;
    trajectoryCanvas.height = 450;
    if (trajectoryData) {
        drawTrajectory(trajectoryData.trajectories[currentTrajectoryIndex], false);
        if (heatmapEcharts) {
            heatmapEcharts.resize();
        }
        if (scatter3DEcharts) {
            scatter3DEcharts.resize();
        }
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
    if (document.getElementById('heatmapEcharts')) {
        heatmapEcharts = echarts.init(document.getElementById('heatmapEcharts'));
        window.addEventListener('resize', () => heatmapEcharts.resize());
    }
    if (document.getElementById('scatter3DEcharts')) {
        scatter3DEcharts = echarts.init(document.getElementById('scatter3DEcharts'));
        window.addEventListener('resize', () => scatter3DEcharts.resize());
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
                generateHeatmapData(trajectoryData.trajectories[0]);
                draw3DView(trajectoryData.trajectories[0]);
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
                    pathRatio: 1.27,
                    riskScore: 25.5,
                    isHuman: true,
                    anomalyScore: 0.15
                }
            },
            {
                id: 'demo_002',
                userId: 'user_67890',
                sessionId: 'session_fghij',
                points: generateDemoTrajectory(80),
                features: {
                    avgSpeed: 12.8,
                    maxSpeed: 25.6,
                    avgAcceleration: 1.8,
                    maxAcceleration: 5.2,
                    smoothness: 0.62,
                    totalDistance: 1800,
                    directDistance: 1750,
                    pauseCount: 1,
                    pathRatio: 1.03,
                    riskScore: 78.3,
                    isHuman: false,
                    anomalyScore: 0.85
                }
            },
            {
                id: 'demo_003',
                userId: 'user_abcde',
                sessionId: 'session_klmno',
                points: generateDemoTrajectory(120),
                features: {
                    avgSpeed: 3.2,
                    maxSpeed: 9.1,
                    avgAcceleration: 0.5,
                    maxAcceleration: 2.8,
                    smoothness: 0.82,
                    totalDistance: 1500,
                    directDistance: 1100,
                    pauseCount: 5,
                    pathRatio: 1.36,
                    riskScore: 32.1,
                    isHuman: true,
                    anomalyScore: 0.22
                }
            }
        ]
    };
    updateTrajectorySelector();
    if (trajectoryData.trajectories.length > 0) {
        drawTrajectory(trajectoryData.trajectories[0], false);
        drawFeatureCharts(trajectoryData.trajectories[0]);
        generateHeatmapData(trajectoryData.trajectories[0]);
        draw3DView(trajectoryData.trajectories[0]);
    }
}

function generateDemoTrajectory(pointCount) {
    const points = [];
    let x = 100, y = 200;
    const targetX = 700, targetY = 350;
    let timestamp = Date.now();
    let velocity = 2;

    for (let i = 0; i < pointCount; i++) {
        const progress = i / pointCount;
        const noise = Math.sin(i * 0.1) * 15 + (Math.random() - 0.5) * 10;
        const targetXPos = x + (targetX - x) * progress + noise;
        const targetYPos = y + (targetY - y) * progress + noise * 0.5;
        
        x = Math.max(20, Math.min(trajectoryCanvas?.width - 20 || 800, targetXPos));
        y = Math.max(20, Math.min(trajectoryCanvas?.height - 20 || 400, targetYPos));

        velocity = Math.max(0.5, Math.min(15, velocity + (Math.random() - 0.5) * 0.5));
        
        points.push({
            x: x,
            y: y,
            timestamp: timestamp,
            event: i % 15 === 0 ? 'click' : i % 10 === 0 ? 'pause' : 'move',
            velocity: velocity,
            acceleration: (Math.random() - 0.5) * 0.8,
            pressure: 0.3 + Math.random() * 0.4,
            tiltX: (Math.random() - 0.5) * 30,
            tiltY: (Math.random() - 0.5) * 30
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
        option.textContent = `${traj.userId} - ${traj.sessionId.substring(0, 8)} (风险: ${traj.features?.riskScore?.toFixed(1) || 'N/A'})`;
        selector.appendChild(option);
    });
}

function selectTrajectory(e) {
    const index = parseInt(e.target.value);
    if (!isNaN(index) && trajectoryData.trajectories[index]) {
        currentTrajectoryIndex = index;
        stopPlayback();
        selectedPoints = [];
        drawTrajectory(trajectoryData.trajectories[index], false);
        drawFeatureCharts(trajectoryData.trajectories[index]);
        generateHeatmapData(trajectoryData.trajectories[index]);
        draw3DView(trajectoryData.trajectories[index]);
    }
}

function drawTrajectory(trajectory, animate) {
    if (!trajectoryCanvas || !trajectoryCtx || !trajectory) return;

    const ctx = trajectoryCtx;
    ctx.clearRect(0, 0, trajectoryCanvas.width, trajectoryCanvas.height);

    const points = trajectory.points;
    if (!points || points.length === 0) return;

    drawTrajectoryGrid(ctx);
    
    if (!animate) {
        drawTrajectoryPath(ctx, points);
        drawTrajectoryPoints(ctx, points);
        drawSelectedPoints(ctx);
    }

    if (trajectory.features) {
        updateFeatureDisplay(trajectory.features);
    }
}

function drawTrajectoryGrid(ctx) {
    ctx.strokeStyle = '#e5e7eb';
    ctx.lineWidth = 1;
    ctx.globalAlpha = 0.3;

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

    ctx.globalAlpha = 1;
}

function drawTrajectoryPath(ctx, points) {
    if (points.length < 2) return;

    const gradient = ctx.createLinearGradient(0, 0, ctx.canvas.width, 0);
    gradient.addColorStop(0, '#3b82f6');
    gradient.addColorStop(0.5, '#10b981');
    gradient.addColorStop(1, '#f59e0b');

    ctx.strokeStyle = gradient;
    ctx.lineWidth = 3;
    ctx.lineCap = 'round';
    ctx.lineJoin = 'round';

    ctx.beginPath();
    const firstPoint = normalizePoint(points[0], ctx.canvas.width, ctx.canvas.height);
    ctx.moveTo(firstPoint.x, firstPoint.y);

    for (let i = 1; i < points.length; i++) {
        const point = normalizePoint(points[i], ctx.canvas.width, ctx.canvas.height);
        const prevPoint = normalizePoint(points[i - 1], ctx.canvas.width, ctx.canvas.height);
        
        const cpX = (prevPoint.x + point.x) / 2;
        const cpY = (prevPoint.y + point.y) / 2;
        ctx.quadraticCurveTo(prevPoint.x, prevPoint.y, cpX, cpY);
    }
    
    const lastPoint = normalizePoint(points[points.length - 1], ctx.canvas.width, ctx.canvas.height);
    ctx.lineTo(lastPoint.x, lastPoint.y);
    ctx.stroke();

    ctx.strokeStyle = '#ffffff';
    ctx.lineWidth = 1;
    ctx.stroke();
}

function drawTrajectoryPoints(ctx, points) {
    points.forEach((point, index) => {
        const normalized = normalizePoint(point, ctx.canvas.width, ctx.canvas.height);
        const progress = index / points.length;

        if (point.event === 'click') {
            ctx.fillStyle = '#ef4444';
            ctx.beginPath();
            ctx.arc(normalized.x, normalized.y, 10, 0, Math.PI * 2);
            ctx.fill();

            ctx.strokeStyle = '#ffffff';
            ctx.lineWidth = 3;
            ctx.stroke();

            ctx.fillStyle = '#ffffff';
            ctx.font = 'bold 10px Arial';
            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';
            ctx.fillText('C', normalized.x, normalized.y);
        } else if (point.event === 'pause') {
            ctx.fillStyle = '#9ca3af';
            ctx.beginPath();
            ctx.arc(normalized.x, normalized.y, 6, 0, Math.PI * 2);
            ctx.fill();
        } else {
            const size = 3 + progress * 3;
            const alpha = 0.4 + progress * 0.6;
            ctx.fillStyle = `rgba(59, 130, 246, ${alpha})`;
            ctx.beginPath();
            ctx.arc(normalized.x, normalized.y, size, 0, Math.PI * 2);
            ctx.fill();
        }
    });

    const startPoint = normalizePoint(points[0], ctx.canvas.width, ctx.canvas.height);
    ctx.fillStyle = '#10b981';
    ctx.beginPath();
    ctx.moveTo(startPoint.x, startPoint.y - 8);
    ctx.lineTo(startPoint.x + 6, startPoint.y + 4);
    ctx.lineTo(startPoint.x - 6, startPoint.y + 4);
    ctx.closePath();
    ctx.fill();

    const endPoint = normalizePoint(points[points.length - 1], ctx.canvas.width, ctx.canvas.height);
    ctx.fillStyle = '#ef4444';
    ctx.beginPath();
    ctx.moveTo(endPoint.x, endPoint.y);
    ctx.lineTo(endPoint.x + 8, endPoint.y);
    ctx.lineTo(endPoint.x + 4, endPoint.y - 8);
    ctx.closePath();
    ctx.fill();
}

function drawSelectedPoints(ctx) {
    selectedPoints.forEach(pointIndex => {
        const trajectory = trajectoryData.trajectories[currentTrajectoryIndex];
        if (!trajectory || !trajectory.points[pointIndex]) return;
        
        const point = trajectory.points[pointIndex];
        const normalized = normalizePoint(point, ctx.canvas.width, ctx.canvas.height);
        
        ctx.fillStyle = 'rgba(245, 158, 11, 0.3)';
        ctx.beginPath();
        ctx.arc(normalized.x, normalized.y, 20, 0, Math.PI * 2);
        ctx.fill();
        
        ctx.strokeStyle = '#f59e0b';
        ctx.lineWidth = 2;
        ctx.setLineDash([5, 5]);
        ctx.stroke();
        ctx.setLineDash([]);
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
    
    if (features.riskScore !== undefined) {
        document.getElementById('riskScore').textContent = features.riskScore.toFixed(1);
        const riskBar = document.getElementById('riskScoreBar');
        if (riskBar) {
            riskBar.style.width = features.riskScore + '%';
            riskBar.className = `risk-bar ${getRiskClass(features.riskScore)}`;
        }
        document.getElementById('isHuman').textContent = features.isHuman ? '是' : '否';
    }
    if (features.anomalyScore !== undefined) {
        document.getElementById('anomalyScore').textContent = (features.anomalyScore * 100).toFixed(0) + '%';
    }
}

function getRiskClass(score) {
    if (score >= 70) return 'risk-critical';
    if (score >= 50) return 'risk-high';
    if (score >= 30) return 'risk-medium';
    return 'risk-low';
}

function drawFeatureCharts(trajectory) {
    if (!trajectory.points || trajectory.points.length < 2) return;

    const points = trajectory.points;
    const speeds = calculateSpeeds(points);
    const accelerations = calculateAccelerations(speeds);
    const velocities = points.map(p => p.velocity || 0);

    drawSpeedChart(speeds, points);
    drawAccelerationChart(accelerations, points);
    drawVelocityChart(velocities, points);
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
        backgroundColor: 'transparent',
        title: {
            text: '速度曲线',
            left: 'center',
            textStyle: { fontSize: 14, fontWeight: 'normal', color: '#9ca3af' }
        },
        tooltip: {
            trigger: 'axis',
            backgroundColor: 'rgba(0,0,0,0.8)',
            textStyle: { color: '#fff' },
            formatter: function(params) {
                return `速度: ${params[0].value.toFixed(2)} px/ms`;
            }
        },
        grid: {
            left: '3%',
            right: '4%',
            bottom: '3%',
            top: '15%',
            containLabel: true
        },
        xAxis: {
            type: 'category',
            boundaryGap: false,
            data: points.slice(1).map((p, i) => i),
            axisLabel: { color: '#9ca3af', fontSize: 10 }
        },
        yAxis: {
            type: 'value',
            name: '速度',
            nameTextStyle: { color: '#9ca3af' },
            axisLabel: { color: '#9ca3af' },
            splitLine: { lineStyle: { color: 'rgba(255,255,255,0.1)' } }
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
            itemStyle: { color: '#3b82f6' },
            symbol: 'circle',
            symbolSize: 4
        }]
    };

    speedEcharts.setOption(option);
}

function drawAccelerationChart(accelerations, points) {
    if (!accelerationEcharts) return;

    const option = {
        backgroundColor: 'transparent',
        title: {
            text: '加速度曲线',
            left: 'center',
            textStyle: { fontSize: 14, fontWeight: 'normal', color: '#9ca3af' }
        },
        tooltip: {
            trigger: 'axis',
            backgroundColor: 'rgba(0,0,0,0.8)',
            textStyle: { color: '#fff' },
            formatter: function(params) {
                return `加速度: ${params[0].value.toFixed(3)} px/ms²`;
            }
        },
        grid: {
            left: '3%',
            right: '4%',
            bottom: '3%',
            top: '15%',
            containLabel: true
        },
        xAxis: {
            type: 'category',
            boundaryGap: false,
            data: points.slice(2).map((p, i) => i),
            axisLabel: { color: '#9ca3af', fontSize: 10 }
        },
        yAxis: {
            type: 'value',
            name: '加速度',
            nameTextStyle: { color: '#9ca3af' },
            axisLabel: { color: '#9ca3af' },
            splitLine: { lineStyle: { color: 'rgba(255,255,255,0.1)' } }
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
            itemStyle: { color: '#ef4444' },
            symbol: 'circle',
            symbolSize: 4
        }]
    };

    accelerationEcharts.setOption(option);
}

function drawVelocityChart(velocities, points) {
    if (!document.getElementById('velocityChart')) return;
    
    const velocityChart = echarts.init(document.getElementById('velocityChart'));
    window.addEventListener('resize', () => velocityChart.resize());

    const option = {
        backgroundColor: 'transparent',
        title: {
            text: '速度值序列',
            left: 'center',
            textStyle: { fontSize: 14, fontWeight: 'normal', color: '#9ca3af' }
        },
        tooltip: {
            trigger: 'axis',
            backgroundColor: 'rgba(0,0,0,0.8)',
            textStyle: { color: '#fff' }
        },
        grid: {
            left: '3%',
            right: '4%',
            bottom: '3%',
            top: '15%',
            containLabel: true
        },
        xAxis: {
            type: 'category',
            boundaryGap: false,
            data: points.map((p, i) => i),
            axisLabel: { color: '#9ca3af', fontSize: 10 }
        },
        yAxis: {
            type: 'value',
            name: '速度',
            nameTextStyle: { color: '#9ca3af' },
            axisLabel: { color: '#9ca3af' },
            splitLine: { lineStyle: { color: 'rgba(255,255,255,0.1)' } }
        },
        series: [{
            name: '速度',
            type: 'bar',
            data: velocities,
            itemStyle: {
                color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                    { offset: 0, color: '#10b981' },
                    { offset: 1, color: '#059669' }
                ]),
                borderRadius: [4, 4, 0, 0]
            }
        }]
    };

    velocityChart.setOption(option);
}

function generateHeatmapData(trajectory) {
    if (!trajectory || !trajectory.points) return;

    heatmapData = [];
    const points = trajectory.points;
    
    const gridSize = 40;
    const gridMap = {};
    
    points.forEach(point => {
        const gridX = Math.floor(point.x / gridSize);
        const gridY = Math.floor(point.y / gridSize);
        const key = `${gridX}-${gridY}`;
        
        if (!gridMap[key]) {
            gridMap[key] = 0;
        }
        gridMap[key]++;
    });

    for (const key in gridMap) {
        const [gx, gy] = key.split('-').map(Number);
        heatmapData.push([gx * gridSize + gridSize / 2, gy * gridSize + gridSize / 2, gridMap[key] / points.length]);
    }

    drawHeatmap();
}

function drawHeatmap() {
    if (!heatmapEcharts) return;

    const option = {
        backgroundColor: 'transparent',
        title: {
            text: '轨迹热力图',
            left: 'center',
            textStyle: { fontSize: 14, fontWeight: 'normal', color: '#9ca3af' }
        },
        tooltip: {
            position: 'top',
            backgroundColor: 'rgba(0,0,0,0.8)',
            textStyle: { color: '#fff' },
            formatter: function(params) {
                return `坐标: (${params.data[0].toFixed(0)}, ${params.data[1].toFixed(0)})<br/>强度: ${(params.data[2] * 100).toFixed(1)}%`;
            }
        },
        grid: {
            height: '70%',
            top: '15%'
        },
        xAxis: {
            type: 'value',
            min: 0,
            max: 800,
            splitNumber: 10,
            axisLine: { lineStyle: { color: '#9ca3af' } },
            axisLabel: { color: '#9ca3af' }
        },
        yAxis: {
            type: 'value',
            min: 0,
            max: 450,
            splitNumber: 10,
            axisLine: { lineStyle: { color: '#9ca3af' } },
            axisLabel: { color: '#9ca3af' }
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
            },
            textStyle: { color: '#9ca3af' }
        },
        series: [{
            name: '点击强度',
            type: 'heatmap',
            data: heatmapData,
            label: { show: false },
            emphasis: {
                itemStyle: {
                    shadowBlur: 10,
                    shadowColor: 'rgba(0, 0, 0, 0.5)'
                }
            },
            blurSize: 20,
            pointSize: 30
        }]
    };

    heatmapEcharts.setOption(option);
}

function draw3DView(trajectory) {
    if (!scatter3DEcharts || !trajectory || !trajectory.points) return;

    const points = trajectory.points;
    const data = points.map((p, i) => [p.x, p.y, p.velocity || i * 0.1]);

    const option = {
        backgroundColor: 'transparent',
        title: {
            text: '3D轨迹视图',
            left: 'center',
            textStyle: { fontSize: 14, fontWeight: 'normal', color: '#9ca3af' }
        },
        tooltip: {
            backgroundColor: 'rgba(0,0,0,0.8)',
            textStyle: { color: '#fff' },
            formatter: function(params) {
                const idx = params.dataIndex;
                const point = points[idx];
                return `点 ${idx}<br/>X: ${point.x.toFixed(0)}<br/>Y: ${point.y.toFixed(0)}<br/>速度: ${(point.velocity || 0).toFixed(2)}`;
            }
        },
        visualMap: {
            show: true,
            dimension: 2,
            min: 0,
            max: 15,
            inRange: {
                color: ['#3b82f6', '#10b981', '#f59e0b', '#ef4444']
            },
            textStyle: { color: '#9ca3af' }
        },
        xAxis3D: {
            type: 'value',
            name: 'X',
            nameTextStyle: { color: '#9ca3af' },
            axisLine: { lineStyle: { color: '#9ca3af' } },
            axisLabel: { color: '#9ca3af' }
        },
        yAxis3D: {
            type: 'value',
            name: 'Y',
            nameTextStyle: { color: '#9ca3af' },
            axisLine: { lineStyle: { color: '#9ca3af' } },
            axisLabel: { color: '#9ca3af' }
        },
        zAxis3D: {
            type: 'value',
            name: '速度',
            nameTextStyle: { color: '#9ca3af' },
            axisLine: { lineStyle: { color: '#9ca3af' } },
            axisLabel: { color: '#9ca3af' }
        },
        grid3D: {
            viewControl: {
                autoRotate: true,
                autoRotateSpeed: 3
            },
            light: {
                main: {
                    intensity: 1.2,
                    shadow: true
                },
                ambient: {
                    intensity: 0.3
                }
            }
        },
        series: [{
            type: 'scatter3D',
            symbolSize: 8,
            data: data,
            itemStyle: {
                opacity: 0.8
            }
        }, {
            type: 'line3D',
            data: data,
            lineStyle: {
                width: 3,
                color: '#3b82f6'
            }
        }]
    };

    scatter3DEcharts.setOption(option);
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
                trajectoryCtx.arc(normalized.x, normalized.y, 10, 0, Math.PI * 2);
                trajectoryCtx.fill();
                trajectoryCtx.strokeStyle = '#ffffff';
                trajectoryCtx.lineWidth = 3;
                trajectoryCtx.stroke();
            }
        }

        const currentPoint = points[currentIndex];
        const currentNormalized = normalizePoint(currentPoint, trajectoryCanvas.width, trajectoryCanvas.height);
        trajectoryCtx.fillStyle = '#fbbf24';
        trajectoryCtx.beginPath();
        trajectoryCtx.arc(currentNormalized.x, currentNormalized.y, 8, 0, Math.PI * 2);
        trajectoryCtx.fill();

        if (currentIndex >= points.length - 1) {
            isPlaying = false;
            const btn = document.getElementById('playTrajectoryBtn');
            if (btn) btn.innerHTML = '<i class="fas fa-play"></i>';
            return;
        }

        currentIndex++;
        animationFrameId = requestAnimationFrame(() => {
            setTimeout(animate, 50 / playSpeed);
        });
    }

    animate();
}

function updatePlaybackSpeed(e) {
    playSpeed = parseFloat(e.target.value);
    document.getElementById('speedValue').textContent = playSpeed.toFixed(1) + 'x';
}

function showPrevTrajectory() {
    if (!trajectoryData || trajectoryData.trajectories.length === 0) return;

    stopPlayback();
    currentTrajectoryIndex = (currentTrajectoryIndex - 1 + trajectoryData.trajectories.length) % trajectoryData.trajectories.length;
    document.getElementById('trajectorySelector').value = currentTrajectoryIndex;
    selectedPoints = [];
    drawTrajectory(trajectoryData.trajectories[currentTrajectoryIndex], false);
    drawFeatureCharts(trajectoryData.trajectories[currentTrajectoryIndex]);
    generateHeatmapData(trajectoryData.trajectories[currentTrajectoryIndex]);
    draw3DView(trajectoryData.trajectories[currentTrajectoryIndex]);
}

function showNextTrajectory() {
    if (!trajectoryData || trajectoryData.trajectories.length === 0) return;

    stopPlayback();
    currentTrajectoryIndex = (currentTrajectoryIndex + 1) % trajectoryData.trajectories.length;
    document.getElementById('trajectorySelector').value = currentTrajectoryIndex;
    selectedPoints = [];
    drawTrajectory(trajectoryData.trajectories[currentTrajectoryIndex], false);
    drawFeatureCharts(trajectoryData.trajectories[currentTrajectoryIndex]);
    generateHeatmapData(trajectoryData.trajectories[currentTrajectoryIndex]);
    draw3DView(trajectoryData.trajectories[currentTrajectoryIndex]);
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
        drawFeaturesComparison();
    }
}

function drawComparisonChart() {
    if (!comparisonEcharts || trajectoryData.trajectories.length < 2) return;

    const seriesData = trajectoryData.trajectories.slice(0, 3).map((traj, index) => {
        const speeds = calculateSpeeds(traj.points);
        const sampledSpeeds = [];
        const step = Math.max(1, Math.floor(speeds.length / 50));
        for (let i = 0; i < speeds.length; i += step) {
            sampledSpeeds.push(speeds[i]);
        }
        return {
            name: `${traj.userId.substring(0, 8)} (风险: ${traj.features?.riskScore?.toFixed(0) || 'N/A'})`,
            type: 'line',
            smooth: true,
            data: sampledSpeeds,
            lineStyle: { width: 2 },
            itemStyle: {},
            symbol: 'circle',
            symbolSize: 4
        };
    });

    const colors = ['#3b82f6', '#10b981', '#f59e0b'];
    seriesData.forEach((s, i) => {
        s.lineStyle.color = colors[i];
        s.itemStyle.color = colors[i];
    });

    const option = {
        backgroundColor: 'transparent',
        title: {
            text: '轨迹速度对比',
            left: 'center',
            textStyle: { fontSize: 14, fontWeight: 'normal', color: '#9ca3af' }
        },
        tooltip: {
            trigger: 'axis',
            backgroundColor: 'rgba(0,0,0,0.8)',
            textStyle: { color: '#fff' }
        },
        legend: {
            data: seriesData.map(s => s.name),
            bottom: 0,
            textStyle: { color: '#9ca3af' }
        },
        grid: {
            left: '3%',
            right: '4%',
            bottom: '15%',
            top: '10%',
            containLabel: true
        },
        xAxis: {
            type: 'category',
            boundaryGap: false,
            data: Array.from({length: 50}, (_, i) => i),
            axisLabel: { color: '#9ca3af', fontSize: 10 }
        },
        yAxis: {
            type: 'value',
            name: '速度',
            nameTextStyle: { color: '#9ca3af' },
            axisLabel: { color: '#9ca3af' },
            splitLine: { lineStyle: { color: 'rgba(255,255,255,0.1)' } }
        },
        series: seriesData
    };

    comparisonEcharts.setOption(option);
}

function drawFeaturesComparison() {
    const container = document.getElementById('featuresComparison');
    if (!container) return;

    const features = ['avgSpeed', 'maxSpeed', 'avgAcceleration', 'maxAcceleration', 'smoothness', 'riskScore'];
    const featureNames = {
        avgSpeed: '平均速度',
        maxSpeed: '最大速度',
        avgAcceleration: '平均加速度',
        maxAcceleration: '最大加速度',
        smoothness: '平滑度',
        riskScore: '风险评分'
    };

    let html = '<table class="comparison-table"><thead><tr><th>特征</th>';
    trajectoryData.trajectories.slice(0, 3).forEach((traj, i) => {
        html += `<th>${traj.userId.substring(0, 8)}</th>`;
    });
    html += '</tr></thead><tbody>';

    features.forEach(feature => {
        html += `<tr><td>${featureNames[feature]}</td>`;
        trajectoryData.trajectories.slice(0, 3).forEach(traj => {
            const value = traj.features?.[feature];
            let formatted = value !== undefined ? value.toFixed(2) : '-';
            if (feature === 'riskScore' && value >= 70) {
                html += `<td class="text-danger">${formatted}</td>`;
            } else if (feature === 'smoothness' && value < 0.7) {
                html += `<td class="text-warning">${formatted}</td>`;
            } else {
                html += `<td>${formatted}</td>`;
            }
        });
        html += '</tr>';
    });

    html += '</tbody></table>';
    container.innerHTML = html;
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
        exportedAt: new Date().toISOString(),
        analysis: analyzeTrajectoryData(trajectory)
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

function analyzeTrajectoryData(trajectory) {
    const points = trajectory.points;
    if (!points || points.length < 2) return {};

    let totalDistance = 0;
    let totalTime = 0;
    let maxSpeed = 0;
    let pauseCount = 0;
    
    for (let i = 1; i < points.length; i++) {
        const prev = points[i - 1];
        const curr = points[i];
        const dx = curr.x - prev.x;
        const dy = curr.y - prev.y;
        totalDistance += Math.sqrt(dx * dx + dy * dy);
        totalTime += curr.timestamp - prev.timestamp;
        const time = (curr.timestamp - prev.timestamp) / 1000;
        if (time > 0) {
            const speed = Math.sqrt(dx * dx + dy * dy) / time;
            maxSpeed = Math.max(maxSpeed, speed);
        }
        if (curr.event === 'pause') pauseCount++;
    }

    const start = points[0];
    const end = points[points.length - 1];
    const directDistance = Math.sqrt(Math.pow(end.x - start.x, 2) + Math.pow(end.y - start.y, 2));

    return {
        totalPoints: points.length,
        totalDistance: totalDistance.toFixed(2),
        directDistance: directDistance.toFixed(2),
        pathEfficiency: ((directDistance / totalDistance) * 100).toFixed(1) + '%',
        avgSpeed: (totalDistance / (totalTime / 1000)).toFixed(2),
        maxSpeed: maxSpeed.toFixed(2),
        pauseCount: pauseCount,
        duration: (totalTime / 1000).toFixed(2) + 's'
    };
}

function clearTrajectoryCanvas() {
    if (!trajectoryCtx) return;
    trajectoryCtx.clearRect(0, 0, trajectoryCanvas.width, trajectoryCanvas.height);
    drawTrajectoryGrid(trajectoryCtx);
}

function resetTrajectoryView() {
    stopPlayback();
    currentTrajectoryIndex = 0;
    selectedPoints = [];
    if (trajectoryData && trajectoryData.trajectories.length > 0) {
        drawTrajectory(trajectoryData.trajectories[0], false);
        drawFeatureCharts(trajectoryData.trajectories[0]);
        generateHeatmapData(trajectoryData.trajectories[0]);
        draw3DView(trajectoryData.trajectories[0]);
    } else {
        clearTrajectoryCanvas();
    }
}

function handleCanvasClick(e) {
    if (!trajectoryData || !trajectoryCanvas) return;

    const rect = trajectoryCanvas.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;

    const trajectory = trajectoryData.trajectories[currentTrajectoryIndex];
    if (!trajectory || !trajectory.points) return;

    let closestPointIndex = -1;
    let minDistance = 20;

    trajectory.points.forEach((point, index) => {
        const normalized = normalizePoint(point, trajectoryCanvas.width, trajectoryCanvas.height);
        const distance = Math.sqrt(Math.pow(x - normalized.x, 2) + Math.pow(y - normalized.y, 2));
        if (distance < minDistance) {
            minDistance = distance;
            closestPointIndex = index;
        }
    });

    if (closestPointIndex >= 0) {
        const idx = selectedPoints.indexOf(closestPointIndex);
        if (idx >= 0) {
            selectedPoints.splice(idx, 1);
        } else {
            selectedPoints.push(closestPointIndex);
        }
        drawTrajectory(trajectory, false);
        showPointDetail(trajectory.points[closestPointIndex], closestPointIndex);
    }
}

function handleCanvasHover(e) {
    if (!trajectoryData || !trajectoryCanvas) return;

    const rect = trajectoryCanvas.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;

    const trajectory = trajectoryData.trajectories[currentTrajectoryIndex];
    if (!trajectory || !trajectory.points) return;

    let closestPointIndex = -1;
    let minDistance = 15;

    trajectory.points.forEach((point, index) => {
        const normalized = normalizePoint(point, trajectoryCanvas.width, trajectoryCanvas.height);
        const distance = Math.sqrt(Math.pow(x - normalized.x, 2) + Math.pow(y - normalized.y, 2));
        if (distance < minDistance) {
            minDistance = distance;
            closestPointIndex = index;
        }
    });

    trajectoryCanvas.style.cursor = closestPointIndex >= 0 ? 'pointer' : 'default';
}

function showPointDetail(point, index) {
    const detailPanel = document.getElementById('pointDetailPanel');
    if (!detailPanel) return;

    detailPanel.innerHTML = `
        <div class="point-detail-header">
            <h4>点 ${index} 详情</h4>
            <button onclick="document.getElementById('pointDetailPanel').innerHTML = ''">&times;</button>
        </div>
        <div class="point-detail-body">
            <div>X坐标: ${point.x.toFixed(2)}</div>
            <div>Y坐标: ${point.y.toFixed(2)}</div>
            <div>时间戳: ${new Date(point.timestamp).toLocaleTimeString()}</div>
            <div>事件类型: ${point.event}</div>
            <div>速度: ${(point.velocity || 0).toFixed(2)} px/ms</div>
            <div>加速度: ${(point.acceleration || 0).toFixed(2)} px/ms²</div>
            ${point.pressure !== undefined ? `<div>压力: ${point.pressure.toFixed(2)}</div>` : ''}
            ${point.tiltX !== undefined ? `<div>倾斜X: ${point.tiltX.toFixed(1)}°</div>` : ''}
            ${point.tiltY !== undefined ? `<div>倾斜Y: ${point.tiltY.toFixed(1)}°</div>` : ''}
        </div>
    `;
}

function clearSelection() {
    selectedPoints = [];
    drawTrajectory(trajectoryData.trajectories[currentTrajectoryIndex], false);
}

function analyzeTrajectory() {
    if (!trajectoryData) return;

    const trajectory = trajectoryData.trajectories[currentTrajectoryIndex];
    const analysis = analyzeTrajectoryData(trajectory);
    
    const analysisModal = document.getElementById('analysisModal');
    if (analysisModal) {
        analysisModal.style.display = 'flex';
        document.getElementById('analysisContent').innerHTML = `
            <h3>轨迹分析报告</h3>
            <div class="analysis-grid">
                <div class="analysis-item">
                    <span class="analysis-label">总点数</span>
                    <span class="analysis-value">${analysis.totalPoints}</span>
                </div>
                <div class="analysis-item">
                    <span class="analysis-label">总距离</span>
                    <span class="analysis-value">${analysis.totalDistance} px</span>
                </div>
                <div class="analysis-item">
                    <span class="analysis-label">直线距离</span>
                    <span class="analysis-value">${analysis.directDistance} px</span>
                </div>
                <div class="analysis-item">
                    <span class="analysis-label">路径效率</span>
                    <span class="analysis-value ${parseFloat(analysis.pathEfficiency) < 60 ? 'text-warning' : ''}">${analysis.pathEfficiency}</span>
                </div>
                <div class="analysis-item">
                    <span class="analysis-label">平均速度</span>
                    <span class="analysis-value">${analysis.avgSpeed} px/ms</span>
                </div>
                <div class="analysis-item">
                    <span class="analysis-label">最大速度</span>
                    <span class="analysis-value ${parseFloat(analysis.maxSpeed) > 10 ? 'text-danger' : ''}">${analysis.maxSpeed} px/ms</span>
                </div>
                <div class="analysis-item">
                    <span class="analysis-label">停顿次数</span>
                    <span class="analysis-value">${analysis.pauseCount}</span>
                </div>
                <div class="analysis-item">
                    <span class="analysis-label">持续时间</span>
                    <span class="analysis-value">${analysis.duration}</span>
                </div>
            </div>
            <div class="analysis-summary">
                <h4>风险评估</h4>
                <p>风险评分: <strong class="${getRiskClass(trajectory.features?.riskScore || 0)}">${trajectory.features?.riskScore?.toFixed(1) || 'N/A'}</strong></p>
                <p>疑似真人: <strong>${trajectory.features?.isHuman ? '是' : '否'}</strong></p>
                <p>异常分数: <strong>${(trajectory.features?.anomalyScore * 100 || 0).toFixed(0)}%</strong></p>
            </div>
        `;
    }
}

function closeAnalysisModal() {
    const modal = document.getElementById('analysisModal');
    if (modal) modal.style.display = 'none';
}

function toggleHeatmap() {
    const panel = document.getElementById('heatmapPanel');
    if (panel) {
        panel.style.display = panel.style.display === 'none' ? 'block' : 'none';
    }
}

function toggle3DView() {
    const panel = document.getElementById('scatter3DPanel');
    if (panel) {
        panel.style.display = panel.style.display === 'none' ? 'block' : 'none';
    }
}