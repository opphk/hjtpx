class TrendAnalysis {
    constructor(containerId, options = {}) {
        this.container = document.getElementById(containerId);
        if (!this.container) {
            console.error('TrendAnalysis container not found');
            return;
        }

        this.options = {
            chartType: options.chartType || 'line',
            predictionDays: options.predictionDays || 7,
            confidenceInterval: options.confidenceInterval || 0.95,
            showTrendLine: options.showTrendLine !== false,
            showConfidenceInterval: options.showConfidenceInterval !== false,
            showAnomalies: options.showAnomalies !== false,
            animationDuration: options.animationDuration || 500,
            ...options
        };

        this.historicalData = [];
        this.predictedData = [];
        this.confidenceUpper = [];
        this.confidenceLower = [];
        this.anomalies = [];
        this.trendLine = null;
        this.chart = null;

        this.init();
    }

    init() {
        this.setupChart();
        this.attachEventListeners();
    }

    setupChart() {
        const chartContainer = document.createElement('div');
        chartContainer.className = 'trend-chart-container';
        chartContainer.style.cssText = 'width: 100%; height: 400px;';
        this.container.appendChild(chartContainer);

        this.chart = echarts.init(chartContainer);
        this.renderEmptyChart();
    }

    renderEmptyChart() {
        const option = {
            title: {
                text: 'Trend Analysis',
                left: 'center'
            },
            tooltip: {
                trigger: 'axis',
                axisPointer: {
                    type: 'cross'
                }
            },
            legend: {
                data: ['Historical', 'Predicted', 'Confidence Interval'],
                bottom: 0
            },
            grid: {
                left: '3%',
                right: '4%',
                bottom: '15%',
                top: '15%',
                containLabel: true
            },
            xAxis: {
                type: 'category',
                boundaryGap: false,
                data: []
            },
            yAxis: {
                type: 'value'
            },
            series: []
        };

        this.chart.setOption(option);
    }

    setHistoricalData(data) {
        this.historicalData = data.map(d => ({
            date: d.date,
            value: d.value
        }));

        this.detectAnomalies();
        this.calculateTrendLine();
        this.render();
    }

    addDataPoint(date, value) {
        this.historicalData.push({ date, value });
        this.detectAnomalies();
        this.calculateTrendLine();
        this.render();
    }

    detectAnomalies() {
        if (this.historicalData.length < 3) {
            this.anomalies = [];
            return;
        }

        const values = this.historicalData.map(d => d.value);
        const mean = values.reduce((a, b) => a + b, 0) / values.length;
        const stdDev = Math.sqrt(
            values.reduce((sum, val) => sum + Math.pow(val - mean, 2), 0) / values.length
        );

        const threshold = 2;
        this.anomalies = this.historicalData
            .map((d, i) => ({
                index: i,
                date: d.date,
                value: d.value,
                isAnomaly: Math.abs(d.value - mean) > threshold * stdDev
            }))
            .filter(d => d.isAnomaly);
    }

    calculateTrendLine() {
        if (this.historicalData.length < 2) {
            this.trendLine = null;
            return;
        }

        const n = this.historicalData.length;
        let sumX = 0, sumY = 0, sumXY = 0, sumX2 = 0;

        this.historicalData.forEach((d, i) => {
            sumX += i;
            sumY += d.value;
            sumXY += i * d.value;
            sumX2 += i * i;
        });

        const slope = (n * sumXY - sumX * sumY) / (n * sumX2 - sumX * sumX);
        const intercept = (sumY - slope * sumX) / n;

        this.trendLine = {
            slope,
            intercept,
            equation: `y = ${slope.toFixed(2)}x + ${intercept.toFixed(2)}`,
            direction: slope > 0 ? 'upward' : slope < 0 ? 'downward' : 'stable'
        };
    }

    predict(days = this.options.predictionDays) {
        if (!this.trendLine || this.historicalData.length === 0) {
            return [];
        }

        this.predictedData = [];
        this.confidenceUpper = [];
        this.confidenceLower = [];

        const historicalValues = this.historicalData.map(d => d.value);
        const mean = historicalValues.reduce((a, b) => a + b, 0) / historicalValues.length;
        const stdDev = Math.sqrt(
            historicalValues.reduce((sum, val) => sum + Math.pow(val - mean, 2), 0) / historicalValues.length
        );

        const lastDate = new Date(this.historicalData[this.historicalData.length - 1].date);
        const startIndex = this.historicalData.length;

        for (let i = 1; i <= days; i++) {
            const futureIndex = startIndex + i - 1;
            const predictedValue = this.trendLine.slope * futureIndex + this.trendLine.intercept;

            const confidenceMultiplier = 1.96 * stdDev * Math.sqrt(1 + i / this.historicalData.length);

            this.predictedData.push({
                date: this.formatDate(new Date(lastDate.getTime() + i * 24 * 60 * 60 * 1000)),
                value: Math.max(0, predictedValue)
            });

            this.confidenceUpper.push({
                date: this.formatDate(new Date(lastDate.getTime() + i * 24 * 60 * 60 * 1000)),
                value: Math.max(0, predictedValue + confidenceMultiplier)
            });

            this.confidenceLower.push({
                date: this.formatDate(new Date(lastDate.getTime() + i * 24 * 60 * 60 * 1000)),
                value: Math.max(0, predictedValue - confidenceMultiplier)
            });
        }

        return this.predictedData;
    }

    formatDate(date) {
        return date.toISOString().split('T')[0];
    }

    render() {
        const historical = this.historicalData.map(d => [d.date, d.value]);
        const predicted = this.predictedData.map(d => [d.date, d.value]);
        const upper = this.confidenceUpper.map(d => [d.date, d.value]);
        const lower = this.confidenceLower.map(d => [d.date, d.value]);

        const anomalyData = this.anomalies.map(a => ({
            coord: [a.date, a.value],
            value: a.value,
            itemStyle: {
                color: '#ff4444'
            }
        }));

        const series = [
            {
                name: 'Historical',
                type: 'line',
                data: historical,
                smooth: true,
                showSymbol: false,
                lineStyle: {
                    width: 2,
                    color: '#3b82f6'
                },
                areaStyle: {
                    color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                        { offset: 0, color: 'rgba(59, 130, 246, 0.3)' },
                        { offset: 1, color: 'rgba(59, 130, 246, 0.05)' }
                    ])
                }
            }
        ];

        if (this.options.showTrendLine && this.trendLine) {
            const trendData = this.historicalData.map((d, i) => [
                d.date,
                this.trendLine.slope * i + this.trendLine.intercept
            ]);
            series.push({
                name: 'Trend Line',
                type: 'line',
                data: trendData,
                smooth: true,
                showSymbol: false,
                lineStyle: {
                    width: 2,
                    color: '#10b981',
                    type: 'dashed'
                }
            });
        }

        if (predicted.length > 0) {
            series.push({
                name: 'Predicted',
                type: 'line',
                data: predicted,
                smooth: true,
                showSymbol: false,
                lineStyle: {
                    width: 2,
                    color: '#f59e0b'
                },
                areaStyle: {
                    color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                        { offset: 0, color: 'rgba(245, 158, 11, 0.3)' },
                        { offset: 1, color: 'rgba(245, 158, 11, 0.05)' }
                    ])
                }
            });

            if (this.options.showConfidenceInterval && upper.length > 0) {
                series.push({
                    name: 'Confidence Interval',
                    type: 'custom',
                    renderItem: (params, api) => {
                        const point = api.coord([upper[params.dataIndex].date, upper[params.dataIndex].value]);
                        const bottom = api.coord([lower[params.dataIndex].date, lower[params.dataIndex].value]);

                        return {
                            type: 'rect',
                            shape: {
                                x: point[0] - 10,
                                y: bottom[1],
                                width: 20,
                                height: point[1] - bottom[1]
                            },
                            style: {
                                fill: 'rgba(245, 158, 11, 0.1)'
                            }
                        };
                    },
                    data: upper.map((d, i) => i),
                    z: 0
                });
            }
        }

        if (this.options.showAnomalies && anomalyData.length > 0) {
            series.push({
                name: 'Anomalies',
                type: 'scatter',
                data: anomalyData,
                symbolSize: 10,
                itemStyle: {
                    color: '#ef4444'
                },
                tooltip: {
                    formatter: (params) => {
                        return `Anomaly Detected<br/>Date: ${params.data.coord[0]}<br/>Value: ${params.data.value.toFixed(2)}`;
                    }
                }
            });
        }

        const option = {
            title: {
                text: 'Trend Analysis & Prediction',
                left: 'center'
            },
            tooltip: {
                trigger: 'axis',
                axisPointer: {
                    type: 'cross'
                },
                formatter: (params) => {
                    if (params.length === 0) return '';
                    let result = `<strong>${params[0].axisValue}</strong><br/>`;
                    params.forEach(p => {
                        if (p.seriesName !== 'Confidence Interval' && p.value) {
                            result += `${p.marker} ${p.seriesName}: <strong>${(Array.isArray(p.value) ? p.value[1] : p.value).toFixed(2)}</strong><br/>`;
                        }
                    });
                    return result;
                }
            },
            legend: {
                data: series.map(s => s.name).filter(n => n),
                bottom: 0
            },
            grid: {
                left: '3%',
                right: '4%',
                bottom: '15%',
                top: '15%',
                containLabel: true
            },
            xAxis: {
                type: 'time',
                boundaryGap: false,
                axisLabel: {
                    formatter: (value) => {
                        const date = new Date(value);
                        return `${date.getMonth() + 1}/${date.getDate()}`;
                    }
                }
            },
            yAxis: {
                type: 'value',
                axisLabel: {
                    formatter: (value) => value.toFixed(0)
                }
            },
            series,
            dataZoom: [
                {
                    type: 'inside',
                    start: 0,
                    end: 100
                },
                {
                    type: 'slider',
                    start: 0,
                    end: 100
                }
            ]
        };

        this.chart.setOption(option, true);
    }

    attachEventListeners() {
        window.addEventListener('resize', () => {
            if (this.chart) {
                this.chart.resize();
            }
        });
    }

    getAnalysisSummary() {
        const historicalValues = this.historicalData.map(d => d.value);
        const mean = historicalValues.reduce((a, b) => a + b, 0) / historicalValues.length;
        const min = Math.min(...historicalValues);
        const max = Math.max(...historicalValues);

        let growth = 0;
        if (historicalValues.length >= 2) {
            const first = historicalValues.slice(0, Math.ceil(historicalValues.length / 2));
            const second = historicalValues.slice(Math.floor(historicalValues.length / 2));
            const firstAvg = first.reduce((a, b) => a + b, 0) / first.length;
            const secondAvg = second.reduce((a, b) => a + b, 0) / second.length;
            growth = ((secondAvg - firstAvg) / firstAvg) * 100;
        }

        return {
            dataPoints: this.historicalData.length,
            mean: mean.toFixed(2),
            min: min.toFixed(2),
            max: max.toFixed(2),
            growthRate: growth.toFixed(2) + '%',
            trendDirection: this.trendLine?.direction || 'unknown',
            trendEquation: this.trendLine?.equation || 'N/A',
            anomaliesCount: this.anomalies.length,
            predictedPoints: this.predictedData.length,
            predictionAccuracy: this.options.confidenceInterval * 100 + '%'
        };
    }

    exportData(format = 'json') {
        const data = {
            historical: this.historicalData,
            predicted: this.predictedData,
            anomalies: this.anomalies,
            trendLine: this.trendLine,
            summary: this.getAnalysisSummary(),
            generatedAt: new Date().toISOString()
        };

        if (format === 'csv') {
            let csv = 'type,date,value\n';
            this.historicalData.forEach(d => {
                csv += `historical,${d.date},${d.value}\n`;
            });
            this.predictedData.forEach(d => {
                csv += `predicted,${d.date},${d.value}\n`;
            });
            return csv;
        }

        return JSON.stringify(data, null, 2);
    }

    destroy() {
        if (this.chart) {
            this.chart.dispose();
        }
        this.historicalData = [];
        this.predictedData = [];
        this.anomalies = [];
        if (this.container) {
            this.container.innerHTML = '';
        }
    }
}

class MovingAverage {
    constructor(windowSize = 5) {
        this.windowSize = windowSize;
        this.data = [];
    }

    add(value) {
        this.data.push(value);
        if (this.data.length > this.windowSize) {
            this.data.shift();
        }
    }

    getAverage() {
        if (this.data.length === 0) return 0;
        return this.data.reduce((a, b) => a + b, 0) / this.data.length;
    }

    reset() {
        this.data = [];
    }
}

class ExponentialSmoothing {
    constructor(alpha = 0.3) {
        this.alpha = alpha;
        this.smoothed = null;
    }

    add(value) {
        if (this.smoothed === null) {
            this.smoothed = value;
        } else {
            this.smoothed = this.alpha * value + (1 - this.alpha) * this.smoothed;
        }
        return this.smoothed;
    }

    getSmoothed() {
        return this.smoothed;
    }

    reset() {
        this.smoothed = null;
    }
}

class SimpleExponentialSmoothingForecast {
    constructor(alpha = 0.3) {
        this.alpha = alpha;
        this.level = null;
        this.trend = null;
        this.data = [];
    }

    add(value) {
        this.data.push(value);

        if (this.data.length === 1) {
            this.level = value;
            this.trend = 0;
        } else if (this.data.length === 2) {
            this.trend = value - this.data[0];
            this.level = value;
        } else {
            const lastLevel = this.level;
            const lastTrend = this.trend;

            this.level = this.alpha * value + (1 - this.alpha) * (lastLevel + lastTrend);
            this.trend = this.alpha * (this.level - lastLevel) + (1 - this.alpha) * lastTrend;
        }

        return this.level;
    }

    forecast(steps = 1) {
        const forecasts = [];
        let level = this.level || 0;
        let trend = this.trend || 0;

        for (let i = 1; i <= steps; i++) {
            forecasts.push(level + i * trend);
        }

        return forecasts;
    }

    getSmoothedSeries() {
        const smoothed = new SimpleExponentialSmoothingForecast(this.alpha);
        return this.data.map(v => smoothed.add(v));
    }
}

window.TrendAnalysis = TrendAnalysis;
window.MovingAverage = MovingAverage;
window.ExponentialSmoothing = ExponentialSmoothing;
window.SimpleExponentialSmoothingForecast = SimpleExponentialSmoothingForecast;
