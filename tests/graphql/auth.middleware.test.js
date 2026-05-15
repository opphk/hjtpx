const {
  requireAuth,
  requireAdmin,
  requireModerator,
  requireOwnerOrAdmin,
  checkPermission
} = require('../../src/backend/graphql/middleware/auth');

describe('GraphQL Auth Middleware', () => {
  describe('requireAuth', () => {
    it('should return user when authenticated', () => {
      const context = { user: { id: '1', role: 'user' } };
      
      const result = requireAuth(context);
      
      expect(result).toEqual(context.user);
    });

    it('should throw error when not authenticated', () => {
      const context = { user: null };
      
      expect(() => requireAuth(context)).toThrow('Authentication required');
    });
  });

  describe('requireAdmin', () => {
    it('should return user when user is admin', () => {
      const context = { user: { id: '1', role: 'admin' } };
      
      const result = requireAdmin(context);
      
      expect(result).toEqual(context.user);
    });

    it('should throw error when user is not admin', () => {
      const context = { user: { id: '1', role: 'user' } };
      
      expect(() => requireAdmin(context)).toThrow('Admin privileges required');
    });

    it('should throw error when not authenticated', () => {
      const context = { user: null };
      
      expect(() => requireAdmin(context)).toThrow('Authentication required');
    });
  });

  describe('requireModerator', () => {
    it('should return user when user is admin', () => {
      const context = { user: { id: '1', role: 'admin' } };
      
      const result = requireModerator(context);
      
      expect(result).toEqual(context.user);
    });

    it('should return user when user is moderator', () => {
      const context = { user: { id: '1', role: 'moderator' } };
      
      const result = requireModerator(context);
      
      expect(result).toEqual(context.user);
    });

    it('should throw error when user is not moderator or admin', () => {
      const context = { user: { id: '1', role: 'user' } };
      
      expect(() => requireModerator(context)).toThrow('Moderator privileges required');
    });
  });

  describe('requireOwnerOrAdmin', () => {
    it('should return user when user is the resource owner', () => {
      const context = { user: { id: '1', role: 'user' } };
      
      const result = requireOwnerOrAdmin(context, '1');
      
      expect(result).toEqual(context.user);
    });

    it('should return user when user is admin', () => {
      const context = { user: { id: '1', role: 'admin' } };
      
      const result = requireOwnerOrAdmin(context, '2');
      
      expect(result).toEqual(context.user);
    });

    it('should throw error when user is not owner or admin', () => {
      const context = { user: { id: '1', role: 'user' } };
      
      expect(() => requireOwnerOrAdmin(context, '2')).toThrow('You can only access your own resources');
    });
  });

  describe('checkPermission', () => {
    it('should return user when role is in allowed roles', () => {
      const context = { user: { id: '1', role: 'admin' } };
      
      const result = checkPermission(context, ['admin', 'moderator']);
      
      expect(result).toEqual(context.user);
    });

    it('should throw error when role is not in allowed roles', () => {
      const context = { user: { id: '1', role: 'user' } };
      
      expect(() => checkPermission(context, ['admin', 'moderator']))
        .toThrow('Insufficient permissions');
    });
  });
});
