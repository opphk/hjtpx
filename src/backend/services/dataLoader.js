class DataLoader {
  constructor(batchLoadFn, options = {}) {
    if (typeof batchLoadFn !== 'function') {
      throw new Error('DataLoader requires a batch load function');
    }

    this.batchLoadFn = batchLoadFn;
    this.options = {
      maxBatchSize: options.maxBatchSize || 100,
      batchScheduleFn: options.batchScheduleFn || ((callback) => setImmediate(callback)),
      cache: options.cache !== false,
      cacheKeyFn: options.cacheKeyFn || ((key) => key),
      ...options
    };

    this.cache = new Map();
    this.queue = [];
    this.isScheduled = false;
    this.stats = {
      batches: 0,
      hits: 0,
      misses: 0,
      totalLoadTime: 0
    };
  }

  load(key) {
    if (key === null || key === undefined) {
      throw new Error('DataLoader cannot load null or undefined keys');
    }

    const cacheKey = this.options.cacheKeyFn(key);

    if (this.options.cache && this.cache.has(cacheKey)) {
      const cached = this.cache.get(cacheKey);
      if (cached.status === 'fulfilled') {
        this.stats.hits++;
        return Promise.resolve(cached.value);
      } else if (cached.status === 'rejected') {
        return Promise.reject(cached.error);
      }
    }

    const promise = new Promise((resolve, reject) => {
      this.queue.push({ key, cacheKey, resolve, reject });
    });

    if (this.options.cache) {
      this.cache.set(cacheKey, { status: 'pending', promise });
    }

    if (!this.isScheduled) {
      this.isScheduled = true;
      this.options.batchScheduleFn(() => this.dispatchBatch());
    }

    return promise;
  }

  async dispatchBatch() {
    this.isScheduled = false;

    if (this.queue.length === 0) {
      return;
    }

    const batch = this.queue.slice(0, this.options.maxBatchSize);
    this.queue = this.queue.slice(this.options.maxBatchSize);

    const keys = batch.map(item => item.key);
    const startTime = Date.now();

    try {
      const results = await this.batchLoadFn(keys);
      this.stats.batches++;
      this.stats.totalLoadTime += Date.now() - startTime;

      if (!Array.isArray(results)) {
        throw new Error('Batch load function must return an array');
      }

      if (results.length !== keys.length) {
        throw new Error(`Batch load function returned ${results.length} results, but ${keys.length} keys were provided`);
      }

      batch.forEach((item, index) => {
        const result = results[index];
        
        if (this.options.cache) {
          this.cache.set(item.cacheKey, { status: 'fulfilled', value: result });
        }
        
        item.resolve(result);
      });
    } catch (error) {
      batch.forEach((item) => {
        if (this.options.cache) {
          this.cache.set(item.cacheKey, { status: 'rejected', error });
        }
        item.reject(error);
      });
    }
  }

  clear(key) {
    const cacheKey = this.options.cacheKeyFn(key);
    this.cache.delete(cacheKey);
    return this;
  }

  clearAll() {
    this.cache.clear();
    return this;
  }

  getStats() {
    return {
      ...this.stats,
      cacheSize: this.cache.size,
      queueSize: this.queue.length,
      isScheduled: this.isScheduled,
      hitRate: this.stats.hits / (this.stats.hits + this.stats.misses) || 0,
      avgBatchSize: this.stats.batches > 0 ? this.stats.totalLoadTime / this.stats.batches : 0
    };
  }

  static createCacheKey(key) {
    if (key === null || key === undefined) {
      return 'null';
    }
    if (typeof key === 'object') {
      try {
        return JSON.stringify(key);
      } catch {
        return String(key);
      }
    }
    return String(key);
  }
}

class DataLoaderRegistry {
  constructor() {
    this.loaders = new Map();
  }

  get(name) {
    if (!this.loaders.has(name)) {
      throw new Error(`DataLoader "${name}" not found in registry`);
    }
    return this.loaders.get(name);
  }

  create(name, batchLoadFn, options = {}) {
    if (this.loaders.has(name)) {
      return this.loaders.get(name);
    }

    const loader = new DataLoader(batchLoadFn, {
      cacheKeyFn: options.cacheKeyFn || DataLoader.createCacheKey,
      maxBatchSize: options.maxBatchSize || 100,
      batchScheduleFn: options.batchScheduleFn,
      cache: options.cache !== false,
      ...options
    });

    this.loaders.set(name, loader);
    return loader;
  }

  clear(name) {
    if (this.loaders.has(name)) {
      this.loaders.get(name).clearAll();
    }
    return this;
  }

  clearAll() {
    for (const loader of this.loaders.values()) {
      loader.clearAll();
    }
    return this;
  }

  getAllStats() {
    const stats = {};
    for (const [name, loader] of this.loaders.entries()) {
      stats[name] = loader.getStats();
    }
    return stats;
  }
}

const globalRegistry = new DataLoaderRegistry();

function createLoaders(userService, postService) {
  return {
    userLoader: globalRegistry.create('user', async (ids) => {
      const users = await userService.findByIds(ids);
      return ids.map(id => users.find(u => u.id === id) || null);
    }),

    postLoader: globalRegistry.create('post', async (ids) => {
      const posts = await postService.findByIds(ids);
      return ids.map(id => posts.find(p => p.id === id) || null);
    }),

    userPostsLoader: globalRegistry.create('userPosts', async (userIds) => {
      const postsMap = await postService.findByUserIds(userIds);
      return userIds.map(userId => postsMap[userId] || []);
    }),

    postAuthorLoader: globalRegistry.create('postAuthor', async (postIds) => {
      const posts = await postService.findByIds(postIds);
      const authorIds = [...new Set(posts.map(p => p.authorId).filter(Boolean))];
      const authors = await userService.findByIds(authorIds);
      const authorMap = new Map(authors.map(a => [a.id, a]));
      return postIds.map(postId => {
        const post = posts.find(p => p.id === postId);
        return post ? authorMap.get(post.authorId) || null : null;
      });
    })
  };
}

module.exports = {
  DataLoader,
  DataLoaderRegistry,
  globalRegistry,
  createLoaders
};
