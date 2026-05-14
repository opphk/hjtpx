const pool = require('../../../config/database/db');
const bcrypt = require('bcrypt');
const crypto = require('crypto');

const SESSION_TIMEOUT = 30 * 60 * 1000;
const MAX_FAILED_ATTEMPTS = 5;
const LOCKOUT_DURATION = 15 * 60 * 1000;
const SENSITIVE_OPERATIONS = ['delete', 'update_role', 'reset_password', 'export_data', 'import_data', 'change_settings'];

const PERMISSIONS = {
  admin: ['read', 'write', 'delete', 'manage_users', 'manage_settings', 'view_audit', 'export', 'import'],
  moderator: ['read', 'write', 'delete', 'view_audit'],
  user: ['read', 'write']
};

const SENSITIVE_OPERATION_RULES = {
  delete: { requireReauth: true, requireReason: false },
  update_role: { requireReauth: true, requireReason: true },
  reset_password: { requireReauth: true, requireReason: false },
  export_data: { requireReauth: false, requireReason: true },
  import_data: { requireReauth: false, requireReason: true },
  change_settings: { requireReauth: true, requireReason: true }
};

async function checkPermission(userId, permission) {
  const result = await pool.query(
    'SELECT role FROM users WHERE id = $1',
    [userId]
  );

  if (result.rows.length === 0) {
    return false;
  }

  const userRole = result.rows[0].role;
  const rolePermissions = PERMISSIONS[userRole] || [];

  return rolePermissions.includes(permission);
}

async function checkMultiplePermissions(userId, permissions) {
  const results = {};
  for (const permission of permissions) {
    results[permission] = await checkPermission(userId, permission);
  }
  return results;
}

function requirePermission(permission) {
  return async (req, res, next) => {
    if (!req.user) {
      return res.status(401).json({
        success: false,
        error: 'Authentication required'
      });
    }

    const hasPermission = await checkPermission(req.user.id, permission);

    if (!hasPermission) {
      await logSecurityEvent('PERMISSION_DENIED', {
        userId: req.user.id,
        permission,
        path: req.path,
        ip: req.ip
      });

      return res.status(403).json({
        success: false,
        error: 'Permission denied'
      });
    }

    next();
  };
}

function requireAnyPermission(permissions) {
  return async (req, res, next) => {
    if (!req.user) {
      return res.status(401).json({
        success: false,
        error: 'Authentication required'
      });
    }

    const results = await checkMultiplePermissions(req.user.id, permissions);

    if (!Object.values(results).some(Boolean)) {
      await logSecurityEvent('PERMISSION_DENIED', {
        userId: req.user.id,
        requiredPermissions: permissions,
        path: req.path,
        ip: req.ip
      });

      return res.status(403).json({
        success: false,
        error: 'Permission denied'
      });
    }

    next();
  };
}

function requireAllPermissions(permissions) {
  return async (req, res, next) => {
    if (!req.user) {
      return res.status(401).json({
        success: false,
        error: 'Authentication required'
      });
    }

    const results = await checkMultiplePermissions(req.user.id, permissions);

    if (!Object.values(results).every(Boolean)) {
      await logSecurityEvent('PERMISSION_DENIED', {
        userId: req.user.id,
        requiredPermissions: permissions,
        path: req.path,
        ip: req.ip
      });

      return res.status(403).json({
        success: false,
        error: 'Permission denied'
      });
    }

    next();
  };
}

function requireMinimumRole(minimumRole) {
  const roleHierarchy = ['user', 'moderator', 'admin'];
  const minimumIndex = roleHierarchy.indexOf(minimumRole);

  return async (req, res, next) => {
    if (!req.user) {
      return res.status(401).json({
        success: false,
        error: 'Authentication required'
      });
    }

    const userRoleIndex = roleHierarchy.indexOf(req.user.role);

    if (userRoleIndex < minimumIndex) {
      await logSecurityEvent('ROLE_INSUFFICIENT', {
        userId: req.user.id,
        userRole: req.user.role,
        requiredRole: minimumRole,
        path: req.path,
        ip: req.ip
      });

      return res.status(403).json({
        success: false,
        error: 'Insufficient role privileges'
      });
    }

    next();
  };
}

async function verifyOperationAllowed(userId, operation, additionalData = {}) {
  if (!SENSITIVE_OPERATIONS.includes(operation)) {
    return { allowed: true };
  }

  const rule = SENSITIVE_OPERATION_RULES[operation];

  if (rule.requireReason && !additionalData.reason) {
    return { allowed: false, reason: 'Reason is required for this operation' };
  }

  return { allowed: true };
}

async function requireSensitiveOperationVerification(req, res, next) {
  const operation = req.params.operation || req.body?.operation;

  if (!operation || !SENSITIVE_OPERATIONS.includes(operation)) {
    return next();
  }

  if (!req.user) {
    return res.status(401).json({
      success: false,
      error: 'Authentication required'
    });
  }

  const rule = SENSITIVE_OPERATION_RULES[operation];

  if (rule.requireReauth) {
    const { password } = req.body;

    if (!password) {
      return res.status(400).json({
        success: false,
        error: 'Password confirmation required for this operation'
      });
    }

    const result = await pool.query(
      'SELECT password FROM users WHERE id = $1',
      [req.user.id]
    );

    if (result.rows.length === 0) {
      return res.status(401).json({
        success: false,
        error: 'User not found'
      });
    }

    const isValid = await bcrypt.compare(password, result.rows[0].password);

    if (!isValid) {
      await logSecurityEvent('SENSITIVE_OPERATION_REAUTH_FAILED', {
        userId: req.user.id,
        operation,
        ip: req.ip,
        userAgent: req.headers['user-agent']
      });

      return res.status(401).json({
        success: false,
        error: 'Invalid password'
      });
    }
  }

  if (rule.requireReason) {
    const { reason } = req.body;

    if (!reason || reason.trim().length < 10) {
      return res.status(400).json({
        success: false,
        error: 'A detailed reason (at least 10 characters) is required for this operation'
      });
    }
  }

  await logSecurityEvent('SENSITIVE_OPERATION_VERIFIED', {
    userId: req.user.id,
    operation,
    reason: req.body.reason,
    ip: req.ip
  });

  next();
}

async function createAuditLog(userId, action, resourceType, resourceId, additionalData = {}) {
  const {
    ipAddress,
    userAgent,
    requestData,
    responseStatus
  } = additionalData;

  await pool.query(
    `INSERT INTO audit_logs 
     (user_id, action, resource_type, resource_id, ip_address, user_agent, request_data, response_status)
     VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
    [
      userId,
      action,
      resourceType,
      resourceId,
      ipAddress || null,
      userAgent || null,
      requestData ? JSON.stringify(requestData) : null,
      responseStatus || null
    ]
  );
}

async function getAuditLogs(options = {}) {
  const {
    userId,
    action,
    resourceType,
    startDate,
    endDate,
    page = 1,
    limit = 50
  } = options;

  const conditions = [];
  const params = [];
  let paramIndex = 1;

  if (userId) {
    conditions.push(`user_id = $${paramIndex++}`);
    params.push(userId);
  }

  if (action) {
    conditions.push(`action = $${paramIndex++}`);
    params.push(action);
  }

  if (resourceType) {
    conditions.push(`resource_type = $${paramIndex++}`);
    params.push(resourceType);
  }

  if (startDate) {
    conditions.push(`created_at >= $${paramIndex++}`);
    params.push(startDate);
  }

  if (endDate) {
    conditions.push(`created_at <= $${paramIndex++}`);
    params.push(endDate);
  }

  const whereClause = conditions.length > 0 ? `WHERE ${conditions.join(' AND ')}` : '';
  const offset = (page - 1) * limit;

  const [logs, countResult] = await Promise.all([
    pool.query(
      `SELECT * FROM audit_logs ${whereClause} ORDER BY created_at DESC LIMIT $${paramIndex} OFFSET $${paramIndex + 1}`,
      [...params, limit, offset]
    ),
    pool.query(
      `SELECT COUNT(*) FROM audit_logs ${whereClause}`,
      params
    )
  ]);

  return {
    logs: logs.rows,
    pagination: {
      page,
      limit,
      total: parseInt(countResult.rows[0].count),
      pages: Math.ceil(parseInt(countResult.rows[0].count) / limit)
    }
  };
}

async function logSecurityEvent(eventType, details) {
  try {
    await pool.query(
      `INSERT INTO security_events (event_type, user_id, ip_address, user_agent, details)
       VALUES ($1, $2, $3, $4, $5)`,
      [
        eventType,
        details.userId || null,
        details.ip || null,
        details.userAgent || null,
        JSON.stringify(details)
      ]
    );
  } catch (error) {
    console.error('Failed to log security event:', error);
  }
}

async function detectAnomalousBehavior(userId, ipAddress) {
  const recentWindow = new Date(Date.now() - 60 * 60 * 1000);
  const suspiciousThreshold = 10;
  const failedLoginThreshold = 3;

  const [failedLogins, securityEvents, auditEvents] = await Promise.all([
    pool.query(
      `SELECT COUNT(*) as count FROM login_attempts 
       WHERE user_id = $1 AND success = false AND attempted_at > $2`,
      [userId, recentWindow]
    ),
    pool.query(
      `SELECT COUNT(*) as count FROM security_events 
       WHERE user_id = $1 AND created_at > $2`,
      [userId, recentWindow]
    ),
    pool.query(
      `SELECT COUNT(*) as count FROM audit_logs 
       WHERE user_id = $1 AND created_at > $2 AND action IN ('DELETE', 'UPDATE_ROLE', 'EXPORT_DATA')`,
      [userId, recentWindow]
    )
  ]);

  const failedLoginCount = parseInt(failedLogins.rows[0].count);
  const securityEventCount = parseInt(securityEvents.rows[0].count);
  const sensitiveOperationCount = parseInt(auditEvents.rows[0].count);

  const anomalies = [];

  if (failedLoginCount >= failedLoginThreshold) {
    anomalies.push({
      type: 'EXCESSIVE_FAILED_LOGINS',
      severity: 'high',
      count: failedLoginCount
    });
  }

  if (securityEventCount >= suspiciousThreshold) {
    anomalies.push({
      type: 'HIGH_SECURITY_EVENT_RATE',
      severity: 'high',
      count: securityEventCount
    });
  }

  if (sensitiveOperationCount >= 5) {
    anomalies.push({
      type: 'EXCESSIVE_SENSITIVE_OPERATIONS',
      severity: 'medium',
      count: sensitiveOperationCount
    });
  }

  const [ipChangeResult] = await pool.query(
    `SELECT COUNT(DISTINCT ip_address) as count FROM audit_logs 
     WHERE user_id = $1 AND created_at > $2`,
    [userId, recentWindow]
  );

  if (parseInt(ipChangeResult.rows[0].count) > 3) {
    anomalies.push({
      type: 'MULTIPLE_IP_CHANGE',
      severity: 'medium',
      count: parseInt(ipChangeResult.rows[0].count)
    });
  }

  if (anomalies.length > 0) {
    await logSecurityEvent('ANOMALOUS_BEHAVIOR_DETECTED', {
      userId,
      ipAddress,
      anomalies
    });
  }

  return {
    isAnomalous: anomalies.length > 0,
    anomalies
  };
}

async function lockAccount(userId, reason, lockedBy = null, lockedUntil = null) {
  const lockDuration = lockedUntil || new Date(Date.now() + LOCKOUT_DURATION);

  await pool.query(
    `INSERT INTO account_locks (user_id, lock_reason, locked_until, locked_by)
     VALUES ($1, $2, $3, $4)`,
    [userId, reason, lockDuration, lockedBy]
  );

  await pool.query(
    `UPDATE users SET locked = true, locked_at = CURRENT_TIMESTAMP WHERE id = $1`,
    [userId]
  );

  await logSecurityEvent('ACCOUNT_LOCKED', {
    userId,
    reason,
    lockedUntil: lockDuration
  });

  return { locked: true, until: lockDuration };
}

async function unlockAccount(userId, unlockedBy) {
  await pool.query(
    `UPDATE account_locks SET unlocked_at = CURRENT_TIMESTAMP, unlocked_by = $2 
     WHERE user_id = $1 AND unlocked_at IS NULL`,
    [userId, unlockedBy]
  );

  await pool.query(
    `UPDATE users SET locked = false, locked_at = NULL WHERE id = $1`,
    [userId]
  );

  await logSecurityEvent('ACCOUNT_UNLOCKED', {
    userId,
    unlockedBy
  });

  return { unlocked: true };
}

async function isAccountLocked(userId) {
  const result = await pool.query(
    `SELECT * FROM account_locks 
     WHERE user_id = $1 AND locked_until > CURRENT_TIMESTAMP 
     ORDER BY locked_at DESC LIMIT 1`,
    [userId]
  );

  if (result.rows.length > 0) {
    const lock = result.rows[0];
    return {
      locked: true,
      reason: lock.lock_reason,
      until: lock.locked_until,
      lockedAt: lock.locked_at
    };
  }

  return { locked: false };
}

async function checkSessionTimeout(req, res, next) {
  if (!req.user || !req.user.sessionId) {
    return next();
  }

  const result = await pool.query(
    `SELECT last_activity FROM sessions WHERE id = $1`,
    [req.user.sessionId]
  );

  if (result.rows.length === 0) {
    return res.status(401).json({
      success: false,
      error: 'Session not found'
    });
  }

  const lastActivity = new Date(result.rows[0].last_activity);
  const now = new Date();

  if (now - lastActivity > SESSION_TIMEOUT) {
    await pool.query(
      `DELETE FROM sessions WHERE id = $1`,
      [req.user.sessionId]
    );

    await logSecurityEvent('SESSION_TIMEOUT', {
      userId: req.user.id,
      sessionId: req.user.sessionId,
      lastActivity: lastActivity.toISOString()
    });

    return res.status(401).json({
      success: false,
      error: 'Session expired due to inactivity'
    });
  }

  await pool.query(
    `UPDATE sessions SET last_activity = CURRENT_TIMESTAMP WHERE id = $1`,
    [req.user.sessionId]
  );

  next();
}

async function refreshSession(req, res, next) {
  if (!req.user || !req.user.sessionId) {
    return next();
  }

  try {
    await pool.query(
      `UPDATE sessions SET last_activity = CURRENT_TIMESTAMP WHERE id = $1`,
      [req.user.sessionId]
    );
  } catch (error) {
    console.error('Failed to refresh session:', error);
  }

  next();
}

function createPermissionChecker(options = {}) {
  const {
    requireTwoFactor = false,
    requireIpWhitelist = false,
    allowedIps = []
  } = options;

  return async (req, res, next) => {
    if (!req.user) {
      return res.status(401).json({
        success: false,
        error: 'Authentication required'
      });
    }

    if (requireIpWhitelist && allowedIps.length > 0) {
      const clientIp = req.ip || req.connection.remoteAddress;

      if (!allowedIps.includes(clientIp)) {
        await logSecurityEvent('IP_NOT_WHITELISTED', {
          userId: req.user.id,
          ip: clientIp,
          path: req.path
        });

        return res.status(403).json({
          success: false,
          error: 'Access from this IP is not allowed'
        });
      }
    }

    if (requireTwoFactor) {
      const result = await pool.query(
        'SELECT two_factor_enabled FROM users WHERE id = $1',
        [req.user.id]
      );

      if (result.rows.length > 0 && !result.rows[0].two_factor_enabled) {
        return res.status(403).json({
          success: false,
          error: 'Two-factor authentication is required'
        });
      }
    }

    const { isAnomalous } = await detectAnomalousBehavior(req.user.id, req.ip);

    if (isAnomalous) {
      await lockAccount(req.user.id, 'Anomalous behavior detected');

      return res.status(403).json({
        success: false,
        error: 'Account temporarily locked due to suspicious activity'
      });
    }

    next();
  };
}

module.exports = {
  checkPermission,
  checkMultiplePermissions,
  requirePermission,
  requireAnyPermission,
  requireAllPermissions,
  requireMinimumRole,
  verifyOperationAllowed,
  requireSensitiveOperationVerification,
  createAuditLog,
  getAuditLogs,
  logSecurityEvent,
  detectAnomalousBehavior,
  lockAccount,
  unlockAccount,
  isAccountLocked,
  checkSessionTimeout,
  refreshSession,
  createPermissionChecker,
  PERMISSIONS,
  SENSITIVE_OPERATIONS,
  SESSION_TIMEOUT,
  MAX_FAILED_ATTEMPTS,
  LOCKOUT_DURATION
};
