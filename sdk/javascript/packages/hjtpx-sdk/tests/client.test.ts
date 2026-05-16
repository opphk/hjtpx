import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { CaptchaClient } from '../src/index';
import { 
  SDKError, 
  NetworkError, 
  TimeoutError, 
  InvalidParamsError,
  RateLimitedError,
  UnauthorizedError,
  ServerError,
  isSDKError,
  getErrorCode,
} from '../src/errors';

describe('CaptchaClient', () => {
  let client: CaptchaClient;

  beforeEach(() => {
    client = new CaptchaClient({
      baseUrl: 'http://localhost:8080',
      debugMode: true,
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('constructor', () => {
    it('should create client with default config', () => {
      const c = new CaptchaClient();
      expect(c).toBeDefined();
      expect(c.getConfig().baseUrl).toBe('http://localhost:8080');
    });

    it('should create client with custom config', () => {
      const c = new CaptchaClient({
        baseUrl: 'https://api.example.com',
        appId: 'test-id',
        appSecret: 'test-secret',
        timeout: 60000,
        maxRetries: 5,
      });

      const config = c.getConfig();
      expect(config.baseUrl).toBe('https://api.example.com');
      expect(config.appId).toBe('test-id');
      expect(config.appSecret).toBe('test-secret');
      expect(config.timeout).toBe(60000);
      expect(config.maxRetries).toBe(5);
    });
  });

  describe('runtime configuration', () => {
    it('should set debug mode', () => {
      client.setDebugMode(true);
      expect(client.getConfig().debugMode).toBe(true);

      client.setDebugMode(false);
      expect(client.getConfig().debugMode).toBe(false);
    });

    it('should set timeout', () => {
      client.setTimeout(60000);
      expect(client.getConfig().timeout).toBe(60000);
    });

    it('should set max retries', () => {
      client.setMaxRetries(10);
      expect(client.getConfig().maxRetries).toBe(10);
    });

    it('should set retry delay', () => {
      client.setRetryDelay(500);
      expect(client.getConfig().retryDelay).toBe(500);
    });
  });

  describe('getStats', () => {
    it('should return stats', () => {
      const stats = client.getStats();
      expect(stats).toBeDefined();
      expect(stats.totalRequests).toBe(0);
      expect(stats.successfulRequests).toBe(0);
      expect(stats.failedRequests).toBe(0);
      expect(stats.retriedRequests).toBe(0);
      expect(stats.successRate).toBe(0);
    });
  });

  describe('extractBase64Image', () => {
    it('should extract PNG image', () => {
      const testData = 'data:image/png;base64,SGVsbG8gV29ybGQ=';
      const result = client.extractBase64Image(testData);
      expect(result).toBeDefined();
    });

    it('should extract JPEG image', () => {
      const testData = 'data:image/jpeg;base64,SGVsbG8gV29ybGQ=';
      const result = client.extractBase64Image(testData);
      expect(result).toBeDefined();
    });

    it('should throw error for empty data URI', () => {
      expect(() => client.extractBase64Image('')).toThrow(InvalidParamsError);
    });

    it('should throw error for unsupported format', () => {
      expect(() => client.extractBase64Image('data:image/gif;base64,abc')).toThrow(InvalidParamsError);
    });
  });

  describe('generateImageCaptcha validation', () => {
    it('should generate image captcha', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: {
          challenge_id: 'test-challenge-id',
          image: 'data:image/png;base64,abc',
        },
      };

      const fetchMock = vi.fn().mockResolvedValue({
        status: 200,
        headers: new Map(),
        json: async () => mockResponse,
      });

      const testClient = new CaptchaClient({
        fetch: fetchMock,
      });

      const result = await testClient.generateImageCaptcha();
      expect(result.challenge_id).toBe('test-challenge-id');
      expect(fetchMock).toHaveBeenCalled();
    });
  });

  describe('verifyImageCaptcha validation', () => {
    it('should throw error for missing challenge ID', async () => {
      await expect(client.verifyImageCaptcha('', '1234')).rejects.toThrow(InvalidParamsError);
    });

    it('should throw error for missing answer', async () => {
      await expect(client.verifyImageCaptcha('test-id', '')).rejects.toThrow(InvalidParamsError);
    });
  });

  describe('verifySliderCaptcha validation', () => {
    it('should throw error for missing challenge ID', async () => {
      await expect(client.verifySliderCaptcha('', '120')).rejects.toThrow(InvalidParamsError);
    });

    it('should throw error for missing offset', async () => {
      await expect(client.verifySliderCaptcha('test-id', '')).rejects.toThrow(InvalidParamsError);
    });
  });

  describe('verifyClickCaptcha validation', () => {
    it('should throw error for missing challenge ID', async () => {
      await expect(
        client.verifyClickCaptcha('', [{ x: 100, y: 120, duration: 500 }])
      ).rejects.toThrow(InvalidParamsError);
    });

    it('should throw error for empty clicks', async () => {
      await expect(client.verifyClickCaptcha('test-id', [])).rejects.toThrow(InvalidParamsError);
    });

    it('should throw error for null clicks', async () => {
      await expect(client.verifyClickCaptcha('test-id', null as any)).rejects.toThrow(InvalidParamsError);
    });
  });
});

describe('Error classes', () => {
  describe('SDKError', () => {
    it('should create error with code and message', () => {
      const error = new SDKError(500, 'Test error');
      expect(error.code).toBe(500);
      expect(error.message).toBe('Test error');
      expect(error.isServerError()).toBe(true);
      expect(error.isRateLimited()).toBe(false);
      expect(error.isUnauthorized()).toBe(false);
      expect(error.isInvalidParams()).toBe(false);
    });

    it('should create error with retryAfter', () => {
      const error = new SDKError(429, 'Rate limited', 60);
      expect(error.code).toBe(429);
      expect(error.retryAfter).toBe(60);
      expect(error.isRateLimited()).toBe(true);
    });
  });

  describe('NetworkError', () => {
    it('should create network error', () => {
      const error = new NetworkError('Connection failed');
      expect(error.code).toBe(0);
      expect(error.message).toBe('Connection failed');
    });
  });

  describe('TimeoutError', () => {
    it('should create timeout error', () => {
      const error = new TimeoutError();
      expect(error.code).toBe(408);
      expect(error.message).toBe('Request timeout');
    });
  });

  describe('InvalidParamsError', () => {
    it('should create invalid params error', () => {
      const error = new InvalidParamsError('Invalid param');
      expect(error.code).toBe(400);
      expect(error.isInvalidParams()).toBe(true);
    });
  });

  describe('RateLimitedError', () => {
    it('should create rate limited error', () => {
      const error = new RateLimitedError('Rate limited', 30);
      expect(error.code).toBe(429);
      expect(error.retryAfter).toBe(30);
      expect(error.isRateLimited()).toBe(true);
    });
  });

  describe('UnauthorizedError', () => {
    it('should create unauthorized error', () => {
      const error = new UnauthorizedError();
      expect(error.code).toBe(401);
      expect(error.isUnauthorized()).toBe(true);
    });
  });

  describe('ServerError', () => {
    it('should create server error', () => {
      const error = new ServerError(500);
      expect(error.code).toBe(500);
      expect(error.isServerError()).toBe(true);
    });
  });

  describe('isSDKError', () => {
    it('should return true for SDKError', () => {
      const error = new SDKError(500, 'Test');
      expect(isSDKError(error)).toBe(true);
    });

    it('should return false for regular Error', () => {
      const error = new Error('Test');
      expect(isSDKError(error)).toBe(false);
    });
  });

  describe('getErrorCode', () => {
    it('should return error code for SDKError', () => {
      const error = new SDKError(500, 'Test');
      expect(getErrorCode(error)).toBe(500);
    });

    it('should return 0 for regular Error', () => {
      const error = new Error('Test');
      expect(getErrorCode(error)).toBe(0);
    });
  });
});
