const { Pool } = require('pg');
const fs = require('fs');
const path = require('path');
const { v4: uuidv4 } = require('uuid');

const isProduction = process.env.NODE_ENV === 'production';
const isStaging = process.env.NODE_ENV === 'staging';

const config = {
  host: process.env.DB_HOST || 'localhost',
  port: parseInt(process.env.DB_PORT) || 5432,
  database: process.env.DB_NAME || 'hjtpx',
  user: process.env.DB_USER || 'postgres',
  password: process.env.DB_PASSWORD || 'postgres',
  max: parseInt(process.env.DB_POOL_MAX) || (isProduction ? 30 : 10),
  min: parseInt(process.env.DB_POOL_MIN) || 2,
  idleTimeoutMillis: parseInt(process.env.DB_IDLE_TIMEOUT) || 30000,
  connectionTimeoutMillis: parseInt(process.env.DB_CONNECTION_TIMEOUT) || 5000,
  acquireTimeoutMillis: parseInt(process.env.DB_ACQUIRE_TIMEOUT) || 10000,
  reapIntervalMillis: 1000,
  allowExitOnIdle: false
};

const pool = new Pool(config);

const queryLogFile = path.join(__dirname, '../../../logs/query.log');
const healthLogFile = path.join(__dirname, '../../../logs/health.log');
const leakLogFile = path.join(__dirname, '../../../logs/connection-leaks.log');
const monitorLogFile = path.join(__dirname, '../../../logs/pool-monitor.log');

if (!fs.existsSync(path.dirname(queryLogFile))) {
  fs.mkdirSync(path.dirname(queryLogFile), { recursive: true });
}

class ConnectionLeakDetector {
  constructor() {
    this.checkedOutConnections = new Map();
    this.warningThreshold = parseInt(process.env.DB_LEAK_WARNING_THRESHOLD) || 5000;
    this.criticalThreshold = parseInt(process.env.DB_LEAK_CRITICAL_THRESHOLD) || 10000;
  }

  track(client) {
    const id = uuidv4();
    const entry = {
      id,
      client,
      checkedOutAt: Date.now(),
      lastQueryAt: Date.now(),
      queryCount: 0
    };
    this.checkedOutConnections.set(id, entry);
    this.scheduleWarning(id);
    return id;
  }

  update(id) {
    const entry = this.checkedOutConnections.get(id);
    if (entry) {
      entry.lastQueryAt = Date.now();
      entry.queryCount++;
    }
  }

  release(id) {
    const entry = this.checkedOutConnections.get(id);
    if (entry) {
      clearTimeout(entry.warningTimer);
      clearTimeout(entry.criticalTimer);
      this.checkedOutConnections.delete(id);
      return entry;
    }
    return null;
  }

  scheduleWarning(id) {
    const entry = this.checkedOutConnections.get(id);
    if (!entry) return;

    entry.warningTimer = setTimeout(() => {
      const duration = Date.now() - entry.checkedOutAt;
      const warning = {
        type: 'warning',
        connectionId: id,
        duration,
        threshold: this.warningThreshold,
        queryCount: entry.queryCount,
        timestamp: new Date().toISOString()
      };
      fs.appendFileSync(leakLogFile, JSON.stringify(warning) + '\n');
      console.warn(`[ConnectionLeak] Warning: Connection ${id} checked out for ${duration}ms`);
    }, this.warningThreshold);

    entry.criticalTimer = setTimeout(() => {
      const duration = Date.now() - entry.checkedOutAt;
      const critical = {
        type: 'critical',
        connectionId: id,
        duration,
        threshold: this.criticalThreshold,
        queryCount: entry.queryCount,
        timestamp: new Date().toISOString()
      };
      fs.appendFileSync(leakLogFile, JSON.stringify(critical) + '\n');
      console.error(`[ConnectionLeak] CRITICAL: Connection ${id} checked out for ${duration}ms`);
    }, this.criticalThreshold);
  }

  getStats() {
    const connections = Array.from(this.checkedOutConnections.values()).map(entry => ({
      id: entry.id,
      checkedOutAt: new Date(entry.checkedOutAt).toISOString(),
      duration: Date.now() - entry.checkedOutAt,
      queryCount: entry.queryCount
    }));

    return {
      activeConnections: this.checkedOutConnections.size,
      connections: connections.sort((a, b) => b.duration - a.duration)
    };
  }
}

const leakDetector = new ConnectionLeakDetector();

function logQuery(query, params, duration, requestId = null) {
  const logEntry = {
    timestamp: new Date().toISOString(),
    requestId,
    query,
    params: params ? params.map(p => typeof p === 'string' ? p.substring(0, 100) : p) : null,
    duration: `${duration}ms`,
    slow: duration > 1000
  };
  fs.appendFileSync(queryLogFile, JSON.stringify(logEntry) + '\n');

  if (duration > 1000) {
    console.warn(`[SlowQuery] ${duration}ms: ${query.substring(0, 100)}`);
  }
}

pool.on('error', (err, client) => {
  const errorLog = {
    timestamp: new Date().toISOString(),
    type: 'pool_error',
    message: err.message,
    stack: err.stack
  };
  fs.appendFileSync(monitorLogFile, JSON.stringify(errorLog) + '\n');
  console.error('Unexpected error on idle client', err);
  process.exit(-1);
});

pool.on('connect', (client) => {
  const connectLog = {
    timestamp: new Date().toISOString(),
    type: 'connect',
    poolTotal: pool.totalCount,
    poolIdle: pool.idleCount
  };
  console.log('New client connected to PostgreSQL');
});

pool.on('remove', () => {
  console.log('Client removed from pool');
});

pool.on('acquire', () => {
  console.log('Client acquired from pool');
});

pool.on('wait', () => {
  console.warn('Client waiting for available connection');
});

async function query(text, params, requestId = null) {
  const start = Date.now();
  const connectionId = uuidv4();

  try {
    const res = await pool.query(text, params);
    const duration = Date.now() - start;
    logQuery(text, params, duration, requestId);

    if (process.env.NODE_ENV === 'development') {
      console.log('Executed query', { text, duration: `${duration}ms`, rows: res.rowCount });
    }

    return res;
  } catch (error) {
    const duration = Date.now() - start;
    const errorLog = {
      timestamp: new Date().toISOString(),
      requestId,
      type: 'query_error',
      query: text,
      error: error.message,
      code: error.code,
      duration: `${duration}ms`
    };
    fs.appendFileSync(monitorLogFile, JSON.stringify(errorLog) + '\n');

    console.error('Database query error:', error.message);
    throw error;
  }
}

async function getClient() {
  const client = await pool.connect();
  const connectionId = leakDetector.track(client);
  const originalQuery = client.query.bind(client);
  const originalRelease = client.release.bind(client);

  const timeout = setTimeout(() => {
    const warning = {
      timestamp: new Date().toISOString(),
      type: 'checkout_timeout',
      connectionId,
      duration: 5000,
      message: 'A client has been checked out for more than 5 seconds!'
    };
    fs.appendFileSync(monitorLogFile, JSON.stringify(warning) + '\n');
    console.error('A client has been checked out for more than 5 seconds!');
  }, 5000);

  client.query = (...args) => {
    leakDetector.update(connectionId);
    return originalQuery(...args);
  };

  client.release = () => {
    clearTimeout(timeout);
    leakDetector.release(connectionId);
    return originalRelease();
  };

  client.connectionId = connectionId;
  return client;
}

async function transaction(callback, requestId = null) {
  const client = await getClient();
  const startTime = Date.now();

  try {
    await client.query('BEGIN');
    const result = await callback(client);
    await client.query('COMMIT');

    const duration = Date.now() - startTime;
    if (duration > 2000) {
      const slowTxLog = {
        timestamp: new Date().toISOString(),
        type: 'slow_transaction',
        requestId,
        duration: `${duration}ms`,
        connectionId: client.connectionId
      };
      fs.appendFileSync(monitorLogFile, JSON.stringify(slowTxLog) + '\n');
    }

    return result;
  } catch (error) {
    await client.query('ROLLBACK');
    throw error;
  } finally {
    client.release();
  }
}

async function healthCheck() {
  const start = Date.now();
  const checks = {
    timestamp: new Date().toISOString(),
    overall: 'healthy',
    checks: {}
  };

  try {
    const connRes = await query('SELECT NOW() as now, version() as version', null);
    checks.checks.database = {
      status: 'healthy',
      latency: `${Date.now() - start}ms`,
      version: connRes.rows[0].version.split(' ')[0] + ' ' + connRes.rows[0].version.split(' ')[1]
    };
  } catch (error) {
    checks.checks.database = {
      status: 'unhealthy',
      error: error.message
    };
    checks.overall = 'unhealthy';
  }

  try {
    const poolStats = await getPoolStats();
    checks.checks.pool = {
      status: poolStats.totalCount > 0 ? 'healthy' : 'unknown',
      ...poolStats
    };
  } catch (error) {
    checks.checks.pool = {
      status: 'unknown',
      error: error.message
    };
  }

  if (checks.overall === 'unhealthy') {
    checks.status = 'unhealthy';
  } else {
    checks.status = 'healthy';
  }

  const healthLog = {
    timestamp: checks.timestamp,
    type: 'health_check',
    ...checks
  };
  fs.appendFileSync(healthLogFile, JSON.stringify(healthLog) + '\n');

  return checks;
}

async function getPoolStats() {
  return {
    totalCount: pool.totalCount,
    idleCount: pool.idleCount,
    waitingCount: pool.waitingCount,
    maxConnections: config.max,
    minConnections: config.min,
    usagePercent: Math.round((pool.totalCount / config.max) * 100),
    leakDetector: leakDetector.getStats()
  };
}

async function getDetailedStats() {
  const stats = await getPoolStats();

  return {
    ...stats,
    configuration: {
      host: config.host,
      port: config.port,
      database: config.database,
      max: config.max,
      min: config.min,
      idleTimeoutMillis: config.idleTimeoutMillis,
      connectionTimeoutMillis: config.connectionTimeoutMillis,
      acquireTimeoutMillis: config.acquireTimeoutMillis
    },
    health: await healthCheck()
  };
}

async function validateConnection() {
  const client = await getClient();
  try {
    await client.query('SELECT 1');
    return { valid: true };
  } catch (error) {
    return { valid: false, error: error.message };
  } finally {
    client.release();
  }
}

async function close() {
  console.log('Closing database pool...');
  await pool.end();
  console.log('Database pool closed');
}

process.on('SIGTERM', async () => {
  console.log('SIGTERM received, closing pool gracefully...');
  await close();
  process.exit(0);
});

module.exports = {
  query,
  getClient,
  transaction,
  healthCheck,
  getPoolStats,
  getDetailedStats,
  validateConnection,
  close,
  pool
};
