const { createClient } = require('redis');

const isProduction = process.env.NODE_ENV === 'production';
const redisHost = process.env.REDIS_HOST || 'localhost';
const redisPort = process.env.REDIS_PORT || 6379;
const redisPassword = process.env.REDIS_PASSWORD || undefined;
const redisDb = parseInt(process.env.REDIS_DB || '0', 10);

const redisClient = createClient({
  socket: {
    host: redisHost,
    port: parseInt(redisPort, 10),
    reconnectStrategy: (retries) => {
      if (retries > 20) {
        return new Error('Max retries reached');
      }
      return Math.min(retries * 100, 3000);
    },
    connectTimeout: 10000,
    keepAlive: 30000
  },
  password: redisPassword,
  database: redisDb,
  legacyMode: false,
  commandTimeout: 5000
});

redisClient.on('error', (err) => console.error('Redis Client Error:', err));
redisClient.on('connect', () => console.log('Redis Client Connected'));
redisClient.on('ready', () => console.log('Redis Client Ready'));
redisClient.on('reconnecting', () => console.log('Redis Client Reconnecting'));
redisClient.on('end', () => console.log('Redis Client Disconnected'));

module.exports = redisClient;
