const request = require('supertest');
const express = require('express');
const {
  versionNegotiationMiddleware
} = require('../../../backend/middleware/apiVersionNegotiation');

describe('API Version Coexistence', () => {
  let app;

  beforeEach(() => {
    app = express();
    app.use(express.json());
  });

  describe('Version Routing', () => {
    it('should route to v1 endpoints', async () => {
      const v1Routes = require('../../../backend/routes/v1');
      app.use('/api/v1', v1Routes);

      const response = await request(app).get('/api/v1');
      expect(response.status).toBe(200);
      expect(response.body.data.version).toBe('v1');
    });

    it('should route to v2 endpoints', async () => {
      const v2Routes = require('../../../backend/routes/v2');
      app.use('/api/v2', v2Routes);

      const response = await request(app).get('/api/v2');
      expect(response.status).toBe(200);
      expect(response.body.data.version).toBe('v2');
    });
  });

  describe('Health Endpoints', () => {
    it('should return health for v1', async () => {
      const v1Routes = require('../../../backend/routes/v1');
      app.use('/api/v1', v1Routes);

      const response = await request(app).get('/api/v1/health');
      expect(response.status).toBe(200);
      expect(response.body.success).toBe(true);
    });

    it('should return health for v2', async () => {
      const v2Routes = require('../../../backend/routes/v2');
      app.use('/api/v2', v2Routes);

      const response = await request(app).get('/api/v2/health');
      expect(response.status).toBe(200);
      expect(response.body.success).toBe(true);
    });
  });

  describe('Version Information', () => {
    it('should return version list for v1', async () => {
      const v1Routes = require('../../../backend/routes/v1');
      app.use('/api/v1', v1Routes);

      const response = await request(app).get('/api/v1/versions');
      expect(response.status).toBe(200);
      expect(response.body.data.supportedVersions).toBeDefined();
      expect(Array.isArray(response.body.data.supportedVersions)).toBe(true);
    });

    it('should return version list for v2', async () => {
      const v2Routes = require('../../../backend/routes/v2');
      app.use('/api/v2', v2Routes);

      const response = await request(app).get('/api/v2/versions');
      expect(response.status).toBe(200);
      expect(response.body.data.currentVersion).toBe('v2');
    });
  });

  describe('Migration Endpoint', () => {
    it('should return migration guide for v1 to v2', async () => {
      const v1Routes = require('../../../backend/routes/v1');
      app.use('/api/v1', v1Routes);

      const response = await request(app).get('/api/v1/versions/migration/v1/v2');
      expect(response.status).toBe(200);
      expect(response.body.data.migration).toBeDefined();
    });
  });

  describe('Response Format Differences', () => {
    it('should have different response structures for v1 and v2', async () => {
      const v1Routes = require('../../../backend/routes/v1');
      const v2Routes = require('../../../backend/routes/v2');

      app.use('/api/v1', v1Routes);
      app.use('/api/v2', v2Routes);

      const v1Response = await request(app).get('/api/v1');
      const v2Response = await request(app).get('/api/v2');

      expect(v1Response.body.data.version).toBe('v1');
      expect(v2Response.body.data.version).toBe('v2');
      expect(v1Response.body.data.description).toBe('HJTPX API Version 1');
      expect(v2Response.body.data.description).toBe('HJTPX API Version 2 (Beta)');
    });
  });

  describe('Version Negotiation with Routes', () => {
    it('should set version header for v1 routes', async () => {
      const v1Routes = require('../../../backend/routes/v1');
      app.use(versionNegotiationMiddleware);
      app.use('/api/v1', v1Routes);

      const response = await request(app).get('/api/v1');
      expect(response.headers['x-api-version']).toBeDefined();
    });

    it('should set version header for v2 routes', async () => {
      const v2Routes = require('../../../backend/routes/v2');
      app.use(versionNegotiationMiddleware);
      app.use('/api/v2', v2Routes);

      const response = await request(app).get('/api/v2');
      expect(response.headers['x-api-version']).toBeDefined();
    });
  });
});
