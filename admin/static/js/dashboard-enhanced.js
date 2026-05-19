let enhancedCharts = {};
let realtimeDataBuffers = {};
let chartUpdateTimers = {};
const CHART_UPDATE_INTERVAL = 3000;
const MAX_DATA_POINTS = 100;
let wsConnection = null;
let isConnected = false;

document.addEventListener('DOMContentLoaded', () => {
    initializeEnhancedCharts();
    initWebSocketConnection();
    setupChartInteractions();
    startRealtimeUpdates();
});

function initializeEnhancedCharts() {
    initRequestDistributionChart();
    initResponseTimeChart();
    initUserActivityChart();
    initGeographicChart();
    initCaptchaSuccessRateChart();
    initAnomalyDetectionChart();
    initTrendForecastChart();
    initSessionDurationChart();
}

function initRequestDistributionChart() {
    const container = document.getElementById('requestDistributionChart');
    if (!container) return;

    enhancedCharts.requestDistribution = echarts.init(container);
    window.addEventListener('resize', () => enhancedCharts.requestDistribution.resize());

    const option = {
        title: {
            text: '请求分布',
            left: 'center',
            textStyle: { fontSize: 14, fontWeight: 'normal' }
        },
        tooltip: {
            trigger: 'item',
            formatter: '{b}: {c} ({d}%)'
        },
        legend: {
            orient: 'vertical',
            left: 'left',
            top: 'middle'
        },
        series: [{
            name: '请求类型',
            type: 'pie',
            radius: ['40%', '70%'],
            center: ['60%', '50%'],
            avoidLabelOverlap: false,
            itemStyle: {
                borderRadius: 10,
                borderColor: '#fff',
                borderWidth: 2
            },
            label: {
                show: true,
                formatter: '{d}%'
            },
            emphasis: {
                label: {
                    show: true,
                    fontSize: 16,
                    fontWeight: 'bold'
                }
            },
            data: [
                { value: 335, name: '滑动验证', itemStyle: { color: '#3b82f6' } },
                { value: 234, name: '点选验证', itemStyle: { color: '#10b981' } },
                { value: 154, name: '图形验证', itemStyle: { color: '#f59e0b' } },
                { value: 135, name: '语音验证', itemStyle: { color: '#8b5cf6' } },
                { value: 148, name: '3D验证', itemStyle: { color: '#ec4899' } }
            ]
        }]
    };

    enhancedCharts.requestDistribution.setOption(option);
    initDataBuffer('requestDistribution');
}

function initResponseTimeChart() {
    const container = document.getElementById('responseTimeChart');
    if (!container) return;

    enhancedCharts.responseTime = echarts.init(container);
    window.addEventListener('resize', () => enhancedCharts.responseTime.resize());

    const hours = Array.from({length: 24}, (_, i) => `${i}:00`);

    const option = {
        title: {
            text: '响应时间分布',
            left: 'center',
            textStyle: { fontSize: 14, fontWeight: 'normal' }
        },
        tooltip: {
            trigger: 'axis',
            axisPointer: { type: 'shadow' }
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
            data: hours,
            axisLabel: { color: '#666', interval: 3, rotate: 45 }
        },
        yAxis: {
            type: 'value',
            name: '响应时间 (ms)',
            axisLabel: { color: '#666' }
        },
        series: [{
            name: '平均响应时间',
            type: 'bar',
            data: generateMockResponseTimeData(),
            itemStyle: {
                color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                    { offset: 0, color: '#3b82f6' },
                    { offset: 1, color: '#1e40af' }
                ])
            },
            barWidth: '60%',
            emphasis: {
                itemStyle: {
                    shadowBlur: 10,
                    shadowColor: 'rgba(0, 0, 0, 0.3)'
                }
            }
        }]
    };

    enhancedCharts.responseTime.setOption(option);
    initDataBuffer('responseTime');
}

function initUserActivityChart() {
    const container = document.getElementById('userActivityChart');
    if (!container) return;

    enhancedCharts.userActivity = echarts.init(container);
    window.addEventListener('resize', () => enhancedCharts.userActivity.resize());

    const days = ['周一', '周二', '周三', '周四', '周五', '周六', '周日'];

    const option = {
        title: {
            text: '用户活动趋势',
            left: 'center',
            textStyle: { fontSize: 14, fontWeight: 'normal' }
        },
        tooltip: {
            trigger: 'axis',
            axisPointer: { type: 'line' }
        },
        legend: {
            data: ['活跃用户', '新注册用户'],
            bottom: '0'
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
            data: days,
            axisLabel: { color: '#666' }
        },
        yAxis: {
            type: 'value',
            axisLabel: { color: '#666' }
        },
        series: [
            {
                name: '活跃用户',
                type: 'line',
                smooth: true,
                data: [320, 432, 501, 434, 590, 730, 620],
                areaStyle: {
                    color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                        { offset: 0, color: 'rgba(59, 130, 246, 0.3)' },
                        { offset: 1, color: 'rgba(59, 130, 246, 0.05)' }
                    ])
                },
                lineStyle: { color: '#3b82f6', width: 2 },
                itemStyle: { color: '#3b82f6' }
            },
            {
                name: '新注册用户',
                type: 'line',
                smooth: true,
                data: [45, 52, 61, 54, 70, 85, 72],
                areaStyle: {
                    color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                        { offset: 0, color: 'rgba(16, 185, 129, 0.3)' },
                        { offset: 1, color: 'rgba(16, 185, 129, 0.05)' }
                    ])
                },
                lineStyle: { color: '#10b981', width: 2 },
                itemStyle: { color: '#10b981' }
            }
        ]
    };

    enhancedCharts.userActivity.setOption(option);
}

function initGeographicChart() {
    const container = document.getElementById('geographicChart');
    if (!container) return;

    enhancedCharts.geographic = echarts.init(container);
    window.addEventListener('resize', () => enhancedCharts.geographic.resize());

    const geoData = [
        { name: '北京', value: 3500 },
        { name: '上海', value: 3200 },
        { name: '广州', value: 2800 },
        { name: '深圳', value: 2600 },
        { name: '杭州', value: 2100 },
        { name: '成都', value: 1800 },
        { name: '武汉', value: 1600 },
        { name: '西安', value: 1400 },
        { name: '南京', value: 1300 },
        { name: '重庆', value: 1200 }
    ];

    const option = {
        title: {
            text: '地域分布 TOP10',
            left: 'center',
            textStyle: { fontSize: 14, fontWeight: 'normal' }
        },
        tooltip: {
            trigger: 'item',
            formatter: '{b}: {c}'
        },
        series: [{
            type: 'bar',
            data: geoData.sort((a, b) => b.value - a.value),
            barWidth: '50%',
            itemStyle: {
                color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [
                    { offset: 0, color: '#3b82f6' },
                    { offset: 1, color: '#10b981' }
                ])
            },
            label: {
                show: true,
                position: 'right',
                formatter: '{c}'
            }
        }],
        grid: {
            left: '3%',
            right: '15%',
            bottom: '3%',
            top: '20%',
            containLabel: true
        },
        xAxis: {
            type: 'value',
            axisLabel: { color: '#666' }
        },
        yAxis: {
            type: 'category',
            data: geoData.sort((a, b) => b.value - a.value).map(d => d.name),
            axisLabel: { color: '#666' }
        }
    };

    enhancedCharts.geographic.setOption(option);
}

function initCaptchaSuccessRateChart() {
    const container = document.getElementById('captchaSuccessRateChart');
    if (!container) return;

    enhancedCharts.captchaSuccessRate = echarts.init(container);
    window.addEventListener('resize', () => enhancedCharts.captchaSuccessRate.resize());

    const hours = Array.from({length: 24}, (_, i) => `${i}:00`);

    const option = {
        title: {
            text: '验证码成功率',
            left: 'center',
            textStyle: { fontSize: 14, fontWeight: 'normal' }
        },
        tooltip: {
            trigger: 'axis',
            formatter: function(params) {
                return `${params[0].axisValue}<br/>成功率: ${params[0].value}%`;
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
            data: hours,
            axisLabel: { color: '#666', interval: 3, rotate: 45 }
        },
        yAxis: {
            type: 'value',
            min: 80,
            max: 100,
            axisLabel: { color: '#666', formatter: '{value}%' }
        },
        series: [{
            name: '成功率',
            type: 'line',
            smooth: true,
            data: generateMockSuccessRateData(),
            areaStyle: {
                color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                    { offset: 0, color: 'rgba(16, 185, 129, 0.5)' },
                    { offset: 1, color: 'rgba(16, 185, 129, 0.05)' }
                ])
            },
            lineStyle: { color: '#10b981', width: 2 },
            itemStyle: { color: '#10b981' },
            markLine: {
                silent: true,
                lineStyle: { color: '#ef4444', type: 'dashed' },
                data: [
                    { yAxis: 95, name: '目标95%' },
                    { yAxis: 90, name: '警戒线90%' }
                ]
            }
        }]
    };

    enhancedCharts.captchaSuccessRate.setOption(option);
    initDataBuffer('captchaSuccessRate');
}

function initAnomalyDetectionChart() {
    const container = document.getElementById('anomalyDetectionChart');
    if (!container) return;

    enhancedCharts.anomalyDetection = echarts.init(container);
    window.addEventListener('resize', () => enhancedCharts.anomalyDetection.resize());

    const option = {
        title: {
            text: '异常检测',
            left: 'center',
            textStyle: { fontSize: 14, fontWeight: 'normal' }
        },
        tooltip: {
            trigger: 'axis',
            axisPointer: { type: 'line' }
        },
        legend: {
            data: ['正常请求', '异常请求'],
            bottom: '0'
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
            data: Array.from({length: 24}, (_, i) => `${i}:00`),
            axisLabel: { color: '#666', interval: 3, rotate: 45 }
        },
        yAxis: {
            type: 'value',
            axisLabel: { color: '#666' }
        },
        series: [
            {
                name: '正常请求',
                type: 'bar',
                stack: '总量',
                data: [120, 132, 101, 134, 90, 230, 210, 150, 132, 101, 134, 90, 120, 132, 101, 134, 90, 230, 210, 150, 132, 101, 134, 90],
                itemStyle: { color: '#10b981' }
            },
            {
                name: '异常请求',
                type: 'bar',
                stack: '总量',
                data: [5, 8, 12, 25, 18, 15, 22, 8, 5, 8, 12, 15, 5, 8, 12, 15, 18, 15, 22, 8, 5, 8, 12, 15],
                itemStyle: { color: '#ef4444' }
            }
        ]
    };

    enhancedCharts.anomalyDetection.setOption(option);
}

function initTrendForecastChart() {
    const container = document.getElementById('trendForecastChart');
    if (!container) return;

    enhancedCharts.trendForecast = echarts.init(container);
    window.addEventListener('resize', () => enhancedCharts.trendForecast.resize());

    const days = [];
    for (let i = 7; i >= 0; i--) {
        const date = new Date();
        date.setDate(date.getDate() - i);
        days.push(`${date.getMonth() + 1}-${date.getDate()}`);
    }

    for (let i = 1; i <= 3; i++) {
        const date = new Date();
        date.setDate(date.getDate() + i);
        days.push(`${date.getMonth() + 1}-${date.getDate()}`);
    }

    const historicalData = [820, 932, 901, 934, 1290, 1330, 1320, 1250];
    const forecastData = [null, null, null, null, null, null, null, null, 1280, 1350, 1420];

    const option = {
        title: {
            text: '趋势预测',
            left: 'center',
            textStyle: { fontSize: 14, fontWeight: 'normal' }
        },
        tooltip: {
            trigger: 'axis',
            formatter: function(params) {
                let result = params[0].axisValue + '<br/>';
                params.forEach(param => {
                    if (param.value !== null) {
                        result += param.marker + param.seriesName + ': ' + param.value + '<br/>';
                    }
                });
                return result;
            }
        },
        legend: {
            data: ['历史数据', '预测数据'],
            bottom: '0'
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
            data: days,
            boundaryGap: false,
            axisLabel: { color: '#666' }
        },
        yAxis: {
            type: 'value',
            axisLabel: { color: '#666' }
        },
        series: [
            {
                name: '历史数据',
                type: 'line',
                smooth: true,
                data: historicalData,
                areaStyle: {
                    color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                        { offset: 0, color: 'rgba(59, 130, 246, 0.3)' },
                        { offset: 1, color: 'rgba(59, 130, 246, 0.05)' }
                    ])
                },
                lineStyle: { color: '#3b82f6', width: 2 },
                itemStyle: { color: '#3b82f6' }
            },
            {
                name: '预测数据',
                type: 'line',
                smooth: true,
                data: forecastData,
                lineStyle: { color: '#f59e0b', width: 2, type: 'dashed' },
                itemStyle: { color: '#f59e0b' },
                areaStyle: {
                    color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                        { offset: 0, color: 'rgba(245, 158, 11, 0.3)' },
                        { offset: 1, color: 'rgba(245, 158, 11, 0.05)' }
                    ])
                },
                markArea: {
                    silent: true,
                    data: [
                        [
                            { xAxis: 7, itemStyle: { color: 'rgba(245, 158, 11, 0.1)' } },
                            { xAxis: 10 }
                        ]
                    ]
                }
            }
        ]
    };

    enhancedCharts.trendForecast.setOption(option);
}

function initSessionDurationChart() {
    const container = document.getElementById('sessionDurationChart');
    if (!container) return;

    enhancedCharts.sessionDuration = echarts.init(container);
    window.addEventListener('resize', () => enhancedCharts.sessionDuration.resize());

    const option = {
        title: {
            text: '会话时长分布',
            left: 'center',
            textStyle: { fontSize: 14, fontWeight: 'normal' }
        },
        tooltip: {
            trigger: 'axis',
            axisPointer: { type: 'shadow' },
            formatter: function(params) {
                return `${params[0].name}<br/>会话数: ${params[0].value}`;
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
            data: ['<1s', '1-3s', '3-5s', '5-10s', '10-30s', '>30s'],
            axisLabel: { color: '#666' }
        },
        yAxis: {
            type: 'value',
            name: '会话数',
            axisLabel: { color: '#666' }
        },
        series: [{
            name: '会话数',
            type: 'bar',
            data: [
                { value: 120, itemStyle: { color: '#3b82f6' } },
                { value: 350, itemStyle: { color: '#10b981' } },
                { value: 520, itemStyle: { color: '#f59e0b' } },
                { value: 280, itemStyle: { color: '#ef4444' } },
                { value: 150, itemStyle: { color: '#8b5cf6' } },
                { value: 80, itemStyle: { color: '#ec4899' } }
            ],
            barWidth: '50%',
            emphasis: {
                itemStyle: {
                    shadowBlur: 10,
                    shadowColor: 'rgba(0, 0, 0, 0.3)'
                }
            }
        }]
    };

    enhancedCharts.sessionDuration.setOption(option);
}

function initDataBuffer(chartName) {
    realtimeDataBuffers[chartName] = {
        data: [],
        maxPoints: MAX_DATA_POINTS
    };
}

function initWebSocketConnection() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/v1/admin/dashboard/ws`;

    try {
        wsConnection = new WebSocket(wsUrl);

        wsConnection.onopen = function() {
            isConnected = true;
            updateConnectionStatus(true);
            console.log('Enhanced WebSocket connected');
        };

        wsConnection.onmessage = function(event) {
            try {
                const data = JSON.parse(event.data);
                handleRealtimeData(data);
            } catch (e) {
                console.error('Parse enhanced WebSocket data failed:', e);
            }
        };

        wsConnection.onerror = function(error) {
            console.error('Enhanced WebSocket error:', error);
            isConnected = false;
            updateConnectionStatus(false);
        };

        wsConnection.onclose = function() {
            isConnected = false;
            updateConnectionStatus(false);
            console.log('Enhanced WebSocket disconnected, reconnecting...');
            setTimeout(initWebSocketConnection, 3000);
        };
    } catch (e) {
        console.error('Enhanced WebSocket init failed:', e);
        isConnected = false;
        updateConnectionStatus(false);
    }
}

function updateConnectionStatus(connected) {
    const statusEl = document.getElementById('enhancedWsStatus');
    if (!statusEl) return;

    if (connected) {
        statusEl.className = 'badge badge-success';
        statusEl.innerHTML = '<i class="fas fa-wifi mr-1"></i>实时已连接';
    } else {
        statusEl.className = 'badge badge-danger';
        statusEl.innerHTML = '<i class="fas fa-wifi mr-1"></i>实时已断开';
    }
}

function handleRealtimeData(data) {
    if (data.type === 'metrics') {
        updateChartWithData('responseTime', data.payload.avgResponseTime);
        updateChartWithData('captchaSuccessRate', data.payload.successRate);
    } else if (data.type === 'stats') {
        updateChartWithData('requestDistribution', data.payload.distribution);
    }
}

function updateChartWithData(chartName, value) {
    if (!realtimeDataBuffers[chartName]) {
        initDataBuffer(chartName);
    }

    const buffer = realtimeDataBuffers[chartName];
    buffer.data.push({
        time: new Date(),
        value: value
    });

    if (buffer.data.length > buffer.maxPoints) {
        buffer.data.shift();
    }

    updateChartDisplay(chartName, buffer.data);
}

function updateChartDisplay(chartName, data) {
    const chart = enhancedCharts[chartName];
    if (!chart) return;

    const labels = data.map(d => d.time.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }));
    const values = data.map(d => d.value);

    if (chartName === 'responseTime') {
        chart.setOption({
            xAxis: { data: labels },
            series: [{ data: values }]
        }, false);
    } else if (chartName === 'captchaSuccessRate') {
        chart.setOption({
            xAxis: { data: labels },
            series: [{ data: values }]
        }, false);
    }
}

function startRealtimeUpdates() {
    Object.keys(enhancedCharts).forEach(chartName => {
        chartUpdateTimers[chartName] = setInterval(() => {
            if (!isConnected) {
                simulateDataUpdate(chartName);
            }
        }, CHART_UPDATE_INTERVAL);
    });
}

function simulateDataUpdate(chartName) {
    if (!realtimeDataBuffers[chartName]) {
        initDataBuffer(chartName);
    }

    const buffer = realtimeDataBuffers[chartName];
    let newValue;

    switch (chartName) {
        case 'responseTime':
            newValue = 50 + Math.random() * 30;
            break;
        case 'captchaSuccessRate':
            newValue = 90 + Math.random() * 8;
            break;
        default:
            return;
    }

    updateChartWithData(chartName, newValue);
}

function setupChartInteractions() {
    Object.keys(enhancedCharts).forEach(chartName => {
        const chart = enhancedCharts[chartName];

        chart.on('click', function(params) {
            showChartDetailModal(chartName, params);
        });

        chart.on('datazoom', function(params) {
            console.log(`${chartName} zoom:`, params);
        });
    });
}

function showChartDetailModal(chartName, params) {
    const modal = document.getElementById('enhancedChartModal');
    if (!modal) return;

    const title = getChartTitle(chartName);
    const content = generateChartDetailContent(chartName, params);

    document.getElementById('enhancedModalTitle').textContent = title;
    document.getElementById('enhancedModalContent').innerHTML = content;

    const bsModal = new bootstrap.Modal(modal);
    bsModal.show();
}

function getChartTitle(chartName) {
    const titles = {
        requestDistribution: '请求分布详情',
        responseTime: '响应时间详情',
        userActivity: '用户活动详情',
        geographic: '地域分布详情',
        captchaSuccessRate: '验证码成功率详情',
        anomalyDetection: '异常检测详情',
        trendForecast: '趋势预测详情',
        sessionDuration: '会话时长详情'
    };
    return titles[chartName] || '图表详情';
}

function generateChartDetailContent(chartName, params) {
    let content = '<div class="text-center">';

    if (params.name) {
        content += `<h4 class="text-primary">${params.name}</h4>`;
    }

    if (params.value !== undefined) {
        content += `<p class="text-muted">数值: <strong>${params.value}</strong></p>`;
    }

    content += `<p class="text-muted">更新时间: ${new Date().toLocaleString('zh-CN')}</p>`;
    content += '</div>';

    return content;
}

function exportChartData(chartName) {
    const chart = enhancedCharts[chartName];
    if (!chart) return;

    try {
        const url = chart.getDataURL({
            type: 'png',
            pixelRatio: 2,
            backgroundColor: '#fff'
        });

        const link = document.createElement('a');
        link.download = `${chartName}_${Date.now()}.png`;
        link.href = url;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);

        showToast('图表导出成功', 'success');
    } catch (error) {
        console.error('图表导出失败:', error);
        showToast('图表导出失败', 'error');
    }
}

function refreshAllCharts() {
    Object.keys(enhancedCharts).forEach(chartName => {
        const chart = enhancedCharts[chartName];
        if (chart) {
            chart.clear();
            switch (chartName) {
                case 'requestDistribution':
                    initRequestDistributionChart();
                    break;
                case 'responseTime':
                    initResponseTimeChart();
                    break;
                case 'userActivity':
                    initUserActivityChart();
                    break;
                case 'geographic':
                    initGeographicChart();
                    break;
                case 'captchaSuccessRate':
                    initCaptchaSuccessRateChart();
                    break;
                case 'anomalyDetection':
                    initAnomalyDetectionChart();
                    break;
                case 'trendForecast':
                    initTrendForecastChart();
                    break;
                case 'sessionDuration':
                    initSessionDurationChart();
                    break;
            }
        }
    });
    showToast('图表已刷新', 'success');
}

function generateMockResponseTimeData() {
    return Array.from({length: 24}, () => Math.floor(Math.random() * 30) + 50);
}

function generateMockSuccessRateData() {
    return Array.from({length: 24}, () => (90 + Math.random() * 8).toFixed(1));
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
    Object.values(chartUpdateTimers).forEach(timer => clearInterval(timer));
    Object.values(enhancedCharts).forEach(chart => chart.dispose());
    if (wsConnection) wsConnection.close();
});
