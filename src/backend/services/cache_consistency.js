const redisClient = require('../../../config/redis/client');

const generateUUID = () => {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = Math.random() * 16 | 0;
    const v = c === 'x' ? r : (r & 0x3 | 0x8);
    return v.toString(16);
  });
};

const CACHE_KEYS = {
  VERSION: 'cache:version:',
  LOCK: 'cache:lock:',
  TRANSACTION: 'cache:tx:',
  WATCH: 'cache:watch:',
  EVENT: 'cache:event:'
};

class CacheConsistency {
  constructor() {
    this.subscribers = new Map();
    this.versionListeners = new Map();
    this.transactionLog = [];
    this.maxTransactionLogSize = 1000;
    this.defaultLockTimeout = 10000;
    this.defaultRetryDelay = 100;
    this.defaultRetryCount = 50;
    this.isConnected = false;

    this.init();
  }

  async init() {
    try {
      if (!redisClient.isOpen) {
        await redisClient.connect();
      }
      this.isConnected = true;
      this.setupPubSub();
    } catch (error) {
      console.error('Cache Consistency init error:', error);
      this.isConnected = false;
    }
  }

  async setupPubSub() {
    try {
      const subscriber = redisClient.duplicate();
      await subscriber.connect();

      subscriber.on('message', (channel, message) => {
        this.handleMessage(channel, message);
      });

      await subscriber.subscribe(CACHE_KEYS.EVENT + '*', () => {});
    } catch (error) {
      console.error('PubSub setup error:', error);
    }
  }

  handleMessage(channel, message) {
    try {
      const data = JSON.parse(message);
      const key = channel.replace(CACHE_KEYS.EVENT, '');

      if (this.subscribers.has(key)) {
        this.subscribers.get(key).forEach(callback => callback(data));
      }

      if (this.versionListeners.has(key)) {
        this.versionListeners.get(key).forEach(callback => callback(data));
      }
    } catch (error) {
      console.error('Message handling error:', error);
    }
  }

  async getVersion(key) {
    if (!this.isConnected) return 0;

    try {
      const versionKey = `${CACHE_KEYS.VERSION}${key}`;
      const version = await redisClient.get(versionKey);
      return parseInt(version) || 0;
    } catch (error) {
      console.error('Get version error:', error);
      return 0;
    }
  }

  async incrementVersion(key) {
    if (!this.isConnected) return 0;

    try {
      const versionKey = `${CACHE_KEYS.VERSION}${key}`;
      const newVersion = await redisClient.incr(versionKey);

      await this.publishVersionChange(key, newVersion);

      return newVersion;
    } catch (error) {
      console.error('Increment version error:', error);
      return 0;
    }
  }

  async publishVersionChange(key, version) {
    try {
      const eventKey = `${CACHE_KEYS.EVENT}${key}`;
      await redisClient.publish(
        eventKey,
        JSON.stringify({
          type: 'version_change',
          key,
          version,
          timestamp: Date.now()
        })
      );
    } catch (error) {
      console.error('Publish version change error:', error);
    }
  }

  async acquireLock(key, options = {}) {
    if (!this.isConnected) {
      return { acquired: false, reason: 'not_connected' };
    }

    const {
      timeout = this.defaultLockTimeout,
      retries = this.defaultRetryCount,
      delay = this.defaultRetryDelay,
      owner = generateUUID()
    } = options;
    
    const lockKey = `${CACHE_KEYS.LOCK}${key}`;
    const lockValue = `${owner}:${Date.now()}:${generateUUID()}`;

    for (let i = 0; i < retries; i++) {
      try {
        const result = await redisClient.set(lockKey, lockValue, {
          NX: true,
          PX: timeout
        });

        if (result === 'OK') {
          return {
            acquired: true,
            value: lockValue,
            owner,
            timeout,
            acquiredAt: Date.now()
          };
        }
      } catch (error) {
        console.error(`Lock acquisition attempt ${i + 1} failed:`, error);
      }

      await this.sleep(delay);
    }

    return {
      acquired: false,
      reason: 'max_retries_exceeded',
      attempts: retries
    };
  }

  async releaseLock(key, lockValue) {
    if (!this.isConnected) return false;

    const lockKey = `${CACHE_KEYS.LOCK}${key}`;

    try {
      const script = `
        if redis.call("get", KEYS[1]) == ARGV[1] then
          return redis.call("del", KEYS[1])
        else
          return 0
        end
      `;

      const result = await redisClient.eval(script, {
        keys: [lockKey],
        arguments: [lockValue]
      });

      return result === 1;
    } catch (error) {
      console.error('Release lock error:', error);
      return false;
    }
  }

  async extendLock(key, lockValue, extension = null) {
    if (!this.isConnected) return false;

    const lockKey = `${CACHE_KEYS.LOCK}${key}`;
    const timeout = extension || this.defaultLockTimeout;

    try {
      const script = `
        if redis.call("get", KEYS[1]) == ARGV[1] then
          return redis.call("pexpire", KEYS[1], ARGV[2])
        else
          return 0
        end
      `;

      const result = await redisClient.eval(script, {
        keys: [lockKey],
        arguments: [lockValue, timeout]
      });

      return result === 1;
    } catch (error) {
      console.error('Extend lock error:', error);
      return false;
    }
  }

  async isLocked(key) {
    if (!this.isConnected) return false;

    const lockKey = `${CACHE_KEYS.LOCK}${key}`;

    try {
      const result = await redisClient.exists(lockKey);
      return result === 1;
    } catch (error) {
      console.error('Check lock error:', error);
      return false;
    }
  }

  async withLock(key, callback, options = {}) {
    const lock = await this.acquireLock(key, options);

    if (!lock.acquired) {
      throw new Error(`Failed to acquire lock for key: ${key} (${lock.reason})`);
    }

    try {
      const result = await callback();
      return result;
    } finally {
      await this.releaseLock(key, lock.value);
    }
  }

  async startTransaction() {
    const transactionId = generateUUID();

    return {
      id: transactionId,
      operations: [],
      startedAt: Date.now(),

      async get(key) {
        this.operations.push({ type: 'get', key, timestamp: Date.now() });
        return null;
      },

      async set(key, value, ttl) {
        this.operations.push({ type: 'set', key, value, ttl, timestamp: Date.now() });
        return this;
      },

      async delete(key) {
        this.operations.push({ type: 'delete', key, timestamp: Date.now() });
        return this;
      },

      async commit(cacheService) {
        const log = {
          id: transactionId,
          operations: this.operations,
          startedAt: this.startedAt,
          committedAt: Date.now(),
          status: 'success'
        };

        try {
          for (const op of this.operations) {
            switch (op.type) {
              case 'set':
                await cacheService.set(op.key, op.value, { ttl: op.ttl });
                break;
              case 'delete':
                await cacheService.delete(op.key);
                break;
            }
          }

          this.logTransaction(log);
          return { success: true, log };
        } catch (error) {
          log.status = 'failed';
          log.error = error.message;
          this.logTransaction(log);
          return { success: false, error: error.message, log };
        }
      },

      rollback() {
        const log = {
          id: transactionId,
          operations: this.operations,
          startedAt: this.startedAt,
          rolledBackAt: Date.now(),
          status: 'rolled_back'
        };

        this.logTransaction(log);
        return { success: true, log };
      }
    };
  }

  logTransaction(log) {
    this.transactionLog.push(log);

    if (this.transactionLog.length > this.maxTransactionLogSize) {
      this.transactionLog = this.transactionLog.slice(-this.maxTransactionLogSize);
    }
  }

  getTransactionLog(limit = 100) {
    return this.transactionLog.slice(-limit);
  }

  async watch(keys, callback) {
    if (!this.isConnected) {
      return await callback();
    }

    const watchKey = `${CACHE_KEYS.WATCH}${generateUUID()}`;

    try {
      for (const key of keys) {
        await redisClient.set(watchKey, '1', { EX: 60 });
      }

      const result = await callback();

      await redisClient.del(watchKey);

      return result;
    } catch (error) {
      await redisClient.del(watchKey);
      throw error;
    }
  }

  async compareAndSet(key, expectedValue, newValue, options = {}) {
    const { ttl = 300 } = options;

    return await this.withLock(key, async () => {
      const currentVersion = await this.getVersion(key);
      const currentValue = await redisClient.get(key);

      if (currentValue !== expectedValue) {
        return {
          success: false,
          reason: 'value_mismatch',
          currentValue,
          currentVersion
        };
      }

      await redisClient.set(key, newValue, { EX: ttl });
      await this.incrementVersion(key);

      return {
        success: true,
        previousValue: currentValue,
        newVersion: currentVersion + 1
      };
    });
  }

  async multiVersionIncrement(key, delta = 1) {
    return await this.withLock(key, async () => {
      const currentValue = await redisClient.get(key);
      const currentNum = parseInt(currentValue) || 0;
      const newValue = currentNum + delta;

      await redisClient.set(key, newValue.toString());
      const newVersion = await this.incrementVersion(key);

      return {
        previousValue: currentNum,
        newValue,
        newVersion
      };
    });
  }

  subscribe(key, callback) {
    if (!this.subscribers.has(key)) {
      this.subscribers.set(key, new Set());
    }

    this.subscribers.get(key).add(callback);

    return () => {
      const subscribers = this.subscribers.get(key);
      if (subscribers) {
        subscribers.delete(callback);
      }
    };
  }

  onVersionChange(key, callback) {
    if (!this.versionListeners.has(key)) {
      this.versionListeners.set(key, new Set());
    }

    this.versionListeners.get(key).add(callback);

    return () => {
      const listeners = this.versionListeners.get(key);
      if (listeners) {
        listeners.delete(callback);
      }
    };
  }

  async invalidateAcrossCluster(key, value = null) {
    try {
      await this.incrementVersion(key);

      const eventKey = `${CACHE_KEYS.EVENT}${key}`;
      await redisClient.publish(
        eventKey,
        JSON.stringify({
          type: 'invalidate',
          key,
          value,
          timestamp: Date.now(),
          invalidatedBy: process.pid
        })
      );

      return true;
    } catch (error) {
      console.error('Invalidate across cluster error:', error);
      return false;
    }
  }

  async optimisticUpdate(key, updateFn, options = {}) {
    const { maxRetries = 3, retryDelay = 100 } = options;

    for (let attempt = 0; attempt < maxRetries; attempt++) {
      const currentVersion = await this.getVersion(key);
      const currentValue = await redisClient.get(key);

      try {
        const newValue = await updateFn(currentValue);

        const result = await this.compareAndSet(key, currentValue, newValue, { ttl: options.ttl });

        if (result.success) {
          return result;
        }
      } catch (error) {
        if (attempt === maxRetries - 1) {
          throw error;
        }
      }

      await this.sleep(retryDelay * (attempt + 1));
    }

    throw new Error(`Optimistic update failed after ${maxRetries} attempts`);
  }

  async cacheAside(key, readFn, writeFn, options = {}) {
    const cached = await redisClient.get(key);

    if (cached !== null) {
      return {
        source: 'cache',
        data: JSON.parse(cached),
        version: await this.getVersion(key)
      };
    }

    const data = await readFn();

    if (data !== null && data !== undefined) {
      await this.withLock(key, async () => {
        await redisClient.set(key, JSON.stringify(data), { EX: options.ttl || 300 });
        await this.incrementVersion(key);
      });
    }

    return {
      source: 'database',
      data,
      version: await this.getVersion(key)
    };
  }

  async writeThrough(key, value, options = {}) {
    const { ttl = 300, writeFn = null } = options;

    return await this.withLock(key, async () => {
      await redisClient.set(key, JSON.stringify(value), { EX: ttl });
      const newVersion = await this.incrementVersion(key);

      if (writeFn) {
        await writeFn(value);
      }

      return {
        success: true,
        version: newVersion,
        cached: true
      };
    });
  }

  async writeBehind(key, value, options = {}) {
    const { ttl = 300, writeFn = null } = options;

    await redisClient.set(key, JSON.stringify(value), { EX: ttl });
    await this.incrementVersion(key);

    if (writeFn) {
      setImmediate(async () => {
        try {
          await writeFn(value);
        } catch (error) {
          console.error('Write behind failed:', error);
        }
      });
    }

    return {
      success: true,
      version: await this.getVersion(key),
      cached: true
    };
  }

  async getConsistencyStats() {
    return {
      connected: this.isConnected,
      transactionLogSize: this.transactionLog.length,
      subscribersCount: this.subscribers.size,
      versionListenersCount: this.versionListeners.size,
      recentTransactions: this.getTransactionLog(10)
    };
  }

  sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  async healthCheck() {
    try {
      if (!this.isConnected) {
        return { healthy: false, reason: 'not_connected' };
      }

      const pong = await redisClient.ping();
      if (pong !== 'PONG') {
        return { healthy: false, reason: 'ping_failed' };
      }

      return { healthy: true };
    } catch (error) {
      return { healthy: false, reason: error.message };
    }
  }
}

const cacheConsistency = new CacheConsistency();

module.exports = cacheConsistency;
module.exports.CACHE_KEYS = CACHE_KEYS;
