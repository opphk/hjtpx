process.env.JWT_SECRET = 'test-secret-key';
process.env.NODE_ENV = 'test';

const bcrypt = require('bcryptjs');
const jwt = require('jsonwebtoken');

jest.mock('../../../config/database/db', () => ({
  query: jest.fn(),
  pool: {
    query: jest.fn()
  }
}));

const pool = require('../../../config/database/db');
const authService = require('../../services/authService');

describe('authService', () => {
  const TEST_JWT_SECRET = 'test-secret-key';

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('validatePassword', () => {
    test('should accept valid password with uppercase, lowercase, and number', () => {
      expect(authService.validatePassword('Password123')).toBe(true);
    });

    test('should reject password shorter than 8 characters', () => {
      expect(() => authService.validatePassword('Pass1')).toThrow(
        'Password must be at least 8 characters long'
      );
    });

    test('should reject password without uppercase letter', () => {
      expect(() => authService.validatePassword('password123')).toThrow(
        'Password must contain at least one uppercase letter, one lowercase letter, and one number'
      );
    });

    test('should reject password without lowercase letter', () => {
      expect(() => authService.validatePassword('PASSWORD123')).toThrow(
        'Password must contain at least one uppercase letter, one lowercase letter, and one number'
      );
    });

    test('should reject password without number', () => {
      expect(() => authService.validatePassword('PasswordABC')).toThrow(
        'Password must contain at least one uppercase letter, one lowercase letter, and one number'
      );
    });

    test('should reject null password', () => {
      expect(() => authService.validatePassword(null)).toThrow(
        'Password must be at least 8 characters long'
      );
    });

    test('should reject undefined password', () => {
      expect(() => authService.validatePassword(undefined)).toThrow(
        'Password must be at least 8 characters long'
      );
    });
  });

  describe('generateToken', () => {
    test('should generate a valid JWT token', () => {
      const user = { id: 1, email: 'test@example.com', role: 'user' };
      const token = authService.generateToken(user);

      expect(token).toBeDefined();
      expect(typeof token).toBe('string');

      const decoded = jwt.verify(token, 'test-secret-key');
      expect(decoded.id).toBe(1);
      expect(decoded.email).toBe('test@example.com');
      expect(decoded.role).toBe('user');
    });

    test('should include expiration in token', () => {
      const user = { id: 1, email: 'test@example.com', role: 'user' };
      const token = authService.generateToken(user);

      const decoded = jwt.decode(token);
      expect(decoded.exp).toBeDefined();
    });
  });

  describe('register', () => {
    test('should register a new user successfully', async () => {
      const mockUser = {
        id: 1,
        email: 'newuser@example.com',
        name: 'New User',
        role: 'user',
        created_at: new Date()
      };

      pool.query
        .mockResolvedValueOnce({ rows: [] })
        .mockResolvedValueOnce({ rows: [mockUser] });

      const result = await authService.register({
        email: 'newuser@example.com',
        name: 'New User',
        password: 'Password123'
      });

      expect(result.user).toBeDefined();
      expect(result.token).toBeDefined();
      expect(result.user.email).toBe('newuser@example.com');
    });

    test('should throw error if email already registered', async () => {
      pool.query.mockResolvedValueOnce({ rows: [{ id: 1 }] });

      await expect(
        authService.register({
          email: 'existing@example.com',
          name: 'Existing User',
          password: 'Password123'
        })
      ).rejects.toThrow('Email already registered');
    });

    test('should throw error for weak password', async () => {
      pool.query.mockResolvedValueOnce({ rows: [] });

      await expect(
        authService.register({
          email: 'newuser@example.com',
          name: 'New User',
          password: 'weak'
        })
      ).rejects.toThrow('Password must be at least 8 characters long');
    });

    test('should hash password before storing', async () => {
      const mockUser = {
        id: 1,
        email: 'newuser@example.com',
        name: 'New User',
        role: 'user',
        created_at: new Date()
      };

      pool.query
        .mockResolvedValueOnce({ rows: [] })
        .mockResolvedValueOnce({ rows: [mockUser] });

      await authService.register({
        email: 'newuser@example.com',
        name: 'New User',
        password: 'Password123'
      });

      const queryCall = pool.query.mock.calls[1];
      expect(queryCall[1][2]).toMatch(/^\$2[aby]?\$/);
    });

    test('should register user with admin role', async () => {
      const mockUser = {
        id: 1,
        email: 'admin@example.com',
        name: 'Admin User',
        role: 'admin',
        created_at: new Date()
      };

      pool.query
        .mockResolvedValueOnce({ rows: [] })
        .mockResolvedValueOnce({ rows: [mockUser] });

      const result = await authService.register({
        email: 'admin@example.com',
        name: 'Admin User',
        password: 'Password123',
        role: 'admin'
      });

      expect(result.user.role).toBe('admin');
    });
  });

  describe('login', () => {
    test('should login successfully with correct credentials', async () => {
      const hashedPassword = await bcrypt.hash('Password123', 10);
      const mockUser = {
        id: 1,
        email: 'user@example.com',
        name: 'Test User',
        password: hashedPassword,
        role: 'user'
      };

      pool.query.mockResolvedValueOnce({ rows: [mockUser] });

      const result = await authService.login({
        email: 'user@example.com',
        password: 'Password123'
      });

      expect(result.user).toBeDefined();
      expect(result.token).toBeDefined();
      expect(result.user.password).toBeUndefined();
    });

    test('should throw error for non-existent user', async () => {
      pool.query.mockResolvedValueOnce({ rows: [] });

      await expect(
        authService.login({
          email: 'nonexistent@example.com',
          password: 'Password123'
        })
      ).rejects.toThrow('Invalid credentials');
    });

    test('should throw error for wrong password', async () => {
      const hashedPassword = await bcrypt.hash('Password123', 10);
      const mockUser = {
        id: 1,
        email: 'user@example.com',
        name: 'Test User',
        password: hashedPassword,
        role: 'user'
      };

      pool.query.mockResolvedValueOnce({ rows: [mockUser] });

      await expect(
        authService.login({
          email: 'user@example.com',
          password: 'WrongPassword123'
        })
      ).rejects.toThrow('Invalid credentials');
    });
  });

  describe('forgotPassword', () => {
    test('should return message for existing user', async () => {
      pool.query
        .mockResolvedValueOnce({ rows: [{ id: 1 }] })
        .mockResolvedValueOnce({ rows: [] });

      const result = await authService.forgotPassword('user@example.com');

      expect(result.message).toBe('If email exists, reset link will be sent');
      expect(pool.query).toHaveBeenCalledTimes(2);
    });

    test('should return same message for non-existent user (security)', async () => {
      pool.query.mockResolvedValueOnce({ rows: [] });

      const result = await authService.forgotPassword('nonexistent@example.com');

      expect(result.message).toBe('If email exists, reset link will be sent');
      expect(pool.query).toHaveBeenCalledTimes(1);
    });

    test('should generate reset token in development mode', async () => {
      process.env.NODE_ENV = 'development';

      pool.query
        .mockResolvedValueOnce({ rows: [{ id: 1 }] })
        .mockResolvedValueOnce({ rows: [] });

      const result = await authService.forgotPassword('user@example.com');

      expect(result.resetToken).toBeDefined();
      expect(result.resetToken.length).toBe(64);
    });
  });

  describe('resetPassword', () => {
    test('should reset password with valid token', async () => {
      const hashedToken = await bcrypt.hash('valid-reset-token', 10);
      const hashedNewPassword = await bcrypt.hash('NewPassword123', 10);
      const mockUser = {
        id: 1,
        reset_token: hashedToken,
        reset_token_expires: new Date(Date.now() + 3600000)
      };

      pool.query
        .mockResolvedValueOnce({ rows: [mockUser] })
        .mockResolvedValueOnce({ rows: [] });

      const result = await authService.resetPassword({
        token: 'valid-reset-token',
        newPassword: 'NewPassword123'
      });

      expect(result.message).toBe('Password successfully reset');
    });

    test('should throw error for invalid token', async () => {
      pool.query.mockResolvedValueOnce({ rows: [] });

      await expect(
        authService.resetPassword({
          token: 'invalid-token',
          newPassword: 'NewPassword123'
        })
      ).rejects.toThrow('Invalid or expired reset token');
    });

    test('should throw error for weak new password', async () => {
      const hashedToken = await bcrypt.hash('valid-reset-token', 10);
      const mockUser = {
        id: 1,
        reset_token: hashedToken,
        reset_token_expires: new Date(Date.now() + 3600000)
      };

      pool.query.mockResolvedValueOnce({ rows: [mockUser] });

      await expect(
        authService.resetPassword({
          token: 'valid-reset-token',
          newPassword: 'weak'
        })
      ).rejects.toThrow('Password must be at least 8 characters long');
    });
  });

  describe('getCurrentUser', () => {
    test('should return user data for valid user ID', async () => {
      const mockUser = {
        id: 1,
        email: 'user@example.com',
        name: 'Test User',
        role: 'user',
        created_at: new Date()
      };

      pool.query.mockResolvedValueOnce({ rows: [mockUser] });

      const result = await authService.getCurrentUser(1);

      expect(result).toEqual(mockUser);
    });

    test('should throw error for non-existent user', async () => {
      pool.query.mockResolvedValueOnce({ rows: [] });

      await expect(authService.getCurrentUser(999)).rejects.toThrow('User not found');
    });
  });

  describe('logout', () => {
    test('should delete user sessions', async () => {
      pool.query.mockResolvedValueOnce({ rows: [] });

      const result = await authService.logout(1);

      expect(result.message).toBe('Logged out successfully');
      expect(pool.query).toHaveBeenCalledWith('DELETE FROM sessions WHERE user_id = $1', [1]);
    });
  });

  describe('validateSession', () => {
    test('should validate valid token', async () => {
      const user = { id: 1, email: 'user@example.com', role: 'user' };
      const token = jwt.sign(user, TEST_JWT_SECRET, { expiresIn: '1h' });

      const result = await authService.validateSession(token);

      expect(result.id).toBe(1);
      expect(result.email).toBe('user@example.com');
    });

    test('should throw error for invalid token', async () => {
      await expect(authService.validateSession('invalid-token')).rejects.toThrow(
        'Invalid session'
      );
    });

    test('should throw error for expired token', async () => {
      const user = { id: 1, email: 'user@example.com', role: 'user' };
      const token = jwt.sign(user, TEST_JWT_SECRET, { expiresIn: '-1h' });

      await expect(authService.validateSession(token)).rejects.toThrow('Invalid session');
    });
  });
});
