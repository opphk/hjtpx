const AdvancedRateLimiter = require('../../src/backend/middleware/rateLimiterAdvanced');
const { advancedLimiter, rateLimiters } = require('../../src/backend/middleware/rateLimiterAdvanced');

describe('AdvancedRateLimiter', () => {
  let mockReq;
  let mockRes;
  let mockNext;
  
  beforeEach(() => {
    mockReq = {
      ip: '192.168.1.1',
      path: '/api/test',
      user: { id: 'user123' }
    };
    
    mockRes = {
      status: jest.fn().mockReturnThis(),
      json: jest.fn(),
      set: jest.fn()
    };
    
    mockNext = jest.fn();
  });
  
  describe('createLimiter', () => {
    test('创建基本限流器', () => {
      const limiter = advancedLimiter.createLimiter({
        key: 'test',
        maxRequests: 10,
        windowMs: 60000
      });
      
      expect(limiter.key).toBe('test');
      expect(limiter.maxRequests).toBe(10);
      expect(limiter.windowMs).toBe(60000);
    });
    
    test('创建带自定义key生成器的限流器', () => {
      const limiter = advancedLimiter.createLimiter({
        key: 'custom',
        keyGenerator: (req) => `custom:${req.user.id}`
      });
      
      const key = limiter.keyGenerator(mockReq);
      expect(key).toBe('custom:user123');
    });
  });
  
  describe('generateKey', () => {
    test('生成IP维度key', () => {
      const key = advancedLimiter.generateKey(mockReq, 'ip');
      expect(key).toBe('ratelimit:ip:192.168.1.1');
    });
    
    test('生成用户维度key', () => {
      const key = advancedLimiter.generateKey(mockReq, 'user');
      expect(key).toBe('ratelimit:user:user123');
    });
    
    test('生成端点维度key', () => {
      const key = advancedLimiter.generateKey(mockReq, 'endpoint');
      expect(key).toBe('ratelimit:endpoint:/api/test');
    });
  });
  
  describe('exemptions', () => {
    test('添加豁免', () => {
      advancedLimiter.addExemption('192.168.1.100', 'Test exemption');
      expect(advancedLimiter.exemptions.has('192.168.1.100')).toBe(true);
    });
    
    test('移除豁免', () => {
      advancedLimiter.addExemption('192.168.1.100');
      advancedLimiter.removeExemption('192.168.1.100');
      expect(advancedLimiter.exemptions.has('192.168.1.100')).toBe(false);
    });
    
    test('检查豁免用户', () => {
      mockReq.ip = '192.168.1.100';
      advancedLimiter.addExemption('192.168.1.100');
      expect(advancedLimiter.isExempt(mockReq)).toBe(true);
    });
  });
  
  describe('multi-dimensional limiting', () => {
    test('单维度限流', async () => {
      const limiter = advancedLimiter.createMultiDimensionalLimiter({
        dimensions: ['ip'],
        limits: {
          ip: { max: 5, window: 1000 }
        }
      });
      
      await limiter(mockReq, mockRes, mockNext);
      expect(mockNext).toHaveBeenCalled();
    });
    
    test('多维度限流', async () => {
      const limiter = advancedLimiter.createMultiDimensionalLimiter({
        dimensions: ['ip', 'user'],
        limits: {
          ip: { max: 10, window: 1000 },
          user: { max: 10, window: 1000 }
        }
      });
      
      await limiter(mockReq, mockRes, mockNext);
      expect(mockNext).toHaveBeenCalled();
    });
  });
  
  describe('rate limit headers', () => {
    test('设置限流头', async () => {
      const limiter = advancedLimiter.createMultiDimensionalLimiter({
        dimensions: ['ip'],
        limits: {
          ip: { max: 100, window: 60000 }
        }
      });
      
      await limiter(mockReq, mockRes, mockNext);
      
      expect(mockRes.set).toHaveBeenCalledWith(
        expect.objectContaining({
          'X-RateLimit-Limit': expect.any(Number),
          'X-RateLimit-Remaining': expect.any(Number),
          'X-RateLimit-Reset': expect.any(Number)
        })
      );
    });
  });
});

describe('Rate Limit Configuration', () => {
  const rateLimitConfig = require('../../src/backend/config/rateLimit');
  
  test('默认配置存在', () => {
    expect(rateLimitConfig.default).toBeDefined();
    expect(rateLimitConfig.default.windowMs).toBe(60000);
    expect(rateLimitConfig.default.max).toBe(100);
  });
  
  test('端点配置存在', () => {
    expect(rateLimitConfig.endpoints).toBeDefined();
    expect(rateLimitConfig.endpoints['/api/auth/login']).toBeDefined();
  });
  
  test('白名单配置', () => {
    expect(rateLimitConfig.whitelist).toContain('127.0.0.1');
  });
  
  test('动态规则配置', () => {
    expect(rateLimitConfig.dynamicRules).toBeDefined();
    expect(rateLimitConfig.dynamicRules.loadBased.enabled).toBe(true);
  });
});

describe('Rate Limit Integration', () => {
  test('预定义限流器存在', () => {
    expect(rateLimiters.api).toBeDefined();
    expect(rateLimiters.auth).toBeDefined();
    expect(rateLimiters.upload).toBeDefined();
    expect(rateLimiters.search).toBeDefined();
  });
});
