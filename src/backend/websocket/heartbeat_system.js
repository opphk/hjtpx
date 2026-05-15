let logger;
try {
  const loggerModule = require('../middleware/logger');
  logger = loggerModule.logger || loggerModule;
} catch (error) {
  logger = {
    info: () => {},
    error: () => {},
    warn: () => {},
    debug: () => {}
  };
}

if (!logger.info) logger.info = () => {};
if (!logger.error) logger.error = () => {};
if (!logger.warn) logger.warn = () => {};
if (!logger.debug) logger.debug = () => {};

class HeartbeatSystem {
  constructor() {
    this.io = null;
    this.connectionManager = null;
    this.heartbeatInterval = 25000;
    this.heartbeatTimeout = 10000;
    this.maxMissedHeartbeats = 3;
    this.reconnectDelay = 1000;
    this.maxReconnectDelay = 30000;
    this.reconnectAttempts = new Map();
    this.heartbeatStats = new Map();
    this.checkInterval = null;
    this.checkIntervalMs = 5000;
    this.stats = {
      totalHeartbeats: 0,
      successfulHeartbeats: 0,
      missedHeartbeats: 0,
      timedOutConnections: 0,
      reconnectAttempts: 0,
      reconnectSuccesses: 0,
      reconnectFailures: 0
    };
  }

  initialize(io, connectionManager) {
    this.io = io;
    this.connectionManager = connectionManager;
    this.startHeartbeatChecker();
    this.setupHeartbeatEvents();
    logger.info('Heartbeat system initialized', {
      interval: this.heartbeatInterval,
      timeout: this.heartbeatTimeout,
      checkInterval: this.checkIntervalMs
    });
  }

  setupHeartbeatEvents() {
    if (!this.io) return;

    this.io.on('connection', socket => {
      this.registerSocketHeartbeat(socket);
    });
  }

  registerSocketHeartbeat(socket) {
    const socketId = socket.id;
    const userId = socket.userId;

    this.heartbeatStats.set(socketId, {
      socketId,
      userId,
      lastHeartbeat: Date.now(),
      missedCount: 0,
      totalHeartbeats: 0,
      failedHeartbeats: 0
    });

    socket.on('heartbeat', () => {
      this.handleHeartbeatResponse(socketId);
    });

    socket.on('pong', () => {
      this.handleHeartbeatResponse(socketId);
    });

    socket.on('disconnect', reason => {
      this.unregisterSocketHeartbeat(socketId, reason);
    });

    logger.debug('Socket heartbeat registered', {
      socketId,
      userId
    });
  }

  unregisterSocketHeartbeat(socketId, reason = 'disconnect') {
    const stats = this.heartbeatStats.get(socketId);

    if (stats) {
      logger.debug('Socket heartbeat unregistered', {
        socketId,
        userId: stats.userId,
        reason,
        totalHeartbeats: stats.totalHeartbeats,
        missedCount: stats.missedCount
      });

      this.heartbeatStats.delete(socketId);
    }

    this.reconnectAttempts.delete(socketId);
  }

  handleHeartbeatResponse(socketId) {
    const stats = this.heartbeatStats.get(socketId);

    if (!stats) {
      logger.warn('Heartbeat response from unregistered socket', { socketId });
      return false;
    }

    const now = Date.now();
    const timeSinceLastHeartbeat = now - stats.lastHeartbeat;

    if (timeSinceLastHeartbeat > this.heartbeatTimeout) {
      logger.warn('Late heartbeat response', {
        socketId,
        userId: stats.userId,
        delay: timeSinceLastHeartbeat
      });
    }

    stats.lastHeartbeat = now;
    stats.missedCount = 0;
    stats.totalHeartbeats++;
    this.stats.successfulHeartbeats++;

    return true;
  }

  sendHeartbeat(socketId) {
    if (!this.io) return false;

    try {
      this.io.to(socketId).emit('ping', {
        timestamp: Date.now(),
        serverTime: new Date().toISOString()
      });

      this.stats.totalHeartbeats++;

      const stats = this.heartbeatStats.get(socketId);
      if (stats) {
        stats.missedCount++;
        this.stats.missedHeartbeats++;

        if (stats.missedCount >= this.maxMissedHeartbeats) {
          this.handleHeartbeatTimeout(socketId);
        }
      }

      return true;
    } catch (error) {
      logger.error('Failed to send heartbeat', {
        socketId,
        error: error.message
      });
      return false;
    }
  }

  handleHeartbeatTimeout(socketId) {
    const connection = this.connectionManager.getConnection(socketId);

    if (!connection) {
      logger.warn('Heartbeat timeout for unknown connection', { socketId });
      return false;
    }

    const stats = this.heartbeatStats.get(socketId);
    const userId = connection.userId;

    this.stats.timedOutConnections++;

    logger.warn('Connection heartbeat timeout', {
      socketId,
      userId,
      missedCount: stats?.missedCount || 0,
      connectedDuration: connection.connectedAt ? Date.now() - connection.connectedAt.getTime() : 0
    });

    if (this.reconnectAttempts.has(socketId)) {
      const attempts = this.reconnectAttempts.get(socketId);

      if (attempts >= this.maxReconnectAttempts()) {
        logger.warn('Max reconnect attempts reached, disconnecting', {
          socketId,
          userId,
          attempts
        });

        this.io.to(socketId).emit('connection:timeout', {
          socketId,
          reason: 'heartbeat_timeout',
          attempts
        });

        this.connectionManager.removeConnection(socketId);
        return false;
      }

      this.reconnectAttempts.set(socketId, attempts + 1);
    } else {
      this.reconnectAttempts.set(socketId, 1);
    }

    this.stats.reconnectAttempts++;

    const delay = this.calculateReconnectDelay(this.reconnectAttempts.get(socketId));

    this.io.to(socketId).emit('reconnect:attempt', {
      socketId,
      attempt: this.reconnectAttempts.get(socketId),
      delay,
      maxAttempts: this.maxReconnectAttempts()
    });

    logger.info('Scheduling reconnect attempt', {
      socketId,
      userId,
      attempt: this.reconnectAttempts.get(socketId),
      delay
    });

    setTimeout(() => {
      this.attemptReconnect(socketId, userId);
    }, delay);

    return true;
  }

  attemptReconnect(socketId, userId) {
    if (!this.connectionManager.getConnection(socketId)) {
      logger.warn('Cannot reconnect - connection no longer exists', {
        socketId,
        userId
      });
      this.stats.reconnectFailures++;
      return false;
    }

    this.io.to(socketId).emit('reconnect:ping', {
      socketId,
      timestamp: Date.now()
    });

    logger.debug('Reconnect ping sent', { socketId, userId });

    setTimeout(() => {
      const stats = this.heartbeatStats.get(socketId);
      if (stats && stats.missedCount > 0) {
        this.handleHeartbeatTimeout(socketId);
      } else {
        this.stats.reconnectSuccesses++;
        this.reconnectAttempts.delete(socketId);

        this.io.to(socketId).emit('reconnect:success', {
          socketId,
          timestamp: new Date()
        });

        logger.info('Reconnect successful', {
          socketId,
          userId,
          totalAttempts: this.reconnectAttempts.get(socketId) || 1
        });
      }
    }, this.heartbeatTimeout);

    return true;
  }

  startHeartbeatChecker() {
    if (this.checkInterval) {
      clearInterval(this.checkInterval);
    }

    this.checkInterval = setInterval(() => {
      this.checkAllConnections();
    }, this.checkIntervalMs);

    logger.debug('Heartbeat checker started', {
      interval: this.checkIntervalMs
    });
  }

  stopHeartbeatChecker() {
    if (this.checkInterval) {
      clearInterval(this.checkInterval);
      this.checkInterval = null;
      logger.debug('Heartbeat checker stopped');
    }
  }

  checkAllConnections() {
    const now = Date.now();
    const connections = this.connectionManager.connections;
    const staleConnections = [];

    for (const [socketId, connection] of connections.entries()) {
      const stats = this.heartbeatStats.get(socketId);

      if (!stats) {
        continue;
      }

      const timeSinceLastHeartbeat = now - stats.lastHeartbeat;

      if (timeSinceLastHeartbeat > this.heartbeatInterval + this.heartbeatTimeout) {
        staleConnections.push(socketId);
      }

      if (stats.missedCount > 0 && stats.missedCount < this.maxMissedHeartbeats) {
        this.sendHeartbeat(socketId);
      }
    }

    if (staleConnections.length > 0) {
      logger.debug('Found stale connections', {
        count: staleConnections.length
      });

      staleConnections.forEach(socketId => {
        this.handleHeartbeatTimeout(socketId);
      });
    }
  }

  calculateReconnectDelay(attempt) {
    const delay = Math.min(this.reconnectDelay * Math.pow(2, attempt - 1), this.maxReconnectDelay);
    return delay;
  }

  maxReconnectAttempts() {
    return 5;
  }

  getConnectionHeartbeatStats(socketId) {
    return this.heartbeatStats.get(socketId);
  }

  getAllHeartbeatStats() {
    const stats = [];

    for (const [socketId, stat] of this.heartbeatStats.entries()) {
      stats.push({
        ...stat,
        currentStatus: this.getHeartbeatStatus(socketId)
      });
    }

    return stats;
  }

  getHeartbeatStatus(socketId) {
    const stats = this.heartbeatStats.get(socketId);

    if (!stats) {
      return 'unknown';
    }

    const timeSinceLastHeartbeat = Date.now() - stats.lastHeartbeat;

    if (timeSinceLastHeartbeat < this.heartbeatInterval) {
      return 'healthy';
    } else if (timeSinceLastHeartbeat < this.heartbeatInterval + this.heartbeatTimeout) {
      return 'warning';
    } else {
      return 'critical';
    }
  }

  getStats() {
    const healthy = [];
    const warning = [];
    const critical = [];

    for (const [socketId] of this.heartbeatStats.entries()) {
      const status = this.getHeartbeatStatus(socketId);
      if (status === 'healthy') healthy.push(socketId);
      else if (status === 'warning') warning.push(socketId);
      else if (status === 'critical') critical.push(socketId);
    }

    const avgMissedHeartbeats =
      this.stats.totalHeartbeats > 0
        ? ((this.stats.missedHeartbeats / this.stats.totalHeartbeats) * 100).toFixed(2)
        : 0;

    return {
      ...this.stats,
      heartbeatInterval: this.heartbeatInterval,
      heartbeatTimeout: this.heartbeatTimeout,
      maxMissedHeartbeats: this.maxMissedHeartbeats,
      connectionHealth: {
        total: this.heartbeatStats.size,
        healthy: healthy.length,
        warning: warning.length,
        critical: critical.length
      },
      avgMissedHeartbeatRate: `${avgMissedHeartbeats}%`,
      activeReconnectAttempts: this.reconnectAttempts.size,
      uptime: process.uptime()
    };
  }

  cleanup() {
    this.stopHeartbeatChecker();

    for (const socketId of this.heartbeatStats.keys()) {
      this.unregisterSocketHeartbeat(socketId, 'cleanup');
    }

    this.reconnectAttempts.clear();

    this.stats = {
      totalHeartbeats: 0,
      successfulHeartbeats: 0,
      missedHeartbeats: 0,
      timedOutConnections: 0,
      reconnectAttempts: 0,
      reconnectSuccesses: 0,
      reconnectFailures: 0
    };

    logger.info('Heartbeat system cleaned up');
  }
}

const heartbeatSystem = new HeartbeatSystem();

module.exports = heartbeatSystem;
module.exports.HeartbeatSystem = HeartbeatSystem;
