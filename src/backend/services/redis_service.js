const { createClient } = require('redis');
const { EventEmitter } = require('events');

class RedisPoolManager extends EventEmitter {
  constructor(options = {}) {
    super();
    this.options = {
      host: options.host || process.env.REDIS_HOST || 'localhost',
      port: options.port || parseInt(process.env.REDIS_PORT || '6379', 10),
      password: options.password || process.env.REDIS_PASSWORD || undefined,
      db: options.db || parseInt(process.env.REDIS_DB || '0', 10),
      maxRetries: options.maxRetries || 20,
      connectTimeout: options.connectTimeout || 10000,
      keepAlive: options.keepAlive || 30000,
      commandTimeout: options.commandTimeout || 5000,
      poolSize: options.poolSize || 10,
      ...options
    };
    
    this.client = null;
    this.isConnected = false;
    this.healthCheckInterval = null;
    this.stats = {
      totalConnections: 0,
      activeConnections: 0,
      idleConnections: 0,
      failedConnections: 0,
      totalCommands: 0,
      failedCommands: 0,
      averageResponseTime: 0,
      lastHealthCheck: null,
      connectionLeaks: []
    };
    this.connectionLeakThreshold = this.options.connectionLeakThreshold || 30000;
    this.commandStartTimes = new Map();
    
    this.initialize();
  }

  async initialize() {
    try {
      this.client = createClient({
        socket: {
          host: this.options.host,
          port: this.options.port,
          reconnectStrategy: (retries) => {
            if (retries > this.options.maxRetries) {
              this.emit('error', new Error('Max retries reached for Redis connection'));
              return false;
            }
            const delay = Math.min(retries * 100, 3000);
            this.emit('reconnecting', { retries, delay });
            return delay;
          },
          connectTimeout: this.options.connectTimeout,
          keepAlive: this.options.keepAlive
        },
        password: this.options.password,
        database: this.options.db,
        legacyMode: false,
        commandTimeout: this.options.commandTimeout
      });

      this.client.on('error', (err) => {
        this.stats.failedConnections++;
        this.emit('error', err);
      });

      this.client.on('connect', () => {
        this.isConnected = true;
        this.stats.totalConnections++;
        this.emit('connected');
        this.startHealthCheck();
      });

      this.client.on('ready', () => {
        this.emit('ready');
      });

      this.client.on('reconnecting', () => {
        this.emit('reconnecting');
      });

      this.client.on('end', () => {
        this.isConnected = false;
        this.stopHealthCheck();
        this.emit('disconnected');
      });

      await this.client.connect();
    } catch (error) {
      this.stats.failedConnections++;
      this.emit('error', error);
      throw error;
    }
  }

  startHealthCheck() {
    if (this.healthCheckInterval) {
      return;
    }

    const interval = parseInt(process.env.REDIS_HEALTH_CHECK_INTERVAL || '30000', 10);
    this.healthCheckInterval = setInterval(async () => {
      await this.performHealthCheck();
    }, interval);
  }

  stopHealthCheck() {
    if (this.healthCheckInterval) {
      clearInterval(this.healthCheckInterval);
      this.healthCheckInterval = null;
    }
  }

  async performHealthCheck() {
    const startTime = Date.now();
    try {
      const result = await this.client.ping();
      const responseTime = Date.now() - startTime;
      
      this.stats.lastHealthCheck = new Date();
      this.stats.activeConnections = this.client.isOpen ? 1 : 0;
      
      if (responseTime > 5000) {
        this.emit('warning', { 
          type: 'slow_health_check', 
          responseTime,
          threshold: 5000 
        });
      }

      this.emit('healthCheck', {
        status: 'healthy',
        responseTime,
        connected: this.client.isOpen
      });

      return true;
    } catch (error) {
      this.emit('healthCheck', {
        status: 'unhealthy',
        error: error.message,
        connected: false
      });
      return false;
    }
  }

  async executeCommand(command, ...args) {
    if (!this.client || !this.client.isOpen) {
      throw new Error('Redis client is not connected');
    }

    const commandKey = `${command}:${Date.now()}`;
    this.commandStartTimes.set(commandKey, Date.now());
    this.stats.totalCommands++;

    try {
      const result = await this.client[command](...args);
      
      const startTime = this.commandStartTimes.get(commandKey);
      if (startTime) {
        const duration = Date.now() - startTime;
        this.updateAverageResponseTime(duration);
        this.commandStartTimes.delete(commandKey);
        
        if (duration > this.connectionLeakThreshold) {
          this.emit('warning', {
            type: 'slow_command',
            command,
            duration,
            threshold: this.connectionLeakThreshold
          });
        }
      }

      return result;
    } catch (error) {
      this.stats.failedCommands++;
      throw error;
    }
  }

  updateAverageResponseTime(newResponseTime) {
    const totalCommands = this.stats.totalCommands;
    const currentAvg = this.stats.averageResponseTime;
    this.stats.averageResponseTime = 
      ((currentAvg * (totalCommands - 1)) + newResponseTime) / totalCommands;
  }

  async get(key) {
    return this.executeCommand('get', key);
  }

  async set(key, value, options = {}) {
    const args = [key, value];
    if (options.EX !== undefined) {
      args.push('EX', options.EX);
    }
    if (options.PX !== undefined) {
      args.push('PX', options.PX);
    }
    if (options.NX) {
      args.push('NX');
    }
    if (options.XX) {
      args.push('XX');
    }
    return this.executeCommand('set', ...args);
  }

  async del(...keys) {
    return this.executeCommand('del', ...keys);
  }

  async exists(...keys) {
    return this.executeCommand('exists', ...keys);
  }

  async hget(key, field) {
    return this.executeCommand('hget', key, field);
  }

  async hset(key, field, value) {
    return this.executeCommand('hset', key, field, value);
  }

  async hgetall(key) {
    return this.executeCommand('hgetall', key);
  }

  async expire(key, seconds) {
    return this.executeCommand('expire', key, seconds);
  }

  async ttl(key) {
    return this.executeCommand('ttl', key);
  }

  async mget(...keys) {
    return this.executeCommand('mget', ...keys);
  }

  async mset(...keyValuePairs) {
    return this.executeCommand('mset', ...keyValuePairs);
  }

  async incr(key) {
    return this.executeCommand('incr', key);
  }

  async decr(key) {
    return this.executeCommand('decr', key);
  }

  async scan(cursor, options = {}) {
    const args = [cursor];
    if (options.match) {
      args.push('MATCH', options.match);
    }
    if (options.count) {
      args.push('COUNT', options.count);
    }
    return this.executeCommand('scan', ...args);
  }

  async keys(pattern) {
    return this.executeCommand('keys', pattern);
  }

  async flushdb() {
    return this.executeCommand('flushdb');
  }

  getStats() {
    return {
      ...this.stats,
      isConnected: this.isConnected,
      pendingCommands: this.commandStartTimes.size,
      uptime: this.client?.uptime || 0
    };
  }

  async detectConnectionLeaks() {
    const now = Date.now();
    const leaks = [];

    for (const [key, startTime] of this.commandStartTimes.entries()) {
      const duration = now - startTime;
      if (duration > this.connectionLeakThreshold) {
        leaks.push({
          commandKey: key,
          duration,
          startTime: new Date(startTime)
        });
      }
    }

    if (leaks.length > 0) {
      this.stats.connectionLeaks = leaks;
      this.emit('connectionLeak', { leaks });
    }

    return leaks;
  }

  async close() {
    this.stopHealthCheck();
    
    for (const [key] of this.commandStartTimes.entries()) {
      this.commandStartTimes.delete(key);
    }

    if (this.client) {
      await this.client.quit();
    }
    
    this.isConnected = false;
    this.emit('closed');
  }
}

const redisPoolManager = new RedisPoolManager();

module.exports = redisPoolManager;
module.exports.RedisPoolManager = RedisPoolManager;
