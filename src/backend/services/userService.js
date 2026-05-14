const pool = require('../../config/database/db');
const bcrypt = require('bcrypt');
const authService = require('./authService');

const VALID_ROLES = ['admin', 'user', 'moderator'];

async function getAllUsers() {
  const result = await pool.query('SELECT id, email, name, role, created_at FROM users ORDER BY created_at DESC');
  return result.rows;
}

async function getUserById(id) {
  const result = await pool.query('SELECT id, email, name, role, created_at FROM users WHERE id = $1', [id]);
  return result.rows[0];
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
  return result.rows[0];
}

async function deleteUser(id) {
  await pool.query('DELETE FROM users WHERE id = $1', [id]);
}

module.exports = {
  getAllUsers,
  getUserById,
  createUser,
  updateUser,
  deleteUser,
  VALID_ROLES
};
