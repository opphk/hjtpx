jest.mock('../../../../config/redis/client', () => ({
  isOpen: true,
  connect: jest.fn().mockResolvedValue(undefined),
  get: jest.fn(),
  setEx: jest.fn().mockResolvedValue('OK'),
  del: jest.fn().mockResolvedValue(1),
  expire: jest.fn().mockResolvedValue(1),
  scan: jest.fn().mockResolvedValue({ cursor: 0, keys: [] }),
  sMembers: jest.fn().mockResolvedValue([]),
  sAdd: jest.fn().mockResolvedValue(1),
  flushDb: jest.fn().mockResolvedValue('OK'),
  ping: jest.fn().mockResolvedValue('PONG'),
  multi: jest.fn().mockReturnValue({
    sAdd: jest.fn().mockReturnThis(),
    expire: jest.fn().mockReturnThis(),
    setEx: jest.fn().mockReturnThis(),
    exec: jest.fn().mockResolvedValue([])
  }),
  mGet: jest.fn().mockResolvedValue([]),
  on: jest.fn(),
  off: jest.fn()
}));

const cacheService = require('../../services/cacheService');

describe('CacheService', () => {
  beforeEach(() => {
    cacheService.resetStats();
    cacheService.memoryCache.clear();
  });

  describe('Basic Operations', () => {
    test('should set and get a value', async () => {
      const key = 'test:basic:get';
      const value = { message: 'Hello World' };

      await cacheService.set(key, value, 60);
      const result = await cacheService.get(key);

      expect(result).toEqual(value);
    });

    test('should return null for non-existent key', async () => {
      const result = await cacheService.get('non:existent:key');
      expect(result).toBeNull();
    });

    test('should delete a key', async () => {
      const key = 'test:basic:delete';
      const value = { data: 'to delete' };

      await cacheService.set(key, value, 60);
      await cacheService.del(key);
      const result = await cacheService.get(key);

      expect(result).toBeNull();
    });

    test('should handle multiple set and get operations', async () => {
      const items = {
        'test:multi:1': { id: 1, name: 'Item 1' },
        'test:multi:2': { id: 2, name: 'Item 2' },
        'test:multi:3': { id: 3, name: 'Item 3' }
      };

      await cacheService.setMulti(items, 60);
      const results = await cacheService.getMulti(Object.keys(items));

      expect(results).toHaveLength(3);
      expect(results[0]).toEqual(items['test:multi:1']);
      expect(results[1]).toEqual(items['test:multi:2']);
      expect(results[2]).toEqual(items['test:multi:3']);
    });
  });

  describe('Session Caching', () => {
    test('should set and get session', async () => {
      const sessionToken = 'test-session-token-123';
      const sessionData = {
        id: 'session-123',
        userId: 'user-456',
        email: 'test@example.com',
        role: 'user'
      };

      await cacheService.setSession(sessionToken, sessionData);
      const cached = await cacheService.getSession(sessionToken);

      expect(cached).toEqual(sessionData);
    });

    test('should invalidate session', async () => {
      const sessionToken = 'test-session-token-456';
      const sessionData = { id: 'session-456', userId: 'user-789' };

      await cacheService.setSession(sessionToken, sessionData);
      await cacheService.invalidateSession(sessionToken);
      const cached = await cacheService.getSession(sessionToken);

      expect(cached).toBeNull();
    });
  });

  describe('User Caching', () => {
    test('should cache user data', async () => {
      const userId = 'user-123';
      const userData = {
        id: userId,
        email: 'user@example.com',
        name: 'Test User',
        role: 'user'
      };

      await cacheService.setCachedUser(userId, userData);
      const cached = await cacheService.getCachedUser(userId);

      expect(cached).toEqual(userData);
    });

    test('should invalidate user cache', async () => {
      const userId = 'user-456';
      const userData = { id: userId, email: 'user2@example.com' };

      await cacheService.setCachedUser(userId, userData);
      await cacheService.invalidateUserCache(userId);
      const cached = await cacheService.getCachedUser(userId);

      expect(cached).toBeNull();
    });
  });

  describe('API Response Caching', () => {
    test('should cache API response', async () => {
      const endpoint = '/api/users';
      const responseData = { users: [], count: 0 };

      await cacheService.setCachedApiResponse(endpoint, responseData, true);
      const cached = await cacheService.getCachedApiResponse(endpoint);

      expect(cached).toEqual(responseData);
    });

    test('should cache user-specific API response', async () => {
      const endpoint = '/api/profile';
      const userId = 'user-789';
      const userData = { name: 'John', email: 'john@example.com' };

      await cacheService.setCachedApiResponse(`${endpoint}:user:${userId}`, userData);
      const cached = await cacheService.getCachedApiResponse(`${endpoint}:user:${userId}`);

      expect(cached).toEqual(userData);
    });
  });

  describe('Pattern Invalidation', () => {
    test('should invalidate pattern', async () => {
      const baseKey = 'test:pattern:invalidate';

      await cacheService.set(`${baseKey}:1`, { data: 1 }, 60);
      await cacheService.set(`${baseKey}:2`, { data: 2 }, 60);
      await cacheService.set(`${baseKey}:3`, { data: 3 }, 60);

      await cacheService.invalidatePattern(`${baseKey}:*`);

      const results = await Promise.all([
        cacheService.get(`${baseKey}:1`),
        cacheService.get(`${baseKey}:2`),
        cacheService.get(`${baseKey}:3`)
      ]);

      expect(results.every(r => r === null)).toBe(true);
    });
  });

  describe('Statistics', () => {
    test('should track cache hits and misses', async () => {
      const key = 'test:stats:track';

      await cacheService.set(key, { data: 'value' }, 60);
      await cacheService.get(key);
      await cacheService.get('test:stats:non-existent');

      const stats = cacheService.getStats();

      expect(stats.overall.hits).toBeGreaterThan(0);
      expect(stats.overall.misses).toBeGreaterThan(0);
    });

    test('should track session cache stats', async () => {
      const sessionToken = 'test-session-stats';
      const sessionData = { id: 'session-stats' };

      await cacheService.setSession(sessionToken, sessionData);
      await cacheService.getSession(sessionToken);
      await cacheService.getSession('non-existent-session');

      const stats = cacheService.getStats();

      expect(stats.session.hits).toBeGreaterThan(0);
      expect(stats.session.misses).toBeGreaterThan(0);
    });

    test('should reset statistics', async () => {
      await cacheService.set('test:reset:1', { data: 1 }, 60);
      await cacheService.get('test:reset:1');

      cacheService.resetStats();
      const stats = cacheService.getStats();

      expect(stats.overall.hits).toBe(0);
      expect(stats.overall.misses).toBe(0);
    });
  });

  describe('Health Check', () => {
    test('should report healthy status', async () => {
      const isHealthy = await cacheService.isHealthy();
      expect(typeof isHealthy).toBe('boolean');
    });
  });

  describe('Cache Key Generation', () => {
    test('should generate session key', () => {
      const sessionToken = 'abc123';
      const key = cacheService.generateSessionKey(sessionToken);

      expect(key).toBe('session:abc123');
    });

    test('should generate user key', () => {
      const userId = 'user-123';
      const key = cacheService.generateUserKey(userId);

      expect(key).toBe('user:user-123');
    });

    test('should generate API key with user', () => {
      const endpoint = '/api/profile';
      const userId = 'user-123';
      const key = cacheService.generateApiKey(endpoint, userId);

      expect(key).toContain('api:/api/profile:user:user-123');
    });

    test('should generate API key with params', () => {
      const endpoint = '/api/users';
      const params = { page: 1, limit: 10 };
      const key = cacheService.generateApiKey(endpoint, null, params);

      expect(key).toContain('api:/api/users');
      expect(key).toContain('limit=10');
      expect(key).toContain('page=1');
    });
  });
});
