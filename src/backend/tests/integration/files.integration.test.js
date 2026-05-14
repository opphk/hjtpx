const request = require('supertest');
const express = require('express');
const bcrypt = require('bcryptjs');
const jwt = require('jsonwebtoken');
const path = require('path');
const fs = require('fs');

const pool = require('../../../config/database/db');
const filesRoutes = require('../../routes/files');

const app = express();
app.use(express.json());
app.use('/api/files', filesRoutes);

const JWT_SECRET = process.env.JWT_SECRET || 'hjtpx-secret-key-change-in-production';

describe('Files API Integration Tests', () => {
  let testUser;
  let testToken;
  let testFileId;

  beforeAll(async () => {
    const hashedPassword = await bcrypt.hash('TestPassword123!', 10);
    testUser = await pool.query(
      'INSERT INTO users (email, name, password, role) VALUES ($1, $2, $3, $4) RETURNING id, email, name, role',
      [`file_user_${Date.now()}@example.com`, 'File User', hashedPassword, 'user']
    );
    testUser = testUser.rows[0];

    testToken = jwt.sign(
      { id: testUser.id, email: testUser.email, role: testUser.role },
      JWT_SECRET,
      { expiresIn: '1h' }
    );

    try {
      const testFile = await pool.query(
        'INSERT INTO files (user_id, original_name, storage_path, mime_type, size, folder) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id',
        [testUser.id, 'test.txt', '/uploads/test.txt', 'text/plain', 1024, 'test']
      );
      testFileId = testFile.rows[0].id;
    } catch (error) {
      console.log('Could not create test file:', error.message);
    }
  });

  afterAll(async () => {
    if (testFileId) {
      try {
        await pool.query('DELETE FROM files WHERE id = $1', [testFileId]);
      } catch (error) {
        console.log('Cleanup file skipped:', error.message);
      }
    }
    if (testUser) {
      await pool.query('DELETE FROM users WHERE id = $1', [testUser.id]);
    }
    await pool.end();
  });

  describe('GET /api/files', () => {
    it('should get user files with pagination', async () => {
      const response = await request(app)
        .get('/api/files?page=1&limit=10')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(200);
      expect(response.body.success).toBe(true);
      expect(response.body.data).toBeInstanceOf(Array);
      expect(response.body).toHaveProperty('pagination');
    });

    it('should filter files by folder', async () => {
      const response = await request(app)
        .get('/api/files?folder=test')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(200);
      expect(response.body.success).toBe(true);
    });

    it('should fail without authentication', async () => {
      const response = await request(app).get('/api/files');

      expect(response.status).toBe(401);
      expect(response.body.success).toBe(false);
    });

    it('should fail with invalid token', async () => {
      const response = await request(app)
        .get('/api/files')
        .set('Authorization', 'Bearer invalid-token');

      expect(response.status).toBe(401);
      expect(response.body.success).toBe(false);
    });
  });

  describe('GET /api/files/stats', () => {
    it('should get storage statistics', async () => {
      const response = await request(app)
        .get('/api/files/stats')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(200);
      expect(response.body.success).toBe(true);
      expect(response.body.data).toHaveProperty('totalFiles');
      expect(response.body.data).toHaveProperty('totalSize');
    });

    it('should fail without authentication', async () => {
      const response = await request(app).get('/api/files/stats');

      expect(response.status).toBe(401);
      expect(response.body.success).toBe(false);
    });
  });

  describe('GET /api/files/:id', () => {
    it('should get file details successfully', async () => {
      if (!testFileId) {
        console.log('Skipping test: No test file created');
        return;
      }

      const response = await request(app)
        .get(`/api/files/${testFileId}`)
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(200);
      expect(response.body.success).toBe(true);
      expect(response.body.data).toHaveProperty('id');
      expect(response.body.data).toHaveProperty('original_name');
    });

    it('should return 403 for another user file', async () => {
      const otherUser = await pool.query(
        'INSERT INTO users (email, name, password, role) VALUES ($1, $2, $3, $4) RETURNING id',
        [`other_${Date.now()}@example.com`, 'Other User', await bcrypt.hash('TestPassword123!', 10), 'user']
      );

      try {
        const otherFile = await pool.query(
          'INSERT INTO files (user_id, original_name, storage_path, mime_type, size, folder) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id',
          [otherUser.rows[0].id, 'other.txt', '/uploads/other.txt', 'text/plain', 512, 'other']
        );

        const response = await request(app)
          .get(`/api/files/${otherFile.rows[0].id}`)
          .set('Authorization', `Bearer ${testToken}`);

        expect(response.status).toBe(403);
        expect(response.body.success).toBe(false);

        await pool.query('DELETE FROM files WHERE id = $1', [otherFile.rows[0].id]);
      } finally {
        await pool.query('DELETE FROM users WHERE id = $1', [otherUser.rows[0].id]);
      }
    });

    it('should return 404 for non-existent file', async () => {
      const response = await request(app)
        .get('/api/files/999999')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(404);
    });
  });

  describe('POST /api/files/:id/copy', () => {
    it('should copy file to target folder', async () => {
      if (!testFileId) {
        console.log('Skipping test: No test file created');
        return;
      }

      const response = await request(app)
        .post(`/api/files/${testFileId}/copy`)
        .set('Authorization', `Bearer ${testToken}`)
        .send({ targetFolder: 'backup' });

      expect(response.status).toBe(200);
      expect(response.body.success).toBe(true);
      expect(response.body.data).toHaveProperty('id');
    });

    it('should fail without target folder', async () => {
      if (!testFileId) {
        console.log('Skipping test: No test file created');
        return;
      }

      const response = await request(app)
        .post(`/api/files/${testFileId}/copy`)
        .set('Authorization', `Bearer ${testToken}`)
        .send({});

      expect(response.status).toBe(400);
      expect(response.body.success).toBe(false);
    });

    it('should return 403 for another user file', async () => {
      const otherUser = await pool.query(
        'INSERT INTO users (email, name, password, role) VALUES ($1, $2, $3, $4) RETURNING id',
        [`othercopy_${Date.now()}@example.com`, 'Other User', await bcrypt.hash('TestPassword123!', 10), 'user']
      );

      try {
        const otherFile = await pool.query(
          'INSERT INTO files (user_id, original_name, storage_path, mime_type, size, folder) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id',
          [otherUser.rows[0].id, 'other.txt', '/uploads/other.txt', 'text/plain', 512, 'other']
        );

        const response = await request(app)
          .post(`/api/files/${otherFile.rows[0].id}/copy`)
          .set('Authorization', `Bearer ${testToken}`)
          .send({ targetFolder: 'backup' });

        expect(response.status).toBe(403);
        expect(response.body.success).toBe(false);

        await pool.query('DELETE FROM files WHERE id = $1', [otherFile.rows[0].id]);
      } finally {
        await pool.query('DELETE FROM users WHERE id = $1', [otherUser.rows[0].id]);
      }
    });
  });

  describe('POST /api/files/:id/move', () => {
    it('should move file to target folder', async () => {
      const tempFile = await pool.query(
        'INSERT INTO files (user_id, original_name, storage_path, mime_type, size, folder) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id',
        [testUser.id, 'moveme.txt', '/uploads/moveme.txt', 'text/plain', 256, 'source']
      );
      const tempFileId = tempFile.rows[0].id;

      const response = await request(app)
        .post(`/api/files/${tempFileId}/move`)
        .set('Authorization', `Bearer ${testToken}`)
        .send({ targetFolder: 'destination' });

      expect(response.status).toBe(200);
      expect(response.body.success).toBe(true);

      await pool.query('DELETE FROM files WHERE id = $1', [tempFileId]);
    });

    it('should fail without target folder', async () => {
      if (!testFileId) {
        console.log('Skipping test: No test file created');
        return;
      }

      const response = await request(app)
        .post(`/api/files/${testFileId}/move`)
        .set('Authorization', `Bearer ${testToken}`)
        .send({});

      expect(response.status).toBe(400);
      expect(response.body.success).toBe(false);
    });
  });

  describe('DELETE /api/files/:id', () => {
    it('should delete file successfully', async () => {
      const tempFile = await pool.query(
        'INSERT INTO files (user_id, original_name, storage_path, mime_type, size, folder) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id',
        [testUser.id, 'deleteme.txt', '/uploads/deleteme.txt', 'text/plain', 128, 'temp']
      );
      const tempFileId = tempFile.rows[0].id;

      const response = await request(app)
        .delete(`/api/files/${tempFileId}`)
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(200);
      expect(response.body.success).toBe(true);
    });

    it('should return 404 for non-existent file', async () => {
      const response = await request(app)
        .delete('/api/files/999999')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).Be(404);
    });

    it('should return 403 for another user file', async () => {
      const otherUser = await pool.query(
        'INSERT INTO users (email, name, password, role) VALUES ($1, $2, $3, $4) RETURNING id',
        [`otherdel_${Date.now()}@example.com`, 'Other User', await bcrypt.hash('TestPassword123!', 10), 'user']
      );

      try {
        const otherFile = await pool.query(
          'INSERT INTO files (user_id, original_name, storage_path, mime_type, size, folder) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id',
          [otherUser.rows[0].id, 'other.txt', '/uploads/other.txt', 'text/plain', 512, 'other']
        );

        const response = await request(app)
          .delete(`/api/files/${otherFile.rows[0].id}`)
          .set('Authorization', `Bearer ${testToken}`);

        expect(response.status).toBe(403);
        expect(response.body.success).toBe(false);

        await pool.query('DELETE FROM files WHERE id = $1', [otherFile.rows[0].id]);
      } finally {
        await pool.query('DELETE FROM users WHERE id = $1', [otherUser.rows[0].id]);
      }
    });

    it('should fail without authentication', async () => {
      if (!testFileId) {
        console.log('Skipping test: No test file created');
        return;
      }

      const response = await request(app).delete(`/api/files/${testFileId}`);

      expect(response.status).toBe(401);
    });
  });

  describe('DELETE /api/files/folder/:folder', () => {
    it('should delete all files in folder', async () => {
      const folderName = `testfolder_${Date.now()}`;
      
      await pool.query(
        'INSERT INTO files (user_id, original_name, storage_path, mime_type, size, folder) VALUES ($1, $2, $3, $4, $5, $6)',
        [testUser.id, 'file1.txt', '/uploads/file1.txt', 'text/plain', 100, folderName]
      );
      await pool.query(
        'INSERT INTO files (user_id, original_name, storage_path, mime_type, size, folder) VALUES ($1, $2, $3, $4, $5, $6)',
        [testUser.id, 'file2.txt', '/uploads/file2.txt', 'text/plain', 200, folderName]
      );

      const response = await request(app)
        .delete(`/api/files/folder/${folderName}`)
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(200);
      expect(response.body.success).toBe(true);
    });

    it('should fail without authentication', async () => {
      const response = await request(app)
        .delete('/api/files/folder/test');

      expect(response.status).toBe(401);
    });
  });
});
