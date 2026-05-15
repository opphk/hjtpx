const request = require('supertest');
const express = require('express');
const { securityHeaders } = require('../../src/backend/middleware/securityHeaders');

describe('Enhanced CSP Security Tests', () => {
  let app;

  beforeEach(() => {
    app = express();
    app.use(express.json());
  });

  describe('Enhanced CSP Directives', () => {
    test('should include strict-dynamic in script-src', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      const cspHeader = response.headers['content-security-policy'];
      expect(cspHeader).toContain("'strict-dynamic'");
    });

    test('should include worker-src directive', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      const cspHeader = response.headers['content-security-policy'];
      expect(cspHeader).toContain('worker-src');
    });

    test('should include manifest-src directive', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      const cspHeader = response.headers['content-security-policy'];
      expect(cspHeader).toContain('manifest-src');
    });

    test('should include child-src directive', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      const cspHeader = response.headers['content-security-policy'];
      expect(cspHeader).toContain('child-src');
    });

    test('should allow API endpoints in connect-src', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      const cspHeader = response.headers['content-security-policy'];
      expect(cspHeader).toContain('https://api.captchax.com');
      expect(cspHeader).toContain('https://api.hjtpx.com');
    });
  });

  describe('Enhanced Security Response Headers', () => {
    test('should set Cross-Origin-Embedder-Policy header', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['cross-origin-embedder-policy']).toBe('require-corp');
    });

    test('should set X-DNS-Prefetch-Control header', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['x-dns-prefetch-control']).toBe('off');
    });

    test('should set Origin-Agent-Cluster header', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['origin-agent-cluster']).toBe('?1');
    });

    test('should set X-Permitted-Cross-Domain-Policies header', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['x-permitted-cross-domain-policies']).toBe('none');
    });

    test('should set Pragma header for cache control', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['pragma']).toBe('no-cache');
    });
  });

  describe('CSP Report Only Mode', () => {
    const originalEnv = process.env.NODE_ENV;

    afterEach(() => {
      delete process.env.CSP_REPORT_ONLY;
      process.env.NODE_ENV = originalEnv;
    });

    test('should set CSP-Report-Only when enabled', async () => {
      process.env.CSP_REPORT_ONLY = 'true';
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['content-security-policy-report-only']).toBeDefined();
    });

    test('should not set CSP-Report-Only by default', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      const reportOnly = response.headers['content-security-policy-report-only'];
      expect(reportOnly === undefined || reportOnly === 'undefined').toBe(true);
    });
  });

  describe('Nonce Generation', () => {
    test('should generate unique nonce for each request', async () => {
      app.use(securityHeaders);
      app.get('/test1', (req, res) => {
        res.json({ nonce: res.locals.cspNonce });
      });
      app.get('/test2', (req, res) => {
        res.json({ nonce: res.locals.cspNonce });
      });

      const response1 = await request(app).get('/test1');
      const response2 = await request(app).get('/test2');

      expect(response1.body.nonce).toBeDefined();
      expect(response2.body.nonce).toBeDefined();
      expect(response1.body.nonce).not.toBe(response2.body.nonce);
    });

    test('should include nonce in CSP script-src directive', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => {
        res.json({ cspNonce: res.locals.cspNonce });
      });

      const response = await request(app).get('/test');
      const cspHeader = response.headers['content-security-policy'];
      const nonce = response.body.cspNonce;

      expect(cspHeader).toContain(`'nonce-${nonce}'`);
    });

    test('should generate base64 encoded nonce', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => {
        res.json({ nonce: res.locals.cspNonce });
      });

      const response = await request(app).get('/test');
      const nonce = response.body.nonce;
      
      expect(nonce).toMatch(/^[A-Za-z0-9+/]+=*$/);
    });
  });

  describe('Security Header Integration', () => {
    test('should apply security headers correctly', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');

      const requiredHeaders = [
        'x-content-type-options',
        'x-frame-options',
        'x-xss-protection',
        'strict-transport-security',
        'referrer-policy',
        'x-download-options',
        'cross-origin-opener-policy',
        'cross-origin-resource-policy',
        'cross-origin-embedder-policy',
        'x-permitted-cross-domain-policies',
        'origin-agent-cluster'
      ];

      requiredHeaders.forEach(header => {
        expect(response.headers[header]).toBeDefined();
      });
    });

    test('should apply securityHeaders and provide custom headers', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');

      expect(response.headers['content-security-policy']).toBeDefined();
      expect(response.headers['strict-transport-security']).toBeDefined();
      expect(response.headers['x-request-id']).toBeDefined();
    });
  });
});
