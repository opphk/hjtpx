const {
  apiCache,
  invalidateCache,
  userCacheMiddleware,
  shouldCache,
  generateCacheKey,
  getCacheConfig,
  cacheConfig
} = require('../../middleware/cacheMiddleware');

jest.mock('../../services/cacheService', () => ({
  getCachedApiResponse: jest.fn(),
  setCachedApiResponse: jest.fn(),
  getCachedUser: jest.fn(),
  setCachedUser: jest.fn(),
  invalidateApiCache: jest.fn(),
  isHealthy: jest.fn()
}));

const cacheService = require('../../services/cacheService');

describe('Cache Middleware', () => {
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
      json: jest.fn(data => data),
      send: jest.fn(data => data),
      on: jest.fn()
    };

    mockNext = jest.fn();
  });

  describe('shouldCache', () => {
    test('should return false for non-GET methods', () => {
      mockReq.method = 'POST';
      expect(shouldCache(mockReq)).toBe(false);
    });

    test('should return false when noCache query is true', () => {
      mockReq.query.noCache = 'true';
      expect(shouldCache(mockReq)).toBe(false);
    });

    test('should return false for admin users', () => {
      mockReq.user = { id: 1, role: 'admin' };
      expect(shouldCache(mockReq)).toBe(false);
    });

    test('should return true for GET requests without noCache', () => {
      mockReq.method = 'GET';
      expect(shouldCache(mockReq)).toBe(true);
    });
  });

  describe('generateCacheKey', () => {
    test('should generate basic cache key', () => {
      mockReq.method = 'GET';
      mockReq.originalUrl = '/api/v1/users';
      mockReq.user = null;

      const key = generateCacheKey(mockReq);

      expect(key).toContain('GET:/api/v1/users');
    });

    test('should include user ID in cache key for authenticated requests', () => {
      mockReq.method = 'GET';
      mockReq.originalUrl = '/api/v1/profile';
      mockReq.user = { id: 'user-123' };

      const key = generateCacheKey(mockReq);

      expect(key).toContain('user-123');
    });

    test('should include query parameters in cache key', () => {
      mockReq.method = 'GET';
      mockReq.originalUrl = '/api/v1/users';
      mockReq.query = { page: 1, limit: 10 };
      mockReq.user = null;

      const key = generateCacheKey(mockReq);

      expect(key).toContain('page=1');
      expect(key).toContain('limit=10');
    });
  });

  describe('getCacheConfig', () => {
    test('should return config for users endpoint', () => {
      const config = getCacheConfig('/api/v1/users');

      expect(config).toHaveProperty('ttl');
      expect(config).toHaveProperty('isPublic');
    });

    test('should return default config for unknown endpoint', () => {
      const config = getCacheConfig('/api/v1/unknown');

      expect(config).toHaveProperty('ttl');
      expect(config).toHaveProperty('isPublic');
    });

    test('should return config for health endpoint', () => {
      const config = getCacheConfig('/api/v1/health');

      expect(config.ttl).toBeLessThanOrEqual(10);
    });

    test('should return config for notifications endpoint', () => {
      const config = getCacheConfig('/api/v1/notifications');

      expect(config).toHaveProperty('ttl');
    });
  });

  describe('apiCache middleware', () => {
    test('should call next for non-cacheable requests', async () => {
      mockReq.method = 'POST';

      const middleware = apiCache();
      await middleware(mockReq, mockRes, mockNext);

      expect(mockNext).toHaveBeenCalled();
      expect(cacheService.getCachedApiResponse).not.toHaveBeenCalled();
    });

    test('should return cached response on cache hit', async () => {
      const cachedData = { users: [{ id: 1, name: 'Test User' }] };
      cacheService.getCachedApiResponse.mockResolvedValue(cachedData);

      const middleware = apiCache();
      await middleware(mockReq, mockRes, mockNext);

      expect(mockRes.set).toHaveBeenCalledWith('X-Cache', 'HIT');
      expect(mockRes.json).toHaveBeenCalledWith(cachedData);
    });

    test('should set X-Cache header to MISS on cache miss', async () => {
      cacheService.getCachedApiResponse.mockResolvedValue(null);

      const middleware = apiCache();
      await middleware(mockReq, mockRes, mockNext);

      expect(mockRes.set).toHaveBeenCalledWith('X-Cache', 'MISS');
      expect(mockNext).toHaveBeenCalled();
    });

    test('should not cache admin user responses', async () => {
      mockReq.user = { id: 1, role: 'admin' };

      const middleware = apiCache();
      await middleware(mockReq, mockRes, mockNext);

      expect(mockNext).toHaveBeenCalled();
      expect(cacheService.getCachedApiResponse).not.toHaveBeenCalled();
    });

    test('should cache responses when cacheService is available', async () => {
      cacheService.getCachedApiResponse.mockResolvedValue(null);

      mockRes.on = jest.fn((event, callback) => {
        if (event === 'finish') {
          mockRes.finishCallback = callback;
        }
      });

      const middleware = apiCache(300);
      await middleware(mockReq, mockRes, mockNext);

      expect(mockNext).toHaveBeenCalled();
      expect(mockRes.on).toHaveBeenCalledWith('finish', expect.any(Function));
    });
  });

  describe('invalidateCache middleware', () => {
    test('should call next and invalidate pattern', async () => {
      cacheService.invalidateApiCache.mockResolvedValue(true);

      const middleware = invalidateCache('users:*');
      await middleware(mockReq, mockRes, mockNext);

      expect(cacheService.invalidateApiCache).toHaveBeenCalledWith('users:*');
      expect(mockNext).toHaveBeenCalled();
    });

    test('should call next even on error', async () => {
      cacheService.invalidateApiCache.mockRejectedValue(new Error('Cache error'));

      const middleware = invalidateCache('*');
      await middleware(mockReq, mockRes, mockNext);

      expect(mockNext).toHaveBeenCalled();
    });
  });

  describe('userCacheMiddleware', () => {
    test('should skip for non-GET requests', async () => {
      mockReq.method = 'POST';
      mockReq.user = { id: 'user-123' };

      const middleware = userCacheMiddleware();
      await middleware(mockReq, mockRes, mockNext);

      expect(mockNext).toHaveBeenCalled();
      expect(cacheService.getCachedUser).not.toHaveBeenCalled();
    });

    test('should skip for unauthenticated requests', async () => {
      mockReq.method = 'GET';
      mockReq.user = null;

      const middleware = userCacheMiddleware();
      await middleware(mockReq, mockRes, mockNext);

      expect(mockNext).toHaveBeenCalled();
      expect(cacheService.getCachedUser).not.toHaveBeenCalled();
    });

    test('should set X-User-Cache header to HIT on cache hit', async () => {
      const cachedUser = { id: 'user-123', name: 'Test User' };
      cacheService.getCachedUser.mockResolvedValue(cachedUser);

      mockReq.method = 'GET';
      mockReq.user = { id: 'user-123' };

      const middleware = userCacheMiddleware();
      await middleware(mockReq, mockRes, mockNext);

      expect(mockRes.set).toHaveBeenCalledWith('X-User-Cache', 'HIT');
      expect(mockReq.cachedUser).toEqual(cachedUser);
    });

    test('should set X-User-Cache header to MISS on cache miss', async () => {
      cacheService.getCachedUser.mockResolvedValue(null);

      mockReq.method = 'GET';
      mockReq.user = { id: 'user-123' };

      const middleware = userCacheMiddleware();
      await middleware(mockReq, mockRes, mockNext);

      expect(mockRes.set).toHaveBeenCalledWith('X-User-Cache', 'MISS');
      expect(mockNext).toHaveBeenCalled();
    });
  });

  describe('cacheConfig', () => {
    test('should have predefined endpoints', () => {
      expect(cacheConfig).toHaveProperty('/api/v1/users');
      expect(cacheConfig).toHaveProperty('/api/v1/notifications');
      expect(cacheConfig).toHaveProperty('/api/v1/health');
    });

    test('should have appropriate TTL values', () => {
      expect(cacheConfig['/api/v1/health'].ttl).toBeLessThanOrEqual(10);
      expect(cacheConfig['/api/v1/notifications'].ttl).toBeLessThanOrEqual(60);
    });
  });
});
