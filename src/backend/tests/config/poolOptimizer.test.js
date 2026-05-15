const ConnectionPoolOptimizer = require('../../config/database/poolOptimizer');

describe('ConnectionPoolOptimizer', () => {
  let optimizer;

  beforeEach(() => {
    optimizer = new ConnectionPoolOptimizer();
  });

  describe('Initialization', () => {
    test('should detect environment correctly', () => {
      expect(optimizer.environment).toBeDefined();
      expect(['production', 'staging', 'test', 'development']).toContain(optimizer.environment);
    });

    test('should detect production environment', () => {
      process.env.NODE_ENV = 'production';
      const prodOptimizer = new ConnectionPoolOptimizer();
      expect(prodOptimizer.isProduction).toBe(true);
      expect(prodOptimizer.environment).toBe('production');
    });

    test('should detect development environment', () => {
      process.env.NODE_ENV = 'development';
      const devOptimizer = new ConnectionPoolOptimizer();
      expect(devOptimizer.isProduction).toBe(false);
    });

    test('should get system information', () => {
      expect(optimizer.cpuCores).toBeGreaterThan(0);
      expect(optimizer.totalMemory).toBeGreaterThan(0);
      expect(optimizer.freeMemory).toBeGreaterThan(0);
    });
  });

  describe('Pool Size Optimization', () => {
    test('should calculate optimal pool size', () => {
      const poolSize = optimizer.getOptimalPoolSize();

      expect(poolSize).toHaveProperty('min');
      expect(poolSize).toHaveProperty('max');
      expect(poolSize).toHaveProperty('cpuCores');
      expect(poolSize).toHaveProperty('cpuBasedSize');
      expect(poolSize).toHaveProperty('memoryBasedSize');
      expect(poolSize).toHaveProperty('recommendedSize');
    });

    test('should have min less than or equal to max', () => {
      const poolSize = optimizer.getOptimalPoolSize();
      expect(poolSize.min).toBeLessThanOrEqual(poolSize.max);
    });

    test('should respect environment minimum pool size', () => {
      process.env.DB_POOL_MIN = '10';
      const opt = new ConnectionPoolOptimizer();
      const poolSize = opt.getOptimalPoolSize();
      
      expect(poolSize.min).toBe(10);
    });

    test('should respect environment maximum pool size', () => {
      process.env.DB_POOL_MAX = '30';
      const opt = new ConnectionPoolOptimizer();
      const poolSize = opt.getOptimalPoolSize();
      
      expect(poolSize.max).toBe(30);
    });

    test('should calculate memory-based size correctly', () => {
      const memoryBasedSize = optimizer._calculateMemoryBasedSize();
      
      expect(memoryBasedSize).toBeGreaterThan(0);
      expect(memoryBasedSize).toBeLessThanOrEqual(100);
    });

    test('should get correct multiplier for production', () => {
      process.env.NODE_ENV = 'production';
      const opt = new ConnectionPoolOptimizer();
      
      expect(opt._getMultiplier()).toBe(2.5);
    });

    test('should get correct multiplier for test', () => {
      process.env.NODE_ENV = 'test';
      const opt = new ConnectionPoolOptimizer();
      
      expect(opt._getMultiplier()).toBe(0.5);
    });
  });

  describe('Timeout Optimization', () => {
    test('should get optimal timeouts', () => {
      const timeouts = optimizer.getOptimalTimeouts();

      expect(timeouts).toHaveProperty('idleTimeoutMillis');
      expect(timeouts).toHaveProperty('connectionTimeoutMillis');
      expect(timeouts).toHaveProperty('statementTimeout');
      expect(timeouts).toHaveProperty('keepAlive');
    });

    test('should have valid timeout values', () => {
      const timeouts = optimizer.getOptimalTimeouts();

      expect(timeouts.idleTimeoutMillis).toBeGreaterThan(0);
      expect(timeouts.connectionTimeoutMillis).toBeGreaterThan(0);
      expect(timeouts.statementTimeout).toBeGreaterThan(0);
    });

    test('should respect environment idle timeout', () => {
      process.env.DB_IDLE_TIMEOUT = '45000';
      const opt = new ConnectionPoolOptimizer();
      const timeouts = opt.getOptimalTimeouts();
      
      expect(timeouts.idleTimeoutMillis).toBe(45000);
    });

    test('should respect environment connection timeout', () => {
      process.env.DB_CONNECTION_TIMEOUT = '8000';
      const opt = new ConnectionPoolOptimizer();
      const timeouts = opt.getOptimalTimeouts();
      
      expect(timeouts.connectionTimeoutMillis).toBe(8000);
    });

    test('should enable keepAlive', () => {
      const timeouts = optimizer.getOptimalTimeouts();
      expect(timeouts.keepAlive).toBe(true);
    });

    test('should have longer timeouts in production', () => {
      process.env.NODE_ENV = 'production';
      const opt = new ConnectionPoolOptimizer();
      const prodTimeouts = opt.getOptimalTimeouts();

      process.env.NODE_ENV = 'development';
      const devOpt = new ConnectionPoolOptimizer();
      const devTimeouts = devOpt.getOptimalTimeouts();

      expect(prodTimeouts.idleTimeoutMillis).toBeGreaterThanOrEqual(devTimeouts.idleTimeoutMillis);
    });
  });

  describe('Health Check Configuration', () => {
    test('should get optimal health check config', () => {
      const healthCheck = optimizer.getOptimalHealthCheckConfig();

      expect(healthCheck).toHaveProperty('interval');
      expect(healthCheck).toHaveProperty('maxResponseTime');
      expect(healthCheck).toHaveProperty('minIdleConnections');
      expect(healthCheck).toHaveProperty('maxConnectionUsage');
    });

    test('should respect environment health check interval', () => {
      process.env.DB_HEALTH_CHECK_INTERVAL = '60000';
      const opt = new ConnectionPoolOptimizer();
      const healthCheck = opt.getOptimalHealthCheckConfig();
      
      expect(healthCheck.interval).toBe(60000);
    });

    test('should calculate min idle connections based on pool size', () => {
      const healthCheck = optimizer.getOptimalHealthCheckConfig();

      expect(healthCheck.minIdleConnections).toBeGreaterThanOrEqual(2);
    });

    test('should set max connection usage below 1', () => {
      const healthCheck = optimizer.getOptimalHealthCheckConfig();

      expect(healthCheck.maxConnectionUsage).toBeLessThan(1);
      expect(healthCheck.maxConnectionUsage).toBeGreaterThan(0.5);
    });
  });

  describe('Leak Detection Configuration', () => {
    test('should get optimal leak detection config', () => {
      const leakDetection = optimizer.getOptimalLeakDetectionConfig();

      expect(leakDetection).toHaveProperty('threshold');
      expect(leakDetection).toHaveProperty('checkInterval');
      expect(leakDetection).toHaveProperty('maxRecords');
      expect(leakDetection).toHaveProperty('autoCleanup');
    });

    test('should respect environment leak threshold', () => {
      process.env.DB_LEAK_THRESHOLD = '45000';
      const opt = new ConnectionPoolOptimizer();
      const leakDetection = opt.getOptimalLeakDetectionConfig();
      
      expect(leakDetection.threshold).toBe(45000);
    });

    test('should enable auto cleanup in production', () => {
      process.env.NODE_ENV = 'production';
      const opt = new ConnectionPoolOptimizer();
      const leakDetection = opt.getOptimalLeakDetectionConfig();
      
      expect(leakDetection.autoCleanup).toBe(true);
    });

    test('should have reasonable check interval', () => {
      const leakDetection = optimizer.getOptimalLeakDetectionConfig();

      expect(leakDetection.checkInterval).toBeGreaterThan(0);
      expect(leakDetection.checkInterval).toBeLessThan(60000);
    });
  });

  describe('Reaping Configuration', () => {
    test('should get optimal reaping config', () => {
      const reaping = optimizer.getReapingConfig();

      expect(reaping).toHaveProperty('interval');
      expect(reaping).toHaveProperty('excessIdleThreshold');
    });

    test('should respect environment reaping interval', () => {
      process.env.DB_REAPING_INTERVAL = '90000';
      const opt = new ConnectionPoolOptimizer();
      const reaping = opt.getReapingConfig();
      
      expect(reaping.interval).toBe(90000);
    });

    test('should calculate excess idle threshold', () => {
      const reaping = optimizer.getReapingConfig();

      expect(reaping.excessIdleThreshold).toBeGreaterThan(0);
    });
  });

  describe('Slow Query Threshold', () => {
    test('should get slow query threshold', () => {
      const threshold = optimizer.getSlowQueryThreshold();

      expect(threshold).toBeGreaterThan(0);
    });

    test('should respect environment slow query threshold', () => {
      process.env.SLOW_QUERY_THRESHOLD = '2000';
      const opt = new ConnectionPoolOptimizer();
      const threshold = opt.getSlowQueryThreshold();
      
      expect(threshold).toBe(2000);
    });

    test('should have lower threshold in development', () => {
      process.env.NODE_ENV = 'development';
      const opt = new ConnectionPoolOptimizer();
      const devThreshold = opt.getSlowQueryThreshold();

      process.env.NODE_ENV = 'production';
      const prodOpt = new ConnectionPoolOptimizer();
      const prodThreshold = prodOpt.getOptimalLeakDetectionConfig();

      expect(devThreshold).toBeLessThan(1000);
    });
  });

  describe('Complete Configuration', () => {
    test('should generate complete configuration', () => {
      const config = optimizer.getCompleteConfiguration();

      expect(config).toHaveProperty('environment');
      expect(config).toHaveProperty('isProduction');
      expect(config).toHaveProperty('systemInfo');
      expect(config).toHaveProperty('pool');
      expect(config).toHaveProperty('timeouts');
      expect(config).toHaveProperty('healthCheck');
      expect(config).toHaveProperty('leakDetection');
      expect(config).toHaveProperty('reaping');
      expect(config).toHaveProperty('slowQueryThreshold');
    });

    test('should include system information', () => {
      const config = optimizer.getCompleteConfiguration();

      expect(config.systemInfo.cpuCores).toBe(optimizer.cpuCores);
      expect(config.systemInfo.totalMemoryGB).toBeGreaterThan(0);
      expect(config.systemInfo.freeMemoryGB).toBeGreaterThan(0);
    });

    test('should include reasoning for pool configuration', () => {
      const config = optimizer.getCompleteConfiguration();

      expect(config.pool).toHaveProperty('reasoning');
      expect(config.pool.reasoning).toHaveProperty('cpuMultiplier');
      expect(config.pool.reasoning).toHaveProperty('memoryBasedOn');
    });
  });

  describe('Configuration Validation', () => {
    test('should validate valid configuration', () => {
      const config = optimizer.getCompleteConfiguration();
      const validation = optimizer.validateConfiguration(config);

      expect(validation.valid).toBe(true);
      expect(validation.errors).toHaveLength(0);
    });

    test('should detect negative minimum pool size', () => {
      const config = {
        pool: { min: -1, max: 10 },
        timeouts: optimizer.getOptimalTimeouts(),
        leakDetection: optimizer.getOptimalLeakDetectionConfig()
      };
      
      const validation = optimizer.validateConfiguration(config);

      expect(validation.valid).toBe(false);
      expect(validation.errors).toContain('Minimum pool size cannot be negative');
    });

    test('should detect invalid min/max relationship', () => {
      const config = {
        pool: { min: 20, max: 10 },
        timeouts: optimizer.getOptimalTimeouts(),
        leakDetection: optimizer.getOptimalLeakDetectionConfig()
      };
      
      const validation = optimizer.validateConfiguration(config);

      expect(validation.valid).toBe(false);
      expect(validation.errors).toContain('Maximum pool size must be greater than or equal to minimum pool size');
    });

    test('should warn about excessive max pool size', () => {
      const config = {
        pool: { min: 5, max: 150 },
        timeouts: optimizer.getOptimalTimeouts(),
        leakDetection: optimizer.getOptimalLeakDetectionConfig()
      };
      
      const validation = optimizer.validateConfiguration(config);

      expect(validation.warnings).toContain('Maximum pool size exceeds recommended limit of 100');
    });

    test('should warn about high connection timeout', () => {
      const config = {
        pool: { min: 5, max: 20 },
        timeouts: {
          connectionTimeoutMillis: 60000,
          idleTimeoutMillis: 30000,
          statementTimeout: 30000
        },
        leakDetection: optimizer.getOptimalLeakDetectionConfig()
      };
      
      const validation = optimizer.validateConfiguration(config);

      expect(validation.warnings).toContain('Connection timeout is very high, consider reducing');
    });

    test('should warn about low idle timeout', () => {
      const config = {
        pool: { min: 5, max: 20 },
        timeouts: {
          connectionTimeoutMillis: 10000,
          idleTimeoutMillis: 5000,
          statementTimeout: 10000
        },
        leakDetection: optimizer.getOptimalLeakDetectionConfig()
      };
      
      const validation = optimizer.validateConfiguration(config);

      expect(validation.warnings).toContain('Idle timeout is very low, connections may be closed too quickly');
    });
  });

  describe('Environment-Specific Configurations', () => {
    test('should configure for production environment', () => {
      process.env.NODE_ENV = 'production';
      const opt = new ConnectionPoolOptimizer();
      
      const config = opt.getCompleteConfiguration();

      expect(config.environment).toBe('production');
      expect(config.isProduction).toBe(true);
      expect(config.pool.max).toBeLessThanOrEqual(50);
    });

    test('should configure for staging environment', () => {
      process.env.NODE_ENV = 'staging';
      const opt = new ConnectionPoolOptimizer();
      
      const config = opt.getCompleteConfiguration();

      expect(config.environment).toBe('staging');
      expect(config.isProduction).toBe(false);
    });

    test('should configure for test environment', () => {
      process.env.NODE_ENV = 'test';
      const opt = new ConnectionPoolOptimizer();
      
      const config = opt.getCompleteConfiguration();

      expect(config.environment).toBe('test');
      expect(config.pool.max).toBeLessThanOrEqual(5);
    });

    test('should configure for development environment', () => {
      process.env.NODE_ENV = 'development';
      const opt = new ConnectionPoolOptimizer();
      
      const config = opt.getCompleteConfiguration();

      expect(config.environment).toBe('development');
      expect(config.pool.max).toBeLessThanOrEqual(20);
    });
  });
});
