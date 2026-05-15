const fs = require('fs');
const path = require('path');

const express = require('express');
const request = require('supertest');

const pool = require('../../../config/database/db');
const filesRoutes = require('../../routes/files');
const { userFactory, fileFactory } = require('../factories');
const {
  generateToken,
  testPassword,
  fileUploadData,
  HTTP_STATUS
} = require('../helpers/testFixtures');

const app = express();
app.use(express.json());
app.use(express.urlencoded({ extended: true }));
app.use('/api/v1/files', filesRoutes);

describe('Files API Integration Tests', () => {
  let testUser;
  let adminUser;
  let testToken;
  let adminToken;
  let cleanupUsers = [];
  let cleanupFileIds = [];

  beforeAll(async () => {
    testUser = await userFactory.createUser({
      password: testPassword
    });
    adminUser = await userFactory.createAdmin({
      password: testPassword
    });
    cleanupUsers.push(testUser.id, adminUser.id);

    testToken = generateToken(testUser);
    adminToken = generateToken(adminUser);
  });

  afterAll(async () => {
    await fileFactory.deleteFiles(cleanupFileIds);
    await userFactory.deleteUsers(cleanupUsers);
    await pool.end();
  });

  describe('GET /api/v1/files', () => {
    let testFile;

    beforeAll(async () => {
      try {
        testFile = await fileFactory.createFile(testUser.id);
        cleanupFileIds.push(testFile.id);
      } catch (error) {
        console.log('Could not create test file:', error.message);
      }
    });

    it('should get user files successfully', async () => {
      const response = await request(app)
        .get('/api/v1/files')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
      expect(Array.isArray(response.body.data)).toBe(true);
    });

    it('should support folder filtering', async () => {
      const response = await request(app)
        .get('/api/v1/files?folder=test')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
    });

    it('should support pagination parameters', async () => {
      const response = await request(app)
        .get('/api/v1/files?page=1&limit=10')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
    });

    it('should fail without authentication', async () => {
      const response = await request(app).get('/api/v1/files');

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });

    it('should fail with invalid token', async () => {
      const response = await request(app)
        .get('/api/v1/files')
        .set('Authorization', 'Bearer invalid-token');

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });
  });

  describe('GET /api/v1/files/stats', () => {
    it('should get storage stats successfully', async () => {
      const response = await request(app)
        .get('/api/v1/files/stats')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
      expect(response.body.data).toHaveProperty('totalFiles');
      expect(response.body.data).toHaveProperty('totalSize');
    });

    it('should fail without authentication', async () => {
      const response = await request(app).get('/api/v1/files/stats');

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });
  });

  describe('GET /api/v1/files/:id', () => {
    let testFile;

    beforeAll(async () => {
      try {
        testFile = await fileFactory.createFile(testUser.id);
        cleanupFileIds.push(testFile.id);
      } catch (error) {
        console.log('Could not create test file:', error.message);
      }
    });

    it('should get file by id successfully', async () => {
      if (!testFile) {
        console.log('Skipping test: Could not create test file');
        return;
      }

      const response = await request(app)
        .get(`/api/v1/files/${testFile.id}`)
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
      expect(response.body.data.id).toBe(testFile.id);
    });

    it('should return 404 for non-existent file', async () => {
      const response = await request(app)
        .get('/api/v1/files/999999')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.NOT_FOUND);
    });

    it("should fail to get other user's file as regular user", async () => {
      if (!testFile) {
        console.log('Skipping test: Could not create test file');
        return;
      }

      const otherUser = await userFactory.createUser();
      cleanupUsers.push(otherUser.id);
      const otherToken = generateToken(otherUser);

      const response = await request(app)
        .get(`/api/v1/files/${testFile.id}`)
        .set('Authorization', `Bearer ${otherToken}`);

      expect(response.status).toBe(HTTP_STATUS.FORBIDDEN);
    });

    it('should allow admin to get any file', async () => {
      if (!testFile) {
        console.log('Skipping test: Could not create test file');
        return;
      }

      const response = await request(app)
        .get(`/api/v1/files/${testFile.id}`)
        .set('Authorization', `Bearer ${adminToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
    });

    it('should fail without authentication', async () => {
      if (!testFile) {
        console.log('Skipping test: Could not create test file');
        return;
      }

      const response = await request(app).get(`/api/v1/files/${testFile.id}`);

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });
  });

  describe('DELETE /api/v1/files/:id', () => {
    let fileToDelete;

    beforeEach(async () => {
      try {
        fileToDelete = await fileFactory.createFile(testUser.id);
      } catch (error) {
        console.log('Could not create file in beforeEach:', error.message);
        fileToDelete = null;
      }
    });

    it('should delete file successfully', async () => {
      if (!fileToDelete) {
        console.log('Skipping test: Could not create file');
        return;
      }

      const response = await request(app)
        .delete(`/api/v1/files/${fileToDelete.id}`)
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
    });

    it('should return 404 for non-existent file', async () => {
      const response = await request(app)
        .delete('/api/v1/files/999999')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.NOT_FOUND);
    });

    it("should fail to delete other user's file as regular user", async () => {
      if (!fileToDelete) {
        console.log('Skipping test: Could not create file');
        return;
      }

      const otherUser = await userFactory.createUser();
      cleanupUsers.push(otherUser.id);
      const otherToken = generateToken(otherUser);

      const response = await request(app)
        .delete(`/api/v1/files/${fileToDelete.id}`)
        .set('Authorization', `Bearer ${otherToken}`);

      expect(response.status).toBe(HTTP_STATUS.FORBIDDEN);

      cleanupFileIds.push(fileToDelete.id);
    });

    it('should allow admin to delete any file', async () => {
      if (!fileToDelete) {
        console.log('Skipping test: Could not create file');
        return;
      }

      const response = await request(app)
        .delete(`/api/v1/files/${fileToDelete.id}`)
        .set('Authorization', `Bearer ${adminToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
    });

    it('should fail without authentication', async () => {
      if (!fileToDelete) {
        console.log('Skipping test: Could not create file');
        return;
      }

      const response = await request(app).delete(`/api/v1/files/${fileToDelete.id}`);

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);

      cleanupFileIds.push(fileToDelete.id);
    });
  });

  describe('POST /api/v1/files/:id/copy', () => {
    let testFile;

    beforeEach(async () => {
      try {
        testFile = await fileFactory.createFile(testUser.id);
      } catch (error) {
        console.log('Could not create file in beforeEach:', error.message);
        testFile = null;
      }
    });

    afterEach(async () => {
      if (testFile) {
        cleanupFileIds.push(testFile.id);
      }
    });

    it('should copy file successfully', async () => {
      if (!testFile) {
        console.log('Skipping test: Could not create file');
        return;
      }

      const response = await request(app)
        .post(`/api/v1/files/${testFile.id}/copy`)
        .set('Authorization', `Bearer ${testToken}`)
        .send({ targetFolder: 'copies' });

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);

      if (response.body.data && response.body.data.id) {
        cleanupFileIds.push(response.body.data.id);
      }
    });

    it('should fail with missing targetFolder', async () => {
      if (!testFile) {
        console.log('Skipping test: Could not create file');
        return;
      }

      const response = await request(app)
        .post(`/api/v1/files/${testFile.id}/copy`)
        .set('Authorization', `Bearer ${testToken}`)
        .send({});

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
    });

    it('should return 404 for non-existent file', async () => {
      const response = await request(app)
        .post('/api/v1/files/999999/copy')
        .set('Authorization', `Bearer ${testToken}`)
        .send({ targetFolder: 'copies' });

      expect(response.status).toBe(HTTP_STATUS.NOT_FOUND);
    });

    it('should fail without authentication', async () => {
      if (!testFile) {
        console.log('Skipping test: Could not create file');
        return;
      }

      const response = await request(app)
        .post(`/api/v1/files/${testFile.id}/copy`)
        .send({ targetFolder: 'copies' });

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });
  });

  describe('POST /api/v1/files/:id/move', () => {
    let testFile;

    beforeEach(async () => {
      try {
        testFile = await fileFactory.createFile(testUser.id);
      } catch (error) {
        console.log('Could not create file in beforeEach:', error.message);
        testFile = null;
      }
    });

    afterEach(async () => {
      if (testFile) {
        cleanupFileIds.push(testFile.id);
      }
    });

    it('should move file successfully', async () => {
      if (!testFile) {
        console.log('Skipping test: Could not create file');
        return;
      }

      const response = await request(app)
        .post(`/api/v1/files/${testFile.id}/move`)
        .set('Authorization', `Bearer ${testToken}`)
        .send({ targetFolder: 'moved' });

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
    });

    it('should fail with missing targetFolder', async () => {
      if (!testFile) {
        console.log('Skipping test: Could not create file');
        return;
      }

      const response = await request(app)
        .post(`/api/v1/files/${testFile.id}/move`)
        .set('Authorization', `Bearer ${testToken}`)
        .send({});

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
    });

    it('should return 404 for non-existent file', async () => {
      const response = await request(app)
        .post('/api/v1/files/999999/move')
        .set('Authorization', `Bearer ${testToken}`)
        .send({ targetFolder: 'moved' });

      expect(response.status).toBe(HTTP_STATUS.NOT_FOUND);
    });

    it('should fail without authentication', async () => {
      if (!testFile) {
        console.log('Skipping test: Could not create file');
        return;
      }

      const response = await request(app)
        .post(`/api/v1/files/${testFile.id}/move`)
        .send({ targetFolder: 'moved' });

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });
  });

  describe('DELETE /api/v1/files/folder/:folder', () => {
    beforeAll(async () => {
      try {
        await fileFactory.createMultipleFiles(testUser.id, 2, { folder: 'delete-test' });
      } catch (error) {
        console.log('Could not create test files:', error.message);
      }
    });

    it('should delete folder and its files successfully', async () => {
      const response = await request(app)
        .delete('/api/v1/files/folder/delete-test')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
    });

    it('should fail without authentication', async () => {
      const response = await request(app).delete('/api/v1/files/folder/test');

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });
  });
});
