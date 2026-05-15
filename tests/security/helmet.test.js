const request = require('supertest');
const express = require('express');

describe('Helmet Security Middleware', () => {
  let app;
  let helmetMiddleware;

  beforeAll(() => {
    const { helmetMiddleware: helmet } = require('../../src/backend/middleware/securityHeaders');
    helmetMiddleware = helmet;
  });

  beforeEach(() => {
    app = express();
    app.use(express.json());
  });

  describe('helmetMiddleware', () => {
    test('should set X-Content-Type-Options header', async () => {
      app.use(helmetMiddleware);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['x-content-type-options']).toBe('nosniff');
    });

    test('should set X-Frame-Options header', async () => {
      app.use(helmetMiddleware);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['x-frame-options']).toBe('DENY');
    });

    test('should set X-XSS-Protection header', async () => {
      app.use(helmetMiddleware);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      const xssProtection = response.headers['x-xss-protection'];
      expect(xssProtection).toBeDefined();
      expect(xssProtection).toBe('0');
    });

    test('should set Strict-Transport-Security header', async () => {
      app.use(helmetMiddleware);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['strict-transport-security']).toBe(
        'max-age=31536000; includeSubDomains; preload'
      );
    });

    test('should set Referrer-Policy header', async () => {
      app.use(helmetMiddleware);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['referrer-policy']).toBe('strict-origin-when-cross-origin');
    });

    test('should set Permissions-Policy header', async () => {
      app.use(helmetMiddleware);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      const permissionsPolicy = response.headers['permissions-policy'];
      expect(permissionsPolicy).toBeDefined();
      expect(typeof permissionsPolicy).toBe('string');
    });

    test('should set X-Download-Options header', async () => {
      app.use(helmetMiddleware);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['x-download-options']).toBe('noopen');
    });

    test('should set Cross-Origin-Resource-Policy header', async () => {
      app.use(helmetMiddleware);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['cross-origin-resource-policy']).toBe('same-origin');
    });

    test('should set Cross-Origin-Opener-Policy header', async () => {
      app.use(helmetMiddleware);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['cross-origin-opener-policy']).toBe('same-origin');
    });

    test('should set Cross-Origin-Embedder-Policy header', async () => {
      app.use(helmetMiddleware);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['cross-origin-embedder-policy']).toBe('require-corp');
    });

    test('should set X-Permitted-Cross-Domain-Policies header', async () => {
      app.use(helmetMiddleware);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['x-permitted-cross-domain-policies']).toBe('none');
    });

    test('should set Origin-Agent-Cluster header', async () => {
      app.use(helmetMiddleware);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['origin-agent-cluster']).toBe('?1');
    });

    test('should set Content-Security-Policy header', async () => {
      app.use(helmetMiddleware);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['content-security-policy']).toBeDefined();
      expect(response.headers['content-security-policy']).toContain("default-src 'self'");
    });

    test('should block inline scripts in CSP', async () => {
      app.use(helmetMiddleware);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      const cspHeader = response.headers['content-security-policy'];
      expect(cspHeader).toContain("'unsafe-inline'");
    });

    test('should block object-src in CSP', async () => {
      app.use(helmetMiddleware);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      const cspHeader = response.headers['content-security-policy'];
      expect(cspHeader).toContain("object-src 'none'");
    });

    test('should block frame-src in CSP', async () => {
      app.use(helmetMiddleware);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      const cspHeader = response.headers['content-security-policy'];
      expect(cspHeader).toContain("frame-src 'none'");
    });

    test('should restrict frame-ancestors in CSP', async () => {
      app.use(helmetMiddleware);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      const cspHeader = response.headers['content-security-policy'];
      expect(cspHeader).toContain("frame-ancestors 'none'");
    });

    test('should allow whitelisted CDN domains', async () => {
      app.use(helmetMiddleware);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      const cspHeader = response.headers['content-security-policy'];
      expect(cspHeader).toContain('cdn.jsdelivr.net');
      expect(cspHeader).toContain('cdnjs.cloudflare.com');
      expect(cspHeader).toContain('unpkg.com');
    });
  });

  describe('Enhanced Security Headers Integration', () => {
    test('should apply both helmet and custom security headers', async () => {
      const { securityHeaders } = require('../../src/backend/middleware/securityHeaders');
      
      app.use(helmetMiddleware);
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['x-content-type-options']).toBe('nosniff');
      expect(response.headers['content-security-policy']).toBeDefined();
      expect(response.headers['x-request-id']).toBeDefined();
    });
  });
});
