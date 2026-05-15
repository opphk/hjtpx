const { errorHandler, notFoundHandler, asyncHandler } = require('../../backend/middleware/errorHandler');
const { AppError, ValidationError, AuthenticationError } = require('../../backend/utils/appErrors');

describe('Error Handler Middleware', () => {
  let mockReq;
  let mockRes;
  let mockNext;

  beforeEach(() => {
    mockReq = {
      path: '/test',
      method: 'GET',
      ip: '127.0.0.1',
      user: { id: 'user123' },
      requestId: 'req-123'
    };

    mockRes = {
      status: jest.fn().mockReturnThis(),
      json: jest.fn().mockReturnThis()
    };

    mockNext = jest.fn();
  });

  test('should handle AppError correctly', () => {
    const error = new AppError('AUTH_001', 'Invalid credentials', 401);

    errorHandler(error, mockReq, mockRes, mockNext);

    expect(mockRes.status).toHaveBeenCalledWith(401);
    expect(mockRes.json).toHaveBeenCalledWith({
      success: false,
      error: {
        code: 'AUTH_001',
        message: 'Invalid credentials',
        details: null
      }
    });
  });

  test('should handle ValidationError correctly', () => {
    const error = new ValidationError('Validation failed', { field: 'email' });

    errorHandler(error, mockReq, mockRes, mockNext);

    expect(mockRes.status).toHaveBeenCalledWith(400);
    expect(mockRes.json).toHaveBeenCalledWith({
      success: false,
      error: {
        code: 'VAL_001',
        message: 'Validation failed',
        details: { field: 'email' }
      }
    });
  });

  test('should handle unknown errors with generic message in production', () => {
    const originalEnv = process.env.NODE_ENV;
    process.env.NODE_ENV = 'production';

    const error = new Error('Detailed error message');

    errorHandler(error, mockReq, mockRes, mockNext);

    expect(mockRes.status).toHaveBeenCalledWith(500);
    expect(mockRes.json).toHaveBeenCalledWith({
      success: false,
      error: {
        code: 'SRV_001',
        message: 'Internal server error'
      }
    });

    process.env.NODE_ENV = originalEnv;
  });

  test('should handle JWT errors', () => {
    const error = { name: 'JsonWebTokenError', message: 'Invalid token' };

    errorHandler(error, mockReq, mockRes, mockNext);

    expect(mockRes.status).toHaveBeenCalledWith(401);
    expect(mockRes.json).toHaveBeenCalledWith({
      success: false,
      error: {
        code: 'AUTH_003',
        message: 'Invalid token',
        details: null
      }
    });
  });

  test('should handle TokenExpiredError', () => {
    const error = { name: 'TokenExpiredError', message: 'Token expired' };

    errorHandler(error, mockReq, mockRes, mockNext);

    expect(mockRes.status).toHaveBeenCalledWith(401);
    expect(mockRes.json).toHaveBeenCalledWith({
      success: false,
      error: {
        code: 'AUTH_002',
        message: 'Token expired',
        details: null
      }
    });
  });
});

describe('Not Found Handler', () => {
  test('should return 404 with route info', () => {
    const mockReq = { originalUrl: '/unknown-route' };
    const mockRes = {
      status: jest.fn().mockReturnThis(),
      json: jest.fn().mockReturnThis()
    };
    const mockNext = jest.fn();

    notFoundHandler(mockReq, mockRes, mockNext);

    expect(mockRes.status).toHaveBeenCalledWith(404);
    expect(mockRes.json).toHaveBeenCalledWith({
      success: false,
      error: {
        code: 'DB_004',
        message: 'Route /unknown-route not found',
        details: null,
        statusCode: 404
      }
    });
  });
});

describe('Async Handler', () => {
  test('should pass successful result', async () => {
    const mockReq = {};
    const mockRes = { json: jest.fn() };
    const mockNext = jest.fn();
    const handler = asyncHandler(async (req, res) => {
      res.json({ success: true });
    });

    await handler(mockReq, mockRes, mockNext);

    expect(mockRes.json).toHaveBeenCalledWith({ success: true });
    expect(mockNext).not.toHaveBeenCalled();
  });

  test('should catch and forward errors', async () => {
    const mockReq = {};
    const mockRes = {};
    const mockNext = jest.fn();
    const testError = new Error('Test error');
    const handler = asyncHandler(async (req, res) => {
      throw testError;
    });

    await handler(mockReq, mockRes, mockNext);

    expect(mockNext).toHaveBeenCalledWith(testError);
  });
});
