import { describe, it, expect } from '@jest/globals';
import { CaptchaXServer, createCaptchaXServer } from '../server/client';

describe('CaptchaXServer', () => {
  describe('constructor', () => {
    it('should create instance with config', () => {
      const client = new CaptchaXServer({
        apiKey: 'test-api-key',
        apiSecret: 'test-api-secret'
      });
      expect(client).toBeDefined();
    });

    it('should use default serverUrl if not provided', () => {
      const client = new CaptchaXServer({
        apiKey: 'test-api-key',
        apiSecret: 'test-api-secret'
      });
      expect(client).toBeDefined();
    });

    it('should use custom serverUrl if provided', () => {
      const client = new CaptchaXServer({
        apiKey: 'test-api-key',
        apiSecret: 'test-api-secret',
        serverUrl: 'https://custom.captchax.com'
      });
      expect(client).toBeDefined();
    });
  });

  describe('createCaptchaXServer', () => {
    it('should create server instance', () => {
      const client = createCaptchaXServer({
        apiKey: 'test-api-key',
        apiSecret: 'test-api-secret'
      });
      expect(client).toBeInstanceOf(CaptchaXServer);
    });
  });

  describe('verify', () => {
    it('should return error for empty token', async () => {
      const client = new CaptchaXServer({
        apiKey: 'test-api-key',
        apiSecret: 'test-api-secret'
      });

      const result = await client.verify({
        token: ''
      });

      expect(result.success).toBe(false);
      expect(result.error).toBe('Token is required');
    });

    it('should handle network errors gracefully', async () => {
      const client = new CaptchaXServer({
        apiKey: 'test-api-key',
        apiSecret: 'test-api-secret',
        serverUrl: 'https://invalid-domain-that-does-not-exist.com'
      });

      const result = await client.verify({
        token: 'test-token'
      });

      expect(result.success).toBe(false);
      expect(result.error).toBeDefined();
    });
  });

  describe('getChallenge', () => {
    it('should handle network errors gracefully', async () => {
      const client = new CaptchaXServer({
        apiKey: 'test-api-key',
        apiSecret: 'test-api-secret',
        serverUrl: 'https://invalid-domain-that-does-not-exist.com'
      });

      const result = await client.getChallenge('test-scene');

      expect(result.success).toBe(false);
      expect(result.error).toBeDefined();
    });
  });
});
