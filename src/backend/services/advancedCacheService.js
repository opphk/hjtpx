const db = require('../../../config/database/db');
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
  LOCK: 'lock:',
  VERSION: 'version:'
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

const CACHE_LEVELS = {
  L1_MEMORY: 1,
  L2_REDIS: 2,
  L3_DATABASE: 3
};

class AdvancedCacheService {
  constructor() {
    this.l1Cache = new Map();
    this.l1CacheOrder = [];
    this.l1CacheTimers = new Map();
    this.l2Connected = false;
    this.l3Connected = false;

    this.config = {
      l1MaxSize: 1000,
      l1Ttl: 60000,
      l2Ttl: 300,
      enableL1: true,
      enableL2: true,
      enableL3: true,
      compressionThreshold: 1024,
      enableCompression: true,
      enableSharding: false,
      shardCount: 4,
      lockTimeout: 10000,
      lockRetryDelay: 100,
      lockRetryCount: 50
    };

    this.stats = {
      l1: { hits: 0, misses: 0, sets: 0, evictions: 0 },
      l2: { hits: 0, misses: 0, sets: 0, errors: 0 },
      l3: { hits: 0, misses: 0, sets: 0, errors: 0 },
      total: { hits: 0, misses: 0, sets: 0, invalidations: 0 },
      latency: { l1: [], l2: [], l3: [] },
      memory: { used: 0, peak: 0 }
    };

    this.versionMap = new Map();

    this.initConnections();
    this.startMonitoring();
  }

  async initConnections() {
    try {
      if (!redisClient.isOpen) {
        await redisClient.connect();
      }
      this.l2Connected = true;
      redisClient.on('error', err => {
        console.error('L2 Cache (Redis) error:', err);
        this.l2Connected = false;
      });
      redisClient.on('connect', () => {
        this.l2Connected = true;
      });
    } catch (error) {
      console.error('Failed to connect to L2 Cache:', error);
      this.l2Connected = false;
    }

    try {
      if (db && db.sequelize) {
        await db.sequelize.authenticate();
        this.l3Connected = true;
      }
    } catch (error) {
      console.error('Failed to connect to L3 Cache (Database):', error);
      this.l3Connected = false;
    }
  }

  measureLatency(level, startTime) {
    const latency = Date.now() - startTime;
    const key =
      level === CACHE_LEVELS.L1_MEMORY ? 'l1' : level === CACHE_LEVELS.L2_REDIS ? 'l2' : 'l3';
    if (this.stats.latency[key]) {
      this.stats.latency[key].push(latency);
      if (this.stats.latency[key].length > 1000) {
        this.stats.latency[key].shift();
      }
    }
  }

  getAverageLatency(level) {
    const key =
      level === CACHE_LEVELS.L1_MEMORY ? 'l1' : level === CACHE_LEVELS.L2_REDIS ? 'l2' : 'l3';
    const latencies = this.stats.latency[key];
    if (latencies.length === 0) return 0;
    const sum = latencies.reduce((a, b) => a + b, 0);
    return (sum / latencies.length).toFixed(2);
  }

  async get(key, options = {}) {
    const startTime = Date.now();
    const { bypassL1 = false, bypassL2 = false, useDatabase = false } = options;

    try {
      if (this.config.enableL1 && !bypassL1) {
        const l1Result = this.getFromL1(key);
        if (l1Result) {
          this.stats.l1.hits++;
          this.stats.total.hits++;
          this.measureLatency(CACHE_LEVELS.L1_MEMORY, startTime);
          return l1Result;
        }
        this.stats.l1.misses++;
      }

      if (this.config.enableL2 && !bypassL2 && this.l2Connected) {
        try {
          const l2Result = await redisClient.get(key);
          if (l2Result) {
            this.stats.l2.hits++;
            this.stats.total.hits++;
            this.measureLatency(CACHE_LEVELS.L2_REDIS, startTime);

            if (this.config.enableL1 && !bypassL1) {
              this.setToL1(key, JSON.parse(l2Result));
            }

            return JSON.parse(l2Result);
          }
          this.stats.l2.misses++;
        } catch (error) {
          this.stats.l2.errors++;
          console.error('L2 Cache get error:', error);
        }
      }

      if (useDatabase && this.config.enableL3 && this.l3Connected) {
        const l3Result = await this.getFromL3(key);
        if (l3Result) {
          this.stats.l3.hits++;
          this.stats.total.hits++;
          this.measureLatency(CACHE_LEVELS.L3_DATABASE, startTime);

          if (this.config.enableL2 && this.l2Connected) {
            await redisClient.setEx(key, this.config.l2Ttl, JSON.stringify(l3Result));
          }

          if (this.config.enableL1) {
            this.setToL1(key, l3Result);
          }

          return l3Result;
        }
        this.stats.l3.misses++;
      }

      this.stats.total.misses++;
      return null;
    } catch (error) {
      console.error('Cache get error:', error);
      this.stats.total.misses++;
      return null;
    }
  }

  getFromL1(key) {
    if (!this.l1Cache.has(key)) return null;

    const entry = this.l1Cache.get(key);
    if (Date.now() > entry.expiresAt) {
      this.l1Cache.delete(key);
      this.removeFromL1Order(key);
      this.stats.l1.evictions++;
      return null;
    }

    this.updateL1Order(key);
    return entry.value;
  }

  setToL1(key, value, ttl = this.config.l1Ttl) {
    const size = this.calculateSize(value);

    if (this.l1Cache.size >= this.config.l1MaxSize && !this.l1Cache.has(key)) {
      this.evictFromL1();
    }

    this.l1Cache.set(key, {
      value,
      size,
      expiresAt: Date.now() + ttl,
      priority: CACHE_PRIORITY.MEDIUM
    });

    this.addToL1Order(key);

    this.updateMemoryUsage();

    if (this.l1CacheTimers.has(key)) {
      clearTimeout(this.l1CacheTimers.get(key));
    }

    const timer = setTimeout(() => {
      this.l1Cache.delete(key);
      this.removeFromL1Order(key);
      this.l1CacheTimers.delete(key);
      this.stats.l1.evictions++;
    }, ttl);

    this.l1CacheTimers.set(key, timer);
    this.stats.l1.sets++;
  }

  updateL1Order(key) {
    const index = this.l1CacheOrder.indexOf(key);
    if (index > -1) {
      this.l1CacheOrder.splice(index, 1);
    }
    this.l1CacheOrder.push(key);
  }

  addToL1Order(key) {
    if (!this.l1CacheOrder.includes(key)) {
      this.l1CacheOrder.push(key);
    }
  }

  removeFromL1Order(key) {
    const index = this.l1CacheOrder.indexOf(key);
    if (index > -1) {
      this.l1CacheOrder.splice(index, 1);
    }
  }

  evictFromL1() {
    while (this.l1Cache.size >= this.config.l1MaxSize && this.l1CacheOrder.length > 0) {
      const oldestKey = this.l1CacheOrder.shift();
      if (this.l1Cache.has(oldestKey)) {
        this.l1Cache.delete(oldestKey);
        if (this.l1CacheTimers.has(oldestKey)) {
          clearTimeout(this.l1CacheTimers.get(oldestKey));
          this.l1CacheTimers.delete(oldestKey);
        }
        this.stats.l1.evictions++;
      }
    }
  }

  async set(key, value, options = {}) {
    const startTime = Date.now();
    const {
      ttl = this.config.l2Ttl,
      bypassL1 = false,
      bypassL2 = false,
      compress = false
    } = options;

    try {
      let dataToStore = value;

      if (this.config.enableCompression && compress) {
        dataToStore = this.compress(JSON.stringify(value));
      }

      if (this.config.enableL1 && !bypassL1) {
        this.setToL1(key, value, Math.min(ttl * 1000, this.config.l1Ttl));
      }

      if (this.config.enableL2 && !bypassL2 && this.l2Connected) {
        try {
          await redisClient.setEx(key, ttl, JSON.stringify(dataToStore));
          this.stats.l2.sets++;
          this.measureLatency(CACHE_LEVELS.L2_REDIS, startTime);
        } catch (error) {
          this.stats.l2.errors++;
          console.error('L2 Cache set error:', error);
        }
      }

      this.stats.total.sets++;
      this.incrementVersion(key);

      return true;
    } catch (error) {
      console.error('Cache set error:', error);
      return false;
    }
  }

  async delete(key) {
    try {
      if (this.config.enableL1) {
        this.l1Cache.delete(key);
        this.removeFromL1Order(key);
        if (this.l1CacheTimers.has(key)) {
          clearTimeout(this.l1CacheTimers.get(key));
          this.l1CacheTimers.delete(key);
        }
      }

      if (this.l2Connected) {
        await redisClient.del(key);
      }

      this.stats.total.invalidations++;
      this.incrementVersion(key);

      return true;
    } catch (error) {
      console.error('Cache delete error:', error);
      return false;
    }
  }

  async invalidatePattern(pattern) {
    try {
      const regex = new RegExp('^' + pattern.replace(/\*/g, '.*').replace(/\?/g, '.') + '$');

      if (this.config.enableL1) {
        for (const key of this.l1Cache.keys()) {
          if (regex.test(key)) {
            this.l1Cache.delete(key);
            this.removeFromL1Order(key);
            if (this.l1CacheTimers.has(key)) {
              clearTimeout(this.l1CacheTimers.get(key));
              this.l1CacheTimers.delete(key);
            }
          }
        }
      }

      if (this.l2Connected) {
        let cursor = 0;
        do {
          const result = await redisClient.scan(cursor, { MATCH: pattern, COUNT: 100 });
          cursor = result.cursor;
          if (result.keys.length > 0) {
            await redisClient.del(result.keys);
            this.stats.total.invalidations += result.keys.length;
          }
        } while (cursor !== 0);
      }

      return true;
    } catch (error) {
      console.error('Cache invalidate pattern error:', error);
      return false;
    }
  }

  async getFromL3(key) {
    return null;
  }

  async setToL3(key, value) {
    return true;
  }

  calculateSize(value) {
    const str = JSON.stringify(value);
    return str.length * 2;
  }

  updateMemoryUsage() {
    let total = 0;
    for (const entry of this.l1Cache.values()) {
      total += entry.size || 0;
    }
    this.stats.memory.used = total;
    if (total > this.stats.memory.peak) {
      this.stats.memory.peak = total;
    }
  }

  compress(data) {
    return data;
  }

  decompress(data) {
    return JSON.parse(data);
  }

  incrementVersion(key) {
    const current = this.versionMap.get(key) || 0;
    this.versionMap.set(key, current + 1);
  }

  getVersion(key) {
    return this.versionMap.get(key) || 0;
  }

  async acquireLock(key, ttl = this.config.lockTimeout) {
    if (!this.l2Connected) return false;

    const lockKey = `${CACHE_KEYS.LOCK}${key}`;
    const lockValue = `${Date.now()}-${Math.random()}`;

    try {
      const result = await redisClient.set(lockKey, lockValue, {
        NX: true,
        PX: ttl
      });

      if (result === 'OK') {
        return { acquired: true, value: lockValue };
      }
      return { acquired: false, value: null };
    } catch (error) {
      console.error('Lock acquisition error:', error);
      return { acquired: false, value: null };
    }
  }

  async releaseLock(key, lockValue) {
    if (!this.l2Connected) return false;

    const fullKey = `${CACHE_KEYS.LOCK}${key}`;

    try {
      const script = `
        if redis.call("get", KEYS[1]) == ARGV[1] then
          return redis.call("del", KEYS[1])
        else
          return 0
        end
      `;

      const result = await redisClient.eval(script, {
        keys: [fullKey],
        arguments: [lockValue]
      });

      return result === 1;
    } catch (error) {
      console.error('Lock release error:', error);
      return false;
    }
  }

  async withLock(key, callback, options = {}) {
    const { retries = this.config.lockRetryCount, delay = this.config.lockRetryDelay } = options;

    for (let i = 0; i < retries; i++) {
      const lock = await this.acquireLock(key);

      if (lock.acquired) {
        try {
          const result = await callback();
          return result;
        } finally {
          await this.releaseLock(key, lock.value);
        }
      }

      await new Promise(resolve => setTimeout(resolve, delay));
    }

    throw new Error(`Failed to acquire lock for key: ${key}`);
  }

  async getOrSet(key, fetchFn, options = {}) {
    const cached = await this.get(key, options);

    if (cached !== null) {
      return cached;
    }

    const value = await fetchFn();

    if (value !== null && value !== undefined) {
      await this.set(key, value, options);
    }

    return value;
  }

  async warmUp(data) {
    const { key, value, ttl = this.config.l2Ttl } = data;
    await this.set(key, value, { ttl });
  }

  startMonitoring() {
    setInterval(() => {
      this.cleanup();
    }, 60000);
  }

  cleanup() {
    if (this.config.enableL1) {
      const now = Date.now();
      for (const [key, entry] of this.l1Cache.entries()) {
        if (now > entry.expiresAt) {
          this.l1Cache.delete(key);
          this.removeFromL1Order(key);
          if (this.l1CacheTimers.has(key)) {
            clearTimeout(this.l1CacheTimers.get(key));
            this.l1CacheTimers.delete(key);
          }
          this.stats.l1.evictions++;
        }
      }
    }
  }

  async clear() {
    try {
      if (this.config.enableL1) {
        this.l1Cache.clear();
        this.l1CacheOrder = [];
        for (const timer of this.l1CacheTimers.values()) {
          clearTimeout(timer);
        }
        this.l1CacheTimers.clear();
      }

      if (this.l2Connected) {
        await redisClient.flushDb();
      }

      this.versionMap.clear();

      return true;
    } catch (error) {
      console.error('Cache clear error:', error);
      return false;
    }
  }

  getStats() {
    const totalL1 = this.stats.l1.hits + this.stats.l1.misses;
    const totalL2 = this.stats.l2.hits + this.stats.l2.misses;
    const totalL3 = this.stats.l3.hits + this.stats.l3.misses;
    const total = this.stats.total.hits + this.stats.total.misses;

    return {
      l1: {
        hits: this.stats.l1.hits,
        misses: this.stats.l1.misses,
        sets: this.stats.l1.sets,
        evictions: this.stats.l1.evictions,
        hitRate: totalL1 > 0 ? ((this.stats.l1.hits / totalL1) * 100).toFixed(2) + '%' : '0%',
        size: this.l1Cache.size,
        maxSize: this.config.l1MaxSize,
        avgLatency: this.getAverageLatency(CACHE_LEVELS.L1_MEMORY) + 'ms'
      },
      l2: {
        hits: this.stats.l2.hits,
        misses: this.stats.l2.misses,
        sets: this.stats.l2.sets,
        errors: this.stats.l2.errors,
        hitRate: totalL2 > 0 ? ((this.stats.l2.hits / totalL2) * 100).toFixed(2) + '%' : '0%',
        connected: this.l2Connected,
        avgLatency: this.getAverageLatency(CACHE_LEVELS.L2_REDIS) + 'ms'
      },
      l3: {
        hits: this.stats.l3.hits,
        misses: this.stats.l3.misses,
        sets: this.stats.l3.sets,
        errors: this.stats.l3.errors,
        hitRate: totalL3 > 0 ? ((this.stats.l3.hits / totalL3) * 100).toFixed(2) + '%' : '0%',
        connected: this.l3Connected,
        avgLatency: this.getAverageLatency(CACHE_LEVELS.L3_DATABASE) + 'ms'
      },
      total: {
        hits: this.stats.total.hits,
        misses: this.stats.total.misses,
        sets: this.stats.total.sets,
        invalidations: this.stats.total.invalidations,
        hitRate: total > 0 ? ((this.stats.total.hits / total) * 100).toFixed(2) + '%' : '0%'
      },
      memory: {
        used: this.stats.memory.used,
        peak: this.stats.memory.peak,
        usedFormatted: this.formatBytes(this.stats.memory.used),
        peakFormatted: this.formatBytes(this.stats.memory.peak)
      },
      config: this.config,
      status: {
        l1Enabled: this.config.enableL1,
        l2Enabled: this.config.enableL2 && this.l2Connected,
        l3Enabled: this.config.enableL3 && this.l3Connected,
        healthy: this.l2Connected
      }
    };
  }

  formatBytes(bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  }

  resetStats() {
    this.stats = {
      l1: { hits: 0, misses: 0, sets: 0, evictions: 0 },
      l2: { hits: 0, misses: 0, sets: 0, errors: 0 },
      l3: { hits: 0, misses: 0, sets: 0, errors: 0 },
      total: { hits: 0, misses: 0, sets: 0, invalidations: 0 },
      latency: { l1: [], l2: [], l3: [] },
      memory: { used: 0, peak: 0 }
    };
  }
}

const advancedCacheService = new AdvancedCacheService();

module.exports = advancedCacheService;
module.exports.CACHE_KEYS = CACHE_KEYS;
module.exports.CACHE_TTL = CACHE_TTL;
module.exports.CACHE_PRIORITY = CACHE_PRIORITY;
module.exports.CACHE_LEVELS = CACHE_LEVELS;
