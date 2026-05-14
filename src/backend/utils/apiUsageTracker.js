const fs = require('fs').promises;
const path = require('path');

class APIUsageTracker {
  constructor(options = {}) {
    this.statsDir = options.statsDir || path.join(__dirname, '../../docs/api/stats');
    this.dailyStatsFile = options.dailyStatsFile || path.join(this.statsDir, 'daily-stats.json');
    this.hourlyStatsFile = options.hourlyStatsFile || path.join(this.statsDir, 'hourly-stats.json');
    this.endpointStatsFile = options.endpointStatsFile || path.join(this.statsDir, 'endpoint-stats.json');
    this.stats = {
      totalRequests: 0,
      endpointStats: {},
      statusCodes: {},
      responseTimes: [],
      errors: [],
      lastUpdated: new Date().toISOString()
    };
  }

  async initialize() {
    try {
      await fs.mkdir(this.statsDir, { recursive: true });
      await this.loadStats();
    } catch (error) {
      console.error('Failed to initialize API usage tracker:', error);
    }
  }

  async loadStats() {
    try {
      const data = await fs.readFile(path.join(this.statsDir, 'stats.json'), 'utf8');
      this.stats = JSON.parse(data);
    } catch (error) {
      this.stats = {
        totalRequests: 0,
        endpointStats: {},
        statusCodes: {},
        responseTimes: [],
        errors: [],
        lastUpdated: new Date().toISOString()
      };
    }
  }

  async saveStats() {
    this.stats.lastUpdated = new Date().toISOString();
    await fs.writeFile(
      path.join(this.statsDir, 'stats.json'),
      JSON.stringify(this.stats, null, 2),
      'utf8'
    );
  }

  trackRequest(req, res, duration) {
    const endpoint = `${req.method} ${req.path}`;
    
    this.stats.totalRequests++;

    if (!this.stats.endpointStats[endpoint]) {
      this.stats.endpointStats[endpoint] = {
        count: 0,
        avgResponseTime: 0,
        minResponseTime: Infinity,
        maxResponseTime: 0,
        lastCalled: null,
        statusCodes: {}
      };
    }

    const endpointStats = this.stats.endpointStats[endpoint];
    endpointStats.count++;
    endpointStats.lastCalled = new Date().toISOString();
    
    endpointStats.minResponseTime = Math.min(endpointStats.minResponseTime, duration);
    endpointStats.maxResponseTime = Math.max(endpointStats.maxResponseTime, duration);
    endpointStats.avgResponseTime = 
      (endpointStats.avgResponseTime * (endpointStats.count - 1) + duration) / endpointStats.count;

    const statusCode = res.statusCode.toString();
    if (!endpointStats.statusCodes[statusCode]) {
      endpointStats.statusCodes[statusCode] = 0;
    }
    endpointStats.statusCodes[statusCode]++;

    if (!this.stats.statusCodes[statusCode]) {
      this.stats.statusCodes[statusCode] = 0;
    }
    this.stats.statusCodes[statusCode]++;

    this.stats.responseTimes.push({
      endpoint,
      duration,
      timestamp: new Date().toISOString()
    });

    if (this.stats.responseTimes.length > 10000) {
      this.stats.responseTimes = this.stats.responseTimes.slice(-5000);
    }

    if (res.statusCode >= 400) {
      this.stats.errors.push({
        endpoint,
        statusCode: res.statusCode,
        timestamp: new Date().toISOString(),
        method: req.method
      });

      if (this.stats.errors.length > 1000) {
        this.stats.errors = this.stats.errors.slice(-500);
      }
    }

    this.saveStats();
  }

  getStats() {
    return {
      ...this.stats,
      averageResponseTime: this.calculateAverageResponseTime(),
      requestsPerMinute: this.calculateRequestsPerMinute(),
      errorRate: this.calculateErrorRate(),
      topEndpoints: this.getTopEndpoints(10)
    };
  }

  calculateAverageResponseTime() {
    if (this.stats.responseTimes.length === 0) return 0;
    
    const total = this.stats.responseTimes.reduce((sum, rt) => sum + rt.duration, 0);
    return total / this.stats.responseTimes.length;
  }

  calculateRequestsPerMinute() {
    const oneMinuteAgo = new Date(Date.now() - 60000).toISOString();
    const recentRequests = this.stats.responseTimes.filter(rt => rt.timestamp > oneMinuteAgo);
    return recentRequests.length;
  }

  calculateErrorRate() {
    const total = this.stats.totalRequests;
    const errors = Object.entries(this.stats.statusCodes)
      .filter(([code]) => parseInt(code) >= 400)
      .reduce((sum, [, count]) => sum + count, 0);
    
    return total > 0 ? (errors / total) * 100 : 0;
  }

  getTopEndpoints(limit = 10) {
    return Object.entries(this.stats.endpointStats)
      .sort((a, b) => b[1].count - a[1].count)
      .slice(0, limit)
      .map(([endpoint, stats]) => ({
        endpoint,
        count: stats.count,
        avgResponseTime: stats.avgResponseTime
      }));
  }

  async generateDailyReport() {
    const now = new Date();
    const startOfDay = new Date(now.getFullYear(), now.getMonth(), now.getDate()).toISOString();
    
    const dailyRequests = this.stats.responseTimes.filter(rt => rt.timestamp >= startOfDay);
    
    const report = {
      date: startOfDay.split('T')[0],
      totalRequests: dailyRequests.length,
      uniqueEndpoints: new Set(dailyRequests.map(rt => rt.endpoint)).size,
      averageResponseTime: dailyRequests.length > 0
        ? dailyRequests.reduce((sum, rt) => sum + rt.duration, 0) / dailyRequests.length
        : 0,
      errorCount: dailyRequests.filter(rt => rt.duration > 5000).length,
      topEndpoints: this.getTopEndpoints(5),
      statusCodeDistribution: this.getStatusCodeDistribution(dailyRequests)
    };

    await this.appendToDailyStats(report);
    
    return report;
  }

  getStatusCodeDistribution(requests) {
    const distribution = {};
    requests.forEach(rt => {
      const code = rt.statusCode || 200;
      distribution[code] = (distribution[code] || 0) + 1;
    });
    return distribution;
  }

  async appendToDailyStats(report) {
    try {
      let dailyStats = [];
      try {
        const data = await fs.readFile(this.dailyStatsFile, 'utf8');
        dailyStats = JSON.parse(data);
      } catch (error) {}

      dailyStats.unshift(report);
      if (dailyStats.length > 365) {
        dailyStats = dailyStats.slice(0, 365);
      }

      await fs.writeFile(this.dailyStatsFile, JSON.stringify(dailyStats, null, 2), 'utf8');
    } catch (error) {
      console.error('Failed to save daily stats:', error);
    }
  }

  middleware() {
    return (req, res, next) => {
      const startTime = Date.now();

      res.on('finish', () => {
        const duration = Date.now() - startTime;
        this.trackRequest(req, res, duration);
      });

      next();
    };
  }

  getDocumentationStats() {
    return {
      totalEndpoints: Object.keys(this.stats.endpointStats).length,
      totalRequests: this.stats.totalRequests,
      averageResponseTime: this.calculateAverageResponseTime(),
      errorRate: this.calculateErrorRate(),
      documentationCoverage: this.calculateDocumentationCoverage()
    };
  }

  calculateDocumentationCoverage() {
    return {
      documented: Object.keys(this.stats.endpointStats).length,
      used: Object.keys(this.stats.endpointStats).filter(ep => this.stats.endpointStats[ep].count > 0).length,
      percentage: Object.keys(this.stats.endpointStats).length > 0
        ? (Object.keys(this.stats.endpointStats).filter(ep => this.stats.endpointStats[ep].count > 0).length / Object.keys(this.stats.endpointStats).length) * 100
        : 0
    };
  }
}

module.exports = new APIUsageTracker();
