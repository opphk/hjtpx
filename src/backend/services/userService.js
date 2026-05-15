const bcrypt = require('bcryptjs');

const pool = require('../../../config/database/db');
const queryOptimizer = require('../utils/queryOptimizer');

const authService = require('./authService');
const cacheService = require('./cacheService');

const VALID_ROLES = ['admin', 'user', 'moderator'];
const CACHE_KEYS = {
  ALL_USERS: 'users:all',
  USER_BY_ID: id => `users:id:${id}`
};
const CACHE_TTL = 300;

async function getAllUsers({ limit = 100, offset = 0 } = {}) {
  const result = await queryOptimizer.cachedQuery(
    'SELECT id, email, name, role, created_at FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2',
    [limit, offset],
    CACHE_TTL
  );

  return result;
}

async function getUserById(id) {
  const cacheKey = CACHE_KEYS.USER_BY_ID(id);
  const cached = await cacheService.get(cacheKey);
  if (cached) {
    return cached;
  }

  const result = await queryOptimizer.cachedQuery(
    'SELECT id, email, name, role, created_at FROM users WHERE id = $1',
    [id],
    CACHE_TTL
  );

  if (result[0]) {
    await cacheService.set(cacheKey, result[0], CACHE_TTL);
  }
  return result[0];
}

async function createUser({ email, name, password, role = 'user' }) {
  if (!password || password.length < 8) {
    throw new Error('Password must be at least 8 characters long');
  }

  if (role && !VALID_ROLES.includes(role)) {
    throw new Error(`Invalid role. Must be one of: ${VALID_ROLES.join(', ')}`);
  }

  authService.validatePassword(password);

  const hashedPassword = await bcrypt.hash(password, 10);
  const result = await pool.query(
    'INSERT INTO users (email, name, password, role) VALUES ($1, $2, $3, $4) RETURNING id, email, name, role, created_at',
    [email, name, hashedPassword, role]
  );

  await cacheService.del(CACHE_KEYS.ALL_USERS);
  queryOptimizer.clearCache();

  return result.rows[0];
}

async function updateUser(id, { email, name, password, role }) {
  const updates = [];
  const values = [];
  let paramCount = 1;

  if (email) {
    updates.push(`email = $${paramCount++}`);
    values.push(email);
  }
  if (name) {
    updates.push(`name = $${paramCount++}`);
    values.push(name);
  }
  if (password) {
    authService.validatePassword(password);
    updates.push(`password = $${paramCount++}`);
    values.push(await bcrypt.hash(password, 10));
  }
  if (role) {
    if (!VALID_ROLES.includes(role)) {
      throw new Error(`Invalid role. Must be one of: ${VALID_ROLES.join(', ')}`);
    }
    updates.push(`role = $${paramCount++}`);
    values.push(role);
  }

  if (updates.length === 0) {
    return getUserById(id);
  }

  values.push(id);
  const result = await pool.query(
    `UPDATE users SET ${updates.join(', ')}, updated_at = CURRENT_TIMESTAMP WHERE id = $${paramCount} RETURNING id, email, name, role, created_at`,
    values
  );

  await Promise.all([
    cacheService.del(CACHE_KEYS.ALL_USERS),
    cacheService.del(CACHE_KEYS.USER_BY_ID(id))
  ]);
  queryOptimizer.clearCache();

  return result.rows[0];
}

async function deleteUser(id) {
  await pool.query('DELETE FROM users WHERE id = $1', [id]);

  await Promise.all([
    cacheService.del(CACHE_KEYS.ALL_USERS),
    cacheService.del(CACHE_KEYS.USER_BY_ID(id))
  ]);
  queryOptimizer.clearCache();
}

async function getUserByEmail(email) {
  const result = await pool.query(
    'SELECT id, email, name, role, password, created_at FROM users WHERE email = $1',
    [email]
  );
  return result.rows[0];
}

module.exports = {
  getAllUsers,
  getUserById,
  getUserByEmail,
  createUser,
  updateUser,
  deleteUser,
  VALID_ROLES
};
