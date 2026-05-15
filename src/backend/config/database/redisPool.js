const redis = require('ioredis');

class RedisConnectionPool {
  constructor(options = {}) {
    this.options = {
      host: options.host || process.env.REDIS_HOST || 'localhost',
      port: options.port || parseInt(process.env.REDIS_PORT || '6379', 10),
      password: options.password || process.env.REDIS_PASSWORD || undefined,
      db: options.db || parseInt(process.env.REDIS_DB || '0', 10),
      maxRetriesPerRequest: options.maxRetriesPerRequest || 3,
      retryDelayOnFailover: options.retryDelayOnFailover || 100,
      enableReadyCheck: options.enableReadyCheck !== false,
      connectTimeout: options.connectTimeout || 10000,
      commandTimeout: options.commandTimeout || 5000,
      lazyConnect: options.lazyConnect || false,
      keepAlive: options.keepAlive || 30000,
      ...options
    };

    this.client = null;
    this.isInitialized = false;
    this.stats = {
      totalConnections: 0,
      activeConnections: 0,
      failedConnections: 0,
      totalCommands: 0,
      failedCommands: 0,
      averageLatency: 0,
      lastCommandTime: null
    };
  }

  async initialize() {
    if (this.isInitialized) {
      return this;
    }

    try {
      this.client = new redis({
        host: this.options.host,
        port: this.options.port,
        password: this.options.password,
        db: this.options.db,
        maxRetriesPerRequest: this.options.maxRetriesPerRequest,
        retryStrategy: (times) => {
          if (times > 20) {
            return null;
          }
          return Math.min(times * 100, 3000);
        },
        enableReadyCheck: this.options.enableReadyCheck,
        connectTimeout: this.options.connectTimeout,
        commandTimeout: this.options.commandTimeout,
        lazyConnect: this.options.lazyConnect,
        keepAlive: this.options.keepAlive
      });

      this.client.on('error', (err) => {
        this.stats.failedConnections++;
        console.error('Redis Connection Error:', err.message);
      });

      this.client.on('connect', () => {
        this.stats.totalConnections++;
        console.log('Redis Connected');
      });

      this.client.on('ready', () => {
        console.log('Redis Ready');
      });

      this.client.on('reconnecting', () => {
        console.log('Redis Reconnecting...');
      });

      this.client.on('close', () => {
        console.log('Redis Connection Closed');
      });

      if (!this.options.lazyConnect) {
        await this.client.connect();
      }

      this.isInitialized = true;
      return this;
    } catch (error) {
      this.stats.failedConnections++;
      throw error;
    }
  }

  async execute(command, ...args) {
    if (!this.client) {
      throw new Error('Redis client not initialized');
    }

    const startTime = Date.now();
    this.stats.lastCommandTime = new Date();

    try {
      const result = await this.client[command](...args);
      const latency = Date.now() - startTime;
      
      this.stats.totalCommands++;
      this.updateAverageLatency(latency);

      return result;
    } catch (error) {
      this.stats.failedCommands++;
      throw error;
    }
  }

  updateAverageLatency(newLatency) {
    const total = this.stats.totalCommands;
    const current = this.stats.averageLatency;
    this.stats.averageLatency = ((current * (total - 1)) + newLatency) / total;
  }

  async get(key) {
    return this.execute('get', key);
  }

  async set(key, value, ttl) {
    if (ttl) {
      return this.execute('set', key, value, 'EX', ttl);
    }
    return this.execute('set', key, value);
  }

  async del(key) {
    return this.execute('del', key);
  }

  async exists(key) {
    return this.execute('exists', key);
  }

  async expire(key, seconds) {
    return this.execute('expire', key, seconds);
  }

  async ttl(key) {
    return this.execute('ttl', key);
  }

  async hget(key, field) {
    return this.execute('hget', key, field);
  }

  async hset(key, field, value) {
    return this.execute('hset', key, field, value);
  }

  async hgetall(key) {
    return this.execute('hgetall', key);
  }

  async mget(...keys) {
    return this.execute('mget', ...keys);
  }

  async mset(...keyValues) {
    return this.execute('mset', ...keyValues);
  }

  async incr(key) {
    return this.execute('incr', key);
  }

  async decr(key) {
    return this.execute('decr', key);
  }

  async ping() {
    return this.execute('ping');
  }

  async scan(cursor, pattern, count) {
    const args = [cursor];
    if (pattern) {
      args.push('MATCH', pattern);
    }
    if (count) {
      args.push('COUNT', count);
    }
    return this.execute('scan', ...args);
  }

  async keys(pattern) {
    return this.execute('keys', pattern);
  }

  async flushdb() {
    return this.execute('flushdb');
  }

  getStats() {
    return {
      ...this.stats,
      isConnected: this.client?.status === 'ready',
      clientStatus: this.client?.status
    };
  }

  async healthCheck() {
    try {
      const startTime = Date.now();
      await this.ping();
      const latency = Date.now() - startTime;
      
      return {
        status: 'healthy',
        latency,
        timestamp: new Date().toISOString()
      };
    } catch (error) {
      return {
        status: 'unhealthy',
        error: error.message,
        timestamp: new Date().toISOString()
      };
    }
  }

  async close() {
    if (this.client) {
      await this.client.quit();
      this.isInitialized = false;
    }
  }
}

module.exports = RedisConnectionPool;
