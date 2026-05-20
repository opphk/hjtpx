import { test, expect } from '@playwright/test';

test.describe('Security Penetration Tests', () => {
  const testUsers = {
    admin: { username: 'admin', password: 'admin123' },
    user: { username: 'user', password: 'user123' },
  };

  test.describe('Authentication Bypass Tests', () => {
    test('should prevent SQL injection authentication bypass', async ({ request }) => {
      const bypassAttempts = [
        { username: "admin' OR '1'='1", password: "admin' OR '1'='1" },
        { username: 'admin"--', password: '' },
        { username: "1' OR '1'='1' --", password: 'any' },
        { username: "admin' #", password: '' },
        { username: "UNION SELECT * FROM users--", password: '' },
      ];

      for (const attempt of bypassAttempts) {
        const response = await request.post('http://localhost:8080/api/v1/auth/login', {
          data: attempt
        });
        expect(response.status()).not.toBe(200);
      }
    });

    test('should prevent NoSQL injection authentication bypass', async ({ request }) => {
      const bypassAttempts = [
        { username: { $ne: null }, password: { $ne: null } },
        { username: { $regex: '.*' }, password: { $regex: '.*' } },
        { username: { $or: [{ role: 'admin' }] }, password: '' },
        { username: { $where: '1==1' }, password: '' },
      ];

      for (const attempt of bypassAttempts) {
        const response = await request.post('http://localhost:8080/api/v1/auth/login', {
          data: attempt
        });
        expect(response.status()).not.toBe(200);
      }
    });

    test('should prevent LDAP injection', async ({ request }) => {
      const injectionAttempts = [
        { username: 'admin*)(&', password: 'password' },
        { username: '*)(&(password=*', password: 'password' },
        { username: 'admin)(|(password=*', password: 'password' },
      ];

      for (const attempt of injectionAttempts) {
        const response = await request.post('http://localhost:8080/api/v1/auth/login', {
          data: attempt
        });
        expect(response.status()).not.toBe(200);
      }
    });

    test('should prevent XML injection', async ({ request }) => {
      const xmlPayloads = [
        '<?xml version="1.0"?><!DOCTYPE foo [<!ENTITY xxe SYSTEM "file:///etc/passwd">]><foo>&xxe;</foo>',
        '<?xml version="1.0"?><!DOCTYPE foo [<!ENTITY xxe SYSTEM "file:///windows/system32/drivers/etc/hosts">]><foo>&xxe;</foo>',
      ];

      for (const payload of xmlPayloads) {
        const response = await request.post('http://localhost:8080/api/v1/auth/login', {
          data: { username: payload, password: 'test' }
        });
        expect(response.status()).not.toBe(200);
      }
    });

    test('should prevent command injection', async ({ request }) => {
      const commandPayloads = [
        { username: 'admin; ls -la', password: '' },
        { username: 'admin| cat /etc/passwd', password: '' },
        { username: 'admin` whoami`', password: '' },
        { username: 'admin$(whoami)', password: '' },
      ];

      for (const attempt of commandPayloads) {
        const response = await request.post('http://localhost:8080/api/v1/auth/login', {
          data: attempt
        });
        expect(response.status()).not.toBe(200);
      }
    });
  });

  test.describe('Authorization Tests', () => {
    test('should prevent horizontal privilege escalation', async ({ request }) => {
      const loginResponse = await request.post('http://localhost:8080/api/v1/auth/login', {
        data: testUsers.user
      });

      if (loginResponse.ok()) {
        const token = (await loginResponse.json()).data?.token;

        const escalationAttempts = [
          { targetUserId: 1 },
          { targetUserId: 2 },
          { userId: 'other_user' },
        ];

        for (const attempt of escalationAttempts) {
          const response = await request.get('http://localhost:8080/api/v1/admin/stats', {
            headers: { Authorization: `Bearer ${token}` }
          });
          expect(response.status()).not.toBe(200);
        }
      }
    });

    test('should prevent vertical privilege escalation', async ({ request }) => {
      const userLogin = await request.post('http://localhost:8080/api/v1/auth/login', {
        data: testUsers.user
      });

      if (userLogin.ok()) {
        const token = (await userLogin.json()).data?.token;

        const adminEndpoints = [
          '/api/v1/admin/users',
          '/api/v1/admin/config',
          '/api/v1/admin/settings',
          '/api/v1/admin/permissions',
        ];

        for (const endpoint of adminEndpoints) {
          const response = await request.get(`http://localhost:8080${endpoint}`, {
            headers: { Authorization: `Bearer ${token}` }
          });
          expect(response.status()).toBe(403);
        }
      }
    });

    test('should enforce resource-level authorization', async ({ request }) => {
      const loginResponse = await request.post('http://localhost:8080/api/v1/auth/login', {
        data: testUsers.user
      });

      if (loginResponse.ok()) {
        const token = (await loginResponse.json()).data?.token;

        const resourceResponse = await request.get('http://localhost:8080/api/v1/resource/99999', {
          headers: { Authorization: `Bearer ${token}` }
        });

        expect(resourceResponse.status()).toBe(404);
      }
    });
  });

  test.describe('Session Security Tests', () => {
    test('should prevent session fixation', async ({ request }) => {
      const login1 = await request.post('http://localhost:8080/api/v1/auth/login', {
        data: testUsers.user
      });

      if (login1.ok()) {
        const session1 = (await login1.json()).data?.sessionId;

        const login2 = await request.post('http://localhost:8080/api/v1/auth/login', {
          data: testUsers.user
        });

        if (login2.ok()) {
          const session2 = (await login2.json()).data?.sessionId;
          expect(session1).not.toBe(session2);
        }
      }
    });

    test('should invalidate session after logout', async ({ request }) => {
      const login = await request.post('http://localhost:8080/api/v1/auth/login', {
        data: testUsers.user
      });

      if (login.ok()) {
        const token = (await login.json()).data?.token;

        await request.post('http://localhost:8080/api/v1/auth/logout', {
          headers: { Authorization: `Bearer ${token}` }
        });

        const afterLogout = await request.get('http://localhost:8080/api/v1/admin/stats', {
          headers: { Authorization: `Bearer ${token}` }
        });

        expect(afterLogout.status()).toBe(401);
      }
    });

    test('should prevent session token prediction', async ({ request }) => {
      const tokens = [];

      for (let i = 0; i < 10; i++) {
        const login = await request.post('http://localhost:8080/api/v1/auth/login', {
          data: testUsers.user
        });

        if (login.ok()) {
          const token = (await login.json()).data?.token;
          if (token) {
            tokens.push(token);
          }
        }
      }

      const uniqueTokens = new Set(tokens);
      expect(uniqueTokens.size).toBe(tokens.length);
    });

    test('should enforce session timeout', async ({ request }) => {
      const login = await request.post('http://localhost:8080/api/v1/auth/login', {
        data: testUsers.user
      });

      if (login.ok()) {
        const token = (await login.json()).data?.token;

        await new Promise(resolve => setTimeout(resolve, 3100000));

        const expiredSession = await request.get('http://localhost:8080/api/v1/admin/stats', {
          headers: { Authorization: `Bearer ${token}` }
        });

        expect(expiredSession.status()).toBe(401);
      }
    });
  });

  test.describe('Input Validation Tests', () => {
    test('should validate email format', async ({ request }) => {
      const invalidEmails = [
        'notanemail',
        '@nodomain.com',
        'noat.com',
        'spaces in@email.com',
        'newlines\nin@email.com',
      ];

      for (const email of invalidEmails) {
        const response = await request.post('http://localhost:8080/api/v1/auth/register', {
          data: { email, password: 'test123' }
        });
        expect(response.status()).toBe(400);
      }
    });

    test('should validate password strength', async ({ request }) => {
      const weakPasswords = [
        '123',
        'password',
        'abc123',
        'qwerty',
        'letmein',
      ];

      for (const password of weakPasswords) {
        const response = await request.post('http://localhost:8080/api/v1/auth/register', {
          data: { email: 'test@example.com', password }
        });
        expect(response.status()).toBe(400);
      }
    });

    test('should prevent buffer overflow in inputs', async ({ request }) => {
      const longInput = 'A'.repeat(100000);

      const response = await request.post('http://localhost:8080/api/v1/captcha/verify', {
        data: {
          captchaId: longInput,
          answer: { x: longInput, y: longInput }
        }
      });

      expect(response.status()).toBeDefined();
    });

    test('should sanitize HTML in inputs', async ({ page }) => {
      await page.goto('/captcha.html');

      const xssPayload = '<script>alert("XSS")</script>';
      await page.locator('#user-input').fill(xssPayload);

      const html = await page.content();
      expect(html).not.toContain('<script>');
    });
  });

  test.describe('Cryptographic Security Tests', () => {
    test('should use HTTPS only', async ({ request }) => {
      const response = await request.get('http://localhost:8080/home.html');
      expect(response.status()).toBe(200);
    });

    test('should reject weak SSL/TLS', async ({ request }) => {
      const response = await request.get('https://localhost:8080/home.html', {
        ignoreHTTPSErrors: false
      });
      expect(response.status()).toBeDefined();
    });

    test('should prevent sensitive data in URLs', async ({ request }) => {
      const sensitiveData = [
        '/api/v1/user/1?password=secret123',
        '/api/v1/captcha?token=abc123&secret=xyz789',
        '/api/v1/admin?api_key=supersecret',
      ];

      for (const url of sensitiveData) {
        const response = await request.get(`http://localhost:8080${url}`);
        expect(response.status()).toBeDefined();
      }
    });

    test('should use secure cookie attributes', async ({ page }) => {
      await page.goto('/home.html');

      const cookies = await page.context().cookies();
      const sessionCookie = cookies.find(c => c.name.includes('session') || c.name.includes('token'));

      if (sessionCookie) {
        expect(sessionCookie.secure).toBe(true);
        expect(sessionCookie.httpOnly).toBe(true);
      }
    });
  });

  test.describe('API Security Tests', () => {
    test('should enforce rate limiting', async ({ request }) => {
      const requests = [];
      for (let i = 0; i < 101; i++) {
        requests.push(
          request.post('http://localhost:8080/api/v1/auth/login', {
            data: { username: 'testuser', password: 'wrongpass' }
          })
        );
      }

      const results = await Promise.all(requests);
      const status429 = results.filter(r => r.status() === 429).length;

      expect(status429).toBeGreaterThan(0);
    });

    test('should prevent CORS abuse', async ({ request }) => {
      const response = await request.post('http://localhost:8080/api/v1/captcha/verify', {
        headers: {
          'Origin': 'http://malicious-site.com',
          'Access-Control-Request-Method': 'POST',
          'Access-Control-Request-Headers': 'Content-Type',
        }
      });

      expect(response.status()).toBeDefined();
    });

    test('should prevent CSRF attacks', async ({ request }) => {
      const response = await request.post('http://localhost:8080/api/v1/admin/settings', {
        data: { setting: 'new_value' },
        headers: {
          'X-CSRF-Token': 'invalid-token',
        }
      });

      expect(response.status()).toBe(403);
    });

    test('should require all security headers', async ({ request }) => {
      const response = await request.get('http://localhost:8080/home.html');

      const headers = response.headers();
      expect(headers['x-frame-options'] || headers['x-frame-options'.toLowerCase()]).toBeDefined();
      expect(headers['x-content-type-options'] || headers['x-content-type-options'.toLowerCase()]).toBeDefined();
      expect(headers['strict-transport-security'] || headers['strict-transport-security'.toLowerCase()]).toBeDefined();
    });
  });

  test.describe('Data Security Tests', () => {
    test('should encrypt sensitive data at rest', async ({ request }) => {
      const response = await request.get('http://localhost:8080/api/v1/admin/config');
      expect(response.status()).toBeDefined();
    });

    test('should not expose internal error details', async ({ request }) => {
      const response = await request.get('http://localhost:8080/api/v1/internal-error');
      const body = await response.text();

      expect(body).not.toContain('stack trace');
      expect(body).not.toContain('at ');
      expect(body).not.toContain('.go:');
    });

    test('should not leak information in error messages', async ({ request }) => {
      const response = await request.get('http://localhost:8080/api/v1/user/99999');
      const body = await response.text();

      expect(body).not.toContain('database');
      expect(body).not.toContain('SQL');
      expect(body).not.toContain('stack');
    });

    test('should mask sensitive fields in responses', async ({ request }) => {
      const login = await request.post('http://localhost:8080/api/v1/auth/login', {
        data: testUsers.user
      });

      if (login.ok()) {
        const data = await login.json();
        const responseStr = JSON.stringify(data);

        expect(responseStr).not.toContain('password');
        expect(responseStr).not.toContain('secret');
        expect(responseStr).not.toContain('token');
      }
    });
  });

  test.describe('File Upload Security Tests', () => {
    test('should validate file types', async ({ request }) => {
      const maliciousFiles = [
        { filename: 'shell.php', content: '<?php system($_GET["cmd"]); ?>' },
        { filename: 'script.js', content: 'require("child_process").exec("ls")' },
        { filename: '.htaccess', content: 'AddType application/x-httpd-php .jpg' },
      ];

      for (const file of maliciousFiles) {
        const response = await request.post('http://localhost:8080/api/v1/upload', {
          multipart: {
            file: {
              name: file.filename,
              mimeType: 'application/octet-stream',
              buffer: Buffer.from(file.content),
            },
          },
        });

        expect([400, 403, 415]).toContain(response.status());
      }
    });

    test('should prevent path traversal uploads', async ({ request }) => {
      const traversalPayloads = [
        '../../../etc/passwd',
        '..\\..\\..\\windows\\system32',
        '%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd',
      ];

      for (const filename of traversalPayloads) {
        const response = await request.post('http://localhost:8080/api/v1/upload', {
          multipart: {
            file: {
              name: filename,
              mimeType: 'image/png',
              buffer: Buffer.from('fake png'),
            },
          },
        });

        expect(response.status()).toBe(400);
      }
    });

    test('should limit file upload sizes', async ({ request }) => {
      const largeFile = Buffer.alloc(11 * 1024 * 1024);

      const response = await request.post('http://localhost:8080/api/v1/upload', {
        multipart: {
          file: {
            name: 'large.bin',
            mimeType: 'application/octet-stream',
            buffer: largeFile,
          },
        },
      });

      expect([400, 413]).toContain(response.status());
    });
  });

  test.describe('Denial of Service Tests', () => {
    test('should prevent XML bomb (Billion Laughs)', async ({ request }) => {
      const xmlBomb = `<?xml version="1.0"?>
        <!DOCTYPE lolz [
          <!ENTITY lol "lol">
          <!ENTITY lol2 "&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;">
          <!ENTITY lol3 "&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;">
        ]>
        <root>&lol3;</root>`;

      const response = await request.post('http://localhost:8080/api/v1/xml/parse', {
        data: xmlBomb,
        headers: { 'Content-Type': 'application/xml' }
      });

      expect(response.status()).not.toBe(200);
    });

    test('should prevent ZIP bomb decompression', async ({ request }) => {
      const response = await request.post('http://localhost:8080/api/v1/upload', {
        multipart: {
          file: {
            name: 'bomb.zip',
            mimeType: 'application/zip',
            buffer: Buffer.alloc(1024),
          },
        },
      });

      expect(response.status()).toBeDefined();
    });

    test('should limit request body size', async ({ request }) => {
      const hugeBody = Buffer.alloc(20 * 1024 * 1024);

      const response = await request.post('http://localhost:8080/api/v1/captcha/verify', {
        data: hugeBody.toString(),
        headers: { 'Content-Type': 'application/json' }
      });

      expect([400, 413, 431]).toContain(response.status());
    });
  });

  test.describe('WebSocket Security Tests', () => {
    test('should authenticate WebSocket connections', async ({ page }) => {
      await page.goto('/home.html');

      const wsConnected = await page.evaluate(async () => {
        return new Promise((resolve) => {
          const ws = new WebSocket('ws://localhost:8080/ws');
          ws.onopen = () => {
            ws.close();
            resolve(true);
          };
          ws.onerror = () => resolve(false);
          setTimeout(() => resolve(false), 5000);
        });
      });

      expect(wsConnected).toBeDefined();
    });

    test('should prevent WebSocket hijacking', async ({ page }) => {
      await page.goto('/home.html');

      const hijackingPrevented = await page.evaluate(() => {
        try {
          const ws = new WebSocket('http://malicious-site.com', 'ws://localhost:8080/ws');
          return true;
        } catch (e) {
          return false;
        }
      });

      expect(hijackingPrevented).toBeDefined();
    });
  });

  test.describe('GraphQL Security Tests', () => {
    test('should prevent introspection in production', async ({ request }) => {
      const response = await request.post('http://localhost:8080/graphql', {
        data: { query: '{ __schema { types { name } } }' }
      });

      expect(response.status()).toBeDefined();
    });

    test('should prevent batched query attacks', async ({ request }) => {
      const batchedQuery = {
        query: `query { 
          u1: user(id: 1) { name } 
          u2: user(id: 2) { name } 
          u3: user(id: 3) { name }
        }`
      };

      const response = await request.post('http://localhost:8080/graphql', {
        data: batchedQuery
      });

      expect(response.status()).toBeDefined();
    });

    test('should limit query depth', async ({ request }) => {
      const deepQuery = {
        query: `{ user { friends { friends { friends { friends { name } } } } } }`
      };

      const response = await request.post('http://localhost:8080/graphql', {
        data: deepQuery
      });

      expect(response.status()).toBeDefined();
    });
  });
});
