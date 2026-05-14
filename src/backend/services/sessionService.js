const pool = require('../../../config/database/db');
const jwt = require('jsonwebtoken');
const crypto = require('crypto');

const JWT_SECRET = process.env.JWT_SECRET || 'hjtpx-secret-key-change-in-production';
const SESSION_EXPIRY = 7 * 24 * 60 * 60 * 1000;

async function createSession(userId, deviceInfo = {}) {
  const sessionToken = crypto.randomBytes(64).toString('hex');
  const expiresAt = new Date(Date.now() + SESSION_EXPIRY);

  const result = await pool.query(
    `INSERT INTO sessions (user_id, token, expires_at, device_info, ip_address, user_agent)
     VALUES ($1, $2, $3, $4, $5, $6)
     RETURNING id, user_id, expires_at, created_at`,
    [
      userId,
      sessionToken,
      expiresAt,
      JSON.stringify(deviceInfo),
      deviceInfo.ipAddress || null,
      deviceInfo.userAgent || null
    ]
  );

  const session = result.rows[0];

  await logLoginHistory(userId, 'login', deviceInfo);

  return {
    sessionToken,
    expiresAt: session.expires_at,
    sessionId: session.id
  };
}

async function validateSession(token) {
  const result = await pool.query(
    `SELECT s.*, u.id as user_id, u.email, u.name, u.role
     FROM sessions s
     JOIN users u ON s.user_id = u.id
     WHERE s.token = $1`,
    [token]
  );

  if (result.rows.length === 0) {
    throw new Error('Session not found');
  }

  const session = result.rows[0];

  if (new Date(session.expires_at) < new Date()) {
    await deleteSession(token);
    throw new Error('Session expired');
  }

  if (session.is_revoked) {
    throw new Error('Session revoked');
  }

  return {
    id: session.id,
    userId: session.user_id,
    email: session.email,
    name: session.name,
    role: session.role,
    createdAt: session.created_at,
    expiresAt: session.expires_at
  };
}

async function refreshSession(token) {
  const session = await validateSession(token);

  const newExpiresAt = new Date(Date.now() + SESSION_EXPIRY);
  const newToken = crypto.randomBytes(64).toString('hex');

  await pool.query(
    `UPDATE sessions
     SET token = $1, expires_at = $2, last_activity = CURRENT_TIMESTAMP
     WHERE token = $3`,
    [newToken, newExpiresAt, token]
  );

  return {
    sessionToken: newToken,
    expiresAt: newExpiresAt
  };
}

async function deleteSession(token) {
  await pool.query('DELETE FROM sessions WHERE token = $1', [token]);
  return { message: 'Session deleted successfully' };
}

async function deleteUserSessions(userId, exceptToken = null) {
  if (exceptToken) {
    await pool.query(
      'DELETE FROM sessions WHERE user_id = $1 AND token != $2',
      [userId, exceptToken]
    );
  } else {
    await pool.query('DELETE FROM sessions WHERE user_id = $1', [userId]);
  }
  return { message: 'User sessions deleted successfully' };
}

async function getUserSessions(userId) {
  const result = await pool.query(
    `SELECT id, expires_at, created_at, last_activity, device_info, ip_address, user_agent, is_current
     FROM sessions
     WHERE user_id = $1
     ORDER BY created_at DESC`,
    [userId]
  );

  return result.rows.map(row => ({
    id: row.id,
    expiresAt: row.expires_at,
    createdAt: row.created_at,
    lastActivity: row.last_activity,
    deviceInfo: row.device_info,
    ipAddress: row.ip_address,
    userAgent: row.user_agent,
    isCurrent: row.is_current,
    isActive: new Date(row.expires_at) > new Date() && !row.is_revoked
  }));
}

async function cleanupExpiredSessions() {
  const result = await pool.query(
    'DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP OR is_revoked = true RETURNING id'
  );
  return { deletedCount: result.rowCount };
}

async function revokeSession(sessionId, userId) {
  await pool.query(
    'UPDATE sessions SET is_revoked = true WHERE id = $1 AND user_id = $2',
    [sessionId, userId]
  );
  return { message: 'Session revoked successfully' };
}

async function revokeAllUserSessions(userId) {
  await pool.query(
    'UPDATE sessions SET is_revoked = true WHERE user_id = $1',
    [userId]
  );
  return { message: 'All sessions revoked successfully' };
}

async function logLoginHistory(userId, action, deviceInfo = {}) {
  await pool.query(
    `INSERT INTO login_history (user_id, action, ip_address, user_agent, device_info, success)
     VALUES ($1, $2, $3, $4, $5, $6)`,
    [
      userId,
      action,
      deviceInfo.ipAddress || null,
      deviceInfo.userAgent || null,
      JSON.stringify(deviceInfo),
      action !== 'failed'
    ]
  );
}

async function getLoginHistory(userId, limit = 10) {
  const result = await pool.query(
    `SELECT * FROM login_history
     WHERE user_id = $1
     ORDER BY created_at DESC
     LIMIT $2`,
    [userId, limit]
  );

  return result.rows;
}

async function getActiveSessionsCount(userId) {
  const result = await pool.query(
    `SELECT COUNT(*) as count FROM sessions
     WHERE user_id = $1
     AND expires_at > CURRENT_TIMESTAMP
     AND is_revoked = false`,
    [userId]
  );

  return parseInt(result.rows[0].count);
}

async function enforceMaxSessions(userId, maxSessions = 5) {
  const result = await pool.query(
    `SELECT id FROM sessions
     WHERE user_id = $1
     AND expires_at > CURRENT_TIMESTAMP
     AND is_revoked = false
     ORDER BY last_activity ASC
     LIMIT $2`,
    [userId, Math.max(0, await getActiveSessionsCount(userId) - maxSessions)]
  );

  if (result.rows.length > 0) {
    const idsToDelete = result.rows.map(row => row.id);
    await pool.query(
      'DELETE FROM sessions WHERE id = ANY($1)',
      [idsToDelete]
    );
  }

  return { removedCount: result.rows.length };
}

module.exports = {
  createSession,
  validateSession,
  refreshSession,
  deleteSession,
  deleteUserSessions,
  getUserSessions,
  cleanupExpiredSessions,
  revokeSession,
  revokeAllUserSessions,
  logLoginHistory,
  getLoginHistory,
  getActiveSessionsCount,
  enforceMaxSessions
};
