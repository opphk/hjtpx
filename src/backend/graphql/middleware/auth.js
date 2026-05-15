const { AuthenticationError, ForbiddenError } = require('apollo-server-express');

const requireAuth = (context) => {
  if (!context.user) {
    throw new AuthenticationError('Authentication required');
  }
  return context.user;
};

const requireAdmin = (context) => {
  const user = requireAuth(context);
  if (user.role !== 'admin') {
    throw new ForbiddenError('Admin privileges required');
  }
  return user;
};

const requireModerator = (context) => {
  const user = requireAuth(context);
  if (user.role !== 'admin' && user.role !== 'moderator') {
    throw new ForbiddenError('Moderator privileges required');
  }
  return user;
};

const requireOwnerOrAdmin = (context, resourceUserId) => {
  const user = requireAuth(context);
  if (user.id !== resourceUserId && user.role !== 'admin') {
    throw new ForbiddenError('You can only access your own resources');
  }
  return user;
};

const checkPermission = (context, requiredRoles) => {
  const user = requireAuth(context);
  if (!requiredRoles.includes(user.role)) {
    throw new ForbiddenError('Insufficient permissions');
  }
  return user;
};

module.exports = {
  requireAuth,
  requireAdmin,
  requireModerator,
  requireOwnerOrAdmin,
  checkPermission
};
