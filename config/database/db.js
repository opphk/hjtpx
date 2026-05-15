const { Pool } = require('pg');
const fs = require('fs');
const path = require('path');
const EventEmitter = require('events');

const isProduction = process.env.NODE_ENV === 'production';

const config = {
  host: process.env.DB_HOST || 'localhost',
  port: process.env.DB_PORT || 5432,
  database: process.env.DB_NAME || 'hjtpx',
  user: process.env.DB_USER || 'postgres',
  password: process.env.DB_PASSWORD || 'postgres',
  max: parseInt(process.env.DB_POOL_MAX) || (isProduction ? 50 : 10),
  min: parseInt(process.env.DB_POOL_MIN) || (isProduction ? 10 : 2),
  idleTimeoutMillis: parseInt(process.env.DB_IDLE_TIMEOUT) || (isProduction ? 60000 : 30000),
  connectionTimeoutMillis: parseInt(process.env.DB_CONNECTION_TIMEOUT) || (isProduction ? 10000 : 5000),
  statement_timeout: parseInt(process.env.DB_STATEMENT_TIMEOUT) || (isProduction ? 60000 : 30000),
  allowExitOnIdle: false,
  ...(isProduction ? {
    keepAlive: true,
    keepAliveInitialDelayMillis: 30000
  } : {})
};

const pool = new Pool(config);
const poolEvents = new EventEmitter();

const queryLogFile = path.join(__dirname, '../../logs/query.log');
const checkedOutClients = new Map();

if (!fs.existsSync(path.dirname(queryLogFile))) {
  fs.mkdirSync(path.dirname(queryLogFile), { recursive: true });
}

let stats = {
  queries: 0,
  slowQueries: 0,
  errors: 0,
  totalQueryTime: 0,
  connectionLeaks: 0
};
const queryTimes = [];
const slowQueryThreshold = parseInt(process.env.SLOW_QUERY_THRESHOLD) || 100;

function logQuery(query, params, duration) {
  const logEntry = {
    timestamp: new Date().toISOString(),
    query,
    duration: `${duration}ms`,
  };
  try {
    fs.appendFileSync(queryLogFile, JSON.stringify(logEntry) + '\n');
  } catch (e) {
    console.error('Failed to write query log:', e);
  }
}

pool.on('error', (err, client) => {
  console.error('Unexpected error on idle client', err);
  stats.errors++;
  poolEvents.emit('error', { error: err, timestamp: new Date().toISOString() });
});

pool.on('connect', () => {
  console.log('New client connected to PostgreSQL');
  poolEvents.emit('connect', { timestamp: new Date().toISOString() });
});

pool.on('acquire', () => {
  poolEvents.emit('acquire', { timestamp: new Date().toISOString() });
});

pool.on('remove', () => {
  console.log('Client removed from pool');
  poolEvents.emit('remove', { timestamp: new Date().toISOString() });
});

async function query(text, params) {
  const start = Date.now();
  try {
    const res = await pool.query(text, params);
    const duration = Date.now() - start;
    
    stats.queries++;
    stats.totalQueryTime += duration;
    queryTimes.push(duration);
    
    if (queryTimes.length > 1000) {
      queryTimes.shift();
    }
    
    if (duration > slowQueryThreshold) {
      stats.slowQueries++;
      console.warn(`Slow query (${duration}ms): ${text.substring(0, 200)}`);
      poolEvents.emit('slowQuery', { query: text.substring(0, 500), duration, timestamp: new Date().toISOString() });
    }

    logQuery(text, params, duration);

    if (process.env.NODE_ENV === 'development') {
      console.log('Executed query', { text, duration: `${duration}ms`, rows: res.rowCount });
    }

    return res;
  } catch (error) {
    stats.errors++;
    console.error('Database query error:', error.message);
    poolEvents.emit('queryError', { error: error.message, query: text.substring(0, 200), timestamp: new Date().toISOString() });
    throw error;
  }
}

async function getClient() {
  const client = await pool.connect();
  const originalRelease = client.release.bind(client);
  const clientId = Date.now() + '-' + Math.random().toString(36).substr(2, 9);
  const stackTrace = new Error().stack;
  const checkedOutAt = Date.now();

  checkedOutClients.set(clientId, { client, checkedOutAt, stackTrace });

  client.release = () => {
    checkedOutClients.delete(clientId);
    return originalRelease();
  };

  return client;
}

async function transaction(callback) {
  const client = await getClient();
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

async function healthCheck() {
  const start = Date.now();
  try {
    const [nowRes, sizeRes] = await Promise.all([
      pool.query('SELECT NOW() as now'),
      pool.query('SELECT pg_database_size($1) as db_size', [config.database])
    ]);
    const duration = Date.now() - start;
    return {
      status: 'healthy',
      timestamp: nowRes.rows[0].now,
      responseTime: duration,
      dbSize: sizeRes.rows[0].db_size
    };
  } catch (error) {
    return { status: 'unhealthy', error: error.message, timestamp: new Date().toISOString() };
  }
}

async function getPoolStats() {
  const avgQueryTime = queryTimes.length > 0
    ? queryTimes.reduce((a, b) => a + b, 0) / queryTimes.length
    : 0;
    
  const hitRate = stats.queries > 0
    ? ((1 - stats.slowQueries / stats.queries) * 100).toFixed(2) + '%'
    : '100%';

  return {
    totalCount: pool.totalCount,
    idleCount: pool.idleCount,
    waitingCount: pool.waitingCount,
    checkedOutCount: checkedOutClients.size,
    queryStats: {
      totalQueries: stats.queries,
      slowQueries: stats.slowQueries,
      errors: stats.errors,
      avgQueryTime: Math.round(avgQueryTime * 100) / 100,
      hitRate,
      connectionLeaks: stats.connectionLeaks
    }
  };
}

async function close() {
  await pool.end();
}

const leakThreshold = parseInt(process.env.DB_LEAK_THRESHOLD) || 30000;
const leakCheckInterval = setInterval(() => {
  const now = Date.now();
  for (const [clientId, { checkedOutAt, stackTrace }] of checkedOutClients) {
    const duration = now - checkedOutAt;
    if (duration > leakThreshold) {
      stats.connectionLeaks++;
      console.warn(`Potential connection leak detected: client checked out for ${duration}ms`, stackTrace);
      poolEvents.emit('connectionLeak', { clientId, duration, stackTrace, timestamp: new Date().toISOString() });
    }
  }
}, 10000);

leakCheckInterval.unref();

module.exports = {
  query,
  getClient,
  transaction,
  healthCheck,
  getPoolStats,
  close,
  pool,
  events: poolEvents
};
