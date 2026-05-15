const {
  apiCache,
  generateCacheKey,
  generateAdvancedCacheKey,
  getCacheConfig,
  cacheConfig,
  shouldCache,
  shouldCacheResponse
} = require('../../middleware/cacheMiddleware');

jest.mock('../../services/cacheService', () => ({
  getCachedApiResponse: jest.fn(),
  setCachedApiResponse: jest.fn(),
  getCachedUser: jest.fn(),
  setCachedUser: jest.fn(),
  invalidateApiCache: jest.fn(),
  isHealthy: jest.fn(),
  getStats: jest.fn(() => ({
    hits: 0,
    misses: 0,
    overall: { hitRate: '0%' }
  }))
}));

jest.mock('../../services/cacheMetricsService', () => ({
  recordHit: jest.fn(),
  recordMiss: jest.fn(),
  recordLatency: jest.fn(),
  recordApiCacheHit: jest.fn(),
  recordApiCacheMiss: jest.fn(),
  setApiCacheSize: jest.fn()
}));

const cacheService = require('../../services/cacheService');
const cacheMetricsService = require('../../services/cacheMetricsService');

describe('Cache Middleware Enhanced', () => {
  let mockReq;
  let mockRes;
  let mockNext;

  beforeEach(() => {
    jest.clearAllMocks();
    
    mockReq = {
      method: 'GET',
      originalUrl: '/api/v1/users',
      path: '/api/v1/users',
      query: {},
      user: null,
      headers: { accept: 'application/json' },
      xhr: false
    };

    mockRes = {
      statusCode: 200,
      set: jest.fn(),
      json: jest.fn((data) => data),
      send: jest.fn((data) => data),
      on: jest.fn()
    };

    mockNext = jest.fn();
  });

  describe('Cache Key Generation Strategies', () => {
    test('should generate basic cache key', () => {
      const key = generateCacheKey(mockReq);
      
      expect(key).toContain('GET');
      expect(key).toContain('/api/v1/users');
    });

    test('should generate cache key with user ID', () => {
      mockReq.user = { id: 'user-123', role: 'user' };
      
      const key = generateCacheKey(mockReq);
      
      expect(key).toContain('user-123');
    });

    test('should generate cache key with query parameters', () => {
      mockReq.query = { page: 1, limit: 10 };
      
      const key = generateCacheKey(mockReq);
      
      expect(key).toContain('page=1');
      expect(key).toContain('limit=10');
    });

    test('should generate advanced cache key with multiple components', () => {
      mockReq.user = { id: 'user-123', role: 'admin' };
      mockReq.query = { sort: 'name' };
      mockReq.headers['accept-language'] = 'en-US,en;q=0.9';
      
      const key = generateAdvancedCacheKey(mockReq);
      
      expect(key).toContain('u:user-123');
      expect(key).toContain('r:admin');
      expect(key).toContain('l:en-US');
      expect(key).toContain('q:sort=name');
    });

    test('should include role in advanced cache key', () => {
      mockReq.user = { id: 'user-123', role: 'superadmin' };
      
      const key = generateAdvancedCacheKey(mockReq);
      
      expect(key).toContain('r:superadmin');
    });

    test('should handle custom key generation options', () => {
      mockReq.user = { id: 'user-123', role: 'user' };
      
      const key = generateAdvancedCacheKey(mockReq, {
        customKey: 'custom-data',
        includeUserId: true,
        includeQuery: true
      });
      
      expect(key).toContain('c:custom-data');
    });

    test('should exclude user ID when configured', () => {
      mockReq.user = { id: 'user-123', role: 'user' };
      
      const key = generateAdvancedCacheKey(mockReq, {
        includeUserId: false
      });
      
      expect(key).not.toContain('u:user-123');
    });

    test('should exclude query when configured', () => {
      mockReq.query = { page: 1 };
      
      const key = generateAdvancedCacheKey(mockReq, {
        includeQuery: false
      });
      
      expect(key).not.toContain('q:page=1');
    });
  });

  describe('Cache Hit/Miss Handling', () => {
    test('should record metrics on cache hit', async () => {
      const cachedData = { users: [] };
      cacheService.getCachedApiResponse.mockResolvedValue(cachedData);
      
      const middleware = apiCache();
      await middleware(mockReq, mockRes, mockNext);
      
      expect(cacheMetricsService.recordApiCacheHit).toHaveBeenCalled();
      expect(mockRes.set).toHaveBeenCalledWith('X-Cache', 'HIT');
    });

    test('should record metrics on cache miss', async () => {
      cacheService.getCachedApiResponse.mockResolvedValue(null);
      
      const middleware = apiCache();
      await middleware(mockReq, mockRes, mockNext);
      
      expect(cacheMetricsService.recordApiCacheMiss).toHaveBeenCalled();
      expect(mockRes.set).toHaveBeenCalledWith('X-Cache', 'MISS');
    });

    test('should record latency for cache operations', async () => {
      cacheService.getCachedApiResponse.mockResolvedValue({ data: 'test' });
      
      const middleware = apiCache();
      await middleware(mockReq, mockRes, mockNext);
      
      expect(cacheMetricsService.recordLatency).toHaveBeenCalled();
    });

    test('should set cache headers on hit', async () => {
      const cachedData = { users: [{ id: 1 }] };
      cacheService.getCachedApiResponse.mockResolvedValue(cachedData);
      
      const middleware = apiCache(300);
      await middleware(mockReq, mockRes, mockNext);
      
      expect(mockRes.set).toHaveBeenCalledWith('X-Cache-Key', expect.any(String));
      expect(mockRes.set).toHaveBeenCalledWith('X-Cache-TTL', expect.any(String));
      expect(mockRes.set).toHaveBeenCalledWith('X-Cache-Hit-Time', expect.any(String));
    });

    test('should set cache headers on miss', async () => {
      cacheService.getCachedApiResponse.mockResolvedValue(null);
      
      const middleware = apiCache(300);
      await middleware(mockReq, mockRes, mockNext);
      
      expect(mockRes.set).toHaveBeenCalledWith('X-Cache-Key', expect.any(String));
      expect(mockRes.set).toHaveBeenCalledWith('X-Cache-Miss-Time', expect.any(String));
    });
  });

  describe('Cache Configuration', () => {
    test('should return config for known endpoint', () => {
      const config = getCacheConfig('/api/v1/users');
      
      expect(config).toHaveProperty('ttl');
      expect(config).toHaveProperty('isPublic');
      expect(config).toHaveProperty('tags');
    });

    test('should return default config for unknown endpoint', () => {
      const config = getCacheConfig('/api/v1/unknown-endpoint');
      
      expect(config.ttl).toBe(300);
      expect(config.isPublic).toBe(true);
    });

    test('should match prefix patterns', () => {
      const config = getCacheConfig('/api/v1/users/123');
      
      expect(config).toHaveProperty('ttl');
    });

    test('should have health endpoint with short TTL', () => {
      const config = getCacheConfig('/api/v1/health');
      
      expect(config.ttl).toBeLessThanOrEqual(10);
      expect(config.isPublic).toBe(true);
    });
  });

  describe('Should Cache Logic', () => {
    test('should cache GET requests', () => {
      mockReq.method = 'GET';
      expect(shouldCache(mockReq)).toBe(true);
    });

    test('should cache HEAD requests', () => {
      mockReq.method = 'HEAD';
      expect(shouldCache(mockReq)).toBe(true);
    });

    test('should not cache POST requests', () => {
      mockReq.method = 'POST';
      expect(shouldCache(mockReq)).toBe(false);
    });

    test('should not cache when noCache is requested', () => {
      mockReq.query.noCache = 'true';
      expect(shouldCache(mockReq)).toBe(false);
    });

    test('should not cache when cache-control no-cache is set', () => {
      mockReq.headers['cache-control'] = 'no-cache';
      expect(shouldCache(mockReq)).toBe(false);
    });

    test('should not cache admin user responses', () => {
      mockReq.user = { id: 1, role: 'admin' };
      expect(shouldCache(mockReq)).toBe(false);
    });

    test('should cache non-admin user responses', () => {
      mockReq.user = { id: 1, role: 'user' };
      expect(shouldCache(mockReq)).toBe(true);
    });
  });

  describe('Should Cache Response Logic', () => {
    test('should cache 200 OK responses', () => {
      mockRes.statusCode = 200;
      expect(shouldCacheResponse(mockRes)).toBe(true);
    });

    test('should cache 201 Created responses', () => {
      mockRes.statusCode = 201;
      expect(shouldCacheResponse(mockRes)).toBe(true);
    });

    test('should not cache 400 Bad Request responses', () => {
      mockRes.statusCode = 400;
      expect(shouldCacheResponse(mockRes)).toBe(false);
    });

    test('should not cache 500 Server Error responses', () => {
      mockRes.statusCode = 500;
      expect(shouldCacheResponse(mockRes)).toBe(false);
    });

    test('should not cache 404 Not Found responses', () => {
      mockRes.statusCode = 404;
      expect(shouldCacheResponse(mockRes)).toBe(false);
    });
  });

  describe('Cache Statistics', () => {
    test('should track endpoint-specific metrics', async () => {
      const cachedData = { users: [] };
      cacheService.getCachedApiResponse.mockResolvedValue(cachedData);
      
      const middleware = apiCache();
      await middleware(mockReq, mockRes, mockNext);
      
      expect(cacheMetricsService.recordApiCacheHit).toHaveBeenCalledWith('/api/v1/users');
    });

    test('should update cache size on operations', async () => {
      const cachedData = { users: [] };
      cacheService.getCachedApiResponse.mockResolvedValue(cachedData);
      
      const middleware = apiCache();
      await middleware(mockReq, mockRes, mockNext);
      
      expect(cacheMetricsService.setApiCacheSize).toHaveBeenCalled();
    });
  });

  describe('Custom Cache Options', () => {
    test('should use custom TTL when provided', async () => {
      cacheService.getCachedApiResponse.mockResolvedValue({ data: 'test' });
      
      const middleware = apiCache(600);
      await middleware(mockReq, mockRes, mockNext);
      
      expect(mockRes.set).toHaveBeenCalledWith('X-Cache-TTL', expect.any(String));
    });

    test('should use custom key generator', async () => {
      const customKeyGenerator = jest.fn(() => 'custom:key');
      cacheService.getCachedApiResponse.mockResolvedValue(null);
      
      const middleware = apiCache(300, { keyGenerator: customKeyGenerator });
      await middleware(mockReq, mockRes, mockNext);
      
      expect(customKeyGenerator).toHaveBeenCalledWith(mockReq);
    });

    test('should use custom tags when provided', async () => {
      cacheService.getCachedApiResponse.mockResolvedValue(null);
      
      const middleware = apiCache(300, { tags: ['custom-tag'] });
      await middleware(mockReq, mockRes, mockNext);
      
      expect(mockRes.set).toHaveBeenCalled();
    });
  });

  describe('Middleware Flow', () => {
    test('should skip non-cacheable requests', async () => {
      mockReq.method = 'POST';
      
      const middleware = apiCache();
      await middleware(mockReq, mockRes, mockNext);
      
      expect(mockNext).toHaveBeenCalled();
      expect(cacheService.getCachedApiResponse).not.toHaveBeenCalled();
    });

    test('should proceed to next middleware on cache miss', async () => {
      cacheService.getCachedApiResponse.mockResolvedValue(null);
      
      const middleware = apiCache();
      await middleware(mockReq, mockRes, mockNext);
      
      expect(mockNext).toHaveBeenCalled();
    });

    test('should not call next on cache hit', async () => {
      const cachedData = { users: [] };
      cacheService.getCachedApiResponse.mockResolvedValue(cachedData);
      
      const middleware = apiCache();
      await middleware(mockReq, mockRes, mockNext);
      
      expect(mockRes.json).toHaveBeenCalledWith(cachedData);
    });

    test('should handle errors gracefully', async () => {
      cacheService.getCachedApiResponse.mockRejectedValue(new Error('Redis error'));
      
      const middleware = apiCache();
      await middleware(mockReq, mockRes, mockNext);
      
      expect(mockNext).toHaveBeenCalled();
    });
  });
});
