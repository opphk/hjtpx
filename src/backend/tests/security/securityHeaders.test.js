const express = require('express');
const request = require('supertest');

const {
  securityHeaders,
  additionalSecurityHeaders,
  nonceMiddleware
} = require('../../middleware/securityHeaders');

describe('Security Headers Middleware', () => {
  let app;

  beforeEach(() => {
    app = express();
    app.use(express.json());
  });

  describe('additionalSecurityHeaders', () => {
    test('should set X-Content-Type-Options header', async () => {
      app.use(additionalSecurityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['x-content-type-options']).toBe('nosniff');
    });

    test('should set X-Frame-Options header', async () => {
      app.use(additionalSecurityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(['DENY', 'SAMEORIGIN']).toContain(response.headers['x-frame-options']);
    });

    test('should set X-XSS-Protection header', async () => {
      app.use(additionalSecurityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['x-xss-protection']).toBe('1; mode=block');
    });

    test('should set Strict-Transport-Security header', async () => {
      app.use(additionalSecurityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['strict-transport-security']).toBe(
        'max-age=31536000; includeSubDomains; preload'
      );
    });

    test('should set Referrer-Policy header', async () => {
      app.use(additionalSecurityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['referrer-policy']).toBe('strict-origin-when-cross-origin');
    });

    test('should set Permissions-Policy header', async () => {
      app.use(additionalSecurityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['permissions-policy']).toBeDefined();
      expect(response.headers['permissions-policy']).toContain('accelerometer=()');
      expect(response.headers['permissions-policy']).toContain('camera=()');
      expect(response.headers['permissions-policy']).toContain('microphone=()');
    });

    test('should set X-Download-Options header', async () => {
      app.use(additionalSecurityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['x-download-options']).toBe('noopen');
    });

    test('should set Cache-Control header', async () => {
      app.use(additionalSecurityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['cache-control']).toBe(
        'no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0'
      );
    });

    test('should set X-Request-ID header', async () => {
      app.use((req, res, next) => {
        req.requestId = 'test-request-id';
        next();
      });
      app.use(additionalSecurityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['x-request-id']).toBe('test-request-id');
    });
  });

  describe('nonceMiddleware', () => {
    test('should generate nonce for each request', async () => {
      app.use(nonceMiddleware);
      app.get('/test', (req, res) => {
        res.json({ nonce: res.locals.nonce });
      });

      const response = await request(app).get('/test');
      expect(response.body.nonce).toBeDefined();
      expect(typeof response.body.nonce).toBe('string');
    });

    test('should generate unique nonces', async () => {
      app.use(nonceMiddleware);
      app.get('/test', (req, res) => {
        res.json({ nonce: res.locals.nonce });
      });

      const response1 = await request(app).get('/test');
      const response2 = await request(app).get('/test');

      expect(response1.body.nonce).not.toBe(response2.body.nonce);
    });
  });

  describe('securityHeaders with CSP', () => {
    test('should set Content-Security-Policy header', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['content-security-policy']).toBeDefined();
      expect(response.headers['content-security-policy']).toContain("default-src 'self'");
    });

    test('should set nonce in CSP script-src directive', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => {
        res.json({ cspNonce: res.locals.cspNonce });
      });

      const response = await request(app).get('/test');
      const cspHeader = response.headers['content-security-policy'];
      const nonce = response.body.cspNonce;

      expect(cspHeader).toContain(`'nonce-${nonce}'`);
    });

    test('should block frame-src in CSP', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      const cspHeader = response.headers['content-security-policy'];
      expect(cspHeader).toContain("frame-src 'none'");
    });

    test('should block object-src in CSP', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      const cspHeader = response.headers['content-security-policy'];
      expect(cspHeader).toContain("object-src 'none'");
    });

    test('should restrict base-uri in CSP', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      const cspHeader = response.headers['content-security-policy'];
      expect(cspHeader).toContain("base-uri 'self'");
    });

    test('should restrict form-action in CSP', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      const cspHeader = response.headers['content-security-policy'];
      expect(cspHeader).toContain("form-action 'self'");
    });
  });

  describe('HSTS Configuration', () => {
    test('should enforce HSTS in production', async () => {
      const originalEnv = process.env.NODE_ENV;
      process.env.NODE_ENV = 'production';

      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['strict-transport-security']).toBeDefined();
      expect(response.headers['strict-transport-security']).toContain('max-age=31536000');
      expect(response.headers['strict-transport-security']).toContain('includeSubDomains');
      expect(response.headers['strict-transport-security']).toContain('preload');

      process.env.NODE_ENV = originalEnv;
    });
  });

  describe('Cross-Origin Policies', () => {
    test('should set Cross-Origin-Resource-Policy header', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['cross-origin-resource-policy']).toBe('same-origin');
    });
  });
});
