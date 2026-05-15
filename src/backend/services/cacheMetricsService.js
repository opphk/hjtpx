const cacheConfig = require('../config/cacheConfig');

class CacheMetricsService {
  constructor() {
    this.metrics = {
      cache: {
        hits: 0,
        misses: 0,
        sets: 0,
        deletes: 0,
        errors: 0
      },
      latency: {
        get: [],
        set: [],
        del: []
      },
      memory: {
        currentSize: 0,
        maxSize: cacheConfig.maxSize.MEMORY_CACHE,
        evictions: 0
      },
      errors: {
        total: 0,
        byType: {}
      },
      session: {
        creates: 0,
        reads: 0,
        updates: 0,
        deletes: 0,
        expired: 0,
        active: 0,
        totalOperations: 0
      },
      api: {
        hits: 0,
        misses: 0,
        size: 0,
        endpoints: {}
      },
      performance: {
        concurrentOperations: 0,
        operationsPerSecond: 0,
        bytesRead: 0,
        bytesWritten: 0
      },
      history: [],
      startTime: Date.now()
    };

    this.maxHistorySize = 1440;
    this.statsCollectionInterval = null;
    this.startStatsCollection();
  }

  startStatsCollection() {
    if (cacheConfig.stats && cacheConfig.stats.ENABLED) {
      const interval = cacheConfig.stats.COLLECTION_INTERVAL || 60000;
      this.statsCollectionInterval = setInterval(() => {
        this.recordSnapshot();
      }, interval);
    }
  }

  recordHit(type = 'general', count = 1) {
    this.metrics.cache.hits += count;
    this.metrics.performance.operationsPerSecond += count;
  }

  recordMiss(type = 'general', count = 1) {
    this.metrics.cache.misses += count;
  }

  recordSet(type = 'general', count = 1) {
    this.metrics.cache.sets += count;
    this.metrics.performance.operationsPerSecond += count;
  }

  recordDelete(type = 'general', count = 1) {
    this.metrics.cache.deletes += count;
    this.metrics.performance.operationsPerSecond += count;
  }

  recordLatency(type, latencyMs) {
    if (!this.metrics.latency[type]) {
      this.metrics.latency[type] = [];
    }

    this.metrics.latency[type].push(latencyMs);

    if (this.metrics.latency[type].length > 10000) {
      this.metrics.latency[type].shift();
    }
  }

  setMemoryCacheSize(size) {
    this.metrics.memory.currentSize = size;
  }

  setMemoryCacheMaxSize(maxSize) {
    this.metrics.memory.maxSize = maxSize;
  }

  recordEviction(strategy = 'LRU', count = 1) {
    this.metrics.memory.evictions += count;
  }

  recordError(type, message) {
    this.metrics.cache.errors++;
    this.metrics.errors.total++;

    if (!this.metrics.errors.byType[type]) {
      this.metrics.errors.byType[type] = 0;
    }
    this.metrics.errors.byType[type]++;
  }

  recordSessionOperation(operation, count = 1) {
    this.metrics.session.totalOperations += count;

    switch (operation) {
      case 'create':
        this.metrics.session.creates += count;
        break;
      case 'read':
        this.metrics.session.reads += count;
        break;
      case 'update':
        this.metrics.session.updates += count;
        break;
      case 'delete':
        this.metrics.session.deletes += count;
        break;
    }
  }

  setActiveSessions(count) {
    this.metrics.session.active = count;
  }

  incrementActiveSessions() {
    this.metrics.session.active++;
  }

  decrementActiveSessions() {
    if (this.metrics.session.active > 0) {
      this.metrics.session.active--;
    }
  }

  recordSessionExpiration(count = 1) {
    this.metrics.session.expired += count;
  }

  recordApiCacheHit(endpoint) {
    this.metrics.api.hits++;

    if (!this.metrics.api.endpoints[endpoint]) {
      this.metrics.api.endpoints[endpoint] = {
        hits: 0,
        misses: 0,
        totalLatency: 0,
        requestCount: 0
      };
    }

    this.metrics.api.endpoints[endpoint].hits++;
    this.metrics.api.endpoints[endpoint].requestCount++;
  }

  recordApiCacheMiss(endpoint) {
    this.metrics.api.misses++;

    if (!this.metrics.api.endpoints[endpoint]) {
      this.metrics.api.endpoints[endpoint] = {
        hits: 0,
        misses: 0,
        totalLatency: 0,
        requestCount: 0
      };
    }

    this.metrics.api.endpoints[endpoint].misses++;
    this.metrics.api.endpoints[endpoint].requestCount++;
  }

  recordApiLatency(endpoint, latencyMs) {
    if (this.metrics.api.endpoints[endpoint]) {
      this.metrics.api.endpoints[endpoint].totalLatency += latencyMs;
    }
  }

  setApiCacheSize(size) {
    this.metrics.api.size = size;
  }

  setConcurrentOperations(count) {
    this.metrics.performance.concurrentOperations = count;
  }

  incrementConcurrentOperations() {
    this.metrics.performance.concurrentOperations++;
  }

  decrementConcurrentOperations() {
    if (this.metrics.performance.concurrentOperations > 0) {
      this.metrics.performance.concurrentOperations--;
    }
  }

  recordBytesRead(bytes) {
    this.metrics.performance.bytesRead += bytes;
  }

  recordBytesWritten(bytes) {
    this.metrics.performance.bytesWritten += bytes;
  }

  recordOperation(type, bytes = 0) {
    this.metrics.performance.operationsPerSecond++;
    if (bytes > 0) {
      this.recordBytesRead(bytes);
    }
  }

  getHitRate(type = 'general') {
    const total = this.metrics.cache.hits + this.metrics.cache.misses;
    if (total === 0) return 0;
    return (this.metrics.cache.hits / total) * 100;
  }

  getAverageLatency(type) {
    const latencies = this.metrics.latency[type];
    if (!latencies || latencies.length === 0) return 0;

    const sum = latencies.reduce((a, b) => a + b, 0);
    return sum / latencies.length;
  }

  getPercentileLatency(type, percentile) {
    const latencies = [...(this.metrics.latency[type] || [])].sort((a, b) => a - b);
    if (latencies.length === 0) return 0;

    const index = Math.floor((percentile / 100) * latencies.length);
    return latencies[index] || latencies[latencies.length - 1];
  }

  getMemoryUsagePercent() {
    if (this.metrics.memory.maxSize === 0) return 0;
    return (this.metrics.memory.currentSize / this.metrics.memory.maxSize) * 100;
  }

  getErrorRate() {
    const totalOperations = this.metrics.cache.hits + this.metrics.cache.misses +
                          this.metrics.cache.sets + this.metrics.cache.deletes;
    if (totalOperations === 0) return 0;
    return (this.metrics.cache.errors / totalOperations) * 100;
  }

  getEndpointHitRate(endpoint) {
    const endpointMetrics = this.metrics.api.endpoints[endpoint];
    if (!endpointMetrics) return 0;

    const total = endpointMetrics.hits + endpointMetrics.misses;
    if (total === 0) return 0;

    return (endpointMetrics.hits / total) * 100;
  }

  getThroughput() {
    const uptime = (Date.now() - this.metrics.startTime) / 1000;
    const totalOps = this.metrics.cache.hits + this.metrics.cache.misses +
                     this.metrics.cache.sets + this.metrics.cache.deletes;
    return uptime > 0 ? parseFloat((totalOps / uptime).toFixed(2)) : 0;
  }

  recordSnapshot() {
    const snapshot = {
      timestamp: Date.now(),
      hits: this.metrics.cache.hits,
      misses: this.metrics.cache.misses,
      sets: this.metrics.cache.sets,
      deletes: this.metrics.cache.deletes,
      errors: this.metrics.cache.errors,
      hitRate: this.getHitRate(),
      errorRate: this.getErrorRate(),
      memoryUsage: this.getMemoryUsagePercent(),
      activeSessions: this.metrics.session.active,
      concurrentOperations: this.metrics.performance.concurrentOperations,
      throughput: this.getThroughput()
    };

    this.metrics.history.push(snapshot);

    if (this.metrics.history.length > this.maxHistorySize) {
      this.metrics.history.shift();
    }
  }

  getHistory(count = 10) {
    return this.metrics.history.slice(-count);
  }

  getTrend(metric) {
    const history = this.getHistory(10);
    if (history.length < 2) return null;

    const first = history[0][metric];
    const last = history[history.length - 1][metric];

    if (first === 0) return null;

    const change = ((last - first) / first) * 100;
    return {
      first,
      last,
      change: change.toFixed(2),
      direction: change > 0 ? 'up' : change < 0 ? 'down' : 'stable'
    };
  }

  checkAlerts() {
    const alerts = [];

    const errorRate = this.getErrorRate();
    if (errorRate > 5) {
      alerts.push({
        type: 'high_error_rate',
        severity: 'warning',
        message: `Error rate is ${errorRate.toFixed(2)}%`,
        timestamp: Date.now()
      });
    }

    const hitRate = this.getHitRate();
    if (hitRate < 50 && this.metrics.cache.hits + this.metrics.cache.misses > 100) {
      alerts.push({
        type: 'low_hit_rate',
        severity: 'warning',
        message: `Hit rate is ${hitRate.toFixed(2)}%`,
        timestamp: Date.now()
      });
    }

    const memoryUsage = this.getMemoryUsagePercent();
    if (memoryUsage > 90) {
      alerts.push({
        type: 'high_memory_usage',
        severity: 'critical',
        message: `Memory usage is ${memoryUsage.toFixed(2)}%`,
        timestamp: Date.now()
      });
    } else if (memoryUsage > 80) {
      alerts.push({
        type: 'high_memory_usage',
        severity: 'warning',
        message: `Memory usage is ${memoryUsage.toFixed(2)}%`,
        timestamp: Date.now()
      });
    }

    return alerts;
  }

  getMetrics() {
    return {
      cache: { ...this.metrics.cache },
      latency: {
        get: [...this.metrics.latency.get],
        set: [...this.metrics.latency.set],
        del: [...this.metrics.latency.del]
      },
      memory: { ...this.metrics.memory },
      errors: {
        total: this.metrics.errors.total,
        byType: { ...this.metrics.errors.byType }
      },
      session: { ...this.metrics.session },
      api: {
        hits: this.metrics.api.hits,
        misses: this.metrics.api.misses,
        size: this.metrics.api.size,
        endpoints: { ...this.metrics.api.endpoints }
      },
      performance: { ...this.metrics.performance },
      history: [...this.metrics.history],
      uptime: Date.now() - this.metrics.startTime
    };
  }

  getSummary() {
    return {
      hitRate: this.getHitRate().toFixed(2) + '%',
      errorRate: this.getErrorRate().toFixed(2) + '%',
      memoryUsage: this.getMemoryUsagePercent().toFixed(2) + '%',
      totalOperations: this.metrics.cache.hits + this.metrics.cache.misses +
                      this.metrics.cache.sets + this.metrics.cache.deletes,
      activeSessions: this.metrics.session.active,
      throughput: this.getThroughput(),
      uptime: this.formatUptime()
    };
  }

  formatUptime() {
    const uptime = Date.now() - this.metrics.startTime;
    const seconds = Math.floor(uptime / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);

    if (days > 0) return `${days}d ${hours % 24}h`;
    if (hours > 0) return `${hours}h ${minutes % 60}m`;
    if (minutes > 0) return `${minutes}m ${seconds % 60}s`;
    return `${seconds}s`;
  }

  exportMetrics(format = 'json') {
    if (format === 'json') {
      return JSON.stringify({
        timestamp: Date.now(),
        metrics: this.getMetrics(),
        summary: this.getSummary(),
        alerts: this.checkAlerts()
      }, null, 2);
    }

    return this.getMetrics();
  }

  resetMetrics() {
    this.metrics.cache = {
      hits: 0,
      misses: 0,
      sets: 0,
      deletes: 0,
      errors: 0
    };

    this.metrics.latency = {
      get: [],
      set: [],
      del: []
    };

    this.metrics.errors = {
      total: 0,
      byType: {}
    };

    this.metrics.session = {
      creates: 0,
      reads: 0,
      updates: 0,
      deletes: 0,
      expired: 0,
      active: this.metrics.session.active,
      totalOperations: 0
    };

    this.metrics.api = {
      hits: 0,
      misses: 0,
      size: this.metrics.api.size,
      endpoints: {}
    };

    this.metrics.performance = {
      concurrentOperations: 0,
      operationsPerSecond: 0,
      bytesRead: 0,
      bytesWritten: 0
    };

    this.metrics.history = [];
    this.metrics.startTime = Date.now();
  }

  async cleanup() {
    if (this.statsCollectionInterval) {
      clearInterval(this.statsCollectionInterval);
      this.statsCollectionInterval = null;
    }

    this.metrics.latency.get = [];
    this.metrics.latency.set = [];
    this.metrics.latency.del = [];
    this.metrics.history = [];
  }
}

const cacheMetricsService = new CacheMetricsService();

module.exports = cacheMetricsService;
