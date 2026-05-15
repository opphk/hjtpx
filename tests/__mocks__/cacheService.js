const mockCacheService = {
  get: jest.fn().mockResolvedValue(null),
  set: jest.fn().mockResolvedValue(true),
  del: jest.fn().mockResolvedValue(true),
  getStats: jest.fn().mockResolvedValue({ hits: 0, misses: 0, sets: 0 }),
  flushAll: jest.fn().mockResolvedValue(true),
  isRedisConnected: false,
  initRedisConnection: jest.fn().mockResolvedValue(true),
  memoryCache: new Map(),
  resetStats: jest.fn()
};

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

module.exports = mockCacheService;
module.exports.CACHE_KEYS = CACHE_KEYS;
module.exports.CACHE_TTL = CACHE_TTL;
module.exports.CACHE_PRIORITY = CACHE_PRIORITY;
module.exports.CACHE_POLICY = CACHE_POLICY;
