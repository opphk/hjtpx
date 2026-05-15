import {
  generateFingerprint,
  generateSessionId,
  generateToken
} from '../../../src/utils/fingerprint';

describe('fingerprint utils', () => {
  describe('generateSessionId', () => {
    test('generates unique session IDs', () => {
      const id1 = generateSessionId();
      const id2 = generateSessionId();

      expect(id1).not.toBe(id2);
    });

    test('includes timestamp', () => {
      const id = generateSessionId();
      const timestamp = id.split('-')[0];

      expect(parseInt(timestamp)).toBeLessThanOrEqual(Date.now());
    });
  });

  describe('generateToken', () => {
    test('generates valid token format', () => {
      const token = generateToken();

      expect(token).toMatch(/^[\w-]+-[\w-]+$/);
    });

    test('generates unique tokens', () => {
      const token1 = generateToken();
      const token2 = generateToken();

      expect(token1).not.toBe(token2);
    });
  });

  describe('generateFingerprint', () => {
    test('returns object with hash and components', async () => {
      const fingerprint = await generateFingerprint();

      expect(fingerprint).toHaveProperty('hash');
      expect(fingerprint).toHaveProperty('components');
    });

    test('hash is a valid hex string', async () => {
      const fingerprint = await generateFingerprint();

      expect(fingerprint.hash).toMatch(/^[a-f0-9]{64}$/);
    });

    test('components includes expected properties', async () => {
      const fingerprint = await generateFingerprint();

      expect(fingerprint.components).toHaveProperty('canvas');
      expect(fingerprint.components).toHaveProperty('screen');
      expect(fingerprint.components).toHaveProperty('timezone');
      expect(fingerprint.components).toHaveProperty('language');
    });

    test('generates consistent hash for same browser', async () => {
      const fp1 = await generateFingerprint();
      const fp2 = await generateFingerprint();

      expect(fp1.hash).toBe(fp2.hash);
    });
  });
});
