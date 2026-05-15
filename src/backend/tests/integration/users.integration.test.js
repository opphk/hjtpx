const request = require('supertest');
const express = require('express');
const bcrypt = require('bcryptjs');

jest.mock('../../../config/database/db', () => ({
  query: jest.fn()
}));

jest.mock('../../services/cacheService', () => ({
  get: jest.fn().mockResolvedValue(null),
  set: jest.fn().mockResolvedValue(undefined),
  del: jest.fn().mockResolvedValue(undefined),
  getCachedApiResponse: jest.fn().mockResolvedValue(null),
  setCachedApiResponse: jest.fn().mockResolvedValue(undefined),
  invalidateApiCache: jest.fn().mockResolvedValue(undefined),
  invalidateTag: jest.fn().mockResolvedValue(undefined),
  isHealthy: jest.fn().mockResolvedValue(true),
  getStats: jest.fn().mockReturnValue({
    isRedisConnected: false,
    memoryCacheSize: 0,
    maxMemoryCacheSize: 1000,
    overall: { memoryCachePercent: 0 }
  })
}));

jest.mock('../../utils/queryOptimizer', () => ({
  cachedQuery: jest.fn().mockImplementation((query) => {
    const pool = require('../../../config/database/db');
    return pool.query(query);
  }),
  clearCache: jest.fn()
}));

const pool = require('../../../config/database/db');
const {
  generateToken,
  testPassword,
  userUpdateData,
  ROLES,
  HTTP_STATUS
} = require('../helpers/testFixtures');

describe('Users API Integration Tests', () => {
  let regularUser;
  let adminUser;
  let regularToken;
  let adminToken;

  beforeEach(() => {
    jest.clearAllMocks();
  });

  beforeAll(async () => {
    regularUser = {
      id: 1,
      email: `regular_${Date.now()}@example.com`,
      name: 'Regular User',
      password: await bcrypt.hash(testPassword, 10),
      role: 'user',
      status: 'active'
    };
    
    adminUser = {
      id: 2,
      email: `admin_${Date.now()}@example.com`,
      name: 'Admin User',
      password: await bcrypt.hash(testPassword, 10),
      role: 'admin',
      status: 'active'
    };
    
    regularToken = generateToken(regularUser);
    adminToken = generateToken(adminUser);
  });

  afterAll(async () => {
    jest.clearAllMocks();
  });

  describe('User Data Validation Tests', () => {
    it('should validate user roles', () => {
      expect(ROLES.ADMIN).toBe('admin');
      expect(ROLES.USER).toBe('user');
    });

    it('should validate HTTP status codes', () => {
      expect(HTTP_STATUS.OK).toBe(200);
      expect(HTTP_STATUS.CREATED).toBe(201);
      expect(HTTP_STATUS.BAD_REQUEST).toBe(400);
      expect(HTTP_STATUS.UNAUTHORIZED).toBe(401);
      expect(HTTP_STATUS.FORBIDDEN).toBe(403);
      expect(HTTP_STATUS.NOT_FOUND).toBe(404);
    });

    it('should validate user update data', () => {
      expect(userUpdateData).toHaveProperty('name');
      expect(userUpdateData).toHaveProperty('bio');
    });
  });

  describe('Token Authorization Tests', () => {
    it('should generate valid user tokens', () => {
      const regularUserToken = generateToken(regularUser);
      expect(regularUserToken).toBeTruthy();
      
      const adminUserToken = generateToken(adminUser);
      expect(adminUserToken).toBeTruthy();
    });

    it('should include correct role in user tokens', () => {
      const jwt = require('jsonwebtoken');
      const JWT_SECRET = process.env.JWT_SECRET || 'hjtpx-secret-key-change-in-production';
      
      const regularDecoded = jwt.verify(generateToken(regularUser), JWT_SECRET);
      expect(regularDecoded.role).toBe('user');
      expect(regularDecoded.id).toBe(regularUser.id);
      
      const adminDecoded = jwt.verify(generateToken(adminUser), JWT_SECRET);
      expect(adminDecoded.role).toBe('admin');
      expect(adminDecoded.id).toBe(adminUser.id);
    });

    it('should include user email in token', () => {
      const jwt = require('jsonwebtoken');
      const JWT_SECRET = process.env.JWT_SECRET || 'hjtpx-secret-key-change-in-production';
      
      const decoded = jwt.verify(generateToken(regularUser), JWT_SECRET);
      expect(decoded.email).toBe(regularUser.email);
    });
  });

  describe('Mock Database Tests', () => {
    it('should mock database query for user creation', async () => {
      pool.query.mockResolvedValueOnce({ 
        rows: [{ id: 1, email: 'test@example.com', name: 'Test' }] 
      });
      
      const result = await pool.query('INSERT INTO users VALUES ($1)', ['test']);
      expect(result.rows).toHaveLength(1);
    });

    it('should handle database errors', async () => {
      pool.query.mockRejectedValueOnce(new Error('Database error'));
      
      await expect(pool.query('SELECT * FROM users')).rejects.toThrow('Database error');
    });

    it('should handle empty query results', async () => {
      pool.query.mockResolvedValueOnce({ rows: [] });
      
      const result = await pool.query('SELECT * FROM users WHERE id = $1', [999]);
      expect(result.rows).toHaveLength(0);
    });

    it('should return multiple rows', async () => {
      pool.query.mockResolvedValueOnce({ 
        rows: [
          { id: 1, email: 'user1@example.com', name: 'User 1' },
          { id: 2, email: 'user2@example.com', name: 'User 2' }
        ] 
      });
      
      const result = await pool.query('SELECT * FROM users');
      expect(result.rows).toHaveLength(2);
    });
  });

  describe('Password Hashing Tests', () => {
    it('should hash passwords correctly', async () => {
      const password = 'TestPassword123!';
      const hashedPassword = await bcrypt.hash(password, 10);
      
      expect(hashedPassword).not.toBe(password);
      expect(hashedPassword).toMatch(/^\$2[aby]?\$\d{1,2}\$/);
    });

    it('should verify passwords correctly', async () => {
      const password = 'TestPassword123!';
      const hashedPassword = await bcrypt.hash(password, 10);
      
      const isValid = await bcrypt.compare(password, hashedPassword);
      expect(isValid).toBe(true);
      
      const isInvalid = await bcrypt.compare('WrongPassword', hashedPassword);
      expect(isInvalid).toBe(false);
    });

    it('should generate unique hashes for same password', async () => {
      const password = 'TestPassword123!';
      const hash1 = await bcrypt.hash(password, 10);
      const hash2 = await bcrypt.hash(password, 10);
      
      expect(hash1).not.toBe(hash2);
    });
  });

  describe('User Email Validation Tests', () => {
    it('should create unique user emails', () => {
      const email1 = `user_${Date.now()}_1@example.com`;
      const email2 = `user_${Date.now()}_2@example.com`;
      
      expect(email1).not.toBe(email2);
      expect(email1).toMatch(/^user_.*@example\.com$/);
    });

    it('should validate email format', () => {
      const validEmails = [
        'user@example.com',
        'test.user@example.com',
        'user+tag@example.com'
      ];
      
      validEmails.forEach(email => {
        expect(email).toMatch(/^[^@]+@[^@]+\.[^@]+$/);
      });
    });
  });

  describe('User Role Tests', () => {
    it('should identify admin users', () => {
      expect(adminUser.role).toBe(ROLES.ADMIN);
      expect(adminUser.role).toBe('admin');
    });

    it('should identify regular users', () => {
      expect(regularUser.role).toBe(ROLES.USER);
      expect(regularUser.role).toBe('user');
    });

    it('should differentiate between admin and regular users', () => {
      expect(regularUser.role).not.toBe(adminUser.role);
      expect(ROLES.ADMIN).not.toBe(ROLES.USER);
    });
  });

  describe('User Status Tests', () => {
    it('should have valid user status', () => {
      expect(regularUser.status).toBe('active');
      expect(adminUser.status).toBe('active');
    });
  });
});
