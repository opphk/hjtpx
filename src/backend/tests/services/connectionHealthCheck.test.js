const ConnectionHealthCheck = require('../../services/connectionHealthCheck');

describe('ConnectionHealthCheck Service', () => {
  let healthCheck;
  let mockDbPoolManager;

  beforeEach(() => {
    mockDbPoolManager = {
      query: jest.fn(),
      getPoolStats: jest.fn(),
      config: {
        database: 'test_db'
      }
    };
    
    healthCheck = new ConnectionHealthCheck(mockDbPoolManager);
  });

  afterEach(() => {
    healthCheck.stop();
    jest.clearAllMocks();
  });

  describe('Initialization', () => {
    test('should initialize with default thresholds', () => {
      expect(healthCheck.healthThresholds).toBeDefined();
      expect(healthCheck.healthThresholds.maxResponseTime).toBeDefined();
      expect(healthCheck.healthThresholds.minIdleConnections).toBeDefined();
      expect(healthCheck.healthThresholds.maxConnectionUsage).toBeDefined();
    });

    test('should set custom thresholds from environment', () => {
      process.env.DB_HEALTH_MAX_RESPONSE_TIME = '3000';
      process.env.DB_HEALTH_MIN_IDLE = '5';
      
      const customHealthCheck = new ConnectionHealthCheck(mockDbPoolManager);
      
      expect(customHealthCheck.healthThresholds.maxResponseTime).toBe(3000);
      expect(customHealthCheck.healthThresholds.minIdleConnections).toBe(5);
    });
  });

  describe('Health Check Execution', () => {
    test('should perform health check successfully', async () => {
      mockDbPoolManager.query
        .mockResolvedValueOnce({ rows: [{ health: 1, timestamp: new Date(), version: 'test' }] })
        .mockResolvedValueOnce({ rows: [{ size: 1000000 }] })
        .mockResolvedValueOnce({ rows: [{ active_connections: '5' }] });
      
      mockDbPoolManager.getPoolStats.mockReturnValue({
        total: 10,
        idle: 5,
        busy: 5,
        config: { max: 20 }
      });

      const result = await healthCheck.performHealthCheck();

      expect(result.overallStatus).toBe('healthy');
      expect(result.healthScore).toBeGreaterThan(0);
      expect(result.checks).toBeDefined();
      expect(result.checks.length).toBe(5);
    });

    test('should detect unhealthy state', async () => {
      mockDbPoolManager.query.mockRejectedValue(new Error('Connection failed'));

      const result = await healthCheck.performHealthCheck();

      expect(result.overallStatus).toBe('unhealthy');
      expect(healthCheck.isHealthy).toBe(false);
    });

    test('should track consecutive failures', async () => {
      mockDbPoolManager.query.mockRejectedValue(new Error('Connection failed'));

      await healthCheck.performHealthCheck();
      expect(healthCheck.consecutiveFailures).toBe(1);

      await healthCheck.performHealthCheck();
      expect(healthCheck.consecutiveFailures).toBe(2);
    });

    test('should reset consecutive failures on success', async () => {
      mockDbPoolManager.query
        .mockRejectedValueOnce(new Error('Connection failed'))
        .mockResolvedValue({ rows: [{ health: 1 }] });
      
      mockDbPoolManager.getPoolStats.mockReturnValue({
        total: 5,
        idle: 3,
        busy: 2,
        config: { max: 10 }
      });

      await healthCheck.performHealthCheck();
      expect(healthCheck.consecutiveFailures).toBe(1);

      await healthCheck.performHealthCheck();
      expect(healthCheck.consecutiveFailures).toBe(0);
    });
  });

  describe('Database Connectivity Check', () => {
    test('should check database connectivity', async () => {
      const mockResult = { rows: [{ health: 1, timestamp: new Date(), version: 'PostgreSQL 14' }] };
      mockDbPoolManager.query.mockResolvedValue(mockResult);

      const result = await healthCheck._checkDatabaseConnectivity();

      expect(result.passed).toBe(true);
      expect(result.version).toBe('PostgreSQL 14');
      expect(mockDbPoolManager.query).toHaveBeenCalled();
    });

    test('should fail connectivity check on slow response', async () => {
      mockDbPoolManager.query.mockImplementation(async () => {
        await new Promise(resolve => setTimeout(resolve, 100));
        return { rows: [{ health: 1 }] };
      });

      healthCheck.healthThresholds.maxResponseTime = 50;

      const result = await healthCheck._checkDatabaseConnectivity();

      expect(result.passed).toBe(false);
      expect(result.responseTime).toBeGreaterThan(50);
    });
  });

  describe('Connection Pool Status Check', () => {
    test('should check pool status', async () => {
      mockDbPoolManager.getPoolStats.mockReturnValue({
        total: 20,
        idle: 5,
        idleConnections: 5,
        busy: 15,
        config: { max: 20 }
      });

      const result = await healthCheck._checkConnectionPoolStatus();

      expect(result.totalConnections).toBe(20);
      expect(result.idleConnections).toBe(5);
      expect(result.busyConnections).toBe(15);
    });

    test('should fail when idle connections too low', async () => {
      mockDbPoolManager.getPoolStats.mockReturnValue({
        total: 20,
        idle: 1,
        idleConnections: 1,
        busy: 19,
        config: { max: 20 }
      });

      healthCheck.healthThresholds.minIdleConnections = 3;

      const result = await healthCheck._checkConnectionPoolStatus();

      expect(result.passed).toBe(false);
    });
  });

  describe('Query Performance Check', () => {
    test('should check query performance', async () => {
      mockDbPoolManager.query.mockResolvedValue({ rows: [] });

      const result = await healthCheck._checkQueryPerformance();

      expect(result.passed).toBe(true);
      expect(result.responseTime).toBeGreaterThanOrEqual(0);
    });
  });

  describe('Active Connections Check', () => {
    test('should check active connections', async () => {
      mockDbPoolManager.query.mockResolvedValue({ rows: [{ active_connections: '10' }] });
      mockDbPoolManager.getPoolStats.mockReturnValue({ config: { max: 50 } });

      const result = await healthCheck._checkActiveConnections();

      expect(result.activeConnections).toBe(10);
      expect(result.maxConnections).toBe(50);
      expect(result.passed).toBe(true);
    });
  });

  describe('Deep Health Check', () => {
    test('should perform deep health check', async () => {
      mockDbPoolManager.query.mockResolvedValue({ rows: [] });
      mockDbPoolManager.getPoolStats.mockReturnValue({
        total: 10,
        idle: 5,
        config: { max: 20 }
      });

      const result = await healthCheck.performDeepHealthCheck();

      expect(result.checks.length).toBeGreaterThan(0);
      expect(result.timestamp).toBeDefined();
    });
  });

  describe('Health History', () => {
    test('should maintain health history', async () => {
      mockDbPoolManager.query
        .mockResolvedValueOnce({ rows: [{ health: 1 }] })
        .mockResolvedValueOnce({ rows: [{ size: 1000 }] })
        .mockResolvedValueOnce({ rows: [{ active_connections: '5' }] });
      
      mockDbPoolManager.getPoolStats.mockReturnValue({
        total: 5,
        idle: 2,
        busy: 3,
        config: { max: 10 }
      });

      await healthCheck.performHealthCheck();
      await healthCheck.performHealthCheck();
      await healthCheck.performHealthCheck();

      expect(healthCheck.healthHistory.length).toBe(3);
      expect(healthCheck.healthHistory.length).toBeLessThanOrEqual(healthCheck.maxHistorySize);
    });

    test('should limit history size', () => {
      healthCheck.maxHistorySize = 5;
      
      for (let i = 0; i < 10; i++) {
        healthCheck.healthHistory.push({ timestamp: new Date().toISOString() });
      }

      expect(healthCheck.healthHistory.length).toBe(5);
    });
  });

  describe('Event Emission', () => {
    test('should emit healthCheck event', async () => {
      const handler = jest.fn();
      healthCheck.on('healthCheck', handler);

      mockDbPoolManager.query
        .mockResolvedValueOnce({ rows: [{ health: 1 }] })
        .mockResolvedValueOnce({ rows: [{ size: 1000 }] })
        .mockResolvedValueOnce({ rows: [{ active_connections: '5' }] });
      
      mockDbPoolManager.getPoolStats.mockReturnValue({
        total: 5,
        idle: 2,
        busy: 3,
        config: { max: 10 }
      });

      await healthCheck.performHealthCheck();

      expect(handler).toHaveBeenCalled();
    });

    test('should emit unhealthyState event after consecutive failures', async () => {
      const handler = jest.fn();
      healthCheck.on('unhealthyState', handler);

      healthCheck.maxConsecutiveFailures = 3;
      mockDbPoolManager.query.mockRejectedValue(new Error('Failed'));

      for (let i = 0; i < 3; i++) {
        await healthCheck.performHealthCheck();
      }

      expect(handler).toHaveBeenCalled();
    });
  });

  describe('Start and Stop', () => {
    test('should start health check interval', () => {
      const intervalSpy = jest.spyOn(global, 'setInterval');

      healthCheck.start(5000);

      expect(intervalSpy).toHaveBeenCalled();
      expect(healthCheck.checkInterval).toBeDefined();
    });

    test('should stop health check interval', () => {
      healthCheck.start(5000);
      healthCheck.stop();

      expect(healthCheck.checkInterval).toBeNull();
    });
  });

  describe('Health Report', () => {
    test('should generate health report', () => {
      const report = healthCheck.getHealthReport();

      expect(report).toHaveProperty('checkCount');
      expect(report).toHaveProperty('healthScore');
      expect(report).toHaveProperty('isHealthy');
      expect(report).toHaveProperty('thresholds');
    });

    test('should generate detailed stats', () => {
      const stats = healthCheck.getDetailedStats();

      expect(stats).toHaveProperty('summary');
      expect(stats).toHaveProperty('recentHistory');
      expect(stats).toHaveProperty('allTimeStats');
    });
  });

  describe('Reset', () => {
    test('should reset health check state', () => {
      healthCheck.healthHistory = [{ id: 1 }, { id: 2 }];
      healthCheck.checkCount = 10;
      healthCheck.consecutiveFailures = 3;
      healthCheck.healthScore = 50;

      healthCheck.reset();

      expect(healthCheck.healthHistory).toEqual([]);
      expect(healthCheck.checkCount).toBe(0);
      expect(healthCheck.consecutiveFailures).toBe(0);
      expect(healthCheck.healthScore).toBe(100);
    });
  });
});
