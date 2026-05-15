const request = require('supertest');
const express = require('express');

const mockFileService = {
  getUserFiles: jest.fn(),
  getStorageStats: jest.fn(),
  getFile: jest.fn(),
  uploadFile: jest.fn(),
  copyFile: jest.fn(),
  moveFile: jest.fn(),
  deleteFile: jest.fn(),
  deleteFolder: jest.fn()
};

jest.mock('../../services/fileService', () => mockFileService);

const filesRoutes = require('../../routes/files');
const {
  generateToken,
  testPassword,
  HTTP_STATUS
} = require('../helpers/testFixtures');

const app = express();
app.use(express.json());
app.use('/api/v1/files', filesRoutes);

describe('Files API Unit Tests', () => {
  const mockUser = {
    id: 1,
    email: 'test@example.com',
    name: 'Test User',
    role: 'user'
  };

  const mockAdminUser = {
    id: 2,
    email: 'admin@example.com',
    name: 'Admin User',
    role: 'admin'
  };

  let testToken;
  let adminToken;

  const mockFile = {
    id: 1,
    user_id: mockUser.id,
    filename: 'test-file.txt',
    original_name: 'test-file.txt',
    mime_type: 'text/plain',
    size: 1024,
    folder: 'test',
    path: '/uploads/test/test-file.txt',
    created_at: new Date()
  };

  beforeEach(() => {
    jest.clearAllMocks();
    testToken = generateToken(mockUser);
    adminToken = generateToken(mockAdminUser);

    mockFileService.getUserFiles.mockResolvedValue({
      files: [mockFile],
      pagination: { page: 1, limit: 20, total: 1 }
    });
    mockFileService.getStorageStats.mockResolvedValue({
      totalFiles: 10,
      totalSize: 10240
    });
    mockFileService.getFile.mockResolvedValue(mockFile);
    mockFileService.copyFile.mockResolvedValue({ ...mockFile, id: 2 });
    mockFileService.moveFile.mockResolvedValue({ ...mockFile, folder: 'moved' });
    mockFileService.deleteFile.mockResolvedValue(true);
    mockFileService.deleteFolder.mockResolvedValue({ deletedCount: 3 });
  });

  describe('GET /api/v1/files', () => {
    it('should get user files successfully', async () => {
      const response = await request(app)
        .get('/api/v1/files')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
    });

    it('should support folder filtering', async () => {
      const response = await request(app)
        .get('/api/v1/files?folder=test')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(mockFileService.getUserFiles).toHaveBeenCalledWith(
        mockUser.id,
        expect.objectContaining({ folder: 'test' })
      );
    });

    it('should support pagination parameters', async () => {
      const response = await request(app)
        .get('/api/v1/files?page=1&limit=10')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(mockFileService.getUserFiles).toHaveBeenCalledWith(
        mockUser.id,
        expect.objectContaining({ page: 1, limit: 10 })
      );
    });

    it('should fail without authentication', async () => {
      const response = await request(app).get('/api/v1/files');

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
    it('should get file by id successfully', async () => {
      const response = await request(app)
        .get('/api/v1/files/1')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
      expect(response.body.data.id).toBe(mockFile.id);
    });

    it('should return 404 for non-existent file', async () => {
      mockFileService.getFile.mockRejectedValue(new Error('File not found'));

      const response = await request(app)
        .get('/api/v1/files/999')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.NOT_FOUND);
    });

    it('should fail to get other user file as regular user', async () => {
      mockFileService.getFile.mockResolvedValue({ ...mockFile, user_id: 999 });

      const response = await request(app)
        .get('/api/v1/files/1')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.FORBIDDEN);
    });

    it('should allow admin to get any file', async () => {
      mockFileService.getFile.mockResolvedValue({ ...mockFile, user_id: 999 });

      const response = await request(app)
        .get('/api/v1/files/1')
        .set('Authorization', `Bearer ${adminToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
    });

    it('should fail without authentication', async () => {
      const response = await request(app)
        .get('/api/v1/files/1');

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });
  });

  describe('POST /api/v1/files/:id/copy', () => {
    it('should copy file successfully', async () => {
      const response = await request(app)
        .post('/api/v1/files/1/copy')
        .set('Authorization', `Bearer ${testToken}`)
        .send({ targetFolder: 'copies' });

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
    });

    it('should fail with missing targetFolder', async () => {
      const response = await request(app)
        .post('/api/v1/files/1/copy')
        .set('Authorization', `Bearer ${testToken}`)
        .send({});

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
    });

    it('should return 404 for non-existent file', async () => {
      mockFileService.copyFile.mockRejectedValue(new Error('File not found'));

      const response = await request(app)
        .post('/api/v1/files/999/copy')
        .set('Authorization', `Bearer ${testToken}`)
        .send({ targetFolder: 'copies' });

      expect(response.status).toBe(HTTP_STATUS.NOT_FOUND);
    });

    it('should fail without authentication', async () => {
      const response = await request(app)
        .post('/api/v1/files/1/copy')
        .send({ targetFolder: 'copies' });

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });
  });

  describe('POST /api/v1/files/:id/move', () => {
    it('should move file successfully', async () => {
      const response = await request(app)
        .post('/api/v1/files/1/move')
        .set('Authorization', `Bearer ${testToken}`)
        .send({ targetFolder: 'moved' });

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
    });

    it('should fail with missing targetFolder', async () => {
      const response = await request(app)
        .post('/api/v1/files/1/move')
        .set('Authorization', `Bearer ${testToken}`)
        .send({});

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
    });

    it('should return 404 for non-existent file', async () => {
      mockFileService.moveFile.mockRejectedValue(new Error('File not found'));

      const response = await request(app)
        .post('/api/v1/files/999/move')
        .set('Authorization', `Bearer ${testToken}`)
        .send({ targetFolder: 'moved' });

      expect(response.status).toBe(HTTP_STATUS.NOT_FOUND);
    });

    it('should fail without authentication', async () => {
      const response = await request(app)
        .post('/api/v1/files/1/move')
        .send({ targetFolder: 'moved' });

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });
  });

  describe('DELETE /api/v1/files/:id', () => {
    it('should delete file successfully', async () => {
      const response = await request(app)
        .delete('/api/v1/files/1')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
    });

    it('should return 404 for non-existent file', async () => {
      mockFileService.deleteFile.mockRejectedValue(new Error('File not found'));

      const response = await request(app)
        .delete('/api/v1/files/999')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.NOT_FOUND);
    });

    it('should fail to delete other user file as regular user', async () => {
      mockFileService.getFile.mockResolvedValue({ ...mockFile, user_id: 999 });

      const response = await request(app)
        .delete('/api/v1/files/1')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.FORBIDDEN);
    });

    it('should allow admin to delete any file', async () => {
      mockFileService.getFile.mockResolvedValue({ ...mockFile, user_id: 999 });

      const response = await request(app)
        .delete('/api/v1/files/1')
        .set('Authorization', `Bearer ${adminToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
    });

    it('should fail without authentication', async () => {
      const response = await request(app)
        .delete('/api/v1/files/1');

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });
  });

  describe('DELETE /api/v1/files/folder/:folder', () => {
    it('should delete folder and its files successfully', async () => {
      const response = await request(app)
        .delete('/api/v1/files/folder/test')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
    });

    it('should fail without authentication', async () => {
      const response = await request(app)
        .delete('/api/v1/files/folder/test');

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });
  });
});
