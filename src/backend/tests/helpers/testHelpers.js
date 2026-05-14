const pool = require('../../../config/database/db');
const bcrypt = require('bcrypt');
const jwt = require('jsonwebtoken');

async function connectDatabase() {
  try {
    const client = await pool.connect();
    return client;
  } catch (error) {
    console.error('Database connection failed:', error);
    throw error;
  }
}

async function disconnectDatabase() {
  try {
    await pool.end();
  } catch (error) {
    console.error('Database disconnection failed:', error);
    throw error;
  }
}

async function createTestUser(overrides = {}) {
  const defaultUser = {
    email: `test_${Date.now()}@example.com`,
    name: `Test User ${Date.now()}`,
    password: 'TestPassword123!'
  };
  const user = { ...defaultUser, ...overrides };
  const hashedPassword = await bcrypt.hash(user.password, 10);
  
  const result = await pool.query(
    'INSERT INTO users (email, name, password) VALUES ($1, $2, $3) RETURNING id, email, name, created_at',
    [user.email, user.name, hashedPassword]
  );
  
  return {
    ...result.rows[0],
    plainPassword: user.password
  };
}

async function cleanTestData() {
  try {
    await pool.query('DELETE FROM users WHERE email LIKE $1', ['%test_%@example.com']);
  } catch (error) {
    console.error('Clean test data failed:', error);
    throw error;
  }
}

async function generateTestToken(userId, email) {
  return jwt.sign(
    { userId, email },
    process.env.JWT_SECRET || 'test-secret-key',
    { expiresIn: '1h' }
  );
}

async function generateExpiredToken(userId, email) {
  return jwt.sign(
    { userId, email },
    process.env.JWT_SECRET || 'test-secret-key',
    { expiresIn: '-1h' }
  );
}

module.exports = {
  connectDatabase,
  disconnectDatabase,
  createTestUser,
  cleanTestData,
  generateTestToken,
  generateExpiredToken
};
