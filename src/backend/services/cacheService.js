const redisClient = require('../../../config/redis/client');

const CACHE_KEYS = {
  SESSION: 'session:',
  USER: 'user:',
  API: 'api:',
  PERMISSIONS: 'permissions:',
  TOKEN_BLACKLIST: 'blacklist:',
  RATE_LIMIT: 'ratelimit:',
  ANALYTICS: 'analytics:'
};

const CACHE_TTL = {
  SESSION: 604800,
  USER: 1800,
  API_PUBLIC: 300,
  API_PRIVATE: 60,
  PERMISSIONS: 3600,
  TOKEN_BLACKLIST: 604800,
  RATE_LIMIT: 60,
  ANALYTICS: 300
};

const CACHE_STRATEGY = {
  SESSION: {
    strategy: 'write-through',
    ttl: CACHE_TTL.SESSION,
    maxSize: 10000,
    compression: true
  },
  USER: {
    strategy: 'cache-aside',
    ttl: CACHE_TTL.USER,
    maxSize: 5000,
    compression: false
  },
  API_PUBLIC: {
    strategy: 'cache-aside',
    ttl: CACHE_TTL.API_PUBLIC,
    maxSize: 2000,
    compression: true
  },
  API_PRIVATE: {
    strategy: 'cache-aside',
    ttl: CACHE_TTL.API_PRIVATE,
    maxSize: 1000,
    compression: false
  }
};

class CacheMetrics {
  constructor() {
    this.startTime = Date.now();
    this.metrics = {
      hits: { total: 0, byType: {} },
      misses: { total: 0, byType: {} },
      sets: { total: 0, byType: {} },
      deletes: { total: 0, byType: {} },
      errors: { total: 0, byType: {} },
      latency: { total: 0, count: 0, p50: 0, p95: 0, p99: 0 },
      memoryUsage: { used: 0, peak: 0 },
      hitRate: { session: '0%', api: '0%', user: '0%', overall: '0%' }
    };
    this.latencies = [];
  }

  recordHit(type = 'general') {
    this.metrics.hits.total++;
    this.metrics.hits.byType[type] = (this.metrics.hits.byType[type] || 0) + 1;
    this.updateHitRate(type);
  }

  recordMiss(type = 'general') {
    this.metrics.misses.total++;
    this.metrics.misses.byType[type] = (this.metrics.misses.byType[type] || 0) + 1;
    this.updateHitRate(type);
  }

  recordSet(type = 'general') {
    this.metrics.sets.total++;
    this.metrics.sets.byType[type] = (this.metrics.sets.byType[type] || 0) + 1;
  }

  recordDelete(type = 'general') {
    this.metrics.deletes.total++;
    this.metrics.deletes.byType[type] = (this.metrics.deletes.byType[type] || 0) + 1;
  }

  recordError(type = 'general', error) {
    this.metrics.errors.total++;
    this.metrics.errors.byType[type] = (this.metrics.errors.byType[type] || 0) + 1;
    console.error(`Cache error [${type}]:`, error);
  }

  recordLatency(duration) {
    this.latencies.push(duration);
    this.metrics.latency.total += duration;
    this.metrics.latency.count++;
    
    if (this.latencies.length > 1000) {
      this.latencies = this.latencies.slice(-1000);
    }
    
    this.calculatePercentiles();
  }

  updateHitRate(type) {
    const typeMap = {
      'session': 'session',
      'api': 'api',
      'user': 'user'
    };
    
    const mappedType = typeMap[type] || 'overall';
    const hits = this.metrics.hits.byType[type] || 0;
    const misses = this.metrics.misses.byType[type] || 0;
    const total = hits + misses;
    
    this.metrics.hitRate[mappedType] = total > 0 
      ? ((hits / total) * 100).toFixed(2) + '%' 
      : '0%';
  }

  calculatePercentiles() {
    if (this.latencies.length === 0) return;
    
    const sorted = [...this.latencies].sort((a, b) => a - b);
    const p50Index = Math.floor(sorted.length * 0.5);
    const p95Index = Math.floor(sorted.length * 0.95);
    const p99Index = Math.floor(sorted.length * 0.99);
    
    this.metrics.latency.p50 = sorted[p50Index] || 0;
    this.metrics.latency.p95 = sorted[p95Index] || 0;
    this.metrics.latency.p99 = sorted[p99Index] || 0;
  }

  updateMemoryUsage(used) {
    this.metrics.memoryUsage.used = used;
    if (used > this.metrics.memoryUsage.peak) {
      this.metrics.memoryUsage.peak = used;
    }
  }

  getMetrics() {
    const totalHits = this.metrics.hits.total;
    const totalMisses = this.metrics.misses.total;
    const total = totalHits + totalMisses;
    
    this.metrics.hitRate.overall = total > 0 
      ? ((totalHits / total) * 100).toFixed(2) + '%' 
      : '0%';
    
    this.metrics.latency.avg = this.metrics.latency.count > 0
      ? Math.round(this.metrics.latency.total / this.metrics.latency.count)
      : 0;
    
    return {
      ...this.metrics,
      uptime: Date.now() - this.startTime,
      uptimeFormatted: this.formatUptime()
    };
  }

  formatUptime() {
    const uptime = Date.now() - this.startTime;
    const hours = Math.floor(uptime / 3600000);
    const minutes = Math.floor((uptime % 3600000) / 60000);
    const seconds = Math.floor((uptime % 60000) / 1000);
    return `${hours}h ${minutes}m ${seconds}s`;
  }

  reset() {
    this.startTime = Date.now();
    this.latencies = [];
    this.metrics = {
      hits: { total: 0, byType: {} },
      misses: { total: 0, byType: {} },
      sets: { total: 0, byType: {} },
      deletes: { total: 0, byType: {} },
      errors: { total: 0, byType: {} },
      latency: { total: 0, count: 0, p50: 0, p95: 0, p99: 0 },
      memoryUsage: { used: 0, peak: 0 },
      hitRate: { session: '0%', api: '0%', user: '0%', overall: '0%' }
    };
  }
}

class CacheInvalidationStrategy {
  constructor(cacheService) {
    this.cacheService = cacheService;
    this.invalidationQueue = [];
    this.isProcessing = false;
  }

  async queueInvalidation(pattern, reason, metadata = {}) {
    this.invalidationQueue.push({
      pattern,
      reason,
      metadata,
      queuedAt: Date.now(),
      priority: metadata.priority || 'normal'
    });
    
    if (!this.isProcessing) {
      this.processQueue();
    }
  }

  async processQueue() {
    if (this.isProcessing || this.invalidationQueue.length === 0) {
      return;
    }

    this.isProcessing = true;

    while (this.invalidationQueue.length > 0) {
      const item = this.invalidationQueue.shift();
      
      try {
        await this.cacheService.invalidatePattern(item.pattern);
        console.log(`Invalidated cache: ${item.pattern} (reason: ${item.reason})`);
      } catch (error) {
        console.error(`Failed to invalidate ${item.pattern}:`, error);
        this.invalidationQueue.unshift(item);
        break;
      }
    }

    this.isProcessing = false;
  }

  async invalidateUserRelated(userId) {
    await Promise.all([
      this.cacheService.invalidateUserCache(userId),
      this.cacheService.invalidatePattern(`${CACHE_KEYS.SESSION}*${userId}*`),
      this.cacheService.invalidatePattern(`${CACHE_KEYS.API}*:user:${userId}*`),
      this.cacheService.invalidatePattern(`${CACHE_KEYS.PERMISSIONS}${userId}`)
    ]);
  }

  async invalidateApiEndpoints(endpoints = ['*']) {
    const patterns = endpoints.map(endpoint => 
      endpoint === '*' ? `${CACHE_KEYS.API}*` : `${CACHE_KEYS.API}${endpoint}*`
    );
    
    await Promise.all(patterns.map(pattern => 
      this.cacheService.invalidatePattern(pattern)
    ));
  }

  async invalidateByPrefix(prefix) {
    return await this.cacheService.invalidatePattern(`${prefix}*`);
  }
}

class SessionCacheOptimizer {
  constructor(cacheService) {
    this.cacheService = cacheService;
    this.sessionExtensions = new Map();
    this.touchInterval = null;
    this.startPeriodicTouch();
  }

  async setSession(sessionToken, sessionData, ttl = CACHE_TTL.SESSION) {
    const enhancedData = {
      ...sessionData,
      createdAt: Date.now(),
      lastAccessed: Date.now(),
      accessCount: 0
    };

    await this.cacheService.setSession(sessionToken, enhancedData, ttl);
    this.scheduleExtension(sessionToken, ttl);
  }

  async getSession(sessionToken) {
    const session = await this.cacheService.getSession(sessionToken);
    
    if (session) {
      session.lastAccessed = Date.now();
      session.accessCount = (session.accessCount || 0) + 1;
      await this.cacheService.setSession(sessionToken, session, CACHE_TTL.SESSION);
    }
    
    return session;
  }

  scheduleExtension(sessionToken, ttl) {
    const extensionTime = Math.floor(ttl * 0.8);
    
    if (this.sessionExtensions.has(sessionToken)) {
      clearTimeout(this.sessionExtensions.get(sessionToken));
    }

    const timer = setTimeout(async () => {
      try {
        const session = await this.cacheService.getSession(sessionToken);
        if (session && Date.now() < session.expires_at) {
          await this.cacheService.setSession(sessionToken, session, ttl);
          this.scheduleExtension(sessionToken, ttl);
        } else {
          this.sessionExtensions.delete(sessionToken);
        }
      } catch (error) {
        console.error('Session extension failed:', error);
      }
    }, extensionTime * 1000);

    this.sessionExtensions.set(sessionToken, timer);
  }

  startPeriodicTouch() {
    this.touchInterval = setInterval(async () => {
      for (const [token, timer] of this.sessionExtensions.entries()) {
        clearTimeout(timer);
        try {
          const session = await this.cacheService.getSession(token);
          if (session) {
            this.scheduleExtension(token, CACHE_TTL.SESSION);
          } else {
            this.sessionExtensions.delete(token);
          }
        } catch (error) {
          this.sessionExtensions.delete(token);
        }
      }
    }, 60000);
  }

  stopPeriodicTouch() {
    if (this.touchInterval) {
      clearInterval(this.touchInterval);
      this.touchInterval = null;
    }
    
    for (const timer of this.sessionExtensions.values()) {
      clearTimeout(timer);
    }
    this.sessionExtensions.clear();
  }
}

class CacheService {
  constructor() {
    this.memoryCache = new Map();
    this.cacheTimers = new Map();
    this.metrics = new CacheMetrics();
    this.invalidationStrategy = new CacheInvalidationStrategy(this);
    this.sessionOptimizer = new SessionCacheOptimizer(this);
    this.defaultTTL = 300;
    this.memoryCacheTTL = 60000;
    this.isRedisConnected = false;
    this.compressionEnabled = true;
    this.initRedisConnection();
  }

  async initRedisConnection() {
    try {
      if (redisClient && !redisClient.isOpen) {
        await redisClient.connect();
      }
      this.isRedisConnected = redisClient ? redisClient.isOpen : false;
      
      if (redisClient) {
        redisClient.on('error', err => {
          this.isRedisConnected = false;
          this.metrics.recordError('connection', err);
        });
        
        redisClient.on('connect', () => {
          this.isRedisConnected = true;
        });
        
        redisClient.on('ready', () => {
          this.isRedisConnected = true;
        });
      }
    } catch (error) {
      console.error('Failed to connect to Redis:', error);
      this.isRedisConnected = false;
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
    const paramHash = Object.keys(params).sort()
      .map(k => `${k}=${params[k]}`)
      .join('&');
    return paramHash ? `${base}:${paramHash}` : base;
  }

  async getSession(sessionToken) {
    const key = this.generateSessionKey(sessionToken);
    const cached = await this.get(key);
    if (cached) {
      this.metrics.recordHit('session');
      return cached;
    }
    this.metrics.recordMiss('session');
    return null;
  }

  async setSession(sessionToken, sessionData, ttl = CACHE_TTL.SESSION) {
    const key = this.generateSessionKey(sessionToken);
    this.metrics.recordSet('session');
    return await this.set(key, sessionData, ttl);
  }

  async invalidateSession(sessionToken) {
    const key = this.generateSessionKey(sessionToken);
    this.metrics.recordDelete('session');
    return await this.del(key);
  }

  async invalidateUserSessions(userId) {
    await this.invalidationStrategy.queueInvalidation(
      `${CACHE_KEYS.SESSION}*:${userId}*`,
      'user_sessions_invalidated'
    );
  }

  async getCachedUser(userId) {
    const key = this.generateUserKey(userId);
    const cached = await this.get(key);
    if (cached) {
      this.metrics.recordHit('user');
      return cached;
    }
    this.metrics.recordMiss('user');
    return null;
  }

  async setCachedUser(userId, userData, ttl = CACHE_TTL.USER) {
    const key = this.generateUserKey(userId);
    this.metrics.recordSet('user');
    return await this.set(key, userData, ttl);
  }

  async invalidateUserCache(userId) {
    const key = this.generateUserKey(userId);
    this.metrics.recordDelete('user');
    return await this.del(key);
  }

  async getCachedApiResponse(key) {
    const fullKey = `${CACHE_KEYS.API}${key}`;
    const cached = await this.get(fullKey);
    if (cached) {
      this.metrics.recordHit('api');
      return cached;
    }
    this.metrics.recordMiss('api');
    return null;
  }

  async setCachedApiResponse(key, responseData, isPublic = true, ttl = CACHE_TTL.API_PUBLIC) {
    const fullKey = `${CACHE_KEYS.API}${key}`;
    this.metrics.recordSet('api');
    return await this.set(fullKey, responseData, isPublic ? CACHE_TTL.API_PUBLIC : CACHE_TTL.API_PRIVATE);
  }

  async invalidateApiCache(pattern = '*') {
    await this.invalidationStrategy.queueInvalidation(
      `${CACHE_KEYS.API}${pattern}`,
      'api_cache_invalidated'
    );
  }

  async invalidateAllUserCache(userId) {
    await this.invalidationStrategy.invalidateUserRelated(userId);
  }

  async get(key) {
    const startTime = Date.now();
    
    try {
      if (this.isRedisConnected && redisClient) {
        const redisValue = await redisClient.get(key);
        if (redisValue) {
          this.metrics.recordHit();
          const parsed = JSON.parse(redisValue);
          this.metrics.recordLatency(Date.now() - startTime);
          return parsed;
        }
      }

      if (this.memoryCache.has(key)) {
        const memEntry = this.memoryCache.get(key);
        if (Date.now() < memEntry.expiresAt) {
          this.metrics.recordHit();
          if (this.isRedisConnected && redisClient) {
            redisClient.setEx(key, this.defaultTTL, JSON.stringify(memEntry.value)).catch(() => {});
          }
          this.metrics.recordLatency(Date.now() - startTime);
          return memEntry.value;
        }
        this.memoryCache.delete(key);
      }

      this.metrics.recordMiss();
      this.metrics.recordLatency(Date.now() - startTime);
      return null;
    } catch (error) {
      this.metrics.recordError('get', error);
      this.metrics.recordLatency(Date.now() - startTime);
      return null;
    }
  }

  async set(key, value, ttl = this.defaultTTL) {
    try {
      const serialized = JSON.stringify(value);

      if (this.isRedisConnected && redisClient) {
        await redisClient.setEx(key, ttl, serialized);
      }

      this.memoryCache.set(key, {
        value,
        expiresAt: Date.now() + Math.min(ttl * 1000, this.memoryCacheTTL)
      });

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
      return true;
    } catch (error) {
      this.metrics.recordError('set', error);
      return false;
    }
  }

  async del(key) {
    try {
      if (this.isRedisConnected && redisClient) {
        await redisClient.del(key);
      }

      this.memoryCache.delete(key);
      if (this.cacheTimers.has(key)) {
        clearTimeout(this.cacheTimers.get(key));
        this.cacheTimers.delete(key);
      }

      return true;
    } catch (error) {
      this.metrics.recordError('del', error);
      return false;
    }
  }

  async invalidatePattern(pattern) {
    try {
      if (this.isRedisConnected && redisClient) {
        const keys = await redisClient.keys(pattern);
        if (keys.length > 0) {
          await redisClient.del(keys);
        }
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
      this.metrics.recordError('invalidate', error);
      return false;
    }
  }

  matchPattern(key, pattern) {
    const regexPattern = pattern.replace(/\*/g, '.*').replace(/\?/g, '.');
    return new RegExp(`^${regexPattern}$`).test(key);
  }

  async getMulti(keys) {
    try {
      if (!this.isRedisConnected || !redisClient) {
        return keys.map(key => this.memoryCache.get(key)?.value || null);
      }

      const values = await redisClient.mGet(keys.map(k => k));
      return values.map(v => (v ? JSON.parse(v) : null));
    } catch (error) {
      this.metrics.recordError('getMulti', error);
      return keys.map(() => null);
    }
  }

  async setMulti(items, ttl = this.defaultTTL) {
    try {
      if (!this.isRedisConnected || !redisClient) {
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

      return true;
    } catch (error) {
      this.metrics.recordError('setMulti', error);
      return false;
    }
  }

  async clear() {
    try {
      if (this.isRedisConnected && redisClient) {
        await redisClient.flushDb();
      }

      this.memoryCache.clear();
      for (const timer of this.cacheTimers.values()) {
        clearTimeout(timer);
      }
      this.cacheTimers.clear();

      return true;
    } catch (error) {
      this.metrics.recordError('clear', error);
      return false;
    }
  }

  async isHealthy() {
    try {
      if (!this.isRedisConnected || !redisClient) {
        return false;
      }
      const pong = await redisClient.ping();
      return pong === 'PONG';
    } catch (error) {
      return false;
    }
  }

  getStats() {
    return this.metrics.getMetrics();
  }

  resetStats() {
    this.metrics.reset();
  }
}

const cacheService = new CacheService();

module.exports = cacheService;
module.exports.CACHE_KEYS = CACHE_KEYS;
module.exports.CACHE_TTL = CACHE_TTL;
module.exports.CACHE_STRATEGY = CACHE_STRATEGY;
module.exports.CacheMetrics = CacheMetrics;
module.exports.CacheInvalidationStrategy = CacheInvalidationStrategy;
module.exports.SessionCacheOptimizer = SessionCacheOptimizer;
