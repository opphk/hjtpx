const { Pool } = require('pg');
const EventEmitter = require('events');
const fs = require('fs');
const path = require('path');

class DatabasePoolManager extends EventEmitter {
  constructor() {
    super();
    this.pool = null;
    this.isProduction = process.env.NODE_ENV === 'production';
    this.config = this._getOptimizedConfig();
    this.queryLogFile = path.join(__dirname, '../../../logs/query.log');
    this.checkedOutClients = new Map();
    this._initializeLogging();
    this.stats = this._initializeStats();
    this.queryTimes = [];
    this.slowQueryThreshold = parseInt(process.env.SLOW_QUERY_THRESHOLD) || 100;
    this.healthCheckInterval = null;
    this.leakCheckInterval = null;
    this.reapingInterval = null;
    this.initialized = false;
    this.lastHealthCheck = null;
    this.healthCheckHistory = [];
  }

  _initializeStats() {
    return {
      queries: 0,
      slowQueries: 0,
      errors: 0,
      avgQueryTime: 0,
      totalQueryTime: 0,
      maxQueryTime: 0,
      minQueryTime: Infinity,
      connectionLeaks: 0,
      connectionLeakEvents: [],
      healthCheckFailures: 0,
      connectionErrors: 0,
      lastSuccessfulQuery: null,
      lastError: null
    };
  }

  _initializeLogging() {
    if (!fs.existsSync(path.dirname(this.queryLogFile))) {
      fs.mkdirSync(path.dirname(this.queryLogFile), { recursive: true });
    }
  }

  _getOptimizedConfig() {
    const cpuCores = require('os').cpus().length;
    const isProduction = process.env.NODE_ENV === 'production';
    
    const baseConfig = {
      host: process.env.DB_HOST || 'localhost',
      port: parseInt(process.env.DB_PORT) || 5432,
      database: process.env.DB_NAME || 'hjtpx',
      user: process.env.DB_USER || 'postgres',
      password: process.env.DB_PASSWORD || '',
      allowExitOnIdle: false
    };

    let poolSize;
    if (isProduction) {
      const envMax = parseInt(process.env.DB_POOL_MAX);
      const envMin = parseInt(process.env.DB_POOL_MIN);
      
      poolSize = {
        min: envMin || Math.max(5, Math.floor(cpuCores * 1.5)),
        max: envMax || Math.min(50, cpuCores * 10)
      };
    } else {
      poolSize = {
        min: parseInt(process.env.DB_POOL_MIN) || 2,
        max: parseInt(process.env.DB_POOL_MAX) || 10
      };
    }

    const connectionTimeout = parseInt(process.env.DB_CONNECTION_TIMEOUT);
    const idleTimeout = parseInt(process.env.DB_IDLE_TIMEOUT);
    const statementTimeout = parseInt(process.env.DB_STATEMENT_TIMEOUT);

    return {
      ...baseConfig,
      min: poolSize.min,
      max: poolSize.max,
      idleTimeoutMillis: idleTimeout || (isProduction ? 60000 : 30000),
      connectionTimeoutMillis: connectionTimeout || (isProduction ? 10000 : 5000),
      statement_timeout: statementTimeout || (isProduction ? 60000 : 30000),
      keepAlive: true,
      keepAliveInitialDelayMillis: 30000,
      keepAliveInitialDelay: 30000,
      application_name: process.env.APP_NAME || 'hjtpx',
      ...(isProduction ? {
        ssl: this._getSSLConfig(),
        types: {
          getTypeParser: (oid) => {
            if (oid === 1082) return (val) => val; 
            if (oid === 1114) return (val) => val;
            if (oid === 1184) return (val) => val;
            return null;
          }
        }
      } : {})
    };
  }

  _getSSLConfig() {
    const sslConfig = {
      rejectUnauthorized: process.env.DB_SSL_REJECT_UNAUTHORIZED !== 'false'
    };
    
    if (process.env.DB_SSL_CERT) {
      sslConfig.cert = process.env.DB_SSL_CERT;
    }
    if (process.env.DB_SSL_KEY) {
      sslConfig.key = process.env.DB_SSL_KEY;
    }
    if (process.env.DB_SSL_CA) {
      sslConfig.ca = process.env.DB_SSL_CA;
    }
    
    return sslConfig;
  }

  initialize() {
    if (this.pool) {
      return this.pool;
    }

    this.pool = new Pool(this.config);

    this._setupEventListeners();
    this._startHealthCheck();
    this._startLeakDetection();
    this._startConnectionReaping();
    this.initialized = true;

    console.log('Database pool initialized with optimized configuration');
    console.log(`Pool config: min=${this.config.min}, max=${this.config.max}, idleTimeout=${this.config.idleTimeoutMillis}ms`);
    
    if (this.isProduction) {
      console.log('Production mode: SSL enabled, detailed logging disabled');
    }
    
    return this.pool;
  }

  _setupEventListeners() {
    this.pool.on('error', (err, client) => {
      console.error('Unexpected database pool error:', err);
      this.stats.errors++;
      this.stats.connectionErrors++;
      this.stats.lastError = err.message;
      this.emit('poolError', { error: err, timestamp: new Date().toISOString() });
    });

    this.pool.on('connect', (client) => {
      if (this.isProduction) {
        console.log('New client connected to database pool');
      }
      this.emit('clientConnected', { timestamp: new Date().toISOString() });
    });

    this.pool.on('acquire', (client) => {
      this.emit('clientAcquired', { timestamp: new Date().toISOString() });
    });

    this.pool.on('remove', (client) => {
      if (this.isProduction) {
        console.log('Client removed from pool');
      }
      this.emit('clientRemoved', { timestamp: new Date().toISOString() });
    });
  }

  _logQuery(query, params, duration) {
    if (!this.queryLogFile) return;
    
    const logEntry = {
      timestamp: new Date().toISOString(),
      query: query.substring(0, 1000),
      params: params ? params.map(p => 
        typeof p === 'string' && p.length > 100 ? p.substring(0, 100) + '...' : p
      ) : null,
      duration: `${duration}ms`
    };
    
    try {
      fs.appendFileSync(this.queryLogFile, JSON.stringify(logEntry) + '\n');
    } catch (e) {
      console.error('Failed to write query log:', e);
    }
  }

  _logSlowQuery(query, duration) {
    const slowQueryLogFile = this.queryLogFile.replace('query.log', 'slow-query.log');
    const logEntry = {
      timestamp: new Date().toISOString(),
      query: query.substring(0, 500),
      duration: `${duration}ms`,
      threshold: this.slowQueryThreshold
    };
    
    try {
      fs.appendFileSync(slowQueryLogFile, JSON.stringify(logEntry) + '\n');
    } catch (e) {
      console.error('Failed to write slow query log:', e);
    }
  }

  _startHealthCheck() {
    const interval = parseInt(process.env.DB_HEALTH_CHECK_INTERVAL) || 30000;
    
    this.healthCheckInterval = setInterval(async () => {
      const health = await this.healthCheck();
      this.lastHealthCheck = health;
      this.emit('healthCheck', health);
      
      this.healthCheckHistory.push(health);
      if (this.healthCheckHistory.length > 100) {
        this.healthCheckHistory.shift();
      }
      
      if (!health.healthy) {
        this.stats.healthCheckFailures++;
        console.error('Database health check failed:', health.error);
      }
    }, interval);

    this.healthCheckInterval.unref();
  }

  _startConnectionReaping() {
    const reapingInterval = parseInt(process.env.DB_REAPING_INTERVAL) || 60000;
    
    this.reapingInterval = setInterval(async () => {
      if (!this.pool) return;
      
      try {
        const result = await this.pool.query('SELECT 1');
        const idleCount = this.pool.idleCount;
        const totalCount = this.pool.totalCount;
        
        if (idleCount > this.config.min) {
          const excessIdle = idleCount - this.config.min;
          console.log(`Connection reaping: ${excessIdle} idle connections available for cleanup`);
        }
      } catch (error) {
        console.error('Connection reaping error:', error);
      }
    }, reapingInterval);

    this.reapingInterval.unref();
  }

  _startLeakDetection() {
    const leakThreshold = parseInt(process.env.DB_LEAK_THRESHOLD) || 30000;
    
    this.leakCheckInterval = setInterval(() => {
      const now = Date.now();
      const leakedClients = [];
      
      for (const [clientId, { client, checkedOutAt, stackTrace }] of this.checkedOutClients) {
        const duration = now - checkedOutAt;
        
        if (duration > leakThreshold) {
          this.stats.connectionLeaks++;
          const leakEvent = {
            clientId,
            duration,
            stackTrace,
            timestamp: new Date().toISOString()
          };
          this.stats.connectionLeakEvents.push(leakEvent);
          leakedClients.push(leakEvent);
          
          if (this.stats.connectionLeakEvents.length > 100) {
            this.stats.connectionLeakEvents.shift();
          }
          
          console.warn(`Potential connection leak detected: client checked out for ${duration}ms`, stackTrace);
          this.emit('connectionLeak', leakEvent);
        }
      }
      
      if (leakedClients.length > 0 && this.stats.connectionLeaks > 10) {
        console.error(`Connection leak warning: ${leakedClients.length} leaks detected, total: ${this.stats.connectionLeaks}`);
      }
    }, 10000);

    this.leakCheckInterval.unref();
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
        this.stats.lastSuccessfulQuery = new Date().toISOString();
        this.queryTimes.push(duration);

        if (this.queryTimes.length > 1000) {
          this.queryTimes.shift();
        }

        this.stats.avgQueryTime =
          this.queryTimes.reduce((a, b) => a + b, 0) / this.queryTimes.length;
        
        this.stats.maxQueryTime = Math.max(this.stats.maxQueryTime, duration);
        this.stats.minQueryTime = Math.min(this.stats.minQueryTime, duration);

        this._logQuery(text, params, duration);

        if (duration > this.slowQueryThreshold) {
          this.stats.slowQueries++;
          console.warn(`Slow query (${duration}ms): ${text.substring(0, 200)}`);
          this._logSlowQuery(text, duration);
          this.emit('slowQuery', { query: text.substring(0, 500), duration, timestamp: new Date().toISOString() });
        }
      }

      return result;
    } catch (error) {
      this.stats.errors++;
      this.stats.lastError = error.message;
      this.emit('queryError', { error: error.message, query: text.substring(0, 200), timestamp: new Date().toISOString() });
      throw error;
    }
  }

  async getClient() {
    const client = await this.pool.connect();
    const clientId = Date.now() + '-' + Math.random().toString(36).substr(2, 9);
    const stackTrace = new Error().stack;
    const originalRelease = client.release.bind(client);
    const checkedOutAt = Date.now();

    this.checkedOutClients.set(clientId, { client, checkedOutAt, stackTrace });

    client.release = () => {
      this.checkedOutClients.delete(clientId);
      return originalRelease();
    };

    return client;
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
    const start = Date.now();
    try {
      const [pingResult, statsResult, connectionResult] = await Promise.all([
        this.pool.query('SELECT 1 as health, NOW() as timestamp'),
        this.pool.query('SELECT pg_database_size($1) as db_size', [this.config.database]),
        this.getPoolStats()
      ]);
      
      const duration = Date.now() - start;
      return {
        healthy: true,
        responseTime: duration,
        timestamp: new Date().toISOString(),
        dbTime: pingResult.rows[0].timestamp,
        dbSize: statsResult.rows[0].db_size,
        poolStatus: {
          total: connectionResult.total,
          idle: connectionResult.idle,
          busy: connectionResult.busy,
          waiting: connectionResult.waiting,
          checkedOut: connectionResult.checkedOut
        },
        capacityUsage: connectionResult.total > 0 
          ? ((connectionResult.busy / connectionResult.total) * 100).toFixed(2) + '%'
          : '0%'
      };
    } catch (error) {
      return {
        healthy: false,
        error: error.message,
        timestamp: new Date().toISOString(),
        responseTime: Date.now() - start
      };
    }
  }

  getPoolStats() {
    if (!this.pool) {
      return { error: 'Pool not initialized' };
    }
    
    const total = this.pool.totalCount;
    const idle = this.pool.idleCount;
    const busy = total - idle;
    const waiting = this.pool.waitingCount || 0;
    
    return {
      total,
      idle,
      busy,
      waiting,
      checkedOut: this.checkedOutClients.size,
      capacityUsage: total > 0 ? ((busy / total) * 100).toFixed(2) + '%' : '0%',
      config: {
        min: this.config.min,
        max: this.config.max,
        idleTimeout: this.config.idleTimeoutMillis,
        connectionTimeout: this.config.connectionTimeoutMillis
      },
      health: {
        lastCheck: this.lastHealthCheck,
        consecutiveFailures: this.healthCheckHistory.slice(-5).filter(h => !h.healthy).length,
        healthScore: this._calculateHealthScore()
      }
    };
  }

  _calculateHealthScore() {
    const checks = {
      poolHealth: this.stats.errors === 0 ? 1 : 0.5,
      connectionHealth: this.stats.connectionErrors === 0 ? 1 : 0.5,
      leakHealth: this.stats.connectionLeaks < 10 ? 1 : (this.stats.connectionLeaks < 50 ? 0.5 : 0),
      queryHealth: this.stats.queries > 0 
        ? (1 - (this.stats.slowQueries / this.stats.queries)) 
        : 1
    };
    
    const score = (checks.poolHealth + checks.connectionHealth + checks.leakHealth + checks.queryHealth) / 4;
    return (score * 100).toFixed(2) + '%';
  }

  getQueryStats() {
    const hitRate = this.stats.queries > 0
      ? ((1 - this.stats.slowQueries / this.stats.queries) * 100).toFixed(2) + '%'
      : '100%';
    
    const p50 = this._calculatePercentile(50);
    const p75 = this._calculatePercentile(75);
    const p95 = this._calculatePercentile(95);
    const p99 = this._calculatePercentile(99);
    
    return {
      ...this.stats,
      avgQueryTime: Math.round(this.stats.avgQueryTime * 100) / 100,
      maxQueryTime: this.stats.maxQueryTime === 0 ? 0 : Math.round(this.stats.maxQueryTime * 100) / 100,
      minQueryTime: this.stats.minQueryTime === Infinity ? 0 : Math.round(this.stats.minQueryTime * 100) / 100,
      p50QueryTime: p50,
      p75QueryTime: p75,
      p95QueryTime: p95,
      p99QueryTime: p99,
      hitRate,
      recentLeakCount: this.stats.connectionLeakEvents.length,
      totalQueryTime: Math.round(this.stats.totalQueryTime),
      errorRate: this.stats.queries > 0 
        ? ((this.stats.errors / this.stats.queries) * 100).toFixed(2) + '%'
        : '0%',
      lastSuccessfulQuery: this.stats.lastSuccessfulQuery,
      lastError: this.stats.lastError
    };
  }

  getDetailedStats() {
    return {
      pool: this.getPoolStats(),
      queries: this.getQueryStats(),
      healthCheck: {
        lastCheck: this.lastHealthCheck,
        history: this.healthCheckHistory.slice(-10),
        failures: this.stats.healthCheckFailures
      },
      leaks: {
        total: this.stats.connectionLeaks,
        recent: this.stats.connectionLeakEvents.slice(-10)
      }
    };
  }

  _calculatePercentile(percentile) {
    if (this.queryTimes.length === 0) return 0;
    
    const sorted = [...this.queryTimes].sort((a, b) => a - b);
    const index = Math.ceil(percentile / 100 * sorted.length) - 1;
    return sorted[index] || 0;
  }

  resetStats() {
    this.stats = this._initializeStats();
    this.queryTimes = [];
  }

  async close() {
    if (this.healthCheckInterval) {
      clearInterval(this.healthCheckInterval);
      this.healthCheckInterval = null;
    }
    
    if (this.leakCheckInterval) {
      clearInterval(this.leakCheckInterval);
      this.leakCheckInterval = null;
    }
    
    if (this.reapingInterval) {
      clearInterval(this.reapingInterval);
      this.reapingInterval = null;
    }
    
    if (this.pool) {
      const idleConnections = this.pool.idleCount;
      if (idleConnections > 0) {
        console.log(`Closing pool with ${idleConnections} idle connections`);
      }
      
      await this.pool.end();
      this.pool = null;
      this.initialized = false;
      console.log('Database pool closed successfully');
    }
  }
}

const dbPoolManager = new DatabasePoolManager();

if (process.env.NODE_ENV !== 'test') {
  dbPoolManager.initialize();
}

module.exports = dbPoolManager;
