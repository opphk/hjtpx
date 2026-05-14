const ROLES = {
  ADMIN: 'admin',
  USER: 'user',
  GUEST: 'guest'
};

const ROLE_HIERARCHY = {
  admin: 3,
  user: 2,
  guest: 1
};

function checkRole(...allowedRoles) {
  return (req, res, next) => {
    if (!req.user) {
      return res.status(401).json({
        success: false,
        error: 'Authentication required'
      });
    }

    const userRole = req.user.role;
    if (!allowedRoles.includes(userRole)) {
      return res.status(403).json({
        success: false,
        error: 'Insufficient permissions'
      });
    }

    next();
  };
}

function checkMinimumRole(minimumRole) {
  return (req, res, next) => {
    if (!req.user) {
      return res.status(401).json({
        success: false,
        error: 'Authentication required'
      });
    }

    const userRoleLevel = ROLE_HIERARCHY[req.user.role] || 0;
    const requiredLevel = ROLE_HIERARCHY[minimumRole] || 0;

    if (userRoleLevel < requiredLevel) {
      return res.status(403).json({
        success: false,
        error: 'Insufficient permissions'
      });
    }

    next();
  };
}

function checkPermission(...permissions) {
  return (req, res, next) => {
    if (!req.user) {
      return res.status(401).json({
        success: false,
        error: 'Authentication required'
      });
    }

    const userPermissions = req.user.permissions || [];
    const hasPermission = permissions.every(perm => userPermissions.includes(perm));

    if (!hasPermission) {
      return res.status(403).json({
        success: false,
        error: 'Insufficient permissions'
      });
    }

    next();
  };
}

function requireAdmin(req, res, next) {
  return checkRole(ROLES.ADMIN)(req, res, next);
}

function requireUser(req, res, next) {
  return checkMinimumRole(ROLES.USER)(req, res, next);
}

function requireGuest(req, res, next) {
  return checkMinimumRole(ROLES.GUEST)(req, res, next);
}

function isAdmin(req) {
  return req.user && req.user.role === ROLES.ADMIN;
}

function isUser(req) {
  return req.user && (req.user.role === ROLES.USER || req.user.role === ROLES.ADMIN);
}

function isGuest(req) {
  return req.user && (req.user.role === ROLES.GUEST || req.user.role === ROLES.USER || req.user.role === ROLES.ADMIN);
}

module.exports = {
  ROLES,
  ROLE_HIERARCHY,
  checkRole,
  checkMinimumRole,
  checkPermission,
  requireAdmin,
  requireUser,
  requireGuest,
  isAdmin,
  isUser,
  isGuest
};
