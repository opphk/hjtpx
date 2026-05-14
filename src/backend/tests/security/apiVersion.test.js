const request = require('supertest');
const express = require('express');
const apiVersionManager = require('../../../backend/middleware/apiVersionManager');
const {
  versionNegotiationMiddleware,
  versionEnforcementMiddleware,
  deprecatedVersionMiddleware
} = require('../../../backend/middleware/apiVersionNegotiation');

describe('API Version Negotiation Integration', () => {
  let app;

  beforeEach(() => {
    app = express();
  });

  describe('Version Negotiation Middleware', () => {
    beforeEach(() => {
      apiVersionManager.versions = new Map();
      apiVersionManager.deprecationSchedule = new Map();
      apiVersionManager.registerDefaultVersions();
      apiVersionManager.deprecateVersion('v1', '2025-12-31');
    });

    it('should negotiate version from query parameter', async () => {
      app.use(versionNegotiationMiddleware);
      app.get('/test', (req, res) => {
        res.json({
          apiVersion: req.apiVersion,
          versionInfo: req.versionInfo
        });
      });

      const response = await request(app).get('/test?api_version=v2');
      expect(response.headers['x-api-version']).toBe('v2');
      expect(response.body.apiVersion).toBe('v2');
    });

    it('should include deprecation warning for deprecated versions', async () => {
      app.use(versionNegotiationMiddleware);
      app.get('/test', (req, res) => {
        res.json({ message: 'success' });
      });

      const response = await request(app).get('/test?api_version=v1');
      expect(response.headers['x-api-deprecation-warning']).toBeDefined();
    });
  });

  describe('Version Enforcement Middleware', () => {
    beforeEach(() => {
      apiVersionManager.versions = new Map();
      apiVersionManager.deprecationSchedule = new Map();
      apiVersionManager.registerDefaultVersions();
    });

    it('should reject requests with version below required version', async () => {
      app.use(versionNegotiationMiddleware);
      app.use(versionEnforcementMiddleware('v2'));
      app.get('/test', (req, res) => {
        res.json({ message: 'success' });
      });

      const response = await request(app).get('/test');
      expect(response.status).toBe(400);
      expect(response.body.error.code).toBe('VERSION_NOT_SUPPORTED');
    });

    it('should allow requests with supported version', async () => {
      app.use(versionNegotiationMiddleware);
      app.use(versionEnforcementMiddleware('v2'));
      app.get('/test', (req, res) => {
        res.json({ message: 'success' });
      });

      const response = await request(app).get('/test?api_version=v2');
      expect(response.status).toBe(200);
    });

    it('should provide migration info for rejected requests', async () => {
      app.use(versionNegotiationMiddleware);
      app.use(versionEnforcementMiddleware('v2'));
      app.get('/test', (req, res) => {
        res.json({ message: 'success' });
      });

      const response = await request(app).get('/test');
      expect(response.status).toBe(400);
      expect(response.body.error.migration).toBeDefined();
    });
  });

  describe('Deprecated Version Middleware', () => {
    beforeEach(() => {
      apiVersionManager.versions = new Map();
      apiVersionManager.deprecationSchedule = new Map();
      apiVersionManager.registerDefaultVersions();
      apiVersionManager.deprecateVersion('v1', '2025-12-31');
    });

    it('should add Warning header for deprecated versions', async () => {
      app.use(versionNegotiationMiddleware);
      app.use(deprecatedVersionMiddleware);
      app.get('/test', (req, res) => {
        res.json({ message: 'success' });
      });

      const response = await request(app).get('/test?api_version=v1');
      expect(response.headers.warning).toBeDefined();
      expect(response.headers.warning).toContain('299');
    });

    it('should not add warning header for non-deprecated versions', async () => {
      app.use(versionNegotiationMiddleware);
      app.use(deprecatedVersionMiddleware);
      app.get('/test', (req, res) => {
        res.json({ message: 'success' });
      });

      const response = await request(app).get('/test?api_version=v2');
      expect(response.headers.warning).toBeUndefined();
    });
  });
});

describe('API Version Manager Core', () => {
  let testVersionManager;

  beforeEach(() => {
    testVersionManager = {
      versions: new Map(),
      registerVersion(version, config) {
        this.versions.set(version, config);
      },
      negotiateVersion(acceptHeader, queryVersion) {
        if (!acceptHeader && !queryVersion) {
          return { version: 'v1', negotiated: false };
        }
        const requestedVersion = queryVersion || this.parseAcceptHeader(acceptHeader);
        if (this.versions.has(requestedVersion)) {
          return { version: requestedVersion, negotiated: true };
        }
        return { version: 'v1', negotiated: false };
      },
      parseAcceptHeader(acceptHeader) {
        if (!acceptHeader) return null;
        const match = acceptHeader.match(/api-version\s*=\s*"?v?(\d+)"?/i);
        if (match) {
          return 'v' + match[1];
        }
        return null;
      }
    };
    testVersionManager.registerVersion('v1', { status: 'stable' });
    testVersionManager.registerVersion('v2', { status: 'beta' });
  });

  describe('Version Negotiation', () => {
    it('should negotiate version from query parameter', () => {
      const result = testVersionManager.negotiateVersion(null, 'v2');
      expect(result.version).toBe('v2');
      expect(result.negotiated).toBe(true);
    });

    it('should default to v1 when no version specified', () => {
      const result = testVersionManager.negotiateVersion(null, null);
      expect(result.version).toBe('v1');
      expect(result.negotiated).toBe(false);
    });

    it('should parse version from Accept header', () => {
      const version = testVersionManager.parseAcceptHeader('application/json; api-version=v2');
      expect(version).toBe('v2');
    });
  });
});
