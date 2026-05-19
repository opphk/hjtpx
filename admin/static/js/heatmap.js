let heatmapInstances = {};
let heatmapData = {};
let heatmapConfig = {
    radius: 25,
    blur: 15,
    maxOpacity: 0.6,
    minOpacity: 0.1,
    gradient: {
        '0.0': '#313695',
        '0.2': '#4575b4',
        '0.4': '#74add1',
        '0.6': '#fee090',
        '0.8': '#fdae61',
        '1.0': '#d73027'
    }
};

document.addEventListener('DOMContentLoaded', () => {
    initializeHeatmaps();
    setupHeatmapControls();
});

function initializeHeatmaps() {
    initUserBehaviorHeatmap();
    initCaptchaClickHeatmap();
    initTimeDistributionHeatmap();
    initGeoHeatmap();
}

function initUserBehaviorHeatmap() {
    const container = document.getElementById('userBehaviorHeatmap');
    if (!container) return;

    const heatmap = echarts.init(container);
    window.addEventListener('resize', () => heatmap.resize());

    const points = generateUserBehaviorData();
    heatmapData.userBehavior = points;

    const option = {
        title: {
            text: '用户行为热力图',
            subtext: '鼠标移动和点击密度分析',
            left: 'center'
        },
        tooltip: {
            position: 'top',
            formatter: function(params) {
                return `坐标: (${params.data[0]}, ${params.data[1]})<br/>强度: ${(params.data[2] * 100).toFixed(1)}%`;
            }
        },
        grid: {
            left: '5%',
            right: '10%',
            top: '15%',
            bottom: '15%',
            containLabel: true
        },
        xAxis: {
            type: 'value',
            min: 0,
            max: 1920,
            splitNumber: 12,
            axisLabel: {
                formatter: function(value) {
                    return value + 'px';
                }
            }
        },
        yAxis: {
            type: 'value',
            min: 0,
            max: 1080,
            splitNumber: 9,
            axisLabel: {
                formatter: function(value) {
                    return value + 'px';
                }
            }
        },
        visualMap: {
            min: 0,
            max: 1,
            calculable: true,
            orient: 'horizontal',
            left: 'center',
            bottom: '0%',
            inRange: {
                color: ['#313695', '#4575b4', '#74add1', '#abd9e9', '#ffffbf', '#fee090', '#fdae61', '#f46d43', '#d73027', '#a50026']
            },
            textStyle: {
                color: '#333'
            }
        },
        series: [{
            name: '用户行为',
            type: 'heatmap',
            data: points,
            label: {
                show: false
            },
            emphasis: {
                itemStyle: {
                    shadowBlur: 20,
                    shadowColor: 'rgba(0, 0, 0, 0.5)'
                }
            },
            blurSize: heatmapConfig.blur,
            pointSize: heatmapConfig.radius
        }]
    };

    heatmap.setOption(option);
    heatmapInstances.userBehavior = heatmap;

    setupHeatmapInteraction(heatmap, 'userBehavior');
}

function initCaptchaClickHeatmap() {
    const container = document.getElementById('captchaClickHeatmap');
    if (!container) return;

    const heatmap = echarts.init(container);
    window.addEventListener('resize', () => heatmap.resize());

    const points = generateCaptchaClickData();
    heatmapData.captchaClick = points;

    const option = {
        title: {
            text: '验证码点击热力图',
            subtext: '验证码区域点击密度',
            left: 'center'
        },
        tooltip: {
            position: 'top',
            formatter: function(params) {
                const intensity = params.data[2] * 100;
                let level = '低';
                if (intensity > 70) level = '极高';
                else if (intensity > 50) level = '高';
                else if (intensity > 30) level = '中';
                return `坐标: (${params.data[0]}, ${params.data[1]})<br/>强度: ${intensity.toFixed(1)}%<br/>等级: ${level}`;
            }
        },
        grid: {
            left: '5%',
            right: '10%',
            top: '15%',
            bottom: '15%',
            containLabel: true
        },
        xAxis: {
            type: 'value',
            min: 0,
            max: 300,
            splitNumber: 6,
            axisLabel: {
                formatter: function(value) {
                    return value + 'px';
                }
            }
        },
        yAxis: {
            type: 'value',
            min: 0,
            max: 300,
            splitNumber: 6,
            axisLabel: {
                formatter: function(value) {
                    return value + 'px';
                }
            }
        },
        visualMap: {
            min: 0,
            max: 1,
            calculable: true,
            orient: 'horizontal',
            left: 'center',
            bottom: '0%',
            inRange: {
                color: ['#00ff00', '#ffff00', '#ff9900', '#ff0000']
            },
            textStyle: {
                color: '#333'
            }
        },
        series: [{
            name: '验证码点击',
            type: 'heatmap',
            data: points,
            label: {
                show: false
            },
            emphasis: {
                itemStyle: {
                    shadowBlur: 15,
                    shadowColor: 'rgba(0, 0, 0, 0.5)'
                }
            },
            blurSize: 20,
            pointSize: 30
        }]
    };

    heatmap.setOption(option);
    heatmapInstances.captchaClick = heatmap;

    setupHeatmapInteraction(heatmap, 'captchaClick');
}

function initTimeDistributionHeatmap() {
    const container = document.getElementById('timeDistributionHeatmap');
    if (!container) return;

    const heatmap = echarts.init(container);
    window.addEventListener('resize', () => heatmap.resize());

    const hours = ['0时', '1时', '2时', '3时', '4时', '5时', '6时', '7时', '8时', '9时', '10时', '11时',
                   '12时', '13时', '14时', '15时', '16时', '17时', '18时', '19时', '20时', '21时', '22时', '23时'];
    const days = ['周一', '周二', '周三', '周四', '周五', '周六', '周日'];

    const data = [];
    for (let i = 0; i < 7; i++) {
        for (let j = 0; j < 24; j++) {
            let value = Math.floor(Math.random() * 500) + 100;

            if (j >= 9 && j <= 11) value += 200;
            if (j >= 14 && j <= 17) value += 150;
            if (j >= 19 && j <= 21) value += 180;

            if (i >= 5) {
                if (j >= 10 && j <= 20) value += 300;
            }

            data.push([j, i, value]);
        }
    }

    heatmapData.timeDistribution = data;

    const option = {
        title: {
            text: '时间分布热力图',
            subtext: '按小时和星期分布的用户活动',
            left: 'center'
        },
        tooltip: {
            position: 'top',
            formatter: function(params) {
                return hours[params.data[0]] + ' ' + days[params.data[1]] + '<br/>活动量: ' + params.data[2];
            }
        },
        grid: {
            left: '3%',
            right: '8%',
            top: '15%',
            bottom: '15%',
            containLabel: true
        },
        xAxis: {
            type: 'category',
            data: hours,
            splitArea: { show: true },
            axisLabel: {
                interval: 2,
                color: '#666'
            }
        },
        yAxis: {
            type: 'category',
            data: days,
            splitArea: { show: true },
            axisLabel: {
                color: '#666'
            }
        },
        visualMap: {
            min: 0,
            max: 800,
            calculable: true,
            orient: 'vertical',
            right: '0',
            top: 'center',
            inRange: {
                color: ['#e8f5e9', '#81c784', '#4caf50', '#2e7d32', '#1b5e20']
            },
            textStyle: {
                color: '#333'
            }
        },
        series: [{
            name: '活动量',
            type: 'heatmap',
            data: data,
            label: {
                show: false
            },
            emphasis: {
                itemStyle: {
                    shadowBlur: 10,
                    shadowColor: 'rgba(0, 0, 0, 0.5)'
                }
            }
        }]
    };

    heatmap.setOption(option);
    heatmapInstances.timeDistribution = heatmap;

    setupHeatmapInteraction(heatmap, 'timeDistribution');
}

function initGeoHeatmap() {
    const container = document.getElementById('geoHeatmap');
    if (!container) return;

    const geoChart = echarts.init(container);
    window.addEventListener('resize', () => geoChart.resize());

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
        { name: '重庆', value: 1200 },
        { name: '天津', value: 1100 },
        { name: '苏州', value: 1050 },
        { name: '郑州', value: 1000 },
        { name: '长沙', value: 950 },
        { name: '沈阳', value: 900 }
    ];

    const option = {
        title: {
            text: '地域分布热力图',
            subtext: '用户地域分布密度',
            left: 'center'
        },
        tooltip: {
            trigger: 'item',
            formatter: function(params) {
                return params.name + '<br/>用户数: ' + params.value;
            }
        },
        visualMap: {
            min: 0,
            max: 4000,
            calculable: true,
            orient: 'vertical',
            right: '0',
            top: 'center',
            inRange: {
                color: ['#313695', '#4575b4', '#74add1', '#abd9e9', '#ffffbf', '#fee090', '#fdae61', '#f46d43', '#d73027', '#a50026']
            },
            textStyle: {
                color: '#333'
            }
        },
        series: [{
            name: '地域分布',
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
            axisLabel: {
                color: '#666'
            }
        },
        yAxis: {
            type: 'category',
            data: geoData.sort((a, b) => b.value - a.value).map(d => d.name),
            axisLabel: {
                color: '#666'
            }
        }
    };

    geoChart.setOption(option);
    heatmapInstances.geo = geoChart;
}

function generateUserBehaviorData() {
    const points = [];
    const hotspotCount = 10;

    for (let i = 0; i < hotspotCount; i++) {
        const centerX = Math.random() * 1600 + 160;
        const centerY = Math.random() * 900 + 90;
        const intensity = 0.5 + Math.random() * 0.5;

        for (let j = 0; j < 50; j++) {
            const offsetX = (Math.random() - 0.5) * 200;
            const offsetY = (Math.random() - 0.5) * 200;
            const pointIntensity = intensity * (0.5 + Math.random() * 0.5);

            points.push([
                Math.min(1920, Math.max(0, centerX + offsetX)),
                Math.min(1080, Math.max(0, centerY + offsetY)),
                pointIntensity
            ]);
        }
    }

    return points;
}

function generateCaptchaClickData() {
    const points = [];
    const gridSize = 50;

    for (let i = 0; i < 6; i++) {
        for (let j = 0; j < 6; j++) {
            const baseX = i * gridSize + Math.random() * 20;
            const baseY = j * gridSize + Math.random() * 20;

            const clickCount = Math.floor(Math.random() * 20);
            for (let k = 0; k < clickCount; k++) {
                const offsetX = (Math.random() - 0.5) * 30;
                const offsetY = (Math.random() - 0.5) * 30;
                const intensity = Math.random();

                points.push([
                    Math.min(300, Math.max(0, baseX + offsetX)),
                    Math.min(300, Math.max(0, baseY + offsetY)),
                    intensity
                ]);
            }
        }
    }

    return points;
}

function setupHeatmapInteraction(heatmap, type) {
    heatmap.on('click', function(params) {
        showHeatmapDetail(type, params);
    });

    heatmap.on('mouseover', function(params) {
        highlightHeatmapPoint(heatmap, params);
    });

    heatmap.on('mouseout', function() {
        resetHeatmapHighlight(heatmap);
    });
}

function highlightHeatmapPoint(heatmap, params) {
    if (!params.data) return;

    const option = heatmap.getOption();
    const seriesIndex = params.seriesIndex;

    option.series[seriesIndex].emphasis = {
        itemStyle: {
            borderColor: '#fff',
            borderWidth: 2,
            shadowBlur: 20,
            shadowColor: 'rgba(255, 255, 0, 0.8)'
        }
    };

    heatmap.setOption(option);
}

function resetHeatmapHighlight(heatmap) {
    const option = heatmap.getOption();
    option.series.forEach(s => {
        s.emphasis = {
            itemStyle: {
                shadowBlur: 10,
                shadowColor: 'rgba(0, 0, 0, 0.5)'
            }
        };
    });
    heatmap.setOption(option);
}

function showHeatmapDetail(type, params) {
    const modal = document.getElementById('heatmapDetailModal');
    if (!modal) return;

    const titles = {
        userBehavior: '用户行为热力图详情',
        captchaClick: '验证码点击热力图详情',
        timeDistribution: '时间分布热力图详情',
        geo: '地域分布详情'
    };

    let content = '<div class="text-center">';
    content += `<h5 class="text-primary">${titles[type] || '热力图详情'}</h5>`;

    if (params.data) {
        if (type === 'timeDistribution') {
            const hours = ['0时', '1时', '2时', '3时', '4时', '5时', '6时', '7时', '8时', '9时', '10时', '11时',
                          '12时', '13时', '14时', '15时', '16时', '17时', '18时', '19时', '20时', '21时', '22时', '23时'];
            const days = ['周一', '周二', '周三', '周四', '周五', '周六', '周日'];
            content += `<p class="text-muted">时间: ${hours[params.data[0]]} ${days[params.data[1]]}</p>`;
            content += `<p class="text-muted">活动量: <strong>${params.data[2]}</strong></p>`;
        } else {
            content += `<p class="text-muted">坐标: (${params.data[0]}, ${params.data[1]})</p>`;
            content += `<p class="text-muted">强度: <strong>${(params.data[2] * 100).toFixed(1)}%</strong></p>`;
        }
    } else if (params.name) {
        content += `<p class="text-muted">地区: <strong>${params.name}</strong></p>`;
        content += `<p class="text-muted">用户数: <strong>${params.value}</strong></p>`;
    }

    content += `<p class="text-muted mt-3">更新时间: ${new Date().toLocaleString('zh-CN')}</p>`;
    content += '</div>';

    document.getElementById('heatmapDetailTitle').textContent = titles[type] || '热力图详情';
    document.getElementById('heatmapDetailContent').innerHTML = content;

    const bsModal = new bootstrap.Modal(modal);
    bsModal.show();
}

function setupHeatmapControls() {
    const refreshBtn = document.getElementById('refreshHeatmapBtn');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', refreshAllHeatmaps);
    }

    const exportBtns = document.querySelectorAll('[data-heatmap-export]');
    exportBtns.forEach(btn => {
        btn.addEventListener('click', (e) => {
            const type = e.target.dataset.heatmapExport;
            exportHeatmapImage(type);
        });
    });
}

function refreshAllHeatmaps() {
    Object.keys(heatmapInstances).forEach(type => {
        const heatmap = heatmapInstances[type];
        if (heatmap) {
            heatmap.clear();

            switch (type) {
                case 'userBehavior':
                    initUserBehaviorHeatmap();
                    break;
                case 'captchaClick':
                    initCaptchaClickHeatmap();
                    break;
                case 'timeDistribution':
                    initTimeDistributionHeatmap();
                    break;
                case 'geo':
                    initGeoHeatmap();
                    break;
            }
        }
    });

    showToast('热力图已刷新', 'success');
}

function exportHeatmapImage(type) {
    const heatmap = heatmapInstances[type];
    if (!heatmap) return;

    try {
        const url = heatmap.getDataURL({
            type: 'png',
            pixelRatio: 2,
            backgroundColor: '#fff'
        });

        const link = document.createElement('a');
        link.download = `heatmap_${type}_${Date.now()}.png`;
        link.href = url;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);

        showToast('热力图导出成功', 'success');
    } catch (error) {
        console.error('热力图导出失败:', error);
        showToast('热力图导出失败', 'error');
    }
}

function updateHeatmapData(type, newData) {
    if (!heatmapInstances[type]) return;

    const heatmap = heatmapInstances[type];
    heatmapData[type] = newData;

    const option = heatmap.getOption();
    if (option.series[0]) {
        option.series[0].data = newData;
        heatmap.setOption(option);
    }
}

function addHeatmapPoint(type, x, y, intensity) {
    if (!heatmapData[type]) {
        heatmapData[type] = [];
    }

    heatmapData[type].push([x, y, intensity]);

    updateHeatmapData(type, heatmapData[type]);
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
    Object.values(heatmapInstances).forEach(heatmap => {
        if (heatmap) heatmap.dispose();
    });
});
