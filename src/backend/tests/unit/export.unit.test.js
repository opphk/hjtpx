const request = require('supertest');
const express = require('express');

const mockExportService = {
  exportData: jest.fn(),
  EXPORT_FORMATS: ['csv', 'excel', 'json'],
  MAX_EXPORT_RECORDS: 10000
};

jest.mock('../../services/exportService', () => mockExportService);

const exportRoutes = require('../../routes/export');
const {
  generateToken,
  testPassword,
  HTTP_STATUS
} = require('../helpers/testFixtures');

const app = express();
app.use(express.json());
app.use('/api/v1/export', exportRoutes);

describe('Export API Unit Tests', () => {
  const mockAdminUser = {
    id: 1,
    email: 'admin@example.com',
    name: 'Admin User',
    role: 'admin'
  };

  const mockRegularUser = {
    id: 2,
    email: 'user@example.com',
    name: 'Regular User',
    role: 'user'
  };

  let adminToken;
  let regularToken;

  const mockData = [
    { id: 1, email: 'test1@example.com', name: 'Test 1' },
    { id: 2, email: 'test2@example.com', name: 'Test 2' }
  ];

  beforeEach(() => {
    jest.clearAllMocks();
    adminToken = generateToken(mockAdminUser);
    regularToken = generateToken(mockRegularUser);

    mockExportService.exportData.mockResolvedValue({
      filename: 'export.json',
      content: JSON.stringify(mockData),
      mimeType: 'application/json',
      size: 100
    });
  });

  describe('GET /api/v1/export/:format', () => {
    it('should export users to JSON format successfully', async () => {
      const response = await request(app)
        .get('/api/v1/export/json')
        .query({ table: 'users' })
        .set('Authorization', `Bearer ${adminToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.headers['content-type']).toContain('application/json');
      expect(response.headers['content-disposition']).toContain('attachment');
    });

    it('should export users to CSV format successfully', async () => {
      mockExportService.exportData.mockResolvedValue({
        filename: 'export.csv',
        content: 'id,email,name\n1,test1@example.com,Test 1',
        mimeType: 'text/csv',
        size: 50
      });

      const response = await request(app)
        .get('/api/v1/export/csv')
        .query({ table: 'users' })
        .set('Authorization', `Bearer ${adminToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.headers['content-type']).toContain('text/csv');
    });

    it('should export users to Excel format successfully', async () => {
      mockExportService.exportData.mockResolvedValue({
        filename: 'export.xlsx',
        content: Buffer.from([0, 1, 2, 3]),
        mimeType: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
        size: 4
      });

      const response = await request(app)
        .get('/api/v1/export/excel')
        .query({ table: 'users' })
        .set('Authorization', `Bearer ${adminToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.headers['content-type']).toContain('spreadsheetml');
    });

    it('should export with custom filename', async () => {
      const response = await request(app)
        .get('/api/v1/export/json')
        .query({ table: 'users', filename: 'custom-export' })
        .set('Authorization', `Bearer ${adminToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.headers['content-disposition']).toContain('custom-export');
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

    it('should allow regular user to export', async () => {
      const response = await request(app)
        .get('/api/v1/export/json')
        .query({ table: 'users' })
        .set('Authorization', `Bearer ${regularToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
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
      mockExportService.exportData.mockResolvedValue({
        filename: 'batch_export.csv',
        content: 'table,success,count\nusers,true,10',
        mimeType: 'text/csv',
        size: 50
      });

      const response = await request(app)
        .post('/api/v1/export/batch')
        .set('Authorization', `Bearer ${adminToken}`)
        .send({
          tables: ['users'],
          format: 'csv'
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

    it('should allow regular user to batch export', async () => {
      const response = await request(app)
        .post('/api/v1/export/batch')
        .set('Authorization', `Bearer ${regularToken}`)
        .send({
          tables: ['users']
        });

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
      mockExportService.exportData.mockResolvedValue({
        filename: 'export.xlsx',
        content: Buffer.from([0, 1, 2, 3]),
        mimeType: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
        size: 4
      });

      const response = await request(app)
        .get('/api/v1/export/xlsx')
        .query({ table: 'users' })
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
