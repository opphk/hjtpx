const os = require('os');

const { logInfo, logDebug } = require('../utils/productionLogger');
const { alertManager } = require('./alertService');

class HealthMonitor {
  constructor() {
    this.checks = new Map();
    this.lastCheckResults = new Map();
    this.interval = parseInt(process.env.HEALTH_CHECK_INTERVAL) || 60000;
    this.intervalId = null;
    this.metricsHistory = {
      cpu: [],
      memory: [],
      responseTime: [],
      errorRate: []
    };
    this.maxHistorySize = 60;
  }

  registerCheck(name, checkFn) {
    this.checks.set(name, checkFn);
  }

  async runCheck(name) {
    const checkFn = this.checks.get(name);
    if (!checkFn) {
      return { healthy: false, error: 'Check not found' };
    }

    const start = Date.now();
    try {
      const result = await checkFn();
      const duration = Date.now() - start;

      const checkResult = {
        name,
        healthy: result.healthy !== false,
        message: result.message || 'OK',
        duration,
        timestamp: new Date().toISOString(),
        details: result.details || {}
      };

      this.lastCheckResults.set(name, checkResult);
      return checkResult;
    } catch (error) {
      return {
        name,
        healthy: false,
        message: error.message,
        duration: Date.now() - start,
        timestamp: new Date().toISOString(),
        error: error.stack
      };
    }
  }

  async runAllChecks() {
    const results = [];
    for (const name of this.checks.keys()) {
      results.push(await this.runCheck(name));
    }
    return results;
  }

  startPeriodicChecks() {
    if (this.intervalId) {
      return;
    }

    this.intervalId = setInterval(async () => {
      try {
        await this.collectSystemMetrics();
        await this.checkThresholds();
      } catch (error) {
        logDebug('Error during periodic health check', { error: error.message });
      }
    }, this.interval);

    logInfo('Health monitor started', { interval: this.interval });
  }

  stopPeriodicChecks() {
    if (this.intervalId) {
      clearInterval(this.intervalId);
      this.intervalId = null;
      logInfo('Health monitor stopped');
    }
  }

  async collectSystemMetrics() {
    const cpuUsage = this.getCpuUsage();
    const memUsage = this.getMemoryUsage();

    this.addToHistory('cpu', cpuUsage);
    this.addToHistory('memory', memUsage);

    alertManager.checkCpuUsage(cpuUsage);
    alertManager.checkMemoryUsage(memUsage);

    return { cpu: cpuUsage, memory: memUsage };
  }

  getCpuUsage() {
    const cpus = os.cpus();
    let totalIdle = 0;
    let totalTick = 0;

    cpus.forEach(cpu => {
      for (const type in cpu.times) {
        totalTick += cpu.times[type];
      }
      totalIdle += cpu.times.idle;
    });

    const idle = totalIdle / cpus.length;
    const total = totalTick / cpus.length;
    const usage = 1 - idle / total;

    return usage;
  }

  getMemoryUsage() {
    const total = os.totalmem();
    const free = os.freemem();
    const used = total - free;
    return used / total;
  }

  addToHistory(metric, value) {
    this.metricsHistory[metric].push({
      value,
      timestamp: Date.now()
    });

    if (this.metricsHistory[metric].length > this.maxHistorySize) {
      this.metricsHistory[metric].shift();
    }
  }

  async checkThresholds() {
    const checks = await this.runAllChecks();
    const errorCount = checks.filter(c => !c.healthy).length;
    const totalChecks = checks.length;

    if (totalChecks > 0) {
      const errorRate = errorCount / totalChecks;
      alertManager.checkErrorRate(errorCount, totalChecks);
    }
  }

  getHealthStatus() {
    const checks = Array.from(this.lastCheckResults.values());
    const healthy = checks.every(c => c.healthy);

    return {
      healthy,
      status: healthy ? 'healthy' : 'degraded',
      timestamp: new Date().toISOString(),
      uptime: process.uptime(),
      checks: {
        total: checks.length,
        passed: checks.filter(c => c.healthy).length,
        failed: checks.filter(c => !c.healthy).length
      },
      lastCheck: checks.length > 0 ? checks[checks.length - 1].timestamp : null
    };
  }

  getDetailedHealth() {
    const checks = Array.from(this.lastCheckResults.values());
    const healthy = checks.every(c => c.healthy);

    return {
      healthy,
      status: healthy ? 'healthy' : 'degraded',
      timestamp: new Date().toISOString(),
      uptime: process.uptime(),
      version: process.env.APP_VERSION || '1.0.0',
      environment: process.env.NODE_ENV,
      nodeVersion: process.version,
      checks: checks.map(c => ({
        name: c.name,
        healthy: c.healthy,
        message: c.message,
        duration: c.duration,
        details: c.details
      })),
      system: {
        cpu: {
          usage: this.getCpuUsage(),
          cores: os.cpus().length,
          loadAverage: os.loadavg()
        },
        memory: {
          used: os.freemem(),
          total: os.totalmem(),
          usage: this.getMemoryUsage()
        },
        process: {
          memory: process.memoryUsage(),
          cpu: process.cpuUsage()
        }
      },
      metrics: {
        history: {
          cpu: this.metricsHistory.cpu.slice(-10),
          memory: this.metricsHistory.memory.slice(-10)
        }
      },
      alerts: alertManager.getActiveAlerts()
    };
  }

  getMetrics() {
    return {
      health: this.getHealthStatus(),
      system: {
        cpu: this.getCpuUsage(),
        memory: this.getMemoryUsage()
      },
      checks: this.lastCheckResults,
      alerts: alertManager.getMetrics()
    };
  }
}

const healthMonitor = new HealthMonitor();

healthMonitor.registerCheck('application', async () => {
  return { healthy: true, message: 'Application is running' };
});

healthMonitor.registerCheck('memory', async () => {
  const memUsage = healthMonitor.getMemoryUsage();
  const threshold = parseFloat(process.env.ALERT_MEMORY_THRESHOLD) || 0.85;

  if (memUsage > threshold) {
    return {
      healthy: false,
      message: `Memory usage is high: ${(memUsage * 100).toFixed(2)}%`,
      details: { usage: memUsage, threshold }
    };
  }

  return { healthy: true, message: 'Memory usage is normal' };
});

healthMonitor.registerCheck('cpu', async () => {
  const cpuUsage = healthMonitor.getCpuUsage();
  const threshold = parseFloat(process.env.ALERT_CPU_THRESHOLD) || 0.8;

  if (cpuUsage > threshold) {
    return {
      healthy: false,
      message: `CPU usage is high: ${(cpuUsage * 100).toFixed(2)}%`,
      details: { usage: cpuUsage, threshold }
    };
  }

  return { healthy: true, message: 'CPU usage is normal' };
});

module.exports = {
  HealthMonitor,
  healthMonitor
};
