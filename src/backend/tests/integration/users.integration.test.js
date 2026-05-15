const bcrypt = require('bcryptjs');
const express = require('express');
const request = require('supertest');

const usersRoutes = require('../../routes/v1/users');
const {
  generateToken,
  testPassword,
  userUpdateData,
  ROLES,
  HTTP_STATUS
} = require('../helpers/testFixtures');

const app = express();
app.use(express.json());
app.use('/api/v1/users', usersRoutes);

describe('Users API Integration Tests', () => {
  let regularUser;
  let adminUser;
  let regularToken;
  let adminToken;
  let cleanupUsers = [];

  beforeAll(async () => {
    const hashedPassword = await bcrypt.hash(testPassword, 10);

    regularUser = {
      id: 1,
      email: `regular_${Date.now()}@example.com`,
      name: 'Regular User',
      password: hashedPassword,
      role: 'user',
      status: 'active'
    };

    adminUser = {
      id: 2,
      email: `admin_${Date.now()}@example.com`,
      name: 'Admin User',
      password: hashedPassword,
      role: 'admin',
      status: 'active'
    };

    cleanupUsers.push(regularUser.id, adminUser.id);

    regularToken = generateToken(regularUser);
    adminToken = generateToken(adminUser);
  });

  afterAll(async () => {
    cleanupUsers = [];
  });

  describe('GET /api/v1/users/me', () => {
    it('should get current user profile successfully', async () => {
      const response = await request(app)
        .get('/api/v1/users/me')
        .set('Authorization', `Bearer ${regularToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
      expect(response.body.data).toHaveProperty('id');
      expect(response.body.data).toHaveProperty('email');
      expect(response.body.data).toHaveProperty('name');
      expect(response.body.data.email).toBe(regularUser.email);
    });

    it('should fail without authentication', async () => {
      const response = await request(app).get('/api/v1/users/me');

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
      expect(response.body.success).toBe(false);
    });

    it('should fail with invalid token', async () => {
      const response = await request(app)
        .get('/api/v1/users/me')
        .set('Authorization', 'Bearer invalid-token');

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
      expect(response.body.success).toBe(false);
    });
  });

  describe('PUT /api/v1/users/me', () => {
    it('should update current user profile successfully', async () => {
      const newName = `Updated Name ${Date.now()}`;
      const response = await request(app)
        .put('/api/v1/users/me')
        .set('Authorization', `Bearer ${regularToken}`)
        .send({
          name: newName
        });

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
      expect(response.body.data).toHaveProperty('name', newName);
    });

    it('should fail without authentication', async () => {
      const response = await request(app).put('/api/v1/users/me').send(userUpdateData);

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
      expect(response.body.success).toBe(false);
    });

    it('should fail to update protected fields (role)', async () => {
      const response = await request(app)
        .put('/api/v1/users/me')
        .set('Authorization', `Bearer ${regularToken}`)
        .send({
          name: 'Updated Name',
          role: ROLES.ADMIN
        });

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.data.role).not.toBe(ROLES.ADMIN);
    });
  });

  describe('GET /api/v1/users/:id', () => {
    it('should get user by id as admin', async () => {
      const response = await request(app)
        .get(`/api/v1/users/${regularUser.id}`)
        .set('Authorization', `Bearer ${adminToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
      expect(response.body.data.id).toBe(regularUser.id);
    });

    it('should get own user data as regular user', async () => {
      const response = await request(app)
        .get(`/api/v1/users/${regularUser.id}`)
        .set('Authorization', `Bearer ${regularToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
      expect(response.body.data.id).toBe(regularUser.id);
    });

    it('should fail to get other user data as regular user', async () => {
      const response = await request(app)
        .get(`/api/v1/users/${adminUser.id}`)
        .set('Authorization', `Bearer ${regularToken}`);

      expect(response.status).toBe(HTTP_STATUS.FORBIDDEN);
    });

    it('should return 404 for non-existent user', async () => {
      const response = await request(app)
        .get('/api/v1/users/999999')
        .set('Authorization', `Bearer ${adminToken}`);

      expect(response.status).toBe(HTTP_STATUS.NOT_FOUND);
    });
  });

  describe('PUT /api/v1/users/:id', () => {
    it('should update user by id as admin', async () => {
      const newName = `Admin Updated ${Date.now()}`;
      const response = await request(app)
        .put(`/api/v1/users/${regularUser.id}`)
        .set('Authorization', `Bearer ${adminToken}`)
        .send({ name: newName });

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
      expect(response.body.data.name).toBe(newName);
    });

    it('should update own user data as regular user', async () => {
      const newName = `Self Updated ${Date.now()}`;
      const response = await request(app)
        .put(`/api/v1/users/${regularUser.id}`)
        .set('Authorization', `Bearer ${regularToken}`)
        .send({ name: newName });

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
      expect(response.body.data.name).toBe(newName);
    });

    it('should fail to update other user as regular user', async () => {
      const response = await request(app)
        .put(`/api/v1/users/${adminUser.id}`)
        .set('Authorization', `Bearer ${regularToken}`)
        .send({ name: 'Hacked' });

      expect(response.status).toBe(HTTP_STATUS.FORBIDDEN);
    });

    it('should return 404 for non-existent user', async () => {
      const response = await request(app)
        .put('/api/v1/users/999999')
        .set('Authorization', `Bearer ${adminToken}`)
        .send({ name: 'Test' });

      expect(response.status).toBe(HTTP_STATUS.NOT_FOUND);
    });
  });

  describe('GET /api/v1/users', () => {
    it('should get all users for admin', async () => {
      const response = await request(app)
        .get('/api/v1/users')
        .set('Authorization', `Bearer ${adminToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
      expect(Array.isArray(response.body.data)).toBe(true);
    });

    it('should fail for non-admin users', async () => {
      const response = await request(app)
        .get('/api/v1/users')
        .set('Authorization', `Bearer ${regularToken}`);

      expect(response.status).toBe(HTTP_STATUS.FORBIDDEN);
      expect(response.body.success).toBe(false);
    });

    it('should fail without authentication', async () => {
      const response = await request(app).get('/api/v1/users');

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });
  });

  describe('POST /api/v1/users', () => {
    let testEmail;

    beforeEach(() => {
      testEmail = `newuser_${Date.now()}@example.com`;
    });

    afterEach(async () => {
      await pool.query('DELETE FROM users WHERE email = $1', [testEmail]);
    });

    it('should create a new user for admin', async () => {
      const response = await request(app)
        .post('/api/v1/users')
        .set('Authorization', `Bearer ${adminToken}`)
        .send({
          email: testEmail,
          name: 'New User',
          password: testPassword,
          role: ROLES.USER
        });

      expect(response.status).toBe(HTTP_STATUS.CREATED);
      expect(response.body.success).toBe(true);
      expect(response.body.data).toHaveProperty('id');
      expect(response.body.data.email).toBe(testEmail);
    });

    it('should fail for non-admin users', async () => {
      const response = await request(app)
        .post('/api/v1/users')
        .set('Authorization', `Bearer ${regularToken}`)
        .send({
          email: testEmail,
          name: 'New User',
          password: testPassword
        });

      expect(response.status).toBe(HTTP_STATUS.FORBIDDEN);
    });

    it('should fail without authentication', async () => {
      const response = await request(app).post('/api/v1/users').send({
        email: testEmail,
        name: 'New User',
        password: testPassword
      });

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });

    it('should fail with duplicate email', async () => {
      await request(app).post('/api/v1/users').set('Authorization', `Bearer ${adminToken}`).send({
        email: testEmail,
        name: 'New User',
        password: testPassword
      });

      const response = await request(app)
        .post('/api/v1/users')
        .set('Authorization', `Bearer ${adminToken}`)
        .send({
          email: testEmail,
          name: 'Duplicate User',
          password: testPassword
        });

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
    });
  });

  describe('DELETE /api/v1/users/:id', () => {
    let userToDelete;

    beforeEach(async () => {
      userToDelete = await userFactory.createUser();
      cleanupUsers.push(userToDelete.id);
    });

    it('should delete user for admin', async () => {
      const response = await request(app)
        .delete(`/api/v1/users/${userToDelete.id}`)
        .set('Authorization', `Bearer ${adminToken}`);

      expect(response.status).toBe(HTTP_STATUS.NO_CONTENT);
    });

    it('should return 404 for non-existent user', async () => {
      const response = await request(app)
        .delete('/api/v1/users/999999')
        .set('Authorization', `Bearer ${adminToken}`);

      expect(response.status).toBe(HTTP_STATUS.NOT_FOUND);
    });

    it('should fail for non-admin users', async () => {
      const response = await request(app)
        .delete(`/api/v1/users/${userToDelete.id}`)
        .set('Authorization', `Bearer ${regularToken}`);

      expect(response.status).toBe(HTTP_STATUS.FORBIDDEN);
    });

    it('should fail without authentication', async () => {
      const response = await request(app).delete(`/api/v1/users/${userToDelete.id}`);

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });
  });
});
