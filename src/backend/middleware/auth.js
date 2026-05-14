const jwt = require('jsonwebtoken');
const sessionService = require('../services/sessionService');
const pool = require('../../../config/database/db');

const JWT_SECRET = process.env.JWT_SECRET || 'hjtpx-secret-key-change-in-production';
const JWT_EXPIRES_IN = process.env.JWT_EXPIRES_IN || '7d';
const MAX_LOGIN_ATTEMPTS = 5;
const LOCKOUT_DURATION = 15 * 60 * 1000;
const MAX_CONCURRENT_SESSIONS = 5;

async function isAccountLocked(email) {
  const result = await pool.query(
    `SELECT COUNT(*) as attempts FROM login_attempts
     WHERE email = $1
     AND attempted_at > NOW() - INTERVAL '1 minute' * ($2 / 60)
     AND success = false`,
    [email, LOCKOUT_DURATION / 60000]
  );

  return parseInt(result.rows[0].attempts) >= MAX_LOGIN_ATTEMPTS;
}

async function recordLoginAttempt(email, success, ipAddress) {
  await pool.query(
    `INSERT INTO login_attempts (email, ip_address, success)
     VALUES ($1, $2, $3)`,
    [email, ipAddress, success]
  );
}

async function getLoginAttempts(email) {
  const result = await pool.query(
    `SELECT * FROM login_attempts
     WHERE email = $1
     ORDER BY attempted_at DESC
     LIMIT $2`,
    [email, 10]
  );
  return result.rows;
}

async function clearLoginAttempts(email) {
  await pool.query(
    `DELETE FROM login_attempts WHERE email = $1`,
    [email]
  );
}

function auth(req, res, next) {
  return authenticate(req, res, next);
}

async function authenticate(req, res, next) {
  try {
    const token = req.headers.authorization?.split(' ')[1];
    if (!token) {
      return res.status(401).json({
        success: false,
        error: 'No token provided'
      });
    }

    const decoded = jwt.verify(token, JWT_SECRET);
    req.user = decoded;
    next();
  } catch (error) {
    res.status(401).json({
      success: false,
      error: 'Invalid token'
    });
  }
}

async function authenticateWithSession(req, res, next) {
  try {
    const token = req.headers.authorization?.split(' ')[1];
    if (!token) {
      return res.status(401).json({
        success: false,
        error: 'No token provided'
      });
    }

    const session = await sessionService.validateSession(token);
    req.user = {
      id: session.userId,
      email: session.email,
      name: session.name,
      role: session.role,
      sessionId: session.id
    };
    req.session = session;
    next();
  } catch (error) {
    res.status(401).json({
      success: false,
      error: 'Invalid or expired session'
    });
  }
}

async function multiDeviceAuth(req, res, next) {
  try {
    const token = req.headers.authorization?.split(' ')[1];
    if (!token) {
      return res.status(401).json({
        success: false,
        error: 'No token provided'
      });
    }

    const decoded = jwt.verify(token, JWT_SECRET);
    const activeSessions = await sessionService.getActiveSessionsCount(decoded.id);

    if (activeSessions >= MAX_CONCURRENT_SESSIONS) {
      await sessionService.enforceMaxSessions(decoded.id, MAX_CONCURRENT_SESSIONS - 1);
    }

    req.user = decoded;
    req.sessionCount = activeSessions;
    next();
  } catch (error) {
    res.status(401).json({
      success: false,
      error: 'Invalid token'
    });
  }
}

async function createToken(user) {
  return jwt.sign(
    {
      id: user.id,
      email: user.email,
      name: user.name,
      role: user.role
    },
    JWT_SECRET,
    { expiresIn: JWT_EXPIRES_IN }
  );
}

async function verifyToken(token) {
  return jwt.verify(token, JWT_SECRET);
}

function optionalAuth(req, res, next) {
  const token = req.headers.authorization?.split(' ')[1];
  if (!token) {
    return next();
  }

  try {
    const decoded = jwt.verify(token, JWT_SECRET);
    req.user = decoded;
  } catch (error) {
  }

  next();
}

async function requireVerifiedEmail(req, res, next) {
  if (!req.user) {
    return res.status(401).json({
      success: false,
      error: 'Authentication required'
    });
  }

  const result = await pool.query(
    'SELECT email_verified FROM users WHERE id = $1',
    [req.user.id]
  );

  if (result.rows.length > 0 && result.rows[0].email_verified) {
    return next();
  }

  res.status(403).json({
    success: false,
    error: 'Email verification required'
  });
}

async function getDeviceInfo(req) {
  return {
    ipAddress: req.ip || req.connection.remoteAddress,
    userAgent: req.headers['user-agent'],
    deviceInfo: {
      browser: req.headers['user-agent']?.includes('Chrome') ? 'Chrome' :
               req.headers['user-agent']?.includes('Firefox') ? 'Firefox' :
               req.headers['user-agent']?.includes('Safari') ? 'Safari' : 'Unknown',
      os: req.headers['user-agent']?.includes('Windows') ? 'Windows' :
          req.headers['user-agent']?.includes('Mac') ? 'macOS' :
          req.headers['user-agent']?.includes('Linux') ? 'Linux' : 'Unknown',
      mobile: /mobile|android|iphone|ipad|tablet/i.test(req.headers['user-agent'] || '')
    }
  };
}

module.exports = {
  auth,
  authenticate,
  authenticateWithSession,
  multiDeviceAuth,
  createToken,
  verifyToken,
  optionalAuth,
  requireVerifiedEmail,
  isAccountLocked,
  recordLoginAttempt,
  getLoginAttempts,
  clearLoginAttempts,
  getDeviceInfo,
  MAX_LOGIN_ATTEMPTS,
  LOCKOUT_DURATION,
  MAX_CONCURRENT_SESSIONS
};
