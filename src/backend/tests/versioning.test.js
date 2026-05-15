const request = require('supertest');

const app = require('../../../src/index');

describe('API Versioning', () => {
  describe('Version Negotiation', () => {
    test('should return v1 when explicitly requested via URL', async () => {
      const response = await request(app).get('/api/v1');
      expect(response.status).toBe(200);
      expect(response.body.data.version).toBe('v1');
    });

    test('should return v2 when explicitly requested via URL', async () => {
      const response = await request(app).get('/api/v2');
      expect(response.status).toBe(200);
      expect(response.body.data.version).toBe('v2');
    });

    test('should return deprecation warnings for v1', async () => {
      const response = await request(app).get('/api/v1/health');
      expect(response.headers.warning).toBeDefined();
      expect(response.headers['x-api-deprecation-date']).toBe('2026-01-01');
      expect(response.headers['x-api-sunset-date']).toBe('2026-07-01');
    });

    test('should not return deprecation warnings for v2', async () => {
      const response = await request(app).get('/api/v2/health');
      expect(response.headers.warning).toBeUndefined();
    });

    test('should include X-API-Version header in all responses', async () => {
      const responseV1 = await request(app).get('/api/v1/health');
      expect(responseV1.headers['x-api-version']).toBe('v1');

      const responseV2 = await request(app).get('/api/v2/health');
      expect(responseV2.headers['x-api-version']).toBe('v2');
    });
  });

  describe('v1 API', () => {
    test('should return v1 health check', async () => {
      const response = await request(app).get('/api/v1/health');
      expect(response.status).toBe(200);
      expect(response.body.success).toBe(true);
      expect(response.body.data.status).toBe('healthy');
    });

    test('should return v1 users list', async () => {
      const response = await request(app).get('/api/v1/users');
      expect(response.status).toBe(200);
      expect(response.body.success).toBe(true);
    });
  });

  describe('v2 API', () => {
    test('should return v2 health check with enhanced info', async () => {
      const response = await request(app).get('/api/v2/health');
      expect(response.status).toBe(200);
      expect(response.body.success).toBe(true);
      expect(response.body.data.services).toBeDefined();
    });

    test('should return v2 users with pagination', async () => {
      const response = await request(app).get('/api/v2/users');
      expect(response.status).toBe(200);
      expect(response.body.success).toBe(true);
      expect(response.body.data.pagination).toBeDefined();
      expect(response.body.meta).toBeDefined();
    });

    test('should return v2 user details with profile', async () => {
      const response = await request(app).get('/api/v2/users/1');
      expect(response.status).toBe(200);
      expect(response.body.success).toBe(true);
      expect(response.body.data.profile).toBeDefined();
    });
  });

  describe('Root endpoint', () => {
    test('should list all supported API versions', async () => {
      const response = await request(app).get('/');
      expect(response.status).toBe(200);
      expect(response.body.data.apiVersions).toBeDefined();
      expect(response.body.data.apiVersions.length).toBeGreaterThan(0);
      expect(response.body.data.apiVersions.some(v => v.version === 'v1')).toBe(true);
      expect(response.body.data.apiVersions.some(v => v.version === 'v2')).toBe(true);
    });
  });
});
