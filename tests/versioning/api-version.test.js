const express = require('express');
const request = require('supertest');

const {
  apiVersionNegotiator,
  VERSIONS,
  DEFAULT_VERSION,
  LATEST_STABLE_VERSION
} = require('../../src/backend/middleware/apiVersionNegotiation');
const {
  deprecationWarning,
  getDeprecationInfo
} = require('../../src/backend/middleware/deprecationWarning');

const createTestApp = () => {
  const app = express();
  app.use(apiVersionNegotiator);
  app.use(deprecationWarning);

  app.get('/api/v1', (req, res) => {
    res.json({
      success: true,
      version: req.apiVersion,
      data: { message: 'v1 endpoint' }
    });
  });

  app.get('/api/v2', (req, res) => {
    res.json({
      success: true,
      version: req.apiVersion,
      data: { message: 'v2 endpoint' }
    });
  });

  app.get('/api/health', (req, res) => {
    res.json({
      success: true,
      version: req.apiVersion,
      data: { negotiatedVersion: req.apiVersion }
    });
  });

  return app;
};

describe('API Version Negotiation Middleware', () => {
  let app;

  beforeAll(() => {
    app = createTestApp();
  });

  describe('URL Path Version', () => {
    test('should extract version from /api/v1/ path', async () => {
      const response = await request(app).get('/api/v1');
      expect(response.status).toBe(200);
      expect(response.headers['x-api-version']).toBe('v1');
    });

    test('should extract version from /api/v2/ path', async () => {
      const response = await request(app).get('/api/v2');
      expect(response.status).toBe(200);
      expect(response.headers['x-api-version']).toBe('v2');
    });

    test('should set negotiation method to url for path-based version', async () => {
      const response = await request(app).get('/api/v1');
      expect(response.body.version).toBe('v1');
    });
  });

  describe('Accept-Version Header', () => {
    test('should negotiate version from Accept-Version header', async () => {
      const response = await request(app).get('/api/health').set('Accept-Version', 'v1');

      expect(response.status).toBe(200);
      expect(response.headers['x-api-version']).toBe('v1');
      expect(response.headers['x-api-version-negotiated']).toBe('true');
    });

    test('should negotiate v2 from Accept-Version header', async () => {
      const response = await request(app).get('/api/health').set('Accept-Version', 'v2');

      expect(response.status).toBe(200);
      expect(response.headers['x-api-version']).toBe('v2');
    });

    test('should fallback to default when Accept-Version is unsupported', async () => {
      const response = await request(app).get('/api/health').set('Accept-Version', 'v99');

      expect(response.status).toBe(200);
      expect(response.headers['x-api-version']).toBe(DEFAULT_VERSION);
      expect(response.headers['x-api-version-negotiated']).toBe('true');
      expect(response.headers['x-api-version-upgrade']).toBeDefined();
    });
  });

  describe('Version Downgrade Strategy', () => {
    test('should allow downgrade to older supported version', async () => {
      const response = await request(app).get('/api/health').set('Accept-Version', 'v1');

      expect(response.status).toBe(200);
      expect(response.headers['x-api-version']).toBe('v1');
      expect(VERSIONS.v1.deprecated).toBe(true);
    });

    test('should include version status in response headers', async () => {
      const response = await request(app).get('/api/v1');
      expect(response.headers['x-api-version-status']).toBe('stable');

      const responseV2 = await request(app).get('/api/v2');
      expect(responseV2.headers['x-api-version-status']).toBe('stable');
    });

    test('should list all supported versions', async () => {
      const response = await request(app).get('/api/health');
      expect(response.headers['x-api-supported-versions']).toBeDefined();
      expect(response.headers['x-api-supported-versions']).toContain('v1');
      expect(response.headers['x-api-supported-versions']).toContain('v2');
    });

    test('should indicate latest version', async () => {
      const response = await request(app).get('/api/health');
      expect(response.headers['x-api-latest-version']).toBe(LATEST_STABLE_VERSION);
    });
  });

  describe('Version Coexistence', () => {
    test('should serve both v1 and v2 endpoints simultaneously', async () => {
      const v1Promise = request(app).get('/api/v1');
      const v2Promise = request(app).get('/api/v2');

      const [v1Response, v2Response] = await Promise.all([v1Promise, v2Promise]);

      expect(v1Response.status).toBe(200);
      expect(v2Response.status).toBe(200);
      expect(v1Response.headers['x-api-version']).toBe('v1');
      expect(v2Response.headers['x-api-version']).toBe('v2');
    });

    test('should maintain version isolation between requests', async () => {
      const v1First = await request(app).get('/api/v1');
      const v2Middle = await request(app).get('/api/v2');
      const v1Second = await request(app).get('/api/v1');

      expect(v1First.headers['x-api-version']).toBe('v1');
      expect(v2Middle.headers['x-api-version']).toBe('v2');
      expect(v1Second.headers['x-api-version']).toBe('v1');
    });

    test('should handle concurrent requests with different versions', async () => {
      const requests = [
        request(app).get('/api/v1'),
        request(app).get('/api/v2'),
        request(app).get('/api/v1'),
        request(app).get('/api/v2')
      ];

      const responses = await Promise.all(requests);

      expect(responses[0].headers['x-api-version']).toBe('v1');
      expect(responses[1].headers['x-api-version']).toBe('v2');
      expect(responses[2].headers['x-api-version']).toBe('v1');
      expect(responses[3].headers['x-api-version']).toBe('v2');
    });
  });

  describe('Error Handling', () => {
    test('should handle missing version gracefully', async () => {
      const response = await request(app).get('/api/health');
      expect(response.status).toBe(200);
      expect(response.headers['x-api-version']).toBe(DEFAULT_VERSION);
    });

    test('should handle invalid version header gracefully', async () => {
      const response = await request(app)
        .get('/api/health')
        .set('Accept-Version', 'invalid-version');

      expect(response.status).toBe(200);
      expect(response.headers['x-api-version']).toBe(DEFAULT_VERSION);
    });

    test('should handle missing Accept-Version header', async () => {
      const response = await request(app).get('/api/health');
      expect(response.status).toBe(200);
      expect(response.headers['x-api-version']).toBe(DEFAULT_VERSION);
    });
  });

  describe('Response Headers', () => {
    test('should include X-API-Version header in all responses', async () => {
      const response = await request(app).get('/api/v1');
      expect(response.headers['x-api-version']).toBeDefined();
      expect(response.headers['x-api-version']).toBe('v1');
    });

    test('should include X-API-Supported-Versions header', async () => {
      const response = await request(app).get('/api/health');
      expect(response.headers['x-api-supported-versions']).toBeDefined();
      expect(response.headers['x-api-supported-versions']).toContain('v1');
      expect(response.headers['x-api-supported-versions']).toContain('v2');
    });

    test('should include X-API-Latest-Version header', async () => {
      const response = await request(app).get('/api/health');
      expect(response.headers['x-api-latest-version']).toBe(LATEST_STABLE_VERSION);
    });
  });

  describe('Version Information', () => {
    test('should expose version info in request object', async () => {
      const response = await request(app).get('/api/v1');
      expect(response.body.version).toBe('v1');
    });

    test('should provide version-specific route information', async () => {
      const response = await request(app).get('/api/health');
      expect(response.body.data).toBeDefined();
      expect(response.body.data.negotiatedVersion).toBeDefined();
    });
  });
});

describe('Deprecation Warning Middleware', () => {
  let app;

  beforeAll(() => {
    app = createTestApp();
  });

  describe('Deprecation Headers', () => {
    test('should add Deprecation header for deprecated versions', async () => {
      const response = await request(app).get('/api/v1');
      expect(response.headers.deprecation).toBeDefined();
      expect(response.headers.deprecation).toContain('v1');
    });

    test('should not add Deprecation header for current versions', async () => {
      const response = await request(app).get('/api/v2');
      expect(response.headers.deprecation).toBeUndefined();
    });

    test('should include sunset date for deprecated versions', async () => {
      const response = await request(app).get('/api/v1');
      expect(response.headers['x-api-sunset-date']).toBeDefined();
    });

    test('should include migration suggestions', async () => {
      const response = await request(app).get('/api/v1');
      expect(response.headers.link).toBeDefined();
      expect(response.headers.link).toContain('rel="deprecation"');
    });
  });

  describe('Response Body Deprecation Info', () => {
    test('should include deprecation info in v1 response body', async () => {
      const response = await request(app).get('/api/v1');
      expect(response.body.deprecation).toBeDefined();
      expect(response.body.deprecation.deprecated).toBe(true);
      expect(response.body.deprecation.currentVersion).toBe('v1');
      expect(response.body.deprecation.latestVersion).toBe('v2');
    });

    test('should include migration guide URL', async () => {
      const response = await request(app).get('/api/v1');
      expect(response.body.deprecation.migrationGuide).toBeDefined();
      expect(response.body.deprecation.migrationGuide).toContain('v1-migration-guide');
    });

    test('should not include deprecation info in v2 response', async () => {
      const response = await request(app).get('/api/v2');
      expect(response.body.deprecation).toBeUndefined();
    });

    test('should include breaking changes list', async () => {
      const response = await request(app).get('/api/v1');
      expect(response.body.deprecation.breakingChanges).toBeInstanceOf(Array);
      expect(response.body.deprecation.breakingChanges.length).toBeGreaterThan(0);
    });
  });

  describe('Deprecation Utility Functions', () => {
    test('getDeprecationInfo should return info for deprecated version', () => {
      const info = getDeprecationInfo('v1');
      expect(info).toBeDefined();
      expect(info.deprecated).toBe(true);
      expect(info.currentVersion).toBe('v1');
      expect(info.latestVersion).toBe('v2');
    });

    test('getDeprecationInfo should return null for non-deprecated version', () => {
      const info = getDeprecationInfo('v2');
      expect(info).toBeNull();
    });

    test('getDeprecationInfo should return null for unknown version', () => {
      const info = getDeprecationInfo('v99');
      expect(info).toBeNull();
    });
  });

  describe('Version-Specific Feature Flags', () => {
    test('v1 should have limited features', async () => {
      const response = await request(app).get('/api/v1');
      expect(response.body.deprecation.featuresEnabled).toBeDefined();
      expect(Array.isArray(response.body.deprecation.featuresEnabled)).toBe(true);
    });

    test('v2 should have all features enabled', async () => {
      const response = await request(app).get('/api/v2');
      expect(response.body.features).toBeDefined();
      expect(response.body.features.allAvailable).toBe(true);
    });
  });

  describe('Warning Header', () => {
    test('should include Warning header with message', async () => {
      const response = await request(app).get('/api/v1');
      expect(response.headers.warning).toBeDefined();
      expect(response.headers.warning).toContain('deprecated');
    });

    test('should not include Warning header for v2', async () => {
      const response = await request(app).get('/api/v2');
      expect(response.headers.warning).toBeUndefined();
    });
  });
});

describe('Version Coexistence Integration', () => {
  let app;

  beforeAll(() => {
    app = createTestApp();
  });

  describe('Unified Error Handling', () => {
    test('should handle 404 consistently across versions', async () => {
      const v1Response = await request(app).get('/api/v1/nonexistent');
      const v2Response = await request(app).get('/api/v2/nonexistent');

      expect(v1Response.status).toBe(404);
      expect(v2Response.status).toBe(404);
    });

    test('should include version info in error responses', async () => {
      const response = await request(app).get('/api/v1/nonexistent');
      expect(response.headers['x-api-version']).toBe('v1');
    });
  });

  describe('Version-Specific Routes', () => {
    test('v1 should have v1-specific endpoints', async () => {
      const response = await request(app).get('/api/v1');
      expect(response.status).toBe(200);
      expect(response.body.version).toBe('v1');
    });

    test('v2 should have v2-specific endpoints', async () => {
      const response = await request(app).get('/api/v2');
      expect(response.status).toBe(200);
      expect(response.body.version).toBe('v2');
    });

    test('both versions should work independently', async () => {
      const v1Response = await request(app).get('/api/v1');
      const v2Response = await request(app).get('/api/v2');

      expect(v1Response.status).toBe(200);
      expect(v2Response.status).toBe(200);
      expect(v1Response.body.version).not.toBe(v2Response.body.version);
    });
  });

  describe('Health Check Across Versions', () => {
    test('v1 health check should work', async () => {
      const response = await request(app).get('/api/v1').set('Accept-Version', 'v1');

      expect(response.status).toBe(200);
      expect(response.headers['x-api-version']).toBe('v1');
    });

    test('v2 health check should work', async () => {
      const response = await request(app).get('/api/v2').set('Accept-Version', 'v2');

      expect(response.status).toBe(200);
      expect(response.headers['x-api-version']).toBe('v2');
    });
  });
});
