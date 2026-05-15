const advancedCacheService = require('./advancedCacheService');
const db = require('../../../config/database/db');

class CacheWarmer {
  constructor() {
    this.warmingIntervals = new Map();
    this.warmingConfigs = {
      startup: { enabled: true, priority: 'high' },
      scheduled: { enabled: true, interval: 3600000 },
      hotData: { enabled: true, interval: 300000 }
    };

    this.hotDataPatterns = [
      { key: 'api:popular:*', weight: 10 },
      { key: 'user:*', weight: 5 },
      { key: 'session:*', weight: 3 }
    ];

    this.stats = {
      startupWarmings: 0,
      scheduledWarmings: 0,
      hotDataWarmings: 0,
      totalItemsWarmed: 0,
      lastStartupWarm: null,
      lastScheduledWarm: null,
      lastHotDataWarm: null,
      errors: 0
    };
  }

  async initialize() {
    console.log('🚀 Initializing Cache Warmer...');

    if (this.warmingConfigs.startup.enabled) {
      await this.warmOnStartup();
    }

    if (this.warmingConfigs.scheduled.enabled) {
      this.startScheduledWarming();
    }

    if (this.warmingConfigs.hotData.enabled) {
      this.startHotDataWarming();
    }

    this.setupGracefulShutdown();

    console.log('✅ Cache Warmer initialized');
  }

  async warmOnStartup() {
    console.log('🔥 Performing startup cache warming...');
    const startTime = Date.now();

    try {
      await Promise.all([
        this.warmUserCache(),
        this.warmSessionCache(),
        this.warmApiCache(),
        this.warmPermissionCache()
      ]);

      this.stats.startupWarmings++;
      this.stats.lastStartupWarm = new Date().toISOString();

      const duration = Date.now() - startTime;
      console.log(`✅ Startup cache warming completed in ${duration}ms`);

      return true;
    } catch (error) {
      this.stats.errors++;
      console.error('❌ Startup cache warming failed:', error);
      return false;
    }
  }

  async warmUserCache() {
    try {
      if (!db || !db.sequelize) return;

      const User = db.models?.User || this.getModel('User');
      if (!User) return;

      const recentUsers = await User.findAll({
        limit: 100,
        order: [['updatedAt', 'DESC']],
        attributes: ['id', 'email', 'username', 'role', 'status']
      });

      for (const user of recentUsers) {
        const key = `user:${user.id}`;
        await advancedCacheService.set(key, user.toJSON(), { ttl: 1800 });
        this.stats.totalItemsWarmed++;
      }

      console.log(`📦 Warmed ${recentUsers.length} user cache entries`);
    } catch (error) {
      console.error('User cache warming failed:', error);
      this.stats.errors++;
    }
  }

  async warmSessionCache() {
    try {
      console.log('📦 Skipping session cache warming (security reason)');
    } catch (error) {
      console.error('Session cache warming failed:', error);
      this.stats.errors++;
    }
  }

  async warmApiCache() {
    try {
      const publicEndpoints = ['/api/v1/health', '/api/v1/features', '/api/v1/config'];

      for (const endpoint of publicEndpoints) {
        const key = `api:${endpoint}`;
        try {
          const response = await fetch(`http://localhost:3000${endpoint}`);
          if (response.ok) {
            const data = await response.json();
            await advancedCacheService.set(key, data, { ttl: 300 });
            this.stats.totalItemsWarmed++;
          }
        } catch (error) {
          console.log(`Could not warm ${endpoint}: ${error.message}`);
        }
      }

      console.log(`📦 Warmed ${publicEndpoints.length} API cache entries`);
    } catch (error) {
      console.error('API cache warming failed:', error);
      this.stats.errors++;
    }
  }

  async warmPermissionCache() {
    try {
      if (!db || !db.sequelize) return;

      const Permission = db.models?.Permission || this.getModel('Permission');
      const Role = db.models?.Role || this.getModel('Role');

      if (!Permission || !Role) return;

      const permissions = await Permission.findAll({
        attributes: ['id', 'name', 'resource', 'action']
      });

      for (const permission of permissions) {
        const key = `permissions:${permission.id}`;
        await advancedCacheService.set(key, permission.toJSON(), { ttl: 3600 });
        this.stats.totalItemsWarmed++;
      }

      const roles = await Role.findAll({
        include: ['permissions']
      });

      for (const role of roles) {
        const key = `permissions:role:${role.id}`;
        await advancedCacheService.set(key, role.toJSON(), { ttl: 3600 });
        this.stats.totalItemsWarmed++;
      }

      console.log(
        `📦 Warmed ${permissions.length} permission and ${roles.length} role cache entries`
      );
    } catch (error) {
      console.error('Permission cache warming failed:', error);
      this.stats.errors++;
    }
  }

  getModel(modelName) {
    if (!db || !db.sequelize) return null;

    const model = db.sequelize.models[modelName];
    if (model) return model;

    try {
      const modelPath = `../../../models/${modelName.toLowerCase()}`;
      return require(modelPath);
    } catch (error) {
      return null;
    }
  }

  startScheduledWarming() {
    const interval = this.warmingConfigs.scheduled.interval;

    const timer = setInterval(async () => {
      await this.performScheduledWarming();
    }, interval);

    this.warmingIntervals.set('scheduled', timer);
    console.log(`⏰ Scheduled cache warming started (interval: ${interval}ms)`);
  }

  async performScheduledWarming() {
    console.log('⏰ Performing scheduled cache warming...');
    const startTime = Date.now();

    try {
      await this.warmUserCache();
      await this.warmApiCache();
      await this.warmPermissionCache();

      this.stats.scheduledWarmings++;
      this.stats.lastScheduledWarm = new Date().toISOString();

      const duration = Date.now() - startTime;
      console.log(`✅ Scheduled cache warming completed in ${duration}ms`);
    } catch (error) {
      this.stats.errors++;
      console.error('❌ Scheduled cache warming failed:', error);
    }
  }

  startHotDataWarming() {
    const interval = this.warmingConfigs.hotData.interval;

    const timer = setInterval(async () => {
      await this.performHotDataWarming();
    }, interval);

    this.warmingIntervals.set('hotData', timer);
    console.log(`🔥 Hot data cache warming started (interval: ${interval}ms)`);
  }

  async performHotDataWarming() {
    try {
      const stats = advancedCacheService.getStats();

      if (parseFloat(stats.total.hitRate) < 70) {
        console.log('🔍 Cache hit rate is low, performing hot data warming...');
        await this.refreshHotData();
      }

      this.stats.hotDataWarmings++;
      this.stats.lastHotDataWarm = new Date().toISOString();
    } catch (error) {
      this.stats.errors++;
      console.error('❌ Hot data warming failed:', error);
    }
  }

  async refreshHotData() {
    for (const pattern of this.hotDataPatterns) {
      try {
        const hotKeys = await this.getHotKeys(pattern.key, pattern.weight);

        for (const key of hotKeys) {
          const cached = await advancedCacheService.get(key, { bypassL1: true });

          if (!cached) {
            const freshData = await this.fetchDataForKey(key);
            if (freshData) {
              await advancedCacheService.set(key, freshData);
              this.stats.totalItemsWarmed++;
            }
          }
        }
      } catch (error) {
        console.error(`Failed to refresh hot data for pattern ${pattern.key}:`, error);
      }
    }
  }

  async getHotKeys(pattern, weight) {
    return [];
  }

  async fetchDataForKey(key) {
    return null;
  }

  async warmCustomCache(cacheItems) {
    if (!Array.isArray(cacheItems)) {
      cacheItems = [cacheItems];
    }

    let warmed = 0;
    for (const item of cacheItems) {
      const { key, value, ttl } = item;
      if (key && value) {
        await advancedCacheService.set(key, value, { ttl: ttl || 300 });
        warmed++;
      }
    }

    console.log(`📦 Warmed ${warmed} custom cache entries`);
    return warmed;
  }

  async warmByTag(tag) {
    try {
      const tagKey = `cache_tags:${tag}`;
      const keys = (await advancedCacheService.getFromRedis)
        ? await advancedCacheService.getFromRedis(tagKey)
        : [];

      let warmed = 0;
      for (const key of keys) {
        const cached = await advancedCacheService.get(key);
        if (cached) {
          warmed++;
        }
      }

      console.log(`📦 Refreshed ${warmed} cache entries for tag: ${tag}`);
      return warmed;
    } catch (error) {
      console.error(`Failed to warm cache by tag ${tag}:`, error);
      return 0;
    }
  }

  async warmByPattern(pattern) {
    try {
      const keys = await this.scanKeys(pattern);

      let warmed = 0;
      for (const key of keys) {
        const cached = await advancedCacheService.get(key, { bypassL1: true });
        if (!cached) {
          const freshData = await this.fetchDataForKey(key);
          if (freshData) {
            await advancedCacheService.set(key, freshData);
            warmed++;
          }
        }
      }

      console.log(`📦 Refreshed ${warmed} cache entries for pattern: ${pattern}`);
      return warmed;
    } catch (error) {
      console.error(`Failed to warm cache by pattern ${pattern}:`, error);
      return 0;
    }
  }

  async scanKeys(pattern) {
    return [];
  }

  stopWarming() {
    console.log('🛑 Stopping cache warming...');

    for (const [name, timer] of this.warmingIntervals) {
      clearInterval(timer);
      console.log(`   Stopped ${name} warming`);
    }

    this.warmingIntervals.clear();
  }

  setupGracefulShutdown() {
    const shutdown = () => {
      console.log('🛑 Shutting down Cache Warmer...');
      this.stopWarming();
    };

    process.on('SIGTERM', shutdown);
    process.on('SIGINT', shutdown);
  }

  getStats() {
    return {
      ...this.stats,
      warmingIntervals: Array.from(this.warmingIntervals.keys()),
      config: this.warmingConfigs
    };
  }

  updateConfig(config) {
    this.warmingConfigs = { ...this.warmingConfigs, ...config };

    if (config.scheduled?.interval) {
      this.stopWarming();
      if (this.warmingConfigs.scheduled.enabled) {
        this.startScheduledWarming();
      }
      if (this.warmingConfigs.hotData.enabled) {
        this.startHotDataWarming();
      }
    }
  }

  async forceWarming(type = 'all') {
    switch (type) {
      case 'startup':
        return await this.warmOnStartup();
      case 'scheduled':
        return await this.performScheduledWarming();
      case 'hotData':
        return await this.performHotDataWarming();
      case 'all':
        return await Promise.all([
          this.warmOnStartup(),
          this.performScheduledWarming(),
          this.performHotDataWarming()
        ]);
      default:
        console.error(`Unknown warming type: ${type}`);
        return false;
    }
  }
}

const cacheWarmer = new CacheWarmer();

module.exports = cacheWarmer;
