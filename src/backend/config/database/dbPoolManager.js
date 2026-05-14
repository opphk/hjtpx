const { Pool } = require('pg');

class DatabasePoolManager {
  constructor() {
    this.pool = null;
    this.config = {
      min: parseInt(process.env.DB_POOL_MIN) || 5,
      max: parseInt(process.env.DB_POOL_MAX) || 20,
      idleTimeoutMillis: parseInt(process.env.DB_IDLE_TIMEOUT) || 30000,
      connectionTimeoutMillis: parseInt(process.env.DB_CONNECTION_TIMEOUT) || 5000,
      statement_timeout: parseInt(process.env.DB_STATEMENT_TIMEOUT) || 30000,
      allowExitOnIdle: false,
      keepAlive: true,
      keepAliveInitialDelayMillis: 10000,
      maxUses: parseInt(process.env.DB_MAX_USES) || 7500,
      idleTimeout: parseInt(process.env.DB_IDLE_TIMEOUT) || 30000,
      parserFallback: true,
      types: {
        getTypeParser: () => (val) => val
      }
    };
    this.stats = {
      queries: 0,
      slowQueries: 0,
      errors: 0,
      avgQueryTime: 0,
      totalQueryTime: 0,
      connectionsAcquired: 0,
      connectionsReleased: 0,
      connectionsCreated: 0,
      connectionsDestroyed: 0
    };
    this.queryTimes = [];
    this.slowQueryThreshold = parseInt(process.env.SLOW_QUERY_THRESHOLD) || 100;
    this.healthCheckInterval = parseInt(process.env.DB_HEALTH_CHECK_INTERVAL) || 60000;
    this.leakDetectionThreshold = parseInt(process.env.DB_LEAK_DETECTION_THRESHOLD) || 30000;
    this.activeConnections = new Map();
    this.connectionLeaks = [];
    this.healthCheckTimer = null;
    this.metricsHistory = [];
    this.maxMetricsHistory = 100;
  }

  initialize() {
    if (this.pool) {
      return this.pool;
    }

    this.pool = new Pool({
      ...this.config,
      host: process.env.DB_HOST || 'localhost',
      port: parseInt(process.env.DB_PORT) || 5432,
      database: process.env.DB_NAME || 'hjtpx',
      user: process.env.DB_USER || 'postgres',
      password: process.env.DB_PASSWORD || ''
    });

    if (process.env.NODE_ENV === 'production') {
      this.pool.ssl = {
        rejectUnauthorized: process.env.DB_SSL_REJECT_UNAUTHORIZED !== 'false',
        ...(process.env.DB_SSL_CERT && { cert: process.env.DB_SSL_CERT }),
        ...(process.env.DB_SSL_KEY && { key: process.env.DB_SSL_KEY }),
        ...(process.env.DB_SSL_CA && { ca: process.env.DB_SSL_CA })
      };
    }

    this.setupEventListeners();
    this.startHealthCheck();
    this.startLeakDetection();

    console.log('Database pool initialized with optimized configuration');
    return this.pool;
  }

  setupEventListeners() {
    this.pool.on('error', (err) => {
      console.error('Unexpected database pool error:', err);
      this.stats.errors++;
      this.recordMetric('error', { message: err.message });
    });

    this.pool.on('connect', (client) => {
      console.log('New client connected to database pool');
      this.stats.connectionsCreated++;
      this.trackConnection(client);
    });

    this.pool.on('acquire', () => {
      this.stats.connectionsAcquired++;
    });

    this.pool.on('remove', () => {
      this.stats.connectionsReleased++;
    });
  }

  trackConnection(client) {
    const connectionId = client.processID || Date.now();
    const connectionInfo = {
      id: connectionId,
      acquiredAt: Date.now(),
      lastUsed: Date.now(),
      queryCount: 0,
      released: false
    };
    this.activeConnections.set(connectionId, connectionInfo);
  }

  async getClient() {
    const client = await this.pool.connect();
    const connectionId = client.processID || Date.now();

    this.activeConnections.set(connectionId, {
      id: connectionId,
      client,
      acquiredAt: Date.now(),
      lastUsed: Date.now(),
      queryCount: 0,
      released: false
    });

    const originalQuery = client.query.bind(client);
    const originalRelease = client.release.bind(client);

    client.query = async (...args) => {
      this.activeConnections.get(connectionId).lastUsed = Date.now();
      this.activeConnections.get(connectionId).queryCount++;
      return originalQuery(...args);
    };

    client.release = () => {
      this.activeConnections.get(connectionId).released = true;
      this.activeConnections.get(connectionId).releasedAt = Date.now();
      return originalRelease();
    };

    return client;
  }

  startHealthCheck() {
    if (this.healthCheckTimer) {
      clearInterval(this.healthCheckTimer);
    }

    this.healthCheckTimer = setInterval(async () => {
      try {
        const healthStatus = await this.performHealthCheck();
        if (!healthStatus.healthy) {
          console.warn('Database health check failed:', healthStatus.error);
          this.recordMetric('health_check_failed', { error: healthStatus.error });
        } else {
          console.log('Database health check passed, response time:', healthStatus.responseTime, 'ms');
        }
      } catch (error) {
        console.error('Health check error:', error);
      }
    }, this.healthCheckInterval);

    console.log(`Health check started with interval: ${this.healthCheckInterval}ms`);
  }

  async performHealthCheck() {
    const start = Date.now();
    try {
      const result = await this.pool.query('SELECT 1 as health, now() as timestamp');
      const duration = Date.now() - start;

      await this.verifyConnectionIntegrity();

      return {
        healthy: true,
        responseTime: duration,
        timestamp: new Date().toISOString(),
        connectionStatus: this.getPoolStats()
      };
    } catch (error) {
      return {
        healthy: false,
        error: error.message,
        timestamp: new Date().toISOString()
      };
    }
  }

  async verifyConnectionIntegrity() {
    const clients = await this.pool.connect().catch(() => []);
    if (clients) {
      try {
        await clients.query('SELECT 1');
      } finally {
        clients.release();
      }
    }
  }

  startLeakDetection() {
    setInterval(() => {
      this.detectConnectionLeaks();
    }, this.leakDetectionThreshold);
  }

  detectConnectionLeaks() {
    const now = Date.now();
    const threshold = this.leakDetectionThreshold;
    const leaks = [];

    this.activeConnections.forEach((connection, id) => {
      if (!connection.released) {
        const holdTime = now - connection.acquiredAt;
        if (holdTime > threshold) {
          leaks.push({
            connectionId: id,
            holdTime,
            queryCount: connection.queryCount,
            acquiredAt: new Date(connection.acquiredAt).toISOString()
          });
        }
      }
    });

    if (leaks.length > 0) {
      console.warn('Connection leaks detected:', leaks);
      this.connectionLeaks.push(...leaks);
      this.recordMetric('connection_leak', { leaks });

      if (this.connectionLeaks.length > 100) {
        this.connectionLeaks = this.connectionLeaks.slice(-100);
      }
    }

    return leaks;
  }

  async query(text, params, options = {}) {
    const start = Date.now();
    const trackStats = options.trackStats !== false;

    try {
      const result = await this.pool.query(text, params);
      const duration = Date.now() - start;

      if (trackStats) {
        this.stats.queries++;
        this.stats.totalQueryTime += duration;
        this.queryTimes.push(duration);

        if (this.queryTimes.length > 100) {
          this.queryTimes.shift();
        }

        this.stats.avgQueryTime =
          this.queryTimes.reduce((a, b) => a + b, 0) / this.queryTimes.length;

        if (duration > this.slowQueryThreshold) {
          this.stats.slowQueries++;
          console.warn(`Slow query (${duration}ms): ${text.substring(0, 100)}`);
          this.recordMetric('slow_query', { duration, query: text.substring(0, 100) });
        }
      }

      return result;
    } catch (error) {
      this.stats.errors++;
      this.recordMetric('query_error', { error: error.message, query: text.substring(0, 100) });
      throw error;
    }
  }

  async transaction(callback) {
    const client = await this.getClient();
    try {
      await client.query('BEGIN');
      const result = await callback(client);
      await client.query('COMMIT');
      return result;
    } catch (error) {
      await client.query('ROLLBACK');
      throw error;
    } finally {
      client.release();
    }
  }

  async batchQuery(queries) {
    const client = await this.getClient();
    try {
      await client.query('BEGIN');
      const results = await Promise.all(
        queries.map(async ({ query, params }) => {
          const result = await client.query(query, params);
          return result.rows;
        })
      );
      await client.query('COMMIT');
      return results;
    } catch (error) {
      await client.query('ROLLBACK');
      throw error;
    } finally {
      client.release();
    }
  }

  async healthCheck() {
    return this.performHealthCheck();
  }

  getPoolStats() {
    return {
      total: this.pool.totalCount,
      idle: this.pool.idleCount,
      busy: this.pool.totalCount - this.pool.idleCount,
      waiting: this.pool.waitingCount,
      max: this.config.max,
      min: this.config.min,
      utilizationPercent: ((this.pool.totalCount - this.pool.idleCount) / this.config.max * 100).toFixed(2) + '%'
    };
  }

  getQueryStats() {
    const hitRate = this.stats.queries > 0
      ? ((1 - this.stats.slowQueries / this.stats.queries) * 100).toFixed(2) + '%'
      : '100%';
    return {
      ...this.stats,
      avgQueryTime: Math.round(this.stats.avgQueryTime * 100) / 100,
      hitRate,
      queryTimes: [...this.queryTimes]
    };
  }

  getConnectionLeaks() {
    return [...this.connectionLeaks];
  }

  getMetrics() {
    return {
      poolStats: this.getPoolStats(),
      queryStats: this.getQueryStats(),
      activeConnections: this.activeConnections.size,
      connectionLeaks: this.connectionLeaks.length,
      metricsHistory: [...this.metricsHistory]
    };
  }

  recordMetric(type, data) {
    const metric = {
      type,
      timestamp: new Date().toISOString(),
      ...data
    };
    this.metricsHistory.push(metric);

    if (this.metricsHistory.length > this.maxMetricsHistory) {
      this.metricsHistory.shift();
    }
  }

  resetStats() {
    this.stats = {
      queries: 0,
      slowQueries: 0,
      errors: 0,
      avgQueryTime: 0,
      totalQueryTime: 0,
      connectionsAcquired: 0,
      connectionsReleased: 0,
      connectionsCreated: 0,
      connectionsDestroyed: 0
    };
    this.queryTimes = [];
  }

  clearLeakHistory() {
    this.connectionLeaks = [];
    console.log('Connection leak history cleared');
  }

  async close() {
    if (this.healthCheckTimer) {
      clearInterval(this.healthCheckTimer);
      this.healthCheckTimer = null;
    }

    if (this.pool) {
      await this.pool.end();
      this.pool = null;
      this.activeConnections.clear();
      console.log('Database pool closed');
    }
  }

  async refreshPool() {
    console.log('Refreshing database pool...');
    const oldPool = this.pool;
    this.pool = null;
    this.initialize();

    if (oldPool) {
      await oldPool.end().catch(err => console.error('Error closing old pool:', err));
    }

    console.log('Database pool refreshed');
  }

  setConfig(newConfig) {
    this.config = { ...this.config, ...newConfig };
    console.log('Pool configuration updated:', this.config);
  }
}

const dbPoolManager = new DatabasePoolManager();

if (process.env.NODE_ENV !== 'test') {
  dbPoolManager.initialize();
}

module.exports = dbPoolManager;
