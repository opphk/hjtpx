const request = require('supertest');
const app = require('../../src/index');

describe('Extended API Tests', () => {
  describe('User Management', () => {
    test('POST /api/users - 创建用户 - 成功', async () => {
      const response = await request(app)
        .post('/api/users')
        .send({
          username: 'newuser',
          email: 'newuser@example.com',
          password: 'Password123!',
          role: 'user'
        });
      
      expect(response.status).toBe(201);
      expect(response.body.success).toBe(true);
      expect(response.body.data).toHaveProperty('id');
      expect(response.body.data).toHaveProperty('username', 'newuser');
    });
    
    test('POST /api/users - 创建用户 - 邮箱格式错误', async () => {
      const response = await request(app)
        .post('/api/users')
        .send({
          username: 'newuser',
          email: 'invalid-email',
          password: 'Password123!'
        });
      
      expect(response.status).toBe(400);
      expect(response.body.success).toBe(false);
      expect(response.body.errors).toContainEqual(
        expect.objectContaining({
          field: 'email'
        })
      );
    });
    
    test('POST /api/users - 创建用户 - 密码强度不足', async () => {
      const response = await request(app)
        .post('/api/users')
        .send({
          username: 'newuser',
          email: 'newuser@example.com',
          password: '123' // 太弱
        });
      
      expect(response.status).toBe(400);
      expect(response.body.success).toBe(false);
      expect(response.body.errors).toContainEqual(
        expect.objectContaining({
          field: 'password',
          message: expect.stringContaining('密码强度')
        })
      );
    });
    
    test('POST /api/users - 创建用户 - 用户名已存在', async () => {
      // 先创建一个用户
      await request(app)
        .post('/api/users')
        .send({
          username: 'existinguser',
          email: 'existing@example.com',
          password: 'Password123!'
        });
      
      // 尝试创建相同用户名的用户
      const response = await request(app)
        .post('/api/users')
        .send({
          username: 'existinguser',
          email: 'different@example.com',
          password: 'Password123!'
        });
      
      expect(response.status).toBe(409);
      expect(response.body.success).toBe(false);
    });
  });
  
  describe('Authentication', () => {
    test('POST /api/auth/login - 登录成功', async () => {
      const response = await request(app)
        .post('/api/auth/login')
        .send({
          email: 'test@example.com',
          password: 'Password123!'
        });
      
      expect(response.status).toBe(200);
      expect(response.body.success).toBe(true);
      expect(response.body.data).toHaveProperty('token');
    });
    
    test('POST /api/auth/login - 错误密码', async () => {
      const response = await request(app)
        .post('/api/auth/login')
        .send({
          email: 'test@example.com',
          password: 'WrongPassword!'
        });
      
      expect(response.status).toBe(401);
      expect(response.body.success).toBe(false);
    });
    
    test('POST /api/auth/login - 用户不存在', async () => {
      const response = await request(app)
        .post('/api/auth/login')
        .send({
          email: 'nonexistent@example.com',
          password: 'Password123!'
        });
      
      expect(response.status).toBe(401);
    });
    
    test('POST /api/auth/logout - 登出成功', async () => {
      // 先登录获取token
      const loginResponse = await request(app)
        .post('/api/auth/login')
        .send({
          email: 'test@example.com',
          password: 'Password123!'
        });
      
      const token = loginResponse.body.data.token;
      
      // 登出
      const response = await request(app)
        .post('/api/auth/logout')
        .set('Authorization', `Bearer ${token}`);
      
      expect(response.status).toBe(200);
      expect(response.body.success).toBe(true);
    });
  });
  
  describe('Input Validation', () => {
    test('XSS攻击防护', async () => {
      const response = await request(app)
        .post('/api/users')
        .send({
          username: '<script>alert("XSS")</script>',
          email: 'xss@example.com',
          password: 'Password123!'
        });
      
      // 应该拒绝或转义
      expect(response.status).toBe(400);
    });
    
    test('SQL注入防护', async () => {
      const response = await request(app)
        .post('/api/users')
        .send({
          username: "admin'; DROP TABLE users; --",
          email: 'sqli@example.com',
          password: 'Password123!'
        });
      
      // 应该被拒绝
      expect(response.status).toBe(400);
    });
    
    test('空字符串验证', async () => {
      const response = await request(app)
        .post('/api/users')
        .send({
          username: '',
          email: '',
          password: ''
        });
      
      expect(response.status).toBe(400);
      expect(response.body.errors.length).toBeGreaterThan(0);
    });
    
    test('超长输入验证', async () => {
      const response = await request(app)
        .post('/api/users')
        .send({
          username: 'a'.repeat(100),
          email: 'long@example.com',
          password: 'Password123!'
        });
      
      expect(response.status).toBe(400);
    });
  });
  
  describe('Authorization', () => {
    test('访问受保护资源 - 无token', async () => {
      const response = await request(app)
        .get('/api/users/profile');
      
      expect(response.status).toBe(401);
    });
    
    test('访问受保护资源 - 无效token', async () => {
      const response = await request(app)
        .get('/api/users/profile')
        .set('Authorization', 'Bearer invalid-token');
      
      expect(response.status).toBe(401);
    });
    
    test('访问管理员资源 - 普通用户', async () => {
      // 以普通用户登录
      const loginResponse = await request(app)
        .post('/api/auth/login')
        .send({
          email: 'user@example.com',
          password: 'UserPassword123!'
        });
      
      const token = loginResponse.body.data.token;
      
      // 尝试访问管理员资源
      const response = await request(app)
        .get('/api/admin/users')
        .set('Authorization', `Bearer ${token}`);
      
      expect(response.status).toBe(403);
    });
  });
  
  describe('Pagination & Filtering', () => {
    test('用户列表分页', async () => {
      const response = await request(app)
        .get('/api/users?page=1&limit=10');
      
      expect(response.status).toBe(200);
      expect(response.body.data).toHaveProperty('users');
      expect(response.body.data).toHaveProperty('pagination');
      expect(response.body.data.pagination).toHaveProperty('page', 1);
      expect(response.body.data.pagination).toHaveProperty('limit', 10);
    });
    
    test('无效分页参数', async () => {
      const response = await request(app)
        .get('/api/users?page=-1&limit=1000');
      
      expect(response.status).toBe(400);
    });
    
    test('用户列表搜索', async () => {
      const response = await request(app)
        .get('/api/users?search=admin');
      
      expect(response.status).toBe(200);
      expect(Array.isArray(response.body.data.users)).toBe(true);
    });
    
    test('用户列表排序', async () => {
      const response = await request(app)
        .get('/api/users?sort=createdAt&order=desc');
      
      expect(response.status).toBe(200);
    });
  });
  
  describe('Error Handling', () => {
    test('404 - 资源不存在', async () => {
      const response = await request(app)
        .get('/api/nonexistent');
      
      expect(response.status).toBe(404);
      expect(response.body.success).toBe(false);
      expect(response.body.error).toContain('Not Found');
    });
    
    test('500 - 服务器错误', async () => {
      const response = await request(app)
        .post('/api/users')
        .send({
          // 故意触发服务器错误的数据
          triggerServerError: true
        });
      
      // 应该是400或500
      expect([400, 500]).toContain(response.status);
    });
    
    test('429 - 限流', async () => {
      // 发送大量请求触发限流
      const promises = [];
      for (let i = 0; i < 150; i++) {
        promises.push(
          request(app).get('/api/users')
        );
      }
      
      const responses = await Promise.all(promises);
      const rateLimitedResponses = responses.filter(r => r.status === 429);
      
      expect(rateLimitedResponses.length).toBeGreaterThan(0);
    });
  });
  
  describe('CORS & Security Headers', () => {
    test('CORS头存在', async () => {
      const response = await request(app)
        .get('/api/users');
      
      expect(response.headers).toHaveProperty('access-control-allow-origin');
    });
    
    test('安全头存在', async () => {
      const response = await request(app)
        .get('/api/users');
      
      expect(response.headers).toHaveProperty('x-content-type-options', 'nosniff');
      expect(response.headers).toHaveProperty('x-frame-options');
      expect(response.headers).toHaveProperty('x-xss-protection');
    });
  });
  
  describe('Response Format', () => {
    test('成功响应格式', async () => {
      const response = await request(app)
        .get('/api/users');
      
      expect(response.body).toHaveProperty('success', true);
      expect(response.body).toHaveProperty('data');
      expect(response.body).toHaveProperty('timestamp');
    });
    
    test('错误响应格式', async () => {
      const response = await request(app)
        .get('/api/nonexistent');
      
      expect(response.body).toHaveProperty('success', false);
      expect(response.body).toHaveProperty('error');
      expect(response.body).toHaveProperty('timestamp');
    });
  });
});
