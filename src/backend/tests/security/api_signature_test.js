const { APISignature } = require('../../../backend/middleware/security/api_signature');

describe('API Signature Tests', () => {
  const testSecret = 'test-secret-key-12345';

  describe('Signature Generation', () => {
    test('should generate valid signature', () => {
      const params = { userId: 123, action: 'login' };
      const result = APISignature.generate(testSecret, params);
      
      expect(result.signature).toBeDefined();
      expect(result.timestamp).toBeDefined();
      expect(result.params).toBeDefined();
      expect(typeof result.signature).toBe('string');
      expect(result.signature.length).toBe(64);
    });

    test('should include timestamp in signature', () => {
      const params = { userId: 123 };
      const before = Date.now();
      const result = APISignature.generate(testSecret, params);
      const after = Date.now();

      expect(result.timestamp).toBeGreaterThanOrEqual(before);
      expect(result.timestamp).toBeLessThanOrEqual(after);
    });

    test('should sort parameters alphabetically', () => {
      const params = { zebra: 1, apple: 2, mango: 3 };
      const result = APISignature.generate(testSecret, params);
      
      expect(Object.keys(result.params)).toEqual(['apple', 'mango', 'zebra']);
    });

    test('should generate different signatures for different params', () => {
      const params1 = { userId: 123 };
      const params2 = { userId: 456 };
      
      const result1 = APISignature.generate(testSecret, params1);
      const result2 = APISignature.generate(testSecret, params2);
      
      expect(result1.signature).not.toBe(result2.signature);
    });

    test('should generate same signature for same params and timestamp', () => {
      const params = { userId: 123 };
      const timestamp = Date.now();
      
      const signature1 = APISignature.generate(testSecret, params);
      const result = APISignature.verify(testSecret, signature1.signature, timestamp, params);
      
      expect(result.valid).toBe(true);
    });
  });

  describe('Signature Verification', () => {
    test('should verify valid signature within time window', () => {
      const params = { userId: 123 };
      const { signature, timestamp } = APISignature.generate(testSecret, params);
      
      const result = APISignature.verify(testSecret, signature, timestamp, params);
      expect(result.valid).toBe(true);
      expect(result.reason).toBe('valid');
    });

    test('should reject expired signatures', () => {
      const params = { userId: 123 };
      const { signature } = APISignature.generate(testSecret, params);
      const expiredTimestamp = Date.now() - (10 * 60 * 1000);
      
      const result = APISignature.verify(testSecret, signature, expiredTimestamp, params);
      expect(result.valid).toBe(false);
      expect(result.reason).toBe('timestamp_expired');
    });

    test('should reject tampered signatures', () => {
      const params = { userId: 123 };
      const { signature } = APISignature.generate(testSecret, params);
      const tamperedSignature = signature.replace(/./g, c => 
        c === 'a' ? 'b' : c === 'b' ? 'a' : c
      );
      
      const result = APISignature.verify(testSecret, tamperedSignature, Date.now(), params);
      expect(result.valid).toBe(false);
      expect(result.reason).toBe('signature_mismatch');
    });

    test('should reject signatures with modified params', () => {
      const originalParams = { userId: 123 };
      const { signature, timestamp } = APISignature.generate(testSecret, originalParams);
      const modifiedParams = { userId: 456 };
      
      const result = APISignature.verify(testSecret, signature, timestamp, modifiedParams);
      expect(result.valid).toBe(false);
      expect(result.reason).toBe('signature_mismatch');
    });

    test('should reject signatures with wrong secret', () => {
      const params = { userId: 123 };
      const { signature, timestamp } = APISignature.generate(testSecret, params);
      
      const result = APISignature.verify('wrong-secret', signature, timestamp, params);
      expect(result.valid).toBe(false);
    });
  });

  describe('Parameter Handling', () => {
    test('should handle empty params', () => {
      const result = APISignature.generate(testSecret, {});
      expect(result.signature).toBeDefined();
      expect(result.params).toEqual({});
    });

    test('should handle null params', () => {
      const result = APISignature.generate(testSecret, null);
      expect(result.signature).toBeDefined();
      expect(result.params).toEqual({});
    });

    test('should handle special characters in params', () => {
      const params = { 
        name: "John O'Brien", 
        query: '<script>alert(1)</script>' 
      };
      const result = APISignature.generate(testSecret, params);
      expect(result.signature).toBeDefined();
    });

    test('should handle nested objects', () => {
      const params = { user: { id: 123, name: 'test' } };
      const result = APISignature.generate(testSecret, params);
      expect(result.params.user).toBe('{"id":123,"name":"test"}');
    });
  });
});
