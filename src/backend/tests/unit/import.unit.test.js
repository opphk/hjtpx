const request = require('supertest');
const express = require('express');
const path = require('path');
const fs = require('fs');

const mockImportService = {
  importFromFile: jest.fn(),
  parseCSV: jest.fn(),
  parseJSON: jest.fn(),
  ALLOWED_FILE_TYPES: ['csv', 'json'],
  MAX_FILE_SIZE: 10 * 1024 * 1024
};

jest.mock('../../services/importService', () => mockImportService);

const importRoutes = require('../../routes/import');
const {
  generateToken,
  testPassword,
  HTTP_STATUS
} = require('../helpers/testFixtures');

const app = express();
app.use(express.json());
app.use(express.urlencoded({ extended: true }));
app.use('/api/v1/import', importRoutes);

describe('Import API Unit Tests', () => {
  const mockAdminUser = {
    id: 1,
    email: 'admin@example.com',
    name: 'Admin User',
    role: 'admin'
  };

  const mockManagerUser = {
    id: 2,
    email: 'manager@example.com',
    name: 'Manager User',
    role: 'manager'
  };

  const mockRegularUser = {
    id: 3,
    email: 'user@example.com',
    name: 'Regular User',
    role: 'user'
  };

  let adminToken;
  let managerToken;
  let regularToken;

  beforeEach(() => {
    jest.clearAllMocks();
    adminToken = generateToken(mockAdminUser);
    managerToken = generateToken(mockManagerUser);
    regularToken = generateToken(mockRegularUser);

    mockImportService.importFromFile.mockResolvedValue({
      imported: 10,
      failed: 0,
      errors: [],
      totalProcessed: 10
    });
    mockImportService.parseCSV.mockResolvedValue([
      { email: 'test1@example.com', name: 'Test 1' },
      { email: 'test2@example.com', name: 'Test 2' }
    ]);
    mockImportService.parseJSON.mockResolvedValue([
      { email: 'json1@example.com', name: 'JSON 1' }
    ]);
  });

  describe('POST /api/v1/import/csv', () => {
    it('should import CSV file successfully as admin', async () => {
      mockImportService.importFromFile.mockResolvedValue({
        imported: 5,
        failed: 0,
        errors: []
      });

      const csvContent = 'email,name\ntest1@example.com,Test 1\ntest2@example.com,Test 2';
      
      const response = await request(app)
        .post('/api/v1/import/csv')
        .set('Authorization', `Bearer ${adminToken}`)
        .field('table', 'users')
        .attach('file', Buffer.from(csvContent), 'test.csv');

      expect([HTTP_STATUS.OK, HTTP_STATUS.CREATED]).toContain(response.status);
    });

    it('should import CSV file successfully as manager', async () => {
      mockImportService.importFromFile.mockResolvedValue({
        imported: 5,
        failed: 0,
        errors: []
      });

      const csvContent = 'email,name\ntest1@example.com,Test 1';
      
      const response = await request(app)
        .post('/api/v1/import/csv')
        .set('Authorization', `Bearer ${managerToken}`)
        .field('table', 'users')
        .attach('file', Buffer.from(csvContent), 'test.csv');

      expect([HTTP_STATUS.OK, HTTP_STATUS.CREATED]).toContain(response.status);
    });

    it('should fail without file', async () => {
      const response = await request(app)
        .post('/api/v1/import/csv')
        .set('Authorization', `Bearer ${adminToken}`)
        .field('table', 'users');

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
      expect(response.body.success).toBe(false);
    });

    it('should fail without table name', async () => {
      const csvContent = 'email,name\ntest@example.com,Test';
      
      const response = await request(app)
        .post('/api/v1/import/csv')
        .set('Authorization', `Bearer ${adminToken}`)
        .attach('file', Buffer.from(csvContent), 'test.csv');

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
      expect(response.body.success).toBe(false);
    });

    it('should deny access for regular user', async () => {
      const csvContent = 'email,name\ntest@example.com,Test';
      
      const response = await request(app)
        .post('/api/v1/import/csv')
        .set('Authorization', `Bearer ${regularToken}`)
        .field('table', 'users')
        .attach('file', Buffer.from(csvContent), 'test.csv');

      expect(response.status).toBe(HTTP_STATUS.FORBIDDEN);
    });

    it('should fail without authentication', async () => {
      const csvContent = 'email,name\ntest@example.com,Test';
      
      const response = await request(app)
        .post('/api/v1/import/csv')
        .field('table', 'users')
        .attach('file', Buffer.from(csvContent), 'test.csv');

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });
  });

  describe('POST /api/v1/import/json', () => {
    it('should import JSON file successfully as admin', async () => {
      mockImportService.importFromFile.mockResolvedValue({
        imported: 3,
        failed: 0,
        errors: []
      });

      const jsonContent = JSON.stringify([
        { email: 'test1@example.com', name: 'Test 1' }
      ]);
      
      const response = await request(app)
        .post('/api/v1/import/json')
        .set('Authorization', `Bearer ${adminToken}`)
        .field('table', 'users')
        .attach('file', Buffer.from(jsonContent), 'test.json');

      expect([HTTP_STATUS.OK, HTTP_STATUS.CREATED]).toContain(response.status);
    });

    it('should import JSON file successfully as manager', async () => {
      mockImportService.importFromFile.mockResolvedValue({
        imported: 3,
        failed: 0,
        errors: []
      });

      const jsonContent = JSON.stringify([
        { email: 'test1@example.com', name: 'Test 1' }
      ]);
      
      const response = await request(app)
        .post('/api/v1/import/json')
        .set('Authorization', `Bearer ${managerToken}`)
        .field('table', 'users')
        .attach('file', Buffer.from(jsonContent), 'test.json');

      expect([HTTP_STATUS.OK, HTTP_STATUS.CREATED]).toContain(response.status);
    });

    it('should fail without file', async () => {
      const response = await request(app)
        .post('/api/v1/import/json')
        .set('Authorization', `Bearer ${adminToken}`)
        .field('table', 'users');

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
      expect(response.body.success).toBe(false);
    });

    it('should fail without table name', async () => {
      const jsonContent = JSON.stringify([{ email: 'test@example.com', name: 'Test' }]);
      
      const response = await request(app)
        .post('/api/v1/import/json')
        .set('Authorization', `Bearer ${adminToken}`)
        .attach('file', Buffer.from(jsonContent), 'test.json');

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
      expect(response.body.success).toBe(false);
    });

    it('should deny access for regular user', async () => {
      const jsonContent = JSON.stringify([{ email: 'test@example.com', name: 'Test' }]);
      
      const response = await request(app)
        .post('/api/v1/import/json')
        .set('Authorization', `Bearer ${regularToken}`)
        .field('table', 'users')
        .attach('file', Buffer.from(jsonContent), 'test.json');

      expect(response.status).toBe(HTTP_STATUS.FORBIDDEN);
    });
  });

  describe('POST /api/v1/import/preview', () => {
    it('should preview CSV file successfully', async () => {
      mockImportService.parseCSV.mockResolvedValue([
        { email: 'preview1@example.com', name: 'Preview 1' },
        { email: 'preview2@example.com', name: 'Preview 2' },
        { email: 'preview3@example.com', name: 'Preview 3' }
      ]);

      const csvContent = 'email,name\npreview1@example.com,Preview 1\npreview2@example.com,Preview 2\npreview3@example.com,Preview 3';
      
      const response = await request(app)
        .post('/api/v1/import/preview')
        .set('Authorization', `Bearer ${adminToken}`)
        .attach('file', Buffer.from(csvContent), 'test.csv');

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
      expect(response.body.data).toHaveProperty('preview');
      expect(response.body.data).toHaveProperty('columns');
      expect(response.body.data).toHaveProperty('totalRows');
    });

    it('should preview JSON file successfully', async () => {
      mockImportService.parseJSON.mockResolvedValue([
        { email: 'json1@example.com', name: 'JSON 1' }
      ]);

      const jsonContent = JSON.stringify([
        { email: 'json1@example.com', name: 'JSON 1' }
      ]);
      
      const response = await request(app)
        .post('/api/v1/import/preview')
        .set('Authorization', `Bearer ${adminToken}`)
        .attach('file', Buffer.from(jsonContent), 'test.json');

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
      expect(response.body.data).toHaveProperty('preview');
    });

    it('should limit preview to 10 rows', async () => {
      const csvContent = 'email,name\n' + Array.from({ length: 15 }, (_, i) => 
        `row${i+1}@example.com,Row ${i+1}`
      ).join('\n');
      
      const response = await request(app)
        .post('/api/v1/import/preview')
        .set('Authorization', `Bearer ${adminToken}`)
        .attach('file', Buffer.from(csvContent), 'test.csv');

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.data.preview.length).toBeLessThanOrEqual(10);
      expect(response.body.data.totalRows).toBe(15);
    });

    it('should fail without file', async () => {
      const response = await request(app)
        .post('/api/v1/import/preview')
        .set('Authorization', `Bearer ${adminToken}`);

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
      expect(response.body.success).toBe(false);
    });

    it('should deny access for regular user', async () => {
      const csvContent = 'email,name\ntest@example.com,Test';
      
      const response = await request(app)
        .post('/api/v1/import/preview')
        .set('Authorization', `Bearer ${regularToken}`)
        .attach('file', Buffer.from(csvContent), 'test.csv');

      expect(response.status).toBe(HTTP_STATUS.FORBIDDEN);
    });
  });
});
