const bcrypt = require('bcryptjs');
const express = require('express');
const jwt = require('jsonwebtoken');
const request = require('supertest');

const authRoutes = require('../../routes/v1/auth');
const {
  generateToken,
  testPassword,
  validUserCredentials,
  invalidUserCredentials,
  validRegistrationData,
  invalidEmailFormat,
  weakPasswordData,
  ROLES,
  HTTP_STATUS
} = require('../helpers/testFixtures');

const app = express();
app.use(express.json());
app.use('/api/v1/auth', authRoutes);

const JWT_SECRET = process.env.JWT_SECRET || 'hjtpx-secret-key-change-in-production';

describe('Auth API Integration Tests', () => {
  let testUser;
  let testToken;
  let cleanupUsers = [];

  beforeAll(async () => {
    const bcrypt = require('bcryptjs');
    testUser = {
      id: 1,
      email: `existing_${Date.now()}@example.com`,
      name: 'Existing User',
      password: await bcrypt.hash(testPassword, 10),
      role: 'user',
      status: 'active'
    };
    cleanupUsers.push(testUser.id);
    testToken = generateToken(testUser);
  });

  afterAll(async () => {
    cleanupUsers = [];
  });

  describe('POST /api/v1/auth/register', () => {
    it('should register a new user successfully', async () => {
      const uniqueEmail = `newuser_${Date.now()}@example.com`;
      const response = await request(app).post('/api/v1/auth/register').send({
        email: uniqueEmail,
        name: 'New Test User',
        password: testPassword
      });

      expect(response.status).toBe(HTTP_STATUS.CREATED);
      expect(response.body.success).toBe(true);
      expect(response.body.data).toHaveProperty('user');
      expect(response.body.data).toHaveProperty('token');
    });

    it('should fail with invalid email format', async () => {
      const response = await request(app).post('/api/v1/auth/register').send(invalidEmailFormat);

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
      expect(response.body.success).toBe(false);
    });

    it('should fail with weak password', async () => {
      const response = await request(app)
        .post('/api/v1/auth/register')
        .send({
          ...weakPasswordData,
          email: `weakpass_${Date.now()}@example.com`
        });

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
      expect(response.body.success).toBe(false);
    });

    it('should fail when email already exists', async () => {
      const response = await request(app).post('/api/v1/auth/register').send({
        email: testUser.email,
        name: 'Duplicate User',
        password: testPassword
      });

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
      expect(response.body.success).toBe(false);
    });

    it('should fail with missing required fields', async () => {
      const response = await request(app)
        .post('/api/v1/auth/register')
        .send({
          email: `missing_${Date.now()}@example.com`
        });

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
    });
  });

  describe('POST /api/v1/auth/login', () => {
    it('should login successfully with valid credentials', async () => {
      const response = await request(app).post('/api/v1/auth/login').send({
        email: testUser.email,
        password: testPassword
      });

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
      expect(response.body.data).toHaveProperty('user');
      expect(response.body.data).toHaveProperty('token');
      expect(response.body.data.user.email).toBe(testUser.email);
    });

    it('should fail with incorrect password', async () => {
      const response = await request(app).post('/api/v1/auth/login').send({
        email: testUser.email,
        password: 'WrongPassword123!'
      });

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
      expect(response.body.success).toBe(false);
    });

    it('should fail with non-existent email', async () => {
      const response = await request(app).post('/api/v1/auth/login').send(invalidUserCredentials);

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
      expect(response.body.success).toBe(false);
    });

    it('should fail with missing credentials', async () => {
      const response = await request(app).post('/api/v1/auth/login').send({});

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
      expect(response.body.success).toBe(false);
    });

    it('should fail with empty email', async () => {
      const response = await request(app)
        .post('/api/v1/auth/login')
        .send({ email: '', password: testPassword });

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
    });

    it('should fail with empty password', async () => {
      const response = await request(app)
        .post('/api/v1/auth/login')
        .send({ email: testUser.email, password: '' });

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
    });
  });

  describe('POST /api/v1/auth/verify', () => {
    it('should verify valid token successfully', async () => {
      const response = await request(app).post('/api/v1/auth/verify').send({ token: testToken });

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
      expect(response.body.data.valid).toBe(true);
      expect(response.body.data.user).toHaveProperty('id');
    });

    it('should fail with invalid token', async () => {
      const response = await request(app)
        .post('/api/v1/auth/verify')
        .send({ token: 'invalid-token' });

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
      expect(response.body.success).toBe(false);
    });

    it('should fail with expired token', async () => {
      const expiredToken = jwt.sign({ id: testUser.id, email: testUser.email }, JWT_SECRET, {
        expiresIn: '-1h'
      });
      const response = await request(app).post('/api/v1/auth/verify').send({ token: expiredToken });

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });

    it('should fail without token', async () => {
      const response = await request(app).post('/api/v1/auth/verify').send({});

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
    });
  });

  describe('POST /api/v1/auth/refresh', () => {
    it('should refresh token successfully', async () => {
      const response = await request(app).post('/api/v1/auth/refresh').send({ token: testToken });

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
      expect(response.body.data).toHaveProperty('token');
      expect(response.body.data.token).not.toBe(testToken);
    });

    it('should fail with invalid token', async () => {
      const response = await request(app)
        .post('/api/v1/auth/refresh')
        .send({ token: 'invalid-token' });

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
      expect(response.body.success).toBe(false);
    });

    it('should fail with expired token', async () => {
      const expiredToken = jwt.sign({ id: testUser.id, email: testUser.email }, JWT_SECRET, {
        expiresIn: '-1h'
      });
      const response = await request(app)
        .post('/api/v1/auth/refresh')
        .send({ token: expiredToken });

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
    });

    it('should fail without token', async () => {
      const response = await request(app).post('/api/v1/auth/refresh').send({});

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
      expect(response.body.success).toBe(false);
    });
  });

  describe('POST /api/v1/auth/logout', () => {
    it('should logout successfully', async () => {
      const response = await request(app)
        .post('/api/v1/auth/logout')
        .set('Authorization', `Bearer ${testToken}`);

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
      expect(response.body.message).toContain('Logout');
    });

    it('should logout successfully even without token', async () => {
      const response = await request(app).post('/api/v1/auth/logout');

      expect(response.status).toBe(HTTP_STATUS.OK);
    });
  });
});
