const crypto = require('crypto');

const bcrypt = require('bcryptjs');
const jwt = require('jsonwebtoken');

const pool = require('../../../config/database/db');

const JWT_SECRET = process.env.JWT_SECRET || 'hjtpx-secret-key-change-in-production';
const JWT_EXPIRES_IN = process.env.JWT_EXPIRES_IN || '7d';
const SALT_ROUNDS = 10;
const PASSWORD_RESET_TOKEN_EXPIRY = 3600000;

async function register({ email, name, password, role = 'user' }) {
  const existingUser = await pool.query('SELECT id FROM users WHERE email = $1', [email]);
  if (existingUser.rows.length > 0) {
    throw new Error('Email already registered');
  }

  validatePassword(password);

  const hashedPassword = await bcrypt.hash(password, SALT_ROUNDS);
  const result = await pool.query(
    'INSERT INTO users (email, name, password, role) VALUES ($1, $2, $3, $4) RETURNING id, email, name, role, created_at',
    [email, name, hashedPassword, role]
  );

  const user = result.rows[0];
  const token = generateToken(user);

  return { user, token };
}

async function login({ email, password }) {
  const result = await pool.query(
    'SELECT id, email, name, password, role FROM users WHERE email = $1',
    [email]
  );

  if (result.rows.length === 0) {
    throw new Error('Invalid credentials');
  }

  const user = result.rows[0];
  const isValidPassword = await bcrypt.compare(password, user.password);

  if (!isValidPassword) {
    throw new Error('Invalid credentials');
  }

  const token = generateToken(user);

  delete user.password;

  return { user, token };
}

async function forgotPassword(email) {
  const result = await pool.query('SELECT id FROM users WHERE email = $1', [email]);

  if (result.rows.length === 0) {
    return { message: 'If email exists, reset link will be sent' };
  }

  const resetToken = crypto.randomBytes(32).toString('hex');
  const hashedToken = await bcrypt.hash(resetToken, SALT_ROUNDS);
  const expiresAt = new Date(Date.now() + PASSWORD_RESET_TOKEN_EXPIRY);

  await pool.query('UPDATE users SET reset_token = $1, reset_token_expires = $2 WHERE email = $3', [
    hashedToken,
    expiresAt,
    email
  ]);

  console.log(`Password reset token for ${email}: ${resetToken}`);

  return {
    message: 'If email exists, reset link will be sent',
    resetToken: process.env.NODE_ENV === 'development' ? resetToken : undefined
  };
}

async function resetPassword({ token, newPassword }) {
  const result = await pool.query(
    'SELECT id, reset_token, reset_token_expires FROM users WHERE reset_token IS NOT NULL'
  );

  let user = null;
  for (const row of result.rows) {
    const isValid = await bcrypt.compare(token, row.reset_token);
    if (isValid && new Date(row.reset_token_expires) > new Date()) {
      user = row;
      break;
    }
  }

  if (!user) {
    throw new Error('Invalid or expired reset token');
  }

  validatePassword(newPassword);

  const hashedPassword = await bcrypt.hash(newPassword, SALT_ROUNDS);

  await pool.query(
    'UPDATE users SET password = $1, reset_token = NULL, reset_token_expires = NULL WHERE id = $2',
    [hashedPassword, user.id]
  );

  return { message: 'Password successfully reset' };
}

async function getCurrentUser(userId) {
  const result = await pool.query(
    'SELECT id, email, name, role, created_at FROM users WHERE id = $1',
    [userId]
  );

  if (result.rows.length === 0) {
    throw new Error('User not found');
  }

  return result.rows[0];
}

async function logout(userId) {
  await pool.query('DELETE FROM sessions WHERE user_id = $1', [userId]);
  return { message: 'Logged out successfully' };
}

async function validateSession(token) {
  try {
    const decoded = jwt.verify(token, JWT_SECRET);
    return decoded;
  } catch (error) {
    throw new Error('Invalid session');
  }
}

function generateToken(user) {
  return jwt.sign({ id: user.id, email: user.email, role: user.role }, JWT_SECRET, {
    expiresIn: JWT_EXPIRES_IN
  });
}

function validatePassword(password) {
  if (!password || password.length < 8) {
    throw new Error('Password must be at least 8 characters long');
  }
  if (!/(?=.*[a-z])(?=.*[A-Z])(?=.*\d)/.test(password)) {
    throw new Error(
      'Password must contain at least one uppercase letter, one lowercase letter, and one number'
    );
  }
  return true;
}

module.exports = {
  register,
  login,
  forgotPassword,
  resetPassword,
  getCurrentUser,
  logout,
  validateSession,
  generateToken,
  validatePassword
};
