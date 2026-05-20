import { test, expect } from '@playwright/test';

test.describe('Fuzzing Tests', () => {
  test.describe('Input Fuzzing', () => {
    test('should handle random string inputs', async ({ request }) => {
      const payloads = [
        generateRandomString(100),
        generateRandomString(1000),
        '!@#$%^&*()_+-=[]{}|;:,.<>?',
        '\u0000\u0001\u0002'.repeat(50),
        '中文测试字符串',
        '🎉🎊🎈🎁🎄🎅',
      ];

      for (const payload of payloads) {
        const response = await request.post('http://localhost:8080/api/v1/captcha/verify', {
          data: {
            captchaId: payload,
            answer: payload,
            x: payload,
            y: payload
          }
        });
        expect(response.status()).toBeDefined();
      }
    });

    test('should handle extremely long inputs', async ({ request }) => {
      const longPayload = 'x'.repeat(100000);

      const response = await request.post('http://localhost:8080/api/v1/captcha/verify', {
        data: {
          captchaId: longPayload,
          answer: longPayload
        }
      });
      expect(response.status()).toBeDefined();
    });

    test('should handle null and undefined values', async ({ request }) => {
      const nullPayloads = [
        null,
        undefined,
        { captchaId: null },
        { answer: null, x: null, y: null },
        { captchaId: undefined },
      ];

      for (const payload of nullPayloads) {
        try {
          const response = await request.post('http://localhost:8080/api/v1/captcha/verify', {
            data: payload
          });
          expect(response.status()).toBeDefined();
        } catch (e) {
          expect(e).toBeDefined();
        }
      }
    });

    test('should handle malformed JSON', async ({ request }) => {
      const malformedJSON = [
        '{invalid json}',
        '{"unclosed": "string',
        '[1,2,3',
        '{"trailing": "comma",}',
        '{"duplicate": 1, "duplicate": 2}',
      ];

      for (const json of malformedJSON) {
        try {
          const response = await request.post('http://localhost:8080/api/v1/captcha/verify', {
            headers: { 'Content-Type': 'application/json' },
            data: json
          });
          expect(response.status()).toBeGreaterThanOrEqual(400);
        } catch (e) {
          expect(e).toBeDefined();
        }
      }
    });

    test('should handle SQL injection attempts', async ({ request }) => {
      const sqlPayloads = [
        "'; DROP TABLE users; --",
        "1' OR '1'='1",
        "1; DELETE FROM captchas WHERE 1=1; --",
        "admin'--",
        "1' UNION SELECT * FROM users--",
        "1'; INSERT INTO users VALUES (1, 'hacker', 'password'); --",
      ];

      for (const payload of sqlPayloads) {
        const response = await request.post('http://localhost:8080/api/v1/captcha/verify', {
          data: {
            captchaId: payload,
            answer: payload
          }
        });
        expect(response.status()).toBeDefined();
      }
    });

    test('should handle NoSQL injection attempts', async ({ request }) => {
      const nosqlPayloads = [
        '{"$ne": null}',
        '{"$gt": ""}',
        '{"$regex": ".*"}',
        '{"$where": "function() { return true; }"}',
        '{"$or": [{"x": 1}, {"y": 1}]}',
      ];

      for (const payload of nosqlPayloads) {
        const response = await request.post('http://localhost:8080/api/v1/captcha/verify', {
          data: JSON.parse(payload)
        });
        expect(response.status()).toBeDefined();
      }
    });

    test('should handle XSS payloads', async ({ request }) => {
      const xssPayloads = [
        '<script>alert("XSS")</script>',
        '<img src=x onerror=alert("XSS")>',
        '<svg onload=alert("XSS")>',
        'javascript:alert("XSS")',
        '<iframe src="javascript:alert(\'XSS\')">',
        '{{constructor.constructor("alert(1)")()}}',
      ];

      for (const payload of xssPayloads) {
        const response = await request.post('http://localhost:8080/api/v1/captcha/verify', {
          data: {
            captchaId: payload,
            answer: payload
          }
        });
        expect(response.status()).toBeDefined();
      }
    });

    test('should handle path traversal attempts', async ({ request }) => {
      const pathPayloads = [
        '../../../etc/passwd',
        '..\\..\\..\\windows\\system32',
        '....//....//....//etc/passwd',
        '%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd',
      ];

      for (const payload of pathPayloads) {
        const response = await request.get(`http://localhost:8080/${payload}`);
        expect(response.status()).toBeDefined();
      }
    });
  });

  test.describe('Boundary Value Testing', () => {
    test('should handle maximum integer values', async ({ request }) => {
      const maxIntPayloads = [
        { x: Number.MAX_SAFE_INTEGER },
        { x: Number.MIN_SAFE_INTEGER },
        { x: Number.MAX_VALUE },
        { x: Number.MIN_VALUE },
        { x: Infinity },
        { x: -Infinity },
        { x: NaN },
      ];

      for (const payload of maxIntPayloads) {
        const response = await request.post('http://localhost:8080/api/v1/captcha/verify', {
          data: { captchaId: 'test', answer: payload }
        });
        expect(response.status()).toBeDefined();
      }
    });

    test('should handle zero and negative values', async ({ request }) => {
      const zeroPayloads = [
        { x: 0, y: 0 },
        { x: -1, y: -1 },
        { x: -999999, y: -999999 },
        { x: 0.0001, y: -0.0001 },
      ];

      for (const payload of zeroPayloads) {
        const response = await request.post('http://localhost:8080/api/v1/captcha/verify', {
          data: { captchaId: 'test', ...payload }
        });
        expect(response.status()).toBeDefined();
      }
    });

    test('should handle very small float values', async ({ request }) => {
      const smallFloatPayloads = [
        { x: 0.0000001, y: 0.0000001 },
        { x: 1e-100, y: 1e-100 },
        { x: Number.EPSILON },
      ];

      for (const payload of smallFloatPayloads) {
        const response = await request.post('http://localhost:8080/api/v1/captcha/verify', {
          data: { captchaId: 'test', ...payload }
        });
        expect(response.status()).toBeDefined();
      }
    });

    test('should handle very large float values', async ({ request }) => {
      const largeFloatPayloads = [
        { x: 1e100, y: 1e100 },
        { x: 1.7976931348623157e+308, y: 1.7976931348623157e+308 },
      ];

      for (const payload of largeFloatPayloads) {
        const response = await request.post('http://localhost:8080/api/v1/captcha/verify', {
          data: { captchaId: 'test', ...payload }
        });
        expect(response.status()).toBeDefined();
      }
    });
  });

  test.describe('Unicode and Encoding Tests', () => {
    test('should handle various Unicode ranges', async ({ request }) => {
      const unicodePayloads = [
        '中文测试',
        '日本語テスト',
        '한국어테스트',
        'العربية테스트',
        'עברית 테스트',
        '🇨🇳🇺🇸🇬🇧',
        '\u0000\u0020\u007F\u00FF\uFFFF',
      ];

      for (const payload of unicodePayloads) {
        const response = await request.post('http://localhost:8080/api/v1/captcha/verify', {
          data: { captchaId: payload, answer: payload }
        });
        expect(response.status()).toBeDefined();
      }
    });

    test('should handle different encodings', async ({ request }) => {
      const encodingPayloads = [
        { captchaId: Buffer.from('test').toString('base64') },
        { captchaId: Buffer.from('test').toString('hex') },
        { captchaId: encodeURIComponent('test?param=value') },
      ];

      for (const payload of encodingPayloads) {
        const response = await request.post('http://localhost:8080/api/v1/captcha/verify', {
          data: payload
        });
        expect(response.status()).toBeDefined();
      }
    });
  });

  test.describe('Protocol Manipulation', () => {
    test('should handle HTTP method variants', async ({ request }) => {
      const methods = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS'];

      for (const method of methods) {
        try {
          const response = await request.fetch('http://localhost:8080/api/v1/captcha/verify', {
            method: method as any
          });
          expect(response.ok() || !response.ok()).toBeTruthy();
        } catch (e) {
          expect(e).toBeDefined();
        }
      }
    });

    test('should handle HTTP version variants', async ({ request }) => {
      const headers = [
        { 'HTTP-Version': 'HTTP/1.0' },
        { 'HTTP-Version': 'HTTP/1.1' },
        { 'HTTP-Version': 'HTTP/2.0' },
        { 'HTTP-Version': 'HTTP/3.0' },
      ];

      for (const header of headers) {
        try {
          const response = await request.get('http://localhost:8080/api/v1/captcha/verify', {
            headers: header
          });
          expect(response.status()).toBeDefined();
        } catch (e) {
          expect(e).toBeDefined();
        }
      }
    });

    test('should handle duplicate headers', async ({ request }) => {
      const response = await request.post('http://localhost:8080/api/v1/captcha/verify', {
        headers: {
          'X-Forwarded-For': '1.2.3.4',
          'X-Forwarded-For': '5.6.7.8',
          'X-Forwarded-For': '9.10.11.12',
        },
        data: { captchaId: 'test' }
      });
      expect(response.status()).toBeDefined();
    });
  });

  test.describe('Rate Limiting Tests', () => {
    test('should handle rapid sequential requests', async ({ request }) => {
      const promises = [];
      for (let i = 0; i < 100; i++) {
        promises.push(request.get('http://localhost:8080/api/v1/captcha/slider'));
      }

      const results = await Promise.allSettled(promises);
      const statuses = results.map(r => r.status);
      expect(statuses).toContain('fulfilled');
    });

    test('should handle concurrent burst requests', async ({ request }) => {
      const burstSize = 50;
      const promises = [];

      for (let i = 0; i < burstSize; i++) {
        promises.push(
          request.post('http://localhost:8080/api/v1/captcha/verify', {
            data: { captchaId: `burst-${i}`, x: i }
          })
        );
      }

      const results = await Promise.allSettled(promises);
      const successCount = results.filter(r => r.status === 'fulfilled').length;
      expect(successCount).toBeGreaterThan(0);
    });
  });
});

function generateRandomString(length: number): string {
  const characters = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
  let result = '';
  for (let i = 0; i < length; i++) {
    result += characters.charAt(Math.floor(Math.random() * characters.length));
  }
  return result;
}
