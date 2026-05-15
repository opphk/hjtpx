import { describe, it, expect } from '@jest/globals';
import { generateToken, generateSignature, verifySignature, isTimestampValid } from '../server/utils';

describe('Server Utils', () => {
  describe('generateToken', () => {
    it('should generate a hex string token', () => {
      const token = generateToken();
      expect(typeof token).toBe('string');
      expect(token).toMatch(/^[a-f0-9]+$/);
    });

    it('should generate token with correct length', () => {
      const token = generateToken();
      expect(token.length).toBe(64);
    });

    it('should generate unique tokens', () => {
      const token1 = generateToken();
      const token2 = generateToken();
      expect(token1).not.toBe(token2);
    });
  });

  describe('generateSignature', () => {
    it('should generate a hex signature', () => {
      const signature = generateSignature('secret', 'data', 123456789);
      expect(typeof signature).toBe('string');
      expect(signature).toMatch(/^[a-f0-9]+$/);
    });

    it('should generate consistent signatures for same input', () => {
      const sig1 = generateSignature('secret', 'data', 123456789);
      const sig2 = generateSignature('secret', 'data', 123456789);
      expect(sig1).toBe(sig2);
    });

    it('should generate different signatures for different secrets', () => {
      const sig1 = generateSignature('secret1', 'data', 123456789);
      const sig2 = generateSignature('secret2', 'data', 123456789);
      expect(sig1).not.toBe(sig2);
    });

    it('should generate different signatures for different data', () => {
      const sig1 = generateSignature('secret', 'data1', 123456789);
      const sig2 = generateSignature('secret', 'data2', 123456789);
      expect(sig1).not.toBe(sig2);
    });
  });

  describe('verifySignature', () => {
    it('should verify valid signature', () => {
      const signature = generateSignature('secret', 'data', 123456789);
      const isValid = verifySignature('secret', 'data', 123456789, signature);
      expect(isValid).toBe(true);
    });

    it('should reject invalid signature', () => {
      const signature = generateSignature('secret', 'data', 123456789);
      const isValid = verifySignature('secret', 'data', 123456789, signature + '0');
      expect(isValid).toBe(false);
    });

    it('should reject signature with wrong secret', () => {
      const signature = generateSignature('secret1', 'data', 123456789);
      const isValid = verifySignature('secret2', 'data', 123456789, signature);
      expect(isValid).toBe(false);
    });
  });

  describe('isTimestampValid', () => {
    it('should accept timestamp within window', () => {
      const now = Date.now();
      const isValid = isTimestampValid(now, 300000);
      expect(isValid).toBe(true);
    });

    it('should accept timestamp slightly older than window', () => {
      const now = Date.now();
      const timestamp = now - 60000;
      const isValid = isTimestampValid(timestamp, 120000);
      expect(isValid).toBe(true);
    });

    it('should reject timestamp too old', () => {
      const now = Date.now();
      const timestamp = now - 600000;
      const isValid = isTimestampValid(timestamp, 300000);
      expect(isValid).toBe(false);
    });
  });
});
