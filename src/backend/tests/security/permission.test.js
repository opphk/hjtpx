jest.mock('../../../../config/database/db', () => ({
  query: jest.fn()
}));

jest.mock('bcrypt', () => ({
  compare: jest.fn()
}));

const {
  checkPermission,
  requirePermission,
  requireMinimumRole,
  PERMISSIONS
} = require('../../services/permissionService');

const pool = require('../../../../config/database/db');

describe('Permission Service', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('checkPermission', () => {
    test('should return true for admin with any permission', async () => {
      pool.query.mockResolvedValueOnce({ rows: [{ role: 'admin' }] });
      
      const result = await checkPermission(1, 'delete');
      expect(result).toBe(true);
    });

    test('should return false for user without admin permission', async () => {
      pool.query.mockResolvedValueOnce({ rows: [{ role: 'user' }] });
      
      const result = await checkPermission(1, 'manage_users');
      expect(result).toBe(false);
    });

    test('should return true for user with basic permission', async () => {
      pool.query.mockResolvedValueOnce({ rows: [{ role: 'user' }] });
      
      const result = await checkPermission(1, 'read');
      expect(result).toBe(true);
    });

    test('should return false for non-existent user', async () => {
      pool.query.mockResolvedValueOnce({ rows: [] });
      
      const result = await checkPermission(999, 'read');
      expect(result).toBe(false);
    });
  });

  describe('requirePermission middleware', () => {
    test('should call next for authorized user', async () => {
      pool.query.mockResolvedValueOnce({ rows: [{ role: 'admin' }] });
      
      const middleware = requirePermission('delete');
      const req = { user: { id: 1 } };
      const res = {
        status: jest.fn().mockReturnThis(),
        json: jest.fn()
      };
      const next = jest.fn();

      await middleware(req, res, next);
      expect(next).toHaveBeenCalled();
    });

    test('should return 401 for unauthenticated user', async () => {
      const middleware = requirePermission('delete');
      const req = {};
      const res = {
        status: jest.fn().mockReturnThis(),
        json: jest.fn()
      };
      const next = jest.fn();

      await middleware(req, res, next);
      
      expect(res.status).toHaveBeenCalledWith(401);
      expect(res.json).toHaveBeenCalledWith({
        success: false,
        error: 'Authentication required'
      });
      expect(next).not.toHaveBeenCalled();
    });

    test('should return 403 for unauthorized user', async () => {
      pool.query.mockResolvedValueOnce({ rows: [{ role: 'user' }] });
      
      const middleware = requirePermission('manage_users');
      const req = { user: { id: 1, ip: '127.0.0.1' } };
      const res = {
        status: jest.fn().mockReturnThis(),
        json: jest.fn()
      };
      const next = jest.fn();

      await middleware(req, res, next);
      
      expect(res.status).toHaveBeenCalledWith(403);
      expect(res.json).toHaveBeenCalledWith({
        success: false,
        error: 'Permission denied'
      });
    });
  });

  describe('requireMinimumRole middleware', () => {
    test('should allow higher role access', async () => {
      const middleware = requireMinimumRole('moderator');
      const req = { user: { id: 1, role: 'admin' } };
      const res = {
        status: jest.fn().mockReturnThis(),
        json: jest.fn()
      };
      const next = jest.fn();

      await middleware(req, res, next);
      expect(next).toHaveBeenCalled();
    });

    test('should allow same role access', async () => {
      const middleware = requireMinimumRole('moderator');
      const req = { user: { id: 1, role: 'moderator' } };
      const res = {
        status: jest.fn().mockReturnThis(),
        json: jest.fn()
      };
      const next = jest.fn();

      await middleware(req, res, next);
      expect(next).toHaveBeenCalled();
    });

    test('should deny lower role access', async () => {
      const middleware = requireMinimumRole('moderator');
      const req = { user: { id: 1, role: 'user' } };
      const res = {
        status: jest.fn().mockReturnThis(),
        json: jest.fn()
      };
      const next = jest.fn();

      await middleware(req, res, next);
      
      expect(res.status).toHaveBeenCalledWith(403);
      expect(res.json).toHaveBeenCalledWith({
        success: false,
        error: 'Insufficient role privileges'
      });
    });
  });

  describe('PERMISSIONS constant', () => {
    test('should define correct admin permissions', () => {
      expect(PERMISSIONS.admin).toContain('read');
      expect(PERMISSIONS.admin).toContain('write');
      expect(PERMISSIONS.admin).toContain('delete');
      expect(PERMISSIONS.admin).toContain('manage_users');
    });

    test('should define correct moderator permissions', () => {
      expect(PERMISSIONS.moderator).toContain('read');
      expect(PERMISSIONS.moderator).toContain('write');
      expect(PERMISSIONS.moderator).toContain('delete');
    });

    test('should define correct user permissions', () => {
      expect(PERMISSIONS.user).toContain('read');
      expect(PERMISSIONS.user).toContain('write');
      expect(PERMISSIONS.user).not.toContain('delete');
    });
  });
});
