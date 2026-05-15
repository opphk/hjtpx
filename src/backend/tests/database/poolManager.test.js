jest.mock('../../config/database/dbPoolManager', () => {
  const EventEmitter = require('events');
  
  class MockDatabasePoolManager extends EventEmitter {
    constructor() {
      super();
      this.pool = {
        query: jest.fn(),
        connect: jest.fn(),
        end: jest.fn(),
        totalCount: 10,
        idleCount: 5,
        waitingCount: 0,
        on: jest.fn(),
        emit: jest.fn()
      };
      this.isProduction = process.env.NODE_ENV === 'production';
      this.config = {
        host: 'localhost',
        port: 5432,
        database: 'test',
        min: 2,
        max: 10,
        idleTimeoutMillis: 30000,
        connectionTimeoutMillis: 5000,
        keepAlive: true,
        keepAliveInitialDelayMillis: 30000
      };
      this.checkedOutClients = new Map();
      this.stats = {
        queries: 0,
        slowQueries: 0,
        errors: 0,
        avgQueryTime: 0,
        totalQueryTime: 0,
        maxQueryTime: 0,
        minQueryTime: Infinity,
        connectionLeaks: 0,
        connectionLeakEvents: [],
        healthCheckFailures: 0,
        connectionErrors: 0,
        lastSuccessfulQuery: null,
        lastError: null
      };
      this.queryTimes = [];
      this.slowQueryThreshold = 100;
      this.healthCheckInterval = null;
      this.leakCheckInterval = null;
      this.reapingInterval = null;
      this.initialized = false;
      this.lastHealthCheck = null;
      this.healthCheckHistory = [];
    }

    initialize() {
      if (this.pool) {
        return this.pool;
      }
      this.initialized = true;
      return this.pool;
    }

    async query(text, params, options = {}) {
      const start = Date.now();
      try {
        const result = await this.pool.query(text, params);
        const duration = Date.now() - start;
        
        if (options.trackStats !== false) {
          this.stats.queries++;
          this.stats.totalQueryTime += duration;
          this.stats.lastSuccessfulQuery = new Date().toISOString();
          this.queryTimes.push(duration);
        }
        
        return result;
      } catch (error) {
        this.stats.errors++;
        this.stats.lastError = error.message;
        throw error;
      }
    }

    async getClient() {
      const mockClient = {
        query: jest.fn().mockResolvedValue({ rows: [] }),
        release: jest.fn()
      };
      return mockClient;
    }

    async transaction(callback) {
      const client = await this.getClient();
      try {
        await client.query('BEGIN');
        const result = await callback(client);
        await client.query('COMMIT');
        return result;
      } catch (error) {
        await client.query('ROLLBACK');
        throw error;
      } finally {
        client.release();
      }
    }

    async batchQuery(queries) {
      const client = await this.getClient();
      try {
        await client.query('BEGIN');
        const results = await Promise.all(
          queries.map(async ({ query, params }) => {
            const result = await client.query(query, params);
            return result.rows;
          })
        );
        await client.query('COMMIT');
        return results;
      } catch (error) {
        await client.query('ROLLBACK');
        throw error;
      } finally {
        client.release();
      }
    }

    async healthCheck() {
      const start = Date.now();
      try {
        return {
          healthy: true,
          responseTime: Date.now() - start,
          timestamp: new Date().toISOString(),
          poolStatus: {
            total: 10,
            idle: 5,
            busy: 5,
            waiting: 0
          }
        };
      } catch (error) {
        return {
          healthy: false,
          error: error.message,
          timestamp: new Date().toISOString()
        };
      }
    }

    getPoolStats() {
      return {
        total: this.pool.totalCount,
        idle: this.pool.idleCount,
        busy: this.pool.totalCount - this.pool.idleCount,
        waiting: this.pool.waitingCount,
        checkedOut: this.checkedOutClients.size,
        capacityUsage: `${((this.pool.totalCount - this.pool.idleCount) / this.pool.totalCount * 100).toFixed(2)}%`,
        config: this.config
      };
    }

    getQueryStats() {
      return {
        ...this.stats,
        p50QueryTime: this.queryTimes.length > 0 ? this.queryTimes[Math.floor(this.queryTimes.length * 0.5)] : 0,
        p75QueryTime: this.queryTimes.length > 0 ? this.queryTimes[Math.floor(this.queryTimes.length * 0.75)] : 0,
        p95QueryTime: this.queryTimes.length > 0 ? this.queryTimes[Math.floor(this.queryTimes.length * 0.95)] : 0,
        p99QueryTime: this.queryTimes.length > 0 ? this.queryTimes[Math.floor(this.queryTimes.length * 0.99)] : 0,
        errorRate: this.stats.queries > 0 ? `${(this.stats.errors / this.stats.queries * 100).toFixed(2)}%` : '0%',
        hitRate: this.stats.queries > 0 ? `${((1 - this.stats.slowQueries / this.stats.queries) * 100).toFixed(2)}%` : '100%'
      };
    }

    resetStats() {
      this.stats = {
        queries: 0,
        slowQueries: 0,
        errors: 0,
        avgQueryTime: 0,
        totalQueryTime: 0,
        maxQueryTime: 0,
        minQueryTime: Infinity,
        connectionLeaks: 0,
        connectionLeakEvents: [],
        healthCheckFailures: 0,
        connectionErrors: 0,
        lastSuccessfulQuery: null,
        lastError: null
      };
      this.queryTimes = [];
    }

    async close() {
      this.pool = null;
      this.initialized = false;
    }
  }

  return new MockDatabasePoolManager();
});

describe('Database Pool Manager Tests', () => {
  let poolManager;

  beforeEach(() => {
    jest.clearAllMocks();
    poolManager = require('../../config/database/dbPoolManager');
  });

  afterEach(async () => {
    if (poolManager.pool) {
      await poolManager.close();
    }
  });

  describe('Pool Configuration', () => {
    test('should initialize with optimized config', () => {
      const config = poolManager.config;
      
      expect(config).toHaveProperty('host');
      expect(config).toHaveProperty('port');
      expect(config).toHaveProperty('database');
      expect(config).toHaveProperty('min');
      expect(config).toHaveProperty('max');
      expect(config).toHaveProperty('idleTimeoutMillis');
      expect(config).toHaveProperty('connectionTimeoutMillis');
    });

    test('should set correct pool size based on CPU cores in production', () => {
      const os = require('os');
      const cpuCores = os.cpus().length;
      const config = poolManager.config;
      
      if (process.env.NODE_ENV === 'production') {
        const expectedMin = Math.max(5, Math.floor(cpuCores * 1.5));
        const expectedMax = Math.min(50, cpuCores * 10);
        
        expect(config.min).toBeGreaterThanOrEqual(expectedMin - 1);
        expect(config.max).toBeLessThanOrEqual(expectedMax + 1);
      }
    });

    test('should set reasonable timeouts', () => {
      const config = poolManager.config;
      
      expect(config.idleTimeoutMillis).toBeGreaterThan(0);
      expect(config.connectionTimeoutMillis).toBeGreaterThan(0);
      expect(config.idleTimeoutMillis).toBeLessThan(300000);
      expect(config.connectionTimeoutMillis).toBeLessThan(60000);
    });

    test('should enable keepAlive for connection stability', () => {
      const config = poolManager.config;
      
      expect(config.keepAlive).toBe(true);
      expect(config.keepAliveInitialDelayMillis).toBeGreaterThan(0);
    });
  });

  describe('Pool Initialization', () => {
    test('should initialize pool without error', () => {
      const pool = poolManager.initialize();
      
      expect(pool).toBeDefined();
      expect(poolManager.pool).toBeDefined();
      expect(poolManager.initialized).toBe(true);
    });

    test('should not reinitialize existing pool', () => {
      const firstPool = poolManager.initialize();
      const secondPool = poolManager.initialize();
      
      expect(firstPool).toBe(secondPool);
    });
  });

  describe('Query Execution', () => {
    beforeEach(() => {
      poolManager.initialize();
    });

    test('should execute simple query', async () => {
      const mockResult = { rows: [{ id: 1 }], rowCount: 1 };
      poolManager.pool.query = jest.fn().mockResolvedValue(mockResult);
      
      const result = await poolManager.query('SELECT 1');
      
      expect(result).toEqual(mockResult);
      expect(poolManager.stats.queries).toBeGreaterThanOrEqual(1);
    });

    test('should track slow queries', async () => {
      const slowQueryDuration = poolManager.slowQueryThreshold + 100;
      poolManager.pool.query = jest.fn().mockImplementation(async () => {
        await new Promise(resolve => setTimeout(resolve, slowQueryDuration));
        return { rows: [] };
      });
      
      await poolManager.query('SELECT pg_sleep(1)');
      
      expect(poolManager.stats.slowQueries).toBeGreaterThanOrEqual(1);
    });

    test('should log query execution time', async () => {
      poolManager.pool.query = jest.fn().mockResolvedValue({ rows: [] });
      
      await poolManager.query('SELECT 1');
      
      expect(poolManager.stats.queries).toBeGreaterThanOrEqual(1);
      expect(poolManager.stats.totalQueryTime).toBeGreaterThan(0);
    });
  });

  describe('Connection Management', () => {
    beforeEach(() => {
      poolManager.initialize();
    });

    test('should acquire client from pool', async () => {
      const mockClient = {
        query: jest.fn().mockResolvedValue({ rows: [] }),
        release: jest.fn()
      };
      poolManager.pool.connect = jest.fn().mockResolvedValue(mockClient);
      
      const client = await poolManager.getClient();
      
      expect(client).toBeDefined();
      expect(client.release).toBeDefined();
    });

    test('should track checked out clients', async () => {
      const mockClient = {
        query: jest.fn().mockResolvedValue({ rows: [] }),
        release: jest.fn()
      };
      poolManager.pool.connect = jest.fn().mockResolvedValue(mockClient);
      
      const client = await poolManager.getClient();
      
      expect(poolManager.checkedOutClients.size).toBeGreaterThan(0);
      
      client.release();
      
      expect(poolManager.checkedOutClients.size).toBe(0);
    });
  });

  describe('Transaction Support', () => {
    beforeEach(() => {
      poolManager.initialize();
    });

    test('should execute transaction with commit', async () => {
      const mockClient = {
        query: jest.fn().mockResolvedValue({ rows: [] }),
        release: jest.fn()
      };
      poolManager.pool.connect = jest.fn().mockResolvedValue(mockClient);
      
      const result = await poolManager.transaction(async (client) => {
        return { success: true };
      });
      
      expect(result.success).toBe(true);
      expect(mockClient.query).toHaveBeenCalledWith('BEGIN');
      expect(mockClient.query).toHaveBeenCalledWith('COMMIT');
      expect(mockClient.release).toHaveBeenCalled();
    });

    test('should execute transaction with rollback on error', async () => {
      const mockClient = {
        query: jest.fn().mockResolvedValue({ rows: [] }),
        release: jest.fn()
      };
      poolManager.pool.connect = jest.fn().mockResolvedValue(mockClient);
      
      await expect(poolManager.transaction(async () => {
        throw new Error('Transaction failed');
      })).rejects.toThrow('Transaction failed');
      
      expect(mockClient.query).toHaveBeenCalledWith('BEGIN');
      expect(mockClient.query).toHaveBeenCalledWith('ROLLBACK');
      expect(mockClient.release).toHaveBeenCalled();
    });
  });

  describe('Health Check', () => {
    beforeEach(() => {
      poolManager.initialize();
    });

    test('should perform health check successfully', async () => {
      poolManager.pool.query = jest.fn().mockResolvedValue({ rows: [{ health: 1, timestamp: new Date() }] });
      poolManager.pool.totalCount = 10;
      poolManager.pool.idleCount = 5;
      poolManager.pool.waitingCount = 0;
      
      const health = await poolManager.healthCheck();
      
      expect(health.healthy).toBe(true);
      expect(health.responseTime).toBeGreaterThan(0);
      expect(health.poolStatus).toBeDefined();
    });

    test('should detect unhealthy pool', async () => {
      poolManager.pool.query = jest.fn().mockRejectedValue(new Error('Connection failed'));
      
      const health = await poolManager.healthCheck();
      
      expect(health.healthy).toBe(false);
      expect(health.error).toBeDefined();
    });
  });

  describe('Pool Statistics', () => {
    beforeEach(() => {
      poolManager.initialize();
    });

    test('should return pool stats', () => {
      poolManager.pool.totalCount = 10;
      poolManager.pool.idleCount = 3;
      poolManager.pool.waitingCount = 0;
      
      const stats = poolManager.getPoolStats();
      
      expect(stats.total).toBe(10);
      expect(stats.idle).toBe(3);
      expect(stats.busy).toBe(7);
      expect(stats.config).toBeDefined();
    });

    test('should return query stats', () => {
      const stats = poolManager.getQueryStats();
      
      expect(stats).toHaveProperty('queries');
      expect(stats).toHaveProperty('slowQueries');
      expect(stats).toHaveProperty('errors');
      expect(stats).toHaveProperty('avgQueryTime');
    });

    test('should calculate percentiles', () => {
      poolManager.queryTimes = [10, 20, 30, 40, 50, 60, 70, 80, 90, 100];
      
      const stats = poolManager.getQueryStats();
      
      expect(stats.p50QueryTime).toBeGreaterThan(0);
      expect(stats.p95QueryTime).toBeGreaterThanOrEqual(stats.p50QueryTime);
    });
  });

  describe('Connection Leak Detection', () => {
    beforeEach(() => {
      poolManager.initialize();
    });

    test('should track checked out clients', async () => {
      const mockClient = {
        query: jest.fn().mockResolvedValue({ rows: [] }),
        release: jest.fn()
      };
      poolManager.pool.connect = jest.fn().mockResolvedValue(mockClient);
      
      await poolManager.getClient();
      
      expect(poolManager.checkedOutClients.size).toBeGreaterThan(0);
    });

    test('should detect potential leaks during interval check', () => {
      const mockClient = {
        checkedOutAt: Date.now() - 60000,
        stackTrace: 'Test stack trace'
      };
      poolManager.checkedOutClients.set('test-client-1', mockClient);
      
      const stats = poolManager.getPoolStats();
      
      expect(stats.checkedOut).toBeGreaterThan(0);
    });
  });

  describe('Error Handling', () => {
    beforeEach(() => {
      poolManager.initialize();
    });

    test('should handle query errors', async () => {
      const error = new Error('Database error');
      poolManager.pool.query = jest.fn().mockRejectedValue(error);
      
      await expect(poolManager.query('SELECT 1')).rejects.toThrow('Database error');
      
      expect(poolManager.stats.errors).toBeGreaterThanOrEqual(1);
      expect(poolManager.stats.lastError).toBe('Database error');
    });

    test('should emit error events', async () => {
      const errorHandler = jest.fn();
      poolManager.on('queryError', errorHandler);
      
      poolManager.pool.query = jest.fn().mockRejectedValue(new Error('Test error'));
      
      try {
        await poolManager.query('SELECT 1');
      } catch (e) {}
      
      expect(errorHandler).toHaveBeenCalled();
    });
  });

  describe('Batch Query', () => {
    beforeEach(() => {
      poolManager.initialize();
    });

    test('should execute batch queries', async () => {
      const mockClient = {
        query: jest.fn().mockResolvedValue({ rows: [] }),
        release: jest.fn()
      };
      poolManager.pool.connect = jest.fn().mockResolvedValue(mockClient);
      
      const queries = [
        { query: 'INSERT INTO users (email) VALUES ($1)', params: ['test1@example.com'] },
        { query: 'INSERT INTO users (email) VALUES ($1)', params: ['test2@example.com'] }
      ];
      
      const results = await poolManager.batchQuery(queries);
      
      expect(results).toHaveLength(2);
      expect(mockClient.query).toHaveBeenCalledWith('BEGIN');
      expect(mockClient.query).toHaveBeenCalledWith('COMMIT');
    });

    test('should rollback batch queries on error', async () => {
      const mockClient = {
        query: jest.fn()
          .mockResolvedValueOnce({ rows: [] })
          .mockRejectedValueOnce(new Error('Batch error')),
        release: jest.fn()
      };
      poolManager.pool.connect = jest.fn().mockResolvedValue(mockClient);
      
      const queries = [
        { query: 'INSERT INTO users (email) VALUES ($1)', params: ['test@example.com'] },
        { query: 'INVALID SQL', params: [] }
      ];
      
      await expect(poolManager.batchQuery(queries)).rejects.toThrow();
      
      expect(mockClient.query).toHaveBeenCalledWith('ROLLBACK');
    });
  });

  describe('Pool Cleanup', () => {
    test('should close pool properly', async () => {
      poolManager.initialize();
      poolManager.pool.end = jest.fn().mockResolvedValue();
      
      await poolManager.close();
      
      expect(poolManager.pool).toBeNull();
      expect(poolManager.initialized).toBe(false);
    });

    test('should clear intervals on close', async () => {
      poolManager.initialize();
      poolManager.pool.end = jest.fn().mockResolvedValue();
      
      await poolManager.close();
      
      expect(poolManager.healthCheckInterval).toBeNull();
      expect(poolManager.leakCheckInterval).toBeNull();
      expect(poolManager.reapingInterval).toBeNull();
    });
  });

  describe('Event Emission', () => {
    beforeEach(() => {
      poolManager.initialize();
    });

    test('should emit clientConnected event', (done) => {
      poolManager.on('clientConnected', (data) => {
        expect(data.timestamp).toBeDefined();
        done();
      });
      
      poolManager.pool.emit('connect', {});
    });

    test('should emit clientAcquired event', (done) => {
      poolManager.on('clientAcquired', (data) => {
        expect(data.timestamp).toBeDefined();
        done();
      });
      
      poolManager.pool.emit('acquire', {});
    });

    test('should emit poolError event', (done) => {
      poolManager.on('poolError', (data) => {
        expect(data.error).toBeDefined();
        expect(data.timestamp).toBeDefined();
        done();
      });
      
      poolManager.pool.emit('error', new Error('Test error'), {});
    });
  });

  describe('Connection Reaping', () => {
    test('should start connection reaping', () => {
      poolManager.initialize();
      poolManager.pool.query = jest.fn().mockResolvedValue({ rows: [] });
      poolManager.pool.idleCount = 10;
      poolManager.pool.totalCount = 10;
      
      expect(poolManager.reapingInterval).toBeDefined();
    });
  });

  describe('Health Check History', () => {
    test('should maintain health check history', async () => {
      poolManager.initialize();
      poolManager.pool.query = jest.fn().mockResolvedValue({ 
        rows: [{ health: 1, timestamp: new Date() }] 
      });
      poolManager.pool.totalCount = 5;
      poolManager.pool.idleCount = 2;
      poolManager.pool.waitingCount = 0;
      
      await poolManager.healthCheck();
      await poolManager.healthCheck();
      await poolManager.healthCheck();
      
      expect(poolManager.healthCheckHistory.length).toBeGreaterThan(0);
      expect(poolManager.healthCheckHistory.length).toBeLessThanOrEqual(100);
    });
  });

  describe('Stats Reset', () => {
    test('should reset all stats', () => {
      poolManager.stats.queries = 100;
      poolManager.stats.errors = 5;
      poolManager.queryTimes = [10, 20, 30];
      
      poolManager.resetStats();
      
      expect(poolManager.stats.queries).toBe(0);
      expect(poolManager.stats.errors).toBe(0);
      expect(poolManager.queryTimes).toEqual([]);
    });
  });

  describe('Connection Capacity', () => {
    test('should calculate capacity usage correctly', () => {
      poolManager.pool = {
        totalCount: 20,
        idleCount: 5,
        waitingCount: 0
      };
      
      const stats = poolManager.getPoolStats();
      
      expect(stats.capacityUsage).toBe('75.00%');
    });

    test('should handle zero total connections', () => {
      poolManager.pool = {
        totalCount: 0,
        idleCount: 0,
        waitingCount: 0
      };
      
      const stats = poolManager.getPoolStats();
      
      expect(stats.capacityUsage).toBe('0%');
    });
  });
});
