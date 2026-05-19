let predictionChart, riskChart, captchaTypeChart;

document.addEventListener('DOMContentLoaded', function() {
    initCharts();
    startAutoUpdate();
});

function initCharts() {
    initPredictionChart();
    initRiskChart();
    initCaptchaTypeChart();
}

function initPredictionChart() {
    const container = document.getElementById('predictionChart');
    if (!container) return;

    predictionChart = echarts.init(container);
    window.addEventListener('resize', () => predictionChart.resize());

    const dates = [];
    const actualData = [];
    const predictedData = [];

    const today = new Date();
    for (let i = 14; i >= 0; i--) {
        const date = new Date(today);
        date.setDate(date.getDate() - i);
        dates.push(date.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' }));
        
        if (i > 7) {
            actualData.push(Math.floor(Math.random() * 50000) + 80000);
            predictedData.push(null);
        } else if (i === 7) {
            actualData.push(Math.floor(Math.random() * 50000) + 80000);
            predictedData.push(actualData[actualData.length - 1]);
        } else {
            actualData.push(null);
            const lastValue = predictedData.findLast(v => v !== null) || 100000;
            predictedData.push(Math.floor(lastValue * (0.9 + Math.random() * 0.3)));
        }
    }

    predictionChart.setOption({
        backgroundColor: 'transparent',
        tooltip: {
            trigger: 'axis',
            backgroundColor: 'rgba(0,0,0,0.8)',
            borderColor: 'rgba(0,212,255,0.3)',
            textStyle: { color: '#fff' },
            axisPointer: { type: 'cross' }
        },
        legend: {
            data: ['实际值', '预测值'],
            textStyle: { color: 'rgba(224,230,237,0.7)' },
            top: 0
        },
        grid: { left: '3%', right: '4%', bottom: '10%', top: '15%', containLabel: true },
        xAxis: {
            type: 'category',
            boundaryGap: false,
            data: dates,
            axisLine: { lineStyle: { color: 'rgba(255,255,255,0.1)' } },
            axisLabel: { color: 'rgba(224,230,237,0.5)' },
            splitLine: { show: false }
        },
        yAxis: {
            type: 'value',
            axisLine: { lineStyle: { color: 'rgba(255,255,255,0.1)' } },
            axisLabel: { color: 'rgba(224,230,237,0.5)' },
            splitLine: { lineStyle: { color: 'rgba(255,255,255,0.05)' } }
        },
        series: [
            {
                name: '实际值',
                type: 'line',
                smooth: true,
                data: actualData,
                lineStyle: { color: '#00d4ff', width: 3 },
                itemStyle: { color: '#00d4ff' },
                areaStyle: {
                    color: {
                        type: 'linear',
                        x: 0, y: 0, x2: 0, y2: 1,
                        colorStops: [
                            { offset: 0, color: 'rgba(0,212,255,0.3)' },
                            { offset: 1, color: 'rgba(0,212,255,0.05)' }
                        ]
                    }
                }
            },
            {
                name: '预测值',
                type: 'line',
                smooth: true,
                data: predictedData,
                lineStyle: { color: '#6610f2', width: 3, type: 'dashed' },
                itemStyle: { color: '#6610f2' },
                areaStyle: {
                    color: {
                        type: 'linear',
                        x: 0, y: 0, x2: 0, y2: 1,
                        colorStops: [
                            { offset: 0, color: 'rgba(102,16,242,0.2)' },
                            { offset: 1, color: 'rgba(102,16,242,0.02)' }
                        ]
                    }
                }
            }
        ]
    });
}

function initRiskChart() {
    const container = document.getElementById('riskChart');
    if (!container) return;

    riskChart = echarts.init(container);
    window.addEventListener('resize', () => riskChart.resize());

    riskChart.setOption({
        backgroundColor: 'transparent',
        tooltip: {
            trigger: 'item',
            backgroundColor: 'rgba(0,0,0,0.8)',
            textStyle: { color: '#fff' }
        },
        legend: {
            orient: 'vertical',
            left: 'left',
            textStyle: { color: 'rgba(224,230,237,0.7)' }
        },
        series: [{
            type: 'pie',
            radius: ['40%', '70%'],
            center: ['60%', '50%'],
            avoidLabelOverlap: false,
            itemStyle: {
                borderRadius: 10,
                borderColor: 'rgba(20,25,45,1)',
                borderWidth: 2
            },
            label: {
                show: false,
                position: 'center'
            },
            emphasis: {
                label: {
                    show: true,
                    fontSize: 16,
                    fontWeight: 'bold',
                    color: '#fff'
                }
            },
            labelLine: {
                show: false
            },
            data: [
                { value: 70, name: '低风险', itemStyle: { color: '#28a745' } },
                { value: 20, name: '中风险', itemStyle: { color: '#ffc107' } },
                { value: 8, name: '高风险', itemStyle: { color: '#dc3545' } },
                { value: 2, name: '严重风险', itemStyle: { color: '#8b0000' } }
            ]
        }]
    });
}

function initCaptchaTypeChart() {
    const container = document.getElementById('captchaTypeChart');
    if (!container) return;

    captchaTypeChart = echarts.init(container);
    window.addEventListener('resize', () => captchaTypeChart.resize());

    const days = ['周一', '周二', '周三', '周四', '周五', '周六', '周日'];
    const sliderData = days.map(() => Math.floor(Math.random() * 3000) + 2000);
    const clickData = days.map(() => Math.floor(Math.random() * 2000) + 1500);
    const gestureData = days.map(() => Math.floor(Math.random() * 1500) + 1000);
    const puzzleData = days.map(() => Math.floor(Math.random() * 1000) + 800);

    captchaTypeChart.setOption({
        backgroundColor: 'transparent',
        tooltip: {
            trigger: 'axis',
            backgroundColor: 'rgba(0,0,0,0.8)',
            textStyle: { color: '#fff' }
        },
        legend: {
            data: ['滑块', '点击', '手势', '拼图'],
            textStyle: { color: 'rgba(224,230,237,0.7)' },
            bottom: 0
        },
        grid: { left: '3%', right: '4%', bottom: '20%', top: '10%', containLabel: true },
        xAxis: {
            type: 'category',
            data: days,
            axisLine: { lineStyle: { color: 'rgba(255,255,255,0.1)' } },
            axisLabel: { color: 'rgba(224,230,237,0.5)' }
        },
        yAxis: {
            type: 'value',
            axisLine: { lineStyle: { color: 'rgba(255,255,255,0.1)' } },
            axisLabel: { color: 'rgba(224,230,237,0.5)' },
            splitLine: { lineStyle: { color: 'rgba(255,255,255,0.05)' } }
        },
        series: [
            {
                name: '滑块',
                type: 'bar',
                stack: 'total',
                data: sliderData,
                itemStyle: { color: '#00d4ff', borderRadius: [4, 4, 0, 0] }
            },
            {
                name: '点击',
                type: 'bar',
                stack: 'total',
                data: clickData,
                itemStyle: { color: '#28a745', borderRadius: [4, 4, 0, 0] }
            },
            {
                name: '手势',
                type: 'bar',
                stack: 'total',
                data: gestureData,
                itemStyle: { color: '#ffc107', borderRadius: [4, 4, 0, 0] }
            },
            {
                name: '拼图',
                type: 'bar',
                stack: 'total',
                data: puzzleData,
                itemStyle: { color: '#dc3545', borderRadius: [4, 4, 0, 0] }
            }
        ]
    });
}

function generateReport() {
    if (!confirm('确定要生成新的智能报表吗？\n此操作可能需要几秒钟。')) {
        return;
    }

    const btn = event.target;
    const originalText = btn.innerHTML;
    btn.innerHTML = '<i class="fas fa-spinner fa-spin me-2"></i>生成中...';
    btn.disabled = true;

    setTimeout(() => {
        btn.innerHTML = originalText;
        btn.disabled = false;

        const tableBody = document.getElementById('reportHistoryTable');
        const newRow = document.createElement('tr');
        const now = new Date();
        newRow.innerHTML = `
            <td><strong>智能分析报告 - 实时</strong></td>
            <td><span class="metric-badge bg-purple bg-opacity-25 text-purple">AI分析</span></td>
            <td>${now.toLocaleString('zh-CN')}</td>
            <td>最近 24 小时</td>
            <td><span class="badge bg-success">已完成</span></td>
            <td>
                <button class="btn btn-sm btn-outline-primary" onclick="viewReport(${Date.now()})">查看</button>
                <button class="btn btn-sm btn-outline-secondary" onclick="downloadReport(${Date.now()})">下载</button>
            </td>
        `;
        tableBody.insertBefore(newRow, tableBody.firstChild);

        alert('报表生成成功！');
    }, 2000);
}

function exportReport() {
    const reportData = {
        title: '墨盾验证 AI 分析报表',
        generatedAt: new Date().toISOString(),
        summary: '系统运行正常，各项指标良好',
        metrics: {
            predictionAccuracy: 98.5,
            dailyRequests: '1.2M',
            successRate: 99.2,
            avgResponseTime: '123ms'
        },
        recommendations: [
            '建议在高峰期前 30 分钟提前扩容',
            '优化缓存策略可进一步提升性能',
            '关注移动端用户体验优化'
        ]
    };

    const dataStr = JSON.stringify(reportData, null, 2);
    const dataBlob = new Blob([dataStr], { type: 'application/json' });
    const link = document.createElement('a');
    link.href = URL.createObjectURL(dataBlob);
    link.download = `ai-report-${Date.now()}.json`;
    link.click();
}

function viewReport(id) {
    alert(`正在查看报表 #${id}...\n（实际项目中会打开报表详情页面）`);
}

function downloadReport(id) {
    alert(`正在下载报表 #${id}...\n（实际项目中会触发文件下载）`);
}

function applyRecommendation(type) {
    if (!confirm('确定要应用此 AI 推荐吗？')) {
        return;
    }

    alert(`已应用推荐: ${type}\n系统将自动进行优化配置。`);
}

function startAutoUpdate() {
    setInterval(() => {
        updateForecastData();
    }, 10000);
}

function updateForecastData() {
    const forecastList = document.getElementById('forecastList');
    if (!forecastList) return;

    const items = forecastList.querySelectorAll('.d-flex');
    items.forEach((item, index) => {
        const valueEl = item.querySelector('.fw-bold');
        if (valueEl) {
            const currentValue = parseFloat(valueEl.textContent);
            const change = (Math.random() - 0.5) * 4;
            const newValue = Math.max(-10, Math.min(30, currentValue + change));
            valueEl.textContent = (newValue >= 0 ? '+' : '') + newValue.toFixed(0) + '%';
            valueEl.className = 'fw-bold ' + (newValue >= 0 ? 'text-primary' : 'text-danger');
        }
    });
}
