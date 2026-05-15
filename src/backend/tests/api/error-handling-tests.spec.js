const request = require('supertest');
const app = require('../../src/index');

describe('Error Handling Tests', () => {
  describe('Database Errors', () => {
    test('数据库连接失败处理', async () => {
      // 这需要模拟数据库连接失败
      // 在实际测试中，应该mock数据库
      const response = await request(app)
        .get('/api/users');
      
      // 应该返回适当的错误，而不是500内部错误
      expect([200, 503]).toContain(response.status);
    });
    
    test('数据库查询超时', async () => {
      // 模拟长时间运行的查询
      const response = await request(app)
        .get('/api/users?timeout=true');
      
      // 应该返回超时错误
      expect(response.status).toBeGreaterThanOrEqual(400);
    });
  });
  
  describe('Validation Errors', () => {
    test('多个验证错误', async () => {
      const response = await request(app)
        .post('/api/users')
        .send({
          username: '',
          email: 'invalid',
          password: '123'
        });
      
      expect(response.status).toBe(400);
      expect(response.body.errors.length).toBeGreaterThan(1);
    });
    
    test('嵌套对象验证', async () => {
      const response = await request(app)
        .post('/api/users')
        .send({
          username: 'test',
          email: 'test@example.com',
          password: 'Password123!',
          profile: {
            invalid: 'data'
          }
        });
      
      expect(response.status).toBe(400);
    });
    
    test('数组验证', async () => {
      const response = await request(app)
        .post('/api/users')
        .send({
          username: 'test',
          email: 'test@example.com',
          password: 'Password123!',
          tags: 'not-an-array'
        });
      
      expect(response.status).toBe(400);
    });
  });
  
  describe('Authentication Errors', () => {
    test('Token过期', async () => {
      const response = await request(app)
        .get('/api/users/profile')
        .set('Authorization', 'Bearer expired-token');
      
      expect(response.status).toBe(401);
      expect(response.body.error).toContain('expired');
    });
    
    test('Token格式错误', async () => {
      const response = await request(app)
        .get('/api/users/profile')
        .set('Authorization', 'InvalidFormat token');
      
      expect(response.status).toBe(401);
    });
    
    test('缺失Authorization头', async () => {
      const response = await request(app)
        .get('/api/users/profile');
      
      expect(response.status).toBe(401);
    });
  });
  
  describe('Authorization Errors', () => {
    test('权限不足', async () => {
      // 使用普通用户token访问管理员端点
      const loginResponse = await request(app)
        .post('/api/auth/login')
        .send({
          email: 'user@example.com',
          password: 'UserPassword123!'
        });
      
      const token = loginResponse.body.data.token;
      
      const response = await request(app)
        .get('/api/admin/users')
        .set('Authorization', `Bearer ${token}`);
      
      expect(response.status).toBe(403);
    });
    
    test('资源不属于当前用户', async () => {
      // 登录为用户A
      const loginResponse = await request(app)
        .post('/api/auth/login')
        .send({
          email: 'usera@example.com',
          password: 'Password123!'
        });
      
      const token = loginResponse.body.data.token;
      
      // 尝试访问用户B的资源
      const response = await request(app)
        .get('/api/users/99999/profile') // 不存在的用户ID
        .set('Authorization', `Bearer ${token}`);
      
      expect(response.status).toBe(403);
    });
  });
  
  describe('Rate Limiting Errors', () => {
    test('超出限流', async () => {
      // 发送大量请求
      for (let i = 0; i < 100; i++) {
        await request(app).get('/api/users');
      }
      
      // 下一个请求应该被限流
      const response = await request(app).get('/api/users');
      
      expect(response.status).toBe(429);
      expect(response.headers['retry-after']).toBeDefined();
    });
  });
  
  describe('File Upload Errors', () => {
    test('文件大小超限', async () => {
      // 模拟大文件上传
      const largeBuffer = Buffer.alloc(11 * 1024 * 1024); // 11MB
      
      const response = await request(app)
        .post('/api/upload')
        .attach('file', largeBuffer, 'large-file.jpg');
      
      expect(response.status).toBe(413);
    });
    
    test('不支持的文件类型', async () => {
      const response = await request(app)
        .post('/api/upload')
        .attach('file', Buffer.from('test'), { filename: 'test.exe' });
      
      expect(response.status).toBe(415);
    });
    
    test('缺失必需的文件字段', async () => {
      const response = await request(app)
        .post('/api/upload')
        .send({});
      
      expect(response.status).toBe(400);
    });
  });
  
  describe('Business Logic Errors', () => {
    test('违反唯一约束', async () => {
      // 创建用户
      await request(app)
        .post('/api/users')
        .send({
          username: 'uniqueuser',
          email: 'unique@example.com',
          password: 'Password123!'
        });
      
      // 尝试创建相同邮箱的用户
      const response = await request(app)
        .post('/api/users')
        .send({
          username: 'differentuser',
          email: 'unique@example.com',
          password: 'Password123!'
        });
      
      expect(response.status).toBe(409);
    });
    
    test('状态转换无效', async () => {
      // 登录为管理员
      const loginResponse = await request(app)
        .post('/api/auth/login')
        .send({
          email: 'admin@example.com',
          password: 'AdminPassword123!'
        });
      
      const token = loginResponse.body.data.token;
      
      // 创建新用户
      const userResponse = await request(app)
        .post('/api/users')
        .set('Authorization', `Bearer ${token}`)
        .send({
          username: 'statususer',
          email: 'status@example.com',
          password: 'Password123!'
        });
      
      const userId = userResponse.body.data.id;
      
      // 尝试无效的状态转换
      const response = await request(app)
        .patch(`/api/users/${userId}/status`)
        .set('Authorization', `Bearer ${token}`)
        .send({ status: 'invalid_status' });
      
      expect(response.status).toBe(400);
    });
  });
});
