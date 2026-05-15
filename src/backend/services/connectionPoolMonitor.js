const EventEmitter = require('events');
const fs = require('fs');
const path = require('path');

class ConnectionPoolMonitor extends EventEmitter {
  constructor(dbPoolManager) {
    super();
    this.dbPoolManager = dbPoolManager;
    this.metricsHistory = [];
    this.maxHistorySize = 1000;
    this.collectionInterval = null;
    this.reportInterval = null;
    this.metricsFile = path.join(__dirname, '../../../logs/pool-metrics.json');
    this.reportFile = path.join(__dirname, '../../../logs/pool-report.json');
    this.alertThresholds = this._initializeAlertThresholds();
    this.alerts = [];
    this.metricsCollectors = [];
    this.isMonitoring = false;
    
    this._initializeLogging();
    this._setupMetricsCollectors();
  }

  _initializeAlertThresholds() {
    return {
      highConnectionUsage: parseFloat(process.env.MONITOR_HIGH_CONNECTION_USAGE) || 0.85,
      criticalConnectionUsage: parseFloat(process.env.MONITOR_CRITICAL_CONNECTION_USAGE) || 0.95,
      highQueryTime: parseInt(process.env.MONITOR_HIGH_QUERY_TIME) || 1000,
      criticalQueryTime: parseInt(process.env.MONITOR_CRITICAL_QUERY_TIME) || 5000,
      highErrorRate: parseFloat(process.env.MONITOR_HIGH_ERROR_RATE) || 0.05,
      criticalErrorRate: parseFloat(process.env.MONITOR_CRITICAL_ERROR_RATE) || 0.10,
      highPoolWaiters: parseInt(process.env.MONITOR_HIGH_POOL_WAITERS) || 5,
      criticalPoolWaiters: parseInt(process.env.MONITOR_CRITICAL_POOL_WAITERS) || 10
    };
  }

  _initializeLogging() {
    const logDir = path.dirname(this.metricsFile);
    if (!fs.existsSync(logDir)) {
      fs.mkdirSync(logDir, { recursive: true });
    }
  }

  _setupMetricsCollectors() {
    this.metricsCollectors = [
      {
        name: 'pool_stats',
        collect: () => this._collectPoolStats(),
        interval: 5000
      },
      {
        name: 'query_stats',
        collect: () => this._collectQueryStats(),
        interval: 10000
      },
      {
        name: 'system_stats',
        collect: () => this._collectSystemStats(),
        interval: 30000
      }
    ];
  }

  start(collectionIntervalMs = 30000) {
    if (this.isMonitoring) {
      console.log('Monitor is already running');
      return;
    }

    this.isMonitoring = true;
    console.log('Connection pool monitor started');

    this.metricsCollectors.forEach(collector => {
      this._startMetricsCollector(collector);
    });

    this.reportInterval = setInterval(() => {
      this._generateAndSaveReport();
    }, collectionIntervalMs);

    this.reportInterval.unref();
  }

  stop() {
    if (!this.isMonitoring) {
      return;
    }

    this.isMonitoring = false;
    
    if (this.collectionInterval) {
      clearInterval(this.collectionInterval);
      this.collectionInterval = null;
    }

    if (this.reportInterval) {
      clearInterval(this.reportInterval);
      this.reportInterval = null;
    }

    console.log('Connection pool monitor stopped');
  }

  _startMetricsCollector(collector) {
    const intervalId = setInterval(() => {
      try {
        const metrics = collector.collect();
        this._storeMetrics(collector.name, metrics);
        this._checkThresholds(collector.name, metrics);
      } catch (error) {
        console.error(`Error collecting ${collector.name} metrics:`, error);
      }
    }, collector.interval);

    intervalId.unref();
  }

  _collectPoolStats() {
    const stats = this.dbPoolManager.getPoolStats();
    const timestamp = new Date().ISOString();

    return {
      timestamp,
      total: stats.total,
      idle: stats.idle,
      busy: stats.busy,
      waiting: stats.waiting,
      checkedOut: stats.checkedOut || 0,
      capacityUsage: parseFloat(stats.capacityUsage) || 0,
      minConnections: stats.config?.min || 0,
      maxConnections: stats.config?.max || 0
    };
  }

  _collectQueryStats() {
    const stats = this.dbPoolManager.getQueryStats();
    const timestamp = new Date().ISOString();

    return {
      timestamp,
      totalQueries: stats.queries,
      slowQueries: stats.slowQueries,
      errors: stats.errors,
      avgQueryTime: stats.avgQueryTime,
      p50QueryTime: stats.p50QueryTime,
      p95QueryTime: stats.p95QueryTime,
      p99QueryTime: stats.p99QueryTime,
      errorRate: parseFloat(stats.errorRate) || 0,
      hitRate: parseFloat(stats.hitRate) || 0
    };
  }

  _collectSystemStats() {
    const os = require('os');
    const timestamp = new Date().ISOString();

    return {
      timestamp,
      cpuUsage: os.loadavg()[0],
      totalMemory: os.totalmem(),
      freeMemory: os.freemem(),
      usedMemory: os.totalmem() - os.freemem(),
      memoryUsagePercent: ((os.totalmem() - os.freemem()) / os.totalmem()) * 100
    };
  }

  _storeMetrics(collectorName, metrics) {
    const metricEntry = {
      collector: collectorName,
      ...metrics
    };

    this.metricsHistory.push(metricEntry);

    if (this.metricsHistory.length > this.maxHistorySize) {
      this.metricsHistory.shift();
    }
  }

  _checkThresholds(collectorName, metrics) {
    if (collectorName === 'pool_stats') {
      this._checkPoolThresholds(metrics);
    } else if (collectorName === 'query_stats') {
      this._checkQueryThresholds(metrics);
    }
  }

  _checkPoolThresholds(metrics) {
    const { capacityUsage, waiting } = metrics;
    const thresholds = this.alertThresholds;

    if (capacityUsage >= thresholds.criticalConnectionUsage) {
      this._emitAlert('critical', 'connection_usage', {
        message: `Critical: Connection usage at ${capacityUsage}%`,
        current: capacityUsage,
        threshold: thresholds.criticalConnectionUsage,
        ...metrics
      });
    } else if (capacityUsage >= thresholds.highConnectionUsage) {
      this._emitAlert('warning', 'connection_usage', {
        message: `Warning: Connection usage at ${capacityUsage}%`,
        current: capacityUsage,
        threshold: thresholds.highConnectionUsage,
        ...metrics
      });
    }

    if (waiting >= thresholds.criticalPoolWaiters) {
      this._emitAlert('critical', 'pool_waiters', {
        message: `Critical: ${waiting} queries waiting for connections`,
        current: waiting,
        threshold: thresholds.criticalPoolWaiters,
        ...metrics
      });
    } else if (waiting >= thresholds.highPoolWaiters) {
      this._emitAlert('warning', 'pool_waiters', {
        message: `Warning: ${waiting} queries waiting for connections`,
        current: waiting,
        threshold: thresholds.highPoolWaiters,
        ...metrics
      });
    }
  }

  _checkQueryThresholds(metrics) {
    const { p95QueryTime, errorRate } = metrics;
    const thresholds = this.alertThresholds;

    if (p95QueryTime >= thresholds.criticalQueryTime) {
      this._emitAlert('critical', 'query_time', {
        message: `Critical: P95 query time at ${p95QueryTime}ms`,
        current: p95QueryTime,
        threshold: thresholds.criticalQueryTime,
        ...metrics
      });
    } else if (p95QueryTime >= thresholds.highQueryTime) {
      this._emitAlert('warning', 'query_time', {
        message: `Warning: P95 query time at ${p95QueryTime}ms`,
        current: p95QueryTime,
        threshold: thresholds.highQueryTime,
        ...metrics
      });
    }

    if (errorRate >= thresholds.criticalErrorRate) {
      this._emitAlert('critical', 'error_rate', {
        message: `Critical: Error rate at ${errorRate}%`,
        current: errorRate,
        threshold: thresholds.criticalErrorRate,
        ...metrics
      });
    } else if (errorRate >= thresholds.highErrorRate) {
      this._emitAlert('warning', 'error_rate', {
        message: `Warning: Error rate at ${errorRate}%`,
        current: errorRate,
        threshold: thresholds.highErrorRate,
        ...metrics
      });
    }
  }

  _emitAlert(severity, type, details) {
    const alert = {
      id: `alert_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
      severity,
      type,
      ...details,
      timestamp: new Date().ISOString()
    };

    this.alerts.push(alert);

    if (this.alerts.length > 100) {
      this.alerts.shift();
    }

    this.emit('alert', alert);

    if (severity === 'critical') {
      this.emit('criticalAlert', alert);
    }
  }

  _generateAndSaveReport() {
    try {
      const report = this.generateHealthReport();
      
      fs.writeFileSync(this.reportFile, JSON.stringify(report, null, 2));
    } catch (error) {
      console.error('Failed to save health report:', error);
    }
  }

  generateHealthReport() {
    const poolStats = this._collectPoolStats();
    const queryStats = this._collectQueryStats();
    const systemStats = this._collectSystemStats();
    const recentMetrics = this.metricsHistory.slice(-100);
    const recentAlerts = this.alerts.slice(-20);

    return {
      generatedAt: new Date().ISOString(),
      overallHealth: this._calculateOverallHealth(poolStats, queryStats),
      pool: poolStats,
      queries: queryStats,
      system: systemStats,
      trends: this._calculateTrends(recentMetrics),
      alerts: {
        count: this.alerts.length,
        recent: recentAlerts
      },
      thresholds: this.alertThresholds,
      recommendations: this._generateRecommendations(poolStats, queryStats)
    };
  }

  _calculateOverallHealth(poolStats, queryStats) {
    let score = 100;
    const penalties = [];

    if (poolStats.capacityUsage > 0.85) {
      score -= (poolStats.capacityUsage - 0.85) * 100;
      penalties.push('high_connection_usage');
    }

    if (queryStats.errorRate > 0.05) {
      score -= queryStats.errorRate * 200;
      penalties.push('high_error_rate');
    }

    if (queryStats.p95QueryTime > 1000) {
      score -= Math.min(20, (queryStats.p95QueryTime - 1000) / 100);
      penalties.push('high_query_time');
    }

    if (this.alerts.length > 10) {
      score -= Math.min(20, (this.alerts.length - 10) * 2);
      penalties.push('too_many_alerts');
    }

    return {
      score: Math.max(0, Math.round(score)),
      status: score >= 80 ? 'healthy' : score >= 60 ? 'warning' : 'critical',
      penalties
    };
  }

  _calculateTrends(recentMetrics) {
    const poolMetrics = recentMetrics.filter(m => m.collector === 'pool_stats');
    
    if (poolMetrics.length < 2) {
      return { connectionUsage: 'stable', queryTime: 'stable' };
    }

    const recentHalf = poolMetrics.slice(Math.floor(poolMetrics.length / 2));
    const olderHalf = poolMetrics.slice(0, Math.floor(poolMetrics.length / 2));

    const avgRecentUsage = recentHalf.reduce((sum, m) => sum + (m.capacityUsage || 0), 0) / recentHalf.length;
    const avgOlderUsage = olderHalf.reduce((sum, m) => sum + (m.capacityUsage || 0), 0) / olderHalf.length;

    let usageTrend = 'stable';
    if (avgRecentUsage > avgOlderUsage * 1.2) {
      usageTrend = 'increasing';
    } else if (avgRecentUsage < avgOlderUsage * 0.8) {
      usageTrend = 'decreasing';
    }

    const queryMetrics = recentMetrics.filter(m => m.collector === 'query_stats');
    let queryTrend = 'stable';
    
    if (queryMetrics.length >= 2) {
      const recentQueryTime = queryMetrics.slice(-5).reduce((sum, m) => sum + (m.p95QueryTime || 0), 0) / Math.min(5, queryMetrics.length);
      const olderQueryTime = queryMetrics.slice(0, 5).reduce((sum, m) => sum + (m.p95QueryTime || 0), 0) / Math.min(5, queryMetrics.length);

      if (recentQueryTime > olderQueryTime * 1.3) {
        queryTrend = 'increasing';
      } else if (recentQueryTime < olderQueryTime * 0.7) {
        queryTrend = 'decreasing';
      }
    }

    return {
      connectionUsage: usageTrend,
      queryTime: queryTrend
    };
  }

  _generateRecommendations(poolStats, queryStats) {
    const recommendations = [];

    if (poolStats.capacityUsage > 0.85) {
      recommendations.push({
        priority: 'high',
        type: 'pool_size',
        message: 'Connection pool is highly utilized. Consider increasing max connections or optimizing queries.',
        currentUsage: poolStats.capacityUsage
      });
    }

    if (queryStats.p95QueryTime > 1000) {
      recommendations.push({
        priority: 'medium',
        type: 'query_optimization',
        message: 'Query performance is degrading. Review slow queries and optimize indexes.',
        p95Time: queryStats.p95QueryTime
      });
    }

    if (queryStats.slowQueries > 10) {
      recommendations.push({
        priority: 'medium',
        type: 'slow_queries',
        message: 'High number of slow queries detected. Consider query optimization or increasing resources.',
        slowQueryCount: queryStats.slowQueries
      });
    }

    if (queryStats.errorRate > 0.05) {
      recommendations.push({
        priority: 'high',
        type: 'error_rate',
        message: 'Error rate is elevated. Investigate error logs for root cause.',
        errorRate: queryStats.errorRate
      });
    }

    if (poolStats.waiting > 5) {
      recommendations.push({
        priority: 'high',
        type: 'connection_contention',
        message: 'Queries are waiting for connections. This may indicate connection leaks or pool exhaustion.',
        waitingCount: poolStats.waiting
      });
    }

    return recommendations;
  }

  getMetricsHistory(collectorName, limit = 100) {
    const filtered = collectorName
      ? this.metricsHistory.filter(m => m.collector === collectorName)
      : this.metricsHistory;

    return filtered.slice(-limit);
  }

  getAggregatedMetrics(timeRange = '1h') {
    const now = Date.now();
    const ranges = {
      '10m': 10 * 60 * 1000,
      '30m': 30 * 60 * 1000,
      '1h': 60 * 60 * 1000,
      '6h': 6 * 60 * 60 * 1000,
      '24h': 24 * 60 * 60 * 1000
    };

    const rangeMs = ranges[timeRange] || ranges['1h'];
    const cutoffTime = now - rangeMs;

    const relevantMetrics = this.metricsHistory.filter(
      m => new Date(m.timestamp).getTime() > cutoffTime
    );

    const poolMetrics = relevantMetrics.filter(m => m.collector === 'pool_stats');
    const queryMetrics = relevantMetrics.filter(m => m.collector === 'query_stats');
    const systemMetrics = relevantMetrics.filter(m => m.collector === 'system_stats');

    return {
      timeRange,
      dataPoints: relevantMetrics.length,
      pool: {
        avgCapacityUsage: this._average(poolMetrics.map(m => m.capacityUsage)),
        maxCapacityUsage: Math.max(...poolMetrics.map(m => m.capacityUsage || 0), 0),
        avgBusyConnections: this._average(poolMetrics.map(m => m.busy)),
        avgIdleConnections: this._average(poolMetrics.map(m => m.idle)),
        maxWaiting: Math.max(...poolMetrics.map(m => m.waiting || 0), 0)
      },
      queries: {
        totalQueries: queryMetrics.reduce((sum, m) => sum + (m.totalQueries || 0), 0),
        totalErrors: queryMetrics.reduce((sum, m) => sum + (m.errors || 0), 0),
        avgQueryTime: this._average(queryMetrics.map(m => m.avgQueryTime)),
        p95QueryTime: this._average(queryMetrics.map(m => m.p95QueryTime)),
        avgErrorRate: this._average(queryMetrics.map(m => m.errorRate))
      },
      system: {
        avgCpuLoad: this._average(systemMetrics.map(m => m.cpuUsage)),
        avgMemoryUsage: this._average(systemMetrics.map(m => m.memoryUsagePercent)),
        maxMemoryUsage: Math.max(...systemMetrics.map(m => m.memoryUsagePercent || 0), 0)
      }
    };
  }

  _average(values) {
    const validValues = values.filter(v => typeof v === 'number' && !isNaN(v));
    if (validValues.length === 0) return 0;
    return validValues.reduce((sum, v) => sum + v, 0) / validValues.length;
  }

  getAlerts(severity = null, limit = 50) {
    let filtered = this.alerts;
    
    if (severity) {
      filtered = filtered.filter(a => a.severity === severity);
    }

    return filtered.slice(-limit);
  }

  clearAlerts() {
    this.alerts = [];
    console.log('All alerts cleared');
    this.emit('alertsCleared');
  }

  setAlertThreshold(type, value) {
    if (this.alertThresholds.hasOwnProperty(type)) {
      this.alertThresholds[type] = value;
      console.log(`Alert threshold ${type} updated to ${value}`);
      this.emit('thresholdUpdated', { type, value });
    }
  }

  exportMetrics(format = 'json') {
    const data = {
      exportedAt: new Date().ISOString(),
      metricsHistory: this.metricsHistory,
      alerts: this.alerts,
      aggregatedMetrics: this.getAggregatedMetrics('24h'),
      healthReport: this.generateHealthReport()
    };

    if (format === 'csv') {
      return this._convertMetricsToCSV(data.metricsHistory);
    }

    return JSON.stringify(data, null, 2);
  }

  _convertMetricsToCSV(metrics) {
    if (metrics.length === 0) {
      return 'No metrics to export';
    }

    const headers = Object.keys(metrics[0]);
    const rows = metrics.map(m => headers.map(h => m[h] || '').join(','));

    return [headers.join(','), ...rows].join('\n');
  }

  forceCollection() {
    const collectedMetrics = {};
    
    this.metricsCollectors.forEach(collector => {
      try {
        collectedMetrics[collector.name] = collector.collect();
      } catch (error) {
        console.error(`Error collecting ${collector.name}:`, error);
        collectedMetrics[collector.name] = null;
      }
    });

    return collectedMetrics;
  }

  reset() {
    this.metricsHistory = [];
    this.alerts = [];
    console.log('Monitor metrics and alerts reset');
    this.emit('reset');
  }
}

module.exports = ConnectionPoolMonitor;
