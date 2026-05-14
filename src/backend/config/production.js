module.exports = {
  production: {
    trustProxy: true,
    enableCluster: process.env.ENABLE_CLUSTER === 'true',
    clusterWorkers: parseInt(process.env.CLUSTER_WORKERS) || require('os').cpus().length,
    enableCompression: true,
    compressionLevel: parseInt(process.env.COMPRESSION_LEVEL) || 6,
    compressionThreshold: 1024
  },

  database: {
    pool: {
      min: parseInt(process.env.DB_POOL_MIN) || 2,
      max: parseInt(process.env.DB_POOL_MAX) || 10,
      idleTimeoutMillis: parseInt(process.env.DB_IDLE_TIMEOUT) || 30000,
      connectionTimeoutMillis: parseInt(process.env.DB_CONNECTION_TIMEOUT) || 2000
    },
    ssl: process.env.DB_SSL === 'true',
    sslMode: process.env.DB_SSL_MODE || 'require',
    statementTimeout: parseInt(process.env.DB_STATEMENT_TIMEOUT) || 30000
  },

  redis: {
    maxRetriesPerRequest: 3,
    enableReadyCheck: true,
    enableOfflineQueue: false,
    connectTimeout: 10000,
    commandTimeout: 5000,
    retryDelayOnFailover: 100,
    maxRetriesPerRequest: 3,
    lazyConnect: process.env.NODE_ENV === 'production',
    tls: process.env.REDIS_TLS === 'true' ? {} : undefined
  },

  cache: {
    enabled: process.env.CACHE_ENABLED !== 'false',
    defaultTTL: parseInt(process.env.CACHE_DEFAULT_TTL) || 3600,
    checkPeriod: parseInt(process.env.CACHE_CHECK_PERIOD) || 60,
    maxMemory: process.env.CACHE_MAX_MEMORY || '256mb'
  },

  security: {
    enableHelmet: process.env.ENABLE_HELMET !== 'false',
    enableCsrf: process.env.ENABLE_CSRF === 'true',
    enableRateLimit: process.env.ENABLE_RATE_LIMIT !== 'false',
    enableCors: process.env.ENABLE_CORS !== 'false',
    enableHSTS: process.env.ENABLE_HSTS !== 'false',
    hstsMaxAge: 31536000,
    enableCSP: process.env.ENABLE_CSP === 'true',
    enableXSSFilter: true,
    enableNoSniff: true,
    enableFrameguard: process.env.NODE_ENV === 'production'
  },

  rateLimit: {
    windowMs: parseInt(process.env.RATE_LIMIT_WINDOW_MS) || 15 * 60 * 1000,
    maxRequests: parseInt(process.env.RATE_LIMIT_MAX_REQUESTS) || 100,
    standardHeaders: true,
    legacyHeaders: false,
    skipSuccessfulRequests: false,
    skipFailedRequests: false,
    message: {
      success: false,
      error: 'Too many requests, please try again later',
      code: 'RATE_LIMIT_EXCEEDED'
    }
  },

  performance: {
    requestTimeout: parseInt(process.env.REQUEST_TIMEOUT) || 30000,
    keepAliveTimeout: parseInt(process.env.KEEP_ALIVE_TIMEOUT) || 65000,
    maxConnections: parseInt(process.env.MAX_CONNECTIONS) || 10000,
    enableETag: true,
    staticCacheMaxAge: parseInt(process.env.STATIC_CACHE_MAX_AGE) || 86400000
  },

  monitoring: {
    enableMetrics: process.env.ENABLE_METRICS === 'true',
    enableHealthCheck: true,
    enableApm: process.env.ENABLE_APM === 'true',
    apmServiceName: process.env.APM_SERVICE_NAME || 'hjtpx-api',
    apmLogLevel: process.env.APM_LOG_LEVEL || 'info'
  },

  logging: {
    level: process.env.LOG_LEVEL || 'info',
    maxFiles: process.env.LOG_MAX_FILES || '30d',
    maxSize: process.env.LOG_MAX_SIZE || '20m',
    enableFile: true,
    enableConsole: process.env.NODE_ENV !== 'production',
    enableJsonFormat: process.env.NODE_ENV === 'production',
    enableRotation: true,
    enableCompression: true,
    zippedArchive: true
  }
};
