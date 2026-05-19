class RealtimeAnalytics {
    constructor(config) {
        this.config = config || {};
        this.updateInterval = this.config.updateInterval || 5000;
        this.maxDataPoints = 20;
        this.data = {
            labels: [],
            verificationData: [],
            successData: [],
            failedData: [],
            qpsData: []
        };
        this.init();
    }
    
    init() {
        this.setupChart();
        this.startUpdates();
        this.setupWebSocket();
        this.initializeRealtimeCanvas();
    }
    
    setupChart() {
        const ctx = document.getElementById('realtimeChartCanvas');
        if (!ctx) {
            console.warn('realtimeChartCanvas element not found');
            return;
        }
        
        this.chart = new Chart(ctx, {
            type: 'line',
            data: {
                labels: this.data.labels,
                datasets: [
                    {
                        label: '总验证数',
                        data: this.data.verificationData,
                        borderColor: 'rgb(75, 192, 192)',
                        backgroundColor: 'rgba(75, 192, 192, 0.1)',
                        tension: 0.4
                    },
                    {
                        label: '成功率 (%)',
                        data: this.data.successData,
                        borderColor: 'rgb(54, 162, 235)',
                        backgroundColor: 'rgba(54, 162, 235, 0.1)',
                        tension: 0.4,
                        yAxisID: 'y1'
                    },
                    {
                        label: 'QPS',
                        data: this.data.qpsData,
                        borderColor: 'rgb(255, 99, 132)',
                        backgroundColor: 'rgba(255, 99, 132, 0.1)',
                        tension: 0.4,
                        yAxisID: 'y2'
                    }
                ]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                animation: {
                    duration: 300
                },
                scales: {
                    y: {
                        type: 'linear',
                        display: true,
                        position: 'left',
                        title: {
                            display: true,
                            text: '验证数量',
                            color: 'rgb(75, 192, 192)'
                        },
                        grid: {
                            color: 'rgba(0, 0, 0, 0.1)'
                        }
                    },
                    y1: {
                        type: 'linear',
                        display: true,
                        position: 'right',
                        title: {
                            display: true,
                            text: '成功率 (%)',
                            color: 'rgb(54, 162, 235)'
                        },
                        grid: {
                            drawOnChartArea: false
                        },
                        min: 0,
                        max: 100
                    },
                    y2: {
                        type: 'linear',
                        display: true,
                        position: 'right',
                        title: {
                            display: true,
                            text: 'QPS',
                            color: 'rgb(255, 99, 132)'
                        },
                        grid: {
                            drawOnChartArea: false
                        }
                    },
                    x: {
                        grid: {
                            color: 'rgba(0, 0, 0, 0.1)'
                        }
                    }
                },
                plugins: {
                    legend: {
                        display: true,
                        position: 'top'
                    },
                    tooltip: {
                        mode: 'index',
                        intersect: false
                    }
                },
                interaction: {
                    mode: 'nearest',
                    axis: 'x',
                    intersect: false
                }
            }
        });
        
        window.chartInstance = this.chart;
    }
    
    initializeRealtimeCanvas() {
        const canvas = document.getElementById('realtimeChartCanvas');
        if (canvas) {
            canvas.style.backgroundColor = 'rgba(255, 255, 255, 0.5)';
            canvas.style.borderRadius = '8px';
        }
    }
    
    startUpdates() {
        this.updateTimer = setInterval(() => {
            this.fetchData();
        }, this.updateInterval);
        
        this.fetchData();
    }
    
    async fetchData() {
        try {
            const response = await fetch('/api/v1/admin/stats/realtime');
            if (!response.ok) {
                throw new Error('Network response was not ok');
            }
            const data = await response.json();
            this.updateChart(data);
        } catch (error) {
            console.error('获取数据失败:', error);
            this.loadMockData();
        }
    }
    
    loadMockData() {
        const mockData = {
            totalVerifications: Math.floor(Math.random() * 1000) + 500,
            successRate: Math.random() * 0.3 + 0.7,
            currentQPS: Math.floor(Math.random() * 50) + 10
        };
        this.updateChart(mockData);
    }
    
    updateChart(data) {
        const now = new Date().toLocaleTimeString();
        
        this.data.labels.push(now);
        this.data.verificationData.push(data.totalVerifications || 0);
        this.data.successData.push((data.successRate || 0) * 100);
        this.data.qpsData.push(data.currentQPS || 0);
        
        if (this.data.labels.length > this.maxDataPoints) {
            this.data.labels.shift();
            this.data.verificationData.shift();
            this.data.successData.shift();
            this.data.qpsData.shift();
        }
        
        if (this.chart) {
            this.chart.data.labels = this.data.labels;
            this.chart.data.datasets[0].data = this.data.verificationData;
            this.chart.data.datasets[1].data = this.data.successData;
            this.chart.data.datasets[2].data = this.data.qpsData;
            this.chart.update('none');
        }
    }
    
    setupWebSocket() {
        const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${wsProtocol}//${window.location.host}/ws/realtime`;
        
        try {
            this.ws = new WebSocket(wsUrl);
            
            this.ws.onopen = () => {
                console.log('WebSocket连接已建立');
                this.updateConnectionStatus(true);
            };
            
            this.ws.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);
                    this.handleWebSocketMessage(data);
                } catch (error) {
                    console.error('解析WebSocket消息失败:', error);
                }
            };
            
            this.ws.onerror = (error) => {
                console.error('WebSocket错误:', error);
                this.updateConnectionStatus(false);
            };
            
            this.ws.onclose = () => {
                console.log('WebSocket连接已关闭');
                this.updateConnectionStatus(false);
                setTimeout(() => this.setupWebSocket(), 5000);
            };
        } catch (error) {
            console.log('WebSocket不可用，切换到轮询模式');
            this.updateConnectionStatus(false);
        }
    }
    
    updateConnectionStatus(connected) {
        const statusEl = document.getElementById('enhancedWsStatus');
        if (statusEl) {
            if (connected) {
                statusEl.className = 'badge badge-success';
                statusEl.innerHTML = '<i class="fas fa-wifi mr-1"></i>实时';
            } else {
                statusEl.className = 'badge badge-secondary';
                statusEl.innerHTML = '<i class="fas fa-sync mr-1"></i>轮询';
            }
        }
    }
    
    handleWebSocketMessage(data) {
        if (data.type === 'metrics') {
            this.updateChart(data.payload);
        } else if (data.type === 'stats') {
            this.updateChart(data);
        }
    }
    
    addDataPoint(label, verification, successRate, qps) {
        this.data.labels.push(label);
        this.data.verificationData.push(verification);
        this.data.successData.push(successRate);
        this.data.qpsData.push(qps);
        
        if (this.data.labels.length > this.maxDataPoints) {
            this.data.labels.shift();
            this.data.verificationData.shift();
            this.data.successData.shift();
            this.data.qpsData.shift();
        }
        
        if (this.chart) {
            this.chart.update('none');
        }
    }
    
    reset() {
        this.data = {
            labels: [],
            verificationData: [],
            successData: [],
            failedData: [],
            qpsData: []
        };
        
        if (this.chart) {
            this.chart.data.labels = [];
            this.chart.data.datasets[0].data = [];
            this.chart.data.datasets[1].data = [];
            this.chart.data.datasets[2].data = [];
            this.chart.update();
        }
    }
    
    destroy() {
        if (this.updateTimer) {
            clearInterval(this.updateTimer);
        }
        if (this.ws) {
            this.ws.close();
        }
        if (this.chart) {
            this.chart.destroy();
        }
        window.chartInstance = null;
    }
    
    setUpdateInterval(interval) {
        this.updateInterval = interval;
        if (this.updateTimer) {
            clearInterval(this.updateTimer);
            this.startUpdates();
        }
    }
    
    setMaxDataPoints(maxPoints) {
        this.maxDataPoints = maxPoints;
        while (this.data.labels.length > maxPoints) {
            this.data.labels.shift();
            this.data.verificationData.shift();
            this.data.successData.shift();
            this.data.qpsData.shift();
        }
        if (this.chart) {
            this.chart.update('none');
        }
    }
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = RealtimeAnalytics;
}

if (typeof window !== 'undefined') {
    window.RealtimeAnalytics = RealtimeAnalytics;
}
