class ChartManager {
    constructor() {
        this.charts = new Map();
        this.defaultColors = [
            'rgba(59, 130, 246, 0.8)',
            'rgba(16, 185, 129, 0.8)',
            'rgba(245, 158, 11, 0.8)',
            'rgba(239, 68, 68, 0.8)',
            'rgba(139, 92, 246, 0.8)',
            'rgba(236, 72, 153, 0.8)',
            'rgba(20, 184, 166, 0.8)',
            'rgba(249, 115, 22, 0.8)'
        ];
        this.defaultBorderColor = [
            'rgba(59, 130, 246, 1)',
            'rgba(16, 185, 129, 1)',
            'rgba(245, 158, 11, 1)',
            'rgba(239, 68, 68, 1)',
            'rgba(139, 92, 246, 1)',
            'rgba(236, 72, 153, 1)',
            'rgba(20, 184, 166, 1)',
            'rgba(249, 115, 22, 1)'
        ];
    }

    createLineChart(canvasId, data, options = {}) {
        const canvas = document.getElementById(canvasId);
        if (!canvas) {
            console.error(`Canvas element with id '${canvasId}' not found`);
            return null;
        }

        const ctx = canvas.getContext('2d');
        const defaultOptions = {
            type: 'line',
            data: {
                labels: data.labels || [],
                datasets: this.formatLineDatasets(data.datasets || [])
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                interaction: {
                    mode: 'index',
                    intersect: false
                },
                plugins: {
                    legend: {
                        display: options.showLegend !== false,
                        position: 'top',
                        labels: {
                            usePointStyle: true,
                            padding: 20
                        }
                    },
                    tooltip: {
                        backgroundColor: 'rgba(0, 0, 0, 0.8)',
                        padding: 12,
                        titleFont: { size: 14 },
                        bodyFont: { size: 13 },
                        cornerRadius: 8
                    }
                },
                scales: {
                    x: {
                        grid: {
                            display: options.gridX !== false,
                            color: 'rgba(0, 0, 0, 0.05)'
                        },
                        ticks: {
                            maxRotation: 45,
                            minRotation: 0
                        }
                    },
                    y: {
                        beginAtZero: options.beginAtZero !== false,
                        grid: {
                            color: 'rgba(0, 0, 0, 0.05)'
                        },
                        ticks: {
                            callback: function(value) {
                                if (typeof value === 'number') {
                                    if (value >= 1000000) return (value / 1000000).toFixed(1) + 'M';
                                    if (value >= 1000) return (value / 1000).toFixed(1) + 'K';
                                }
                                return value;
                            }
                        }
                    }
                },
                ...options.chartOptions
            }
        };

        if (this.charts.has(canvasId)) {
            this.charts.get(canvasId).destroy();
        }

        const chart = new Chart(ctx, defaultOptions);
        this.charts.set(canvasId, chart);
        return chart;
    }

    createPieChart(canvasId, data, options = {}) {
        const canvas = document.getElementById(canvasId);
        if (!canvas) {
            console.error(`Canvas element with id '${canvasId}' not found`);
            return null;
        }

        const ctx = canvas.getContext('2d');
        const defaultOptions = {
            type: 'doughnut',
            data: {
                labels: data.labels || [],
                datasets: [{
                    data: data.values || [],
                    backgroundColor: this.defaultColors.slice(0, data.labels?.length || 0),
                    borderColor: 'rgba(255, 255, 255, 1)',
                    borderWidth: 2
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        display: options.showLegend !== false,
                        position: options.legendPosition || 'right',
                        labels: {
                            usePointStyle: true,
                            padding: 15
                        }
                    },
                    tooltip: {
                        backgroundColor: 'rgba(0, 0, 0, 0.8)',
                        padding: 12,
                        callbacks: {
                            label: function(context) {
                                const value = context.raw;
                                const total = context.dataset.data.reduce((a, b) => a + b, 0);
                                const percentage = ((value / total) * 100).toFixed(1);
                                return `${context.label}: ${value} (${percentage}%)`;
                            }
                        }
                    }
                },
                cutout: options.cutout || '60%'
            }
        };

        if (this.charts.has(canvasId)) {
            this.charts.get(canvasId).destroy();
        }

        const chart = new Chart(ctx, defaultOptions);
        this.charts.set(canvasId, chart);
        return chart;
    }

    createBarChart(canvasId, data, options = {}) {
        const canvas = document.getElementById(canvasId);
        if (!canvas) {
            console.error(`Canvas element with id '${canvasId}' not found`);
            return null;
        }

        const ctx = canvas.getContext('2d');
        const defaultOptions = {
            type: 'bar',
            data: {
                labels: data.labels || [],
                datasets: this.formatBarDatasets(data.datasets || [])
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        display: options.showLegend !== false,
                        position: 'top',
                        labels: {
                            usePointStyle: true,
                            padding: 20
                        }
                    },
                    tooltip: {
                        backgroundColor: 'rgba(0, 0, 0, 0.8)',
                        padding: 12
                    }
                },
                scales: {
                    x: {
                        grid: {
                            display: false
                        }
                    },
                    y: {
                        beginAtZero: options.beginAtZero !== false,
                        grid: {
                            color: 'rgba(0, 0, 0, 0.05)'
                        }
                    }
                }
            }
        };

        if (this.charts.has(canvasId)) {
            this.charts.get(canvasId).destroy();
        }

        const chart = new Chart(ctx, defaultOptions);
        this.charts.set(canvasId, chart);
        return chart;
    }

    createRadarChart(canvasId, data, options = {}) {
        const canvas = document.getElementById(canvasId);
        if (!canvas) {
            console.error(`Canvas element with id '${canvasId}' not found`);
            return null;
        }

        const ctx = canvas.getContext('2d');
        const defaultOptions = {
            type: 'radar',
            data: {
                labels: data.labels || [],
                datasets: this.formatRadarDatasets(data.datasets || [])
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        display: options.showLegend !== false,
                        position: 'top'
                    },
                    tooltip: {
                        backgroundColor: 'rgba(0, 0, 0, 0.8)'
                    }
                },
                scales: {
                    r: {
                        beginAtZero: true,
                        grid: {
                            color: 'rgba(0, 0, 0, 0.1)'
                        }
                    }
                }
            }
        };

        if (this.charts.has(canvasId)) {
            this.charts.get(canvasId).destroy();
        }

        const chart = new Chart(ctx, defaultOptions);
        this.charts.set(canvasId, chart);
        return chart;
    }

    createPolarAreaChart(canvasId, data, options = {}) {
        const canvas = document.getElementById(canvasId);
        if (!canvas) {
            console.error(`Canvas element with id '${canvasId}' not found`);
            return null;
        }

        const ctx = canvas.getContext('2d');
        const defaultOptions = {
            type: 'polarArea',
            data: {
                labels: data.labels || [],
                datasets: [{
                    data: data.values || [],
                    backgroundColor: this.defaultColors.slice(0, data.labels?.length || 0).map(c => c.replace('0.8', '0.6')),
                    borderWidth: 2,
                    borderColor: 'rgba(255, 255, 255, 1)'
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        display: options.showLegend !== false,
                        position: 'right'
                    },
                    tooltip: {
                        backgroundColor: 'rgba(0, 0, 0, 0.8)'
                    }
                }
            }
        };

        if (this.charts.has(canvasId)) {
            this.charts.get(canvasId).destroy();
        }

        const chart = new Chart(ctx, defaultOptions);
        this.charts.set(canvasId, chart);
        return chart;
    }

    formatLineDatasets(datasets) {
        return datasets.map((dataset, index) => ({
            label: dataset.label || `Dataset ${index + 1}`,
            data: dataset.data || [],
            borderColor: dataset.borderColor || this.defaultBorderColor[index % this.defaultBorderColor.length],
            backgroundColor: dataset.backgroundColor || this.defaultColors[index % this.defaultColors.length],
            borderWidth: dataset.borderWidth || 2,
            fill: dataset.fill !== undefined ? dataset.fill : false,
            tension: dataset.tension || 0.4,
            pointRadius: dataset.pointRadius || 3,
            pointHoverRadius: dataset.pointHoverRadius || 6
        }));
    }

    formatBarDatasets(datasets) {
        return datasets.map((dataset, index) => ({
            label: dataset.label || `Dataset ${index + 1}`,
            data: dataset.data || [],
            backgroundColor: dataset.backgroundColor || this.defaultColors[index % this.defaultColors.length],
            borderColor: dataset.borderColor || this.defaultBorderColor[index % this.defaultBorderColor.length],
            borderWidth: dataset.borderWidth || 0,
            borderRadius: dataset.borderRadius || 4
        }));
    }

    formatRadarDatasets(datasets) {
        return datasets.map((dataset, index) => ({
            label: dataset.label || `Dataset ${index + 1}`,
            data: dataset.data || [],
            borderColor: this.defaultBorderColor[index % this.defaultBorderColor.length],
            backgroundColor: this.defaultColors[index % this.defaultColors.length].replace('0.8', '0.2'),
            borderWidth: 2,
            pointRadius: 3,
            pointHoverRadius: 5
        }));
    }

    updateChart(canvasId, newData) {
        const chart = this.charts.get(canvasId);
        if (!chart) {
            console.warn(`Chart '${canvasId}' not found`);
            return;
        }

        if (newData.labels) {
            chart.data.labels = newData.labels;
        }

        if (newData.datasets) {
            newData.datasets.forEach((dataset, index) => {
                if (chart.data.datasets[index]) {
                    if (dataset.data) {
                        chart.data.datasets[index].data = dataset.data;
                    }
                    if (dataset.label) {
                        chart.data.datasets[index].label = dataset.label;
                    }
                }
            });
        }

        chart.update('active');
    }

    addDataset(canvasId, dataset) {
        const chart = this.charts.get(canvasId);
        if (!chart) {
            console.warn(`Chart '${canvasId}' not found`);
            return;
        }

        const index = chart.data.datasets.length;
        const newDataset = {
            label: dataset.label || `Dataset ${index + 1}`,
            data: dataset.data || [],
            borderColor: dataset.borderColor || this.defaultBorderColor[index % this.defaultBorderColor.length],
            backgroundColor: dataset.backgroundColor || this.defaultColors[index % this.defaultColors.length]
        };

        chart.data.datasets.push(newDataset);
        chart.update('active');
    }

    removeDataset(canvasId, datasetIndex) {
        const chart = this.charts.get(canvasId);
        if (!chart) {
            console.warn(`Chart '${canvasId}' not found`);
            return;
        }

        if (datasetIndex >= 0 && datasetIndex < chart.data.datasets.length) {
            chart.data.datasets.splice(datasetIndex, 1);
            chart.update('active');
        }
    }

    destroyChart(canvasId) {
        const chart = this.charts.get(canvasId);
        if (chart) {
            chart.destroy();
            this.charts.delete(canvasId);
        }
    }

    destroyAllCharts() {
        this.charts.forEach((chart, canvasId) => {
            chart.destroy();
        });
        this.charts.clear();
    }

    exportChart(canvasId, format = 'png', filename = 'chart') {
        const chart = this.charts.get(canvasId);
        if (!chart) {
            console.warn(`Chart '${canvasId}' not found`);
            return null;
        }

        const mimeTypes = {
            'png': 'image/png',
            'jpeg': 'image/jpeg',
            'jpg': 'image/jpeg',
            'webp': 'image/webp',
            'svg': 'image/svg+xml'
        };

        const mimeType = mimeTypes[format.toLowerCase()] || 'image/png';
        const dataUrl = chart.toBase64Image(mimeType, 1.0);

        if (format.toLowerCase() === 'svg') {
            return this.convertToSvg(dataUrl);
        }

        return dataUrl;
    }

    downloadChart(canvasId, format = 'png', filename = 'chart') {
        const dataUrl = this.exportChart(canvasId, format, filename);
        if (!dataUrl) return;

        const link = document.createElement('a');
        link.href = dataUrl;
        link.download = `${filename}.${format}`;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
    }

    convertToSvg(dataUrl) {
        const img = new Image();
        const canvas = document.createElement('canvas');
        const ctx = canvas.getContext('2d');

        return new Promise((resolve) => {
            img.onload = () => {
                canvas.width = img.width;
                canvas.height = img.height;
                ctx.drawImage(img, 0, 0);
                const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${img.width}" height="${img.height}">
                    <image href="${canvas.toDataURL('image/png')}" width="${img.width}" height="${img.height}"/>
                </svg>`;
                resolve(svg);
            };
            img.src = dataUrl;
        });
    }

    getChartInstance(canvasId) {
        return this.charts.get(canvasId) || null;
    }

    hasChart(canvasId) {
        return this.charts.has(canvasId);
    }

    animateChart(canvasId, newData, duration = 1000) {
        const chart = this.charts.get(canvasId);
        if (!chart) return;

        const startData = JSON.parse(JSON.stringify(chart.data));
        const endLabels = newData.labels || chart.data.labels;
        const endDatasets = newData.datasets || chart.data.datasets;

        const startTime = performance.now();

        const animate = (currentTime) => {
            const elapsed = currentTime - startTime;
            const progress = Math.min(elapsed / duration, 1);
            const easedProgress = this.easeOutCubic(progress);

            if (newData.labels) {
                chart.data.labels = endLabels;
            }

            endDatasets.forEach((dataset, index) => {
                if (chart.data.datasets[index] && dataset.data) {
                    chart.data.datasets[index].data = dataset.data.map((value, i) => {
                        const startValue = startData.datasets[index]?.data[i] || 0;
                        return startValue + (value - startValue) * easedProgress;
                    });
                }
            });

            chart.update('none');

            if (progress < 1) {
                requestAnimationFrame(animate);
            }
        };

        requestAnimationFrame(animate);
    }

    easeOutCubic(x) {
        return 1 - Math.pow(1 - x, 3);
    }

    createRealtimeChart(canvasId, options = {}) {
        const canvas = document.getElementById(canvasId);
        if (!canvas) {
            console.error(`Canvas element with id '${canvasId}' not found`);
            return null;
        }

        const maxDataPoints = options.maxDataPoints || 20;
        const labels = [];
        const datasetsData = options.datasets || [{ label: 'Data', data: [] }];

        for (let i = maxDataPoints - 1; i >= 0; i--) {
            labels.push(this.getTimeLabel(-i, options.interval || 'second'));
        }

        const data = {
            labels: labels,
            datasets: datasetsData.map((ds, index) => ({
                label: ds.label || `Dataset ${index + 1}`,
                data: new Array(maxDataPoints).fill(null),
                borderColor: ds.borderColor || this.defaultBorderColor[index % this.defaultBorderColor.length],
                backgroundColor: ds.backgroundColor || this.defaultColors[index % this.defaultColors.length],
                borderWidth: 2,
                fill: false,
                tension: 0.4,
                pointRadius: 0
            }))
        };

        const ctx = canvas.getContext('2d');
        const chart = new Chart(ctx, {
            type: 'line',
            data: data,
            options: {
                responsive: true,
                maintainAspectRatio: false,
                animation: {
                    duration: 300
                },
                plugins: {
                    legend: {
                        display: options.showLegend !== false
                    },
                    tooltip: {
                        mode: 'index',
                        intersect: false
                    }
                },
                scales: {
                    x: {
                        display: true,
                        grid: {
                            display: false
                        },
                        ticks: {
                            maxRotation: 0,
                            autoSkip: true,
                            maxTicksLimit: 10
                        }
                    },
                    y: {
                        display: true,
                        grid: {
                            color: 'rgba(0, 0, 0, 0.05)'
                        },
                        beginAtZero: options.beginAtZero !== false
                    }
                },
                interaction: {
                    mode: 'nearest',
                    axis: 'x',
                    intersect: false
                }
            }
        });

        this.charts.set(canvasId, chart);

        chart.realtimeUpdate = (newValues) => {
            if (!Array.isArray(newValues)) {
                newValues = [newValues];
            }

            const now = new Date();
            const timeLabel = options.formatTime ? options.formatTime(now) :
                `${now.getHours().toString().padStart(2, '0')}:${now.getMinutes().toString().padStart(2, '0')}:${now.getSeconds().toString().padStart(2, '0')}`;

            chart.data.labels.push(timeLabel);
            if (chart.data.labels.length > maxDataPoints) {
                chart.data.labels.shift();
            }

            newValues.forEach((value, index) => {
                if (chart.data.datasets[index]) {
                    chart.data.datasets[index].data.push(value);
                    if (chart.data.datasets[index].data.length > maxDataPoints) {
                        chart.data.datasets[index].data.shift();
                    }
                }
            });

            chart.update('none');
        };

        return chart;
    }

    getTimeLabel(offset, interval) {
        const now = new Date();
        now.setSeconds(now.getSeconds() + offset);

        if (interval === 'minute') {
            return `${now.getHours().toString().padStart(2, '0')}:${now.getMinutes().toString().padStart(2, '0')}`;
        } else if (interval === 'hour') {
            return `${now.getHours().toString().padStart(2, '0')}:00`;
        } else if (interval === 'day') {
            return `${now.getMonth() + 1}/${now.getDate()}`;
        } else {
            return `${now.getMinutes().toString().padStart(2, '0')}:${now.getSeconds().toString().padStart(2, '0')}`;
        }
    }

    createGradient(chart, datasetIndex, colorStart, colorEnd) {
        const ctx = chart.ctx;
        const gradient = ctx.createLinearGradient(0, 0, 0, chart.height);
        gradient.addColorStop(0, colorStart);
        gradient.addColorStop(1, colorEnd);
        return gradient;
    }
}

const chartManager = new ChartManager();
