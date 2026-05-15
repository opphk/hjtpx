const os = require('os');

class ConnectionPoolOptimizer {
  constructor() {
    this.cpuCores = os.cpus().length;
    this.totalMemory = os.totalmem();
    this.freeMemory = os.freemem();
    this.isProduction = process.env.NODE_ENV === 'production';
    this.environment = this._detectEnvironment();
  }

  _detectEnvironment() {
    if (process.env.NODE_ENV === 'production') {
      return 'production';
    } else if (process.env.NODE_ENV === 'staging') {
      return 'staging';
    } else if (process.env.NODE_ENV === 'test') {
      return 'test';
    }
    return 'development';
  }

  getOptimalPoolSize() {
    const cpuBasedSize = Math.ceil(this.cpuCores * this._getMultiplier());
    const memoryBasedSize = this._calculateMemoryBasedSize();
    const recommendedSize = Math.min(cpuBasedSize, memoryBasedSize);

    return {
      min: this._getMinPoolSize(recommendedSize),
      max: this._getMaxPoolSize(recommendedSize),
      cpuCores: this.cpuCores,
      cpuBasedSize,
      memoryBasedSize,
      recommendedSize,
      reasoning: {
        cpuMultiplier: this._getMultiplier(),
        memoryBasedOn: `${Math.round(this.totalMemory / (1024 * 1024 * 1024))}GB total RAM`
      }
    };
  }

  _getMultiplier() {
    switch (this.environment) {
      case 'production':
        return 2.5;
      case 'staging':
        return 2;
      case 'test':
        return 0.5;
      default:
        return 1.5;
    }
  }

  _calculateMemoryBasedSize() {
    const availableMemoryGB = this.freeMemory / (1024 * 1024 * 1024);
    const connectionMemoryEstimateMB = 15;
    const maxConnectionsByMemory = Math.floor((availableMemoryGB * 1024) / connectionMemoryEstimateMB);
    
    return Math.max(5, Math.min(maxConnectionsByMemory, 100));
  }

  _getMinPoolSize(recommendedSize) {
    const envMin = parseInt(process.env.DB_POOL_MIN);
    if (envMin) {
      return Math.min(envMin, this._getMaxPoolSize(recommendedSize));
    }

    switch (this.environment) {
      case 'production':
        return Math.max(5, Math.floor(recommendedSize * 0.2));
      case 'staging':
        return Math.max(2, Math.floor(recommendedSize * 0.15));
      case 'test':
        return 1;
      default:
        return 2;
    }
  }

  _getMaxPoolSize(recommendedSize) {
    const envMax = parseInt(process.env.DB_POOL_MAX);
    if (envMax) {
      return envMax;
    }

    switch (this.environment) {
      case 'production':
        return Math.min(50, recommendedSize);
      case 'staging':
        return Math.min(30, recommendedSize);
      case 'test':
        return 5;
      default:
        return Math.min(20, recommendedSize);
    }
  }

  getOptimalTimeouts() {
    return {
      idleTimeoutMillis: this._getIdleTimeout(),
      connectionTimeoutMillis: this._getConnectionTimeout(),
      statementTimeout: this._getStatementTimeout(),
      keepAliveInitialDelayMillis: this._getKeepAliveDelay(),
      keepAlive: true,
      reasoning: {
        environment: this.environment,
        production: this.isProduction
      }
    };
  }

  _getIdleTimeout() {
    const envTimeout = parseInt(process.env.DB_IDLE_TIMEOUT);
    if (envTimeout) {
      return envTimeout;
    }

    switch (this.environment) {
      case 'production':
        return 60000;
      case 'staging':
        return 45000;
      case 'test':
        return 10000;
      default:
        return 30000;
    }
  }

  _getConnectionTimeout() {
    const envTimeout = parseInt(process.env.DB_CONNECTION_TIMEOUT);
    if (envTimeout) {
      return envTimeout;
    }

    switch (this.environment) {
      case 'production':
        return 10000;
      case 'staging':
        return 7500;
      case 'test':
        return 2000;
      default:
        return 5000;
    }
  }

  _getStatementTimeout() {
    const envTimeout = parseInt(process.env.DB_STATEMENT_TIMEOUT);
    if (envTimeout) {
      return envTimeout;
    }

    switch (this.environment) {
      case 'production':
        return 60000;
      case 'staging':
        return 45000;
      case 'test':
        return 10000;
      default:
        return 30000;
    }
  }

  _getKeepAliveDelay() {
    const envDelay = parseInt(process.env.DB_KEEPALIVE_DELAY);
    if (envDelay) {
      return envDelay;
    }

    return 30000;
  }

  getOptimalHealthCheckConfig() {
    return {
      interval: this._getHealthCheckInterval(),
      maxResponseTime: this._getHealthCheckMaxResponseTime(),
      minIdleConnections: this._getMinIdleConnections(),
      maxConnectionUsage: this._getMaxConnectionUsage(),
      maxConsecutiveFailures: 3
    };
  }

  _getHealthCheckInterval() {
    const envInterval = parseInt(process.env.DB_HEALTH_CHECK_INTERVAL);
    if (envInterval) {
      return envInterval;
    }

    switch (this.environment) {
      case 'production':
        return 30000;
      case 'staging':
        return 45000;
      case 'test':
        return 60000;
      default:
        return 30000;
    }
  }

  _getHealthCheckMaxResponseTime() {
    const envTime = parseInt(process.env.DB_HEALTH_MAX_RESPONSE_TIME);
    if (envTime) {
      return envTime;
    }

    switch (this.environment) {
      case 'production':
        return 5000;
      case 'staging':
        return 7500;
      default:
        return 5000;
    }
  }

  _getMinIdleConnections() {
    const poolConfig = this.getOptimalPoolSize();
    return Math.max(2, Math.floor(poolConfig.min * 0.5));
  }

  _getMaxConnectionUsage() {
    const envUsage = parseFloat(process.env.DB_HEALTH_MAX_USAGE);
    if (envUsage) {
      return Math.min(0.95, Math.max(0.5, envUsage));
    }

    switch (this.environment) {
      case 'production':
        return 0.85;
      case 'staging':
        return 0.75;
      default:
        return 0.80;
    }
  }

  getOptimalLeakDetectionConfig() {
    return {
      threshold: this._getLeakThreshold(),
      checkInterval: this._getLeakCheckInterval(),
      maxRecords: this._getMaxLeakRecords(),
      autoCleanup: this._shouldEnableAutoCleanup(),
      autoCleanupTimeout: this._getAutoCleanupTimeout()
    };
  }

  _getLeakThreshold() {
    const envThreshold = parseInt(process.env.DB_LEAK_THRESHOLD);
    if (envThreshold) {
      return envThreshold;
    }

    switch (this.environment) {
      case 'production':
        return 30000;
      case 'staging':
        return 45000;
      case 'test':
        return 10000;
      default:
        return 30000;
    }
  }

  _getLeakCheckInterval() {
    const envInterval = parseInt(process.env.DB_LEAK_CHECK_INTERVAL);
    if (envInterval) {
      return envInterval;
    }

    return 10000;
  }

  _getMaxLeakRecords() {
    const envRecords = parseInt(process.env.DB_LEAK_MAX_RECORDS);
    if (envRecords) {
      return envRecords;
    }

    switch (this.environment) {
      case 'production':
        return 100;
      case 'staging':
        return 50;
      default:
        return 100;
    }
  }

  _shouldEnableAutoCleanup() {
    if (process.env.DB_LEAK_AUTO_CLEANUP) {
      return process.env.DB_LEAK_AUTO_CLEANUP === 'true';
    }

    return this.environment === 'production';
  }

  _getAutoCleanupTimeout() {
    const envTimeout = parseInt(process.env.DB_LEAK_AUTO_CLEANUP_TIMEOUT);
    if (envTimeout) {
      return envTimeout;
    }

    switch (this.environment) {
      case 'production':
        return 60000;
      case 'staging':
        return 90000;
      default:
        return 60000;
    }
  }

  getReapingConfig() {
    return {
      interval: this._getReapingInterval(),
      excessIdleThreshold: this._getExcessIdleThreshold()
    };
  }

  _getReapingInterval() {
    const envInterval = parseInt(process.env.DB_REAPING_INTERVAL);
    if (envInterval) {
      return envInterval;
    }

    switch (this.environment) {
      case 'production':
        return 60000;
      case 'staging':
        return 90000;
      default:
        return 120000;
    }
  }

  _getExcessIdleThreshold() {
    const poolConfig = this.getOptimalPoolSize();
    return Math.max(3, Math.floor(poolConfig.min * 0.5));
  }

  getSlowQueryThreshold() {
    const envThreshold = parseInt(process.env.SLOW_QUERY_THRESHOLD);
    if (envThreshold) {
      return envThreshold;
    }

    switch (this.environment) {
      case 'production':
        return 1000;
      case 'staging':
        return 500;
      default:
        return 100;
    }
  }

  getCompleteConfiguration() {
    const poolSize = this.getOptimalPoolSize();
    const timeouts = this.getOptimalTimeouts();
    const healthCheck = this.getOptimalHealthCheckConfig();
    const leakDetection = this.getOptimalLeakDetectionConfig();
    const reaping = this.getReapingConfig();

    return {
      environment: this.environment,
      isProduction: this.isProduction,
      systemInfo: {
        cpuCores: this.cpuCores,
        totalMemoryGB: Math.round(this.totalMemory / (1024 * 1024 * 1024) * 100) / 100,
        freeMemoryGB: Math.round(this.freeMemory / (1024 * 1024 * 1024) * 100) / 100
      },
      pool: {
        min: poolSize.min,
        max: poolSize.max,
        reasoning: poolSize.reasoning
      },
      timeouts,
      healthCheck,
      leakDetection,
      reaping,
      slowQueryThreshold: this.getSlowQueryThreshold()
    };
  }

  validateConfiguration(config) {
    const errors = [];
    const warnings = [];

    if (config.pool.min < 0) {
      errors.push('Minimum pool size cannot be negative');
    }

    if (config.pool.max < config.pool.min) {
      errors.push('Maximum pool size must be greater than or equal to minimum pool size');
    }

    if (config.pool.max > 100) {
      warnings.push('Maximum pool size exceeds recommended limit of 100');
    }

    if (config.timeouts.connectionTimeoutMillis > 30000) {
      warnings.push('Connection timeout is very high, consider reducing');
    }

    if (config.timeouts.idleTimeoutMillis < 10000) {
      warnings.push('Idle timeout is very low, connections may be closed too quickly');
    }

    if (config.leakDetection.autoCleanup && !this.isProduction) {
      warnings.push('Auto cleanup is enabled in non-production environment');
    }

    return {
      valid: errors.length === 0,
      errors,
      warnings
    };
  }
}

module.exports = ConnectionPoolOptimizer;
