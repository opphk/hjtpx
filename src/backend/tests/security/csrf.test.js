const CSRFTokenManager = require('../../../frontend/src/utils/csrf_token_manager');

describe('CSRF Protection Tests', () => {
  const mockRes = {
    cookie: jest.fn(),
    header: jest.fn()
  };

  const mockReq = {
    headers: {},
    cookies: {},
    body: {},
    method: 'POST'
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Token Generation', () => {
    test('should generate a random token', () => {
      const token = CSRFTokenManager.generateToken();
      expect(token).toBeDefined();
      expect(typeof token).toBe('string');
      expect(token.length).toBeGreaterThan(0);
    });

    test('should generate unique tokens', () => {
      const token1 = CSRFTokenManager.generateToken();
      const token2 = CSRFTokenManager.generateToken();
      expect(token1).not.toBe(token2);
    });

    test('should generate token with specified length', () => {
      const token = CSRFTokenManager.generateToken(64);
      expect(token.length).toBeGreaterThanOrEqual(64);
    });
  });

  describe('Cookie Token Setting', () => {
    test('should set CSRF token as HttpOnly cookie', () => {
      const token = 'test-token';
      CSRFTokenManager.setCookieToken(mockRes, token);

      expect(mockRes.cookie).toHaveBeenCalledWith(
        '_csrf',
        token,
        expect.objectContaining({
          httpOnly: true,
          secure: expect.any(Boolean),
          sameSite: expect.any(String)
        })
      );
    });

    test('should set cookie with custom options', () => {
      const token = 'test-token';
      const options = { maxAge: 3600000, path: '/api' };
      CSRFTokenManager.setCookieToken(mockRes, token, options);

      expect(mockRes.cookie).toHaveBeenCalledWith(
        '_csrf',
        token,
        expect.objectContaining({
          httpOnly: true,
          maxAge: 3600000,
          path: '/api'
        })
      );
    });
  });

  describe('Token Validation', () => {
    test('should validate matching tokens', () => {
      const token = CSRFTokenManager.generateToken();
      mockReq.headers['x-csrf-token'] = token;
      mockReq.cookies._csrf = token;

      const isValid = CSRFTokenManager.validateToken(mockReq);
      expect(isValid).toBe(true);
    });

    test('should reject mismatched tokens', () => {
      mockReq.headers['x-csrf-token'] = 'token-a';
      mockReq.cookies._csrf = 'token-b';

      const isValid = CSRFTokenManager.validateToken(mockReq);
      expect(isValid).toBe(false);
    });

    test('should reject requests without CSRF token', () => {
      mockReq.headers = {};
      mockReq.cookies = {};
      mockReq.body = {};

      const isValid = CSRFTokenManager.validateToken(mockReq);
      expect(isValid).toBe(false);
    });

    test('should accept token from request body', () => {
      const token = CSRFTokenManager.generateToken();
      mockReq.headers = {};
      mockReq.cookies = { _csrf: token };
      mockReq.body = { _csrf: token };

      const isValid = CSRFTokenManager.validateToken(mockReq);
      expect(isValid).toBe(true);
    });

    test('should reject empty tokens', () => {
      mockReq.headers['x-csrf-token'] = '';
      mockReq.cookies._csrf = '';

      const isValid = CSRFTokenManager.validateToken(mockReq);
      expect(isValid).toBe(false);
    });

    test('should handle GET requests (safe methods)', () => {
      const mockGetReq = { ...mockReq, method: 'GET' };
      const isValid = CSRFTokenManager.validateToken(mockGetReq);
      expect(isValid).toBe(true);
    });
  });
});
