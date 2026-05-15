const request = require('supertest');
const app = require('../../src/index');

describe('Boundary Condition Tests', () => {
  describe('String Length Boundaries', () => {
    test('用户名最小长度', async () => {
      const response = await request(app)
        .post('/api/users')
        .send({
          username: 'ab', // 2个字符，应该失败
          email: 'min@example.com',
          password: 'Password123!'
        });
      
      expect(response.status).toBe(400);
    });
    
    test('用户名最大长度', async () => {
      const response = await request(app)
        .post('/api/users')
        .send({
          username: 'a'.repeat(51), // 51个字符，应该失败
          email: 'max@example.com',
          password: 'Password123!'
        });
      
      expect(response.status).toBe(400);
    });
    
    test('用户名边界值（3字符）', async () => {
      const response = await request(app)
        .post('/api/users')
        .send({
          username: 'abc', // 3个字符，应该成功
          email: 'boundary@example.com',
          password: 'Password123!'
        });
      
      expect(response.status).toBe(201);
    });
    
    test('用户名边界值（50字符）', async () => {
      const response = await request(app)
        .post('/api/users')
        .send({
          username: 'a'.repeat(50), // 50个字符，应该成功
          email: 'boundary50@example.com',
          password: 'Password123!'
        });
      
      expect(response.status).toBe(201);
    });
  });
  
  describe('Email Validation Boundaries', () => {
    test('无效邮箱格式', async () => {
      const invalidEmails = [
        'notanemail',
        '@example.com',
        'test@',
        'test@.com',
        'test space@example.com',
        ''
      ];
      
      for (const email of invalidEmails) {
        const response = await request(app)
          .post('/api/users')
          .send({
            username: 'test',
            email: email,
            password: 'Password123!'
          });
        
        expect(response.status).toBe(400);
      }
    });
    
    test('有效邮箱格式', async () => {
      const validEmails = [
        'simple@example.com',
        'very.common@example.com',
        'disposable.style.email.with+symbol@example.com',
        'other.email-with-hyphen@example.com',
        'user.name+tag+sorting@example.com',
        'x@example.com'
      ];
      
      for (const email of validEmails) {
        const response = await request(app)
          .post('/api/users')
          .send({
            username: email.split('@')[0].replace(/[^a-zA-Z0-9]/g, ''),
            email: email,
            password: 'Password123!'
          });
        
        // 应该通过格式验证（可能因其他原因失败，但不应该是邮箱格式）
        expect(response.body.errors).not.toContainEqual(
          expect.objectContaining({ field: 'email' })
        );
      }
    });
  });
  
  describe('Password Strength Boundaries', () => {
    test('密码太短', async () => {
      const response = await request(app)
        .post('/api/users')
        .send({
          username: 'test',
          email: 'test@example.com',
          password: 'Pass1!' // 太短
        });
      
      expect(response.status).toBe(400);
    });
    
    test('密码缺少大写字母', async () => {
      const response = await request(app)
        .post('/api/users')
        .send({
          username: 'test',
          email: 'test@example.com',
          password: 'password123!' // 缺少大写
        });
      
      expect(response.status).toBe(400);
    });
    
    test('密码缺少小写字母', async () => {
      const response = await request(app)
        .post('/api/users')
        .send({
          username: 'test',
          email: 'test@example.com',
          password: 'PASSWORD123!' // 缺少小写
        });
      
      expect(response.status).toBe(400);
    });
    
    test('密码缺少数字', async () => {
      const response = await request(app)
        .post('/api/users')
        .send({
          username: 'test',
          email: 'test@example.com',
          password: 'Password!' // 缺少数字
        });
      
      expect(response.status).toBe(400);
    });
    
    test('密码缺少特殊字符', async () => {
      const response = await request(app)
        .post('/api/users')
        .send({
          username: 'test',
          email: 'test@example.com',
          password: 'Password123' // 缺少特殊字符
        });
      
      expect(response.status).toBe(400);
    });
    
    test('满足所有要求的密码', async () => {
      const response = await request(app)
        .post('/api/users')
        .send({
          username: 'test',
          email: 'test@example.com',
          password: 'Password123!' // 满足所有要求
        });
      
      expect(response.status).toBe(201);
    });
  });
  
  describe('Pagination Boundaries', () => {
    test('页码为0', async () => {
      const response = await request(app)
        .get('/api/users?page=0');
      
      expect(response.status).toBe(400);
    });
    
    test('页码为负数', async () => {
      const response = await request(app)
        .get('/api/users?page=-1');
      
      expect(response.status).toBe(400);
    });
    
    test('页码为非数字', async () => {
      const response = await request(app)
        .get('/api/users?page=abc');
      
      expect(response.status).toBe(400);
    });
    
    test('每页数量为0', async () => {
      const response = await request(app)
        .get('/api/users?limit=0');
      
      expect(response.status).toBe(400);
    });
    
    test('每页数量超过最大限制', async () => {
      const response = await request(app)
        .get('/api/users?limit=1001');
      
      expect(response.status).toBe(400);
    });
    
    test('每页数量为负数', async () => {
      const response = await request(app)
        .get('/api/users?limit=-1');
      
      expect(response.status).toBe(400);
    });
    
    test('有效的页码和数量', async () => {
      const response = await request(app)
        .get('/api/users?page=1&limit=50');
      
      expect(response.status).toBe(200);
    });
  });
  
  describe('Date/Time Boundaries', () => {
    test('无效日期格式', async () => {
      const response = await request(app)
        .get('/api/users?createdAfter=invalid-date');
      
      expect(response.status).toBe(400);
    });
    
    test('未来日期', async () => {
      const futureDate = new Date(Date.now() + 86400000 * 365).toISOString();
      const response = await request(app)
        .get(`/api/users?createdAfter=${futureDate}`);
      
      // 应该成功但返回空数组
      expect(response.status).toBe(200);
    });
    
    test('开始日期晚于结束日期', async () => {
      const start = '2025-01-01';
      const end = '2024-01-01';
      const response = await request(app)
        .get(`/api/users?startDate=${start}&endDate=${end}`);
      
      expect(response.status).toBe(400);
    });
  });
});
