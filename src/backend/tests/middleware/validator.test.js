const validator = require('../../middleware/validator');

describe('validator middleware', () => {
  let mockReq;
  let mockRes;
  let mockNext;

  beforeEach(() => {
    mockReq = {
      body: {},
      query: {},
      params: {}
    };
    mockRes = {
      status: jest.fn().mockReturnThis(),
      json: jest.fn().mockReturnThis()
    };
    mockNext = jest.fn();
  });

  describe('userSchema validation', () => {
    test('should pass validation with valid user data', () => {
      mockReq.body = {
        email: 'user@example.com',
        name: 'Test User',
        password: 'Password123'
      };

      const middleware = validator('userSchema', 'body');
      middleware(mockReq, mockRes, mockNext);

      expect(mockNext).toHaveBeenCalled();
      expect(mockRes.status).not.toHaveBeenCalled();
    });

    test('should reject invalid email', () => {
      mockReq.body = {
        email: 'invalid-email',
        name: 'Test User',
        password: 'Password123'
      };

      const middleware = validator('userSchema', 'body');
      middleware(mockReq, mockRes, mockNext);

      expect(mockNext).not.toHaveBeenCalled();
      expect(mockRes.status).toHaveBeenCalledWith(400);
      expect(mockRes.json).toHaveBeenCalledWith(
        expect.objectContaining({
          success: false,
          error: expect.objectContaining({
            code: 'VALIDATION_ERROR'
          })
        })
      );
    });

    test('should reject missing required fields', () => {
      mockReq.body = {
        email: 'user@example.com'
      };

      const middleware = validator('userSchema', 'body');
      middleware(mockReq, mockRes, mockNext);

      expect(mockNext).not.toHaveBeenCalled();
      expect(mockRes.status).toHaveBeenCalledWith(400);
    });

    test('should reject short name', () => {
      mockReq.body = {
        email: 'user@example.com',
        name: 'A',
        password: 'Password123'
      };

      const middleware = validator('userSchema', 'body');
      middleware(mockReq, mockRes, mockNext);

      expect(mockNext).not.toHaveBeenCalled();
      expect(mockRes.status).toHaveBeenCalledWith(400);
    });

    test('should reject weak password', () => {
      mockReq.body = {
        email: 'user@example.com',
        name: 'Test User',
        password: 'weak'
      };

      const middleware = validator('userSchema', 'body');
      middleware(mockReq, mockRes, mockNext);

      expect(mockNext).not.toHaveBeenCalled();
      expect(mockRes.status).toHaveBeenCalledWith(400);
    });

    test('should reject password without uppercase', () => {
      mockReq.body = {
        email: 'user@example.com',
        name: 'Test User',
        password: 'password123'
      };

      const middleware = validator('userSchema', 'body');
      middleware(mockReq, mockRes, mockNext);

      expect(mockNext).not.toHaveBeenCalled();
      expect(mockRes.status).toHaveBeenCalledWith(400);
    });

    test('should reject password without number', () => {
      mockReq.body = {
        email: 'user@example.com',
        name: 'Test User',
        password: 'PasswordABC'
      };

      const middleware = validator('userSchema', 'body');
      middleware(mockReq, mockRes, mockNext);

      expect(mockNext).not.toHaveBeenCalled();
      expect(mockRes.status).toHaveBeenCalledWith(400);
    });
  });

  describe('loginSchema validation', () => {
    test('should pass validation with valid login data', () => {
      mockReq.body = {
        email: 'user@example.com',
        password: 'Password123'
      };

      const middleware = validator('loginSchema', 'body');
      middleware(mockReq, mockRes, mockNext);

      expect(mockNext).toHaveBeenCalled();
      expect(mockRes.status).not.toHaveBeenCalled();
    });

    test('should reject invalid email format', () => {
      mockReq.body = {
        email: 'not-an-email',
        password: 'Password123'
      };

      const middleware = validator('loginSchema', 'body');
      middleware(mockReq, mockRes, mockNext);

      expect(mockNext).not.toHaveBeenCalled();
      expect(mockRes.status).toHaveBeenCalledWith(400);
    });

    test('should reject empty password', () => {
      mockReq.body = {
        email: 'user@example.com',
        password: ''
      };

      const middleware = validator('loginSchema', 'body');
      middleware(mockReq, mockRes, mockNext);

      expect(mockNext).not.toHaveBeenCalled();
      expect(mockRes.status).toHaveBeenCalledWith(400);
    });

    test('should reject missing email', () => {
      mockReq.body = {
        password: 'Password123'
      };

      const middleware = validator('loginSchema', 'body');
      middleware(mockReq, mockRes, mockNext);

      expect(mockNext).not.toHaveBeenCalled();
      expect(mockRes.status).toHaveBeenCalledWith(400);
    });
  });

  describe('updateUserSchema validation', () => {
    test('should pass validation with valid update data', () => {
      mockReq.body = {
        name: 'Updated Name'
      };

      const middleware = validator('updateUserSchema', 'body');
      middleware(mockReq, mockRes, mockNext);

      expect(mockNext).toHaveBeenCalled();
    });

    test('should reject empty update object', () => {
      mockReq.body = {};

      const middleware = validator('updateUserSchema', 'body');
      middleware(mockReq, mockRes, mockNext);

      expect(mockNext).not.toHaveBeenCalled();
      expect(mockRes.status).toHaveBeenCalledWith(400);
    });

    test('should validate email format in update', () => {
      mockReq.body = {
        email: 'invalid-email'
      };

      const middleware = validator('updateUserSchema', 'body');
      middleware(mockReq, mockRes, mockNext);

      expect(mockNext).not.toHaveBeenCalled();
      expect(mockRes.status).toHaveBeenCalledWith(400);
    });

    test('should validate password strength in update', () => {
      mockReq.body = {
        password: 'weak'
      };

      const middleware = validator('updateUserSchema', 'body');
      middleware(mockReq, mockRes, mockNext);

      expect(mockNext).not.toHaveBeenCalled();
      expect(mockRes.status).toHaveBeenCalledWith(400);
    });
  });

  describe('forgotPasswordSchema validation', () => {
    test('should pass validation with valid email', () => {
      mockReq.body = {
        email: 'user@example.com'
      };

      const middleware = validator('forgotPasswordSchema', 'body');
      middleware(mockReq, mockRes, mockNext);

      expect(mockNext).toHaveBeenCalled();
    });

    test('should reject invalid email', () => {
      mockReq.body = {
        email: 'not-an-email'
      };

      const middleware = validator('forgotPasswordSchema', 'body');
      middleware(mockReq, mockRes, mockNext);

      expect(mockNext).not.toHaveBeenCalled();
      expect(mockRes.status).toHaveBeenCalledWith(400);
    });

    test('should reject missing email', () => {
      mockReq.body = {};

      const middleware = validator('forgotPasswordSchema', 'body');
      middleware(mockReq, mockRes, mockNext);

      expect(mockNext).not.toHaveBeenCalled();
      expect(mockRes.status).toHaveBeenCalledWith(400);
    });
  });

  describe('resetPasswordSchema validation', () => {
    test('should pass validation with valid token and password', () => {
      mockReq.body = {
        token: 'a'.repeat(64),
        newPassword: 'NewPassword123!'
      };

      const middleware = validator('resetPasswordSchema', 'body');
      middleware(mockReq, mockRes, mockNext);

      expect(mockNext).toHaveBeenCalled();
    });

    test('should reject short token', () => {
      mockReq.body = {
        token: 'short',
        newPassword: 'NewPassword123!'
      };

      const middleware = validator('resetPasswordSchema', 'body');
      middleware(mockReq, mockRes, mockNext);

      expect(mockNext).not.toHaveBeenCalled();
      expect(mockRes.status).toHaveBeenCalledWith(400);
    });

    test('should reject weak password', () => {
      mockReq.body = {
        token: 'a'.repeat(64),
        newPassword: 'weak'
      };

      const middleware = validator('resetPasswordSchema', 'body');
      middleware(mockReq, mockRes, mockNext);

      expect(mockNext).not.toHaveBeenCalled();
      expect(mockRes.status).toHaveBeenCalledWith(400);
    });

    test('should reject password without special character', () => {
      mockReq.body = {
        token: 'a'.repeat(64),
        newPassword: 'NewPassword123'
      };

      const middleware = validator('resetPasswordSchema', 'body');
      middleware(mockReq, mockRes, mockNext);

      expect(mockNext).not.toHaveBeenCalled();
      expect(mockRes.status).toHaveBeenCalledWith(400);
    });
  });

  describe('query parameter validation', () => {
    test('should validate query parameters', () => {
      mockReq.query = {
        email: 'user@example.com'
      };

      const middleware = validator('forgotPasswordSchema', 'query');
      middleware(mockReq, mockRes, mockNext);

      expect(mockNext).toHaveBeenCalled();
    });

    test('should return 400 for invalid query params', () => {
      mockReq.query = {
        email: 'invalid-email'
      };

      const middleware = validator('forgotPasswordSchema', 'query');
      middleware(mockReq, mockRes, mockNext);

      expect(mockNext).not.toHaveBeenCalled();
      expect(mockRes.status).toHaveBeenCalledWith(400);
    });
  });

  describe('params validation', () => {
    test('should validate request params', () => {
      mockReq.params = {
        id: '123'
      };

      const middleware = validator('loginSchema', 'params');
      middleware(mockReq, mockRes, mockNext);

      expect(mockRes.status).toHaveBeenCalledWith(400);
    });
  });

  describe('schema not found', () => {
    test('should return 500 for non-existent schema', () => {
      mockReq.body = {
        email: 'user@example.com'
      };

      const middleware = validator('nonExistentSchema', 'body');
      middleware(mockReq, mockRes, mockNext);

      expect(mockNext).not.toHaveBeenCalled();
      expect(mockRes.status).toHaveBeenCalledWith(500);
      expect(mockRes.json).toHaveBeenCalledWith(
        expect.objectContaining({
          success: false,
          error: expect.objectContaining({
            code: 'VALIDATION_SCHEMA_NOT_FOUND'
          })
        })
      );
    });
  });

  describe('data sanitization', () => {
    test('should strip unknown fields', () => {
      mockReq.body = {
        email: 'user@example.com',
        name: 'Test User',
        password: 'Password123',
        unknownField: 'should be removed'
      };

      const middleware = validator('userSchema', 'body');
      middleware(mockReq, mockRes, mockNext);

      expect(mockNext).toHaveBeenCalled();
      expect(mockReq.body.unknownField).toBeUndefined();
    });

    test('should transform and normalize data', () => {
      mockReq.body = {
        email: 'USER@EXAMPLE.COM',
        name: '  Test User  ',
        password: 'Password123'
      };

      const middleware = validator('userSchema', 'body');
      middleware(mockReq, mockRes, mockNext);

      expect(mockNext).toHaveBeenCalled();
    });
  });

  describe('error response format', () => {
    test('should include field, message, and type in error details', () => {
      mockReq.body = {
        email: 'invalid',
        name: 'A',
        password: 'weak'
      };

      const middleware = validator('userSchema', 'body');
      middleware(mockReq, mockRes, mockNext);

      expect(mockRes.json).toHaveBeenCalledWith(
        expect.objectContaining({
          success: false,
          error: expect.objectContaining({
            code: 'VALIDATION_ERROR',
            message: 'Validation failed',
            details: expect.arrayContaining([
              expect.objectContaining({
                field: expect.any(String),
                message: expect.any(String),
                type: expect.any(String)
              })
            ])
          })
        })
      );
    });

    test('should report all errors (not just first)', () => {
      mockReq.body = {
        email: 'invalid',
        name: 'A',
        password: 'weak'
      };

      const middleware = validator('userSchema', 'body');
      middleware(mockReq, mockRes, mockNext);

      const response = mockRes.json.mock.calls[0][0];
      expect(response.error.details.length).toBeGreaterThan(1);
    });
  });
});
