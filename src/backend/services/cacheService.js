const redisClient = require('../../../config/redis/client');

const CACHE_KEYS = {
  SESSION: 'session:',
  USER: 'user:',
  API: 'api:',
  PERMISSIONS: 'permissions:',
  TOKEN_BLACKLIST: 'blacklist:',
  RATE_LIMIT: 'ratelimit:',
  ANALYTICS: 'analytics:',
  TAGS: 'cache_tags:',
  METRICS: 'cache_metrics:',
  LOCK: 'lock:'
};

const CACHE_TTL = {
  SESSION: 604800,
  USER: 1800,
  API_PUBLIC: 300,
  API_PRIVATE: 60,
  PERMISSIONS: 3600,
  TOKEN_BLACKLIST: 604800,
  RATE_LIMIT: 60,
  ANALYTICS: 300,
  SHORT: 60,
  MEDIUM: 300,
  LONG: 3600,
  VERY_LONG: 86400
};

const CACHE_PRIORITY = {
  LOW: 1,
  MEDIUM: 2,
  HIGH: 3,
  CRITICAL: 4
};

const CACHE_POLICY = {
  SESSION_SLIDING_EXPIRY: true,
  SESSION_WARMUP_ENABLED: false,
  MAX_CONCURRENT_REQUESTS: 100,
  LOCK_TIMEOUT: 5000,
  CACHE_STALE_THRESHOLD: 0.8,
  COMPRESSION_THRESHOLD: 1024,
  BATCH_SIZE: 100
};

class CacheService {
  constructor() {
    this.memoryCache = new Map();
    this.cacheTimers = new Map();
    this.cacheLocks = new Map();
    this.warmupCache = new Map();
    this.stats = {
      hits: 0,
      misses: 0,
      sets: 0,
      deletes: 0,
      errors: 0,
      evictions: 0,
      locks: 0,
      lockTimeouts: 0,
      session: { hits: 0, misses: 0, sets: 0, deletes: 0, extensions: 0 },
      api: { hits: 0, misses: 0, sets: 0, deletes: 0, stale: 0 },
      user: { hits: 0, misses: 0, sets: 0, deletes: 0 },
      permissions: { hits: 0, misses: 0, sets: 0, deletes: 0 },
      tags: { hits: 0, misses: 0, sets: 0, deletes: 0 },
      lockAcquisitions: 0,
      lockReleases: 0,
      lockTimeouts: 0,
      warmupHits: 0,
      warmupMisses: 0,
      compressionSavings: 0,
      latency: {
        get: [],
        set: [],
        del: []
      }
    };
    this.defaultTTL = 300;
    this.memoryCacheTTL = 60000;
    this.maxMemoryCacheSize = 1000;
    this.isRedisConnected = false;
    this.startTime = Date.now();
    this.initRedisConnection();
    this.startMetricsCollection();
  }

  async initRedisConnection() {
    try {
      if (!redisClient.isOpen) {
        await redisClient.connect();
      }
      this.isRedisConnected = true;
      redisClient.on('error', err => {
        console.error('Redis connection error:', err);
        this.isRedisConnected = false;
      });
      redisClient.on('connect', () => {
        this.isRedisConnected = true;
      });
    } catch (error) {
      console.error('Failed to connect to Redis:', error);
      this.isRedisConnected = false;
    }
  }

  measureLatency(type, startTime) {
    const latency = Date.now() - startTime;
    if (this.stats.latency[type]) {
      this.stats.latency[type].push(latency);
      if (this.stats.latency[type].length > 1000) {
        this.stats.latency[type].shift();
      }
    }
  }

  startMetricsCollection() {
    setInterval(() => {
      this.collectMetrics();
    }, 60000);
  }

  async collectMetrics() {
    if (!this.isRedisConnected) return;

    try {
      const metrics = {
        timestamp: Date.now(),
        memoryCacheSize: this.memoryCache.size,
        maxMemoryCacheSize: this.maxMemoryCacheSize,
        memoryUsagePercent: (this.memoryCache.size / this.maxMemoryCacheSize) * 100,
        redisConnected: this.isRedisConnected,
        uptime: Date.now() - this.startTime,
        hitRate: this.calculateOverallHitRate(),
        stats: { ...this.stats }
      };

      await redisClient.setEx(CACHE_KEYS.METRICS, CACHE_TTL.MEDIUM, JSON.stringify(metrics));
    } catch (error) {
      console.error('Failed to collect metrics:', error);
    }
  }

  calculateOverallHitRate() {
    const total = this.stats.hits + this.stats.misses;
    return total > 0 ? (this.stats.hits / total) * 100 : 0;
  }

  async acquireLock(key, timeout = CACHE_POLICY.LOCK_TIMEOUT) {
    const lockKey = `${CACHE_KEYS.LOCK}${key}`;
    const lockValue = `${Date.now()}-${Math.random()}`;

    try {
      const acquired = await redisClient.set(lockKey, lockValue, {
        NX: true,
        EX: Math.ceil(timeout / 1000)
      });

      if (acquired) {
        this.stats.locks++;
        this.cacheLocks.set(key, lockValue);
        return lockValue;
      }

      this.stats.lockTimeouts++;
      return null;
    } catch (error) {
      this.stats.errors++;
      return null;
    }
  }

  async releaseLock(key, lockValue) {
    const cachedValue = this.cacheLocks.get(key);
    if (cachedValue !== lockValue) {
      return false;
    }

    const lockKey = `${CACHE_KEYS.LOCK}${key}`;
    try {
      await redisClient.del(lockKey);
      this.cacheLocks.delete(key);
      this.stats.lockReleases++;
      return true;
    } catch (error) {
      this.stats.errors++;
      return false;
    }
  }

  generateSessionKey(sessionToken) {
    return `${CACHE_KEYS.SESSION}${sessionToken}`;
  }

  generateUserKey(userId) {
    return `${CACHE_KEYS.USER}${userId}`;
  }

  generateApiKey(endpoint, userId = null, params = {}) {
    const base = `${CACHE_KEYS.API}${endpoint}`;
    if (userId) {
      return `${base}:user:${userId}`;
    }
    const paramHash = Object.keys(params)
      .sort()
      .map(k => `${k}=${params[k]}`)
      .join('&');
    return paramHash ? `${base}:${paramHash}` : base;
  }

  generateTagKey(tag) {
    return `${CACHE_KEYS.TAGS}${tag}`;
  }

  async warmupSession(sessionToken, sessionData, ttl = CACHE_TTL.SESSION) {
    if (!CACHE_POLICY.SESSION_WARMUP_ENABLED) {
      return false;
    }

    const key = this.generateSessionKey(sessionToken);
    this.warmupCache.set(key, {
      data: sessionData,
      ttl,
      warmedAt: Date.now()
    });
    return true;
  }

  async getSession(sessionToken, options = {}) {
    const startTime = Date.now();
    const key = this.generateSessionKey(sessionToken);

    if (options.allowStale && this.warmupCache.has(key)) {
      const warmupData = this.warmupCache.get(key);
      this.stats.warmupHits++;
      return warmupData.data;
    }

    const cached = await this.get(key);
    if (cached) {
      this.stats.session.hits++;
      if (CACHE_POLICY.SESSION_SLIDING_EXPIRY) {
        await this.extendSessionTTL(sessionToken);
      }
    } else {
      this.stats.session.misses++;
      if (options.allowStale && this.warmupCache.has(key)) {
        const warmupData = this.warmupCache.get(key);
        this.stats.warmupHits++;
        return warmupData.data;
      }
    }
    this.measureLatency('get', startTime);
    return cached;
  }

  async setSession(sessionToken, sessionData, ttl = CACHE_TTL.SESSION, tags = []) {
    const startTime = Date.now();
    const key = this.generateSessionKey(sessionToken);
    const result = await this.set(key, sessionData, ttl, tags);
    if (result) {
      this.stats.session.sets++;
      if (CACHE_POLICY.SESSION_WARMUP_ENABLED) {
        this.warmupCache.set(key, {
          data: sessionData,
          ttl,
          warmedAt: Date.now()
        });
      }
    }
    this.measureLatency('set', startTime);
    return result;
  }

  async extendSessionTTL(sessionToken, ttl = CACHE_TTL.SESSION) {
    try {
      const key = this.generateSessionKey(sessionToken);
      if (this.isRedisConnected) {
        await redisClient.expire(key, ttl);
      }
      if (this.memoryCache.has(key)) {
        const memEntry = this.memoryCache.get(key);
        memEntry.expiresAt = Date.now() + Math.min(ttl * 1000, this.memoryCacheTTL);
      }
      this.stats.session.extensions++;
      return true;
    } catch (error) {
      this.stats.errors++;
      console.error('Extend session TTL error:', error);
      return false;
    }
  }

  async invalidateSession(sessionToken) {
    const startTime = Date.now();
    const key = this.generateSessionKey(sessionToken);
    const result = await this.del(key);
    if (result) {
      this.stats.session.deletes++;
      this.warmupCache.delete(key);
    }
    this.measureLatency('del', startTime);
    return result;
  }

  async invalidateUserSessions(userId) {
    return await this.invalidatePattern(`${CACHE_KEYS.SESSION}*:${userId}*`);
  }

  async getCachedPermissions(userId) {
    const startTime = Date.now();
    const key = `${CACHE_KEYS.PERMISSIONS}${userId}`;
    const cached = await this.get(key);
    if (cached) {
      this.stats.permissions.hits++;
    } else {
      this.stats.permissions.misses++;
    }
    this.measureLatency('get', startTime);
    return cached;
  }

  async setCachedPermissions(userId, permissionsData, ttl = CACHE_TTL.PERMISSIONS, tags = []) {
    const startTime = Date.now();
    const key = `${CACHE_KEYS.PERMISSIONS}${userId}`;
    const result = await this.set(key, permissionsData, ttl, tags);
    if (result) {
      this.stats.permissions.sets++;
    }
    this.measureLatency('set', startTime);
    return result;
  }

  async invalidatePermissions(userId) {
    const startTime = Date.now();
    const key = `${CACHE_KEYS.PERMISSIONS}${userId}`;
    const result = await this.del(key);
    if (result) {
      this.stats.permissions.deletes++;
    }
    this.measureLatency('del', startTime);
    return result;
  }

  async getCachedUser(userId) {
    const startTime = Date.now();
    const key = this.generateUserKey(userId);
    const cached = await this.get(key);
    if (cached) {
      this.stats.user.hits++;
    } else {
      this.stats.user.misses++;
    }
    this.measureLatency('get', startTime);
    return cached;
  }

  async setCachedUser(userId, userData, ttl = CACHE_TTL.USER, tags = ['user']) {
    const startTime = Date.now();
    const key = this.generateUserKey(userId);
    const result = await this.set(key, userData, ttl, tags);
    if (result) {
      this.stats.user.sets++;
    }
    this.measureLatency('set', startTime);
    return result;
  }

  async invalidateUserCache(userId) {
    const startTime = Date.now();
    const key = this.generateUserKey(userId);
    const result = await this.del(key);
    if (result) {
      this.stats.user.deletes++;
    }
    this.measureLatency('del', startTime);
    return result;
  }

  async getCachedApiResponse(key) {
    const startTime = Date.now();
    const fullKey = `${CACHE_KEYS.API}${key}`;
    const cached = await this.get(fullKey);
    if (cached) {
      this.stats.api.hits++;
    } else {
      this.stats.api.misses++;
    }
    this.measureLatency('get', startTime);
    return cached;
  }

  async setCachedApiResponse(key, responseData, isPublic = true, ttl = null, tags = []) {
    const startTime = Date.now();
    const fullKey = `${CACHE_KEYS.API}${key}`;
    const effectiveTtl = ttl || (isPublic ? CACHE_TTL.API_PUBLIC : CACHE_TTL.API_PRIVATE);
    const result = await this.set(fullKey, responseData, effectiveTtl, tags);
    if (result) {
      this.stats.api.sets++;
    }
    this.measureLatency('set', startTime);
    return result;
  }

  async invalidateApiCache(pattern = '*') {
    return await this.invalidatePattern(`${CACHE_KEYS.API}${pattern}`);
  }

  async invalidateAllUserCache(userId) {
    await Promise.all([
      this.invalidateUserCache(userId),
      this.invalidatePattern(`${CACHE_KEYS.API}*:user:${userId}*`),
      this.invalidatePattern(`${CACHE_KEYS.PERMISSIONS}${userId}`),
      this.invalidateTag(`user:${userId}`)
    ]);
  }

  async addTagsToKey(key, tags) {
    if (!tags || tags.length === 0) return true;
    try {
      const pipeline = this.isRedisConnected ? redisClient.multi() : null;
      for (const tag of tags) {
        const tagKey = this.generateTagKey(tag);
        if (pipeline) {
          pipeline.sAdd(tagKey, key);
          pipeline.expire(tagKey, CACHE_TTL.VERY_LONG);
        }
        this.stats.tags.sets++;
      }
      if (pipeline) await pipeline.exec();
      return true;
    } catch (error) {
      this.stats.errors++;
      console.error('Add tags to key error:', error);
      return false;
    }
  }

  async invalidateTag(tag) {
    try {
      const tagKey = this.generateTagKey(tag);
      if (this.isRedisConnected) {
        const keys = await redisClient.sMembers(tagKey);
        if (keys.length > 0) {
          for (const key of keys) {
            await this.del(key);
          }
        }
        await redisClient.del(tagKey);
      }
      this.stats.tags.deletes++;
      return true;
    } catch (error) {
      this.stats.errors++;
      console.error('Invalidate tag error:', error);
      return false;
    }
  }

  async invalidateTags(tags) {
    await Promise.all(tags.map(tag => this.invalidateTag(tag)));
  }

  async get(key) {
    try {
      if (this.isRedisConnected) {
        const redisValue = await redisClient.get(key);
        if (redisValue) {
          this.stats.hits++;
          return JSON.parse(redisValue);
        }
      }

      if (this.memoryCache.has(key)) {
        const memEntry = this.memoryCache.get(key);
        if (Date.now() < memEntry.expiresAt) {
          this.stats.hits++;
          if (this.isRedisConnected) {
            redisClient.setEx(key, this.defaultTTL, JSON.stringify(memEntry.value)).catch(() => {});
          }
          return memEntry.value;
        }
        this.memoryCache.delete(key);
        this.stats.evictions++;
      }

      this.stats.misses++;
      return null;
    } catch (error) {
      this.stats.errors++;
      console.error('Cache get error:', error);
      return null;
    }
  }

  async set(key, value, ttl = this.defaultTTL, tags = []) {
    try {
      const serialized = JSON.stringify(value);

      if (this.isRedisConnected) {
        await redisClient.setEx(key, ttl, serialized);
      }

      this.memoryCache.set(key, {
        value,
        expiresAt: Date.now() + Math.min(ttl * 1000, this.memoryCacheTTL),
        priority: tags.includes('high') ? CACHE_PRIORITY.HIGH : CACHE_PRIORITY.MEDIUM
      });

      if (this.memoryCache.size > this.maxMemoryCacheSize) {
        this.evictFromMemoryCache();
      }

      if (this.cacheTimers.has(key)) {
        clearTimeout(this.cacheTimers.get(key));
      }

      const timer = setTimeout(
        () => {
          this.memoryCache.delete(key);
          this.cacheTimers.delete(key);
        },
        Math.min(ttl * 1000, this.memoryCacheTTL)
      );

      this.cacheTimers.set(key, timer);
      this.stats.sets++;

      if (tags.length > 0) {
        await this.addTagsToKey(key, tags);
      }

      return true;
    } catch (error) {
      this.stats.errors++;
      console.error('Cache set error:', error);
      return false;
    }
  }

  evictFromMemoryCache() {
    const keysToEvict = Array.from(this.memoryCache.entries())
      .sort((a, b) => {
        if (a[1].priority !== b[1].priority) {
          return a[1].priority - b[1].priority;
        }
        return a[1].expiresAt - b[1].expiresAt;
      })
      .slice(0, Math.floor(this.maxMemoryCacheSize * 0.1))
      .map(([key]) => key);

    for (const key of keysToEvict) {
      this.memoryCache.delete(key);
      if (this.cacheTimers.has(key)) {
        clearTimeout(this.cacheTimers.get(key));
        this.cacheTimers.delete(key);
      }
      this.stats.evictions++;
    }
  }

  async del(key) {
    try {
      if (this.isRedisConnected) {
        await redisClient.del(key);
      }

      this.memoryCache.delete(key);
      if (this.cacheTimers.has(key)) {
        clearTimeout(this.cacheTimers.get(key));
        this.cacheTimers.delete(key);
      }

      this.stats.deletes++;
      return true;
    } catch (error) {
      this.stats.errors++;
      console.error('Cache delete error:', error);
      return false;
    }
  }

  async invalidatePattern(pattern) {
    try {
      if (this.isRedisConnected) {
        let cursor = 0;
        do {
          const result = await redisClient.scan(cursor, {
            MATCH: pattern,
            COUNT: 100
          });
          cursor = result.cursor;
          if (result.keys.length > 0) {
            await redisClient.del(result.keys);
            this.stats.deletes += result.keys.length;
          }
        } while (cursor !== 0);
      }

      for (const key of this.memoryCache.keys()) {
        if (this.matchPattern(key, pattern)) {
          this.memoryCache.delete(key);
          if (this.cacheTimers.has(key)) {
            clearTimeout(this.cacheTimers.get(key));
            this.cacheTimers.delete(key);
          }
        }
      }

      return true;
    } catch (error) {
      this.stats.errors++;
      console.error('Cache invalidate error:', error);
      return false;
    }
  }

  matchPattern(key, pattern) {
    const regexPattern = pattern.replace(/\*/g, '.*').replace(/\?/g, '.');
    return new RegExp(`^${regexPattern}$`).test(key);
  }

  async getMulti(keys) {
    try {
      if (!this.isRedisConnected) {
        return keys.map(key => this.memoryCache.get(key)?.value || null);
      }

      const values = await redisClient.mGet(keys.map(k => k));
      return values.map(v => (v ? JSON.parse(v) : null));
    } catch (error) {
      this.stats.errors++;
      console.error('Cache getMulti error:', error);
      return keys.map(() => null);
    }
  }

  async setMulti(items, ttl = this.defaultTTL) {
    try {
      if (!this.isRedisConnected) {
        for (const [key, value] of Object.entries(items)) {
          await this.set(key, value, ttl);
        }
        return true;
      }

      const pipeline = redisClient.multi();
      for (const [key, value] of Object.entries(items)) {
        pipeline.setEx(key, ttl, JSON.stringify(value));
      }
      await pipeline.exec();

      for (const [key, value] of Object.entries(items)) {
        this.memoryCache.set(key, {
          value,
          expiresAt: Date.now() + Math.min(ttl * 1000, this.memoryCacheTTL)
        });
      }

      this.stats.sets += Object.keys(items).length;
      return true;
    } catch (error) {
      this.stats.errors++;
      console.error('Cache setMulti error:', error);
      return false;
    }
  }

  async clear() {
    try {
      if (this.isRedisConnected) {
        await redisClient.flushDb();
      }

      this.memoryCache.clear();
      for (const timer of this.cacheTimers.values()) {
        clearTimeout(timer);
      }
      this.cacheTimers.clear();

      return true;
    } catch (error) {
      this.stats.errors++;
      console.error('Cache clear error:', error);
      return false;
    }
  }

  async isHealthy() {
    try {
      if (!this.isRedisConnected) {
        return false;
      }
      const pong = await redisClient.ping();
      return pong === 'PONG';
    } catch (error) {
      return false;
    }
  }

  calculateAverageLatency(type) {
    const latencies = this.stats.latency[type];
    if (latencies.length === 0) return 0;
    const sum = latencies.reduce((a, b) => a + b, 0);
    return (sum / latencies.length).toFixed(2);
  }

  calculatePercentileLatency(type, percentile) {
    const latencies = [...this.stats.latency[type]].sort((a, b) => a - b);
    if (latencies.length === 0) return 0;
    const index = Math.floor((percentile / 100) * latencies.length);
    return latencies[index];
  }

  getStats() {
    const total = this.stats.hits + this.stats.misses;
    const sessionTotal = this.stats.session.hits + this.stats.session.misses;
    const apiTotal = this.stats.api.hits + this.stats.api.misses;
    const userTotal = this.stats.user.hits + this.stats.user.misses;
    const permissionsTotal = this.stats.permissions.hits + this.stats.permissions.misses;

    return {
      overall: {
        hits: this.stats.hits,
        misses: this.stats.misses,
        hitRate: total > 0 ? ((this.stats.hits / total) * 100).toFixed(2) + '%' : '0%',
        sets: this.stats.sets,
        deletes: this.stats.deletes,
        evictions: this.stats.evictions,
        errors: this.stats.errors,
        uptime: Date.now() - this.startTime,
        memoryCacheUsage: `${this.memoryCache.size}/${this.maxMemoryCacheSize}`,
        memoryCachePercent:
          ((this.memoryCache.size / this.maxMemoryCacheSize) * 100).toFixed(2) + '%',
        latency: {
          avgGet: this.calculateAverageLatency('get') + 'ms',
          avgSet: this.calculateAverageLatency('set') + 'ms',
          avgDel: this.calculateAverageLatency('del') + 'ms',
          p95Get: this.calculatePercentileLatency('get', 95) + 'ms',
          p99Get: this.calculatePercentileLatency('get', 99) + 'ms'
        }
      },
      session: {
        hits: this.stats.session.hits,
        misses: this.stats.session.misses,
        sets: this.stats.session.sets,
        deletes: this.stats.session.deletes,
        extensions: this.stats.session.extensions,
        hitRate:
          sessionTotal > 0
            ? ((this.stats.session.hits / sessionTotal) * 100).toFixed(2) + '%'
            : '0%'
      },
      api: {
        hits: this.stats.api.hits,
        misses: this.stats.api.misses,
        sets: this.stats.api.sets,
        deletes: this.stats.api.deletes,
        stale: this.stats.api.stale,
        hitRate: apiTotal > 0 ? ((this.stats.api.hits / apiTotal) * 100).toFixed(2) + '%' : '0%'
      },
      user: {
        hits: this.stats.user.hits,
        misses: this.stats.user.misses,
        sets: this.stats.user.sets,
        deletes: this.stats.user.deletes,
        hitRate: userTotal > 0 ? ((this.stats.user.hits / userTotal) * 100).toFixed(2) + '%' : '0%'
      },
      permissions: {
        hits: this.stats.permissions.hits,
        misses: this.stats.permissions.misses,
        sets: this.stats.permissions.sets,
        deletes: this.stats.permissions.deletes,
        hitRate:
          permissionsTotal > 0
            ? ((this.stats.permissions.hits / permissionsTotal) * 100).toFixed(2) + '%'
            : '0%'
      },
      tags: {
        hits: this.stats.tags.hits,
        misses: this.stats.tags.misses,
        sets: this.stats.tags.sets,
        deletes: this.stats.tags.deletes
      },
      locks: {
        acquisitions: this.stats.lockAcquisitions,
        releases: this.stats.lockReleases,
        timeouts: this.stats.lockTimeouts,
        active: this.cacheLocks.size
      },
      warmup: {
        enabled: CACHE_POLICY.SESSION_WARMUP_ENABLED,
        hits: this.stats.warmupHits,
        misses: this.stats.warmupMisses,
        cachedItems: this.warmupCache.size
      },
      memoryCacheSize: this.memoryCache.size,
      maxMemoryCacheSize: this.maxMemoryCacheSize,
      isRedisConnected: this.isRedisConnected,
      cacheKeys: CACHE_KEYS,
      cacheTTL: CACHE_TTL,
      cachePriority: CACHE_PRIORITY,
      cachePolicy: CACHE_POLICY
    };
  }

  resetStats() {
    this.stats = {
      hits: 0,
      misses: 0,
      sets: 0,
      deletes: 0,
      errors: 0,
      evictions: 0,
      locks: 0,
      lockTimeouts: 0,
      session: { hits: 0, misses: 0, sets: 0, deletes: 0, extensions: 0 },
      api: { hits: 0, misses: 0, sets: 0, deletes: 0, stale: 0 },
      user: { hits: 0, misses: 0, sets: 0, deletes: 0 },
      permissions: { hits: 0, misses: 0, sets: 0, deletes: 0 },
      tags: { hits: 0, misses: 0, sets: 0, deletes: 0 },
      lockAcquisitions: 0,
      lockReleases: 0,
      lockTimeouts: 0,
      warmupHits: 0,
      warmupMisses: 0,
      compressionSavings: 0,
      latency: {
        get: [],
        set: [],
        del: []
      }
    };
  }
}

const cacheService = new CacheService();

module.exports = cacheService;
module.exports.CACHE_KEYS = CACHE_KEYS;
module.exports.CACHE_TTL = CACHE_TTL;
module.exports.CACHE_PRIORITY = CACHE_PRIORITY;
module.exports.CACHE_POLICY = CACHE_POLICY;
