class AdvancedRateLimiter {
  constructor(options = {}) {
    this.limits = new Map();
    this.redis = options.redis || null;
    this.defaultWindowMs = options.windowMs || 60000;
    this.defaultMax = options.max || 100;
    this.prefix = options.prefix || 'ratelimit:';
    this.enabled = options.enabled !== false;
  }

  createLimiter(options = {}) {
    const {
      name = 'default',
      windowMs = this.defaultWindowMs,
      max = this.defaultMax,
      keyGenerator = (req) => req.ip,
      skip = (req) => false,
      handler = (req, res) => {
        res.status(429).json({
          error: 'too_many_requests',
          message: 'Rate limit exceeded',
          retryAfter: Math.ceil(windowMs / 1000)
        });
      },
      skipFailedRequests = false,
      keyPrefix = this.prefix
    } = options;

    const limiter = {
      name,
      windowMs,
      max,
      keyGenerator,
      skip,
      handler,
      skipFailedRequests,
      keyPrefix
    };

    this.limits.set(name, limiter);

    return async (req, res, next) => {
      if (this.enabled === false || skip(req)) {
        return next();
      }

      const key = `${keyPrefix}${name}:${keyGenerator(req)}`;

      try {
        if (this.redis) {
          await this.incrementRedisLimiter(key, windowMs, max, req, res, next, handler);
        } else {
          this.incrementMemoryLimiter(key, windowMs, max, req, res, next, handler);
        }
      } catch (error) {
        console.error('Rate limiter error:', error);
        next();
      }
    };
  }

  async incrementRedisLimiter(key, windowMs, max, req, res, next, handler) {
    const Redis = require('ioredis');
    const redis = this.redis instanceof Redis ? this.redis : new Redis(this.redis);

    const multi = redis.multi();
    multi.incr(key);
    multi.pttl(key);

    const results = await multi.exec();
    const current = results[0][1];
    const ttl = results[1][1];

    if (ttl === -1) {
      await redis.pexpire(key, windowMs);
    }

    res.setHeader('X-RateLimit-Limit', max);
    res.setHeader('X-RateLimit-Remaining', Math.max(0, max - current));
    res.setHeader('X-RateLimit-Reset', Math.ceil((Date.now() + ttl) / 1000));

    if (current > max) {
      return handler(req, res);
    }

    next();
  }

  incrementMemoryLimiter(key, windowMs, max, req, res, next, handler) {
    const now = Date.now();
    const record = this.memoryStore.get(key);

    if (!record || now - record.timestamp > windowMs) {
      this.memoryStore.set(key, { count: 1, timestamp: now });
      return this.setHeaders(res, max, 1, max);
    }

    record.count++;
    this.setHeaders(res, max, record.count, Math.max(0, max - record.count));

    if (record.count > max) {
      return handler(req, res);
    }

    next();
  }

  setHeaders(res, limit, current, remaining) {
    res.setHeader('X-RateLimit-Limit', limit);
    res.setHeader('X-RateLimit-Remaining', remaining);
    res.setHeader('X-RateLimit-Reset', Math.ceil((Date.now() + this.defaultWindowMs) / 1000));
  }

  get memoryStore() {
    if (!this._memoryStore) {
      this._memoryStore = new Map();
      
      setInterval(() => {
        const now = Date.now();
        for (const [key, record] of this._memoryStore.entries()) {
          if (now - record.timestamp > this.defaultWindowMs * 2) {
            this._memoryStore.delete(key);
          }
        }
      }, 60000);
    }
    return this._memoryStore;
  }

  reset(key) {
    if (this.redis) {
      return this.redis.del(`${this.prefix}${key}`);
    }
    this.memoryStore.delete(key);
  }

  resetAll() {
    if (this.redis) {
      return this.redis.flushdb();
    }
    this.memoryStore.clear();
  }

  getStatus(key) {
    if (this.redis) {
      return this.redis.get(`${this.prefix}${key}`);
    }
    const record = this.memoryStore.get(key);
    return record ? { count: record.count, timestamp: record.timestamp } : null;
  }

  removeLimiter(name) {
    this.limits.delete(name);
  }

  enable() {
    this.enabled = true;
  }

  disable() {
    this.enabled = false;
  }

  setRedis(redis) {
    this.redis = redis;
  }

  getConfig() {
    return {
      enabled: this.enabled,
      defaultWindowMs: this.defaultWindowMs,
      defaultMax: this.defaultMax,
      limits: Array.from(this.limits.keys()),
      storage: this.redis ? 'redis' : 'memory'
    };
  }
}

const rateLimiter = new AdvancedRateLimiter();

const globalLimiter = rateLimiter.createLimiter({
  name: 'global',
  windowMs: 60000,
  max: 100
});

const apiLimiter = rateLimiter.createLimiter({
  name: 'api',
  windowMs: 1000,
  max: 10
});

const authLimiter = rateLimiter.createLimiter({
  name: 'auth',
  windowMs: 300000,
  max: 5,
  handler: (req, res) => {
    res.status(429).json({
      error: 'too_many_requests',
      message: 'Too many authentication attempts. Please try again later.',
      retryAfter: 300
    });
  }
});

module.exports = { AdvancedRateLimiter, rateLimiter, globalLimiter, apiLimiter, authLimiter };
