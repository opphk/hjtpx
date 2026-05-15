const EventEmitter = require('events');
const fs = require('fs');
const path = require('path');

class ConnectionHealthCheck extends EventEmitter {
  constructor(dbPoolManager) {
    super();
    this.dbPoolManager = dbPoolManager;
    this.checkInterval = null;
    this.healthHistory = [];
    this.maxHistorySize = 100;
    this.lastCheck = null;
    this.consecutiveFailures = 0;
    this.maxConsecutiveFailures = 3;
    this.healthThresholds = {
      maxResponseTime: parseInt(process.env.DB_HEALTH_MAX_RESPONSE_TIME) || 5000,
      minIdleConnections: parseInt(process.env.DB_HEALTH_MIN_IDLE) || 2,
      maxConnectionUsage: parseFloat(process.env.DB_HEALTH_MAX_USAGE) || 0.9
    };
    this.checkCount = 0;
    this.healthScore = 100;
    this.isHealthy = true;
    this.reportFile = path.join(__dirname, '../../../logs/health-report.json');
    
    this._initializeLogging();
  }

  _initializeLogging() {
    const logDir = path.dirname(this.reportFile);
    if (!fs.existsSync(logDir)) {
      fs.mkdirSync(logDir, { recursive: true });
    }
  }

  start(intervalMs = 30000) {
    if (this.checkInterval) {
      this.stop();
    }

    this.checkInterval = setInterval(() => {
      this.performHealthCheck();
    }, intervalMs);

    this.checkInterval.unref();
    console.log(`Connection health check started with interval: ${intervalMs}ms`);
    
    this.performHealthCheck();
  }

  stop() {
    if (this.checkInterval) {
      clearInterval(this.checkInterval);
      this.checkInterval = null;
      console.log('Connection health check stopped');
    }
  }

  async performHealthCheck() {
    const startTime = Date.now();
    this.checkCount++;
    
    try {
      const checks = await Promise.allSettled([
        this._checkDatabaseConnectivity(),
        this._checkConnectionPoolStatus(),
        this._checkQueryPerformance(),
        this._checkDatabaseSize(),
        this._checkActiveConnections()
      ]);

      const results = checks.map((check, index) => ({
        checkName: this._getCheckName(index),
        status: check.status === 'fulfilled' ? 'passed' : 'failed',
        ...(check.status === 'fulfilled' ? check.value : { error: check.reason?.message })
      }));

      const duration = Date.now() - startTime;
      const allPassed = results.every(r => r.status === 'passed');
      
      this.lastCheck = {
        timestamp: new Date().toISOString(),
        duration,
        checks: results,
        overallStatus: allPassed ? 'healthy' : 'unhealthy',
        healthScore: this._calculateScore(results)
      };

      this._updateHealthHistory(this.lastCheck);
      this._updateHealthState(allPassed, results);
      this._emitHealthEvents(results, allPassed);
      this._logHealthReport();

      if (!allPassed) {
        this._handleUnhealthyState(results);
      }

      return this.lastCheck;
    } catch (error) {
      this.consecutiveFailures++;
      this.isHealthy = false;
      this.healthScore = 0;
      
      this.lastCheck = {
        timestamp: new Date().ISOString(),
        duration: Date.now() - startTime,
        overallStatus: 'unhealthy',
        error: error.message,
        healthScore: 0
      };

      this.emit('healthCheckFailed', this.lastCheck);
      return this.lastCheck;
    }
  }

  _getCheckName(index) {
    const names = [
      'database_connectivity',
      'connection_pool_status',
      'query_performance',
      'database_size',
      'active_connections'
    ];
    return names[index] || `check_${index}`;
  }

  async _checkDatabaseConnectivity() {
    const start = Date.now();
    const result = await this.dbPoolManager.query(
      'SELECT 1 as health, NOW() as timestamp, version() as version'
    );
    const responseTime = Date.now() - start;

    return {
      responseTime,
      timestamp: result.rows[0].timestamp,
      version: result.rows[0].version,
      passed: responseTime < this.healthThresholds.maxResponseTime
    };
  }

  async _checkConnectionPoolStatus() {
    const stats = this.dbPoolManager.getPoolStats();
    
    if (stats.error) {
      throw new Error(stats.error);
    }

    const idleConnections = stats.idle;
    const totalConnections = stats.total;
    const usageRatio = totalConnections > 0 ? (totalConnections - idleConnections) / totalConnections : 0;

    return {
      totalConnections,
      idleConnections,
      busyConnections: totalConnections - idleConnections,
      usageRatio,
      passed: idleConnections >= this.healthThresholds.minIdleConnections &&
              usageRatio < this.healthThresholds.maxConnectionUsage
    };
  }

  async _checkQueryPerformance() {
    const start = Date.now();
    await this.dbPoolManager.query('SELECT 1');
    const responseTime = Date.now() - start;

    return {
      responseTime,
      threshold: this.healthThresholds.maxResponseTime,
      passed: responseTime < this.healthThresholds.maxResponseTime
    };
  }

  async _checkDatabaseSize() {
    const result = await this.dbPoolManager.query(
      'SELECT pg_database_size(current_database()) as size'
    );
    
    const sizeBytes = result.rows[0].size;
    const sizeMB = sizeBytes / (1024 * 1024);
    const sizeGB = sizeMB / 1024;

    return {
      sizeBytes,
      sizeMB: Math.round(sizeMB * 100) / 100,
      sizeGB: Math.round(sizeGB * 100) / 100,
      passed: true
    };
  }

  async _checkActiveConnections() {
    const result = await this.dbPoolManager.query(
      'SELECT count(*) as active_connections FROM pg_stat_activity WHERE state = $1',
      ['active']
    );

    const activeConnections = parseInt(result.rows[0].active_connections);
    const stats = this.dbPoolManager.getPoolStats();
    const maxConnections = stats.config?.max || 100;
    const usageRatio = activeConnections / maxConnections;

    return {
      activeConnections,
      maxConnections,
      usageRatio: Math.round(usageRatio * 100) / 100,
      passed: usageRatio < this.healthThresholds.maxConnectionUsage
    };
  }

  _calculateScore(results) {
    const passedChecks = results.filter(r => r.status === 'passed').length;
    const totalChecks = results.length;
    const baseScore = (passedChecks / totalChecks) * 100;

    const penalties = results
      .filter(r => r.status === 'passed' && r.passed !== undefined && !r.passed)
      .length * 10;

    return Math.max(0, baseScore - penalties);
  }

  _updateHealthHistory(check) {
    this.healthHistory.push(check);
    if (this.healthHistory.length > this.maxHistorySize) {
      this.healthHistory.shift();
    }
  }

  _updateHealthState(isHealthy, results) {
    if (isHealthy) {
      this.consecutiveFailures = 0;
      this.isHealthy = true;
    } else {
      this.consecutiveFailures++;
    }

    this.healthScore = this._calculateScore(results);
  }

  _emitHealthEvents(results, allPassed) {
    this.emit('healthCheck', this.lastCheck);
    
    if (!allPassed) {
      const failedChecks = results.filter(r => r.status !== 'passed');
      this.emit('healthChecksFailed', {
        timestamp: this.lastCheck.timestamp,
        failedChecks
      });
    }

    if (this.consecutiveFailures >= this.maxConsecutiveFailures && !this.isHealthy) {
      this.emit('unhealthyState', {
        consecutiveFailures: this.consecutiveFailures,
        lastCheck: this.lastCheck
      });
    }
  }

  _handleUnhealthyState(results) {
    const failedChecks = results
      .filter(r => r.status !== 'passed')
      .map(r => r.checkName);

    console.error(`Health check failed! Failed checks: ${failedChecks.join(', ')}`);
    
    if (this.consecutiveFailures >= this.maxConsecutiveFailures) {
      console.error(`Critical: ${this.consecutiveFailures} consecutive health check failures detected`);
      this.emit('criticalHealthFailure', {
        consecutiveFailures: this.consecutiveFailures,
        failedChecks
      });
    }
  }

  _logHealthReport() {
    try {
      const report = {
        generatedAt: new Date().ISOString(),
        checkCount: this.checkCount,
        currentHealth: this.lastCheck,
        healthHistory: this.healthHistory.slice(-10),
        thresholds: this.healthThresholds,
        consecutiveFailures: this.consecutiveFailures,
        healthScore: this.healthScore,
        isHealthy: this.isHealthy
      };

      fs.writeFileSync(this.reportFile, JSON.stringify(report, null, 2));
    } catch (error) {
      console.error('Failed to write health report:', error);
    }
  }

  async performDeepHealthCheck() {
    const checks = await Promise.allSettled([
      this._checkDatabaseConnectivity(),
      this._checkConnectionPoolStatus(),
      this._checkDatabaseSize(),
      this._checkLongRunningQueries(),
      this._checkReplicationStatus(),
      this._checkDiskSpace(),
      this._checkLockWaits(),
      this._checkCacheHitRatio()
    ]);

    return {
      timestamp: new Date().toISOString(),
      checks: checks.map((check, index) => ({
        checkName: this._getDeepCheckName(index),
        status: check.status === 'fulfilled' ? 'passed' : 'failed',
        ...(check.status === 'fulfilled' ? check.value : { error: check.reason?.message })
      }))
    };
  }

  _getDeepCheckName(index) {
    const names = [
      'database_connectivity',
      'connection_pool_status',
      'database_size',
      'long_running_queries',
      'replication_status',
      'disk_space',
      'lock_waits',
      'cache_hit_ratio'
    ];
    return names[index] || `deep_check_${index}`;
  }

  async _checkLongRunningQueries() {
    const result = await this.dbPoolManager.query(
      `SELECT pid, now() - query_start as duration, state, query 
       FROM pg_stat_activity 
       WHERE state != 'idle' 
       AND query_start < now() - interval '5 minutes'
       ORDER BY query_start`
    );

    const longRunning = result.rows;
    return {
      count: longRunning.length,
      queries: longRunning.map(q => ({
        pid: q.pid,
        duration: q.duration,
        state: q.state
      })),
      passed: longRunning.length < 10
    };
  }

  async _checkReplicationStatus() {
    try {
      const result = await this.dbPoolManager.query(
        'SELECT * FROM pg_stat_replication'
      );
      
      return {
        replicationSlots: result.rows.length,
        passed: true
      };
    } catch (error) {
      return {
        replicationSlots: 0,
        passed: true,
        note: 'Replication not configured or primary database'
      };
    }
  }

  async _checkDiskSpace() {
    const result = await this.dbPoolManager.query(
      `SELECT pg_total_relation_size($1) as total_size, 
              pg_database_size($1) as database_size`,
      [this.dbPoolManager.config.database]
    );

    return {
      totalSize: result.rows[0].total_size,
      databaseSize: result.rows[0].database_size,
      passed: true
    };
  }

  async _checkLockWaits() {
    const result = await this.dbPoolManager.query(
      `SELECT count(*) as lock_waits 
       FROM pg_stat_activity 
       WHERE wait_event_type = 'Lock'`
    );

    const lockWaits = parseInt(result.rows[0].lock_waits);
    return {
      lockWaits,
      passed: lockWaits < 5
    };
  }

  async _checkCacheHitRatio() {
    const result = await this.dbPoolManager.query(
      `SELECT 
        sum(heap_blks_read) as heap_read,
        sum(heap_blks_hit) as heap_hit,
        sum(heap_blks_hit) / nullif(sum(heap_blks_hit) + sum(heap_blks_read), 0) * 100 as ratio
       FROM pg_statio_user_tables`
    );

    const ratio = parseFloat(result.rows[0]?.ratio) || 0;
    return {
      ratio: Math.round(ratio * 100) / 100,
      passed: ratio > 80
    };
  }

  getHealthReport() {
    return {
      checkCount: this.checkCount,
      healthScore: this.healthScore,
      isHealthy: this.isHealthy,
      consecutiveFailures: this.consecutiveFailures,
      lastCheck: this.lastCheck,
      history: this.healthHistory.slice(-10),
      thresholds: this.healthThresholds
    };
  }

  getDetailedStats() {
    return {
      summary: this.getHealthReport(),
      recentHistory: this.healthHistory,
      allTimeStats: {
        totalChecks: this.checkCount,
        averageHealthScore: this.healthHistory.length > 0
          ? this.healthHistory.reduce((sum, h) => sum + (h.healthScore || 0), 0) / this.healthHistory.length
          : 0
      }
    };
  }

  reset() {
    this.healthHistory = [];
    this.checkCount = 0;
    this.consecutiveFailures = 0;
    this.healthScore = 100;
    this.isHealthy = true;
  }
}

module.exports = ConnectionHealthCheck;
