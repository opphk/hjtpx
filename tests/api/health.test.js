const request = require('supertest');
const app = require('../../src/index');

describe('Health API Tests', () => {
  let responseTime;

  describe('GET /api/v1/health', () => {
    test('should return healthy status', async () => {
      const startTime = Date.now();
      
      const response = await request(app)
        .get('/api/v1/health')
        .expect('Content-Type', /json/)
        .expect(200);

      responseTime = Date.now() - startTime;

      expect(response.body.success).toBe(true);
      expect(response.body.data).toHaveProperty('status', 'healthy');
      expect(response.body.data).toHaveProperty('service', 'HJTPX API');
      expect(response.body.data).toHaveProperty('version', '1.0.0');
      expect(response.body.data).toHaveProperty('timestamp');
      expect(response.body.data).toHaveProperty('uptime');
      expect(response.body.data).toHaveProperty('environment');
    });

    test('should respond within acceptable time', () => {
      expect(responseTime).toBeLessThan(1000);
    });

    test('should include required fields in response', async () => {
      const response = await request(app)
        .get('/api/v1/health')
        .expect(200);

      const requiredFields = ['status', 'service', 'version', 'timestamp', 'uptime', 'environment'];
      requiredFields.forEach(field => {
        expect(response.body.data).toHaveProperty(field);
      });
    });

    test('should return valid ISO timestamp', async () => {
      const response = await request(app)
        .get('/api/v1/health')
        .expect(200);

      const timestamp = new Date(response.body.data.timestamp);
      expect(timestamp).toBeInstanceOf(Date);
      expect(timestamp.getTime()).not.toBeNaN();
    });

    test('should return positive uptime value', async () => {
      const response = await request(app)
        .get('/api/v1/health')
        .expect(200);

      expect(response.body.data.uptime).toBeGreaterThan(0);
    });
  });

  describe('GET /api/v1/health/detailed', () => {
    test('should return detailed health status', async () => {
      const response = await request(app)
        .get('/api/v1/health/detailed')
        .expect('Content-Type', /json/);

      expect([200, 503]).toContain(response.status);
      expect(response.body).toHaveProperty('data');
      expect(response.body.data).toHaveProperty('checks');
      expect(response.body.data).toHaveProperty('responseTime');
    });

    test('should include all health check components', async () => {
      const response = await request(app)
        .get('/api/v1/health/detailed');

      if (response.status === 200) {
        expect(response.body.data.checks).toHaveProperty('database');
        expect(response.body.data.checks).toHaveProperty('redis');
        expect(response.body.data.checks).toHaveProperty('memory');
        expect(response.body.data.checks).toHaveProperty('cpu');
      }
    });

    test('should return database check status', async () => {
      const response = await request(app)
        .get('/api/v1/health/detailed');

      if (response.status === 200) {
        expect(['healthy', 'unhealthy', 'unavailable']).toContain(
          response.body.data.checks.database.status
        );
        expect(response.body.data.checks.database).toHaveProperty('message');
      }
    });

    test('should return redis check status', async () => {
      const response = await request(app)
        .get('/api/v1/health/detailed');

      if (response.status === 200) {
        expect(['healthy', 'unhealthy', 'unavailable']).toContain(
          response.body.data.checks.redis.status
        );
        expect(response.body.data.checks.redis).toHaveProperty('message');
      }
    });

    test('should return memory usage information', async () => {
      const response = await request(app)
        .get('/api/v1/health/detailed');

      if (response.status === 200) {
        expect(response.body.data.checks.memory).toHaveProperty('status');
        expect(response.body.data.checks.memory).toHaveProperty('message');
        expect(response.body.data.checks.memory).toHaveProperty('usage');
        expect(response.body.data.checks.memory.usage).toHaveProperty('used');
        expect(response.body.data.checks.memory.usage).toHaveProperty('total');
        expect(response.body.data.checks.memory.usage).toHaveProperty('unit', 'MB');
      }
    });

    test('should return response time', async () => {
      const response = await request(app)
        .get('/api/v1/health/detailed')
        .expect(200);

      expect(response.body.data.responseTime).toMatch(/^\d+ms$/);
    });

    test('should include version and environment info', async () => {
      const response = await request(app)
        .get('/api/v1/health/detailed')
        .expect(200);

      expect(response.body.data).toHaveProperty('service', 'HJTPX API');
      expect(response.body.data).toHaveProperty('version', '1.0.0');
      expect(['development', 'production', 'test']).toContain(
        response.body.data.environment
      );
    });
  });

  describe('Error Handling', () => {
    test('should return 404 for non-existent health check endpoint', async () => {
      const response = await request(app)
        .get('/api/v1/health/nonexistent')
        .expect(404);

      expect(response.body.success).toBe(false);
      expect(response.body.error).toHaveProperty('code', 'NOT_FOUND');
    });

    test('should handle invalid JSON gracefully', async () => {
      const response = await request(app)
        .post('/api/v1/users')
        .set('Content-Type', 'application/json')
        .send('invalid json')
        .expect(400);

      expect(response.body.success).toBe(false);
    });
  });

  describe('Response Format', () => {
    test('should return consistent success response format', async () => {
      const response = await request(app)
        .get('/api/v1/health')
        .expect(200);

      expect(response.body).toHaveProperty('success', true);
      expect(response.body).toHaveProperty('data');
      expect(response.body).toHaveProperty('timestamp');
    });

    test('should return consistent error response format', async () => {
      const response = await request(app)
        .get('/api/v1/health/nonexistent')
        .expect(404);

      expect(response.body).toHaveProperty('success', false);
      expect(response.body).toHaveProperty('error');
      expect(response.body.error).toHaveProperty('code');
      expect(response.body.error).toHaveProperty('message');
      expect(response.body).toHaveProperty('timestamp');
    });
  });

  describe('Security Headers', () => {
    test('should include X-Request-ID header', async () => {
      const response = await request(app)
        .get('/api/v1/health')
        .expect(200);

      expect(response.headers).toHaveProperty('x-request-id');
    });
  });

  describe('CORS', () => {
    test('should include CORS headers', async () => {
      const response = await request(app)
        .get('/api/v1/health')
        .expect(200);

      expect(response.headers).toHaveProperty('access-control-allow-origin');
    });

    test('should handle OPTIONS request', async () => {
      const response = await request(app)
        .options('/api/v1/health')
        .expect(204);

      expect(response.headers).toHaveProperty('access-control-allow-methods');
      expect(response.headers).toHaveProperty('access-control-allow-headers');
    });
  });
});
