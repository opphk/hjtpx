const crypto = require('crypto');

class CacheSharding {
  constructor(options = {}) {
    this.shardCount = options.shardCount || 4;
    this.shards = new Map();
    this.config = {
      maxMemoryPerShard: options.maxMemoryPerShard || 100 * 1024 * 1024,
      evictionPolicy: options.evictionPolicy || 'lru',
      enableCompression: options.enableCompression || false,
      compressionThreshold: options.compressionThreshold || 1024
    };

    this.initShards();
  }

  initShards() {
    for (let i = 0; i < this.shardCount; i++) {
      this.shards.set(i, {
        data: new Map(),
        accessOrder: [],
        stats: {
          hits: 0,
          misses: 0,
          sets: 0,
          evictions: 0,
          size: 0
        }
      });
    }
  }

  getShardIndex(key) {
    const hash = this.hashKey(key);
    return hash % this.shardCount;
  }

  hashKey(key) {
    const hash = crypto.createHash('md5');
    hash.update(key);
    const digest = hash.digest('hex');
    return parseInt(digest.substring(0, 8), 16);
  }

  getShard(key) {
    const index = this.getShardIndex(key);
    return this.shards.get(index);
  }

  async get(key) {
    const shard = this.getShard(key);

    if (!shard.data.has(key)) {
      shard.stats.misses++;
      return null;
    }

    const entry = shard.data.get(key);

    if (Date.now() > entry.expiresAt) {
      this.deleteFromShard(shard, key);
      shard.stats.misses++;
      shard.stats.evictions++;
      return null;
    }

    this.updateAccessOrder(shard, key);
    shard.stats.hits++;

    return entry.value;
  }

  async set(key, value, options = {}) {
    const { ttl = 300000, compress = false } = options;

    let dataToStore = value;

    if (compress && this.config.enableCompression) {
      dataToStore = this.compress(JSON.stringify(value));
    }

    const size = this.calculateSize(dataToStore);
    const shard = this.getShard(key);

    if (shard.data.has(key)) {
      const oldEntry = shard.data.get(key);
      shard.stats.size -= oldEntry.size || 0;
    }

    shard.data.set(key, {
      value: dataToStore,
      size,
      expiresAt: Date.now() + ttl,
      createdAt: Date.now(),
      compressed: compress
    });

    shard.stats.size += size;
    shard.stats.sets++;

    this.updateAccessOrder(shard, key);

    await this.checkShardMemory(shard);

    return true;
  }

  updateAccessOrder(shard, key) {
    const index = shard.accessOrder.indexOf(key);
    if (index > -1) {
      shard.accessOrder.splice(index, 1);
    }
    shard.accessOrder.push(key);
  }

  deleteFromShard(shard, key) {
    if (shard.data.has(key)) {
      const entry = shard.data.get(key);
      shard.stats.size -= entry.size || 0;
      shard.data.delete(key);

      const index = shard.accessOrder.indexOf(key);
      if (index > -1) {
        shard.accessOrder.splice(index, 1);
      }
    }
  }

  async delete(key) {
    const shard = this.getShard(key);
    this.deleteFromShard(shard, key);
    return true;
  }

  async checkShardMemory(shard) {
    if (shard.stats.size > this.config.maxMemoryPerShard) {
      await this.evictFromShard(shard);
    }
  }

  async evictFromShard(shard, count = null) {
    const evictCount = count || Math.floor(shard.accessOrder.length * 0.1);

    for (let i = 0; i < evictCount && shard.accessOrder.length > 0; i++) {
      const oldestKey = shard.accessOrder.shift();
      if (shard.data.has(oldestKey)) {
        shard.data.delete(oldestKey);
        shard.stats.evictions++;
      }
    }
  }

  calculateSize(value) {
    const str = typeof value === 'string' ? value : JSON.stringify(value);
    return Buffer.byteLength(str, 'utf8');
  }

  compress(data) {
    return data;
  }

  decompress(data) {
    try {
      return JSON.parse(data);
    } catch {
      return data;
    }
  }

  async clear() {
    for (const shard of this.shards.values()) {
      shard.data.clear();
      shard.accessOrder = [];
      shard.stats = {
        hits: 0,
        misses: 0,
        sets: 0,
        evictions: 0,
        size: 0
      };
    }
  }

  getStats() {
    const total = {
      shards: this.shardCount,
      totalSize: 0,
      totalEntries: 0,
      totalHits: 0,
      totalMisses: 0,
      totalSets: 0,
      totalEvictions: 0
    };

    const shardStats = [];

    for (const [index, shard] of this.shards.entries()) {
      const entries = shard.data.size;
      const size = shard.stats.size;
      const hits = shard.stats.hits;
      const misses = shard.stats.misses;
      const totalOps = hits + misses;

      total.totalSize += size;
      total.totalEntries += entries;
      total.totalHits += hits;
      total.totalMisses += misses;
      total.totalSets += shard.stats.sets;
      total.totalEvictions += shard.stats.evictions;

      shardStats.push({
        shardId: index,
        entries,
        size,
        sizeFormatted: this.formatBytes(size),
        hits,
        misses,
        hitRate: totalOps > 0 ? ((hits / totalOps) * 100).toFixed(2) + '%' : '0%',
        sets: shard.stats.sets,
        evictions: shard.stats.evictions
      });
    }

    const totalOps = total.totalHits + total.totalMisses;

    return {
      config: this.config,
      summary: {
        ...total,
        totalSizeFormatted: this.formatBytes(total.totalSize),
        overallHitRate: totalOps > 0 ? ((total.totalHits / totalOps) * 100).toFixed(2) + '%' : '0%'
      },
      shards: shardStats
    };
  }

  formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  }
}

class CacheMemoryOptimizer {
  constructor() {
    this.config = {
      maxMemoryUsage: 500 * 1024 * 1024,
      gcThreshold: 0.8,
      enableAutoGC: true,
      gcInterval: 300000
    };

    this.usage = 0;
    this.peakUsage = 0;
    this.gcTimer = null;

    if (this.config.enableAutoGC) {
      this.startAutoGC();
    }
  }

  startAutoGC() {
    this.gcTimer = setInterval(() => {
      this.performGC();
    }, this.config.gcInterval);
  }

  stopAutoGC() {
    if (this.gcTimer) {
      clearInterval(this.gcTimer);
      this.gcTimer = null;
    }
  }

  async performGC() {
    const usageRatio = this.usage / this.config.maxMemoryUsage;

    if (usageRatio > this.config.gcThreshold) {
      console.log(`🧹 Performing garbage collection (usage: ${(usageRatio * 100).toFixed(2)}%)`);
      await this.compactMemory();
    }
  }

  async compactMemory() {
    return true;
  }

  updateUsage(delta) {
    this.usage += delta;

    if (this.usage > this.peakUsage) {
      this.peakUsage = this.usage;
    }

    if (this.usage > this.config.maxMemoryUsage) {
      console.warn(`⚠️ Memory usage exceeded limit: ${this.formatBytes(this.usage)}`);
      return false;
    }

    return true;
  }

  formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  }

  getStats() {
    return {
      current: this.formatBytes(this.usage),
      peak: this.formatBytes(this.peakUsage),
      limit: this.formatBytes(this.config.maxMemoryUsage),
      usagePercent: ((this.usage / this.config.maxMemoryUsage) * 100).toFixed(2) + '%',
      gcEnabled: this.config.enableAutoGC,
      gcThreshold: this.config.gcThreshold * 100 + '%'
    };
  }
}

class CacheCompression {
  constructor(options = {}) {
    this.config = {
      enabled: options.enabled || false,
      threshold: options.threshold || 1024,
      algorithm: options.algorithm || 'gzip'
    };
  }

  compress(data) {
    if (!this.config.enabled) return data;

    const str = typeof data === 'string' ? data : JSON.stringify(data);

    if (str.length < this.config.threshold) {
      return data;
    }

    try {
      const zlib = require('zlib');
      const compressed = zlib.gzipSync(Buffer.from(str));
      return compressed.toString('base64');
    } catch (error) {
      console.error('Compression error:', error);
      return data;
    }
  }

  decompress(data, isCompressed = false) {
    if (!this.config.enabled || !isCompressed) {
      try {
        return JSON.parse(data);
      } catch {
        return data;
      }
    }

    try {
      const zlib = require('zlib');
      const buffer = Buffer.from(data, 'base64');
      const decompressed = zlib.gunzipSync(buffer);
      return JSON.parse(decompressed.toString());
    } catch (error) {
      console.error('Decompression error:', error);
      return null;
    }
  }

  shouldCompress(data) {
    if (!this.config.enabled) return false;

    const str = typeof data === 'string' ? data : JSON.stringify(data);
    return str.length >= this.config.threshold;
  }

  getStats() {
    return this.config;
  }
}

const cacheSharding = new CacheSharding();
const cacheMemoryOptimizer = new CacheMemoryOptimizer();
const cacheCompression = new CacheCompression({ enabled: false });

module.exports = {
  CacheSharding,
  CacheMemoryOptimizer,
  CacheCompression,
  cacheSharding,
  cacheMemoryOptimizer,
  cacheCompression
};
