const cacheMetricsService = require('../../services/cacheMetricsService');

describe('CacheMetricsService', () => {
  beforeEach(() => {
    cacheMetricsService.resetMetrics();
    cacheMetricsService.cleanup();
  });

  afterEach(async () => {
    if (cacheMetricsService) {
      await cacheMetricsService.cleanup();
    }
  });

  describe('Metrics Collection', () => {
    test('should initialize with default metrics', () => {
      const metrics = cacheMetricsService.getMetrics();
      
      expect(metrics).toBeDefined();
      expect(metrics.cache).toBeDefined();
      expect(metrics.cache.hits).toBeDefined();
      expect(metrics.cache.misses).toBeDefined();
    });

    test('should track cache hits', () => {
      cacheMetricsService.recordHit('session', 10);
      cacheMetricsService.recordHit('api', 15);
      
      const metrics = cacheMetricsService.getMetrics();
      
      expect(metrics.cache.hits).toBe(25);
    });

    test('should track cache misses', () => {
      cacheMetricsService.recordMiss('session', 5);
      cacheMetricsService.recordMiss('api', 8);
      
      const metrics = cacheMetricsService.getMetrics();
      
      expect(metrics.cache.misses).toBe(13);
    });

    test('should calculate hit rate', () => {
      cacheMetricsService.recordHit('test', 10);
      cacheMetricsService.recordMiss('test', 5);
      
      const hitRate = cacheMetricsService.getHitRate('test');
      
      expect(hitRate).toBeGreaterThan(60);
      expect(hitRate).toBeLessThan(70);
    });

    test('should track set operations', () => {
      cacheMetricsService.recordSet('session', 5);
      cacheMetricsService.recordSet('api', 3);
      
      const metrics = cacheMetricsService.getMetrics();
      
      expect(metrics.cache.sets).toBe(8);
    });

    test('should track delete operations', () => {
      cacheMetricsService.recordDelete('session', 2);
      cacheMetricsService.recordDelete('api', 4);
      
      const metrics = cacheMetricsService.getMetrics();
      
      expect(metrics.cache.deletes).toBe(6);
    });
  });

  describe('Latency Metrics', () => {
    test('should record latency for operations', () => {
      cacheMetricsService.recordLatency('get', 50);
      cacheMetricsService.recordLatency('get', 100);
      cacheMetricsService.recordLatency('set', 75);
      
      const metrics = cacheMetricsService.getMetrics();
      
      expect(metrics.latency.get).toBeDefined();
      expect(metrics.latency.get.length).toBeGreaterThan(0);
    });

    test('should calculate average latency', () => {
      cacheMetricsService.recordLatency('get', 50);
      cacheMetricsService.recordLatency('get', 100);
      cacheMetricsService.recordLatency('get', 150);
      
      const avgLatency = cacheMetricsService.getAverageLatency('get');
      
      expect(avgLatency).toBe(100);
    });

    test('should calculate percentile latency', () => {
      for (let i = 0; i < 100; i++) {
        cacheMetricsService.recordLatency('get', i);
      }
      
      const p95 = cacheMetricsService.getPercentileLatency('get', 95);
      
      expect(p95).toBeGreaterThan(90);
      expect(p95).toBeLessThan(100);
    });

    test('should return 0 for empty latency data', () => {
      const avgLatency = cacheMetricsService.getAverageLatency('nonexistent');
      expect(avgLatency).toBe(0);
    });
  });

  describe('Memory Metrics', () => {
    test('should track memory cache size', () => {
      cacheMetricsService.setMemoryCacheSize(500);
      cacheMetricsService.setMemoryCacheSize(750);
      
      const metrics = cacheMetricsService.getMetrics();
      
      expect(metrics.memory.currentSize).toBe(750);
    });

    test('should track memory cache max size', () => {
      cacheMetricsService.setMemoryCacheMaxSize(1000);
      
      const metrics = cacheMetricsService.getMetrics();
      
      expect(metrics.memory.maxSize).toBe(1000);
    });

    test('should calculate memory usage percentage', () => {
      cacheMetricsService.setMemoryCacheMaxSize(1000);
      cacheMetricsService.setMemoryCacheSize(800);
      
      const usagePercent = cacheMetricsService.getMemoryUsagePercent();
      
      expect(usagePercent).toBe(80);
    });

    test('should track memory evictions', () => {
      cacheMetricsService.recordEviction('LRU', 5);
      cacheMetricsService.recordEviction('SIZE', 3);
      
      const metrics = cacheMetricsService.getMetrics();
      
      expect(metrics.memory.evictions).toBe(8);
    });
  });

  describe('Error Tracking', () => {
    test('should track cache errors', () => {
      cacheMetricsService.recordError('connection', 'Redis connection failed');
      cacheMetricsService.recordError('timeout', 'Operation timed out');
      
      const metrics = cacheMetricsService.getMetrics();
      
      expect(metrics.errors.total).toBe(2);
      expect(metrics.errors.byType.connection).toBe(1);
      expect(metrics.errors.byType.timeout).toBe(1);
    });

    test('should track error rate', () => {
      cacheMetricsService.recordHit('test', 100);
      cacheMetricsService.recordError('test', 'Test error');
      
      const errorRate = cacheMetricsService.getErrorRate();
      
      expect(errorRate).toBeGreaterThan(0);
      expect(errorRate).toBeLessThan(2);
    });
  });

  describe('Session Metrics', () => {
    test('should track session operations', () => {
      cacheMetricsService.recordSessionOperation('create', 1);
      cacheMetricsService.recordSessionOperation('read', 10);
      cacheMetricsService.recordSessionOperation('update', 3);
      cacheMetricsService.recordSessionOperation('delete', 2);
      
      const metrics = cacheMetricsService.getMetrics();
      
      expect(metrics.session.totalOperations).toBe(16);
      expect(metrics.session.creates).toBe(1);
      expect(metrics.session.reads).toBe(10);
      expect(metrics.session.updates).toBe(3);
      expect(metrics.session.deletes).toBe(2);
    });

    test('should track active sessions', () => {
      cacheMetricsService.setActiveSessions(50);
      cacheMetricsService.incrementActiveSessions();
      cacheMetricsService.decrementActiveSessions();
      
      const metrics = cacheMetricsService.getMetrics();
      
      expect(metrics.session.active).toBe(50);
    });

    test('should track session expiration rate', () => {
      cacheMetricsService.recordSessionExpiration();
      cacheMetricsService.recordSessionExpiration();
      cacheMetricsService.recordSessionExpiration();
      
      const metrics = cacheMetricsService.getMetrics();
      
      expect(metrics.session.expired).toBe(3);
    });
  });

  describe('API Cache Metrics', () => {
    test('should track API cache hits and misses', () => {
      cacheMetricsService.recordApiCacheHit('/api/v1/users');
      cacheMetricsService.recordApiCacheHit('/api/v1/users');
      cacheMetricsService.recordApiCacheHit('/api/v1/profile');
      cacheMetricsService.recordApiCacheMiss('/api/v1/users');
      
      const metrics = cacheMetricsService.getMetrics();
      
      expect(metrics.api.hits).toBe(3);
      expect(metrics.api.misses).toBe(1);
      expect(metrics.api.endpoints['/api/v1/users'].hits).toBe(2);
      expect(metrics.api.endpoints['/api/v1/users'].misses).toBe(1);
    });

    test('should track endpoint-specific hit rate', () => {
      cacheMetricsService.recordApiCacheHit('/api/v1/health');
      cacheMetricsService.recordApiCacheHit('/api/v1/health');
      cacheMetricsService.recordApiCacheMiss('/api/v1/health');
      
      const hitRate = cacheMetricsService.getEndpointHitRate('/api/v1/health');
      
      expect(hitRate).toBeCloseTo(66.67, 1);
    });

    test('should track API cache size', () => {
      cacheMetricsService.setApiCacheSize(100);
      
      const metrics = cacheMetricsService.getMetrics();
      
      expect(metrics.api.size).toBe(100);
    });
  });

  describe('Historical Metrics', () => {
    test('should store historical data', () => {
      cacheMetricsService.recordHit('test', 5);
      cacheMetricsService.recordMiss('test', 2);
      
      cacheMetricsService.recordSnapshot();
      
      const history = cacheMetricsService.getHistory(1);
      
      expect(history).toBeDefined();
      expect(history.length).toBeGreaterThan(0);
    });

    test('should limit history size', () => {
      for (let i = 0; i < 150; i++) {
        cacheMetricsService.recordHit('test', 1);
        cacheMetricsService.recordSnapshot();
      }
      
      const history = cacheMetricsService.getHistory();
      
      expect(history.length).toBeLessThanOrEqual(1440);
    });

    test('should calculate trends', () => {
      cacheMetricsService.recordHit('trend', 10);
      cacheMetricsService.recordSnapshot();
      cacheMetricsService.recordHit('trend', 20);
      cacheMetricsService.recordSnapshot();
      cacheMetricsService.recordHit('trend', 30);
      cacheMetricsService.recordSnapshot();
      
      const trend = cacheMetricsService.getTrend('hits');
      
      expect(trend).toBeDefined();
    });
  });

  describe('Performance Metrics', () => {
    test('should calculate throughput', () => {
      for (let i = 0; i < 10; i++) {
        cacheMetricsService.recordOperation('read', 1000);
      }
      
      const throughput = cacheMetricsService.getThroughput();
      
      expect(parseFloat(throughput)).toBeGreaterThanOrEqual(0);
    });

    test('should track concurrent operations', () => {
      const initialValue = cacheMetricsService.metrics.performance.concurrentOperations;
      cacheMetricsService.incrementConcurrentOperations();
      
      const afterFirst = cacheMetricsService.metrics.performance.concurrentOperations;
      expect(afterFirst).toBe(initialValue + 1);
      
      cacheMetricsService.incrementConcurrentOperations();
      const afterSecond = cacheMetricsService.metrics.performance.concurrentOperations;
      expect(afterSecond).toBe(initialValue + 2);
      
      cacheMetricsService.decrementConcurrentOperations();
      const afterDecrement = cacheMetricsService.metrics.performance.concurrentOperations;
      expect(afterDecrement).toBe(initialValue + 1);
    });
  });

  describe('Alerts', () => {
    test('should generate alerts for high error rate', () => {
      cacheMetricsService.recordHit('test', 10);
      for (let i = 0; i < 20; i++) {
        cacheMetricsService.recordError('test', 'Test error');
      }
      
      const alerts = cacheMetricsService.checkAlerts();
      
      expect(alerts).toBeDefined();
    });

    test('should generate alerts for low hit rate', () => {
      cacheMetricsService.recordMiss('test', 50);
      
      const alerts = cacheMetricsService.checkAlerts();
      
      expect(alerts).toBeDefined();
    });

    test('should generate alerts for high memory usage', () => {
      cacheMetricsService.setMemoryCacheMaxSize(100);
      cacheMetricsService.setMemoryCacheSize(95);
      
      const alerts = cacheMetricsService.checkAlerts();
      
      expect(alerts).toBeDefined();
    });
  });

  describe('Export', () => {
    test('should export metrics as JSON', () => {
      cacheMetricsService.recordHit('export', 10);
      
      const json = cacheMetricsService.exportMetrics('json');
      
      expect(json).toBeDefined();
      expect(typeof json).toBe('string');
      expect(() => JSON.parse(json)).not.toThrow();
    });

    test('should export metrics summary', () => {
      const summary = cacheMetricsService.getSummary();
      
      expect(summary).toBeDefined();
      expect(summary.hitRate).toBeDefined();
      expect(summary.totalOperations).toBeDefined();
    });
  });

  describe('Reset', () => {
    test('should reset all metrics', () => {
      cacheMetricsService.recordHit('reset', 100);
      cacheMetricsService.recordMiss('reset', 50);
      
      cacheMetricsService.resetMetrics();
      
      const metrics = cacheMetricsService.getMetrics();
      
      expect(metrics.cache.hits).toBe(0);
      expect(metrics.cache.misses).toBe(0);
    });
  });

  describe('Cleanup', () => {
    test('should cleanup resources', async () => {
      await cacheMetricsService.cleanup();
      
      expect(cacheMetricsService).toBeDefined();
    });
  });
});
