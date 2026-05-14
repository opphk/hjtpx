const { Pool } = require('pg');
const fs = require('fs');
const path = require('path');

const config = {
  host: process.env.DB_HOST || 'localhost',
  port: process.env.DB_PORT || 5432,
  database: process.env.DB_NAME || 'hjtpx',
  user: process.env.DB_USER || 'postgres',
  password: process.env.DB_PASSWORD || 'postgres',
  max: 20,
  idleTimeoutMillis: 30000,
  connectionTimeoutMillis: 2000,
};

const pool = new Pool(config);

const queryLogFile = path.join(__dirname, '../../logs/query.log');

if (!fs.existsSync(path.dirname(queryLogFile))) {
  fs.mkdirSync(path.dirname(queryLogFile), { recursive: true });
}

function logQuery(query, params, duration) {
  const logEntry = {
    timestamp: new Date().toISOString(),
    query,
    params,
    duration: `${duration}ms`,
  };
  fs.appendFileSync(queryLogFile, JSON.stringify(logEntry) + '\n');
}

pool.on('error', (err, client) => {
  console.error('Unexpected error on idle client', err);
  process.exit(-1);
});

pool.on('connect', () => {
  console.log('New client connected to PostgreSQL');
});

pool.on('remove', () => {
  console.log('Client removed from pool');
});

async function query(text, params) {
  const start = Date.now();
  try {
    const res = await pool.query(text, params);
    const duration = Date.now() - start;
    logQuery(text, params, duration);

    if (process.env.NODE_ENV === 'development') {
      console.log('Executed query', { text, duration: `${duration}ms`, rows: res.rowCount });
    }

    return res;
  } catch (error) {
    console.error('Database query error:', error.message);
    throw error;
  }
}

async function getClient() {
  const client = await pool.connect();
  const originalQuery = client.query.bind(client);
  const originalRelease = client.release.bind(client);

  const timeout = setTimeout(() => {
    console.error('A client has been checked out for more than 5 seconds!');
  }, 5000);

  client.query = (...args) => {
    return originalQuery(...args);
  };

  client.release = () => {
    clearTimeout(timeout);
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
  try {
    const res = await query('SELECT NOW() as now');
    return { status: 'healthy', timestamp: res.rows[0].now };
  } catch (error) {
    return { status: 'unhealthy', error: error.message };
  }
}

async function getPoolStats() {
  return {
    totalCount: pool.totalCount,
    idleCount: pool.idleCount,
    waitingCount: pool.waitingCount,
  };
}

async function close() {
  await pool.end();
}

module.exports = {
  query,
  getClient,
  transaction,
  healthCheck,
  getPoolStats,
  close,
  pool,
};
