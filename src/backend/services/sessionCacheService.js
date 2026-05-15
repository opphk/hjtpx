const redisClient = require('../../../config/redis/client');

const SESSION_PREFIX = 'session:';
const SESSION_TTL = {
  DEFAULT: 604800,
  SHORT: 3600,
  MEDIUM: 18000,
  LONG: 86400
};

const CLEANUP_INTERVAL = 60000;

class SessionCacheService {
  constructor() {
    this.localCache = new Map();
    this.localCacheTTL = 5000;
    this.autoCleanupInterval = null;
    this.isRedisConnected = false;

    this.stats = {
      hits: 0,
      misses: 0,
      sets: 0,
      gets: 0,
      invalidations: 0,
      refreshes: 0,
      expiredSessions: 0,
      totalSets: 0,
      totalGets: 0,
      totalInvalidations: 0,
      errors: 0
    };

    this.initRedisConnection();
    this.startAutoCleanup();
  }

  async initRedisConnection() {
    try {
      if (redisClient && !redisClient.isOpen) {
        await redisClient.connect();
      }
      this.isRedisConnected = true;

      if (redisClient) {
        redisClient.on('error', () => {
          this.isRedisConnected = false;
        });
        redisClient.on('connect', () => {
          this.isRedisConnected = true;
        });
      }
    } catch (error) {
      console.error('Session cache Redis connection error:', error);
      this.isRedisConnected = false;
    }
  }

  generateSessionKey(sessionToken) {
    return `${SESSION_PREFIX}${sessionToken}`;
  }

  async storeSession(sessionToken, sessionData, ttl = SESSION_TTL.DEFAULT) {
    if (!sessionData) {
      throw new Error('Session data cannot be null or undefined');
    }

    const key = this.generateSessionKey(sessionToken);

    try {
      const sessionEntry = {
        data: sessionData,
        ttl: ttl,
        createdAt: Date.now(),
        expiresAt: Date.now() + (ttl * 1000),
        lastAccessed: Date.now()
      };

      this.localCache.set(key, sessionEntry);
      setTimeout(() => this.localCache.delete(key), this.localCacheTTL);

      if (this.isRedisConnected && redisClient) {
        await redisClient.setEx(key, ttl, JSON.stringify(sessionEntry));
      }

      this.stats.sets++;
      this.stats.totalSets++;
      return true;
    } catch (error) {
      this.stats.errors++;
      console.error('Session store error:', error);
      return false;
    }
  }

  async getSession(sessionToken) {
    const key = this.generateSessionKey(sessionToken);

    try {
      if (this.localCache.has(key)) {
        const entry = this.localCache.get(key);
        if (Date.now() < entry.expiresAt) {
          entry.lastAccessed = Date.now();
          this.stats.hits++;
          this.stats.gets++;
          this.stats.totalGets++;
          return entry.data;
        } else {
          this.localCache.delete(key);
        }
      }

      if (this.isRedisConnected && redisClient) {
        const cached = await redisClient.get(key);
        if (cached) {
          const entry = JSON.parse(cached);
          if (Date.now() < entry.expiresAt) {
            this.localCache.set(key, entry);
            setTimeout(() => this.localCache.delete(key), this.localCacheTTL);
            this.stats.hits++;
            this.stats.gets++;
            this.stats.totalGets++;
            return entry.data;
          } else {
            this.stats.expiredSessions++;
            await this.invalidateSession(sessionToken);
          }
        }
      }

      this.stats.misses++;
      this.stats.gets++;
      this.stats.totalGets++;
      return null;
    } catch (error) {
      this.stats.errors++;
      this.stats.misses++;
      console.error('Session get error:', error);
      return null;
    }
  }

  async getSessionWithMetadata(sessionToken) {
    const key = this.generateSessionKey(sessionToken);

    try {
      if (this.localCache.has(key)) {
        const entry = this.localCache.get(key);
        if (Date.now() < entry.expiresAt) {
          entry.lastAccessed = Date.now();
          this.stats.hits++;
          this.stats.gets++;
          this.stats.totalGets++;
          return {
            data: entry.data,
            ttl: Math.ceil((entry.expiresAt - Date.now()) / 1000),
            createdAt: entry.createdAt,
            lastAccessed: entry.lastAccessed,
            fromCache: 'local'
          };
        }
      }

      if (this.isRedisConnected && redisClient) {
        const cached = await redisClient.get(key);
        if (cached) {
          const entry = JSON.parse(cached);
          if (Date.now() < entry.expiresAt) {
            this.localCache.set(key, entry);
            setTimeout(() => this.localCache.delete(key), this.localCacheTTL);
            this.stats.hits++;
            this.stats.gets++;
            this.stats.totalGets++;
            return {
              data: entry.data,
              ttl: Math.ceil((entry.expiresAt - Date.now()) / 1000),
              createdAt: entry.createdAt,
              lastAccessed: entry.lastAccessed,
              fromCache: 'redis'
            };
          } else {
            this.stats.expiredSessions++;
            await this.invalidateSession(sessionToken);
          }
        }
      }

      this.stats.misses++;
      this.stats.gets++;
      this.stats.totalGets++;
      return null;
    } catch (error) {
      this.stats.errors++;
      this.stats.misses++;
      console.error('Session get metadata error:', error);
      return null;
    }
  }

  async getRemainingTTL(sessionToken) {
    const key = this.generateSessionKey(sessionToken);

    try {
      if (this.localCache.has(key)) {
        const entry = this.localCache.get(key);
        const remaining = Math.ceil((entry.expiresAt - Date.now()) / 1000);
        return remaining > 0 ? remaining : -1;
      }

      if (this.isRedisConnected && redisClient) {
        const ttl = await redisClient.ttl(key);
        return ttl > 0 ? ttl : -1;
      }

      return -1;
    } catch (error) {
      this.stats.errors++;
      console.error('Get remaining TTL error:', error);
      return -1;
    }
  }

  async extendSession(sessionToken, newTTL) {
    const key = this.generateSessionKey(sessionToken);

    try {
      if (this.localCache.has(key)) {
        const entry = this.localCache.get(key);
        if (Date.now() < entry.expiresAt) {
          entry.ttl = newTTL;
          entry.expiresAt = Date.now() + (newTTL * 1000);

          if (this.isRedisConnected && redisClient) {
            await redisClient.expire(key, newTTL);
          }
          return true;
        }
      }

      if (this.isRedisConnected && redisClient) {
        const exists = await redisClient.get(key);
        if (exists) {
          await redisClient.expire(key, newTTL);
          return true;
        }
      }

      return false;
    } catch (error) {
      this.stats.errors++;
      console.error('Extend session error:', error);
      return false;
    }
  }

  async refreshSession(sessionToken) {
    const metadata = await this.getSessionWithMetadata(sessionToken);
    if (!metadata) {
      return false;
    }

    return await this.extendSession(sessionToken, metadata.ttl);
  }

  async invalidateSession(sessionToken) {
    const key = this.generateSessionKey(sessionToken);

    try {
      this.localCache.delete(key);

      if (this.isRedisConnected && redisClient) {
        await redisClient.del(key);
      }

      this.stats.invalidations++;
      this.stats.totalInvalidations++;
      return true;
    } catch (error) {
      this.stats.errors++;
      console.error('Session invalidation error:', error);
      return false;
    }
  }

  async invalidateUserSessions(userId) {
    try {
      let invalidatedCount = 0;

      for (const [key, entry] of this.localCache.entries()) {
        if (entry.data && entry.data.userId === userId) {
          this.localCache.delete(key);
          invalidatedCount++;
        }
      }

      if (this.isRedisConnected && redisClient) {
        let cursor = 0;
        do {
          const result = await redisClient.scan(cursor, {
            MATCH: `${SESSION_PREFIX}*`,
            COUNT: 100
          });
          cursor = result.cursor;

          for (const key of result.keys) {
            const cached = await redisClient.get(key);
            if (cached) {
              const entry = JSON.parse(cached);
              if (entry.data && entry.data.userId === userId) {
                await redisClient.del(key);
                invalidatedCount++;
              }
            }
          }
        } while (cursor !== 0);
      }

      this.stats.invalidations += invalidatedCount;
      this.stats.totalInvalidations += invalidatedCount;
      return invalidatedCount;
    } catch (error) {
      this.stats.errors++;
      console.error('Invalidate user sessions error:', error);
      return 0;
    }
  }

  async invalidateByPattern(pattern) {
    try {
      let invalidatedCount = 0;
      const fullPattern = pattern.includes(SESSION_PREFIX) ? pattern : `${SESSION_PREFIX}${pattern}`;

      for (const key of this.localCache.keys()) {
        if (this.matchPattern(key, fullPattern)) {
          this.localCache.delete(key);
          invalidatedCount++;
        }
      }

      if (this.isRedisConnected && redisClient) {
        let cursor = 0;
        do {
          const result = await redisClient.scan(cursor, {
            MATCH: fullPattern,
            COUNT: 100
          });
          cursor = result.cursor;

          if (result.keys.length > 0) {
            await redisClient.del(result.keys);
            invalidatedCount += result.keys.length;
          }
        } while (cursor !== 0);
      }

      this.stats.invalidations += invalidatedCount;
      this.stats.totalInvalidations += invalidatedCount;
      return invalidatedCount;
    } catch (error) {
      this.stats.errors++;
      console.error('Invalidate by pattern error:', error);
      return 0;
    }
  }

  matchPattern(key, pattern) {
    const regexPattern = pattern.replace(/\*/g, '.*').replace(/\?/g, '.');
    return new RegExp(`^${regexPattern}$`).test(key);
  }

  async validateSession(sessionToken) {
    const session = await this.getSession(sessionToken);
    return session !== null;
  }

  async getMultipleSessions(sessionTokens) {
    const results = await Promise.all(
      sessionTokens.map(token => this.getSession(token))
    );
    return results;
  }

  async storeMultipleSessions(sessions) {
    await Promise.all(
      sessions.map(s => this.storeSession(s.token, s.data, s.ttl || SESSION_TTL.DEFAULT))
    );
  }

  async getTTLStats() {
    const ttls = [];

    for (const entry of this.localCache.values()) {
      if (Date.now() < entry.expiresAt) {
        ttls.push(entry.ttl);
      }
    }

    if (this.isRedisConnected && redisClient) {
      let cursor = 0;
      do {
        const result = await redisClient.scan(cursor, {
          MATCH: `${SESSION_PREFIX}*`,
          COUNT: 100
        });
        cursor = result.cursor;

        for (const key of result.keys) {
          const ttl = await redisClient.ttl(key);
          if (ttl > 0) {
            ttls.push(ttl);
          }
        }
      } while (cursor !== 0);
    }

    return {
      totalSessions: ttls.length,
      average: ttls.length > 0 ? Math.round(ttls.reduce((a, b) => a + b, 0) / ttls.length) : 0,
      min: ttls.length > 0 ? Math.min(...ttls) : 0,
      max: ttls.length > 0 ? Math.max(...ttls) : 0
    };
  }

  startAutoCleanup() {
    this.autoCleanupInterval = setInterval(async () => {
      await this.cleanupExpiredSessions();
    }, CLEANUP_INTERVAL);
  }

  async cleanupExpiredSessions() {
    try {
      let cleanedCount = 0;

      for (const [key, entry] of this.localCache.entries()) {
        if (Date.now() >= entry.expiresAt) {
          this.localCache.delete(key);
          cleanedCount++;
          this.stats.expiredSessions++;
        }
      }

      if (this.isRedisConnected && redisClient) {
        let cursor = 0;
        do {
          const result = await redisClient.scan(cursor, {
            MATCH: `${SESSION_PREFIX}*`,
            COUNT: 100
          });
          cursor = result.cursor;

          for (const key of result.keys) {
            const ttl = await redisClient.ttl(key);
            if (ttl <= 0) {
              await redisClient.del(key);
              cleanedCount++;
              this.stats.expiredSessions++;
            }
          }
        } while (cursor !== 0);
      }

      return cleanedCount;
    } catch (error) {
      this.stats.errors++;
      console.error('Cleanup expired sessions error:', error);
      return 0;
    }
  }

  getStats() {
    const total = this.stats.hits + this.stats.misses;
    const hitRate = total > 0 ? ((this.stats.hits / total) * 100).toFixed(2) : '0.00';

    return {
      hits: this.stats.hits,
      misses: this.stats.misses,
      hitRate: parseFloat(hitRate),
      sets: this.stats.sets,
      gets: this.stats.gets,
      invalidations: this.stats.invalidations,
      refreshes: this.stats.refreshes,
      expiredSessions: this.stats.expiredSessions,
      errors: this.stats.errors,
      totalSets: this.stats.totalSets,
      totalGets: this.stats.totalGets,
      totalInvalidations: this.stats.totalInvalidations,
      localCacheSize: this.localCache.size,
      redisConnected: this.isRedisConnected
    };
  }

  resetStats() {
    this.stats = {
      hits: 0,
      misses: 0,
      sets: 0,
      gets: 0,
      invalidations: 0,
      refreshes: 0,
      expiredSessions: 0,
      totalSets: 0,
      totalGets: 0,
      totalInvalidations: 0,
      errors: 0
    };
  }

  resetAllStats() {
    this.resetStats();
  }

  async cleanup() {
    if (this.autoCleanupInterval) {
      clearInterval(this.autoCleanupInterval);
      this.autoCleanupInterval = null;
    }
    this.localCache.clear();
  }
}

const sessionCacheService = new SessionCacheService();

module.exports = sessionCacheService;
module.exports.SESSION_TTL = SESSION_TTL;
module.exports.SESSION_PREFIX = SESSION_PREFIX;
