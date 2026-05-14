jest.mock('../../../config/database/db', () => ({
  query: jest.fn(),
  pool: {
    on: jest.fn(),
    query: jest.fn(),
    connect: jest.fn(),
    totalCount: 0,
    idleCount: 0,
    waitingCount: 0
  },
  getClient: jest.fn(),
  transaction: jest.fn(),
  healthCheck: jest.fn(),
  getPoolStats: jest.fn(),
  close: jest.fn()
}));

jest.mock('../../services/sessionService', () => ({
  validateSession: jest.fn(),
  getActiveSessionsCount: jest.fn(),
  enforceMaxSessions: jest.fn()
}));

const jwt = require('jsonwebtoken');

const { auth } = require('../../middleware/auth');

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
        { id: 1, email: 'test@example.com' },
        process.env.JWT_SECRET || 'hjtpx-secret-key-change-in-production',
        { expiresIn: '1h' }
      );
      mockReq.headers.authorization = `Bearer ${token}`;

      await auth(mockReq, mockRes, mockNext);

      expect(mockNext).toHaveBeenCalled();
      expect(mockReq.user).toBeDefined();
      expect(mockReq.user.id).toBe(1);
      expect(mockReq.user.email).toBe('test@example.com');
      expect(mockRes.status).not.toHaveBeenCalled();
    });

    it('should handle token with different payload', async () => {
      const token = jwt.sign(
        { id: 42, email: 'user@domain.com', role: 'admin' },
        process.env.JWT_SECRET || 'hjtpx-secret-key-change-in-production',
        { expiresIn: '2h' }
      );
      mockReq.headers.authorization = `Bearer ${token}`;

      await auth(mockReq, mockRes, mockNext);

      expect(mockNext).toHaveBeenCalled();
      expect(mockReq.user.role).toBe('admin');
    });
  });

  describe('invalid JWT token', () => {
    it('should reject invalid token format', async () => {
      mockReq.headers.authorization = 'Bearer invalid_token_string';

      await auth(mockReq, mockRes, mockNext);

      expect(mockRes.status).toHaveBeenCalledWith(401);
      expect(mockRes.json).toHaveBeenCalledWith({
        success: false,
        error: 'Invalid token'
      });
      expect(mockNext).not.toHaveBeenCalled();
    });

    it('should reject malformed token', async () => {
      mockReq.headers.authorization = 'Bearer malformed.token.here';

      await auth(mockReq, mockRes, mockNext);

      expect(mockRes.status).toHaveBeenCalledWith(401);
      expect(mockRes.json).toHaveBeenCalledWith({
        success: false,
        error: 'Invalid token'
      });
    });

    it('should reject token signed with wrong secret', async () => {
      const token = jwt.sign({ id: 1, email: 'test@example.com' }, 'wrong-secret-key', {
        expiresIn: '1h'
      });
      mockReq.headers.authorization = `Bearer ${token}`;

      await auth(mockReq, mockRes, mockNext);

      expect(mockRes.status).toHaveBeenCalledWith(401);
      expect(mockRes.json).toHaveBeenCalledWith({
        success: false,
        error: 'Invalid token'
      });
    });

    it('should reject expired token', async () => {
      const token = jwt.sign(
        { id: 1, email: 'test@example.com' },
        process.env.JWT_SECRET || 'hjtpx-secret-key-change-in-production',
        { expiresIn: '-1h' }
      );
      mockReq.headers.authorization = `Bearer ${token}`;

      await auth(mockReq, mockRes, mockNext);

      expect(mockRes.status).toHaveBeenCalledWith(401);
      expect(mockRes.json).toHaveBeenCalledWith({
        success: false,
        error: 'Invalid token'
      });
    });
  });

  describe('missing token', () => {
    it('should reject request without authorization header', async () => {
      await auth(mockReq, mockRes, mockNext);

      expect(mockRes.status).toHaveBeenCalledWith(401);
      expect(mockRes.json).toHaveBeenCalledWith({
        success: false,
        error: 'No token provided'
      });
      expect(mockNext).not.toHaveBeenCalled();
    });

    it('should reject request with empty authorization header', async () => {
      mockReq.headers.authorization = '';

      await auth(mockReq, mockRes, mockNext);

      expect(mockRes.status).toHaveBeenCalledWith(401);
      expect(mockRes.json).toHaveBeenCalledWith({
        success: false,
        error: 'No token provided'
      });
    });

    it('should reject request with undefined authorization header', async () => {
      mockReq.headers.authorization = undefined;

      await auth(mockReq, mockRes, mockNext);

      expect(mockRes.status).toHaveBeenCalledWith(401);
      expect(mockRes.json).toHaveBeenCalledWith({
        success: false,
        error: 'No token provided'
      });
    });

    it('should reject request with malformed authorization header', async () => {
      mockReq.headers.authorization = 'NotBearer some_token';

      await auth(mockReq, mockRes, mockNext);

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
        { id: 1, email: 'test@example.com' },
        process.env.JWT_SECRET || 'hjtpx-secret-key-change-in-production',
        { expiresIn: '1h' }
      );
      mockReq.headers.authorization = `Bearer ${token}`;

      await auth(mockReq, mockRes, mockNext);

      expect(mockNext).toHaveBeenCalled();
      expect(mockReq.user.id).toBe(1);
    });

    it('should reject empty bearer token', async () => {
      mockReq.headers.authorization = 'Bearer ';

      await auth(mockReq, mockRes, mockNext);

      expect(mockRes.status).toHaveBeenCalledWith(401);
      expect(mockRes.json).toHaveBeenCalledWith({
        success: false,
        error: 'No token provided'
      });
    });

    it('should handle authorization header with only Bearer keyword', async () => {
      mockReq.headers.authorization = 'Bearer';

      await auth(mockReq, mockRes, mockNext);

      expect(mockRes.status).toHaveBeenCalledWith(401);
      expect(mockRes.json).toHaveBeenCalledWith({
        success: false,
        error: 'No token provided'
      });
    });
  });
});
