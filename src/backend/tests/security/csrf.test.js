const { 
  createCsrfToken, 
  generateSignedToken, 
  verifyCsrfToken,
  csrfTokenGenerator,
  doubleSubmitCookie 
} = require('../../middleware/csrfProtection');

describe('CSRF Protection', () => {
  describe('createCsrfToken', () => {
    test('should generate a random token', () => {
      const token = createCsrfToken();
      expect(token).toBeDefined();
      expect(typeof token).toBe('string');
      expect(token.length).toBe(64);
    });

    test('should generate unique tokens', () => {
      const token1 = createCsrfToken();
      const token2 = createCsrfToken();
      expect(token1).not.toBe(token2);
    });
  });

  describe('generateSignedToken', () => {
    test('should generate a signed token', () => {
      const token = generateSignedToken('test-secret');
      expect(token).toBeDefined();
      expect(token.includes('.')).toBe(true);
      const [randomPart, signature] = token.split('.');
      expect(randomPart.length).toBe(64);
      expect(signature.length).toBe(64);
    });

    test('should generate different signatures for different secrets', () => {
      const token1 = generateSignedToken('secret1');
      const token2 = generateSignedToken('secret2');
      expect(token1).not.toBe(token2);
    });
  });

  describe('verifyCsrfToken', () => {
    test('should verify a valid token', () => {
      const secret = 'test-secret';
      const token = generateSignedToken(secret);
      expect(verifyCsrfToken(token, secret)).toBe(true);
    });

    test('should reject token with wrong secret', () => {
      const token = generateSignedToken('secret1');
      expect(verifyCsrfToken(token, 'secret2')).toBe(false);
    });

    test('should reject malformed token', () => {
      expect(verifyCsrfToken('invalid-token', 'secret')).toBe(false);
    });
  });

  describe('csrfTokenGenerator middleware', () => {
    test('should generate and set CSRF token', () => {
      const req = { csrfToken: undefined };
      const res = {
        cookie: jest.fn()
      };
      const next = jest.fn();

      const middleware = csrfTokenGenerator();
      middleware(req, res, next);

      expect(req.csrfToken).toBeDefined();
      expect(res.cookie).toHaveBeenCalledWith(
        'csrf-token',
        expect.any(String),
        expect.objectContaining({
          httpOnly: false,
          sameSite: 'strict'
        })
      );
      expect(next).toHaveBeenCalled();
    });

    test('should reuse existing token', () => {
      const existingToken = 'existing-token';
      const req = { csrfToken: existingToken };
      const res = { cookie: jest.fn() };
      const next = jest.fn();

      const middleware = csrfTokenGenerator();
      middleware(req, res, next);

      expect(req.csrfToken).toBe(existingToken);
      expect(res.cookie).not.toHaveBeenCalled();
      expect(next).toHaveBeenCalled();
    });
  });

  describe('doubleSubmitCookie middleware', () => {
    test('should validate matching tokens', () => {
      const token = createCsrfToken();
      const req = {
        method: 'POST',
        cookies: { csrf_token: token },
        headers: {},
        body: { _csrf: token }
      };
      const res = {
        status: jest.fn().mockReturnThis(),
        json: jest.fn()
      };
      const next = jest.fn();

      const middleware = doubleSubmitCookie();
      middleware(req, res, next);

      expect(next).toHaveBeenCalled();
      expect(req.csrfValid).toBe(true);
    });

    test('should reject mismatched tokens', () => {
      const req = {
        method: 'POST',
        cookies: { csrf_token: 'token1' },
        headers: {},
        body: { _csrf: 'token2' }
      };
      const res = {
        status: jest.fn().mockReturnThis(),
        json: jest.fn()
      };
      const next = jest.fn();

      const middleware = doubleSubmitCookie();
      middleware(req, res, next);

      expect(res.status).toHaveBeenCalledWith(403);
      expect(res.json).toHaveBeenCalledWith({
        success: false,
        error: 'CSRF token mismatch'
      });
      expect(next).not.toHaveBeenCalled();
    });

    test('should reject missing tokens', () => {
      const req = {
        method: 'POST',
        cookies: {},
        headers: {},
        body: {}
      };
      const res = {
        status: jest.fn().mockReturnThis(),
        json: jest.fn()
      };
      const next = jest.fn();

      const middleware = doubleSubmitCookie();
      middleware(req, res, next);

      expect(res.status).toHaveBeenCalledWith(403);
      expect(res.json).toHaveBeenCalledWith({
        success: false,
        error: 'CSRF token missing'
      });
    });

    test('should skip validation for GET requests', () => {
      const req = {
        method: 'GET',
        cookies: {},
        headers: {}
      };
      const res = {
        cookie: jest.fn()
      };
      const next = jest.fn();

      const middleware = doubleSubmitCookie();
      middleware(req, res, next);

      expect(res.cookie).toHaveBeenCalled();
      expect(next).toHaveBeenCalled();
    });
  });
});
