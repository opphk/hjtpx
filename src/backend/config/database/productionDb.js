const { Pool } = require('pg');

const productionPoolConfig = {
  min: parseInt(process.env.DB_POOL_MIN) || 5,
  max: parseInt(process.env.DB_POOL_MAX) || 20,
  idleTimeoutMillis: parseInt(process.env.DB_IDLE_TIMEOUT) || 30000,
  connectionTimeoutMillis: parseInt(process.env.DB_CONNECTION_TIMEOUT) || 5000,
  statement_timeout: parseInt(process.env.DB_STATEMENT_TIMEOUT) || 30000,
  allowExitOnIdle: false
};

if (process.env.NODE_ENV === 'production') {
  productionPoolConfig.ssl = {
    rejectUnauthorized: process.env.DB_SSL_REJECT_UNAUTHORIZED !== 'false',
    ...(process.env.DB_SSL_CERT && { cert: process.env.DB_SSL_CERT }),
    ...(process.env.DB_SSL_KEY && { key: process.env.DB_SSL_KEY }),
    ...(process.env.DB_SSL_CA && { ca: process.env.DB_SSL_CA })
  };
}

const createOptimizedPool = (config = {}) => {
  const poolConfig = {
    ...productionPoolConfig,
    ...config,
    host: config.host || process.env.DB_HOST,
    port: config.port || parseInt(process.env.DB_PORT) || 5432,
    database: config.database || process.env.DB_NAME,
    user: config.user || process.env.DB_USER,
    password: config.password || process.env.DB_PASSWORD
  };

  const pool = new Pool(poolConfig);

  pool.on('error', (err, client) => {
    console.error('Unexpected error on idle client', err);
  });

  pool.on('connect', () => {
    console.log('New client connected to database');
  });

  return pool;
};

const productionPool = process.env.NODE_ENV === 'production' ? createOptimizedPool() : null;

const healthCheck = async pool => {
  const start = Date.now();
  try {
    const result = await pool.query('SELECT 1 as health');
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
};

module.exports = {
  createOptimizedPool,
  productionPool,
  healthCheck,
  productionPoolConfig
};
