import { describe, it, expect, beforeEach } from '@jest/globals';
import { create } from 'domain';

describe('Middleware', () => {
  describe('createCaptchaMiddleware', () => {
    it('should create middleware function', () => {
      const middleware = createCaptchaMiddleware();
      expect(typeof middleware).toBe('function');
    });

    it('should accept custom options', () => {
      const middleware = createCaptchaMiddleware({
        apiKey: 'test-key',
        apiSecret: 'test-secret',
        protectedPaths: ['/api/*'],
        captchaPaths: ['/login', '/register']
      });
      expect(typeof middleware).toBe('function');
    });
  });
});

describe('createCaptchaXServer', () => {
  it('should create server with default config', async () => {
    const { createCaptchaXServer } = await import('../server');
    const server = createCaptchaXServer({
      apiKey: 'test-key',
      apiSecret: 'test-secret'
    });
    expect(server).toBeDefined();
  });

  it('should create server with custom serverUrl', async () => {
    const { createCaptchaXServer } = await import('../server');
    const server = createCaptchaXServer({
      apiKey: 'test-key',
      apiSecret: 'test-secret',
      serverUrl: 'https://custom.captchax.com'
    });
    expect(server).toBeDefined();
  });
});

describe('verifyCaptchaServer', () => {
  it('should handle missing config', async () => {
    const { verifyCaptchaServer } = await import('../server');
    
    const result = await verifyCaptchaServer('', {
      apiKey: '',
      apiSecret: ''
    });

    expect(result.success).toBe(false);
    expect(result.error).toBe('Token is required');
  });
});
