const request = require('supertest');
const express = require('express');
const userService = require('../../../backend/services/userService');

jest.mock('../../../backend/services/userService');

const usersRouter = require('../../../backend/routes/users');

const app = express();
app.use(express.json());
app.use('/api/users', usersRouter);

describe('Users API Routes', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('GET /api/users', () => {
    it('should return all users', async () => {
      const mockUsers = [
        { id: 1, email: 'user1@example.com', name: 'User One', created_at: '2026-05-14T10:30:00.000Z' },
        { id: 2, email: 'user2@example.com', name: 'User Two', created_at: '2026-05-14T10:30:00.000Z' }
      ];

      userService.getAllUsers.mockResolvedValue(mockUsers);

      const response = await request(app).get('/api/users');

      expect(response.status).toBe(200);
      expect(response.body.success).toBe(true);
      expect(response.body.data).toHaveLength(2);
      expect(response.body.data[0].id).toBe(1);
      expect(response.body.data[0].email).toBe('user1@example.com');
      expect(userService.getAllUsers).toHaveBeenCalled();
    });

    it('should return empty array when no users exist', async () => {
      userService.getAllUsers.mockResolvedValue([]);

      const response = await request(app).get('/api/users');

      expect(response.status).toBe(200);
      expect(response.body.success).toBe(true);
      expect(response.body.data).toEqual([]);
    });

    it('should handle errors', async () => {
      userService.getAllUsers.mockRejectedValue(new Error('Database error'));

      const response = await request(app).get('/api/users');

      expect(response.status).toBe(500);
      expect(response.body.success).toBe(false);
      expect(response.body.error).toBe('Database error');
    });
  });

  describe('GET /api/users/:id', () => {
    it('should return user by id', async () => {
      const mockUser = { id: 1, email: 'user@example.com', name: 'Test User', created_at: '2026-05-14T10:30:00.000Z' };

      userService.getUserById.mockResolvedValue(mockUser);

      const response = await request(app).get('/api/users/1');

      expect(response.status).toBe(200);
      expect(response.body.success).toBe(true);
      expect(response.body.data.id).toBe(1);
      expect(response.body.data.email).toBe('user@example.com');
      expect(userService.getUserById).toHaveBeenCalledWith('1');
    });

    it('should return 404 when user not found', async () => {
      userService.getUserById.mockResolvedValue(undefined);

      const response = await request(app).get('/api/users/999');

      expect(response.status).toBe(404);
      expect(response.body.success).toBe(false);
      expect(response.body.error).toBe('User not found');
    });

    it('should handle errors', async () => {
      userService.getUserById.mockRejectedValue(new Error('Database error'));

      const response = await request(app).get('/api/users/1');

      expect(response.status).toBe(500);
      expect(response.body.success).toBe(false);
      expect(response.body.error).toBe('Database error');
    });
  });

  describe('POST /api/users', () => {
    it('should create new user', async () => {
      const newUser = { email: 'new@example.com', name: 'New User', password: 'password123' };
      const createdUser = { id: 3, email: newUser.email, name: newUser.name, created_at: '2026-05-14T10:30:00.000Z' };

      userService.createUser.mockResolvedValue(createdUser);

      const response = await request(app)
        .post('/api/users')
        .send(newUser);

      expect(response.status).toBe(201);
      expect(response.body.success).toBe(true);
      expect(response.body.data.id).toBe(3);
      expect(response.body.data.email).toBe('new@example.com');
      expect(userService.createUser).toHaveBeenCalledWith(newUser);
    });

    it('should handle creation errors', async () => {
      userService.createUser.mockRejectedValue(new Error('User already exists'));

      const response = await request(app)
        .post('/api/users')
        .send({ email: 'existing@example.com', name: 'User', password: 'pass123' });

      expect(response.status).toBe(400);
      expect(response.body.success).toBe(false);
      expect(response.body.error).toBe('User already exists');
    });

    it('should handle validation errors from service', async () => {
      userService.createUser.mockRejectedValue(new Error('Missing required fields'));

      const response = await request(app)
        .post('/api/users')
        .send({ email: 'incomplete@example.com' });

      expect(response.status).toBe(400);
      expect(response.body.success).toBe(false);
      expect(response.body.error).toBe('Missing required fields');
    });
  });

  describe('PUT /api/users/:id', () => {
    it('should update user', async () => {
      const updateData = { email: 'updated@example.com', name: 'Updated Name' };
      const updatedUser = { id: 1, email: updateData.email, name: updateData.name, created_at: '2026-05-14T10:30:00.000Z' };

      userService.updateUser.mockResolvedValue(updatedUser);

      const response = await request(app)
        .put('/api/users/1')
        .send(updateData);

      expect(response.status).toBe(200);
      expect(response.body.success).toBe(true);
      expect(response.body.data.id).toBe(1);
      expect(response.body.data.email).toBe('updated@example.com');
      expect(userService.updateUser).toHaveBeenCalledWith('1', updateData);
    });

    it('should return 400 for update errors', async () => {
      userService.updateUser.mockRejectedValue(new Error('Update failed'));

      const response = await request(app)
        .put('/api/users/1')
        .send({ email: 'updated@example.com' });

      expect(response.status).toBe(400);
      expect(response.body.success).toBe(false);
      expect(response.body.error).toBe('Update failed');
    });
  });

  describe('DELETE /api/users/:id', () => {
    it('should delete user', async () => {
      userService.deleteUser.mockResolvedValue();

      const response = await request(app).delete('/api/users/1');

      expect(response.status).toBe(200);
      expect(response.body.success).toBe(true);
      expect(response.body.message).toBe('User deleted');
      expect(userService.deleteUser).toHaveBeenCalledWith('1');
    });

    it('should return 400 for deletion errors', async () => {
      userService.deleteUser.mockRejectedValue(new Error('Delete failed'));

      const response = await request(app).delete('/api/users/1');

      expect(response.status).toBe(400);
      expect(response.body.success).toBe(false);
      expect(response.body.error).toBe('Delete failed');
    });
  });
});
