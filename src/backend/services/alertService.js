const { logError, logWarn, logInfo } = require('../utils/productionLogger');

class AlertManager {
  constructor() {
    this.alerts = new Map();
    this.handlers = new Map();
    this.thresholds = {
      errorRate: parseFloat(process.env.ALERT_ERROR_RATE_THRESHOLD) || 0.05,
      responseTime: parseInt(process.env.ALERT_RESPONSE_TIME_THRESHOLD) || 3000,
      cpuUsage: parseFloat(process.env.ALERT_CPU_THRESHOLD) || 0.8,
      memoryUsage: parseFloat(process.env.ALERT_MEMORY_THRESHOLD) || 0.85,
      activeConnections: parseInt(process.env.ALERT_CONNECTIONS_THRESHOLD) || 1000,
      databaseConnections: parseInt(process.env.ALERT_DB_CONNECTIONS_THRESHOLD) || 50,
      cacheHitRatio: parseFloat(process.env.ALERT_CACHE_HIT_RATIO_THRESHOLD) || 0.5,
      slowQueries: parseInt(process.env.ALERT_SLOW_QUERIES_THRESHOLD) || 10
    };

    this.alertCooldown = new Map();
    this.cooldownPeriod = parseInt(process.env.ALERT_COOLDOWN_MINUTES) || 30;
  }

  registerHandler(alertType, handler) {
    this.handlers.set(alertType, handler);
  }

  async triggerAlert(alertType, data) {
    const alertId = `${alertType}-${Date.now()}`;
    const now = Date.now();

    const lastAlert = this.alertCooldown.get(alertType);
    if (lastAlert && now - lastAlert < this.cooldownPeriod * 60 * 1000) {
      return null;
    }

    const alert = {
      id: alertId,
      type: alertType,
      timestamp: new Date().toISOString(),
      data,
      acknowledged: false,
      resolved: false
    };

    this.alerts.set(alertId, alert);
    this.alertCooldown.set(alertType, now);

    logWarn(`Alert triggered: ${alertType}`, { alertId, data });

    const handler = this.handlers.get(alertType);
    if (handler) {
      try {
        await handler(alert);
      } catch (error) {
        logError(error, null, { context: `Alert handler for ${alertType}` });
      }
    }

    return alert;
  }

  checkErrorRate(errorCount, totalRequests) {
    if (totalRequests === 0) return;

    const errorRate = errorCount / totalRequests;
    if (errorRate > this.thresholds.errorRate) {
      this.triggerAlert('high_error_rate', {
        errorRate,
        threshold: this.thresholds.errorRate,
        errorCount,
        totalRequests
      });
    }
  }

  checkResponseTime(avgResponseTime, route) {
    if (avgResponseTime > this.thresholds.responseTime) {
      this.triggerAlert('slow_response_time', {
        avgResponseTime,
        threshold: this.thresholds.responseTime,
        route
      });
    }
  }

  checkCpuUsage(cpuUsage) {
    if (cpuUsage > this.thresholds.cpuUsage) {
      this.triggerAlert('high_cpu_usage', {
        cpuUsage,
        threshold: this.thresholds.cpuUsage
      });
    }
  }

  checkMemoryUsage(memoryUsage) {
    if (memoryUsage > this.thresholds.memoryUsage) {
      this.triggerAlert('high_memory_usage', {
        memoryUsage,
        threshold: this.thresholds.memoryUsage
      });
    }
  }

  checkActiveConnections(connections) {
    if (connections > this.thresholds.activeConnections) {
      this.triggerAlert('high_connection_count', {
        connections,
        threshold: this.thresholds.activeConnections
      });
    }
  }

  checkDatabaseConnections(connections) {
    if (connections > this.thresholds.databaseConnections) {
      this.triggerAlert('high_db_connections', {
        connections,
        threshold: this.thresholds.databaseConnections
      });
    }
  }

  checkCacheHitRatio(hitRatio) {
    if (hitRatio < this.thresholds.cacheHitRatio) {
      this.triggerAlert('low_cache_hit_ratio', {
        hitRatio,
        threshold: this.thresholds.cacheHitRatio
      });
    }
  }

  checkSlowQueries(count) {
    if (count > this.thresholds.slowQueries) {
      this.triggerAlert('high_slow_query_count', {
        count,
        threshold: this.thresholds.slowQueries
      });
    }
  }

  acknowledgeAlert(alertId) {
    const alert = this.alerts.get(alertId);
    if (alert) {
      alert.acknowledged = true;
      alert.acknowledgedAt = new Date().toISOString();
      return alert;
    }
    return null;
  }

  resolveAlert(alertId) {
    const alert = this.alerts.get(alertId);
    if (alert) {
      alert.resolved = true;
      alert.resolvedAt = new Date().toISOString();
      this.alertCooldown.delete(alert.type);
      return alert;
    }
    return null;
  }

  getActiveAlerts() {
    return Array.from(this.alerts.values()).filter(alert => !alert.resolved);
  }

  getAlertsByType(type) {
    return Array.from(this.alerts.values()).filter(alert => alert.type === type && !alert.resolved);
  }

  clearOldAlerts(maxAgeHours = 24) {
    const cutoff = Date.now() - maxAgeHours * 60 * 60 * 1000;
    for (const [id, alert] of this.alerts) {
      if (new Date(alert.timestamp).getTime() < cutoff && alert.resolved) {
        this.alerts.delete(id);
      }
    }
  }

  getMetrics() {
    return {
      activeAlerts: this.getActiveAlerts().length,
      totalAlerts: this.alerts.size,
      alertsByType: this.getAlertsByType.bind(this),
      thresholds: this.thresholds
    };
  }
}

const alertManager = new AlertManager();

alertManager.registerHandler('high_error_rate', async alert => {
  logWarn('High error rate detected', alert.data);
});

alertManager.registerHandler('slow_response_time', async alert => {
  logWarn('Slow response time detected', alert.data);
});

alertManager.registerHandler('high_cpu_usage', async alert => {
  logWarn('High CPU usage detected', alert.data);
});

alertManager.registerHandler('high_memory_usage', async alert => {
  logWarn('High memory usage detected', alert.data);
});

alertManager.registerHandler('high_connection_count', async alert => {
  logWarn('High connection count detected', alert.data);
});

alertManager.registerHandler('high_db_connections', async alert => {
  logWarn('High database connection count detected', alert.data);
});

alertManager.registerHandler('low_cache_hit_ratio', async alert => {
  logWarn('Low cache hit ratio detected', alert.data);
});

alertManager.registerHandler('high_slow_query_count', async alert => {
  logWarn('High slow query count detected', alert.data);
});

module.exports = {
  AlertManager,
  alertManager
};
