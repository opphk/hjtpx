const PKCE = require('../../src/backend/oauth/pkce');

describe('OAuth PKCE', () => {
  describe('generateCodeVerifier', () => {
    test('should generate a code verifier with default length', () => {
      const verifier = PKCE.generateCodeVerifier();
      expect(verifier).toHaveLength(128);
      expect(typeof verifier).toBe('string');
    });

    test('should generate a code verifier with custom length', () => {
      const verifier = PKCE.generateCodeVerifier(64);
      expect(verifier).toHaveLength(64);
    });

    test('should only contain valid characters', () => {
      const verifier = PKCE.generateCodeVerifier();
      const validCharsRegex = /^[A-Za-z0-9-._~]+$/;
      expect(validCharsRegex.test(verifier)).toBe(true);
    });

    test('should generate unique verifiers', () => {
      const verifier1 = PKCE.generateCodeVerifier();
      const verifier2 = PKCE.generateCodeVerifier();
      expect(verifier1).not.toBe(verifier2);
    });
  });

  describe('generateCodeChallenge', () => {
    test('should generate a code challenge from verifier', async () => {
      const verifier = PKCE.generateCodeVerifier();
      const challenge = await PKCE.generateCodeChallenge(verifier);

      expect(typeof challenge).toBe('string');
      expect(challenge.length).toBeGreaterThan(0);
      expect(challenge).not.toBe(verifier);
    });

    test('should generate URL-safe challenge', async () => {
      const verifier = PKCE.generateCodeVerifier();
      const challenge = await PKCE.generateCodeChallenge(verifier);

      expect(challenge).not.toContain('+');
      expect(challenge).not.toContain('/');
      expect(challenge).not.toContain('=');
    });

    test('should generate same challenge for same verifier', async () => {
      const verifier = PKCE.generateCodeVerifier();
      const challenge1 = await PKCE.generateCodeChallenge(verifier);
      const challenge2 = await PKCE.generateCodeChallenge(verifier);

      expect(challenge1).toBe(challenge2);
    });
  });

  describe('verifyCodeChallenge', () => {
    test('should verify valid challenge', async () => {
      const verifier = PKCE.generateCodeVerifier();
      const challenge = await PKCE.generateCodeChallenge(verifier);

      const isValid = await PKCE.verifyCodeChallenge(verifier, challenge);
      expect(isValid).toBe(true);
    });

    test('should reject invalid challenge', async () => {
      const verifier = PKCE.generateCodeVerifier();
      const wrongChallenge = PKCE.generateCodeVerifier(43);

      const isValid = await PKCE.verifyCodeChallenge(verifier, wrongChallenge);
      expect(isValid).toBe(false);
    });
  });

  describe('validateCodeVerifier', () => {
    test('should validate correct verifier', () => {
      const verifier = PKCE.generateCodeVerifier();
      expect(PKCE.validateCodeVerifier(verifier)).toBe(true);
    });

    test('should reject null verifier', () => {
      expect(PKCE.validateCodeVerifier(null)).toBe(false);
    });

    test('should reject undefined verifier', () => {
      expect(PKCE.validateCodeVerifier(undefined)).toBe(false);
    });

    test('should reject empty string verifier', () => {
      expect(PKCE.validateCodeVerifier('')).toBe(false);
    });

    test('should reject too short verifier', () => {
      expect(PKCE.validateCodeVerifier('abc')).toBe(false);
    });

    test('should reject too long verifier', () => {
      const longVerifier = 'a'.repeat(129);
      expect(PKCE.validateCodeVerifier(longVerifier)).toBe(false);
    });

    test('should reject verifier with invalid characters', () => {
      const invalidVerifier = 'abc@#$%^&*()def';
      expect(PKCE.validateCodeVerifier(invalidVerifier)).toBe(false);
    });

    test('should accept verifier with valid special characters', () => {
      const validVerifier = 'ABCabc123-._~ABCabc123-._~ABCabc123-._~ABCabc12';
      expect(PKCE.validateCodeVerifier(validVerifier)).toBe(true);
    });
  });

  describe('validateCodeChallenge', () => {
    test('should validate correct challenge', async () => {
      const verifier = PKCE.generateCodeVerifier();
      const challenge = await PKCE.generateCodeChallenge(verifier);

      expect(PKCE.validateCodeChallenge(challenge)).toBe(true);
    });

    test('should reject null challenge', () => {
      expect(PKCE.validateCodeChallenge(null)).toBe(false);
    });

    test('should reject empty challenge', () => {
      expect(PKCE.validateCodeChallenge('')).toBe(false);
    });

    test('should reject challenge with invalid characters', () => {
      expect(PKCE.validateCodeChallenge('abc@def')).toBe(false);
      expect(PKCE.validateCodeChallenge('abc def')).toBe(false);
    });
  });
});
