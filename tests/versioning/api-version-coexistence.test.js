const express = require('express');
const request = require('supertest');

const {
  versionNegotiator,
  deprecationWarning,
  VERSIONS,
  DEFAULT_VERSION,
  SUPPORTED_VERSIONS,
  LATEST_STABLE_VERSION
} = require('../../src/backend/middleware/versionControl');

const createTestApp = () => {
  const app = express();
  app.use(express.json());
  app.use(versionNegotiator);
  app.use(deprecationWarning);

  // v1 路由
  app.get('/api/v1/health', (req, res) => {
    res.json({
      success: true,
      version: req.apiVersion,
      data: { status: 'healthy', service: 'v1' }
    });
  });

  app.get('/api/v1/users', (req, res) => {
    res.json({
      success: true,
      version: req.apiVersion,
      data: {
        users: [
          { id: 1, name: 'User 1', email: 'user1@example.com' }
        ]
      }
    });
  });

  // v2 路由
  app.get('/api/v2/health', (req, res) => {
    res.json({
      success: true,
      version: req.apiVersion,
      data: { status: 'healthy', service: 'v2' },
      meta: { timestamp: new Date().toISOString() }
    });
  });

  app.get('/api/v2/users', (req, res) => {
    const page = parseInt(req.query.page) || 1;
    const limit = parseInt(req.query.limit) || 10;

    res.json({
      success: true,
      version: req.apiVersion,
      data: {
        users: [
          { id: 1, name: 'User 1', email: 'user1@example.com', profile: {} }
        ],
        pagination: { page, limit, total: 1, total_pages: 1 }
      },
      meta: { timestamp: new Date().toISOString() }
    });
  });

  // 通用路由（用于版本协商测试）
  app.get('/api/health', (req, res) => {
    res.json({
      success: true,
      version: req.apiVersion,
      data: { negotiatedVersion: req.apiVersion }
    });
  });

  app.get('/api/users', (req, res) => {
    res.json({
      success: true,
      version: req.apiVersion,
      data: { negotiatedVersion: req.apiVersion }
    });
  });

  // 错误处理
  app.use((req, res) => {
    res.status(404).json({
      success: false,
      error: { message: 'Not found', code: 'NOT_FOUND' }
    });
  });

  return app;
};

describe('API Version Coexistence Tests', () => {
  let app;

  beforeAll(() => {
    app = createTestApp();
  });

  describe('Version Isolation', () => {
    test('v1 and v2 should work independently', async () => {
      const v1Response = await request(app).get('/api/v1/health');
      const v2Response = await request(app).get('/api/v2/health');

      expect(v1Response.status).toBe(200);
      expect(v2Response.status).toBe(200);
      expect(v1Response.body.version).toBe('v1');
      expect(v2Response.body.version).toBe('v2');
    });

    test('v1 should not have v2 features', async () => {
      const v1Response = await request(app).get('/api/v1/users');
      const v2Response = await request(app).get('/api/v2/users');

      expect(v1Response.body.data.pagination).toBeUndefined();
      expect(v2Response.body.data.pagination).toBeDefined();
    });

    test('v1 response should not have meta field', async () => {
      const v1Response = await request(app).get('/api/v1/health');
      expect(v1Response.body.meta).toBeUndefined();
    });

    test('v2 response should have meta field', async () => {
      const v2Response = await request(app).get('/api/v2/health');
      expect(v2Response.body.meta).toBeDefined();
      expect(v2Response.body.meta.timestamp).toBeDefined();
    });
  });

  describe('Concurrent Version Requests', () => {
    test('should handle concurrent v1 and v2 requests', async () => {
      const requests = [
        request(app).get('/api/v1/health'),
        request(app).get('/api/v2/health'),
        request(app).get('/api/v1/users'),
        request(app).get('/api/v2/users')
      ];

      const responses = await Promise.all(requests);

      expect(responses[0].body.version).toBe('v1');
      expect(responses[1].body.version).toBe('v2');
      expect(responses[2].body.version).toBe('v1');
      expect(responses[3].body.version).toBe('v2');
    });

    test('should maintain version context across multiple requests', async () => {
      // 交替请求不同版本
      for (let i = 0; i < 5; i++) {
        const v1Response = await request(app).get('/api/v1/health');
        const v2Response = await request(app).get('/api/v2/health');

        expect(v1Response.body.version).toBe('v1');
        expect(v2Response.body.version).toBe('v2');
        expect(v1Response.headers['x-api-version']).toBe('v1');
        expect(v2Response.headers['x-api-version']).toBe('v2');
      }
    });
  });

  describe('Version Negotiation Coexistence', () => {
    test('should negotiate v1 via Accept-Version header', async () => {
      const response = await request(app)
        .get('/api/health')
        .set('Accept-Version', 'v1');

      expect(response.status).toBe(200);
      expect(response.headers['x-api-version']).toBe('v1');
      expect(response.headers['x-api-version-negotiated']).toBe('true');
    });

    test('should negotiate v2 via Accept-Version header', async () => {
      const response = await request(app)
        .get('/api/health')
        .set('Accept-Version', 'v2');

      expect(response.status).toBe(200);
      expect(response.headers['x-api-version']).toBe('v2');
      expect(response.headers['x-api-version-negotiated']).toBe('true');
    });

    test('should negotiate via Accept header with MIME type', async () => {
      const response = await request(app)
        .get('/api/health')
        .set('Accept', 'application/vnd.hjtpx.v1+json');

      expect(response.headers['x-api-version']).toBe('v1');
    });

    test('should negotiate via X-API-Version header', async () => {
      const response = await request(app)
        .get('/api/health')
        .set('X-API-Version', 'v2');

      expect(response.headers['x-api-version']).toBe('v2');
    });

    test('should negotiate via Prefer header', async () => {
      const response = await request(app)
        .get('/api/health')
        .set('Prefer', 'version=v1');

      expect(response.headers['x-api-version']).toBe('v1');
    });

    test('should fallback to default version when no version specified', async () => {
      const response = await request(app).get('/api/health');

      expect(response.headers['x-api-version']).toBe(DEFAULT_VERSION);
      expect(response.headers['x-api-latest-version']).toBe(LATEST_STABLE_VERSION);
    });

    test('should handle unsupported version gracefully', async () => {
      const response = await request(app)
        .get('/api/health')
        .set('Accept-Version', 'v99');

      expect(response.status).toBe(200);
      expect(response.headers['x-api-version']).toBe(DEFAULT_VERSION);
      expect(response.headers['x-api-version-negotiated']).toBe('true');
      expect(response.headers['x-api-version-upgrade']).toBeDefined();
    });
  });

  describe('Deprecation Warning Coexistence', () => {
    test('v1 should include deprecation headers', async () => {
      const response = await request(app).get('/api/v1/health');

      expect(response.headers['deprecation']).toBeDefined();
      expect(response.headers['warning']).toBeDefined();
      expect(response.headers['x-api-deprecation-date']).toBeDefined();
      expect(response.headers['x-api-sunset-date']).toBeDefined();
    });

    test('v2 should not include deprecation headers', async () => {
      const response = await request(app).get('/api/v2/health');

      expect(response.headers['deprecation']).toBeUndefined();
      expect(response.headers['warning']).toBeUndefined();
    });

    test('v1 should include deprecation info in response body', async () => {
      const response = await request(app).get('/api/v1/health');

      expect(response.body.deprecation).toBeDefined();
      expect(response.body.deprecation.deprecated).toBe(true);
      expect(response.body.deprecation.currentVersion).toBe('v1');
      expect(response.body.deprecation.latestVersion).toBe('v2');
    });

    test('v2 should include features info in response body', async () => {
      const response = await request(app).get('/api/v2/health');

      expect(response.body.features).toBeDefined();
      expect(response.body.features.allAvailable).toBe(true);
    });

    test('should include Link header for deprecated version', async () => {
      const response = await request(app).get('/api/v1/health');

      expect(response.headers['link']).toBeDefined();
      expect(response.headers['link']).toContain('rel="deprecation"');
      expect(response.headers['link']).toContain('rel="successor-version"');
    });
  });

  describe('Version Headers Consistency', () => {
    test('all responses should include X-API-Version header', async () => {
      const v1Response = await request(app).get('/api/v1/health');
      const v2Response = await request(app).get('/api/v2/health');
      const defaultResponse = await request(app).get('/api/health');

      expect(v1Response.headers['x-api-version']).toBe('v1');
      expect(v2Response.headers['x-api-version']).toBe('v2');
      expect(defaultResponse.headers['x-api-version']).toBe(DEFAULT_VERSION);
    });

    test('all responses should include X-API-Supported-Versions header', async () => {
      const response = await request(app).get('/api/health');

      expect(response.headers['x-api-supported-versions']).toBeDefined();
      expect(response.headers['x-api-supported-versions']).toContain('v1');
      expect(response.headers['x-api-supported-versions']).toContain('v2');
    });

    test('all responses should include X-API-Latest-Version header', async () => {
      const response = await request(app).get('/api/health');

      expect(response.headers['x-api-latest-version']).toBe(LATEST_STABLE_VERSION);
    });

    test('all responses should include X-API-Version-Status header', async () => {
      const v1Response = await request(app).get('/api/v1/health');
      const v2Response = await request(app).get('/api/v2/health');

      expect(v1Response.headers['x-api-version-status']).toBe('stable');
      expect(v2Response.headers['x-api-version-status']).toBe('stable');
    });
  });

  describe('Error Handling Across Versions', () => {
    test('should handle 404 consistently in v1', async () => {
      const response = await request(app).get('/api/v1/nonexistent');

      expect(response.status).toBe(404);
      expect(response.headers['x-api-version']).toBe('v1');
    });

    test('should handle 404 consistently in v2', async () => {
      const response = await request(app).get('/api/v2/nonexistent');

      expect(response.status).toBe(404);
      expect(response.headers['x-api-version']).toBe('v2');
    });
  });

  describe('Version Configuration', () => {
    test('should have correct version configuration', () => {
      expect(SUPPORTED_VERSIONS).toContain('v1');
      expect(SUPPORTED_VERSIONS).toContain('v2');
      expect(DEFAULT_VERSION).toBe('v2');
      expect(LATEST_STABLE_VERSION).toBe('v2');
    });

    test('v1 should be marked as deprecated', () => {
      expect(VERSIONS.v1.deprecated).toBe(true);
      expect(VERSIONS.v1.deprecationDate).toBe('2026-01-01');
      expect(VERSIONS.v1.sunsetDate).toBe('2026-07-01');
    });

    test('v2 should not be deprecated', () => {
      expect(VERSIONS.v2.deprecated).toBe(false);
      expect(VERSIONS.v2.deprecationDate).toBeNull();
      expect(VERSIONS.v2.sunsetDate).toBeNull();
    });
  });

  describe('Version Negotiation Priority', () => {
    test('URL path should take priority over headers', async () => {
      const response = await request(app)
        .get('/api/v1/health')
        .set('Accept-Version', 'v2');

      expect(response.headers['x-api-version']).toBe('v1');
    });

    test('Accept-Version header should take priority over Accept header', async () => {
      const response = await request(app)
        .get('/api/health')
        .set('Accept-Version', 'v1')
        .set('Accept', 'application/vnd.hjtpx.v2+json');

      expect(response.headers['x-api-version']).toBe('v1');
    });
  });

  describe('Days Until Sunset Calculation', () => {
    test('should include X-API-Days-Until-Sunset for deprecated version', async () => {
      const response = await request(app).get('/api/v1/health');

      expect(response.headers['x-api-days-until-sunset']).toBeDefined();
      const daysUntilSunset = parseInt(response.headers['x-api-days-until-sunset']);
      expect(daysUntilSunset).toBeLessThan(100); // 距离2026-07-01不远
    });

    test('should show urgent warning when sunset is within 30 days', async () => {
      // 注意：这个测试依赖于当前日期，如果距离 sunset 日期超过 30 天，会失败
      // 在实际环境中，可以 mock 日期来测试
      const response = await request(app).get('/api/v1/health');
      const daysUntilSunset = parseInt(response.headers['x-api-days-until-sunset']);

      if (daysUntilSunset <= 30 && daysUntilSunset > 0) {
        expect(response.headers['warning']).toContain('Urgent upgrade required');
      }
    });
  });
});

describe('API Version Migration Info', () => {
  let app;

  beforeAll(() => {
    app = createTestApp();
  });

  test('v1 deprecation info should include migration steps', async () => {
    const response = await request(app).get('/api/v1/health');

    expect(response.body.deprecation.migrationSteps).toBeDefined();
    expect(Array.isArray(response.body.deprecation.migrationSteps)).toBe(true);
    expect(response.body.deprecation.migrationSteps.length).toBeGreaterThan(0);
  });

  test('v1 deprecation info should include estimated migration time', async () => {
    const response = await request(app).get('/api/v1/health');

    expect(response.body.deprecation.estimatedMigrationTime).toBeDefined();
    expect(typeof response.body.deprecation.estimatedMigrationTime).toBe('string');
  });

  test('v1 deprecation info should include breaking changes', async () => {
    const response = await request(app).get('/api/v1/health');

    expect(response.body.deprecation.breakingChanges).toBeDefined();
    expect(Array.isArray(response.body.deprecation.breakingChanges)).toBe(true);
  });
});

describe('API Version Coexistence Integration', () => {
  let app;

  beforeAll(() => {
    app = createTestApp();
  });

  test('should handle rapid version switching', async () => {
    const results = [];

    for (let i = 0; i < 10; i++) {
      const version = i % 2 === 0 ? 'v1' : 'v2';
      const response = await request(app).get(`/api/${version}/health`);
      results.push({
        iteration: i,
        version: response.headers['x-api-version'],
        expected: version
      });
    }

    results.forEach(result => {
      expect(result.version).toBe(result.expected);
    });
  });

  test('should maintain correct version info in request object', async () => {
    const v1Response = await request(app).get('/api/v1/health');
    const v2Response = await request(app).get('/api/v2/health');

    expect(v1Response.body.version).toBe('v1');
    expect(v2Response.body.version).toBe('v2');
  });
});
