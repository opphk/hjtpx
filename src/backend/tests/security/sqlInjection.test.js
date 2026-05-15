const SQLInjectionProtection = require('../../../backend/utils/sqlInjectionProtection');

describe('SQL Injection Tests', () => {
  describe('Common SQL Injection Patterns', () => {
    test('should block DROP TABLE attacks', () => {
      const maliciousInput = "'; DROP TABLE users;--";
      const sanitized = SQLInjectionProtection.sanitize(maliciousInput);
      expect(sanitized).not.toContain('DROP');
      expect(sanitized).not.toContain('--');
      expect(sanitized).toContain("''");
    });

    test('should block OR 1=1 attacks', () => {
      const maliciousInput = "1 OR 1=1";
      const sanitized = SQLInjectionProtection.sanitize(maliciousInput);
      expect(sanitized).toContain('1');
      expect(sanitized).not.toContain(' OR ');
    });

    test('should block AND 1=1 attacks', () => {
      const maliciousInput = "1' AND '1'='1";
      const sanitized = SQLInjectionProtection.sanitize(maliciousInput);
      expect(sanitized).toContain("''");
    });

    test('should block UNION SELECT attacks', () => {
      const maliciousInput = "1 UNION SELECT password FROM users--";
      const sanitized = SQLInjectionProtection.sanitize(maliciousInput);
      expect(sanitized).not.toContain('UNION');
      expect(sanitized).not.toContain('SELECT');
      expect(sanitized).not.toContain('--');
    });

    test('should block comment-based attacks', () => {
      const maliciousInput = "admin'--";
      const sanitized = SQLInjectionProtection.sanitize(maliciousInput);
      expect(sanitized).not.toContain('--');
    });

    test('should block stacked queries', () => {
      const maliciousInput = "; INSERT INTO users VALUES ('hacker','password');";
      const sanitized = SQLInjectionProtection.sanitize(maliciousInput);
      expect(sanitized).not.toContain('INSERT');
      expect(sanitized).toContain("''");
    });
  });

  describe('Second-Order SQL Injection', () => {
    test('should sanitize stored malicious input', () => {
      const storedInput = "'; DELETE FROM sessions;--";
      const sanitized = SQLInjectionProtection.sanitize(storedInput);
      expect(sanitized).not.toContain('DELETE');
    });
  });

  describe('Blind SQL Injection', () => {
    test('should escape dangerous SQL patterns', () => {
      const maliciousInput = "'; IF(1=1) EXEC('xp_cmdshell')--";
      const sanitized = SQLInjectionProtection.sanitize(maliciousInput);
      expect(sanitized).not.toContain('EXEC');
      expect(sanitized).toContain("''");
    });

    test('should block boolean-based blind injection', () => {
      const maliciousInput = "1 AND 1=1";
      const sanitized = SQLInjectionProtection.sanitize(maliciousInput);
      expect(sanitized).not.toContain('AND');
    });
  });

  describe('Edge Cases', () => {
    test('should handle empty string', () => {
      const input = '';
      const sanitized = SQLInjectionProtection.sanitize(input);
      expect(sanitized).toBe('');
    });

    test('should handle numeric input', () => {
      const input = 12345;
      const sanitized = SQLInjectionProtection.sanitize(input);
      expect(sanitized).toBe(12345);
    });

    test('should preserve legitimate special characters', () => {
      const input = "John O'Brien";
      const sanitized = SQLInjectionProtection.sanitize(input);
      expect(sanitized).toContain("O''Brien");
    });

    test('should handle null input', () => {
      const sanitized = SQLInjectionProtection.sanitize(null);
      expect(sanitized).toBe('');
    });
  });
});
