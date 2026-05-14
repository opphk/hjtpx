const os = require('os');

class PerformanceMonitor {
  constructor() {
    this.metrics = {
      requests: 0,
      errors: 0,
      totalResponseTime: 0,
      avgResponseTime: 0,
      minResponseTime: Infinity,
      maxResponseTime: 0,
      slowRequests: 0,
      memoryUsage: [],
      cpuUsage: []
    };
    this.responseTimes = [];
    this.maxResponseTimes = 1000;
    this.slowRequestThreshold = parseInt(process.env.SLOW_REQUEST_THRESHOLD) || 1000;
    this.startTime = Date.now();
  }

  recordRequest(responseTime, statusCode = 200) {
    this.metrics.requests++;

    this.responseTimes.push(responseTime);
    if (this.responseTimes.length > this.maxResponseTimes) {
      this.responseTimes.shift();
    }

    this.metrics.totalResponseTime += responseTime;
    this.metrics.avgResponseTime =
      this.metrics.totalResponseTime / this.metrics.requests;

    if (responseTime < this.metrics.minResponseTime) {
      this.metrics.minResponseTime = responseTime;
    }

    if (responseTime > this.metrics.maxResponseTime) {
      this.metrics.maxResponseTime = responseTime;
    }

    if (responseTime > this.slowRequestThreshold) {
      this.metrics.slowRequests++;
    }

    if (statusCode >= 400) {
      this.metrics.errors++;
    }

    this.recordSystemMetrics();
  }

  recordSystemMetrics() {
    const memoryUsage = process.memoryUsage();
    const cpuUsage = process.cpuUsage();

    this.metrics.memoryUsage.push({
      heapUsed: memoryUsage.heapUsed,
      heapTotal: memoryUsage.heapTotal,
      external: memoryUsage.external,
      rss: memoryUsage.rss,
      timestamp: Date.now()
    });

    this.metrics.cpuUsage.push({
      user: cpuUsage.user,
      system: cpuUsage.system,
      timestamp: Date.now()
    });

    if (this.metrics.memoryUsage.length > 60) {
      this.metrics.memoryUsage.shift();
    }

    if (this.metrics.cpuUsage.length > 60) {
      this.metrics.cpuUsage.shift();
    }
  }

  getMetrics() {
    const uptime = Date.now() - this.startTime;
    const memoryUsage = process.memoryUsage();
    const cpuUsage = process.cpuUsage();

    const percentiles = this.calculatePercentiles();

    return {
      uptime: Math.floor(uptime / 1000),
      requests: {
        total: this.metrics.requests,
        errors: this.metrics.errors,
        errorRate: this.metrics.requests > 0
          ? ((this.metrics.errors / this.metrics.requests) * 100).toFixed(2) + '%'
          : '0%'
      },
      responseTime: {
        avg: Math.round(this.metrics.avgResponseTime * 100) / 100,
        min: Math.round(this.metrics.minResponseTime * 100) / 100,
        max: Math.round(this.metrics.maxResponseTime * 100) / 100,
        p50: percentiles.p50,
        p90: percentiles.p90,
        p95: percentiles.p95,
        p99: percentiles.p99
      },
      slowRequests: {
        count: this.metrics.slowRequests,
        threshold: this.slowRequestThreshold
      },
      memory: {
        heapUsed: this.formatBytes(memoryUsage.heapUsed),
        heapTotal: this.formatBytes(memoryUsage.heapTotal),
        external: this.formatBytes(memoryUsage.external),
        rss: this.formatBytes(memoryUsage.rss),
        systemTotal: this.formatBytes(os.totalmem()),
        systemFree: this.formatBytes(os.freemem()),
        systemUsed: this.formatBytes(os.totalmem() - os.freemem())
      },
      cpu: {
        current: {
          user: cpuUsage.user,
          system: cpuUsage.system
        },
        cores: os.cpus().length,
        loadAverage: os.loadavg()
      },
      percentiles
    };
  }

  calculatePercentiles() {
    if (this.responseTimes.length === 0) {
      return { p50: 0, p90: 0, p95: 0, p99: 0 };
    }

    const sorted = [...this.responseTimes].sort((a, b) => a - b);
    const percentiles = {};

    const percentileRanks = [50, 90, 95, 99];

    for (const rank of percentileRanks) {
      const index = Math.ceil((rank / 100) * sorted.length) - 1;
      percentiles[`p${rank}`] = Math.round(sorted[index] * 100) / 100;
    }

    return percentiles;
  }

  formatBytes(bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  }

  reset() {
    this.metrics = {
      requests: 0,
      errors: 0,
      totalResponseTime: 0,
      avgResponseTime: 0,
      minResponseTime: Infinity,
      maxResponseTime: 0,
      slowRequests: 0,
      memoryUsage: [],
      cpuUsage: []
    };
    this.responseTimes = [];
    this.startTime = Date.now();
  }

  healthCheck() {
    const memoryUsage = process.memoryUsage();
    const heapUsedPercent = (memoryUsage.heapUsed / memoryUsage.heapTotal) * 100;

    return {
      status: heapUsedPercent < 90 && this.metrics.errorRate < 5 ? 'healthy' : 'degraded',
      timestamp: new Date().toISOString(),
      uptime: Math.floor((Date.now() - this.startTime) / 1000),
      memory: {
        heapUsedPercent: heapUsedPercent.toFixed(2) + '%',
        isHealthy: heapUsedPercent < 90
      },
      errors: {
        count: this.metrics.errors,
        isHealthy: this.metrics.errorRate < 5
      }
    };
  }
}

const performanceMonitor = new PerformanceMonitor();

module.exports = performanceMonitor;
