const bcrypt = require('bcryptjs');

jest.mock('../../config/database/db', () => ({
  query: jest.fn().mockResolvedValue({ rows: [] }),
  pool: {
    query: jest.fn().mockResolvedValue({ rows: [] })
  }
}));

jest.mock('../../utils/queryOptimizer', () => ({
  cachedQuery: jest.fn().mockResolvedValue([]),
  clearCache: jest.fn().mockResolvedValue(undefined)
}));

jest.mock('../../services/authService', () => ({
  validatePassword: jest.fn().mockReturnValue(true)
}));

jest.mock('../../services/cacheService', () => ({
  get: jest.fn().mockResolvedValue(null),
  set: jest.fn().mockResolvedValue(true),
  del: jest.fn().mockResolvedValue(true)
}));

const pool = require('../../config/database/db');
const userService = require('../../services/userService');

describe('userService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    pool.query.mockResolvedValue({ rows: [] });
  });

  describe('VALID_ROLES', () => {
    test('should export valid roles', () => {
      expect(userService.VALID_ROLES).toContain('admin');
      expect(userService.VALID_ROLES).toContain('user');
      expect(userService.VALID_ROLES).toContain('moderator');
    });
  });

  describe('createUser', () => {
    test('should create a new user successfully', async () => {
      const newUser = {
        id: 1,
        email: 'newuser@example.com',
        name: 'New User',
        role: 'user',
        created_at: new Date()
      };

      pool.query.mockResolvedValueOnce({ rows: [newUser] });

      const result = await userService.createUser({
        email: 'newuser@example.com',
        name: 'New User',
        password: 'Password123'
      });

      expect(result).toEqual(newUser);
      expect(pool.query).toHaveBeenCalled();
    });

    test('should throw error for weak password', async () => {
      await expect(
        userService.createUser({
          email: 'newuser@example.com',
          name: 'New User',
          password: 'weak'
        })
      ).rejects.toThrow('Password must be at least 8 characters long');
    });

    test('should throw error for empty password', async () => {
      await expect(
        userService.createUser({
          email: 'newuser@example.com',
          name: 'New User',
          password: ''
        })
      ).rejects.toThrow('Password must be at least 8 characters long');
    });

    test('should throw error for invalid role', async () => {
      await expect(
        userService.createUser({
          email: 'newuser@example.com',
          name: 'New User',
          password: 'Password123',
          role: 'invalid_role'
        })
      ).rejects.toThrow('Invalid role. Must be one of:');
    });

    test('should create user with admin role', async () => {
      const newUser = {
        id: 1,
        email: 'admin@example.com',
        name: 'Admin User',
        role: 'admin',
        created_at: new Date()
      };

      pool.query.mockResolvedValueOnce({ rows: [newUser] });

      const result = await userService.createUser({
        email: 'admin@example.com',
        name: 'Admin User',
        password: 'Password123',
        role: 'admin'
      });

      expect(result.role).toBe('admin');
    });

    test('should hash password before storing', async () => {
      const newUser = {
        id: 1,
        email: 'newuser@example.com',
        name: 'New User',
        role: 'user',
        created_at: new Date()
      };

      pool.query.mockResolvedValueOnce({ rows: [newUser] });

      await userService.createUser({
        email: 'newuser@example.com',
        name: 'New User',
        password: 'Password123'
      });

      const queryCall = pool.query.mock.calls[0];
      const storedPassword = queryCall[1][2];
      expect(storedPassword).toMatch(/^\$2[aby]?\$/);
    });
  });

  describe('updateUser', () => {
    test('should update user name successfully', async () => {
      const updatedUser = {
        id: 1,
        email: 'user@example.com',
        name: 'Updated Name',
        role: 'user'
      };

      pool.query.mockResolvedValueOnce({ rows: [updatedUser] });

      const result = await userService.updateUser(1, { name: 'Updated Name' });

      expect(result.name).toBe('Updated Name');
    });

    test('should update user email successfully', async () => {
      const updatedUser = {
        id: 1,
        email: 'newemail@example.com',
        name: 'Test User',
        role: 'user'
      };

      pool.query.mockResolvedValueOnce({ rows: [updatedUser] });

      const result = await userService.updateUser(1, { email: 'newemail@example.com' });

      expect(result.email).toBe('newemail@example.com');
    });

    test('should update user role successfully', async () => {
      const updatedUser = {
        id: 1,
        email: 'user@example.com',
        name: 'Test User',
        role: 'moderator'
      };

      pool.query.mockResolvedValueOnce({ rows: [updatedUser] });

      const result = await userService.updateUser(1, { role: 'moderator' });

      expect(result.role).toBe('moderator');
    });

    test('should throw error for invalid role update', async () => {
      await expect(userService.updateUser(1, { role: 'invalid_role' })).rejects.toThrow(
        'Invalid role. Must be one of:'
      );
    });

    test('should update multiple fields at once', async () => {
      const updatedUser = {
        id: 1,
        email: 'newemail@example.com',
        name: 'Updated Name',
        role: 'moderator'
      };

      pool.query.mockResolvedValueOnce({ rows: [updatedUser] });

      const result = await userService.updateUser(1, {
        email: 'newemail@example.com',
        name: 'Updated Name',
        role: 'moderator'
      });

      expect(result.email).toBe('newemail@example.com');
      expect(result.name).toBe('Updated Name');
      expect(result.role).toBe('moderator');
    });
  });

  describe('deleteUser', () => {
    test('should delete user successfully', async () => {
      pool.query.mockResolvedValueOnce({ rows: [] });

      await userService.deleteUser(1);

      expect(pool.query).toHaveBeenCalledWith('DELETE FROM users WHERE id = $1', [1]);
    });
  });
});
