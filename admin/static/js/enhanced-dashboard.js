/**
 * HJTPX Admin Enhanced Dashboard JavaScript
 * 增强仪表盘功能模块
 */

const EnhancedDashboard = {
    charts: {},
    realtimeData: [],
    ws: null,
    wsConnected: false,
    autoRefreshInterval: null,
    config: {
        maxRealtimePoints: 60,
        updateInterval: 5000,
        wsReconnectDelay: 3000,
        themeKey: 'dashboardTheme',
        timeRangeKey: 'dashboardTimeRange'
    },

    init() {
        this.loadConfig();
        this.initCharts();
        this.initWebSocket();
        this.setupEventListeners();
        this.loadData();
        this.startAutoRefresh();
    },

    loadConfig() {
        const savedTimeRange = localStorage.getItem(this.config.timeRangeKey);
        this.currentTimeRange = savedTimeRange || '24h';
        
        const savedTheme = localStorage.getItem(this.config.themeKey);
        if (savedTheme) {
            this.setTheme(savedTheme);
        }
    },

    saveConfig() {
        localStorage.setItem(this.config.timeRangeKey, this.currentTimeRange);
    },

    initCharts() {
        this.initTrendChart();
        this.initPieChart();
        this.initRadarChart();
        this.initHeatmapChart();
        this.initStackChart();
        this.initRealtimeChart();
    },

    initTrendChart() {
        const container = document.getElementById('trendChart');
        if (!container) return;

        this.charts.trend = echarts.init(container);
        window.addEventListener('resize', () => this.charts.trend.resize());
        
        this.charts.trend.setOption({
            tooltip: {
                trigger: 'axis',
                backgroundColor: 'rgba(0,0,0,0.8)',
                textStyle: { color: '#fff' }
            },
            legend: {
                data: ['验证请求', '通过数', '拦截数'],
                bottom: '0'
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
                data: []
            },
            yAxis: {
                type: 'value'
            },
            series: []
        });
    },

    initPieChart() {
        const container = document.getElementById('pieChart');
        if (!container) return;

        this.charts.pie = echarts.init(container);
        window.addEventListener('resize', () => this.charts.pie.resize());
    },

    initRadarChart() {
        const container = document.getElementById('radarChart');
        if (!container) return;

        this.charts.radar = echarts.init(container);
        window.addEventListener('resize', () => this.charts.radar.resize());

        this.charts.radar.setOption({
            radar: {
                indicator: [
                    { name: '响应速度', max: 100 },
                    { name: '稳定性', max: 100 },
                    { name: '安全性', max: 100 },
                    { name: '可用性', max: 100 },
                    { name: '准确性', max: 100 }
                ],
                radius: '65%'
            },
            series: [{
                type: 'radar',
                data: [{
                    value: [85, 90, 88, 92, 87],
                    name: '系统性能'
                }]
            }]
        });
    },

    initHeatmapChart() {
        const container = document.getElementById('heatmapChart');
        if (!container) return;

        this.charts.heatmap = echarts.init(container);
        window.addEventListener('resize', () => this.charts.heatmap.resize());
    },

    initStackChart() {
        const container = document.getElementById('stackChart');
        if (!container) return;

        this.charts.stack = echarts.init(container);
        window.addEventListener('resize', () => this.charts.stack.resize());
    },

    initRealtimeChart() {
        const container = document.getElementById('realtimeChart');
        if (!container) return;

        this.realtimeData = Array(this.config.maxRealtimePoints).fill(0).map(() => ({
            time: new Date(),
            value: 0
        }));

        this.charts.realtime = echarts.init(container);
        window.addEventListener('resize', () => this.charts.realtime.resize());
        
        this.updateRealtimeChart();
    },

    updateRealtimeChart() {
        if (!this.charts.realtime) return;

        this.charts.realtime.setOption({
            xAxis: {
                type: 'category',
                data: this.realtimeData.map(p => this.formatTime(p.time)),
                boundaryGap: false
            },
            yAxis: {
                type: 'value'
            },
            series: [{
                data: this.realtimeData.map(p => p.value),
                type: 'line',
                smooth: true,
                areaStyle: {
                    color: {
                        type: 'linear',
                        x: 0, y: 0, x2: 0, y2: 1,
                        colorStops: [
                            { offset: 0, color: 'rgba(16, 185, 129, 0.3)' },
                            { offset: 1, color: 'rgba(16, 185, 129, 0.05)' }
                        ]
                    }
                },
                lineStyle: { color: '#10b981', width: 2 },
                itemStyle: { color: '#10b981' }
            }],
            grid: { left: '3%', right: '4%', bottom: '15%', containLabel: true }
        }, false);
    },

    initWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/api/v1/admin/dashboard/ws`;

        try {
            this.ws = new WebSocket(wsUrl);
            
            this.ws.onopen = () => {
                this.wsConnected = true;
                this.updateWsStatus(true);
            };
            
            this.ws.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);
                    this.handleRealtimeData(data);
                } catch (e) {
                    console.error('Parse WebSocket data failed:', e);
                }
            };
            
            this.ws.onerror = (error) => {
                console.error('WebSocket error:', error);
                this.wsConnected = false;
                this.updateWsStatus(false);
            };
            
            this.ws.onclose = () => {
                this.wsConnected = false;
                this.updateWsStatus(false);
                setTimeout(() => this.initWebSocket(), this.config.wsReconnectDelay);
            };
        } catch (e) {
            console.error('WebSocket init failed:', e);
            this.wsConnected = false;
            this.updateWsStatus(false);
        }
    },

    updateWsStatus(connected) {
        const statusEl = document.getElementById('wsStatus');
        if (!statusEl) return;
        
        if (connected) {
            statusEl.className = 'badge badge-success';
            statusEl.innerHTML = '<i class="fas fa-wifi mr-1"></i>已连接';
        } else {
            statusEl.className = 'badge badge-danger';
            statusEl.innerHTML = '<i class="fas fa-wifi mr-1"></i>已断开';
        }
    },

    handleRealtimeData(data) {
        if (data.type === 'metrics') {
            this.updateMetrics(data.payload);
        } else if (data.type === 'verification') {
            this.addVerificationRow(data.payload);
        } else if (data.type === 'stats') {
            this.updateDashboard(data.payload);
        }
    },

    updateMetrics(data) {
        if (data.total_requests !== undefined) {
            this.animateValue('totalRequests', data.total_requests);
        }
        
        if (data.qps !== undefined) {
            this.addRealtimeDataPoint(data.qps);
        }
    },

    addRealtimeDataPoint(value) {
        this.realtimeData.push({
            time: new Date(),
            value: value
        });
        
        if (this.realtimeData.length > this.config.maxRealtimePoints) {
            this.realtimeData.shift();
        }
        
        this.updateRealtimeChart();
        
        const values = this.realtimeData.map(p => p.value);
        document.getElementById('realtimeMin').textContent = Math.min(...values).toFixed(0);
        document.getElementById('realtimeAvg').textContent = (values.reduce((a, b) => a + b, 0) / values.length).toFixed(0);
        document.getElementById('realtimeMax').textContent = Math.max(...values).toFixed(0);
    },

    updateDashboard(data) {
        if (data.summary) {
            this.animateValue('totalRequests', data.summary.total_requests, '%');
            this.animateValue('passRate', data.summary.pass_rate, '%');
            this.animateValue('blockRate', data.summary.block_rate, '%');
            this.animateValue('avgResponseTime', data.summary.avg_response_time, 'ms');
        }
        
        if (data.trend && this.charts.trend) {
            this.updateTrendChart(data.trend);
        }
    },

    updateTrendChart(data) {
        if (!this.charts.trend || !data) return;

        this.charts.trend.setOption({
            xAxis: {
                data: data.map(t => t.time)
            },
            series: [
                {
                    name: '验证请求',
                    data: data.map(t => t.requests)
                },
                {
                    name: '通过数',
                    data: data.map(t => t.passed || Math.floor(t.requests * 0.85))
                },
                {
                    name: '拦截数',
                    data: data.map(t => t.blocked || Math.floor(t.requests * 0.05))
                }
            ]
        }, false);
    },

    addVerificationRow(data) {
        const tbody = document.getElementById('recentVerifications');
        if (!tbody) return;

        const time = new Date(data.timestamp || Date.now()).toLocaleTimeString('zh-CN');
        const row = document.createElement('tr');
        row.innerHTML = `
            <td><small>${time}</small></td>
            <td><small>${this.escapeHtml(data.app || '-')}</small></td>
            <td><small>${this.escapeHtml(data.type || '-')}</small></td>
            <td><span class="badge badge-${data.status === 'success' ? 'success' : 'danger'}">${data.status === 'success' ? '成功' : '失败'}</span></td>
            <td><small>${data.response_time || 0}ms</small></td>
            <td>
                <div class="btn-group btn-group-sm">
                    <button type="button" class="btn btn-outline-primary btn-xs" onclick="EnhancedDashboard.viewDetail('${data.id}')">
                        <i class="fas fa-eye"></i>
                    </button>
                </div>
            </td>
        `;

        tbody.insertBefore(row, tbody.firstChild);

        while (tbody.children.length > 10) {
            tbody.removeChild(tbody.lastChild);
        }
    },

    viewDetail(id) {
        console.log('View detail for:', id);
    },

    setupEventListeners() {
        const refreshBtn = document.getElementById('refreshBtn');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => this.loadData());
        }

        const fullscreenBtn = document.getElementById('fullscreenBtn');
        if (fullscreenBtn) {
            fullscreenBtn.addEventListener('click', () => this.toggleFullscreen());
        }

        document.querySelectorAll('.time-range').forEach(btn => {
            btn.addEventListener('click', (e) => {
                e.preventDefault();
                document.querySelectorAll('.time-range').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                this.currentTimeRange = btn.dataset.range;
                this.saveConfig();
                this.loadData();
            });
        });

        document.querySelectorAll('[data-period]').forEach(btn => {
            btn.addEventListener('click', () => {
                const parent = btn.closest('.btn-group');
                parent.querySelectorAll('[data-period]').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                this.loadTrendData(btn.dataset.period);
            });
        });
    },

    async loadData() {
        try {
            const response = await fetch(`/admin/api/dashboard?range=${this.currentTimeRange}`);
            const result = await response.json();
            
            if (result.code === 0) {
                this.updateDashboard(result.data);
            } else {
                this.loadMockData();
            }
        } catch (error) {
            console.error('Load data failed:', error);
            this.loadMockData();
        }
    },

    loadMockData() {
        const mockData = {
            summary: {
                total_requests: Math.floor(Math.random() * 10000) + 5000,
                pass_rate: (Math.random() * 20 + 80).toFixed(1),
                block_rate: (Math.random() * 10 + 5).toFixed(1),
                avg_response_time: Math.floor(Math.random() * 50) + 20
            },
            trend: this.generateMockTrendData()
        };
        
        this.updateDashboard(mockData);
    },

    generateMockTrendData() {
        const data = [];
        for (let i = 23; i >= 0; i--) {
            const hour = new Date();
            hour.setHours(hour.getHours() - i);
            data.push({
                time: hour.getHours() + ':00',
                requests: Math.floor(Math.random() * 500) + 100,
                passed: Math.floor(Math.random() * 450) + 50,
                blocked: Math.floor(Math.random() * 50) + 10
            });
        }
        return data;
    },

    async loadTrendData(period) {
        try {
            const response = await fetch(`/admin/api/dashboard/trend?period=${period}&range=${this.currentTimeRange}`);
            const result = await response.json();
            
            if (result.code === 0) {
                this.updateTrendChart(result.data);
            }
        } catch (error) {
            console.error('Load trend data failed:', error);
        }
    },

    startAutoRefresh() {
        if (this.autoRefreshInterval) {
            clearInterval(this.autoRefreshInterval);
        }
        
        this.autoRefreshInterval = setInterval(() => {
            this.loadData();
        }, this.config.updateInterval);
    },

    stopAutoRefresh() {
        if (this.autoRefreshInterval) {
            clearInterval(this.autoRefreshInterval);
            this.autoRefreshInterval = null;
        }
    },

    animateValue(elementId, value, suffix = '') {
        const element = document.getElementById(elementId);
        if (!element) return;

        const currentText = element.textContent;
        const current = parseFloat(currentText.replace(/[^\d.]/g, '')) || 0;
        const duration = 1000;
        const startTime = performance.now();

        const update = (currentTime) => {
            const elapsed = currentTime - startTime;
            const progress = Math.min(elapsed / duration, 1);
            const easeProgress = 1 - Math.pow(1 - progress, 4);
            const newValue = current + (value - current) * easeProgress;

            if (suffix === '%') {
                element.textContent = newValue.toFixed(1) + '%';
            } else if (suffix === 'ms') {
                element.textContent = Math.floor(newValue) + 'ms';
            } else {
                element.textContent = this.formatNumber(Math.floor(newValue));
            }

            if (progress < 1) {
                requestAnimationFrame(update);
            }
        };

        requestAnimationFrame(update);
    },

    formatNumber(num) {
        if (num >= 1000000) {
            return (num / 1000000).toFixed(1) + 'M';
        } else if (num >= 1000) {
            return (num / 1000).toFixed(1) + 'K';
        }
        return num.toString();
    },

    formatTime(date) {
        return date.toLocaleTimeString('zh-CN', { 
            hour: '2-digit', 
            minute: '2-digit', 
            second: '2-digit' 
        });
    },

    escapeHtml(text) {
        if (text === null || text === undefined) return '';
        const div = document.createElement('div');
        div.textContent = String(text);
        return div.innerHTML;
    },

    setTheme(theme) {
        if (theme === 'auto') {
            document.documentElement.setAttribute('data-theme', 
                window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
        } else {
            document.documentElement.setAttribute('data-theme', theme);
        }
        localStorage.setItem(this.config.themeKey, theme);
    },

    toggleFullscreen() {
        if (!document.fullscreenElement) {
            document.documentElement.requestFullscreen().catch(err => {
                console.error('Fullscreen request failed:', err);
            });
        } else {
            if (document.exitFullscreen) {
                document.exitFullscreen();
            }
        }
    },

    exportChart(chartName) {
        const chart = this.charts[chartName];
        if (!chart) return;

        try {
            const url = chart.getDataURL({
                type: 'png',
                pixelRatio: 2,
                backgroundColor: '#fff'
            });
            
            const link = document.createElement('a');
            link.download = chartName + '_' + Date.now() + '.png';
            link.href = url;
            document.body.appendChild(link);
            link.click();
            document.body.removeChild(link);
            
            this.showToast('图表导出成功', 'success');
        } catch (error) {
            console.error('图表导出失败:', error);
            this.showToast('图表导出失败', 'error');
        }
    },

    exportData(format) {
        const data = {
            timestamp: new Date().toISOString(),
            timeRange: this.currentTimeRange,
            summary: {
                totalRequests: document.getElementById('totalRequests')?.textContent,
                passRate: document.getElementById('passRate')?.textContent,
                blockRate: document.getElementById('blockRate')?.textContent,
                avgResponseTime: document.getElementById('avgResponseTime')?.textContent
            }
        };
        
        if (format === 'json') {
            this.downloadFile(JSON.stringify(data, null, 2), 'dashboard_export.json', 'application/json');
        } else if (format === 'csv') {
            const csv = '指标,数值\n' +
                       '总验证,' + data.summary.totalRequests + '\n' +
                       '通过率,' + data.summary.passRate + '\n' +
                       '拦截率,' + data.summary.blockRate + '\n' +
                       '平均响应,' + data.summary.avgResponseTime;
            this.downloadFile('\uFEFF' + csv, 'dashboard_export.csv', 'text/csv');
        }
        
        this.showToast('数据导出成功', 'success');
    },

    downloadFile(content, filename, mimeType) {
        const blob = new Blob([content], { type: mimeType });
        const url = URL.createObjectURL(blob);
        const link = document.createElement('a');
        link.href = url;
        link.download = filename;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        URL.revokeObjectURL(url);
    },

    showToast(message, type = 'info') {
        const container = document.querySelector('.toast-container');
        if (!container) return;

        const toast = document.createElement('div');
        toast.className = `toast alert-${type}`;
        toast.setAttribute('role', 'alert');
        toast.innerHTML = `
            <div class="toast-body">
                ${message}
                <button type="button" class="close ml-2" data-dismiss="toast" aria-label="关闭">
                    <span aria-hidden="true">&times;</span>
                </button>
            </div>
        `;

        container.appendChild(toast);
        toast.classList.add('show');

        setTimeout(() => {
            toast.classList.remove('show');
            setTimeout(() => toast.remove(), 300);
        }, 3000);
    }
};

if (typeof module !== 'undefined' && module.exports) {
    module.exports = EnhancedDashboard;
}
