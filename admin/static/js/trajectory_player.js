let playerCharts = {};
let trajectoryData = [];
let currentTrajectoryIndex = 0;
let isPlaying = false;
let isPaused = false;
let playbackSpeed = 1;
let animationFrameId = null;
let playbackProgress = 0;
let timelineMarkers = [];

const TIMELINE_UPDATE_INTERVAL = 50;
const DEFAULT_POINT_DISPLAY_DURATION = 30;

document.addEventListener('DOMContentLoaded', () => {
    initializeTrajectoryPlayer();
    setupEventListeners();
    loadTrajectoryData();
});

function initializeTrajectoryPlayer() {
    initPlayerChart();
    initSpeedChart();
    initTimelineChart();
    initFeatureAnalysisChart();
}

function initPlayerChart() {
    const container = document.getElementById('trajectoryPlayerChart');
    if (!container) return;

    playerCharts.main = echarts.init(container);
    window.addEventListener('resize', () => playerCharts.main.resize());
}

function initSpeedChart() {
    const container = document.getElementById('playerSpeedChart');
    if (!container) return;

    playerCharts.speed = echarts.init(container);
    window.addEventListener('resize', () => playerCharts.speed.resize());
}

function initTimelineChart() {
    const container = document.getElementById('playerTimelineChart');
    if (!container) return;

    playerCharts.timeline = echarts.init(container);
    window.addEventListener('resize', () => playerCharts.timeline.resize());
}

function initFeatureAnalysisChart() {
    const container = document.getElementById('playerFeatureChart');
    if (!container) return;

    playerCharts.feature = echarts.init(container);
    window.addEventListener('resize', () => playerCharts.feature.resize());
}

function setupEventListeners() {
    document.getElementById('playPauseBtn')?.addEventListener('click', togglePlayPause);
    document.getElementById('stopBtn')?.addEventListener('click', stopPlayback);
    document.getElementById('prevBtn')?.addEventListener('click', showPrevious);
    document.getElementById('nextBtn')?.addEventListener('click', showNext);
    document.getElementById('speedSlider')?.addEventListener('input', updatePlaybackSpeed);
    document.getElementById('trajectorySelector')?.addEventListener('change', selectTrajectory);
    document.getElementById('timelineSlider')?.addEventListener('input', seekTimeline);
    document.getElementById('exportBtn')?.addEventListener('click', exportTrajectory);
    document.getElementById('loopBtn')?.addEventListener('click', toggleLoop);
    document.getElementById('showGridBtn')?.addEventListener('click', toggleGrid);
}

async function loadTrajectoryData() {
    try {
        const response = await fetch('/admin/behavior-analytics');
        const result = await response.json();
        if (result.code === 0 && result.data.trajectories) {
            trajectoryData = result.data.trajectories;
            initializePlayer();
        } else {
            loadDemoData();
        }
    } catch (error) {
        console.error('加载轨迹数据失败:', error);
        loadDemoData();
    }
}

function loadDemoData() {
    trajectoryData = [
        {
            id: 'demo_001',
            userId: 'user_demo',
            sessionId: 'session_demo',
            points: generateDemoPoints(150),
            features: {
                avgSpeed: 2.8,
                maxSpeed: 8.5,
                avgAcceleration: 0.35,
                smoothness: 0.87,
                totalDistance: 1350,
                pauseCount: 2
            }
        },
        {
            id: 'demo_002',
            userId: 'user_demo2',
            sessionId: 'session_demo2',
            points: generateDemoPoints(200),
            features: {
                avgSpeed: 3.2,
                maxSpeed: 9.2,
                avgAcceleration: 0.42,
                smoothness: 0.82,
                totalDistance: 1580,
                pauseCount: 4
            }
        }
    ];
    initializePlayer();
}

function generateDemoPoints(count) {
    const points = [];
    let x = 100, y = 200;
    const targetX = 700, targetY = 350;
    let timestamp = Date.now();

    for (let i = 0; i < count; i++) {
        const progress = i / count;
        const targetXPos = x + (targetX - x) * progress + (Math.random() - 0.5) * 20;
        const targetYPos = y + (targetY - y) * progress + (Math.random() - 0.5) * 20;
        x = targetXPos;
        y = targetYPos;

        points.push({
            x: x,
            y: y,
            timestamp: timestamp,
            event: i % 20 === 0 ? 'click' : 'move',
            velocity: 1 + Math.random() * 3,
            acceleration: (Math.random() - 0.5) * 0.5
        });

        timestamp += 15 + Math.random() * 20;
    }

    return points;
}

function initializePlayer() {
    updateTrajectorySelector();
    if (trajectoryData.length > 0) {
        loadTrajectory(0);
    }
}

function updateTrajectorySelector() {
    const selector = document.getElementById('trajectorySelector');
    if (!selector) return;

    selector.innerHTML = '';
    trajectoryData.forEach((traj, index) => {
        const option = document.createElement('option');
        option.value = index;
        option.textContent = `${traj.userId} - ${traj.sessionId.substring(0, 8)}`;
        selector.appendChild(option);
    });
}

function selectTrajectory(e) {
    const index = parseInt(e.target.value);
    if (!isNaN(index) && trajectoryData[index]) {
        stopPlayback();
        loadTrajectory(index);
    }
}

function loadTrajectory(index) {
    currentTrajectoryIndex = index;
    const trajectory = trajectoryData[index];
    if (!trajectory) return;

    playbackProgress = 0;
    updateProgressDisplay(0);
    updateFeatureDisplay(trajectory.features);
    drawMainChart(trajectory);
    drawSpeedChart(trajectory);
    drawTimelineChart(trajectory);
    drawFeatureAnalysisChart(trajectory);

    document.getElementById('trajectorySelector').value = index;
    updateMarkerInfo(trajectory);
}

function drawMainChart(trajectory) {
    if (!playerCharts.main) return;

    const points = trajectory.points;
    const pathData = [];
    const clickPoints = [];

    points.forEach((point, index) => {
        pathData.push([point.x, point.y]);

        if (point.event === 'click') {
            clickPoints.push({
                coord: [point.x, point.y],
                value: index,
                symbol: 'circle',
                symbolSize: 12,
                itemStyle: {
                    color: '#ef4444',
                    borderColor: '#fff',
                    borderWidth: 2
                }
            });
        }
    });

    const option = {
        title: {
            text: '轨迹回放',
            subtext: `轨迹 ${currentTrajectoryIndex + 1} / ${trajectoryData.length}`,
            left: 'center'
        },
        tooltip: {
            trigger: 'item',
            formatter: function(params) {
                if (params.data && params.data.value !== undefined) {
                    return `点击事件 #${params.data.value}<br/>坐标: (${params.data.coord[0]}, ${params.data.coord[1]})`;
                }
                return `坐标: (${params.data[0]}, ${params.data[1]})`;
            }
        },
        grid: {
            left: '3%',
            right: '4%',
            top: '15%',
            bottom: '10%',
            containLabel: true
        },
        xAxis: {
            type: 'value',
            min: 0,
            max: 800,
            axisLabel: {
                formatter: '{value}px'
            }
        },
        yAxis: {
            type: 'value',
            min: 0,
            max: 400,
            axisLabel: {
                formatter: '{value}px'
            }
        },
        series: [
            {
                name: '轨迹路径',
                type: 'line',
                data: pathData,
                smooth: true,
                lineStyle: {
                    color: '#3b82f6',
                    width: 2
                },
                showSymbol: false,
                areaStyle: {
                    color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                        { offset: 0, color: 'rgba(59, 130, 246, 0.3)' },
                        { offset: 1, color: 'rgba(59, 130, 246, 0.05)' }
                    ])
                }
            },
            {
                name: '点击点',
                type: 'scatter',
                data: clickPoints,
                zlevel: 2
            }
        ]
    };

    playerCharts.main.setOption(option, true);
}

function drawSpeedChart(trajectory) {
    if (!playerCharts.speed) return;

    const points = trajectory.points;
    const speeds = [];
    const timestamps = [];

    for (let i = 1; i < points.length; i++) {
        const prev = points[i - 1];
        const curr = points[i];
        const dx = curr.x - prev.x;
        const dy = curr.y - prev.y;
        const distance = Math.sqrt(dx * dx + dy * dy);
        const time = (curr.timestamp - prev.timestamp) / 1000;
        const speed = time > 0 ? distance / time : 0;

        speeds.push(speed.toFixed(2));
        timestamps.push(i);
    }

    const option = {
        title: {
            text: '速度变化',
            left: 'center',
            textStyle: { fontSize: 14, fontWeight: 'normal' }
        },
        tooltip: {
            trigger: 'axis'
        },
        grid: {
            left: '3%',
            right: '4%',
            bottom: '10%',
            top: '20%',
            containLabel: true
        },
        xAxis: {
            type: 'category',
            data: timestamps,
            axisLabel: {
                interval: Math.floor(timestamps.length / 10)
            }
        },
        yAxis: {
            type: 'value',
            name: '速度 (px/ms)'
        },
        series: [{
            name: '速度',
            type: 'line',
            data: speeds,
            smooth: true,
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

    playerCharts.speed.setOption(option);
}

function drawTimelineChart(trajectory) {
    if (!playerCharts.timeline) return;

    const points = trajectory.points;
    const totalDuration = points[points.length - 1].timestamp - points[0].timestamp;

    const option = {
        title: {
            text: '时间轴',
            left: 'center',
            textStyle: { fontSize: 14, fontWeight: 'normal' }
        },
        tooltip: {
            trigger: 'axis',
            formatter: function(params) {
                const index = params[0].dataIndex;
                const point = points[index];
                const time = ((point.timestamp - points[0].timestamp) / 1000).toFixed(2);
                return `时间: ${time}s<br/>事件: ${point.event}<br/>坐标: (${point.x}, ${point.y})`;
            }
        },
        grid: {
            left: '3%',
            right: '4%',
            bottom: '15%',
            top: '20%',
            containLabel: true
        },
        xAxis: {
            type: 'category',
            data: points.map((_, i) => i),
            axisLabel: {
                interval: Math.floor(points.length / 10)
            }
        },
        yAxis: {
            type: 'value',
            show: false
        },
        series: [{
            name: '时间轴',
            type: 'bar',
            data: points.map(p => p.event === 'click' ? 1 : 0.3),
            itemStyle: {
                color: function(params) {
                    return points[params.dataIndex].event === 'click' ? '#ef4444' : '#3b82f6';
                }
            }
        }]
    };

    playerCharts.timeline.setOption(option);
}

function drawFeatureAnalysisChart(trajectory) {
    if (!playerCharts.feature) return;

    const features = trajectory.features;

    const option = {
        title: {
            text: '特征分析',
            left: 'center',
            textStyle: { fontSize: 14, fontWeight: 'normal' }
        },
        tooltip: {
            trigger: 'item'
        },
        series: [{
            name: '特征值',
            type: 'radar',
            data: [{
                value: [
                    features.avgSpeed,
                    features.maxSpeed,
                    features.avgAcceleration * 10,
                    features.smoothness * 100,
                    features.totalDistance / 20,
                    (100 - features.pauseCount * 10)
                ],
                name: '特征'
            }],
            itemStyle: {
                color: '#3b82f6'
            },
            areaStyle: {
                color: 'rgba(59, 130, 246, 0.3)'
            }
        }],
        radar: {
            indicator: [
                { name: '平均速度', max: 10 },
                { name: '最大速度', max: 15 },
                { name: '加速度', max: 5 },
                { name: '平滑度', max: 100 },
                { name: '总距离', max: 100 },
                { name: '稳定性', max: 100 }
            ],
            center: ['50%', '55%'],
            radius: '65%'
        }
    };

    playerCharts.feature.setOption(option);
}

function updateFeatureDisplay(features) {
    document.getElementById('playerAvgSpeed') && (document.getElementById('playerAvgSpeed').textContent = features.avgSpeed.toFixed(2));
    document.getElementById('playerMaxSpeed') && (document.getElementById('playerMaxSpeed').textContent = features.maxSpeed.toFixed(2));
    document.getElementById('playerSmoothness') && (document.getElementById('playerSmoothness').textContent = features.smoothness.toFixed(2));
    document.getElementById('playerDistance') && (document.getElementById('playerDistance').textContent = features.totalDistance.toFixed(0));
    document.getElementById('playerPauses') && (document.getElementById('playerPauses').textContent = features.pauseCount);
}

function togglePlayPause() {
    if (isPlaying && !isPaused) {
        pausePlayback();
    } else if (isPlaying && isPaused) {
        resumePlayback();
    } else {
        startPlayback();
    }
}

function startPlayback() {
    if (trajectoryData.length === 0) return;

    isPlaying = true;
    isPaused = false;
    updatePlayPauseButton();

    const trajectory = trajectoryData[currentTrajectoryIndex];
    const points = trajectory.points;
    let currentIndex = 0;

    function animate() {
        if (!isPlaying || isPaused) return;

        currentIndex++;
        playbackProgress = currentIndex / points.length;

        updateProgressDisplay(playbackProgress);
        highlightCurrentPoint(currentIndex);

        if (currentIndex >= points.length) {
            isPlaying = false;
            updatePlayPauseButton();
            return;
        }

        animationFrameId = requestAnimationFrame(() => {
            setTimeout(animate, DEFAULT_POINT_DISPLAY_DURATION / playbackSpeed);
        });
    }

    animate();
}

function pausePlayback() {
    isPaused = true;
    updatePlayPauseButton();
}

function resumePlayback() {
    isPaused = false;
    updatePlayPauseButton();
    startPlayback();
}

function stopPlayback() {
    isPlaying = false;
    isPaused = false;
    playbackProgress = 0;
    updateProgressDisplay(0);
    updatePlayPauseButton();

    if (animationFrameId) {
        cancelAnimationFrame(animationFrameId);
        animationFrameId = null;
    }

    if (trajectoryData[currentTrajectoryIndex]) {
        loadTrajectory(currentTrajectoryIndex);
    }
}

function showPrevious() {
    stopPlayback();
    if (trajectoryData.length === 0) return;

    currentTrajectoryIndex = (currentTrajectoryIndex - 1 + trajectoryData.length) % trajectoryData.length;
    loadTrajectory(currentTrajectoryIndex);
}

function showNext() {
    stopPlayback();
    if (trajectoryData.length === 0) return;

    currentTrajectoryIndex = (currentTrajectoryIndex + 1) % trajectoryData.length;
    loadTrajectory(currentTrajectoryIndex);
}

function updatePlaybackSpeed(e) {
    playbackSpeed = parseFloat(e.target.value);
    document.getElementById('speedValue').textContent = playbackSpeed.toFixed(1) + 'x';
}

function seekTimeline(e) {
    const progress = parseFloat(e.target.value) / 100;
    playbackProgress = progress;
    updateProgressDisplay(progress);

    const trajectory = trajectoryData[currentTrajectoryIndex];
    if (!trajectory) return;

    const currentIndex = Math.floor(progress * trajectory.points.length);
    highlightCurrentPoint(currentIndex);
}

function highlightCurrentPoint(index) {
    if (!playerCharts.main) return;

    const option = playerCharts.main.getOption();
    const trajectory = trajectoryData[currentTrajectoryIndex];
    if (!trajectory) return;

    const currentPoint = trajectory.points[index];
    if (!currentPoint) return;

    option.series.push({
        name: '当前位置',
        type: 'scatter',
        data: [[currentPoint.x, currentPoint.y]],
        symbolSize: 20,
        itemStyle: {
            color: '#10b981',
            borderColor: '#fff',
            borderWidth: 3,
            shadowBlur: 10,
            shadowColor: 'rgba(16, 185, 129, 0.8)'
        },
        zlevel: 3
    });

    playerCharts.main.setOption(option);
}

function updateProgressDisplay(progress) {
    const progressBar = document.getElementById('playbackProgressBar');
    if (progressBar) {
        progressBar.style.width = (progress * 100) + '%';
    }

    const slider = document.getElementById('timelineSlider');
    if (slider) {
        slider.value = progress * 100;
    }

    const percentage = document.getElementById('progressPercentage');
    if (percentage) {
        percentage.textContent = (progress * 100).toFixed(0) + '%';
    }

    const trajectory = trajectoryData[currentTrajectoryIndex];
    if (trajectory && trajectory.points.length > 0) {
        const currentTime = (progress * (trajectory.points[trajectory.points.length - 1].timestamp - trajectory.points[0].timestamp) / 1000).toFixed(2);
        const timeDisplay = document.getElementById('currentTimeDisplay');
        if (timeDisplay) {
            timeDisplay.textContent = currentTime + 's';
        }
    }
}

function updatePlayPauseButton() {
    const btn = document.getElementById('playPauseBtn');
    if (!btn) return;

    if (isPlaying && !isPaused) {
        btn.innerHTML = '<i class="fas fa-pause"></i>';
    } else {
        btn.innerHTML = '<i class="fas fa-play"></i>';
    }
}

function toggleLoop() {
    const btn = document.getElementById('loopBtn');
    if (!btn) return;

    btn.classList.toggle('active');
    const isLooping = btn.classList.contains('active');

    if (isLooping && !isPlaying) {
        stopPlayback();
    }
}

function toggleGrid() {
    if (!playerCharts.main) return;

    const option = playerCharts.main.getOption();
    if (!option.grid) {
        option.grid = {
            show: true,
            borderWidth: 1,
            borderColor: '#e5e7eb'
        };
    } else {
        delete option.grid;
    }

    playerCharts.main.setOption(option);
}

function updateMarkerInfo(trajectory) {
    timelineMarkers = [];

    trajectory.points.forEach((point, index) => {
        if (point.event === 'click') {
            timelineMarkers.push({
                index: index,
                x: point.x,
                y: point.y,
                timestamp: point.timestamp
            });
        }
    });

    const markerList = document.getElementById('markerList');
    if (markerList) {
        markerList.innerHTML = timelineMarkers.map((marker, i) =>
            `<div class="marker-item" onclick="seekToMarker(${marker.index})">点击 #${i + 1}</div>`
        ).join('');
    }
}

function seekToMarker(index) {
    const trajectory = trajectoryData[currentTrajectoryIndex];
    if (!trajectory) return;

    playbackProgress = index / trajectory.points.length;
    updateProgressDisplay(playbackProgress);
    highlightCurrentPoint(index);
}

function exportTrajectory() {
    if (trajectoryData.length === 0) return;

    const trajectory = trajectoryData[currentTrajectoryIndex];
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

    showToast('轨迹导出成功', 'success');
}

function showToast(message, type = 'info') {
    const toast = document.createElement('div');
    toast.className = `alert alert-${type} alert-dismissible fade show position-fixed`;
    toast.style.cssText = 'top: 20px; right: 20px; z-index: 9999; min-width: 250px;';
    toast.innerHTML = `
        ${message}
        <button type="button" class="close" data-dismiss="alert" aria-label="Close">
            <span aria-hidden="true">&times;</span>
        </button>
    `;
    document.body.appendChild(toast);

    setTimeout(() => {
        toast.remove();
    }, 3000);
}

window.addEventListener('beforeunload', () => {
    Object.values(playerCharts).forEach(chart => {
        if (chart) chart.dispose();
    });
    if (animationFrameId) {
        cancelAnimationFrame(animationFrameId);
    }
});
