const XSSProtection = require('../../../frontend/src/utils/xss_protection');

describe('XSS Protection Tests', () => {
  describe('HTML Escaping', () => {
    test('should escape HTML characters in user input', () => {
      const input = '<script>alert("XSS")</script>';
      const sanitized = XSSProtection.sanitize(input);
      expect(sanitized).not.toContain('<script>');
      expect(sanitized).not.toContain('</script>');
      expect(sanitized).toContain('&lt;script&gt;');
    });

    test('should escape ampersand character', () => {
      const input = 'foo & bar';
      const sanitized = XSSProtection.sanitize(input);
      expect(sanitized).toContain('&amp;');
      expect(sanitized).not.toContain('& ');
    });

    test('should escape double quotes', () => {
      const input = 'He said "hello"';
      const sanitized = XSSProtection.sanitize(input);
      expect(sanitized).toContain('&quot;');
    });

    test('should escape single quotes', () => {
      const input = "It's a test";
      const sanitized = XSSProtection.sanitize(input);
      expect(sanitized).toContain('&#x27;');
    });

    test('should escape greater than and less than signs', () => {
      const input = 'a < b > c';
      const sanitized = XSSProtection.sanitize(input);
      expect(sanitized).toContain('&lt;');
      expect(sanitized).toContain('&gt;');
    });
  });

  describe('Dangerous Tag Removal', () => {
    test('should remove script tags completely', () => {
      const input = '<script>alert("XSS")</script>Hello';
      const sanitized = XSSProtection.sanitize(input);
      expect(sanitized).not.toContain('<script>');
      expect(sanitized).not.toContain('</script>');
    });

    test('should remove inline event handlers', () => {
      const input = '<img src="x" onerror="alert(1)">';
      const sanitized = XSSProtection.sanitize(input);
      expect(sanitized).not.toContain('onerror');
    });

    test('should remove javascript: protocol', () => {
      const input = '<a href="javascript:alert(1)">Click</a>';
      const sanitized = XSSProtection.sanitize(input);
      expect(sanitized).not.toContain('javascript:');
    });

    test('should remove onload event handlers', () => {
      const input = '<body onload="alert(1)">';
      const sanitized = XSSProtection.sanitize(input);
      expect(sanitized).not.toContain('onload');
    });

    test('should remove onClick handlers', () => {
      const input = '<div onClick="evil()">Click me</div>';
      const sanitized = XSSProtection.sanitize(input);
      expect(sanitized).not.toContain('onClick');
    });
  });

  describe('Stored XSS Prevention', () => {
    test('should sanitize stored content before rendering', () => {
      const maliciousStoredContent = '<div><script>stealCookies()</script></div>';
      const sanitized = XSSProtection.sanitize(maliciousStoredContent);
      expect(sanitized).not.toContain('<script>');
    });

    test('should handle nested malicious content', () => {
      const input = '<div><p><script>alert("nested")</script></p></div>';
      const sanitized = XSSProtection.sanitize(input);
      expect(sanitized).not.toContain('<script>');
    });
  });

  describe('Reflected XSS Prevention', () => {
    test('should escape user-provided URL parameters', () => {
      const userInput = '?name=<script>alert(1)</script>';
      const sanitized = XSSProtection.sanitize(userInput);
      expect(sanitized).not.toContain('<script>');
    });

    test('should escape JSON user input', () => {
      const userInput = JSON.stringify({ xss: '<script>alert(1)</script>' });
      const sanitized = XSSProtection.sanitize(userInput);
      expect(sanitized).not.toContain('<script>');
    });
  });

  describe('DOM-based XSS Prevention', () => {
    test('should escape user input before DOM insertion', () => {
      const userInput = '${userVariable}<img src=x onerror=alert(1)>';
      const sanitized = XSSProtection.sanitize(userInput);
      expect(sanitized).not.toContain('<img');
      expect(sanitized).not.toContain('onerror');
    });

    test('should handle template literal injection', () => {
      const userInput = '${constructor.constructor("alert(1)")()}';
      const sanitized = XSSProtection.sanitize(userInput);
      expect(sanitized).not.toContain('${');
      expect(sanitized).not.toContain('constructor');
    });
  });

  describe('Edge Cases', () => {
    test('should handle empty string', () => {
      const input = '';
      const sanitized = XSSProtection.sanitize(input);
      expect(sanitized).toBe('');
    });

    test('should handle null input', () => {
      const sanitized = XSSProtection.sanitize(null);
      expect(sanitized).toBe('');
    });

    test('should handle undefined input', () => {
      const sanitized = XSSProtection.sanitize(undefined);
      expect(sanitized).toBe('');
    });

    test('should handle numeric input', () => {
      const input = 12345;
      const sanitized = XSSProtection.sanitize(input);
      expect(sanitized).toBe('12345');
    });

    test('should preserve normal HTML', () => {
      const input = '<p>Hello, World!</p>';
      const sanitized = XSSProtection.sanitize(input);
      expect(sanitized).toBe('&lt;p&gt;Hello, World!&lt;/p&gt;');
    });
  });
});
