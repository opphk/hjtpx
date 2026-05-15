const advancedCacheService = require('../../src/backend/services/advancedCacheService');
const cacheWarmer = require('../../src/backend/services/cache_warming');
const cacheMonitor = require('../../src/backend/services/cacheMonitor');
const cacheConsistency = require('../../src/backend/services/cache_consistency');

describe('Advanced Cache Service', () => {
  beforeEach(async () => {
    await advancedCacheService.clear();
    advancedCacheService.resetStats();
  });

  afterAll(async () => {
    await advancedCacheService.clear();
  });

  describe('Multi-Level Cache Operations', () => {
    test('should set and get value from cache', async () => {
      const key = 'test:key';
      const value = { data: 'test data' };
      
      await advancedCacheService.set(key, value);
      const result = await advancedCacheService.get(key);
      
      expect(result).toEqual(value);
    });

    test('should return null for non-existent key', async () => {
      const result = await advancedCacheService.get('non:existent');
      expect(result).toBeNull();
    });

    test('should delete cache entry', async () => {
      const key = 'test:delete';
      const value = { data: 'delete test' };
      
      await advancedCacheService.set(key, value);
      await advancedCacheService.delete(key);
      
      const result = await advancedCacheService.get(key);
      expect(result).toBeNull();
    });

    test('should track cache statistics', async () => {
      const key = 'test:stats';
      const value = { stats: 'test' };
      
      await advancedCacheService.set(key, value);
      await advancedCacheService.get(key);
      await advancedCacheService.get('non:existent');
      
      const stats = advancedCacheService.getStats();
      
      expect(stats.l1.sets).toBeGreaterThan(0);
      expect(stats.total.sets).toBeGreaterThan(0);
    });
  });

  describe('L1 Cache Operations', () => {
    test('should store in L1 cache when enabled', async () => {
      const key = 'l1:test';
      const value = { level: 1 };
      
      await advancedCacheService.set(key, value, { bypassL1: false });
      const result = advancedCacheService.getFromL1(key);
      
      expect(result).toEqual(value);
    });

    test('should respect L1 size limit', async () => {
      const originalMaxSize = advancedCacheService.config.l1MaxSize;
      advancedCacheService.config.l1MaxSize = 5;
      
      for (let i = 0; i < 10; i++) {
        await advancedCacheService.set(`l1:limit:${i}`, { i });
      }
      
      expect(advancedCacheService.l1Cache.size).toBeLessThanOrEqual(5);
      
      advancedCacheService.config.l1MaxSize = originalMaxSize;
    });
  });

  describe('Cache Locking', () => {
    test('should acquire lock successfully', async () => {
      const lock = await advancedCacheService.acquireLock('test:lock');
      
      expect(lock.acquired).toBe(true);
      expect(lock.value).toBeDefined();
      
      if (lock.acquired) {
        await advancedCacheService.releaseLock('test:lock', lock.value);
      }
    });

    test('should release lock successfully', async () => {
      const lock = await advancedCacheService.acquireLock('test:lock:release');
      
      if (lock.acquired) {
        const released = await advancedCacheService.releaseLock('test:lock:release', lock.value);
        expect(released).toBe(true);
      }
    });

    test('should execute callback with lock', async () => {
      let executed = false;
      
      await advancedCacheService.withLock('test:lock:callback', async () => {
        executed = true;
        return 'result';
      });
      
      expect(executed).toBe(true);
    });
  });

  describe('Cache Versioning', () => {
    test('should increment version on set', async () => {
      const key = 'version:test';
      const versionBefore = advancedCacheService.getVersion(key);
      
      await advancedCacheService.set(key, { data: 'test' });
      const versionAfter = advancedCacheService.getVersion(key);
      
      expect(versionAfter).toBeGreaterThan(versionBefore);
    });
  });

  describe('Cache Statistics', () => {
    test('should calculate hit rate correctly', async () => {
      await advancedCacheService.set('stats:1', { data: 1 });
      await advancedCacheService.get('stats:1');
      await advancedCacheService.get('stats:non');
      
      const stats = advancedCacheService.getStats();
      
      expect(stats.total.hits).toBeGreaterThan(0);
      expect(stats.total.misses).toBeGreaterThan(0);
    });

    test('should track latency', async () => {
      await advancedCacheService.set('latency:test', { data: 'test' });
      await advancedCacheService.get('latency:test');
      
      const avgLatency = advancedCacheService.getAverageLatency(1);
      expect(typeof avgLatency).toBe('string');
    });
  });

  describe('Cache Pattern Operations', () => {
    test('should invalidate by pattern', async () => {
      await advancedCacheService.set('pattern:1', { data: 1 });
      await advancedCacheService.set('pattern:2', { data: 2 });
      await advancedCacheService.set('other:key', { data: 3 });
      
      await advancedCacheService.invalidatePattern('pattern:*');
      
      const result1 = await advancedCacheService.get('pattern:1');
      const result2 = await advancedCacheService.get('pattern:2');
      const result3 = await advancedCacheService.get('other:key');
      
      expect(result1).toBeNull();
      expect(result2).toBeNull();
      expect(result3).toEqual({ data: 3 });
    });
  });

  describe('Get or Set', () => {
    test('should return cached value if exists', async () => {
      const key = 'getset:test';
      const value = { cached: true };
      
      await advancedCacheService.set(key, value);
      
      const result = await advancedCacheService.getOrSet(key, async () => ({
        fetched: true
      }));
      
      expect(result).toEqual(value);
    });

    test('should fetch and cache if not exists', async () => {
      const key = 'getset:miss';
      const fetchedValue = { fetched: true };
      
      const result = await advancedCacheService.getOrSet(key, async () => fetchedValue);
      
      expect(result).toEqual(fetchedValue);
      
      const cached = await advancedCacheService.get(key);
      expect(cached).toEqual(fetchedValue);
    });
  });
});

describe('Cache Warmer', () => {
  test('should initialize without errors', () => {
    expect(cacheWarmer).toBeDefined();
    expect(cacheWarmer.stats).toBeDefined();
  });

  test('should return warming statistics', () => {
    const stats = cacheWarmer.getStats();
    
    expect(stats).toHaveProperty('startupWarmings');
    expect(stats).toHaveProperty('scheduledWarmings');
    expect(stats).toHaveProperty('hotDataWarmings');
    expect(stats).toHaveProperty('totalItemsWarmed');
  });

  test('should update configuration', () => {
    const newConfig = {
      startup: { enabled: false },
      scheduled: { interval: 7200000 }
    };
    
    cacheWarmer.updateConfig(newConfig);
    
    expect(cacheWarmer.warmingConfigs.startup.enabled).toBe(false);
    expect(cacheWarmer.warmingConfigs.scheduled.interval).toBe(7200000);
  });

  test('should warm custom cache', async () => {
    const items = [
      { key: 'custom:1', value: { data: 1 }, ttl: 300 },
      { key: 'custom:2', value: { data: 2 }, ttl: 300 }
    ];
    
    const warmed = await cacheWarmer.warmCustomCache(items);
    
    expect(warmed).toBe(2);
  });
});

describe('Cache Monitor', () => {
  test('should initialize monitoring', () => {
    expect(cacheMonitor).toBeDefined();
    expect(cacheMonitor.metricsHistory).toBeDefined();
    expect(cacheMonitor.alerts).toBeDefined();
  });

  test('should collect metrics', () => {
    const metrics = cacheMonitor.collectMetrics();
    
    expect(metrics).toHaveProperty('timestamp');
    expect(metrics).toHaveProperty('hitRate');
    expect(metrics).toHaveProperty('latency');
    expect(metrics).toHaveProperty('memory');
  });

  test('should generate report', () => {
    const report = cacheMonitor.generateReport();
    
    expect(report).toHaveProperty('generatedAt');
    expect(report).toHaveProperty('summary');
    expect(report).toHaveProperty('performance');
    expect(report).toHaveProperty('capacity');
    expect(report).toHaveProperty('recommendations');
  });

  test('should export metrics in different formats', () => {
    const jsonReport = cacheMonitor.exportMetrics('json');
    const csvReport = cacheMonitor.exportMetrics('csv');
    
    expect(typeof jsonReport).toBe('string');
    expect(typeof csvReport).toBe('string');
    expect(csvReport).toContain('Metric,Value');
  });

  test('should check health status', () => {
    const health = cacheMonitor.getHealthStatus();
    
    expect(health).toHaveProperty('healthy');
    expect(health).toHaveProperty('l1');
    expect(health).toHaveProperty('l2');
    expect(health).toHaveProperty('l3');
    expect(health).toHaveProperty('alerts');
  });

  test('should get alerts', () => {
    const alerts = cacheMonitor.getAlerts();
    
    expect(Array.isArray(alerts)).toBe(true);
  });

  test('should set alert thresholds', () => {
    cacheMonitor.setAlertThresholds({
      hitRate: { warning: 70, critical: 50 },
      latency: { warning: 150, critical: 600 }
    });
    
    expect(cacheMonitor.alertThresholds.hitRate.warning).toBe(70);
    expect(cacheMonitor.alertThresholds.latency.critical).toBe(600);
  });
});

describe('Cache Consistency', () => {
  test('should initialize consistency module', () => {
    expect(cacheConsistency).toBeDefined();
    expect(cacheConsistency.transactionLog).toBeDefined();
  });

  test('should create transaction', async () => {
    const tx = await cacheConsistency.startTransaction();
    
    expect(tx).toHaveProperty('id');
    expect(tx).toHaveProperty('get');
    expect(tx).toHaveProperty('set');
    expect(tx).toHaveProperty('delete');
    expect(tx).toHaveProperty('commit');
    expect(tx).toHaveProperty('rollback');
  });

  test('should get transaction log', () => {
    const log = cacheConsistency.getTransactionLog();
    
    expect(Array.isArray(log)).toBe(true);
  });

  test('should get consistency stats', () => {
    const stats = cacheConsistency.getConsistencyStats();
    
    expect(stats).toHaveProperty('connected');
    expect(stats).toHaveProperty('transactionLogSize');
    expect(stats).toHaveProperty('subscribersCount');
  });

  test('should check health', async () => {
    const health = await cacheConsistency.healthCheck();
    
    expect(health).toHaveProperty('healthy');
  });
});
