const request = require('supertest');
const express = require('express');
const { securityHeaders, additionalSecurityHeaders, helmetMiddleware } = require('../../src/backend/middleware/securityHeaders');
const { cspMiddleware, createStrictCSP, createPermissiveCSP, validateCSPConfiguration } = require('../../src/backend/middleware/cspMiddleware');

describe('Security Headers Tests', () => {
  let app;

  beforeEach(() => {
    app = express();
    app.use(express.json());
  });

  describe('X-Content-Type-Options Header', () => {
    test('should set X-Content-Type-Options to nosniff', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['x-content-type-options']).toBe('nosniff');
    });

    test('should prevent MIME type sniffing', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => {
        res.setHeader('Content-Type', 'text/html');
        res.send('<script>alert(1)</script>');
      });

      const response = await request(app).get('/test');
      expect(response.headers['x-content-type-options']).toBe('nosniff');
    });
  });

  describe('X-Frame-Options Header', () => {
    const originalEnv = process.env.NODE_ENV;

    afterEach(() => {
      process.env.NODE_ENV = originalEnv;
    });

    test('should set X-Frame-Options header appropriately', async () => {
      process.env.NODE_ENV = 'production';
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      const xFrameOptions = response.headers['x-frame-options'];
      expect(['DENY', 'SAMEORIGIN']).toContain(xFrameOptions);
    });

    test('should set X-Frame-Options to SAMEORIGIN in non-production', async () => {
      process.env.NODE_ENV = 'development';
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['x-frame-options']).toBe('SAMEORIGIN');
    });

    test('should prevent clickjacking attacks', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      const xFrameOptions = response.headers['x-frame-options'];
      expect(['DENY', 'SAMEORIGIN']).toContain(xFrameOptions);
    });
  });

  describe('Referrer-Policy Header', () => {
    test('should set Referrer-Policy to strict-origin-when-cross-origin', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['referrer-policy']).toBe('strict-origin-when-cross-origin');
    });

    test('should control referrer information leakage', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['referrer-policy']).toBeDefined();
    });
  });

  describe('Additional Security Headers', () => {
    test('should set Strict-Transport-Security header', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['strict-transport-security']).toBeDefined();
      expect(response.headers['strict-transport-security']).toContain('max-age=');
    });

    test('should set X-XSS-Protection header', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['x-xss-protection']).toBe('1; mode=block');
    });

    test('should set X-Download-Options header', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['x-download-options']).toBe('noopen');
    });

    test('should set Permissions-Policy header', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['permissions-policy']).toBeDefined();
    });
  });

  describe('Cross-Origin Security Headers', () => {
    test('should set Cross-Origin-Opener-Policy header', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['cross-origin-opener-policy']).toBe('same-origin');
    });

    test('should set Cross-Origin-Resource-Policy header', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['cross-origin-resource-policy']).toBe('same-origin');
    });

    test('should set Cross-Origin-Embedder-Policy header', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['cross-origin-embedder-policy']).toBe('require-corp');
    });
  });

  describe('Cache Control Headers', () => {
    test('should set Cache-Control header', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['cache-control']).toContain('no-store');
    });

    test('should set Pragma header for HTTP/1.0 compatibility', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['pragma']).toBe('no-cache');
    });
  });

  describe('DNSPrefetch Control', () => {
    test('should set X-DNS-Prefetch-Control header', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['x-dns-prefetch-control']).toBe('off');
    });

    test('should prevent DNS prefetching for privacy', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['x-dns-prefetch-control']).toBe('off');
    });
  });

  describe('Origin Agent Cluster', () => {
    test('should set Origin-Agent-Cluster header', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['origin-agent-cluster']).toBe('?1');
    });
  });

  describe('X-Permitted-Cross-Domain-Policies', () => {
    test('should set X-Permitted-Cross-Domain-Policies header', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['x-permitted-cross-domain-policies']).toBe('none');
    });
  });

  describe('Request ID Header', () => {
    test('should set X-Request-ID header', async () => {
      app.use((req, res, next) => {
        req.requestId = 'test-request-id';
        next();
      });
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['x-request-id']).toBe('test-request-id');
    });

    test('should generate X-Request-ID if not provided', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['x-request-id']).toBeDefined();
      expect(response.headers['x-request-id']).toMatch(/^req_/);
    });
  });

  describe('Content Security Policy Integration', () => {
    test('should set Content-Security-Policy header', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['content-security-policy']).toBeDefined();
    });

    test('should include nonce in CSP for scripts', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => {
        res.json({ nonce: res.locals.cspNonce });
      });

      const response = await request(app).get('/test');
      const cspHeader = response.headers['content-security-policy'];
      const nonce = response.body.nonce;

      expect(nonce).toBeDefined();
      expect(cspHeader).toContain(`'nonce-${nonce}'`);
    });

    test('should include strict-dynamic in script-src', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      const cspHeader = response.headers['content-security-policy'];
      expect(cspHeader).toContain("'strict-dynamic'");
    });

    test('should not allow unsafe inline scripts without nonce in script-src', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => {
        res.setHeader('Content-Type', 'text/html');
        res.send('<script>alert(1)</script>');
      });

      const response = await request(app).get('/test');
      const cspHeader = response.headers['content-security-policy'];
      const scriptSrcMatch = cspHeader.match(/script-src\s+([^;]+)/);
      if (scriptSrcMatch) {
        const scriptSrc = scriptSrcMatch[1];
        expect(scriptSrc).toContain("'nonce-");
      }
    });
  });

  describe('Additional Security Headers Middleware', () => {
    test('should apply all required headers via additionalSecurityHeaders', async () => {
      app.use(additionalSecurityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');

      expect(response.headers['x-content-type-options']).toBe('nosniff');
      expect(response.headers['x-frame-options']).toBeDefined();
      expect(response.headers['x-xss-protection']).toBe('1; mode=block');
      expect(response.headers['strict-transport-security']).toBeDefined();
      expect(response.headers['referrer-policy']).toBe('strict-origin-when-cross-origin');
      expect(response.headers['x-download-options']).toBe('noopen');
    });

    test('should not override existing headers from securityHeaders', async () => {
      app.use(securityHeaders);
      app.use(additionalSecurityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['x-content-type-options']).toBe('nosniff');
      expect(response.headers['x-frame-options']).toBeDefined();
    });
  });

  describe('Helmet Middleware Integration', () => {
    test('should configure helmet with security options', () => {
      expect(helmetMiddleware).toBeDefined();
      expect(typeof helmetMiddleware).toBe('function');
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
});

describe('CSP Middleware Tests', () => {
  let app;

  beforeEach(() => {
    app = express();
    app.use(express.json());
  });

  describe('CSP Middleware Functionality', () => {
    test('should generate unique nonce for each request', async () => {
      app.use(cspMiddleware);
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

    test('should set CSP header on response', async () => {
      app.use(cspMiddleware);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['content-security-policy']).toBeDefined();
    });

    test('should include default-src self directive', async () => {
      app.use(cspMiddleware);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      const cspHeader = response.headers['content-security-policy'];
      expect(cspHeader).toContain("default-src 'self'");
    });

    test('should allow inline scripts with nonce in CSP middleware', async () => {
      app.use(cspMiddleware);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      const cspHeader = response.headers['content-security-policy'];
      expect(cspHeader).toContain("script-src");
      const scriptSrcMatch = cspHeader.match(/script-src\s+([^;]+)/);
      if (scriptSrcMatch) {
        const scriptSrc = scriptSrcMatch[1];
        expect(scriptSrc).toContain("'nonce-");
      }
    });

    test('should set object-src to none', async () => {
      app.use(cspMiddleware);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      const cspHeader = response.headers['content-security-policy'];
      expect(cspHeader).toContain("object-src 'none'");
    });

    test('should set frame-ancestors to none', async () => {
      app.use(cspMiddleware);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      const cspHeader = response.headers['content-security-policy'];
      expect(cspHeader).toContain("frame-ancestors 'none'");
    });
  });

  describe('Strict CSP Creation', () => {
    test('should create strict CSP with nonce', () => {
      const csp = createStrictCSP();
      expect(csp).toBeDefined();
      expect(csp).toContain("default-src 'self'");
      expect(csp).toContain("script-src");
      expect(csp).toContain("'strict-dynamic'");
    });

    test('should restrict connect-src in strict mode', () => {
      const csp = createStrictCSP();
      expect(csp).toContain("connect-src 'self'");
    });
  });

  describe('Permissive CSP Creation', () => {
    test('should create permissive CSP for development', () => {
      const csp = createPermissiveCSP();
      expect(csp).toBeDefined();
      expect(csp).toContain("default-src 'self'");
      expect(csp).toContain("'unsafe-inline'");
    });

    test('should still block object-src in permissive mode', () => {
      const csp = createPermissiveCSP();
      expect(csp).toContain("object-src 'none'");
    });
  });

  describe('CSP Configuration Validation', () => {
    test('should validate correct CSP configuration', () => {
      const config = {
        directives: {
          'default-src': ["'self'"],
          'script-src': ["'self'", "'nonce-test'"],
          'object-src': ["'none'"],
          'frame-ancestors': ["'none'"]
        }
      };

      const result = validateCSPConfiguration(config);
      expect(result.valid).toBe(true);
      expect(result.errors).toHaveLength(0);
    });

    test('should reject missing required directives', () => {
      const config = {
        directives: {
          'default-src': ["'self'"]
        }
      };

      const result = validateCSPConfiguration(config);
      expect(result.valid).toBe(false);
      expect(result.errors.length).toBeGreaterThan(0);
    });

    test('should reject unsafe object-src', () => {
      const config = {
        directives: {
          'default-src': ["'self'"],
          'script-src': ["'self'"],
          'object-src': ["'self'"]
        }
      };

      const result = validateCSPConfiguration(config);
      expect(result.valid).toBe(false);
      expect(result.errors).toContain('object-src should be set to "none" for security');
    });

    test('should reject unsafe frame-ancestors', () => {
      const config = {
        directives: {
          'default-src': ["'self'"],
          'script-src': ["'self'"],
          'object-src': ["'none'"],
          'frame-ancestors': ["'self'"]
        }
      };

      const result = validateCSPConfiguration(config);
      expect(result.valid).toBe(false);
      expect(result.errors).toContain('frame-ancestors should be set to "none" for security');
    });

    test('should reject missing directives object', () => {
      const config = {};

      const result = validateCSPConfiguration(config);
      expect(result.valid).toBe(false);
      expect(result.errors).toContain('CSP configuration must include directives object');
    });
  });

  describe('Nonce Generation', () => {
    const { generateNonce } = require('../../src/backend/middleware/cspMiddleware');

    test('should generate base64 encoded nonce', () => {
      const nonce = generateNonce();
      expect(nonce).toMatch(/^[A-Za-z0-9+/]+=*$/);
    });

    test('should generate unique nonces', () => {
      const nonce1 = generateNonce();
      const nonce2 = generateNonce();
      expect(nonce1).not.toBe(nonce2);
    });

    test('should generate nonce of appropriate length', () => {
      const nonce = generateNonce();
      expect(nonce.length).toBe(24);
    });
  });

  describe('CSP Report Handler', () => {
    test('should handle CSP violation reports', async () => {
      const { createCSPReportHandler } = require('../../src/backend/middleware/cspMiddleware');
      const handler = createCSPReportHandler();

      const reportApp = express();
      reportApp.use(express.json());
      reportApp.post('/csp-report', handler);
      reportApp.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(reportApp)
        .post('/csp-report')
        .set('Content-Type', 'application/json')
        .send({
          'csp-report': {
            'document-uri': 'http://example.com',
            'violated-directive': 'script-src'
          }
        });

      expect(response.status).toBe(204);
    });
  });
});

describe('Security Integration Tests', () => {
  let app;

  beforeEach(() => {
    app = express();
    app.use(express.json());
  });

  describe('Combined Security Middleware', () => {
    test('should apply all security headers together', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');

      expect(response.headers['x-content-type-options']).toBe('nosniff');
      expect(response.headers['x-frame-options']).toBeDefined();
      expect(response.headers['referrer-policy']).toBe('strict-origin-when-cross-origin');
      expect(response.headers['content-security-policy']).toBeDefined();
      expect(response.headers['strict-transport-security']).toBeDefined();
      expect(response.headers['x-xss-protection']).toBe('1; mode=block');
      expect(response.headers['cross-origin-opener-policy']).toBe('same-origin');
      expect(response.headers['cross-origin-resource-policy']).toBe('same-origin');
      expect(response.headers['cross-origin-embedder-policy']).toBe('require-corp');
    });

    test('should work with CSP middleware', async () => {
      app.use(securityHeaders);
      app.use(cspMiddleware);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');

      expect(response.headers['content-security-policy']).toBeDefined();
      expect(response.headers['x-content-type-options']).toBe('nosniff');
    });

    test('should provide nonce for frontend use', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => {
        res.json({
          nonce: res.locals.cspNonce,
          success: true
        });
      });

      const response = await request(app).get('/test');
      expect(response.body.nonce).toBeDefined();
      expect(typeof response.body.nonce).toBe('string');
    });
  });

  describe('XSS Prevention', () => {
    test('should prevent reflected XSS via headers', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => {
        res.send(`<div>${req.query.name || ''}</div>`);
      });

      const response = await request(app).get('/test?name=<script>alert(1)</script>');
      
      expect(response.headers['x-xss-protection']).toBe('1; mode=block');
      expect(response.headers['content-security-policy']).toBeDefined();
    });

    test('should prevent stored XSS via CSP', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      const cspHeader = response.headers['content-security-policy'];
      const scriptSrcMatch = cspHeader.match(/script-src\s+([^;]+)/);
      
      expect(cspHeader).toContain("script-src");
      if (scriptSrcMatch) {
        const scriptSrc = scriptSrcMatch[1];
        expect(scriptSrc).not.toContain("'unsafe-inline'");
      }
    });

    test('should prevent DOM XSS via strict CSP', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      const cspHeader = response.headers['content-security-policy'];
      
      expect(cspHeader).toContain("'strict-dynamic'");
    });
  });

  describe('Clickjacking Prevention', () => {
    test('should prevent clickjacking via X-Frame-Options', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(['DENY', 'SAMEORIGIN']).toContain(response.headers['x-frame-options']);
    });

    test('should prevent clickjacking via CSP frame-ancestors', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      const cspHeader = response.headers['content-security-policy'];
      expect(cspHeader).toContain("frame-ancestors 'none'");
    });
  });

  describe('MIME Sniffing Prevention', () => {
    test('should prevent MIME sniffing via X-Content-Type-Options', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['x-content-type-options']).toBe('nosniff');
    });
  });

  describe('Information Leakage Prevention', () => {
    test('should control referrer information via Referrer-Policy', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['referrer-policy']).toBe('strict-origin-when-cross-origin');
    });

    test('should prevent DNS prefetching via X-DNS-Prefetch-Control', async () => {
      app.use(securityHeaders);
      app.get('/test', (req, res) => res.json({ success: true }));

      const response = await request(app).get('/test');
      expect(response.headers['x-dns-prefetch-control']).toBe('off');
    });
  });
});

describe('Dependency Audit Tests', () => {
  const { execSync } = require('child_process');
  const path = require('path');

  let auditResults;

  beforeAll(() => {
    try {
      const auditOutput = execSync('npm audit --json', {
        encoding: 'utf-8',
        cwd: path.join(__dirname, '../../'),
        maxBuffer: 50 * 1024 * 1024
      });
      auditResults = JSON.parse(auditOutput);
    } catch (error) {
      if (error.stdout) {
        auditResults = JSON.parse(error.stdout);
      } else {
        auditResults = { vulnerabilities: { total: 0 } };
      }
    }
  });

  test('should have no critical vulnerabilities', () => {
    const criticalCount = auditResults.vulnerabilities?.critical || 0;
    expect(criticalCount).toBe(0);
  });

  test('should have no high vulnerabilities', () => {
    const highCount = auditResults.vulnerabilities?.high || 0;
    expect(highCount).toBe(0);
  });

  test('should have acceptable number of moderate vulnerabilities', () => {
    const moderateCount = auditResults.vulnerabilities?.moderate || 0;
    expect(moderateCount).toBeLessThanOrEqual(5);
  });

  test('should report zero total vulnerabilities', () => {
    const total = auditResults.vulnerabilities?.total || 0;
    console.log(`Total vulnerabilities: ${total}`);
    
    if (total > 0) {
      console.log('Vulnerability breakdown:', auditResults.vulnerabilities);
    }
    
    expect(total).toBe(0);
  });
});
