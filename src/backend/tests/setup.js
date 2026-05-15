process.env.NODE_ENV = 'test';
process.env.JWT_SECRET = 'test-secret-key-for-testing';

const { Pool } = require('pg');

let testPool;

beforeAll(async () => {
  console.log('Test environment initialized');
  
  if (process.env.RUN_INTEGRATION_TESTS === 'true') {
    testPool = new Pool({
      host: process.env.TEST_DB_HOST || 'localhost',
      port: process.env.TEST_DB_PORT || 5432,
      database: process.env.TEST_DB_NAME || 'hjtpx_test',
      user: process.env.TEST_DB_USER || 'postgres',
      password: process.env.TEST_DB_PASSWORD || 'postgres',
      max: 5,
      idleTimeoutMillis: 30000,
      connectionTimeoutMillis: 2000,
    });
    
    try {
      await testPool.query('SELECT 1');
      console.log('Test database connected successfully');
    } catch (error) {
      console.log('Test database not available, using mocks');
      testPool = null;
    }
  }
});

afterAll(async () => {
  console.log('Test environment cleanup completed');
  
  if (testPool) {
    try {
      await testPool.end();
    } catch (error) {
      console.log('Error closing test pool:', error.message);
    }
  }
});
