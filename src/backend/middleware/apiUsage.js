const fs = require('fs');
const path = require('path');

class ApiUsageTracker {
  constructor(storagePath = './docs/api-usage') {
    this.storagePath = storagePath;
    this.statsFile = path.join(this.storagePath, 'usage.json');
    this.dailyDir = path.join(this.storagePath, 'daily');
    this.hourlyDir = path.join(this.storagePath, 'hourly');
    
    this.ensureDirectories();
    this.stats = this.loadStats();
    this.hourlyStats = this.loadHourlyStats();
    this.dailyStats = this.loadDailyStats();
  }

  ensureDirectories() {
    [this.storagePath, this.dailyDir, this.hourlyDir].forEach(dir => {
      if (!fs.existsSync(dir)) {
        fs.mkdirSync(dir, { recursive: true });
      }
    });
  }

  loadStats() {
    if (fs.existsSync(this.statsFile)) {
      try {
        return JSON.parse(fs.readFileSync(this.statsFile, 'utf-8'));
      } catch {
        return this.createEmptyStats();
      }
    }
    return this.createEmptyStats();
  }

  createEmptyStats() {
    return {
      totalRequests: 0,
      totalErrors: 0,
      totalResponseTime: 0,
      endpoints: {},
      methods: {},
      statusCodes: {},
      startTime: new Date().toISOString(),
      lastUpdated: new Date().toISOString()
    };
  }

  loadHourlyStats() {
    const today = new Date().toISOString().split('T')[0];
    const hourlyFile = path.join(this.hourlyDir, `${today}.json`);
    
    if (fs.existsSync(hourlyFile)) {
      try {
        return JSON.parse(fs.readFileSync(hourlyFile, 'utf-8'));
      } catch {
        return this.createEmptyPeriodStats();
      }
    }
    return this.createEmptyPeriodStats();
  }

  loadDailyStats() {
    const statsDir = path.join(this.storagePath, 'daily');
    const files = fs.readdirSync(statsDir).filter(f => f.endsWith('.json'));
    
    const dailyStats = {};
    files.forEach(file => {
      try {
        const data = JSON.parse(fs.readFileSync(path.join(statsDir, file), 'utf-8'));
        dailyStats[file.replace('.json', '')] = data;
      } catch {
        // Skip invalid files
      }
    });
    
    return dailyStats;
  }

  createEmptyPeriodStats() {
    return {
      requests: 0,
      errors: 0,
      responseTime: 0,
      endpoints: {},
      startHour: new Date().toISOString()
    };
  }

  saveStats() {
    this.stats.lastUpdated = new Date().toISOString();
    fs.writeFileSync(this.statsFile, JSON.stringify(this.stats, null, 2));
  }

  saveHourlyStats() {
    const today = new Date().toISOString().split('T')[0];
    const hourlyFile = path.join(this.hourlyDir, `${today}.json`);
    fs.writeFileSync(hourlyFile, JSON.stringify(this.hourlyStats, null, 2));
  }

  recordRequest(req, res, responseTime) {
    const now = new Date();
    const today = new Date();
    const method = req.method.toUpperCase();
    const pathKey = this.normalizePath(req.path);
    const statusCode = res.statusCode;
    const hour = `${today.toISOString().split('T')[0]}-${String(now.getHours()).padStart(2, '0')}`;

    this.stats.totalRequests++;
    this.stats.totalResponseTime += responseTime;
    this.hourlyStats.requests++;
    this.hourlyStats.responseTime += responseTime;

    if (statusCode >= 400) {
      this.stats.totalErrors++;
      this.hourlyStats.errors++;
      this.stats.statusCodes[statusCode] = (this.stats.statusCodes[statusCode] || 0) + 1;
    }

    if (!this.stats.methods[method]) {
      this.stats.methods[method] = { count: 0, errors: 0 };
    }
    this.stats.methods[method].count++;
    if (statusCode >= 400) {
      this.stats.methods[method].errors++;
    }

    if (!this.stats.endpoints[pathKey]) {
      this.stats.endpoints[pathKey] = {
        method,
        path: req.path,
        count: 0,
        errors: 0,
        totalResponseTime: 0,
        lastCalled: null,
        statusCodes: {}
      };
    }
    this.stats.endpoints[pathKey].count++;
    this.stats.endpoints[pathKey].totalResponseTime += responseTime;
    this.stats.endpoints[pathKey].lastCalled = now.toISOString();
    this.stats.endpoints[pathKey].statusCodes[statusCode] = 
      (this.stats.endpoints[pathKey].statusCodes[statusCode] || 0) + 1;
    
    if (statusCode >= 400) {
      this.stats.endpoints[pathKey].errors++;
    }

    if (!this.hourlyStats.endpoints[pathKey]) {
      this.hourlyStats.endpoints[pathKey] = { count: 0, errors: 0 };
    }
    this.hourlyStats.endpoints[pathKey].count++;
    if (statusCode >= 400) {
      this.hourlyStats.endpoints[pathKey].errors++;
    }

    this.saveStats();
    this.saveHourlyStats();
  }

  normalizePath(path) {
    return path
      .replace(/\/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/gi, '/:id')
      .replace(/\/\d+/g, '/:id');
  }

  getOverallStats() {
    return {
      summary: {
        totalRequests: this.stats.totalRequests,
        totalErrors: this.stats.totalErrors,
        errorRate: this.stats.totalRequests > 0 
          ? ((this.stats.totalErrors / this.stats.totalRequests) * 100).toFixed(2) + '%' 
          : '0%',
        averageResponseTime: this.stats.totalRequests > 0 
          ? (this.stats.totalResponseTime / this.stats.totalRequests).toFixed(2) + 'ms' 
          : '0ms',
        startTime: this.stats.startTime,
        lastUpdated: this.stats.lastUpdated,
        uniqueEndpoints: Object.keys(this.stats.endpoints).length
      },
      methods: this.stats.methods,
      statusCodes: this.stats.statusCodes
    };
  }

  getTopEndpoints(limit = 10, sortBy = 'count') {
    return Object.values(this.stats.endpoints)
      .map(ep => ({
        ...ep,
        avgResponseTime: ep.count > 0 ? (ep.totalResponseTime / ep.count).toFixed(2) : 0,
        errorRate: ep.count > 0 ? ((ep.errors / ep.count) * 100).toFixed(2) + '%' : '0%'
      }))
      .sort((a, b) => {
        if (sortBy === 'count') return b.count - a.count;
        if (sortBy === 'responseTime') return b.avgResponseTime - a.avgResponseTime;
        if (sortBy === 'errors') return b.errors - a.errors;
        return 0;
      })
      .slice(0, limit);
  }

  getSlowEndpoints(limit = 10, threshold = 1000) {
    return Object.values(this.stats.endpoints)
      .filter(ep => ep.count > 0 && (ep.totalResponseTime / ep.count) > threshold)
      .map(ep => ({
        ...ep,
        avgResponseTime: (ep.totalResponseTime / ep.count).toFixed(2)
      }))
      .sort((a, b) => b.avgResponseTime - a.avgResponseTime)
      .slice(0, limit);
  }

  getHourlyStats() {
    return this.hourlyStats;
  }

  getDailyStats(date = null) {
    const targetDate = date || new Date().toISOString().split('T')[0];
    return this.dailyStats[targetDate] || null;
  }

  generateUsageReport() {
    const topEndpoints = this.getTopEndpoints(20);
    const slowEndpoints = this.getSlowEndpoints(10);
    const overall = this.getOverallStats();

    let report = `# API Usage Report\n\n`;
    report += `**Generated:** ${new Date().toISOString()}\n\n`;
    
    report += `## Summary\n\n`;
    report += `| Metric | Value |\n`;
    report += `|--------|-------|\n`;
    report += `| Total Requests | ${overall.summary.totalRequests.toLocaleString()} |\n`;
    report += `| Total Errors | ${overall.summary.totalErrors.toLocaleString()} |\n`;
    report += `| Error Rate | ${overall.summary.errorRate} |\n`;
    report += `| Avg Response Time | ${overall.summary.averageResponseTime} |\n`;
    report += `| Unique Endpoints | ${overall.summary.uniqueEndpoints} |\n`;
    report += `| Period Start | ${overall.summary.startTime} |\n`;
    report += '\n';

    report += `## Top 20 Endpoints\n\n`;
    report += `| Endpoint | Method | Calls | Errors | Error Rate | Avg Response Time |\n`;
    report += `|----------|--------|-------|--------|------------|-------------------|\n`;
    topEndpoints.forEach(ep => {
      report += `| ${ep.path} | ${ep.method} | ${ep.count.toLocaleString()} | ${ep.errors} | ${ep.errorRate} | ${ep.avgResponseTime}ms |\n`;
    });
    report += '\n';

    if (slowEndpoints.length > 0) {
      report += `## Slow Endpoints (>1000ms avg)\n\n`;
      report += `| Endpoint | Method | Calls | Avg Response Time |\n`;
      report += `|----------|--------|-------|-------------------|\n`;
      slowEndpoints.forEach(ep => {
        report += `| ${ep.path} | ${ep.method} | ${ep.count.toLocaleString()} | ${ep.avgResponseTime}ms |\n`;
      });
      report += '\n';
    }

    report += `## Request Methods\n\n`;
    report += `| Method | Count | Errors |\n`;
    report += `|--------|-------|--------|\n`;
    Object.entries(overall.methods).forEach(([method, data]) => {
      report += `| ${method} | ${data.count.toLocaleString()} | ${data.errors} |\n`;
    });
    report += '\n';

    report += `## Status Codes\n\n`;
    report += `| Code | Count |\n`;
    report += `|------|-------|\n`;
    Object.entries(overall.statusCodes)
      .sort((a, b) => b[1] - a[1])
      .forEach(([code, count]) => {
        report += `| ${code} | ${count.toLocaleString()} |\n`;
      });

    return report;
  }

  exportStats(format = 'json') {
    const stats = this.getOverallStats();
    
    if (format === 'json') {
      return JSON.stringify(stats, null, 2);
    } else if (format === 'markdown') {
      return this.generateUsageReport();
    }
    
    return stats;
  }

  clearStats() {
    this.stats = this.createEmptyStats();
    this.saveStats();
    console.log('✅ API usage stats cleared');
  }
}

const apiUsageTracker = new ApiUsageTracker();

const apiUsageMiddleware = (req, res, next) => {
  const startTime = Date.now();
  
  const excludePaths = [
    '/api-docs',
    '/docs',
    '/swagger',
    '/static',
    '/favicon.ico'
  ];
  
  const shouldTrack = !excludePaths.some(p => req.path.startsWith(p));
  
  if (shouldTrack) {
    const originalEnd = res.end;
    res.end = function (...args) {
      const responseTime = Date.now() - startTime;
      apiUsageTracker.recordRequest(req, res, responseTime);
      originalEnd.apply(res, args);
    };
  }
  
  next();
};

module.exports = {
  apiUsageMiddleware,
  ApiUsageTracker,
  getApiUsageTracker: () => apiUsageTracker
};
