import { CaptchaXClient, CaptchaXError } from '../src';

const BASE_URL = 'https://captchax.example.com';
const APP_ID = 'test-app';

describe('CaptchaXClient', () => {
  let client: CaptchaXClient;

  beforeEach(() => {
    client = new CaptchaXClient({
      baseUrl: BASE_URL,
      appId: APP_ID,
      timeout: 5000,
      retryTimes: 2,
    });
  });

  describe('constructor', () => {
    it('should create client with valid config', () => {
      expect(client).toBeInstanceOf(CaptchaXClient);
    });

    it('should throw error when baseUrl is missing', () => {
      expect(() => {
        new CaptchaXClient({ baseUrl: '' });
      }).toThrow('baseUrl is required');
    });

    it('should use v1 by default', () => {
      expect(client.getApiVersion()).toBe('v1');
    });

    it('should accept custom apiVersion', () => {
      const clientV2 = new CaptchaXClient({
        baseUrl: BASE_URL,
        apiVersion: 'v2',
      });
      expect(clientV2.getApiVersion()).toBe('v2');
    });
  });

  describe('setAppId', () => {
    it('should update appId', () => {
      client.setAppId('new-app-id');
      expect(() => {
        client.generateSliderCaptcha();
      }).toThrow('appId is required');
    });
  });

  describe('setApiVersion', () => {
    it('should update api version to v2', () => {
      client.setApiVersion('v2');
      expect(client.getApiVersion()).toBe('v2');
    });

    it('should update api version back to v1', () => {
      client.setApiVersion('v2');
      client.setApiVersion('v1');
      expect(client.getApiVersion()).toBe('v1');
    });
  });

  describe('createClientInfo', () => {
    it('should create client info with default values', () => {
      const info = client.createClientInfo();
      const parsed = JSON.parse(info);
      expect(parsed).toHaveProperty('timestamp');
    });

    it('should include custom fields', () => {
      const info = client.createClientInfo({ customField: 'value' });
      const parsed = JSON.parse(info);
      expect(parsed.customField).toBe('value');
    });
  });
});

describe('CaptchaXError', () => {
  it('should create error with default values', () => {
    const error = new CaptchaXError('Test error');
    expect(error.message).toBe('Test error');
    expect(error.code).toBe(500);
    expect(error.statusCode).toBe(500);
    expect(error.name).toBe('CaptchaXError');
  });

  it('should create error with custom values', () => {
    const error = new CaptchaXError('Test error', 400, 400, { details: 'test' });
    expect(error.message).toBe('Test error');
    expect(error.code).toBe(400);
    expect(error.statusCode).toBe(400);
    expect(error.details).toEqual({ details: 'test' });
  });

  it('should be instance of Error', () => {
    const error = new CaptchaXError('Test error');
    expect(error instanceof Error).toBe(true);
  });
});
