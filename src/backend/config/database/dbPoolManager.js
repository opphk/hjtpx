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
      allowExitOnIdle: false
    };
    this.stats = {
      queries: 0,
      slowQueries: 0,
      errors: 0,
      avgQueryTime: 0,
      totalQueryTime: 0
    };
    this.queryTimes = [];
    this.slowQueryThreshold = parseInt(process.env.SLOW_QUERY_THRESHOLD) || 100;
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
        rejectUnauthorized: process.env.DB_SSL_REJECT_UNAUTHORIZED !== 'false'
      };
    }

    this.pool.on('error', (err) => {
      console.error('Unexpected database pool error:', err);
      this.stats.errors++;
    });

    this.pool.on('connect', () => {
      console.log('New client connected to database pool');
    });

    console.log('Database pool initialized');
    return this.pool;
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
        }
      }

      return result;
    } catch (error) {
      this.stats.errors++;
      throw error;
    }
  }

  async getClient() {
    return await this.pool.connect();
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
      const result = await this.pool.query('SELECT 1 as health');
      const duration = Date.now() - start;
      return {
        healthy: true,
        responseTime: duration,
        timestamp: new Date().toISOString()
      };
    } catch (error) {
      return {
        healthy: false,
        error: error.message,
        timestamp: new Date().toISOString()
      };
    }
  }

  getPoolStats() {
    return {
      total: this.pool.totalCount,
      idle: this.pool.idleCount,
      busy: this.pool.totalCount - this.pool.idleCount,
      waiting: this.pool.waitingCount
    };
  }

  getQueryStats() {
    return {
      ...this.stats,
      avgQueryTime: Math.round(this.stats.avgQueryTime * 100) / 100,
      hitRate: this.stats.queries > 0
        ? ((1 - this.stats.slowQueries / this.stats.queries) * 100).toFixed(2) + '%'
        : '100%'
    };
  }

  resetStats() {
    this.stats = {
      queries: 0,
      slowQueries: 0,
      errors: 0,
      avgQueryTime: 0,
      totalQueryTime: 0
    };
    this.queryTimes = [];
  }

  async close() {
    if (this.pool) {
      await this.pool.end();
      this.pool = null;
      console.log('Database pool closed');
    }
  }
}

const dbPoolManager = new DatabasePoolManager();

if (process.env.NODE_ENV !== 'test') {
  dbPoolManager.initialize();
}

module.exports = dbPoolManager;
