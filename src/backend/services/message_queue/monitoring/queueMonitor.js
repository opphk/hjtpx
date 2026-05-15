const config = require('../../config/messageQueue');
const connectionManager = require('../connectionManager');

class QueueMonitor {
  constructor() {
    this.metrics = new Map();
    this.alertThresholds = config.monitoring.alertThreshold;
    this.isMonitoring = false;
    this.intervalId = null;
    this.alertHandlers = new Map();
  }

  async start(interval = config.monitoring.metricsInterval) {
    if (this.isMonitoring) {
      return;
    }

    this.isMonitoring = true;
    this.intervalId = setInterval(() => this.collectMetrics(), interval);
    await this.collectMetrics();
    console.log('[QueueMonitor] Started monitoring');
  }

  stop() {
    if (this.intervalId) {
      clearInterval(this.intervalId);
      this.intervalId = null;
    }
    this.isMonitoring = false;
    console.log('[QueueMonitor] Stopped monitoring');
  }

  async collectMetrics() {
    for (const [queueName, queueConfig] of Object.entries(config.queues)) {
      try {
        const metrics = await this.collectQueueMetrics(queueName, queueConfig);
        this.metrics.set(queueName, metrics);
        await this.checkAlerts(queueName, metrics);
      } catch (error) {
        console.error(`[QueueMonitor] Failed to collect metrics for ${queueName}:`, error);
      }
    }
  }

  async collectQueueMetrics(queueName, queueConfig) {
    const client = connectionManager.getClient();

    const streamLength = await client.xlen(queueConfig.stream).catch(() => 0);
    const dlqLength = await client.xlen(queueConfig.deadLetterStream).catch(() => 0);

    let groupInfo = null;
    try {
      groupInfo = await connectionManager.getConsumerGroupInfo(
        queueConfig.stream,
        queueConfig.consumerGroup
      );
    } catch (error) {}

    return {
      queueName,
      stream: queueConfig.stream,
      timestamp: Date.now(),
      streamLength,
      dlqLength,
      pendingMessages: groupInfo?.pending || 0,
      consumerCount: groupInfo?.consumers || 0,
      lastDeliveredId: groupInfo?.lastDeliveredId || null,
      lag: groupInfo?.lag || null
    };
  }

  async checkAlerts(queueName, metrics) {
    const alerts = [];

    if (metrics.streamLength > this.alertThresholds.queueLength) {
      alerts.push({
        type: 'queue_length',
        queue: queueName,
        message: `Queue ${queueName} has ${metrics.streamLength} messages (threshold: ${this.alertThresholds.queueLength})`,
        severity: 'warning'
      });
    }

    if (metrics.dlqLength > 0) {
      alerts.push({
        type: 'dead_letter',
        queue: queueName,
        message: `Queue ${queueName} has ${metrics.dlqLength} messages in DLQ`,
        severity: 'error'
      });
    }

    if (metrics.pendingMessages > this.alertThresholds.queueLength * 0.5) {
      alerts.push({
        type: 'pending_messages',
        queue: queueName,
        message: `Queue ${queueName} has ${metrics.pendingMessages} pending messages`,
        severity: 'warning'
      });
    }

    for (const alert of alerts) {
      await this.triggerAlert(alert);
    }
  }

  registerAlertHandler(type, handler) {
    this.alertHandlers.set(type, handler);
  }

  async triggerAlert(alert) {
    const handler = this.alertHandlers.get(alert.type);
    if (handler) {
      try {
        await handler(alert);
      } catch (error) {
        console.error(`[QueueMonitor] Alert handler error:`, error);
      }
    }

    console.warn(`[QueueMonitor] Alert: ${alert.message}`);
  }

  getMetrics(queueName) {
    return this.metrics.get(queueName);
  }

  getAllMetrics() {
    return Object.fromEntries(this.metrics);
  }

  async getDetailedStats() {
    const stats = {};

    for (const [queueName, queueConfig] of Object.entries(config.queues)) {
      const metrics = this.metrics.get(queueName);
      const groupInfo = await connectionManager.getConsumerGroupInfo(
        queueConfig.stream,
        queueConfig.consumerGroup
      );

      stats[queueName] = {
        ...metrics,
        healthStatus: this.calculateHealthStatus(metrics),
        consumerGroup: groupInfo
      };
    }

    return stats;
  }

  calculateHealthStatus(metrics) {
    if (!metrics) {
      return 'unknown';
    }

    if (metrics.dlqLength > 0) {
      return 'degraded';
    }

    if (metrics.streamLength > this.alertThresholds.queueLength * 0.8) {
      return 'warning';
    }

    if (metrics.consumerCount === 0) {
      return 'critical';
    }

    return 'healthy';
  }

  async healthCheck() {
    try {
      const connectionHealth = await connectionManager.healthCheck();
      const metricsCount = this.metrics.size;

      return {
        healthy: this.isMonitoring && connectionHealth.healthy,
        monitoring: this.isMonitoring,
        connectionHealthy: connectionHealth.healthy,
        metricsCollected: metricsCount,
        alerts: this.getAllMetrics()
      };
    } catch (error) {
      return {
        healthy: false,
        error: error.message
      };
    }
  }
}

const queueMonitor = new QueueMonitor();

module.exports = queueMonitor;
