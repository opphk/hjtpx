const mockPool = {
  query: jest.fn(),
  pool: {
    on: jest.fn(),
    query: jest.fn(),
    connect: jest.fn(),
    totalCount: 0,
    idleCount: 0,
    waitingCount: 0
  }
};

const mockBcrypt = {
  hash: jest.fn().mockResolvedValue('hashedPassword123'),
  compare: jest.fn()
};

const mockJwt = {
  sign: jest.fn().mockReturnValue('mocked-token'),
  verify: jest.fn()
};

jest.mock('../../../../src/config/database/db', () => mockPool);
jest.mock('bcrypt', () => mockBcrypt);
jest.mock('jsonwebtoken', () => mockJwt);

const jwt = require('jsonwebtoken');

const { auth: authMiddleware } = require('../../middleware/auth');
const { checkRole, ROLES } = require('../../middleware/roleCheck');
const authService = require('../../services/authService');

describe('Auth Service', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Password Validation', () => {
    test('should reject password shorter than 8 characters', () => {
      expect(() => authService.validatePassword('Short1!')).toThrow(
        'Password must be at least 8 characters long'
      );
    });

    test('should reject password without required characters', () => {
      expect(() => authService.validatePassword('lowercase1!')).toThrow(
        'Password must contain at least one uppercase letter'
      );
      expect(() => authService.validatePassword('UPPERCASE1!')).toThrow(
        'Password must contain at least one uppercase letter'
      );
      expect(() => authService.validatePassword('NoNumbers!')).toThrow(
        'Password must contain at least one uppercase letter'
      );
      expect(() => authService.validatePassword('NoSpecial1')).toThrow(
        'Password must contain at least one uppercase letter'
      );
    });

    test('should accept valid password', () => {
      expect(authService.validatePassword('ValidPass1!')).toBe(true);
    });
  });

  describe('Register', () => {
    test('should register new user successfully', async () => {
      mockPool.query.mockResolvedValueOnce({ rows: [] }).mockResolvedValueOnce({
        rows: [
          {
            id: '123e4567-e89b-12d3-a456-426614174000',
            email: 'test@example.com',
            name: 'Test User',
            role: 'user',
            created_at: new Date()
          }
        ]
      });

      const result = await authService.register({
        email: 'test@example.com',
        name: 'Test User',
        password: 'ValidPass1!'
      });

      expect(result).toHaveProperty('user');
      expect(result).toHaveProperty('token');
      expect(result.user.email).toBe('test@example.com');
      expect(mockBcrypt.hash).toHaveBeenCalledWith('ValidPass1!', 10);
    });

    test('should throw error for already registered email', async () => {
      mockPool.query.mockResolvedValueOnce({
        rows: [{ id: 'existing-user-id' }]
      });

      await expect(
        authService.register({
          email: 'existing@example.com',
          name: 'Test User',
          password: 'ValidPass1!'
        })
      ).rejects.toThrow('Email already registered');
    });

    test('should reject weak password during registration', async () => {
      mockPool.query.mockResolvedValueOnce({ rows: [] });

      await expect(
        authService.register({
          email: 'test@example.com',
          name: 'Test User',
          password: 'weak'
        })
      ).rejects.toThrow('Password must be at least 8 characters long');
    });

    test('should register user with custom role', async () => {
      mockPool.query.mockResolvedValueOnce({ rows: [] }).mockResolvedValueOnce({
        rows: [
          {
            id: '123e4567-e89b-12d3-a456-426614174000',
            email: 'admin@example.com',
            name: 'Admin User',
            role: 'admin',
            created_at: new Date()
          }
        ]
      });

      const result = await authService.register({
        email: 'admin@example.com',
        name: 'Admin User',
        password: 'ValidPass1!',
        role: 'admin'
      });

      expect(result.user.role).toBe('admin');
    });
  });

  describe('Login', () => {
    test('should login successfully with valid credentials', async () => {
      mockPool.query.mockResolvedValueOnce({
        rows: [
          {
            id: '123e4567-e89b-12d3-a456-426614174000',
            email: 'test@example.com',
            name: 'Test User',
            password: 'hashedPassword123',
            role: 'user'
          }
        ]
      });
      mockBcrypt.compare.mockResolvedValueOnce(true);

      const result = await authService.login({
        email: 'test@example.com',
        password: 'ValidPass1!'
      });

      expect(result).toHaveProperty('user');
      expect(result).toHaveProperty('token');
      expect(result.user.email).toBe('test@example.com');
      expect(result.user).not.toHaveProperty('password');
    });

    test('should throw error for non-existent user', async () => {
      mockPool.query.mockResolvedValueOnce({ rows: [] });

      await expect(
        authService.login({
          email: 'nonexistent@example.com',
          password: 'ValidPass1!'
        })
      ).rejects.toThrow('Invalid credentials');
    });

    test('should throw error for invalid password', async () => {
      mockPool.query.mockResolvedValueOnce({
        rows: [
          {
            id: '123e4567-e89b-12d3-a456-426614174000',
            email: 'test@example.com',
            name: 'Test User',
            password: 'hashedPassword123',
            role: 'user'
          }
        ]
      });
      mockBcrypt.compare.mockResolvedValueOnce(false);

      await expect(
        authService.login({
          email: 'test@example.com',
          password: 'wrongPassword!'
        })
      ).rejects.toThrow('Invalid credentials');
    });
  });

  describe('JWT Token Generation', () => {
    test('should generate valid JWT token', () => {
      const user = {
        id: '123e4567-e89b-12d3-a456-426614174000',
        email: 'test@example.com',
        role: 'user'
      };

      const token = authService.generateToken(user);

      expect(token).toBeDefined();
      expect(typeof token).toBe('string');
      expect(mockJwt.sign).toHaveBeenCalledWith(
        { id: user.id, email: user.email, role: user.role },
        expect.any(String),
        { expiresIn: '7d' }
      );
    });
  });

  describe('Session Validation', () => {
    test('should validate valid session', async () => {
      const user = {
        id: '123e4567-e89b-12d3-a456-426614174000',
        email: 'test@example.com',
        role: 'user'
      };

      const token = authService.generateToken(user);
      mockJwt.verify.mockResolvedValueOnce(user);

      const decoded = await authService.validateSession(token);

      expect(decoded.id).toBe(user.id);
      expect(mockJwt.verify).toHaveBeenCalledWith(token, expect.any(String));
    });

    test('should throw error for invalid session', async () => {
      mockJwt.verify.mockRejectedValueOnce(new Error('Invalid session'));

      await expect(authService.validateSession('invalid-token')).rejects.toThrow('Invalid session');
    });
  });

  describe('Forgot Password', () => {
    test('should return success message for existing email', async () => {
      mockPool.query
        .mockResolvedValueOnce({ rows: [{ id: '123' }] })
        .mockResolvedValueOnce({ rows: [] });

      const result = await authService.forgotPassword('test@example.com');

      expect(result.message).toBe('If email exists, reset link will be sent');
      expect(result).toHaveProperty('resetToken');
    });

    test('should return same message for non-existing email', async () => {
      mockPool.query.mockResolvedValueOnce({ rows: [] });

      const result = await authService.forgotPassword('nonexistent@example.com');

      expect(result.message).toBe('If email exists, reset link will be sent');
      expect(result).not.toHaveProperty('resetToken');
    });
  });

  describe('Reset Password', () => {
    test('should reset password with valid token', async () => {
      const resetToken = 'valid-reset-token';
      mockBcrypt.compare.mockResolvedValueOnce(true);
      mockPool.query
        .mockResolvedValueOnce({
          rows: [
            {
              id: '123e4567-e89b-12d3-a456-426614174000',
              reset_token: 'hashedToken',
              reset_token_expires: new Date(Date.now() + 3600000)
            }
          ]
        })
        .mockResolvedValueOnce({ rows: [] });

      const result = await authService.resetPassword({
        token: resetToken,
        newPassword: 'NewValid1!'
      });

      expect(result.message).toBe('Password successfully reset');
      expect(mockBcrypt.hash).toHaveBeenCalledWith('NewValid1!', 10);
    });

    test('should throw error for invalid token', async () => {
      mockBcrypt.compare.mockResolvedValue(false);
      mockPool.query.mockResolvedValueOnce({
        rows: [
          {
            id: '123',
            reset_token: 'hashedToken',
            reset_token_expires: new Date(Date.now() + 3600000)
          }
        ]
      });

      await expect(
        authService.resetPassword({
          token: 'invalid-token',
          newPassword: 'NewValid1!'
        })
      ).rejects.toThrow('Invalid or expired reset token');
    });

    test('should reject weak password during reset', async () => {
      mockBcrypt.compare.mockResolvedValue(true);
      mockPool.query.mockResolvedValueOnce({
        rows: [
          {
            id: '123',
            reset_token: 'hashedToken',
            reset_token_expires: new Date(Date.now() + 3600000)
          }
        ]
      });

      await expect(
        authService.resetPassword({
          token: 'valid-token',
          newPassword: 'weak'
        })
      ).rejects.toThrow('Password must be at least 8 characters long');
    });
  });

  describe('Get Current User', () => {
    test('should return user data for valid user id', async () => {
      mockPool.query.mockResolvedValueOnce({
        rows: [
          {
            id: '123e4567-e89b-12d3-a456-426614174000',
            email: 'test@example.com',
            name: 'Test User',
            role: 'user',
            created_at: new Date()
          }
        ]
      });

      const user = await authService.getCurrentUser('123e4567-e89b-12d3-a456-426614174000');

      expect(user.email).toBe('test@example.com');
      expect(user).not.toHaveProperty('password');
    });

    test('should throw error for non-existent user', async () => {
      mockPool.query.mockResolvedValueOnce({ rows: [] });

      await expect(authService.getCurrentUser('non-existent-id')).rejects.toThrow('User not found');
    });
  });

  describe('Logout', () => {
    test('should delete user sessions', async () => {
      mockPool.query.mockResolvedValueOnce({ rows: [] });

      const result = await authService.logout('123e4567-e89b-12d3-a456-426614174000');

      expect(result.message).toBe('Logged out successfully');
      expect(mockPool.query).toHaveBeenCalledWith('DELETE FROM sessions WHERE user_id = $1', [
        '123e4567-e89b-12d3-a456-426614174000'
      ]);
    });
  });
});

describe('Role Check Middleware', () => {
  let mockReq, mockRes, mockNext;

  beforeEach(() => {
    mockReq = {};
    mockRes = {
      status: jest.fn().mockReturnThis(),
      json: jest.fn()
    };
    mockNext = jest.fn();
  });

  test('should allow access for user with correct role', () => {
    mockReq.user = { id: '123', role: 'admin' };

    const middleware = checkRole('admin', 'moderator');
    middleware(mockReq, mockRes, mockNext);

    expect(mockNext).toHaveBeenCalled();
    expect(mockRes.status).not.toHaveBeenCalled();
  });

  test('should deny access for user with incorrect role', () => {
    mockReq.user = { id: '123', role: 'user' };

    const middleware = checkRole('admin');
    middleware(mockReq, mockRes, mockNext);

    expect(mockRes.status).toHaveBeenCalledWith(403);
    expect(mockRes.json).toHaveBeenCalledWith({
      success: false,
      error: 'Insufficient permissions'
    });
    expect(mockNext).not.toHaveBeenCalled();
  });

  test('should deny access for unauthenticated user', () => {
    mockReq.user = null;

    const middleware = checkRole('admin');
    middleware(mockReq, mockRes, mockNext);

    expect(mockRes.status).toHaveBeenCalledWith(401);
    expect(mockRes.json).toHaveBeenCalledWith({
      success: false,
      error: 'Authentication required'
    });
  });

  test('should have correct role constants', () => {
    expect(ROLES.ADMIN).toBe('admin');
    expect(ROLES.USER).toBe('user');
    expect(ROLES.MODERATOR).toBe('moderator');
  });
});

describe('Auth Middleware', () => {
  let mockReq, mockRes, mockNext;

  beforeEach(() => {
    mockReq = { headers: {} };
    mockRes = {
      status: jest.fn().mockReturnThis(),
      json: jest.fn()
    };
    mockNext = jest.fn();
    mockJwt.verify.mockReset();
  });

  test('should reject request without token', () => {
    authMiddleware(mockReq, mockRes, mockNext);

    expect(mockRes.status).toHaveBeenCalledWith(401);
    expect(mockRes.json).toHaveBeenCalledWith({
      success: false,
      error: 'No token provided'
    });
    expect(mockNext).not.toHaveBeenCalled();
  });

  test('should reject request with invalid token', async () => {
    mockReq.headers.authorization = 'Bearer invalid-token';
    mockJwt.verify.mockImplementation(() => {
      throw new Error('Invalid token');
    });

    await authMiddleware(mockReq, mockRes, mockNext);

    expect(mockRes.status).toHaveBeenCalledWith(401);
    expect(mockRes.json).toHaveBeenCalledWith({
      success: false,
      error: 'Invalid token'
    });
  });

  test('should allow request with valid token', async () => {
    const mockReqLocal = { headers: { authorization: 'Bearer valid-token' } };
    const decodedUser = { id: '123', email: 'test@example.com', role: 'user' };
    mockJwt.verify.mockImplementation((token, secret) => {
      if (token === 'valid-token') {
        return decodedUser;
      }
      throw new Error('Invalid token');
    });

    await authMiddleware(mockReqLocal, mockRes, mockNext);

    expect(mockReqLocal.user).toBeDefined();
    expect(mockReqLocal.user.id).toBe('123');
    expect(mockReqLocal.user.email).toBe('test@example.com');
    expect(mockNext).toHaveBeenCalled();
  });

  test('should reject malformed authorization header', async () => {
    mockReq.headers.authorization = 'InvalidFormat token';
    mockJwt.verify.mockImplementation(() => {
      throw new Error('Invalid token');
    });

    await authMiddleware(mockReq, mockRes, mockNext);

    expect(mockRes.status).toHaveBeenCalledWith(401);
  });
});
