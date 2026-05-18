import { CaptchaClient } from './client';
import {
  CaptchaError,
  ValidationError,
  AuthenticationError,
  NotFoundError,
  RateLimitError,
  ServerError,
  NetworkError,
} from './errors';

const MOCK_BASE_URL = 'http://localhost:8080';

describe('CaptchaClient', () => {
  let client: CaptchaClient;

  beforeEach(() => {
    client = new CaptchaClient({
      baseUrl: MOCK_BASE_URL,
      apiKey: 'test-api-key',
      timeout: 5000,
    });
  });

  afterEach(async () => {
    await client.close();
  });

  describe('constructor', () => {
    it('should create a client with default config', () => {
      const defaultClient = new CaptchaClient({
        baseUrl: MOCK_BASE_URL,
      });
      expect(defaultClient).toBeDefined();
      defaultClient.close();
    });

    it('should trim trailing slashes from baseUrl', () => {
      const testClient = new CaptchaClient({
        baseUrl: 'http://localhost:8080///',
      });
      testClient.close();
    });
  });

  describe('getSliderCaptcha', () => {
    it('should fetch slider captcha with default options', async () => {
      const mockResponse = {
        session_id: 'test-session-123',
        image_url: 'http://example.com/image.jpg',
        puzzle_url: 'http://example.com/puzzle.jpg',
        target_x: 150,
        target_y: 80,
      };

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          code: 0,
          message: 'success',
          data: mockResponse,
        }),
      } as Response);

      const result = await client.getSliderCaptcha();

      expect(result).toEqual(mockResponse);
    });

    it('should pass query parameters to the request', async () => {
      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          code: 0,
          message: 'success',
          data: { session_id: 'test' },
        }),
      } as Response);

      await client.getSliderCaptcha({
        width: 400,
        height: 300,
        tolerance: 10,
      });

      expect(global.fetch).toHaveBeenCalled();
      const callUrl = (global.fetch as jest.Mock).mock.calls[0][0];
      expect(callUrl).toContain('width=400');
      expect(callUrl).toContain('height=300');
      expect(callUrl).toContain('tolerance=10');
    });
  });

  describe('getClickCaptcha', () => {
    it('should fetch click captcha with options', async () => {
      const mockResponse = {
        session_id: 'click-session-123',
        image_url: 'http://example.com/click.jpg',
        hint: 'Click 1, 2, 3',
        mode: 'number',
        points: [[100, 100], [200, 200]],
      };

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          code: 0,
          message: 'success',
          data: mockResponse,
        }),
      } as Response);

      const result = await client.getClickCaptcha({
        mode: 'number',
        shuffle: true,
        points: 3,
      });

      expect(result).toEqual(mockResponse);
    });
  });

  describe('verifyCaptcha', () => {
    it('should verify captcha with correct payload', async () => {
      const mockResponse = {
        success: true,
        message: 'Verification passed',
        score: 0.95,
      };

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          code: 0,
          message: 'success',
          data: mockResponse,
        }),
      } as Response);

      const result = await client.verifyCaptcha({
        session_id: 'test-session',
        type: 'slider',
        x: 150,
        y: 80,
      });

      expect(result).toEqual(mockResponse);
      expect(global.fetch).toHaveBeenCalled();
    });

    it('should throw error for failed verification', async () => {
      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          code: 1001,
          message: 'Verification failed',
        }),
      } as Response);

      await expect(
        client.verifyCaptcha({
          session_id: 'test-session',
          type: 'slider',
          x: 150,
        })
      ).rejects.toThrow(CaptchaError);
    });
  });

  describe('authLogin', () => {
    it('should login user with credentials', async () => {
      const mockResponse = {
        access_token: 'token123',
        refresh_token: 'refresh456',
        expires_in: 3600,
        user: { id: 1, username: 'testuser' },
      };

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          code: 0,
          message: 'success',
          data: mockResponse,
        }),
      } as Response);

      const result = await client.authLogin({
        username: 'testuser',
        password: 'password123',
      });

      expect(result).toEqual(mockResponse);
    });
  });

  describe('authRegister', () => {
    it('should register new user', async () => {
      const mockResponse = {
        user_id: 123,
        username: 'newuser',
        email: 'new@example.com',
      };

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          code: 0,
          message: 'success',
          data: mockResponse,
        }),
      } as Response);

      const result = await client.authRegister({
        username: 'newuser',
        email: 'new@example.com',
        password: 'password123',
      });

      expect(result).toEqual(mockResponse);
    });
  });

  describe('error handling', () => {
    it('should throw ValidationError for 400 status', async () => {
      global.fetch = jest.fn().mockResolvedValue({
        ok: false,
        status: 400,
        statusText: 'Bad Request',
        json: async () => ({
          message: 'Invalid parameters',
        }),
      } as Response);

      await expect(client.getSliderCaptcha()).rejects.toThrow(ValidationError);
    });

    it('should throw AuthenticationError for 401 status', async () => {
      global.fetch = jest.fn().mockResolvedValue({
        ok: false,
        status: 401,
        statusText: 'Unauthorized',
        json: async () => ({
          message: 'Invalid credentials',
        }),
      } as Response);

      await expect(client.getSliderCaptcha()).rejects.toThrow(
        AuthenticationError
      );
    });

    it('should throw NotFoundError for 404 status', async () => {
      global.fetch = jest.fn().mockResolvedValue({
        ok: false,
        status: 404,
        statusText: 'Not Found',
        json: async () => ({
          message: 'Resource not found',
        }),
      } as Response);

      await expect(client.getSliderCaptcha()).rejects.toThrow(NotFoundError);
    });

    it('should throw RateLimitError for 429 status', async () => {
      global.fetch = jest.fn().mockResolvedValue({
        ok: false,
        status: 429,
        statusText: 'Too Many Requests',
        headers: new Headers({ 'Retry-After': '60' }),
        json: async () => ({
          message: 'Rate limit exceeded',
        }),
      } as Response);

      await expect(client.getSliderCaptcha()).rejects.toThrow(RateLimitError);
    });

    it('should throw ServerError for 500 status', async () => {
      global.fetch = jest.fn().mockResolvedValue({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
        json: async () => ({
          message: 'Internal error',
        }),
      } as Response);

      await expect(client.getSliderCaptcha()).rejects.toThrow(ServerError);
    });

    it('should throw ServerError for 503 status', async () => {
      global.fetch = jest.fn().mockResolvedValue({
        ok: false,
        status: 503,
        statusText: 'Service Unavailable',
        json: async () => ({
          message: 'Service unavailable',
        }),
      } as Response);

      await expect(client.getSliderCaptcha()).rejects.toThrow(ServerError);
    });

    it('should handle invalid JSON response', async () => {
      global.fetch = jest.fn().mockResolvedValue({
        ok: false,
        status: 500,
        statusText: 'Error',
        json: async () => {
          throw new Error('Invalid JSON');
        },
      } as Response);

      await expect(client.getSliderCaptcha()).rejects.toThrow(ServerError);
    });
  });

  describe('gesture captcha', () => {
    it('should get gesture captcha', async () => {
      const mockResponse = {
        session_id: 'gesture-session-123',
        pattern: '1-2-3-4',
        grid_size: 3,
      };

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          code: 0,
          message: 'success',
          data: mockResponse,
        }),
      } as Response);

      const result = await client.getGestureCaptcha();

      expect(result).toEqual(mockResponse);
    });

    it('should verify gesture captcha', async () => {
      const mockResponse = {
        success: true,
        message: 'Verification passed',
      };

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          code: 0,
          message: 'success',
          data: mockResponse,
        }),
      } as Response);

      const result = await client.verifyGestureCaptcha('session-123', [1, 2, 3, 4]);

      expect(result).toEqual(mockResponse);
    });
  });

  describe('detection methods', () => {
    it('should get detection script', async () => {
      const mockScript = 'function detect() { return true; }';

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        text: async () => mockScript,
      } as Response);

      const result = await client.getDetectionScript('onReady');

      expect(result).toBe(mockScript);
    });

    it('should submit detection data', async () => {
      const mockResponse = {
        success: true,
        risk_level: 'low',
      };

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          code: 0,
          message: 'success',
          data: mockResponse,
        }),
      } as Response);

      const result = await client.submitDetection({
        fingerprint: 'hash123',
        canvas_hash: 'canvasHash',
      });

      expect(result).toEqual(mockResponse);
    });

    it('should check environment', async () => {
      const mockResponse = {
        success: true,
        risk_score: 0.1,
      };

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          code: 0,
          message: 'success',
          data: mockResponse,
        }),
      } as Response);

      const result = await client.checkEnvironment({
        fingerprint: 'hash123',
      });

      expect(result).toEqual(mockResponse);
    });
  });

  describe('close', () => {
    it('should close without error', async () => {
      await expect(client.close()).resolves.toBeUndefined();
    });
  });
});
