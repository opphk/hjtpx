const fs = require('fs');
const path = require('path');

class ApiUsageStats {
  constructor(storagePath = './docs/stats') {
    this.storagePath = storagePath;
    this.statsFile = path.join(this.storagePath, 'usage-stats.json');
    this.ensureDirExists();
    this.stats = this.loadStats();
  }

  ensureDirExists() {
    if (!fs.existsSync(this.storagePath)) {
      fs.mkdirSync(this.storagePath, { recursive: true });
    }
  }

  loadStats() {
    if (fs.existsSync(this.statsFile)) {
      return JSON.parse(fs.readFileSync(this.statsFile, 'utf-8'));
    }
    return {
      endpoints: {},
      totalRequests: 0,
      totalErrors: 0,
      startTime: new Date().toISOString()
    };
  }

  saveStats() {
    fs.writeFileSync(this.statsFile, JSON.stringify(this.stats, null, 2));
  }

  recordRequest(method, path, statusCode, responseTime, timestamp = new Date()) {
    const endpointKey = `${method.toUpperCase()} ${path}`;
    
    if (!this.stats.endpoints[endpointKey]) {
      this.stats.endpoints[endpointKey] = {
        method: method.toUpperCase(),
        path,
        calls: 0,
        errors: 0,
        totalResponseTime: 0,
        lastCalled: null
      };
    }

    const endpoint = this.stats.endpoints[endpointKey];
    endpoint.calls++;
    endpoint.totalResponseTime += responseTime;
    endpoint.lastCalled = timestamp.toISOString();
    
    if (statusCode >= 400) {
      endpoint.errors++;
      this.stats.totalErrors++;
    }
    
    this.stats.totalRequests++;
    
    this.saveStats();
  }

  getStats() {
    const endpoints = Object.values(this.stats.endpoints).map(ep => ({
      ...ep,
      avgResponseTime: ep.calls > 0 ? ep.totalResponseTime / ep.calls : 0,
      errorRate: ep.calls > 0 ? (ep.errors / ep.calls) * 100 : 0
    }));

    return {
      summary: {
        totalRequests: this.stats.totalRequests,
        totalErrors: this.stats.totalErrors,
        errorRate: this.stats.totalRequests > 0 ? (this.stats.totalErrors / this.stats.totalRequests) * 100 : 0,
        startTime: this.stats.startTime,
        uniqueEndpoints: endpoints.length
      },
      endpoints: endpoints.sort((a, b) => b.calls - a.calls)
    };
  }

  getEndpointStats(method, path) {
    const endpointKey = `${method.toUpperCase()} ${path}`;
    const endpoint = this.stats.endpoints[endpointKey];
    if (!endpoint) {
      return null;
    }
    return {
      ...endpoint,
      avgResponseTime: endpoint.calls > 0 ? endpoint.totalResponseTime / endpoint.calls : 0,
      errorRate: endpoint.calls > 0 ? (endpoint.errors / endpoint.calls) * 100 : 0
    };
  }

  clearStats() {
    this.stats = {
      endpoints: {},
      totalRequests: 0,
      totalErrors: 0,
      startTime: new Date().toISOString()
    };
    this.saveStats();
  }
}

module.exports = ApiUsageStats;
