const Redis = require('ioredis');
const config = require('../config/messageQueue');

class StreamConnectionManager {
  constructor() {
    this.client = null;
    this.connections = new Map();
    this.isConnected = false;
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 10;
    this.eventHandlers = new Map();
  }

  async connect() {
    if (this.client && this.isConnected) {
      return this.client;
    }

    const redisConfig = {
      host: config.redis.host,
      port: config.redis.port,
      password: config.redis.password,
      db: config.redis.db,
      keyPrefix: config.redis.keyPrefix,
      connectTimeout: config.redis.connectTimeout,
      commandTimeout: config.redis.commandTimeout,
      retryStrategy: config.redis.retryStrategy,
      maxRetriesPerRequest: config.redis.maxRetriesPerRequest,
      enableReadyCheck: config.redis.enableReadyCheck,
      enableOfflineQueue: config.redis.enableOfflineQueue,
      lazyConnect: false
    };

    this.client = new Redis(redisConfig);

    this.client.on('connect', () => {
      console.log('[StreamConnection] Connected to Redis');
      this.isConnected = true;
      this.reconnectAttempts = 0;
      this.emit('connect');
    });

    this.client.on('ready', () => {
      console.log('[StreamConnection] Redis ready');
      this.emit('ready');
    });

    this.client.on('error', (err) => {
      console.error('[StreamConnection] Redis error:', err.message);
      this.emit('error', err);
    });

    this.client.on('close', () => {
      console.log('[StreamConnection] Redis connection closed');
      this.isConnected = false;
      this.emit('close');
    });

    this.client.on('reconnecting', () => {
      this.reconnectAttempts++;
      console.log(`[StreamConnection] Reconnecting... Attempt ${this.reconnectAttempts}`);
      this.emit('reconnecting', { attempt: this.reconnectAttempts });
    });

    await this.client.connect().catch(err => {
      console.error('[StreamConnection] Initial connection failed:', err.message);
      throw err;
    });

    return this.client;
  }

  async disconnect() {
    if (this.client) {
      await this.client.quit();
      this.client = null;
      this.isConnected = false;
      console.log('[StreamConnection] Disconnected from Redis');
    }
  }

  getClient() {
    if (!this.client) {
      throw new Error('Redis client not initialized. Call connect() first.');
    }
    return this.client;
  }

  async healthCheck() {
    try {
      if (!this.client) {
        return { healthy: false, error: 'Client not initialized' };
      }
      const start = Date.now();
      await this.client.ping();
      const latency = Date.now() - start;
      return {
        healthy: true,
        latency,
        connected: this.isConnected
      };
    } catch (error) {
      return {
        healthy: false,
        error: error.message,
        connected: false
      };
    }
  }

  async getStreamInfo(streamKey) {
    try {
      const client = this.getClient();
      const info = await client.xinfoStream(streamKey);
      return {
        stream: streamKey,
        length: info[1][1],
        firstEntry: info[3][1],
        lastEntry: info[5][1],
        groups: info[7][1],
        lastGeneratedId: info[9][1]
      };
    } catch (error) {
      if (error.message.includes('no such key')) {
        return null;
      }
      throw error;
    }
  }

  async getConsumerGroupInfo(streamKey, groupName) {
    try {
      const client = this.getClient();
      const info = await client.xinfoGroups(streamKey);
      const group = info.find(g => g[1] === groupName);
      if (!group) {
        return null;
      }
      return {
        name: group[1],
        consumers: group[3],
        pending: group[5],
        lastDeliveredId: group[7],
        entriesRead: group[9],
        lag: group[11]
      };
    } catch (error) {
      if (error.message.includes('no such key') || error.message.includes('BUSYGROUP')) {
        return null;
      }
      throw error;
    }
  }

  on(event, handler) {
    if (!this.eventHandlers.has(event)) {
      this.eventHandlers.set(event, []);
    }
    this.eventHandlers.get(event).push(handler);
  }

  emit(event, ...args) {
    const handlers = this.eventHandlers.get(event);
    if (handlers) {
      handlers.forEach(handler => {
        try {
          handler(...args);
        } catch (error) {
          console.error(`[StreamConnection] Event handler error:`, error);
        }
      });
    }
  }

  async ensureStream(streamKey, maxLen = config.streams.maxLen) {
    const client = this.getClient();
    await client.xadd(streamKey, 'MAXLEN', '~', maxLen.toString(), '*', 'init', 'true');
  }

  async ensureConsumerGroup(streamKey, groupName, startId = '0') {
    const client = this.getClient();
    try {
      await client.xgroup('CREATE', streamKey, groupName, startId, 'MKSTREAM');
      console.log(`[StreamConnection] Created consumer group: ${groupName} for stream: ${streamKey}`);
    } catch (error) {
      if (!error.message.includes('BUSYGROUP')) {
        throw error;
      }
    }
  }

  async cleanup() {
    const client = this.getClient();
    const pattern = `${config.redis.keyPrefix}hjtpx:streams:*`;

    const keys = await client.keys(pattern);
    if (keys.length > 0) {
      const streamsWithoutPrefix = keys.map(k => k.replace(config.redis.keyPrefix, ''));
      for (const key of streamsWithoutPrefix) {
        await client.del(key);
      }
      console.log(`[StreamConnection] Cleaned up ${keys.length} stream keys`);
    }
  }
}

const streamConnectionManager = new StreamConnectionManager();

module.exports = streamConnectionManager;
