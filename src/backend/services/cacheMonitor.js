const advancedCacheService = require('./advancedCacheService');

class CacheMonitor {
  constructor() {
    this.metricsHistory = {
      hitRate: [],
      latency: [],
      memory: [],
      evictions: []
    };

    this.alertThresholds = {
      hitRate: { warning: 60, critical: 40 },
      latency: { warning: 100, critical: 500 },
      memory: { warning: 80, critical: 95 },
      errorRate: { warning: 5, critical: 10 }
    };

    this.alerts = [];
    this.monitoringInterval = null;
    this.historyRetention = 1440;

    this.startMonitoring();
  }

  startMonitoring() {
    this.monitoringInterval = setInterval(() => {
      this.collectMetrics();
      this.checkThresholds();
      this.pruneHistory();
    }, 60000);

    console.log('📊 Cache monitoring started');
  }

  stopMonitoring() {
    if (this.monitoringInterval) {
      clearInterval(this.monitoringInterval);
      this.monitoringInterval = null;
      console.log('📊 Cache monitoring stopped');
    }
  }

  collectMetrics() {
    const stats = advancedCacheService.getStats();

    const metrics = {
      timestamp: Date.now(),
      hitRate: parseFloat(stats.total.hitRate),
      l1HitRate: parseFloat(stats.l1.hitRate),
      l2HitRate: parseFloat(stats.l2.hitRate),
      l3HitRate: parseFloat(stats.l3.hitRate),
      latency: {
        l1: parseFloat(stats.l1.avgLatency),
        l2: parseFloat(stats.l2.avgLatency),
        l3: parseFloat(stats.l3.avgLatency)
      },
      memory: {
        used: stats.memory.used,
        peak: stats.memory.peak,
        usagePercent: (stats.memory.used / (stats.l1.maxSize * 100)) * 100
      },
      operations: {
        hits: stats.total.hits,
        misses: stats.total.misses,
        sets: stats.total.sets,
        invalidations: stats.total.invalidations
      },
      evictions: {
        l1: stats.l1.evictions,
        total: stats.l1.evictions
      },
      errors: {
        l2: stats.l2.errors,
        l3: stats.l3.errors,
        total: stats.l2.errors + stats.l3.errors
      }
    };

    this.metricsHistory.hitRate.push(metrics.hitRate);
    this.metricsHistory.latency.push(metrics.latency);
    this.metricsHistory.memory.push(metrics.memory);
    this.metricsHistory.evictions.push(metrics.evictions);

    return metrics;
  }

  checkThresholds() {
    const currentMetrics = this.getCurrentMetrics();

    if (!currentMetrics) return;

    if (currentMetrics.hitRate <= this.alertThresholds.hitRate.critical) {
      this.triggerAlert(
        'critical',
        'Cache Hit Rate Critical',
        `Cache hit rate is at ${currentMetrics.hitRate}% (critical: <${this.alertThresholds.hitRate.critical}%)`
      );
    } else if (currentMetrics.hitRate <= this.alertThresholds.hitRate.warning) {
      this.triggerAlert(
        'warning',
        'Cache Hit Rate Low',
        `Cache hit rate is at ${currentMetrics.hitRate}% (warning: <${this.alertThresholds.hitRate.warning}%)`
      );
    }

    if (currentMetrics.latency && currentMetrics.latency.l2 >= this.alertThresholds.latency.critical) {
      this.triggerAlert(
        'critical',
        'Cache Latency Critical',
        `L2 cache latency is at ${currentMetrics.latency.l2}ms (critical: >${this.alertThresholds.latency.critical}ms)`
      );
    } else if (currentMetrics.latency && currentMetrics.latency.l2 >= this.alertThresholds.latency.warning) {
      this.triggerAlert(
        'warning',
        'Cache Latency High',
        `L2 cache latency is at ${currentMetrics.latency.l2}ms (warning: >${this.alertThresholds.latency.warning}ms)`
      );
    }

    const errorRate = this.calculateErrorRate(currentMetrics);
    if (errorRate >= this.alertThresholds.errorRate.critical) {
      this.triggerAlert(
        'critical',
        'Cache Error Rate Critical',
        `Cache error rate is at ${errorRate}% (critical: >${this.alertThresholds.errorRate.critical}%)`
      );
    } else if (errorRate >= this.alertThresholds.errorRate.warning) {
      this.triggerAlert(
        'warning',
        'Cache Error Rate High',
        `Cache error rate is at ${errorRate}% (warning: >${this.alertThresholds.errorRate.warning}%)`
      );
    }
  }

  calculateErrorRate(metrics) {
    if (!metrics || !metrics.operations) return 0;
    const totalOperations = metrics.operations.hits + metrics.operations.misses;
    if (totalOperations === 0) return 0;
    return ((metrics.errors.total / totalOperations) * 100).toFixed(2);
  }

  triggerAlert(level, title, message) {
    const existingAlert = this.alerts.find(
      a => a.title === title && a.level === level && Date.now() - a.timestamp < 300000
    );

    if (existingAlert) return;

    const alert = {
      id: `alert_${Date.now()}`,
      level,
      title,
      message,
      timestamp: Date.now(),
      acknowledged: false
    };

    this.alerts.push(alert);
    console.warn(`[${level.toUpperCase()}] ${title}: ${message}`);

    if (this.alerts.length > 100) {
      this.alerts = this.alerts.slice(-100);
    }
  }

  acknowledgeAlert(alertId) {
    const alert = this.alerts.find(a => a.id === alertId);
    if (alert) {
      alert.acknowledged = true;
      alert.acknowledgedAt = Date.now();
    }
  }

  pruneHistory() {
    if (this.metricsHistory.hitRate.length > this.historyRetention) {
      this.metricsHistory.hitRate = this.metricsHistory.hitRate.slice(-this.historyRetention);
    }
    if (this.metricsHistory.latency.length > this.historyRetention) {
      this.metricsHistory.latency = this.metricsHistory.latency.slice(-this.historyRetention);
    }
    if (this.metricsHistory.memory.length > this.historyRetention) {
      this.metricsHistory.memory = this.metricsHistory.memory.slice(-this.historyRetention);
    }
    if (this.metricsHistory.evictions.length > this.historyRetention) {
      this.metricsHistory.evictions = this.metricsHistory.evictions.slice(-this.historyRetention);
    }
  }

  getCurrentMetrics() {
    if (this.metricsHistory.hitRate.length === 0) {
      return this.collectMetrics();
    }

    return {
      timestamp: Date.now(),
      hitRate: this.metricsHistory.hitRate[this.metricsHistory.hitRate.length - 1],
      latency: this.metricsHistory.latency[this.metricsHistory.latency.length - 1],
      memory: this.metricsHistory.memory[this.metricsHistory.memory.length - 1],
      evictions: this.metricsHistory.evictions[this.metricsHistory.evictions.length - 1]
    };
  }

  getMetricsHistory(period = 'hour') {
    const counts = {
      minute: 60,
      hour: 60,
      day: 1440
    };

    const count = counts[period] || 60;

    return {
      hitRate: this.metricsHistory.hitRate.slice(-count),
      latency: this.metricsHistory.latency.slice(-count),
      memory: this.metricsHistory.memory.slice(-count),
      evictions: this.metricsHistory.evictions.slice(-count)
    };
  }

  getStatistics(period = 'hour') {
    const history = this.getMetricsHistory(period);

    return {
      hitRate: this.calculateStats(history.hitRate),
      latency: {
        l1: this.calculateStats(history.latency.map(l => l?.l1 || 0)),
        l2: this.calculateStats(history.latency.map(l => l?.l2 || 0)),
        l3: this.calculateStats(history.latency.map(l => l?.l3 || 0))
      },
      memory: {
        used: this.calculateStats(history.memory.map(m => m?.used || 0)),
        peak: this.calculateStats(history.memory.map(m => m?.peak || 0)),
        usagePercent: this.calculateStats(history.memory.map(m => m?.usagePercent || 0))
      },
      evictions: {
        l1: this.calculateStats(history.evictions.map(e => e?.l1 || 0)),
        total: this.calculateStats(history.evictions.map(e => e?.total || 0))
      }
    };
  }

  calculateStats(values) {
    if (values.length === 0) return { min: 0, max: 0, avg: 0, median: 0 };

    const sorted = [...values].sort((a, b) => a - b);
    const sum = sorted.reduce((a, b) => a + b, 0);

    return {
      min: sorted[0],
      max: sorted[sorted.length - 1],
      avg: (sum / sorted.length).toFixed(2),
      median: sorted[Math.floor(sorted.length / 2)]
    };
  }

  getAlerts(includeAcknowledged = false) {
    if (includeAcknowledged) {
      return this.alerts;
    }
    return this.alerts.filter(a => !a.acknowledged);
  }

  clearAcknowledgedAlerts() {
    this.alerts = this.alerts.filter(a => !a.acknowledged);
  }

  generateReport() {
    const currentMetrics = this.getCurrentMetrics();
    const stats = this.getStatistics('hour');
    const cacheStats = advancedCacheService.getStats();
    const activeAlerts = this.getAlerts(false);

    const report = {
      generatedAt: new Date().toISOString(),
      period: 'last hour',
      summary: {
        status: this.evaluateOverallStatus(activeAlerts),
        totalAlerts: activeAlerts.length,
        criticalAlerts: activeAlerts.filter(a => a.level === 'critical').length,
        warningAlerts: activeAlerts.filter(a => a.level === 'warning').length
      },
      performance: {
        hitRate: {
          overall: `${currentMetrics.hitRate?.toFixed(2) || 0}%`,
          l1: `${currentMetrics.latency?.l1 || 0}ms avg`,
          l2: `${currentMetrics.latency?.l2 || 0}ms avg`,
          l3: `${currentMetrics.latency?.l3 || 0}ms avg`
        },
        statistics: stats
      },
      capacity: {
        memory: {
          used: this.formatBytes(cacheStats?.memory?.used || 0),
          peak: this.formatBytes(cacheStats?.memory?.peak || 0),
          l1Size: cacheStats?.l1?.size || 0,
          l1MaxSize: cacheStats?.l1?.maxSize || 0
        },
        utilization: {
          l1: `${(((cacheStats?.l1?.size || 0) / (cacheStats?.l1?.maxSize || 1)) * 100).toFixed(2)}%`
        }
      },
      operations: {
        totalHits: cacheStats?.total?.hits || 0,
        totalMisses: cacheStats?.total?.misses || 0,
        totalSets: cacheStats?.total?.sets || 0,
        totalInvalidations: cacheStats?.total?.invalidations || 0,
        evictions: cacheStats?.l1?.evictions || 0
      },
      health: {
        l1Enabled: cacheStats?.status?.l1Enabled || false,
        l2Connected: cacheStats?.status?.l2Enabled || false,
        l3Enabled: cacheStats?.status?.l3Enabled || false,
        overallHealthy: cacheStats?.status?.healthy || false
      },
      recommendations: this.generateRecommendations(currentMetrics, activeAlerts)
    };

    return report;
  }

  evaluateOverallStatus(alerts) {
    if (alerts.some(a => a.level === 'critical')) {
      return 'critical';
    }
    if (alerts.some(a => a.level === 'warning')) {
      return 'warning';
    }
    return 'healthy';
  }

  generateRecommendations(metrics, alerts) {
    const recommendations = [];

    if (metrics.hitRate < 70) {
      recommendations.push({
        priority: 'high',
        category: 'performance',
        message: 'Cache hit rate is below 70%. Consider increasing cache TTL or warming hot data.',
        action: 'Review cache warming strategy and increase cache size'
      });
    }

    if (metrics.latency?.l2 > 100) {
      recommendations.push({
        priority: 'medium',
        category: 'performance',
        message: 'L2 cache (Redis) latency is high.',
        action: 'Check Redis server performance and network latency'
      });
    }

    if (metrics.memory?.usagePercent > 80) {
      recommendations.push({
        priority: 'high',
        category: 'capacity',
        message: 'L1 cache memory usage is above 80%.',
        action: 'Consider increasing L1 cache size or implementing cache sharding'
      });
    }

    if (alerts.some(a => a.title.includes('Error Rate'))) {
      recommendations.push({
        priority: 'critical',
        category: 'reliability',
        message: 'Cache error rate is elevated.',
        action: 'Investigate cache infrastructure and implement fallback mechanisms'
      });
    }

    if (recommendations.length === 0) {
      recommendations.push({
        priority: 'info',
        category: 'general',
        message: 'Cache is performing optimally.',
        action: 'Continue monitoring and periodic maintenance'
      });
    }

    return recommendations;
  }

  formatBytes(bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  }

  exportMetrics(format = 'json') {
    const report = this.generateReport();

    if (format === 'json') {
      return JSON.stringify(report, null, 2);
    } else if (format === 'csv') {
      return this.exportToCSV(report);
    }

    return report;
  }

  exportToCSV(report) {
    const lines = ['Metric,Value'];

    lines.push(`Overall Status,${report.summary.status}`);
    lines.push(`Total Alerts,${report.summary.totalAlerts}`);
    lines.push(`Hit Rate,${report.performance.hitRate.overall}`);
    lines.push(`L1 Size,${report.capacity.memory.used}`);
    lines.push(`Total Hits,${report.operations.totalHits}`);
    lines.push(`Total Misses,${report.operations.totalMisses}`);
    lines.push(`Evictions,${report.operations.evictions}`);

    return lines.join('\n');
  }

  setAlertThresholds(thresholds) {
    this.alertThresholds = { ...this.alertThresholds, ...thresholds };
  }

  getHealthStatus() {
    const cacheStats = advancedCacheService.getStats();

    return {
      healthy: cacheStats?.status?.healthy || false,
      l1: {
        enabled: cacheStats?.status?.l1Enabled || false,
        status: cacheStats?.status?.l1Enabled ? 'up' : 'down'
      },
      l2: {
        enabled: cacheStats?.status?.l2Enabled || false,
        connected: cacheStats?.l2?.connected || false,
        status: cacheStats?.l2?.connected ? 'up' : 'down'
      },
      l3: {
        enabled: cacheStats?.status?.l3Enabled || false,
        connected: cacheStats?.l3?.connected || false,
        status: cacheStats?.l3?.connected ? 'up' : 'down'
      },
      alerts: this.getAlerts(false).length
    };
  }
}

const cacheMonitor = new CacheMonitor();

module.exports = cacheMonitor;
