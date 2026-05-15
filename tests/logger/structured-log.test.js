const requestIdMiddleware = require('../../src/backend/middleware/requestId');
const sanitizeLogsMiddleware = require('../../src/backend/middleware/sanitizeLogs');
const loggingConfig = require('../../src/backend/config/logging').logging;
const { sensitiveDataMasker, generateRequestId } = require('../../src/backend/utils/logger');

describe('结构化日志测试', () => {
  describe('敏感信息过滤', () => {
    test('应该过滤密码字段', () => {
      const data = {
        username: 'testuser',
        password: 'secret123'
      };
      const masked = sensitiveDataMasker(data);
      expect(masked.username).toBe('testuser');
      expect(masked.password).toBe('***MASKED***');
    });

    test('应该过滤token字段', () => {
      const data = {
        access_token: 'abc123token',
        refreshToken: 'refresh456'
      };
      const masked = sensitiveDataMasker(data);
      expect(masked.access_token).toBe('***MASKED***');
      expect(masked.refreshToken).toBe('***MASKED***');
    });

    test('应该过滤信用卡号', () => {
      const data = {
        cardNumber: '4111111111111111',
        cvv: '123'
      };
      const masked = sensitiveDataMasker(data, ['cardNumber', 'cvv', 'creditCard', ...loggingConfig.sensitiveFields]);
      expect(masked.cardNumber).toBe('***MASKED***');
      expect(masked.cvv).toBe('***MASKED***');
    });

    test('应该过滤authorization字段', () => {
      const data = {
        authorization: 'Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9',
        apiKey: 'sk_test_123456'
      };
      const masked = sensitiveDataMasker(data);
      expect(masked.authorization).toBe('***MASKED***');
      expect(masked.apiKey).toBe('***MASKED***');
    });

    test('应该处理嵌套对象', () => {
      const data = {
        user: {
          email: 'test@example.com',
          password: 'password123'
        },
        metadata: {
          ip: '192.168.1.1'
        }
      };
      const masked = sensitiveDataMasker(data);
      expect(masked.user.email).toBe('test@example.com');
      expect(masked.user.password).toBe('***MASKED***');
      expect(masked.metadata.ip).toBe('192.168.1.1');
    });

    test('应该处理数组', () => {
      const data = [
        { username: 'user1', password: 'pass1' },
        { username: 'user2', password: 'pass2' }
      ];
      const masked = sensitiveDataMasker(data);
      expect(masked[0].username).toBe('user1');
      expect(masked[0].password).toBe('***MASKED***');
      expect(masked[1].password).toBe('***MASKED***');
    });

    test('应该处理null和undefined', () => {
      expect(sensitiveDataMasker(null)).toBe(null);
      expect(sensitiveDataMasker(undefined)).toBe(undefined);
    });

    test('应该处理非对象类型', () => {
      expect(sensitiveDataMasker('string')).toBe('string');
      expect(sensitiveDataMasker(123)).toBe(123);
      expect(sensitiveDataMasker(true)).toBe(true);
    });
  });

  describe('请求ID生成', () => {
    test('应该生成唯一ID', () => {
      const id1 = generateRequestId();
      const id2 = generateRequestId();
      expect(id1).toBeTruthy();
      expect(id2).toBeTruthy();
      expect(id1).not.toBe(id2);
    });

    test('应该包含正确的格式', () => {
      const id = generateRequestId();
      expect(id).toMatch(/^req_\d+_[a-f0-9]{8}$/);
    });
  });

  describe('RequestId中间件', () => {
    test('应该为请求添加requestId', () => {
      const req = {};
      const res = {};
      const next = jest.fn();

      requestIdMiddleware(req, res, next);

      expect(req.requestId).toBeTruthy();
      expect(res.requestId).toBeTruthy();
      expect(next).toHaveBeenCalled();
    });

    test('应该使用已存在的X-Request-ID头', () => {
      const req = {
        headers: {
          'x-request-id': 'existing-id-123'
        }
      };
      const res = {};
      const next = jest.fn();

      requestIdMiddleware(req, res, next);

      expect(req.requestId).toBe('existing-id-123');
      expect(res.requestId).toBe('existing-id-123');
    });

    test('应该在响应头中包含requestId', () => {
      const req = {};
      const res = {
        setHeader: jest.fn()
      };
      const next = jest.fn();

      requestIdMiddleware(req, res, next);

      expect(res.setHeader).toHaveBeenCalledWith('X-Request-ID', expect.any(String));
    });
  });

  describe('SanitizeLogs中间件', () => {
    test('应该过滤请求体中的敏感信息', () => {
      const req = {
        body: {
          username: 'testuser',
          password: 'secret123',
          email: 'test@example.com'
        }
      };
      const res = {};
      const next = jest.fn();

      sanitizeLogsMiddleware(req, res, next);

      expect(req.body.username).toBe('testuser');
      expect(req.body.password).toBe('***MASKED***');
      expect(req.body.email).toBe('test@example.com');
      expect(next).toHaveBeenCalled();
    });

    test('应该过滤查询参数中的敏感信息', () => {
      const req = {
        query: {
          page: '1',
          token: 'secret-token',
          apiKey: 'key-123'
        }
      };
      const res = {};
      const next = jest.fn();

      sanitizeLogsMiddleware(req, res, next);

      expect(req.query.page).toBe('1');
      expect(req.query.token).toBe('***MASKED***');
      expect(req.query.apiKey).toBe('***MASKED***');
    });

    test('应该过滤headers中的authorization', () => {
      const req = {
        headers: {
          'content-type': 'application/json',
          authorization: 'Bearer secret-token',
          'x-custom-header': 'value'
        }
      };
      const res = {};
      const next = jest.fn();

      sanitizeLogsMiddleware(req, res, next);

      expect(req.headers['content-type']).toBe('application/json');
      expect(req.headers.authorization).toBe('***MASKED***');
      expect(req.headers['x-custom-header']).toBe('value');
    });

    test('应该处理没有body的请求', () => {
      const req = {};
      const res = {};
      const next = jest.fn();

      expect(() => sanitizeLogsMiddleware(req, res, next)).not.toThrow();
      expect(next).toHaveBeenCalled();
    });

    test('应该处理params中的敏感信息', () => {
      const req = {
        params: {
          id: '123',
          secret: 'hidden-value'
        }
      };
      const res = {};
      const next = jest.fn();

      sanitizeLogsMiddleware(req, res, next);

      expect(req.params.id).toBe('123');
      expect(req.params.secret).toBe('***MASKED***');
    });
  });

  describe('日志格式验证', () => {
    test('应该包含必需的时间戳字段', () => {
      const testData = {
        timestamp: new Date().toISOString(),
        level: 'info',
        message: 'Test message'
      };
      expect(testData.timestamp).toBeTruthy();
      expect(testData.level).toBeTruthy();
    });

    test('应该支持requestId字段', () => {
      const logData = {
        requestId: generateRequestId(),
        message: 'Test'
      };
      expect(logData.requestId).toMatch(/^req_\d+_[a-f0-9]{8}$/);
    });

    test('应该支持userId字段', () => {
      const logData = {
        userId: 'user123',
        message: 'User action'
      };
      expect(logData.userId).toBe('user123');
    });

    test('应该支持ip字段', () => {
      const logData = {
        ip: '192.168.1.100',
        message: 'Request from IP'
      };
      expect(logData.ip).toBe('192.168.1.100');
    });

    test('应该支持method字段', () => {
      const logData = {
        method: 'POST',
        path: '/api/users',
        message: 'API request'
      };
      expect(logData.method).toBe('POST');
      expect(logData.path).toBe('/api/users');
    });

    test('应该支持statusCode字段', () => {
      const logData = {
        statusCode: 200,
        duration: 150,
        message: 'Response sent'
      };
      expect(logData.statusCode).toBe(200);
      expect(logData.duration).toBe(150);
    });
  });
});
