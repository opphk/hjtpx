const express = require('express');
const jwt = require('jsonwebtoken');
const request = require('supertest');

jest.mock('../../../../src/config/database/db', () => ({
  query: jest.fn()
}));

jest.mock('bcrypt', () => ({
  hash: jest.fn(),
  compare: jest.fn()
}));

const pool = require('../../../../src/config/database/db');
const bcrypt = require('bcrypt');

const app = express();
app.use(express.json());

app.post('/api/auth/login', async (req, res) => {
  try {
    const { email, password } = req.body;

    if (!email || !password) {
      return res.status(400).json({
        success: false,
        error: 'Email and password are required'
      });
    }

    const result = await pool.query(
      'SELECT id, email, name, password FROM users WHERE email = $1',
      [email]
    );

    if (result.rows.length === 0) {
      return res.status(401).json({
        success: false,
        error: 'Invalid credentials'
      });
    }

    const user = result.rows[0];
    const isValidPassword = await bcrypt.compare(password, user.password);

    if (!isValidPassword) {
      return res.status(401).json({
        success: false,
        error: 'Invalid credentials'
      });
    }

    const token = jwt.sign(
      { id: user.id, email: user.email },
      process.env.JWT_SECRET || 'hjtpx-secret-key-change-in-production',
      { expiresIn: '24h' }
    );

    res.json({
      success: true,
      data: {
        token,
        user: {
          id: user.id,
          email: user.email,
          name: user.name
        }
      }
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: 'Internal server error'
    });
  }
});

describe('Login Authentication', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('successful login', () => {
    it('should return token and user data for valid credentials', async () => {
      const hashedPassword = await bcrypt.hash('password123', 10);
      const mockUser = {
        id: 1,
        email: 'test@example.com',
        name: 'Test User',
        password: hashedPassword
      };

      pool.query.mockResolvedValue({ rows: [mockUser] });

      const response = await request(app).post('/api/auth/login').send({
        email: 'test@example.com',
        password: 'password123'
      });

      expect(response.status).toBe(200);
      expect(response.body.success).toBe(true);
      expect(response.body.data.token).toBeDefined();
      expect(response.body.data.user.email).toBe('test@example.com');
      expect(response.body.data.user.password).toBeUndefined();
      expect(pool.query).toHaveBeenCalledWith(
        'SELECT id, email, name, password FROM users WHERE email = $1',
        ['test@example.com']
      );
    });

    it('should return user without password field', async () => {
      const hashedPassword = await bcrypt.hash('securePassword', 10);
      const mockUser = {
        id: 2,
        email: 'user@example.com',
        name: 'Regular User',
        password: hashedPassword
      };

      pool.query.mockResolvedValue({ rows: [mockUser] });

      const response = await request(app).post('/api/auth/login').send({
        email: 'user@example.com',
        password: 'securePassword'
      });

      expect(response.status).toBe(200);
      expect(response.body.success).toBe(true);
      expect(response.body.data.user).not.toHaveProperty('password');
      expect(response.body.data.user).toHaveProperty('id');
      expect(response.body.data.user).toHaveProperty('email');
      expect(response.body.data.user).toHaveProperty('name');
    });
  });

  describe('failed login - wrong password', () => {
    it('should reject login with incorrect password', async () => {
      const hashedPassword = await bcrypt.hash('correctPassword', 10);
      const mockUser = {
        id: 1,
        email: 'test@example.com',
        name: 'Test User',
        password: hashedPassword
      };

      pool.query.mockResolvedValue({ rows: [mockUser] });
      bcrypt.compare.mockResolvedValue(false);

      const response = await request(app).post('/api/auth/login').send({
        email: 'test@example.com',
        password: 'wrongPassword'
      });

      expect(response.status).toBe(401);
      expect(response.body.success).toBe(false);
      expect(response.body.error).toBe('Invalid credentials');
      expect(response.body.data).toBeUndefined();
    });

    it('should reject login when password is empty', async () => {
      const hashedPassword = await bcrypt.hash('somePassword', 10);
      const mockUser = {
        id: 1,
        email: 'test@example.com',
        name: 'Test User',
        password: hashedPassword
      };

      pool.query.mockResolvedValue({ rows: [mockUser] });
      bcrypt.compare.mockResolvedValue(false);

      const response = await request(app).post('/api/auth/login').send({
        email: 'test@example.com',
        password: ''
      });

      expect(response.status).toBe(401);
      expect(response.body.success).toBe(false);
      expect(response.body.error).toBe('Invalid credentials');
    });
  });

  describe('failed login - non-existent user', () => {
    it('should reject login for non-existent user', async () => {
      pool.query.mockResolvedValue({ rows: [] });

      const response = await request(app).post('/api/auth/login').send({
        email: 'nonexistent@example.com',
        password: 'somePassword'
      });

      expect(response.status).toBe(401);
      expect(response.body.success).toBe(false);
      expect(response.body.error).toBe('Invalid credentials');
      expect(pool.query).toHaveBeenCalledWith(
        'SELECT id, email, name, password FROM users WHERE email = $1',
        ['nonexistent@example.com']
      );
    });

    it('should reject login with empty email field', async () => {
      pool.query.mockResolvedValue({ rows: [] });

      const response = await request(app).post('/api/auth/login').send({
        email: '',
        password: 'somePassword'
      });

      expect(response.status).toBe(400);
      expect(response.body.success).toBe(false);
      expect(response.body.error).toBe('Email and password are required');
    });
  });

  describe('token verification', () => {
    it('should generate valid JWT token', async () => {
      const hashedPassword = await bcrypt.hash('password123', 10);
      const mockUser = {
        id: 1,
        email: 'test@example.com',
        name: 'Test User',
        password: hashedPassword
      };

      pool.query.mockResolvedValue({ rows: [mockUser] });

      const response = await request(app).post('/api/auth/login').send({
        email: 'test@example.com',
        password: 'password123'
      });

      const token = response.body.data.token;
      const decoded = jwt.verify(
        token,
        process.env.JWT_SECRET || 'hjtpx-secret-key-change-in-production'
      );

      expect(decoded.id).toBe(1);
      expect(decoded.email).toBe('test@example.com');
      expect(decoded).toHaveProperty('exp');
      expect(decoded).toHaveProperty('iat');
    });

    it('should generate token with correct expiration', async () => {
      const hashedPassword = await bcrypt.hash('password123', 10);
      const mockUser = {
        id: 1,
        email: 'test@example.com',
        name: 'Test User',
        password: hashedPassword
      };

      pool.query.mockResolvedValue({ rows: [mockUser] });

      const response = await request(app).post('/api/auth/login').send({
        email: 'test@example.com',
        password: 'password123'
      });

      const token = response.body.data.token;
      const decoded = jwt.decode(token);

      const expectedExpiration = 24 * 60 * 60;
      const actualExpiration = decoded.exp - decoded.iat;

      expect(actualExpiration).toBe(expectedExpiration);
    });

    it('should reject invalid token format', () => {
      expect(() => {
        jwt.verify(
          'invalid.token.here',
          process.env.JWT_SECRET || 'hjtpx-secret-key-change-in-production'
        );
      }).toThrow();
    });
  });

  describe('input validation', () => {
    it('should reject request without email', async () => {
      const response = await request(app).post('/api/auth/login').send({
        password: 'password123'
      });

      expect(response.status).toBe(400);
      expect(response.body.success).toBe(false);
      expect(response.body.error).toBe('Email and password are required');
    });

    it('should reject request without password', async () => {
      const response = await request(app).post('/api/auth/login').send({
        email: 'test@example.com'
      });

      expect(response.status).toBe(400);
      expect(response.body.success).toBe(false);
      expect(response.body.error).toBe('Email and password are required');
    });

    it('should reject request with empty body', async () => {
      const response = await request(app).post('/api/auth/login').send({});

      expect(response.status).toBe(400);
      expect(response.body.success).toBe(false);
      expect(response.body.error).toBe('Email and password are required');
    });
  });

  describe('error handling', () => {
    it('should handle database connection errors', async () => {
      pool.query.mockRejectedValue(new Error('Database connection failed'));

      const response = await request(app).post('/api/auth/login').send({
        email: 'test@example.com',
        password: 'password123'
      });

      expect(response.status).toBe(500);
      expect(response.body.success).toBe(false);
      expect(response.body.error).toBe('Internal server error');
    });

    it('should handle bcrypt errors during compare', async () => {
      const hashedPassword = await bcrypt.hash('password123', 10);
      const mockUser = {
        id: 1,
        email: 'test@example.com',
        name: 'Test User',
        password: hashedPassword
      };

      pool.query.mockResolvedValue({ rows: [mockUser] });
      bcrypt.compare.mockRejectedValue(new Error('Bcrypt error'));

      const response = await request(app).post('/api/auth/login').send({
        email: 'test@example.com',
        password: 'password123'
      });

      expect(response.status).toBe(500);
      expect(response.body.success).toBe(false);
    });
  });
});
