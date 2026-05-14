module.exports = {
  logging: {
    level: process.env.LOG_LEVEL || 'info',
    format: process.env.LOG_FORMAT || 'json',
    enableConsole: process.env.LOG_CONSOLE !== 'false',
    enableFile: process.env.LOG_FILE === 'true',
    enableRemote: process.env.LOG_REMOTE === 'true',
    maxFileSize: parseInt(process.env.LOG_MAX_SIZE) || 5242880,
    maxFiles: parseInt(process.env.LOG_MAX_FILES) || 5
  },

  monitoring: {
    enabled: process.env.MONITORING_ENABLED === 'true',
    port: parseInt(process.env.MONITORING_PORT) || 9090,
    endpoint: process.env.MONITORING_ENDPOINT || '/metrics',
    collectDefaultMetrics: process.env.COLLECT_DEFAULT_METRICS !== 'false',
    prefix: process.env.METRICS_PREFIX || 'hjtpx_'
  },

  alerts: {
    enabled: process.env.ALERTS_ENABLED === 'true',
    slowRequestThreshold: parseInt(process.env.SLOW_REQUEST_THRESHOLD) || 3000,
    errorRateThreshold: parseFloat(process.env.ERROR_RATE_THRESHOLD) || 0.05,
    cpuThreshold: parseFloat(process.env.CPU_THRESHOLD) || 0.8,
    memoryThreshold: parseFloat(process.env.MEMORY_THRESHOLD) || 0.85
  },

  healthCheck: {
    detailed: process.env.HEALTH_CHECK_DETAILED === 'true',
    includeDatabase: process.env.HEALTH_CHECK_DB !== 'false',
    includeRedis: process.env.HEALTH_CHECK_REDIS !== 'false',
    timeout: parseInt(process.env.HEALTH_CHECK_TIMEOUT) || 5000
  }
};
