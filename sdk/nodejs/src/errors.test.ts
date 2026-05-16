import {
  CaptchaError,
  ValidationError,
  AuthenticationError,
  NotFoundError,
  RateLimitError,
  ServerError,
  NetworkError,
} from './errors';

describe('Errors', () => {
  describe('CaptchaError', () => {
    it('should create an error with the correct properties', () => {
      const error = new CaptchaError('Test message', 'TEST_ERROR', 400, true);
      expect(error.message).toBe('Test message');
      expect(error.code).toBe('TEST_ERROR');
      expect(error.statusCode).toBe(400);
      expect(error.retryable).toBe(true);
      expect(error.name).toBe('CaptchaError');
    });

    it('should use default values for optional parameters', () => {
      const error = new CaptchaError('Test message');
      expect(error.code).toBe('UNKNOWN_ERROR');
      expect(error.statusCode).toBeUndefined();
      expect(error.retryable).toBe(false);
    });
  });

  describe('ValidationError', () => {
    it('should create a validation error with correct properties', () => {
      const error = new ValidationError('Invalid input');
      expect(error.message).toBe('Invalid input');
      expect(error.code).toBe('VALIDATION_ERROR');
      expect(error.statusCode).toBe(400);
      expect(error.retryable).toBe(false);
      expect(error.name).toBe('ValidationError');
    });
  });

  describe('AuthenticationError', () => {
    it('should create an authentication error with correct properties', () => {
      const error = new AuthenticationError('Invalid credentials');
      expect(error.message).toBe('Invalid credentials');
      expect(error.code).toBe('AUTHENTICATION_ERROR');
      expect(error.statusCode).toBe(401);
      expect(error.retryable).toBe(false);
      expect(error.name).toBe('AuthenticationError');
    });

    it('should use default message if not provided', () => {
      const error = new AuthenticationError();
      expect(error.message).toBe('Authentication failed');
    });
  });

  describe('NotFoundError', () => {
    it('should create a not found error with correct properties', () => {
      const error = new NotFoundError('Resource not found');
      expect(error.message).toBe('Resource not found');
      expect(error.code).toBe('NOT_FOUND');
      expect(error.statusCode).toBe(404);
      expect(error.retryable).toBe(false);
      expect(error.name).toBe('NotFoundError');
    });

    it('should use default message if not provided', () => {
      const error = new NotFoundError();
      expect(error.message).toBe('Resource not found');
    });
  });

  describe('RateLimitError', () => {
    it('should create a rate limit error with correct properties', () => {
      const error = new RateLimitError('Too many requests', 60);
      expect(error.message).toBe('Too many requests');
      expect(error.code).toBe('RATE_LIMIT_ERROR');
      expect(error.statusCode).toBe(429);
      expect(error.retryable).toBe(true);
      expect(error.retryAfter).toBe(60);
      expect(error.name).toBe('RateLimitError');
    });

    it('should use default message if not provided', () => {
      const error = new RateLimitError();
      expect(error.message).toBe('Rate limit exceeded');
    });
  });

  describe('ServerError', () => {
    it('should create a server error with correct properties', () => {
      const error = new ServerError('Internal server error', 503);
      expect(error.message).toBe('Internal server error');
      expect(error.code).toBe('SERVER_ERROR');
      expect(error.statusCode).toBe(503);
      expect(error.retryable).toBe(true);
      expect(error.name).toBe('ServerError');
    });

    it('should use default values if not provided', () => {
      const error = new ServerError();
      expect(error.message).toBe('Server error');
      expect(error.statusCode).toBe(500);
    });
  });

  describe('NetworkError', () => {
    it('should create a network error with correct properties', () => {
      const cause = new Error('Connection refused');
      const error = new NetworkError('Network issue', cause);
      expect(error.message).toBe('Network issue');
      expect(error.code).toBe('NETWORK_ERROR');
      expect(error.retryable).toBe(true);
      expect(error.cause).toBe(cause);
      expect(error.name).toBe('NetworkError');
    });

    it('should use default message if not provided', () => {
      const error = new NetworkError();
      expect(error.message).toBe('Network error');
    });
  });
});
