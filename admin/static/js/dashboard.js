(function() {
  'use strict';

  const Dashboard = {
    charts: {
      trend: null,
      pie: null,
      captchaType: null,
      realtime: null,
      mini: null
    },
    ws: null,
    wsConnected: false,
    realtimeDataPoints: [],
    autoRefreshInterval: null,
    isAutoRefreshEnabled: false,
    previousStats: null,
    MAX_REALTIME_POINTS: 60,
    REALTIME_UPDATE_INTERVAL: 5000,
    AUTO_REFRESH_DELAY: 30000,
    WS_RECONNECT_DELAY: 3000,

    init: function() {
      this.initCharts();
      this.initWebSocket();
      this.setupEventListeners();
      this.loadDashboardData();
      this.loadSystemStatus();
      this.loadRecentVerifications();
      this.startAutoRefresh();
      this.initResponsiveHandlers();
      this.showLoadingState();
    },

    initCharts: function() {
      this.initTrendChart();
      this.initPieChart();
      this.initCaptchaTypeChart();
      this.initRealtimeChart();
      this.initMiniChart();

      if (typeof ThemeManager !== 'undefined') {
        ThemeManager.addThemeChangeListener((data) => {
          this.updateChartsTheme(data.theme);
        });
      }
    },

    initResponsiveHandlers: function() {
      let resizeTimer;
      
      window.addEventListener('resize', () => {
        clearTimeout(resizeTimer);
        resizeTimer = setTimeout(() => {
          this.resizeAllCharts();
          this.adjustLayoutForScreenSize();
        }, 250);
      });

      window.addEventListener('orientationchange', () => {
        setTimeout(() => {
          this.resizeAllCharts();
        }, 100);
      });
    },

    adjustLayoutForScreenSize: function() {
      const width = window.innerWidth;
      const charts = ['trendChart', 'pieChart', 'captchaTypeChart', 'realtimeChart'];
      const chartHeights = {
        trendChart: 300,
        pieChart: 250,
        captchaTypeChart: 250,
        realtimeChart: 200
      };

      if (width < 576) {
        Object.keys(chartHeights).forEach(chartName => {
          const chartInstance = this.charts[chartName.replace('Chart', '').toLowerCase()];
          if (chartInstance) {
            chartInstance.resize({ height: chartHeights[chartName] * 0.6 });
          }
        });
      } else if (width < 768) {
        Object.keys(chartHeights).forEach(chartName => {
          const chartInstance = this.charts[chartName.replace('Chart', '').toLowerCase()];
          if (chartInstance) {
            chartInstance.resize({ height: chartHeights[chartName] * 0.8 });
          }
        });
      } else {
        this.resizeAllCharts();
      }
    },

    initTrendChart: function() {
      const container = document.getElementById('trendChart');
      if (!container) return;

      this.charts.trend = echarts.init(container);
      this.charts.trend.setOption(this.getTrendChartOption());
    },

    initPieChart: function() {
      const container = document.getElementById('pieChart');
      if (!container) return;

      this.charts.pie = echarts.init(container);
      this.charts.pie.setOption(this.getPieChartOption());
    },

    initCaptchaTypeChart: function() {
      const container = document.getElementById('captchaTypeChart');
      if (!container) return;

      this.charts.captchaType = echarts.init(container);
      this.charts.captchaType.setOption(this.getCaptchaTypeChartOption());
    },

    initRealtimeChart: function() {
      const container = document.getElementById('realtimeChart');
      if (!container) return;

      this.realtimeDataPoints = Array(this.MAX_REALTIME_POINTS).fill(0).map((_, i) => ({
        time: this.formatTime(new Date(Date.now() - (this.MAX_REALTIME_POINTS - i) * 1000)),
        value: 0
      }));

      this.charts.realtime = echarts.init(container);
      this.charts.realtime.setOption(this.getRealtimeChartOption());
    },

    initMiniChart: function() {
      const container = document.getElementById('requestsMiniChart');
      if (!container) return;

      this.charts.mini = echarts.init(container);
      this.charts.mini.setOption(this.getMiniChartOption());
    },

    getChartColors: function() {
      if (typeof ThemeManager !== 'undefined') {
        const colors = ThemeManager.getThemeColors();
        return {
          text: colors.text,
          primary: colors.primary,
          success: colors.success,
          warning: colors.warning,
          danger: colors.danger,
          grid: colors.chart.grid,
          axis: colors.chart.axis
        };
      }

      const isDark = document.documentElement.getAttribute('data-theme') === 'dark';
      return {
        text: isDark ? '#e9ecef' : '#333333',
        primary: isDark ? '#4a9eff' : '#007bff',
        success: isDark ? '#2fd56a' : '#28a745',
        warning: isDark ? '#ffc107' : '#ffc107',
        danger: isDark ? '#ff4757' : '#dc3545',
        grid: isDark ? '#3d434a' : '#e0e0e0',
        axis: isDark ? '#e9ecef' : '#666666'
      };
    },

    getTrendChartOption: function() {
      const colors = this.getChartColors();
      return {
        color: [colors.primary],
        tooltip: {
          trigger: 'axis',
          backgroundColor: colors.chart.tooltip,
          textStyle: { color: '#fff' },
          axisPointer: { type: 'cross' }
        },
        grid: {
          left: '3%',
          right: '4%',
          bottom: '10%',
          top: '10%',
          containLabel: true
        },
        xAxis: {
          type: 'category',
          boundaryGap: false,
          data: [],
          axisLabel: { color: colors.axis, rotate: 45 },
          axisLine: { lineStyle: { color: colors.grid } },
          splitLine: { show: false }
        },
        yAxis: {
          type: 'value',
          axisLabel: { color: colors.axis },
          axisLine: { lineStyle: { color: colors.grid } },
          splitLine: { lineStyle: { color: colors.grid, type: 'dashed' } }
        },
        series: [{
          name: '验证请求',
          type: 'line',
          smooth: true,
          symbol: 'circle',
          symbolSize: 6,
          sampling: 'lttb',
          itemStyle: {
            color: colors.primary,
            borderWidth: 2
          },
          areaStyle: {
            color: {
              type: 'linear',
              x: 0, y: 0, x2: 0, y2: 1,
              colorStops: [
                { offset: 0, color: colors.primary + '40' },
                { offset: 1, color: colors.primary + '05' }
              ]
            }
          },
          lineStyle: { width: 2 },
          data: []
        }],
        animation: true,
        animationDuration: 800,
        animationEasing: 'cubicOut'
      };
    },

    getPieChartOption: function() {
      const colors = ['#28a745', '#ffc107', '#fd7e14', '#dc3545'];
      return {
        tooltip: {
          trigger: 'item',
          backgroundColor: 'rgba(0,0,0,0.8)',
          textStyle: { color: '#fff' },
          formatter: '{b}: {c} ({d}%)'
        },
        legend: {
          orient: 'vertical',
          left: 'left',
          top: 'middle',
          textStyle: { color: '#666' }
        },
        series: [{
          type: 'pie',
          radius: ['40%', '70%'],
          center: ['60%', '50%'],
          avoidLabelOverlap: true,
          itemStyle: {
            borderRadius: 4,
            borderColor: '#fff',
            borderWidth: 2
          },
          label: {
            show: true,
            formatter: '{b}: {d}%',
            color: '#666'
          },
          emphasis: {
            itemStyle: {
              shadowBlur: 10,
              shadowOffsetX: 0,
              shadowColor: 'rgba(0, 0, 0, 0.5)'
            },
            label: {
              show: true,
              fontSize: 14,
              fontWeight: 'bold'
            }
          },
          data: []
        }],
        animation: true,
        animationDuration: 1000,
        animationEasing: 'cubicInOut'
      };
    },

    getCaptchaTypeChartOption: function() {
      const colors = this.getChartColors();
      return {
        color: [colors.primary, colors.success, colors.warning, colors.danger],
        tooltip: {
          trigger: 'axis',
          backgroundColor: 'rgba(0,0,0,0.8)',
          textStyle: { color: '#fff' }
        },
        grid: {
          left: '3%',
          right: '4%',
          bottom: '10%',
          top: '10%',
          containLabel: true
        },
        xAxis: {
          type: 'category',
          data: [],
          axisLabel: { color: colors.axis, rotate: 30 },
          axisLine: { lineStyle: { color: colors.grid } }
        },
        yAxis: {
          type: 'value',
          axisLabel: { color: colors.axis },
          axisLine: { lineStyle: { color: colors.grid } },
          splitLine: { lineStyle: { color: colors.grid, type: 'dashed' } }
        },
        series: [{
          type: 'bar',
          barWidth: '60%',
          itemStyle: {
            borderRadius: [4, 4, 0, 0],
            color: {
              type: 'linear',
              x: 0, y: 0, x2: 0, y2: 1,
              colorStops: [
                { offset: 0, color: colors.primary },
                { offset: 1, color: colors.primary + '80' }
              ]
            }
          },
          data: [],
          animationDelay: function(idx) {
            return idx * 50;
          }
        }],
        animation: true,
        animationDuration: 1000,
        animationEasing: 'elasticOut'
      };
    },

    getRealtimeChartOption: function() {
      const colors = this.getChartColors();
      return {
        tooltip: {
          trigger: 'axis',
          backgroundColor: 'rgba(0,0,0,0.8)',
          textStyle: { color: '#fff' },
          formatter: '{time}<br/>{value} QPS'
        },
        grid: {
          left: '3%',
          right: '4%',
          bottom: '10%',
          top: '10%',
          containLabel: true
        },
        xAxis: {
          type: 'category',
          data: [],
          axisLabel: { color: colors.axis, rotate: 45 },
          boundaryGap: false,
          axisLine: { lineStyle: { color: colors.grid } }
        },
        yAxis: {
          type: 'value',
          axisLabel: { color: colors.axis },
          axisLine: { lineStyle: { color: colors.grid } },
          splitLine: { lineStyle: { color: colors.grid, type: 'dashed' } }
        },
        series: [{
          type: 'line',
          smooth: true,
          symbol: 'none',
          sampling: 'lttb',
          lineStyle: { width: 2, color: colors.success },
          areaStyle: {
            color: {
              type: 'linear',
              x: 0, y: 0, x2: 0, y2: 1,
              colorStops: [
                { offset: 0, color: colors.success + '40' },
                { offset: 1, color: colors.success + '05' }
              ]
            }
          },
          data: [],
          animation: false
        }],
        animation: false
      };
    },

    getMiniChartOption: function() {
      return {
        xAxis: {
          show: false,
          type: 'category',
          data: []
        },
        yAxis: {
          show: false,
          type: 'value'
        },
        series: [{
          type: 'line',
          smooth: true,
          symbol: 'none',
          lineStyle: {
            color: 'rgba(255,255,255,0.6)',
            width: 1.5
          },
          areaStyle: {
            color: 'rgba(255,255,255,0.25)'
          },
          data: []
        }],
        grid: {
          left: 0,
          right: 0,
          top: 2,
          bottom: 2
        },
        animation: false
      };
    },

    updateChartsTheme: function(theme) {
      const colors = this.getChartColors();

      if (this.charts.trend) {
        this.charts.trend.setOption({
          xAxis: {
            axisLabel: { color: colors.axis },
            axisLine: { lineStyle: { color: colors.grid } }
          },
          yAxis: {
            axisLabel: { color: colors.axis },
            axisLine: { lineStyle: { color: colors.grid } },
            splitLine: { lineStyle: { color: colors.grid, type: 'dashed' } }
          }
        }, false);
      }

      if (this.charts.captchaType) {
        this.charts.captchaType.setOption({
          xAxis: {
            axisLabel: { color: colors.axis },
            axisLine: { lineStyle: { color: colors.grid } }
          },
          yAxis: {
            axisLabel: { color: colors.axis },
            axisLine: { lineStyle: { color: colors.grid } },
            splitLine: { lineStyle: { color: colors.grid, type: 'dashed' } }
          }
        }, false);
      }

      this.resizeAllCharts();
    },

    resizeAllCharts: function() {
      Object.values(this.charts).forEach(chart => {
        if (chart && typeof chart.resize === 'function') {
          chart.resize();
        }
      });
    },

    showLoadingState: function() {
      const cards = document.querySelectorAll('.card');
      cards.forEach(card => {
        const loadingOverlay = document.createElement('div');
        loadingOverlay.className = 'chart-loading-overlay';
        loadingOverlay.innerHTML = '<div class="spinner-border text-primary" role="status"><span class="sr-only">加载中...</span></div>';
        loadingOverlay.style.cssText = 'position:absolute;top:0;left:0;right:0;bottom:0;background:rgba(255,255,255,0.7);display:flex;align-items:center;justify-content:center;z-index:10;';
        card.style.position = 'relative';
        card.appendChild(loadingOverlay);
      });
    },

    hideLoadingState: function() {
      const overlays = document.querySelectorAll('.chart-loading-overlay');
      overlays.forEach(overlay => {
        overlay.style.opacity = '0';
        overlay.style.transition = 'opacity 0.3s ease';
        setTimeout(() => overlay.remove(), 300);
      });
    },

    initWebSocket: function() {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const wsUrl = `${protocol}//${window.location.host}/api/v1/admin/dashboard/ws`;

      try {
        this.ws = new WebSocket(wsUrl);

        this.ws.onopen = () => {
          this.wsConnected = true;
          this.updateWsStatus(true);
          console.log('WebSocket connected');
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
          console.log('WebSocket disconnected, reconnecting in', this.WS_RECONNECT_DELAY, 'ms...');
          setTimeout(() => this.initWebSocket(), this.WS_RECONNECT_DELAY);
        };
      } catch (e) {
        console.error('WebSocket init failed:', e);
        this.wsConnected = false;
        this.updateWsStatus(false);
        setTimeout(() => this.initWebSocket(), this.WS_RECONNECT_DELAY);
      }
    },

    updateWsStatus: function(connected) {
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

    handleRealtimeData: function(data) {
      if (data.type === 'metrics') {
        this.updateRealtimeMetrics(data.payload);
      } else if (data.type === 'verification') {
        this.addVerificationRow(data.payload);
      } else if (data.type === 'stats') {
        this.updateDashboard(data.payload);
      }
    },

    updateRealtimeMetrics: function(data) {
      if (data.total_requests !== undefined) {
        this.animateValue('totalRequests', 0, data.total_requests, 1000);
      }

      if (data.pass_rate !== undefined) {
        document.getElementById('passRate').textContent = data.pass_rate.toFixed(1) + '%';
        document.getElementById('passRateProgress').style.width = data.pass_rate + '%';
      }

      if (data.block_rate !== undefined) {
        document.getElementById('blockRate').textContent = data.block_rate.toFixed(1) + '%';
        document.getElementById('blockRateProgress').style.width = data.block_rate + '%';
      }

      if (data.avg_response_time !== undefined) {
        document.getElementById('avgResponseTime').textContent = data.avg_response_time + 'ms';
      }

      if (data.qps !== undefined) {
        document.getElementById('currentQPS').textContent = data.qps.toFixed(0) + ' QPS';
        this.updateRealtimeChart(data.qps);
      }

      if (data.recent_requests !== undefined) {
        this.updateMiniChart(data.recent_requests);
      }

      if (data.system_status !== undefined) {
        this.updateSystemStatus(data.system_status);
      }
    },

    updateRealtimeChart: function(value) {
      if (!this.charts.realtime) return;

      const now = new Date();
      const timeLabel = this.formatTime(now);

      this.realtimeDataPoints.push({ time: timeLabel, value: value });

      if (this.realtimeDataPoints.length > this.MAX_REALTIME_POINTS) {
        this.realtimeDataPoints.shift();
      }

      this.charts.realtime.setOption({
        xAxis: {
          data: this.realtimeDataPoints.map(p => p.time)
        },
        series: [{
          data: this.realtimeDataPoints.map(p => p.value)
        }]
      }, false);
    },

    updateMiniChart: function(data) {
      if (!this.charts.mini || !data || data.length === 0) return;

      this.charts.mini.setOption({
        xAxis: { data: data.map((_, i) => i) },
        series: [{
          data: data,
          smooth: true
        }]
      }, false);
    },

    setupEventListeners: function() {
      const refreshBtn = document.getElementById('refreshBtn');
      if (refreshBtn) {
        refreshBtn.addEventListener('click', async () => {
          refreshBtn.disabled = true;
          const icon = refreshBtn.querySelector('i');
          if (icon) {
            icon.classList.add('fa-spin');
          }

          await Promise.all([
            this.loadDashboardData(),
            this.loadSystemStatus(),
            this.loadRecentVerifications()
          ]);

          if (icon) {
            icon.classList.remove('fa-spin');
          }
          refreshBtn.disabled = false;

          if (typeof showAdminToast === 'function') {
            showAdminToast('数据已刷新', 'success');
          }
        });
      }

      const autoRefreshBtn = document.getElementById('autoRefreshBtn');
      if (autoRefreshBtn) {
        autoRefreshBtn.addEventListener('click', () => this.toggleAutoRefresh());
      }

      const fullscreenBtn = document.getElementById('fullscreenBtn');
      if (fullscreenBtn) {
        fullscreenBtn.addEventListener('click', () => this.toggleFullscreen());
      }

      document.querySelectorAll('[data-period]').forEach(btn => {
        btn.addEventListener('click', async (e) => {
          document.querySelectorAll('[data-period]').forEach(b => b.classList.remove('active'));
          e.target.classList.add('active');
          await this.loadTrendData(e.target.dataset.period);
        });
      });
    },

    toggleAutoRefresh: function() {
      this.isAutoRefreshEnabled = !this.isAutoRefreshEnabled;
      const statusEl = document.getElementById('autoRefreshStatus');
      const btnEl = document.getElementById('autoRefreshBtn');

      if (this.isAutoRefreshEnabled) {
        statusEl.textContent = '开启';
        btnEl.classList.add('btn-success');
        btnEl.classList.remove('btn-default');
        this.startAutoRefresh();
        if (typeof showAdminToast === 'function') {
          showAdminToast('自动刷新已开启（每30秒）', 'info');
        }
      } else {
        statusEl.textContent = '关闭';
        btnEl.classList.remove('btn-success');
        btnEl.classList.add('btn-default');
        this.stopAutoRefresh();
        if (typeof showAdminToast === 'function') {
          showAdminToast('自动刷新已关闭', 'info');
        }
      }
    },

    startAutoRefresh: function() {
      if (this.autoRefreshInterval) {
        clearInterval(this.autoRefreshInterval);
      }

      this.autoRefreshInterval = setInterval(async () => {
        await Promise.all([
          this.loadDashboardData(),
          this.loadSystemStatus()
        ]);
      }, this.AUTO_REFRESH_DELAY);
    },

    stopAutoRefresh: function() {
      if (this.autoRefreshInterval) {
        clearInterval(this.autoRefreshInterval);
        this.autoRefreshInterval = null;
      }
    },

    async loadDashboardData() {
      try {
        const response = await fetch('/admin/api/dashboard');
        if (!response.ok) throw new Error('Network error');

        const result = await response.json();
        if (result.code === 0) {
          this.updateDashboard(result.data);
        } else {
          this.loadMockData();
        }
      } catch (error) {
        console.error('Dashboard data load failed:', error);
        this.loadMockData();
      }
    },

    loadMockData: function() {
      const mockData = {
        summary: {
          total_requests: Math.floor(Math.random() * 10000) + 5000,
          pass_rate: (Math.random() * 20 + 80).toFixed(1),
          block_rate: (Math.random() * 10 + 5).toFixed(1),
          avg_response_time: Math.floor(Math.random() * 50) + 20
        },
        trend: this.generateMockTrendData(),
        risk_distribution: {
          low: Math.floor(Math.random() * 5000) + 3000,
          medium: Math.floor(Math.random() * 2000) + 500,
          high: Math.floor(Math.random() * 500) + 100,
          critical: Math.floor(Math.random() * 100) + 20
        },
        captcha_type: [
          { type: '滑动验证', count: Math.floor(Math.random() * 3000) + 2000 },
          { type: '点选验证', count: Math.floor(Math.random() * 2000) + 1000 },
          { type: '图片验证', count: Math.floor(Math.random() * 1500) + 500 },
          { type: '文字验证', count: Math.floor(Math.random() * 1000) + 200 }
        ]
      };
      this.updateDashboard(mockData);
    },

    generateMockTrendData: function() {
      const data = [];
      for (let i = 23; i >= 0; i--) {
        const hour = new Date();
        hour.setHours(hour.getHours() - i);
        data.push({
          time: hour.getHours() + ':00',
          requests: Math.floor(Math.random() * 500) + 100
        });
      }
      return data;
    },

    updateDashboard: function(data) {
      this.hideLoadingState();

      if (data.summary) {
        this.animateValue('totalRequests', 0, data.summary.total_requests, 1000);
        document.getElementById('passRate').textContent = data.summary.pass_rate + '%';
        document.getElementById('blockRate').textContent = data.summary.block_rate + '%';
        document.getElementById('avgResponseTime').textContent = data.summary.avg_response_time + 'ms';
        document.getElementById('passRateProgress').style.width = data.summary.pass_rate + '%';
        document.getElementById('blockRateProgress').style.width = data.summary.block_rate + '%';
      }

      if (data.trend && this.charts.trend) {
        const labels = data.trend.map(t => t.time);
        const values = data.trend.map(t => t.requests);

        this.charts.trend.setOption({
          xAxis: { data: labels },
          series: [{ data: values }]
        }, false);
      }

      if (data.risk_distribution && this.charts.pie) {
        const riskData = [
          { value: data.risk_distribution.low || 0, name: '低风险' },
          { value: data.risk_distribution.medium || 0, name: '中风险' },
          { value: data.risk_distribution.high || 0, name: '高风险' },
          { value: data.risk_distribution.critical || 0, name: '极高风险' }
        ];

        this.charts.pie.setOption({
          series: [{ data: riskData }]
        }, false);

        const legendHtml = riskData.map(item => {
          const colors = ['success', 'warning', 'orange', 'danger'];
          const colorIndex = riskData.indexOf(item);
          return `<span class="mr-3"><i class="fas fa-circle text-${colors[colorIndex]}"></i> ${item.name}: ${item.value}</span>`;
        }).join('');
        document.getElementById('riskLegend').innerHTML = legendHtml;
      }

      if (data.captcha_type && this.charts.captchaType) {
        this.charts.captchaType.setOption({
          xAxis: {
            data: data.captcha_type.map(c => c.type)
          },
          series: [{
            data: data.captcha_type.map(c => c.count)
          }]
        }, false);
      }
    },

    async loadTrendData: function(period) {
      try {
        const response = await fetch(`/admin/api/dashboard/trend?period=${period}`);
        if (!response.ok) throw new Error('Network error');

        const result = await response.json();
        if (result.code === 0) {
          this.updateTrendChart(result.data, period);
        }
      } catch (error) {
        console.error('Trend data load failed:', error);
      }
    },

    updateTrendChart: function(data, period) {
      if (!this.charts.trend || !data) return;

      const labels = period === 'hour' ?
        data.map(t => t.time) :
        data.map(t => t.day || t.date || t.label);
      const values = data.map(t => t.requests || t.count);

      this.charts.trend.setOption({
        xAxis: {
          data: labels,
          axisLabel: { rotate: period === 'day' ? 45 : 0 }
        },
        series: [{ data: values }]
      }, false);
    },

    async loadSystemStatus: function() {
      try {
        const response = await fetch('/admin/api/system-status');
        if (!response.ok) throw new Error('Network error');

        const result = await response.json();
        if (result.code === 0) {
          this.updateSystemStatus(result.data);
        } else {
          this.loadMockSystemStatus();
        }
      } catch (error) {
        console.error('System status load failed:', error);
        this.loadMockSystemStatus();
      }
    },

    loadMockSystemStatus: function() {
      this.updateSystemStatus({
        database: { status: 'healthy', latency: Math.floor(Math.random() * 50) + 10 },
        redis: { status: 'healthy', latency: Math.floor(Math.random() * 10) + 1 },
        api: { status: 'healthy', latency: Math.floor(Math.random() * 100) + 20 },
        storage: { status: 'healthy', latency: Math.floor(Math.random() * 30) + 5 },
        cpu: Math.floor(Math.random() * 40) + 20,
        memory: Math.floor(Math.random() * 30) + 40,
        disk: Math.floor(Math.random() * 20) + 50
      });
    },

    updateSystemStatus: function(data) {
      if (data.database) {
        document.getElementById('dbLatency').textContent = data.database.latency + 'ms';
        document.getElementById('dbStatus').className = 'info-box-icon ' +
          (data.database.status === 'healthy' ? 'bg-success' : 'bg-danger');
      }

      if (data.redis) {
        document.getElementById('redisLatency').textContent = data.redis.latency + 'ms';
        document.getElementById('redisStatus').className = 'info-box-icon ' +
          (data.redis.status === 'healthy' ? 'bg-success' : 'bg-danger');
      }

      if (data.api) {
        document.getElementById('apiLatency').textContent = data.api.latency + 'ms';
        document.getElementById('apiStatus').className = 'info-box-icon ' +
          (data.api.status === 'healthy' ? 'bg-success' : 'bg-danger');
      }

      if (data.storage) {
        document.getElementById('storageLatency').textContent = data.storage.latency + 'ms';
        document.getElementById('storageStatus').className = 'info-box-icon ' +
          (data.storage.status === 'healthy' ? 'bg-success' : 'bg-danger');
      }

      if (data.cpu !== undefined) {
        document.getElementById('cpuUsage').textContent = data.cpu + '%';
        document.getElementById('cpuProgress').style.width = data.cpu + '%';
        const cpuBar = document.getElementById('cpuProgress');
        cpuBar.className = 'progress-bar ' +
          (data.cpu > 80 ? 'bg-danger' : data.cpu > 60 ? 'bg-warning' : 'bg-primary');
      }

      if (data.memory !== undefined) {
        document.getElementById('memUsage').textContent = data.memory + '%';
        document.getElementById('memProgress').style.width = data.memory + '%';
        const memBar = document.getElementById('memProgress');
        memBar.className = 'progress-bar ' +
          (data.memory > 80 ? 'bg-danger' : data.memory > 60 ? 'bg-warning' : 'bg-success');
      }

      if (data.disk !== undefined) {
        document.getElementById('diskUsage').textContent = data.disk + '%';
        document.getElementById('diskProgress').style.width = data.disk + '%';
      }
    },

    async loadRecentVerifications: function() {
      try {
        const response = await fetch('/admin/api/recent-verifications');
        if (!response.ok) throw new Error('Network error');

        const result = await response.json();
        if (result.code === 0) {
          this.renderRecentVerifications(result.data);
        } else {
          this.renderMockRecentVerifications();
        }
      } catch (error) {
        console.error('Recent verifications load failed:', error);
        this.renderMockRecentVerifications();
      }
    },

    renderMockRecentVerifications: function() {
      const mockData = [];
      for (let i = 0; i < 10; i++) {
        const date = new Date();
        date.setMinutes(date.getMinutes() - i * 5);
        mockData.push({
          time: date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' }),
          app: 'App_' + (Math.floor(Math.random() * 10) + 1),
          type: ['滑动验证', '点选验证', '图片验证'][Math.floor(Math.random() * 3)],
          status: Math.random() > 0.1 ? 'success' : 'failed',
          response_time: Math.floor(Math.random() * 100) + 20
        });
      }
      this.renderRecentVerifications(mockData);
    },

    renderRecentVerifications: function(data) {
      const tbody = document.getElementById('recentVerifications');
      if (!tbody || !data || data.length === 0) {
        tbody.innerHTML = '<tr><td colspan="5" class="text-center text-muted">暂无数据</td></tr>';
        return;
      }

      tbody.innerHTML = data.slice(0, 10).map(item => `
        <tr class="fade-in">
          <td><small>${item.time || '-'}</small></td>
          <td><small>${this.escapeHtml(item.app || '-')}</small></td>
          <td><small>${this.escapeHtml(item.type || '-')}</small></td>
          <td><span class="badge badge-${item.status === 'success' ? 'success' : 'danger'}">${item.status === 'success' ? '成功' : '失败'}</span></td>
          <td><small>${item.response_time || 0}ms</small></td>
        </tr>
      `).join('');
    },

    addVerificationRow: function(data) {
      const tbody = document.getElementById('recentVerifications');
      if (!tbody) return;

      const time = new Date(data.timestamp || Date.now()).toLocaleTimeString('zh-CN');
      const row = document.createElement('tr');
      row.className = 'fade-in';
      row.innerHTML = `
        <td><small>${time}</small></td>
        <td><small>${this.escapeHtml(data.app || '-')}</small></td>
        <td><small>${this.escapeHtml(data.type || '-')}</small></td>
        <td><span class="badge badge-${data.status === 'success' ? 'success' : 'danger'}">${data.status === 'success' ? '成功' : '失败'}</span></td>
        <td><small>${data.response_time || 0}ms</small></td>
      `;

      tbody.insertBefore(row, tbody.firstChild);

      while (tbody.children.length > 10) {
        tbody.removeChild(tbody.lastChild);
      }
    },

    animateValue: function(elementId, start, end, duration) {
      const element = document.getElementById(elementId);
      if (!element) return;

      const startTime = performance.now();

      const update = (currentTime) => {
        const elapsed = currentTime - startTime;
        const progress = Math.min(elapsed / duration, 1);
        const easeProgress = 1 - Math.pow(1 - progress, 4);
        const value = Math.floor(start + (end - start) * easeProgress);
        element.textContent = this.formatNumber(value);

        if (progress < 1) {
          requestAnimationFrame(update);
        }
      };

      requestAnimationFrame(update);
    },

    formatNumber: function(num) {
      if (num >= 1000000) {
        return (num / 1000000).toFixed(1) + 'M';
      } else if (num >= 1000) {
        return (num / 1000).toFixed(1) + 'K';
      }
      return num.toString();
    },

    formatTime: function(date) {
      return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
    },

    escapeHtml: function(text) {
      if (text === null || text === undefined) return '';
      const div = document.createElement('div');
      div.textContent = String(text);
      return div.innerHTML;
    },

    toggleFullscreen: function() {
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

    exportData: function(format) {
      const data = this.getExportData();

      if (format === 'csv') {
        this.exportCSV(data);
      } else if (format === 'excel') {
        this.exportExcel(data);
      } else if (format === 'json') {
        this.exportJSON(data);
      }
    },

    getExportData: function() {
      return {
        summary: {
          total_requests: document.getElementById('totalRequests').textContent,
          pass_rate: document.getElementById('passRate').textContent,
          block_rate: document.getElementById('blockRate').textContent,
          avg_response_time: document.getElementById('avgResponseTime').textContent
        },
        timestamp: new Date().toISOString()
      };
    },

    exportCSV: function(data) {
      const csvContent = 'data:text/csv;charset=utf-8,\uFEFF';
      const headers = ['指标', '数值'];
      const rows = [
        headers,
        ['总验证数', data.summary.total_requests],
        ['通过率', data.summary.pass_rate],
        ['拦截率', data.summary.block_rate],
        ['平均响应时间', data.summary.avg_response_time],
        ['导出时间', data.timestamp]
      ];

      const csv = csvContent + rows.map(row => row.join(',')).join('\n');
      this.downloadFile(csv, 'dashboard_export.csv', 'text/csv');
    },

    exportExcel: function(data) {
      let excelContent = '<?xml version="1.0" encoding="UTF-8"?>\n';
      excelContent += '<Workbook xmlns="urn:schemas-microsoft-com:office:spreadsheet">\n';
      excelContent += '<Worksheet ss:Name="Dashboard Data">\n';
      excelContent += '<Table>\n';

      excelContent += '<Row><Cell><Data ss:Type="String">指标</Data></Cell><Cell><Data ss:Type="String">数值</Data></Cell></Row>\n';
      excelContent += `<Row><Cell><Data ss:Type="String">总验证数</Data></Cell><Cell><Data ss:Type="String">${data.summary.total_requests}</Data></Cell></Row>\n`;
      excelContent += `<Row><Cell><Data ss:Type="String">通过率</Data></Cell><Cell><Data ss:Type="String">${data.summary.pass_rate}</Data></Cell></Row>\n`;
      excelContent += `<Row><Cell><Data ss:Type="String">拦截率</Data></Cell><Cell><Data ss:Type="String">${data.summary.block_rate}</Data></Cell></Row>\n`;
      excelContent += `<Row><Cell><Data ss:Type="String">平均响应时间</Data></Cell><Cell><Data ss:Type="String">${data.summary.avg_response_time}</Data></Cell></Row>\n`;
      excelContent += `<Row><Cell><Data ss:Type="String">导出时间</Data></Cell><Cell><Data ss:Type="String">${data.timestamp}</Data></Cell></Row>\n`;

      excelContent += '</Table>\n</Worksheet>\n</Workbook>';

      this.downloadFile(excelContent, 'dashboard_export.xls', 'application/vnd.ms-excel');
    },

    exportJSON: function(data) {
      const jsonContent = JSON.stringify(data, null, 2);
      this.downloadFile(jsonContent, 'dashboard_export.json', 'application/json');
    },

    downloadFile: function(content, filename, mimeType) {
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

    destroy: function() {
      this.stopAutoRefresh();

      if (this.ws) {
        this.ws.close();
        this.ws = null;
      }

      Object.values(this.charts).forEach(chart => {
        if (chart && typeof chart.dispose === 'function') {
          chart.dispose();
        }
      });

      if (typeof ThemeManager !== 'undefined') {
        ThemeManager.removeThemeChangeListener();
      }

      console.log('Dashboard destroyed');
    }
  };

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => Dashboard.init());
  } else {
    Dashboard.init();
  }

  if (typeof window !== 'undefined') {
    window.Dashboard = Dashboard;
  }
})();

function exportData(format) {
  if (typeof Dashboard !== 'undefined') {
    Dashboard.exportData(format);
  }
}
