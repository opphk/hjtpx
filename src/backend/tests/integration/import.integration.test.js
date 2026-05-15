const request = require('supertest');
const express = require('express');
const path = require('path');
const fs = require('fs');
const { v4: uuidv4 } = require('uuid');

const pool = require('../../../config/database/db');
const importRoutes = require('../../routes/import');
const { userFactory } = require('../factories');
const {
  generateToken,
  testPassword,
  HTTP_STATUS
} = require('../helpers/testFixtures');

const app = express();
app.use(express.json());
app.use(express.urlencoded({ extended: true }));
app.use('/api/v1/import', importRoutes);

describe('Import API Integration Tests', () => {
  let adminUser;
  let managerUser;
  let regularUser;
  let adminToken;
  let managerToken;
  let regularToken;
  let cleanupUsers = [];

  beforeAll(async () => {
    adminUser = await userFactory.createAdmin({ password: testPassword });
    managerUser = await userFactory.createUser({
      password: testPassword,
      role: 'manager',
      name: 'Manager User'
    });
    regularUser = await userFactory.createUser({ password: testPassword });
    cleanupUsers.push(adminUser.id, managerUser.id, regularUser.id);

    adminToken = generateToken(adminUser);
    managerToken = generateToken(managerUser);
    regularToken = generateToken(regularUser);
  });

  afterAll(async () => {
    await userFactory.deleteUsers(cleanupUsers);
    await pool.end();
  });

  describe('POST /api/v1/import/csv', () => {
    const testUploadDir = path.join(__dirname, '../../../uploads/imports');

    beforeAll(() => {
      if (!fs.existsSync(testUploadDir)) {
        fs.mkdirSync(testUploadDir, { recursive: true });
      }
    });

    afterEach(() => {
      const files = fs.readdirSync(testUploadDir);
      files.forEach(file => {
        fs.unlinkSync(path.join(testUploadDir, file));
      });
    });

    it('should import CSV file successfully as admin', async () => {
      const csvContent = 'email,name,role\ntest1@example.com,Test User 1,user\ntest2@example.com,Test User 2,user';
      const csvPath = path.join(testUploadDir, 'test-import.csv');
      fs.writeFileSync(csvPath, csvContent);

      const response = await request(app)
        .post('/api/v1/import/csv')
        .set('Authorization', `Bearer ${adminToken}`)
        .attach('file', csvPath)
        .field('table', 'users');

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
      expect(response.body.data).toHaveProperty('imported');
    });

    it('should import CSV file successfully as manager', async () => {
      const csvContent = 'email,name\nmanager_test1@example.com,Manager Test 1\nmanager_test2@example.com,Manager Test 2';
      const csvPath = path.join(testUploadDir, 'test-manager-import.csv');
      fs.writeFileSync(csvPath, csvContent);

      const response = await request(app)
        .post('/api/v1/import/csv')
        .set('Authorization', `Bearer ${managerToken}`)
        .attach('file', csvPath)
        .field('table', 'users');

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
    });

    it('should fail without file attachment', async () => {
      const response = await request(app)
        .post('/api/v1/import/csv')
        .set('Authorization', `Bearer ${adminToken}`)
        .field('table', 'users');

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
      expect(response.body.success).toBe(false);
    });

    it('should fail without table name', async () => {
      const csvContent = 'email,name\ntest@example.com,Test User';
      const csvPath = path.join(testUploadDir, 'test-no-table.csv');
      fs.writeFileSync(csvPath, csvContent);

      const response = await request(app)
        .post('/api/v1/import/csv')
        .set('Authorization', `Bearer ${adminToken}`)
        .attach('file', csvPath);

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
      expect(response.body.success).toBe(false);
    });

    it('should fail for regular user without permission', async () => {
      const csvContent = 'email,name\nregular@example.com,Regular User';
      const csvPath = path.join(testUploadDir, 'test-regular.csv');
      fs.writeFileSync(csvPath, csvContent);

      const response = await request(app)
        .post('/api/v1/import/csv')
        .set('Authorization', `Bearer ${regularToken}`)
        .attach('file', csvPath)
        .field('table', 'users');

      expect(response.status).toBe(HTTP_STATUS.FORBIDDEN);
    });

    it('should fail without authentication', async () => {
      const csvContent = 'email,name\ntest@example.com,Test User';
      const csvPath = path.join(testUploadDir, 'test-auth.csv');
      fs.writeFileSync(csvPath, csvContent);

      const response = await request(app)
        .post('/api/v1/import/csv')
        .attach('file', csvPath)
        .field('table', 'users');

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });

    it('should support custom delimiter', async () => {
      const csvContent = 'email|name\ndelimiter_test@example.com|Delimiter Test';
      const csvPath = path.join(testUploadDir, 'test-delimiter.csv');
      fs.writeFileSync(csvPath, csvContent);

      const response = await request(app)
        .post('/api/v1/import/csv')
        .set('Authorization', `Bearer ${adminToken}`)
        .attach('file', csvPath)
        .field('table', 'users')
        .field('delimiter', '|');

      expect(response.status).toBe(HTTP_STATUS.OK);
    });
  });

  describe('POST /api/v1/import/json', () => {
    const testUploadDir = path.join(__dirname, '../../../uploads/imports');

    beforeAll(() => {
      if (!fs.existsSync(testUploadDir)) {
        fs.mkdirSync(testUploadDir, { recursive: true });
      }
    });

    afterEach(() => {
      const files = fs.readdirSync(testUploadDir);
      files.forEach(file => {
        fs.unlinkSync(path.join(testUploadDir, file));
      });
    });

    it('should import JSON file successfully as admin', async () => {
      const jsonContent = JSON.stringify([
        { email: 'json_test1@example.com', name: 'JSON Test 1' },
        { email: 'json_test2@example.com', name: 'JSON Test 2' }
      ]);
      const jsonPath = path.join(testUploadDir, 'test-import.json');
      fs.writeFileSync(jsonPath, jsonContent);

      const response = await request(app)
        .post('/api/v1/import/json')
        .set('Authorization', `Bearer ${adminToken}`)
        .attach('file', jsonPath)
        .field('table', 'users');

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
      expect(response.body.data).toHaveProperty('imported');
    });

    it('should import JSON file successfully as manager', async () => {
      const jsonContent = JSON.stringify({
        data: [
          { email: 'manager_json1@example.com', name: 'Manager JSON 1' }
        ]
      });
      const jsonPath = path.join(testUploadDir, 'test-manager-json.json');
      fs.writeFileSync(jsonPath, jsonContent);

      const response = await request(app)
        .post('/api/v1/import/json')
        .set('Authorization', `Bearer ${managerToken}`)
        .attach('file', jsonPath)
        .field('table', 'users');

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
    });

    it('should fail without file attachment', async () => {
      const response = await request(app)
        .post('/api/v1/import/json')
        .set('Authorization', `Bearer ${adminToken}`)
        .field('table', 'users');

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
      expect(response.body.success).toBe(false);
    });

    it('should fail without table name', async () => {
      const jsonContent = JSON.stringify([{ email: 'test@example.com', name: 'Test' }]);
      const jsonPath = path.join(testUploadDir, 'test-no-table.json');
      fs.writeFileSync(jsonPath, jsonContent);

      const response = await request(app)
        .post('/api/v1/import/json')
        .set('Authorization', `Bearer ${adminToken}`)
        .attach('file', jsonPath);

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
      expect(response.body.success).toBe(false);
    });

    it('should fail for regular user without permission', async () => {
      const jsonContent = JSON.stringify([{ email: 'regular@example.com', name: 'Regular' }]);
      const jsonPath = path.join(testUploadDir, 'test-regular.json');
      fs.writeFileSync(jsonPath, jsonContent);

      const response = await request(app)
        .post('/api/v1/import/json')
        .set('Authorization', `Bearer ${regularToken}`)
        .attach('file', jsonPath)
        .field('table', 'users');

      expect(response.status).toBe(HTTP_STATUS.FORBIDDEN);
    });

    it('should fail without authentication', async () => {
      const jsonContent = JSON.stringify([{ email: 'test@example.com', name: 'Test' }]);
      const jsonPath = path.join(testUploadDir, 'test-auth.json');
      fs.writeFileSync(jsonPath, jsonContent);

      const response = await request(app)
        .post('/api/v1/import/json')
        .attach('file', jsonPath)
        .field('table', 'users');

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });
  });

  describe('POST /api/v1/import/preview', () => {
    const testUploadDir = path.join(__dirname, '../../../uploads/imports');

    beforeAll(() => {
      if (!fs.existsSync(testUploadDir)) {
        fs.mkdirSync(testUploadDir, { recursive: true });
      }
    });

    afterEach(() => {
      const files = fs.readdirSync(testUploadDir);
      files.forEach(file => {
        fs.unlinkSync(path.join(testUploadDir, file));
      });
    });

    it('should preview CSV file successfully', async () => {
      const csvContent = 'email,name\npreview1@example.com,Preview 1\npreview2@example.com,Preview 2\npreview3@example.com,Preview 3';
      const csvPath = path.join(testUploadDir, 'test-preview.csv');
      fs.writeFileSync(csvPath, csvContent);

      const response = await request(app)
        .post('/api/v1/import/preview')
        .set('Authorization', `Bearer ${adminToken}`)
        .attach('file', csvPath);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
      expect(response.body.data).toHaveProperty('preview');
      expect(response.body.data).toHaveProperty('columns');
      expect(response.body.data).toHaveProperty('totalRows');
      expect(Array.isArray(response.body.data.preview)).toBe(true);
      expect(Array.isArray(response.body.data.columns)).toBe(true);
    });

    it('should preview JSON file successfully', async () => {
      const jsonContent = JSON.stringify([
        { email: 'json1@example.com', name: 'JSON 1' },
        { email: 'json2@example.com', name: 'JSON 2' }
      ]);
      const jsonPath = path.join(testUploadDir, 'test-preview.json');
      fs.writeFileSync(jsonPath, jsonContent);

      const response = await request(app)
        .post('/api/v1/import/preview')
        .set('Authorization', `Bearer ${adminToken}`)
        .attach('file', jsonPath);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
      expect(response.body.data).toHaveProperty('preview');
      expect(response.body.data).toHaveProperty('columns');
    });

    it('should limit preview to 10 rows', async () => {
      const csvContent = 'email,name\nrow1@example.com,Row 1\nrow2@example.com,Row 2\nrow3@example.com,Row 3\nrow4@example.com,Row 4\nrow5@example.com,Row 5\nrow6@example.com,Row 6\nrow7@example.com,Row 7\nrow8@example.com,Row 8\nrow9@example.com,Row 9\nrow10@example.com,Row 10\nrow11@example.com,Row 11\nrow12@example.com,Row 12';
      const csvPath = path.join(testUploadDir, 'test-limit.csv');
      fs.writeFileSync(csvPath, csvContent);

      const response = await request(app)
        .post('/api/v1/import/preview')
        .set('Authorization', `Bearer ${adminToken}`)
        .attach('file', csvPath);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.data.preview.length).toBeLessThanOrEqual(10);
      expect(response.body.data.totalRows).toBe(12);
    });

    it('should fail without file attachment', async () => {
      const response = await request(app)
        .post('/api/v1/import/preview')
        .set('Authorization', `Bearer ${adminToken}`);

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
      expect(response.body.success).toBe(false);
    });

    it('should fail for regular user without permission', async () => {
      const csvContent = 'email,name\ntest@example.com,Test';
      const csvPath = path.join(testUploadDir, 'test-preview-auth.csv');
      fs.writeFileSync(csvPath, csvContent);

      const response = await request(app)
        .post('/api/v1/import/preview')
        .set('Authorization', `Bearer ${regularToken}`)
        .attach('file', csvPath);

      expect(response.status).toBe(HTTP_STATUS.FORBIDDEN);
    });

    it('should fail without authentication', async () => {
      const csvContent = 'email,name\ntest@example.com,Test';
      const csvPath = path.join(testUploadDir, 'test-preview-noauth.csv');
      fs.writeFileSync(csvPath, csvContent);

      const response = await request(app)
        .post('/api/v1/import/preview')
        .attach('file', csvPath);

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });
  });

  describe('Role-based access control', () => {
    it('should deny access to CSV import for regular user', async () => {
      const response = await request(app)
        .post('/api/v1/import/csv')
        .set('Authorization', `Bearer ${regularToken}`);

      expect(response.status).toBe(HTTP_STATUS.FORBIDDEN);
    });

    it('should deny access to JSON import for regular user', async () => {
      const response = await request(app)
        .post('/api/v1/import/json')
        .set('Authorization', `Bearer ${regularToken}`);

      expect(response.status).toBe(HTTP_STATUS.FORBIDDEN);
    });

    it('should deny access to preview for regular user', async () => {
      const response = await request(app)
        .post('/api/v1/import/preview')
        .set('Authorization', `Bearer ${regularToken}`);

      expect(response.status).toBe(HTTP_STATUS.FORBIDDEN);
    });
  });
});
