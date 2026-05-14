const { auth } = require('../../../backend/middleware/auth');
const jwt = require('jsonwebtoken');

describe('Auth Middleware', () => {
  let mockReq;
  let mockRes;
  let mockNext;

  beforeEach(() => {
    mockReq = {
      headers: {}
    };
    mockRes = {
      status: jest.fn().mockReturnThis(),
      json: jest.fn().mockReturnThis()
    };
    mockNext = jest.fn();
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  describe('valid JWT token', () => {
    it('should pass valid token and call next', async () => {
      const token = jwt.sign(
        { userId: 1, email: 'test@example.com' },
        process.env.JWT_SECRET || 'test-secret-key',
        { expiresIn: '1h' }
      );
      mockReq.headers.authorization = `Bearer ${token}`;

      await auth(mockReq, mockRes, mockNext);

      expect(mockNext).toHaveBeenCalled();
      expect(mockReq.user).toBeDefined();
      expect(mockReq.user.userId).toBe(1);
      expect(mockReq.user.email).toBe('test@example.com');
      expect(mockRes.status).not.toHaveBeenCalled();
    });

    it('should handle token with different payload', async () => {
      const token = jwt.sign(
        { userId: 42, email: 'user@domain.com', role: 'admin' },
        process.env.JWT_SECRET || 'test-secret-key',
        { expiresIn: '2h' }
      );
      mockReq.headers.authorization = `Bearer ${token}`;

      await auth(mockReq, mockRes, mockNext);

      expect(mockNext).toHaveBeenCalled();
      expect(mockReq.user.role).toBe('admin');
    });
  });

  describe('invalid JWT token', () => {
    it('should reject invalid token format', () => {
      mockReq.headers.authorization = 'Bearer invalid_token_string';

      auth(mockReq, mockRes, mockNext);

      expect(mockRes.status).toHaveBeenCalledWith(401);
      expect(mockRes.json).toHaveBeenCalledWith({
        success: false,
        error: 'Invalid token'
      });
      expect(mockNext).not.toHaveBeenCalled();
    });

    it('should reject malformed token', () => {
      mockReq.headers.authorization = 'Bearer malformed.token.here';

      auth(mockReq, mockRes, mockNext);

      expect(mockRes.status).toHaveBeenCalledWith(401);
      expect(mockRes.json).toHaveBeenCalledWith({
        success: false,
        error: 'Invalid token'
      });
    });

    it('should reject token signed with wrong secret', () => {
      const token = jwt.sign(
        { userId: 1, email: 'test@example.com' },
        'wrong-secret-key',
        { expiresIn: '1h' }
      );
      mockReq.headers.authorization = `Bearer ${token}`;

      auth(mockReq, mockRes, mockNext);

      expect(mockRes.status).toHaveBeenCalledWith(401);
      expect(mockRes.json).toHaveBeenCalledWith({
        success: false,
        error: 'Invalid token'
      });
    });

    it('should reject expired token', () => {
      const token = jwt.sign(
        { userId: 1, email: 'test@example.com' },
        process.env.JWT_SECRET || 'test-secret-key',
        { expiresIn: '-1h' }
      );
      mockReq.headers.authorization = `Bearer ${token}`;

      auth(mockReq, mockRes, mockNext);

      expect(mockRes.status).toHaveBeenCalledWith(401);
      expect(mockRes.json).toHaveBeenCalledWith({
        success: false,
        error: 'Invalid token'
      });
    });
  });

  describe('missing token', () => {
    it('should reject request without authorization header', () => {
      auth(mockReq, mockRes, mockNext);

      expect(mockRes.status).toHaveBeenCalledWith(401);
      expect(mockRes.json).toHaveBeenCalledWith({
        success: false,
        error: 'No token provided'
      });
      expect(mockNext).not.toHaveBeenCalled();
    });

    it('should reject request with empty authorization header', () => {
      mockReq.headers.authorization = '';

      auth(mockReq, mockRes, mockNext);

      expect(mockRes.status).toHaveBeenCalledWith(401);
      expect(mockRes.json).toHaveBeenCalledWith({
        success: false,
        error: 'No token provided'
      });
    });

    it('should reject request with undefined authorization header', () => {
      mockReq.headers.authorization = undefined;

      auth(mockReq, mockRes, mockNext);

      expect(mockRes.status).toHaveBeenCalledWith(401);
      expect(mockRes.json).toHaveBeenCalledWith({
        success: false,
        error: 'No token provided'
      });
    });

    it('should reject request with malformed authorization header', () => {
      mockReq.headers.authorization = 'NotBearer some_token';

      auth(mockReq, mockRes, mockNext);

      expect(mockRes.status).toHaveBeenCalledWith(401);
      expect(mockRes.json).toHaveBeenCalledWith({
        success: false,
        error: 'Invalid token'
      });
      expect(mockNext).not.toHaveBeenCalled();
    });
  });

  describe('token extraction edge cases', () => {
    it('should handle token with multiple spaces', async () => {
      const token = jwt.sign(
        { userId: 1, email: 'test@example.com' },
        process.env.JWT_SECRET || 'test-secret-key',
        { expiresIn: '1h' }
      );
      mockReq.headers.authorization = `Bearer ${token}`;

      await auth(mockReq, mockRes, mockNext);

      expect(mockNext).toHaveBeenCalled();
      expect(mockReq.user.userId).toBe(1);
    });

    it('should reject empty bearer token', () => {
      mockReq.headers.authorization = 'Bearer ';

      auth(mockReq, mockRes, mockNext);

      expect(mockRes.status).toHaveBeenCalledWith(401);
      expect(mockRes.json).toHaveBeenCalledWith({
        success: false,
        error: 'No token provided'
      });
    });

    it('should handle authorization header with only Bearer keyword', () => {
      mockReq.headers.authorization = 'Bearer';

      auth(mockReq, mockRes, mockNext);

      expect(mockRes.status).toHaveBeenCalledWith(401);
      expect(mockRes.json).toHaveBeenCalledWith({
        success: false,
        error: 'No token provided'
      });
    });
  });
});
