const { logger } = require('../middleware/logger');

class HeartbeatManager {
  constructor(config = {}) {
    this.config = {
      pingInterval: config.pingInterval || 25000,
      pingTimeout: config.pingTimeout || config.pingInterval / 2 || 12500,
      maxMissedPongs: config.maxMissedPongs || 3,
      checkInterval: config.checkInterval || 5000,
      enableAdaptivePing: config.enableAdaptivePing !== false,
      minPingInterval: config.minPingInterval || 10000,
      maxPingInterval: config.maxPingInterval || 60000,
      pongThreshold: config.pongThreshold || 0.95
    };

    this.sockets = new Map();
    this.stats = {
      totalPings: 0,
      totalPongs: 0,
      missedPongs: 0,
      averageResponseTime: 0,
      adaptiveAdjustments: 0,
      socketHealth: {
        healthy: 0,
        degraded: 0,
        unhealthy: 0
      }
    };

    this.monitoringInterval = null;
    this.isRunning = false;
    this.responseTimes = [];
    this.maxResponseTimes = 100;
  }

  registerSocket(socket) {
    const socketInfo = {
      socket,
      userId: socket.userId,
      socketId: socket.id,
      lastPing: null,
      lastPong: null,
      missedPongs: 0,
      pingSequence: 0,
      responseTimes: [],
      healthScore: 100,
      isHealthy: true,
      status: 'active',
      registeredAt: Date.now(),
      currentPingInterval: this.config.pingInterval
    };

    this.sockets.set(socket.id, socketInfo);

    this.setupSocketHandlers(socket);

    logger.debug('Socket registered with heartbeat manager', {
      socketId: socket.id,
      userId: socket.userId,
      pingInterval: this.config.pingInterval
    });

    return socketInfo;
  }

  setupSocketHandlers(socket) {
    socket.on('pong', (data) => {
      this.handlePong(socket.id, data);
    });

    socket.on('disconnect', (reason) => {
      this.unregisterSocket(socket.id, reason);
    });

    socket.on('error', (error) => {
      this.handleSocketError(socket.id, error);
    });
  }

  handlePong(socketId, data) {
    const socketInfo = this.sockets.get(socketId);
    if (!socketInfo) return;

    const now = Date.now();
    const responseTime = socketInfo.lastPing ? now - socketInfo.lastPing : 0;

    socketInfo.lastPong = now;
    socketInfo.missedPongs = 0;
    socketInfo.pingSequence++;
    socketInfo.responseTimes.push(responseTime);

    if (socketInfo.responseTimes.length > 10) {
      socketInfo.responseTimes.shift();
    }

    this.stats.totalPongs++;
    this.updateSocketHealth(socketId, responseTime);
    this.recordResponseTime(responseTime);

    if (this.config.enableAdaptivePing) {
      this.adjustPingInterval(socketId, responseTime);
    }

    logger.debug('Pong received', {
      socketId,
      responseTime,
      healthScore: socketInfo.healthScore
    });
  }

  recordResponseTime(responseTime) {
    this.responseTimes.push(responseTime);

    if (this.responseTimes.length > this.maxResponseTimes) {
      this.responseTimes.shift();
    }

    const sum = this.responseTimes.reduce((a, b) => a + b, 0);
    this.stats.averageResponseTime = sum / this.responseTimes.length;
  }

  updateSocketHealth(socketId, lastResponseTime) {
    const socketInfo = this.sockets.get(socketId);
    if (!socketInfo) return;

    const avgResponseTime = socketInfo.responseTimes.length > 0
      ? socketInfo.responseTimes.reduce((a, b) => a + b, 0) / socketInfo.responseTimes.length
      : 0;

    const latencyScore = Math.max(0, 100 - (avgResponseTime / 10));
    const stabilityScore = Math.max(0, 100 - (socketInfo.missedPongs * 20));
    const uptimeScore = Math.min(100, (Date.now() - socketInfo.registeredAt) / 1000);

    socketInfo.healthScore = Math.round((latencyScore * 0.5) + (stabilityScore * 0.3) + (uptimeScore * 0.2));

    if (socketInfo.healthScore >= 80) {
      socketInfo.isHealthy = true;
      socketInfo.status = 'healthy';
      this.stats.socketHealth.healthy++;
    } else if (socketInfo.healthScore >= 50) {
      socketInfo.isHealthy = true;
      socketInfo.status = 'degraded';
      this.stats.socketHealth.degraded++;
    } else {
      socketInfo.isHealthy = false;
      socketInfo.status = 'unhealthy';
      this.stats.socketHealth.unhealthy++;
    }
  }

  adjustPingInterval(socketId, responseTime) {
    const socketInfo = this.sockets.get(socketId);
    if (!socketInfo) return;

    const targetResponseTime = responseTime * 2;
    const currentInterval = socketInfo.currentPingInterval;

    let newInterval = currentInterval;

    if (responseTime > this.config.pingTimeout * 0.8) {
      newInterval = Math.max(this.config.minPingInterval, currentInterval - 5000);
      logger.debug('Decreasing ping interval due to high latency', {
        socketId,
        oldInterval: currentInterval,
        newInterval,
        responseTime
      });
    } else if (socketInfo.responseTimes.length >= 5) {
      const avgResponse = socketInfo.responseTimes.reduce((a, b) => a + b, 0) / socketInfo.responseTimes.length;
      const variance = this.calculateVariance(socketInfo.responseTimes, avgResponse);

      if (variance < 100 && socketInfo.missedPongs === 0) {
        newInterval = Math.min(this.config.maxPingInterval, currentInterval + 2000);
        logger.debug('Increasing ping interval for stable connection', {
          socketId,
          oldInterval: currentInterval,
          newInterval
        });
      }
    }

    if (newInterval !== currentInterval) {
      socketInfo.currentPingInterval = newInterval;
      this.stats.adaptiveAdjustments++;
    }
  }

  calculateVariance(values, mean) {
    if (values.length === 0) return 0;
    const squaredDiffs = values.map(v => Math.pow(v - mean, 2));
    return squaredDiffs.reduce((a, b) => a + b, 0) / values.length;
  }

  handleMissedPong(socketId) {
    const socketInfo = this.sockets.get(socketId);
    if (!socketInfo) return;

    socketInfo.missedPongs++;
    this.stats.missedPongs++;

    logger.warn('Missed pong', {
      socketId,
      userId: socketInfo.userId,
      missedCount: socketInfo.missedPongs,
      maxAllowed: this.config.maxMissedPongs
    });

    this.updateSocketHealth(socketId, 0);

    if (socketInfo.missedPongs >= this.config.maxMissedPongs) {
      this.handleConnectionFailure(socketId);
    }
  }

  handleConnectionFailure(socketId) {
    const socketInfo = this.sockets.get(socketId);
    if (!socketInfo) return;

    logger.error('Connection failure due to missed pongs', {
      socketId,
      userId: socketInfo.userId,
      missedPongs: socketInfo.missedPongs
    });

    socketInfo.status = 'failed';
    socketInfo.isHealthy = false;

    if (socketInfo.socket.connected) {
      socketInfo.socket.emit('heartbeat:timeout', {
        socketId,
        missedPongs: socketInfo.missedPongs,
        timestamp: Date.now()
      });

      setTimeout(() => {
        if (!socketInfo.isHealthy && socketInfo.socket.connected) {
          socketInfo.socket.disconnect();
        }
      }, 1000);
    }
  }

  handleSocketError(socketId, error) {
    const socketInfo = this.sockets.get(socketId);
    if (!socketInfo) return;

    logger.error('Socket error in heartbeat manager', {
      socketId,
      userId: socketInfo.userId,
      error: error.message || error
    });

    socketInfo.healthScore = Math.max(0, socketInfo.healthScore - 20);
  }

  unregisterSocket(socketId, reason = 'unknown') {
    const socketInfo = this.sockets.get(socketId);
    if (!socketInfo) return;

    logger.debug('Socket unregistered from heartbeat manager', {
      socketId,
      userId: socketInfo.userId,
      reason,
      totalUptime: Date.now() - socketInfo.registeredAt,
      totalPings: socketInfo.pingSequence
    });

    this.sockets.delete(socketId);

    return socketInfo;
  }

  startMonitoring() {
    if (this.isRunning) {
      logger.warn('Heartbeat manager already running');
      return;
    }

    this.isRunning = true;

    this.monitoringInterval = setInterval(() => {
      this.performHealthChecks();
    }, this.config.checkInterval);

    logger.info('Heartbeat manager started', {
      pingInterval: this.config.pingInterval,
      pingTimeout: this.config.pingTimeout,
      checkInterval: this.config.checkInterval,
      adaptivePing: this.config.enableAdaptivePing
    });
  }

  stopMonitoring() {
    if (!this.isRunning) {
      return;
    }

    this.isRunning = false;

    if (this.monitoringInterval) {
      clearInterval(this.monitoringInterval);
      this.monitoringInterval = null;
    }

    logger.info('Heartbeat manager stopped');
  }

  performHealthChecks() {
    const now = Date.now();
    const socketsToPing = [];

    this.sockets.forEach((socketInfo, socketId) => {
      if (!socketInfo.socket.connected) {
        this.unregisterSocket(socketId, 'disconnected');
        return;
      }

      const timeSinceLastPong = socketInfo.lastPong ? now - socketInfo.lastPong : now - socketInfo.registeredAt;
      const timeSinceLastPing = socketInfo.lastPing ? now - socketInfo.lastPing : 0;

      if (timeSinceLastPong > socketInfo.currentPingInterval + this.config.pingTimeout) {
        this.handleMissedPong(socketId);
      }

      if (timeSinceLastPing >= socketInfo.currentPingInterval) {
        socketsToPing.push(socketInfo);
      }
    });

    socketsToPing.forEach(socketInfo => {
      this.sendPing(socketInfo);
    });

    this.cleanupInactiveSockets();
  }

  sendPing(socketInfo) {
    const now = Date.now();
    socketInfo.lastPing = now;
    socketInfo.socket.ping();

    this.stats.totalPings++;

    logger.debug('Ping sent', {
      socketId: socketInfo.socketId,
      sequence: socketInfo.pingSequence,
      interval: socketInfo.currentPingInterval
    });

    setTimeout(() => {
      if (socketInfo.lastPing === now && socketInfo.missedPongs < this.config.maxMissedPongs) {
        this.handleMissedPong(socketInfo.socketId);
      }
    }, this.config.pingTimeout);
  }

  cleanupInactiveSockets() {
    const now = Date.now();
    const maxInactivityTime = this.config.pingInterval * this.config.maxMissedPongs * 2;

    this.sockets.forEach((socketInfo, socketId) => {
      const lastActivity = socketInfo.lastPong || socketInfo.registeredAt;

      if (now - lastActivity > maxInactivityTime) {
        logger.warn('Removing inactive socket', {
          socketId,
          userId: socketInfo.userId,
          inactivityTime: now - lastActivity
        });

        if (socketInfo.socket.connected) {
          socketInfo.socket.disconnect();
        }

        this.unregisterSocket(socketId, 'inactive');
      }
    });
  }

  getSocketInfo(socketId) {
    return this.sockets.get(socketId);
  }

  getAllSockets() {
    return Array.from(this.sockets.values()).map(info => ({
      socketId: info.socketId,
      userId: info.userId,
      status: info.status,
      healthScore: info.healthScore,
      isHealthy: info.isHealthy,
      lastPing: info.lastPing,
      lastPong: info.lastPong,
      missedPongs: info.missedPongs,
      currentPingInterval: info.currentPingInterval,
      registeredAt: info.registeredAt
    }));
  }

  getStats() {
    return {
      ...this.stats,
      socketHealth: { ...this.stats.socketHealth },
      totalSockets: this.sockets.size,
      averageResponseTime: this.stats.averageResponseTime
    };
  }

  getHealthReport() {
    const sockets = this.getAllSockets();
    const healthy = sockets.filter(s => s.isHealthy && s.healthScore >= 80);
    const degraded = sockets.filter(s => s.isHealthy && s.healthScore >= 50 && s.healthScore < 80);
    const unhealthy = sockets.filter(s => !s.isHealthy || s.healthScore < 50);

    return {
      total: sockets.length,
      healthy: {
        count: healthy.length,
        percentage: sockets.length > 0 ? (healthy.length / sockets.length) * 100 : 0,
        sockets: healthy.map(s => ({ socketId: s.socketId, userId: s.userId }))
      },
      degraded: {
        count: degraded.length,
        percentage: sockets.length > 0 ? (degraded.length / sockets.length) * 100 : 0,
        sockets: degraded.map(s => ({ socketId: s.socketId, userId: s.userId }))
      },
      unhealthy: {
        count: unhealthy.length,
        percentage: sockets.length > 0 ? (unhealthy.length / sockets.length) * 100 : 0,
        sockets: unhealthy.map(s => ({ socketId: s.socketId, userId: s.userId }))
      },
      stats: this.getStats()
    };
  }

  resetStats() {
    this.stats = {
      totalPings: 0,
      totalPongs: 0,
      missedPongs: 0,
      averageResponseTime: 0,
      adaptiveAdjustments: 0,
      socketHealth: {
        healthy: 0,
        degraded: 0,
        unhealthy: 0
      }
    };
    this.responseTimes = [];
  }

  configure(config) {
    this.config = {
      ...this.config,
      ...config
    };

    logger.info('Heartbeat manager reconfigured', this.config);
  }
}

const heartbeatManager = new HeartbeatManager();

module.exports = heartbeatManager;
