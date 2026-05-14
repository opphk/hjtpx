const { escapeHtml, escapeHtmlDeep, removeXssPatterns, sanitizeObject } = require('../../utils/xssSanitizer');

describe('XSS Sanitizer', () => {
  describe('escapeHtml', () => {
    test('should escape HTML special characters', () => {
      expect(escapeHtml('<script>alert("xss")</script>')).toBe(
        '&lt;script&gt;alert(&quot;xss&quot;)&lt;&#x2F;script&gt;'
      );
    });

    test('should escape ampersand', () => {
      expect(escapeHtml('foo & bar')).toBe('foo &amp; bar');
    });

    test('should escape quotes', () => {
      expect(escapeHtml("it's a \"test\"")).toBe('it&#x27;s a &quot;test&quot;');
    });

    test('should handle non-string inputs', () => {
      expect(escapeHtml(123)).toBe(123);
      expect(escapeHtml(null)).toBe(null);
      expect(escapeHtml(undefined)).toBe(undefined);
    });

    test('should escape backticks and equals', () => {
      expect(escapeHtml('`test` = value')).toBe('&#x60;test&#x60; &#x3D; value');
    });
  });

  describe('escapeHtmlDeep', () => {
    test('should escape strings in objects', () => {
      const input = { name: '<b>John</b>', bio: '<script>evil()</script>' };
      const result = escapeHtmlDeep(input);
      expect(result.name).toBe('&lt;b&gt;John&lt;&#x2F;b&gt;');
      expect(result.bio).toBe('&lt;script&gt;evil()&lt;&#x2F;script&gt;');
    });

    test('should escape strings in arrays', () => {
      const input = ['<div>test</div>', { name: '<span>hi</span>' }];
      const result = escapeHtmlDeep(input);
      expect(result[0]).toBe('&lt;div&gt;test&lt;&#x2F;div&gt;');
      expect(result[1].name).toBe('&lt;span&gt;hi&lt;&#x2F;span&gt;');
    });

    test('should preserve non-string values', () => {
      const input = { count: 42, active: true, data: null };
      const result = escapeHtmlDeep(input);
      expect(result).toEqual(input);
    });
  });

  describe('removeXssPatterns', () => {
    test('should remove javascript: protocol', () => {
      expect(removeXssPatterns('javascript:alert(1)')).toBe('alert(1)');
    });

    test('should remove event handlers', () => {
      expect(removeXssPatterns('<img onerror=alert(1)>')).toBe('<img alert(1)>');
    });

    test('should remove data:text/html', () => {
      expect(removeXssPatterns('data:text/html,<script>')).toBe(',<script>');
    });

    test('should remove vbscript: protocol', () => {
      expect(removeXssPatterns('vbscript:msgbox("hi")')).toBe('msgbox("hi")');
    });

    test('should handle non-string inputs', () => {
      expect(removeXssPatterns(123)).toBe(123);
    });
  });

  describe('sanitizeObject', () => {
    test('should sanitize request body', () => {
      const req = {
        body: {
          username: '<script>alert("xss")</script>',
          email: 'test@example.com'
        }
      };
      const sanitized = sanitizeObject(req.body);
      expect(sanitized.username).not.toContain('<script>');
      expect(sanitized.email).toBe('test@example.com');
    });

    test('should respect maxLength option', () => {
      const input = 'a'.repeat(200);
      const result = sanitizeObject(input, { maxLength: 100 });
      expect(result.length).toBe(100);
    });

    test('should handle nested objects', () => {
      const input = {
        user: {
          profile: {
            name: '<b>Test</b>'
          }
        }
      };
      const result = sanitizeObject(input);
      expect(result.user.profile.name).not.toContain('<b>');
    });
  });
});

describe('XSS Protection Middleware', () => {
  const { xssProtection, addSecurityHeaders } = require('../../middleware/xssProtection');

  describe('xssProtection middleware', () => {
    test('should sanitize request body', () => {
      const req = {
        body: { name: '<script>alert(1)</script>' },
        query: {},
        params: {},
        path: '/api/test'
      };
      const res = {};
      const next = jest.fn();

      const middleware = xssProtection();
      middleware(req, res, next);

      expect(req.body.name).not.toContain('<script>');
      expect(next).toHaveBeenCalled();
    });

    test('should sanitize query parameters', () => {
      const req = {
        body: {},
        query: { search: '<img onerror=alert(1)>' },
        params: {},
        path: '/api/search'
      };
      const res = {};
      const next = jest.fn();

      const middleware = xssProtection();
      middleware(req, res, next);

      expect(req.query.search).not.toContain('onerror');
      expect(next).toHaveBeenCalled();
    });

    test('should skip excluded paths', () => {
      const req = {
        body: { name: '<script>alert(1)</script>' },
        query: {},
        params: {},
        path: '/webhook/external'
      };
      const res = {};
      const next = jest.fn();

      const middleware = xssProtection({ excludedPaths: ['/webhook'] });
      middleware(req, res, next);

      expect(req.body.name).toBe('<script>alert(1)</script>');
      expect(next).toHaveBeenCalled();
    });
  });

  describe('addSecurityHeaders middleware', () => {
    test('should set security headers', () => {
      const req = {};
      const res = {
        setHeader: jest.fn()
      };
      const next = jest.fn();

      const middleware = addSecurityHeaders();
      middleware(req, res, next);

      expect(res.setHeader).toHaveBeenCalledWith('X-XSS-Protection', '1; mode=block');
      expect(res.setHeader).toHaveBeenCalledWith('X-Content-Type-Options', 'nosniff');
      expect(res.setHeader).toHaveBeenCalledWith('X-Frame-Options', 'DENY');
      expect(next).toHaveBeenCalled();
    });
  });
});
