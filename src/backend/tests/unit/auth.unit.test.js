const express = require('express');
const request = require('supertest');

const mockAuthService = {
  login: jest.fn(),
  verifyToken: jest.fn(),
  refreshToken: jest.fn()
};

const mockUserService = {
  createUser: jest.fn()
};

jest.mock('../../services/authService', () => mockAuthService);
jest.mock('../../services/userService', () => mockUserService);

const authRoutes = require('../../routes/v1/auth');
const {
  generateToken,
  testPassword,
  invalidUserCredentials,
  invalidEmailFormat,
  weakPasswordData,
  HTTP_STATUS
} = require('../helpers/testFixtures');

const app = express();
app.use(express.json());
app.use('/api/v1/auth', authRoutes);

describe('Auth API Unit Tests', () => {
  const mockUser = {
    id: 1,
    email: 'test@example.com',
    name: 'Test User',
    role: 'user'
  };

  beforeEach(() => {
    jest.clearAllMocks();
    mockAuthService.login.mockResolvedValue({
      user: mockUser,
      token: 'mock-token'
    });
    mockUserService.createUser.mockResolvedValue(mockUser);
  });

  describe('POST /api/v1/auth/login', () => {
    it('should login successfully with valid credentials', async () => {
      mockAuthService.login.mockResolvedValue({
        user: mockUser,
        token: 'valid-token'
      });

      const response = await request(app).post('/api/v1/auth/login').send({
        email: mockUser.email,
        password: testPassword
      });

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
      expect(response.body.data).toHaveProperty('user');
      expect(response.body.data).toHaveProperty('token');
    });

    it('should fail with incorrect password', async () => {
      mockAuthService.login.mockRejectedValue(new Error('Invalid credentials'));

      const response = await request(app).post('/api/v1/auth/login').send({
        email: mockUser.email,
        password: 'WrongPassword123!'
      });

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
      expect(response.body.success).toBe(false);
    });

    it('should fail with non-existent email', async () => {
      mockAuthService.login.mockRejectedValue(new Error('Invalid credentials'));

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
        .send({ email: mockUser.email, password: '' });

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
    });
  });

  describe('POST /api/v1/auth/register', () => {
    it('should register a new user successfully', async () => {
      const newUser = { ...mockUser, email: 'newuser@example.com' };
      mockAuthService.login.mockResolvedValue({
        user: newUser,
        token: 'new-token'
      });
      mockUserService.createUser.mockResolvedValue(newUser);

      const response = await request(app).post('/api/v1/auth/register').send({
        email: 'newuser@example.com',
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
      const response = await request(app).post('/api/v1/auth/register').send(weakPasswordData);

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
      expect(response.body.success).toBe(false);
    });

    it('should fail when email already exists', async () => {
      const error = new Error('Email already exists');
      error.code = '23505';
      mockUserService.createUser.mockRejectedValue(error);

      const response = await request(app).post('/api/v1/auth/register').send({
        email: mockUser.email,
        name: 'Duplicate User',
        password: testPassword
      });

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
      expect(response.body.success).toBe(false);
    });

    it('should fail with missing required fields', async () => {
      const response = await request(app).post('/api/v1/auth/register').send({
        email: 'missing@example.com'
      });

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
    });
  });

  describe('POST /api/v1/auth/verify', () => {
    it('should verify valid token successfully', async () => {
      const testToken = generateToken(mockUser);
      mockAuthService.verifyToken.mockResolvedValue(mockUser);

      const response = await request(app).post('/api/v1/auth/verify').send({ token: testToken });

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
      expect(response.body.data.valid).toBe(true);
      expect(response.body.data.user).toHaveProperty('id');
    });

    it('should fail with invalid token', async () => {
      mockAuthService.verifyToken.mockRejectedValue(new Error('Invalid token'));

      const response = await request(app)
        .post('/api/v1/auth/verify')
        .send({ token: 'invalid-token' });

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
      expect(response.body.success).toBe(false);
    });

    it('should fail without token', async () => {
      const response = await request(app).post('/api/v1/auth/verify').send({});

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
    });
  });

  describe('POST /api/v1/auth/refresh', () => {
    it('should refresh token successfully', async () => {
      const oldToken = generateToken(mockUser);
      const newToken = 'refreshed-token';
      mockAuthService.refreshToken.mockResolvedValue({ token: newToken });

      const response = await request(app).post('/api/v1/auth/refresh').send({ token: oldToken });

      expect(response.status).toBe(HTTP_STATUS.OK);
      expect(response.body.success).toBe(true);
      expect(response.body.data).toHaveProperty('token');
      expect(response.body.data.token).toBe(newToken);
    });

    it('should fail with invalid token', async () => {
      mockAuthService.refreshToken.mockRejectedValue(new Error('Invalid token'));

      const response = await request(app)
        .post('/api/v1/auth/refresh')
        .send({ token: 'invalid-token' });

      expect(response.status).toBe(HTTP_STATUS.UNAUTHORIZED);
      expect(response.body.success).toBe(false);
    });

    it('should fail without token', async () => {
      const response = await request(app).post('/api/v1/auth/refresh').send({});

      expect(response.status).toBe(HTTP_STATUS.BAD_REQUEST);
      expect(response.body.success).toBe(false);
    });
  });

  describe('POST /api/v1/auth/logout', () => {
    it('should logout successfully', async () => {
      const testToken = generateToken(mockUser);

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
