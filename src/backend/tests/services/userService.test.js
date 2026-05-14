const userService = require('../../services/userService');

jest.mock('../../../../src/config/database/db', () => ({
  query: jest.fn()
}));

jest.mock('../../../../src/backend/utils/queryOptimizer', () => ({
  cachedQuery: jest.fn(),
  clearCache: jest.fn()
}));

jest.mock('../../../../src/backend/services/cacheService', () => ({
  get: jest.fn(),
  set: jest.fn(),
  del: jest.fn()
}));

jest.mock('bcrypt', () => ({
  hash: jest.fn(),
  compare: jest.fn()
}));

const pool = require('../../../../src/config/database/db');
const queryOptimizer = require('../../../../src/backend/utils/queryOptimizer');
const cacheService = require('../../../../src/backend/services/cacheService');
const bcrypt = require('bcrypt');

describe('UserService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    cacheService.get.mockResolvedValue(null);
    cacheService.del.mockResolvedValue(undefined);
    cacheService.set.mockResolvedValue(undefined);
    queryOptimizer.clearCache.mockResolvedValue(undefined);
  });

  describe('getAllUsers', () => {
    it('should return all users', async () => {
      const mockUsers = [
        { id: 1, email: 'user1@example.com', name: 'User One', created_at: new Date() },
        { id: 2, email: 'user2@example.com', name: 'User Two', created_at: new Date() }
      ];

      queryOptimizer.cachedQuery.mockResolvedValue(mockUsers);

      const result = await userService.getAllUsers();

      expect(queryOptimizer.cachedQuery).toHaveBeenCalled();
      expect(result).toEqual(mockUsers);
      expect(result).toHaveLength(2);
    });

    it('should return empty array when no users exist', async () => {
      queryOptimizer.cachedQuery.mockResolvedValue([]);

      const result = await userService.getAllUsers();

      expect(result).toEqual([]);
      expect(result).toHaveLength(0);
    });

    it('should handle database errors', async () => {
      queryOptimizer.cachedQuery.mockRejectedValue(new Error('Database connection failed'));

      await expect(userService.getAllUsers()).rejects.toThrow('Database connection failed');
    });
  });

  describe('getUserById', () => {
    it('should return user when found', async () => {
      const mockUser = {
        id: 1,
        email: 'user@example.com',
        name: 'Test User',
        created_at: new Date()
      };

      queryOptimizer.cachedQuery.mockResolvedValue([mockUser]);

      const result = await userService.getUserById(1);

      expect(queryOptimizer.cachedQuery).toHaveBeenCalled();
      expect(result).toEqual(mockUser);
    });

    it('should return undefined when user not found', async () => {
      queryOptimizer.cachedQuery.mockResolvedValue([]);

      const result = await userService.getUserById(999);

      expect(result).toBeUndefined();
    });

    it('should handle database errors', async () => {
      queryOptimizer.cachedQuery.mockRejectedValue(new Error('Database error'));

      await expect(userService.getUserById(1)).rejects.toThrow('Database error');
    });
  });

  describe('createUser', () => {
    it('should create user with hashed password', async () => {
      const userData = { email: 'new@example.com', name: 'New User', password: 'Password123!' };
      const hashedPassword = 'hashed_password_123';
      const mockCreatedUser = {
        id: 3,
        email: userData.email,
        name: userData.name,
        password: hashedPassword,
        created_at: new Date()
      };

      bcrypt.hash.mockResolvedValue(hashedPassword);
      pool.query.mockResolvedValue({ rows: [mockCreatedUser] });

      const result = await userService.createUser(userData);

      expect(bcrypt.hash).toHaveBeenCalledWith(userData.password, 10);
      expect(pool.query).toHaveBeenCalled();
      expect(result).toEqual(mockCreatedUser);
      expect(result.password).toBe(hashedPassword);
    });

    it('should handle password hashing errors', async () => {
      const userData = { email: 'new@example.com', name: 'New User', password: 'Password123!' };

      bcrypt.hash.mockRejectedValue(new Error('Hashing failed'));

      await expect(userService.createUser(userData)).rejects.toThrow('Hashing failed');
    });

    it('should handle database errors during user creation', async () => {
      const userData = { email: 'new@example.com', name: 'New User', password: 'Password123!' };

      bcrypt.hash.mockResolvedValue('hashed');
      pool.query.mockRejectedValue(new Error('Insert failed'));

      await expect(userService.createUser(userData)).rejects.toThrow('Insert failed');
    });
  });

  describe('updateUser', () => {
    it('should update user email', async () => {
      const mockUpdatedUser = {
        id: 1,
        email: 'updated@example.com',
        name: 'Test User',
        created_at: new Date()
      };

      pool.query.mockResolvedValue({ rows: [mockUpdatedUser] });

      const result = await userService.updateUser(1, { email: 'updated@example.com' });

      expect(pool.query).toHaveBeenCalled();
      expect(result).toEqual(mockUpdatedUser);
    });

    it('should update user name', async () => {
      const mockUpdatedUser = {
        id: 1,
        email: 'user@example.com',
        name: 'Updated Name',
        created_at: new Date()
      };

      pool.query.mockResolvedValue({ rows: [mockUpdatedUser] });

      const result = await userService.updateUser(1, { name: 'Updated Name' });

      expect(pool.query).toHaveBeenCalled();
      expect(result).toEqual(mockUpdatedUser);
    });

    it('should update user password with hashing', async () => {
      const mockUpdatedUser = {
        id: 1,
        email: 'user@example.com',
        name: 'Test User',
        created_at: new Date()
      };

      bcrypt.hash.mockResolvedValue('new_hashed_password');
      pool.query.mockResolvedValue({ rows: [mockUpdatedUser] });

      const result = await userService.updateUser(1, { password: 'NewPassword123!' });

      expect(bcrypt.hash).toHaveBeenCalledWith('NewPassword123!', 10);
      expect(result).toEqual(mockUpdatedUser);
    });

    it('should update multiple fields at once', async () => {
      const mockUpdatedUser = {
        id: 1,
        email: 'new@example.com',
        name: 'New Name',
        created_at: new Date()
      };

      pool.query.mockResolvedValue({ rows: [mockUpdatedUser] });

      const result = await userService.updateUser(1, {
        email: 'new@example.com',
        name: 'New Name'
      });

      expect(pool.query).toHaveBeenCalled();
      expect(result).toEqual(mockUpdatedUser);
    });

    it('should return undefined when updating non-existent user', async () => {
      pool.query.mockResolvedValue({ rows: [] });

      const result = await userService.updateUser(999, { email: 'test@example.com' });

      expect(result).toBeUndefined();
    });

    it('should handle database errors', async () => {
      pool.query.mockRejectedValue(new Error('Update failed'));

      await expect(userService.updateUser(1, { email: 'test@example.com' })).rejects.toThrow(
        'Update failed'
      );
    });
  });

  describe('deleteUser', () => {
    it('should delete user successfully', async () => {
      pool.query.mockResolvedValue({ rows: [] });

      await expect(userService.deleteUser(1)).resolves.toBeUndefined();
      expect(pool.query).toHaveBeenCalledWith('DELETE FROM users WHERE id = $1', [1]);
    });

    it('should handle database errors during deletion', async () => {
      pool.query.mockRejectedValue(new Error('Delete failed'));

      await expect(userService.deleteUser(1)).rejects.toThrow('Delete failed');
    });
  });
});
