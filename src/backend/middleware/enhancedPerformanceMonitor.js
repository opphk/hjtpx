const os = require('os');

const {
  recordDatabaseQuery,
  recordRedisOperation,
  updateSystemMetrics
} = require('../services/metricsService');

const SLOW_REQUEST_THRESHOLD = parseInt(process.env.SLOW_REQUEST_THRESHOLD) || 1000;

const performanceMiddleware = (req, res, next) => {
  const startTime = process.hrtime.bigint();
  const requestId = req.requestId || `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;

  req.performanceStartTime = startTime;
  req.requestId = requestId;
  res.setHeader('X-Request-ID', requestId);

  const originalSend = res.send;
  const originalJson = res.json;

  res.send = function (data) {
    res.performanceData = res.performanceData || {};
    res.performanceData.endTime = process.hrtime.bigint();
    res.performanceData.duration = Number(res.performanceData.endTime - startTime) / 1e6;
    res.performanceData.statusCode = res.statusCode;

    if (res.performanceData.duration > SLOW_REQUEST_THRESHOLD) {
      console.warn(
        `⚠️ 慢请求检测: ${req.method} ${req.path} - ${res.performanceData.duration.toFixed(2)}ms`
      );

      if (process.env.SENTRY_DSN) {
        const { Sentry } = require('../config/sentry');
        Sentry.addBreadcrumb({
          category: 'performance',
          message: `慢请求: ${req.method} ${req.path}`,
          level: 'warning',
          data: {
            duration: res.performanceData.duration,
            threshold: SLOW_REQUEST_THRESHOLD,
            method: req.method,
            path: req.path,
            statusCode: res.statusCode
          }
        });
      }
    }

    updateSystemMetrics();

    return originalSend.call(this, data);
  };

  res.json = function (data) {
    res.performanceData = res.performanceData || {};
    res.performanceData.endTime = process.hrtime.bigint();
    res.performanceData.duration = Number(res.performanceData.endTime - startTime) / 1e6;
    res.performanceData.statusCode = res.statusCode;

    if (res.performanceData.duration > SLOW_REQUEST_THRESHOLD) {
      console.warn(
        `⚠️ 慢请求检测: ${req.method} ${req.path} - ${res.performanceData.duration.toFixed(2)}ms`
      );
    }

    updateSystemMetrics();

    return originalJson.call(this, data);
  };

  res.on('finish', () => {
    if (!res.performanceData) {
      res.performanceData = {
        endTime: process.hrtime.bigint(),
        duration: Number(process.hrtime.bigint() - startTime) / 1e6,
        statusCode: res.statusCode
      };
    }

    const duration = res.performanceData.duration;

    if (duration > SLOW_REQUEST_THRESHOLD) {
      console.warn(`慢请求: ${req.method} ${req.path} ${duration.toFixed(2)}ms ${res.statusCode}`);
    }
  });

  next();
};

class DatabaseQueryMonitor {
  constructor() {
    this.queryTimings = new Map();
    this.slowQueryThreshold = parseInt(process.env.SLOW_QUERY_THRESHOLD) || 100;
  }

  startQueryMonitor(queryId, queryType, table) {
    const startTime = process.hrtime.bigint();
    this.queryTimings.set(queryId, {
      queryType,
      table,
      startTime,
      startMemory: process.memoryUsage().heapUsed
    });
    return startTime;
  }

  endQueryMonitor(queryId, queryType, table, error = null) {
    const timing = this.queryTimings.get(queryId);
    if (!timing) return;

    const endTime = process.hrtime.bigint();
    const duration = Number(endTime - timing.startTime) / 1e6;
    const endMemory = process.memoryUsage().heapUsed;
    const memoryDelta = endMemory - timing.startMemory;

    recordDatabaseQuery(queryType, table, duration / 1000, error);

    if (duration > this.slowQueryThreshold) {
      console.warn(`慢查询检测: ${queryType} ${table} - ${duration.toFixed(2)}ms`);
    }

    this.queryTimings.delete(queryId);

    return {
      duration,
      memoryDelta,
      error
    };
  }

  getQueryStats() {
    return {
      activeQueries: this.queryTimings.size,
      slowQueryThreshold: this.slowQueryThreshold
    };
  }
}

class RedisOperationMonitor {
  constructor() {
    this.operationTimings = new Map();
    this.slowOperationThreshold = parseInt(process.env.SLOW_REDIS_THRESHOLD) || 50;
  }

  startOperationMonitor(operationId, operation) {
    const startTime = process.hrtime.bigint();
    this.operationTimings.set(operationId, {
      operation,
      startTime
    });
    return startTime;
  }

  endOperationMonitor(operationId, operation, error = null) {
    const timing = this.operationTimings.get(operationId);
    if (!timing) return;

    const endTime = process.hrtime.bigint();
    const duration = Number(endTime - timing.startTime) / 1e6;
    const status = error ? 'error' : 'success';

    recordRedisOperation(operation, status, duration / 1000);

    if (duration > this.slowOperationThreshold) {
      console.warn(`慢Redis操作: ${operation} - ${duration.toFixed(2)}ms`);
    }

    this.operationTimings.delete(operationId);

    return {
      duration,
      error
    };
  }

  getOperationStats() {
    return {
      activeOperations: this.operationTimings.size,
      slowOperationThreshold: this.slowOperationThreshold
    };
  }
}

const dbMonitor = new DatabaseQueryMonitor();
const redisMonitor = new RedisOperationMonitor();

function monitorDatabaseQuery(queryType, table) {
  const queryId = `query_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  dbMonitor.startQueryMonitor(queryId, queryType, table);

  return {
    queryId,
    end: (error = null) => dbMonitor.endQueryMonitor(queryId, queryType, table, error)
  };
}

function monitorRedisOperation(operation) {
  const operationId = `redis_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  redisMonitor.startOperationMonitor(operationId, operation);

  return {
    operationId,
    end: (error = null) => redisMonitor.endOperationMonitor(operationId, operation, error)
  };
}

function getSystemMetrics() {
  const cpuLoad = os.loadavg();
  const totalMemory = os.totalmem();
  const freeMemory = os.freemem();
  const usedMemory = totalMemory - freeMemory;
  const memoryUsagePercent = (usedMemory / totalMemory) * 100;

  const cpus = os.cpus();
  let totalIdle = 0;
  let totalTick = 0;

  cpus.forEach(cpu => {
    for (const type in cpu.times) {
      totalTick += cpu.times[type];
    }
    totalIdle += cpu.times.idle;
  });

  const cpuUsagePercent = 100 - (100 * totalIdle) / totalTick;

  const processMemory = process.memoryUsage();
  const heapUsedPercent = (processMemory.heapUsed / processMemory.heapTotal) * 100;

  return {
    system: {
      cpu: {
        usage: cpuUsagePercent.toFixed(2),
        loadAverage: {
          '1min': cpuLoad[0].toFixed(2),
          '5min': cpuLoad[1].toFixed(2),
          '15min': cpuLoad[2].toFixed(2)
        },
        cores: cpus.length
      },
      memory: {
        total: formatBytes(totalMemory),
        used: formatBytes(usedMemory),
        free: formatBytes(freeMemory),
        usagePercent: memoryUsagePercent.toFixed(2)
      }
    },
    process: {
      memory: {
        heapUsed: formatBytes(processMemory.heapUsed),
        heapTotal: formatBytes(processMemory.heapTotal),
        heapUsagePercent: heapUsedPercent.toFixed(2),
        rss: formatBytes(processMemory.rss),
        external: formatBytes(processMemory.external)
      },
      uptime: process.uptime(),
      pid: process.pid
    },
    monitoring: {
      database: dbMonitor.getQueryStats(),
      redis: redisMonitor.getOperationStats()
    }
  };
}

function formatBytes(bytes) {
  if (bytes === 0) return '0 Bytes';
  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

module.exports = {
  performanceMiddleware,
  monitorDatabaseQuery,
  monitorRedisOperation,
  getSystemMetrics,
  dbMonitor,
  redisMonitor
};
