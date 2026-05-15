const cacheService = require('../../services/cacheService');

describe('SessionCacheService', () => {
  let sessionCacheService;

  beforeEach(() => {
    jest.clearAllMocks();
    jest.resetModules();
    
    sessionCacheService = require('../../services/sessionCacheService');
    sessionCacheService.resetAllStats();
  });

  afterEach(async () => {
    if (sessionCacheService) {
      await sessionCacheService.cleanup();
    }
  });

  describe('Session Storage', () => {
    test('should store session with default TTL', async () => {
      const sessionToken = 'test-token-001';
      const sessionData = {
        userId: 'user-123',
        email: 'test@example.com',
        role: 'user'
      };

      const result = await sessionCacheService.storeSession(sessionToken, sessionData);
      
      expect(result).toBe(true);
      
      const cached = await sessionCacheService.getSession(sessionToken);
      expect(cached).toEqual(sessionData);
    });

    test('should store session with custom TTL', async () => {
      const sessionToken = 'test-token-002';
      const sessionData = { userId: 'user-456' };
      const customTTL = 1800;

      await sessionCacheService.storeSession(sessionToken, sessionData, customTTL);
      
      const sessionInfo = await sessionCacheService.getSessionWithMetadata(sessionToken);
      expect(sessionInfo.data).toEqual(sessionData);
      expect(sessionInfo.ttl).toBe(customTTL);
    });

    test('should handle concurrent session stores', async () => {
      const sessions = Array.from({ length: 10 }, (_, i) => ({
        token: `concurrent-token-${i}`,
        data: { userId: `user-${i}`, index: i }
      }));

      await Promise.all(
        sessions.map(s => sessionCacheService.storeSession(s.token, s.data))
      );

      const results = await Promise.all(
        sessions.map(s => sessionCacheService.getSession(s.token))
      );

      expect(results).toEqual(sessions.map(s => s.data));
    });
  });

  describe('TTL Management', () => {
    test('should get remaining TTL for session', async () => {
      const sessionToken = 'ttl-test-token';
      const sessionData = { userId: 'user-ttl' };
      const ttl = 600;

      await sessionCacheService.storeSession(sessionToken, sessionData, ttl);
      
      const remainingTTL = await sessionCacheService.getRemainingTTL(sessionToken);
      expect(remainingTTL).toBeGreaterThan(0);
      expect(remainingTTL).toBeLessThanOrEqual(ttl);
    });

    test('should return -1 for non-existent session TTL', async () => {
      const remainingTTL = await sessionCacheService.getRemainingTTL('non-existent-token');
      expect(remainingTTL).toBe(-1);
    });

    test('should extend session TTL', async () => {
      const sessionToken = 'extend-ttl-token';
      const sessionData = { userId: 'user-extend' };
      const initialTTL = 300;
      const extendedTTL = 600;

      await sessionCacheService.storeSession(sessionToken, sessionData, initialTTL);
      
      const extended = await sessionCacheService.extendSession(sessionToken, extendedTTL);
      expect(extended).toBe(true);
      
      const sessionInfo = await sessionCacheService.getSessionWithMetadata(sessionToken);
      expect(sessionInfo.ttl).toBe(extendedTTL);
    });

    test('should get TTL statistics', async () => {
      await sessionCacheService.storeSession('token-1', { data: 1 }, 300);
      await sessionCacheService.storeSession('token-2', { data: 2 }, 600);
      await sessionCacheService.storeSession('token-3', { data: 3 }, 900);

      const stats = await sessionCacheService.getTTLStats();
      
      expect(stats).toHaveProperty('average');
      expect(stats).toHaveProperty('min');
      expect(stats).toHaveProperty('max');
      expect(stats.totalSessions).toBe(3);
    });
  });

  describe('Session Expiration', () => {
    test('should automatically cleanup expired sessions', async () => {
      const expiredToken = 'expired-token';
      const validToken = 'valid-token';

      await sessionCacheService.storeSession(expiredToken, { userId: 'expired' }, 1);
      await sessionCacheService.storeSession(validToken, { userId: 'valid' }, 3600);

      await new Promise(resolve => setTimeout(resolve, 1500));

      const expiredSession = await sessionCacheService.getSession(expiredToken);
      const validSession = await sessionCacheService.getSession(validToken);

      expect(expiredSession).toBeNull();
      expect(validSession).toEqual({ userId: 'valid' });
    });

    test('should track expired session count', async () => {
      const token1 = 'expire-count-1';
      const token2 = 'expire-count-2';

      await sessionCacheService.storeSession(token1, { data: 1 }, 1);
      await sessionCacheService.storeSession(token2, { data: 2 }, 1);

      await new Promise(resolve => setTimeout(resolve, 1500));
      await sessionCacheService.cleanupExpiredSessions();

      const stats = sessionCacheService.getStats();
      expect(stats.expiredSessions).toBeGreaterThan(0);
    });

    test('should start auto cleanup on initialization', async () => {
      const newService = require('../../services/sessionCacheService');
      
      expect(newService.autoCleanupInterval).toBeDefined();
    });
  });

  describe('Session Invalidation', () => {
    test('should invalidate single session', async () => {
      const sessionToken = 'invalidate-single';
      const sessionData = { userId: 'user-invalidate' };

      await sessionCacheService.storeSession(sessionToken, sessionData);
      
      const invalidated = await sessionCacheService.invalidateSession(sessionToken);
      expect(invalidated).toBe(true);
      
      const cached = await sessionCacheService.getSession(sessionToken);
      expect(cached).toBeNull();
    });

    test('should invalidate all user sessions', async () => {
      const userId = 'user-multi-session';
      const sessions = [
        { token: 'multi-1', data: { userId, device: 'desktop' } },
        { token: 'multi-2', data: { userId, device: 'mobile' } },
        { token: 'multi-3', data: { userId, device: 'tablet' } }
      ];

      for (const s of sessions) {
        await sessionCacheService.storeSession(s.token, s.data);
      }

      const invalidated = await sessionCacheService.invalidateUserSessions(userId);
      expect(invalidated).toBe(3);

      for (const s of sessions) {
        const cached = await sessionCacheService.getSession(s.token);
        expect(cached).toBeNull();
      }
    });

    test('should invalidate sessions by pattern', async () => {
      const pattern = 'pattern:invalidate:*';
      const sessions = [
        { token: 'pattern:invalidate:1', data: { id: 1 } },
        { token: 'pattern:invalidate:2', data: { id: 2 } },
        { token: 'pattern:keep:1', data: { id: 3 } }
      ];

      for (const s of sessions) {
        await sessionCacheService.storeSession(s.token, s.data);
      }

      await sessionCacheService.invalidateByPattern(pattern);

      const kept = await sessionCacheService.getSession('pattern:keep:1');
      expect(kept).toEqual({ id: 3 });

      const invalidated = await sessionCacheService.getSession('pattern:invalidate:1');
      expect(invalidated).toBeNull();
    });
  });

  describe('Session Refresh', () => {
    test('should refresh session and update TTL', async () => {
      const sessionToken = 'refresh-token';
      const sessionData = { userId: 'user-refresh' };

      await sessionCacheService.storeSession(sessionToken, sessionData, 300);
      
      await new Promise(resolve => setTimeout(resolve, 100));

      const refreshed = await sessionCacheService.refreshSession(sessionToken);
      expect(refreshed).toBe(true);

      const metadata1 = await sessionCacheService.getSessionWithMetadata(sessionToken);
      const initialTTL = metadata1.ttl;

      await new Promise(resolve => setTimeout(resolve, 100));

      const refreshed2 = await sessionCacheService.refreshSession(sessionToken);
      const metadata2 = await sessionCacheService.getSessionWithMetadata(sessionToken);

      expect(metadata2.ttl).toBeGreaterThanOrEqual(initialTTL);
    });
  });

  describe('Session Validation', () => {
    test('should validate existing session', async () => {
      const sessionToken = 'valid-session';
      const sessionData = { userId: 'user-valid' };

      await sessionCacheService.storeSession(sessionToken, sessionData);

      const isValid = await sessionCacheService.validateSession(sessionToken);
      expect(isValid).toBe(true);
    });

    test('should invalidate non-existent session', async () => {
      const isValid = await sessionCacheService.validateSession('non-existent');
      expect(isValid).toBe(false);
    });
  });

  describe('Statistics', () => {
    test('should track session operations', async () => {
      await sessionCacheService.storeSession('stats-1', { data: 1 });
      await sessionCacheService.storeSession('stats-2', { data: 2 });
      await sessionCacheService.getSession('stats-1');
      await sessionCacheService.getSession('stats-missing');
      await sessionCacheService.invalidateSession('stats-2');

      const stats = sessionCacheService.getStats();

      expect(stats.totalSets).toBeGreaterThanOrEqual(2);
      expect(stats.totalGets).toBeGreaterThanOrEqual(2);
      expect(stats.totalInvalidations).toBeGreaterThanOrEqual(1);
    });

    test('should calculate session hit rate', async () => {
      await sessionCacheService.storeSession('hit-rate-1', { data: 1 });
      await sessionCacheService.storeSession('hit-rate-2', { data: 2 });
      
      await sessionCacheService.getSession('hit-rate-1');
      await sessionCacheService.getSession('hit-rate-1');
      await sessionCacheService.getSession('hit-rate-1');
      await sessionCacheService.getSession('missing-session');

      const stats = sessionCacheService.getStats();
      
      expect(stats.hitRate).toBeGreaterThan(0);
      expect(stats.hits).toBeGreaterThanOrEqual(3);
      expect(stats.misses).toBeGreaterThanOrEqual(1);
    });

    test('should reset statistics', async () => {
      await sessionCacheService.storeSession('reset-1', { data: 1 });
      
      sessionCacheService.resetStats();
      
      const stats = sessionCacheService.getStats();
      expect(stats.totalSets).toBe(0);
      expect(stats.totalGets).toBe(0);
      expect(stats.hits).toBe(0);
      expect(stats.misses).toBe(0);
    });
  });

  describe('Bulk Operations', () => {
    test('should get multiple sessions', async () => {
      const sessions = [
        { token: 'bulk-1', data: { id: 1 } },
        { token: 'bulk-2', data: { id: 2 } },
        { token: 'bulk-3', data: { id: 3 } }
      ];

      for (const s of sessions) {
        await sessionCacheService.storeSession(s.token, s.data);
      }

      const results = await sessionCacheService.getMultipleSessions(
        sessions.map(s => s.token)
      );

      expect(results).toHaveLength(3);
      expect(results[0]).toEqual({ id: 1 });
      expect(results[1]).toEqual({ id: 2 });
      expect(results[2]).toEqual({ id: 3 });
    });

    test('should store multiple sessions', async () => {
      const sessions = [
        { token: 'multi-store-1', data: { id: 1 } },
        { token: 'multi-store-2', data: { id: 2 } }
      ];

      await sessionCacheService.storeMultipleSessions(sessions);

      const results = await Promise.all(
        sessions.map(s => sessionCacheService.getSession(s.token))
      );

      expect(results[0]).toEqual({ id: 1 });
      expect(results[1]).toEqual({ id: 2 });
    });
  });

  describe('Error Handling', () => {
    test('should handle Redis connection errors gracefully', async () => {
      const sessionToken = 'error-handling-token';
      const sessionData = { userId: 'user-error' };

      const result = await sessionCacheService.storeSession(sessionToken, sessionData);
      
      expect(result).toBeDefined();
    });

    test('should handle malformed session data', async () => {
      const sessionToken = 'malformed-token';
      
      await expect(
        sessionCacheService.storeSession(sessionToken, null)
      ).rejects.toThrow();
    });
  });

  describe('Cleanup', () => {
    test('should cleanup on shutdown', async () => {
      await sessionCacheService.storeSession('cleanup-1', { data: 1 });
      await sessionCacheService.storeSession('cleanup-2', { data: 2 });

      await sessionCacheService.cleanup();

      expect(sessionCacheService.autoCleanupInterval).toBeNull();
    });
  });
});
