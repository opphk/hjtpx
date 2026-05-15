const cacheConfig = require('../../config/cacheConfig');

describe('CacheConfig', () => {
  describe('Cache TTL Configuration', () => {
    test('should have default TTL values', () => {
      expect(cacheConfig.ttl).toBeDefined();
      expect(cacheConfig.ttl.DEFAULT).toBeDefined();
      expect(cacheConfig.ttl.SESSION).toBeDefined();
      expect(cacheConfig.ttl.USER).toBeDefined();
      expect(cacheConfig.ttl.API_PUBLIC).toBeDefined();
      expect(cacheConfig.ttl.API_PRIVATE).toBeDefined();
    });

    test('should have session TTL configured', () => {
      expect(cacheConfig.ttl.SESSION).toBeGreaterThan(0);
      expect(cacheConfig.ttl.SESSION).toBeLessThanOrEqual(604800);
    });

    test('should have user cache TTL configured', () => {
      expect(cacheConfig.ttl.USER).toBeGreaterThan(0);
      expect(cacheConfig.ttl.USER).toBeLessThanOrEqual(3600);
    });

    test('should have API cache TTL configured', () => {
      expect(cacheConfig.ttl.API_PUBLIC).toBeDefined();
      expect(cacheConfig.ttl.API_PRIVATE).toBeDefined();
      expect(cacheConfig.ttl.API_PUBLIC).toBeGreaterThan(0);
      expect(cacheConfig.ttl.API_PRIVATE).toBeGreaterThan(0);
    });
  });

  describe('Cache Size Limits', () => {
    test('should have max size configuration', () => {
      expect(cacheConfig.maxSize).toBeDefined();
      expect(cacheConfig.maxSize.MEMORY_CACHE).toBeDefined();
      expect(cacheConfig.maxSize.MEMORY_CACHE).toBeGreaterThan(0);
    });

    test('should have max entry size configured', () => {
      expect(cacheConfig.maxSize.MAX_ENTRY_SIZE).toBeDefined();
      expect(cacheConfig.maxSize.MAX_ENTRY_SIZE).toBeGreaterThan(0);
    });

    test('should have eviction threshold configured', () => {
      expect(cacheConfig.maxSize.EVICTION_THRESHOLD).toBeDefined();
      expect(cacheConfig.maxSize.EVICTION_THRESHOLD).toBeGreaterThan(0);
      expect(cacheConfig.maxSize.EVICTION_THRESHOLD).toBeLessThanOrEqual(1);
    });
  });

  describe('Cache Warmup Strategy', () => {
    test('should have warmup configuration', () => {
      expect(cacheConfig.warmup).toBeDefined();
      expect(cacheConfig.warmup.ENABLED).toBeDefined();
      expect(typeof cacheConfig.warmup.ENABLED).toBe('boolean');
    });

    test('should have warmup endpoints configured', () => {
      expect(cacheConfig.warmup.ENDPOINTS).toBeDefined();
      expect(Array.isArray(cacheConfig.warmup.ENDPOINTS)).toBe(true);
    });

    test('should have warmup priority levels', () => {
      expect(cacheConfig.warmup.PRIORITY).toBeDefined();
      expect(cacheConfig.warmup.PRIORITY.HIGH).toBeDefined();
      expect(cacheConfig.warmup.PRIORITY.MEDIUM).toBeDefined();
      expect(cacheConfig.warmup.PRIORITY.LOW).toBeDefined();
    });

    test('should have warmup interval configured', () => {
      expect(cacheConfig.warmup.INTERVAL).toBeDefined();
      expect(cacheConfig.warmup.INTERVAL).toBeGreaterThan(0);
    });
  });

  describe('Cache Policy Configuration', () => {
    test('should have eviction policy configured', () => {
      expect(cacheConfig.policy).toBeDefined();
      expect(cacheConfig.policy.EVICTION_STRATEGY).toBeDefined();
    });

    test('should have compression settings', () => {
      expect(cacheConfig.policy.COMPRESSION_ENABLED).toBeDefined();
      expect(cacheConfig.policy.COMPRESSION_THRESHOLD).toBeDefined();
      expect(cacheConfig.policy.COMPRESSION_THRESHOLD).toBeGreaterThan(0);
    });

    test('should have stale-while-revalidate setting', () => {
      expect(cacheConfig.policy.STALE_WHILE_REVALIDATE).toBeDefined();
      expect(typeof cacheConfig.policy.STALE_WHILE_REVALIDATE).toBe('boolean');
    });

    test('should have sliding expiration setting', () => {
      expect(cacheConfig.policy.SLIDING_EXPIRATION).toBeDefined();
      expect(typeof cacheConfig.policy.SLIDING_EXPIRATION).toBe('boolean');
    });
  });

  describe('Cache Endpoint Configuration', () => {
    test('should have endpoint-specific configurations', () => {
      expect(cacheConfig.endpoints).toBeDefined();
      expect(typeof cacheConfig.endpoints).toBe('object');
    });

    test('should have health check endpoint config', () => {
      const healthConfig = cacheConfig.endpoints['/api/v1/health'];
      expect(healthConfig).toBeDefined();
      expect(healthConfig.ttl).toBeDefined();
      expect(healthConfig.isPublic).toBe(true);
    });

    test('should have user endpoint config', () => {
      const userConfig = cacheConfig.endpoints['/api/v1/users'];
      expect(userConfig).toBeDefined();
      expect(userConfig.ttl).toBeDefined();
      expect(userConfig.tags).toBeDefined();
    });
  });

  describe('Cache Statistics Configuration', () => {
    test('should have statistics configuration', () => {
      expect(cacheConfig.stats).toBeDefined();
      expect(cacheConfig.stats.ENABLED).toBeDefined();
      expect(typeof cacheConfig.stats.ENABLED).toBe('boolean');
    });

    test('should have stats collection interval', () => {
      expect(cacheConfig.stats.COLLECTION_INTERVAL).toBeDefined();
      expect(cacheConfig.stats.COLLECTION_INTERVAL).toBeGreaterThan(0);
    });

    test('should have histogram buckets configured', () => {
      expect(cacheConfig.stats.HISTOGRAM_BUCKETS).toBeDefined();
      expect(Array.isArray(cacheConfig.stats.HISTOGRAM_BUCKETS)).toBe(true);
      expect(cacheConfig.stats.HISTOGRAM_BUCKETS.length).toBeGreaterThan(0);
    });
  });

  describe('Cache Tags Configuration', () => {
    test('should have tags configuration', () => {
      expect(cacheConfig.tags).toBeDefined();
      expect(Array.isArray(cacheConfig.tags)).toBe(true);
    });

    test('should have common tags defined', () => {
      expect(cacheConfig.tags).toContain('user');
      expect(cacheConfig.tags).toContain('session');
    });
  });

  describe('Cache Helpers', () => {
    test('should have getTTL helper function', () => {
      expect(cacheConfig.getTTL).toBeDefined();
      expect(typeof cacheConfig.getTTL).toBe('function');
    });

    test('should get correct TTL for known endpoint', () => {
      const ttl = cacheConfig.getTTL('/api/v1/health');
      expect(ttl).toBeGreaterThan(0);
    });

    test('should return default TTL for unknown endpoint', () => {
      const ttl = cacheConfig.getTTL('/api/v1/unknown');
      expect(ttl).toBe(cacheConfig.ttl.DEFAULT);
    });

    test('should have getMaxSize helper function', () => {
      expect(cacheConfig.getMaxSize).toBeDefined();
      expect(typeof cacheConfig.getMaxSize).toBe('function');
    });

    test('should have isCacheable helper function', () => {
      expect(cacheConfig.isCacheable).toBeDefined();
      expect(typeof cacheConfig.isCacheable).toBe('function');
    });
  });
});
