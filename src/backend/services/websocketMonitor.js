const { logger } = require('../../backend/middleware/logger');

class WebSocketMonitor {
  constructor() {
    this.metrics = {
      connections: {
        total: 0,
        active: 0,
        maxConcurrent: 0,
        connectionHistory: [],
        peakTime: null
      },
      messages: {
        sent: 0,
        received: 0,
        broadcast: 0,
        byType: {},
        totalSize: 0
      },
      performance: {
        averageLatency: 0,
        minLatency: Infinity,
        maxLatency: 0,
        latencySamples: [],
        messageProcessingTime: []
      },
      errors: {
        total: 0,
        byType: {},
        recent: []
      },
      rooms: {
        activeRooms: new Set(),
        totalSubscriptions: 0,
        subscriptionsByRoom: {}
      },
      health: {
        uptime: 0,
        restarts: 0,
        lastError: null,
        lastErrorTime: null
      }
    };

    this.startTime = Date.now();
    this.maxLatencySamples = 1000;
    this.maxErrorHistory = 100;
    this.maxConnectionHistory = 100;
    this.alertThresholds = {
      maxConnections: 1000,
      maxLatency: 5000,
      maxErrorRate: 10,
      maxMemoryGrowth: 100 * 1024 * 1024
    };

    this.monitors = new Map();
    this.alerts = [];
    this.alertCallbacks = [];
  }

  recordConnection(socketId, userId, metadata = {}) {
    const now = Date.now();
    this.metrics.connections.total++;
    this.metrics.connections.active++;

    if (this.metrics.connections.active > this.metrics.connections.maxConcurrent) {
      this.metrics.connections.maxConcurrent = this.metrics.connections.active;
      this.metrics.connections.peakTime = now;
    }

    this.metrics.connections.connectionHistory.push({
      type: 'connect',
      socketId,
      userId,
      timestamp: now,
      metadata
    });

    if (this.metrics.connections.connectionHistory.length > this.maxConnectionHistory) {
      this.metrics.connections.connectionHistory.shift();
    }

    this.checkThresholds();

    logger.debug('WebSocket connection recorded', {
      socketId,
      userId,
      activeConnections: this.metrics.connections.active
    });
  }

  recordDisconnection(socketId, userId, reason, metadata = {}) {
    const now = Date.now();
    this.metrics.connections.active--;

    this.metrics.connections.connectionHistory.push({
      type: 'disconnect',
      socketId,
      userId,
      reason,
      timestamp: now,
      metadata
    });

    if (this.metrics.connections.connectionHistory.length > this.maxConnectionHistory) {
      this.metrics.connections.connectionHistory.shift();
    }

    logger.debug('WebSocket disconnection recorded', {
      socketId,
      userId,
      reason,
      activeConnections: this.metrics.connections.active
    });
  }

  recordMessage(direction, messageType, size = 0, latency = 0) {
    const now = Date.now();

    if (direction === 'sent') {
      this.metrics.messages.sent++;
    } else {
      this.metrics.messages.received++;
    }

    if (messageType === 'broadcast') {
      this.metrics.messages.broadcast++;
    }

    this.metrics.messages.byType[messageType] = (this.metrics.messages.byType[messageType] || 0) + 1;
    this.metrics.messages.totalSize += size;

    if (latency > 0) {
      this.recordLatency(latency);
    }

    if (size > 0) {
      this.metrics.performance.messageProcessingTime.push({
        timestamp: now,
        size,
        duration: latency
      });

      if (this.metrics.performance.messageProcessingTime.length > this.maxLatencySamples) {
        this.metrics.performance.messageProcessingTime.shift();
      }
    }
  }

  recordLatency(latency) {
    this.metrics.performance.latencySamples.push(latency);

    if (this.metrics.performance.latencySamples.length > this.maxLatencySamples) {
      this.metrics.performance.latencySamples.shift();
    }

    const samples = this.metrics.performance.latencySamples;
    this.metrics.performance.averageLatency =
      samples.reduce((a, b) => a + b, 0) / samples.length;

    this.metrics.performance.minLatency = Math.min(
      this.metrics.performance.minLatency,
      latency
    );

    this.metrics.performance.maxLatency = Math.max(
      this.metrics.performance.maxLatency,
      latency
    );
  }

  recordError(errorType, error, metadata = {}) {
    const now = Date.now();
    this.metrics.errors.total++;
    this.metrics.errors.byType[errorType] =
      (this.metrics.errors.byType[errorType] || 0) + 1;

    const errorRecord = {
      type: errorType,
      message: error.message || error,
      timestamp: now,
      metadata
    };

    this.metrics.errors.recent.push(errorRecord);

    if (this.metrics.errors.recent.length > this.maxErrorHistory) {
      this.metrics.errors.recent.shift();
    }

    this.metrics.health.lastError = errorRecord;
    this.metrics.health.lastErrorTime = now;

    this.checkThresholds();

    logger.error('WebSocket error recorded', errorRecord);
  }

  recordRoomJoin(room, socketId, userId) {
    this.metrics.rooms.activeRooms.add(room);
    this.metrics.rooms.totalSubscriptions++;
    this.metrics.rooms.subscriptionsByRoom[room] =
      (this.metrics.rooms.subscriptionsByRoom[room] || 0) + 1;

    logger.debug('Room subscription recorded', { room, socketId, userId });
  }

  recordRoomLeave(room, socketId, userId) {
    if (this.metrics.rooms.subscriptionsByRoom[room] > 0) {
      this.metrics.rooms.subscriptionsByRoom[room]--;
      this.metrics.rooms.totalSubscriptions--;

      if (this.metrics.rooms.subscriptionsByRoom[room] === 0) {
        delete this.metrics.rooms.subscriptionsByRoom[room];
        this.metrics.rooms.activeRooms.delete(room);
      }
    }

    logger.debug('Room unsubscription recorded', { room, socketId, userId });
  }

  checkThresholds() {
    const alerts = [];

    if (this.metrics.connections.active > this.alertThresholds.maxConnections) {
      alerts.push({
        type: 'HIGH_CONNECTIONS',
        severity: 'warning',
        message: `Active connections (${this.metrics.connections.active}) exceeds threshold (${this.alertThresholds.maxConnections})`,
        timestamp: Date.now()
      });
    }

    if (this.metrics.performance.maxLatency > this.alertThresholds.maxLatency) {
      alerts.push({
        type: 'HIGH_LATENCY',
        severity: 'warning',
        message: `Max latency (${this.metrics.performance.maxLatency}ms) exceeds threshold (${this.alertThresholds.maxLatency}ms)`,
        timestamp: Date.now()
      });
    }

    const errorRate = this.calculateErrorRate();
    if (errorRate > this.alertThresholds.maxErrorRate) {
      alerts.push({
        type: 'HIGH_ERROR_RATE',
        severity: 'critical',
        message: `Error rate (${errorRate.toFixed(2)}%) exceeds threshold (${this.alertThresholds.maxErrorRate}%)`,
        timestamp: Date.now()
      });
    }

    if (alerts.length > 0) {
      this.alerts.push(...alerts);
      this.alertCallbacks.forEach(callback => {
        alerts.forEach(alert => callback(alert));
      });
    }

    return alerts;
  }

  calculateErrorRate() {
    const totalOperations =
      this.metrics.connections.total +
      this.metrics.messages.sent +
      this.metrics.messages.received;

    if (totalOperations === 0) return 0;

    return (this.metrics.errors.total / totalOperations) * 100;
  }

  onAlert(callback) {
    this.alertCallbacks.push(callback);
  }

  removeAlertCallback(callback) {
    const index = this.alertCallbacks.indexOf(callback);
    if (index > -1) {
      this.alertCallbacks.splice(index, 1);
    }
  }

  setAlertThreshold(name, value) {
    if (name in this.alertThresholds) {
      this.alertThresholds[name] = value;
    }
  }

  getMetrics() {
    const uptime = Date.now() - this.startTime;
    this.metrics.health.uptime = uptime;

    return {
      connections: {
        ...this.metrics.connections,
        activeRooms: this.metrics.rooms.activeRooms.size
      },
      messages: { ...this.metrics.messages },
      performance: {
        ...this.metrics.performance,
        errorRate: this.calculateErrorRate()
      },
      errors: {
        ...this.metrics.errors,
        errorRate: this.calculateErrorRate()
      },
      rooms: {
        activeRooms: Array.from(this.metrics.rooms.activeRooms),
        subscriptionsByRoom: { ...this.metrics.rooms.subscriptionsByRoom },
        totalSubscriptions: this.metrics.rooms.totalSubscriptions
      },
      health: { ...this.metrics.health }
    };
  }

  getActiveConnections() {
    return {
      current: this.metrics.connections.active,
      maxConcurrent: this.metrics.connections.maxConcurrent,
      peakTime: this.metrics.connections.peakTime,
      total: this.metrics.connections.total
    };
  }

  getPerformanceMetrics() {
    return {
      latency: {
        average: this.metrics.performance.averageLatency,
        min: this.metrics.performance.minLatency === Infinity ? 0 : this.metrics.performance.minLatency,
        max: this.metrics.performance.maxLatency
      },
      messages: {
        sent: this.metrics.messages.sent,
        received: this.metrics.messages.received,
        broadcast: this.metrics.messages.broadcast,
        totalSize: this.metrics.messages.totalSize,
        byType: { ...this.metrics.messages.byType }
      },
      errorRate: this.calculateErrorRate()
    };
  }

  getRoomMetrics() {
    return {
      activeRooms: Array.from(this.metrics.rooms.activeRooms),
      totalSubscriptions: this.metrics.rooms.totalSubscriptions,
      subscriptionsByRoom: { ...this.metrics.rooms.subscriptionsByRoom }
    };
  }

  getRecentErrors(count = 10) {
    return this.metrics.errors.recent.slice(-count);
  }

  getRecentAlerts(count = 10) {
    return this.alerts.slice(-count);
  }

  getConnectionHistory(count = 20) {
    return this.metrics.connections.connectionHistory.slice(-count);
  }

  startMonitoring(intervalMs = 5000) {
    const monitorId = `monitor_${Date.now()}`;

    const monitor = setInterval(() => {
      this.generatePeriodicReport();
    }, intervalMs);

    this.monitors.set(monitorId, monitor);

    return monitorId;
  }

  stopMonitoring(monitorId) {
    if (this.monitors.has(monitorId)) {
      clearInterval(this.monitors.get(monitorId));
      this.monitors.delete(monitorId);
      return true;
    }
    return false;
  }

  stopAllMonitoring() {
    this.monitors.forEach(monitor => clearInterval(monitor));
    this.monitors.clear();
  }

  generatePeriodicReport() {
    const report = {
      timestamp: Date.now(),
      uptime: Date.now() - this.startTime,
      ...this.getMetrics()
    };

    const alerts = this.checkThresholds();
    if (alerts.length > 0) {
      report.alerts = alerts;
    }

    logger.info('WebSocket periodic report', {
      activeConnections: report.connections.active,
      messagesPerSecond: report.performance.latency.average > 0
        ? this.metrics.messages.sent / (report.uptime / 1000)
        : 0,
      errorRate: report.errors.errorRate
    });

    return report;
  }

  reset() {
    this.metrics.messages.sent = 0;
    this.metrics.messages.received = 0;
    this.metrics.messages.broadcast = 0;
    this.metrics.messages.byType = {};
    this.metrics.messages.totalSize = 0;

    this.metrics.performance.latencySamples = [];
    this.metrics.performance.averageLatency = 0;
    this.metrics.performance.minLatency = Infinity;
    this.metrics.performance.maxLatency = 0;
    this.metrics.performance.messageProcessingTime = [];

    this.metrics.errors.total = 0;
    this.metrics.errors.byType = {};
    this.metrics.errors.recent = [];

    this.metrics.rooms.activeRooms.clear();
    this.metrics.rooms.totalSubscriptions = 0;
    this.metrics.rooms.subscriptionsByRoom = {};

    this.alerts = [];

    logger.info('WebSocket monitor metrics reset');
  }

  getHealthStatus() {
    const metrics = this.getMetrics();
    const errorRate = this.calculateErrorRate();

    let status = 'healthy';
    let issues = [];

    if (errorRate > 5) {
      status = 'degraded';
      issues.push('High error rate');
    }

    if (this.metrics.performance.averageLatency > 1000) {
      status = 'degraded';
      issues.push('High average latency');
    }

    if (this.metrics.connections.active > this.alertThresholds.maxConnections * 0.8) {
      status = 'degraded';
      issues.push('High connection count');
    }

    if (this.metrics.health.lastErrorTime &&
        Date.now() - this.metrics.health.lastErrorTime < 60000) {
      status = 'degraded';
      issues.push('Recent errors detected');
    }

    return {
      status,
      issues,
      uptime: metrics.health.uptime,
      activeConnections: metrics.connections.active,
      errorRate
    };
  }

  exportMetrics() {
    return {
      exportedAt: Date.now(),
      startTime: this.startTime,
      metrics: this.getMetrics(),
      alertThresholds: { ...this.alertThresholds },
      healthStatus: this.getHealthStatus()
    };
  }
}

const websocketMonitor = new WebSocketMonitor();

module.exports = websocketMonitor;
