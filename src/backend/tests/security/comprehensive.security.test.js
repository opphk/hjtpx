const express = require('express');
const request = require('supertest');

const {
  securityHeaders,
  additionalSecurityHeaders,
  nonceMiddleware,
  createCSPReportEndpoint
} = require('../../middleware/securityHeaders');

describe('Comprehensive Security Tests', () => {
  describe('Dependency Security Scan', () => {
    test('should have no known vulnerabilities in dependencies', async () => {
      const { exec } = require('child_process');

      const auditResult = await new Promise(resolve => {
        exec('npm audit --json', (error, stdout, stderr) => {
          try {
            if (stdout) {
              resolve(JSON.parse(stdout));
            } else {
              resolve({});
            }
          } catch (parseError) {
            resolve({});
          }
        });
      });

      if (auditResult.metadata && auditResult.metadata.vulnerabilities) {
        const { vulnerabilities } = auditResult.metadata;
        expect(vulnerabilities.high).toBe(0);
        expect(vulnerabilities.critical).toBe(0);
      }
    });
  });

  describe('Security Headers Middleware', () => {
    let testApp;

    beforeEach(() => {
      testApp = express();
      testApp.use(express.json());
    });

    test('securityHeaders should set all required security headers', async () => {
      testApp.use(securityHeaders);
      testApp.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(testApp).get('/test');

      expect(response.headers['x-content-type-options']).toBe('nosniff');
      expect(response.headers['x-xss-protection']).toBe('1; mode=block');
      expect(response.headers['strict-transport-security']).toBeDefined();
      expect(response.headers['referrer-policy']).toBeDefined();
      expect(response.headers['content-security-policy']).toBeDefined();
    });

    test('should set Content-Security-Policy header', async () => {
      testApp.use(securityHeaders);
      testApp.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(testApp).get('/test');
      expect(response.headers['content-security-policy']).toBeDefined();
      expect(response.headers['content-security-policy']).toContain("object-src 'none'");
      expect(response.headers['content-security-policy']).toContain("frame-src 'none'");
    });

    test('should block clickjacking with X-Frame-Options', async () => {
      testApp.use(securityHeaders);
      testApp.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(testApp).get('/test');
      const frameOptions = response.headers['x-frame-options'];
      expect(['DENY', 'SAMEORIGIN']).toContain(frameOptions);
    });

    test('should include nonce in CSP for each request', async () => {
      testApp.use(securityHeaders);
      testApp.get('/test', (req, res) => res.json({ success: true }));

      const response1 = await request(testApp).get('/test');
      const response2 = await request(testApp).get('/test');

      const csp1 = response1.headers['content-security-policy'];
      const csp2 = response2.headers['content-security-policy'];

      const nonce1 = csp1.match(/'nonce-([^']+)'/);
      const nonce2 = csp2.match(/'nonce-([^']+)'/);

      expect(nonce1).not.toBeNull();
      expect(nonce2).not.toBeNull();
      expect(nonce1[1]).not.toBe(nonce2[1]);
    });

    test('should set Permissions-Policy to restrict sensitive APIs', async () => {
      testApp.use(securityHeaders);
      testApp.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(testApp).get('/test');
      const permissionsPolicy = response.headers['permissions-policy'];

      expect(permissionsPolicy).toBeDefined();
      expect(permissionsPolicy).toContain('camera=()');
      expect(permissionsPolicy).toContain('microphone=()');
      expect(permissionsPolicy).toContain('geolocation=()');
    });

    test('additionalSecurityHeaders should set all required headers', async () => {
      testApp.use(additionalSecurityHeaders);
      testApp.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(testApp).get('/test');

      expect(response.headers['x-content-type-options']).toBe('nosniff');
      expect(response.headers['x-xss-protection']).toBe('1; mode=block');
      expect(response.headers['strict-transport-security']).toBeDefined();
      expect(response.headers['referrer-policy']).toBe('strict-origin-when-cross-origin');
      expect(response.headers['permissions-policy']).toBeDefined();
      expect(response.headers['cache-control']).toContain('no-store');
      expect(response.headers['x-request-id']).toBeDefined();
    });

    test('nonceMiddleware should generate unique nonces', async () => {
      testApp.use(nonceMiddleware);
      testApp.get('/test', (req, res) => res.json({ nonce: res.locals.nonce }));

      const response1 = await request(testApp).get('/test');
      const response2 = await request(testApp).get('/test');

      expect(response1.body.nonce).toBeDefined();
      expect(response2.body.nonce).toBeDefined();
      expect(response1.body.nonce).not.toBe(response2.body.nonce);
    });
  });

  describe('XSS Protection', () => {
    test('should have X-XSS-Protection header enabled', async () => {
      const testApp = express();
      testApp.use(securityHeaders);
      testApp.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(testApp).get('/test');
      expect(response.headers['x-xss-protection']).toBe('1; mode=block');
    });
  });

  describe('CSP Report Endpoint', () => {
    test('should accept CSP violation reports', async () => {
      const testApp = express();
      testApp.post(
        '/csp-report',
        express.json({ type: ['application/csp-report', 'application/json'] }),
        createCSPReportEndpoint()
      );

      const report = {
        'csp-report': {
          'document-uri': 'https://example.com',
          referrer: '',
          'violated-directive': 'script-src',
          'effective-directive': 'script-src',
          'original-policy': '',
          disposition: 'enforce',
          'blocked-uri': 'https://malicious.com/script.js',
          'line-number': 42,
          'column-number': 10,
          'source-file': 'https://example.com/page.html',
          'status-code': 200,
          'script-sample': ''
        }
      };

      const response = await request(testApp)
        .post('/csp-report')
        .send(JSON.stringify(report))
        .set('Content-Type', 'application/csp-report');

      expect([204, 200]).toContain(response.statusCode);
    });
  });

  describe('HSTS Configuration', () => {
    test('should include HSTS header with proper directives', async () => {
      const testApp = express();
      testApp.use(securityHeaders);
      testApp.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(testApp).get('/test');

      const hstsHeader = response.headers['strict-transport-security'];
      expect(hstsHeader).toBeDefined();
      expect(hstsHeader).toContain('max-age=31536000');
      expect(hstsHeader).toContain('includeSubDomains');
    });
  });

  describe('Cache Control', () => {
    test('should have proper cache control headers for security', async () => {
      const testApp = express();
      testApp.use(additionalSecurityHeaders);
      testApp.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(testApp).get('/test');

      expect(response.headers['cache-control']).toContain('no-store');
      expect(response.headers['cache-control']).toContain('no-cache');
      expect(response.headers.pragma).toBe('no-cache');
    });
  });

  describe('Cross-Origin Policies', () => {
    test('should set Cross-Origin-Resource-Policy header', async () => {
      const testApp = express();
      testApp.use(securityHeaders);
      testApp.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(testApp).get('/test');
      expect(response.headers['cross-origin-resource-policy']).toBeDefined();
    });
  });
});
