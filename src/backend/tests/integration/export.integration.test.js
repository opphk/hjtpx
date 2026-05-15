const request = require('supertest');
const express = require('express');

const pool = require('../../../config/database/db');
const exportRoutes = require('../../routes/export');
const { userFactory } = require('../factories');
const {
  generateToken,
  testPassword,
  HTTP_STATUS
} = require('../helpers/testFixtures');

const app = express();
app.use(express.json());
app.use('/api/v1/export', exportRoutes);

describe('Export API Integration Tests', () => {
  let adminUser;
  let regularUser;
  let adminToken;
  let regularToken;
  let cleanupUsers = [];

  beforeAll(async () => {
    adminUser = await userFactory.createAdmin({ password: testPassword });
    regularUser = await userFactory.createUser({ password: testPassword });
    cleanupUsers.push(adminUser.id, regularUser.id);

    adminToken = generateToken(adminUser);
    regularToken = generateToken(regularUser);
  });

  afterAll(async () => {
    await userFactory.deleteUsers(cleanupUsers);
    await pool.end();
  });

  describe('GET /api/v1/export/:format', () => {
    it('should export users to CSV format successfully', async () => {
      const response = await request(app)
        .get('/api/v1/export/csv')
        .query({ table: 'users' })
        .set('Authorization', `Bearer ${adminToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.headers['content-type']).toContain('text/csv');
      expect(response.headers['content-disposition']).toContain('attachment');
    });

    it('should export users to JSON format successfully', async () => {
      const response = await request(app)
        .get('/api/v1/export/json')
        .query({ table: 'users' })
        .set('Authorization', `Bearer ${adminToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.headers['content-type']).toContain('application/json');
      expect(response.headers['content-disposition']).toContain('attachment');
    });

    it('should export users to Excel format successfully', async () => {
      const response = await request(app)
        .get('/api/v1/export/excel')
        .query({ table: 'users' })
        .set('Authorization', `Bearer ${adminToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.headers['content-type']).toContain('spreadsheetml');
      expect(response.headers['content-disposition']).toContain('attachment');
    });

    it('should export with custom filename', async () => {
      const response = await request(app)
        .get('/api/v1/export/json')
        .query({ table: 'users', filename: 'custom-users-export' })
        .set('Authorization', `Bearer ${adminToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.headers['content-disposition']).toContain('custom-users-export');
    });

    it('should export with specific fields', async () => {
      const response = await request(app)
        .get('/api/v1/export/csv')
        .query({ table: 'users', fields: 'id,email,name' })
        .set('Authorization', `Bearer ${adminToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
    });

    it('should fail without table parameter', async () => {
      const response = await request(app)
        .get('/api/v1/export/csv')
        .set('Authorization', `Bearer ${adminToken}`);

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
      expect(response.body.success).toBe(false);
    });

    it('should fail with unsupported format', async () => {
      const response = await request(app)
        .get('/api/v1/export/xml')
        .query({ table: 'users' })
        .set('Authorization', `Bearer ${adminToken}`);

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
      expect(response.body.success).toBe(false);
    });

    it('should fail with disallowed table', async () => {
      const response = await request(app)
        .get('/api/v1/export/json')
        .query({ table: 'passwords' })
        .set('Authorization', `Bearer ${adminToken}`);

      expect(response.status).toBe(500);
    });

    it('should fail without authentication', async () => {
      const response = await request(app)
        .get('/api/v1/export/csv')
        .query({ table: 'users' });

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });

    it('should fail with invalid token', async () => {
      const response = await request(app)
        .get('/api/v1/export/csv')
        .query({ table: 'users' })
        .set('Authorization', 'Bearer invalid-token');

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });
  });

  describe('POST /api/v1/export/batch', () => {
    it('should export multiple tables in batch as JSON', async () => {
      const response = await request(app)
        .post('/api/v1/export/batch')
        .set('Authorization', `Bearer ${adminToken}`)
        .send({
          tables: ['users', 'notifications'],
          format: 'json'
        });

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.headers['content-type']).toContain('application/json');
      expect(response.headers['content-disposition']).toContain('batch_export');
    });

    it('should export multiple tables in batch as CSV', async () => {
      const response = await request(app)
        .post('/api/v1/export/batch')
        .set('Authorization', `Bearer ${adminToken}`)
        .send({
          tables: ['users'],
          format: 'csv'
        });

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.headers['content-type']).toContain('text/csv');
    });

    it('should return partial results when some tables fail', async () => {
      const response = await request(app)
        .post('/api/v1/export/batch')
        .set('Authorization', `Bearer ${adminToken}`)
        .send({
          tables: ['users', 'invalid_table']
        });

      expect(response.status).toBe(HTTP_STATUS.OK);
    });

    it('should fail with empty tables array', async () => {
      const response = await request(app)
        .post('/api/v1/export/batch')
        .set('Authorization', `Bearer ${adminToken}`)
        .send({
          tables: []
        });

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
      expect(response.body.success).toBe(false);
    });

    it('should fail with invalid tables parameter', async () => {
      const response = await request(app)
        .post('/api/v1/export/batch')
        .set('Authorization', `Bearer ${adminToken}`)
        .send({
          tables: 'users'
        });

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
      expect(response.body.success).toBe(false);
    });

    it('should fail without tables parameter', async () => {
      const response = await request(app)
        .post('/api/v1/export/batch')
        .set('Authorization', `Bearer ${adminToken}`)
        .send({
          format: 'json'
        });

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
      expect(response.body.success).toBe(false);
    });

    it('should fail without authentication', async () => {
      const response = await request(app)
        .post('/api/v1/export/batch')
        .send({
          tables: ['users']
        });

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });

    it('should fail with invalid token', async () => {
      const response = await request(app)
        .post('/api/v1/export/batch')
        .set('Authorization', 'Bearer invalid-token')
        .send({
          tables: ['users']
        });

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });
  });

  describe('Permission and authorization', () => {
    it('should allow regular user to export users', async () => {
      const response = await request(app)
        .get('/api/v1/export/json')
        .query({ table: 'users' })
        .set('Authorization', `Bearer ${regularToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
    });

    it('should allow regular user to batch export', async () => {
      const response = await request(app)
        .post('/api/v1/export/batch')
        .set('Authorization', `Bearer ${regularToken}`)
        .send({
          tables: ['users']
        });

      expect(response.status).toBe(HTTP_STATUS.OK);
    });

    it('should export notifications table successfully', async () => {
      const response = await request(app)
        .get('/api/v1/export/json')
        .query({ table: 'notifications' })
        .set('Authorization', `Bearer ${adminToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
    });

    it('should export products table successfully', async () => {
      const response = await request(app)
        .get('/api/v1/export/json')
        .query({ table: 'products' })
        .set('Authorization', `Bearer ${adminToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
    });

    it('should export orders table successfully', async () => {
      const response = await request(app)
        .get('/api/v1/export/json')
        .query({ table: 'orders' })
        .set('Authorization', `Bearer ${adminToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
    });
  });

  describe('Edge cases and error handling', () => {
    it('should handle uppercase format correctly', async () => {
      const response = await request(app)
        .get('/api/v1/export/CSV')
        .query({ table: 'users' })
        .set('Authorization', `Bearer ${adminToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
    });

    it('should handle lowercase format correctly', async () => {
      const response = await request(app)
        .get('/api/v1/export/json')
        .query({ table: 'users' })
        .set('Authorization', `Bearer ${adminToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
    });

    it('should handle xlsx format as alias for excel', async () => {
      const response = await request(app)
        .get('/api/v1/export/xlsx')
        .query({ table: 'users' })
        .set('Authorization', `Bearer ${adminToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
    });

    it('should handle special characters in filename', async () => {
      const response = await request(app)
        .get('/api/v1/export/json')
        .query({ table: 'users', filename: 'export-with-special_chars' })
        .set('Authorization', `Bearer ${adminToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
    });

    it('should include Content-Length header', async () => {
      const response = await request(app)
        .get('/api/v1/export/json')
        .query({ table: 'users' })
        .set('Authorization', `Bearer ${adminToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.headers).toHaveProperty('content-length');
    });
  });
});
