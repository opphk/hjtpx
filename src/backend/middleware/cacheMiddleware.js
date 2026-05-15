const cacheService = require('../services/cacheService');
const cacheMetricsService = require('../services/cacheMetricsService');

const CACHEABLE_METHODS = ['GET', 'HEAD'];
const DEFAULT_TTL = 300;

const CACHE_INVALIDATION_STRATEGIES = {
  IMMEDIATE: 'immediate',
  DEFERRED: 'deferred',
  BATCHED: 'batched',
  TTL_BASED: 'ttl_based'
};

const endpointCacheConfig = {
  '/api/v1/users': { ttl: 60, isPublic: false, tags: ['users'], invalidationStrategy: CACHE_INVALIDATION_STRATEGIES.IMMEDIATE },
  '/api/v1/notifications': { ttl: 30, isPublic: false, tags: ['notifications'], invalidationStrategy: CACHE_INVALIDATION_STRATEGIES.IMMEDIATE },
  '/api/v1/health': { ttl: 5, isPublic: true, tags: ['health'], invalidationStrategy: CACHE_INVALIDATION_STRATEGIES.TTL_BASED },
  '/api/v1/analytics': { ttl: 60, isPublic: false, tags: ['analytics'], invalidationStrategy: CACHE_INVALIDATION_STRATEGIES.DEFERRED },
  '/api/v1/permissions': { ttl: 300, isPublic: false, tags: ['permissions'], invalidationStrategy: CACHE_INVALIDATION_STRATEGIES.IMMEDIATE },
  '/api/docs': { ttl: 3600, isPublic: true, tags: ['docs'], invalidationStrategy: CACHE_INVALIDATION_STRATEGIES.TTL_BASED }
};

function generateCacheKey(req) {
  const base = `${req.method}:${req.originalUrl}`;

  if (req.user) {
    return `${base}:user:${req.user.id}`;
  }

  const queryKeys = Object.keys(req.query).sort();
  if (queryKeys.length > 0) {
    const queryHash = queryKeys.map(k => `${k}=${req.query[k]}`).join('&');
    return `${base}:${queryHash}`;
  }

  return base;
}

function generateAdvancedCacheKey(req, options = {}) {
  const parts = [req.method, req.originalUrl];

  if (req.user && options.includeUserId !== false) {
    parts.push(`u:${req.user.id}`);
  }

  if (req.user && req.user.role) {
    parts.push(`r:${req.user.role}`);
  }

  if (req.headers['accept-language']) {
    parts.push(`l:${req.headers['accept-language'].split(',')[0]}`);
  }

  const queryKeys = Object.keys(req.query).sort();
  if (queryKeys.length > 0 && options.includeQuery !== false) {
    const queryHash = queryKeys.map(k => `${k}=${req.query[k]}`).join('&');
    parts.push(`q:${queryHash}`);
  }

  if (options.customKey) {
    parts.push(`c:${options.customKey}`);
  }

  return parts.join('|');
}

function shouldCache(req) {
  if (!CACHEABLE_METHODS.includes(req.method)) {
    return false;
  }

  if (req.query.noCache === 'true' || req.headers['cache-control']?.includes('no-cache')) {
    return false;
  }

  if (req.user && req.user.role === 'admin') {
    return false;
  }

  return true;
}

function shouldCacheResponse(res) {
  return res.statusCode >= 200 && res.statusCode < 300;
}

function getCacheConfig(path) {
  if (endpointCacheConfig && typeof endpointCacheConfig.getEndpointConfig === 'function') {
    return endpointCacheConfig.getEndpointConfig(path);
  }

  for (const [pattern, config] of Object.entries(endpointCacheConfig)) {
    if (path.startsWith(pattern)) {
      return config;
    }
  }
  return { ttl: DEFAULT_TTL, isPublic: true, tags: [], invalidationStrategy: CACHE_INVALIDATION_STRATEGIES.IMMEDIATE };
}

function apiCache(ttl = DEFAULT_TTL, options = {}) {
  return async (req, res, next) => {
    if (!shouldCache(req)) {
      return next();
    }

    const cacheKey = options.keyGenerator
      ? options.keyGenerator(req)
      : generateCacheKey(req);
    const config = getCacheConfig(req.path);
    const effectiveTtl = options.ttl || config.ttl || ttl;
    const tags = options.tags || config.tags || [];

    try {
      const cachedResponse = await cacheService.getCachedApiResponse(cacheKey);

      if (cachedResponse) {
        res.set('X-Cache', 'HIT');
        res.set('X-Cache-Key', cacheKey);
        res.set('X-Cache-TTL', effectiveTtl.toString());
        res.set('X-Cache-Hit-Time', new Date().toISOString());

        if (cacheMetricsService) {
          cacheMetricsService.recordHit('api', 1);
          cacheMetricsService.recordApiCacheHit(req.path);
          cacheMetricsService.recordLatency('get', Date.now() - req.startTime);
        }

        if (options.onCacheHit) {
          await options.onCacheHit(req, res, cachedResponse);
        }

        if (req.xhr || req.headers.accept?.includes('application/json')) {
          return res.json(cachedResponse);
        }
        return res.send(cachedResponse);
      }

      res.set('X-Cache', 'MISS');
      res.set('X-Cache-Key', cacheKey);
      res.set('X-Cache-Miss-Time', new Date().toISOString());

      if (cacheMetricsService) {
        cacheMetricsService.recordMiss('api', 1);
        cacheMetricsService.recordApiCacheMiss(req.path);
      }

      const originalJson = res.json.bind(res);
      const originalSend = res.send.bind(res);
      let responseData = null;
      let hasCached = false;

      res.json = (data) => {
        if (!hasCached && shouldCacheResponse(res)) {
          responseData = data;
          hasCached = true;
        }
        return originalJson(data);
      };

      res.send = (data) => {
        if (!hasCached && shouldCacheResponse(res) && typeof data !== 'string' || !data.startsWith('<')) {
          responseData = data;
          hasCached = true;
        }
        return originalSend(data);
      };

      res.on('finish', async () => {
        if (responseData && shouldCacheResponse(res)) {
          try {
            await cacheService.setCachedApiResponse(
              cacheKey,
              responseData,
              config.isPublic,
              effectiveTtl,
              tags
            );

            if (cacheMetricsService) {
              cacheMetricsService.recordSet('api', 1);
              cacheMetricsService.setApiCacheSize(cacheMetricsService.metrics.api.size + 1);
            }

            if (options.onCacheSet) {
              await options.onCacheSet(req, res, cacheKey, responseData);
            }
          } catch (error) {
            console.error('Failed to cache API response:', error);
            if (cacheMetricsService) {
              cacheMetricsService.recordError('cache_set', error.message);
            }
          }
        }
      });

      next();
    } catch (error) {
      console.error('Cache middleware error:', error);
      next();
    }
  };
}

function apiCacheWithValidation(ttl = DEFAULT_TTL, options = {}) {
  return async (req, res, next) => {
    if (!shouldCache(req)) {
      return next();
    }

    const cacheKey = options.keyGenerator
      ? options.keyGenerator(req)
      : generateCacheKey(req);
    const config = getCacheConfig(req.path);
    const effectiveTtl = options.ttl || config.ttl || ttl;
    const tags = options.tags || config.tags || [];

    try {
      const cachedResponse = await cacheService.getCachedApiResponse(cacheKey);

      if (cachedResponse) {
        res.set('X-Cache', 'HIT');
        res.set('X-Cache-Key', cacheKey);
        res.set('X-Cache-TTL', effectiveTtl.toString());

        if (options.validateResponse) {
          const isValid = await options.validateResponse(req, cachedResponse);
          if (!isValid) {
            res.set('X-Cache', 'STALE');
            await cacheService.invalidateApiCache(cacheKey);
            return next();
          }
        }

        if (req.xhr || req.headers.accept?.includes('application/json')) {
          return res.json(cachedResponse);
        }
        return res.send(cachedResponse);
      }

      res.set('X-Cache', 'MISS');
      res.set('X-Cache-Key', cacheKey);

      const originalJson = res.json.bind(res);
      const originalSend = res.send.bind(res);
      let responseData = null;
      let hasCached = false;

      res.json = (data) => {
        if (!hasCached && shouldCacheResponse(res)) {
          responseData = data;
          hasCached = true;
        }
        return originalJson(data);
      };

      res.send = (data) => {
        if (!hasCached && shouldCacheResponse(res) && typeof data !== 'string' || !data.startsWith('<')) {
          responseData = data;
          hasCached = true;
        }
        return originalSend(data);
      };

      res.on('finish', async () => {
        if (responseData && shouldCacheResponse(res)) {
          await cacheService.setCachedApiResponse(
            cacheKey,
            responseData,
            config.isPublic,
            effectiveTtl,
            tags
          );
        }
      });

      next();
    } catch (error) {
      console.error('Cache middleware error:', error);
      next();
    }
  };
}

function invalidateCache(pattern = '*') {
  return async (req, res, next) => {
    try {
      await cacheService.invalidateApiCache(pattern);
      next();
    } catch (error) {
      console.error('Cache invalidation error:', error);
      next();
    }
  };
}

function invalidateCacheByTag(tag) {
  return async (req, res, next) => {
    try {
      await cacheService.invalidateTag(tag);
      next();
    } catch (error) {
      console.error('Tag cache invalidation error:', error);
      next();
    }
  };
}

function invalidateCacheByUser(userId) {
  return async (req, res, next) => {
    try {
      await cacheService.invalidateTag(`user:${userId}`);
      await cacheService.invalidateApiCache(`*:user:${userId}*`);
      next();
    } catch (error) {
      console.error('User cache invalidation error:', error);
      next();
    }
  };
}

function invalidateAllRelatedCache(options = {}) {
  return async (req, res, next) => {
    try {
      const invalidations = [];

      if (options.userId || req.user?.id) {
        const userId = options.userId || req.user.id;
        invalidations.push(
          cacheService.invalidateTag(`user:${userId}`),
          cacheService.invalidateApiCache(`*:user:${userId}*`),
          cacheService.invalidatePermissions(userId),
          cacheService.invalidateUserCache(userId)
        );
      }

      if (options.patterns) {
        for (const pattern of options.patterns) {
          invalidations.push(cacheService.invalidateApiCache(pattern));
        }
      }

      if (options.tags) {
        for (const tag of options.tags) {
          invalidations.push(cacheService.invalidateTag(tag));
        }
      }

      await Promise.all(invalidations);
      next();
    } catch (error) {
      console.error('Bulk cache invalidation error:', error);
      next();
    }
  };
}

function userCacheMiddleware() {
  return async (req, res, next) => {
    if (req.method !== 'GET' || !req.user) {
      return next();
    }

    try {
      const userId = req.user.id;
      const cacheKey = `user:${userId}`;

      const cachedUser = await cacheService.getCachedUser(cacheKey);

      if (cachedUser) {
        res.set('X-User-Cache', 'HIT');
        req.cachedUser = cachedUser;
      } else {
        res.set('X-User-Cache', 'MISS');
      }

      const originalJson = res.json.bind(res);
      res.json = (data) => {
        if (res.statusCode === 200 && data && data.data && data.data.id === userId) {
          cacheService.setCachedUser(cacheKey, data.data, undefined, [`user:${userId}`, 'user']).catch(() => {});
        }
        return originalJson(data);
      };

      next();
    } catch (error) {
      console.error('User cache middleware error:', error);
      next();
    }
  };
}

function permissionsCacheMiddleware() {
  return async (req, res, next) => {
    if (req.method !== 'GET' || !req.user) {
      return next();
    }

    try {
      const userId = req.user.id;
      const cacheKey = `permissions:${userId}`;

      const cachedPermissions = await cacheService.getCachedPermissions(cacheKey);

      if (cachedPermissions) {
        res.set('X-Permissions-Cache', 'HIT');
        req.cachedPermissions = cachedPermissions;
      } else {
        res.set('X-Permissions-Cache', 'MISS');
      }

      next();
    } catch (error) {
      console.error('Permissions cache middleware error:', error);
      next();
    }
  };
}

function cacheStatsMiddleware() {
  return async (req, res, next) => {
    if (req.path === '/api/v1/cache/stats' && req.method === 'GET') {
      try {
        const stats = cacheService.getStats();
        return res.success(stats, 'Cache statistics retrieved successfully');
      } catch (error) {
        console.error('Cache stats error:', error);
        return res.error('Failed to retrieve cache statistics', 500);
      }
    }
    next();
  };
}

function cacheHealthMiddleware() {
  return async (req, res, next) => {
    if (req.path === '/api/v1/cache/health' && req.method === 'GET') {
      try {
        const isHealthy = await cacheService.isHealthy();
        const stats = cacheService.getStats();

        return res.success({
          healthy: isHealthy,
          redisConnected: stats.isRedisConnected,
          memoryCacheSize: stats.memoryCacheSize,
          maxMemoryCacheSize: stats.maxMemoryCacheSize,
          memoryCachePercent: stats.overall.memoryCachePercent
        }, 'Cache health status retrieved successfully');
      } catch (error) {
        console.error('Cache health check error:', error);
        return res.error('Failed to retrieve cache health status', 500);
      }
    }
    next();
  };
}

function cacheInvalidationStatsMiddleware() {
  return async (req, res, next) => {
    if (req.path === '/api/v1/cache/invalidate' && req.method === 'POST') {
      try {
        const { pattern, tag, userId, bulk } = req.body;

        if (bulk && Array.isArray(bulk)) {
          await Promise.all(bulk.map(item => {
            if (item.pattern) return cacheService.invalidateApiCache(item.pattern);
            if (item.tag) return cacheService.invalidateTag(item.tag);
            return Promise.resolve();
          }));
        } else if (pattern) {
          await cacheService.invalidateApiCache(pattern);
        } else if (tag) {
          await cacheService.invalidateTag(tag);
        } else if (userId) {
          await cacheService.invalidateAllUserCache(userId);
        } else {
          return res.error('Invalid invalidation request', 400);
        }

        return res.success({ invalidated: true }, 'Cache invalidation successful');
      } catch (error) {
        console.error('Cache invalidation error:', error);
        return res.error('Failed to invalidate cache', 500);
      }
    }
    next();
  };
}

module.exports = {
  apiCache,
  apiCacheWithValidation,
  invalidateCache,
  invalidateCacheByTag,
  invalidateCacheByUser,
  invalidateAllRelatedCache,
  userCacheMiddleware,
  permissionsCacheMiddleware,
  cacheStatsMiddleware,
  cacheHealthMiddleware,
  cacheInvalidationStatsMiddleware,
  shouldCache,
  shouldCacheResponse,
  generateCacheKey,
  generateAdvancedCacheKey,
  getCacheConfig,
  endpointCacheConfig,
  CACHE_INVALIDATION_STRATEGIES
};
