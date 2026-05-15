const { logInfo, logWarning, logError } = require('../middleware/logger');

class WebSocketMonitorService {
  constructor(wsServer) {
    this.wsServer = wsServer;
    this.monitoringConfig = {
      reportInterval: parseInt(process.env.WS_MONITOR_REPORT_INTERVAL) || 10000,
      maxConnectionThreshold: parseInt(process.env.WS_MAX_CONNECTION_THRESHOLD) || 1000,
      maxLatencyThreshold: parseInt(process.env.WS_MAX_LATENCY_THRESHOLD) || 1000,
      alertCooldown: parseInt(process.env.WS_ALERT_COOLDOWN) || 60000
    };
    
    this.alerts = {
      connectionThreshold: false,
      latencyThreshold: false,
      highErrorRate: false
    };
    
    this.lastAlertTime = {
      connectionThreshold: 0,
      latencyThreshold: 0,
      highErrorRate: 0
    };
    
    this.monitoringInterval = null;
    this.startMonitoring();
  }

  startMonitoring() {
    this.monitoringInterval = setInterval(() => {
      this.collectMetrics();
      this.checkThresholds();
      this.generateReport();
    }, this.monitoringConfig.reportInterval);
  }

  collectMetrics() {
    if (!this.wsServer || !this.wsServer.getDetailedMetrics) {
      return null;
    }

    const metrics = this.wsServer.getDetailedMetrics();
    const timestamp = Date.now();

    const collectedMetrics = {
      timestamp,
      connections: {
        current: metrics.currentConnections,
        total: metrics.totalConnections,
        maxConcurrent: this.wsServer.connectionStateMonitor?.maxConcurrentConnections || 0,
        onlineUsers: metrics.onlineUsers
      },
      messages: {
        sent: metrics.messagesSent,
        received: metrics.messagesReceived,
        rate: this.calculateMessageRate(metrics.messagesSent, metrics.messagesReceived)
      },
      heartbeats: {
        sent: metrics.heartbeatMetrics?.heartbeatsSent || 0,
        received: metrics.heartbeatMetrics?.heartbeatsReceived || 0,
        missed: metrics.heartbeatMetrics?.missedHeartbeats || 0,
        active: metrics.heartbeatMetrics?.activeHeartbeats || 0,
        missedRate: this.calculateMissedHeartbeatRate(
          metrics.heartbeatMetrics?.heartbeatsSent || 0,
          metrics.heartbeatMetrics?.missedHeartbeats || 0
        )
      },
      errors: {
        total: metrics.errors,
        errorRate: this.calculateErrorRate(metrics.errors, metrics.totalConnections)
      },
      latency: this.collectLatencyMetrics(),
      rooms: {
        count: metrics.rooms?.length || 0,
        subscriptions: metrics.subscriptions || []
      },
      uptime: metrics.uptime
    };

    return collectedMetrics;
  }

  calculateMessageRate(sent, received) {
    const total = sent + received;
    if (total === 0) return 0;
    return (sent / total) * 100;
  }

  calculateMissedHeartbeatRate(sent, missed) {
    const total = sent + missed;
    if (total === 0) return 0;
    return (missed / total) * 100;
  }

  calculateErrorRate(errors, connections) {
    if (connections === 0) return 0;
    return (errors / connections) * 100;
  }

  collectLatencyMetrics() {
    if (!this.wsServer.metrics.latencySamples || this.wsServer.metrics.latencySamples.length === 0) {
      return {
        p50: 0,
        p95: 0,
        p99: 0,
        avg: 0,
        max: 0,
        samples: 0
      };
    }

    const samples = [...this.wsServer.metrics.latencySamples].sort((a, b) => a - b);
    const sum = samples.reduce((acc, val) => acc + val, 0);
    
    return {
      p50: samples[Math.floor(samples.length * 0.5)],
      p95: samples[Math.floor(samples.length * 0.95)] || samples[samples.length - 1],
      p99: samples[Math.floor(samples.length * 0.99)] || samples[samples.length - 1],
      avg: sum / samples.length,
      max: samples[samples.length - 1],
      samples: samples.length
    };
  }

  checkThresholds() {
    const metrics = this.collectMetrics();
    if (!metrics) return;

    const now = Date.now();

    if (metrics.connections.current > this.monitoringConfig.maxConnectionThreshold) {
      if (now - this.lastAlertTime.connectionThreshold > this.monitoringConfig.alertCooldown) {
        this.alerts.connectionThreshold = true;
        this.lastAlertTime.connectionThreshold = now;
        logger.warn('WebSocket connection threshold exceeded', {
          current: metrics.connections.current,
          threshold: this.monitoringConfig.maxConnectionThreshold
        });
      }
    } else {
      this.alerts.connectionThreshold = false;
    }

    if (metrics.latency.p99 > this.monitoringConfig.maxLatencyThreshold) {
      if (now - this.lastAlertTime.latencyThreshold > this.monitoringConfig.alertCooldown) {
        this.alerts.latencyThreshold = true;
        this.lastAlertTime.latencyThreshold = now;
        logger.warn('WebSocket latency threshold exceeded', {
          p99: metrics.latency.p99,
          threshold: this.monitoringConfig.maxLatencyThreshold
        });
      }
    } else {
      this.alerts.latencyThreshold = false;
    }

    if (metrics.errors.errorRate > 5) {
      if (now - this.lastAlertTime.highErrorRate > this.monitoringConfig.alertCooldown) {
        this.alerts.highErrorRate = true;
        this.lastAlertTime.highErrorRate = now;
        logger.warn('WebSocket high error rate detected', {
          errorRate: metrics.errors.errorRate,
          totalErrors: metrics.errors.total
        });
      }
    } else {
      this.alerts.highErrorRate = false;
    }
  }

  generateReport() {
    const metrics = this.collectMetrics();
    if (!metrics) return;

    logInfo('WebSocket Monitoring Report', {
      connections: {
        current: metrics.connections.current,
        maxConcurrent: metrics.connections.maxConcurrent,
        onlineUsers: metrics.connections.onlineUsers
      },
      messages: {
        total: metrics.messages.sent + metrics.messages.received,
        sentRate: metrics.messages.rate.toFixed(2) + '%'
      },
      latency: {
        p50: metrics.latency.p50 + 'ms',
        p95: metrics.latency.p95 + 'ms',
        p99: metrics.latency.p99 + 'ms'
      },
      heartbeat: {
        missedRate: metrics.heartbeats.missedRate.toFixed(2) + '%',
        active: metrics.heartbeats.active
      },
      errors: {
        total: metrics.errors.total,
        rate: metrics.errors.errorRate.toFixed(2) + '%'
      },
      rooms: {
        count: metrics.rooms.count,
        subscriptions: metrics.rooms.subscriptions.length
      },
      uptime: (metrics.uptime / 1000 / 60).toFixed(2) + ' minutes',
      alerts: this.alerts
    });
  }

  getMonitoringStatus() {
    return {
      isMonitoring: this.monitoringInterval !== null,
      config: this.monitoringConfig,
      alerts: this.alerts,
      metrics: this.collectMetrics()
    };
  }

  stopMonitoring() {
    if (this.monitoringInterval) {
      clearInterval(this.monitoringInterval);
      this.monitoringInterval = null;
      logInfo('WebSocket monitoring stopped');
    }
  }

  resetAlerts() {
    this.alerts = {
      connectionThreshold: false,
      latencyThreshold: false,
      highErrorRate: false
    };
    this.lastAlertTime = {
      connectionThreshold: 0,
      latencyThreshold: 0,
      highErrorRate: 0
    };
  }
}

module.exports = WebSocketMonitorService;
