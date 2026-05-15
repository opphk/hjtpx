const cacheConfig = {
  ttl: {
    DEFAULT: 300,
    SESSION: 604800,
    USER: 1800,
    API_PUBLIC: 300,
    API_PRIVATE: 60,
    PERMISSIONS: 3600,
    ANALYTICS: 300,
    SHORT: 60,
    MEDIUM: 300,
    LONG: 3600,
    VERY_LONG: 86400,
    TOKEN_BLACKLIST: 604800,
    RATE_LIMIT: 60,
    HEALTH_CHECK: 5,
    NOTIFICATIONS: 30,
    DOCS: 3600
  },

  maxSize: {
    MEMORY_CACHE: 1000,
    MAX_ENTRY_SIZE: 1048576,
    EVICTION_THRESHOLD: 0.8,
    MAX_BATCH_SIZE: 100
  },

  warmup: {
    ENABLED: false,
    INTERVAL: 300000,
    BATCH_SIZE: 10,
    PRIORITY: {
      HIGH: 1,
      MEDIUM: 2,
      LOW: 3
    },
    ENDPOINTS: [
      {
        path: '/api/v1/health',
        ttl: 5,
        priority: 1
      },
      {
        path: '/api/v1/config',
        ttl: 3600,
        priority: 2
      }
    ]
  },

  policy: {
    EVICTION_STRATEGY: 'LRU',
    COMPRESSION_ENABLED: false,
    COMPRESSION_THRESHOLD: 1024,
    STALE_WHILE_REVALIDATE: true,
    SLIDING_EXPIRATION: true,
    LOCK_TIMEOUT: 5000,
    CACHE_STALE_THRESHOLD: 0.8,
    BATCH_SIZE: 100,
    MAX_CONCURRENT_REQUESTS: 100
  },

  endpoints: {
    '/api/v1/users': {
      ttl: 60,
      isPublic: false,
      tags: ['users'],
      invalidationStrategy: 'immediate'
    },
    '/api/v1/notifications': {
      ttl: 30,
      isPublic: false,
      tags: ['notifications'],
      invalidationStrategy: 'immediate'
    },
    '/api/v1/health': {
      ttl: 5,
      isPublic: true,
      tags: ['health'],
      invalidationStrategy: 'ttl_based'
    },
    '/api/v1/analytics': {
      ttl: 60,
      isPublic: false,
      tags: ['analytics'],
      invalidationStrategy: 'deferred'
    },
    '/api/v1/permissions': {
      ttl: 300,
      isPublic: false,
      tags: ['permissions'],
      invalidationStrategy: 'immediate'
    },
    '/api/docs': {
      ttl: 3600,
      isPublic: true,
      tags: ['docs'],
      invalidationStrategy: 'ttl_based'
    },
    '/api/v1/profile': {
      ttl: 300,
      isPublic: false,
      tags: ['user'],
      invalidationStrategy: 'immediate'
    },
    '/api/v1/dashboard': {
      ttl: 60,
      isPublic: false,
      tags: ['dashboard'],
      invalidationStrategy: 'deferred'
    }
  },

  stats: {
    ENABLED: true,
    COLLECTION_INTERVAL: 60000,
    HISTOGRAM_BUCKETS: [1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000],
    PERCENTILES: [50, 90, 95, 99]
  },

  tags: [
    'user',
    'session',
    'permissions',
    'api',
    'analytics',
    'health',
    'notifications',
    'dashboard',
    'docs'
  ],

  getTTL(path, options = {}) {
    if (options.customTTL) {
      return options.customTTL;
    }

    if (this.endpoints[path]) {
      return this.endpoints[path].ttl;
    }

    for (const [pattern, config] of Object.entries(this.endpoints)) {
      if (path.startsWith(pattern)) {
        return config.ttl;
      }
    }

    return this.ttl.DEFAULT;
  },

  getMaxSize(type = 'MEMORY_CACHE') {
    return this.maxSize[type] || this.maxSize.MEMORY_CACHE;
  },

  isCacheable(req, options = {}) {
    if (options.forceCache) {
      return true;
    }

    const cacheableMethods = ['GET', 'HEAD'];
    if (!cacheableMethods.includes(req.method)) {
      return false;
    }

    if (req.query.noCache === 'true' || req.headers['cache-control']?.includes('no-cache')) {
      return false;
    }

    if (req.user && req.user.role === 'admin' && !options.cacheAdmin) {
      return false;
    }

    return true;
  },

  getEndpointConfig(path) {
    if (this.endpoints[path]) {
      return this.endpoints[path];
    }

    for (const [pattern, config] of Object.entries(this.endpoints)) {
      if (path.startsWith(pattern)) {
        return config;
      }
    }

    return {
      ttl: this.ttl.DEFAULT,
      isPublic: true,
      tags: [],
      invalidationStrategy: 'immediate'
    };
  },

  isPublicEndpoint(path) {
    const config = this.getEndpointConfig(path);
    return config.isPublic === true;
  },

  getTagsForEndpoint(path) {
    const config = this.getEndpointConfig(path);
    return config.tags || [];
  },

  shouldCompress(data) {
    if (!this.policy.COMPRESSION_ENABLED) {
      return false;
    }
    const size = typeof data === 'string' ? data.length : JSON.stringify(data).length;
    return size > this.policy.COMPRESSION_THRESHOLD;
  },

  getEvictionThreshold() {
    return Math.floor(this.maxSize.MEMORY_CACHE * this.maxSize.EVICTION_THRESHOLD);
  }
};

module.exports = cacheConfig;
